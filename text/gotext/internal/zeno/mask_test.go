package zeno

import "testing"

func square(x0, y0, x1, y1 float32) PointsCommands {
	return PointsCommands{
		Points: []Point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}},
		Verbs:  []Verb{VerbMoveTo, VerbLineTo, VerbLineTo, VerbLineTo, VerbClose},
	}
}

func TestMask_AlphaSquare(t *testing.T) {
	buf, p := NewMask(square(0.5, 0.5, 2.5, 2.5)).Render()
	if p != (Placement{Left: 0, Top: 0, Width: 3, Height: 3}) {
		t.Fatalf("placement %+v", p)
	}
	want := []byte{
		64, 128, 64,
		128, 255, 128,
		64, 128, 64,
	}
	for i := range want {
		d := int(buf[i]) - int(want[i])
		if d < -1 || d > 1 {
			t.Fatalf("pixel %d = %d, want %d (%v)", i, buf[i], want[i], buf)
		}
	}
}

func TestMask_IntegerSquareIsOpaque(t *testing.T) {
	buf, p := NewMask(square(1, 1, 3, 4)).Render()
	if p.Width != 2 || p.Height != 3 || p.Left != 1 || p.Top != 1 {
		t.Fatalf("placement %+v", p)
	}
	for i, b := range buf {
		if b != 255 {
			t.Fatalf("pixel %d = %d", i, b)
		}
	}
}

func TestMask_EvenOddHole(t *testing.T) {
	path := PointsCommands{
		Points: []Point{{0, 0}, {3, 0}, {3, 3}, {0, 3}, {1, 1}, {2, 1}, {2, 2}, {1, 2}},
		Verbs:  []Verb{VerbMoveTo, VerbLineTo, VerbLineTo, VerbLineTo, VerbClose, VerbMoveTo, VerbLineTo, VerbLineTo, VerbLineTo, VerbClose},
	}
	buf, _ := NewMask(path).Style(FillEvenOdd).Render()
	if buf[4] != 0 || buf[0] != 255 {
		t.Fatalf("even-odd %v", buf)
	}
	buf, _ = NewMask(path).Render()
	if buf[4] != 255 {
		t.Fatalf("non-zero %v", buf)
	}
}

func TestMask_BottomLeftOriginFlipsRows(t *testing.T) {
	var data []byte
	m := MaskWithScratch(square(0, 0, 2, 1), NewScratch()).Origin(OriginBottomLeft)
	m.Inspect(func(f Format, w, h uint32) {
		if w != 2 || h != 1 {
			t.Fatalf("size %dx%d", w, h)
		}
		data = make([]byte, f.BufferSize(w, h))
	})
	p := m.RenderInto(data, 0)
	if p.Top != 1 || p.Left != 0 {
		t.Fatalf("placement %+v", p)
	}
	rect := PointsCommands{
		Points: []Point{{0, 0}, {1, 0}, {1, 2}, {0, 2}},
		Verbs:  []Verb{VerbMoveTo, VerbLineTo, VerbLineTo, VerbLineTo, VerbClose},
	}
	up := NewMask(rect).Origin(OriginBottomLeft).Offset(NewVector(0, 0.5)).RenderOffset(NewVector(0, 0.5))
	var buf []byte
	up.Inspect(func(f Format, w, h uint32) { buf = make([]byte, f.BufferSize(w, h)) })
	p = up.RenderInto(buf, 0)
	if p.Height != 3 || p.Top != 3 {
		t.Fatalf("placement %+v", p)
	}
	if buf[0] < 120 || buf[0] > 136 || buf[1] != 255 || buf[2] < 120 || buf[2] > 136 {
		t.Fatalf("rows %v", buf)
	}
}

func TestMask_CustomSubpixelChannels(t *testing.T) {
	m := NewMask(square(1, 0, 2, 1)).Format(FormatSubpixelBGRA())
	buf, p := m.Render()
	if len(buf) != int(p.Width*p.Height*4) || p.Width != 1 {
		t.Fatalf("len %d placement %+v", len(buf), p)
	}
	wide, wp := NewMask(square(1, 0, 2, 1)).Format(FormatSubpixelBGRA()).Size(3, 1).Render()
	if wp.Width != 3 {
		t.Fatalf("placement %+v", wp)
	}
	px := func(x, c int) int { return int(wide[x*4+c]) }
	if px(1, 1) != 255 || px(0, 1) != 0 || px(2, 1) != 0 {
		t.Fatalf("green %v", wide)
	}
	if px(1, 0) < 175 || px(1, 0) > 180 || px(2, 0) < 75 || px(2, 0) > 80 || px(0, 0) != 0 {
		t.Fatalf("channel 0 %v", wide)
	}
	if px(1, 2) < 175 || px(1, 2) > 180 || px(0, 2) < 75 || px(0, 2) > 80 || px(2, 2) != 0 {
		t.Fatalf("channel 2 %v", wide)
	}
	for x := range 3 {
		if px(x, 3) != 0 {
			t.Fatalf("channel 3 %v", wide)
		}
	}
}

func TestMask_CurvesStayInsideControlBounds(t *testing.T) {
	path := PointsCommands{
		Points: []Point{{0, 0}, {4, 8}, {8, 0}, {6, -2}, {2, -2}, {0, 0}},
		Verbs:  []Verb{VerbMoveTo, VerbQuadTo, VerbCurveTo, VerbClose},
	}
	buf, p := NewMask(path).Render()
	if p.Width != 8 || p.Height != 10 || p.Top != -2 {
		t.Fatalf("placement %+v", p)
	}
	var sum int
	for _, b := range buf {
		sum += int(b)
	}
	if sum == 0 {
		t.Fatal("empty coverage")
	}
}
