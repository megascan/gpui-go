package etagere

import (
	"fmt"
	"io"
	"math"
	"math/bits"
	"slices"
	"strconv"
)

const (
	binBits  = 12
	itemBits = 12
	genBits  = 8

	binMask  uint32 = (1 << binBits) - 1
	itemMask uint32 = ((1 << itemBits) - 1) << binBits
	genMask  uint32 = ((1 << genBits) - 1) << (binBits + itemBits)

	maxItemsPerBin uint16 = uint16(itemMask >> 12)
	maxBinCount           = int(binMask)
	maxShelfCount         = math.MaxUint16

	checksEnabled = false
)

type bucketIndex uint16

const bucketIndexInvalid bucketIndex = math.MaxUint16

type shelf struct {
	x, y        uint16
	height      uint16
	bucketWidth uint16
	firstBucket bucketIndex
}

type bucket struct {
	x          uint16
	freeSpace  uint16
	next       bucketIndex
	refcount   uint16
	itemCount  uint16
	shelf      uint16
	generation uint8
}

// BucketedAtlasAllocator is a shelf-packing dynamic atlas allocator that frees space per bucket of items.
type BucketedAtlasAllocator struct {
	shelves                []shelf
	buckets                []bucket
	availableHeight        uint16
	width                  uint16
	height                 uint16
	firstUnallocatedBucket bucketIndex
	flipXY                 bool
	alignment              Size
	currentColumn          uint16
	columnWidth            uint16
	numColumns             uint16
	allocatedSpace         int32
}

// NewBucketedAtlasAllocator creates an atlas allocator with default options.
func NewBucketedAtlasAllocator(size Size) *BucketedAtlasAllocator {
	return BucketedAtlasAllocatorWithOptions(size, DefaultOptions)
}

// BucketedAtlasAllocatorWithOptions creates an atlas allocator with the provided options.
func BucketedAtlasAllocatorWithOptions(size Size, options AllocatorOptions) *BucketedAtlasAllocator {
	assert(size.Width < math.MaxUint16, "size.width < u16::MAX as i32")
	assert(size.Height < math.MaxUint16, "size.height < u16::MAX as i32")

	var width, height, shelfAlignment uint16
	if options.VerticalShelves {
		width, height, shelfAlignment = uint16(size.Height), uint16(size.Width), uint16(options.Alignment.Height)
	} else {
		width, height, shelfAlignment = uint16(size.Width), uint16(size.Height), uint16(options.Alignment.Width)
	}

	columnWidth := width / uint16(options.NumColumns)
	columnWidth = columnWidth - columnWidth%shelfAlignment

	return &BucketedAtlasAllocator{
		availableHeight:        height,
		width:                  width,
		height:                 height,
		firstUnallocatedBucket: bucketIndexInvalid,
		flipXY:                 options.VerticalShelves,
		alignment:              options.Alignment,
		numColumns:             uint16(options.NumColumns),
		columnWidth:            columnWidth,
	}
}

// Clone returns a deep copy of the allocator.
func (a *BucketedAtlasAllocator) Clone() *BucketedAtlasAllocator {
	c := *a
	c.shelves = slices.Clone(a.shelves)
	c.buckets = slices.Clone(a.buckets)
	return &c
}

// Clear removes every allocation and keeps the current size.
func (a *BucketedAtlasAllocator) Clear() {
	a.shelves = a.shelves[:0]
	a.buckets = a.buckets[:0]
	a.firstUnallocatedBucket = bucketIndexInvalid
	a.availableHeight = a.height
	a.currentColumn = 0
	a.allocatedSpace = 0
}

// Size returns the current size of the atlas.
func (a *BucketedAtlasAllocator) Size() Size {
	w, h := convertCoordinates(a.flipXY, a.width, a.height)
	return Size2(int32(w), int32(h))
}

// Grow enlarges the atlas to newSize, which must not be smaller than the current size in either dimension.
func (a *BucketedAtlasAllocator) Grow(newSize Size) {
	assert(newSize.Width < math.MaxUint16, "new_size.width < u16::MAX as i32")
	assert(newSize.Height < math.MaxUint16, "new_size.height < u16::MAX as i32")

	var newWidth, newHeight uint16
	if a.flipXY {
		newWidth, newHeight = uint16(newSize.Height), uint16(newSize.Width)
	} else {
		newWidth, newHeight = uint16(newSize.Width), uint16(newSize.Height)
	}

	assert(newWidth >= a.width, "new_width >= self.width")
	assert(newHeight >= a.height, "new_height >= self.height")

	a.availableHeight += newHeight - a.height
	a.width = newWidth
	a.height = newHeight

	if a.numColumns == 1 {
		additionalWidth := a.width - a.columnWidth

		length := len(a.shelves)
		for shelfIndex := range length {
			s := a.shelves[shelfIndex]
			x := a.columnWidth
			bucketWidth := s.bucketWidth

			maxNewBuckets := uint16(maxBinCount - len(a.buckets))
			numBucketsToAdd := additionalWidth / bucketWidth
			numBucketsToAdd = min(numBucketsToAdd, maxNewBuckets)

			bucketNext := s.firstBucket

			for range numBucketsToAdd {
				bk := bucket{
					next:       bucketNext,
					x:          x,
					freeSpace:  bucketWidth,
					refcount:   0,
					shelf:      uint16(shelfIndex),
					generation: 0,
					itemCount:  0,
				}

				x += bucketWidth

				bucketNext = a.addBucket(bk)
			}

			a.shelves[shelfIndex].firstBucket = bucketNext
		}

		a.columnWidth = a.width
	} else {
		a.numColumns = a.width / a.columnWidth
	}
}

// IsEmpty reports whether nothing is allocated.
func (a *BucketedAtlasAllocator) IsEmpty() bool {
	return len(a.shelves) == 0
}

// Allocate reserves a rectangle of requestedSize and reports false when there is no room.
func (a *BucketedAtlasAllocator) Allocate(requestedSize Size) (Allocation, bool) {
	if requestedSize.IsEmpty() || requestedSize.Width > math.MaxUint16 || requestedSize.Height > math.MaxUint16 {
		return Allocation{}, false
	}

	adjustSize(a.alignment.Width, &requestedSize.Width)
	adjustSize(a.alignment.Height, &requestedSize.Height)

	if requestedSize.Width > int32(a.columnWidth) || requestedSize.Height > int32(a.height) {
		return Allocation{}, false
	}

	w, h := convertCoordinates(a.flipXY, uint16(requestedSize.Width), uint16(requestedSize.Height))

	var selectedShelf int
	selectedBucket := bucketIndexInvalid
	bestWaste := uint16(math.MaxUint16)

	canAddShelf := (a.availableHeight >= h || a.currentColumn+1 < a.numColumns) &&
		len(a.shelves) < maxShelfCount &&
		len(a.buckets) < maxBinCount

shelves:
	for shelfIndex := range a.shelves {
		s := a.shelves[shelfIndex]
		if s.height < h || s.bucketWidth < w {
			continue
		}

		yWaste := s.height - h
		if yWaste > bestWaste || (canAddShelf && yWaste > h) {
			continue
		}

		bucketIdx := s.firstBucket
		for bucketIdx != bucketIndexInvalid {
			bk := a.buckets[bucketIdx]

			if bk.freeSpace >= w && bk.itemCount < maxItemsPerBin {
				if yWaste == 0 && bk.freeSpace == w {
					selectedShelf = shelfIndex
					selectedBucket = bucketIdx

					break shelves
				}

				if yWaste < bestWaste {
					bestWaste = yWaste
					selectedShelf = shelfIndex
					selectedBucket = bucketIdx
					break
				}
			}

			bucketIdx = bk.next
		}
	}

	if selectedBucket == bucketIndexInvalid {
		if canAddShelf {
			selectedShelf = a.addShelf(w, h)
			selectedBucket = a.shelves[selectedShelf].firstBucket
		} else {
			selectedShelf, selectedBucket = a.coalesceShelves(w, h)
		}
	}

	if selectedBucket != bucketIndexInvalid {
		return a.allocFromBucket(selectedShelf, selectedBucket, w), true
	}

	return Allocation{}, false
}

// Deallocate frees the allocation id; space is reclaimed only when every item of its bucket is freed.
func (a *BucketedAtlasAllocator) Deallocate(id AllocID) {
	if a.deallocateFromBucket(id) {
		a.cleanupShelves()
	}

	a.check()
}

// AllocatedSpace returns the occupied area of the atlas.
func (a *BucketedAtlasAllocator) AllocatedSpace() int32 {
	return a.allocatedSpace
}

// FreeSpace returns the area available for future allocations.
func (a *BucketedAtlasAllocator) FreeSpace() int32 {
	return int32(a.width)*int32(a.height) - a.allocatedSpace
}

func (a *BucketedAtlasAllocator) allocFromBucket(shelfIndex int, bucketIdx bucketIndex, width uint16) Allocation {
	s := &a.shelves[shelfIndex]
	bk := &a.buckets[bucketIdx]

	assert(bk.freeSpace >= width, "bucket.free_space >= width")

	minX := bk.x + s.bucketWidth - bk.freeSpace
	minY := s.y
	maxX := minX + width
	maxY := minY + s.height

	minX, minY = convertCoordinates(a.flipXY, minX, minY)
	maxX, maxY = convertCoordinates(a.flipXY, maxX, maxY)

	bk.freeSpace -= width
	bk.refcount++
	bk.itemCount++

	id := AllocID(
		(uint32(bucketIdx) & binMask) |
			((uint32(bk.itemCount) << 12) & itemMask) |
			(uint32(bk.generation) << 24),
	)

	rectangle := Rectangle{
		Min: Point2(int32(minX), int32(minY)),
		Max: Point2(int32(maxX), int32(maxY)),
	}

	a.allocatedSpace += rectangle.Size().Area()

	a.check()

	return Allocation{ID: id, Rectangle: rectangle}
}

func (a *BucketedAtlasAllocator) addBucket(bk bucket) bucketIndex {
	bucketIdx := a.firstUnallocatedBucket

	if bucketIdx == bucketIndexInvalid {
		bucketIdx = bucketIndex(len(a.buckets))
		a.buckets = append(a.buckets, bk)
	} else {
		bk.generation = a.buckets[bucketIdx].generation + 1
		a.firstUnallocatedBucket = a.buckets[bucketIdx].next
		a.buckets[bucketIdx] = bk
	}

	return bucketIdx
}

func (a *BucketedAtlasAllocator) addShelf(width, height uint16) int {
	canAddColumn := a.currentColumn+1 < a.numColumns

	if a.availableHeight != 0 && a.availableHeight < height && canAddColumn {
		a.addShelf(0, a.availableHeight)
		assert(a.availableHeight == 0, "self.available_height == 0")
	}

	if a.availableHeight == 0 && canAddColumn {
		a.currentColumn++
		a.availableHeight = a.height
	}

	height = min(shelfHeight(height), a.availableHeight)
	numBuckets := a.numBuckets(width, height)
	bucketWidth := a.columnWidth / numBuckets
	bucketWidth = bucketWidth - (bucketWidth % uint16(a.alignment.Width))
	y := a.height - a.availableHeight
	a.availableHeight -= height

	shelfIndex := len(a.shelves)

	x := a.currentColumn * a.columnWidth
	bucketNext := bucketIndexInvalid
	for range numBuckets {
		bk := bucket{
			next:       bucketNext,
			x:          x,
			freeSpace:  bucketWidth,
			refcount:   0,
			shelf:      uint16(shelfIndex),
			generation: 0,
			itemCount:  0,
		}

		x += bucketWidth

		bucketNext = a.addBucket(bk)
	}

	a.shelves = append(a.shelves, shelf{
		x:           a.currentColumn * a.columnWidth,
		y:           y,
		height:      height,
		bucketWidth: bucketWidth,
		firstBucket: bucketNext,
	})

	return shelfIndex
}

func (a *BucketedAtlasAllocator) coalesceShelves(w, h uint16) (int, bucketIndex) {
	length := len(a.shelves)
	coalesceFound := false
	coalesceStart, coalesceEnd := 0, 0
	var coalescedHeight uint16

outer:
	for shelfIndex := range length {
		if a.shelves[shelfIndex].bucketWidth < w {
			continue
		}
		if !a.shelfIsEmpty(shelfIndex) {
			continue
		}
		shelfX := a.shelves[shelfIndex].x
		coalescedHeight = a.shelves[shelfIndex].height
		for i := 1; i < 3; i++ {
			if a.shelves[shelfIndex+i].x != shelfX {
				continue outer
			}

			if shelfIndex+i >= length {
				break outer
			}

			if !a.shelfIsEmpty(shelfIndex + i) {
				continue outer
			}

			coalescedHeight += a.shelves[shelfIndex+i].height

			if coalescedHeight >= h {
				coalesceFound = true
				coalesceStart, coalesceEnd = shelfIndex, shelfIndex+i+1
				break outer
			}
		}
	}

	if coalesceFound {
		yTop := a.shelves[coalesceStart].y + coalescedHeight
		for i := coalesceStart + 1; i < coalesceEnd; i++ {
			a.shelves[i].y = yTop
			a.shelves[i].height = 0
		}

		s := &a.shelves[coalesceStart]
		s.height = coalescedHeight

		return coalesceStart, s.firstBucket
	}

	return 0, bucketIndexInvalid
}

func (a *BucketedAtlasAllocator) numBuckets(width, height uint16) uint16 {
	var n uint16
	switch q := a.columnWidth / max(width, height); {
	case q <= 4:
		n = 1
	case q <= 16:
		n = 2
	case q <= 32:
		n = 4
	default:
		n = nextPowerOfTwo(q/16 - 1)
	}
	return min(n, uint16(maxBinCount-len(a.buckets)))
}

func (a *BucketedAtlasAllocator) deallocateFromBucket(id AllocID) bool {
	bucketIdx := int(uint32(id) & binMask)
	generation := uint8((uint32(id) & genMask) >> 24)

	bk := &a.buckets[bucketIdx]

	expectedGeneration := bk.generation
	if generation != expectedGeneration {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", generation, expectedGeneration))
	}

	assert(bk.refcount > 0, "bucket.refcount > 0")
	bk.refcount--

	s := a.shelves[bk.shelf]

	bucketIsEmpty := bk.refcount == 0
	if bucketIsEmpty {
		a.allocatedSpace -= int32(s.bucketWidth-bk.freeSpace) * int32(s.height)
		bk.freeSpace = s.bucketWidth
	}

	return bucketIsEmpty && int(bk.shelf) == len(a.shelves)-1
}

func (a *BucketedAtlasAllocator) cleanupShelves() {
	for len(a.shelves) > 0 {
		s := a.shelves[len(a.shelves)-1]
		bucketIdx := s.firstBucket
		lastBucket := s.firstBucket

		for bucketIdx != bucketIndexInvalid {
			bk := a.buckets[bucketIdx]

			if bk.refcount != 0 {
				return
			}

			lastBucket = bucketIdx
			bucketIdx = bk.next
		}

		assert(lastBucket != bucketIndexInvalid, "last_bucket != BucketIndex::INVALID")

		a.buckets[lastBucket].next = a.firstUnallocatedBucket
		a.firstUnallocatedBucket = s.firstBucket

		if s.y == 0 && a.currentColumn > 0 {
			a.currentColumn--
			prev := a.shelves[len(a.shelves)-2]
			a.availableHeight = a.height - (prev.y + prev.height)
		} else {
			a.availableHeight += s.height
		}

		a.shelves = a.shelves[:len(a.shelves)-1]
	}
}

func (a *BucketedAtlasAllocator) shelfIsEmpty(idx int) bool {
	bucketIdx := a.shelves[idx].firstBucket

	for bucketIdx != bucketIndexInvalid {
		bk := a.buckets[bucketIdx]

		if bk.refcount != 0 {
			return false
		}

		bucketIdx = bk.next
	}

	return true
}

// DumpSvg writes a visual representation of the atlas in SVG format.
func (a *BucketedAtlasAllocator) DumpSvg(output io.Writer) error {
	if _, err := fmt.Fprintf(output, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %s %s\">\n",
		svgFloat(float32(a.width)), svgFloat(float32(a.height))); err != nil {
		return err
	}

	if err := a.DumpIntoSvg(nil, output); err != nil {
		return err
	}

	_, err := io.WriteString(output, "</svg>\n")
	return err
}

// DumpIntoSvg writes the SVG body of the atlas without the document wrapper, scaled and translated into rect when it is not nil.
func (a *BucketedAtlasAllocator) DumpIntoSvg(rect *Rectangle, output io.Writer) error {
	sx, sy := float32(1), float32(1)
	tx, ty := float32(0), float32(0)

	if rect != nil {
		sx = float32(rect.Size().Width) / float32(a.width)
		sy = float32(rect.Size().Height) / float32(a.height)
		tx = float32(rect.Min.X)
		ty = float32(rect.Min.Y)
	}

	emit := func(x, y, w, h float32, r, g, b uint8) error {
		_, err := fmt.Fprintf(output, "    %s\n", svgRect(x, y, w, h, r, g, b))
		return err
	}

	if err := emit(tx, ty, float32(a.width)*sx, float32(a.height)*sy, 40, 40, 40); err != nil {
		return err
	}

	for _, s := range a.shelves {
		bucketIdx := s.firstBucket

		y := float32(s.y) * sy
		h := float32(s.height) * sy

		for bucketIdx != bucketIndexInvalid {
			bk := a.buckets[bucketIdx]

			x := float32(bk.x) * sx
			w := float32(s.bucketWidth-bk.freeSpace) * sx

			bx, by, bw, bh := x, y, w, h
			if a.flipXY {
				bx, by, bw, bh = y, x, h, w
			}
			if err := emit(bx+tx, by+ty, bw, bh, 70, 70, 180); err != nil {
				return err
			}

			if bk.freeSpace > 0 {
				xFree := x + w
				wFree := float32(bk.freeSpace) * sx

				fx, fy, fw, fh := xFree, y, wFree, h
				if a.flipXY {
					fx, fy, fw, fh = y, xFree, h, wFree
				}
				if err := emit(fx+tx, fy+ty, fw, fh, 50, 50, 50); err != nil {
					return err
				}
			}

			bucketIdx = bk.next
		}
	}

	return nil
}

func (a *BucketedAtlasAllocator) check() {
	if !checksEnabled {
		return
	}

	var h uint16
	for _, s := range a.shelves {
		h += s.height
	}
	h += a.availableHeight

	assert(h%a.height == 0, "h % self.height == 0")
	assert(h <= a.height*a.numColumns, "h <= self.height * self.num_columns")
	assert(h >= a.height, "h >= self.height")
	assert(a.IsEmpty() == (a.AllocatedSpace() == 0), "self.is_empty() == (self.allocated_space() == 0)")
}

func svgRect(x, y, w, h float32, r, g, b uint8) string {
	return fmt.Sprintf(
		`<rect x="%s" y="%s" width="%s" height="%s" ry="0" style="fill:rgb(%d,%d,%d);stroke:rgb(0,0,0);stroke-width:1;fill-opacity:1;stroke-opacity:1;" />`,
		svgFloat(x), svgFloat(y), svgFloat(w), svgFloat(h), r, g, b,
	)
}

func svgFloat(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func convertCoordinates(flipXY bool, x, y uint16) (uint16, uint16) {
	if flipXY {
		return y, x
	}
	return x, y
}

func shelfHeight(size uint16) uint16 {
	var alignment uint16
	switch {
	case size <= 31:
		alignment = 8
	case size <= 127:
		alignment = 16
	case size <= 511:
		alignment = 32
	default:
		alignment = 64
	}

	rem := size % alignment
	if rem > 0 {
		size += alignment - rem
	}

	return size
}

func adjustSize(alignment int32, size *int32) {
	rem := *size % alignment
	if rem > 0 {
		*size += alignment - rem
	}
}

func nextPowerOfTwo(n uint16) uint16 {
	if n <= 1 {
		return 1
	}
	return uint16(1) << bits.Len16(n-1)
}

func assert(ok bool, cond string) {
	if !ok {
		panic("assertion failed: " + cond)
	}
}
