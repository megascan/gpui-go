package consumer

import (
	"testing"

	"github.com/megascan/gpui-go/a11y"
)

func assertFilterResult(t *testing.T, expected FilterResult, tree *Tree, id a11y.NodeID) {
	t.Helper()
	n := byID(tree, id)
	eq(t, expected, CommonFilter(&n))
}

func hiddenNode(role a11y.Role, children ...a11y.NodeID) a11y.Node {
	n := node(role, children...)
	n.SetHidden()
	return n
}

func TestFilters_Normal(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, node(a11y.RoleButton))), false)
	assertFilterResult(t, FilterResultInclude, tree, 1)
}

func TestFilters_Hidden(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, hiddenNode(a11y.RoleButton))), false)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 1)
}

func TestFilters_HiddenButFocused(t *testing.T) {
	tree := NewTree(rootUpdate(1, ent(0, node(a11y.RoleWindow, 1)), ent(1, hiddenNode(a11y.RoleButton))), true)
	assertFilterResult(t, FilterResultInclude, tree, 1)
}

func TestFilters_GenericContainer(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleGenericContainer, 1)), ent(1, node(a11y.RoleButton))), false)
	assertFilterResult(t, FilterResultExcludeNode, tree, 0)
	root := byID(tree, 0)
	eq(t, FilterResultInclude, CommonFilterWithRootException(&root))
	assertFilterResult(t, FilterResultInclude, tree, 1)
}

func TestFilters_HiddenParent(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, hiddenNode(a11y.RoleGenericContainer, 1)), ent(1, node(a11y.RoleButton))), false)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 0)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 1)
}

func TestFilters_HiddenParentButFocused(t *testing.T) {
	tree := NewTree(rootUpdate(1, ent(0, hiddenNode(a11y.RoleGenericContainer, 1)), ent(1, node(a11y.RoleButton))), true)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 0)
	assertFilterResult(t, FilterResultInclude, tree, 1)
}

func TestFilters_TextRun(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleTextInput, 1)), ent(1, node(a11y.RoleTextRun))), false)
	assertFilterResult(t, FilterResultExcludeNode, tree, 1)
}

func clippedChildrenTestTree() *Tree {
	root := node(a11y.RoleScrollView, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	root.SetClipsChildren()
	root.SetBounds(a11y.NewRect(0.0, 0.0, 30.0, 30.0))
	entries := []a11y.NodeEntry{ent(0, root)}
	bounded := func(y0, y1 float64) a11y.Node {
		n := node(a11y.RoleUnknown)
		n.SetBounds(a11y.NewRect(0.0, y0, 30.0, y1))
		return n
	}
	entries = append(entries,
		ent(1, bounded(-30.0, -20.0)),
		ent(2, bounded(-20.0, -10.0)),
		ent(3, bounded(-10.0, 0.0)),
		ent(4, hiddenNode(a11y.RoleUnknown)),
		ent(5, bounded(0.0, 10.0)),
		ent(6, bounded(10.0, 20.0)),
		ent(7, bounded(20.0, 30.0)),
		ent(8, hiddenNode(a11y.RoleUnknown)),
		ent(9, bounded(30.0, 40.0)),
		ent(10, bounded(40.0, 50.0)),
		ent(11, bounded(50.0, 60.0)),
	)
	return NewTree(rootUpdate(0, entries...), false)
}

func TestFilters_ClippedChildrenExcludedAbove(t *testing.T) {
	tree := clippedChildrenTestTree()
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 1)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 2)
}

func TestFilters_ClippedChildrenIncludedAbove(t *testing.T) {
	assertFilterResult(t, FilterResultInclude, clippedChildrenTestTree(), 3)
}

func TestFilters_ClippedChildrenHidden(t *testing.T) {
	tree := clippedChildrenTestTree()
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 4)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 8)
}

func TestFilters_ClippedChildrenVisible(t *testing.T) {
	tree := clippedChildrenTestTree()
	assertFilterResult(t, FilterResultInclude, tree, 5)
	assertFilterResult(t, FilterResultInclude, tree, 6)
	assertFilterResult(t, FilterResultInclude, tree, 7)
}

func TestFilters_ClippedChildrenIncludedBelow(t *testing.T) {
	assertFilterResult(t, FilterResultInclude, clippedChildrenTestTree(), 9)
}

func TestFilters_ClippedChildrenExcludedBelow(t *testing.T) {
	tree := clippedChildrenTestTree()
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 10)
	assertFilterResult(t, FilterResultExcludeSubtree, tree, 11)
}

func TestFilters_GraftNode(t *testing.T) {
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, graftNode(treeIDFromU128(1)))), false)
	assertFilterResult(t, FilterResultExcludeNode, tree, 1)
}
