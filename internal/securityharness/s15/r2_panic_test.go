//go:build securityharness

package s15

import (
	"fmt"
	"os"
	"strings"
	"time"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/scheduler"
)

type panicResultAsset struct{}

func (panicResultAsset) Load(source string, cx *gpui.App) func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
	return func(*scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
		var frames []int
		_ = frames[len(source)+7]
		return scheduler.Result[*gpui.RenderImage]{}
	}
}

type errResultAsset struct{}

func (errResultAsset) Load(source string, cx *gpui.App) func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
	return func(*scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
		return scheduler.Result[*gpui.RenderImage]{Err: fmt.Errorf("decode failed: %s", source)}
	}
}

func TestR2_S15_00_ControlErroringAssetDoesNotCrashImg(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		tcx := tc.Cx()
		src := gpui.ImageSourceFromFunc(func(window *gpui.Window, cx *gpui.App) (*gpui.RenderImage, bool, error) {
			r, ok := window.UseAsset[errResultAsset]("img", cx)
			return r.Value, ok, r.Err
		})
		view, cx := tcx.AddWindowView(func(*gpui.Window, *gpui.Context[r2ImgView]) *r2ImgView { return &r2ImgView{src: src} })
		defer view.Release()
		cx.RunUntilParked()
		t.Logf("control: the same view with a load that returns an error renders without panicking")
	})
}

func TestR2_S15_01_PanickingAssetLoadsAsSuccess(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *gpui.App) { cx.FetchAsset[panicResultAsset]("x") })
		cx.RunUntilParked()
		type got struct {
			value  *gpui.RenderImage
			err    error
			loaded bool
		}
		g := cx.UpdateR(func(cx *gpui.App) got {
			r, ok := cx.FetchAsset[panicResultAsset]("x")
			return got{r.Value, r.Err, ok}
		})
		t.Logf("after a panic inside the asset's background load: loaded=%v value=%v err=%v", g.loaded, g.value, g.err)
		if g.loaded && g.err == nil && g.value == nil {
			t.Errorf("SWALLOWED PANIC: the asset cache reports the load as finished with a nil value and a nil error (newObservedCachedLoad drops the task's PanicError); img() treats (nil, loaded, nil) as a decoded image")
		}
	})
}

type r2ImgView struct{ src gpui.ImageSource }

func (v *r2ImgView) Render(window *gpui.Window, cx *gpui.Context[r2ImgView]) gpui.IntoElement {
	return gpui.NewImg(v.src).Size(gpui.Px(50))
}

func TestR2_S15_02_PanicReachesUIThreadViaImg(t *testing.T) {
	o, parent := bombchild.Run(t, childLimit, 30*time.Second, "", func(string) (string, error) {
		gpui.RunTest(t, func(tc *gpui.TestRun) {
			tcx := tc.Cx()
			src := gpui.ImageSourceFromFunc(func(window *gpui.Window, cx *gpui.App) (*gpui.RenderImage, bool, error) {
				r, ok := window.UseAsset[panicResultAsset]("img", cx)
				return r.Value, ok, r.Err
			})
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("UIPANIC %v\n", r)
					os.Exit(0)
				}
			}()
			view, cx := tcx.AddWindowView(func(*gpui.Window, *gpui.Context[r2ImgView]) *r2ImgView { return &r2ImgView{src: src} })
			cx.RunUntilParked()
			view.Release()
		})
		return "rendered without a UI-thread panic", nil
	})
	if parent {
		t.Logf("%s", o)
		for _, line := range strings.Split(o.Output, "\n") {
			if strings.HasPrefix(line, "UIPANIC ") {
				t.Errorf("CRASH: a background asset-load panic, recovered by SEC-037's wrapper, comes back as (nil image, loaded, nil error) and img() dereferences it on the UI thread: %s", strings.TrimPrefix(line, "UIPANIC "))
			}
		}
	}
}
