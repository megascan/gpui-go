//go:build windows

package wgc

import (
	"errors"
	"runtime"
	"slices"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

var (
	modUser32        = windows.NewLazySystemDLL("user32.dll")
	modCombase       = windows.NewLazySystemDLL("combase.dll")
	modCoreMessaging = windows.NewLazySystemDLL("CoreMessaging.dll")
	modD3D11         = windows.NewLazySystemDLL("d3d11.dll")

	procGetWindowTextW                       = modUser32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW                 = modUser32.NewProc("GetWindowTextLengthW")
	procFindWindowW                          = modUser32.NewProc("FindWindowW")
	procEnumDisplayDevicesW                  = modUser32.NewProc("EnumDisplayDevicesW")
	procGetDisplayConfigBufferSizes          = modUser32.NewProc("GetDisplayConfigBufferSizes")
	procQueryDisplayConfig                   = modUser32.NewProc("QueryDisplayConfig")
	procDisplayConfigGetDeviceInfo           = modUser32.NewProc("DisplayConfigGetDeviceInfo")
	procRoInitialize                         = modCombase.NewProc("RoInitialize")
	procCreateDispatcherQueueController      = modCoreMessaging.NewProc("CreateDispatcherQueueController")
	procCreateDirect3D11DeviceFromDXGIDevice = modD3D11.NewProc("CreateDirect3D11DeviceFromDXGIDevice")
)

const (
	roInitMultithreaded = 1

	dqtypeThreadCurrent = 2
	dqtatComNone        = 0

	qdcOnlyActivePaths = 2

	displayconfigDeviceInfoGetSourceName = 1
	displayconfigDeviceInfoGetTargetName = 2

	dwmwaExtendedFrameBounds = 9

	wsChild = 0x40000000

	processQueryInformation = 0x0400
	processVMRead           = 0x0010

	errorInvalidThreadIDHRESULT = -2_147_023_452

	d3dFeatureLevel10_0 uint32 = 0xa000
	d3dFeatureLevel9_3  uint32 = 0x9300
	d3dFeatureLevel9_2  uint32 = 0x9200
	d3dFeatureLevel9_1  uint32 = 0x9100
)

type luid struct {
	LowPart  uint32
	HighPart int32
}

type displayconfigPathSourceInfo struct {
	AdapterID   luid
	ID          uint32
	ModeInfoIdx uint32
	StatusFlags uint32
}

type displayconfigRational struct {
	Numerator   uint32
	Denominator uint32
}

type displayconfigPathTargetInfo struct {
	AdapterID        luid
	ID               uint32
	ModeInfoIdx      uint32
	OutputTechnology uint32
	Rotation         uint32
	Scaling          uint32
	RefreshRate      displayconfigRational
	ScanLineOrdering uint32
	TargetAvailable  int32
	StatusFlags      uint32
}

type displayconfigPathInfo struct {
	SourceInfo displayconfigPathSourceInfo
	TargetInfo displayconfigPathTargetInfo
	Flags      uint32
}

type displayconfigModeInfo struct {
	InfoType  uint32
	ID        uint32
	AdapterID luid
	Union     [6]uint64
}

type displayconfigDeviceInfoHeader struct {
	Type      uint32
	Size      uint32
	AdapterID luid
	ID        uint32
}

type displayconfigSourceDeviceName struct {
	Header            displayconfigDeviceInfoHeader
	ViewGdiDeviceName [32]uint16
}

type displayconfigTargetDeviceName struct {
	Header                    displayconfigDeviceInfoHeader
	Flags                     uint32
	OutputTechnology          uint32
	EdidManufactureID         uint16
	EdidProductCodeID         uint16
	ConnectorInstance         uint32
	MonitorFriendlyDeviceName [64]uint16
	MonitorDevicePath         [128]uint16
}

type displayDeviceW struct {
	Cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type dispatcherQueueOptions struct {
	DwSize        uint32
	ThreadType    int32
	ApartmentType int32
}

func getWindowTextLengthW(hwnd win32.HWND) int32 {
	r, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	return int32(r)
}

func getWindowTextW(hwnd win32.HWND, buf []uint16) int32 {
	r, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return int32(r)
}

func findWindowW(title string) (win32.HWND, error) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}
	r, _, e := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	if r == 0 {
		if errno, ok := e.(windows.Errno); ok && errno != 0 {
			return 0, errno
		}
	}
	return win32.HWND(r), nil
}

func enumDisplayDevicesW(device *uint16, devNum uint32, info *displayDeviceW, flags uint32) bool {
	r, _, _ := procEnumDisplayDevicesW.Call(uintptr(unsafe.Pointer(device)), uintptr(devNum), uintptr(unsafe.Pointer(info)), uintptr(flags))
	return r != 0
}

func win32Err(r uintptr) error {
	if r == 0 {
		return nil
	}
	return com.HRESULTFromWin32(uint32(r))
}

func getDisplayConfigBufferSizes(flags uint32, paths, modes *uint32) error {
	r, _, _ := procGetDisplayConfigBufferSizes.Call(uintptr(flags), uintptr(unsafe.Pointer(paths)), uintptr(unsafe.Pointer(modes)))
	return win32Err(r)
}

func queryDisplayConfig(flags uint32, numPaths *uint32, paths []displayconfigPathInfo, numModes *uint32, modes []displayconfigModeInfo) error {
	var p *displayconfigPathInfo
	if len(paths) > 0 {
		p = &paths[0]
	}
	var m *displayconfigModeInfo
	if len(modes) > 0 {
		m = &modes[0]
	}
	r, _, _ := procQueryDisplayConfig.Call(uintptr(flags), uintptr(unsafe.Pointer(numPaths)), uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(numModes)), uintptr(unsafe.Pointer(m)), 0)
	runtime.KeepAlive(paths)
	runtime.KeepAlive(modes)
	return win32Err(r)
}

func displayConfigGetDeviceInfo(header *displayconfigDeviceInfoHeader) int32 {
	r, _, _ := procDisplayConfigGetDeviceInfo.Call(uintptr(unsafe.Pointer(header)))
	return int32(r)
}

func roInitialize(initType uint32) com.HRESULT {
	r, _, _ := procRoInitialize.Call(uintptr(initType))
	return com.HRESULTFromUintptr(r)
}

func createDispatcherQueueController(options dispatcherQueueOptions) (*DispatcherQueueController, error) {
	if err := procCreateDispatcherQueueController.Find(); err != nil {
		return nil, err
	}
	var out *DispatcherQueueController
	args := dispatcherQueueOptionsArgs(&options)
	args = append(args, uintptr(unsafe.Pointer(&out)))
	r, _, _ := procCreateDispatcherQueueController.Call(args...)
	runtime.KeepAlive(&options)
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func dispatcherQueueOptionsArgs(o *dispatcherQueueOptions) []uintptr {
	switch runtime.GOARCH {
	case "arm64":
		return []uintptr{uintptr(o.DwSize) | uintptr(uint32(o.ThreadType))<<32, uintptr(uint32(o.ApartmentType))}
	case "386", "arm":
		return []uintptr{uintptr(o.DwSize), uintptr(uint32(o.ThreadType)), uintptr(uint32(o.ApartmentType))}
	}
	return []uintptr{uintptr(unsafe.Pointer(o))}
}

func createDirect3D11DeviceFromDXGIDevice(dxgiDevice *com.IUnknown) (*com.IInspectable, error) {
	if err := procCreateDirect3D11DeviceFromDXGIDevice.Find(); err != nil {
		return nil, err
	}
	var out *com.IInspectable
	r, _, _ := procCreateDirect3D11DeviceFromDXGIDevice.Call(uintptr(unsafe.Pointer(dxgiDevice)), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

var errFromUTF16 = errors.New("invalid utf-16: lone surrogate found")

func utf16UntilNul(s []uint16) (string, error) {
	n := slices.Index(s, 0)
	if n < 0 {
		n = len(s)
	}
	s = s[:n]
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 0xd800 && c < 0xdc00:
			if i+1 >= len(s) || s[i+1] < 0xdc00 || s[i+1] >= 0xe000 {
				return "", errFromUTF16
			}
			i++
		case c >= 0xdc00 && c < 0xe000:
			return "", errFromUTF16
		}
	}
	return string(utf16.Decode(s)), nil
}
