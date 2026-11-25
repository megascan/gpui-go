//go:build darwin

package ios

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/uikit"
)

func TestEvents_TouchPhaseFromNative(t *testing.T) {
	cases := map[uikit.UITouchPhase]gpui.TouchPhase{
		uikit.UITouchPhaseBegan:         gpui.TouchPhaseStarted,
		uikit.UITouchPhaseMoved:         gpui.TouchPhaseMoved,
		uikit.UITouchPhaseStationary:    gpui.TouchPhaseMoved,
		uikit.UITouchPhaseEnded:         gpui.TouchPhaseEnded,
		uikit.UITouchPhaseCancelled:     gpui.TouchPhaseCancelled,
		uikit.UITouchPhaseRegionEntered: gpui.TouchPhaseCancelled,
	}
	for native, want := range cases {
		if got := touchPhaseFromNative(native); got != want {
			t.Fatalf("touchPhaseFromNative(%d) = %d, want %d", native, got, want)
		}
	}
}

func TestEvents_BackReleaseAndForce(t *testing.T) {
	if backReleasePhase(0.6, 0) != gpui.BackGesturePhaseCommitted || backReleasePhase(0.6, -400) != gpui.BackGesturePhaseCancelled || backReleasePhase(0.1, 400) != gpui.BackGesturePhaseCommitted || backReleasePhase(0.4, 0) != gpui.BackGesturePhaseCancelled {
		t.Fatal("back release phases wrong")
	}
	if _, ok := touchForce(1, 0); ok {
		t.Fatal("force without a maximum")
	}
	if f, ok := touchForce(3, 2); !ok || f != 1 {
		t.Fatalf("force = %v %v", f, ok)
	}
}
