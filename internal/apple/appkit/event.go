//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSEventType is NSEvent's event type.
type NSEventType uint

// NSEventType values.
const (
	NSEventTypeLeftMouseDown      NSEventType = 1
	NSEventTypeLeftMouseUp        NSEventType = 2
	NSEventTypeRightMouseDown     NSEventType = 3
	NSEventTypeRightMouseUp       NSEventType = 4
	NSEventTypeMouseMoved         NSEventType = 5
	NSEventTypeLeftMouseDragged   NSEventType = 6
	NSEventTypeRightMouseDragged  NSEventType = 7
	NSEventTypeMouseEntered       NSEventType = 8
	NSEventTypeMouseExited        NSEventType = 9
	NSEventTypeKeyDown            NSEventType = 10
	NSEventTypeKeyUp              NSEventType = 11
	NSEventTypeFlagsChanged       NSEventType = 12
	NSEventTypeAppKitDefined      NSEventType = 13
	NSEventTypeSystemDefined      NSEventType = 14
	NSEventTypeApplicationDefined NSEventType = 15
	NSEventTypePeriodic           NSEventType = 16
	NSEventTypeCursorUpdate       NSEventType = 17
	NSEventTypeRotate             NSEventType = 18
	NSEventTypeBeginGesture       NSEventType = 19
	NSEventTypeEndGesture         NSEventType = 20
	NSEventTypeScrollWheel        NSEventType = 22
	NSEventTypeTabletPoint        NSEventType = 23
	NSEventTypeTabletProximity    NSEventType = 24
	NSEventTypeOtherMouseDown     NSEventType = 25
	NSEventTypeOtherMouseUp       NSEventType = 26
	NSEventTypeOtherMouseDragged  NSEventType = 27
	NSEventTypeGesture            NSEventType = 29
	NSEventTypeMagnify            NSEventType = 30
	NSEventTypeSwipe              NSEventType = 31
	NSEventTypeSmartMagnify       NSEventType = 32
	NSEventTypeQuickLook          NSEventType = 33
	NSEventTypePressure           NSEventType = 34
	NSEventTypeDirectTouch        NSEventType = 37
	NSEventTypeChangeMode         NSEventType = 38
	NSEventTypeMouseCancelled     NSEventType = 40
)

// NSEventModifierFlags is NSEvent's modifier flag set.
type NSEventModifierFlags uint

// NSEventModifierFlags values.
const (
	NSEventModifierFlagCapsLock                   NSEventModifierFlags = 1 << 16
	NSEventModifierFlagShift                      NSEventModifierFlags = 1 << 17
	NSEventModifierFlagControl                    NSEventModifierFlags = 1 << 18
	NSEventModifierFlagOption                     NSEventModifierFlags = 1 << 19
	NSEventModifierFlagCommand                    NSEventModifierFlags = 1 << 20
	NSEventModifierFlagNumericPad                 NSEventModifierFlags = 1 << 21
	NSEventModifierFlagHelp                       NSEventModifierFlags = 1 << 22
	NSEventModifierFlagFunction                   NSEventModifierFlags = 1 << 23
	NSEventModifierFlagDeviceIndependentFlagsMask NSEventModifierFlags = 0xffff0000
)

// Contains reports whether every flag in other is set in f.
func (f NSEventModifierFlags) Contains(other NSEventModifierFlags) bool {
	return f&other == other
}

// NSEventPhase is the phase of a gesture or scroll event.
type NSEventPhase uint

// NSEventPhase values.
const (
	NSEventPhaseNone       NSEventPhase = 0
	NSEventPhaseBegan      NSEventPhase = 1 << 0
	NSEventPhaseStationary NSEventPhase = 1 << 1
	NSEventPhaseChanged    NSEventPhase = 1 << 2
	NSEventPhaseEnded      NSEventPhase = 1 << 3
	NSEventPhaseCancelled  NSEventPhase = 1 << 4
	NSEventPhaseMayBegin   NSEventPhase = 1 << 5
)

// Function key characters reported by NSEvent.
const (
	NSUpArrowFunctionKey      uint32 = 0xF700
	NSDownArrowFunctionKey    uint32 = 0xF701
	NSLeftArrowFunctionKey    uint32 = 0xF702
	NSRightArrowFunctionKey   uint32 = 0xF703
	NSF1FunctionKey           uint32 = 0xF704
	NSF2FunctionKey           uint32 = 0xF705
	NSF3FunctionKey           uint32 = 0xF706
	NSF4FunctionKey           uint32 = 0xF707
	NSF5FunctionKey           uint32 = 0xF708
	NSF6FunctionKey           uint32 = 0xF709
	NSF7FunctionKey           uint32 = 0xF70A
	NSF8FunctionKey           uint32 = 0xF70B
	NSF9FunctionKey           uint32 = 0xF70C
	NSF10FunctionKey          uint32 = 0xF70D
	NSF11FunctionKey          uint32 = 0xF70E
	NSF12FunctionKey          uint32 = 0xF70F
	NSF13FunctionKey          uint32 = 0xF710
	NSF14FunctionKey          uint32 = 0xF711
	NSF15FunctionKey          uint32 = 0xF712
	NSF16FunctionKey          uint32 = 0xF713
	NSF17FunctionKey          uint32 = 0xF714
	NSF18FunctionKey          uint32 = 0xF715
	NSF19FunctionKey          uint32 = 0xF716
	NSF20FunctionKey          uint32 = 0xF717
	NSF21FunctionKey          uint32 = 0xF718
	NSF22FunctionKey          uint32 = 0xF719
	NSF23FunctionKey          uint32 = 0xF71A
	NSF24FunctionKey          uint32 = 0xF71B
	NSF25FunctionKey          uint32 = 0xF71C
	NSF26FunctionKey          uint32 = 0xF71D
	NSF27FunctionKey          uint32 = 0xF71E
	NSF28FunctionKey          uint32 = 0xF71F
	NSF29FunctionKey          uint32 = 0xF720
	NSF30FunctionKey          uint32 = 0xF721
	NSF31FunctionKey          uint32 = 0xF722
	NSF32FunctionKey          uint32 = 0xF723
	NSF33FunctionKey          uint32 = 0xF724
	NSF34FunctionKey          uint32 = 0xF725
	NSF35FunctionKey          uint32 = 0xF726
	NSInsertFunctionKey       uint32 = 0xF727
	NSDeleteFunctionKey       uint32 = 0xF728
	NSHomeFunctionKey         uint32 = 0xF729
	NSBeginFunctionKey        uint32 = 0xF72A
	NSEndFunctionKey          uint32 = 0xF72B
	NSPageUpFunctionKey       uint32 = 0xF72C
	NSPageDownFunctionKey     uint32 = 0xF72D
	NSPrintScreenFunctionKey  uint32 = 0xF72E
	NSScrollLockFunctionKey   uint32 = 0xF72F
	NSPauseFunctionKey        uint32 = 0xF730
	NSSysReqFunctionKey       uint32 = 0xF731
	NSBreakFunctionKey        uint32 = 0xF732
	NSResetFunctionKey        uint32 = 0xF733
	NSStopFunctionKey         uint32 = 0xF734
	NSMenuFunctionKey         uint32 = 0xF735
	NSUserFunctionKey         uint32 = 0xF736
	NSSystemFunctionKey       uint32 = 0xF737
	NSPrintFunctionKey        uint32 = 0xF738
	NSClearLineFunctionKey    uint32 = 0xF739
	NSClearDisplayFunctionKey uint32 = 0xF73A
	NSInsertLineFunctionKey   uint32 = 0xF73B
	NSDeleteLineFunctionKey   uint32 = 0xF73C
	NSInsertCharFunctionKey   uint32 = 0xF73D
	NSDeleteCharFunctionKey   uint32 = 0xF73E
	NSPrevFunctionKey         uint32 = 0xF73F
	NSNextFunctionKey         uint32 = 0xF740
	NSSelectFunctionKey       uint32 = 0xF741
	NSExecuteFunctionKey      uint32 = 0xF742
	NSUndoFunctionKey         uint32 = 0xF743
	NSRedoFunctionKey         uint32 = 0xF744
	NSFindFunctionKey         uint32 = 0xF745
	NSHelpFunctionKey         uint32 = 0xF746
	NSModeSwitchFunctionKey   uint32 = 0xF747
)

// NSEvent wraps an NSEvent object.
type NSEvent objc.ID

// ID returns the underlying object.
func (e NSEvent) ID() objc.ID { return objc.ID(e) }

// Type returns the event's type.
func (e NSEvent) Type() NSEventType { return send[NSEventType](e.ID(), "type") }

// ModifierFlags returns the modifier keys held for the event.
func (e NSEvent) ModifierFlags() NSEventModifierFlags {
	return send[NSEventModifierFlags](e.ID(), "modifierFlags")
}

// IsARepeat reports whether a key event is an auto-repeat.
func (e NSEvent) IsARepeat() bool { return send[bool](e.ID(), "isARepeat") }

// ButtonNumber returns the mouse button number.
func (e NSEvent) ButtonNumber() int { return send[int](e.ID(), "buttonNumber") }

// LocationInWindow returns the event location in window coordinates.
func (e NSEvent) LocationInWindow() objc.CGPoint {
	return send[objc.CGPoint](e.ID(), "locationInWindow")
}

// ClickCount returns the number of mouse clicks.
func (e NSEvent) ClickCount() int { return send[int](e.ID(), "clickCount") }

// Stage returns the pressure stage.
func (e NSEvent) Stage() int { return send[int](e.ID(), "stage") }

// Pressure returns the pressure of a pressure event.
func (e NSEvent) Pressure() float32 { return send[float32](e.ID(), "pressure") }

// Phase returns the gesture phase.
func (e NSEvent) Phase() NSEventPhase { return send[NSEventPhase](e.ID(), "phase") }

// MomentumPhase returns the momentum scroll phase.
func (e NSEvent) MomentumPhase() NSEventPhase { return send[NSEventPhase](e.ID(), "momentumPhase") }

// DeltaX returns the horizontal delta.
func (e NSEvent) DeltaX() objc.CGFloat { return send[objc.CGFloat](e.ID(), "deltaX") }

// DeltaY returns the vertical delta.
func (e NSEvent) DeltaY() objc.CGFloat { return send[objc.CGFloat](e.ID(), "deltaY") }

// Magnification returns the magnification of a magnify event.
func (e NSEvent) Magnification() objc.CGFloat { return send[objc.CGFloat](e.ID(), "magnification") }

// ScrollingDeltaX returns the horizontal scroll delta.
func (e NSEvent) ScrollingDeltaX() objc.CGFloat {
	return send[objc.CGFloat](e.ID(), "scrollingDeltaX")
}

// ScrollingDeltaY returns the vertical scroll delta.
func (e NSEvent) ScrollingDeltaY() objc.CGFloat {
	return send[objc.CGFloat](e.ID(), "scrollingDeltaY")
}

// HasPreciseScrollingDeltas reports whether the scroll deltas are in pixels.
func (e NSEvent) HasPreciseScrollingDeltas() bool {
	return send[bool](e.ID(), "hasPreciseScrollingDeltas")
}

// Characters returns the characters of a key event as an NSString, or nil.
func (e NSEvent) Characters() objc.ID { return send[objc.ID](e.ID(), "characters") }

// CharactersIgnoringModifiers returns the characters of a key event ignoring modifiers.
func (e NSEvent) CharactersIgnoringModifiers() objc.ID {
	return send[objc.ID](e.ID(), "charactersIgnoringModifiers")
}

// KeyCode returns the virtual key code of a key event.
func (e NSEvent) KeyCode() uint16 { return send[uint16](e.ID(), "keyCode") }

// Window returns the window the event belongs to.
func (e NSEvent) Window() NSWindow { return NSWindow(send[objc.ID](e.ID(), "window")) }

// Timestamp returns the event time in seconds since system startup.
func (e NSEvent) Timestamp() float64 { return send[float64](e.ID(), "timestamp") }

// NSEventMouseLocation returns the current mouse location in screen coordinates.
func NSEventMouseLocation() objc.CGPoint {
	return sendClass[objc.CGPoint]("NSEvent", "mouseLocation")
}

// NSEventPressedMouseButtons returns a bit mask of the mouse buttons currently down.
func NSEventPressedMouseButtons() uint {
	return sendClass[uint]("NSEvent", "pressedMouseButtons")
}

// NSEventModifierFlagsNow returns the modifier keys currently held.
func NSEventModifierFlagsNow() NSEventModifierFlags {
	return sendClass[NSEventModifierFlags]("NSEvent", "modifierFlags")
}
