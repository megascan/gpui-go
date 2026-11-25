package gpui

import (
	"fmt"
	"unsafe"

	"github.com/megascan/gpui-go/internal/taffy"
)

type measureFn func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels]

type nodeContext struct {
	measure measureFn
}

// TaffyLayoutEngine lays out the element tree of a frame with taffy.
type TaffyLayoutEngine struct {
	taffy                    *taffy.TaffyTree[nodeContext]
	absoluteLayoutBounds     map[LayoutID]Bounds[Pixels]
	absoluteOuterOrigins     map[LayoutID]Point[float32]
	computedLayouts          map[LayoutID]struct{}
	layoutBoundsScratchSpace []LayoutID
}

const expectMessage = "we should avoid taffy layout errors by construction if possible"

func expectTaffy(err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %v", expectMessage, err))
	}
}

// NewTaffyLayoutEngine creates a layout engine with taffy's rounding disabled.
func NewTaffyLayoutEngine() *TaffyLayoutEngine {
	tree := taffy.NewTaffyTree[nodeContext]()
	tree.DisableRounding()
	return &TaffyLayoutEngine{
		taffy:                    tree,
		absoluteLayoutBounds:     make(map[LayoutID]Bounds[Pixels]),
		absoluteOuterOrigins:     make(map[LayoutID]Point[float32]),
		computedLayouts:          make(map[LayoutID]struct{}),
		layoutBoundsScratchSpace: nil,
	}
}

// Clear drops every node and cached layout.
func (e *TaffyLayoutEngine) Clear() {
	e.taffy.Clear()
	clear(e.absoluteLayoutBounds)
	clear(e.absoluteOuterOrigins)
	clear(e.computedLayouts)
}

// RequestLayout adds a node with the given style and children to the layout tree.
func (e *TaffyLayoutEngine) RequestLayout(style Style, remSize Pixels, scaleFactor float32, children []LayoutID) LayoutID {
	taffyStyle := style.toTaffy(remSize, scaleFactor)

	if len(children) == 0 {
		id, err := e.taffy.NewLeaf(taffyStyle)
		expectTaffy(err)
		return LayoutID{id: id}
	}
	id, err := e.taffy.NewWithChildren(taffyStyle, layoutIDToTaffySlice(children))
	expectTaffy(err)
	return LayoutID{id: id}
}

// RequestMeasuredLayout adds a leaf whose size is computed by measure during layout.
func (e *TaffyLayoutEngine) RequestMeasuredLayout(style Style, remSize Pixels, scaleFactor float32, measure func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels]) LayoutID {
	taffyStyle := style.toTaffy(remSize, scaleFactor)

	id, err := e.taffy.NewLeafWithContext(taffyStyle, nodeContext{measure: measure})
	expectTaffy(err)
	return LayoutID{id: id}
}

// StretchAutoSizeToFill treats any auto dimension of the node's style as filling size.
func (e *TaffyLayoutEngine) StretchAutoSizeToFill(id LayoutID, size Size[Pixels], scaleFactor float32) {
	style, err := e.taffy.Style(id.id)
	expectTaffy(err)
	stretchWidth := style.Size.Width.IsAuto()
	stretchHeight := style.Size.Height.IsAuto()
	if !stretchWidth && !stretchHeight {
		return
	}
	cloned := *style
	if stretchWidth {
		cloned.Size.Width = taffy.DimensionLength(roundToDevicePixel(float32(size.Width), scaleFactor))
	}
	if stretchHeight {
		cloned.Size.Height = taffy.DimensionLength(roundToDevicePixel(float32(size.Height), scaleFactor))
	}
	expectTaffy(e.taffy.SetStyle(id.id, cloned))
}

func (e *TaffyLayoutEngine) countAllChildren(parent LayoutID) (uint32, error) {
	var count uint32
	children, err := e.taffy.Children(parent.id)
	if err != nil {
		return 0, err
	}
	for _, child := range children {
		count++
		n, err := e.countAllChildren(LayoutID{id: child})
		if err != nil {
			return 0, err
		}
		count += n
	}
	return count, nil
}

func (e *TaffyLayoutEngine) maxDepth(depth uint32, parent LayoutID) (uint32, error) {
	fmt.Printf("%v at depth %d has %d children\n", parent, depth, e.taffy.ChildCount(parent.id))

	var maxChildDepth uint32
	children, err := e.taffy.Children(parent.id)
	if err != nil {
		return 0, err
	}
	for _, child := range children {
		d, err := e.maxDepth(0, LayoutID{id: child})
		if err != nil {
			return 0, err
		}
		maxChildDepth = max(maxChildDepth, d)
	}
	return depth + 1 + maxChildDepth, nil
}

func (e *TaffyLayoutEngine) getEdges(parent LayoutID) ([][2]LayoutID, error) {
	var edges [][2]LayoutID
	children, err := e.taffy.Children(parent.id)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		edges = append(edges, [2]LayoutID{parent, {id: child}})
		sub, err := e.getEdges(LayoutID{id: child})
		if err != nil {
			return nil, err
		}
		edges = append(edges, sub...)
	}
	return edges, nil
}

// ComputeLayout computes the layout of id and its descendants within availableSpace.
func (e *TaffyLayoutEngine) ComputeLayout(id LayoutID, availableSpace SizeOf[AvailableSpace], window *Window, cx *App) {
	if _, ok := e.computedLayouts[id]; ok {
		stack := e.layoutBoundsScratchSpace
		stack = append(stack, id)
		for len(stack) > 0 {
			id := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			delete(e.absoluteLayoutBounds, id)
			delete(e.absoluteOuterOrigins, id)
			children, err := e.taffy.Children(id.id)
			expectTaffy(err)
			for _, child := range children {
				stack = append(stack, LayoutID{id: child})
			}
		}
		e.layoutBoundsScratchSpace = stack[:0]
	} else {
		e.computedLayouts[id] = struct{}{}
	}

	scaleFactor := window.ScaleFactor()

	transform := func(v AvailableSpace) AvailableSpace {
		if pixels, ok := v.AsDefinite(); ok {
			return AvailableSpaceDefinite(Pixels(float32(pixels) * scaleFactor))
		}
		return v
	}
	availableSpace = NewSizeOf(transform(availableSpace.Width), transform(availableSpace.Height))

	err := e.taffy.ComputeLayoutWithMeasure(
		id.id,
		sizeAvailableSpaceToTaffy(availableSpace),
		func(input taffy.LayoutInput, _ taffy.NodeID, nodeContext *nodeContext, style *taffy.Style[string]) taffy.LayoutOutput {
			return taffy.ComputeLeafLayout(
				input,
				style.Ref(),
				func(unsafe.Pointer, float32) float32 { return 0.0 },
				func(knownDimensions taffy.Size[Option[float32]], availableSpace taffy.Size[taffy.AvailableSpace]) taffy.Size[float32] {
					if nodeContext == nil {
						return taffy.Size[float32]{}
					}

					known := SizeOf[Option[Pixels]]{
						Width:  optionPixelsFromF32(knownDimensions.Width, scaleFactor),
						Height: optionPixelsFromF32(knownDimensions.Height, scaleFactor),
					}

					space := sizeAvailableSpaceFromTaffy(availableSpace)
					untransform := func(ev AvailableSpace) AvailableSpace {
						if pixels, ok := ev.AsDefinite(); ok {
							return AvailableSpaceDefinite(Pixels(float32(pixels) / scaleFactor))
						}
						return ev
					}
					space = NewSizeOf(untransform(space.Width), untransform(space.Height))

					measuredSize := nodeContext.measure(known, space, window, cx)
					snapped := snapMeasuredSizeToDevicePixels(measuredSize, scaleFactor)
					return taffy.Size[float32]{Width: snapped.Width, Height: snapped.Height}
				},
			)
		},
	)
	expectTaffy(err)
}

func optionPixelsFromF32(v Option[float32], scaleFactor float32) Option[Pixels] {
	if e, ok := v.Get(); ok {
		return Some(Pixels(e / scaleFactor))
	}
	return None[Pixels]()
}

// LayoutBounds returns the bounds of id relative to the window, snapped to device pixels.
func (e *TaffyLayoutEngine) LayoutBounds(id LayoutID, scaleFactor float32) Bounds[Pixels] {
	if layout, ok := e.absoluteLayoutBounds[id]; ok {
		return layout
	}

	layout, err := e.taffy.Layout(id.id)
	expectTaffy(err)
	layoutLocation := layout.Location
	layoutSize := layout.Size
	parent := e.taffy.Parent(id.id)

	var absoluteOuterOrigin Point[float32]
	if parentNode, ok := parent.Get(); ok {
		parentID := LayoutID{id: parentNode}
		e.LayoutBounds(parentID, scaleFactor)
		parentOrigin, ok := e.absoluteOuterOrigins[parentID]
		if !ok {
			panic("parent absolute outer origin should be cached")
		}
		absoluteOuterOrigin = parentOrigin.Add(pointFromTaffy(layoutLocation))
	} else {
		absoluteOuterOrigin = pointFromTaffy(layoutLocation)
	}
	e.absoluteOuterOrigins[id] = absoluteOuterOrigin

	absoluteFar := absoluteOuterOrigin.Add(Point[float32]{X: layoutSize.Width, Y: layoutSize.Height})
	snappedBounds := BoundsFromCorners(
		absoluteOuterOrigin.Map(roundHalfTowardZero),
		absoluteFar.Map(roundHalfTowardZero),
	)

	bounds := snappedBounds.Div(scaleFactor).Map(func(v float32) Pixels { return Pixels(v) })
	e.absoluteLayoutBounds[id] = bounds
	return bounds
}

// LayoutID is a unique identifier for a layout node, generated when requesting a layout from taffy.
type LayoutID struct {
	id taffy.NodeID
}

func layoutIDToTaffySlice(nodeIDs []LayoutID) []taffy.NodeID {
	if len(nodeIDs) == 0 {
		return nil
	}
	return unsafe.Slice((*taffy.NodeID)(unsafe.Pointer(unsafe.SliceData(nodeIDs))), len(nodeIDs))
}

// Hash feeds the node id to h as upstream's Hash impl does.
func (id LayoutID) Hash(h *FxHasher) {
	h.WriteU64(taffy.U64FromNodeID(id.id))
}

// String formats the id like upstream's Debug impl.
func (id LayoutID) String() string {
	return fmt.Sprintf("LayoutId(%v)", id.id)
}

func snapMeasuredSizeToDevicePixels(size Size[Pixels], scaleFactor float32) Size[float32] {
	return Size[float32]{
		Width:  ceilToDevicePixel(max(float32(size.Width), 0.0), scaleFactor),
		Height: ceilToDevicePixel(max(float32(size.Height), 0.0), scaleFactor),
	}
}

func borderWidthsToTaffy(widths *EdgesOf[AbsoluteLength], remSize Pixels, scaleFactor float32) taffy.Rect[taffy.LengthPercentage] {
	snap := func(w *AbsoluteLength) taffy.LengthPercentage {
		return taffy.LengthPercentageLength(roundStrokeToDevicePixel(float32(w.ToPixels(remSize)), scaleFactor))
	}
	return taffy.Rect[taffy.LengthPercentage]{
		Top:    snap(&widths.Top),
		Right:  snap(&widths.Right),
		Bottom: snap(&widths.Bottom),
		Left:   snap(&widths.Left),
	}
}

func (s *Style) toTaffy(remSize Pixels, scaleFactor float32) taffy.Style[string] {
	toGridLine := func(placement GridPlacementRange) taffy.Line[taffy.GridPlacement[string]] {
		return taffy.Line[taffy.GridPlacement[string]]{
			Start: gridPlacementToTaffy(placement.Start),
			End:   gridPlacementToTaffy(placement.End),
		}
	}

	toGridRepeat := func(unit Option[GridTemplate]) []taffy.GridTemplateComponent[string] {
		template, ok := unit.Get()
		if !ok {
			return nil
		}
		switch template.MinSize {
		case GridTemplateMinSizeMinContent:
			return []taffy.GridTemplateComponent[string]{taffy.Repeat[string](
				template.Repeat,
				[]taffy.TrackSizingFunction{taffy.Minmax[taffy.TrackSizingFunction](taffy.MinContent[taffy.MinTrackSizingFunction](), taffy.Fr[taffy.MaxTrackSizingFunction](1.0))},
			)}
		case GridTemplateMinSizeMaxContent:
			return []taffy.GridTemplateComponent[string]{taffy.Repeat[string](
				template.Repeat,
				[]taffy.TrackSizingFunction{taffy.Minmax[taffy.TrackSizingFunction](taffy.Length[taffy.MinTrackSizingFunction](0.0), taffy.MaxContent[taffy.MaxTrackSizingFunction]())},
			)}
		default:
			return []taffy.GridTemplateComponent[string]{taffy.Repeat[string](
				template.Repeat,
				[]taffy.TrackSizingFunction{taffy.Minmax[taffy.TrackSizingFunction](taffy.Length[taffy.MinTrackSizingFunction](0.0), taffy.Fr[taffy.MaxTrackSizingFunction](1.0))},
			)}
		}
	}

	t := taffy.StyleDefault[string]()
	t.Display = displayToTaffy(s.Display)
	t.Overflow = taffy.Point[taffy.Overflow]{X: overflowToTaffy(s.Overflow.X), Y: overflowToTaffy(s.Overflow.Y)}
	t.ScrollbarWidth = absoluteLengthToTaffy(s.ScrollbarWidth, remSize, scaleFactor)
	t.Position = positionToTaffy(s.Position)
	t.Inset = edgesLengthToTaffy(s.Inset, remSize, scaleFactor)
	t.Size = taffy.Size[taffy.Dimension]{
		Width:  lengthToTaffyDimension(s.Size.Width, remSize, scaleFactor),
		Height: lengthToTaffyDimension(s.Size.Height, remSize, scaleFactor),
	}
	t.MinSize = sizeLengthToTaffy(s.MinSize, remSize, scaleFactor)
	t.MaxSize = sizeLengthToTaffy(s.MaxSize, remSize, scaleFactor)
	t.AspectRatio = s.AspectRatio
	t.Margin = edgesLengthToTaffy(s.Margin, remSize, scaleFactor)
	t.Padding = taffy.Rect[taffy.LengthPercentage]{
		Top:    definiteLengthToTaffyLengthPercentage(s.Padding.Top, remSize, scaleFactor),
		Right:  definiteLengthToTaffyLengthPercentage(s.Padding.Right, remSize, scaleFactor),
		Bottom: definiteLengthToTaffyLengthPercentage(s.Padding.Bottom, remSize, scaleFactor),
		Left:   definiteLengthToTaffyLengthPercentage(s.Padding.Left, remSize, scaleFactor),
	}
	t.Border = borderWidthsToTaffy(&s.BorderWidths, remSize, scaleFactor)
	t.AlignItems = taffy.AlignItemsDefault
	if v, ok := s.AlignItems.Get(); ok {
		t.AlignItems = alignItemsToTaffy(v)
	}
	t.AlignSelf = None[taffy.AlignSelf]()
	if v, ok := s.AlignSelf.Get(); ok {
		t.AlignSelf = Some(alignItemsToTaffy(v))
	}
	t.AlignContent = taffy.AlignContentDefault
	if v, ok := s.AlignContent.Get(); ok {
		t.AlignContent = alignContentToTaffy(v)
	}
	t.JustifyContent = taffy.AlignContentDefault
	if v, ok := s.JustifyContent.Get(); ok {
		t.JustifyContent = alignContentToTaffy(v)
	}
	t.Gap = taffy.Size[taffy.LengthPercentage]{
		Width:  definiteLengthToTaffyLengthPercentage(s.Gap.Width, remSize, scaleFactor),
		Height: definiteLengthToTaffyLengthPercentage(s.Gap.Height, remSize, scaleFactor),
	}
	t.FlexDirection = flexDirectionToTaffy(s.FlexDirection)
	t.FlexWrap = flexWrapToTaffy(s.FlexWrap)
	t.FlexBasis = lengthToTaffyDimension(s.FlexBasis, remSize, scaleFactor)
	t.FlexGrow = s.FlexGrow
	t.FlexShrink = s.FlexShrink
	t.GridTemplateRows = toGridRepeat(s.GridRows)
	t.GridTemplateColumns = toGridRepeat(s.GridCols)
	t.GridRow = taffy.Line[taffy.GridPlacement[string]]{}
	t.GridColumn = taffy.Line[taffy.GridPlacement[string]]{}
	if location, ok := s.GridLocation.Get(); ok {
		t.GridRow = toGridLine(location.Row)
		t.GridColumn = toGridLine(location.Column)
	}
	return t
}

func absoluteLengthToTaffy(a AbsoluteLength, remSize Pixels, scaleFactor float32) float32 {
	return roundToDevicePixel(float32(a.ToPixels(remSize)), scaleFactor)
}

func lengthToTaffyLengthPercentageAuto(l Length, remSize Pixels, scaleFactor float32) taffy.LengthPercentageAuto {
	if length, ok := l.AsDefinite(); ok {
		return definiteLengthToTaffyLengthPercentageAuto(length, remSize, scaleFactor)
	}
	return taffy.LengthPercentageAutoAuto()
}

func lengthToTaffyDimension(l Length, remSize Pixels, scaleFactor float32) taffy.Dimension {
	if length, ok := l.AsDefinite(); ok {
		return definiteLengthToTaffyDimension(length, remSize, scaleFactor)
	}
	return taffy.DimensionAuto()
}

func definiteLengthToTaffyLengthPercentage(d DefiniteLength, remSize Pixels, scaleFactor float32) taffy.LengthPercentage {
	if fraction, ok := d.AsFraction(); ok {
		return taffy.LengthPercentagePercent(fraction)
	}
	length, _ := d.AsAbsolute()
	return taffy.LengthPercentageLength(absoluteLengthToTaffy(length, remSize, scaleFactor))
}

func definiteLengthToTaffyLengthPercentageAuto(d DefiniteLength, remSize Pixels, scaleFactor float32) taffy.LengthPercentageAuto {
	if fraction, ok := d.AsFraction(); ok {
		return taffy.LengthPercentageAutoPercent(fraction)
	}
	length, _ := d.AsAbsolute()
	return taffy.LengthPercentageAutoLength(absoluteLengthToTaffy(length, remSize, scaleFactor))
}

func definiteLengthToTaffyDimension(d DefiniteLength, remSize Pixels, scaleFactor float32) taffy.Dimension {
	if fraction, ok := d.AsFraction(); ok {
		return taffy.DimensionPercent(fraction)
	}
	length, _ := d.AsAbsolute()
	return taffy.DimensionLength(absoluteLengthToTaffy(length, remSize, scaleFactor))
}

func sizeLengthToTaffy(s SizeOf[Length], remSize Pixels, scaleFactor float32) taffy.Size[taffy.LengthPercentageAuto] {
	return taffy.Size[taffy.LengthPercentageAuto]{
		Width:  lengthToTaffyLengthPercentageAuto(s.Width, remSize, scaleFactor),
		Height: lengthToTaffyLengthPercentageAuto(s.Height, remSize, scaleFactor),
	}
}

func edgesLengthToTaffy(e EdgesOf[Length], remSize Pixels, scaleFactor float32) taffy.Rect[taffy.LengthPercentageAuto] {
	return taffy.Rect[taffy.LengthPercentageAuto]{
		Top:    lengthToTaffyLengthPercentageAuto(e.Top, remSize, scaleFactor),
		Right:  lengthToTaffyLengthPercentageAuto(e.Right, remSize, scaleFactor),
		Bottom: lengthToTaffyLengthPercentageAuto(e.Bottom, remSize, scaleFactor),
		Left:   lengthToTaffyLengthPercentageAuto(e.Left, remSize, scaleFactor),
	}
}

func pointFromTaffy[T Number](point taffy.Point[T]) Point[T] {
	return Point[T]{X: point.X, Y: point.Y}
}

func pointToTaffy[T Number](point Point[T]) taffy.Point[T] {
	return taffy.Point[T]{X: point.X, Y: point.Y}
}

func sizeFromTaffy[T Number](size taffy.Size[T]) Size[T] {
	return Size[T]{Width: size.Width, Height: size.Height}
}

func sizeToTaffy[T Number](size Size[T]) taffy.Size[T] {
	return taffy.Size[T]{Width: size.Width, Height: size.Height}
}

// AvailableSpace is the space available for an element to be laid out in. The zero value is
// AvailableSpaceMinContent, upstream's default.
type AvailableSpace struct {
	kind     availableSpaceKind
	definite Pixels
}

type availableSpaceKind uint8

const (
	availableSpaceMinContent availableSpaceKind = iota
	availableSpaceDefinite
	availableSpaceMaxContent
)

// AvailableSpaceDefinite is the Definite variant: the specified number of pixels.
func AvailableSpaceDefinite(pixels Pixels) AvailableSpace {
	return AvailableSpace{kind: availableSpaceDefinite, definite: pixels}
}

// AvailableSpaceMinContent is the MinContent variant: layout under a min-content constraint.
func AvailableSpaceMinContent() AvailableSpace {
	return AvailableSpace{kind: availableSpaceMinContent}
}

// AvailableSpaceMaxContent is the MaxContent variant: layout under a max-content constraint.
func AvailableSpaceMaxContent() AvailableSpace {
	return AvailableSpace{kind: availableSpaceMaxContent}
}

// AsDefinite returns the pixels when a is the Definite variant.
func (a AvailableSpace) AsDefinite() (Pixels, bool) {
	return a.definite, a.kind == availableSpaceDefinite
}

// IsMinContent reports whether a is the MinContent variant.
func (a AvailableSpace) IsMinContent() bool {
	return a.kind == availableSpaceMinContent
}

// IsMaxContent reports whether a is the MaxContent variant.
func (a AvailableSpace) IsMaxContent() bool {
	return a.kind == availableSpaceMaxContent
}

// String formats the value like upstream's Debug impl.
func (a AvailableSpace) String() string {
	switch a.kind {
	case availableSpaceDefinite:
		return fmt.Sprintf("Definite(%v)", a.definite)
	case availableSpaceMaxContent:
		return "MaxContent"
	}
	return "MinContent"
}

// AvailableSpaceMinSize returns a size with both dimensions set to AvailableSpaceMinContent.
func AvailableSpaceMinSize() SizeOf[AvailableSpace] {
	return SizeOf[AvailableSpace]{
		Width:  AvailableSpaceMinContent(),
		Height: AvailableSpaceMinContent(),
	}
}

// SizeAvailableSpaceFromPixels converts a size in pixels into a definite available space.
func SizeAvailableSpaceFromPixels(size Size[Pixels]) SizeOf[AvailableSpace] {
	return SizeOf[AvailableSpace]{
		Width:  AvailableSpaceDefinite(size.Width),
		Height: AvailableSpaceDefinite(size.Height),
	}
}

func availableSpaceToTaffy(space AvailableSpace) taffy.AvailableSpace {
	switch space.kind {
	case availableSpaceDefinite:
		return taffy.AvailableSpaceDefinite(float32(space.definite))
	case availableSpaceMaxContent:
		return taffy.AvailableSpaceMaxContent
	}
	return taffy.AvailableSpaceMinContent
}

func availableSpaceFromTaffy(space taffy.AvailableSpace) AvailableSpace {
	switch space.Kind() {
	case taffy.AvailableSpaceKindDefinite:
		return AvailableSpaceDefinite(Pixels(space.Unwrap()))
	case taffy.AvailableSpaceKindMaxContent:
		return AvailableSpaceMaxContent()
	}
	return AvailableSpaceMinContent()
}

func sizeAvailableSpaceToTaffy(size SizeOf[AvailableSpace]) taffy.Size[taffy.AvailableSpace] {
	return taffy.Size[taffy.AvailableSpace]{Width: availableSpaceToTaffy(size.Width), Height: availableSpaceToTaffy(size.Height)}
}

func sizeAvailableSpaceFromTaffy(size taffy.Size[taffy.AvailableSpace]) SizeOf[AvailableSpace] {
	return SizeOf[AvailableSpace]{Width: availableSpaceFromTaffy(size.Width), Height: availableSpaceFromTaffy(size.Height)}
}

func alignItemsToTaffy(value AlignItems) taffy.AlignItems {
	switch value {
	case AlignItemsStart:
		return taffy.AlignItemsStart
	case AlignItemsEnd:
		return taffy.AlignItemsEnd
	case AlignItemsFlexStart:
		return taffy.AlignItemsFlexStart
	case AlignItemsFlexEnd:
		return taffy.AlignItemsFlexEnd
	case AlignItemsCenter:
		return taffy.AlignItemsCenter
	case AlignItemsBaseline:
		return taffy.AlignItemsBaseline
	case AlignItemsStretch:
		return taffy.AlignItemsStretch
	}
	panic(fmt.Sprintf("gpui: invalid AlignItems %d", value))
}

func alignContentToTaffy(value AlignContent) taffy.AlignContent {
	switch value {
	case AlignContentStart:
		return taffy.AlignContentStart
	case AlignContentEnd:
		return taffy.AlignContentEnd
	case AlignContentFlexStart:
		return taffy.AlignContentFlexStart
	case AlignContentFlexEnd:
		return taffy.AlignContentFlexEnd
	case AlignContentCenter:
		return taffy.AlignContentCenter
	case AlignContentStretch:
		return taffy.AlignContentStretch
	case AlignContentSpaceBetween:
		return taffy.AlignContentSpaceBetween
	case AlignContentSpaceEvenly:
		return taffy.AlignContentSpaceEvenly
	case AlignContentSpaceAround:
		return taffy.AlignContentSpaceAround
	}
	panic(fmt.Sprintf("gpui: invalid AlignContent %d", value))
}

func displayToTaffy(value Display) taffy.Display {
	switch value {
	case DisplayBlock:
		return taffy.DisplayBlock
	case DisplayFlex:
		return taffy.DisplayFlex
	case DisplayGrid:
		return taffy.DisplayGrid
	case DisplayNone:
		return taffy.DisplayNone
	}
	panic(fmt.Sprintf("gpui: invalid Display %d", value))
}

func flexWrapToTaffy(value FlexWrap) taffy.FlexWrap {
	switch value {
	case FlexWrapNoWrap:
		return taffy.FlexWrapNoWrap
	case FlexWrapWrap:
		return taffy.FlexWrapWrap
	case FlexWrapWrapReverse:
		return taffy.FlexWrapWrapReverse
	}
	panic(fmt.Sprintf("gpui: invalid FlexWrap %d", value))
}

func flexDirectionToTaffy(value FlexDirection) taffy.FlexDirection {
	switch value {
	case FlexDirectionRow:
		return taffy.FlexDirectionRow
	case FlexDirectionColumn:
		return taffy.FlexDirectionColumn
	case FlexDirectionRowReverse:
		return taffy.FlexDirectionRowReverse
	case FlexDirectionColumnReverse:
		return taffy.FlexDirectionColumnReverse
	}
	panic(fmt.Sprintf("gpui: invalid FlexDirection %d", value))
}

func overflowToTaffy(value Overflow) taffy.Overflow {
	switch value {
	case OverflowVisible:
		return taffy.OverflowVisible
	case OverflowClip:
		return taffy.OverflowClip
	case OverflowHidden:
		return taffy.OverflowHidden
	case OverflowScroll:
		return taffy.OverflowScroll
	}
	panic(fmt.Sprintf("gpui: invalid Overflow %d", value))
}

func positionToTaffy(value Position) taffy.Position {
	switch value {
	case PositionRelative:
		return taffy.PositionRelative
	case PositionAbsolute:
		return taffy.PositionAbsolute
	}
	panic(fmt.Sprintf("gpui: invalid Position %d", value))
}

func gridPlacementToTaffy(placement GridPlacement) taffy.GridPlacement[string] {
	if index, ok := placement.AsLine(); ok {
		return taffy.GridPlacement[string]{}.FromLineIndex(index)
	}
	if span, ok := placement.AsSpan(); ok {
		return taffy.GridPlacement[string]{}.FromSpan(span)
	}
	return taffy.GridPlacementAuto[string]()
}
