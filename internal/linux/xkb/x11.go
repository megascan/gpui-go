//go:build linux

package xkb

// SetupXkbExtensionFlags are xkb_x11_setup_xkb_extension_flags.
type SetupXkbExtensionFlags = uint32

// Values from xkbcommon-x11.h.
const (
	X11MinMajorXkbVersion       uint16                 = 1
	X11MinMinorXkbVersion       uint16                 = 0
	X11SetupXkbExtensionNoFlags SetupXkbExtensionFlags = 0
)

// X11SetupXkbExtension calls xkb_x11_setup_xkb_extension on an xcb_connection_t pointer.
func X11SetupXkbExtension(conn uintptr, majorXkbVersion, minorXkbVersion uint16, flags SetupXkbExtensionFlags, majorXkbVersionOut, minorXkbVersionOut *uint16, baseEventOut, baseErrorOut *uint8) bool {
	if err := LoadX11(); err != nil {
		panic(err)
	}
	return xkb_x11_setup_xkb_extension(conn, majorXkbVersion, minorXkbVersion, flags, majorXkbVersionOut, minorXkbVersionOut, baseEventOut, baseErrorOut) != 0
}

// X11GetCoreKeyboardDeviceID returns the core keyboard device id, or -1.
func X11GetCoreKeyboardDeviceID(conn uintptr) int32 {
	if err := LoadX11(); err != nil {
		panic(err)
	}
	return xkb_x11_get_core_keyboard_device_id(conn)
}

// X11KeymapNewFromDevice builds a keymap from an X11 keyboard device. It returns nil on failure.
func X11KeymapNewFromDevice(ctx *Context, conn uintptr, deviceID int32, flags KeymapCompileFlags) *Keymap {
	if err := LoadX11(); err != nil {
		panic(err)
	}
	p := xkb_x11_keymap_new_from_device(ctx.ptr, conn, deviceID, flags)
	if p == 0 {
		return nil
	}
	return &Keymap{ptr: p}
}

// X11StateNewFromDevice builds a state from an X11 keyboard device.
func X11StateNewFromDevice(keymap *Keymap, conn uintptr, deviceID int32) *State {
	if err := LoadX11(); err != nil {
		panic(err)
	}
	return &State{ptr: xkb_x11_state_new_from_device(keymap.ptr, conn, deviceID)}
}
