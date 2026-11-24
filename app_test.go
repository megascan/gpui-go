package gpui

import (
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

func newBareApp() *AppCell {
	cell := &AppCell{borrowOrigin: None[appBorrowOrigin]()}
	cell.app = App{
		entities:                   newEntityMap(),
		windows:                    newWindowSlotMap(),
		windowHandles:              map[WindowID]AnyWindowHandle{},
		focusHandles:               defaultFocusMap(),
		globalActionListeners:      map[reflect.Type][]func(action any, phase DispatchPhase, cx *App){},
		newEntityObservers:         newSubscriberSet[reflect.Type, newEntityListener](),
		observers:                  newSubscriberSet[EntityID, handler](),
		eventListeners:             newSubscriberSet[EntityID, eventListener](),
		releaseListeners:           newSubscriberSet[EntityID, releaseListener](),
		globalObservers:            newSubscriberSet[reflect.Type, handler](),
		globalsByType:              map[reflect.Type]any{},
		pendingNotifications:       collections.HashSet[EntityID]{},
		pendingGlobalNotifications: collections.TypeIDHashSet{},
		windowInvalidatorsByEntity: map[EntityID]map[WindowID]windowInvalidator{},
		trackedEntities:            map[WindowID]collections.HashSet[EntityID]{},
		currentWindowByEntity:      map[EntityID]WindowID{},
	}
	return cell
}

func catchPanic(f func()) (message string) {
	defer func() {
		if r := recover(); r != nil {
			message = fmt.Sprint(r)
		}
	}()
	f()
	return ""
}

func callerLine() int {
	return gpuiutil.CallerLocation(1).Line
}

func TestApp_AppBorrowDiagnostics(t *testing.T) {
	app := newBareApp()
	file := gpuiutil.CallerLocation(0).File

	firstLine := callerLine() + 1
	first := app.Borrow()
	second := app.Borrow()
	first.Release()
	if _, err := app.TryBorrowMut(); err == nil {
		t.Fatal("try_borrow_mut succeeded during a shared borrow")
	}
	attemptedLine := callerLine() + 1
	message := catchPanic(func() { app.BorrowMut().Release() })
	if message == "" {
		t.Fatal("conflicting borrow did not panic")
	}
	if want := fmt.Sprintf("mutable App borrow at %s:%d", file, attemptedLine); !strings.Contains(message, want) {
		t.Fatalf("message %q does not contain %q", message, want)
	}
	if want := fmt.Sprintf("shared borrow group began at %s:%d", file, firstLine); !strings.Contains(message, want) {
		t.Fatalf("message %q does not contain %q", message, want)
	}
	second.Release()

	originLine := callerLine() + 2
	message = catchPanic(func() {
		borrow, err := app.TryBorrowMut()
		if err != nil {
			t.Fatalf("initial borrow: %v", err)
		}
		defer borrow.Release()
		app.Borrow().Release()
	})
	if message == "" {
		t.Fatal("conflicting borrow did not panic")
	}
	if !strings.Contains(message, "attempted shared App borrow") {
		t.Fatalf("message %q does not mention the shared borrow", message)
	}
	if want := fmt.Sprintf("mutable borrow group began at %s:%d", file, originLine); !strings.Contains(message, want) {
		t.Fatalf("message %q does not contain %q", message, want)
	}

	nextLine := callerLine() + 1
	borrow := app.Borrow()
	defer borrow.Release()
	origin, ok := app.borrowOrigin.Get()
	if !ok {
		t.Fatal("new origin missing")
	}
	if origin.location.Line != nextLine {
		t.Fatalf("origin line = %d, want %d", origin.location.Line, nextLine)
	}
}

func TestApp_AppRefReleaseEndsBorrowGroup(t *testing.T) {
	app := newBareApp()
	shared := app.Borrow()
	shared.Release()
	shared.Release()
	mutable, err := app.TryBorrowMut()
	if err != nil {
		t.Fatalf("mutable borrow after shared Release: %v", err)
	}
	mutable.Release()
	mutable.Release()
	again, err := app.TryBorrowMut()
	if err != nil {
		t.Fatalf("mutable borrow after mutable Release: %v", err)
	}
	again.Release()
}

func TestApp_GpuiBorrowReleaseNotifies(t *testing.T) {
	cell := newBareApp()
	cx := &cell.app
	observations := 0

	var state Entity[bool]
	cx.update(func(cx *App) {
		state = cx.entities.insert(cx.entities.reserve[bool](), new(bool))
		observeInternal(cx, state, func(Entity[bool], *App) bool {
			observations++
			return true
		}).Detach()
	})

	cx.update(func(cx *App) {
		borrow := newGpuiBorrow(state, cx)
		*borrow.Get() = true
		borrow.Release()
		borrow.Release()
		if message := catchPanic(func() { borrow.Get() }); message == "" {
			t.Fatal("Get after Release did not panic")
		}
	})

	if observations != 1 {
		t.Fatalf("observations = %d, want 1", observations)
	}
	if !*cx.entities.read(state) {
		t.Fatal("write through GpuiBorrow was lost")
	}
	state.Release()
	cx.update(func(*App) {})
}

func TestApp_EffectsFlushInOrder(t *testing.T) {
	cell := newBareApp()
	cx := &cell.app
	var log []string

	type counter struct{ n int }
	cx.update(func(cx *App) {
		cx.SetGlobal(counter{})
		cx.ObserveGlobal[counter](func(cx *App) {
			log = append(log, fmt.Sprintf("global %d", cx.Global[counter]().n))
		}).Detach()
		cx.Defer(func(*App) { log = append(log, "defer") })
	})
	if len(log) != 1 || log[0] != "defer" {
		t.Fatalf("log = %v, want [defer]", log)
	}

	cx.update(func(cx *App) {
		UpdateGlobal(cx, func(c *counter, cx *App) { c.n++ })
		cx.GlobalMut[counter]().n++
		cx.Defer(func(*App) { log = append(log, "defer 2") })
	})
	want := []string{"defer", "global 2", "defer 2"}
	if fmt.Sprint(log) != fmt.Sprint(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}

	removed := cx.RemoveGlobal[counter]()
	if removed.n != 2 || cx.HasGlobal[counter]() {
		t.Fatalf("RemoveGlobal = %v, HasGlobal = %v", removed, cx.HasGlobal[counter]())
	}
	if message := catchPanic(func() { cx.Global[counter]() }); !strings.Contains(message, "no state of type") {
		t.Fatalf("Global panic = %q", message)
	}
}

func TestApp_DispatchGlobalActionPhases(t *testing.T) {
	cell := newBareApp()
	cx := &cell.app
	var phases []DispatchPhase
	actionType := reflect.TypeFor[*testNoopAction]()
	cx.globalActionListeners[actionType] = []func(action any, phase DispatchPhase, cx *App){
		func(_ any, phase DispatchPhase, _ *App) { phases = append(phases, phase) },
	}
	calls := 0
	cx.OnAction(func(*testNoopAction, *App) { calls++ })
	cx.dispatchGlobalAction(&testNoopAction{})
	if calls != 1 {
		t.Fatalf("bubble listener calls = %d, want 1", calls)
	}
	if len(phases) != 2 || phases[0] != DispatchPhaseCapture || phases[1] != DispatchPhaseBubble {
		t.Fatalf("phases = %v", phases)
	}
}

type testNoopAction struct{}

func (*testNoopAction) Name() string                { return "test::Noop" }
func (a *testNoopAction) BoxedClone() Action        { return &testNoopAction{} }
func (*testNoopAction) PartialEq(other Action) bool { _, ok := other.(*testNoopAction); return ok }

func TestApp_AsyncAppRefreshFlushesRefreshEffect(t *testing.T) {
	t.Skip("blocked:M0-blocked-tests: needs windows (TestAppContext.AddWindow) and the Render path")
}

func TestApp_MissingGlyphCallbacksFollowSubscriptionLifetime(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		report := func(grapheme string) {
			cx.Update(func(cx *App) {
				cx.TextSystem().missingGlyphReporter.Report([]MissingGlyph{testMissingGlyph(grapheme)})
			})
			cx.RunUntilParked()
		}

		var observed []MissingGlyph
		subscription := cx.UpdateR(func(cx *App) Subscription {
			return cx.OnMissingGlyphs(func(missingGlyphs []MissingGlyph, _ *App) {
				observed = append(observed, missingGlyphs...)
			})
		})
		report("active")
		if want := []MissingGlyph{testMissingGlyph("active")}; !slices.Equal(observed, want) {
			t.Fatalf("observed = %v, want %v", observed, want)
		}

		var secondObserved []MissingGlyph
		secondSubscription := cx.UpdateR(func(cx *App) Subscription {
			return cx.OnMissingGlyphs(func(missingGlyphs []MissingGlyph, _ *App) {
				secondObserved = append(secondObserved, missingGlyphs...)
			})
		})
		report("replacement")
		if want := []MissingGlyph{testMissingGlyph("active")}; !slices.Equal(observed, want) {
			t.Fatalf("observed = %v, want %v", observed, want)
		}
		if want := []MissingGlyph{testMissingGlyph("replacement")}; !slices.Equal(secondObserved, want) {
			t.Fatalf("second observed = %v, want %v", secondObserved, want)
		}

		subscription.Unsubscribe()
		report("after old drop")
		if want := []MissingGlyph{testMissingGlyph("replacement"), testMissingGlyph("after old drop")}; !slices.Equal(secondObserved, want) {
			t.Fatalf("second observed = %v, want %v", secondObserved, want)
		}

		secondSubscription.Unsubscribe()
		report("inactive")
		if len(secondObserved) != 2 {
			t.Fatalf("len(second observed) = %d, want 2", len(secondObserved))
		}
	})
}

func TestApp_GpuiBorrow(t *testing.T) {
	cx := TestAppContextSingle()
	defer cx.Close()
	observationCount := 0

	state := cx.UpdateR(func(cx *App) Entity[bool] {
		state := cx.New(func(*Context[bool]) *bool { return new(bool) })
		cx.Observe(state, func(Entity[bool], *App) {
			observationCount++
		}).Detach()
		return state
	})
	defer state.Release()

	cx.Update(func(cx *App) {
		borrow := state.AsMut(cx)
		*borrow.Get() = true
		borrow.Release()
	})

	cx.Update(func(cx *App) {
		state.Write(cx, false)
	})

	if observationCount != 2 {
		t.Fatalf("observation count = %d, want 2", observationCount)
	}
}

func TestApp_RestartPreservesPathAndArguments(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		tc.BlockTest(func(s *scheduler.Scope) {
			userDataDir := "/tmp/zed data/\xff"
			if runtime.GOOS == "windows" {
				userDataDir = "C:\\zed data"
			}
			arguments := []string{"--user-data-dir", userDataDir}
			restartPath := "updated-zed"
			_ = (&Application{app: cx.app}).WithRestartArguments(slices.Clone(arguments))
			restart := cx.ExpectRestart()

			cx.Update(func(cx *App) {
				cx.SetRestartPath(restartPath)
				cx.Restart()
			})

			request, err := restart.Recv(s)
			if err != nil {
				panic("restart was not requested")
			}
			if path, ok := request.BinaryPath.Get(); !ok || path != restartPath {
				t.Fatalf("restart path = %v, want Some(%q)", request.BinaryPath, restartPath)
			}
			if !slices.Equal(request.Arguments, arguments) {
				t.Fatalf("restart arguments = %q, want %q", request.Arguments, arguments)
			}
		})
	})
}
