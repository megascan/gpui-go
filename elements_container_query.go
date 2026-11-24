package gpui

// NewContainerQuery constructs a container query element with the given render callback, in the
// spirit of CSS container queries. The callback receives the size the element was assigned during
// layout and returns the contents to display within it.
//
// By default the element fills its parent (equivalent to SizeFull); use the Styled methods to
// size it differently. Because the contents don't exist until after layout, they cannot influence
// the element's size.
func NewContainerQuery[E IntoElement](render func(size Size[Pixels], window *Window, cx *App) E) *ContainerQuery {
	var baseStyle StyleRefinement
	baseStyle.Size.Width = Some(Relative(1.).ToLength())
	baseStyle.Size.Height = Some(Relative(1.).ToLength())

	q := &ContainerQuery{
		render: func(size Size[Pixels], window *Window, cx *App) AnyElement {
			return render(size, window, cx).IntoAnyElement()
		},
		style: baseStyle,
	}
	BindStyled(&q.Styled, q)
	BindFluent(&q.Fluent, q)
	return q
}

// ContainerQuery is a container query element, created with NewContainerQuery.
type ContainerQuery struct {
	Styled[*ContainerQuery]
	Fluent[*ContainerQuery]
	render func(size Size[Pixels], window *Window, cx *App) AnyElement
	style  StyleRefinement
}

// ElementID reports that a container query has no id.
func (q *ContainerQuery) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (q *ContainerQuery) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests a leaf layout node with the element's style.
func (q *ContainerQuery) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Refine(&q.style)
	layoutID := window.RequestLayout(style, nil, cx)
	return layoutID, struct{}{}
}

// Prepaint renders the contents for the assigned size, then lays them out and prepaints them.
func (q *ContainerQuery) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) Option[AnyElement] {
	render := q.render
	if render == nil {
		return None[AnyElement]()
	}
	q.render = nil
	child := render(bounds.Size, window, cx)
	child.LayoutAsRoot(bounds.Size.MapOf(AvailableSpaceDefinite), window, cx)
	child.PrepaintAt(bounds.Origin, window, cx)
	return Some(child)
}

// Paint paints the rendered contents.
func (q *ContainerQuery) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, prepaint *Option[AnyElement], window *Window, cx *App) {
	if child, ok := prepaint.Get(); ok {
		child.Paint(window, cx)
	}
}

// Style returns the element's style refinement.
func (q *ContainerQuery) Style() *StyleRefinement {
	return &q.style
}

// IntoAnyElement converts the container query into an AnyElement.
func (q *ContainerQuery) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, Option[AnyElement]](q)
}
