package gpui

import (
	"fmt"
	"math"
)

// NewUniformList provides lazy rendering for a set of items that are of uniform height. When
// rendered into a container with overflow-y: hidden and a fixed (or max) height, the list will
// only render the visible subset of items.
//
// Rather than use the full taffy layout system, the list measures the first element and then
// lays out all remaining elements in a line based on that measurement. This is much faster than
// the full layout system, but only works for elements with uniform height.
func NewUniformList[R IntoElement, I ElementIDSource](id I, itemCount int, f func(visibleRange Range, window *Window, cx *App) []R) *UniformList {
	var baseStyle StyleRefinement
	baseStyle.Overflow.Y = Some(OverflowScroll)

	renderRange := func(visibleRange Range, window *Window, cx *App) []AnyElement {
		components := f(visibleRange, window, cx)
		elements := make([]AnyElement, 0, len(components))
		for _, component := range components {
			elements = append(elements, component.IntoAnyElement())
		}
		return elements
	}

	l := &UniformList{
		itemCount:                itemCount,
		itemToMeasureIndex:       0,
		renderItems:              renderRange,
		sizingBehavior:           ListSizingBehaviorAuto,
		horizontalSizingBehavior: ListHorizontalSizingBehaviorFitList,
		scrollHandle:             None[UniformListScrollHandle](),
		interactivity:            Interactivity{sourceLocation: CallerSourceLocation(1)},
	}
	l.interactivity.ElementID = Some(NewElementID(id))
	l.interactivity.BaseStyle = baseStyle
	BindStyled(&l.Styled, l)
	BindInteractive(&l.Interactive, l)
	BindInteractiveID(&l.InteractiveID, l)
	BindFluent(&l.Fluent, l)
	return l
}

// UniformList is a list element for efficiently laying out and displaying a list of
// uniform-height elements.
type UniformList struct {
	Styled[*UniformList]
	Interactive[*UniformList]
	InteractiveID[*UniformList]
	Fluent[*UniformList]
	itemCount                int
	itemToMeasureIndex       int
	renderItems              func(visibleRange Range, window *Window, cx *App) []AnyElement
	decorations              []UniformListDecoration
	interactivity            Interactivity
	scrollHandle             Option[UniformListScrollHandle]
	sizingBehavior           ListSizingBehavior
	horizontalSizingBehavior ListHorizontalSizingBehavior
}

// UniformListFrameState is the frame state used by the UniformList.
type UniformListFrameState struct {
	items       []AnyElement
	decorations []AnyElement
}

// UniformListScrollHandle is a handle for controlling the scroll position of a uniform list.
// This should be stored in your view and passed to the list on each frame. Copies share the
// state, as upstream's Rc clones do; create one with NewUniformListScrollHandle.
type UniformListScrollHandle struct {
	// State is the shared scroll state (upstream's tuple field .0).
	State *UniformListScrollState
}

// ScrollStrategy is where to place the element scrolled to.
type ScrollStrategy uint8

const (
	// ScrollStrategyTop places the element at the top of the list's viewport.
	ScrollStrategyTop ScrollStrategy = iota
	// ScrollStrategyCenter attempts to place the element in the middle of the list's viewport.
	// May not be possible if there's not enough list items above the item scrolled to: in this
	// case, the element will be placed at the closest possible position.
	ScrollStrategyCenter
	// ScrollStrategyBottom attempts to place the element at the bottom of the list's viewport.
	// May not be possible if there's not enough list items above the item scrolled to: in this
	// case, the element will be placed at the closest possible position.
	ScrollStrategyBottom
	// ScrollStrategyNearest, if the element is not visible, attempts to place it at the top of
	// the list's viewport if the target element is above currently visible elements, or at the
	// bottom if it is below them.
	ScrollStrategyNearest
)

var scrollStrategyNames = [...]string{"Top", "Center", "Bottom", "Nearest"}

// String returns the variant name.
func (s ScrollStrategy) String() string {
	if int(s) < len(scrollStrategyNames) {
		return scrollStrategyNames[s]
	}
	return fmt.Sprintf("ScrollStrategy(%d)", uint8(s))
}

// DeferredScrollToItem is a scroll request applied during the next layout.
type DeferredScrollToItem struct {
	// ItemIndex is the item index to scroll to.
	ItemIndex int
	// Strategy is the scroll strategy to use.
	Strategy ScrollStrategy
	// Offset is the offset in number of items.
	Offset int
	// ScrollStrict is whether to scroll even when the item is already visible.
	ScrollStrict bool
}

// UniformListScrollState is the state behind a UniformListScrollHandle.
type UniformListScrollState struct {
	// BaseHandle is the scroll handle tracking the list's offset.
	BaseHandle ScrollHandle
	// DeferredScrollToItem is the pending scroll request.
	DeferredScrollToItem Option[DeferredScrollToItem]
	// LastItemSize is the size of the item, captured during last layout.
	LastItemSize Option[ItemSize]
	// YFlipped is whether the list was vertically flipped during last layout.
	YFlipped bool
}

// ItemSize is the size of the item and its contents.
type ItemSize struct {
	// Item is the size of the item.
	Item Size[Pixels]
	// Contents is the size of the item's contents, which may be larger than the item itself,
	// if the item was bounded by a parent element.
	Contents Size[Pixels]
}

// NewUniformListScrollHandle creates a new scroll handle to bind to a uniform list.
func NewUniformListScrollHandle() UniformListScrollHandle {
	return UniformListScrollHandle{State: &UniformListScrollState{
		BaseHandle:           NewScrollHandle(),
		DeferredScrollToItem: None[DeferredScrollToItem](),
		LastItemSize:         None[ItemSize](),
		YFlipped:             false,
	}}
}

// ScrollToItem scrolls the list so that the given item index is visible.
//
// This uses non-strict scrolling: if the item is already fully visible, no scrolling occurs. If
// the item is out of view, it scrolls the minimum amount to bring it into view according to the
// strategy.
func (h UniformListScrollHandle) ScrollToItem(ix int, strategy ScrollStrategy) {
	h.State.DeferredScrollToItem = Some(DeferredScrollToItem{ItemIndex: ix, Strategy: strategy, Offset: 0, ScrollStrict: false})
}

// ScrollToItemStrict scrolls the list so that the given item index is at scroll strategy
// position.
//
// This uses strict scrolling: the item will always be scrolled to match the strategy position,
// even if it's already visible. Use this when you need precise positioning.
func (h UniformListScrollHandle) ScrollToItemStrict(ix int, strategy ScrollStrategy) {
	h.State.DeferredScrollToItem = Some(DeferredScrollToItem{ItemIndex: ix, Strategy: strategy, Offset: 0, ScrollStrict: true})
}

// ScrollToItemWithOffset scrolls the list to the given item index with an offset in number of
// items.
//
// This uses non-strict scrolling: if the item is already visible within the offset region, no
// scrolling occurs.
//
// The offset parameter shrinks the effective viewport by the specified number of items from the
// corresponding edge, then applies the scroll strategy within that reduced viewport:
// ScrollStrategyTop shrinks from top and positions the item at the new top, ScrollStrategyCenter
// shrinks from top and centers the item in the reduced viewport, ScrollStrategyBottom shrinks
// from bottom and positions the item at the new bottom.
func (h UniformListScrollHandle) ScrollToItemWithOffset(ix int, strategy ScrollStrategy, offset int) {
	h.State.DeferredScrollToItem = Some(DeferredScrollToItem{ItemIndex: ix, Strategy: strategy, Offset: offset, ScrollStrict: false})
}

// ScrollToItemStrictWithOffset scrolls the list so that the given item index is at the exact
// scroll strategy position with an offset.
//
// This uses strict scrolling: the item will always be scrolled to match the strategy position,
// even if it's already visible. The offset shrinks the viewport as in ScrollToItemWithOffset.
func (h UniformListScrollHandle) ScrollToItemStrictWithOffset(ix int, strategy ScrollStrategy, offset int) {
	h.State.DeferredScrollToItem = Some(DeferredScrollToItem{ItemIndex: ix, Strategy: strategy, Offset: offset, ScrollStrict: true})
}

// YFlipped reports whether the list is flipped vertically.
func (h UniformListScrollHandle) YFlipped() bool {
	return h.State.YFlipped
}

// LogicalScrollTopIndex returns the index of the topmost visible child.
func (h UniformListScrollHandle) LogicalScrollTopIndex() int {
	if deferred, ok := h.State.DeferredScrollToItem.Get(); ok {
		return deferred.ItemIndex
	}
	ix, _ := h.State.BaseHandle.LogicalScrollTop()
	return ix
}

// IsScrollable reports whether the list can be scrolled vertically.
func (h UniformListScrollHandle) IsScrollable() bool {
	if size, ok := h.State.LastItemSize.Get(); ok {
		return size.Contents.Height > size.Item.Height
	}
	return false
}

// IsScrolledToEnd reports whether the list is scrolled to the end; ok is false if the list is
// not scrollable.
func (h UniformListScrollHandle) IsScrolledToEnd() (scrolledToEnd bool, ok bool) {
	state := h.State
	maxOffset := state.BaseHandle.MaxOffset()
	if maxOffset.Y <= Px(0.) {
		return false, false
	}
	offset := state.BaseHandle.Offset()
	return -offset.Y >= maxOffset.Y, true
}

// ScrollToBottom scrolls to the bottom of the list.
func (h UniformListScrollHandle) ScrollToBottom() {
	h.ScrollToItem(math.MaxInt, ScrollStrategyBottom)
}

// Style returns the list's base style refinement.
func (l *UniformList) Style() *StyleRefinement {
	return &l.interactivity.BaseStyle
}

// Interactivity returns the list's interactivity.
func (l *UniformList) Interactivity() *Interactivity {
	return &l.interactivity
}

// ElementID returns the list's element id.
func (l *UniformList) ElementID() (ElementID, bool) {
	return l.interactivity.ElementID.Get()
}

// SourceLocation returns nil.
func (l *UniformList) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout measures an item and requests the list's layout.
func (l *UniformList) RequestLayout(globalID *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, UniformListFrameState) {
	maxItems := l.itemCount
	itemSize := l.measureItem(None[Pixels](), window, cx)
	layoutID := l.interactivity.RequestLayout(globalID, inspectorID, window, cx, func(style Style, window *Window, cx *App) LayoutID {
		switch l.sizingBehavior {
		case ListSizingBehaviorInfer:
			return window.WithTextStyle(optionalTextStyle(&style), func(window *Window) LayoutID {
				return window.RequestMeasuredLayout(style, func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], _ *Window, _ *App) Size[Pixels] {
					desiredHeight := pixelsMulUsize(itemSize.Height, maxItems)
					width, ok := knownDimensions.Width.Get()
					if !ok {
						if x, definite := availableSpace.Width.AsDefinite(); definite {
							width = x
						} else {
							width = itemSize.Width
						}
					}
					height := desiredHeight
					if h, definite := availableSpace.Height.AsDefinite(); definite {
						height = min(desiredHeight, h)
					}
					return NewSize(width, height)
				})
			})
		default:
			return window.WithTextStyle(optionalTextStyle(&style), func(window *Window) LayoutID {
				return window.RequestLayout(style, nil, cx)
			})
		}
	})

	return layoutID, UniformListFrameState{}
}

// Prepaint lays out and prepaints the visible items and decorations.
func (l *UniformList) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], frameState *UniformListFrameState, window *Window, cx *App) Option[Hitbox] {
	style := l.interactivity.ComputeStyle(globalID, nil, window, cx)
	border := EdgesAbsoluteLengthToPixels(style.BorderWidths, window.RemSize())
	padding := EdgesDefiniteLengthToPixels(style.Padding, SizeAbsoluteLengthFromPixels(bounds.Size), window.RemSize())

	paddedBounds := BoundsFromCorners(
		bounds.Origin.Add(NewPoint(border.Left+padding.Left, border.Top+padding.Top)),
		bounds.BottomRight().Sub(NewPoint(border.Right+padding.Right, border.Bottom+padding.Bottom)),
	)

	canScrollHorizontally := l.horizontalSizingBehavior == ListHorizontalSizingBehaviorUnconstrained

	longestItemSize := l.measureItem(None[Pixels](), window, cx)
	contentWidth := paddedBounds.Size.Width
	if canScrollHorizontally {
		contentWidth = max(paddedBounds.Size.Width, longestItemSize.Width)
	}
	contentSize := Size[Pixels]{Width: contentWidth, Height: pixelsMulUsize(longestItemSize.Height, l.itemCount)}

	sharedScrollOffset := l.interactivity.scrollOffset
	if sharedScrollOffset == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	itemHeight := longestItemSize.Height
	sharedScrollToItem := None[DeferredScrollToItem]()
	if handle, ok := l.scrollHandle.Get(); ok {
		handle.State.LastItemSize = Some(ItemSize{Item: paddedBounds.Size, Contents: contentSize})
		sharedScrollToItem = handle.State.DeferredScrollToItem.Take()
	}

	return l.interactivity.Prepaint(globalID, inspectorID, bounds, contentSize, window, cx, func(_ *Style, scrollOffset Point[Pixels], hitbox Option[Hitbox], window *Window, cx *App) Option[Hitbox] {
		yFlipped := false
		if scrollHandle, ok := l.scrollHandle.Get(); ok {
			yFlipped = scrollHandle.State.YFlipped
		}

		if l.itemCount > 0 {
			contentHeight := pixelsMulUsize(itemHeight, l.itemCount)

			isScrolledVertically := !scrollOffset.Y.IsZero()
			maxScrollOffset := paddedBounds.Size.Height - contentHeight

			if isScrolledVertically && scrollOffset.Y < maxScrollOffset {
				sharedScrollOffset.Y = maxScrollOffset
				scrollOffset.Y = maxScrollOffset
			}

			contentWidth := contentSize.Width + padding.Left + padding.Right
			isScrolledHorizontally := canScrollHorizontally && !scrollOffset.X.IsZero()
			if isScrolledHorizontally && contentWidth <= paddedBounds.Size.Width {
				sharedScrollOffset.X = PixelsZero
				scrollOffset.X = PixelsZero
			}

			if deferred, ok := sharedScrollToItem.Get(); ok {
				itemIndex := deferred.ItemIndex
				strategy := deferred.Strategy
				if yFlipped {
					if next := uint(itemIndex) + 1; next >= uint(l.itemCount) {
						itemIndex = 0
					} else {
						itemIndex = l.itemCount - int(next)
					}
				}
				listHeight := paddedBounds.Size.Height
				updatedScrollOffset := sharedScrollOffset
				itemTop := pixelsMulUsize(itemHeight, itemIndex)
				itemBottom := itemTop + itemHeight
				scrollTop := -updatedScrollOffset.Y
				offsetPixels := pixelsMulUsize(itemHeight, deferred.Offset)

				isAbove := itemTop < scrollTop+offsetPixels
				isBelow := itemBottom > scrollTop+listHeight

				if deferred.ScrollStrict || isAbove || isBelow {
					if strategy == ScrollStrategyNearest {
						if isAbove {
							strategy = ScrollStrategyTop
						} else if isBelow {
							strategy = ScrollStrategyBottom
						}
					}

					maxScrollOffset := max(contentHeight-listHeight, PixelsZero)
					switch strategy {
					case ScrollStrategyTop:
						updatedScrollOffset.Y = -clampPixels(itemTop-offsetPixels, PixelsZero, maxScrollOffset)
					case ScrollStrategyCenter:
						itemCenter := itemTop + itemHeight/2.0

						viewportHeight := listHeight - offsetPixels
						viewportCenter := offsetPixels + viewportHeight/2.0
						targetScrollTop := itemCenter - viewportCenter
						updatedScrollOffset.Y = -clampPixels(targetScrollTop, PixelsZero, maxScrollOffset)
					case ScrollStrategyBottom:
						updatedScrollOffset.Y = -clampPixels(itemBottom-listHeight, PixelsZero, maxScrollOffset)
					case ScrollStrategyNearest:
					}
				}
				scrollOffset = *updatedScrollOffset
			}

			scrollTop := -scrollOffset.Y.ToF64()
			itemHeightF64 := itemHeight.ToF64()
			firstVisibleElementIx := f64AsUsize(math.Floor((scrollTop - padding.Top.ToF64()) / itemHeightF64))
			lastVisibleElementIx := f64AsUsize(math.Ceil((scrollTop + paddedBounds.Size.Height.ToF64()) / itemHeightF64))

			visibleRange := Range{Start: firstVisibleElementIx, End: min(lastVisibleElementIx, l.itemCount)}

			var items []AnyElement
			if yFlipped {
				flippedRange := Range{Start: saturatingSubUsize(l.itemCount, visibleRange.End), End: saturatingSubUsize(l.itemCount, visibleRange.Start)}
				items = l.renderItems(flippedRange, window, cx)
				for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
					items[i], items[j] = items[j], items[i]
				}
			} else {
				items = l.renderItems(visibleRange, window, cx)
			}

			contentMask := ContentMask[Pixels]{Bounds: bounds}
			window.WithContentMask(Some(contentMask), func(window *Window) struct{} {
				for i := range items {
					ix := visibleRange.Start + i
					if ix >= visibleRange.End {
						break
					}
					item := items[i]
					itemTop := itemHeightF64*float64(ix) - scrollTop
					itemOrigin := paddedBounds.Origin.Add(NewPoint(scrollOffset.X, Pixels(float32(itemTop))))

					availableWidth := paddedBounds.Size.Width
					if canScrollHorizontally {
						availableWidth = paddedBounds.Size.Width + scrollOffset.X.Abs()
					}
					availableSpace := NewSizeOf(AvailableSpaceDefinite(availableWidth), AvailableSpaceDefinite(itemHeight))
					item.LayoutAsRoot(availableSpace, window, cx)
					item.PrepaintAt(itemOrigin, window, cx)
					frameState.items = append(frameState.items, item)
				}

				bounds := NewBounds(paddedBounds.Origin.Add(scrollOffset), paddedBounds.Size)
				for _, decoration := range l.decorations {
					element := decoration.Compute(visibleRange, bounds, scrollOffset, itemHeight, l.itemCount, window, cx)
					availableSpace := NewSizeOf(AvailableSpaceDefinite(bounds.Size.Width), AvailableSpaceDefinite(bounds.Size.Height))
					element.LayoutAsRoot(availableSpace, window, cx)
					element.PrepaintAt(bounds.Origin, window, cx)
					frameState.decorations = append(frameState.decorations, element)
				}
				return struct{}{}
			})
		}

		return hitbox
	})
}

// Paint paints the list, its visible items and its decorations.
func (l *UniformList) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *UniformListFrameState, hitbox *Option[Hitbox], window *Window, cx *App) {
	var hitboxRef *Hitbox
	if hitbox.IsSome() {
		hitboxRef = hitbox.AsMut()
	}
	l.interactivity.Paint(globalID, inspectorID, bounds, hitboxRef, window, cx, func(_ *Style, window *Window, cx *App) {
		for i := range requestLayout.items {
			requestLayout.items[i].Paint(window, cx)
		}
		for i := range requestLayout.decorations {
			requestLayout.decorations[i].Paint(window, cx)
		}
	})
}

// IntoAnyElement converts the list into an AnyElement.
func (l *UniformList) IntoAnyElement() AnyElement {
	return NewAnyElement[UniformListFrameState, Option[Hitbox]](l)
}

// UniformListDecoration is a decoration for a UniformList. This can be used for various things,
// such as rendering indent guides, or other visual effects.
type UniformListDecoration interface {
	// Compute computes the decoration element, given the visible range of list items, the
	// bounds of the list, and the height of each item.
	Compute(visibleRange Range, bounds Bounds[Pixels], scrollOffset Point[Pixels], itemHeight Pixels, itemCount int, window *Window, cx *App) AnyElement
}

type entityUniformListDecoration[T any, PT interface {
	*T
	UniformListDecoration
}] struct {
	entity Entity[T]
}

func (d entityUniformListDecoration[T, PT]) Compute(visibleRange Range, bounds Bounds[Pixels], scrollOffset Point[Pixels], itemHeight Pixels, itemCount int, window *Window, cx *App) AnyElement {
	return d.entity.UpdateR(cx, func(inner *T, cx *Context[T]) AnyElement {
		return PT(inner).Compute(visibleRange, bounds, scrollOffset, itemHeight, itemCount, window, cx.App)
	})
}

// UniformListDecorationFromEntity adapts an entity whose state is a decoration, upstream's impl
// of UniformListDecoration for Entity<T>. The adapter borrows entity; the caller keeps its count.
func UniformListDecorationFromEntity[T any, PT interface {
	*T
	UniformListDecoration
}](entity Entity[T]) UniformListDecoration {
	return entityUniformListDecoration[T, PT]{entity: entity}
}

// WithWidthFromItem selects a specific list item for measurement; None measures the first.
func (l *UniformList) WithWidthFromItem(itemIndex Option[int]) *UniformList {
	l.itemToMeasureIndex = itemIndex.UnwrapOr(0)
	return l
}

// WithSizingBehavior sets the sizing behavior, similar to the List element.
func (l *UniformList) WithSizingBehavior(behavior ListSizingBehavior) *UniformList {
	l.sizingBehavior = behavior
	return l
}

// WithHorizontalSizingBehavior sets the horizontal sizing behavior, controlling the way list
// items are laid out horizontally. With ListHorizontalSizingBehaviorUnconstrained, every item
// and the list itself will have the size of the widest item and lay out pushing the end slot to
// the right end.
func (l *UniformList) WithHorizontalSizingBehavior(behavior ListHorizontalSizingBehavior) *UniformList {
	l.horizontalSizingBehavior = behavior
	switch behavior {
	case ListHorizontalSizingBehaviorFitList:
		l.interactivity.BaseStyle.Overflow.X = None[Overflow]()
	case ListHorizontalSizingBehaviorUnconstrained:
		l.interactivity.BaseStyle.Overflow.X = Some(OverflowScroll)
	}
	return l
}

// WithDecoration adds a decoration element to the list.
func (l *UniformList) WithDecoration(decoration UniformListDecoration) *UniformList {
	l.decorations = append(l.decorations, decoration)
	return l
}

func (l *UniformList) measureItem(listWidth Option[Pixels], window *Window, cx *App) Size[Pixels] {
	if l.itemCount == 0 {
		return Size[Pixels]{}
	}

	itemIx := min(l.itemToMeasureIndex, l.itemCount-1)
	items := l.renderItems(Range{Start: itemIx, End: itemIx + 1}, window, cx)
	if len(items) == 0 {
		return Size[Pixels]{}
	}
	itemToMeasure := items[len(items)-1]
	width := AvailableSpaceMaxContent()
	if w, ok := listWidth.Get(); ok {
		width = AvailableSpaceDefinite(w)
	}
	availableSpace := NewSizeOf(width, AvailableSpaceMinContent())
	return itemToMeasure.LayoutAsRoot(availableSpace, window, cx)
}

// TrackUniformListScroll tracks and renders scroll state of this list with reference to the
// given scroll handle; it is upstream's UniformList::track_scroll.
func (l *UniformList) TrackUniformListScroll(handle UniformListScrollHandle) *UniformList {
	l.interactivity.trackedScrollHandle = Some(handle.State.BaseHandle)
	l.scrollHandle = Some(handle)
	return l
}

// YFlipped sets whether the list is flipped vertically, such that item 0 appears at the bottom.
func (l *UniformList) YFlipped(yFlipped bool) *UniformList {
	if scrollHandle, ok := l.scrollHandle.Get(); ok {
		scrollState := scrollHandle.State
		baseHandle := scrollState.BaseHandle
		offset := baseHandle.Offset()
		lastSize, hasLastSize := scrollState.LastItemSize.Get()
		switch {
		case hasLastSize && scrollState.YFlipped != yFlipped:
			newYOffset := -(offset.Y + lastSize.Contents.Height - lastSize.Item.Height)
			baseHandle.SetOffset(NewPoint(offset.X, newYOffset))
			scrollState.YFlipped = yFlipped
		case !hasLastSize && yFlipped:
			baseHandle.SetOffset(NewPoint(offset.X, PixelsMin))
			scrollState.YFlipped = yFlipped
		}
	}
	return l
}

func pixelsMulUsize(p Pixels, n int) Pixels {
	return p * Pixels(float32(n))
}

func saturatingSubUsize(a, b int) int {
	if b >= a {
		return 0
	}
	return a - b
}

func f64AsUsize(v float64) int {
	if !(v > 0) {
		return 0
	}
	if v >= math.MaxInt {
		return math.MaxInt
	}
	return int(v)
}
