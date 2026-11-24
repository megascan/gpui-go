package gpui

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/scheduler"
)

// RequestLayout adds a node to the layout tree for the current frame from the element's style and
// the layout ids of its children. Call it only from request_layout or prepaint.
func (w *Window) RequestLayout(style Style, children []LayoutID, cx *App) LayoutID {
	w.invalidator.debugAssertPrepaint()

	cx.layoutIDBuffer = append(cx.layoutIDBuffer[:0], children...)
	remSize := w.RemSize()
	scaleFactor := w.ScaleFactor()

	return w.layoutEngine.RequestLayout(style, remSize, scaleFactor, cx.layoutIDBuffer)
}

// RequestMeasuredLayout adds a leaf to the layout tree whose size measure computes at layout
// time. Call it only from request_layout or prepaint.
func (w *Window) RequestMeasuredLayout(style Style, measure func(knownDimensions SizeOf[Option[Pixels]], availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels]) LayoutID {
	w.invalidator.debugAssertPrepaint()

	remSize := w.RemSize()
	scaleFactor := w.ScaleFactor()
	return w.layoutEngine.RequestMeasuredLayout(style, remSize, scaleFactor, measure)
}

// ComputeLayout computes the layout of layoutID within availableSpace. Call it only from prepaint.
func (w *Window) ComputeLayout(layoutID LayoutID, availableSpace SizeOf[AvailableSpace], cx *App) {
	w.invalidator.debugAssertPrepaint()

	layoutEngine := w.layoutEngine
	if layoutEngine == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	w.layoutEngine = nil
	layoutEngine.ComputeLayout(layoutID, availableSpace, w, cx)
	w.layoutEngine = layoutEngine
}

// LayoutBounds returns the bounds computed for layoutID relative to the window.
func (w *Window) LayoutBounds(layoutID LayoutID) Bounds[Pixels] {
	w.invalidator.debugAssertPrepaint()

	scaleFactor := w.ScaleFactor()
	bounds := w.layoutEngine.LayoutBounds(layoutID, scaleFactor)
	snappedOffset := w.PixelSnapPoint(w.ElementOffset())
	bounds.Origin = bounds.Origin.Add(snappedOffset)
	return bounds
}

// InsertHitbox registers a hitbox for the current frame; use the returned Hitbox during paint or
// in event handlers to test whether it was the topmost. Call it only from prepaint.
func (w *Window) InsertHitbox(bounds Bounds[Pixels], behavior HitboxBehavior) Hitbox {
	w.invalidator.debugAssertPrepaint()

	contentMask := w.ContentMask()
	id := w.nextHitboxID
	w.nextHitboxID = w.nextHitboxID.next()
	hitbox := Hitbox{
		ID:          id,
		Bounds:      bounds,
		ContentMask: contentMask,
		Behavior:    behavior,
	}
	w.nextFrame.hitboxes = append(w.nextFrame.hitboxes, hitbox)
	return hitbox
}

// InsertWindowControlHitbox sets a hitbox that acts as a control area of the platform window.
// Call it only from paint.
func (w *Window) InsertWindowControlHitbox(area WindowControlArea, hitbox Hitbox) {
	w.invalidator.debugAssertPaint()
	w.nextFrame.windowControlHitboxes = append(w.nextFrame.windowControlHitboxes, windowControlHitbox{area: area, hitbox: hitbox})
}

// SetKeyContext sets the key context of the current element. Call it only from paint.
func (w *Window) SetKeyContext(context KeyContext) {
	w.invalidator.debugAssertPaint()
	w.nextFrame.dispatchTree.setKeyContext(context)
}

// SetFocusHandle sets the focus handle of the current element. Call it only from prepaint.
func (w *Window) SetFocusHandle(focusHandle FocusHandle, _ *App) {
	w.invalidator.debugAssertPrepaint()
	if focusHandle.IsFocused(w) {
		w.nextFrame.focus = Some(focusHandle.id)
	}
	w.nextFrame.dispatchTree.setFocusID(focusHandle.id)
}

// SetViewID sets the view id of the current element, used for view caching. Call it only from
// prepaint.
func (w *Window) SetViewID(viewID EntityID) {
	w.invalidator.debugAssertPrepaint()
	w.nextFrame.dispatchTree.setViewID(viewID)
}

// CurrentView returns the id of the view being rendered.
func (w *Window) CurrentView() EntityID {
	w.invalidator.debugAssertPaintOrPrepaint()
	if len(w.renderedEntityStack) == 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return w.renderedEntityStack[len(w.renderedEntityStack)-1]
}

func (w *Window) withRenderedView[R any](id EntityID, f func(window *Window) R) R {
	w.renderedEntityStack = append(w.renderedEntityStack, id)
	result := f(w)
	w.renderedEntityStack = w.renderedEntityStack[:len(w.renderedEntityStack)-1]
	return result
}

// WithImageCache runs f with imageCache, if any, as the current image cache.
func (w *Window) WithImageCache[R any](imageCache Option[AnyImageCache], f func(window *Window) R) R {
	if cache, ok := imageCache.Get(); ok {
		w.imageCacheStack = append(w.imageCacheStack, cache)
		result := f(w)
		w.imageCacheStack = w.imageCacheStack[:len(w.imageCacheStack)-1]
		return result
	}
	return f(w)
}

// HandleInput sets the input handler that receives text input for the upcoming frame while
// focusHandle is focused. Call it only from paint.
func (w *Window) HandleInput(focusHandle FocusHandle, inputHandler InputHandler, cx *App) {
	w.invalidator.debugAssertPaint()

	if focusHandle.IsFocused(w) {
		acx := w.ToAsync(cx)
		w.nextFrame.inputHandlers = append(w.nextFrame.inputHandlers, NewPlatformInputHandler(*acx, inputHandler).withFocus(focusHandle.id))
	}
}

// NotifyTextInputChanged tells native text services that the focused component's text changed.
func (w *Window) NotifyTextInputChanged(focusHandle FocusHandle) {
	if w.focusedTextInput == Some(focusHandle.id) && focusHandle.IsFocused(w) {
		w.platformWindow.TextInputStateChanged(TextInputStateChangeContentChanged)
	}
}

// NotifyTextSelectionChanged tells native text services that the focused component's selection
// changed.
func (w *Window) NotifyTextSelectionChanged(focusHandle FocusHandle) {
	if w.focusedTextInput == Some(focusHandle.id) && focusHandle.IsFocused(w) {
		w.platformWindow.TextInputStateChanged(TextInputStateChangeSelectionChanged)
	}
}

func (w *Window) applyTextInputConfiguration(cx *App) {
	var configuration TextInputConfiguration
	if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
		configuration = inputHandler.TextInputConfiguration(w, cx)
		w.platformWindow.SetInputHandler(inputHandler)
	} else {
		configuration = TextInputConfiguration{}
	}
	if last, ok := w.lastTextInputConfiguration.Get(); !ok || last != configuration {
		w.platformWindow.SetTextInputConfiguration(configuration)
		w.lastTextInputConfiguration = Some(configuration)
	}
}

// OnMouseEvent registers a mouse event listener for the next frame; it runs for events of type E.
// Call it only from paint.
func (w *Window) OnMouseEvent[E MouseEvent](listener func(event *E, phase DispatchPhase, window *Window, cx *App)) {
	w.invalidator.debugAssertPaint()

	w.nextFrame.mouseListeners = append(w.nextFrame.mouseListeners, func(event any, phase DispatchPhase, window *Window, cx *App) {
		if event, ok := event.(*E); ok {
			listener(event, phase, window, cx)
		}
	})
}

// OnKeyEvent registers a key event listener on the current node for the next frame; it runs for
// events of type E. Call it only from paint.
func (w *Window) OnKeyEvent[E KeyEvent](listener func(event *E, phase DispatchPhase, window *Window, cx *App)) {
	w.invalidator.debugAssertPaint()

	w.nextFrame.dispatchTree.onKeyEvent(func(event any, phase DispatchPhase, window *Window, cx *App) {
		if event, ok := event.(*E); ok {
			listener(event, phase, window, cx)
		}
	})
}

// OnModifiersChanged registers a modifiers changed listener on the current node for the next
// frame. Call it only from paint.
func (w *Window) OnModifiersChanged(listener func(event *ModifiersChangedEvent, window *Window, cx *App)) {
	w.invalidator.debugAssertPaint()

	w.nextFrame.dispatchTree.onModifiersChanged(func(event *ModifiersChangedEvent, window *Window, cx *App) {
		listener(event, window, cx)
	})
}

// OnFocusIn registers a listener called when handle or one of its descendants receives focus.
func (w *Window) OnFocusIn(handle FocusHandle, cx *App, listener func(window *Window, cx *App)) Subscription {
	focusID := handle.id
	subscription, activate := w.newFocusListener(func(event *windowFocusEvent, window *Window, cx *App) bool {
		if event.isFocusIn(focusID) {
			listener(window, cx)
		}
		return true
	})
	cx.Defer(func(*App) { activate() })
	return subscription
}

// OnFocusOut registers a listener called when handle or one of its descendants loses focus.
func (w *Window) OnFocusOut(handle FocusHandle, cx *App, listener func(event FocusOutEvent, window *Window, cx *App)) Subscription {
	focusID := handle.id
	subscription, activate := w.newFocusListener(func(event *windowFocusEvent, window *Window, cx *App) bool {
		if n := len(event.previousFocusPath); n > 0 && event.isFocusOut(focusID) {
			blurredID := event.previousFocusPath[n-1]
			listener(FocusOutEvent{Blurred: newWeakFocusHandle(blurredID, cx.focusHandles)}, window, cx)
		}
		return true
	})
	cx.Defer(func(*App) { activate() })
	return subscription
}

func (w *Window) resetCursorStyle(cx *App) {
	if w.IsWindowHovered() {
		style, ok := cx.ActiveDragCursorStyle()
		if !ok {
			style, ok = w.renderedFrame.cursorStyle(w)
		}
		if !ok {
			style = CursorStyleArrow
		}
		cx.platform.SetCursorStyle(style)
	}
}

// DispatchKeystroke dispatches keystroke as though the user had typed it.
func (w *Window) DispatchKeystroke(keystroke Keystroke, cx *App) bool {
	keystroke = keystroke.WithSimulatedIME()
	result := w.DispatchEvent(KeyDownEvent{
		Keystroke:            keystroke,
		IsHeld:               false,
		PreferCharacterInput: false,
	}, cx)
	if !result.Propagate {
		return true
	}

	if input, ok := keystroke.KeyChar.Get(); ok {
		if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
			inputHandler.DispatchInput(input, w, cx)
			w.platformWindow.SetInputHandler(inputHandler)
			return true
		}
	}

	return false
}

// KeystrokeTextFor returns the display text of the highest precedence binding for action, or the
// action's name when it has none.
func (w *Window) KeystrokeTextFor(action Action, cx *App) string {
	binding, ok := w.HighestPrecedenceBindingForAction(action)
	if !ok {
		return action.Name()
	}
	keystrokes := binding.Keystrokes()
	texts := make([]string, 0, len(keystrokes))
	for _, keystroke := range keystrokes {
		texts = append(texts, keystroke.Display(cx.PlatformInfo().Os).String())
	}
	return strings.Join(texts, " ")
}

// DispatchEvent dispatches a mouse, keyboard or touch event on the window.
func (w *Window) DispatchEvent(event PlatformInput, cx *App) DispatchEventResult {
	if profilerFeatureEnabled {
		w.windowProfiler.BeginInput(event.KindName())
	}
	updateCountBefore := w.invalidator.updateCount()
	oldModality := w.lastInputModality
	switch event.(type) {
	case KeyDownEvent:
		w.lastInputModality = inputModalityKeyboard
	case MouseMoveEvent, MouseDownEvent:
		w.lastInputModality = inputModalityMouse
	case TouchEvent:
		w.lastInputModality = inputModalityTouch
	}
	if w.lastInputModality != oldModality {
		w.Refresh()
	}

	cx.propagateEvent = true
	w.defaultPrevented = false

	switch e := event.(type) {
	case MouseMoveEvent:
		w.mousePosition = e.Position
		w.modifiers = e.Modifiers
	case MouseDownEvent:
		w.mousePosition = e.Position
		w.modifiers = e.Modifiers
	case MouseUpEvent:
		w.mousePosition = e.Position
		w.modifiers = e.Modifiers
	case MousePressureEvent:
	case MouseExitEvent:
		w.modifiers = e.Modifiers
	case ModifiersChangedEvent:
		w.modifiers = e.Modifiers
		w.capslock = e.Capslock
	case ScrollEvent:
		w.mousePosition = e.Position
		w.modifiers = e.Modifiers
	case PinchEvent:
		w.mousePosition = e.Position
		w.modifiers = e.Modifiers
	case FileDropEventEntered:
		w.mousePosition = e.Position
		sourceWindow := w.handle.WindowID()
		if !cx.restorePlatformDrag(sourceWindow) && cx.activeDrag.IsNone() {
			paths := e.Paths
			view := cx.New(func(*Context[ExternalPaths]) *ExternalPaths {
				p := slices.Clone(paths)
				return &p
			})
			cx.activeDrag = Some(AnyDrag{
				Value:                 slices.Clone(paths),
				View:                  AnyViewFromEntity(view),
				CursorOffset:          e.Position,
				CursorStyle:           None[CursorStyle](),
				ExternalPayloadSource: None[ExternalDragPayloadSource](),
			})
		}
		event = MouseMoveEvent{
			Position:      e.Position,
			PressedButton: Some(MouseButtonLeft),
			Modifiers:     Modifiers{},
		}
	case FileDropEventPending:
		w.mousePosition = e.Position
		event = MouseMoveEvent{
			Position:      e.Position,
			PressedButton: Some(MouseButtonLeft),
			Modifiers:     Modifiers{},
		}
	case FileDropEventSubmit:
		cx.Activate(true)
		w.mousePosition = e.Position
		event = MouseUpEvent{
			Button:     MouseButtonLeft,
			Position:   e.Position,
			Modifiers:  Modifiers{},
			ClickCount: 1,
		}
	case FileDropEventExited:
		if !cx.handRestoredDragToPlatform(w.handle.WindowID()) {
			releaseActiveDrag(cx.activeDrag.Take())
		}
		w.Refresh()
	case FileDropEventEnded:
		cx.endPlatformDrag(w.handle.WindowID())
		w.Refresh()
	case TouchEvent:
	case LongPressEvent:
		if e.Phase == TouchPhaseStarted {
			w.mousePosition = e.StartPosition
		} else {
			w.mousePosition = e.Position
		}
		if e.Phase == TouchPhaseStarted {
			w.longPressCapture = None[EntityID]()
		}
	case TouchDragEvent:
		w.mousePosition = e.StartPosition
	case BackGestureEvent:
		w.mousePosition = e.StartPosition
	case KeyDownEvent, KeyUpEvent:
	}

	if anyMouseEvent := platformInputMouseEvent(event); anyMouseEvent != nil {
		w.dispatchMouseEvent(anyMouseEvent, cx)
	} else if anyKeyEvent := platformInputKeyboardEvent(event); anyKeyEvent != nil {
		w.dispatchKeyEvent(anyKeyEvent, cx)
	} else if touchEvent, ok := event.TouchEvent(); ok {
		w.dispatchTouchEvent(&touchEvent, cx)
	}
	if longPress, ok := event.(LongPressEvent); ok {
		switch longPress.Phase {
		case TouchPhaseStarted:
			if !w.defaultPrevented {
				w.longPressCapture = None[EntityID]()
			}
		case TouchPhaseEnded, TouchPhaseCancelled:
			w.longPressCapture = None[EntityID]()
		}
	}

	w.promoteExternalDragToPlatform(event, cx)

	causedInvalidation := w.invalidator.updateCount() > updateCountBefore
	if causedInvalidation {
		w.inputRateTracker.recordInput()
	}
	if profilerFeatureEnabled {
		w.windowProfiler.EndInput(causedInvalidation)
	}

	return DispatchEventResult{
		Propagate:        cx.propagateEvent,
		DefaultPrevented: w.defaultPrevented,
	}
}

func releaseActiveDrag(drag Option[AnyDrag]) {
	if drag, ok := drag.Get(); ok {
		drag.View.Release()
	}
}

func (w *Window) promoteExternalDragToPlatform(event PlatformInput, cx *App) {
	mouseMove, ok := event.(MouseMoveEvent)
	if !ok {
		return
	}
	if mouseMove.PressedButton != Some(MouseButtonLeft) {
		return
	}
	if (Bounds[Pixels]{Origin: Point[Pixels]{}, Size: w.viewportSize}).Contains(mouseMove.Position) {
		return
	}
	if !w.platformWindow.CanStartExternalDrag() {
		return
	}
	drag := cx.activeDrag.AsMut()
	if drag == nil {
		return
	}
	payloadSource, ok := drag.ExternalPayloadSource.Take().Get()
	if !ok {
		return
	}
	payload, ok := payloadSource(w, cx)
	if !ok {
		return
	}
	if w.platformWindow.StartExternalDrag(payload) && cx.handActiveDragToPlatform(w.handle.WindowID()) {
		w.Refresh()
	}
}

// TouchPredictionEnabled reports whether recognized touch pans may use the platform's predicted
// touch positions. It defaults to true.
func (w *Window) TouchPredictionEnabled() bool {
	return w.touchPredictionEnabled
}

// SetTouchPredictionEnabled sets whether recognized touch pans may use the platform's predicted
// touch positions.
func (w *Window) SetTouchPredictionEnabled(enabled bool) {
	w.touchPredictionEnabled = enabled
}

func (w *Window) dispatchTouchEvent(touchEvent *TouchEvent, cx *App) {
	event := *touchEvent
	if !w.touchPredictionEnabled {
		event.PredictedPosition = None[Point[Pixels]]()
	}
	recognizedGestures := w.touchGestures.handleEvent(&event)
	if event.Phase == TouchPhaseStarted {
		if touchDrag, ok := w.touchGestures.offerTouchDrag(event.ID); ok {
			w.dispatchRecognizedTouchGesture(touchDrag, cx)
		}
	}
	if event.Phase == TouchPhaseStarted {
		if _, _, ok := w.touchGestures.pendingLongPress(); ok {
			w.longPressCapture = None[EntityID]()
		}
	}
	tapped := false
	for _, gesture := range recognizedGestures {
		_, isTap := gesture.(recognizedTouchGestureTap)
		tapped = tapped || isTap
		w.dispatchRecognizedTouchGesture(gesture, cx)
	}
	if event.Phase == TouchPhaseStarted {
		w.scheduleLongPressTimer(cx)
	} else if _, _, ok := w.touchGestures.pendingLongPress(); !ok {
		w.takeLongPressTimer()
	}
	if tapped && w.invalidator.isDirty() {
		w.Draw(cx).Clear(cx)
	}
	if w.touchGestures.hasMomentum() {
		w.scheduleTouchMomentumTick()
	}
}

func (w *Window) takeLongPressTimer() {
	if timer := w.longPressTimer; timer != nil {
		w.longPressTimer = nil
		timer.Cancel()
	}
}

func (w *Window) dispatchRecognizedTouchGesture(gesture recognizedTouchGesture, cx *App) {
	switch gesture := gesture.(type) {
	case recognizedTouchGestureScroll:
		scrollWheel := gesture.ScrollEvent
		w.mousePosition = scrollWheel.Position
		cx.propagateEvent = true
		w.dispatchMouseEvent(&scrollWheel, cx)
	case recognizedTouchGestureTap:
		down, up := gesture.down, gesture.up
		w.mousePosition = up.Position
		cx.propagateEvent = true
		w.dispatchMouseEvent(&down, cx)
		cx.propagateEvent = true
		w.dispatchMouseEvent(&up, cx)
	case recognizedTouchGestureTouchDrag:
		touchDrag := gesture.TouchDragEvent
		w.mousePosition = touchDrag.StartPosition
		cx.propagateEvent = true
		w.defaultPrevented = false
		started := touchDrag.Phase == TouchPhaseStarted
		w.dispatchMouseEvent(&touchDrag, cx)
		if started {
			w.touchGestures.resolveTouchDrag(w.defaultPrevented)
		}
	case recognizedTouchGestureLongPress:
		longPress := gesture.LongPressEvent
		if longPress.Phase == TouchPhaseStarted {
			w.mousePosition = longPress.StartPosition
		} else {
			w.mousePosition = longPress.Position
		}
		cx.propagateEvent = true
		w.defaultPrevented = false
		started := longPress.Phase == TouchPhaseStarted
		ended := longPress.Phase == TouchPhaseEnded || longPress.Phase == TouchPhaseCancelled
		w.dispatchMouseEvent(&longPress, cx)
		if started {
			claimed := w.defaultPrevented
			w.touchGestures.resolveLongPress(claimed)
			if !claimed {
				w.longPressCapture = None[EntityID]()
			}
		}
		if ended {
			w.longPressCapture = None[EntityID]()
		}
	}
}

func (w *Window) spawnWindowTask(cx *App, f func(cx *AsyncWindowContext)) *Task[struct{}] {
	handle := w.handle
	return cx.Spawn(func(app *AsyncApp) (struct{}, error) {
		f(newAsyncWindowContext(app, handle))
		return struct{}{}, nil
	})
}

func (w *Window) scheduleLongPressTimer(cx *App) {
	w.takeLongPressTimer()
	touchID, duration, ok := w.touchGestures.pendingLongPress()
	if !ok {
		return
	}
	w.longPressTimer = w.spawnWindowTask(cx, func(cx *AsyncWindowContext) {
		_, _ = cx.BackgroundExecutor().Timer(duration).Await(cx)
		err := cx.Update(func(window *Window, cx *App) {
			window.takeLongPressTimer()
			if gesture, ok := window.touchGestures.offerLongPress(touchID); ok {
				window.dispatchRecognizedTouchGesture(gesture, cx)
			}
		})
		logErrContext(err, "handling long-press timer for window %#v", cx.WindowHandle().WindowID())
	})
}

func (w *Window) scheduleTouchMomentumTick() {
	w.OnNextFrame(func(window *Window, cx *App) {
		if gesture, ok := window.touchGestures.tickMomentum(); ok {
			window.dispatchRecognizedTouchGesture(gesture, cx)
		}
		if window.touchGestures.hasMomentum() {
			window.scheduleTouchMomentumTick()
		}
	})
}

func (w *Window) dispatchMouseEvent(event any, cx *App) {
	w.mouseHitTest = w.renderedFrame.hitTest(w.MousePosition())

	if w.IsInspectorPicking(cx) {
		w.resetCursorStyle(cx)
		w.handleInspectorMouseEvent(event, cx)
		return
	}

	mouseListeners := w.renderedFrame.mouseListeners
	w.renderedFrame.mouseListeners = nil

	for _, listener := range mouseListeners {
		listener(event, DispatchPhaseCapture, w, cx)
		if !cx.propagateEvent {
			break
		}
	}

	if cx.propagateEvent {
		for i := len(mouseListeners) - 1; i >= 0; i-- {
			mouseListeners[i](event, DispatchPhaseBubble, w, cx)
			if !cx.propagateEvent {
				break
			}
		}
	}

	w.renderedFrame.mouseListeners = mouseListeners

	_, isMouseMove := event.(*MouseMoveEvent)
	_, isMouseUp := event.(*MouseUpEvent)
	if cx.HasActiveDrag() {
		if isMouseMove {
			w.Refresh()
		} else if isMouseUp {
			releaseActiveDrag(cx.activeDrag.Take())
			w.Refresh()
		}
	}

	if isMouseUp && w.capturedHitbox.IsSome() {
		w.capturedHitbox = None[HitboxID]()
	}

	w.resetCursorStyle(cx)
}

func (w *Window) inputPreference(event any, cx *App) InputPreference {
	keyDownEvent, ok := event.(*KeyDownEvent)
	preferCharacterInput := ok && keyDownEvent.PreferCharacterInput
	if !preferCharacterInput {
		return InputPreferenceKeyBindings
	}
	inputHandler := w.platformWindow.TakeInputHandler()
	if inputHandler == nil {
		return InputPreferenceKeyBindings
	}
	acceptsTextInput := inputHandler.AcceptsTextInput(w, cx)
	w.platformWindow.SetInputHandler(inputHandler)
	if acceptsTextInput {
		return InputPreferenceCharacterInput
	}
	return InputPreferenceKeyBindings
}

func releasePendingInput(pending *pendingInput) {
	if pending != nil && pending.timeout != nil {
		pending.timeout.release()
		pending.timeout = nil
	}
}

func (w *Window) dispatchKeyEvent(event any, cx *App) {
	if w.invalidator.isDirty() {
		w.Draw(cx).Clear(cx)
	}

	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	dispatchPath := w.renderedFrame.dispatchTree.dispatchPath(nodeID)

	keystroke := None[Keystroke]()

	if modifiersChanged, ok := event.(*ModifiersChangedEvent); ok {
		if modifiersChanged.Modifiers.NumberOfModifiers() == 0 && w.pendingModifier.modifiers.NumberOfModifiers() == 1 && !w.pendingModifier.sawOtherInput {
			key := ""
			switch modifiers := w.pendingModifier.modifiers; {
			case modifiers.Shift:
				key = "shift"
			case modifiers.Control:
				key = "control"
			case modifiers.Alt:
				key = "alt"
			case modifiers.Platform:
				key = "platform"
			case modifiers.Function:
				key = "function"
			}
			if key != "" {
				keystroke = Some(Keystroke{
					Key:       key,
					KeyChar:   None[string](),
					Modifiers: Modifiers{},
				})
			}
		}

		if w.pendingModifier.modifiers.NumberOfModifiers() == 0 && modifiersChanged.Modifiers.NumberOfModifiers() == 1 {
			w.pendingModifier.sawOtherInput = false
		} else if modifiersChanged.Modifiers.NumberOfModifiers() > 1 {
			w.pendingModifier.sawOtherInput = true
		}
		w.pendingModifier.modifiers = modifiersChanged.Modifiers
	} else if keyDownEvent, ok := event.(*KeyDownEvent); ok {
		w.pendingModifier.sawOtherInput = true
		keystroke = Some(keyDownEvent.Keystroke)
		if keyDownEvent.Keystroke.KeyChar.IsSome() && (cx.cursorHideMode == CursorHideModeOnTyping || cx.cursorHideMode == CursorHideModeOnTypingAndAction) {
			cx.platform.HideCursorUntilMouseMoves()
		}
	}

	ks, ok := keystroke.Get()
	if !ok {
		w.finishDispatchKeyEvent(event, nil, InputPreferenceKeyBindings, dispatchPath, w.ContextStack(), cx)
		return
	}

	inputPreference := w.inputPreference(event, cx)

	cx.propagateEvent = true
	w.dispatchKeystrokeInterceptors(&ks, inputPreference, w.ContextStack(), cx)
	if !cx.propagateEvent {
		w.finishDispatchKeyEvent(event, &ks, inputPreference, dispatchPath, w.ContextStack(), cx)
		return
	}

	currentlyPending := w.pendingInput
	w.pendingInput = nil
	if currentlyPending == nil {
		currentlyPending = &pendingInput{}
	}
	if currentlyPending.focus.IsSome() && currentlyPending.focus != w.focus {
		releasePendingInput(currentlyPending)
		currentlyPending = &pendingInput{}
	}

	matchResult := w.renderedFrame.dispatchTree.dispatchKey(currentlyPending.keystrokes, ks, dispatchPath)
	currentlyPending.keystrokes = nil

	if len(matchResult.toReplay) != 0 {
		w.replayPendingInput(matchResult.toReplay, cx)
		cx.propagateEvent = true
	}

	if len(matchResult.pending) != 0 {
		previousTimeout := currentlyPending.timeout
		currentlyPending.timeout = nil
		currentlyPending.keystrokes = matchResult.pending
		currentlyPending.focus = w.focus

		textInputRequiresTimeout := false
		if keyDown, ok := event.(*KeyDownEvent); ok && keyDown.Keystroke.KeyChar.IsSome() {
			if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
				accepts := inputHandler.AcceptsTextInput(w, cx)
				w.platformWindow.SetInputHandler(inputHandler)
				textInputRequiresTimeout = accepts
			}
		}

		standaloneModifier := len(currentlyPending.keystrokes) == 1 && currentlyPending.keystrokes[0].IsModifierKey()
		needsTimeout := previousTimeout != nil || matchResult.pendingHasBinding || textInputRequiresTimeout || standaloneModifier
		if needsTimeout {
			if previousTimeout != nil && previousTimeout.isPaused() {
				previousTimeout.resetDuration(pendingInputTimeoutDuration)
				currentlyPending.timeout = previousTimeout
			} else {
				if previousTimeout != nil {
					previousTimeout.release()
				}
				timeout := w.newPendingInputTimeout(pendingInputTimeoutDuration, cx)
				currentlyPending.timeout = &timeout
			}
		} else {
			currentlyPending.timeout = nil
		}
		w.pendingInput = currentlyPending
		w.pendingInputChanged(cx)
		cx.propagateEvent = false
		return
	}
	defer releasePendingInput(currentlyPending)

	inputPreference = w.inputPreference(event, cx)
	if inputPreference == InputPreferenceKeyBindings {
		for _, binding := range matchResult.bindings {
			w.dispatchActionOnNode(nodeID, binding.action, cx)
			if !cx.propagateEvent {
				w.dispatchKeystrokeObservers(&ks, inputPreference, binding.action, matchResult.contextStack, cx)
				w.pendingInputChanged(cx)
				return
			}
		}
	}

	w.finishDispatchKeyEvent(event, &ks, inputPreference, dispatchPath, matchResult.contextStack, cx)
	w.pendingInputChanged(cx)
}

func (w *Window) newPendingInputTimeout(duration time.Duration, cx *App) pendingInputTimeout {
	startedAt, task := w.startPendingInputTimeout(duration, cx)
	return pendingInputTimeout{
		duration:  duration,
		remaining: duration,
		state:     pendingInputTimeoutRunning{startedAt: startedAt, task: task},
	}
}

func (w *Window) startPendingInputTimeout(remaining time.Duration, cx *App) (scheduler.Instant, *Task[struct{}]) {
	startedAt := cx.BackgroundExecutor().Now()
	task := w.spawnWindowTask(cx, func(cx *AsyncWindowContext) {
		_, _ = cx.BackgroundExecutor().Timer(remaining).Await(cx)
		err := cx.Update(func(window *Window, cx *App) { window.flushPendingInput(cx) })
		logErrContext(err, "handling pending input timeout for window %#v", cx.WindowHandle().WindowID())
	})
	return startedAt, task
}

func (w *Window) flushPendingInput(cx *App) {
	currentlyPending := w.pendingInput
	w.pendingInput = nil
	if currentlyPending == nil {
		return
	}
	defer releasePendingInput(currentlyPending)
	if currentlyPending.focus != w.focus {
		return
	}

	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	dispatchPath := w.renderedFrame.dispatchTree.dispatchPath(nodeID)

	toReplay := w.renderedFrame.dispatchTree.flushDispatch(currentlyPending.keystrokes, dispatchPath)

	w.pendingInputChanged(cx)
	w.replayPendingInput(toReplay, cx)
}

func (w *Window) finishDispatchKeyEvent(event any, recognizedKeystroke *Keystroke, inputPreference InputPreference, dispatchPath []dispatchNodeID, contextStack []KeyContext, cx *App) {
	w.dispatchKeyDownUpEvent(event, dispatchPath, cx)
	if !cx.propagateEvent {
		return
	}

	w.dispatchModifiersChangedEvent(event, dispatchPath, cx)
	if !cx.propagateEvent {
		return
	}

	if recognizedKeystroke != nil {
		w.dispatchKeystrokeObservers(recognizedKeystroke, inputPreference, nil, contextStack, cx)
	}
}

func (w *Window) pendingInputChanged(cx *App) {
	w.pendingInputObservers.retain(func(callback *anyObserver) bool { return (*callback)(w, cx) })
}

func (w *Window) deferPendingInputChanged(cx *App) {
	windowHandle := w.handle
	cx.Defer(func(cx *App) {
		_ = windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) {
			window.pendingInputChanged(cx)
		})
	})
}

func (w *Window) dispatchKeyDownUpEvent(event any, dispatchPath []dispatchNodeID, cx *App) {
	for _, nodeID := range dispatchPath {
		node := w.renderedFrame.dispatchTree.node(nodeID)

		for _, keyListener := range slices.Clone(node.keyListeners) {
			keyListener(event, DispatchPhaseCapture, w, cx)
			if !cx.propagateEvent {
				return
			}
		}
	}

	for i := len(dispatchPath) - 1; i >= 0; i-- {
		node := w.renderedFrame.dispatchTree.node(dispatchPath[i])
		for _, keyListener := range slices.Clone(node.keyListeners) {
			keyListener(event, DispatchPhaseBubble, w, cx)
			if !cx.propagateEvent {
				return
			}
		}
	}
}

func (w *Window) dispatchModifiersChangedEvent(event any, dispatchPath []dispatchNodeID, cx *App) {
	modifiersChanged, ok := event.(*ModifiersChangedEvent)
	if !ok {
		return
	}
	for i := len(dispatchPath) - 1; i >= 0; i-- {
		node := w.renderedFrame.dispatchTree.node(dispatchPath[i])
		for _, listener := range slices.Clone(node.modifiersChangedListeners) {
			listener(modifiersChanged, w, cx)
			if !cx.propagateEvent {
				return
			}
		}
	}
}

// HasPendingKeystrokes reports whether a multi-stroke key binding is in progress on this window.
func (w *Window) HasPendingKeystrokes() bool {
	_, ok := w.PendingInput()
	return ok
}

func (w *Window) pendingInputIsNone() bool {
	return w.pendingInput == nil
}

func (w *Window) clearPendingKeystrokes(cx *App) {
	if pending := w.pendingInput; pending != nil {
		w.pendingInput = nil
		releasePendingInput(pending)
		w.deferPendingInputChanged(cx)
	}
}

// PendingInput returns the pending input that can still complete a multi-stroke key binding.
// Input left over from a previous focus can never complete one.
func (w *Window) PendingInput() (PendingInputStatus, bool) {
	pending := w.pendingInput
	if pending == nil || pending.focus != w.focus {
		return PendingInputStatus{}, false
	}
	timeout := None[PendingInputTimeoutStatus]()
	if pending.timeout != nil {
		timeout = Some(pending.timeout.status())
	}
	return PendingInputStatus{keystrokes: pending.keystrokes, timeout: timeout}, true
}

// SetPendingInputTimeoutPaused pauses or resumes the current pending input timeout on behalf of
// owner. A paused timeout resumes automatically when owner is released. It reports whether the
// timeout state changed; a timeout paused by one owner cannot be resumed by another.
func (w *Window) SetPendingInputTimeoutPaused[T any](owner Entity[T], paused bool, cx *App) bool {
	ownerID := owner.EntityID()
	if !paused {
		return w.resumePendingInputTimeout(ownerID, cx)
	}

	timeout := w.focusedPendingInputTimeout()
	if timeout == nil {
		return false
	}
	if timeout.isPaused() {
		return false
	}

	releaseSubscription := w.ObserveRelease(owner, cx, func(_ *T, window *Window, cx *App) {
		window.resumePendingInputTimeout(ownerID, cx)
	})
	now := cx.BackgroundExecutor().Now()
	changed := false
	if timeout := w.focusedPendingInputTimeout(); timeout != nil {
		changed = timeout.pause(pendingInputTimeoutPause{ownerID: ownerID, releaseSubscription: releaseSubscription}, now)
	} else {
		releaseSubscription.Unsubscribe()
	}

	if changed {
		w.deferPendingInputChanged(cx)
	}
	return changed
}

func (w *Window) focusedPendingInputTimeout() *pendingInputTimeout {
	if w.pendingInput == nil || w.pendingInput.focus != w.focus {
		return nil
	}
	return w.pendingInput.timeout
}

func (w *Window) resumePendingInputTimeout(ownerID EntityID, cx *App) bool {
	if w.pendingInput == nil || w.pendingInput.timeout == nil {
		return false
	}
	if id, ok := w.pendingInput.timeout.pauseOwnerID(); !ok || id != ownerID {
		return false
	}
	remaining := w.pendingInput.timeout.remaining

	startedAt, task := w.startPendingInputTimeout(remaining, cx)
	changed := false
	if w.pendingInput != nil && w.pendingInput.timeout != nil {
		changed = w.pendingInput.timeout.resume(ownerID, startedAt, task)
	} else {
		task.Cancel()
	}

	if changed {
		w.deferPendingInputChanged(cx)
	}
	return changed
}

// PendingInputKeystrokes returns the pending keystrokes that might complete a multi-stroke key
// binding.
func (w *Window) PendingInputKeystrokes() ([]Keystroke, bool) {
	pending, ok := w.PendingInput()
	if !ok {
		return nil, false
	}
	return pending.Keystrokes(), true
}

func (w *Window) replayPendingInput(replays []replay, cx *App) {
	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	dispatchPath := w.renderedFrame.dispatchTree.dispatchPath(nodeID)

replayLoop:
	for _, replay := range replays {
		event := &KeyDownEvent{
			Keystroke:            replay.keystroke,
			IsHeld:               false,
			PreferCharacterInput: true,
		}

		cx.propagateEvent = true
		for _, binding := range replay.bindings {
			w.dispatchActionOnNode(nodeID, binding.action, cx)
			if !cx.propagateEvent {
				keystroke := replay.keystroke
				w.dispatchKeystrokeObservers(&keystroke, InputPreferenceKeyBindings, binding.action, nil, cx)
				continue replayLoop
			}
		}

		w.dispatchKeyDownUpEvent(event, dispatchPath, cx)
		if !cx.propagateEvent {
			continue replayLoop
		}
		if input, ok := replay.keystroke.KeyChar.Get(); ok {
			if inputHandler := w.platformWindow.TakeInputHandler(); inputHandler != nil {
				inputHandler.DispatchInput(input, w, cx)
				w.platformWindow.SetInputHandler(inputHandler)
			}
		}
	}
}

func (w *Window) focusNodeIDInRenderedFrame(focusID Option[FocusID]) dispatchNodeID {
	if id, ok := focusID.Get(); ok {
		if nodeID, ok := w.renderedFrame.dispatchTree.focusableNodeID(id); ok {
			return nodeID
		}
	}
	return w.renderedFrame.dispatchTree.rootNodeID()
}

func (w *Window) dispatchActionOnNode(nodeID dispatchNodeID, action Action, cx *App) {
	w.dispatchActionOnNodeInner(nodeID, action, cx)

	if !cx.propagateEvent && cx.cursorHideMode == CursorHideModeOnTypingAndAction && w.LastInputWasKeyboard() {
		cx.platform.HideCursorUntilMouseMoves()
	}
}

func (w *Window) dispatchActionOnNodeInner(nodeID dispatchNodeID, action Action, cx *App) {
	dispatchPath := w.renderedFrame.dispatchTree.dispatchPath(nodeID)
	actionType := reflect.TypeOf(action)

	cx.propagateEvent = true
	if globalListeners, ok := cx.globalActionListeners[actionType]; ok {
		delete(cx.globalActionListeners, actionType)
		for _, listener := range globalListeners {
			if profilerFeatureEnabled {
				w.windowProfiler.BeginActionHandler(action, cx)
			}
			listener(action, DispatchPhaseCapture, cx)
			if profilerFeatureEnabled {
				w.windowProfiler.EndActionHandler()
			}
			if !cx.propagateEvent {
				break
			}
		}

		globalListeners = append(globalListeners, cx.globalActionListeners[actionType]...)
		delete(cx.globalActionListeners, actionType)
		cx.globalActionListeners[actionType] = globalListeners
	}

	if !cx.propagateEvent {
		return
	}

	for _, nodeID := range dispatchPath {
		node := w.renderedFrame.dispatchTree.node(nodeID)
		for _, listener := range slices.Clone(node.actionListeners) {
			if listener.actionType == actionType {
				if profilerFeatureEnabled {
					w.windowProfiler.BeginActionHandler(action, cx)
				}
				listener.listener(action, DispatchPhaseCapture, w, cx)
				if profilerFeatureEnabled {
					w.windowProfiler.EndActionHandler()
				}

				if !cx.propagateEvent {
					return
				}
			}
		}
	}

	for i := len(dispatchPath) - 1; i >= 0; i-- {
		node := w.renderedFrame.dispatchTree.node(dispatchPath[i])
		for _, listener := range slices.Clone(node.actionListeners) {
			if listener.actionType == actionType {
				cx.propagateEvent = false
				if profilerFeatureEnabled {
					w.windowProfiler.BeginActionHandler(action, cx)
				}
				listener.listener(action, DispatchPhaseBubble, w, cx)
				if profilerFeatureEnabled {
					w.windowProfiler.EndActionHandler()
				}

				if !cx.propagateEvent {
					return
				}
			}
		}
	}

	if globalListeners, ok := cx.globalActionListeners[actionType]; ok {
		delete(cx.globalActionListeners, actionType)
		for i := len(globalListeners) - 1; i >= 0; i-- {
			cx.propagateEvent = false

			if profilerFeatureEnabled {
				w.windowProfiler.BeginActionHandler(action, cx)
			}
			globalListeners[i](action, DispatchPhaseBubble, cx)
			if profilerFeatureEnabled {
				w.windowProfiler.EndActionHandler()
			}
			if !cx.propagateEvent {
				break
			}
		}

		globalListeners = append(globalListeners, cx.globalActionListeners[actionType]...)
		delete(cx.globalActionListeners, actionType)
		cx.globalActionListeners[actionType] = globalListeners
	}
}

// ObserveGlobal registers f to be invoked, with this window, whenever the global of type G is
// updated.
func (w *Window) ObserveGlobal[G any](cx *App, f func(window *Window, cx *App)) Subscription {
	windowHandle := w.handle
	subscription, activate := cx.globalObservers.insertPending(reflect.TypeFor[G](), func(cx *App) bool {
		return windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) { f(window, cx) }) == nil
	})
	cx.Defer(func(*App) { activate() })
	return subscription
}

// ActivateWindow focuses the window and brings it to the foreground at the platform level.
func (w *Window) ActivateWindow() {
	w.platformWindow.Activate()
}

// RequestAttention asks the operating system to draw attention to this window.
func (w *Window) RequestAttention() {
	w.platformWindow.RequestAttention()
}

// MinimizeWindow minimizes the window at the platform level.
func (w *Window) MinimizeWindow() {
	w.platformWindow.Minimize()
}

// ToggleFullscreen toggles full screen status of the window at the platform level.
func (w *Window) ToggleFullscreen() {
	w.platformWindow.ToggleFullscreen()
}

// ToggleSimpleFullscreen toggles simple (borderless) fullscreen; it only has an effect on macOS.
func (w *Window) ToggleSimpleFullscreen() {
	w.platformWindow.ToggleSimpleFullscreen()
}

// InvalidateCharacterCoordinates updates the IME panel position on the next frame.
func (w *Window) InvalidateCharacterCoordinates() {
	w.OnNextFrame(func(window *Window, cx *App) {
		if inputHandler := window.platformWindow.TakeInputHandler(); inputHandler != nil {
			if bounds, ok := inputHandler.SelectedBounds(window, cx); ok {
				window.platformWindow.UpdateIMEPosition(bounds)
			}
			window.platformWindow.SetInputHandler(inputHandler)
		}
	})
}

// Prompt presents a dialog with message and a button for each answer; the returned receiver
// receives the index of the clicked button.
func (w *Window) Prompt(level PromptLevel, message string, detail Option[string], answers []PromptButton, cx *App) *scheduler.OneshotReceiver[int] {
	builder, ok := cx.promptBuilder.Take().Get()
	if !ok {
		panic("internal error: entered unreachable code: Re-entrant window prompting is not supported by GPUI")
	}

	answers = slices.Clone(answers)

	var receiver *scheduler.OneshotReceiver[int]
	switch builder.(type) {
	case promptBuilderDefault:
		receiver = w.platformWindow.Prompt(level, message, detail, answers)
		if receiver == nil {
			receiver = w.buildCustomPrompt(builder, level, message, detail, answers, cx)
		}
	case promptBuilderCustom:
		receiver = w.buildCustomPrompt(builder, level, message, detail, answers, cx)
	}

	cx.promptBuilder = Some(builder)

	return receiver
}

func (w *Window) buildCustomPrompt(builder promptBuilder, level PromptLevel, message string, detail Option[string], answers []PromptButton, cx *App) *scheduler.OneshotReceiver[int] {
	sender, receiver := scheduler.Oneshot[int]()
	handle := newPromptHandle(sender)
	renderable := promptBuilderRenderer(builder)(level, message, detail, answers, handle, w, cx)
	if previous, ok := w.prompt.Replace(renderable).Get(); ok {
		previous.release()
	}
	return receiver
}

// HasActivePrompt reports whether a prompt rendered by GPUI is active in this window; platform
// native prompt dialogs do not count.
func (w *Window) HasActivePrompt() bool {
	return w.prompt.IsSome()
}

// ContextStack returns the current key context stack.
func (w *Window) ContextStack() []KeyContext {
	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	dispatchTree := w.renderedFrame.dispatchTree
	var contexts []KeyContext
	for _, nodeID := range dispatchTree.dispatchPath(nodeID) {
		if context, ok := dispatchTree.node(nodeID).context.Get(); ok {
			contexts = append(contexts, context.Clone())
		}
	}
	return contexts
}

// AvailableActions returns every action available on the focused element.
func (w *Window) AvailableActions(cx *App) []Action {
	nodeID := w.focusNodeIDInRenderedFrame(w.focus)
	actions := w.renderedFrame.dispatchTree.availableActions(nodeID)
	for actionType := range cx.globalActionListeners {
		ix, found := slices.BinarySearchFunc(actions, actionType, func(a Action, actionType reflect.Type) int {
			return compareActionTypes(reflect.TypeOf(a), actionType)
		})
		if !found {
			if action, err := cx.actions.buildActionType(actionType); err == nil {
				actions = slices.Insert(actions, ix, action)
			}
		}
	}
	return actions
}

// BindingsForAction returns the key bindings that invoke action on the focused element, in the
// order they were added; for display, the last one takes precedence.
func (w *Window) BindingsForAction(action Action) []KeyBinding {
	return w.renderedFrame.dispatchTree.bindingsForAction(action, w.renderedFrame.dispatchTree.contextStack)
}

// HighestPrecedenceBindingForAction returns the highest precedence key binding that invokes
// action on the focused element.
func (w *Window) HighestPrecedenceBindingForAction(action Action) (KeyBinding, bool) {
	return w.renderedFrame.dispatchTree.highestPrecedenceBindingForAction(action, w.renderedFrame.dispatchTree.contextStack)
}

// BindingsForActionInContext returns the key bindings for action in context.
func (w *Window) BindingsForActionInContext(action Action, context KeyContext) []KeyBinding {
	return w.renderedFrame.dispatchTree.bindingsForAction(action, []KeyContext{context})
}

// HighestPrecedenceBindingForActionInContext returns the highest precedence key binding for
// action in context.
func (w *Window) HighestPrecedenceBindingForActionInContext(action Action, context KeyContext) (KeyBinding, bool) {
	return w.renderedFrame.dispatchTree.highestPrecedenceBindingForAction(action, []KeyContext{context})
}

// BindingsForActionIn returns the bindings that would invoke action on focusHandle if it were
// focused, in the order they were added.
func (w *Window) BindingsForActionIn(action Action, focusHandle FocusHandle) []KeyBinding {
	contextStack, ok := w.contextStackForFocusHandle(focusHandle)
	if !ok {
		return nil
	}
	return w.renderedFrame.dispatchTree.bindingsForAction(action, contextStack)
}

// HighestPrecedenceBindingForActionIn returns the highest precedence binding that would invoke
// action on focusHandle if it were focused.
func (w *Window) HighestPrecedenceBindingForActionIn(action Action, focusHandle FocusHandle) (KeyBinding, bool) {
	contextStack, ok := w.contextStackForFocusHandle(focusHandle)
	if !ok {
		return KeyBinding{}, false
	}
	return w.renderedFrame.dispatchTree.highestPrecedenceBindingForAction(action, contextStack)
}

// PossibleBindingsForInput returns the bindings that can follow input in the current context
// stack.
func (w *Window) PossibleBindingsForInput(input []Keystroke) []KeyBinding {
	return w.renderedFrame.dispatchTree.possibleNextBindingsForInput(input, w.ContextStack())
}

func (w *Window) contextStackForFocusHandle(focusHandle FocusHandle) ([]KeyContext, bool) {
	dispatchTree := w.renderedFrame.dispatchTree
	nodeID, ok := dispatchTree.focusableNodeID(focusHandle.id)
	if !ok {
		return nil, false
	}
	contextStack := []KeyContext{}
	for _, nodeID := range dispatchTree.dispatchPath(nodeID) {
		if context, ok := dispatchTree.node(nodeID).context.Get(); ok {
			contextStack = append(contextStack, context.Clone())
		}
	}
	return contextStack, true
}

// ListenerFor returns an event listener that runs f with view's state and context.
func (w *Window) ListenerFor[T, E any](view Entity[T], f func(this *T, event *E, window *Window, cx *Context[T])) func(event *E, window *Window, cx *App) {
	weak := view.Downgrade()
	return func(e *E, window *Window, cx *App) {
		_ = weak.Update(cx, func(view *T, cx *Context[T]) { f(view, e, window, cx) })
	}
}

// HandlerFor returns a handler that runs f with entity's state and context.
func (w *Window) HandlerFor[E any](entity Entity[E], f func(this *E, window *Window, cx *Context[E])) func(window *Window, cx *App) {
	weak := entity.Downgrade()
	return func(window *Window, cx *App) {
		_ = weak.Update(cx, func(entity *E, cx *Context[E]) { f(entity, window, cx) })
	}
}

// OnWindowShouldClose registers a callback that can stop the window from closing by returning
// false.
func (w *Window) OnWindowShouldClose(cx *App, f func(window *Window, cx *App) bool) {
	acx := w.ToAsync(cx)
	w.platformWindow.OnShouldClose(func() bool {
		result, err := acx.UpdateR(func(window *Window, cx *App) bool { return f(window, cx) })
		if err != nil {
			return true
		}
		return result
	})
}

// OnAction registers an action listener on the current node for the next frame; it runs for
// actions of actionType. Call it only from paint.
func (w *Window) OnAction(actionType reflect.Type, listener func(action any, phase DispatchPhase, window *Window, cx *App)) {
	w.invalidator.debugAssertPaint()

	w.nextFrame.dispatchTree.onAction(actionType, listener)
}

// OnActionWhen is OnAction when condition is true. Call it only from paint.
func (w *Window) OnActionWhen(condition bool, actionType reflect.Type, listener func(action any, phase DispatchPhase, window *Window, cx *App)) {
	w.invalidator.debugAssertPaint()

	if condition {
		w.nextFrame.dispatchTree.onAction(actionType, listener)
	}
}

// GPUSpecs returns information about the GPU backing this window, if the platform reports it.
func (w *Window) GPUSpecs() (GPUSpecs, bool) {
	return w.platformWindow.GPUSpecs()
}

// TitlebarDoubleClick performs the titlebar double-click action; it is macOS specific.
func (w *Window) TitlebarDoubleClick() {
	w.platformWindow.TitlebarDoubleClick(w.isResizable, w.isMinimizable)
}

// WindowTitle returns the window's title at the platform level; it is macOS specific.
func (w *Window) WindowTitle() string {
	return w.platformWindow.GetTitle()
}

// TabbedWindows returns the tabbed windows and their titles; it is macOS specific.
func (w *Window) TabbedWindows() ([]SystemWindowTab, bool) {
	return w.platformWindow.TabbedWindows()
}

// TabBarVisible reports whether the tab bar is visible; it is macOS specific.
func (w *Window) TabBarVisible() bool {
	return w.platformWindow.TabBarVisible()
}

// MergeAllWindows merges all open windows into one tabbed window; it is macOS specific.
func (w *Window) MergeAllWindows() {
	w.platformWindow.MergeAllWindows()
}

// MoveTabToNewWindow moves the tab to a new containing window; it is macOS specific.
func (w *Window) MoveTabToNewWindow() {
	w.platformWindow.MoveTabToNewWindow()
}

// ToggleWindowTabOverview shows or hides the window tab overview; it is macOS specific.
func (w *Window) ToggleWindowTabOverview() {
	w.platformWindow.ToggleWindowTabOverview()
}

// SetTabbingIdentifier sets the tabbing identifier of the window; it is macOS specific.
func (w *Window) SetTabbingIdentifier(tabbingIdentifier Option[string]) {
	w.platformWindow.SetTabbingIdentifier(tabbingIdentifier)
}

// PlaySystemBell asks the operating system to play an alert sound.
func (w *Window) PlaySystemBell() {
	w.platformWindow.PlaySystemBell()
}

// IsA11yEnabled reports whether accessibility support is enabled for this window, independent of
// whether assistive technology is connected.
func (w *Window) IsA11yEnabled() bool {
	return w.a11y.isEnabled()
}

// IsA11yActive reports whether an accessibility tree is being built for this frame.
func (w *Window) IsA11yActive() bool {
	return w.a11y.isActive()
}

// DebugA11yTreeJSON returns a debug representation of the last frame's accessibility
// information.
func (w *Window) DebugA11yTreeJSON() (string, bool) {
	return w.a11y.debugTreeJSON()
}

// OnA11yAction registers a listener for an accessibility action on the node nodeID.
func (w *Window) OnA11yAction(nodeID a11y.NodeID, action a11y.Action, listener func(data a11y.ActionData, window *Window, cx *App)) {
	w.a11y.actionListeners[nodeID] = append(w.a11y.actionListeners[nodeID], a11yActionEntry{action: action, listener: listener})
}

func (w *Window) handleA11yAction(request a11y.ActionRequest, cx *App) {
	if listeners, ok := w.a11y.actionListeners[request.TargetNode]; ok {
		delete(w.a11y.actionListeners, request.TargetNode)
		extraData := request.Data
		matched := false
		for _, entry := range listeners {
			if entry.action == request.Action {
				entry.listener(extraData, w, cx)
				matched = true
			}
		}
		w.a11y.actionListeners[request.TargetNode] = listeners
		if matched {
			return
		}
	}

	switch request.Action {
	case a11y.ActionClick:
		if bounds, ok := w.a11y.nodeBounds[request.TargetNode]; ok {
			center := bounds.Center()
			mouseDown := MouseDownEvent{
				Button:     MouseButtonLeft,
				Position:   center,
				Modifiers:  Modifiers{},
				ClickCount: 1,
				FirstMouse: false,
			}
			mouseUp := MouseUpEvent{
				Button:     MouseButtonLeft,
				Position:   center,
				Modifiers:  Modifiers{},
				ClickCount: 1,
			}
			w.DispatchEvent(mouseDown, cx)
			w.DispatchEvent(mouseUp, cx)
		}
	case a11y.ActionFocus:
		if focusID, ok := w.a11y.focusIDs[request.TargetNode]; ok {
			if handle, ok := focusHandleForID(focusID, cx.focusHandles); ok {
				w.Focus(handle, cx)
				handle.Release()
			}
		}
	case a11y.ActionBlur:
		w.Blur(cx)
	default:
		slog.Debug(fmt.Sprintf("Unhandled a11y action: %v on %v", request.Action, request.TargetNode))
	}
}

// ToggleInspector toggles inspector mode on this window.
func (w *Window) ToggleInspector(cx *App) {
	if inspector, ok := w.inspector.Take().Get(); ok {
		w.renderedFrame.nextInspectorInstanceIDs = nil
		w.renderedFrame.inspectorHitboxes = nil
		w.nextFrame.nextInspectorInstanceIDs = nil
		w.nextFrame.inspectorHitboxes = nil
		inspector.Release()
	} else {
		w.inspector = Some(cx.New(func(*Context[Inspector]) *Inspector { return newInspector() }))
	}
	w.Refresh()
}

// IsInspectorPicking reports whether the window's inspector is picking an element.
func (w *Window) IsInspectorPicking(cx *App) bool {
	if inspector, ok := w.inspector.Get(); ok {
		return inspector.Read(cx).IsPicking()
	}
	return false
}

// WithInspectorState runs f with mutable access to the inspector state of type T of the element
// inspectorID, if it is the inspector's active element.
func (w *Window) WithInspectorState[T, R any](inspectorID *InspectorElementID, cx *App, f func(state *Option[T], window *Window) R) (R, bool) {
	var zero R
	if inspectorID == nil {
		return zero, false
	}
	inspector, ok := w.inspector.Get()
	if !ok {
		return zero, false
	}
	activeID, ok := inspector.Read(cx).ActiveElementID()
	if !ok || !activeID.Equal(*inspectorID) {
		return zero, false
	}
	inspector = inspector.Clone()
	defer inspector.Release()
	return inspector.UpdateR(cx, func(inspector *Inspector, _ *Context[Inspector]) R {
		return withActiveElementStateOf(inspector, w, f)
	}), true
}

func withActiveElementStateOf[T, R any](inspector *Inspector, window *Window, f func(state *Option[T], window *Window) R) R {
	return inspector.withActiveElementState(window, f)
}

func (w *Window) inspectorEnabled() bool {
	return w.inspector.IsSome()
}

func (w *Window) buildInspectorElementID(path InspectorElementPath) InspectorElementID {
	w.invalidator.debugAssertPaintOrPrepaint()
	sharedPath := &path
	var hasher FxHasher
	path.Hash(&hasher)
	key := hasher.Finish()
	bucket := w.nextFrame.nextInspectorInstanceIDs[key]
	for i := range bucket {
		if bucket[i].path.Equal(path) {
			instanceID := bucket[i].next
			bucket[i].next++
			return InspectorElementID{Path: sharedPath, InstanceID: instanceID}
		}
	}
	if w.nextFrame.nextInspectorInstanceIDs == nil {
		w.nextFrame.nextInspectorInstanceIDs = make(map[uint64][]inspectorInstanceIDs)
	}
	w.nextFrame.nextInspectorInstanceIDs[key] = append(bucket, inspectorInstanceIDs{path: path, next: 1})
	return InspectorElementID{Path: sharedPath, InstanceID: 0}
}

func (w *Window) prepaintInspector(inspectorWidth Pixels, cx *App) Option[AnyElement] {
	inspector, ok := w.inspector.Take().Get()
	if !ok {
		return None[AnyElement]()
	}
	inspectorElement := AnyViewFromEntity(inspector.Clone()).IntoAnyElement()
	inspectorElement.PrepaintAsRoot(
		Point[Pixels]{X: w.viewportSize.Width - inspectorWidth, Y: Px(0.0)},
		SizeAvailableSpaceFromPixels(Size[Pixels]{Width: inspectorWidth, Height: w.viewportSize.Height}),
		w,
		cx,
	)
	w.inspector = Some(inspector)
	return Some(inspectorElement)
}

func (w *Window) paintInspector(inspectorElement Option[AnyElement], cx *App) {
	if element := inspectorElement.AsMut(); element != nil {
		element.Paint(w, cx)
	}
}

// InsertInspectorHitbox registers a hitbox for inspector picking mode, so users can select the
// element inspectorID by clicking on it.
func (w *Window) InsertInspectorHitbox(hitboxID HitboxID, inspectorID *InspectorElementID, cx *App) {
	w.invalidator.debugAssertPaintOrPrepaint()
	if !w.IsInspectorPicking(cx) {
		return
	}
	if inspectorID != nil {
		if w.nextFrame.inspectorHitboxes == nil {
			w.nextFrame.inspectorHitboxes = make(map[HitboxID]InspectorElementID)
		}
		w.nextFrame.inspectorHitboxes[hitboxID] = *inspectorID
	}
}

func (w *Window) paintInspectorHitbox(cx *App) {
	if inspector, ok := w.inspector.Get(); ok {
		state := inspector.Read(cx)
		if hitboxID, _, ok := w.hoveredInspectorHitbox(state, w.nextFrame); ok {
			for _, hitbox := range w.nextFrame.hitboxes {
				if hitbox.ID == hitboxID {
					w.PaintQuad(NewFill(hitbox.Bounds, NewRgba(0x61afef4d).ToBackground()))
					break
				}
			}
		}
	}
}

func (w *Window) handleInspectorMouseEvent(event any, cx *App) {
	inspector, ok := w.inspector.Get()
	if !ok {
		return
	}
	inspector = inspector.Clone()
	defer inspector.Release()
	switch event := event.(type) {
	case *MouseMoveEvent:
		inspector.Update(cx, func(inspector *Inspector, _ *Context[Inspector]) {
			if _, inspectorID, ok := w.hoveredInspectorHitbox(inspector, w.renderedFrame); ok {
				inspector.hover(inspectorID, w)
			}
		})
	case *MouseDownEvent:
		inspector.Update(cx, func(inspector *Inspector, _ *Context[Inspector]) {
			if _, inspectorID, ok := w.hoveredInspectorHitbox(inspector, w.renderedFrame); ok {
				inspector.selectElement(inspectorID, w)
			}
		})
	case *ScrollEvent:
		const scrollLines float32 = 3.0
		const scrollPixelsPerLayer float32 = 36.0
		deltaY := event.Delta.PixelDelta(Px(scrollPixelsPerLayer / scrollLines)).Y
		inspector.Update(cx, func(inspector *Inspector, _ *Context[Inspector]) {
			if depth := inspector.pickDepth.AsMut(); depth != nil {
				*depth += float32(deltaY) / scrollPixelsPerLayer
				maxDepth := float32(len(w.mouseHitTest.ids)) - 0.5
				if *depth < 0.0 {
					*depth = 0.0
				} else if *depth > maxDepth {
					*depth = maxDepth
				}
				if _, inspectorID, ok := w.hoveredInspectorHitbox(inspector, w.renderedFrame); ok {
					inspector.setActiveElementID(inspectorID, w)
				}
			}
		})
	}
}

func (w *Window) hoveredInspectorHitbox(inspector *Inspector, frame *frame) (HitboxID, InspectorElementID, bool) {
	if pickDepth, ok := inspector.pickDepth.Get(); ok {
		depth := int64(pickDepth)
		if depth < 0 {
			depth = 0
		}
		maxSkipped := max(len(w.mouseHitTest.ids)-1, 0)
		skipCount := min(int(depth), maxSkipped)
		for _, hitboxID := range w.mouseHitTest.ids[skipCount:] {
			if inspectorID, ok := frame.inspectorHitboxes[hitboxID]; ok {
				return hitboxID, inspectorID, true
			}
		}
	}
	return 0, InspectorElementID{}, false
}

// SetModifiers sets the current modifier keys state without generating events; it is for tests.
func (w *Window) SetModifiers(modifiers Modifiers) {
	w.modifiers = modifiers
}

// SimulateMouseMove dispatches a mouse move to position through the normal event path; it is for
// tests.
func (w *Window) SimulateMouseMove(position Point[Pixels], cx *App) {
	event := MouseMoveEvent{
		Position:      position,
		Modifiers:     w.modifiers,
		PressedButton: None[MouseButton](),
	}
	_ = w.DispatchEvent(event, cx)
}

func (w *Window) takeElementState(globalID *GlobalElementID, stateType reflect.Type) (elementStateKey, elementStateBox, bool) {
	key := elementStateKey{id: GlobalElementID{ids: slices.Clone(globalID.ids), hash: globalID.hash}, typ: stateType}
	w.nextFrame.accessedElementStates = append(w.nextFrame.accessedElementStates, key)
	if _, state, ok := w.nextFrame.elementStates.removeEntry(key); ok {
		return key, state, true
	}
	if _, state, ok := w.renderedFrame.elementStates.removeEntry(key); ok {
		return key, state, true
	}
	return key, elementStateBox{}, false
}

func (w *Window) insertElementState(key elementStateKey, state elementStateBox) {
	w.nextFrame.elementStates.insert(key, state)
}

func observeKeyedState[S any](state Entity[S], currentView EntityID, cx *App) {
	cx.Observe(state, func(_ Entity[S], cx *App) {
		cx.Notify(currentView)
	}).Detach()
}
