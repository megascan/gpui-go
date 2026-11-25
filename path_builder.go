package gpui

import (
	"errors"

	"github.com/megascan/gpui-go/internal/lyon"
	"github.com/megascan/gpui-go/internal/lyon/geom"
	skiapath "github.com/xo/okage/path"
)

// Transform is lyon's 2D transform, a row-major 3x2 matrix.
type Transform = lyon.Transform

// FillRule is how overlapping contours combine into the filled region.
type FillRule = lyon.FillRule

// LineCap is the shape at the ends of open subpaths and dashes.
type LineCap = lyon.LineCap

// LineJoin is the shape of corners.
type LineJoin = lyon.LineJoin

const (
	// FillRuleEvenOdd fills where the winding number is odd.
	FillRuleEvenOdd = lyon.FillRuleEvenOdd
	// FillRuleNonZero fills where the winding number is not zero.
	FillRuleNonZero = lyon.FillRuleNonZero

	// LineCapButt ends the stroke at the endpoint.
	LineCapButt = lyon.LineCapButt
	// LineCapSquare extends the stroke by half its width.
	LineCapSquare = lyon.LineCapSquare
	// LineCapRound ends the stroke with a half circle.
	LineCapRound = lyon.LineCapRound

	// LineJoinMiter extends to the miter limit, then bevels.
	LineJoinMiter = lyon.LineJoinMiter
	// LineJoinMiterClip extends to the miter limit, then clips the corner.
	LineJoinMiterClip = lyon.LineJoinMiterClip
	// LineJoinRound joins with a circular arc.
	LineJoinRound = lyon.LineJoinRound
	// LineJoinBevel connects the outside edges.
	LineJoinBevel = lyon.LineJoinBevel
)

// PathStyle is the style of a PathBuilder: PathStyleStroke or PathStyleFill.
type PathStyle interface {
	isPathStyle()
}

// PathStyleStroke strokes the path.
type PathStyleStroke struct{ StrokeOptions }

// PathStyleFill fills the path.
type PathStyleFill struct{ FillOptions }

func (PathStyleStroke) isPathStyle() {}
func (PathStyleFill) isPathStyle()   {}

// StrokeOptions is how to stroke a path.
type StrokeOptions struct {
	// LineWidth is the width of the stroke.
	LineWidth float32
	// LineCap is the shape of the ends of open subpaths and dashes.
	LineCap LineCap
	// LineJoin is the shape of corners.
	LineJoin LineJoin
	// MiterLimit is how long a miter join may get, as a multiple of the line width, before it is beveled.
	MiterLimit float32
}

// DefaultStrokeOptions returns a 1px butt-capped, miter-joined stroke with a miter limit of 4.
func DefaultStrokeOptions() StrokeOptions {
	return StrokeOptions{
		LineWidth:  1.0,
		LineCap:    LineCapButt,
		LineJoin:   LineJoinMiter,
		MiterLimit: 4.0,
	}
}

// FillOptions is how to fill a path.
type FillOptions struct {
	// FillRule is how overlapping contours combine into the filled region.
	FillRule FillRule
}

// DefaultFillOptions returns the even-odd fill.
func DefaultFillOptions() FillOptions {
	return FillOptions{FillRule: FillRuleEvenOdd}
}

// PathBuilder builds a Path.
type PathBuilder struct {
	raw       *lyon.WithSvg[*lyon.BuilderImpl]
	transform Option[Transform]
	// Style is the style of the PathBuilder.
	Style     PathStyle
	dashArray Option[[]Pixels]
}

func newPathBuilderFromBuilder(builder *lyon.Builder) *PathBuilder {
	b := DefaultPathBuilder()
	b.raw = builder.WithSvg()
	return b
}

func newPathBuilderFromSvg(raw *lyon.WithSvg[*lyon.BuilderImpl]) *PathBuilder {
	b := DefaultPathBuilder()
	b.raw = raw
	return b
}

func pointFromLyon(p lyon.Point) Point[Pixels] {
	return NewPoint(Px(p.X), Px(p.Y))
}

func lyonPoint(p Point[Pixels]) lyon.Point {
	return lyon.NewPoint(float32(p.X), float32(p.Y))
}

func lyonVector(p Point[Pixels]) lyon.Vector {
	return lyon.NewVector(float32(p.X), float32(p.Y))
}

// DefaultPathBuilder returns an empty builder for an even-odd fill.
func DefaultPathBuilder() *PathBuilder {
	return &PathBuilder{
		raw:       lyon.PathSvgBuilder(),
		Style:     PathStyleFill{DefaultFillOptions()},
		transform: None[Transform](),
		dashArray: None[[]Pixels](),
	}
}

// PathBuilderStroke returns a builder for a stroke of the given width.
func PathBuilderStroke(width Pixels) *PathBuilder {
	b := DefaultPathBuilder()
	options := DefaultStrokeOptions()
	options.LineWidth = float32(width)
	b.Style = PathStyleStroke{options}
	return b
}

// PathBuilderFill returns a builder for a fill.
func PathBuilderFill() *PathBuilder {
	return DefaultPathBuilder()
}

// WithStyle sets the style of the builder.
func (b *PathBuilder) WithStyle(style PathStyle) *PathBuilder {
	b.Style = style
	return b
}

// DashArray sets the dash array of the builder; fills ignore it. An odd number of
// values is repeated to yield an even number, so 5,3,2 is 5,3,2,5,3,2.
func (b *PathBuilder) DashArray(dashArray []Pixels) *PathBuilder {
	var array []Pixels
	if len(dashArray)%2 == 1 {
		array = make([]Pixels, 0, 2*len(dashArray))
		array = append(array, dashArray...)
		array = append(array, dashArray...)
	} else {
		array = append(make([]Pixels, 0, len(dashArray)), dashArray...)
	}
	b.dashArray = Some(array)
	return b
}

// MoveTo moves the current point to the given point.
func (b *PathBuilder) MoveTo(to Point[Pixels]) {
	b.raw.MoveTo(lyonPoint(to))
}

// LineTo draws a straight line from the current point to the given point.
func (b *PathBuilder) LineTo(to Point[Pixels]) {
	b.raw.LineTo(lyonPoint(to))
}

// CurveTo draws a quadratic curve from the current point to the given point.
func (b *PathBuilder) CurveTo(to Point[Pixels], ctrl Point[Pixels]) {
	b.raw.QuadraticBezierTo(lyonPoint(ctrl), lyonPoint(to))
}

// CubicBezierTo adds a cubic Bézier given its two control points and its end point.
func (b *PathBuilder) CubicBezierTo(to Point[Pixels], controlA Point[Pixels], controlB Point[Pixels]) {
	b.raw.CubicBezierTo(lyonPoint(controlA), lyonPoint(controlB), lyonPoint(to))
}

// ArcTo adds an elliptical arc.
func (b *PathBuilder) ArcTo(radii Point[Pixels], xRotation Pixels, largeArc bool, sweep bool, to Point[Pixels]) {
	b.raw.ArcTo(
		lyonVector(radii),
		geom.AngleDegrees(float32(xRotation)),
		lyon.ArcFlags{LargeArc: largeArc, Sweep: sweep},
		lyonPoint(to),
	)
}

// RelativeArcTo is ArcTo in relative coordinates.
func (b *PathBuilder) RelativeArcTo(radii Point[Pixels], xRotation Pixels, largeArc bool, sweep bool, to Point[Pixels]) {
	b.raw.RelativeArcTo(
		lyonVector(radii),
		geom.AngleDegrees(float32(xRotation)),
		lyon.ArcFlags{LargeArc: largeArc, Sweep: sweep},
		lyonVector(to),
	)
}

// AddPolygon adds a polygon.
func (b *PathBuilder) AddPolygon(points []Point[Pixels], closed bool) {
	converted := make([]lyon.Point, len(points))
	for i, p := range points {
		converted[i] = lyonPoint(p)
	}
	b.raw.AddPolygon(lyon.Polygon[lyon.Point]{Points: converted, Closed: closed})
}

// Close closes the current sub-path.
func (b *PathBuilder) Close() {
	b.raw.Close()
}

// Transform applies a transform to the path.
func (b *PathBuilder) Transform(transform Transform) {
	b.transform = Some(transform)
}

// Translate applies a translation to the path.
func (b *PathBuilder) Translate(to Point[Pixels]) {
	if transform, ok := b.transform.Get(); ok {
		b.transform = Some(transform.ThenTranslate(lyon.NewVector(float32(to.X), float32(to.Y))))
	} else {
		b.transform = Some(geom.TransformTranslation(float32(to.X), float32(to.Y)))
	}
}

// Scale applies a uniform scale to the path.
func (b *PathBuilder) Scale(scale float32) {
	if transform, ok := b.transform.Get(); ok {
		b.transform = Some(transform.ThenScale(scale, scale))
	} else {
		b.transform = Some(geom.TransformScale(scale, scale))
	}
}

// Rotate applies a rotation to the path; angle is in degrees, from 0 to 360.
func (b *PathBuilder) Rotate(angle float32) {
	radians := geom.ToRadians(angle)
	if transform, ok := b.transform.Get(); ok {
		b.transform = Some(transform.ThenRotate(geom.AngleRadians(radians)))
	} else {
		b.transform = Some(geom.TransformRotation(geom.AngleRadians(radians)))
	}
}

// Build builds the path.
func (b *PathBuilder) Build() (Path, error) {
	curves, fillRule, err := b.outline()
	if err != nil {
		return Path{}, err
	}
	return pathFromCurves(curves, fillRule), nil
}

func (b *PathBuilder) outline() ([]PathCurve, FillRule, error) {
	path := b.raw.Build()
	if transform, ok := b.transform.Get(); ok {
		path = path.Transformed(transform)
	}

	style := b.Style
	switch s := style.(type) {
	case *PathStyleStroke:
		if s != nil {
			style = *s
		}
	case *PathStyleFill:
		if s != nil {
			style = *s
		}
	}

	switch style := style.(type) {
	case PathStyleStroke:
		dashArray, hasDash := b.dashArray.Get()
		curves, err := strokeOutline(path, style.StrokeOptions, dashArray, hasDash)
		return curves, FillRuleNonZero, err
	case PathStyleFill:
		curves, err := fillOutline(path)
		return curves, style.FillRule, err
	}
	return nil, FillRuleNonZero, errPathStyle
}

var errPathStyle = errors.New("gpui: PathBuilder.Style must be a PathStyleStroke or PathStyleFill")

func strokeOutline(path lyon.Path, options StrokeOptions, dashArray []Pixels, hasDash bool) ([]PathCurve, error) {
	skia, ok := skiaPath(path)
	if ok && hasDash {
		array := make([]float32, len(dashArray))
		for i, length := range dashArray {
			array[i] = float32(length)
		}
		if dash, valid := skiapath.NewStrokeDash(array, 0.0); valid {
			if dashed, ok := skia.Dash(dash, strokeResolutionScale); ok {
				skia = dashed
			}
		}
	}
	var stroked *skiapath.Path
	if ok {
		stroke := skiapath.Stroke{
			Width:      options.LineWidth,
			MiterLimit: options.MiterLimit,
			Dash:       nil,
		}
		switch options.LineCap {
		case LineCapButt:
			stroke.LineCap = skiapath.LineCapButt
		case LineCapSquare:
			stroke.LineCap = skiapath.LineCapSquare
		case LineCapRound:
			stroke.LineCap = skiapath.LineCapRound
		}
		switch options.LineJoin {
		case LineJoinMiter:
			stroke.LineJoin = skiapath.LineJoinMiter
		case LineJoinMiterClip:
			stroke.LineJoin = skiapath.LineJoinMiterClip
		case LineJoinRound:
			stroke.LineJoin = skiapath.LineJoinRound
		case LineJoinBevel:
			stroke.LineJoin = skiapath.LineJoinBevel
		}
		stroked, ok = skia.Stroke(&stroke, strokeResolutionScale)
	}
	if !ok {
		return nil, nil
	}

	var output outline
	at := func(p skiapath.Point) Point[float32] { return NewPoint(p.X, p.Y) }
	segments := stroked.Segments()
	for {
		segment, ok := segments.Next()
		if !ok {
			break
		}
		switch segment.Kind {
		case skiapath.PathSegmentMoveTo:
			output.moveTo(at(segment.Points[0]))
		case skiapath.PathSegmentLineTo:
			output.lineTo(at(segment.Points[0]))
		case skiapath.PathSegmentQuadTo:
			output.curveTo(at(segment.Points[1]), at(segment.Points[0]))
		case skiapath.PathSegmentCubicTo:
			output.cubicTo(at(segment.Points[2]), at(segment.Points[0]), at(segment.Points[1]))
		case skiapath.PathSegmentClose:
			output.close()
		}
	}
	return output.intoCurves()
}

func fillOutline(path lyon.Path) ([]PathCurve, error) {
	var output outline
	at := func(p lyon.Point) Point[float32] { return NewPoint(p.X, p.Y) }
	for event := range path.Iter() {
		switch event.Kind {
		case lyon.EventBegin:
			output.moveTo(at(event.At))
		case lyon.EventLine:
			output.lineTo(at(event.To))
		case lyon.EventQuadratic:
			output.curveTo(at(event.To), at(event.Ctrl))
		case lyon.EventCubic:
			output.cubicTo(at(event.To), at(event.Ctrl1), at(event.Ctrl2))
		case lyon.EventEnd:
			output.close()
		}
	}
	return output.intoCurves()
}

// Path is a filled path, ready to paint; build one with PathBuilder.
type Path struct {
	curves      []PathCurve
	evenOddFill bool
}

func pathFromCurves(curves []PathCurve, fillRule FillRule) Path {
	return Path{
		curves:      curves,
		evenOddFill: fillRule == FillRuleEvenOdd,
	}
}

func (p Path) evenOdd() bool {
	return p.evenOddFill
}

func (p Path) bin(scale float32) *pathTiles {
	return bin(p.curves, p.evenOddFill, scale)
}

type outline struct {
	curves    []PathCurve
	start     Point[float32]
	current   Point[float32]
	nonFinite bool
}

func (o *outline) accepts(points ...Point[float32]) bool {
	allFinite := true
	for _, point := range points {
		if !(isFiniteF32(point.X) && isFiniteF32(point.Y)) {
			allFinite = false
			break
		}
	}
	o.nonFinite = o.nonFinite || !allFinite
	return !o.nonFinite
}

func (o *outline) moveTo(to Point[float32]) {
	if !o.accepts(to) {
		return
	}
	o.close()
	o.start = to
	o.current = to
}

func (o *outline) lineTo(to Point[float32]) {
	if o.accepts(to) && to != o.current {
		o.curves = append(o.curves, pathCurveLine(o.current, to))
		o.current = to
	}
}

func (o *outline) curveTo(to Point[float32], ctrl Point[float32]) {
	if !o.accepts(to, ctrl) || (to == o.current && ctrl == o.current) {
		return
	}
	o.curves = pathCurvePushMonotone(o.curves, o.current, ctrl, to)
	o.current = to
}

func (o *outline) cubicTo(to Point[float32], ctrl1 Point[float32], ctrl2 Point[float32]) {
	const tolerance float32 = 0.25
	if !o.accepts(to, ctrl1, ctrl2) {
		return
	}
	lyonPoint := func(p Point[float32]) lyon.Point { return lyon.NewPoint(p.X, p.Y) }
	cubic := geom.CubicBezierSegment{
		From:  lyonPoint(o.current),
		Ctrl1: lyonPoint(ctrl1),
		Ctrl2: lyonPoint(ctrl2),
		To:    lyonPoint(to),
	}
	cubic.ForEachQuadraticBezier(tolerance, func(quadratic *geom.QuadraticBezierSegment) {
		o.curveTo(NewPoint(quadratic.To.X, quadratic.To.Y), NewPoint(quadratic.Ctrl.X, quadratic.Ctrl.Y))
	})
}

func (o *outline) close() {
	start := o.start
	o.lineTo(start)
}

func (o *outline) intoCurves() ([]PathCurve, error) {
	if o.nonFinite {
		return nil, errors.New("path has a non-finite point")
	}
	o.close()
	return o.curves, nil
}

func newPathCurve(p0 Point[float32], ctrl Point[float32], p1 Point[float32], sign float32) PathCurve {
	return PathCurve{
		P0: p0,
		P1: p1,
		A: NewPoint(
			float32(p0.X-float32(2.0*ctrl.X))+p1.X,
			float32(p0.Y-float32(2.0*ctrl.Y))+p1.Y,
		),
		B:    NewPoint(2.0*(ctrl.X-p0.X), 2.0*(ctrl.Y-p0.Y)),
		Sign: sign,
		Pad:  0,
	}
}

func pathCurveLine(from Point[float32], to Point[float32]) PathCurve {
	p0, p1, sign := downward(from, to)
	return PathCurve{
		P0:   p0,
		P1:   p1,
		A:    NewPoint[float32](0.0, 0.0),
		B:    NewPoint(p1.X-p0.X, p1.Y-p0.Y),
		Sign: sign,
		Pad:  0,
	}
}

func pathCurvePushMonotone(curves []PathCurve, p0 Point[float32], ctrl Point[float32], p1 Point[float32]) []PathCurve {
	curve := geom.QuadraticBezierSegment{
		From: lyon.NewPoint(p0.X, p0.Y),
		Ctrl: lyon.NewPoint(ctrl.X, ctrl.Y),
		To:   lyon.NewPoint(p1.X, p1.Y),
	}
	curve.ForEachMonotonic(func(monotone *geom.QuadraticBezierSegment) {
		from := NewPoint(monotone.From.X, monotone.From.Y)
		to := NewPoint(monotone.To.X, monotone.To.Y)
		if from == to {
			return
		}
		p0, p1, sign := downward(from, to)
		ctrl := NewPoint(
			clampF32(monotone.Ctrl.X, minF32(p0.X, p1.X), maxF32(p0.X, p1.X)),
			clampF32(monotone.Ctrl.Y, p0.Y, p1.Y),
		)
		curves = append(curves, newPathCurve(p0, ctrl, p1, sign))
	})
	return curves
}

func downward(from Point[float32], to Point[float32]) (Point[float32], Point[float32], float32) {
	if from.Y > to.Y {
		return to, from, -1.0
	}
	return from, to, 1.0
}

const strokeResolutionScale float32 = 2.0

func skiaPath(path lyon.Path) (*skiapath.Path, bool) {
	builder := skiapath.NewPathBuilder()
	for event := range path.Iter() {
		switch event.Kind {
		case lyon.EventBegin:
			builder.MoveTo(event.At.X, event.At.Y)
		case lyon.EventLine:
			builder.LineTo(event.To.X, event.To.Y)
		case lyon.EventQuadratic:
			builder.QuadTo(event.Ctrl.X, event.Ctrl.Y, event.To.X, event.To.Y)
		case lyon.EventCubic:
			builder.CubicTo(event.Ctrl1.X, event.Ctrl1.Y, event.Ctrl2.X, event.Ctrl2.Y, event.To.X, event.To.Y)
		case lyon.EventEnd:
			if event.Close {
				builder.Close()
			}
		}
	}
	return builder.Finish()
}
