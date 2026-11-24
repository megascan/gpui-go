//go:build darwin

package ios

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
)

const backCommitVelocity = 300.

// BackEvent converts a step of the screen-edge back gesture, whose Started phase was already
// delivered when the recognizer asked to begin. Like UINavigationController, a release commits
// when the gesture is past halfway without flicking back, or when it flicks forward quickly. It
// reports false for states that carry no step.
func BackEvent(recognizer uikit.UIScreenEdgePanGestureRecognizer, view uikit.UIView, startPosition gpui.Point[gpui.Pixels]) (gpui.BackGestureEvent, bool) {
	direction := 1.
	if recognizer.Edges() == uikit.UIRectEdgeRight {
		direction = -1.
	}
	width := max(view.Bounds().Size.Width, 1.)
	travel := recognizer.TranslationInView(view).X * direction
	progress := min(max(travel/width, 0.), 1.)
	var phase gpui.BackGesturePhase
	switch recognizer.State() {
	case uikit.UIGestureRecognizerStateBegan, uikit.UIGestureRecognizerStateChanged:
		phase = gpui.BackGesturePhaseProgressed
	case uikit.UIGestureRecognizerStateEnded:
		velocity := recognizer.VelocityInView(view).X * direction
		phase = backReleasePhase(progress, velocity)
	case uikit.UIGestureRecognizerStateCancelled, uikit.UIGestureRecognizerStateFailed:
		phase = gpui.BackGesturePhaseCancelled
	default:
		return gpui.BackGestureEvent{}, false
	}
	return gpui.BackGestureEvent{
		Phase:         phase,
		StartPosition: startPosition,
		Progress:      float32(progress),
	}, true
}

func backReleasePhase(progress, velocity float64) gpui.BackGesturePhase {
	if velocity > backCommitVelocity || (progress > 0.5 && velocity > -backCommitVelocity) {
		return gpui.BackGesturePhaseCommitted
	}
	return gpui.BackGesturePhaseCancelled
}

// TouchLocationInView returns the touch position in view's coordinates.
func TouchLocationInView(touch uikit.UITouch, view uikit.UIView) gpui.Point[gpui.Pixels] {
	location := touch.LocationInView(view)
	return gpui.Point[gpui.Pixels]{X: gpui.Px(float32(location.X)), Y: gpui.Px(float32(location.Y))}
}

// TouchPhase converts the touch's UIKit phase.
func TouchPhase(touch uikit.UITouch) gpui.TouchPhase {
	return touchPhaseFromNative(touch.Phase())
}

func touchPhaseFromNative(phase uikit.UITouchPhase) gpui.TouchPhase {
	switch phase {
	case uikit.UITouchPhaseBegan:
		return gpui.TouchPhaseStarted
	case uikit.UITouchPhaseMoved, uikit.UITouchPhaseStationary:
		return gpui.TouchPhaseMoved
	case uikit.UITouchPhaseEnded:
		return gpui.TouchPhaseEnded
	default:
		return gpui.TouchPhaseCancelled
	}
}

// TouchID returns the touch's identity, its object address.
func TouchID(touch uikit.UITouch) gpui.TouchID {
	return gpui.TouchID(uint64(touch.ID()))
}

// ScrollEvent converts one step of a trackpad or mouse-wheel scroll gesture. The recognizer's
// translation is reset after each step, so every event carries only the movement since the
// previous one. It reports false for states that carry no scroll.
func ScrollEvent(recognizer uikit.UIPanGestureRecognizer, view uikit.UIView) (gpui.ScrollEvent, bool) {
	var touchPhase gpui.TouchPhase
	switch recognizer.State() {
	case uikit.UIGestureRecognizerStateBegan:
		touchPhase = gpui.TouchPhaseStarted
	case uikit.UIGestureRecognizerStateChanged:
		touchPhase = gpui.TouchPhaseMoved
	case uikit.UIGestureRecognizerStateEnded:
		touchPhase = gpui.TouchPhaseEnded
	case uikit.UIGestureRecognizerStateCancelled, uikit.UIGestureRecognizerStateFailed:
		touchPhase = gpui.TouchPhaseCancelled
	default:
		return gpui.ScrollEvent{}, false
	}
	translation := recognizer.TranslationInView(view)
	recognizer.SetTranslationInView(objc.CGPoint{}, view)
	location := recognizer.LocationInView(view)
	return gpui.ScrollEvent{
		Position: gpui.Point[gpui.Pixels]{X: gpui.Px(float32(location.X)), Y: gpui.Px(float32(location.Y))},
		Delta: gpui.ScrollDeltaPixels(gpui.Point[gpui.Pixels]{
			X: gpui.Px(float32(translation.X)),
			Y: gpui.Px(float32(translation.Y)),
		}),
		Modifiers:  modifiers(recognizer.ModifierFlags()),
		TouchPhase: touchPhase,
		Source:     gpui.ScrollSourcePointer,
	}, true
}

func modifiers(flags uikit.UIKeyModifierFlags) gpui.Modifiers {
	return gpui.Modifiers{
		Control:  flags.Contains(uikit.UIKeyModifierControl),
		Alt:      flags.Contains(uikit.UIKeyModifierAlternate),
		Shift:    flags.Contains(uikit.UIKeyModifierShift),
		Platform: flags.Contains(uikit.UIKeyModifierCommand),
		Function: false,
	}
}

// TouchForce returns the touch force normalized to 0..1, or false when the device reports none.
func TouchForce(touch uikit.UITouch) (float32, bool) {
	return touchForce(touch.Force(), touch.MaximumPossibleForce())
}

func touchForce(force, maximumForce float64) (float32, bool) {
	if maximumForce <= 0.0 {
		return 0, false
	}
	return float32(min(max(force/maximumForce, 0.0), 1.0)), true
}
