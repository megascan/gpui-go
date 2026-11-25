//go:build windows

package com

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// HRESULT is a COM status code; negative values are failures. A failed HRESULT is an error.
type HRESULT int32

// Common HRESULT values.
const (
	S_OK                  HRESULT = 0
	S_FALSE               HRESULT = 1
	E_NOTIMPL             HRESULT = -0x7FFFBFFF
	E_NOINTERFACE         HRESULT = -0x7FFFBFFE
	E_POINTER             HRESULT = -0x7FFFBFFD
	E_ABORT               HRESULT = -0x7FFFBFFC
	E_FAIL                HRESULT = -0x7FFFBFFB
	E_UNEXPECTED          HRESULT = -0x7FFF0001
	E_ACCESSDENIED        HRESULT = -0x7FF8FFFB
	E_HANDLE              HRESULT = -0x7FF8FFFA
	E_OUTOFMEMORY         HRESULT = -0x7FF8FFF2
	E_INVALIDARG          HRESULT = -0x7FF8FFA9
	CO_E_NOTINITIALIZED   HRESULT = -0x7FFBFE10
	RPC_E_CHANGED_MODE    HRESULT = -0x7FFEFEFA
	REGDB_E_CLASSNOTREG   HRESULT = -0x7FFBFEAC
	CLASS_E_NOAGGREGATION HRESULT = -0x7FFBFEF0
)

// HRESULTFromUintptr converts a syscall return value to an HRESULT.
func HRESULTFromUintptr(r uintptr) HRESULT {
	return HRESULT(int32(uint32(r)))
}

// HRESULTFromWin32 is HRESULT_FROM_WIN32.
func HRESULTFromWin32(code uint32) HRESULT {
	if int32(code) <= 0 {
		return HRESULT(int32(code))
	}
	return HRESULT(int32(code&0xFFFF | 0x80070000))
}

// HRESULTFromError converts err to an HRESULT: an HRESULT is itself, an Errno maps through
// HRESULT_FROM_WIN32, nil is S_OK and anything else is E_FAIL.
func HRESULTFromError(err error) HRESULT {
	switch e := err.(type) {
	case nil:
		return S_OK
	case HRESULT:
		return e
	case syscall.Errno:
		return HRESULTFromWin32(uint32(e))
	default:
		return E_FAIL
	}
}

// Succeeded reports whether hr is a success code.
func (hr HRESULT) Succeeded() bool {
	return hr >= 0
}

// Failed reports whether hr is a failure code.
func (hr HRESULT) Failed() bool {
	return hr < 0
}

// Err returns hr as an error, or nil if it is a success code.
func (hr HRESULT) Err() error {
	if hr >= 0 {
		return nil
	}
	return hr
}

// Uintptr returns hr as a callback return value.
func (hr HRESULT) Uintptr() uintptr {
	return uintptr(uint32(hr))
}

// Message returns the system's description of hr, or "" if it has none.
func (hr HRESULT) Message() string {
	var buf [512]uint16
	flags := uint32(windows.FORMAT_MESSAGE_FROM_SYSTEM | windows.FORMAT_MESSAGE_IGNORE_INSERTS)
	n, err := windows.FormatMessage(flags, 0, uint32(hr), 0, buf[:], nil)
	if err != nil || n == 0 {
		return ""
	}
	return strings.TrimRight(windows.UTF16ToString(buf[:n]), " \t\n\v\f\r")
}

// Error formats hr as windows-rs does: the message followed by the code, or the code alone.
func (hr HRESULT) Error() string {
	code := fmt.Sprintf("0x%08X", uint32(hr))
	message := hr.Message()
	if message == "" {
		return code
	}
	return message + " (" + code + ")"
}
