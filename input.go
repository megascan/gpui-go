package gpui

import (
	"fmt"
	"reflect"
)

// EntityInputHandler is implemented by views (as *T) that handle textual input, such as editors
// and fields. Wrap the view in an ElementInputHandler and assign it during paint with
// Window.HandleInput. Embed EntityInputHandlerDefaults for the provided methods and bind it
// with BindEntityInputHandlerDefaults. See InputHandler for each method.
type EntityInputHandler[T any] interface {
	// TextForRange: see InputHandler.TextForRange.
	TextForRange(rangeUTF16 Range, adjustedRange *Option[Range], window *Window, cx *Context[T]) (string, bool)
	// SelectedTextRange: see InputHandler.SelectedTextRange.
	SelectedTextRange(ignoreDisabledInput bool, window *Window, cx *Context[T]) (UTF16Selection, bool)
	// MarkedTextRange: see InputHandler.MarkedTextRange.
	MarkedTextRange(window *Window, cx *Context[T]) (Range, bool)
	// UnmarkText: see InputHandler.UnmarkText.
	UnmarkText(window *Window, cx *Context[T])
	// Paste: see InputHandler.Paste.
	Paste(item ClipboardItem, window *Window, cx *Context[T])
	// ReplaceTextInRange: see InputHandler.ReplaceTextInRange.
	ReplaceTextInRange(rangeUTF16 Option[Range], text string, window *Window, cx *Context[T])
	// ReplaceAndMarkTextInRange: see InputHandler.ReplaceAndMarkTextInRange.
	ReplaceAndMarkTextInRange(rangeUTF16 Option[Range], newText string, newSelectedRange Option[Range], window *Window, cx *Context[T])
	// BoundsForRange: see InputHandler.BoundsForRange.
	BoundsForRange(rangeUTF16 Range, elementBounds Bounds[Pixels], window *Window, cx *Context[T]) (Bounds[Pixels], bool)
	// SelectionBoundsForRange: see InputHandler.SelectionBoundsForRange.
	SelectionBoundsForRange(rangeUTF16 Range, elementBounds Bounds[Pixels], window *Window, cx *Context[T]) []Bounds[Pixels]
	// CharacterIndexForPoint: see InputHandler.CharacterIndexForPoint.
	CharacterIndexForPoint(point Point[Pixels], window *Window, cx *Context[T]) (int, bool)
	// SetSelectedTextRange: see InputHandler.SetSelectedTextRange.
	SetSelectedTextRange(rangeUTF16 Range, window *Window, cx *Context[T])
	// TextLengthUTF16: see InputHandler.TextLengthUTF16.
	TextLengthUTF16(window *Window, cx *Context[T]) (int, bool)
	// AcceptsTextInput: see InputHandler.AcceptsTextInput.
	AcceptsTextInput(window *Window, cx *Context[T]) bool
	// TextInputConfiguration: see InputHandler.TextInputConfiguration.
	TextInputConfiguration(window *Window, cx *Context[T]) TextInputConfiguration
	// TextInputEditableRange: see InputHandler.TextInputEditableRange.
	TextInputEditableRange(window *Window, cx *Context[T]) (Range, bool)
}

// EntityInputHandlerDefaults carries the default bodies of EntityInputHandler's provided methods (D-DEFAULTS).
type EntityInputHandlerDefaults[T any] struct {
	self EntityInputHandler[T]
}

// BindEntityInputHandlerDefaults makes the provided methods that call other methods dispatch to self.
func BindEntityInputHandlerDefaults[T any](d *EntityInputHandlerDefaults[T], self EntityInputHandler[T]) {
	d.self = self
}

// Paste replaces the selection with the plain-text portion of item.
func (d *EntityInputHandlerDefaults[T]) Paste(item ClipboardItem, window *Window, cx *Context[T]) {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "EntityInputHandlerDefaults", "BindEntityInputHandlerDefaults"))
	}
	if text, ok := item.Text(); ok {
		d.self.ReplaceTextInRange(None[Range](), text, window, cx)
	}
}

// SelectionBoundsForRange returns no rectangles.
func (d *EntityInputHandlerDefaults[T]) SelectionBoundsForRange(rangeUTF16 Range, elementBounds Bounds[Pixels], window *Window, cx *Context[T]) []Bounds[Pixels] {
	return nil
}

// SetSelectedTextRange does nothing.
func (d *EntityInputHandlerDefaults[T]) SetSelectedTextRange(rangeUTF16 Range, window *Window, cx *Context[T]) {
}

// TextLengthUTF16 returns no length.
func (d *EntityInputHandlerDefaults[T]) TextLengthUTF16(window *Window, cx *Context[T]) (int, bool) {
	return 0, false
}

// AcceptsTextInput returns true.
func (d *EntityInputHandlerDefaults[T]) AcceptsTextInput(window *Window, cx *Context[T]) bool {
	return true
}

// TextInputConfiguration returns the default configuration.
func (d *EntityInputHandlerDefaults[T]) TextInputConfiguration(window *Window, cx *Context[T]) TextInputConfiguration {
	return TextInputConfiguration{}
}

// TextInputEditableRange returns no bound.
func (d *EntityInputHandlerDefaults[T]) TextInputEditableRange(window *Window, cx *Context[T]) (Range, bool) {
	return Range{}, false
}

// ElementInputHandler is the canonical InputHandler: it forwards to a view whose *V implements
// EntityInputHandler[V]. Pass one to Window.HandleInput during an element's paint.
type ElementInputHandler[V any] struct {
	InputHandlerDefaults
	view          Entity[V]
	elementBounds Bounds[Pixels]
}

// NewElementInputHandler creates a handler for view with the element's bounds. It panics when
// *V does not implement EntityInputHandler[V].
func NewElementInputHandler[V any](elementBounds Bounds[Pixels], view Entity[V]) *ElementInputHandler[V] {
	if _, ok := any((*V)(nil)).(EntityInputHandler[V]); !ok {
		panic(fmt.Sprintf("gpui: %v does not implement EntityInputHandler", reflect.TypeFor[*V]()))
	}
	h := &ElementInputHandler[V]{view: view, elementBounds: elementBounds}
	BindInputHandlerDefaults(&h.InputHandlerDefaults, h)
	return h
}

// Release releases the handler's view; the frame that owns the handler calls it when the handler is dropped.
func (h *ElementInputHandler[V]) Release() {
	h.view.Release()
}

func entityInputHandler[V any](view *V) EntityInputHandler[V] {
	return any(view).(EntityInputHandler[V])
}

type optionalResult[T any] struct {
	value T
	ok    bool
}

// SelectedTextRange forwards to the view.
func (h *ElementInputHandler[V]) SelectedTextRange(ignoreDisabledInput bool, window *Window, cx *App) (UTF16Selection, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[UTF16Selection] {
		v, ok := entityInputHandler(view).SelectedTextRange(ignoreDisabledInput, window, cx)
		return optionalResult[UTF16Selection]{v, ok}
	})
	return r.value, r.ok
}

// MarkedTextRange forwards to the view.
func (h *ElementInputHandler[V]) MarkedTextRange(window *Window, cx *App) (Range, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[Range] {
		v, ok := entityInputHandler(view).MarkedTextRange(window, cx)
		return optionalResult[Range]{v, ok}
	})
	return r.value, r.ok
}

// TextForRange forwards to the view.
func (h *ElementInputHandler[V]) TextForRange(rangeUTF16 Range, adjustedRange *Option[Range], window *Window, cx *App) (string, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[string] {
		v, ok := entityInputHandler(view).TextForRange(rangeUTF16, adjustedRange, window, cx)
		return optionalResult[string]{v, ok}
	})
	return r.value, r.ok
}

// ReplaceTextInRange forwards to the view.
func (h *ElementInputHandler[V]) ReplaceTextInRange(replacementRange Option[Range], text string, window *Window, cx *App) {
	h.view.Update(cx, func(view *V, cx *Context[V]) {
		entityInputHandler(view).ReplaceTextInRange(replacementRange, text, window, cx)
	})
}

// ReplaceAndMarkTextInRange forwards to the view.
func (h *ElementInputHandler[V]) ReplaceAndMarkTextInRange(rangeUTF16 Option[Range], newText string, newSelectedRange Option[Range], window *Window, cx *App) {
	h.view.Update(cx, func(view *V, cx *Context[V]) {
		entityInputHandler(view).ReplaceAndMarkTextInRange(rangeUTF16, newText, newSelectedRange, window, cx)
	})
}

// UnmarkText forwards to the view.
func (h *ElementInputHandler[V]) UnmarkText(window *Window, cx *App) {
	h.view.Update(cx, func(view *V, cx *Context[V]) {
		entityInputHandler(view).UnmarkText(window, cx)
	})
}

// Paste forwards to the view.
func (h *ElementInputHandler[V]) Paste(item ClipboardItem, window *Window, cx *App) {
	h.view.Update(cx, func(view *V, cx *Context[V]) {
		entityInputHandler(view).Paste(item, window, cx)
	})
}

// BoundsForRange forwards to the view with the element's bounds.
func (h *ElementInputHandler[V]) BoundsForRange(rangeUTF16 Range, window *Window, cx *App) (Bounds[Pixels], bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[Bounds[Pixels]] {
		v, ok := entityInputHandler(view).BoundsForRange(rangeUTF16, h.elementBounds, window, cx)
		return optionalResult[Bounds[Pixels]]{v, ok}
	})
	return r.value, r.ok
}

// SelectionBoundsForRange forwards to the view with the element's bounds.
func (h *ElementInputHandler[V]) SelectionBoundsForRange(rangeUTF16 Range, window *Window, cx *App) []Bounds[Pixels] {
	return h.view.UpdateR(cx, func(view *V, cx *Context[V]) []Bounds[Pixels] {
		return entityInputHandler(view).SelectionBoundsForRange(rangeUTF16, h.elementBounds, window, cx)
	})
}

// CharacterIndexForPoint forwards to the view.
func (h *ElementInputHandler[V]) CharacterIndexForPoint(point Point[Pixels], window *Window, cx *App) (int, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[int] {
		v, ok := entityInputHandler(view).CharacterIndexForPoint(point, window, cx)
		return optionalResult[int]{v, ok}
	})
	return r.value, r.ok
}

// SetSelectedTextRange forwards to the view.
func (h *ElementInputHandler[V]) SetSelectedTextRange(rangeUTF16 Range, window *Window, cx *App) {
	h.view.Update(cx, func(view *V, cx *Context[V]) {
		entityInputHandler(view).SetSelectedTextRange(rangeUTF16, window, cx)
	})
}

// ElementBounds returns the element's bounds.
func (h *ElementInputHandler[V]) ElementBounds(window *Window, cx *App) (Bounds[Pixels], bool) {
	return h.elementBounds, true
}

// TextLengthUTF16 forwards to the view.
func (h *ElementInputHandler[V]) TextLengthUTF16(window *Window, cx *App) (int, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[int] {
		v, ok := entityInputHandler(view).TextLengthUTF16(window, cx)
		return optionalResult[int]{v, ok}
	})
	return r.value, r.ok
}

// AcceptsTextInput forwards to the view.
func (h *ElementInputHandler[V]) AcceptsTextInput(window *Window, cx *App) bool {
	return h.view.UpdateR(cx, func(view *V, cx *Context[V]) bool {
		return entityInputHandler(view).AcceptsTextInput(window, cx)
	})
}

// PrefersIMEForPrintableKeys forwards to the view's AcceptsTextInput.
func (h *ElementInputHandler[V]) PrefersIMEForPrintableKeys(window *Window, cx *App) bool {
	return h.view.UpdateR(cx, func(view *V, cx *Context[V]) bool {
		return entityInputHandler(view).AcceptsTextInput(window, cx)
	})
}

// TextInputConfiguration forwards to the view.
func (h *ElementInputHandler[V]) TextInputConfiguration(window *Window, cx *App) TextInputConfiguration {
	return h.view.UpdateR(cx, func(view *V, cx *Context[V]) TextInputConfiguration {
		return entityInputHandler(view).TextInputConfiguration(window, cx)
	})
}

// TextInputEditableRange forwards to the view.
func (h *ElementInputHandler[V]) TextInputEditableRange(window *Window, cx *App) (Range, bool) {
	r := h.view.UpdateR(cx, func(view *V, cx *Context[V]) optionalResult[Range] {
		v, ok := entityInputHandler(view).TextInputEditableRange(window, cx)
		return optionalResult[Range]{v, ok}
	})
	return r.value, r.ok
}
