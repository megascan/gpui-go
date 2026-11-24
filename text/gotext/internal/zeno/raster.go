package zeno

const (
	pixelBits int32 = 8
	onePixel  int32 = 1 << pixelBits
)

func coverage(fill Fill, coverage int32) uint8 {
	coverage >>= pixelBits*2 + 1 - 8
	if fill == FillEvenOdd {
		coverage &= 511
		if coverage >= 256 {
			coverage = 511 - coverage
		}
	} else {
		if coverage < 0 {
			coverage = ^coverage
		}
		if coverage >= 256 {
			coverage = 255
		}
	}
	return uint8(coverage)
}

// Rasterizer accumulates path coverage into cells.
type Rasterizer struct {
	storage *HeapStorage
	xmin    int32
	xmax    int32
	ymin    int32
	ymax    int32
	height  int32
	shift   Vector
	start   FixedPoint
	closed  bool
	current Point
	x       int32
	y       int32
	px      int32
	py      int32
	cover   int32
	area    int32
	invalid bool
}

// NewRasterizer creates a rasterizer over the storage.
func NewRasterizer(storage *HeapStorage) *Rasterizer {
	return &Rasterizer{storage: storage}
}

func (r *Rasterizer) begin(shift Vector, width, height uint32, apply func(*Rasterizer)) {
	w := int32(width)
	h := int32(height)
	r.storage.Reset(FixedPoint{}, FixedPoint{X: w, Y: h})
	r.shift = shift
	r.start = FixedPoint{}
	r.closed = true
	r.current = VectorZero
	r.xmin = 0
	r.ymin = 0
	r.xmax = w
	r.ymax = h
	r.height = h
	r.x = 0
	r.y = 0
	r.px = 0
	r.py = 0
	r.invalid = true
	apply(r)
	if !r.closed {
		r.lineToFixed(r.start)
	}
	if !r.invalid {
		r.storage.Set(r.x, r.y, r.area, r.cover)
	}
}

// Rasterize renders the path into an 8-bit buffer.
func (r *Rasterizer) Rasterize(shift Vector, width, height uint32, apply func(*Rasterizer), fill Fill, buffer []byte, pitch int, yUp bool) {
	r.RasterizeWrite(shift, width, height, apply, fill, pitch, yUp, func(rowOffset, x, count int, c uint8) {
		row := buffer[rowOffset:]
		for i := x; i < x+count; i++ {
			row[i] = c
		}
	})
}

// RasterizeWrite renders the path and hands each covered span to write.
func (r *Rasterizer) RasterizeWrite(shift Vector, width, height uint32, apply func(*Rasterizer), fill Fill, pitch int, yUp bool, write func(rowOffset, x, count int, coverage uint8)) {
	r.begin(shift, width, height, apply)
	indices := r.storage.Indices()
	cells := r.storage.Cells()
	min := FixedPoint{X: r.xmin, Y: r.ymin}
	max := FixedPoint{X: r.xmax, Y: r.ymax}
	h := int(height)
	for i, index := range indices {
		if index == -1 {
			continue
		}
		y := int(int32(i) - min.Y)
		var rowOffset int
		if yUp {
			rowOffset = pitch * (h - 1 - y)
		} else {
			rowOffset = pitch * y
		}
		x := min.X
		var cover int32
		for {
			cell := &cells[index]
			if cover != 0 && cell.x > x {
				write(rowOffset, int(x), int(cell.x-x), coverage(fill, cover))
			}
			cover += cell.cover * (onePixel * 2)
			area := cover - cell.area
			if area != 0 && cell.x >= min.X {
				write(rowOffset, int(cell.x), 1, coverage(fill, area))
			}
			x = cell.x + 1
			index = cell.next
			if index == -1 {
				break
			}
		}
		if cover != 0 {
			write(rowOffset, int(x), int(max.X-x), coverage(fill, cover))
		}
	}
}

func (r *Rasterizer) setCell(x, y int32) {
	if !r.invalid && (r.area != 0 || r.cover != 0) {
		r.storage.Set(r.x, r.y, r.area, r.cover)
	}
	r.area = 0
	r.cover = 0
	if x > r.xmin-1 {
		r.x = x
	} else {
		r.x = r.xmin - 1
	}
	r.y = y
	r.invalid = y >= r.ymax || y < r.ymin || x >= r.xmax
}

func (r *Rasterizer) moveToFixed(to FixedPoint) {
	r.setCell(trunc(to.X), trunc(to.Y))
	r.px = to.X
	r.py = to.Y
}

func udiv(a, b int32) int32 {
	return int32((uint64(a) * uint64(b)) >> (4*8 - pixelBits))
}

func (r *Rasterizer) lineToFixed(to FixedPoint) {
	toX := to.X
	toY := to.Y
	ey1 := trunc(r.py)
	ey2 := trunc(toY)
	if (ey1 >= r.ymax && ey2 >= r.ymax) || (ey1 < r.ymin && ey2 < r.ymin) {
		r.px = toX
		r.py = toY
		return
	}
	ex1 := trunc(r.px)
	ex2 := trunc(toX)
	fx1 := fract(r.px)
	fy1 := fract(r.py)
	dx := toX - r.px
	dy := toY - r.py
	switch {
	case ex1 == ex2 && ey1 == ey2:
	case dy == 0:
		r.setCell(ex2, ey2)
		r.px = toX
		r.py = toY
		return
	case dx == 0:
		if dy > 0 {
			for {
				fy2 := onePixel
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * fx1 * 2
				fy1 = 0
				ey1++
				r.setCell(ex1, ey1)
				if ey1 == ey2 {
					break
				}
			}
		} else {
			for {
				var fy2 int32
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * fx1 * 2
				fy1 = onePixel
				ey1--
				r.setCell(ex1, ey1)
				if ey1 == ey2 {
					break
				}
			}
		}
	default:
		prod := dx*fy1 - dy*fx1
		var dxR, dyR int32
		if ex1 != ex2 {
			dxR = 0x00FFFFFF / dx
		}
		if ey1 != ey2 {
			dyR = 0x00FFFFFF / dy
		}
		for {
			switch {
			case prod <= 0 && prod-dx*onePixel > 0:
				var fx2 int32
				fy2 := udiv(-prod, -dxR)
				prod -= dy * onePixel
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * (fx1 + fx2)
				fx1 = onePixel
				fy1 = fy2
				ex1--
			case prod-dx*onePixel <= 0 && prod-dx*onePixel+dy*onePixel > 0:
				prod -= dx * onePixel
				fx2 := udiv(-prod, dyR)
				fy2 := onePixel
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * (fx1 + fx2)
				fx1 = fx2
				fy1 = 0
				ey1++
			case prod-dx*onePixel+dy*onePixel <= 0 && prod+dy*onePixel >= 0:
				prod += dy * onePixel
				fx2 := onePixel
				fy2 := udiv(prod, dxR)
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * (fx1 + fx2)
				fx1 = 0
				fy1 = fy2
				ex1++
			default:
				fx2 := udiv(prod, -dyR)
				var fy2 int32
				prod += dx * onePixel
				r.cover += fy2 - fy1
				r.area += (fy2 - fy1) * (fx1 + fx2)
				fx1 = fx2
				fy1 = onePixel
				ey1--
			}
			r.setCell(ex1, ey1)
			if ex1 == ex2 && ey1 == ey2 {
				break
			}
		}
	}
	fx2 := fract(toX)
	fy2 := fract(toY)
	r.cover += fy2 - fy1
	r.area += (fy2 - fy1) * (fx1 + fx2)
	r.px = toX
	r.py = toY
}

func absI32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func (r *Rasterizer) quadToFixed(control, to FixedPoint) {
	var arc [16*2 + 1]FixedPoint
	arc[0] = to
	arc[1] = control
	arc[2] = FixedPoint{X: r.px, Y: r.py}
	if (trunc(arc[0].Y) >= r.ymax && trunc(arc[1].Y) >= r.ymax && trunc(arc[2].Y) >= r.ymax) ||
		(trunc(arc[0].Y) < r.ymin && trunc(arc[1].Y) < r.ymin && trunc(arc[2].Y) < r.ymin) {
		r.px = arc[0].X
		r.py = arc[0].Y
		return
	}
	dx := absI32(arc[2].X + arc[0].X - 2*arc[1].X)
	dy := absI32(arc[2].Y + arc[0].Y - 2*arc[1].Y)
	if dx < dy {
		dx = dy
	}
	var draw int32 = 1
	for dx > onePixel/4 {
		dx >>= 2
		draw <<= 1
	}
	a := 0
	for {
		split := draw & -draw
		for {
			split >>= 1
			if split == 0 {
				break
			}
			splitQuad(arc[a:])
			a += 2
		}
		r.lineToFixed(arc[a])
		draw--
		if draw == 0 {
			break
		}
		a -= 2
	}
}

func (r *Rasterizer) curveToFixed(control1, control2, to FixedPoint) {
	var arc [16*8 + 1]FixedPoint
	arc[0] = to
	arc[1] = control2
	arc[2] = control1
	arc[3] = FixedPoint{X: r.px, Y: r.py}
	if (trunc(arc[0].Y) >= r.ymax && trunc(arc[1].Y) >= r.ymax && trunc(arc[2].Y) >= r.ymax && trunc(arc[3].Y) >= r.ymax) ||
		(trunc(arc[0].Y) < r.ymin && trunc(arc[1].Y) < r.ymin && trunc(arc[2].Y) < r.ymin && trunc(arc[3].Y) < r.ymin) {
		r.px = arc[0].X
		r.py = arc[0].Y
		return
	}
	a := 0
	for {
		if absI32(2*arc[a].X-3*arc[a+1].X+arc[a+3].X) > onePixel/2 ||
			absI32(2*arc[a].Y-3*arc[a+1].Y+arc[a+3].Y) > onePixel/2 ||
			absI32(arc[a].X-3*arc[a+2].X+2*arc[a+3].X) > onePixel/2 ||
			absI32(arc[a].Y-3*arc[a+2].Y+2*arc[a+3].Y) > onePixel/2 {
			buf := arc[a:]
			if len(buf) >= 7 {
				splitCubic(buf)
				a += 3
				continue
			}
			r.lineToFixed(to)
			return
		}
		r.lineToFixed(arc[a])
		if a == 0 {
			return
		}
		a -= 3
	}
}

// CurrentPoint returns the current point including the shift.
func (r *Rasterizer) CurrentPoint() Point {
	return r.current.Add(r.shift)
}

// MoveTo begins a subpath, closing the previous one.
func (r *Rasterizer) MoveTo(to Point) {
	if !r.closed {
		r.lineToFixed(r.start)
	}
	p := FixedPointFromPoint(to.Add(r.shift))
	r.moveToFixed(p)
	r.closed = false
	r.start = p
	r.current = to
}

// LineTo adds a line.
func (r *Rasterizer) LineTo(to Point) {
	r.current = to
	r.closed = false
	r.lineToFixed(FixedPointFromPoint(to.Add(r.shift)))
}

// QuadTo adds a quadratic curve.
func (r *Rasterizer) QuadTo(control, to Point) {
	r.current = to
	r.closed = false
	r.quadToFixed(FixedPointFromPoint(control.Add(r.shift)), FixedPointFromPoint(to.Add(r.shift)))
}

// CurveTo adds a cubic curve.
func (r *Rasterizer) CurveTo(control1, control2, to Point) {
	r.current = to
	r.closed = false
	r.curveToFixed(FixedPointFromPoint(control1.Add(r.shift)), FixedPointFromPoint(control2.Add(r.shift)), FixedPointFromPoint(to.Add(r.shift)))
}

// Close closes the subpath.
func (r *Rasterizer) Close() {
	r.lineToFixed(r.start)
	r.closed = true
}

// Cell is one accumulated coverage cell.
type Cell struct {
	x     int32
	cover int32
	area  int32
	next  int32
}

// HeapStorage is heap-backed cell storage.
type HeapStorage struct {
	min     FixedPoint
	max     FixedPoint
	cells   []Cell
	indices []int32
}

// Reset clears the storage for a new raster of the given extent.
func (s *HeapStorage) Reset(min, max FixedPoint) {
	s.min = min
	s.max = max
	s.cells = s.cells[:0]
	n := int(max.Y - min.Y)
	if cap(s.indices) < n {
		s.indices = make([]int32, n)
	}
	s.indices = s.indices[:n]
	for i := range s.indices {
		s.indices[i] = -1
	}
}

// Cells returns the cells.
func (s *HeapStorage) Cells() []Cell { return s.cells }

// Indices returns the first cell index of each row.
func (s *HeapStorage) Indices() []int32 { return s.indices }

// Set accumulates area and cover into the cell at (x, y).
func (s *HeapStorage) Set(x, y, area, cover int32) {
	yindex := int(y - s.min.Y)
	cellIndex := s.indices[yindex]
	lastIndex := int32(-1)
	for cellIndex != -1 {
		cell := &s.cells[cellIndex]
		if cell.x > x {
			break
		} else if cell.x == x {
			cell.area += area
			cell.cover += cover
			return
		}
		lastIndex = cellIndex
		cellIndex = cell.next
	}
	newIndex := int32(len(s.cells))
	if lastIndex != -1 {
		s.cells[lastIndex].next = newIndex
	} else {
		s.indices[yindex] = newIndex
	}
	s.cells = append(s.cells, Cell{x: x, area: area, cover: cover, next: cellIndex})
}

func splitQuad(base []FixedPoint) {
	base[4].X = base[2].X
	a := base[0].X + base[1].X
	b := base[1].X + base[2].X
	base[3].X = b >> 1
	base[2].X = (a + b) >> 2
	base[1].X = a >> 1
	base[4].Y = base[2].Y
	a = base[0].Y + base[1].Y
	b = base[1].Y + base[2].Y
	base[3].Y = b >> 1
	base[2].Y = (a + b) >> 2
	base[1].Y = a >> 1
}

func splitCubic(base []FixedPoint) {
	base[6].X = base[3].X
	a := base[0].X + base[1].X
	b := base[1].X + base[2].X
	c := base[2].X + base[3].X
	base[5].X = c >> 1
	c += b
	base[4].X = c >> 2
	base[1].X = a >> 1
	a += b
	base[2].X = a >> 2
	base[3].X = (a + c) >> 3
	base[6].Y = base[3].Y
	a = base[0].Y + base[1].Y
	b = base[1].Y + base[2].Y
	c = base[2].Y + base[3].Y
	base[5].Y = c >> 1
	c += b
	base[4].Y = c >> 2
	base[1].Y = a >> 1
	a += b
	base[2].Y = a >> 2
	base[3].Y = (a + c) >> 3
}

// FixedPoint is a point in 24.8 fixed point.
type FixedPoint struct {
	X, Y int32
}

// FixedPointFromPoint converts a point to fixed point.
func FixedPointFromPoint(p Point) FixedPoint {
	return FixedPoint{X: toFixed(p.X), Y: toFixed(p.Y)}
}

func toFixed(v float32) int32 {
	return int32(v * 256)
}

func trunc(x int32) int32 {
	return x >> pixelBits
}

func fract(x int32) int32 {
	return x & (onePixel - 1)
}
