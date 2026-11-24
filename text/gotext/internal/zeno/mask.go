package zeno

// Format is the output image format for rendering.
type Format struct {
	kind   formatKind
	offset [3]float32
}

type formatKind uint8

const (
	formatAlpha formatKind = iota
	formatSubpixel
	formatCustomSubpixel
)

var (
	// FormatAlpha is an 8-bit alpha mask.
	FormatAlpha = Format{kind: formatAlpha}
	// FormatSubpixel is a 32-bit RGBA subpixel mask with 1/3 pixel offsets for the red and blue channels.
	FormatSubpixel = Format{kind: formatSubpixel}
)

// FormatCustomSubpixel is a 32-bit RGBA subpixel mask with custom offsets.
func FormatCustomSubpixel(offsets [3]float32) Format {
	return Format{kind: formatCustomSubpixel, offset: offsets}
}

// FormatSubpixelBGRA creates a format for BGRA subpixel rendering.
func FormatSubpixelBGRA() Format {
	return FormatCustomSubpixel([3]float32{0.3, 0, -0.3})
}

// IsAlpha reports whether the format is the 8-bit alpha mask.
func (f Format) IsAlpha() bool { return f.kind == formatAlpha }

func (f Format) bytesPerPixel() uint32 {
	if f.kind == formatAlpha {
		return 1
	}
	return 4
}

// BufferSize returns the buffer size needed for an image of the given size.
func (f Format) BufferSize(width, height uint32) int {
	return int(width * height * f.bytesPerPixel())
}

// Mask is a builder for configuring and rendering a mask.
type Mask struct {
	data         PathData
	fill         Fill
	transform    *Transform
	format       Format
	origin       Origin
	offset       Vector
	renderOffset Vector
	width        uint32
	height       uint32
	explicitSize bool
	hasSize      bool
	boundsOffset Vector
	scratch      *Scratch
}

// NewMask creates a mask builder for the path data.
func NewMask(data PathData) *Mask {
	return &Mask{data: data}
}

// MaskWithScratch creates a mask builder for the path data and scratch memory.
func MaskWithScratch(data PathData, scratch *Scratch) *Mask {
	return &Mask{data: data, scratch: scratch}
}

// Style sets the fill rule. The default is non-zero.
func (m *Mask) Style(fill Fill) *Mask {
	m.fill = fill
	return m
}

// Transform sets the transformation matrix of the path.
func (m *Mask) Transform(transform *Transform) *Mask {
	m.transform = transform
	return m
}

// Format sets the format of the mask.
func (m *Mask) Format(format Format) *Mask {
	m.format = format
	return m
}

// Origin sets the origin of the mask's coordinate system.
func (m *Mask) Origin(origin Origin) *Mask {
	m.origin = origin
	return m
}

// Offset sets the offset for the path's rendered bounds.
func (m *Mask) Offset(offset Vector) *Mask {
	m.offset = offset
	return m
}

// Size sets an explicit size for the mask.
func (m *Mask) Size(width, height uint32) *Mask {
	m.width = width
	m.height = height
	m.explicitSize = true
	m.hasSize = true
	return m
}

// RenderOffset sets an additional offset applied only while rendering.
func (m *Mask) RenderOffset(offset Vector) *Mask {
	m.renderOffset = offset
	return m
}

// Inspect calls f with the format, width and height of the mask.
func (m *Mask) Inspect(f func(format Format, width, height uint32)) *Mask {
	m.ensureSize()
	f(m.format, m.width, m.height)
	return m
}

// RenderInto renders the mask into buffer; pitch 0 means tightly packed rows.
func (m *Mask) RenderInto(buffer []byte, pitch int) Placement {
	offset, placement := m.placement()
	if pitch == 0 {
		pitch = int(placement.Width * m.format.bytesPerPixel())
	}
	render(m, offset, placement, buffer, pitch)
	return placement
}

// Render renders the mask into a new buffer.
func (m *Mask) Render() ([]byte, Placement) {
	offset, placement := m.placement()
	buf := make([]byte, m.format.BufferSize(placement.Width, placement.Height))
	pitch := int(placement.Width * m.format.bytesPerPixel())
	render(m, offset, placement, buf, pitch)
	return buf, placement
}

func (m *Mask) ensureSize() {
	if m.hasSize {
		return
	}
	offset, placement := m.placement()
	m.boundsOffset = offset
	m.width = placement.Width
	m.height = placement.Height
	m.explicitSize = false
	m.hasSize = true
}

func (m *Mask) placement() (Vector, Placement) {
	placement := Placement{Width: m.width, Height: m.height}
	offset := m.offset
	if m.explicitSize {
		return offset, placement
	} else if !m.hasSize {
		var bounds Bounds
		if m.scratch != nil {
			bounds = m.scratch.Bounds(m.data, m.transform)
		} else {
			bounds = PathBounds(m.data, m.transform)
		}
		bounds.Min = bounds.Min.Add(m.offset).Floor()
		bounds.Max = bounds.Max.Add(m.offset).Ceil()
		offset = NewVector(-bounds.Min.X, -bounds.Min.Y)
		placement.Width = uint32(bounds.Width())
		placement.Height = uint32(bounds.Height())
	} else {
		offset = m.boundsOffset
	}
	placement.Left = int32(-offset.X)
	if m.origin == OriginBottomLeft {
		placement.Top = int32(NewVector(0, -offset.Y).Floor().Y + float32(m.height))
	} else {
		placement.Top = int32(-offset.Y)
	}
	return offset, placement
}

func render(m *Mask, offset Vector, placement Placement, buf []byte, pitch int) {
	yUp := m.origin == OriginBottomLeft
	var isSubpx bool
	var subpx [3]Vector
	switch m.format.kind {
	case formatSubpixel:
		isSubpx = true
		subpx = [3]Vector{NewVector(-0.3, 0), VectorZero, NewVector(0.3, 0)}
	case formatCustomSubpixel:
		isSubpx = true
		subpx = [3]Vector{NewVector(m.format.offset[0], 0), NewVector(m.format.offset[1], 0), NewVector(m.format.offset[2], 0)}
	}
	w := placement.Width
	h := placement.Height
	shift := offset.Add(m.renderOffset)
	scratch := m.scratch
	if scratch == nil {
		scratch = NewScratch()
	}
	ras := NewRasterizer(&scratch.render)
	apply := func(r *Rasterizer) {
		scratch.Apply(m.data, m.transform, r)
	}
	if isSubpx {
		for channel := range 3 {
			ras.RasterizeWrite(shift.Add(subpx[channel]), w, h, apply, m.fill, pitch, yUp, func(rowOffset, x, count int, c uint8) {
				row := buf[rowOffset:]
				j := x*4 + channel
				for range count {
					row[j] = c
					j += 4
				}
			})
		}
		return
	}
	ras.Rasterize(shift, w, h, apply, m.fill, buf, pitch, yUp)
}
