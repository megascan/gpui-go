//go:build darwin && !ios

package macos

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

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

func nativeViewsAddWindowMethods() []objc.MethodDef {
	panic("gpui: unimplemented nativeViewsAddWindowMethods")
}

func nativeViewsAddViewMethods() []objc.MethodDef {
	panic("gpui: unimplemented nativeViewsAddViewMethods")
}

func checkPointerAfterFrame(windowState *lockedWindowState) {
	panic("gpui: unimplemented checkPointerAfterFrame")
}

func nativeViewSetsCursor(state *macWindowState) bool {
	panic("gpui: unimplemented nativeViewSetsCursor")
}

func isRoutingKeyEquivalent(windowState *lockedWindowState) bool {
	panic("gpui: unimplemented isRoutingKeyEquivalent")
}

func gpuiViewIsFocused(state *macWindowState) bool {
	panic("gpui: unimplemented gpuiViewIsFocused")
}
