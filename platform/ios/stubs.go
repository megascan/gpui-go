//go:build ios

package ios

import gpui "github.com/megascan/gpui-go"

// IosWindow is a UIKit window as a gpui.PlatformWindow.
type IosWindow struct {
	gpui.PlatformWindow
}

func newIosWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams, platform *iosPlatformState) (*IosWindow, error) {
	panic("gpui: unimplemented newIosWindow")
}

func (w *IosWindow) register() {
	panic("gpui: unimplemented IosWindow.register")
}
