//go:build js && wasm

package web

import (
	"fmt"
	"log/slog"
	"strings"
	"syscall/js"
	"unicode/utf16"

	"github.com/megascan/gpui-go"
)

const contextChars = 512

const minEdgeChars = 64

type mirrorTextArea interface {
	value() string
	setValue(value string)
	selectionStart() (uint32, bool)
	selectionEnd() (uint32, bool)
	setSelectionRange(start, end uint32)
}

type mirrorInputHandler interface {
	SelectedTextRange(ignoreDisabledInput bool) (gpui.UTF16Selection, bool)
	TextForRange(rangeUTF16 gpui.Range, adjusted *gpui.Option[gpui.Range]) (string, bool)
	TextInputEditableRange() (gpui.Range, bool)
}

type jsTextArea struct {
	element js.Value
}

func (a jsTextArea) value() string {
	s, _ := jsString(jsGet(a.element, "value"))
	return s
}

func (a jsTextArea) setValue(value string) {
	_ = jsTry(func() { a.element.Set("value", value) })
}

func (a jsTextArea) selectionOffset(property string) (uint32, bool) {
	n, ok := jsNumber(jsGet(a.element, property))
	if !ok {
		return 0, false
	}
	return uint32(n), true
}

func (a jsTextArea) selectionStart() (uint32, bool) {
	return a.selectionOffset("selectionStart")
}

func (a jsTextArea) selectionEnd() (uint32, bool) {
	return a.selectionOffset("selectionEnd")
}

func (a jsTextArea) setSelectionRange(start, end uint32) {
	_, _ = jsCall(a.element, "setSelectionRange", start, end)
}

type imeMirror struct {
	element                 js.Value
	area                    mirrorTextArea
	text                    string
	selection               [2]uint32
	windowHint              int
	syncScheduled           bool
	selectionImportRejected bool
}

func newImeMirror(document js.Value, body js.Value, touchInput bool) (*imeMirror, error) {
	element, err := jsCall(document, "createElement", "textarea")
	if err != nil {
		return nil, fmt.Errorf("Failed to create textarea element: %s", jsErrorDebug(err))
	}
	if !jsInstanceOf(element, "HTMLTextAreaElement") {
		return nil, fmt.Errorf("Created element is not a textarea: %s", jsDebug(element))
	}
	style := jsGet(element, "style")
	for _, pv := range [...][2]string{
		{"position", "fixed"},
		{"top", "0"},
		{"left", "0"},
		{"width", "1px"},
		{"height", "1px"},
		{"opacity", "0"},
		{"font-size", "16px"},
	} {
		_, _ = jsCall(style, "setProperty", pv[0], pv[1])
	}
	if _, err := jsCall(body, "appendChild", element); err != nil {
		return nil, fmt.Errorf("Failed to append input to body: %s", jsErrorDebug(err))
	}
	focusMirrorElement(element)
	if touchInput {
		if _, err := jsCall(element, "setAttribute", "inputmode", "none"); err != nil {
			return nil, fmt.Errorf("Failed to configure input mode: %s", jsErrorDebug(err))
		}
	}

	m := &imeMirror{element: element, area: jsTextArea{element: element}}
	m.applyConfiguration(gpui.TextInputConfiguration{})
	return m, nil
}

func (m *imeMirror) applyConfiguration(configuration gpui.TextInputConfiguration) {
	_ = jsTry(func() { m.element.Set("spellcheck", configuration.Suggestions) })
	onOff := func(enabled bool) string {
		if enabled {
			return "on"
		}
		return "off"
	}
	_, _ = jsCall(m.element, "setAttribute", "autocomplete", onOff(configuration.Suggestions))
	_, _ = jsCall(m.element, "setAttribute", "autocorrect", onOff(configuration.Autocorrect))
	var autocapitalize string
	switch configuration.Autocapitalize {
	case gpui.AutocapitalizeNone:
		autocapitalize = "off"
	case gpui.AutocapitalizeWords:
		autocapitalize = "words"
	case gpui.AutocapitalizeSentences:
		autocapitalize = "sentences"
	case gpui.AutocapitalizeCharacters:
		autocapitalize = "characters"
	}
	_, _ = jsCall(m.element, "setAttribute", "autocapitalize", autocapitalize)
	if hint, ok := enterKeyHint(configuration.InputAction); ok {
		_, _ = jsCall(m.element, "setAttribute", "enterkeyhint", hint)
	} else {
		_, _ = jsCall(m.element, "removeAttribute", "enterkeyhint")
	}
}

func enterKeyHint(action gpui.TextInputAction) (string, bool) {
	switch action {
	case gpui.TextInputActionEnter:
		return "enter", true
	case gpui.TextInputActionDone:
		return "done", true
	case gpui.TextInputActionGo:
		return "go", true
	case gpui.TextInputActionNext:
		return "next", true
	case gpui.TextInputActionPrevious:
		return "previous", true
	case gpui.TextInputActionSearch:
		return "search", true
	case gpui.TextInputActionSend:
		return "send", true
	default:
		return "", false
	}
}

func (m *imeMirror) eventTarget() js.Value {
	return m.element
}

func (m *imeMirror) focus() {
	focusMirrorElement(m.element)
}

func focusMirrorElement(element js.Value) {
	options := js.Global().Get("Object").New()
	options.Set("preventScroll", true)
	if _, err := jsCall(element, "focus", options); err != nil {
		slog.Warn("Failed to focus IME mirror: " + jsErrorDebug(err))
	}
}

func (m *imeMirror) isFocused() bool {
	document := jsGet(browserWindow(), "document")
	if jsIsNullish(document) {
		return false
	}
	active := jsGet(document, "activeElement")
	return !jsIsNullish(active) && active.Equal(m.element)
}

func (m *imeMirror) blur() {
	_, _ = jsCall(m.element, "blur")
}

func (m *imeMirror) virtualKeyboardEnabled() bool {
	value, err := jsCall(m.element, "getAttribute", "inputmode")
	if err != nil {
		return true
	}
	mode, ok := jsString(value)
	return !ok || mode != "none"
}

func (m *imeMirror) setReadOnly(readOnly bool) {
	_ = jsTry(func() { m.element.Set("readOnly", readOnly) })
}

func (m *imeMirror) setVirtualKeyboardEnabled(enabled bool) {
	mode := "none"
	if enabled {
		mode = "text"
	}
	if _, err := jsCall(m.element, "setAttribute", "inputmode", mode); err != nil {
		slog.Warn("Failed to configure input mode: " + jsErrorDebug(err))
	}
}

func (m *imeMirror) remove() {
	_, _ = jsCall(m.element, "remove")
}

func (m *imeMirror) value() string {
	return m.area.value()
}

func (m *imeMirror) selectionStart() (uint32, bool) {
	return m.area.selectionStart()
}

func (m *imeMirror) elementSelectionEnd() (uint32, bool) {
	return m.area.selectionEnd()
}

func (m *imeMirror) storedText() string {
	return m.text
}

func (m *imeMirror) storedSelection() (uint32, uint32) {
	return m.selection[0], m.selection[1]
}

func (m *imeMirror) adoptElementState() {
	m.text = m.area.value()
	selectionStart, ok := m.selectionStart()
	if !ok {
		selectionStart = 0
	}
	selectionEnd, ok := m.area.selectionEnd()
	if !ok {
		selectionEnd = selectionStart
	}
	m.selection = [2]uint32{selectionStart, selectionEnd}
}

func (m *imeMirror) rejectSelectionImport() {
	m.selectionImportRejected = true
}

func imeMirrorScheduleSync(window *webWindowInner) {
	mirror := window.imeMirror
	if mirror.syncScheduled {
		return
	}
	mirror.syncScheduled = true
	closure := jsFuncOnce(func([]js.Value) {
		mirror.syncScheduled = false
		syncWindowImeMirror(window)
	})
	if _, err := jsCall(window.browserWindow, "setTimeout", closure); err != nil {
		closure.Release()
	}
}

func syncWindowImeMirror(window *webWindowInner) {
	handler := window.state.inputHandler
	window.state.inputHandler = nil
	defer func() { window.state.inputHandler = handler }()
	var inputHandler mirrorInputHandler
	if handler != nil {
		inputHandler = handler
	}
	window.imeMirror.sync(window.isComposing, inputHandler)
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

func textForRange(handler mirrorInputHandler, r gpui.Range, adjusted *gpui.Option[gpui.Range]) string {
	if handler == nil {
		return ""
	}
	text, _ := handler.TextForRange(r, adjusted)
	return text
}

func (m *imeMirror) sync(isComposing bool, handler mirrorInputHandler) {
	if isComposing {
		return
	}
	rejected := m.selectionImportRejected
	m.selectionImportRejected = false
	if !rejected && m.text == m.area.value() {
		liveStart, ok := m.selectionStart()
		if !ok {
			liveStart = 0
		}
		liveEnd, ok := m.elementSelectionEnd()
		if !ok {
			liveEnd = liveStart
		}
		if [2]uint32{liveStart, liveEnd} != m.selection {
			return
		}
	}
	var selection gpui.UTF16Selection
	hasSelection := false
	if handler != nil {
		selection, hasSelection = handler.SelectedTextRange(false)
	}
	if !hasSelection {
		if m.text != "" {
			m.area.setValue("")
			m.text = ""
		}
		m.selection = [2]uint32{0, 0}
		return
	}
	editableRange := gpui.None[gpui.Range]()
	if r, ok := handler.TextInputEditableRange(); ok {
		editableRange = gpui.Some(r)
	}

	if m.isConsistent(handler, selection.Range, editableRange, minEdgeChars) {
		return
	}

	if m.moveSelectionWithinWindow(handler, selection.Range, editableRange, minEdgeChars) {
		return
	}

	windowRange := gpui.Range{
		Start: max(selection.Range.Start-contextChars, 0),
		End:   selection.Range.End + contextChars,
	}
	if r, ok := editableRange.Get(); ok {
		windowRange.Start = max(windowRange.Start, r.Start)
		windowRange.End = max(min(windowRange.End, r.End), windowRange.Start)
	}
	adjusted := gpui.None[gpui.Range]()
	text := textForRange(handler, windowRange, &adjusted)
	windowStart := adjusted.UnwrapOr(windowRange).Start

	if m.text != text || m.area.value() != text {
		m.area.setValue(text)
		m.text = text
	}

	m.windowHint = windowStart
	selectionStart := uint32(max(selection.Range.Start-windowStart, 0))
	selectionEnd := uint32(max(selection.Range.End-windowStart, 0))
	currentStart, startOk := m.area.selectionStart()
	currentEnd, endOk := m.area.selectionEnd()
	if !startOk || currentStart != selectionStart || !endOk || currentEnd != selectionEnd {
		m.area.setSelectionRange(selectionStart, selectionEnd)
	}
	actualStart, ok := m.area.selectionStart()
	if !ok {
		actualStart = selectionStart
	}
	actualEnd, ok := m.area.selectionEnd()
	if !ok {
		actualEnd = selectionEnd
	}
	m.selection = [2]uint32{actualStart, actualEnd}
}

func (m *imeMirror) moveSelectionWithinWindow(handler mirrorInputHandler, appSelection gpui.Range, editableRange gpui.Option[gpui.Range], minEdge int) bool {
	storedText := m.text
	storedLength := utf16Len(storedText)
	if storedLength == 0 || m.area.value() != storedText {
		return false
	}
	windowStart := m.windowHint

	editable, hasEditable := editableRange.Get()
	if hasEditable && (windowStart < editable.Start || windowStart+storedLength > editable.End) {
		return false
	}
	leftBoundary := 0
	if hasEditable {
		leftBoundary = editable.Start
	}
	if appSelection.Start < windowStart {
		return false
	}
	selectionStart := appSelection.Start - windowStart
	selectionEnd := selectionStart + (appSelection.End - appSelection.Start)
	if selectionEnd > storedLength {
		return false
	}
	if selectionStart < minEdge && windowStart > leftBoundary {
		return false
	}

	adjusted := gpui.None[gpui.Range]()
	documentText := textForRange(handler, gpui.Range{Start: windowStart, End: windowStart + storedLength + 1}, &adjusted)
	documentTextLength := utf16Len(documentText)
	windowAtRightBoundary := documentTextLength == storedLength ||
		(hasEditable && windowStart+storedLength >= editable.End)
	if selectionEnd+minEdge > storedLength && !windowAtRightBoundary {
		return false
	}
	if !strings.HasPrefix(documentText, storedText) || documentTextLength > storedLength+1 {
		return false
	}

	m.area.setSelectionRange(uint32(selectionStart), uint32(selectionEnd))
	actualStart, startOk := m.area.selectionStart()
	actualEnd, endOk := m.area.selectionEnd()
	if !startOk || actualStart != uint32(selectionStart) || !endOk || actualEnd != uint32(selectionEnd) {
		return false
	}
	m.selection = [2]uint32{uint32(selectionStart), uint32(selectionEnd)}
	return true
}

func (m *imeMirror) isConsistent(handler mirrorInputHandler, appSelection gpui.Range, editableRange gpui.Option[gpui.Range], minEdge int) bool {
	elementSelectionStart := int(m.selection[0])
	elementSelectionEnd := int(m.selection[1])
	storedText := m.text
	storedLength := utf16Len(storedText)

	if storedLength == 0 {
		return false
	}
	if start, ok := m.area.selectionStart(); !ok || int(start) != elementSelectionStart {
		return false
	}
	if end, ok := m.area.selectionEnd(); !ok || int(end) != elementSelectionEnd {
		return false
	}
	if appSelection.Start < elementSelectionStart {
		return false
	}
	appWindowStart := appSelection.Start - elementSelectionStart
	editable, hasEditable := editableRange.Get()
	if hasEditable && (appWindowStart < editable.Start || appWindowStart+storedLength > editable.End) {
		return false
	}
	leftBoundary := 0
	if hasEditable {
		leftBoundary = editable.Start
	}
	hasLeftContext := elementSelectionStart >= minEdge || appWindowStart <= leftBoundary
	rightContext := max(storedLength-elementSelectionEnd, 0)
	if !hasLeftContext || rightContext < minEdge {
		windowEnd := appWindowStart + storedLength
		var atRightBoundary bool
		if hasEditable {
			atRightBoundary = windowEnd >= editable.End
		} else {
			adjusted := gpui.None[gpui.Range]()
			atRightBoundary = textForRange(handler, gpui.Range{Start: windowEnd, End: windowEnd + 1}, &adjusted) == ""
		}
		if !hasLeftContext || !atRightBoundary {
			return false
		}
	}
	adjusted := gpui.None[gpui.Range]()
	documentText := textForRange(handler, gpui.Range{Start: appWindowStart, End: appWindowStart + storedLength}, &adjusted)
	if documentText != storedText {
		return false
	}
	if m.area.value() != storedText {
		return false
	}
	return appSelection.End >= appWindowStart && appSelection.End-appWindowStart == elementSelectionEnd
}
