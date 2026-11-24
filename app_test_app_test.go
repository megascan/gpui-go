package gpui

import (
	"fmt"
	"testing"
)

type testAppCounter struct {
	count       int
	focusHandle FocusHandle
}

func newTestAppCounter(_ *Window, cx *Context[testAppCounter]) *testAppCounter {
	return &testAppCounter{count: 0, focusHandle: cx.FocusHandle()}
}

func (c *testAppCounter) increment(_ *Context[testAppCounter]) {
	c.count++
}

func (c *testAppCounter) FocusHandle(_ *App) FocusHandle {
	return c.focusHandle.Clone()
}

func (c *testAppCounter) Render(_ *Window, _ *Context[testAppCounter]) IntoElement {
	return NewDiv().ChildText(fmt.Sprintf("Count: %d", c.count))
}

func (c *testAppCounter) Drop(*App) {
	c.focusHandle.Release()
}

func testAppReleaseWindows(app *TestApp) {
	for _, window := range app.Windows() {
		for _, step := range []func(*Window){divTestClearTabStops, divTestTakeRoot} {
			app.Update(func(cx *App) {
				_, _ = AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, _ *App) struct{} {
					step(window)
					return struct{}{}
				})
			})
		}
	}
}

func TestTestApp_BasicUsage(t *testing.T) {
	app := NewTestApp()
	defer app.Close()

	window := app.OpenWindow(newTestAppCounter)

	window.Update(func(counter *testAppCounter, _ *Window, cx *Context[testAppCounter]) {
		counter.increment(cx)
	})

	window.Read(func(counter *testAppCounter, _ *App) {
		if counter.count != 1 {
			t.Fatalf("count = %d, want 1", counter.count)
		}
	})

	testAppReleaseWindows(app)
	app.Update(func(cx *App) { cx.Shutdown() })
}

func TestTestApp_SimulateScaleFactorChange(t *testing.T) {
	app := NewTestApp()
	defer app.Close()
	window := app.OpenWindow(func(_ *Window, _ *Context[EmptyView]) *EmptyView { return &EmptyView{} })
	viewportSize := window.UpdateR(func(_ *EmptyView, window *Window, _ *Context[EmptyView]) Size[Pixels] {
		if got := window.ScaleFactor(); got != 2.0 {
			t.Fatalf("scale factor = %v, want 2", got)
		}
		return window.ViewportSize()
	})

	window.SimulateScaleFactorChange(1.0)

	window.Update(func(_ *EmptyView, window *Window, _ *Context[EmptyView]) {
		if got := window.ScaleFactor(); got != 1.0 {
			t.Fatalf("scale factor = %v, want 1", got)
		}
		if got := window.ViewportSize(); got != viewportSize {
			t.Fatalf("viewport size = %v, want %v", got, viewportSize)
		}
	})

	testAppReleaseWindows(app)
	app.Update(func(cx *App) { cx.Shutdown() })
}

func TestTestApp_EntityCreation(t *testing.T) {
	app := NewTestApp()
	defer app.Close()

	entity := app.NewEntity(func(cx *Context[testAppCounter]) *testAppCounter {
		return &testAppCounter{count: 42, focusHandle: cx.FocusHandle()}
	})
	defer entity.Release()

	app.ReadEntity(entity, func(counter *testAppCounter, _ *App) {
		if counter.count != 42 {
			t.Fatalf("count = %d, want 42", counter.count)
		}
	})

	app.UpdateEntity(entity, func(counter *testAppCounter, _ *Context[testAppCounter]) {
		counter.count++
	})

	app.ReadEntity(entity, func(counter *testAppCounter, _ *App) {
		if counter.count != 43 {
			t.Fatalf("count = %d, want 43", counter.count)
		}
	})
}

type testAppGlobal struct {
	value string
}

func TestTestApp_Globals(t *testing.T) {
	app := NewTestApp()
	defer app.Close()

	if app.HasGlobal[testAppGlobal]() {
		t.Fatal("global exists before it was set")
	}

	app.SetGlobal(testAppGlobal{value: "hello"})

	if !app.HasGlobal[testAppGlobal]() {
		t.Fatal("global missing after SetGlobal")
	}

	app.ReadGlobal(func(global *testAppGlobal, _ *App) struct{} {
		if global.value != "hello" {
			t.Fatalf("global = %q, want %q", global.value, "hello")
		}
		return struct{}{}
	})

	app.UpdateGlobal(func(global *testAppGlobal, _ *App) {
		global.value = "world"
	})

	app.ReadGlobal(func(global *testAppGlobal, _ *App) struct{} {
		if global.value != "world" {
			t.Fatalf("global = %q, want %q", global.value, "world")
		}
		return struct{}{}
	})
}
