package gotext

import (
	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/megascan/gpui-go/text/gotext/internal/zeno"
)

type swashContent uint8

const (
	swashContentMask swashContent = iota
	swashContentSubpixelMask
	swashContentColor
)

type swashSource uint8

const (
	swashSourceOutline swashSource = iota
	swashSourceColorOutline
	swashSourceColorBitmap
	swashSourceBitmap
)

type swashImage struct {
	source    swashSource
	content   swashContent
	placement zeno.Placement
	data      []byte
}

type swashScaleContext struct {
	rcx     *zeno.Scratch
	outline swashOutline
}

func newSwashScaleContext() *swashScaleContext {
	return &swashScaleContext{rcx: zeno.NewScratch()}
}

type swashOutline struct {
	points []zeno.Point
	verbs  []zeno.Verb
}

func (o *swashOutline) clear() {
	o.points = o.points[:0]
	o.verbs = o.verbs[:0]
}

func (o *swashOutline) path() zeno.PointsCommands {
	return zeno.PointsCommands{Points: o.points, Verbs: o.verbs}
}

type swashScaler struct {
	cx    *swashScaleContext
	face  *font.Face
	scale float32
}

func (cx *swashScaleContext) builder(face *font.Face, size float32) *swashScaler {
	upem := face.Upem()
	scale := float32(0)
	if size != 0 && upem != 0 {
		scale = size / float32(upem)
	}
	return &swashScaler{cx: cx, face: face, scale: scale}
}

func (s *swashScaler) scaleOutline(glyphID uint16) bool {
	data, ok := s.face.GlyphDataOutline(font.GID(glyphID))
	if !ok {
		return false
	}
	o := &s.cx.outline
	pt := func(p ot.SegmentPoint) zeno.Point {
		return zeno.NewVector(p.X*s.scale, p.Y*s.scale)
	}
	open := false
	for _, seg := range data.Segments {
		switch seg.Op {
		case ot.SegmentOpMoveTo:
			if open {
				o.verbs = append(o.verbs, zeno.VerbClose)
			}
			o.points = append(o.points, pt(seg.Args[0]))
			o.verbs = append(o.verbs, zeno.VerbMoveTo)
			open = true
		case ot.SegmentOpLineTo:
			o.points = append(o.points, pt(seg.Args[0]))
			o.verbs = append(o.verbs, zeno.VerbLineTo)
		case ot.SegmentOpQuadTo:
			o.points = append(o.points, pt(seg.Args[0]), pt(seg.Args[1]))
			o.verbs = append(o.verbs, zeno.VerbQuadTo)
		case ot.SegmentOpCubeTo:
			o.points = append(o.points, pt(seg.Args[0]), pt(seg.Args[1]), pt(seg.Args[2]))
			o.verbs = append(o.verbs, zeno.VerbCurveTo)
		}
	}
	if open {
		o.verbs = append(o.verbs, zeno.VerbClose)
	}
	return true
}

type swashRender struct {
	sources []swashSource
	format  zeno.Format
	offset  zeno.Vector
}

func (r *swashRender) renderInto(scaler *swashScaler, glyphID uint16, image *swashImage) bool {
	for _, source := range r.sources {
		if source != swashSourceOutline {
			continue
		}
		scaler.cx.outline.clear()
		if !scaler.scaleOutline(glyphID) {
			continue
		}
		path := scaler.cx.outline.path()
		placement := zeno.MaskWithScratch(path, scaler.cx.rcx).
			Format(r.format).
			Origin(zeno.OriginBottomLeft).
			Style(zeno.FillNonZero).
			Offset(r.offset).
			RenderOffset(r.offset).
			Inspect(func(f zeno.Format, w, h uint32) {
				n := f.BufferSize(w, h)
				if cap(image.data) < n {
					image.data = append(image.data[:cap(image.data)], make([]byte, n-cap(image.data))...)
				}
				image.data = image.data[:n]
			}).
			RenderInto(image.data, 0)
		image.placement = placement
		if r.format.IsAlpha() {
			image.content = swashContentMask
		} else {
			image.content = swashContentSubpixelMask
		}
		image.source = swashSourceOutline
		return true
	}
	return false
}

func (r *swashRender) render(scaler *swashScaler, glyphID uint16) (swashImage, bool) {
	var image swashImage
	if r.renderInto(scaler, glyphID, &image) {
		return image, true
	}
	return swashImage{}, false
}
