// Package lists holds the snippets of docs/guide/lists-and-scrolling.md.
package lists

import (
	"fmt"

	"github.com/megascan/gpui-go"
)

func weakProcessor[T, E, R any](cx *gpui.Context[T], f func(this *T, event E, window *gpui.Window, cx *gpui.Context[T]) R) func(event E, window *gpui.Window, cx *gpui.App) R {
	view := cx.WeakEntity()
	return func(event E, window *gpui.Window, cx *gpui.App) R {
		result, _ := view.UpdateR(cx, func(this *T, cx *gpui.Context[T]) R {
			return f(this, event, window, cx)
		})
		return result
	}
}

// Table shows many rows of the same height.
type Table struct {
	rows         []string
	scrollHandle gpui.UniformListScrollHandle
	lastVisible  gpui.Range
}

// NewTable creates a table with count rows.
func NewTable(count int) *Table {
	rows := make([]string, count)
	for ix := range rows {
		rows[ix] = fmt.Sprintf("Row %d", ix)
	}
	return &Table{rows: rows, scrollHandle: gpui.NewUniformListScrollHandle()}
}

// Reveal scrolls row ix into view.
func (t *Table) Reveal(ix int, cx *gpui.Context[Table]) {
	t.scrollHandle.ScrollToItem(ix, gpui.ScrollStrategyCenter)
	cx.Notify()
}

// Render renders the view.
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

// Log shows lines of different heights, newest at the bottom.
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

// Render renders the view.
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

// Notes is a plain scrollable column.
type Notes struct {
	notes        []string
	scrollHandle gpui.ScrollHandle
}

// NewNotes creates the view.
func NewNotes(notes []string) *Notes {
	return &Notes{notes: notes, scrollHandle: gpui.NewScrollHandle()}
}

// ScrollToTop scrolls back to the first note.
func (n *Notes) ScrollToTop(cx *gpui.Context[Notes]) {
	n.scrollHandle.SetOffset(gpui.NewPoint(gpui.Px(0), gpui.Px(0)))
	cx.Notify()
}

// Render renders the view.
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
