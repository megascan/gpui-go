package gpui

import "time"

// TestDisplay is the display of the test platform.
type TestDisplay struct {
	PlatformDisplayDefaults
	id              DisplayID
	uuid            UUID
	bounds          Bounds[Pixels]
	refreshInterval Option[time.Duration]
}

// NewTestDisplay returns a 1920x1080 test display with id 1.
func NewTestDisplay() *TestDisplay {
	return TestDisplayWithID(DisplayID(1))
}

// TestDisplayWithID returns a 1920x1080 test display with the given id, refreshing at 60 Hz.
func TestDisplayWithID(id DisplayID) *TestDisplay {
	d := &TestDisplay{
		id:              id,
		uuid:            NewUUIDV4(),
		bounds:          BoundsFromCorners(Point[Pixels]{}, NewPoint(Px(1920.), Px(1080.))),
		refreshInterval: Some(time.Second / 60),
	}
	BindPlatformDisplayDefaults(&d.PlatformDisplayDefaults, d)
	return d
}

// Clone returns a copy of the display.
func (d *TestDisplay) Clone() *TestDisplay {
	c := &TestDisplay{id: d.id, uuid: d.uuid, bounds: d.bounds, refreshInterval: d.refreshInterval}
	BindPlatformDisplayDefaults(&c.PlatformDisplayDefaults, c)
	return c
}

// ID returns the display's id.
func (d *TestDisplay) ID() DisplayID {
	return d.id
}

// UUID returns the display's random UUID.
func (d *TestDisplay) UUID() (UUID, error) {
	return d.uuid, nil
}

// Bounds returns the display's bounds.
func (d *TestDisplay) Bounds() Bounds[Pixels] {
	return d.bounds
}

// RefreshInterval returns the display's refresh interval, if it has one.
func (d *TestDisplay) RefreshInterval() (time.Duration, bool) {
	return d.refreshInterval.Get()
}
