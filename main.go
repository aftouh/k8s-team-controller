package main

import (
	"flag"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig. Not need inside the cluster")
)

func init() {
	klog.InitFlags(nil)
	flag.Parse()
}

func main() {
	klog.Info("starting controller")
	klog.V(5).Infof("kubeconfig set to: %q", *kubeconfig)

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Fatalf("failed loading config, %s", err)
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed building kubernetes clientset. %s", err)
	}

	list, err := clientSet.AppsV1().Deployments("tekton-pipelines").List(metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Failed fetching pods")
	}
	for _, item := range list.Items {
		klog.Infoln(item.Name)
	}
}
