//go:build linux

package linux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	scenarioEnv            = "GPUI_LINUX_DISPLAY_MODE_SCENARIO"
	missingWaylandDisplay  = "/gpui-test/wayland-display-does-not-exist"
	displayModeTestsPrefix = "TestPlatformDisplayModeTests_"
)

func TestPlatformDisplayModeTests_FailedWaylandAttachPreservesHeadlessAppAndAllowsRetry(t *testing.T) {
	runScenario(t, "FailedWaylandAttachPreservesHeadlessAppAndAllowsRetry", windowingModesAll, func(cx *gpui.AsyncApp) {
		entity := cx.New(func(cx *gpui.Context[int]) *int {
			value := 1
			return &value
		})
		defer entity.Release()
		releaseTask, waitForRelease := scheduler.Oneshot[struct{}]()
		queuedTaskRan := false
		queuedTask := cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
			if _, err := waitForRelease.Recv(cx); err != nil {
				panic("queued task release sender remains alive")
			}
			queuedTaskRan = true
			return struct{}{}, nil
		})

		var firstAttach *gpui.Task[struct{}]
		cx.Update(func(cx *gpui.App) {
			firstAttach = cx.RequestWindowing(gpui.WindowingRequestWindowed{Environment: gpui.GraphicalEnvironmentDetect()})
		})
		_, err := firstAttach.Await(cx)
		assertErrorContains(err, "failed to connect to Wayland compositor")
		entity.Update(cx, func(value *int, _ *gpui.Context[int]) { *value++ })
		if err := releaseTask.Send(struct{}{}); err != nil {
			panic("queued task remains attached to the executor")
		}
		_, _ = queuedTask.Await(cx)
		if !queuedTaskRan {
			panic("queued task did not run")
		}

		var secondAttach *gpui.Task[struct{}]
		cx.Update(func(cx *gpui.App) {
			secondAttach = cx.RequestWindowing(gpui.WindowingRequestWindowed{Environment: gpui.GraphicalEnvironmentDetect()})
		})
		_, err = secondAttach.Await(cx)
		assertErrorContains(err, "failed to connect to Wayland compositor")
		entity.Update(cx, func(value *int, _ *gpui.Context[int]) { *value++ })
		if got := entity.ReadWith(cx, func(value *int, _ *gpui.App) int { return *value }); got != 3 {
			panic(fmt.Sprintf("value = %d, want 3", got))
		}
	})
}

func TestPlatformDisplayModeTests_HeadlessWindowsBlockSwitchingToADisplayServer(t *testing.T) {
	t.Skip("blocked:M0-window-paint: needs App.OpenWindow, WindowHandle.Update and Window.RemoveWindow")
}

func TestPlatformDisplayModeTests_DisplayModesLimitSwitching(t *testing.T) {
	runScenario(t, "DisplayModesLimitSwitching", gpui.WindowingModesHeadless, func(cx *gpui.AsyncApp) {
		var attach *gpui.Task[struct{}]
		cx.Update(func(cx *gpui.App) {
			attach = cx.RequestWindowing(gpui.WindowingRequestWindowed{Environment: gpui.GraphicalEnvironmentDetect()})
		})
		_, err := attach.Await(cx)
		assertErrorContains(err, "names no display server that")
		if got := cx.UpdateR(func(cx *gpui.App) string { return cx.CompositorName() }); got != "headless" {
			panic(fmt.Sprintf("compositor name = %q, want headless", got))
		}
		if cx.UpdateR(func(cx *gpui.App) bool { _, ok := cx.GraphicalEnvironment(); return ok }) {
			panic("graphical environment is set while headless")
		}

		var detach *gpui.Task[struct{}]
		cx.Update(func(cx *gpui.App) { detach = cx.RequestWindowing(gpui.WindowingRequestHeadless{}) })
		_, err = detach.Await(cx)
		assertErrorContains(err, "already headless")
	})
}

func runScenario(t *testing.T, name string, modes gpui.WindowingModes, scenario func(cx *gpui.AsyncApp)) {
	t.Helper()
	if os.Getenv(scenarioEnv) == name {
		platform := newLinuxPlatform(modes)
		defer platform.Close()
		platform.SetInitialWindowing(gpui.WindowingRequestHeadless{})
		completed := false
		gpui.ApplicationWithPlatform(platform).
			WithQuitMode(gpui.QuitModeExplicit).
			Run(func(cx *gpui.App) {
				cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
					defer cx.Update(func(cx *gpui.App) { cx.Quit() })
					scenario(cx)
					completed = true
					return struct{}{}, nil
				}).Detach()
			})
		if !completed {
			t.Fatal("scenario did not complete")
		}
		return
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("current test executable: %v", err)
	}
	testName := displayModeTestsPrefix + name
	command := exec.Command(executable, "-test.run", "^"+testName+"$", "-test.v", "-test.count=1")
	command.Env = append(filteredEnv("DISPLAY", "WAYLAND_DISPLAY", scenarioEnv), scenarioEnv+"="+name, "WAYLAND_DISPLAY="+missingWaylandDisplay)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "--- PASS: "+testName) {
		t.Fatalf("scenario failed: %v\noutput:\n%s", err, output)
	}
}

func filteredEnv(names ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		keep := true
		for _, removed := range names {
			if name == removed {
				keep = false
			}
		}
		if keep {
			env = append(env, kv)
		}
	}
	return env
}

func assertErrorContains(err error, expected string) {
	if err == nil {
		panic("the transition fails")
	}
	if !strings.Contains(err.Error(), expected) {
		panic(fmt.Sprintf("expected an error containing %q, got: %v", expected, err))
	}
}
