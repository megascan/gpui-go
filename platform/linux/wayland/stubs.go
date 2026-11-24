//go:build linux

package wayland

import (
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// WaylandWindowState is the state of one Wayland window (M1-wayland-window ports it).
type WaylandWindowState struct{}

// WaylandWindowStatePtr is a shared reference to a window's state (M1-wayland-window ports it).
type WaylandWindowStatePtr struct {
	state *WaylandWindowState
}

// WaylandWindow is the platform window (M1-wayland-window ports it).
type WaylandWindow struct {
	gpui.PlatformWindow
	ptr WaylandWindowStatePtr
}

func newWaylandWindow(handle gpui.AnyWindowHandle, globals *Globals, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], client WaylandClientStatePtr, params gpui.WindowParams, appearance gpui.WindowAppearance, parent gpui.Option[WaylandWindowStatePtr], popupGrab gpui.Option[popupGrab], targetOutput *wlproto.WlOutput) (*WaylandWindow, ObjectID, error) {
	panic("gpui: unimplemented newWaylandWindow")
}

// Handle returns the window's handle.
func (w WaylandWindowStatePtr) Handle() gpui.AnyWindowHandle {
	panic("gpui: unimplemented WaylandWindowStatePtr.Handle")
}

// Surface returns the window's wl_surface.
func (w WaylandWindowStatePtr) Surface() *wlproto.WlSurface {
	panic("gpui: unimplemented WaylandWindowStatePtr.Surface")
}

// Toplevel returns the window's xdg_toplevel, or nil for other surface kinds.
func (w WaylandWindowStatePtr) Toplevel() *wlproto.XdgToplevel {
	panic("gpui: unimplemented WaylandWindowStatePtr.Toplevel")
}

// PtrEq reports whether both pointers refer to the same window.
func (w WaylandWindowStatePtr) PtrEq(other WaylandWindowStatePtr) bool {
	return w.state == other.state
}

// IsBlocked reports whether a modal child blocks the window.
func (w WaylandWindowStatePtr) IsBlocked() bool {
	panic("gpui: unimplemented WaylandWindowStatePtr.IsBlocked")
}

// PrimaryOutputScale returns the integer scale of the window's primary output.
func (w WaylandWindowStatePtr) PrimaryOutputScale() int32 {
	panic("gpui: unimplemented WaylandWindowStatePtr.PrimaryOutputScale")
}

// GetIMEArea returns the IME cursor area.
func (w WaylandWindowStatePtr) GetIMEArea() (gpui.Bounds[gpui.Pixels], bool) {
	panic("gpui: unimplemented WaylandWindowStatePtr.GetIMEArea")
}

// ScheduledFrameFired handles a tick of the frame ping.
func (w WaylandWindowStatePtr) ScheduledFrameFired() {
	panic("gpui: unimplemented WaylandWindowStatePtr.ScheduledFrameFired")
}

// RetryTimerFired handles a frame retry tick.
func (w WaylandWindowStatePtr) RetryTimerFired(deadline time.Time) {
	panic("gpui: unimplemented WaylandWindowStatePtr.RetryTimerFired")
}

// FrameCallbackFired handles a wl_callback.done of the surface's frame callback.
func (w WaylandWindowStatePtr) FrameCallbackFired(signal gpui.Option[scheduler.Instant]) {
	panic("gpui: unimplemented WaylandWindowStatePtr.FrameCallbackFired")
}

// HandleSurfaceEvent handles a wl_surface event.
func (w WaylandWindowStatePtr) HandleSurfaceEvent(event WlSurfaceEvent, outputs map[ObjectID]Output) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleSurfaceEvent")
}

// HandleOutputChanged handles a completed wl_output description.
func (w WaylandWindowStatePtr) HandleOutputChanged(id ObjectID, output Output) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleOutputChanged")
}

// HandleXdgSurfaceEvent handles an xdg_surface event.
func (w WaylandWindowStatePtr) HandleXdgSurfaceEvent(event XdgSurfaceEvent) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleXdgSurfaceEvent")
}

// HandleToplevelEvent handles an xdg_toplevel event and reports whether the window should close.
func (w WaylandWindowStatePtr) HandleToplevelEvent(event XdgToplevelEvent) bool {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleToplevelEvent")
}

// HandleLayersurfaceEvent handles a zwlr_layer_surface_v1 event and reports whether the window
// should close.
func (w WaylandWindowStatePtr) HandleLayersurfaceEvent(event ZwlrLayerSurfaceV1Event) bool {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleLayersurfaceEvent")
}

// HandlePopupEvent handles an xdg_popup event and reports whether the window should close.
func (w WaylandWindowStatePtr) HandlePopupEvent(event XdgPopupEvent) bool {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandlePopupEvent")
}

// Close closes the window.
func (w WaylandWindowStatePtr) Close() {
	panic("gpui: unimplemented WaylandWindowStatePtr.Close")
}

// SetAppearance sets the window's appearance.
func (w WaylandWindowStatePtr) SetAppearance(appearance gpui.WindowAppearance) {
	panic("gpui: unimplemented WaylandWindowStatePtr.SetAppearance")
}

// SetButtonLayout re-reads the button layout.
func (w WaylandWindowStatePtr) SetButtonLayout() {
	panic("gpui: unimplemented WaylandWindowStatePtr.SetButtonLayout")
}

const (
	fileListMimeType = "text/uri-list"
)

var textMimeTypes = [3]string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain"}

// Clipboard holds the clipboard and primary selection contents (M1-wayland-window ports it).
type Clipboard struct{}

// DataOffer is a data offer with the mime types it advertised (M1-wayland-window ports it).
type DataOffer[T any] struct {
	Inner T
}

// NewClipboard creates the clipboard state of a connection.
func NewClipboard(connection *wl.Display, loopHandle *common.LoopHandle) *Clipboard {
	panic("gpui: unimplemented NewClipboard")
}

// Set stores item as the clipboard contents.
func (c *Clipboard) Set(item gpui.ClipboardItem) {
	panic("gpui: unimplemented Clipboard.Set")
}

// SetPrimary stores item as the primary selection contents.
func (c *Clipboard) SetPrimary(item gpui.ClipboardItem) {
	panic("gpui: unimplemented Clipboard.SetPrimary")
}

// SelfMime returns the mime type that marks this process's own offers.
func (c *Clipboard) SelfMime() string {
	panic("gpui: unimplemented Clipboard.SelfMime")
}

// Read reads the clipboard.
func (c *Clipboard) Read() (gpui.ClipboardItem, bool) {
	panic("gpui: unimplemented Clipboard.Read")
}

// ReadPrimary reads the primary selection.
func (c *Clipboard) ReadPrimary() (gpui.ClipboardItem, bool) {
	panic("gpui: unimplemented Clipboard.ReadPrimary")
}

// IMEInput is an input method action for a window (M1-wayland-window ports it).
type IMEInput interface{ isIMEInput() }

// IMEInputInsertText inserts committed text.
type IMEInputInsertText struct{ Text string }

// IMEInputSetMarkedText sets the pre-edit text.
type IMEInputSetMarkedText struct{ Text string }

// IMEInputUnmarkText commits the pre-edit text as is.
type IMEInputUnmarkText struct{}

// IMEInputDeleteText deletes the pre-edit text.
type IMEInputDeleteText struct{}

func (IMEInputInsertText) isIMEInput()    {}
func (IMEInputSetMarkedText) isIMEInput() {}
func (IMEInputUnmarkText) isIMEInput()    {}
func (IMEInputDeleteText) isIMEInput()    {}

// HandleIME applies an input method action.
func (w WaylandWindowStatePtr) HandleIME(ime IMEInput) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleIME")
}

// HandleInput dispatches a platform input event to the window.
func (w WaylandWindowStatePtr) HandleInput(input gpui.PlatformInput) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleInput")
}

// SetFocused sets whether the window has keyboard focus.
func (w WaylandWindowStatePtr) SetFocused(focus bool) {
	panic("gpui: unimplemented WaylandWindowStatePtr.SetFocused")
}

// SetHovered sets whether the pointer is over the window.
func (w WaylandWindowStatePtr) SetHovered(focus bool) {
	panic("gpui: unimplemented WaylandWindowStatePtr.SetHovered")
}

// HandleFractionalScaleEvent handles a wp_fractional_scale_v1 event.
func (w WaylandWindowStatePtr) HandleFractionalScaleEvent(event WpFractionalScaleV1Event) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleFractionalScaleEvent")
}

// HandleToplevelDecorationEvent handles a zxdg_toplevel_decoration_v1 event.
func (w WaylandWindowStatePtr) HandleToplevelDecorationEvent(event ZxdgToplevelDecorationV1Event) {
	panic("gpui: unimplemented WaylandWindowStatePtr.HandleToplevelDecorationEvent")
}

// NewDataOffer wraps offer with no mime types yet.
func NewDataOffer[T any](offer T) DataOffer[T] {
	panic("gpui: unimplemented NewDataOffer")
}

// AddMimeType records a mime type the offer advertised.
func (o *DataOffer[T]) AddMimeType(mimeType string) {
	panic("gpui: unimplemented DataOffer.AddMimeType")
}

// SetOffer sets the current clipboard offer.
func (c *Clipboard) SetOffer(dataOffer gpui.Option[DataOffer[*wlproto.WlDataOffer]]) {
	panic("gpui: unimplemented Clipboard.SetOffer")
}

// SetPrimaryOffer sets the current primary selection offer.
func (c *Clipboard) SetPrimaryOffer(dataOffer gpui.Option[DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]]) {
	panic("gpui: unimplemented Clipboard.SetPrimaryOffer")
}

// Send writes the clipboard contents to fd and closes it.
func (c *Clipboard) Send(mimeType string, fd int) {
	panic("gpui: unimplemented Clipboard.Send")
}

// SendPrimary writes the primary selection contents to fd and closes it.
func (c *Clipboard) SendPrimary(mimeType string, fd int) {
	panic("gpui: unimplemented Clipboard.SendPrimary")
}

// SendBytes writes bytes to fd and closes it.
func (c *Clipboard) SendBytes(fd int, bytes []byte) {
	panic("gpui: unimplemented Clipboard.SendBytes")
}
