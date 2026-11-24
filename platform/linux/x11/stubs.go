//go:build linux

package x11

import (
	"errors"
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/xcbshim"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xext"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// XcbAtoms are the atoms the X11 back end interns at startup (M1-x11-window ports window.rs).
type XcbAtoms struct {
	XA_ATOM                          xproto.Atom
	XdndAware                        xproto.Atom
	XdndStatus                       xproto.Atom
	XdndEnter                        xproto.Atom
	XdndLeave                        xproto.Atom
	XdndPosition                     xproto.Atom
	XdndSelection                    xproto.Atom
	XdndDrop                         xproto.Atom
	XdndFinished                     xproto.Atom
	XdndTypeList                     xproto.Atom
	XdndActionCopy                   xproto.Atom
	TextUriList                      xproto.Atom
	UTF8_STRING                      xproto.Atom
	TEXT                             xproto.Atom
	STRING                           xproto.Atom
	TEXT_PLAIN_UTF8                  xproto.Atom
	TEXT_PLAIN                       xproto.Atom
	XDND_DATA                        xproto.Atom
	WM_PROTOCOLS                     xproto.Atom
	WM_DELETE_WINDOW                 xproto.Atom
	WM_CHANGE_STATE                  xproto.Atom
	WM_TRANSIENT_FOR                 xproto.Atom
	_NET_WM_PID                      xproto.Atom
	_NET_WM_NAME                     xproto.Atom
	_NET_WM_ICON                     xproto.Atom
	_NET_WM_STATE                    xproto.Atom
	_NET_WM_STATE_MAXIMIZED_VERT     xproto.Atom
	_NET_WM_STATE_MAXIMIZED_HORZ     xproto.Atom
	_NET_WM_STATE_FULLSCREEN         xproto.Atom
	_NET_WM_STATE_HIDDEN             xproto.Atom
	_NET_WM_STATE_FOCUSED            xproto.Atom
	_NET_ACTIVE_WINDOW               xproto.Atom
	_NET_WM_SYNC_REQUEST             xproto.Atom
	_NET_WM_SYNC_REQUEST_COUNTER     xproto.Atom
	_NET_WM_BYPASS_COMPOSITOR        xproto.Atom
	_NET_WM_MOVERESIZE               xproto.Atom
	_NET_WM_WINDOW_TYPE              xproto.Atom
	_NET_WM_WINDOW_TYPE_NOTIFICATION xproto.Atom
	_NET_WM_WINDOW_TYPE_DIALOG       xproto.Atom
	_NET_WM_STATE_MODAL              xproto.Atom
	_NET_WM_SYNC                     xproto.Atom
	_NET_SUPPORTED                   xproto.Atom
	_MOTIF_WM_HINTS                  xproto.Atom
	_GTK_SHOW_WINDOW_MENU            xproto.Atom
	_GTK_FRAME_EXTENTS               xproto.Atom
	_GTK_EDGE_CONSTRAINTS            xproto.Atom
	_NET_CLIENT_LIST_STACKING        xproto.Atom
}

// NewXcbAtoms interns every atom of XcbAtoms (x11rb's atom_manager: all requests first,
// then all replies).
func NewXcbAtoms(c *xgb.Conn) (XcbAtoms, error) {
	var atoms XcbAtoms
	names := []struct {
		dst  *xproto.Atom
		name string
	}{
		{&atoms.XA_ATOM, "XA_ATOM"}, {&atoms.XdndAware, "XdndAware"}, {&atoms.XdndStatus, "XdndStatus"},
		{&atoms.XdndEnter, "XdndEnter"}, {&atoms.XdndLeave, "XdndLeave"}, {&atoms.XdndPosition, "XdndPosition"},
		{&atoms.XdndSelection, "XdndSelection"}, {&atoms.XdndDrop, "XdndDrop"}, {&atoms.XdndFinished, "XdndFinished"},
		{&atoms.XdndTypeList, "XdndTypeList"}, {&atoms.XdndActionCopy, "XdndActionCopy"},
		{&atoms.TextUriList, "text/uri-list"}, {&atoms.UTF8_STRING, "UTF8_STRING"}, {&atoms.TEXT, "TEXT"},
		{&atoms.STRING, "STRING"}, {&atoms.TEXT_PLAIN_UTF8, "text/plain;charset=utf-8"},
		{&atoms.TEXT_PLAIN, "text/plain"}, {&atoms.XDND_DATA, "XDND_DATA"}, {&atoms.WM_PROTOCOLS, "WM_PROTOCOLS"},
		{&atoms.WM_DELETE_WINDOW, "WM_DELETE_WINDOW"}, {&atoms.WM_CHANGE_STATE, "WM_CHANGE_STATE"},
		{&atoms.WM_TRANSIENT_FOR, "WM_TRANSIENT_FOR"}, {&atoms._NET_WM_PID, "_NET_WM_PID"},
		{&atoms._NET_WM_NAME, "_NET_WM_NAME"}, {&atoms._NET_WM_ICON, "_NET_WM_ICON"},
		{&atoms._NET_WM_STATE, "_NET_WM_STATE"}, {&atoms._NET_WM_STATE_MAXIMIZED_VERT, "_NET_WM_STATE_MAXIMIZED_VERT"},
		{&atoms._NET_WM_STATE_MAXIMIZED_HORZ, "_NET_WM_STATE_MAXIMIZED_HORZ"},
		{&atoms._NET_WM_STATE_FULLSCREEN, "_NET_WM_STATE_FULLSCREEN"}, {&atoms._NET_WM_STATE_HIDDEN, "_NET_WM_STATE_HIDDEN"},
		{&atoms._NET_WM_STATE_FOCUSED, "_NET_WM_STATE_FOCUSED"}, {&atoms._NET_ACTIVE_WINDOW, "_NET_ACTIVE_WINDOW"},
		{&atoms._NET_WM_SYNC_REQUEST, "_NET_WM_SYNC_REQUEST"}, {&atoms._NET_WM_SYNC_REQUEST_COUNTER, "_NET_WM_SYNC_REQUEST_COUNTER"},
		{&atoms._NET_WM_BYPASS_COMPOSITOR, "_NET_WM_BYPASS_COMPOSITOR"}, {&atoms._NET_WM_MOVERESIZE, "_NET_WM_MOVERESIZE"},
		{&atoms._NET_WM_WINDOW_TYPE, "_NET_WM_WINDOW_TYPE"},
		{&atoms._NET_WM_WINDOW_TYPE_NOTIFICATION, "_NET_WM_WINDOW_TYPE_NOTIFICATION"},
		{&atoms._NET_WM_WINDOW_TYPE_DIALOG, "_NET_WM_WINDOW_TYPE_DIALOG"}, {&atoms._NET_WM_STATE_MODAL, "_NET_WM_STATE_MODAL"},
		{&atoms._NET_WM_SYNC, "_NET_WM_SYNC"}, {&atoms._NET_SUPPORTED, "_NET_SUPPORTED"},
		{&atoms._MOTIF_WM_HINTS, "_MOTIF_WM_HINTS"}, {&atoms._GTK_SHOW_WINDOW_MENU, "_GTK_SHOW_WINDOW_MENU"},
		{&atoms._GTK_FRAME_EXTENTS, "_GTK_FRAME_EXTENTS"}, {&atoms._GTK_EDGE_CONSTRAINTS, "_GTK_EDGE_CONSTRAINTS"},
		{&atoms._NET_CLIENT_LIST_STACKING, "_NET_CLIENT_LIST_STACKING"},
	}
	cookies := make([]xproto.InternAtomCookie, len(names))
	for i, n := range names {
		cookies[i] = xproto.InternAtom(c, false, uint16(len(n.name)), n.name)
	}
	for i, cookie := range cookies {
		reply, err := cookie.Reply()
		if err != nil {
			return XcbAtoms{}, err
		}
		*names[i].dst = reply.Atom
	}
	return atoms, nil
}

func xcbFlush(c *xgb.Conn) {}

func checkReply(failureContext string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", failureContext, handleConnectionError(err))
	}
	return nil
}

func getReply[T any](failureContext string, reply *T, err error) (*T, error) {
	if err != nil {
		return nil, fmt.Errorf("%s: %w", failureContext, handleConnectionError(err))
	}
	if reply == nil {
		return nil, fmt.Errorf("%s: %w", failureContext, errConnectionClosed)
	}
	return reply, nil
}

var errConnectionClosed = errors.New("X11 connection: IOError: connection closed")

func handleConnectionError(err error) error {
	return err
}

// X11WindowState is the state of one X11 window (M1-x11-window ports it).
type X11WindowState struct {
	handle          gpui.AnyWindowHandle
	lastSyncCounter gpui.Option[xext.SyncInt64]
}

// X11WindowStatePtr is a shared reference to a window's state (M1-x11-window ports it).
type X11WindowStatePtr struct {
	state   *X11WindowState
	xWindow xproto.Window
}

// X11Window is the platform window (M1-x11-window ports it).
type X11Window struct {
	gpui.PlatformWindow
	ptr X11WindowStatePtr
}

// NewX11Window creates and maps an X11 window (M1-x11-window ports it).
func NewX11Window(handle gpui.AnyWindowHandle, client X11ClientStatePtr, executor gpui.ForegroundExecutor, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], params gpui.WindowParams, conn *xext.Conn, xcb *xcbshim.Connection, clientSideDecorationsSupported bool, xRootIndex int, xWindow xproto.Window, atoms *XcbAtoms, scaleFactor float32, appearance gpui.WindowAppearance, parentWindow gpui.Option[X11WindowStatePtr], supportsXinputGestures bool, isBGR bool) (*X11Window, error) {
	panic("gpui: unimplemented NewX11Window")
}

// ShouldClose asks the window's should-close callback.
func (w X11WindowStatePtr) ShouldClose() bool {
	panic("gpui: unimplemented X11WindowStatePtr.ShouldClose")
}

// PropertyNotify handles a PropertyNotify event on the window.
func (w X11WindowStatePtr) PropertyNotify(event xproto.PropertyNotifyEvent) error {
	panic("gpui: unimplemented X11WindowStatePtr.PropertyNotify")
}

// IsBlocked reports whether a modal child blocks the window.
func (w X11WindowStatePtr) IsBlocked() bool {
	panic("gpui: unimplemented X11WindowStatePtr.IsBlocked")
}

// Close runs the window's close callback.
func (w X11WindowStatePtr) Close() {
	panic("gpui: unimplemented X11WindowStatePtr.Close")
}

// Refresh requests a frame.
func (w X11WindowStatePtr) Refresh(requestFrameOptions gpui.RequestFrameOptions) {
	panic("gpui: unimplemented X11WindowStatePtr.Refresh")
}

// HandleInput dispatches platform input to the window.
func (w X11WindowStatePtr) HandleInput(input gpui.PlatformInput) {
	panic("gpui: unimplemented X11WindowStatePtr.HandleInput")
}

// HandleIMECommit commits IME text.
func (w X11WindowStatePtr) HandleIMECommit(text string) {
	panic("gpui: unimplemented X11WindowStatePtr.HandleIMECommit")
}

// HandleIMEPreedit sets IME preedit text.
func (w X11WindowStatePtr) HandleIMEPreedit(text string) {
	panic("gpui: unimplemented X11WindowStatePtr.HandleIMEPreedit")
}

// HandleIMEUnmark clears the IME marked text.
func (w X11WindowStatePtr) HandleIMEUnmark() {
	panic("gpui: unimplemented X11WindowStatePtr.HandleIMEUnmark")
}

// HandleIMEDelete deletes the IME marked text.
func (w X11WindowStatePtr) HandleIMEDelete() {
	panic("gpui: unimplemented X11WindowStatePtr.HandleIMEDelete")
}

// GetIMEArea returns the IME cursor area in device pixels.
func (w X11WindowStatePtr) GetIMEArea() (gpui.Bounds[gpui.ScaledPixels], bool) {
	panic("gpui: unimplemented X11WindowStatePtr.GetIMEArea")
}

// SetBounds applies a ConfigureNotify.
func (w X11WindowStatePtr) SetBounds(bounds gpui.Bounds[int32]) error {
	panic("gpui: unimplemented X11WindowStatePtr.SetBounds")
}

// SetActive records keyboard focus.
func (w X11WindowStatePtr) SetActive(focus bool) {
	panic("gpui: unimplemented X11WindowStatePtr.SetActive")
}

// SetHovered records pointer hover.
func (w X11WindowStatePtr) SetHovered(focus bool) {
	panic("gpui: unimplemented X11WindowStatePtr.SetHovered")
}

// SetVisibility reports presentation visibility.
func (w X11WindowStatePtr) SetVisibility(visibility gpui.WindowVisibility) {
	panic("gpui: unimplemented X11WindowStatePtr.SetVisibility")
}

// SetAppearance applies a system appearance change.
func (w X11WindowStatePtr) SetAppearance(appearance gpui.WindowAppearance) {
	panic("gpui: unimplemented X11WindowStatePtr.SetAppearance")
}

// SetButtonLayout applies a window button layout change.
func (w X11WindowStatePtr) SetButtonLayout() {
	panic("gpui: unimplemented X11WindowStatePtr.SetButtonLayout")
}

// ClipboardKind selects an X selection (M1-x11-window ports clipboard.rs).
type ClipboardKind uint8

const (
	// ClipboardKindClipboard is CLIPBOARD.
	ClipboardKindClipboard ClipboardKind = iota
	// ClipboardKindPrimary is PRIMARY.
	ClipboardKindPrimary
	// ClipboardKindSecondary is SECONDARY.
	ClipboardKindSecondary
)

// WaitConfig selects how long SetText waits for a clipboard manager (M1-x11-window ports it).
type WaitConfig uint8

const (
	// WaitConfigNone does not wait.
	WaitConfigNone WaitConfig = iota
)

// Clipboard owns and reads X selections (M1-x11-window ports clipboard.rs).
type Clipboard struct{}

// NewClipboard opens the clipboard's own connection to display. The stub returns an empty
// Clipboard so the client can be attached before M1-x11-window ports clipboard.rs.
func NewClipboard(display string) (*Clipboard, error) {
	return &Clipboard{}, nil
}

// SetText offers text on the selection.
func (c *Clipboard) SetText(text string, kind ClipboardKind, wait WaitConfig) error {
	panic("gpui: unimplemented Clipboard.SetText")
}

// GetAny reads the selection.
func (c *Clipboard) GetAny(kind ClipboardKind) (gpui.ClipboardItem, error) {
	panic("gpui: unimplemented Clipboard.GetAny")
}

// IsOwner reports whether this process owns the selection.
func (c *Clipboard) IsOwner(kind ClipboardKind) bool {
	panic("gpui: unimplemented Clipboard.IsOwner")
}

// Close drops the clipboard.
func (c *Clipboard) Close() {}

func scapScreenSources(executor gpui.ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	panic("gpui: unimplemented ScapScreenSources")
}
