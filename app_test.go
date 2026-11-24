package gpui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
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
