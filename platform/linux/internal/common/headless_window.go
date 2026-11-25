//go:build linux

package common

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

type headlessWindowState struct {
	bounds       gpui.Bounds[gpui.Pixels]
	lease        *windowLease
	inputHandler *gpui.PlatformInputHandler
	title        gpui.Option[string]
	isFullscreen bool
}

// HeadlessWindow is a window with no compositor surface and no GPU: layout, text shaping and
// entity plumbing run normally, Draw discards the scene and the sprite atlas hands out tiles
// without uploading pixels. Close is its drop and releases the connection's window lease.
type HeadlessWindow struct {
	gpui.PlatformWindowDefaults
	state *headlessWindowState
}

func newHeadlessWindow(params gpui.WindowParams, lease *windowLease) *HeadlessWindow {
	lease.count++
	w := &HeadlessWindow{state: &headlessWindowState{bounds: params.Bounds, lease: lease}}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w
}

// Close releases the window lease; it runs at most once.
func (w *HeadlessWindow) Close() {
	if w.state.lease != nil {
		w.state.lease.count--
		w.state.lease = nil
	}
}

// RawWindowHandle returns an empty handle: headless windows are not backed by a native window.
func (w *HeadlessWindow) RawWindowHandle() gpui.RawWindowHandle {
	return gpui.RawWindowHandle{Kind: gpui.RawWindowHandleNone}
}

// Bounds returns the window's bounds.
func (w *HeadlessWindow) Bounds() gpui.Bounds[gpui.Pixels] { return w.state.bounds }

// IsMaximized returns false.
func (w *HeadlessWindow) IsMaximized() bool { return false }

// WindowBounds returns the windowed bounds.
func (w *HeadlessWindow) WindowBounds() gpui.WindowBounds {
	return gpui.WindowBoundsWindowed(w.Bounds())
}

// ContentSize returns the size of the bounds.
func (w *HeadlessWindow) ContentSize() gpui.Size[gpui.Pixels] { return w.Bounds().Size }

// Resize sets the size of the bounds.
func (w *HeadlessWindow) Resize(size gpui.Size[gpui.Pixels]) { w.state.bounds.Size = size }

// ScaleFactor returns 1.
func (w *HeadlessWindow) ScaleFactor() float32 { return 1.0 }

// Appearance returns WindowAppearanceDark.
func (w *HeadlessWindow) Appearance() gpui.WindowAppearance { return gpui.WindowAppearanceDark }

// Display returns nil.
func (w *HeadlessWindow) Display() gpui.PlatformDisplay { return nil }

// MousePosition returns the origin.
func (w *HeadlessWindow) MousePosition() gpui.Point[gpui.Pixels] { return gpui.Point[gpui.Pixels]{} }

// Modifiers returns no modifiers.
func (w *HeadlessWindow) Modifiers() gpui.Modifiers { return gpui.Modifiers{} }

// Capslock returns capslock off.
func (w *HeadlessWindow) Capslock() gpui.Capslock { return gpui.Capslock{} }

// SetInputHandler stores the input handler.
func (w *HeadlessWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the input handler.
func (w *HeadlessWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	h := w.state.inputHandler
	w.state.inputHandler = nil
	return h
}

// Prompt returns nil, falling back to GPUI's rendered prompts.
func (w *HeadlessWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	return nil
}

// Activate does nothing.
func (w *HeadlessWindow) Activate() {}

// IsActive returns false.
func (w *HeadlessWindow) IsActive() bool { return false }

// Visibility returns WindowVisibilityHidden: there is no display server, Draw discards every scene.
func (w *HeadlessWindow) Visibility() gpui.WindowVisibility { return gpui.WindowVisibilityHidden }

// IsHovered returns false.
func (w *HeadlessWindow) IsHovered() bool { return false }

// BackgroundAppearance returns WindowBackgroundAppearanceOpaque.
func (w *HeadlessWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return gpui.WindowBackgroundAppearanceOpaque
}

// SetTitle stores the title.
func (w *HeadlessWindow) SetTitle(title string) { w.state.title = gpui.Some(title) }

// GetTitle returns the stored title, or "".
func (w *HeadlessWindow) GetTitle() string { return w.state.title.UnwrapOrDefault() }

// SetBackgroundAppearance does nothing.
func (w *HeadlessWindow) SetBackgroundAppearance(background gpui.WindowBackgroundAppearance) {}

// Minimize does nothing.
func (w *HeadlessWindow) Minimize() {}

// Zoom does nothing.
func (w *HeadlessWindow) Zoom() {}

// ToggleFullscreen flips the fullscreen flag.
func (w *HeadlessWindow) ToggleFullscreen() { w.state.isFullscreen = !w.state.isFullscreen }

// IsFullscreen returns the fullscreen flag.
func (w *HeadlessWindow) IsFullscreen() bool { return w.state.isFullscreen }

// OnRequestFrame drops the callback: no compositor drives a frame loop.
func (w *HeadlessWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {}

// OnInput drops the callback.
func (w *HeadlessWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {}

// OnActiveStatusChange drops the callback.
func (w *HeadlessWindow) OnActiveStatusChange(callback func(active bool)) {}

// OnVisibilityChange drops the callback.
func (w *HeadlessWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {}

// OnHoverStatusChange drops the callback.
func (w *HeadlessWindow) OnHoverStatusChange(callback func(hovered bool)) {}

// OnResize drops the callback.
func (w *HeadlessWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {}

// OnMoved drops the callback.
func (w *HeadlessWindow) OnMoved(callback func()) {}

// OnDisplayChanged drops the callback.
func (w *HeadlessWindow) OnDisplayChanged(callback func()) {}

// OnShouldClose drops the callback.
func (w *HeadlessWindow) OnShouldClose(callback func() bool) {}

// OnClose drops the callback.
func (w *HeadlessWindow) OnClose(callback func()) {}

// OnHitTestWindowControl drops the callback.
func (w *HeadlessWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {}

// OnAppearanceChanged drops the callback.
func (w *HeadlessWindow) OnAppearanceChanged(callback func()) {}

// Draw discards the scene.
func (w *HeadlessWindow) Draw(scene *gpui.Scene) {}

// SpriteAtlas returns a new HeadlessAtlas.
func (w *HeadlessWindow) SpriteAtlas() gpui.PlatformAtlas { return &gpui.HeadlessAtlas{} }

// IsSubpixelRenderingSupported returns false.
func (w *HeadlessWindow) IsSubpixelRenderingSupported() bool { return false }

// UpdateIMEPosition does nothing.
func (w *HeadlessWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {}

// GPUSpecs reports no GPU.
func (w *HeadlessWindow) GPUSpecs() (gpui.GPUSpecs, bool) { return gpui.GPUSpecs{}, false }

var _ gpui.PlatformWindow = (*HeadlessWindow)(nil)
