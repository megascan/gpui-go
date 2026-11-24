//go:build linux

package x11

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/linux/xcbshim"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xext"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// XcbAtoms are the atoms the X11 back end interns at startup.
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

func queryRenderExtent(c *xgb.Conn, xWindow xproto.Window) (gpui.Size[gpui.DevicePixels], error) {
	reply, err := xproto.GetGeometry(c, xproto.Drawable(xWindow)).Reply()
	r, err := getReply("X11 GetGeometry failed.", reply, err)
	if err != nil {
		return gpui.Size[gpui.DevicePixels]{}, err
	}
	return gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(r.Width),
		Height: gpui.DevicePixels(r.Height),
	}, nil
}

func resizeEdgeToMoveresize(edge gpui.ResizeEdge) uint32 {
	switch edge {
	case gpui.ResizeEdgeTopLeft:
		return 0
	case gpui.ResizeEdgeTop:
		return 1
	case gpui.ResizeEdgeTopRight:
		return 2
	case gpui.ResizeEdgeRight:
		return 3
	case gpui.ResizeEdgeBottomRight:
		return 4
	case gpui.ResizeEdgeBottom:
		return 5
	case gpui.ResizeEdgeBottomLeft:
		return 6
	default:
		return 7
	}
}

type edgeConstraints struct {
	topTiled        bool
	topResizable    bool
	rightTiled      bool
	rightResizable  bool
	bottomTiled     bool
	bottomResizable bool
	leftTiled       bool
	leftResizable   bool
}

func edgeConstraintsFromAtom(atom uint32) edgeConstraints {
	return edgeConstraints{
		topTiled:        atom&(1<<0) != 0,
		topResizable:    atom&(1<<1) != 0,
		rightTiled:      atom&(1<<2) != 0,
		rightResizable:  atom&(1<<3) != 0,
		bottomTiled:     atom&(1<<4) != 0,
		bottomResizable: atom&(1<<5) != 0,
		leftTiled:       atom&(1<<6) != 0,
		leftResizable:   atom&(1<<7) != 0,
	}
}

func (e *edgeConstraints) toTiling() gpui.Tiling {
	return gpui.Tiling{Top: e.topTiled, Right: e.rightTiled, Bottom: e.bottomTiled, Left: e.leftTiled}
}

type visual struct {
	id       xproto.Visualid
	colormap uint32
	depth    uint8
}

type visualSet struct {
	inherit     visual
	opaque      gpui.Option[visual]
	transparent gpui.Option[visual]
	root        uint32
	blackPixel  uint32
}

func findVisuals(setup *xproto.SetupInfo, screenIndex int) visualSet {
	screen := &setup.Roots[screenIndex]
	set := visualSet{
		inherit: visual{
			id:       screen.RootVisual,
			colormap: uint32(screen.DefaultColormap),
			depth:    screen.RootDepth,
		},
		root:       uint32(screen.Root),
		blackPixel: screen.BlackPixel,
	}
	for _, depthInfo := range screen.AllowedDepths {
		for _, visualType := range depthInfo.Visuals {
			v := visual{id: visualType.VisualId, depth: depthInfo.Depth}
			slog.Debug(fmt.Sprintf("Visual id: %d, class: %d, depth: %d, bits_per_value: %d, masks: 0x%x 0x%x 0x%x",
				visualType.VisualId, visualType.Class, depthInfo.Depth, visualType.BitsPerRgbValue,
				visualType.RedMask, visualType.GreenMask, visualType.BlueMask))
			if visualType.RedMask != 0xFF0000 || visualType.GreenMask != 0xFF00 || visualType.BlueMask != 0xFF {
				continue
			}
			colorMask := visualType.RedMask | visualType.GreenMask | visualType.BlueMask
			alphaMask := uint64(colorMask) ^ ((uint64(1) << depthInfo.Depth) - 1)
			if alphaMask == 0 {
				if set.opaque.IsNone() {
					set.opaque = gpui.Some(v)
				}
			} else if set.transparent.IsNone() {
				set.transparent = gpui.Some(v)
			}
		}
	}
	return set
}

type rawWindow struct {
	connection uintptr
	screenID   int
	windowID   uint32
	visualID   uint32
}

func (w rawWindow) handle() gpui.RawWindowHandle {
	if w.windowID == 0 {
		slog.Error("RawWindow.window_id zero when getting window handle.")
	}
	if w.connection == 0 {
		slog.Error("Null RawWindow.connection when getting display handle.")
	}
	return gpui.RawWindowHandle{
		Kind:          gpui.RawWindowHandleXcb,
		XcbConnection: w.connection,
		XcbWindow:     w.windowID,
		XcbVisual:     w.visualID,
	}
}

// Callbacks are the callbacks a window reports platform events to.
type Callbacks struct {
	requestFrame        func(options gpui.RequestFrameOptions)
	input               func(input gpui.PlatformInput) gpui.DispatchEventResult
	activeStatusChange  func(active bool)
	visibilityChange    func(visibility gpui.WindowVisibility)
	hoveredStatusChange func(hovered bool)
	resize              func(size gpui.Size[gpui.Pixels], scaleFactor float32)
	moved               func()
	shouldClose         func() bool
	close               func()
	appearanceChanged   func()
	buttonLayoutChanged func()
}

// X11WindowState is the state of one X11 window.
type X11WindowState struct {
	parent                         gpui.Option[X11WindowStatePtr]
	children                       map[xproto.Window]struct{}
	client                         X11ClientStatePtr
	executor                       gpui.ForegroundExecutor
	atoms                          XcbAtoms
	xRootWindow                    xproto.Window
	xScreenIndex                   int
	visualID                       uint32
	counterID                      xext.SyncCounter
	lastSyncCounter                gpui.Option[xext.SyncInt64]
	bounds                         gpui.Bounds[gpui.Pixels]
	scaleFactor                    float32
	renderer                       *wgpu.WgpuRenderer
	display                        gpui.PlatformDisplay
	inputHandler                   *gpui.PlatformInputHandler
	appearance                     gpui.WindowAppearance
	backgroundAppearance           gpui.WindowBackgroundAppearance
	maximizedVertical              bool
	maximizedHorizontal            bool
	hidden                         bool
	active                         bool
	visibility                     gpui.WindowVisibility
	hovered                        bool
	forceRenderAfterRecovery       bool
	fullscreen                     bool
	clientSideDecorationsSupported bool
	decorations                    gpui.WindowDecorations
	edgeConstraints                gpui.Option[edgeConstraints]
	handle                         gpui.AnyWindowHandle
	lastInsets                     [4]uint32
	accesskitAdapter               a11yAdapter
}

func (s *X11WindowState) isTransparent() bool {
	return s.backgroundAppearance != gpui.WindowBackgroundAppearanceOpaque
}

func (s *X11WindowState) updateTransparency() {
	isTransparent := s.isTransparent()
	if s.renderer != nil {
		s.renderer.UpdateTransparency(isTransparent)
	}
}

func (s *X11WindowState) isMaximized() bool {
	return !s.hidden && s.maximizedVertical && s.maximizedHorizontal
}

// X11WindowStatePtr is a shared reference to a window's state.
type X11WindowStatePtr struct {
	state     *X11WindowState
	callbacks *Callbacks
	conn      *xext.Conn
	xcb       *xcbshim.Connection
	xWindow   xproto.Window
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

func logErr(err error) bool {
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

const (
	hintInput        uint32 = 1
	hintState        uint32 = 1 << 1
	hintIconPixmap   uint32 = 1 << 2
	hintIconWindow   uint32 = 1 << 3
	hintIconPosition uint32 = 1 << 4
	hintIconMask     uint32 = 1 << 5
	hintWindowGroup  uint32 = 1 << 6
	hintUrgency      uint32 = 1 << 8

	numWmHintsElements            = 9
	numWmSizeHintsElements        = 18
	pMinSize               uint32 = 1 << 4
	pMaxSize               uint32 = 1 << 5
)

type wmHints struct {
	flags  uint32
	values [numWmHintsElements - 1]uint32
	urgent bool
}

func parseWmHints(reply *xproto.GetPropertyReply) (wmHints, bool, error) {
	if reply.Type == xproto.AtomNone {
		return wmHints{}, false, nil
	}
	if reply.Type != xproto.AtomWmHints || reply.Format != 32 {
		return wmHints{}, false, errors.New("invalid value")
	}
	words := len(reply.Value) / 4
	if words < numWmHintsElements-1 {
		return wmHints{}, false, errors.New("insufficient data")
	}
	var h wmHints
	h.flags = binary.LittleEndian.Uint32(reply.Value)
	for i := 1; i < numWmHintsElements && i < words; i++ {
		h.values[i-1] = binary.LittleEndian.Uint32(reply.Value[i*4:])
	}
	if words < numWmHintsElements {
		h.flags &^= hintWindowGroup
	}
	if h.flags&hintState != 0 && h.values[1] != 1 && h.values[1] != 3 {
		return wmHints{}, false, errors.New("invalid value")
	}
	h.urgent = h.flags&hintUrgency != 0
	return h, true, nil
}

func (h wmHints) serialize() []byte {
	known := hintInput | hintState | hintIconPixmap | hintIconWindow | hintIconPosition | hintIconMask | hintWindowGroup
	flags := h.flags & known
	if h.urgent {
		flags |= hintUrgency
	}
	values := h.values
	if flags&hintInput != 0 && values[0] != 0 {
		values[0] = 1
	}
	out := binary.LittleEndian.AppendUint32(nil, flags)
	for i, v := range values {
		bit := []uint32{hintInput, hintState, hintIconPixmap, hintIconWindow, hintIconPosition, hintIconPosition, hintIconMask, hintWindowGroup}[i]
		if flags&bit == 0 {
			v = 0
		}
		out = binary.LittleEndian.AppendUint32(out, v)
	}
	return out
}

func setWmHintsUrgency(c *xgb.Conn, xWindow xproto.Window, urgent bool) {
	var hints wmHints
	reply, err := xproto.GetProperty(c, false, xWindow, xproto.AtomWmHints, xproto.AtomWmHints, 0, numWmHintsElements).Reply()
	if err != nil {
		slog.Debug(fmt.Sprintf("failed to read X11 WM_HINTS before setting urgency: %v", err))
	} else if reply != nil {
		existing, ok, err := parseWmHints(reply)
		if err != nil {
			slog.Debug(fmt.Sprintf("failed to read X11 WM_HINTS before setting urgency: %v", err))
		} else if ok {
			hints = existing
		}
	}
	if !urgent && !hints.urgent {
		return
	}
	hints.urgent = urgent
	data := hints.serialize()
	logErr(checkReply("X11 ChangeProperty for WM_HINTS urgency failed.",
		xproto.ChangePropertyChecked(c, xproto.PropModeReplace, xWindow, xproto.AtomWmHints, xproto.AtomWmHints, 32, numWmHintsElements, data).Check()))
	xcbFlush(c)
}

func wmSizeHintsNormal(minSize gpui.Option[[2]int32], maxSize gpui.Option[[2]int32]) []byte {
	var words [numWmSizeHintsElements]uint32
	if v, ok := minSize.Get(); ok {
		words[0] |= pMinSize
		words[5], words[6] = uint32(v[0]), uint32(v[1])
	}
	if v, ok := maxSize.Get(); ok {
		words[0] |= pMaxSize
		words[7], words[8] = uint32(v[0]), uint32(v[1])
	}
	out := make([]byte, 0, numWmSizeHintsElements*4)
	for _, w := range words {
		out = binary.LittleEndian.AppendUint32(out, w)
	}
	return out
}

func u32Bytes(values ...uint32) []byte {
	out := make([]byte, 0, len(values)*4)
	for _, v := range values {
		out = binary.LittleEndian.AppendUint32(out, v)
	}
	return out
}

func changeProperty32(c *xgb.Conn, window xproto.Window, property, typ xproto.Atom, data ...uint32) error {
	return xproto.ChangePropertyChecked(c, xproto.PropModeReplace, window, property, typ, 32, uint32(len(data)), u32Bytes(data...)).Check()
}

func changeProperty8(c *xgb.Conn, window xproto.Window, property, typ xproto.Atom, data []byte) error {
	return xproto.ChangePropertyChecked(c, xproto.PropModeReplace, window, property, typ, 8, uint32(len(data)), data).Check()
}

func clientMessage(window xproto.Window, typ xproto.Atom, data [5]uint32) string {
	message := xproto.ClientMessageEvent{
		Format: 32,
		Window: window,
		Type:   typ,
		Data:   xproto.ClientMessageDataUnionData32New(data[:]),
	}
	return string(message.Bytes())
}

const substructureMask = xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify

func iconPropertyData(img *image.NRGBA) []uint32 {
	b := img.Bounds()
	width, height := uint32(b.Dx()), uint32(b.Dy())
	data := make([]uint32, 0, 2+int(width*height))
	data = append(data, width, height)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := img.PixOffset(x, y)
			r, g, bl, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
			data = append(data, binary.LittleEndian.Uint32([]byte{bl, g, r, a}))
		}
	}
	return data
}

// NewX11WindowState creates the X window, its properties and its renderer.
func NewX11WindowState(handle gpui.AnyWindowHandle, client X11ClientStatePtr, executor gpui.ForegroundExecutor, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], params gpui.WindowParams, conn *xext.Conn, xcb *xcbshim.Connection, clientSideDecorationsSupported bool, xMainScreenIndex int, xWindow xproto.Window, atoms *XcbAtoms, scaleFactor float32, appearance gpui.WindowAppearance, parentWindow gpui.Option[X11WindowStatePtr], supportsXinputGestures bool, isBGR bool) (*X11WindowState, error) {
	if _, ok := params.Kind.(gpui.WindowKindAnchoredPopup); ok {
		return nil, gpui.PopupNotSupportedError{}
	}
	_, isPopUp := params.Kind.(gpui.WindowKindPopUp)
	_, isFloating := params.Kind.(gpui.WindowKindFloating)
	_, isDialog := params.Kind.(gpui.WindowKindDialog)

	c := conn.X
	xScreenIndex := xMainScreenIndex
	if did, ok := params.DisplayID.Get(); ok {
		xScreenIndex = int(uint64(did))
	}
	setup := xproto.Setup(c)
	if xScreenIndex < 0 || xScreenIndex >= len(setup.Roots) {
		return nil, fmt.Errorf("No screen found with index %d", xScreenIndex)
	}
	vs := findVisuals(setup, xScreenIndex)

	vis, ok := vs.transparent.Get()
	if !ok {
		slog.Warn("Unable to find a transparent visual")
		vis = vs.inherit
	}
	slog.Info(fmt.Sprintf("Using %+v", vis))

	colormap := vis.colormap
	if colormap == 0 {
		id, err := xproto.NewColormapId(c)
		if err != nil {
			return nil, err
		}
		slog.Info(fmt.Sprintf("Creating colormap %d", id))
		if err := checkReply(fmt.Sprintf("X11 CreateColormap failed. id: %d", id),
			xproto.CreateColormapChecked(c, xproto.ColormapAllocNone, id, xproto.Window(vs.root), vis.id).Check()); err != nil {
			return nil, err
		}
		colormap = uint32(id)
	}

	overrideRedirect := uint32(0)
	if isPopUp {
		overrideRedirect = 1
	}
	eventMask := uint32(xproto.EventMaskExposure | xproto.EventMaskStructureNotify | xproto.EventMaskFocusChange |
		xproto.EventMaskKeyPress | xproto.EventMaskKeyRelease | xproto.EventMaskPropertyChange |
		xproto.EventMaskVisibilityChange)
	valueMask := uint32(xproto.CwBorderPixel | xproto.CwOverrideRedirect | xproto.CwEventMask | xproto.CwColormap)
	valueList := []uint32{vs.blackPixel, overrideRedirect, eventMask, colormap}

	bounds := params.Bounds.ToDevicePixels(scaleFactor)
	if bounds.Size.Width == 0 || bounds.Size.Height == 0 {
		slog.Warn(fmt.Sprintf("Window bounds contain a zero value. height=%d, width=%d. Falling back to defaults.",
			bounds.Size.Height, bounds.Size.Width))
		bounds.Size.Width = 800
		bounds.Size.Height = 600
	}

	if err := checkReply(
		fmt.Sprintf("X11 CreateWindow failed. depth: %d, x_window: %d, visual_set.root: %d, bounds.origin.x.0: %d, bounds.origin.y.0: %d, bounds.size.width.0: %d, bounds.size.height.0: %d",
			vis.depth, xWindow, vs.root, bounds.Origin.X+2, bounds.Origin.Y, bounds.Size.Width, bounds.Size.Height),
		xproto.CreateWindowChecked(c, vis.depth, xWindow, xproto.Window(vs.root),
			int16(bounds.Origin.X+2), int16(bounds.Origin.Y), uint16(bounds.Size.Width), uint16(bounds.Size.Height),
			0, xproto.WindowClassInputOutput, vis.id, valueMask, valueList).Check()); err != nil {
		return nil, err
	}

	state, setupErr := func() (*X11WindowState, error) {
		pid := uint32(os.Getpid())
		if err := checkReply("X11 ChangeProperty for _NET_WM_PID failed.",
			changeProperty32(c, xWindow, atoms._NET_WM_PID, xproto.AtomCardinal, pid)); err != nil {
			return nil, err
		}

		geomReply, err := xproto.GetGeometry(c, xproto.Drawable(xWindow)).Reply()
		geom, err := getReply("X11 GetGeometry failed.", geomReply, err)
		if err != nil {
			return nil, err
		}
		if geom.X == 0 && geom.Y == 0 {
			bounds.Origin.X += 2
			x, y := int32(bounds.Origin.X), int32(bounds.Origin.Y)
			if err := checkReply(fmt.Sprintf("X11 ConfigureWindow failed. x: %d, y: %d", x, y),
				xproto.ConfigureWindowChecked(c, xWindow, xproto.ConfigWindowX|xproto.ConfigWindowY, []uint32{uint32(x), uint32(y)}).Check()); err != nil {
				return nil, err
			}
		}
		if titlebar, ok := params.Titlebar.Get(); ok {
			if title, ok := titlebar.Title.Get(); ok {
				if err := checkReply("X11 ChangeProperty8 on WM_NAME failed.",
					changeProperty8(c, xWindow, xproto.AtomWmName, xproto.AtomString, []byte(title.String()))); err != nil {
					return nil, err
				}
				if err := checkReply("X11 ChangeProperty8 on _NET_WM_NAME failed.",
					changeProperty8(c, xWindow, atoms._NET_WM_NAME, atoms.UTF8_STRING, []byte(title.String()))); err != nil {
					return nil, err
				}
			}
		}
		if isPopUp {
			if err := checkReply("X11 ChangeProperty32 setting window type for pop-up failed.",
				changeProperty32(c, xWindow, atoms._NET_WM_WINDOW_TYPE, xproto.AtomAtom, uint32(atoms._NET_WM_WINDOW_TYPE_NOTIFICATION))); err != nil {
				return nil, err
			}
		}
		if isFloating || isDialog {
			if pw, ok := parentWindow.Get(); ok {
				if err := checkReply("X11 ChangeProperty32 setting WM_TRANSIENT_FOR for floating window failed.",
					changeProperty32(c, xWindow, atoms.WM_TRANSIENT_FOR, xproto.AtomWindow, uint32(pw.xWindow))); err != nil {
					return nil, err
				}
			}
		}

		parent := gpui.None[X11WindowStatePtr]()
		if pw, ok := parentWindow.Get(); ok && isDialog {
			pw.addChild(xWindow)
			parent = gpui.Some(pw)
		}

		if isDialog {
			if err := checkReply("X11 ChangeProperty32 setting window type for dialog window failed.",
				changeProperty32(c, xWindow, atoms._NET_WM_WINDOW_TYPE, xproto.AtomAtom, uint32(atoms._NET_WM_WINDOW_TYPE_DIALOG))); err != nil {
				return nil, err
			}
			if err := checkReply("X11 ChangeProperty32 setting modal state for dialog window failed.",
				changeProperty32(c, xWindow, atoms._NET_WM_STATE, xproto.AtomAtom, uint32(atoms._NET_WM_STATE_MODAL))); err != nil {
				return nil, err
			}
		}

		if err := checkReply("X11 ChangeProperty32 setting protocols failed.",
			changeProperty32(c, xWindow, atoms.WM_PROTOCOLS, xproto.AtomAtom, uint32(atoms.WM_DELETE_WINDOW), uint32(atoms._NET_WM_SYNC_REQUEST))); err != nil {
			return nil, err
		}

		syncReply, err := xext.SyncInitialize(c, 3, 1).Reply()
		if _, err := getReply("X11 sync protocol initialize failed.", syncReply, err); err != nil {
			return nil, err
		}
		syncRequestCounter, err := xext.NewSyncCounterID(c)
		if err != nil {
			return nil, err
		}
		createCookie := xext.SyncCreateCounterChecked(c, syncRequestCounter, xext.SyncInt64{Lo: 0, Hi: 0})
		if err := checkReply("X11 sync CreateCounter failed.", createCookie.Check()); err != nil {
			return nil, err
		}
		if err := checkReply("X11 ChangeProperty32 setting sync request counter failed.",
			changeProperty32(c, xWindow, atoms._NET_WM_SYNC_REQUEST_COUNTER, xproto.AtomCardinal, uint32(syncRequestCounter))); err != nil {
			return nil, err
		}

		xiEventMask := xext.XinputXIEventMaskMotion | xext.XinputXIEventMaskButtonPress | xext.XinputXIEventMaskButtonRelease |
			xext.XinputXIEventMaskEnter | xext.XinputXIEventMaskLeave
		if supportsXinputGestures {
			xiEventMask |= xext.XinputXIEventMask(1<<xext.XinputGesturePinchBegin) |
				xext.XinputXIEventMask(1<<xext.XinputGesturePinchUpdate) |
				xext.XinputXIEventMask(1<<xext.XinputGesturePinchEnd)
		}
		selectCookie := xext.XinputXISelectEventsChecked(c, xWindow, []xext.XinputEventMask{{
			Deviceid: XinputAllDeviceGroups,
			Mask:     []xext.XinputXIEventMask{xiEventMask},
		}})
		if err := checkReply("X11 XiSelectEvents failed.", selectCookie.Check()); err != nil {
			return nil, err
		}
		deviceCookie := xext.XinputXISelectEventsChecked(c, xWindow, []xext.XinputEventMask{{
			Deviceid: XinputAllDevices,
			Mask:     []xext.XinputXIEventMask{xext.XinputXIEventMaskHierarchy | xext.XinputXIEventMaskDeviceChanged},
		}})
		if err := checkReply("X11 XiSelectEvents for device changes failed.", deviceCookie.Check()); err != nil {
			return nil, err
		}

		xcbFlush(c)
		focusReply, err := xproto.GetInputFocus(c).Reply()
		if _, err := getReply("X11 GetInputFocus round trip failed.", focusReply, err); err != nil {
			return nil, err
		}

		size, err := queryRenderExtent(c, xWindow)
		if err != nil {
			return nil, err
		}
		var xcbRaw uintptr
		if xcb != nil {
			xcbRaw = xcb.Raw()
		}
		raw := rawWindow{connection: xcbRaw, screenID: xScreenIndex, windowID: uint32(xWindow), visualID: uint32(vis.id)}
		config := wgpu.WgpuSurfaceConfig{
			Size:                 size,
			Transparent:          false,
			PreferredPresentMode: gpui.None[wgpu.PresentMode](),
		}
		renderer, err := wgpu.NewWgpuRenderer(gpuContext, raw.handle(), config, compositorGPU)
		if err != nil {
			return nil, err
		}
		cleanup := true
		defer func() {
			if cleanup {
				renderer.Destroy()
			}
		}()

		renderer.SetSubpixelLayout(isBGR)

		maxTextureSize := renderer.MaxTextureSize()
		minSize := gpui.None[[2]int32]()
		if size, ok := params.WindowMinSize.Get(); ok {
			minSize = gpui.Some([2]int32{int32(float32(size.Width)), int32(float32(size.Height))})
		}
		maxSize := gpui.Some([2]int32{int32(maxTextureSize), int32(maxTextureSize)})
		if err := checkReply(fmt.Sprintf("X11 change of WM_SIZE_HINTS failed. max_size: %d", maxTextureSize),
			xproto.ChangePropertyChecked(c, xproto.PropModeReplace, xWindow, xproto.AtomWmNormalHints, xproto.AtomWmSizeHints, 32, numWmSizeHintsElements, wmSizeHintsNormal(minSize, maxSize)).Check()); err != nil {
			return nil, err
		}

		if img, ok := params.Icon.Get(); ok && img != nil {
			if err := checkReply("X11 ChangeProperty32 for _NET_ICON_NAME failed.",
				changeProperty32(c, xWindow, atoms._NET_WM_ICON, xproto.AtomCardinal, iconPropertyData(img)...)); err != nil {
				return nil, err
			}
		}

		display, err := NewX11Display(setup, scaleFactor, xScreenIndex)
		if err != nil {
			return nil, err
		}

		cleanup = false
		return &X11WindowState{
			parent:                         parent,
			children:                       map[xproto.Window]struct{}{},
			client:                         client,
			executor:                       executor,
			display:                        display,
			xRootWindow:                    xproto.Window(vs.root),
			xScreenIndex:                   xScreenIndex,
			visualID:                       uint32(vis.id),
			bounds:                         bounds.ToPixels(scaleFactor),
			scaleFactor:                    scaleFactor,
			renderer:                       renderer,
			atoms:                          *atoms,
			visibility:                     gpui.WindowVisibilityHidden,
			appearance:                     appearance,
			handle:                         handle,
			backgroundAppearance:           gpui.WindowBackgroundAppearanceOpaque,
			clientSideDecorationsSupported: clientSideDecorationsSupported,
			decorations:                    gpui.WindowDecorationsServer,
			counterID:                      syncRequestCounter,
			lastSyncCounter:                gpui.None[xext.SyncInt64](),
		}, nil
	}()

	if setupErr != nil {
		if pw, ok := parentWindow.Get(); ok && isDialog {
			delete(pw.state.children, xWindow)
		}
		if err := checkReply("X11 DestroyWindow failed while cleaning it up after setup failure.",
			xproto.DestroyWindowChecked(c, xWindow).Check()); err != nil {
			return nil, err
		}
		xcbFlush(c)
		return nil, setupErr
	}
	return state, nil
}

func (s *X11WindowState) contentSize() gpui.Size[gpui.Pixels] {
	return s.bounds.Size
}

// X11Window is the X11 platform window.
type X11Window struct {
	gpui.PlatformWindowDefaults
	ptr    X11WindowStatePtr
	closed bool
}

// Close is the window's drop: it destroys the renderer, then the X window, forgets the window
// in the client and runs the close callback on the foreground executor.
func (w *X11Window) Close() {
	if w.closed {
		return
	}
	w.closed = true
	state := w.ptr.state
	if parent, ok := state.parent.Get(); ok {
		delete(parent.state.children, w.ptr.xWindow)
	}
	if state.renderer != nil {
		renderer := state.renderer
		state.renderer = nil
		renderer.Destroy()
	}
	c := w.ptr.conn.X
	if logErr(checkReply("X11 DestroyWindow failure.", xproto.DestroyWindowChecked(c, w.ptr.xWindow).Check())) {
		xcbFlush(c)
	}
	client := state.client
	executor := state.executor
	client.DropWindow(uint32(w.ptr.xWindow))
	thisPtr := w.ptr
	executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		thisPtr.Close()
		return struct{}{}, nil
	}).Detach()
}

type wmHintPropertyState uint32

const wmHintPropertyStateToggle wmHintPropertyState = 2

// NewX11Window creates an X11 window.
func NewX11Window(handle gpui.AnyWindowHandle, client X11ClientStatePtr, executor gpui.ForegroundExecutor, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], params gpui.WindowParams, conn *xext.Conn, xcb *xcbshim.Connection, clientSideDecorationsSupported bool, xMainScreenIndex int, xWindow xproto.Window, atoms *XcbAtoms, scaleFactor float32, appearance gpui.WindowAppearance, parentWindow gpui.Option[X11WindowStatePtr], supportsXinputGestures bool, isBGR bool) (*X11Window, error) {
	state, err := NewX11WindowState(handle, client, executor, gpuContext, compositorGPU, params, conn, xcb,
		clientSideDecorationsSupported, xMainScreenIndex, xWindow, atoms, scaleFactor, appearance, parentWindow,
		supportsXinputGestures, isBGR)
	if err != nil {
		return nil, err
	}
	ptr := X11WindowStatePtr{
		state:     state,
		callbacks: &Callbacks{},
		conn:      conn,
		xcb:       xcb,
		xWindow:   xWindow,
	}
	if err := ptr.setWmProperties(); err != nil {
		if parent, ok := state.parent.Get(); ok {
			delete(parent.state.children, xWindow)
		}
		if state.renderer != nil {
			renderer := state.renderer
			state.renderer = nil
			renderer.Destroy()
		}
		return nil, err
	}
	w := &X11Window{ptr: ptr}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w, nil
}

func (w *X11Window) setWmHints(failureContext string, hintState wmHintPropertyState, prop1, prop2 uint32) error {
	state := w.ptr.state
	message := clientMessage(w.ptr.xWindow, state.atoms._NET_WM_STATE, [5]uint32{uint32(hintState), prop1, prop2, 1, 0})
	c := w.ptr.conn.X
	if err := checkReply(failureContext, xproto.SendEventChecked(c, false, state.xRootWindow, substructureMask, message).Check()); err != nil {
		return err
	}
	xcbFlush(c)
	return nil
}

func (w *X11Window) getRootPosition(position gpui.Point[gpui.Pixels]) (*xproto.TranslateCoordinatesReply, error) {
	state := w.ptr.state
	reply, err := xproto.TranslateCoordinates(w.ptr.conn.X, w.ptr.xWindow, state.xRootWindow,
		int16(float32(position.X)*state.scaleFactor), int16(float32(position.Y)*state.scaleFactor)).Reply()
	return getReply("X11 TranslateCoordinates failed.", reply, err)
}

func (w *X11Window) sendMoveresize(flag uint32) error {
	state := w.ptr.state
	c := w.ptr.conn.X
	if err := checkReply("X11 UngrabPointer before move/resize of window failed.",
		xproto.UngrabPointerChecked(c, xproto.TimeCurrentTime).Check()); err != nil {
		return err
	}
	reply, err := xproto.QueryPointer(c, w.ptr.xWindow).Reply()
	pointer, err := getReply("X11 QueryPointer before move/resize of window failed.", reply, err)
	if err != nil {
		return err
	}
	message := clientMessage(w.ptr.xWindow, state.atoms._NET_WM_MOVERESIZE,
		[5]uint32{uint32(int32(pointer.RootX)), uint32(int32(pointer.RootY)), flag, 0, 0})
	if err := checkReply("X11 SendEvent to move/resize window failed.",
		xproto.SendEventChecked(c, false, state.xRootWindow, substructureMask, message).Check()); err != nil {
		return err
	}
	xcbFlush(c)
	return nil
}

// ShouldClose asks the window's should-close callback.
func (w X11WindowStatePtr) ShouldClose() bool {
	cb := w.callbacks
	if shouldClose := cb.shouldClose; shouldClose != nil {
		cb.shouldClose = nil
		result := shouldClose()
		cb.shouldClose = shouldClose
		return result
	}
	return true
}

// PropertyNotify handles a PropertyNotify event on the window.
func (w X11WindowStatePtr) PropertyNotify(event xproto.PropertyNotifyEvent) error {
	state := w.state
	if event.Atom == state.atoms._NET_WM_STATE {
		wasFullscreen := state.fullscreen
		if err := w.setWmProperties(); err != nil {
			return err
		}
		if w.state.fullscreen != wasFullscreen {
			w.notifyResize()
		}
	} else if event.Atom == state.atoms._GTK_EDGE_CONSTRAINTS {
		if err := w.setEdgeConstraints(); err != nil {
			return err
		}
	}
	return nil
}

func (w X11WindowStatePtr) setEdgeConstraints() error {
	state := w.state
	r, err := xproto.GetProperty(w.conn.X, false, w.xWindow, state.atoms._GTK_EDGE_CONSTRAINTS, xproto.AtomCardinal, 0, 4).Reply()
	reply, err := getReply("X11 GetProperty for _GTK_EDGE_CONSTRAINTS failed.", r, err)
	if err != nil {
		return err
	}
	if reply.ValueLen != 0 {
		if len(reply.Value) >= 4 {
			atom := binary.LittleEndian.Uint32(reply.Value[0:4])
			state.edgeConstraints = gpui.Some(edgeConstraintsFromAtom(atom))
		} else {
			slog.Error("Failed to parse GTK_EDGE_CONSTRAINTS")
		}
	}
	return nil
}

func (w X11WindowStatePtr) setWmProperties() error {
	state := w.state
	r, err := xproto.GetProperty(w.conn.X, false, w.xWindow, state.atoms._NET_WM_STATE, xproto.AtomAtom, 0, math.MaxUint32).Reply()
	reply, err := getReply("X11 GetProperty for _NET_WM_STATE failed.", r, err)
	if err != nil {
		return err
	}

	wasActive := state.active
	wasMaximized := state.maximizedVertical && state.maximizedHorizontal
	state.active = false
	state.fullscreen = false
	state.maximizedVertical = false
	state.maximizedHorizontal = false
	state.hidden = false

	for i := 0; i+4 <= len(reply.Value); i += 4 {
		atom := xproto.Atom(binary.LittleEndian.Uint32(reply.Value[i:]))
		switch atom {
		case state.atoms._NET_WM_STATE_FOCUSED:
			state.active = true
		case state.atoms._NET_WM_STATE_FULLSCREEN:
			state.fullscreen = true
		case state.atoms._NET_WM_STATE_MAXIMIZED_VERT:
			state.maximizedVertical = true
		case state.atoms._NET_WM_STATE_MAXIMIZED_HORZ:
			state.maximizedHorizontal = true
		case state.atoms._NET_WM_STATE_HIDDEN:
			state.hidden = true
		}
	}

	if state.active && !wasActive {
		setWmHintsUrgency(w.conn.X, w.xWindow, false)
	}

	maximizedChanged := (state.maximizedVertical && state.maximizedHorizontal) != wasMaximized
	if maximizedChanged {
		if fun := w.callbacks.moved; fun != nil {
			w.callbacks.moved = nil
			fun()
			w.callbacks.moved = fun
		}
	}
	return nil
}

func (w X11WindowStatePtr) addChild(child xproto.Window) {
	w.state.children[child] = struct{}{}
}

// IsBlocked reports whether a modal child blocks the window.
func (w X11WindowStatePtr) IsBlocked() bool {
	return len(w.state.children) != 0
}

// Close closes the window's modal children and runs its close callback.
func (w X11WindowStatePtr) Close() {
	state := w.state
	client := state.client
	children := make([]xproto.Window, 0, len(state.children))
	for child := range state.children {
		children = append(children, child)
	}
	if c, ok := client.GetClient(); ok {
		for _, child := range children {
			if childWindow, ok := c.getWindow(child); ok {
				childWindow.Close()
			}
		}
	}
	if fun := w.callbacks.close; fun != nil {
		w.callbacks.close = nil
		fun()
	}
}

// Refresh requests a frame.
func (w X11WindowStatePtr) Refresh(requestFrameOptions gpui.RequestFrameOptions) {
	fun := w.callbacks.requestFrame
	if fun == nil {
		return
	}
	w.callbacks.requestFrame = nil
	requestFrameOptions.ForceRender = requestFrameOptions.ForceRender || w.state.forceRenderAfterRecovery
	w.state.forceRenderAfterRecovery = false
	fun(requestFrameOptions)
	w.callbacks.requestFrame = fun
}

// HandleInput dispatches platform input to the window.
func (w X11WindowStatePtr) HandleInput(input gpui.PlatformInput) {
	if w.IsBlocked() {
		return
	}
	if fun := w.callbacks.input; fun != nil {
		w.callbacks.input = nil
		result := fun(input)
		w.callbacks.input = fun
		if !result.Propagate {
			return
		}
	}
	if event, ok := input.(gpui.KeyDownEvent); ok {
		if event.Keystroke.Modifiers.IsSubsetOf(gpui.ModifiersShift()) {
			state := w.state
			if inputHandler := state.inputHandler; inputHandler != nil {
				state.inputHandler = nil
				if keyChar, ok := event.Keystroke.KeyChar.Get(); ok {
					inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), keyChar)
				}
				w.state.inputHandler = inputHandler
			}
		}
	}
}

func (w X11WindowStatePtr) withInputHandler(f func(h *gpui.PlatformInputHandler)) {
	state := w.state
	if inputHandler := state.inputHandler; inputHandler != nil {
		state.inputHandler = nil
		f(inputHandler)
		w.state.inputHandler = inputHandler
	}
}

// HandleIMECommit commits IME text.
func (w X11WindowStatePtr) HandleIMECommit(text string) {
	if w.IsBlocked() {
		return
	}
	w.withInputHandler(func(h *gpui.PlatformInputHandler) {
		h.ReplaceTextInRange(gpui.None[gpui.Range](), text)
	})
}

// HandleIMEPreedit sets IME preedit text.
func (w X11WindowStatePtr) HandleIMEPreedit(text string) {
	if w.IsBlocked() {
		return
	}
	w.withInputHandler(func(h *gpui.PlatformInputHandler) {
		h.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), text, gpui.None[gpui.Range]())
	})
}

// HandleIMEUnmark clears the IME marked text.
func (w X11WindowStatePtr) HandleIMEUnmark() {
	if w.IsBlocked() {
		return
	}
	w.withInputHandler(func(h *gpui.PlatformInputHandler) {
		h.UnmarkText()
	})
}

// HandleIMEDelete deletes the IME marked text.
func (w X11WindowStatePtr) HandleIMEDelete() {
	if w.IsBlocked() {
		return
	}
	w.withInputHandler(func(h *gpui.PlatformInputHandler) {
		if marked, ok := h.MarkedTextRange(); ok {
			h.ReplaceTextInRange(gpui.Some(marked), "")
		}
	})
}

// GetIMEArea returns the IME cursor area in device pixels.
func (w X11WindowStatePtr) GetIMEArea() (gpui.Bounds[gpui.ScaledPixels], bool) {
	scaleFactor := w.state.scaleFactor
	var bounds gpui.Bounds[gpui.Pixels]
	found := false
	w.withInputHandler(func(h *gpui.PlatformInputHandler) {
		if selection, ok := h.SelectedTextRange(true); ok {
			bounds, found = h.BoundsForRange(selection.Range)
		}
	})
	if !found {
		return gpui.Bounds[gpui.ScaledPixels]{}, false
	}
	return bounds.Scale(scaleFactor), true
}

// SetBounds applies a ConfigureNotify.
func (w X11WindowStatePtr) SetBounds(bounds gpui.Bounds[int32]) error {
	state := w.state
	sf := state.scaleFactor
	pxBounds := gpui.Bounds[gpui.Pixels]{
		Origin: gpui.Point[gpui.Pixels]{X: gpui.Px(float32(bounds.Origin.X) / sf), Y: gpui.Px(float32(bounds.Origin.Y) / sf)},
		Size:   gpui.Size[gpui.Pixels]{Width: gpui.Px(float32(bounds.Size.Width) / sf), Height: gpui.Px(float32(bounds.Size.Height) / sf)},
	}
	isResize := pxBounds.Size.Width != state.bounds.Size.Width || pxBounds.Size.Height != state.bounds.Size.Height
	if isResize {
		state.bounds.Size = pxBounds.Size
	} else {
		state.bounds = pxBounds
	}

	gpuSize, err := queryRenderExtent(w.conn.X, w.xWindow)
	if err != nil {
		return err
	}
	if state.renderer != nil {
		state.renderer.UpdateDrawableSize(gpuSize)
	}
	contentSize, scaleFactor := state.contentSize(), state.scaleFactor
	if value, ok := state.lastSyncCounter.Get(); ok {
		state.lastSyncCounter = gpui.None[xext.SyncInt64]()
		if err := checkReply("X11 sync SetCounter failed.", xext.SyncSetCounterChecked(w.conn.X, state.counterID, value).Check()); err != nil {
			return err
		}
	}

	if fun := w.callbacks.resize; fun != nil {
		fun(contentSize, scaleFactor)
	}
	if !isResize {
		if fun := w.callbacks.moved; fun != nil {
			fun()
		}
	}
	return nil
}

func (w X11WindowStatePtr) notifyResize() {
	contentSize, scaleFactor := w.state.contentSize(), w.state.scaleFactor
	if fun := w.callbacks.resize; fun != nil {
		w.callbacks.resize = nil
		fun(contentSize, scaleFactor)
		w.callbacks.resize = fun
	}
}

// SetActive records keyboard focus.
func (w X11WindowStatePtr) SetActive(focus bool) {
	if fun := w.callbacks.activeStatusChange; fun != nil {
		w.callbacks.activeStatusChange = nil
		fun(focus)
		w.callbacks.activeStatusChange = fun
	}
	if adapter := w.state.accesskitAdapter; adapter != nil {
		adapter.UpdateWindowFocusState(focus)
	}
}

// SetHovered records pointer hover.
func (w X11WindowStatePtr) SetHovered(focus bool) {
	if fun := w.callbacks.hoveredStatusChange; fun != nil {
		w.callbacks.hoveredStatusChange = nil
		fun(focus)
		w.callbacks.hoveredStatusChange = fun
	}
}

// SetVisibility reports presentation visibility.
func (w X11WindowStatePtr) SetVisibility(visibility gpui.WindowVisibility) {
	previous := w.state.visibility
	w.state.visibility = visibility
	if previous == visibility {
		return
	}
	if fun := w.callbacks.visibilityChange; fun != nil {
		w.callbacks.visibilityChange = nil
		fun(visibility)
		w.callbacks.visibilityChange = fun
	}
}

// SetAppearance applies a system appearance change.
func (w X11WindowStatePtr) SetAppearance(appearance gpui.WindowAppearance) {
	state := w.state
	state.appearance = appearance
	state.updateTransparency()
	state.appearance = appearance
	if fun := w.callbacks.appearanceChanged; fun != nil {
		w.callbacks.appearanceChanged = nil
		fun()
		w.callbacks.appearanceChanged = fun
	}
}

// SetButtonLayout applies a window button layout change.
func (w X11WindowStatePtr) SetButtonLayout() {
	if fun := w.callbacks.buttonLayoutChanged; fun != nil {
		w.callbacks.buttonLayoutChanged = nil
		fun()
		w.callbacks.buttonLayoutChanged = fun
	}
}

// RawWindowHandle returns the xcb window and display handle.
func (w *X11Window) RawWindowHandle() gpui.RawWindowHandle {
	var connection uintptr
	if w.ptr.xcb != nil {
		connection = w.ptr.xcb.Raw()
	}
	return gpui.RawWindowHandle{Kind: gpui.RawWindowHandleXcb, XcbConnection: connection, XcbWindow: uint32(w.ptr.xWindow)}
}

// Bounds returns the window bounds.
func (w *X11Window) Bounds() gpui.Bounds[gpui.Pixels] { return w.ptr.state.bounds }

// IsMaximized reports whether the window is maximized.
func (w *X11Window) IsMaximized() bool { return w.ptr.state.isMaximized() }

// WindowBounds returns the window bounds with its maximized state.
func (w *X11Window) WindowBounds() gpui.WindowBounds {
	state := w.ptr.state
	if w.IsMaximized() {
		return gpui.WindowBoundsMaximized(state.bounds)
	}
	return gpui.WindowBoundsWindowed(state.bounds)
}

// InnerWindowBounds returns the window bounds without the client-side decoration insets.
func (w *X11Window) InnerWindowBounds() gpui.WindowBounds {
	state := w.ptr.state
	if w.IsMaximized() {
		return gpui.WindowBoundsMaximized(state.bounds)
	}
	bounds := state.bounds
	insets := state.lastInsets
	left := gpui.Px(float32(insets[0]) / state.scaleFactor)
	right := gpui.Px(float32(insets[1]) / state.scaleFactor)
	top := gpui.Px(float32(insets[2]) / state.scaleFactor)
	bottom := gpui.Px(float32(insets[3]) / state.scaleFactor)
	bounds.Origin.X += left
	bounds.Origin.Y += top
	bounds.Size.Width -= left + right
	bounds.Size.Height -= top + bottom
	return gpui.WindowBoundsWindowed(bounds)
}

// ContentSize returns the content size in logical pixels.
func (w *X11Window) ContentSize() gpui.Size[gpui.Pixels] { return w.ptr.state.contentSize() }

// Resize asks the window manager to resize the window.
func (w *X11Window) Resize(size gpui.Size[gpui.Pixels]) {
	state := w.ptr.state
	device := size.ToDevicePixels(state.scaleFactor)
	width, height := uint32(device.Width), uint32(device.Height)
	logErr(checkReply(fmt.Sprintf("X11 ConfigureWindow failed. width: %d, height: %d", width, height),
		xproto.ConfigureWindowChecked(w.ptr.conn.X, w.ptr.xWindow, xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{width, height}).Check()))
	xcbFlush(w.ptr.conn.X)
}

// ScaleFactor returns the scale factor.
func (w *X11Window) ScaleFactor() float32 { return w.ptr.state.scaleFactor }

// Appearance returns the window appearance.
func (w *X11Window) Appearance() gpui.WindowAppearance { return w.ptr.state.appearance }

// Display returns the X screen of the window.
func (w *X11Window) Display() gpui.PlatformDisplay { return w.ptr.state.display }

// MousePosition returns the pointer position relative to the window.
func (w *X11Window) MousePosition() gpui.Point[gpui.Pixels] {
	r, err := xproto.QueryPointer(w.ptr.conn.X, w.ptr.xWindow).Reply()
	reply, err := getReply("X11 QueryPointer failed.", r, err)
	if !logErr(err) {
		return gpui.Point[gpui.Pixels]{}
	}
	sf := w.ptr.state.scaleFactor
	return gpui.Point[gpui.Pixels]{X: gpui.Px(float32(reply.WinX) / sf), Y: gpui.Px(float32(reply.WinY) / sf)}
}

// Modifiers returns the last modifier state seen by the client.
func (w *X11Window) Modifiers() gpui.Modifiers {
	if c, ok := w.ptr.state.client.GetClient(); ok {
		return c.s.Modifiers
	}
	return gpui.Modifiers{}
}

// Capslock returns the last capslock state seen by the client.
func (w *X11Window) Capslock() gpui.Capslock {
	if c, ok := w.ptr.state.client.GetClient(); ok {
		return c.s.Capslock
	}
	return gpui.Capslock{}
}

// SetInputHandler installs the input handler.
func (w *X11Window) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.ptr.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the input handler.
func (w *X11Window) TakeInputHandler() *gpui.PlatformInputHandler {
	h := w.ptr.state.inputHandler
	w.ptr.state.inputHandler = nil
	return h
}

// Prompt returns nil: X11 has no native prompts.
func (w *X11Window) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	return nil
}

// Activate asks the window manager to activate the window.
func (w *X11Window) Activate() {
	state := w.ptr.state
	message := clientMessage(w.ptr.xWindow, state.atoms._NET_ACTIVE_WINDOW, [5]uint32{1, uint32(xproto.TimeCurrentTime), 0, 0, 0})
	xproto.SendEvent(w.ptr.conn.X, false, state.xRootWindow, substructureMask, message)
	xcbFlush(w.ptr.conn.X)
}

// RequestAttention sets the urgency hint unless the window is active.
func (w *X11Window) RequestAttention() {
	if w.IsActive() {
		return
	}
	setWmHintsUrgency(w.ptr.conn.X, w.ptr.xWindow, true)
}

// IsActive reports whether the window has focus.
func (w *X11Window) IsActive() bool { return w.ptr.state.active }

// Visibility returns the last reported visibility.
func (w *X11Window) Visibility() gpui.WindowVisibility { return w.ptr.state.visibility }

// IsHovered reports whether the pointer is over the window.
func (w *X11Window) IsHovered() bool { return w.ptr.state.hovered }

// SetTitle sets WM_NAME and _NET_WM_NAME.
func (w *X11Window) SetTitle(title string) {
	c := w.ptr.conn.X
	logErr(checkReply("X11 ChangeProperty8 on WM_NAME failed.",
		changeProperty8(c, w.ptr.xWindow, xproto.AtomWmName, xproto.AtomString, []byte(title))))
	logErr(checkReply("X11 ChangeProperty8 on _NET_WM_NAME failed.",
		changeProperty8(c, w.ptr.xWindow, w.ptr.state.atoms._NET_WM_NAME, w.ptr.state.atoms.UTF8_STRING, []byte(title))))
	xcbFlush(c)
}

// SetAppID sets WM_CLASS.
func (w *X11Window) SetAppID(appID string) {
	data := make([]byte, 0, len(appID)*2+2)
	data = append(data, appID...)
	data = append(data, 0)
	data = append(data, appID...)
	data = append(data, 0)
	logErr(checkReply("X11 ChangeProperty8 for WM_CLASS failed.",
		changeProperty8(w.ptr.conn.X, w.ptr.xWindow, xproto.AtomWmClass, xproto.AtomString, data)))
}

// MapWindow maps the window.
func (w *X11Window) MapWindow() error {
	return checkReply("X11 MapWindow failed.", xproto.MapWindowChecked(w.ptr.conn.X, w.ptr.xWindow).Check())
}

// SetBackgroundAppearance sets the background appearance.
func (w *X11Window) SetBackgroundAppearance(backgroundAppearance gpui.WindowBackgroundAppearance) {
	state := w.ptr.state
	state.backgroundAppearance = backgroundAppearance
	state.updateTransparency()
}

// BackgroundAppearance returns the background appearance.
func (w *X11Window) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return w.ptr.state.backgroundAppearance
}

// IsSubpixelRenderingSupported reports whether the GPU supports dual-source blending.
func (w *X11Window) IsSubpixelRenderingSupported() bool {
	c, ok := w.ptr.state.client.GetClient()
	if !ok {
		return false
	}
	gpuContext := c.s.gpuContext
	return gpuContext != nil && gpuContext.Context != nil && gpuContext.Context.SupportsDualSourceBlending()
}

// Minimize asks the window manager to iconify the window.
func (w *X11Window) Minimize() {
	state := w.ptr.state
	const windowIconicState uint32 = 3
	message := clientMessage(w.ptr.xWindow, state.atoms.WM_CHANGE_STATE, [5]uint32{windowIconicState, 0, 0, 0, 0})
	logErr(checkReply("X11 SendEvent to minimize window failed.",
		xproto.SendEventChecked(w.ptr.conn.X, false, state.xRootWindow, substructureMask, message).Check()))
}

// Zoom toggles the maximized state.
func (w *X11Window) Zoom() {
	state := w.ptr.state
	logErr(w.setWmHints("X11 SendEvent to maximize a window failed.", wmHintPropertyStateToggle,
		uint32(state.atoms._NET_WM_STATE_MAXIMIZED_VERT), uint32(state.atoms._NET_WM_STATE_MAXIMIZED_HORZ)))
}

// ToggleFullscreen toggles the fullscreen state.
func (w *X11Window) ToggleFullscreen() {
	state := w.ptr.state
	logErr(w.setWmHints("X11 SendEvent to fullscreen a window failed.", wmHintPropertyStateToggle,
		uint32(state.atoms._NET_WM_STATE_FULLSCREEN), uint32(xproto.AtomNone)))
}

// IsFullscreen reports whether the window is fullscreen.
func (w *X11Window) IsFullscreen() bool { return w.ptr.state.fullscreen }

// OnRequestFrame registers the frame callback.
func (w *X11Window) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.ptr.callbacks.requestFrame = callback
}

// OnInput registers the input callback.
func (w *X11Window) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.ptr.callbacks.input = callback
}

// OnActiveStatusChange registers the focus callback.
func (w *X11Window) OnActiveStatusChange(callback func(active bool)) {
	w.ptr.callbacks.activeStatusChange = callback
}

// OnVisibilityChange registers the visibility callback.
func (w *X11Window) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.ptr.callbacks.visibilityChange = callback
}

// OnHoverStatusChange registers the hover callback.
func (w *X11Window) OnHoverStatusChange(callback func(hovered bool)) {
	w.ptr.callbacks.hoveredStatusChange = callback
}

// OnResize registers the resize callback.
func (w *X11Window) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.ptr.callbacks.resize = callback
}

// OnMoved registers the move callback.
func (w *X11Window) OnMoved(callback func()) { w.ptr.callbacks.moved = callback }

// OnDisplayChanged ignores callback: an X11 display is an X screen spanning every monitor,
// and a window never changes screen.
func (w *X11Window) OnDisplayChanged(callback func()) {}

// OnShouldClose registers the should-close callback.
func (w *X11Window) OnShouldClose(callback func() bool) { w.ptr.callbacks.shouldClose = callback }

// OnClose registers the close callback.
func (w *X11Window) OnClose(callback func()) { w.ptr.callbacks.close = callback }

// OnHitTestWindowControl ignores callback.
func (w *X11Window) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {}

// OnAppearanceChanged registers the appearance callback.
func (w *X11Window) OnAppearanceChanged(callback func()) {
	w.ptr.callbacks.appearanceChanged = callback
}

// OnButtonLayoutChanged registers the button layout callback.
func (w *X11Window) OnButtonLayoutChanged(callback func()) {
	w.ptr.callbacks.buttonLayoutChanged = callback
}

// Draw renders scene, recovering the GPU device first when it was lost.
func (w *X11Window) Draw(scene *gpui.Scene) {
	inner := w.ptr.state
	renderer := inner.renderer
	if renderer == nil {
		return
	}
	if renderer.DeviceLost() {
		var connection uintptr
		if w.ptr.xcb != nil {
			connection = w.ptr.xcb.Raw()
		}
		raw := rawWindow{connection: connection, screenID: inner.xScreenIndex, windowID: uint32(w.ptr.xWindow), visualID: inner.visualID}
		if err := renderer.Recover(raw.handle()); err != nil {
			slog.Warn(fmt.Sprintf("GPU recovery failed, will retry on next frame: %v", err))
		}
		inner.forceRenderAfterRecovery = true
		return
	}
	renderer.Draw(scene)
	if renderer.NeedsRedraw() {
		inner.forceRenderAfterRecovery = true
	}
}

// SpriteAtlas returns the renderer's atlas.
func (w *X11Window) SpriteAtlas() gpui.PlatformAtlas {
	renderer := w.ptr.state.renderer
	if renderer == nil {
		panic("the renderer is only taken when the window is dropped")
	}
	return renderer.SpriteAtlas()
}

// ShowWindowMenu asks the window manager to show its window menu at position.
func (w *X11Window) ShowWindowMenu(position gpui.Point[gpui.Pixels]) {
	state := w.ptr.state
	c := w.ptr.conn.X
	logErr(checkReply("X11 UngrabPointer failed.", xproto.UngrabPointerChecked(c, xproto.TimeCurrentTime).Check()))
	coords, err := w.getRootPosition(position)
	if !logErr(err) {
		return
	}
	message := clientMessage(w.ptr.xWindow, state.atoms._GTK_SHOW_WINDOW_MENU,
		[5]uint32{uint32(XinputAllDeviceGroups), uint32(int32(coords.DstX)), uint32(int32(coords.DstY)), 0, 0})
	logErr(checkReply("X11 SendEvent to show window menu failed.",
		xproto.SendEventChecked(c, false, state.xRootWindow, substructureMask, message).Check()))
}

// StartWindowMove starts an interactive move.
func (w *X11Window) StartWindowMove() {
	const moveresizeMove uint32 = 8
	logErr(w.sendMoveresize(moveresizeMove))
}

// StartWindowResize starts an interactive resize from edge.
func (w *X11Window) StartWindowResize(edge gpui.ResizeEdge) {
	logErr(w.sendMoveresize(resizeEdgeToMoveresize(edge)))
}

// WindowDecorations returns the active decorations.
func (w *X11Window) WindowDecorations() gpui.Decorations {
	state := w.ptr.state
	if !state.clientSideDecorationsSupported {
		return gpui.DecorationsServer()
	}
	switch state.decorations {
	case gpui.WindowDecorationsClient:
		var tiling gpui.Tiling
		if state.fullscreen {
			tiling = gpui.TilingTiled()
		} else if ec, ok := state.edgeConstraints.Get(); ok {
			tiling = ec.toTiling()
		} else {
			tiling = gpui.Tiling{
				Top:    state.maximizedVertical,
				Bottom: state.maximizedVertical,
				Left:   state.maximizedHorizontal,
				Right:  state.maximizedHorizontal,
			}
		}
		return gpui.DecorationsClient(tiling)
	default:
		return gpui.DecorationsServer()
	}
}

// SetClientInset publishes the client-side decoration insets as _GTK_FRAME_EXTENTS.
func (w *X11Window) SetClientInset(inset gpui.Pixels) {
	state := w.ptr.state
	dp := uint32(float32(inset) * state.scaleFactor)
	var insets [4]uint32
	if state.fullscreen {
		insets = [4]uint32{}
	} else if ec, ok := state.edgeConstraints.Get(); ok {
		pick := func(tiled bool) uint32 {
			if tiled {
				return 0
			}
			return dp
		}
		insets = [4]uint32{pick(ec.leftTiled), pick(ec.rightTiled), pick(ec.topTiled), pick(ec.bottomTiled)}
	} else {
		left, right := dp, dp
		if state.maximizedHorizontal {
			left, right = 0, 0
		}
		top, bottom := dp, dp
		if state.maximizedVertical {
			top, bottom = 0, 0
		}
		insets = [4]uint32{left, right, top, bottom}
	}
	if state.lastInsets != insets {
		state.lastInsets = insets
		logErr(checkReply("X11 ChangeProperty for _GTK_FRAME_EXTENTS failed.",
			xproto.ChangePropertyChecked(w.ptr.conn.X, xproto.PropModeReplace, w.ptr.xWindow, state.atoms._GTK_FRAME_EXTENTS,
				xproto.AtomCardinal, 32, 4, u32Bytes(insets[:]...)).Check()))
	}
}

// RequestDecorations sets _MOTIF_WM_HINTS for decorations.
func (w *X11Window) RequestDecorations(decorations gpui.WindowDecorations) {
	state := w.ptr.state
	if decorations == gpui.WindowDecorationsClient && !state.clientSideDecorationsSupported {
		slog.Info("x11: no compositor present, falling back to server-side window decorations")
		decorations = gpui.WindowDecorationsServer
	}
	hintsData := [5]uint32{1 << 1, 0, 1, 0, 0}
	if decorations == gpui.WindowDecorationsClient {
		hintsData = [5]uint32{1 << 1, 0, 0, 0, 0}
	}
	if !logErr(checkReply("X11 ChangeProperty for _MOTIF_WM_HINTS failed.",
		xproto.ChangePropertyChecked(w.ptr.conn.X, xproto.PropModeReplace, w.ptr.xWindow, state.atoms._MOTIF_WM_HINTS,
			state.atoms._MOTIF_WM_HINTS, 32, 5, u32Bytes(hintsData[:]...)).Check())) {
		return
	}
	state.decorations = decorations
	state.updateTransparency()
	if fun := w.ptr.callbacks.appearanceChanged; fun != nil {
		fun()
	}
}

// UpdateIMEPosition moves the XIM preedit spot.
func (w *X11Window) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	w.ptr.state.client.UpdateIMEPosition(bounds)
}

// GPUSpecs returns the renderer's GPU description.
func (w *X11Window) GPUSpecs() (gpui.GPUSpecs, bool) {
	renderer := w.ptr.state.renderer
	if renderer == nil {
		return gpui.GPUSpecs{}, false
	}
	return renderer.GPUSpecs()
}

// PlaySystemBell rings the X bell at the system volume.
func (w *X11Window) PlaySystemBell() {
	xproto.Bell(w.ptr.conn.X, 0)
}

// A11yInit creates the window's accessibility adapter.
func (w *X11Window) A11yInit(callbacks gpui.A11yCallbacks) {
	w.ptr.state.accesskitAdapter = newA11yAdapter(callbacks)
}

// A11yTreeUpdate pushes treeUpdate to an active adapter.
func (w *X11Window) A11yTreeUpdate(treeUpdate a11y.TreeUpdate) {
	if adapter := w.ptr.state.accesskitAdapter; adapter != nil {
		adapter.UpdateIfActive(func() a11y.TreeUpdate { return treeUpdate })
	}
}

// A11yUpdateWindowBounds reports the outer and inner window bounds to the adapter.
func (w *X11Window) A11yUpdateWindowBounds() {
	state := w.ptr.state
	scale := state.scaleFactor
	bounds := state.bounds
	insets := state.lastInsets
	x, y := float32(bounds.Origin.X), float32(bounds.Origin.Y)
	width, height := float32(bounds.Size.Width), float32(bounds.Size.Height)
	outer := a11y.Rect{
		X0: float64(x * scale),
		Y0: float64(y * scale),
		X1: float64((x + width) * scale),
		Y1: float64((y + height) * scale),
	}
	inner := a11y.Rect{
		X0: float64(x*scale) + float64(insets[0]),
		Y0: float64(y*scale) + float64(insets[2]),
		X1: float64((x+width)*scale) - float64(insets[1]),
		Y1: float64((y+height)*scale) - float64(insets[3]),
	}
	if adapter := state.accesskitAdapter; adapter != nil {
		adapter.SetRootWindowBounds(outer, inner)
	}
}
