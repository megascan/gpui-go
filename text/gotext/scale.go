package gotext

import (
	"math"

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
	source       swashSource
	paletteIndex uint16
	strike       swashStrikeWith
	content      swashContent
	placement    zeno.Placement
	data         []byte
}

func (i *swashImage) clear() {
	i.source = swashSourceOutline
	i.paletteIndex = 0
	i.strike = swashStrikeWithExactSize
	i.content = swashContentMask
	i.placement = zeno.Placement{}
	i.data = i.data[:0]
}

type swashScaleContext struct {
	rcx      *zeno.Scratch
	outline  swashOutline
	scratch0 []byte
	scratch1 []byte
}

func newSwashScaleContext() *swashScaleContext {
	return &swashScaleContext{rcx: zeno.NewScratch()}
}

type swashLayerData struct {
	points        [2]int
	verbs         [2]int
	colorIndex    uint16
	hasColorIndex bool
}

type swashOutline struct {
	points []zeno.Point
	verbs  []zeno.Verb
	layers []swashLayerData
}

type swashOutlineLayer struct {
	points        []zeno.Point
	verbs         []zeno.Verb
	colorIndex    uint16
	hasColorIndex bool
}

func (l swashOutlineLayer) path() zeno.PointsCommands {
	return zeno.PointsCommands{Points: l.points, Verbs: l.verbs}
}

func (o *swashOutline) clear() {
	o.points = o.points[:0]
	o.verbs = o.verbs[:0]
	o.layers = o.layers[:0]
}

func (o *swashOutline) len() int { return len(o.layers) }

func (o *swashOutline) get(index int) (swashOutlineLayer, bool) {
	if index < 0 || index >= len(o.layers) {
		return swashOutlineLayer{}, false
	}
	d := o.layers[index]
	if d.points[0] > d.points[1] || d.points[1] > len(o.points) || d.verbs[0] > d.verbs[1] || d.verbs[1] > len(o.verbs) {
		return swashOutlineLayer{}, false
	}
	return swashOutlineLayer{
		points:        o.points[d.points[0]:d.points[1]],
		verbs:         o.verbs[d.verbs[0]:d.verbs[1]],
		colorIndex:    d.colorIndex,
		hasColorIndex: d.hasColorIndex,
	}, true
}

func (o *swashOutline) path() zeno.PointsCommands {
	return zeno.PointsCommands{Points: o.points, Verbs: o.verbs}
}

func (o *swashOutline) bounds() zeno.Bounds {
	return zeno.BoundsFromPoints(o.points)
}

func (o *swashOutline) beginLayer(colorIndex uint16, hasColorIndex bool) {
	pointsEnd, verbsEnd := len(o.points), len(o.verbs)
	if n := len(o.layers); n > 0 {
		o.layers[n-1].points[1] = pointsEnd
		o.layers[n-1].verbs[1] = verbsEnd
	}
	o.layers = append(o.layers, swashLayerData{
		points:        [2]int{pointsEnd, pointsEnd},
		verbs:         [2]int{verbsEnd, verbsEnd},
		colorIndex:    colorIndex,
		hasColorIndex: hasColorIndex,
	})
}

func (o *swashOutline) maybeClose() {
	if n := len(o.verbs); n > 0 && o.verbs[n-1] != zeno.VerbClose {
		o.verbs = append(o.verbs, zeno.VerbClose)
	}
}

func (o *swashOutline) finish() {
	pointsEnd, verbsEnd := len(o.points), len(o.verbs)
	if n := len(o.layers); n > 0 {
		o.layers[n-1].points[1] = pointsEnd
		o.layers[n-1].verbs[1] = verbsEnd
	} else {
		o.layers = append(o.layers, swashLayerData{points: [2]int{0, pointsEnd}, verbs: [2]int{0, verbsEnd}})
	}
}

type swashScaler struct {
	cx    *swashScaleContext
	font  *cosmicFont
	face  *font.Face
	size  float32
	scale float32
}

func (cx *swashScaleContext) builder(f *cosmicFont, size float32) *swashScaler {
	upem := f.face.Upem()
	scale := float32(0)
	if size != 0 && upem != 0 {
		scale = size / float32(upem)
	}
	return &swashScaler{cx: cx, font: f, face: f.face, size: size, scale: scale}
}

func (s *swashScaler) hasColorOutlines() bool {
	return len(s.font.color.colr) != 0 && len(s.font.color.cpal) != 0
}

func (s *swashScaler) hasBitmaps() bool { return s.font.bitmaps.hasAlpha() }

func (s *swashScaler) hasColorBitmaps() bool { return s.font.bitmaps.hasColor() }

func (s *swashScaler) scaleOutline(glyphID uint16) bool {
	return s.scaleOutlineImpl(glyphID, 0, false)
}

func (s *swashScaler) scaleOutlineImpl(glyphID uint16, colorIndex uint16, hasColorIndex bool) bool {
	data, ok := s.face.GlyphDataOutline(font.GID(glyphID))
	if !ok {
		return false
	}
	o := &s.cx.outline
	o.beginLayer(colorIndex, hasColorIndex)
	pt := func(p ot.SegmentPoint) zeno.Point {
		return zeno.NewVector(p.X*s.scale, p.Y*s.scale)
	}
	for _, seg := range data.Segments {
		switch seg.Op {
		case ot.SegmentOpMoveTo:
			o.maybeClose()
			o.points = append(o.points, pt(seg.Args[0]))
			o.verbs = append(o.verbs, zeno.VerbMoveTo)
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
	o.maybeClose()
	o.finish()
	return true
}

func (s *swashScaler) scaleColorOutlineImpl(glyphID uint16) bool {
	if !s.hasColorOutlines() {
		return false
	}
	layers, ok := s.font.color.layers(glyphID)
	if !ok {
		return false
	}
	s.cx.outline.clear()
	for i := uint16(0); i < layers.len; i++ {
		layer, ok := layers.get(i)
		if !ok {
			return false
		}
		if !s.scaleOutlineImpl(layer.glyphID, layer.colorIndex, layer.hasColorIndex) {
			return false
		}
	}
	return true
}

type swashStrikeWith uint8

const (
	swashStrikeWithExactSize swashStrikeWith = iota
	swashStrikeWithBestFit
	swashStrikeWithLargestSize
)

func (s *swashScaler) scaleBitmapInto(glyphID uint16, strike swashStrikeWith, image *swashImage) bool {
	return s.scaleBitmapImpl(glyphID, false, strike, image)
}

func (s *swashScaler) scaleColorBitmapInto(glyphID uint16, strike swashStrikeWith, image *swashImage) bool {
	return s.scaleBitmapImpl(glyphID, true, strike, image)
}

func (s *swashScaler) scaleBitmapImpl(glyphID uint16, color bool, strike swashStrikeWith, image *swashImage) bool {
	image.clear()
	size := s.size
	var strikes swashBitmapStrikes
	if color {
		strikes = s.font.bitmaps.materializeColor()
	} else {
		strikes = s.font.bitmaps.materializeAlpha()
	}
	var found swashBitmapStrike
	var ok bool
	switch strike {
	case swashStrikeWithExactSize:
		if size == 0 {
			return false
		}
		found, ok = strikes.findByExactPpem(uint16(size), glyphID)
	case swashStrikeWithBestFit:
		if size == 0 {
			return false
		}
		found, ok = strikes.findByNearestPpem(uint16(size), glyphID)
	default:
		found, ok = strikes.findByLargestPpem(glyphID)
	}
	if !ok {
		return false
	}
	bitmap, ok := found.get(glyphID)
	if !ok {
		return false
	}
	if bitmap.ppem == 0 {
		return false
	}
	_, _, bufsize := bitmap.scaledSize(size)
	image.data = resizeBytes(image.data, bufsize)
	s.cx.scratch0 = s.cx.scratch0[:0]
	s.cx.scratch1 = s.cx.scratch1[:0]
	w, h := bitmap.width, bitmap.height
	scale := size / float32(bitmap.ppem)
	if size != 0 && scale != 1 {
		s.cx.scratch0 = resizeBytes(s.cx.scratch0, bitmap.format.bufferSize(w, h))
		w = uint32(float32(w) * scale)
		h = uint32(float32(h) * scale)
		image.data = resizeBytes(image.data, bitmap.format.bufferSize(w, h))
		if !bitmap.decode(s.cx.scratch0) {
			return false
		}
		if !swashResize(s.cx.scratch0, bitmap.width, bitmap.height, bitmap.format.channels(), image.data, w, h, swashFilterMitchell, &s.cx.scratch1) {
			return false
		}
		image.placement = zeno.Placement{
			Left:   int32(float32(bitmap.left) * scale),
			Top:    int32(float32(bitmap.top) * scale),
			Width:  w,
			Height: h,
		}
	} else {
		image.data = resizeBytes(image.data, bitmap.format.bufferSize(w, h))
		if !bitmap.decode(image.data) {
			return false
		}
		image.placement = zeno.Placement{Left: bitmap.left, Top: bitmap.top, Width: w, Height: h}
	}
	if color {
		image.source = swashSourceColorBitmap
	} else {
		image.source = swashSourceBitmap
	}
	image.strike = strike
	if bitmap.format.channels() == 1 {
		image.content = swashContentMask
	} else {
		image.content = swashContentColor
	}
	return true
}

type swashSourceEntry struct {
	source       swashSource
	paletteIndex uint16
	strike       swashStrikeWith
}

type swashRender struct {
	sources    []swashSourceEntry
	format     zeno.Format
	offset     zeno.Vector
	foreground [4]byte
}

func newSwashRender(sources []swashSourceEntry) swashRender {
	return swashRender{sources: sources, format: zeno.FormatAlpha, foreground: [4]byte{128, 128, 128, 255}}
}

func (r *swashRender) renderInto(scaler *swashScaler, glyphID uint16, image *swashImage) bool {
	for _, entry := range r.sources {
		switch entry.source {
		case swashSourceOutline:
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
					image.data = resizeBytes(image.data, f.BufferSize(w, h))
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
		case swashSourceColorOutline:
			if !scaler.hasColorOutlines() {
				continue
			}
			scaler.cx.outline.clear()
			if !scaler.scaleColorOutlineImpl(glyphID) {
				continue
			}
			cx := scaler.cx
			outline := &cx.outline
			palette, hasPalette := scaler.font.color.palette(entry.paletteIndex)
			totalBounds := outline.bounds()
			baseX := int32(math.Floor(float64(totalBounds.Min.X + r.offset.X)))
			baseY := int32(math.Ceil(float64(totalBounds.Min.Y + r.offset.Y)))
			baseW := uint32(math.Ceil(float64(totalBounds.Width())))
			baseH := uint32(math.Ceil(float64(totalBounds.Height())))
			image.data = resizeBytes(image.data, int(baseW*baseH*4))
			image.placement.Left = baseX
			image.placement.Top = int32(baseH) + baseY
			image.placement.Width = baseW
			image.placement.Height = baseH
			ok := true
			for i := 0; i < outline.len(); i++ {
				layer, found := outline.get(i)
				if !found {
					ok = false
					break
				}
				cx.scratch0 = cx.scratch0[:0]
				placement := zeno.MaskWithScratch(layer.path(), cx.rcx).
					Origin(zeno.OriginBottomLeft).
					Style(zeno.FillNonZero).
					Offset(r.offset).
					RenderOffset(r.offset).
					Inspect(func(f zeno.Format, w, h uint32) {
						cx.scratch0 = resizeBytes(cx.scratch0, f.BufferSize(w, h))
					}).
					RenderInto(cx.scratch0, 0)
				color := r.foreground
				if layer.hasColorIndex && hasPalette {
					color = palette.get(layer.colorIndex)
				}
				swashBlit(cx.scratch0, placement.Width, placement.Height,
					placement.Left-baseX, int32(baseH)+baseY-placement.Top,
					color, image.data, baseW, baseH)
			}
			if ok {
				image.source = swashSourceColorOutline
				image.paletteIndex = entry.paletteIndex
				image.content = swashContentColor
				return true
			}
		case swashSourceBitmap:
			if !scaler.hasBitmaps() {
				continue
			}
			if scaler.scaleBitmapInto(glyphID, entry.strike, image) {
				return true
			}
		case swashSourceColorBitmap:
			if !scaler.hasColorBitmaps() {
				continue
			}
			if scaler.scaleColorBitmapInto(glyphID, entry.strike, image) {
				return true
			}
		}
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
