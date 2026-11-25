package lists

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestTable_RendersOnlyVisibleRows(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		table, vcx := tc.Cx().AddWindowView(func(_ *gpui.Window, _ *gpui.Context[Table]) *Table {
			return NewTable(10_000)
		})
		defer table.Release()

		visible := table.ReadWith(vcx, func(t *Table, _ *gpui.App) gpui.Range { return t.lastVisible })
		if visible.End == 0 || visible.End-visible.Start >= 10_000 {
			t.Fatalf("visible range = %+v, want a small window of rows", visible)
		}
	})
}

func TestLog_Append(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		log, vcx := tc.Cx().AddWindowView(func(_ *gpui.Window, _ *gpui.Context[Log]) *Log {
			return NewLog()
		})
		defer log.Release()

		for ix := range 100 {
			log.Update(vcx, func(l *Log, cx *gpui.Context[Log]) {
				l.Append("line", cx)
			})
			if ix%10 == 0 {
				vcx.RunUntilParked()
			}
		}
		vcx.RunUntilParked()

		got := log.ReadWith(vcx, func(l *Log, _ *gpui.App) int { return l.state.ItemCount() })
		if got != 100 {
			t.Fatalf("items = %d, want 100", got)
		}
	})
}
