//go:build darwin

// Package ios is the native iOS platform for GPUI: it connects GPUI to UIKit, Core Text and Metal
// through the shared Apple renderer, and owns the iOS application lifecycle, windowing, native
// text input, safe-area and keyboard insets, and raw touch delivery.
package ios

// StatusBarContentStyle is the foreground color used by the iOS status bar. The zero value is
// StatusBarContentStyleDark, upstream's default.
type StatusBarContentStyle uint8

const (
	// StatusBarContentStyleDark is dark status-bar content for light backgrounds.
	StatusBarContentStyleDark StatusBarContentStyle = iota
	// StatusBarContentStyleLight is light status-bar content for dark backgrounds.
	StatusBarContentStyleLight
)

// String formats the style as upstream's Debug does.
func (s StatusBarContentStyle) String() string {
	if s == StatusBarContentStyleLight {
		return "Light"
	}
	return "Dark"
}
