package taffy

import (
	"fmt"
	"strings"
	"testing"
)

func TestGeometry_Axes(t *testing.T) {
	if AbsoluteAxisHorizontal.OtherAxis() != AbsoluteAxisVertical || AbsoluteAxisVertical.OtherAxis() != AbsoluteAxisHorizontal {
		t.Error("OtherAxis")
	}
	if AbstractAxisInline.Other() != AbstractAxisBlock || AbstractAxisBlock.AsAbsNaive() != AbsoluteAxisVertical {
		t.Error("AbstractAxis")
	}
	s := Size[int]{Width: 1, Height: 2}
	if s.GetAbs(AbsoluteAxisVertical) != 2 || s.Transpose() != (Size[int]{Width: 2, Height: 1}) {
		t.Error("Size axis helpers")
	}
	if s.get(AbstractAxisInline) != 1 || s.with(AbstractAxisBlock, 5).Height != 5 {
		t.Error("Size abstract axis helpers")
	}
	s.set(AbstractAxisInline, 9)
	if s.Width != 9 {
		t.Error("Size.set")
	}
	b := inBothAbsAxis[string]{horizontal: "h", vertical: "v"}
	if b.get(AbsoluteAxisVertical) != "v" {
		t.Error("inBothAbsAxis.get")
	}
	p := Point[float32]{X: 1, Y: 2}
	p.Set(AbstractAxisBlock, 4)
	if p.Get(AbstractAxisBlock) != 4 || p.Transpose() != (Point[float32]{X: 4, Y: 1}) {
		t.Error("Point axis helpers")
	}
	if SizeFromPoint(p) != (Size[float32]{Width: 1, Height: 4}) {
		t.Error("SizeFromPoint")
	}
}

func TestGeometry_RectArithmetic(t *testing.T) {
	a := NewRectF32(1, 2, 3, 4)
	b := Rect[float32]{Left: 10, Right: 20, Top: 30, Bottom: 40}
	if got := a.Add(b); got != NewRectF32(11, 22, 33, 44) {
		t.Errorf("Add = %v", got)
	}
	if a.HorizontalAxisSum() != 3 || a.VerticalAxisSum() != 7 || a.SumAxes() != (Size[float32]{Width: 3, Height: 7}) {
		t.Error("axis sums")
	}
	if a.GridAxisSum(AbsoluteAxisVertical) != 7 {
		t.Error("GridAxisSum")
	}
	if a.Transpose() != NewRectF32(3, 4, 1, 2) {
		t.Error("Transpose")
	}
	if a.HorizontalComponents() != (Line[float32]{Start: 1, End: 2}) || a.VerticalComponents() != (Line[float32]{Start: 3, End: 4}) {
		t.Error("components")
	}
	if got := a.Map(func(v float32) int { return int(v) * 2 }); got != (Rect[int]{Left: 2, Right: 4, Top: 6, Bottom: 8}) {
		t.Errorf("Map = %v", got)
	}
	if got := a.zipSize(Size[float32]{Width: 100, Height: 1000}, func(v, s float32) float32 { return v + s }); got != NewRectF32(101, 102, 1003, 1004) {
		t.Errorf("zipSize = %v", got)
	}
	if got := NewRectF32(0, 5, 0, 5).Union(NewRectF32(-1, 3, 2, 8)); got != NewRectF32(-1, 5, 0, 8) {
		t.Errorf("Union = %v", got)
	}
	if (Line[float32]{Start: 2, End: 3}).Sum() != 5 || (Line[uint16]{Start: 2, End: 3}).Sum() != 5 {
		t.Error("Line.Sum")
	}
	if LineBoolTrue != (Line[bool]{Start: true, End: true}) || LineBoolFalse.Start {
		t.Error("Line bool constants")
	}
	if RectF32Zero != (Rect[float32]{}) || RectOptionF32None.Left.IsSome() {
		t.Error("Rect constants")
	}
}

func TestGeometry_SizeArithmetic(t *testing.T) {
	a := Size[float32]{Width: 1, Height: 5}
	b := Size[float32]{Width: 3, Height: 2}
	if a.Add(b) != (Size[float32]{Width: 4, Height: 7}) || a.Sub(b) != (Size[float32]{Width: -2, Height: 3}) {
		t.Error("Add/Sub")
	}
	if a.F32Max(b) != (Size[float32]{Width: 3, Height: 5}) || a.F32Min(b) != (Size[float32]{Width: 1, Height: 2}) {
		t.Error("F32Max/F32Min")
	}
	if !a.HasNonZeroArea() || (Size[float32]{Width: 1}).HasNonZeroArea() || SizeF32Zero.HasNonZeroArea() {
		t.Error("HasNonZeroArea")
	}
	if a.MapWidth(func(v float32) float32 { return v * 10 }) != (Size[float32]{Width: 10, Height: 5}) {
		t.Error("MapWidth")
	}
	if a.MapHeight(func(v float32) float32 { return v * 10 }) != (Size[float32]{Width: 1, Height: 50}) {
		t.Error("MapHeight")
	}
	if got := a.ZipMap(Size[bool]{Width: true}, func(v float32, ok bool) string { return fmt.Sprint(v, ok) }); got != (Size[string]{Width: "1 true", Height: "5 false"}) {
		t.Errorf("ZipMap = %v", got)
	}
	if (Point[float32]{X: 1, Y: 2}).Add(Point[float32]{X: 3, Y: 4}) != (Point[float32]{X: 4, Y: 6}) || PointF32Zero != (Point[float32]{}) || PointOptionF32None.X.IsSome() {
		t.Error("Point")
	}
}

func TestGeometry_SizeOption(t *testing.T) {
	ratio := some[float32](2)
	if got := SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: some[float32](10)}, ratio); got != NewSizeOptionF32(10, 5) {
		t.Errorf("width only = %v", got)
	}
	if got := SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Height: some[float32](10)}, ratio); got != NewSizeOptionF32(20, 10) {
		t.Errorf("height only = %v", got)
	}
	if got := SizeOptionF32MaybeApplyAspectRatio(SizeOptionF32None, ratio); got != SizeOptionF32None {
		t.Errorf("none = %v", got)
	}
	if got := SizeOptionF32MaybeApplyAspectRatio(Size[Option[float32]]{Width: some[float32](10)}, none[float32]()); got != (Size[Option[float32]]{Width: some[float32](10)}) {
		t.Errorf("no ratio = %v", got)
	}
	partial := Size[Option[float32]]{Width: some[float32](1)}
	if SizeOptionUnwrapOr(partial, Size[float32]{Width: 7, Height: 8}) != (Size[float32]{Width: 1, Height: 8}) {
		t.Error("SizeOptionUnwrapOr")
	}
	if SizeOptionOr(partial, NewSizeOptionF32(3, 4)) != NewSizeOptionF32(1, 4) {
		t.Error("SizeOptionOr")
	}
	if SizeOptionBothAxisDefined(partial) || !SizeOptionBothAxisDefined(NewSizeOptionF32(1, 2)) {
		t.Error("SizeOptionBothAxisDefined")
	}
	if SizeDimensionFromLengths(1, 2) != (Size[Dimension]{Width: DimensionLength(1), Height: DimensionLength(2)}) {
		t.Error("SizeDimensionFromLengths")
	}
}

func TestGeometry_ArithmeticDoesNotAllocate(t *testing.T) {
	r := NewRectF32(1, 2, 3, 4)
	s := Size[float32]{Width: 1, Height: 2}
	var sink float32
	allocs := testing.AllocsPerRun(100, func() {
		sum := r.Add(r).SumAxes().Add(s).Sub(s).F32Max(s).F32Min(s)
		sink += sum.Width + r.Union(r).Left + (Line[float32]{Start: 1, End: 2}).Sum()
	})
	if allocs != 0 {
		t.Errorf("allocs = %v, want 0", allocs)
	}
	_ = sink
}

func TestGeometry_ConditionalMethodsPanic(t *testing.T) {
	cases := []struct {
		name string
		f    func()
		want string
	}{
		{"Add", func() { (Size[string]{}).Add(Size[string]{}) }, "taffy: Add is not defined for string"},
		{"F32Max", func() { (Size[int]{}).F32Max(Size[int]{}) }, "Size.F32Max is only defined for Size[float32]"},
		{"Union", func() { (Rect[int]{}).Union(Rect[int]{}) }, "Rect.Union is only defined for Rect[float32]"},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if r := fmt.Sprint(recover()); !strings.Contains(r, c.want) {
					t.Errorf("%s: panic = %q, want %q", c.name, r, c.want)
				}
			}()
			c.f()
		}()
	}
}
