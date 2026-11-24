package taffy

import "testing"

func TestHandWrittenMinMaxOverrides_MinOverridesMax(t *testing.T) {
	taffy := newTestTree()

	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	s.MinSize = Size[LengthPercentageAuto]{Width: hwLPA(100.0), Height: hwLPA(100.0)}
	s.MaxSize = Size[LengthPercentageAuto]{Width: hwLPA(10.0), Height: hwLPA(10.0)}
	child := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(child, hwDefinite(100.0, 100.0)))

	hwAssertEq(t, hwLayout(taffy, child).Size, Size[float32]{Width: 100.0, Height: 100.0})
}

func TestHandWrittenMinMaxOverrides_MaxOverridesSize(t *testing.T) {
	taffy := newTestTree()

	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	s.MaxSize = Size[LengthPercentageAuto]{Width: hwLPA(10.0), Height: hwLPA(10.0)}
	child := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(child, hwDefinite(100.0, 100.0)))

	hwAssertEq(t, hwLayout(taffy, child).Size, Size[float32]{Width: 10.0, Height: 10.0})
}

func TestHandWrittenMinMaxOverrides_MinOverridesSize(t *testing.T) {
	taffy := newTestTree()

	s := defaultStyle()
	s.Size = hwSizeDim(hwDim(50.0), hwDim(50.0))
	s.MinSize = Size[LengthPercentageAuto]{Width: hwLPA(100.0), Height: hwLPA(100.0)}
	child := hwUnwrap(taffy.NewLeaf(s))

	hwOK(taffy.ComputeLayout(child, hwDefinite(100.0, 100.0)))

	hwAssertEq(t, hwLayout(taffy, child).Size, Size[float32]{Width: 100.0, Height: 100.0})
}
