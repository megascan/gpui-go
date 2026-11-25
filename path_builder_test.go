package gpui

import (
	"slices"
	"testing"
)

func TestPathBuilder_DashArrayRepeatsOddLengths(t *testing.T) {
	b := PathBuilderStroke(Px(1.)).DashArray([]Pixels{Px(5.), Px(3.), Px(2.)})
	got, ok := b.dashArray.Get()
	want := []Pixels{Px(5.), Px(3.), Px(2.), Px(5.), Px(3.), Px(2.)}
	if !ok || !slices.Equal(got, want) {
		t.Fatalf("dash array = %v, %v; want %v", got, ok, want)
	}
}

func TestPathBuilder_Defaults(t *testing.T) {
	if got := DefaultStrokeOptions(); got != (StrokeOptions{LineWidth: 1.0, LineCap: LineCapButt, LineJoin: LineJoinMiter, MiterLimit: 4.0}) {
		t.Fatalf("DefaultStrokeOptions() = %+v", got)
	}
	style, ok := PathBuilderStroke(Px(3.)).Style.(PathStyleStroke)
	if !ok || style.LineWidth != 3.0 || style.MiterLimit != 4.0 {
		t.Fatalf("PathBuilderStroke style = %#v", style)
	}
	fill, ok := PathBuilderFill().Style.(PathStyleFill)
	if !ok || fill.FillRule != FillRuleEvenOdd {
		t.Fatalf("PathBuilderFill style = %#v", fill)
	}
}

func TestPathBuilder_StrokeIsNonZeroAndClosed(t *testing.T) {
	b := PathBuilderStroke(Px(2.))
	b.MoveTo(NewPoint(Px(0.), Px(0.)))
	b.LineTo(NewPoint(Px(10.), Px(0.)))
	path, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if path.evenOdd() {
		t.Fatal("strokes fill by the non-zero rule")
	}
	if len(path.curves) == 0 {
		t.Fatal("stroke produced no curves")
	}
}

func TestPathBuilder_TransformAppliesToFill(t *testing.T) {
	b := PathBuilderFill()
	b.Translate(NewPoint(Px(5.), Px(7.)))
	b.AddPolygon([]Point[Pixels]{NewPoint(Px(0.), Px(0.)), NewPoint(Px(1.), Px(0.)), NewPoint(Px(1.), Px(1.))}, true)
	path, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, curve := range path.curves {
		if curve.P0.X < 5 || curve.P0.Y < 7 {
			t.Fatalf("curve %+v is not translated", curve)
		}
	}
}

func TestPathBuilder_PointerStylesBuild(t *testing.T) {
	stroke := PathBuilderStroke(Px(2.))
	stroke.Style = &PathStyleStroke{DefaultStrokeOptions()}
	stroke.MoveTo(NewPoint(Px(0.), Px(0.)))
	stroke.LineTo(NewPoint(Px(10.), Px(0.)))
	path, err := stroke.Build()
	if err != nil || len(path.curves) == 0 || path.evenOdd() {
		t.Fatalf("pointer stroke: err = %v, curves = %d, evenOdd = %v", err, len(path.curves), path.evenOdd())
	}

	fill := PathBuilderFill()
	fill.Style = &PathStyleFill{FillOptions{FillRule: FillRuleEvenOdd}}
	fill.AddPolygon([]Point[Pixels]{NewPoint(Px(0.), Px(0.)), NewPoint(Px(1.), Px(0.)), NewPoint(Px(1.), Px(1.))}, true)
	path, err = fill.Build()
	if err != nil || len(path.curves) == 0 || !path.evenOdd() {
		t.Fatalf("pointer fill: err = %v, curves = %d, evenOdd = %v", err, len(path.curves), path.evenOdd())
	}
}

func TestPathBuilder_InvalidStyleReturnsError(t *testing.T) {
	for _, style := range []PathStyle{nil, (*PathStyleStroke)(nil), (*PathStyleFill)(nil)} {
		b := PathBuilderFill()
		b.Style = style
		b.AddPolygon([]Point[Pixels]{NewPoint(Px(0.), Px(0.)), NewPoint(Px(1.), Px(0.)), NewPoint(Px(1.), Px(1.))}, true)
		if _, err := b.Build(); err == nil {
			t.Fatalf("Build with style %#v returned no error", style)
		}
	}
}
