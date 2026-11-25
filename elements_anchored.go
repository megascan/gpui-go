package gpui

// AnchoredState is the state that the anchored element uses to track its children.
type AnchoredState struct {
	childLayoutIDs []LayoutID
}

// Anchored is an anchored element that can be used to display UI that will avoid overflowing
// the window bounds.
type Anchored struct {
	Parent[*Anchored]
	Fluent[*Anchored]
	children     []AnyElement
	anchor       Anchor
	fitMode      AnchoredFitMode
	placement    anchoredPlacement
	positionMode AnchoredPositionMode
	offset       Option[Point[Pixels]]
}

type anchoredPlacementKind uint8

const (
	anchoredPlacementInPlace anchoredPlacementKind = iota
	anchoredPlacementAt
	anchoredPlacementAttached
)

type anchoredPlacement struct {
	kind     anchoredPlacementKind
	position Point[Pixels]
	target   Bounds[Pixels]
	corner   Anchor
}

// NewAnchored gives you an element that will avoid overflowing the window bounds. Its children
// should have no margin to avoid measurement issues.
func NewAnchored() *Anchored {
	a := &Anchored{
		anchor:       AnchorTopLeft,
		fitMode:      AnchoredFitModeSwitchAnchor(),
		placement:    anchoredPlacement{kind: anchoredPlacementInPlace},
		positionMode: AnchoredPositionModeWindow,
	}
	BindParent(&a.Parent, a)
	BindFluent(&a.Fluent, a)
	return a
}

// Anchor sets which corner of the anchored element should be anchored to the current position.
func (a *Anchored) Anchor(anchor Anchor) *Anchored {
	a.anchor = anchor
	return a
}

// Position sets the position in window coordinates (otherwise the location the anchored element
// is rendered is used).
func (a *Anchored) Position(anchor Point[Pixels]) *Anchored {
	a.placement = anchoredPlacement{kind: anchoredPlacementAt, position: anchor}
	return a
}

// Attach places the element next to target, with the element's Anchor corner on corner of
// target. For example, a submenu puts its top-left corner on its row's top-right corner. Use
// this instead of Position.
//
// If the element doesn't fit in the window, it moves to the other side of target, as long as
// that side fits better. A submenu moves from the right of its row to the left, and a dropdown
// moves from below its button to above. The offset flips with it. AnchoredFitModeSwitchAnchor
// has no effect here.
func (a *Anchored) Attach(target Bounds[Pixels], corner Anchor) *Anchored {
	a.placement = anchoredPlacement{kind: anchoredPlacementAttached, target: target, corner: corner}
	return a
}

// Offset offsets the final position by this amount. Useful when you want to anchor to an element
// but offset from it, such as in PopoverMenu.
func (a *Anchored) Offset(offset Point[Pixels]) *Anchored {
	a.offset = Some(offset)
	return a
}

// PositionMode sets the position mode for this anchored element. Local will have this interpret
// its Position or Attach target as relative to the parent element. While Window will have it
// interpret the position as relative to the window.
func (a *Anchored) PositionMode(mode AnchoredPositionMode) *Anchored {
	a.positionMode = mode
	return a
}

// SnapToWindow snaps to window edge instead of switching anchor corner when an overflow would
// occur.
func (a *Anchored) SnapToWindow() *Anchored {
	a.fitMode = AnchoredFitModeSnapToWindow()
	return a
}

// SnapToWindowWithMargin snaps to window edge and leaves some margins.
func (a *Anchored) SnapToWindowWithMargin(edges Edges[Pixels]) *Anchored {
	a.fitMode = AnchoredFitModeSnapToWindowWithMargin(edges)
	return a
}

// Extend adds child elements.
func (a *Anchored) Extend(elements []AnyElement) {
	a.children = append(a.children, elements...)
}

// ElementID reports that an anchored element has no id.
func (a *Anchored) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (a *Anchored) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests the children's layouts under an absolutely positioned flex node.
func (a *Anchored) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, AnchoredState) {
	childLayoutIDs := make([]LayoutID, 0, len(a.children))
	for i := range a.children {
		childLayoutIDs = append(childLayoutIDs, a.children[i].RequestLayout(window, cx))
	}

	anchoredStyle := DefaultStyle()
	anchoredStyle.Position = PositionAbsolute
	anchoredStyle.Display = DisplayFlex

	layoutID := window.RequestLayout(anchoredStyle, childLayoutIDs, cx)

	return layoutID, AnchoredState{childLayoutIDs: childLayoutIDs}
}

// Prepaint positions the children so they avoid overflowing the window, then prepaints them.
func (a *Anchored) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], requestLayout *AnchoredState, window *Window, cx *App) struct{} {
	if len(requestLayout.childLayoutIDs) == 0 {
		return struct{}{}
	}

	childrenBounds := window.LayoutBounds(requestLayout.childLayoutIDs[0])
	for _, id := range requestLayout.childLayoutIDs[1:] {
		childrenBounds = childrenBounds.Union(window.LayoutBounds(id))
	}

	var anchorPosition Option[Point[Pixels]]
	switch a.placement.kind {
	case anchoredPlacementInPlace:
		anchorPosition = None[Point[Pixels]]()
	case anchoredPlacementAt:
		anchorPosition = Some(a.placement.position)
	case anchoredPlacementAttached:
		anchorPosition = Some(a.placement.target.Corner(a.placement.corner))
	}
	origin, desired := a.positionMode.getPositionAndBounds(anchorPosition, a.anchor, childrenBounds.Size, bounds, a.offset)

	limits := Bounds[Pixels]{Origin: Point[Pixels]{}, Size: window.ViewportSize()}

	if a.placement.kind == anchoredPlacementAttached {
		target, corner := a.placement.target, a.placement.corner
		visible := limits.Inset(window.clientInset.UnwrapOr(Px(0.)))
		overflowAlong := func(bounds Bounds[Pixels], axis Axis) Pixels {
			if axis == AxisHorizontal {
				return max(visible.Left()-bounds.Left(), Px(0.)) + max(bounds.Right()-visible.Right(), Px(0.))
			}
			return max(visible.Top()-bounds.Top(), Px(0.)) + max(bounds.Bottom()-visible.Bottom(), Px(0.))
		}
		anchor := a.anchor
		offset := a.offset.UnwrapOrDefault()
		for _, axis := range []Axis{AxisHorizontal, AxisVertical} {
			cornerSide := anchoredSideAlong(corner, axis)
			isOutsideTarget := cornerSide != 0 && anchoredSideAlong(anchor, axis) == -cornerSide
			overflow := overflowAlong(desired, axis)
			if !isOutsideTarget || overflow == Px(0.) {
				continue
			}

			flippedCorner := corner.OtherSideAlong(axis)
			flippedAnchor := anchor.OtherSideAlong(axis)
			var flippedOffset Point[Pixels]
			if axis == AxisHorizontal {
				flippedOffset = NewPoint(-offset.X, offset.Y)
			} else {
				flippedOffset = NewPoint(offset.X, -offset.Y)
			}
			_, flipped := a.positionMode.getPositionAndBounds(Some(target.Corner(flippedCorner)), flippedAnchor, childrenBounds.Size, bounds, Some(flippedOffset))
			if overflowAlong(flipped, axis) < overflow {
				corner = flippedCorner
				anchor = flippedAnchor
				offset = flippedOffset
				desired = flipped
			}
		}
	} else if a.fitMode == AnchoredFitModeSwitchAnchor() {
		anchor := a.anchor

		if desired.Left() < limits.Left() || desired.Right() > limits.Right() {
			switched := BoundsFromAnchorAndSize(anchor.OtherSideAlong(AxisHorizontal), origin, childrenBounds.Size)
			if !(switched.Left() < limits.Left() || switched.Right() > limits.Right()) {
				anchor = anchor.OtherSideAlong(AxisHorizontal)
				desired = switched
			}
		}

		if desired.Top() < limits.Top() || desired.Bottom() > limits.Bottom() {
			switched := BoundsFromAnchorAndSize(anchor.OtherSideAlong(AxisVertical), origin, childrenBounds.Size)
			if !(switched.Top() < limits.Top() || switched.Bottom() > limits.Bottom()) {
				desired = switched
			}
		}
	}

	clientInset := window.clientInset.UnwrapOr(Px(0.))
	edges, ok := a.fitMode.AsSnapToWindowWithMargin()
	if !ok {
		edges = Edges[Pixels]{}
	}
	edges = edges.Map(func(edge Pixels) Pixels { return edge + clientInset })

	if desired.Right() > limits.Right() {
		desired.Origin.X -= desired.Right() - limits.Right() + edges.Right
	}
	if desired.Left() < limits.Left() {
		desired.Origin.X = limits.Origin.X + edges.Left
	}

	if desired.Bottom() > limits.Bottom() {
		desired.Origin.Y -= desired.Bottom() - limits.Bottom() + edges.Bottom
	}
	if desired.Top() < limits.Top() {
		desired.Origin.Y = limits.Origin.Y + edges.Top
	}

	offset := desired.Origin.Sub(bounds.Origin)
	offset = NewPoint(offset.X.Round(), offset.Y.Round())

	return window.WithElementOffset(offset, func(window *Window) struct{} {
		for i := range a.children {
			a.children[i].Prepaint(window, cx)
		}
		return struct{}{}
	})
}

// Paint paints the children.
func (a *Anchored) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *AnchoredState, _ *struct{}, window *Window, cx *App) {
	for i := range a.children {
		a.children[i].Paint(window, cx)
	}
}

// IntoAnyElement converts the anchored element into an AnyElement.
func (a *Anchored) IntoAnyElement() AnyElement {
	return NewAnyElement[AnchoredState, struct{}](a)
}

func anchoredSideAlong(anchor Anchor, axis Axis) int {
	if axis == AxisHorizontal {
		switch anchor {
		case AnchorTopLeft, AnchorBottomLeft, AnchorLeftCenter:
			return -1
		case AnchorTopCenter, AnchorBottomCenter:
			return 0
		default:
			return 1
		}
	}
	switch anchor {
	case AnchorTopLeft, AnchorTopRight, AnchorTopCenter:
		return -1
	case AnchorLeftCenter, AnchorRightCenter:
		return 0
	default:
		return 1
	}
}

type anchoredFitModeKind uint8

const (
	anchoredFitModeSnapToWindow anchoredFitModeKind = iota
	anchoredFitModeSnapToWindowWithMargin
	anchoredFitModeSwitchAnchor
)

// AnchoredFitMode is which algorithm to use when fitting the anchored element to be inside the
// window. It is a comparable tagged value.
type AnchoredFitMode struct {
	kind    anchoredFitModeKind
	margins Edges[Pixels]
}

// AnchoredFitModeSnapToWindow snaps the anchored element to the window edge.
func AnchoredFitModeSnapToWindow() AnchoredFitMode {
	return AnchoredFitMode{kind: anchoredFitModeSnapToWindow}
}

// AnchoredFitModeSnapToWindowWithMargin snaps to window edge and leaves some margins.
func AnchoredFitModeSnapToWindowWithMargin(edges Edges[Pixels]) AnchoredFitMode {
	return AnchoredFitMode{kind: anchoredFitModeSnapToWindowWithMargin, margins: edges}
}

// AnchoredFitModeSwitchAnchor switches which corner anchor this anchored element is attached to.
func AnchoredFitModeSwitchAnchor() AnchoredFitMode {
	return AnchoredFitMode{kind: anchoredFitModeSwitchAnchor}
}

// AsSnapToWindowWithMargin returns the margins when m is SnapToWindowWithMargin.
func (m AnchoredFitMode) AsSnapToWindowWithMargin() (Edges[Pixels], bool) {
	return m.margins, m.kind == anchoredFitModeSnapToWindowWithMargin
}

// AnchoredPositionMode is which algorithm to use when positioning the anchored element.
type AnchoredPositionMode uint8

const (
	// AnchoredPositionModeWindow positions the anchored element relative to the window.
	AnchoredPositionModeWindow AnchoredPositionMode = iota
	// AnchoredPositionModeLocal positions the anchored element relative to its parent.
	AnchoredPositionModeLocal
)

func (m AnchoredPositionMode) getPositionAndBounds(anchorPosition Option[Point[Pixels]], anchor Anchor, size Size[Pixels], bounds Bounds[Pixels], offset Option[Point[Pixels]]) (Point[Pixels], Bounds[Pixels]) {
	off := offset.UnwrapOrDefault()

	switch m {
	case AnchoredPositionModeWindow:
		position := anchorPosition.UnwrapOr(bounds.Origin)
		return position, BoundsFromAnchorAndSize(anchor, position.Add(off), size)
	default:
		position := anchorPosition.UnwrapOrDefault()
		return position, BoundsFromAnchorAndSize(anchor, bounds.Origin.Add(position).Add(off), size)
	}
}
