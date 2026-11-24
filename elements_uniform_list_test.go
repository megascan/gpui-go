package gpui

import (
	"fmt"
	"slices"
	"testing"
)

type uniformListTestSelectNext struct{}

func (uniformListTestSelectNext) Name() string         { return "example::SelectNext" }
func (a uniformListTestSelectNext) BoxedClone() Action { return a }
func (uniformListTestSelectNext) PartialEq(other Action) bool {
	_, ok := other.(uniformListTestSelectNext)
	return ok
}

type uniformListTestSelectPrev struct{}

func (uniformListTestSelectPrev) Name() string         { return "example::SelectPrev" }
func (a uniformListTestSelectPrev) BoxedClone() Action { return a }
func (uniformListTestSelectPrev) PartialEq(other Action) bool {
	_, ok := other.(uniformListTestSelectPrev)
	return ok
}

func uniformListTestProcessor[T, E, R any](cx *Context[T], f func(this *T, event E, window *Window, cx *Context[T]) R) func(event E, window *Window, cx *App) R {
	view := cx.WeakEntity()
	return func(e E, window *Window, cx *App) R {
		result, err := view.UpdateR(cx, func(view *T, cx *Context[T]) R { return f(view, e, window, cx) })
		windowTestMust(err)
		return result
	}
}

type uniformListTestNearestView struct {
	index        int
	length       int
	scrollHandle UniformListScrollHandle
	focusHandle  FocusHandle
	visibleRange Range
}

func (v *uniformListTestNearestView) selectNext(_ *uniformListTestSelectNext, window *Window, _ *Context[uniformListTestNearestView]) {
	if v.index+1 == v.length {
		v.index = 0
	} else {
		v.index++
	}
	v.scrollHandle.ScrollToItem(v.index, ScrollStrategyNearest)
	window.Refresh()
}

func (v *uniformListTestNearestView) selectPrevious(_ *uniformListTestSelectPrev, window *Window, _ *Context[uniformListTestNearestView]) {
	if v.index == 0 {
		v.index = v.length - 1
	} else {
		v.index--
	}
	v.scrollHandle.ScrollToItem(v.index, ScrollStrategyNearest)
	window.Refresh()
}

func (v *uniformListTestNearestView) Render(_ *Window, cx *Context[uniformListTestNearestView]) IntoElement {
	selectNext := cx.Listener(func(this *uniformListTestNearestView, action *uniformListTestSelectNext, window *Window, cx *Context[uniformListTestNearestView]) {
		this.selectNext(action, window, cx)
	})
	selectPrevious := cx.Listener(func(this *uniformListTestNearestView, action *uniformListTestSelectPrev, window *Window, cx *Context[uniformListTestNearestView]) {
		this.selectPrevious(action, window, cx)
	})
	return NewDiv().
		ID("list-example").
		TrackFocus(v.focusHandle).
		OnAction(func(action uniformListTestSelectNext, window *Window, cx *App) { selectNext(&action, window, cx) }).
		OnAction(func(action uniformListTestSelectPrev, window *Window, cx *App) { selectPrevious(&action, window, cx) }).
		SizeFull().
		Child(
			NewUniformList(
				"entries",
				v.length,
				uniformListTestProcessor(cx, func(this *uniformListTestNearestView, visibleRange Range, _ *Window, _ *Context[uniformListTestNearestView]) []*Stateful[*Div] {
					this.visibleRange = visibleRange
					items := make([]*Stateful[*Div], 0, visibleRange.End-visibleRange.Start)
					for ix := visibleRange.Start; ix < visibleRange.End; ix++ {
						items = append(items, NewDiv().ID(ix).H(Px(20.0)).ChildText(fmt.Sprintf("Item %d", ix)))
					}
					return items
				}),
			).
				TrackUniformListScroll(v.scrollHandle).
				H(Px(200.0)),
		)
}

func (v *uniformListTestNearestView) Drop(*App) {
	v.focusHandle.Release()
}

func TestUniformList_ScrollStrategyNearest(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		view, cx := tcx.AddWindowView(func(window *Window, cx *Context[uniformListTestNearestView]) *uniformListTestNearestView {
			focusHandle := cx.FocusHandle()
			window.Focus(focusHandle, cx.App)
			return &uniformListTestNearestView{
				scrollHandle: NewUniformListScrollHandle(),
				index:        0,
				focusHandle:  focusHandle,
				length:       47,
				visibleRange: Range{},
			}
		})
		defer func() {
			view.Release()
			windowTestReleaseAll(tcx)
		}()

		check := func(index int, visibleRange Range) {
			t.Helper()
			view.ReadWith(cx, func(view *uniformListTestNearestView, _ *App) struct{} {
				if view.index != index || view.visibleRange != visibleRange {
					t.Fatalf("index %d, range %v; want index %d, range %v", view.index, view.visibleRange, index, visibleRange)
				}
				return struct{}{}
			})
		}

		for ix := 1; ix < 10; ix++ {
			cx.DispatchAction(uniformListTestSelectNext{})
			check(ix, Range{Start: 0, End: 10})
		}

		for ix := 10; ix < 47; ix++ {
			cx.DispatchAction(uniformListTestSelectNext{})
			check(ix, Range{Start: ix - 9, End: ix + 1})
		}

		cx.DispatchAction(uniformListTestSelectNext{})
		check(0, Range{Start: 0, End: 10})

		cx.DispatchAction(uniformListTestSelectPrev{})
		check(46, Range{Start: 37, End: 47})

		for ix := 45; ix >= 37; ix-- {
			cx.DispatchAction(uniformListTestSelectPrev{})
			check(ix, Range{Start: 37, End: 47})
		}

		for ix := 36; ix >= 0; ix-- {
			cx.DispatchAction(uniformListTestSelectPrev{})
			check(ix, Range{Start: ix, End: ix + 10})
		}
	})
}

type uniformListTestItemBounds struct {
	ix     int
	bounds Bounds[Pixels]
}

type uniformListTestLongView struct {
	scrollHandle UniformListScrollHandle
	itemBounds   *[]uniformListTestItemBounds
}

func (v *uniformListTestLongView) Render(_ *Window, cx *Context[uniformListTestLongView]) IntoElement {
	itemBounds := v.itemBounds
	return NewUniformList(
		"entries",
		10_000_000,
		uniformListTestProcessor(cx, func(_ *uniformListTestLongView, visibleRange Range, _ *Window, _ *Context[uniformListTestLongView]) []*Canvas[struct{}] {
			items := make([]*Canvas[struct{}], 0, visibleRange.End-visibleRange.Start)
			for ix := visibleRange.Start; ix < visibleRange.End; ix++ {
				items = append(items, NewCanvas(
					func(bounds Bounds[Pixels], _ *Window, _ *App) struct{} {
						*itemBounds = append(*itemBounds, uniformListTestItemBounds{ix: ix, bounds: bounds})
						return struct{}{}
					},
					func(Bounds[Pixels], struct{}, *Window, *App) {},
				).WFull().H(Px(24.0)))
			}
			return items
		}),
	).
		TrackUniformListScroll(v.scrollHandle).
		H(Px(200.0))
}

func TestUniformList_ItemPositionsFarDownLongList(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		itemBounds := &[]uniformListTestItemBounds{}
		view, cx := tcx.AddWindowView(func(*Window, *Context[uniformListTestLongView]) *uniformListTestLongView {
			return &uniformListTestLongView{scrollHandle: NewUniformListScrollHandle(), itemBounds: itemBounds}
		})
		defer func() {
			view.Release()
			windowTestReleaseAll(tcx)
		}()

		*itemBounds = (*itemBounds)[:0]
		view.Update(cx, func(view *uniformListTestLongView, cx *Context[uniformListTestLongView]) {
			view.scrollHandle.State.BaseHandle.SetOffset(NewPoint(Px(0.0), Px(-216_000_064.0)))
			cx.Notify()
		})
		cx.RunUntilParked()

		type itemTop struct {
			ix  int
			top Pixels
		}
		var itemTops []itemTop
		for _, item := range *itemBounds {
			itemTops = append(itemTops, itemTop{item.ix, item.bounds.Origin.Y})
		}
		var expectedTops []itemTop
		for ix := 9_000_002; ix < 9_000_011; ix++ {
			expectedTops = append(expectedTops, itemTop{ix, Px(-16.0 + 24.0*float32(ix-9_000_002))})
		}
		if !slices.Equal(itemTops, expectedTops) {
			t.Fatalf("item tops = %v, want %v", itemTops, expectedTops)
		}
	})
}
