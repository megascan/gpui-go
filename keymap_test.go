package gpui

import (
	"slices"
	"strings"
	"testing"
)

type keymapTestActionAlpha struct{}
type keymapTestActionBeta struct{}
type keymapTestActionGamma struct{}
type keymapTestActionDelta struct{}

func (keymapTestActionAlpha) Name() string         { return "test_only::ActionAlpha" }
func (a keymapTestActionAlpha) BoxedClone() Action { return a }
func (keymapTestActionAlpha) PartialEq(other Action) bool {
	_, ok := other.(keymapTestActionAlpha)
	return ok
}
func (keymapTestActionBeta) Name() string         { return "test_only::ActionBeta" }
func (a keymapTestActionBeta) BoxedClone() Action { return a }
func (keymapTestActionBeta) PartialEq(other Action) bool {
	_, ok := other.(keymapTestActionBeta)
	return ok
}
func (keymapTestActionGamma) Name() string         { return "test_only::ActionGamma" }
func (a keymapTestActionGamma) BoxedClone() Action { return a }
func (keymapTestActionGamma) PartialEq(other Action) bool {
	_, ok := other.(keymapTestActionGamma)
	return ok
}
func (keymapTestActionDelta) Name() string         { return "test_only::ActionDelta" }
func (a keymapTestActionDelta) BoxedClone() Action { return a }
func (keymapTestActionDelta) PartialEq(other Action) bool {
	_, ok := other.(keymapTestActionDelta)
	return ok
}

func mustParseKeystroke(t *testing.T, source string) Keystroke {
	t.Helper()
	keystroke, err := KeystrokeParse(source)
	if err != nil {
		t.Fatalf("KeystrokeParse(%q): %v", source, err)
	}
	return keystroke
}

func keystrokes(t *testing.T, sources ...string) []Keystroke {
	t.Helper()
	var result []Keystroke
	for _, source := range sources {
		result = append(result, mustParseKeystroke(t, source))
	}
	return result
}

func contexts(t *testing.T, sources ...string) []KeyContext {
	t.Helper()
	var result []KeyContext
	for _, source := range sources {
		result = append(result, mustParseContext(t, source))
	}
	return result
}

func kb(keystrokes string, action Action) KeyBinding {
	return NewKeyBinding(keystrokes, action, None[string]())
}

func kbIn(keystrokes string, action Action, context string) KeyBinding {
	return NewKeyBinding(keystrokes, action, Some(context))
}

func assertDepth(t *testing.T, keymap *Keymap, binding KeyBinding, ctx []KeyContext, want int, wantOK bool) {
	t.Helper()
	got, ok := keymap.bindingEnabled(&binding, ctx)
	if ok != wantOK || (ok && got != want) {
		t.Fatalf("bindingEnabled = %d, %v; want %d, %v", got, ok, want, wantOK)
	}
}

func TestKeymap_Keymap(t *testing.T) {
	bindings := []KeyBinding{
		kb("ctrl-a", keymapTestActionAlpha{}),
		kbIn("ctrl-a", keymapTestActionBeta{}, "pane"),
		kbIn("ctrl-a", keymapTestActionGamma{}, "editor && mode==full"),
	}

	keymap := DefaultKeymap()
	cloned := make([]KeyBinding, 0, len(bindings))
	for _, b := range bindings {
		cloned = append(cloned, b.Clone())
	}
	keymap.AddBindings(cloned)

	assertDepth(t, keymap, bindings[0], nil, 0, true)
	assertDepth(t, keymap, bindings[0], contexts(t, "terminal"), 1, true)

	assertDepth(t, keymap, bindings[1], contexts(t, "barf x=y"), 0, false)
	assertDepth(t, keymap, bindings[1], contexts(t, "pane x=y"), 1, true)

	assertDepth(t, keymap, bindings[2], contexts(t, "editor"), 0, false)
	assertDepth(t, keymap, bindings[2], contexts(t, "editor mode=full"), 1, true)
}

func TestKeymap_DepthPrecedence(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-a", keymapTestActionBeta{}, "pane"),
		kbIn("ctrl-a", keymapTestActionGamma{}, "editor"),
	})

	result, pending := keymap.BindingsForInput(keystrokes(t, "ctrl-a"), contexts(t, "pane", "editor"))

	if pending || len(result) != 2 {
		t.Fatalf("pending = %v, len = %d", pending, len(result))
	}
	if !result[0].action.PartialEq(keymapTestActionGamma{}) || !result[1].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}
}

func TestKeymap_KeymapDisabled(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-a", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-b", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-a", NoAction{}, "editor && mode==full"),
		kb("ctrl-b", NoAction{}),
	})

	empty := func(input string, ctx string) bool {
		result, _ := keymap.BindingsForInput(keystrokes(t, input), contexts(t, ctx))
		return len(result) == 0
	}
	if !empty("ctrl-a", "barf") {
		t.Fatal("ctrl-a barf")
	}
	if empty("ctrl-a", "editor") {
		t.Fatal("ctrl-a editor")
	}
	if !empty("ctrl-a", "editor mode=full") {
		t.Fatal("ctrl-a editor mode=full")
	}
	if !empty("ctrl-b", "barf") {
		t.Fatal("ctrl-b barf")
	}
}

func assertResult(t *testing.T, keymap *Keymap, input []Keystroke, ctx []KeyContext, wantLen int, wantPending bool) []KeyBinding {
	t.Helper()
	result, pending := keymap.BindingsForInput(input, ctx)
	if len(result) != wantLen || pending != wantPending {
		t.Fatalf("len = %d, pending = %v; want %d, %v", len(result), pending, wantLen, wantPending)
	}
	return result
}

func TestKeymap_MultipleKeystrokeBindingDisabled(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("space w w", keymapTestActionAlpha{}, "workspace"),
		kbIn("space w w", NoAction{}, "editor"),
	})

	space := keystrokes(t, "space")
	spaceW := keystrokes(t, "space", "w")
	spaceWW := keystrokes(t, "space", "w", "w")
	workspaceContext := contexts(t, "workspace")
	editorWorkspaceContext := contexts(t, "workspace", "editor")

	assertResult(t, keymap, space, workspaceContext, 0, true)
	assertResult(t, keymap, space, editorWorkspaceContext, 0, false)
	assertResult(t, keymap, spaceW, workspaceContext, 0, true)
	assertResult(t, keymap, spaceW, editorWorkspaceContext, 0, false)
	assertResult(t, keymap, spaceWW, workspaceContext, 1, false)
	assertResult(t, keymap, spaceWW, editorWorkspaceContext, 0, false)

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("space w w", keymapTestActionAlpha{}, "workspace"),
		kbIn("space w w", NoAction{}, "editor"),
		kbIn("space w x", keymapTestActionAlpha{}, "editor"),
	})
	assertResult(t, keymap, space, editorWorkspaceContext, 0, true)

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("space w w", keymapTestActionAlpha{}, "workspace"),
		kbIn("space w x", keymapTestActionAlpha{}, "editor"),
		kbIn("space w w", NoAction{}, "editor"),
	})
	assertResult(t, keymap, space, editorWorkspaceContext, 0, true)

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("space w w", keymapTestActionAlpha{}, "workspace"),
		kbIn("space w x", keymapTestActionAlpha{}, "workspace"),
		kbIn("space w w", NoAction{}, "editor"),
	})
	assertResult(t, keymap, space, editorWorkspaceContext, 0, true)
}

func TestKeymap_OverrideMultikey(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-w left", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-w", NoAction{}, "editor"),
	})
	assertResult(t, keymap, keystrokes(t, "ctrl-w"), contexts(t, "editor"), 0, true)

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-w left", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-w", keymapTestActionBeta{}, "editor"),
	})
	assertResult(t, keymap, keystrokes(t, "ctrl-w"), contexts(t, "editor"), 1, false)
}

func TestKeymap_SimpleDisable(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-x", NoAction{}, "editor"),
	})
	assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "editor"), 0, false)
}

func TestKeymap_DisableWeakerSourcesOnly(t *testing.T) {
	const (
		user        KeyBindingMetaIndex = 0
		vim         KeyBindingMetaIndex = 1
		base        KeyBindingMetaIndex = 2
		defaultMeta KeyBindingMetaIndex = 3
	)
	editorContext := contexts(t, "editor")
	ctrlX := keystrokes(t, "ctrl-x")

	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "editor").WithMeta(defaultMeta),
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(base),
	})
	if result, _ := keymap.BindingsForInput(ctrlX, editorContext); len(result) != 0 {
		t.Fatalf("base null: %v", result)
	}

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(defaultMeta),
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(base),
		kb("ctrl-x", keymapTestActionBeta{}).WithMeta(user),
	})
	result, _ := keymap.BindingsForInput(ctrlX, editorContext)
	if len(result) != 1 || !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("user binding: %v", result)
	}

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(base),
		kbIn("ctrl-x", keymapTestActionBeta{}, "workspace").WithMeta(user),
	})
	result, _ = keymap.BindingsForInput(ctrlX, contexts(t, "workspace", "editor"))
	if len(result) != 1 || !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("shallower user binding: %v", result)
	}

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "editor").WithMeta(defaultMeta),
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(base),
		kbIn("ctrl-x", keymapTestActionGamma{}, "editor").WithMeta(vim),
	})
	result, _ = keymap.BindingsForInput(ctrlX, editorContext)
	if len(result) != 1 || !result[0].action.PartialEq(keymapTestActionGamma{}) {
		t.Fatalf("vim binding: %v", result)
	}

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "editor").WithMeta(defaultMeta),
		kbIn("ctrl-x", keymapTestActionGamma{}, "editor").WithMeta(vim),
		kbIn("ctrl-x", NoAction{}, "editor").WithMeta(user),
	})
	if result, _ := keymap.BindingsForInput(ctrlX, editorContext); len(result) != 0 {
		t.Fatalf("user null: %v", result)
	}
}

func TestKeymap_FailToDisable(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "editor"),
		kbIn("ctrl-x", NoAction{}, "workspace"),
	})
	assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "workspace", "editor"), 1, false)
}

func TestKeymap_DisableDeeper(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionAlpha{}, "workspace"),
		kbIn("ctrl-x", NoAction{}, "editor"),
	})
	assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "workspace", "editor"), 0, false)
}

func TestKeymap_PendingMatchEnabled(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionBeta{}, "vim_mode == normal"),
		kbIn("ctrl-x 0", keymapTestActionAlpha{}, "Workspace"),
	})
	result := assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "Workspace", "Pane", "Editor vim_mode=normal"), 1, true)
	if !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}
}

func TestKeymap_PendingMatchEnabledExtended(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionBeta{}, "vim_mode == normal"),
		kbIn("ctrl-x 0", NoAction{}, "Workspace"),
	})
	result := assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "Workspace", "Pane", "Editor vim_mode=normal"), 1, false)
	if !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}

	keymap = DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x", keymapTestActionBeta{}, "Workspace"),
		kbIn("ctrl-x 0", NoAction{}, "vim_mode == normal"),
	})
	result = assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "Workspace", "Pane", "Editor vim_mode=normal"), 1, false)
	if !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}
}

func TestKeymap_OverridingPrefix(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-x 0", keymapTestActionAlpha{}, "Workspace"),
		kbIn("ctrl-x", keymapTestActionBeta{}, "vim_mode == normal"),
	})
	result := assertResult(t, keymap, keystrokes(t, "ctrl-x"), contexts(t, "Workspace", "Pane", "Editor vim_mode=normal"), 1, false)
	if !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}
}

func TestKeymap_ContextPrecedenceWithSameSource(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("cmd-r", keymapTestActionAlpha{}, "Workspace"),
		kbIn("cmd-r", keymapTestActionBeta{}, "Editor"),
	})
	result, _ := keymap.BindingsForInput(keystrokes(t, "cmd-r"), contexts(t, "Workspace", "Editor"))
	if len(result) != 2 || !result[0].action.PartialEq(keymapTestActionBeta{}) || !result[1].action.PartialEq(keymapTestActionAlpha{}) {
		t.Fatalf("result = %v", result)
	}
}

func assertBindingsForAction(t *testing.T, keymap *Keymap, action Action, expected []string) {
	t.Helper()
	actual := []string{}
	for _, binding := range keymap.BindingsForAction(action) {
		actual = append(actual, binding.keystrokes[0].Inner().Unparse(OperatingSystemMac))
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("%s: got %v, want %v", action.Name(), actual, expected)
	}
}

func TestKeymap_BindingsForAction(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-a", keymapTestActionAlpha{}, "pane"),
		kbIn("ctrl-b", keymapTestActionBeta{}, "editor && mode == full"),
		kbIn("ctrl-c", keymapTestActionGamma{}, "workspace"),
		kbIn("ctrl-a", NoAction{}, "pane && active"),
		kbIn("ctrl-b", NoAction{}, "editor"),
	})

	assertBindingsForAction(t, keymap, keymapTestActionAlpha{}, []string{"ctrl-a"})
	assertBindingsForAction(t, keymap, keymapTestActionBeta{}, []string{})
	assertBindingsForAction(t, keymap, keymapTestActionGamma{}, []string{"ctrl-c"})
}

func TestKeymap_TargetedUnbindIgnoresTargetContext(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("tab", keymapTestActionAlpha{}, "Editor"),
		kbIn("tab", keymapTestActionBeta{}, "Editor && showing_completions"),
		kbIn("tab", Unbind("test_only::ActionAlpha"), "Editor && edit_prediction"),
	})
	result := assertResult(t, keymap, keystrokes(t, "tab"), contexts(t, "Editor showing_completions edit_prediction"), 1, false)
	if !result[0].action.PartialEq(keymapTestActionBeta{}) {
		t.Fatalf("result = %v", result)
	}
}

func TestKeymap_BindingsForActionKeepsBindingForNarrowerTargetedUnbind(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("tab", keymapTestActionAlpha{}, "Editor"),
		kbIn("tab", Unbind("test_only::ActionAlpha"), "Editor && edit_prediction"),
		kbIn("tab", keymapTestActionBeta{}, "Editor && showing_completions"),
	})
	assertBindingsForAction(t, keymap, keymapTestActionAlpha{}, []string{"tab"})
	assertBindingsForAction(t, keymap, keymapTestActionBeta{}, []string{"tab"})
}

func TestKeymap_BindingsForActionRemovesBindingForBroaderTargetedUnbind(t *testing.T) {
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("tab", keymapTestActionAlpha{}, "Editor && edit_prediction"),
		kbIn("tab", Unbind("test_only::ActionAlpha"), "Editor"),
	})
	if got := keymap.BindingsForAction(keymapTestActionAlpha{}); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestKeymap_SourcePrecedenceSorting(t *testing.T) {
	keymap := DefaultKeymap()

	defaultBinding := kbIn("cmd-r", keymapTestActionAlpha{}, "Editor")
	defaultBinding.SetMeta(3)
	keymap.AddBindings([]KeyBinding{defaultBinding})

	userBinding := kbIn("cmd-r", keymapTestActionBeta{}, "Editor")
	userBinding.SetMeta(0)
	keymap.AddBindings([]KeyBinding{userBinding})

	result, _ := keymap.BindingsForInput(keystrokes(t, "cmd-r"), contexts(t, "Editor"))
	if len(result) != 2 || !result[0].action.PartialEq(keymapTestActionBeta{}) || !result[1].action.PartialEq(keymapTestActionAlpha{}) {
		t.Fatalf("result = %v", result)
	}
}

func describeBindings(bindings []KeyBinding) []string {
	result := []string{}
	for _, binding := range bindings {
		var parts []string
		for _, keystroke := range binding.Keystrokes() {
			parts = append(parts, keystroke.Inner().Unparse(OperatingSystemMac))
		}
		result = append(result, strings.Join(parts, " ")+"|"+binding.Action().Name())
	}
	return result
}

func assertDescribed(t *testing.T, bindings []KeyBinding, expected ...string) {
	t.Helper()
	if expected == nil {
		expected = []string{}
	}
	if got := describeBindings(bindings); !slices.Equal(got, expected) {
		t.Fatalf("got %v, want %v", got, expected)
	}
}

func TestKeymap_PossibleNextBindingsExcludeNullBindings(t *testing.T) {
	keymap := NewKeymap([]KeyBinding{
		kbIn("ctrl-k a", keymapTestActionAlpha{}, "Editor"),
		kbIn("ctrl-k b", keymapTestActionBeta{}, "Editor"),
		kbIn("ctrl-k a", NoAction{}, "Editor"),
	})
	bindings := keymap.PossibleNextBindingsForInput(keystrokes(t, "ctrl-k"), contexts(t, "Editor"))
	assertDescribed(t, bindings, "ctrl-k b|test_only::ActionBeta")
}

func TestKeymap_PossibleNextBindingsExcludeUnboundBindings(t *testing.T) {
	keymap := NewKeymap([]KeyBinding{
		kbIn("ctrl-k a", keymapTestActionAlpha{}, "Editor"),
		kbIn("ctrl-k b", keymapTestActionBeta{}, "Editor"),
		kbIn("ctrl-k a", Unbind("test_only::ActionAlpha"), "Editor"),
	})
	bindings := keymap.PossibleNextBindingsForInput(keystrokes(t, "ctrl-k"), contexts(t, "Editor"))
	assertDescribed(t, bindings, "ctrl-k b|test_only::ActionBeta")
}

func TestKeymap_PossibleNextBindingsKeepUserBindingOverBaseKeymapNull(t *testing.T) {
	const (
		user        KeyBindingMetaIndex = 0
		base        KeyBindingMetaIndex = 2
		defaultMeta KeyBindingMetaIndex = 3
	)
	keymap := DefaultKeymap()
	keymap.AddBindings([]KeyBinding{
		kbIn("ctrl-k a", keymapTestActionAlpha{}, "Editor").WithMeta(defaultMeta),
		kbIn("ctrl-k a", NoAction{}, "Editor").WithMeta(base),
		kbIn("ctrl-k a", keymapTestActionBeta{}, "Editor").WithMeta(user),
	})
	bindings := keymap.PossibleNextBindingsForInput(keystrokes(t, "ctrl-k"), contexts(t, "Editor"))
	assertDescribed(t, bindings, "ctrl-k a|test_only::ActionBeta")
}

func TestKeymap_PossibleNextBindingsApplyDisablingByContext(t *testing.T) {
	keymap := NewKeymap([]KeyBinding{
		kbIn("ctrl-k a", keymapTestActionAlpha{}, "Workspace"),
		kbIn("ctrl-k a", keymapTestActionBeta{}, "Editor"),
		kbIn("ctrl-k b", keymapTestActionGamma{}, "Editor"),
		kbIn("ctrl-k a", Unbind("test_only::ActionBeta"), "Editor"),
		kbIn("ctrl-k b", NoAction{}, "Terminal"),
	})
	bindings := keymap.PossibleNextBindingsForInput(keystrokes(t, "ctrl-k"), contexts(t, "Workspace", "Editor"))
	assertDescribed(t, bindings, "ctrl-k b|test_only::ActionGamma", "ctrl-k a|test_only::ActionAlpha")
}

func TestKeymap_PossibleNextBindingsExcludeLongerChordDisabledInDeeperContext(t *testing.T) {
	keymap := NewKeymap([]KeyBinding{
		kbIn("space w w", keymapTestActionAlpha{}, "Workspace"),
		kbIn("space w x", keymapTestActionBeta{}, "Workspace"),
		kbIn("space w w", NoAction{}, "AgentPanel"),
	})
	contextStack := contexts(t, "Workspace", "AgentPanel")

	for _, input := range []string{"space", "space w"} {
		bindings := keymap.PossibleNextBindingsForInput(keystrokes(t, strings.Split(input, " ")...), contextStack)
		assertDescribed(t, bindings, "space w x|test_only::ActionBeta")
	}
}

func TestKeymap_PossibleNextBindingsExcludeDisabledBindingsWithTypedInput(t *testing.T) {
	typedInput := keystrokes(t, "alt-s->ß")
	keymap := NewKeymap([]KeyBinding{
		kb("ß a", keymapTestActionAlpha{}),
		kb("alt-s a", NoAction{}),
	})
	if got := keymap.PossibleNextBindingsForInput(typedInput, nil); len(got) != 0 {
		t.Fatalf("got %v", describeBindings(got))
	}

	typedThenA := keystrokes(t, "alt-s->ß", "a")
	if dispatched, _ := keymap.BindingsForInput(typedThenA, nil); len(dispatched) != 0 {
		t.Fatalf("dispatched %v", describeBindings(dispatched))
	}
}
