//go:build darwin && !ios

package macos

import (
	"testing"

	"github.com/megascan/gpui-go/internal/apple/appkit"
)

func TestEvents_KeyToMenuEquivalent(t *testing.T) {
	cases := map[string]string{
		"space":     " ",
		"backspace": "\b",
		"escape":    "\x1b",
		"delete":    "\x7f",
		"up":        string(rune(appkit.NSUpArrowFunctionKey)),
		"insert":    string(rune(appkit.NSHelpFunctionKey)),
		"f35":       string(rune(appkit.NSF35FunctionKey)),
		"a":         "a",
		"enter":     "enter",
	}
	for key, want := range cases {
		if got := keyToMenuEquivalent(key); got != want {
			t.Errorf("keyToMenuEquivalent(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestEvents_AsciiHelpers(t *testing.T) {
	if asciiToUpper("sö") != "Sö" {
		t.Fatal("asciiToUpper must touch ASCII only")
	}
	if !allASCIILower("") || !allASCIILower("abc") || allASCIILower("aB") || allASCIILower("ö") {
		t.Fatal("allASCIILower")
	}
	if !isASCII("az!") || isASCII("ս") {
		t.Fatal("isASCII")
	}
	if stringFromUTF16OrDefault([]uint16{0xd800}) != "" || stringFromUTF16OrDefault([]uint16{'a', 0x0301}) != "á" {
		t.Fatal("stringFromUTF16OrDefault")
	}
}

func TestEvents_NilEvent(t *testing.T) {
	if platformInputFromNative(0, noHeight()) != nil {
		t.Fatal("nil event must map to nil")
	}
}
