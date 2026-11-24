//go:build windows

package windows

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func TestDisplay_UUIDV5MatchesUuidCrate(t *testing.T) {
	if got := newUUIDV5(namespaceDNS, []byte("python.org")).String(); got != "886313e1-3b8a-5372-9b90-0c9aee199e5d" {
		t.Fatalf("uuid v5 = %s", got)
	}
	var device [32]uint16
	for i, c := range `\\.\DISPLAY1` {
		device[i] = uint16(c)
	}
	if got := generateUUID(device[:]).String(); got != "78692035-3a04-52f3-a2bf-1133155ae828" {
		t.Fatalf("generateUUID = %s", got)
	}
}

func TestDisplay_EnumeratesMonitors(t *testing.T) {
	displays := WindowsDisplayDisplays()
	primary, ok := WindowsDisplayPrimaryMonitor()
	if !ok {
		if len(displays) != 0 {
			t.Fatal("monitors enumerated but no primary monitor")
		}
		t.Skip("no monitor attached")
	}
	var _ gpui.PlatformDisplay = primary
	if primary.ScaleFactor() <= 0 {
		t.Fatalf("scale factor %v", primary.ScaleFactor())
	}
	found := false
	for _, display := range displays {
		if display.ID() == primary.ID() {
			found = true
		}
	}
	if !found {
		t.Fatal("primary monitor missing from WindowsDisplayDisplays")
	}
	if !primary.CheckGivenBounds(primary.Bounds()) {
		t.Fatal("primary monitor does not contain its own bounds")
	}
	if b := primary.PhysicalBounds(); b.Size.Width <= 0 || b.Size.Height <= 0 {
		t.Fatalf("physical bounds %+v", b)
	}
	if monitorLists.Len() != 0 {
		t.Fatal("monitor enumeration left its list in the handle table")
	}
}
