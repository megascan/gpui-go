//go:build darwin

package uikit

import "github.com/megascan/gpui-go/internal/apple/objc"

// UITouch wraps a UITouch object.
type UITouch objc.ID

// ID returns the underlying object.
func (t UITouch) ID() objc.ID { return objc.ID(t) }

// LocationInView returns the touch position in view's coordinates.
func (t UITouch) LocationInView(view UIView) objc.CGPoint {
	return send[objc.CGPoint](t.ID(), "locationInView:", view.ID())
}

// Phase returns the touch phase.
func (t UITouch) Phase() UITouchPhase { return send[UITouchPhase](t.ID(), "phase") }

// Force returns the touch force.
func (t UITouch) Force() objc.CGFloat { return send[objc.CGFloat](t.ID(), "force") }

// MaximumPossibleForce returns the maximum force the device reports.
func (t UITouch) MaximumPossibleForce() objc.CGFloat {
	return send[objc.CGFloat](t.ID(), "maximumPossibleForce")
}

// UIGestureRecognizer wraps a UIGestureRecognizer object.
type UIGestureRecognizer objc.ID

// ID returns the underlying object.
func (r UIGestureRecognizer) ID() objc.ID { return objc.ID(r) }

// State returns the recognizer state.
func (r UIGestureRecognizer) State() UIGestureRecognizerState {
	return send[UIGestureRecognizerState](r.ID(), "state")
}

// LocationInView returns the gesture position in view's coordinates.
func (r UIGestureRecognizer) LocationInView(view UIView) objc.CGPoint {
	return send[objc.CGPoint](r.ID(), "locationInView:", view.ID())
}

// ModifierFlags returns the modifier keys held during the gesture.
func (r UIGestureRecognizer) ModifierFlags() UIKeyModifierFlags {
	return send[UIKeyModifierFlags](r.ID(), "modifierFlags")
}

// UIPanGestureRecognizer wraps a UIPanGestureRecognizer object.
type UIPanGestureRecognizer struct{ UIGestureRecognizer }

// TranslationInView returns the pan translation in view's coordinates.
func (r UIPanGestureRecognizer) TranslationInView(view UIView) objc.CGPoint {
	return send[objc.CGPoint](r.ID(), "translationInView:", view.ID())
}

// SetTranslationInView resets the pan translation.
func (r UIPanGestureRecognizer) SetTranslationInView(translation objc.CGPoint, view UIView) {
	sendVoid(r.ID(), "setTranslation:inView:", translation, view.ID())
}

// VelocityInView returns the pan velocity in points per second.
func (r UIPanGestureRecognizer) VelocityInView(view UIView) objc.CGPoint {
	return send[objc.CGPoint](r.ID(), "velocityInView:", view.ID())
}

// UIScreenEdgePanGestureRecognizer wraps a UIScreenEdgePanGestureRecognizer object.
type UIScreenEdgePanGestureRecognizer struct{ UIPanGestureRecognizer }

// Edges returns the edges the recognizer tracks.
func (r UIScreenEdgePanGestureRecognizer) Edges() UIRectEdge {
	return send[UIRectEdge](r.ID(), "edges")
}
