package main

import (
	"flag"
	"time"

	teamClient "github.com/aftouh/k8s-sample-controller/pkg/client/clientset/versioned"
	teamInformer "github.com/aftouh/k8s-sample-controller/pkg/client/informers/externalversions"

	"github.com/aftouh/k8s-sample-controller/util/signals"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig. Not needed inside the cluster")
)

const resyncPeriod = time.Second * 30

func main() {

	klog.InitFlags(nil)
	flag.Parse()
	klog.V(5).Infof("kubeconfig set to: %q", *kubeconfig)

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Fatalf("failed loading config, %s", err)
	}

	tClientSet, err := teamClient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed building team client. %s", err)
	}

	kClientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed building kubernetes client. %s", err)
	}

	stopChan := signals.StopChan()

	tInfomerFactory := teamInformer.NewSharedInformerFactory(tClientSet, resyncPeriod)
	kInformerFactory := kubeinformers.NewSharedInformerFactory(kClientSet, resyncPeriod)

	controller := NewTeamController(tClientSet,
		kClientSet,
		tInfomerFactory.Aftouh().V1().Teams(),
		kInformerFactory.Core().V1().Namespaces(),
		kInformerFactory.Core().V1().ResourceQuotas())

	tInfomerFactory.Start(stopChan)
	kInformerFactory.Start(stopChan)

	if err := controller.Run(2, stopChan); err != nil {
		klog.Fatalf("failed starting team controller. %s", err)
	}
}
