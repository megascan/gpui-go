//go:build darwin

package apple

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

func postNotification(center objc.ID, name objc.ID) {
	objc.SendVoid(center, objc.Sel("postNotificationName:object:"), name, processInfo())
}

func TestProcessInfo_NotificationsAreDeferredAndCancelledOnDrop(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		foreground := cx.ForegroundExecutor()
		background := cx.Executor()
		for _, name := range []objc.ID{
			objc.FoundationConstant("NSProcessInfoPowerStateDidChangeNotification"),
			objc.FoundationConstant("NSProcessInfoThermalStateDidChangeNotification"),
		} {
			checkNotificationLifecycle(t, &foreground, &background, cx.RunUntilParked, name)
		}
	})
}

func checkNotificationLifecycle(t *testing.T, foreground *gpui.ForegroundExecutor, background *gpui.BackgroundExecutor, runUntilParked func(), name objc.ID) {
	t.Helper()
	center := objc.Own(objc.AllocInit(objc.GetClass("NSNotificationCenter")))
	defer center.Release()
	calls := 0
	observer := processInfoObserverWithCenter(
		objc.RetainObject(center.ID()),
		name,
		foreground,
		func() { calls++ },
	)
	postNotification(center.ID(), name)
	if calls != 0 {
		t.Fatalf("calls = %d, want 0", calls)
	}
	runUntilParked()
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}

	centerID := center.ID()
	nameString := objc.GoString(name)
	background.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		n := objc.NSStringFromGo(nameString)
		defer objc.Release(n)
		postNotification(centerID, n)
		return struct{}{}, nil
	}).Detach()
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	runUntilParked()
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}

	postNotification(center.ID(), name)
	observer.Close()
	runUntilParked()
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	postNotification(center.ID(), name)
	runUntilParked()
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestProcessInfo_StateQueries(t *testing.T) {
	info := processInfo()
	for _, s := range []string{"thermalState", "isLowPowerModeEnabled", "beginActivityWithOptions:reason:", "endActivity:"} {
		if !objc.RespondsToSelector(info, objc.Sel(s)) {
			t.Fatalf("NSProcessInfo does not respond to %s", s)
		}
	}
	switch ThermalState() {
	case gpui.ThermalStateNominal, gpui.ThermalStateFair, gpui.ThermalStateSerious, gpui.ThermalStateCritical:
	default:
		t.Fatal("unknown thermal state")
	}
	switch PowerMode() {
	case gpui.PowerModeLowPower, gpui.PowerModeUnknown:
	default:
		t.Fatal("unexpected power mode")
	}
	center := objc.GetClass("NSNotificationCenter")
	for _, s := range []string{"addObserverForName:object:queue:usingBlock:", "removeObserver:", "postNotificationName:object:"} {
		if !objc.InstancesRespondToSelector(center, objc.Sel(s)) {
			t.Fatalf("NSNotificationCenter does not respond to %s", s)
		}
	}
}
