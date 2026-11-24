package taffy

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"unsafe"
)

func defaultStyle() Style[defaultCheapStr] {
	return StyleDefault[defaultCheapStr]()
}

func sizeMeasureFunction(inputs LayoutInput, _ NodeID, nodeContext *Size[float32], style *Style[defaultCheapStr]) LayoutOutput {
	return ComputeLeafLayout(
		inputs,
		style.Ref(),
		func(unsafe.Pointer, float32) float32 { return 0.0 },
		func(knownDimensions Size[Option[float32]], _ Size[AvailableSpace]) Size[float32] {
			ctx := SizeF32Zero
			if nodeContext != nil {
				ctx = *nodeContext
			}
			return SizeOptionUnwrapOr(knownDimensions, ctx)
		},
	)
}

func mustNode(t *testing.T, id NodeID, err error) NodeID {
	t.Helper()
	if err != nil {
		t.Fatalf("called `Result::unwrap()` on an `Err` value: %v", err)
	}
	return id
}

func unwrapResult[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	return v
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("called `Result::unwrap()` on an `Err` value: %v", err)
	}
}

func leaf(t *testing.T, taffy *TaffyTree[struct{}]) NodeID {
	t.Helper()
	id, err := taffy.NewLeaf(defaultStyle())
	return mustNode(t, id, err)
}

func withChildren(t *testing.T, taffy *TaffyTree[struct{}], children ...NodeID) NodeID {
	t.Helper()
	id, err := taffy.NewWithChildren(defaultStyle(), children)
	return mustNode(t, id, err)
}

func children(t *testing.T, taffy *TaffyTree[struct{}], node NodeID) []NodeID {
	t.Helper()
	c, err := taffy.Children(node)
	mustOK(t, err)
	return c
}

func dirty(t *testing.T, taffy *TaffyTree[struct{}], node NodeID) bool {
	t.Helper()
	d, err := taffy.Dirty(node)
	mustOK(t, err)
	return d
}

func layoutOf[C any](t *testing.T, taffy *TaffyTree[C], node NodeID) *Layout {
	t.Helper()
	l, err := taffy.Layout(node)
	mustOK(t, err)
	return l
}

func TestTreeTaffyTree_NewShouldAllocateDefaultCapacity(t *testing.T) {
	const defaultCapacity = 16
	taffy := NewTaffyTree[struct{}]()

	if taffy.children.capacity() < defaultCapacity || taffy.parents.capacity() < defaultCapacity || taffy.nodes.capacity() < defaultCapacity {
		t.Fatal("capacity below default")
	}
}

func TestTreeTaffyTree_WithCapacity(t *testing.T) {
	const capacity = 8
	taffy := TaffyTreeWithCapacity[struct{}](capacity)

	if taffy.children.capacity() < capacity || taffy.parents.capacity() < capacity || taffy.nodes.capacity() < capacity {
		t.Fatal("capacity below requested")
	}
}

func TestTreeTaffyTree_NewLeaf(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	node, err := taffy.NewLeaf(defaultStyle())
	if err != nil {
		t.Fatal(err)
	}

	if taffy.ChildCount(node) != 0 {
		t.Fatal("leaf has children")
	}
}

func TestTreeTaffyTree_NewLeafWithContext(t *testing.T) {
	taffy := NewTaffyTree[Size[float32]]()

	node, err := taffy.NewLeafWithContext(defaultStyle(), SizeF32Zero)
	if err != nil {
		t.Fatal(err)
	}

	if taffy.ChildCount(node) != 0 {
		t.Fatal("leaf has children")
	}
}

func TestTreeTaffyTree_NewWithChildren(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	if taffy.ChildCount(node) != 2 {
		t.Fatalf("child count = %d", taffy.ChildCount(node))
	}
	if children(t, taffy, node)[0] != child0 || children(t, taffy, node)[1] != child1 {
		t.Fatal("children mismatch")
	}
}

func TestTreeTaffyTree_RemoveNodeShouldRemove(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	node := leaf(t, taffy)

	_, err := taffy.Remove(node)
	mustOK(t, err)
}

func TestTreeTaffyTree_RemoveNodeShouldDetachHierarchy(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	node2 := leaf(t, taffy)
	node1 := withChildren(t, taffy, node2)
	node0 := withChildren(t, taffy, node1)

	if !slices.Equal(children(t, taffy, node0), []NodeID{node1}) || !slices.Equal(children(t, taffy, node1), []NodeID{node2}) {
		t.Fatal("hierarchy mismatch")
	}

	_, err := taffy.Remove(node1)
	mustOK(t, err)

	if len(children(t, taffy, node0)) != 0 || len(children(t, taffy, node2)) != 0 {
		t.Fatal("hierarchy not detached")
	}
}

func TestTreeTaffyTree_RemoveLastNode(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	parent := leaf(t, taffy)
	child := leaf(t, taffy)
	mustOK(t, taffy.AddChild(parent, child))

	_, err := taffy.Remove(child)
	mustOK(t, err)
	_, err = taffy.Remove(parent)
	mustOK(t, err)
}

func TestTreeTaffyTree_RemoveNodeMarksParentDirty(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	childStyle := defaultStyle()
	childStyle.Border = RectZero[LengthPercentage]()
	childStyle.Border.Bottom = Length[LengthPercentage](1)
	child := unwrapResult(taffy.NewLeaf(childStyle))
	root := withChildren(t, taffy, child)

	mustOK(t, taffy.ComputeLayout(root, SizeMinContent[AvailableSpace]()))
	if got := layoutOf(t, taffy, root).Size.Height; got != 1 {
		t.Fatalf("height = %v, want 1", got)
	}

	_, err := taffy.Remove(child)
	mustOK(t, err)
	if !dirty(t, taffy, root) {
		t.Fatal("root not dirty")
	}

	mustOK(t, taffy.ComputeLayout(root, SizeMinContent[AvailableSpace]()))
	if got := layoutOf(t, taffy, root).Size.Height; got != 0 {
		t.Fatalf("height = %v, want 0", got)
	}
}

func TestTreeTaffyTree_SetMeasure(t *testing.T) {
	taffy := NewTaffyTree[Size[float32]]()
	node := unwrapResult(taffy.NewLeafWithContext(defaultStyle(), Size[float32]{Width: 200.0, Height: 200.0}))
	mustOK(t, taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), sizeMeasureFunction))
	if got := layoutOf(t, taffy, node).Size.Width; got != 200.0 {
		t.Fatalf("width = %v, want 200", got)
	}

	mustOK(t, taffy.SetNodeContext(node, some(Size[float32]{Width: 100.0, Height: 100.0})))
	mustOK(t, taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), sizeMeasureFunction))
	if got := layoutOf(t, taffy, node).Size.Width; got != 100.0 {
		t.Fatalf("width = %v, want 100", got)
	}
}

func TestTreeTaffyTree_SetMeasureOfPreviouslyUnmeasuredNode(t *testing.T) {
	taffy := NewTaffyTree[Size[float32]]()
	node := unwrapResult(taffy.NewLeaf(defaultStyle()))
	mustOK(t, taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), sizeMeasureFunction))
	if got := layoutOf(t, taffy, node).Size.Width; got != 0.0 {
		t.Fatalf("width = %v, want 0", got)
	}

	mustOK(t, taffy.SetNodeContext(node, some(Size[float32]{Width: 100.0, Height: 100.0})))
	mustOK(t, taffy.ComputeLayoutWithMeasure(node, SizeMaxContent[AvailableSpace](), sizeMeasureFunction))
	if got := layoutOf(t, taffy, node).Size.Width; got != 100.0 {
		t.Fatalf("width = %v, want 100", got)
	}
}

func TestTreeTaffyTree_AddChild(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	node := leaf(t, taffy)
	if taffy.ChildCount(node) != 0 {
		t.Fatal("want 0")
	}

	child0 := leaf(t, taffy)
	mustOK(t, taffy.AddChild(node, child0))
	if taffy.ChildCount(node) != 1 {
		t.Fatal("want 1")
	}

	child1 := leaf(t, taffy)
	mustOK(t, taffy.AddChild(node, child1))
	if taffy.ChildCount(node) != 2 {
		t.Fatal("want 2")
	}
}

func TestTreeTaffyTree_InsertChildAtIndex(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	child2 := leaf(t, taffy)

	node := leaf(t, taffy)
	if taffy.ChildCount(node) != 0 {
		t.Fatal("want 0")
	}

	mustOK(t, taffy.InsertChildAtIndex(node, 0, child0))
	if !slices.Equal(children(t, taffy, node), []NodeID{child0}) {
		t.Fatal("after first insert")
	}

	mustOK(t, taffy.InsertChildAtIndex(node, 0, child1))
	if !slices.Equal(children(t, taffy, node), []NodeID{child1, child0}) {
		t.Fatal("after second insert")
	}

	mustOK(t, taffy.InsertChildAtIndex(node, 1, child2))
	if !slices.Equal(children(t, taffy, node), []NodeID{child1, child2, child0}) {
		t.Fatal("after third insert")
	}
}

func TestTreeTaffyTree_SetChildren(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	if !slices.Equal(children(t, taffy, node), []NodeID{child0, child1}) {
		t.Fatal("initial children")
	}

	child2 := leaf(t, taffy)
	child3 := leaf(t, taffy)
	mustOK(t, taffy.SetChildren(node, []NodeID{child2, child3}))

	if taffy.ChildCount(node) != 2 || !slices.Equal(children(t, taffy, node), []NodeID{child2, child3}) {
		t.Fatal("children after set")
	}
}

func TestTreeTaffyTree_RemoveChild(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	if taffy.ChildCount(node) != 2 {
		t.Fatal("want 2")
	}

	_, err := taffy.RemoveChild(node, child0)
	mustOK(t, err)
	if taffy.ChildCount(node) != 1 || children(t, taffy, node)[0] != child1 {
		t.Fatal("after first remove")
	}

	_, err = taffy.RemoveChild(node, child1)
	mustOK(t, err)
	if taffy.ChildCount(node) != 0 {
		t.Fatal("want 0")
	}
}

func TestTreeTaffyTree_RemoveChildAtIndex(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	if taffy.ChildCount(node) != 2 {
		t.Fatal("want 2")
	}

	_, err := taffy.RemoveChildAtIndex(node, 0)
	mustOK(t, err)
	if taffy.ChildCount(node) != 1 || children(t, taffy, node)[0] != child1 {
		t.Fatal("after first remove")
	}

	_, err = taffy.RemoveChildAtIndex(node, 0)
	mustOK(t, err)
	if taffy.ChildCount(node) != 0 {
		t.Fatal("want 0")
	}
}

func TestTreeTaffyTree_RemoveChildrenRange(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	child2 := leaf(t, taffy)
	child3 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1, child2, child3)

	if taffy.ChildCount(node) != 4 {
		t.Fatal("want 4")
	}

	mustOK(t, taffy.RemoveChildrenRange(node, 1, 3))
	if taffy.ChildCount(node) != 2 || !slices.Equal(children(t, taffy, node), []NodeID{child0, child3}) {
		t.Fatal("children after range removal")
	}
	for _, child := range []NodeID{child0, child3} {
		if taffy.Parent(child) != some(node) {
			t.Fatalf("parent of %v", child)
		}
	}
	for _, child := range []NodeID{child1, child2} {
		if taffy.Parent(child) != none[NodeID]() {
			t.Fatalf("parent of %v", child)
		}
	}
}

func TestTreeTaffyTree_RemoveChildUpdatesParents(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	parent := leaf(t, taffy)
	child := leaf(t, taffy)

	mustOK(t, taffy.AddChild(parent, child))

	_, err := taffy.Remove(parent)
	mustOK(t, err)

	if err := taffy.SetChildren(child, nil); err != nil {
		t.Fatal(err)
	}
}

func TestTreeTaffyTree_ReplaceChildAtIndex(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)

	node := withChildren(t, taffy, child0)
	if taffy.ChildCount(node) != 1 || children(t, taffy, node)[0] != child0 {
		t.Fatal("initial")
	}

	_, err := taffy.ReplaceChildAtIndex(node, 0, child1)
	mustOK(t, err)
	if taffy.ChildCount(node) != 1 || children(t, taffy, node)[0] != child1 {
		t.Fatal("after replace")
	}
}

func TestTreeTaffyTree_ChildAtIndex(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	child2 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1, child2)

	for i, want := range []NodeID{child0, child1, child2} {
		if got, err := taffy.ChildAtIndex(node, i); err != nil || got != want {
			t.Fatalf("child %d", i)
		}
	}
}

func TestTreeTaffyTree_ChildCount(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	if taffy.ChildCount(node) != 2 || taffy.ChildCount(child0) != 0 || taffy.ChildCount(child1) != 0 {
		t.Fatal("child counts")
	}
}

func TestTreeTaffyTree_Children(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	var want []NodeID
	want = append(want, child0)
	want = append(want, child1)

	if !slices.Equal(children(t, taffy, node), want) {
		t.Fatal("children mismatch")
	}

	if len(children(t, taffy, child0)) != 0 {
		t.Fatal("leaf has children")
	}
}

func TestTreeTaffyTree_SetStyle(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	node := leaf(t, taffy)
	style, err := taffy.Style(node)
	mustOK(t, err)
	if style.Display != DisplayFlex {
		t.Fatal("default display")
	}

	newStyle := defaultStyle()
	newStyle.Display = DisplayNone
	mustOK(t, taffy.SetStyle(node, newStyle))
	style, err = taffy.Style(node)
	mustOK(t, err)
	if style.Display != DisplayNone {
		t.Fatal("display after set")
	}
}

func TestTreeTaffyTree_Style(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	style := defaultStyle()
	style.Display = DisplayNone
	style.FlexDirection = FlexDirectionRowReverse

	node := unwrapResult(taffy.NewLeaf(style))

	res, err := taffy.Style(node)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*res, style) {
		t.Fatal("style mismatch")
	}
}

func TestTreeTaffyTree_Layout(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	node := leaf(t, taffy)

	if _, err := taffy.Layout(node); err != nil {
		t.Fatal(err)
	}
}

func TestTreeTaffyTree_MarkDirty(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child0 := leaf(t, taffy)
	child1 := leaf(t, taffy)
	node := withChildren(t, taffy, child0, child1)

	mustOK(t, taffy.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))

	if dirty(t, taffy, child0) || dirty(t, taffy, child1) || dirty(t, taffy, node) {
		t.Fatal("dirty after layout")
	}

	mustOK(t, taffy.MarkDirty(node))
	if dirty(t, taffy, child0) || dirty(t, taffy, child1) || !dirty(t, taffy, node) {
		t.Fatal("after marking node")
	}

	mustOK(t, taffy.ComputeLayout(node, SizeMaxContent[AvailableSpace]()))
	mustOK(t, taffy.MarkDirty(child0))
	if !dirty(t, taffy, child0) || dirty(t, taffy, child1) || !dirty(t, taffy, node) {
		t.Fatal("after marking child0")
	}
}

func TestTreeTaffyTree_ComputeLayoutShouldProduceValidResult(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	style := defaultStyle()
	style.Size = Size[Dimension]{Width: DimensionLength(10), Height: DimensionLength(10)}
	node, err := taffy.NewLeaf(style)
	if err != nil {
		t.Fatal(err)
	}
	if err := taffy.ComputeLayout(node, Size[AvailableSpace]{Width: AvailableSpaceDefinite(100), Height: AvailableSpaceDefinite(100)}); err != nil {
		t.Fatal(err)
	}
}

func TestTreeTaffyTree_MakeSureLayoutLocationIsTopLeft(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	nodeStyle := defaultStyle()
	nodeStyle.Size = Size[Dimension]{Width: DimensionPercent(1), Height: DimensionPercent(1)}
	node := unwrapResult(taffy.NewLeaf(nodeStyle))

	rootStyle := defaultStyle()
	rootStyle.Size = Size[Dimension]{Width: DimensionLength(100), Height: DimensionLength(100)}
	rootStyle.Padding = Rect[LengthPercentage]{
		Left:   Length[LengthPercentage](10),
		Right:  Length[LengthPercentage](20),
		Top:    Length[LengthPercentage](30),
		Bottom: Length[LengthPercentage](40),
	}
	root := unwrapResult(taffy.NewWithChildren(rootStyle, []NodeID{node}))

	mustOK(t, taffy.ComputeLayout(root, SizeMaxContent[AvailableSpace]()))

	layout := layoutOf(t, taffy, node)
	if layout.Location.X != 10 || layout.Location.Y != 30 {
		t.Fatalf("location = %v", layout.Location)
	}
}

func TestTreeTaffyTree_SetChildrenReparents(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	child := leaf(t, taffy)
	oldParent := withChildren(t, taffy, child)

	newParent := leaf(t, taffy)
	mustOK(t, taffy.SetChildren(newParent, []NodeID{child}))

	if len(children(t, taffy, oldParent)) != 0 {
		t.Fatal("old parent keeps child")
	}
}

type dropCounted struct {
	id int
}

func TestTreeTaffyTree_RemoveDropsTheNodesContext(t *testing.T) {
	taffy := NewTaffyTree[dropCounted]()
	node := unwrapResult(taffy.NewLeafWithContext(defaultStyle(), dropCounted{id: 1}))

	_, err := taffy.Remove(node)
	mustOK(t, err)

	if taffy.nodeContextData.len() != 0 || taffy.GetNodeContext(node) != nil {
		t.Fatal("removing a node must drop its context rather than strand it in node_context_data")
	}
}

func TestTreeTaffyTree_ClearDropsEveryNodesContext(t *testing.T) {
	taffy := NewTaffyTree[dropCounted]()
	var nodes []NodeID
	for i := range 8 {
		nodes = append(nodes, unwrapResult(taffy.NewLeafWithContext(defaultStyle(), dropCounted{id: i})))
	}

	taffy.Clear()

	if taffy.nodeContextData.len() != 0 {
		t.Fatal("clear() drops all nodes, so it must drop their contexts too")
	}
	for _, node := range nodes {
		if taffy.GetNodeContext(node) != nil {
			t.Fatal("clear() drops all nodes, so it must drop their contexts too")
		}
	}
}

func TestTreeTaffyTree_NodeIDMatchesSlotMapKeyData(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	a := leaf(t, taffy)
	b := leaf(t, taffy)
	if U64FromNodeID(a) != 1<<32|1 || U64FromNodeID(b) != 1<<32|2 {
		t.Fatalf("ids = %#x %#x", U64FromNodeID(a), U64FromNodeID(b))
	}
	_, err := taffy.Remove(a)
	mustOK(t, err)
	c := leaf(t, taffy)
	if U64FromNodeID(c) != 3<<32|1 {
		t.Fatalf("reused id = %#x, want version 3 at slot 1", U64FromNodeID(c))
	}
	taffy.Clear()
	d := leaf(t, taffy)
	if U64FromNodeID(d) != 3<<32|2 {
		t.Fatalf("id after clear = %#x, want the last drained slot", U64FromNodeID(d))
	}
}

func TestTreeTaffyTree_ErrorDisplay(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()
	node := leaf(t, taffy)
	_, err := taffy.ChildAtIndex(node, 0)
	want := "Index (is 0) should be < child_count (0) for parent node NodeId(4294967297)"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if err != TaffyErrorChildIndexOutOfBounds(node, 0, 0) {
		t.Fatal("error value mismatch")
	}
}

func TestTreeTaffyTree_DisjointNodeContextMut(t *testing.T) {
	taffy := NewTaffyTree[int]()
	a := unwrapResult(taffy.NewLeafWithContext(defaultStyle(), 1))
	b := unwrapResult(taffy.NewLeafWithContext(defaultStyle(), 2))
	refs, ok := taffy.GetDisjointNodeContextMut(a, b)
	if !ok || *refs[0] != 1 || *refs[1] != 2 {
		t.Fatal("disjoint lookup")
	}
	if _, ok := taffy.GetDisjointNodeContextMut(a, a); ok {
		t.Fatal("duplicate keys must fail")
	}
	if _, ok := taffy.GetDisjointNodeContextMut(a, b); !ok {
		t.Fatal("versions not restored")
	}
}
