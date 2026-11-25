//go:build securityharness

package s15

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/scheduler"
)

var r2Loads atomic.Int64

type countAsset struct{}

func (countAsset) Load(source int, cx *gpui.App) func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
	return func(*scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
		r2Loads.Add(1)
		return scheduler.Result[*gpui.RenderImage]{Err: fmt.Errorf("missing %d", source)}
	}
}

type r2GalleryView struct{ n int }

func (v *r2GalleryView) Render(window *gpui.Window, cx *gpui.Context[r2GalleryView]) gpui.IntoElement {
	r2Renders.Add(1)
	for i := range v.n {
		window.UseAsset[countAsset](i, cx.App)
	}
	return gpui.NewDiv()
}

var r2Renders atomic.Int64

func galleryChild(t *testing.T, n int) func(string) (string, error) {
	return func(string) (string, error) {
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				case <-time.After(2 * time.Second):
					fmt.Printf("progress: %d loads started, %d renders\n", r2Loads.Load(), r2Renders.Load())
				}
			}
		}()
		var out string
		gpui.RunTest(t, func(tc *gpui.TestRun) {
			tcx := tc.Cx()
			view, cx := tcx.AddWindowView(func(*gpui.Window, *gpui.Context[r2GalleryView]) *r2GalleryView { return &r2GalleryView{n: n} })
			cx.RunUntilParked()
			out = fmt.Sprintf("%d images settled after %d loads, %d renders", n, r2Loads.Load(), r2Renders.Load())
			view.Release()
		})
		close(done)
		return out, nil
	}
}

func TestR2_S15_04_LruControlUnderCapacitySettles(t *testing.T) {
	o, parent := bombchild.Run(t, childLimit, 40*time.Second, "", galleryChild(t, 4000))
	if parent {
		t.Logf("%s", o)
		if o.Verdict != "ok" {
			t.Fatalf("harness-error: 4000-image control did not settle: %s", o)
		}
	}
}

func TestR2_S15_05_LruThrashOverCapacityNeverSettles(t *testing.T) {
	const n = 4200
	o, parent := bombchild.Run(t, childLimit, 40*time.Second, "", galleryChild(t, n))
	if parent {
		last := ""
		for _, line := range splitLines(o.Output) {
			if len(line) > 9 && line[:9] == "progress:" {
				last = line
			}
		}
		t.Logf("%s; last %s", o, last)
		if o.Verdict == "timeout" {
			t.Errorf("THRASH: a view with %d distinct images (cache holds 4096) never settles: eviction notifies the views, which re-request and evict the next entry; %s after 40s", n, last)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}
