package main

import (
	"fmt"
	"time"

	"k8s.io/klog"

	teamClientSet "github.com/aftouh/k8s-sample-controller/pkg/client/clientset/versioned"
	teamInformer "github.com/aftouh/k8s-sample-controller/pkg/client/informers/externalversions/team/v1"
	teamLister "github.com/aftouh/k8s-sample-controller/pkg/client/listers/team/v1"

	aftouh "github.com/aftouh/k8s-sample-controller/pkg/apis/team/v1"

	core "k8s.io/client-go/informers/core/v1"
	coreListers "k8s.io/client-go/listers/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	maxRetries = 10
)

//TeamController defines a kubernetes controller for team resource
type TeamController struct {
	//kubernetes
	kClientSet kubernetes.Interface

	//team
	tClientSet    teamClientSet.Interface
	tLister       teamLister.TeamLister
	tListerSynced cache.InformerSynced

	//namespace
	nLister       coreListers.NamespaceLister
	nListerSynced cache.InformerSynced

	//workqueue
	queue workqueue.RateLimitingInterface
}

//NewController creates team controller
func NewController(tClientSet teamClientSet.Interface,
	kClientSet kubernetes.Interface,
	tInformer teamInformer.TeamInformer,
	nInfomer core.NamespaceInformer) *TeamController {

	tc := &TeamController{
		kClientSet: kClientSet,

		tClientSet:    tClientSet,
		tLister:       tInformer.Lister(),
		tListerSynced: tInformer.Informer().HasSynced,

		nLister:       nInfomer.Lister(),
		nListerSynced: nInfomer.Informer().HasSynced,

		queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	tInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    tc.addTeam,
		UpdateFunc: tc.updateTeam,
		DeleteFunc: tc.deleteTeam,
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

func (tc *TeamController) deleteTeam(obj interface{}) {
	t, ok := obj.(*aftouh.Team)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		t, ok = tombstone.Obj.(*aftouh.Team)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Team %#v", obj))
			return
		}
	}
	klog.V(4).Infof("Deleting team %s", t.Name)
	tc.enqueue(t)
}

func (tc *TeamController) enqueue(t *aftouh.Team) {
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(t); err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", t, err))
		return
	}

	tc.queue.Add(key)
}

//Run starts controller
func (tc *TeamController) Run(workers int, stopCh <-chan struct{}) error {
	klog.Info("starting team controller")
	defer tc.queue.ShutDown()

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, tc.tListerSynced); !ok {
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

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("failed splitting key. %s", err)
	}

	team, err := tc.tLister.Teams(namespace).Get(name)
	if errors.IsNotFound(err) {
		klog.V(2).Infof("Team %v has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	return tc.sync(team.DeepCopy())
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
