//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSAutoresizingMaskOptions is a view's autoresizing mask.
type NSAutoresizingMaskOptions uint

// NSAutoresizingMaskOptions values.
const (
	NSViewNotSizable    NSAutoresizingMaskOptions = 0
	NSViewMinXMargin    NSAutoresizingMaskOptions = 1
	NSViewWidthSizable  NSAutoresizingMaskOptions = 2
	NSViewMaxXMargin    NSAutoresizingMaskOptions = 4
	NSViewMinYMargin    NSAutoresizingMaskOptions = 8
	NSViewHeightSizable NSAutoresizingMaskOptions = 16
	NSViewMaxYMargin    NSAutoresizingMaskOptions = 32
)

// NSTrackingAreaOptions configures an NSTrackingArea.
type NSTrackingAreaOptions uint

// NSTrackingAreaOptions values.
const (
	NSTrackingMouseEnteredAndExited    NSTrackingAreaOptions = 0x01
	NSTrackingMouseMoved               NSTrackingAreaOptions = 0x02
	NSTrackingCursorUpdate             NSTrackingAreaOptions = 0x04
	NSTrackingActiveWhenFirstResponder NSTrackingAreaOptions = 0x10
	NSTrackingActiveInKeyWindow        NSTrackingAreaOptions = 0x20
	NSTrackingActiveInActiveApp        NSTrackingAreaOptions = 0x40
	NSTrackingActiveAlways             NSTrackingAreaOptions = 0x80
	NSTrackingAssumeInside             NSTrackingAreaOptions = 0x100
	NSTrackingInVisibleRect            NSTrackingAreaOptions = 0x200
	NSTrackingEnabledDuringMouseDrag   NSTrackingAreaOptions = 0x400
)

// NSViewLayerContentsRedrawPolicy is a layer-backed view's redraw policy.
type NSViewLayerContentsRedrawPolicy int

// NSViewLayerContentsRedrawPolicy values.
const (
	NSViewLayerContentsRedrawNever             NSViewLayerContentsRedrawPolicy = 0
	NSViewLayerContentsRedrawOnSetNeedsDisplay NSViewLayerContentsRedrawPolicy = 1
	NSViewLayerContentsRedrawDuringViewResize  NSViewLayerContentsRedrawPolicy = 2
	NSViewLayerContentsRedrawBeforeViewResize  NSViewLayerContentsRedrawPolicy = 3
	NSViewLayerContentsRedrawCrossfade         NSViewLayerContentsRedrawPolicy = 4
)

// NSView wraps an NSView object.
type NSView objc.ID

// ID returns the underlying object.
func (v NSView) ID() objc.ID { return objc.ID(v) }

// InitWithFrame initialises an allocated view object.
func InitWithFrame(alloc objc.ID, frame objc.CGRect) NSView {
	return NSView(send[objc.ID](alloc, "initWithFrame:", frame))
}

// Frame returns the view frame.
func (v NSView) Frame() objc.CGRect { return send[objc.CGRect](v.ID(), "frame") }

// Bounds returns the view bounds.
func (v NSView) Bounds() objc.CGRect { return send[objc.CGRect](v.ID(), "bounds") }

// SetFrame sets the view frame.
func (v NSView) SetFrame(rect objc.CGRect) { sendVoid(v.ID(), "setFrame:", rect) }

// SetFrameSize sets the view frame size.
func (v NSView) SetFrameSize(size objc.CGSize) { sendVoid(v.ID(), "setFrameSize:", size) }

// Window returns the view's window or nil.
func (v NSView) Window() NSWindow { return NSWindow(send[objc.ID](v.ID(), "window")) }

// Superview returns the superview or nil.
func (v NSView) Superview() NSView { return NSView(send[objc.ID](v.ID(), "superview")) }

// AddSubview adds a subview.
func (v NSView) AddSubview(view NSView) { sendVoid(v.ID(), "addSubview:", view.ID()) }

// RemoveFromSuperview removes the view from its superview.
func (v NSView) RemoveFromSuperview() { sendVoid(v.ID(), "removeFromSuperview") }

// SetHidden sets whether the view is hidden.
func (v NSView) SetHidden(flag bool) { sendVoid(v.ID(), "setHidden:", flag) }

// SetWantsLayer sets whether the view is layer-backed.
func (v NSView) SetWantsLayer(flag bool) { sendVoid(v.ID(), "setWantsLayer:", flag) }

// Layer returns the backing layer.
func (v NSView) Layer() objc.ID { return send[objc.ID](v.ID(), "layer") }

// SetLayer sets the backing layer.
func (v NSView) SetLayer(layer objc.ID) { sendVoid(v.ID(), "setLayer:", layer) }

// SetLayerContentsRedrawPolicy sets the layer redraw policy.
func (v NSView) SetLayerContentsRedrawPolicy(policy NSViewLayerContentsRedrawPolicy) {
	sendVoid(v.ID(), "setLayerContentsRedrawPolicy:", policy)
}

// SetAutoresizingMask sets the autoresizing mask.
func (v NSView) SetAutoresizingMask(mask NSAutoresizingMaskOptions) {
	sendVoid(v.ID(), "setAutoresizingMask:", mask)
}

// SetNeedsDisplay marks the view for redisplay.
func (v NSView) SetNeedsDisplay(flag bool) { sendVoid(v.ID(), "setNeedsDisplay:", flag) }

// ConvertPointFromView converts a point from another view's coordinates.
func (v NSView) ConvertPointFromView(point objc.CGPoint, view NSView) objc.CGPoint {
	return send[objc.CGPoint](v.ID(), "convertPoint:fromView:", point, view.ID())
}

// ConvertPointToView converts a point to another view's coordinates.
func (v NSView) ConvertPointToView(point objc.CGPoint, view NSView) objc.CGPoint {
	return send[objc.CGPoint](v.ID(), "convertPoint:toView:", point, view.ID())
}

// AddTrackingArea adds a tracking area.
func (v NSView) AddTrackingArea(area objc.ID) { sendVoid(v.ID(), "addTrackingArea:", area) }

// RemoveTrackingArea removes a tracking area.
func (v NSView) RemoveTrackingArea(area objc.ID) { sendVoid(v.ID(), "removeTrackingArea:", area) }

// InputContext returns the view's NSTextInputContext.
func (v NSView) InputContext() objc.ID { return send[objc.ID](v.ID(), "inputContext") }

// EffectiveAppearance returns the view's effective appearance.
func (v NSView) EffectiveAppearance() NSAppearance {
	return NSAppearance(send[objc.ID](v.ID(), "effectiveAppearance"))
}

// NewTrackingArea returns a +1 NSTrackingArea.
func NewTrackingArea(rect objc.CGRect, options NSTrackingAreaOptions, owner objc.ID) objc.ID {
	alloc := objc.Alloc(class("NSTrackingArea"))
	return send[objc.ID](alloc, "initWithRect:options:owner:userInfo:", rect, options, owner, objc.ID(0))
}
