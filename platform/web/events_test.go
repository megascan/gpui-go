//go:build js && wasm

package web

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestEvents_BrowserPointerIDReuseGetsANewTouchID(t *testing.T) {
	var touchIDs touchIDs
	first, ok := touchIDs.start(7)
	if !ok {
		t.Fatal("first touch id")
	}
	concurrent, ok := touchIDs.start(8)
	if !ok {
		t.Fatal("concurrent touch id")
	}

	if first == concurrent {
		t.Fatalf("first %v == concurrent %v", first, concurrent)
	}
	if id, ok := touchIDs.activeID(7); !ok || id != first {
		t.Fatalf("active(7) = %v, %v", id, ok)
	}
	if id, ok := touchIDs.end(7); !ok || id != first {
		t.Fatalf("end(7) = %v, %v", id, ok)
	}
	if _, ok := touchIDs.activeID(7); ok {
		t.Fatal("active(7) after end")
	}

	reused, ok := touchIDs.start(7)
	if !ok {
		t.Fatal("reused pointer touch id")
	}
	if reused == first || reused == concurrent {
		t.Fatalf("reused %v collides", reused)
	}
}

func TestEvents_ShiftIsDroppedOnlyFromSymbolKeys(t *testing.T) {
	shift := gpui.ModifiersShift()
	for _, c := range []struct {
		key       string
		modifiers gpui.Modifiers
		want      gpui.Modifiers
	}{
		{"!", shift, gpui.ModifiersNone()},
		{"?", shift, gpui.ModifiersNone()},
		{"a", shift, shift},
		{"tab", shift, shift},
		{"!", gpui.ModifiersControlShift(), gpui.ModifiersControl()},
	} {
		if got := keystrokeModifiers(c.key, c.modifiers); got != c.want {
			t.Errorf("keystrokeModifiers(%q, %+v) = %+v, want %+v", c.key, c.modifiers, got, c.want)
		}
	}
}

func TestEvents_DiffImeEditAnchorsSuffixAtCaret(t *testing.T) {
	edit := diffImeEdit("pact", "pactor pact", gpui.Some[uint32](7), [2]uint32{0, 0})
	if edit.insertedText != "pactor " || edit.removedBeforeSelection != 0 || edit.removedAfterSelection != 0 {
		t.Fatalf("anchored: %+v", edit)
	}
	edit = diffImeEdit("helo wrld", "helo world", gpui.None[uint32](), [2]uint32{9, 9})
	if edit.insertedText != "o" || edit.removedBeforeSelection != 3 || edit.removedAfterSelection != 0 {
		t.Fatalf("greedy: %+v", edit)
	}
	edit = diffImeEdit("abc", "a", gpui.Some[uint32](1), [2]uint32{3, 3})
	if edit.insertedText != "" || edit.removedBeforeSelection != 2 {
		t.Fatalf("delete: %+v", edit)
	}
}
