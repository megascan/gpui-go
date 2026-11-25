//go:build windows

package screencap

import (
	"fmt"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/wgc"
	"github.com/megascan/gpui-go/internal/win/win32"
)

func winGetAllTargets() ([]gpui.ScapTarget, error) {
	var targets []gpui.ScapTarget

	displays, err := wgc.MonitorEnumerate()
	if err != nil {
		return nil, fmt.Errorf("Failed to enumerate monitors: %w", err)
	}
	for _, display := range displays {
		id := uint32(display.AsRawHMonitor())
		title, err := monitorTitle(display)
		if err != nil {
			return nil, fmt.Errorf("Failed to get monitor name: %w", err)
		}
		width, err := display.Width()
		if err != nil {
			return nil, err
		}
		height, err := display.Height()
		if err != nil {
			return nil, err
		}
		targets = append(targets, &gpui.ScapDisplay{
			ID:        id,
			Title:     title,
			RawHandle: display.AsRawHMonitor(),
			Width:     uint16(width),
			Height:    uint16(height),
		})
	}

	windowsList, err := wgc.WindowEnumerate()
	if err != nil {
		return nil, fmt.Errorf("Failed to enumerate windows: %w", err)
	}
	for _, window := range windowsList {
		id := uint32(window.AsRawHWND())
		title, err := window.Title()
		if err != nil {
			return nil, fmt.Errorf("Window title not found: %w", err)
		}
		targets = append(targets, &gpui.ScapWindow{
			ID:        id,
			Title:     title,
			RawHandle: window.AsRawHWND(),
		})
	}
	return targets, nil
}

func winGetMainDisplay() (*gpui.ScapDisplay, error) {
	display, err := wgc.MonitorPrimary()
	if err != nil {
		return nil, fmt.Errorf("Failed to get primary monitor: %w", err)
	}
	id := uint32(display.AsRawHMonitor())
	title, err := monitorTitle(display)
	if err != nil {
		return nil, fmt.Errorf("Failed to get monitor name: %w", err)
	}
	width, err := display.Width()
	if err != nil {
		return nil, err
	}
	height, err := display.Height()
	if err != nil {
		return nil, err
	}
	return &gpui.ScapDisplay{
		ID:        id,
		Title:     title,
		RawHandle: display.AsRawHMonitor(),
		Width:     uint16(width),
		Height:    uint16(height),
	}, nil
}

func monitorTitle(monitor wgc.Monitor) (string, error) {
	if name, err := monitor.Name(); err == nil {
		return name, nil
	}
	if s, err := monitor.DeviceString(); err == nil {
		return s, nil
	}
	return monitor.DeviceName()
}

func winGetScaleFactor(target gpui.ScapTarget) float64 {
	const baseDPI = 96
	var dpiX, dpiY uint32
	var dpi uint32
	switch t := target.(type) {
	case *gpui.ScapWindow:
		dpi = win32.GetDpiForWindow(win32.HWND(t.RawHandle))
	case *gpui.ScapDisplay:
		if win32.GetDpiForMonitor(win32.HMONITOR(t.RawHandle), win32.MDT_EFFECTIVE_DPI, &dpiX, &dpiY) == nil {
			dpi = dpiX
		} else {
			dpi = baseDPI
		}
	}
	return float64(dpi) / float64(baseDPI)
}

func winGetTargetDimensions(target gpui.ScapTarget) (uint64, uint64) {
	switch t := target.(type) {
	case *gpui.ScapWindow:
		var rect win32.RECT
		_ = win32.GetWindowRect(win32.HWND(t.RawHandle), &rect)
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		return uint64(int64(width)), uint64(int64(height))
	case *gpui.ScapDisplay:
		monitor := wgc.MonitorFromRawHMonitor(t.RawHandle)
		width, err := monitor.Width()
		if err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
		height, err := monitor.Height()
		if err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
		return uint64(width), uint64(height)
	}
	panic("screencap: unknown target type")
}
