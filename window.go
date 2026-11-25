package gpui

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// DefaultWindowSize is the default window size used when no explicit size is provided.
var DefaultWindowSize = NewSize(Px(1536.), Px(1095.))

// DefaultAdditionalWindowSize is a 6:5 aspect ratio minimum window size for functional windows
// additional to the main one, like settings windows.
var DefaultAdditionalWindowSize = Size[Pixels]{Width: Pixels(900.), Height: Pixels(750.)}

// DispatchPhase is one of the two phases of event dispatch.
type DispatchPhase uint8

const (
	// DispatchPhaseBubble invokes mouse listeners front to back and keyboard listeners from the
	// focused element to the root. It comes after the capture phase and is the default.
	DispatchPhaseBubble DispatchPhase = iota
	// DispatchPhaseCapture invokes mouse listeners back to front and keyboard listeners from the
	// root toward the focused element.
	DispatchPhaseCapture
)

// Bubble reports whether this is the bubble phase.
func (p DispatchPhase) Bubble() bool {
	return p == DispatchPhaseBubble
}

// Capture reports whether this is the capture phase.
func (p DispatchPhase) Capture() bool {
	return p == DispatchPhaseCapture
}

// String formats the phase like upstream's Debug.
func (p DispatchPhase) String() string {
	if p == DispatchPhaseCapture {
		return "Capture"
	}
	return "Bubble"
}

type windowInvalidatorInner struct {
	windowID      WindowID
	dirty         bool
	drawPhase     drawPhase
	dirtyViews    collections.HashSet[EntityID]
	updateCount   int
	frameDirty    frameDirtyAccumulator
	platformWaker func()
}

type frameDirtyAccumulator struct {
	dirtyAt       Option[scheduler.Instant]
	invalidations uint64
}

type windowInvalidator struct {
	inner *windowInvalidatorInner
}

func newWindowInvalidator(windowID WindowID) windowInvalidator {
	return windowInvalidator{inner: &windowInvalidatorInner{
		windowID:   windowID,
		dirty:      true,
		drawPhase:  drawPhaseNone,
		dirtyViews: collections.HashSet[EntityID]{},
	}}
}

func (w windowInvalidator) invalidateView(entity EntityID, cx *App) bool {
	inner := w.inner
	inner.updateCount++
	inner.dirtyViews[entity] = struct{}{}
	if inner.drawPhase == drawPhaseNone {
		var dirtyAt scheduler.Instant
		if profilerFeatureEnabled {
			dirtyAt = recordFrameDirty(inner)
		}
		becameDirty := !inner.dirty
		inner.dirty = true
		var waker func()
		if becameDirty {
			waker = inner.platformWaker
		}
		windowID := inner.windowID
		if profilerFeatureEnabled && becameDirty {
			recordFramePending(windowID, dirtyAt)
		}
		cx.pushEffect(effectNotify{emitter: entity})
		if waker != nil {
			waker()
		}
		return true
	}
	return false
}

func (w windowInvalidator) isDirty() bool {
	return w.inner.dirty
}

func (w windowInvalidator) setDirty(dirty bool) {
	inner := w.inner
	becameDirty := dirty && !inner.dirty
	inner.dirty = dirty
	if dirty {
		inner.updateCount++
	}
	var dirtyAt Option[scheduler.Instant]
	if profilerFeatureEnabled && dirty {
		dirtyAt = Some(recordFrameDirty(inner))
	}
	var waker func()
	if becameDirty {
		waker = inner.platformWaker
	}
	windowID := inner.windowID
	if profilerFeatureEnabled && becameDirty {
		if at, ok := dirtyAt.Get(); ok {
			recordFramePending(windowID, at)
		}
	}
	if waker != nil {
		waker()
	}
}

func (w windowInvalidator) setPlatformWaker(waker func()) {
	inner := w.inner
	inner.platformWaker = waker
	var wake func()
	if inner.dirty {
		wake = inner.platformWaker
	}
	if wake != nil {
		wake()
	}
}

func (w windowInvalidator) wakePlatform() {
	if waker := w.inner.platformWaker; waker != nil {
		waker()
	}
}

func (w windowInvalidator) setPhase(phase drawPhase) {
	w.inner.drawPhase = phase
}

func (w windowInvalidator) updateCount() int {
	return w.inner.updateCount
}

func recordFrameDirty(inner *windowInvalidatorInner) scheduler.Instant {
	if !inner.frameDirty.dirtyAt.IsSome() {
		inner.frameDirty.dirtyAt = Some(scheduler.InstantNow())
	}
	inner.frameDirty.invalidations++
	return inner.frameDirty.dirtyAt.Unwrap()
}

func (w windowInvalidator) takeFrameDirty() frameDirtyAccumulator {
	taken := w.inner.frameDirty
	w.inner.frameDirty = frameDirtyAccumulator{}
	return taken
}

func (w windowInvalidator) takeViews() collections.HashSet[EntityID] {
	views := w.inner.dirtyViews
	w.inner.dirtyViews = collections.HashSet[EntityID]{}
	return views
}

func (w windowInvalidator) replaceViews(views collections.HashSet[EntityID]) {
	w.inner.dirtyViews = views
}

func (w windowInvalidator) notDrawing() bool {
	return w.inner.drawPhase == drawPhaseNone
}

func (w windowInvalidator) debugAssertPaint() {
	if gpuiutil.DebugAssertions && w.inner.drawPhase != drawPhasePaint {
		panic("this method can only be called during paint")
	}
}

func (w windowInvalidator) debugAssertPrepaint() {
	if gpuiutil.DebugAssertions && w.inner.drawPhase != drawPhasePrepaint {
		panic("this method can only be called during request_layout, or prepaint")
	}
}

func (w windowInvalidator) debugAssertPaintOrPrepaint() {
	if gpuiutil.DebugAssertions && w.inner.drawPhase != drawPhasePaint && w.inner.drawPhase != drawPhasePrepaint {
		panic("this method can only be called during request_layout, prepaint, or paint")
	}
}

type anyObserver func(window *Window, cx *App) bool

type anyWindowFocusListener func(event *windowFocusEvent, window *Window, cx *App) bool

type windowFocusEvent struct {
	previousFocusPath []FocusID
	currentFocusPath  []FocusID
}

func (e *windowFocusEvent) isFocusIn(focusID FocusID) bool {
	return !slices.Contains(e.previousFocusPath, focusID) && slices.Contains(e.currentFocusPath, focusID)
}

func (e *windowFocusEvent) isFocusOut(focusID FocusID) bool {
	return slices.Contains(e.previousFocusPath, focusID) && !slices.Contains(e.currentFocusPath, focusID)
}

// FocusOutEvent is provided when subscribing for Context.OnFocusOut events.
type FocusOutEvent struct {
	// Blurred is a weak focus handle representing what was blurred.
	Blurred WeakFocusHandle
}

// FocusID is a globally unique identifier for a focusable element (a slot map key: version in
// the high 32 bits, index in the low 32 bits).
type FocusID uint64

func (id FocusID) idx() uint32 {
	return uint32(id)
}

func (id FocusID) version() uint32 {
	return uint32(id>>32) | 1
}

// GoString formats the id like upstream's Debug, FocusId(<idx>v<version>).
func (id FocusID) GoString() string {
	return fmt.Sprintf("FocusId(%dv%d)", id.idx(), id.version())
}

type elementArena struct {
	scopeDepth int
	drops      []func()
}

func (a *elementArena) beginScope() {
	a.scopeDepth++
}

func (a *elementArena) endScope() {
	if a.scopeDepth == 0 {
		panic("Arena::end_scope called without a matching begin_scope")
	}
	a.scopeDepth--
}

func (a *elementArena) clear() {
	if a.scopeDepth == 0 {
		a.forceClear()
	} else {
		slog.Debug(fmt.Sprintf("deferring arena clear; %d enclosing scope(s) still active", a.scopeDepth))
	}
}

func (a *elementArena) forceClear() {
	drops := a.drops
	a.drops = nil
	for _, drop := range drops {
		drop()
	}
}

var fallbackElementArena = &elementArena{}

var currentElementArena atomic.Pointer[elementArena]

var appElementArenas struct {
	mu     sync.Mutex
	arenas map[weak.Pointer[AppCell]]*elementArena
}

func (cx *App) elementArena() *elementArena {
	appElementArenas.mu.Lock()
	defer appElementArenas.mu.Unlock()
	if arena, ok := appElementArenas.arenas[cx.this]; ok {
		return arena
	}
	if appElementArenas.arenas == nil {
		appElementArenas.arenas = make(map[weak.Pointer[AppCell]]*elementArena)
	}
	arena := &elementArena{}
	appElementArenas.arenas[cx.this] = arena
	if cell := cx.this.Value(); cell != nil {
		runtime.AddCleanup(cell, func(key weak.Pointer[AppCell]) {
			appElementArenas.mu.Lock()
			delete(appElementArenas.arenas, key)
			appElementArenas.mu.Unlock()
		}, cx.this)
	}
	return arena
}

func drawInProgress() bool {
	return currentElementArena.Load() != nil
}

func withElementArena[R any](callback func(arena *elementArena) R) R {
	if arena := currentElementArena.Load(); arena != nil {
		return callback(arena)
	}
	return callback(fallbackElementArena)
}

type elementArenaScope struct {
	entered  *elementArena
	previous *elementArena
	exited   bool
	ended    bool
}

func enterElementArenaScope(arena *elementArena) *elementArenaScope {
	arena.beginScope()
	previous := currentElementArena.Swap(arena)
	return &elementArenaScope{entered: arena, previous: previous}
}

func (s *elementArenaScope) exit(arena *elementArena) ArenaClearNeeded {
	if s.entered != arena {
		panic("ElementArenaScope::exit called with a different arena than was entered")
	}
	s.exited = true
	s.End()
	return ArenaClearNeeded{arena: arena}
}

// End restores the previously current arena and ends the arena's clear-deferral scope; it is
// upstream's Drop and runs once, on the normal path from exit and with defer on a panic.
func (s *elementArenaScope) End() {
	if s.ended {
		return
	}
	s.ended = true
	currentElementArena.Store(s.previous)
	s.entered.endScope()
	if !s.exited {
		slog.Error("ElementArenaScope dropped without calling exit(); the arena clear for this draw was never requested")
	}
}

// ArenaClearNeeded is returned when the element arena has been used and so must be cleared
// before the next draw.
type ArenaClearNeeded struct {
	arena *elementArena
}

// Clear clears the element arena of the App the draw ran against. If an enclosing draw is still
// in progress, the clear is deferred to that draw's own ArenaClearNeeded.
func (a ArenaClearNeeded) Clear(cx *App) {
	if a.arena != cx.elementArena() {
		panic("ArenaClearNeeded::clear called with a different App than the draw ran against")
	}
	a.arena.clear()
}

func (w *Window) deferArenaDrop(f func()) {
	withElementArena(func(arena *elementArena) struct{} {
		arena.drops = append(arena.drops, f)
		return struct{}{}
	})
}

type focusRef struct {
	refCount atomic.Uint64
	tabIndex int
	tabStop  bool
}

type focusSlot struct {
	version  uint32
	nextFree uint32
	ref      *focusRef
}

type focusMap struct {
	mu             sync.RWMutex
	slots          []focusSlot
	freeHead       uint32
	droppedHandles []FocusID
}

func defaultFocusMap() *focusMap {
	return &focusMap{slots: []focusSlot{{}}, freeHead: 1}
}

func (m *focusMap) insert(ref *focusRef) FocusID {
	if int(m.freeHead) < len(m.slots) {
		idx := m.freeHead
		slot := &m.slots[idx]
		occupied := slot.version | 1
		m.freeHead = slot.nextFree
		slot.ref = ref
		slot.version = occupied
		return FocusID(uint64(occupied)<<32 | uint64(idx))
	}
	idx := uint32(len(m.slots))
	m.slots = append(m.slots, focusSlot{version: 1, ref: ref})
	m.freeHead = idx + 1
	return FocusID(uint64(1)<<32 | uint64(idx))
}

func (m *focusMap) get(id FocusID) (*focusRef, bool) {
	idx := id.idx()
	if int(idx) >= len(m.slots) || idx == 0 {
		return nil, false
	}
	slot := &m.slots[idx]
	if slot.version != uint32(id>>32) || slot.ref == nil {
		return nil, false
	}
	return slot.ref, true
}

func (m *focusMap) removeHandle(id FocusID) {
	if _, ok := m.get(id); !ok {
		return
	}
	idx := id.idx()
	slot := &m.slots[idx]
	slot.ref = nil
	slot.nextFree = m.freeHead
	m.freeHead = idx
	slot.version++
}

// IsFocused reports whether the element associated with this id is currently focused.
func (id FocusID) IsFocused(window *Window) bool {
	focus, ok := window.focus.Get()
	return ok && focus == id
}

// ContainsFocused reports whether the element associated with this id contains the focused
// element or is itself focused.
func (id FocusID) ContainsFocused(window *Window, cx *App) bool {
	focused, ok := window.Focused(cx)
	if !ok {
		return false
	}
	defer focused.Release()
	return id.contains(focused.id, window)
}

// WithinFocused reports whether the element associated with this id is contained within the
// focused element or is itself focused.
func (id FocusID) WithinFocused(window *Window, cx *App) bool {
	focused, ok := window.Focused(cx)
	if !ok {
		return false
	}
	defer focused.Release()
	return focused.id.contains(id, window)
}

func (id FocusID) contains(other FocusID, window *Window) bool {
	return window.renderedFrame.dispatchTree.focusContains(id, other)
}

type focusHandleObject struct {
	released atomic.Bool
	cleanup  runtime.Cleanup
}

type focusHandleCleanup struct {
	id      FocusID
	handles weak.Pointer[focusMap]
}

// FocusHandle is a ref-counted handle used to track and manipulate the focused element in a
// window. Clone makes a new counted handle; Release drops it (D-RAII).
type FocusHandle struct {
	id      FocusID
	handles *focusMap
	object  *focusHandleObject
	// TabIndex is the index of this element in the tab order.
	TabIndex int
	// TabStop reports whether this element can be focused by tab navigation.
	TabStop bool
}

func newFocusHandleObject(id FocusID, handles *focusMap) *focusHandleObject {
	object := &focusHandleObject{}
	object.cleanup = runtime.AddCleanup(object, collectFocusHandle, focusHandleCleanup{id: id, handles: weak.Make(handles)})
	return object
}

func collectFocusHandle(c focusHandleCleanup) {
	if currentLeakDetectionMode() == leakDetectionStrict {
		recordStrictLeak(fmt.Sprintf("Focus handle %#v was collected without Release", c.id))
	}
	if handles := c.handles.Value(); handles != nil {
		releaseFocusRef(c.id, handles)
	}
}

func newFocusHandle(handles *focusMap) FocusHandle {
	ref := &focusRef{}
	ref.refCount.Store(1)
	handles.mu.Lock()
	id := handles.insert(ref)
	handles.mu.Unlock()
	return FocusHandle{
		id:      id,
		handles: handles,
		object:  newFocusHandleObject(id, handles),
	}
}

func focusHandleForID(id FocusID, handles *focusMap) (FocusHandle, bool) {
	handles.mu.RLock()
	focus, ok := handles.get(id)
	if !ok {
		handles.mu.RUnlock()
		return FocusHandle{}, false
	}
	if atomicIncrIfNotZero(&focus.refCount) == 0 {
		handles.mu.RUnlock()
		return FocusHandle{}, false
	}
	tabIndex, tabStop := focus.tabIndex, focus.tabStop
	handles.mu.RUnlock()
	return FocusHandle{
		id:       id,
		handles:  handles,
		object:   newFocusHandleObject(id, handles),
		TabIndex: tabIndex,
		TabStop:  tabStop,
	}, true
}

func (h FocusHandle) checkLive() {
	if h.object != nil && h.object.released.Load() {
		panic("gpui: use of a released FocusHandle")
	}
}

// WithTabIndex sets the tab index of the element associated with this handle (upstream's
// tab_index builder, D-FIELDMETHOD).
func (h FocusHandle) WithTabIndex(index int) FocusHandle {
	h.TabIndex = index
	if h.handles != nil {
		h.handles.mu.Lock()
		if focus, ok := h.handles.get(h.id); ok {
			focus.tabIndex = index
		}
		h.handles.mu.Unlock()
	}
	return h
}

// WithTabStop sets whether the element associated with this handle is a tab stop; when false it
// is not included in the tab order (upstream's tab_stop builder, D-FIELDMETHOD).
func (h FocusHandle) WithTabStop(tabStop bool) FocusHandle {
	h.TabStop = tabStop
	if h.handles != nil {
		h.handles.mu.Lock()
		if focus, ok := h.handles.get(h.id); ok {
			focus.tabStop = tabStop
		}
		h.handles.mu.Unlock()
	}
	return h
}

// Downgrade converts this focus handle into a weak variant, which does not prevent it from being
// released.
func (h FocusHandle) Downgrade() WeakFocusHandle {
	h.checkLive()
	return newWeakFocusHandle(h.id, h.handles)
}

// Focus moves the focus to the element associated with this handle.
func (h FocusHandle) Focus(window *Window, cx *App) {
	window.Focus(h, cx)
}

// IsFocused reports whether the element associated with this handle is currently focused.
func (h FocusHandle) IsFocused(window *Window) bool {
	return h.id.IsFocused(window)
}

// ContainsFocused reports whether the element associated with this handle contains the focused
// element or is itself focused.
func (h FocusHandle) ContainsFocused(window *Window, cx *App) bool {
	return h.id.ContainsFocused(window, cx)
}

// WithinFocused reports whether the element associated with this handle is contained within the
// focused element or is itself focused.
func (h FocusHandle) WithinFocused(window *Window, cx *App) bool {
	return h.id.WithinFocused(window, cx)
}

// Contains reports whether this handle contains other in the most recently rendered frame.
func (h FocusHandle) Contains(other FocusHandle, window *Window) bool {
	return h.id.contains(other.id, window)
}

// IsRendered reports whether the element associated with this handle is part of the most
// recently rendered frame.
func (h FocusHandle) IsRendered(window *Window) bool {
	_, ok := window.renderedFrame.dispatchTree.focusableNodeID(h.id)
	return ok
}

// DispatchAction dispatches an action on the element that rendered this focus handle.
func (h FocusHandle) DispatchAction(action Action, window *Window, cx *App) {
	if nodeID, ok := window.renderedFrame.dispatchTree.focusableNodeID(h.id); ok {
		window.dispatchActionOnNode(nodeID, action, cx)
	}
}

// Clone returns a new counted handle to the same element. A handle built without a focus map
// (a FocusHandle literal) is not counted and is copied.
func (h FocusHandle) Clone() FocusHandle {
	if h.handles == nil {
		return h
	}
	h.checkLive()
	clone, ok := focusHandleForID(h.id, h.handles)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return clone
}

// Equal reports whether both handles refer to the same element.
func (h FocusHandle) Equal(other FocusHandle) bool {
	return h.id == other.id
}

// EqualWeak reports whether h and the weak handle refer to the same element.
func (h FocusHandle) EqualWeak(other WeakFocusHandle) bool {
	return h.id == other.id
}

// GoString formats the handle like upstream's Debug, FocusHandle(FocusId(<idx>v<version>)).
func (h FocusHandle) GoString() string {
	return "FocusHandle(" + h.id.GoString() + ")"
}

// Release drops this handle; the last release queues the id for removal at the next effect
// flush. It is idempotent per handle.
func (h FocusHandle) Release() {
	if h.handles == nil || h.object == nil {
		return
	}
	if h.object.released.Swap(true) {
		return
	}
	h.object.cleanup.Stop()
	releaseFocusRef(h.id, h.handles)
}

func releaseFocusRef(id FocusID, handles *focusMap) {
	handles.mu.Lock()
	defer handles.mu.Unlock()
	focus, ok := handles.get(id)
	if !ok {
		panic("focus handle missing")
	}
	previousCount := focus.refCount.Add(^uint64(0)) + 1
	if previousCount == 0 {
		panic("assertion `left != right` failed: over-released focus handle\n  left: 0\n right: 0")
	}
	if previousCount == 1 {
		handles.droppedHandles = append(handles.droppedHandles, id)
	}
}

// WeakFocusHandle is a weak reference to a focus handle.
type WeakFocusHandle struct {
	id      FocusID
	handles weak.Pointer[focusMap]
}

func newWeakFocusHandle(id FocusID, handles *focusMap) WeakFocusHandle {
	var pointer weak.Pointer[focusMap]
	if handles != nil {
		pointer = weak.Make(handles)
	}
	return WeakFocusHandle{id: id, handles: pointer}
}

// Upgrade attempts to upgrade the weak handle to a FocusHandle the caller must release.
func (w WeakFocusHandle) Upgrade() (FocusHandle, bool) {
	handles := w.handles.Value()
	if handles == nil {
		return FocusHandle{}, false
	}
	return focusHandleForID(w.id, handles)
}

// Equal reports whether both weak handles refer to the same element.
func (w WeakFocusHandle) Equal(other WeakFocusHandle) bool {
	return w.id == other.id
}

// EqualHandle reports whether the weak handle and h refer to the same element.
func (w WeakFocusHandle) EqualHandle(other FocusHandle) bool {
	return w.id == other.id
}

// Focusable is implemented by views that can be focused with window.FocusView.
type Focusable interface {
	// FocusHandle returns the focus handle associated with this view; the caller releases it.
	FocusHandle(cx *App) FocusHandle
}

// FocusHandle returns the focus handle of the entity's state; *T must implement Focusable, it
// panics naming the type otherwise (D-COND).
func (e Entity[T]) FocusHandle(cx *App) FocusHandle {
	focusable, ok := any(e.Read(cx)).(Focusable)
	if !ok {
		panic(fmt.Sprintf("gpui: %s does not implement Focusable", reflect.TypeFor[*T]()))
	}
	return focusable.FocusHandle(cx)
}

// ManagedView is a view (like a modal, popover or menu) whose lifecycle is handled by another
// view; it emits DismissEvent when it should be dismissed (D-EMIT).
type ManagedView[T any] interface {
	Focusable
	Render[T]
}

// DismissEvent is emitted by a ManagedView to indicate the view should be dismissed.
type DismissEvent struct{}

// FrameCallback is a callback run before the next frame of a window.
type FrameCallback func(window *Window, cx *App)

type anyMouseListener func(event any, phase DispatchPhase, window *Window, cx *App)

type cursorStyleRequest struct {
	hitboxID Option[HitboxID]
	style    CursorStyle
}

type hitTest struct {
	ids              []HitboxID
	hoverHitboxCount int
}

func (t hitTest) equal(other hitTest) bool {
	return t.hoverHitboxCount == other.hoverHitboxCount && slices.Equal(t.ids, other.ids)
}

// WindowControlArea is a window control area that corresponds to the platform window.
type WindowControlArea uint8

const (
	// WindowControlAreaDrag allows dragging the platform window.
	WindowControlAreaDrag WindowControlArea = iota
	// WindowControlAreaClose allows closing the platform window.
	WindowControlAreaClose
	// WindowControlAreaMax allows maximizing the platform window.
	WindowControlAreaMax
	// WindowControlAreaMin allows minimizing the platform window.
	WindowControlAreaMin
)

// String formats the area like upstream's Debug.
func (a WindowControlArea) String() string {
	switch a {
	case WindowControlAreaDrag:
		return "Drag"
	case WindowControlAreaClose:
		return "Close"
	case WindowControlAreaMax:
		return "Max"
	case WindowControlAreaMin:
		return "Min"
	}
	return "WindowControlArea(" + strconv.Itoa(int(a)) + ")"
}

// HitboxID identifies a Hitbox within a frame.
type HitboxID uint64

// HitboxIDPlaceholder is a placeholder id for test APIs that need a hitbox whose value does not
// matter.
func HitboxIDPlaceholder() HitboxID {
	return 0
}

// IsHovered reports whether the hitbox with this id is currently hovered. It returns false
// during keyboard input modality, so that keyboard navigation suppresses hover highlights.
func (id HitboxID) IsHovered(window *Window) bool {
	if captured, ok := window.capturedHitbox.Get(); ok && captured == id {
		return true
	}
	if window.LastInputWasKeyboard() {
		return false
	}
	return id.hitTest(window)
}

func (id HitboxID) isHoveredIgnoringLastInput(window *Window) bool {
	if captured, ok := window.capturedHitbox.Get(); ok && captured == id {
		return true
	}
	return id.hitTest(window)
}

func (id HitboxID) hitTest(window *Window) bool {
	test := &window.mouseHitTest
	for _, hovered := range test.ids[:min(test.hoverHitboxCount, len(test.ids))] {
		if id == hovered {
			return true
		}
	}
	return false
}

// ShouldHandleScroll reports whether the hitbox with this id contains the mouse and should
// handle scroll events.
func (id HitboxID) ShouldHandleScroll(window *Window) bool {
	return slices.Contains(window.mouseHitTest.ids, id)
}

func (id HitboxID) next() HitboxID {
	return id + 1
}

// Hitbox is a rectangular region that potentially blocks hitboxes inserted prior; see
// Window.InsertHitbox. It embeds its bounds (upstream's Deref).
type Hitbox struct {
	// ID is a unique identifier for the hitbox.
	ID HitboxID
	Bounds[Pixels]
	// ContentMask is the content mask when the hitbox was inserted.
	ContentMask ContentMask[Pixels]
	// Behavior specifies how the hitbox affects mouse handling.
	Behavior HitboxBehavior
}

// IsHovered reports whether the hitbox is currently hovered; see HitboxID.IsHovered. Scroll
// handling should use ShouldHandleScroll instead.
func (h Hitbox) IsHovered(window *Window) bool {
	return h.ID.IsHovered(window)
}

// IsHoveredAt reports whether this hitbox would be hovered at position, regardless of the
// current input modality or mouse position.
func (h Hitbox) IsHoveredAt(position Point[Pixels], window *Window) bool {
	test := window.renderedFrame.hitTest(position)
	return slices.Contains(test.ids[:min(test.hoverHitboxCount, len(test.ids))], h.ID)
}

// ShouldHandleScroll reports whether the hitbox contains the mouse and should handle scroll
// events.
func (h Hitbox) ShouldHandleScroll(window *Window) bool {
	return h.ID.ShouldHandleScroll(window)
}

// HitboxBehavior is how a hitbox affects mouse behavior.
type HitboxBehavior uint8

const (
	// HitboxBehaviorNormal does not affect mouse handling for other hitboxes.
	HitboxBehaviorNormal HitboxBehavior = iota
	// HitboxBehaviorBlockMouse makes every hitbox behind this one report IsHovered and
	// ShouldHandleScroll false (InteractiveElement.Occlude).
	HitboxBehaviorBlockMouse
	// HitboxBehaviorBlockMouseExceptScroll makes every hitbox behind this one report IsHovered
	// false while ShouldHandleScroll can stay true (InteractiveElement.BlockMouseExceptScroll).
	HitboxBehaviorBlockMouseExceptScroll
)

// String formats the behavior like upstream's Debug.
func (b HitboxBehavior) String() string {
	switch b {
	case HitboxBehaviorNormal:
		return "Normal"
	case HitboxBehaviorBlockMouse:
		return "BlockMouse"
	case HitboxBehaviorBlockMouseExceptScroll:
		return "BlockMouseExceptScroll"
	}
	return "HitboxBehavior(" + strconv.Itoa(int(b)) + ")"
}

// TooltipID identifies a tooltip.
type TooltipID uint64

// IsHovered reports whether the tooltip is currently hovered.
func (id TooltipID) IsHovered(window *Window) bool {
	bounds, ok := window.tooltipBounds.Get()
	return ok && bounds.id == id && bounds.bounds.Contains(window.MousePosition())
}

type tooltipBounds struct {
	id     TooltipID
	bounds Bounds[Pixels]
}

type tooltipRequest struct {
	id      TooltipID
	tooltip AnyTooltip
}

func releaseTooltipRequests(requests []*tooltipRequest) {
	for i, request := range requests {
		requests[i] = nil
		if request != nil {
			request.tooltip.View.Release()
		}
	}
}

type elementIDStack struct {
	ids    []ElementID
	hashes []uint64
}

func (s *elementIDStack) push(id ElementID) {
	s.hashes = append(s.hashes, extendPathHash(s.hash(), id))
	s.ids = append(s.ids, id)
}

func (s *elementIDStack) pop() (ElementID, bool) {
	if len(s.hashes) > 0 {
		s.hashes = s.hashes[:len(s.hashes)-1]
	}
	if len(s.ids) == 0 {
		return ElementID{}, false
	}
	id := s.ids[len(s.ids)-1]
	s.ids = s.ids[:len(s.ids)-1]
	return id, true
}

func (s *elementIDStack) clear() {
	s.ids = s.ids[:0]
	s.hashes = s.hashes[:0]
}

func (s *elementIDStack) hash() uint64 {
	if len(s.hashes) == 0 {
		return emptyPathHash
	}
	return s.hashes[len(s.hashes)-1]
}

func (s *elementIDStack) globalID() GlobalElementID {
	return globalElementIDWithHash(s.ids, s.hash())
}

func (s elementIDStack) clone() elementIDStack {
	return elementIDStack{ids: slices.Clone(s.ids), hashes: slices.Clone(s.hashes)}
}

type deferredDraw struct {
	currentView      EntityID
	priority         int
	parentNode       dispatchNodeID
	elementIDStack   elementIDStack
	textStyleStack   []TextStyleRefinement
	contentMask      Option[ContentMask[Pixels]]
	remSize          Pixels
	element          Option[AnyElement]
	absoluteOffset   Point[Pixels]
	prepaintRange    prepaintStateRange
	paintRange       paintRange
	accessedEntities collections.HashSet[EntityID]
}

type elementStateKey struct {
	id  GlobalElementID
	typ reflect.Type
}

type elementStateBucket struct {
	hash uint64
	typ  reflect.Type
}

type elementStateEntry struct {
	key   elementStateKey
	state elementStateBox
}

type elementStateMap struct {
	buckets map[elementStateBucket][]elementStateEntry
	count   int
}

func (m *elementStateMap) bucket(key elementStateKey) elementStateBucket {
	return elementStateBucket{hash: key.id.hash, typ: key.typ}
}

func (m *elementStateMap) get(key elementStateKey) (*elementStateBox, bool) {
	entries := m.buckets[m.bucket(key)]
	for i := range entries {
		if entries[i].key.id.Equal(key.id) {
			return &entries[i].state, true
		}
	}
	return nil, false
}

func (m *elementStateMap) insert(key elementStateKey, state elementStateBox) {
	if m.buckets == nil {
		m.buckets = make(map[elementStateBucket][]elementStateEntry)
	}
	b := m.bucket(key)
	entries := m.buckets[b]
	for i := range entries {
		if entries[i].key.id.Equal(key.id) {
			old := entries[i].state
			entries[i].state = state
			if old.inner != state.inner {
				old.release()
			}
			return
		}
	}
	m.buckets[b] = append(entries, elementStateEntry{key: key, state: state})
	m.count++
}

func (m *elementStateMap) removeEntry(key elementStateKey) (elementStateKey, elementStateBox, bool) {
	b := m.bucket(key)
	entries := m.buckets[b]
	for i := range entries {
		if entries[i].key.id.Equal(key.id) {
			entry := entries[i]
			entries = slices.Delete(entries, i, i+1)
			if len(entries) == 0 {
				delete(m.buckets, b)
			} else {
				m.buckets[b] = entries
			}
			m.count--
			return entry.key, entry.state, true
		}
	}
	return elementStateKey{}, elementStateBox{}, false
}

func (m *elementStateMap) len() int {
	return m.count
}

func (m *elementStateMap) clear() {
	buckets := m.buckets
	m.buckets = nil
	m.count = 0
	for _, entries := range buckets {
		for _, entry := range entries {
			entry.state.release()
		}
	}
}

type windowControlHitbox struct {
	area   WindowControlArea
	hitbox Hitbox
}

type debugBoundsRecord struct {
	selector string
	bounds   Bounds[Pixels]
}

type inspectorInstanceIDs struct {
	path InspectorElementPath
	next int
}

type frame struct {
	focus                    Option[FocusID]
	windowActive             bool
	elementStates            elementStateMap
	accessedElementStates    []elementStateKey
	mouseListeners           []anyMouseListener
	dispatchTree             *dispatchTree
	scene                    Scene
	hitboxes                 []Hitbox
	windowControlHitboxes    []windowControlHitbox
	deferredDraws            []deferredDraw
	inputHandlers            []*PlatformInputHandler
	tooltipRequests          []*tooltipRequest
	cursorStyles             []cursorStyleRequest
	debugBounds              map[string]Bounds[Pixels]
	debugBoundsRecords       []debugBoundsRecord
	nextInspectorInstanceIDs map[uint64][]inspectorInstanceIDs
	inspectorHitboxes        map[HitboxID]InspectorElementID
	tabStops                 tabStopMap
}

type prepaintStateIndex struct {
	hitboxesIndex              int
	tooltipsIndex              int
	deferredDrawsIndex         int
	dispatchTreeIndex          int
	accessedElementStatesIndex int
	lineLayoutIndex            LineLayoutIndex
}

type paintIndex struct {
	sceneIndex                 int
	debugBoundsIndex           int
	mouseListenersIndex        int
	inputHandlersIndex         int
	cursorStylesIndex          int
	accessedElementStatesIndex int
	tabHandleIndex             int
	lineLayoutIndex            LineLayoutIndex
}

func (f *frame) recordDebugBounds(selector string, bounds Bounds[Pixels]) {
	f.debugBounds[selector] = bounds
	f.debugBoundsRecords = append(f.debugBoundsRecords, debugBoundsRecord{selector: selector, bounds: bounds})
}

func newFrame(dispatchTree *dispatchTree) *frame {
	return &frame{
		dispatchTree: dispatchTree,
		debugBounds:  make(map[string]Bounds[Pixels]),
	}
}

func (f *frame) clear() {
	f.elementStates.clear()
	clear(f.accessedElementStates)
	f.accessedElementStates = f.accessedElementStates[:0]
	clear(f.mouseListeners)
	f.mouseListeners = f.mouseListeners[:0]
	f.dispatchTree.clear()
	f.scene.Clear()
	for i, inputHandler := range f.inputHandlers {
		f.inputHandlers[i] = nil
		if inputHandler != nil {
			inputHandler.release()
		}
	}
	f.inputHandlers = f.inputHandlers[:0]
	releaseTooltipRequests(f.tooltipRequests)
	f.tooltipRequests = f.tooltipRequests[:0]
	f.cursorStyles = f.cursorStyles[:0]
	f.hitboxes = f.hitboxes[:0]
	f.windowControlHitboxes = f.windowControlHitboxes[:0]
	clear(f.deferredDraws)
	f.deferredDraws = f.deferredDraws[:0]
	f.tabStops.clear()
	f.focus = None[FocusID]()

	clear(f.debugBounds)
	f.debugBoundsRecords = f.debugBoundsRecords[:0]

	clear(f.nextInspectorInstanceIDs)
	clear(f.inspectorHitboxes)
}

func (f *frame) cursorStyle(window *Window) (CursorStyle, bool) {
	var style Option[CursorStyle]
	for i := len(f.cursorStyles) - 1; i >= 0; i-- {
		request := f.cursorStyles[i]
		hitboxID, ok := request.hitboxID.Get()
		if !ok {
			return request.style, true
		}
		if style.IsNone() && hitboxID.isHoveredIgnoringLastInput(window) {
			style = Some(request.style)
		}
	}
	return style.Get()
}

func (f *frame) hitTest(position Point[Pixels]) hitTest {
	setHoverHitboxCount := false
	var test hitTest
	for i := len(f.hitboxes) - 1; i >= 0; i-- {
		hitbox := &f.hitboxes[i]
		bounds := hitbox.Bounds.Intersect(hitbox.ContentMask.Bounds)
		if bounds.Contains(position) {
			test.ids = append(test.ids, hitbox.ID)
			if !setHoverHitboxCount && hitbox.Behavior == HitboxBehaviorBlockMouseExceptScroll {
				test.hoverHitboxCount = len(test.ids)
				setHoverHitboxCount = true
			}
			if hitbox.Behavior == HitboxBehaviorBlockMouse {
				break
			}
		}
	}
	if !setHoverHitboxCount {
		test.hoverHitboxCount = len(test.ids)
	}
	return test
}

func (f *frame) focusPath() []FocusID {
	if focusID, ok := f.focus.Get(); ok {
		return f.dispatchTree.focusPath(focusID)
	}
	return nil
}

func (f *frame) finish(prevFrame *frame) {
	for _, key := range f.accessedElementStates {
		if key, state, ok := prevFrame.elementStates.removeEntry(key); ok {
			f.elementStates.insert(key, state)
		}
	}
	f.scene.Finish()
}

type inputModality uint8

const (
	inputModalityMouse inputModality = iota
	inputModalityKeyboard
	inputModalityTouch
)

// Window holds the state for a specific window.
type Window struct {
	closeA11yChannels          func()
	handle                     AnyWindowHandle
	invalidator                windowInvalidator
	removed                    bool
	platformWindow             PlatformWindow
	displayID                  Option[DisplayID]
	displayObservers           singleSubscriberSet[anyObserver]
	isResizable                bool
	isMinimizable              bool
	lastWindowBounds           WindowBounds
	spriteAtlas                PlatformAtlas
	textSystem                 *WindowTextSystem
	textRenderingMode          *TextRenderingMode
	remSize                    Pixels
	remSizeOverrideStack       []Pixels
	viewportSize               Size[Pixels]
	layoutEngine               *TaffyLayoutEngine
	root                       Option[AnyView]
	elementIDStack             elementIDStack
	textStyleStack             []TextStyleRefinement
	renderedEntityStack        []EntityID
	elementOffsetStack         []Point[Pixels]
	currentElementOpacity      float32
	contentMaskStack           []ContentMask[Pixels]
	requestedAutoscroll        Option[Bounds[Pixels]]
	lastTextInputConfiguration Option[TextInputConfiguration]
	focusedTextInput           Option[FocusID]
	imageCacheStack            []AnyImageCache
	renderedFrame              *frame
	nextFrame                  *frame
	nextHitboxID               HitboxID
	nativeViews                []weak.Pointer[nativeViewState]
	nextNativeViewID           uint64
	nextTooltipID              TooltipID
	tooltipBounds              Option[tooltipBounds]
	nextFrameCallbacks         *[]FrameCallback
	dirtyViews                 map[EntityID]struct{}
	focusListeners             singleSubscriberSet[anyWindowFocusListener]
	focusLostListeners         singleSubscriberSet[anyObserver]
	focusLostPath              []FocusID
	defaultPrevented           bool
	mousePosition              Point[Pixels]
	mouseHitTest               hitTest
	modifiers                  Modifiers
	capslock                   Capslock
	scaleFactor                float32
	boundsObservers            singleSubscriberSet[anyObserver]
	appearance                 WindowAppearance
	appearanceObservers        singleSubscriberSet[anyObserver]
	buttonLayoutObservers      singleSubscriberSet[anyObserver]
	active                     *bool
	visibility                 WindowVisibility
	visibilityObservers        singleSubscriberSet[func(visibility WindowVisibility, window *Window, cx *App) bool]
	hovered                    *bool
	needsPresent               *bool
	sceneAnimationInterval     *Option[time.Duration]
	inputRateTracker           *inputRateTracker
	windowProfiler             *WindowProfiler
	lastInputModality          inputModality
	touchGestures              *touchGestureRecognizer
	touchPredictionEnabled     bool
	longPressTimer             *Task[struct{}]
	longPressCapture           Option[EntityID]
	refreshing                 bool
	activationObservers        singleSubscriberSet[anyObserver]
	focus                      Option[FocusID]
	focusEnabled               bool
	focusGeneration            uint64
	pendingInput               *pendingInput
	pendingModifier            modifierState
	pendingInputObservers      singleSubscriberSet[anyObserver]
	prompt                     Option[RenderablePromptHandle]
	clientInset                Option[Pixels]
	capturedHitbox             Option[HitboxID]
	inspector                  Option[Entity[Inspector]]
	debugFrameOverlay          DebugFrameOverlay
	a11y                       windowA11y
}

type modifierState struct {
	modifiers     Modifiers
	sawOtherInput bool
}

type inputRateTracker struct {
	timestamps      []scheduler.Instant
	window          time.Duration
	inputsPerSecond uint32
	sustainUntil    scheduler.Instant
	sustainDuration time.Duration
}

const frameIntervalTolerance = time.Millisecond

type framePacer struct {
	lastFrame Option[scheduler.Instant]
}

func (p *framePacer) shouldDefer(now scheduler.Instant, minimumInterval Option[time.Duration]) bool {
	interval, hasInterval := minimumInterval.Get()
	lastFrame, hasLastFrame := p.lastFrame.Get()
	if hasInterval && hasLastFrame {
		deadline := lastFrame.Add(interval)
		if now.Add(frameIntervalTolerance).Before(deadline) {
			return true
		}
		if now.SaturatingDurationSince(deadline) <= interval/4 {
			p.lastFrame = Some(deadline)
		} else {
			p.lastFrame = Some(now)
		}
	} else {
		p.lastFrame = Some(now)
	}
	return false
}

func maxDurationOption(a, b Option[time.Duration]) Option[time.Duration] {
	av, aok := a.Get()
	bv, bok := b.Get()
	switch {
	case !aok:
		return b
	case !bok:
		return a
	case bv >= av:
		return b
	default:
		return a
	}
}

func minimumFrameInterval(shouldThrottle, inactiveWithoutHighRateInput bool, inactiveFrameInterval, lowPowerFrameInterval, thermalFrameInterval Option[time.Duration], thermalState ThermalState, powerMode PowerMode) Option[time.Duration] {
	if !shouldThrottle {
		return None[time.Duration]()
	}
	result := None[time.Duration]()
	if inactiveWithoutHighRateInput {
		result = maxDurationOption(result, inactiveFrameInterval)
	}
	if powerMode == PowerModeLowPower {
		result = maxDurationOption(result, lowPowerFrameInterval)
	}
	if thermalState == ThermalStateCritical || thermalState == ThermalStateSerious {
		result = maxDurationOption(result, thermalFrameInterval)
	}
	return result
}

func defaultInputRateTracker() *inputRateTracker {
	return &inputRateTracker{
		window:          100 * time.Millisecond,
		inputsPerSecond: 60,
		sustainUntil:    scheduler.InstantNow(),
		sustainDuration: time.Second,
	}
}

func (t *inputRateTracker) recordInput() {
	now := scheduler.InstantNow()
	t.timestamps = append(t.timestamps, now)
	t.pruneOldTimestamps(now)

	minEvents := uint64(t.inputsPerSecond) * uint64(t.window.Milliseconds()) / 1000
	if uint64(len(t.timestamps)) >= minEvents {
		t.sustainUntil = now.Add(t.sustainDuration)
	}
}

func (t *inputRateTracker) isHighRate() bool {
	return scheduler.InstantNow().Before(t.sustainUntil)
}

func (t *inputRateTracker) pruneOldTimestamps(now scheduler.Instant) {
	t.timestamps = slices.DeleteFunc(t.timestamps, func(ts scheduler.Instant) bool {
		return now.SaturatingDurationSince(ts) > t.window
	})
}

type drawPhase uint8

const (
	drawPhaseNone drawPhase = iota
	drawPhasePrepaint
	drawPhasePaint
	drawPhaseFocus
)

const pendingInputTimeoutDuration = time.Second

// PendingInputStatus is the pending input for a potential multi-stroke key binding.
type PendingInputStatus struct {
	keystrokes []Keystroke
	timeout    Option[PendingInputTimeoutStatus]
}

// Keystrokes returns the keystrokes entered so far; the slice is read-only.
func (s PendingInputStatus) Keystrokes() []Keystroke {
	return s.keystrokes
}

// Timeout returns the timeout state for flushing this input, if it needs a timeout.
func (s PendingInputStatus) Timeout() (PendingInputTimeoutStatus, bool) {
	return s.timeout.Get()
}

// PendingInputTimeoutStatus is the timeout state for pending input.
type PendingInputTimeoutStatus struct {
	duration  time.Duration
	remaining time.Duration
	startedAt Option[scheduler.Instant]
	paused    bool
}

// Duration returns the full timeout duration.
func (s PendingInputTimeoutStatus) Duration() time.Duration {
	return s.duration
}

// Remaining returns the duration remaining before pending input is flushed.
func (s PendingInputTimeoutStatus) Remaining(cx *App) time.Duration {
	if startedAt, ok := s.startedAt.Get(); ok {
		elapsed := cx.BackgroundExecutor().Now().Sub(startedAt)
		if elapsed >= s.remaining {
			return 0
		}
		return s.remaining - elapsed
	}
	return s.remaining
}

// IsPaused reports whether the timeout is paused.
func (s PendingInputTimeoutStatus) IsPaused() bool {
	return s.paused
}

type pendingInputTimeout struct {
	duration  time.Duration
	remaining time.Duration
	state     pendingInputTimeoutState
}

type pendingInputTimeoutState interface {
	isPendingInputTimeoutState()
}

type pendingInputTimeoutRunning struct {
	startedAt scheduler.Instant
	task      *Task[struct{}]
}

type pendingInputTimeoutPaused struct {
	pause pendingInputTimeoutPause
}

func (pendingInputTimeoutRunning) isPendingInputTimeoutState() {}

func (pendingInputTimeoutPaused) isPendingInputTimeoutState() {}

type pendingInputTimeoutPause struct {
	ownerID             EntityID
	releaseSubscription Subscription
}

func saturatingSubDuration(a, b time.Duration) time.Duration {
	if b >= a {
		return 0
	}
	return a - b
}

func (t *pendingInputTimeout) isPaused() bool {
	_, paused := t.state.(pendingInputTimeoutPaused)
	return paused
}

func (t *pendingInputTimeout) pause(pause pendingInputTimeoutPause, now scheduler.Instant) bool {
	switch state := t.state.(type) {
	case pendingInputTimeoutRunning:
		t.state = pendingInputTimeoutPaused{pause: pause}
		t.remaining = saturatingSubDuration(t.remaining, now.Sub(state.startedAt))
		state.task.Cancel()
		return true
	default:
		pause.releaseSubscription.Unsubscribe()
		return false
	}
}

func (t *pendingInputTimeout) pauseOwnerID() (EntityID, bool) {
	if paused, ok := t.state.(pendingInputTimeoutPaused); ok {
		return paused.pause.ownerID, true
	}
	return 0, false
}

func (t *pendingInputTimeout) resume(ownerID EntityID, startedAt scheduler.Instant, task *Task[struct{}]) bool {
	if paused, ok := t.state.(pendingInputTimeoutPaused); ok && paused.pause.ownerID == ownerID {
		t.state = pendingInputTimeoutRunning{startedAt: startedAt, task: task}
		paused.pause.releaseSubscription.Unsubscribe()
		return true
	}
	task.Cancel()
	return false
}

func (t *pendingInputTimeout) resetDuration(duration time.Duration) {
	t.duration = duration
	t.remaining = duration
}

func (t *pendingInputTimeout) status() PendingInputTimeoutStatus {
	startedAt, paused := None[scheduler.Instant](), true
	if running, ok := t.state.(pendingInputTimeoutRunning); ok {
		startedAt, paused = Some(running.startedAt), false
	}
	return PendingInputTimeoutStatus{
		duration:  t.duration,
		remaining: t.remaining,
		startedAt: startedAt,
		paused:    paused,
	}
}

func (t *pendingInputTimeout) release() {
	switch state := t.state.(type) {
	case pendingInputTimeoutRunning:
		state.task.Cancel()
	case pendingInputTimeoutPaused:
		state.pause.releaseSubscription.Unsubscribe()
	}
}

type pendingInput struct {
	keystrokes []Keystroke
	focus      Option[FocusID]
	timeout    *pendingInputTimeout
}

type elementStateBox struct {
	inner    any
	typeName string
	drop     func()
}

type elementStateDropper interface {
	dropElementState()
}

func newElementStateBox[S any](state *Option[S], typeName string) elementStateBox {
	return elementStateBox{inner: state, typeName: typeName, drop: func() {
		if value, ok := state.Take().Get(); ok {
			if dropper, ok := any(&value).(elementStateDropper); ok {
				dropper.dropElementState()
			}
		}
	}}
}

func (b elementStateBox) release() {
	if b.drop != nil {
		b.drop()
	}
}

func defaultBounds(activeWindowBounds Option[WindowBounds], display PlatformDisplay) WindowBounds {
	const cascadeOffset float32 = 25.0

	defaultPlacement := func() Bounds[Pixels] {
		return NewBounds(NewPoint(Px(0.), Px(0.)), DefaultWindowSize)
	}

	var displayBounds Bounds[Pixels]
	if display != nil {
		displayBounds = display.VisibleBounds()
	} else {
		displayBounds = defaultPlacement()
	}

	var base Bounds[Pixels]
	windowBoundsCtor := WindowBoundsWindowed
	if bounds, ok := activeWindowBounds.Get(); ok {
		base = bounds.GetBounds()
		switch bounds.Kind() {
		case WindowBoundsKindMaximized:
			windowBoundsCtor = WindowBoundsMaximized
		case WindowBoundsKindFullscreen:
			windowBoundsCtor = WindowBoundsFullscreen
		}
	} else if display != nil {
		base = display.DefaultBounds()
	} else {
		base = defaultPlacement()
	}
	baseOrigin, baseSize := base.Origin, base.Size

	offset := NewPoint(Px(cascadeOffset), Px(cascadeOffset))
	proposedOrigin := baseOrigin.Add(offset)
	proposedBounds := NewBounds(proposedOrigin, baseSize)

	displayRight := displayBounds.Origin.X + displayBounds.Size.Width
	displayBottom := displayBounds.Origin.Y + displayBounds.Size.Height
	windowRight := proposedBounds.Origin.X + proposedBounds.Size.Width
	windowBottom := proposedBounds.Origin.Y + proposedBounds.Size.Height

	fitsHorizontally := windowRight <= displayRight
	fitsVertically := windowBottom <= displayBottom

	var finalOrigin Point[Pixels]
	switch {
	case fitsHorizontally && fitsVertically:
		finalOrigin = proposedOrigin
	case !fitsHorizontally && fitsVertically:
		finalOrigin = NewPoint(displayBounds.Origin.X, baseOrigin.Y)
	case fitsHorizontally && !fitsVertically:
		finalOrigin = NewPoint(baseOrigin.X, displayBounds.Origin.Y)
	default:
		finalOrigin = displayBounds.Origin
	}
	return windowBoundsCtor(NewBounds(finalOrigin, baseSize))
}

func logErrContext(err error, format string, args ...any) {
	if err != nil {
		gpuiutil.LogErr(fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err))
	}
}

func cloneTreeUpdate(update a11y.TreeUpdate) a11y.TreeUpdate {
	nodes := make([]a11y.NodeEntry, len(update.Nodes))
	for i, entry := range update.Nodes {
		nodes[i] = a11y.NodeEntry{ID: entry.ID, Node: entry.Node.Clone()}
	}
	update.Nodes = nodes
	return update
}

func newWindow(handle AnyWindowHandle, options WindowOptions, cx *App) (*Window, error) {
	windowBoundsOption := options.WindowBounds
	titlebar := options.Titlebar
	focus := options.Focus
	show := options.Show
	kind := options.Kind
	isMovable := options.IsMovable
	appOwnsTitlebarDrag := options.AppOwnsTitlebarDrag
	inactiveFrameInterval := options.InactiveFrameInterval
	lowPowerFrameInterval := options.LowPowerFrameInterval
	thermalFrameInterval := options.ThermalFrameInterval
	isResizable := options.IsResizable
	isMinimizable := options.IsMinimizable
	displayIDOption := options.DisplayID
	windowBackground := options.WindowBackground
	appID := options.AppID
	windowMinSize := options.WindowMinSize
	windowDecorations := options.WindowDecorations
	icon := options.Icon
	tabbingIdentifier := options.TabbingIdentifier
	useDisplayP3 := options.UseDisplayP3

	initialWindowTitle := None[SharedString]()
	if t, ok := titlebar.Get(); ok {
		initialWindowTitle = t.Title
	}

	activeDisplayID, activeBounds, hasActivePlacement := cx.activeWindowPlacement()
	targetDisplayID := displayIDOption
	if targetDisplayID.IsNone() && hasActivePlacement && windowBoundsOption.IsNone() {
		targetDisplayID = activeDisplayID
	}
	var display PlatformDisplay
	if id, ok := targetDisplayID.Get(); ok {
		display = cx.FindDisplay(id)
	}
	if display == nil {
		display = cx.PrimaryDisplay()
	}
	displayID := targetDisplayID
	if display != nil {
		displayID = Some(display.ID())
	}
	activeWindowBounds := None[WindowBounds]()
	if hasActivePlacement && activeDisplayID == displayID {
		activeWindowBounds = Some(activeBounds)
	}
	windowBounds, ok := windowBoundsOption.Get()
	if !ok {
		windowBounds = defaultBounds(activeWindowBounds, display)
	}
	platformWindow, err := cx.platform.OpenWindow(handle, WindowParams{
		Bounds:              windowBounds.GetBounds(),
		Titlebar:            titlebar,
		Kind:                kind,
		IsMovable:           isMovable,
		AppOwnsTitlebarDrag: appOwnsTitlebarDrag,
		IsResizable:         isResizable,
		IsMinimizable:       isMinimizable,
		Focus:               focus,
		Show:                show,
		DisplayID:           displayID,
		WindowMinSize:       windowMinSize,
		AppID:               appID,
		Icon:                icon,
		TabbingIdentifier:   tabbingIdentifier,
		UseDisplayP3:        useDisplayP3,
	})
	if err != nil {
		return nil, err
	}

	tabBarVisible := platformWindow.TabBarVisible()
	SystemWindowTabControllerInitVisible(cx, tabBarVisible)
	if tabs, ok := platformWindow.TabbedWindows(); ok {
		SystemWindowTabControllerAddTab(cx, handle.WindowID(), tabs)
	}

	displayID = None[DisplayID]()
	if d := platformWindow.Display(); d != nil {
		displayID = Some(d.ID())
	}
	spriteAtlas := platformWindow.SpriteAtlas()
	mousePosition := platformWindow.MousePosition()
	modifiers := platformWindow.Modifiers()
	capslock := platformWindow.Capslock()
	contentSize := platformWindow.ContentSize()
	scaleFactor := platformWindow.ScaleFactor()
	appearance := platformWindow.Appearance()
	textSystem := NewWindowTextSystem(cx.TextSystem())
	invalidator := newWindowInvalidator(handle.WindowID())
	active := new(bool)
	*active = platformWindow.IsActive()
	visibility := platformWindow.Visibility()
	if profilerFeatureEnabled {
		recordWindowVisibility(handle.WindowID(), visibility)
	}
	hovered := new(bool)
	*hovered = platformWindow.IsHovered()
	needsPresent := new(bool)
	sceneAnimationInterval := new(Option[time.Duration])
	nextFrameCallbacks := new([]FrameCallback)
	inputRateTracker := defaultInputRateTracker()

	platformWindow.RequestDecorations(windowDecorations.UnwrapOr(WindowDecorationsServer))
	platformWindow.SetBackgroundAppearance(windowBackground)

	switch windowBounds.Kind() {
	case WindowBoundsKindFullscreen:
		platformWindow.ToggleFullscreen()
	case WindowBoundsKindMaximized:
		platformWindow.Zoom()
	}

	accessibilityForceDisabled := cx.accessibilityForceDisabled
	a11yActiveFlag := new(atomic.Bool)
	var closeA11yChannels func()

	if runtime.GOARCH != "wasm" && !accessibilityForceDisabled {
		initialRootNode := a11y.NewNode(a11y.RoleWindow)
		if title, ok := initialWindowTitle.Get(); ok {
			initialRootNode.SetLabel(string(title))
		}
		initialTree := a11y.TreeUpdate{
			Nodes:  []a11y.NodeEntry{{ID: rootNodeID, Node: initialRootNode}},
			Tree:   Some(a11y.NewTree(rootNodeID)),
			TreeID: a11y.TreeIDRoot(),
			Focus:  rootNodeID,
		}
		activationSender, activationReceiver := scheduler.Chan[struct{}](0)
		deactivationSender, deactivationReceiver := scheduler.Chan[struct{}](0)
		actionSender, actionReceiver := scheduler.Chan[a11y.ActionRequest](0)
		closeA11yChannels = func() {
			activationSender.Close()
			deactivationSender.Close()
			actionSender.Close()
		}

		platformWindow.A11yInit(A11yCallbacks{
			Activation: func() (a11y.TreeUpdate, bool) {
				slog.Info("Accessibility activated")
				a11yActiveFlag.Store(true)
				logErrContext(activationSender.TrySend(struct{}{}), "queuing accessibility activation for window %#v", handle.WindowID())
				return cloneTreeUpdate(initialTree), true
			},
			Action: func(request a11y.ActionRequest) {
				logErrContext(actionSender.TrySend(request), "queuing accessibility action for window %#v", handle.WindowID())
			},
			Deactivation: func() {
				slog.Info("Accessibility deactivated")
				a11yActiveFlag.Store(false)
				logErrContext(deactivationSender.TrySend(struct{}{}), "queuing accessibility deactivation for window %#v", handle.WindowID())
			},
		})

		activationCx := cx.ToAsync()
		cx.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
			for {
				if _, ok := activationReceiver.Recv(s); !ok {
					return struct{}{}, nil
				}
				logErrContext(handle.Update(activationCx, func(_ AnyView, window *Window, _ *App) { window.Refresh() }), "handling accessibility activation for window %#v", handle.WindowID())
			}
		}).Detach()

		deactivationCx := cx.ToAsync()
		cx.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
			for {
				if _, ok := deactivationReceiver.Recv(s); !ok {
					return struct{}{}, nil
				}
				logErrContext(handle.Update(deactivationCx, func(_ AnyView, window *Window, _ *App) { window.Refresh() }), "handling accessibility deactivation for window %#v", handle.WindowID())
			}
		}).Detach()

		actionCx := cx.ToAsync()
		cx.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
			for {
				request, ok := actionReceiver.Recv(s)
				if !ok {
					return struct{}{}, nil
				}
				logErrContext(handle.Update(actionCx, func(_ AnyView, window *Window, cx *App) { window.handleA11yAction(request, cx) }), "handling accessibility action for window %#v", handle.WindowID())
			}
		}).Detach()
	}

	{
		windowID := handle.WindowID()
		closeCx := cx.ToAsync()
		platformWindow.OnClose(func() {
			closeCx.Spawn(func(cx *AsyncApp) (struct{}, error) {
				cx.Update(func(cx *App) {
					if cx.windows.containsKey(windowID) {
						logErrContext(handle.Update(cx, func(_ AnyView, window *Window, _ *App) { window.RemoveWindow() }), "handling close for window %#v", windowID)
					}
					SystemWindowTabControllerRemoveTab(cx, windowID)
				})
				return struct{}{}, nil
			}).Detach()
		})
	}
	{
		frameCx := cx.ToAsync()
		deferredForceRender := false
		var pacer framePacer
		platformWindow.OnRequestFrame(func(requestFrameOptions RequestFrameOptions) {
			windowID := handle.WindowID()
			if profilerFeatureEnabled {
				beginForegroundTurn()
				defer endForegroundTurn()
				recordPlatformSignal(PlatformSignal{
					WindowID:  windowID,
					SignalAt:  requestFrameOptions.SignalAt,
					HandledAt: scheduler.InstantNow(),
					Source:    requestFrameOptions.SignalSource,
				})
			}
			if drawInProgress() || frameCx.appBorrowed() {
				slog.Debug("deferring re-entrant window draw request")
				deferredForceRender = deferredForceRender || requestFrameOptions.ForceRender
				if profilerFeatureEnabled {
					recordReentrantFrameSkipped(windowID, scheduler.InstantNow())
				}
				return
			}
			forceRender := deferredForceRender || requestFrameOptions.ForceRender
			deferredForceRender = false

			type powerState struct {
				thermalState ThermalState
				powerMode    PowerMode
			}
			state, err := handle.UpdateR(frameCx, func(_ AnyView, _ *Window, cx *App) powerState {
				return powerState{cx.ThermalState(), cx.PowerMode()}
			})
			logErrContext(err, "reading power state for frame request for window %#v", handle.WindowID())

			highRateInput := inputRateTracker.isHighRate()
			hasNextFrameCallbacks := len(*nextFrameCallbacks) != 0
			hasDemandBesidesAnimation := invalidator.isDirty() ||
				forceRender ||
				hasNextFrameCallbacks ||
				requestFrameOptions.RequirePresentation ||
				*needsPresent ||
				highRateInput
			hasFrameDemand := hasDemandBesidesAnimation || sceneAnimationInterval.IsSome()

			shouldThrottle := !requestFrameOptions.RequirePresentation
			inactiveWithoutHighRateInput := shouldThrottle && !*active && !highRateInput
			thermalState, powerMode := ThermalStateNominal, PowerModeUnknown
			if err == nil {
				thermalState, powerMode = state.thermalState, state.powerMode
			}
			minFrameInterval := minimumFrameInterval(
				shouldThrottle,
				inactiveWithoutHighRateInput,
				inactiveFrameInterval,
				lowPowerFrameInterval,
				thermalFrameInterval,
				thermalState,
				powerMode,
			)
			animationFrameInterval := None[time.Duration]()
			if interval, ok := sceneAnimationInterval.Get(); ok && !hasDemandBesidesAnimation && interval != 0 {
				animationFrameInterval = Some(interval)
			}
			minFrameInterval = maxDurationOption(minFrameInterval, animationFrameInterval)

			now := scheduler.InstantNow()
			if hasFrameDemand && pacer.shouldDefer(now, minFrameInterval) {
				deferredForceRender = deferredForceRender || forceRender
				logErrContext(handle.Update(frameCx, func(_ AnyView, window *Window, _ *App) {
					window.platformWindow.ScheduleFrame()
				}), "rescheduling throttled frame for window %#v", handle.WindowID())
				invalidator.wakePlatform()
				if profilerFeatureEnabled {
					var reason FrameSkipReason
					switch {
					case animationFrameInterval.IsSome() && animationFrameInterval == minFrameInterval:
						reason = FrameSkipReasonAnimationFrameRateLimit
					case inactiveWithoutHighRateInput && inactiveFrameInterval == minFrameInterval:
						reason = FrameSkipReasonInactiveFrameRateLimit
					case powerMode == PowerModeLowPower && lowPowerFrameInterval == minFrameInterval:
						reason = FrameSkipReasonLowPowerFrameRateLimit
					default:
						reason = FrameSkipReasonThermalFrameRateLimit
					}
					recordFrameSkipped(windowID, scheduler.InstantNow(), reason)
				}
				return
			}

			pendingNextFrameCallbacks := *nextFrameCallbacks
			*nextFrameCallbacks = nil
			if len(pendingNextFrameCallbacks) != 0 {
				logErrContext(handle.Update(frameCx, func(_ AnyView, window *Window, cx *App) {
					for _, callback := range pendingNextFrameCallbacks {
						callback(window, cx)
					}
				}), "running next-frame callbacks for window %#v", handle.WindowID())
			}

			shouldDraw := invalidator.isDirty() || forceRender
			needsPresentNow := requestFrameOptions.RequirePresentation ||
				*needsPresent ||
				sceneAnimationInterval.IsSome() ||
				highRateInput
			if shouldDraw {
				gpuiutil.Measure("frame duration", func() struct{} {
					logErrContext(handle.Update(frameCx, func(_ AnyView, window *Window, cx *App) {
						if profilerFeatureEnabled {
							window.windowProfiler.RecordFrameSignal(requestFrameOptions.SignalAt)
						}
						if forceRender {
							window.Refresh()
						}
						arenaClearNeeded := window.Draw(cx)
						window.advanceSceneAnimations(cx)
						window.present()
						arenaClearNeeded.Clear(cx)
					}), "drawing and presenting frame for window %#v", handle.WindowID())
					return struct{}{}
				})
			} else if needsPresentNow {
				logErrContext(handle.Update(frameCx, func(_ AnyView, window *Window, cx *App) {
					window.advanceSceneAnimations(cx)
					window.present()
				}), "presenting frame for window %#v", handle.WindowID())
			}

			logErrContext(handle.Update(frameCx, func(_ AnyView, window *Window, _ *App) {
				if window.invalidator.isDirty() ||
					len(*window.nextFrameCallbacks) != 0 ||
					window.sceneAnimationInterval.IsSome() {
					window.platformWindow.ScheduleFrame()
				}
			}), "scheduling follow-up frame for window %#v", handle.WindowID())

			if invalidator.isDirty() ||
				len(*nextFrameCallbacks) != 0 ||
				sceneAnimationInterval.IsSome() {
				invalidator.wakePlatform()
			}
			if profilerFeatureEnabled && !shouldDraw {
				recordFrameSkipped(windowID, scheduler.InstantNow(), FrameSkipReasonNoRenderNeeded)
			}
		})
	}
	invalidator.setPlatformWaker(platformWindow.FrameWaker())
	{
		viewportCx := cx.ToAsync()
		platformWindow.OnVisualViewportChanged(func() {
			logErrContext(handle.Update(viewportCx, func(_ AnyView, window *Window, _ *App) { window.Refresh() }), "handling visual viewport change for window %#v", handle.WindowID())
		})
	}
	{
		insetsCx := cx.ToAsync()
		platformWindow.OnInsetsChanged(func(WindowInsets) {
			logErrContext(handle.Update(insetsCx, func(_ AnyView, window *Window, _ *App) { window.Refresh() }), "handling insets change for window %#v", handle.WindowID())
		})
	}
	{
		resizeCx := cx.ToAsync()
		platformWindow.OnResize(func(Size[Pixels], float32) {
			logErrContext(handle.Update(resizeCx, func(_ AnyView, window *Window, cx *App) { window.BoundsChanged(cx) }), "handling resize for window %#v", handle.WindowID())
		})
	}
	{
		movedCx := cx.ToAsync()
		platformWindow.OnMoved(func() {
			logErrContext(handle.Update(movedCx, func(_ AnyView, window *Window, cx *App) { window.BoundsChanged(cx) }), "handling move for window %#v", handle.WindowID())
		})
	}
	{
		displayCx := cx.ToAsync()
		platformWindow.OnDisplayChanged(func() {
			displayCx.Update(func(cx *App) { windowDisplayChanged(handle, cx) })
		})
	}
	{
		appearanceCx := cx.ToAsync()
		foregroundExecutor := appearanceCx.ForegroundExecutor()
		platformWindow.OnAppearanceChanged(func() {
			taskCx := appearanceCx.clone()
			foregroundExecutor.Spawn(func(*scheduler.Scope) (struct{}, error) {
				logErrContext(handle.Update(taskCx, func(_ AnyView, window *Window, cx *App) { window.appearanceChanged(cx) }), "handling appearance change for window %#v", handle.WindowID())
				return struct{}{}, nil
			}).Detach()
		})
	}
	{
		buttonLayoutCx := cx.ToAsync()
		platformWindow.OnButtonLayoutChanged(func() {
			logErrContext(handle.Update(buttonLayoutCx, func(_ AnyView, window *Window, cx *App) { window.buttonLayoutChanged(cx) }), "handling button layout change for window %#v", handle.WindowID())
		})
	}
	{
		activeCx := cx.ToAsync()
		platformWindow.OnActiveStatusChange(func(isActive bool) {
			logErrContext(handle.Update(activeCx, func(_ AnyView, window *Window, cx *App) {
				*window.active = isActive
				window.modifiers = window.platformWindow.Modifiers()
				window.capslock = window.platformWindow.Capslock()
				window.activationObservers.retain(func(callback *anyObserver) bool { return (*callback)(window, cx) })

				window.BoundsChanged(cx)
				window.Refresh()

				SystemWindowTabControllerUpdateLastActive(cx, window.handle.id)
			}), "handling active status change for window %#v", handle.WindowID())
		})
	}
	{
		visibilityCx := cx.ToAsync()
		platformWindow.OnVisibilityChange(func(WindowVisibility) {
			logErrContext(handle.Update(visibilityCx, func(_ AnyView, window *Window, cx *App) {
				window.refreshVisibility(cx)
			}), "handling visibility change for window %#v", handle.WindowID())
		})
	}
	{
		hoverCx := cx.ToAsync()
		platformWindow.OnHoverStatusChange(func(isHovered bool) {
			logErrContext(handle.Update(hoverCx, func(_ AnyView, window *Window, _ *App) {
				*window.hovered = isHovered
				window.Refresh()
			}), "handling hover status change for window %#v", handle.WindowID())
		})
	}
	{
		inputCx := cx.ToAsync()
		platformWindow.OnInput(func(event PlatformInput) DispatchEventResult {
			result, err := handle.UpdateR(inputCx, func(_ AnyView, window *Window, cx *App) DispatchEventResult {
				return window.DispatchEvent(event, cx)
			})
			if err != nil {
				logErrContext(err, "dispatching input for window %#v", handle.WindowID())
				return DispatchEventResult{}
			}
			return result
		})
	}
	{
		controlCx := cx.ToAsync()
		platformWindow.OnHitTestWindowControl(func() (WindowControlArea, bool) {
			result, err := handle.UpdateR(controlCx, func(_ AnyView, window *Window, _ *App) Option[WindowControlArea] {
				for _, control := range window.renderedFrame.windowControlHitboxes {
					if slices.Contains(window.mouseHitTest.ids, control.hitbox.ID) {
						return Some(control.area)
					}
				}
				return None[WindowControlArea]()
			})
			if err != nil {
				logErrContext(err, "hit-testing window controls for window %#v", handle.WindowID())
				return 0, false
			}
			return result.Get()
		})
	}
	{
		nativeViewCx := cx.ToAsync()
		platformWindow.OnHitTestNativeView(func(position Point[Pixels]) (NativeViewID, bool) {
			result, err := handle.UpdateR(nativeViewCx, func(_ AnyView, window *Window, _ *App) Option[NativeViewID] {
				id, ok := window.hitTestNativeView(position)
				if !ok {
					return None[NativeViewID]()
				}
				return Some(id)
			})
			if err != nil {
				return 0, false
			}
			return result.Get()
		})
	}
	{
		keyDownCx := cx.ToAsync()
		platformWindow.OnKeyDownBeforeNativeView(func(event KeyDownEvent) bool {
			result, err := handle.UpdateR(keyDownCx, func(_ AnyView, window *Window, cx *App) bool {
				return window.dispatchKeyDownBeforeNativeView(event, cx)
			})
			if err != nil {
				return false
			}
			return result
		})
	}
	{
		focusedCx := cx.ToAsync()
		platformWindow.OnNativeViewFocused(func(id NativeViewID) {
			logErrContext(handle.Update(focusedCx, func(_ AnyView, window *Window, cx *App) { window.focusNativeView(id, cx) }), "focusing a native view in window %#v", handle.WindowID())
		})
	}
	{
		moveTabCx := cx.ToAsync()
		platformWindow.OnMoveTabToNewWindow(func() {
			logErrContext(handle.Update(moveTabCx, func(_ AnyView, _ *Window, cx *App) {
				SystemWindowTabControllerMoveTabToNewWindow(cx, handle.WindowID())
			}), "moving tab to new window from window %#v", handle.WindowID())
		})
	}
	{
		mergeCx := cx.ToAsync()
		platformWindow.OnMergeAllWindows(func() {
			logErrContext(handle.Update(mergeCx, func(_ AnyView, _ *Window, cx *App) {
				SystemWindowTabControllerMergeAllWindows(cx, handle.WindowID())
			}), "merging all windows into window %#v", handle.WindowID())
		})
	}
	{
		nextTabCx := cx.ToAsync()
		platformWindow.OnSelectNextTab(func() {
			logErrContext(handle.Update(nextTabCx, func(_ AnyView, _ *Window, cx *App) {
				SystemWindowTabControllerSelectNextTab(cx, handle.WindowID())
			}), "selecting next tab for window %#v", handle.WindowID())
		})
	}
	{
		previousTabCx := cx.ToAsync()
		platformWindow.OnSelectPreviousTab(func() {
			logErrContext(handle.Update(previousTabCx, func(_ AnyView, _ *Window, cx *App) {
				SystemWindowTabControllerSelectPreviousTab(cx, handle.WindowID())
			}), "selecting previous tab for window %#v", handle.WindowID())
		})
	}
	{
		toggleTabBarCx := cx.ToAsync()
		platformWindow.OnToggleTabBar(func() {
			logErrContext(handle.Update(toggleTabBarCx, func(_ AnyView, window *Window, cx *App) {
				tabBarVisible := window.platformWindow.TabBarVisible()
				SystemWindowTabControllerSetVisible(cx, tabBarVisible)
			}), "toggling tab bar for window %#v", handle.WindowID())
		})
	}

	if id, ok := appID.Get(); ok {
		platformWindow.SetAppID(id)
	}

	if err := platformWindow.MapWindow(); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}

	var windowProfiler *WindowProfiler
	var debugFrameOverlay DebugFrameOverlay
	if profilerFeatureEnabled {
		windowProfiler, err = NewWindowProfiler(handle.WindowID())
		if err != nil {
			return nil, err
		}
		debugFrameOverlay = NewDebugFrameOverlay()
	}

	gestureTuning := DefaultGestureTuning()
	if gestures := cx.platform.Gestures(); gestures != nil {
		gestureTuning = gestures.Tuning()
	}

	return &Window{
		closeA11yChannels:      closeA11yChannels,
		handle:                 handle,
		invalidator:            invalidator,
		removed:                false,
		platformWindow:         platformWindow,
		displayID:              displayID,
		displayObservers:       newSingleSubscriberSet[anyObserver](),
		isResizable:            isResizable,
		isMinimizable:          isMinimizable,
		lastWindowBounds:       windowBounds,
		spriteAtlas:            spriteAtlas,
		textSystem:             textSystem,
		textRenderingMode:      cx.textRenderingMode,
		remSize:                Px(16.),
		viewportSize:           contentSize,
		layoutEngine:           NewTaffyLayoutEngine(),
		root:                   None[AnyView](),
		currentElementOpacity:  1.0,
		requestedAutoscroll:    None[Bounds[Pixels]](),
		renderedFrame:          newFrame(newDispatchTree(cx.keymap, cx.actions)),
		nextFrame:              newFrame(newDispatchTree(cx.keymap, cx.actions)),
		nextFrameCallbacks:     nextFrameCallbacks,
		nextHitboxID:           0,
		nextTooltipID:          0,
		dirtyViews:             make(map[EntityID]struct{}),
		focusListeners:         newSingleSubscriberSet[anyWindowFocusListener](),
		focusLostListeners:     newSingleSubscriberSet[anyObserver](),
		defaultPrevented:       true,
		mousePosition:          mousePosition,
		modifiers:              modifiers,
		capslock:               capslock,
		scaleFactor:            scaleFactor,
		boundsObservers:        newSingleSubscriberSet[anyObserver](),
		appearance:             appearance,
		appearanceObservers:    newSingleSubscriberSet[anyObserver](),
		buttonLayoutObservers:  newSingleSubscriberSet[anyObserver](),
		active:                 active,
		visibility:             visibility,
		visibilityObservers:    newSingleSubscriberSet[func(visibility WindowVisibility, window *Window, cx *App) bool](),
		hovered:                hovered,
		needsPresent:           needsPresent,
		sceneAnimationInterval: sceneAnimationInterval,
		inputRateTracker:       inputRateTracker,
		windowProfiler:         windowProfiler,
		lastInputModality:      inputModalityMouse,
		touchGestures:          newTouchGestureRecognizer(gestureTuning),
		touchPredictionEnabled: true,
		refreshing:             false,
		activationObservers:    newSingleSubscriberSet[anyObserver](),
		focusEnabled:           true,
		pendingInputObservers:  newSingleSubscriberSet[anyObserver](),
		debugFrameOverlay:      debugFrameOverlay,
		a11y:                   newWindowA11y(a11yActiveFlag, accessibilityForceDisabled, initialWindowTitle),
	}, nil
}

func (w *Window) newFocusListener(value anyWindowFocusListener) (Subscription, func()) {
	return w.focusListeners.insertPending(value)
}

// DispatchEventResult reports what happened during event dispatch.
type DispatchEventResult struct {
	// Propagate reports whether the event should continue to the platform.
	Propagate bool
	// DefaultPrevented reports whether a handler prevented the default action.
	DefaultPrevented bool
}

// ContentMask indicates which region of the window is visible; content outside of it is not
// rendered.
type ContentMask[T Number] struct {
	// Bounds are the visible bounds.
	Bounds Bounds[T]
}

// Scale scales the content mask's pixel units by the given factor.
func (m ContentMask[T]) Scale(factor float32) ContentMask[ScaledPixels] {
	return ContentMask[ScaledPixels]{Bounds: m.Bounds.Scale(factor)}
}

// Intersect intersects the content mask with other.
func (m ContentMask[T]) Intersect(other ContentMask[T]) ContentMask[T] {
	return ContentMask[T]{Bounds: m.Bounds.Intersect(other.Bounds)}
}

// WindowID is a unique identifier for a window (a slot map key: version in the high 32 bits,
// index in the low 32 bits).
type WindowID uint64

// AsU64 converts this window id to a uint64.
func (id WindowID) AsU64() uint64 {
	return uint64(id)
}

// GoString formats the id like upstream's Debug, WindowId(<idx>v<version>).
func (id WindowID) GoString() string {
	return fmt.Sprintf("WindowId(%dv%d)", uint32(id), uint32(id>>32)|1)
}

// WindowHandle is a handle to a window with root view type V. It does not keep the window
// alive on its own; it embeds its AnyWindowHandle (upstream's Deref).
type WindowHandle[V any] struct {
	AnyWindowHandle
}

// GoString formats the handle like upstream's Debug.
func (h WindowHandle[V]) GoString() string {
	return "WindowHandle { any_handle: " + strconv.FormatUint(h.id.AsU64(), 10) + " }"
}

// NewWindowHandle creates a handle from a window id; it does not check that the root type of
// the window is V.
func NewWindowHandle[V any](id WindowID) WindowHandle[V] {
	return WindowHandle[V]{AnyWindowHandle: AnyWindowHandle{
		id:                 id,
		stateType:          reflect.TypeFor[V](),
		rootEntityTypeName: typeNameOf(reflect.TypeFor[V]()),
	}}
}

// Root returns the root view of this window, which the caller releases. It fails if the window
// is closed or the root view's type is not V.
func (h WindowHandle[V]) Root(cx AppContext) (Entity[V], error) {
	var result Entity[V]
	var downcastErr error
	err := cx.updateWindow(h.AnyWindowHandle, func(rootView AnyView, _ *Window, _ *App) {
		view, ok := rootView.Downcast[V]()
		if !ok {
			downcastErr = fmt.Errorf("the type of the window's root view has changed")
			return
		}
		result = view.Clone()
	})
	if err != nil {
		return Entity[V]{}, err
	}
	return result, downcastErr
}

// Update updates the root view of this window. It fails if the window has been closed or the
// root view's type is not V.
func (h WindowHandle[V]) Update(cx AppContext, update func(view *V, window *Window, cx *Context[V])) error {
	_, err := h.UpdateR(cx, func(view *V, window *Window, cx *Context[V]) struct{} {
		update(view, window, cx)
		return struct{}{}
	})
	return err
}

// UpdateR is Update for a callback that returns a value.
func (h WindowHandle[V]) UpdateR[R any](cx AppContext, update func(view *V, window *Window, cx *Context[V]) R) (R, error) {
	var result R
	var downcastErr error
	err := cx.updateWindow(h.AnyWindowHandle, func(rootView AnyView, window *Window, app *App) {
		view, ok := rootView.Downcast[V]()
		if !ok {
			downcastErr = fmt.Errorf("the type of the window's root view has changed")
			return
		}
		result = view.UpdateR(app, func(state *V, cx *Context[V]) R { return update(state, window, cx) })
	})
	if err != nil {
		return result, err
	}
	return result, downcastErr
}

// Read reads the root view out of this window. It fails if the window is closed or the root
// view's type is not V.
func (h WindowHandle[V]) Read(cx *App) (*V, error) {
	window, ok := cx.windows.get(h.id)
	if !ok || window == nil {
		return nil, fmt.Errorf("window not found")
	}
	rootView, ok := window.root.Get()
	if !ok {
		return nil, fmt.Errorf("window not found")
	}
	view, ok := rootView.Downcast[V]()
	if !ok {
		return nil, fmt.Errorf("the type of the window's root view has changed")
	}
	return view.Read(cx), nil
}

// ReadWith reads the root view of the window with a callback. It fails if the
// window is closed or the root view's type is not V.
func (h WindowHandle[V]) ReadWith[R any](cx AppContext, readWith func(view *V, cx *App) R) (R, error) {
	return AppContextReadWindow(cx, h, func(rootView Entity[V], cx *App) R { return readWith(rootView.Read(cx), cx) })
}

// Entity returns the root view of this window, which the caller releases. It fails if the
// window is closed or the root view's type is not V.
func (h WindowHandle[V]) Entity(cx AppContext) (Entity[V], error) {
	return AppContextReadWindow(cx, h, func(rootView Entity[V], _ *App) Entity[V] { return rootView.Clone() })
}

// IsActive reports whether this window is active; it reports false for ok when the window is
// closed or currently borrowed.
func (h WindowHandle[V]) IsActive(cx *App) (active bool, ok bool) {
	active, err := AppContextUpdateWindow(cx, h.AnyWindowHandle, func(_ AnyView, window *Window, _ *App) bool {
		return window.IsWindowActive()
	})
	return active, err == nil
}

// AnyWindowHandle is a handle to a window with any root view type, which can be downcast to a
// handle with a specific root view type.
type AnyWindowHandle struct {
	id                 WindowID
	stateType          reflect.Type
	rootEntityTypeName string
}

// WindowID returns the id of this window.
func (h AnyWindowHandle) WindowID() WindowID {
	return h.id
}

// RootEntityTypeName returns the name of the window's declared root entity type.
func (h AnyWindowHandle) RootEntityTypeName() string {
	return h.rootEntityTypeName
}

// Downcast converts the handle to a handle with root view type T; it reports false if the types
// do not match.
func (h AnyWindowHandle) Downcast[T any]() (WindowHandle[T], bool) {
	if reflect.TypeFor[T]() == h.stateType {
		return WindowHandle[T]{AnyWindowHandle: h}, true
	}
	return WindowHandle[T]{}, false
}

// Update updates the state of the root view of this window. It fails if the window has been
// closed.
func (h AnyWindowHandle) Update(cx AppContext, update func(root AnyView, window *Window, cx *App)) error {
	return cx.updateWindow(h, update)
}

// UpdateR is Update for a callback that returns a value.
func (h AnyWindowHandle) UpdateR[R any](cx AppContext, update func(root AnyView, window *Window, cx *App) R) (R, error) {
	return AppContextUpdateWindow(cx, h, update)
}

// Read reads the state of the root view of this window, whose type must be T. It fails if the
// window has been closed.
func (h AnyWindowHandle) Read[T, R any](cx AppContext, read func(root Entity[T], cx *App) R) (R, error) {
	view, ok := h.Downcast[T]()
	if !ok {
		var zero R
		return zero, fmt.Errorf("the type of the window's root view has changed")
	}
	return AppContextReadWindow(cx, view, read)
}

// RawWindowHandle returns the platform window handle (upstream's HasWindowHandle and
// HasDisplayHandle impls, D-NAME).
func (w *Window) RawWindowHandle() RawWindowHandle {
	return w.platformWindow.RawWindowHandle()
}

// ElementID is an identifier for an element among its siblings: a comparable tagged value
// built from a string, a number, both, or one of the other variant constructors.
type ElementID struct {
	kind  uint8
	name  SharedString
	num   uint64
	bytes [20]byte
	line  uint32
	col   uint32
	child *ElementID
}

const (
	elementIDView uint8 = iota
	elementIDInteger
	elementIDName
	elementIDUUID
	elementIDFocusHandle
	elementIDNamedInteger
	elementIDPath
	elementIDCodeLocation
	elementIDNamedChild
	elementIDOpaqueID
)

// ElementIDSource is the set of types an ElementID can be built from with NewElementID (D-INTO):
// strings (Name), integers (Integer), an ElementID, a *FocusHandle (FocusHandle), a *SourceLocation
// (CodeLocation), a [16]byte UUID (Uuid) and a [20]byte opaque id (OpaqueId).
type ElementIDSource interface {
	~string | ~int | ~int32 | ~uint | ~uint32 | ~uint64 | ElementID | *FocusHandle | *SourceLocation | [16]byte | [20]byte
}

// NewElementID builds an ElementID from one of the ElementIDSource types.
func NewElementID[I ElementIDSource](id I) ElementID {
	switch v := any(id).(type) {
	case ElementID:
		return v
	case *FocusHandle:
		return ElementIDFocusHandle(v.id)
	case *SourceLocation:
		return ElementIDCodeLocation(v)
	case [16]byte:
		return ElementIDUUID(v)
	case [20]byte:
		return ElementIDOpaqueID(v)
	}
	value := reflect.ValueOf(id)
	switch value.Kind() {
	case reflect.String:
		return ElementIDName(SharedString(value.String()))
	case reflect.Int, reflect.Int32:
		return ElementIDInteger(uint64(value.Int()))
	default:
		return ElementIDInteger(value.Uint())
	}
}

// ElementIDView is the id of a view element.
func ElementIDView(id EntityID) ElementID {
	return ElementID{kind: elementIDView, num: uint64(id)}
}

// ElementIDInteger is an integer id.
func ElementIDInteger(id uint64) ElementID {
	return ElementID{kind: elementIDInteger, num: id}
}

// ElementIDName is a string based id.
func ElementIDName(name SharedString) ElementID {
	return ElementID{kind: elementIDName, name: name}
}

// ElementIDUUID is a UUID id.
func ElementIDUUID(uuid [16]byte) ElementID {
	id := ElementID{kind: elementIDUUID}
	copy(id.bytes[:], uuid[:])
	return id
}

// ElementIDFocusHandle is an id that is equated with a focus handle.
func ElementIDFocusHandle(id FocusID) ElementID {
	return ElementID{kind: elementIDFocusHandle, num: uint64(id)}
}

// ElementIDNamedInteger is a combination of a name and an integer.
func ElementIDNamedInteger(name SharedString, integer uint64) ElementID {
	return ElementID{kind: elementIDNamedInteger, name: name, num: integer}
}

// ElementIDNamedUsize constructs a NamedInteger id from a name and an int.
func ElementIDNamedUsize(name string, integer int) ElementID {
	return ElementIDNamedInteger(SharedString(name), uint64(integer))
}

// ElementIDPath is a path id.
func ElementIDPath(path string) ElementID {
	return ElementID{kind: elementIDPath, name: SharedString(path)}
}

// ElementIDCodeLocation is a code location id.
func ElementIDCodeLocation(location *SourceLocation) ElementID {
	return ElementID{kind: elementIDCodeLocation, name: SharedString(location.File), line: location.Line, col: location.Column}
}

// ElementIDNamedChild is a labeled child of an element.
func ElementIDNamedChild(id ElementID, name SharedString) ElementID {
	return ElementID{kind: elementIDNamedChild, child: internElementID(id), name: name}
}

// ElementIDOpaqueID is a byte array id (used for text anchors).
func ElementIDOpaqueID(opaqueID [20]byte) ElementID {
	return ElementID{kind: elementIDOpaqueID, bytes: opaqueID}
}

var elementIDInterner struct {
	mu  sync.Mutex
	ids map[ElementID]*ElementID
}

func internElementID(id ElementID) *ElementID {
	elementIDInterner.mu.Lock()
	defer elementIDInterner.mu.Unlock()
	if interned, ok := elementIDInterner.ids[id]; ok {
		return interned
	}
	if elementIDInterner.ids == nil {
		elementIDInterner.ids = make(map[ElementID]*ElementID)
	}
	interned := new(ElementID)
	*interned = id
	elementIDInterner.ids[id] = interned
	return interned
}

func hashPath(h *FxHasher, path string) {
	bytes := []byte(path)
	componentStart := 0
	bytesHashed := 0
	for i := range bytes {
		if bytes[i] == '/' {
			if i > componentStart {
				toHash := bytes[componentStart:i]
				h.Write(toHash)
				bytesHashed += len(toHash)
			}
			componentStart = i + 1
			tail := bytes[componentStart:]
			if len(tail) == 1 && tail[0] == '.' || len(tail) >= 2 && tail[0] == '.' && tail[1] == '/' {
				componentStart++
			}
		}
	}
	if componentStart < len(bytes) {
		toHash := bytes[componentStart:]
		h.Write(toHash)
		bytesHashed += len(toHash)
	}
	h.WriteUsize(uint64(bytesHashed))
}

// Hash feeds the id to h as upstream's derived Hash does: the variant's discriminant, then its
// fields.
func (id ElementID) Hash(h *FxHasher) {
	h.WriteUsize(uint64(id.kind))
	switch id.kind {
	case elementIDView, elementIDInteger, elementIDFocusHandle:
		h.WriteU64(id.num)
	case elementIDName:
		h.WriteString(string(id.name))
	case elementIDUUID:
		h.Write(id.bytes[:16])
	case elementIDNamedInteger:
		h.WriteString(string(id.name))
		h.WriteU64(id.num)
	case elementIDPath:
		hashPath(h, string(id.name))
	case elementIDCodeLocation:
		h.WriteString(string(id.name))
		h.WriteU64(uint64(id.line))
		h.WriteU64(uint64(id.col))
	case elementIDNamedChild:
		id.child.Hash(h)
		h.WriteString(string(id.name))
	case elementIDOpaqueID:
		h.WriteUsize(uint64(len(id.bytes)))
		h.Write(id.bytes[:])
	}
}

func formatUUID(b []byte) string {
	s := hex.EncodeToString(b)
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}

// String formats the id like upstream's Display impl.
func (id ElementID) String() string {
	switch id.kind {
	case elementIDView:
		return "view-" + EntityID(id.num).String()
	case elementIDInteger:
		return strconv.FormatUint(id.num, 10)
	case elementIDName:
		return string(id.name)
	case elementIDFocusHandle:
		return "FocusHandle"
	case elementIDNamedInteger:
		return string(id.name) + "-" + strconv.FormatUint(id.num, 10)
	case elementIDUUID:
		return formatUUID(id.bytes[:16])
	case elementIDPath:
		return string(id.name)
	case elementIDCodeLocation:
		return string(id.name) + ":" + strconv.FormatUint(uint64(id.line), 10) + ":" + strconv.FormatUint(uint64(id.col), 10)
	case elementIDNamedChild:
		return id.child.String() + "-" + string(id.name)
	case elementIDOpaqueID:
		parts := make([]string, len(id.bytes))
		for i, b := range id.bytes {
			parts[i] = strconv.FormatUint(uint64(b), 16)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return ""
}

// GoString formats the id like upstream's derived Debug impl.
func (id ElementID) GoString() string {
	switch id.kind {
	case elementIDView:
		return "View(" + EntityID(id.num).debugString() + ")"
	case elementIDInteger:
		return "Integer(" + strconv.FormatUint(id.num, 10) + ")"
	case elementIDName:
		return "Name(" + strconv.Quote(string(id.name)) + ")"
	case elementIDUUID:
		return "Uuid(" + formatUUID(id.bytes[:16]) + ")"
	case elementIDFocusHandle:
		return "FocusHandle(" + FocusID(id.num).GoString() + ")"
	case elementIDNamedInteger:
		return "NamedInteger(" + strconv.Quote(string(id.name)) + ", " + strconv.FormatUint(id.num, 10) + ")"
	case elementIDPath:
		return "Path(" + strconv.Quote(string(id.name)) + ")"
	case elementIDCodeLocation:
		return fmt.Sprintf("CodeLocation(Location { file: %s, line: %d, col: %d })", strconv.Quote(string(id.name)), id.line, id.col)
	case elementIDNamedChild:
		return "NamedChild(" + id.child.GoString() + ", " + strconv.Quote(string(id.name)) + ")"
	case elementIDOpaqueID:
		parts := make([]string, len(id.bytes))
		for i, b := range id.bytes {
			parts[i] = strconv.FormatUint(uint64(b), 10)
		}
		return "OpaqueId([" + strings.Join(parts, ", ") + "])"
	}
	return ""
}

// TryIntoSharedString returns the name of a Name id; it fails for every other variant.
func (id ElementID) TryIntoSharedString() (SharedString, error) {
	if id.kind == elementIDName {
		return id.name, nil
	}
	return "", fmt.Errorf("element id is not string")
}

// PaintQuad is a rectangle rendered in the window at the given position and size; it is passed
// to Window.PaintQuad.
type PaintQuad struct {
	// Bounds are the bounds of the quad within the window.
	Bounds Bounds[Pixels]
	// CornerRadii are the radii of the quad's corners.
	CornerRadii Corners[Pixels]
	// Background is the background of the quad.
	Background Background
	// BorderWidths are the widths of the quad's borders.
	BorderWidths Edges[Pixels]
	// BorderColor is the color of the quad's borders.
	BorderColor Hsla
	// BorderStyle is the style of the quad's borders.
	BorderStyle BorderStyle
}

// WithCornerRadii sets the corner radii of the quad (upstream's corner_radii, D-FIELDMETHOD).
func (q PaintQuad) WithCornerRadii(cornerRadii Corners[Pixels]) PaintQuad {
	q.CornerRadii = cornerRadii
	return q
}

// WithBorderWidths sets the border widths of the quad (upstream's border_widths, D-FIELDMETHOD).
func (q PaintQuad) WithBorderWidths(borderWidths Edges[Pixels]) PaintQuad {
	q.BorderWidths = borderWidths
	return q
}

// WithBorderColor sets the border color of the quad (upstream's border_color, D-FIELDMETHOD).
func (q PaintQuad) WithBorderColor(borderColor IntoHsla) PaintQuad {
	q.BorderColor = borderColor.ToHsla()
	return q
}

// WithBackground sets the background of the quad (upstream's background, D-FIELDMETHOD).
func (q PaintQuad) WithBackground(background Background) PaintQuad {
	q.Background = background
	return q
}

// NewQuad creates a quad with the given parameters.
func NewQuad(bounds Bounds[Pixels], cornerRadii Corners[Pixels], background Background, borderWidths Edges[Pixels], borderColor Hsla, borderStyle BorderStyle) PaintQuad {
	return PaintQuad{
		Bounds:       bounds,
		CornerRadii:  cornerRadii,
		Background:   background,
		BorderWidths: borderWidths,
		BorderColor:  borderColor,
		BorderStyle:  borderStyle,
	}
}

// NewFill creates a filled quad with the given bounds and background.
func NewFill(bounds Bounds[Pixels], background Background) PaintQuad {
	return PaintQuad{
		Bounds:       bounds,
		CornerRadii:  CornersAll(Px(0.)),
		Background:   background,
		BorderWidths: EdgesAll(Px(0.)),
		BorderColor:  TransparentBlack(),
		BorderStyle:  BorderStyleSolid,
	}
}

// Outline creates a rectangle outline with the given bounds, border color and a 1px border width.
func Outline(bounds Bounds[Pixels], borderColor Hsla, borderStyle BorderStyle) PaintQuad {
	return PaintQuad{
		Bounds:       bounds,
		CornerRadii:  CornersAll(Px(0.)),
		Background:   TransparentBlack().ToBackground(),
		BorderWidths: EdgesAll(Px(1.)),
		BorderColor:  borderColor,
		BorderStyle:  borderStyle,
	}
}
