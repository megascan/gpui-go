package taffy

import "math"

const fitTolerance float32 = 0.001

// ContentSlot is an empty slot that avoids floats, suitable for non-floated content to be laid out into.
type ContentSlot struct {
	SegmentID Option[int]
	X         float32
	Y         float32
	Width     float32
	Height    float32
}

// BfcSlot is an empty slot that avoids floats, suitable for a box that establishes an independent formatting context.
type BfcSlot struct {
	SegmentID    Option[int]
	X            float32
	Y            float32
	BorderWidth  float32
	StretchWidth float32
}

// PlacedFloatedBox is a floated box.
type PlacedFloatedBox struct {
	Width  float32
	Height float32
	XInset float32
	Y      float32
}

type f32Range struct {
	start float32
	end   float32
}

func (r f32Range) contains(v float32) bool {
	return r.start <= v && v < r.end
}

type usizeRange struct {
	start int
	end   int
}

type segment struct {
	y        f32Range
	insets   [2]float32
	hasFloat [2]bool
}

func (s *segment) fitsFloatWidth(floatedBox Size[float32], direction FloatDirection, bfcWidth float32, cbInsets [2]float32) bool {
	return floatFitsHorizontally(floatedBox.Width, direction, bfcWidth, s.insets, cbInsets)
}

func floatFitsHorizontally(width float32, direction FloatDirection, bfcWidth float32, floatInsets [2]float32, cbInsets [2]float32) bool {
	lead := int(direction)
	trail := 1 - lead
	xInset := f32Max(floatInsets[lead], cbInsets[lead])
	fitsOppositeFloats := floatInsets[trail] == 0.0 || xInset+width <= bfcWidth-floatInsets[trail]+fitTolerance
	fitsContainingBlock := floatInsets[lead] == 0.0 || xInset+width <= bfcWidth-cbInsets[trail]+fitTolerance
	return fitsOppositeFloats && fitsContainingBlock
}

type floatFitter struct {
	bfcWidth    float32
	slotHeight  float64
	floatInsets [2]float32
	cbInsets    [2]float32
}

func newFloatFitter(bfcWidth float32, slotHeight float32, cbInsets [2]float32) floatFitter {
	return floatFitter{bfcWidth: bfcWidth, slotHeight: float64(slotHeight), floatInsets: [2]float32{0.0, 0.0}, cbInsets: cbInsets}
}

func (f *floatFitter) unionInsets(insets [2]float32) {
	f.floatInsets[0] = f32Max(f.floatInsets[0], insets[0])
	f.floatInsets[1] = f32Max(f.floatInsets[1], insets[1])
}

func (f *floatFitter) placedInset(direction FloatDirection) float32 {
	lead := int(direction)
	return f32Max(f.floatInsets[lead], f.cbInsets[lead])
}

func (f *floatFitter) fitsHoriontally(width float32, direction FloatDirection) bool {
	return floatFitsHorizontally(width, direction, f.bfcWidth, f.floatInsets, f.cbInsets)
}

func (f *floatFitter) addHeight(height float32) {
	f.slotHeight += float64(height)
}

func (f *floatFitter) fitsVertically(height float32) bool {
	return f.slotHeight >= float64(height)
}

// FloatContext is a context for placing floated boxes.
type FloatContext struct {
	availableWidth   float32
	hasFloats        bool
	leftFloats       []PlacedFloatedBox
	rightFloats      []PlacedFloatedBox
	segments         []segment
	lastPlacedFloats [2]usizeRange
	clearBottoms     [2]Option[float32]
	floatCeiling     Option[float32]
}

// NewFloatContext creates a new empty FloatContext.
func NewFloatContext() FloatContext {
	return FloatContext{}
}

// HasFloats reports whether the float context contains any floats.
func (c *FloatContext) HasFloats() bool {
	return c.hasFloats
}

// HasActiveFloats reports whether the float context contains any floats that extend to or below minY.
func (c *FloatContext) HasActiveFloats(minY float32) bool {
	lastEnd := float32(0.0)
	if n := len(c.segments); n > 0 {
		lastEnd = c.segments[n-1].y.end
	}
	return c.hasFloats && lastEnd > minY
}

// SetWidth sets the width of the FloatContext.
func (c *FloatContext) SetWidth(availableWidth float32) {
	c.availableWidth = availableWidth
}

// LeftFloats returns the placed left floats.
func (c *FloatContext) LeftFloats() []PlacedFloatedBox {
	return c.leftFloats
}

// RightFloats returns the placed right floats.
func (c *FloatContext) RightFloats() []PlacedFloatedBox {
	return c.rightFloats
}

func (c *FloatContext) subdivideSegment(idx int, divideAtY float32) {
	oldSegment := &c.segments[idx]
	newSegment := segment{insets: oldSegment.insets, hasFloat: oldSegment.hasFloat, y: f32Range{start: divideAtY, end: oldSegment.y.end}}
	if !oldSegment.y.contains(divideAtY) || oldSegment.y.start == divideAtY {
		panic("assertion failed: old_segment.y.contains(&divide_at_y) && old_segment.y.start != divide_at_y")
	}
	oldSegment.y.end = divideAtY

	c.segments = append(c.segments, segment{})
	copy(c.segments[idx+2:], c.segments[idx+1:])
	c.segments[idx+1] = newSegment
}

func (c *FloatContext) updateLastPlacedFloat(direction FloatDirection, placement usizeRange) {
	slot := int(direction)
	c.lastPlacedFloats[slot].start = max(c.lastPlacedFloats[slot].start, placement.start)
	c.lastPlacedFloats[slot].end = max(c.lastPlacedFloats[slot].end, placement.end)
}

// PlaceFloatedBox positions a floated box within the context, returning its (x, y) coordinates.
func (c *FloatContext) PlaceFloatedBox(floatedBox Size[float32], minY float32, containingBlockInsets [2]float32, direction FloatDirection, clear Clear) Point[float32] {
	c.hasFloats = true

	placedFloatedBox := c.placeFloatedBoxInner(floatedBox, minY, containingBlockInsets, direction, clear)

	slot := int(direction)
	bottom := placedFloatedBox.Y + placedFloatedBox.Height
	if b, ok := c.clearBottoms[slot].Get(); ok {
		c.clearBottoms[slot] = some(f32Max(b, bottom))
	} else {
		c.clearBottoms[slot] = some(bottom)
	}
	y := placedFloatedBox.Y
	if ceiling, ok := c.floatCeiling.Get(); ok {
		c.floatCeiling = some(f32Max(ceiling, y))
	} else {
		c.floatCeiling = some(y)
	}

	xInset := placedFloatedBox.XInset
	y = placedFloatedBox.Y
	switch direction {
	case FloatDirectionLeft:
		c.leftFloats = append(c.leftFloats, placedFloatedBox)
		return Point[float32]{X: xInset, Y: y}
	default:
		c.rightFloats = append(c.rightFloats, placedFloatedBox)
		return Point[float32]{X: c.availableWidth - xInset - floatedBox.Width, Y: y}
	}
}

func (c *FloatContext) firstSegmentEndingAfter(hwm int, minY float32) int {
	if hwm <= len(c.segments) {
		for i := hwm; i < len(c.segments); i++ {
			if c.segments[i].y.end > minY {
				return i
			}
		}
	}
	return len(c.segments)
}

func (c *FloatContext) placeFloatedBoxInner(floatedBox Size[float32], minY float32, containingBlockInsets [2]float32, direction FloatDirection, clear Clear) PlacedFloatedBox {
	slot := int(direction)
	negInf := float32(math.Inf(-1))

	minY = f32Max(f32Max(minY, c.floatCeiling.UnwrapOr(negInf)), c.ClearedThreshold(clear).UnwrapOr(negInf))

	floatStart := max(c.lastPlacedFloats[0].start, c.lastPlacedFloats[1].start)
	var hwm int
	switch clear {
	case ClearLeft:
		hwm = max(floatStart, c.lastPlacedFloats[0].end+1)
	case ClearRight:
		hwm = max(floatStart, c.lastPlacedFloats[1].end+1)
	case ClearBoth:
		hwm = max(c.lastPlacedFloats[0].end, c.lastPlacedFloats[1].end) + 1
	default:
		hwm = floatStart
	}

	startIdx := c.firstSegmentEndingAfter(hwm, minY)
	startY := minY
	endIdx := startIdx

	var start, end Option[int]
	var placedInset float32
outer:
	for {
		if startIdx >= len(c.segments) {
			start, end, placedInset = none[int](), none[int](), containingBlockInsets[slot]
			break outer
		}
		startSegment := &c.segments[startIdx]

		if !startSegment.fitsFloatWidth(floatedBox, direction, c.availableWidth, containingBlockInsets) {
			startIdx++
			endIdx = max(endIdx, startIdx)
			continue
		}

		startY = f32Max(startY, startSegment.y.start)
		availableHeight := startSegment.y.end - startY
		fitter := newFloatFitter(c.availableWidth, availableHeight, containingBlockInsets)
		fitter.unionInsets(startSegment.insets)

		for {
			if endIdx >= len(c.segments) {
				inset := fitter.placedInset(direction)
				start, end, placedInset = some(startIdx), none[int](), inset
				break outer
			}
			endSegment := &c.segments[endIdx]

			fitter.unionInsets(endSegment.insets)
			if !fitter.fitsHoriontally(floatedBox.Width, direction) {
				startIdx++
				endIdx = max(endIdx, startIdx)
				continue outer
			}

			if endIdx != startIdx {
				fitter.addHeight(endSegment.y.end - endSegment.y.start)
			}
			if !fitter.fitsVertically(floatedBox.Height) {
				endIdx++
				continue
			}

			inset := fitter.placedInset(direction)
			start, end, placedInset = some(startIdx), some(endIdx), inset
			break outer
		}
	}

	if floatedBox.Height == 0.0 {
		return PlacedFloatedBox{
			Width:  floatedBox.Width,
			Height: floatedBox.Height,
			Y:      startY,
			XInset: placedInset,
		}
	}

	if start.IsNone() {
		lastYEnd := c.lastSegmentEnd(0.0)
		if startY > lastYEnd {
			c.segments = append(c.segments, segment{y: f32Range{start: lastYEnd, end: startY}})
		}

		startY := f32Max(lastYEnd, startY)

		insets := containingBlockInsets
		insets[slot] += floatedBox.Width
		var hasFloat [2]bool
		hasFloat[slot] = true
		c.segments = append(c.segments, segment{y: f32Range{start: startY, end: startY + floatedBox.Height}, insets: insets, hasFloat: hasFloat})

		startIdx := len(c.segments) - 1
		endIdx := startIdx + 1
		c.updateLastPlacedFloat(direction, usizeRange{start: startIdx, end: endIdx})

		return PlacedFloatedBox{
			Width:  floatedBox.Width,
			Height: floatedBox.Height,
			Y:      startY,
			XInset: containingBlockInsets[slot],
		}
	}

	startIdx = start.Unwrap()

	if startY != c.segments[startIdx].y.start {
		c.subdivideSegment(startIdx, startY)
		startIdx++
		if e, ok := end.Get(); ok {
			end = some(e + 1)
		}
	}

	if e, ok := end.Get(); !ok {
		lastYEnd := c.lastSegmentEnd(0.0)
		if minY > lastYEnd {
			c.segments = append(c.segments, segment{y: f32Range{start: lastYEnd, end: minY}})
		}
		endIdx = len(c.segments) - 1
	} else {
		endIdx = e
		endY := startY + floatedBox.Height

		for endIdx > startIdx && endY <= c.segments[endIdx].y.start {
			endIdx--
		}

		if c.segments[endIdx].y.start < endY && endY < c.segments[endIdx].y.end {
			c.subdivideSegment(endIdx, endY)
		}
	}

	placedInsetPlusWidth := placedInset + floatedBox.Width
	for i := startIdx; i <= endIdx; i++ {
		c.segments[i].insets[slot] = placedInsetPlusWidth
		c.segments[i].hasFloat[slot] = true
	}

	c.updateLastPlacedFloat(direction, usizeRange{start: startIdx, end: endIdx + 1})

	return PlacedFloatedBox{Width: floatedBox.Width, Height: floatedBox.Height, Y: startY, XInset: placedInset}
}

func (c *FloatContext) lastSegmentEnd(def float32) float32 {
	if n := len(c.segments); n > 0 {
		return c.segments[n-1].y.end
	}
	return def
}

func (c *FloatContext) clearedSegment(clear Clear) Option[int] {
	leftEnd := c.lastPlacedFloats[0].end
	rightEnd := c.lastPlacedFloats[1].end
	switch {
	case clear == ClearLeft && leftEnd > 0:
		return some(leftEnd)
	case clear == ClearRight && rightEnd > 0:
		return some(rightEnd)
	case clear == ClearBoth && (leftEnd > 0 || rightEnd > 0):
		return some(max(leftEnd, rightEnd))
	default:
		return none[int]()
	}
}

// ClearedThreshold returns the bottom of the lowest relevant float for the clear property.
func (c *FloatContext) ClearedThreshold(clear Clear) Option[float32] {
	switch clear {
	case ClearLeft:
		return c.clearBottoms[0]
	case ClearRight:
		return c.clearBottoms[1]
	case ClearBoth:
		l, lok := c.clearBottoms[0].Get()
		r, rok := c.clearBottoms[1].Get()
		if lok && rok {
			return some(f32Max(l, r))
		}
		return c.clearBottoms[0].Or(c.clearBottoms[1])
	default:
		return none[float32]()
	}
}

func (c *FloatContext) searchHwm(clear Clear, after Option[int]) int {
	atLeast := 0
	if idx, ok := after.Get(); ok {
		atLeast = idx + 1
	}
	cleared := 0
	if idx, ok := c.clearedSegment(clear).Get(); ok {
		cleared = idx + 1
	}
	return max(atLeast, cleared)
}

// FindContentSlot searches for a space suitable for laying out non-floated content into.
func (c *FloatContext) FindContentSlot(minY float32, containingBlockInsets [2]float32, clear Clear, after Option[int]) ContentSlot {
	inf := float32(math.Inf(1))
	if !c.HasActiveFloats(minY) {
		return ContentSlot{
			SegmentID: none[int](),
			X:         containingBlockInsets[0],
			Y:         minY,
			Width:     c.availableWidth - containingBlockInsets[0] - containingBlockInsets[1],
			Height:    inf,
		}
	}

	minY = f32Max(minY, c.ClearedThreshold(clear).UnwrapOr(float32(math.Inf(-1))))

	hwm := c.searchHwm(clear, after)

	startIdx := c.firstSegmentEndingAfter(hwm, minY)
	if startIdx < len(c.segments) {
		seg := &c.segments[startIdx]
		insetLeft := f32Max(seg.insets[0], containingBlockInsets[0])
		insetRight := f32Max(seg.insets[1], containingBlockInsets[1])
		return ContentSlot{
			SegmentID: some(startIdx),
			X:         insetLeft,
			Y:         f32Max(seg.y.start, minY),
			Width:     c.availableWidth - insetLeft - insetRight,
			Height:    inf,
		}
	}
	return ContentSlot{
		SegmentID: none[int](),
		X:         containingBlockInsets[0],
		Y:         minY,
		Width:     c.availableWidth - containingBlockInsets[0] - containingBlockInsets[1],
		Height:    inf,
	}
}

// FindBfcSlot searches for a space suitable for laying out a box that establishes an independent formatting context.
func (c *FloatContext) FindBfcSlot(minY float32, containingBlockInsets [2]float32, margins [2]float32, direction Direction, clear Clear, after Option[int]) BfcSlot {
	marginInsets := [2]float32{containingBlockInsets[0] + margins[0], containingBlockInsets[1] + margins[1]}
	noFloatWidth := c.availableWidth - marginInsets[0] - marginInsets[1]
	noFloatSlot := BfcSlot{
		SegmentID:    none[int](),
		X:            marginInsets[0],
		Y:            minY,
		BorderWidth:  noFloatWidth,
		StretchWidth: noFloatWidth,
	}

	if !c.HasActiveFloats(minY) {
		return noFloatSlot
	}

	minY = f32Max(minY, c.ClearedThreshold(clear).UnwrapOr(float32(math.Inf(-1))))

	hwm := c.searchHwm(clear, after)

	startIdx := c.firstSegmentEndingAfter(hwm, minY)
	if startIdx < len(c.segments) {
		seg := &c.segments[startIdx]
		lead := 0
		if direction == DirectionRtl {
			lead = 1
		}
		trail := 1 - lead
		hasLeadFloat := seg.hasFloat[lead]
		hasTrailFloat := seg.hasFloat[trail]
		var fitInsets, stretchInsets [2]float32
		if hasLeadFloat {
			fitInsets[lead] = f32Max(seg.insets[lead], marginInsets[lead])
		} else {
			fitInsets[lead] = marginInsets[lead]
		}
		stretchInsets[lead] = fitInsets[lead]
		if hasTrailFloat {
			fitInsets[trail] = f32Max(seg.insets[trail], containingBlockInsets[trail])
		} else {
			fitInsets[trail] = f32Min(marginInsets[trail], containingBlockInsets[trail])
		}
		if hasTrailFloat {
			stretchInsets[trail] = f32Max(seg.insets[trail], marginInsets[trail])
		} else {
			stretchInsets[trail] = marginInsets[trail]
		}
		return BfcSlot{
			SegmentID:    some(startIdx),
			X:            fitInsets[0],
			Y:            f32Max(seg.y.start, minY),
			BorderWidth:  c.availableWidth - fitInsets[0] - fitInsets[1],
			StretchWidth: c.availableWidth - stretchInsets[0] - stretchInsets[1],
		}
	}
	slot := noFloatSlot
	slot.Y = f32Max(c.lastSegmentEnd(minY), minY)
	return slot
}

// FloatIntrinsicWidthCalculator computes the intrinsic width contribution of a set of floats.
type FloatIntrinsicWidthCalculator struct {
	availableWidth AvailableSpace
	sideSums       [2]float32
	contribution   float32
	widest         float32
}

// NewFloatIntrinsicWidthCalculator creates a new FloatIntrinsicWidthCalculator.
func NewFloatIntrinsicWidthCalculator(availableWidth AvailableSpace) FloatIntrinsicWidthCalculator {
	return FloatIntrinsicWidthCalculator{availableWidth: availableWidth}
}

// AddFloat adds a float to the computation.
func (c *FloatIntrinsicWidthCalculator) AddFloat(width float32, direction FloatDirection, clear Clear) {
	if c.availableWidth.Kind() == AvailableSpaceKindMinContent {
		c.contribution = f32Max(c.contribution, width)
	} else {
		if clear == ClearLeft || clear == ClearBoth {
			c.sideSums[0] = 0.0
		}
		if clear == ClearRight || clear == ClearBoth {
			c.sideSums[1] = 0.0
		}
		c.sideSums[int(direction)] += width
		c.contribution = f32Max(c.contribution, c.sideSums[0]+c.sideSums[1])
	}
	c.widest = f32Max(c.widest, width)
}

// Result returns the computed float contribution to intrinsic width.
func (c *FloatIntrinsicWidthCalculator) Result() float32 {
	if availableWidth, ok := c.availableWidth.IntoOption().Get(); ok {
		return f32Max(f32Min(c.contribution, availableWidth), c.widest)
	}
	return c.contribution
}
