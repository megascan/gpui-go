//go:build js && wasm

package web

import (
	"syscall/js"
	"time"

	"github.com/megascan/gpui-go"
)

// WebDisplay is the browser's only display.
type WebDisplay struct {
	gpui.PlatformDisplayDefaults
	id            gpui.DisplayID
	uuid          gpui.UUID
	browserWindow js.Value
}

var _ gpui.PlatformDisplay = (*WebDisplay)(nil)

// NewWebDisplay returns the display of browserWindow.
func NewWebDisplay(browserWindow js.Value) *WebDisplay {
	d := &WebDisplay{
		id:            gpui.NewDisplayID(1),
		uuid:          gpui.NewUUIDV4(),
		browserWindow: browserWindow,
	}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d
}

func (d *WebDisplay) screenSize() gpui.Size[gpui.Pixels] {
	screen := jsGet(d.browserWindow, "screen")
	if jsIsNullish(screen) {
		return gpui.Size[gpui.Pixels]{Width: gpui.Px(1920.), Height: gpui.Px(1080.)}
	}
	width, ok := jsNumber(jsGet(screen, "width"))
	if !ok {
		width = 1920
	}
	height, ok := jsNumber(jsGet(screen, "height"))
	if !ok {
		height = 1080
	}
	return gpui.Size[gpui.Pixels]{Width: gpui.Px(float32(int32(width))), Height: gpui.Px(float32(int32(height)))}
}

func (d *WebDisplay) viewportSize() gpui.Size[gpui.Pixels] {
	width, ok := jsNumber(jsGet(d.browserWindow, "innerWidth"))
	if !ok {
		width = 1920.0
	}
	height, ok := jsNumber(jsGet(d.browserWindow, "innerHeight"))
	if !ok {
		height = 1080.0
	}
	return gpui.Size[gpui.Pixels]{Width: gpui.Px(float32(width)), Height: gpui.Px(float32(height))}
}

// RefreshInterval reports no interval: browsers don't expose the refresh rate.
func (d *WebDisplay) RefreshInterval() (time.Duration, bool) {
	return 0, false
}

// ID returns the display id, always 1.
func (d *WebDisplay) ID() gpui.DisplayID {
	return d.id
}

// UUID returns the display's random UUID.
func (d *WebDisplay) UUID() (gpui.UUID, error) {
	return d.uuid, nil
}

// Bounds returns the screen size at the origin.
func (d *WebDisplay) Bounds() gpui.Bounds[gpui.Pixels] {
	return gpui.Bounds[gpui.Pixels]{Size: d.screenSize()}
}

// VisibleBounds returns the browser viewport size at the origin.
func (d *WebDisplay) VisibleBounds() gpui.Bounds[gpui.Pixels] {
	return gpui.Bounds[gpui.Pixels]{Size: d.viewportSize()}
}

// DefaultBounds returns three quarters of the viewport, centered.
func (d *WebDisplay) DefaultBounds() gpui.Bounds[gpui.Pixels] {
	visible := d.VisibleBounds()
	width := visible.Size.Width * 0.75
	height := visible.Size.Height * 0.75
	originX := (visible.Size.Width - width) / 2.0
	originY := (visible.Size.Height - height) / 2.0
	return gpui.Bounds[gpui.Pixels]{
		Origin: gpui.NewPoint(originX, originY),
		Size:   gpui.Size[gpui.Pixels]{Width: width, Height: height},
	}
}
