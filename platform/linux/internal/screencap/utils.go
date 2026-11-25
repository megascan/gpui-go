//go:build linux

package screencap

// HasPermission reports whether the process may capture the screen; always true on Linux.
func HasPermission() bool { return linuxHasPermission() }

// RequestPermission prompts for screen-capture permission; Linux has no permission system.
func RequestPermission() bool { return true }

// IsSupported reports whether scap supports this system.
func IsSupported() bool { return linuxIsSupported() }
