package gorsn

import (
	"io/fs"
	"sync/atomic"
)

func (sn *snotifier) workers(done *atomic.Bool) {
	var i uint32
	max := sn.opts.maxworkers.Load()
	for i < max {
		sn.wg.Add(1)
		go sn.work(done)
		i++
	}
}

func (sn *snotifier) work(done *atomic.Bool) {
	var fi fs.FileInfo
	var pt pathType
	var err error
	for {
		select {
		case fse := <-sn.iqueue:
			fi, err = fse.d.Info()
			if err != nil {
				// emit ERROR event earlier since no futuer check could be done.
				if !sn.opts.event.ignoreErrors.Load() {
					sn.queueEvent(&Event{fse.path, getPathType(fi.Mode().Type()), ERROR, err})
				}
				continue
			}

			pt = getPathType(fse.d.Type())
			// use check to support the dynamic nature of `sn.opts` value.
			// pass nil since `fse.err` is used to build the event later.
			if ignore, _ := sn.check(fse.path, pt, nil); ignore {
				continue
			}

			sn.event(pt, fse, fi)
		default:
			// default ensures usage & non-blocking of select.
			if len(sn.iqueue) == 0 && done.Load() {
				sn.wg.Done()
				return
			}
		}
	}
}

// event processes the path based on its recent state and emit or
// not an appropriate event to the external queue.
func (sn *snotifier) event(pt pathType, fse *fsEntry, fi fs.FileInfo) {
	val, exists := sn.paths.Load(fse.path)

	if !exists {
		sn.paths.Store(fse.path, &pathInfos{fi.ModTime(), fi.Mode().Type(), true})
		if !sn.opts.event.ignoreCreate.Load() {
			sn.queueEvent(&Event{fse.path, pt, CREATE, fse.err})
		}
		return
	}
	pi := val.(*pathInfos)
	pi.visited = true
	change := false
	if fi.Mode().Type().Perm() != pi.mode.Perm() {
		change = true
		pi.mode = fi.Mode().Type()
		if !sn.opts.event.ignorePerm.Load() {
			sn.queueEvent(&Event{fse.path, pt, PERM, fse.err})
		}
	}

	if fi.ModTime() != pi.modTime {
		change = true
		pi.modTime = fi.ModTime()
		if !sn.opts.event.ignoreModify.Load() {
			sn.queueEvent(&Event{fse.path, pt, MODIFY, fse.err})
		}
	}

	if !change && !sn.opts.event.ignoreNoChange.Load() {
		sn.queueEvent(&Event{fse.path, pt, NOCHANGE, fse.err})
	}
}
