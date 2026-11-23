package gpui

// App is the GPUI application context (M0-app-core ports it).
type App struct{}

// FindDisplay returns the display with the given id, or nil (M0-app-core ports it).
func (cx *App) FindDisplay(id DisplayID) PlatformDisplay {
	panic("gpui: unimplemented App.FindDisplay")
}

// PrimaryDisplay returns the primary display, or nil (M0-app-core ports it).
func (cx *App) PrimaryDisplay() PlatformDisplay {
	panic("gpui: unimplemented App.PrimaryDisplay")
}

// DisplayID is an opaque identifier for a hardware display (M0-platform-types ports it).
type DisplayID uint64

// PlatformDisplay is a display attached to the platform (M0-platform-types ports the full method set).
type PlatformDisplay interface {
	ID() DisplayID
	Bounds() Bounds[Pixels]
	VisibleBounds() Bounds[Pixels]
}

// Window is a GPUI window (M0-window-core ports it).
type Window struct{}

// RemSize returns the window's rem size (M0-window-core ports it).
func (w *Window) RemSize() Pixels {
	panic("gpui: unimplemented Window.RemSize")
}
