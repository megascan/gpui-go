package gpui

import (
	"encoding/json/v2"
	"fmt"
	"math"
	"testing"
	"unsafe"
)

var (
	_ Along[Pixels, Point[Pixels]]   = Point[Pixels]{}
	_ Along[Pixels, Size[Pixels]]    = Size[Pixels]{}
	_ Along[Pixels, Edges[Pixels]]   = Edges[Pixels]{}
	_ Along[string, PointOf[string]] = PointOf[string]{}
	_ Along[string, SizeOf[string]]  = SizeOf[string]{}
)

func TestGeometry_BoundsJoin(t *testing.T) {
	bounds := Bounds[float32]{Origin: Point[float32]{X: 5, Y: 5}, Size: Size[float32]{Width: 5, Height: 5}}
	other := Bounds[float32]{Origin: Point[float32]{X: 20, Y: 0}, Size: Size[float32]{Width: 5, Height: 5}}
	if bounds.Join(other) != bounds.Union(other) {
		t.Fatalf("join %v != union %v", bounds.Join(other), bounds.Union(other))
	}
	empty := Bounds[float32]{Origin: Point[float32]{X: 0, Y: 0}, Size: Size[float32]{Width: 0, Height: 10}}
	if bounds.Join(empty) != bounds {
		t.Fatalf("bounds.join(empty) = %v", bounds.Join(empty))
	}
	if empty.Join(bounds) != bounds {
		t.Fatalf("empty.join(bounds) = %v", empty.Join(bounds))
	}
	if (Bounds[float32]{}).Join(bounds) != bounds {
		t.Fatalf("default.join(bounds) = %v", (Bounds[float32]{}).Join(bounds))
	}
	if bounds.Union(empty) == bounds {
		t.Fatalf("bounds.union(empty) == bounds")
	}
}

func TestGeometry_BoundsIntersects(t *testing.T) {
	bounds1 := Bounds[float32]{Origin: Point[float32]{X: 0, Y: 0}, Size: Size[float32]{Width: 5, Height: 5}}
	bounds2 := Bounds[float32]{Origin: Point[float32]{X: 4, Y: 4}, Size: Size[float32]{Width: 5, Height: 5}}
	bounds3 := Bounds[float32]{Origin: Point[float32]{X: 10, Y: 10}, Size: Size[float32]{Width: 5, Height: 5}}
	if !bounds1.Intersects(bounds2) {
		t.Fatal("bounds1 should intersect bounds2")
	}
	if bounds1.Intersects(bounds3) {
		t.Fatal("bounds1 should not intersect bounds3")
	}
	if !bounds1.Intersects(bounds1) {
		t.Fatal("bounds1 should intersect itself")
	}
}

func TestGeometry_DocExamples(t *testing.T) {
	check := func(name string, got, want any) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	check("point", NewPoint(10, 20), Point[int]{X: 10, Y: 20})
	check("point map", Point[int]{X: 3, Y: 4}.Map(func(c int) float32 { return float32(c) }), Point[float32]{X: 3, Y: 4})
	check("point scale", Point[Pixels]{X: 10, Y: 20}.Scale(1.5), Point[ScaledPixels]{X: 15, Y: 30})
	check("magnitude", Point[Pixels]{X: 3, Y: 4}.Magnitude(), 5.0)
	check("point max", Point[int]{X: 3, Y: 7}.Max(Point[int]{X: 5, Y: 2}), Point[int]{X: 5, Y: 7})
	check("point min", Point[int]{X: 3, Y: 7}.Min(Point[int]{X: 5, Y: 2}), Point[int]{X: 3, Y: 2})
	minP, maxP := Point[int]{X: 0, Y: 5}, Point[int]{X: 15, Y: 25}
	check("clamp in", Point[int]{X: 10, Y: 20}.Clamp(minP, maxP), Point[int]{X: 10, Y: 20})
	check("clamp out", Point[int]{X: -5, Y: 30}.Clamp(minP, maxP), Point[int]{X: 0, Y: 25})
	check("size", NewSize(10, 20), Size[int]{Width: 10, Height: 20})
	check("size map", Size[int]{Width: 10, Height: 20}.Map(func(d int) float32 { return float32(d) * 1.5 }), Size[float32]{Width: 15, Height: 30})
	check("size scale", Size[Pixels]{Width: 100, Height: 50}.Scale(2), Size[ScaledPixels]{Width: 200, Height: 100})
	check("size max", Size[int]{Width: 30, Height: 40}.Max(Size[int]{Width: 50, Height: 20}), Size[int]{Width: 50, Height: 40})
	check("size min", Size[int]{Width: 30, Height: 40}.Min(Size[int]{Width: 50, Height: 20}), Size[int]{Width: 30, Height: 20})
	origin, size := Point[int]{}, Size[int]{Width: 10, Height: 20}
	check("bounds new", NewBounds(origin, size), Bounds[int]{Origin: origin, Size: size})
	fc := BoundsFromCorners(Point[int]{}, Point[int]{X: 10, Y: 10})
	check("from corners", fc, Bounds[int]{Size: Size[int]{Width: 10, Height: 10}})
	b1 := Bounds[int]{Size: Size[int]{Width: 10, Height: 10}}
	b2 := Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 10, Height: 10}}
	b3 := Bounds[int]{Origin: Point[int]{X: 20, Y: 20}, Size: Size[int]{Width: 10, Height: 10}}
	check("intersects", b1.Intersects(b2), true)
	check("not intersects", b1.Intersects(b3), false)
	check("center", Bounds[int]{Size: Size[int]{Width: 10, Height: 20}}.Center(), Point[int]{X: 5, Y: 10})
	check("half perimeter", Bounds[int]{Size: Size[int]{Width: 10, Height: 20}}.HalfPerimeter(), 30)
	dil := Bounds[int]{Origin: Point[int]{X: 10, Y: 10}, Size: Size[int]{Width: 10, Height: 10}}.Dilate(5)
	check("dilate", dil, Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 20, Height: 20}})
	check("intersect", b1.Intersect(b2), Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 5, Height: 5}})
	check("intersect none", b1.Intersect(b3), Bounds[int]{Origin: Point[int]{X: 20, Y: 20}})
	u2 := Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 15, Height: 15}}
	check("union", b1.Union(u2), Bounds[int]{Size: Size[int]{Width: 20, Height: 20}})
	jb := Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 10, Height: 10}}
	je := Bounds[int]{Size: Size[int]{Width: 0, Height: 10}}
	check("join", jb.Join(je), jb)
	check("join rev", je.Join(jb), jb)
	b := Bounds[int]{Size: Size[int]{Width: 10, Height: 20}}
	check("top right", b.TopRight(), Point[int]{X: 10, Y: 0})
	check("bottom right", b.BottomRight(), Point[int]{X: 10, Y: 20})
	check("bottom left", b.BottomLeft(), Point[int]{X: 0, Y: 20})
	check("corner", b.Corner(AnchorBottomLeft), Point[int]{X: 0, Y: 20})
	check("contains", b1.Contains(Point[int]{X: 5, Y: 5}), true)
	check("contains out", b1.Contains(Point[int]{X: 15, Y: 15}), false)
	check("contains edge", b1.Contains(Point[int]{X: 10, Y: 0}), false)
	outer := Bounds[int]{Size: Size[int]{Width: 20, Height: 20}}
	check("contained", Bounds[int]{Origin: Point[int]{X: 5, Y: 5}, Size: Size[int]{Width: 10, Height: 10}}.IsContainedWithin(outer), true)
	check("not contained", Bounds[int]{Origin: Point[int]{X: 15, Y: 15}, Size: Size[int]{Width: 10, Height: 10}}.IsContainedWithin(outer), false)
	fb := Bounds[float64]{Origin: Point[float64]{X: 10, Y: 10}, Size: Size[float64]{Width: 10, Height: 20}}
	check("map", fb.Map(func(v float64) float64 { return v * 1.5 }), Bounds[float64]{Origin: Point[float64]{X: 15, Y: 15}, Size: Size[float64]{Width: 15, Height: 30}})
	check("map origin", fb.MapOrigin(func(v float64) float64 { return v * 1.5 }), Bounds[float64]{Origin: Point[float64]{X: 15, Y: 15}, Size: Size[float64]{Width: 10, Height: 20}})
	check("map size", fb.MapSize(func(v float64) float64 { return v * 1.5 }), Bounds[float64]{Origin: Point[float64]{X: 10, Y: 10}, Size: Size[float64]{Width: 15, Height: 30}})
	pb := Bounds[Pixels]{Origin: Point[Pixels]{X: 10, Y: 20}, Size: Size[Pixels]{Width: 30, Height: 40}}
	check("bounds scale", pb.Scale(2), Bounds[ScaledPixels]{Origin: Point[ScaledPixels]{X: 20, Y: 40}, Size: Size[ScaledPixels]{Width: 60, Height: 80}})
	e := Edges[int]{Top: 10, Right: 0, Bottom: 5, Left: 0}
	check("any zero", e.Any(func(v int) bool { return v == 0 }), true)
	check("any pos", e.Any(func(v int) bool { return v > 0 }), true)
	check("any big", e.Any(func(v int) bool { return v > 10 }), false)
	check("edges map", Edges[int]{Top: 10, Right: 20, Bottom: 30, Left: 40}.Map(func(v int) float32 { return float32(v) * 1.1 }),
		Edges[float32]{Top: float32(10) * 1.1, Right: float32(20) * 1.1, Bottom: float32(30) * 1.1, Left: float32(40) * 1.1})
	check("edges all", EdgesAll[float32](10), Edges[float32]{Top: 10, Right: 10, Bottom: 10, Left: 10})
	check("edges scale", Edges[Pixels]{Top: 10, Right: 20, Bottom: 30, Left: 40}.Scale(2), Edges[ScaledPixels]{Top: 20, Right: 40, Bottom: 60, Left: 80})
	check("edges max", Edges[Pixels]{Top: 1, Right: 4, Bottom: 3, Left: 2}.Max(), Pixels(4))
	check("opposite", AnchorTopLeft.Opposite(), AnchorBottomRight)
	check("other side", AnchorTopLeft.OtherSideAlong(AxisHorizontal), AnchorTopRight)
	check("invert", AxisVertical.Invert(), AxisHorizontal)
}

func TestGeometry_Anchor(t *testing.T) {
	all := []Anchor{AnchorTopLeft, AnchorTopRight, AnchorBottomLeft, AnchorBottomRight, AnchorTopCenter, AnchorBottomCenter, AnchorLeftCenter, AnchorRightCenter}
	b := Bounds[int]{Origin: Point[int]{X: 10, Y: 20}, Size: Size[int]{Width: 30, Height: 40}}
	for _, a := range all {
		if a.Opposite().Opposite() != a {
			t.Errorf("%v: opposite is not an involution", a)
		}
		for _, axis := range []Axis{AxisVertical, AxisHorizontal} {
			if a.OtherSideAlong(axis).OtherSideAlong(axis) != a {
				t.Errorf("%v: other side along %v is not an involution", a, axis)
			}
		}
		placed := BoundsFromAnchorAndSize(a, b.Corner(a), b.Size)
		if placed != b {
			t.Errorf("%v: from anchor and size = %v, want %v", a, placed, b)
		}
	}
	if !AnchorLeftCenter.IsCenter() || AnchorTopLeft.IsCenter() {
		t.Error("IsCenter")
	}
	if !AnchorBottomCenter.IsBottom() || AnchorLeftCenter.IsBottom() {
		t.Error("IsBottom")
	}
}

func TestGeometry_DevicePixelConversions(t *testing.T) {
	b := Bounds[Pixels]{Origin: Point[Pixels]{X: 1.25, Y: -1.25}, Size: Size[Pixels]{Width: 10.5, Height: 3}}
	got := b.ToDevicePixels(2)
	want := Bounds[DevicePixels]{Origin: Point[DevicePixels]{X: 3, Y: -3}, Size: Size[DevicePixels]{Width: 21, Height: 6}}
	if got != want {
		t.Fatalf("to device pixels = %v, want %v", got, want)
	}
	back := want.ToPixels(2)
	if back != (Bounds[Pixels]{Origin: Point[Pixels]{X: 1.5, Y: -1.5}, Size: Size[Pixels]{Width: 10.5, Height: 3}}) {
		t.Fatalf("to pixels = %v", back)
	}
	huge := Size[Pixels]{Width: Pixels(math.MaxFloat32), Height: Pixels(float32(math.NaN()))}.ToDevicePixels(2)
	if huge != (Size[DevicePixels]{Width: math.MaxInt32, Height: 0}) {
		t.Fatalf("saturating = %v", huge)
	}
}

func TestGeometry_PixelsOrdering(t *testing.T) {
	nan := Pixels(float32(math.NaN()))
	if got := (Point[Pixels]{X: nan, Y: 1}).Max(Point[Pixels]{X: 1, Y: 2}); got.X == 1 || got.Y != 2 {
		t.Fatalf("pixels max with NaN = %v", got)
	}
	fnan := float32(math.NaN())
	if got := (Point[float32]{X: fnan, Y: 1}).Max(Point[float32]{X: 1, Y: 2}); got.X != 1 {
		t.Fatalf("f32 max with NaN = %v", got)
	}
	negZero := ScaledPixels(float32(math.Copysign(0, -1)))
	if got := (Point[ScaledPixels]{X: 0, Y: 0}).Min(Point[ScaledPixels]{X: negZero, Y: 0}); !math.Signbit(float64(got.X)) {
		t.Fatalf("scaled pixels min of 0 and -0 = %v", got)
	}
}

func TestGeometry_Formatting(t *testing.T) {
	cases := []struct{ got, want string }{
		{fmt.Sprint(Point[int]{X: 1, Y: 2}), "(1, 2)"},
		{fmt.Sprintf("%#v", Point[int]{X: 1, Y: 2}), "Point { x: 1, y: 2 }"},
		{fmt.Sprint(Size[int]{Width: 3, Height: 4}), "3 × 4"},
		{fmt.Sprintf("%#v", Size[int]{Width: 3, Height: 4}), "Size { 3 × 4 }"},
		{fmt.Sprint(Bounds[int]{Origin: Point[int]{X: 1, Y: 2}, Size: Size[int]{Width: 3, Height: 4}}), "(1, 2) - (4, 6) (size 3 × 4)"},
		{fmt.Sprintf("%#v", Bounds[int]{Origin: Point[int]{X: 1, Y: 2}, Size: Size[int]{Width: 3, Height: 4}}), "Bounds { origin: Point { x: 1, y: 2 }, size: Size { 3 × 4 } }"},
		{fmt.Sprint(Edges[int]{Top: 1, Right: 2, Bottom: 3, Left: 4}), "Edges { top: 1, right: 2, bottom: 3, left: 4 }"},
		{AxisHorizontal.String(), "Horizontal"},
		{AnchorRightCenter.String(), "RightCenter"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestGeometry_Serde(t *testing.T) {
	b := Bounds[Pixels]{Origin: Point[Pixels]{X: 1, Y: 2}, Size: Size[Pixels]{Width: 3, Height: 4}}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"origin":{"x":1,"y":2},"size":{"width":3,"height":4}}` {
		t.Fatalf("bounds json = %s", data)
	}
	var back Bounds[Pixels]
	if err := json.Unmarshal(data, &back); err != nil || back != b {
		t.Fatalf("round trip = %v, %v", back, err)
	}
	axisData, err := json.Marshal([]Axis{AxisVertical, AxisHorizontal})
	if err != nil || string(axisData) != `["Vertical","Horizontal"]` {
		t.Fatalf("axis json = %s, %v", axisData, err)
	}
	var axes []Axis
	if err := json.Unmarshal(axisData, &axes); err != nil || len(axes) != 2 || axes[1] != AxisHorizontal {
		t.Fatalf("axis round trip = %v, %v", axes, err)
	}
	var bad Axis
	if err := json.Unmarshal([]byte(`"Diagonal"`), &bad); err == nil {
		t.Fatal("unknown axis variant accepted")
	}
	r := PointRefinement[Pixels]{X: Some[Pixels](5)}
	data, err = json.Marshal(r)
	if err != nil || string(data) != `{"x":5}` {
		t.Fatalf("refinement json = %s, %v", data, err)
	}
}

func TestGeometry_Refinement(t *testing.T) {
	b := Bounds[Pixels]{Origin: Point[Pixels]{X: 1, Y: 2}, Size: Size[Pixels]{Width: 3, Height: 4}}
	refined := b.Refined(BoundsRefinement[Pixels]{Size: Some(Size[Pixels]{Width: 9, Height: 9})})
	if refined != (Bounds[Pixels]{Origin: b.Origin, Size: Size[Pixels]{Width: 9, Height: 9}}) {
		t.Fatalf("refined = %v", refined)
	}
	e := EdgesOf[string]{Top: "a", Right: "b", Bottom: "c", Left: "d"}
	e.Refine(&EdgesOfRefinement[string]{Left: Some("z")})
	if e.Left != "z" || e.Top != "a" {
		t.Fatalf("edges refine = %v", e)
	}
	if !e.IsSupersetOf(&EdgesOfRefinement[string]{Left: Some("z")}) {
		t.Fatal("superset")
	}
}

func TestGeometry_Layout(t *testing.T) {
	if unsafe.Sizeof(Point[ScaledPixels]{}) != 8 || unsafe.Sizeof(Size[DevicePixels]{}) != 8 ||
		unsafe.Sizeof(Bounds[ScaledPixels]{}) != 16 || unsafe.Sizeof(Edges[ScaledPixels]{}) != 16 {
		t.Fatal("geometry structs must be tightly packed 4-byte fields")
	}
	if unsafe.Offsetof(Bounds[ScaledPixels]{}.Size) != 8 || unsafe.Offsetof(Edges[ScaledPixels]{}.Left) != 12 {
		t.Fatal("geometry field order changed")
	}
}

func TestSharedString_Basics(t *testing.T) {
	s := NewSharedString("hello")
	if s.AsStr() != "hello" || s != NewSharedStringStatic("hello") || fmt.Sprint(s) != "hello" {
		t.Fatalf("shared string = %v", s)
	}
	if got := fmt.Sprintf("%#v", NewSharedString("a\"b\n\x01")); got != `"a\"b\n\u{1}"` {
		t.Fatalf("debug = %s", got)
	}
	data, err := json.Marshal(s)
	if err != nil || string(data) != `"hello"` {
		t.Fatalf("json = %s, %v", data, err)
	}
	var zero SharedString
	if zero != "" {
		t.Fatal("default is not empty")
	}
	u := SharedURI("https://zed.dev")
	if fmt.Sprint(u) != "https://zed.dev" || fmt.Sprintf("%#v", u) != `"https://zed.dev"` {
		t.Fatalf("uri = %v", u)
	}
}

func TestGeometry_MapAcrossFamilies(t *testing.T) {
	label := func(p Pixels) string { return fmt.Sprint(float32(p)) }
	if got := (Size[Pixels]{Width: 1, Height: 2}).MapOf(label); got != (SizeOf[string]{Width: "1", Height: "2"}) {
		t.Fatalf("size map of = %v", got)
	}
	if got := (Point[Pixels]{X: 1, Y: 2}).MapOf(label); got != (PointOf[string]{X: "1", Y: "2"}) {
		t.Fatalf("point map of = %v", got)
	}
	if got := (Edges[Pixels]{Top: 1, Right: 2, Bottom: 3, Left: 4}).MapOf(label); got != (EdgesOf[string]{Top: "1", Right: "2", Bottom: "3", Left: "4"}) {
		t.Fatalf("edges map of = %v", got)
	}
	length := func(s string) Pixels { return Pixels(len(s)) }
	if got := (SizeOf[string]{Width: "a", Height: "bb"}).MapNumber(length); got != (Size[Pixels]{Width: 1, Height: 2}) {
		t.Fatalf("size map number = %v", got)
	}
	if got := (PointOf[string]{X: "a", Y: "bb"}).MapNumber(length); got != (Point[Pixels]{X: 1, Y: 2}) {
		t.Fatalf("point map number = %v", got)
	}
	if got := (EdgesOf[string]{Top: "a", Right: "bb", Bottom: "ccc", Left: ""}).MapNumber(length); got != (Edges[Pixels]{Top: 1, Right: 2, Bottom: 3}) {
		t.Fatalf("edges map number = %v", got)
	}
}

func TestGeometry_ComparisonsDoNotAllocate(t *testing.T) {
	b := Bounds[Pixels]{Size: Size[Pixels]{Width: 10, Height: 10}}
	p := Point[Pixels]{X: 5, Y: 5}
	q := Point[ScaledPixels]{X: 1, Y: 2}
	var sink bool
	allocs := testing.AllocsPerRun(100, func() {
		sink = b.Contains(p) && b.Intersects(b)
		p = p.Max(Point[Pixels]{X: 1, Y: 9})
		q = q.Min(Point[ScaledPixels]{X: 0, Y: 3})
	})
	if allocs != 0 || !sink {
		t.Fatalf("allocs = %v, sink = %v", allocs, sink)
	}
}
