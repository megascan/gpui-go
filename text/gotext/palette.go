package gotext

type swashColorPalettes struct {
	data []byte
	len  int
}

func newSwashColorPalettes(data []byte) swashColorPalettes {
	n, _ := u16At(data, 4)
	return swashColorPalettes{data: data, len: int(n)}
}

func (p swashColorPalettes) get(index int) (swashColorPalette, bool) {
	if index >= p.len {
		return swashColorPalette{}, false
	}
	b := p.data
	version, ok := u16At(b, 0)
	if !ok {
		return swashColorPalette{}, false
	}
	numEntries, ok := u16At(b, 2)
	if !ok {
		return swashColorPalette{}, false
	}
	offset, ok := u32At(b, 8)
	if !ok {
		return swashColorPalette{}, false
	}
	first, ok := u16At(b, 12+index*2)
	if !ok {
		return swashColorPalette{}, false
	}
	return swashColorPalette{
		data:       b,
		version:    version,
		index:      index,
		numEntries: numEntries,
		offset:     int(offset) + int(first)*4,
	}, true
}

type swashColorPalette struct {
	data       []byte
	version    uint16
	index      int
	numEntries uint16
	offset     int
}

func (p swashColorPalette) get(index uint16) [4]byte {
	if index >= p.numEntries {
		return [4]byte{}
	}
	offset := p.offset + int(index)*4
	b, _ := u8At(p.data, offset)
	g, _ := u8At(p.data, offset+1)
	r, _ := u8At(p.data, offset+2)
	a, _ := u8At(p.data, offset+3)
	return [4]byte{r, g, b, a}
}
