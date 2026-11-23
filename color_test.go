package gpui

import (
	"encoding/json/v2"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"unsafe"
)

var (
	_ IntoHsla = Hsla{}
	_ IntoHsla = Rgba{}
)

func unmarshalRgba(t *testing.T, s string) Rgba {
	t.Helper()
	var actual Rgba
	if err := json.Unmarshal([]byte(s), &actual); err != nil {
		t.Fatal(err)
	}
	return actual
}

func TestColor_DeserializeThreeValueHexToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `"#f09"`); actual != NewRgba(0xff0099ff) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_DeserializeFourValueHexToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `"#f09f"`); actual != NewRgba(0xff0099ff) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_DeserializeSixValueHexToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `"#ff0099"`); actual != NewRgba(0xff0099ff) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_DeserializeEightValueHexToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `"#ff0099ff"`); actual != NewRgba(0xff0099ff) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_DeserializeEightValueHexWithPaddingToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `" #f5f5f5ff   "`); actual != NewRgba(0xf5f5f5ff) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_DeserializeEightValueHexWithMixedCaseToRgba(t *testing.T) {
	if actual := unmarshalRgba(t, `"#DeAdbEeF"`); actual != NewRgba(0xdeadbeef) {
		t.Fatalf("got %v", actual)
	}
}

func TestColor_BackgroundSolid(t *testing.T) {
	color := NewRgba(0xff0099ff).ToHsla()
	background := color.ToBackground()
	if background.tag != BackgroundTagSolid {
		t.Fatalf("tag = %v", background.tag)
	}
	if !background.solid.Eq(color) {
		t.Fatalf("solid = %#v", background.solid)
	}
	if !background.Opacity(0.5).solid.Eq(color.Opacity(0.5)) {
		t.Fatal("opacity")
	}
	if background.IsTransparent() {
		t.Fatal("solid background is transparent")
	}
	background.solid = NewHsla(0.0, 0.0, 0.0, 0.0)
	if !background.IsTransparent() {
		t.Fatal("transparent background is not transparent")
	}
}

func TestColor_BackgroundLinearGradient(t *testing.T) {
	from := NewLinearColorStop(NewRgba(0xff0099ff), 0.0)
	to := NewLinearColorStop(NewRgba(0x00ff99ff), 1.0)
	background := LinearGradient(90.0, from, to)
	if background.tag != BackgroundTagLinearGradient {
		t.Fatalf("tag = %v", background.tag)
	}
	if !background.colors[0].Eq(from) || !background.colors[1].Eq(to) {
		t.Fatal("colors")
	}
	if !background.Opacity(0.5).colors[0].Eq(from.Opacity(0.5)) || !background.Opacity(0.5).colors[1].Eq(to.Opacity(0.5)) {
		t.Fatal("opacity")
	}
	if background.IsTransparent() {
		t.Fatal("gradient is transparent")
	}
	if !background.Opacity(0.0).IsTransparent() {
		t.Fatal("zero opacity gradient is not transparent")
	}
}

func TestColor_RgbaAlpha(t *testing.T) {
	color := Rgba{R: 0.2, G: 0.6, B: 1.0, A: 0.8}
	if color.Alpha(0.25).A != 0.25 {
		t.Fatal("alpha 0.25")
	}
	if color.Alpha(1.5).A != 1.0 {
		t.Fatal("alpha 1.5")
	}
}

func TestColor_RgbaOpacity(t *testing.T) {
	color := Rgba{R: 0.2, G: 0.6, B: 1.0, A: 0.8}
	if math.Abs(float64(color.Opacity(0.5).A-0.4)) >= 1e-6 {
		t.Fatal("opacity 0.5")
	}
	if color.Opacity(2.0).A != 0.8 {
		t.Fatal("opacity 2.0")
	}
}

func TestColor_Conversions(t *testing.T) {
	if got := Rgb(0xff0099); got != NewRgba(0xff0099ff) {
		t.Fatalf("rgb = %v", got)
	}
	for _, hex := range []uint32{0xff0099ff, 0x00000000, 0xffffffff, 0x123456ff, 0xdeadbeef} {
		c := NewRgba(hex)
		if c.ToU32() != hex {
			t.Errorf("%08x round trip = %08x", hex, c.ToU32())
		}
		back := c.ToHsla().ToRgb()
		for _, d := range []float32{back.R - c.R, back.G - c.G, back.B - c.B, back.A - c.A} {
			if math.Abs(float64(d)) > 1e-6 {
				t.Errorf("%08x hsla round trip = %v", hex, back)
			}
		}
	}
	if got := Red().ToRgb(); got != (Rgba{R: 1, G: 0, B: 0, A: 1}) {
		t.Errorf("red = %v", got)
	}
	if got := fmt.Sprint(NewRgba(0xff0099ff)); got != "rgba(0xff0099ff)" {
		t.Errorf("rgba debug = %q", got)
	}
	if got := fmt.Sprint(NewRgba(0x000000ff)); got != "rgba(0x000000ff)" {
		t.Errorf("rgba debug = %q", got)
	}
	if got := fmt.Sprint(Red()); got != "hsla(0.00, 100.00%, 50.00%, 1.00)" {
		t.Errorf("hsla display = %q", got)
	}
	if got := fmt.Sprintf("%#v", Black()); got != "Hsla { h: 0.0, s: 0.0, l: 0.0, a: 1.0 }" {
		t.Errorf("hsla debug = %q", got)
	}
	if got := fmt.Sprint(Black().ToBackground()); got != "Solid(Hsla { h: 0.0, s: 0.0, l: 0.0, a: 1.0 })" {
		t.Errorf("background debug = %q", got)
	}
	if got := fmt.Sprint(ColorSpaceOklab); got != "Oklab" {
		t.Errorf("color space = %q", got)
	}
}

func TestColor_Parse(t *testing.T) {
	for _, bad := range []string{"", "#", "f09", "a#f09", "#ff", "#ggg", "#fffff", "#é00"} {
		if _, err := ParseRgba(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := ParseRgba("ff0099"); err == nil || err.Error() != "invalid RGBA hex color: 'ff0099'. Expected #rgb, #rgba, #rrggbb, or #rrggbbaa" {
		t.Errorf("error = %v", err)
	}
	var c Rgba
	if err := json.Unmarshal([]byte(`5`), &c); err == nil {
		t.Error("number accepted as color")
	}
}

func TestColor_Serde(t *testing.T) {
	data, err := json.Marshal(NewRgba(0xff0099ff))
	if err != nil || string(data) != `"#ff0099ff"` {
		t.Fatalf("rgba json = %s, %v", data, err)
	}
	data, err = json.Marshal(Red())
	if err != nil || string(data) != `"#ff0000ff"` {
		t.Fatalf("hsla json = %s, %v", data, err)
	}
	var h Hsla
	if err := json.Unmarshal(data, &h); err != nil || !h.Eq(Red()) {
		t.Fatalf("hsla round trip = %#v, %v", h, err)
	}
	bg := LinearGradient(45, NewLinearColorStop(Red(), 0), NewLinearColorStop(Blue(), 1)).ColorSpace(ColorSpaceOklab)
	data, err = json.Marshal(bg)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"tag":"LinearGradient","color_space":"Oklab","solid":"#00000000","gradient_angle_or_pattern_height":45,"colors":[{"color":"#ff0000ff","percentage":0},{"color":"#0000ffff","percentage":1}],"pad":0}`
	if string(data) != want {
		t.Fatalf("background json = %s", data)
	}
	var back Background
	if err := json.Unmarshal(data, &back); err != nil || back.tag != BackgroundTagLinearGradient || back.colorSpace != ColorSpaceOklab {
		t.Fatalf("background round trip = %v, %v", back, err)
	}
}

func TestColor_Patterns(t *testing.T) {
	p := PatternSlash(Red(), 1, 2)
	if p.tag != BackgroundTagPatternSlash || p.gradientAngleOrPatternHeight != float32(255*0xFFFF+510) {
		t.Fatalf("pattern = %v", p)
	}
	if _, ok := p.AsSolid(); ok {
		t.Fatal("pattern is solid")
	}
	if c, ok := SolidBackground(Rgb(0xff0000)).AsSolid(); !ok || !c.Eq(Red()) {
		t.Fatalf("solid = %v %v", c, ok)
	}
	if Checkerboard(Red(), 4).tag != BackgroundTagCheckerboard {
		t.Fatal("checkerboard")
	}
	px := []byte{50, 100, 200, 128}
	SwapRgbaPaToBgra(px)
	if px[0] != 255 || px[2] != 99 {
		t.Fatalf("swap = %v", px)
	}
	h := White()
	h.FadeOut(0.25)
	if h.A != 0.75 || !White().Blend(TransparentBlack()).Eq(White()) || NewHsla(2, -1, 0.5, 1) != (Hsla{H: 1, S: 0, L: 0.5, A: 1}) {
		t.Fatal("hsla helpers")
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		if c := HslaOpaqueStrategy(rng); c.H < 0 || c.H > 1 || c.A != 1 {
			t.Fatalf("strategy = %#v", c)
		}
	}
	if unsafe.Sizeof(Background{}) != 72 || unsafe.Offsetof(Background{}.solid) != 8 || unsafe.Offsetof(Background{}.pad) != 68 {
		t.Fatalf("background layout = %d", unsafe.Sizeof(Background{}))
	}
}
