package main

import (
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/jeamon/gorsn"
)

func main() {
	// step 1. define a path to a valid folder.
	root, err := os.Getwd() // lets use this package folder
	if err != nil {
		log.Fatal(err)
	}

	// step 2. set some options.
	excludeRule := regexp.MustCompile(`.*(\.git).*`)
	// exclude `.git` folder. No include rule.
	opts := gorsn.RegexOpts(excludeRule, nil).
		// 5 events can wait into the queue.
		SetQueueSize(5).
		// 2 goroutines to build & emit events.
		SetMaxWorkers(2).
		// 0 sec - immediate, so no delay to scan.
		SetScanInterval(0)
		// others options keep their default values.

	// step 3. get an instance based on above settings.
	sn, err := gorsn.New(root, opts)
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

	// step 5. start the scan notifier on the defined path.
	err = sn.Start() // blocks unless it fails or until stopped.
	if err != nil {
		log.Fatal(err)
	}
}
