//go:build linux

package wayland

import "testing"

func rawSelectionSerial(t *SerialTracker) (uint32, bool) {
	s, ok := t.SelectionSerial()
	if !ok {
		return 0, false
	}
	return s.AsRaw(), true
}

func assertSelection(t *testing.T, tracker *SerialTracker, want uint32, wantOK bool) {
	t.Helper()
	got, ok := rawSelectionSerial(tracker)
	if ok != wantOK || got != want {
		t.Fatalf("selection serial = (%d, %v), want (%d, %v)", got, ok, want, wantOK)
	}
}

func TestSerial_SelectionSerialIgnoresUnrelatedSerialKinds(t *testing.T) {
	tracker := NewSerialTracker()
	tracker.Update(SerialKindMousePress, 3783)
	tracker.Update(SerialKindInputMethod, 5011)
	tracker.Update(SerialKindKeyPress, 3787)
	tracker.Update(SerialKindMouseEnter, 6000)
	tracker.Update(SerialKindDataDevice, 7000)
	assertSelection(t, tracker, 3787, true)
}

func TestSerial_SelectionSerialUsesMousePressWithoutKeyPress(t *testing.T) {
	tracker := NewSerialTracker()
	tracker.Update(SerialKindMousePress, 3783)
	assertSelection(t, tracker, 3783, true)
}

func TestSerial_SelectionSerialUsesEventArrivalOrderAcrossRollover(t *testing.T) {
	tracker := NewSerialTracker()
	tracker.Update(SerialKindKeyPress, 0xffff_fff0)
	tracker.Update(SerialKindMousePress, 0x0000_0010)
	assertSelection(t, tracker, 0x0000_0010, true)
}

func TestSerial_SelectionSerialIsUnavailableWithoutEligibleInput(t *testing.T) {
	tracker := NewSerialTracker()
	assertSelection(t, tracker, 0, false)
	tracker.Update(SerialKindInputMethod, 5011)
	tracker.Update(SerialKindMouseEnter, 6000)
	tracker.Update(SerialKindDataDevice, 7000)
	assertSelection(t, tracker, 0, false)
}

func TestSerial_ZeroIsAValidSelectionSerial(t *testing.T) {
	tracker := NewSerialTracker()
	tracker.Update(SerialKindKeyPress, 0)
	assertSelection(t, tracker, 0, true)
}
