//go:build darwin && !ios

package macos

import (
	"os"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
)

func skipUnlessTestDisplay(t *testing.T) {
	t.Helper()
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" {
		t.Skip("Requires macOS main thread")
	}
}

func TestGpuiPlatform_ForegroundTasksRunWithRunUntilParked(t *testing.T) {
	skipUnlessTestDisplay(t)
	cx := gpui.NewVisualTestAppContext(NewMacPlatform(false))

	taskRan := false
	cx.Update(func(cx *gpui.App) {
		cx.Spawn(func(_ *gpui.AsyncApp) (struct{}, error) {
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	if taskRan {
		t.Fatal("assertion failed: !*task_ran.borrow()")
	}
	cx.RunUntilParked()
	if !taskRan {
		t.Fatal("assertion failed: *task_ran.borrow()")
	}
}

func TestGpuiPlatform_AdvanceClockTriggersDelayedTasks(t *testing.T) {
	skipUnlessTestDisplay(t)
	cx := gpui.NewVisualTestAppContext(NewMacPlatform(false))

	taskRan := false
	executor := cx.Executor()
	cx.Update(func(cx *gpui.App) {
		cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
			if _, err := executor.Timer(500 * time.Millisecond).Await(cx); err != nil {
				return struct{}{}, err
			}
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	cx.RunUntilParked()
	if taskRan {
		t.Fatal("assertion failed: !*task_ran.borrow()")
	}
	cx.AdvanceClock(600 * time.Millisecond)
	if !taskRan {
		t.Fatal("assertion failed: *task_ran.borrow()")
	}
}

func TestGpuiPlatform_WindowSpawnUsesTestDispatcher(t *testing.T) {
	skipUnlessTestDisplay(t)
	cx := gpui.NewVisualTestAppContext(NewMacPlatform(false))

	taskRan := false
	window, err := cx.OpenOffscreenWindowDefault(func(_ *gpui.Window, cx *gpui.App) gpui.Entity[gpui.Empty] {
		return cx.New(func(_ *gpui.Context[gpui.Empty]) *gpui.Empty { return &gpui.Empty{} })
	})
	if err != nil {
		t.Fatalf("Failed to open window: %v", err)
	}

	_ = cx.UpdateWindow(window.AnyWindowHandle, func(_ gpui.AnyView, window *gpui.Window, cx *gpui.App) {
		window.Spawn(cx, func(_ *gpui.AsyncWindowContext) (struct{}, error) {
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	if taskRan {
		t.Fatal("assertion failed: !*task_ran.borrow()")
	}
	cx.RunUntilParked()
	if !taskRan {
		t.Fatal("assertion failed: *task_ran.borrow()")
	}
}
