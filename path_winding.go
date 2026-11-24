package gpui

import (
	"fmt"
	"iter"
	"log/slog"
	"math"
	"slices"
	"sync"

	"github.com/megascan/gpui-go/gpuiutil"
)

const pathWindingMargin float32 = 0.5 + 1.0/256.0

func fastMin(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func fastMax(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func fastClamp(v, lo, hi float32) float32 {
	return fastMin(fastMax(v, lo), hi)
}

func floorF32(v float32) float32 {
	return float32(math.Floor(float64(v)))
}

func sqrtF32(v float32) float32 {
	return float32(math.Sqrt(float64(v)))
}

func isFiniteF32(v float32) bool {
	return !math.IsInf(float64(v), 0) && !math.IsNaN(float64(v))
}

func saturatingF32ToUsize(v float32) int {
	u := saturatingF64ToU64(float64(v))
	if u > math.MaxInt {
		return math.MaxInt
	}
	return int(u)
}

type pathTiles struct {
	tileBounds Bounds[ScaledPixels]
	tileList   []PathTile
	curveList  []PathCurve
}

func (t *pathTiles) bounds() Bounds[ScaledPixels] {
	return t.tileBounds
}

func (t *pathTiles) paintedBounds() Bounds[ScaledPixels] {
	if len(t.tileList) == 0 {
		return t.tileBounds
	}
	return t.tileBounds.Dilate(ScaledPixels(pathWindingMargin))
}

func (t *pathTiles) tiles() []PathTile {
	return t.tileList
}

func (t *pathTiles) curves() []PathCurve {
	return t.curveList
}

func (t *pathTiles) retainVisible(mask Bounds[ScaledPixels]) {
	t.tileList = slices.DeleteFunc(t.tileList, func(tile PathTile) bool {
		return !(tile.TopLeft.X < float32(mask.Right()) &&
			tile.TopLeft.Y < float32(mask.Bottom()) &&
			tile.BottomRight.X > float32(mask.Origin.X) &&
			tile.BottomRight.Y > float32(mask.Origin.Y))
	})
}

var pathBinnerPool = sync.Pool{New: func() any { return &pathBinner{} }}

func bin(curves []PathCurve, evenOdd bool, scale float32) *pathTiles {
	if gpuiutil.DebugAssertions && !(scale > 0.0 && isFiniteF32(scale)) {
		panic(fmt.Sprintf("invalid scale %s", rustDisplayF32(scale)))
	}
	if len(curves) == 0 {
		return &pathTiles{}
	}
	binner := pathBinnerPool.Get().(*pathBinner)
	defer pathBinnerPool.Put(binner)
	minPoint, maxPoint, length := binner.scale(curves, scale)
	grid, ok := pathGridCovering(minPoint, maxPoint, len(curves), length)
	if !ok {
		slog.Warn(fmt.Sprintf("path spanning %s to %s device pixels overflows; skipping", minPoint.GoString(), maxPoint.GoString()))
		return &pathTiles{}
	}
	binner.findSpans(&grid)
	binner.fillCells()
	tiles, binned := binner.mergeCells(&grid, evenOdd)
	device := func(p Point[float32]) Point[ScaledPixels] {
		return Point[ScaledPixels]{X: ScaledPixels(p.X), Y: ScaledPixels(p.Y)}
	}
	return &pathTiles{
		tileBounds: BoundsFromCorners(device(minPoint), device(maxPoint)),
		tileList:   tiles,
		curveList:  binned,
	}
}

func pathCellSize(curves float32, length float32, size Point[float32]) float32 {
	const listingCPUCost float32 = 320.0
	const emptyCellCPUCost float32 = 95.0
	const minCellSize float32 = 16.0
	const maxGridCells float32 = 4096.0

	gridCells := func(cellSize float32) float32 {
		return float32((floorF32(size.X/cellSize) + 1.0) * (floorF32(size.Y/cellSize) + 1.0))
	}
	extent := fastMax(size.X, size.Y)
	cellSize := minCellSize
	for gridCells(cellSize) > maxGridCells {
		cellSize *= 2.0
	}
	bestCost := float32(math.Inf(1))
	bestSize := cellSize
	for {
		var listings float32
		if cellSize >= extent {
			listings = curves
		} else {
			listings = curves + length/cellSize
		}
		area := float32(fastMin(cellSize, size.X)*fastMin(cellSize, size.Y)) / 2.0
		emptyCells := fastMax(gridCells(cellSize)-listings, 0.0)
		cost := float32(listings*(listingCPUCost+area)) + float32(emptyCells*emptyCellCPUCost)
		if cost < bestCost {
			bestCost = cost
			bestSize = cellSize
		}
		if cellSize >= extent {
			return bestSize
		}
		cellSize *= 2.0
	}
}

func fills(winding int32, evenOdd bool) bool {
	if evenOdd {
		return winding%2 != 0
	}
	return winding != 0
}

type pathBinner struct {
	input    []PathCurve
	spans    []pathSpan
	cellEnds []int
	steps    []int32
	keys     []uint64
	runs     []pathRun
	above    []pathRun
}

type pathSpan struct {
	cell  int
	count int
	key   uint64
}

func subslice[T any](s []T, lo, hi int) ([]T, bool) {
	if lo < 0 || hi < lo || hi > len(s) {
		return nil, false
	}
	return s[lo:hi], true
}

func (b *pathBinner) scale(curves []PathCurve, scale float32) (Point[float32], Point[float32], float32) {
	minPoint := Point[float32]{X: float32(math.Inf(1)), Y: float32(math.Inf(1))}
	maxPoint := Point[float32]{X: float32(math.Inf(-1)), Y: float32(math.Inf(-1))}
	var length float32
	b.input = b.input[:0]
	for i := range curves {
		curve := curves[i].scaled(scale)
		minPoint.X = fastMin(minPoint.X, curve.minX())
		maxPoint.X = fastMax(maxPoint.X, curve.maxX())
		minPoint.Y = fastMin(minPoint.Y, curve.P0.Y)
		maxPoint.Y = fastMax(maxPoint.Y, curve.P1.Y)
		length += curve.maxX() - curve.minX() + curve.P1.Y - curve.P0.Y
		b.input = append(b.input, curve)
	}
	return minPoint, maxPoint, length
}

func (b *pathBinner) findSpans(grid *pathGrid) {
	cells := grid.rows * grid.columns
	b.spans = b.spans[:0]
	b.cellEnds = slices.Grow(b.cellEnds[:0], cells)[:cells]
	clear(b.cellEnds)
	b.steps = slices.Grow(b.steps[:0], cells)[:cells]
	clear(b.steps)
	for index := range b.input {
		curve := &b.input[index]
		key := uint64(orderedBits(curve.minX()))<<32 | uint64(index)
		lastRow := grid.rowOf(curve.P1.Y + pathWindingMargin)
		for row := grid.rowOf(curve.P0.Y - pathWindingMargin); row <= lastRow; row++ {
			rowTop := grid.y(row)
			low, high := curve.xRange(rowTop-pathWindingMargin, grid.y(row+1)+pathWindingMargin)
			first := grid.columnOf(low - pathWindingMargin)
			last := grid.columnOf(high + pathWindingMargin)
			cell := row*grid.columns + first
			count := last + 1 - first
			b.spans = append(b.spans, pathSpan{cell: cell, count: count, key: key})
			if listings, ok := subslice(b.cellEnds, cell, cell+count); ok {
				for i := range listings {
					listings[i]++
				}
			}

			if curve.crosses(rowTop) && last+1 < grid.columns && cell+count >= 0 && cell+count < len(b.steps) {
				b.steps[cell+count] += curve.step()
			}
		}
	}
}

func (b *pathBinner) fillCells() {
	total := 0
	for i, count := range b.cellEnds {
		b.cellEnds[i] = total
		total += count
	}
	b.keys = slices.Grow(b.keys[:0], total)[:total]
	clear(b.keys)
	for _, span := range b.spans {
		cursors, ok := subslice(b.cellEnds, span.cell, span.cell+span.count)
		if !ok {
			continue
		}
		for i := range cursors {
			if cursors[i] >= 0 && cursors[i] < len(b.keys) {
				b.keys[cursors[i]] = span.key
			}
			cursors[i]++
		}
	}
	start := 0
	for _, end := range b.cellEnds {
		if list, ok := subslice(b.keys, start, end); ok {
			slices.Sort(list)
		}
		start = end
	}
}

func (b *pathBinner) mergeCells(grid *pathGrid, evenOdd bool) ([]PathTile, []PathCurve) {
	input := b.input
	steps := b.steps
	cellEnds := b.cellEnds
	keys := b.keys
	var tiles []PathTile
	curves := make([]PathCurve, 0, len(keys))
	b.above = b.above[:0]
	cellStart := 0
	for row := 0; row < grid.rows; row++ {
		b.runs = b.runs[:0]
		var backdrop int32
		for column := 0; column < grid.columns; column++ {
			index := row*grid.columns + column
			if index < len(steps) {
				backdrop += steps[index]
			}
			cellEnd := cellStart
			if index < len(cellEnds) {
				cellEnd = cellEnds[index]
			}
			list := [2]int{cellStart, cellEnd}
			cellStart = cellEnd

			cell := grid.cell(column, row)
			var topLeft, bottomRight Point[float32]
			var ok bool
			if list[0] == list[1] {
				if fills(backdrop, evenOdd) {
					topLeft, bottomRight, ok = cell.topLeft(), cell.bottomRight(), true
				}
			} else {
				measure := newPathMeasure(cell, backdrop)
				for curve := range listed(keys, input, list) {
					measure.add(curve)
				}
				topLeft, bottomRight, ok = measure.quad(evenOdd)
			}
			if !ok {
				continue
			}
			run := pathRun{
				start:       column,
				end:         column + 1,
				backdrop:    backdrop,
				list:        list,
				topLeft:     topLeft,
				bottomRight: bottomRight,
				tile:        0,
			}
			if n := len(b.runs); n > 0 && b.runs[n-1].end == run.start && b.runs[n-1].shadesLike(&run, keys) {
				last := &b.runs[n-1]
				last.end = run.end
				extendQuad(&last.topLeft, &last.bottomRight, &run)
			} else {
				b.runs = append(b.runs, run)
			}
		}

		aboveIndex := 0
		for r := range b.runs {
			run := &b.runs[r]
			for aboveIndex < len(b.above) && b.above[aboveIndex].start < run.start {
				aboveIndex++
			}
			matched := false
			if aboveIndex < len(b.above) {
				other := &b.above[aboveIndex]
				if other.start == run.start && other.end == run.end && other.shadesLike(run, keys) {
					matched = true
					aboveIndex++
					if other.tile < len(tiles) {
						tile := &tiles[other.tile]
						extendQuad(&tile.TopLeft, &tile.BottomRight, run)
					}
					run.tile = other.tile
				}
			}
			if !matched {
				curveStart := len(curves)
				curvesTopLeft := Point[float32]{X: float32(math.Inf(1)), Y: float32(math.Inf(1))}
				curvesBottomRight := Point[float32]{X: float32(math.Inf(-1)), Y: float32(math.Inf(-1))}
				for curve := range listed(keys, input, run.list) {
					const boxMargin float32 = pathWindingMargin + 1.0/16.0

					curvesTopLeft.X = fastMin(curvesTopLeft.X, curve.minX()-boxMargin)
					curvesTopLeft.Y = fastMin(curvesTopLeft.Y, curve.P0.Y-boxMargin)
					curvesBottomRight.X = fastMax(curvesBottomRight.X, curve.maxX()+boxMargin)
					curvesBottomRight.Y = fastMax(curvesBottomRight.Y, curve.P1.Y+boxMargin)
					curves = append(curves, *curve)
				}
				tiles = append(tiles, PathTile{
					Paint:             0,
					CurveStart:        uint32(curveStart),
					CurveCount:        uint32(len(curves) - curveStart),
					TopLeft:           run.topLeft,
					BottomRight:       run.bottomRight,
					Reference:         grid.cell(run.start, row).topLeft(),
					Backdrop:          run.backdrop,
					CurvesTopLeft:     curvesTopLeft,
					CurvesBottomRight: curvesBottomRight,
				})
				run.tile = len(tiles) - 1
			}
		}
		b.above, b.runs = b.runs, b.above
	}
	return tiles, curves
}

func orderedBits(x float32) uint32 {
	bits := math.Float32bits(x)
	if bits>>31 == 1 {
		return ^bits
	}
	return bits | 1<<31
}

type pathRun struct {
	start       int
	end         int
	backdrop    int32
	list        [2]int
	topLeft     Point[float32]
	bottomRight Point[float32]
	tile        int
}

func (r *pathRun) shadesLike(other *pathRun, keys []uint64) bool {
	if r.backdrop != other.backdrop {
		return false
	}
	a, aok := subslice(keys, r.list[0], r.list[1])
	b, bok := subslice(keys, other.list[0], other.list[1])
	if aok != bok {
		return false
	}
	return !aok || slices.Equal(a, b)
}

func extendQuad(topLeft *Point[float32], bottomRight *Point[float32], run *pathRun) {
	topLeft.X = fastMin(topLeft.X, run.topLeft.X)
	topLeft.Y = fastMin(topLeft.Y, run.topLeft.Y)
	bottomRight.X = fastMax(bottomRight.X, run.bottomRight.X)
	bottomRight.Y = fastMax(bottomRight.Y, run.bottomRight.Y)
}

func listed(keys []uint64, input []PathCurve, list [2]int) iter.Seq[*PathCurve] {
	return func(yield func(*PathCurve) bool) {
		selected, _ := subslice(keys, list[0], list[1])
		for _, key := range selected {
			if index := int(uint32(key)); index < len(input) {
				if !yield(&input[index]) {
					return
				}
			}
		}
	}
}

type pathGrid struct {
	left     float32
	top      float32
	cellSize float32
	columns  int
	rows     int
}

func pathGridCovering(minPoint, maxPoint Point[float32], curves int, length float32) (pathGrid, bool) {
	left := floorF32(minPoint.X - pathWindingMargin)
	top := floorF32(minPoint.Y - pathWindingMargin)
	size := Point[float32]{X: maxPoint.X + pathWindingMargin - left, Y: maxPoint.Y + pathWindingMargin - top}
	if !(isFiniteF32(size.X) && isFiniteF32(size.Y)) {
		return pathGrid{}, false
	}
	cellSize := pathCellSize(float32(curves), length, size)
	return pathGrid{
		left:     left,
		top:      top,
		cellSize: cellSize,
		columns:  saturatingF32ToUsize(size.X/cellSize) + 1,
		rows:     saturatingF32ToUsize(size.Y/cellSize) + 1,
	}, true
}

func (g *pathGrid) columnOf(x float32) int {
	column := (x - g.left) / g.cellSize
	return int(saturatingF32ToU32(fastClamp(column, 0.0, float32(g.columns-1))))
}

func (g *pathGrid) rowOf(y float32) int {
	row := (y - g.top) / g.cellSize
	return int(saturatingF32ToU32(fastClamp(row, 0.0, float32(g.rows-1))))
}

func (g *pathGrid) x(column int) float32 {
	return g.left + float32(float32(column)*g.cellSize)
}

func (g *pathGrid) y(row int) float32 {
	return g.top + float32(float32(row)*g.cellSize)
}

func (g *pathGrid) cell(column, row int) pathCell {
	return pathCell{
		left:   g.x(column),
		top:    g.y(row),
		right:  g.x(column + 1),
		bottom: g.y(row + 1),
	}
}

type pathCell struct {
	left   float32
	top    float32
	right  float32
	bottom float32
}

func (c pathCell) topLeft() Point[float32] {
	return Point[float32]{X: c.left, Y: c.top}
}

func (c pathCell) bottomRight() Point[float32] {
	return Point[float32]{X: c.right, Y: c.bottom}
}

type pathMeasure struct {
	cell          pathCell
	backdrop      int32
	min           Point[float32]
	max           Point[float32]
	rightWinding  int32
	bottomWinding int32
}

func newPathMeasure(cell pathCell, backdrop int32) pathMeasure {
	return pathMeasure{
		cell:          cell,
		backdrop:      backdrop,
		min:           Point[float32]{X: float32(math.Inf(1)), Y: float32(math.Inf(1))},
		max:           Point[float32]{X: float32(math.Inf(-1)), Y: float32(math.Inf(-1))},
		rightWinding:  backdrop,
		bottomWinding: backdrop,
	}
}

func (m *pathMeasure) add(curve *PathCurve) {
	cell := m.cell
	m.min.X = fastMin(m.min.X, curve.minX())
	m.max.X = fastMax(m.max.X, curve.maxX())
	m.min.Y = fastMin(m.min.Y, curve.P0.Y)
	m.max.Y = fastMax(m.max.Y, curve.P1.Y)
	if curve.crosses(cell.top) {
		m.rightWinding += curve.step()
	}
	for _, endpoint := range curve.endpoints() {
		if endpoint.point.X <= cell.left && endpoint.point.Y > cell.top {
			m.bottomWinding -= endpoint.step
		}
	}
}

func (m *pathMeasure) quad(evenOdd bool) (Point[float32], Point[float32], bool) {
	cell := m.cell
	clampX := func(x float32) float32 { return fastClamp(x, cell.left, cell.right) }
	clampY := func(y float32) float32 { return fastClamp(y, cell.top, cell.bottom) }
	unfilled := !fills(m.backdrop, evenOdd)
	left := cell.left
	if unfilled {
		left = clampX(m.min.X - pathWindingMargin)
	}
	top := cell.top
	if unfilled {
		top = clampY(m.min.Y - pathWindingMargin)
	}
	right := clampX(m.max.X + pathWindingMargin)
	if fills(m.rightWinding, evenOdd) {
		right = cell.right
	}
	bottom := clampY(m.max.Y + pathWindingMargin)
	if fills(m.bottomWinding, evenOdd) {
		bottom = cell.bottom
	}
	if left < right && top < bottom {
		return Point[float32]{X: left, Y: top}, Point[float32]{X: right, Y: bottom}, true
	}
	return Point[float32]{}, Point[float32]{}, false
}

func (c *PathCurve) scaled(scale float32) PathCurve {
	scalePoint := func(p Point[float32]) Point[float32] {
		return Point[float32]{X: p.X * scale, Y: p.Y * scale}
	}
	return PathCurve{
		P0:   scalePoint(c.P0),
		P1:   scalePoint(c.P1),
		A:    scalePoint(c.A),
		B:    scalePoint(c.B),
		Sign: c.Sign,
		Pad:  0,
	}
}

func (c *PathCurve) minX() float32 {
	return fastMin(c.P0.X, c.P1.X)
}

func (c *PathCurve) maxX() float32 {
	return fastMax(c.P0.X, c.P1.X)
}

func (c *PathCurve) step() int32 {
	if c.Sign < 0.0 {
		return -1
	}
	return 1
}

func (c *PathCurve) crosses(y float32) bool {
	return c.P0.Y <= y && y < c.P1.Y
}

type pathEndpoint struct {
	point Point[float32]
	step  int32
}

func (c *PathCurve) endpoints() [2]pathEndpoint {
	return [2]pathEndpoint{{point: c.P0, step: c.step()}, {point: c.P1, step: -c.step()}}
}

func (c *PathCurve) xRange(top, bottom float32) (float32, float32) {
	top = fastMax(top, c.P0.Y)
	bottom = fastMin(bottom, c.P1.Y)
	xTop := c.P0.X
	if top > c.P0.Y {
		xTop = c.xAtY(top)
	}
	xBottom := c.P1.X
	if bottom < c.P1.Y {
		xBottom = c.xAtY(bottom)
	}
	return fastMax(fastMin(xTop, xBottom), c.minX()), fastMin(fastMax(xTop, xBottom), c.maxX())
}

func (c *PathCurve) xAtY(y float32) float32 {
	if c.A.Y == 0.0 {
		if c.B.Y == 0.0 {
			return c.P0.X
		}
		t := fastClamp((y-c.P0.Y)/c.B.Y, 0.0, 1.0)
		return float32(float32(float32(c.A.X*t)+c.B.X)*t) + c.P0.X
	}
	t := monotoneQuadraticRoot(c.A.Y, c.B.Y, c.P0.Y-y, 1.0)
	return float32(float32(float32(c.A.X*t)+c.B.X)*t) + c.P0.X
}

func monotoneQuadraticRoot(a, b, c, direction float32) float32 {
	discriminant := fastMax(float32(b*b)-float32(float32(4.0*a)*c), 0.0)
	denominator := b + float32(direction*sqrtF32(discriminant))
	if denominator == 0.0 {
		return 0.0
	}
	return fastClamp(float32(-2.0*c)/denominator, 0.0, 1.0)
}
