//go:build darwin && !ios

package macos

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

const coreVideoPath = "/System/Library/Frameworks/CoreVideo.framework/CoreVideo"

var (
	registryMu sync.Mutex
	registry   = displayLinkRegistry{displays: map[uint32]*displayEntry{}}
)

type displayLinkRegistry struct {
	displays         map[uint32]*displayEntry
	nextSubscriberID uint64
}

type displayEntry struct {
	link        DisplayLink
	running     bool
	subscribers []displayLinkSubscriber
}

type displayLinkSubscriber struct {
	id            subscriberID
	frameRequests dispatchSource
	signal        *gpui.PlatformFrameSignal
}

type subscriberID uint64

func lockRegistry() *displayLinkRegistry {
	registryMu.Lock()
	return &registry
}

func unlockRegistry() {
	registryMu.Unlock()
}

func debugAssertMainThread() {
	if gpuiutil.DebugAssertions && !objc.IsMainThread() {
		panic("display link registry mutations must happen on the main thread; " +
			"the registry's lock ordering and state consistency depend on it")
	}
}

var (
	outputCallbackOnce sync.Once
	outputCallback     CVDisplayLinkOutputCallback
)

func displayLinkOutputCallbackPtr() CVDisplayLinkOutputCallback {
	outputCallbackOnce.Do(func() {
		outputCallback = CVDisplayLinkOutputCallback(purego.NewCallback(displayLinkOutputCallback))
	})
	return outputCallback
}

func displayLinkOutputCallback(_ uintptr, currentTime *cvTimeStamp, _ *cvTimeStamp, _ int64, _ uintptr, displayID uintptr) int32 {
	signalAt, hasSignal := gpui.PlatformFrameSignalCapture(func() scheduler.Instant {
		currentHostTime := cvGetCurrentHostTime()
		receivedAt := scheduler.InstantNow()
		hostClockFrequency := cvGetHostClockFrequency()
		return frameRequestInstant(currentTime, currentHostTime, hostClockFrequency, receivedAt)
	})
	id := uint32(displayID)
	r := lockRegistry()
	defer unlockRegistry()
	if entry, ok := r.displays[id]; ok {
		for _, subscriber := range entry.subscribers {
			if hasSignal {
				subscriber.signal.Record(signalAt, gpui.FrameRequestSourceNativeCallback)
			}
			subscriber.frameRequests.mergeData(1)
		}
	}
	return 0
}

func frameRequestInstant(timestamp *cvTimeStamp, currentHostTime uint64, hostClockFrequency float64, receivedAt scheduler.Instant) scheduler.Instant {
	if timestamp == nil || timestamp.version != 0 || timestamp.flags&Kcvtimestamphosttimevalid == 0 {
		return receivedAt
	}
	if math.IsInf(hostClockFrequency, 0) || math.IsNaN(hostClockFrequency) || hostClockFrequency <= 0 {
		return receivedAt
	}
	if currentHostTime < timestamp.hostTime {
		return receivedAt
	}
	elapsedTicks := currentHostTime - timestamp.hostTime
	secs := float64(elapsedTicks) / hostClockFrequency
	if math.IsInf(secs, 0) || math.IsNaN(secs) || secs < 0 || secs >= float64(math.MaxInt64)/1e9 {
		return receivedAt
	}
	elapsed := time.Duration(math.Round(secs * 1e9))
	return receivedAt.Add(-elapsed)
}

func subscribe(displayID uint32, frameRequests dispatchSource, signal *gpui.PlatformFrameSignal) (subscriberID, error) {
	debugAssertMainThread()

	r := lockRegistry()
	_, exists := r.displays[displayID]
	unlockRegistry()
	var newLink DisplayLink
	hasNewLink := false
	if !exists {
		link, err := NewDisplayLink(displayID, displayLinkOutputCallbackPtr(), uintptr(displayID))
		if err != nil {
			return 0, err
		}
		newLink, hasNewLink = link, true
	}

	id, linkToStart, err := func() (subscriberID, DisplayLink, error) {
		r := lockRegistry()
		defer unlockRegistry()
		id := subscriberID(r.nextSubscriberID)
		r.nextSubscriberID++
		entry, ok := r.displays[displayID]
		switch {
		case ok:
			if hasNewLink {
				newLink.Release()
			}
		case hasNewLink:
			entry = &displayEntry{link: newLink}
			r.displays[displayID] = entry
		default:
			return 0, 0, fmt.Errorf("display link registry entry vanished for display %d", displayID)
		}
		entry.subscribers = append(entry.subscribers, displayLinkSubscriber{
			id:            id,
			frameRequests: frameRequests.retain(),
			signal:        signal,
		})
		if entry.running {
			return id, 0, nil
		}
		entry.running = true
		return id, entry.link.Clone(), nil
	}()
	if err != nil {
		return 0, err
	}

	if linkToStart != 0 {
		err := linkToStart.Ref().Start()
		linkToStart.Release()
		if err != nil {
			r := lockRegistry()
			if entry, ok := r.displays[displayID]; ok {
				entry.running = false
				kept := entry.subscribers[:0]
				for _, subscriber := range entry.subscribers {
					if subscriber.id == id {
						subscriber.signal.Take()
						subscriber.frameRequests.release()
					} else {
						kept = append(kept, subscriber)
					}
				}
				entry.subscribers = kept
			}
			unlockRegistry()
			return 0, err
		}
	}
	return id, nil
}

func unsubscribe(displayID uint32, id subscriberID) {
	debugAssertMainThread()

	linkToStop := func() DisplayLink {
		r := lockRegistry()
		defer unlockRegistry()
		entry, ok := r.displays[displayID]
		if !ok {
			return 0
		}
		kept := entry.subscribers[:0]
		for _, subscriber := range entry.subscribers {
			if subscriber.id != id {
				kept = append(kept, subscriber)
			} else {
				subscriber.frameRequests.release()
			}
		}
		entry.subscribers = kept
		if len(entry.subscribers) == 0 && entry.running {
			entry.running = false
			return entry.link.Clone()
		}
		return 0
	}()

	if linkToStop != 0 {
		gpuiutil.LogErr(linkToStop.Ref().Stop())
		linkToStop.Release()
	}
}

// MetalFrames is a window's CAMetalDisplayLink, shared with GPUI's frame waker so that a wake
// reaches the current link without taking the window's lock. The link pauses after any update in
// which GPUI neither drew, presented, nor asked for another frame.
type MetalFrames struct {
	source *apple.MetalFrameSource
	woken  bool
}

// Replace swaps the frame source, closing the previous one.
func (f *MetalFrames) Replace(source *apple.MetalFrameSource) {
	old := f.source
	f.source = source
	if old != nil && old != source {
		old.Close()
	}
}

// IsAttached reports whether a frame source is attached.
func (f *MetalFrames) IsAttached() bool {
	return f.source != nil
}

// Wake records that GPUI asked for a frame and unpauses the link.
func (f *MetalFrames) Wake() {
	f.woken = true
	if f.source != nil {
		f.source.SetPaused(false)
	}
}

// BeginFrame starts an update.
func (f *MetalFrames) BeginFrame() {
	f.woken = false
}

// DebugState returns the link's diagnostic state.
func (f *MetalFrames) DebugState() jsontext.Value {
	var paused *bool
	if f.source != nil {
		p := f.source.IsPaused()
		paused = &p
	}
	return mustMarshalDebugState(struct {
		Attached bool   `json:"attached"`
		Kind     string `json:"kind"`
		Paused   *bool  `json:"paused"`
		Woken    bool   `json:"woken"`
	}{f.IsAttached(), "CAMetalDisplayLink", paused, f.woken})
}

// EndFrame ends an update, pausing the link if GPUI neither used its drawable nor asked for
// another frame while it ran. It reports whether it paused.
func (f *MetalFrames) EndFrame(usedDrawable bool) bool {
	pause := !usedDrawable && !f.woken
	if pause && f.source != nil {
		f.source.SetPaused(true)
	}
	return pause
}

func mustMarshalDebugState(v any) jsontext.Value {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return jsontext.Value(b)
}

// WindowFrameSource is a per-window source of frame requests, paced by the display the window is
// on. Its dispatch source coalesces vsync ticks from the display's io thread and invokes
// callback(data) on the main queue. Close is its drop.
type WindowFrameSource struct {
	frameRequests dispatchSource
	registration  *windowFrameRegistration
	signal        *gpui.PlatformFrameSignal
	closed        bool
}

type windowFrameRegistration struct {
	displayID    uint32
	subscriberID subscriberID
}

// NewWindowFrameSource creates a frame source whose event handler calls callback(data) on the
// main queue.
func NewWindowFrameSource(data uintptr, callback gcd.Function) *WindowFrameSource {
	mustLoadDispatch()
	frameRequests := dispatchSource(dispatchSourceCreate(dispatchSourceTypeDataAdd, 0, 0, gcd.MainQueue()))
	dispatchSetContext(uintptr(frameRequests), data)
	dispatchSourceSetEventHandlerF(uintptr(frameRequests), callback)
	dispatchResume(uintptr(frameRequests))
	return &WindowFrameSource{
		frameRequests: frameRequests,
		signal:        gpui.NewPlatformFrameSignal(),
	}
}

// Start subscribes the source to the display link of displayID, replacing any earlier
// subscription.
func (s *WindowFrameSource) Start(displayID uint32) error {
	s.Stop()
	id, err := subscribe(displayID, s.frameRequests, s.signal)
	if err != nil {
		return err
	}
	s.registration = &windowFrameRegistration{displayID: displayID, subscriberID: id}
	return nil
}

// DebugState returns the source's diagnostic state.
func (s *WindowFrameSource) DebugState() jsontext.Value {
	var displayID *uint32
	if s.registration != nil {
		id := s.registration.displayID
		displayID = &id
	}
	return mustMarshalDebugState(struct {
		Attached  bool    `json:"attached"`
		DisplayID *uint32 `json:"display_id"`
		Kind      string  `json:"kind"`
	}{s.registration != nil, displayID, "CVDisplayLink"})
}

// TakeSignal drains the first frame request time and source since the last call.
func (s *WindowFrameSource) TakeSignal() (scheduler.Instant, gpui.FrameRequestSource, bool) {
	return s.signal.Take()
}

// Stop unsubscribes from the display link.
func (s *WindowFrameSource) Stop() {
	if registration := s.registration; registration != nil {
		s.registration = nil
		unsubscribe(registration.displayID, registration.subscriberID)
	}
	s.signal.Take()
}

// Close stops the source, cancels its dispatch source so the event handler never runs again, and
// releases it. It is idempotent.
func (s *WindowFrameSource) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.Stop()
	dispatchSourceCancel(uintptr(s.frameRequests))
	s.frameRequests.release()
}

type dispatchSource uintptr

func (d dispatchSource) retain() dispatchSource {
	dispatchRetain(uintptr(d))
	return d
}

func (d dispatchSource) release() {
	dispatchRelease(uintptr(d))
}

func (d dispatchSource) mergeData(value uintptr) {
	dispatchSourceMergeData(uintptr(d), value)
}

var (
	dispatchOnce sync.Once
	dispatchErr  error

	dispatchSourceTypeDataAdd      uintptr
	dispatchSourceCreate           func(sourceType uintptr, handle uintptr, mask uintptr, queue gcd.Queue) uintptr
	dispatchSetContext             func(object uintptr, context uintptr)
	dispatchSourceSetEventHandlerF func(source uintptr, handler gcd.Function)
	dispatchResume                 func(object uintptr)
	dispatchSourceMergeData        func(source uintptr, value uintptr)
	dispatchSourceCancel           func(source uintptr)
	dispatchRetain                 func(object uintptr)
	dispatchRelease                func(object uintptr)
)

func loadDispatch() error {
	dispatchOnce.Do(func() {
		if err := gcd.Load(); err != nil {
			dispatchErr = err
			return
		}
		lib := gcd.LibSystem()
		addr, err := purego.Dlsym(lib, "_dispatch_source_type_data_add")
		if err != nil {
			dispatchErr = fmt.Errorf("macos: %w", err)
			return
		}
		dispatchSourceTypeDataAdd = addr
		purego.RegisterLibFunc(&dispatchSourceCreate, lib, "dispatch_source_create")
		purego.RegisterLibFunc(&dispatchSetContext, lib, "dispatch_set_context")
		purego.RegisterLibFunc(&dispatchSourceSetEventHandlerF, lib, "dispatch_source_set_event_handler_f")
		purego.RegisterLibFunc(&dispatchResume, lib, "dispatch_resume")
		purego.RegisterLibFunc(&dispatchSourceMergeData, lib, "dispatch_source_merge_data")
		purego.RegisterLibFunc(&dispatchSourceCancel, lib, "dispatch_source_cancel")
		purego.RegisterLibFunc(&dispatchRetain, lib, "dispatch_retain")
		purego.RegisterLibFunc(&dispatchRelease, lib, "dispatch_release")
	})
	return dispatchErr
}

func mustLoadDispatch() {
	if err := loadDispatch(); err != nil {
		panic(err)
	}
}

// CVDisplayLink is the opaque CoreVideo display link object.
type CVDisplayLink struct{}

type cvTimeStamp struct {
	version            uint32
	videoTimeScale     int32
	videoTime          int64
	hostTime           uint64
	rateScalar         float64
	videoRefreshPeriod int64
	smpteTime          cvSMPTETime
	flags              uint64
	reserved           uint64
}

// CVTimeStampFlags are the flags of a CVTimeStamp.
type CVTimeStampFlags = uint64

// CVTimeStamp flags.
const (
	Kcvtimestampvideotimevalid          CVTimeStampFlags = 1 << 0
	Kcvtimestamphosttimevalid           CVTimeStampFlags = 1 << 1
	Kcvtimestampsmptetimevalid          CVTimeStampFlags = 1 << 2
	Kcvtimestampvideorefreshperiodvalid CVTimeStampFlags = 1 << 3
	Kcvtimestampratescalarvalid         CVTimeStampFlags = 1 << 4
	Kcvtimestamptopfield                CVTimeStampFlags = 1 << 16
	Kcvtimestampbottomfield             CVTimeStampFlags = 1 << 17
	Kcvtimestampvideohosttimevalid                       = Kcvtimestampvideotimevalid | Kcvtimestamphosttimevalid
	Kcvtimestampisinterlaced                             = Kcvtimestamptopfield | Kcvtimestampbottomfield
)

type cvSMPTETime struct {
	subframes       int16
	subframeDivisor int16
	counter         uint32
	timeType        uint32
	flags           uint32
	hours           int16
	minutes         int16
	seconds         int16
	frames          int16
}

// CVSMPTETimeType is the type of a CVSMPTETime.
type CVSMPTETimeType = uint32

// CVSMPTETime types.
const (
	Kcvsmptetimetype24       CVSMPTETimeType = 0
	Kcvsmptetimetype25       CVSMPTETimeType = 1
	Kcvsmptetimetype30drop   CVSMPTETimeType = 2
	Kcvsmptetimetype30       CVSMPTETimeType = 3
	Kcvsmptetimetype2997     CVSMPTETimeType = 4
	Kcvsmptetimetype2997drop CVSMPTETimeType = 5
	Kcvsmptetimetype60       CVSMPTETimeType = 6
	Kcvsmptetimetype5994     CVSMPTETimeType = 7
)

// CVSMPTETimeFlags are the flags of a CVSMPTETime.
type CVSMPTETimeFlags = uint32

// CVSMPTETime flags.
const (
	Kcvsmptetimevalid   CVSMPTETimeFlags = 1 << 0
	Kcvsmptetimerunning CVSMPTETimeFlags = 1 << 1
)

// CVDisplayLinkOutputCallback is a C function pointer of CoreVideo's output callback type.
type CVDisplayLinkOutputCallback uintptr

var (
	coreVideoOnce sync.Once
	coreVideoErr  error

	cvDisplayLinkCreateWithActiveCGDisplays func(displayLinkOut *uintptr) int32
	cvDisplayLinkSetCurrentCGDisplay        func(displayLink uintptr, displayID uint32) int32
	cvDisplayLinkSetOutputCallback          func(displayLink uintptr, callback CVDisplayLinkOutputCallback, userInfo uintptr) int32
	cvDisplayLinkStart                      func(displayLink uintptr) int32
	cvDisplayLinkStop                       func(displayLink uintptr) int32
	cvDisplayLinkRelease                    func(displayLink uintptr)
	cvDisplayLinkRetain                     func(displayLink uintptr) uintptr
	cvGetCurrentHostTime                    func() uint64
	cvGetHostClockFrequency                 func() float64
)

func loadCoreVideo() error {
	coreVideoOnce.Do(func() {
		lib, err := purego.Dlopen(coreVideoPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			coreVideoErr = fmt.Errorf("macos: open %s: %w", coreVideoPath, err)
			return
		}
		purego.RegisterLibFunc(&cvDisplayLinkCreateWithActiveCGDisplays, lib, "CVDisplayLinkCreateWithActiveCGDisplays")
		purego.RegisterLibFunc(&cvDisplayLinkSetCurrentCGDisplay, lib, "CVDisplayLinkSetCurrentCGDisplay")
		purego.RegisterLibFunc(&cvDisplayLinkSetOutputCallback, lib, "CVDisplayLinkSetOutputCallback")
		purego.RegisterLibFunc(&cvDisplayLinkStart, lib, "CVDisplayLinkStart")
		purego.RegisterLibFunc(&cvDisplayLinkStop, lib, "CVDisplayLinkStop")
		purego.RegisterLibFunc(&cvDisplayLinkRelease, lib, "CVDisplayLinkRelease")
		purego.RegisterLibFunc(&cvDisplayLinkRetain, lib, "CVDisplayLinkRetain")
		purego.RegisterLibFunc(&cvGetCurrentHostTime, lib, "CVGetCurrentHostTime")
		purego.RegisterLibFunc(&cvGetHostClockFrequency, lib, "CVGetHostClockFrequency")
	})
	return coreVideoErr
}

// DisplayLink owns one reference to a CVDisplayLink. Release is its drop.
type DisplayLink uintptr

// DisplayLinkRef is a borrowed CVDisplayLink.
type DisplayLinkRef uintptr

// NewDisplayLink creates a display link for displayID that calls callback with userInfo
// (CVDisplayLinkCreateWithActiveCGDisplays, then the callback and the display).
func NewDisplayLink(displayID uint32, callback CVDisplayLinkOutputCallback, userInfo uintptr) (DisplayLink, error) {
	if err := loadCoreVideo(); err != nil {
		return 0, err
	}
	var displayLink uintptr
	if code := cvDisplayLinkCreateWithActiveCGDisplays(&displayLink); code != 0 {
		return 0, fmt.Errorf("could not create display link, code: %d", code)
	}
	link := DisplayLink(displayLink)
	if code := cvDisplayLinkSetOutputCallback(displayLink, callback, userInfo); code != 0 {
		link.Release()
		return 0, fmt.Errorf("could not set output callback, code: %d", code)
	}
	if code := cvDisplayLinkSetCurrentCGDisplay(displayLink, displayID); code != 0 {
		link.Release()
		return 0, fmt.Errorf("could not assign display to display link, code: %d", code)
	}
	return link, nil
}

// Ref borrows the link.
func (l DisplayLink) Ref() DisplayLinkRef { return DisplayLinkRef(l) }

// Clone retains the link and returns the new reference.
func (l DisplayLink) Clone() DisplayLink {
	return DisplayLink(cvDisplayLinkRetain(uintptr(l)))
}

// Release drops this reference.
func (l DisplayLink) Release() {
	if l != 0 {
		cvDisplayLinkRelease(uintptr(l))
	}
}

// Start starts the link (CVDisplayLinkStart).
func (l DisplayLinkRef) Start() error {
	if code := cvDisplayLinkStart(uintptr(l)); code != 0 {
		return fmt.Errorf("could not start display link, code: %d", code)
	}
	return nil
}

// Stop stops the link (CVDisplayLinkStop).
func (l DisplayLinkRef) Stop() error {
	if code := cvDisplayLinkStop(uintptr(l)); code != 0 {
		return fmt.Errorf("could not stop display link, code: %d", code)
	}
	return nil
}
