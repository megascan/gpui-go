//go:build darwin

package appkit

import (
	"testing"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

func TestAppkit_ModifierFlagsContains(t *testing.T) {
	flags := NSEventModifierFlagShift | NSEventModifierFlagCommand
	if !flags.Contains(NSEventModifierFlagShift) || flags.Contains(NSEventModifierFlagControl) {
		t.Fatalf("Contains wrong for %#x", flags)
	}
}

func TestAppkit_LoadAndConstants(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"NSApplication", "NSWindow", "NSView", "NSEvent", "NSScreen", "NSPasteboard", "NSMenu", "NSMenuItem", "NSCursor", "NSColor", "NSAppearance", "NSTrackingArea"} {
		if class(name) == 0 {
			t.Fatalf("class %s missing", name)
		}
	}
	if got := objc.GoString(NSPasteboardTypeString()); got != "public.utf8-plain-text" {
		t.Fatalf("NSPasteboardTypeString = %q", got)
	}
	if NSAppearanceNamed(NSAppearanceNameAqua()) == 0 {
		t.Fatal("aqua appearance missing")
	}
	if NSColorWithSRGB(1, 0, 0, 1) == 0 {
		t.Fatal("NSColor missing")
	}
}

func TestAppkit_UniquePasteboard(t *testing.T) {
	objc.AutoreleasePool(func() {
		pb := NSPasteboardWithUniqueName()
		defer pb.ReleaseGlobally()
		pb.ClearContents()
		s := objc.NSStringFromGo("hello")
		defer objc.Release(s)
		if !pb.SetStringForType(s, NSPasteboardTypeString()) {
			t.Fatal("setString failed")
		}
		if got := objc.GoString(pb.StringForType(NSPasteboardTypeString())); got != "hello" {
			t.Fatalf("stringForType = %q", got)
		}
	})
}
