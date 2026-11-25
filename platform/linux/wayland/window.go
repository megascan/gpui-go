//go:build linux

package wayland

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// Callbacks are the callbacks a window reports platform events through.
type Callbacks struct {
	requestFrame        func(options gpui.RequestFrameOptions)
	input               func(input gpui.PlatformInput) gpui.DispatchEventResult
	activeStatusChange  func(active bool)
	visibilityChange    func(visibility gpui.WindowVisibility)
	hoverStatusChange   func(hovered bool)
	resize              func(size gpui.Size[gpui.Pixels], scaleFactor float32)
	moved               func()
	displayChanged      func()
	shouldClose         func() bool
	close               func()
	appearanceChanged   func()
	buttonLayoutChanged func()
}

func rawWindow(surface *wlproto.WlSurface, globals *Globals) gpui.RawWindowHandle {
	return gpui.RawWindowHandle{
		Kind:           gpui.RawWindowHandleWayland,
		WaylandDisplay: globals.Display.Ptr(),
		WaylandSurface: surface.Ptr(),
	}
}

type inProgressConfigure struct {
	size       gpui.Option[gpui.Size[gpui.Pixels]]
	fullscreen bool
	maximized  bool
	resizing   bool
	visibility gpui.WindowVisibility
	tiling     gpui.Tiling
}

type outputEntry struct {
	id     ObjectID
	output Output
}

// WaylandWindowState is the state of one Wayland window.
type WaylandWindowState struct {
	surfaceState             WaylandSurfaceState
	parent                   gpui.Option[WaylandWindowStatePtr]
	children                 map[ObjectID]bool
	surface                  *wlproto.WlSurface
	appID                    gpui.Option[string]
	appearance               gpui.WindowAppearance
	kdeBlur                  *wlproto.OrgKdeKwinBlur
	backgroundEffects        *wlproto.ExtBackgroundEffectSurfaceV1
	viewport                 *wlproto.WpViewport
	outputs                  map[ObjectID]Output
	display                  gpui.Option[outputEntry]
	globals                  *Globals
	renderer                 *wgpu.WgpuRenderer
	bounds                   gpui.Bounds[gpui.Pixels]
	scale                    float32
	inputHandler             *gpui.PlatformInputHandler
	decorations              gpui.WindowDecorations
	backgroundAppearance     gpui.WindowBackgroundAppearance
	fullscreen               bool
	maximized                bool
	visibility               gpui.WindowVisibility
	tiling                   gpui.Tiling
	windowBounds             gpui.Bounds[gpui.Pixels]
	client                   WaylandClientStatePtr
	handle                   gpui.AnyWindowHandle
	active                   bool
	hovered                  bool
	redrawRequested          bool
	presentation             presentationState
	pendingFrameCallback     *wlproto.WlCallback
	inProgressConfigure      gpui.Option[inProgressConfigure]
	resizeThrottle           bool
	inProgressWindowControls gpui.Option[gpui.WindowControls]
	windowControls           gpui.WindowControls
	clientInset              gpui.Option[gpui.Pixels]
	clientCornerRadii        gpui.Corners[gpui.Pixels]
	accesskitAdapter         a11yAdapter
}

// WaylandSurfaceState is the role of a window's surface: exactly one of its variants is set.
type WaylandSurfaceState struct {
	xdg        *WaylandXdgSurfaceState
	layerShell *WaylandLayerSurfaceState
	popup      *WaylandPopupSurfaceState
}

func newWaylandSurfaceState(surface *wlproto.WlSurface, globals *Globals, client WaylandClientStatePtr, params *gpui.WindowParams, parent gpui.Option[WaylandWindowStatePtr], grab gpui.Option[popupGrab], targetOutput *wlproto.WlOutput) (WaylandSurfaceState, error) {
	surfaceID := &surface.Proxy
	if options, ok := params.Kind.(gpui.WindowKindLayerShell); ok {
		if globals.LayerShell == nil {
			return WaylandSurfaceState{}, gpui.LayerShellNotSupportedError{}
		}

		layerSurface := globals.LayerShell.GetLayerSurface(surface, targetOutput, waylandLayer(options.Layer), options.Namespace)
		layerSurface.SetListener(client.LayerSurfaceListener(surfaceID))

		layerSurface.SetSize(params.Bounds.Size.Width.ToU32(), params.Bounds.Size.Height.ToU32())

		layerSurface.SetAnchor(layerShellWaylandAnchor(options.Anchor))
		layerSurface.SetKeyboardInteractivity(waylandKeyboardInteractivity(options.KeyboardInteractivity))

		if margin, ok := options.Margin.Get(); ok {
			layerSurface.SetMargin(margin[0].ToI32(), margin[1].ToI32(), margin[2].ToI32(), margin[3].ToI32())
		}

		if exclusiveZone, ok := options.ExclusiveZone.Get(); ok {
			layerSurface.SetExclusiveZone(exclusiveZone.ToI32())
		}

		if exclusiveEdge, ok := options.ExclusiveEdge.Get(); ok {
			applyExclusiveEdge(layerSurface, options.Anchor, exclusiveEdge)
		}

		return WaylandSurfaceState{layerShell: &WaylandLayerSurfaceState{
			layerSurface: layerSurface,
			anchor:       options.Anchor,
		}}, nil
	}

	if options, ok := params.Kind.(gpui.WindowKindAnchoredPopup); ok {
		parent, ok := parent.Get()
		if !ok {
			return WaylandSurfaceState{}, errors.New("popup parent window not found")
		}

		positioner := buildPopupPositioner(globals, &options.PopupOptions, params.Bounds.Size, parent.WindowGeometry())

		xdgSurface := globals.WmBase.GetXdgSurface(surface)
		xdgSurface.SetListener(client.XdgSurfaceListener(surfaceID))

		var xdgPopup *wlproto.XdgPopup
		if parentLayerSurface := parent.LayerSurface(); parentLayerSurface != nil {
			xdgPopup = xdgSurface.GetPopup(nil, positioner)
			xdgPopup.SetListener(client.PopupListener(surfaceID))
			parentLayerSurface.GetPopup(xdgPopup)
		} else {
			xdgPopup = xdgSurface.GetPopup(parent.XdgSurface(), positioner)
			xdgPopup.SetListener(client.PopupListener(surfaceID))
		}
		positioner.Destroy()

		if g, ok := grab.Get(); ok {
			xdgPopup.Grab(g.Seat, g.Serial)
		}

		parent.AddChild(surfaceID, false)

		return WaylandSurfaceState{popup: &WaylandPopupSurfaceState{
			xdgSurface: xdgSurface,
			xdgPopup:   xdgPopup,
			options:    options.PopupOptions,
		}}, nil
	}

	xdgSurface := globals.WmBase.GetXdgSurface(surface)
	xdgSurface.SetListener(client.XdgSurfaceListener(surfaceID))

	toplevel := xdgSurface.GetToplevel()
	toplevel.SetListener(client.ToplevelListener(surfaceID))
	var xdgParent *wlproto.XdgToplevel
	if parent, ok := parent.Get(); ok {
		xdgParent = parent.Toplevel()
	}

	_, isFloating := params.Kind.(gpui.WindowKindFloating)
	_, isDialog := params.Kind.(gpui.WindowKindDialog)
	if isFloating || isDialog {
		toplevel.SetParent(xdgParent)
	}

	var dialog *wlproto.XdgDialogV1
	if isDialog {
		if globals.Dialog != nil {
			dialog = globals.Dialog.GetXdgDialog(toplevel)
			dialog.SetModal()
		}

		if parent, ok := parent.Get(); ok {
			parent.AddChild(surfaceID, true)
		}
	}

	if size, ok := params.WindowMinSize.Get(); ok {
		toplevel.SetMinSize(size.Width.ToI32(), size.Height.ToI32())
	}

	var decoration *wlproto.ZxdgToplevelDecorationV1
	if globals.DecorationManager != nil {
		decoration = globals.DecorationManager.GetToplevelDecoration(toplevel)
		decoration.SetListener(client.ToplevelDecorationListener(surfaceID))
	}

	return WaylandSurfaceState{xdg: &WaylandXdgSurfaceState{
		xdgSurface: xdgSurface,
		toplevel:   toplevel,
		decoration: decoration,
		dialog:     dialog,
	}}, nil
}

// WaylandXdgSurfaceState is the xdg_toplevel role of a window.
type WaylandXdgSurfaceState struct {
	xdgSurface *wlproto.XdgSurface
	toplevel   *wlproto.XdgToplevel
	decoration *wlproto.ZxdgToplevelDecorationV1
	dialog     *wlproto.XdgDialogV1
}

// WaylandLayerSurfaceState is the zwlr_layer_surface_v1 role of a window.
type WaylandLayerSurfaceState struct {
	layerSurface *wlproto.ZwlrLayerSurfaceV1
	anchor       gpui.LayerShellAnchor
}

// WaylandPopupSurfaceState is the xdg_popup role of a window.
type WaylandPopupSurfaceState struct {
	xdgSurface          *wlproto.XdgSurface
	xdgPopup            *wlproto.XdgPopup
	options             gpui.PopupOptions
	nextRepositionToken uint32
}

func buildPopupPositioner(globals *Globals, options *gpui.PopupOptions, size gpui.Size[gpui.Pixels], parentGeometry gpui.Bounds[gpui.Pixels]) *wlproto.XdgPositioner {
	positioner := globals.WmBase.CreatePositioner()
	positioner.SetSize(
		gpui.Px(max(size.Width.AsF32(), 1.0)).ToI32(),
		gpui.Px(max(size.Height.AsF32(), 1.0)).ToI32(),
	)

	anchorRect := gpui.NewBounds(options.AnchorRect.Origin.Sub(parentGeometry.Origin), options.AnchorRect.Size)
	one := gpui.NewPoint(gpui.Px(1.0), gpui.Px(1.0))
	geometryBottomRight := gpui.NewPoint(parentGeometry.Size.Width, parentGeometry.Size.Height)
	topLeft := anchorRect.Origin.Min(geometryBottomRight.Sub(one)).Max(gpui.Point[gpui.Pixels]{})
	bottomRight := anchorRect.BottomRight().Min(geometryBottomRight).Max(topLeft.Add(one))
	anchorRect = gpui.BoundsFromCorners(topLeft, bottomRight)
	positioner.SetAnchorRect(
		anchorRect.Origin.X.ToI32(),
		anchorRect.Origin.Y.ToI32(),
		anchorRect.Size.Width.ToI32(),
		anchorRect.Size.Height.ToI32(),
	)

	positioner.SetAnchor(popupWaylandAnchor(options.Anchor))
	positioner.SetGravity(waylandGravity(options.Gravity))
	positioner.SetConstraintAdjustment(waylandConstraintAdjustment(options.ConstraintAdjustment))
	positioner.SetOffset(options.Offset.X.ToI32(), options.Offset.Y.ToI32())
	return positioner
}

func (s *WaylandSurfaceState) ackConfigure(serial uint32) {
	switch {
	case s.xdg != nil:
		s.xdg.xdgSurface.AckConfigure(serial)
	case s.layerShell != nil:
		s.layerShell.layerSurface.AckConfigure(serial)
	case s.popup != nil:
		s.popup.xdgSurface.AckConfigure(serial)
	}
}

func (s *WaylandSurfaceState) decoration() *wlproto.ZxdgToplevelDecorationV1 {
	if s.xdg != nil {
		return s.xdg.decoration
	}
	return nil
}

func (s *WaylandSurfaceState) toplevel() *wlproto.XdgToplevel {
	if s.xdg != nil {
		return s.xdg.toplevel
	}
	return nil
}

func (s *WaylandSurfaceState) xdgSurface() *wlproto.XdgSurface {
	switch {
	case s.xdg != nil:
		return s.xdg.xdgSurface
	case s.popup != nil:
		return s.popup.xdgSurface
	default:
		return nil
	}
}

func (s *WaylandSurfaceState) layerSurface() *wlproto.ZwlrLayerSurfaceV1 {
	if s.layerShell != nil {
		return s.layerShell.layerSurface
	}
	return nil
}

func (s *WaylandSurfaceState) setGeometry(x, y, width, height int32) {
	switch {
	case s.xdg != nil:
		s.xdg.xdgSurface.SetWindowGeometry(x, y, width, height)
	case s.layerShell != nil:
		s.layerShell.layerSurface.SetSize(uint32(width), uint32(height))
	case s.popup != nil:
		s.popup.xdgSurface.SetWindowGeometry(x, y, width, height)
	}
}

func (s *WaylandSurfaceState) repositionPopup(globals *Globals, size gpui.Size[gpui.Pixels], parentGeometry gpui.Bounds[gpui.Pixels]) {
	popup := s.popup
	if popup == nil || popup.xdgPopup.Version() < wlproto.XdgPopupRepositionSinceVersion {
		return
	}
	token := popup.nextRepositionToken
	popup.nextRepositionToken = token + 1

	positioner := buildPopupPositioner(globals, &popup.options, size, parentGeometry)
	popup.xdgPopup.Reposition(positioner, token)
	positioner.Destroy()
}

func (s *WaylandSurfaceState) setExclusiveZone(zone int32) bool {
	if s.layerShell != nil {
		s.layerShell.layerSurface.SetExclusiveZone(zone)
		return true
	}
	return false
}

func applyExclusiveEdge(layerSurface *wlproto.ZwlrLayerSurfaceV1, anchor, edge gpui.LayerShellAnchor) bool {
	if popcount32(uint32(edge)) == 1 && anchor.Contains(edge) {
		layerSurface.SetExclusiveEdge(layerShellWaylandAnchor(edge))
		return true
	}
	slog.Warn(fmt.Sprintf("ignoring exclusive edge %s: must be a single edge of the surface anchor %s", anchorDebug(edge), anchorDebug(anchor)))
	return false
}

func popcount32(v uint32) int {
	count := 0
	for ; v != 0; v &= v - 1 {
		count++
	}
	return count
}

func anchorDebug(anchor gpui.LayerShellAnchor) string {
	names := []struct {
		flag gpui.LayerShellAnchor
		name string
	}{
		{gpui.LayerShellAnchorTop, "TOP"},
		{gpui.LayerShellAnchorBottom, "BOTTOM"},
		{gpui.LayerShellAnchorLeft, "LEFT"},
		{gpui.LayerShellAnchorRight, "RIGHT"},
	}
	out := ""
	rest := anchor
	for _, n := range names {
		if anchor.Contains(n.flag) {
			if out != "" {
				out += " | "
			}
			out += n.name
			rest &^= n.flag
		}
	}
	if rest != 0 {
		if out != "" {
			out += " | "
		}
		out += fmt.Sprintf("0x%x", uint32(rest))
	}
	if out == "" {
		out = "(empty)"
	}
	return "Anchor(" + out + ")"
}

func (s *WaylandSurfaceState) setExclusiveEdge(edge gpui.LayerShellAnchor) bool {
	if s.layerShell != nil {
		return applyExclusiveEdge(s.layerShell.layerSurface, s.layerShell.anchor, edge)
	}
	return false
}

func (s *WaylandSurfaceState) destroy() {
	switch {
	case s.xdg != nil:
		if s.xdg.dialog != nil {
			s.xdg.dialog.Destroy()
		}
		s.xdg.toplevel.Destroy()
		s.xdg.xdgSurface.Destroy()
	case s.layerShell != nil:
		s.layerShell.layerSurface.Destroy()
	case s.popup != nil:
		s.popup.xdgPopup.Destroy()
		s.popup.xdgSurface.Destroy()
	}
}

// WaylandWindowStatePtr is a shared reference to a window's state.
type WaylandWindowStatePtr struct {
	state            *WaylandWindowState
	callbacks        *Callbacks
	frameLoop        *frameLoop
	framePing        *common.Ping
	scheduledFrameAt *gpui.Option[scheduler.Instant]
}

func newWaylandWindowState(handle gpui.AnyWindowHandle, surface *wlproto.WlSurface, surfaceState WaylandSurfaceState, appearance gpui.WindowAppearance, viewport *wlproto.WpViewport, client WaylandClientStatePtr, globals *Globals, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], options gpui.WindowParams, parent gpui.Option[WaylandWindowStatePtr]) (*WaylandWindowState, error) {
	config := wgpu.WgpuSurfaceConfig{
		Size: gpui.Size[gpui.DevicePixels]{
			Width:  gpui.DevicePixels(options.Bounds.Size.Width.ToI32()),
			Height: gpui.DevicePixels(options.Bounds.Size.Height.ToI32()),
		},
		Transparent:          true,
		PreferredPresentMode: gpui.Some(wgpu.PresentModeMailbox),
	}
	renderer, err := wgpu.NewWgpuRenderer(gpuContext, rawWindow(surface, globals), config, compositorGPU)
	if err != nil {
		return nil, err
	}

	if xdgState := surfaceState.xdg; xdgState != nil {
		if titlebar, ok := options.Titlebar.Get(); ok {
			if title, ok := titlebar.Title.Get(); ok {
				xdgState.toplevel.SetTitle(title.String())
			}
		}

		if appID, ok := options.AppID.Get(); ok {
			xdgState.toplevel.SetAppId(appID)
		}

		maxTextureSize := int32(renderer.MaxTextureSize())
		xdgState.toplevel.SetMaxSize(maxTextureSize, maxTextureSize)
	}

	return &WaylandWindowState{
		surfaceState:         surfaceState,
		parent:               parent,
		children:             map[ObjectID]bool{},
		surface:              surface,
		appID:                options.AppID,
		viewport:             viewport,
		globals:              globals,
		outputs:              map[ObjectID]Output{},
		renderer:             renderer,
		bounds:               options.Bounds,
		scale:                1.0,
		decorations:          gpui.WindowDecorationsClient,
		backgroundAppearance: gpui.WindowBackgroundAppearanceOpaque,
		visibility:           gpui.WindowVisibilityVisible,
		windowBounds:         options.Bounds,
		client:               client,
		appearance:           appearance,
		handle:               handle,
		presentation:         presentationStateUnpresented,
		windowControls:       gpui.DefaultWindowControls(),
	}, nil
}

// IsTransparent reports whether the window's surface has transparent pixels.
func (s *WaylandWindowState) IsTransparent() bool {
	return s.decorations == gpui.WindowDecorationsClient ||
		s.backgroundAppearance != gpui.WindowBackgroundAppearanceOpaque
}

func (s *WaylandWindowState) updateSubpixelLayout() {
	isBGR := false
	if entry, ok := s.display.Get(); ok {
		if subpixel, ok := entry.output.Subpixel.Get(); ok {
			isBGR = subpixel == wlproto.WlOutputSubpixelHorizontalBgr
		}
	}
	if s.renderer != nil {
		s.renderer.SetSubpixelLayout(isBGR)
	}
}

// PrimaryOutputScale picks the window's display, the entered output with the largest scale,
// and returns the largest scale.
func (s *WaylandWindowState) PrimaryOutputScale() int32 {
	scale := int32(1)
	currentOutput := s.display
	s.display = gpui.None[outputEntry]()
	if entry, ok := currentOutput.Get(); ok {
		if _, entered := s.outputs[entry.id]; !(len(s.outputs) == 0 || entered) {
			currentOutput = gpui.None[outputEntry]()
		}
	}
	for id, output := range s.outputs {
		if entry, ok := currentOutput.Get(); ok {
			if output.Scale > entry.output.Scale {
				currentOutput = gpui.Some(outputEntry{id: id, output: output})
			}
		} else {
			currentOutput = gpui.Some(outputEntry{id: id, output: output})
		}
		scale = max(scale, output.Scale)
	}
	s.display = currentOutput
	return scale
}

// Inset returns the client-side decoration inset.
func (s *WaylandWindowState) Inset() gpui.Pixels {
	switch s.decorations {
	case gpui.WindowDecorationsServer:
		return gpui.Px(0.0)
	default:
		return s.clientInset.UnwrapOr(gpui.Px(0.0))
	}
}

type presentationState uint8

const (
	presentationStateUnpresented presentationState = iota
	presentationStatePresented
	presentationStateRetryBeforeFirstPresent
	presentationStateRetryAfterPresent
)

func (s presentationState) requiresPresentation() bool {
	return s == presentationStateRetryBeforeFirstPresent || s == presentationStateRetryAfterPresent
}

func (s presentationState) failed() presentationState {
	switch s {
	case presentationStateUnpresented, presentationStateRetryBeforeFirstPresent:
		return presentationStateRetryBeforeFirstPresent
	default:
		return presentationStateRetryAfterPresent
	}
}

type frameLoop uint8

const (
	frameLoopUnconfigured frameLoop = iota
	frameLoopTicking
	frameLoopRescheduleRequested
	frameLoopPresentationFailed
	frameLoopAwaitingCallback
	frameLoopScheduled
	frameLoopRetryScheduled
	frameLoopParked
)

// WaylandWindow is a Wayland window as a gpui.PlatformWindow. Close destroys it.
type WaylandWindow struct {
	gpui.PlatformWindowDefaults
	ptr    WaylandWindowStatePtr
	closed bool
}

var _ gpui.PlatformWindow = (*WaylandWindow)(nil)

// IMEInput is an input method action for a window: IMEInputInsertText, IMEInputSetMarkedText,
// IMEInputUnmarkText or IMEInputDeleteText.
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

// Close destroys the window: its renderer first, then its protocol objects, the wl_surface
// last. It is idempotent.
func (w *WaylandWindow) Close() {
	if w.closed {
		return
	}
	w.closed = true
	*w.ptr.frameLoop = frameLoopParked

	state := w.ptr.state
	surfaceID := ObjectID(&state.surface.Proxy)

	if parent, ok := state.parent.Get(); ok {
		delete(parent.state.children, surfaceID)
	}

	client := state.client

	if state.renderer != nil {
		state.renderer.Destroy()
		state.renderer = nil
	}

	if state.kdeBlur != nil {
		state.kdeBlur.Release()
	}

	if state.backgroundEffects != nil {
		state.backgroundEffects.Destroy()
	}

	if decoration := state.surfaceState.decoration(); decoration != nil {
		decoration.Destroy()
	}

	state.surfaceState.destroy()

	if state.viewport != nil {
		state.viewport.Destroy()
	}

	state.surface.Destroy()

	if state.accesskitAdapter != nil {
		state.accesskitAdapter.Release()
		state.accesskitAdapter = nil
	}

	client.DropWindow(surfaceID)
	statePtr := w.ptr
	state.globals.Executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		statePtr.Close()
		return struct{}{}, nil
	}).Detach()
}

func newWaylandWindow(handle gpui.AnyWindowHandle, globals *Globals, gpuContext *wgpu.GPUContext, compositorGPU gpui.Option[wgpu.CompositorGPUHint], client WaylandClientStatePtr, params gpui.WindowParams, appearance gpui.WindowAppearance, parent gpui.Option[WaylandWindowStatePtr], grab gpui.Option[popupGrab], targetOutput *wlproto.WlOutput) (*WaylandWindow, ObjectID, error) {
	surface := globals.Compositor.CreateSurface()
	surface.SetListener(client.SurfaceListener(surface))
	surfaceID := ObjectID(&surface.Proxy)
	surfaceState, err := newWaylandSurfaceState(surface, globals, client, &params, parent, grab, targetOutput)
	if err != nil {
		return nil, nil, err
	}

	if globals.FractionalScaleManager != nil {
		fractionalScale := globals.FractionalScaleManager.GetFractionalScale(surface)
		fractionalScale.SetListener(client.FractionalScaleListener(surfaceID))
	}

	var viewport *wlproto.WpViewport
	if globals.Viewporter != nil {
		viewport = globals.Viewporter.GetViewport(surface)
	}

	state, err := newWaylandWindowState(handle, surface, surfaceState, appearance, viewport, client, globals, gpuContext, compositorGPU, params, parent)
	if err != nil {
		return nil, nil, err
	}
	frameLoopCell := frameLoopUnconfigured
	scheduledFrameAt := gpui.None[scheduler.Instant]()
	w := &WaylandWindow{ptr: WaylandWindowStatePtr{
		state:            state,
		callbacks:        &Callbacks{},
		frameLoop:        &frameLoopCell,
		framePing:        globals.FramePing,
		scheduledFrameAt: &scheduledFrameAt,
	}}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)

	surface.Commit()

	return w, surfaceID, nil
}

// Handle returns the window's handle.
func (p WaylandWindowStatePtr) Handle() gpui.AnyWindowHandle {
	return p.state.handle
}

// Surface returns the window's wl_surface.
func (p WaylandWindowStatePtr) Surface() *wlproto.WlSurface {
	return p.state.surface
}

// Toplevel returns the window's xdg_toplevel, or nil for other surface kinds.
func (p WaylandWindowStatePtr) Toplevel() *wlproto.XdgToplevel {
	return p.state.surfaceState.toplevel()
}

// XdgSurface returns the xdg_surface backing the window, or nil; child popups anchor to it.
func (p WaylandWindowStatePtr) XdgSurface() *wlproto.XdgSurface {
	return p.state.surfaceState.xdgSurface()
}

// LayerSurface returns the layer-shell surface backing the window, or nil; child popups anchor
// to it.
func (p WaylandWindowStatePtr) LayerSurface() *wlproto.ZwlrLayerSurfaceV1 {
	return p.state.surfaceState.layerSurface()
}

// WindowGeometry returns the window's xdg window geometry in surface-local coordinates.
func (p WaylandWindowStatePtr) WindowGeometry() gpui.Bounds[gpui.Pixels] {
	state := p.state
	return insetByTiling(state.bounds.MapOrigin(zeroPixels), state.Inset(), state.tiling)
}

func zeroPixels(gpui.Pixels) gpui.Pixels { return gpui.Px(0.0) }

// PtrEq reports whether both pointers refer to the same window.
func (p WaylandWindowStatePtr) PtrEq(other WaylandWindowStatePtr) bool {
	return p.state == other.state
}

// AddChild records a child surface and whether it blocks this window's input.
func (p WaylandWindowStatePtr) AddChild(child ObjectID, blocking bool) {
	p.state.children[child] = blocking
}

// IsBlocked reports whether a modal child blocks the window.
func (p WaylandWindowStatePtr) IsBlocked() bool {
	for _, blocking := range p.state.children {
		if blocking {
			return true
		}
	}
	return false
}

// Frame runs one frame of the window's render loop.
func (p WaylandWindowStatePtr) Frame(signalAt gpui.Option[scheduler.Instant], signalSource gpui.FrameRequestSource) {
	*p.frameLoop = frameLoopTicking
	state := p.state
	state.resizeThrottle = false
	forceRender := state.redrawRequested
	requirePresentation := state.presentation.requiresPresentation()

	requestFrameCallback := p.callbacks.requestFrame
	if requestFrameCallback == nil {
		*p.frameLoop = frameLoopParked
		return
	}
	requestFrameCallback(gpui.RequestFrameOptions{
		ForceRender:         forceRender,
		RequirePresentation: requirePresentation,
		SignalAt:            signalAt,
		SignalSource:        signalSource,
	})
	p.updateIMEEnabled()

	p.completeFrame()
}

func (p WaylandWindowStatePtr) requestFrameCallback() {
	state := p.state
	callback := state.surface.Frame()
	callback.SetListener(state.client.FrameCallbackListener(&state.surface.Proxy))
	state.pendingFrameCallback = callback
}

func (p WaylandWindowStatePtr) completeFrame() {
	state := p.state

	loop := *p.frameLoop
	if loop == frameLoopAwaitingCallback {
		return
	}

	if state.presentation.requiresPresentation() {
		if loop == frameLoopPresentationFailed && state.presentation == presentationStateRetryAfterPresent {
			if state.pendingFrameCallback == nil {
				p.requestFrameCallback()
			}
			state.surface.Commit()
			*p.frameLoop = frameLoopAwaitingCallback
			return
		}

		*p.frameLoop = frameLoopRetryScheduled
		state.client.ScheduleFrameRetry(&state.surface.Proxy)
		return
	}

	if loop == frameLoopRescheduleRequested || state.redrawRequested {
		*p.frameLoop = frameLoopRetryScheduled
		state.client.ScheduleFrameRetry(&state.surface.Proxy)
		return
	}

	*p.frameLoop = frameLoopParked
}

// FrameCallbackFired handles a wl_callback.done of the surface's frame callback.
func (p WaylandWindowStatePtr) FrameCallbackFired(signalAt gpui.Option[scheduler.Instant]) {
	p.state.pendingFrameCallback = nil
	if *p.frameLoop == frameLoopAwaitingCallback {
		p.Frame(signalAt, gpui.FrameRequestSourceNativeCallback)
	}
}

// ScheduledFrameFired handles a tick of the frame ping.
func (p WaylandWindowStatePtr) ScheduledFrameFired() {
	if *p.frameLoop == frameLoopScheduled {
		signalAt := *p.scheduledFrameAt
		*p.scheduledFrameAt = gpui.None[scheduler.Instant]()
		p.Frame(signalAt, gpui.FrameRequestSourceLocalSchedule)
	}
}

// RetryTimerFired handles a frame retry tick that was due at deadline.
func (p WaylandWindowStatePtr) RetryTimerFired(deadline time.Time) {
	if *p.frameLoop == frameLoopRetryScheduled {
		p.Frame(gpui.OptionFrom(gpui.PlatformFrameSignalCapture(func() scheduler.Instant {
			return scheduler.InstantNow().Add(-time.Since(deadline))
		})), gpui.FrameRequestSourceLocalSchedule)
	}
}

// IsConfigured reports whether the window received its first configure.
func (p WaylandWindowStatePtr) IsConfigured() bool {
	return *p.frameLoop != frameLoopUnconfigured
}

// ScheduleFrame arms a wake-up for the next frame unless one is already armed.
func (p WaylandWindowStatePtr) ScheduleFrame() {
	switch *p.frameLoop {
	case frameLoopParked:
		*p.frameLoop = frameLoopScheduled
		*p.scheduledFrameAt = gpui.OptionFrom(gpui.PlatformFrameSignalCapture(scheduler.InstantNow))
		p.framePing.Ping()
	case frameLoopTicking:
		*p.frameLoop = frameLoopRescheduleRequested
	}
}

func (p WaylandWindowStatePtr) requestRedraw() {
	p.state.redrawRequested = true
	p.ScheduleFrame()
}

func (p WaylandWindowStatePtr) updateIMEEnabled() {
	state := p.state
	if !state.active {
		return
	}
	client := state.client
	imeEnabled := false
	if inputHandler := state.inputHandler; inputHandler != nil {
		state.inputHandler = nil
		acceptsTextInput := inputHandler.QueryAcceptsTextInput()
		p.state.inputHandler = inputHandler
		imeEnabled = acceptsTextInput
	}
	if enabled, ok := client.IMEEnabled().Get(); ok && enabled == imeEnabled {
		return
	}

	if imeEnabled {
		client.EnableIME()
	} else {
		client.DisableIME()
	}
}

// HandleXdgSurfaceEvent handles an xdg_surface event.
func (p WaylandWindowStatePtr) HandleXdgSurfaceEvent(event XdgSurfaceEvent) {
	configureEvent, ok := event.(XdgSurfaceEventConfigure)
	if !ok {
		return
	}
	serial := configureEvent.Serial
	state := p.state
	if windowControls, ok := state.inProgressWindowControls.Get(); ok {
		state.inProgressWindowControls = gpui.None[gpui.WindowControls]()
		state.windowControls = windowControls

		if appearanceChanged := p.callbacks.appearanceChanged; appearanceChanged != nil {
			appearanceChanged()
		}
	}
	if configure, ok := state.inProgressConfigure.Get(); ok {
		state.inProgressConfigure = gpui.None[inProgressConfigure]()
		gotUnmaximized := state.maximized && !configure.maximized
		fullscreenChanged := state.fullscreen != configure.fullscreen
		modeChanged := fullscreenChanged || state.maximized != configure.maximized
		state.fullscreen = configure.fullscreen
		state.maximized = configure.maximized
		state.tiling = configure.tiling
		visibilityChanged := state.visibility != configure.visibility
		state.visibility = configure.visibility
		throttled := configure.resizing && state.resizeThrottle
		if throttled {
			state.surfaceState.ackConfigure(serial)
		} else {
			if configure.resizing {
				state.resizeThrottle = true
			}
			if !configure.fullscreen && !configure.maximized {
				if gotUnmaximized {
					configure.size = gpui.Some(state.windowBounds.Size)
				} else {
					configure.size = computeOuterSize(state.Inset(), configure.size, state.tiling)
				}
				if size, ok := configure.size.Get(); ok {
					state.windowBounds = gpui.NewBounds(gpui.Point[gpui.Pixels]{}, size)
				}
			}
		}
		size, hasSize := configure.size.Get()
		resized := hasSize && size != state.bounds.Size
		if visibilityChanged {
			p.reportVisibility(configure.visibility)
		}
		if !throttled && hasSize {
			p.Resize(size)
		}
		if fullscreenChanged {
			p.notifyResize()
		}
		if throttled {
			return
		}
		if modeChanged && !resized && !fullscreenChanged {
			if fun := p.callbacks.moved; fun != nil {
				p.callbacks.moved = nil
				fun()
				p.callbacks.moved = fun
			}
		}
	}
	state.surfaceState.ackConfigure(serial)

	windowGeometry := insetByTiling(state.bounds.MapOrigin(zeroPixels), state.Inset(), state.tiling)
	geometry := windowGeometry.Map(pixelsToI32).MapSize(atLeastOne)

	state.surfaceState.setGeometry(geometry.Origin.X, geometry.Origin.Y, geometry.Size.Width, geometry.Size.Height)

	initialConfigure := *p.frameLoop == frameLoopUnconfigured
	updateWindow(state)
	if initialConfigure {
		p.Frame(gpui.None[scheduler.Instant](), gpui.FrameRequestSourceNativeCallback)
	} else {
		p.requestRedraw()
	}
}

func pixelsToI32(v gpui.Pixels) int32 { return v.ToI32() }

func atLeastOne(v int32) int32 {
	if v <= 0 {
		return 1
	}
	return v
}

func (p WaylandWindowStatePtr) callAppearanceChanged() {
	if fun := p.callbacks.appearanceChanged; fun != nil {
		p.callbacks.appearanceChanged = nil
		fun()
		p.callbacks.appearanceChanged = fun
	}
}

// HandleToplevelDecorationEvent handles a zxdg_toplevel_decoration_v1 event.
func (p WaylandWindowStatePtr) HandleToplevelDecorationEvent(event ZxdgToplevelDecorationV1Event) {
	configure, ok := event.(ZxdgToplevelDecorationV1EventConfigure)
	if !ok {
		return
	}
	switch configure.Mode {
	case wlproto.ZxdgToplevelDecorationV1ModeServerSide:
		p.state.decorations = gpui.WindowDecorationsServer
		p.callAppearanceChanged()
	case wlproto.ZxdgToplevelDecorationV1ModeClientSide:
		p.state.decorations = gpui.WindowDecorationsClient
		p.callAppearanceChanged()
	default:
		slog.Warn(fmt.Sprintf("Unknown decoration mode: %d", uint32(configure.Mode)))
		return
	}
	updateWindow(p.state)
	p.requestRedraw()
}

// HandleFractionalScaleEvent handles a wp_fractional_scale_v1 event.
func (p WaylandWindowStatePtr) HandleFractionalScaleEvent(event WpFractionalScaleV1Event) {
	if preferred, ok := event.(WpFractionalScaleV1EventPreferredScale); ok {
		p.Rescale(float32(preferred.Scale) / 120.0)
		p.requestRedraw()
	}
}

func extractStates(states []byte) []uint32 {
	out := make([]uint32, 0, len(states)/4)
	for i := 0; i+4 <= len(states); i += 4 {
		out = append(out, binary.NativeEndian.Uint32(states[i:i+4]))
	}
	return out
}

func eventSize(width, height int32) gpui.Option[gpui.Size[gpui.Pixels]] {
	if width == 0 || height == 0 {
		return gpui.None[gpui.Size[gpui.Pixels]]()
	}
	return gpui.Some(gpui.NewSize(gpui.Px(float32(width)), gpui.Px(float32(height))))
}

// HandleToplevelEvent handles an xdg_toplevel event and reports whether the window should close.
func (p WaylandWindowStatePtr) HandleToplevelEvent(event XdgToplevelEvent) bool {
	switch event := event.(type) {
	case XdgToplevelEventConfigure:
		size := eventSize(event.Width, event.Height)

		tiling := gpui.Tiling{}
		fullscreen := false
		maximized := false
		resizing := false
		visibility := gpui.WindowVisibilityVisible

		for _, state := range extractStates(event.States) {
			switch wlproto.XdgToplevelState(state) {
			case wlproto.XdgToplevelStateMaximized:
				maximized = true
			case wlproto.XdgToplevelStateFullscreen:
				fullscreen = true
			case wlproto.XdgToplevelStateResizing:
				resizing = true
			case wlproto.XdgToplevelStateSuspended:
				visibility = gpui.WindowVisibilityHidden
			case wlproto.XdgToplevelStateTiledTop:
				tiling.Top = true
			case wlproto.XdgToplevelStateTiledLeft:
				tiling.Left = true
			case wlproto.XdgToplevelStateTiledRight:
				tiling.Right = true
			case wlproto.XdgToplevelStateTiledBottom:
				tiling.Bottom = true
			}
		}

		if fullscreen || maximized {
			tiling = gpui.TilingTiled()
		}

		p.state.inProgressConfigure = gpui.Some(inProgressConfigure{
			size:       size,
			fullscreen: fullscreen,
			maximized:  maximized,
			resizing:   resizing,
			visibility: visibility,
			tiling:     tiling,
		})

		return false
	case XdgToplevelEventClose:
		shouldClose := p.callbacks.shouldClose
		if shouldClose == nil {
			return true
		}
		p.callbacks.shouldClose = nil
		result := shouldClose()
		p.callbacks.shouldClose = shouldClose
		if result {
			p.Close()
		}
		return result
	case XdgToplevelEventWmCapabilities:
		windowControls := gpui.WindowControls{}

		for _, capability := range extractStates(event.Capabilities) {
			switch wlproto.XdgToplevelWmCapabilities(capability) {
			case wlproto.XdgToplevelWmCapabilitiesMaximize:
				windowControls.Maximize = true
			case wlproto.XdgToplevelWmCapabilitiesMinimize:
				windowControls.Minimize = true
			case wlproto.XdgToplevelWmCapabilitiesFullscreen:
				windowControls.Fullscreen = true
			case wlproto.XdgToplevelWmCapabilitiesWindowMenu:
				windowControls.WindowMenu = true
			}
		}

		p.state.inProgressWindowControls = gpui.Some(windowControls)
		return false
	default:
		return false
	}
}

func plainConfigure(size gpui.Option[gpui.Size[gpui.Pixels]]) inProgressConfigure {
	return inProgressConfigure{
		size:       size,
		visibility: gpui.WindowVisibilityVisible,
	}
}

// HandleLayersurfaceEvent handles a zwlr_layer_surface_v1 event and reports whether the window
// should close.
func (p WaylandWindowStatePtr) HandleLayersurfaceEvent(event ZwlrLayerSurfaceV1Event) bool {
	switch event := event.(type) {
	case ZwlrLayerSurfaceV1EventConfigure:
		size := gpui.None[gpui.Size[gpui.Pixels]]()
		if event.Width != 0 && event.Height != 0 {
			size = gpui.Some(gpui.NewSize(gpui.Px(float32(event.Width)), gpui.Px(float32(event.Height))))
		}

		p.state.inProgressConfigure = gpui.Some(plainConfigure(size))

		p.HandleXdgSurfaceEvent(XdgSurfaceEventConfigure{Serial: event.Serial})

		return false
	case ZwlrLayerSurfaceV1EventClosed:
		return true
	default:
		return false
	}
}

// HandlePopupEvent handles an xdg_popup event and reports whether the popup should close.
func (p WaylandWindowStatePtr) HandlePopupEvent(event XdgPopupEvent) bool {
	switch event := event.(type) {
	case XdgPopupEventConfigure:
		size := gpui.None[gpui.Size[gpui.Pixels]]()
		if event.Width > 0 && event.Height > 0 {
			size = gpui.Some(gpui.NewSize(gpui.Px(float32(event.Width)), gpui.Px(float32(event.Height))))
		}

		p.state.inProgressConfigure = gpui.Some(plainConfigure(size))

		return false
	case XdgPopupEventPopupDone:
		return true
	default:
		return false
	}
}

func displayID(display gpui.Option[outputEntry]) ObjectID {
	if entry, ok := display.Get(); ok {
		return entry.id
	}
	return nil
}

// HandleSurfaceEvent handles a wl_surface event.
func (p WaylandWindowStatePtr) HandleSurfaceEvent(event WlSurfaceEvent, outputs map[ObjectID]Output) {
	state := p.state

	switch event := event.(type) {
	case WlSurfaceEventEnter:
		id := ObjectID(&event.Output.Proxy)

		output, ok := outputs[id]
		if !ok {
			return
		}

		state.outputs[id] = output
		p.outputsChanged()
	case WlSurfaceEventLeave:
		delete(state.outputs, ObjectID(&event.Output.Proxy))
		p.outputsChanged()
	case WlSurfaceEventPreferredBufferScale:
		if state.globals.FractionalScaleManager == nil {
			state.surface.SetBufferScale(event.Factor)
			p.Rescale(float32(event.Factor))
			p.requestRedraw()
		}
	}
}

func (p WaylandWindowStatePtr) outputsChanged() {
	state := p.state
	previousDisplay := displayID(state.display)
	scale := state.PrimaryOutputScale()
	displayChanged := displayID(state.display) != previousDisplay
	state.updateSubpixelLayout()

	if state.surface.Version() < wlproto.WlSurfacePreferredBufferScaleSinceVersion {
		state.surface.SetBufferScale(scale)
		p.Rescale(float32(scale))
	}
	if displayChanged {
		p.reportDisplayChanged()
	}
	p.requestRedraw()
}

// HandleIME applies an input method action.
func (p WaylandWindowStatePtr) HandleIME(ime IMEInput) {
	if p.IsBlocked() {
		return
	}
	state := p.state
	inputHandler := state.inputHandler
	if inputHandler == nil {
		return
	}
	state.inputHandler = nil
	switch ime := ime.(type) {
	case IMEInputInsertText:
		inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), ime.Text)
	case IMEInputSetMarkedText:
		inputHandler.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), ime.Text, gpui.None[gpui.Range]())
	case IMEInputUnmarkText:
		inputHandler.UnmarkText()
	case IMEInputDeleteText:
		if marked, ok := inputHandler.MarkedTextRange(); ok {
			inputHandler.ReplaceTextInRange(gpui.Some(marked), "")
		}
	}
	p.state.inputHandler = inputHandler
}

// GetIMEArea returns the IME candidate bounds.
func (p WaylandWindowStatePtr) GetIMEArea() (gpui.Bounds[gpui.Pixels], bool) {
	state := p.state
	inputHandler := state.inputHandler
	if inputHandler == nil {
		return gpui.Bounds[gpui.Pixels]{}, false
	}
	state.inputHandler = nil
	bounds, ok := inputHandler.IMECandidateBounds()
	p.state.inputHandler = inputHandler
	return bounds, ok
}

// SetSizeAndScale updates the window's size and scale and reports the change.
func (p WaylandWindowStatePtr) SetSizeAndScale(size gpui.Option[gpui.Size[gpui.Pixels]], scale gpui.Option[float32]) {
	state := p.state
	newSize, hasSize := size.Get()
	newScale, hasScale := scale.Get()
	if (!hasSize || newSize == state.bounds.Size) && (!hasScale || newScale == state.scale) {
		return
	}
	if hasSize {
		state.bounds.Size = newSize
	}
	if hasScale {
		state.scale = newScale
	}
	deviceBounds := state.bounds.ToDevicePixels(state.scale)
	if state.renderer != nil {
		state.renderer.UpdateDrawableSize(deviceBounds.Size)
	}
	reportedSize, reportedScale := state.bounds.Size, state.scale

	if fun := p.callbacks.resize; fun != nil {
		p.callbacks.resize = nil
		fun(reportedSize, reportedScale)
		p.callbacks.resize = fun
	}

	if state.viewport != nil {
		state.viewport.SetDestination(reportedSize.Width.ToI32(), reportedSize.Height.ToI32())
	}
	updateWindow(state)
}

func (p WaylandWindowStatePtr) notifyResize() {
	size, scale := p.state.bounds.Size, p.state.scale
	if fun := p.callbacks.resize; fun != nil {
		p.callbacks.resize = nil
		fun(size, scale)
		p.callbacks.resize = fun
	}
}

// Resize sets the window's size.
func (p WaylandWindowStatePtr) Resize(size gpui.Size[gpui.Pixels]) {
	p.SetSizeAndScale(gpui.Some(size), gpui.None[float32]())
}

// Rescale sets the window's scale.
func (p WaylandWindowStatePtr) Rescale(scale float32) {
	p.SetSizeAndScale(gpui.None[gpui.Size[gpui.Pixels]](), gpui.Some(scale))
}

// Close closes the window's children and runs its close callback.
func (p WaylandWindowStatePtr) Close() {
	state := p.state
	client, hasClient := state.client.TryGetClient()
	children := make([]ObjectID, 0, len(state.children))
	for child := range state.children {
		children = append(children, child)
	}

	if hasClient {
		for _, child := range children {
			if window, ok := getWindow(client, child); ok {
				window.Close()
			}
		}
	}
	if fun := p.callbacks.close; fun != nil {
		p.callbacks.close = nil
		fun()
	}
}

// HandleInput dispatches a platform input event to the window.
func (p WaylandWindowStatePtr) HandleInput(input gpui.PlatformInput) {
	if p.IsBlocked() {
		return
	}
	if fun := p.callbacks.input; fun != nil {
		p.callbacks.input = nil
		result := fun(input)
		p.callbacks.input = fun
		if !result.Propagate {
			return
		}
	}
	event, ok := input.(gpui.KeyDownEvent)
	if !ok || !event.Keystroke.Modifiers.IsSubsetOf(gpui.ModifiersShift()) {
		return
	}
	keyChar, ok := event.Keystroke.KeyChar.Get()
	if !ok {
		return
	}
	state := p.state
	if inputHandler := state.inputHandler; inputHandler != nil {
		state.inputHandler = nil
		inputHandler.ReplaceTextInRange(gpui.None[gpui.Range](), keyChar)
		p.state.inputHandler = inputHandler
	}
}

// SetFocused sets whether the window has keyboard focus.
func (p WaylandWindowStatePtr) SetFocused(focus bool) {
	p.state.active = focus
	if fun := p.callbacks.activeStatusChange; fun != nil {
		p.callbacks.activeStatusChange = nil
		fun(focus)
		p.callbacks.activeStatusChange = fun
	}
	if adapter := p.state.accesskitAdapter; adapter != nil {
		adapter.UpdateWindowFocusState(focus)
	}
}

// SetHovered reports whether the pointer is over the window.
func (p WaylandWindowStatePtr) SetHovered(focus bool) {
	if fun := p.callbacks.hoverStatusChange; fun != nil {
		p.callbacks.hoverStatusChange = nil
		fun(focus)
		p.callbacks.hoverStatusChange = fun
	}
}

func (p WaylandWindowStatePtr) reportDisplayChanged() {
	if fun := p.callbacks.displayChanged; fun != nil {
		p.callbacks.displayChanged = nil
		fun()
		p.callbacks.displayChanged = fun
	}
}

// HandleOutputChanged updates the window's copy of an output it is on after the compositor
// changes the output's properties.
func (p WaylandWindowStatePtr) HandleOutputChanged(id ObjectID, output Output) {
	state := p.state
	if _, ok := state.outputs[id]; !ok {
		return
	}
	state.outputs[id] = output
	if entry, ok := state.display.Get(); ok && entry.id == id {
		state.display = gpui.Some(outputEntry{id: id, output: output})
	}
}

func (p WaylandWindowStatePtr) reportVisibility(visibility gpui.WindowVisibility) {
	if fun := p.callbacks.visibilityChange; fun != nil {
		p.callbacks.visibilityChange = nil
		fun(visibility)
		p.callbacks.visibilityChange = fun
	}
}

// SetAppearance sets the window's appearance.
func (p WaylandWindowStatePtr) SetAppearance(appearance gpui.WindowAppearance) {
	p.state.appearance = appearance
	p.callAppearanceChanged()
}

// SetButtonLayout reports a changed button layout.
func (p WaylandWindowStatePtr) SetButtonLayout() {
	if fun := p.callbacks.buttonLayoutChanged; fun != nil {
		p.callbacks.buttonLayoutChanged = nil
		fun()
		p.callbacks.buttonLayoutChanged = fun
	}
}

// PrimaryOutputScale returns the integer scale of the window's primary output.
func (p WaylandWindowStatePtr) PrimaryOutputScale() int32 {
	return p.state.PrimaryOutputScale()
}

// RawWindowHandle returns the wl_display and wl_surface pair.
func (w *WaylandWindow) RawWindowHandle() gpui.RawWindowHandle {
	return rawWindow(w.ptr.state.surface, w.ptr.state.globals)
}

// Bounds returns the window's bounds.
func (w *WaylandWindow) Bounds() gpui.Bounds[gpui.Pixels] {
	return w.ptr.state.bounds
}

// IsMaximized reports whether the window is maximized.
func (w *WaylandWindow) IsMaximized() bool {
	return w.ptr.state.maximized
}

// WindowBounds returns the bounds to restore the window with.
func (w *WaylandWindow) WindowBounds() gpui.WindowBounds {
	state := w.ptr.state
	switch {
	case state.fullscreen:
		return gpui.WindowBoundsFullscreen(state.windowBounds)
	case state.maximized:
		return gpui.WindowBoundsMaximized(state.windowBounds)
	default:
		return gpui.WindowBoundsWindowed(w.Bounds())
	}
}

// InnerWindowBounds returns the window bounds without the client-side decoration inset.
func (w *WaylandWindow) InnerWindowBounds() gpui.WindowBounds {
	state := w.ptr.state
	switch {
	case state.fullscreen:
		return gpui.WindowBoundsFullscreen(state.windowBounds)
	case state.maximized:
		return gpui.WindowBoundsMaximized(state.windowBounds)
	default:
		inset := state.Inset()
		return gpui.WindowBoundsWindowed(w.Bounds().Inset(inset))
	}
}

// ContentSize returns the window's size.
func (w *WaylandWindow) ContentSize() gpui.Size[gpui.Pixels] {
	return w.ptr.state.bounds.Size
}

// Resize asks for a new window size.
func (w *WaylandWindow) Resize(size gpui.Size[gpui.Pixels]) {
	state := w.ptr.state
	statePtr := w.ptr

	if state.surfaceState.popup != nil {
		if statePtr.IsConfigured() {
			parentGeometry := gpui.Bounds[gpui.Pixels]{}
			if parent, ok := state.parent.Get(); ok {
				parentGeometry = parent.WindowGeometry()
			}
			state.surfaceState.repositionPopup(state.globals, size, parentGeometry)
		}
		return
	}

	windowGeometry := insetByTiling(gpui.NewBounds(gpui.Point[gpui.Pixels]{}, size), state.Inset(), state.tiling)
	geometry := windowGeometry.Map(pixelsToI32).MapSize(atLeastOne)

	state.surfaceState.setGeometry(geometry.Origin.X, geometry.Origin.Y, geometry.Size.Width, geometry.Size.Height)

	state.globals.Executor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		statePtr.Resize(size)
		return struct{}{}, nil
	}).Detach()
}

// ScaleFactor returns the window's scale.
func (w *WaylandWindow) ScaleFactor() float32 {
	return w.ptr.state.scale
}

// Appearance returns the window's appearance.
func (w *WaylandWindow) Appearance() gpui.WindowAppearance {
	return w.ptr.state.appearance
}

// Display returns the output the window is on, or the sole output before the compositor names
// one.
func (w *WaylandWindow) Display() gpui.PlatformDisplay {
	state := w.ptr.state
	entry, ok := state.display.Get()
	if !ok {
		entry.id, entry.output, ok = state.client.SoleOutput()
		if !ok {
			return nil
		}
	}
	client, ok := state.client.TryGetClient()
	if !ok {
		return nil
	}
	output, ok := client.wlOutputs[entry.id]
	if !ok {
		return nil
	}
	return newWaylandDisplay(output, entry.output.Name, entry.output.Bounds.ToPixels(state.scale), entry.output.RefreshInterval)
}

// MousePosition returns the pointer position in the focused window.
func (w *WaylandWindow) MousePosition() gpui.Point[gpui.Pixels] {
	return w.ptr.state.client.GetClient().MouseLocation.UnwrapOrDefault()
}

// Modifiers returns the current keyboard modifiers.
func (w *WaylandWindow) Modifiers() gpui.Modifiers {
	return w.ptr.state.client.GetClient().Modifiers
}

// Capslock returns the current caps lock state.
func (w *WaylandWindow) Capslock() gpui.Capslock {
	return w.ptr.state.client.GetClient().Capslock
}

// SetInputHandler sets the window's input handler.
func (w *WaylandWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.ptr.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the window's input handler.
func (w *WaylandWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	inputHandler := w.ptr.state.inputHandler
	w.ptr.state.inputHandler = nil
	return inputHandler
}

// Prompt returns nil: Wayland has no native prompts.
func (w *WaylandWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	return nil
}

// Activate asks the compositor to focus the window through xdg-activation.
func (w *WaylandWindow) Activate() {
	state := w.ptr.state
	appID, ok := state.appID.Get()
	if state.globals.Activation == nil || !ok {
		return
	}
	token := state.globals.Activation.GetActivationToken()
	token.SetListener(state.client.activationTokenListener(token, PendingActivationWindow{Window: &state.surface.Proxy}))
	token.SetAppId(appID)
	if context, ok := state.client.ActivationContext(); ok {
		token.SetSerial(context.Serial.AsRaw(), state.globals.Seat)
		token.SetSurface(context.RequestingSurface)
	} else {
		token.SetSurface(state.surface)
	}
	token.Commit()
}

// RequestAttention does nothing on Wayland.
func (w *WaylandWindow) RequestAttention() {}

// IsActive reports whether the window has keyboard focus.
func (w *WaylandWindow) IsActive() bool {
	return w.ptr.state.active
}

// Visibility returns the window's visibility.
func (w *WaylandWindow) Visibility() gpui.WindowVisibility {
	return w.ptr.state.visibility
}

// IsHovered reports whether the pointer is over the window.
func (w *WaylandWindow) IsHovered() bool {
	return w.ptr.state.hovered
}

// SetTitle sets the toplevel's title.
func (w *WaylandWindow) SetTitle(title string) {
	if toplevel := w.ptr.state.surfaceState.toplevel(); toplevel != nil {
		toplevel.SetTitle(title)
	}
}

// SetAppID sets the toplevel's app id.
func (w *WaylandWindow) SetAppID(appID string) {
	state := w.ptr.state
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		toplevel.SetAppId(appID)
	}
	state.appID = gpui.Some(appID)
}

// SetBackgroundAppearance sets the window's background appearance.
func (w *WaylandWindow) SetBackgroundAppearance(backgroundAppearance gpui.WindowBackgroundAppearance) {
	state := w.ptr.state
	if state.backgroundAppearance == backgroundAppearance {
		return
	}
	state.backgroundAppearance = backgroundAppearance
	updateWindow(state)
	w.ptr.requestRedraw()
}

// BackgroundAppearance returns the window's background appearance.
func (w *WaylandWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return w.ptr.state.backgroundAppearance
}

// IsSubpixelRenderingSupported reports whether the GPU supports dual-source blending.
func (w *WaylandWindow) IsSubpixelRenderingSupported() bool {
	client := w.ptr.state.client.GetClient()
	return client.GPUContext != nil && client.GPUContext.Context != nil && client.GPUContext.Context.SupportsDualSourceBlending()
}

// Minimize minimizes the toplevel.
func (w *WaylandWindow) Minimize() {
	if toplevel := w.ptr.state.surfaceState.toplevel(); toplevel != nil {
		toplevel.SetMinimized()
	}
}

// Zoom toggles the toplevel's maximized state.
func (w *WaylandWindow) Zoom() {
	state := w.ptr.state
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		if !state.maximized {
			toplevel.SetMaximized()
		} else {
			toplevel.UnsetMaximized()
		}
	}
}

// ToggleFullscreen toggles the toplevel's fullscreen state.
func (w *WaylandWindow) ToggleFullscreen() {
	state := w.ptr.state
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		if !state.fullscreen {
			toplevel.SetFullscreen(nil)
		} else {
			toplevel.UnsetFullscreen()
		}
	}
}

// IsFullscreen reports whether the window is fullscreen.
func (w *WaylandWindow) IsFullscreen() bool {
	return w.ptr.state.fullscreen
}

// OnRequestFrame sets the frame callback.
func (w *WaylandWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.ptr.callbacks.requestFrame = callback
}

// OnInput sets the input callback.
func (w *WaylandWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.ptr.callbacks.input = callback
}

// OnActiveStatusChange sets the focus callback.
func (w *WaylandWindow) OnActiveStatusChange(callback func(active bool)) {
	w.ptr.callbacks.activeStatusChange = callback
}

// OnVisibilityChange sets the visibility callback.
func (w *WaylandWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.ptr.callbacks.visibilityChange = callback
}

// OnHoverStatusChange sets the hover callback.
func (w *WaylandWindow) OnHoverStatusChange(callback func(hovered bool)) {
	w.ptr.callbacks.hoverStatusChange = callback
}

// OnResize sets the resize callback.
func (w *WaylandWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.ptr.callbacks.resize = callback
}

// OnMoved sets the move callback.
func (w *WaylandWindow) OnMoved(callback func()) {
	w.ptr.callbacks.moved = callback
}

// OnDisplayChanged sets the display change callback.
func (w *WaylandWindow) OnDisplayChanged(callback func()) {
	w.ptr.callbacks.displayChanged = callback
}

// OnShouldClose sets the close request callback.
func (w *WaylandWindow) OnShouldClose(callback func() bool) {
	w.ptr.callbacks.shouldClose = callback
}

// OnClose sets the close callback.
func (w *WaylandWindow) OnClose(callback func()) {
	w.ptr.callbacks.close = callback
}

// OnHitTestWindowControl does nothing on Wayland.
func (w *WaylandWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {}

// OnAppearanceChanged sets the appearance callback.
func (w *WaylandWindow) OnAppearanceChanged(callback func()) {
	w.ptr.callbacks.appearanceChanged = callback
}

// OnButtonLayoutChanged sets the button layout callback.
func (w *WaylandWindow) OnButtonLayoutChanged(callback func()) {
	w.ptr.callbacks.buttonLayoutChanged = callback
}

// Draw renders scene, recovering the GPU device first when it was lost.
func (w *WaylandWindow) Draw(scene *gpui.Scene) {
	state := w.ptr.state
	renderer := state.renderer
	if renderer == nil {
		return
	}

	if renderer.DeviceLost() {
		if err := renderer.Recover(rawWindow(state.surface, state.globals)); err != nil {
			slog.Warn(fmt.Sprintf("GPU recovery failed, will retry on next frame: %v", err))
		}

		state.redrawRequested = true
		return
	}

	state.redrawRequested = false
	if state.pendingFrameCallback == nil {
		w.ptr.requestFrameCallback()
	}
	if renderer.Draw(scene) {
		state.presentation = presentationStatePresented
		*w.ptr.frameLoop = frameLoopAwaitingCallback
	} else {
		state.presentation = state.presentation.failed()
		*w.ptr.frameLoop = frameLoopPresentationFailed
	}

	if renderer.NeedsRedraw() {
		state.redrawRequested = true
	}
}

// ScheduleFrame arms a wake-up for the next frame.
func (w *WaylandWindow) ScheduleFrame() {
	w.ptr.ScheduleFrame()
}

// SpriteAtlas returns the renderer's atlas.
func (w *WaylandWindow) SpriteAtlas() gpui.PlatformAtlas {
	renderer := w.ptr.state.renderer
	if renderer == nil {
		panic("the renderer is only taken when the window is dropped")
	}
	return renderer.SpriteAtlas()
}

// ShowWindowMenu opens the compositor's window menu at position.
func (w *WaylandWindow) ShowWindowMenu(position gpui.Point[gpui.Pixels]) {
	state := w.ptr.state
	serial := state.client.GetSerial(SerialKindMousePress)
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		toplevel.ShowWindowMenu(state.globals.Seat, serial.AsRaw(), position.X.ToI32(), position.Y.ToI32())
	}
}

// StartWindowMove starts an interactive move.
func (w *WaylandWindow) StartWindowMove() {
	state := w.ptr.state
	serial := state.client.GetSerial(SerialKindMousePress)
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		toplevel.Move(state.globals.Seat, serial.AsRaw())
	}
}

// CanStartExternalDrag reports true.
func (w *WaylandWindow) CanStartExternalDrag() bool {
	return true
}

// StartExternalDrag starts an outbound drag of payload.
func (w *WaylandWindow) StartExternalDrag(payload gpui.ExternalDragPayload) bool {
	state := w.ptr.state
	return state.client.StartExternalDrag(state.surface, payload)
}

// StartWindowResize starts an interactive resize from edge.
func (w *WaylandWindow) StartWindowResize(edge gpui.ResizeEdge) {
	state := w.ptr.state
	if toplevel := state.surfaceState.toplevel(); toplevel != nil {
		toplevel.Resize(state.globals.Seat, state.client.GetSerial(SerialKindMousePress).AsRaw(), resizeEdgeToXdg(edge))
	}
}

// SetExclusiveZone sets a layer surface's exclusive zone.
func (w *WaylandWindow) SetExclusiveZone(zone gpui.Pixels) {
	state := w.ptr.state
	if state.surfaceState.setExclusiveZone(zone.ToI32()) {
		state.surface.Commit()
	}
}

// SetExclusiveEdge sets a layer surface's exclusive edge.
func (w *WaylandWindow) SetExclusiveEdge(edge gpui.LayerShellAnchor) {
	state := w.ptr.state
	if state.surfaceState.setExclusiveEdge(edge) {
		state.surface.Commit()
	}
}

// SetInputRegion restricts input to region; None lets the whole surface receive input.
func (w *WaylandWindow) SetInputRegion(region gpui.Option[[]gpui.Bounds[gpui.Pixels]]) {
	state := w.ptr.state
	if rects, ok := region.Get(); ok {
		wlRegion := state.globals.Compositor.CreateRegion()
		for _, rect := range rects {
			r := rect.Map(pixelsToI32)
			wlRegion.Add(r.Origin.X, r.Origin.Y, r.Size.Width, r.Size.Height)
		}
		state.surface.SetInputRegion(wlRegion)
		wlRegion.Destroy()
	} else {
		state.surface.SetInputRegion(nil)
	}

	state.surface.Commit()
}

// WindowDecorations returns the window's current decorations.
func (w *WaylandWindow) WindowDecorations() gpui.Decorations {
	state := w.ptr.state
	switch state.decorations {
	case gpui.WindowDecorationsServer:
		return gpui.DecorationsServer()
	default:
		return gpui.DecorationsClient(state.tiling)
	}
}

// RequestDecorations asks for server- or client-side decorations.
func (w *WaylandWindow) RequestDecorations(decorations gpui.WindowDecorations) {
	state := w.ptr.state
	if decoration := state.surfaceState.decoration(); decoration != nil {
		decoration.SetMode(decorationsToXdg(decorations))
		state.decorations = decorations
		updateWindow(state)
	} else {
		if decorations == gpui.WindowDecorationsServer {
			slog.Info("Server-side decorations requested, but the Wayland server does not support them. Falling back to client-side decorations.")
		}
		state.decorations = gpui.WindowDecorationsClient
		updateWindow(state)
	}
	w.ptr.requestRedraw()
}

// WindowControls returns the controls the compositor supports.
func (w *WaylandWindow) WindowControls() gpui.WindowControls {
	return w.ptr.state.windowControls
}

// SetClientInset sets the client-side decoration inset.
func (w *WaylandWindow) SetClientInset(inset gpui.Pixels) {
	state := w.ptr.state
	if current, ok := state.clientInset.Get(); !ok || current != inset {
		state.clientInset = gpui.Some(inset)
		updateWindow(state)
		w.ptr.requestRedraw()
	}
}

// SetClientCornerRadii sets the client-side decoration corner radii.
func (w *WaylandWindow) SetClientCornerRadii(radii gpui.Corners[gpui.Pixels]) {
	state := w.ptr.state
	if radii != state.clientCornerRadii {
		state.clientCornerRadii = radii
		updateWindow(state)
		w.ptr.requestRedraw()
	}
}

// UpdateIMEPosition moves the IME cursor rectangle of the focused window.
func (w *WaylandWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {
	state := w.ptr.state
	if !state.active {
		return
	}
	state.client.UpdateIMEPosition(bounds)
}

// GPUSpecs describes the renderer's GPU.
func (w *WaylandWindow) GPUSpecs() (gpui.GPUSpecs, bool) {
	renderer := w.ptr.state.renderer
	if renderer == nil {
		return gpui.GPUSpecs{}, false
	}
	return renderer.GPUSpecs()
}

// PlaySystemBell rings the compositor's system bell.
func (w *WaylandWindow) PlaySystemBell() {
	state := w.ptr.state
	var surface *wlproto.WlSurface
	if state.surfaceState.toplevel() != nil {
		surface = state.surface
	}
	if state.globals.SystemBell != nil {
		state.globals.SystemBell.Ring(surface)
	}
}

// A11yInit creates the window's accessibility adapter.
func (w *WaylandWindow) A11yInit(callbacks gpui.A11yCallbacks) {
	w.ptr.state.accesskitAdapter = newA11yAdapter(callbacks)
}

// A11yTreeUpdate sends treeUpdate to the adapter when a screen reader is active.
func (w *WaylandWindow) A11yTreeUpdate(treeUpdate a11y.TreeUpdate) {
	if adapter := w.ptr.state.accesskitAdapter; adapter != nil {
		adapter.UpdateIfActive(func() a11y.TreeUpdate { return treeUpdate })
	}
}

// A11yUpdateWindowBounds does nothing: Wayland does not expose window positions.
func (w *WaylandWindow) A11yUpdateWindowBounds() {}

func addRoundedRect(area gpui.Bounds[int32], radii gpui.Corners[gpui.Pixels], addRectangle func(x, y, width, height int32)) {
	if area.Size.Width <= 0 || area.Size.Height <= 0 {
		return
	}

	clampRadius := func(radius gpui.Pixels) int32 {
		return min(max(radius.ToI32(), 0), area.Size.Width/2, area.Size.Height/2)
	}
	topLeft := clampRadius(radii.TopLeft)
	topRight := clampRadius(radii.TopRight)
	bottomLeft := clampRadius(radii.BottomLeft)
	bottomRight := clampRadius(radii.BottomRight)

	cornerInset := func(radius, row int32) int32 {
		if row >= radius {
			return 0
		}
		y := float64(radius) - (float64(row) + 0.5)
		x := math.Sqrt(math.Pow(float64(radius), 2) - y*y)
		return int32(math.Ceil(float64(radius) - x - 0.5))
	}

	for _, corners := range [2]struct {
		left, right int32
		bottom      bool
	}{
		{topLeft, topRight, false},
		{bottomLeft, bottomRight, true},
	} {
		for row := int32(0); row < max(corners.left, corners.right); row++ {
			leftInset := cornerInset(corners.left, row)
			rightInset := cornerInset(corners.right, row)
			width := area.Size.Width - leftInset - rightInset
			if width <= 0 {
				continue
			}
			var y int32
			if corners.bottom {
				y = area.Origin.Y + area.Size.Height - row - 1
			} else {
				y = area.Origin.Y + row
			}
			addRectangle(area.Origin.X+leftInset, y, width, 1)
		}
	}

	topHeight := max(topLeft, topRight)
	bottomHeight := max(bottomLeft, bottomRight)
	height := area.Size.Height - topHeight - bottomHeight
	if height > 0 {
		addRectangle(area.Origin.X, area.Origin.Y+topHeight, area.Size.Width, height)
	}
}

func updateWindow(state *WaylandWindowState) {
	opaque := !state.IsTransparent()

	if state.renderer != nil {
		state.renderer.UpdateTransparency(!opaque)
	}

	area := insetByTiling(state.bounds.MapOrigin(zeroPixels), state.Inset(), state.tiling).Map(pixelsToI32).
		MapSize(func(value int32) int32 { return max(value, 0) })

	region := state.globals.Compositor.CreateRegion()

	if state.backgroundAppearance == gpui.WindowBackgroundAppearanceOpaque && state.decorations == gpui.WindowDecorationsServer {
		region.Add(area.Origin.X, area.Origin.Y, area.Size.Width, area.Size.Height)

		state.surface.SetOpaqueRegion(region)
	} else {
		state.surface.SetOpaqueRegion(nil)
	}

	if backgroundEffectsManager := state.globals.BackgroundEffectsManager; backgroundEffectsManager != nil {
		if state.backgroundAppearance == gpui.WindowBackgroundAppearanceBlurred {
			if state.backgroundEffects == nil {
				slog.Debug("Using ext-background-effect protocol")
				state.backgroundEffects = backgroundEffectsManager.GetBackgroundEffect(state.surface)
			}

			addRoundedRect(area, state.clientCornerRadii, func(x, y, width, height int32) {
				region.Add(x, y, width, height)
			})

			state.backgroundEffects.SetBlurRegion(region)
		} else if state.backgroundEffects != nil {
			backgroundEffects := state.backgroundEffects
			state.backgroundEffects = nil
			backgroundEffects.Destroy()
		}
	} else if blurManager := state.globals.KdeBlurManager; blurManager != nil {
		if state.backgroundAppearance == gpui.WindowBackgroundAppearanceBlurred {
			if state.kdeBlur == nil {
				slog.Debug("Using kde-blur protocol")
				state.kdeBlur = blurManager.Create(state.surface)
			}
			state.kdeBlur.Commit()
		} else {
			blurManager.Unset(state.surface)
			if state.kdeBlur != nil {
				blur := state.kdeBlur
				state.kdeBlur = nil
				blur.Release()
			}
		}
	}

	region.Destroy()
}

func decorationsToXdg(decorations gpui.WindowDecorations) wlproto.ZxdgToplevelDecorationV1Mode {
	switch decorations {
	case gpui.WindowDecorationsClient:
		return wlproto.ZxdgToplevelDecorationV1ModeClientSide
	default:
		return wlproto.ZxdgToplevelDecorationV1ModeServerSide
	}
}

func resizeEdgeToXdg(edge gpui.ResizeEdge) wlproto.XdgToplevelResizeEdge {
	switch edge {
	case gpui.ResizeEdgeTop:
		return wlproto.XdgToplevelResizeEdgeTop
	case gpui.ResizeEdgeTopRight:
		return wlproto.XdgToplevelResizeEdgeTopRight
	case gpui.ResizeEdgeRight:
		return wlproto.XdgToplevelResizeEdgeRight
	case gpui.ResizeEdgeBottomRight:
		return wlproto.XdgToplevelResizeEdgeBottomRight
	case gpui.ResizeEdgeBottom:
		return wlproto.XdgToplevelResizeEdgeBottom
	case gpui.ResizeEdgeBottomLeft:
		return wlproto.XdgToplevelResizeEdgeBottomLeft
	case gpui.ResizeEdgeLeft:
		return wlproto.XdgToplevelResizeEdgeLeft
	default:
		return wlproto.XdgToplevelResizeEdgeTopLeft
	}
}

func computeOuterSize(inset gpui.Pixels, newSize gpui.Option[gpui.Size[gpui.Pixels]], tiling gpui.Tiling) gpui.Option[gpui.Size[gpui.Pixels]] {
	size, ok := newSize.Get()
	if !ok {
		return newSize
	}
	if !tiling.Top {
		size.Height += inset
	}
	if !tiling.Bottom {
		size.Height += inset
	}
	if !tiling.Left {
		size.Width += inset
	}
	if !tiling.Right {
		size.Width += inset
	}
	return gpui.Some(size)
}

func insetByTiling(bounds gpui.Bounds[gpui.Pixels], inset gpui.Pixels, tiling gpui.Tiling) gpui.Bounds[gpui.Pixels] {
	if !tiling.Top {
		bounds.Origin.Y += inset
		bounds.Size.Height -= inset
	}
	if !tiling.Bottom {
		bounds.Size.Height -= inset
	}
	if !tiling.Left {
		bounds.Origin.X += inset
		bounds.Size.Width -= inset
	}
	if !tiling.Right {
		bounds.Size.Width -= inset
	}
	return bounds
}
