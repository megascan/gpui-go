//go:build darwin

// Package apple is the shared Apple support of GPUI: Metal rendering, GCD task dispatch and system
// services (Keychain, thermal and power notifications), leaving application lifecycle, windowing
// and input to each platform back end.
package apple

import "github.com/megascan/gpui-go/internal/apple/objc"

// MainThreadMarker returns a main-thread marker when called on the main thread.
func MainThreadMarker() (objc.MainThreadMarker, bool) {
	return objc.NewMainThreadMarker()
}
