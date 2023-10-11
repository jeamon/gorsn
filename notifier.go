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
	defaultQueueSize    = 10
	defaultMaxWorkers   = 1
	defaultScanInterval = 1 * time.Second
)

// ScanNotifier is an interface which defines a set of available actions.
type ScanNotifier interface {
	// Queue returns the channel to listen on for receiving changes events.
	// The returned channel is read-only to avoid closing or writing on.
	Queue() <-chan Event

	// Start begins periodic scanning of root directory and emitting events.
	Start(context.Context) error

	// Stop aborts the scanning of root directory and sending events.
	Stop() error

	// IsRunning reports whether the scan notifier has started.
	IsRunning() bool

	// Flush clears internal cache history of files and directories under monitoring.
	// Once succeeded, `CREATE` is the next event for each item under monitoring.
	// This could be used directly after initialization of the scan notifier instance
	// in order to receive the list of item (via `CREATE` event) inside root directory.
	// Calling this while the scan notifier has started will make the scanner to detect
	// each item like newly created into the root directory so the notifier will emit
	// `CREATE` event for those items almost immediately.
	Flush()

	// Pause instructs the scanner to escape at each polling interval so no changes
	// detection will happen then no new events will be sent.
	// Use Resume() to restart the normal scanning and event notification processes.
	Pause() error
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
	root     string
	opts     *Options
	paths    sync.Map
	queue    chan Event
	iqueue   chan *fsEntry
	stop     chan struct{}
	ready    bool
	wg       *sync.WaitGroup
	running  atomic.Bool
	stopping atomic.Bool
	paused   atomic.Bool
}

// Queue returns a read only channel of events.
func (sn *snotifier) Queue() <-chan Event {
	return sn.queue
}

// Stop triggers the notifier routines to exit
// and to close the events queue.
func (sn *snotifier) Stop() error {
	if sn.isStopping() {
		return ErrScanIsStopping
	}
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

// isStopping tells wether the scan notifier is in the process of closing.
func (sn *snotifier) isStopping() bool {
	return sn.stopping.Load()
}

// Flush remove all root directory items recent history.
func (sn *snotifier) Flush() {
	sn.flush()
}

func (sn *snotifier) flush() {
	if sn == nil {
		return
	}
	sn.paths.Range(func(key interface{}, value interface{}) bool {
		sn.paths.Delete(key)
		return true
	})
}

// New provides an initialized object which satisfies the ScanNotifier interface.
// It initializes itself based on the `root` value which is expected to be a
// path to an accessible directory. The content of the root directory will be
// parsed and loaded based on the options provided by `opts`. It returns and error
// which wraps `ErrInvalidRootDirPath` in case the root path is not an accessible
// directory. `ErrInitialization` means the initialization encoutered an error.
func New(root string, opts *Options) (ScanNotifier, error) {
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRootDirPath, err)
	}

	opts = opts.setup()

	sn := &snotifier{
		root:  root,
		opts:  opts,
		paths: sync.Map{},
	}

	if err := filepath.WalkDir(sn.root, sn.init); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitialization, err)
	}

	sn.queue = make(chan Event, opts.queueSize)
	sn.iqueue = make(chan *fsEntry, opts.queueSize)
	sn.stop = make(chan struct{})
	sn.wg = &sync.WaitGroup{}
	sn.ready = true
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
func (sn *snotifier) Start(ctx context.Context) error {
	if sn.IsRunning() {
		return ErrScanAlreadyStarted
	}
	if sn.isStopping() {
		return ErrScanIsStopping
	}
	if !sn.ready {
		return ErrScanIsNotReady
	}
	sn.running.Store(true)
	// sn.workers()
	sn.scanner(ctx)
	return nil
}

// scanner runs an infinite scan loop after each interval of time.
// it exits on context cancellation or on call to stop the notifier.
func (sn *snotifier) scanner(ctx context.Context) {
	var done atomic.Bool
	for {
		select {
		case <-sn.stop:
			sn.finalize()
			return
		case <-ctx.Done():
			sn.finalize()
			return
		default:
			if sn.paused.Load() {
				time.Sleep(sn.opts.scanInterval.Load().(time.Duration))
				continue
			}
			done.Store(false)
			sn.workers(&done)
			filepath.WalkDir(sn.root, sn.scan)
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
			ev := Event{path, getPathType(pi.mode), DELETE, nil}
			sn.queueEvent(ev)
		}
		sn.paths.Delete(path)
		return true
	})
}

// Pause triggers the scanner routine to escape at each intervall
// so that no new changes will be detected and no events to be sent.
func (sn *snotifier) Pause() error {
	if sn.isStopping() {
		return ErrScanIsStopping
	}
	if !sn.IsRunning() {
		return ErrScanIsNotRunning
	}
	sn.paused.Store(true)
	return nil
}
