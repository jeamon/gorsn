package gorsn

type eventName string

const (
	CREATE   eventName = "CREATE"
	MODIFY   eventName = "MODIFY"
	DELETE   eventName = "DELETE"
	PERM     eventName = "PERM"
	ERROR    eventName = "ERROR"
	NOCHANGE eventName = "NOCHANGE"
)

type pathType string

const (
	FILE        pathType = "FILE"
	DIR         pathType = "DIRECTORY"
	SYMLINK     pathType = "SYMLINK"
	UNSUPPORTED pathType = "UNSUPPORTED"
)

type Event struct {
	Path  string
	Type  pathType
	Name  eventName
	Error error
}

// queueEvent emits an to the queue after constructing the event.
func (sn *snotifier) queueEvent(ev Event) bool {
	if !sn.running.Load() {
		return false
	}
	select {
	case sn.queue <- ev:
		return true
	case <-sn.stop:
	}
	return false
}
