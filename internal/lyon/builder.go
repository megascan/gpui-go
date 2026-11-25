package lyon

import "github.com/megascan/gpui-go/internal/lyon/geom"

// PathBuilder is lyon_path's PathBuilder trait, the interface for path building.
type PathBuilder interface {
	NumAttributes() int
	Begin(at Point, customAttributes Attributes) EndpointID
	End(close bool)
	Close()
	LineTo(to Point, customAttributes Attributes) EndpointID
	QuadraticBezierTo(ctrl, to Point, customAttributes Attributes) EndpointID
	CubicBezierTo(ctrl1, ctrl2, to Point, customAttributes Attributes) EndpointID
	Reserve(endpoints, ctrlPoints int)
}

// Build is lyon_path's Build trait, for builders that produce a Path.
type Build interface {
	Build() Path
}

// PathBuilderPathEvent is PathBuilder::path_event.
func PathBuilderPathEvent(b PathBuilder, event PathEvent, attributes Attributes) {
	switch event.Kind {
	case EventBegin:
		b.Begin(event.At, attributes)
	case EventLine:
		b.LineTo(event.To, attributes)
	case EventQuadratic:
		b.QuadraticBezierTo(event.Ctrl, event.To, attributes)
	case EventCubic:
		b.CubicBezierTo(event.Ctrl1, event.Ctrl2, event.To, attributes)
	case EventEnd:
		b.End(event.Close)
	}
}

// PathBuilderAddPolygon is PathBuilder::add_polygon.
func PathBuilderAddPolygon(b PathBuilder, polygon Polygon[Point], attributes Attributes) {
	if len(polygon.Points) == 0 {
		return
	}

	b.Reserve(len(polygon.Points), 0)

	b.Begin(polygon.Points[0], attributes)
	for _, p := range polygon.Points[1:] {
		b.LineTo(p, attributes)
	}

	b.End(polygon.Closed)
}

// SvgPathBuilder is lyon_path's SvgPathBuilder trait, a path building interface that tries
// to stay close to SVG's path specification.
type SvgPathBuilder interface {
	MoveTo(to Point) EndpointID
	Close()
	LineTo(to Point) EndpointID
	QuadraticBezierTo(ctrl, to Point) EndpointID
	CubicBezierTo(ctrl1, ctrl2, to Point) EndpointID
	RelativeMoveTo(to Vector)
	RelativeLineTo(to Vector)
	RelativeQuadraticBezierTo(ctrl, to Vector)
	RelativeCubicBezierTo(ctrl1, ctrl2, to Vector)
	SmoothCubicBezierTo(ctrl2, to Point)
	SmoothRelativeCubicBezierTo(ctrl2, to Vector)
	SmoothQuadraticBezierTo(to Point)
	SmoothRelativeQuadraticBezierTo(to Vector)
	HorizontalLineTo(x float32)
	RelativeHorizontalLineTo(dx float32)
	VerticalLineTo(y float32)
	RelativeVerticalLineTo(dy float32)
	ArcTo(radii Vector, xRotation Angle, flags ArcFlags, to Point)
	RelativeArcTo(radii Vector, xRotation Angle, flags ArcFlags, to Vector)
	Reserve(endpoints, ctrlPoints int)
	AddPolygon(polygon Polygon[Point])
}

// NoAttributes is lyon_path's NoAttributes, a builder wrapper that hides custom attributes.
type NoAttributes[B PathBuilder] struct {
	inner B
}

// NoAttributesWrap is NoAttributes::wrap.
func NoAttributesWrap[B PathBuilder](inner B) *NoAttributes[B] {
	if inner.NumAttributes() != 0 {
		panic("assertion `left == right` failed")
	}
	return &NoAttributes[B]{inner: inner}
}

// Begin is NoAttributes::begin.
func (n *NoAttributes[B]) Begin(at Point) EndpointID { return n.inner.Begin(at, nil) }

// End is NoAttributes::end.
func (n *NoAttributes[B]) End(close bool) { n.inner.End(close) }

// Close is NoAttributes::close.
func (n *NoAttributes[B]) Close() { n.inner.Close() }

// LineTo is NoAttributes::line_to.
func (n *NoAttributes[B]) LineTo(to Point) EndpointID { return n.inner.LineTo(to, nil) }

// QuadraticBezierTo is NoAttributes::quadratic_bezier_to.
func (n *NoAttributes[B]) QuadraticBezierTo(ctrl, to Point) EndpointID {
	return n.inner.QuadraticBezierTo(ctrl, to, nil)
}

// CubicBezierTo is NoAttributes::cubic_bezier_to.
func (n *NoAttributes[B]) CubicBezierTo(ctrl1, ctrl2, to Point) EndpointID {
	return n.inner.CubicBezierTo(ctrl1, ctrl2, to, nil)
}

// Reserve is NoAttributes::reserve.
func (n *NoAttributes[B]) Reserve(endpoints, ctrlPoints int) { n.inner.Reserve(endpoints, ctrlPoints) }

// PathEvent is NoAttributes::path_event.
func (n *NoAttributes[B]) PathEvent(event PathEvent) { PathBuilderPathEvent(n.inner, event, nil) }

// AddPolygon is NoAttributes::add_polygon.
func (n *NoAttributes[B]) AddPolygon(polygon Polygon[Point]) {
	PathBuilderAddPolygon(n.inner, polygon, nil)
}

// WithSvg is NoAttributes::with_svg.
func (n *NoAttributes[B]) WithSvg() *WithSvg[B] { return NewWithSvg(n.inner) }

// Build is NoAttributes::build. The inner builder must implement Build.
func (n *NoAttributes[B]) Build() Path { return buildOf(n.inner) }

// Inner is NoAttributes::inner.
func (n *NoAttributes[B]) Inner() B { return n.inner }

// IntoInner is NoAttributes::into_inner.
func (n *NoAttributes[B]) IntoInner() B { return n.inner }

func buildOf(b PathBuilder) Path {
	builder, ok := b.(Build)
	if !ok {
		panic("lyon: builder does not implement Build")
	}
	return builder.Build()
}

// WithSvg is lyon_path's WithSvg, a builder adapter that implements SvgPathBuilder.
type WithSvg[B PathBuilder] struct {
	builder         B
	firstPosition   Point
	currentPosition Point
	lastCtrl        Point
	lastCmd         verb
	needMoveto      bool
	isEmpty         bool
	attributeBuffer []float32
}

// NewWithSvg is WithSvg::new.
func NewWithSvg[B PathBuilder](builder B) *WithSvg[B] {
	attributeBuffer := make([]float32, builder.NumAttributes())
	return &WithSvg[B]{
		builder:         builder,
		firstPosition:   NewPoint(0, 0),
		currentPosition: NewPoint(0, 0),
		lastCtrl:        NewPoint(0, 0),
		needMoveto:      true,
		isEmpty:         true,
		lastCmd:         verbEnd,
		attributeBuffer: attributeBuffer,
	}
}

// Build is WithSvg::build. The wrapped builder must implement Build.
func (w *WithSvg[B]) Build() Path {
	w.endIfNeeded()
	return buildOf(w.builder)
}

// MoveTo is WithSvg::move_to.
func (w *WithSvg[B]) MoveTo(to Point) EndpointID {
	w.endIfNeeded()

	id := w.builder.Begin(to, w.attributeBuffer)

	w.isEmpty = false
	w.needMoveto = false
	w.firstPosition = to
	w.currentPosition = to
	w.lastCmd = verbBegin

	return id
}

// LineTo is WithSvg::line_to.
func (w *WithSvg[B]) LineTo(to Point) EndpointID {
	if id, ok := w.beginIfNeeded(to); ok {
		return id
	}

	w.currentPosition = to
	w.lastCmd = verbLineTo

	return w.builder.LineTo(to, w.attributeBuffer)
}

// Close is WithSvg::close.
func (w *WithSvg[B]) Close() {
	if w.needMoveto {
		return
	}

	w.currentPosition = w.firstPosition
	w.needMoveto = true
	w.lastCmd = verbClose

	w.builder.Close()
}

// QuadraticBezierTo is WithSvg::quadratic_bezier_to.
func (w *WithSvg[B]) QuadraticBezierTo(ctrl, to Point) EndpointID {
	if id, ok := w.beginIfNeeded(to); ok {
		return id
	}

	w.currentPosition = to
	w.lastCmd = verbQuadraticTo
	w.lastCtrl = ctrl

	return w.builder.QuadraticBezierTo(ctrl, to, w.attributeBuffer)
}

// CubicBezierTo is WithSvg::cubic_bezier_to.
func (w *WithSvg[B]) CubicBezierTo(ctrl1, ctrl2, to Point) EndpointID {
	if id, ok := w.beginIfNeeded(to); ok {
		return id
	}

	w.currentPosition = to
	w.lastCmd = verbCubicTo
	w.lastCtrl = ctrl2

	return w.builder.CubicBezierTo(ctrl1, ctrl2, to, w.attributeBuffer)
}

// Arc is WithSvg::arc.
func (w *WithSvg[B]) Arc(center Point, radii Vector, sweepAngle, xRotation Angle) {
	nanCheck(center)
	nanCheck(radii.ToPoint())

	w.lastCtrl = w.currentPosition

	if w.currentPosition.ApproxEq(center) {
		return
	}

	startAngle := w.currentPosition.Sub(center).AngleFromXAxis().Sub(xRotation)

	arc := geom.Arc{
		Center:     center,
		Radii:      radii,
		StartAngle: startAngle,
		SweepAngle: sweepAngle,
		XRotation:  xRotation,
	}

	arcStart := arc.From()
	if w.needMoveto {
		w.MoveTo(arcStart)
	} else if arcStart.Sub(w.currentPosition).SquareLength() > 0.01 {
		w.builder.LineTo(arcStart, w.attributeBuffer)
	}

	arc.ForEachQuadraticBezier(func(curve *geom.QuadraticBezierSegment) {
		w.builder.QuadraticBezierTo(curve.Ctrl, curve.To, w.attributeBuffer)
		w.currentPosition = curve.To
	})
}

func (w *WithSvg[B]) beginIfNeeded(def Point) (EndpointID, bool) {
	if w.needMoveto {
		return w.insertMoveTo(def)
	}
	return 0, false
}

func (w *WithSvg[B]) insertMoveTo(def Point) (EndpointID, bool) {
	if w.isEmpty {
		return w.MoveTo(def), true
	}

	w.MoveTo(w.firstPosition)

	return 0, false
}

func (w *WithSvg[B]) endIfNeeded() {
	if w.lastCmd <= verbBegin {
		w.builder.End(false)
	}
}

// CurrentPosition is WithSvg::current_position.
func (w *WithSvg[B]) CurrentPosition() Point { return w.currentPosition }

// Reserve is WithSvg::reserve.
func (w *WithSvg[B]) Reserve(endpoints, ctrlPoints int) { w.builder.Reserve(endpoints, ctrlPoints) }

func (w *WithSvg[B]) getSmoothCubicCtrl() Point {
	if w.lastCmd == verbCubicTo {
		return w.currentPosition.Add(w.currentPosition.Sub(w.lastCtrl))
	}
	return w.currentPosition
}

func (w *WithSvg[B]) getSmoothQuadraticCtrl() Point {
	if w.lastCmd == verbQuadraticTo {
		return w.currentPosition.Add(w.currentPosition.Sub(w.lastCtrl))
	}
	return w.currentPosition
}

func (w *WithSvg[B]) relativeToAbsolute(v Vector) Point { return w.currentPosition.Add(v) }

// RelativeMoveTo is SvgPathBuilder::relative_move_to.
func (w *WithSvg[B]) RelativeMoveTo(to Vector) {
	w.MoveTo(w.relativeToAbsolute(to))
}

// RelativeLineTo is SvgPathBuilder::relative_line_to.
func (w *WithSvg[B]) RelativeLineTo(to Vector) {
	w.LineTo(w.relativeToAbsolute(to))
}

// RelativeQuadraticBezierTo is SvgPathBuilder::relative_quadratic_bezier_to.
func (w *WithSvg[B]) RelativeQuadraticBezierTo(ctrl, to Vector) {
	c := w.relativeToAbsolute(ctrl)
	t := w.relativeToAbsolute(to)
	w.QuadraticBezierTo(c, t)
}

// RelativeCubicBezierTo is SvgPathBuilder::relative_cubic_bezier_to.
func (w *WithSvg[B]) RelativeCubicBezierTo(ctrl1, ctrl2, to Vector) {
	t := w.relativeToAbsolute(to)
	c1 := w.relativeToAbsolute(ctrl1)
	c2 := w.relativeToAbsolute(ctrl2)
	w.CubicBezierTo(c1, c2, t)
}

// SmoothCubicBezierTo is SvgPathBuilder::smooth_cubic_bezier_to.
func (w *WithSvg[B]) SmoothCubicBezierTo(ctrl2, to Point) {
	ctrl1 := w.getSmoothCubicCtrl()
	w.CubicBezierTo(ctrl1, ctrl2, to)
}

// SmoothRelativeCubicBezierTo is SvgPathBuilder::smooth_relative_cubic_bezier_to.
func (w *WithSvg[B]) SmoothRelativeCubicBezierTo(ctrl2, to Vector) {
	ctrl1 := w.getSmoothCubicCtrl()
	c2 := w.relativeToAbsolute(ctrl2)
	t := w.relativeToAbsolute(to)
	w.CubicBezierTo(ctrl1, c2, t)
}

// SmoothQuadraticBezierTo is SvgPathBuilder::smooth_quadratic_bezier_to.
func (w *WithSvg[B]) SmoothQuadraticBezierTo(to Point) {
	ctrl := w.getSmoothQuadraticCtrl()
	w.QuadraticBezierTo(ctrl, to)
}

// SmoothRelativeQuadraticBezierTo is SvgPathBuilder::smooth_relative_quadratic_bezier_to.
func (w *WithSvg[B]) SmoothRelativeQuadraticBezierTo(to Vector) {
	ctrl := w.getSmoothQuadraticCtrl()
	t := w.relativeToAbsolute(to)
	w.QuadraticBezierTo(ctrl, t)
}

// HorizontalLineTo is SvgPathBuilder::horizontal_line_to.
func (w *WithSvg[B]) HorizontalLineTo(x float32) {
	y := w.currentPosition.Y
	w.LineTo(NewPoint(x, y))
}

// RelativeHorizontalLineTo is SvgPathBuilder::relative_horizontal_line_to.
func (w *WithSvg[B]) RelativeHorizontalLineTo(dx float32) {
	p := w.currentPosition
	w.LineTo(NewPoint(p.X+dx, p.Y))
}

// VerticalLineTo is SvgPathBuilder::vertical_line_to.
func (w *WithSvg[B]) VerticalLineTo(y float32) {
	x := w.currentPosition.X
	w.LineTo(NewPoint(x, y))
}

// RelativeVerticalLineTo is SvgPathBuilder::relative_vertical_line_to.
func (w *WithSvg[B]) RelativeVerticalLineTo(dy float32) {
	p := w.currentPosition
	w.LineTo(NewPoint(p.X, p.Y+dy))
}

// ArcTo is SvgPathBuilder::arc_to.
func (w *WithSvg[B]) ArcTo(radii Vector, xRotation Angle, flags ArcFlags, to Point) {
	svgArc := geom.SvgArc{
		From:      w.currentPosition,
		To:        to,
		Radii:     radii,
		XRotation: xRotation,
		Flags:     ArcFlags{LargeArc: flags.LargeArc, Sweep: flags.Sweep},
	}

	if svgArc.IsStraightLine() {
		w.LineTo(to)
	} else {
		arc := svgArc.ToArc()
		w.Arc(arc.Center, arc.Radii, arc.SweepAngle, arc.XRotation)
	}
}

// RelativeArcTo is SvgPathBuilder::relative_arc_to.
func (w *WithSvg[B]) RelativeArcTo(radii Vector, xRotation Angle, flags ArcFlags, to Vector) {
	t := w.relativeToAbsolute(to)
	w.ArcTo(radii, xRotation, flags, t)
}

// AddPolygon is SvgPathBuilder::add_polygon.
func (w *WithSvg[B]) AddPolygon(polygon Polygon[Point]) {
	if len(polygon.Points) == 0 {
		return
	}

	w.Reserve(len(polygon.Points), 0)

	w.MoveTo(polygon.Points[0])
	for _, p := range polygon.Points[1:] {
		w.LineTo(p)
	}

	if polygon.Closed {
		w.Close()
	}
}

var _ SvgPathBuilder = (*WithSvg[*BuilderImpl])(nil)

var _ PathBuilder = (*BuilderImpl)(nil)
