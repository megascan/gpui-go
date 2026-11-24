package gpui

import (
	"reflect"
	"testing"
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

const keyDispatchBlocked = "blocked:M0-blocked-tests: needs windows (TestAppContext.AddWindowView), elements and Window dispatch"

func TestKeyDispatch_PendingInputObserversNotifiedOnFocusChangeAndBlur(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_StandaloneModifierDispatch(t *testing.T) { t.Skip(keyDispatchBlocked) }

func TestKeyDispatch_UnboundStandaloneModifierObservation(t *testing.T) { t.Skip(keyDispatchBlocked) }

func TestKeyDispatch_RawModifierHandlerCanSuppressObservation(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingModifierObservedWhenBindingResolves(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_CombinedShortcutHasNoStandaloneModifier(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_KeystrokeInputPreference(t *testing.T) { t.Skip(keyDispatchBlocked) }

func TestKeyDispatch_InputPreferenceRecheckedAfterInterception(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_KeystrokeInterceptorsCanConsumePreferredCharacterInput(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_ConsumedModifierPreservesPendingInput(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PrintablePendingInputReplaysOnTimeout(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingInputTimeoutDispatchesShorterBinding(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_RunningPendingInputTimeoutResetsWhenBindingAdvances(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingInputTimeoutStartsWhenBindingBecomesAmbiguous(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_InvalidContinuationWhileTimeoutPausedReplaysPendingInput(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingInputTimeoutPausesAndResumes(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingInputTimeoutResumesWhenOwnerIsReleased(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_PendingInputTimeoutResetsWhenBindingAdvances(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_ClearingPendingInputInvalidatesTimeoutPause(t *testing.T) {
	t.Skip(keyDispatchBlocked)
}

func TestKeyDispatch_InputHandlerPending(t *testing.T) { t.Skip(keyDispatchBlocked) }

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
