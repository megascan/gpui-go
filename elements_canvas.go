package gpui

// NewCanvas constructs a canvas element with the given paint callback. Useful for adding short
// term custom drawing to a view.
func NewCanvas[T any](prepaint func(bounds Bounds[Pixels], window *Window, cx *App) T, paint func(bounds Bounds[Pixels], state T, window *Window, cx *App)) *Canvas[T] {
	c := &Canvas[T]{prepaint: prepaint, paint: paint}
	BindStyled(&c.Styled, c)
	BindFluent(&c.Fluent, c)
	return c
}

// Canvas is an element meant for accessing the low level paint API without defining a whole
// custom element.
type Canvas[T any] struct {
	Styled[*Canvas[T]]
	Fluent[*Canvas[T]]
	prepaint func(bounds Bounds[Pixels], window *Window, cx *App) T
	paint    func(bounds Bounds[Pixels], state T, window *Window, cx *App)
	style    StyleRefinement
}

// ElementID reports that a canvas has no id.
func (c *Canvas[T]) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (c *Canvas[T]) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests a leaf layout node with the canvas style.
func (c *Canvas[T]) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, Style) {
	style := DefaultStyle()
	style.Refine(&c.style)
	layoutID := window.RequestLayout(style, nil, cx)
	return layoutID, style
}

// Prepaint runs the prepaint callback.
func (c *Canvas[T]) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *Style, window *Window, cx *App) Option[T] {
	prepaint := c.prepaint
	if prepaint == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	c.prepaint = nil
	return Some(prepaint(bounds, window, cx))
}

// Paint paints the canvas style and runs the paint callback.
func (c *Canvas[T]) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], style *Style, prepaint *Option[T], window *Window, cx *App) {
	state := prepaint.Take().Unwrap()
	style.Paint(bounds, window, cx, func(window *Window, cx *App) {
		paint := c.paint
		if paint == nil {
			panic("called `Option::unwrap()` on a `None` value")
		}
		c.paint = nil
		paint(bounds, state, window, cx)
	})
}

// Style returns the canvas style refinement.
func (c *Canvas[T]) Style() *StyleRefinement {
	return &c.style
}

// IntoAnyElement converts the canvas into an AnyElement.
func (c *Canvas[T]) IntoAnyElement() AnyElement {
	return NewAnyElement[Style, Option[T]](c)
}
