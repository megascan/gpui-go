package gotext

import (
	"encoding/binary"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/harfbuzz"
)

type cosmicFont struct {
	id      faceID
	font    *font.Font
	face    *font.Face
	shaper  *harfbuzz.Font
	metrics skrifaMetrics
	swash   swashMetrics
	glyphs  swashGlyphMetrics
	color   swashColorProxy
	bitmaps swashBitmapStrikesProxy
}

func newCosmicFont(db *fontDatabase, id faceID) (*cosmicFont, bool) {
	parsed, loader, err := db.loadFont(id)
	if err != nil {
		return nil, false
	}
	face := font.NewFace(parsed)
	tables := readRawTables(loader)
	return &cosmicFont{
		id:      id,
		font:    parsed,
		face:    face,
		shaper:  harfbuzz.NewFont(face),
		metrics: newSkrifaMetrics(tables),
		swash:   newSwashMetrics(tables),
		glyphs:  newSwashGlyphMetrics(tables),
		color:   newSwashColorProxy(tables),
		bitmaps: newSwashBitmapStrikesProxy(tables),
	}, true
}

func (f *cosmicFont) charmap(ch rune) uint16 {
	gid, ok := f.font.Cmap.Lookup(ch)
	if !ok || gid > 0xFFFF {
		return 0
	}
	return uint16(gid)
}

type rawTables struct {
	head, hhea, os2, post, hmtx, vhea, vmtx, maxp  []byte
	colr, cpal, eblc, ebdt, cblc, cbdt, sbix, name []byte
}

func readRawTables(loader *ot.Loader) rawTables {
	get := func(tag ot.Tag) []byte {
		data, err := loader.RawTable(tag)
		if err != nil {
			return nil
		}
		return data
	}
	return rawTables{
		head: get(tagHead),
		hhea: get(tagHhea),
		os2:  get(tagOS2),
		post: get(tagPost),
		hmtx: get(tagHmtx),
		vhea: get(tagVhea),
		vmtx: get(tagVmtx),
		maxp: get(tagMaxp),
		colr: get(tagCOLR),
		cpal: get(tagCPAL),
		eblc: get(tagEBLC),
		ebdt: get(tagEBDT),
		cblc: get(tagCBLC),
		cbdt: get(tagCBDT),
		sbix: get(tagSbix),
		name: get(tagName),
	}
}

func u16At(data []byte, offset int) (uint16, bool) {
	if offset+2 > len(data) {
		return 0, false
	}
	return binary.BigEndian.Uint16(data[offset:]), true
}

func i16At(data []byte, offset int) (int16, bool) {
	v, ok := u16At(data, offset)
	return int16(v), ok
}

func i16Or0(data []byte, offset int) int16 {
	v, _ := i16At(data, offset)
	return v
}

func u16Or0(data []byte, offset int) uint16 {
	v, _ := u16At(data, offset)
	return v
}

type skrifaMetrics struct {
	unitsPerEm float32
	ascent     float32
	descent    float32
}

func newSkrifaMetrics(t rawTables) skrifaMetrics {
	m := skrifaMetrics{unitsPerEm: float32(u16Or0(t.head, 18))}
	usedTypo := false
	hasOS2 := len(t.os2) >= 78
	if hasOS2 && u16Or0(t.os2, 62)&(1<<7) != 0 {
		m.ascent = float32(i16Or0(t.os2, 68))
		m.descent = float32(i16Or0(t.os2, 70))
		usedTypo = true
	}
	if !usedTypo {
		if len(t.hhea) >= 36 {
			m.ascent = float32(i16Or0(t.hhea, 4))
			m.descent = float32(i16Or0(t.hhea, 6))
		}
		if m.ascent == 0 && m.descent == 0 && hasOS2 {
			typoAscender, typoDescender := i16Or0(t.os2, 68), i16Or0(t.os2, 70)
			if typoAscender != 0 || typoDescender != 0 {
				m.ascent = float32(typoAscender)
				m.descent = float32(typoDescender)
			} else {
				m.ascent = float32(u16Or0(t.os2, 74))
				m.descent = -float32(u16Or0(t.os2, 76))
			}
		}
	}
	return m
}

type swashMetrics struct {
	unitsPerEm      uint16
	ascent          float32
	descent         float32
	leading         float32
	capHeight       float32
	xHeight         float32
	maxWidth        float32
	underlineOffset float32
	strokeSize      float32
}

func newSwashMetrics(t rawTables) swashMetrics {
	m := swashMetrics{unitsPerEm: 1}
	if len(t.head) < 54 {
		return m
	}
	m.unitsPerEm = u16Or0(t.head, 18)
	haveLineMetrics := false
	if len(t.os2) >= 78 {
		flags := u16Or0(t.os2, 62)
		m.strokeSize = float32(i16Or0(t.os2, 26))
		if u16Or0(t.os2, 0) >= 2 {
			m.xHeight = float32(i16Or0(t.os2, 86))
			m.capHeight = float32(i16Or0(t.os2, 88))
		}
		if flags&(1<<7) != 0 {
			m.ascent = float32(i16Or0(t.os2, 68))
			m.descent = -float32(i16Or0(t.os2, 70))
			m.leading = float32(i16Or0(t.os2, 72))
			haveLineMetrics = m.ascent != 0
		}
	}
	if len(t.hhea) >= 36 {
		m.maxWidth = float32(u16Or0(t.hhea, 10))
		if !haveLineMetrics {
			m.ascent = float32(i16Or0(t.hhea, 4))
			m.descent = -float32(i16Or0(t.hhea, 6))
			m.leading = float32(i16Or0(t.hhea, 8))
		}
	}
	if len(t.post) >= 32 {
		m.underlineOffset = float32(i16Or0(t.post, 8))
		m.strokeSize = float32(i16Or0(t.post, 10))
	}
	return m
}

type swashGlyphMetrics struct {
	glyphCount        uint16
	hmtx              []byte
	hmtxCount         uint16
	vmtx              []byte
	vmtxCount         uint16
	synthesizedHeight float32
}

func newSwashGlyphMetrics(t rawTables) swashGlyphMetrics {
	m := swashGlyphMetrics{glyphCount: u16Or0(t.maxp, 4), hmtx: t.hmtx, hmtxCount: 1}
	if len(t.hhea) >= 36 {
		m.hmtxCount = u16Or0(t.hhea, 34)
	}
	if len(t.vhea) >= 36 && len(t.vmtx) > 0 {
		m.vmtx = t.vmtx
		m.vmtxCount = u16Or0(t.vhea, 34)
	} else {
		s := newSwashMetrics(t)
		m.synthesizedHeight = s.ascent + s.descent
	}
	return m
}

func longMetric(table []byte, count, glyph uint16) uint16 {
	if count == 0 {
		return 0
	}
	if glyph >= count {
		glyph = count - 1
	}
	return u16Or0(table, int(glyph)*4)
}

func (m swashGlyphMetrics) advanceWidth(glyph uint16) float32 {
	if glyph >= m.glyphCount || m.hmtx == nil {
		return 0
	}
	return float32(longMetric(m.hmtx, m.hmtxCount, glyph))
}

func (m swashGlyphMetrics) advanceHeight(glyph uint16) float32 {
	if glyph >= m.glyphCount {
		return 0
	}
	if m.vmtx == nil {
		return m.synthesizedHeight
	}
	return float32(longMetric(m.vmtx, m.vmtxCount, glyph))
}
