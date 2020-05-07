package signals

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"k8s.io/klog"
)

var stopCh = make(chan struct{})
var once sync.Once

//StopChan returns channel that is closed when receiving one the these signals: syscall.SIGINT, syscall.SIGTERM
func StopChan() <-chan struct{} {
	once.Do(func() {
		signalCh := make(chan os.Signal, 2)
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-signalCh
			klog.Info("stopping process gracefully")
			close(stopCh)
			<-signalCh
			klog.Warningf("force exit")
			os.Exit(1)
		}()
	})

	return stopCh
}
