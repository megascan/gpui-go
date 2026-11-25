//go:build linux

package xkb

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func newTestContext(t *testing.T) *Context {
	t.Helper()
	if err := Load(); err != nil {
		t.Skipf("libxkbcommon unavailable: %v", err)
	}
	ctx := NewContext(ContextNoEnvironmentNames)
	if ctx.GetRawPtr() == 0 {
		t.Fatal("libxkbcommon failed to create an XKB context")
	}
	t.Cleanup(ctx.Close)
	return ctx
}

func newTestKeymap(t *testing.T, ctx *Context) *Keymap {
	t.Helper()
	keymap := NewKeymapFromNames(ctx, "evdev", "pc105", "us", "", nil, KeymapCompileNoFlags)
	if keymap == nil {
		t.Skip("xkeyboard-config data unavailable")
	}
	t.Cleanup(keymap.Close)
	return keymap
}

func TestXkb_DefaultKeymapLetterA(t *testing.T) {
	ctx := newTestContext(t)
	keymap := newTestKeymap(t, ctx)
	state := NewState(keymap)
	defer state.Close()

	a, ok := keymap.KeyByName("AC01")
	if !ok {
		t.Fatal("AC01 missing")
	}
	if got := state.KeyGetUtf8(a); got != "a" {
		t.Fatalf("utf8 = %q, want a", got)
	}
	if got := state.KeyGetOneSym(a); got != KEY_a {
		t.Fatalf("keysym = %#x, want KEY_a", got)
	}
	if got := KeysymGetName(state.KeyGetOneSym(a)); got != "a" {
		t.Fatalf("name = %q", got)
	}
	if state.ModNameIsActive(ModNameShift, StateModsEffective) {
		t.Fatal("shift active before press")
	}

	lfsh, ok := keymap.KeyByName("LFSH")
	if !ok {
		t.Fatal("LFSH missing")
	}
	if !state.KeyGetOneSym(lfsh).IsModifierKey() {
		t.Fatal("Shift_L is not a modifier key")
	}
	state.UpdateKey(lfsh, KeyDown)
	if !state.ModNameIsActive(ModNameShift, StateModsEffective) {
		t.Fatal("shift inactive after press")
	}
	if got := state.KeyGetUtf8(a); got != "A" {
		t.Fatalf("shifted utf8 = %q, want A", got)
	}
	if got := state.KeyGetUtf32(a); got != 'A' {
		t.Fatalf("shifted utf32 = %d", got)
	}

	mods := state.SerializeMods(StateModsDepressed)
	other := NewState(keymap)
	defer other.Close()
	other.UpdateMask(mods, 0, 0, 0, 0, 0)
	if got := other.KeyGetUtf8(a); got != "A" {
		t.Fatalf("update_mask utf8 = %q, want A", got)
	}
	state.UpdateKey(lfsh, KeyUp)
	if got := state.KeyGetUtf8(a); got != "a" {
		t.Fatalf("released utf8 = %q", got)
	}

	layout := state.SerializeLayout(StateLayoutEffective)
	if name := keymap.LayoutGetName(layout); name == "" {
		t.Fatal("empty layout name")
	}
	km := state.GetKeymap()
	km.Close()
	km.Close()
}

func TestXkb_KeymapFromStringAndFd(t *testing.T) {
	ctx := newTestContext(t)
	keymap := newTestKeymap(t, ctx)
	text := keymap.GetAsString(KeymapFormatTextV1)
	if text == "" {
		t.Fatal("empty keymap string")
	}
	fromString := NewKeymapFromString(ctx, text, KeymapFormatTextV1, KeymapCompileNoFlags)
	if fromString == nil {
		t.Fatal("keymap from string failed")
	}
	fromString.Close()

	f, err := os.CreateTemp(t.TempDir(), "keymap")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(text + "\x00"); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Dup(int(f.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	fromFd, err := NewKeymapFromFd(ctx, fd, len(text)+1, KeymapFormatTextV1, KeymapCompileNoFlags)
	if err != nil || fromFd == nil {
		t.Fatalf("keymap from fd: %v", err)
	}
	defer fromFd.Close()
	if _, ok := fromFd.KeyByName("AE09"); !ok {
		t.Fatal("AE09 missing")
	}
}

func TestXkb_Compose(t *testing.T) {
	ctx := newTestContext(t)
	table, ok := NewComposeTableFromBuffer(ctx, []byte("<dead_acute> <e> : \"é\" eacute\n"), "C", ComposeFormatTextV1, ComposeCompileNoFlags)
	if !ok {
		t.Fatal("compose table failed")
	}
	defer table.Close()
	state := NewComposeState(table, ComposeStateNoFlags)
	defer state.Close()
	if state.Feed(KEY_dead_acute) != FeedResultAccepted || state.Status() != StatusComposing {
		t.Fatal("not composing")
	}
	state.Feed(KEY_e)
	if state.Status() != StatusComposed {
		t.Fatalf("status = %d", state.Status())
	}
	if s, ok := state.Utf8(); !ok || s != "é" {
		t.Fatalf("utf8 = %q", s)
	}
	if k, ok := state.Keysym(); !ok || k != KEY_eacute {
		t.Fatalf("keysym = %#x", k)
	}
	state.Reset()
	if state.Status() != StatusNothing {
		t.Fatal("reset failed")
	}
	if _, ok := state.Keysym(); ok {
		t.Fatal("keysym after reset")
	}
}

func TestXkb_Keysyms(t *testing.T) {
	if err := Load(); err != nil {
		t.Skipf("libxkbcommon unavailable: %v", err)
	}
	if !KEY_KP_Prior.IsKeypadKey() || KEY_Prior.IsKeypadKey() {
		t.Fatal("IsKeypadKey")
	}
	if KEY_a.IsModifierKey() || !KEY_Num_Lock.IsModifierKey() || !KEY_Mode_switch.IsModifierKey() {
		t.Fatal("IsModifierKey")
	}
	if KeysymFromName("BackSpace", KeysymNoFlags) != KEY_BackSpace {
		t.Fatal("KeysymFromName")
	}
	if KeysymToUtf8(KEY_eacute) != "é" || KeysymToUtf32(KEY_eacute) != 0xe9 || Utf32ToKeysym('a') != KEY_a {
		t.Fatal("utf conversions")
	}
	if KeysymGetName(Keysym(0xffffffff)) == "BackSpace" {
		t.Fatal("bogus name")
	}
}

func TestXkb_RejectsNullXkbContext(t *testing.T) {
	ctx := ContextFromRawPtr(0)
	if ctx.GetRawPtr() != 0 {
		t.Fatal("null XKB context should be rejected")
	}
	ctx.Close()
	if ctx.Clone().GetRawPtr() != 0 {
		t.Fatal("clone of null context not null")
	}
}
