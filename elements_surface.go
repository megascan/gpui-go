package gpui

import "runtime"

// SurfaceSource is a source of a surface's content: on macOS and iOS a CoreVideo image buffer
// (CVPixelBufferRef); elsewhere it has no variants and a surface paints nothing.
type SurfaceSource struct {
	// Surface is the retained CVPixelBufferRef.
	Surface uintptr
}

var surfaceSourceSize func(surface uintptr) Size[DevicePixels]

// RegisterSurfaceSourceSize registers the function that reads a CVPixelBufferRef's width and
// height; the macOS and iOS platforms call it from init.
func RegisterSurfaceSourceSize(size func(surface uintptr) Size[DevicePixels]) {
	surfaceSourceSize = size
}

// Surface is a surface element.
type Surface struct {
	Styled[*Surface]
	Fluent[*Surface]
	source    SurfaceSource
	objectFit ObjectFit
	style     StyleRefinement
}

// NewSurface creates a new surface element.
func NewSurface(source SurfaceSource) *Surface {
	s := &Surface{source: source, objectFit: ObjectFitContain}
	BindStyled(&s.Styled, s)
	BindFluent(&s.Fluent, s)
	return s
}

// ObjectFit sets the object fit for the image.
func (s *Surface) ObjectFit(objectFit ObjectFit) *Surface {
	s.objectFit = objectFit
	return s
}

// ElementID reports that a surface has no id.
func (s *Surface) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (s *Surface) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests a leaf layout node with the surface style.
func (s *Surface) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Refine(&s.style)
	layoutID := window.RequestLayout(style, nil, cx)
	return layoutID, struct{}{}
}

// Prepaint does nothing.
func (s *Surface) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *Window, _ *App) struct{} {
	return struct{}{}
}

// Paint paints the image buffer fitted into bounds on macOS and iOS.
func (s *Surface) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, _ *App) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "ios" {
		return
	}
	if surfaceSourceSize == nil {
		panic("gpui: no surface size function registered")
	}
	size := surfaceSourceSize(s.source.Surface)
	newBounds := s.objectFit.GetBounds(bounds, size)
	window.PaintSurface(newBounds, s.source.Surface)
}

// Style returns the surface style refinement.
func (s *Surface) Style() *StyleRefinement {
	return &s.style
}

// IntoAnyElement converts the surface into an AnyElement.
func (s *Surface) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](s)
}
