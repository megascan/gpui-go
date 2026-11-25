# Lists and scrolling

GPUI has three ways to show content that does not fit:

| You have | Use |
|---|---|
| a few dozen children of any size | a scrollable div (`OverflowYScroll`) |
| many rows that all have the same height | `gpui.NewUniformList` |
| many rows of different heights | `gpui.NewList` with a `gpui.ListState` |

The two list elements only build the rows that are visible. The code in this guide is in [snippets/lists/lists.go](snippets/lists/lists.go), with tests in [lists_test.go](snippets/lists/lists_test.go).

## A scrollable div

Give the div an id, a bounded height and an overflow mode. Keep a `gpui.ScrollHandle` in your view if you want to read or set the scroll position:

```go
func (n *Notes) Render(_ *gpui.Window, _ *gpui.Context[Notes]) gpui.IntoElement {
	column := gpui.NewDiv().
		ID("notes").
		H(gpui.Px(300)).
		OverflowYScroll().
		TrackScroll(n.scrollHandle).
		Flex().
		FlexCol().
		Gap2()
	for _, note := range n.notes {
		column = column.Child(gpui.NewDiv().P(gpui.Px(8)).ChildText(note))
	}
	return column
}
```

Create the handle once, with `gpui.NewScrollHandle()`, and keep it in the view. A `ScrollHandle` value shares its state across copies, so storing it in a struct field and passing it to `TrackScroll` refer to the same scroll position:

```go
func (n *Notes) ScrollToTop(cx *gpui.Context[Notes]) {
	n.scrollHandle.SetOffset(gpui.NewPoint(gpui.Px(0), gpui.Px(0)))
	cx.Notify()
}
```

`OverflowScroll` scrolls both ways, `OverflowXScroll` horizontally. These methods need `ID`, like every other stateful behavior. The scrolled element needs a bounded size: a fixed height, or a flex child with `Flex1` inside a parent with a fixed height. Without one, the div grows to fit its content and never scrolls.

## Uniform lists

`NewUniformList` measures the first row and assumes every row has that height. It then asks you only for the rows in view:

```go
func (t *Table) Render(_ *gpui.Window, cx *gpui.Context[Table]) gpui.IntoElement {
	return gpui.NewDiv().
		SizeFull().
		Child(
			gpui.NewUniformList(
				"rows",
				len(t.rows),
				weakProcessor(cx, func(this *Table, visible gpui.Range, _ *gpui.Window, _ *gpui.Context[Table]) []*gpui.Stateful[*gpui.Div] {
					this.lastVisible = visible
					items := make([]*gpui.Stateful[*gpui.Div], 0, visible.End-visible.Start)
					for ix := visible.Start; ix < visible.End; ix++ {
						items = append(items, gpui.NewDiv().
							ID(ix).
							H(gpui.Px(24)).
							Px2().
							ChildText(this.rows[ix]))
					}
					return items
				}),
			).
				TrackUniformListScroll(t.scrollHandle).
				HFull(),
		)
}
```

- The callback gets a `gpui.Range` (`Start` inclusive, `End` exclusive) and returns one element per row.
- Give the list a bounded height (`HFull` inside a sized parent here). With unbounded height it lays out every row, which defeats the point.
- Keep a `gpui.UniformListScrollHandle` (from `gpui.NewUniformListScrollHandle()`) in the view and attach it with `TrackUniformListScroll` to scroll from code:

```go
func (t *Table) Reveal(ix int, cx *gpui.Context[Table]) {
	t.scrollHandle.ScrollToItem(ix, gpui.ScrollStrategyCenter)
	cx.Notify()
}
```

The test in [lists_test.go](snippets/lists/lists_test.go) opens a window with 10,000 rows and checks that only a small range was rendered.

### Reading view state from the row callback

The row callback runs during layout, after `Render` has returned, so it needs a way back to the view's state. Rust GPUI uses `cx.processor`. In gpui-go, `cx.Processor` exists but holds a strong handle to the entity that is never released, so strict leak detection reports it in tests and the view is never freed. Use a weak version instead:

```go
func weakProcessor[T, E, R any](cx *gpui.Context[T], f func(this *T, event E, window *gpui.Window, cx *gpui.Context[T]) R) func(event E, window *gpui.Window, cx *gpui.App) R {
	view := cx.WeakEntity()
	return func(event E, window *gpui.Window, cx *gpui.App) R {
		result, _ := view.UpdateR(cx, func(this *T, cx *gpui.Context[T]) R {
			return f(this, event, window, cx)
		})
		return result
	}
}
```

If the rows only need data that does not change while the frame is built, you can also capture a copy of that data in a plain closure.

## Variable-height lists

`NewList` measures rows as they come into view. It keeps its state in a `gpui.ListState`, which you create once and update when your data changes:

```go
type Log struct {
	lines []string
	state gpui.ListState
}

// NewLog creates an empty log.
func NewLog() *Log {
	return &Log{state: gpui.NewListState(0, gpui.ListAlignmentBottom, gpui.Px(200))}
}

// Append adds a line and keeps the view at the end.
func (l *Log) Append(line string, cx *gpui.Context[Log]) {
	n := len(l.lines)
	l.lines = append(l.lines, line)
	l.state.Splice(gpui.Range{Start: n, End: n}, 1)
	l.state.ScrollToEnd()
	cx.Notify()
}
```

- `NewListState(count, alignment, overdraw)`: `ListAlignmentTop` for normal lists, `ListAlignmentBottom` for chat logs and terminals that stick to the end. `overdraw` is how far past the viewport to render, in pixels.
- Tell the state about every change to your data: `Splice(oldRange, newCount)` for inserts, removals and replacements, `Reset(count)` to start over. If the state and your data disagree, the list renders the wrong rows or indexes out of range.
- `ScrollToEnd`, `ScrollToRevealItem(ix)` and `MeasureAll()` (measure every row up front, for accurate scrollbars) cover most needs.

Render it with a callback that returns one row as a `gpui.AnyElement`:

```go
func (l *Log) Render(_ *gpui.Window, cx *gpui.Context[Log]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		Child(
			gpui.NewList(l.state, weakProcessor(cx, func(this *Log, ix int, _ *gpui.Window, _ *gpui.Context[Log]) gpui.AnyElement {
				return gpui.NewDiv().
					P(gpui.Px(4)).
					ChildText(this.lines[ix]).
					IntoAnyElement()
			})).
				Flex1(),
		)
}
```

`ListState` is a value that shares its state across copies, like `ScrollHandle`. Store it in the view and pass it to `NewList` on every render.

## Examples

- [examples/uniform_list](../../examples/uniform_list/main.go): a minimal uniform list.
- [examples/data_table](../../examples/data_table/main.go): a large table with a custom scrollbar drawn on a canvas.
- [examples/list_example](../../examples/list_example/main.go): a bottom-aligned list with variable row heights.
- [examples/scrollable](../../examples/scrollable/main.go): a scrollable div.
