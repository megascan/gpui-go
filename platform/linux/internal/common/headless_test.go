//go:build linux

package common

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestHeadless_WindowLeaseTracksOpenWindows(t *testing.T) {
	connection := NewHeadlessConnection()
	if connection.HasWindows() {
		t.Fatal("new connection has windows")
	}
	window, err := connection.OpenWindow(gpui.AnyWindowHandle{}, gpui.WindowParams{})
	if err != nil {
		t.Fatal(err)
	}
	if !connection.HasWindows() {
		t.Fatal("open window not counted")
	}
	window.SetTitle("title")
	if window.GetTitle() != "title" {
		t.Fatalf("title = %q", window.GetTitle())
	}
	window.Close()
	window.Close()
	if connection.HasWindows() {
		t.Fatal("closed window still counted")
	}
}
