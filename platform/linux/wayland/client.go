//go:build linux

package wayland

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/platform/linux/internal/dbus"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// ObjectID identifies a live protocol object; unlike its protocol id it is never reused.
type ObjectID = *wl.Proxy

const minKeycode uint32 = 8

const unknownKeyboardLayoutName gpui.SharedString = "unknown"

const xdgActivationTokenEnvVar = "XDG_ACTIVATION_TOKEN"

type imeCursorRectangle struct {
	x      int32
	y      int32
	width  int32
	height int32
}

func imeCursorRectangleFrom(bounds gpui.Bounds[gpui.Pixels]) imeCursorRectangle {
	return imeCursorRectangle{
		x:      int32(bounds.Origin.X.AsF32()),
		y:      int32(bounds.Origin.Y.AsF32()),
		width:  int32(bounds.Size.Width.AsF32()),
		height: int32(bounds.Size.Height.AsF32()),
	}
}

type imeCursorRectangleSink interface {
	setIMECursorRectangle(x, y, width, height int32)
	commitIMEState()
}

type textInputSink struct{ textInput *wlproto.ZwpTextInputV3 }

func (s textInputSink) setIMECursorRectangle(x, y, width, height int32) {
	s.textInput.SetCursorRectangle(x, y, width, height)
}

func (s textInputSink) commitIMEState() {
	s.textInput.Commit()
}

func setIMECursorRectangle(textInput imeCursorRectangleSink, cursorRectangle imeCursorRectangle) {
	textInput.setIMECursorRectangle(cursorRectangle.x, cursorRectangle.y, cursorRectangle.width, cursorRectangle.height)
}

func updateIMECursorRectangle(textInput imeCursorRectangleSink, lastIMECursorRectangle *gpui.Option[imeCursorRectangle], bounds gpui.Bounds[gpui.Pixels]) {
	cursorRectangle := imeCursorRectangleFrom(bounds)
	if last, ok := lastIMECursorRectangle.Get(); ok && last == cursorRectangle {
		return
	}
	*lastIMECursorRectangle = gpui.Some(cursorRectangle)
	setIMECursorRectangle(textInput, cursorRectangle)
	textInput.commitIMEState()
}

func setIMECursorRectangleAfterDone(textInput imeCursorRectangleSink, lastIMECursorRectangle *gpui.Option[imeCursorRectangle], bounds gpui.Bounds[gpui.Pixels], shouldCommit bool) {
	if shouldCommit {
		updateIMECursorRectangle(textInput, lastIMECursorRectangle, bounds)
	} else {
		setIMECursorRectangle(textInput, imeCursorRectangleFrom(bounds))
	}
}

const frameRetryInterval = 16_667 * time.Microsecond

// TakeStartupActivationTokenFromEnvironment takes XDG_ACTIVATION_TOKEN out of the environment,
// so that programs we launch don't inherit it.
func TakeStartupActivationTokenFromEnvironment() gpui.Option[string] {
	token, ok := os.LookupEnv(xdgActivationTokenEnvVar)
	os.Unsetenv(xdgActivationTokenEnvVar)
	if !ok || token == "" {
		return gpui.None[string]()
	}
	return gpui.Some(token)
}

// TakeWaylandSocketFromEnvironment takes the connection that WAYLAND_SOCKET hands to this
// process, as libwayland does, and marks it close-on-exec.
func TakeWaylandSocketFromEnvironment() gpui.Option[int] {
	const waylandSocketEnvVar = "WAYLAND_SOCKET"
	value, ok := os.LookupEnv(waylandSocketEnvVar)
	if !ok {
		return gpui.None[int]()
	}
	os.Unsetenv(waylandSocketEnvVar)
	fd, err := strconv.ParseInt(value, 10, 32)
	if err != nil || fd < 0 {
		slog.Error(fmt.Sprintf("ignoring WAYLAND_SOCKET=%q, which is not a file descriptor", value))
		return gpui.None[int]()
	}
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	if err == nil {
		_, err = unix.FcntlInt(uintptr(fd), unix.F_SETFD, flags|unix.FD_CLOEXEC)
	}
	if err != nil {
		slog.Error(fmt.Sprintf("ignoring WAYLAND_SOCKET=%d: %v", fd, err))
		return gpui.None[int]()
	}
	return gpui.Some(int(fd))
}

// Globals are the globals a connection binds. Optional globals are nil when the compositor lacks
// them.
type Globals struct {
	Display                  *wl.Display
	Activation               *wlproto.XdgActivationV1
	Compositor               *wlproto.WlCompositor
	CursorShapeManager       *wlproto.WpCursorShapeManagerV1
	DataDeviceManager        *wlproto.WlDataDeviceManager
	PrimarySelectionManager  *wlproto.ZwpPrimarySelectionDeviceManagerV1
	WmBase                   *wlproto.XdgWmBase
	Shm                      *wlproto.WlShm
	Seat                     *wlproto.WlSeat
	Viewporter               *wlproto.WpViewporter
	FractionalScaleManager   *wlproto.WpFractionalScaleManagerV1
	DecorationManager        *wlproto.ZxdgDecorationManagerV1
	LayerShell               *wlproto.ZwlrLayerShellV1
	KdeBlurManager           *wlproto.OrgKdeKwinBlurManager
	BackgroundEffectsManager *wlproto.ExtBackgroundEffectManagerV1
	TextInputManager         *wlproto.ZwpTextInputManagerV3
	GestureManager           *wlproto.ZwpPointerGesturesV1
	Dialog                   *wlproto.XdgWmDialogV1
	SystemBell               *wlproto.XdgSystemBellV1
	Executor                 gpui.ForegroundExecutor
	FramePing                *common.Ping
}

type globalEntry struct {
	name    uint32
	iface   string
	version uint32
}

type globalList struct {
	registry *wlproto.WlRegistry
	list     []globalEntry
}

var errBindNotPresent = errors.New("the requested global was not found in the registry")

func bindGlobal[T any, PT interface {
	*T
	wl.Object
}](g *globalList, iface *wl.Interface, minVersion, maxVersion uint32) (PT, error) {
	for _, global := range g.list {
		if global.iface != iface.Name {
			continue
		}
		if global.version < minVersion {
			return nil, fmt.Errorf("the requested global was found in the registry but has version %d, below the requested minimum %d", global.version, minVersion)
		}
		return wlproto.Bind[T, PT](g.registry, global.name, iface, min(global.version, maxVersion)), nil
	}
	return nil, errBindNotPresent
}

func optionalGlobal[T any, PT interface {
	*T
	wl.Object
}](g *globalList, iface *wl.Interface, minVersion, maxVersion uint32) PT {
	obj, err := bindGlobal[T, PT](g, iface, minVersion, maxVersion)
	if err != nil {
		return nil
	}
	return obj
}

func newGlobals(globals *globalList, display *wl.Display, executor gpui.ForegroundExecutor, seat *wlproto.WlSeat, framePing *common.Ping) (*Globals, error) {
	dialogV := wlproto.XdgWmDialogV1Interface.Version
	compositor, err := bindGlobal[wlproto.WlCompositor](globals, wlproto.WlCompositorInterface, wlproto.WlSurfaceSetBufferScaleSinceVersion, wlproto.WlSurfacePreferredBufferScaleSinceVersion)
	if err != nil {
		return nil, fmt.Errorf("Wayland compositor does not provide a compatible wl_compositor: %w", err)
	}
	g := &Globals{
		Display:                 display,
		Activation:              optionalGlobal[wlproto.XdgActivationV1](globals, wlproto.XdgActivationV1Interface, 1, 1),
		Compositor:              compositor,
		CursorShapeManager:      optionalGlobal[wlproto.WpCursorShapeManagerV1](globals, wlproto.WpCursorShapeManagerV1Interface, 1, 1),
		DataDeviceManager:       optionalGlobal[wlproto.WlDataDeviceManager](globals, wlproto.WlDataDeviceManagerInterface, wlDataDeviceManagerVersion, wlDataDeviceManagerVersion),
		PrimarySelectionManager: optionalGlobal[wlproto.ZwpPrimarySelectionDeviceManagerV1](globals, wlproto.ZwpPrimarySelectionDeviceManagerV1Interface, 1, 1),
	}
	if g.Shm, err = bindGlobal[wlproto.WlShm](globals, wlproto.WlShmInterface, 1, 1); err != nil {
		return nil, fmt.Errorf("Wayland compositor does not provide wl_shm: %w", err)
	}
	g.Seat = seat
	if g.WmBase, err = bindGlobal[wlproto.XdgWmBase](globals, wlproto.XdgWmBaseInterface, 1, 6); err != nil {
		return nil, fmt.Errorf("Wayland compositor does not provide xdg_wm_base: %w", err)
	}
	wmBase := g.WmBase
	wmBase.SetListener(wlproto.XdgWmBaseListener{Ping: func(serial uint32) { wmBase.Pong(serial) }})
	g.Viewporter = optionalGlobal[wlproto.WpViewporter](globals, wlproto.WpViewporterInterface, 1, 1)
	g.FractionalScaleManager = optionalGlobal[wlproto.WpFractionalScaleManagerV1](globals, wlproto.WpFractionalScaleManagerV1Interface, 1, 1)
	g.DecorationManager = optionalGlobal[wlproto.ZxdgDecorationManagerV1](globals, wlproto.ZxdgDecorationManagerV1Interface, 1, 1)
	g.LayerShell = optionalGlobal[wlproto.ZwlrLayerShellV1](globals, wlproto.ZwlrLayerShellV1Interface, 1, 5)
	g.KdeBlurManager = optionalGlobal[wlproto.OrgKdeKwinBlurManager](globals, wlproto.OrgKdeKwinBlurManagerInterface, 1, 1)
	g.BackgroundEffectsManager = optionalGlobal[wlproto.ExtBackgroundEffectManagerV1](globals, wlproto.ExtBackgroundEffectManagerV1Interface, 1, 1)
	g.TextInputManager = optionalGlobal[wlproto.ZwpTextInputManagerV3](globals, wlproto.ZwpTextInputManagerV3Interface, 1, 1)
	g.GestureManager = optionalGlobal[wlproto.ZwpPointerGesturesV1](globals, wlproto.ZwpPointerGesturesV1Interface, 1, 3)
	g.Dialog = optionalGlobal[wlproto.XdgWmDialogV1](globals, wlproto.XdgWmDialogV1Interface, dialogV, dialogV)
	g.SystemBell = optionalGlobal[wlproto.XdgSystemBellV1](globals, wlproto.XdgSystemBellV1Interface, 1, 1)
	g.Executor = executor
	g.FramePing = framePing
	return g, nil
}

// InProgressOutput accumulates a wl_output's properties until its done event.
type InProgressOutput struct {
	name            gpui.Option[string]
	scale           gpui.Option[int32]
	position        gpui.Option[gpui.Point[gpui.DevicePixels]]
	size            gpui.Option[gpui.Size[gpui.DevicePixels]]
	subpixel        gpui.Option[wlproto.WlOutputSubpixel]
	refreshInterval gpui.Option[time.Duration]
}

func (o *InProgressOutput) complete() (Output, bool) {
	position, okPosition := o.position.Get()
	size, okSize := o.size.Get()
	if !okPosition || !okSize {
		return Output{}, false
	}
	return Output{
		Name:            o.name,
		Scale:           o.scale.UnwrapOr(1),
		Bounds:          gpui.NewBounds(position, size),
		Subpixel:        o.subpixel,
		RefreshInterval: o.refreshInterval,
	}, true
}

// Output is a described wl_output.
type Output struct {
	Name            gpui.Option[string]
	Scale           int32
	Bounds          gpui.Bounds[gpui.DevicePixels]
	Subpixel        gpui.Option[wlproto.WlOutputSubpixel]
	RefreshInterval gpui.Option[time.Duration]
}

// WaylandClientState is the state of a Wayland connection, used only on the main goroutine.
type WaylandClientState struct {
	serialTracker          *SerialTracker
	globals                *Globals
	GPUContext             *wgpu.GPUContext
	CompositorGPU          gpui.Option[wgpu.CompositorGPUHint]
	wlSeat                 *wlproto.WlSeat
	wlPointer              *wlproto.WlPointer
	pinchGesture           *wlproto.ZwpPointerGesturePinchV1
	pinchScale             float32
	wlKeyboard             *wlproto.WlKeyboard
	cursorShapeDevice      *wlproto.WpCursorShapeDeviceV1
	dataDevice             *wlproto.WlDataDevice
	primarySelection       *wlproto.ZwpPrimarySelectionDeviceV1
	textInput              *wlproto.ZwpTextInputV3
	preEditText            gpui.Option[string]
	imePreEdit             gpui.Option[string]
	composing              bool
	lastIMECursorRectangle gpui.Option[imeCursorRectangle]
	windows                map[ObjectID]WaylandWindowStatePtr
	outputs                map[ObjectID]Output
	inProgressOutputs      map[ObjectID]*InProgressOutput
	wlOutputs              map[ObjectID]*wlproto.WlOutput
	outputGlobals          map[uint32]ObjectID
	keyboardLayout         *common.LinuxKeyboardLayout
	keymapState            *xkb.State
	composeState           *xkb.ComposeState
	drag                   dragState[*wlproto.WlDataOffer, WaylandWindowStatePtr]
	externalDrag           *ExternalDrag
	click                  ClickState
	repeat                 KeyRepeat
	Modifiers              gpui.Modifiers
	Capslock               gpui.Capslock
	axisSource             wlproto.WlPointerAxisSource
	MouseLocation          gpui.Option[gpui.Point[gpui.Pixels]]
	continuousScrollDelta  gpui.Option[gpui.Point[gpui.Pixels]]
	discreteScrollDelta    gpui.Option[gpui.Point[float32]]
	verticalModifier       float32
	horizontalModifier     float32
	scrollEventReceived    bool
	enterToken             bool
	buttonPressed          gpui.Option[gpui.MouseButton]
	mouseFocusedWindow     gpui.Option[WaylandWindowStatePtr]
	keyboardFocusedWindow  gpui.Option[WaylandWindowStatePtr]
	loopHandle             *common.LoopHandle
	cursorStyle            gpui.Option[gpui.CursorStyle]
	cursorHiddenWindow     gpui.Option[WaylandWindowStatePtr]
	clipboard              *Clipboard
	dataOffers             []DataOffer[*wlproto.WlDataOffer]
	primaryDataOffer       *DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]
	cursor                 *Cursor
	startupActivationToken gpui.Option[string]
	graphicalEnvironment   gpui.GraphicalEnvironment
	registrations          []common.RegistrationToken
	Common                 *common.LinuxCommon
	imeEnabled             gpui.Option[bool]
	connection             *wl.Display
	xdpSource              *dbus.XDPEventSource
	xdpSender              *common.ChannelSender[dbus.Event]
	windowIdentifiers      map[ObjectID]*waylandWindowIdentifier
}

type fileDragDataOffer interface {
	Finish()
	Destroy()
}

type dragState[O fileDragDataOffer, W any] struct {
	dataOffer         gpui.Option[O]
	window            gpui.Option[W]
	position          gpui.Point[gpui.Pixels]
	uriReadGeneration uint64
}

func (d *dragState[O, W]) beginURIRead() uint64 {
	d.invalidateURIRead()
	return d.uriReadGeneration
}

func (d *dragState[O, W]) invalidateURIRead() {
	d.uriReadGeneration++
}

func (d *dragState[O, W]) isURIReadCurrent(generation uint64) bool {
	return d.uriReadGeneration == generation
}

func (d *dragState[O, W]) handleLeave() (W, gpui.PlatformInput, bool) {
	d.invalidateURIRead()
	var zero W
	window, ok := d.window.Get()
	if !ok {
		return zero, nil, false
	}
	dataOffer, ok := d.dataOffer.Get()
	if !ok {
		return zero, nil, false
	}
	dataOffer.Destroy()
	d.dataOffer = gpui.None[O]()
	d.window = gpui.None[W]()
	return window, gpui.FileDropEventExited{}, true
}

func (d *dragState[O, W]) handleDrop() (W, gpui.PlatformInput, bool) {
	d.invalidateURIRead()
	var zero W
	window, ok := d.window.Get()
	if !ok {
		return zero, nil, false
	}
	dataOffer, ok := d.dataOffer.Get()
	if !ok {
		return zero, nil, false
	}
	dataOffer.Finish()
	dataOffer.Destroy()
	d.dataOffer = gpui.None[O]()
	d.window = gpui.None[W]()
	return window, gpui.FileDropEventSubmit{Position: d.position}, true
}

func (d *dragState[O, W]) completeURIRead(generation uint64, dataOffer O, window W, position gpui.Point[gpui.Pixels], paths gpui.ExternalPaths) (W, gpui.PlatformInput, bool) {
	var zero W
	if !d.isURIReadCurrent(generation) {
		dataOffer.Destroy()
		return zero, nil, false
	}
	d.dataOffer = gpui.Some(dataOffer)
	d.window = gpui.Some(window)
	d.position = position
	return window, gpui.FileDropEventEntered{Position: position, Paths: paths}, true
}

// DataSourceKind is what a wl_data_source was created for.
type DataSourceKind uint8

const (
	DataSourceKindClipboard DataSourceKind = iota
	DataSourceKindDrag
)

// ExternalDrag is an outbound file drag in progress.
type ExternalDrag struct {
	source *wlproto.WlDataSource
	bytes  []byte
	window WaylandWindowStatePtr
}

func fileURIList(paths gpui.FileDragPaths) string {
	var list strings.Builder
	for _, entry := range paths.Entries() {
		uri, ok := fileURLFromPath(entry.Path)
		if !ok {
			continue
		}
		list.WriteString(uri)
		list.WriteString("\r\n")
	}
	return list.String()
}

func fileURLFromPath(path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	var serialization strings.Builder
	serialization.WriteString("file://")
	empty := true
	for component := range strings.SplitSeq(path, "/") {
		if component == "" || component == "." {
			continue
		}
		empty = false
		serialization.WriteByte('/')
		for i := 0; i < len(component); i++ {
			b := component[i]
			if isSpecialPathSegmentByte(b) {
				fmt.Fprintf(&serialization, "%%%02X", b)
			} else {
				serialization.WriteByte(b)
			}
		}
	}
	if empty {
		serialization.WriteByte('/')
	}
	return serialization.String(), true
}

func isSpecialPathSegmentByte(b byte) bool {
	if b < 0x20 || b >= 0x7f {
		return true
	}
	switch b {
	case ' ', '"', '<', '>', '`', '#', '?', '{', '}', '/', '%', '\\':
		return true
	}
	return false
}

// ClickState tracks consecutive clicks for click counts.
type ClickState struct {
	lastMouseButton gpui.Option[gpui.MouseButton]
	lastClick       time.Time
	lastLocation    gpui.Point[gpui.Pixels]
	currentCount    int
}

// KeyRepeat is the keyboard's repeat configuration and the key repeating.
type KeyRepeat struct {
	charactersPerSecond uint32
	delay               time.Duration
	currentID           uint64
	currentKeycode      gpui.Option[xkb.Keycode]
}

// PendingActivation is what an xdg_activation_v1 token was requested for.
type PendingActivation interface{ isPendingActivation() }

// PendingActivationURI is a URI to open in the web browser.
type PendingActivationURI struct{ URI string }

// PendingActivationPath is a path to open in the file explorer.
type PendingActivationPath struct{ Path string }

// PendingActivationWindow is a window from ourselves to raise.
type PendingActivationWindow struct{ Window ObjectID }

func (PendingActivationURI) isPendingActivation()    {}
func (PendingActivationPath) isPendingActivation()   {}
func (PendingActivationWindow) isPendingActivation() {}

type popupGrab struct {
	Serial uint32
	Seat   *wlproto.WlSeat
}

func (s *WaylandClientState) consumeStartupActivationToken(surface *wlproto.WlSurface) {
	token, ok := s.startupActivationToken.Get()
	if !ok {
		return
	}
	s.startupActivationToken = gpui.None[string]()
	if s.globals.Activation == nil {
		return
	}
	s.globals.Activation.Activate(token, surface)
}

type clientCell struct {
	state *WaylandClientState
}

// WaylandClientStatePtr is a weak reference to the client state, handed to windows and
// listeners.
type WaylandClientStatePtr struct {
	cell *clientCell
}

// GetClient returns the client state; it panics once the connection is closed.
func (p WaylandClientStatePtr) GetClient() *WaylandClientState {
	if p.cell == nil || p.cell.state == nil {
		panic("The pointer should always be valid when dispatching in wayland")
	}
	return p.cell.state
}

func (p WaylandClientStatePtr) reportDisplaysChanged() {
	c := p.GetClient().Common
	callback := c.Callbacks.DisplaysChanged
	c.Callbacks.DisplaysChanged = nil
	if callback != nil {
		callback()
		if c.Callbacks.DisplaysChanged == nil {
			c.Callbacks.DisplaysChanged = callback
		}
	}
}

// TryGetClient returns the client state, or false once the connection is closed, for example by
// a switch to headless mode.
func (p WaylandClientStatePtr) TryGetClient() (*WaylandClientState, bool) {
	if p.cell == nil || p.cell.state == nil {
		return nil, false
	}
	return p.cell.state, true
}

// SoleOutput returns the only output, when exactly one is connected.
func (p WaylandClientStatePtr) SoleOutput() (ObjectID, Output, bool) {
	state, ok := p.TryGetClient()
	if !ok || len(state.outputs) != 1 {
		return nil, Output{}, false
	}
	for id, output := range state.outputs {
		return id, output, true
	}
	return nil, Output{}, false
}

// DispatchScheduledFrames ticks every window after the frame ping.
func (p WaylandClientStatePtr) DispatchScheduledFrames() {
	client, ok := p.TryGetClient()
	if !ok {
		return
	}
	windows := make([]WaylandWindowStatePtr, 0, len(client.windows))
	for _, window := range client.windows {
		windows = append(windows, window)
	}
	for _, window := range windows {
		window.ScheduledFrameFired()
	}
}

// ScheduleFrameRetry queues a retry tick for surfaceID one refresh interval from now. An
// immediate retry would spin against the frame-rate throttle that deferred the draw.
func (p WaylandClientStatePtr) ScheduleFrameRetry(surfaceID ObjectID) {
	state := p.GetClient()
	state.loopHandle.InsertTimerAfter(frameRetryInterval, func(deadline time.Time) common.TimeoutAction {
		client, ok := p.TryGetClient()
		if !ok {
			return common.TimeoutActionDrop()
		}
		if window, ok := getWindow(client, surfaceID); ok {
			window.RetryTimerFired(deadline)
		}
		return common.TimeoutActionDrop()
	})
}

// GetSerial returns the latest serial of kind.
func (p WaylandClientStatePtr) GetSerial(kind SerialKind) Serial {
	return p.GetClient().serialTracker.Get(kind)
}

// ActivationContext returns the press an activation token request should reference.
func (p WaylandClientStatePtr) ActivationContext() (ActivationContext, bool) {
	state, ok := p.TryGetClient()
	if !ok {
		return ActivationContext{}, false
	}
	return state.serialTracker.ActivationContext()
}

// StartExternalDrag starts an outbound drag of payload from surface.
func (p WaylandClientStatePtr) StartExternalDrag(surface *wlproto.WlSurface, payload gpui.ExternalDragPayload) bool {
	state := p.GetClient()
	window, ok := getWindow(state, &surface.Proxy)
	if !ok {
		return false
	}
	dataDeviceManager, dataDevice := state.globals.DataDeviceManager, state.dataDevice
	if dataDeviceManager == nil || dataDevice == nil {
		return false
	}
	files, ok := payload.(gpui.ExternalDragPayloadFiles)
	if !ok {
		return false
	}
	uriList := fileURIList(files.FileDragPaths)
	if uriList == "" {
		return false
	}
	serial := state.serialTracker.Get(SerialKindMousePress)
	source := dataDeviceManager.CreateDataSource()
	source.SetListener(p.dataSourceListener(source, DataSourceKindDrag))
	source.Offer(fileListMimeType)
	source.SetActions(wlproto.WlDataDeviceManagerDndActionCopy | wlproto.WlDataDeviceManagerDndActionMove)
	dataDevice.StartDrag(source, surface, nil, serial.AsRaw())
	state.externalDrag = &ExternalDrag{source: source, bytes: []byte(uriList), window: window}
	return true
}

// EnableIME enables the text input for the focused window.
func (p WaylandClientStatePtr) EnableIME() {
	state := p.GetClient()
	state.imeEnabled = gpui.Some(true)
	state.lastIMECursorRectangle = gpui.None[imeCursorRectangle]()
	textInput := state.textInput
	if textInput == nil {
		return
	}
	state.textInput = nil
	textInput.Enable()
	textInput.SetContentType(wlproto.ZwpTextInputV3ContentHintNone, wlproto.ZwpTextInputV3ContentPurposeNormal)
	cursorRectangle := gpui.None[imeCursorRectangle]()
	if window, ok := state.keyboardFocusedWindow.Get(); ok {
		if area, ok := window.GetIMEArea(); ok {
			rect := imeCursorRectangleFrom(area)
			setIMECursorRectangle(textInputSink{textInput}, rect)
			cursorRectangle = gpui.Some(rect)
		}
		state = p.GetClient()
	}
	textInput.Commit()
	state.lastIMECursorRectangle = cursorRectangle
	state.textInput = textInput
}

// DisableIME disables the text input.
func (p WaylandClientStatePtr) DisableIME() {
	state := p.GetClient()
	state.imeEnabled = gpui.Some(false)
	state.composing = false
	if state.textInput != nil {
		state.textInput.Disable()
		state.textInput.Commit()
	}
}

// IMEEnabled reports whether the IME was last enabled or disabled, or None before either.
func (p WaylandClientStatePtr) IMEEnabled() gpui.Option[bool] {
	return p.GetClient().imeEnabled
}

// UpdateIMEPosition moves the IME cursor rectangle to bounds.
func (p WaylandClientStatePtr) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	state := p.GetClient()
	if state.preEditText.IsSome() {
		return
	}
	if state.textInput == nil {
		return
	}
	updateIMECursorRectangle(textInputSink{state.textInput}, &state.lastIMECursorRectangle, bounds)
}

// HandleKeyboardLayoutChange updates the keyboard layout and notifies the app when it changed.
func (p WaylandClientStatePtr) HandleKeyboardLayoutChange() {
	state := p.GetClient()
	var changed bool
	if state.keymapState != nil {
		layoutIdx := state.keymapState.SerializeLayout(xkb.StateLayoutEffective)
		keymap := state.keymapState.GetKeymap()
		layoutName := keymap.LayoutGetName(layoutIdx)
		keymap.Close()
		changed = layoutName != state.keyboardLayout.Name()
		if changed {
			state.keyboardLayout = common.NewLinuxKeyboardLayout(gpui.NewSharedString(layoutName))
		}
	} else {
		changed = string(unknownKeyboardLayoutName) != state.keyboardLayout.Name()
		if changed {
			state.keyboardLayout = common.NewLinuxKeyboardLayout(unknownKeyboardLayoutName)
		}
	}
	if !changed {
		return
	}
	callback := state.Common.Callbacks.KeyboardLayoutChange
	state.Common.Callbacks.KeyboardLayoutChange = nil
	if callback != nil {
		callback()
		state = p.GetClient()
		state.Common.Callbacks.KeyboardLayoutChange = callback
	}
}

// DropWindow forgets the window of surfaceID.
func (p WaylandClientStatePtr) DropWindow(surfaceID ObjectID) {
	state := p.GetClient()
	closedWindow, ok := state.windows[surfaceID]
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	delete(state.windows, surfaceID)
	if identifier, ok := state.windowIdentifiers[surfaceID]; ok {
		identifier.Destroy()
		delete(state.windowIdentifiers, surfaceID)
	}
	if window, ok := state.mouseFocusedWindow.Get(); ok && window.PtrEq(closedWindow) {
		state.mouseFocusedWindow = gpui.None[WaylandWindowStatePtr]()
	}
	if window, ok := state.keyboardFocusedWindow.Get(); ok && window.PtrEq(closedWindow) {
		state.keyboardFocusedWindow = gpui.None[WaylandWindowStatePtr]()
	}
	if window, ok := state.cursorHiddenWindow.Get(); ok && window.PtrEq(closedWindow) {
		state.cursorHiddenWindow = gpui.None[WaylandWindowStatePtr]()
	}
}

func (s *WaylandClientState) hideCursorUntilMouseMoves() {
	if s.cursorHiddenWindow.IsSome() {
		return
	}
	focusedWindow, ok := s.mouseFocusedWindow.Get()
	if !ok {
		return
	}
	if s.wlPointer == nil {
		return
	}
	serial := s.serialTracker.Get(SerialKindMouseEnter)
	s.wlPointer.SetCursor(serial.AsRaw(), nil, 0, 0)
	s.cursorHiddenWindow = gpui.Some(focusedWindow)
}

func (s *WaylandClientState) restoreCursorAfterHide() {
	if s.cursorHiddenWindow.IsNone() {
		return
	}
	s.cursorHiddenWindow = gpui.None[WaylandWindowStatePtr]()
	style, ok := s.cursorStyle.Get()
	if !ok {
		return
	}
	serial := s.serialTracker.Get(SerialKindMouseEnter)
	if s.cursorShapeDevice != nil {
		s.cursorShapeDevice.SetShape(serial.AsRaw(), toShape(style))
		return
	}
	focusedWindow, ok := s.mouseFocusedWindow.Get()
	if !ok {
		slog.Warn(fmt.Sprintf("wayland: no focused surface to restore cursor style %v after hide; cursor may stay invisible", style))
		return
	}
	if s.wlPointer == nil {
		slog.Warn(fmt.Sprintf("wayland: no wl_pointer to restore cursor style %v after hide; cursor may stay invisible", style))
		return
	}
	scale := focusedWindow.PrimaryOutputScale()
	s.cursor.SetIcon(s.wlPointer, serial.AsRaw(), common.CursorStyleToIconNames(style), scale)
}

// WaylandConnection is a connection to the Wayland compositor. Close tears down the connection
// and its event-loop sources.
type WaylandConnection struct {
	cell *clientCell
}

var _ common.LinuxClient = (*WaylandConnection)(nil)

func (c *WaylandConnection) state() *WaylandClientState {
	return WaylandClientStatePtr{c.cell}.GetClient()
}

func (c *WaylandConnection) ptr() WaylandClientStatePtr {
	return WaylandClientStatePtr{c.cell}
}

// Close removes the connection's event sources, releases its seat devices and disconnects. It
// is idempotent.
func (c *WaylandConnection) Close() {
	state := c.cell.state
	if state == nil {
		return
	}
	registrations := state.registrations
	state.registrations = nil
	for _, token := range registrations {
		state.loopHandle.Remove(token)
	}

	clear(state.windows)

	if state.wlPointer != nil {
		state.wlPointer.Release()
	}
	if state.cursorShapeDevice != nil {
		state.cursorShapeDevice.Destroy()
	}
	if state.dataDevice != nil {
		state.dataDevice.Release()
	}
	if state.textInput != nil {
		state.textInput.Destroy()
	}

	if state.xdpSource != nil {
		state.xdpSource.Close()
	}
	if state.xdpSender != nil {
		state.xdpSender.Close()
	}
	if state.keymapState != nil {
		state.keymapState.Close()
	}
	if state.composeState != nil {
		state.composeState.Close()
	}
	for _, identifier := range state.windowIdentifiers {
		identifier.Destroy()
	}
	clear(state.windowIdentifiers)
	state.cursor.Destroy()
	c.cell.state = nil
	state.connection.Disconnect()
}

const wlDataDeviceManagerVersion uint32 = 3

func waylandSocketPath(environment gpui.GraphicalEnvironment) (string, error) {
	display, ok := environment.WaylandDisplay.Get()
	if !ok || display == "" {
		return "", errors.New("WAYLAND_DISPLAY is not set")
	}
	if filepath.IsAbs(display) {
		return display, nil
	}
	runtimeDirectory, ok := environment.XdgRuntimeDir.Get()
	if !ok || runtimeDirectory == "" {
		return "", errors.New("XDG_RUNTIME_DIR is not set")
	}
	return filepath.Join(runtimeDirectory, display), nil
}

func wlSeatVersion(version uint32) (uint32, error) {
	const wlSeatMinVersion uint32 = 5
	const wlSeatMaxVersion uint32 = 9
	if version < wlSeatMinVersion {
		return 0, fmt.Errorf("wl_seat below required version: %d < %d", version, wlSeatMinVersion)
	}
	return min(max(version, wlSeatMinVersion), wlSeatMaxVersion), nil
}

func wlOutputVersion(version uint32) (uint32, error) {
	const wlOutputMinVersion uint32 = 2
	const wlOutputMaxVersion uint32 = 4
	if version < wlOutputMinVersion {
		return 0, fmt.Errorf("wl_output below required version: %d < %d", version, wlOutputMinVersion)
	}
	return min(max(version, wlOutputMinVersion), wlOutputMaxVersion), nil
}

func connectUnixSocket(path string) (int, error) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	if err := unix.Connect(fd, &unix.SockaddrUnix{Name: path}); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

// Attach connects to the Wayland compositor environment names, or through inheritedSocket when
// given, and registers its event sources on handle. Closing the connection removes its sources,
// leaving the loop and common usable without Wayland.
func Attach(handle *common.LoopHandle, linuxCommon *common.LinuxCommon, environment gpui.GraphicalEnvironment, inheritedSocket gpui.Option[int], activationToken gpui.Option[string]) (*WaylandConnection, error) {
	fd, ok := inheritedSocket.Get()
	if !ok {
		socketPath, err := waylandSocketPath(environment)
		if err != nil {
			return nil, err
		}
		fd, err = connectUnixSocket(socketPath)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Wayland compositor at %s: %w", socketPath, err)
		}
	}
	conn, err := wl.ConnectToFD(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Wayland compositor: %w", err)
	}
	attached := false
	defer func() {
		if !attached && conn != nil {
			conn.Disconnect()
		}
	}()

	cell := &clientCell{}
	pointer := WaylandClientStatePtr{cell}

	registry := wlproto.GetDisplay(conn).GetRegistry()
	globals := &globalList{registry: registry}
	registry.SetListener(wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			globals.list = append(globals.list, globalEntry{name: name, iface: iface, version: version})
		},
		GlobalRemove: func(name uint32) {
			for i, g := range globals.list {
				if g.name == name {
					globals.list = append(globals.list[:i], globals.list[i+1:]...)
					break
				}
			}
		},
	})
	if _, err := conn.Roundtrip(); err != nil {
		return nil, fmt.Errorf("failed to initialize Wayland registry: %w", err)
	}
	registry.SetListener(pointer.registryListener(registry))

	var seat *wlproto.WlSeat
	inProgressOutputs := map[ObjectID]*InProgressOutput{}
	wlOutputs := map[ObjectID]*wlproto.WlOutput{}
	outputGlobals := map[uint32]ObjectID{}
	for _, global := range globals.list {
		switch global.iface {
		case "wl_seat":
			version, err := wlSeatVersion(global.version)
			if err != nil {
				return nil, err
			}
			seat = wlproto.Bind[wlproto.WlSeat](registry, global.name, wlproto.WlSeatInterface, version)
			seat.SetListener(pointer.seatListener(seat))
		case "wl_output":
			version, err := wlOutputVersion(global.version)
			if err != nil {
				return nil, err
			}
			output := wlproto.Bind[wlproto.WlOutput](registry, global.name, wlproto.WlOutputInterface, version)
			output.SetListener(pointer.outputListener(output))
			inProgressOutputs[&output.Proxy] = &InProgressOutput{}
			outputGlobals[global.name] = &output.Proxy
			wlOutputs[&output.Proxy] = output
		}
	}

	compositorGPU := detectCompositorGPU()
	gpuContext := &wgpu.GPUContext{}

	framePing, framePingSource, err := common.MakePing()
	if err != nil {
		return nil, fmt.Errorf("failed to create Wayland frame ping: %w", err)
	}
	if seat == nil {
		framePingSource.Close()
		return nil, errors.New("Wayland compositor does not provide wl_seat")
	}
	foregroundExecutor := linuxCommon.ForegroundExecutor
	boundGlobals, err := newGlobals(globals, conn, foregroundExecutor, seat, framePing)
	if err != nil {
		framePingSource.Close()
		return nil, fmt.Errorf("failed to bind required Wayland globals: %w", err)
	}

	var dataDevice *wlproto.WlDataDevice
	if boundGlobals.DataDeviceManager != nil {
		dataDevice = boundGlobals.DataDeviceManager.GetDataDevice(seat)
		dataDevice.SetListener(pointer.dataDeviceListener())
	}
	var primarySelection *wlproto.ZwpPrimarySelectionDeviceV1
	if boundGlobals.PrimarySelectionManager != nil {
		primarySelection = boundGlobals.PrimarySelectionManager.GetDevice(seat)
		primarySelection.SetListener(pointer.primarySelectionDeviceListener())
	}

	cursor := NewCursor(conn, boundGlobals, 24)

	environmentWithoutToken := environment
	environmentWithoutToken.ActivationToken = gpui.None[string]()

	cell.state = &WaylandClientState{
		serialTracker:     NewSerialTracker(),
		globals:           boundGlobals,
		GPUContext:        gpuContext,
		CompositorGPU:     compositorGPU,
		wlSeat:            seat,
		pinchScale:        1.0,
		dataDevice:        dataDevice,
		primarySelection:  primarySelection,
		outputs:           map[ObjectID]Output{},
		inProgressOutputs: inProgressOutputs,
		outputGlobals:     outputGlobals,
		wlOutputs:         wlOutputs,
		windows:           map[ObjectID]WaylandWindowStatePtr{},
		windowIdentifiers: map[ObjectID]*waylandWindowIdentifier{},
		Common:            linuxCommon,
		keyboardLayout:    common.NewLinuxKeyboardLayout(unknownKeyboardLayoutName),
		click: ClickState{
			lastClick: time.Now(),
		},
		repeat: KeyRepeat{
			charactersPerSecond: 16,
			delay:               500 * time.Millisecond,
		},
		axisSource:             wlproto.WlPointerAxisSourceWheel,
		verticalModifier:       -1.0,
		horizontalModifier:     -1.0,
		loopHandle:             handle,
		clipboard:              NewClipboard(conn, handle),
		cursor:                 cursor,
		startupActivationToken: activationToken,
		graphicalEnvironment:   environmentWithoutToken,
		connection:             conn,
	}

	attached = true
	client := &WaylandConnection{cell: cell}

	token, err := common.InsertSource[struct{}](handle, framePingSource, func(struct{}) {
		pointer.DispatchScheduledFrames()
	})
	if err := client.registerSource(token, err, "failed to register Wayland frame source"); err != nil {
		framePingSource.Close()
		client.Close()
		return nil, err
	}

	xdpSender, xdpChannel, err := common.NewChannel[dbus.Event]()
	if err == nil {
		token, err = common.InsertSource[common.ChannelEvent[dbus.Event]](handle, xdpChannel, func(event common.ChannelEvent[dbus.Event]) {
			if event.Closed {
				return
			}
			pointer.handleXDPEvent(event.Msg)
		})
	}
	if err := client.registerSource(token, err, "failed to register desktop portal source"); err != nil {
		client.Close()
		return nil, err
	}
	cell.state.xdpSender = xdpSender
	if bus, err := linuxCommon.SessionBus(); err == nil {
		cell.state.xdpSource = dbus.NewXDPEventSource(bus, func(event dbus.Event) {
			_ = xdpSender.Send(event)
		})
	} else {
		slog.Error(fmt.Sprintf("failed to connect to the session bus for desktop portal settings: %v", err))
	}

	if _, err := conn.Roundtrip(); err != nil {
		slog.Error(fmt.Sprintf("failed to receive initial Wayland state: %v", err))
	}

	token, err = handle.InsertEventSource(newWaylandSource(conn))
	if err := client.registerSource(token, err, "failed to register Wayland connection source"); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func (p WaylandClientStatePtr) handleXDPEvent(event dbus.Event) {
	client, ok := p.TryGetClient()
	if !ok {
		return
	}
	switch event := event.(type) {
	case dbus.EventWindowAppearance:
		client.Common.Appearance = event.Appearance
		for _, window := range client.windows {
			window.SetAppearance(event.Appearance)
		}
	case dbus.EventButtonLayout:
		layout, err := gpui.WindowButtonLayoutParse(event.Layout)
		if err != nil {
			slog.Error(err.Error())
			layout = gpui.WindowButtonLayoutLinuxDefault()
		}
		client.Common.ButtonLayout = layout
		for _, window := range client.windows {
			window.SetButtonLayout()
		}
	case dbus.EventCursorTheme:
		client.cursor.SetTheme(event.Theme)
	case dbus.EventCursorSize:
		client.cursor.SetSize(event.Size)
	case dbus.EventTextScaleFactor:
		c := client.Common
		if c.TextScaleFactor == event.Factor {
			return
		}
		c.TextScaleFactor = event.Factor
		callback := c.Callbacks.TextScaleFactorChange
		c.Callbacks.TextScaleFactorChange = nil
		if callback != nil {
			callback()
			if c.Callbacks.TextScaleFactorChange == nil {
				c.Callbacks.TextScaleFactorChange = callback
			}
		}
	}
}

func (c *WaylandConnection) registerSource(token common.RegistrationToken, err error, description string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", description, err)
	}
	state := c.state()
	state.registrations = append(state.registrations, token)
	return nil
}

// HasWindows reports whether the connection has open windows.
func (c *WaylandConnection) HasWindows() bool {
	return len(c.state().windows) > 0
}

// GraphicalEnvironment returns the environment the connection was made in, without its
// activation token.
func (c *WaylandConnection) GraphicalEnvironment() gpui.GraphicalEnvironment {
	return c.state().graphicalEnvironment
}

// KeyboardLayout returns the current keyboard layout.
func (c *WaylandConnection) KeyboardLayout() gpui.PlatformKeyboardLayout {
	layout := *c.state().keyboardLayout
	return &layout
}

// Displays returns the described outputs.
func (c *WaylandConnection) Displays() []gpui.PlatformDisplay {
	state := c.state()
	displays := make([]gpui.PlatformDisplay, 0, len(state.outputs))
	for id, output := range state.outputs {
		displays = append(displays, newWaylandDisplay(state.wlOutputs[id], output.Name, output.Bounds.ToPixels(float32(output.Scale)), output.RefreshInterval))
	}
	return displays
}

// PrimaryDisplay returns nil: Wayland has no primary output.
func (c *WaylandConnection) PrimaryDisplay() gpui.PlatformDisplay {
	return nil
}

// IsScreenCaptureSupported reports true.
func (c *WaylandConnection) IsScreenCaptureSupported() bool {
	return true
}

// ScreenCaptureSources reports that Wayland screen capture is not implemented.
func (c *WaylandConnection) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	sourcesTx, sourcesRx := scheduler.Oneshot[scheduler.Result[[]gpui.ScreenCaptureSource]]()
	_ = sourcesTx.Send(scheduler.Result[[]gpui.ScreenCaptureSource]{Err: errors.New("Wayland screen capture not yet implemented.")})
	return sourcesRx
}

// OpenWindow opens a window.
func (c *WaylandConnection) OpenWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error) {
	state := c.state()

	var parent gpui.Option[WaylandWindowStatePtr]
	grab := gpui.None[popupGrab]()
	if popup, ok := params.Kind.(gpui.WindowKindAnchoredPopup); ok {
		found := false
		for _, window := range state.windows {
			if window.Handle() == popup.Parent {
				parent = gpui.Some(window)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("popup parent window not found")
		}
		if popup.Grab {
			serial := max(state.serialTracker.Get(SerialKindMousePress).AsRaw(), state.serialTracker.Get(SerialKindKeyPress).AsRaw())
			if serial != 0 {
				grab = gpui.Some(popupGrab{Serial: serial, Seat: state.wlSeat})
			}
		}
	} else {
		parent = state.keyboardFocusedWindow
	}

	var targetOutput *wlproto.WlOutput
	if displayID, ok := params.DisplayID.Get(); ok {
		targetProtocolID := uint64(displayID)
		for id, output := range state.wlOutputs {
			if uint64(id.ID()) == targetProtocolID {
				targetOutput = output
				break
			}
		}
	}

	appearance := state.Common.Appearance
	compositorGPU := state.CompositorGPU
	state.CompositorGPU = gpui.None[wgpu.CompositorGPUHint]()

	window, surfaceID, err := newWaylandWindow(handle, state.globals, state.GPUContext, compositorGPU, c.ptr(), params, appearance, parent, grab, targetOutput)
	if err != nil {
		return nil, err
	}

	if window.ptr.Toplevel() != nil {
		state.consumeStartupActivationToken(window.ptr.Surface())
	}
	state.windows[surfaceID] = window.ptr

	return window, nil
}

// SetCursorStyle sets the pointer's cursor.
func (c *WaylandConnection) SetCursorStyle(style gpui.CursorStyle) {
	state := c.state()

	current, hasCurrent := state.cursorStyle.Get()
	focused, hasFocused := state.mouseFocusedWindow.Get()
	needUpdate := (!hasCurrent || current != style) && (!hasFocused || !focused.IsBlocked())
	if !needUpdate {
		return
	}

	state.cursorStyle = gpui.Some(style)

	if state.cursorHiddenWindow.IsSome() {
		return
	}

	serial := state.serialTracker.Get(SerialKindMouseEnter)
	if state.cursorShapeDevice != nil {
		state.cursorShapeDevice.SetShape(serial.AsRaw(), toShape(style))
	} else if hasFocused {
		if state.wlPointer == nil {
			panic("window is focused by pointer")
		}
		scale := focused.PrimaryOutputScale()
		state.cursor.SetIcon(state.wlPointer, serial.AsRaw(), common.CursorStyleToIconNames(style), scale)
	}
}

// HideCursorUntilMouseMoves hides the cursor until the pointer moves.
func (c *WaylandConnection) HideCursorUntilMouseMoves() {
	c.state().hideCursorUntilMouseMoves()
}

// IsCursorVisible reports whether the cursor is shown.
func (c *WaylandConnection) IsCursorVisible() bool {
	return c.state().cursorHiddenWindow.IsNone()
}

func (c *WaylandConnection) requestActivationToken(pending PendingActivation) bool {
	state := c.state()
	activation := state.globals.Activation
	if activation == nil {
		return false
	}
	context, ok := state.serialTracker.ActivationContext()
	if !ok {
		return false
	}
	token := activation.GetActivationToken()
	token.SetListener(c.ptr().activationTokenListener(token, pending))
	token.SetSerial(context.Serial.AsRaw(), state.wlSeat)
	token.SetSurface(context.RequestingSurface)
	token.Commit()
	return true
}

// OpenURI opens uri, with an activation token when the compositor provides one.
func (c *WaylandConnection) OpenURI(uri string) {
	if c.requestActivationToken(PendingActivationURI{URI: uri}) {
		return
	}
	state := c.state()
	common.OpenURIInternal(state.Common.BackgroundExecutor, state.Common.SessionBus, uri, gpui.None[string](), state.graphicalEnvironment)
}

// RevealPath shows path in the file manager, with an activation token when the compositor
// provides one.
func (c *WaylandConnection) RevealPath(path string) {
	if c.requestActivationToken(PendingActivationPath{Path: path}) {
		return
	}
	state := c.state()
	common.RevealPathInternal(state.Common.BackgroundExecutor, state.Common.SessionBus, path, gpui.None[string](), state.graphicalEnvironment)
}

// WriteToPrimary takes the primary selection with item.
func (c *WaylandConnection) WriteToPrimary(item gpui.ClipboardItem) {
	state := c.state()
	primarySelectionManager, primarySelection := state.globals.PrimarySelectionManager, state.primarySelection
	if primarySelectionManager == nil || primarySelection == nil {
		return
	}
	if state.mouseFocusedWindow.IsNone() && state.keyboardFocusedWindow.IsNone() {
		return
	}
	state.clipboard.SetPrimary(item)
	serial, ok := state.serialTracker.SelectionSerial()
	if !ok {
		slog.Warn("Skipping Wayland primary selection ownership request because no keyboard or pointer press serial has been received")
		return
	}
	dataSource := primarySelectionManager.CreateSource()
	dataSource.SetListener(c.ptr().primarySelectionSourceListener(dataSource))
	for _, mimeType := range textMimeTypes {
		dataSource.Offer(mimeType)
	}
	dataSource.Offer(state.clipboard.SelfMime())
	primarySelection.SetSelection(dataSource, serial.AsRaw())
}

// WriteToClipboard takes the clipboard with item.
func (c *WaylandConnection) WriteToClipboard(item gpui.ClipboardItem) {
	state := c.state()
	dataDeviceManager, dataDevice := state.globals.DataDeviceManager, state.dataDevice
	if dataDeviceManager == nil || dataDevice == nil {
		return
	}
	if state.mouseFocusedWindow.IsNone() && state.keyboardFocusedWindow.IsNone() {
		return
	}
	state.clipboard.Set(item)
	serial, ok := state.serialTracker.SelectionSerial()
	if !ok {
		slog.Warn("Skipping Wayland clipboard ownership request because no keyboard or pointer press serial has been received")
		return
	}
	dataSource := dataDeviceManager.CreateDataSource()
	dataSource.SetListener(c.ptr().dataSourceListener(dataSource, DataSourceKindClipboard))
	for _, mimeType := range textMimeTypes {
		dataSource.Offer(mimeType)
	}
	dataSource.Offer(state.clipboard.SelfMime())
	dataDevice.SetSelection(dataSource, serial.AsRaw())
}

// ReadFromPrimary reads the primary selection.
func (c *WaylandConnection) ReadFromPrimary() (gpui.ClipboardItem, bool) {
	return c.state().clipboard.ReadPrimary()
}

// ReadFromClipboard reads the clipboard.
func (c *WaylandConnection) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	return c.state().clipboard.Read()
}

// ActiveWindow returns the keyboard-focused window.
func (c *WaylandConnection) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	window, ok := c.state().keyboardFocusedWindow.Get()
	if !ok {
		return gpui.AnyWindowHandle{}, false
	}
	return window.Handle(), true
}

// WindowStack reports no stacking order: Wayland does not expose one.
func (c *WaylandConnection) WindowStack() ([]gpui.AnyWindowHandle, bool) {
	return nil, false
}

// CompositorName returns "Wayland".
func (c *WaylandConnection) CompositorName() string {
	return "Wayland"
}

// WindowIdentifier returns the portal parent-window identifier of the active window, exported
// through xdg-foreign; the export lives until the window closes (D-WLFOREIGN).
func (c *WaylandConnection) WindowIdentifier() *gpui.Task[gpui.Option[string]] {
	state := c.state()
	window, ok := state.keyboardFocusedWindow.Get()
	if !ok {
		return scheduler.ReadyTask(gpui.None[string]())
	}
	surface := window.Surface()
	if identifier, ok := state.windowIdentifiers[&surface.Proxy]; ok {
		return scheduler.ReadyTask(gpui.Some(identifier.String()))
	}
	identifier, ok := waylandExportHandle(state.connection, surface)
	if !ok {
		return scheduler.ReadyTask(gpui.None[string]())
	}
	state.windowIdentifiers[&surface.Proxy] = identifier
	return scheduler.ReadyTask(gpui.Some(identifier.String()))
}

func detectCompositorGPU() gpui.Option[wgpu.CompositorGPUHint] {
	connection, err := wl.Connect("")
	if err != nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	defer connection.Disconnect()
	registry := wlproto.GetDisplay(connection).GetRegistry()
	globals := &globalList{registry: registry}
	registry.SetListener(wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			globals.list = append(globals.list, globalEntry{name: name, iface: iface, version: version})
		},
	})
	if _, err := connection.Roundtrip(); err != nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	dmabuf, err := bindGlobal[wlproto.ZwpLinuxDmabufV1](globals, wlproto.ZwpLinuxDmabufV1Interface, 4, 4)
	if err != nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	feedback := dmabuf.GetDefaultFeedback()
	device := gpui.None[uint64]()
	feedback.SetListener(wlproto.ZwpLinuxDmabufFeedbackV1Listener{
		MainDevice: func(bytes []byte) {
			if len(bytes) == 8 {
				device = gpui.Some(binary.NativeEndian.Uint64(bytes))
			}
		},
	})
	if _, err := connection.Roundtrip(); err != nil {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	feedback.Destroy()
	dmabuf.Destroy()
	dev, ok := device.Get()
	if !ok {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	vendorID, deviceID, ok := common.CompositorGPUHintFromDevT(dev)
	if !ok {
		return gpui.None[wgpu.CompositorGPUHint]()
	}
	return gpui.Some(wgpu.CompositorGPUHint{VendorID: vendorID, DeviceID: deviceID})
}

func (p WaylandClientStatePtr) registryListener(registry *wlproto.WlRegistry) wlproto.WlRegistryListener {
	return wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			state := p.GetClient()
			switch iface {
			case "wl_seat":
				version, err := wlSeatVersion(version)
				if err != nil {
					slog.Error("ignoring wl_seat below the required version")
					return
				}
				if state.wlPointer != nil {
					state.wlPointer.Release()
					state.wlPointer = nil
				}
				if state.wlKeyboard != nil {
					state.wlKeyboard.Release()
					state.wlKeyboard = nil
				}
				state.wlSeat.Release()
				seat := wlproto.Bind[wlproto.WlSeat](registry, name, wlproto.WlSeatInterface, version)
				seat.SetListener(p.seatListener(seat))
				state.wlSeat = seat
			case "wl_output":
				version, err := wlOutputVersion(version)
				if err != nil {
					slog.Error("ignoring wl_output below the required version")
					return
				}
				output := wlproto.Bind[wlproto.WlOutput](registry, name, wlproto.WlOutputInterface, version)
				output.SetListener(p.outputListener(output))
				state.inProgressOutputs[&output.Proxy] = &InProgressOutput{}
				state.outputGlobals[name] = &output.Proxy
				state.wlOutputs[&output.Proxy] = output
			}
		},
		GlobalRemove: func(name uint32) {
			state := p.GetClient()
			id, ok := state.outputGlobals[name]
			if !ok {
				return
			}
			delete(state.outputGlobals, name)
			_, wasComplete := state.outputs[id]
			delete(state.outputs, id)
			delete(state.inProgressOutputs, id)
			if output, ok := state.wlOutputs[id]; ok {
				delete(state.wlOutputs, id)
				if output.Version() >= wlproto.WlOutputReleaseSinceVersion {
					output.Release()
				}
			}
			if wasComplete {
				p.reportDisplaysChanged()
			}
		},
	}
}

// FrameCallbackListener is the listener of a surface's frame callback.
func (p WaylandClientStatePtr) FrameCallbackListener(surfaceID ObjectID) wlproto.WlCallbackListener {
	return wlproto.WlCallbackListener{
		Done: func(uint32) {
			window, ok := getWindow(p.GetClient(), surfaceID)
			if !ok {
				return
			}
			window.FrameCallbackFired(gpui.OptionFrom(gpui.PlatformFrameSignalCapture(scheduler.InstantNow)))
		},
	}
}

func getWindow(state *WaylandClientState, surfaceID ObjectID) (WaylandWindowStatePtr, bool) {
	window, ok := state.windows[surfaceID]
	return window, ok
}

// WlSurfaceEvent is a wl_surface event.
type WlSurfaceEvent interface{ isWlSurfaceEvent() }

// WlSurfaceEventEnter is wl_surface.enter.
type WlSurfaceEventEnter struct{ Output *wlproto.WlOutput }

// WlSurfaceEventLeave is wl_surface.leave.
type WlSurfaceEventLeave struct{ Output *wlproto.WlOutput }

// WlSurfaceEventPreferredBufferScale is wl_surface.preferred_buffer_scale.
type WlSurfaceEventPreferredBufferScale struct{ Factor int32 }

// WlSurfaceEventPreferredBufferTransform is wl_surface.preferred_buffer_transform.
type WlSurfaceEventPreferredBufferTransform struct {
	Transform wlproto.WlOutputTransform
}

func (WlSurfaceEventEnter) isWlSurfaceEvent()                    {}
func (WlSurfaceEventLeave) isWlSurfaceEvent()                    {}
func (WlSurfaceEventPreferredBufferScale) isWlSurfaceEvent()     {}
func (WlSurfaceEventPreferredBufferTransform) isWlSurfaceEvent() {}

// SurfaceListener is the listener of a window's wl_surface.
func (p WaylandClientStatePtr) SurfaceListener(surface *wlproto.WlSurface) wlproto.WlSurfaceListener {
	dispatch := func(event WlSurfaceEvent) {
		state := p.GetClient()
		window, ok := getWindow(state, &surface.Proxy)
		if !ok {
			return
		}
		outputs := make(map[ObjectID]Output, len(state.outputs))
		for id, output := range state.outputs {
			outputs[id] = output
		}
		window.HandleSurfaceEvent(event, outputs)
	}
	return wlproto.WlSurfaceListener{
		Enter:                func(output *wlproto.WlOutput) { dispatch(WlSurfaceEventEnter{Output: output}) },
		Leave:                func(output *wlproto.WlOutput) { dispatch(WlSurfaceEventLeave{Output: output}) },
		PreferredBufferScale: func(factor int32) { dispatch(WlSurfaceEventPreferredBufferScale{Factor: factor}) },
		PreferredBufferTransform: func(transform wlproto.WlOutputTransform) {
			dispatch(WlSurfaceEventPreferredBufferTransform{Transform: transform})
		},
	}
}

func (p WaylandClientStatePtr) outputListener(output *wlproto.WlOutput) wlproto.WlOutputListener {
	id := &output.Proxy
	inProgress := func() (*WaylandClientState, *InProgressOutput, bool) {
		state := p.GetClient()
		o, ok := state.inProgressOutputs[id]
		return state, o, ok
	}
	return wlproto.WlOutputListener{
		Name: func(name string) {
			if _, o, ok := inProgress(); ok {
				o.name = gpui.Some(name)
			}
		},
		Scale: func(factor int32) {
			if _, o, ok := inProgress(); ok {
				o.scale = gpui.Some(factor)
			}
		},
		Geometry: func(x, y, _, _ int32, subpixel wlproto.WlOutputSubpixel, _, _ string, _ wlproto.WlOutputTransform) {
			_, o, ok := inProgress()
			if !ok {
				return
			}
			o.position = gpui.Some(gpui.NewPoint(gpui.DevicePixels(x), gpui.DevicePixels(y)))
			if subpixel <= wlproto.WlOutputSubpixelVerticalBgr {
				o.subpixel = gpui.Some(subpixel)
			}
		},
		Mode: func(flags wlproto.WlOutputMode, width, height, refresh int32) {
			_, o, ok := inProgress()
			if !ok {
				return
			}
			if flags.Contains(wlproto.WlOutputModeCurrent) {
				o.size = gpui.Some(gpui.NewSize(gpui.DevicePixels(width), gpui.DevicePixels(height)))
				o.refreshInterval = refreshIntervalFromMillihertz(refresh)
			}
		},
		Done: func() {
			state, o, ok := inProgress()
			if !ok {
				return
			}
			complete, ok := o.complete()
			if !ok {
				return
			}
			state.outputs[id] = complete
			windows := make([]WaylandWindowStatePtr, 0, len(state.windows))
			for _, window := range state.windows {
				windows = append(windows, window)
			}
			for _, window := range windows {
				window.HandleOutputChanged(id, complete)
			}
			p.reportDisplaysChanged()
		},
	}
}

// XdgSurfaceEvent is an xdg_surface event.
type XdgSurfaceEvent interface{ isXdgSurfaceEvent() }

// XdgSurfaceEventConfigure is xdg_surface.configure.
type XdgSurfaceEventConfigure struct{ Serial uint32 }

func (XdgSurfaceEventConfigure) isXdgSurfaceEvent() {}

// XdgSurfaceListener is the listener of a window's xdg_surface.
func (p WaylandClientStatePtr) XdgSurfaceListener(surfaceID ObjectID) wlproto.XdgSurfaceListener {
	return wlproto.XdgSurfaceListener{
		Configure: func(serial uint32) {
			window, ok := getWindow(p.GetClient(), surfaceID)
			if !ok {
				return
			}
			window.HandleXdgSurfaceEvent(XdgSurfaceEventConfigure{Serial: serial})
		},
	}
}

// XdgToplevelEvent is an xdg_toplevel event.
type XdgToplevelEvent interface{ isXdgToplevelEvent() }

// XdgToplevelEventConfigure is xdg_toplevel.configure.
type XdgToplevelEventConfigure struct {
	Width  int32
	Height int32
	States []byte
}

// XdgToplevelEventClose is xdg_toplevel.close.
type XdgToplevelEventClose struct{}

// XdgToplevelEventConfigureBounds is xdg_toplevel.configure_bounds.
type XdgToplevelEventConfigureBounds struct {
	Width  int32
	Height int32
}

// XdgToplevelEventWmCapabilities is xdg_toplevel.wm_capabilities.
type XdgToplevelEventWmCapabilities struct{ Capabilities []byte }

func (XdgToplevelEventConfigure) isXdgToplevelEvent()       {}
func (XdgToplevelEventClose) isXdgToplevelEvent()           {}
func (XdgToplevelEventConfigureBounds) isXdgToplevelEvent() {}
func (XdgToplevelEventWmCapabilities) isXdgToplevelEvent()  {}

func (p WaylandClientStatePtr) dispatchClosable(surfaceID ObjectID, handle func(WaylandWindowStatePtr) bool) {
	window, ok := getWindow(p.GetClient(), surfaceID)
	if !ok {
		return
	}
	if handle(window) {
		window.Close()
	}
}

// ToplevelListener is the listener of a window's xdg_toplevel.
func (p WaylandClientStatePtr) ToplevelListener(surfaceID ObjectID) wlproto.XdgToplevelListener {
	dispatch := func(event XdgToplevelEvent) {
		p.dispatchClosable(surfaceID, func(window WaylandWindowStatePtr) bool { return window.HandleToplevelEvent(event) })
	}
	return wlproto.XdgToplevelListener{
		Configure: func(width, height int32, states []byte) {
			dispatch(XdgToplevelEventConfigure{Width: width, Height: height, States: states})
		},
		Close: func() { dispatch(XdgToplevelEventClose{}) },
		ConfigureBounds: func(width, height int32) {
			dispatch(XdgToplevelEventConfigureBounds{Width: width, Height: height})
		},
		WmCapabilities: func(capabilities []byte) {
			dispatch(XdgToplevelEventWmCapabilities{Capabilities: capabilities})
		},
	}
}

// ZwlrLayerSurfaceV1Event is a zwlr_layer_surface_v1 event.
type ZwlrLayerSurfaceV1Event interface{ isZwlrLayerSurfaceV1Event() }

// ZwlrLayerSurfaceV1EventConfigure is zwlr_layer_surface_v1.configure.
type ZwlrLayerSurfaceV1EventConfigure struct {
	Serial uint32
	Width  uint32
	Height uint32
}

// ZwlrLayerSurfaceV1EventClosed is zwlr_layer_surface_v1.closed.
type ZwlrLayerSurfaceV1EventClosed struct{}

func (ZwlrLayerSurfaceV1EventConfigure) isZwlrLayerSurfaceV1Event() {}
func (ZwlrLayerSurfaceV1EventClosed) isZwlrLayerSurfaceV1Event()    {}

// LayerSurfaceListener is the listener of a window's zwlr_layer_surface_v1.
func (p WaylandClientStatePtr) LayerSurfaceListener(surfaceID ObjectID) wlproto.ZwlrLayerSurfaceV1Listener {
	dispatch := func(event ZwlrLayerSurfaceV1Event) {
		p.dispatchClosable(surfaceID, func(window WaylandWindowStatePtr) bool { return window.HandleLayersurfaceEvent(event) })
	}
	return wlproto.ZwlrLayerSurfaceV1Listener{
		Configure: func(serial, width, height uint32) {
			dispatch(ZwlrLayerSurfaceV1EventConfigure{Serial: serial, Width: width, Height: height})
		},
		Closed: func() { dispatch(ZwlrLayerSurfaceV1EventClosed{}) },
	}
}

// XdgPopupEvent is an xdg_popup event.
type XdgPopupEvent interface{ isXdgPopupEvent() }

// XdgPopupEventConfigure is xdg_popup.configure.
type XdgPopupEventConfigure struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
}

// XdgPopupEventPopupDone is xdg_popup.popup_done.
type XdgPopupEventPopupDone struct{}

// XdgPopupEventRepositioned is xdg_popup.repositioned.
type XdgPopupEventRepositioned struct{ Token uint32 }

func (XdgPopupEventConfigure) isXdgPopupEvent()    {}
func (XdgPopupEventPopupDone) isXdgPopupEvent()    {}
func (XdgPopupEventRepositioned) isXdgPopupEvent() {}

// PopupListener is the listener of a window's xdg_popup.
func (p WaylandClientStatePtr) PopupListener(surfaceID ObjectID) wlproto.XdgPopupListener {
	dispatch := func(event XdgPopupEvent) {
		p.dispatchClosable(surfaceID, func(window WaylandWindowStatePtr) bool { return window.HandlePopupEvent(event) })
	}
	return wlproto.XdgPopupListener{
		Configure: func(x, y, width, height int32) {
			dispatch(XdgPopupEventConfigure{X: x, Y: y, Width: width, Height: height})
		},
		PopupDone:    func() { dispatch(XdgPopupEventPopupDone{}) },
		Repositioned: func(token uint32) { dispatch(XdgPopupEventRepositioned{Token: token}) },
	}
}

func (p WaylandClientStatePtr) activationTokenListener(token *wlproto.XdgActivationTokenV1, pending PendingActivation) wlproto.XdgActivationTokenV1Listener {
	return wlproto.XdgActivationTokenV1Listener{
		Done: func(activationToken string) {
			state := p.GetClient()
			executor := state.Common.BackgroundExecutor
			environment := state.graphicalEnvironment
			switch pending := pending.(type) {
			case PendingActivationURI:
				common.OpenURIInternal(executor, state.Common.SessionBus, pending.URI, gpui.Some(activationToken), environment)
			case PendingActivationPath:
				common.RevealPathInternal(executor, state.Common.SessionBus, pending.Path, gpui.Some(activationToken), environment)
			case PendingActivationWindow:
				activation := state.globals.Activation
				if window, ok := getWindow(state, pending.Window); ok && activation != nil {
					activation.Activate(activationToken, window.Surface())
				}
			}
			token.Destroy()
		},
	}
}

type waylandSource struct {
	display     *wl.Display
	readPending bool
	storedError error
}

func newWaylandSource(display *wl.Display) *waylandSource {
	return &waylandSource{display: display}
}

func (s *waylandSource) Fd() int { return s.display.FD() }

func (s *waylandSource) Interest() common.Interest { return common.InterestRead }

func (s *waylandSource) flush() error {
	if err := s.display.Flush(); err != nil && !errors.Is(err, syscall.EAGAIN) {
		slog.Error(fmt.Sprintf("Error trying to flush the wayland display: %v", err))
		return err
	}
	return nil
}

func (s *waylandSource) BeforeSleep() (common.Readiness, bool, error) {
	if err := s.flush(); err != nil {
		return common.Readiness{}, false, err
	}
	if s.display.PrepareRead() {
		s.readPending = true
		return common.Readiness{}, false, nil
	}
	return common.Readiness{}, true, nil
}

func (s *waylandSource) BeforeHandleEvents(_ common.Readiness, ready bool) {
	guard := s.readPending
	s.readPending = false
	if !guard {
		return
	}
	if !ready {
		s.display.CancelRead()
		return
	}
	if err := s.display.ReadEvents(); err != nil && !errors.Is(err, syscall.EAGAIN) {
		s.storedError = err
	}
}

func (s *waylandSource) ProcessEvents(common.Readiness) (common.PostAction, error) {
	if err := s.storedError; err != nil {
		s.storedError = nil
		return common.PostActionContinue, err
	}
	for {
		n, err := s.display.DispatchPending()
		if err != nil {
			var protocolError *wl.ProtocolError
			if errors.As(err, &protocolError) {
				slog.Error(fmt.Sprintf("Protocol error received on display: %v", err))
				return common.PostActionContinue, syscall.EPROTO
			}
			if errors.Is(err, wl.ErrProtocolViolation) {
				slog.Error(fmt.Sprintf("Bad message on display: %v", err))
				return common.PostActionContinue, syscall.EPROTO
			}
			return common.PostActionContinue, err
		}
		if n == 0 {
			break
		}
	}
	if err := s.flush(); err != nil {
		return common.PostActionContinue, err
	}
	return common.PostActionContinue, nil
}
