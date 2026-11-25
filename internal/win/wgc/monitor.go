//go:build windows

package wgc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/win32"
)

// Monitor errors (monitor::Error).
var (
	ErrMonitorNotFound                   = errors.New("Failed to find the specified monitor.")
	ErrMonitorNameNotFound               = errors.New("Failed to get the monitor's name.")
	ErrMonitorIndexIsLowerThanOne        = errors.New("The monitor index must be greater than zero.")
	ErrMonitorFailedToGetMonitorInfo     = errors.New("Failed to get monitor information.")
	ErrMonitorFailedToGetMonitorSettings = errors.New("Failed to get the monitor's display settings.")
	ErrMonitorFailedToGetMonitorName     = errors.New("Failed to get the monitor's device name.")
)

// MonitorError wraps the data-carrying monitor::Error variants.
type MonitorError struct {
	// Msg is the variant's message prefix.
	Msg string
	// Err is the cause.
	Err error
}

func (e *MonitorError) Error() string { return fmt.Sprintf("%s: %v", e.Msg, e.Err) }

func (e *MonitorError) Unwrap() error { return e.Err }

func monitorWindowsError(err error) error {
	return &MonitorError{Msg: "A Windows API call failed", Err: err}
}

func monitorStringError(err error) error {
	return &MonitorError{Msg: "Failed to convert a Windows string", Err: err}
}

// Monitor is a display monitor.
type Monitor struct {
	monitor win32.HMONITOR
}

// MonitorPrimary returns the monitor containing the origin.
func MonitorPrimary() (Monitor, error) {
	monitor := win32.MonitorFromPoint(win32.POINT{X: 0, Y: 0}, win32.MONITOR_DEFAULTTONULL)
	if monitor == 0 {
		return Monitor{}, ErrMonitorNotFound
	}
	return Monitor{monitor: monitor}, nil
}

// MonitorFromIndex returns the monitor at a one-based index.
func MonitorFromIndex(index int) (Monitor, error) {
	if index < 1 {
		return Monitor{}, ErrMonitorIndexIsLowerThanOne
	}
	monitors, err := MonitorEnumerate()
	if err != nil {
		return Monitor{}, err
	}
	if index-1 >= len(monitors) {
		return Monitor{}, ErrMonitorNotFound
	}
	return monitors[index-1], nil
}

// Index returns the monitor's index parsed from its device name.
func (m Monitor) Index() (int, error) {
	deviceName, err := m.DeviceName()
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(strings.ReplaceAll(deviceName, `\\.\DISPLAY`, ""), 10, 0)
	if err != nil {
		return 0, &MonitorError{Msg: "Failed to parse the monitor index", Err: err}
	}
	return int(n), nil
}

func (m Monitor) monitorInfo() (win32.MONITORINFOEXW, error) {
	var info win32.MONITORINFOEXW
	info.MonitorInfo.CbSize = uint32(unsafe.Sizeof(info))
	if err := win32.GetMonitorInfoW(m.monitor, (*win32.MONITORINFO)(unsafe.Pointer(&info))); err != nil {
		return info, ErrMonitorFailedToGetMonitorInfo
	}
	return info, nil
}

// Name returns the monitor's friendly name.
func (m Monitor) Name() (string, error) {
	info, err := m.monitorInfo()
	if err != nil {
		return "", err
	}
	var numberOfPaths, numberOfModes uint32
	if err := getDisplayConfigBufferSizes(qdcOnlyActivePaths, &numberOfPaths, &numberOfModes); err != nil {
		return "", monitorWindowsError(err)
	}
	paths := make([]displayconfigPathInfo, numberOfPaths)
	modes := make([]displayconfigModeInfo, numberOfModes)
	if err := queryDisplayConfig(qdcOnlyActivePaths, &numberOfPaths, paths, &numberOfModes, modes); err != nil {
		return "", monitorWindowsError(err)
	}
	for _, path := range paths {
		source := displayconfigSourceDeviceName{Header: displayconfigDeviceInfoHeader{
			Type:      displayconfigDeviceInfoGetSourceName,
			Size:      uint32(unsafe.Sizeof(displayconfigSourceDeviceName{})),
			AdapterID: path.SourceInfo.AdapterID,
			ID:        path.SourceInfo.ID,
		}}
		deviceName, err := m.DeviceName()
		if err != nil {
			return "", err
		}
		viewGdiDeviceName, err := utf16UntilNul(info.SzDevice[:])
		if err != nil {
			return "", monitorStringError(err)
		}
		if displayConfigGetDeviceInfo(&source.Header) == 0 && deviceName == viewGdiDeviceName {
			target := displayconfigTargetDeviceName{Header: displayconfigDeviceInfoHeader{
				Type:      displayconfigDeviceInfoGetTargetName,
				Size:      uint32(unsafe.Sizeof(displayconfigTargetDeviceName{})),
				AdapterID: path.SourceInfo.AdapterID,
				ID:        path.TargetInfo.ID,
			}}
			if displayConfigGetDeviceInfo(&target.Header) == 0 {
				name, err := utf16UntilNul(target.MonitorFriendlyDeviceName[:])
				if err != nil {
					return "", monitorStringError(err)
				}
				return name, nil
			}
			return "", ErrMonitorFailedToGetMonitorInfo
		}
	}
	return "", ErrMonitorNameNotFound
}

// DeviceName returns the monitor's GDI device name.
func (m Monitor) DeviceName() (string, error) {
	info, err := m.monitorInfo()
	if err != nil {
		return "", err
	}
	name, err := utf16UntilNul(info.SzDevice[:])
	if err != nil {
		return "", monitorStringError(err)
	}
	return name, nil
}

// DeviceString returns the monitor's adapter description.
func (m Monitor) DeviceString() (string, error) {
	info, err := m.monitorInfo()
	if err != nil {
		return "", err
	}
	device := displayDeviceW{Cb: uint32(unsafe.Sizeof(displayDeviceW{}))}
	if !enumDisplayDevicesW(&info.SzDevice[0], 0, &device, 0) {
		return "", ErrMonitorFailedToGetMonitorName
	}
	s, err := utf16UntilNul(device.DeviceString[:])
	if err != nil {
		return "", monitorStringError(err)
	}
	return s, nil
}

func (m Monitor) currentSettings() (win32.DEVMODEW, error) {
	var mode win32.DEVMODEW
	mode.DmSize = uint16(unsafe.Sizeof(mode))
	deviceName, err := m.DeviceName()
	if err != nil {
		return mode, err
	}
	name, err := windows.UTF16PtrFromString(deviceName)
	if err != nil {
		return mode, monitorWindowsError(err)
	}
	if !win32.EnumDisplaySettingsW(name, win32.ENUM_CURRENT_SETTINGS, &mode) {
		return mode, ErrMonitorFailedToGetMonitorSettings
	}
	return mode, nil
}

// RefreshRate returns the monitor's refresh rate in hertz.
func (m Monitor) RefreshRate() (uint32, error) {
	mode, err := m.currentSettings()
	return mode.DmDisplayFrequency, err
}

// Width returns the monitor's width in pixels.
func (m Monitor) Width() (uint32, error) {
	mode, err := m.currentSettings()
	return mode.DmPelsWidth, err
}

// Height returns the monitor's height in pixels.
func (m Monitor) Height() (uint32, error) {
	mode, err := m.currentSettings()
	return mode.DmPelsHeight, err
}

var (
	monitorEnum         win32.Handles[*[]Monitor]
	monitorCallbackOnce sync.Once
	monitorCallback     uintptr
)

func enumMonitorsCallback() uintptr {
	monitorCallbackOnce.Do(func() {
		monitorCallback = windows.NewCallback(func(monitor, _, _, data uintptr) uintptr {
			if monitors, ok := monitorEnum.Get(data); ok {
				*monitors = append(*monitors, Monitor{monitor: win32.HMONITOR(monitor)})
			}
			return 1
		})
	})
	return monitorCallback
}

// MonitorEnumerate returns every monitor.
func MonitorEnumerate() ([]Monitor, error) {
	var monitors []Monitor
	key := monitorEnum.Insert(&monitors)
	defer monitorEnum.Take(key)
	if err := win32.EnumDisplayMonitors(0, nil, enumMonitorsCallback(), win32.LPARAM(key)); err != nil {
		return nil, monitorWindowsError(err)
	}
	return monitors, nil
}

// MonitorFromRawHMonitor wraps an HMONITOR.
func MonitorFromRawHMonitor(monitor uintptr) Monitor {
	return Monitor{monitor: win32.HMONITOR(monitor)}
}

// AsRawHMonitor returns the HMONITOR.
func (m Monitor) AsRawHMonitor() uintptr { return uintptr(m.monitor) }

// TryIntoCaptureItem creates a capture item for the monitor.
func (m Monitor) TryIntoCaptureItem() (*GraphicsCaptureItem, CaptureItemTypes, error) {
	interop, err := GraphicsCaptureItemInteropFactory()
	if err != nil {
		return nil, CaptureItemTypes{}, err
	}
	defer interop.Release()
	item, err := interop.CreateForMonitor(m.monitor)
	if err != nil {
		return nil, CaptureItemTypes{}, err
	}
	return item, CaptureItemTypes{Monitor: &m}, nil
}
