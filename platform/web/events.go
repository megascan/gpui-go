//go:build js && wasm

package web

import (
	"log/slog"
	"math"
	"strconv"
	"strings"
	"syscall/js"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/megascan/gpui-go"
)

// WebEventListeners owns the DOM listeners of a WebWindow.
type WebEventListeners struct {
	handles []*EventListenerHandle
}

// Close removes every listener.
func (l *WebEventListeners) Close() {
	if l == nil {
		return
	}
	for _, handle := range l.handles {
		handle.Close()
	}
	l.handles = nil
}

// EventListenerHandle is a DOM event listener that is removed from its target by Close.
type EventListenerHandle struct {
	target    js.Value
	eventName string
	closure   js.Func
	closed    bool
}

func newListenerClosure(handler func(event js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		handler(argOrUndefined(args))
		return nil
	})
}

func eventListenerHandleAdd(target js.Value, eventName string, handler func(event js.Value)) *EventListenerHandle {
	closure := newListenerClosure(handler)
	_, _ = jsCall(target, "addEventListener", eventName, closure)
	return &EventListenerHandle{target: target, eventName: eventName, closure: closure}
}

func eventListenerHandleAddNonPassive(target js.Value, eventName string, handler func(event js.Value)) *EventListenerHandle {
	closure := newListenerClosure(handler)
	options := js.Global().Get("Object").New()
	options.Set("passive", false)
	if add := jsGet(target, "addEventListener"); add.Type() == js.TypeFunction {
		_ = jsTry(func() { add.Call("call", target, eventName, closure, options) })
	}
	return &EventListenerHandle{target: target, eventName: eventName, closure: closure}
}

// Close removes the listener and releases its function; it is upstream's drop and runs once.
func (h *EventListenerHandle) Close() {
	if h == nil || h.closed {
		return
	}
	h.closed = true
	_, _ = jsCall(h.target, "removeEventListener", h.eventName, h.closure)
	h.closure.Release()
}

type clickState struct {
	lastPosition gpui.Point[gpui.Pixels]
	lastTime     float64
	currentCount int
}

type touchIDs struct {
	next   uint64
	active map[int32]gpui.TouchID
}

func (t *touchIDs) start(pointerID int32) (gpui.TouchID, bool) {
	if t.next == math.MaxUint64 {
		return 0, false
	}
	touchID := gpui.TouchID(t.next)
	t.next++
	if t.active == nil {
		t.active = map[int32]gpui.TouchID{}
	}
	t.active[pointerID] = touchID
	return touchID, true
}

func (t *touchIDs) activeID(pointerID int32) (gpui.TouchID, bool) {
	id, ok := t.active[pointerID]
	return id, ok
}

func (t *touchIDs) end(pointerID int32) (gpui.TouchID, bool) {
	id, ok := t.active[pointerID]
	if ok {
		delete(t.active, pointerID)
	}
	return id, ok
}

func (c *clickState) registerClick(position gpui.Point[gpui.Pixels], time float64) int {
	dx := float32(position.X) - float32(c.lastPosition.X)
	dy := float32(position.Y) - float32(c.lastPosition.Y)
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if (time-c.lastTime) < 400.0 && distance < 5.0 {
		c.currentCount++
	} else {
		c.currentCount = 1
	}

	c.lastPosition = position
	c.lastTime = time
	return c.currentCount
}

func (w *webWindowInner) registerEventListeners() *WebEventListeners {
	handles := []*EventListenerHandle{
		w.registerPointerDown(),
		w.registerPointerUp(),
		w.registerPointerCancel(),
		w.registerTouchEnd(),
		w.registerPointerMove(),
		w.registerPointerLeave(),
		w.registerWheel(),
		w.registerContextMenu(),
		w.registerDragover(),
		w.registerDrop(),
		w.registerKeyDown(),
		w.registerKeyUp(),
		w.registerBeforeInput(),
		w.registerInput(),
		w.registerPaste(),
		w.registerCompositionStart(),
		w.registerCompositionUpdate(),
		w.registerCompositionEnd(),
		w.registerFocus(),
		w.registerBlur(),
		w.registerPointerEnter(),
	}
	for _, optional := range []*EventListenerHandle{
		w.registerSelectionChange(),
		w.registerVisibilityChange(),
		w.registerAppearanceChange(),
		w.registerFullscreenChange(),
	} {
		if optional != nil {
			handles = append(handles, optional)
		}
	}
	handles = append(handles, w.registerViewportChanges()...)
	return &WebEventListeners{handles: handles}
}

func (w *webWindowInner) registerViewportChanges() []*EventListenerHandle {
	targets := []js.Value{w.browserWindow}
	if viewport := jsGet(w.browserWindow, "visualViewport"); !jsIsNullish(viewport) {
		targets = append(targets, viewport)
	}
	var handles []*EventListenerHandle
	for _, target := range targets {
		for _, eventName := range [...]string{"resize", "scroll"} {
			handles = append(handles, eventListenerHandleAdd(target, eventName, func(js.Value) {
				w.notifyViewportChanged()
			}))
		}
	}
	return handles
}

func (w *webWindowInner) listen(eventName string, handler func(js.Value)) *EventListenerHandle {
	return eventListenerHandleAdd(w.canvas, eventName, handler)
}

func (w *webWindowInner) listenInput(eventName string, handler func(js.Value)) *EventListenerHandle {
	return eventListenerHandleAdd(w.imeMirror.eventTarget(), eventName, handler)
}

func (w *webWindowInner) listenNonPassive(eventName string, handler func(js.Value)) *EventListenerHandle {
	return eventListenerHandleAddNonPassive(w.canvas, eventName, handler)
}

func (w *webWindowInner) dispatchInput(input gpui.PlatformInput) (gpui.DispatchEventResult, bool) {
	var result gpui.DispatchEventResult
	ok := withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(gpui.PlatformInput) gpui.DispatchEventResult] {
		return &c.input
	},
		func(callback func(gpui.PlatformInput) gpui.DispatchEventResult) { result = callback(input) })
	return result, ok
}

func (w *webWindowInner) updateModifiers(modifiers gpui.Modifiers, capslock gpui.Capslock) bool {
	changed := w.state.modifiers != modifiers || w.state.capslock != capslock
	w.state.modifiers = modifiers
	w.state.capslock = capslock
	return changed
}

func preventDefault(event js.Value) {
	_, _ = jsCall(event, "preventDefault")
}

func eventString(event js.Value, property string) string {
	s, _ := jsString(jsGet(event, property))
	return s
}

func eventBool(event js.Value, property string) bool {
	b, _ := jsBool(jsGet(event, property))
	return b
}

func eventNumber(event js.Value, property string) float64 {
	n, _ := jsNumber(jsGet(event, property))
	return n
}

func eventInt32(event js.Value, property string) int32 {
	return int32(eventNumber(event, property))
}

func pointerID(event js.Value) int32 {
	return eventInt32(event, "pointerId")
}

func (w *webWindowInner) registerPointerDown() *EventListenerHandle {
	return w.listen("pointerdown", func(event js.Value) {
		preventDefault(event)

		pointerType := eventString(event, "pointerType")
		position := pointerPositionInElement(event)
		w.gestureStartVisualViewportHeight = w.visualViewportHeight()

		_, _ = jsCall(w.canvas, "setPointerCapture", pointerID(event))

		if pointerType == "touch" {
			if w.keyboardLikelyDismissed() {
				w.setVirtualKeyboardEnabled(false)
			}
			touchID, ok := w.touchIDs.start(pointerID(event))
			if !ok {
				slog.Error("exhausted touch identifiers")
				return
			}
			w.state.mousePosition = position
			if w.touchTapCandidate.IsNone() {
				w.touchTapCandidate = gpui.Some(touchTapCandidate{pointerID: pointerID(event), position: position})
			}
			w.dispatchInput(gpui.TouchEvent{
				ID:                touchID,
				Phase:             gpui.TouchPhaseStarted,
				Position:          position,
				PredictedPosition: gpui.None[gpui.Point[gpui.Pixels]](),
				Force:             gpui.None[float32](),
			})
			return
		}

		button := domMouseButtonToGpui(int16(eventNumber(event, "button")))
		modifiers := modifiersFromMouseEvent(event, w.isMac)
		time := dateNow()

		w.pressedButton = gpui.Some(button)
		clickCount := w.clickState.registerClick(position, time)

		w.state.mousePosition = position
		w.state.modifiers = modifiers

		w.dispatchInput(gpui.MouseDownEvent{
			Button:     button,
			Position:   position,
			Modifiers:  modifiers,
			ClickCount: clickCount,
			FirstMouse: false,
		})

		w.imeMirror.focus()
	})
}

func (w *webWindowInner) pointerTargetsTextInput(position gpui.Point[gpui.Pixels]) bool {
	result, ok := withInputHandler(w, func(handler *gpui.PlatformInputHandler) bool {
		if !handler.QueryAcceptsTextInput() {
			return false
		}
		bounds, ok := handler.ElementBounds()
		return ok && bounds.Contains(position)
	})
	return ok && result
}

func (w *webWindowInner) registerPointerUp() *EventListenerHandle {
	return w.listen("pointerup", func(event js.Value) {
		preventDefault(event)

		position := pointerPositionInElement(event)

		if eventString(event, "pointerType") == "touch" {
			touchID, ok := w.touchIDs.end(pointerID(event))
			if !ok {
				return
			}
			w.state.mousePosition = position
			completesTap := false
			if candidate, ok := w.touchTapCandidate.Get(); ok && candidate.pointerID == pointerID(event) {
				w.touchTapCandidate = gpui.None[touchTapCandidate]()
				completesTap = true
			}
			dispatchResult, dispatched := w.dispatchInput(gpui.TouchEvent{
				ID:                touchID,
				Phase:             gpui.TouchPhaseEnded,
				Position:          position,
				PredictedPosition: gpui.None[gpui.Point[gpui.Pixels]](),
				Force:             gpui.None[float32](),
			})

			viewportStable := w.gestureStartVisualViewportHeight == w.visualViewportHeight()
			if completesTap &&
				viewportStable &&
				dispatched && !dispatchResult.DefaultPrevented &&
				w.pointerTargetsTextInput(position) {
				w.showVirtualKeyboard()
			}
			w.scheduleImeMirrorSync()
			return
		}

		button := domMouseButtonToGpui(int16(eventNumber(event, "button")))
		modifiers := modifiersFromMouseEvent(event, w.isMac)

		w.pressedButton = gpui.None[gpui.MouseButton]()
		clickCount := w.clickState.currentCount

		w.state.mousePosition = position
		w.state.modifiers = modifiers

		w.dispatchInput(gpui.MouseUpEvent{
			Button:     button,
			Position:   position,
			Modifiers:  modifiers,
			ClickCount: clickCount,
		})

		w.scheduleImeMirrorSync()
	})
}

func (w *webWindowInner) visualViewportHeight() float64 {
	viewport := jsGet(w.browserWindow, "visualViewport")
	if jsIsNullish(viewport) {
		return 0.0
	}
	return eventNumber(viewport, "height") * eventNumber(viewport, "scale")
}

func (w *webWindowInner) keyboardLikelyDismissed() bool {
	if !w.touchInput {
		return false
	}
	if document := jsGet(w.browserWindow, "document"); !jsIsNullish(document) {
		if meta, err := jsCall(document, "querySelector", "meta[name=viewport]"); err == nil && !jsIsNullish(meta) {
			if attr, err := jsCall(meta, "getAttribute", "content"); err == nil {
				if content, ok := jsString(attr); ok && viewportOverlaysContent(content) {
					return false
				}
			}
		}
	}
	height, ok := w.viewport.keyboardHeight(w.browserWindow)
	return ok && height < 1.
}

func viewportOverlaysContent(content string) bool {
	for directive := range strings.SplitSeq(content, ",") {
		key, value, ok := strings.Cut(directive, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "interactive-widget") &&
			strings.EqualFold(strings.TrimSpace(value), "overlays-content") {
			return true
		}
	}
	return false
}

func (w *webWindowInner) setVirtualKeyboardEnabled(enabled bool) {
	w.imeMirror.setVirtualKeyboardEnabled(enabled)
	if enabled {
		w.viewport.holdCanvasHeight(w.browserWindow, w.canvas)
	} else {
		w.viewport.releaseCanvasHeight()
	}
}

func (w *webWindowInner) registerPointerCancel() *EventListenerHandle {
	return w.listen("pointercancel", func(event js.Value) {
		if eventString(event, "pointerType") == "touch" {
			touchID, ok := w.touchIDs.end(pointerID(event))
			if !ok {
				return
			}
			if candidate, ok := w.touchTapCandidate.Get(); ok && candidate.pointerID == pointerID(event) {
				w.touchTapCandidate = gpui.None[touchTapCandidate]()
			}
			w.dispatchInput(gpui.TouchEvent{
				ID:                touchID,
				Phase:             gpui.TouchPhaseCancelled,
				Position:          pointerPositionInElement(event),
				PredictedPosition: gpui.None[gpui.Point[gpui.Pixels]](),
				Force:             gpui.None[float32](),
			})
		} else {
			w.pressedButton = gpui.None[gpui.MouseButton]()
		}
	})
}

func (w *webWindowInner) registerTouchEnd() *EventListenerHandle {
	return w.listenNonPassive("touchend", func(event js.Value) {
		if eventBool(event, "cancelable") {
			preventDefault(event)
		}
	})
}

func (w *webWindowInner) scheduleImeMirrorSync() {
	imeMirrorScheduleSync(w)
}

func (w *webWindowInner) showVirtualKeyboard() {
	if !w.touchInput {
		return
	}
	wasEnabled := w.imeMirror.virtualKeyboardEnabled()
	w.setVirtualKeyboardEnabled(true)
	if !wasEnabled || !w.imeMirror.isFocused() || w.keyboardLikelyDismissed() {
		w.focusImeMirror()
	}
}

func (w *webWindowInner) focusImeMirror() {
	w.suppressFocusStatusEvents = true
	refocusRestartsInput := platformInfo(w.browserWindow).Os == gpui.OperatingSystemIos
	if w.imeMirror.isFocused() && !refocusRestartsInput {
		w.imeMirror.blur()
	}
	w.imeMirror.focus()
	w.suppressFocusStatusEvents = false

	callback := jsFuncOnce(func([]js.Value) {
		w.refreshActiveStatus()
	})
	if _, err := jsCall(w.browserWindow, "setTimeout", callback); err != nil {
		callback.Release()
		slog.Warn("failed to defer web window activation: " + jsErrorDebug(err))
	}
}

func (w *webWindowInner) dispatchSyntheticKeystroke(key string, modifiers gpui.Modifiers) {
	keystroke := gpui.Keystroke{
		Modifiers: modifiers,
		Key:       key,
		KeyChar:   gpui.None[string](),
	}
	w.dispatchInput(gpui.KeyDownEvent{
		Keystroke:            keystroke,
		IsHeld:               false,
		PreferCharacterInput: false,
	})
	w.dispatchInput(gpui.KeyUpEvent{Keystroke: keystroke})
}

func (w *webWindowInner) registerPointerMove() *EventListenerHandle {
	return w.listen("pointermove", func(event js.Value) {
		preventDefault(event)

		position := pointerPositionInElement(event)

		if eventString(event, "pointerType") == "touch" {
			touchID, ok := w.touchIDs.activeID(pointerID(event))
			if !ok {
				return
			}
			w.state.mousePosition = position
			if candidate, ok := w.touchTapCandidate.Get(); ok &&
				candidate.pointerID == pointerID(event) &&
				position.Sub(candidate.position).Magnitude() > float64(gpui.DefaultGestureTuning().TouchSlop) {
				w.touchTapCandidate = gpui.None[touchTapCandidate]()
			}
			w.dispatchInput(gpui.TouchEvent{
				ID:                touchID,
				Phase:             gpui.TouchPhaseMoved,
				Position:          position,
				PredictedPosition: predictedPointerPosition(event, position),
				Force:             gpui.None[float32](),
			})
			return
		}

		modifiers := modifiersFromMouseEvent(event, w.isMac)
		currentPressed := w.pressedButton

		w.state.mousePosition = position
		w.state.modifiers = modifiers

		w.dispatchInput(gpui.MouseMoveEvent{
			Position:      position,
			PressedButton: currentPressed,
			Modifiers:     modifiers,
		})
	})
}

func (w *webWindowInner) registerPointerLeave() *EventListenerHandle {
	return w.listen("pointerleave", func(event js.Value) {
		position := pointerPositionInElement(event)
		modifiers := modifiersFromMouseEvent(event, w.isMac)
		currentPressed := w.pressedButton

		w.state.mousePosition = position
		w.state.modifiers = modifiers
		w.state.isHovered = false

		w.dispatchInput(gpui.MouseExitEvent{
			Position:      position,
			PressedButton: currentPressed,
			Modifiers:     modifiers,
		})

		withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(bool)] { return &c.hoverStatusChange },
			func(callback func(bool)) { callback(false) })
	})
}

func (w *webWindowInner) registerWheel() *EventListenerHandle {
	return w.listenNonPassive("wheel", func(event js.Value) {
		preventDefault(event)

		position := mousePositionInElement(event)
		modifiers := modifiersFromWheelEvent(event, w.isMac)

		deltaMode := uint32(eventNumber(event, "deltaMode"))
		deltaX := eventNumber(event, "deltaX")
		deltaY := eventNumber(event, "deltaY")
		var delta gpui.ScrollDelta
		if deltaMode == 1 {
			delta = gpui.ScrollDeltaLines(gpui.NewPoint(float32(-deltaX), float32(-deltaY)))
		} else {
			delta = gpui.ScrollDeltaPixels(gpui.NewPoint(gpui.Px(float32(-deltaX)), gpui.Px(float32(-deltaY))))
		}

		w.state.modifiers = modifiers

		w.dispatchInput(gpui.ScrollEvent{
			Position:   position,
			Delta:      delta,
			Modifiers:  modifiers,
			TouchPhase: gpui.TouchPhaseMoved,
			Source:     gpui.ScrollSourcePointer,
		})
	})
}

func (w *webWindowInner) registerContextMenu() *EventListenerHandle {
	return w.listen("contextmenu", preventDefault)
}

func (w *webWindowInner) registerDragover() *EventListenerHandle {
	return w.listen("dragover", preventDefault)
}

func (w *webWindowInner) registerDrop() *EventListenerHandle {
	return w.listen("drop", preventDefault)
}

func (w *webWindowInner) registerKeyDown() *EventListenerHandle {
	return w.listenInput("keydown", func(event js.Value) {
		modifiers := modifiersFromKeyboardEvent(event, w.isMac)
		capslock := capslockFromKeyboardEvent(event)

		if w.updateModifiers(modifiers, capslock) {
			w.dispatchInput(gpui.ModifiersChangedEvent{Modifiers: modifiers, Capslock: capslock})
		}

		rawKey := eventString(event, "key")
		key := domKeyToGpuiKey(rawKey)

		if isModifierOnlyKey(key) {
			return
		}

		isHeld := eventBool(event, "repeat")
		keyChar := computeKeyChar(rawKey, key, modifiers)

		if w.touchInput &&
			w.imeMirror.virtualKeyboardEnabled() &&
			w.state.inputHandler != nil &&
			!modifiers.Platform &&
			!modifiers.Control &&
			!modifiers.Alt &&
			(keyChar.IsSome() || key == "backspace" || key == "delete" || key == "enter") {
			return
		}

		keystroke := gpui.Keystroke{
			Modifiers: keystrokeModifiers(key, modifiers),
			Key:       key,
			KeyChar:   keyChar,
		}

		if result, ok := w.dispatchInput(gpui.KeyDownEvent{
			Keystroke:            keystroke,
			IsHeld:               isHeld,
			PreferCharacterInput: false,
		}); ok && !result.Propagate {
			preventDefault(event)
			w.scheduleImeMirrorSync()
			return
		}

		if w.isComposing || eventBool(event, "isComposing") {
			preventDefault(event)
			return
		}

		if text, ok := keyChar.Get(); ok && keystrokeInsertsText(modifiers, w.isMac) {
			withInputHandler(w, func(handler *gpui.PlatformInputHandler) struct{} {
				handler.ReplaceTextInRange(gpui.None[gpui.Range](), text)
				return struct{}{}
			})
			preventDefault(event)
		}
		w.scheduleImeMirrorSync()
	})
}

func (w *webWindowInner) registerKeyUp() *EventListenerHandle {
	return w.listenInput("keyup", func(event js.Value) {
		modifiers := modifiersFromKeyboardEvent(event, w.isMac)
		capslock := capslockFromKeyboardEvent(event)

		if w.updateModifiers(modifiers, capslock) {
			w.dispatchInput(gpui.ModifiersChangedEvent{Modifiers: modifiers, Capslock: capslock})
		}

		rawKey := eventString(event, "key")
		key := domKeyToGpuiKey(rawKey)

		if isModifierOnlyKey(key) {
			return
		}

		keyChar := computeKeyChar(rawKey, key, modifiers)

		keystroke := gpui.Keystroke{
			Modifiers: keystrokeModifiers(key, modifiers),
			Key:       key,
			KeyChar:   keyChar,
		}

		if result, ok := w.dispatchInput(gpui.KeyUpEvent{Keystroke: keystroke}); ok && !result.Propagate {
			preventDefault(event)
		}
	})
}

type imeEdit struct {
	insertedText           string
	removedBeforeSelection int
	removedAfterSelection  int
}

func diffImeEdit(oldValue, newValue string, postEditCaret gpui.Option[uint32], storedSelection [2]uint32) imeEdit {
	oldUnits := utf16.Encode([]rune(oldValue))
	newUnits := utf16.Encode([]rune(newValue))

	suffixLength := -1
	if caret, ok := postEditCaret.Get(); ok {
		length := max(len(newUnits)-int(caret), 0)
		if length <= len(oldUnits) && unitsEqual(oldUnits[len(oldUnits)-length:], newUnits[len(newUnits)-length:]) {
			suffixLength = length
		}
	}
	if suffixLength < 0 {
		suffixLength = 0
		for suffixLength < len(oldUnits) && suffixLength < len(newUnits) &&
			oldUnits[len(oldUnits)-1-suffixLength] == newUnits[len(newUnits)-1-suffixLength] {
			suffixLength++
		}
	}
	prefixLength := 0
	for prefixLength < len(oldUnits) && prefixLength < len(newUnits) && oldUnits[prefixLength] == newUnits[prefixLength] {
		prefixLength++
	}
	prefixLength = min(prefixLength, len(oldUnits)-suffixLength, len(newUnits)-suffixLength)

	insertedText := string(utf16.Decode(newUnits[prefixLength : len(newUnits)-suffixLength]))
	replacedOldEnd := len(oldUnits) - suffixLength

	return imeEdit{
		insertedText:           insertedText,
		removedBeforeSelection: max(int(storedSelection[0])-prefixLength, 0),
		removedAfterSelection:  max(replacedOldEnd-int(storedSelection[1]), 0),
	}
}

func unitsEqual(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (w *webWindowInner) registerInput() *EventListenerHandle {
	return w.listenInput("input", func(event js.Value) {
		if w.isComposing || eventBool(event, "isComposing") {
			return
		}

		newValue := w.imeMirror.value()
		oldValue := w.imeMirror.storedText()
		if newValue == oldValue {
			return
		}

		start, ok := w.imeMirror.selectionStart()
		postEditCaret := gpui.None[uint32]()
		if ok {
			postEditCaret = gpui.Some(start)
		}
		storedStart, storedEnd := w.imeMirror.storedSelection()
		edit := diffImeEdit(oldValue, newValue, postEditCaret, [2]uint32{storedStart, storedEnd})

		applied, ok := withInputHandler(w, func(handler *gpui.PlatformInputHandler) bool {
			selection, ok := handler.SelectedTextRange(false)
			if !ok {
				return false
			}
			editRange := gpui.Range{
				Start: max(selection.Range.Start-edit.removedBeforeSelection, 0),
				End:   selection.Range.End + edit.removedAfterSelection,
			}
			handler.ReplaceTextInRange(gpui.Some(editRange), edit.insertedText)
			return true
		})
		if !ok || !applied {
			return
		}

		w.imeMirror.adoptElementState()
	})
}

func (w *webWindowInner) registerSelectionChange() *EventListenerHandle {
	document := jsGet(w.browserWindow, "document")
	if jsIsNullish(document) {
		return nil
	}
	return eventListenerHandleAdd(document, "selectionchange", func(js.Value) {
		if w.isComposing || !w.imeMirror.isFocused() {
			return
		}
		if w.imeMirror.value() != w.imeMirror.storedText() {
			return
		}
		elementStart, ok := w.imeMirror.selectionStart()
		if !ok {
			return
		}
		elementEnd, ok := w.imeMirror.elementSelectionEnd()
		if !ok {
			elementEnd = elementStart
		}
		storedStart, storedEnd := w.imeMirror.storedSelection()
		if elementStart == storedStart && elementEnd == storedEnd {
			return
		}
		applied, ok := withInputHandler(w, func(handler *gpui.PlatformInputHandler) bool {
			selection, ok := handler.SelectedTextRange(false)
			if !ok {
				return false
			}
			alignment := selection.Range.Start - int(storedStart)
			if alignment < 0 || selection.Range.End-int(storedEnd) != alignment {
				return false
			}
			handler.SetSelectedTextRange(gpui.Range{
				Start: alignment + int(elementStart),
				End:   alignment + int(elementEnd),
			})
			return true
		})
		if ok && applied {
			w.imeMirror.adoptElementState()
		} else {
			w.imeMirror.rejectSelectionImport()
			w.scheduleImeMirrorSync()
		}
	})
}

func (w *webWindowInner) registerBeforeInput() *EventListenerHandle {
	return w.listenInput("beforeinput", func(event js.Value) {
		if w.isComposing || eventBool(event, "isComposing") {
			return
		}

		switch eventString(event, "inputType") {
		case "insertLineBreak", "insertParagraph":
			preventDefault(event)
			w.dispatchSyntheticKeystroke("enter", gpui.Modifiers{})
			w.scheduleImeMirrorSync()
		}
	})
}

type pastedImageFile struct {
	format gpui.ImageFormat
	file   js.Value
}

func (w *webWindowInner) registerPaste() *EventListenerHandle {
	return w.listenInput("paste", func(event js.Value) {
		clipboardData := jsGet(event, "clipboardData")
		if jsIsNullish(clipboardData) {
			return
		}
		text := gpui.None[string]()
		if value, err := jsCall(clipboardData, "getData", "text/plain"); err == nil {
			if s, ok := jsString(value); ok && s != "" {
				text = gpui.Some(s)
			}
		}

		var imageFiles []pastedImageFile
		items := jsGet(clipboardData, "items")
		length := int(eventNumber(items, "length"))
		for index := range length {
			item := items.Index(index)
			if jsIsNullish(item) {
				continue
			}
			if eventString(item, "kind") != "file" {
				continue
			}
			format, ok := gpui.ImageFormatFromMimeType(eventString(item, "type"))
			if !ok {
				continue
			}
			if file, err := jsCall(item, "getAsFile"); err == nil && !jsIsNullish(file) {
				imageFiles = append(imageFiles, pastedImageFile{format: format, file: file})
			}
		}

		if text.IsNone() && len(imageFiles) == 0 {
			return
		}
		preventDefault(event)

		if len(imageFiles) == 0 {
			if s, ok := text.Get(); ok {
				withInputHandler(w, func(handler *gpui.PlatformInputHandler) struct{} {
					handler.Paste(gpui.NewClipboardItemString(s))
					return struct{}{}
				})
			}
			return
		}

		var entries []gpui.ClipboardEntry
		if s, ok := text.Get(); ok {
			entries = append(entries, gpui.ClipboardEntryString{ClipboardString: gpui.NewClipboardString(s)})
		}
		w.readPastedImages(imageFiles, entries)
	})
}

func (w *webWindowInner) readPastedImages(remaining []pastedImageFile, entries []gpui.ClipboardEntry) {
	if len(remaining) == 0 {
		if len(entries) == 0 {
			return
		}
		withInputHandler(w, func(handler *gpui.PlatformInputHandler) struct{} {
			handler.Paste(gpui.ClipboardItem{Entries: entries})
			return struct{}{}
		})
		return
	}
	next := remaining[0]
	fail := func(err js.Value) {
		slog.Error("failed to read pasted image: " + jsErrorMessage(err))
		w.readPastedImages(remaining[1:], entries)
	}
	promise, err := jsCall(next.file, "arrayBuffer")
	if err != nil {
		fail(jsErrorValue(err))
		return
	}
	if err := promiseThen(promise, func(outcome promiseOutcome) {
		if outcome.rejected {
			fail(outcome.value)
			return
		}
		array := js.Global().Get("Uint8Array").New(outcome.value)
		bytes := make([]byte, array.Length())
		js.CopyBytesToGo(bytes, array)
		entries = append(entries, gpui.ClipboardEntryImage{Image: gpui.ImageFromBytes(next.format, bytes)})
		w.readPastedImages(remaining[1:], entries)
	}); err != nil {
		fail(jsErrorValue(err))
	}
}

func (w *webWindowInner) registerCompositionStart() *EventListenerHandle {
	return w.listenInput("compositionstart", func(js.Value) {
		w.isComposing = true
	})
}

func (w *webWindowInner) registerCompositionUpdate() *EventListenerHandle {
	return w.listenInput("compositionupdate", func(event js.Value) {
		data := eventString(event, "data")
		w.isComposing = true
		withInputHandler(w, func(handler *gpui.PlatformInputHandler) struct{} {
			handler.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), data, gpui.None[gpui.Range]())
			return struct{}{}
		})
	})
}

func (w *webWindowInner) registerCompositionEnd() *EventListenerHandle {
	return w.listenInput("compositionend", func(event js.Value) {
		data := eventString(event, "data")
		w.isComposing = false
		withInputHandler(w, func(handler *gpui.PlatformInputHandler) struct{} {
			if _, ok := handler.MarkedTextRange(); ok {
				handler.ReplaceTextInRange(gpui.None[gpui.Range](), data)
			}
			handler.UnmarkText()
			return struct{}{}
		})
		w.imeMirror.adoptElementState()
		w.scheduleImeMirrorSync()
	})
}

func (w *webWindowInner) registerFocus() *EventListenerHandle {
	return eventListenerHandleAdd(w.browserWindow, "focus", func(js.Value) {
		if w.suppressFocusStatusEvents {
			return
		}
		w.refreshActiveStatus()
	})
}

func (w *webWindowInner) registerBlur() *EventListenerHandle {
	return eventListenerHandleAdd(w.browserWindow, "blur", func(js.Value) {
		if w.suppressFocusStatusEvents {
			return
		}
		w.refreshActiveStatus()
	})
}

func (w *webWindowInner) registerPointerEnter() *EventListenerHandle {
	return w.listen("pointerenter", func(js.Value) {
		w.state.isHovered = true
		withCallback(w, func(c *webWindowCallbacks) *gpui.Option[func(bool)] { return &c.hoverStatusChange },
			func(callback func(bool)) { callback(true) })
	})
}

func domKeyToGpuiKey(key string) string {
	switch key {
	case "Enter":
		return "enter"
	case "Backspace":
		return "backspace"
	case "Tab":
		return "tab"
	case "Escape":
		return "escape"
	case "Delete":
		return "delete"
	case " ":
		return "space"
	case "ArrowLeft":
		return "left"
	case "ArrowRight":
		return "right"
	case "ArrowUp":
		return "up"
	case "ArrowDown":
		return "down"
	case "Home":
		return "home"
	case "End":
		return "end"
	case "PageUp":
		return "pageup"
	case "PageDown":
		return "pagedown"
	case "Insert":
		return "insert"
	case "Control":
		return "control"
	case "Alt":
		return "alt"
	case "Shift":
		return "shift"
	case "Meta":
		return "platform"
	case "CapsLock":
		return "capslock"
	}
	if rest, ok := strings.CutPrefix(key, "F"); ok {
		digits := rest
		if len(digits) > 1 && digits[0] == '+' {
			digits = digits[1:]
		}
		if number, err := strconv.ParseUint(digits, 10, 8); err == nil && number >= 1 && number <= 35 {
			return "f" + strconv.FormatUint(number, 10)
		}
	}
	return strings.ToLower(key)
}

func domMouseButtonToGpui(button int16) gpui.MouseButton {
	switch button {
	case 0:
		return gpui.MouseButtonLeft
	case 1:
		return gpui.MouseButtonMiddle
	case 2:
		return gpui.MouseButtonRight
	case 3:
		return gpui.MouseButtonNavigate(gpui.NavigationDirectionBack)
	case 4:
		return gpui.MouseButtonNavigate(gpui.NavigationDirectionForward)
	default:
		return gpui.MouseButtonLeft
	}
}

func modifiersFromEvent(event js.Value) gpui.Modifiers {
	return gpui.Modifiers{
		Control:  eventBool(event, "ctrlKey"),
		Alt:      eventBool(event, "altKey"),
		Shift:    eventBool(event, "shiftKey"),
		Platform: eventBool(event, "metaKey"),
		Function: false,
	}
}

func modifiersFromKeyboardEvent(event js.Value, _ bool) gpui.Modifiers {
	return modifiersFromEvent(event)
}

func modifiersFromMouseEvent(event js.Value, _ bool) gpui.Modifiers {
	return modifiersFromEvent(event)
}

func modifiersFromWheelEvent(event js.Value, _ bool) gpui.Modifiers {
	return modifiersFromEvent(event)
}

func capslockFromKeyboardEvent(event js.Value) gpui.Capslock {
	on := false
	if value, err := jsCall(event, "getModifierState", "CapsLock"); err == nil {
		on, _ = jsBool(value)
	}
	return gpui.Capslock{On: on}
}

func isMacPlatform(browserWindow js.Value) bool {
	switch platformInfo(browserWindow).Os {
	case gpui.OperatingSystemMac, gpui.OperatingSystemIos:
		return true
	}
	return false
}

func platformInfo(browserWindow js.Value) gpui.PlatformInfo {
	navigator := jsGet(browserWindow, "navigator")
	userAgent := eventString(navigator, "userAgent")
	platform := eventString(navigator, "platform")
	maxTouchPoints := int32(eventNumber(navigator, "maxTouchPoints"))
	return gpui.PlatformInfo{
		Os:     operatingSystemFromNavigator(userAgent, platform, maxTouchPoints),
		Target: gpui.PlatformTargetWeb,
	}
}

func operatingSystemFromNavigator(userAgent, platform string, maxTouchPoints int32) gpui.OperatingSystem {
	switch {
	case strings.Contains(userAgent, "Android"):
		return gpui.OperatingSystemAndroid
	case strings.Contains(userAgent, "iPhone") ||
		strings.Contains(userAgent, "iPad") ||
		strings.Contains(userAgent, "iPod") ||
		(strings.Contains(platform, "Mac") && maxTouchPoints > 1):
		return gpui.OperatingSystemIos
	case strings.Contains(platform, "Mac") || strings.Contains(userAgent, "Mac"):
		return gpui.OperatingSystemMac
	case strings.Contains(platform, "Win") || strings.Contains(userAgent, "Windows"):
		return gpui.OperatingSystemWindows
	case strings.Contains(platform, "FreeBSD") || strings.Contains(userAgent, "FreeBSD"):
		return gpui.OperatingSystemFreeBsd
	case strings.Contains(platform, "Linux") ||
		strings.Contains(userAgent, "Linux") ||
		strings.Contains(userAgent, "CrOS"):
		return gpui.OperatingSystemLinux
	default:
		return gpui.OperatingSystemUnknown
	}
}

func keystrokeModifiers(key string, modifiers gpui.Modifiers) gpui.Modifiers {
	if modifiers.Shift && utf8.RuneCountInString(key) == 1 && strings.ToLower(key) == strings.ToUpper(key) {
		modifiers.Shift = false
	}
	return modifiers
}

func isModifierOnlyKey(key string) bool {
	switch key {
	case "control", "alt", "shift", "platform", "capslock", "compose", "process":
		return true
	}
	return false
}

func keystrokeInsertsText(modifiers gpui.Modifiers, isMac bool) bool {
	if isMac {
		return !modifiers.Platform && !modifiers.Control
	}
	return modifiers.IsSubsetOf(gpui.ModifiersShift()) || (modifiers.Control && modifiers.Alt)
}

func computeKeyChar(rawKey string, gpuiKey string, modifiers gpui.Modifiers) gpui.Option[string] {
	if (modifiers.Platform || modifiers.Control) && !(modifiers.Control && modifiers.Alt) {
		return gpui.None[string]()
	}

	if isModifierOnlyKey(gpuiKey) {
		return gpui.None[string]()
	}

	if gpuiKey == "space" {
		return gpui.Some(" ")
	}

	if len(rawKey) == 1 {
		return gpui.Some(rawKey)
	}

	return gpui.None[string]()
}

func pointerPositionInElement(event js.Value) gpui.Point[gpui.Pixels] {
	return mousePositionInElement(event)
}

const maxPredictionLeadMs = 10.

func predictedPointerPosition(event js.Value, position gpui.Point[gpui.Pixels]) gpui.Option[gpui.Point[gpui.Pixels]] {
	none := gpui.None[gpui.Point[gpui.Pixels]]()
	if jsGet(event, "getPredictedEvents").Type() != js.TypeFunction {
		return none
	}
	predictedEvents, err := jsCall(event, "getPredictedEvents")
	if err != nil || !jsInstanceOf(predictedEvents, "Array") {
		return none
	}
	eventTime := eventNumber(event, "timeStamp")
	bestDistance := 0.0
	best := js.Undefined()
	found := false
	for i := range predictedEvents.Length() {
		predicted := predictedEvents.Index(i)
		if !jsInstanceOf(predicted, "PointerEvent") {
			continue
		}
		lead := eventNumber(predicted, "timeStamp") - eventTime
		if lead <= 0. {
			continue
		}
		distanceToCap := math.Abs(lead - maxPredictionLeadMs)
		if !found || distanceToCap < bestDistance {
			bestDistance = distanceToCap
			best = predicted
			found = true
		}
	}
	if !found {
		return none
	}
	lead := eventNumber(best, "timeStamp") - eventTime
	scale := float32(min(maxPredictionLeadMs/lead, 1.))
	return gpui.Some(gpui.NewPoint(
		position.X+gpui.Px(float32(eventInt32(best, "clientX")-eventInt32(event, "clientX"))*scale),
		position.Y+gpui.Px(float32(eventInt32(best, "clientY")-eventInt32(event, "clientY"))*scale),
	))
}

func mousePositionInElement(event js.Value) gpui.Point[gpui.Pixels] {
	return gpui.NewPoint(gpui.Px(float32(eventInt32(event, "offsetX"))), gpui.Px(float32(eventInt32(event, "offsetY"))))
}
