//go:build darwin

package ios

import "testing"

func TestWindow_CallbackIsUnavailableDuringDispatchAndRestoredAfterward(t *testing.T) {
	var slot callbackSlot[int]
	slot.set(1)
	got, ok := callbackSlotWithR(&slot, func(value *int) int {
		*value++
		if slot.with(func(*int) { t.Fatal("reentered the same handler") }) {
			t.Fatal("reentered the same handler")
		}
		return *value
	})
	if !ok || got != 2 {
		t.Fatalf("with = %d, %t; want 2, true", got, ok)
	}
	got, ok = callbackSlotWithR(&slot, func(value *int) int { return *value })
	if !ok || got != 2 {
		t.Fatalf("with = %d, %t; want 2, true", got, ok)
	}
}

func TestWindow_CallbackReplacementDuringDispatchIsPreserved(t *testing.T) {
	var slot callbackSlot[int]
	slot.set(1)
	slot.with(func(*int) {
		slot.set(2)
		slot.with(func(value *int) { *value++ })
	})
	if got, ok := slot.take(); !ok || got != 3 {
		t.Fatalf("take = %d, %t; want 3, true", got, ok)
	}
}

func TestWindow_WithdrawingAnInFlightHandlerDoesNotRestoreIt(t *testing.T) {
	var slot callbackSlot[int]
	slot.set(1)
	slot.with(func(*int) {
		if _, ok := slot.take(); ok {
			t.Fatal("take during dispatch returned the in-flight handler")
		}
	})
	if slot.with(func(*int) { t.Fatal("restored a withdrawn handler") }) {
		t.Fatal("restored a withdrawn handler")
	}
}

func TestWindow_RemovingAReplacementDoesNotResurrectThePreviousHandler(t *testing.T) {
	var slot callbackSlot[int]
	slot.set(1)
	slot.with(func(*int) {
		slot.set(2)
		if got, ok := slot.take(); !ok || got != 2 {
			t.Fatalf("take = %d, %t; want 2, true", got, ok)
		}
	})
	if _, ok := slot.take(); ok {
		t.Fatal("removed replacement resurrected the previous handler")
	}
}
