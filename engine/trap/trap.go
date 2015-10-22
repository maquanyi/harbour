package trap

import (
	"sync"
	"sync/atomic"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/Sirupsen/logrus"
)

var (
	lock                sync.RWMutex
	shutdown            bool
	shutdownWait        sync.WaitGroup
	shutdownCallback    []func()
	CleanupDone      = make(chan int, 1)
)

func ShutdownCallback(h func()) {
	lock.Lock()
	shutdownCallback = append(shutdownCallback, h)
	shutdownWait.Add(1)
	lock.Unlock()
}

func Shutdown() {
	lock.Lock()
	if shutdown {
		lock.Unlock()
		shutdownWait.Wait()
		return
	}
	shutdown = true
	lock.Unlock()

	// Call shutdown handlers, if any.
	// Timeout after 10 seconds.
	for _, h := range shutdownCallback {
		go func(h func()) {
			h()
			shutdownWait.Done()
		}(h)
	}
	done := make(chan struct{})
	go func() {
		shutdownWait.Wait()
		close(done)
	}()
	select {
		case <-time.After(time.Second * 10):
		case <-done:
	}
	return
}

func SignalsHandler(cleanup func()) {
	logrus.Debugln("trap init...")
	c := make(chan os.Signal, 1)
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if os.Getenv("DEBUG") == "" {
		signals = append(signals, syscall.SIGQUIT)
	}
	signal.Notify(c, signals...)
	go func() {
		interruptCount := uint32(0)
		for sig := range c {
			go func(sig os.Signal) {
				logrus.Infof("Received signal '%v', starting shutdown of harbour...", sig)
				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					// If the user really wants to interrupt, let him do so.
					if atomic.LoadUint32(&interruptCount) < 3 {
						// Initiate the cleanup only once
						if atomic.AddUint32(&interruptCount, 1) == 1 {
							// Call cleanup handler
							cleanup()
							CleanupDone <- 1
						} else {
							return
						}
					} else {
						logrus.Infof("Force shutdown of docker, interrupting cleanup")
					}
				case syscall.SIGQUIT:
				}
				os.Exit(128 + int(sig.(syscall.Signal)))
			}(sig)
		}
	}()
}
