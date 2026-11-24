//go:build darwin && !ios

package platform

import (
	"testing"
	"time"

	"github.com/megascan/gpui-go"
)

func TestGpuiPlatform_ForegroundTasksRunWithRunUntilParked(t *testing.T) {
	t.Skip("Requires macOS main thread")
	cx := gpui.NewVisualTestAppContext(CurrentPlatform(false))

	taskRan := false

	cx.Update(func(cx *gpui.App) {
		cx.Spawn(func(_ *gpui.AsyncApp) (struct{}, error) {
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	if taskRan {
		t.Fatal("the task ran before run_until_parked")
	}

	cx.RunUntilParked()

	if !taskRan {
		t.Fatal("the task did not run")
	}
}

func TestGpuiPlatform_AdvanceClockTriggersDelayedTasks(t *testing.T) {
	t.Skip("Requires macOS main thread")
	cx := gpui.NewVisualTestAppContext(CurrentPlatform(false))

	taskRan := false

	executor := cx.Executor()
	cx.Update(func(cx *gpui.App) {
		cx.Spawn(func(cx *gpui.AsyncApp) (struct{}, error) {
			_ = executor.Timer(500 * time.Millisecond).Wait(cx)
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	cx.RunUntilParked()
	if taskRan {
		t.Fatal("the task ran before the timer fired")
	}

	cx.AdvanceClock(600 * time.Millisecond)

	if !taskRan {
		t.Fatal("the task did not run")
	}
}

func TestGpuiPlatform_WindowSpawnUsesTestDispatcher(t *testing.T) {
	t.Skip("Requires macOS main thread - window creation fails on test threads")
	cx := gpui.NewVisualTestAppContext(CurrentPlatform(false))

	taskRan := false

	window, err := cx.OpenOffscreenWindowDefault(func(_ *gpui.Window, cx *gpui.App) gpui.Entity[gpui.Empty] {
		return cx.New(func(*gpui.Context[gpui.Empty]) *gpui.Empty { return &gpui.Empty{} })
	})
	if err != nil {
		t.Fatalf("Failed to open window: %v", err)
	}

	_ = cx.UpdateWindow(window.AnyWindowHandle, func(_ gpui.AnyView, window *gpui.Window, cx *gpui.App) {
		window.Spawn(cx, func(*gpui.AsyncWindowContext) (struct{}, error) {
			taskRan = true
			return struct{}{}, nil
		}).Detach()
	})

	if taskRan {
		t.Fatal("the task ran before run_until_parked")
	}

	cx.RunUntilParked()

	if !taskRan {
		t.Fatal("the task did not run")
	}
}
