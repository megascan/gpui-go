package gpui

// InputEvent is an event from a platform input source.
type InputEvent interface {
	// ToPlatformInput converts the event into the platform input enum.
	ToPlatformInput() PlatformInput
	sealedInputEvent()
}

// KeyEvent is a key event from the platform.
type KeyEvent interface {
	InputEvent
	isKeyEvent()
}

// MouseEvent is a mouse event from the platform.
type MouseEvent interface {
	InputEvent
	isMouseEvent()
}

// GestureEvent is a gesture event from the platform.
type GestureEvent interface {
	InputEvent
	isGestureEvent()
}

// KeyDownEvent is the key down event equivalent for the platform.
type KeyDownEvent struct {
	// Keystroke is the keystroke that was generated.
	Keystroke Keystroke
	// IsHeld reports whether the key is currently held down.
	IsHeld bool
	// PreferCharacterInput prefers character input over keybindings for this keystroke, as
	// for AltGr on Windows.
	PreferCharacterInput bool
}

// KeyUpEvent is the key up event equivalent for the platform.
type KeyUpEvent struct {
	// Keystroke is the keystroke that was released.
	Keystroke Keystroke
}

// ModifiersChangedEvent is the modifiers changed event equivalent for the platform. It embeds
// the new modifier state (upstream's Deref to Modifiers).
type ModifiersChangedEvent struct {
	Modifiers
	// Capslock is the new state of the capslock key.
	Capslock Capslock
}

// TouchPhase is the phase of a touch motion event. The zero value is TouchPhaseMoved,
// upstream's default.
type TouchPhase uint8

const (
	// TouchPhaseMoved means the touch is moving.
	TouchPhaseMoved TouchPhase = iota
	// TouchPhaseStarted means the touch started.
	TouchPhaseStarted
	// TouchPhaseEnded means the touch ended.
	TouchPhaseEnded
	// TouchPhaseCancelled means the system took the touch and it will not end normally;
	// consumers must unwind any in-progress interaction.
	TouchPhaseCancelled
)

// TouchID identifies one touch for its lifetime; the platform assigns it and never reuses it
// for a later touch in the same window.
type TouchID uint64

// TouchEvent is a raw touch event from the platform. A touch is hit-tested once, at
// TouchPhaseStarted; later events for the same TouchID go to the elements under the starting
// position.
type TouchEvent struct {
	// ID is the touch this event belongs to.
	ID TouchID
	// Phase is the phase of the touch.
	Phase TouchPhase
	// Position is the position of the touch in window coordinates.
	Position Point[Pixels]
	// PredictedPosition is where the platform predicts the touch will be about one frame
	// from now, for TouchPhaseMoved events; latency compensation only.
	PredictedPosition Option[Point[Pixels]]
	// Force is the normalized touch force in 0..=1, if the hardware reports it.
	Force Option[float32]
}

// MouseDownEvent is a mouse down event from the platform.
type MouseDownEvent struct {
	// Button is the mouse button that was pressed.
	Button MouseButton
	// Position is the position of the mouse on the window.
	Position Point[Pixels]
	// Modifiers are the modifiers held down when the mouse was pressed.
	Modifiers Modifiers
	// ClickCount is the number of times the button has been clicked.
	ClickCount int
	// FirstMouse reports whether this is the first, focusing click.
	FirstMouse bool
}

// IsFocusing reports whether this mouse event should focus the element.
func (e MouseDownEvent) IsFocusing() bool {
	return e.Button == MouseButtonLeft
}

// MouseUpEvent is a mouse up event from the platform.
type MouseUpEvent struct {
	// Button is the mouse button that was released.
	Button MouseButton
	// Position is the position of the mouse on the window.
	Position Point[Pixels]
	// Modifiers are the modifiers held down when the mouse was released.
	Modifiers Modifiers
	// ClickCount is the number of times the button has been clicked.
	ClickCount int
}

// IsFocusing reports whether this mouse up event should focus the element.
func (e MouseUpEvent) IsFocusing() bool {
	return e.Button == MouseButtonLeft
}

// MouseClickEvent is a click generated when a mouse button is pressed and released.
type MouseClickEvent struct {
	// Down is the mouse event when the button was pressed.
	Down MouseDownEvent
	// Up is the mouse event when the button was released.
	Up MouseUpEvent
}

// PressureStage is the stage of a pressure click event.
type PressureStage uint8

const (
	// PressureStageZero is no pressure.
	PressureStageZero PressureStage = iota
	// PressureStageNormal is normal click pressure.
	PressureStageNormal
	// PressureStageForce is enough pressure to trigger a force click.
	PressureStageForce
)

// MousePressureEvent is generated when a force-sensitive trackpad is pressed hard (macOS only).
type MousePressureEvent struct {
	// Pressure is the pressure of the current stage, between 0 and 1.
	Pressure float32
	// Stage is the pressure stage of the event.
	Stage PressureStage
	// Position is the position of the mouse on the window.
	Position Point[Pixels]
	// Modifiers are the modifiers held down when the pressure changed.
	Modifiers Modifiers
}

// KeyboardClickEvent is a click generated by a keyboard button being pressed and released.
type KeyboardClickEvent struct {
	// Button is the keyboard button that triggered the click.
	Button KeyboardButton
	// Bounds are the bounds of the element that was clicked.
	Bounds Bounds[Pixels]
}

// TouchClickEvent is a click generated by a recognized tap gesture on a touch screen.
type TouchClickEvent struct {
	// Position is the position of the tap in window coordinates.
	Position Point[Pixels]
	// TapCount is the number of consecutive taps at this location.
	TapCount int
	// LongPress reports a long press, touch's secondary activation, delivered to aux-click
	// listeners rather than primary click listeners.
	LongPress bool
}

// ClickEventKind selects the variant of a ClickEvent.
type ClickEventKind uint8

const (
	// ClickEventKindKeyboard is a click from a keyboard button; it is the zero value, as
	// upstream's default.
	ClickEventKindKeyboard ClickEventKind = iota
	// ClickEventKindMouse is a click from a mouse button.
	ClickEventKindMouse
	// ClickEventKindTouch is a click from a tap gesture.
	ClickEventKindTouch
)

// ClickEvent is generated when a mouse or keyboard button is pressed and released, or when a
// tap is recognized. Kind selects which of Mouse, Keyboard and Touch is set.
type ClickEvent struct {
	// Kind selects the variant.
	Kind ClickEventKind
	// Mouse is the mouse variant.
	Mouse MouseClickEvent
	// Keyboard is the keyboard variant.
	Keyboard KeyboardClickEvent
	// Touch is the touch variant.
	Touch TouchClickEvent
}

// ClickEventMouse returns the Mouse variant.
func ClickEventMouse(event MouseClickEvent) ClickEvent {
	return ClickEvent{Kind: ClickEventKindMouse, Mouse: event}
}

// ClickEventKeyboard returns the Keyboard variant.
func ClickEventKeyboard(event KeyboardClickEvent) ClickEvent {
	return ClickEvent{Kind: ClickEventKindKeyboard, Keyboard: event}
}

// ClickEventTouch returns the Touch variant.
func ClickEventTouch(event TouchClickEvent) ClickEvent {
	return ClickEvent{Kind: ClickEventKindTouch, Touch: event}
}

// Modifiers returns the modifiers held during the click: none for keyboard and touch clicks,
// the mouse up event's modifiers for mouse clicks.
func (e ClickEvent) Modifiers() Modifiers {
	switch e.Kind {
	case ClickEventKindMouse:
		return e.Mouse.Up.Modifiers
	default:
		return Modifiers{}
	}
}

// Position returns the bottom left corner of the clicked hitbox for keyboard clicks, the mouse
// position at release for mouse clicks and the tap position for touch clicks.
func (e ClickEvent) Position() Point[Pixels] {
	switch e.Kind {
	case ClickEventKindMouse:
		return e.Mouse.Up.Position
	case ClickEventKindTouch:
		return e.Touch.Position
	default:
		return e.Keyboard.Bounds.BottomLeft()
	}
}

// MousePosition returns the mouse position at release for mouse clicks; ok is false otherwise.
func (e ClickEvent) MousePosition() (Point[Pixels], bool) {
	if e.Kind == ClickEventKindMouse {
		return e.Mouse.Up.Position, true
	}
	return Point[Pixels]{}, false
}

// IsRightClick reports whether the right button was pressed and released.
func (e ClickEvent) IsRightClick() bool {
	if e.Kind == ClickEventKindMouse {
		return e.Mouse.Down.Button == MouseButtonRight && e.Mouse.Up.Button == MouseButtonRight
	}
	return false
}

// IsMiddleClick reports whether the middle button was pressed and released.
func (e ClickEvent) IsMiddleClick() bool {
	if e.Kind == ClickEventKindMouse {
		return e.Mouse.Down.Button == MouseButtonMiddle && e.Mouse.Up.Button == MouseButtonMiddle
	}
	return false
}

// IsSecondary reports whether the click is a secondary activation: a right click or a long press.
func (e ClickEvent) IsSecondary() bool {
	switch e.Kind {
	case ClickEventKindMouse:
		return e.Mouse.Down.Button == MouseButtonRight && e.Mouse.Up.Button == MouseButtonRight
	case ClickEventKindTouch:
		return e.Touch.LongPress
	default:
		return false
	}
}

// StandardClick reports whether the click was a standard click: always for keyboard clicks,
// a left click for mouse clicks, a tap that is not a long press for touch clicks.
func (e ClickEvent) StandardClick() bool {
	switch e.Kind {
	case ClickEventKindMouse:
		return e.Mouse.Down.Button == MouseButtonLeft && e.Mouse.Up.Button == MouseButtonLeft
	case ClickEventKindTouch:
		return !e.Touch.LongPress
	default:
		return true
	}
}

// FirstFocus reports whether the click focused the element; only mouse clicks can.
func (e ClickEvent) FirstFocus() bool {
	if e.Kind == ClickEventKindMouse {
		return e.Mouse.Down.FirstMouse
	}
	return false
}

// ClickCount returns 1 for keyboard clicks, the mouse up click count for mouse clicks and the
// tap count for touch clicks.
func (e ClickEvent) ClickCount() int {
	switch e.Kind {
	case ClickEventKindMouse:
		return e.Mouse.Up.ClickCount
	case ClickEventKindTouch:
		return e.Touch.TapCount
	default:
		return 1
	}
}

// IsKeyboard reports whether the click was generated by a keyboard event.
func (e ClickEvent) IsKeyboard() bool {
	return e.Kind == ClickEventKindKeyboard
}

// KeyboardButton is the keyboard button that was pressed for a click event.
type KeyboardButton uint8

const (
	// KeyboardButtonEnter is the enter key.
	KeyboardButtonEnter KeyboardButton = iota
	// KeyboardButtonSpace is the space key.
	KeyboardButtonSpace
)

// MouseButton is the mouse button that was pressed. Navigate(direction) is flattened into
// MouseButtonNavigateBack and MouseButtonNavigateForward.
type MouseButton uint8

const (
	// MouseButtonLeft is the left mouse button.
	MouseButtonLeft MouseButton = iota
	// MouseButtonRight is the right mouse button.
	MouseButtonRight
	// MouseButtonMiddle is the middle mouse button.
	MouseButtonMiddle
	// MouseButtonNavigateBack is the back navigation button.
	MouseButtonNavigateBack
	// MouseButtonNavigateForward is the forward navigation button.
	MouseButtonNavigateForward
)

// MouseButtonNavigate returns the navigation button for direction.
func MouseButtonNavigate(direction NavigationDirection) MouseButton {
	if direction == NavigationDirectionForward {
		return MouseButtonNavigateForward
	}
	return MouseButtonNavigateBack
}

// Navigation returns the direction of a navigation button; ok is false for other buttons.
func (b MouseButton) Navigation() (NavigationDirection, bool) {
	switch b {
	case MouseButtonNavigateBack:
		return NavigationDirectionBack, true
	case MouseButtonNavigateForward:
		return NavigationDirectionForward, true
	}
	return NavigationDirectionBack, false
}

// MouseButtonAll returns all the mouse buttons.
func MouseButtonAll() []MouseButton {
	return []MouseButton{
		MouseButtonLeft,
		MouseButtonRight,
		MouseButtonMiddle,
		MouseButtonNavigate(NavigationDirectionBack),
		MouseButtonNavigate(NavigationDirectionForward),
	}
}

// NavigationDirection is a navigation direction, such as back or forward.
type NavigationDirection uint8

const (
	// NavigationDirectionBack is the back button.
	NavigationDirectionBack NavigationDirection = iota
	// NavigationDirectionForward is the forward button.
	NavigationDirectionForward
)

// MouseMoveEvent is a mouse move event from the platform.
type MouseMoveEvent struct {
	// Position is the position of the mouse on the window.
	Position Point[Pixels]
	// PressedButton is the mouse button that is pressed, if any.
	PressedButton Option[MouseButton]
	// Modifiers are the modifiers held down when the mouse was moved.
	Modifiers Modifiers
}

// Dragging reports whether the left mouse button is held down.
func (e MouseMoveEvent) Dragging() bool {
	return e.PressedButton == Some(MouseButtonLeft)
}

// ScrollEvent is one step of scrolling from a mouse wheel, trackpad or touch pan. It embeds
// its modifiers (upstream's Deref to Modifiers).
type ScrollEvent struct {
	// Position is the position the scroll targets; for touch pans, where the pan started.
	Position Point[Pixels]
	// Delta is the change in scroll position for this event.
	Delta ScrollDelta
	Modifiers
	// TouchPhase is the phase of the scroll gesture.
	TouchPhase TouchPhase
	// Source is the kind of input that produced this scroll.
	Source ScrollSource
}

// ScrollWheelEvent is the former name of ScrollEvent.
//
// Deprecated: renamed to ScrollEvent.
type ScrollWheelEvent = ScrollEvent

// ScrollSource is the kind of input that produced a ScrollEvent.
type ScrollSource uint8

const (
	// ScrollSourcePointer is a scroll from a pointing device such as a wheel or trackpad.
	ScrollSourcePointer ScrollSource = iota
	// ScrollSourceTouch is a touchscreen pan recognized by GPUI, including its momentum.
	ScrollSourceTouch
)

// ScrollDelta is the scroll delta of a ScrollEvent: exact pixels or inexact lines. The zero
// value is Lines(0, 0), upstream's default.
type ScrollDelta struct {
	precise bool
	pixels  Point[Pixels]
	lines   Point[float32]
}

// ScrollDeltaPixels returns an exact scroll delta in pixels.
func ScrollDeltaPixels(delta Point[Pixels]) ScrollDelta {
	return ScrollDelta{precise: true, pixels: delta}
}

// ScrollDeltaLines returns an inexact scroll delta in lines.
func ScrollDeltaLines(delta Point[float32]) ScrollDelta {
	return ScrollDelta{lines: delta}
}

// AsPixels returns the pixel delta of a Pixels delta.
func (d ScrollDelta) AsPixels() (Point[Pixels], bool) {
	return d.pixels, d.precise
}

// AsLines returns the line delta of a Lines delta.
func (d ScrollDelta) AsLines() (Point[float32], bool) {
	return d.lines, !d.precise
}

// PinchEvent is a pinch gesture event from the platform. It embeds its modifiers (upstream's
// Deref to Modifiers).
type PinchEvent struct {
	// Position is the position of the pinch center on the window.
	Position Point[Pixels]
	// Delta is the zoom delta: positive zooms in, 0.1 is a 10% increase.
	Delta float32
	Modifiers
	// Phase is the phase of the pinch gesture.
	Phase TouchPhase
}

// Precise reports whether this is a precise scroll delta in pixels.
func (d ScrollDelta) Precise() bool {
	return d.precise
}

// PixelDelta converts the delta into exact pixels.
func (d ScrollDelta) PixelDelta(lineHeight Pixels) Point[Pixels] {
	if d.precise {
		return d.pixels
	}
	return NewPoint(Pixels(float32(lineHeight)*d.lines.X), Pixels(float32(lineHeight)*d.lines.Y))
}

// Coalesce combines two scroll deltas: per axis, deltas of the same sign are added and
// otherwise other wins; deltas of different kinds yield other.
func (d ScrollDelta) Coalesce(other ScrollDelta) ScrollDelta {
	switch {
	case d.precise && other.precise:
		a, b := d.pixels, other.pixels
		x := b.X
		if a.X.Signum() == b.X.Signum() {
			x = a.X + b.X
		}
		y := b.Y
		if a.Y.Signum() == b.Y.Signum() {
			y = a.Y + b.Y
		}
		return ScrollDeltaPixels(NewPoint(x, y))
	case !d.precise && !other.precise:
		a, b := d.lines, other.lines
		x := b.X
		if signumF32(a.X) == signumF32(b.X) {
			x = a.X + b.X
		}
		y := b.Y
		if signumF32(a.Y) == signumF32(b.Y) {
			y = a.Y + b.Y
		}
		return ScrollDeltaLines(NewPoint(x, y))
	default:
		return other
	}
}

// MouseExitEvent is generated when the mouse leaves the window. It embeds its modifiers
// (upstream's Deref to Modifiers).
type MouseExitEvent struct {
	// Position is the position of the mouse relative to the window.
	Position Point[Pixels]
	// PressedButton is the mouse button that is pressed, if any.
	PressedButton Option[MouseButton]
	Modifiers
}

// ExternalPaths is a collection of paths from the platform, such as from a file drop.
type ExternalPaths []string

// Paths returns the paths.
func (p ExternalPaths) Paths() []string {
	return p
}

// Render renders nothing: the platform renders icons for the dragged files.
func (p *ExternalPaths) Render(window *Window, cx *Context[ExternalPaths]) IntoElement {
	return Empty{}
}

// ExternalDragPayload is data offered to the platform when an internal drag leaves the window
// and is promoted to a native drag session.
type ExternalDragPayload interface {
	isExternalDragPayload()
}

// ExternalDragPayloadFiles is real on-disk paths, handed to the platform as an outbound file drag.
type ExternalDragPayloadFiles struct {
	FileDragPaths
}

func (ExternalDragPayloadFiles) isExternalDragPayload() {}

// FileDragPath is one entry of FileDragPaths: a path and whether it is a directory.
type FileDragPath struct {
	// Path is the dragged path.
	Path string
	// IsDir reports whether the path is a directory.
	IsDir bool
}

// FileDragPaths are the paths handed to the platform for a native file drag. The caller
// provides the directory metadata so it is not queried when the drag starts.
type FileDragPaths struct {
	entries []FileDragPath
}

// NewFileDragPaths creates a native file-drag payload from paths paired with whether each is a directory.
func NewFileDragPaths(entries []FileDragPath) FileDragPaths {
	return FileDragPaths{entries: append([]FileDragPath(nil), entries...)}
}

// Entries returns the dragged paths, each paired with whether it is a directory.
func (p FileDragPaths) Entries() []FileDragPath {
	return p.entries
}

// FileDropEvent is generated when files are dragged and dropped onto the window. The variants
// are FileDropEventEntered, FileDropEventPending, FileDropEventSubmit, FileDropEventExited and
// FileDropEventEnded.
type FileDropEvent interface {
	PlatformInput
	isFileDropEvent()
}

// FileDropEventEntered means the files have entered the window.
type FileDropEventEntered struct {
	// Position is the position of the mouse relative to the window.
	Position Point[Pixels]
	// Paths are the paths of the files being dragged.
	Paths ExternalPaths
}

// FileDropEventPending means the files are being dragged over the window.
type FileDropEventPending struct {
	// Position is the position of the mouse relative to the window.
	Position Point[Pixels]
}

// FileDropEventSubmit means the files have been dropped onto the window.
type FileDropEventSubmit struct {
	// Position is the position of the mouse relative to the window.
	Position Point[Pixels]
}

// FileDropEventExited means the user stopped dragging the files over the window.
type FileDropEventExited struct{}

// FileDropEventEnded means the platform-owned drag session has ended.
type FileDropEventEnded struct{}

func (FileDropEventEntered) isFileDropEvent() {}
func (FileDropEventPending) isFileDropEvent() {}
func (FileDropEventSubmit) isFileDropEvent()  {}
func (FileDropEventExited) isFileDropEvent()  {}
func (FileDropEventEnded) isFileDropEvent()   {}

// PlatformInput is any kind of platform input event. The variants are the event types
// themselves: KeyDownEvent, KeyUpEvent, ModifiersChangedEvent, MouseDownEvent, MouseUpEvent,
// MousePressureEvent, MouseMoveEvent, MouseExitEvent, ScrollEvent, PinchEvent, LongPressEvent,
// TouchDragEvent, BackGestureEvent, the FileDropEvent variants and TouchEvent.
type PlatformInput interface {
	isPlatformInput()
	// KindName returns a short static name for the input's variant, for diagnostics and telemetry.
	KindName() string
	// TouchEvent returns the touch event contained in the input, if any.
	TouchEvent() (TouchEvent, bool)
}

func platformInputMouseEvent(input PlatformInput) any {
	switch e := input.(type) {
	case MouseDownEvent:
		return &e
	case MouseUpEvent:
		return &e
	case MouseMoveEvent:
		return &e
	case MousePressureEvent:
		return &e
	case MouseExitEvent:
		return &e
	case ScrollEvent:
		return &e
	case PinchEvent:
		return &e
	case LongPressEvent:
		return &e
	case TouchDragEvent:
		return &e
	case BackGestureEvent:
		return &e
	case FileDropEvent:
		return &e
	}
	return nil
}

func platformInputKeyboardEvent(input PlatformInput) any {
	switch e := input.(type) {
	case KeyDownEvent:
		return &e
	case KeyUpEvent:
		return &e
	case ModifiersChangedEvent:
		return &e
	}
	return nil
}

func (KeyDownEvent) isPlatformInput() {}

// KindName returns "key_down".
func (KeyDownEvent) KindName() string { return "key_down" }

// TouchEvent returns false: the input is not a touch event.
func (KeyDownEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e KeyDownEvent) ToPlatformInput() PlatformInput { return e }

func (KeyDownEvent) sealedInputEvent() {}

func (KeyDownEvent) isKeyEvent() {}

func (KeyUpEvent) isPlatformInput() {}

// KindName returns "key_up".
func (KeyUpEvent) KindName() string { return "key_up" }

// TouchEvent returns false: the input is not a touch event.
func (KeyUpEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e KeyUpEvent) ToPlatformInput() PlatformInput { return e }

func (KeyUpEvent) sealedInputEvent() {}

func (KeyUpEvent) isKeyEvent() {}

func (ModifiersChangedEvent) isPlatformInput() {}

// KindName returns "modifiers_changed".
func (ModifiersChangedEvent) KindName() string { return "modifiers_changed" }

// TouchEvent returns false: the input is not a touch event.
func (ModifiersChangedEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e ModifiersChangedEvent) ToPlatformInput() PlatformInput { return e }

func (ModifiersChangedEvent) sealedInputEvent() {}

func (ModifiersChangedEvent) isKeyEvent() {}

func (MouseDownEvent) isPlatformInput() {}

// KindName returns "mouse_down".
func (MouseDownEvent) KindName() string { return "mouse_down" }

// TouchEvent returns false: the input is not a touch event.
func (MouseDownEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e MouseDownEvent) ToPlatformInput() PlatformInput { return e }

func (MouseDownEvent) sealedInputEvent() {}

func (MouseDownEvent) isMouseEvent() {}

func (MouseUpEvent) isPlatformInput() {}

// KindName returns "mouse_up".
func (MouseUpEvent) KindName() string { return "mouse_up" }

// TouchEvent returns false: the input is not a touch event.
func (MouseUpEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e MouseUpEvent) ToPlatformInput() PlatformInput { return e }

func (MouseUpEvent) sealedInputEvent() {}

func (MouseUpEvent) isMouseEvent() {}

func (MousePressureEvent) isPlatformInput() {}

// KindName returns "mouse_pressure".
func (MousePressureEvent) KindName() string { return "mouse_pressure" }

// TouchEvent returns false: the input is not a touch event.
func (MousePressureEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e MousePressureEvent) ToPlatformInput() PlatformInput { return e }

func (MousePressureEvent) sealedInputEvent() {}

func (MousePressureEvent) isMouseEvent() {}

func (MouseMoveEvent) isPlatformInput() {}

// KindName returns "mouse_move".
func (MouseMoveEvent) KindName() string { return "mouse_move" }

// TouchEvent returns false: the input is not a touch event.
func (MouseMoveEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e MouseMoveEvent) ToPlatformInput() PlatformInput { return e }

func (MouseMoveEvent) sealedInputEvent() {}

func (MouseMoveEvent) isMouseEvent() {}

func (MouseExitEvent) isPlatformInput() {}

// KindName returns "mouse_exited".
func (MouseExitEvent) KindName() string { return "mouse_exited" }

// TouchEvent returns false: the input is not a touch event.
func (MouseExitEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e MouseExitEvent) ToPlatformInput() PlatformInput { return e }

func (MouseExitEvent) sealedInputEvent() {}

func (MouseExitEvent) isMouseEvent() {}

func (ScrollEvent) isPlatformInput() {}

// KindName returns "scroll".
func (ScrollEvent) KindName() string { return "scroll" }

// TouchEvent returns false: the input is not a touch event.
func (ScrollEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e ScrollEvent) ToPlatformInput() PlatformInput { return e }

func (ScrollEvent) sealedInputEvent() {}

func (ScrollEvent) isMouseEvent() {}

func (PinchEvent) isPlatformInput() {}

// KindName returns "pinch".
func (PinchEvent) KindName() string { return "pinch" }

// TouchEvent returns false: the input is not a touch event.
func (PinchEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e PinchEvent) ToPlatformInput() PlatformInput { return e }

func (PinchEvent) sealedInputEvent() {}

func (PinchEvent) isMouseEvent() {}

func (PinchEvent) isGestureEvent() {}

func (LongPressEvent) isPlatformInput() {}

// KindName returns "long_press".
func (LongPressEvent) KindName() string { return "long_press" }

// TouchEvent returns false: the input is not a touch event.
func (LongPressEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

func (TouchDragEvent) isPlatformInput() {}

// KindName returns "touch_drag".
func (TouchDragEvent) KindName() string { return "touch_drag" }

// TouchEvent returns false: the input is not a touch event.
func (TouchDragEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

func (BackGestureEvent) isPlatformInput() {}

// KindName returns "back_gesture".
func (BackGestureEvent) KindName() string { return "back_gesture" }

// TouchEvent returns false: the input is not a touch event.
func (BackGestureEvent) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

func (FileDropEventEntered) isPlatformInput() {}

// KindName returns "file_drop".
func (FileDropEventEntered) KindName() string { return "file_drop" }

// TouchEvent returns false: the input is not a touch event.
func (FileDropEventEntered) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e FileDropEventEntered) ToPlatformInput() PlatformInput { return e }

func (FileDropEventEntered) sealedInputEvent() {}

func (FileDropEventEntered) isMouseEvent() {}

func (FileDropEventPending) isPlatformInput() {}

// KindName returns "file_drop".
func (FileDropEventPending) KindName() string { return "file_drop" }

// TouchEvent returns false: the input is not a touch event.
func (FileDropEventPending) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e FileDropEventPending) ToPlatformInput() PlatformInput { return e }

func (FileDropEventPending) sealedInputEvent() {}

func (FileDropEventPending) isMouseEvent() {}

func (FileDropEventSubmit) isPlatformInput() {}

// KindName returns "file_drop".
func (FileDropEventSubmit) KindName() string { return "file_drop" }

// TouchEvent returns false: the input is not a touch event.
func (FileDropEventSubmit) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e FileDropEventSubmit) ToPlatformInput() PlatformInput { return e }

func (FileDropEventSubmit) sealedInputEvent() {}

func (FileDropEventSubmit) isMouseEvent() {}

func (FileDropEventExited) isPlatformInput() {}

// KindName returns "file_drop".
func (FileDropEventExited) KindName() string { return "file_drop" }

// TouchEvent returns false: the input is not a touch event.
func (FileDropEventExited) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e FileDropEventExited) ToPlatformInput() PlatformInput { return e }

func (FileDropEventExited) sealedInputEvent() {}

func (FileDropEventExited) isMouseEvent() {}

func (FileDropEventEnded) isPlatformInput() {}

// KindName returns "file_drop".
func (FileDropEventEnded) KindName() string { return "file_drop" }

// TouchEvent returns false: the input is not a touch event.
func (FileDropEventEnded) TouchEvent() (TouchEvent, bool) { return TouchEvent{}, false }

// ToPlatformInput returns e as a PlatformInput.
func (e FileDropEventEnded) ToPlatformInput() PlatformInput { return e }

func (FileDropEventEnded) sealedInputEvent() {}

func (FileDropEventEnded) isMouseEvent() {}

func (TouchEvent) isPlatformInput() {}

// KindName returns "touch".
func (TouchEvent) KindName() string { return "touch" }

// TouchEvent returns e.
func (e TouchEvent) TouchEvent() (TouchEvent, bool) { return e, true }

// ToPlatformInput returns e as a PlatformInput.
func (e TouchEvent) ToPlatformInput() PlatformInput { return e }

func (TouchEvent) sealedInputEvent() {}
