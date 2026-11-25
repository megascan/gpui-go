package consumer

import (
	"slices"
	"testing"

	"github.com/megascan/gpui-go/a11y"
)

func (it *ChildIDs) collect() []NodeID {
	return slices.Collect(it.All())
}

func (it *Nodes) collect() []Node {
	return slices.Collect(it.All())
}

func localIDs(seq func(func(Node) bool)) []a11y.NodeID {
	var ids []a11y.NodeID
	for n := range seq {
		ids = append(ids, n.ID().localID)
	}
	return ids
}

func localIDsOf(seq func(func(NodeID) bool)) []a11y.NodeID {
	var ids []a11y.NodeID
	for id := range seq {
		ids = append(ids, id.localID)
	}
	return ids
}

func eqIDs(t *testing.T, want []a11y.NodeID, got []a11y.NodeID) {
	t.Helper()
	if !slices.Equal(want, got) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func byID(tree *Tree, id a11y.NodeID) Node {
	return must(tree.State().NodeByID(nid(id)))
}

func localOf(n Node, ok bool) (a11y.NodeID, bool) {
	return n.ID().localID, ok
}

func TestNode_ParentAndIndex(t *testing.T) {
	tree := testTree()
	_, _, ok := tree.State().Root().ParentAndIndex()
	check(t, !ok)
	p, i, ok := byID(tree, paragraph0ID).ParentAndIndex()
	check(t, ok && p.ID().localID == rootID && i == 0)
	p, i, ok = byID(tree, label00IgnoredID).ParentAndIndex()
	check(t, ok && p.ID().localID == paragraph0ID && i == 0)
	p, i, ok = byID(tree, paragraph1IgnoredID).ParentAndIndex()
	check(t, ok && p.ID().localID == rootID && i == 1)
}

func TestNode_DeepestFirstChild(t *testing.T) {
	tree := testTree()
	eq(t, label00IgnoredID, must(tree.State().Root().DeepestFirstChild()).ID().localID)
	eq(t, label00IgnoredID, must(byID(tree, paragraph0ID).DeepestFirstChild()).ID().localID)
	_, ok := byID(tree, label00IgnoredID).DeepestFirstChild()
	check(t, !ok)
}

func TestNode_FilteredParent(t *testing.T) {
	tree := testTree()
	eq(t, rootID, must(byID(tree, label11ID).FilteredParent(testTreeFilter)).ID().localID)
	_, ok := tree.State().Root().FilteredParent(testTreeFilter)
	check(t, !ok)
}

func TestNode_DeepestFirstFilteredChild(t *testing.T) {
	tree := testTree()
	eq(t, paragraph0ID, must(tree.State().Root().DeepestFirstFilteredChild(testTreeFilter)).ID().localID)
	_, ok := byID(tree, paragraph0ID).DeepestFirstFilteredChild(testTreeFilter)
	check(t, !ok)
	_, ok = byID(tree, label00IgnoredID).DeepestFirstFilteredChild(testTreeFilter)
	check(t, !ok)
}

func TestNode_DeepestLastChild(t *testing.T) {
	tree := testTree()
	eq(t, emptyContainer33IgnoredID, must(tree.State().Root().DeepestLastChild()).ID().localID)
	eq(t, emptyContainer33IgnoredID, must(byID(tree, paragraph3IgnoredID).DeepestLastChild()).ID().localID)
	_, ok := byID(tree, button32ID).DeepestLastChild()
	check(t, !ok)
}

func TestNode_DeepestLastFilteredChild(t *testing.T) {
	tree := testTree()
	eq(t, button32ID, must(tree.State().Root().DeepestLastFilteredChild(testTreeFilter)).ID().localID)
	eq(t, button32ID, must(byID(tree, paragraph3IgnoredID).DeepestLastFilteredChild(testTreeFilter)).ID().localID)
	_, ok := byID(tree, button32ID).DeepestLastFilteredChild(testTreeFilter)
	check(t, !ok)
	_, ok = byID(tree, paragraph0ID).DeepestLastFilteredChild(testTreeFilter)
	check(t, !ok)
}

func TestNode_IsDescendantOf(t *testing.T) {
	tree := testTree()
	root := tree.State().Root()
	check(t, byID(tree, paragraph0ID).IsDescendantOf(root))
	check(t, byID(tree, label00IgnoredID).IsDescendantOf(root))
	check(t, byID(tree, label00IgnoredID).IsDescendantOf(byID(tree, paragraph0ID)))
	check(t, !byID(tree, label00IgnoredID).IsDescendantOf(byID(tree, paragraph2ID)))
	check(t, !byID(tree, paragraph0ID).IsDescendantOf(byID(tree, paragraph2ID)))
}

func TestNode_IsRoot(t *testing.T) {
	tree := testTree()
	check(t, byID(tree, rootID).IsRoot())
	check(t, !byID(tree, paragraph0ID).IsRoot())
}

func TestNode_Locate(t *testing.T) {
	tree := testTree()
	l, tr := tree.State().Root().Locate()
	check(t, l == rootID && tr == a11y.TreeIDRoot())
	l, tr = byID(tree, paragraph0ID).Locate()
	check(t, l == paragraph0ID && tr == a11y.TreeIDRoot())
}

func TestNode_BoundingBox(t *testing.T) {
	tree := testTree()
	_, ok := byID(tree, rootID).BoundingBox()
	check(t, !ok)
	eq(t, a11y.Rect{X0: 10.0, Y0: 40.0, X1: 810.0, Y1: 80.0}, must(byID(tree, paragraph1IgnoredID).BoundingBox()))
	eq(t, a11y.Rect{X0: 20.0, Y0: 50.0, X1: 100.0, Y1: 70.0}, must(byID(tree, label11ID).BoundingBox()))
}

func TestNode_NodeAtPoint(t *testing.T) {
	tree := testTree()
	root := tree.State().Root()
	_, ok := root.NodeAtPoint(a11y.NewPoint(10.0, 40.0), testTreeFilter)
	check(t, !ok)
	eq(t, nid(label11ID), must(root.NodeAtPoint(a11y.NewPoint(20.0, 50.0), testTreeFilter)).ID())
	eq(t, nid(label11ID), must(root.NodeAtPoint(a11y.NewPoint(50.0, 60.0), testTreeFilter)).ID())
	_, ok = root.NodeAtPoint(a11y.NewPoint(100.0, 70.0), testTreeFilter)
	check(t, !ok)
}

func TestNode_FirstFilteredChild(t *testing.T) {
	tree := testTree()
	eq(t, paragraph0ID, must(localOf(tree.State().Root().firstFilteredChild(testTreeFilter))))
	eq(t, label310ID, must(localOf(byID(tree, paragraph3IgnoredID).firstFilteredChild(testTreeFilter))))
	_, ok := byID(tree, label20ID).firstFilteredChild(testTreeFilter)
	check(t, !ok)
}

func TestNode_LastFilteredChild(t *testing.T) {
	tree := testTree()
	eq(t, button32ID, must(localOf(byID(tree, paragraph3IgnoredID).lastFilteredChild(testTreeFilter))))
	_, ok := byID(tree, label20ID).lastFilteredChild(testTreeFilter)
	check(t, !ok)
}

func TestNode_Transform(t *testing.T) {
	tree := testTree()
	eq(t, a11y.AffineIdentity(), tree.State().Root().Transform())
	expected := a11y.AffineTranslate(a11y.Vec2{X: 10.0, Y: 40.0})
	eq(t, expected, byID(tree, paragraph1IgnoredID).Transform())
	eq(t, expected, byID(tree, label11ID).Transform())
}

func TestNode_RelativeTransform(t *testing.T) {
	tree := testTree()
	label := byID(tree, label11ID)
	paragraph := byID(tree, paragraph1IgnoredID)
	eq(t, a11y.AffineIdentity(), label.relativeTransform(paragraph))
	eq(t, a11y.AffineTranslate(a11y.Vec2{X: 10.0, Y: 40.0}), label.relativeTransform(tree.State().Root()))
}

func TestNode_NoLabelOrLabelledBy(t *testing.T) {
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, node(a11y.RoleButton)),
	), false)
	_, ok := byID(tree, 1).Label()
	check(t, !ok)
}

func TestNode_LabelFromLabelledBy(t *testing.T) {
	const label1 = "Check email every"
	const label2 = "minutes"
	checkBox := node(a11y.RoleCheckBox)
	checkBox.SetLabelledBy([]a11y.NodeID{2, 4})
	l1 := node(a11y.RoleLabel)
	l1.SetValue(label1)
	input := node(a11y.RoleTextInput)
	input.PushLabelledBy(4)
	l2 := node(a11y.RoleLabel)
	l2.SetValue(label2)
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1, 2, 3, 4)),
		ent(1, checkBox),
		ent(2, l1),
		ent(3, input),
		ent(4, l2),
	), false)
	eq(t, label1+" "+label2, must(byID(tree, 1).Label()))
	eq(t, label2, must(byID(tree, 3).Label()))
}

func TestNode_LabelFromDescendantLabel(t *testing.T) {
	type labelled struct {
		id, labelID a11y.NodeID
		role        a11y.Role
		labelRole   a11y.Role
		label       string
	}
	entries := []a11y.NodeEntry{ent(0, node(a11y.RoleWindow, 1, 3, 6, 8, 10, 12))}
	add := func(id a11y.NodeID, n a11y.Node) { entries = append(entries, ent(id, n)) }
	labelNode := func(role a11y.Role, text string) a11y.Node {
		n := node(role)
		if role == a11y.RoleLabel {
			n.SetValue(text)
		} else {
			n.SetLabel(text)
		}
		return n
	}
	cases := []labelled{
		{1, 2, a11y.RoleDefaultButton, a11y.RoleImage, "Play"},
		{6, 7, a11y.RoleCheckBox, a11y.RoleLabel, "Resume from previous position"},
		{8, 9, a11y.RoleRadioButton, a11y.RoleLabel, "Normal speed"},
		{10, 11, a11y.RoleButton, a11y.RoleLabel, "More"},
		{13, 14, a11y.RoleMenuItem, a11y.RoleLabel, "Share"},
		{15, 16, a11y.RoleMenuItemCheckBox, a11y.RoleLabel, "Apply volume processing"},
		{17, 18, a11y.RoleMenuItemRadio, a11y.RoleLabel, "Maximize loudness for noisy environment"},
	}
	for _, c := range cases {
		add(c.id, node(c.role, c.labelID))
		add(c.labelID, labelNode(c.labelRole, c.label))
	}
	const linkLabel = "Watch in browser"
	add(3, node(a11y.RoleLink, 4))
	add(4, node(a11y.RoleGenericContainer, 5))
	add(5, labelNode(a11y.RoleLabel, linkLabel))
	add(12, node(a11y.RoleMenu, 13, 15, 17))
	tree := NewTree(rootUpdate(0, entries...), false)
	for _, c := range cases {
		eq(t, c.label, must(byID(tree, c.id).Label()))
	}
	eq(t, linkLabel, must(byID(tree, 3).Label()))
}

func textInputTree(selectionIndex int, runValue string, runRight float64, lengths []uint8, positions, widths []float32, wordStarts []uint8) *Tree {
	input := node(a11y.RoleMultilineTextInput, 2)
	input.SetBounds(a11y.Rect{X0: 8.0, Y0: 8.0, X1: 296.0, Y1: 69.5})
	input.SetPlaceholder("John Doe")
	pos := a11y.TextPosition{Node: 2, CharacterIndex: selectionIndex}
	input.SetTextSelection(a11y.TextSelection{Anchor: pos, Focus: pos})
	input.AddAction(a11y.ActionFocus)
	run := node(a11y.RoleTextRun)
	run.SetBounds(a11y.Rect{X0: 12.0, Y0: 10.0, X1: runRight, Y1: 24.0})
	run.SetValue(runValue)
	run.SetCharacterLengths(lengths)
	run.SetCharacterPositions(positions)
	run.SetCharacterWidths(widths)
	if wordStarts != nil {
		run.SetWordStarts(wordStarts)
	}
	run.SetTextDirection(a11y.TextDirectionLeftToRight)
	return NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, input),
		ent(2, run),
	), false)
}

func TestNode_PlaceholderShouldBeExposedOnEmptyTextInput(t *testing.T) {
	tree := textInputTree(0, "", 12.0, []uint8{}, []float32{}, []float32{}, nil)
	eq(t, "John Doe", must(byID(tree, 1).Placeholder()))
}

func TestNode_PlaceholderShouldBeIgnoredOnNonEmptyTextInput(t *testing.T) {
	tree := textInputTree(1, "A", 20.0, []uint8{1}, []float32{0.0}, []float32{8.0}, []uint8{0})
	_, ok := byID(tree, 1).Placeholder()
	check(t, !ok)
}

func TestNode_HiddenFlagShouldBeInherited(t *testing.T) {
	container := node(a11y.RoleGenericContainer)
	container.SetHidden()
	container.PushChild(2)
	leaf := node(a11y.RoleButton)
	leaf.SetLabel("OK")
	tree := NewTree(rootUpdate(0,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, container),
		ent(2, leaf),
	), false)
	check(t, byID(tree, 2).IsHidden())
}

func TestNode_NodeId_NewAndToComponentsRoundTrip(t *testing.T) {
	id := newNodeID(42, treeIndex(7))
	l, ti := id.toComponents()
	check(t, l == 42 && ti == 7)
}

func TestNode_NodeId_WithSameTreePreservesTreeIndex(t *testing.T) {
	id := newNodeID(100, treeIndex(5))
	l, ti := id.withSameTree(200).toComponents()
	check(t, l == 200 && ti == 5)
}

func TestNode_NodeId_IntoU128(t *testing.T) {
	id := newNodeID(12345, treeIndex(67))
	l, ti := id.toComponents()
	check(t, l == 12345 && ti == 67)
	hi, lo := id.ToU128()
	check(t, hi == 12345 && lo == 67)
}

func TestNode_NodeId_Equality(t *testing.T) {
	id1 := newNodeID(1, treeIndex(2))
	id2 := newNodeID(1, treeIndex(2))
	id3 := newNodeID(1, treeIndex(3))
	id4 := newNodeID(2, treeIndex(2))
	check(t, id1 == id2)
	check(t, id1 != id3)
	check(t, id1 != id4)
}

func TestNode_IsFocusedWhenNodeHasFocus(t *testing.T) {
	tree := NewTree(rootUpdate(1, ent(0, node(a11y.RoleWindow, 1)), ent(1, node(a11y.RoleButton))), true)
	check(t, byID(tree, 1).IsFocused())
}

func TestNode_IsFocusedWhenNodeDoesNotHaveFocus(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, node(a11y.RoleButton))), true)
	check(t, !byID(tree, 1).IsFocused())
}

func activeDescendantTree() *Tree {
	listBox := node(a11y.RoleListBox, 2)
	listBox.SetActiveDescendant(2)
	return NewTree(rootUpdate(1,
		ent(0, node(a11y.RoleWindow, 1)),
		ent(1, listBox),
		ent(2, node(a11y.RoleListBoxOption)),
	), true)
}

func TestNode_IsFocusedActiveDescendantIsFocused(t *testing.T) {
	check(t, byID(activeDescendantTree(), 2).IsFocused())
}

func TestNode_IsFocusedNodeWithActiveDescendantIsNotFocused(t *testing.T) {
	check(t, !byID(activeDescendantTree(), 1).IsFocused())
}
