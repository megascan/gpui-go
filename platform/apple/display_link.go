//go:build darwin

package apple

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const quartzCorePath = "/System/Library/Frameworks/QuartzCore.framework/QuartzCore"

var (
	quartzCoreOnce sync.Once
	quartzCoreErr  error
)

func loadQuartzCore() error {
	quartzCoreOnce.Do(func() {
		if err := objc.Load(); err != nil {
			quartzCoreErr = err
			return
		}
		if _, err := purego.Dlopen(quartzCorePath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
			quartzCoreErr = fmt.Errorf("apple: open %s: %w", quartzCorePath, err)
		}
	})
	return quartzCoreErr
}

var (
	displayLinkDelegateOnce  sync.Once
	displayLinkDelegateClass objc.Class
	displayLinkDelegateErr   error
	displayLinkCallbacksMu   sync.Mutex
	displayLinkCallbacks     = map[objc.ID]func(update objc.ID){}
)

func metalDisplayLinkDelegateClass() (objc.Class, error) {
	displayLinkDelegateOnce.Do(func() {
		displayLinkDelegateClass, displayLinkDelegateErr = objc.RegisterClass(
			"GPUIMetalDisplayLinkDelegate",
			objc.GetClass("NSObject"),
			[]string{"NSObject", "CAMetalDisplayLinkDelegate"},
			nil,
			[]objc.MethodDef{{
				Cmd: objc.Sel("metalDisplayLink:needsUpdate:"),
				Fn:  metalDisplayLinkNeedsUpdate,
			}},
		)
	})
	return displayLinkDelegateClass, displayLinkDelegateErr
}

func metalDisplayLinkNeedsUpdate(self objc.ID, _ objc.SEL, _ objc.ID, update objc.ID) {
	displayLinkCallbacksMu.Lock()
	callback := displayLinkCallbacks[self]
	displayLinkCallbacksMu.Unlock()
	if callback == nil {
		return
	}
	delegate := objc.RetainObject(self)
	defer delegate.Release()
	callback(update)
}

func newMetalDisplayLinkDelegate(mainThread objc.MainThreadMarker, callback func(update objc.ID)) (objc.ID, error) {
	class, err := metalDisplayLinkDelegateClass()
	if err != nil {
		return objc.Nil, err
	}
	this := objc.AllocInit(class)
	displayLinkCallbacksMu.Lock()
	displayLinkCallbacks[this] = callback
	displayLinkCallbacksMu.Unlock()
	return this, nil
}

// MetalFrameSource is a main-run-loop source of Metal drawables on macOS 14+ and iOS/iPadOS 17+.
// Callbacks run on the main run loop, where GPUI renders. Close is its drop.
type MetalFrameSource struct {
	link       *objc.Retained
	delegate   *objc.Retained
	mainThread objc.MainThreadMarker
	closed     bool
}

// MetalFrameSourceIsAvailable reports whether this system has CAMetalDisplayLink (macOS 14+,
// iOS/iPadOS 17+).
func MetalFrameSourceIsAvailable() bool {
	if loadQuartzCore() != nil {
		return false
	}
	return objc.GetClass("CAMetalDisplayLink") != 0
}

// NewMetalFrameSource starts delivering CAMetalDisplayLinkUpdate objects for the CAMetalLayer
// layer to callback, or returns nil where CAMetalDisplayLink is unavailable. It panics, as
// upstream's non-null Retained does, when initWithMetalLayer: returns nil.
func NewMetalFrameSource(layer objc.ID, mainThread objc.MainThreadMarker, callback func(update objc.ID)) *MetalFrameSource {
	if !MetalFrameSourceIsAvailable() {
		return nil
	}
	delegateID, err := newMetalDisplayLinkDelegate(mainThread, callback)
	if err != nil {
		panic(err)
	}
	delegate := objc.Own(delegateID)
	linkID := objc.Send[objc.ID](objc.Alloc(objc.GetClass("CAMetalDisplayLink")), objc.Sel("initWithMetalLayer:"), layer)
	if linkID == objc.Nil {
		displayLinkCallbacksMu.Lock()
		delete(displayLinkCallbacks, delegate.ID())
		displayLinkCallbacksMu.Unlock()
		delegate.Release()
		panic("unexpected NULL returned from -[CAMetalDisplayLink initWithMetalLayer:]")
	}
	link := objc.Own(linkID)
	objc.SendVoid(link.ID(), objc.Sel("setDelegate:"), delegate.ID())
	mainRunLoop := objc.SendClass[objc.ID](objc.GetClass("NSRunLoop"), objc.Sel("mainRunLoop"))
	objc.SendVoid(link.ID(), objc.Sel("addToRunLoop:forMode:"), mainRunLoop, objc.FoundationConstant("NSRunLoopCommonModes"))
	return &MetalFrameSource{
		link:       link,
		delegate:   delegate,
		mainThread: mainThread,
	}
}

// SetPaused pauses or resumes the link; a paused link delivers no updates and acquires no drawables.
func (s *MetalFrameSource) SetPaused(paused bool) {
	objc.SendVoid(s.link.ID(), objc.Sel("setPaused:"), paused)
}

// IsPaused reports whether the link is paused.
func (s *MetalFrameSource) IsPaused() bool {
	return objc.Send[bool](s.link.ID(), objc.Sel("isPaused"))
}

// Close invalidates the link and releases it and its delegate. It is idempotent.
func (s *MetalFrameSource) Close() {
	if s.closed {
		return
	}
	s.closed = true
	objc.SendVoid(s.link.ID(), objc.Sel("invalidate"))
	s.link.Release()
	displayLinkCallbacksMu.Lock()
	delete(displayLinkCallbacks, s.delegate.ID())
	displayLinkCallbacksMu.Unlock()
	s.delegate.Release()
}
