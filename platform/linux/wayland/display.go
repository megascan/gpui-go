//go:build linux

package wayland

import (
	"crypto/sha1"
	"errors"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

// WaylandDisplay is a wl_output as a gpui.PlatformDisplay.
type WaylandDisplay struct {
	gpui.PlatformDisplayDefaults
	id              *wlproto.WlOutput
	name            gpui.Option[string]
	bounds          gpui.Bounds[gpui.Pixels]
	refreshInterval gpui.Option[time.Duration]
}

func newWaylandDisplay(id *wlproto.WlOutput, name gpui.Option[string], bounds gpui.Bounds[gpui.Pixels], refreshInterval gpui.Option[time.Duration]) *WaylandDisplay {
	d := &WaylandDisplay{id: id, name: name, bounds: bounds, refreshInterval: refreshInterval}
	gpui.BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d
}

// ID returns the wl_output's protocol id.
func (d *WaylandDisplay) ID() gpui.DisplayID {
	return gpui.NewDisplayID(uint64(d.id.ID()))
}

// UUID returns a v5 UUID in the DNS namespace derived from the output's name.
func (d *WaylandDisplay) UUID() (gpui.UUID, error) {
	name, ok := d.name.Get()
	if !ok {
		return gpui.UUID{}, errors.New("Wayland display does not have a name")
	}
	return uuidNewV5(uuidNamespaceDNS, []byte(name)), nil
}

// Bounds returns the output's logical bounds.
func (d *WaylandDisplay) Bounds() gpui.Bounds[gpui.Pixels] { return d.bounds }

// RefreshInterval returns the output's current mode's refresh interval.
func (d *WaylandDisplay) RefreshInterval() (time.Duration, bool) { return d.refreshInterval.Get() }

var uuidNamespaceDNS = gpui.UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}

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

func refreshIntervalFromMillihertz(millihertz int32) gpui.Option[time.Duration] {
	return gpui.OptionFrom(gpui.RefreshIntervalFromHz(float64(millihertz) / 1000.0))
}
