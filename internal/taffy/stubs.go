package taffy

type itemPlacement = inBothAbsAxis[Line[originZeroGridPlacement]]

// DetailedGridInfo is the detailed information of a grid container's layout (owner: M0-taffy-grid-sizing).
type DetailedGridInfo[S CheapCloneStr] struct{}

// ComputeGridLayout computes the layout of a CSS grid container and its direct children (owner: M0-taffy-grid-sizing).
func ComputeGridLayout[S CheapCloneStr](tree LayoutGridContainer[S], node NodeID, inputs LayoutInput) LayoutOutput {
	panic("taffy: unimplemented ComputeGridLayout")
}

func (d *DetailedGridInfo[S]) resolveAbsoluteGridArea(gridRow, gridColumn Line[GridPlacement[S]], direction Direction, areaRect Rect[float32]) Rect[float32] {
	panic("taffy: unimplemented DetailedGridInfo.resolveAbsoluteGridArea")
}
