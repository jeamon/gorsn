package gorsn

import (
	"io/fs"
	"path/filepath"
)

func getPathType(fm fs.FileMode) pathType {
	switch {
	case fm.IsDir() || fm&fs.ModeDir != 0:
		return DIR
	case fm.IsRegular():
		return FILE
	case fm&fs.ModeSymlink != 0:
		return SYMLINK
	default:
		return UNSUPPORTED
	}
}

func (sn *snotifier) finalize() {
	sn.stopping.Store(true)
	close(sn.stop)
	close(sn.iqueue)
	close(sn.queue)
	sn.wg.Wait()
	sn.flush()
	sn.running.Store(false)
	sn.stopping.Store(false)
	sn.ready = false
}

func (sn *snotifier) check(s string, t pathType, err error) (bool, error) {
	if t == UNSUPPORTED {
		return true, nil
	}

	if s == sn.root {
		// skip root folder.
		return true, nil
	}

	if sn.opts.excludePaths != nil && sn.opts.excludePaths.MatchString(s) {
		return true, nil
	}

	if sn.opts.includePaths != nil && !sn.opts.includePaths.MatchString(s) {
		return true, nil
	}

	if t == FILE && sn.opts.event.ignoreFile.Load() {
		return true, nil
	}

	if t == DIR && sn.opts.event.ignoreFolderContent.Load() {
		return true, filepath.SkipDir
	}

	if t == DIR && sn.opts.event.ignoreFolder.Load() {
		return true, nil
	}

	if t == SYMLINK && sn.opts.event.ignoreSymlink.Load() {
		return true, nil
	}

	return false, nil
}
