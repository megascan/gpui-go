//go:build windows

package d3d

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	modDComp                     = windows.NewLazySystemDLL("dcomp.dll")
	procDCompositionCreateDevice = modDComp.NewProc("DCompositionCreateDevice")
)

// DCompositionCreateDevice creates a DirectComposition device over a DXGI device.
func DCompositionCreateDevice(dxgiDevice *IDXGIDevice) (*IDCompositionDevice, error) {
	if err := procDCompositionCreateDevice.Find(); err != nil {
		return nil, err
	}
	var out *IDCompositionDevice
	r, _, _ := procDCompositionCreateDevice.Call(iptr(dxgiDevice), uintptr(unsafe.Pointer(&IID_IDCompositionDevice)), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// IDCompositionDevice is a DirectComposition device.
type IDCompositionDevice struct{ com.IUnknown }

// Commit commits pending composition changes.
func (d *IDCompositionDevice) Commit() error {
	return com.CallHR(d.Ptr(), com.IUnknownSlots)
}

// CreateTargetForHwnd creates a composition target for a window.
func (d *IDCompositionDevice) CreateTargetForHwnd(hwnd windows.HWND, topmost bool) (*IDCompositionTarget, error) {
	var out *IDCompositionTarget
	err := com.CallHR(d.Ptr(), com.IUnknownSlots+3, uintptr(hwnd), uintptr(BoolOf(topmost)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateVisual creates a visual.
func (d *IDCompositionDevice) CreateVisual() (*IDCompositionVisual, error) {
	var out *IDCompositionVisual
	err := com.CallHR(d.Ptr(), com.IUnknownSlots+4, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDCompositionTarget binds a visual tree to a window.
type IDCompositionTarget struct{ com.IUnknown }

// SetRoot sets the root visual.
func (t *IDCompositionTarget) SetRoot(visual *IDCompositionVisual) error {
	return com.CallHR(t.Ptr(), com.IUnknownSlots, iptr(visual))
}

// IDCompositionVisual is a composition visual.
type IDCompositionVisual struct{ com.IUnknown }

// SetContent sets the visual's content, such as a swap chain.
func (v *IDCompositionVisual) SetContent(content *com.IUnknown) error {
	return com.CallHR(v.Ptr(), com.IUnknownSlots+12, iptr(content))
}
