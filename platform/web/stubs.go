//go:build js && wasm

package web

import (
	"syscall/js"

	"github.com/megascan/gpui-go"
)

// EventListenerHandle is a DOM event listener that is removed from its target by Close.
type EventListenerHandle struct {
	target    js.Value
	eventName string
	closure   js.Func
	closed    bool
}

func eventListenerHandleAdd(target js.Value, eventName string, handler func(event js.Value)) *EventListenerHandle {
	closure := js.FuncOf(func(this js.Value, args []js.Value) any {
		handler(argOrUndefined(args))
		return nil
	})
	_, _ = jsCall(target, "addEventListener", eventName, closure)
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

func (w *webWindowInner) registerEventListeners() *WebEventListeners {
	panic("gpui: unimplemented registerEventListeners")
}

func (w *webWindowInner) showVirtualKeyboard() {
	panic("gpui: unimplemented showVirtualKeyboard")
}

func (w *webWindowInner) setVirtualKeyboardEnabled(enabled bool) {
	panic("gpui: unimplemented setVirtualKeyboardEnabled")
}

func (w *webWindowInner) focusImeMirror() {
	panic("gpui: unimplemented focusImeMirror")
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

func isMacPlatform(browserWindow js.Value) bool {
	panic("gpui: unimplemented isMacPlatform")
}

func platformInfo(browserWindow js.Value) gpui.PlatformInfo {
	panic("gpui: unimplemented platformInfo")
}

type imeMirror struct {
	element js.Value
}

func newImeMirror(document js.Value, body js.Value, touchInput bool) (*imeMirror, error) {
	panic("gpui: unimplemented newImeMirror")
}

func (m *imeMirror) applyConfiguration(configuration gpui.TextInputConfiguration) {
	panic("gpui: unimplemented imeMirror.applyConfiguration")
}

func (m *imeMirror) focus() {
	panic("gpui: unimplemented imeMirror.focus")
}

func (m *imeMirror) isFocused() bool {
	panic("gpui: unimplemented imeMirror.isFocused")
}

func (m *imeMirror) blur() {
	panic("gpui: unimplemented imeMirror.blur")
}

func (m *imeMirror) virtualKeyboardEnabled() bool {
	panic("gpui: unimplemented imeMirror.virtualKeyboardEnabled")
}

func (m *imeMirror) setReadOnly(readOnly bool) {
	panic("gpui: unimplemented imeMirror.setReadOnly")
}

func (m *imeMirror) remove() {
	panic("gpui: unimplemented imeMirror.remove")
}

// CanvasFontFallback controls browser-font fallback when loaded fonts lack a glyph or its emoji
// presentation.
type CanvasFontFallback uint8

const (
	// CanvasFontFallbackDisabled uses only fonts loaded into GPUI.
	CanvasFontFallbackDisabled CanvasFontFallback = iota
	// CanvasFontFallbackEmoji uses Canvas for eligible emoji; it is the default.
	CanvasFontFallbackEmoji
	// CanvasFontFallbackEmojiAndCjk also allows approximate rendering of eligible horizontal CJK text.
	CanvasFontFallbackEmojiAndCjk
)

// DefaultCanvasFontFallback returns CanvasFontFallbackEmoji.
func DefaultCanvasFontFallback() CanvasFontFallback {
	return CanvasFontFallbackEmoji
}

func newWebTextSystem(fallbackFontFamily string, canvasFontFallback CanvasFontFallback) gpui.PlatformTextSystem {
	panic("gpui: unimplemented newWebTextSystem")
}
