package taffy

import "testing"

var noneF32 = none[float32]()

func checkOpt(t *testing.T, got, want Option[float32]) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func checkF32(t *testing.T, got, want float32) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUtilMath_LhsOptionF32RhsOptionF32_MaybeMin(t *testing.T) {
	checkOpt(t, OptionF32MaybeMin(some[float32](3.0), some[float32](5.0)), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMin(some[float32](5.0), some[float32](3.0)), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMin(some[float32](3.0), noneF32), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMin(noneF32, some[float32](3.0)), noneF32)
	checkOpt(t, OptionF32MaybeMin(noneF32, noneF32), noneF32)
}

func TestUtilMath_LhsOptionF32RhsOptionF32_MaybeMax(t *testing.T) {
	checkOpt(t, OptionF32MaybeMax(some[float32](3.0), some[float32](5.0)), some[float32](5.0))
	checkOpt(t, OptionF32MaybeMax(some[float32](5.0), some[float32](3.0)), some[float32](5.0))
	checkOpt(t, OptionF32MaybeMax(some[float32](3.0), noneF32), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMax(noneF32, some[float32](3.0)), noneF32)
	checkOpt(t, OptionF32MaybeMax(noneF32, noneF32), noneF32)
}

func TestUtilMath_LhsOptionF32RhsOptionF32_MaybeAdd(t *testing.T) {
	checkOpt(t, OptionF32MaybeAdd(some[float32](3.0), some[float32](5.0)), some[float32](8.0))
	checkOpt(t, OptionF32MaybeAdd(some[float32](5.0), some[float32](3.0)), some[float32](8.0))
	checkOpt(t, OptionF32MaybeAdd(some[float32](3.0), noneF32), some[float32](3.0))
	checkOpt(t, OptionF32MaybeAdd(noneF32, some[float32](3.0)), noneF32)
	checkOpt(t, OptionF32MaybeAdd(noneF32, noneF32), noneF32)
}

func TestUtilMath_LhsOptionF32RhsOptionF32_MaybeSub(t *testing.T) {
	checkOpt(t, OptionF32MaybeSub(some[float32](3.0), some[float32](5.0)), some[float32](-2.0))
	checkOpt(t, OptionF32MaybeSub(some[float32](5.0), some[float32](3.0)), some[float32](2.0))
	checkOpt(t, OptionF32MaybeSub(some[float32](3.0), noneF32), some[float32](3.0))
	checkOpt(t, OptionF32MaybeSub(noneF32, some[float32](3.0)), noneF32)
	checkOpt(t, OptionF32MaybeSub(noneF32, noneF32), noneF32)
}

func TestUtilMath_LhsOptionF32RhsF32_MaybeMin(t *testing.T) {
	checkOpt(t, OptionF32MaybeMinF32(some[float32](3.0), 5.0), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMinF32(some[float32](5.0), 3.0), some[float32](3.0))
	checkOpt(t, OptionF32MaybeMinF32(noneF32, 3.0), noneF32)
}

func TestUtilMath_LhsOptionF32RhsF32_MaybeMax(t *testing.T) {
	checkOpt(t, OptionF32MaybeMaxF32(some[float32](3.0), 5.0), some[float32](5.0))
	checkOpt(t, OptionF32MaybeMaxF32(some[float32](5.0), 3.0), some[float32](5.0))
	checkOpt(t, OptionF32MaybeMaxF32(noneF32, 3.0), noneF32)
}

func TestUtilMath_LhsOptionF32RhsF32_MaybeAdd(t *testing.T) {
	checkOpt(t, OptionF32MaybeAddF32(some[float32](3.0), 5.0), some[float32](8.0))
	checkOpt(t, OptionF32MaybeAddF32(some[float32](5.0), 3.0), some[float32](8.0))
	checkOpt(t, OptionF32MaybeAddF32(noneF32, 3.0), noneF32)
}

func TestUtilMath_LhsOptionF32RhsF32_MaybeSub(t *testing.T) {
	checkOpt(t, OptionF32MaybeSubF32(some[float32](3.0), 5.0), some[float32](-2.0))
	checkOpt(t, OptionF32MaybeSubF32(some[float32](5.0), 3.0), some[float32](2.0))
	checkOpt(t, OptionF32MaybeSubF32(noneF32, 3.0), noneF32)
}

func TestUtilMath_LhsF32RhsOptionF32_MaybeMin(t *testing.T) {
	checkF32(t, F32MaybeMin(3.0, some[float32](5.0)), 3.0)
	checkF32(t, F32MaybeMin(5.0, some[float32](3.0)), 3.0)
	checkF32(t, F32MaybeMin(3.0, noneF32), 3.0)
}

func TestUtilMath_LhsF32RhsOptionF32_MaybeMax(t *testing.T) {
	checkF32(t, F32MaybeMax(3.0, some[float32](5.0)), 5.0)
	checkF32(t, F32MaybeMax(5.0, some[float32](3.0)), 5.0)
	checkF32(t, F32MaybeMax(3.0, noneF32), 3.0)
}

func TestUtilMath_LhsF32RhsOptionF32_MaybeAdd(t *testing.T) {
	checkF32(t, F32MaybeAdd(3.0, some[float32](5.0)), 8.0)
	checkF32(t, F32MaybeAdd(5.0, some[float32](3.0)), 8.0)
	checkF32(t, F32MaybeAdd(3.0, noneF32), 3.0)
}

func TestUtilMath_LhsF32RhsOptionF32_MaybeSub(t *testing.T) {
	checkF32(t, F32MaybeSub(3.0, some[float32](5.0)), -2.0)
	checkF32(t, F32MaybeSub(5.0, some[float32](3.0)), 2.0)
	checkF32(t, F32MaybeSub(3.0, noneF32), 3.0)
}

func TestUtilMath_MaybeClamp(t *testing.T) {
	checkOpt(t, OptionF32MaybeClamp(some[float32](7), some[float32](1), some[float32](5)), some[float32](5))
	checkOpt(t, OptionF32MaybeClamp(some[float32](7), noneF32, some[float32](5)), some[float32](5))
	checkOpt(t, OptionF32MaybeClamp(some[float32](0), some[float32](1), noneF32), some[float32](1))
	checkOpt(t, OptionF32MaybeClamp(some[float32](3), noneF32, noneF32), some[float32](3))
	checkOpt(t, OptionF32MaybeClamp(noneF32, some[float32](1), some[float32](5)), noneF32)
	checkOpt(t, OptionF32MaybeClamp(some[float32](3), some[float32](5), some[float32](1)), some[float32](5))
	checkOpt(t, OptionF32MaybeClampF32(some[float32](9), 1, 5), some[float32](5))
	checkF32(t, F32MaybeClamp(9, some[float32](1), some[float32](5)), 5)
	checkF32(t, F32MaybeClamp(9, noneF32, noneF32), 9)
}

func TestUtilMath_AvailableSpace(t *testing.T) {
	def := AvailableSpaceDefinite
	cases := []struct {
		got, want AvailableSpace
	}{
		{def(3).MaybeMinF32(5), def(3)},
		{AvailableSpaceMinContent.MaybeMinF32(5), def(5)},
		{AvailableSpaceMaxContent.MaybeMinF32(5), def(5)},
		{def(3).MaybeMaxF32(5), def(5)},
		{AvailableSpaceMinContent.MaybeMaxF32(5), AvailableSpaceMinContent},
		{def(9).MaybeClampF32(1, 5), def(5)},
		{AvailableSpaceMaxContent.MaybeClampF32(1, 5), AvailableSpaceMaxContent},
		{def(3).MaybeAddF32(5), def(8)},
		{def(3).MaybeSubF32(5), def(-2)},
		{def(3).MaybeMin(some[float32](5)), def(3)},
		{def(3).MaybeMin(noneF32), def(3)},
		{AvailableSpaceMinContent.MaybeMin(some[float32](5)), def(5)},
		{AvailableSpaceMinContent.MaybeMin(noneF32), AvailableSpaceMinContent},
		{AvailableSpaceMaxContent.MaybeMin(some[float32](5)), def(5)},
		{AvailableSpaceMaxContent.MaybeMin(noneF32), AvailableSpaceMaxContent},
		{def(3).MaybeMax(some[float32](5)), def(5)},
		{AvailableSpaceMaxContent.MaybeMax(some[float32](5)), AvailableSpaceMaxContent},
		{def(9).MaybeClamp(some[float32](1), some[float32](5)), def(5)},
		{def(0).MaybeClamp(some[float32](1), noneF32), def(1)},
		{AvailableSpaceMinContent.MaybeClamp(some[float32](1), some[float32](5)), AvailableSpaceMinContent},
		{def(3).MaybeAdd(some[float32](5)), def(8)},
		{def(3).MaybeAdd(noneF32), def(3)},
		{def(3).MaybeSub(some[float32](5)), def(-2)},
		{AvailableSpaceMinContent.MaybeSub(some[float32](5)), AvailableSpaceMinContent},
	}
	for i, c := range cases {
		if c.got != c.want {
			t.Errorf("case %d: got %v, want %v", i, c.got, c.want)
		}
	}
}

func TestUtilMath_Size(t *testing.T) {
	s := NewSizeOptionF32(3, 5)
	if got := SizeOptionF32MaybeMin(s, Size[Option[float32]]{Width: some[float32](4)}); got != (Size[Option[float32]]{Width: some[float32](3), Height: some[float32](5)}) {
		t.Errorf("got %v", got)
	}
	if got := SizeF32MaybeAdd(Size[float32]{Width: 1, Height: 2}, NewSizeOptionF32(1, 1)); got != (Size[float32]{Width: 2, Height: 3}) {
		t.Errorf("got %v", got)
	}
	as := Size[AvailableSpace]{Width: AvailableSpaceDefinite(10), Height: AvailableSpaceMinContent}
	if got := SizeAvailableSpaceMaybeSubF32(as, Size[float32]{Width: 4, Height: 4}); got != (Size[AvailableSpace]{Width: AvailableSpaceDefinite(6), Height: AvailableSpaceMinContent}) {
		t.Errorf("got %v", got)
	}
	if got := SizeOptionF32MaybeClampF32(s, Size[float32]{Width: 4, Height: 0}, Size[float32]{Width: 9, Height: 4}); got != NewSizeOptionF32(4, 4) {
		t.Errorf("got %v", got)
	}
}

func TestUtilMath_NaN(t *testing.T) {
	nan := float32FromBits(0x7fc00000)
	checkOpt(t, OptionF32MaybeMin(some(nan), some[float32](1)), some[float32](1))
	checkOpt(t, OptionF32MaybeMax(some[float32](1), some(nan)), some[float32](1))
	checkF32(t, F32MaybeMax(nan, some[float32](2)), 2)
}
