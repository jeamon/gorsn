package gorsn

import "fmt"

// ErrorCode describes a failure on the resource scan notifiesn.
type ErrorCode string

const (
	// Unexpected error
	ErrInternalError ErrorCode = "internal error"

	// Operations errors
	ErrInvalidRootDirPath ErrorCode = "invalid root directory path"
	ErrInitialization     ErrorCode = "error parsing root directory"
	ErrScanIsNotRunning   ErrorCode = "scan notifier is not running"
	ErrScanAlreadyStarted ErrorCode = "scan notifier has already started"
)

// Error returns the real error message.
func (e ErrorCode) Error() string {
	return fmt.Sprintf("gorsn: %s", string(e))
}
