package sample

import (
	"encoding/json/v2"
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/refineable"
)

var _ refineable.Refineable[Style, StyleRefinement] = (*Style)(nil)
var _ refineable.Refineable[StyleRefinement, StyleRefinement] = (*StyleRefinement)(nil)
var _ refineable.IsEmpty = StyleRefinement{}

func TestSample_RefineAndRefined(t *testing.T) {
	s := Style{Visible: true, BorderWidth: 1, Tags: []string{"a"}}
	var r StyleRefinement
	if !r.IsEmpty() || r.IsSome() {
		t.Fatal("zero refinement not empty")
	}
	r.Origin.X = gpuiutil.Some[float32](3)
	r.AspectRatio = gpuiutil.Some[float32](0)
	r.Visible = gpuiutil.Some(false)
	s.Refine(&r)
	if s.Visible || s.Origin.X != 3 || s.AspectRatio != gpuiutil.Some[float32](0) || s.BorderWidth != 1 {
		t.Fatalf("refine: %+v", s)
	}
	if !s.IsSupersetOf(&r) {
		t.Fatal("refined style is not a superset of its refinement")
	}
	r2 := StyleRefinement{BorderWidth: gpuiutil.Some(5)}
	if s.IsSupersetOf(&r2) {
		t.Fatal("unexpected superset")
	}
	s2 := s.Refined(r2)
	if s2.BorderWidth != 5 || s.BorderWidth != 1 {
		t.Fatal("refined")
	}
}

func TestSample_Subtract(t *testing.T) {
	s := Style{BorderWidth: 2, Origin: Point[float32]{X: 1, Y: 2}}
	r := StyleRefinement{BorderWidth: gpuiutil.Some(2), Origin: PointRefinement[float32]{X: gpuiutil.Some[float32](1)}}
	d := s.Subtract(&r)
	if d.BorderWidth.IsSome() || d.Origin.X.IsSome() || d.Origin.Y != gpuiutil.Some[float32](2) || d.Visible != gpuiutil.Some(false) {
		t.Fatalf("subtract: %v", d)
	}
	rr := r.Subtract(&r)
	if !rr.IsEmpty() {
		t.Fatalf("r - r = %v", rr)
	}
}

func TestSample_FromRefinementAndCascade(t *testing.T) {
	var c refineable.Cascade[Style, StyleRefinement]
	c.Base().BorderWidth = gpuiutil.Some(1)
	slot := c.Reserve()
	c.Set(slot, gpuiutil.Some(StyleRefinement{BorderWidth: gpuiutil.Some(4), Margin: EdgesRefinement[int]{Top: gpuiutil.Some(7)}}))
	other := c.Reserve()
	c.Set(other, gpuiutil.None[StyleRefinement]())
	s := refineable.FromCascade(&c)
	if s.BorderWidth != 4 || s.Margin.Top != 7 {
		t.Fatalf("from cascade: %+v", s)
	}
	if got := StyleFromRefinement(c.Merged()); got.BorderWidth != 4 || got.Margin.Top != 7 {
		t.Fatalf("from refinement: %+v", got)
	}
}

func TestSample_DebugEqJSON(t *testing.T) {
	r := PointRefinement[int]{X: gpuiutil.Some(1)}
	if got := r.String(); got != "PointRefinement { X: Some(1), .. }" {
		t.Fatalf("debug = %q", got)
	}
	if got := (PointRefinement[int]{}).String(); got != "PointRefinement { .. }" {
		t.Fatalf("debug = %q", got)
	}
	if !r.Eq(PointRefinement[int]{X: gpuiutil.Some(1)}) || r.Eq(PointRefinement[int]{}) {
		t.Fatal("eq")
	}
	b, err := json.Marshal(StyleRefinement{BorderWidth: gpuiutil.Some(3), Origin: PointRefinement[float32]{X: gpuiutil.Some[float32](1)}})
	if err != nil || string(b) != `{"origin":{"x":1},"border_width":3}` {
		t.Fatalf("json = %s, %v", b, err)
	}
	var back StyleRefinement
	if err := json.Unmarshal(b, &back); err != nil || back.BorderWidth != gpuiutil.Some(3) || back.Origin.X != gpuiutil.Some[float32](1) {
		t.Fatalf("unmarshal: %v %v", back, err)
	}
}
