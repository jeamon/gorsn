// Package gorsn implements utility routines for periodically monitoring a folder and
// its sub-folders content to detect any change such as file or folder deletion and
// creation along with content or permissions modification. Then it emits an appropriate
// event object on a consumable channel. The notifier system it provides could be stopped
// and status checked once started. It finally accepts a set of options which are safe to
// be modified by multiple goroutines even during its operations.
package gorsn

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultQueueSize  = 10
	defaultMaxWorkers = 1
)

// ScanNotifier is an interface which defines
// a set of available actions by a resource
// scan notifier.
type ScanNotifier interface {
	Queue() <-chan *Event
	Start() error
	Stop() error
	IsRunning() bool
}

type pathInfos struct {
	modTime time.Time
	mode    fs.FileMode
	visited bool
}

type fsEntry struct {
	path string
	d    fs.DirEntry
	err  error
}

type snotifier struct {
	root    string
	opts    *Options
	paths   sync.Map
	queue   chan *Event
	iqueue  chan *fsEntry
	stop    chan struct{}
	wg      *sync.WaitGroup
	ctx     context.Context
	running atomic.Bool
}

// Queue returns a read only channel of events.
func (sn *snotifier) Queue() <-chan *Event {
	return sn.queue
}

// Stop triggers the notifier routines to exit
// and to close the events queue.
func (sn *snotifier) Stop() error {
	if !sn.IsRunning() {
		return ErrScanIsNotRunning
	}
	sn.stop <- struct{}{}
	return nil
}

// IsRunning tells wether the scan notifier is still monitoring
// for new events or was stopped or was not started yet.
func (sn *snotifier) IsRunning() bool {
	return sn.running.Load()
}

// New provides an initialized object which satisfies the ScanNotifier interface.
// It initializes itself based on the `root` value which is expected to be a
// path to an accessible directory. The content of the root directory will be
// parsed and loaded based on the options provided by `opts`. It returns and error
// which wraps `ErrInvalidRootDirPath` in case the root path is not an accessible
// directory. `ErrInitialization` means the initialization encoutered an error.
func New(ctx context.Context, root string, opts *Options) (ScanNotifier, error) {
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRootDirPath, err)
	}

	if opts.queueSize <= 0 {
		opts.queueSize = defaultQueueSize
	}

	if opts.maxworkers.Load() <= 0 {
		opts.maxworkers.Store(defaultMaxWorkers)
	}

	opts.scanInterval.Store(1 * time.Second)

	sn := &snotifier{
		root:  root,
		opts:  opts,
		paths: sync.Map{},
	}

	if err := filepath.WalkDir(sn.root, sn.init); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitialization, err)
	}

	opts.event.ignoreNoChange.Store(true)
	sn.queue = make(chan *Event, opts.queueSize)
	sn.iqueue = make(chan *fsEntry, opts.queueSize)
	sn.stop = make(chan struct{})
	sn.wg = &sync.WaitGroup{}
	sn.ctx = ctx

	return sn, nil
}

func (sn *snotifier) init(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	t := getPathType(d.Type())
	if ignore, err := sn.check(s, t, err); ignore {
		return err
	}

	if fi, err := d.Info(); err == nil {
		sn.paths.Store(s, &pathInfos{fi.ModTime(), d.Type(), false})
	}

	return err
}

// Start is a blocking method which pre-boots the consumers
// and starts the infinite loop scanner to monitor the root
// directory contents.
func (sn *snotifier) Start() error {
	if sn.IsRunning() {
		return ErrScanAlreadyStarted
	}
	sn.running.Store(true)
	// sn.workers()
	sn.scanner()
	return nil
}

// scanner runs an infinite scan loop after each interval of time.
// it exits on context cancellation or on call to stop the notifiesn.
func (sn *snotifier) scanner() {
	var done atomic.Bool
	for {
		select {
		case <-sn.stop:
			sn.finalize()
			return
		case <-sn.ctx.Done():
			sn.finalize()
			return
		default:
			done.Store(false)
			sn.workers(&done)
			if err := filepath.WalkDir(sn.root, sn.scan); err != nil {
				fmt.Println(err)
			}
			done.Store(true)
			sn.wg.Wait()

			if !sn.opts.event.ignoreDelete.Load() {
				sn.missingPaths()
			}
			time.Sleep(sn.opts.scanInterval.Load().(time.Duration))
		}
	}
}

func (sn *snotifier) scan(s string, d fs.DirEntry, err error) error {
	t := getPathType(d.Type())
	if ignore, cerr := sn.check(s, t, err); ignore {
		return cerr
	}

	fse := &fsEntry{path: s, d: d}
	if err != nil {
		fse.err = err
	}

	sn.iqueue <- fse
	return nil
}

// missingPaths scans all latest registered paths to find
// deleted paths and trigger a `DELETE` event for each if
// this option was enabled. It aborts once the notifier is
// stopped.
func (sn *snotifier) missingPaths() {
	sn.paths.Range(func(key, value any) bool {
		if !sn.running.Load() {
			return false
		}
		path := key.(string)
		pi := value.(*pathInfos)
		if pi.visited {
			pi.visited = false
			return true
		}

		if !sn.opts.event.ignoreDelete.Load() {
			ev := &Event{path, getPathType(pi.mode), DELETE, nil}
			sn.queueEvent(ev)
		}
		sn.paths.Delete(path)
		return true
	})
}
