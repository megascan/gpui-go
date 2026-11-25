//go:build windows

package windows

import (
	"unsafe"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// WindowsSystemSettings are the Windows settings read from SystemParametersInfo.
type WindowsSystemSettings struct {
	MouseWheelSettings MouseWheelSettings
}

// MouseWheelSettings are the mouse wheel scroll amounts.
type MouseWheelSettings struct {
	// WheelScrollChars is SPI_GETWHEELSCROLLCHARS.
	WheelScrollChars uint32
	// WheelScrollLines is SPI_GETWHEELSCROLLLINES.
	WheelScrollLines uint32
}

// NewWindowsSystemSettings reads the current settings.
func NewWindowsSystemSettings() *WindowsSystemSettings {
	settings := &WindowsSystemSettings{}
	settings.init()
	return settings
}

func (s *WindowsSystemSettings) init() {
	s.MouseWheelSettings.update()
}

// Update rereads the settings a WM_SETTINGCHANGE with this wParam changed.
func (s *WindowsSystemSettings) Update(wparam uintptr) {
	switch uint32(wparam) {
	case win32.SPI_GETWHEELSCROLLLINES, win32.SPI_GETWHEELSCROLLCHARS:
		s.updateMouseWheelSettings()
	}
}

func (s *WindowsSystemSettings) updateMouseWheelSettings() {
	s.MouseWheelSettings.update()
}

func (m *MouseWheelSettings) update() {
	m.updateWheelScrollChars()
	m.updateWheelScrollLines()
}

func (m *MouseWheelSettings) updateWheelScrollChars() {
	var value uint32
	err := win32.SystemParametersInfoW(win32.SPI_GETWHEELSCROLLCHARS, 0, unsafe.Pointer(&value), 0)

	if _, ok := gpuiutil.Result(struct{}{}, err).LogErr(); ok && m.WheelScrollChars != value {
		m.WheelScrollChars = value
	}
}

func (m *MouseWheelSettings) updateWheelScrollLines() {
	var value uint32
	err := win32.SystemParametersInfoW(win32.SPI_GETWHEELSCROLLLINES, 0, unsafe.Pointer(&value), 0)

	if _, ok := gpuiutil.Result(struct{}{}, err).LogErr(); ok && m.WheelScrollLines != value {
		m.WheelScrollLines = value
	}
}
