package gpui

import (
	"reflect"
	"strconv"
	"time"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

const dragThreshold float64 = 2.

const defaultTooltipShowDelay = 500 * time.Millisecond

const hoverableTooltipHideDelay = 500 * time.Millisecond

// GroupStyle is the styling information for a given group.
type GroupStyle struct {
	// Group is the identifier for this group.
	Group SharedString
	// Style is the style refinement this group applies to its children.
	Style *StyleRefinement
}

// DragMoveEvent is an event for when a drag is moving over this element, with the given state type.
type DragMoveEvent[T any] struct {
	// Event is the mouse move event that triggered this drag move event.
	Event MouseMoveEvent
	// Bounds are the bounds of this element.
	Bounds      Bounds[Pixels]
	draggedItem any
}

// Drag returns the drag state for this event.
func (e *DragMoveEvent[T]) Drag(cx *App) T {
	if drag, ok := cx.activeDrag.Get(); ok {
		if value, ok := drag.Value.(T); ok {
			return value
		}
	}
	panic("DragMoveEvent is only valid when the stored active drag is of the same type.")
}

// DraggedItem returns the item that is about to be dropped.
func (e *DragMoveEvent[T]) DraggedItem() any {
	return e.draggedItem
}

// NewInteractivity creates an Interactivity, capturing the caller location.
func NewInteractivity() *Interactivity {
	return &Interactivity{sourceLocation: CallerSourceLocation(1)}
}

// SourceLocation returns the source location of construction, or nil.
func (i *Interactivity) SourceLocation() *SourceLocation {
	return i.sourceLocation
}

// OnMouseDown binds listener to the mouse down event for button, during the bubble phase.
func (i *Interactivity) OnMouseDown(button MouseButton, listener func(event *MouseDownEvent, window *Window, cx *App)) {
	i.mouseDownListeners = append(i.mouseDownListeners, func(event *MouseDownEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && event.Button == button && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// CaptureAnyMouseDown binds listener to the mouse down event for any button, during the capture phase.
func (i *Interactivity) CaptureAnyMouseDown(listener func(event *MouseDownEvent, window *Window, cx *App)) {
	i.mouseDownListeners = append(i.mouseDownListeners, func(event *MouseDownEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnAnyMouseDown binds listener to the mouse down event for any button, during the bubble phase.
func (i *Interactivity) OnAnyMouseDown(listener func(event *MouseDownEvent, window *Window, cx *App)) {
	i.mouseDownListeners = append(i.mouseDownListeners, func(event *MouseDownEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnMousePressure binds listener to the mouse pressure event, during the bubble phase.
func (i *Interactivity) OnMousePressure(listener func(event *MousePressureEvent, window *Window, cx *App)) {
	i.mousePressureListeners = append(i.mousePressureListeners, func(event *MousePressureEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// CaptureMousePressure binds listener to the mouse pressure event, during the capture phase.
func (i *Interactivity) CaptureMousePressure(listener func(event *MousePressureEvent, window *Window, cx *App)) {
	i.mousePressureListeners = append(i.mousePressureListeners, func(event *MousePressureEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnMouseUp binds listener to the mouse up event for button, during the bubble phase.
func (i *Interactivity) OnMouseUp(button MouseButton, listener func(event *MouseUpEvent, window *Window, cx *App)) {
	i.mouseUpListeners = append(i.mouseUpListeners, func(event *MouseUpEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && event.Button == button && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// CaptureAnyMouseUp binds listener to the mouse up event for any button, during the capture phase.
func (i *Interactivity) CaptureAnyMouseUp(listener func(event *MouseUpEvent, window *Window, cx *App)) {
	i.mouseUpListeners = append(i.mouseUpListeners, func(event *MouseUpEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnAnyMouseUp binds listener to the mouse up event for any button, during the bubble phase.
func (i *Interactivity) OnAnyMouseUp(listener func(event *MouseUpEvent, window *Window, cx *App)) {
	i.mouseUpListeners = append(i.mouseUpListeners, func(event *MouseUpEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnMouseDownOut binds listener to the mouse down event for any button, during the capture phase,
// when the mouse is outside the bounds of this element.
func (i *Interactivity) OnMouseDownOut(listener func(event *MouseDownEvent, window *Window, cx *App)) {
	i.mouseDownListeners = append(i.mouseDownListeners, func(event *MouseDownEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture && !window.HasActivePrompt() && !hitbox.Contains(window.MousePosition()) {
			listener(event, window, cx)
		}
	})
}

// OnMouseUpOut binds listener to the mouse up event for button, during the capture phase, when
// the mouse is outside the bounds of this element.
func (i *Interactivity) OnMouseUpOut(button MouseButton, listener func(event *MouseUpEvent, window *Window, cx *App)) {
	i.mouseUpListeners = append(i.mouseUpListeners, func(event *MouseUpEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture && event.Button == button && !hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnMouseMove binds listener to the mouse move event, during the bubble phase.
func (i *Interactivity) OnMouseMove(listener func(event *MouseMoveEvent, window *Window, cx *App)) {
	i.mouseMoveListeners = append(i.mouseMoveListeners, func(event *MouseMoveEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnMouseExit binds listener to the mouse exit event, during the bubble phase.
func (i *Interactivity) OnMouseExit(listener func(event *MouseExitEvent, window *Window, cx *App)) {
	i.mouseExitListeners = append(i.mouseExitListeners, func(event *MouseExitEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnFileDropExit binds listener to FileDropEventExited when a platform file drag leaves this
// element's window while the element is hovered.
func (i *Interactivity) OnFileDropExit(listener func(event FileDropEvent, window *Window, cx *App)) {
	i.fileDropExitListeners = append(i.fileDropExitListeners, func(event FileDropEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if _, exited := event.(FileDropEventExited); phase == DispatchPhaseBubble && exited && hitbox.ID.isHoveredIgnoringLastInput(window) {
			listener(event, window, cx)
		}
	})
}

// OnDragMove binds listener to mouse moves while a drag of type T that started over this element
// is active, inside or outside the element.
func (i *Interactivity) OnDragMove[T any](listener func(event *DragMoveEvent[T], window *Window, cx *App)) {
	i.mouseMoveListeners = append(i.mouseMoveListeners, func(event *MouseMoveEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase != DispatchPhaseCapture {
			return
		}
		drag, ok := cx.activeDrag.Get()
		if !ok || reflect.TypeOf(drag.Value) != reflect.TypeFor[T]() {
			return
		}
		listener(&DragMoveEvent[T]{
			Event:       *event,
			Bounds:      hitbox.Bounds,
			draggedItem: drag.Value,
		}, window, cx)
	})
}

// OnScroll binds listener to scroll events during the bubble phase.
func (i *Interactivity) OnScroll(listener func(event *ScrollEvent, window *Window, cx *App)) {
	i.scrollListeners = append(i.scrollListeners, func(event *ScrollEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.ShouldHandleScroll(window) {
			listener(event, window, cx)
		}
	})
}

// OnScrollWheel is the former name of OnScroll.
//
// Deprecated: renamed to OnScroll.
func (i *Interactivity) OnScrollWheel(listener func(event *ScrollEvent, window *Window, cx *App)) {
	i.OnScroll(listener)
}

// OnPinch binds listener to pinch gesture events during the bubble phase.
func (i *Interactivity) OnPinch(listener func(event *PinchEvent, window *Window, cx *App)) {
	i.pinchListeners = append(i.pinchListeners, func(event *PinchEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
		}
	})
}

// OnBackGesture binds listener to back-navigation gestures that start over this element.
func (i *Interactivity) OnBackGesture(listener func(event *BackGestureEvent, window *Window, cx *App)) {
	i.backGestureListeners = append(i.backGestureListeners, func(event *BackGestureEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
			listener(event, window, cx)
			window.PreventDefault()
			cx.StopPropagation()
		}
	})
}

// CapturePinch binds listener to pinch gesture events during the capture phase.
func (i *Interactivity) CapturePinch(listener func(event *PinchEvent, window *Window, cx *App)) {
	i.pinchListeners = append(i.pinchListeners, func(event *PinchEvent, phase DispatchPhase, _ *Hitbox, window *Window, cx *App) {
		if phase == DispatchPhaseCapture {
			listener(event, window, cx)
		} else {
			cx.Propagate()
		}
	})
}

// CaptureAction binds listener to dispatches of action type A during the capture phase.
func (i *Interactivity) CaptureAction[A Action](listener func(action A, window *Window, cx *App)) {
	i.actionListeners = append(i.actionListeners, actionListenerEntry{
		actionType: reflect.TypeFor[A](),
		listener: func(action any, phase DispatchPhase, window *Window, cx *App) {
			a := action.(A)
			if phase == DispatchPhaseCapture {
				listener(a, window, cx)
			} else {
				cx.Propagate()
			}
		},
	})
}

// OnAction binds listener to dispatches of action type A during the bubble phase.
func (i *Interactivity) OnAction[A Action](listener func(action A, window *Window, cx *App)) {
	i.actionListeners = append(i.actionListeners, actionListenerEntry{
		actionType: reflect.TypeFor[A](),
		listener: func(action any, phase DispatchPhase, window *Window, cx *App) {
			a := action.(A)
			if phase == DispatchPhaseBubble {
				listener(a, window, cx)
			}
		},
	})
}

// OnBoxedAction binds listener to dispatches of action's type during the bubble phase; the
// listener receives a clone of action.
func (i *Interactivity) OnBoxedAction(action Action, listener func(action Action, window *Window, cx *App)) {
	action = action.BoxedClone()
	i.actionListeners = append(i.actionListeners, actionListenerEntry{
		actionType: reflect.TypeOf(action),
		listener: func(_ any, phase DispatchPhase, window *Window, cx *App) {
			if phase == DispatchPhaseBubble {
				listener(action, window, cx)
			}
		},
	})
}

// OnKeyDown binds listener to the key down event during the bubble phase.
func (i *Interactivity) OnKeyDown(listener func(event *KeyDownEvent, window *Window, cx *App)) {
	i.keyDownListeners = append(i.keyDownListeners, func(event *KeyDownEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseBubble {
			listener(event, window, cx)
		}
	})
}

// CaptureKeyDown binds listener to the key down event during the capture phase.
func (i *Interactivity) CaptureKeyDown(listener func(event *KeyDownEvent, window *Window, cx *App)) {
	i.keyDownListeners = append(i.keyDownListeners, func(event *KeyDownEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseCapture {
			listener(event, window, cx)
		}
	})
}

// OnKeyUp binds listener to the key up event during the bubble phase.
func (i *Interactivity) OnKeyUp(listener func(event *KeyUpEvent, window *Window, cx *App)) {
	i.keyUpListeners = append(i.keyUpListeners, func(event *KeyUpEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseBubble {
			listener(event, window, cx)
		}
	})
}

// CaptureKeyUp binds listener to the key up event during the capture phase.
func (i *Interactivity) CaptureKeyUp(listener func(event *KeyUpEvent, window *Window, cx *App)) {
	i.keyUpListeners = append(i.keyUpListeners, func(event *KeyUpEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseCapture {
			listener(event, window, cx)
		}
	})
}

// OnModifiersChanged binds listener to modifier changes.
func (i *Interactivity) OnModifiersChanged(listener func(event *ModifiersChangedEvent, window *Window, cx *App)) {
	i.modifiersChangedListeners = append(i.modifiersChangedListeners, listener)
}

// OnDrop binds listener to drops of a dragged value of type T on this element.
func (i *Interactivity) OnDrop[T any](listener func(value T, window *Window, cx *App)) {
	i.dropListeners = append(i.dropListeners, dropListenerEntry{
		valueType: reflect.TypeFor[T](),
		listener: func(draggedValue any, window *Window, cx *App) {
			listener(draggedValue.(T), window, cx)
		},
	})
}

// CanDrop sets a predicate deciding whether a dragged value can be dropped on this element.
func (i *Interactivity) CanDrop(predicate func(value any, window *Window, cx *App) bool) {
	i.canDropPredicate = predicate
}

// OnClick binds listener to the click event.
func (i *Interactivity) OnClick(listener func(event *ClickEvent, window *Window, cx *App)) {
	i.clickListeners = append(i.clickListeners, listener)
}

// OnAuxClick binds listener to the auxiliary (middle) click event.
func (i *Interactivity) OnAuxClick(listener func(event *ClickEvent, window *Window, cx *App)) {
	i.auxClickListeners = append(i.auxClickListeners, listener)
}

// OnDrag starts a drag of value when this element is dragged, rendering the view constructor builds.
func (i *Interactivity) OnDrag[T, W any](value T, constructor func(value T, offset Point[Pixels], window *Window, cx *App) Entity[W]) {
	if gpuiutil.DebugAssertions && i.dragListener != nil {
		panic("calling on_drag more than once on the same element is not supported")
	}
	i.dragListener = &dragListener{
		value: value,
		render: func(value any, offset Point[Pixels], window *Window, cx *App) AnyView {
			return AnyViewFromEntity(constructor(value.(T), offset, window, cx))
		},
	}
}

// ExternalDragPayload sets how the drag started by OnDrag resolves the payload handed to the
// platform when it leaves the window; T must be OnDrag's value type.
func (i *Interactivity) ExternalDragPayload[T any](resolver func(value T, window *Window, cx *App) (ExternalDragPayload, bool)) {
	listener := i.dragListener
	if listener == nil {
		if gpuiutil.DebugAssertions {
			panic("external_drag_payload must be called after on_drag")
		}
		return
	}
	if gpuiutil.DebugAssertions && reflect.TypeOf(listener.value) != reflect.TypeFor[T]() {
		panic("external_drag_payload must use the same dragged value type as on_drag")
	}
	if gpuiutil.DebugAssertions && listener.externalPayload != nil {
		panic("calling external_drag_payload more than once on the same element is not supported")
	}
	listener.externalPayload = func(value any, window *Window, cx *App) (ExternalDragPayload, bool) {
		v, ok := value.(T)
		if !ok {
			return nil, false
		}
		return resolver(v, window, cx)
	}
}

// OnHover binds listener to hover changes of this element.
func (i *Interactivity) OnHover(listener func(hovered bool, window *Window, cx *App)) {
	if gpuiutil.DebugAssertions && i.hoverListener != nil {
		panic("calling on_hover more than once on the same element is not supported")
	}
	i.hoverListener = listener
}

// HoverListenerMode sets how the hover listener decides whether the element is hovered.
func (i *Interactivity) HoverListenerMode(mode HoverListenerMode) {
	i.hoverListenerMode = mode
}

// Tooltip sets the tooltip shown while this element is hovered.
func (i *Interactivity) Tooltip(buildTooltip func(window *Window, cx *App) AnyView) {
	if gpuiutil.DebugAssertions && i.tooltipBuilder != nil {
		panic("calling tooltip more than once on the same element is not supported")
	}
	i.tooltipBuilder = &tooltipBuilder{build: buildTooltip, hoverable: false}
}

// HoverableTooltip sets a tooltip that stays visible while the mouse is over it.
func (i *Interactivity) HoverableTooltip(buildTooltip func(window *Window, cx *App) AnyView) {
	if gpuiutil.DebugAssertions && i.tooltipBuilder != nil {
		panic("calling tooltip more than once on the same element is not supported")
	}
	i.tooltipBuilder = &tooltipBuilder{build: buildTooltip, hoverable: true}
}

// TooltipShowDelay sets the delay before the tooltip is shown.
func (i *Interactivity) TooltipShowDelay(delay time.Duration) {
	i.tooltipShowDelay = Some(delay)
}

// OccludeMouse blocks mouse interactions with elements behind this one.
func (i *Interactivity) OccludeMouse() {
	i.hitboxBehavior = HitboxBehaviorBlockMouse
}

// WindowControlArea marks this element as a platform window control area.
func (i *Interactivity) WindowControlArea(area WindowControlArea) {
	i.windowControl = Some(area)
}

// BlockMouseExceptScroll blocks mouse interactions except scrolling with elements behind this one.
func (i *Interactivity) BlockMouseExceptScroll() {
	i.hitboxBehavior = HitboxBehaviorBlockMouseExceptScroll
}

func (i *Interactivity) hasPinchListeners() bool {
	return len(i.pinchListeners) != 0
}

// Group sets the group name other elements' group styles refer to.
func (m *Interactive[S]) Group(group SharedString) S {
	m.self.Interactivity().group = Some(group)
	return m.self
}

// ID assigns the element an id, making it stateful.
func (i *InteractiveID[S]) ID[I ElementIDSource](id I) *Stateful[S] {
	i.self.Interactivity().ElementID = Some(NewElementID(id))
	return newStateful(i.self)
}

func newStateful[E any](element E) *Stateful[E] {
	s := &Stateful[E]{element: element}
	BindStyled(&s.Styled, s)
	BindInteractive(&s.Interactive, s)
	BindStatefulInteractive(&s.StatefulInteractive, s)
	BindParent(&s.Parent, s)
	BindFluent(&s.Fluent, s)
	return s
}

// TrackFocus tracks handle's focus state for the element and makes it focusable.
func (m *Interactive[S]) TrackFocus(handle FocusHandle) S {
	i := m.self.Interactivity()
	i.focusable = true
	i.trackedFocusHandle = Some(handle.Clone())
	return m.self
}

// TabStop sets whether the element is a tab stop.
func (m *Interactive[S]) TabStop(tabStop bool) S {
	m.self.Interactivity().tabStop = tabStop
	return m.self
}

// TabIndex sets the element's tab index, making it focusable and a tab stop.
func (m *Interactive[S]) TabIndex(index int) S {
	i := m.self.Interactivity()
	i.focusable = true
	i.tabIndex = Some(index)
	i.tabStop = true
	return m.self
}

// TabGroup makes the element a tab group, with tab index 0 unless one is set.
func (m *Interactive[S]) TabGroup() S {
	i := m.self.Interactivity()
	i.tabGroup = true
	if i.tabIndex.IsNone() {
		i.tabIndex = Some(0)
	}
	return m.self
}

// KeyContext sets the element's key context; a string is parsed and a parse error is logged.
func (m *Interactive[S]) KeyContext[C ~string | KeyContext](keyContext C) S {
	switch v := any(keyContext).(type) {
	case KeyContext:
		m.self.Interactivity().keyContext = Some(v)
	default:
		parsed, err := KeyContextParse(reflect.ValueOf(keyContext).String())
		if err != nil {
			gpuiutil.LogErr(err)
		} else {
			m.self.Interactivity().keyContext = Some(parsed)
		}
	}
	return m.self
}

// Hover sets the style applied while the element is hovered.
func (m *Interactive[S]) Hover(f func(style *StyleRefinement) *StyleRefinement) S {
	i := m.self.Interactivity()
	if gpuiutil.DebugAssertions && i.hoverStyle != nil {
		panic("hover style already set")
	}
	i.hoverStyle = f(&StyleRefinement{})
	return m.self
}

// GroupHover sets the style applied while the named group is hovered.
func (m *Interactive[S]) GroupHover(groupName SharedString, f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().groupHoverStyle = &GroupStyle{Group: groupName, Style: f(&StyleRefinement{})}
	return m.self
}

// OnMouseDown binds listener to the mouse down event for button, during the bubble phase.
func (m *Interactive[S]) OnMouseDown(button MouseButton, listener func(event *MouseDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseDown(button, listener)
	return m.self
}

// DebugSelector sets a selector tests use to find the element; it is ignored without debug assertions.
func (m *Interactive[S]) DebugSelector(f func() string) S {
	if gpuiutil.DebugAssertions {
		m.self.Interactivity().debugSelector = Some(f())
	}
	return m.self
}

// CaptureAnyMouseDown binds listener to the mouse down event for any button, during the capture phase.
func (m *Interactive[S]) CaptureAnyMouseDown(listener func(event *MouseDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureAnyMouseDown(listener)
	return m.self
}

// OnAnyMouseDown binds listener to the mouse down event for any button, during the bubble phase.
func (m *Interactive[S]) OnAnyMouseDown(listener func(event *MouseDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnAnyMouseDown(listener)
	return m.self
}

// OnMouseUp binds listener to the mouse up event for button, during the bubble phase.
func (m *Interactive[S]) OnMouseUp(button MouseButton, listener func(event *MouseUpEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseUp(button, listener)
	return m.self
}

// CaptureAnyMouseUp binds listener to the mouse up event for any button, during the capture phase.
func (m *Interactive[S]) CaptureAnyMouseUp(listener func(event *MouseUpEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureAnyMouseUp(listener)
	return m.self
}

// OnMousePressure binds listener to the mouse pressure event, during the bubble phase.
func (m *Interactive[S]) OnMousePressure(listener func(event *MousePressureEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMousePressure(listener)
	return m.self
}

// CaptureMousePressure binds listener to the mouse pressure event, during the capture phase.
func (m *Interactive[S]) CaptureMousePressure(listener func(event *MousePressureEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureMousePressure(listener)
	return m.self
}

// OnMouseDownOut binds listener to mouse downs outside the element, during the capture phase.
func (m *Interactive[S]) OnMouseDownOut(listener func(event *MouseDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseDownOut(listener)
	return m.self
}

// OnMouseUpOut binds listener to mouse ups of button outside the element, during the capture phase.
func (m *Interactive[S]) OnMouseUpOut(button MouseButton, listener func(event *MouseUpEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseUpOut(button, listener)
	return m.self
}

// OnMouseMove binds listener to the mouse move event, during the bubble phase.
func (m *Interactive[S]) OnMouseMove(listener func(event *MouseMoveEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseMove(listener)
	return m.self
}

// OnMouseExit binds listener to the mouse exit event, during the bubble phase.
func (m *Interactive[S]) OnMouseExit(listener func(event *MouseExitEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnMouseExit(listener)
	return m.self
}

// OnFileDropExit binds listener to FileDropEventExited while the element is hovered.
func (m *Interactive[S]) OnFileDropExit(listener func(event FileDropEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnFileDropExit(listener)
	return m.self
}

// OnDragMove binds listener to mouse moves during a drag of type T.
func (m *Interactive[S]) OnDragMove[T any](listener func(event *DragMoveEvent[T], window *Window, cx *App)) S {
	m.self.Interactivity().OnDragMove(listener)
	return m.self
}

// OnScroll binds listener to scroll events during the bubble phase.
func (m *Interactive[S]) OnScroll(listener func(event *ScrollEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnScroll(listener)
	return m.self
}

// OnScrollWheel is the former name of OnScroll.
//
// Deprecated: renamed to OnScroll.
func (m *Interactive[S]) OnScrollWheel(listener func(event *ScrollEvent, window *Window, cx *App)) S {
	return m.OnScroll(listener)
}

// OnPinch binds listener to pinch gesture events during the bubble phase.
func (m *Interactive[S]) OnPinch(listener func(event *PinchEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnPinch(listener)
	return m.self
}

// OnBackGesture binds listener to back-navigation gestures that start over this element.
func (m *Interactive[S]) OnBackGesture(listener func(event *BackGestureEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnBackGesture(listener)
	return m.self
}

// CapturePinch binds listener to pinch gesture events during the capture phase.
func (m *Interactive[S]) CapturePinch(listener func(event *PinchEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CapturePinch(listener)
	return m.self
}

// CaptureAction binds listener to dispatches of action type A during the capture phase.
func (m *Interactive[S]) CaptureAction[A Action](listener func(action A, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureAction(listener)
	return m.self
}

// OnAction binds listener to dispatches of action type A during the bubble phase.
func (m *Interactive[S]) OnAction[A Action](listener func(action A, window *Window, cx *App)) S {
	m.self.Interactivity().OnAction(listener)
	return m.self
}

// OnBoxedAction binds listener to dispatches of action's type during the bubble phase.
func (m *Interactive[S]) OnBoxedAction(action Action, listener func(action Action, window *Window, cx *App)) S {
	m.self.Interactivity().OnBoxedAction(action, listener)
	return m.self
}

// OnKeyDown binds listener to the key down event during the bubble phase.
func (m *Interactive[S]) OnKeyDown(listener func(event *KeyDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnKeyDown(listener)
	return m.self
}

// CaptureKeyDown binds listener to the key down event during the capture phase.
func (m *Interactive[S]) CaptureKeyDown(listener func(event *KeyDownEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureKeyDown(listener)
	return m.self
}

// OnKeyUp binds listener to the key up event during the bubble phase.
func (m *Interactive[S]) OnKeyUp(listener func(event *KeyUpEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnKeyUp(listener)
	return m.self
}

// CaptureKeyUp binds listener to the key up event during the capture phase.
func (m *Interactive[S]) CaptureKeyUp(listener func(event *KeyUpEvent, window *Window, cx *App)) S {
	m.self.Interactivity().CaptureKeyUp(listener)
	return m.self
}

// OnModifiersChanged binds listener to modifier changes.
func (m *Interactive[S]) OnModifiersChanged(listener func(event *ModifiersChangedEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnModifiersChanged(listener)
	return m.self
}

// DragOver sets the style applied while a value of type T is dragged over the element.
func (m *Interactive[S]) DragOver[T any](f func(style *StyleRefinement, value T, window *Window, cx *App) *StyleRefinement) S {
	i := m.self.Interactivity()
	i.dragOverStyles = append(i.dragOverStyles, dragOverStyle{
		valueType: reflect.TypeFor[T](),
		style: func(currentlyDragged any, window *Window, cx *App) *StyleRefinement {
			return f(&StyleRefinement{}, currentlyDragged.(T), window, cx)
		},
	})
	return m.self
}

// GroupDragOver sets the style applied while a value of type T is dragged over the named group.
func (m *Interactive[S]) GroupDragOver[T any](groupName SharedString, f func(style *StyleRefinement) *StyleRefinement) S {
	i := m.self.Interactivity()
	i.groupDragOverStyles = append(i.groupDragOverStyles, groupDragOverStyle{
		valueType: reflect.TypeFor[T](),
		style:     GroupStyle{Group: groupName, Style: f(&StyleRefinement{})},
	})
	return m.self
}

// OnDrop binds listener to drops of a value of type T on the element.
func (m *Interactive[S]) OnDrop[T any](listener func(value T, window *Window, cx *App)) S {
	m.self.Interactivity().OnDrop(listener)
	return m.self
}

// CanDrop sets a predicate deciding whether a dragged value can be dropped on the element.
func (m *Interactive[S]) CanDrop(predicate func(value any, window *Window, cx *App) bool) S {
	m.self.Interactivity().CanDrop(predicate)
	return m.self
}

// Occlude blocks mouse interactions with elements behind this one.
func (m *Interactive[S]) Occlude() S {
	m.self.Interactivity().OccludeMouse()
	return m.self
}

// WindowControlArea marks the element as a platform window control area.
func (m *Interactive[S]) WindowControlArea(area WindowControlArea) S {
	m.self.Interactivity().WindowControlArea(area)
	return m.self
}

// BlockMouseExceptScroll blocks mouse interactions except scrolling with elements behind this one.
func (m *Interactive[S]) BlockMouseExceptScroll() S {
	m.self.Interactivity().BlockMouseExceptScroll()
	return m.self
}

// Focus sets the style applied while the element is focused.
func (m *Interactive[S]) Focus(f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().focusStyle = f(&StyleRefinement{})
	return m.self
}

// InFocus sets the style applied while the element or a descendant is focused.
func (m *Interactive[S]) InFocus(f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().inFocusStyle = f(&StyleRefinement{})
	return m.self
}

// FocusVisible sets the style applied while the element is focused by keyboard.
func (m *Interactive[S]) FocusVisible(f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().focusVisibleStyle = f(&StyleRefinement{})
	return m.self
}

// Role overrides the element's accessibility role.
func (m *StatefulInteractive[S]) Role(role a11y.Role) S {
	if gpuiutil.DebugAssertions && role == a11y.RoleGenericContainer {
		panic("GenericContainer is filtered out of the a11y tree and has no effect")
	}
	m.self.Interactivity().overrideRole = Some(role)
	return m.self
}

// AccessibilityID sets the author-supplied accessibility id.
func (m *StatefulInteractive[S]) AccessibilityID(id SharedString) S {
	m.self.Interactivity().aria.authorID = Some(id)
	return m.self
}

// AriaLabel sets the accessible label.
func (m *StatefulInteractive[S]) AriaLabel(label SharedString) S {
	m.self.Interactivity().aria.label = Some(label)
	return m.self
}

// AriaDescription sets the accessible description.
func (m *StatefulInteractive[S]) AriaDescription(description SharedString) S {
	m.self.Interactivity().aria.description = Some(description)
	return m.self
}

// AriaKeyshortcuts sets the accessible keyboard shortcuts.
func (m *StatefulInteractive[S]) AriaKeyshortcuts(keyshortcuts SharedString) S {
	m.self.Interactivity().aria.keyshortcuts = Some(keyshortcuts)
	return m.self
}

// AriaActiveDescendant reports focus of this element as the active descendant.
func (m *StatefulInteractive[S]) AriaActiveDescendant() S {
	m.self.Interactivity().reportActiveDescendantFocus = true
	return m.self
}

// A11ySyntheticChildren adds synthetic accessibility nodes below the element.
func (m *StatefulInteractive[S]) A11ySyntheticChildren(f func(builder *A11ySubtreeBuilder)) S {
	m.self.Interactivity().a11ySyntheticChildren = f
	return m.self
}

// AriaSelected sets the accessible selected state.
func (m *StatefulInteractive[S]) AriaSelected(selected bool) S {
	m.self.Interactivity().aria.selected = Some(selected)
	return m.self
}

// AriaExpanded sets the accessible expanded state.
func (m *StatefulInteractive[S]) AriaExpanded(expanded bool) S {
	m.self.Interactivity().aria.expanded = Some(expanded)
	return m.self
}

// AriaToggled sets the accessible toggled state.
func (m *StatefulInteractive[S]) AriaToggled(toggled a11y.Toggled) S {
	m.self.Interactivity().aria.toggled = Some(toggled)
	return m.self
}

// AriaNumericValue sets the accessible numeric value.
func (m *StatefulInteractive[S]) AriaNumericValue(value float64) S {
	m.self.Interactivity().aria.numericValue = Some(value)
	return m.self
}

// AriaNumericValueStep sets the accessible numeric value step.
func (m *StatefulInteractive[S]) AriaNumericValueStep(step float64) S {
	m.self.Interactivity().aria.numericValueStep = Some(step)
	return m.self
}

// AriaValue sets the accessible value.
func (m *StatefulInteractive[S]) AriaValue(value SharedString) S {
	m.self.Interactivity().aria.value = Some(value)
	return m.self
}

// AriaPlaceholder sets the accessible placeholder.
func (m *StatefulInteractive[S]) AriaPlaceholder(placeholder SharedString) S {
	m.self.Interactivity().aria.placeholder = Some(placeholder)
	return m.self
}

// AriaMinNumericValue sets the accessible minimum numeric value.
func (m *StatefulInteractive[S]) AriaMinNumericValue(value float64) S {
	m.self.Interactivity().aria.minNumericValue = Some(value)
	return m.self
}

// AriaMaxNumericValue sets the accessible maximum numeric value.
func (m *StatefulInteractive[S]) AriaMaxNumericValue(value float64) S {
	m.self.Interactivity().aria.maxNumericValue = Some(value)
	return m.self
}

// AriaOrientation sets the accessible orientation.
func (m *StatefulInteractive[S]) AriaOrientation(orientation a11y.Orientation) S {
	m.self.Interactivity().aria.orientation = Some(orientation)
	return m.self
}

// AriaLevel sets the accessible hierarchical level.
func (m *StatefulInteractive[S]) AriaLevel(level int) S {
	m.self.Interactivity().aria.level = Some(level)
	return m.self
}

// AriaPositionInSet sets the accessible position in set.
func (m *StatefulInteractive[S]) AriaPositionInSet(position int) S {
	m.self.Interactivity().aria.positionInSet = Some(position)
	return m.self
}

// AriaSizeOfSet sets the accessible size of set.
func (m *StatefulInteractive[S]) AriaSizeOfSet(size int) S {
	m.self.Interactivity().aria.sizeOfSet = Some(size)
	return m.self
}

// AriaRowIndex sets the accessible row index.
func (m *StatefulInteractive[S]) AriaRowIndex(index int) S {
	m.self.Interactivity().aria.rowIndex = Some(index)
	return m.self
}

// AriaColumnIndex sets the accessible column index.
func (m *StatefulInteractive[S]) AriaColumnIndex(index int) S {
	m.self.Interactivity().aria.columnIndex = Some(index)
	return m.self
}

// AriaRowCount sets the accessible row count.
func (m *StatefulInteractive[S]) AriaRowCount(count int) S {
	m.self.Interactivity().aria.rowCount = Some(count)
	return m.self
}

// AriaColumnCount sets the accessible column count.
func (m *StatefulInteractive[S]) AriaColumnCount(count int) S {
	m.self.Interactivity().aria.columnCount = Some(count)
	return m.self
}

// OnA11yAction binds listener to an accessibility action on the element; data is nil when the
// action carries none.
func (m *StatefulInteractive[S]) OnA11yAction(action a11y.Action, listener func(data a11y.ActionData, window *Window, cx *App)) S {
	i := m.self.Interactivity()
	i.a11yActionListeners = append(i.a11yActionListeners, a11yActionEntry{action: action, listener: listener})
	return m.self
}

// Focusable makes the element focusable.
func (m *StatefulInteractive[S]) Focusable() S {
	m.self.Interactivity().focusable = true
	return m.self
}

// OverflowScroll makes the element scroll on both axes when its content overflows.
func (m *StatefulInteractive[S]) OverflowScroll() S {
	i := m.self.Interactivity()
	i.BaseStyle.Overflow.X = Some(OverflowScroll)
	i.BaseStyle.Overflow.Y = Some(OverflowScroll)
	return m.self
}

// OverflowXScroll makes the element scroll horizontally when its content overflows.
func (m *StatefulInteractive[S]) OverflowXScroll() S {
	m.self.Interactivity().BaseStyle.Overflow.X = Some(OverflowScroll)
	return m.self
}

// OverflowYScroll makes the element scroll vertically when its content overflows.
func (m *StatefulInteractive[S]) OverflowYScroll() S {
	m.self.Interactivity().BaseStyle.Overflow.Y = Some(OverflowScroll)
	return m.self
}

// RestrictScrollToAxis restricts scrolling to the axis of the input gesture.
func (m *StatefulInteractive[S]) RestrictScrollToAxis() S {
	m.self.Interactivity().BaseStyle.RestrictScrollToAxis = Some(true)
	return m.self
}

// TrackScroll tracks the element's scroll state with scrollHandle.
func (m *StatefulInteractive[S]) TrackScroll(scrollHandle ScrollHandle) S {
	m.self.Interactivity().trackedScrollHandle = Some(scrollHandle)
	return m.self
}

// AnchorScroll sets the scroll anchor the element scrolls to.
func (m *StatefulInteractive[S]) AnchorScroll(scrollAnchor Option[ScrollAnchor]) S {
	m.self.Interactivity().scrollAnchor = scrollAnchor
	return m.self
}

// Active sets the style applied while the element is pressed.
func (m *StatefulInteractive[S]) Active(f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().activeStyle = f(&StyleRefinement{})
	return m.self
}

// GroupActive sets the style applied while the named group is pressed.
func (m *StatefulInteractive[S]) GroupActive(groupName SharedString, f func(style *StyleRefinement) *StyleRefinement) S {
	m.self.Interactivity().groupActiveStyle = &GroupStyle{Group: groupName, Style: f(&StyleRefinement{})}
	return m.self
}

// OnClick binds listener to the click event.
func (m *StatefulInteractive[S]) OnClick(listener func(event *ClickEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnClick(listener)
	return m.self
}

// OnAuxClick binds listener to the auxiliary (middle) click event.
func (m *StatefulInteractive[S]) OnAuxClick(listener func(event *ClickEvent, window *Window, cx *App)) S {
	m.self.Interactivity().OnAuxClick(listener)
	return m.self
}

// OnDrag starts a drag of value when the element is dragged, rendering the view constructor builds.
func (m *StatefulInteractive[S]) OnDrag[T, W any](value T, constructor func(value T, offset Point[Pixels], window *Window, cx *App) Entity[W]) S {
	m.self.Interactivity().OnDrag(value, constructor)
	return m.self
}

// ExternalDragPayload sets how the drag started by OnDrag resolves its platform payload.
func (m *StatefulInteractive[S]) ExternalDragPayload[T any](resolver func(value T, window *Window, cx *App) (ExternalDragPayload, bool)) S {
	m.self.Interactivity().ExternalDragPayload(resolver)
	return m.self
}

// OnHover binds listener to hover changes of the element.
func (m *StatefulInteractive[S]) OnHover(listener func(hovered bool, window *Window, cx *App)) S {
	m.self.Interactivity().OnHover(listener)
	return m.self
}

// HoverListenerMode sets how the hover listener decides whether the element is hovered.
func (m *StatefulInteractive[S]) HoverListenerMode(mode HoverListenerMode) S {
	m.self.Interactivity().HoverListenerMode(mode)
	return m.self
}

// Tooltip sets the tooltip shown while the element is hovered.
func (m *StatefulInteractive[S]) Tooltip(buildTooltip func(window *Window, cx *App) AnyView) S {
	m.self.Interactivity().Tooltip(buildTooltip)
	return m.self
}

// HoverableTooltip sets a tooltip that stays visible while the mouse is over it.
func (m *StatefulInteractive[S]) HoverableTooltip(buildTooltip func(window *Window, cx *App) AnyView) S {
	m.self.Interactivity().HoverableTooltip(buildTooltip)
	return m.self
}

// TooltipShowDelay sets the delay before the tooltip is shown.
func (m *StatefulInteractive[S]) TooltipShowDelay(delay time.Duration) S {
	m.self.Interactivity().TooltipShowDelay(delay)
	return m.self
}

type mouseDownListener func(event *MouseDownEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type mouseUpListener func(event *MouseUpEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type mousePressureListener func(event *MousePressureEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type mouseMoveListener func(event *MouseMoveEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type mouseExitListener func(event *MouseExitEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type fileDropExitListener func(event FileDropEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type scrollListener func(event *ScrollEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type pinchListener func(event *PinchEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type backGestureListener func(event *BackGestureEvent, phase DispatchPhase, hitbox *Hitbox, window *Window, cx *App)

type clickListener func(event *ClickEvent, window *Window, cx *App)

// HoverListenerMode is how an element's hover listener decides whether it is hovered.
type HoverListenerMode uint8

const (
	// HoverListenerModeInputModalityAware treats the element as unhovered while keyboard input is active.
	HoverListenerModeInputModalityAware HoverListenerMode = iota
	// HoverListenerModeInputModalityIndependent follows the mouse position regardless of input modality.
	HoverListenerModeInputModalityIndependent
)

// String formats the mode like upstream's Debug.
func (m HoverListenerMode) String() string {
	switch m {
	case HoverListenerModeInputModalityAware:
		return "InputModalityAware"
	case HoverListenerModeInputModalityIndependent:
		return "InputModalityIndependent"
	}
	return "HoverListenerMode(" + strconv.Itoa(int(m)) + ")"
}

func (m HoverListenerMode) isHovered(hitbox *Hitbox, window *Window) bool {
	if m == HoverListenerModeInputModalityIndependent {
		return hitbox.ID.isHoveredIgnoringLastInput(window)
	}
	return hitbox.IsHovered(window)
}

type dragListener struct {
	value           any
	render          func(value any, offset Point[Pixels], window *Window, cx *App) AnyView
	externalPayload externalDragPayloadResolver
}

type externalDragPayloadResolver func(value any, window *Window, cx *App) (ExternalDragPayload, bool)

type dropListener func(value any, window *Window, cx *App)

type canDropPredicate func(value any, window *Window, cx *App) bool

type tooltipBuilder struct {
	build     func(window *Window, cx *App) AnyView
	hoverable bool
}

type keyDownListener func(event *KeyDownEvent, phase DispatchPhase, window *Window, cx *App)

type keyUpListener func(event *KeyUpEvent, phase DispatchPhase, window *Window, cx *App)

type actionListener func(action any, phase DispatchPhase, window *Window, cx *App)

type actionListenerEntry struct {
	actionType reflect.Type
	listener   actionListener
}

type dropListenerEntry struct {
	valueType reflect.Type
	listener  dropListener
}

type dragOverStyle struct {
	valueType reflect.Type
	style     func(value any, window *Window, cx *App) *StyleRefinement
}

type groupDragOverStyle struct {
	valueType reflect.Type
	style     GroupStyle
}

// NewDiv constructs a Div, the basic container element.
func NewDiv() *Div {
	d := &Div{}
	d.interactivity.sourceLocation = CallerSourceLocation(1)
	BindStyled(&d.Styled, d)
	BindInteractive(&d.Interactive, d)
	BindInteractiveID(&d.InteractiveID, d)
	BindParent(&d.Parent, d)
	BindFluent(&d.Fluent, d)
	return d
}

// Div is the basic container element.
type Div struct {
	Styled[*Div]
	Interactive[*Div]
	InteractiveID[*Div]
	Parent[*Div]
	Fluent[*Div]
	interactivity     Interactivity
	children          []AnyElement
	prepaintListener  func(childBounds []Bounds[Pixels], window *Window, cx *App)
	imageCache        ImageCacheProvider
	prepaintOrderFunc func(window *Window, cx *App) []int
}

// OnChildrenPrepainted sets a listener receiving the bounds of the children after prepaint.
func (d *Div) OnChildrenPrepainted(listener func(childBounds []Bounds[Pixels], window *Window, cx *App)) *Div {
	d.prepaintListener = listener
	return d
}

// ImageCache sets the image cache provider for the div's subtree.
func (d *Div) ImageCache(cache ImageCacheProvider) *Div {
	d.imageCache = cache
	return d
}

// WithDynamicPrepaintOrder sets a function returning the order in which children are prepainted.
func (d *Div) WithDynamicPrepaintOrder(orderFn func(window *Window, cx *App) []int) *Div {
	d.prepaintOrderFunc = orderFn
	return d
}

// DivFrameState is the request-layout state of a Div.
type DivFrameState struct {
	childLayoutIDs []LayoutID
}

// DivInspectorState is the inspector state of a Div.
type DivInspectorState struct {
	// BaseStyle is the div's base style.
	BaseStyle *StyleRefinement
	// Bounds are the div's bounds.
	Bounds Bounds[Pixels]
	// ContentSize is the size of the div's content.
	ContentSize Size[Pixels]
}

// Style returns the div's base style refinement.
func (d *Div) Style() *StyleRefinement {
	return &d.interactivity.BaseStyle
}

// Interactivity returns the div's interactivity.
func (d *Div) Interactivity() *Interactivity {
	return &d.interactivity
}

// Extend adds child elements.
func (d *Div) Extend(elements []AnyElement) {
	d.children = append(d.children, elements...)
}

type ariaProperties struct {
	authorID         Option[SharedString]
	label            Option[SharedString]
	description      Option[SharedString]
	keyshortcuts     Option[SharedString]
	selected         Option[bool]
	expanded         Option[bool]
	toggled          Option[a11y.Toggled]
	numericValue     Option[float64]
	minNumericValue  Option[float64]
	maxNumericValue  Option[float64]
	numericValueStep Option[float64]
	value            Option[SharedString]
	placeholder      Option[SharedString]
	orientation      Option[a11y.Orientation]
	level            Option[int]
	positionInSet    Option[int]
	sizeOfSet        Option[int]
	rowIndex         Option[int]
	columnIndex      Option[int]
	rowCount         Option[int]
	columnCount      Option[int]
}

// Interactivity powers the general-purpose interactivity of the Div element.
type Interactivity struct {
	// ElementID is the element's id, required for the stateful subset of the interactivity.
	ElementID Option[ElementID]
	// Active is whether the element was clicked; present only after layout.
	Active Option[bool]
	// Hovered is whether the element was hovered; present only after paint if a hitbox was created.
	Hovered             Option[bool]
	tooltipID           Option[TooltipID]
	contentSize         Size[Pixels]
	keyContext          Option[KeyContext]
	focusable           bool
	trackedFocusHandle  Option[FocusHandle]
	trackedScrollHandle Option[ScrollHandle]
	scrollAnchor        Option[ScrollAnchor]
	scrollOffset        *Point[Pixels]
	ongoingScroll       *OngoingScroll
	group               Option[SharedString]
	// BaseStyle is the element's style before focus, active and other modifications.
	BaseStyle                   StyleRefinement
	focusStyle                  *StyleRefinement
	inFocusStyle                *StyleRefinement
	focusVisibleStyle           *StyleRefinement
	hoverStyle                  *StyleRefinement
	groupHoverStyle             *GroupStyle
	activeStyle                 *StyleRefinement
	groupActiveStyle            *GroupStyle
	dragOverStyles              []dragOverStyle
	groupDragOverStyles         []groupDragOverStyle
	mouseDownListeners          []mouseDownListener
	mouseUpListeners            []mouseUpListener
	mousePressureListeners      []mousePressureListener
	mouseMoveListeners          []mouseMoveListener
	mouseExitListeners          []mouseExitListener
	fileDropExitListeners       []fileDropExitListener
	scrollListeners             []scrollListener
	pinchListeners              []pinchListener
	backGestureListeners        []backGestureListener
	keyDownListeners            []keyDownListener
	keyUpListeners              []keyUpListener
	modifiersChangedListeners   []modifiersChangedListener
	actionListeners             []actionListenerEntry
	dropListeners               []dropListenerEntry
	canDropPredicate            canDropPredicate
	clickListeners              []clickListener
	auxClickListeners           []clickListener
	dragListener                *dragListener
	hoverListener               func(hovered bool, window *Window, cx *App)
	hoverListenerMode           HoverListenerMode
	tooltipBuilder              *tooltipBuilder
	tooltipShowDelay            Option[time.Duration]
	windowControl               Option[WindowControlArea]
	hitboxBehavior              HitboxBehavior
	tabIndex                    Option[int]
	tabGroup                    bool
	tabStop                     bool
	a11yActionListeners         []a11yActionEntry
	a11ySyntheticChildren       func(builder *A11ySubtreeBuilder)
	reportActiveDescendantFocus bool
	overrideRole                Option[a11y.Role]
	aria                        ariaProperties
	sourceLocation              *SourceLocation
	debugSelector               Option[string]
}

// InteractiveElement is implemented by elements that embed the Interactive mixin.
type InteractiveElement interface {
	Interactivity() *Interactivity
}

// Interactive is the InteractiveElement mixin; bind it with BindInteractive.
type Interactive[S InteractiveElement] struct {
	self S
}

// BindInteractive binds an Interactive mixin to the element that embeds it.
func BindInteractive[S InteractiveElement](m *Interactive[S], self S) {
	m.self = self
}

// InteractiveID carries InteractiveElement's ID method, which cannot live on Interactive (D-IDMIXIN); bind it with BindInteractiveID.
type InteractiveID[S InteractiveElement] struct {
	self S
}

// BindInteractiveID binds an InteractiveID mixin to the element that embeds it.
func BindInteractiveID[S InteractiveElement](m *InteractiveID[S], self S) {
	m.self = self
}

// StatefulInteractiveElement is implemented by stateful interactive elements.
type StatefulInteractiveElement interface {
	InteractiveElement
}

// StatefulInteractive is the StatefulInteractiveElement mixin; bind it with BindStatefulInteractive.
type StatefulInteractive[S StatefulInteractiveElement] struct {
	self S
}

// BindStatefulInteractive binds a StatefulInteractive mixin to the element that embeds it.
func BindStatefulInteractive[S StatefulInteractiveElement](m *StatefulInteractive[S], self S) {
	m.self = self
}
