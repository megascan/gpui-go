//go:build linux

package xcbshim

import (
	"errors"
	"fmt"
	"runtime"
)

// Connection is an xcb_connection_t created through libxcb.so.1. Close must run after
// every wgpu surface made from the connection has been released.
type Connection struct {
	ptr           uintptr
	defaultScreen int
}

// Connect opens an xcb connection to display, or to $DISPLAY when display is empty.
func Connect(display string) (*Connection, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	var name *byte
	var buf []byte
	if display != "" {
		buf = append([]byte(display), 0)
		name = &buf[0]
	}
	var screen int32
	ptr := xcb_connect(name, &screen)
	runtime.KeepAlive(buf)
	if ptr == 0 {
		return nil, errors.New("xcbshim: xcb_connect returned null")
	}
	if code := xcb_connection_has_error(ptr); code != 0 {
		xcb_disconnect(ptr)
		return nil, fmt.Errorf("xcbshim: connect to %q failed with xcb error %d", display, code)
	}
	return &Connection{ptr: ptr, defaultScreen: int(screen)}, nil
}

// Raw returns the xcb_connection_t pointer for wgpu surface creation.
func (c *Connection) Raw() uintptr {
	c.mustOpen()
	return c.ptr
}

// DefaultScreen returns the screen index that xcb_connect selected.
func (c *Connection) DefaultScreen() int {
	c.mustOpen()
	return c.defaultScreen
}

// ScreenCount returns the number of screens in the connection setup.
func (c *Connection) ScreenCount() int {
	c.mustOpen()
	return int(load8(xcb_get_setup(c.ptr) + setupRootsLen))
}

// RootVisual returns the root visual id of screen, and false when the index is out of range.
func (c *Connection) RootVisual(screen int) (uint32, bool) {
	c.mustOpen()
	return rootVisualOf(xcb_get_setup(c.ptr), screen)
}

// VisualForDepth returns the first visual id of the given depth on screen, and false when none exists.
func (c *Connection) VisualForDepth(screen int, depth uint8) (uint32, bool) {
	c.mustOpen()
	return visualForDepthOf(xcb_get_setup(c.ptr), screen, depth)
}

// Flush writes queued requests to the server.
func (c *Connection) Flush() error {
	c.mustOpen()
	if rc := xcb_flush(c.ptr); rc <= 0 {
		return fmt.Errorf("xcbshim: xcb_flush failed (error %d)", xcb_connection_has_error(c.ptr))
	}
	return nil
}

// Close disconnects the connection. It is idempotent.
func (c *Connection) Close() {
	if c.ptr == 0 {
		return
	}
	xcb_disconnect(c.ptr)
	c.ptr = 0
}

func (c *Connection) mustOpen() {
	if c.ptr == 0 {
		panic("xcbshim: connection used after Close")
	}
}
