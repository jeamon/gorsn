package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jeamon/gorsn"
)

func main() {
	// define a path to a valid folder.
	// lets use this project folder so
	// we can observe each change.
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// set some options.
	// No exclude & include regex.
	// 10 events can wait into the queue.
	// 2 goroutines to build & emit events.
	// 0 sec - immediate, so no delay to scan.
	opts := gorsn.RegexOpts(nil, nil).
		SetQueueSize(10).
		SetMaxWorkers(2).
		SetScanInterval(0)

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
