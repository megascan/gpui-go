package gpui

import "testing"

type contextTestAction struct{ name string }

func (a contextTestAction) Name() string       { return "test_only::" + a.name }
func (a contextTestAction) BoxedClone() Action { return a }
func (a contextTestAction) PartialEq(other Action) bool {
	o, ok := other.(contextTestAction)
	return ok && a == o
}

func mustParseContext(t *testing.T, source string) KeyContext {
	t.Helper()
	context, err := KeyContextParse(source)
	if err != nil {
		t.Fatalf("KeyContextParse(%q): %v", source, err)
	}
	return context
}

func mustParsePredicate(t *testing.T, source string) KeyBindingContextPredicate {
	t.Helper()
	predicate, err := KeyBindingContextPredicateParse(source)
	if err != nil {
		t.Fatalf("KeyBindingContextPredicateParse(%q): %v", source, err)
	}
	return predicate
}

func TestContext_ActionsDefinition(t *testing.T) {
	for _, name := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"} {
		var action Action = contextTestAction{name: name}
		if !action.PartialEq(action.BoxedClone()) || action.Name() != "test_only::"+name {
			t.Fatalf("action %s", name)
		}
	}
}

func TestContext_ParseContext(t *testing.T) {
	var expected KeyContext
	expected.Add("baz")
	expected.Set("foo", "bar")
	for _, source := range []string{"baz foo=bar", "baz foo = bar", "  baz foo   =   bar baz", " baz foo = bar"} {
		if got := mustParseContext(t, source); !got.equal(expected) {
			t.Fatalf("KeyContextParse(%q) = %v, want %v", source, got, expected)
		}
	}
}

func TestContext_ParseIdentifiers(t *testing.T) {
	if got := mustParsePredicate(t, "abc12"); got != (KeyBindingContextPredicateIdentifier{Name: "abc12"}) {
		t.Fatalf("got %v", got)
	}
	if got := mustParsePredicate(t, "_1a"); got != (KeyBindingContextPredicateIdentifier{Name: "_1a"}) {
		t.Fatalf("got %v", got)
	}
}

func predIdent(s string) KeyBindingContextPredicate {
	return KeyBindingContextPredicateIdentifier{Name: SharedString(s)}
}

func predEq(a, b string) KeyBindingContextPredicate {
	return KeyBindingContextPredicateEqual{Left: SharedString(a), Right: SharedString(b)}
}

func predNotEq(a, b string) KeyBindingContextPredicate {
	return KeyBindingContextPredicateNotEqual{Left: SharedString(a), Right: SharedString(b)}
}

func predAnd(a, b KeyBindingContextPredicate) KeyBindingContextPredicate {
	return KeyBindingContextPredicateAnd{Left: a, Right: b}
}

func predOr(a, b KeyBindingContextPredicate) KeyBindingContextPredicate {
	return KeyBindingContextPredicateOr{Left: a, Right: b}
}

func predDescendant(a, b KeyBindingContextPredicate) KeyBindingContextPredicate {
	return KeyBindingContextPredicateDescendant{Parent: a, Child: b}
}

func predNot(a KeyBindingContextPredicate) KeyBindingContextPredicate {
	return KeyBindingContextPredicateNot{Predicate: a}
}

func TestContext_ParseNegations(t *testing.T) {
	if got := mustParsePredicate(t, "!abc"); got != predNot(predIdent("abc")) {
		t.Fatalf("got %v", got)
	}
	if got := mustParsePredicate(t, " ! ! abc"); got != predNot(predNot(predIdent("abc"))) {
		t.Fatalf("got %v", got)
	}
}

func TestContext_ParseEqualityOperators(t *testing.T) {
	if got := mustParsePredicate(t, "a == b"); got != predEq("a", "b") {
		t.Fatalf("got %v", got)
	}
	if got := mustParsePredicate(t, "c!=d"); got != predNotEq("c", "d") {
		t.Fatalf("got %v", got)
	}
	_, err := KeyBindingContextPredicateParse("c == !d")
	if err == nil || err.Error() != "operands of == must be identifiers" {
		t.Fatalf("err = %v", err)
	}
}

func TestContext_ParseBooleanOperators(t *testing.T) {
	cases := []struct {
		source string
		want   KeyBindingContextPredicate
	}{
		{"a || b", predOr(predIdent("a"), predIdent("b"))},
		{"a || !b && c", predOr(predIdent("a"), predAnd(predNot(predIdent("b")), predIdent("c")))},
		{"a && b || c&&d", predOr(predAnd(predIdent("a"), predIdent("b")), predAnd(predIdent("c"), predIdent("d")))},
		{"a == b && c || d == e && f", predOr(predAnd(predEq("a", "b"), predIdent("c")), predAnd(predEq("d", "e"), predIdent("f")))},
		{"a && b && c && d", predAnd(predAnd(predAnd(predIdent("a"), predIdent("b")), predIdent("c")), predIdent("d"))},
	}
	for _, c := range cases {
		if got := mustParsePredicate(t, c.source); got != c.want {
			t.Fatalf("%q: got %v, want %v", c.source, got, c.want)
		}
	}
}

func TestContext_ParseParenthesizedExpressions(t *testing.T) {
	if got := mustParsePredicate(t, "a && (b == c || d != e)"); got != predAnd(predIdent("a"), predOr(predEq("b", "c"), predNotEq("d", "e"))) {
		t.Fatalf("got %v", got)
	}
	if got := mustParsePredicate(t, " ( a || b ) "); got != predOr(predIdent("a"), predIdent("b")) {
		t.Fatalf("got %v", got)
	}
}

func TestContext_IsSuperset(t *testing.T) {
	assertIsSuperset := func(a, b string, result bool) {
		t.Helper()
		pa := mustParsePredicate(t, a)
		pb := mustParsePredicate(t, b)
		if pa.IsSuperset(pb) != result {
			t.Fatalf("(%v).is_superset(%v) != %v", pa, pb, result)
		}
	}
	assertIsSuperset("editor", "editor", true)
	assertIsSuperset("editor", "workspace", false)

	assertIsSuperset("editor", "editor && vim_mode", true)
	assertIsSuperset("editor", "mode == full && editor", true)
	assertIsSuperset("editor && mode == full", "editor", false)

	assertIsSuperset("editor", "something > editor", true)
	assertIsSuperset("editor", "editor > menu", false)

	assertIsSuperset("foo || bar || baz", "bar", true)
	assertIsSuperset("foo || bar || baz", "quux", false)
}

func assertEval(t *testing.T, predicate KeyBindingContextPredicate, contexts []KeyContext, want bool) {
	t.Helper()
	if got := predicate.Eval(contexts); got != want {
		t.Fatalf("(%v).Eval(%v) = %v, want %v", predicate, contexts, got, want)
	}
}

func TestContext_ChildOperator(t *testing.T) {
	predicate := mustParsePredicate(t, "parent > child")

	parentContext := mustParseContext(t, "parent")
	childContext := mustParseContext(t, "child")

	assertEval(t, predicate, []KeyContext{parentContext, childContext}, true)

	grandparentContext := mustParseContext(t, "grandparent")
	assertEval(t, predicate, []KeyContext{grandparentContext, parentContext, childContext}, true)

	otherContext := mustParseContext(t, "other")
	assertEval(t, predicate, []KeyContext{otherContext, childContext}, false)
	assertEval(t, predicate, []KeyContext{parentContext, otherContext, childContext}, true)

	assertEval(t, predicate, nil, false)
	assertEval(t, predicate, []KeyContext{childContext}, false)
	assertEval(t, predicate, []KeyContext{parentContext}, false)

	zanyPredicate := mustParsePredicate(t, "child > child")
	assertEval(t, zanyPredicate, []KeyContext{childContext}, false)
	assertEval(t, zanyPredicate, []KeyContext{childContext, childContext}, true)
}

func TestContext_NotOperator(t *testing.T) {
	notPredicate := mustParsePredicate(t, "!editor")
	editorContext := mustParseContext(t, "editor")
	workspaceContext := mustParseContext(t, "workspace")
	parentContext := mustParseContext(t, "parent")
	childContext := mustParseContext(t, "child")

	assertEval(t, notPredicate, []KeyContext{workspaceContext}, true)
	assertEval(t, notPredicate, []KeyContext{editorContext}, false)
	assertEval(t, notPredicate, []KeyContext{editorContext, workspaceContext}, false)
	assertEval(t, notPredicate, []KeyContext{workspaceContext, editorContext}, false)

	complexNot := mustParsePredicate(t, "!editor && workspace")
	assertEval(t, complexNot, []KeyContext{workspaceContext}, true)
	assertEval(t, complexNot, []KeyContext{editorContext, workspaceContext}, false)

	notModePredicate := mustParsePredicate(t, "!(mode == full)")
	var modeContext KeyContext
	modeContext.Set("mode", "full")
	assertEval(t, notModePredicate, []KeyContext{modeContext}, false)

	var otherModeContext KeyContext
	otherModeContext.Set("mode", "partial")
	assertEval(t, notModePredicate, []KeyContext{otherModeContext}, true)

	notDescendant := mustParsePredicate(t, "!(parent > child)")
	assertEval(t, notDescendant, []KeyContext{parentContext}, true)
	assertEval(t, notDescendant, []KeyContext{childContext}, true)
	assertEval(t, notDescendant, []KeyContext{parentContext, childContext}, false)

	notDescendant = mustParsePredicate(t, "parent > !child")
	assertEval(t, notDescendant, []KeyContext{parentContext}, false)
	assertEval(t, notDescendant, []KeyContext{childContext}, false)
	assertEval(t, notDescendant, []KeyContext{parentContext, childContext}, false)

	doubleNot := mustParsePredicate(t, "!!editor")
	assertEval(t, doubleNot, []KeyContext{editorContext}, true)
	assertEval(t, doubleNot, []KeyContext{workspaceContext}, false)

	workspaceContext = mustParseContext(t, "Workspace")
	paneContext := mustParseContext(t, "Pane")
	editorContext = mustParseContext(t, "Editor")

	workspacePaneEditor := []KeyContext{workspaceContext, paneContext, editorContext}

	assertEval(t, mustParsePredicate(t, "Pane > (Pane > Editor)"), workspacePaneEditor, false)
	assertEval(t, mustParsePredicate(t, "Workspace > Pane > Editor"), workspacePaneEditor, true)
	assertEval(t, mustParsePredicate(t, "(Pane > Pane) > Editor"), workspacePaneEditor, false)

	paneNotWorkspace := mustParsePredicate(t, "Pane > !Workspace")
	assertEval(t, paneNotWorkspace, []KeyContext{paneContext, editorContext}, true)
	assertEval(t, paneNotWorkspace, []KeyContext{paneContext, workspaceContext}, false)

	notWorkspace := mustParsePredicate(t, "!Workspace")
	assertEval(t, notWorkspace, []KeyContext{workspaceContext}, false)
	assertEval(t, notWorkspace, []KeyContext{paneContext}, true)
	assertEval(t, notWorkspace, []KeyContext{editorContext}, true)
	assertEval(t, notWorkspace, workspacePaneEditor, false)
}

func TestContext_ContextDisplay(t *testing.T) {
	ident, eq, notEq, and, or, descendant, not := predIdent, predEq, predNotEq, predAnd, predOr, predDescendant, predNot
	testCases := []struct {
		predicate KeyBindingContextPredicate
		expected  string
	}{
		{ident("a"), "a"},
		{eq("a", "b"), "a == b"},
		{notEq("a", "b"), "a != b"},
		{descendant(ident("a"), ident("b")), "a > b"},
		{not(ident("a")), "!a"},
		{notEq("a", "b"), "a != b"},
		{descendant(ident("a"), ident("b")), "a > b"},
		{not(and(ident("a"), ident("b"))), "!(a && b)"},
		{not(or(ident("a"), ident("b"))), "!(a || b)"},
		{and(ident("a"), ident("b")), "a && b"},
		{and(and(ident("a"), ident("b")), ident("c")), "a && b && c"},
		{or(ident("a"), ident("b")), "a || b"},
		{or(or(ident("a"), ident("b")), ident("c")), "a || b || c"},
		{or(ident("a"), and(ident("b"), ident("c"))), "a || (b && c)"},
		{and(and(and(ident("a"), eq("b", "c")), not(descendant(ident("d"), ident("e")))), eq("f", "g")), "a && b == c && !(d > e) && f == g"},
		{and(and(ident("a"), or(ident("b"), ident("c"))), ident("d")), "a && (b || c) && d"},
		{or(or(ident("a"), and(ident("b"), ident("c"))), ident("d")), "a || (b && c) || d"},
	}

	for _, c := range testCases {
		actual := c.predicate.String()
		if actual != c.expected {
			t.Fatalf("got %q, want %q", actual, c.expected)
		}
		parsed := mustParsePredicate(t, actual)
		if parsed != c.predicate {
			t.Fatalf("parse(%q) = %#v, want %#v", actual, parsed, c.predicate)
		}
	}
}
