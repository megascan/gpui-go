// Package async holds the snippets of docs/guide/async-and-tasks.md.
package async

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

// Clock counts seconds while it exists.
type Clock struct {
	seconds int
	ticker  *gpui.Task[struct{}]
}

// NewClock creates a clock and starts its ticker.
func NewClock(cx *gpui.Context[Clock]) *Clock {
	return &Clock{ticker: startTicker(cx)}
}

func startTicker(cx *gpui.Context[Clock]) *gpui.Task[struct{}] {
	return cx.Spawn(func(this gpui.WeakEntity[Clock], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, fmt.Errorf("tick timer: %w", err)
			}
			err := this.Update(cx, func(clock *Clock, cx *gpui.Context[Clock]) {
				clock.seconds++
				cx.Notify()
			})
			if err != nil {
				return struct{}{}, nil
			}
		}
	})
}

// Reset restarts the ticker from zero.
func (c *Clock) Reset(cx *gpui.Context[Clock]) {
	c.ticker.Cancel()
	c.seconds = 0
	c.ticker = startTicker(cx)
	cx.Notify()
}

// Drop stops the ticker.
func (c *Clock) Drop(_ *gpui.App) {
	c.ticker.Cancel()
}

// Render renders the view.
func (c *Clock) Render(_ *gpui.Window, _ *gpui.Context[Clock]) gpui.IntoElement {
	return gpui.NewDiv().ChildText(fmt.Sprintf("%ds", c.seconds))
}

// Report shows the result of a slow computation.
type Report struct {
	status string
	job    *gpui.Task[struct{}]
}

// NewReport creates an idle report.
func NewReport(_ *gpui.Context[Report]) *Report {
	return &Report{status: "idle", job: scheduler.ReadyTask(struct{}{})}
}

func countWords(text string) int {
	return len(strings.Fields(text))
}

// Analyze counts words on a background thread and stores the result.
func (r *Report) Analyze(text string, cx *gpui.Context[Report]) {
	r.status = "working"
	cx.Notify()
	r.job.Cancel()
	r.job = cx.Spawn(func(this gpui.WeakEntity[Report], cx *gpui.AsyncApp) (struct{}, error) {
		words, err := cx.BackgroundSpawn(func(_ *scheduler.Scope) (int, error) {
			return countWords(text), nil
		}).Await(cx)
		if err != nil {
			return struct{}{}, err
		}
		_ = this.Update(cx, func(report *Report, cx *gpui.Context[Report]) {
			report.status = fmt.Sprintf("%d words", words)
			cx.Notify()
		})
		return struct{}{}, nil
	})
}

// Drop cancels a running analysis.
func (r *Report) Drop(_ *gpui.App) {
	r.job.Cancel()
}

// Render renders the view.
func (r *Report) Render(_ *gpui.Window, _ *gpui.Context[Report]) gpui.IntoElement {
	return gpui.NewDiv().ChildText(r.status)
}

// Title shows a countdown in the window title.
type Title struct{}

// StartCountdown updates the window title once per second, then stops.
func (t *Title) StartCountdown(window *gpui.Window, cx *gpui.Context[Title]) {
	cx.SpawnIn(window, func(_ gpui.WeakEntity[Title], cx *gpui.AsyncWindowContext) (struct{}, error) {
		for n := 3; n > 0; n-- {
			err := cx.Update(func(window *gpui.Window, _ *gpui.App) {
				window.SetWindowTitle(fmt.Sprintf("Closing in %d", n))
			})
			if err != nil {
				return struct{}{}, nil
			}
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	}).Detach()
}

// Render renders the view.
func (t *Title) Render(_ *gpui.Window, _ *gpui.Context[Title]) gpui.IntoElement {
	return gpui.Empty{}
}

var errOffline = errors.New("offline")

func saveSettings(cx *gpui.App) {
	task := cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
		return struct{}{}, errOffline
	})
	gpui.TaskDetachAndLogErr(task, cx)
}

func waitForAnswer(report gpui.WeakEntity[Report], cx *gpui.App) {
	tx, rx := scheduler.Oneshot[string]()
	cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
		answer, err := rx.Recv(cx)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, report.Update(cx, func(report *Report, cx *gpui.Context[Report]) {
			report.status = answer
			cx.Notify()
		})
	}).Detach()
	cx.BackgroundExecutor().Spawn(func(_ *scheduler.Scope) (struct{}, error) {
		return struct{}{}, tx.Send("42")
	}).Detach()
}
