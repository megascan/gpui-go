//go:build darwin && !ios

package macos

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func noHeight() gpui.Option[gpui.Pixels] { return gpui.None[gpui.Pixels]() }

func TestKeyboard_KeyEquivalents(t *testing.T) {
	if getKeyEquivalents("com.apple.keylayout.US") != nil {
		t.Fatal("US layout has no key equivalents")
	}
	german := getKeyEquivalents("com.apple.keylayout.ABC-QWERTZ")
	if german['['] != 'ö' || german['}'] != 'Ä' || len(german) != 23 {
		t.Fatalf("ABC-QWERTZ = %v", german)
	}
	if welsh := getKeyEquivalents("com.apple.keylayout.Welsh"); len(welsh) != 1 || welsh['#'] != '£' {
		t.Fatalf("Welsh = %v", welsh)
	}
}

func TestKeyboard_MapKeyEquivalent(t *testing.T) {
	mapper := NewMacKeyboardMapper("com.apple.keylayout.ABC-QWERTZ")
	keystroke := gpui.Keystroke{Modifiers: gpui.Modifiers{Platform: true}, Key: "["}
	if got := mapper.MapKeyEquivalent(keystroke, true).Key(); got != "ö" {
		t.Fatalf("mapped key = %q", got)
	}
	if got := mapper.MapKeyEquivalent(keystroke, false).Key(); got != "[" {
		t.Fatalf("unmapped key = %q", got)
	}
}
