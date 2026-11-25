package consumer

import (
	"testing"

	"github.com/megascan/gpui-go/a11y"
)

func TestIterators_FollowingSiblings(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().FollowingSiblings().Next()
	check(t, !ok)
	eq(t, 0, tree.State().Root().FollowingSiblings().Len())
	eqIDs(t, []a11y.NodeID{paragraph1IgnoredID, paragraph2ID, paragraph3IgnoredID}, localIDsOf(byID(tree, paragraph0ID).FollowingSiblingIDs().All()))
	eq(t, 3, byID(tree, paragraph0ID).FollowingSiblings().Len())
	_, ok = byID(tree, paragraph3IgnoredID).FollowingSiblings().Next()
	check(t, !ok)
	eq(t, 0, byID(tree, paragraph3IgnoredID).FollowingSiblings().Len())
}

func TestIterators_FollowingSiblingsReversed(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().FollowingSiblings().NextBack()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{paragraph3IgnoredID, paragraph2ID, paragraph1IgnoredID}, localIDsOf(byID(tree, paragraph0ID).FollowingSiblingIDs().Backward()))
	_, ok = byID(tree, paragraph3IgnoredID).FollowingSiblings().NextBack()
	check(t, !ok)
}

func TestIterators_PrecedingSiblings(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().PrecedingSiblings().Next()
	check(t, !ok)
	eq(t, 0, tree.State().Root().PrecedingSiblings().Len())
	eqIDs(t, []a11y.NodeID{paragraph2ID, paragraph1IgnoredID, paragraph0ID}, localIDsOf(byID(tree, paragraph3IgnoredID).PrecedingSiblingIDs().All()))
	eq(t, 3, byID(tree, paragraph3IgnoredID).PrecedingSiblings().Len())
	_, ok = byID(tree, paragraph0ID).PrecedingSiblings().Next()
	check(t, !ok)
	eq(t, 0, byID(tree, paragraph0ID).PrecedingSiblings().Len())
}

func TestIterators_PrecedingSiblingsReversed(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().PrecedingSiblings().NextBack()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{paragraph0ID, paragraph1IgnoredID, paragraph2ID}, localIDsOf(byID(tree, paragraph3IgnoredID).PrecedingSiblingIDs().Backward()))
	_, ok = byID(tree, paragraph0ID).PrecedingSiblings().NextBack()
	check(t, !ok)
}

func TestIterators_FollowingFilteredSiblings(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().FollowingFilteredSiblings(testTreeFilter).Next()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{label11ID, paragraph2ID, label310ID, button32ID}, localIDs(byID(tree, paragraph0ID).FollowingFilteredSiblings(testTreeFilter).All()))
	eqIDs(t, []a11y.NodeID{button32ID}, localIDs(byID(tree, label310ID).FollowingFilteredSiblings(testTreeFilter).All()))
	_, ok = byID(tree, paragraph3IgnoredID).FollowingFilteredSiblings(testTreeFilter).Next()
	check(t, !ok)
}

func TestIterators_FollowingFilteredSiblingsReversed(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().FollowingFilteredSiblings(testTreeFilter).NextBack()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{button32ID, label310ID, paragraph2ID, label11ID}, localIDs(byID(tree, paragraph0ID).FollowingFilteredSiblings(testTreeFilter).Backward()))
	eqIDs(t, []a11y.NodeID{button32ID}, localIDs(byID(tree, label310ID).FollowingFilteredSiblings(testTreeFilter).Backward()))
	_, ok = byID(tree, paragraph3IgnoredID).FollowingFilteredSiblings(testTreeFilter).NextBack()
	check(t, !ok)
}

func TestIterators_PrecedingFilteredSiblings(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().PrecedingFilteredSiblings(testTreeFilter).Next()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{paragraph2ID, label11ID, paragraph0ID}, localIDs(byID(tree, paragraph3IgnoredID).PrecedingFilteredSiblings(testTreeFilter).All()))
	eqIDs(t, []a11y.NodeID{paragraph2ID, label11ID, paragraph0ID}, localIDs(byID(tree, label310ID).PrecedingFilteredSiblings(testTreeFilter).All()))
	_, ok = byID(tree, paragraph0ID).PrecedingFilteredSiblings(testTreeFilter).Next()
	check(t, !ok)
}

func TestIterators_PrecedingFilteredSiblingsReversed(t *testing.T) {
	tree := testTree()
	_, ok := tree.State().Root().PrecedingFilteredSiblings(testTreeFilter).NextBack()
	check(t, !ok)
	eqIDs(t, []a11y.NodeID{paragraph0ID, label11ID, paragraph2ID}, localIDs(byID(tree, paragraph3IgnoredID).PrecedingFilteredSiblings(testTreeFilter).Backward()))
	eqIDs(t, []a11y.NodeID{paragraph0ID, label11ID, paragraph2ID}, localIDs(byID(tree, label310ID).PrecedingFilteredSiblings(testTreeFilter).Backward()))
	_, ok = byID(tree, paragraph0ID).PrecedingFilteredSiblings(testTreeFilter).NextBack()
	check(t, !ok)
}

func TestIterators_FilteredChildren(t *testing.T) {
	tree := testTree()
	eqIDs(t, []a11y.NodeID{paragraph0ID, label11ID, paragraph2ID, label310ID, button32ID}, localIDs(tree.State().Root().FilteredChildren(testTreeFilter).All()))
	_, ok := byID(tree, paragraph0ID).FilteredChildren(testTreeFilter).Next()
	check(t, !ok)
	_, ok = byID(tree, label00IgnoredID).FilteredChildren(testTreeFilter).Next()
	check(t, !ok)
}

func TestIterators_FilteredChildrenReversed(t *testing.T) {
	tree := testTree()
	eqIDs(t, []a11y.NodeID{button32ID, label310ID, paragraph2ID, label11ID, paragraph0ID}, localIDs(tree.State().Root().FilteredChildren(testTreeFilter).Backward()))
	_, ok := byID(tree, paragraph0ID).FilteredChildren(testTreeFilter).NextBack()
	check(t, !ok)
	_, ok = byID(tree, label00IgnoredID).FilteredChildren(testTreeFilter).NextBack()
	check(t, !ok)
}

func TestIterators_GraftNodeWithoutSubtreeHasNoFilteredChildren(t *testing.T) {
	subtreeID := treeIDFromU128(1)
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, graftNode(subtreeID))), false)
	graft := must(tree.State().NodeByID(newNodeID(1, treeIndex(0))))
	_, ok := graft.FilteredChildren(CommonFilter).Next()
	check(t, !ok)
}

func TestIterators_FilteredChildrenCrossesSubtreeBoundary(t *testing.T) {
	subtreeID := treeIDFromU128(1)
	tree := NewTree(rootUpdate(0, ent(0, node(a11y.RoleWindow, 1)), ent(1, graftNode(subtreeID))), false)
	tree.UpdateAndProcessChanges(treeUpdate(subtreeID, someTree(0), 0,
		ent(0, node(a11y.RoleDocument, 1)),
		ent(1, node(a11y.RoleButton)),
	), &testHandler{})
	root := tree.State().Root()
	var filtered []Node
	for n := range root.FilteredChildren(CommonFilter).All() {
		filtered = append(filtered, n)
	}
	eq(t, 1, len(filtered))
	eq(t, newNodeID(0, treeIndex(1)), filtered[0].ID())
	var docChildren []Node
	for n := range filtered[0].FilteredChildren(CommonFilter).All() {
		docChildren = append(docChildren, n)
	}
	eq(t, 1, len(docChildren))
	eq(t, newNodeID(1, treeIndex(1)), docChildren[0].ID())
}
