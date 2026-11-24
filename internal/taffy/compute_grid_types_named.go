package taffy

import (
	"fmt"
	"iter"
	"slices"
	"sort"
	"strings"
)

type areaLines struct {
	rowStart    uint32
	rowEnd      uint32
	columnStart uint32
	columnEnd   uint32
}

func (a *areaLines) line(axis gridAreaAxis, end gridAreaEnd) Option[uint32] {
	line := *a.lineMut(axis, end)
	if line != 0 {
		return some(line)
	}
	return none[uint32]()
}

func (a *areaLines) lineMut(axis gridAreaAxis, end gridAreaEnd) *uint32 {
	switch {
	case axis == gridAreaAxisRow && end == gridAreaEndStart:
		return &a.rowStart
	case axis == gridAreaAxisRow && end == gridAreaEndEnd:
		return &a.rowEnd
	case axis == gridAreaAxisColumn && end == gridAreaEndStart:
		return &a.columnStart
	default:
		return &a.columnEnd
	}
}

type gridAreasMap[S CheapCloneStr] = map[S]*areaLines

type namedGridLinesMap[S CheapCloneStr] = map[S][]uint32

func splitAreaEdgeName(name string) (string, gridAreaEnd, bool) {
	if areaName, ok := strings.CutSuffix(name, "-start"); ok {
		return areaName, gridAreaEndStart, true
	}
	if areaName, ok := strings.CutSuffix(name, "-end"); ok {
		return areaName, gridAreaEndEnd, true
	}
	return "", 0, false
}

type namedLineResolverAxis[S CheapCloneStr] struct {
	lines              namedGridLinesMap[S]
	areas              gridAreasMap[S]
	axis               gridAreaAxis
	explicitTrackCount uint16
}

type lineNamePair[S CheapCloneStr] struct {
	line uint32
	name S
}

type namedLineResolver[S CheapCloneStr] struct {
	rowLines            namedGridLinesMap[S]
	columnLines         namedGridLinesMap[S]
	areas               gridAreasMap[S]
	areaColumnCount     uint16
	areaRowCount        uint16
	explicitColumnCount uint16
	explicitRowCount    uint16
	columnLineNamePairs []lineNamePair[S]
	rowLineNamePairs    []lineNamePair[S]
}

func upsertLineNameMap[S CheapCloneStr](m namedGridLinesMap[S], areas gridAreasMap[S], axis gridAreaAxis, key S, value uint32) {
	var areaLine uint32
	if areaName, end, ok := splitAreaEdgeName(string(key)); ok {
		if _, exists := areas[S(areaName)]; !exists {
			areas[S(areaName)] = &areaLines{}
		}
		edge := areas[S(areaName)].lineMut(axis, end)
		areaLine = *edge
		if *edge == 0 {
			*edge = value
		} else {
			*edge = min(*edge, value)
		}
	}

	if lines, ok := m[key]; ok {
		m[key] = append(lines, value)
	} else if areaLine == 0 {
		m[key] = []uint32{value}
	} else {
		m[key] = []uint32{areaLine, value}
	}
}

func (r namedLineResolverAxis[S]) areaLine(areaName string, end gridAreaEnd) Option[uint32] {
	area, ok := r.areas[S(areaName)]
	if !ok {
		return none[uint32]()
	}
	return area.line(r.axis, end)
}

func nonNamedFromResolved[S CheapCloneStr](p GridPlacement[S]) nonNamedGridPlacement {
	switch p.Kind() {
	case GridPlacementKindAuto:
		return GenericGridPlacementAuto[GridLine]()
	case GridPlacementKindLine:
		return GenericGridPlacementLine(p.Line())
	case GridPlacementKindSpan:
		return GenericGridPlacementSpan[GridLine](p.Span())
	case GridPlacementKindNamedSpan:
		return GenericGridPlacementSpan[GridLine](1)
	default:
		panic("internal error: entered unreachable code")
	}
}

func (r namedLineResolverAxis[S]) resolveLineNames(line Line[GridPlacement[S]]) Line[nonNamedGridPlacement] {
	startLineResolved := line.Start
	if line.Start.Kind() == GridPlacementKindNamedLine {
		startLineResolved = GridPlacementLine[S](r.findNamedLine(line.Start.Name(), line.Start.Index(), gridAreaEndStart))
	}

	endLineResolved := line.End
	if line.End.Kind() == GridPlacementKindNamedLine {
		endLineResolved = GridPlacementLine[S](r.findNamedLine(line.End.Name(), line.End.Index(), gridAreaEndEnd))
	}

	switch {
	case startLineResolved.Kind() == GridPlacementKindLine && endLineResolved.Kind() == GridPlacementKindNamedSpan && startLineResolved.Line().AsI16() != 0:
		startLine := startLineResolved.Line()
		afterStartLine := startLine.intoOriginZeroLine(r.explicitTrackCount).AddU16(1)
		endLine := r.nthLineForwards(r.linesNamed(string(endLineResolved.Name())), afterStartLine, int32(max(endLineResolved.Span(), 1)))
		return Line[nonNamedGridPlacement]{
			Start: GenericGridPlacementLine(startLine),
			End:   GenericGridPlacementLine(endLine.intoGridLine(r.explicitTrackCount)),
		}
	case startLineResolved.Kind() == GridPlacementKindNamedSpan && endLineResolved.Kind() == GridPlacementKindLine && endLineResolved.Line().AsI16() != 0:
		endLine := endLineResolved.Line()
		beforeEndLine := endLine.intoOriginZeroLine(r.explicitTrackCount).SubU16(1)
		startLine := r.nthLineBackwards(r.linesNamed(string(startLineResolved.Name())), beforeEndLine, int32(max(startLineResolved.Span(), 1)))
		return Line[nonNamedGridPlacement]{
			Start: GenericGridPlacementLine(startLine.intoGridLine(r.explicitTrackCount)),
			End:   GenericGridPlacementLine(endLine),
		}
	default:
		return Line[nonNamedGridPlacement]{
			Start: nonNamedFromResolved(startLineResolved),
			End:   nonNamedFromResolved(endLineResolved),
		}
	}
}

func (r namedLineResolverAxis[S]) findNamedLine(nameS S, idx int16, end gridAreaEnd) GridLine {
	name := string(nameS)

	if idx == 0 {
		if firstLine, ok := r.areaLine(name, end).Get(); ok {
			return originZeroLineClamped(ozLine(firstLine)).intoGridLine(r.explicitTrackCount)
		}
	}

	lines := r.linesNamed(name)
	var line OriginZeroLine
	if idx >= 0 {
		line = r.nthLineForwards(lines, OriginZeroLine(0), max(int32(idx), 1))
	} else {
		line = r.nthLineBackwards(lines, r.lastExplicitLine(), -int32(idx))
	}
	return line.intoGridLine(r.explicitTrackCount)
}

func (r namedLineResolverAxis[S]) linesNamed(name string) []uint32 {
	if lines, ok := r.lines[S(name)]; ok {
		return lines
	}
	if areaName, areaEnd, ok := splitAreaEdgeName(name); ok {
		if line, ok := r.areaLine(areaName, areaEnd).Get(); ok {
			return []uint32{line}
		}
	}
	return nil
}

func (r namedLineResolverAxis[S]) lastExplicitLine() OriginZeroLine {
	return OriginZeroLine(int16(r.explicitTrackCount))
}

func (r namedLineResolverAxis[S]) nthLineForwards(lines []uint32, firstLine OriginZeroLine, n int32) OriginZeroLine {
	first := int32(max(int16(firstLine), 0))
	lines = lines[sort.Search(len(lines), func(i int) bool { return !(ozLine(lines[i]) < first) }):]
	var line int32
	if int(n)-1 < len(lines) {
		line = ozLine(lines[n-1])
	} else {
		line = max(first, int32(r.lastExplicitLine())+1) + (n - int32(len(lines))) - 1
	}
	return originZeroLineClamped(line)
}

func (r namedLineResolverAxis[S]) nthLineBackwards(lines []uint32, lastLine OriginZeroLine, n int32) OriginZeroLine {
	last := int32(min(int16(lastLine), int16(r.lastExplicitLine())))
	lines = lines[:sort.Search(len(lines), func(i int) bool { return !(ozLine(lines[i]) <= last) })]
	var line int32
	if len(lines) >= int(n) {
		line = ozLine(lines[len(lines)-int(n)])
	} else {
		line = min(last, -1) - (n - int32(len(lines))) + 1
	}
	return originZeroLineClamped(line)
}

func ozLine(line uint32) int32 {
	return int32(line) - 1
}

func collectTemplateLineNames[S CheapCloneStr](
	tracks Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]],
	lineNames Option[TemplateLineNames[S]],
	autoRepetitions uint16,
	lines namedGridLinesMap[S],
	areas gridAreasMap[S],
	axis gridAreaAxis,
	pairs *[]lineNamePair[S],
) {
	var currentLine uint32
	templateTracks, ok := tracks.Get()
	if !ok {
		return
	}
	lineNamesIter, ok := lineNames.Get()
	if !ok {
		return
	}
	nextTrack := 0
	for _, names := range lineNamesIter {
		currentLine++
		for _, lineName := range names {
			*pairs = append(*pairs, lineNamePair[S]{line: currentLine, name: lineName})
			upsertLineNameMap(lines, areas, axis, lineName, currentLine)
		}

		if nextTrack >= len(templateTracks) {
			continue
		}
		component := templateTracks[nextTrack]
		nextTrack++
		if component.Kind() != GenericGridTemplateComponentKindRepeat {
			continue
		}
		repeat := component.Repeat()
		var repeatCount uint16
		switch count := repeat.Count(); count.Kind() {
		case RepetitionCountKindCount:
			repeatCount = count.Count()
		default:
			repeatCount = autoRepetitions
		}

		lineNameSetCount := uint32(len(repeat.LinesNames()))
		linesPerRepetition := uint32(repeat.TrackCount())
		if !(lineNameSetCount == 0 || lineNameSetCount == linesPerRepetition+1) {
			panic(fmt.Sprintf("grid template repetition must have no line name sets or exactly track count + 1 of them (%d tracks but %d line name sets)", linesPerRepetition, lineNameSetCount))
		}

		for range repeatCount {
			line := currentLine
			for _, lineNameSet := range repeat.LinesNames() {
				for _, lineName := range lineNameSet {
					*pairs = append(*pairs, lineNamePair[S]{line: line, name: lineName})
					upsertLineNameMap(lines, areas, axis, lineName, line)
				}
				line++
			}
			currentLine += linesPerRepetition

			if currentLine > uint32(maxGridTracks) {
				break
			}
		}
		if repeatCount > 0 && currentLine > 0 {
			currentLine--
		}
	}
}

func newNamedLineResolver[S CheapCloneStr](style GridContainerStyle[S], columnAutoRepetitions uint16, rowAutoRepetitions uint16) namedLineResolver[S] {
	columnLines := namedGridLinesMap[S]{}
	rowLines := namedGridLinesMap[S]{}
	areas := gridAreasMap[S]{}

	var columnLineNamePairs []lineNamePair[S]
	var rowLineNamePairs []lineNamePair[S]

	areaColumnCount := style.GridTemplateAreaColumnCount()
	areaRowCount := style.GridTemplateAreaRowCount()
	if areaList, ok := style.GridTemplateAreas().Get(); ok {
		for _, area := range areaList {
			areas[area.Name] = &areaLines{
				rowStart:    uint32(area.RowStart),
				rowEnd:      uint32(area.RowEnd),
				columnStart: uint32(area.ColumnStart),
				columnEnd:   uint32(area.ColumnEnd),
			}
		}
	}

	collectTemplateLineNames(style.GridTemplateColumns(), style.GridTemplateColumnNames(), columnAutoRepetitions, columnLines, areas, gridAreaAxisColumn, &columnLineNamePairs)
	collectTemplateLineNames(style.GridTemplateRows(), style.GridTemplateRowNames(), rowAutoRepetitions, rowLines, areas, gridAreaAxisRow, &rowLineNamePairs)

	for name, lines := range columnLines {
		slices.Sort(lines)
		columnLines[name] = slices.Compact(lines)
	}
	for name, lines := range rowLines {
		slices.Sort(lines)
		rowLines[name] = slices.Compact(lines)
	}

	return namedLineResolver[S]{
		areaColumnCount:     areaColumnCount,
		areaRowCount:        areaRowCount,
		explicitColumnCount: 0,
		explicitRowCount:    0,
		rowLines:            rowLines,
		columnLines:         columnLines,
		areas:               areas,
		columnLineNamePairs: columnLineNamePairs,
		rowLineNamePairs:    rowLineNamePairs,
	}
}

func (r *namedLineResolver[S]) detailedLineNames(axis AbsoluteAxis) GridLineNames[S] {
	var pairs []lineNamePair[S]
	var explicitTrackCount uint16
	switch axis {
	case AbsoluteAxisHorizontal:
		pairs, explicitTrackCount = r.columnLineNamePairs, r.explicitColumnCount
	default:
		pairs, explicitTrackCount = r.rowLineNamePairs, r.explicitRowCount
	}

	if len(pairs) == 0 {
		return GridLineNames[S]{}
	}

	sortedPairs := slices.Clone(pairs)
	slices.SortStableFunc(sortedPairs, func(a, b lineNamePair[S]) int {
		switch {
		case a.line < b.line:
			return -1
		case a.line > b.line:
			return 1
		default:
			return 0
		}
	})

	lineCount := int(explicitTrackCount) + 1
	lineNames := gridLineNamesWithCapacity[S](len(sortedPairs), lineCount+1)
	next := 0
	for line := uint32(1); line <= uint32(lineCount); line++ {
		lineNames.startLine()
		for next < len(sortedPairs) {
			pair := sortedPairs[next]
			if pair.line != line {
				break
			}
			next++
			if !lineNames.currentLineContains(string(pair.name)) {
				lineNames.pushName(pair.name)
			}
		}
	}
	return lineNames
}

func (r *namedLineResolver[S]) resolveRowNames(line Line[GridPlacement[S]]) Line[nonNamedGridPlacement] {
	return r.resolveLineNames(line, gridAreaAxisRow)
}

func (r *namedLineResolver[S]) resolveColumnNames(line Line[GridPlacement[S]]) Line[nonNamedGridPlacement] {
	return r.resolveLineNames(line, gridAreaAxisColumn)
}

func (r *namedLineResolver[S]) resolveLineNames(line Line[GridPlacement[S]], axis gridAreaAxis) Line[nonNamedGridPlacement] {
	var lines namedGridLinesMap[S]
	var explicitTrackCount uint16
	switch axis {
	case gridAreaAxisRow:
		lines, explicitTrackCount = r.rowLines, r.explicitRowCount
	default:
		lines, explicitTrackCount = r.columnLines, r.explicitColumnCount
	}
	return namedLineResolverAxis[S]{lines: lines, areas: r.areas, axis: axis, explicitTrackCount: explicitTrackCount}.resolveLineNames(line)
}

func (r *namedLineResolver[S]) populateDetailedLineResolvers(rows, columns *GridLineNames[S]) gridAreasMap[S] {
	rows.resolver = r.rowLines
	columns.resolver = r.columnLines
	return r.areas
}

func (r *namedLineResolver[S]) getAreaColumnCount() uint16 {
	return r.areaColumnCount
}

func (r *namedLineResolver[S]) getAreaRowCount() uint16 {
	return r.areaRowCount
}

func (r *namedLineResolver[S]) setExplicitColumnCount(count uint16) {
	r.explicitColumnCount = count
}

func (r *namedLineResolver[S]) setExplicitRowCount(count uint16) {
	r.explicitRowCount = count
}

func (r *namedLineResolver[S]) String() string {
	var f strings.Builder
	f.WriteString("Grid Areas (row-start / column-start / row-end / column-end):\n")
	for name, area := range r.areas {
		fmt.Fprintf(&f, "%s: %d / %d / %d / %d\n", string(name), area.rowStart, area.columnStart, area.rowEnd, area.columnEnd)
	}

	f.WriteString("Grid Rows:\n")
	for name, lines := range r.rowLines {
		fmt.Fprintf(&f, "%s: ", string(name))
		for _, line := range lines {
			fmt.Fprintf(&f, "%d  ", line)
		}
		f.WriteByte('\n')
	}

	f.WriteString("Grid Columns:\n")
	for name, lines := range r.columnLines {
		fmt.Fprintf(&f, "%s: ", string(name))
		for _, line := range lines {
			fmt.Fprintf(&f, "%d  ", line)
		}
		f.WriteByte('\n')
	}

	return f.String()
}

// GridLineNames holds the names of each explicit grid line in one axis, in compressed sparse row form.
type GridLineNames[S CheapCloneStr] struct {
	names    []S
	offsets  []uint32
	resolver namedGridLinesMap[S]
}

func gridLineNamesWithCapacity[S CheapCloneStr](nameCapacity, offsetCapacity int) GridLineNames[S] {
	offsets := make([]uint32, 0, offsetCapacity)
	offsets = append(offsets, 0)
	return GridLineNames[S]{names: make([]S, 0, nameCapacity), offsets: offsets, resolver: namedGridLinesMap[S]{}}
}

func (n *GridLineNames[S]) startLine() {
	n.offsets = append(n.offsets, uint32(len(n.names)))
}

func (n *GridLineNames[S]) pushName(name S) {
	n.names = append(n.names, name)
	n.offsets[len(n.offsets)-1] = uint32(len(n.names))
}

func (n *GridLineNames[S]) currentLineContains(name string) bool {
	for _, existing := range n.Line(n.LineCount() - 1) {
		if string(existing) == name {
			return true
		}
	}
	return false
}

func (n *GridLineNames[S]) resolveLineNames(line Line[GridPlacement[S]], areas gridAreasMap[S], axis gridAreaAxis, explicitTrackCount uint16) Line[nonNamedGridPlacement] {
	return namedLineResolverAxis[S]{lines: n.resolver, areas: areas, axis: axis, explicitTrackCount: explicitTrackCount}.resolveLineNames(line)
}

// IsEmpty reports whether the axis has any named lines at all.
func (n *GridLineNames[S]) IsEmpty() bool {
	return len(n.names) == 0
}

// LineCount returns the number of grid lines represented (zero if the grid has no named lines).
func (n *GridLineNames[S]) LineCount() int {
	return max(len(n.offsets)-1, 0)
}

// Line returns the names of the line with the given 0-indexed line index, or an empty slice if it has none or the index is out of range.
func (n *GridLineNames[S]) Line(lineIndex int) []S {
	if lineIndex < 0 || lineIndex+1 >= len(n.offsets) {
		return []S{}
	}
	return n.names[n.offsets[lineIndex]:n.offsets[lineIndex+1]]
}

// Iter returns an iterator over the name group of each grid line in line order.
func (n *GridLineNames[S]) Iter() GridLineNamesIter[S] {
	return n.iterPadded(0, 0)
}

// All yields the name group of each grid line in line order (upstream `impl IntoIterator for &GridLineNames`).
func (n *GridLineNames[S]) All() iter.Seq[[]S] {
	return n.Iter().All()
}

// Equal reports whether n and other hold the same names, offsets and resolver map (upstream's derived PartialEq).
func (n *GridLineNames[S]) Equal(other *GridLineNames[S]) bool {
	if !slices.Equal(n.names, other.names) || !slices.Equal(n.offsets, other.offsets) || len(n.resolver) != len(other.resolver) {
		return false
	}
	for name, lines := range n.resolver {
		otherLines, ok := other.resolver[name]
		if !ok || !slices.Equal(lines, otherLines) {
			return false
		}
	}
	return true
}

func (n *GridLineNames[S]) iterPadded(leadingEmpty, trailingEmpty int) GridLineNamesIter[S] {
	return GridLineNamesIter[S]{names: n.names, offsets: n.offsets, leadingEmpty: leadingEmpty, trailingEmpty: trailingEmpty}
}

// GridLineNamesIter iterates over the per-line name groups of a GridLineNames, yielding an empty group for unnamed lines.
type GridLineNamesIter[S CheapCloneStr] struct {
	names         []S
	offsets       []uint32
	leadingEmpty  int
	trailingEmpty int
}

// Next returns the next name group, or false when the iterator is exhausted.
func (it *GridLineNamesIter[S]) Next() ([]S, bool) {
	if it.leadingEmpty > 0 {
		it.leadingEmpty--
		return []S{}, true
	}
	if len(it.offsets) >= 2 {
		window := it.offsets[:2]
		it.offsets = it.offsets[1:]
		return it.names[window[0]:window[1]], true
	}
	if it.trailingEmpty > 0 {
		it.trailingEmpty--
		return []S{}, true
	}
	return nil, false
}

// Len returns the number of name groups remaining (upstream's exact size_hint).
func (it *GridLineNamesIter[S]) Len() int {
	return it.leadingEmpty + max(len(it.offsets)-1, 0) + it.trailingEmpty
}

// All yields the remaining name groups.
func (it GridLineNamesIter[S]) All() iter.Seq[[]S] {
	return func(yield func([]S) bool) {
		for {
			group, ok := it.Next()
			if !ok || !yield(group) {
				return
			}
		}
	}
}
