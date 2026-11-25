package taffy

func computeScrollableOverflowContribution(
	location Point[float32],
	size Size[float32],
	scrollableOverflowRect Rect[float32],
	overflow Point[Overflow],
	contain Contain,
	parentIsScrollContainer bool,
) Rect[float32] {
	isScrollContainer := overflow.X.IsScrollContainer() || overflow.Y.IsScrollContainer()
	overflowIsContained := contain.ContainsScrollableOverflow()
	propagates := Point[bool]{
		X: !isScrollContainer && !overflowIsContained && overflow.X == OverflowVisible,
		Y: !isScrollContainer && !overflowIsContained && overflow.Y == OverflowVisible,
	}
	endExtent := Size[float32]{Width: size.Width, Height: size.Height}
	if propagates.X {
		endExtent.Width = f32Max(size.Width, scrollableOverflowRect.Right)
	}
	if propagates.Y {
		endExtent.Height = f32Max(size.Height, scrollableOverflowRect.Bottom)
	}
	if endExtent.Width <= 0.0 || endExtent.Height <= 0.0 {
		return RectF32Zero
	}
	var startExtent Point[float32]
	if propagates.X {
		startExtent.X = f32Min(0.0, scrollableOverflowRect.Left)
	}
	if propagates.Y {
		startExtent.Y = f32Min(0.0, scrollableOverflowRect.Top)
	}
	contribution := Rect[float32]{
		Left:   location.X + startExtent.X,
		Right:  location.X + endExtent.Width,
		Top:    location.Y + startExtent.Y,
		Bottom: location.Y + endExtent.Height,
	}
	isWhollyUnreachable := contribution.Right <= 0.0 || contribution.Bottom <= 0.0
	if parentIsScrollContainer && isWhollyUnreachable {
		return RectF32Zero
	}
	return contribution
}
