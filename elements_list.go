package gpui

import (
	"fmt"

	"github.com/megascan/gpui-go/sumtree"
)

type listRenderItemFn = func(ix int, window *Window, cx *App) AnyElement

type listUnit = struct{}

type listItemTree = sumtree.SumTree[listItem, listItemSummary, listUnit]

// NewList constructs a new list element.
func NewList(state ListState, renderItem func(ix int, window *Window, cx *App) AnyElement) *List {
	l := &List{
		state:          state,
		renderItem:     renderItem,
		style:          StyleRefinement{},
		sizingBehavior: ListSizingBehaviorAuto,
	}
	BindStyled(&l.Styled, l)
	BindFluent(&l.Fluent, l)
	return l
}

// List is a list element that can be used to render a large number of differently sized
// elements efficiently.
type List struct {
	Styled[*List]
	Fluent[*List]
	state          ListState
	renderItem     listRenderItemFn
	style          StyleRefinement
	sizingBehavior ListSizingBehavior
}

// WithSizingBehavior sets the sizing behavior for the list.
func (l *List) WithSizingBehavior(behavior ListSizingBehavior) *List {
	l.sizingBehavior = behavior
	return l
}

// ListState is the list state that views must hold on behalf of the list element. Copies share
// the state, as upstream's Rc clones do; create one with NewListState.
type ListState struct {
	inner *listStateInner
}

// GoString formats the state like upstream's Debug.
func (s ListState) GoString() string {
	return "ListState"
}

type listStateInner struct {
	lastLayoutBounds         Option[Bounds[Pixels]]
	lastPadding              Option[Edges[Pixels]]
	items                    listItemTree
	scrollAnchor             Option[listScrollAnchor]
	alignment                ListAlignment
	overdraw                 Pixels
	reset                    bool
	scrollHandler            func(event *ListScrollEvent, window *Window, cx *App)
	scrollbarDragStartHeight Option[Pixels]
	measuringBehavior        ListMeasuringBehavior
	itemMeasurementReady     func(index int, cx *App) bool
	followState              listFollowState
}

type listScrollAnchor struct {
	position    ListOffset
	pending     Option[listPendingScroll]
	provisional bool
	pendingEnd  bool
}

type listPendingScroll struct {
	proportional bool
	absolute     Pixels
	fraction     float32
}

func listPendingScrollAbsolute(offset Pixels) listPendingScroll {
	return listPendingScroll{absolute: offset}
}

func listPendingScrollProportional(fraction float32) listPendingScroll {
	return listPendingScroll{proportional: true, fraction: fraction}
}

type listScrollAnchorMode uint8

const (
	listScrollAnchorModeAbsolute listScrollAnchorMode = iota
	listScrollAnchorModeProportional
)

// FollowMode controls whether the list automatically follows new content at the end.
type FollowMode uint8

const (
	// FollowModeNormal is normal scrolling, with no automatic following.
	FollowModeNormal FollowMode = iota
	// FollowModeTail makes the list auto-scroll along with the tail, when scrolled to bottom.
	FollowModeTail
)

// String returns the variant name.
func (m FollowMode) String() string {
	switch m {
	case FollowModeNormal:
		return "Normal"
	case FollowModeTail:
		return "Tail"
	}
	return fmt.Sprintf("FollowMode(%d)", uint8(m))
}

type listFollowState struct {
	tail        bool
	isFollowing bool
}

func (f listFollowState) isFollowingTail() bool {
	return f.tail && f.isFollowing
}

func (f listFollowState) hasStoppedFollowing() bool {
	return f.tail && !f.isFollowing
}

func (f *listFollowState) startFollowing() {
	if f.tail && !f.isFollowing {
		*f = listFollowState{tail: true, isFollowing: true}
	}
}

func (f *listFollowState) stopFollowing() {
	if f.tail && f.isFollowing {
		*f = listFollowState{tail: true, isFollowing: false}
	}
}

// ListAlignment is whether the list is scrolling from top to bottom or bottom to top.
type ListAlignment uint8

const (
	// ListAlignmentTop means the list is scrolling from top to bottom, like most lists.
	ListAlignmentTop ListAlignment = iota
	// ListAlignmentBottom means the list is scrolling from bottom to top, like a chat log.
	ListAlignmentBottom
)

// String returns the variant name.
func (a ListAlignment) String() string {
	switch a {
	case ListAlignmentTop:
		return "Top"
	case ListAlignmentBottom:
		return "Bottom"
	}
	return fmt.Sprintf("ListAlignment(%d)", uint8(a))
}

// ListScrollEvent is a scroll event that has been converted to be in terms of the list's items.
type ListScrollEvent struct {
	// VisibleRange is the range of items currently visible in the list, after applying the
	// scroll event.
	VisibleRange Range
	// Count is the number of items that are currently visible in the list, after applying the
	// scroll event.
	Count int
	// IsScrolled is whether the list has been scrolled.
	IsScrolled bool
	// IsFollowingTail is whether the list is currently in follow-tail mode (auto-scrolling to
	// end).
	IsFollowingTail bool
}

// ListSizingBehavior is the sizing behavior to apply during layout. The zero value is
// upstream's default, Auto.
type ListSizingBehavior uint8

const (
	// ListSizingBehaviorAuto means the list should not calculate a fixed size.
	ListSizingBehaviorAuto ListSizingBehavior = iota
	// ListSizingBehaviorInfer means the list should calculate its size based on the size of its items.
	ListSizingBehaviorInfer
)

// String returns the variant name.
func (b ListSizingBehavior) String() string {
	switch b {
	case ListSizingBehaviorAuto:
		return "Auto"
	case ListSizingBehaviorInfer:
		return "Infer"
	}
	return fmt.Sprintf("ListSizingBehavior(%d)", uint8(b))
}

// ListMeasuringBehavior is the measuring behavior to apply during layout. The zero value is
// upstream's default, Visible.
type ListMeasuringBehavior struct {
	measure     bool
	hasMeasured bool
}

// ListMeasuringBehaviorMeasure measures all items in the list. This can be expensive for the
// first frame in a large list.
func ListMeasuringBehaviorMeasure(hasMeasured bool) ListMeasuringBehavior {
	return ListMeasuringBehavior{measure: true, hasMeasured: hasMeasured}
}

// ListMeasuringBehaviorVisible only measures visible items.
func ListMeasuringBehaviorVisible() ListMeasuringBehavior {
	return ListMeasuringBehavior{}
}

// AsMeasure returns the Measure variant's flag.
func (b ListMeasuringBehavior) AsMeasure() (hasMeasured bool, ok bool) {
	return b.hasMeasured, b.measure
}

func (b *ListMeasuringBehavior) reset() {
	if b.measure {
		b.hasMeasured = false
	}
}

// ListHorizontalSizingBehavior is the horizontal sizing behavior to apply during layout.
type ListHorizontalSizingBehavior uint8

const (
	// ListHorizontalSizingBehaviorFitList means list items' width can never exceed the width of the list.
	ListHorizontalSizingBehaviorFitList ListHorizontalSizingBehavior = iota
	// ListHorizontalSizingBehaviorUnconstrained means list items' width may go over the width of the list, if any item is wider.
	ListHorizontalSizingBehaviorUnconstrained
)

// String returns the variant name.
func (b ListHorizontalSizingBehavior) String() string {
	switch b {
	case ListHorizontalSizingBehaviorFitList:
		return "FitList"
	case ListHorizontalSizingBehaviorUnconstrained:
		return "Unconstrained"
	}
	return fmt.Sprintf("ListHorizontalSizingBehavior(%d)", uint8(b))
}

type listLayoutItemsResponse struct {
	maxItemWidth Pixels
	scrollTop    ListOffset
	itemLayouts  []listItemLayout
}

type listItemLayout struct {
	index   int
	element AnyElement
	size    Size[Pixels]
}

// ListPrepaintState is the frame state used by the List element after layout.
type ListPrepaintState struct {
	hitbox Hitbox
	layout listLayoutItemsResponse
}

type listDeque[T any] struct {
	front []T
	back  []T
}

func (d *listDeque[T]) pushFront(v T) { d.front = append(d.front, v) }

func (d *listDeque[T]) pushBack(v T) { d.back = append(d.back, v) }

func (d *listDeque[T]) len() int { return len(d.front) + len(d.back) }

func (d *listDeque[T]) dropFront() { d.front = nil }

func (d *listDeque[T]) items() []T {
	out := make([]T, 0, d.len())
	for i := len(d.front) - 1; i >= 0; i-- {
		out = append(out, d.front[i])
	}
	return append(out, d.back...)
}

type listItem struct {
	measured    bool
	size        Size[Pixels]
	sizeHint    Option[Size[Pixels]]
	focusHandle Option[FocusHandle]
}

func listItemUnmeasured(sizeHint Option[Size[Pixels]], focusHandle Option[FocusHandle]) listItem {
	return listItem{sizeHint: sizeHint, focusHandle: focusHandle}
}

func listItemMeasured(size Size[Pixels], focusHandle Option[FocusHandle]) listItem {
	return listItem{measured: true, size: size, focusHandle: focusHandle}
}

func (i listItem) withMeasurement(size Size[Pixels], ready bool) listItem {
	if ready {
		return listItemMeasured(size, i.focusHandle)
	}
	return listItemUnmeasured(Some(size), i.focusHandle)
}

func (i listItem) itemSize() (Size[Pixels], bool) {
	if i.measured {
		return i.size, true
	}
	return Size[Pixels]{}, false
}

func (i listItem) itemSizeHint() (Size[Pixels], bool) {
	if i.measured {
		return i.size, true
	}
	return i.sizeHint.Get()
}

func (i listItem) containsFocused(window *Window, cx *App) bool {
	handle, ok := i.focusHandle.Get()
	return ok && handle.ContainsFocused(window, cx)
}

func (i listItem) GoString() string {
	if i.measured {
		return fmt.Sprintf("Rendered { size: %#v }", i.size)
	}
	return "Unrendered"
}

func (i listItem) Summary(listUnit) listItemSummary {
	if i.measured {
		return listItemSummary{
			count:            1,
			renderedCount:    1,
			unrenderedCount:  0,
			height:           i.size.Height,
			hasFocusHandles:  i.focusHandle.IsSome(),
			hasUnknownHeight: false,
		}
	}
	height := PixelsZero
	if hint, ok := i.sizeHint.Get(); ok {
		height = hint.Height
	}
	return listItemSummary{
		count:            1,
		renderedCount:    0,
		unrenderedCount:  1,
		height:           height,
		hasFocusHandles:  i.focusHandle.IsSome(),
		hasUnknownHeight: i.sizeHint.IsNone(),
	}
}

type listItemSummary struct {
	count            int
	renderedCount    int
	unrenderedCount  int
	height           Pixels
	hasFocusHandles  bool
	hasUnknownHeight bool
}

func (listItemSummary) Zero(listUnit) listItemSummary { return listItemSummary{} }

func (s listItemSummary) AddSummary(summary listItemSummary, _ listUnit) listItemSummary {
	s.count += summary.count
	s.renderedCount += summary.renderedCount
	s.unrenderedCount += summary.unrenderedCount
	s.height += summary.height
	s.hasFocusHandles = s.hasFocusHandles || summary.hasFocusHandles
	s.hasUnknownHeight = s.hasUnknownHeight || summary.hasUnknownHeight
	return s
}

type listCount int

func (listCount) Zero(listUnit) listCount { return 0 }

func (c listCount) AddSummary(summary listItemSummary, _ listUnit) listCount {
	return c + listCount(summary.count)
}

func (c listCount) Cmp(cursorLocation listCount, _ listUnit) int {
	return compareInts(int(c), int(cursorLocation))
}

type listHeight Pixels

func (listHeight) Zero(listUnit) listHeight { return 0 }

func (h listHeight) AddSummary(summary listItemSummary, _ listUnit) listHeight {
	return h + listHeight(summary.height)
}

type listCountSeek int

func (c listCountSeek) Cmp(cursorLocation listItemSummary, _ listUnit) int {
	return compareInts(int(c), cursorLocation.count)
}

type listHeightSeek Pixels

func (h listHeightSeek) Cmp(cursorLocation listItemSummary, _ listUnit) int {
	return Pixels(h).Cmp(cursorLocation.height)
}

type listCountHeight = sumtree.Dimensions[listItemSummary, listUnit, listCount, listHeight, sumtree.Unit[listItemSummary, listUnit]]

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func newListItemTree(items []listItem) listItemTree {
	return sumtree.SumTreeFromIter[listItem, listItemSummary, listUnit](items, listUnit{})
}

func (s *listStateInner) countCursor() *sumtree.Cursor[listItem, listItemSummary, listUnit, listCount] {
	return s.items.Cursor[listCount](listUnit{})
}

func (s *listStateInner) summaryCursor() *sumtree.Cursor[listItem, listItemSummary, listUnit, listItemSummary] {
	return s.items.Cursor[listItemSummary](listUnit{})
}

// NewListState constructs a new list state, for storage on a view.
//
// The overdraw parameter controls how much extra space is rendered above and below the visible
// area. Elements within this area will be measured even though they are not visible. This can
// help ensure that the list doesn't flicker or pop in when scrolling.
func NewListState(itemCount int, alignment ListAlignment, overdraw Pixels) ListState {
	state := ListState{inner: &listStateInner{
		lastLayoutBounds:         None[Bounds[Pixels]](),
		lastPadding:              None[Edges[Pixels]](),
		items:                    sumtree.NewSumTree[listItem, listItemSummary, listUnit](listUnit{}),
		scrollAnchor:             None[listScrollAnchor](),
		alignment:                alignment,
		overdraw:                 overdraw,
		scrollHandler:            nil,
		reset:                    false,
		scrollbarDragStartHeight: None[Pixels](),
		measuringBehavior:        ListMeasuringBehaviorVisible(),
		itemMeasurementReady:     nil,
		followState:              listFollowState{},
	}}
	state.Splice(Range{Start: 0, End: 0}, itemCount)
	return state
}

// MeasureAll sets the list to measure all items in the list in the first layout phase.
//
// This is useful for ensuring that the scrollbar size is correct instead of based on only
// rendered elements.
func (s ListState) MeasureAll() ListState {
	s.inner.measuringBehavior = ListMeasuringBehaviorMeasure(false)
	return s
}

// WithUniformItemHeight pre-populates every unmeasured item with a uniform height hint so the
// scrollbar thumb is correctly sized from the first frame, without measuring all items up front.
//
// As items are actually rendered their real heights replace the hint, so the scrollbar converges
// to the exact size over time. This is a cheaper alternative to MeasureAll for lists where items
// have roughly uniform heights (e.g. table rows).
func (s ListState) WithUniformItemHeight(height Pixels) ListState {
	s.applyUniformItemHeight(height)
	return s
}

// Reset resets this instantiation of the list state.
//
// Note that this will cause scroll events to be dropped until the next paint.
func (s ListState) Reset(elementCount int) {
	state := s.inner
	state.reset = true
	state.measuringBehavior.reset()
	state.setScrollAnchor(None[ListOffset]())
	state.scrollbarDragStartHeight = None[Pixels]()
	oldCount := state.items.Summary().count

	s.Splice(Range{Start: 0, End: oldCount}, elementCount)
}

// ResetWithUniformHeight resets the list to elementCount items, pre-populating every item with a
// uniform height hint so the scrollbar thumb is correctly sized from the first frame even for
// off-screen items.
func (s ListState) ResetWithUniformHeight(elementCount int, height Pixels) {
	s.Reset(elementCount)
	s.applyUniformItemHeight(height)
}

func (s ListState) applyUniformItemHeight(height Pixels) {
	sizeHint := Size[Pixels]{Width: Px(0.), Height: height}
	state := s.inner
	var newItems []listItem
	for item := range state.items.Iter() {
		hint, ok := item.itemSizeHint()
		if !ok {
			hint = sizeHint
		}
		newItems = append(newItems, listItemUnmeasured(Some(hint), item.focusHandle))
	}
	tree := sumtree.NewSumTree[listItem, listItemSummary, listUnit](listUnit{})
	tree.Extend(newItems, listUnit{})
	state.items = tree
}

// Remeasure remeasures all items while preserving proportional scroll position.
//
// Use this when item heights may have changed (e.g., font size changes) but the number and
// identity of items remains the same.
func (s ListState) Remeasure() {
	count := s.ItemCount()
	s.remeasureItemsWithScrollAnchor(Range{Start: 0, End: count}, listScrollAnchorModeProportional)
}

// RemeasureItems marks items in itemRange as needing remeasurement while preserving the current
// scroll position. Unlike Splice, this does not change the number of items or blow away
// LogicalScrollTop.
//
// Use this when an item's content has changed and its rendered height may be different (e.g.,
// streaming text, tool results loading), but the item itself still exists at the same index.
func (s ListState) RemeasureItems(itemRange Range) {
	s.remeasureItemsWithScrollAnchor(itemRange, listScrollAnchorModeAbsolute)
}

func (s ListState) remeasureItemsWithScrollAnchor(itemRange Range, mode listScrollAnchorMode) {
	state := s.inner

	if anchor := state.scrollAnchor.AsMut(); anchor != nil && rangeContains(itemRange, anchor.position.ItemIx) {
		switch mode {
		case listScrollAnchorModeAbsolute:
			anchor.pending = Some(listPendingScrollAbsolute(anchor.position.OffsetInItem))
		case listScrollAnchorModeProportional:
			cursor := state.countCursor()
			cursor.Seek(listCount(anchor.position.ItemIx), sumtree.BiasRight)
			pending := anchor.pending
			if item, ok := cursor.Item(); ok {
				if size, ok := item.itemSize(); ok {
					var fraction float32
					if size.Height > Px(0.) {
						fraction = clampF32(float32(anchor.position.OffsetInItem/size.Height), 0., 1.)
					}
					pending = Some(listPendingScrollProportional(fraction))
				}
			}
			anchor.pending = pending
		}
	}

	cursor := state.countCursor()
	newItems := cursor.Slice(listCount(itemRange.Start), sumtree.BiasRight)
	invalidated := cursor.Slice(listCount(itemRange.End), sumtree.BiasRight)
	var replaced []listItem
	for item := range invalidated.Iter() {
		replaced = append(replaced, listItemUnmeasured(item.sizeHint0(), item.focusHandle))
	}
	newItems.Extend(replaced, listUnit{})
	newItems.Append(cursor.Suffix(), listUnit{})
	state.items = newItems
	state.measuringBehavior.reset()
}

func (i listItem) sizeHint0() Option[Size[Pixels]] {
	if hint, ok := i.itemSizeHint(); ok {
		return Some(hint)
	}
	return None[Size[Pixels]]()
}

// ItemCount returns the number of items in this list.
func (s ListState) ItemCount() int {
	return s.inner.items.Summary().count
}

// IsScrolledToEnd reports whether the list is scrolled to the end; ok is false if the list is not
// scrollable or the total content height is not yet known.
func (s ListState) IsScrolledToEnd() (scrolledToEnd bool, ok bool) {
	state := s.inner
	bounds, ok := state.lastLayoutBounds.Get()
	if !ok {
		return false, false
	}
	summary := state.items.Summary()
	if summary.hasUnknownHeight {
		return false, false
	}
	padding := state.lastPadding.UnwrapOrDefault()
	contentHeight := summary.height + padding.Top + padding.Bottom
	scrollMax := max(contentHeight-bounds.Size.Height, Px(0.))
	if scrollMax <= Px(0.) {
		return false, false
	}
	scrollTop := state.scrollTop(state.logicalScrollTop())
	return scrollTop >= scrollMax, true
}

// Splice informs the list state that the items in oldRange have been replaced by count new items
// that must be recalculated.
func (s ListState) Splice(oldRange Range, count int) {
	focusHandles := make([]Option[FocusHandle], count)
	for i := range focusHandles {
		focusHandles[i] = None[FocusHandle]()
	}
	s.SpliceFocusable(oldRange, focusHandles...)
}

// SpliceFocusable registers with the list state that the items in oldRange have been replaced by
// new items. As opposed to Splice, this method allows optional focus handles to be supplied to
// properly integrate with items in the list that can be focused. If a focused item is scrolled
// out of view, the list will continue to render it to allow keyboard interaction. The list takes
// over the handles and releases them when their items are removed.
func (s ListState) SpliceFocusable(oldRange Range, focusHandles ...Option[FocusHandle]) {
	state := s.inner

	oldItems := state.countCursor()
	newItems := oldItems.Slice(listCount(oldRange.Start), sumtree.BiasRight)
	removed := oldItems.Slice(listCount(oldRange.End), sumtree.BiasRight)

	splicedCount := 0
	spliced := make([]listItem, 0, len(focusHandles))
	for _, focusHandle := range focusHandles {
		splicedCount++
		spliced = append(spliced, listItemUnmeasured(None[Size[Pixels]](), focusHandle))
	}
	newItems.Extend(spliced, listUnit{})
	newItems.Append(oldItems.Suffix(), listUnit{})
	state.items = newItems
	for item := range removed.Iter() {
		if handle, ok := item.focusHandle.Get(); ok {
			handle.Release()
		}
	}

	if anchor := state.scrollAnchor.AsMut(); anchor != nil {
		if anchor.provisional && oldRange.Start <= anchor.position.ItemIx {
			anchor.provisional = false
			anchor.pending = None[listPendingScroll]()
		}
		if rangeContains(oldRange, anchor.position.ItemIx) {
			anchor.position = ListOffset{ItemIx: oldRange.Start, OffsetInItem: Px(0.)}
			anchor.pending = None[listPendingScroll]()
		} else if oldRange.End <= anchor.position.ItemIx {
			anchor.position.ItemIx = anchor.position.ItemIx - (oldRange.End - oldRange.Start) + splicedCount
		}
	}
}

// SetScrollHandler sets a handler that will be called when the list is scrolled.
func (s ListState) SetScrollHandler(handler func(event *ListScrollEvent, window *Window, cx *App)) {
	s.inner.scrollHandler = handler
}

// LogicalScrollTop returns the current scroll offset, in terms of the list's items.
func (s ListState) LogicalScrollTop() ListOffset {
	return s.inner.logicalScrollTop()
}

// ScrollBy scrolls the list by the given offset.
func (s ListState) ScrollBy(distance Pixels) {
	if distance == Px(0.) {
		return
	}

	currentOffset := s.LogicalScrollTop()
	state := s.inner

	if distance < Px(0.) {
		state.followState.stopFollowing()
	}

	cursor := state.summaryCursor()
	cursor.Seek(listCountSeek(currentOffset.ItemIx), sumtree.BiasRight)

	startPixelOffset := cursor.Start().height + currentOffset.OffsetInItem
	newPixelOffset := max(startPixelOffset+distance, Px(0.))
	if newPixelOffset > startPixelOffset {
		cursor.SeekForward(listHeightSeek(newPixelOffset), sumtree.BiasRight)
	} else {
		cursor.Seek(listHeightSeek(newPixelOffset), sumtree.BiasRight)
	}

	scrollTop := ListOffset{
		ItemIx:       cursor.Start().count,
		OffsetInItem: newPixelOffset - cursor.Start().height,
	}
	state.setScrollAnchor(Some(scrollTop))
}

// ScrollToEnd scrolls the list to the very end (past the last item).
//
// Unlike ScrollToRevealItem, this uses the total item count as the anchor, so the list's layout
// pass will walk backwards from the end and always show the bottom of the last item, even when
// that item is still growing (e.g. during streaming).
func (s ListState) ScrollToEnd() {
	state := s.inner
	itemCount := state.items.Summary().count
	state.setScrollAnchor(Some(ListOffset{ItemIx: itemCount, OffsetInItem: Px(0.)}))
}

// SetFollowMode sets the follow mode for the list. In Tail mode, the list will auto-scroll to
// the end and re-engage after the user scrolls back to the bottom. In Normal mode, no automatic
// following occurs.
func (s ListState) SetFollowMode(mode FollowMode) {
	if mode == FollowModeNormal {
		s.PauseFollowingTail()
	}
	state := s.inner

	switch mode {
	case FollowModeNormal:
		state.followState = listFollowState{}
	case FollowModeTail:
		state.followState = listFollowState{tail: true, isFollowing: true}
		itemCount := state.items.Summary().count
		state.setScrollAnchor(Some(ListOffset{ItemIx: itemCount, OffsetInItem: Px(0.)}))
	}
}

// PauseFollowingTail pauses tail-following, freezing the list at its current scroll position.
// Unlike SetFollowMode with FollowModeNormal, this keeps the list in Tail mode, so it will resume
// following automatically once the view returns to the bottom. No-op when the list isn't
// currently following.
//
// Useful when something other than the user grows an item (e.g. zooming a diagram) and the
// current position should stay put rather than snapping to the end.
func (s ListState) PauseFollowingTail() {
	state := s.inner
	if state.followState.isFollowingTail() {
		if anchor := state.scrollAnchor.AsMut(); anchor != nil {
			anchor.pendingEnd = false
		}
	}
	state.followState.stopFollowing()
}

// SetItemMeasurementReadiness sets a predicate queried after item layout to determine whether
// the measured height is stable. Measurements reported as false remain provisional and are
// retried on later layouts. The consumer must request a new frame when measurement becomes
// ready.
func (s ListState) SetItemMeasurementReadiness(ready func(index int, cx *App) bool) {
	s.inner.itemMeasurementReady = ready
}

// IsFollowingTail returns whether the list is currently actively following the tail (snapping
// to the end on each layout).
func (s ListState) IsFollowingTail() bool {
	return s.inner.followState.isFollowingTail()
}

// ScrollTo scrolls the list to the given offset.
func (s ListState) ScrollTo(scrollTop ListOffset) {
	state := s.inner
	itemCount := state.items.Summary().count
	if scrollTop.ItemIx >= itemCount {
		scrollTop.ItemIx = itemCount
		scrollTop.OffsetInItem = Px(0.)
	}

	if scrollTop.ItemIx < itemCount {
		state.followState.stopFollowing()
	}

	state.setScrollAnchor(Some(scrollTop))
}

// ScrollToRevealItem scrolls the list to the given item, such that the item is fully visible.
func (s ListState) ScrollToRevealItem(ix int) {
	state := s.inner

	scrollTop := state.logicalScrollTop()
	height := Px(0.)
	if bounds, ok := state.lastLayoutBounds.Get(); ok {
		height = bounds.Size.Height
	}
	padding := state.lastPadding.UnwrapOrDefault()

	if ix <= scrollTop.ItemIx {
		scrollTop.ItemIx = ix
		scrollTop.OffsetInItem = Px(0.)
	} else {
		cursor := state.summaryCursor()
		cursor.Seek(listCountSeek(ix+1), sumtree.BiasRight)
		bottom := cursor.Start().height + padding.Top
		goalTop := max(Px(0.), bottom-height+padding.Bottom)

		cursor.Seek(listHeightSeek(goalTop), sumtree.BiasLeft)
		startIx := cursor.Start().count
		startItemTop := cursor.Start().height

		if startIx >= scrollTop.ItemIx {
			scrollTop.ItemIx = startIx
			scrollTop.OffsetInItem = goalTop - startItemTop
		}
	}

	state.setScrollAnchor(Some(scrollTop))
}

// BoundsForItem returns the bounds for the given item in window coordinates, if it's been
// rendered.
func (s ListState) BoundsForItem(ix int) (Bounds[Pixels], bool) {
	state := s.inner

	bounds := state.lastLayoutBounds.UnwrapOrDefault()
	padding := state.lastPadding.UnwrapOrDefault()
	logicalScrollTop := state.logicalScrollTop()
	if ix < logicalScrollTop.ItemIx {
		return Bounds[Pixels]{}, false
	}

	cursor := state.items.Cursor[listCountHeight](listUnit{})
	cursor.Seek(sumtree.DimensionsTarget[listItemSummary, listUnit, listCount, listHeight, sumtree.Unit[listItemSummary, listUnit]]{Target: listCount(logicalScrollTop.ItemIx)}, sumtree.BiasRight)

	scrollTop := Pixels(cursor.Start().V1) + logicalScrollTop.OffsetInItem

	cursor.SeekForward(sumtree.DimensionsTarget[listItemSummary, listUnit, listCount, listHeight, sumtree.Unit[listItemSummary, listUnit]]{Target: listCount(ix)}, sumtree.BiasRight)
	if item, ok := cursor.Item(); ok && item.measured {
		start := cursor.Start()
		if int(start.V0) == ix {
			top := bounds.Top() + Pixels(start.V1) - scrollTop
			return BoundsFromCorners(
				NewPoint(bounds.Left()+padding.Left, top),
				NewPoint(bounds.Right()-padding.Right, top+item.size.Height),
			), true
		}
	}
	return Bounds[Pixels]{}, false
}

// ScrollbarDragStarted is called when the user starts dragging the scrollbar.
//
// This will prevent the height reported to the scrollbar from changing during the drag as items
// in the overdraw get measured, and help offset scroll position changes accordingly.
func (s ListState) ScrollbarDragStarted() {
	state := s.inner
	state.scrollbarDragStartHeight = Some(state.items.Summary().height)
}

// ScrollbarDragEnded is called when the user stops dragging the scrollbar.
//
// See ScrollbarDragStarted.
func (s ListState) ScrollbarDragEnded() {
	s.inner.scrollbarDragStartHeight.Take()
}

// IsScrollbarDragging returns true if the scrollbar is currently being dragged.
//
// This is set between ScrollbarDragStarted and ScrollbarDragEnded calls. Useful for consumers
// that need to distinguish scrollbar drags from wheel/trackpad scrolls, e.g. to suppress
// auto-scroll behavior during manual positioning.
func (s ListState) IsScrollbarDragging() bool {
	return s.inner.scrollbarDragStartHeight.IsSome()
}

// SetOffsetFromScrollbar sets the offset from the scrollbar.
func (s ListState) SetOffsetFromScrollbar(point Point[Pixels]) {
	s.inner.setOffsetFromScrollbar(point)
}

// MaxOffsetForScrollbar returns the maximum scroll offset according to the items we have
// measured. This value remains constant while dragging to prevent the scrollbar from moving away
// unexpectedly.
func (s ListState) MaxOffsetForScrollbar() Point[Pixels] {
	return NewPoint(PixelsZero, s.inner.maxScrollOffset())
}

// ScrollPxOffsetForScrollbar returns the current scroll offset adjusted for the scrollbar.
//
// The returned offset has a negative y component representing how far the content has
// scrolled.
func (s ListState) ScrollPxOffsetForScrollbar() Point[Pixels] {
	state := s.inner

	if state.scrollAnchor.IsNone() && state.alignment == ListAlignmentBottom {
		return NewPoint(Px(0.), -state.maxScrollOffset())
	}

	logicalScrollTop := state.logicalScrollTop()

	cursor := state.summaryCursor()
	summary := cursor.Summary[listItemSummary](listCountSeek(logicalScrollTop.ItemIx), sumtree.BiasRight)
	offset := summary.height + logicalScrollTop.OffsetInItem

	return NewPoint(Px(0.), -offset)
}

// ViewportBounds returns the bounds of the viewport in pixels.
func (s ListState) ViewportBounds() Bounds[Pixels] {
	return s.inner.lastLayoutBounds.UnwrapOrDefault()
}

// ItemIsAboveViewport returns whether the item is entirely above the viewport; ok is false if
// the list has not measured enough layout to know.
//
// A zero-height viewport still yields a definitive answer: callers may size sibling UI based on
// this query (potentially squeezing the list itself to zero height), so returning no answer in
// that case would make the answer oscillate from frame to frame.
func (s ListState) ItemIsAboveViewport(ix int) (above bool, ok bool) {
	viewportBounds, ok := s.inner.lastLayoutBounds.Get()
	if !ok {
		return false, false
	}

	scrollTop := s.LogicalScrollTop()
	if ix < scrollTop.ItemIx {
		return true, true
	}

	itemBounds, ok := s.BoundsForItem(ix)
	if !ok {
		return false, false
	}
	return itemBounds.Bottom() <= viewportBounds.Top(), true
}

// ItemIsBelowViewport returns whether the item is entirely below the viewport; ok is false if
// the list has not measured enough layout to know.
//
// See ItemIsAboveViewport for why a zero-height viewport still yields a definitive answer.
func (s ListState) ItemIsBelowViewport(ix int) (below bool, ok bool) {
	viewportBounds, ok := s.inner.lastLayoutBounds.Get()
	if !ok {
		return false, false
	}

	scrollTop := s.LogicalScrollTop()
	if ix < scrollTop.ItemIx {
		return false, true
	}

	itemBounds, ok := s.BoundsForItem(ix)
	if !ok {
		return false, false
	}
	return itemBounds.Top() >= viewportBounds.Bottom(), true
}

func (s *listStateInner) maxScrollOffset() Pixels {
	bounds := s.lastLayoutBounds.UnwrapOrDefault()
	height, ok := s.scrollbarDragStartHeight.Get()
	if !ok {
		height = s.items.Summary().height
	}
	return max(height-bounds.Size.Height, Px(0.))
}

func listVisibleRange(items listItemTree, height Pixels, scrollTop ListOffset) Range {
	cursor := items.Cursor[listItemSummary](listUnit{})
	cursor.Seek(listCountSeek(scrollTop.ItemIx), sumtree.BiasRight)
	startY := cursor.Start().height + scrollTop.OffsetInItem
	cursor.SeekForward(listHeightSeek(startY+height), sumtree.BiasLeft)
	return Range{Start: scrollTop.ItemIx, End: cursor.Start().count + 1}
}

func (s *listStateInner) scroll(scrollTop ListOffset, height Pixels, delta Point[Pixels], currentView EntityID, window *Window, cx *App) {
	if s.reset || delta.Y == Px(0.) {
		return
	}

	padding := s.lastPadding.UnwrapOrDefault()
	scrollMax := max(s.items.Summary().height+padding.Top+padding.Bottom-height, Px(0.))
	newScrollTop := min(max(s.scrollTop(scrollTop)-delta.Y, Px(0.)), scrollMax)

	if s.alignment == ListAlignmentBottom && newScrollTop == scrollMax {
		s.setScrollAnchor(None[ListOffset]())
	} else {
		start, _, _, _ := s.items.Find[listItemSummary](listUnit{}, listHeightSeek(newScrollTop), sumtree.BiasRight)
		s.setScrollAnchor(Some(ListOffset{ItemIx: start.count, OffsetInItem: newScrollTop - start.height}))
	}

	if delta.Y > Px(0.) {
		s.followState.stopFollowing()
	}

	if s.scrollHandler != nil {
		visibleRange := listVisibleRange(s.items, height, scrollTop)
		s.scrollHandler(&ListScrollEvent{
			VisibleRange:    visibleRange,
			Count:           s.items.Summary().count,
			IsScrolled:      s.scrollAnchor.IsSome(),
			IsFollowingTail: s.followState.isFollowingTail(),
		}, window, cx)
	}

	cx.Notify(currentView)
}

func (s *listStateInner) logicalScrollTop() ListOffset {
	if anchor, ok := s.scrollAnchor.Get(); ok {
		return anchor.position
	}
	switch s.alignment {
	case ListAlignmentBottom:
		return ListOffset{ItemIx: s.items.Summary().count, OffsetInItem: Px(0.)}
	default:
		return ListOffset{ItemIx: 0, OffsetInItem: Px(0.)}
	}
}

func (s *listStateInner) scrollTop(logicalScrollTop ListOffset) Pixels {
	start, _, _, _ := s.items.Find[listItemSummary](listUnit{}, listCountSeek(logicalScrollTop.ItemIx), sumtree.BiasRight)
	return start.height + logicalScrollTop.OffsetInItem
}

func (s *listStateInner) isItemReady(index int, cx *App) bool {
	return s.itemMeasurementReady == nil || s.itemMeasurementReady(index, cx)
}

func (s *listStateInner) isProvisional() bool {
	anchor, ok := s.scrollAnchor.Get()
	return ok && anchor.provisional
}

func (s *listStateInner) setScrollAnchor(position Option[ListOffset]) {
	pos, ok := position.Get()
	if !ok {
		s.scrollAnchor = None[listScrollAnchor]()
		return
	}
	pending := None[listPendingScroll]()
	if anchor, ok := s.scrollAnchor.Get(); ok {
		if p, ok := anchor.pending.Get(); ok && pos.ItemIx < s.items.Summary().count {
			if !p.proportional || s.isProvisional() {
				pending = Some(listPendingScrollAbsolute(pos.OffsetInItem))
			} else {
				cursor := s.countCursor()
				cursor.Seek(listCount(pos.ItemIx), sumtree.BiasRight)
				if item, ok := cursor.Item(); ok {
					if size, ok := item.itemSizeHint(); ok && size.Height > Px(0.) {
						pending = Some(listPendingScrollProportional(clampF32(float32(pos.OffsetInItem/size.Height), 0., 1.)))
					}
				}
			}
		}
	}
	s.scrollAnchor = Some(listScrollAnchor{
		position:    pos,
		pending:     pending,
		provisional: false,
		pendingEnd:  false,
	})
}

func (s *listStateInner) layoutAllItems(availableWidth Pixels, renderItem listRenderItemFn, window *Window, cx *App) {
	if !s.measuringBehavior.measure {
		return
	}
	if s.measuringBehavior.hasMeasured {
		return
	}
	s.measuringBehavior.hasMeasured = true

	cursor := s.countCursor()
	availableItemSpace := NewSizeOf(AvailableSpaceDefinite(availableWidth), AvailableSpaceMinContent())

	var measuredItems []listItem

	ix := 0
	for item := range cursor.All() {
		size, ok := item.itemSize()
		if !ok {
			element := renderItem(ix, window, cx)
			size = element.LayoutAsRoot(availableItemSpace, window, cx)
		}

		measuredItems = append(measuredItems, item.withMeasurement(size, s.isItemReady(ix, cx)))
		ix++
	}

	s.items = newListItemTree(measuredItems)
	s.measuringBehavior = ListMeasuringBehaviorMeasure(s.items.Summary().unrenderedCount == 0)
}

func (s *listStateInner) layoutItems(availableWidth Option[Pixels], availableHeight Pixels, padding Edges[Pixels], renderItem listRenderItemFn, window *Window, cx *App) listLayoutItemsResponse {
	oldItems := s.items
	var measuredItems listDeque[listItem]
	var itemLayouts listDeque[listItemLayout]
	renderedHeight := padding.Top
	maxItemWidth := Px(0.)
	scrollTop := s.logicalScrollTop()

	if s.followState.isFollowingTail() || (s.scrollAnchor.IsSome() && s.scrollAnchor.Unwrap().pendingEnd) {
		scrollTop = ListOffset{ItemIx: s.items.Summary().count, OffsetInItem: Px(0.)}
		s.setScrollAnchor(Some(scrollTop))
	}

	preserveAnchor := scrollTop.ItemIx < s.items.Summary().count
	if !preserveAnchor {
		s.followState.startFollowing()
	}
	provisional := false
	renderedFocusedItem := false

	widthSpace := AvailableSpaceMaxContent()
	if width, ok := availableWidth.Get(); ok {
		widthSpace = AvailableSpaceDefinite(width)
	}
	availableItemSpace := NewSizeOf(widthSpace, AvailableSpaceMinContent())

	cursor := oldItems.Cursor[listCount](listUnit{})

	cursor.Seek(listCount(scrollTop.ItemIx), sumtree.BiasRight)
	ix := 0
	for item := range cursor.All() {
		visibleHeight := renderedHeight - scrollTop.OffsetInItem
		if visibleHeight >= availableHeight+s.overdraw {
			break
		}

		itemIndex := scrollTop.ItemIx + ix
		var size Size[Pixels]
		var ready bool
		if measuredSize, ok := item.itemSize(); visibleHeight >= availableHeight && ok {
			size, ready = measuredSize, s.isItemReady(itemIndex, cx)
		} else {
			element := renderItem(itemIndex, window, cx)
			elementSize := element.LayoutAsRoot(availableItemSpace, window, cx)
			ready = s.isItemReady(itemIndex, cx)
			if ix == 0 && ready {
				if anchor := s.scrollAnchor.AsMut(); anchor != nil {
					if pending, ok := anchor.pending.Take().Get(); ok {
						if pending.proportional {
							scrollTop.OffsetInItem = elementSize.Height * Pixels(pending.fraction)
						} else {
							scrollTop.OffsetInItem = min(pending.absolute, elementSize.Height)
						}
						anchor.position = scrollTop
					}
				}
			}

			if visibleHeight < availableHeight {
				itemLayouts.pushBack(listItemLayout{index: itemIndex, element: element, size: elementSize})
				if item.containsFocused(window, cx) {
					renderedFocusedItem = true
				}
			}
			size = elementSize
		}

		renderedHeight += size.Height
		maxItemWidth = max(maxItemWidth, size.Width)
		provisional = provisional || !ready
		measuredItems.pushBack(item.withMeasurement(size, ready))
		if !ready && visibleHeight >= availableHeight {
			break
		}
		ix++
	}
	renderedHeight += padding.Bottom

	cursor.Seek(listCount(scrollTop.ItemIx), sumtree.BiasRight)

	if !provisional && renderedHeight-scrollTop.OffsetInItem < availableHeight {
		forwardFocused := renderedFocusedItem
		for renderedHeight < availableHeight {
			cursor.Prev()
			item, ok := cursor.Item()
			if !ok {
				break
			}
			itemIndex := int(cursor.Start())
			element := renderItem(itemIndex, window, cx)
			elementSize := element.LayoutAsRoot(availableItemSpace, window, cx)
			ready := s.isItemReady(itemIndex, cx)
			provisional = provisional || !ready
			renderedHeight += elementSize.Height
			measuredItems.pushFront(item.withMeasurement(elementSize, ready))
			itemLayouts.pushFront(listItemLayout{index: itemIndex, element: element, size: elementSize})
			if item.containsFocused(window, cx) {
				renderedFocusedItem = true
			}
			if preserveAnchor && provisional {
				break
			}
		}

		if preserveAnchor && provisional {
			itemLayouts.dropFront()
			renderedFocusedItem = forwardFocused
		} else {
			scrollTop = ListOffset{ItemIx: int(cursor.Start()), OffsetInItem: renderedHeight - availableHeight}

			switch s.alignment {
			case ListAlignmentTop:
				scrollTop.OffsetInItem = max(scrollTop.OffsetInItem, Px(0.))
				s.setScrollAnchor(Some(scrollTop))
			case ListAlignmentBottom:
				scrollTop = ListOffset{ItemIx: int(cursor.Start()), OffsetInItem: renderedHeight - availableHeight}
				s.setScrollAnchor(None[ListOffset]())
			}
		}
	}

	if anchor := s.scrollAnchor.AsMut(); anchor != nil {
		anchor.provisional = preserveAnchor && provisional
		anchor.pendingEnd = !preserveAnchor && provisional
	}

	if !provisional {
		leadingOverdraw := scrollTop.OffsetInItem
		for leadingOverdraw < s.overdraw {
			cursor.Prev()
			item, ok := cursor.Item()
			if !ok {
				break
			}
			size, measured := item.itemSize()
			if !measured {
				element := renderItem(int(cursor.Start()), window, cx)
				size = element.LayoutAsRoot(availableItemSpace, window, cx)
			}

			leadingOverdraw += size.Height
			ready := s.isItemReady(int(cursor.Start()), cx)
			measuredItems.pushFront(item.withMeasurement(size, ready))
			if !ready {
				break
			}
		}
	}

	measuredRange := Range{Start: int(cursor.Start()), End: int(cursor.Start()) + measuredItems.len()}
	rebuild := oldItems.Cursor[listCount](listUnit{})
	newItems := rebuild.Slice(listCount(measuredRange.Start), sumtree.BiasRight)
	newItems.Extend(measuredItems.items(), listUnit{})
	rebuild.Seek(listCount(measuredRange.End), sumtree.BiasRight)
	newItems.Append(rebuild.Suffix(), listUnit{})
	s.items = newItems

	if !provisional && s.followState.hasStoppedFollowing() {
		padding := s.lastPadding.UnwrapOrDefault()
		totalHeight := s.items.Summary().height + padding.Top + padding.Bottom
		scrollOffset := s.scrollTop(scrollTop)
		if scrollOffset+availableHeight >= totalHeight-Px(1.0) {
			s.followState.startFollowing()
		}
	}

	layouts := itemLayouts.items()
	if !renderedFocusedItem {
		focusCursor := s.items.Filter[listCount](listUnit{}, func(summary listItemSummary) bool { return summary.hasFocusHandles })
		focusCursor.Next()
		for {
			item, ok := focusCursor.Item()
			if !ok {
				break
			}
			if item.containsFocused(window, cx) {
				itemIndex := int(focusCursor.Start())
				element := renderItem(itemIndex, window, cx)
				size := element.LayoutAsRoot(availableItemSpace, window, cx)
				layouts = append(layouts, listItemLayout{index: itemIndex, element: element, size: size})
				break
			}
			focusCursor.Next()
		}
	}

	return listLayoutItemsResponse{
		maxItemWidth: maxItemWidth,
		scrollTop:    scrollTop,
		itemLayouts:  layouts,
	}
}

type listAutoscrollRequest struct {
	offset ListOffset
}

func (r listAutoscrollRequest) Error() string {
	return fmt.Sprintf("list autoscroll request to %+v", r.offset)
}

func (s *listStateInner) prepaintItems(bounds Bounds[Pixels], padding Edges[Pixels], autoscroll bool, renderItem listRenderItemFn, window *Window, cx *App) (listLayoutItemsResponse, error) {
	itemWidth := bounds.Size.Width - padding.Along(AxisHorizontal)
	return window.Transact(func(window *Window) (listLayoutItemsResponse, error) {
		if s.measuringBehavior.measure && !s.measuringBehavior.hasMeasured {
			s.layoutAllItems(itemWidth, renderItem, window, cx)
		}

		layoutResponse := s.layoutItems(Some(itemWidth), bounds.Size.Height, padding, renderItem, window, cx)

		window.TakeAutoscroll()

		if bounds.Size.Height > padding.Top+padding.Bottom {
			itemOrigin := bounds.Origin.Add(NewPoint(padding.Left, padding.Top))
			itemOrigin.Y -= layoutResponse.scrollTop.OffsetInItem
			for i := range layoutResponse.itemLayouts {
				item := &layoutResponse.itemLayouts[i]
				window.WithContentMask(Some(ContentMask[Pixels]{Bounds: bounds}), func(window *Window) struct{} {
					item.element.PrepaintAt(itemOrigin, window, cx)
					return struct{}{}
				})

				if autoscrollBounds, ok := window.TakeAutoscroll(); ok && autoscroll {
					if autoscrollBounds.Top() < bounds.Top() {
						itemIx := item.index
						offsetInItem := autoscrollBounds.Top() - itemOrigin.Y

						if offsetInItem < PixelsZero {
							cursor := s.countCursor()
							cursor.Seek(listCount(itemIx), sumtree.BiasRight)
							for offsetInItem < PixelsZero {
								cursor.Prev()
								prevItem, ok := cursor.Item()
								if !ok {
									offsetInItem = PixelsZero
									break
								}
								size, measured := prevItem.itemSize()
								if !measured {
									element := renderItem(int(cursor.Start()), window, cx)
									size = element.LayoutAsRoot(NewSizeOf(AvailableSpaceDefinite(itemWidth), AvailableSpaceMinContent()), window, cx)
								}
								itemIx = int(cursor.Start())
								offsetInItem += size.Height
							}
						}

						return listLayoutItemsResponse{}, listAutoscrollRequest{offset: ListOffset{ItemIx: itemIx, OffsetInItem: offsetInItem}}
					} else if autoscrollBounds.Bottom() > bounds.Bottom() {
						cursor := s.countCursor()
						cursor.Seek(listCount(item.index), sumtree.BiasRight)
						height := bounds.Size.Height - padding.Top - padding.Bottom

						height -= autoscrollBounds.Bottom() - itemOrigin.Y

						for height > PixelsZero {
							cursor.Prev()
							prevItem, ok := cursor.Item()
							if !ok {
								break
							}

							size, measured := prevItem.itemSize()
							if !measured {
								element := renderItem(int(cursor.Start()), window, cx)
								size = element.LayoutAsRoot(NewSizeOf(AvailableSpaceDefinite(itemWidth), AvailableSpaceMinContent()), window, cx)
							}
							height -= size.Height
						}

						offsetInItem := PixelsZero
						if height < PixelsZero {
							offsetInItem = -height
						}
						return listLayoutItemsResponse{}, listAutoscrollRequest{offset: ListOffset{ItemIx: int(cursor.Start()), OffsetInItem: offsetInItem}}
					}
				}

				itemOrigin.Y += item.size.Height
			}
		} else {
			layoutResponse.itemLayouts = nil
		}

		return layoutResponse, nil
	})
}

func (s *listStateInner) setOffsetFromScrollbar(point Point[Pixels]) {
	bounds, ok := s.lastLayoutBounds.Get()
	if !ok {
		return
	}
	height := bounds.Size.Height

	padding := s.lastPadding.UnwrapOrDefault()
	contentHeight, ok := s.scrollbarDragStartHeight.Get()
	if !ok {
		contentHeight = s.items.Summary().height
	}
	scrollMax := max(contentHeight+padding.Top+padding.Bottom-height, Px(0.))
	newScrollTop := min(max(-point.Y, Px(0.)), scrollMax)

	draggedToEnd := scrollMax > Px(0.) && newScrollTop >= max(scrollMax-Px(1.0), Px(0.))
	if draggedToEnd && s.followState.tail {
		s.followState = listFollowState{tail: true, isFollowing: true}
		itemCount := s.items.Summary().count
		s.setScrollAnchor(Some(ListOffset{ItemIx: itemCount, OffsetInItem: Px(0.)}))
		return
	}

	s.followState.stopFollowing()

	if s.alignment == ListAlignmentBottom && newScrollTop == scrollMax {
		s.setScrollAnchor(None[ListOffset]())
	} else {
		start, _, _, _ := s.items.Find[listItemSummary](listUnit{}, listHeightSeek(newScrollTop), sumtree.BiasRight)
		s.setScrollAnchor(Some(ListOffset{ItemIx: start.count, OffsetInItem: newScrollTop - start.height}))
	}
}

// ListOffset is an offset into the list's items, in terms of the item index and the number of
// pixels off the top left of the item.
type ListOffset struct {
	// ItemIx is the index of an item in the list.
	ItemIx int
	// OffsetInItem is the number of pixels to offset from the item index.
	OffsetInItem Pixels
}

// ElementID reports that List has no id.
func (l *List) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (l *List) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests the list's layout; with ListSizingBehaviorInfer it lays out items to
// measure its content.
func (l *List) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	var layoutID LayoutID
	switch l.sizingBehavior {
	case ListSizingBehaviorInfer:
		style := DefaultStyle()
		style.Overflow.Y = OverflowScroll
		style.Refine(&l.style)
		layoutID = window.WithTextStyle(optionalTextStyle(&style), func(window *Window) LayoutID {
			state := l.state.inner

			availableHeight := state.overdraw
			if lastBounds, ok := state.lastLayoutBounds.Get(); ok {
				availableHeight = lastBounds.Size.Height
			}
			padding := listContentInsets(&style, state.lastLayoutBounds.UnwrapOrDefault().Size, window.RemSize())

			layoutResponse := state.layoutItems(None[Pixels](), availableHeight, padding, l.renderItem, window, cx)
			maxElementWidth := layoutResponse.maxItemWidth

			summary := state.items.Summary()
			totalHeight := summary.height

			return window.RequestMeasuredLayout(style, func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], _ *Window, _ *App) Size[Pixels] {
				width, ok := knownDimensions.Width.Get()
				if !ok {
					if x, definite := availableSpace.Width.AsDefinite(); definite {
						width = x
					} else {
						width = maxElementWidth
					}
				}
				height := totalHeight
				if h, definite := availableSpace.Height.AsDefinite(); definite {
					height = min(totalHeight, h)
				}
				return NewSize(width, height)
			})
		})
	default:
		style := DefaultStyle()
		style.Refine(&l.style)
		layoutID = window.WithTextStyle(optionalTextStyle(&style), func(window *Window) LayoutID {
			return window.RequestLayout(style, nil, cx)
		})
	}
	return layoutID, struct{}{}
}

// Prepaint lays out and prepaints the visible items.
func (l *List) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) ListPrepaintState {
	state := l.state.inner
	state.reset = false

	style := DefaultStyle()
	style.Refine(&l.style)

	hitbox := window.InsertHitbox(bounds, HitboxBehaviorNormal)

	padding := listContentInsets(&style, bounds.Size, window.RemSize())

	lastBounds, hasBounds := state.lastLayoutBounds.Get()
	lastPadding, hasPadding := state.lastPadding.Get()
	if !hasBounds || !hasPadding || lastBounds.Size.Width-lastPadding.Along(AxisHorizontal) != bounds.Size.Width-padding.Along(AxisHorizontal) {
		var newItems []listItem
		for item := range state.items.Iter() {
			newItems = append(newItems, listItemUnmeasured(None[Size[Pixels]](), item.focusHandle))
		}
		state.items = newListItemTree(newItems)
		state.measuringBehavior.reset()
	}

	autoscroll := true
	var layout listLayoutItemsResponse
	for {
		result, err := state.prepaintItems(bounds, padding, autoscroll, l.renderItem, window, cx)
		if err == nil {
			layout = result
			break
		}
		state.setScrollAnchor(Some(err.(listAutoscrollRequest).offset))
		autoscroll = false
	}

	state.lastLayoutBounds = Some(bounds)
	state.lastPadding = Some(padding)
	return ListPrepaintState{hitbox: hitbox, layout: layout}
}

// Paint registers the list's scroll listener and paints the visible items.
func (l *List) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, prepaint *ListPrepaintState, window *Window, cx *App) {
	currentView := window.CurrentView()

	listState := l.state
	height := bounds.Size.Height
	scrollTop := prepaint.layout.scrollTop
	hitboxID := prepaint.hitbox.ID
	var accumulatedScrollDelta ScrollDelta
	window.OnMouseEvent(func(event *ScrollEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitboxID.ShouldHandleScroll(window) {
			accumulatedScrollDelta = accumulatedScrollDelta.Coalesce(event.Delta)
			pixelDelta := accumulatedScrollDelta.PixelDelta(Px(20.))
			listState.inner.scroll(scrollTop, height, pixelDelta, currentView, window, cx)
		}
	})

	window.WithContentMask(Some(ContentMask[Pixels]{Bounds: bounds}), func(window *Window) struct{} {
		for i := range prepaint.layout.itemLayouts {
			prepaint.layout.itemLayouts[i].element.Paint(window, cx)
		}
		return struct{}{}
	})
}

// IntoAnyElement converts the list into an AnyElement.
func (l *List) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, ListPrepaintState](l)
}

func listContentInsets(style *Style, size Size[Pixels], remSize Pixels) Edges[Pixels] {
	padding := EdgesDefiniteLengthToPixels(style.Padding, SizeAbsoluteLengthFromPixels(size), remSize)
	border := EdgesAbsoluteLengthToPixels(style.BorderWidths, remSize)

	return padding.Add(border)
}

// Style returns the list's style refinement.
func (l *List) Style() *StyleRefinement {
	return &l.style
}
