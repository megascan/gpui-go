//go:build linux

// Package xkb binds libxkbcommon and libxkbcommon-x11 through purego, mirroring the xkbcommon-rs API that gpui_linux uses.
package xkb

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

type (
	// Keycode is an xkb_keycode_t.
	Keycode uint32
	// LayoutIndex is an xkb_layout_index_t.
	LayoutIndex = uint32
	// LevelIndex is an xkb_level_index_t.
	LevelIndex = uint32
	// ModIndex is an xkb_mod_index_t.
	ModIndex = uint32
	// ModMask is an xkb_mod_mask_t.
	ModMask = uint32
	// LedIndex is an xkb_led_index_t.
	LedIndex = uint32
	// ContextFlags are xkb_context_flags.
	ContextFlags = uint32
	// KeymapFormat is an xkb_keymap_format.
	KeymapFormat = uint32
	// KeymapCompileFlags are xkb_keymap_compile_flags.
	KeymapCompileFlags = uint32
	// StateComponent is an xkb_state_component mask.
	StateComponent = uint32
	// KeyDirection is an xkb_key_direction.
	KeyDirection = uint32
	// KeysymFlags are xkb_keysym_flags.
	KeysymFlags = uint32
)

// Flag and enum values from xkbcommon.h.
const (
	ContextNoFlags            ContextFlags       = 0
	ContextNoDefaultIncludes  ContextFlags       = 1 << 0
	ContextNoEnvironmentNames ContextFlags       = 1 << 1
	ContextNoSecureGetenv     ContextFlags       = 1 << 2
	KeymapFormatTextV1        KeymapFormat       = 1
	KeymapFormatUseOriginal   KeymapFormat       = 0xffffffff
	KeymapCompileNoFlags      KeymapCompileFlags = 0
	KeyUp                     KeyDirection       = 0
	KeyDown                   KeyDirection       = 1
	StateModsDepressed        StateComponent     = 1 << 0
	StateModsLatched          StateComponent     = 1 << 1
	StateModsLocked           StateComponent     = 1 << 2
	StateModsEffective        StateComponent     = 1 << 3
	StateLayoutDepressed      StateComponent     = 1 << 4
	StateLayoutLatched        StateComponent     = 1 << 5
	StateLayoutLocked         StateComponent     = 1 << 6
	StateLayoutEffective      StateComponent     = 1 << 7
	StateLeds                 StateComponent     = 1 << 8
	KeysymNoFlags             KeysymFlags        = 0
	KeysymCaseInsensitive     KeysymFlags        = 1 << 0
	KeycodeInvalid            Keycode            = 0xffffffff
	LayoutInvalid             LayoutIndex        = 0xffffffff
	LevelInvalid              LevelIndex         = 0xffffffff
	ModInvalid                ModIndex           = 0xffffffff
	LedInvalid                LedIndex           = 0xffffffff
	KeycodeMax                Keycode            = 0xffffffff - 1
)

// Modifier and LED names from xkbcommon-names.h.
const (
	ModNameShift          = "Shift"
	ModNameCaps           = "Lock"
	ModNameCtrl           = "Control"
	ModNameAlt            = "Mod1"
	ModNameNum            = "Mod2"
	ModNameMod3           = "Mod3"
	ModNameLogo           = "Mod4"
	ModNameIsoLevel3Shift = "Mod5"
	LedNameCaps           = "Caps Lock"
	LedNameNum            = "Num Lock"
	LedNameScroll         = "Scroll Lock"
)

// KeycodeIsLegalExt reports whether key is a legal extended keycode.
func KeycodeIsLegalExt(key uint32) bool { return key <= uint32(KeycodeMax) }

// KeycodeIsLegalX11 reports whether key is a legal X11 keycode.
func KeycodeIsLegalX11(key uint32) bool { return key >= 8 && key <= 255 }

// Context is an xkb_context.
type Context struct{ ptr uintptr }

// NewContext calls xkb_context_new. The returned context wraps a null pointer when libxkbcommon fails; check GetRawPtr.
func NewContext(flags ContextFlags) *Context {
	mustLoad()
	return &Context{ptr: xkb_context_new(flags)}
}

// ContextFromRawPtr takes ownership of an existing xkb_context reference, which may be null.
func ContextFromRawPtr(ptr uintptr) *Context { return &Context{ptr: ptr} }

// GetRawPtr returns the underlying xkb_context pointer.
func (c *Context) GetRawPtr() uintptr { return c.ptr }

// Clone takes a new reference to the context.
func (c *Context) Clone() *Context {
	if c.ptr == 0 {
		return &Context{}
	}
	return &Context{ptr: xkb_context_ref(c.ptr)}
}

// Close drops this reference. It is idempotent.
func (c *Context) Close() {
	if c.ptr != 0 {
		xkb_context_unref(c.ptr)
		c.ptr = 0
	}
}

type ruleNames struct {
	rules, model, layout, variant, options *byte
}

func cstr(s string) *byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return &b[0]
}

// Keymap is an xkb_keymap.
type Keymap struct{ ptr uintptr }

// KeymapFromRawPtr takes ownership of an existing xkb_keymap reference.
func KeymapFromRawPtr(ptr uintptr) *Keymap { return &Keymap{ptr: ptr} }

// NewKeymapFromNames calls xkb_keymap_new_from_names. options may be nil for the default. It returns nil on failure.
func NewKeymapFromNames(ctx *Context, rules, model, layout, variant string, options *string, flags KeymapCompileFlags) *Keymap {
	mustLoad()
	names := &ruleNames{rules: cstr(rules), model: cstr(model), layout: cstr(layout), variant: cstr(variant)}
	if options != nil {
		names.options = cstr(*options)
	}
	p := xkb_keymap_new_from_names(ctx.ptr, names, flags)
	runtime.KeepAlive(names)
	if p == 0 {
		return nil
	}
	return &Keymap{ptr: p}
}

// NewKeymapFromString calls xkb_keymap_new_from_string. It returns nil on failure.
func NewKeymapFromString(ctx *Context, s string, format KeymapFormat, flags KeymapCompileFlags) *Keymap {
	mustLoad()
	p := xkb_keymap_new_from_string(ctx.ptr, s, format, flags)
	if p == 0 {
		return nil
	}
	return &Keymap{ptr: p}
}

// NewKeymapFromFile reads the file and compiles it like NewKeymapFromString.
func NewKeymapFromFile(ctx *Context, file *os.File, format KeymapFormat, flags KeymapCompileFlags) *Keymap {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil
	}
	return NewKeymapFromString(ctx, string(data), format, flags)
}

// NewKeymapFromFd maps fd privately and read-only for size bytes, compiles the buffer without its trailing NUL and closes fd. A nil keymap with a nil error means libxkbcommon rejected the keymap.
func NewKeymapFromFd(ctx *Context, fd int, size int, format KeymapFormat, flags KeymapCompileFlags) (*Keymap, error) {
	mustLoad()
	defer unix.Close(fd)
	if size <= 0 {
		return nil, fmt.Errorf("xkb: map keymap fd: %w", unix.EINVAL)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return nil, fmt.Errorf("xkb: stat keymap fd: %w", err)
	}
	if st.Size < int64(size) {
		return nil, fmt.Errorf("xkb: keymap fd holds %d bytes, declared %d: %w", st.Size, size, unix.EINVAL)
	}
	data, err := unix.Mmap(fd, 0, size, unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("xkb: map keymap fd: %w", err)
	}
	defer unix.Munmap(data)
	p := xkb_keymap_new_from_buffer(ctx.ptr, uintptrOf(&data[0]), uintptr(size-1), format, flags)
	runtime.KeepAlive(data)
	if p == 0 {
		return nil, nil
	}
	return &Keymap{ptr: p}, nil
}

// GetRawPtr returns the underlying xkb_keymap pointer.
func (k *Keymap) GetRawPtr() uintptr { return k.ptr }

// Clone takes a new reference to the keymap.
func (k *Keymap) Clone() *Keymap { return &Keymap{ptr: xkb_keymap_ref(k.ptr)} }

// Close drops this reference. It is idempotent.
func (k *Keymap) Close() {
	if k.ptr != 0 {
		xkb_keymap_unref(k.ptr)
		k.ptr = 0
	}
}

// GetAsString serializes the keymap.
func (k *Keymap) GetAsString(format KeymapFormat) string {
	p := xkb_keymap_get_as_string(k.ptr, format)
	if p == 0 {
		return ""
	}
	s := goString(p)
	libcFree(p)
	return s
}

// MinKeycode returns the minimum keycode.
func (k *Keymap) MinKeycode() Keycode { return Keycode(xkb_keymap_min_keycode(k.ptr)) }

// MaxKeycode returns the maximum keycode.
func (k *Keymap) MaxKeycode() Keycode { return Keycode(xkb_keymap_max_keycode(k.ptr)) }

// NumMods returns the number of modifiers.
func (k *Keymap) NumMods() ModIndex { return xkb_keymap_num_mods(k.ptr) }

// ModGetName returns the name of modifier idx.
func (k *Keymap) ModGetName(idx ModIndex) string {
	return goString(xkb_keymap_mod_get_name(k.ptr, idx))
}

// ModGetIndex returns the index of the named modifier or ModInvalid.
func (k *Keymap) ModGetIndex(name string) ModIndex { return xkb_keymap_mod_get_index(k.ptr, name) }

// NumLayouts returns the number of layouts.
func (k *Keymap) NumLayouts() LayoutIndex { return xkb_keymap_num_layouts(k.ptr) }

// LayoutGetName returns the name of layout idx.
func (k *Keymap) LayoutGetName(idx LayoutIndex) string {
	return goString(xkb_keymap_layout_get_name(k.ptr, idx))
}

// LayoutGetIndex returns the index of the named layout or LayoutInvalid.
func (k *Keymap) LayoutGetIndex(name string) LayoutIndex {
	return xkb_keymap_layout_get_index(k.ptr, name)
}

// NumLeds returns the number of LEDs.
func (k *Keymap) NumLeds() LedIndex { return xkb_keymap_num_leds(k.ptr) }

// LedGetName returns the name of LED idx.
func (k *Keymap) LedGetName(idx LedIndex) string {
	return goString(xkb_keymap_led_get_name(k.ptr, idx))
}

// LedGetIndex returns the index of the named LED or LedInvalid.
func (k *Keymap) LedGetIndex(name string) LedIndex { return xkb_keymap_led_get_index(k.ptr, name) }

// KeyGetName returns the name of key.
func (k *Keymap) KeyGetName(key Keycode) (string, bool) {
	p := xkb_keymap_key_get_name(k.ptr, uint32(key))
	if p == 0 {
		return "", false
	}
	return goString(p), true
}

// KeyByName returns the keycode of the named key.
func (k *Keymap) KeyByName(name string) (Keycode, bool) {
	code := Keycode(xkb_keymap_key_by_name(k.ptr, name))
	if code == KeycodeInvalid {
		return 0, false
	}
	return code, true
}

// NumLayoutsForKey returns the number of layouts for key.
func (k *Keymap) NumLayoutsForKey(key Keycode) LayoutIndex {
	return xkb_keymap_num_layouts_for_key(k.ptr, uint32(key))
}

// NumLevelsForKey returns the number of levels for key in layout.
func (k *Keymap) NumLevelsForKey(key Keycode, layout LayoutIndex) LevelIndex {
	return xkb_keymap_num_levels_for_key(k.ptr, uint32(key), layout)
}

// KeyGetSymsByLevel returns the keysyms bound to key at layout and level. The slice aliases keymap memory.
func (k *Keymap) KeyGetSymsByLevel(key Keycode, layout LayoutIndex, level LevelIndex) []Keysym {
	var out uintptr
	n := xkb_keymap_key_get_syms_by_level(k.ptr, uint32(key), layout, level, &out)
	return unsafeKeysyms(out, int(n))
}

// KeyRepeats reports whether key repeats.
func (k *Keymap) KeyRepeats(key Keycode) bool { return xkb_keymap_key_repeats(k.ptr, uint32(key)) != 0 }

// State is an xkb_state.
type State struct{ ptr uintptr }

// StateFromRawPtr takes ownership of an existing xkb_state reference.
func StateFromRawPtr(ptr uintptr) *State { return &State{ptr: ptr} }

// NewState calls xkb_state_new.
func NewState(keymap *Keymap) *State {
	mustLoad()
	return &State{ptr: xkb_state_new(keymap.ptr)}
}

// GetRawPtr returns the underlying xkb_state pointer.
func (s *State) GetRawPtr() uintptr { return s.ptr }

// Clone takes a new reference to the same state.
func (s *State) Clone() *State { return &State{ptr: xkb_state_ref(s.ptr)} }

// Close drops this reference. It is idempotent.
func (s *State) Close() {
	if s.ptr != 0 {
		xkb_state_unref(s.ptr)
		s.ptr = 0
	}
}

// GetKeymap returns a new reference to the state's keymap; the caller closes it.
func (s *State) GetKeymap() *Keymap {
	return &Keymap{ptr: xkb_keymap_ref(xkb_state_get_keymap(s.ptr))}
}

// UpdateKey updates the state for a key press or release.
func (s *State) UpdateKey(key Keycode, direction KeyDirection) StateComponent {
	return xkb_state_update_key(s.ptr, uint32(key), direction)
}

// UpdateMask sets the serialized modifier and layout state.
func (s *State) UpdateMask(depressedMods, latchedMods, lockedMods ModMask, depressedLayout, latchedLayout, lockedLayout LayoutIndex) StateComponent {
	return xkb_state_update_mask(s.ptr, depressedMods, latchedMods, lockedMods, depressedLayout, latchedLayout, lockedLayout)
}

// KeyGetSyms returns the keysyms for key. The slice aliases state memory.
func (s *State) KeyGetSyms(key Keycode) []Keysym {
	var out uintptr
	n := xkb_state_key_get_syms(s.ptr, uint32(key), &out)
	return unsafeKeysyms(out, int(n))
}

// KeyGetUtf8 returns the UTF-8 text for key, truncated to 64 bytes as in xkbcommon-rs.
func (s *State) KeyGetUtf8(key Keycode) string {
	var buf [64]byte
	ret := xkb_state_key_get_utf8(s.ptr, uint32(key), &buf[0], uintptr(len(buf)))
	return string(buf[:clampLen(ret, len(buf))])
}

// KeyGetUtf32 returns the single codepoint for key, or 0.
func (s *State) KeyGetUtf32(key Keycode) uint32 { return xkb_state_key_get_utf32(s.ptr, uint32(key)) }

// KeyGetOneSym returns the single keysym for key, or KEY_NoSymbol.
func (s *State) KeyGetOneSym(key Keycode) Keysym {
	return Keysym(xkb_state_key_get_one_sym(s.ptr, uint32(key)))
}

// KeyGetLayout returns the effective layout for key.
func (s *State) KeyGetLayout(key Keycode) LayoutIndex {
	return xkb_state_key_get_layout(s.ptr, uint32(key))
}

// KeyGetLevel returns the effective level for key in layout.
func (s *State) KeyGetLevel(key Keycode, layout LayoutIndex) LevelIndex {
	return xkb_state_key_get_level(s.ptr, uint32(key), layout)
}

// SerializeMods serializes the requested modifier components.
func (s *State) SerializeMods(components StateComponent) ModMask {
	return xkb_state_serialize_mods(s.ptr, components)
}

// SerializeLayout serializes the requested layout components.
func (s *State) SerializeLayout(components StateComponent) LayoutIndex {
	return xkb_state_serialize_layout(s.ptr, components)
}

// ModNameIsActive reports whether the named modifier is active in typ.
func (s *State) ModNameIsActive(name string, typ StateComponent) bool {
	return xkb_state_mod_name_is_active(s.ptr, name, typ) == 1
}

// ModIndexIsActive reports whether modifier idx is active in typ.
func (s *State) ModIndexIsActive(idx ModIndex, typ StateComponent) bool {
	return xkb_state_mod_index_is_active(s.ptr, idx, typ) == 1
}

// ModIndexIsConsumed reports whether modifier idx is consumed by key.
func (s *State) ModIndexIsConsumed(key Keycode, idx ModIndex) bool {
	return xkb_state_mod_index_is_consumed(s.ptr, uint32(key), idx) == 1
}

// ModMaskRemoveConsumed removes the modifiers key consumes from mask.
func (s *State) ModMaskRemoveConsumed(key Keycode, mask ModMask) ModMask {
	return xkb_state_mod_mask_remove_consumed(s.ptr, uint32(key), mask)
}

// KeyGetConsumedMods returns the modifiers key consumes.
func (s *State) KeyGetConsumedMods(key Keycode) ModMask {
	return xkb_state_key_get_consumed_mods(s.ptr, uint32(key))
}

// LayoutNameIsActive reports whether the named layout is active in typ.
func (s *State) LayoutNameIsActive(name string, typ StateComponent) bool {
	return xkb_state_layout_name_is_active(s.ptr, name, typ) != 0
}

// LayoutIndexIsActive reports whether layout idx is active in typ.
func (s *State) LayoutIndexIsActive(idx LayoutIndex, typ StateComponent) bool {
	return xkb_state_layout_index_is_active(s.ptr, idx, typ) != 0
}

// LedNameIsActive reports whether the named LED is lit.
func (s *State) LedNameIsActive(name string) bool {
	return xkb_state_led_name_is_active(s.ptr, name) != 0
}

// LedIndexIsActive reports whether LED idx is lit.
func (s *State) LedIndexIsActive(idx LedIndex) bool {
	return xkb_state_led_index_is_active(s.ptr, idx) != 0
}

func clampLen(ret int32, max int) int {
	if ret < 0 {
		return 0
	}
	if int(ret) > max {
		return max
	}
	return int(ret)
}

// KeysymGetName returns the name of keysym, or "" when it has none.
func KeysymGetName(keysym Keysym) string {
	mustLoad()
	var buf [64]byte
	n := xkb_keysym_get_name(uint32(keysym), &buf[0], uintptr(len(buf)))
	if n <= 0 {
		return ""
	}
	return string(buf[:min(int(n), len(buf))])
}

// KeysymFromName looks up a keysym by name.
func KeysymFromName(name string, flags KeysymFlags) Keysym {
	mustLoad()
	return Keysym(xkb_keysym_from_name(name, flags))
}

// KeysymToUtf8 returns the UTF-8 text of keysym.
func KeysymToUtf8(keysym Keysym) string {
	mustLoad()
	var buf [8]byte
	n := xkb_keysym_to_utf8(uint32(keysym), &buf[0], uintptr(len(buf)))
	if n <= 0 {
		return ""
	}
	return string(buf[:n-1])
}

// KeysymToUtf32 returns the codepoint of keysym, or 0.
func KeysymToUtf32(keysym Keysym) uint32 {
	mustLoad()
	return xkb_keysym_to_utf32(uint32(keysym))
}

// Utf32ToKeysym returns the keysym of a codepoint.
func Utf32ToKeysym(ucs uint32) Keysym {
	mustLoad()
	return Keysym(xkb_utf32_to_keysym(ucs))
}
