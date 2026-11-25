package a11y

import (
	"math"
	"testing"
)

func TestAffine_DefaultIsIdentity(t *testing.T) {
	if DefaultAffine() != AffineIdentity() {
		t.Fatal("DefaultAffine differs from AffineIdentity")
	}
	p := NewPoint(3, -4)
	if got := DefaultAffine().MulPoint(p); got != p {
		t.Fatalf("identity moved point: %v", got)
	}
	if got := AffineIdentity().AsCoeffs(); got != [6]float64{1, 0, 0, 1, 0, 0} {
		t.Fatalf("identity coeffs = %v", got)
	}
}

func TestAffine_MulPointAndTranslate(t *testing.T) {
	if got := AffineTranslate(NewVec2(3, 4)).MulPoint(NewPoint(1, 2)); got != NewPoint(4, 6) {
		t.Fatalf("translate = %v", got)
	}
	if got := AffineScale(2).Determinant(); got != 4 {
		t.Fatalf("determinant = %v", got)
	}
}

func TestAffine_MulComposesRightFirst(t *testing.T) {
	m := AffineScale(2).Mul(AffineTranslate(NewVec2(1, 1)))
	if got := m.MulPoint(NewPoint(0, 0)); got != NewPoint(2, 2) {
		t.Fatalf("composed origin = %v", got)
	}
	a := AffineScale(2)
	a.MulAssign(AffineTranslate(NewVec2(1, 1)))
	if a != m {
		t.Fatalf("MulAssign = %v, want %v", a, m)
	}
}

func TestAffine_InverseUndoesTransform(t *testing.T) {
	a := NewAffine([6]float64{2, 0, 0, 2, 6, 8})
	inv := a.Inverse()
	want := [6]float64{0.5, math.Copysign(0, -1), math.Copysign(0, -1), 0.5, -3, -4}
	if inv.AsCoeffs() != want {
		t.Fatalf("inverse = %v, want %v", inv.AsCoeffs(), want)
	}
	if got := inv.MulPoint(a.MulPoint(NewPoint(1, 2))); got != NewPoint(1, 2) {
		t.Fatalf("round trip = %v", got)
	}
}

func TestAffine_TransformRectBbox(t *testing.T) {
	rot := NewAffine([6]float64{0, 1, -1, 0, 0, 0})
	got := rot.TransformRectBbox(NewRect(0, 0, 2, 1))
	if got != NewRect(-1, 0, 0, 2) {
		t.Fatalf("bbox = %v", got)
	}
}

func TestAffine_FiniteAndNaN(t *testing.T) {
	ok := AffineScale(1)
	if !ok.IsFinite() || ok.IsNan() {
		t.Fatal("identity should be finite and not NaN")
	}
	bad := NewAffine([6]float64{math.NaN(), 0, 0, 1, 0, 0})
	if bad.IsFinite() || !bad.IsNan() {
		t.Fatal("NaN coefficient not detected")
	}
	inf := NewAffine([6]float64{1, 0, 0, math.Inf(1), 0, 0})
	if inf.IsFinite() || inf.IsNan() {
		t.Fatal("infinite coefficient should be non-finite but not NaN")
	}
}

func TestAffine_Float64MulAffine(t *testing.T) {
	got := Float64MulAffine(2, NewAffine([6]float64{1, 2, 3, 4, 5, 6}))
	if got.AsCoeffs() != [6]float64{2, 4, 6, 8, 10, 12} {
		t.Fatalf("scaled = %v", got.AsCoeffs())
	}
}

func TestRect_AbsAndFromPoints(t *testing.T) {
	want := NewRect(0, 0, 10, 10)
	if got := RectFromPoints(NewPoint(10, 10), NewPoint(0, 0)); got != want {
		t.Fatalf("from points = %v", got)
	}
	if got := RectFromOriginSize(NewPoint(5, 5), NewSize(2, 3)); got != NewRect(5, 5, 7, 8) {
		t.Fatalf("from origin size = %v", got)
	}
}

func TestRect_UnionIntersectNaN(t *testing.T) {
	nan := math.NaN()
	if got := NewRect(nan, 0, 1, 1).Union(NewRect(0, 0, 2, 2)); got.X0 != 0 {
		t.Fatalf("union with NaN min = %v", got)
	}
	if got := NewRect(nan, 0, 1, 1).MinX(); got != 1 {
		t.Fatalf("MinX with NaN = %v", got)
	}
	if got := NewRect(0, 0, 1, 1).Intersect(NewRect(2, 2, 3, 3)); got != NewRect(2, 2, 2, 2) {
		t.Fatalf("disjoint intersect = %v", got)
	}
}

func TestRect_Queries(t *testing.T) {
	r := NewRect(1, 2, 4, 6)
	if r.Width() != 3 || r.Height() != 4 || r.Area() != 12 {
		t.Fatalf("size queries wrong: %v %v %v", r.Width(), r.Height(), r.Area())
	}
	if !r.Contains(NewPoint(1, 2)) || r.Contains(NewPoint(4, 2)) {
		t.Fatal("Contains must include min edge and exclude max edge")
	}
	if !NewRect(1, 1, 1, 5).IsEmpty() || r.IsEmpty() {
		t.Fatal("IsEmpty wrong")
	}
	if got := r.UnionPt(NewPoint(0, 9)); got != NewRect(0, 2, 4, 9) {
		t.Fatalf("UnionPt = %v", got)
	}
	if got := r.WithSize(NewSize(1, 1)); got != NewRect(1, 2, 2, 3) {
		t.Fatalf("WithSize = %v", got)
	}
	if got := r.WithOrigin(NewPoint(0, 0)); got != NewRect(0, 0, 3, 4) {
		t.Fatalf("WithOrigin = %v", got)
	}
	if got := r.Add(NewVec2(1, 1)).Sub(NewVec2(1, 1)); got != r {
		t.Fatalf("Add/Sub round trip = %v", got)
	}
}

func TestVec2_DivUsesReciprocal(t *testing.T) {
	v := NewVec2(1, 2)
	if got, want := v.Div(3), v.Mul(1/3.0); got != want {
		t.Fatalf("Div = %v, want reciprocal product %v", got, want)
	}
	w := v
	w.DivAssign(3)
	if w != v.Div(3) {
		t.Fatalf("DivAssign = %v", w)
	}
	if got := v.Neg(); got != NewVec2(-1, -2) {
		t.Fatalf("Neg = %v", got)
	}
}

func TestSize_Arithmetic(t *testing.T) {
	s := NewSize(2, 4)
	if got := Float64MulSize(3, s); got != NewSize(6, 12) {
		t.Fatalf("Float64MulSize = %v", got)
	}
	s.DivAssign(2)
	if s != NewSize(1, 2) {
		t.Fatalf("DivAssign = %v", s)
	}
	s.AddAssign(NewSize(1, 1))
	s.SubAssign(NewSize(0, 1))
	if s != NewSize(2, 2) {
		t.Fatalf("Add/Sub = %v", s)
	}
}

func TestPoint_Operators(t *testing.T) {
	p := NewPoint(1, 1)
	p.AddAssign(NewVec2(2, 3))
	p.SubAssign(NewVec2(1, 1))
	if p != NewPoint(2, 3) {
		t.Fatalf("AddAssign/SubAssign = %v", p)
	}
	if got := p.SubPoint(NewPoint(1, 1)); got != NewVec2(1, 2) {
		t.Fatalf("SubPoint = %v", got)
	}
}

func TestDebugStrings(t *testing.T) {
	cases := []struct {
		got  string
		want string
	}{
		{debugF64(1), "1.0"},
		{debugF64(0.5), "0.5"},
		{debugF64(123456), "123456.0"},
		{debugF64(1e16), "1e16"},
		{debugF64(1.5e-5), "1.5e-5"},
		{debugF64(math.Copysign(0, -1)), "-0.0"},
		{debugF64(math.NaN()), "NaN"},
		{debugF64(math.Inf(-1)), "-inf"},
		{NewPoint(1, -2.5).String(), "(1.0, -2.5)"},
		{NewSize(1, 2).String(), "1.0W×2.0H"},
		{NewRect(0, 0, 1, 1).String(), "Rect { x0: 0.0, y0: 0.0, x1: 1.0, y1: 1.0 }"},
		{NewVec2(1, -2.5).String(), "Vec2 { x: 1.0, y: -2.5 }"},
		{AffineIdentity().String(), "Affine([1.0, 0.0, 0.0, 1.0, 0.0, 0.0])"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
