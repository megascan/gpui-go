package taffy

import "testing"

func TestComputeMod_HiddenLayoutShouldHideRecursively(t *testing.T) {
	taffy := NewTaffyTree[struct{}]()

	style := StyleDefault[defaultCheapStr]()
	style.Display = DisplayFlex
	style.Size = SizeDimensionFromLengths(50.0, 50.0)

	grandchild00, _ := taffy.NewLeaf(style)
	grandchild01, _ := taffy.NewLeaf(style)
	child00, _ := taffy.NewWithChildren(style, []NodeID{grandchild00, grandchild01})

	grandchild02, _ := taffy.NewLeaf(style)
	child01, _ := taffy.NewWithChildren(style, []NodeID{grandchild02})

	rootStyle := StyleDefault[defaultCheapStr]()
	rootStyle.Display = DisplayNone
	rootStyle.Size = SizeDimensionFromLengths(50.0, 50.0)
	root, _ := taffy.NewWithChildren(rootStyle, []NodeID{child00, child01})

	ComputeHiddenLayout(taffy.asLayoutTree(), root)

	for _, node := range []NodeID{root, child00, child01, grandchild00, grandchild01, grandchild02} {
		layout, err := taffy.Layout(node)
		if err != nil {
			t.Fatal(err)
		}
		if layout.Size != SizeZero[float32]() {
			t.Errorf("%v size = %v, want zero", node, layout.Size)
		}
		if layout.Location != PointZero[float32]() {
			t.Errorf("%v location = %v, want zero", node, layout.Location)
		}
	}
}
