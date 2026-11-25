package gpui

import (
	"slices"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

var (
	_ AppContext    = (*TestAppContext)(nil)
	_ AppContext    = (*HeadlessAppContext)(nil)
	_ AppContext    = (*VisualTestAppContext)(nil)
	_ VisualContext = (*VisualTestContext)(nil)
)

type lifecycleObserved struct{}

func TestTestContext_MobileLifecycleSubscriptions(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var phases []AppLifecyclePhase
		subscription := cx.UpdateR(func(cx *App) Subscription {
			return cx.OnAppLifecycle(func(phase AppLifecyclePhase, cx *App) {
				phases = append(phases, phase)
				cx.Defer(func(cx *App) {
					cx.SetGlobal(lifecycleObserved{})
				})
			})
		})

		callback := cx.testPlatform.appLifecycleCallback
		cx.testPlatform.appLifecycleCallback = nil
		if callback == nil {
			t.Fatal("App registers the native lifecycle callback")
		}
		for _, phase := range []AppLifecyclePhase{
			AppLifecyclePhaseForeground,
			AppLifecyclePhaseActive,
			AppLifecyclePhaseInactive,
			AppLifecyclePhaseBackground,
			AppLifecyclePhaseDisconnected,
		} {
			callback(phase)
		}
		if len(phases) != 5 {
			t.Fatalf("len(phases) = %d, want 5", len(phases))
		}
		if phases[len(phases)-1] != AppLifecyclePhaseDisconnected {
			t.Fatalf("last phase = %v, want Disconnected", phases[len(phases)-1])
		}
		cx.Update(func(cx *App) {
			if !cx.HasGlobal[lifecycleObserved]() {
				t.Fatal("lifecycle observer did not run its deferred update")
			}
		})
		subscription.Unsubscribe()
		callback(AppLifecyclePhaseForeground)
		if len(phases) != 5 {
			t.Fatalf("len(phases) = %d after unsubscribe, want 5", len(phases))
		}
	})
}

func TestTestContext_MemoryWarningSubscriptions(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		warnings := 0
		subscription := cx.UpdateR(func(cx *App) Subscription {
			return cx.OnMemoryWarning(func(*App) { warnings++ })
		})
		callback := cx.testPlatform.memoryWarningCallback
		cx.testPlatform.memoryWarningCallback = nil
		if callback == nil {
			t.Fatal("App registers the native memory warning callback")
		}
		callback()
		callback()
		if warnings != 2 {
			t.Fatalf("warnings = %d, want 2", warnings)
		}
		subscription.Unsubscribe()
		callback()
		if warnings != 2 {
			t.Fatalf("warnings = %d after unsubscribe, want 2", warnings)
		}
	})
}

func notificationsEqual(a, b []SystemNotification) bool {
	return slices.EqualFunc(a, b, func(x, y SystemNotification) bool {
		return x.Tag == y.Tag && x.Title == y.Title && x.Body == y.Body && slices.Equal(x.Actions, y.Actions)
	})
}

func TestTestContext_SystemNotificationsRequireIdentityAndReplaceMatchingTags(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			cx.Update(func(cx *App) {
				cx.ShowSystemNotification(SystemNotification{
					Tag:   "thread-1",
					Title: "Task started",
					Body:  "Running tests",
				})
			})
			if len(cx.ShownSystemNotifications()) != 0 {
				t.Fatal("notification shown without an app identity")
			}
			if len(cx.DeliveredSystemNotifications()) != 0 {
				t.Fatal("notification delivered without an app identity")
			}

			cx.Update(func(cx *App) {
				cx.SetAppIdentity("com.example.tasks", "Tasks")
				cx.ShowSystemNotification(SystemNotification{
					Tag:   "thread-1",
					Title: "Task started",
					Body:  "Running tests",
				})
				cx.ShowSystemNotification(SystemNotification{
					Tag:   "thread-1",
					Title: "Task finished",
					Body:  "All tests passed",
					Actions: []SystemNotificationAction{{
						ID:    "open",
						Label: "Open",
					}},
				})
			})

			identifier, name, ok := cx.AppIdentity()
			if !ok || identifier != "com.example.tasks" || name != "Tasks" {
				t.Fatalf("AppIdentity() = (%q, %q, %v)", identifier, name, ok)
			}
			if n := len(cx.ShownSystemNotifications()); n != 2 {
				t.Fatalf("len(ShownSystemNotifications()) = %d, want 2", n)
			}
			want := []SystemNotification{{
				Tag:   "thread-1",
				Title: "Task finished",
				Body:  "All tests passed",
				Actions: []SystemNotificationAction{{
					ID:    "open",
					Label: "Open",
				}},
			}}
			if got := cx.DeliveredSystemNotifications(); !notificationsEqual(got, want) {
				t.Fatalf("DeliveredSystemNotifications() = %v, want %v", got, want)
			}

			cx.Update(func(cx *App) { cx.DismissSystemNotification("thread-1").Detach() })
			if len(cx.DeliveredSystemNotifications()) != 0 {
				t.Fatal("dismissed notification is still delivered")
			}
			if got := cx.DismissedSystemNotifications(); !slices.Equal(got, []SharedString{"thread-1"}) {
				t.Fatalf("DismissedSystemNotifications() = %v", got)
			}
		})
	})
}

func TestTestContext_SystemNotificationBodyAndActionResponses(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			var responses []SystemNotificationResponse
			cx.Update(func(cx *App) {
				cx.OnSystemNotificationResponse(func(response SystemNotificationResponse, _ *App) {
					responses = append(responses, response)
				})
			})

			cx.SimulateSystemNotificationResponse(SystemNotificationResponse{
				Tag:      "thread-1",
				ActionID: None[SharedString](),
			})
			cx.SimulateSystemNotificationResponse(SystemNotificationResponse{
				Tag:      "thread-1",
				ActionID: Some[SharedString]("default"),
			})

			want := []SystemNotificationResponse{
				{Tag: "thread-1", ActionID: None[SharedString]()},
				{Tag: "thread-1", ActionID: Some[SharedString]("default")},
			}
			if !slices.Equal(responses, want) {
				t.Fatalf("responses = %v, want %v", responses, want)
			}
		})
	})
}

func TestTestContext_SystemNotificationResponseHandlerCanBeReplaced(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			var firstResponses, secondResponses []SystemNotificationResponse
			cx.Update(func(cx *App) {
				cx.OnSystemNotificationResponse(func(response SystemNotificationResponse, _ *App) {
					firstResponses = append(firstResponses, response)
				})
				cx.OnSystemNotificationResponse(func(response SystemNotificationResponse, _ *App) {
					secondResponses = append(secondResponses, response)
				})
			})

			response := SystemNotificationResponse{
				Tag:      "thread-1",
				ActionID: None[SharedString](),
			}
			cx.SimulateSystemNotificationResponse(response)

			if len(firstResponses) != 0 {
				t.Fatalf("first handler received %v", firstResponses)
			}
			if !slices.Equal(secondResponses, []SystemNotificationResponse{response}) {
				t.Fatalf("second handler received %v", secondResponses)
			}
		})
	})
}

func TestTestContext_SystemNotificationResponseHandlerCanReenterApp(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			cx.Update(func(cx *App) {
				cx.SetAppIdentity("com.example.tasks", "Tasks")
				cx.ShowSystemNotification(SystemNotification{
					Tag:   "thread-1",
					Title: "Task finished",
					Body:  "All tests passed",
				})
				cx.OnSystemNotificationResponse(func(response SystemNotificationResponse, cx *App) {
					cx.DismissSystemNotification(string(response.Tag)).Detach()
				})
			})

			cx.SimulateSystemNotificationResponse(SystemNotificationResponse{
				Tag:      "thread-1",
				ActionID: None[SharedString](),
			})

			if len(cx.DeliveredSystemNotifications()) != 0 {
				t.Fatal("notification is still delivered")
			}
			if got := cx.DismissedSystemNotifications(); !slices.Equal(got, []SharedString{"thread-1"}) {
				t.Fatalf("DismissedSystemNotifications() = %v", got)
			}
		})
	})
}

func TestTestContext_SimulatePathPromptResponse(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			if cx.DidPromptForPaths() {
				t.Fatal("a paths prompt is pending before any was opened")
			}

			receiver := cx.UpdateR(func(cx *App) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]] {
				return cx.PromptForPaths(PathPromptOptions{
					Files:       false,
					Directories: true,
					Multiple:    true,
					Prompt:      None[SharedString](),
				})
			})
			if !cx.DidPromptForPaths() {
				t.Fatal("no paths prompt is pending")
			}

			selected := []string{"/a", "/b"}
			cx.SimulatePathPromptResponse(func(options *PathPromptOptions) Option[[]string] {
				if !options.Multiple {
					t.Fatal("options.Multiple is false")
				}
				return Some(slices.Clone(selected))
			})
			if cx.DidPromptForPaths() {
				t.Fatal("paths prompt still pending after the response")
			}

			result, err := receiver.Recv(s)
			if err != nil {
				t.Fatalf("receiver: %v", err)
			}
			if result.Err != nil {
				t.Fatalf("prompt failed: %v", result.Err)
			}
			paths, ok := result.Value.Get()
			if !ok || !slices.Equal(paths, selected) {
				t.Fatalf("response = %v, want Some(%v)", result.Value, selected)
			}
		})
	})
}

func TestTestContext_SimulatePathPromptCancellation(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			receiver := cx.UpdateR(func(cx *App) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]] {
				return cx.PromptForPaths(PathPromptOptions{
					Files:       true,
					Directories: false,
					Multiple:    false,
					Prompt:      None[SharedString](),
				})
			})

			cx.SimulatePathPromptResponse(func(*PathPromptOptions) Option[[]string] { return None[[]string]() })

			result, err := receiver.Recv(s)
			if err != nil {
				t.Fatalf("receiver: %v", err)
			}
			if result.Err != nil {
				t.Fatalf("prompt failed: %v", result.Err)
			}
			if result.Value.IsSome() {
				t.Fatalf("response = %v, want None", result.Value)
			}
		})
	})
}
