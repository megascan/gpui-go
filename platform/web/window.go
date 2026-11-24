//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"syscall/js"
	"weak"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

type webWindowCallbacks struct {
	requestFrame          gpui.Option[func(gpui.RequestFrameOptions)]
	input                 gpui.Option[func(gpui.PlatformInput) gpui.DispatchEventResult]
	activeStatusChange    gpui.Option[func(bool)]
	visibilityChange      gpui.Option[func(gpui.WindowVisibility)]
	hoverStatusChange     gpui.Option[func(bool)]
	resize                gpui.Option[func(gpui.Size[gpui.Pixels], float32)]
	visualViewportChanged gpui.Option[func()]
	moved                 gpui.Option[func()]
	shouldClose           gpui.Option[func() bool]
	close                 gpui.Option[func()]
	appearanceChanged     gpui.Option[func()]
	hitTestWindowControl  gpui.Option[func() (gpui.WindowControlArea, bool)]
}

type webWindowMutableState struct {
	renderer            *wgpu.WgpuRenderer
	bounds              gpui.Bounds[gpui.Pixels]
	scaleFactor         float32
	maxTextureDimension uint32
	title               string
	inputHandler        *gpui.PlatformInputHandler
	isFullscreen        bool
	isActive            bool
	visibility          gpui.WindowVisibility
	isHovered           bool
	mousePosition       gpui.Point[gpui.Pixels]
	modifiers           gpui.Modifiers
	capslock            gpui.Capslock
}

type touchTapCandidate struct {
	pointerID int32
	position  gpui.Point[gpui.Pixels]
}

type physicalSize struct {
	width, height uint32
}

type webWindowInner struct {
	browserWindow                    js.Value
	canvas                           js.Value
	imeMirror                        *imeMirror
	viewport                         *WebViewport
	hasDevicePixelSupport            bool
	isMac                            bool
	touchInput                       bool
	state                            webWindowMutableState
	callbacks                        webWindowCallbacks
	clickState                       clickState
	touchIDs                         touchIDs
	pressedButton                    gpui.Option[gpui.MouseButton]
	lastPhysicalSize                 physicalSize
	notifyScale                      bool
	isComposing                      bool
	suppressFocusStatusEvents        bool
	gestureStartVisualViewportHeight float64
	touchTapCandidate                gpui.Option[touchTapCandidate]
	mqlHandle                        *mqlHandle
	pendingPhysicalSize              gpui.Option[physicalSize]
	rafID                            gpui.Option[int]
	rafFunction                      gpui.Option[js.Value]
}

// WebWindow is the browser's single top-level window, drawn into a document-owned canvas.
type WebWindow struct {
	gpui.PlatformWindowDefaults
	inner                   *webWindowInner
	display                 gpui.PlatformDisplay
	lifecycle               *webWindowLifecycle
	activeWindow            *gpui.Option[gpui.AnyWindowHandle]
	rafClosure              js.Func
	resizeObserver          gpui.Option[js.Value]
	resizeObserverClosure   js.Func
	safeAreaObserver        gpui.Option[js.Value]
	safeAreaObserverClosure js.Func
	eventListeners          *WebEventListeners
	closed                  bool
}

var _ gpui.PlatformWindow = (*WebWindow)(nil)

var canvasCount int

func prepareCanvas(browserWindow js.Value) (js.Value, error) {
	document := jsGet(browserWindow, "document")
	if jsIsNullish(document) {
		return js.Undefined(), errors.New("No `document` found on window")
	}
	canvas, err := jsCall(document, "createElement", "canvas")
	if err != nil {
		return js.Undefined(), fmt.Errorf("Failed to create canvas element: %s", jsErrorDebug(err))
	}
	if !jsInstanceOf(canvas, "HTMLCanvasElement") {
		return js.Undefined(), fmt.Errorf("Created element is not a canvas: %s", jsDebug(canvas))
	}
	canvasCount++
	canvas.Set("id", "gpui-canvas-"+strconv.Itoa(canvasCount))
	canvas.Set("tabIndex", -1)

	style := canvas.Get("style")
	for _, pv := range [...][2]string{
		{"width", "100%"},
		{"height", "100%"},
		{"display", "block"},
		{"outline", "none"},
		{"touch-action", "none"},
		{"-webkit-touch-callout", "none"},
		{"user-select", "none"},
		{"-webkit-user-select", "none"},
	} {
		if _, err := jsCall(style, "setProperty", pv[0], pv[1]); err != nil {
			return js.Undefined(), fmt.Errorf("Failed to set canvas %s style: %s", pv[0], jsErrorDebug(err))
		}
	}

	body := jsGet(document, "body")
	if jsIsNullish(body) {
		return js.Undefined(), errors.New("No `body` found on document")
	}
	if _, err := jsCall(body, "appendChild", canvas); err != nil {
		return js.Undefined(), fmt.Errorf("Failed to append canvas to body: %s", jsErrorDebug(err))
	}
	return canvas, nil
}

func newWebWindow(
	handle gpui.AnyWindowHandle,
	params gpui.WindowParams,
	context *wgpu.WgpuContext,
	canvas js.Value,
	graphics *wgpu.PreparedWebGraphics,
	browserWindow js.Value,
	lifecycle *webWindowLifecycle,
	activeWindow *gpui.Option[gpui.AnyWindowHandle],
) (*WebWindow, error) {
	document := jsGet(browserWindow, "document")
	if jsIsNullish(document) {
		return nil, errors.New("No `document` found on window")
	}
	body := jsGet(document, "body")
	if jsIsNullish(body) {
		return nil, errors.New("No `body` found on document")
	}
	dprValue, _ := jsNumber(jsGet(browserWindow, "devicePixelRatio"))
	dpr := float32(dprValue)
	maxTextureDimension := context.Device.Limits().MaxTextureDimension2D
	hasDevicePixelSupport := checkDevicePixelSupport()
	rendererConfig := wgpu.WgpuSurfaceConfig{
		Size:                 gpui.Size[gpui.DevicePixels]{Width: 0, Height: 0},
		Transparent:          false,
		PreferredPresentMode: gpui.None[wgpu.PresentMode](),
	}
	surface := graphics.Surface
	graphics.Surface = nil
	renderer, err := wgpu.NewWgpuRendererFromSurface(context, surface, rendererConfig)
	if err != nil {
		return nil, err
	}

	touchInput := false
	if query, err := jsCall(browserWindow, "matchMedia", "(pointer: coarse)"); err == nil && !jsIsNullish(query) {
		touchInput, _ = jsBool(jsGet(query, "matches"))
	}
	mirror, err := newImeMirror(document, body, touchInput)
	if err != nil {
		renderer.Destroy()
		return nil, err
	}
	viewport, err := newWebViewport(document)
	if err != nil {
		renderer.Destroy()
		return nil, err
	}
	if _, err := viewport.update(browserWindow, canvas); err != nil {
		viewport.Close()
		renderer.Destroy()
		return nil, err
	}

	display := gpui.PlatformDisplay(NewWebDisplay(browserWindow))

	mutableState := webWindowMutableState{
		renderer:            renderer,
		bounds:              gpui.Bounds[gpui.Pixels]{},
		scaleFactor:         dpr,
		maxTextureDimension: maxTextureDimension,
		title:               "",
		inputHandler:        nil,
		isFullscreen:        false,
		isActive:            documentIsActive(browserWindow),
		visibility:          documentVisibility(browserWindow),
		isHovered:           false,
		mousePosition:       gpui.Point[gpui.Pixels]{},
		modifiers:           gpui.Modifiers{},
		capslock:            gpui.Capslock{},
	}

	isMac := isMacPlatform(browserWindow)

	inner := &webWindowInner{
		browserWindow:         browserWindow,
		canvas:                canvas,
		imeMirror:             mirror,
		viewport:              viewport,
		hasDevicePixelSupport: hasDevicePixelSupport,
		isMac:                 isMac,
		touchInput:            touchInput,
		state:                 mutableState,
		touchIDs:              touchIDs{active: map[int32]gpui.TouchID{}},
	}

	rafClosure := inner.createRafClosure()
	inner.wakeFrameLoop()

	resizeObserverClosure := createResizeObserverClosure(inner)
	resizeObserver := gpui.None[js.Value]()
	if observer, err := jsNew(js.Global().Get("ResizeObserver"), resizeObserverClosure); err == nil {
		resizeObserver = gpui.Some(observer)
	}

	if observer, ok := resizeObserver.Get(); ok {
		inner.observeCanvas(observer)
		inner.watchDprChanges(observer)
	}

	eventListeners := inner.registerEventListeners()
	weakInner := weak.Make(inner)
	safeAreaObserverClosure := js.FuncOf(func(this js.Value, args []js.Value) any {
		if inner := weakInner.Value(); inner != nil {
			inner.notifyViewportChanged()
		}
		return nil
	})
	safeAreaObserver := gpui.None[js.Value]()
	if observer, err := jsNew(js.Global().Get("ResizeObserver"), safeAreaObserverClosure); err == nil {
		inner.viewport.observeSafeArea(observer)
		safeAreaObserver = gpui.Some(observer)
	} else {
		slog.Warn(fmt.Sprintf("Failed to observe safe area: %s", jsErrorDebug(err)))
	}

	w := &WebWindow{
		inner:                   inner,
		display:                 display,
		lifecycle:               lifecycle,
		activeWindow:            activeWindow,
		rafClosure:              rafClosure,
		resizeObserver:          resizeObserver,
		resizeObserverClosure:   resizeObserverClosure,
		eventListeners:          eventListeners,
		safeAreaObserver:        safeAreaObserver,
		safeAreaObserverClosure: safeAreaObserverClosure,
	}
	gpui.BindPlatformWindowDefaults(&w.PlatformWindowDefaults, w)
	return w, nil
}

func createResizeObserverClosure(inner *webWindowInner) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		entries := argOrUndefined(args)
		if !jsIsObject(entries) {
			return nil
		}
		entry := entries.Index(0)
		if !jsInstanceOf(entry, "ResizeObserverEntry") {
			return nil
		}

		dpr, _ := jsNumber(jsGet(inner.browserWindow, "devicePixelRatio"))
		dprF32 := float32(dpr)

		var physicalWidth, physicalHeight uint32
		var logicalWidth, logicalHeight float32
		if inner.hasDevicePixelSupport {
			size := jsGet(entry, "devicePixelContentBoxSize").Index(0)
			inline, _ := jsNumber(jsGet(size, "inlineSize"))
			block, _ := jsNumber(jsGet(size, "blockSize"))
			physicalWidth = saturatingU32(inline)
			physicalHeight = saturatingU32(block)
			logicalWidth = float32(float64(physicalWidth) / dpr)
			logicalHeight = float32(float64(physicalHeight) / dpr)
		} else {
			rect := jsGet(entry, "contentRect")
			width, _ := jsNumber(jsGet(rect, "width"))
			height, _ := jsNumber(jsGet(rect, "height"))
			logicalWidth = float32(width)
			logicalHeight = float32(height)
			physicalWidth = saturatingU32(roundHalfAway(float64(logicalWidth) * dpr))
			physicalHeight = saturatingU32(roundHalfAway(float64(logicalHeight) * dpr))
		}

		scaleChanged := inner.notifyScale
		inner.notifyScale = false
		sizeChanged := inner.lastPhysicalSize != physicalSize{physicalWidth, physicalHeight}

		if !scaleChanged && !sizeChanged {
			return nil
		}
		inner.lastPhysicalSize = physicalSize{physicalWidth, physicalHeight}

		if physicalWidth == 0 || physicalHeight == 0 {
			inner.state.bounds.Size = gpui.Size[gpui.Pixels]{}
			inner.state.scaleFactor = dprF32
			withCallback(inner, func(c *webWindowCallbacks) *gpui.Option[func(gpui.Size[gpui.Pixels], float32)] { return &c.resize },
				func(callback func(gpui.Size[gpui.Pixels], float32)) { callback(gpui.Size[gpui.Pixels]{}, dprF32) })
			return nil
		}

		maxTextureDimension := inner.state.maxTextureDimension
		clampedWidth := min(physicalWidth, maxTextureDimension)
		clampedHeight := min(physicalHeight, maxTextureDimension)

		if clampedWidth != physicalWidth || clampedHeight != physicalHeight {
			logicalWidth = float32(float64(clampedWidth) / dpr)
			logicalHeight = float32(float64(clampedHeight) / dpr)
		}

		inner.pendingPhysicalSize = gpui.Some(physicalSize{clampedWidth, clampedHeight})

		inner.state.bounds.Size = gpui.Size[gpui.Pixels]{Width: gpui.Px(logicalWidth), Height: gpui.Px(logicalHeight)}
		inner.state.scaleFactor = dprF32

		newSize := gpui.Size[gpui.Pixels]{Width: gpui.Px(logicalWidth), Height: gpui.Px(logicalHeight)}

		withCallback(inner, func(c *webWindowCallbacks) *gpui.Option[func(gpui.Size[gpui.Pixels], float32)] { return &c.resize },
			func(callback func(gpui.Size[gpui.Pixels], float32)) { callback(newSize, dprF32) })

		withCallback(inner, func(c *webWindowCallbacks) *gpui.Option[func(gpui.RequestFrameOptions)] { return &c.requestFrame },
			func(callback func(gpui.RequestFrameOptions)) {
				callback(gpui.RequestFrameOptions{
					RequirePresentation: true,
					ForceRender:         true,
					SignalAt:            gpui.None[scheduler.Instant](),
					SignalSource:        gpui.FrameRequestSourceNativeCallback,
				})
			})
		return nil
	})
}

func roundHalfAway(v float64) float64 {
	if v < 0 {
		return -roundHalfAway(-v)
	}
	r := float64(int64(v))
	if v-r >= 0.5 {
		r++
	}
	return r
}

func (w *webWindowInner) sampleViewport() bool {
	changed, err := w.viewport.update(w.browserWindow, w.canvas)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to update browser viewport: %v", err))
		return false
	}
	return changed
}

func (w *webWindowInner) notifyViewportChanged() {
	withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func()] { return &c.visualViewportChanged },
		func(callback func()) { callback() })
}

func (w *webWindowInner) refreshActiveStatus() {
	active := documentIsActive(w.browserWindow)
	previous := w.state.isActive
	w.state.isActive = active
	if previous != active {
		withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(bool)] { return &c.activeStatusChange },
			func(callback func(bool)) { callback(active) })
	}
}

func withCallback[C any](w *webWindowInner, slot func(*webWindowCallbacks) *gpui.Option[C], invoke func(C)) bool {
	callback, ok := slot(&w.callbacks).Take().Get()
	if !ok {
		return false
	}
	invoke(callback)
	*slot(&w.callbacks) = gpui.Some(callback)
	return true
}

func (w *webWindowInner) createRafClosure() js.Func {
	closure := js.FuncOf(func(this js.Value, args []js.Value) any {
		w.rafID = gpui.None[int]()
		withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(gpui.RequestFrameOptions)] { return &c.requestFrame },
			func(callback func(gpui.RequestFrameOptions)) {
				callback(gpui.RequestFrameOptions{
					RequirePresentation: false,
					ForceRender:         false,
					SignalAt:            gpui.None[scheduler.Instant](),
					SignalSource:        gpui.FrameRequestSourceNativeCallback,
				})
			})
		return nil
	})
	w.rafFunction = gpui.Some(closure.Value)
	return closure
}

func (w *webWindowInner) wakeFrameLoop() {
	if w.rafID.IsSome() {
		return
	}
	if fn, ok := w.rafFunction.Get(); ok {
		w.rafID = gpui.None[int]()
		if id, err := jsCall(w.browserWindow, "requestAnimationFrame", fn); err == nil {
			if n, ok := jsNumber(id); ok {
				w.rafID = gpui.Some(int(n))
			}
		}
	}
}

func (w *webWindowInner) observeCanvas(observer js.Value) {
	_, _ = jsCall(observer, "unobserve", w.canvas)
	if w.hasDevicePixelSupport {
		options := js.Global().Get("Object").New()
		options.Set("box", "device-pixel-content-box")
		_, _ = jsCall(observer, "observe", w.canvas, options)
	} else {
		_, _ = jsCall(observer, "observe", w.canvas)
	}
}

func (w *webWindowInner) watchDprChanges(observer js.Value) {
	currentDpr, _ := jsNumber(jsGet(w.browserWindow, "devicePixelRatio"))
	dpr := formatF64(currentDpr)
	mediaQuery := fmt.Sprintf("(resolution: %sdppx), (-webkit-device-pixel-ratio: %s)", dpr, dpr)
	mql, err := jsCall(w.browserWindow, "matchMedia", mediaQuery)
	if err != nil || jsIsNullish(mql) {
		return
	}

	closure := js.FuncOf(func(this js.Value, args []js.Value) any {
		w.notifyScale = true
		w.observeCanvas(observer)
		w.watchDprChanges(observer)
		return nil
	})

	_, _ = jsCall(mql, "addEventListener", "change", closure)

	previous := w.mqlHandle
	w.mqlHandle = &mqlHandle{mql: mql, closure: closure}
	previous.Close()
}

func (w *webWindowInner) registerVisibilityChange() *EventListenerHandle {
	document := jsGet(w.browserWindow, "document")
	if jsIsNullish(document) {
		return nil
	}
	return eventListenerHandleAdd(document, "visibilitychange", func(js.Value) {
		visibility := documentVisibility(w.browserWindow)
		visibilityChanged := w.state.visibility != visibility
		w.state.visibility = visibility
		w.refreshActiveStatus()
		if visibilityChanged {
			withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(gpui.WindowVisibility)] { return &c.visibilityChange },
				func(callback func(gpui.WindowVisibility)) { callback(visibility) })
		}
	})
}

func (w *webWindowInner) registerFullscreenChange() *EventListenerHandle {
	document := jsGet(w.browserWindow, "document")
	if jsIsNullish(document) {
		return nil
	}
	return eventListenerHandleAdd(document, "fullscreenchange", func(js.Value) {
		document := jsGet(w.browserWindow, "document")
		w.state.isFullscreen = !jsIsNullish(document) && !jsIsNullish(jsGet(document, "fullscreenElement"))
	})
}

func withInputHandler[R any](w *webWindowInner, f func(*gpui.PlatformInputHandler) R) (R, bool) {
	handler := w.state.inputHandler
	w.state.inputHandler = nil
	if handler == nil {
		var zero R
		return zero, false
	}
	result := f(handler)
	w.state.inputHandler = handler
	return result, true
}

func (w *webWindowInner) registerAppearanceChange() *EventListenerHandle {
	mql, err := jsCall(w.browserWindow, "matchMedia", "(prefers-color-scheme: dark)")
	if err != nil || jsIsNullish(mql) {
		return nil
	}
	return eventListenerHandleAdd(mql, "change", func(js.Value) {
		withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func()] { return &c.appearanceChanged },
			func(callback func()) { callback() })
	})
}

// Close cancels the frame request, disconnects the observers, removes the canvas and the IME
// mirror and marks the platform's window closed; it is upstream's drop and runs once.
func (w *WebWindow) Close() {
	if w.closed {
		return
	}
	w.closed = true
	if rafID, ok := w.inner.rafID.Take().Get(); ok {
		_, _ = jsCall(w.inner.browserWindow, "cancelAnimationFrame", rafID)
	}
	w.inner.rafFunction.Take()
	if observer, ok := w.resizeObserver.Get(); ok {
		_, _ = jsCall(observer, "disconnect")
	}
	if observer, ok := w.safeAreaObserver.Get(); ok {
		_, _ = jsCall(observer, "disconnect")
	}

	handle := w.inner.mqlHandle
	w.inner.mqlHandle = nil
	handle.Close()

	_, _ = jsCall(w.inner.canvas, "remove")
	w.inner.imeMirror.remove()
	w.activeWindow.Take()
	*w.lifecycle = webWindowLifecycleClosed

	w.rafClosure.Release()
	w.resizeObserverClosure.Release()
	w.safeAreaObserverClosure.Release()
	w.eventListeners.Close()
	w.inner.viewport.Close()
	w.inner.state.renderer.Destroy()
}

func currentAppearance(browserWindow js.Value) gpui.WindowAppearance {
	isDark := false
	if mql, err := jsCall(browserWindow, "matchMedia", "(prefers-color-scheme: dark)"); err == nil && !jsIsNullish(mql) {
		isDark, _ = jsBool(jsGet(mql, "matches"))
	}
	if isDark {
		return gpui.WindowAppearanceDark
	}
	return gpui.WindowAppearanceLight
}

func documentVisibility(browserWindow js.Value) gpui.WindowVisibility {
	isVisible := true
	if document := jsGet(browserWindow, "document"); !jsIsNullish(document) {
		if state, ok := jsString(jsGet(document, "visibilityState")); ok {
			isVisible = state == "visible"
		}
	}
	if isVisible {
		return gpui.WindowVisibilityVisible
	}
	return gpui.WindowVisibilityHidden
}

func documentIsActive(browserWindow js.Value) bool {
	if !documentVisibility(browserWindow).IsVisible() {
		return false
	}
	document := jsGet(browserWindow, "document")
	if jsIsNullish(document) {
		return true
	}
	focused, err := jsCall(document, "hasFocus")
	if err != nil {
		return true
	}
	if b, ok := jsBool(focused); ok {
		return b
	}
	return true
}

type mqlHandle struct {
	mql     js.Value
	closure js.Func
}

// Close removes the media-query listener and releases its function.
func (h *mqlHandle) Close() {
	if h == nil {
		return
	}
	_, _ = jsCall(h.mql, "removeEventListener", "change", h.closure)
	h.closure.Release()
}

func checkDevicePixelSupport() bool {
	constructor := js.Global().Get("ResizeObserverEntry")
	if !jsIsObject(constructor) {
		return false
	}
	prototype := jsGet(constructor, "prototype")
	if !jsIsObject(prototype) {
		return false
	}
	descriptor, err := jsCall(js.Global().Get("Object"), "getOwnPropertyDescriptor", prototype, "devicePixelContentBoxSize")
	return err == nil && !descriptor.IsUndefined()
}

// RawWindowHandle returns the canvas, named by its id.
func (w *WebWindow) RawWindowHandle() gpui.RawWindowHandle {
	id, _ := jsString(jsGet(w.inner.canvas, "id"))
	return gpui.RawWindowHandle{Kind: gpui.RawWindowHandleWeb, WebCanvasID: id}
}

// Bounds returns the canvas bounds.
func (w *WebWindow) Bounds() gpui.Bounds[gpui.Pixels] {
	return w.inner.state.bounds
}

// IsMaximized reports false.
func (w *WebWindow) IsMaximized() bool {
	return false
}

// WindowBounds returns the bounds as windowed.
func (w *WebWindow) WindowBounds() gpui.WindowBounds {
	return gpui.WindowBoundsWindowed(w.Bounds())
}

// ContentSize returns the canvas size.
func (w *WebWindow) ContentSize() gpui.Size[gpui.Pixels] {
	return w.inner.state.bounds.Size
}

// Resize sets the canvas's CSS size.
func (w *WebWindow) Resize(size gpui.Size[gpui.Pixels]) {
	style := jsGet(w.inner.canvas, "style")
	_, _ = jsCall(style, "setProperty", "width", formatF32(float32(size.Width))+"px")
	w.inner.viewport.setCanvasHeight(w.inner.canvas, formatF32(float32(size.Height))+"px")
}

// ScaleFactor returns the device pixel ratio.
func (w *WebWindow) ScaleFactor() float32 {
	return w.inner.state.scaleFactor
}

// Appearance returns the page's preferred color scheme.
func (w *WebWindow) Appearance() gpui.WindowAppearance {
	return currentAppearance(w.inner.browserWindow)
}

// Display returns the browser display.
func (w *WebWindow) Display() gpui.PlatformDisplay {
	return w.display
}

// MousePosition returns the last pointer position.
func (w *WebWindow) MousePosition() gpui.Point[gpui.Pixels] {
	return w.inner.state.mousePosition
}

// Modifiers returns the last modifier state.
func (w *WebWindow) Modifiers() gpui.Modifiers {
	return w.inner.state.modifiers
}

// Capslock returns the last caps lock state.
func (w *WebWindow) Capslock() gpui.Capslock {
	return w.inner.state.capslock
}

// SetInputHandler sets the input handler.
func (w *WebWindow) SetInputHandler(inputHandler *gpui.PlatformInputHandler) {
	w.inner.state.inputHandler = inputHandler
}

// TakeInputHandler removes and returns the input handler.
func (w *WebWindow) TakeInputHandler() *gpui.PlatformInputHandler {
	handler := w.inner.state.inputHandler
	w.inner.state.inputHandler = nil
	return handler
}

// SetTextInputConfiguration applies configuration to the IME mirror.
func (w *WebWindow) SetTextInputConfiguration(configuration gpui.TextInputConfiguration) {
	w.inner.imeMirror.applyConfiguration(configuration)
}

// ShowSoftKeyboard shows the software keyboard on touch devices.
func (w *WebWindow) ShowSoftKeyboard() {
	w.inner.showVirtualKeyboard()
}

// HideSoftKeyboard hides the software keyboard on touch devices.
func (w *WebWindow) HideSoftKeyboard() {
	if !w.inner.touchInput {
		return
	}
	w.inner.setVirtualKeyboardEnabled(false)
	w.inner.suppressFocusStatusEvents = true
	w.inner.imeMirror.blur()
	w.inner.suppressFocusStatusEvents = false
	weakInner := weak.Make(w.inner)
	callback := jsFuncOnce(func([]js.Value) {
		if inner := weakInner.Value(); inner != nil && !inner.imeMirror.virtualKeyboardEnabled() && inner.state.isActive {
			inner.suppressFocusStatusEvents = true
			inner.imeMirror.focus()
			inner.suppressFocusStatusEvents = false
		}
	})
	if _, err := jsCall(w.inner.browserWindow, "setTimeout", callback); err != nil {
		callback.Release()
		slog.Warn(fmt.Sprintf("failed to restore keyboard event receiver: %s", jsErrorDebug(err)))
	}
}

// VisualViewportBounds returns the visible part of the canvas.
func (w *WebWindow) VisualViewportBounds() gpui.Bounds[gpui.Pixels] {
	return w.inner.viewport.visibleBounds
}

// PrepareFrame samples the browser viewport.
func (w *WebWindow) PrepareFrame() bool {
	return w.inner.sampleViewport()
}

// OnVisualViewportChanged registers the visual viewport callback.
func (w *WebWindow) OnVisualViewportChanged(callback func()) {
	w.inner.callbacks.visualViewportChanged = gpui.Some(callback)
}

// Insets returns the safe-area insets.
func (w *WebWindow) Insets() gpui.WindowInsets {
	return w.inner.viewport.insets
}

// TextInputStateChanged reacts to editability transitions.
func (w *WebWindow) TextInputStateChanged(change gpui.TextInputStateChange) {
	if w.inner.touchInput {
		switch change {
		case gpui.TextInputStateChangeFocusGained:
			w.inner.showVirtualKeyboard()
		case gpui.TextInputStateChangeFocusLost:
			w.HideSoftKeyboard()
		}
		return
	}
	switch change {
	case gpui.TextInputStateChangeFocusGained:
		w.inner.imeMirror.setReadOnly(false)
		if !w.inner.imeMirror.isFocused() {
			w.inner.focusImeMirror()
		}
	case gpui.TextInputStateChangeFocusLost:
		w.inner.imeMirror.setReadOnly(true)
	}
}

// Prompt returns nil: the browser has no native prompts.
func (w *WebWindow) Prompt(level gpui.PromptLevel, msg string, detail gpui.Option[string], answers []gpui.PromptButton) *scheduler.OneshotReceiver[int] {
	return nil
}

// Activate marks the window active.
func (w *WebWindow) Activate() {
	w.inner.state.isActive = true
}

// IsActive reports whether the page is visible and focused.
func (w *WebWindow) IsActive() bool {
	return w.inner.state.isActive
}

// Visibility returns the page visibility.
func (w *WebWindow) Visibility() gpui.WindowVisibility {
	return w.inner.state.visibility
}

// IsHovered reports whether the pointer is over the canvas.
func (w *WebWindow) IsHovered() bool {
	return w.inner.state.isHovered
}

// BackgroundAppearance returns opaque.
func (w *WebWindow) BackgroundAppearance() gpui.WindowBackgroundAppearance {
	return gpui.WindowBackgroundAppearanceOpaque
}

// SetTitle sets the document title.
func (w *WebWindow) SetTitle(title string) {
	w.inner.state.title = title
	if document := jsGet(w.inner.browserWindow, "document"); !jsIsNullish(document) {
		document.Set("title", title)
	}
}

// SetBackgroundAppearance does nothing.
func (w *WebWindow) SetBackgroundAppearance(background gpui.WindowBackgroundAppearance) {}

// Minimize logs that it is unsupported.
func (w *WebWindow) Minimize() {
	slog.Warn("WebWindow::minimize is not supported in the browser")
}

// Zoom logs that it is unsupported.
func (w *WebWindow) Zoom() {
	slog.Warn("WebWindow::zoom is not supported in the browser")
}

// ToggleFullscreen requests or exits fullscreen for the canvas.
func (w *WebWindow) ToggleFullscreen() {
	document := jsGet(w.inner.browserWindow, "document")
	if jsIsNullish(document) {
		return
	}
	if !jsIsNullish(jsGet(document, "fullscreenElement")) {
		_, _ = jsCall(document, "exitFullscreen")
	} else {
		_, _ = jsCall(w.inner.canvas, "requestFullscreen")
	}
}

// IsFullscreen reports the document's fullscreen state.
func (w *WebWindow) IsFullscreen() bool {
	return w.inner.state.isFullscreen
}

// FrameWaker returns a function that requests an animation frame.
func (w *WebWindow) FrameWaker() func() {
	weakInner := weak.Make(w.inner)
	return func() {
		if inner := weakInner.Value(); inner != nil {
			inner.wakeFrameLoop()
		}
	}
}

// OnRequestFrame registers the frame callback.
func (w *WebWindow) OnRequestFrame(callback func(options gpui.RequestFrameOptions)) {
	w.inner.callbacks.requestFrame = gpui.Some(callback)
}

// OnInput registers the input callback.
func (w *WebWindow) OnInput(callback func(input gpui.PlatformInput) gpui.DispatchEventResult) {
	w.inner.callbacks.input = gpui.Some(callback)
}

// OnActiveStatusChange registers the activation callback.
func (w *WebWindow) OnActiveStatusChange(callback func(active bool)) {
	w.inner.callbacks.activeStatusChange = gpui.Some(callback)
}

// OnVisibilityChange registers the visibility callback.
func (w *WebWindow) OnVisibilityChange(callback func(visibility gpui.WindowVisibility)) {
	w.inner.callbacks.visibilityChange = gpui.Some(callback)
}

// OnHoverStatusChange registers the hover callback.
func (w *WebWindow) OnHoverStatusChange(callback func(hovered bool)) {
	w.inner.callbacks.hoverStatusChange = gpui.Some(callback)
}

// OnResize registers the resize callback.
func (w *WebWindow) OnResize(callback func(size gpui.Size[gpui.Pixels], scaleFactor float32)) {
	w.inner.callbacks.resize = gpui.Some(callback)
}

// OnMoved registers the move callback.
func (w *WebWindow) OnMoved(callback func()) {
	w.inner.callbacks.moved = gpui.Some(callback)
}

// OnDisplayChanged does nothing: the browser is the only display.
func (w *WebWindow) OnDisplayChanged(callback func()) {}

// OnShouldClose registers the should-close callback.
func (w *WebWindow) OnShouldClose(callback func() bool) {
	w.inner.callbacks.shouldClose = gpui.Some(callback)
}

// OnClose registers the close callback.
func (w *WebWindow) OnClose(callback func()) {
	w.inner.callbacks.close = gpui.Some(callback)
}

// OnHitTestWindowControl registers the window-control hit test.
func (w *WebWindow) OnHitTestWindowControl(callback func() (gpui.WindowControlArea, bool)) {
	w.inner.callbacks.hitTestWindowControl = gpui.Some(callback)
}

// OnAppearanceChanged registers the appearance callback.
func (w *WebWindow) OnAppearanceChanged(callback func()) {
	w.inner.callbacks.appearanceChanged = gpui.Some(callback)
}

// Draw resizes the canvas backing store if a resize is pending and draws scene.
func (w *WebWindow) Draw(scene *gpui.Scene) {
	if size, ok := w.inner.pendingPhysicalSize.Take().Get(); ok {
		width, _ := jsNumber(jsGet(w.inner.canvas, "width"))
		height, _ := jsNumber(jsGet(w.inner.canvas, "height"))
		if uint32(width) != size.width || uint32(height) != size.height {
			w.inner.canvas.Set("width", size.width)
			w.inner.canvas.Set("height", size.height)
		}
		w.inner.state.renderer.UpdateDrawableSize(gpui.Size[gpui.DevicePixels]{
			Width:  gpui.DevicePixels(int32(size.width)),
			Height: gpui.DevicePixels(int32(size.height)),
		})
	}
	w.inner.state.renderer.Draw(scene)
}

// SpriteAtlas returns the renderer's atlas.
func (w *WebWindow) SpriteAtlas() gpui.PlatformAtlas {
	return w.inner.state.renderer.SpriteAtlas()
}

// IsSubpixelRenderingSupported reports dual-source blending support.
func (w *WebWindow) IsSubpixelRenderingSupported() bool {
	return w.inner.state.renderer.SupportsDualSourceBlending()
}

// GPUSpecs returns the renderer's GPU specs.
func (w *WebWindow) GPUSpecs() (gpui.GPUSpecs, bool) {
	return w.inner.state.renderer.GPUSpecs()
}

// UpdateIMEPosition does nothing.
func (w *WebWindow) UpdateIMEPosition(bounds gpui.Bounds[gpui.Pixels]) {}

// RequestDecorations does nothing.
func (w *WebWindow) RequestDecorations(decorations gpui.WindowDecorations) {}

// ShowWindowMenu does nothing.
func (w *WebWindow) ShowWindowMenu(position gpui.Point[gpui.Pixels]) {}

// StartWindowMove does nothing.
func (w *WebWindow) StartWindowMove() {}

// StartWindowResize does nothing.
func (w *WebWindow) StartWindowResize(edge gpui.ResizeEdge) {}

// WindowDecorations returns server decorations.
func (w *WebWindow) WindowDecorations() gpui.Decorations {
	return gpui.DecorationsServer()
}

// SetAppID does nothing.
func (w *WebWindow) SetAppID(appID string) {}

// WindowControls reports fullscreen as the only control.
func (w *WebWindow) WindowControls() gpui.WindowControls {
	return gpui.WindowControls{
		Fullscreen: true,
		Maximize:   false,
		Minimize:   false,
		WindowMenu: false,
	}
}

// SetClientInset does nothing.
func (w *WebWindow) SetClientInset(inset gpui.Pixels) {}
