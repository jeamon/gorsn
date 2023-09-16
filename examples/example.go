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
	// define a path to a valid folder.
	// lets use this project folder so
	// we can observe each change.
	root := "../"
	// set some options.
	opts := gorsn.RegexOpts(nil, nil).
		SetQueueSize(10).
		SetMaxWorkers(2).
		SetScanInterval(1 * time.Second)
	// get an instance based on above settings.
	sn, err := gorsn.New(context.Background(), root, opts)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		// stop on CTRL+C
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGQUIT,
			syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

		<-sigChan
		signal.Stop(sigChan)
		sn.Stop()
	}()

	// asynchronously receive events from the queue.
	go func() {
		for event := range sn.Queue() {
			log.Printf("received %q %s %s %v\n", event.Path, event.Type, event.Name, event.Error)
		}
	}()

	// start the scan notifier on the defined path.
	// then block until it fails or get stopped.
	err = sn.Start()
	if err != nil {
		log.Fatal(err)
	}
}
