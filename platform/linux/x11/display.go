//go:build linux

package x11

import (
	"fmt"
	"time"

	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
)

// X11Display is one X screen as a gpui.PlatformDisplay.
type X11Display struct {
	gpui.PlatformDisplayDefaults
	xScreenIndex int
	bounds       gpui.Bounds[gpui.Pixels]
	uuid         gpui.UUID
}

// NewX11Display describes screen xScreenIndex of setup at scaleFactor.
func NewX11Display(setup *xproto.SetupInfo, scaleFactor float32, xScreenIndex int) (*X11Display, error) {
	if xScreenIndex < 0 || xScreenIndex >= len(setup.Roots) {
		return nil, fmt.Errorf("No screen found with index %d", xScreenIndex)
	}
	screen := setup.Roots[xScreenIndex]
	d := &X11Display{
		xScreenIndex: xScreenIndex,
		bounds: gpui.Bounds[gpui.Pixels]{
			Size: gpui.Size[gpui.Pixels]{
				Width:  gpui.Px(float32(screen.WidthInPixels) / scaleFactor),
				Height: gpui.Px(float32(screen.HeightInPixels) / scaleFactor),
			},
		},
	}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d, nil
}

// RefreshInterval reports none: an X screen spans every monitor, so it has no single refresh rate.
func (d *X11Display) RefreshInterval() (time.Duration, bool) { return 0, false }

// ID returns the screen index as the display id.
func (d *X11Display) ID() gpui.DisplayID { return gpui.NewDisplayID(uint64(d.xScreenIndex)) }

// UUID returns the nil UUID.
func (d *X11Display) UUID() (gpui.UUID, error) { return d.uuid, nil }

// Bounds returns the screen bounds in logical pixels.
func (d *X11Display) Bounds() gpui.Bounds[gpui.Pixels] { return d.bounds }
