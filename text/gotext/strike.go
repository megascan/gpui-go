package gotext

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"

	ot "github.com/go-text/typesetting/font/opentype"
)

var (
	tagEBLC = ot.MustNewTag("EBLC")
	tagEBDT = ot.MustNewTag("EBDT")
	tagCBLC = ot.MustNewTag("CBLC")
	tagCBDT = ot.MustNewTag("CBDT")
	tagSbix = ot.MustNewTag("sbix")
)

const appleColorEmoji = "Apple Color Emoji"

type swashBitmapStrikesProxy struct {
	bitmaps      [2][]byte
	colorBitmaps [2][]byte
	colorIsSbix  bool
	upem         uint16
	isApple      bool
}

func newSwashBitmapStrikesProxy(tables rawTables) swashBitmapStrikesProxy {
	upem, ok := u16At(tables.head, 18)
	if !ok {
		upem = 1
	}
	p := swashBitmapStrikesProxy{upem: upem}
	if len(tables.eblc) != 0 && len(tables.ebdt) != 0 {
		p.bitmaps = [2][]byte{tables.eblc, tables.ebdt}
	}
	if len(tables.sbix) != 0 {
		p.colorBitmaps = [2][]byte{tables.sbix, tables.sbix}
		p.colorIsSbix = true
		if families := collectFamilies(nameIDFamily, parseNameRecords(tables.name)); len(families) > 0 {
			p.isApple = families[0].name == appleColorEmoji
		}
	} else if len(tables.cblc) != 0 && len(tables.cbdt) != 0 {
		p.colorBitmaps = [2][]byte{tables.cblc, tables.cbdt}
	}
	return p
}

func (p *swashBitmapStrikesProxy) hasAlpha() bool { return len(p.bitmaps[0]) != 0 }

func (p *swashBitmapStrikesProxy) hasColor() bool { return len(p.colorBitmaps[0]) != 0 }

func (p *swashBitmapStrikesProxy) materializeAlpha() swashBitmapStrikes {
	return newSwashBitmapStrikes(p.bitmaps[0], p.bitmaps[1], p.upem, false, false)
}

func (p *swashBitmapStrikesProxy) materializeColor() swashBitmapStrikes {
	return newSwashBitmapStrikes(p.colorBitmaps[0], p.colorBitmaps[1], p.upem, p.colorIsSbix, p.isApple)
}

type swashBitmapStrikes struct {
	data       []byte
	bitmapData []byte
	isSbix     bool
	isApple    bool
	upem       uint16
	len        int
}

func newSwashBitmapStrikes(data, bitmapData []byte, upem uint16, isSbix, isApple bool) swashBitmapStrikes {
	n, _ := u32At(data, 4)
	return swashBitmapStrikes{data: data, bitmapData: bitmapData, upem: upem, isSbix: isSbix, isApple: isApple, len: int(n)}
}

func (s swashBitmapStrikes) get(index int) (swashBitmapStrike, bool) {
	if index >= s.len {
		return swashBitmapStrike{}, false
	}
	var offset int
	if s.isSbix {
		o, ok := u32At(s.data, 8+index*4)
		if !ok {
			return swashBitmapStrike{}, false
		}
		offset = int(o)
	} else {
		offset = 8 + index*48
	}
	return swashBitmapStrike{data: s.data, bitmapData: s.bitmapData, upem: s.upem, isSbix: s.isSbix, isApple: s.isApple, offset: offset}, true
}

func (s swashBitmapStrikes) findByNearestPpem(ppem uint16, glyphID uint16) (swashBitmapStrike, bool) {
	var best swashBitmapStrike
	found := false
	bestSize := uint16(0)
	for i := 0; i < s.len; i++ {
		strike, ok := s.get(i)
		if !ok {
			continue
		}
		if !strike.contains(glyphID) {
			continue
		}
		best, found = strike, true
		strikePpem := strike.ppem()
		if strikePpem > bestSize {
			best = strike
			bestSize = strikePpem
		}
		if strikePpem >= ppem {
			return strike, true
		}
	}
	return best, found
}

func (s swashBitmapStrikes) findByExactPpem(ppem uint16, glyphID uint16) (swashBitmapStrike, bool) {
	for i := 0; i < s.len; i++ {
		strike, ok := s.get(i)
		if !ok {
			continue
		}
		if !strike.contains(glyphID) {
			continue
		}
		if strike.ppem() == ppem {
			return strike, true
		}
	}
	return swashBitmapStrike{}, false
}

func (s swashBitmapStrikes) findByLargestPpem(glyphID uint16) (swashBitmapStrike, bool) {
	var largest swashBitmapStrike
	found := false
	largestPpem := uint16(0)
	for i := 0; i < s.len; i++ {
		strike, ok := s.get(i)
		if !ok {
			continue
		}
		if !strike.contains(glyphID) {
			continue
		}
		strikePpem := strike.ppem()
		if !found || strikePpem > largestPpem {
			largest, found = strike, true
			largestPpem = strikePpem
		}
	}
	return largest, found
}

type swashBitmapStrike struct {
	data       []byte
	bitmapData []byte
	offset     int
	upem       uint16
	isSbix     bool
	isApple    bool
}

func (s swashBitmapStrike) ppem() uint16 {
	if s.isSbix {
		v, _ := u16At(s.data, s.offset)
		return v
	}
	v, _ := u8At(s.data, s.offset+45)
	return uint16(v)
}

func (s swashBitmapStrike) contains(glyphID uint16) bool {
	v, ok := getCoverage(s.data, s.offset, s.isSbix, glyphID)
	return ok && v
}

func (s swashBitmapStrike) get(glyphID uint16) (swashBitmap, bool) {
	loc, ok := getLocation(s.data, s.offset, s.isSbix, glyphID)
	if !ok {
		return swashBitmap{}, false
	}
	return loc.get(s.bitmapData, s.upem, s.isApple)
}

type swashBitmapFormatKind uint8

const (
	swashBitmapFormatAlpha swashBitmapFormatKind = iota
	swashBitmapFormatPacked
	swashBitmapFormatColor
	swashBitmapFormatPng
)

type swashBitmapFormat struct {
	kind swashBitmapFormatKind
	bits uint8
}

func (f swashBitmapFormat) channels() uint32 {
	switch f.kind {
	case swashBitmapFormatAlpha, swashBitmapFormatPacked:
		return 1
	default:
		return 4
	}
}

func (f swashBitmapFormat) bufferSize(width, height uint32) int {
	return int(width * height * f.channels())
}

type swashBitmap struct {
	format swashBitmapFormat
	ppem   uint16
	width  uint32
	height uint32
	left   int32
	top    int32
	data   []byte
}

func newSwashBitmap(data *swashBitmapData, upem uint16, isApple bool) (swashBitmap, bool) {
	var format swashBitmapFormat
	switch {
	case data.isPacked:
		format = swashBitmapFormat{kind: swashBitmapFormatPacked, bits: data.bitDepth}
	case data.isPng:
		format = swashBitmapFormat{kind: swashBitmapFormatPng}
	case data.bitDepth == 32:
		format = swashBitmapFormat{kind: swashBitmapFormatColor}
	default:
		format = swashBitmapFormat{kind: swashBitmapFormatAlpha, bits: data.bitDepth}
	}
	var width, height uint32
	var left, top int32
	if data.isPng {
		w, ok := u32At(data.data, 16)
		if !ok {
			return swashBitmap{}, false
		}
		h, ok := u32At(data.data, 20)
		if !ok {
			return swashBitmap{}, false
		}
		width, height = w, h
		left, top = int32(data.metrics.x), int32(data.metrics.y)
		if data.isSbix {
			if top == 0 && isApple {
				s := float32(data.ppem) / float32(upem)
				top = int32(math.Round(float64(-100 * s)))
			}
			top += int32(height)
		}
	} else {
		width, height = uint32(data.width), uint32(data.height)
		left, top = int32(data.metrics.x), int32(data.metrics.y)
	}
	return swashBitmap{format: format, ppem: data.ppem, width: width, height: height, left: left, top: top, data: data.data}, true
}

func (b *swashBitmap) decodedSize() int {
	return b.format.bufferSize(b.width, b.height)
}

func (b *swashBitmap) scaledSize(size float32) (uint32, uint32, int) {
	w, h := b.width, b.height
	if size != 0 {
		scale := size / float32(b.ppem)
		w = uint32(float32(w) * scale)
		h = uint32(float32(h) * scale)
	}
	return w, h, b.format.bufferSize(w, h)
}

func (b *swashBitmap) decode(target []byte) bool {
	size := b.decodedSize()
	if len(target) < size {
		return false
	}
	w, h := int(b.width), int(b.height)
	src, dst := b.data, target
	unpackRows := func(bits int, rowLen int) bool {
		di := 0
		for start := 0; start < len(src) && di < w*h; start += rowLen {
			end := min(start+rowLen, len(src))
			row := src[start:end]
			for x := 0; x < w; x++ {
				var v byte
				switch bits {
				case 1:
					if x>>3 >= len(row) {
						return false
					}
					v = (row[x>>3] >> (^x & 7) & 1) * 255
				case 2:
					if x>>2 >= len(row) {
						return false
					}
					v = (row[x>>2] >> (^(x * 2) & 2) & 3) * 85
				case 4:
					if x>>1 >= len(row) {
						return false
					}
					v = (row[x>>1] >> (^(x * 4) & 4) & 15) * 17
				}
				dst[di] = v
				di++
			}
		}
		return true
	}
	switch b.format.kind {
	case swashBitmapFormatPacked:
		switch b.format.bits {
		case 1:
			for x := 0; x < w*h; x++ {
				dst[x] = (src[x>>3] >> (^x & 7) & 1) * 255
			}
		case 2:
			for x := 0; x < w*h; x++ {
				dst[x] = (src[x>>2] >> (^(x * 2) & 2) & 3) * 85
			}
		case 4:
			for x := 0; x < w*h; x++ {
				dst[x] = (src[x>>1] >> (^(x * 4) & 4) & 15) * 17
			}
		case 8, 32:
			if len(src) != len(dst) {
				panic("source slice length does not match destination slice length")
			}
			copy(dst, src)
		default:
			return false
		}
	case swashBitmapFormatAlpha:
		switch bits := int(b.format.bits); bits {
		case 1, 2, 4:
			if !unpackRows(bits, (w*bits+7)/8) {
				return false
			}
		case 8, 32:
			if len(src) != len(dst) {
				panic("source slice length does not match destination slice length")
			}
			copy(dst, src)
		default:
			return false
		}
	case swashBitmapFormatColor:
		if len(src) != len(dst) {
			panic("source slice length does not match destination slice length")
		}
		copy(dst, src)
	case swashBitmapFormatPng:
		if _, _, ok := decodePng(src, target); !ok {
			return false
		}
	}
	return true
}

func decodePng(data []byte, target []byte) (uint32, uint32, bool) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, false
	}
	r := img.Bounds()
	w, h := r.Dx(), r.Dy()
	if len(target) < w*h*4 {
		return 0, 0, false
	}
	dst := &image.NRGBA{Pix: target[:w*h*4], Stride: w * 4, Rect: image.Rect(0, 0, w, h)}
	draw.Draw(dst, dst.Rect, img, r.Min, draw.Src)
	return uint32(w), uint32(h), true
}

type swashLocation struct {
	format          uint8
	flags           uint8
	offset          uint32
	size            uint32
	ppem            uint16
	bitDepth        uint8
	width           uint8
	height          uint8
	metrics         swashBitmapMetrics
	verticalMetrics swashBitmapMetrics
}

func (l *swashLocation) get(data []byte, upem uint16, isApple bool) (swashBitmap, bool) {
	d, ok := getData(data, l)
	if !ok {
		return swashBitmap{}, false
	}
	return newSwashBitmap(&d, upem, isApple)
}

func getCoverage(table []byte, strikeBase int, isSbix bool, glyphID uint16) (bool, bool) {
	if isSbix {
		_, _, ok := sbixRange(table, strikeBase, glyphID, 0)
		return ok, true
	}
	if strikeBase > len(table) {
		return false, false
	}
	b := table[strikeBase:]
	start, ok := u16At(b, 40)
	if !ok || glyphID < start {
		return false, false
	}
	end, ok := u16At(b, 42)
	if !ok || glyphID > end {
		return false, false
	}
	count, ok := u32At(b, 8)
	if !ok {
		return false, false
	}
	arrayOffset, ok := u32At(b, 0)
	if !ok || int(arrayOffset) > len(table) {
		return false, false
	}
	a := table[arrayOffset:]
	for i := 0; i < int(count); i++ {
		offset := i * 8
		first, ok := u16At(a, offset)
		if !ok || glyphID < first {
			return false, false
		}
		last, ok := u16At(a, offset+2)
		if !ok {
			return false, false
		}
		if glyphID > last {
			continue
		}
		return true, true
	}
	return false, false
}

func getLocation(d []byte, strikeBase int, isSbix bool, glyphID uint16) (swashLocation, bool) {
	if isSbix {
		start, end, ok := sbixRange(d, strikeBase, glyphID, 0)
		if !ok {
			return swashLocation{}, false
		}
		length := end - start
		x, ok := i16At(d, int(start))
		if !ok {
			return swashLocation{}, false
		}
		y, ok := i16At(d, int(start)+2)
		if !ok {
			return swashLocation{}, false
		}
		ppem, ok := u16At(d, strikeBase)
		if !ok {
			return swashLocation{}, false
		}
		return swashLocation{
			ppem:     ppem,
			bitDepth: 32,
			metrics:  swashBitmapMetrics{x: int8(x), y: int8(y)},
			format:   0xFF,
			flags:    1,
			offset:   start + 8,
			size:     length - 8,
		}, true
	}
	start, ok := u16At(d, strikeBase+40)
	if !ok || glyphID < start {
		return swashLocation{}, false
	}
	end, ok := u16At(d, strikeBase+42)
	if !ok || glyphID > end {
		return swashLocation{}, false
	}
	count, ok1 := u32At(d, strikeBase+8)
	ppem, ok2 := u8At(d, strikeBase+45)
	bitDepth, ok3 := u8At(d, strikeBase+46)
	flags, ok4 := u8At(d, strikeBase+47)
	arrayOffset, ok5 := u32At(d, strikeBase)
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return swashLocation{}, false
	}
	for i := 0; i < int(count); i++ {
		offset := int(arrayOffset) + i*8
		first, ok := u16At(d, offset)
		if !ok || glyphID < first {
			return swashLocation{}, false
		}
		last, ok := u16At(d, offset+2)
		if !ok {
			return swashLocation{}, false
		}
		if glyphID > last {
			continue
		}
		sub, ok := u32At(d, offset+4)
		if !ok {
			return swashLocation{}, false
		}
		offset = int(arrayOffset) + int(sub)
		indexFormat, ok1 := u16At(d, offset)
		imageFormat, ok2 := u16At(d, offset+2)
		imageOffset, ok3 := u32At(d, offset+4)
		if !ok1 || !ok2 || !ok3 {
			return swashLocation{}, false
		}
		base := offset + 8
		loc := swashLocation{ppem: uint16(ppem), bitDepth: bitDepth, format: uint8(imageFormat), flags: flags}
		switch indexFormat {
		case 1:
			v, ok := u32At(d, base+int(glyphID-first)*4)
			if !ok {
				return swashLocation{}, false
			}
			loc.offset = imageOffset + v
			return loc, true
		case 2:
			size, ok := u32At(d, base)
			if !ok {
				return swashLocation{}, false
			}
			loc.size = size
			loc.offset = imageOffset + loc.size*uint32(glyphID-first)
			w, h, ok := getMetrics(d, base+4, flags, true, &loc.metrics, &loc.verticalMetrics)
			if !ok {
				return swashLocation{}, false
			}
			loc.width, loc.height = w, h
			return loc, true
		case 3:
			v, ok := u16At(d, base+int(glyphID-first)*2)
			if !ok {
				return swashLocation{}, false
			}
			loc.offset = imageOffset + uint32(v)
			return loc, true
		case 4:
			l := 0
			hv, ok := u32At(d, base)
			if !ok {
				return swashLocation{}, false
			}
			h := int(hv)
			for l < h {
				i := (l + h) / 2
				rec := base + i*4
				id, ok := u16At(d, rec)
				if !ok {
					return swashLocation{}, false
				}
				switch {
				case glyphID < id:
					h = i
				case glyphID > id:
					l = i + i
				default:
					offset1, ok1 := u16At(d, rec+2)
					offset2, ok2 := u16At(d, rec+6)
					if !ok1 || !ok2 || offset2 <= offset1 {
						return swashLocation{}, false
					}
					loc.offset = imageOffset + uint32(offset1)
					loc.size = uint32(offset2) - uint32(offset1)
					return loc, true
				}
			}
		default:
			return swashLocation{}, false
		}
	}
	return swashLocation{}, false
}

func readBytes(d []byte, offset, size int) ([]byte, bool) {
	if offset < 0 || size < 0 || offset+size > len(d) {
		return nil, false
	}
	return d[offset : offset+size], true
}

func getData(d []byte, loc *swashLocation) (swashBitmapData, bool) {
	depth := int(loc.bitDepth)
	bitmap := swashBitmapData{
		ppem:            loc.ppem,
		bitDepth:        loc.bitDepth,
		width:           loc.width,
		height:          loc.height,
		metrics:         loc.metrics,
		verticalMetrics: loc.verticalMetrics,
	}
	offset := int(loc.offset)
	flags := loc.flags
	size := int(loc.size)
	var ok bool
	switch loc.format {
	case 0xFF:
		if bitmap.data, ok = readBytes(d, offset, size); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPng = true
		bitmap.isSbix = true
	case 1:
		if !bitmap.readMetrics(d, offset, flags, false) {
			return swashBitmapData{}, false
		}
		w := (int(bitmap.width)*depth + 7) / 8
		if bitmap.data, ok = readBytes(d, offset+5, w*int(bitmap.height)); !ok {
			return swashBitmapData{}, false
		}
	case 2:
		if !bitmap.readMetrics(d, offset, flags, false) {
			return swashBitmapData{}, false
		}
		w := int(bitmap.width) * depth
		if bitmap.data, ok = readBytes(d, offset+5, (w*int(bitmap.height)+7)/8); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPacked = true
	case 5:
		if bitmap.data, ok = readBytes(d, offset, size); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPacked = true
	case 6:
		if !bitmap.readMetrics(d, offset, flags, true) {
			return swashBitmapData{}, false
		}
		w := (int(bitmap.width)*depth + 7) / 8
		if bitmap.data, ok = readBytes(d, offset+8, w*int(bitmap.height)); !ok {
			return swashBitmapData{}, false
		}
	case 7:
		if !bitmap.readMetrics(d, offset, flags, true) {
			return swashBitmapData{}, false
		}
		w := int(bitmap.width) * depth
		if bitmap.data, ok = readBytes(d, offset+8, (w*int(bitmap.height)+7)/8); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPacked = true
	case 17:
		if !bitmap.readMetrics(d, offset, flags, false) {
			return swashBitmapData{}, false
		}
		n, ok := u32At(d, offset+5)
		if !ok {
			return swashBitmapData{}, false
		}
		if bitmap.data, ok = readBytes(d, offset+9, int(n)); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPng = true
	case 18:
		if !bitmap.readMetrics(d, offset, flags, true) {
			return swashBitmapData{}, false
		}
		n, ok := u32At(d, offset+8)
		if !ok {
			return swashBitmapData{}, false
		}
		if bitmap.data, ok = readBytes(d, offset+12, int(n)); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPng = true
	case 19:
		n, ok := u32At(d, offset)
		if !ok {
			return swashBitmapData{}, false
		}
		if bitmap.data, ok = readBytes(d, offset+4, int(n)); !ok {
			return swashBitmapData{}, false
		}
		bitmap.isPng = true
	default:
		return swashBitmapData{}, false
	}
	return bitmap, true
}

var (
	tagDupe = ot.MustNewTag("dupe")
	tagPng  = ot.MustNewTag("png ")
)

func sbixRange(table []byte, strikeBase int, glyphID uint16, recurse int) (uint32, uint32, bool) {
	if recurse > 1 {
		return 0, 0, false
	}
	id := int(glyphID)
	base := strikeBase + 4
	start, ok := u32At(table, base+id*4)
	if !ok {
		return 0, 0, false
	}
	end, ok := u32At(table, base+(id+1)*4)
	if !ok || end <= start {
		return 0, 0, false
	}
	start += uint32(strikeBase)
	end += uint32(strikeBase)
	tag, ok := u32At(table, int(start)+4)
	if !ok {
		return 0, 0, false
	}
	switch ot.Tag(tag) {
	case tagDupe:
		dupe, ok := u16At(table, int(start)+8)
		if !ok {
			return 0, 0, false
		}
		return sbixRange(table, strikeBase, dupe, recurse+1)
	case tagPng:
		return start, end, true
	}
	return 0, 0, false
}

type swashBitmapData struct {
	data            []byte
	ppem            uint16
	bitDepth        uint8
	width           uint8
	height          uint8
	metrics         swashBitmapMetrics
	verticalMetrics swashBitmapMetrics
	isPacked        bool
	isPng           bool
	isSbix          bool
}

func (b *swashBitmapData) readMetrics(d []byte, offset int, flags uint8, big bool) bool {
	w, h, ok := getMetrics(d, offset, flags, big, &b.metrics, &b.verticalMetrics)
	if !ok {
		return false
	}
	b.width, b.height = w, h
	return true
}

type swashBitmapMetrics struct {
	x       int8
	y       int8
	advance uint8
}

func getMetrics(d []byte, offset int, flags uint8, big bool, h, v *swashBitmapMetrics) (uint8, uint8, bool) {
	width, ok1 := u8At(d, offset+1)
	height, ok2 := u8At(d, offset)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	read := func(m *swashBitmapMetrics, at int) bool {
		x, ok1 := u8At(d, at)
		y, ok2 := u8At(d, at+1)
		adv, ok3 := u8At(d, at+2)
		if !ok1 || !ok2 || !ok3 {
			return false
		}
		m.x, m.y, m.advance = int8(x), int8(y), adv
		return true
	}
	if big {
		if !read(h, offset+2) || !read(v, offset+5) {
			return 0, 0, false
		}
	} else {
		m := h
		if flags&2 != 0 {
			m = v
		}
		if !read(m, offset+2) {
			return 0, 0, false
		}
	}
	return width, height, true
}
