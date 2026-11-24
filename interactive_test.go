package gpui

import "testing"

type interactiveTestAction struct{}

func (interactiveTestAction) Name() string         { return "test_only::TestAction" }
func (a interactiveTestAction) BoxedClone() Action { return a }
func (interactiveTestAction) PartialEq(other Action) bool {
	_, ok := other.(interactiveTestAction)
	return ok
}

type interactiveTestView struct {
	sawKeyDown  bool
	sawAction   bool
	focusHandle FocusHandle
}

func (v *interactiveTestView) Render(_ *Window, cx *Context[interactiveTestView]) IntoElement {
	onKeyDown := cx.Listener(func(this *interactiveTestView, _ *KeyDownEvent, _ *Window, cx *Context[interactiveTestView]) {
		cx.StopPropagation()
		this.sawKeyDown = true
	})
	onAction := cx.Listener(func(this *interactiveTestView, _ *interactiveTestAction, _ *Window, _ *Context[interactiveTestView]) {
		this.sawAction = true
	})
	return NewDiv().ID("testview").Child(
		NewDiv().
			KeyContext("parent").
			OnKeyDown(onKeyDown).
			OnAction(func(action interactiveTestAction, window *Window, cx *App) { onAction(&action, window, cx) }).
			Child(NewDiv().KeyContext("nested").TrackFocus(v.focusHandle)),
	)
}

func newInteractiveTestView(cx *Context[interactiveTestView]) *interactiveTestView {
	return &interactiveTestView{focusHandle: cx.FocusHandle()}
}

func TestInteractive_OnEvents(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var window WindowHandle[interactiveTestView]
		var view Entity[interactiveTestView]
		cx.Update(func(cx *App) {
			handle, err := cx.OpenWindow(DefaultWindowOptions(), func(_ *Window, cx *App) Entity[interactiveTestView] {
				view = cx.New(newInteractiveTestView)
				return view.Clone()
			})
			windowTestMust(err)
			window = handle
		})
		defer func() {
			var focusHandle FocusHandle
			cx.Update(func(cx *App) { focusHandle = view.Read(cx).focusHandle })
			view.Release()
			focusHandle.Release()
		}()

		cx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-g", interactiveTestAction{}, Some("parent"))})
		})

		windowTestMust(window.Update(cx, func(testView *interactiveTestView, window *Window, cx *Context[interactiveTestView]) {
			window.Focus(testView.focusHandle, cx.App)
		}))

		cx.DispatchKeystroke(window.AnyWindowHandle, mustParseKeystroke(t, "a"))
		cx.DispatchKeystroke(window.AnyWindowHandle, mustParseKeystroke(t, "ctrl-g"))

		windowTestMust(window.Update(cx, func(testView *interactiveTestView, _ *Window, _ *Context[interactiveTestView]) {
			if !testView.sawKeyDown || !testView.sawAction {
				t.Fatalf("saw key down %v, saw action %v", testView.sawKeyDown, testView.sawAction)
			}
		}))
	})
}

func TestInteractive_MultiModifierGestureDoesNotDispatchStandaloneModifierBinding(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		testView, cx := tcx.AddWindowView(func(_ *Window, cx *Context[interactiveTestView]) *interactiveTestView {
			return newInteractiveTestView(cx)
		})
		defer func() {
			var focusHandle FocusHandle
			cx.Update(func(_ *Window, cx *App) { focusHandle = testView.Read(cx).focusHandle })
			testView.Release()
			focusHandle.Release()
		}()

		cx.Update(func(_ *Window, cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("shift", interactiveTestAction{}, None[string]())})
		})
		windowTestMust(testView.UpdateIn(cx, func(testView *interactiveTestView, window *Window, cx *Context[interactiveTestView]) {
			window.Focus(testView.focusHandle, cx.App)
		}))

		sawAction := func() bool {
			return testView.ReadWith(cx, func(testView *interactiveTestView, _ *App) bool { return testView.sawAction })
		}

		cx.SimulateModifiersChange(ModifiersAlt())
		cx.SimulateModifiersChange(Modifiers{Alt: true, Shift: true})
		cx.SimulateModifiersChange(ModifiersShift())
		cx.SimulateModifiersChange(ModifiersNone())
		if sawAction() {
			t.Fatal("multi-modifier gesture dispatched the standalone binding")
		}

		cx.SimulateModifiersChange(ModifiersShift())
		cx.SimulateModifiersChange(ModifiersNone())
		if !sawAction() {
			t.Fatal("standalone shift did not dispatch")
		}
	})
}
