//go:build linux

package wl

import (
	"errors"
	"fmt"
	"runtime"
)

// CursorTheme is a wl_cursor_theme from libwayland-cursor.
type CursorTheme struct {
	ptr     uintptr
	display *Display
}

// Cursor is a wl_cursor owned by its theme.
type Cursor struct {
	ptr   uintptr
	theme *CursorTheme
}

// CursorImage is one frame of a Cursor.
type CursorImage struct {
	ptr      uintptr
	theme    *CursorTheme
	Width    uint32
	Height   uint32
	HotspotX uint32
	HotspotY uint32
	Delay    uint32
}

// LoadCursorTheme loads the cursor theme name (the default theme when empty) at size
// pixels, with buffers allocated from shm, a wl_shm proxy.
func LoadCursorTheme(name string, size int, shm Object) (*CursorTheme, error) {
	if err := LoadCursor(); err != nil {
		return nil, err
	}
	p := shm.base()
	if !p.alive {
		return nil, errors.New("wl: cursor theme needs a live wl_shm")
	}
	var arg *byte
	var buf []byte
	if name != "" {
		var err error
		if buf, err = cStringChecked(name); err != nil {
			return nil, fmt.Errorf("wl: cursor theme name: %w", err)
		}
		arg = &buf[0]
	}
	ptr := wl_cursor_theme_load(arg, int32(size), p.ptr)
	runtime.KeepAlive(buf)
	if ptr == 0 {
		return nil, errors.New("wl: wl_cursor_theme_load failed")
	}
	return &CursorTheme{ptr: ptr, display: p.display}, nil
}

// Cursor returns the cursor called name, and false when the theme lacks it.
func (t *CursorTheme) Cursor(name string) (*Cursor, bool) {
	t.mustOpen()
	buf, err := cStringChecked(name)
	if err != nil {
		return nil, false
	}
	ptr := wl_cursor_theme_get_cursor(t.ptr, &buf[0])
	runtime.KeepAlive(buf)
	if ptr == 0 {
		return nil, false
	}
	return &Cursor{ptr: ptr, theme: t}, true
}

// Destroy frees the theme with its cursors and buffers. It is idempotent.
func (t *CursorTheme) Destroy() {
	if t.ptr == 0 {
		return
	}
	wl_cursor_theme_destroy(t.ptr)
	t.ptr = 0
}

func (t *CursorTheme) mustOpen() {
	if t.ptr == 0 {
		panic("wl: use of destroyed cursor theme")
	}
}

// Name returns the cursor's name.
func (c *Cursor) Name() string {
	c.theme.mustOpen()
	return goString(loadPtr(c.ptr + 16))
}

// Images returns the cursor's frames.
func (c *Cursor) Images() []CursorImage {
	c.theme.mustOpen()
	n := int(load32(c.ptr))
	images := loadPtr(c.ptr + 8)
	out := make([]CursorImage, n)
	for k := range out {
		img := loadPtr(images + uintptr(k)*8)
		out[k] = CursorImage{
			ptr:      img,
			theme:    c.theme,
			Width:    load32(img),
			Height:   load32(img + 4),
			HotspotX: load32(img + 8),
			HotspotY: load32(img + 12),
			Delay:    load32(img + 16),
		}
	}
	return out
}

// FrameAndDuration returns the frame to show time milliseconds into the animation and
// how long it stays.
func (c *Cursor) FrameAndDuration(time uint32) (int, uint32) {
	c.theme.mustOpen()
	var d uint32
	frame := wl_cursor_frame_and_duration(c.ptr, time, &d)
	return int(frame), d
}

// Buffer returns the image's wl_buffer pointer, owned by the theme. Wrap it with Wrap
// and the wl_buffer interface to attach it.
func (i CursorImage) Buffer() uintptr {
	i.theme.mustOpen()
	return wl_cursor_image_get_buffer(i.ptr)
}

// Display returns the connection the theme's buffers belong to.
func (t *CursorTheme) Display() *Display { return t.display }
