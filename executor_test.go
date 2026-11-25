package gpui

import (
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

func TestExecutor_SendForegroundExecutorSubmitsFromWorker(t *testing.T) {
	dispatcher := NewTestDispatcher(0)
	foregroundExecutor := NewForegroundExecutor(dispatcher.Clone())
	sendExecutor := foregroundExecutor.ToSend()
	submitted := make(chan *Task[int])
	go func() {
		submitted <- sendExecutor.Spawn(func(*scheduler.Scope) (int, error) {
			if !dispatcher.IsMainThread() {
				t.Error("foreground task must run on the main thread")
			}
			return 42, nil
		})
	}()
	task := <-submitted

	dispatcher.RunUntilParked()
	if !task.IsReady() {
		t.Fatal("task did not complete")
	}
	r := scheduler.BlockOn[scheduler.Result[int]](task)
	if r.Err != nil || r.Value != 42 {
		t.Fatalf("got %d, %v, want 42", r.Value, r.Err)
	}
}

func createExecutorTestApp() (*TestDispatcher, BackgroundExecutor, *AppCell) {
	dispatcher := NewTestDispatcher(0)
	arcDispatcher := dispatcher.Clone()
	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)

	platform := NewTestPlatform(backgroundExecutor, foregroundExecutor)
	httpClient := fakeHTTPClientWith404Response()

	app := newApp(platform, NoAssets{}, httpClient)
	return dispatcher, backgroundExecutor, app
}

func TestExecutor_SanityTestTasksRun(t *testing.T) {
	dispatcher, _, app := createExecutorTestApp()
	borrow := app.Borrow()
	foregroundExecutor := borrow.Get().foregroundExecutor
	borrow.Release()

	taskRan := false

	foregroundExecutor.Spawn(func(*scheduler.Scope) (struct{}, error) {
		taskRan = true
		return struct{}{}, nil
	}).Detach()

	dispatcher.RunUntilParked()

	if !taskRan {
		t.Fatal("Task should run normally when app is alive")
	}
}

func TestExecutor_ScopedWaitsForEveryFuture(t *testing.T) {
	dispatcher := NewTestDispatcher(0)
	background := NewBackgroundExecutor(dispatcher)
	foreground := NewForegroundExecutor(dispatcher)
	var results [4]int
	task := foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		background.Scoped(s, func(scope *Scope) {
			for i := range results {
				scope.Spawn(func(*scheduler.Scope) {
					results[i] = i + 1
				})
			}
		})
		return struct{}{}, nil
	})
	dispatcher.RunUntilParked()
	if !task.IsReady() {
		t.Fatal("scoped task did not complete")
	}
	task.Detach()
	if results != [4]int{1, 2, 3, 4} {
		t.Fatalf("results %v", results)
	}
}

func TestExecutor_ScopedCancelClosesUnstartedFutures(t *testing.T) {
	dispatcher := NewTestDispatcher(0)
	background := NewBackgroundExecutor(dispatcher)
	foreground := NewForegroundExecutor(dispatcher)
	ran := 0
	task := foreground.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		background.Scoped(s, func(scope *Scope) {
			for range 3 {
				scope.Spawn(func(*scheduler.Scope) { ran++ })
			}
		})
		return struct{}{}, nil
	})
	dispatcher.Tick(false)
	task.Cancel()
	dispatcher.RunUntilParked()
	if ran > 3 {
		t.Fatalf("ran %d", ran)
	}
}

func TestExecutor_TimerZeroIsReady(t *testing.T) {
	background := NewBackgroundExecutor(NewTestDispatcher(0))
	if !background.Timer(0).IsReady() {
		t.Fatal("zero timer must be ready")
	}
}
