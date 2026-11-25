//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSWindowStyleMask is a window's style mask.
type NSWindowStyleMask uint

// NSWindowStyleMask values.
const (
	NSWindowStyleMaskBorderless             NSWindowStyleMask = 0
	NSWindowStyleMaskTitled                 NSWindowStyleMask = 1 << 0
	NSWindowStyleMaskClosable               NSWindowStyleMask = 1 << 1
	NSWindowStyleMaskMiniaturizable         NSWindowStyleMask = 1 << 2
	NSWindowStyleMaskResizable              NSWindowStyleMask = 1 << 3
	NSWindowStyleMaskUtilityWindow          NSWindowStyleMask = 1 << 4
	NSWindowStyleMaskDocModalWindow         NSWindowStyleMask = 1 << 6
	NSWindowStyleMaskNonactivatingPanel     NSWindowStyleMask = 1 << 7
	NSWindowStyleMaskTexturedBackground     NSWindowStyleMask = 1 << 8
	NSWindowStyleMaskUnifiedTitleAndToolbar NSWindowStyleMask = 1 << 12
	NSWindowStyleMaskHUDWindow              NSWindowStyleMask = 1 << 13
	NSWindowStyleMaskFullScreen             NSWindowStyleMask = 1 << 14
	NSWindowStyleMaskFullSizeContentView    NSWindowStyleMask = 1 << 15
)

// Contains reports whether every flag in other is set in m.
func (m NSWindowStyleMask) Contains(other NSWindowStyleMask) bool { return m&other == other }

// NSWindowCollectionBehavior is a window's collection behaviour.
type NSWindowCollectionBehavior uint

// NSWindowCollectionBehavior values.
const (
	NSWindowCollectionBehaviorDefault                   NSWindowCollectionBehavior = 0
	NSWindowCollectionBehaviorCanJoinAllSpaces          NSWindowCollectionBehavior = 1 << 0
	NSWindowCollectionBehaviorMoveToActiveSpace         NSWindowCollectionBehavior = 1 << 1
	NSWindowCollectionBehaviorManaged                   NSWindowCollectionBehavior = 1 << 2
	NSWindowCollectionBehaviorTransient                 NSWindowCollectionBehavior = 1 << 3
	NSWindowCollectionBehaviorStationary                NSWindowCollectionBehavior = 1 << 4
	NSWindowCollectionBehaviorParticipatesInCycle       NSWindowCollectionBehavior = 1 << 5
	NSWindowCollectionBehaviorIgnoresCycle              NSWindowCollectionBehavior = 1 << 6
	NSWindowCollectionBehaviorFullScreenPrimary         NSWindowCollectionBehavior = 1 << 7
	NSWindowCollectionBehaviorFullScreenAuxiliary       NSWindowCollectionBehavior = 1 << 8
	NSWindowCollectionBehaviorFullScreenNone            NSWindowCollectionBehavior = 1 << 9
	NSWindowCollectionBehaviorFullScreenAllowsTiling    NSWindowCollectionBehavior = 1 << 11
	NSWindowCollectionBehaviorFullScreenDisallowsTiling NSWindowCollectionBehavior = 1 << 12
	NSWindowCollectionBehaviorPrimary                   NSWindowCollectionBehavior = 1 << 16
	NSWindowCollectionBehaviorAuxiliary                 NSWindowCollectionBehavior = 1 << 17
	NSWindowCollectionBehaviorCanJoinAllApplications    NSWindowCollectionBehavior = 1 << 18
)

// NSBackingStoreType is a window's backing store type.
type NSBackingStoreType uint

// NSBackingStoreType values.
const (
	NSBackingStoreRetained    NSBackingStoreType = 0
	NSBackingStoreNonretained NSBackingStoreType = 1
	NSBackingStoreBuffered    NSBackingStoreType = 2
)

// NSWindowTitleVisibility is a window's title visibility.
type NSWindowTitleVisibility int

// NSWindowTitleVisibility values.
const (
	NSWindowTitleVisible NSWindowTitleVisibility = 0
	NSWindowTitleHidden  NSWindowTitleVisibility = 1
)

// NSWindowButton identifies a standard title bar button.
type NSWindowButton uint

// NSWindowButton values.
const (
	NSWindowCloseButton            NSWindowButton = 0
	NSWindowMiniaturizeButton      NSWindowButton = 1
	NSWindowZoomButton             NSWindowButton = 2
	NSWindowToolbarButton          NSWindowButton = 3
	NSWindowDocumentIconButton     NSWindowButton = 4
	NSWindowDocumentVersionsButton NSWindowButton = 6
)

// NSWindowOrderingMode is the ordering of one window relative to another.
type NSWindowOrderingMode int

// NSWindowOrderingMode values.
const (
	NSWindowAbove NSWindowOrderingMode = 1
	NSWindowBelow NSWindowOrderingMode = -1
	NSWindowOut   NSWindowOrderingMode = 0
)

// NSWindowOcclusionState is a window's occlusion state.
type NSWindowOcclusionState uint

// NSWindowOcclusionStateVisible is set when any part of the window is visible.
const NSWindowOcclusionStateVisible NSWindowOcclusionState = 1 << 1

// NSWindowLevel is a window's level.
type NSWindowLevel int

// NSWindowLevel values.
const (
	NSNormalWindowLevel    NSWindowLevel = 0
	NSFloatingWindowLevel  NSWindowLevel = 3
	NSPopUpMenuWindowLevel NSWindowLevel = 101
)

// NSWindow wraps an NSWindow object.
type NSWindow objc.ID

// ID returns the underlying object.
func (w NSWindow) ID() objc.ID { return objc.ID(w) }

// InitWithContentRect initialises an allocated window object.
func InitWithContentRect(alloc objc.ID, rect objc.CGRect, style NSWindowStyleMask, backing NSBackingStoreType, deferCreation bool) NSWindow {
	return NSWindow(send[objc.ID](alloc, "initWithContentRect:styleMask:backing:defer:", rect, style, backing, deferCreation))
}

// Frame returns the window frame in screen coordinates.
func (w NSWindow) Frame() objc.CGRect { return send[objc.CGRect](w.ID(), "frame") }

// SetFrameDisplay sets the window frame.
func (w NSWindow) SetFrameDisplay(rect objc.CGRect, display bool) {
	sendVoid(w.ID(), "setFrame:display:", rect, display)
}

// SetFrameTopLeftPoint moves the window's top-left corner.
func (w NSWindow) SetFrameTopLeftPoint(point objc.CGPoint) {
	sendVoid(w.ID(), "setFrameTopLeftPoint:", point)
}

// ContentView returns the content view.
func (w NSWindow) ContentView() NSView { return NSView(send[objc.ID](w.ID(), "contentView")) }

// SetContentView sets the content view.
func (w NSWindow) SetContentView(view NSView) { sendVoid(w.ID(), "setContentView:", view.ID()) }

// ContentLayoutRect returns the content layout rectangle.
func (w NSWindow) ContentLayoutRect() objc.CGRect {
	return send[objc.CGRect](w.ID(), "contentLayoutRect")
}

// ContentRectForFrameRect converts a frame rectangle to a content rectangle.
func (w NSWindow) ContentRectForFrameRect(rect objc.CGRect) objc.CGRect {
	return send[objc.CGRect](w.ID(), "contentRectForFrameRect:", rect)
}

// FrameRectForContentRect converts a content rectangle to a frame rectangle.
func (w NSWindow) FrameRectForContentRect(rect objc.CGRect) objc.CGRect {
	return send[objc.CGRect](w.ID(), "frameRectForContentRect:", rect)
}

// Screen returns the window's screen or nil.
func (w NSWindow) Screen() NSScreen { return NSScreen(send[objc.ID](w.ID(), "screen")) }

// BackingScaleFactor returns the backing scale factor.
func (w NSWindow) BackingScaleFactor() objc.CGFloat {
	return send[objc.CGFloat](w.ID(), "backingScaleFactor")
}

// StyleMask returns the style mask.
func (w NSWindow) StyleMask() NSWindowStyleMask { return send[NSWindowStyleMask](w.ID(), "styleMask") }

// SetStyleMask sets the style mask.
func (w NSWindow) SetStyleMask(mask NSWindowStyleMask) { sendVoid(w.ID(), "setStyleMask:", mask) }

// SetTitle sets the window title.
func (w NSWindow) SetTitle(title string) {
	s := objc.NSStringFromGo(title)
	defer objc.Release(s)
	sendVoid(w.ID(), "setTitle:", s)
}

// SetTitleVisibility sets the title visibility.
func (w NSWindow) SetTitleVisibility(v NSWindowTitleVisibility) {
	sendVoid(w.ID(), "setTitleVisibility:", v)
}

// SetTitlebarAppearsTransparent sets whether the title bar is transparent.
func (w NSWindow) SetTitlebarAppearsTransparent(flag bool) {
	sendVoid(w.ID(), "setTitlebarAppearsTransparent:", flag)
}

// StandardWindowButton returns a title bar button or nil.
func (w NSWindow) StandardWindowButton(button NSWindowButton) NSView {
	return NSView(send[objc.ID](w.ID(), "standardWindowButton:", button))
}

// SetDelegate sets the window delegate.
func (w NSWindow) SetDelegate(delegate objc.ID) { sendVoid(w.ID(), "setDelegate:", delegate) }

// MakeKeyAndOrderFront shows the window and makes it key.
func (w NSWindow) MakeKeyAndOrderFront(sender objc.ID) {
	sendVoid(w.ID(), "makeKeyAndOrderFront:", sender)
}

// OrderFront brings the window to the front.
func (w NSWindow) OrderFront(sender objc.ID) { sendVoid(w.ID(), "orderFront:", sender) }

// OrderOut hides the window.
func (w NSWindow) OrderOut(sender objc.ID) { sendVoid(w.ID(), "orderOut:", sender) }

// OrderWindowRelativeTo orders the window relative to another window number.
func (w NSWindow) OrderWindowRelativeTo(mode NSWindowOrderingMode, other int) {
	sendVoid(w.ID(), "orderWindow:relativeTo:", mode, other)
}

// Close closes the window.
func (w NSWindow) Close() { sendVoid(w.ID(), "close") }

// PerformClose simulates a click on the close button.
func (w NSWindow) PerformClose(sender objc.ID) { sendVoid(w.ID(), "performClose:", sender) }

// Miniaturize minimises the window.
func (w NSWindow) Miniaturize(sender objc.ID) { sendVoid(w.ID(), "miniaturize:", sender) }

// Zoom toggles the zoomed state.
func (w NSWindow) Zoom(sender objc.ID) { sendVoid(w.ID(), "zoom:", sender) }

// IsZoomed reports whether the window is zoomed.
func (w NSWindow) IsZoomed() bool { return send[bool](w.ID(), "isZoomed") }

// IsMiniaturized reports whether the window is minimised.
func (w NSWindow) IsMiniaturized() bool { return send[bool](w.ID(), "isMiniaturized") }

// ToggleFullScreen toggles full screen.
func (w NSWindow) ToggleFullScreen(sender objc.ID) { sendVoid(w.ID(), "toggleFullScreen:", sender) }

// IsKeyWindow reports whether the window is key.
func (w NSWindow) IsKeyWindow() bool { return send[bool](w.ID(), "isKeyWindow") }

// IsVisible reports whether the window is on screen.
func (w NSWindow) IsVisible() bool { return send[bool](w.ID(), "isVisible") }

// OcclusionState returns the occlusion state.
func (w NSWindow) OcclusionState() NSWindowOcclusionState {
	return send[NSWindowOcclusionState](w.ID(), "occlusionState")
}

// WindowNumber returns the window number.
func (w NSWindow) WindowNumber() int { return send[int](w.ID(), "windowNumber") }

// SetLevel sets the window level.
func (w NSWindow) SetLevel(level NSWindowLevel) { sendVoid(w.ID(), "setLevel:", level) }

// SetCollectionBehavior sets the collection behaviour.
func (w NSWindow) SetCollectionBehavior(b NSWindowCollectionBehavior) {
	sendVoid(w.ID(), "setCollectionBehavior:", b)
}

// CollectionBehavior returns the collection behaviour.
func (w NSWindow) CollectionBehavior() NSWindowCollectionBehavior {
	return send[NSWindowCollectionBehavior](w.ID(), "collectionBehavior")
}

// SetOpaque sets whether the window is opaque.
func (w NSWindow) SetOpaque(flag bool) { sendVoid(w.ID(), "setOpaque:", flag) }

// SetHasShadow sets whether the window has a shadow.
func (w NSWindow) SetHasShadow(flag bool) { sendVoid(w.ID(), "setHasShadow:", flag) }

// SetBackgroundColor sets the background colour.
func (w NSWindow) SetBackgroundColor(color NSColor) {
	sendVoid(w.ID(), "setBackgroundColor:", color.ID())
}

// SetAcceptsMouseMovedEvents sets whether mouse-moved events are delivered.
func (w NSWindow) SetAcceptsMouseMovedEvents(flag bool) {
	sendVoid(w.ID(), "setAcceptsMouseMovedEvents:", flag)
}

// SetMovable sets whether the window can be moved.
func (w NSWindow) SetMovable(flag bool) { sendVoid(w.ID(), "setMovable:", flag) }

// SetMinSize sets the minimum frame size.
func (w NSWindow) SetMinSize(size objc.CGSize) { sendVoid(w.ID(), "setMinSize:", size) }

// SetContentMinSize sets the minimum content size.
func (w NSWindow) SetContentMinSize(size objc.CGSize) { sendVoid(w.ID(), "setContentMinSize:", size) }

// SetContentSize sets the content size.
func (w NSWindow) SetContentSize(size objc.CGSize) { sendVoid(w.ID(), "setContentSize:", size) }

// MakeFirstResponder sets the first responder.
func (w NSWindow) MakeFirstResponder(responder objc.ID) bool {
	return send[bool](w.ID(), "makeFirstResponder:", responder)
}

// SetAppearance sets the window's appearance; nil follows the application.
func (w NSWindow) SetAppearance(appearance NSAppearance) {
	sendVoid(w.ID(), "setAppearance:", appearance.ID())
}

// EffectiveAppearance returns the window's effective appearance.
func (w NSWindow) EffectiveAppearance() NSAppearance {
	return NSAppearance(send[objc.ID](w.ID(), "effectiveAppearance"))
}

// MouseLocationOutsideOfEventStream returns the mouse location in window coordinates.
func (w NSWindow) MouseLocationOutsideOfEventStream() objc.CGPoint {
	return send[objc.CGPoint](w.ID(), "mouseLocationOutsideOfEventStream")
}

// ConvertPointToScreen converts a window point to screen coordinates.
func (w NSWindow) ConvertPointToScreen(point objc.CGPoint) objc.CGPoint {
	return send[objc.CGPoint](w.ID(), "convertPointToScreen:", point)
}

// ConvertPointFromScreen converts a screen point to window coordinates.
func (w NSWindow) ConvertPointFromScreen(point objc.CGPoint) objc.CGPoint {
	return send[objc.CGPoint](w.ID(), "convertPointFromScreen:", point)
}

// SetReleasedWhenClosed sets whether close releases the window.
func (w NSWindow) SetReleasedWhenClosed(flag bool) {
	sendVoid(w.ID(), "setReleasedWhenClosed:", flag)
}

// SetTabbingIdentifier sets the tabbing identifier.
func (w NSWindow) SetTabbingIdentifier(id string) {
	s := objc.NSStringFromGo(id)
	defer objc.Release(s)
	sendVoid(w.ID(), "setTabbingIdentifier:", s)
}
