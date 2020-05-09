package main

import (
	"fmt"
	"time"

	"k8s.io/klog"

	tclient "github.com/aftouh/k8s-sample-controller/pkg/client/clientset/versioned"
	tinformer "github.com/aftouh/k8s-sample-controller/pkg/client/informers/externalversions/team/v1"
	tlister "github.com/aftouh/k8s-sample-controller/pkg/client/listers/team/v1"

	aftouh "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"

	//Core informers and listers
	cinformer "k8s.io/client-go/informers/core/v1"
	clister "k8s.io/client-go/listers/core/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	coreTyped "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	maxRetries            = 10
	messageResourceExists = "Resource %q already exists and is not managed by Team"
	errResourceExists     = "ErrResourceExists"
)

//TeamController defines a kubernetes controller for team resource
type TeamController struct {
	//kubernetes
	kClientSet kubernetes.Interface

	//team
	tClientSet    tclient.Interface
	tLister       tlister.TeamLister
	tListerSynced cache.InformerSynced

	//namespace
	nLister       clister.NamespaceLister
	nListerSynced cache.InformerSynced

	//resourceQuota
	rqLister       clister.ResourceQuotaLister
	rqListerSynced cache.InformerSynced

	//workqueue
	queue workqueue.RateLimitingInterface

	//kubernetes event recorder
	recorder record.EventRecorder
}

//NewTeamController creates team controller
func NewTeamController(tClientSet tclient.Interface,
	kClientSet kubernetes.Interface,
	tInformer tinformer.TeamInformer,
	nInformer cinformer.NamespaceInformer,
	rqInformer cinformer.ResourceQuotaInformer) *TeamController {

	eventBrodcaster := record.NewBroadcaster()
	eventBrodcaster.StartLogging(klog.Infof)
	eventBrodcaster.StartRecordingToSink(&coreTyped.EventSinkImpl{Interface: kClientSet.CoreV1().Events("")})

	tc := &TeamController{
		kClientSet: kClientSet,

		tClientSet:    tClientSet,
		tLister:       tInformer.Lister(),
		tListerSynced: tInformer.Informer().HasSynced,

		nLister:       nInformer.Lister(),
		nListerSynced: nInformer.Informer().HasSynced,

		rqLister:       rqInformer.Lister(),
		rqListerSynced: rqInformer.Informer().HasSynced,

		queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		recorder: eventBrodcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "team-controller"}),
	}

	tInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    tc.addTeam,
		UpdateFunc: tc.updateTeam,
	})

	nInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: tc.updateNamespace,
		DeleteFunc: tc.deleteNamespace,
	})

	return tc
}

func (tc *TeamController) addTeam(obj interface{}) {
	t := obj.(*aftouh.Team)
	klog.V(4).Infof("Adding team %q", t.Name)
	tc.enqueue(t)
}

func (tc *TeamController) updateTeam(old, cur interface{}) {
	oldT := old.(*aftouh.Team)
	curT := cur.(*aftouh.Team)
	klog.V(4).Infof("Updating team %s", oldT.Name)
	tc.enqueue(curT)
}

func (tc *TeamController) updateNamespace(old, cur interface{}) {
	oldN := old.(*corev1.Namespace)
	curN := cur.(*corev1.Namespace)
	klog.V(4).Infof("Updating namespace %s", oldN.Name)

	//No modification made
	if oldN.ResourceVersion == curN.ResourceVersion {
		return
	}

	ownerRef := metav1.GetControllerOf(curN)
	// If this object is not owned by a Team, we should not do anything more with it
	if ownerRef == nil || ownerRef.Kind != "Team" {
		return
	}

	team, err := tc.tLister.Get(ownerRef.Name)
	if err != nil {
		klog.V(4).Infof("ignoring orphaned namespace %q' of team %q", curN.GetSelfLink(), ownerRef.Name)
		return
	}

	tc.enqueue(team)
}

func (tc *TeamController) deleteNamespace(obj interface{}) {
	namespace, ok := obj.(*corev1.Namespace)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		namespace, ok = tombstone.Obj.(*corev1.Namespace)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Namespace %#v", obj))
			return
		}
	}
	klog.V(4).Infof("Deleting Namespace %s", namespace.Name)

	ownerRef := metav1.GetControllerOf(namespace)
	// If this object is not owned by a Team, we should not do anything more with it
	if ownerRef == nil || ownerRef.Kind != "Team" {
		return
	}

	team, err := tc.tLister.Get(ownerRef.Name)
	if err != nil {
		klog.V(4).Infof("ignoring orphaned namespace %q' of team %q", namespace.GetSelfLink(), ownerRef.Name)
		return
	}

	tc.enqueue(team)
}

func (tc *TeamController) enqueue(t *aftouh.Team) {
	key, err := cache.MetaNamespaceKeyFunc(t)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", t, err))
		return
	}
	tc.queue.Add(key)
}

//Run starts controller
func (tc *TeamController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	klog.Info("starting team controller")
	defer tc.queue.ShutDown()

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, tc.tListerSynced, tc.nListerSynced, tc.rqListerSynced); !ok {
		return fmt.Errorf("failed to sync informer caches")
	}
	klog.Info("informers cache synced sucessfully")

	for i := 0; i < workers; i++ {
		go wait.Until(tc.runWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (tc *TeamController) runWorker() {
	for tc.processNextWorkItem() {
	}
}

func (tc *TeamController) processNextWorkItem() bool {
	key, shutdown := tc.queue.Get()
	if shutdown {
		return false
	}

	defer tc.queue.Done(key)

	err := tc.syncHandler(key.(string))
	tc.handleErr(err, key)

	return true
}

func (tc *TeamController) syncHandler(key string) error {
	startTime := time.Now()
	klog.V(4).Infof("Started syncing team %q", key)
	defer func() {
		klog.V(4).Infof("Finished syncing team %q (%v)", key, time.Since(startTime))
	}()

	team, err := tc.tLister.Get(key)
	switch {
	case errors.IsNotFound(err):
		klog.V(4).Infof("Team %v has been deleted", key)
		err = nil
	case err != nil:
		err = fmt.Errorf("Unable to retrieve team %v from store: %v", key, err)
	default:
		t := team.DeepCopy()
		err = tc.syncNamespace(t)
		if err == nil {
			err = tc.syncResourceQuota(t)
		}
	}

	return err
}

func (tc *TeamController) syncNamespace(t *aftouh.Team) error {
	namespaceName := getTeamNamespace(t)
	namespace, err := tc.nLister.Get(namespaceName)

	//Namespace does not exist. Need to be created
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Creating namespace %s", namespaceName)
		_, err = tc.kClientSet.CoreV1().Namespaces().Create(newNamespace(t))
		return err
	}

	if err != nil {
		return err
	}

	// Namespace should be created by this controller
	if !metav1.IsControlledBy(namespace, t) {
		msg := fmt.Sprintf(messageResourceExists, namespace.Name)
		tc.recorder.Event(t, corev1.EventTypeWarning, errResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// Check namespace labels
	expectedLabels := getTeamLabels(t)
	var updateLabels bool
	for k, v := range expectedLabels {
		v2, ok := namespace.ObjectMeta.Labels[k]
		if !ok || (v != v2) {
			updateLabels = true
			break
		}
	}
	if updateLabels {
		for k, v := range expectedLabels {
			namespace.ObjectMeta.Labels[k] = v
		}
		klog.V(2).Infof("Updating namespace %q labels", namespaceName)
		_, err = tc.kClientSet.CoreV1().Namespaces().Update(namespace)
	}

	return err
}

func (tc *TeamController) syncResourceQuota(t *aftouh.Team) error {
	namespaceName := getTeamNamespace(t)
	rq, err := tc.rqLister.ResourceQuotas(namespaceName).Get(rqName)

	//ResourceQuota does not exist. Need to be created
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Creating resourceQuota %q", rqName)
		_, err = tc.kClientSet.CoreV1().ResourceQuotas(namespaceName).Create(newResourceQuota(t))
		return err
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(rq, t) {
		msg := fmt.Sprintf(messageResourceExists, rq.Name)
		tc.recorder.Event(t, corev1.EventTypeWarning, errResourceExists, msg)
		return fmt.Errorf(msg)
	}
	return nil
}

func (tc *TeamController) handleErr(err error, key interface{}) {
	if err == nil {
		tc.queue.Forget(key)
		return
	}

	if tc.queue.NumRequeues(key) < maxRetries { //Retry
		klog.V(2).Infof("Error syncing team %v: %v", key, err)
		tc.queue.AddRateLimited(key)
		return
	}

	utilruntime.HandleError(err)
	klog.V(2).Infof("Dropping team %q out of the queue: %v", key, err)
	tc.queue.Forget(key)
}
