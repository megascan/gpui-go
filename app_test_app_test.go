package gpui

import "testing"

func TestTestApp_BasicUsage(t *testing.T) {
	t.Skip("blocked:M0-blocked-tests: needs windows (App.OpenWindow), FocusHandle and the div element")
}

func TestTestApp_SimulateScaleFactorChange(t *testing.T) {
	t.Skip("blocked:M0-blocked-tests: needs windows (App.OpenWindow) and EmptyView")
}

func TestTestApp_EntityCreation(t *testing.T) {
	t.Skip("blocked:M0-blocked-tests: Counter holds a FocusHandle (App.FocusHandle, M0-window-core)")
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
