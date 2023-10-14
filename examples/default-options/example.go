package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jeamon/gorsn"
)

func main() {
	// step 1. define a path to a valid folder.
	root, err := os.Getwd() // lets use this package folder
	if err != nil {
		log.Fatal(err)
	}

	// step 2. use default options.
	var opts gorsn.Options

	// step 3. get an instance.
	sn, err := gorsn.New(root, &opts)
	if err != nil {
		log.Fatal(err)
	}

	// [optional] stop the scan notifier on CTRL+C.
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGQUIT,
			syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

		<-sigChan
		signal.Stop(sigChan)
		sn.Stop()
	}()

	// step 4. asynchronously receive events from the queue.
	go func() {
		for event := range sn.Queue() {
			log.Printf("received %q %s %s %v\n", event.Path, event.Type, event.Name, event.Error)
		}
	}()

	done := make(chan struct{})
	// step 5. start the scan notifier on the defined path.
	// lets run it as goroutine to bypass the blocking behavior.
	go func() {
		err = sn.Start(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		done <- struct{}{}
	}()

	// lets change the settings on the fly.
	opts.SetMaxWorkers(10).SetScanInterval(500 * time.Millisecond)

	// wait the runner to exit
	<-done
}
