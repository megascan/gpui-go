package main

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestAnimation_ClickingSpringPositionChangesTheTarget(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		view, cx := tc.Cx().AddWindowView(func(_ *gpui.Window, _ *gpui.Context[AnimationExample]) *AnimationExample {
			return &AnimationExample{
				springPhase:       0,
				springDamping:     14.0,
				springDampingDrag: gpui.None[dampingDrag](),
			}
		})
		defer view.Release()
		cx.SimulateResize(gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)))
		cx.RunUntilParked()

		positionBounds, ok := cx.DebugBounds("spring-position")
		if !ok {
			panic("spring position should be rendered")
		}
		position := positionBounds.Center()
		cx.SimulateMouseMove(position, gpui.None[gpui.MouseButton](), gpui.Modifiers{})
		cx.SimulateClick(position, gpui.Modifiers{})
		cx.RunUntilParked()

		if got := cx.UpdateR(func(_ *gpui.Window, cx *gpui.App) uint8 { return view.Read(cx).springPhase }); got != 1 {
			t.Fatalf("spring_phase = %d, want 1", got)
		}
	})
}
