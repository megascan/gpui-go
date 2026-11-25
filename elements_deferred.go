package gpui

// NewDeferred builds a Deferred element, which delays the layout and paint of its child.
func NewDeferred(child IntoElement) *Deferred {
	d := &Deferred{child: Some(child.IntoAnyElement())}
	BindFluent(&d.Fluent, d)
	return d
}

// Deferred is an element which delays the painting of its child until after all of its
// ancestors, while keeping its layout as part of the current element tree.
type Deferred struct {
	Fluent[*Deferred]
	child    Option[AnyElement]
	priority int
}

// WithPriority sets the priority of the deferred element, which determines the drawing order
// relative to other deferred elements, with higher values being drawn on top.
func (d *Deferred) WithPriority(priority int) *Deferred {
	d.priority = priority
	return d
}

// ElementID reports that a deferred element has no id.
func (d *Deferred) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (d *Deferred) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests the child's layout.
func (d *Deferred) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	child := d.child.AsMut()
	if child == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	layoutID := child.RequestLayout(window, cx)
	return layoutID, struct{}{}
}

// Prepaint hands the child to the window to draw after its ancestors.
func (d *Deferred) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, window *Window, _ *App) struct{} {
	child := d.child.Take().Unwrap()
	elementOffset := window.ElementOffset()
	window.DeferDraw(child, elementOffset, d.priority, None[ContentMask[Pixels]]())
	return struct{}{}
}

// Paint does nothing: the window paints the child with the deferred draws.
func (d *Deferred) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *struct{}, _ *Window, _ *App) {
}

// Priority sets a priority for the element. A higher priority conceptually means painting the
// element on top of deferred draws with a lower priority (i.e. closer to the viewer).
func (d *Deferred) Priority(priority int) *Deferred {
	d.priority = priority
	return d
}

// IntoAnyElement converts the deferred element into an AnyElement.
func (d *Deferred) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](d)
}
