//go:build ios

package ios

import (
	"crypto/sha1"
	"fmt"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
)

var uuidNamespaceOID = gpui.UUID{0x6b, 0xa7, 0xb8, 0x12, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

func uuidNewV5(namespace gpui.UUID, name []byte) gpui.UUID {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write(name)
	var u gpui.UUID
	copy(u[:], h.Sum(nil))
	u[6] = (u[6] & 0x0f) | 0x50
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}

// IosDisplay is a UIScreen as a gpui.PlatformDisplay. Release drops its screen reference.
type IosDisplay struct {
	gpui.PlatformDisplayDefaults
	screen *objc.Retained
}

var _ gpui.PlatformDisplay = (*IosDisplay)(nil)

func mustMainThread() {
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("UIKit requires the main thread")
	}
}

func newIosDisplay(screen *objc.Retained) *IosDisplay {
	d := &IosDisplay{screen: screen}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d
}

// IosDisplayFromScreen wraps screen, retaining it.
func IosDisplayFromScreen(screen uikit.UIScreen) *IosDisplay {
	return newIosDisplay(objc.RetainObject(screen.ID()))
}

// IosDisplayMain returns the main screen.
func IosDisplayMain() *IosDisplay {
	mustMainThread()
	return IosDisplayFromScreen(uikit.UIScreenMainScreen())
}

// IosDisplayAll returns every connected screen.
func IosDisplayAll() []*IosDisplay {
	mustMainThread()
	screens := uikit.UIScreenScreens()
	displays := make([]*IosDisplay, len(screens))
	for i, screen := range screens {
		displays[i] = IosDisplayFromScreen(screen)
	}
	return displays
}

func (d *IosDisplay) uiScreen() uikit.UIScreen { return uikit.UIScreen(d.screen.ID()) }

func (d *IosDisplay) boundsInPoints() objc.CGRect { return d.uiScreen().Bounds() }

// NativeScale returns the native scale factor of this screen.
func (d *IosDisplay) NativeScale() float32 { return float32(d.uiScreen().NativeScale()) }

// Scale returns the current scale factor, which may differ from the native one when zoomed.
func (d *IosDisplay) Scale() float32 { return float32(d.uiScreen().Scale()) }

// RefreshInterval returns the frame interval for the screen's maximum frame rate.
func (d *IosDisplay) RefreshInterval() (time.Duration, bool) {
	return gpui.RefreshIntervalFromHz(float64(d.uiScreen().MaximumFramesPerSecond()))
}

// ID returns the screen's object address, since iOS has no display ids.
func (d *IosDisplay) ID() gpui.DisplayID {
	return gpui.NewDisplayID(uint64(d.screen.ID()))
}

// UUID returns a deterministic UUID derived from the screen's size and scale.
func (d *IosDisplay) UUID() (gpui.UUID, error) {
	bounds := d.boundsInPoints()
	scale := d.NativeScale()
	name := fmt.Sprintf("ios-screen-%d-%d-%d", uint32(bounds.Size.Width), uint32(bounds.Size.Height), uint32(scale*100.0))
	return uuidNewV5(uuidNamespaceOID, []byte(name)), nil
}

// Bounds returns the screen bounds in points, at the origin.
func (d *IosDisplay) Bounds() gpui.Bounds[gpui.Pixels] {
	bounds := d.boundsInPoints()
	return gpui.Bounds[gpui.Pixels]{
		Size: gpui.Size[gpui.Pixels]{Width: gpui.Px(float32(bounds.Size.Width)), Height: gpui.Px(float32(bounds.Size.Height))},
	}
}

// Release drops the screen reference.
func (d *IosDisplay) Release() { d.screen.Release() }
