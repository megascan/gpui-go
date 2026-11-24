//go:build windows

package windows

import (
	"math"
	"reflect"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const defaultViewportSize int32 = 1000

const (
	directmanipulationConfigurationInteraction              = 1
	directmanipulationConfigurationTranslationX             = 2
	directmanipulationConfigurationTranslationY             = 4
	directmanipulationConfigurationScaling                  = 16
	directmanipulationConfigurationTranslationInertia       = 32
	directmanipulationConfigurationRailsX                   = 256
	directmanipulationConfigurationRailsY                   = 512
	directmanipulationViewportOptionsManualupdate           = 2
	directmanipulationViewportOptionsDisablepixelsnap       = 16
	directmanipulationRunning                         int32 = 3
	directmanipulationInertia                         int32 = 4
	directmanipulationReady                           int32 = 5
)

var (
	clsidDirectManipulationManager             = com.MustGUID("{54E211B6-3650-4F75-8334-FA359598E1C5}")
	iidIDirectManipulationManager              = com.MustGUID("{FBF5D3B4-70C7-4163-9322-5A6F660D6FBC}")
	iidIDirectManipulationUpdateManager        = com.MustGUID("{B0AE62FD-BE34-46E7-9CAA-D361FACBB9CC}")
	iidIDirectManipulationViewport             = com.MustGUID("{28B85A3D-60A0-48BD-9BA1-5CE8D9EA3A6D}")
	iidIDirectManipulationViewportEventHandler = com.MustGUID("{952121DA-D69F-45F9-B0F9-F23944321A6D}")
)

type iDirectManipulationManager struct{ com.IUnknown }

func (m *iDirectManipulationManager) activate(window HWND) error {
	return com.CallHR(m.Ptr(), com.IUnknownSlots+0, uintptr(window))
}

func (m *iDirectManipulationManager) deactivate(window HWND) error {
	return com.CallHR(m.Ptr(), com.IUnknownSlots+1, uintptr(window))
}

func (m *iDirectManipulationManager) getUpdateManager() (*iDirectManipulationUpdateManager, error) {
	var out *iDirectManipulationUpdateManager
	err := com.CallHR(m.Ptr(), com.IUnknownSlots+4, uintptr(unsafe.Pointer(&iidIDirectManipulationUpdateManager)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

func (m *iDirectManipulationManager) createViewport(window HWND) (*iDirectManipulationViewport, error) {
	var out *iDirectManipulationViewport
	err := com.CallHR(m.Ptr(), com.IUnknownSlots+5, 0, uintptr(window), uintptr(unsafe.Pointer(&iidIDirectManipulationViewport)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

type iDirectManipulationUpdateManager struct{ com.IUnknown }

func (u *iDirectManipulationUpdateManager) update() error {
	return com.CallHR(u.Ptr(), com.IUnknownSlots+2, 0)
}

type iDirectManipulationViewport struct{ com.IUnknown }

func (v *iDirectManipulationViewport) enable() error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+0)
}

func (v *iDirectManipulationViewport) setContact(pointerID uint32) error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+2, uintptr(pointerID))
}

func (v *iDirectManipulationViewport) setViewportRect(rect *win32.RECT) error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+9, uintptr(unsafe.Pointer(rect)))
}

func (v *iDirectManipulationViewport) zoomToRect(left, top, right, bottom float32, animate bool) error {
	call := dmFloatFn[func(unsafe.Pointer, float32, float32, float32, float32, int32) int32](v.Ptr(), com.IUnknownSlots+10)
	var a int32
	if animate {
		a = 1
	}
	return com.HRESULT(call(v.Ptr(), left, top, right, bottom, a)).Err()
}

func (v *iDirectManipulationViewport) setViewportOptions(options int32) error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+16, uintptr(options))
}

func (v *iDirectManipulationViewport) activateConfiguration(configuration int32) error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+19, uintptr(configuration))
}

func (v *iDirectManipulationViewport) addEventHandler(window HWND, handler *com.IUnknown) (uint32, error) {
	var cookie uint32
	err := com.CallHR(v.Ptr(), com.IUnknownSlots+22, uintptr(window), uintptr(unsafe.Pointer(handler)), uintptr(unsafe.Pointer(&cookie)))
	return cookie, err
}

func (v *iDirectManipulationViewport) stop() error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+26)
}

func (v *iDirectManipulationViewport) abandon() error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+27)
}

type iDirectManipulationContent struct{ com.IUnknown }

func (c *iDirectManipulationContent) getContentTransform(matrix []float32) error {
	return com.CallHR(c.Ptr(), com.IUnknownSlots+6, uintptr(unsafe.Pointer(&matrix[0])), uintptr(len(matrix)))
}

type dmFnKey struct {
	addr uintptr
	typ  reflect.Type
}

var dmFloatFns sync.Map

func dmFloatFn[F any](this unsafe.Pointer, slot int) F {
	key := dmFnKey{addr: com.Method(this, slot), typ: reflect.TypeFor[F]()}
	if v, ok := dmFloatFns.Load(key); ok {
		return v.(F)
	}
	var f F
	purego.RegisterFunc(&f, key.addr)
	v, _ := dmFloatFns.LoadOrStore(key, f)
	return v.(F)
}

// DirectManipulationHandler turns DirectManipulation viewport updates from a precision touchpad
// into scroll and pinch input. Close is its drop.
type DirectManipulationHandler struct {
	manager       *iDirectManipulationManager
	updateManager *iDirectManipulationUpdateManager
	viewport      *iDirectManipulationViewport
	handlerCookie uint32
	handler       *com.Object
	window        HWND
	shared        *directManipulationShared
	closed        bool
}

type directManipulationShared struct {
	scaleFactor   float32
	pendingEvents []gpui.PlatformInput
}

// NewDirectManipulationHandler activates DirectManipulation on window.
func NewDirectManipulationHandler(window HWND, scaleFactor float32) (*DirectManipulationHandler, error) {
	manager, err := com.CreateInstance[iDirectManipulationManager](&clsidDirectManipulationManager, com.CLSCTX_INPROC_SERVER, &iidIDirectManipulationManager)
	if err != nil {
		return nil, err
	}
	h := &DirectManipulationHandler{manager: manager, window: window}
	fail := func(err error) (*DirectManipulationHandler, error) {
		h.releaseInterfaces()
		return nil, err
	}

	if h.updateManager, err = manager.getUpdateManager(); err != nil {
		return fail(err)
	}
	if h.viewport, err = manager.createViewport(window); err != nil {
		return fail(err)
	}
	viewport := h.viewport

	configuration := int32(directmanipulationConfigurationInteraction |
		directmanipulationConfigurationTranslationX |
		directmanipulationConfigurationTranslationY |
		directmanipulationConfigurationTranslationInertia |
		directmanipulationConfigurationRailsX |
		directmanipulationConfigurationRailsY |
		directmanipulationConfigurationScaling)
	if err := viewport.activateConfiguration(configuration); err != nil {
		return fail(err)
	}

	if err := viewport.setViewportOptions(directmanipulationViewportOptionsManualupdate | directmanipulationViewportOptionsDisablepixelsnap); err != nil {
		return fail(err)
	}

	rect := win32.RECT{Left: 0, Top: 0, Right: defaultViewportSize, Bottom: defaultViewportSize}
	if err := viewport.setViewportRect(&rect); err != nil {
		return fail(err)
	}

	if err := manager.activate(window); err != nil {
		return fail(err)
	}
	if err := viewport.enable(); err != nil {
		return fail(err)
	}

	h.shared = &directManipulationShared{scaleFactor: scaleFactor}
	eventHandler := newDirectManipulationEventHandler(window, h.shared)
	h.handler = com.NewObject(eventHandler, directManipulationEventHandlerVtable.Get())

	cookie, err := viewport.addEventHandler(window, h.handler.Interface(0))
	if err != nil {
		return fail(err)
	}
	h.handlerCookie = cookie

	if err := h.updateManager.update(); err != nil {
		return fail(err)
	}

	return h, nil
}

func (h *DirectManipulationHandler) releaseInterfaces() {
	if h.handler != nil {
		h.handler.Release()
		h.handler = nil
	}
	if h.viewport != nil {
		h.viewport.Release()
		h.viewport = nil
	}
	if h.updateManager != nil {
		h.updateManager.Release()
		h.updateManager = nil
	}
	if h.manager != nil {
		h.manager.Release()
		h.manager = nil
	}
}

func (h *DirectManipulationHandler) setScaleFactor(scaleFactor float32) {
	h.shared.scaleFactor = scaleFactor
}

func (h *DirectManipulationHandler) onPointerHitTest(wparam win32.WPARAM) {
	pointerID := uint32(loword(wparam))
	var pointerType int32
	if win32.GetPointerType(pointerID, &pointerType) == nil && pointerType == win32.PT_TOUCHPAD {
		if err := h.viewport.setContact(pointerID); err != nil {
			gpuiutil.LogErr(err)
		}
	}
}

func (h *DirectManipulationHandler) update() {
	if err := h.updateManager.update(); err != nil {
		gpuiutil.LogErr(err)
	}
}

func (h *DirectManipulationHandler) drainEvents() []gpui.PlatformInput {
	events := h.shared.pendingEvents
	h.shared.pendingEvents = nil
	return events
}

// Close stops and abandons the viewport, deactivates the manager and releases its interfaces.
func (h *DirectManipulationHandler) Close() {
	if h.closed {
		return
	}
	h.closed = true
	if err := h.viewport.stop(); err != nil {
		gpuiutil.LogErr(err)
	}
	if err := h.viewport.abandon(); err != nil {
		gpuiutil.LogErr(err)
	}
	if err := h.manager.deactivate(h.window); err != nil {
		gpuiutil.LogErr(err)
	}
	h.releaseInterfaces()
}

type gestureKind uint8

const (
	gestureKindNone gestureKind = iota
	gestureKindScroll
	gestureKindPinch
)

type directManipulationEventHandler struct {
	window      HWND
	shared      *directManipulationShared
	gestureKind gestureKind
	lastScale   float32
	lastXOffset float32
	lastYOffset float32
	scrollPhase gpui.TouchPhase
}

func newDirectManipulationEventHandler(window HWND, shared *directManipulationShared) *directManipulationEventHandler {
	return &directManipulationEventHandler{
		window:      window,
		shared:      shared,
		gestureKind: gestureKindNone,
		lastScale:   1.0,
		scrollPhase: gpui.TouchPhaseStarted,
	}
}

func (e *directManipulationEventHandler) push(event gpui.PlatformInput) {
	e.shared.pendingEvents = append(e.shared.pendingEvents, event)
}

func (e *directManipulationEventHandler) endGesture() {
	position := e.mousePosition()
	modifiers := currentModifiers()
	switch e.gestureKind {
	case gestureKindScroll:
		e.push(gpui.ScrollEvent{
			Position:   position,
			Delta:      gpui.ScrollDeltaPixels(gpui.Point[gpui.Pixels]{X: gpui.Px(0), Y: gpui.Px(0)}),
			Modifiers:  modifiers,
			TouchPhase: gpui.TouchPhaseEnded,
			Source:     gpui.ScrollSourcePointer,
		})
	case gestureKindPinch:
		e.push(gpui.PinchEvent{
			Position:  position,
			Delta:     0,
			Modifiers: modifiers,
			Phase:     gpui.TouchPhaseEnded,
		})
	case gestureKindNone:
	}
	e.gestureKind = gestureKindNone
}

func (e *directManipulationEventHandler) mousePosition() gpui.Point[gpui.Pixels] {
	scaleFactor := e.shared.scaleFactor
	var point win32.POINT
	_ = win32.GetCursorPos(&point)
	_ = win32.ScreenToClient(e.window, &point)
	return logicalPoint(float32(point.X), float32(point.Y), scaleFactor)
}

func (e *directManipulationEventHandler) onViewportStatusChanged(viewport *iDirectManipulationViewport, current, previous int32) com.HRESULT {
	if current == previous {
		return com.S_OK
	}

	if current == directmanipulationRunning && previous == directmanipulationInertia {
		e.endGesture()
	}

	if current == directmanipulationReady {
		e.endGesture()

		if e.lastScale != 1.0 || e.lastXOffset != 0.0 || e.lastYOffset != 0.0 {
			if viewport != nil {
				if err := viewport.zoomToRect(0, 0, float32(defaultViewportSize), float32(defaultViewportSize), false); err != nil {
					gpuiutil.LogErr(err)
				}
			}
		}

		e.lastScale = 1.0
		e.lastXOffset = 0.0
		e.lastYOffset = 0.0
	}

	return com.S_OK
}

func (e *directManipulationEventHandler) onViewportUpdated(*iDirectManipulationViewport) com.HRESULT {
	return com.S_OK
}

func (e *directManipulationEventHandler) onContentUpdated(_ *iDirectManipulationViewport, content *iDirectManipulationContent) com.HRESULT {
	if content == nil {
		return com.E_POINTER
	}

	xform := make([]float32, 6)
	if err := content.getContentTransform(xform); err != nil {
		return com.HRESULTFromError(err)
	}

	scale := xform[0]
	scaleFactor := e.shared.scaleFactor
	xOffset := xform[4] / scaleFactor
	yOffset := xform[5] / scaleFactor

	if scale == 0.0 {
		return com.S_OK
	}

	lastScale := e.lastScale
	lastX := e.lastXOffset
	lastY := e.lastYOffset

	if floatEquals(scale, lastScale) && floatEquals(xOffset, lastX) && floatEquals(yOffset, lastY) {
		return com.S_OK
	}

	position := e.mousePosition()
	modifiers := currentModifiers()

	if !floatEquals(scale, 1.0) {
		if e.gestureKind != gestureKindPinch {
			e.endGesture()
			e.gestureKind = gestureKindPinch
			e.push(gpui.PinchEvent{
				Position:  position,
				Delta:     0,
				Modifiers: modifiers,
				Phase:     gpui.TouchPhaseStarted,
			})
		}
	} else if e.gestureKind == gestureKindNone {
		e.gestureKind = gestureKindScroll
		e.scrollPhase = gpui.TouchPhaseStarted
	}

	switch e.gestureKind {
	case gestureKindScroll:
		dx := xOffset - lastX
		dy := yOffset - lastY
		touchPhase := e.scrollPhase
		e.scrollPhase = gpui.TouchPhaseMoved
		e.push(gpui.ScrollEvent{
			Position:   position,
			Delta:      gpui.ScrollDeltaPixels(gpui.Point[gpui.Pixels]{X: gpui.Px(dx), Y: gpui.Px(dy)}),
			Modifiers:  modifiers,
			TouchPhase: touchPhase,
			Source:     gpui.ScrollSourcePointer,
		})
	case gestureKindPinch:
		scaleDelta := scale / lastScale
		e.push(gpui.PinchEvent{
			Position:  position,
			Delta:     scaleDelta - 1.0,
			Modifiers: modifiers,
			Phase:     gpui.TouchPhaseMoved,
		})
	case gestureKindNone:
	}

	e.lastScale = scale
	e.lastXOffset = xOffset
	e.lastYOffset = yOffset

	return com.S_OK
}

var directManipulationEventHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
	return com.NewVtable([]com.GUID{iidIDirectManipulationViewportEventHandler},
		xwin.NewCallback(func(this *com.IUnknown, viewport *iDirectManipulationViewport, current, previous uintptr) uintptr {
			impl, ok := com.ImplOf[*directManipulationEventHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.onViewportStatusChanged(viewport, int32(current), int32(previous)).Uintptr()
		}),
		xwin.NewCallback(func(this *com.IUnknown, viewport *iDirectManipulationViewport) uintptr {
			impl, ok := com.ImplOf[*directManipulationEventHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.onViewportUpdated(viewport).Uintptr()
		}),
		xwin.NewCallback(func(this *com.IUnknown, viewport *iDirectManipulationViewport, content *iDirectManipulationContent) uintptr {
			impl, ok := com.ImplOf[*directManipulationEventHandler](this)
			if !ok {
				return com.E_FAIL.Uintptr()
			}
			return impl.onContentUpdated(viewport, content).Uintptr()
		}),
	)
})

func floatEquals(f1, f2 float32) bool {
	const epsilonScale float32 = 0.00001
	return float32(math.Abs(float64(f1-f2))) < epsilonScale*max(float32(math.Abs(float64(f1))), float32(math.Abs(float64(f2))), epsilonScale)
}
