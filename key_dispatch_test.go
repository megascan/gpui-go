package gpui

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

type dispatchTestTestAction struct{}
type dispatchTestSecondaryTestAction struct{}

func (dispatchTestTestAction) Name() string         { return "dispatch_test::TestAction" }
func (a dispatchTestTestAction) BoxedClone() Action { return a }
func (dispatchTestTestAction) PartialEq(other Action) bool {
	_, ok := other.(dispatchTestTestAction)
	return ok
}
func (dispatchTestSecondaryTestAction) Name() string         { return "dispatch_test::SecondaryTestAction" }
func (a dispatchTestSecondaryTestAction) BoxedClone() Action { return a }
func (dispatchTestSecondaryTestAction) PartialEq(other Action) bool {
	_, ok := other.(dispatchTestSecondaryTestAction)
	return ok
}

func testDispatchTree(bindings []KeyBinding) *dispatchTree {
	return newDispatchTree(NewKeymap(bindings), newActionRegistry())
}

func TestKeyDispatch_KeybindingForActionBounds(t *testing.T) {
	tree := testDispatchTree([]KeyBinding{kbIn("cmd-n", dispatchTestTestAction{}, "ProjectPanel")})

	keybinding := tree.bindingsForAction(dispatchTestTestAction{}, contexts(t, "Workspace", "ProjectPanel"))

	if !keybinding[0].action.PartialEq(dispatchTestTestAction{}) {
		t.Fatalf("keybinding = %v", keybinding)
	}
}

func targetedUnbindTree() *dispatchTree {
	return testDispatchTree([]KeyBinding{
		kbIn("tab", dispatchTestTestAction{}, "Editor"),
		kbIn("tab", Unbind("dispatch_test::TestAction"), "Editor && edit_prediction"),
		kbIn("tab", dispatchTestSecondaryTestAction{}, "Editor && showing_completions"),
	})
}

func TestKeyDispatch_BindingsForActionHidesTargetedUnbindInActiveContext(t *testing.T) {
	tree := targetedUnbindTree()
	ctx := contexts(t, "Workspace", "Editor showing_completions edit_prediction")

	if bindings := tree.bindingsForAction(dispatchTestTestAction{}, ctx); len(bindings) != 0 {
		t.Fatalf("bindings = %v", bindings)
	}

	if _, ok := tree.highestPrecedenceBindingForAction(dispatchTestTestAction{}, ctx); ok {
		t.Fatal("highest is some")
	}

	fallbackBindings := tree.bindingsForAction(dispatchTestSecondaryTestAction{}, ctx)
	if len(fallbackBindings) != 1 || !fallbackBindings[0].action.PartialEq(dispatchTestSecondaryTestAction{}) {
		t.Fatalf("fallback = %v", fallbackBindings)
	}
}

func TestKeyDispatch_BindingsForActionKeepsTargetedBindingOutsideUnbindContext(t *testing.T) {
	tree := targetedUnbindTree()
	ctx := contexts(t, "Workspace", "Editor")

	bindings := tree.bindingsForAction(dispatchTestTestAction{}, ctx)
	if len(bindings) != 1 || !bindings[0].action.PartialEq(dispatchTestTestAction{}) {
		t.Fatalf("bindings = %v", bindings)
	}

	highest, ok := tree.highestPrecedenceBindingForAction(dispatchTestTestAction{}, ctx)
	if !ok || !highest.action.PartialEq(dispatchTestTestAction{}) {
		t.Fatalf("highest = %v, %v", highest, ok)
	}
}

func TestKeyDispatch_OverlayAfterBaseRestoresShadowedPickerBinding(t *testing.T) {
	ctx := contexts(t, "Picker", "Editor")

	shadowed := testDispatchTree([]KeyBinding{
		kbIn("ctrl-alt-down", dispatchTestTestAction{}, "Picker > Editor"),
		kbIn("ctrl-alt-down", dispatchTestSecondaryTestAction{}, "Editor"),
	})
	if _, ok := shadowed.highestPrecedenceBindingForAction(dispatchTestTestAction{}, ctx); ok {
		t.Fatal("picker binding should be shadowed by the base editor binding")
	}

	fixed := testDispatchTree([]KeyBinding{
		kbIn("ctrl-alt-down", dispatchTestTestAction{}, "Picker > Editor"),
		kbIn("ctrl-alt-down", dispatchTestSecondaryTestAction{}, "Editor"),
		kbIn("ctrl-alt-down", dispatchTestTestAction{}, "Picker > Editor"),
	})
	if highest, ok := fixed.highestPrecedenceBindingForAction(dispatchTestTestAction{}, ctx); !ok || !highest.action.PartialEq(dispatchTestTestAction{}) {
		t.Fatal("overlay loaded after base should restore the picker binding")
	}

	overrideBeforeBase := testDispatchTree([]KeyBinding{
		kbIn("ctrl-alt-down", dispatchTestTestAction{}, "Picker > Editor"),
		kbIn("ctrl-alt-down", dispatchTestTestAction{}, "Picker > Editor"),
		kbIn("ctrl-alt-down", dispatchTestSecondaryTestAction{}, "Editor"),
	})
	if _, ok := overrideBeforeBase.highestPrecedenceBindingForAction(dispatchTestTestAction{}, ctx); ok {
		t.Fatal("an override loaded before the base binding cannot win the equal-depth tie")
	}
}

func TestKeyDispatch_PendingHasBindingState(t *testing.T) {
	tree := testDispatchTree([]KeyBinding{
		kb("ctrl-b h", dispatchTestTestAction{}),
		kbIn("space", dispatchTestTestAction{}, "ContextA"),
		kbIn("space f g", dispatchTestTestAction{}, "ContextB"),
	})

	dispatch := func(pending []Keystroke, key string, path []dispatchNodeID) dispatchResult {
		return tree.dispatchKey(pending, mustParseKeystroke(t, key), path)
	}

	var dispatchPath []dispatchNodeID
	result := dispatch(nil, "ctrl-b", dispatchPath)
	if len(result.pending) != 1 || result.pendingHasBinding {
		t.Fatalf("ctrl-b: %+v", result)
	}

	result = dispatch(result.pending, "h", dispatchPath)
	if len(result.pending) != 0 || len(result.bindings) != 1 || result.pendingHasBinding {
		t.Fatalf("h: %+v", result)
	}

	nodeID := tree.pushNode()
	tree.setKeyContext(mustParseContext(t, "ContextB"))
	tree.popNode()

	dispatchPath = tree.dispatchPath(nodeID)
	result = dispatch(nil, "space", dispatchPath)
	if len(result.pending) != 1 || result.pendingHasBinding {
		t.Fatalf("space: %+v", result)
	}
}

type dispatchTestPendingTextInputView struct {
	InputHandlerDefaults
	focusHandle      FocusHandle
	text             *string
	actionCount      *int
	acceptsTextInput *bool
	custom           bool
}

func newDispatchTestPendingTextInputView(cx *Context[dispatchTestPendingTextInputView], custom bool) *dispatchTestPendingTextInputView {
	accepts := true
	return &dispatchTestPendingTextInputView{
		focusHandle:      cx.FocusHandle(),
		text:             new(string),
		actionCount:      new(int),
		acceptsTextInput: &accepts,
		custom:           custom,
	}
}

func (v *dispatchTestPendingTextInputView) clone() *dispatchTestPendingTextInputView {
	c := &dispatchTestPendingTextInputView{
		focusHandle:      v.focusHandle,
		text:             v.text,
		actionCount:      v.actionCount,
		acceptsTextInput: v.acceptsTextInput,
		custom:           v.custom,
	}
	BindInputHandlerDefaults(&c.InputHandlerDefaults, c)
	return c
}

func (v *dispatchTestPendingTextInputView) Render(_ *Window, _ *Context[dispatchTestPendingTextInputView]) IntoElement {
	return v.clone()
}

func (v *dispatchTestPendingTextInputView) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](v)
}

func (v *dispatchTestPendingTextInputView) ElementID() (ElementID, bool) {
	if v.custom {
		return ElementIDName("custom"), true
	}
	return ElementIDName("pending-text-input-test"), true
}

func (v *dispatchTestPendingTextInputView) SourceLocation() *SourceLocation { return nil }

func (v *dispatchTestPendingTextInputView) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	return window.RequestLayout(DefaultStyle(), nil, cx), struct{}{}
}

func (v *dispatchTestPendingTextInputView) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, window *Window, cx *App) struct{} {
	window.SetFocusHandle(v.focusHandle, cx)
	return struct{}{}
}

func (v *dispatchTestPendingTextInputView) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	var keyContext KeyContext
	keyContext.Add("Terminal")
	window.SetKeyContext(keyContext)
	window.HandleInput(v.focusHandle, v.clone(), cx)
	actionCount := v.actionCount
	custom := v.custom
	window.OnAction(reflect.TypeFor[dispatchTestTestAction](), func(_ any, phase DispatchPhase, _ *Window, _ *App) {
		if !custom && phase == DispatchPhaseBubble {
			*actionCount++
		}
	})
}

func (v *dispatchTestPendingTextInputView) AcceptsTextInput(window *Window, cx *App) bool {
	if v.custom {
		return v.InputHandlerDefaults.AcceptsTextInput(window, cx)
	}
	return *v.acceptsTextInput
}

func (v *dispatchTestPendingTextInputView) SelectedTextRange(bool, *Window, *App) (UTF16Selection, bool) {
	return UTF16Selection{}, false
}

func (v *dispatchTestPendingTextInputView) MarkedTextRange(*Window, *App) (Range, bool) {
	return Range{}, false
}

func (v *dispatchTestPendingTextInputView) TextForRange(Range, *Option[Range], *Window, *App) (string, bool) {
	return "", false
}

func (v *dispatchTestPendingTextInputView) ReplaceTextInRange(replacementRange Option[Range], text string, _ *Window, _ *App) {
	if replacementRange.IsSome() {
		panic("not implemented")
	}
	*v.text += text
}

func (v *dispatchTestPendingTextInputView) ReplaceAndMarkTextInRange(replacementRange Option[Range], newText string, _ Option[Range], _ *Window, _ *App) {
	if replacementRange.IsSome() {
		panic("not implemented")
	}
	*v.text += newText
}

func (v *dispatchTestPendingTextInputView) UnmarkText(*Window, *App) {}

func (v *dispatchTestPendingTextInputView) PrefersIMEForPrintableKeys(window *Window, cx *App) bool {
	if v.custom {
		return v.InputHandlerDefaults.PrefersIMEForPrintableKeys(window, cx)
	}
	return true
}

func (v *dispatchTestPendingTextInputView) BoundsForRange(Range, *Window, *App) (Bounds[Pixels], bool) {
	return Bounds[Pixels]{}, false
}

func (v *dispatchTestPendingTextInputView) CharacterIndexForPoint(Point[Pixels], *Window, *App) (int, bool) {
	return 0, false
}

type dispatchTestPendingInputTimeoutPauseOwner struct{}

type dispatchTestPendingInputView struct {
	focusHandle          FocusHandle
	actionCount          *int
	secondaryActionCount *int
}

func (v *dispatchTestPendingInputView) Render(_ *Window, _ *Context[dispatchTestPendingInputView]) IntoElement {
	actionCount := v.actionCount
	secondaryActionCount := v.secondaryActionCount
	return NewDiv().KeyContext("Terminal").TrackFocus(v.focusHandle).SizeFull().
		OnAction(func(dispatchTestTestAction, *Window, *App) { *actionCount++ }).
		OnAction(func(dispatchTestSecondaryTestAction, *Window, *App) { *secondaryActionCount++ })
}

func dispatchTestReleaseView[V any](cx *VisualTestContext, view Entity[V], focusHandle func(*V) FocusHandle) func() {
	return func() {
		var handle FocusHandle
		cx.Update(func(_ *Window, cx *App) { handle = focusHandle(view.Read(cx)) })
		view.Release()
		windowTestReleaseAll(cx.TestAppContext)
		handle.Release()
	}
}

func setupPendingInputTest(cx *TestAppContext, bindings []KeyBinding) (*VisualTestContext, *int, *int, func()) {
	cx.Update(func(cx *App) { cx.BindKeys(bindings) })

	actionCount := new(int)
	secondaryActionCount := new(int)
	view, vcx := cx.AddWindowView(func(_ *Window, cx *Context[dispatchTestPendingInputView]) *dispatchTestPendingInputView {
		return &dispatchTestPendingInputView{
			focusHandle:          cx.FocusHandle(),
			actionCount:          actionCount,
			secondaryActionCount: secondaryActionCount,
		}
	})
	vcx.Update(func(window *Window, cx *App) {
		window.Focus(view.Read(cx).focusHandle, cx)
		window.ActivateWindow()
	})

	return vcx, actionCount, secondaryActionCount, dispatchTestReleaseView(vcx, view, func(v *dispatchTestPendingInputView) FocusHandle { return v.focusHandle })
}

type dispatchTestObservedKeystroke struct {
	keystroke    Keystroke
	actionName   Option[string]
	contextStack []KeyContext
}

func dispatchTestObservedEqual(got []dispatchTestObservedKeystroke, want []dispatchTestObservedKeystroke) bool {
	return slices.EqualFunc(got, want, func(a, b dispatchTestObservedKeystroke) bool {
		return a.keystroke == b.keystroke && a.actionName == b.actionName && dispatchTestContextsEqual(a.contextStack, b.contextStack)
	})
}

func dispatchTestContextsEqual(a, b []KeyContext) bool {
	return slices.EqualFunc(a, b, func(x, y KeyContext) bool { return x.equal(y) })
}

func captureObservedKeystrokes(cx *VisualTestContext) (*[]dispatchTestObservedKeystroke, Subscription) {
	observed := &[]dispatchTestObservedKeystroke{}
	var subscription Subscription
	cx.Update(func(_ *Window, cx *App) {
		subscription = cx.ObserveKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
			actionName := None[string]()
			if action, ok := event.Action.Get(); ok {
				actionName = Some(action.Name())
			}
			*observed = append(*observed, dispatchTestObservedKeystroke{
				keystroke:    event.Keystroke,
				actionName:   actionName,
				contextStack: slices.Clone(event.ContextStack),
			})
		})
	})
	return observed, subscription
}

func simulateModifierTap(cx *VisualTestContext, modifiers Modifiers) {
	cx.SimulateModifiersChange(modifiers)
	cx.SimulateModifiersChange(ModifiersNone())
}

func terminalContext(t *testing.T) KeyContext {
	t.Helper()
	return mustParseContext(t, "Terminal")
}

func setupPendingInputTimeoutTest(cx *TestAppContext) (*VisualTestContext, *int, *int, func()) {
	return setupPendingInputTest(cx, []KeyBinding{
		NewKeyBinding("ctrl-b", dispatchTestTestAction{}, Some("Terminal")),
		NewKeyBinding("ctrl-b h", dispatchTestSecondaryTestAction{}, Some("Terminal")),
		NewKeyBinding("ctrl-b h j", dispatchTestTestAction{}, Some("Terminal")),
	})
}

func queryPrefersIMEForPrintableKeys(cx *VisualTestContext) Option[bool] {
	platformWindow := cx.testWindow(cx.WindowHandle())
	inputHandler := platformWindow.TakeInputHandler()
	if inputHandler == nil {
		return None[bool]()
	}
	prefersIME := inputHandler.QueryPrefersIMEForPrintableKeys()
	platformWindow.SetInputHandler(inputHandler)
	return Some(prefersIME)
}

func addPendingTextInputView(cx *TestAppContext, custom bool) (Entity[dispatchTestPendingTextInputView], *VisualTestContext, func()) {
	test, vcx := cx.AddWindowView(func(_ *Window, cx *Context[dispatchTestPendingTextInputView]) *dispatchTestPendingTextInputView {
		return newDispatchTestPendingTextInputView(cx, custom)
	})
	return test, vcx, dispatchTestReleaseView(vcx, test, func(v *dispatchTestPendingTextInputView) FocusHandle { return v.focusHandle })
}

func focusPendingTextInputView(cx *VisualTestContext, test Entity[dispatchTestPendingTextInputView]) {
	cx.Update(func(window *Window, cx *App) {
		window.Focus(test.Read(cx).focusHandle, cx)
		window.ActivateWindow()
	})
}

func setupAltgrTest(cx *TestAppContext) (Entity[dispatchTestPendingTextInputView], *VisualTestContext, func()) {
	test, vcx, cleanup := addPendingTextInputView(cx, false)
	windowTestMust(test.UpdateIn(vcx, func(test *dispatchTestPendingTextInputView, window *Window, cx *Context[dispatchTestPendingTextInputView]) {
		window.Focus(test.focusHandle, cx.App)
		window.ActivateWindow()
		cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-alt-a", dispatchTestTestAction{}, Some("Terminal"))})
	}))
	return test, vcx, cleanup
}

func dispatchAltgrKeyDown(t *testing.T, cx *VisualTestContext, preferCharacterInput bool) DispatchEventResult {
	keystroke := mustParseKeystroke(t, "ctrl-alt-a")
	keystroke.KeyChar = Some("ą")
	var result DispatchEventResult
	cx.Update(func(window *Window, cx *App) {
		result = window.DispatchEvent(KeyDownEvent{
			Keystroke:            keystroke,
			IsHeld:               false,
			PreferCharacterInput: preferCharacterInput,
		}, cx)
	})
	return result
}

func simulatePendingBinding(cx *VisualTestContext) {
	cx.SimulateModifiersChange(ModifiersControl())
	cx.SimulateKeystrokes("ctrl-b")
	cx.SimulateModifiersChange(Modifiers{})
}

func dispatchTestNewPauseOwner(cx *VisualTestContext) Entity[dispatchTestPendingInputTimeoutPauseOwner] {
	var owner Entity[dispatchTestPendingInputTimeoutPauseOwner]
	cx.Update(func(_ *Window, cx *App) {
		owner = cx.New(func(*Context[dispatchTestPendingInputTimeoutPauseOwner]) *dispatchTestPendingInputTimeoutPauseOwner {
			return &dispatchTestPendingInputTimeoutPauseOwner{}
		})
	})
	return owner
}

func dispatchTestSetPaused(t *testing.T, cx *VisualTestContext, owner Entity[dispatchTestPendingInputTimeoutPauseOwner], paused bool, want bool) {
	t.Helper()
	cx.Update(func(window *Window, cx *App) {
		if got := window.SetPendingInputTimeoutPaused(owner, paused, cx); got != want {
			t.Fatalf("SetPendingInputTimeoutPaused(%v) = %v, want %v", paused, got, want)
		}
	})
}

func dispatchTestPendingTimeout(t *testing.T, window *Window) (PendingInputStatus, PendingInputTimeoutStatus) {
	t.Helper()
	pendingInput, ok := window.PendingInput()
	if !ok {
		t.Fatal("pending input")
	}
	timeout, ok := pendingInput.Timeout()
	if !ok {
		t.Fatal("pending input timeout")
	}
	return pendingInput, timeout
}

func dispatchTestAssertPending(t *testing.T, cx *VisualTestContext, pending bool) {
	t.Helper()
	cx.Update(func(window *Window, _ *App) {
		if window.HasPendingKeystrokes() != pending {
			t.Fatalf("HasPendingKeystrokes = %v, want %v", !pending, pending)
		}
	})
}

func dispatchTestAssertCounts(t *testing.T, actionCount, secondaryActionCount *int, action, secondary int) {
	t.Helper()
	if *actionCount != action || *secondaryActionCount != secondary {
		t.Fatalf("counts = %d, %d; want %d, %d", *actionCount, *secondaryActionCount, action, secondary)
	}
}

func dispatchTestAdvance(cx *VisualTestContext, d time.Duration) {
	cx.Executor().AdvanceClock(d)
	cx.RunUntilParked()
}

type dispatchTestPendingInputObserver struct {
	subscription Subscription
}

func (o *dispatchTestPendingInputObserver) Drop(*App) { o.subscription.Unsubscribe() }

func TestKeyDispatch_PendingInputObserversNotifiedOnFocusChangeAndBlur(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-b", dispatchTestTestAction{}, Some("Terminal"))})
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-b h", dispatchTestTestAction{}, Some("Terminal"))})
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-d", dispatchTestTestAction{}, None[string]())})
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-d h", dispatchTestTestAction{}, None[string]())})
		})

		test, vcx, cleanup := addPendingTextInputView(cx, true)
		defer cleanup()
		var focusHandle FocusHandle
		vcx.Update(func(_ *Window, cx *App) { focusHandle = test.Read(cx).focusHandle })

		pendingInputChangedCount := 0
		var observer Entity[dispatchTestPendingInputObserver]
		vcx.Update(func(window *Window, cx *App) {
			observer = cx.New(func(cx *Context[dispatchTestPendingInputObserver]) *dispatchTestPendingInputObserver {
				return &dispatchTestPendingInputObserver{
					subscription: cx.ObservePendingInput(window, func(*dispatchTestPendingInputObserver, *Window, *Context[dispatchTestPendingInputObserver]) {
						pendingInputChangedCount++
					}),
				}
			})
		})
		defer observer.Release()

		vcx.Update(func(window *Window, cx *App) {
			window.Focus(focusHandle, cx)
			window.ActivateWindow()
		})

		vcx.SimulateKeystrokes("ctrl-b")

		countAfterPending := 0
		vcx.Update(func(window *Window, cx *App) {
			if !window.HasPendingKeystrokes() {
				t.Fatal("expected pending keystrokes")
			}
			countAfterPending = pendingInputChangedCount
			if countAfterPending <= 0 {
				t.Fatal("observer not notified")
			}

			other := cx.FocusHandle()
			window.Focus(other, cx)
			other.Release()

			if window.HasPendingKeystrokes() {
				t.Fatal("focus change kept pending keystrokes")
			}
		})

		vcx.Update(func(*Window, *App) {
			if pendingInputChangedCount <= countAfterPending {
				t.Fatalf("count after focus change = %d, want > %d", pendingInputChangedCount, countAfterPending)
			}
		})
		if got, want := vcx.testWindow(vcx.WindowHandle()).TextInputStateChanges(), []TextInputStateChange{
			TextInputStateChangeFocusGained,
			TextInputStateChangeFocusLost,
		}; !slices.Equal(got, want) {
			t.Fatalf("text input state changes = %v, want %v", got, want)
		}

		vcx.Update(func(window *Window, cx *App) { window.Focus(focusHandle, cx) })
		vcx.SimulateKeystrokes("ctrl-b")
		countBeforeBlur := pendingInputChangedCount

		vcx.Update(func(window *Window, cx *App) {
			if !window.HasPendingKeystrokes() {
				t.Fatal("expected pending keystrokes")
			}
			window.Blur(cx)
			if window.HasPendingKeystrokes() || !window.pendingInputIsNone() {
				t.Fatal("blur kept pending input")
			}
		})

		vcx.Update(func(*Window, *App) {
			if pendingInputChangedCount <= countBeforeBlur {
				t.Fatal("observer not notified on blur")
			}
		})

		var temporaryFocus FocusHandle
		vcx.Update(func(window *Window, cx *App) {
			temporaryFocus = cx.FocusHandle()
			window.Focus(temporaryFocus, cx)
		})
		vcx.SimulateKeystrokes("ctrl-d")
		dispatchTestAssertPending(t, vcx, true)
		countBeforeDrop := pendingInputChangedCount
		vcx.Update(func(*Window, *App) { temporaryFocus.Release() })
		vcx.Update(func(window *Window, _ *App) {
			if !window.pendingInputIsNone() {
				t.Fatal("dropping the focused handle kept pending input")
			}
		})
		if pendingInputChangedCount != countBeforeDrop+1 {
			t.Fatalf("count = %d, want %d", pendingInputChangedCount, countBeforeDrop+1)
		}

		vcx.Update(func(window *Window, cx *App) { window.DisableFocus(cx) })
		vcx.SimulateKeystrokes("ctrl-d")

		vcx.Update(func(window *Window, cx *App) {
			if !window.HasPendingKeystrokes() {
				t.Fatal("expected pending keystrokes")
			}
			window.Blur(cx)
			if !window.pendingInputIsNone() {
				t.Fatal("blur kept pending input")
			}
		})
	})
}

func TestKeyDispatch_StandaloneModifierDispatch(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, _, cleanup := setupPendingInputTest(tc.Cx(), []KeyBinding{NewKeyBinding("shift", dispatchTestTestAction{}, Some("Terminal"))})
		defer cleanup()
		var interceptedKeystrokes []Keystroke
		shouldConsume := true
		terminal := terminalContext(t)
		observedKeystrokes, observer := captureObservedKeystrokes(cx)
		defer observer.Unsubscribe()
		var interceptor Subscription
		cx.Update(func(_ *Window, cx *App) {
			interceptor = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, cx *App) {
				if event.Action.IsSome() {
					t.Error("interceptor saw an action")
				}
				if !dispatchTestContextsEqual(event.ContextStack, []KeyContext{terminal}) {
					t.Errorf("context stack = %v", event.ContextStack)
				}
				interceptedKeystrokes = append(interceptedKeystrokes, event.Keystroke)
				if shouldConsume {
					cx.StopPropagation()
				}
			})
		})
		defer interceptor.Unsubscribe()
		shift := mustParseKeystroke(t, "shift")

		cx.SimulateModifiersChange(ModifiersShift())
		if len(interceptedKeystrokes) != 0 {
			t.Fatalf("intercepted on press: %v", interceptedKeystrokes)
		}
		cx.SimulateModifiersChange(ModifiersNone())
		if !slices.Equal(interceptedKeystrokes, []Keystroke{shift}) {
			t.Fatalf("intercepted = %v", interceptedKeystrokes)
		}
		if *actionCount != 0 || len(*observedKeystrokes) != 0 {
			t.Fatalf("count %d observed %v", *actionCount, *observedKeystrokes)
		}

		shouldConsume = false
		interceptedKeystrokes = nil
		cx.SimulateModifiersChange(ModifiersShift())
		if len(interceptedKeystrokes) != 0 {
			t.Fatalf("intercepted on press: %v", interceptedKeystrokes)
		}
		cx.SimulateModifiersChange(ModifiersNone())
		if !slices.Equal(interceptedKeystrokes, []Keystroke{shift}) {
			t.Fatalf("intercepted = %v", interceptedKeystrokes)
		}
		if *actionCount != 1 {
			t.Fatalf("action count = %d", *actionCount)
		}
		if want := []dispatchTestObservedKeystroke{{shift, Some(dispatchTestTestAction{}.Name()), []KeyContext{terminal}}}; !dispatchTestObservedEqual(*observedKeystrokes, want) {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}
	})
}

func TestKeyDispatch_UnboundStandaloneModifierObservation(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, _, _, cleanup := setupPendingInputTest(tc.Cx(), nil)
		defer cleanup()
		terminal := terminalContext(t)
		observedKeystrokes, observer := captureObservedKeystrokes(cx)
		defer observer.Unsubscribe()

		for _, c := range []struct {
			modifiers Modifiers
			key       string
		}{
			{ModifiersShift(), "shift"},
			{ModifiersControl(), "ctrl"},
			{ModifiersAlt(), "alt"},
			{ModifiersCommand(), "cmd"},
			{ModifiersFunction(), "fn"},
		} {
			*observedKeystrokes = nil
			cx.SimulateModifiersChange(c.modifiers)
			if len(*observedKeystrokes) != 0 {
				t.Fatal("modifier press must not notify observers")
			}
			cx.SimulateModifiersChange(ModifiersNone())
			want := []dispatchTestObservedKeystroke{{mustParseKeystroke(t, c.key), None[string](), []KeyContext{terminal}}}
			if !dispatchTestObservedEqual(*observedKeystrokes, want) {
				t.Fatalf("%s: observed = %v", c.key, *observedKeystrokes)
			}
		}
	})
}

type dispatchTestModifierListenerView struct {
	focusHandle FocusHandle
	consume     *bool
	releases    *int
}

func (v *dispatchTestModifierListenerView) Render(_ *Window, _ *Context[dispatchTestModifierListenerView]) IntoElement {
	consume := v.consume
	releases := v.releases
	return NewDiv().KeyContext("Terminal").TrackFocus(v.focusHandle).SizeFull().
		OnAction(func(_ dispatchTestTestAction, _ *Window, cx *App) { cx.Propagate() }).
		OnModifiersChanged(func(event *ModifiersChangedEvent, _ *Window, cx *App) {
			if !event.Modifiers.Modified() {
				*releases++
				if *consume {
					cx.StopPropagation()
				}
			}
		})
}

func TestKeyDispatch_RawModifierHandlerCanSuppressObservation(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		tcx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("shift", dispatchTestTestAction{}, Some("Terminal"))})
		})
		consume := true
		releases := 0
		view, cx := tcx.AddWindowView(func(_ *Window, cx *Context[dispatchTestModifierListenerView]) *dispatchTestModifierListenerView {
			return &dispatchTestModifierListenerView{focusHandle: cx.FocusHandle(), consume: &consume, releases: &releases}
		})
		defer dispatchTestReleaseView(cx, view, func(v *dispatchTestModifierListenerView) FocusHandle { return v.focusHandle })()
		cx.Update(func(window *Window, cx *App) {
			window.Focus(view.Read(cx).focusHandle, cx)
			window.ActivateWindow()
		})
		observedKeystrokes, observer := captureObservedKeystrokes(cx)
		defer observer.Unsubscribe()

		simulateModifierTap(cx, ModifiersShift())
		if releases != 1 || len(*observedKeystrokes) != 0 {
			t.Fatalf("releases %d observed %v", releases, *observedKeystrokes)
		}

		consume = false
		simulateModifierTap(cx, ModifiersShift())
		if releases != 2 {
			t.Fatalf("releases = %d", releases)
		}
		want := []dispatchTestObservedKeystroke{{mustParseKeystroke(t, "shift"), None[string](), []KeyContext{terminalContext(t)}}}
		if !dispatchTestObservedEqual(*observedKeystrokes, want) {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}
	})
}

func TestKeyDispatch_PendingModifierObservedWhenBindingResolves(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTest(tc.Cx(), []KeyBinding{
			NewKeyBinding("shift", dispatchTestTestAction{}, Some("Terminal")),
			NewKeyBinding("shift f1", dispatchTestSecondaryTestAction{}, Some("Terminal")),
		})
		defer cleanup()
		observedKeystrokes, observer := captureObservedKeystrokes(cx)
		defer observer.Unsubscribe()

		simulateModifierTap(cx, ModifiersShift())
		if len(*observedKeystrokes) != 0 || *actionCount != 0 {
			t.Fatalf("observed %v count %d", *observedKeystrokes, *actionCount)
		}

		dispatchTestAdvance(cx, pendingInputTimeoutDuration)
		if *actionCount != 1 {
			t.Fatalf("action count = %d", *actionCount)
		}
		want := []dispatchTestObservedKeystroke{{mustParseKeystroke(t, "shift"), Some(dispatchTestTestAction{}.Name()), nil}}
		if !dispatchTestObservedEqual(*observedKeystrokes, want) {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}

		*observedKeystrokes = nil
		simulateModifierTap(cx, ModifiersShift())
		if len(*observedKeystrokes) != 0 {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}
		cx.SimulateKeystrokes("f1")
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 1)
		want = []dispatchTestObservedKeystroke{{mustParseKeystroke(t, "f1"), Some(dispatchTestSecondaryTestAction{}.Name()), []KeyContext{terminalContext(t)}}}
		if !dispatchTestObservedEqual(*observedKeystrokes, want) {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}
	})
}

func TestKeyDispatch_CombinedShortcutHasNoStandaloneModifier(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, _, cleanup := setupPendingInputTest(tc.Cx(), []KeyBinding{
			NewKeyBinding("shift", dispatchTestTestAction{}, Some("Terminal")),
			NewKeyBinding("shift-f1", dispatchTestTestAction{}, Some("Terminal")),
		})
		defer cleanup()
		var interceptedKeystrokes []Keystroke
		observedKeystrokes, observer := captureObservedKeystrokes(cx)
		defer observer.Unsubscribe()
		var interceptor Subscription
		cx.Update(func(_ *Window, cx *App) {
			interceptor = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
				interceptedKeystrokes = append(interceptedKeystrokes, event.Keystroke)
			})
		})
		defer interceptor.Unsubscribe()

		cx.SimulateModifiersChange(ModifiersShift())
		if len(interceptedKeystrokes) != 0 {
			t.Fatalf("intercepted = %v", interceptedKeystrokes)
		}
		cx.SimulateKeystrokes("shift-f1")
		cx.SimulateModifiersChange(ModifiersNone())

		shiftF1 := mustParseKeystroke(t, "shift-f1")
		if !slices.Equal(interceptedKeystrokes, []Keystroke{shiftF1}) {
			t.Fatalf("intercepted = %v", interceptedKeystrokes)
		}
		want := []dispatchTestObservedKeystroke{{shiftF1, Some(dispatchTestTestAction{}.Name()), []KeyContext{terminalContext(t)}}}
		if !dispatchTestObservedEqual(*observedKeystrokes, want) {
			t.Fatalf("observed = %v", *observedKeystrokes)
		}
		if *actionCount != 1 {
			t.Fatalf("action count = %d", *actionCount)
		}
	})
}

func dispatchTestActionCount(cx *VisualTestContext, test Entity[dispatchTestPendingTextInputView]) int {
	var n int
	cx.Update(func(_ *Window, cx *App) { n = *test.Read(cx).actionCount })
	return n
}

func dispatchTestText(cx *VisualTestContext, test Entity[dispatchTestPendingTextInputView]) string {
	var s string
	cx.Update(func(_ *Window, cx *App) { s = *test.Read(cx).text })
	return s
}

func TestKeyDispatch_KeystrokeInputPreference(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		test, cx, cleanup := setupAltgrTest(tc.Cx())
		defer cleanup()
		var interceptedPreferences, observedPreferences []InputPreference
		var interceptor, observer Subscription
		cx.Update(func(_ *Window, cx *App) {
			interceptor = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
				interceptedPreferences = append(interceptedPreferences, event.InputPreference)
			})
		})
		defer interceptor.Unsubscribe()
		cx.Update(func(_ *Window, cx *App) {
			observer = cx.ObserveKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
				observedPreferences = append(observedPreferences, event.InputPreference)
			})
		})
		defer observer.Unsubscribe()

		for _, c := range []struct {
			preferCharacterInput bool
			acceptsTextInput     bool
			inputPreference      InputPreference
		}{
			{true, true, InputPreferenceCharacterInput},
			{false, true, InputPreferenceKeyBindings},
			{true, false, InputPreferenceKeyBindings},
			{false, false, InputPreferenceKeyBindings},
		} {
			test.Update(cx, func(test *dispatchTestPendingTextInputView, _ *Context[dispatchTestPendingTextInputView]) {
				*test.acceptsTextInput = c.acceptsTextInput
				*test.actionCount = 0
			})
			interceptedPreferences = nil
			observedPreferences = nil
			result := dispatchAltgrKeyDown(t, cx, c.preferCharacterInput)
			if result.Propagate != (c.inputPreference == InputPreferenceCharacterInput) {
				t.Fatalf("%+v: propagate = %v", c, result.Propagate)
			}
			if !slices.Equal(interceptedPreferences, []InputPreference{c.inputPreference}) {
				t.Fatalf("%+v: intercepted = %v", c, interceptedPreferences)
			}
			if !slices.Equal(observedPreferences, []InputPreference{c.inputPreference}) {
				t.Fatalf("%+v: observed = %v", c, observedPreferences)
			}
			want := 0
			if c.inputPreference == InputPreferenceKeyBindings {
				want = 1
			}
			if got := dispatchTestActionCount(cx, test); got != want {
				t.Fatalf("%+v: action count = %d", c, got)
			}
		}
	})
}

func TestKeyDispatch_InputPreferenceRecheckedAfterInterception(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		test, cx, cleanup := setupAltgrTest(tc.Cx())
		defer cleanup()
		var acceptsTextInput *bool
		cx.Update(func(_ *Window, cx *App) { acceptsTextInput = test.Read(cx).acceptsTextInput })
		var interceptor, observer Subscription
		cx.Update(func(_ *Window, cx *App) {
			interceptor = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
				if (event.InputPreference == InputPreferenceCharacterInput) != *acceptsTextInput {
					t.Errorf("input preference %v with accepts %v", event.InputPreference, *acceptsTextInput)
				}
				*acceptsTextInput = !*acceptsTextInput
			})
		})
		defer interceptor.Unsubscribe()
		var observedPreferences []InputPreference
		cx.Update(func(_ *Window, cx *App) {
			observer = cx.ObserveKeystrokes(func(event *KeystrokeEvent, _ *Window, _ *App) {
				observedPreferences = append(observedPreferences, event.InputPreference)
			})
		})
		defer observer.Unsubscribe()

		for _, c := range []struct {
			initiallyAcceptsTextInput bool
			observedPreference        InputPreference
		}{
			{true, InputPreferenceKeyBindings},
			{false, InputPreferenceCharacterInput},
		} {
			*acceptsTextInput = c.initiallyAcceptsTextInput
			test.Update(cx, func(test *dispatchTestPendingTextInputView, _ *Context[dispatchTestPendingTextInputView]) {
				*test.actionCount = 0
			})
			observedPreferences = nil
			result := dispatchAltgrKeyDown(t, cx, true)
			if result.Propagate != !c.initiallyAcceptsTextInput {
				t.Fatalf("%+v: propagate = %v", c, result.Propagate)
			}
			if !slices.Equal(observedPreferences, []InputPreference{c.observedPreference}) {
				t.Fatalf("%+v: observed = %v", c, observedPreferences)
			}
			want := 0
			if c.initiallyAcceptsTextInput {
				want = 1
			}
			if got := dispatchTestActionCount(cx, test); got != want {
				t.Fatalf("%+v: action count = %d", c, got)
			}
		}
	})
}

func TestKeyDispatch_KeystrokeInterceptorsCanConsumePreferredCharacterInput(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		test, cx, cleanup := setupAltgrTest(tc.Cx())
		defer cleanup()
		intercepted := false
		var interceptor Subscription
		cx.Update(func(_ *Window, cx *App) {
			interceptor = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, cx *App) {
				if event.InputPreference != InputPreferenceCharacterInput {
					t.Errorf("input preference = %v", event.InputPreference)
				}
				intercepted = true
				cx.StopPropagation()
			})
		})
		defer interceptor.Unsubscribe()

		result := dispatchAltgrKeyDown(t, cx, true)
		if !intercepted || result.Propagate {
			t.Fatalf("intercepted %v propagate %v", intercepted, result.Propagate)
		}
		if got := dispatchTestActionCount(cx, test); got != 0 {
			t.Fatalf("action count = %d", got)
		}
	})
}

func TestKeyDispatch_ConsumedModifierPreservesPendingInput(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		test, cx, cleanup := addPendingTextInputView(tc.Cx(), false)
		defer cleanup()
		focusPendingTextInputView(cx, test)
		var subscription Subscription
		cx.Update(func(_ *Window, cx *App) {
			subscription = cx.InterceptKeystrokes(func(event *KeystrokeEvent, _ *Window, cx *App) {
				if event.Keystroke.Key == "shift" {
					cx.StopPropagation()
				}
			})
		})
		defer subscription.Unsubscribe()

		for _, binding := range []string{"j shift", "j k"} {
			cx.Update(func(_ *Window, cx *App) {
				cx.ClearKeyBindings()
				cx.BindKeys([]KeyBinding{NewKeyBinding(binding, dispatchTestTestAction{}, Some("Terminal"))})
			})
			test.Update(cx, func(test *dispatchTestPendingTextInputView, _ *Context[dispatchTestPendingTextInputView]) {
				*test.text = ""
			})
			cx.SimulateKeystrokes("j")
			dispatchTestAdvance(cx, pendingInputTimeoutDuration/2)
			var pendingKeystrokes []Keystroke
			cx.Update(func(window *Window, _ *App) {
				pendingInput, ok := window.PendingInput()
				if !ok {
					t.Fatal("pending input")
				}
				pendingKeystrokes = slices.Clone(pendingInput.Keystrokes())
			})

			cx.SimulateModifiersChange(ModifiersShift())
			cx.SimulateModifiersChange(ModifiersNone())
			cx.Update(func(window *Window, _ *App) {
				pendingInput, ok := window.PendingInput()
				if !ok {
					t.Fatal("pending input")
				}
				if !slices.Equal(pendingInput.Keystrokes(), pendingKeystrokes) {
					t.Fatalf("%s: keystrokes = %v", binding, pendingInput.Keystrokes())
				}
				if _, ok := pendingInput.Timeout(); !ok {
					t.Fatalf("%s: timeout missing", binding)
				}
			})
			if n, s := dispatchTestActionCount(cx, test), dispatchTestText(cx, test); n != 0 || s != "" {
				t.Fatalf("%s: count %d text %q", binding, n, s)
			}

			dispatchTestAdvance(cx, pendingInputTimeoutDuration/2)
			dispatchTestAssertPending(t, cx, false)
			if n, s := dispatchTestActionCount(cx, test), dispatchTestText(cx, test); n != 0 || s != "j" {
				t.Fatalf("%s: count %d text %q", binding, n, s)
			}
		}
	})
}

func TestKeyDispatch_PrintablePendingInputReplaysOnTimeout(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		tcx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("j k", dispatchTestTestAction{}, Some("Terminal"))})
		})
		test, cx, cleanup := addPendingTextInputView(tcx, false)
		defer cleanup()
		focusPendingTextInputView(cx, test)

		cx.SimulateKeystrokes("j")

		cx.Update(func(window *Window, _ *App) {
			pendingInput, _ := dispatchTestPendingTimeout(t, window)
			if len(pendingInput.Keystrokes()) != 1 {
				t.Fatalf("keystrokes = %v", pendingInput.Keystrokes())
			}
		})
		if n, s := dispatchTestActionCount(cx, test), dispatchTestText(cx, test); n != 0 || s != "" {
			t.Fatalf("count %d text %q", n, s)
		}

		dispatchTestAdvance(cx, pendingInputTimeoutDuration)

		dispatchTestAssertPending(t, cx, false)
		if n, s := dispatchTestActionCount(cx, test), dispatchTestText(cx, test); n != 0 || s != "j" {
			t.Fatalf("count %d text %q", n, s)
		}
	})
}

func TestKeyDispatch_PendingInputTimeoutDispatchesShorterBinding(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)
		cx.Update(func(window *Window, _ *App) {
			pendingInput, timeout := dispatchTestPendingTimeout(t, window)
			if len(pendingInput.Keystrokes()) != 1 {
				t.Fatalf("keystrokes = %v", pendingInput.Keystrokes())
			}
			if timeout.Duration() != pendingInputTimeoutDuration {
				t.Fatalf("duration = %v", timeout.Duration())
			}
		})
		if *actionCount != 0 {
			t.Fatalf("action count = %d", *actionCount)
		}

		for range 10 {
			cx.Executor().AdvanceClock(pendingInputTimeoutDuration / 10)
			cx.Update(func(window *Window, _ *App) { window.Refresh() })
			cx.RunUntilParked()
		}

		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 0)
	})
}

func TestKeyDispatch_RunningPendingInputTimeoutResetsWhenBindingAdvances(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*4/5)
		cx.SimulateKeystrokes("h")
		cx.RunUntilParked()

		cx.Update(func(window *Window, cx *App) {
			pendingInput, timeout := dispatchTestPendingTimeout(t, window)
			if len(pendingInput.Keystrokes()) != 2 || timeout.IsPaused() || timeout.Remaining(cx) != pendingInputTimeoutDuration {
				t.Fatalf("keystrokes %d paused %v remaining %v", len(pendingInput.Keystrokes()), timeout.IsPaused(), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration/5)
		dispatchTestAssertPending(t, cx, true)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 0)

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*4/5)
		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 1)
	})
}

func TestKeyDispatch_PendingInputTimeoutStartsWhenBindingBecomesAmbiguous(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTest(tc.Cx(), []KeyBinding{
			NewKeyBinding("ctrl-b h", dispatchTestSecondaryTestAction{}, Some("Terminal")),
			NewKeyBinding("ctrl-b h j", dispatchTestTestAction{}, Some("Terminal")),
		})
		defer cleanup()
		simulatePendingBinding(cx)

		cx.Update(func(window *Window, _ *App) {
			pendingInput, ok := window.PendingInput()
			if !ok {
				t.Fatal("pending input")
			}
			if len(pendingInput.Keystrokes()) != 1 {
				t.Fatalf("keystrokes = %v", pendingInput.Keystrokes())
			}
			if _, ok := pendingInput.Timeout(); ok {
				t.Fatal("unexpected timeout")
			}
		})

		cx.SimulateKeystrokes("h")
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			pendingInput, timeout := dispatchTestPendingTimeout(t, window)
			if len(pendingInput.Keystrokes()) != 2 || timeout.Remaining(cx) != pendingInputTimeoutDuration {
				t.Fatalf("keystrokes %d remaining %v", len(pendingInput.Keystrokes()), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration)
		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 1)
	})
}

func TestKeyDispatch_InvalidContinuationWhileTimeoutPausedReplaysPendingInput(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTest(tc.Cx(), []KeyBinding{
			NewKeyBinding("ctrl-b", dispatchTestTestAction{}, Some("Terminal")),
			NewKeyBinding("ctrl-b h", dispatchTestSecondaryTestAction{}, Some("Terminal")),
			NewKeyBinding("x", dispatchTestSecondaryTestAction{}, Some("Terminal")),
		})
		defer cleanup()
		simulatePendingBinding(cx)
		pauseOwner := dispatchTestNewPauseOwner(cx)
		dispatchTestSetPaused(t, cx, pauseOwner, true, true)
		cx.RunUntilParked()

		cx.SimulateKeystrokes("x")
		cx.RunUntilParked()

		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 1)

		pauseOwner.Release()
		cx.Update(func(*Window, *App) {})
		cx.RunUntilParked()
		dispatchTestAdvance(cx, pendingInputTimeoutDuration)

		cx.Update(func(window *Window, _ *App) {
			if !window.pendingInputIsNone() {
				t.Fatal("pending input left")
			}
		})
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 1)
	})
}

func TestKeyDispatch_PendingInputTimeoutPausesAndResumes(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*7/10)

		pauseOwner := dispatchTestNewPauseOwner(cx)
		defer pauseOwner.Release()
		otherOwner := dispatchTestNewPauseOwner(cx)
		defer otherOwner.Release()
		dispatchTestSetPaused(t, cx, pauseOwner, true, true)
		dispatchTestSetPaused(t, cx, pauseOwner, true, false)
		dispatchTestSetPaused(t, cx, otherOwner, false, false)
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			_, timeout := dispatchTestPendingTimeout(t, window)
			if !timeout.IsPaused() || timeout.Remaining(cx) != pendingInputTimeoutDuration*3/10 {
				t.Fatalf("paused %v remaining %v", timeout.IsPaused(), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*2)
		dispatchTestAssertPending(t, cx, true)
		if *actionCount != 0 {
			t.Fatalf("action count = %d", *actionCount)
		}

		dispatchTestSetPaused(t, cx, pauseOwner, false, true)
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			_, timeout := dispatchTestPendingTimeout(t, window)
			if timeout.IsPaused() || timeout.Remaining(cx) != pendingInputTimeoutDuration*3/10 {
				t.Fatalf("paused %v remaining %v", timeout.IsPaused(), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*3/10)

		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 0)
	})
}

func TestKeyDispatch_PendingInputTimeoutResumesWhenOwnerIsReleased(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*7/10)

		pauseOwner := dispatchTestNewPauseOwner(cx)
		dispatchTestSetPaused(t, cx, pauseOwner, true, true)
		cx.RunUntilParked()

		pauseOwner.Release()
		cx.Update(func(*Window, *App) {})
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			_, timeout := dispatchTestPendingTimeout(t, window)
			if timeout.IsPaused() || timeout.Remaining(cx) != pendingInputTimeoutDuration*3/10 {
				t.Fatalf("paused %v remaining %v", timeout.IsPaused(), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*3/10)

		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 1, 0)
	})
}

func TestKeyDispatch_PendingInputTimeoutResetsWhenBindingAdvances(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)
		dispatchTestAdvance(cx, pendingInputTimeoutDuration/2)
		pauseOwner := dispatchTestNewPauseOwner(cx)
		defer pauseOwner.Release()
		dispatchTestSetPaused(t, cx, pauseOwner, true, true)
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			_, timeout := dispatchTestPendingTimeout(t, window)
			if timeout.Remaining(cx) != pendingInputTimeoutDuration/2 {
				t.Fatalf("remaining %v", timeout.Remaining(cx))
			}
		})

		cx.SimulateKeystrokes("h")
		cx.RunUntilParked()
		cx.Update(func(window *Window, cx *App) {
			pendingInput, timeout := dispatchTestPendingTimeout(t, window)
			if len(pendingInput.Keystrokes()) != 2 || !timeout.IsPaused() || timeout.Remaining(cx) != pendingInputTimeoutDuration {
				t.Fatalf("keystrokes %d paused %v remaining %v", len(pendingInput.Keystrokes()), timeout.IsPaused(), timeout.Remaining(cx))
			}
		})

		dispatchTestAdvance(cx, pendingInputTimeoutDuration*2)
		dispatchTestAssertPending(t, cx, true)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 0)

		dispatchTestSetPaused(t, cx, pauseOwner, false, true)
		cx.RunUntilParked()
		dispatchTestAdvance(cx, pendingInputTimeoutDuration)

		dispatchTestAssertPending(t, cx, false)
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 1)
	})
}

func TestKeyDispatch_ClearingPendingInputInvalidatesTimeoutPause(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx, actionCount, secondaryActionCount, cleanup := setupPendingInputTimeoutTest(tc.Cx())
		defer cleanup()
		simulatePendingBinding(cx)
		pauseOwner := dispatchTestNewPauseOwner(cx)
		cx.Update(func(window *Window, cx *App) {
			if !window.SetPendingInputTimeoutPaused(pauseOwner, true, cx) {
				t.Fatal("pause failed")
			}
			other := cx.FocusHandle()
			window.Focus(other, cx)
			other.Release()
			if window.HasPendingKeystrokes() {
				t.Fatal("focus change kept pending keystrokes")
			}
		})

		pauseOwner.Release()
		cx.Update(func(*Window, *App) {})
		cx.RunUntilParked()
		dispatchTestAdvance(cx, pendingInputTimeoutDuration)

		cx.Update(func(window *Window, _ *App) {
			if !window.pendingInputIsNone() {
				t.Fatal("pending input left")
			}
		})
		dispatchTestAssertCounts(t, actionCount, secondaryActionCount, 0, 0)
	})
}

func TestKeyDispatch_InputHandlerPending(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		tcx.Update(func(cx *App) {
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-b", dispatchTestTestAction{}, Some("Terminal"))})
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-b h", dispatchTestTestAction{}, Some("Terminal"))})
			cx.BindKeys([]KeyBinding{NewKeyBinding("ctrl-x k", dispatchTestTestAction{}, Some("Terminal"))})
		})
		test, cx, cleanup := addPendingTextInputView(tcx, false)
		defer cleanup()
		var focusHandle FocusHandle
		cx.Update(func(_ *Window, cx *App) { focusHandle = test.Read(cx).focusHandle })
		cx.Update(func(window *Window, cx *App) {
			window.Focus(focusHandle, cx)
			window.ActivateWindow()
		})

		if got := queryPrefersIMEForPrintableKeys(cx); got != Some(true) {
			t.Fatalf("prefers IME = %v", got)
		}
		cx.SimulateKeystrokes("ctrl-x")
		dispatchTestAssertPending(t, cx, true)
		if got := queryPrefersIMEForPrintableKeys(cx); got != Some(false) {
			t.Fatalf("prefers IME while pending = %v", got)
		}

		prefersIMEAfterBlur := None[bool]()
		{
			platformWindow := cx.testWindow(cx.WindowHandle())
			inputHandler := platformWindow.TakeInputHandler()
			cx.Update(func(window *Window, cx *App) {
				window.Blur(cx)
				if window.HasPendingKeystrokes() || !window.pendingInputIsNone() {
					t.Fatal("blur kept pending input")
				}
			})
			if inputHandler != nil {
				prefersIMEAfterBlur = Some(inputHandler.QueryPrefersIMEForPrintableKeys())
				platformWindow.SetInputHandler(inputHandler)
			}
		}
		if prefersIMEAfterBlur != Some(true) {
			t.Fatalf("prefers IME after blur = %v", prefersIMEAfterBlur)
		}
		cx.Update(func(window *Window, cx *App) { window.Focus(focusHandle, cx) })

		cx.SimulateKeystrokes("ctrl-x")
		if got := queryPrefersIMEForPrintableKeys(cx); got != Some(false) {
			t.Fatalf("prefers IME while pending = %v", got)
		}
		cx.SimulateKeystrokes("k")
		dispatchTestAssertPending(t, cx, false)
		if got := queryPrefersIMEForPrintableKeys(cx); got != Some(true) {
			t.Fatalf("prefers IME = %v", got)
		}
		if n, s := dispatchTestActionCount(cx, test), dispatchTestText(cx, test); n != 1 || s != "" {
			t.Fatalf("count %d text %q", n, s)
		}

		cx.SimulateKeystrokes("ctrl-b [")
		if s := dispatchTestText(cx, test); s != "[" {
			t.Fatalf("text = %q", s)
		}
	})
}

func TestKeyDispatch_TreeMaintenance(t *testing.T) {
	tree := testDispatchTree(nil)
	root := tree.pushNode()
	tree.setKeyContext(mustParseContext(t, "Workspace"))
	tree.setFocusID(1)
	tree.setViewID(10)
	child := tree.pushNode()
	tree.setKeyContext(mustParseContext(t, "Editor"))
	tree.setFocusID(2)
	tree.setViewID(20)
	tree.onAction(reflectTypeOfAction(dispatchTestTestAction{}), func(any, DispatchPhase, *Window, *App) {})
	tree.popNode()
	tree.popNode()

	if !tree.focusContains(1, 2) || tree.focusContains(2, 1) {
		t.Fatal("focusContains")
	}
	if got := tree.focusPath(2); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("focusPath = %v", got)
	}
	var views []EntityID
	for v := range tree.viewPathReversed(20) {
		views = append(views, v)
	}
	if len(views) != 2 || views[0] != 20 || views[1] != 10 {
		t.Fatalf("viewPathReversed = %v", views)
	}
	if !tree.isActionAvailable(dispatchTestTestAction{}, child) || tree.isActionAvailable(dispatchTestTestAction{}, root) {
		t.Fatal("isActionAvailable")
	}

	tree.setActiveNode(child)
	if len(tree.contextStack) != 2 || len(tree.nodeStack) != 2 || len(tree.viewStack) != 2 {
		t.Fatalf("setActiveNode stacks: %d %d %d", len(tree.contextStack), len(tree.nodeStack), len(tree.viewStack))
	}

	next := testDispatchTree(nil)
	reused := next.reuseSubtree(Range{Start: 0, End: 2}, tree, Some[FocusID](2))
	if !reused.containsFocus() || next.len() != 2 || reused.refreshNodeID(child) != child {
		t.Fatal("reuseSubtree")
	}
	if len(next.node(child).actionListeners) != 1 || len(tree.node(child).actionListeners) != 0 {
		t.Fatal("listeners not moved")
	}

	next.truncate(1)
	if _, ok := next.focusableNodeID(2); ok || next.len() != 1 {
		t.Fatal("truncate")
	}
}

func reflectTypeOfAction(a Action) reflect.Type { return reflect.TypeOf(a) }
