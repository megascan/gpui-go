package gpui

import (
	"slices"
	"testing"
)

type listTestPainted struct {
	index  int
	top    Pixels
	bottom Pixels
}

type listTestClosureView struct {
	render func(window *Window, cx *Context[listTestClosureView]) IntoElement
}

func (v *listTestClosureView) Render(window *Window, cx *Context[listTestClosureView]) IntoElement {
	return v.render(window, cx)
}

type listTestView struct {
	state      ListState
	renderItem func(index int) AnyElement
}

func (v *listTestView) Render(_ *Window, _ *Context[listTestView]) IntoElement {
	renderItem := v.renderItem
	return NewList(v.state, func(index int, _ *Window, _ *App) AnyElement { return renderItem(index) }).
		WFull().
		HFull()
}

type listTestListView struct {
	state ListState
}

func (v *listTestListView) Render(_ *Window, _ *Context[listTestListView]) IntoElement {
	return NewList(v.state, func(int, *Window, *App) AnyElement {
		return NewDiv().H(Px(20.)).WFull().IntoAnyElement()
	}).
		WFull().
		HFull()
}

func listTestDrawView[V any](cx *VisualTestContext, size Size[Pixels], build func(cx *App) Entity[V]) {
	cx.Draw(NewPoint(Px(0.), Px(0.)), SizeAvailableSpaceFromPixels(size), func(_ *Window, cx *App) Element[Option[AnyElement], Option[AnyElement]] {
		return AnyViewFromEntity(build(cx)).intoElement()
	})
}

func listTestNewView(cx *App, state ListState, renderItem func(ix int, window *Window, cx *App) AnyElement) Entity[listTestClosureView] {
	return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
		return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
			return NewList(state, renderItem).WFull().HFull()
		}}
	})
}

func listTestDrawTestListView(cx *VisualTestContext, size Size[Pixels], state ListState) {
	listTestDrawView(cx, size, func(cx *App) Entity[listTestListView] {
		return cx.New(func(*Context[listTestListView]) *listTestListView { return &listTestListView{state: state} })
	})
}

func listTestDrawList(state ListState, viewportHeight float32, renderItem func(index int) AnyElement, cx *VisualTestContext) {
	listTestDrawView(cx, NewSize(Px(100.), Px(viewportHeight)), func(cx *App) Entity[listTestView] {
		return cx.New(func(*Context[listTestView]) *listTestView {
			return &listTestView{state: state, renderItem: renderItem}
		})
	})
}

func listTestPaintedItem(index int, height float32, painted *[]listTestPainted) AnyElement {
	return NewCanvas(
		func(Bounds[Pixels], *Window, *App) struct{} { return struct{}{} },
		func(bounds Bounds[Pixels], _ struct{}, _ *Window, _ *App) {
			*painted = append(*painted, listTestPainted{index, bounds.Top(), bounds.Bottom()})
		},
	).
		H(Px(height)).
		WFull().
		IntoAnyElement()
}

func listTestItemReadiness(state ListState, ready bool) *bool {
	r := new(bool)
	*r = ready
	state.SetItemMeasurementReadiness(func(int, *App) bool { return *r })
	return r
}

func listTestOffset(itemIx int, offset float32) ListOffset {
	return ListOffset{ItemIx: itemIx, OffsetInItem: Px(offset)}
}

func listTestOpt(v bool, ok bool) Option[bool] {
	if !ok {
		return None[bool]()
	}
	return Some(v)
}

func listTestFirst(painted []listTestPainted) Option[listTestPainted] {
	if len(painted) == 0 {
		return None[listTestPainted]()
	}
	return Some(painted[0])
}

func listTestLast(painted []listTestPainted) Option[listTestPainted] {
	if len(painted) == 0 {
		return None[listTestPainted]()
	}
	return Some(painted[len(painted)-1])
}

func listTestFind(painted []listTestPainted, index int) Option[listTestPainted] {
	for _, p := range painted {
		if p.index == index {
			return Some(p)
		}
	}
	return None[listTestPainted]()
}

func listTestRun(t *testing.T, fn func(t *testing.T, cx *VisualTestContext)) {
	t.Helper()
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		cx := tcx.AddEmptyWindow()
		fn(t, cx)
	})
}

func listTestScrollEvent(position Point[Pixels], delta Point[Pixels]) ScrollEvent {
	return ScrollEvent{Position: position, Delta: ScrollDeltaPixels(delta)}
}

func TestList_AutoscrollAboveItemTopRendersItemsAbove(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.))
		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})

		listTestDrawView(cx, NewSize(Px(100.), Px(60.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(ix int, _ *Window, _ *App) AnyElement {
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
					).
						H(Px(20.)).
						WFull().
						IntoAnyElement()
				}
				return NewDiv().H(Px(20.)).WFull().IntoAnyElement()
			})
		})

		scrollTop := state.LogicalScrollTop()
		if scrollTop.OffsetInItem < Px(0.) {
			t.Fatalf("offset_in_item must never be negative (would leave blank space above), got %v", scrollTop.OffsetInItem)
		}
		if scrollTop.ItemIx != 0 || scrollTop.OffsetInItem != Px(10.) {
			t.Fatalf("scroll top %+v; want item 0 offset 10", scrollTop)
		}
	})
}

func TestList_ResetAfterPaintBeforeScroll(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.))

		state.ScrollTo(ListOffset{ItemIx: 0, OffsetInItem: Px(0.0)})

		listTestDrawView(cx, NewSize(Px(100.), Px(20.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().H(Px(10.)).WFull().IntoAnyElement()
			})
		})

		state.Reset(5)

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(1.), Px(1.)), NewPoint(Px(0.), Px(-500.))))

		if got := state.LogicalScrollTop(); got.ItemIx != 0 || got.OffsetInItem != Px(0.) {
			t.Fatalf("scroll top %+v; want the top of the list", got)
		}
	})
}

func TestList_ScrollByPositiveAndNegativeDistance(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.))

		listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().H(Px(20.)).WFull().IntoAnyElement()
			})
		})

		state.ScrollBy(Px(30.))
		if offset := state.LogicalScrollTop(); offset.ItemIx != 1 || offset.OffsetInItem != Px(10.) {
			t.Fatalf("after +30: %+v", offset)
		}

		state.ScrollBy(Px(-30.))
		if offset := state.LogicalScrollTop(); offset.ItemIx != 0 || offset.OffsetInItem != Px(0.) {
			t.Fatalf("after -30: %+v", offset)
		}

		state.ScrollBy(Px(0.))
		if offset := state.LogicalScrollTop(); offset.ItemIx != 0 || offset.OffsetInItem != Px(0.) {
			t.Fatalf("after 0: %+v", offset)
		}
	})
}

func TestList_ChildScrollHandlerCanStopListScroll(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.))
		childSawEvent := false

		listTestDrawView(cx, NewSize(Px(100.), Px(100.)), func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().
					H(Px(20.)).
					WFull().
					OnScroll(func(_ *ScrollEvent, _ *Window, cx *App) {
						childSawEvent = true
						cx.StopPropagation()
					}).
					IntoAnyElement()
			})
		})

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(10.)), NewPoint(Px(0.), Px(-30.))))

		if !childSawEvent {
			t.Fatalf("the child's scroll-wheel handler should run")
		}
		if offset := state.LogicalScrollTop(); offset.ItemIx != 0 || offset.OffsetInItem != Px(0.) {
			t.Fatalf("list scrolled to %+v after the child stopped propagation", offset)
		}
	})
}

func listTestCheckViewport(t *testing.T, state ListState, ix int, above, below Option[bool]) {
	t.Helper()
	if got := listTestOpt(state.ItemIsAboveViewport(ix)); got != above {
		t.Fatalf("ItemIsAboveViewport(%d) = %v; want %v", ix, got, above)
	}
	if got := listTestOpt(state.ItemIsBelowViewport(ix)); got != below {
		t.Fatalf("ItemIsBelowViewport(%d) = %v; want %v", ix, got, below)
	}
}

func TestList_ItemViewportQueriesReturnNoneBeforeLayout(t *testing.T) {
	RunTest(t, func(*TestRun) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		listTestCheckViewport(t, state, 0, None[bool](), None[bool]())
	})
}

func TestList_ItemViewportQueriesBeforeLogicalScrollTop(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})
		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		listTestCheckViewport(t, state, 1, Some(true), Some(false))
	})
}

func TestList_ItemViewportQueriesMeasuredItemInsideViewport(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})
		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		listTestCheckViewport(t, state, 2, Some(false), Some(false))
	})
}

func TestList_ItemViewportQueriesMeasuredItemAboveViewport(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(20.)})
		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		listTestCheckViewport(t, state, 2, Some(true), Some(false))
	})
}

func TestList_ItemViewportQueriesMeasuredItemBelowViewport(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})
		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		listTestCheckViewport(t, state, 3, Some(false), Some(true))
	})
}

func TestList_ItemViewportQueriesRemainStableWithZeroHeightViewport(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(0.)})
		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		listTestCheckViewport(t, state, 3, Some(false), Some(true))

		listTestDrawTestListView(cx, NewSize(Px(100.), Px(0.)), state)

		listTestCheckViewport(t, state, 1, Some(true), Some(false))
		listTestCheckViewport(t, state, 3, Some(false), Some(true))
	})
}

func TestList_ItemViewportQueriesAfterScrollToEndBeforeLayout(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(5, ListAlignmentTop, Px(10.)).MeasureAll()

		listTestDrawTestListView(cx, NewSize(Px(100.), Px(20.)), state)

		state.ScrollToEnd()

		if got := state.LogicalScrollTop().ItemIx; got != state.ItemCount() {
			t.Fatalf("item_ix %d; want %d", got, state.ItemCount())
		}
		listTestCheckViewport(t, state, 0, Some(true), Some(false))
	})
}

func TestList_MeasureAllAfterWidthChange(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(10, ListAlignmentTop, Px(0.)).MeasureAll()

		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().H(Px(50.)).WFull().IntoAnyElement()
			})
		})

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view.Clone() })
		if got := state.MaxOffsetForScrollbar().Y; got != Px(300.) {
			t.Fatalf("max offset %v; want 300", got)
		}

		listTestDrawView(cx, NewSize(Px(200.), Px(200.)), func(*App) Entity[listTestClosureView] { return view })
		if got := state.MaxOffsetForScrollbar().Y; got != Px(300.) {
			t.Fatalf("max offset after width change %v; want 300", got)
		}
	})
}

func listTestUpdateR[R any](cx *VisualTestContext, f func(cx *App) R) R {
	return cx.UpdateR(func(_ *Window, cx *App) R { return f(cx) })
}

func TestList_Remeasure(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		itemHeight := 100
		state := NewListState(10, ListAlignmentTop, Px(10.))

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

		state.ScrollTo(ListOffset{ItemIx: 2, OffsetInItem: Px(40.)})

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view.Clone() })

		if offset := state.LogicalScrollTop(); offset.ItemIx != 2 || offset.OffsetInItem != Px(40.) {
			t.Fatalf("offset %+v; want item 2 offset 40", offset)
		}

		itemHeight = 50
		state.Remeasure()

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view })

		if offset := state.LogicalScrollTop(); offset.ItemIx != 2 || offset.OffsetInItem != Px(20.) {
			t.Fatalf("offset %+v; want item 2 offset 20", offset)
		}
	})
}

func TestList_RemeasureItemPreservesScrollOffset(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		itemHeight := 100
		state := NewListState(20, ListAlignmentTop, Px(10.))

		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return cx.New(func(*Context[listTestClosureView]) *listTestClosureView {
				return &listTestClosureView{render: func(*Window, *Context[listTestClosureView]) IntoElement {
					height := itemHeight
					return NewList(state, func(index int, _ *Window, _ *App) AnyElement {
						h := 100
						if index == 5 {
							h = height
						}
						return NewDiv().H(Px(float32(h))).WFull().IntoAnyElement()
					}).WFull().HFull()
				}}
			})
		})

		state.ScrollTo(ListOffset{ItemIx: 5, OffsetInItem: Px(40.)})

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view.Clone() })

		itemHeight = 200
		state.RemeasureItems(Range{Start: 5, End: 6})

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view })

		if offset := state.LogicalScrollTop(); offset.ItemIx != 5 || offset.OffsetInItem != Px(40.) {
			t.Fatalf("offset %+v; want item 5 offset 40", offset)
		}
	})
}

func TestList_ProvisionalItemHeightsKeepScrollAnchor(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(20, ListAlignmentTop, Px(100.))
		var painted []listTestPainted
		firstRendered := None[int]()
		ready := listTestItemReadiness(state, false)
		state.SetFollowMode(FollowModeTail)
		state.ScrollTo(listTestOffset(10, 40.))

		for _, step := range []struct {
			height           float32
			measurementReady bool
		}{{0., false}, {100., false}, {100., true}} {
			*ready = step.measurementReady
			painted = painted[:0]
			firstRendered = None[int]()
			height := step.height
			listTestDrawList(state, 200., func(index int) AnyElement {
				firstRendered = Some(min(firstRendered.UnwrapOr(index), index))
				return listTestPaintedItem(index, height, &painted)
			}, cx)
			if got := state.LogicalScrollTop(); got != listTestOffset(10, 40.) {
				t.Fatalf("scroll top %+v", got)
			}
			if state.IsFollowingTail() {
				t.Fatalf("following tail")
			}
			want := 10
			if step.measurementReady {
				want = 9
			}
			if firstRendered != Some(want) {
				t.Fatalf("first rendered %v; want %d", firstRendered, want)
			}
			if height > 0. {
				expected := []listTestPainted{
					{10, Px(-40.), Px(60.)},
					{11, Px(60.), Px(160.)},
					{12, Px(160.), Px(260.)},
				}
				if !slices.Equal(painted, expected) {
					t.Fatalf("painted %v; want %v", painted, expected)
				}
			}
		}
	})
}

func TestList_UnreadyMeasurementsPreservePendingFraction(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, c := range []struct {
			provisionalHeight Option[float32]
			destination       Option[float32]
			finalHeight       float32
			expectedOffset    float32
		}{
			{Some[float32](200.), None[float32](), 200., 100.},
			{None[float32](), Some[float32](40.), 200., 80.},
			{Some[float32](200.), Some[float32](50.), 200., 50.},
			{Some[float32](0.), Some[float32](40.), 20., 20.},
			{Some[float32](10.), Some[float32](40.), 100., 40.},
			{Some[float32](200.), Some[float32](50.), 300., 50.},
		} {
			state := NewListState(10, ListAlignmentTop, Px(0.))
			var painted []listTestPainted
			draw := func(height float32) {
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					h := float32(100.)
					if index == 5 {
						h = height
					}
					return listTestPaintedItem(index, h, &painted)
				}, cx)
			}
			ready := listTestItemReadiness(state, true)
			state.ScrollTo(listTestOffset(5, 50.))
			draw(100.)
			*ready = false
			state.Remeasure()
			if height, ok := c.provisionalHeight.Get(); ok {
				for _, height := range []float32{0., height, height} {
					draw(height)
					if got := state.LogicalScrollTop(); got != listTestOffset(5, 50.) {
						t.Fatalf("%+v: provisional scroll top %+v", c, got)
					}
				}
			}
			if offset, ok := c.destination.Get(); ok {
				state.ScrollTo(listTestOffset(5, offset))
			}
			*ready = true
			for range 2 {
				draw(c.finalHeight)
				if got := state.LogicalScrollTop(); got != listTestOffset(5, c.expectedOffset) {
					t.Fatalf("%+v: scroll top %+v", c, got)
				}
				want := Some(listTestPainted{5, Px(-c.expectedOffset), Px(c.finalHeight - c.expectedOffset)})
				if got := listTestFirst(painted); got != want {
					t.Fatalf("%+v: first painted %v; want %v", c, got, want)
				}
			}
		}
	})
}

func TestList_NavigationCancelsProvisionalAndPaintsDestination(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, measured := range []bool{false, true} {
			upDestination := 6
			if measured {
				upDestination = 0
			}
			navigations := []struct {
				name        string
				navigate    func(ListState)
				destination int
				top         float32
			}{
				{"scroll_to", func(state ListState) { state.ScrollTo(listTestOffset(12, 0.)) }, 12, 0.},
				{"end", ListState.ScrollToEnd, 19, 150.},
				{"down", func(state ListState) { state.ScrollBy(Px(5000.)) }, 19, 150.},
				{"up", func(state ListState) { state.ScrollBy(Px(-5000.)) }, upDestination, 0.},
				{"reveal_below", func(state ListState) { state.ScrollToRevealItem(10) }, 10, 150.},
				{"reveal_above", func(state ListState) { state.ScrollToRevealItem(0) }, 0, 0.},
				{"follow_tail", func(state ListState) { state.SetFollowMode(FollowModeTail) }, 19, 150.},
			}
			for _, nav := range navigations {
				state := NewListState(20, ListAlignmentTop, Px(0.))
				if measured {
					state = state.MeasureAll()
				}
				var painted []listTestPainted
				draw := func() {
					painted = painted[:0]
					listTestDrawList(state, 200., func(index int) AnyElement {
						h := float32(50.)
						if index == 5 {
							h = 0.
						}
						return listTestPaintedItem(index, h, &painted)
					}, cx)
				}
				state.SetItemMeasurementReadiness(func(index int, _ *App) bool { return index != 5 })
				state.ScrollTo(listTestOffset(5, 20.))
				state.RemeasureItems(Range{Start: 5, End: 6})
				draw()
				if !state.inner.isProvisional() {
					t.Fatalf("%s: not provisional after first draw", nav.name)
				}
				nav.navigate(state)
				if state.inner.isProvisional() {
					t.Fatalf("%s: provisional after navigation", nav.name)
				}
				draw()
				if state.inner.isProvisional() {
					t.Fatalf("%s: provisional after second draw", nav.name)
				}
				want := Some(listTestPainted{nav.destination, Px(nav.top), Px(nav.top + 50.)})
				if got := listTestFind(painted, nav.destination); got != want {
					t.Fatalf("%s, measured=%v: %v; want %v", nav.name, measured, got, want)
				}
			}
		}

		navigations := []struct {
			name     string
			navigate func(ListState)
		}{
			{"end_sentinel", func(state ListState) { state.ScrollTo(listTestOffset(3, 0.)) }},
			{"scroll_to_end", ListState.ScrollToEnd},
			{"follow_tail", func(state ListState) { state.SetFollowMode(FollowModeTail) }},
			{"horizontal", ListState.ScrollToEnd},
			{"pause_tail", func(state ListState) { state.SetFollowMode(FollowModeTail) }},
			{"disable_tail", func(state ListState) { state.SetFollowMode(FollowModeTail) }},
		}
		for _, nav := range navigations {
			name := nav.name
			state := NewListState(3, ListAlignmentTop, Px(0.)).MeasureAll()
			ready := listTestItemReadiness(state, false)
			var painted []listTestPainted
			draw := func(height float32) {
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					return listTestPaintedItem(index, height, &painted)
				}, cx)
			}
			nav.navigate(state)
			for _, step := range []struct {
				height           float32
				measurementReady bool
				expected         []listTestPainted
			}{
				{100., false, []listTestPainted{{1, Px(0.), Px(100.)}, {2, Px(100.), Px(200.)}}},
				{150., true, []listTestPainted{{1, Px(-100.), Px(50.)}, {2, Px(50.), Px(200.)}}},
			} {
				if step.measurementReady {
					switch name {
					case "horizontal":
						cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(20.), Px(0.))))
					case "pause_tail":
						state.PauseFollowingTail()
					case "disable_tail":
						state.SetFollowMode(FollowModeNormal)
					}
				}
				*ready = step.measurementReady
				draw(step.height)
				if !step.measurementReady {
					if got := state.ScrollPxOffsetForScrollbar().Y; got != Px(-100.) {
						t.Fatalf("%s: scrollbar offset %v", name, got)
					}
					if got := state.LogicalScrollTop(); got != listTestOffset(1, 0.) {
						t.Fatalf("%s: scroll top %+v", name, got)
					}
					if got := listTestOpt(state.ItemIsAboveViewport(2)); got != None[bool]() {
						t.Fatalf("%s: above viewport %v", name, got)
					}
				}
				paused := name == "pause_tail" || name == "disable_tail"
				expected := step.expected
				if step.measurementReady && paused {
					expected = []listTestPainted{{1, Px(0.), Px(150.)}, {2, Px(150.), Px(300.)}}
				}
				if got, want := state.IsFollowingTail(), name == "follow_tail" || (paused && !step.measurementReady); got != want {
					t.Fatalf("%s: following tail %v; want %v", name, got, want)
				}
				if !slices.Equal(painted, expected) {
					t.Fatalf("%s: painted %v; want %v", name, painted, expected)
				}
			}
			*ready = false
			nav.navigate(state)
			draw(150.)
			state.ScrollBy(Px(-20.))
			draw(150.)
			if got := state.LogicalScrollTop(); got != listTestOffset(1, 80.) {
				t.Fatalf("%s: final scroll top %+v", name, got)
			}
			if state.IsFollowingTail() {
				t.Fatalf("%s: following tail at the end", name)
			}
			expected := []listTestPainted{{1, Px(-80.), Px(70.)}, {2, Px(70.), Px(220.)}}
			if !slices.Equal(painted, expected) {
				t.Fatalf("%s: final painted %v; want %v", name, painted, expected)
			}
		}
	})
}

func TestList_UnreadyOverdrawAndBackfillPreserveScrollAnchor(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, c := range []struct {
			overdraw    float32
			anchor      ListOffset
			unreadyItem int
		}{
			{100., listTestOffset(0, 0.), 2},
			{0., listTestOffset(2, 40.), 1},
		} {
			state := NewListState(3, ListAlignmentTop, Px(c.overdraw))
			ready := false
			unreadyItem := c.unreadyItem
			state.SetItemMeasurementReadiness(func(index int, _ *App) bool { return index != unreadyItem || ready })
			var painted []listTestPainted
			draw := func(height float32) {
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					h := float32(100.)
					if index == unreadyItem {
						h = height
					}
					return listTestPaintedItem(index, h, &painted)
				}, cx)
			}
			state.SetFollowMode(FollowModeTail)
			state.ScrollTo(c.anchor)
			draw(0.)
			if got := state.LogicalScrollTop(); got != c.anchor {
				t.Fatalf("%+v: scroll top %+v", c, got)
			}
			if state.IsFollowingTail() {
				t.Fatalf("%+v: following tail", c)
			}
			expected := []listTestPainted{{2, Px(-40.), Px(60.)}}
			if unreadyItem == 2 {
				expected = []listTestPainted{{0, Px(0.), Px(100.)}, {1, Px(100.), Px(200.)}}
			}
			if !slices.Equal(painted, expected) {
				t.Fatalf("%+v: painted %v; want %v", c, painted, expected)
			}
			ready = true
			draw(500.)
			if unreadyItem == 2 {
				if got := state.LogicalScrollTop(); got != c.anchor {
					t.Fatalf("%+v: ready scroll top %+v", c, got)
				}
				if state.IsFollowingTail() {
					t.Fatalf("%+v: following tail when ready", c)
				}
				if got := state.MaxOffsetForScrollbar().Y; got != Px(500.) {
					t.Fatalf("%+v: max offset %v", c, got)
				}
				expected = []listTestPainted{{0, Px(0.), Px(100.)}, {1, Px(100.), Px(200.)}}
			} else {
				if got := state.LogicalScrollTop(); got != listTestOffset(1, 400.) {
					t.Fatalf("%+v: ready scroll top %+v", c, got)
				}
				if !state.IsFollowingTail() {
					t.Fatalf("%+v: not following tail when ready", c)
				}
				expected = []listTestPainted{{1, Px(-400.), Px(100.)}, {2, Px(100.), Px(200.)}}
			}
			if !slices.Equal(painted, expected) {
				t.Fatalf("%+v: ready painted %v; want %v", c, painted, expected)
			}
		}
	})
}

func TestList_UnreadyOverdrawStopsAtFirstPendingItem(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, anchor := range []int{0, 10} {
			state := NewListState(13, ListAlignmentTop, Px(100.))
			ready := false
			initiallyReady := func(index int) bool {
				if anchor == 0 {
					return index < 2
				}
				return index >= 10
			}
			state.SetItemMeasurementReadiness(func(index int, _ *App) bool { return initiallyReady(index) || ready })
			state.ScrollTo(listTestOffset(anchor, 0.))
			var rendered []int
			var painted []listTestPainted
			expectedFrames := [][]int{{0, 1, 2}, {0, 1, 2}, {0, 1, 2}, {0, 1}}
			if anchor != 0 {
				expectedFrames = [][]int{{10, 11, 12, 9}, {10, 11, 9}, {10, 11, 9}, {10, 11}}
			}
			for frame, expectedRendered := range expectedFrames {
				measurementReady := frame >= 2
				ready = measurementReady
				rendered = rendered[:0]
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					rendered = append(rendered, index)
					height := float32(0.)
					if initiallyReady(index) || measurementReady {
						height = 100.
					}
					return listTestPaintedItem(index, height, &painted)
				}, cx)
				if !slices.Equal(rendered, expectedRendered) {
					t.Fatalf("anchor %d frame %d: rendered %v; want %v", anchor, frame, rendered, expectedRendered)
				}
				if got := state.ItemCount(); got != 13 {
					t.Fatalf("item count %d", got)
				}
				if got := state.LogicalScrollTop(); got != listTestOffset(anchor, 0.) {
					t.Fatalf("anchor %d frame %d: scroll top %+v", anchor, frame, got)
				}
				expected := []listTestPainted{{anchor, Px(0.), Px(100.)}, {anchor + 1, Px(100.), Px(200.)}}
				if !slices.Equal(painted, expected) {
					t.Fatalf("anchor %d frame %d: painted %v; want %v", anchor, frame, painted, expected)
				}
			}
		}
	})
}

func TestList_ScrollbarAndResetDoNotResurrectProvisional(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(20, ListAlignmentTop, Px(0.))
		var painted []listTestPainted
		draw := func() {
			painted = painted[:0]
			listTestDrawList(state, 200., func(index int) AnyElement {
				h := float32(50.)
				if index == 5 {
					h = 0.
				}
				return listTestPaintedItem(index, h, &painted)
			}, cx)
		}
		state.SetItemMeasurementReadiness(func(index int, _ *App) bool { return index != 5 })
		if state.inner.isProvisional() {
			t.Fatalf("provisional before layout")
		}
		state.SetFollowMode(FollowModeTail)
		for _, interveningFrame := range []bool{false, true} {
			state.ScrollTo(listTestOffset(5, 20.))
			state.RemeasureItems(Range{Start: 5, End: 6})
			draw()
			state.ScrollbarDragStarted()
			if interveningFrame {
				draw()
				if !state.inner.isProvisional() {
					t.Fatalf("not provisional after intervening frame")
				}
			}
			state.SetOffsetFromScrollbar(NewPoint(Px(0.), Px(-10_000.)))
			if state.inner.isProvisional() {
				t.Fatalf("provisional after scrollbar offset")
			}
			state.ScrollbarDragEnded()
			if state.IsScrollbarDragging() {
				t.Fatalf("still dragging")
			}
			draw()
			if got, want := listTestLast(painted), Some(listTestPainted{19, Px(150.), Px(200.)}); got != want {
				t.Fatalf("last painted %v; want %v", got, want)
			}
			if state.inner.isProvisional() {
				t.Fatalf("provisional after drag draw")
			}
			state.ScrollTo(listTestOffset(5, 20.))
			if state.inner.isProvisional() {
				t.Fatalf("provisional after scroll_to")
			}
			draw()
			if !state.inner.isProvisional() {
				t.Fatalf("not provisional after scroll_to draw")
			}
			if got := state.LogicalScrollTop(); got != listTestOffset(5, 20.) {
				t.Fatalf("scroll top %+v", got)
			}
		}
		state.RemeasureItems(Range{Start: 5, End: 6})
		state.Reset(20)
		if state.inner.isProvisional() {
			t.Fatalf("provisional after reset")
		}
		draw()
		if state.inner.isProvisional() {
			t.Fatalf("provisional after reset draw")
		}
		if got := state.LogicalScrollTop(); got != (ListOffset{}) {
			t.Fatalf("scroll top after reset %+v", got)
		}
		if got, want := listTestFirst(painted), Some(listTestPainted{0, Px(0.), Px(50.)}); got != want {
			t.Fatalf("first painted %v; want %v", got, want)
		}
	})
}

func TestList_SplicePreservesOnlySurvivingScrollOffsets(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		var painted []listTestPainted
		draw := func(state ListState, height float32) {
			painted = painted[:0]
			listTestDrawList(state, 200., func(index int) AnyElement {
				return listTestPaintedItem(index, height, &painted)
			}, cx)
		}
		for _, c := range []struct {
			provisional      bool
			replacementCount int
		}{{true, 0}, {false, 1}} {
			state := NewListState(20, ListAlignmentTop, Px(0.))
			ready := listTestItemReadiness(state, !c.provisional)
			state.ScrollTo(listTestOffset(8, 50.))
			state.RemeasureItems(Range{Start: 8, End: 9})
			if c.provisional {
				draw(state, 100.)
			}
			state.Splice(Range{Start: 8, End: 9}, c.replacementCount)
			if state.inner.isProvisional() {
				t.Fatalf("%+v: provisional after splice", c)
			}
			*ready = true
			draw(state, 100.)
			if got := state.LogicalScrollTop(); got != listTestOffset(8, 0.) {
				t.Fatalf("%+v: scroll top %+v", c, got)
			}
			if got, want := listTestFirst(painted), Some(listTestPainted{8, Px(0.), Px(100.)}); got != want {
				t.Fatalf("%+v: first painted %v; want %v", c, got, want)
			}
		}
		state := NewListState(20, ListAlignmentTop, Px(0.)).MeasureAll()
		state.ScrollTo(listTestOffset(8, 50.))
		draw(state, 100.)
		state.Remeasure()
		state.Splice(Range{Start: 0, End: 2}, 0)
		draw(state, 80.)
		if got := state.LogicalScrollTop(); got != listTestOffset(6, 40.) {
			t.Fatalf("scroll top %+v; want 6/40", got)
		}
		if got, want := listTestFirst(painted), Some(listTestPainted{6, Px(-40.), Px(40.)}); got != want {
			t.Fatalf("first painted %v; want %v", got, want)
		}
		ready := listTestItemReadiness(state, false)
		state.RemeasureItems(Range{Start: 6, End: 7})
		state.Splice(Range{Start: 8, End: 10}, 3)
		draw(state, 100.)
		if !state.inner.isProvisional() {
			t.Fatalf("not provisional")
		}
		if got := state.LogicalScrollTop(); got != listTestOffset(6, 40.) {
			t.Fatalf("scroll top %+v; want 6/40", got)
		}
		state.Splice(Range{Start: 0, End: 0}, 2)
		if state.inner.isProvisional() {
			t.Fatalf("provisional after leading splice")
		}
		draw(state, 100.)
		if !state.inner.isProvisional() {
			t.Fatalf("not provisional after redraw")
		}
		if got := state.LogicalScrollTop(); got != listTestOffset(8, 40.) {
			t.Fatalf("scroll top %+v; want 8/40", got)
		}
		*ready = true
		draw(state, 100.)
		if got := state.LogicalScrollTop(); got != listTestOffset(8, 40.) {
			t.Fatalf("ready scroll top %+v; want 8/40", got)
		}
		if got, want := listTestFirst(painted), Some(listTestPainted{8, Px(-40.), Px(60.)}); got != want {
			t.Fatalf("ready first painted %v; want %v", got, want)
		}
	})
}

func TestList_ChildAutoscrollOverridesPendingAndReleaseFillsViewport(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, c := range []struct {
			count        int
			child        int
			requestStart float32
			requestEnd   float32
			expected     ListOffset
			released     ListOffset
		}{
			{10, 2, 40.0, 45.0, listTestOffset(2, 40.), listTestOffset(2, 40.)},
			{3, 2, 40.0, 45.0, listTestOffset(2, 40.), listTestOffset(1, 0.)},
			{10, 4, 80.0, 90.0, listTestOffset(2, 90.), listTestOffset(2, 90.)},
			{10, 2, -130.0, 5.0, listTestOffset(0, 70.), listTestOffset(0, 70.)},
			{10, 2, -250.0, 5.0, listTestOffset(0, 0.), listTestOffset(0, 0.)},
		} {
			state := NewListState(c.count, ListAlignmentTop, Px(0.))
			requested := false
			var painted []listTestPainted
			draw := func() {
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					return NewCanvas(
						func(bounds Bounds[Pixels], window *Window, _ *App) struct{} {
							if index == c.child && !requested {
								requested = true
								window.RequestAutoscroll(BoundsFromCorners(
									NewPoint(bounds.Left(), bounds.Top()+Px(c.requestStart)),
									NewPoint(bounds.Right(), bounds.Top()+Px(c.requestEnd)),
								))
							}
							return struct{}{}
						},
						func(bounds Bounds[Pixels], _ struct{}, _ *Window, _ *App) {
							painted = append(painted, listTestPainted{index, bounds.Top(), bounds.Bottom()})
						},
					).
						H(Px(100.)).
						WFull().
						IntoAnyElement()
				}, cx)
			}
			ready := listTestItemReadiness(state, false)
			state.ScrollTo(listTestOffset(2, 50.))
			state.RemeasureItems(Range{Start: 2, End: 3})
			for range 2 {
				draw()
				if !requested {
					t.Fatalf("%+v: autoscroll not requested", c)
				}
				if !state.inner.isProvisional() {
					t.Fatalf("%+v: not provisional", c)
				}
				if got := state.LogicalScrollTop(); got != c.expected {
					t.Fatalf("%+v: scroll top %+v", c, got)
				}
				want := Some(listTestPainted{c.expected.ItemIx, -c.expected.OffsetInItem, Px(100.) - c.expected.OffsetInItem})
				if got := listTestFirst(painted); got != want {
					t.Fatalf("%+v: first painted %v; want %v", c, got, want)
				}
			}
			*ready = true
			draw()
			if state.inner.isProvisional() {
				t.Fatalf("%+v: provisional after release", c)
			}
			if got := state.LogicalScrollTop(); got != c.released {
				t.Fatalf("%+v: released scroll top %+v", c, got)
			}
			visibleCount := 3
			if c.released.OffsetInItem == Px(0.) {
				visibleCount = 2
			}
			var expected []listTestPainted
			for index := range visibleCount {
				top := Px(float32(index)*100.) - c.released.OffsetInItem
				expected = append(expected, listTestPainted{c.released.ItemIx + index, top, top + Px(100.)})
			}
			if !slices.Equal(painted, expected) {
				t.Fatalf("%+v: released painted %v; want %v", c, painted, expected)
			}
		}
	})
}

func TestList_RemeasureThenScrollDoesNotRevertScrollPosition(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		state := NewListState(20, ListAlignmentTop, Px(10.))

		view := listTestUpdateR(cx, func(cx *App) Entity[listTestClosureView] {
			return listTestNewView(cx, state, func(int, *Window, *App) AnyElement {
				return NewDiv().H(Px(100.)).WFull().IntoAnyElement()
			})
		})

		state.ScrollTo(ListOffset{ItemIx: 5, OffsetInItem: Px(40.)})

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view.Clone() })

		state.RemeasureItems(Range{Start: 5, End: 6})

		cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(-30.))))

		if offset := state.LogicalScrollTop(); offset.ItemIx != 5 || offset.OffsetInItem != Px(70.) {
			t.Fatalf("offset %+v; want item 5 offset 70", offset)
		}

		listTestDrawView(cx, NewSize(Px(100.), Px(200.)), func(*App) Entity[listTestClosureView] { return view })

		if offset := state.LogicalScrollTop(); offset.ItemIx != 5 || offset.OffsetInItem != Px(70.) {
			t.Fatalf("scrolling after a remeasure should not be reverted by the stale pending scroll: %+v", offset)
		}
	})
}

func TestList_ScrollAfterRemeasureClampsToShrunkItemHeight(t *testing.T) {
	listTestRun(t, func(t *testing.T, cx *VisualTestContext) {
		for _, provisional := range []bool{false, true} {
			state := NewListState(20, ListAlignmentTop, Px(10.))
			var painted []listTestPainted
			draw := func(height float32) {
				painted = painted[:0]
				listTestDrawList(state, 200., func(index int) AnyElement {
					h := float32(100.)
					if index == 5 {
						h = height
					}
					return listTestPaintedItem(index, h, &painted)
				}, cx)
			}
			ready := listTestItemReadiness(state, true)
			state.ScrollTo(listTestOffset(5, 40.))
			draw(100.)
			*ready = !provisional
			state.RemeasureItems(Range{Start: 5, End: 6})
			if provisional {
				draw(100.)
			}
			*ready = true
			cx.SimulateEvent(listTestScrollEvent(NewPoint(Px(50.), Px(100.)), NewPoint(Px(0.), Px(-30.))))
			if got := state.LogicalScrollTop(); got != listTestOffset(5, 70.) {
				t.Fatalf("provisional=%v: scroll top %+v", provisional, got)
			}
			for range 2 {
				draw(50.)
				if got := state.LogicalScrollTop(); got != listTestOffset(5, 50.) {
					t.Fatalf("provisional=%v: clamped scroll top %+v", provisional, got)
				}
				if got, want := listTestFind(painted, 6), Some(listTestPainted{6, Px(0.), Px(100.)}); got != want {
					t.Fatalf("provisional=%v: item 6 %v; want %v", provisional, got, want)
				}
			}
		}
	})
}
