//go:build linux

package wayland

import (
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func TestClientInputLinuxButtonToGPUI(t *testing.T) {
	cases := map[uint32]gpui.MouseButton{
		0x110: gpui.MouseButtonLeft,
		0x111: gpui.MouseButtonRight,
		0x112: gpui.MouseButtonMiddle,
		0x113: gpui.MouseButtonNavigateBack,
		0x116: gpui.MouseButtonNavigateBack,
		0x114: gpui.MouseButtonNavigateForward,
		0x115: gpui.MouseButtonNavigateForward,
	}
	for raw, want := range cases {
		if got, ok := linuxButtonToGPUI(raw); !ok || got != want {
			t.Errorf("linuxButtonToGPUI(%#x) = %v, %v; want %v", raw, got, ok, want)
		}
	}
	if _, ok := linuxButtonToGPUI(0x117); ok {
		t.Error("unknown button mapped")
	}
}

func TestClientInputParsesURIList(t *testing.T) {
	var paths []string
	for _, line := range rustLines("file:///tmp/a%20b\r\nfile://localhost/x\r\nrelative\r\nhttp://h/y\r\nfile://other/z\r\n") {
		if path, err := fileURLToPath(line); err == nil {
			paths = append(paths, path)
		}
	}
	if want := []string{"/tmp/a b", "/x"}; !slices.Equal(paths, want) {
		t.Fatalf("paths = %q, want %q", paths, want)
	}
}
