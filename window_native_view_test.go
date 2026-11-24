package gpui

import (
	"slices"
	"testing"
)

type nativeViewTestsReserved struct{}

func (nativeViewTestsReserved) Name() string         { return "native_view_tests::Reserved" }
func (a nativeViewTestsReserved) BoxedClone() Action { return a }
func (nativeViewTestsReserved) PartialEq(o Action) bool {
	_, ok := o.(nativeViewTestsReserved)
	return ok
}

type nativeViewTestsOrdinary struct{}

func (nativeViewTestsOrdinary) Name() string         { return "native_view_tests::Ordinary" }
func (a nativeViewTestsOrdinary) BoxedClone() Action { return a }
func (nativeViewTestsOrdinary) PartialEq(o Action) bool {
	_, ok := o.(nativeViewTestsOrdinary)
	return ok
}

type nativeViewTestView struct {
	nativeView            NativeView
	otherFocus            FocusHandle
	actions               []string
	keys                  []string
	presses               []string
	paintsNativeViewTwice bool
	unmarkCount           int
}

func (v *nativeViewTestView) Render(_ *Window, cx *Context[nativeViewTestView]) IntoElement {
	reserved := cx.Listener(func(this *nativeViewTestView, _ *nativeViewTestsReserved, _ *Window, _ *Context[nativeViewTestView]) {
		this.actions = append(this.actions, "reserved")
	})
	ordinary := cx.Listener(func(this *nativeViewTestView, _ *nativeViewTestsOrdinary, _ *Window, _ *Context[nativeViewTestView]) {
		this.actions = append(this.actions, "ordinary")
	})
	push := func(label string) func(*MouseDownEvent, *Window, *App) {
		return cx.Listener(func(this *nativeViewTestView, _ *MouseDownEvent, _ *Window, _ *Context[nativeViewTestView]) {
			this.presses = append(this.presses, label)
		})
	}
	view := cx.WeakEntity()
	focus := v.otherFocus
	return NewDiv().
		SizeFull().
		OnAction(func(action nativeViewTestsReserved, window *Window, cx *App) { reserved(&action, window, cx) }).
		OnAction(func(action nativeViewTestsOrdinary, window *Window, cx *App) { ordinary(&action, window, cx) }).
		OnKeyDown(cx.Listener(func(this *nativeViewTestView, event *KeyDownEvent, _ *Window, _ *Context[nativeViewTestView]) {
			this.keys = append(this.keys, event.Keystroke.Key)
		})).
		Child(NewDiv().
			Absolute().
			Top0().
			Left0().
			Size(Px(100.)).
			Bg(Rgb(0xff0000)).
			OnMouseDown(MouseButtonLeft, push("behind view")).
			Child(v.nativeView)).
		When(v.paintsNativeViewTwice, func(this *Div) *Div {
			return this.Child(NewDiv().
				Absolute().
				Top0().
				Left(Px(200.)).
				Size(Px(100.)).
				OnMouseDown(MouseButtonLeft, push("behind second occurrence")).
				Child(v.nativeView))
		}).
		Child(NewDiv().
			TrackFocus(v.otherFocus).
			Absolute().
			Top0().
			Left0().
			Size(Px(50.)).
			Occlude().
			Bg(Rgb(0x0000ff)).
			OnMouseDownOut(push("outside cover")).
			Child(NewCanvas(
				func(Bounds[Pixels], *Window, *App) struct{} { return struct{}{} },
				func(_ Bounds[Pixels], _ struct{}, window *Window, cx *App) {
					window.HandleInput(focus, newNativeViewTestInput(view), cx)
				},
			).SizeFull()).
			ID("cover"))
}

type nativeViewTestInput struct {
	InputHandlerDefaults
	view WeakEntity[nativeViewTestView]
}

func newNativeViewTestInput(view WeakEntity[nativeViewTestView]) *nativeViewTestInput {
	input := &nativeViewTestInput{view: view}
	BindInputHandlerDefaults(&input.InputHandlerDefaults, input)
	return input
}

func (*nativeViewTestInput) SelectedTextRange(bool, *Window, *App) (UTF16Selection, bool) {
	return UTF16Selection{}, false
}

func (*nativeViewTestInput) MarkedTextRange(*Window, *App) (Range, bool) {
	return Range{}, false
}

func (*nativeViewTestInput) TextForRange(Range, *Option[Range], *Window, *App) (string, bool) {
	return "", false
}

func (*nativeViewTestInput) ReplaceTextInRange(Option[Range], string, *Window, *App) {}

func (*nativeViewTestInput) ReplaceAndMarkTextInRange(Option[Range], string, Option[Range], *Window, *App) {
}

func (i *nativeViewTestInput) UnmarkText(_ *Window, cx *App) {
	_ = i.view.Update(cx, func(view *nativeViewTestView, _ *Context[nativeViewTestView]) { view.unmarkCount++ })
}

func (*nativeViewTestInput) BoundsForRange(Range, *Window, *App) (Bounds[Pixels], bool) {
	return Bounds[Pixels]{}, false
}

func (*nativeViewTestInput) CharacterIndexForPoint(Point[Pixels], *Window, *App) (int, bool) {
	return 0, false
}

func nativeViewTestNewNativeView(window *Window, cx *App) NativeView {
	view, err := NewNativeView(1, window, cx)
	if err != nil {
		panic("the test platform attaches native views")
	}
	return view
}

func nativeViewTestOpenWindow(cx *TestAppContext) WindowHandle[nativeViewTestView] {
	return cx.AddWindow(func(window *Window, cx *Context[nativeViewTestView]) *nativeViewTestView {
		return &nativeViewTestView{
			nativeView: nativeViewTestNewNativeView(window, cx.App),
			otherFocus: cx.FocusHandle(),
		}
	})
}

func nativeViewTestClose(cx *TestAppContext, windows ...WindowHandle[nativeViewTestView]) {
	var views []*nativeViewTestView
	for _, window := range windows {
		_ = window.Update(cx, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) {
			views = append(views, view)
		})
	}
	windowTestReleaseAll(cx)
	for _, view := range views {
		view.nativeView.Release()
		view.otherFocus.Release()
	}
}

func nativeViewTestUpdate[R any](t *testing.T, cx AppContext, window WindowHandle[nativeViewTestView], f func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) R) R {
	t.Helper()
	result, err := window.UpdateR(cx, f)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func nativeViewTestPress(t *testing.T, keystroke string) KeyDownEvent {
	t.Helper()
	parsed, err := KeystrokeParse(keystroke)
	if err != nil {
		t.Fatal(err)
	}
	return KeyDownEvent{Keystroke: parsed}
}

func nativeViewTestFocusNativeView(view *nativeViewTestView, window *Window, cx *App) {
	handle := view.nativeView.FocusHandle(cx)
	window.Focus(handle, cx)
	handle.Release()
}

func TestNativeView_FocusingANativeViewGivesItKeyboardInput(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		platformWindow := cx.testWindow(window.AnyWindowHandle)

		id := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) NativeViewID {
			nativeViewTestFocusNativeView(view, window, cx.App)
			return view.nativeView.state().id
		})
		if got, ok := platformWindow.FocusedNativeView(); !ok || got != id {
			t.Fatalf("focused = %v %v", got, ok)
		}

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			window.Focus(view.otherFocus, cx.App)
			return struct{}{}
		})
		if _, ok := platformWindow.FocusedNativeView(); ok {
			t.Fatal("a native view is still focused")
		}

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			nativeViewTestFocusNativeView(view, window, cx.App)
			window.Blur(cx.App)
			return struct{}{}
		})
		if _, ok := platformWindow.FocusedNativeView(); ok {
			t.Fatal("a native view is still focused")
		}
	})
}

func TestNativeView_PlatformFocusOfANativeViewFocusesItsHandle(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		id := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) NativeViewID {
			window.Focus(view.otherFocus, cx.App)
			return view.nativeView.state().id
		})

		platformWindow.SimulateNativeViewFocused(id)

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			handle := view.nativeView.FocusHandle(cx.App)
			defer handle.Release()
			if !handle.IsFocused(window) {
				t.Fatal("the native view's handle is not focused")
			}
			return struct{}{}
		})
		if got, ok := platformWindow.FocusedNativeView(); !ok || got != id {
			t.Fatalf("focused = %v %v", got, ok)
		}
	})
}

func TestNativeView_OnlyReservedAndPendingBindingsApplyBeforeANativeView(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{
				NewKeyBinding("cmd-r", nativeViewTestsReserved{}, None[string]()),
				NewKeyBinding("cmd-o", nativeViewTestsOrdinary{}, None[string]()),
				NewKeyBinding("cmd-k o", nativeViewTestsOrdinary{}, None[string]()),
				NewKeyBinding("cmd-j r", nativeViewTestsReserved{}, None[string]()),
			})
		})
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			view.nativeView.ReserveAction[nativeViewTestsReserved]()
			nativeViewTestFocusNativeView(view, window, cx.App)
			return struct{}{}
		})
		cx.RunUntilParked()
		actions := func() []string {
			return nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) []string {
				taken := view.actions
				view.actions = nil
				return taken
			})
		}
		keys := func() []string {
			return nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) []string {
				taken := view.keys
				view.keys = nil
				return taken
			})
		}
		expect := func(got []string, want ...string) {
			t.Helper()
			if !slices.Equal(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		}

		if !platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "cmd-r")) {
			t.Fatal("cmd-r was not taken")
		}
		expect(actions(), "reserved")

		if platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "cmd-o")) {
			t.Fatal("cmd-o was taken")
		}
		expect(actions())

		if platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "cmd-k")) {
			t.Fatal("cmd-k was taken")
		}
		cx.SimulateKeystrokes(window.AnyWindowHandle, "cmd-k")
		if platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "x")) {
			t.Fatal("x was taken")
		}
		if nativeViewTestUpdate(t, cx, window, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) bool {
			return window.HasPendingKeystrokes()
		}) {
			t.Fatal("keystrokes are still pending")
		}
		expect(keys(), "k")
		cx.SimulateKeystrokes(window.AnyWindowHandle, "x")
		expect(keys(), "x")
		if platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "o")) {
			t.Fatal("o was taken")
		}
		expect(actions())

		if !platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "cmd-j")) {
			t.Fatal("cmd-j was not taken")
		}
		if !platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "r")) {
			t.Fatal("r was not taken")
		}
		expect(actions(), "reserved")

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			window.Focus(view.otherFocus, cx.App)
			return struct{}{}
		})
		if platformWindow.SimulateKeyDownBeforeNativeView(nativeViewTestPress(t, "cmd-r")) {
			t.Fatal("cmd-r was taken ahead of the window's own content")
		}
	})
}

func TestNativeView_MouseReachesANativeViewOnlyWhereNothingCoversIt(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		id := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) NativeViewID {
			return view.nativeView.state().id
		})
		cx.RunUntilParked()

		if got, ok := platformWindow.HitTestNativeView(NewPoint(Px(75.), Px(75.))); !ok || got != id {
			t.Fatalf("uncovered = %v %v", got, ok)
		}
		if _, ok := platformWindow.HitTestNativeView(NewPoint(Px(25.), Px(25.))); ok {
			t.Fatal("covered point reached the view")
		}
		if _, ok := platformWindow.HitTestNativeView(NewPoint(Px(150.), Px(150.))); ok {
			t.Fatal("outside point reached the view")
		}
	})
}

func TestNativeView_MousePressOnANativeViewReachesNothingBehindIt(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := nativeViewTestOpenWindow(tcx)
		defer nativeViewTestClose(tcx, window)
		tcx.RunUntilParked()
		cx := VisualTestContextFromWindow(window.AnyWindowHandle, tcx)

		cx.SimulateMouseDown(NewPoint(Px(75.), Px(75.)), MouseButtonLeft, Modifiers{})

		presses := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) []string {
			taken := view.presses
			view.presses = nil
			return taken
		})
		if !slices.Equal(presses, []string{"outside cover"}) {
			t.Fatalf("presses = %q", presses)
		}
	})
}

func TestNativeView_ChangingTheBackgroundRepaintsTheNativeView(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		cx.RunUntilParked()

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, cx *Context[nativeViewTestView]) struct{} {
			view.nativeView.SetBackground(Rgb(0x00ff00), cx.App)
			return struct{}{}
		})
		cx.RunUntilParked()

		nativeViewTestUpdate(t, cx, window, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) struct{} {
			var backgrounds []Hsla
			for _, nativeView := range window.renderedFrame.scene.NativeViews {
				backgrounds = append(backgrounds, nativeView.Background)
			}
			if !slices.Equal(backgrounds, []Hsla{Rgb(0x00ff00).ToHsla()}) {
				t.Fatalf("backgrounds = %v", backgrounds)
			}
			return struct{}{}
		})
	})
}

func TestNativeView_NativeViewPaintedTwiceShowsWhereItWasPaintedFirst(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := nativeViewTestOpenWindow(tcx)
		defer nativeViewTestClose(tcx, window)
		nativeViewTestUpdate(t, tcx, window, func(view *nativeViewTestView, _ *Window, cx *Context[nativeViewTestView]) struct{} {
			view.paintsNativeViewTwice = true
			cx.Notify()
			return struct{}{}
		})
		tcx.RunUntilParked()

		nativeViewTestUpdate(t, tcx, window, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) struct{} {
			var xs []ScaledPixels
			for _, nativeView := range window.renderedFrame.scene.NativeViews {
				xs = append(xs, nativeView.Bounds.Origin.X)
			}
			if !slices.Equal(xs, []ScaledPixels{0}) {
				t.Fatalf("origins = %v", xs)
			}
			return struct{}{}
		})

		id := nativeViewTestUpdate(t, tcx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) NativeViewID {
			return view.nativeView.state().id
		})
		platformWindow := tcx.testWindow(window.AnyWindowHandle)
		paintedFirst := NewPoint(Px(75.), Px(75.))
		paintedAgain := NewPoint(Px(250.), Px(50.))
		if got, ok := platformWindow.HitTestNativeView(paintedFirst); !ok || got != id {
			t.Fatalf("painted first = %v %v", got, ok)
		}
		if _, ok := platformWindow.HitTestNativeView(paintedAgain); ok {
			t.Fatal("the second occurrence reached the view")
		}
		cx := VisualTestContextFromWindow(window.AnyWindowHandle, tcx)
		cx.SimulateMouseDown(paintedAgain, MouseButtonLeft, Modifiers{})
		presses := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) []string {
			taken := view.presses
			view.presses = nil
			return taken
		})
		if !slices.Equal(presses, []string{"outside cover", "behind second occurrence"}) {
			t.Fatalf("presses = %q", presses)
		}
	})
}

func TestNativeView_NativeViewIsDrawnInPaintOrder(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		cx.RunUntilParked()

		nativeViewTestUpdate(t, cx, window, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) struct{} {
			var labels []string
			for batch := range window.renderedFrame.scene.Batches() {
				labels = append(labels, batch.Label())
			}
			if !slices.Equal(labels, []string{"quads (1)", "native views (1)", "quads (1)"}) {
				t.Fatalf("labels = %q", labels)
			}
			return struct{}{}
		})
	})
}

func TestNativeView_NativeViewStaysAttachedUntilItsLastHandleIsDropped(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		platformWindow := cx.testWindow(window.AnyWindowHandle)
		otherHandle := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) NativeView {
			return view.nativeView.Clone()
		})
		if got := platformWindow.AttachedNativeViewCount(); got != 1 {
			t.Fatalf("attached = %d", got)
		}

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			old := view.nativeView
			view.nativeView = nativeViewTestNewNativeView(window, cx.App)
			old.Release()
			return struct{}{}
		})
		if got := platformWindow.AttachedNativeViewCount(); got != 2 {
			t.Fatalf("attached = %d", got)
		}

		otherHandle.Release()
		if got := platformWindow.AttachedNativeViewCount(); got != 1 {
			t.Fatalf("attached = %d", got)
		}
	})
}

func TestNativeView_NativeViewShowsNothingInAnotherWindow(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		nativeView := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) NativeView {
			return view.nativeView.Clone()
		})
		otherWindow := cx.AddWindow(func(_ *Window, cx *Context[nativeViewTestView]) *nativeViewTestView {
			return &nativeViewTestView{nativeView: nativeView, otherFocus: cx.FocusHandle()}
		})
		defer nativeViewTestClose(cx, window, otherWindow)
		cx.RunUntilParked()

		nativeViewTestUpdate(t, cx, otherWindow, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) struct{} {
			if len(window.renderedFrame.scene.NativeViews) != 0 {
				t.Fatalf("other window shows %d native views", len(window.renderedFrame.scene.NativeViews))
			}
			return struct{}{}
		})
		nativeViewTestUpdate(t, cx, window, func(_ *nativeViewTestView, window *Window, _ *Context[nativeViewTestView]) struct{} {
			if len(window.renderedFrame.scene.NativeViews) != 1 {
				t.Fatalf("window shows %d native views", len(window.renderedFrame.scene.NativeViews))
			}
			return struct{}{}
		})
	})
}

func TestNativeView_FocusingANativeViewEndsAComposition(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := nativeViewTestOpenWindow(cx)
		defer nativeViewTestClose(cx, window)
		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			window.Focus(view.otherFocus, cx.App)
			return struct{}{}
		})
		cx.RunUntilParked()

		nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, window *Window, cx *Context[nativeViewTestView]) struct{} {
			nativeViewTestFocusNativeView(view, window, cx.App)
			return struct{}{}
		})
		unmarkCount := nativeViewTestUpdate(t, cx, window, func(view *nativeViewTestView, _ *Window, _ *Context[nativeViewTestView]) int {
			return view.unmarkCount
		})
		if unmarkCount != 1 {
			t.Fatalf("unmark count = %d", unmarkCount)
		}
	})
}
