//go:build linux

package xcbshim

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error
)

var (
	xcb_connect              func(display *byte, screen *int32) uintptr
	xcb_connection_has_error func(c uintptr) int32
	xcb_get_setup            func(c uintptr) uintptr
	xcb_flush                func(c uintptr) int32
	xcb_disconnect           func(c uintptr)
)

// Load opens libxcb.so.1 and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen("libxcb.so.1", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("xcbshim: open libxcb.so.1: %w", err)
			return
		}
		loadErr = register(lib, []binding{
			{&xcb_connect, "xcb_connect"},
			{&xcb_connection_has_error, "xcb_connection_has_error"},
			{&xcb_get_setup, "xcb_get_setup"},
			{&xcb_flush, "xcb_flush"},
			{&xcb_disconnect, "xcb_disconnect"},
		})
	})
	return loadErr
}

type binding struct {
	fptr any
	name string
}

func register(lib uintptr, bs []binding) error {
	for _, b := range bs {
		sym, err := purego.Dlsym(lib, b.name)
		if err != nil {
			return fmt.Errorf("xcbshim: resolve %s: %w", b.name, err)
		}
		purego.RegisterFunc(b.fptr, sym)
	}
	return nil
}
