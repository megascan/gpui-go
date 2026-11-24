//go:build darwin

package ios

import (
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func newUserActivity(activityType objc.ID) objc.ID {
	return objc.Send[objc.ID](objc.Alloc(objc.GetClass("NSUserActivity")), objc.Sel("initWithActivityType:"), activityType)
}

func TestApplication_BrowsingActivityDeliversURLWithoutLosingQueryOrFragment(t *testing.T) {
	objc.AutoreleasePool(func() {
		state := &IosApplicationState{}
		var received []string
		state.openURLs.set(func(urls []string) { received = append(received, urls...) })
		urlString := "https://example.com/thread/123?source=link#selection"
		url := objc.NSURLFromString(urlString)
		if url == objc.Nil {
			t.Fatal("test URL is valid")
		}
		activity := newUserActivity(objc.FoundationConstant("NSUserActivityTypeBrowsingWeb"))
		defer objc.Release(activity)
		state.continueUserActivity(activity)
		if len(received) != 0 {
			t.Fatalf("received = %v", received)
		}

		objc.SendVoid(activity, objc.Sel("setWebpageURL:"), url)
		state.continueUserActivity(activity)
		if !slices.Equal(received, []string{urlString}) {
			t.Fatalf("received = %v", received)
		}

		unrelatedType := objc.NSStringFromGo("com.example.edit")
		defer objc.Release(unrelatedType)
		unrelated := newUserActivity(unrelatedType)
		defer objc.Release(unrelated)
		objc.SendVoid(unrelated, objc.Sel("setWebpageURL:"), url)
		state.continueUserActivity(unrelated)
		if !slices.Equal(received, []string{urlString}) {
			t.Fatalf("received = %v", received)
		}
	})
}

func TestApplication_LaunchCallbackRunsOnceAndCanRegisterCallbacks(t *testing.T) {
	state := &IosApplicationState{}
	launches := 0
	state.finishLaunching.set(func() {
		launches++
		state.memoryWarning.set(func() {})
	})
	state.notifyFinishLaunching()
	state.notifyFinishLaunching()
	if launches != 1 {
		t.Fatalf("launches = %d", launches)
	}
	if _, ok := state.memoryWarning.take(); !ok {
		t.Fatal("memory warning callback was not registered")
	}
}

type lifecycleEvent struct {
	handler int
	phase   gpui.AppLifecyclePhase
}

func TestApplication_LifecycleCallbackReplacementSurvivesDispatch(t *testing.T) {
	state := &IosApplicationState{}
	var events []lifecycleEvent
	state.appLifecycle.set(func(phase gpui.AppLifecyclePhase) {
		events = append(events, lifecycleEvent{0, phase})
		state.appLifecycle.set(func(phase gpui.AppLifecyclePhase) {
			events = append(events, lifecycleEvent{1, phase})
		})
	})
	state.notifyAppLifecycle(gpui.AppLifecyclePhaseForeground)
	state.notifyAppLifecycle(gpui.AppLifecyclePhaseActive)
	want := []lifecycleEvent{{0, gpui.AppLifecyclePhaseForeground}, {1, gpui.AppLifecyclePhaseActive}}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestApplication_MemoryWarningCallbackIsRestoredAfterDispatch(t *testing.T) {
	state := &IosApplicationState{}
	warnings := 0
	state.memoryWarning.set(func() {
		warnings++
		state.quit.set(func() {})
	})
	state.notifyMemoryWarning()
	state.notifyMemoryWarning()
	if warnings != 2 {
		t.Fatalf("warnings = %d", warnings)
	}
	if _, ok := state.quit.take(); !ok {
		t.Fatal("quit callback was not registered")
	}
}

func TestApplication_LifecycleRemovalSurvivesReentrantDispatch(t *testing.T) {
	state := &IosApplicationState{}
	var events []gpui.AppLifecyclePhase
	state.appLifecycle.set(func(phase gpui.AppLifecyclePhase) {
		events = append(events, phase)
		state.appLifecycle.take()
		state.notifyAppLifecycle(gpui.AppLifecyclePhaseInactive)
	})
	state.notifyAppLifecycle(gpui.AppLifecyclePhaseDisconnected)
	state.notifyAppLifecycle(gpui.AppLifecyclePhaseForeground)
	if !slices.Equal(events, []gpui.AppLifecyclePhase{gpui.AppLifecyclePhaseDisconnected}) {
		t.Fatalf("events = %v", events)
	}
}

func TestApplication_ApplicationCallbacksAreNotSharedBetweenPlatforms(t *testing.T) {
	first := &IosApplicationState{}
	second := &IosApplicationState{}
	warnings := 0
	first.memoryWarning.set(func() { warnings++ })
	second.notifyMemoryWarning()
	if warnings != 0 {
		t.Fatalf("warnings = %d", warnings)
	}
	first.notifyMemoryWarning()
	if warnings != 1 {
		t.Fatalf("warnings = %d", warnings)
	}
}
