package taffy

import "testing"

func TestHandWrittenRounding_RoundingDoesntLeaveGaps(t *testing.T) {
	taffy := newTestTree()

	wSquare := hwSizeDim(hwDim(100.3), hwDim(100.3))
	sa := defaultStyle()
	sa.Size = wSquare
	childA := hwUnwrap(taffy.NewLeaf(sa))
	childB := hwUnwrap(taffy.NewLeaf(sa))

	rs := defaultStyle()
	rs.Size = hwSizeDim(hwDim(963.3333), hwDim(1000.))
	rs.JustifyContent = AlignContentCenter
	rootNode := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{childA, childB}))

	hwOK(taffy.ComputeLayout(rootNode, SizeMaxContent[AvailableSpace]()))
	taffy.PrintTree(rootNode)

	layoutA := hwLayout(taffy, childA)
	layoutB := hwLayout(taffy, childB)
	hwAssertEq(t, layoutA.Location.X+layoutA.Size.Width, layoutB.Location.X)
}
