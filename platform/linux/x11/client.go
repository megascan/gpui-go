//go:build linux

package x11

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/render"
	"github.com/jezek/xgb/xproto"
	"golang.org/x/sys/unix"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/xcbshim"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/platform/linux/internal/dbus"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xcursor"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xext"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xim"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xrm"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// XinputAllDevices is the DeviceId value that selects all devices.
const XinputAllDevices xext.XinputDeviceID = 0

// XinputAllDeviceGroups is the DeviceId value that selects all device groups ("master devices").
const XinputAllDeviceGroups xext.XinputDeviceID = 1

const gpuiX11ScaleFactorEnv = "GPUI_X11_SCALE_FACTOR"

type windowRef struct {
	window         X11WindowStatePtr
	refreshState   *refreshState
	lastVisibility byte
	isMapped       bool
}

func (w *windowRef) handle() gpui.AnyWindowHandle {
	return w.window.state.handle
}

func (w *windowRef) visibility() gpui.WindowVisibility {
	if w.isMapped && w.lastVisibility != xproto.VisibilityFullyObscured {
		return gpui.WindowVisibilityVisible
	}
	return gpui.WindowVisibilityHidden
}

type refreshState struct {
	periodic       bool
	refreshRate    time.Duration
	eventLoopToken common.RegistrationToken
}

// EventHandlerErrorKind selects an EventHandlerError variant.
type EventHandlerErrorKind uint8

const (
	// EventHandlerErrorXCBConnectionError is a failure of the X connection.
	EventHandlerErrorXCBConnectionError EventHandlerErrorKind = iota
	// EventHandlerErrorXIMClientError is a failure of the XIM client.
	EventHandlerErrorXIMClientError
)

// EventHandlerError is an error that stops X11 event processing.
type EventHandlerError struct {
	Kind EventHandlerErrorKind
	Err  error
}

func (e *EventHandlerError) Error() string { return e.Err.Error() }

func (e *EventHandlerError) Unwrap() error { return e.Err }

// Xdnd is the state of an XDND drag over one of the client's windows.
type Xdnd struct {
	otherWindow xproto.Window
	dragType    uint32
	retrieved   bool
	position    gpui.Point[gpui.Pixels]
}

type pointerDeviceState struct {
	horizontal scrollAxisState
	vertical   scrollAxisState
}

type scrollAxisState struct {
	valuatorNumber gpui.Option[uint16]
	multiplier     float32
	scrollValue    gpui.Option[float32]
}

type xErrorEvent struct{ err xgb.Error }

func (e xErrorEvent) Bytes() []byte  { return nil }
func (e xErrorEvent) String() string { return e.err.Error() }

type eventQueue struct {
	mu     sync.Mutex
	items  []xgb.Event
	closed bool
	ping   *common.Ping
}

func (q *eventQueue) pump(conn *xext.Conn) {
	for {
		ev, err := conn.WaitForEvent()
		q.mu.Lock()
		switch {
		case ev == nil && err == nil:
			q.closed = true
		case err != nil:
			q.items = append(q.items, xErrorEvent{err: err})
		default:
			q.items = append(q.items, ev)
		}
		closed := q.closed
		q.mu.Unlock()
		q.ping.Ping()
		if closed {
			return
		}
	}
}

func (q *eventQueue) poll() (xgb.Event, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) > 0 {
		ev := q.items[0]
		q.items[0] = nil
		q.items = q.items[1:]
		return ev, true, nil
	}
	if q.closed {
		return nil, false, errConnectionClosed
	}
	return nil, false, nil
}

// X11ClientState is the state of one X server connection.
type X11ClientState struct {
	this                 X11ClientStatePtr
	loopHandle           *common.LoopHandle
	registrations        []common.RegistrationToken
	xdpSource            *dbus.XDPEventSource
	graphicalEnvironment gpui.GraphicalEnvironment

	lastClick       time.Time
	lastMouseButton gpui.Option[gpui.MouseButton]
	lastLocation    gpui.Point[gpui.Pixels]
	currentCount    int
	pinchScale      float32

	gpuContext    *wgpu.GPUContext
	compositorGPU gpui.Option[wgpu.CompositorGPUHint]

	scaleFactor float32

	xkbContext                     *xkb.Context
	conn                           *xext.Conn
	xcb                            *xcbshim.Connection
	events                         *eventQueue
	xkbDeviceID                    int32
	clientSideDecorationsSupported bool
	xRootIndex                     int
	resourceDatabase               *xrm.Database
	atoms                          XcbAtoms
	windows                        map[xproto.Window]*windowRef
	mouseFocusedWindow             gpui.Option[xproto.Window]
	keyboardFocusedWindow          gpui.Option[xproto.Window]
	xkb                            *xkb.State
	keyboardLayout                 *common.LinuxKeyboardLayout
	ximc                           *xim.Client
	ximHandler                     *XimHandler
	// Modifiers is the last modifier state seen in an input event.
	Modifiers gpui.Modifiers
	// Capslock is the last capslock state seen.
	Capslock gpui.Capslock
	// LastModifiersChangedEvent is the modifier state of the last ModifiersChanged event.
	LastModifiersChangedEvent gpui.Modifiers
	// LastCapslockChangedEvent is the capslock state of the last ModifiersChanged event.
	LastCapslockChangedEvent gpui.Capslock

	composeState         *xkb.ComposeState
	preEditText          gpui.Option[string]
	composing            bool
	preKeyCharDown       gpui.Option[gpui.Keystroke]
	cursorHandle         *xcursor.Handle
	cursorStyles         map[xproto.Window]gpui.CursorStyle
	cursorCache          map[gpui.CursorStyle]gpui.Option[xproto.Cursor]
	invisibleCursorCache gpui.Option[xproto.Cursor]
	cursorHiddenWindow   gpui.Option[xproto.Window]

	pointerDeviceStates map[xext.XinputDeviceID]*pointerDeviceState

	supportsXinputGestures bool

	common        *common.LinuxCommon
	clipboard     *Clipboard
	clipboardItem gpui.Option[gpui.ClipboardItem]
	xdndState     Xdnd
}

type clientCell struct{ state *X11ClientState }

// X11ClientStatePtr is a weak reference to a client's state; it stops resolving once the
// connection is closed.
type X11ClientStatePtr struct{ cell *clientCell }

// GetClient returns the client while its connection is open.
func (p X11ClientStatePtr) GetClient() (X11Client, bool) {
	if p.cell == nil || p.cell.state == nil {
		return X11Client{}, false
	}
	return X11Client{s: p.cell.state}, true
}

// DropWindow forgets a destroyed window.
func (p X11ClientStatePtr) DropWindow(xWindow uint32) {
	client, ok := p.GetClient()
	if !ok {
		return
	}
	state := client.s
	w := xproto.Window(xWindow)
	if windowRef, ok := state.windows[w]; ok {
		delete(state.windows, w)
		if rs := windowRef.refreshState; rs != nil && rs.periodic {
			state.loopHandle.Remove(rs.eventLoopToken)
		}
	}
	if state.mouseFocusedWindow == gpui.Some(w) {
		state.mouseFocusedWindow = gpui.None[xproto.Window]()
	}
	if state.keyboardFocusedWindow == gpui.Some(w) {
		state.keyboardFocusedWindow = gpui.None[xproto.Window]()
	}
	if state.cursorHiddenWindow == gpui.Some(w) {
		state.cursorHiddenWindow = gpui.None[xproto.Window]()
	}
	delete(state.cursorStyles, w)
}

func spotLocation(area gpui.Bounds[gpui.ScaledPixels]) xim.Point {
	return xim.Point{
		X: int16((area.Origin.X + area.Size.Width).ToU32()),
		Y: int16((area.Origin.Y + area.Size.Height).ToU32()),
	}
}

// UpdateIMEPosition moves the XIM preedit spot to the bottom-right of bounds.
func (p X11ClientStatePtr) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	client, ok := p.GetClient()
	if !ok {
		return
	}
	state := client.s
	if state.composing || state.ximc == nil {
		return
	}
	ximc := state.ximc
	state.ximc = nil
	xh := state.ximHandler
	if xh == nil {
		slog.Error("bug: xim handler not set")
		state.ximc = ximc
		return
	}
	state.ximHandler = nil
	scaled := bounds.Scale(state.scaleFactor)
	icAttributes := ximc.BuildICAttributes().
		Long(xim.AttrInputStyle, uint32(xim.InputStylePreeditCallbacks)).
		Long(xim.AttrClientWindow, uint32(xh.Window)).
		Long(xim.AttrFocusWindow, uint32(xh.Window)).
		Nested(xim.AttrPreeditAttributes, func(b *xim.AttributeBuilder) {
			b.Point(xim.AttrSpotLocation, spotLocation(scaled))
		}).
		Build()
	if err := ximc.SetICValues(xh.ImID, xh.IcID, icAttributes); err != nil {
		slog.Error(err.Error())
	}
	state.ximc = ximc
	state.ximHandler = xh
}

// X11Connection is a connection to an X server. Close removes its event-loop sources,
// forgets its windows and closes the X connection.
type X11Connection struct {
	X11Client
	closed bool
}

// Close is the connection's drop.
func (c *X11Connection) Close() {
	if c.closed {
		return
	}
	c.closed = true
	state := c.s
	state.common.AfterRunnable = nil
	for _, token := range state.registrations {
		state.loopHandle.Remove(token)
	}
	state.registrations = nil
	if state.xdpSource != nil {
		state.xdpSource.Close()
		state.xdpSource = nil
	}
	for _, window := range state.windows {
		if ws := window.window.state; ws != nil && ws.renderer != nil {
			renderer := ws.renderer
			ws.renderer = nil
			renderer.Destroy()
		}
		if rs := window.refreshState; rs != nil {
			window.refreshState = nil
			if rs.periodic {
				state.loopHandle.Remove(rs.eventLoopToken)
			}
		}
	}
	state.windows = map[xproto.Window]*windowRef{}
	state.this.cell.state = nil
	if state.composeState != nil {
		state.composeState.Close()
	}
	if state.xkb != nil {
		state.xkb.Close()
	}
	state.xkbContext.Close()
	state.clipboard.Close()
	state.conn.X.Close()
	if state.xcb != nil {
		state.xcb.Close()
	}
}

var initExtensionsOnce sync.Mutex

// X11ConnectionAttach connects to the X server environment names and registers its event
// sources on handle. Closing the connection removes those sources, leaving the loop and
// linuxCommon usable without X11.
func X11ConnectionAttach(handle *common.LoopHandle, linuxCommon *common.LinuxCommon, environment gpui.GraphicalEnvironment) (*X11Connection, error) {
	display, ok := environment.X11Display.Get()
	if !ok || display == "" {
		return nil, errors.New("DISPLAY is not set")
	}
	if !utf8.ValidString(display) {
		return nil, errors.New("DISPLAY is not valid UTF-8")
	}
	if strings.ContainsRune(display, 0) {
		return nil, errors.New("X display name contains a NUL byte")
	}
	conn, err := xext.Dial(display)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to X server %q: %w", display, err)
	}
	ok = false
	defer func() {
		if !ok {
			conn.X.Close()
		}
	}()
	xRootIndex := conn.X.DefaultScreen

	initExtensionsOnce.Lock()
	xkbErr := xext.InitXkb(conn.X)
	randrErr := randr.Init(conn.X)
	_ = render.Init(conn.X)
	xinputErr := xext.InitXinput(conn.X)
	_ = xext.InitSync(conn.X)
	dri3Err := xext.InitDri3(conn.X)
	initExtensionsOnce.Unlock()
	_ = randrErr

	if xinputErr != nil {
		return nil, fmt.Errorf("XInput XiQueryVersion failed: %w", xinputErr)
	}
	xinputVersion, err := checkedReply("XInput XiQueryVersion failed", replyOf(xinputQueryVersion(conn.X)))
	if err != nil {
		return nil, err
	}
	if xinputVersion.MajorVersion < 2 {
		return nil, errors.New("XInput version >= 2 required.")
	}
	supportsXinputGestures := xinputVersion.MajorVersion > 2 || (xinputVersion.MajorVersion == 2 && xinputVersion.MinorVersion >= 4)
	slog.Info(fmt.Sprintf("XInput version: %d.%d, gesture support: %v", xinputVersion.MajorVersion, xinputVersion.MinorVersion, supportsXinputGestures))

	pointerDeviceStates, ok2 := currentPointerDeviceStates(conn.X, nil)
	if !ok2 {
		pointerDeviceStates = map[xext.XinputDeviceID]*pointerDeviceState{}
	}

	atoms, err := NewXcbAtoms(conn.X)
	if err != nil {
		return nil, fmt.Errorf("Failed to get XCB atoms: %w", err)
	}

	setup := xproto.Setup(conn.X)
	root := setup.Roots[0].Root
	compositorPresent := checkCompositorPresent(conn.X, root)
	gtkFrameExtentsSupported := checkGtkFrameExtentsSupported(conn.X, &atoms, root)
	clientSideDecorationsSupported := compositorPresent && gtkFrameExtentsSupported
	slog.Info(fmt.Sprintf("x11: compositor present: %v, gtk_frame_extents_supported: %v", compositorPresent, gtkFrameExtentsSupported))

	if xkbErr != nil {
		return nil, fmt.Errorf("Failed to initialize XKB extension: %w", xkbErr)
	}
	xkbReply, err := checkedReply("Failed to initialize XKB extension", replyOf(xkbUseExtension(conn.X)))
	if err != nil {
		return nil, err
	}
	if !xkbReply.Supported {
		return nil, errors.New("X server does not support XKB")
	}
	events := xext.XkbEventTypeStateNotify | xext.XkbEventTypeMapNotify | xext.XkbEventTypeNewKeyboardNotify
	mapNotifyParts := xext.XkbMapPartKeyTypes | xext.XkbMapPartKeySyms | xext.XkbMapPartModifierMap |
		xext.XkbMapPartExplicitComponents | xext.XkbMapPartKeyActions | xext.XkbMapPartKeyBehaviors |
		xext.XkbMapPartVirtualMods | xext.XkbMapPartVirtualModMap
	if err := checkReply("Failed to select XKB events", xext.XkbSelectEventsChecked(conn.X, xext.XkbIDUseCoreKbd, 0, events, mapNotifyParts, mapNotifyParts, xext.XkbSelectEventsAux{}).Check()); err != nil {
		return nil, err
	}

	xcb, err := xcbshim.Connect(display)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to X server %q: %w", display, err)
	}
	defer func() {
		if !ok {
			xcb.Close()
		}
	}()
	if !xkb.X11SetupXkbExtension(xcb.Raw(), xkb.X11MinMajorXkbVersion, xkb.X11MinMinorXkbVersion, xkb.X11SetupXkbExtensionNoFlags, nil, nil, nil, nil) {
		return nil, errors.New("X server does not support XKB")
	}

	xkbContext, err := common.NewXkbContext()
	if err != nil {
		return nil, err
	}
	xkbDeviceID := xkb.X11GetCoreKeyboardDeviceID(xcb.Raw())
	xkbState, err := stateFromDevice(xkbContext, xcb, xkbDeviceID)
	if err != nil {
		xkbContext.Close()
		return nil, err
	}
	composeState, hasCompose := common.GetXkbComposeState(xkbContext)
	if !hasCompose {
		composeState = nil
	}
	layoutIdx := xkbState.SerializeLayout(xkb.StateLayoutEffective)
	keymap := xkbState.GetKeymap()
	layoutName := keymap.LayoutGetName(layoutIdx)
	keymap.Close()
	keyboardLayout := common.NewLinuxKeyboardLayout(gpui.SharedString(layoutName))

	resourceDatabase, err := newResourceDatabase(conn.X)
	if err != nil {
		return nil, fmt.Errorf("Failed to create resource database: %w", err)
	}
	scaleFactor := getScaleFactor(conn.X, resourceDatabase, xRootIndex)
	screen := &setup.Roots[xRootIndex]
	cursorHandle, err := xcursor.NewHandle(conn.X, screen, resourceDatabase)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize cursor theme handler: %w", err)
	}

	clipboard, err := NewClipboard(display)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize clipboard: %w", err)
	}

	compositorGPU := gpui.None[wgpu.CompositorGPUHint]()
	if dri3Err == nil {
		compositorGPU = detectCompositorGPU(conn, screen)
	}

	ximc, ximErr := initXimClient(conn.X, screen)
	var ximHandler *XimHandler
	if ximErr == nil {
		ximHandler = NewXimHandler()
	} else {
		ximc = nil
	}

	ping, pingSource, err := common.MakePing()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize X11 event source: %w", err)
	}

	cell := &clientCell{}
	state := &X11ClientState{
		this:                 X11ClientStatePtr{cell: cell},
		graphicalEnvironment: environment,
		loopHandle:           handle,
		common:               linuxCommon,
		lastClick:            time.Now(),
		pinchScale:           1.0,
		gpuContext:           &wgpu.GPUContext{},
		compositorGPU:        compositorGPU,
		scaleFactor:          scaleFactor,

		xkbContext:                     xkbContext,
		conn:                           conn,
		xcb:                            xcb,
		events:                         &eventQueue{ping: ping},
		xkbDeviceID:                    xkbDeviceID,
		clientSideDecorationsSupported: clientSideDecorationsSupported,
		xRootIndex:                     xRootIndex,
		resourceDatabase:               resourceDatabase,
		atoms:                          atoms,
		windows:                        map[xproto.Window]*windowRef{},
		xkb:                            xkbState,
		keyboardLayout:                 keyboardLayout,
		ximc:                           ximc,
		ximHandler:                     ximHandler,
		composeState:                   composeState,
		cursorHandle:                   cursorHandle,
		cursorStyles:                   map[xproto.Window]gpui.CursorStyle{},
		cursorCache:                    map[gpui.CursorStyle]gpui.Option[xproto.Cursor]{},
		pointerDeviceStates:            pointerDeviceStates,
		supportsXinputGestures:         supportsXinputGestures,
		clipboard:                      clipboard,
	}
	state.graphicalEnvironment.ActivationToken = gpui.None[string]()
	cell.state = state
	connection := &X11Connection{X11Client: X11Client{s: state}}
	ok = true
	go state.events.pump(conn)
	if err := connection.registerSources(handle, pingSource); err != nil {
		connection.Close()
		return nil, err
	}
	xcbFlush(conn.X)
	return connection, nil
}

func xinputQueryVersion(c *xgb.Conn) (*xext.XinputXIQueryVersionReply, error) {
	return xext.XinputXIQueryVersion(c, 2, 4).Reply()
}

func xkbUseExtension(c *xgb.Conn) (*xext.XkbUseExtensionReply, error) {
	return xext.XkbUseExtension(c, xkb.X11MinMajorXkbVersion, xkb.X11MinMinorXkbVersion).Reply()
}

func stateFromDevice(ctx *xkb.Context, xcb *xcbshim.Connection, deviceID int32) (*xkb.State, error) {
	keymap := xkb.X11KeymapNewFromDevice(ctx, xcb.Raw(), deviceID, xkb.KeymapCompileNoFlags)
	if keymap == nil {
		return nil, errors.New("failed to get the XKB keymap of the core keyboard")
	}
	defer keymap.Close()
	return xkb.X11StateNewFromDevice(keymap, xcb.Raw(), deviceID), nil
}

func newResourceDatabase(c *xgb.Conn) (*xrm.Database, error) {
	root := xproto.Setup(c).Roots[0].Root
	reply, err := xproto.GetProperty(c, false, root, xrm.AtomResourceManager, xrm.AtomString, 0, xrm.GetResourceDatabaseLongLength).Reply()
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errConnectionClosed
	}
	hostname, _ := os.Hostname()
	return xrm.NewFromDefault(reply.Format, reply.Value, hostname), nil
}

// X11Client is a handle to a client's state.
type X11Client struct{ s *X11ClientState }

func (c X11Client) registerSources(handle *common.LoopHandle, pingSource *common.PingSource) error {
	this := c.s.this
	token, err := common.InsertSource[struct{}](handle, pingSource, func(struct{}) {
		client, ok := this.GetClient()
		if !ok {
			return
		}
		if err := client.processX11Events(); err != nil {
			slog.Error(err.Error())
		}
	})
	if err != nil {
		return fmt.Errorf("Failed to initialize X11 event source: %w", err)
	}
	c.s.registrations = append(c.s.registrations, token)

	if bus, err := c.s.common.SessionBus(); err == nil {
		sender, channel, err := common.NewChannel[dbus.Event]()
		if err != nil {
			return fmt.Errorf("Failed to initialize XDP event source: %w", err)
		}
		token, err := common.InsertSource[common.ChannelEvent[dbus.Event]](handle, channel, func(ev common.ChannelEvent[dbus.Event]) {
			if ev.Closed {
				return
			}
			client, ok := this.GetClient()
			if !ok {
				return
			}
			client.handleXDPEvent(ev.Msg)
		})
		if err != nil {
			sender.Close()
			return fmt.Errorf("Failed to initialize XDP event source: %w", err)
		}
		c.s.registrations = append(c.s.registrations, token)
		c.s.xdpSource = dbus.NewXDPEventSource(bus, func(ev dbus.Event) { _ = sender.Send(ev) })
	}

	c.s.common.AfterRunnable = func() {
		if client, ok := this.GetClient(); ok {
			if err := client.processX11Events(); err != nil {
				slog.Error(err.Error())
			}
		}
	}
	return nil
}

func (c X11Client) handleXDPEvent(event dbus.Event) {
	switch ev := event.(type) {
	case dbus.EventWindowAppearance:
		c.withCommon(func(common *common.LinuxCommon) { common.Appearance = ev.Appearance })
		for _, window := range c.s.windows {
			window.window.SetAppearance(ev.Appearance)
		}
	case dbus.EventButtonLayout:
		layout, err := gpui.WindowButtonLayoutParse(ev.Layout)
		if err != nil {
			slog.Error(err.Error())
			layout = gpui.WindowButtonLayoutLinuxDefault()
		}
		c.withCommon(func(common *common.LinuxCommon) { common.ButtonLayout = layout })
		for _, window := range c.s.windows {
			window.window.SetButtonLayout()
		}
	case dbus.EventCursorTheme, dbus.EventCursorSize:
	case dbus.EventTextScaleFactor:
	}
}

// HasWindows reports whether the client has open windows.
func (c X11Client) HasWindows() bool { return len(c.s.windows) > 0 }

// GraphicalEnvironment returns the environment the connection was made in, without its
// activation token.
func (c X11Client) GraphicalEnvironment() gpui.GraphicalEnvironment {
	return c.s.graphicalEnvironment
}

func (c X11Client) processX11Events() error {
	for {
		var events []xgb.Event
		var windowsToRefresh []xproto.Window
		refreshSet := map[xproto.Window]bool{}
		var lastKeyRelease xgb.Event
		var lastKeymapChangeEvent xgb.Event

		for {
			event, ok, err := c.s.events.poll()
			if err != nil {
				return &EventHandlerError{Kind: EventHandlerErrorXCBConnectionError, Err: err}
			}
			if !ok {
				break
			}
			switch ev := event.(type) {
			case xproto.ExposeEvent:
				if !refreshSet[ev.Window] {
					refreshSet[ev.Window] = true
					windowsToRefresh = append(windowsToRefresh, ev.Window)
				}
			case xproto.KeyReleaseEvent:
				if lastKeymapChangeEvent != nil {
					if lastKeyRelease != nil {
						events = append(events, lastKeyRelease)
						lastKeyRelease = nil
					}
					events = append(events, lastKeymapChangeEvent)
					lastKeymapChangeEvent = nil
				}
				lastKeyRelease = event
			case xproto.KeyPressEvent:
				if lastKeymapChangeEvent != nil {
					if lastKeyRelease != nil {
						events = append(events, lastKeyRelease)
						lastKeyRelease = nil
					}
					events = append(events, lastKeymapChangeEvent)
					lastKeymapChangeEvent = nil
				}
				if keyRelease, ok := lastKeyRelease.(xproto.KeyReleaseEvent); ok {
					if keyRelease.Detail != ev.Detail || saturatingSub(uint32(ev.Time), uint32(keyRelease.Time)) > 20 {
						events = append(events, keyRelease)
					}
				}
				lastKeyRelease = nil
				events = append(events, ev)
			case xext.XkbNewKeyboardNotifyEvent, xext.XkbMapNotifyEvent:
				if lastKeyRelease != nil {
					events = append(events, lastKeyRelease)
					lastKeyRelease = nil
				}
				lastKeymapChangeEvent = event
			default:
				if lastKeyRelease != nil {
					events = append(events, lastKeyRelease)
					lastKeyRelease = nil
				}
				events = append(events, event)
			}
		}

		if lastKeyRelease != nil {
			events = append(events, lastKeyRelease)
		}
		if lastKeymapChangeEvent != nil {
			events = append(events, lastKeymapChangeEvent)
		}

		if len(events) == 0 && len(windowsToRefresh) == 0 {
			break
		}

		for _, event := range events {
			state := c.s
			if !state.hasXim() {
				c.handleEvent(event)
				continue
			}
			ximc, xh, ok := state.takeXim()
			if !ok {
				continue
			}
			ximConnected := xh.Connected

			handled, err := filterXimEvent(ximc, event, xh)
			ximCallbackEvent := xh.LastCallbackEvent
			xh.LastCallbackEvent = nil

			state.restoreXim(ximc, xh)

			if ximCallbackEvent != nil {
				c.handleXimCallbackEvent(ximCallbackEvent)
			}

			if err != nil {
				slog.Error("XIMClientError: " + err.Error())
				state.takeXim()
				c.handleEvent(event)
				continue
			}
			if handled {
				continue
			}
			if ximConnected {
				c.ximHandleEvent(event)
			} else {
				c.handleEvent(event)
			}
		}

		for _, xWindow := range windowsToRefresh {
			windowRef, ok := c.s.windows[xWindow]
			if !ok || !windowRef.isMapped {
				continue
			}
			windowRef.window.Refresh(gpui.RequestFrameOptions{
				RequirePresentation: true,
				ForceRender:         false,
				SignalAt:            gpui.None[scheduler.Instant](),
				SignalSource:        gpui.FrameRequestSourceNativeCallback,
			})
		}
	}
	return nil
}

func saturatingSub(a, b uint32) uint32 {
	if a < b {
		return 0
	}
	return a - b
}

func filterXimEvent(ximc *xim.Client, event xgb.Event, h *XimHandler) (bool, error) {
	var ev xim.Event
	switch e := event.(type) {
	case xproto.SelectionNotifyEvent:
		ev = xim.SelectionNotify{Requestor: uint32(e.Requestor), Selection: uint32(e.Selection), Target: uint32(e.Target), Property: uint32(e.Property)}
	case xproto.ClientMessageEvent:
		msg := xim.ClientMessage{Window: uint32(e.Window), Type: uint32(e.Type), Format: e.Format}
		switch e.Format {
		case 32:
			for i, v := range e.Data.Data32 {
				if i < 5 {
					binary.LittleEndian.PutUint32(msg.Data[i*4:], v)
				}
			}
		case 16:
			for i, v := range e.Data.Data16 {
				if i < 10 {
					binary.LittleEndian.PutUint16(msg.Data[i*2:], v)
				}
			}
		default:
			copy(msg.Data[:], e.Data.Data8)
		}
		ev = msg
	case xErrorEvent:
		werr, ok := e.err.(xproto.WindowError)
		if !ok {
			return false, nil
		}
		ev = xim.WindowError{BadValue: werr.BadValue}
	default:
		return false, nil
	}
	return ximc.FilterEvent(ev, h)
}

// EnableIME creates the XIM input context for the focused window.
func (c X11Client) EnableIME() {
	state := c.s
	if !state.hasXim() {
		return
	}
	ximc, xh, ok := state.takeXim()
	if !ok {
		return
	}
	icAttributes := ximc.BuildICAttributes().
		Long(xim.AttrInputStyle, uint32(xim.InputStylePreeditCallbacks)).
		Long(xim.AttrClientWindow, uint32(xh.Window)).
		Long(xim.AttrFocusWindow, uint32(xh.Window))

	if windowID, ok := state.keyboardFocusedWindow.Get(); ok {
		window, ok := c.getWindow(windowID)
		if !ok {
			slog.Error("Failed to get window for IME positioning")
			state.ximc = ximc
			state.ximHandler = xh
			return
		}
		if scaledArea, ok := window.GetIMEArea(); ok {
			icAttributes = icAttributes.Nested(xim.AttrPreeditAttributes, func(b *xim.AttributeBuilder) {
				b.Point(xim.AttrSpotLocation, spotLocation(scaledArea))
			})
		}
	}
	_ = ximc.CreateIC(xh.ImID, icAttributes.Build())
	state.restoreXim(ximc, xh)
}

// ResetIME resets the XIM input context.
func (c X11Client) ResetIME() {
	state := c.s
	state.composing = false
	if ximc := state.ximc; ximc != nil {
		state.ximc = nil
		if xh := state.ximHandler; xh != nil {
			_ = ximc.ResetIC(xh.ImID, xh.IcID)
		} else {
			slog.Error("bug: xim handler not set in reset_ime")
		}
		state.ximc = ximc
	}
}

func (c X11Client) getWindow(win xproto.Window) (X11WindowStatePtr, bool) {
	windowRef, ok := c.s.windows[win]
	if !ok {
		return X11WindowStatePtr{}, false
	}
	return windowRef.window, true
}

func eventPosition(x, y xext.XinputFp1616, scaleFactor float32) gpui.Point[gpui.Pixels] {
	return gpui.Point[gpui.Pixels]{
		X: gpui.Px(float32(x) / float32(math.MaxUint16) / scaleFactor),
		Y: gpui.Px(float32(y) / float32(math.MaxUint16) / scaleFactor),
	}
}

func (c X11Client) handleEvent(event xgb.Event) {
	switch ev := event.(type) {
	case xproto.UnmapNotifyEvent:
		if windowRef, ok := c.s.windows[ev.Window]; ok {
			windowRef.isMapped = false
		}
		handleVisibilityChanged(c.s, ev.Window)
	case xproto.MapNotifyEvent:
		if windowRef, ok := c.s.windows[ev.Window]; ok {
			windowRef.isMapped = true
		}
		handleVisibilityChanged(c.s, ev.Window)
	case xproto.VisibilityNotifyEvent:
		if windowRef, ok := c.s.windows[ev.Window]; ok {
			windowRef.lastVisibility = ev.State
		}
		handleVisibilityChanged(c.s, ev.Window)
	case xproto.ClientMessageEvent:
		c.handleClientMessage(ev)
	case xproto.SelectionNotifyEvent:
		c.handleSelectionNotify(ev)
	case xproto.ConfigureNotifyEvent:
		bounds := gpui.Bounds[int32]{
			Origin: gpui.Point[int32]{X: int32(ev.X), Y: int32(ev.Y)},
			Size:   gpui.Size[int32]{Width: int32(ev.Width), Height: int32(ev.Height)},
		}
		window, ok := c.getWindow(ev.Window)
		if !ok {
			return
		}
		if err := window.SetBounds(bounds); err != nil {
			slog.Error(fmt.Errorf("X11: Failed to set window bounds: %w", err).Error())
		}
	case xproto.PropertyNotifyEvent:
		window, ok := c.getWindow(ev.Window)
		if !ok {
			return
		}
		if err := window.PropertyNotify(ev); err != nil {
			slog.Error(fmt.Errorf("X11: Failed to handle property notify: %w", err).Error())
		}
	case xproto.FocusInEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		window.SetActive(true)
		c.s.keyboardFocusedWindow = gpui.Some(ev.Event)
		if c.s.ximHandler != nil {
			c.s.ximHandler.Window = ev.Event
		}
		c.EnableIME()
	case xproto.FocusOutEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		window.SetActive(false)
		state := c.s
		resetAllPointerDeviceScrollPositions(state.pointerDeviceStates)
		state.keyboardFocusedWindow = gpui.None[xproto.Window]()
		if state.composeState != nil {
			state.composeState.Reset()
		}
		state.preEditText = gpui.None[string]()
		state.restoreCursorAfterHide()
		c.ResetIME()
		window.HandleIMEDelete()
	case xext.XkbNewKeyboardNotifyEvent, xext.XkbMapNotifyEvent:
		state := c.s
		xkbState, err := stateFromDevice(state.xkbContext, state.xcb, state.xkbDeviceID)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		state.xkb.Close()
		state.xkb = xkbState
		c.handleKeyboardLayoutChange()
	case xext.XkbStateNotifyEvent:
		c.handleXkbStateNotify(ev)
	case xproto.KeyPressEvent:
		c.handleKeyPress(ev)
	case xproto.KeyReleaseEvent:
		c.handleKeyRelease(ev)
	case xext.XinputButtonPressEvent:
		c.handleXinputButtonPress(ev)
	case xext.XinputButtonReleaseEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		state := c.s
		modifiers := modifiersFromXinputInfo(ev.Mods)
		state.Modifiers = modifiers
		position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
		bos, ok := buttonOrScrollFromEventDetail(ev.Detail)
		if ok && bos.kind == buttonOrScrollButton {
			window.HandleInput(gpui.MouseUpEvent{
				Button:     bos.button,
				Position:   position,
				Modifiers:  modifiers,
				ClickCount: state.currentCount,
			})
		}
	case xext.XinputMotionEvent:
		c.handleXinputMotion(ev)
	case xext.XinputEnterEvent:
		if ev.Mode != xext.XinputNotifyModeNormal {
			return
		}
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		window.SetHovered(true)
		c.s.mouseFocusedWindow = gpui.Some(ev.Event)
		c.s.restoreCursorAfterHide()
	case xext.XinputLeaveEvent:
		if ev.Mode != xext.XinputNotifyModeNormal {
			return
		}
		state := c.s
		resetAllPointerDeviceScrollPositions(state.pointerDeviceStates)
		state.mouseFocusedWindow = gpui.None[xproto.Window]()
		var buttons uint32
		if len(ev.Buttons) > 0 {
			buttons = ev.Buttons[0]
		}
		pressedButton := pressedButtonFromMask(buttons)
		position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
		modifiers := modifiersFromXinputInfo(ev.Mods)
		state.Modifiers = modifiers
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		window.HandleInput(gpui.MouseExitEvent{
			PressedButton: pressedButton,
			Position:      position,
			Modifiers:     modifiers,
		})
		window.SetHovered(false)
	case xext.XinputHierarchyEvent:
		state := c.s
		for _, info := range ev.Infos {
			if isPointerDevice(info.Type) {
				delete(state.pointerDeviceStates, info.Deviceid)
			}
		}
		if states, ok := currentPointerDeviceStates(state.conn.X, state.pointerDeviceStates); ok {
			state.pointerDeviceStates = states
		}
	case xext.XinputDeviceChangedEvent:
		if pointer, ok := c.s.pointerDeviceStates[ev.Sourceid]; ok {
			resetPointerDeviceScrollPositions(pointer)
		}
	case xext.XinputGesturePinchBeginEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		state := c.s
		state.pinchScale = 1.0
		modifiers := modifiersFromXinputInfo(ev.Mods)
		state.Modifiers = modifiers
		position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
		window.HandleInput(gpui.PinchEvent{Position: position, Delta: 0, Modifiers: modifiers, Phase: gpui.TouchPhaseStarted})
	case xext.XinputGesturePinchUpdateEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		state := c.s
		modifiers := modifiersFromXinputInfo(ev.Mods)
		state.Modifiers = modifiers
		position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
		newAbsoluteScale := float32(ev.Scale) / 65536.0
		previousScale := state.pinchScale
		zoomDelta := newAbsoluteScale - previousScale
		state.pinchScale = newAbsoluteScale
		window.HandleInput(gpui.PinchEvent{Position: position, Delta: zoomDelta, Modifiers: modifiers, Phase: gpui.TouchPhaseMoved})
	case xext.XinputGesturePinchEndEvent:
		window, ok := c.getWindow(ev.Event)
		if !ok {
			return
		}
		state := c.s
		state.pinchScale = 1.0
		modifiers := modifiersFromXinputInfo(ev.Mods)
		state.Modifiers = modifiers
		position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
		window.HandleInput(gpui.PinchEvent{Position: position, Delta: 0, Modifiers: modifiers, Phase: gpui.TouchPhaseEnded})
	}
}

func clientMessageData32(ev xproto.ClientMessageEvent) [5]uint32 {
	var out [5]uint32
	copy(out[:], ev.Data.Data32)
	return out
}

func (c X11Client) handleClientMessage(ev xproto.ClientMessageEvent) {
	window, ok := c.getWindow(ev.Window)
	if !ok {
		return
	}
	data := clientMessageData32(ev)
	atom, arg1, arg2, arg3, arg4 := data[0], data[1], data[2], data[3], data[4]
	state := c.s

	if atom == uint32(state.atoms.WM_DELETE_WINDOW) {
		if window.ShouldClose() {
			window.Close()
		}
		return
	}

	if atom == uint32(state.atoms._NET_WM_SYNC_REQUEST) {
		window.state.lastSyncCounter = gpui.Some(xext.SyncInt64{Lo: arg2, Hi: int32(arg3)})
	}

	switch ev.Type {
	case state.atoms.XdndEnter:
		state.xdndState.otherWindow = xproto.Window(atom)
		if arg1&0x1 == 0x1 {
			state.xdndState.dragType = xdndGetSupportedAtom(state.conn.X, &state.atoms, state.xdndState.otherWindow)
		} else {
			for _, a := range [...]uint32{arg2, arg3, arg4} {
				if xdndIsAtomSupported(a, &state.atoms) {
					state.xdndState.dragType = a
					break
				}
			}
		}
	case state.atoms.XdndLeave:
		position := state.xdndState.position
		window.HandleInput(gpui.FileDropEventPending{Position: position})
		window.HandleInput(gpui.FileDropEventExited{})
		c.s.xdndState = Xdnd{}
	case state.atoms.XdndPosition:
		if pos, err := checkedReply("Failed to query pointer position", replyOf(xproto.QueryPointer(state.conn.X, ev.Window).Reply())); err == nil {
			state.xdndState.position = gpui.Point[gpui.Pixels]{X: gpui.Px(float32(pos.WinX)), Y: gpui.Px(float32(pos.WinY))}
		}
		if !state.xdndState.retrieved {
			err := checkReply("Failed to convert selection for drag and drop", xproto.ConvertSelectionChecked(state.conn.X, ev.Window, state.atoms.XdndSelection, xproto.Atom(state.xdndState.dragType), state.atoms.XDND_DATA, xproto.Timestamp(arg3)).Check())
			if err != nil {
				slog.Error(err.Error())
			}
		}
		xdndSendStatus(state.conn.X, &state.atoms, ev.Window, state.xdndState.otherWindow, arg4)
		position := state.xdndState.position
		window.HandleInput(gpui.FileDropEventPending{Position: position})
	case state.atoms.XdndDrop:
		xdndSendFinished(state.conn.X, &state.atoms, ev.Window, state.xdndState.otherWindow)
		position := state.xdndState.position
		window.HandleInput(gpui.FileDropEventSubmit{Position: position})
		c.s.xdndState = Xdnd{}
	}
}

type replyResult[T any] struct {
	reply *T
	err   error
}

func replyOf[T any](reply *T, err error) replyResult[T] { return replyResult[T]{reply, err} }

func checkedReply[T any](failureContext string, r replyResult[T]) (*T, error) {
	return getReply(failureContext, r.reply, r.err)
}

func (c X11Client) handleSelectionNotify(ev xproto.SelectionNotifyEvent) {
	window, ok := c.getWindow(ev.Requestor)
	if !ok {
		return
	}
	state := c.s
	reply, err := checkedReply("Failed to get XDND_DATA", replyOf(xproto.GetProperty(state.conn.X, false, ev.Requestor, state.atoms.XDND_DATA, xproto.AtomAny, 0, 1024).Reply()))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	if !utf8.Valid(reply.Value) {
		return
	}
	var paths gpui.ExternalPaths
	for _, line := range rustLines(string(reply.Value)) {
		u, err := url.Parse(line)
		if err != nil || u.Scheme == "" {
			if err == nil {
				err = errors.New("relative URL without a base")
			}
			slog.Error(err.Error())
			continue
		}
		path, ok := urlToFilePath(u)
		if !ok {
			slog.Error(fmt.Sprintf("Failed turn %q into a file path", u.String()))
			continue
		}
		paths = append(paths, path)
	}
	input := gpui.FileDropEventEntered{Position: state.xdndState.position, Paths: paths}
	window.HandleInput(input)
	c.s.xdndState.retrieved = true
}

func rustLines(s string) []string {
	var out []string
	for s != "" {
		line, rest, found := strings.Cut(s, "\n")
		out = append(out, strings.TrimSuffix(line, "\r"))
		if !found {
			break
		}
		s = rest
	}
	return out
}

func urlToFilePath(u *url.URL) (string, bool) {
	if u.Scheme != "file" {
		return "", false
	}
	if u.Host != "" && u.Host != "localhost" {
		return "", false
	}
	if !strings.HasPrefix(u.Path, "/") {
		return "", false
	}
	return u.Path, true
}

func (c X11Client) handleXkbStateNotify(ev xext.XkbStateNotifyEvent) {
	state := c.s
	oldLayout := state.xkb.SerializeLayout(xkb.StateLayoutEffective)
	newLayout := uint32(ev.Group)
	state.xkb.UpdateMask(
		xkb.ModMask(ev.BaseMods),
		xkb.ModMask(ev.LatchedMods),
		xkb.ModMask(ev.LockedMods),
		xkb.LayoutIndex(uint32(int32(ev.BaseGroup))),
		xkb.LayoutIndex(uint32(int32(ev.LatchedGroup))),
		xkb.LayoutIndex(ev.LockedGroup),
	)
	modifiers := common.ModifiersFromXkb(state.xkb)
	capslock := common.CapslockFromXkb(state.xkb)
	if !(state.LastModifiersChangedEvent == modifiers && state.LastCapslockChangedEvent == capslock) {
		focusedWindowID, ok := state.keyboardFocusedWindow.Get()
		if !ok {
			return
		}
		state.Modifiers = modifiers
		state.LastModifiersChangedEvent = modifiers
		state.Capslock = capslock
		state.LastCapslockChangedEvent = capslock
		focusedWindow, ok := c.getWindow(focusedWindowID)
		if !ok {
			return
		}
		focusedWindow.HandleInput(gpui.ModifiersChangedEvent{Modifiers: modifiers, Capslock: capslock})
	}
	if newLayout != oldLayout {
		c.handleKeyboardLayoutChange()
	}
}

func (c X11Client) handleKeyPress(ev xproto.KeyPressEvent) {
	window, ok := c.getWindow(ev.Event)
	if !ok {
		return
	}
	state := c.s
	modifiers := modifiersFromState(ev.State)
	state.Modifiers = modifiers
	state.preKeyCharDown = gpui.None[gpui.Keystroke]()
	keyEventState := xkbStateForKeyEvent(state.xkb, ev.State)
	defer keyEventState.Close()

	code := xkb.Keycode(ev.Detail)
	keystroke := common.KeystrokeFromXkb(keyEventState, modifiers, code)
	keysym := keyEventState.KeyGetOneSym(code)
	if keysym.IsModifierKey() {
		return
	}
	if composeState := state.composeState; composeState != nil {
		state.composeState = nil
		composeState.Feed(keysym)
		switch composeState.Status() {
		case xkb.StatusComposed:
			state.preEditText = gpui.None[string]()
			keystroke.KeyChar = gpui.OptionFrom(composeState.Utf8())
			if ks, ok := composeState.Keysym(); ok {
				keystroke.Key = xkb.KeysymGetName(ks)
			}
		case xkb.StatusComposing:
			keystroke.KeyChar = gpui.None[string]()
			state.preEditText = gpui.OptionFrom(composeState.Utf8()).Or(gpui.OptionFrom(common.KeystrokeUnderlyingDeadKey(keysym)))
			preEdit := state.preEditText.UnwrapOrDefault()
			window.HandleIMEPreedit(preEdit)
		case xkb.StatusCancelled:
			preEdit := state.preEditText
			state.preEditText = gpui.None[string]()
			if text, ok := preEdit.Get(); ok {
				window.HandleIMECommit(text)
			}
			if currentKey, ok := common.KeystrokeUnderlyingDeadKey(keysym); ok {
				window.HandleIMEPreedit(currentKey)
			}
			composeState.Feed(keysym)
		}
		state.composeState = composeState
	}
	window.HandleInput(gpui.KeyDownEvent{Keystroke: keystroke, IsHeld: false, PreferCharacterInput: false})
}

func (c X11Client) handleKeyRelease(ev xproto.KeyReleaseEvent) {
	window, ok := c.getWindow(ev.Event)
	if !ok {
		return
	}
	state := c.s
	modifiers := modifiersFromState(ev.State)
	state.Modifiers = modifiers
	keyEventState := xkbStateForKeyEvent(state.xkb, ev.State)
	defer keyEventState.Close()
	code := xkb.Keycode(ev.Detail)
	keystroke := common.KeystrokeFromXkb(keyEventState, modifiers, code)
	if keyEventState.KeyGetOneSym(code).IsModifierKey() {
		return
	}
	window.HandleInput(gpui.KeyUpEvent{Keystroke: keystroke})
}

func (c X11Client) handleXinputButtonPress(ev xext.XinputButtonPressEvent) {
	window, ok := c.getWindow(ev.Event)
	if !ok {
		return
	}
	state := c.s
	modifiers := modifiersFromXinputInfo(ev.Mods)
	state.Modifiers = modifiers
	position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)

	if state.composing && state.ximc != nil {
		c.ResetIME()
		window.HandleIMEUnmark()
	} else if text, ok := state.preEditText.Get(); ok {
		state.preEditText = gpui.None[string]()
		if state.composeState != nil {
			state.composeState.Reset()
		}
		window.HandleIMECommit(text)
	}
	bos, ok := buttonOrScrollFromEventDetail(ev.Detail)
	switch {
	case ok && bos.kind == buttonOrScrollButton:
		button := bos.button
		clickElapsed := time.Since(state.lastClick)
		if clickElapsed < common.DoubleClickInterval && state.lastMouseButton == gpui.Some(button) && common.IsWithinClickDistance(state.lastLocation, position) {
			state.currentCount++
		} else {
			state.currentCount = 1
		}
		state.lastClick = time.Now()
		state.lastMouseButton = gpui.Some(button)
		state.lastLocation = position
		window.HandleInput(gpui.MouseDownEvent{
			Button:     button,
			Position:   position,
			Modifiers:  modifiers,
			ClickCount: state.currentCount,
			FirstMouse: false,
		})
	case ok:
		if ev.Flags&xext.XinputPointerEventFlagsPointerEmulated == 0 {
			var scrollDelta gpui.Point[float32]
			switch bos.scroll {
			case scrollDirectionUp:
				scrollDelta = gpui.Point[float32]{X: 0, Y: common.ScrollLines}
			case scrollDirectionDown:
				scrollDelta = gpui.Point[float32]{X: 0, Y: -common.ScrollLines}
			case scrollDirectionLeft:
				scrollDelta = gpui.Point[float32]{X: common.ScrollLines, Y: 0}
			case scrollDirectionRight:
				scrollDelta = gpui.Point[float32]{X: -common.ScrollLines, Y: 0}
			}
			window.HandleInput(makeScrollWheelEvent(position, scrollDelta, modifiers))
		}
	default:
		slog.Error(fmt.Sprintf("Unknown x11 button: %d", ev.Detail))
	}
}

func (c X11Client) handleXinputMotion(ev xext.XinputMotionEvent) {
	window, ok := c.getWindow(ev.Event)
	if !ok {
		return
	}
	state := c.s
	state.restoreCursorAfterHide()
	if window.IsBlocked() {
		style := gpui.CursorStyleArrow
		currentStyle, ok := state.cursorStyles[window.xWindow]
		if !ok {
			currentStyle = gpui.CursorStyleArrow
		}
		if currentStyle != style {
			if cursor, ok := state.getCursorIcon(style).Get(); ok {
				state.cursorStyles[window.xWindow] = style
				state.setWindowCursor(window.xWindow, cursor, "Failed to set cursor style")
			}
		}
	}
	var buttonMask uint32
	if len(ev.ButtonMask) > 0 {
		buttonMask = ev.ButtonMask[0]
	}
	pressedButton := pressedButtonFromMask(buttonMask)
	position := eventPosition(ev.EventX, ev.EventY, state.scaleFactor)
	modifiers := modifiersFromXinputInfo(ev.Mods)
	state.Modifiers = modifiers

	if len(ev.ValuatorMask) > 0 && ev.ValuatorMask[0]&3 != 0 {
		window.HandleInput(gpui.MouseMoveEvent{Position: position, PressedButton: pressedButton, Modifiers: modifiers})
	}

	if pointer, ok := state.pointerDeviceStates[ev.Sourceid]; ok {
		if scrollDelta, ok := getScrollDeltaAndUpdateState(pointer, &ev); ok {
			window.HandleInput(makeScrollWheelEvent(position, scrollDelta, modifiers))
		}
	}
}

func (s *X11ClientState) setWindowCursor(window xproto.Window, cursor xproto.Cursor, failureContext string) {
	err := checkReply(failureContext, xproto.ChangeWindowAttributesChecked(s.conn.X, window, xproto.CwCursor, []uint32{uint32(cursor)}).Check())
	if err != nil {
		slog.Error(err.Error())
	}
	xcbFlush(s.conn.X)
}

func (c X11Client) handleXimCallbackEvent(event XimCallbackEvent) {
	switch ev := event.(type) {
	case XimCallbackEventXimXEvent:
		c.handleEvent(ev.Event)
	case XimCallbackEventXimCommitEvent:
		c.ximHandleCommit(ev.Window, ev.Text)
	case XimCallbackEventXimPreeditEvent:
		c.ximHandlePreedit(ev.Window, ev.Text)
	}
}

func (c X11Client) ximHandleEvent(event xgb.Event) {
	var keyEvent xproto.KeyPressEvent
	var responseType uint8
	switch ev := event.(type) {
	case xproto.KeyPressEvent:
		keyEvent, responseType = ev, xproto.KeyPress
	case xproto.KeyReleaseEvent:
		keyEvent, responseType = xproto.KeyPressEvent(ev), xproto.KeyRelease
	default:
		c.handleEvent(event)
		return
	}
	state := c.s
	state.preKeyCharDown = gpui.Some(common.KeystrokeFromXkb(state.xkb, state.Modifiers, xkb.Keycode(keyEvent.Detail)))
	ximc, xh, ok := state.takeXim()
	if !ok {
		return
	}
	xh.Window = keyEvent.Event
	if err := ximc.ForwardEvent(xh.ImID, xh.IcID, 0, ximFromKeyEvent(responseType, keyEvent)); err != nil {
		slog.Error(fmt.Errorf("X11: Failed to forward XIM event: %w", err).Error())
	}
	state.restoreXim(ximc, xh)
}

func (c X11Client) ximHandleCommit(xWindow xproto.Window, text string) {
	window, ok := c.getWindow(xWindow)
	if !ok {
		slog.Error("bug: Failed to get window for XIM commit")
		return
	}
	c.s.composing = false
	window.HandleIMECommit(text)
}

func (c X11Client) ximHandlePreedit(xWindow xproto.Window, text string) {
	window, ok := c.getWindow(xWindow)
	if !ok {
		slog.Error("bug: Failed to get window for XIM preedit")
		return
	}
	state := c.s
	ximc, xh, ok := state.takeXim()
	if !ok {
		return
	}
	state.composing = text != ""
	window.HandleIMEPreedit(text)

	if scaledArea, ok := window.GetIMEArea(); ok {
		icAttributes := ximc.BuildICAttributes().
			Long(xim.AttrInputStyle, uint32(xim.InputStylePreeditCallbacks)).
			Long(xim.AttrClientWindow, uint32(xh.Window)).
			Long(xim.AttrFocusWindow, uint32(xh.Window)).
			Nested(xim.AttrPreeditAttributes, func(b *xim.AttributeBuilder) {
				b.Point(xim.AttrSpotLocation, spotLocation(scaledArea))
			}).
			Build()
		_ = ximc.SetICValues(xh.ImID, xh.IcID, icAttributes)
	}
	state.restoreXim(ximc, xh)
}

func (c X11Client) handleKeyboardLayoutChange() {
	state := c.s
	layoutIdx := state.xkb.SerializeLayout(xkb.StateLayoutEffective)
	keymap := state.xkb.GetKeymap()
	layoutName := keymap.LayoutGetName(layoutIdx)
	keymap.Close()
	if layoutName != state.keyboardLayout.Name() {
		state.keyboardLayout = common.NewLinuxKeyboardLayout(gpui.SharedString(layoutName))
		callback := state.common.Callbacks.KeyboardLayoutChange
		if callback != nil {
			state.common.Callbacks.KeyboardLayoutChange = nil
			callback()
			state.common.Callbacks.KeyboardLayoutChange = callback
		}
	}
}

// CompositorName returns "X11".
func (c X11Client) CompositorName() string { return "X11" }

func (c X11Client) withCommon(f func(*common.LinuxCommon)) { f(c.s.common) }

// KeyboardLayout returns the active keyboard layout.
func (c X11Client) KeyboardLayout() gpui.PlatformKeyboardLayout {
	return common.NewLinuxKeyboardLayout(gpui.SharedString(c.s.keyboardLayout.Name()))
}

// Displays returns one display per X screen.
func (c X11Client) Displays() []gpui.PlatformDisplay {
	setup := xproto.Setup(c.s.conn.X)
	var displays []gpui.PlatformDisplay
	for rootID := range setup.Roots {
		if d, err := NewX11Display(setup, c.s.scaleFactor, rootID); err == nil {
			displays = append(displays, d)
		}
	}
	return displays
}

// PrimaryDisplay returns the default screen, or nil.
func (c X11Client) PrimaryDisplay() gpui.PlatformDisplay {
	d, err := NewX11Display(xproto.Setup(c.s.conn.X), c.s.scaleFactor, c.s.xRootIndex)
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	return d
}

// IsScreenCaptureSupported reports true: X11 capture goes through the registered scap back end.
func (c X11Client) IsScreenCaptureSupported() bool { return true }

// ScreenCaptureSources lists capturable screens through the registered scap back end.
func (c X11Client) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	return scapScreenSources(c.s.common.ForegroundExecutor)
}

// OpenWindow creates an X11 window.
func (c X11Client) OpenWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error) {
	state := c.s
	parentWindow := gpui.None[X11WindowStatePtr]()
	if focused, ok := state.keyboardFocusedWindow.Get(); ok {
		if w, ok := state.windows[focused]; ok {
			parentWindow = gpui.Some(w.window)
		}
	}
	xWindow, err := xproto.NewWindowId(state.conn.X)
	if err != nil {
		return nil, fmt.Errorf("X11: Failed to generate window ID: %w", err)
	}
	compositorGPU := state.compositorGPU
	state.compositorGPU = gpui.None[wgpu.CompositorGPUHint]()
	rgba, ok := state.resourceDatabase.GetString("Xft.rgba", "Xft.Rgba")
	isBGR := ok && strings.EqualFold(rgba, "bgr")
	atoms := state.atoms
	window, err := NewX11Window(
		handle,
		state.this,
		state.common.ForegroundExecutor,
		state.gpuContext,
		compositorGPU,
		params,
		state.conn,
		state.xcb,
		state.clientSideDecorationsSupported,
		state.xRootIndex,
		xWindow,
		&atoms,
		state.scaleFactor,
		state.common.Appearance,
		parentWindow,
		state.supportsXinputGestures,
		isBGR,
	)
	if err != nil {
		return nil, err
	}
	err = checkReply("Failed to set XdndAware property", xproto.ChangePropertyChecked(state.conn.X, xproto.PropModeReplace, xWindow, state.atoms.XdndAware, state.atoms.XA_ATOM, 32, 1, binary.LittleEndian.AppendUint32(nil, 5)).Check())
	if err != nil {
		slog.Error(err.Error())
	}
	xcbFlush(state.conn.X)
	state.windows[xWindow] = &windowRef{
		window:         window.ptr,
		lastVisibility: xproto.VisibilityUnobscured,
	}
	return window, nil
}

// SetCursorStyle sets the cursor of the window under the pointer.
func (c X11Client) SetCursorStyle(style gpui.CursorStyle) {
	state := c.s
	focusedWindow, ok := state.mouseFocusedWindow.Get()
	if !ok {
		return
	}
	currentStyle, ok := state.cursorStyles[focusedWindow]
	if !ok {
		currentStyle = gpui.CursorStyleArrow
	}
	window, hasWindow := state.windows[focusedWindow]
	shouldChange := currentStyle != style && (!hasWindow || !window.window.IsBlocked())
	if !shouldChange {
		return
	}
	state.cursorStyles[focusedWindow] = style
	if state.cursorHiddenWindow == gpui.Some(focusedWindow) {
		return
	}
	cursor, ok := state.getCursorIcon(style).Get()
	if !ok {
		return
	}
	state.setWindowCursor(focusedWindow, cursor, "Failed to set cursor style")
}

// HideCursorUntilMouseMoves hides the cursor over the focused window until the pointer moves.
func (c X11Client) HideCursorUntilMouseMoves() { c.s.hideCursorUntilMouseMoves() }

// IsCursorVisible reports whether the cursor is shown.
func (c X11Client) IsCursorVisible() bool { return c.s.cursorHiddenWindow.IsNone() }

// OpenURI opens uri with the desktop's handler.
func (c X11Client) OpenURI(uri string) {
	common.OpenURIInternal(c.s.common.BackgroundExecutor, c.s.common.SessionBus, uri, gpui.None[string](), c.GraphicalEnvironment())
}

// RevealPath shows path in the file manager.
func (c X11Client) RevealPath(path string) {
	common.RevealPathInternal(c.s.common.BackgroundExecutor, c.s.common.SessionBus, path, gpui.None[string](), c.GraphicalEnvironment())
}

// WriteToPrimary writes item's text to the PRIMARY selection.
func (c X11Client) WriteToPrimary(item gpui.ClipboardItem) {
	text, _ := item.Text()
	if err := c.s.clipboard.SetText(text, ClipboardKindPrimary, WaitConfigNone); err != nil {
		slog.Debug(fmt.Errorf("X11 Failed to write to clipboard (primary): %w", err).Error())
	}
}

// WriteToClipboard writes item's text to CLIPBOARD and caches item.
func (c X11Client) WriteToClipboard(item gpui.ClipboardItem) {
	text, _ := item.Text()
	if err := c.s.clipboard.SetText(text, ClipboardKindClipboard, WaitConfigNone); err != nil {
		slog.Debug(fmt.Errorf("X11: Failed to write to clipboard (clipboard): %w", err).Error())
	}
	c.s.clipboardItem = gpui.Some(item)
}

// ReadFromPrimary reads the PRIMARY selection.
func (c X11Client) ReadFromPrimary() (gpui.ClipboardItem, bool) {
	item, err := c.s.clipboard.GetAny(ClipboardKindPrimary)
	if err != nil {
		slog.Debug(fmt.Errorf("X11: Failed to read from clipboard (primary): %w", err).Error())
		return gpui.ClipboardItem{}, false
	}
	return item, true
}

// ReadFromClipboard reads CLIPBOARD, returning the cached item when this app owns it.
func (c X11Client) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	if c.s.clipboard.IsOwner(ClipboardKindClipboard) {
		return c.s.clipboardItem.Get()
	}
	item, err := c.s.clipboard.GetAny(ClipboardKindClipboard)
	if err != nil {
		slog.Debug(fmt.Errorf("X11: Failed to read from clipboard (clipboard): %w", err).Error())
		return gpui.ClipboardItem{}, false
	}
	return item, true
}

// ActiveWindow returns the keyboard-focused window.
func (c X11Client) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	focused, ok := c.s.keyboardFocusedWindow.Get()
	if !ok {
		return gpui.AnyWindowHandle{}, false
	}
	window, ok := c.s.windows[focused]
	if !ok {
		return gpui.AnyWindowHandle{}, false
	}
	return window.handle(), true
}

// WindowStack returns the client's windows front to back from _NET_CLIENT_LIST_STACKING.
func (c X11Client) WindowStack() ([]gpui.AnyWindowHandle, bool) {
	state := c.s
	root := xproto.Setup(state.conn.X).Roots[state.xRootIndex].Root
	reply, err := xproto.GetProperty(state.conn.X, false, root, state.atoms._NET_CLIENT_LIST_STACKING, xproto.AtomWindow, 0, math.MaxUint32).Reply()
	if err != nil || reply == nil {
		return nil, false
	}
	var windowIDs []xproto.Window
	for i := 0; i+4 <= len(reply.Value); i += 4 {
		windowIDs = append(windowIDs, xproto.Window(binary.NativeEndian.Uint32(reply.Value[i:])))
	}
	handles := []gpui.AnyWindowHandle{}
	for i := len(windowIDs) - 1; i >= 0; i-- {
		if w, ok := state.windows[windowIDs[i]]; ok {
			handles = append(handles, w.handle())
		}
	}
	return handles, true
}

// WindowIdentifier returns the portal identifier ("x11:<hex xid>") of the focused window.
func (c X11Client) WindowIdentifier() *gpui.Task[gpui.Option[string]] {
	if focused, ok := c.s.keyboardFocusedWindow.Get(); ok {
		if w, ok := c.s.windows[focused]; ok {
			return scheduler.ReadyTask(gpui.Some("x11:" + strconv.FormatUint(uint64(w.window.xWindow), 16)))
		}
	}
	return scheduler.ReadyTask(gpui.None[string]())
}

func (s *X11ClientState) hasXim() bool {
	return s.ximc != nil && s.ximHandler != nil
}

func (s *X11ClientState) takeXim() (*xim.Client, *XimHandler, bool) {
	ximc := s.ximc
	if ximc == nil {
		slog.Error("bug: XIM connection not set")
		return nil, nil, false
	}
	s.ximc = nil
	if xh := s.ximHandler; xh != nil {
		s.ximHandler = nil
		return ximc, xh, true
	}
	s.ximc = ximc
	slog.Error("bug: XIM handler not set")
	return nil, nil, false
}

func (s *X11ClientState) restoreXim(ximc *xim.Client, xh *XimHandler) {
	s.ximc = ximc
	s.ximHandler = xh
}

func (s *X11ClientState) updateRefreshLoop(xWindow xproto.Window) {
	windowRef, ok := s.windows[xWindow]
	if !ok {
		return
	}
	isVisible := windowRef.visibility() == gpui.WindowVisibilityVisible
	rs := windowRef.refreshState
	windowRef.refreshState = nil
	switch {
	case !isVisible && (rs == nil || !rs.periodic), isVisible && rs != nil && rs.periodic:
		windowRef.refreshState = rs
	case !isVisible:
		s.loopHandle.Remove(rs.eventLoopToken)
		windowRef.refreshState = &refreshState{refreshRate: rs.refreshRate}
	case rs != nil:
		token := s.startRefreshLoop(xWindow, rs.refreshRate)
		windowRef, ok := s.windows[xWindow]
		if !ok {
			return
		}
		windowRef.refreshState = &refreshState{periodic: true, refreshRate: rs.refreshRate, eventLoopToken: token}
	default:
		screenResources, err := checkedReply("Failed to get screen resources", replyOf(randr.GetScreenResourcesCurrent(s.conn.X, xWindow).Reply()))
		if err != nil {
			slog.Error(err.Error())
			return
		}
		var modeInfo *randr.ModeInfo
		for _, crtc := range screenResources.Crtcs {
			crtcInfo, err := randr.GetCrtcInfo(s.conn.X, crtc, xproto.TimeCurrentTime).Reply()
			if err != nil || crtcInfo == nil {
				continue
			}
			for i := range screenResources.Modes {
				if screenResources.Modes[i].Id == uint32(crtcInfo.Mode) {
					modeInfo = &screenResources.Modes[i]
					break
				}
			}
			if modeInfo != nil {
				break
			}
		}
		var refreshRate time.Duration
		if modeInfo != nil {
			refreshRate = ModeRefreshRate(modeInfo)
		} else {
			slog.Error("Failed to get screen mode info from xrandr, defaulting to 60hz refresh rate.")
			refreshRate = time.Duration(1_000_000/60) * time.Microsecond
		}
		token := s.startRefreshLoop(xWindow, refreshRate)
		windowRef, ok := s.windows[xWindow]
		if !ok {
			return
		}
		windowRef.refreshState = &refreshState{periodic: true, refreshRate: refreshRate, eventLoopToken: token}
	}
}

func (s *X11ClientState) startRefreshLoop(xWindow xproto.Window, refreshRate time.Duration) common.RegistrationToken {
	this := s.this
	return s.loopHandle.InsertTimer(time.Now(), func(instant time.Time) common.TimeoutAction {
		client, ok := this.GetClient()
		if !ok {
			return common.TimeoutActionDrop()
		}
		if window, ok := client.s.windows[xWindow]; ok {
			signalAt := gpui.OptionFrom(gpui.PlatformFrameSignalCapture(func() scheduler.Instant {
				return scheduler.InstantNow().Add(-time.Since(instant))
			}))
			window.window.Refresh(gpui.RequestFrameOptions{
				RequirePresentation: false,
				ForceRender:         false,
				SignalAt:            signalAt,
				SignalSource:        gpui.FrameRequestSourceLocalSchedule,
			})
		}
		if err := client.processX11Events(); err != nil {
			slog.Error(err.Error())
		}
		now := time.Now()
		for instant.Before(now) {
			instant = instant.Add(refreshRate)
		}
		return common.TimeoutActionToInstant(instant)
	})
}

func (s *X11ClientState) getCursorIcon(style gpui.CursorStyle) gpui.Option[xproto.Cursor] {
	if cursor, ok := s.cursorCache[style]; ok {
		return cursor
	}
	var result xproto.Cursor
	var resultErr error
	found := false
	var errs strings.Builder
	cursorIconNames := common.CursorStyleToIconNames(style)
	for _, name := range cursorIconNames {
		loaded, err := s.cursorHandle.LoadCursor(s.conn.X, name)
		if err != nil {
			errs.WriteString(err.Error())
			errs.WriteByte('\n')
			continue
		}
		if loaded != 0 {
			result, found = loaded, true
			break
		}
	}
	if !found {
		if errs.Len() == 0 {
			resultErr = fmt.Errorf("errors while loading cursor icons %q:\n%s", cursorIconNames, errs.String())
		} else {
			resultErr = fmt.Errorf("did not find cursor icons %q", cursorIconNames)
		}
	}
	cursor := gpui.Some(result)
	if !found {
		def, defaultErr := s.cursorHandle.LoadCursor(s.conn.X, common.DefaultCursorIconName)
		if defaultErr == nil {
			common.LogCursorIconWarning(fmt.Sprintf("X11: error loading cursor icon, falling back on default icon '%s': %v", common.DefaultCursorIconName, resultErr))
			cursor = gpui.Some(def)
		} else {
			common.LogCursorIconWarning(fmt.Sprintf("X11: error loading default cursor fallback '%s': %v: %v", common.DefaultCursorIconName, defaultErr, resultErr))
			cursor = gpui.None[xproto.Cursor]()
		}
	}
	s.cursorCache[style] = cursor
	return cursor
}

func (s *X11ClientState) getOrCreateInvisibleCursor() gpui.Option[xproto.Cursor] {
	if cursor, ok := s.invisibleCursorCache.Get(); ok {
		return gpui.Some(cursor)
	}
	cursor, err := createInvisibleCursor(s.conn.X)
	if err != nil {
		slog.Error(fmt.Errorf("X11: error while creating invisible cursor: %w", err).Error())
		return gpui.None[xproto.Cursor]()
	}
	s.invisibleCursorCache = gpui.Some(cursor)
	return gpui.Some(cursor)
}

func (s *X11ClientState) hideCursorUntilMouseMoves() {
	if s.cursorHiddenWindow.IsSome() {
		return
	}
	focusedWindow, ok := s.mouseFocusedWindow.Get()
	if !ok {
		return
	}
	invisibleCursor, ok := s.getOrCreateInvisibleCursor().Get()
	if !ok {
		return
	}
	s.setWindowCursor(focusedWindow, invisibleCursor, "Failed to hide cursor")
	s.cursorHiddenWindow = gpui.Some(focusedWindow)
}

func (s *X11ClientState) restoreCursorAfterHide() {
	hiddenWindow, ok := s.cursorHiddenWindow.Get()
	if !ok {
		return
	}
	s.cursorHiddenWindow = gpui.None[xproto.Window]()
	style, ok := s.cursorStyles[hiddenWindow]
	if !ok {
		style = gpui.CursorStyleArrow
	}
	cursor, ok := s.getCursorIcon(style).Get()
	if !ok {
		slog.Warn(fmt.Sprintf("X11: no cursor icon available to restore %v after hide; cursor may stay invisible", style))
		return
	}
	s.setWindowCursor(hiddenWindow, cursor, "Failed to restore cursor style after hide")
}

// ModeRefreshRate returns the frame interval of a RandR mode, or 16ms for an invalid mode.
func ModeRefreshRate(mode *randr.ModeInfo) time.Duration {
	if mode.DotClock == 0 || mode.Htotal == 0 || mode.Vtotal == 0 {
		return 16 * time.Millisecond
	}
	millihertz := uint64(mode.DotClock) * 1_000 / (uint64(mode.Htotal) * uint64(mode.Vtotal))
	micros := 1_000_000_000 / millihertz
	slog.Info(fmt.Sprintf("Refreshing every %dms", micros/1_000))
	return time.Duration(micros) * time.Microsecond
}

func handleVisibilityChanged(state *X11ClientState, xWindow xproto.Window) {
	state.updateRefreshLoop(xWindow)
	windowRef, ok := state.windows[xWindow]
	if !ok {
		return
	}
	visibility := windowRef.visibility()
	window := windowRef.window
	window.SetVisibility(visibility)
}

func fp3232ToF32(value xext.XinputFp3232) float32 {
	return float32(value.Integral) + float32(value.Frac)/float32(math.MaxUint32)
}

func detectCompositorGPU(conn *xext.Conn, screen *xproto.ScreenInfo) gpui.Option[wgpu.CompositorGPUHint] {
	reply, err := xext.Dri3Open(conn, xproto.Drawable(screen.Root), 0).Reply()
	if err != nil || reply == nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	defer unix.Close(reply.DeviceFd)
	var stat unix.Stat_t
	if err := unix.Stat(fmt.Sprintf("/proc/self/fd/%d", reply.DeviceFd), &stat); err != nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	vendorID, deviceID, ok := common.CompositorGPUHintFromDevT(stat.Rdev)
	if !ok {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	return gpui.Some(wgpu.CompositorGPUHint{VendorID: vendorID, DeviceID: deviceID})
}

func internAtom(c *xgb.Conn, name string) (*xproto.InternAtomReply, error) {
	return checkedReply("Failed to intern "+name, replyOf(xproto.InternAtom(c, false, uint16(len(name)), name).Reply()))
}

func checkCompositorPresent(c *xgb.Conn, root xproto.Window) bool {
	atomName := fmt.Sprintf("_NET_WM_CM_S%d", root)
	method1 := false
	if reply, err := internAtom(c, atomName); err != nil {
		slog.Debug(err.Error())
	} else if reply.Atom != 0 {
		owner, err := checkedReply("Failed to get "+atomName+" owner", replyOf(xproto.GetSelectionOwner(c, reply.Atom).Reply()))
		if err != nil {
			slog.Debug(err.Error())
		} else {
			method1 = owner.Owner != 0
		}
	}

	windowProperty := func(atomName string) (bool, bool) {
		reply, err := internAtom(c, atomName)
		if err != nil {
			slog.Debug(err.Error())
			return false, false
		}
		if reply.Atom == 0 {
			return false, false
		}
		prop, err := checkedReply("Failed to get "+atomName, replyOf(xproto.GetProperty(c, false, root, reply.Atom, xproto.AtomWindow, 0, 1).Reply()))
		return err == nil && prop.ValueLen > 0, true
	}
	method2, ok := windowProperty("_NET_WM_CM_OWNER")
	if !ok {
		return false
	}
	method3, ok := windowProperty("_NET_SUPPORTING_WM_CHECK")
	if !ok {
		return false
	}

	slog.Debug(fmt.Sprintf("Compositor detection: _NET_WM_CM_S?=%v, _NET_WM_CM_OWNER=%v, _NET_SUPPORTING_WM_CHECK=%v", method1, method2, method3))
	return method1 || method2 || method3
}

func checkGtkFrameExtentsSupported(c *xgb.Conn, atoms *XcbAtoms, root xproto.Window) bool {
	supportedAtoms, err := checkedReply("Failed to get _NET_SUPPORTED", replyOf(xproto.GetProperty(c, false, root, atoms._NET_SUPPORTED, xproto.AtomAtom, 0, 1024).Reply()))
	if err != nil {
		slog.Debug(err.Error())
		return false
	}
	for i := 0; i+4 <= len(supportedAtoms.Value); i += 4 {
		if xproto.Atom(binary.NativeEndian.Uint32(supportedAtoms.Value[i:])) == atoms._GTK_FRAME_EXTENTS {
			return true
		}
	}
	return false
}

func xdndIsAtomSupported(atom uint32, atoms *XcbAtoms) bool {
	a := xproto.Atom(atom)
	return a == atoms.TEXT || a == atoms.STRING || a == atoms.UTF8_STRING || a == atoms.TEXT_PLAIN || a == atoms.TEXT_PLAIN_UTF8 || a == atoms.TextUriList
}

func xdndGetSupportedAtom(c *xgb.Conn, supportedAtoms *XcbAtoms, target xproto.Window) uint32 {
	reply, err := checkedReply("Failed to get XDnD supported atoms", replyOf(xproto.GetProperty(c, false, target, supportedAtoms.XdndTypeList, xproto.AtomAny, 0, 1024).Reply()))
	if err != nil {
		slog.Warn(err.Error())
		return 0
	}
	if reply.Format != 32 {
		return 0
	}
	for i := 0; i+4 <= len(reply.Value); i += 4 {
		atom := binary.NativeEndian.Uint32(reply.Value[i:])
		if xdndIsAtomSupported(atom, supportedAtoms) {
			return atom
		}
	}
	return 0
}

func sendClientMessage(c *xgb.Conn, target xproto.Window, typ xproto.Atom, data [5]uint32, failureContext string) {
	message := xproto.ClientMessageEvent{
		Format: 32,
		Window: target,
		Type:   typ,
		Data:   xproto.ClientMessageDataUnionData32New(data[:]),
	}
	err := checkReply(failureContext, xproto.SendEventChecked(c, false, target, xproto.EventMaskNoEvent, string(message.Bytes())).Check())
	if err != nil {
		slog.Error(err.Error())
	}
	xcbFlush(c)
}

func xdndSendFinished(c *xgb.Conn, atoms *XcbAtoms, source, target xproto.Window) {
	sendClientMessage(c, target, atoms.XdndFinished, [5]uint32{uint32(source), 1, uint32(atoms.XdndActionCopy), 0, 0}, "Failed to send XDnD finished event")
}

func xdndSendStatus(c *xgb.Conn, atoms *XcbAtoms, source, target xproto.Window, action uint32) {
	sendClientMessage(c, target, atoms.XdndStatus, [5]uint32{uint32(source), 1, 0, 0, action}, "Failed to send XDnD status event")
}

func currentPointerDeviceStates(c *xgb.Conn, scrollValuesToPreserve map[xext.XinputDeviceID]*pointerDeviceState) (map[xext.XinputDeviceID]*pointerDeviceState, bool) {
	result, err := checkedReply("Failed to query XInput devices", replyOf(xext.XinputXIQueryDevice(c, XinputAllDevices).Reply()))
	if err != nil {
		slog.Error(err.Error())
		return nil, false
	}
	states := map[xext.XinputDeviceID]*pointerDeviceState{}
	for _, info := range result.Infos {
		if !isPointerDevice(info.Type) {
			continue
		}
		var scrollData []xext.XinputDeviceClassDataScroll
		for i := len(info.Classes) - 1; i >= 0; i-- {
			if s, ok := info.Classes[i].AsScroll(); ok {
				scrollData = append(scrollData, s)
			}
		}
		oldState := scrollValuesToPreserve[info.Deviceid]
		var horizontal, vertical *scrollAxisState
		for _, data := range scrollData {
			if data.ScrollType == xext.XinputScrollTypeHorizontal {
				var old *scrollAxisState
				if oldState != nil {
					old = &oldState.horizontal
				}
				s := scrollDataToAxisState(data, old)
				horizontal = &s
				break
			}
		}
		for _, data := range scrollData {
			if data.ScrollType == xext.XinputScrollTypeVertical {
				var old *scrollAxisState
				if oldState != nil {
					old = &oldState.vertical
				}
				s := scrollDataToAxisState(data, old)
				vertical = &s
				break
			}
		}
		if horizontal == nil && vertical == nil {
			continue
		}
		state := &pointerDeviceState{}
		if horizontal != nil {
			state.horizontal = *horizontal
		}
		if vertical != nil {
			state.vertical = *vertical
		}
		states[info.Deviceid] = state
	}
	if len(states) == 0 {
		slog.Error("Found no xinput mouse pointers.")
	}
	return states, true
}

func isPointerDevice(typ xext.XinputDeviceType) bool {
	return typ == xext.XinputDeviceTypeSlavePointer
}

func scrollDataToAxisState(data xext.XinputDeviceClassDataScroll, old *scrollAxisState) scrollAxisState {
	s := scrollAxisState{
		valuatorNumber: gpui.Some(data.Number),
		multiplier:     common.ScrollLines / fp3232ToF32(data.Increment),
	}
	if old != nil {
		s.scrollValue = old.scrollValue
	}
	return s
}

func resetAllPointerDeviceScrollPositions(states map[xext.XinputDeviceID]*pointerDeviceState) {
	for _, s := range states {
		resetPointerDeviceScrollPositions(s)
	}
}

func resetPointerDeviceScrollPositions(pointer *pointerDeviceState) {
	pointer.horizontal.scrollValue = gpui.None[float32]()
	pointer.vertical.scrollValue = gpui.None[float32]()
}

func getScrollDeltaAndUpdateState(pointer *pointerDeviceState, event *xext.XinputMotionEvent) (gpui.Point[float32], bool) {
	deltaX, okX := getAxisScrollDeltaAndUpdateState(event, &pointer.horizontal)
	deltaY, okY := getAxisScrollDeltaAndUpdateState(event, &pointer.vertical)
	if okX || okY {
		return gpui.Point[float32]{X: deltaX, Y: deltaY}, true
	}
	return gpui.Point[float32]{}, false
}

func getAxisScrollDeltaAndUpdateState(event *xext.XinputMotionEvent, axis *scrollAxisState) (float32, bool) {
	valuatorNumber, ok := axis.valuatorNumber.Get()
	if !ok {
		return 0, false
	}
	axisIndex, ok := getValuatorAxisIndex(event.ValuatorMask, valuatorNumber)
	if !ok {
		return 0, false
	}
	if axisIndex >= len(event.Axisvalues) {
		slog.Error("Encountered invalid XInput valuator_mask, scrolling may not work properly.")
		return 0, false
	}
	newScroll := fp3232ToF32(event.Axisvalues[axisIndex])
	old, hadOld := axis.scrollValue.Get()
	axis.scrollValue = gpui.Some(newScroll)
	if !hadOld {
		return 0, false
	}
	return (old - newScroll) * axis.multiplier, true
}

func makeScrollWheelEvent(position gpui.Point[gpui.Pixels], scrollDelta gpui.Point[float32], modifiers gpui.Modifiers) gpui.ScrollEvent {
	delta := scrollDelta
	if modifiers.Shift {
		delta = gpui.Point[float32]{X: scrollDelta.Y, Y: 0}
	}
	return gpui.ScrollEvent{
		Position:   position,
		Delta:      gpui.ScrollDeltaLines(delta),
		Modifiers:  modifiers,
		TouchPhase: gpui.TouchPhase(0),
		Source:     gpui.ScrollSourcePointer,
	}
}

func createInvisibleCursor(c *xgb.Conn) (xproto.Cursor, error) {
	emptyPixmap, err := xproto.NewPixmapId(c)
	if err != nil {
		return 0, err
	}
	root := xproto.Setup(c).Roots[0].Root
	xproto.CreatePixmap(c, 1, emptyPixmap, xproto.Drawable(root), 1, 1)
	cursor, err := xproto.NewCursorId(c)
	if err != nil {
		return 0, err
	}
	xproto.CreateCursor(c, cursor, emptyPixmap, emptyPixmap, 0, 0, 0, 0, 0, 0, 0, 0)
	xproto.FreePixmap(c, emptyPixmap)
	xcbFlush(c)
	return cursor, nil
}

type dpiMode uint8

const (
	dpiModeNotSet dpiMode = iota
	dpiModeRandr
	dpiModeScale
)

func getScaleFactor(c *xgb.Conn, resourceDatabase *xrm.Database, screenIndex int) float32 {
	mode := dpiModeNotSet
	var scale float32
	if v, ok := os.LookupEnv(gpuiX11ScaleFactorEnv); ok {
		if strings.ToLower(v) == "randr" {
			mode = dpiModeRandr
		} else if parsed, err := strconv.ParseFloat(v, 32); err == nil && !strings.ContainsAny(v, "xXpP_") {
			if !validScaleFactor(float32(parsed)) {
				panic(fmt.Sprintf("`%s` must be a positive normal number or `randr`. Got `%s`", gpuiX11ScaleFactorEnv, v))
			}
			mode, scale = dpiModeScale, float32(parsed)
		} else if v == "" {
			mode = dpiModeNotSet
		} else {
			panic(fmt.Sprintf("`%s` must be a positive number or `randr`. Got `%s`", gpuiX11ScaleFactorEnv, v))
		}
	}

	switch mode {
	case dpiModeScale:
		slog.Info(fmt.Sprintf("Using scale factor from %s: %v", gpuiX11ScaleFactorEnv, scale))
		return scale
	case dpiModeRandr:
		if s, ok := getRandrScaleFactor(c, screenIndex); ok {
			slog.Info(fmt.Sprintf("Using RandR scale factor from %s=randr: %v", gpuiX11ScaleFactorEnv, s))
			return s
		}
		slog.Warn("Failed to calculate RandR scale factor, falling back to default")
		return 1.0
	}

	if dpi, ok, err := resourceDatabase.GetFloat32("Xft.dpi", "Xft.dpi"); err == nil && ok {
		s := dpi / 96.0
		slog.Info(fmt.Sprintf("Using scale factor from Xft.dpi: %v", s))
		return s
	}

	if s, ok := getRandrScaleFactor(c, screenIndex); ok {
		slog.Info(fmt.Sprintf("Using RandR scale factor: %v", s))
		return s
	}

	slog.Info("Using default scale factor: 1.0")
	return 1.0
}

func getRandrScaleFactor(c *xgb.Conn, screenIndex int) (float32, bool) {
	roots := xproto.Setup(c).Roots
	if screenIndex < 0 || screenIndex >= len(roots) {
		return 0, false
	}
	root := roots[screenIndex].Root

	versionReply, err := randr.QueryVersion(c, 1, 6).Reply()
	if err != nil || versionReply == nil {
		return 0, false
	}
	if versionReply.MajorVersion < 1 || (versionReply.MajorVersion == 1 && versionReply.MinorVersion < 5) {
		return legacyGetRandrScaleFactor(c, root)
	}

	monitorsReply, err := randr.GetMonitors(c, root, true).Reply()
	if err != nil || monitorsReply == nil {
		return 0, false
	}
	fallback := gpui.None[float32]()
	for _, monitor := range monitorsReply.Monitors {
		if monitor.WidthInMillimeters == 0 || monitor.HeightInMillimeters == 0 {
			continue
		}
		scaleFactor := getDpiFactor(uint32(monitor.Width), uint32(monitor.Height), uint64(monitor.WidthInMillimeters), uint64(monitor.HeightInMillimeters))
		if monitor.Primary {
			return scaleFactor, true
		} else if fallback.IsNone() {
			fallback = gpui.Some(scaleFactor)
		}
	}
	return fallback.Get()
}

func legacyGetRandrScaleFactor(c *xgb.Conn, root xproto.Window) (float32, bool) {
	primaryReply, err := randr.GetOutputPrimary(c, root).Reply()
	if err != nil || primaryReply == nil {
		return 0, false
	}
	primaryOutput := primaryReply.Output

	primaryOutputInfo, err := randr.GetOutputInfo(c, primaryOutput, xproto.TimeCurrentTime).Reply()
	if err != nil || primaryOutputInfo == nil {
		return 0, false
	}

	if primaryOutputInfo.Connection == randr.ConnectionConnected && primaryOutputInfo.MmWidth > 0 && primaryOutputInfo.MmHeight > 0 && primaryOutputInfo.Crtc != 0 {
		crtcInfo, err := randr.GetCrtcInfo(c, primaryOutputInfo.Crtc, xproto.TimeCurrentTime).Reply()
		if err != nil || crtcInfo == nil {
			return 0, false
		}
		if crtcInfo.Width > 0 && crtcInfo.Height > 0 {
			return getDpiFactor(uint32(crtcInfo.Width), uint32(crtcInfo.Height), uint64(primaryOutputInfo.MmWidth), uint64(primaryOutputInfo.MmHeight)), true
		}
	}

	screenResources, err := randr.GetScreenResourcesCurrent(c, root).Reply()
	if err != nil || screenResources == nil {
		return 0, false
	}
	type crtcEntry struct {
		crtc randr.Crtc
		info *randr.GetCrtcInfoReply
	}
	crtcCookies := make([]randr.GetCrtcInfoCookie, len(screenResources.Crtcs))
	for i, crtc := range screenResources.Crtcs {
		crtcCookies[i] = randr.GetCrtcInfo(c, crtc, xproto.TimeCurrentTime)
	}
	var crtcInfos []crtcEntry
	var validOutputs []randr.Output
	seen := map[randr.Output]bool{}
	for i, cookie := range crtcCookies {
		reply, err := cookie.Reply()
		if err != nil || reply == nil || reply.Width == 0 || reply.Height == 0 || len(reply.Outputs) == 0 {
			continue
		}
		crtcInfos = append(crtcInfos, crtcEntry{crtc: screenResources.Crtcs[i], info: reply})
		for _, o := range reply.Outputs {
			if !seen[o] {
				seen[o] = true
				validOutputs = append(validOutputs, o)
			}
		}
	}
	if len(validOutputs) == 0 {
		return 0, false
	}
	outputCookies := make([]randr.GetOutputInfoCookie, len(validOutputs))
	for i, output := range validOutputs {
		outputCookies[i] = randr.GetOutputInfo(c, output, xproto.TimeCurrentTime)
	}
	outputInfos := map[randr.Output]*randr.GetOutputInfoReply{}
	for i, cookie := range outputCookies {
		if reply, err := cookie.Reply(); err == nil && reply != nil {
			outputInfos[validOutputs[i]] = reply
		}
	}

	fallback := gpui.None[float32]()
	for _, entry := range crtcInfos {
		for _, output := range entry.info.Outputs {
			outputInfo, ok := outputInfos[output]
			if !ok || outputInfo.Connection != randr.ConnectionConnected || outputInfo.MmWidth == 0 || outputInfo.MmHeight == 0 {
				continue
			}
			scaleFactor := getDpiFactor(uint32(entry.info.Width), uint32(entry.info.Height), uint64(outputInfo.MmWidth), uint64(outputInfo.MmHeight))
			if output != primaryOutput && fallback.IsNone() {
				fallback = gpui.Some(scaleFactor)
			}
		}
	}
	return fallback.Get()
}

func getDpiFactor(widthPx, heightPx uint32, widthMm, heightMm uint64) float32 {
	ppmm := math.Sqrt((float64(widthPx) * float64(heightPx)) / (float64(widthMm) * float64(heightMm)))

	const mmPerInch = 25.4
	const baseDpi = 96.0
	const quantizeStep = 12.0
	const minScale = 1.0
	const maxScale = 20.0

	dpiFactor := math.Max(math.Round(ppmm*(quantizeStep*mmPerInch/baseDpi))/quantizeStep, minScale)
	validatedFactor := minScale
	if dpiFactor <= maxScale {
		validatedFactor = dpiFactor
	}
	if validScaleFactor(float32(validatedFactor)) {
		return float32(validatedFactor)
	}
	slog.Warn(fmt.Sprintf("Calculated DPI factor %v is invalid, using 1.0", validatedFactor))
	return 1.0
}

func validScaleFactor(scaleFactor float32) bool {
	if math.Signbit(float64(scaleFactor)) || scaleFactor == 0 || math.IsNaN(float64(scaleFactor)) || math.IsInf(float64(scaleFactor), 0) {
		return false
	}
	return math.Abs(float64(scaleFactor)) >= 1.1754943508222875e-38
}

const keyEventModifierMask = xproto.KeyButMaskShift | xproto.KeyButMaskLock | xproto.KeyButMaskControl |
	xproto.KeyButMaskMod1 | xproto.KeyButMaskMod2 | xproto.KeyButMaskMod3 | xproto.KeyButMaskMod4 | xproto.KeyButMaskMod5

func xkbStateForKeyEvent(state *xkb.State, eventState uint16) *xkb.State {
	keymap := state.GetKeymap()
	keyEventState := xkb.NewState(keymap)
	keymap.Close()

	latchedModifiers := state.SerializeMods(xkb.StateModsLatched)
	lockedModifiers := state.SerializeMods(xkb.StateModsLocked)
	activeModifierMask := xkb.ModMask(eventState & keyEventModifierMask)
	depressedModifiers := activeModifierMask &^ (latchedModifiers | lockedModifiers)

	keyEventState.UpdateMask(
		depressedModifiers,
		latchedModifiers,
		lockedModifiers,
		state.SerializeLayout(xkb.StateLayoutDepressed),
		state.SerializeLayout(xkb.StateLayoutLatched),
		state.SerializeLayout(xkb.StateLayoutLocked),
	)
	return keyEventState
}

func initXimClient(c *xgb.Conn, screen *xproto.ScreenInfo) (*xim.Client, error) {
	clientWindow, err := xproto.NewWindowId(c)
	if err != nil {
		return nil, err
	}
	xproto.CreateWindow(c, 0, clientWindow, screen.Root, 0, 0, 1, 1, 0, xproto.WindowClassInputOnly, screen.RootVisual, 0, nil)
	return xim.Init(ximTransport{c: c}, uint32(screen.Root), uint32(clientWindow), "")
}

type ximTransport struct{ c *xgb.Conn }

func (t ximTransport) InternAtom(name string) (uint32, error) {
	reply, err := xproto.InternAtom(t.c, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, errConnectionClosed
	}
	return uint32(reply.Atom), nil
}

func (t ximTransport) AtomName(atom uint32) (string, error) {
	reply, err := xproto.GetAtomName(t.c, xproto.Atom(atom)).Reply()
	if err != nil {
		return "", err
	}
	if reply == nil {
		return "", errConnectionClosed
	}
	return reply.Name, nil
}

func (t ximTransport) GetProperty(window, property, typ uint32, remove bool) (xim.Property, error) {
	reply, err := xproto.GetProperty(t.c, remove, xproto.Window(window), xproto.Atom(property), xproto.Atom(typ), 0, math.MaxUint32).Reply()
	if err != nil {
		return xim.Property{}, err
	}
	if reply == nil {
		return xim.Property{}, errConnectionClosed
	}
	return xim.Property{Type: uint32(reply.Type), Format: reply.Format, Value: reply.Value}, nil
}

func (t ximTransport) AppendProperty(window, property, typ uint32, format uint8, data []byte) error {
	n := uint32(len(data))
	if format > 8 {
		n /= uint32(format / 8)
	}
	xproto.ChangeProperty(t.c, xproto.PropModeAppend, xproto.Window(window), xproto.Atom(property), xproto.Atom(typ), format, n, data)
	return nil
}

func (t ximTransport) SelectionOwner(selection uint32) (uint32, error) {
	reply, err := xproto.GetSelectionOwner(t.c, xproto.Atom(selection)).Reply()
	if err != nil {
		return 0, err
	}
	if reply == nil {
		return 0, errConnectionClosed
	}
	return uint32(reply.Owner), nil
}

func (t ximTransport) ConvertSelection(requestor, selection, target, property uint32) error {
	xproto.ConvertSelection(t.c, xproto.Window(requestor), xproto.Atom(selection), xproto.Atom(target), xproto.Atom(property), xproto.TimeCurrentTime)
	return nil
}

func (t ximTransport) SendClientMessage(window, messageType uint32, format uint8, data [20]byte) error {
	var union xproto.ClientMessageDataUnion
	if format == 32 {
		var words [5]uint32
		for i := range words {
			words[i] = binary.LittleEndian.Uint32(data[i*4:])
		}
		union = xproto.ClientMessageDataUnionData32New(words[:])
	} else {
		union = xproto.ClientMessageDataUnionData8New(data[:])
	}
	message := xproto.ClientMessageEvent{Format: format, Window: xproto.Window(window), Type: xproto.Atom(messageType), Data: union}
	xproto.SendEvent(t.c, false, xproto.Window(window), xproto.EventMaskNoEvent, string(message.Bytes()))
	return nil
}

func (t ximTransport) Flush() error { return nil }

func scapScreenSources(executor gpui.ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	return gpui.ScapScreenSources(executor)
}
