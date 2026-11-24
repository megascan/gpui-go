//go:build darwin

package apple

import (
	"testing"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

func TestDisplayLink_Selectors(t *testing.T) {
	if err := loadQuartzCore(); err != nil {
		t.Fatal(err)
	}
	if objc.GetClass("CAMetalLayer") == 0 {
		t.Fatal("CAMetalLayer not found")
	}
	if !MetalFrameSourceIsAvailable() {
		t.Skip("CAMetalDisplayLink needs macOS 14+ or iOS 17+")
	}
	link := objc.GetClass("CAMetalDisplayLink")
	for _, s := range []string{"initWithMetalLayer:", "setDelegate:", "addToRunLoop:forMode:", "setPaused:", "isPaused", "invalidate"} {
		if !objc.InstancesRespondToSelector(link, objc.Sel(s)) {
			t.Errorf("CAMetalDisplayLink does not respond to %s", s)
		}
	}
	if objc.GetClass("NSRunLoop") == 0 || objc.FoundationConstant("NSRunLoopCommonModes") == objc.Nil {
		t.Fatal("NSRunLoop or NSRunLoopCommonModes missing")
	}
}

func TestDisplayLink_DelegateForwardsUpdates(t *testing.T) {
	if err := loadQuartzCore(); err != nil {
		t.Fatal(err)
	}
	var got []objc.ID
	delegateID, err := newMetalDisplayLinkDelegate(objc.MainThreadMarker{}, func(update objc.ID) { got = append(got, update) })
	if err != nil {
		t.Fatal(err)
	}
	delegate := objc.Own(delegateID)
	class, _ := metalDisplayLinkDelegateClass()
	if !objc.InstancesRespondToSelector(class, objc.Sel("metalDisplayLink:needsUpdate:")) {
		t.Fatal("delegate class lacks metalDisplayLink:needsUpdate:")
	}
	if proto := objc.GetProtocol("CAMetalDisplayLinkDelegate"); proto != nil {
		if !objc.Send[bool](delegate.ID(), objc.Sel("conformsToProtocol:"), proto) {
			t.Error("delegate does not conform to CAMetalDisplayLinkDelegate")
		}
	}
	marker := objc.NSStringFromGo("update")
	defer objc.Release(marker)
	objc.SendVoid(delegate.ID(), objc.Sel("metalDisplayLink:needsUpdate:"), objc.Nil, marker)
	if len(got) != 1 || got[0] != marker {
		t.Fatalf("callback got %v", got)
	}
	displayLinkCallbacksMu.Lock()
	delete(displayLinkCallbacks, delegate.ID())
	displayLinkCallbacksMu.Unlock()
	delegate.Release()
}

func TestDisplayLink_CloseInvalidatesAndReleases(t *testing.T) {
	if !MetalFrameSourceIsAvailable() {
		t.Skip("CAMetalDisplayLink needs macOS 14+ or iOS 17+")
	}
	layer := objc.SendClass[objc.ID](objc.GetClass("CAMetalLayer"), objc.Sel("layer"))
	source := NewMetalFrameSource(layer, objc.MainThreadMarker{}, func(objc.ID) {})
	if source == nil {
		t.Fatal("NewMetalFrameSource returned nil where CAMetalDisplayLink exists")
	}
	source.SetPaused(true)
	if !source.IsPaused() {
		t.Fatal("IsPaused after SetPaused(true) is false")
	}
	source.SetPaused(false)
	if source.IsPaused() {
		t.Fatal("IsPaused after SetPaused(false) is true")
	}
	delegate := source.delegate.ID()
	source.Close()
	source.Close()
	displayLinkCallbacksMu.Lock()
	_, present := displayLinkCallbacks[delegate]
	displayLinkCallbacksMu.Unlock()
	if present {
		t.Fatal("Close left the delegate callback registered")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("link used after Close did not panic")
		}
	}()
	_ = source.link.ID()
}
