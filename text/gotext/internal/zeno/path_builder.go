package zeno

// PathBuilder is a sink for path commands.
type PathBuilder interface {
	CurrentPoint() Point
	MoveTo(to Point)
	LineTo(to Point)
	QuadTo(control, to Point)
	CurveTo(control1, control2, to Point)
	Close()
}

// TransformSink applies a transform to every point before forwarding it.
type TransformSink struct {
	Sink      PathBuilder
	Transform Transform
}

// CurrentPoint returns the current point of the inner sink.
func (s *TransformSink) CurrentPoint() Point { return s.Sink.CurrentPoint() }

// MoveTo forwards a transformed move.
func (s *TransformSink) MoveTo(to Point) { s.Sink.MoveTo(s.Transform.TransformPoint(to)) }

// LineTo forwards a transformed line.
func (s *TransformSink) LineTo(to Point) { s.Sink.LineTo(s.Transform.TransformPoint(to)) }

// QuadTo forwards a transformed quadratic curve.
func (s *TransformSink) QuadTo(control, to Point) {
	s.Sink.QuadTo(s.Transform.TransformPoint(control), s.Transform.TransformPoint(to))
}

// CurveTo forwards a transformed cubic curve.
func (s *TransformSink) CurveTo(control1, control2, to Point) {
	s.Sink.CurveTo(s.Transform.TransformPoint(control1), s.Transform.TransformPoint(control2), s.Transform.TransformPoint(to))
}

// Close forwards a close.
func (s *TransformSink) Close() { s.Sink.Close() }

// CurrentPoint returns the last added point.
func (b *BoundsBuilder) CurrentPoint() Point { return b.current }

// MoveTo adds a point.
func (b *BoundsBuilder) MoveTo(to Point) {
	b.add(to)
	b.current = to
}

// LineTo adds a point.
func (b *BoundsBuilder) LineTo(to Point) {
	b.add(to)
	b.current = to
}

// QuadTo adds the control and end points.
func (b *BoundsBuilder) QuadTo(control, to Point) {
	b.add(control)
	b.add(to)
	b.current = to
}

// CurveTo adds the control and end points.
func (b *BoundsBuilder) CurveTo(control1, control2, to Point) {
	b.add(control1)
	b.add(control2)
	b.add(to)
	b.current = to
}

// Close does nothing.
func (b *BoundsBuilder) Close() {}
