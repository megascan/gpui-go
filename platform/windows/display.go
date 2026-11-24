//go:build windows

package windows

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"syscall"
	"time"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// WindowsDisplay is a monitor as a gpui.PlatformDisplay.
type WindowsDisplay struct {
	gpui.PlatformDisplayDefaults
	Handle         win32.HMONITOR
	DisplayID      gpui.DisplayID
	scaleFactor    float32
	bounds         gpui.Bounds[gpui.Pixels]
	visibleBounds  gpui.Bounds[gpui.Pixels]
	physicalBounds gpui.Bounds[gpui.DevicePixels]
	uuid           gpui.UUID
}

var _ gpui.PlatformDisplay = (*WindowsDisplay)(nil)

// NewWindowsDisplay reads the monitor with the given id; it logs and returns false if the
// monitor is gone.
func NewWindowsDisplay(displayID gpui.DisplayID) (*WindowsDisplay, bool) {
	handle := win32.HMONITOR(uint64(displayID))
	info, ok := gpuiutil.Result(getMonitorInfo(handle)).LogErr()
	if !ok {
		return nil, false
	}
	monitorSize := info.MonitorInfo.RcMonitor
	workArea := info.MonitorInfo.RcWork
	uuid := generateUUID(info.SzDevice[:])
	scaleFactor, ok := gpuiutil.Result(getScaleFactorForMonitor(handle)).LogErr()
	if !ok {
		return nil, false
	}
	physicalSize := gpui.NewSize(
		gpui.DevicePixels(monitorSize.Right-monitorSize.Left),
		gpui.DevicePixels(monitorSize.Bottom-monitorSize.Top),
	)

	d := &WindowsDisplay{
		Handle:      handle,
		DisplayID:   displayID,
		scaleFactor: scaleFactor,
		bounds: gpui.Bounds[gpui.Pixels]{
			Origin: logicalPoint(
				float32(monitorSize.Left),
				float32(monitorSize.Top),
				scaleFactor,
			),
			Size: physicalSize.ToPixels(scaleFactor),
		},
		visibleBounds: gpui.Bounds[gpui.Pixels]{
			Origin: logicalPoint(float32(workArea.Left), float32(workArea.Top), scaleFactor),
			Size: gpui.NewSize(
				float32(workArea.Right-workArea.Left)/scaleFactor,
				float32(workArea.Bottom-workArea.Top)/scaleFactor,
			).Map(gpui.Px),
		},
		physicalBounds: gpui.Bounds[gpui.DevicePixels]{
			Origin: gpui.NewPoint(gpui.DevicePixels(monitorSize.Left), gpui.DevicePixels(monitorSize.Top)),
			Size:   physicalSize,
		},
		uuid: uuid,
	}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d, true
}

func displayIDForMonitor(monitor win32.HMONITOR) gpui.DisplayID {
	return gpui.NewDisplayID(uint64(monitor))
}

// WindowsDisplayPrimaryMonitor returns the primary monitor.
func WindowsDisplayPrimaryMonitor() (*WindowsDisplay, bool) {
	pointZero := win32.POINT{X: 0, Y: 0}
	monitor, lastErr := win32.MonitorFromPointLastError(pointZero, win32.MONITOR_DEFAULTTOPRIMARY)
	if monitor == 0 {
		slog.Error(fmt.Sprintf("can not find the primary monitor: %s", lastOSError(lastErr)))
		return nil, false
	}
	return NewWindowsDisplay(displayIDForMonitor(monitor))
}

// ScaleFactor returns the monitor's DPI scale factor, independent of whatever monitor a
// not-yet-positioned window currently happens to be on.
func (d *WindowsDisplay) ScaleFactor() float32 {
	return d.scaleFactor
}

// CheckGivenBounds reports whether the center of bounds is on this monitor.
func (d *WindowsDisplay) CheckGivenBounds(bounds gpui.Bounds[gpui.Pixels]) bool {
	center := bounds.Center()
	centerPoint := win32.POINT{
		X: int32(center.X.AsF32() * d.scaleFactor),
		Y: int32(center.Y.AsF32() * d.scaleFactor),
	}
	monitor := win32.MonitorFromPoint(centerPoint, win32.MONITOR_DEFAULTTONULL)
	if monitor == 0 {
		return false
	}
	display, ok := NewWindowsDisplay(displayIDForMonitor(monitor))
	if !ok {
		return false
	}
	return display.uuid == d.uuid
}

// WindowsDisplayDisplays returns every monitor.
func WindowsDisplayDisplays() []gpui.PlatformDisplay {
	var displays []gpui.PlatformDisplay
	for _, handle := range availableMonitors() {
		display, ok := NewWindowsDisplay(displayIDForMonitor(handle))
		if !ok {
			continue
		}
		displays = append(displays, display)
	}
	return displays
}

// PhysicalBounds returns the monitor's bounds in device pixels.
func (d *WindowsDisplay) PhysicalBounds() gpui.Bounds[gpui.DevicePixels] {
	return d.physicalBounds
}

// ID returns the display id.
func (d *WindowsDisplay) ID() gpui.DisplayID {
	return d.DisplayID
}

// UUID returns a stable id derived from the monitor's device name.
func (d *WindowsDisplay) UUID() (gpui.UUID, error) {
	return d.uuid, nil
}

// Bounds returns the monitor's bounds.
func (d *WindowsDisplay) Bounds() gpui.Bounds[gpui.Pixels] {
	return d.bounds
}

// VisibleBounds returns the monitor's work area.
func (d *WindowsDisplay) VisibleBounds() gpui.Bounds[gpui.Pixels] {
	return d.visibleBounds
}

// RefreshInterval returns the refresh interval of the monitor's current mode.
func (d *WindowsDisplay) RefreshInterval() (time.Duration, bool) {
	info, ok := gpuiutil.Result(getMonitorInfo(d.Handle)).LogErr()
	if !ok {
		return 0, false
	}
	return refreshIntervalForDevice(&info.SzDevice)
}

func refreshIntervalForDevice(deviceName *[win32.CCHDEVICENAME]uint16) (time.Duration, bool) {
	var mode win32.DEVMODEW
	mode.DmSize = uint16(unsafe.Sizeof(mode))
	found := win32.EnumDisplaySettingsW(&deviceName[0], win32.ENUM_CURRENT_SETTINGS, &mode)
	if !found {
		return 0, false
	}
	if mode.DmDisplayFrequency <= 1 {
		return 0, false
	}
	return gpui.RefreshIntervalFromHz(float64(mode.DmDisplayFrequency))
}

var (
	monitorLists        win32.Handles[*[]win32.HMONITOR]
	monitorEnumProcOnce sync.Once
	monitorEnumProcPtr  uintptr
)

func monitorEnumProcCallback() uintptr {
	monitorEnumProcOnce.Do(func() {
		monitorEnumProcPtr = xwin.NewCallback(monitorEnumProc)
	})
	return monitorEnumProcPtr
}

func availableMonitors() []win32.HMONITOR {
	monitors := make([]win32.HMONITOR, 0, 4)
	key := monitorLists.Insert(&monitors)
	defer monitorLists.Take(key)
	if err := win32.EnumDisplayMonitors(0, nil, monitorEnumProcCallback(), win32.LPARAM(key)); err != nil {
		gpuiutil.LogErr(err)
	}
	return monitors
}

func monitorEnumProc(hmonitor win32.HMONITOR, _ win32.HDC, _ *win32.RECT, data win32.LPARAM) uintptr {
	if monitors, ok := monitorLists.Get(uintptr(data)); ok {
		*monitors = append(*monitors, hmonitor)
	}
	return 1
}

func getMonitorInfo(hmonitor win32.HMONITOR) (win32.MONITORINFOEXW, error) {
	var monitorInfo win32.MONITORINFOEXW
	monitorInfo.MonitorInfo.CbSize = uint32(unsafe.Sizeof(monitorInfo))
	if err := win32.GetMonitorInfoW(hmonitor, &monitorInfo.MonitorInfo); err != nil {
		return monitorInfo, osError(err)
	}
	return monitorInfo, nil
}

var namespaceDNS = gpui.UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

func generateUUID(deviceName []uint16) gpui.UUID {
	name := make([]byte, 0, len(deviceName)*2)
	for _, a := range deviceName {
		name = binary.BigEndian.AppendUint16(name, a)
	}
	return newUUIDV5(namespaceDNS, name)
}

func newUUIDV5(namespace gpui.UUID, name []byte) gpui.UUID {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write(name)
	var u gpui.UUID
	copy(u[:], h.Sum(nil))
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}

func getScaleFactorForMonitor(monitor win32.HMONITOR) (float32, error) {
	var dpiX, dpiY uint32
	if err := win32.GetDpiForMonitor(monitor, win32.MDT_EFFECTIVE_DPI, &dpiX, &dpiY); err != nil {
		return 0, err
	}
	if dpiX != dpiY {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", dpiX, dpiY))
	}
	return float32(dpiX) / float32(win32.USER_DEFAULT_SCREEN_DPI), nil
}

type osErrorValue struct {
	errno syscall.Errno
}

func (e osErrorValue) Error() string {
	return fmt.Sprintf("%s (os error %d)", e.errno.Error(), uint32(e.errno))
}

func (e osErrorValue) Unwrap() error {
	return e.errno
}

func osError(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return osErrorValue{errno: errno}
	}
	return err
}

func lastOSError(lastErr error) error {
	var errno syscall.Errno
	if errors.As(lastErr, &errno) {
		return osErrorValue{errno: errno}
	}
	if lastErr != nil {
		return lastErr
	}
	return osErrorValue{}
}
