package gpui

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

type secPanicAsset struct{}

func (secPanicAsset) Load(source string, _ *App) func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
	return func(*scheduler.Scope) scheduler.Result[*RenderImage] {
		var frames []int
		_ = frames[len(source)+7]
		return scheduler.Result[*RenderImage]{}
	}
}

type secPanicPlainAsset struct{}

func (secPanicPlainAsset) Load(source string, _ *App) func(s *scheduler.Scope) string {
	return func(*scheduler.Scope) string {
		panic("plain load " + source)
	}
}

func TestSecurity_SEC037_R2_2_PanickingLoadStoresError(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) {
			cx.FetchAsset[secPanicAsset]("x")
			cx.FetchAsset[secPanicPlainAsset]("y")
		})
		cx.RunUntilParked()
		cx.Update(func(cx *App) {
			result, ok := cx.FetchAsset[secPanicAsset]("x")
			if !ok || result.Err == nil || result.Value != nil {
				t.Errorf("panicking load = (%v, %v, %v), want a loaded error", result.Value, ok, result.Err)
			}
			plain, ok := cx.FetchAsset[secPanicPlainAsset]("y")
			if !ok || plain != "" {
				t.Errorf("panicking plain load = (%q, %v), want a loaded zero value", plain, ok)
			}
		})
	})
}

var secGalleryLoads atomic.Int64

type secGalleryAsset struct{}

func (secGalleryAsset) Load(source int, _ *App) func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
	if secGalleryLoads.Add(1) > 3*(maxAssetCacheEntries+64) {
		panic("asset cache thrashes: the visible set is evicted and reloaded")
	}
	return func(*scheduler.Scope) scheduler.Result[*RenderImage] {
		return scheduler.Result[*RenderImage]{Err: fmt.Errorf("missing %d", source)}
	}
}

type secGalleryView struct{ n int }

func (v *secGalleryView) Render(window *Window, cx *Context[secGalleryView]) IntoElement {
	for i := range v.n {
		window.UseAsset[secGalleryAsset](i, cx.App)
	}
	return NewDiv()
}

func TestSecurity_SEC036_R2_3_OverCapacityVisibleSetSettles(t *testing.T) {
	secGalleryLoads.Store(0)
	const n = maxAssetCacheEntries + 64
	RunTest(t, func(tc *TestRun) {
		app := tc.Cx()
		view, cx := app.AddWindowView(func(*Window, *Context[secGalleryView]) *secGalleryView { return &secGalleryView{n: n} })
		cx.RunUntilParked()
		if loads := secGalleryLoads.Load(); loads != n {
			t.Errorf("loads = %d, want %d: visible entries were evicted and reloaded", loads, n)
		}
		app.Update(func(cx *App) {
			if got := len(cx.loadingAssets.entries); got != n {
				t.Errorf("cache holds %d entries, want the whole visible set %d", got, n)
			}
		})
		view.Release()
	})
}

func TestSecurity_SEC036_I1_TrimSkipsKeptBackEntry(t *testing.T) {
	c := newAssetCache()
	released := 0
	first := &assetSlot{key: loadingAssetKey{hash: 0}, load: secReleaser{released: &released}}
	c.insert(first, nil)
	second := &assetSlot{key: loadingAssetKey{hash: 1}, load: secReleaser{released: &released}}
	c.insert(second, nil)
	c.loaded(second.key, maxAssetCacheBytes/2, nil)
	c.order.MoveToBack(c.entries[second.key])
	c.order.MoveToBack(c.entries[first.key])
	c.loaded(first.key, maxAssetCacheBytes, nil)
	if released != 1 || c.has(second.key) || !c.has(first.key) || c.bytes > maxAssetCacheBytes {
		t.Fatalf("released=%d has(second)=%v has(first)=%v bytes=%d", released, c.has(second.key), c.has(first.key), c.bytes)
	}
}

func TestSecurity_SEC036_R2_3_EvictionCancelsBackgroundLoad(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		sender, receiver := scheduler.Oneshot[testLoadResult]()
		source := newTestAssetSource(9, receiver)
		cx.Update(func(cx *App) {
			cx.FetchAsset[testAsset](source)
			cx.RemoveAsset[testAsset](source)
		})
		cx.RunUntilParked()
		_ = sender.Send(okLoad(1))
		for cx.bgExecutor.Tick() {
		}
		cx.RunUntilParked()
		if n := source.completionCount.Load(); n != 0 {
			t.Fatalf("background load ran to completion %d times after its entry was removed", n)
		}
	})
}

type secScrollAsset struct{}

func (secScrollAsset) Load(source int, _ *App) func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
	return func(*scheduler.Scope) scheduler.Result[*RenderImage] {
		return scheduler.Result[*RenderImage]{Err: fmt.Errorf("missing %d", source)}
	}
}

type secScrollView struct{ offset int }

func (v *secScrollView) Render(window *Window, cx *Context[secScrollView]) IntoElement {
	for i := range maxAssetCacheEntries {
		window.UseAsset[secScrollAsset](v.offset+i, cx.App)
	}
	return NewDiv()
}

func TestSecurity_SEC036_R2_3_ScrolledOutEntriesAreEvicted(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		app := tc.Cx()
		view, cx := app.AddWindowView(func(*Window, *Context[secScrollView]) *secScrollView { return &secScrollView{} })
		cx.RunUntilParked()
		app.Update(func(cx *App) {
			view.Update(cx, func(v *secScrollView, cx *Context[secScrollView]) {
				v.offset = maxAssetCacheEntries
				cx.Notify()
			})
		})
		cx.RunUntilParked()
		app.Update(func(cx *App) {
			if got := len(cx.loadingAssets.entries); got > maxAssetCacheEntries {
				t.Errorf("cache holds %d entries after scrolling, cap %d", got, maxAssetCacheEntries)
			}
			if cx.HasAsset[secScrollAsset](0) {
				t.Error("an entry no longer visible in any window was not evicted")
			}
			if !cx.HasAsset[secScrollAsset](2*maxAssetCacheEntries - 1) {
				t.Error("a visible entry was evicted")
			}
		})
		view.Release()
	})
}
