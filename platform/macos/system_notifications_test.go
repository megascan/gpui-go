//go:build darwin && !ios

package macos

import (
	"reflect"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

func TestSystemNotifications_NotificationCommandsDoNotWaitForTheService(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		sender, receiver := scheduler.Chan[notificationCommandEntry](0)
		state := newSystemNotificationState()
		state.initialized = true
		state.commands = sender
		notification := gpui.SystemNotification{
			Tag:   "thread",
			Title: "Finished",
			Body:  "Ready for input",
		}

		foregroundExecutor := cx.ForegroundExecutor()
		backgroundExecutor := cx.Executor()
		tasks := []*gpui.Task[struct{}]{
			state.send(foregroundExecutor, backgroundExecutor, notificationCommandShow{notification: notification}),
			state.send(foregroundExecutor, backgroundExecutor, notificationCommandDismiss{tag: "thread"}),
			state.send(foregroundExecutor, backgroundExecutor, notificationCommandShow{notification: notification}),
		}
		for _, task := range tasks {
			if task.IsReady() {
				t.Fatal("assertion failed: task.now_or_never().is_none()")
			}
		}

		var commands []notificationCommand
		var completions []*scheduler.OneshotSender[struct{}]
		for {
			entry, ok, _ := receiver.TryRecv()
			if !ok {
				break
			}
			commands = append(commands, entry.command)
			completions = append(completions, entry.completed)
		}
		want := []notificationCommand{
			notificationCommandShow{notification: notification},
			notificationCommandDismiss{tag: "thread"},
			notificationCommandShow{notification: notification},
		}
		if !reflect.DeepEqual(commands, want) {
			t.Fatalf("assertion `left == right` failed\n  left: %#v\n right: %#v", commands, want)
		}
		for _, completed := range completions {
			if err := completed.Send(struct{}{}); err != nil {
				panic("caller is waiting for completion: " + err.Error())
			}
		}
		tc.BlockTest(func(s *scheduler.Scope) {
			for _, task := range tasks {
				if _, err := task.Await(s); err != nil {
					t.Fatal(err)
				}
			}
		})
	})
}
