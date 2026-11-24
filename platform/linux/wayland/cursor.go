//go:build linux

package wayland

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
)

// Cursor draws themed cursor images on a surface of its own, for compositors without
// cursor-shape-v1. Destroy is its drop.
type Cursor struct {
	loadedTheme *LoadedTheme
	size        uint32
	scaledSize  uint32
	surface     *wlproto.WlSurface
	shm         *wlproto.WlShm
	connection  *wl.Display
}

// LoadedTheme is a cursor theme loaded at one size.
type LoadedTheme struct {
	theme      *wl.CursorTheme
	name       gpui.Option[string]
	scaledSize uint32
}

// Destroy releases the theme and destroys the cursor surface.
func (c *Cursor) Destroy() {
	if c.loadedTheme != nil {
		c.loadedTheme.theme.Destroy()
		c.loadedTheme = nil
	}
	c.surface.Destroy()
}

// NewCursor creates the cursor surface and loads the default theme at size.
func NewCursor(connection *wl.Display, globals *Globals, size uint32) *Cursor {
	c := &Cursor{
		size:       size,
		scaledSize: size,
		surface:    globals.Compositor.CreateSurface(),
		shm:        globals.Shm,
		connection: connection,
	}
	c.setThemeInternal(gpui.None[string]())
	return c
}

func (c *Cursor) setThemeInternal(themeName gpui.Option[string]) {
	if c.loadedTheme != nil && c.loadedTheme.name == themeName && c.loadedTheme.scaledSize == c.scaledSize {
		return
	}
	var theme *wl.CursorTheme
	var err error
	if name, ok := themeName.Get(); ok {
		theme, err = wl.LoadCursorTheme(name, int(c.scaledSize), c.shm)
	} else {
		name := "default"
		if env, ok := os.LookupEnv("XCURSOR_THEME"); ok {
			name = env
		}
		size := c.scaledSize
		if env, ok := os.LookupEnv("XCURSOR_SIZE"); ok {
			if parsed, perr := strconv.ParseUint(env, 10, 32); perr == nil {
				size = uint32(parsed)
			}
		}
		theme, err = wl.LoadCursorTheme(name, int(size), c.shm)
	}
	if err != nil {
		slog.Error(fmt.Errorf("Wayland: Failed to load cursor theme: %w", err).Error())
		return
	}
	if c.loadedTheme != nil {
		c.loadedTheme.theme.Destroy()
	}
	c.loadedTheme = &LoadedTheme{theme: theme, name: themeName, scaledSize: c.scaledSize}
}

// SetTheme loads the theme named themeName.
func (c *Cursor) SetTheme(themeName string) {
	c.setThemeInternal(gpui.Some(themeName))
}

func (c *Cursor) setScaledSize(scaledSize uint32) {
	c.scaledSize = scaledSize
	themeName := gpui.None[string]()
	if c.loadedTheme != nil {
		themeName = c.loadedTheme.name
	}
	c.setThemeInternal(themeName)
}

// SetSize sets the unscaled cursor size.
func (c *Cursor) SetSize(size uint32) {
	c.size = size
	c.setScaledSize(size)
}

// SetIcon shows the first of cursorIconNames the theme has, or the default icon, on wlPointer.
func (c *Cursor) SetIcon(wlPointer *wlproto.WlPointer, serialID uint32, cursorIconNames []string, scale int32) {
	c.setScaledSize(c.size * uint32(scale))

	loadedTheme := c.loadedTheme
	if loadedTheme == nil {
		slog.Warn("Wayland: Unable to load cursor themes")
		return
	}
	theme := loadedTheme.theme

	var image wl.CursorImage
	found := false
	for _, name := range cursorIconNames {
		if cursor, ok := theme.Cursor(name); ok {
			image = cursor.Images()[0]
			found = true
			break
		}
	}
	if !found {
		if cursor, ok := theme.Cursor(common.DefaultCursorIconName); ok {
			image = cursor.Images()[0]
			common.LogCursorIconWarning(fmt.Sprintf("wayland: Unable to get cursor icon %q. Using default cursor icon: '%s'", cursorIconNames, common.DefaultCursorIconName))
		} else {
			common.LogCursorIconWarning(fmt.Sprintf("wayland: Unable to fallback on default cursor icon '%s' for theme '%s'", common.DefaultCursorIconName, loadedTheme.name.UnwrapOr("default")))
			return
		}
	}

	width, height := image.Width, image.Height
	hotX, hotY := image.HotspotX, image.HotspotY

	c.surface.SetBufferScale(scale)
	wlPointer.SetCursor(serialID, c.surface, int32(hotX)/scale, int32(hotY)/scale)

	buffer := wl.Wrap(c.connection, image.Buffer(), wlproto.WlBufferInterface).(*wlproto.WlBuffer)
	c.surface.Attach(buffer, 0, 0)
	c.surface.Damage(0, 0, int32(width), int32(height))
	c.surface.Commit()
}
