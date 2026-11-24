//go:build linux

package xcursor

import (
	"encoding/binary"
	"errors"
	"io"
)

// Image is one image of an Xcursor file.
type Image struct {
	Size       uint32
	Width      uint32
	Height     uint32
	Xhot       uint32
	Yhot       uint32
	Delay      uint32
	PixelsRGBA []byte
	PixelsARGB []byte
}

var errTagMismatch = errors.New("Tag mismatch")

type stream struct{ r io.ReadSeeker }

func (s stream) tag(tag [4]byte) error {
	var data [4]byte
	if _, err := io.ReadFull(s.r, data[:]); err != nil {
		return err
	}
	if data != tag {
		return errTagMismatch
	}
	return nil
}

func (s stream) u32() (uint32, error) {
	var data [4]byte
	if _, err := io.ReadFull(s.r, data[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(data[:]), nil
}

func (s stream) u32s(n int) ([]uint32, error) {
	out := make([]uint32, n)
	for i := range out {
		v, err := s.u32()
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func parseImg(s stream) (Image, error) {
	if err := s.tag([4]byte{0x24, 0, 0, 0}); err != nil {
		return Image{}, err
	}
	if err := s.tag([4]byte{0x02, 0x00, 0xfd, 0xff}); err != nil {
		return Image{}, err
	}
	size, err := s.u32()
	if err != nil {
		return Image{}, err
	}
	if err := s.tag([4]byte{1, 0, 0, 0}); err != nil {
		return Image{}, err
	}
	v, err := s.u32s(5)
	if err != nil {
		return Image{}, err
	}
	width, height, xhot, yhot, delay := v[0], v[1], v[2], v[3], v[4]
	if width > 0x7fff || height > 0x7fff {
		return Image{}, errors.New("Image too large")
	}
	if width == 0 || height == 0 {
		return Image{}, errors.New("Image with zero width or height")
	}
	if xhot > width || yhot > height {
		return Image{}, errors.New("Hotspot outside image")
	}
	rgba := make([]byte, 4*width*height)
	if _, err := io.ReadFull(s.r, rgba); err != nil {
		return Image{}, err
	}
	return Image{Size: size, Width: width, Height: height, Xhot: xhot, Yhot: yhot, Delay: delay, PixelsRGBA: rgba, PixelsARGB: rgbaToARGB(rgba)}, nil
}

func rgbaToARGB(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for i := 0; i+4 <= len(in); i += 4 {
		out = append(out, in[i+3], in[i], in[i+1], in[i+2])
	}
	return out
}

// ParseXcursorStream parses every image of an Xcursor file.
func ParseXcursorStream(input io.ReadSeeker) ([]Image, error) {
	s := stream{input}
	if err := s.tag([4]byte{'X', 'c', 'u', 'r'}); err != nil {
		return nil, err
	}
	header, err := s.u32()
	if err != nil {
		return nil, err
	}
	if _, err := s.u32(); err != nil {
		return nil, err
	}
	ntoc, err := s.u32()
	if err != nil {
		return nil, err
	}
	if _, err := input.Seek(int64(header), io.SeekStart); err != nil {
		return nil, err
	}
	var indices []uint32
	for range ntoc {
		toc, err := s.u32s(3)
		if err != nil {
			return nil, err
		}
		if toc[0] == 0xfffd_0002 {
			indices = append(indices, toc[2])
		}
	}
	imgs := make([]Image, 0, len(indices))
	for _, index := range indices {
		if _, err := input.Seek(int64(index), io.SeekStart); err != nil {
			return nil, err
		}
		img, err := parseImg(s)
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}

var errNoImages = errors.New("no images")

func findBestSize(images []Image, desiredSize uint32) (uint32, error) {
	distance := func(a, b uint32) uint32 { return max(a, b) - min(a, b) }
	var result uint32
	found := false
	for _, entry := range images {
		if !found || distance(entry.Size, desiredSize) < distance(result, desiredSize) {
			result, found = entry.Size, true
		}
	}
	if !found {
		return 0, errNoImages
	}
	return result, nil
}

func parseCursor(input io.ReadSeeker, desiredSize uint32) ([]Image, error) {
	cursors, err := ParseXcursorStream(input)
	if err != nil {
		return nil, err
	}
	size, err := findBestSize(cursors, desiredSize)
	if err != nil {
		return nil, err
	}
	kept := cursors[:0]
	for _, img := range cursors {
		if img.Size == size {
			kept = append(kept, img)
		}
	}
	return kept, nil
}
