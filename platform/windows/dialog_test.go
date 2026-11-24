//go:build windows

package windows

import (
	"testing"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

func onDialogThread(t *testing.T, f func()) {
	t.Helper()
	done := make(chan any, 1)
	err := spawnDialogThread(func() {
		defer func() { done <- recover() }()
		f()
	})
	if err != nil {
		t.Fatalf("spawning dialog thread: %v", err)
	}
	if p := <-done; p != nil {
		panic(p)
	}
}

func nowOrNever[T any](f scheduler.Future[T]) bool {
	_, ok := f.Poll(scheduler.NewContext(scheduler.WakerFn(func() {})))
	return ok
}

func TestDialog_TaskDialogCancellationRequiresCancelButtonOrInternalRequest(t *testing.T) {
	var failures []string
	onDialogThread(t, func() {
		c := newCancellation(nil)
		setThreadCancellation(c)
		defer takeThreadCancellation()
		for _, internallyCancelled := range []bool{false, true} {
			c.requested.Store(internallyCancelled)
			for _, hasCancelButton := range []bool{false, true} {
				for _, button := range []int32{win32.IDCANCEL, win32.IDOK, -1} {
					hasCancel := 0
					if hasCancelButton {
						hasCancel = 1
					}
					result := taskDialogCallback(0, win32.TDN_BUTTON_CLICKED, win32.WPARAM(uintptr(int(button))), 0, hasCancel)
					shouldVeto := button == win32.IDCANCEL && !hasCancelButton && !internallyCancelled
					want := com.S_OK
					if shouldVeto {
						want = com.S_FALSE
					}
					if result != want {
						failures = append(failures, "unexpected result")
					}
				}
			}
		}
	})
	if len(failures) != 0 {
		t.Fatalf("%d task dialog callback results were wrong", len(failures))
	}
}

func TestDialog_ClosingOwnerCancelsEveryOutstandingDialog(t *testing.T) {
	owner := NewDialogOwner(0)
	first := newCancellation(owner)
	second := newCancellation(owner)
	if first.isRequested() || second.isRequested() {
		t.Fatal("cancellation requested before the owner closed")
	}

	owner.close()
	owner.close()

	if !first.isRequested() || !second.isRequested() {
		t.Fatal("closing the owner did not cancel its dialogs")
	}
	if !newCancellation(owner).isRequested() {
		t.Fatal("a new request on a closed owner is not cancelled")
	}
}

func TestDialog_ClosedOwnerWaitsForActiveDialogButQueuedRequestsStayCancelled(t *testing.T) {
	owner := NewDialogOwner(0)
	active, ok := owner.serialization.tryLock()
	if !ok {
		t.Fatal("first dialog")
	}
	queued := newCancellation(owner)

	owner.close()
	if nowOrNever(owner.whenIdle()) {
		t.Fatal("owner idle while a dialog is active")
	}
	active.Close()
	if !nowOrNever(owner.whenIdle()) {
		t.Fatal("owner not idle after the active dialog ended")
	}

	guard, ok := owner.serialization.tryLock()
	if !ok {
		t.Fatal("queued request")
	}
	defer guard.Close()
	if !queued.isRequested() {
		t.Fatal("queued request is not cancelled")
	}
}

func TestDialog_CancellingOneRequestDoesNotCancelItsOwnerOrOtherRequests(t *testing.T) {
	owner := NewDialogOwner(0)
	first := newCancellation(owner)
	second := newCancellation(owner)
	first.requested.Store(true)
	if !first.isRequested() {
		t.Fatal("first request not cancelled")
	}
	if second.isRequested() {
		t.Fatal("second request cancelled")
	}
	if owner.closed.Load() {
		t.Fatal("owner closed")
	}
}

func TestDialog_DialogsSerializePerOwner(t *testing.T) {
	owner := NewDialogOwner(0)
	otherOwner := NewDialogOwner(0)
	guard, ok := owner.serialization.tryLock()
	if !ok {
		t.Fatal("first dialog")
	}

	if _, ok := owner.serialization.tryLock(); ok {
		t.Fatal("second dialog of the same owner started")
	}
	other, ok := otherOwner.serialization.tryLock()
	if !ok {
		t.Fatal("other owner's dialog blocked")
	}
	other.Close()
	guard.Close()
	again, ok := owner.serialization.tryLock()
	if !ok {
		t.Fatal("owner still locked after its dialog ended")
	}
	again.Close()
}

func TestDialog_OwnerlessRequestCanBeCancelledWithoutAWindow(t *testing.T) {
	c := newCancellation(nil)
	if c.isRequested() {
		t.Fatal("ownerless request cancelled")
	}
	c.requested.Store(true)
	if !c.isRequested() {
		t.Fatal("ownerless request not cancelled")
	}
}
