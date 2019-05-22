package apps

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func InitSignals(onQuit func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		wg.Done()
		<-quit
		fmt.Println("quiting")
		onQuit()
	}()

	wg.Wait()
}
