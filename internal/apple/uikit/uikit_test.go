//go:build darwin

package uikit

import "testing"

func TestUikit_ModifierFlagsContains(t *testing.T) {
	flags := UIKeyModifierShift | UIKeyModifierCommand
	if !flags.Contains(UIKeyModifierShift) || flags.Contains(UIKeyModifierControl) {
		t.Fatalf("Contains wrong for %#x", flags)
	}
}

func TestUikit_RectEdgeAll(t *testing.T) {
	if UIRectEdgeAll != 15 {
		t.Fatalf("UIRectEdgeAll = %d", UIRectEdgeAll)
	}
}
