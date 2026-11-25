//go:build linux

package wl

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error

	cursorOnce sync.Once
	cursorErr  error
)

var (
	wl_display_connect                func(name *byte) uintptr
	wl_display_connect_to_fd          func(fd int32) uintptr
	wl_display_disconnect             func(d uintptr)
	wl_display_get_fd                 func(d uintptr) int32
	wl_display_dispatch               func(d uintptr) int32
	wl_display_dispatch_pending       func(d uintptr) int32
	wl_display_dispatch_queue         func(d, q uintptr) int32
	wl_display_dispatch_queue_pending func(d, q uintptr) int32
	wl_display_roundtrip              func(d uintptr) int32
	wl_display_roundtrip_queue        func(d, q uintptr) int32
	wl_display_flush                  func(d uintptr) int32
	wl_display_prepare_read           func(d uintptr) int32
	wl_display_prepare_read_queue     func(d, q uintptr) int32
	wl_display_read_events            func(d uintptr) int32
	wl_display_cancel_read            func(d uintptr)
	wl_display_get_error              func(d uintptr) int32
	wl_display_get_protocol_error     func(d uintptr, iface *uintptr, id *uint32) uint32
	wl_display_create_queue           func(d uintptr) uintptr
	wl_event_queue_destroy            func(q uintptr)

	wl_proxy_marshal_array_flags func(p uintptr, opcode uint32, iface uintptr, version uint32, flags uint32, args unsafe.Pointer) uintptr
	wl_proxy_add_dispatcher      func(p, fn, impl, data uintptr) int32
	wl_proxy_destroy             func(p uintptr)
	wl_proxy_get_version         func(p uintptr) uint32
	wl_proxy_get_id              func(p uintptr) uint32
	wl_proxy_get_class           func(p uintptr) uintptr
	wl_proxy_get_listener        func(p uintptr) uintptr
	wl_proxy_set_queue           func(p, q uintptr)

	errno_location func() uintptr

	wl_cursor_theme_load         func(name *byte, size int32, shm uintptr) uintptr
	wl_cursor_theme_destroy      func(t uintptr)
	wl_cursor_theme_get_cursor   func(t uintptr, name *byte) uintptr
	wl_cursor_image_get_buffer   func(img uintptr) uintptr
	wl_cursor_frame_and_duration func(c uintptr, time uint32, duration *uint32) int32
)

var clientLib uintptr

// Load opens libwayland-client.so.0 and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen("libwayland-client.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("wl: open libwayland-client.so.0: %w", err)
			return
		}
		clientLib = lib
		loadErr = register(lib, []binding{
			{&wl_display_connect, "wl_display_connect"},
			{&wl_display_connect_to_fd, "wl_display_connect_to_fd"},
			{&wl_display_disconnect, "wl_display_disconnect"},
			{&wl_display_get_fd, "wl_display_get_fd"},
			{&wl_display_dispatch, "wl_display_dispatch"},
			{&wl_display_dispatch_pending, "wl_display_dispatch_pending"},
			{&wl_display_dispatch_queue, "wl_display_dispatch_queue"},
			{&wl_display_dispatch_queue_pending, "wl_display_dispatch_queue_pending"},
			{&wl_display_roundtrip, "wl_display_roundtrip"},
			{&wl_display_roundtrip_queue, "wl_display_roundtrip_queue"},
			{&wl_display_flush, "wl_display_flush"},
			{&wl_display_prepare_read, "wl_display_prepare_read"},
			{&wl_display_prepare_read_queue, "wl_display_prepare_read_queue"},
			{&wl_display_read_events, "wl_display_read_events"},
			{&wl_display_cancel_read, "wl_display_cancel_read"},
			{&wl_display_get_error, "wl_display_get_error"},
			{&wl_display_get_protocol_error, "wl_display_get_protocol_error"},
			{&wl_display_create_queue, "wl_display_create_queue"},
			{&wl_event_queue_destroy, "wl_event_queue_destroy"},
			{&wl_proxy_marshal_array_flags, "wl_proxy_marshal_array_flags"},
			{&wl_proxy_add_dispatcher, "wl_proxy_add_dispatcher"},
			{&wl_proxy_destroy, "wl_proxy_destroy"},
			{&wl_proxy_get_version, "wl_proxy_get_version"},
			{&wl_proxy_get_id, "wl_proxy_get_id"},
			{&wl_proxy_get_class, "wl_proxy_get_class"},
			{&wl_proxy_get_listener, "wl_proxy_get_listener"},
			{&wl_proxy_set_queue, "wl_proxy_set_queue"},
		})
		if loadErr == nil {
			if libc, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				if sym, err := purego.Dlsym(libc, "__errno_location"); err == nil {
					purego.RegisterFunc(&errno_location, sym)
				}
			}
			dispatcherFn = purego.NewCallback(dispatcher)
		}
	})
	return loadErr
}

// LoadCursor opens libwayland-cursor.so.0 and resolves its symbols. It is safe to call repeatedly.
func LoadCursor() error {
	if err := Load(); err != nil {
		return err
	}
	cursorOnce.Do(func() {
		lib, err := purego.Dlopen("libwayland-cursor.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			cursorErr = fmt.Errorf("wl: open libwayland-cursor.so.0: %w", err)
			return
		}
		cursorErr = register(lib, []binding{
			{&wl_cursor_theme_load, "wl_cursor_theme_load"},
			{&wl_cursor_theme_destroy, "wl_cursor_theme_destroy"},
			{&wl_cursor_theme_get_cursor, "wl_cursor_theme_get_cursor"},
			{&wl_cursor_image_get_buffer, "wl_cursor_image_get_buffer"},
			{&wl_cursor_frame_and_duration, "wl_cursor_frame_and_duration"},
		})
	})
	return cursorErr
}

// LibrarySymbol returns the address of an exported libwayland-client symbol, such as
// "wl_surface_interface".
func LibrarySymbol(name string) (uintptr, error) {
	if err := Load(); err != nil {
		return 0, err
	}
	sym, err := purego.Dlsym(clientLib, name)
	if err != nil {
		return 0, fmt.Errorf("wl: resolve %s: %w", name, err)
	}
	return sym, nil
}

type binding struct {
	fptr any
	name string
}

func register(lib uintptr, bs []binding) error {
	for _, b := range bs {
		sym, err := purego.Dlsym(lib, b.name)
		if err != nil {
			return fmt.Errorf("wl: resolve %s: %w", b.name, err)
		}
		purego.RegisterFunc(b.fptr, sym)
	}
	return nil
}

func errno() syscall.Errno {
	if errno_location == nil {
		return 0
	}
	return syscall.Errno(int32(load32(errno_location())))
}
