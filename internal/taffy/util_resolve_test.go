package taffy

import (
	"testing"
	"unsafe"
)

func resolveTestCalc(unsafe.Pointer, float32) float32 {
	return 42.42
}

func mrCase[Lhs MaybeResolve[Rhs, Out], Rhs any, Out comparable](t *testing.T, input Lhs, context Rhs, expected Out) {
	t.Helper()
	if got := input.MaybeResolve(context, resolveTestCalc); got != expected {
		t.Fatalf("MaybeResolve(%v, %v) = %v, want %v", input, context, got, expected)
	}
}

func mrSizeCase(t *testing.T, input Size[Dimension], context Size[Option[float32]], expected Size[Option[float32]]) {
	t.Helper()
	if got := SizeMaybeResolve(input, context, resolveTestCalc); got != expected {
		t.Fatalf("SizeMaybeResolve(%v, %v) = %v, want %v", input, context, got, expected)
	}
}

func rozCase[Lhs ResolveOrZero[Rhs, Out], Rhs any, Out comparable](t *testing.T, input Lhs, context Rhs, expected Out) {
	t.Helper()
	if got := input.ResolveOrZero(context, resolveTestCalc); got != expected {
		t.Fatalf("ResolveOrZero(%v, %v) = %v, want %v", input, context, got, expected)
	}
}

func rozRectCase(t *testing.T, input Rect[Dimension], context Size[Option[float32]], expected Rect[float32]) {
	t.Helper()
	if got := RectResolveOrZero[Dimension, Option[float32], float32](input, context, resolveTestCalc); got != expected {
		t.Fatalf("RectResolveOrZero(%v, %v) = %v, want %v", input, context, got, expected)
	}
}

func rozRectOptionCase(t *testing.T, input Rect[Dimension], context Option[float32], expected Rect[float32]) {
	t.Helper()
	if got := RectResolveOrZeroOptionF32[Dimension, float32](input, context, resolveTestCalc); got != expected {
		t.Fatalf("RectResolveOrZeroOptionF32(%v, %v) = %v, want %v", input, context, got, expected)
	}
}

var (
	resolveNone = none[float32]()
	sizeNew     = NewSizeOptionF32
	rectNew     = NewRectF32
)

func TestUtilResolve_MaybeResolveDimension_ResolveAuto(t *testing.T) {
	mrCase(t, DimensionAuto(), resolveNone, resolveNone)
	mrCase(t, DimensionAuto(), some[float32](5.0), resolveNone)
	mrCase(t, DimensionAuto(), some[float32](-5.0), resolveNone)
	mrCase(t, DimensionAuto(), some[float32](0.), resolveNone)
}

func TestUtilResolve_MaybeResolveDimension_ResolveLength(t *testing.T) {
	d := Dimension{}.FromLength(1.0)
	mrCase(t, d, resolveNone, some[float32](1.0))
	mrCase(t, d, some[float32](5.0), some[float32](1.0))
	mrCase(t, d, some[float32](-5.0), some[float32](1.0))
	mrCase(t, d, some[float32](0.), some[float32](1.0))
}

func TestUtilResolve_MaybeResolveDimension_ResolvePercent(t *testing.T) {
	d := Dimension{}.FromPercent(1.0)
	mrCase(t, d, resolveNone, resolveNone)
	mrCase(t, d, some[float32](5.0), some[float32](5.0))
	mrCase(t, d, some[float32](-5.0), some[float32](-5.0))
	mrCase(t, d, some[float32](50.0), some[float32](50.0))
}

func TestUtilResolve_MaybeResolveSizeDimension_MaybeResolveAuto(t *testing.T) {
	auto := SizeAuto[Dimension]()
	mrSizeCase(t, auto, SizeOptionF32None, SizeOptionF32None)
	mrSizeCase(t, auto, sizeNew(5.0, 5.0), SizeOptionF32None)
	mrSizeCase(t, auto, sizeNew(-5.0, -5.0), SizeOptionF32None)
	mrSizeCase(t, auto, sizeNew(0.0, 0.0), SizeOptionF32None)
}

func TestUtilResolve_MaybeResolveSizeDimension_MaybeResolveLength(t *testing.T) {
	s := SizeDimensionFromLengths(5.0, 5.0)
	mrSizeCase(t, s, SizeOptionF32None, sizeNew(5.0, 5.0))
	mrSizeCase(t, s, sizeNew(5.0, 5.0), sizeNew(5.0, 5.0))
	mrSizeCase(t, s, sizeNew(-5.0, -5.0), sizeNew(5.0, 5.0))
	mrSizeCase(t, s, sizeNew(0.0, 0.0), sizeNew(5.0, 5.0))
}

func TestUtilResolve_MaybeResolveSizeDimension_MaybeResolvePercent(t *testing.T) {
	s := SizeDimensionFromPercent(5.0, 5.0)
	mrSizeCase(t, s, SizeOptionF32None, SizeOptionF32None)
	mrSizeCase(t, s, sizeNew(5.0, 5.0), sizeNew(25.0, 25.0))
	mrSizeCase(t, s, sizeNew(-5.0, -5.0), sizeNew(-25.0, -25.0))
	mrSizeCase(t, s, sizeNew(0.0, 0.0), sizeNew(0.0, 0.0))
}

func TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroAuto(t *testing.T) {
	rozCase(t, DimensionAuto(), resolveNone, float32(0.0))
	rozCase(t, DimensionAuto(), some[float32](5.0), float32(0.0))
	rozCase(t, DimensionAuto(), some[float32](-5.0), float32(0.0))
	rozCase(t, DimensionAuto(), some[float32](0.0), float32(0.0))
}

func TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroLength(t *testing.T) {
	d := Dimension{}.FromLength(5.0)
	rozCase(t, d, resolveNone, float32(5.0))
	rozCase(t, d, some[float32](5.0), float32(5.0))
	rozCase(t, d, some[float32](-5.0), float32(5.0))
	rozCase(t, d, some[float32](0.0), float32(5.0))
}

func TestUtilResolve_ResolveOrZeroDimensionToOptionF32_ResolveOrZeroPercent(t *testing.T) {
	d := Dimension{}.FromPercent(5.0)
	rozCase(t, d, resolveNone, float32(0.0))
	rozCase(t, d, some[float32](5.0), float32(25.0))
	rozCase(t, d, some[float32](-5.0), float32(-25.0))
	rozCase(t, d, some[float32](0.0), float32(0.0))
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroAuto(t *testing.T) {
	auto := RectAuto[Dimension]()
	rozRectCase(t, auto, SizeOptionF32None, RectF32Zero)
	rozRectCase(t, auto, sizeNew(5.0, 5.0), RectF32Zero)
	rozRectCase(t, auto, sizeNew(-5.0, -5.0), RectF32Zero)
	rozRectCase(t, auto, sizeNew(0.0, 0.0), RectF32Zero)
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroLength(t *testing.T) {
	r := RectDimensionFromLength(5.0, 5.0, 5.0, 5.0)
	rozRectCase(t, r, SizeOptionF32None, rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectCase(t, r, sizeNew(5.0, 5.0), rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectCase(t, r, sizeNew(-5.0, -5.0), rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectCase(t, r, sizeNew(0.0, 0.0), rectNew(5.0, 5.0, 5.0, 5.0))
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRect_ResolveOrZeroPercent(t *testing.T) {
	r := RectDimensionFromPercent(5.0, 5.0, 5.0, 5.0)
	rozRectCase(t, r, SizeOptionF32None, RectF32Zero)
	rozRectCase(t, r, sizeNew(5.0, 5.0), rectNew(25.0, 25.0, 25.0, 25.0))
	rozRectCase(t, r, sizeNew(-5.0, -5.0), rectNew(-25.0, -25.0, -25.0, -25.0))
	rozRectCase(t, r, sizeNew(0.0, 0.0), RectF32Zero)
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroAuto(t *testing.T) {
	auto := RectAuto[Dimension]()
	rozRectOptionCase(t, auto, resolveNone, RectF32Zero)
	rozRectOptionCase(t, auto, some[float32](5.0), RectF32Zero)
	rozRectOptionCase(t, auto, some[float32](-5.0), RectF32Zero)
	rozRectOptionCase(t, auto, some[float32](0.0), RectF32Zero)
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroLength(t *testing.T) {
	r := RectDimensionFromLength(5.0, 5.0, 5.0, 5.0)
	rozRectOptionCase(t, r, resolveNone, rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectOptionCase(t, r, some[float32](5.0), rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectOptionCase(t, r, some[float32](-5.0), rectNew(5.0, 5.0, 5.0, 5.0))
	rozRectOptionCase(t, r, some[float32](0.0), rectNew(5.0, 5.0, 5.0, 5.0))
}

func TestUtilResolve_ResolveOrZeroRectDimensionToRectF32ViaOption_ResolveOrZeroPercent(t *testing.T) {
	r := RectDimensionFromPercent(5.0, 5.0, 5.0, 5.0)
	rozRectOptionCase(t, r, resolveNone, RectF32Zero)
	rozRectOptionCase(t, r, some[float32](5.0), rectNew(25.0, 25.0, 25.0, 25.0))
	rozRectOptionCase(t, r, some[float32](-5.0), rectNew(-25.0, -25.0, -25.0, -25.0))
	rozRectOptionCase(t, r, some[float32](0.0), RectF32Zero)
}
