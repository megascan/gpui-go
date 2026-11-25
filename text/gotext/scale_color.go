package gotext

import (
	"encoding/binary"

	ot "github.com/go-text/typesetting/font/opentype"
)

var (
	tagCOLR = ot.MustNewTag("COLR")
	tagCPAL = ot.MustNewTag("CPAL")
)

func u8At(data []byte, offset int) (uint8, bool) {
	if offset < 0 || offset >= len(data) {
		return 0, false
	}
	return data[offset], true
}

func u32At(data []byte, offset int) (uint32, bool) {
	if offset < 0 || offset+4 > len(data) {
		return 0, false
	}
	return binary.BigEndian.Uint32(data[offset:]), true
}

type swashColorProxy struct {
	colr []byte
	cpal []byte
}

func newSwashColorProxy(tables rawTables) swashColorProxy {
	return swashColorProxy{colr: tables.colr, cpal: tables.cpal}
}

type swashLayers struct {
	data   []byte
	offset int
	len    uint16
}

type swashLayer struct {
	glyphID       uint16
	colorIndex    uint16
	hasColorIndex bool
}

func (p swashColorProxy) layers(glyphID uint16) (swashLayers, bool) {
	b := p.colr
	if len(b) == 0 {
		return swashLayers{}, false
	}
	count16, ok := u16At(b, 2)
	if !ok {
		return swashLayers{}, false
	}
	base32, ok := u32At(b, 4)
	if !ok {
		return swashLayers{}, false
	}
	count, baseOffset := int(count16), int(base32)
	l, h := 0, count
	for l < h {
		i := l + (h-l)/2
		rec := baseOffset + i*6
		id, ok := u16At(b, rec)
		if !ok {
			return swashLayers{}, false
		}
		switch {
		case glyphID < id:
			h = i
		case glyphID > id:
			l = i + 1
		default:
			first, ok := u16At(b, rec+2)
			if !ok {
				return swashLayers{}, false
			}
			layerBase, ok := u32At(b, 8)
			if !ok {
				return swashLayers{}, false
			}
			n, ok := u16At(b, rec+4)
			if !ok {
				return swashLayers{}, false
			}
			return swashLayers{data: b, offset: int(layerBase) + int(first)*4, len: n}, true
		}
	}
	return swashLayers{}, false
}

func (p swashColorProxy) palette(index uint16) (swashColorPalette, bool) {
	if len(p.cpal) != 0 {
		return newSwashColorPalettes(p.cpal).get(int(index))
	}
	return swashColorPalette{}, false
}

func (l swashLayers) get(index uint16) (swashLayer, bool) {
	base := l.offset + int(index)*4
	glyphID, ok := u16At(l.data, base)
	if !ok {
		return swashLayer{}, false
	}
	colorIndex, ok := u16At(l.data, base+2)
	if !ok {
		return swashLayer{}, false
	}
	return swashLayer{glyphID: glyphID, colorIndex: colorIndex, hasColorIndex: colorIndex != 0xFFFF}, true
}
