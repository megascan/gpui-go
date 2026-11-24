package gpui

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"io"
	"sync"

	"golang.org/x/image/tiff/lzw"
)

const imageDecodeMaxInFlightBytes = imageDecodeMaxTotalBytes

type imageDecodeWaiter struct {
	n     uint64
	ready chan struct{}
}

type imageDecodeSemaphore struct {
	mu      sync.Mutex
	size    uint64
	used    uint64
	waiters []*imageDecodeWaiter
}

var imageDecodeMemory = &imageDecodeSemaphore{size: imageDecodeMaxInFlightBytes}

func (s *imageDecodeSemaphore) acquire(n uint64) func() {
	n = min(n, s.size)
	s.mu.Lock()
	if len(s.waiters) == 0 && s.size-s.used >= n {
		s.used += n
		s.mu.Unlock()
		return s.releaseFunc(n)
	}
	w := &imageDecodeWaiter{n: n, ready: make(chan struct{})}
	s.waiters = append(s.waiters, w)
	s.mu.Unlock()
	<-w.ready
	return s.releaseFunc(n)
}

func (s *imageDecodeSemaphore) releaseFunc(n uint64) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.used -= n
			for len(s.waiters) > 0 {
				w := s.waiters[0]
				if s.size-s.used < w.n {
					break
				}
				s.used += w.n
				s.waiters[0] = nil
				s.waiters = s.waiters[1:]
				close(w.ready)
			}
		})
	}
}

type jpegComponent struct {
	id, h, v uint64
}

type jpegFrameInfo struct {
	components  []jpegComponent
	progressive bool
	jfif        bool
	adobe       bool
	adobeXform  byte
}

func parseJpegFrameInfo(data []byte) (jpegFrameInfo, bool) {
	var info jpegFrameInfo
	i := 2
	for i+1 < len(data) {
		if data[i] != 0xff {
			i++
			continue
		}
		marker := data[i+1]
		switch {
		case marker == 0xff:
			i++
			continue
		case marker == 0x00 || marker == 0x01 || marker >= 0xd0 && marker <= 0xd7:
			i += 2
			continue
		case marker == 0xda || marker == 0xd9:
			return info, len(info.components) > 0
		}
		if i+4 > len(data) {
			return info, false
		}
		length := int(binary.BigEndian.Uint16(data[i+2:]))
		end := i + 2 + length
		if length < 2 || end > len(data) {
			return info, false
		}
		segment := data[i+4 : end]
		switch {
		case marker == 0xe0 && len(segment) >= 5 && string(segment[:5]) == "JFIF\x00":
			info.jfif = true
		case marker == 0xee && len(segment) >= 12 && string(segment[:5]) == "Adobe":
			info.adobe, info.adobeXform = true, segment[11]
		case marker == 0xc0 || marker == 0xc1 || marker == 0xc2:
			if len(info.components) > 0 || len(segment) < 6 {
				return info, false
			}
			count := int(segment[5])
			if len(segment) < 6+3*count || count == 0 {
				return info, false
			}
			info.progressive = marker == 0xc2
			for c := range count {
				hv := segment[7+3*c]
				comp := jpegComponent{id: uint64(segment[6+3*c]), h: uint64(hv >> 4), v: uint64(hv & 0x0f)}
				if comp.h == 0 || comp.v == 0 {
					return info, false
				}
				if count == 1 {
					comp.h, comp.v = 1, 1
				}
				info.components = append(info.components, comp)
			}
		}
		i = end
	}
	return info, false
}

func jpegDecodeBytes(data []byte, config image.Config) ([]uint64, bool) {
	info, ok := parseJpegFrameInfo(data)
	if !ok || config.Width <= 0 || config.Height <= 0 {
		return nil, false
	}
	width, height := uint64(config.Width), uint64(config.Height)
	var maxH, maxV uint64 = 1, 1
	for _, c := range info.components {
		maxH, maxV = max(maxH, c.h), max(maxV, c.v)
	}
	mxx := (width + 8*maxH - 1) / (8 * maxH)
	myy := (height + 8*maxV - 1) / (8 * maxV)
	mcus := mxx * myy
	full := 64 * maxH * maxV * mcus
	var sizes []uint64
	switch n := len(info.components); {
	case n == 1:
		sizes = append(sizes, 64*mcus)
	case n >= 3:
		c1, c2 := info.components[1], info.components[2]
		flex := c1.h != c2.h || c1.v != c2.v || info.components[0].h != maxH || info.components[0].v != maxV
		if !flex {
			switch (maxH/c1.h)<<4 | maxV/c1.v {
			case 0x11, 0x12, 0x21, 0x22, 0x41, 0x42:
			default:
				flex = true
			}
		}
		sizes = append(sizes, full)
		for _, c := range []jpegComponent{c1, c2} {
			if flex {
				sizes = append(sizes, full)
			} else {
				sizes = append(sizes, 64*c.h*c.v*mcus)
			}
		}
		if n == 4 {
			sizes = append(sizes, 64*info.components[3].h*info.components[3].v*mcus, 4*width*height)
		} else if !info.jfif && (info.adobe && info.adobeXform == 0 || info.components[0].id == 'R' && c1.id == 'G' && c2.id == 'B') {
			sizes = append(sizes, 4*width*height)
		}
	default:
		return nil, false
	}
	if info.progressive {
		for _, c := range info.components {
			sizes = append(sizes, 256*c.h*c.v*mcus)
		}
	}
	return sizes, true
}

type tiffIFD struct {
	order   binary.ByteOrder
	data    []byte
	entries map[uint16][]byte
	types   map[uint16]uint16
	counts  map[uint16]uint32
}

func parseTiffIFD(data []byte) (*tiffIFD, bool) {
	order, ok := tiffByteOrder(data)
	if !ok || len(data) < 8 {
		return nil, false
	}
	position := uint64(order.Uint32(data[4:8]))
	if position+2 > uint64(len(data)) {
		return nil, false
	}
	count := uint64(order.Uint16(data[position:]))
	position += 2
	if position+12*count > uint64(len(data)) {
		return nil, false
	}
	ifd := &tiffIFD{order: order, data: data, entries: map[uint16][]byte{}, types: map[uint16]uint16{}, counts: map[uint16]uint32{}}
	for range count {
		entry := data[position : position+12]
		position += 12
		tag, typ, n := order.Uint16(entry[0:2]), order.Uint16(entry[2:4]), order.Uint32(entry[4:8])
		var size uint64
		switch typ {
		case 1, 2, 6, 7:
			size = 1
		case 3, 8:
			size = 2
		case 4, 9:
			size = 4
		default:
			continue
		}
		total := size * uint64(n)
		value := entry[8:12]
		if total > 4 {
			offset := uint64(order.Uint32(entry[8:12]))
			if offset+total > uint64(len(data)) {
				return nil, false
			}
			value = data[offset : offset+total]
		}
		ifd.entries[tag], ifd.types[tag], ifd.counts[tag] = value, typ, n
	}
	return ifd, true
}

func (ifd *tiffIFD) values(tag uint16) []uint64 {
	value, ok := ifd.entries[tag]
	if !ok {
		return nil
	}
	n := ifd.counts[tag]
	out := make([]uint64, 0, min(n, 1<<16))
	for i := range uint64(n) {
		switch ifd.types[tag] {
		case 1, 2, 6, 7:
			out = append(out, uint64(value[i]))
		case 3, 8:
			out = append(out, uint64(ifd.order.Uint16(value[2*i:])))
		case 4, 9:
			out = append(out, uint64(ifd.order.Uint32(value[4*i:])))
		}
	}
	return out
}

func (ifd *tiffIFD) first(tag uint16, fallback uint64) uint64 {
	value, ok := ifd.entries[tag]
	if !ok || ifd.counts[tag] == 0 {
		return fallback
	}
	switch ifd.types[tag] {
	case 3, 8:
		return uint64(ifd.order.Uint16(value))
	case 4, 9:
		return uint64(ifd.order.Uint32(value))
	}
	return uint64(value[0])
}

const (
	tiffCompressionNone      = 1
	tiffCompressionLZW       = 5
	tiffCompressionDeflate   = 8
	tiffCompressionDeflateOl = 32946
	tiffCompressionPackBits  = 32773
)

func tiffPackBitsSize(data []byte, limit uint64) uint64 {
	var total uint64
	for i := 0; i < len(data) && total <= limit; {
		code := int(int8(data[i]))
		i++
		switch {
		case code >= 0:
			n := min(code+1, len(data)-i)
			total += uint64(n)
			i += n
		case code == -128:
		default:
			if i >= len(data) {
				return total
			}
			i++
			total += uint64(1 - code)
		}
	}
	return total
}

func tiffInflatedSize(compression uint64, data []byte, limit uint64) uint64 {
	var r io.Reader
	switch compression {
	case tiffCompressionLZW:
		lr := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
		defer lr.Close()
		r = lr
	case tiffCompressionDeflate, tiffCompressionDeflateOl:
		zr, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return 0
		}
		defer zr.Close()
		r = zr
	case tiffCompressionPackBits:
		return tiffPackBitsSize(data, limit)
	default:
		return 0
	}
	n, _ := io.Copy(io.Discard, io.LimitReader(r, int64(min(limit+1, 1<<62))))
	return uint64(n)
}

func tiffDecodeBytes(data []byte, config image.Config) ([]uint64, error) {
	width, height := uint64(config.Width), uint64(config.Height)
	ifd, ok := parseTiffIFD(data)
	if !ok {
		return []uint64{8 * width * height}, nil
	}
	samples := max(ifd.first(277, 1), 1)
	var bitsPerPixel uint64
	if bps := ifd.values(258); uint64(len(bps)) == samples {
		for _, b := range bps {
			bitsPerPixel += b
		}
	} else {
		bitsPerPixel = ifd.first(258, 1) * samples
	}
	blockWidth, blockHeight := width, height
	offsetsTag, countsTag := uint16(273), uint16(279)
	if tileWidth := ifd.first(322, 0); tileWidth != 0 {
		blockWidth, blockHeight = tileWidth, ifd.first(323, 0)
		offsetsTag, countsTag = 324, 325
	} else if rows := ifd.first(278, 0); rows > 0 && rows < height {
		blockHeight = rows
	}
	if blockWidth > 1<<24 || blockHeight > 1<<24 || bitsPerPixel > 1<<10 {
		return nil, errImageLimitsExceeded
	}
	blockBytes := (blockWidth*bitsPerPixel + 7) / 8 * blockHeight
	if blockBytes > imageDecodeMaxAllocBytes {
		return nil, errImageLimitsExceeded
	}
	compression := ifd.first(259, tiffCompressionNone)
	switch compression {
	case 0, tiffCompressionNone:
		return []uint64{blockBytes}, nil
	case tiffCompressionLZW, tiffCompressionDeflate, tiffCompressionDeflateOl, tiffCompressionPackBits:
	default:
		return []uint64{2 * blockBytes, blockBytes}, nil
	}
	offsets, counts := ifd.values(offsetsTag), ifd.values(countsTag)
	for i := range min(len(offsets), len(counts)) {
		start, n := offsets[i], counts[i]
		if start > uint64(len(data)) {
			continue
		}
		end := min(start+n, uint64(len(data)))
		if tiffInflatedSize(compression, data[start:end], blockBytes) > blockBytes {
			return nil, errImageLimitsExceeded
		}
	}
	return []uint64{2 * blockBytes, blockBytes}, nil
}

func webpBitstreamDims(fourCC string, data []byte) (uint32, uint32, bool) {
	switch fourCC {
	case "VP8L":
		if len(data) < 5 || data[0] != 0x2f {
			return 0, 0, false
		}
		header := binary.LittleEndian.Uint32(data[1:5])
		return header&0x3fff + 1, (header>>14)&0x3fff + 1, true
	case "VP8 ":
		if len(data) < 10 || data[3] != 0x9d || data[4] != 0x01 || data[5] != 0x2a {
			return 0, 0, false
		}
		return uint32(binary.LittleEndian.Uint16(data[6:8]) & 0x3fff), uint32(binary.LittleEndian.Uint16(data[8:10]) & 0x3fff), true
	}
	return 0, 0, false
}
