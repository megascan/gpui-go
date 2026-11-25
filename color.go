package gpui

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"unicode/utf8"

	"github.com/megascan/gpui-go/jsonschema"
)

// Rgb converts a 0xRRGGBB hex value to an opaque Rgba.
func Rgb(hex uint32) Rgba {
	return Rgba{
		R: float32(uint8(hex>>16)) / 255.0,
		G: float32(uint8(hex>>8)) / 255.0,
		B: float32(uint8(hex)) / 255.0,
		A: 1.0,
	}
}

// NewRgba converts a 0xRRGGBBAA hex value to an Rgba.
func NewRgba(hex uint32) Rgba {
	return Rgba{
		R: float32(uint8(hex>>24)) / 255.0,
		G: float32(uint8(hex>>16)) / 255.0,
		B: float32(uint8(hex>>8)) / 255.0,
		A: float32(uint8(hex)) / 255.0,
	}
}

// SwapRgbaPaToBgra swaps a premultiplied RGBA pixel to straight BGRA in place.
func SwapRgbaPaToBgra(color []byte) {
	color[0], color[2] = color[2], color[0]
	if color[3] > 0 {
		a := float32(color[3]) / 255.
		color[0] = saturatingF32ToU8(float32(color[0]) / a)
		color[1] = saturatingF32ToU8(float32(color[1]) / a)
		color[2] = saturatingF32ToU8(float32(color[2]) / a)
	}
}

// Rgba is a color in the sRGB color space with straight alpha.
type Rgba struct {
	R float32
	G float32
	B float32
	A float32
}

// String formats the color like its Rust Debug output, for example `rgba(0xff0099ff)`.
func (c Rgba) String() string {
	return fmt.Sprintf("rgba(0x%08x)", c.ToU32())
}

// Blend composites other over c.
func (c Rgba) Blend(other Rgba) Rgba {
	if other.A >= 1.0 {
		return other
	} else if other.A <= 0.0 {
		return c
	}
	return Rgba{
		R: float32(c.R*(1.0-other.A)) + float32(other.R*other.A),
		G: float32(c.G*(1.0-other.A)) + float32(other.G*other.A),
		B: float32(c.B*(1.0-other.A)) + float32(other.B*other.A),
		A: c.A,
	}
}

// Alpha returns c with its alpha replaced by a, clamped to 0..=1.
func (c Rgba) Alpha(a float32) Rgba {
	return Rgba{R: c.R, G: c.G, B: c.B, A: clampF32(a, 0., 1.)}
}

// Opacity returns c with its alpha multiplied by factor, clamped to 0..=1.
func (c Rgba) Opacity(factor float32) Rgba {
	return Rgba{R: c.R, G: c.G, B: c.B, A: c.A * clampF32(factor, 0., 1.)}
}

// ToU32 packs the color as 0xRRGGBBAA.
func (c Rgba) ToU32() uint32 {
	r := saturatingF32ToU32(c.R * 255.0)
	g := saturatingF32ToU32(c.G * 255.0)
	b := saturatingF32ToU32(c.B * 255.0)
	a := saturatingF32ToU32(c.A * 255.0)
	return (r << 24) | (g << 16) | (b << 8) | a
}

// ToHsla converts the color to Hsla.
func (c Rgba) ToHsla() Hsla {
	r := c.R
	g := c.G
	b := c.B

	maxValue := maxF32(r, maxF32(g, b))
	minValue := minF32(r, minF32(g, b))
	delta := maxValue - minValue

	l := (maxValue + minValue) / 2.0
	var s float32
	if l == 0.0 || l == 1.0 {
		s = 0.0
	} else if l < 0.5 {
		s = delta / (2.0 * l)
	} else {
		s = delta / (2.0 - float32(2.0*l))
	}

	var h float32
	if delta == 0.0 {
		h = 0.0
	} else if maxValue == r {
		h = remEuclidF32((g-b)/delta, 6.0) / 6.0
	} else if maxValue == g {
		h = ((b-r)/delta + 2.0) / 6.0
	} else {
		h = ((r-g)/delta + 4.0) / 6.0
	}

	return Hsla{H: h, S: s, L: l, A: c.A}
}

// ToBackground converts the color to a solid Background.
func (c Rgba) ToBackground() Background {
	return Background{tag: BackgroundTagSolid, solid: c.ToHsla()}
}

// MarshalJSONTo encodes the color as `#rrggbbaa`.
func (c Rgba) MarshalJSONTo(enc *jsontext.Encoder) error {
	r := saturatingF32ToU8(float32(math.Round(float64(c.R * 255.0))))
	g := saturatingF32ToU8(float32(math.Round(float64(c.G * 255.0))))
	b := saturatingF32ToU8(float32(math.Round(float64(c.B * 255.0))))
	a := saturatingF32ToU8(float32(math.Round(float64(c.A * 255.0))))
	return enc.WriteToken(jsontext.String(fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, a)))
}

// UnmarshalJSONFrom decodes the color from a hex string.
func (c *Rgba) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	s, err := readJSONString(dec, "a string in the format #rrggbb or #rrggbbaa")
	if err != nil {
		return err
	}
	v, err := ParseRgba(s)
	if err != nil {
		return err
	}
	*c = v
	return nil
}

// JSONSchema describes the color as a hex string.
func (Rgba) JSONSchema(*jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}}
}

// ParseRgba parses `#rgb`, `#rgba`, `#rrggbb` or `#rrggbbaa`, ignoring surrounding whitespace.
func ParseRgba(value string) (Rgba, error) {
	const expectedFormats = "Expected #rgb, #rgba, #rrggbb, or #rrggbbaa"
	const invalidUnicode = "invalid unicode characters in color"

	before, hex, found := strings.Cut(strings.TrimSpace(value), "#")
	if !found || before != "" {
		return Rgba{}, fmt.Errorf("invalid RGBA hex color: '%s'. %s", value, expectedFormats)
	}

	component := func(start, end int, context string) (uint8, error) {
		part, ok := strSlice(hex, start, end)
		if !ok {
			return 0, errors.New(context)
		}
		return parseU8Hex(part)
	}

	var r, g, b, a uint8
	var err error
	switch len(hex) {
	case 3, 4:
		if r, err = component(0, 1, fmt.Sprintf("%s: r component of #rgb/#rgba for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if g, err = component(1, 2, fmt.Sprintf("%s: g component of #rgb/#rgba for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if b, err = component(2, 3, fmt.Sprintf("%s: b component of #rgb/#rgba for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if len(hex) == 4 {
			if a, err = component(3, 4, fmt.Sprintf("%s: a component of #rgba for value: '%s'", invalidUnicode, value)); err != nil {
				return Rgba{}, err
			}
		} else {
			a = 0xf
		}
		duplicate := func(v uint8) uint8 { return (v << 4) | v }
		r, g, b, a = duplicate(r), duplicate(g), duplicate(b), duplicate(a)
	case 6, 8:
		if r, err = component(0, 2, fmt.Sprintf("%s: r component of #rrggbb/#rrggbbaa for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if g, err = component(2, 4, fmt.Sprintf("%s: g component of #rrggbb/#rrggbbaa for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if b, err = component(4, 6, fmt.Sprintf("%s: b component of #rrggbb/#rrggbbaa for value: '%s'", invalidUnicode, value)); err != nil {
			return Rgba{}, err
		}
		if len(hex) == 8 {
			if a, err = component(6, 8, fmt.Sprintf("%s: a component of #rrggbbaa for value: '%s'", invalidUnicode, value)); err != nil {
				return Rgba{}, err
			}
		} else {
			a = 0xff
		}
	default:
		return Rgba{}, fmt.Errorf("invalid RGBA hex color: '%s'. %s", value, expectedFormats)
	}

	return Rgba{
		R: float32(r) / 255.,
		G: float32(g) / 255.,
		B: float32(b) / 255.,
		A: float32(a) / 255.,
	}, nil
}

// Hsla is a color in the HSLA color space, every component in 0..=1.
type Hsla struct {
	H float32
	S float32
	L float32
	A float32
}

// HslaOpaqueStrategy generates an opaque Hsla with components in 0..=1 (D-PROPTEST).
func HslaOpaqueStrategy(rng *rand.Rand) Hsla {
	return Hsla{H: unitF32(rng), S: unitF32(rng), L: unitF32(rng), A: 1.}
}

// HslaArbitrary generates an Hsla with every component in 0..=1 (D-PROPTEST).
func HslaArbitrary(rng *rand.Rand) Hsla {
	return Hsla{H: unitF32(rng), S: unitF32(rng), L: unitF32(rng), A: unitF32(rng)}
}

// Cmp orders colors by IEEE total order of h, s, l, then a.
func (c Hsla) Cmp(other Hsla) int {
	return cmp.Or(
		totalCmpF32(c.H, other.H),
		totalCmpF32(c.S, other.S),
		totalCmpF32(c.L, other.L),
		totalCmpF32(c.A, other.A),
	)
}

// Eq reports whether the colors are equal under IEEE total order.
func (c Hsla) Eq(other Hsla) bool {
	return c.Cmp(other) == 0
}

// String formats the color like Rust's Display, for example `hsla(0.00, 100.00%, 50.00%, 1.00)`.
func (c Hsla) String() string {
	return fmt.Sprintf("hsla(%.2f, %.2f%%, %.2f%%, %.2f)", c.H*360., c.S*100., c.L*100., c.A)
}

// GoString formats the color like its Rust Debug output.
func (c Hsla) GoString() string {
	return "Hsla { h: " + rustDebugF32(c.H) + ", s: " + rustDebugF32(c.S) + ", l: " + rustDebugF32(c.L) + ", a: " + rustDebugF32(c.A) + " }"
}

// NewHsla constructs an Hsla with every component clamped to 0..=1.
func NewHsla(h, s, l, a float32) Hsla {
	return Hsla{H: clampF32(h, 0., 1.), S: clampF32(s, 0., 1.), L: clampF32(l, 0., 1.), A: clampF32(a, 0., 1.)}
}

// Black returns opaque black.
func Black() Hsla {
	return Hsla{H: 0., S: 0., L: 0., A: 1.}
}

// TransparentBlack returns fully transparent black.
func TransparentBlack() Hsla {
	return Hsla{H: 0., S: 0., L: 0., A: 0.}
}

// TransparentWhite returns fully transparent white.
func TransparentWhite() Hsla {
	return Hsla{H: 0., S: 0., L: 1., A: 0.}
}

// OpaqueGrey returns a grey of the given lightness and opacity, both clamped to 0..=1.
func OpaqueGrey(lightness, opacity float32) Hsla {
	return Hsla{H: 0., S: 0., L: clampF32(lightness, 0., 1.), A: clampF32(opacity, 0., 1.)}
}

// White returns opaque white.
func White() Hsla {
	return Hsla{H: 0., S: 0., L: 1., A: 1.}
}

// Red returns opaque red.
func Red() Hsla {
	return Hsla{H: 0., S: 1., L: 0.5, A: 1.}
}

// Blue returns opaque blue.
func Blue() Hsla {
	return Hsla{H: 0.6666666667, S: 1., L: 0.5, A: 1.}
}

// Green returns opaque green.
func Green() Hsla {
	return Hsla{H: 0.3333333333, S: 1., L: 0.25, A: 1.}
}

// Yellow returns opaque yellow.
func Yellow() Hsla {
	return Hsla{H: 0.1666666667, S: 1., L: 0.5, A: 1.}
}

// ToRgb converts the color to Rgba.
func (c Hsla) ToRgb() Rgba {
	h := c.H
	s := c.S
	l := c.L

	cc := (1.0 - absF32(float32(2.0*l)-1.0)) * s
	x := cc * (1.0 - absF32(remF32(h*6.0, 2.0)-1.0))
	m := l - cc/2.0
	cm := cc + m
	xm := x + m

	var r, g, b float32
	switch saturatingF32ToI32(float32(math.Floor(float64(h * 6.0)))) {
	case 0, 6:
		r, g, b = cm, xm, m
	case 1:
		r, g, b = xm, cm, m
	case 2:
		r, g, b = m, cm, xm
	case 3:
		r, g, b = m, xm, cm
	case 4:
		r, g, b = xm, m, cm
	default:
		r, g, b = cm, m, xm
	}

	return Rgba{R: clampF32(r, 0., 1.), G: clampF32(g, 0., 1.), B: clampF32(b, 0., 1.), A: c.A}
}

// ToHsla returns c.
func (c Hsla) ToHsla() Hsla {
	return c
}

// ToBackground converts the color to a solid Background.
func (c Hsla) ToBackground() Background {
	return Background{tag: BackgroundTagSolid, solid: c}
}

// HslaRed returns opaque red.
func HslaRed() Hsla {
	return Red()
}

// HslaGreen returns opaque green.
func HslaGreen() Hsla {
	return Green()
}

// HslaBlue returns opaque blue.
func HslaBlue() Hsla {
	return Blue()
}

// HslaBlack returns opaque black.
func HslaBlack() Hsla {
	return Black()
}

// HslaWhite returns opaque white.
func HslaWhite() Hsla {
	return White()
}

// HslaTransparentBlack returns fully transparent black.
func HslaTransparentBlack() Hsla {
	return TransparentBlack()
}

// IsTransparent reports whether the alpha is exactly 0.
func (c Hsla) IsTransparent() bool {
	return c.A == 0.0
}

// IsOpaque reports whether the alpha is exactly 1.
func (c Hsla) IsOpaque() bool {
	return c.A == 1.0
}

// Blend composites other over c in sRGB.
func (c Hsla) Blend(other Hsla) Hsla {
	alpha := other.A
	if alpha >= 1.0 {
		return other
	} else if alpha <= 0.0 {
		return c
	}
	return c.ToRgb().Blend(other.ToRgb()).ToHsla()
}

// Grayscale returns c with its saturation set to 0.
func (c Hsla) Grayscale() Hsla {
	return Hsla{H: c.H, S: 0., L: c.L, A: c.A}
}

// FadeOut multiplies the alpha by 1 - factor, factor clamped to 0..=1.
func (c *Hsla) FadeOut(factor float32) {
	c.A *= 1.0 - clampF32(factor, 0., 1.)
}

// Opacity returns c with its alpha multiplied by factor, clamped to 0..=1.
func (c Hsla) Opacity(factor float32) Hsla {
	return Hsla{H: c.H, S: c.S, L: c.L, A: c.A * clampF32(factor, 0., 1.)}
}

// Alpha returns c with its alpha replaced by a, clamped to 0..=1.
func (c Hsla) Alpha(a float32) Hsla {
	return Hsla{H: c.H, S: c.S, L: c.L, A: clampF32(a, 0., 1.)}
}

// MarshalJSONTo encodes the color as an Rgba hex string.
func (c Hsla) MarshalJSONTo(enc *jsontext.Encoder) error {
	return c.ToRgb().MarshalJSONTo(enc)
}

// UnmarshalJSONFrom decodes the color from an Rgba hex string.
func (c *Hsla) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var rgba Rgba
	if err := rgba.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	*c = rgba.ToHsla()
	return nil
}

// JSONSchema describes the color as Rgba does.
func (Hsla) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	return Rgba{}.JSONSchema(g)
}

// IntoHsla is implemented by every value convertible to an Hsla (D-INTO).
type IntoHsla interface {
	ToHsla() Hsla
}

// BackgroundTag identifies the kind of a Background; 4 bytes wide for the GPU layout.
type BackgroundTag uint32

const (
	// BackgroundTagSolid is a solid color.
	BackgroundTagSolid BackgroundTag = 0
	// BackgroundTagLinearGradient is a two-stop linear gradient.
	BackgroundTagLinearGradient BackgroundTag = 1
	// BackgroundTagPatternSlash is a slash pattern.
	BackgroundTagPatternSlash BackgroundTag = 2
	// BackgroundTagCheckerboard is a checkerboard pattern.
	BackgroundTagCheckerboard BackgroundTag = 3
)

var backgroundTagNames = [...]string{"Solid", "LinearGradient", "PatternSlash", "Checkerboard"}

// String formats the tag like its Rust Debug output.
func (t BackgroundTag) String() string {
	if int(t) < len(backgroundTagNames) {
		return backgroundTagNames[t]
	}
	return fmt.Sprintf("BackgroundTag(%d)", uint32(t))
}

// MarshalJSONTo encodes the tag as its variant name.
func (t BackgroundTag) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(t.String()))
}

// UnmarshalJSONFrom decodes the tag from its variant name.
func (t *BackgroundTag) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	i, err := readJSONVariant(dec, backgroundTagNames[:])
	if err != nil {
		return err
	}
	*t = BackgroundTag(i)
	return nil
}

// ColorSpace is the color space a gradient is interpolated in; 4 bytes wide for the GPU layout.
type ColorSpace uint32

const (
	// ColorSpaceSrgb interpolates in sRGB.
	ColorSpaceSrgb ColorSpace = 0
	// ColorSpaceOklab interpolates in Oklab.
	ColorSpaceOklab ColorSpace = 1
)

var colorSpaceNames = [...]string{"Srgb", "Oklab"}

// String formats the color space like Rust's Display: `sRGB` or `Oklab`.
func (c ColorSpace) String() string {
	switch c {
	case ColorSpaceSrgb:
		return "sRGB"
	case ColorSpaceOklab:
		return "Oklab"
	}
	return fmt.Sprintf("ColorSpace(%d)", uint32(c))
}

// GoString formats the color space like its Rust Debug output.
func (c ColorSpace) GoString() string {
	if int(c) < len(colorSpaceNames) {
		return colorSpaceNames[c]
	}
	return fmt.Sprintf("ColorSpace(%d)", uint32(c))
}

// MarshalJSONTo encodes the color space as its variant name.
func (c ColorSpace) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(c.GoString()))
}

// UnmarshalJSONFrom decodes the color space from its variant name.
func (c *ColorSpace) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	i, err := readJSONVariant(dec, colorSpaceNames[:])
	if err != nil {
		return err
	}
	*c = ColorSpace(i)
	return nil
}

// Background is the fill of a quad: a solid color, a gradient or a pattern.
// Its layout matches the GPU struct.
type Background struct {
	tag                          BackgroundTag
	colorSpace                   ColorSpace
	solid                        Hsla
	gradientAngleOrPatternHeight float32
	colors                       [2]LinearColorStop
	pad                          uint32
}

// String formats the background like its Rust Debug output.
func (b Background) String() string {
	switch b.tag {
	case BackgroundTagLinearGradient:
		return fmt.Sprintf("LinearGradient(%s, %#v, %#v)", rustDisplayF32(b.gradientAngleOrPatternHeight), b.colors[0], b.colors[1])
	case BackgroundTagPatternSlash:
		return fmt.Sprintf("PatternSlash(%#v, %s)", b.solid, rustDisplayF32(b.gradientAngleOrPatternHeight))
	case BackgroundTagCheckerboard:
		return fmt.Sprintf("Checkerboard(%#v, %s)", b.solid, rustDisplayF32(b.gradientAngleOrPatternHeight))
	}
	return fmt.Sprintf("Solid(%#v)", b.solid)
}

// Eq reports whether the backgrounds are equal field by field, colors compared as Hsla does.
func (b Background) Eq(other Background) bool {
	return b.tag == other.tag &&
		b.colorSpace == other.colorSpace &&
		b.solid.Eq(other.solid) &&
		b.gradientAngleOrPatternHeight == other.gradientAngleOrPatternHeight &&
		b.colors[0].Eq(other.colors[0]) &&
		b.colors[1].Eq(other.colors[1]) &&
		b.pad == other.pad
}

type backgroundJSON struct {
	Tag                          BackgroundTag      `json:"tag"`
	ColorSpace                   ColorSpace         `json:"color_space"`
	Solid                        Hsla               `json:"solid"`
	GradientAngleOrPatternHeight float32            `json:"gradient_angle_or_pattern_height"`
	Colors                       [2]LinearColorStop `json:"colors"`
	Pad                          uint32             `json:"pad"`
}

// MarshalJSONTo encodes the background with serde's field names.
func (b Background) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, backgroundJSON{
		Tag:                          b.tag,
		ColorSpace:                   b.colorSpace,
		Solid:                        b.solid,
		GradientAngleOrPatternHeight: b.gradientAngleOrPatternHeight,
		Colors:                       b.colors,
		Pad:                          b.pad,
	})
}

// UnmarshalJSONFrom decodes the background from serde's field names.
func (b *Background) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var v backgroundJSON
	if err := json.UnmarshalDecode(dec, &v); err != nil {
		return err
	}
	*b = Background{
		tag:                          v.Tag,
		colorSpace:                   v.ColorSpace,
		solid:                        v.Solid,
		gradientAngleOrPatternHeight: v.GradientAngleOrPatternHeight,
		colors:                       v.Colors,
		pad:                          v.Pad,
	}
	return nil
}

// PatternSlash returns a slash pattern of the given color, stripe width and interval.
func PatternSlash(color IntoHsla, width, interval float32) Background {
	widthScaled := saturatingF32ToU32(width * 255.0)
	intervalScaled := saturatingF32ToU32(interval * 255.0)
	height := float32((widthScaled * 0xFFFF) + intervalScaled)

	return Background{
		tag:                          BackgroundTagPatternSlash,
		solid:                        color.ToHsla(),
		gradientAngleOrPatternHeight: height,
	}
}

// Checkerboard returns a checkerboard pattern of the given color and square size.
func Checkerboard(color IntoHsla, size float32) Background {
	return Background{
		tag:                          BackgroundTagCheckerboard,
		solid:                        color.ToHsla(),
		gradientAngleOrPatternHeight: size,
	}
}

// SolidBackground returns a solid background of the given color.
func SolidBackground(color IntoHsla) Background {
	return Background{solid: color.ToHsla()}
}

// LinearGradient returns a linear gradient at angle degrees between two color stops.
func LinearGradient(angle float32, from, to LinearColorStop) Background {
	return Background{
		tag:                          BackgroundTagLinearGradient,
		gradientAngleOrPatternHeight: angle,
		colors:                       [2]LinearColorStop{from, to},
	}
}

// LinearColorStop is a color at a position, 0..=1, along a linear gradient.
type LinearColorStop struct {
	Color      Hsla    `json:"color"`
	Percentage float32 `json:"percentage"`
}

// NewLinearColorStop returns a color stop.
func NewLinearColorStop(color IntoHsla, percentage float32) LinearColorStop {
	return LinearColorStop{Color: color.ToHsla(), Percentage: percentage}
}

// Opacity returns the stop with its color's alpha multiplied by factor.
func (s LinearColorStop) Opacity(factor float32) LinearColorStop {
	return LinearColorStop{Percentage: s.Percentage, Color: s.Color.Opacity(factor)}
}

// Eq reports whether the stops are equal, colors compared as Hsla does.
func (s LinearColorStop) Eq(other LinearColorStop) bool {
	return s.Color.Eq(other.Color) && s.Percentage == other.Percentage
}

// GoString formats the stop like its Rust Debug output.
func (s LinearColorStop) GoString() string {
	return "LinearColorStop { color: " + s.Color.GoString() + ", percentage: " + rustDebugF32(s.Percentage) + " }"
}

// AsSolid returns the color when the background is solid.
func (b Background) AsSolid() (Hsla, bool) {
	if b.tag == BackgroundTagSolid {
		return b.solid, true
	}
	return Hsla{}, false
}

// ColorSpace returns b with its interpolation color space set.
func (b Background) ColorSpace(colorSpace ColorSpace) Background {
	b.colorSpace = colorSpace
	return b
}

// Opacity returns b with every color's alpha multiplied by factor.
func (b Background) Opacity(factor float32) Background {
	background := b
	background.solid = background.solid.Opacity(factor)
	background.colors = [2]LinearColorStop{
		b.colors[0].Opacity(factor),
		b.colors[1].Opacity(factor),
	}
	return background
}

// IsTransparent reports whether every color of the background is transparent.
func (b Background) IsTransparent() bool {
	switch b.tag {
	case BackgroundTagLinearGradient:
		return b.colors[0].Color.IsTransparent() && b.colors[1].Color.IsTransparent()
	}
	return b.solid.IsTransparent()
}

func readJSONVariant(dec *jsontext.Decoder, names []string) (int, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return 0, err
	}
	if tok.Kind() != '"' {
		return 0, fmt.Errorf("invalid type: %v, expected variant identifier", tok.Kind())
	}
	s := tok.String()
	for i, name := range names {
		if name == s {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown variant `%s`, expected one of `%s`", s, strings.Join(names, "`, `"))
}

func strSlice(s string, start, end int) (string, bool) {
	if end > len(s) || (start < len(s) && !utf8.RuneStart(s[start])) || (end < len(s) && !utf8.RuneStart(s[end])) {
		return "", false
	}
	return s[start:end], true
}

func parseU8Hex(s string) (uint8, error) {
	digits := s
	if len(digits) > 1 && digits[0] == '+' {
		digits = digits[1:]
	}
	if digits == "" {
		return 0, errors.New("cannot parse integer from empty string")
	}
	var v uint32
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		var d uint32
		switch {
		case c >= '0' && c <= '9':
			d = uint32(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint32(c-'A') + 10
		default:
			return 0, errors.New("invalid digit found in string")
		}
		v = v*16 + d
		if v > math.MaxUint8 {
			return 0, errors.New("number too large to fit in target type")
		}
	}
	return uint8(v), nil
}

func unitF32(rng *rand.Rand) float32 {
	return float32(rng.Uint32N(1<<24+1)) / (1 << 24)
}

func clampF32(v, lo, hi float32) float32 {
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}

func maxF32(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a > b {
		return a
	}
	return b
}

func minF32(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func absF32(v float32) float32 {
	return float32(math.Abs(float64(v)))
}

func remF32(a, b float32) float32 {
	return float32(math.Mod(float64(a), float64(b)))
}

func remEuclidF32(a, b float32) float32 {
	r := remF32(a, b)
	if r < 0.0 {
		return r + absF32(b)
	}
	return r
}
