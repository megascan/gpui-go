//go:build windows

package wgc

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/d3d"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// Graphics capture errors (graphics_capture_api::Error).
var (
	ErrUnsupported                      = errors.New("The Graphics Capture API is not supported on this platform.")
	ErrCursorConfigUnsupported          = errors.New("Toggling cursor capture is not supported by the Graphics Capture API on this platform.")
	ErrBorderConfigUnsupported          = errors.New("Toggling the capture border is not supported by the Graphics Capture API on this platform.")
	ErrSecondaryWindowsUnsupported      = errors.New("Capturing secondary windows is not supported by the Graphics Capture API on this platform.")
	ErrMinimumUpdateIntervalUnsupported = errors.New("Setting a minimum update interval is not supported by the Graphics Capture API on this platform.")
	ErrDirtyRegionUnsupported           = errors.New("Dirty region tracking is not supported by the Graphics Capture API on this platform.")
	ErrAlreadyStarted                   = errors.New("The capture has already been started.")
)

// WrappedError is an error variant that carries a cause: Msg is the variant's message prefix.
type WrappedError struct {
	// Msg is the variant's message prefix.
	Msg string
	// Err is the cause.
	Err error
}

func (e *WrappedError) Error() string { return fmt.Sprintf("%s: %v", e.Msg, e.Err) }

func (e *WrappedError) Unwrap() error { return e.Err }

func apiError(err error) error {
	if err == nil {
		return nil
	}
	var d3dErr *D3D11Error
	if errors.Is(err, ErrFeatureLevelNotSatisfied) || errors.As(err, &d3dErr) {
		return &WrappedError{Msg: "DirectX error", Err: err}
	}
	var we *WindowError
	var nf *WindowNotFoundError
	if errors.As(err, &we) || errors.As(err, &nf) || errors.Is(err, ErrNoActiveWindow) || errors.Is(err, ErrWindowFailedToConvertWindowsString) {
		return &WrappedError{Msg: "Window error", Err: err}
	}
	return &WrappedError{Msg: "Windows API error", Err: err}
}

// InternalCaptureControl lets a frame handler stop the capture.
type InternalCaptureControl struct {
	stop *atomic.Bool
}

// NewInternalCaptureControl wraps a stop flag.
func NewInternalCaptureControl(stop *atomic.Bool) InternalCaptureControl {
	return InternalCaptureControl{stop: stop}
}

// Stop asks the capture to stop after this frame.
func (c InternalCaptureControl) Stop() { c.stop.Store(true) }

type lockedHandler struct {
	mu sync.Mutex
	h  GraphicsCaptureApiHandler
}

type resultSlot struct {
	mu  sync.Mutex
	err error
}

func (r *resultSlot) set(err error) {
	r.mu.Lock()
	r.err = err
	r.mu.Unlock()
}

func (r *resultSlot) take() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.err
	r.err = nil
	return err
}

// GraphicsCaptureApi is a capture session on a frame pool with its event handlers.
type GraphicsCaptureApi struct {
	item                    *GraphicsCaptureItem
	d3dDevice               *d3d.ID3D11Device
	direct3dDevice          *IDirect3DDevice
	d3dDeviceContext        *d3d.ID3D11DeviceContext
	framePool               *Direct3D11CaptureFramePool
	session                 *GraphicsCaptureSession
	halt                    *atomic.Bool
	active                  bool
	captureClosedEventToken int64
	frameArrivedEventToken  int64
	released                bool
}

func postQuit(threadID uint32) error {
	return win32.PostThreadMessageW(threadID, win32.WM_QUIT, 0, 0)
}

// NewGraphicsCaptureApi creates the frame pool and session for item and registers the handlers.
// It takes ownership of d3dDevice, d3dDeviceContext and item.
func NewGraphicsCaptureApi(d3dDevice *d3d.ID3D11Device, d3dDeviceContext *d3d.ID3D11DeviceContext, item *GraphicsCaptureItem, itemType CaptureItemTypes, callback *lockedHandler, cursorCaptureSettings CursorCaptureSettings, drawBorderSettings DrawBorderSettings, secondaryWindowSettings SecondaryWindowSettings, minimumUpdateIntervalSettings MinimumUpdateIntervalSettings, dirtyRegionSettings DirtyRegionSettings, colorFormat ColorFormat, threadID uint32, result *resultSlot) (api *GraphicsCaptureApi, err error) {
	fail := func(e error) (*GraphicsCaptureApi, error) {
		item.Release()
		d3dDevice.Release()
		d3dDeviceContext.Release()
		return nil, e
	}
	check := func(supported func() (bool, error), unsupported error) error {
		ok, err := supported()
		if err != nil {
			return err
		}
		if !ok {
			return unsupported
		}
		return nil
	}
	if err := check(GraphicsCaptureApiIsSupported, ErrUnsupported); err != nil {
		return fail(err)
	}
	if cursorCaptureSettings != CursorCaptureSettingsDefault {
		if err := check(GraphicsCaptureApiIsCursorSettingsSupported, ErrCursorConfigUnsupported); err != nil {
			return fail(err)
		}
	}
	if drawBorderSettings != DrawBorderSettingsDefault {
		if err := check(GraphicsCaptureApiIsBorderSettingsSupported, ErrBorderConfigUnsupported); err != nil {
			return fail(err)
		}
	}
	if secondaryWindowSettings != SecondaryWindowSettingsDefault {
		if err := check(GraphicsCaptureApiIsSecondaryWindowsSupported, ErrSecondaryWindowsUnsupported); err != nil {
			return fail(err)
		}
	}
	if minimumUpdateIntervalSettings.IsCustom {
		if err := check(GraphicsCaptureApiIsMinimumUpdateIntervalSupported, ErrMinimumUpdateIntervalUnsupported); err != nil {
			return fail(err)
		}
	}
	if dirtyRegionSettings != DirtyRegionSettingsDefault {
		if err := check(GraphicsCaptureApiIsDirtyRegionSupported, ErrDirtyRegionUnsupported); err != nil {
			return fail(err)
		}
	}

	var titleBarHeight *uint32
	if itemType.Window != nil {
		h, err := itemType.Window.TitleBarHeight()
		if err != nil {
			return fail(apiError(err))
		}
		titleBarHeight = &h
	}

	direct3dDevice, err := CreateDirect3DDevice(d3dDevice)
	if err != nil {
		return fail(apiError(err))
	}
	pixelFormat := DirectXPixelFormat(colorFormat)
	itemSize, err := item.Size()
	if err != nil {
		direct3dDevice.Release()
		return fail(apiError(err))
	}
	framePool, err := Direct3D11CaptureFramePoolCreate(direct3dDevice, pixelFormat, 1, itemSize)
	if err != nil {
		direct3dDevice.Release()
		return fail(apiError(err))
	}
	session, err := framePool.CreateCaptureSession(item)
	if err != nil {
		framePool.Release()
		direct3dDevice.Release()
		return fail(apiError(err))
	}

	g := &GraphicsCaptureApi{
		item:             item,
		d3dDevice:        d3dDevice,
		direct3dDevice:   direct3dDevice,
		d3dDeviceContext: d3dDeviceContext,
		framePool:        framePool,
		session:          session,
		halt:             new(atomic.Bool),
	}
	defer func() {
		if err != nil {
			g.Release()
		}
	}()

	buffer := make([]byte, 3840*2160*4)
	halt := g.halt

	g.captureClosedEventToken, err = item.Closed(func(_, _ *com.IInspectable) error {
		halt.Store(true)
		callback.mu.Lock()
		cerr := callback.h.OnClosed()
		callback.mu.Unlock()
		if cerr != nil {
			result.set(cerr)
		}
		return postQuit(threadID)
	})
	if err != nil {
		return nil, apiError(err)
	}

	lastSize, err := item.Size()
	if err != nil {
		return nil, apiError(err)
	}
	g.frameArrivedEventToken, err = framePool.FrameArrived(func(sender, _ *com.IInspectable) error {
		if halt.Load() {
			return nil
		}
		if sender == nil {
			panic("FrameArrived parameter was None this should never happen.")
		}
		pool := com.As[Direct3D11CaptureFramePool](sender)
		frame, err := pool.TryGetNextFrame()
		if err != nil {
			return err
		}
		if frame == nil {
			return com.E_POINTER
		}
		defer frame.Release()
		timestamp, err := frame.SystemRelativeTime()
		if err != nil {
			return err
		}
		frameContentSize, err := frame.ContentSize()
		if err != nil {
			return err
		}
		frameSurface, err := frame.Surface()
		if err != nil {
			return err
		}
		defer frameSurface.Release()
		frameTexture, err := SurfaceTexture(frameSurface)
		if err != nil {
			return err
		}
		defer frameTexture.Release()
		desc := frameTexture.GetDesc()

		if frameContentSize.Width != lastSize.Width || frameContentSize.Height != lastSize.Height {
			if err := framePool.Recreate(direct3dDevice, pixelFormat, 1, frameContentSize); err != nil {
				return err
			}
			lastSize = frameContentSize
			return nil
		}

		f := NewFrame(d3dDevice, frameSurface, frameTexture, timestamp, d3dDeviceContext, &buffer, desc.Width, desc.Height, colorFormat, titleBarHeight)
		defer f.Release()

		stop := new(atomic.Bool)
		callback.mu.Lock()
		herr := callback.h.OnFrameArrived(f, NewInternalCaptureControl(stop))
		callback.mu.Unlock()

		if stop.Load() || herr != nil {
			if herr != nil {
				result.set(herr)
			}
			halt.Store(true)
			if err := postQuit(threadID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, apiError(err)
	}

	if cursorCaptureSettings != CursorCaptureSettingsDefault {
		ok, err := GraphicsCaptureApiIsCursorSettingsSupported()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrCursorConfigUnsupported
		}
		switch cursorCaptureSettings {
		case CursorCaptureSettingsWithCursor:
			err = session.SetIsCursorCaptureEnabled(true)
		case CursorCaptureSettingsWithoutCursor:
			err = session.SetIsCursorCaptureEnabled(false)
		}
		if err != nil {
			return nil, apiError(err)
		}
	}
	if drawBorderSettings != DrawBorderSettingsDefault {
		ok, err := GraphicsCaptureApiIsBorderSettingsSupported()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrBorderConfigUnsupported
		}
		switch drawBorderSettings {
		case DrawBorderSettingsWithBorder:
			err = session.SetIsBorderRequired(true)
		case DrawBorderSettingsWithoutBorder:
			err = session.SetIsBorderRequired(false)
		}
		if err != nil {
			return nil, apiError(err)
		}
	}
	if secondaryWindowSettings != SecondaryWindowSettingsDefault {
		ok, err := GraphicsCaptureApiIsSecondaryWindowsSupported()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrSecondaryWindowsUnsupported
		}
		switch secondaryWindowSettings {
		case SecondaryWindowSettingsInclude:
			err = session.SetIncludeSecondaryWindows(true)
		case SecondaryWindowSettingsExclude:
			err = session.SetIncludeSecondaryWindows(false)
		}
		if err != nil {
			return nil, apiError(err)
		}
	}
	if minimumUpdateIntervalSettings.IsCustom {
		ok, err := GraphicsCaptureApiIsMinimumUpdateIntervalSupported()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrMinimumUpdateIntervalUnsupported
		}
		if err := session.SetMinUpdateInterval(TimeSpan{Duration: int64(minimumUpdateIntervalSettings.Custom / 100)}); err != nil {
			return nil, apiError(err)
		}
	}
	if dirtyRegionSettings != DirtyRegionSettingsDefault {
		ok, err := GraphicsCaptureApiIsDirtyRegionSupported()
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDirtyRegionUnsupported
		}
		switch dirtyRegionSettings {
		case DirtyRegionSettingsReportOnly:
			err = session.SetDirtyRegionMode(GraphicsCaptureDirtyRegionModeReportOnly)
		case DirtyRegionSettingsReportAndRender:
			err = session.SetDirtyRegionMode(GraphicsCaptureDirtyRegionModeReportAndRender)
		}
		if err != nil {
			return nil, apiError(err)
		}
	}
	return g, nil
}

// StartCapture starts the session.
func (a *GraphicsCaptureApi) StartCapture() error {
	if a.active {
		return ErrAlreadyStarted
	}
	if a.session == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	if err := a.session.StartCapture(); err != nil {
		return apiError(err)
	}
	a.active = true
	return nil
}

// StopCapture removes the handlers and closes the pool and session, then releases a.
func (a *GraphicsCaptureApi) StopCapture() {
	if a.framePool != nil {
		framePool := a.framePool
		a.framePool = nil
		if err := framePool.RemoveFrameArrived(a.frameArrivedEventToken); err != nil {
			panic("Failed to remove Frame Arrived event handler: " + err.Error())
		}
		if err := framePool.Close(); err != nil {
			panic("Failed to Close Frame Pool: " + err.Error())
		}
		framePool.Release()
	}
	if a.session != nil {
		session := a.session
		a.session = nil
		if err := session.Close(); err != nil {
			panic("Failed to Close Capture Session: " + err.Error())
		}
		session.Release()
	}
	if err := a.item.RemoveClosed(a.captureClosedEventToken); err != nil {
		panic("Failed to remove Capture Session Closed event handler: " + err.Error())
	}
	a.Release()
}

// HaltHandle returns the flag that stops frame delivery.
func (a *GraphicsCaptureApi) HaltHandle() *atomic.Bool { return a.halt }

// Release is the drop of GraphicsCaptureApi: it removes the handlers, closes the pool and
// session ignoring errors, and releases every object. It is idempotent.
func (a *GraphicsCaptureApi) Release() {
	if a.released {
		return
	}
	a.released = true
	if a.framePool != nil {
		_ = a.framePool.RemoveFrameArrived(a.frameArrivedEventToken)
		_ = a.framePool.Close()
		a.framePool.Release()
		a.framePool = nil
	}
	if a.session != nil {
		_ = a.session.Close()
		a.session.Release()
		a.session = nil
	}
	_ = a.item.RemoveClosed(a.captureClosedEventToken)
	a.item.Release()
	a.direct3dDevice.Release()
	a.d3dDeviceContext.Release()
	a.d3dDevice.Release()
}

func propertyPresent(property string) (bool, error) {
	ok, err := ApiInformationIsPropertyPresent("Windows.Graphics.Capture.GraphicsCaptureSession", property)
	if err != nil {
		return false, apiError(err)
	}
	if !ok {
		return false, nil
	}
	return GraphicsCaptureApiIsSupported()
}

// GraphicsCaptureApiIsSupported reports whether Windows.Graphics.Capture is available.
func GraphicsCaptureApiIsSupported() (bool, error) {
	ok, err := ApiInformationIsApiContractPresentByMajor("Windows.Foundation.UniversalApiContract", 8)
	if err != nil {
		return false, apiError(err)
	}
	if !ok {
		return false, nil
	}
	ok, err = GraphicsCaptureSessionIsSupported()
	if err != nil {
		return false, apiError(err)
	}
	return ok, nil
}

// GraphicsCaptureApiIsCursorSettingsSupported reports whether cursor capture can be toggled.
func GraphicsCaptureApiIsCursorSettingsSupported() (bool, error) {
	return propertyPresent("IsCursorCaptureEnabled")
}

// GraphicsCaptureApiIsBorderSettingsSupported reports whether the border can be toggled.
func GraphicsCaptureApiIsBorderSettingsSupported() (bool, error) {
	return propertyPresent("IsBorderRequired")
}

// GraphicsCaptureApiIsSecondaryWindowsSupported reports whether secondary windows can be included.
func GraphicsCaptureApiIsSecondaryWindowsSupported() (bool, error) {
	return propertyPresent("IncludeSecondaryWindows")
}

// GraphicsCaptureApiIsMinimumUpdateIntervalSupported reports whether the update interval can be set.
func GraphicsCaptureApiIsMinimumUpdateIntervalSupported() (bool, error) {
	return propertyPresent("MinUpdateInterval")
}

// GraphicsCaptureApiIsDirtyRegionSupported reports whether dirty regions can be tracked.
func GraphicsCaptureApiIsDirtyRegionSupported() (bool, error) {
	return propertyPresent("DirtyRegionMode")
}
