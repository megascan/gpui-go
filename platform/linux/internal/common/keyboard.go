//go:build linux

package common

import "github.com/megascan/gpui-go"

// LinuxKeyboardLayout is a keyboard layout named by the display server.
type LinuxKeyboardLayout struct {
	name gpui.SharedString
}

// ID returns the layout's name.
func (l *LinuxKeyboardLayout) ID() string { return string(l.name) }

// Name returns the layout's name.
func (l *LinuxKeyboardLayout) Name() string { return string(l.name) }

// NewLinuxKeyboardLayout returns the layout called name.
func NewLinuxKeyboardLayout(name gpui.SharedString) *LinuxKeyboardLayout {
	return &LinuxKeyboardLayout{name: name}
}
