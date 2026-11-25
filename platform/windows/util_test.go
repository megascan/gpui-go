//go:build windows

package windows

import (
	"testing"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

func TestUtil_HiLoWord(t *testing.T) {
	w := win32.WPARAM(0x1234_FFFE)
	if hiword(w) != 0x1234 || loword(w) != 0xFFFE || signedHiword(w) != 0x1234 || signedLoword(w) != -2 {
		t.Fatalf("WPARAM words: %#x %#x %d %d", hiword(w), loword(w), signedHiword(w), signedLoword(w))
	}
	l := win32.LPARAM(0xFFFF_0010)
	if hiword(l) != 0xFFFF || signedHiword(l) != -1 || loword(l) != 0x10 {
		t.Fatalf("LPARAM words: %#x %d %#x", hiword(l), signedHiword(l), loword(l))
	}
}

func TestUtil_CredentialsTargetName(t *testing.T) {
	if got := windowsCredentialsTargetName("https://zed.dev"); got != "zed:url=https://zed.dev" {
		t.Fatalf("target name %q", got)
	}
}

func TestUtil_LogicalPoint(t *testing.T) {
	if got := logicalPoint(30, 15, 1.5); got != (gpui.Point[gpui.Pixels]{X: 20, Y: 10}) {
		t.Fatalf("logicalPoint = %+v", got)
	}
}

func TestUtil_UIColorLayout(t *testing.T) {
	if size := unsafe.Sizeof(uiColor{}); size != 4 {
		t.Fatalf("uiColor size %d, want 4 (Windows.UI.Color)", size)
	}
}

func TestUtil_IsColorLight(t *testing.T) {
	if !isColorLight(&uiColor{A: 255, R: 255, G: 255, B: 255}) {
		t.Fatal("white is not light")
	}
	if isColorLight(&uiColor{A: 255}) {
		t.Fatal("black is light")
	}
	if isColorLight(&uiColor{R: 128, G: 128, B: 128}) {
		t.Fatal("8*128 is not above the threshold")
	}
}

func TestUtil_LoadCursorCaches(t *testing.T) {
	first, ok := loadCursor(gpui.CursorStyleIBeam)
	if !ok || first == 0 {
		t.Fatalf("loadCursor(IBeam) = %v, %v", first, ok)
	}
	second, _ := loadCursor(gpui.CursorStyleIBeamCursorForVerticalLayout)
	if second != first {
		t.Fatal("IBeam styles do not share a cursor")
	}
	arrow, _ := loadCursor(gpui.CursorStyleArrow)
	other, _ := loadCursor(gpui.CursorStyleContextualMenu)
	if arrow == 0 || arrow != other {
		t.Fatal("unmapped styles do not fall back to the arrow")
	}
}

func TestUtil_SystemAppearance(t *testing.T) {
	appearance, err := systemAppearance()
	if err != nil {
		t.Skipf("UISettings unavailable: %v", err)
	}
	if appearance != gpui.WindowAppearanceDark && appearance != gpui.WindowAppearanceLight {
		t.Fatalf("appearance %v", appearance)
	}
}

func TestUtil_WithDllLibrary(t *testing.T) {
	got, err := withDllLibrary("user32.dll", func(module win32.HMODULE) (bool, error) {
		return module != 0, nil
	})
	if err != nil || !got {
		t.Fatalf("withDllLibrary(user32) = %v, %v", got, err)
	}
	if _, err := withDllLibrary("gpui-missing-library.dll", func(win32.HMODULE) (bool, error) {
		t.Fatal("callback ran for a missing library")
		return false, nil
	}); err == nil {
		t.Fatal("missing library loaded")
	}
}
