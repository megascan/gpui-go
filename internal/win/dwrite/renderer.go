//go:build windows

package dwrite

import (
	"runtime"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

// IDWriteTextRenderer receives the glyph runs of IDWriteTextLayout.Draw.
type IDWriteTextRenderer struct{ com.IUnknown }

// TextRendererImpl is the Go side of an IDWriteTextRenderer. Each method returns the error its
// COM method reports; HRESULTOf maps it to the returned HRESULT. The float origins of the draw
// methods are not passed, as no caller uses them.
type TextRendererImpl interface {
	IsPixelSnappingDisabled(clientDrawingContext uintptr) (bool, error)
	GetCurrentTransform(clientDrawingContext uintptr) (DWRITE_MATRIX, error)
	GetPixelsPerDip(clientDrawingContext uintptr) (float32, error)
	DrawGlyphRun(clientDrawingContext uintptr, measuringMode int32, glyphRun *DWRITE_GLYPH_RUN, glyphRunDescription *DWRITE_GLYPH_RUN_DESCRIPTION, clientDrawingEffect *com.IUnknown) error
	DrawUnderline(clientDrawingContext uintptr) error
	DrawStrikethrough(clientDrawingContext uintptr) error
	DrawInlineObject(clientDrawingContext uintptr) error
}

var textRendererVtable = com.NewLazyVtable(func() *com.Vtable {
	var drawGlyphRun uintptr
	if runtime.GOARCH == "arm64" {
		drawGlyphRun = windows.NewCallback(drawGlyphRunThunkArm64)
	} else {
		drawGlyphRun = windows.NewCallback(drawGlyphRunThunkAmd64)
	}
	return com.NewVtable(
		[]com.GUID{IID_IDWritePixelSnapping, IID_IDWriteTextRenderer},
		windows.NewCallback(isPixelSnappingDisabledThunk),
		windows.NewCallback(getCurrentTransformThunk),
		windows.NewCallback(getPixelsPerDipThunk),
		drawGlyphRun,
		windows.NewCallback(drawUnderlineThunk),
		windows.NewCallback(drawStrikethroughThunk),
		windows.NewCallback(drawInlineObjectThunk),
	)
})

// NewTextRenderer creates an IDWriteTextRenderer backed by impl, with one reference.
func NewTextRenderer(impl TextRendererImpl) *IDWriteTextRenderer {
	obj := com.NewObject(impl, textRendererVtable.Get())
	return com.As[IDWriteTextRenderer](obj.Interface(0))
}

func rendererImpl(this *com.IUnknown) (TextRendererImpl, bool) {
	return com.ImplOf[TextRendererImpl](this)
}

func isPixelSnappingDisabledThunk(this *com.IUnknown, ctx uintptr, out *BOOL) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	v, err := impl.IsPixelSnappingDisabled(ctx)
	if err != nil {
		return HRESULTOf(err).Uintptr()
	}
	*out = BoolOf(v)
	return com.S_OK.Uintptr()
}

func getCurrentTransformThunk(this *com.IUnknown, ctx uintptr, out *DWRITE_MATRIX) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	v, err := impl.GetCurrentTransform(ctx)
	if err != nil {
		return HRESULTOf(err).Uintptr()
	}
	*out = v
	return com.S_OK.Uintptr()
}

func getPixelsPerDipThunk(this *com.IUnknown, ctx uintptr, out *float32) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	v, err := impl.GetPixelsPerDip(ctx)
	if err != nil {
		return HRESULTOf(err).Uintptr()
	}
	*out = v
	return com.S_OK.Uintptr()
}

func drawGlyphRun(this *com.IUnknown, ctx uintptr, measuringMode uintptr, glyphRun *DWRITE_GLYPH_RUN, desc *DWRITE_GLYPH_RUN_DESCRIPTION, effect *com.IUnknown) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	return HRESULTOf(impl.DrawGlyphRun(ctx, int32(uint32(measuringMode)), glyphRun, desc, effect)).Uintptr()
}

func drawGlyphRunThunkAmd64(this *com.IUnknown, ctx uintptr, _, _ uintptr, measuringMode uintptr, glyphRun *DWRITE_GLYPH_RUN, desc *DWRITE_GLYPH_RUN_DESCRIPTION, effect *com.IUnknown) uintptr {
	return drawGlyphRun(this, ctx, measuringMode, glyphRun, desc, effect)
}

func drawGlyphRunThunkArm64(this *com.IUnknown, ctx uintptr, measuringMode uintptr, glyphRun *DWRITE_GLYPH_RUN, desc *DWRITE_GLYPH_RUN_DESCRIPTION, effect *com.IUnknown) uintptr {
	return drawGlyphRun(this, ctx, measuringMode, glyphRun, desc, effect)
}

func drawUnderlineThunk(this *com.IUnknown, ctx uintptr) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	return HRESULTOf(impl.DrawUnderline(ctx)).Uintptr()
}

func drawStrikethroughThunk(this *com.IUnknown, ctx uintptr) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	return HRESULTOf(impl.DrawStrikethrough(ctx)).Uintptr()
}

func drawInlineObjectThunk(this *com.IUnknown, ctx uintptr) uintptr {
	impl, ok := rendererImpl(this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	return HRESULTOf(impl.DrawInlineObject(ctx)).Uintptr()
}
