//go:build windows

package win32

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// GUID is a COM GUID.
type GUID = windows.GUID

// FILETIME is a 64-bit count of 100-nanosecond intervals.
type FILETIME = windows.Filetime

// POINT is a point in integer coordinates.
type POINT struct {
	X int32
	Y int32
}

// POINTL is POINT with LONG fields.
type POINTL = POINT

// SIZE is an extent in integer coordinates.
type SIZE struct {
	CX int32
	CY int32
}

// RECT is a rectangle by its edges.
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// MSG is a queued window message.
type MSG struct {
	Hwnd    HWND
	Message uint32
	WParam  WPARAM
	LParam  LPARAM
	Time    uint32
	Pt      POINT
}

// WNDCLASSW describes a window class for RegisterClassW.
type WNDCLASSW struct {
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         HICON
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
}

// CREATESTRUCTW is the lParam of WM_NCCREATE and WM_CREATE.
type CREATESTRUCTW struct {
	LpCreateParams uintptr
	HInstance      HINSTANCE
	HMenu          HMENU
	HwndParent     HWND
	Cy             int32
	Cx             int32
	Y              int32
	X              int32
	Style          int32
	LpszName       *uint16
	LpszClass      *uint16
	DwExStyle      uint32
}

// WINDOWPLACEMENT is a window's show state and restored, minimized and maximized positions.
type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    POINT
	PtMaxPosition    POINT
	RcNormalPosition RECT
}

// WINDOWPOS is the lParam of WM_WINDOWPOSCHANGING and WM_WINDOWPOSCHANGED.
type WINDOWPOS struct {
	Hwnd            HWND
	HwndInsertAfter HWND
	X               int32
	Y               int32
	Cx              int32
	Cy              int32
	Flags           uint32
}

// NCCALCSIZE_PARAMS is the lParam of WM_NCCALCSIZE when wParam is TRUE.
type NCCALCSIZE_PARAMS struct {
	Rgrc  [3]RECT
	Lppos *WINDOWPOS
}

// MINMAXINFO is the lParam of WM_GETMINMAXINFO.
type MINMAXINFO struct {
	PtReserved     POINT
	PtMaxSize      POINT
	PtMaxPosition  POINT
	PtMinTrackSize POINT
	PtMaxTrackSize POINT
}

// STYLESTRUCT is the lParam of WM_STYLECHANGING and WM_STYLECHANGED.
type STYLESTRUCT struct {
	StyleOld uint32
	StyleNew uint32
}

// MONITORINFO is a monitor's rectangle, work area and flags.
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// MONITORINFOEXW is MONITORINFO with the device name.
type MONITORINFOEXW struct {
	MonitorInfo MONITORINFO
	SzDevice    [CCHDEVICENAME]uint16
}

// DEVMODEW is a display device's mode, with the display variant of its unions.
type DEVMODEW struct {
	DmDeviceName         [CCHDEVICENAME]uint16
	DmSpecVersion        uint16
	DmDriverVersion      uint16
	DmSize               uint16
	DmDriverExtra        uint16
	DmFields             uint32
	DmPosition           POINTL
	DmDisplayOrientation uint32
	DmDisplayFixedOutput uint32
	DmColor              int16
	DmDuplex             int16
	DmYResolution        int16
	DmTTOption           int16
	DmCollate            int16
	DmFormName           [CCHFORMNAME]uint16
	DmLogPixels          uint16
	DmBitsPerPel         uint32
	DmPelsWidth          uint32
	DmPelsHeight         uint32
	DmDisplayFlags       uint32
	DmDisplayFrequency   uint32
	DmICMMethod          uint32
	DmICMIntent          uint32
	DmMediaType          uint32
	DmDitherType         uint32
	DmReserved1          uint32
	DmReserved2          uint32
	DmPanningWidth       uint32
	DmPanningHeight      uint32
}

// TRACKMOUSEEVENT requests WM_MOUSELEAVE and WM_MOUSEHOVER.
type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   HWND
	DwHoverTime uint32
}

// FLASHWINFO describes a FlashWindowEx request.
type FLASHWINFO struct {
	CbSize    uint32
	Hwnd      HWND
	DwFlags   uint32
	UCount    uint32
	DwTimeout uint32
}

// MOUSEINPUT is a synthesized mouse event.
type MOUSEINPUT struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// KEYBDINPUT is a synthesized keyboard event.
type KEYBDINPUT struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// HARDWAREINPUT is a synthesized event from another input device.
type HARDWAREINPUT struct {
	UMsg    uint32
	WParamL uint16
	WParamH uint16
}

// INPUT is one SendInput event; the union is sized for MOUSEINPUT, its largest member.
type INPUT struct {
	Type  uint32
	union MOUSEINPUT
}

// Mi returns the mouse variant of the union.
func (i *INPUT) Mi() *MOUSEINPUT {
	return &i.union
}

// Ki returns the keyboard variant of the union.
func (i *INPUT) Ki() *KEYBDINPUT {
	return (*KEYBDINPUT)(unsafe.Pointer(&i.union))
}

// Hi returns the hardware variant of the union.
func (i *INPUT) Hi() *HARDWAREINPUT {
	return (*HARDWAREINPUT)(unsafe.Pointer(&i.union))
}

// COMPOSITIONFORM positions an IME composition window.
type COMPOSITIONFORM struct {
	DwStyle      uint32
	PtCurrentPos POINT
	RcArea       RECT
}

// CANDIDATEFORM positions an IME candidate window.
type CANDIDATEFORM struct {
	DwIndex      uint32
	DwStyle      uint32
	PtCurrentPos POINT
	RcArea       RECT
}

// TP_CALLBACK_ENVIRON_V3 is a thread pool callback environment.
type TP_CALLBACK_ENVIRON_V3 struct {
	Version                    uint32
	Pool                       uintptr
	CleanupGroup               uintptr
	CleanupGroupCancelCallback uintptr
	RaceDll                    uintptr
	ActivationContext          uintptr
	FinalizationCallback       uintptr
	Flags                      uint32
	CallbackPriority           int32
	Size                       uint32
}

// PackedU64 is a 64-bit field at a 4-byte aligned offset of a pack(1) structure.
type PackedU64 [2]uint32

// Get returns the value.
func (p PackedU64) Get() uint64 {
	return uint64(p[0]) | uint64(p[1])<<32
}

// UNSIGNED_RATIO is an unsigned fraction (pack(1) in dwmapi.h).
type UNSIGNED_RATIO struct {
	UiNumerator   uint32
	UiDenominator uint32
}

// DWM_TIMING_INFO is DWM composition timing (pack(1) in dwmapi.h; every field happens to be
// 4-byte aligned, so the 64-bit fields are PackedU64).
type DWM_TIMING_INFO struct {
	CbSize                 uint32
	RateRefresh            UNSIGNED_RATIO
	QpcRefreshPeriod       PackedU64
	RateCompose            UNSIGNED_RATIO
	QpcVBlank              PackedU64
	CRefresh               PackedU64
	CDXRefresh             uint32
	QpcCompose             PackedU64
	CFrame                 PackedU64
	CDXPresent             uint32
	CRefreshFrame          PackedU64
	CFrameSubmitted        PackedU64
	CDXPresentSubmitted    uint32
	CFrameConfirmed        PackedU64
	CDXPresentConfirmed    uint32
	CRefreshConfirmed      PackedU64
	CDXRefreshConfirmed    uint32
	CFramesLate            PackedU64
	CFramesOutstanding     uint32
	CFrameDisplayed        PackedU64
	QpcFrameDisplayed      PackedU64
	CRefreshFrameDisplayed PackedU64
	CFrameComplete         PackedU64
	QpcFrameComplete       PackedU64
	CFramePending          PackedU64
	QpcFramePending        PackedU64
	CFramesDisplayed       PackedU64
	CFramesComplete        PackedU64
	CFramesPending         PackedU64
	CFramesAvailable       PackedU64
	CFramesDropped         PackedU64
	CFramesMissed          PackedU64
	CRefreshNextDisplayed  PackedU64
	CRefreshNextPresented  PackedU64
	CRefreshesDisplayed    PackedU64
	CRefreshesPresented    PackedU64
	CRefreshStarted        PackedU64
	CPixelsReceived        PackedU64
	CPixelsDrawn           PackedU64
	CBuffersEmpty          PackedU64
}

// MARGINS are the frame extension margins of DwmExtendFrameIntoClientArea.
type MARGINS struct {
	CxLeftWidth    int32
	CxRightWidth   int32
	CyTopHeight    int32
	CyBottomHeight int32
}

// USEROBJECTFLAGS is the UOI_FLAGS information of a window station or desktop.
type USEROBJECTFLAGS struct {
	FInherit  BOOL
	FReserved BOOL
	DwFlags   uint32
}

// POWERBROADCAST_SETTING is the lParam of PBT_POWERSETTINGCHANGE; Data has DataLength bytes.
type POWERBROADCAST_SETTING struct {
	PowerSetting GUID
	DataLength   uint32
	Data         [1]uint8
}

// SYSTEM_POWER_STATUS is the system's power status.
type SYSTEM_POWER_STATUS struct {
	ACLineStatus        uint8
	BatteryFlag         uint8
	BatteryLifePercent  uint8
	SystemStatusFlag    uint8
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// REASON_CONTEXT is the reason of a power request, with the simple-string variant of its union.
type REASON_CONTEXT struct {
	Version            uint32
	Flags              uint32
	SimpleReasonString *uint16
	reserved           [2]uintptr
}

// CREDENTIAL_ATTRIBUTEW is an application-defined credential attribute.
type CREDENTIAL_ATTRIBUTEW struct {
	Keyword   *uint16
	Flags     uint32
	ValueSize uint32
	Value     *byte
}

// CREDENTIALW is a Credential Manager credential.
type CREDENTIALW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        FILETIME
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         *CREDENTIAL_ATTRIBUTEW
	TargetAlias        *uint16
	UserName           *uint16
}

// FORMATETC is an OLE clipboard / drag and drop data format.
type FORMATETC struct {
	CfFormat uint16
	Ptd      uintptr
	DwAspect uint32
	Lindex   int32
	Tymed    uint32
}

// STGMEDIUM is an OLE storage medium; Data is the hGlobal / pointer variant of its union.
type STGMEDIUM struct {
	Tymed          uint32
	Data           uintptr
	PUnkForRelease uintptr
}

// PROPERTYKEY identifies a shell property.
type PROPERTYKEY struct {
	Fmtid GUID
	Pid   uint32
}

// COMDLG_FILTERSPEC is one file type filter of a common item dialog.
type COMDLG_FILTERSPEC struct {
	PszName *uint16
	PszSpec *uint16
}

// OSVERSIONINFOW is the version information RtlGetVersion fills.
type OSVERSIONINFOW = windows.OsVersionInfoEx
