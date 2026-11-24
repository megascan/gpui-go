//go:build js && wasm

package web

import "github.com/megascan/gpui-go"

// WebKeyboardLayout is the browser's keyboard layout, always reported as US.
type WebKeyboardLayout struct{}

var _ gpui.PlatformKeyboardLayout = WebKeyboardLayout{}

// ID returns "us".
func (WebKeyboardLayout) ID() string {
	return "us"
}

// Name returns "US".
func (WebKeyboardLayout) Name() string {
	return "US"
}
