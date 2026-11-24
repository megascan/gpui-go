//go:build darwin && !ios

package macos

import (
	"cmp"
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func nativeViewsTestFocused(view uint64, shown bool) gpui.Option[focusedNativeView] {
	return gpui.Some(focusedNativeView{id: nativeViewsTestID(view), shown: shown})
}

func nativeViewsTestID(id uint64) gpui.NativeViewID {
	return gpui.NativeViewIDForTest(id)
}

func nativeViewsTestHolder(kind inputHolderKind, id uint64) inputHolder {
	return inputHolder{kind: kind, id: nativeViewsTestID(id)}
}

func TestNativeViews_PlacementClipsAViewToItsContentMask(t *testing.T) {
	scaled := func(x, y, width, height float32) gpui.Bounds[gpui.ScaledPixels] {
		return gpui.Bounds[gpui.ScaledPixels]{
			Origin: gpui.NewPoint(gpui.ScaledPixels(x), gpui.ScaledPixels(y)),
			Size:   gpui.NewSize(gpui.ScaledPixels(width), gpui.ScaledPixels(height)),
		}
	}
	gpuiFrame := [4]float64{0, 10, 400, 300}
	bounds := scaled(100, 40, 200, 400)

	unclipped := newPlacement(bounds, scaled(0, 0, 800, 600), 2, gpuiFrame)
	if want := gpui.Some(placement{container: [4]float64{50, 90, 100, 200}, view: [4]float64{0, 0, 100, 200}}); unclipped != want {
		t.Fatalf("unclipped = %+v, want %+v", unclipped, want)
	}

	clipped := newPlacement(bounds, scaled(120, 100, 800, 600), 2, gpuiFrame)
	if want := gpui.Some(placement{container: [4]float64{60, 90, 90, 170}, view: [4]float64{-10, 0, 100, 200}}); clipped != want {
		t.Fatalf("clipped = %+v, want %+v", clipped, want)
	}

	hidden := newPlacement(bounds, scaled(0, 500, 800, 100), 2, gpuiFrame)
	if hidden.IsSome() {
		t.Fatalf("hidden = %+v, want None", hidden)
	}
}

func TestNativeViews_KeyboardInputFollowsGpuiFocusExceptFromViewsGpuiDidNotAttach(t *testing.T) {
	gpuiView := inputHolder{kind: inputHolderGpuiView}
	window := inputHolder{kind: inputHolderWindow}
	other := inputHolder{kind: inputHolderOther}
	shown := nativeViewsTestFocused(1, true)
	hidden := nativeViewsTestFocused(1, false)
	none := gpui.None[focusedNativeView]()
	toView := gpui.Some(inputTarget{kind: inputTargetNativeView, id: nativeViewsTestID(1)})
	toGpui := gpui.Some(inputTarget{kind: inputTargetGpuiView})
	stay := gpui.None[inputTarget]()
	for _, c := range []struct {
		focused gpui.Option[focusedNativeView]
		current inputHolder
		movesTo gpui.Option[inputTarget]
	}{
		{shown, gpuiView, toView},
		{shown, window, toView},
		{shown, nativeViewsTestHolder(inputHolderNativeView, 2), toView},
		{shown, nativeViewsTestHolder(inputHolderNativeView, 1), stay},
		{none, nativeViewsTestHolder(inputHolderNativeView, 1), toGpui},
		{none, window, toGpui},
		{none, gpuiView, stay},
		{hidden, nativeViewsTestHolder(inputHolderNativeView, 1), toGpui},
		{hidden, gpuiView, stay},
		{shown, other, stay},
		{none, other, stay},
	} {
		if got := inputTargetForFocus(c.focused, c.current); got != c.movesTo {
			t.Errorf("focused: %+v, current: %+v: got %+v, want %+v", c.focused, c.current, got, c.movesTo)
		}
	}
}

func TestNativeViews_ViewsTakeKeyboardInputOnlyWhenGpuiFocusAClickOrVoiceOverAllows(t *testing.T) {
	gpuiView := inputHolder{kind: inputHolderGpuiView}
	other := inputHolder{kind: inputHolderOther}
	view, view2 := nativeViewsTestHolder(inputHolderNativeView, 1), nativeViewsTestHolder(inputHolderNativeView, 2)
	shown, hidden := nativeViewsTestFocused(1, true), nativeViewsTestFocused(1, false)
	none := gpui.None[focusedNativeView]()
	noClick := gpui.None[gpui.NativeViewID]()
	click, click2 := gpui.Some(nativeViewsTestID(1)), gpui.Some(nativeViewsTestID(2))

	check := func(want bool, requested, current inputHolder, focused gpui.Option[focusedNativeView], clicked gpui.Option[gpui.NativeViewID], voiceOver bool) {
		t.Helper()
		if got := mayTakeInput(requested, current, focused, clicked, voiceOver); got != want {
			t.Errorf("mayTakeInput(%+v, %+v, %+v, %+v, %v) = %v, want %v", requested, current, focused, clicked, voiceOver, got, want)
		}
	}
	check(false, view, gpuiView, none, noClick, false)
	check(false, view2, gpuiView, shown, noClick, false)
	check(false, view, gpuiView, none, click2, false)
	check(true, view, gpuiView, shown, noClick, false)
	check(true, view, gpuiView, hidden, noClick, false)
	check(true, view, gpuiView, none, click, false)
	check(true, view, gpuiView, none, noClick, true)

	check(false, gpuiView, view, shown, noClick, false)
	check(true, gpuiView, view, none, noClick, false)
	check(true, gpuiView, view, hidden, noClick, false)
	check(true, gpuiView, view2, shown, noClick, false)

	check(true, other, view, shown, noClick, false)
}

func TestNativeViews_AViewThatDeclinesKeyboardInputLeavesItWithTheGpuiView(t *testing.T) {
	declined := gpui.Some(focusedNativeView{id: nativeViewsTestID(1), shown: true, declined: true})
	if got, want := inputTargetForFocus(declined, inputHolder{kind: inputHolderWindow}), gpui.Some(inputTarget{kind: inputTargetGpuiView}); got != want {
		t.Errorf("from window: got %+v, want %+v", got, want)
	}
	if got := inputTargetForFocus(declined, inputHolder{kind: inputHolderGpuiView}); got.IsSome() {
		t.Errorf("from gpui view: got %+v, want None", got)
	}
	if !mayTakeInput(nativeViewsTestHolder(inputHolderNativeView, 1), inputHolder{kind: inputHolderGpuiView}, declined, gpui.Some(nativeViewsTestID(1)), false) {
		t.Error("a declined view may still take keyboard input on a click")
	}
}

func TestNativeViews_GpuiIsOwedTheReleaseOfEachButtonPressedInANativeView(t *testing.T) {
	var views nativeViews
	views.noteButtonPressedInView(0)
	views.noteButtonPressedInView(1)
	if !views.takeButtonPressedInView(0) {
		t.Error("button 0 should be owed")
	}
	if views.takeButtonPressedInView(0) {
		t.Error("button 0 should be settled")
	}
	if !views.takeButtonPressedInView(1) {
		t.Error("button 1 should be owed")
	}
	if views.takeButtonPressedInView(2) {
		t.Error("button 2 was never pressed in a view")
	}
}

func TestNativeViews_StackingPutsShownViewsBetweenTheGpuiViewAndEverythingElse(t *testing.T) {
	gpuiView, front, back, hidden, behind, above := objc.ID(1), objc.ID(2), objc.ID(3), objc.ID(4), objc.ID(5), objc.ID(6)
	backToFront := []objc.ID{front, behind, gpuiView, hidden, above, back}
	s := &stacking{
		gpuiView:    gpuiView,
		frontToBack: []objc.ID{front, back},
		attached:    []objc.ID{hidden, back, front},
		current:     slices.Clone(backToFront),
	}
	slices.SortStableFunc(backToFront, func(first, second objc.ID) int {
		return cmp.Compare(compareStacking(first, second, s), 0)
	})
	if want := []objc.ID{behind, hidden, back, front, gpuiView, above}; !slices.Equal(backToFront, want) {
		t.Fatalf("got %v, want %v", backToFront, want)
	}
}
