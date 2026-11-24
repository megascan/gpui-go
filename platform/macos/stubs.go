//go:build darwin && !ios

package macos

import (
	"sync/atomic"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type rendererContext struct{}

func newRendererContext() rendererContext {
	return rendererContext{}
}

// MacWindow is a macOS window (M4-macos-window ports it).
type MacWindow struct{}

// MacWindowOpen opens a macOS window (M4-macos-window ports it).
func MacWindowOpen(handle gpui.AnyWindowHandle, options gpui.WindowParams, cursorVisible *atomic.Bool, foregroundExecutor gpui.ForegroundExecutor, backgroundExecutor gpui.BackgroundExecutor, rendererContext rendererContext, marker objc.MainThreadMarker) (gpui.PlatformWindow, error) {
	panic("gpui: unimplemented MacWindowOpen")
}

// MacWindowActiveWindow returns the handle of the key window (M4-macos-window ports it).
func MacWindowActiveWindow() (gpui.AnyWindowHandle, bool) {
	panic("gpui: unimplemented MacWindowActiveWindow")
}

// MacWindowOrderedWindows returns the window handles front to back (M4-macos-window ports it).
func MacWindowOrderedWindows() []gpui.AnyWindowHandle {
	panic("gpui: unimplemented MacWindowOrderedWindows")
}

func setActiveWindowCursorStyle(style gpui.CursorStyle) {
	panic("gpui: unimplemented setActiveWindowCursorStyle")
}

func nativeCursor(style gpui.CursorStyle) objc.ID {
	panic("gpui: unimplemented nativeCursor")
}
