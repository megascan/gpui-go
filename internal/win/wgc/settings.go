//go:build windows

package wgc

import "time"

// CaptureItemTypes is the kind of item being captured: a Monitor or a Window.
type CaptureItemTypes struct {
	// Monitor is set when a monitor is captured.
	Monitor *Monitor
	// Window is set when a window is captured.
	Window *Window
}

// TryIntoCaptureItemWithType converts a target into a GraphicsCaptureItem.
type TryIntoCaptureItemWithType interface {
	TryIntoCaptureItem() (*GraphicsCaptureItem, CaptureItemTypes, error)
}

// ColorFormat is the pixel format of captured frames; the values are DXGI formats.
type ColorFormat int32

// ColorFormat values.
const (
	ColorFormatRgba16F ColorFormat = 10
	ColorFormatRgba8   ColorFormat = 28
	ColorFormatBgra8   ColorFormat = 87
)

// DefaultColorFormat is ColorFormat's default, Rgba8.
const DefaultColorFormat = ColorFormatRgba8

// CursorCaptureSettings selects whether the cursor is captured.
type CursorCaptureSettings uint8

// CursorCaptureSettings values.
const (
	CursorCaptureSettingsDefault CursorCaptureSettings = iota
	CursorCaptureSettingsWithCursor
	CursorCaptureSettingsWithoutCursor
)

// DrawBorderSettings selects whether the capture border is drawn.
type DrawBorderSettings uint8

// DrawBorderSettings values.
const (
	DrawBorderSettingsDefault DrawBorderSettings = iota
	DrawBorderSettingsWithBorder
	DrawBorderSettingsWithoutBorder
)

// SecondaryWindowSettings selects whether secondary windows are captured.
type SecondaryWindowSettings uint8

// SecondaryWindowSettings values.
const (
	SecondaryWindowSettingsDefault SecondaryWindowSettings = iota
	SecondaryWindowSettingsInclude
	SecondaryWindowSettingsExclude
)

// MinimumUpdateIntervalSettings is the minimum interval between frames; Custom is used when
// IsCustom is set.
type MinimumUpdateIntervalSettings struct {
	// IsCustom selects Custom over the system default.
	IsCustom bool
	// Custom is the interval.
	Custom time.Duration
}

// DirtyRegionSettings selects dirty-region reporting.
type DirtyRegionSettings uint8

// DirtyRegionSettings values.
const (
	DirtyRegionSettingsDefault DirtyRegionSettings = iota
	DirtyRegionSettingsReportOnly
	DirtyRegionSettingsReportAndRender
)

// Settings configures a capture session.
type Settings[Flags any] struct {
	item                          TryIntoCaptureItemWithType
	cursorCaptureSettings         CursorCaptureSettings
	drawBorderSettings            DrawBorderSettings
	secondaryWindowSettings       SecondaryWindowSettings
	minimumUpdateIntervalSettings MinimumUpdateIntervalSettings
	dirtyRegionSettings           DirtyRegionSettings
	colorFormat                   ColorFormat
	flags                         Flags
}

// NewSettings creates a Settings.
func NewSettings[Flags any](item TryIntoCaptureItemWithType, cursorCaptureSettings CursorCaptureSettings, drawBorderSettings DrawBorderSettings, secondaryWindowSettings SecondaryWindowSettings, minimumUpdateIntervalSettings MinimumUpdateIntervalSettings, dirtyRegionSettings DirtyRegionSettings, colorFormat ColorFormat, flags Flags) Settings[Flags] {
	return Settings[Flags]{
		item:                          item,
		cursorCaptureSettings:         cursorCaptureSettings,
		drawBorderSettings:            drawBorderSettings,
		secondaryWindowSettings:       secondaryWindowSettings,
		minimumUpdateIntervalSettings: minimumUpdateIntervalSettings,
		dirtyRegionSettings:           dirtyRegionSettings,
		colorFormat:                   colorFormat,
		flags:                         flags,
	}
}

// Item returns the item to capture.
func (s *Settings[Flags]) Item() TryIntoCaptureItemWithType { return s.item }

// CursorCapture returns the cursor setting.
func (s *Settings[Flags]) CursorCapture() CursorCaptureSettings { return s.cursorCaptureSettings }

// DrawBorder returns the border setting.
func (s *Settings[Flags]) DrawBorder() DrawBorderSettings { return s.drawBorderSettings }

// ColorFormat returns the color format.
func (s *Settings[Flags]) ColorFormat() ColorFormat { return s.colorFormat }

// Flags returns the user flags.
func (s *Settings[Flags]) Flags() *Flags { return &s.flags }
