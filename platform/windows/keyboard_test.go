//go:build windows

package windows

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func TestKeyboard_KeyboardMapper(t *testing.T) {
	mapper := NewWindowsKeyboardMapper()

	keystroke := gpui.Keystroke{Modifiers: gpui.ModifiersControl(), Key: "a"}
	mapped := mapper.MapKeyEquivalent(keystroke, true)
	if *mapped.Inner() != keystroke {
		t.Fatalf("inner = %+v, want %+v", *mapped.Inner(), keystroke)
	}
	if mapped.Key() != "a" {
		t.Fatalf("key = %q, want %q", mapped.Key(), "a")
	}
	if mapped.Modifiers() != gpui.ModifiersControl() {
		t.Fatalf("modifiers = %+v", mapped.Modifiers())
	}

	keystroke = gpui.Keystroke{Modifiers: gpui.ModifiersControl(), Key: "$"}
	mapped = mapper.MapKeyEquivalent(keystroke, true)
	if *mapped.Inner() != keystroke {
		t.Fatalf("inner = %+v, want %+v", *mapped.Inner(), keystroke)
	}
	if mapped.Key() != "4" {
		t.Fatalf("key = %q, want %q", mapped.Key(), "4")
	}
	if mapped.Modifiers() != gpui.ModifiersControlShift() {
		t.Fatalf("modifiers = %+v", mapped.Modifiers())
	}

	keystroke = gpui.Keystroke{Modifiers: gpui.ModifiersControlShift(), Key: "$"}
	mapped = mapper.MapKeyEquivalent(keystroke, true)
	if mapped.Inner().Modifiers != gpui.ModifiersControl() {
		t.Fatalf("inner modifiers = %+v", mapped.Inner().Modifiers)
	}
	if mapped.Key() != "4" {
		t.Fatalf("key = %q, want %q", mapped.Key(), "4")
	}
	if mapped.Modifiers() != gpui.ModifiersControlShift() {
		t.Fatalf("modifiers = %+v", mapped.Modifiers())
	}

	keystroke = gpui.Keystroke{Modifiers: gpui.ModifiersControlShift(), Key: "4"}
	mapped = mapper.MapKeyEquivalent(keystroke, true)
	if mapped.Inner().Modifiers != gpui.ModifiersControl() {
		t.Fatalf("inner modifiers = %+v", mapped.Inner().Modifiers)
	}
	if mapped.Inner().Key != "$" {
		t.Fatalf("inner key = %q, want %q", mapped.Inner().Key, "$")
	}
	if mapped.Key() != "4" {
		t.Fatalf("key = %q, want %q", mapped.Key(), "4")
	}
	if mapped.Modifiers() != gpui.ModifiersControlShift() {
		t.Fatalf("modifiers = %+v", mapped.Modifiers())
	}
}
