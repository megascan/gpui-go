//go:build darwin && !ios

package macos

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func windowClass() objc.Class {
	panic("gpui: unimplemented windowClass")
}

func panelClass() objc.Class {
	panic("gpui: unimplemented panelClass")
}

func viewClass() objc.Class {
	panic("gpui: unimplemented viewClass")
}

func blurredViewClass() objc.Class {
	panic("gpui: unimplemented blurredViewClass")
}

func windowStateArchiverDelegateClass() objc.Class {
	panic("gpui: unimplemented windowStateArchiverDelegateClass")
}

func windowStateUnarchiverClass() objc.Class {
	panic("gpui: unimplemented windowStateUnarchiverClass")
}

func isGpuiWindow(window objc.ID) bool {
	panic("gpui: unimplemented isGpuiWindow")
}

func getWindowState(object objc.ID) *lockedWindowState {
	panic("gpui: unimplemented getWindowState")
}

func addMouseTrackingArea(nativeView objc.ID) {
	panic("gpui: unimplemented addMouseTrackingArea")
}

func nativeCursor(style gpui.CursorStyle) objc.ID {
	panic("gpui: unimplemented nativeCursor")
}

func visibility(state *macWindowState) gpui.WindowVisibility {
	panic("gpui: unimplemented visibility")
}

func isMacosVersionAtLeast(version nsOperatingSystemVersion) bool {
	panic("gpui: unimplemented isMacosVersionAtLeast")
}

func nsErrorDescription(err objc.ID) string {
	panic("gpui: unimplemented nsErrorDescription")
}

func stepCallback() gcd.Function {
	panic("gpui: unimplemented stepCallback")
}

func metalFrame(windowState *lockedWindowState, update objc.ID) {
	panic("gpui: unimplemented metalFrame")
}

func displayIDForScreen(screen objc.ID) (uint32, bool) {
	panic("gpui: unimplemented displayIDForScreen")
}

type nativeViews struct {
	background            gpui.Option[gpui.Rgba]
	hitTestCallback       func(position gpui.Point[gpui.Pixels]) (gpui.NativeViewID, bool)
	focusedCallback       func(id gpui.NativeViewID)
	keyDownBeforeCallback func(event gpui.KeyDownEvent) bool
}

type nativeViewLayout struct{}

func nativeViewsPlanLayout(state *macWindowState, scene *gpui.Scene) nativeViewLayout {
	panic("gpui: unimplemented nativeViewsPlanLayout")
}

func nativeViewsUpdateBackground(state *macWindowState, scene *gpui.Scene) {
	panic("gpui: unimplemented nativeViewsUpdateBackground")
}

func nativeViewsSceneDrawn(state *macWindowState, scene *gpui.Scene) {
	panic("gpui: unimplemented nativeViewsSceneDrawn")
}

func nativeViewsApplyLayout(windowState *lockedWindowState, layout nativeViewLayout) {
	panic("gpui: unimplemented nativeViewsApplyLayout")
}

func nativeViewsAttach(windowState *lockedWindowState, id gpui.NativeViewID, view uintptr) gpui.NativeViewAttachment {
	panic("gpui: unimplemented nativeViewsAttach")
}

func nativeViewsSetFocused(windowState *lockedWindowState, id gpui.Option[gpui.NativeViewID]) {
	panic("gpui: unimplemented nativeViewsSetFocused")
}
