//go:build windows

package screencap

// HasPermission reports whether the process may capture the screen; always true on Windows.
func HasPermission() bool { return true }

// RequestPermission prompts for screen-capture permission; Windows is assumed to grant it.
func RequestPermission() bool { return true }

// IsSupported reports whether scap supports this system.
func IsSupported() bool { return winIsSupported() }
