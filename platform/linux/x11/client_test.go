//go:build linux

package x11

import (
	"os"
	"testing"
	"time"

	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/platform/linux/x11/internal/xext"
)

func testKeymap(t *testing.T, layouts string) *xkb.Keymap {
	return testKeymapWithVariant(t, layouts, "")
}

func testKeymapWithVariant(t *testing.T, layouts, variant string) *xkb.Keymap {
	t.Helper()
	ctx := xkb.NewContext(xkb.ContextNoFlags)
	if ctx == nil {
		t.Fatal("context")
	}
	t.Cleanup(ctx.Close)
	keymap := xkb.NewKeymapFromNames(ctx, "", "pc105", layouts, variant, nil, xkb.KeymapCompileNoFlags)
	if keymap == nil {
		t.Fatal("test keymap should compile")
	}
	t.Cleanup(keymap.Close)
	return keymap
}

func newState(t *testing.T, keymap *xkb.Keymap) *xkb.State {
	t.Helper()
	s := xkb.NewState(keymap)
	t.Cleanup(s.Close)
	return s
}

func keyEventState(t *testing.T, s *xkb.State, mask uint16) *xkb.State {
	t.Helper()
	k := xkbStateForKeyEvent(s, mask)
	t.Cleanup(k.Close)
	return k
}

func keyByName(t *testing.T, keymap *xkb.Keymap, name string) xkb.Keycode {
	t.Helper()
	k, ok := keymap.KeyByName(name)
	if !ok {
		t.Fatalf("test key %s should exist in the keymap", name)
	}
	return k
}

func stateWithNonLockedLayout(t *testing.T, keymap *xkb.Keymap) *xkb.State {
	depressed := newState(t, keymap)
	depressed.UpdateMask(0, 0, 0, 1, 0, 0)
	if depressed.SerializeLayout(xkb.StateLayoutEffective) == 1 {
		return depressed
	}
	latched := newState(t, keymap)
	latched.UpdateMask(0, 0, 0, 0, 1, 0)
	if latched.SerializeLayout(xkb.StateLayoutEffective) == 1 {
		return latched
	}
	panic("test keymap should support a non-locked secondary layout")
}

func eq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestClient_KeyEventStateUsesEventModifiersWithoutMutatingServerState(t *testing.T) {
	keymap := testKeymap(t, "us")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AE09")
	k := keyEventState(t, server, xproto.KeyButMaskShift)
	keystroke := common.KeystrokeFromXkb(k, modifiersFromState(xproto.KeyButMaskShift), keycode)
	eq(t, keystroke.Key, "(")
	eq(t, keystroke.KeyChar, gpui.Some("("))
	eq(t, server.KeyGetUtf8(keycode), "9")
}

func TestClient_KeyEventStateIgnoresPointerButtonBits(t *testing.T) {
	keymap := testKeymap(t, "us")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AE09")
	shifted := keyEventState(t, server, xproto.KeyButMaskShift)
	withButton := keyEventState(t, server, xproto.KeyButMaskShift|xproto.KeyButMaskButton1)
	eq(t, withButton.SerializeMods(xkb.StateModsEffective), shifted.SerializeMods(xkb.StateModsEffective))
	eq(t, withButton.KeyGetUtf8(keycode), shifted.KeyGetUtf8(keycode))
}

func TestClient_KeyEventStatePreservesNonLockedLayoutComponents(t *testing.T) {
	keymap := testKeymap(t, "us,ru")
	server := stateWithNonLockedLayout(t, keymap)
	keycode := keyByName(t, keymap, "AD01")
	expected := server.KeyGetUtf8(keycode)
	k := keyEventState(t, server, 0)
	eq(t, k.SerializeLayout(xkb.StateLayoutEffective), server.SerializeLayout(xkb.StateLayoutEffective))
	eq(t, k.KeyGetUtf8(keycode), expected)
}

func TestClient_CapslockToggleProducesUppercase(t *testing.T) {
	keymap := testKeymap(t, "us")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AC01")
	lockMod := xkb.ModMask(xproto.KeyButMaskLock)
	server.UpdateMask(0, 0, lockMod, 0, 0, 0)
	k := keyEventState(t, server, xproto.KeyButMaskLock)
	eq(t, k.SerializeMods(xkb.StateModsLocked)&lockMod, lockMod)
	eq(t, k.KeyGetUtf8(keycode), "A")
}

func TestClient_Neo2Level3ViaCapslockProducesEllipsis(t *testing.T) {
	keymap := testKeymapWithVariant(t, "de", "neo")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AD01")
	k := keyEventState(t, server, xproto.KeyButMaskMod5)
	eq(t, k.KeyGetUtf8(keycode), "…")
}

func TestClient_Neo2LatchedMod5Preserved(t *testing.T) {
	keymap := testKeymapWithVariant(t, "de", "neo")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AD01")
	mod5 := xkb.ModMask(xproto.KeyButMaskMod5)
	server.UpdateMask(0, mod5, 0, 0, 0, 0)
	k := keyEventState(t, server, xproto.KeyButMaskMod5)
	eq(t, k.SerializeMods(xkb.StateModsLatched)&mod5, mod5)
	eq(t, k.KeyGetUtf8(keycode), "…")
}

func TestClient_GermanLayoutCorrectKeyResolution(t *testing.T) {
	keymap := testKeymap(t, "de")
	server := newState(t, keymap)
	keycode := keyByName(t, keymap, "AE07")
	k := keyEventState(t, server, 0)
	eq(t, k.KeyGetUtf8(keycode), "7")
}

func TestClient_SpaceWorksWithCyrillicLayoutActive(t *testing.T) {
	keymap := testKeymap(t, "us,ru")
	server := newState(t, keymap)
	space := keyByName(t, keymap, "SPCE")
	server.UpdateMask(0, 0, 0, 0, 0, 1)
	k := keyEventState(t, server, 0)
	eq(t, k.SerializeLayout(xkb.StateLayoutEffective), 1)
	eq(t, k.KeyGetUtf8(space), " ")
}

func TestClient_MacroShiftBracketProducesBrace(t *testing.T) {
	keymap := testKeymap(t, "us")
	server := newState(t, keymap)
	bracket := keyByName(t, keymap, "AD12")
	eq(t, server.SerializeMods(xkb.StateModsEffective), 0)
	k := keyEventState(t, server, xproto.KeyButMaskShift)
	eq(t, k.KeyGetUtf8(bracket), "}")
}

func TestClient_SequentialKeyEventsDoNotCorruptState(t *testing.T) {
	keymap := testKeymap(t, "us")
	server := newState(t, keymap)
	keys := [][2]string{{"AC01", "a"}, {"SPCE", " "}, {"AC02", "s"}, {"SPCE", " "}, {"AC03", "d"}}
	for _, kv := range keys {
		keycode := keyByName(t, keymap, kv[0])
		k := keyEventState(t, server, 0)
		if got := k.KeyGetUtf8(keycode); got != kv[1] {
			t.Fatalf("key %s should produce %q, got %q", kv[0], kv[1], got)
		}
	}
	eq(t, server.SerializeMods(xkb.StateModsEffective), 0)
	eq(t, server.SerializeLayout(xkb.StateLayoutEffective), 0)
}

func TestClient_SpaceWorksWithCzechLayoutActive(t *testing.T) {
	keymap := testKeymap(t, "us,cz")
	server := newState(t, keymap)
	space := keyByName(t, keymap, "SPCE")
	server.UpdateMask(0, 0, 0, 0, 0, 1)
	k := keyEventState(t, server, 0)
	eq(t, k.KeyGetUtf8(space), " ")
}

func TestClient_ModeRefreshRate(t *testing.T) {
	eq(t, ModeRefreshRate(&randr.ModeInfo{}), 16*time.Millisecond)
	eq(t, ModeRefreshRate(&randr.ModeInfo{DotClock: 148_500_000, Htotal: 2200, Vtotal: 1125}), 16_666*time.Microsecond)
}

func TestClient_AttachDisplay(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" {
		t.Skip("set GPUI_TEST_DISPLAY=1 to run against a live X server")
	}
	conn, err := xext.Dial(os.Getenv("DISPLAY"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.X.Close()
	if _, err := NewXcbAtoms(conn.X); err != nil {
		t.Fatal(err)
	}
}
