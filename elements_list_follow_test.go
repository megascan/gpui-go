package gpui

import (
	"testing"
)

func listFollowTestNewView(cx *VisualTestContext, state ListState, renderItem func(ix int, window *Window, cx *App) AnyElement) Entity[listTestClosureView] {
	return listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
		return listTestNewView(cx, state, renderItem)
	})
}

func listFollowTestFixedItem(int, *Window, *App) AnyElement {
	return NewDiv().H(Px(50.)).WFull().IntoAnyElement()
}

func listFollowTestDraw(cx *VisualTestContext, height float32, view Entity[listTestClosureView]) {
	listTestDrawView(cx, NewSize(Px(100.), Px(height)), func(*App) Entity[listTestClosureView] { return view })
}

func listFollowTestExpectOffset(t *testing.T, state ListState, itemIx int, offset float32) {
	t.Helper()
	if got := state.LogicalScrollTop(); got.ItemIx != itemIx || got.OffsetInItem != Px(offset) {
		t.Fatalf("offset %+v; want item %d offset %v", got, itemIx, offset)
	}
}

func listFollowTestExpectFollowing(t *testing.T, state ListState, following bool, msg string) {
	t.Helper()
	if state.IsFollowingTail() != following {
		t.Fatalf("is_following_tail = %v: %s", !following, msg)
	}
}

func listFollowTestBoundsRecorder(itemBounds *[]Bounds[Pixels]) func(int, *Window, *App) AnyElement {
	return func(int, *Window, *App) AnyElement {
		return NewCanvas(
			func(bounds Bounds[Pixels], _ *Window, _ *App) struct{} {
				*itemBounds = append(*itemBounds, bounds)
				return struct{}{}
			},
			func(Bounds[Pixels], struct{}, *Window, *App) {},
		).WFull().AspectRatio(2.).IntoAnyElement()
	}
}

func TestList_FollowTailStaysAtBottomAsItemsGrow(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		itemHeight := 50
		state := NewListState(10, ListAlignmentTop, Px(0.))
		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					height := itemHeight
					return NewList(state, func(int, *Window, *App) AnyElement {
						return NewDiv().H(Px(float32(height))).WFull().IntoAnyElement()
					}).WFull().HFull()
				}}
			})
		})

		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectOffset(t, state, 6, 0.)
		listFollowTestExpectFollowing(t, state, true, "")

		itemHeight = 80
		state.Remeasure()

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectOffset(t, state, 7, 40.)
		listFollowTestExpectFollowing(t, state, true, "")
	})
}

func TestList_PauseFollowingTailReengagesWhenStillAtBottom(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.))
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)
		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		state.PauseFollowingTail()
		listFollowTestExpectFollowing(t, state, false, "")

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectFollowing(t, state, true, "pausing while at the bottom must re-engage follow-tail on the next layout")
	})
}

func TestList_PauseFollowingTailFreezesOffBottom(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		itemHeight := 50
		state := NewListState(10, ListAlignmentTop, Px(0.))
		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					height := itemHeight
					return NewList(state, func(int, *Window, *App) AnyElement {
						return NewDiv().H(Px(float32(height))).WFull().IntoAnyElement()
					}).WFull().HFull()
				}}
			})
		})
		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		if got := state.LogicalScrollTop().ItemIx; got != 6 {
			t.Fatalf("item_ix = %d", got)
		}
		listFollowTestExpectFollowing(t, state, true, "")

		state.PauseFollowingTail()
		itemHeight = 80
		state.Remeasure()
		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectOffset(t, state, 6, 0.)
		listFollowTestExpectFollowing(t, state, false, "a paused list must not re-engage while the frozen top is off the bottom")

		itemHeight = 50
		state.Remeasure()
		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectFollowing(t, state, true, "returning to the bottom must restore follow-tail")
	})
}

func TestList_FollowTailDisengagesOnUserScroll(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.))
		state.SetFollowMode(FollowModeTail)

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, listFollowTestFixedItem)
		})
		listFollowTestExpectFollowing(t, state, true, "")

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(100.))))

		listFollowTestExpectFollowing(t, state, false, "follow-tail should disengage when the user scrolls toward the start")
	})
}

func TestList_FollowTailDisengagesOnScrollbarReposition(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)
		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-150.)))
		listFollowTestExpectOffset(t, state, 3, 0.)
		listFollowTestExpectFollowing(t, state, false, "follow-tail should disengage when the scrollbar manually repositions the list")

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectOffset(t, state, 3, 0.)
	})
}

func TestList_ScrollbarDragWithGrowingContent(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		lastItemHeight := 50
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					return NewList(state, func(index int, _ *Window, _ *App) AnyElement {
						height := 50
						if index == 9 {
							height = lastItemHeight
						}
						return NewDiv().H(Px(float32(height))).WFull().IntoAnyElement()
					}).WFull().HFull()
				}}
			})
		})

		listFollowTestDraw(cx, 200., view.Clone())

		state.ScrollbarDragStarted()

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-150.)))
		before := state.ScrollPxOffsetForScrollbar()
		listFollowTestExpectOffset(t, state, 3, 0.)

		lastItemHeight = 550
		state.RemeasureItems(Range{Start: 9, End: 10})
		listFollowTestDraw(cx, 200., view)

		if got := state.MaxOffsetForScrollbar().Y; got != Px(300.) {
			t.Fatalf("max offset = %v", got)
		}
		if got := state.ScrollPxOffsetForScrollbar(); got != before {
			t.Fatalf("scrollbar offset = %v; want %v", got, before)
		}

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-150.)))
		listFollowTestExpectOffset(t, state, 3, 0.)
	})
}

func TestList_SetFollowTailSnapsToBottom(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.))
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)

		state.ScrollTo(ListOffset{ItemIx: 3, OffsetInItem: Px(0.)})

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectOffset(t, state, 3, 0.)
		listFollowTestExpectFollowing(t, state, false, "")

		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectOffset(t, state, 6, 0.)
		listFollowTestExpectFollowing(t, state, true, "")
	})
}

func TestList_BottomAlignedScrollbarOffsetAtEnd(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		const items = 10
		const itemSize float32 = 50.0
		state := NewListState(items, ListAlignmentBottom, Px(items*itemSize))

		listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().H(Px(itemSize)).WFull().IntoAnyElement()
			})
		})

		if got := state.LogicalScrollTop().ItemIx; got != items {
			t.Fatalf("item_ix = %d", got)
		}

		maxOffset := state.MaxOffsetForScrollbar()
		scrollOffset := state.ScrollPxOffsetForScrollbar()
		if -scrollOffset.Y != maxOffset.Y {
			t.Fatalf("scrollbar offset (%v) should equal max offset (%v) when list is pinned to bottom", -scrollOffset.Y, maxOffset.Y)
		}
	})
}

func TestList_FollowTailReengagesWhenScrolledBackToBottom(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.))
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)
		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(50.))))
		listFollowTestExpectFollowing(t, state, false, "")

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(-10000.))))

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectFollowing(t, state, true, "follow_tail should re-engage after scrolling back to the bottom")
	})
}

func TestList_FollowTailReengagementNotFooledByUnmeasuredItems(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(20, ListAlignmentTop, Px(1000.))
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)
		state.SetFollowMode(FollowModeTail)

		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(200.))))
		listFollowTestExpectFollowing(t, state, false, "")

		state.RemeasureItems(Range{Start: 19, End: 20})

		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectFollowing(t, state, false, "follow_tail should not falsely re-engage due to an unmeasured item reducing items.summary().height")
	})
}

func TestList_FollowTailReengagesAfterScrollbarDisengagement(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)

		state.SetFollowMode(FollowModeTail)
		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-150.)))
		listFollowTestExpectFollowing(t, state, false, "")

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-300.)))
		listFollowTestDraw(cx, 200., view)
		listFollowTestExpectFollowing(t, state, true, "follow_tail should re-engage after scrolling back to the bottom via the scrollbar")
	})
}

func TestList_FollowTailReengagesAfterScrollbarDragToBottomWhileGrowing(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		view := listFollowTestNewView(cx, state, listFollowTestFixedItem)

		state.SetFollowMode(FollowModeTail)
		listFollowTestDraw(cx, 200., view.Clone())
		listFollowTestExpectFollowing(t, state, true, "")

		state.ScrollbarDragStarted()

		state.Splice(Range{Start: 10, End: 10}, 10)
		listFollowTestDraw(cx, 200., view.Clone())

		state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-300.)))
		state.ScrollbarDragEnded()

		listFollowTestDraw(cx, 200., view)

		listFollowTestExpectFollowing(t, state, true, "follow_tail should re-engage when the user drags the scrollbar to the bottom of its track, even when content has grown during the drag (so frozen_bottom < live_bottom)")
	})
}

func TestList_HorizontalPaddingInsetsItems(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		var itemBounds []Bounds[Pixels]

		listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					return NewList(state, listFollowTestBoundsRecorder(&itemBounds)).
						Pl(Px(10.)).
						Pr(Px(30.)).
						WFull().
						HFull()
				}}
			})
		})

		for _, bounds := range itemBounds {
			if bounds.Origin.X != Px(10.) || bounds.Size.Width != Px(60.) {
				t.Fatalf("item bounds %v", bounds)
			}
		}
		if got := state.MaxOffsetForScrollbar().Y; got != Px(200.) {
			t.Fatalf("max offset = %v", got)
		}
	})
}

func TestList_InferredWidthIncludesHorizontalPadding(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(2, ListAlignmentTop, Px(10.))
		fullWidthItemBounds := None[Bounds[Pixels]]()

		listTestDrawView(cx, NewSize(Px(200.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					return NewDiv().Flex().SizeFull().Child(
						NewList(state, func(ix int, _ *Window, _ *App) AnyElement {
							if ix == 0 {
								return NewDiv().W(Px(50.)).H(Px(20.)).IntoAnyElement()
							}
							return NewCanvas(
								func(bounds Bounds[Pixels], _ *Window, _ *App) struct{} {
									fullWidthItemBounds = Some(bounds)
									return struct{}{}
								},
								func(Bounds[Pixels], struct{}, *Window, *App) {},
							).H(Px(20.)).WFull().IntoAnyElement()
						}).
							WithSizingBehavior(ListSizingBehaviorInfer).
							Pl(Px(10.)).
							Pr(Px(30.)).
							HFull(),
					)
				}}
			})
		})

		bounds := fullWidthItemBounds.Expect("item 1 was not painted")
		if bounds.Origin.X != Px(10.) || bounds.Size.Width != Px(50.) {
			t.Fatalf("bounds %v", bounds)
		}
	})
}

func TestList_AutoscrollAboveItemTopWithHorizontalPadding(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(0.))
		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})

		listTestDrawView(cx, NewSize(Px(100.), Px(60.)), func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					return NewList(state, func(ix int, _ *Window, _ *App) AnyElement {
						if ix == 2 {
							return NewCanvas(
								func(bounds Bounds[Pixels], window *Window, _ *App) struct{} {
									window.RequestAutoscroll(BoundsFromCorners(
										NewPoint(bounds.Left(), bounds.Top()-Px(30.)),
										NewPoint(bounds.Right(), bounds.Top()+Px(5.)),
									))
									return struct{}{}
								},
								func(Bounds[Pixels], struct{}, *Window, *App) {},
							).WFull().AspectRatio(2.).IntoAnyElement()
						}
						return NewDiv().WFull().AspectRatio(2.).IntoAnyElement()
					}).
						Pl(Px(10.)).
						Pr(Px(30.)).
						WFull().
						HFull()
				}}
			})
		})

		listFollowTestExpectOffset(t, state, 1, 0.)
	})
}

func TestList_HorizontalPaddingChangeRemeasuresItems(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()

		for _, step := range []struct{ paddingRight, expectedMaxOffset Pixels }{{Px(30.), Px(200.)}, {Px(10.), Px(300.)}} {
			paddingRight := step.paddingRight
			listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
				return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
					return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
						return NewList(state, func(int, *Window, *App) AnyElement {
							return NewDiv().WFull().AspectRatio(2.).IntoAnyElement()
						}).
							Pl(Px(10.)).
							Pr(paddingRight).
							WFull().
							HFull()
					}}
				})
			})
			if got := state.MaxOffsetForScrollbar().Y; got != step.expectedMaxOffset {
				t.Fatalf("max offset = %v; want %v", got, step.expectedMaxOffset)
			}
		}
	})
}

func TestList_BorderAndPaddingInsetItems(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()
		var itemBounds []Bounds[Pixels]

		listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					return NewList(state, listFollowTestBoundsRecorder(&itemBounds)).
						BorderL4().
						Pl(Px(6.)).
						BorderR8().
						Pr(Px(22.)).
						WFull().
						HFull()
				}}
			})
		})

		if len(itemBounds) == 0 {
			t.Fatal("no items painted")
		}
		for _, bounds := range itemBounds {
			if bounds.Origin.X != Px(10.) || bounds.Size.Width != Px(60.) {
				t.Fatalf("item bounds %v", bounds)
			}
		}
		if got := state.MaxOffsetForScrollbar().Y; got != Px(200.) {
			t.Fatalf("max offset = %v", got)
		}

		reported, ok := state.BoundsForItem(0)
		if !ok {
			panic("item 0 should be measured")
		}
		if reported.Origin.X != Px(10.) || reported.Size.Width != Px(60.) {
			t.Fatalf("reported bounds %v", reported)
		}
	})
}
