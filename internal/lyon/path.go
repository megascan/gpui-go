package lyon

import (
	"iter"

	"github.com/megascan/gpui-go/internal/lyon/geom"
)

type verb uint8

const (
	verbLineTo verb = iota
	verbQuadraticTo
	verbCubicTo
	verbBegin
	verbClose
	verbEnd
)

// Path is lyon_path's Path, a simple path data structure.
type Path struct {
	points        []Point
	verbs         []verb
	numAttributes int
}

// Builder is lyon_path's Builder, a path builder without custom attributes.
type Builder = NoAttributes[*BuilderImpl]

// NewBuilder is Path::builder.
func NewBuilder() *Builder {
	return NoAttributesWrap(NewBuilderImpl())
}

// PathSvgBuilder is Path::svg_builder.
func PathSvgBuilder() *WithSvg[*BuilderImpl] {
	return NewWithSvg(NewBuilderImpl())
}

// NewPath is Path::new.
func NewPath() Path { return Path{} }

// PathFromEvents is the FromIterator<PathEvent> impl for Path.
func PathFromEvents(events iter.Seq[PathEvent]) Path {
	builder := NewBuilder()
	for event := range events {
		builder.PathEvent(event)
	}
	return builder.Build()
}

// Iter is Path::iter.
func (p Path) Iter() iter.Seq[PathEvent] {
	return func(yield func(PathEvent) bool) {
		it := newPathIter(p.numAttributes, p.points, p.verbs)
		for {
			evt, ok := it.next()
			if !ok || !yield(evt) {
				return
			}
		}
	}
}

// Transformed is Path::transformed.
func (p Path) Transformed(transform geom.Transformation) Path {
	p.points = append([]Point(nil), p.points...)
	p.applyTransform(transform)
	return p
}

func (p *Path) applyTransform(transform geom.Transformation) {
	it := newIDIter(p.numAttributes, p.verbs)
	for {
		evt, ok := it.next()
		if !ok {
			return
		}
		switch evt.Kind {
		case EventBegin:
			p.points[evt.At.ToUsize()] = transform.TransformPoint(p.points[evt.At.ToUsize()])
		case EventLine:
			p.points[evt.To.ToUsize()] = transform.TransformPoint(p.points[evt.To.ToUsize()])
		case EventQuadratic:
			p.points[evt.Ctrl.ToUsize()] = transform.TransformPoint(p.points[evt.Ctrl.ToUsize()])
			p.points[evt.To.ToUsize()] = transform.TransformPoint(p.points[evt.To.ToUsize()])
		case EventCubic:
			p.points[evt.Ctrl1.ToUsize()] = transform.TransformPoint(p.points[evt.Ctrl1.ToUsize()])
			p.points[evt.Ctrl2.ToUsize()] = transform.TransformPoint(p.points[evt.Ctrl2.ToUsize()])
			p.points[evt.To.ToUsize()] = transform.TransformPoint(p.points[evt.To.ToUsize()])
		}
	}
}

// BuilderImpl is lyon_path's BuilderImpl, the builder behind Path::builder.
type BuilderImpl struct {
	points    []Point
	verbs     []verb
	first     Point
	validator DebugValidator
}

// NewBuilderImpl is BuilderImpl::new.
func NewBuilderImpl() *BuilderImpl {
	return &BuilderImpl{first: NewPoint(0, 0), validator: NewDebugValidator()}
}

// BuilderImplWithCapacity is BuilderImpl::with_capacity.
func BuilderImplWithCapacity(points, edges int) *BuilderImpl {
	return &BuilderImpl{
		points:    make([]Point, 0, points),
		verbs:     make([]verb, 0, edges),
		first:     NewPoint(0, 0),
		validator: NewDebugValidator(),
	}
}

// WithSvg is BuilderImpl::with_svg.
func (b *BuilderImpl) WithSvg() *WithSvg[*BuilderImpl] {
	if len(b.verbs) != 0 {
		panic("assertion failed: self.verbs.is_empty()")
	}
	return NewWithSvg(b)
}

// NumAttributes is PathBuilder::num_attributes.
func (b *BuilderImpl) NumAttributes() int { return 0 }

// Begin is PathBuilder::begin.
func (b *BuilderImpl) Begin(at Point, _ Attributes) EndpointID {
	b.validator.Begin()
	nanCheck(at)

	id := EndpointID(uint32(len(b.points)))

	b.first = at
	b.points = append(b.points, at)
	b.verbs = append(b.verbs, verbBegin)

	return id
}

// End is PathBuilder::end.
func (b *BuilderImpl) End(close bool) {
	b.validator.End()

	if close {
		b.points = append(b.points, b.first)
	}

	if close {
		b.verbs = append(b.verbs, verbClose)
	} else {
		b.verbs = append(b.verbs, verbEnd)
	}
}

// Close is PathBuilder::close.
func (b *BuilderImpl) Close() { b.End(true) }

// LineTo is PathBuilder::line_to.
func (b *BuilderImpl) LineTo(to Point, _ Attributes) EndpointID {
	b.validator.Edge()
	nanCheck(to)

	id := EndpointID(uint32(len(b.points)))
	b.points = append(b.points, to)
	b.verbs = append(b.verbs, verbLineTo)

	return id
}

// QuadraticBezierTo is PathBuilder::quadratic_bezier_to.
func (b *BuilderImpl) QuadraticBezierTo(ctrl, to Point, _ Attributes) EndpointID {
	b.validator.Edge()
	nanCheck(ctrl)
	nanCheck(to)

	b.points = append(b.points, ctrl)
	id := EndpointID(uint32(len(b.points)))
	b.points = append(b.points, to)
	b.verbs = append(b.verbs, verbQuadraticTo)

	return id
}

// CubicBezierTo is PathBuilder::cubic_bezier_to.
func (b *BuilderImpl) CubicBezierTo(ctrl1, ctrl2, to Point, _ Attributes) EndpointID {
	b.validator.Edge()
	nanCheck(ctrl1)
	nanCheck(ctrl2)
	nanCheck(to)

	b.points = append(b.points, ctrl1, ctrl2)
	id := EndpointID(uint32(len(b.points)))
	b.points = append(b.points, to)
	b.verbs = append(b.verbs, verbCubicTo)

	return id
}

// Reserve is PathBuilder::reserve.
func (b *BuilderImpl) Reserve(endpoints, ctrlPoints int) {
	b.points = growCap(b.points, endpoints+ctrlPoints)
	b.verbs = growCap(b.verbs, endpoints)
}

// Build is Build::build.
func (b *BuilderImpl) Build() Path {
	b.validator.Build()
	return Path{points: b.points, verbs: b.verbs, numAttributes: 0}
}

func growCap[T any](s []T, n int) []T {
	if cap(s)-len(s) >= n {
		return s
	}
	out := make([]T, len(s), len(s)+n)
	copy(out, s)
	return out
}

func nanCheck(Point) {}

type pathIter struct {
	points       pointIter
	verbs        []verb
	current      Point
	first        Point
	attribStride int
}

func newPathIter(numAttributes int, points []Point, verbs []verb) pathIter {
	return pathIter{
		points:       pointIter{slice: points},
		verbs:        verbs,
		current:      NewPoint(0, 0),
		first:        NewPoint(0, 0),
		attribStride: (numAttributes + 1) / 2,
	}
}

func (it *pathIter) skipAttributes() {
	it.points.advanceN(it.attribStride)
}

func (it *pathIter) next() (PathEvent, bool) {
	if len(it.verbs) == 0 {
		return PathEvent{}, false
	}
	v := it.verbs[0]
	it.verbs = it.verbs[1:]
	switch v {
	case verbBegin:
		it.current = it.points.next()
		it.skipAttributes()
		it.first = it.current
		return PathEvent{Kind: EventBegin, At: it.current}, true
	case verbLineTo:
		from := it.current
		it.current = it.points.next()
		it.skipAttributes()
		return PathEvent{Kind: EventLine, From: from, To: it.current}, true
	case verbQuadraticTo:
		from := it.current
		ctrl := it.points.next()
		it.current = it.points.next()
		it.skipAttributes()
		return PathEvent{Kind: EventQuadratic, From: from, Ctrl: ctrl, To: it.current}, true
	case verbCubicTo:
		from := it.current
		ctrl1 := it.points.next()
		ctrl2 := it.points.next()
		it.current = it.points.next()
		it.skipAttributes()
		return PathEvent{Kind: EventCubic, From: from, Ctrl1: ctrl1, Ctrl2: ctrl2, To: it.current}, true
	case verbClose:
		last := it.current
		_ = it.points.next()
		it.skipAttributes()
		return PathEvent{Kind: EventEnd, Last: last, First: it.first, Close: true}, true
	default:
		last := it.current
		it.current = it.first
		return PathEvent{Kind: EventEnd, Last: last, First: it.first, Close: false}, true
	}
}

type pointIter struct {
	slice []Point
}

func (it *pointIter) next() Point {
	if len(it.slice) == 0 {
		nan := float32Nan()
		return NewPoint(nan, nan)
	}
	p := it.slice[0]
	it.slice = it.slice[1:]
	return p
}

func (it *pointIter) advanceN(n int) {
	if len(it.slice) < n {
		panic("assertion failed: self.remaining_len() >= n")
	}
	it.slice = it.slice[n:]
}

type idIter struct {
	verbs          []verb
	current        uint32
	first          uint32
	evt            uint32
	endpointStride uint32
}

func newIDIter(numAttributes int, verbs []verb) idIter {
	return idIter{verbs: verbs, endpointStride: (uint32(numAttributes)+1)/2 + 1}
}

func (it *idIter) next() (IDEvent, bool) {
	if len(it.verbs) == 0 {
		return IDEvent{}, false
	}
	v := it.verbs[0]
	it.verbs = it.verbs[1:]
	switch v {
	case verbBegin:
		at := it.current
		it.first = at
		return IDEvent{Kind: EventBegin, At: EndpointID(at)}, true
	case verbLineTo:
		from := EndpointID(it.current)
		it.current += it.endpointStride
		to := EndpointID(it.current)
		it.evt++
		return IDEvent{Kind: EventLine, From: from, To: to}, true
	case verbQuadraticTo:
		from := EndpointID(it.current)
		base := it.current + it.endpointStride
		ctrl := ControlPointID(base)
		to := EndpointID(base + 1)
		it.current = base + 1
		it.evt++
		return IDEvent{Kind: EventQuadratic, From: from, Ctrl: ctrl, To: to}, true
	case verbCubicTo:
		from := EndpointID(it.current)
		base := it.current + it.endpointStride
		ctrl1 := ControlPointID(base)
		ctrl2 := ControlPointID(base + 1)
		to := EndpointID(base + 2)
		it.current = base + 2
		it.evt++
		return IDEvent{Kind: EventCubic, From: from, Ctrl1: ctrl1, Ctrl2: ctrl2, To: to}, true
	case verbClose:
		last := EndpointID(it.current)
		first := EndpointID(it.first)
		it.current += it.endpointStride * 2
		it.evt++
		return IDEvent{Kind: EventEnd, Last: last, First: first, Close: true}, true
	default:
		last := EndpointID(it.current)
		first := EndpointID(it.first)
		it.current += it.endpointStride
		it.evt++
		return IDEvent{Kind: EventEnd, Last: last, First: first, Close: false}, true
	}
}
