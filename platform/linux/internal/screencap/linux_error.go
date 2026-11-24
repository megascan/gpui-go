//go:build linux

package screencap

// LinCapError is the Linux engines' error type.
type LinCapError struct{ msg string }

// NewLinCapError returns a LinCapError with msg.
func NewLinCapError(msg string) *LinCapError { return &LinCapError{msg: msg} }

// Error returns the message.
func (e *LinCapError) Error() string { return e.msg }

func linCapErrorFrom(err error) *LinCapError { return NewLinCapError(err.Error()) }
