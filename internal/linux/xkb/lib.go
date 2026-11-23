//go:build linux

package xkb

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error

	x11Once sync.Once
	x11Err  error
)

var (
	xkb_context_new                     func(flags uint32) uintptr
	xkb_context_ref                     func(ctx uintptr) uintptr
	xkb_context_unref                   func(ctx uintptr)
	xkb_keymap_new_from_names           func(ctx uintptr, names *ruleNames, flags uint32) uintptr
	xkb_keymap_new_from_string          func(ctx uintptr, s string, format uint32, flags uint32) uintptr
	xkb_keymap_new_from_buffer          func(ctx uintptr, buf uintptr, length uintptr, format uint32, flags uint32) uintptr
	xkb_keymap_ref                      func(km uintptr) uintptr
	xkb_keymap_unref                    func(km uintptr)
	xkb_keymap_get_as_string            func(km uintptr, format uint32) uintptr
	xkb_keymap_min_keycode              func(km uintptr) uint32
	xkb_keymap_max_keycode              func(km uintptr) uint32
	xkb_keymap_num_mods                 func(km uintptr) uint32
	xkb_keymap_mod_get_name             func(km uintptr, idx uint32) uintptr
	xkb_keymap_mod_get_index            func(km uintptr, name string) uint32
	xkb_keymap_num_layouts              func(km uintptr) uint32
	xkb_keymap_layout_get_name          func(km uintptr, idx uint32) uintptr
	xkb_keymap_layout_get_index         func(km uintptr, name string) uint32
	xkb_keymap_num_leds                 func(km uintptr) uint32
	xkb_keymap_led_get_name             func(km uintptr, idx uint32) uintptr
	xkb_keymap_led_get_index            func(km uintptr, name string) uint32
	xkb_keymap_key_get_name             func(km uintptr, key uint32) uintptr
	xkb_keymap_key_by_name              func(km uintptr, name string) uint32
	xkb_keymap_num_layouts_for_key      func(km uintptr, key uint32) uint32
	xkb_keymap_num_levels_for_key       func(km uintptr, key uint32, layout uint32) uint32
	xkb_keymap_key_get_syms_by_level    func(km uintptr, key uint32, layout uint32, level uint32, out *uintptr) int32
	xkb_keymap_key_repeats              func(km uintptr, key uint32) int32
	xkb_state_new                       func(km uintptr) uintptr
	xkb_state_ref                       func(st uintptr) uintptr
	xkb_state_unref                     func(st uintptr)
	xkb_state_get_keymap                func(st uintptr) uintptr
	xkb_state_update_key                func(st uintptr, key uint32, direction uint32) uint32
	xkb_state_update_mask               func(st uintptr, depressedMods, latchedMods, lockedMods, depressedLayout, latchedLayout, lockedLayout uint32) uint32
	xkb_state_key_get_syms              func(st uintptr, key uint32, out *uintptr) int32
	xkb_state_key_get_utf8              func(st uintptr, key uint32, buf *byte, size uintptr) int32
	xkb_state_key_get_utf32             func(st uintptr, key uint32) uint32
	xkb_state_key_get_one_sym           func(st uintptr, key uint32) uint32
	xkb_state_key_get_layout            func(st uintptr, key uint32) uint32
	xkb_state_key_get_level             func(st uintptr, key uint32, layout uint32) uint32
	xkb_state_serialize_mods            func(st uintptr, components uint32) uint32
	xkb_state_serialize_layout          func(st uintptr, components uint32) uint32
	xkb_state_mod_name_is_active        func(st uintptr, name string, typ uint32) int32
	xkb_state_mod_index_is_active       func(st uintptr, idx uint32, typ uint32) int32
	xkb_state_mod_index_is_consumed     func(st uintptr, key uint32, idx uint32) int32
	xkb_state_mod_mask_remove_consumed  func(st uintptr, key uint32, mask uint32) uint32
	xkb_state_key_get_consumed_mods     func(st uintptr, key uint32) uint32
	xkb_state_layout_name_is_active     func(st uintptr, name string, typ uint32) int32
	xkb_state_layout_index_is_active    func(st uintptr, idx uint32, typ uint32) int32
	xkb_state_led_name_is_active        func(st uintptr, name string) int32
	xkb_state_led_index_is_active       func(st uintptr, idx uint32) int32
	xkb_keysym_get_name                 func(sym uint32, buf *byte, size uintptr) int32
	xkb_keysym_from_name                func(name string, flags uint32) uint32
	xkb_keysym_to_utf8                  func(sym uint32, buf *byte, size uintptr) int32
	xkb_keysym_to_utf32                 func(sym uint32) uint32
	xkb_utf32_to_keysym                 func(ucs uint32) uint32
	xkb_compose_table_new_from_locale   func(ctx uintptr, locale string, flags uint32) uintptr
	xkb_compose_table_new_from_buffer   func(ctx uintptr, buf *byte, length uintptr, locale string, format uint32, flags uint32) uintptr
	xkb_compose_table_ref               func(t uintptr) uintptr
	xkb_compose_table_unref             func(t uintptr)
	xkb_compose_state_new               func(t uintptr, flags uint32) uintptr
	xkb_compose_state_ref               func(s uintptr) uintptr
	xkb_compose_state_unref             func(s uintptr)
	xkb_compose_state_get_compose_table func(s uintptr) uintptr
	xkb_compose_state_feed              func(s uintptr, sym uint32) uint32
	xkb_compose_state_reset             func(s uintptr)
	xkb_compose_state_get_status        func(s uintptr) uint32
	xkb_compose_state_get_utf8          func(s uintptr, buf *byte, size uintptr) int32
	xkb_compose_state_get_one_sym       func(s uintptr) uint32

	libc_free func(p uintptr)

	xkb_x11_setup_xkb_extension         func(conn uintptr, major, minor uint16, flags uint32, outMajor, outMinor *uint16, baseEvent, baseError *uint8) int32
	xkb_x11_get_core_keyboard_device_id func(conn uintptr) int32
	xkb_x11_keymap_new_from_device      func(ctx uintptr, conn uintptr, deviceID int32, flags uint32) uintptr
	xkb_x11_state_new_from_device       func(km uintptr, conn uintptr, deviceID int32) uintptr
)

// Load opens libxkbcommon.so.0 and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen("libxkbcommon.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("xkb: open libxkbcommon.so.0: %w", err)
			return
		}
		libc, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("xkb: open libc.so.6: %w", err)
			return
		}
		if loadErr = register(libc, []binding{{&libc_free, "free"}}); loadErr != nil {
			return
		}
		loadErr = register(lib, []binding{
			{&xkb_context_new, "xkb_context_new"},
			{&xkb_context_ref, "xkb_context_ref"},
			{&xkb_context_unref, "xkb_context_unref"},
			{&xkb_keymap_new_from_names, "xkb_keymap_new_from_names"},
			{&xkb_keymap_new_from_string, "xkb_keymap_new_from_string"},
			{&xkb_keymap_new_from_buffer, "xkb_keymap_new_from_buffer"},
			{&xkb_keymap_ref, "xkb_keymap_ref"},
			{&xkb_keymap_unref, "xkb_keymap_unref"},
			{&xkb_keymap_get_as_string, "xkb_keymap_get_as_string"},
			{&xkb_keymap_min_keycode, "xkb_keymap_min_keycode"},
			{&xkb_keymap_max_keycode, "xkb_keymap_max_keycode"},
			{&xkb_keymap_num_mods, "xkb_keymap_num_mods"},
			{&xkb_keymap_mod_get_name, "xkb_keymap_mod_get_name"},
			{&xkb_keymap_mod_get_index, "xkb_keymap_mod_get_index"},
			{&xkb_keymap_num_layouts, "xkb_keymap_num_layouts"},
			{&xkb_keymap_layout_get_name, "xkb_keymap_layout_get_name"},
			{&xkb_keymap_layout_get_index, "xkb_keymap_layout_get_index"},
			{&xkb_keymap_num_leds, "xkb_keymap_num_leds"},
			{&xkb_keymap_led_get_name, "xkb_keymap_led_get_name"},
			{&xkb_keymap_led_get_index, "xkb_keymap_led_get_index"},
			{&xkb_keymap_key_get_name, "xkb_keymap_key_get_name"},
			{&xkb_keymap_key_by_name, "xkb_keymap_key_by_name"},
			{&xkb_keymap_num_layouts_for_key, "xkb_keymap_num_layouts_for_key"},
			{&xkb_keymap_num_levels_for_key, "xkb_keymap_num_levels_for_key"},
			{&xkb_keymap_key_get_syms_by_level, "xkb_keymap_key_get_syms_by_level"},
			{&xkb_keymap_key_repeats, "xkb_keymap_key_repeats"},
			{&xkb_state_new, "xkb_state_new"},
			{&xkb_state_ref, "xkb_state_ref"},
			{&xkb_state_unref, "xkb_state_unref"},
			{&xkb_state_get_keymap, "xkb_state_get_keymap"},
			{&xkb_state_update_key, "xkb_state_update_key"},
			{&xkb_state_update_mask, "xkb_state_update_mask"},
			{&xkb_state_key_get_syms, "xkb_state_key_get_syms"},
			{&xkb_state_key_get_utf8, "xkb_state_key_get_utf8"},
			{&xkb_state_key_get_utf32, "xkb_state_key_get_utf32"},
			{&xkb_state_key_get_one_sym, "xkb_state_key_get_one_sym"},
			{&xkb_state_key_get_layout, "xkb_state_key_get_layout"},
			{&xkb_state_key_get_level, "xkb_state_key_get_level"},
			{&xkb_state_serialize_mods, "xkb_state_serialize_mods"},
			{&xkb_state_serialize_layout, "xkb_state_serialize_layout"},
			{&xkb_state_mod_name_is_active, "xkb_state_mod_name_is_active"},
			{&xkb_state_mod_index_is_active, "xkb_state_mod_index_is_active"},
			{&xkb_state_mod_index_is_consumed, "xkb_state_mod_index_is_consumed"},
			{&xkb_state_mod_mask_remove_consumed, "xkb_state_mod_mask_remove_consumed"},
			{&xkb_state_key_get_consumed_mods, "xkb_state_key_get_consumed_mods"},
			{&xkb_state_layout_name_is_active, "xkb_state_layout_name_is_active"},
			{&xkb_state_layout_index_is_active, "xkb_state_layout_index_is_active"},
			{&xkb_state_led_name_is_active, "xkb_state_led_name_is_active"},
			{&xkb_state_led_index_is_active, "xkb_state_led_index_is_active"},
			{&xkb_keysym_get_name, "xkb_keysym_get_name"},
			{&xkb_keysym_from_name, "xkb_keysym_from_name"},
			{&xkb_keysym_to_utf8, "xkb_keysym_to_utf8"},
			{&xkb_keysym_to_utf32, "xkb_keysym_to_utf32"},
			{&xkb_utf32_to_keysym, "xkb_utf32_to_keysym"},
			{&xkb_compose_table_new_from_locale, "xkb_compose_table_new_from_locale"},
			{&xkb_compose_table_new_from_buffer, "xkb_compose_table_new_from_buffer"},
			{&xkb_compose_table_ref, "xkb_compose_table_ref"},
			{&xkb_compose_table_unref, "xkb_compose_table_unref"},
			{&xkb_compose_state_new, "xkb_compose_state_new"},
			{&xkb_compose_state_ref, "xkb_compose_state_ref"},
			{&xkb_compose_state_unref, "xkb_compose_state_unref"},
			{&xkb_compose_state_get_compose_table, "xkb_compose_state_get_compose_table"},
			{&xkb_compose_state_feed, "xkb_compose_state_feed"},
			{&xkb_compose_state_reset, "xkb_compose_state_reset"},
			{&xkb_compose_state_get_status, "xkb_compose_state_get_status"},
			{&xkb_compose_state_get_utf8, "xkb_compose_state_get_utf8"},
			{&xkb_compose_state_get_one_sym, "xkb_compose_state_get_one_sym"},
		})
	})
	return loadErr
}

// LoadX11 opens libxkbcommon-x11.so.0 in addition to libxkbcommon.so.0.
func LoadX11() error {
	if err := Load(); err != nil {
		return err
	}
	x11Once.Do(func() {
		lib, err := purego.Dlopen("libxkbcommon-x11.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			x11Err = fmt.Errorf("xkb: open libxkbcommon-x11.so.0: %w", err)
			return
		}
		x11Err = register(lib, []binding{
			{&xkb_x11_setup_xkb_extension, "xkb_x11_setup_xkb_extension"},
			{&xkb_x11_get_core_keyboard_device_id, "xkb_x11_get_core_keyboard_device_id"},
			{&xkb_x11_keymap_new_from_device, "xkb_x11_keymap_new_from_device"},
			{&xkb_x11_state_new_from_device, "xkb_x11_state_new_from_device"},
		})
	})
	return x11Err
}

type binding struct {
	fptr any
	name string
}

func register(lib uintptr, bs []binding) error {
	for _, b := range bs {
		sym, err := purego.Dlsym(lib, b.name)
		if err != nil {
			return fmt.Errorf("xkb: resolve %s: %w", b.name, err)
		}
		purego.RegisterFunc(b.fptr, sym)
	}
	return nil
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(unsafeAdd(p, n)) != 0 {
		n++
	}
	return string(unsafeBytes(p, n))
}

func libcFree(p uintptr) { libc_free(p) }
