package async

import (
	"testing"
	"time"

	"github.com/megascan/gpui-go"
)

func TestClock_Ticks(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		clock := cx.New(NewClock)
		defer clock.Release()

		cx.Executor().AdvanceClock(3 * time.Second)
		cx.RunUntilParked()

		got := clock.ReadWith(cx, func(c *Clock, _ *gpui.App) int { return c.seconds })
		if got != 3 {
			t.Fatalf("seconds = %d, want 3", got)
		}
	})
}

func TestReport_Analyze(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		report := cx.New(NewReport)
		defer report.Release()

		report.Update(cx, func(r *Report, cx *gpui.Context[Report]) {
			r.Analyze("one two three", cx)
		})
		cx.RunUntilParked()

		got := report.ReadWith(cx, func(r *Report, _ *gpui.App) string { return r.status })
		if got != "3 words" {
			t.Fatalf("status = %q, want %q", got, "3 words")
		}
	})
}

func TestReport_WaitForAnswer(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		report := cx.New(NewReport)
		defer report.Release()

		cx.Update(func(cx *gpui.App) {
			waitForAnswer(report.Downgrade(), cx)
		})
		cx.RunUntilParked()

		got := report.ReadWith(cx, func(r *Report, _ *gpui.App) string { return r.status })
		if got != "42" {
			t.Fatalf("status = %q, want %q", got, "42")
		}
	})
}
