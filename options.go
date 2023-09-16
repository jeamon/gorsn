package gorsn

import (
	"regexp"
	"sync/atomic"
	"time"
)

// eventOps represents optional fields to define which event to produce.
type eventOps struct {
	ignoreErrors        atomic.Bool
	ignoreNoChange      atomic.Bool // should not emit event when nothing changed. default to true.
	ignoreDelete        atomic.Bool
	ignoreCreate        atomic.Bool
	ignoreModify        atomic.Bool
	ignorePerm          atomic.Bool
	ignoreFile          atomic.Bool // should emit event for regular files.
	ignoreFolder        atomic.Bool // should emit event for directories.
	ignoreSymlink       atomic.Bool
	ignoreFolderContent atomic.Bool // should emit event for each sub-content of a directory included the directory itself.
}

type Options struct {
	queueSize    int
	maxworkers   atomic.Uint32
	event        eventOps
	scanInterval atomic.Value
	excludePaths *regexp.Regexp
	includePaths *regexp.Regexp
}

func RegexOpts(eregex, iregex *regexp.Regexp) *Options {
	if eregex != nil && eregex.String() == "" {
		eregex = nil
	}
	if iregex != nil && iregex.String() == "" {
		iregex = nil
	}

	return &Options{excludePaths: eregex, includePaths: iregex}
}

func (o *Options) SetQueueSize(v int) *Options {
	o.queueSize = v
	return o
}

func (o *Options) SetMaxWorkers(v int) *Options {
	if v <= 0 {
		return o
	}
	o.maxworkers.Store(uint32(v))
	return o
}

func (o *Options) SetScanInterval(v time.Duration) *Options {
	o.scanInterval.Store(v)
	return o
}

func (o *Options) SetIgnoreErrors(v bool) *Options {
	o.event.ignoreErrors.Store(v)
	return o
}

func (o *Options) SetIgnoreNoChangeEvent(v bool) *Options {
	o.event.ignoreNoChange.Store(v)
	return o
}

func (o *Options) SetIgnoreDeleteEvent(v bool) *Options {
	o.event.ignoreDelete.Store(v)
	return o
}

func (o *Options) SetIgnoreCreateEvent(v bool) *Options {
	o.event.ignoreCreate.Store(v)
	return o
}

func (o *Options) SetIgnoreModifyEvent(v bool) *Options {
	o.event.ignoreModify.Store(v)
	return o
}

func (o *Options) SetIgnorePermEvent(v bool) *Options {
	o.event.ignorePerm.Store(v)
	return o
}

func (o *Options) SetIgnoreFileEvent(v bool) *Options {
	o.event.ignoreFile.Store(v)
	return o
}

func (o *Options) SetIgnoreFolderEvent(v bool) *Options {
	o.event.ignoreFolder.Store(v)
	return o
}

func (o *Options) SetIgnoreSymlink(v bool) *Options {
	o.event.ignoreSymlink.Store(v)
	return o
}

func (o *Options) SetIgnoreFolderContentEvent(v bool) *Options {
	o.event.ignoreFolderContent.Store(v)
	return o
}
