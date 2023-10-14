# gorsn

`gorsn` means **Go Resource Scan Notifier**. This is a simple & high-concurrent & options-rich cross-platform go-based library to periodically scan a folder and all its sub-content to get notified at any changes. Options are thread-safe and can be modified even during the program execution. The options allow for example to scale the number of workers/goroutines and to specify which kind of events we are interested in.

## Features

A successful scan-notifier provided by `gorsn.New` method is an interface with below actions.

| Action | Description |
|:------ | :-------------------------------------- |
| Queue() <-chan Event | provides a read-only channel to listen events from |
| Start(context.Context) error | starts the scanner and events notifications routines |
| Stop() error | stops the scanner and events notifications routines |
| Pause() error | triggers to scanner to pause to avoid emitting events |
| Resume() error | restarts the scanner and notifier after being paused |
| IsRunning() bool | informs wether the scanner notifier is stopped or not |
| Flush() | clears latest changes infos of files under monitoring |

## Installation

Just import the `gorsn` library as external package to start using it into your project. There are some examples into the examples folder to learn more. 

**[Step 1] -** Download the package

```shell
$ go get github.com/jeamon/gorsn
```


**[Step 2] -** Import the package into your project

```shell
$ import "github.com/jeamon/gorsn"
```


**[Step 3] -** Optional: Clone the library to run some examples

```shell
$ git clone https://github.com/jeamon/gorsn.git
$ cd gorsn
$ go run examples/default-options/example.go
$ go run examples/custom-options/example.go
```

## Usage

* **default options / settings**

```go
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

	// we can always change the settings at anytime in the program.
	opts.SetMaxWorkers(5).SetScanInterval(100 * time.Millisecond)

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
	opts.SetMaxWorkers(10).
		SetScanInterval(500 * time.Millisecond).
		SetIgnoreNoChangeEvent(false)

	// wait the runner to exit
	<-done
}
```

* **custom options / settings**

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/jeamon/gorsn"
)

func main() {
	// step 1. define a path to a valid folder.
	// lets use this current folder so you can
	// observe any changes into your codebase.
	root := "./"

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
	err = sn.Start(context.Background()) // blocks unless it fails or until stopped.
	if err != nil {
		log.Fatal(err)
	}
}
```

## Contact

Feel free to [reach out to me](https://blog.cloudmentor-scale.com/contact) before any action. Feel free to connect on [Twitter](https://twitter.com/jerome_amon) or [linkedin](https://www.linkedin.com/in/jeromeamon/)
