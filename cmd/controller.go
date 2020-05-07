package main

import (
	"fmt"
	"time"

	"k8s.io/klog"

	teamClientSet "github.com/aftouh/k8s-sample-controller/pkg/client/clientset/versioned"
	teamInformer "github.com/aftouh/k8s-sample-controller/pkg/client/informers/externalversions/team/v1"
	teamLister "github.com/aftouh/k8s-sample-controller/pkg/client/listers/team/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	maxRetries = 10
)

//Controller defines team controller structure
type Controller struct {
	teamClientSet teamClientSet.Interface
	teamLister    teamLister.TeamLister
	teamSynced    cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

//NewController creates team controller
func NewController(tClientSet teamClientSet.Interface, tInformer teamInformer.TeamInformer) *Controller {

	controller := &Controller{
		teamClientSet: tClientSet,
		teamLister:    tInformer.Lister(),
		teamSynced:    tInformer.Informer().HasSynced,
		workqueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	tInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			klog.V(5).Info("object added")
			controller.enqueue(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			klog.V(5).Info("object updated")
			controller.enqueue(new)
		},
		DeleteFunc: func(obj interface{}) {
			klog.V(5).Info("object deleted")
			controller.enqueue(obj)
		},
	})

	return controller
}

func (c *Controller) enqueue(obj interface{}) {
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

//Run starts controller
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	klog.Info("starting team controller")
	defer c.workqueue.ShutDown()

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.teamSynced); !ok {
		return fmt.Errorf("failed to sync informer caches")
	}
	klog.Info("informer caches synced sucessfully")

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	key, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	defer c.workqueue.Done(key)

	err := c.syncHandler(key.(string))
	c.handleErr(err, key)

	return true
}

func (c *Controller) syncHandler(key string) error {
	startTime := time.Now()
	klog.V(4).Infof("Started syncing team %q (%v)", key, startTime)
	defer func() {
		klog.V(4).Infof("Finished syncing team %q (%v)", key, time.Since(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("failed splitting key. %s", err)
	}

	klog.Infof("Do something with team %q whithin namespace %q", name, namespace)

	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.workqueue.Forget(key)
		return
	}

	if c.workqueue.NumRequeues(key) < maxRetries { //Retry
		klog.V(2).Infof("Error syncing deployment %v: %v", key, err)
		c.workqueue.AddRateLimited(key)
		return
	}

	utilruntime.HandleError(err)
	klog.V(2).Infof("Dropping team %q out of the queue: %v", key, err)
	c.workqueue.Forget(key)
}
