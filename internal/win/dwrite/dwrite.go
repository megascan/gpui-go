//go:build windows

// Package dwrite binds the DirectWrite interfaces gpui's Windows text system uses: the factory,
// font collections and sets, font faces, text formats and layouts, glyph run analysis, colour
// glyph runs, rendering params, and a custom text renderer implemented in Go.
package dwrite

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

// Interface ids.
var (
	IID_IDWriteFactory5                 = com.MustGUID("{958DB99A-BE2A-4F09-AF7D-65189803D1D3}")
	IID_IDWriteTextFormat1              = com.MustGUID("{5F174B49-0D8B-4CFB-8BCA-F1CCE9D06C67}")
	IID_IDWriteFontFace3                = com.MustGUID("{D37D7598-09BE-4222-A236-2081341CC1F2}")
	IID_IDWriteRenderingParams1         = com.MustGUID("{94413CF4-A6FC-4248-8B50-6674348FCAD3}")
	IID_IDWritePixelSnapping            = com.MustGUID("{EAF3A2DA-ECF4-4D24-B644-B34F6842024B}")
	IID_IDWriteTextRenderer             = com.MustGUID("{EF8A8135-5CC6-45FE-8825-C5A0724EB819}")
	IID_IDWriteColorGlyphRunEnumerator1 = com.MustGUID("{7C5F86DA-C7A1-4F05-B8E1-55A179FE5A35}")
)

// DirectWrite HRESULTs.
const (
	DWRITE_E_NOFONT               com.HRESULT = -0x7767AFFE
	DWRITE_E_UNSUPPORTEDOPERATION com.HRESULT = -0x7767AFF7
	DWRITE_E_NOCOLOR              com.HRESULT = -0x7767AFF4
	E_NOT_SUFFICIENT_BUFFER       com.HRESULT = -0x7FF8FF86
)

// Enumerations, as their C values.
const (
	DWRITE_FACTORY_TYPE_SHARED uint32 = 0

	DWRITE_FONT_WEIGHT_NORMAL  int32 = 400
	DWRITE_FONT_STRETCH_NORMAL int32 = 5

	DWRITE_FONT_STYLE_NORMAL  int32 = 0
	DWRITE_FONT_STYLE_OBLIQUE int32 = 1
	DWRITE_FONT_STYLE_ITALIC  int32 = 2

	DWRITE_MEASURING_MODE_NATURAL int32 = 0

	DWRITE_RENDERING_MODE1_DEFAULT                       int32 = 0
	DWRITE_RENDERING_MODE1_ALIASED                       int32 = 1
	DWRITE_RENDERING_MODE1_GDI_CLASSIC                   int32 = 2
	DWRITE_RENDERING_MODE1_GDI_NATURAL                   int32 = 3
	DWRITE_RENDERING_MODE1_NATURAL                       int32 = 4
	DWRITE_RENDERING_MODE1_NATURAL_SYMMETRIC             int32 = 5
	DWRITE_RENDERING_MODE1_OUTLINE                       int32 = 6
	DWRITE_RENDERING_MODE1_NATURAL_SYMMETRIC_DOWNSAMPLED int32 = 7

	DWRITE_GRID_FIT_MODE_DEFAULT int32 = 0

	DWRITE_OUTLINE_THRESHOLD_ANTIALIASED int32 = 0

	DWRITE_TEXT_ANTIALIAS_MODE_CLEARTYPE int32 = 0
	DWRITE_TEXT_ANTIALIAS_MODE_GRAYSCALE int32 = 1

	DWRITE_TEXTURE_ALIASED_1x1   int32 = 0
	DWRITE_TEXTURE_CLEARTYPE_3x1 int32 = 1

	DWRITE_PIXEL_GEOMETRY_FLAT int32 = 0
	DWRITE_PIXEL_GEOMETRY_RGB  int32 = 1
	DWRITE_PIXEL_GEOMETRY_BGR  int32 = 2

	DWRITE_GLYPH_IMAGE_FORMATS_NONE                   int32 = 0
	DWRITE_GLYPH_IMAGE_FORMATS_TRUETYPE               int32 = 1
	DWRITE_GLYPH_IMAGE_FORMATS_CFF                    int32 = 2
	DWRITE_GLYPH_IMAGE_FORMATS_COLR                   int32 = 4
	DWRITE_GLYPH_IMAGE_FORMATS_SVG                    int32 = 8
	DWRITE_GLYPH_IMAGE_FORMATS_PNG                    int32 = 16
	DWRITE_GLYPH_IMAGE_FORMATS_JPEG                   int32 = 32
	DWRITE_GLYPH_IMAGE_FORMATS_TIFF                   int32 = 64
	DWRITE_GLYPH_IMAGE_FORMATS_PREMULTIPLIED_B8G8R8A8 int32 = 128
)

// BOOL is a Win32 BOOL.
type BOOL int32

// AsBool reports whether b is nonzero.
func (b BOOL) AsBool() bool {
	return b != 0
}

// BoolOf converts v to a BOOL.
func BoolOf(v bool) BOOL {
	if v {
		return 1
	}
	return 0
}

// DWRITE_TEXT_RANGE is a range of UTF-16 code units.
type DWRITE_TEXT_RANGE struct {
	StartPosition uint32
	Length        uint32
}

func (r DWRITE_TEXT_RANGE) arg() uintptr {
	return uintptr(uint64(r.StartPosition) | uint64(r.Length)<<32)
}

// DWRITE_FONT_FEATURE is an OpenType feature tag and parameter.
type DWRITE_FONT_FEATURE struct {
	NameTag   uint32
	Parameter uint32
}

// DWRITE_GLYPH_OFFSET is a glyph's offset from its advance position.
type DWRITE_GLYPH_OFFSET struct {
	AdvanceOffset  float32
	AscenderOffset float32
}

// DWRITE_GLYPH_RUN is a run of glyphs in one font face.
type DWRITE_GLYPH_RUN struct {
	FontFace      *IDWriteFontFace
	FontEmSize    float32
	GlyphCount    uint32
	GlyphIndices  *uint16
	GlyphAdvances *float32
	GlyphOffsets  *DWRITE_GLYPH_OFFSET
	IsSideways    BOOL
	BidiLevel     uint32
}

// DWRITE_GLYPH_RUN_DESCRIPTION describes the text a glyph run was shaped from.
type DWRITE_GLYPH_RUN_DESCRIPTION struct {
	LocaleName   *uint16
	String       *uint16
	StringLength uint32
	ClusterMap   *uint16
	TextPosition uint32
}

// DWRITE_COLOR_F is a linear RGBA colour.
type DWRITE_COLOR_F struct {
	R, G, B, A float32
}

// DWRITE_COLOR_GLYPH_RUN is one layer of a colour glyph.
type DWRITE_COLOR_GLYPH_RUN struct {
	GlyphRun            DWRITE_GLYPH_RUN
	GlyphRunDescription *DWRITE_GLYPH_RUN_DESCRIPTION
	BaselineOriginX     float32
	BaselineOriginY     float32
	RunColor            DWRITE_COLOR_F
	PaletteIndex        uint16
}

// DWRITE_COLOR_GLYPH_RUN1 is a colour glyph layer with its image format.
type DWRITE_COLOR_GLYPH_RUN1 struct {
	Base             DWRITE_COLOR_GLYPH_RUN
	GlyphImageFormat int32
	MeasuringMode    int32
}

// DWRITE_LINE_METRICS are the metrics of one line of a text layout.
type DWRITE_LINE_METRICS struct {
	Length                   uint32
	TrailingWhitespaceLength uint32
	NewlineLength            uint32
	Height                   float32
	Baseline                 float32
	IsTrimmed                BOOL
}

// DWRITE_TEXT_METRICS are the overall metrics of a text layout.
type DWRITE_TEXT_METRICS struct {
	Left                             float32
	Top                              float32
	Width                            float32
	WidthIncludingTrailingWhitespace float32
	Height                           float32
	LayoutWidth                      float32
	LayoutHeight                     float32
	MaxBidiReorderingDepth           uint32
	LineCount                        uint32
}

// DWRITE_HIT_TEST_METRICS describe a hit-tested range of text.
type DWRITE_HIT_TEST_METRICS struct {
	TextPosition uint32
	Length       uint32
	Left         float32
	Top          float32
	Width        float32
	Height       float32
	BidiLevel    uint32
	IsText       BOOL
	IsTrimmed    BOOL
}

// DWRITE_FONT_METRICS are a font face's design metrics.
type DWRITE_FONT_METRICS struct {
	DesignUnitsPerEm       uint16
	Ascent                 uint16
	Descent                uint16
	LineGap                int16
	CapHeight              uint16
	XHeight                uint16
	UnderlinePosition      int16
	UnderlineThickness     uint16
	StrikethroughPosition  int16
	StrikethroughThickness uint16
}

// DWRITE_FONT_METRICS1 adds the glyph box and script metrics.
type DWRITE_FONT_METRICS1 struct {
	Base                  DWRITE_FONT_METRICS
	GlyphBoxLeft          int16
	GlyphBoxTop           int16
	GlyphBoxRight         int16
	GlyphBoxBottom        int16
	SubscriptPositionX    int16
	SubscriptPositionY    int16
	SubscriptSizeX        int16
	SubscriptSizeY        int16
	SuperscriptPositionX  int16
	SuperscriptPositionY  int16
	SuperscriptSizeX      int16
	SuperscriptSizeY      int16
	HasTypographicMetrics BOOL
}

// DWRITE_GLYPH_METRICS are one glyph's design metrics.
type DWRITE_GLYPH_METRICS struct {
	LeftSideBearing   int32
	AdvanceWidth      uint32
	RightSideBearing  int32
	TopSideBearing    int32
	AdvanceHeight     uint32
	BottomSideBearing int32
	VerticalOriginY   int32
}

// DWRITE_UNICODE_RANGE is an inclusive range of code points.
type DWRITE_UNICODE_RANGE struct {
	First uint32
	Last  uint32
}

// DWRITE_MATRIX is a 2D affine transform.
type DWRITE_MATRIX struct {
	M11, M12, M21, M22, Dx, Dy float32
}

// Vector2 is a 2D point.
type Vector2 struct {
	X, Y float32
}

// RECT is a rectangle by its edges.
type RECT struct {
	Left, Top, Right, Bottom int32
}

// Error is an HRESULT with a message, as windows-rs's Error::new.
type Error struct {
	Code    com.HRESULT
	Message string
}

// NewError returns an Error.
func NewError(code com.HRESULT, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Error returns the message followed by the code.
func (e *Error) Error() string {
	return e.Message + " (" + e.Code.Error() + ")"
}

// Unwrap returns the HRESULT.
func (e *Error) Unwrap() error {
	return e.Code
}

// HRESULTOf maps err to the HRESULT a COM method returns.
func HRESULTOf(err error) com.HRESULT {
	if err == nil {
		return com.S_OK
	}
	var hr com.HRESULT
	if errors.As(err, &hr) {
		return hr
	}
	return com.HRESULTFromError(err)
}

// IsHRESULT reports whether err carries the HRESULT code.
func IsHRESULT(err error, code com.HRESULT) bool {
	var hr com.HRESULT
	return errors.As(err, &hr) && hr == code
}

var (
	modDwrite               = windows.NewLazySystemDLL("dwrite.dll")
	procDWriteCreateFactory = modDwrite.NewProc("DWriteCreateFactory")
)

// DWriteCreateFactory creates the shared IDWriteFactory5.
func DWriteCreateFactory(factoryType uint32) (*IDWriteFactory5, error) {
	if err := procDWriteCreateFactory.Find(); err != nil {
		return nil, err
	}
	var out *IDWriteFactory5
	r, _, _ := procDWriteCreateFactory.Call(uintptr(factoryType), uintptr(unsafe.Pointer(&IID_IDWriteFactory5)), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// PCWSTR returns s as a NUL-terminated UTF-16 string, cut at the first NUL.
func PCWSTR(s string) []uint16 {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		for i := 0; i < len(s); i++ {
			if s[i] == 0 {
				u, _ = windows.UTF16FromString(s[:i])
				break
			}
		}
	}
	return u
}
