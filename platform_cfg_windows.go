package gpui

import (
	"os/exec"
	"syscall"
	"unsafe"
)

// PlatformScreenCaptureFrame is the frame type of the scap screen-capture back end.
type PlatformScreenCaptureFrame = ScapFrame

// GraphicalEnvironment is the graphical session to connect to: the Windows session whose desktop
// windows appear on.
type GraphicalEnvironment struct {
	// SessionID is the session id as ProcessIdToSessionId reports it; None means this process's own session.
	SessionID Option[uint32]
}

var procProcessIDToSessionID = syscall.NewLazyDLL("kernel32.dll").NewProc("ProcessIdToSessionId")

// GraphicalEnvironmentDetect returns the environment of this process's session.
func GraphicalEnvironmentDetect() GraphicalEnvironment {
	var sessionID uint32
	result, _, _ := procProcessIDToSessionID.Call(uintptr(syscall.Getpid()), uintptr(unsafe.Pointer(&sessionID)))
	return GraphicalEnvironment{
		SessionID: OptionFrom(sessionID, result != 0),
	}
}

// ApplyTo does nothing on this platform: programs inherit the session of the process that starts them.
func (e GraphicalEnvironment) ApplyTo(command *exec.Cmd) {}

// DesktopEnvironment reports none: this platform has no desktop environments to tell apart.
func (e GraphicalEnvironment) DesktopEnvironment() (string, bool) {
	return "", false
}
