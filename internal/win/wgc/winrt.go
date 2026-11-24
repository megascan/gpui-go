//go:build windows

package wgc

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/d3d"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const (
	classGraphicsCaptureItem        = "Windows.Graphics.Capture.GraphicsCaptureItem"
	classDirect3D11CaptureFramePool = "Windows.Graphics.Capture.Direct3D11CaptureFramePool"
	classGraphicsCaptureSession     = "Windows.Graphics.Capture.GraphicsCaptureSession"
	classApiInformation             = "Windows.Foundation.Metadata.ApiInformation"
	inspectableSlot                 = com.IInspectableSlots
)

// Interface ids of the Windows Runtime and interop interfaces the capture uses.
var (
	IID_IGraphicsCaptureItem                = com.MustGUID("{79c3f95b-31f7-4ec2-a464-632ef5d30760}")
	IID_IGraphicsCaptureItemInterop         = com.MustGUID("{3628e81b-3cac-4c60-b7f4-23ce0e0c3356}")
	IID_IDirect3D11CaptureFrame             = com.MustGUID("{fa50c623-38da-4b32-acf3-fa9734ad800e}")
	IID_IDirect3D11CaptureFramePool         = com.MustGUID("{24eb6d22-1975-422e-82e7-780dbd8ddf24}")
	IID_IDirect3D11CaptureFramePoolStatics  = com.MustGUID("{7784056a-67aa-4d53-ae54-1088d5a8ca21}")
	IID_IGraphicsCaptureSession             = com.MustGUID("{814e42a9-f70f-4ad7-939b-fddcc6eb880d}")
	IID_IGraphicsCaptureSession2            = com.MustGUID("{2c39ae40-7d2e-5044-804e-8b6799d4cf9e}")
	IID_IGraphicsCaptureSession3            = com.MustGUID("{f2cdd966-22ae-5ea1-9596-3a289344c3be}")
	IID_IGraphicsCaptureSession4            = com.MustGUID("{ae99813c-c257-5759-8ed0-668c9b557ed4}")
	IID_IGraphicsCaptureSession5            = com.MustGUID("{67c0ea62-1f85-5061-925a-239be0ac09cb}")
	IID_IGraphicsCaptureSession6            = com.MustGUID("{d7419236-be20-5e9f-bcd6-c4e98fd6afdc}")
	IID_IGraphicsCaptureSessionStatics      = com.MustGUID("{2224a540-5974-49aa-b232-0882536f4cb5}")
	IID_IApiInformationStatics              = com.MustGUID("{997439fe-f681-4a11-b416-c13a47e8ba36}")
	IID_IClosable                           = com.MustGUID("{30d5a829-7fa4-4026-83bb-d75bae4ea99e}")
	IID_IDirect3DDevice                     = com.MustGUID("{a37624ab-8d5f-4650-9d3e-9eae3d9bc670}")
	IID_IDirect3DSurface                    = com.MustGUID("{0bf4a146-13c1-4694-bee3-7abf15eaf586}")
	IID_IDirect3DDxgiInterfaceAccess        = com.MustGUID("{a9b3d012-3df2-4ee3-b8d1-8695f457d3c1}")
	IID_IDispatcherQueueController          = com.MustGUID("{22f34e66-50db-4e36-a98d-61c01b384d20}")
	IID_IAsyncAction                        = com.MustGUID("{5a648006-843a-4da9-865b-9d26e5dfad7b}")
	IID_AsyncActionCompletedHandler         = com.MustGUID("{a4ed5c81-76c9-40bd-8be6-b1d90fb20ae7}")
	IID_TypedEventHandlerFramePoolInspect   = com.MustGUID("{51a947f7-79cf-5a3e-a3a5-1289cfa6dfe8}")
	IID_TypedEventHandlerCaptureItemInspect = com.MustGUID("{e9c610c0-a68c-5bd9-8021-8589346eeee2}")
)

// SizeInt32 is Windows.Graphics.SizeInt32.
type SizeInt32 struct {
	Width  int32
	Height int32
}

func (s SizeInt32) abi() uintptr {
	return uintptr(uint32(s.Width)) | uintptr(uint32(s.Height))<<32
}

// TimeSpan is Windows.Foundation.TimeSpan, in 100-nanosecond ticks.
type TimeSpan struct {
	Duration int64
}

// DirectXPixelFormat is Windows.Graphics.DirectX.DirectXPixelFormat.
type DirectXPixelFormat int32

// GraphicsCaptureDirtyRegionMode is Windows.Graphics.Capture.GraphicsCaptureDirtyRegionMode.
type GraphicsCaptureDirtyRegionMode int32

// GraphicsCaptureDirtyRegionMode values.
const (
	GraphicsCaptureDirtyRegionModeReportOnly      GraphicsCaptureDirtyRegionMode = 0
	GraphicsCaptureDirtyRegionModeReportAndRender GraphicsCaptureDirtyRegionMode = 1
)

// EventRegistrationToken identifies an event handler registration.
type EventRegistrationToken = int64

func factory[T any](class string, iid *com.GUID) (*T, error) {
	f, err := com.RoGetActivationFactory(class, iid)
	if err != nil {
		return nil, err
	}
	return (*T)(unsafe.Pointer(f)), nil
}

func closeObject(u *com.IUnknown) error {
	c, err := u.QueryInterface(&IID_IClosable)
	if err != nil {
		return err
	}
	defer c.Release()
	return com.CallHR(c.Ptr(), inspectableSlot)
}

// GraphicsCaptureItemInterop is IGraphicsCaptureItemInterop.
type GraphicsCaptureItemInterop struct{ com.IUnknown }

// GraphicsCaptureItemInteropFactory returns GraphicsCaptureItem's IGraphicsCaptureItemInterop factory.
func GraphicsCaptureItemInteropFactory() (*GraphicsCaptureItemInterop, error) {
	return factory[GraphicsCaptureItemInterop](classGraphicsCaptureItem, &IID_IGraphicsCaptureItemInterop)
}

// CreateForWindow creates a capture item for a window.
func (i *GraphicsCaptureItemInterop) CreateForWindow(hwnd win32.HWND) (*GraphicsCaptureItem, error) {
	var out *GraphicsCaptureItem
	if err := com.CallHR(i.Ptr(), com.IUnknownSlots, uintptr(hwnd), uintptr(unsafe.Pointer(&IID_IGraphicsCaptureItem)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateForMonitor creates a capture item for a monitor.
func (i *GraphicsCaptureItemInterop) CreateForMonitor(monitor win32.HMONITOR) (*GraphicsCaptureItem, error) {
	var out *GraphicsCaptureItem
	if err := com.CallHR(i.Ptr(), com.IUnknownSlots+1, uintptr(monitor), uintptr(unsafe.Pointer(&IID_IGraphicsCaptureItem)), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// GraphicsCaptureItem is Windows.Graphics.Capture.GraphicsCaptureItem.
type GraphicsCaptureItem struct{ com.IInspectable }

// Size returns the item's size.
func (i *GraphicsCaptureItem) Size() (SizeInt32, error) {
	var s SizeInt32
	err := com.CallHR(i.Ptr(), inspectableSlot+1, uintptr(unsafe.Pointer(&s)))
	return s, err
}

// Closed registers handler for the item's Closed event.
func (i *GraphicsCaptureItem) Closed(handler func(sender *com.IInspectable, args *com.IInspectable) error) (EventRegistrationToken, error) {
	d := newTypedEventHandler(captureItemHandlerVtable.Get(), handler)
	defer d.Release()
	var token EventRegistrationToken
	err := com.CallHR(i.Ptr(), inspectableSlot+2, uintptr(unsafe.Pointer(d.Interface(0))), uintptr(unsafe.Pointer(&token)))
	return token, err
}

// RemoveClosed removes a Closed handler.
func (i *GraphicsCaptureItem) RemoveClosed(token EventRegistrationToken) error {
	return com.CallHR(i.Ptr(), inspectableSlot+3, uintptr(token))
}

// Direct3D11CaptureFramePool is Windows.Graphics.Capture.Direct3D11CaptureFramePool.
type Direct3D11CaptureFramePool struct{ com.IInspectable }

// Direct3D11CaptureFramePoolCreate is Direct3D11CaptureFramePool::Create.
func Direct3D11CaptureFramePoolCreate(device *IDirect3DDevice, format DirectXPixelFormat, numberOfBuffers int32, size SizeInt32) (*Direct3D11CaptureFramePool, error) {
	statics, err := factory[com.IInspectable](classDirect3D11CaptureFramePool, &IID_IDirect3D11CaptureFramePoolStatics)
	if err != nil {
		return nil, err
	}
	defer statics.Release()
	var out *Direct3D11CaptureFramePool
	if err := com.CallHR(statics.Ptr(), inspectableSlot, uintptr(unsafe.Pointer(device)), uintptr(format), uintptr(numberOfBuffers), size.abi(), uintptr(unsafe.Pointer(&out))); err != nil {
		return nil, err
	}
	return out, nil
}

// Clone adds a reference and returns p.
func (p *Direct3D11CaptureFramePool) Clone() *Direct3D11CaptureFramePool {
	p.AddRef()
	return p
}

// Recreate recreates the pool's buffers.
func (p *Direct3D11CaptureFramePool) Recreate(device *IDirect3DDevice, format DirectXPixelFormat, numberOfBuffers int32, size SizeInt32) error {
	return com.CallHR(p.Ptr(), inspectableSlot, uintptr(unsafe.Pointer(device)), uintptr(format), uintptr(numberOfBuffers), size.abi())
}

// TryGetNextFrame returns the next frame, or nil when none is ready.
func (p *Direct3D11CaptureFramePool) TryGetNextFrame() (*Direct3D11CaptureFrame, error) {
	var out *Direct3D11CaptureFrame
	err := com.CallHR(p.Ptr(), inspectableSlot+1, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// FrameArrived registers handler for the pool's FrameArrived event.
func (p *Direct3D11CaptureFramePool) FrameArrived(handler func(sender *com.IInspectable, args *com.IInspectable) error) (EventRegistrationToken, error) {
	d := newTypedEventHandler(framePoolHandlerVtable.Get(), handler)
	defer d.Release()
	var token EventRegistrationToken
	err := com.CallHR(p.Ptr(), inspectableSlot+2, uintptr(unsafe.Pointer(d.Interface(0))), uintptr(unsafe.Pointer(&token)))
	return token, err
}

// RemoveFrameArrived removes a FrameArrived handler.
func (p *Direct3D11CaptureFramePool) RemoveFrameArrived(token EventRegistrationToken) error {
	return com.CallHR(p.Ptr(), inspectableSlot+3, uintptr(token))
}

// CreateCaptureSession creates a session capturing item into the pool.
func (p *Direct3D11CaptureFramePool) CreateCaptureSession(item *GraphicsCaptureItem) (*GraphicsCaptureSession, error) {
	var out *GraphicsCaptureSession
	err := com.CallHR(p.Ptr(), inspectableSlot+4, uintptr(unsafe.Pointer(item)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// Close closes the pool (IClosable::Close).
func (p *Direct3D11CaptureFramePool) Close() error { return closeObject(&p.IUnknown) }

// Direct3D11CaptureFrame is Windows.Graphics.Capture.Direct3D11CaptureFrame.
type Direct3D11CaptureFrame struct{ com.IInspectable }

// Surface returns the frame's surface.
func (f *Direct3D11CaptureFrame) Surface() (*IDirect3DSurface, error) {
	var out *IDirect3DSurface
	err := com.CallHR(f.Ptr(), inspectableSlot, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// SystemRelativeTime returns the frame's QPC-based timestamp.
func (f *Direct3D11CaptureFrame) SystemRelativeTime() (TimeSpan, error) {
	var t TimeSpan
	err := com.CallHR(f.Ptr(), inspectableSlot+1, uintptr(unsafe.Pointer(&t)))
	return t, err
}

// ContentSize returns the size of the captured content.
func (f *Direct3D11CaptureFrame) ContentSize() (SizeInt32, error) {
	var s SizeInt32
	err := com.CallHR(f.Ptr(), inspectableSlot+2, uintptr(unsafe.Pointer(&s)))
	return s, err
}

// GraphicsCaptureSession is Windows.Graphics.Capture.GraphicsCaptureSession.
type GraphicsCaptureSession struct{ com.IInspectable }

// StartCapture starts the capture.
func (s *GraphicsCaptureSession) StartCapture() error {
	return com.CallHR(s.Ptr(), inspectableSlot)
}

func (s *GraphicsCaptureSession) setOn(iid *com.GUID, arg uintptr) error {
	i, err := s.QueryInterface(iid)
	if err != nil {
		return err
	}
	defer i.Release()
	return com.CallHR(i.Ptr(), inspectableSlot+1, arg)
}

func boolArg(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

// SetIsCursorCaptureEnabled is IGraphicsCaptureSession2::SetIsCursorCaptureEnabled.
func (s *GraphicsCaptureSession) SetIsCursorCaptureEnabled(v bool) error {
	return s.setOn(&IID_IGraphicsCaptureSession2, boolArg(v))
}

// SetIsBorderRequired is IGraphicsCaptureSession3::SetIsBorderRequired.
func (s *GraphicsCaptureSession) SetIsBorderRequired(v bool) error {
	return s.setOn(&IID_IGraphicsCaptureSession3, boolArg(v))
}

// SetDirtyRegionMode is IGraphicsCaptureSession4::SetDirtyRegionMode.
func (s *GraphicsCaptureSession) SetDirtyRegionMode(v GraphicsCaptureDirtyRegionMode) error {
	return s.setOn(&IID_IGraphicsCaptureSession4, uintptr(v))
}

// SetMinUpdateInterval is IGraphicsCaptureSession5::SetMinUpdateInterval.
func (s *GraphicsCaptureSession) SetMinUpdateInterval(v TimeSpan) error {
	return s.setOn(&IID_IGraphicsCaptureSession5, uintptr(v.Duration))
}

// SetIncludeSecondaryWindows is IGraphicsCaptureSession6::SetIncludeSecondaryWindows.
func (s *GraphicsCaptureSession) SetIncludeSecondaryWindows(v bool) error {
	return s.setOn(&IID_IGraphicsCaptureSession6, boolArg(v))
}

// Close closes the session (IClosable::Close).
func (s *GraphicsCaptureSession) Close() error { return closeObject(&s.IUnknown) }

// GraphicsCaptureSessionIsSupported is GraphicsCaptureSession::IsSupported.
func GraphicsCaptureSessionIsSupported() (bool, error) {
	statics, err := factory[com.IInspectable](classGraphicsCaptureSession, &IID_IGraphicsCaptureSessionStatics)
	if err != nil {
		return false, err
	}
	defer statics.Release()
	var v uint8
	err = com.CallHR(statics.Ptr(), inspectableSlot, uintptr(unsafe.Pointer(&v)))
	return v != 0, err
}

func apiInformation() (*com.IInspectable, error) {
	return factory[com.IInspectable](classApiInformation, &IID_IApiInformationStatics)
}

// ApiInformationIsApiContractPresentByMajor is ApiInformation::IsApiContractPresentByMajor.
func ApiInformationIsApiContractPresentByMajor(contractName string, majorVersion uint16) (bool, error) {
	statics, err := apiInformation()
	if err != nil {
		return false, err
	}
	defer statics.Release()
	name, err := com.NewHSTRING(contractName)
	if err != nil {
		return false, err
	}
	defer name.Delete()
	var v uint8
	err = com.CallHR(statics.Ptr(), inspectableSlot+8, uintptr(name), uintptr(majorVersion), uintptr(unsafe.Pointer(&v)))
	return v != 0, err
}

// ApiInformationIsPropertyPresent is ApiInformation::IsPropertyPresent.
func ApiInformationIsPropertyPresent(typeName, propertyName string) (bool, error) {
	statics, err := apiInformation()
	if err != nil {
		return false, err
	}
	defer statics.Release()
	t, err := com.NewHSTRING(typeName)
	if err != nil {
		return false, err
	}
	defer t.Delete()
	p, err := com.NewHSTRING(propertyName)
	if err != nil {
		return false, err
	}
	defer p.Delete()
	var v uint8
	err = com.CallHR(statics.Ptr(), inspectableSlot+4, uintptr(t), uintptr(p), uintptr(unsafe.Pointer(&v)))
	return v != 0, err
}

// IDirect3DDevice is Windows.Graphics.DirectX.Direct3D11.IDirect3DDevice.
type IDirect3DDevice struct{ com.IInspectable }

// Clone adds a reference and returns d.
func (d *IDirect3DDevice) Clone() *IDirect3DDevice {
	d.AddRef()
	return d
}

// IDirect3DSurface is Windows.Graphics.DirectX.Direct3D11.IDirect3DSurface.
type IDirect3DSurface struct{ com.IInspectable }

// IDirect3DDxgiInterfaceAccess is the interop interface reaching a surface's DXGI object.
type IDirect3DDxgiInterfaceAccess struct{ com.IUnknown }

// GetInterface returns the underlying DXGI or D3D11 interface iid.
func (a *IDirect3DDxgiInterfaceAccess) GetInterface(iid *com.GUID) (*com.IUnknown, error) {
	var out *com.IUnknown
	err := com.CallHR(a.Ptr(), com.IUnknownSlots, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// SurfaceTexture returns the ID3D11Texture2D behind a surface.
func SurfaceTexture(surface *IDirect3DSurface) (*d3d.ID3D11Texture2D, error) {
	access, err := com.Cast[IDirect3DDxgiInterfaceAccess](&surface.IUnknown, &IID_IDirect3DDxgiInterfaceAccess)
	if err != nil {
		return nil, err
	}
	defer access.Release()
	t, err := access.GetInterface(&d3d.IID_ID3D11Texture2D)
	if err != nil {
		return nil, err
	}
	return (*d3d.ID3D11Texture2D)(unsafe.Pointer(t)), nil
}

// DispatcherQueueController is Windows.System.DispatcherQueueController.
type DispatcherQueueController struct{ com.IInspectable }

// ShutdownQueueAsync shuts the queue down.
func (c *DispatcherQueueController) ShutdownQueueAsync() (*IAsyncAction, error) {
	var out *IAsyncAction
	err := com.CallHR(c.Ptr(), inspectableSlot+1, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IAsyncAction is Windows.Foundation.IAsyncAction.
type IAsyncAction struct{ com.IInspectable }

// SetCompleted sets the completion handler.
func (a *IAsyncAction) SetCompleted(handler func(asyncInfo *IAsyncAction, status int32) error) error {
	d := com.NewObject(&asyncActionCompletedHandler{f: handler}, asyncActionHandlerVtable.Get())
	defer d.Release()
	return com.CallHR(a.Ptr(), inspectableSlot, uintptr(unsafe.Pointer(d.Interface(0))))
}

type typedEventHandler struct {
	f func(sender *com.IInspectable, args *com.IInspectable) error
}

type asyncActionCompletedHandler struct {
	f func(asyncInfo *IAsyncAction, status int32) error
}

type delegateThunks struct {
	typedInvoke uintptr
	asyncInvoke uintptr
}

var (
	delegateThunksOnce sync.Once
	delegateThunksVal  delegateThunks
)

func thunks() delegateThunks {
	delegateThunksOnce.Do(func() {
		delegateThunksVal = delegateThunks{
			typedInvoke: windows.NewCallback(func(this *com.IUnknown, sender, args uintptr) uintptr {
				h, ok := com.ImplOf[*typedEventHandler](this)
				if !ok {
					return com.E_POINTER.Uintptr()
				}
				return com.HRESULTFromError(h.f((*com.IInspectable)(com.PtrFromUintptr(sender)), (*com.IInspectable)(com.PtrFromUintptr(args)))).Uintptr()
			}),
			asyncInvoke: windows.NewCallback(func(this *com.IUnknown, info, status uintptr) uintptr {
				h, ok := com.ImplOf[*asyncActionCompletedHandler](this)
				if !ok {
					return com.E_POINTER.Uintptr()
				}
				return com.HRESULTFromError(h.f((*IAsyncAction)(com.PtrFromUintptr(info)), int32(status))).Uintptr()
			}),
		}
	})
	return delegateThunksVal
}

var (
	framePoolHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
		return com.NewVtable([]com.GUID{IID_TypedEventHandlerFramePoolInspect, com.IID_IAgileObject}, thunks().typedInvoke)
	})
	captureItemHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
		return com.NewVtable([]com.GUID{IID_TypedEventHandlerCaptureItemInspect, com.IID_IAgileObject}, thunks().typedInvoke)
	})
	asyncActionHandlerVtable = com.NewLazyVtable(func() *com.Vtable {
		return com.NewVtable([]com.GUID{IID_AsyncActionCompletedHandler, com.IID_IAgileObject}, thunks().asyncInvoke)
	})
)

func newTypedEventHandler(vt *com.Vtable, f func(sender *com.IInspectable, args *com.IInspectable) error) *com.Object {
	return com.NewObject(&typedEventHandler{f: f}, vt)
}
