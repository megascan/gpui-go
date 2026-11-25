//go:build darwin && !ios

package macos

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"sync"
	"time"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/cg"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const applicationServicesPath = "/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices"

type cfUUIDBytes struct {
	bytes [16]uint8
}

var (
	displayOnce sync.Once
	displayErr  error

	cgGetActiveDisplayList           func(maxDisplays uint32, activeDisplays *uint32, displayCount *uint32) int32
	cgDisplayBounds                  func(display uint32) cg.Rect
	cgDisplayCopyDisplayMode         func(display uint32) uintptr
	cgDisplayModeGetRefreshRate      func(mode uintptr) float64
	cgDisplayModeRelease             func(mode uintptr)
	cgDisplayCreateUUIDFromDisplayID func(display uint32) cf.TypeRef
	cfUUIDGetUUIDBytes               func(uuid cf.TypeRef) cfUUIDBytes
)

func loadDisplay() error {
	displayOnce.Do(func() {
		if err := objc.Load(); err != nil {
			displayErr = err
			return
		}
		cgLib, err := purego.Dlopen(cg.CoreGraphicsPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			displayErr = fmt.Errorf("macos: open %s: %w", cg.CoreGraphicsPath, err)
			return
		}
		asLib, err := purego.Dlopen(applicationServicesPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			displayErr = fmt.Errorf("macos: open %s: %w", applicationServicesPath, err)
			return
		}
		cfLib, err := purego.Dlopen(cf.CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			displayErr = fmt.Errorf("macos: open %s: %w", cf.CoreFoundationPath, err)
			return
		}
		purego.RegisterLibFunc(&cgGetActiveDisplayList, cgLib, "CGGetActiveDisplayList")
		purego.RegisterLibFunc(&cgDisplayBounds, cgLib, "CGDisplayBounds")
		purego.RegisterLibFunc(&cgDisplayCopyDisplayMode, cgLib, "CGDisplayCopyDisplayMode")
		purego.RegisterLibFunc(&cgDisplayModeGetRefreshRate, cgLib, "CGDisplayModeGetRefreshRate")
		purego.RegisterLibFunc(&cgDisplayModeRelease, cgLib, "CGDisplayModeRelease")
		purego.RegisterLibFunc(&cgDisplayCreateUUIDFromDisplayID, asLib, "CGDisplayCreateUUIDFromDisplayID")
		purego.RegisterLibFunc(&cfUUIDGetUUIDBytes, cfLib, "CFUUIDGetUUIDBytes")
	})
	return displayErr
}

func mustLoadDisplay() {
	if err := loadDisplay(); err != nil {
		panic(err)
	}
}

// MacDisplay is a CGDirectDisplayID as a gpui.PlatformDisplay.
type MacDisplay struct {
	gpui.PlatformDisplayDefaults
	displayID uint32
}

var _ gpui.PlatformDisplay = (*MacDisplay)(nil)

func newMacDisplay(displayID uint32) *MacDisplay {
	d := &MacDisplay{displayID: displayID}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d
}

// MacDisplayFindByID returns the screen with the given DisplayID.
func MacDisplayFindByID(id gpui.DisplayID) (*MacDisplay, bool) {
	for screen := range MacDisplayAll() {
		if screen.ID() == id {
			return screen, true
		}
	}
	return nil, false
}

// MacDisplayPrimary returns the primary screen: the one with the menu bar, whose bottom left
// corner is at the origin of the AppKit coordinate system.
func MacDisplayPrimary() *MacDisplay {
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("displays require the main thread")
	}
	screens := appkit.NSScreenScreens()
	if len(screens) == 0 {
		panic("no primary screen")
	}
	return newMacDisplay(screenNumber(screens[0]))
}

func screenNumber(screen appkit.NSScreen) uint32 {
	var number uint32
	objc.AutoreleasePool(func() {
		key := objc.NSStringFromGo("NSScreenNumber")
		defer objc.Release(key)
		value := objc.NSDictionaryObjectForKey(screen.DeviceDescription(), key)
		if value != objc.Nil {
			number = objc.Send[uint32](value, objc.Sel("unsignedIntValue"))
		}
	})
	return number
}

func screenHasNumber(screen appkit.NSScreen, displayID uint32) bool {
	found := false
	objc.AutoreleasePool(func() {
		key := objc.NSStringFromGo("NSScreenNumber")
		defer objc.Release(key)
		value := objc.NSDictionaryObjectForKey(screen.DeviceDescription(), key)
		found = value != objc.Nil && objc.Send[uint32](value, objc.Sel("unsignedIntValue")) == displayID
	})
	return found
}

// MacDisplayAll iterates over every currently active system display. It is empty when the system
// cannot list them, e.g. without a window server connection.
func MacDisplayAll() iter.Seq[*MacDisplay] {
	mustLoadDisplay()
	displays := make([]uint32, 32)
	var displayCount uint32
	result := cgGetActiveDisplayList(uint32(cap(displays)), &displays[0], &displayCount)
	if result == 0 {
		displays = displays[:displayCount]
	} else {
		slog.Error(fmt.Sprintf("Failed to get active display list. Result: %d", result))
		displays = displays[:0]
	}
	return func(yield func(*MacDisplay) bool) {
		for _, id := range displays {
			if !yield(newMacDisplay(id)) {
				return
			}
		}
	}
}

// DisplayID returns the CGDirectDisplayID.
func (d *MacDisplay) DisplayID() uint32 {
	return d.displayID
}

// ID returns the display's DisplayID.
func (d *MacDisplay) ID() gpui.DisplayID {
	return gpui.NewDisplayID(uint64(d.displayID))
}

// UUID returns the display's stable UUID.
func (d *MacDisplay) UUID() (gpui.UUID, error) {
	mustLoadDisplay()
	cfuuid := cgDisplayCreateUUIDFromDisplayID(d.displayID)
	if cfuuid == 0 {
		return gpui.UUID{}, errors.New("AppKit returned a null from CGDisplayCreateUUIDFromDisplayID")
	}
	bytes := cfUUIDGetUUIDBytes(cfuuid)
	cf.Release(cfuuid)
	return gpui.UUID(bytes.bytes), nil
}

// Bounds returns the display's size with a zero origin; CGDisplayBounds is in global display
// coordinates, where 0 is the top left of the primary display.
func (d *MacDisplay) Bounds() gpui.Bounds[gpui.Pixels] {
	mustLoadDisplay()
	bounds := cgDisplayBounds(d.displayID)
	return gpui.Bounds[gpui.Pixels]{
		Size: gpui.NewSize(gpui.Px(float32(bounds.Size.Width)), gpui.Px(float32(bounds.Size.Height))),
	}
}

// VisibleBounds returns the display's area without the menu bar and dock, with a top-left origin.
func (d *MacDisplay) VisibleBounds() gpui.Bounds[gpui.Pixels] {
	dominatedScreen, ok := d.getNSScreen()
	if !ok {
		return d.Bounds()
	}
	screenFrame := dominatedScreen.Frame()
	visibleFrame := dominatedScreen.VisibleFrame()
	originY := screenFrame.Size.Height - visibleFrame.Origin.Y - visibleFrame.Size.Height + screenFrame.Origin.Y
	return gpui.NewBounds(
		gpui.NewPoint(gpui.Px(float32(visibleFrame.Origin.X)-float32(screenFrame.Origin.X)), gpui.Px(float32(originY))),
		gpui.NewSize(gpui.Px(float32(visibleFrame.Size.Width)), gpui.Px(float32(visibleFrame.Size.Height))),
	)
}

// RefreshInterval returns the display's frame interval: the ProMotion maximum where the screen
// reports one, else the display mode's rate.
func (d *MacDisplay) RefreshInterval() (time.Duration, bool) {
	screenHertz := 0.0
	if screen, ok := d.getNSScreen(); ok && objc.RespondsToSelector(screen.ID(), objc.Sel("maximumFramesPerSecond")) {
		screenHertz = float64(screen.MaximumFramesPerSecond())
	}
	if interval, ok := gpui.RefreshIntervalFromHz(screenHertz); ok {
		return interval, true
	}
	mustLoadDisplay()
	mode := cgDisplayCopyDisplayMode(d.displayID)
	if mode == 0 {
		return 0, false
	}
	defer cgDisplayModeRelease(mode)
	return gpui.RefreshIntervalFromHz(cgDisplayModeGetRefreshRate(mode))
}

func (d *MacDisplay) getNSScreen() (appkit.NSScreen, bool) {
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("displays require the main thread")
	}
	for _, screen := range appkit.NSScreenScreens() {
		if screenHasNumber(screen, d.displayID) {
			return screen, true
		}
	}
	return 0, false
}
