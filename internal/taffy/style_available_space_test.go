package taffy

import (
	"fmt"
	"math"
	"testing"
)

func TestStyleAvailableSpace_Methods(t *testing.T) {
	d := AvailableSpaceDefinite(10)
	if !d.IsDefinite() || AvailableSpaceMinContent.IsDefinite() {
		t.Error("IsDefinite")
	}
	if d.IntoOption() != some[float32](10) || AvailableSpaceMaxContent.IntoOption().IsSome() {
		t.Error("IntoOption")
	}
	if d.UnwrapOr(3) != 10 || AvailableSpaceMinContent.UnwrapOr(3) != 3 {
		t.Error("UnwrapOr")
	}
	if AvailableSpaceMinContent.Or(d) != d || d.Or(AvailableSpaceMinContent) != d {
		t.Error("Or")
	}
	if AvailableSpaceMaxContent.OrElse(func() AvailableSpace { return d }) != d {
		t.Error("OrElse")
	}
	if AvailableSpaceMaxContent.UnwrapOrElse(func() float32 { return 4 }) != 4 || d.UnwrapOrElse(func() float32 { return 4 }) != 10 {
		t.Error("UnwrapOrElse")
	}
	if AvailableSpaceMinContent.MaybeSet(some[float32](2)) != AvailableSpaceDefinite(2) || AvailableSpaceMinContent.MaybeSet(none[float32]()) != AvailableSpaceMinContent {
		t.Error("MaybeSet")
	}
	if d.MapDefiniteValue(func(v float32) float32 { return v / 2 }) != AvailableSpaceDefinite(5) || AvailableSpaceMinContent.MapDefiniteValue(func(float32) float32 { return 1 }) != AvailableSpaceMinContent {
		t.Error("MapDefiniteValue")
	}
	if d.ComputeFreeSpace(4) != 6 || AvailableSpaceMinContent.ComputeFreeSpace(4) != 0 || !math.IsInf(float64(AvailableSpaceMaxContent.ComputeFreeSpace(4)), 1) {
		t.Error("ComputeFreeSpace")
	}
	if !d.IsRoughlyEqual(AvailableSpaceDefinite(10.0000001)) || d.IsRoughlyEqual(AvailableSpaceDefinite(10.01)) || !AvailableSpaceMinContent.IsRoughlyEqual(AvailableSpaceMinContent) || AvailableSpaceMinContent.IsRoughlyEqual(AvailableSpaceMaxContent) || d.IsRoughlyEqual(AvailableSpaceMaxContent) {
		t.Error("IsRoughlyEqual")
	}
	if AvailableSpaceFromF32(1) != AvailableSpaceDefinite(1) || AvailableSpaceFromOptionF32(none[float32]()) != AvailableSpaceMaxContent || AvailableSpaceFromOptionF32(some[float32](1)) != AvailableSpaceDefinite(1) {
		t.Error("From")
	}
	if (AvailableSpace{}).Zero() != AvailableSpaceDefinite(0) || (AvailableSpace{}).FromLength(2) != AvailableSpaceDefinite(2) {
		t.Error("trait values")
	}
	s := Size[AvailableSpace]{Width: d, Height: AvailableSpaceMinContent}
	if SizeAvailableSpaceIntoOptions(s) != (Size[Option[float32]]{Width: some[float32](10)}) {
		t.Error("SizeAvailableSpaceIntoOptions")
	}
	if SizeAvailableSpaceMaybeSet(s, Size[Option[float32]]{Height: some[float32](1)}) != (Size[AvailableSpace]{Width: d, Height: AvailableSpaceDefinite(1)}) {
		t.Error("SizeAvailableSpaceMaybeSet")
	}
	defer func() {
		if r := fmt.Sprint(recover()); r != "called `Option::unwrap()` on a `None` value" {
			t.Errorf("panic = %q", r)
		}
	}()
	AvailableSpaceMinContent.Unwrap()
}
