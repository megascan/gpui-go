package consumer

import (
	"io"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type innerPosition struct {
	node           Node
	characterIndex int
}

func upgradeInnerPosition(treeState *State, weak a11y.TextPosition, nodeID NodeID) (innerPosition, bool) {
	node, ok := treeState.NodeByID(nodeID.withSameTree(weak.Node))
	if !ok {
		return innerPosition{}, false
	}
	if node.Role() != a11y.RoleTextRun {
		return innerPosition{}, false
	}
	characterIndex := weak.CharacterIndex
	if characterIndex > len(node.Data().CharacterLengths()) {
		return innerPosition{}, false
	}
	return innerPosition{node: node, characterIndex: characterIndex}, true
}

func clampedUpgradeInnerPosition(treeState *State, weak a11y.TextPosition, nodeID NodeID) (innerPosition, bool) {
	node, ok := treeState.NodeByID(nodeID.withSameTree(weak.Node))
	if !ok {
		return innerPosition{}, false
	}
	if node.Role() != a11y.RoleTextRun {
		return innerPosition{}, false
	}
	characterIndex := min(weak.CharacterIndex, len(node.Data().CharacterLengths()))
	return innerPosition{node: node, characterIndex: characterIndex}, true
}

func (p innerPosition) isRunStart() bool {
	return p.characterIndex == 0
}

func (p innerPosition) isLineStart() bool {
	_, ok := p.node.Data().PreviousOnLine()
	return p.isRunStart() && !ok
}

func (p innerPosition) isRunEnd() bool {
	return p.characterIndex == len(p.node.Data().CharacterLengths())
}

func (p innerPosition) isLineEnd() bool {
	_, ok := p.node.Data().NextOnLine()
	return p.isRunEnd() && !ok
}

func (p innerPosition) isParagraphEnd() bool {
	if !p.isLineEnd() {
		return false
	}
	return strings.HasSuffix(unwrapStr(p.node.Data().Value()), "\n")
}

func (p innerPosition) isDocumentStart(rootNode Node) bool {
	if !p.isRunStart() {
		return false
	}
	_, ok := p.node.precedingTextRuns(rootNode).Next()
	return !ok
}

func (p innerPosition) isDocumentEnd(rootNode Node) bool {
	if !p.isRunEnd() {
		return false
	}
	_, ok := p.node.followingTextRuns(rootNode).Next()
	return !ok
}

func (p innerPosition) biasedToStart(rootNode Node) innerPosition {
	if p.isRunEnd() {
		if node, ok := p.node.followingTextRuns(rootNode).Next(); ok {
			return innerPosition{node: node, characterIndex: 0}
		}
	}
	return p
}

func (p innerPosition) biasedToEnd(rootNode Node) innerPosition {
	if p.isRunStart() {
		if node, ok := p.node.precedingTextRuns(rootNode).Next(); ok {
			return innerPosition{node: node, characterIndex: len(node.Data().CharacterLengths())}
		}
	}
	return p
}

type textComparable struct {
	path  []int
	index int
}

func compareComparable(a, b textComparable) int {
	if c := slices.Compare(a.path, b.path); c != 0 {
		return c
	}
	switch {
	case a.index < b.index:
		return -1
	case a.index > b.index:
		return 1
	}
	return 0
}

func (p innerPosition) comparable(rootNode Node) textComparable {
	normalized := p.biasedToStart(rootNode)
	return textComparable{path: normalized.node.RelativeIndexPath(rootNode.ID()), index: normalized.characterIndex}
}

func (p innerPosition) lineStart() innerPosition {
	node := p.node
	for {
		id, ok := node.Data().PreviousOnLine()
		if !ok {
			break
		}
		node = unwrapNode(node.TreeState.NodeByID(node.id.withSameTree(id)))
	}
	return innerPosition{node: node, characterIndex: 0}
}

func (p innerPosition) lineEnd() innerPosition {
	node := p.node
	for {
		id, ok := node.Data().NextOnLine()
		if !ok {
			break
		}
		node = unwrapNode(node.TreeState.NodeByID(node.id.withSameTree(id)))
	}
	return innerPosition{node: node, characterIndex: len(node.Data().CharacterLengths())}
}

func (p innerPosition) downgrade() a11y.TextPosition {
	localNodeID, _ := p.node.id.toComponents()
	return a11y.TextPosition{Node: localNodeID, CharacterIndex: p.characterIndex}
}

func (p innerPosition) eq(other innerPosition) bool {
	return p.node.ID() == other.node.ID() && p.characterIndex == other.characterIndex
}

func unwrapNode(n Node, ok bool) Node {
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return n
}

func unwrapStr(s string, ok bool) string {
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return s
}

// Position is a text position within a node that supports text ranges.
type Position struct {
	rootNode Node
	inner    innerPosition
}

// ToRaw returns the position's tree-local form.
func (p Position) ToRaw() a11y.TextPosition {
	return p.inner.downgrade()
}

// InnerNode returns the text run the position is in.
func (p Position) InnerNode() Node {
	return p.inner.node
}

// IsFormatStart reports whether text attributes change at the position.
func (p Position) IsFormatStart() bool {
	if p.IsDocumentStart() {
		return true
	}
	if p.inner.characterIndex != 0 {
		return false
	}
	return p.inner.node.textAttributesDiffer(unwrapNode(p.inner.node.precedingTextRuns(p.rootNode).Next()))
}

// IsWordStart reports whether the position is at the start of a word.
func (p Position) IsWordStart() bool {
	if p.IsParagraphStart() {
		return true
	}
	_, found := slices.BinarySearch(p.inner.node.Data().WordStarts(), uint8(p.inner.characterIndex))
	return found
}

// IsLineStart reports whether the position is at the start of a line.
func (p Position) IsLineStart() bool {
	return p.inner.isLineStart()
}

// IsLineEnd reports whether the position is at the end of a line.
func (p Position) IsLineEnd() bool {
	return p.inner.isLineEnd()
}

// IsParagraphStart reports whether the position is at the start of a paragraph.
func (p Position) IsParagraphStart() bool {
	return p.IsDocumentStart() || (p.IsLineStart() && p.inner.biasedToEnd(p.rootNode).isParagraphEnd())
}

// IsParagraphEnd reports whether the position is at the end of a paragraph.
func (p Position) IsParagraphEnd() bool {
	return p.IsDocumentEnd() || p.inner.isParagraphEnd()
}

// IsParagraphSeparator reports whether the character after the position ends a paragraph.
func (p Position) IsParagraphSeparator() bool {
	if p.IsDocumentEnd() {
		return false
	}
	next := p.ForwardToCharacterEnd()
	return !next.IsDocumentEnd() && next.IsParagraphEnd()
}

// IsPageStart reports whether the position is at the start of a page.
func (p Position) IsPageStart() bool {
	return p.IsDocumentStart()
}

// IsDocumentStart reports whether the position is at the start of the document.
func (p Position) IsDocumentStart() bool {
	return p.inner.isDocumentStart(p.rootNode)
}

// IsDocumentEnd reports whether the position is at the end of the document.
func (p Position) IsDocumentEnd() bool {
	return p.inner.isDocumentEnd(p.rootNode)
}

// ToDegenerateRange returns the empty range at the position.
func (p Position) ToDegenerateRange() Range {
	return newRange(p.rootNode, p.inner, p.inner)
}

func (p Position) globalIndex(length func(string) int) int {
	total := 0
	runs := p.rootNode.textRuns()
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		nodeText := unwrapStr(node.Data().Value())
		if node.ID() == p.inner.node.ID() {
			sliceEnd := sumLengths(node.Data().CharacterLengths()[:p.inner.characterIndex])
			return total + length(nodeText[:sliceEnd])
		}
		total += length(nodeText)
	}
	panic("invalid position")
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

func sumLengths(lengths []uint8) int {
	n := 0
	for _, l := range lengths {
		n += int(l)
	}
	return n
}

// ToGlobalUsvIndex returns the position's index in Unicode scalar values from the document start.
func (p Position) ToGlobalUsvIndex() int {
	return p.globalIndex(utf8.RuneCountInString)
}

// ToGlobalUTF16Index returns the position's index in UTF-16 code units from the document start.
func (p Position) ToGlobalUTF16Index() int {
	return p.globalIndex(utf16Len)
}

// ToLineIndex returns the index of the line the position is on.
func (p Position) ToLineIndex() int {
	pos := p
	if !pos.IsLineStart() {
		pos = pos.BackwardToLineStart()
	}
	linesBeforeCurrent := 0
	for !pos.IsDocumentStart() {
		pos = pos.BackwardToLineStart()
		linesBeforeCurrent++
	}
	return linesBeforeCurrent
}

// BiasedToStart returns the equivalent position at the start of the next run when at a run end.
func (p Position) BiasedToStart() Position {
	return Position{rootNode: p.rootNode, inner: p.inner.biasedToStart(p.rootNode)}
}

// BiasedToEnd returns the equivalent position at the end of the previous run when at a run start.
func (p Position) BiasedToEnd() Position {
	return Position{rootNode: p.rootNode, inner: p.inner.biasedToEnd(p.rootNode)}
}

// ForwardToCharacterStart returns the start of the next character.
func (p Position) ForwardToCharacterStart() Position {
	pos := p.inner.biasedToStart(p.rootNode)
	return Position{rootNode: p.rootNode, inner: innerPosition{node: pos.node, characterIndex: pos.characterIndex + 1}.biasedToStart(p.rootNode)}
}

// ForwardToCharacterEnd returns the end of the current character.
func (p Position) ForwardToCharacterEnd() Position {
	pos := p.inner.biasedToStart(p.rootNode)
	return Position{rootNode: p.rootNode, inner: innerPosition{node: pos.node, characterIndex: pos.characterIndex + 1}}
}

// BackwardToCharacterStart returns the start of the previous character.
func (p Position) BackwardToCharacterStart() Position {
	pos := p.inner.biasedToEnd(p.rootNode)
	if pos.characterIndex == 0 {
		panic("attempt to subtract with overflow")
	}
	return Position{rootNode: p.rootNode, inner: innerPosition{node: pos.node, characterIndex: pos.characterIndex - 1}.biasedToStart(p.rootNode)}
}

// ForwardToFormatStart returns the next position where text attributes change.
func (p Position) ForwardToFormatStart() Position {
	runs := p.inner.node.followingTextRuns(p.rootNode)
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		if p.inner.node.textAttributesDiffer(node) {
			return Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: 0}}
		}
	}
	return p.DocumentEnd()
}

// ForwardToFormatEnd returns the end of the current format run.
func (p Position) ForwardToFormatEnd() Position {
	return p.ForwardToFormatStart().BiasedToEnd()
}

// BackwardToFormatStart returns the start of the current or previous format run.
func (p Position) BackwardToFormatStart() Position {
	if p.inner.characterIndex != 0 {
		testPos := Position{rootNode: p.rootNode, inner: innerPosition{node: p.inner.node, characterIndex: 0}}
		if testPos.IsFormatStart() {
			return testPos
		}
	}
	runs := p.inner.node.precedingTextRuns(p.rootNode)
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		testPos := Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: 0}}
		if testPos.IsFormatStart() {
			return testPos
		}
	}
	return p.DocumentStart()
}

// ForwardToWordStart returns the start of the next word.
func (p Position) ForwardToWordStart() Position {
	pos := p.inner.biasedToStart(p.rootNode)
	{
		wordStarts := pos.node.Data().WordStarts()
		index, found := slices.BinarySearch(wordStarts, uint8(pos.characterIndex))
		if found {
			index++
		}
		if index < len(wordStarts) {
			return Position{rootNode: p.rootNode, inner: innerPosition{node: pos.node, characterIndex: int(wordStarts[index])}}
		}
	}
	runs := pos.node.followingTextRuns(p.rootNode)
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		startPos := Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: 0}}
		if startPos.IsParagraphStart() {
			return startPos
		}
		if ws := node.Data().WordStarts(); len(ws) > 0 {
			return Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: int(ws[0])}}
		}
	}
	return p.DocumentEnd()
}

// ForwardToWordEnd returns the end of the current word.
func (p Position) ForwardToWordEnd() Position {
	return p.ForwardToWordStart().BiasedToEnd()
}

// BackwardToWordStart returns the start of the current or previous word.
func (p Position) BackwardToWordStart() Position {
	{
		wordStarts := p.inner.node.Data().WordStarts()
		index, _ := slices.BinarySearch(wordStarts, uint8(p.inner.characterIndex))
		if index > 0 {
			return Position{rootNode: p.rootNode, inner: innerPosition{node: p.inner.node, characterIndex: int(wordStarts[index-1])}}
		}
	}
	if p.inner.characterIndex != 0 {
		startPos := Position{rootNode: p.rootNode, inner: innerPosition{node: p.inner.node, characterIndex: 0}}
		if startPos.IsParagraphStart() {
			return startPos
		}
	}
	runs := p.inner.node.precedingTextRuns(p.rootNode)
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		if ws := node.Data().WordStarts(); len(ws) > 0 {
			return Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: int(ws[len(ws)-1])}}
		}
		startPos := Position{rootNode: p.rootNode, inner: innerPosition{node: node, characterIndex: 0}}
		if startPos.IsParagraphStart() {
			return startPos
		}
	}
	return p.DocumentStart()
}

// ForwardToLineStart returns the start of the next line.
func (p Position) ForwardToLineStart() Position {
	return Position{rootNode: p.rootNode, inner: p.inner.lineEnd().biasedToStart(p.rootNode)}
}

// ForwardToLineEnd returns the end of the current line.
func (p Position) ForwardToLineEnd() Position {
	pos := p.inner.biasedToStart(p.rootNode)
	return Position{rootNode: p.rootNode, inner: pos.lineEnd()}
}

// BackwardToLineStart returns the start of the current or previous line.
func (p Position) BackwardToLineStart() Position {
	pos := p.inner.biasedToEnd(p.rootNode)
	return Position{rootNode: p.rootNode, inner: pos.lineStart().biasedToStart(p.rootNode)}
}

// ForwardToParagraphStart returns the start of the next paragraph.
func (p Position) ForwardToParagraphStart() Position {
	current := p
	for {
		current = current.ForwardToLineStart()
		if current.IsDocumentEnd() || current.inner.biasedToEnd(p.rootNode).isParagraphEnd() {
			break
		}
	}
	return current
}

// ForwardToParagraphEnd returns the end of the current paragraph.
func (p Position) ForwardToParagraphEnd() Position {
	current := p
	for {
		current = current.ForwardToLineEnd()
		if current.IsDocumentEnd() || current.inner.isParagraphEnd() {
			break
		}
	}
	return current
}

// BackwardToParagraphStart returns the start of the current or previous paragraph.
func (p Position) BackwardToParagraphStart() Position {
	current := p
	for {
		current = current.BackwardToLineStart()
		if current.IsParagraphStart() {
			break
		}
	}
	return current
}

// ForwardToPageStart returns the start of the next page.
func (p Position) ForwardToPageStart() Position {
	return p.DocumentEnd()
}

// ForwardToPageEnd returns the end of the current page.
func (p Position) ForwardToPageEnd() Position {
	return p.DocumentEnd()
}

// BackwardToPageStart returns the start of the current page.
func (p Position) BackwardToPageStart() Position {
	return p.DocumentStart()
}

// DocumentEnd returns the end of the document.
func (p Position) DocumentEnd() Position {
	return p.rootNode.DocumentEnd()
}

// DocumentStart returns the start of the document.
func (p Position) DocumentStart() Position {
	return p.rootNode.DocumentStart()
}

// Equal reports whether p and other are the same position.
func (p Position) Equal(other Position) bool {
	return p.rootNode.ID() == other.rootNode.ID() && p.inner.eq(other.inner)
}

// PartialCmp orders p and other; ok is false when they belong to different root nodes.
func (p Position) PartialCmp(other Position) (int, bool) {
	if p.rootNode.ID() != other.rootNode.ID() {
		return 0, false
	}
	selfComparable := p.inner.comparable(p.rootNode)
	otherComparable := other.inner.comparable(p.rootNode)
	return compareComparable(selfComparable, otherComparable), true
}

// RangePropertyValue is a text attribute's value over a range: one value, or mixed.
type RangePropertyValue[T comparable] struct {
	value T
	mixed bool
}

// RangePropertyValueSingle returns a single-valued RangePropertyValue.
func RangePropertyValueSingle[T comparable](value T) RangePropertyValue[T] {
	return RangePropertyValue[T]{value: value}
}

// RangePropertyValueMixed returns a mixed RangePropertyValue.
func RangePropertyValueMixed[T comparable]() RangePropertyValue[T] {
	return RangePropertyValue[T]{mixed: true}
}

// Single returns the value and true, or false when the value is mixed.
func (v RangePropertyValue[T]) Single() (T, bool) {
	return v.value, !v.mixed
}

// IsMixed reports whether the value is mixed.
func (v RangePropertyValue[T]) IsMixed() bool {
	return v.mixed
}

// RangePropertyValueMap maps a single optional value with f.
func RangePropertyValueMap[T, U comparable](v RangePropertyValue[gpuiutil.Option[T]], f func(T) U) RangePropertyValue[gpuiutil.Option[U]] {
	if v.mixed {
		return RangePropertyValueMixed[gpuiutil.Option[U]]()
	}
	if value, ok := v.value.Get(); ok {
		return RangePropertyValueSingle(gpuiutil.Some(f(value)))
	}
	return RangePropertyValueSingle(gpuiutil.None[U]())
}

// Range is a text range within a node that supports text ranges.
type Range struct {
	node  Node
	start innerPosition
	end   innerPosition
}

func newRange(node Node, start, end innerPosition) Range {
	if compareComparable(start.comparable(node), end.comparable(node)) > 0 {
		start, end = end, start
	}
	return Range{node: node, start: start, end: end}
}

// Node returns the node the range belongs to.
func (r Range) Node() Node {
	return r.node
}

// Start returns the range's start position.
func (r Range) Start() Position {
	return Position{rootNode: r.node, inner: r.start}
}

// End returns the range's end position.
func (r Range) End() Position {
	return Position{rootNode: r.node, inner: r.end}
}

// IsDegenerate reports whether the range is empty.
func (r Range) IsDegenerate() bool {
	return compareComparable(r.start.comparable(r.node), r.end.comparable(r.node)) == 0
}

func (r Range) walk(f func(node *Node) bool) bool {
	var start, end innerPosition
	if r.IsDegenerate() {
		start, end = r.start, r.start
	} else {
		start = r.start.biasedToStart(r.node)
		end = r.end.biasedToEnd(r.node)
	}
	if f(&start.node) {
		return true
	}
	if start.node.ID() == end.node.ID() {
		return false
	}
	runs := start.node.followingTextRuns(r.node)
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		if f(&node) {
			return true
		}
		if node.ID() == end.node.ID() {
			break
		}
	}
	return false
}

// TraverseText calls f with each text run in the range and its text until f returns ok.
func (r Range) TraverseText[T any](f func(node *Node, s string) (T, bool)) (T, bool) {
	var result T
	found := r.walk(func(node *Node) bool {
		characterLengths := node.Data().CharacterLengths()
		startIndex := 0
		if node.ID() == r.start.node.ID() {
			startIndex = r.start.characterIndex
		}
		endIndex := len(characterLengths)
		if node.ID() == r.end.node.ID() {
			endIndex = r.end.characterIndex
		}
		value := unwrapStr(node.Data().Value())
		var s string
		switch {
		case startIndex == endIndex:
			s = ""
		case startIndex == 0 && endIndex == len(characterLengths):
			s = value
		default:
			sliceStart := sumLengths(characterLengths[:startIndex])
			sliceEnd := sliceStart + sumLengths(characterLengths[startIndex:endIndex])
			s = value[sliceStart:sliceEnd]
		}
		v, ok := f(node, s)
		if ok {
			result = v
		}
		return ok
	})
	return result, found
}

// WriteText writes the range's text to writer.
func (r Range) WriteText(writer io.StringWriter) error {
	err, failed := r.TraverseText(func(_ *Node, s string) (error, bool) {
		_, err := writer.WriteString(s)
		return err, err != nil
	})
	if failed {
		return err
	}
	return nil
}

// Text returns the range's text.
func (r Range) Text() string {
	var result strings.Builder
	if err := r.WriteText(&result); err != nil {
		panic(err)
	}
	return result.String()
}

// BoundingBoxes returns the range's transformed bounding boxes relative to the tree's container
// (e.g. window). An empty result means the tree lacks the data to compute them; otherwise there is
// at least one box, zero-width for a degenerate range.
func (r Range) BoundingBoxes() []a11y.Rect {
	var result []a11y.Rect
	if r.walk(func(node *Node) bool {
		rect, ok := node.Data().Bounds()
		if !ok {
			return true
		}
		positions, ok := node.Data().CharacterPositions()
		if !ok {
			return true
		}
		widths, ok := node.Data().CharacterWidths()
		if !ok {
			return true
		}
		direction, ok := node.TextDirection()
		if !ok {
			return true
		}
		characterLengths := node.Data().CharacterLengths()
		startIndex := 0
		if node.ID() == r.start.node.ID() {
			startIndex = r.start.characterIndex
		}
		endIndex := len(characterLengths)
		if node.ID() == r.end.node.ID() {
			endIndex = r.end.characterIndex
		}
		if startIndex != 0 || endIndex != len(characterLengths) {
			var pixelStart32 float32
			if startIndex < len(characterLengths) {
				pixelStart32 = positions[startIndex]
			} else {
				pixelStart32 = positions[startIndex-1] + widths[startIndex-1]
			}
			pixelEnd32 := pixelStart32
			if endIndex != startIndex {
				pixelEnd32 = positions[endIndex-1] + widths[endIndex-1]
			}
			pixelStart := float64(pixelStart32)
			pixelEnd := float64(pixelEnd32)
			switch direction {
			case a11y.TextDirectionLeftToRight:
				origLeft := rect.X0
				rect.X0 = origLeft + pixelStart
				rect.X1 = origLeft + pixelEnd
			case a11y.TextDirectionRightToLeft:
				origRight := rect.X1
				rect.X1 = origRight - pixelStart
				rect.X0 = origRight - pixelEnd
			case a11y.TextDirectionTopToBottom:
				origTop := rect.Y0
				rect.Y0 = origTop + pixelStart
				rect.Y1 = origTop + pixelEnd
			case a11y.TextDirectionBottomToTop:
				origBottom := rect.Y1
				rect.Y1 = origBottom - pixelStart
				rect.Y0 = origBottom - pixelEnd
			}
		}
		result = append(result, node.Transform().TransformRectBbox(rect))
		return false
	}) {
		return []a11y.Rect{}
	}
	return result
}

func fetchProperty[T comparable](r Range, getter func(*Node) T) RangePropertyValue[T] {
	var value gpuiutil.Option[T]
	if r.walk(func(node *Node) bool {
		current := getter(node)
		if v, ok := value.Get(); ok {
			if v != current {
				return true
			}
		} else {
			value = gpuiutil.Some(current)
		}
		return false
	}) {
		return RangePropertyValueMixed[T]()
	}
	return RangePropertyValueSingle(value.Unwrap())
}

func fetchOptionalProperty[T comparable](r Range, getter func(*Node) (T, bool)) RangePropertyValue[gpuiutil.Option[T]] {
	return fetchProperty(r, func(node *Node) gpuiutil.Option[T] { return gpuiutil.OptionFrom(getter(node)) })
}

func (r *Range) fixStartBias() {
	if !r.IsDegenerate() {
		r.start = r.start.biasedToStart(r.node)
	}
}

func assertSameRoot(pos Position, node Node) {
	if pos.rootNode.ID() != node.ID() {
		panic("assertion `left == right` failed")
	}
}

// SetStart moves the range's start to pos, collapsing the range if pos is past the end.
func (r *Range) SetStart(pos Position) {
	assertSameRoot(pos, r.node)
	r.start = pos.inner
	if compareComparable(r.start.comparable(r.node), r.end.comparable(r.node)) >= 0 {
		r.end = r.start
	}
	r.fixStartBias()
}

// SetEnd moves the range's end to pos, collapsing the range if pos is before the start.
func (r *Range) SetEnd(pos Position) {
	assertSameRoot(pos, r.node)
	r.end = pos.inner
	if compareComparable(r.start.comparable(r.node), r.end.comparable(r.node)) >= 0 {
		r.start = r.end
	}
	r.fixStartBias()
}

// ToTextSelection returns the range as a text selection.
func (r Range) ToTextSelection() a11y.TextSelection {
	return a11y.TextSelection{Anchor: r.start.downgrade(), Focus: r.end.downgrade()}
}

// Downgrade returns a WeakRange that does not hold the tree state.
func (r Range) Downgrade() WeakRange {
	return WeakRange{
		nodeID:          r.node.ID(),
		start:           r.start.downgrade(),
		end:             r.end.downgrade(),
		startComparable: r.start.comparable(r.node),
		endComparable:   r.end.comparable(r.node),
	}
}

// Equal reports whether r and other are the same range.
func (r Range) Equal(other Range) bool {
	return r.node.ID() == other.node.ID() && r.start.eq(other.start) && r.end.eq(other.end)
}

// WeakRange is a Range detached from a tree state.
type WeakRange struct {
	nodeID          NodeID
	start           a11y.TextPosition
	end             a11y.TextPosition
	startComparable textComparable
	endComparable   textComparable
}

// NodeID returns the ID of the range's node.
func (w WeakRange) NodeID() NodeID {
	return w.nodeID
}

// StartComparable returns the index path and character index that order the range's start.
func (w WeakRange) StartComparable() ([]int, int) {
	return w.startComparable.path, w.startComparable.index
}

// EndComparable returns the index path and character index that order the range's end.
func (w WeakRange) EndComparable() ([]int, int) {
	return w.endComparable.path, w.endComparable.index
}

// Clone returns a deep copy of w.
func (w WeakRange) Clone() WeakRange {
	c := w
	c.startComparable.path = slices.Clone(w.startComparable.path)
	c.endComparable.path = slices.Clone(w.endComparable.path)
	return c
}

// Equal reports whether w and other are equal.
func (w WeakRange) Equal(other WeakRange) bool {
	return w.nodeID == other.nodeID && w.start == other.start && w.end == other.end &&
		compareComparable(w.startComparable, other.startComparable) == 0 &&
		compareComparable(w.endComparable, other.endComparable) == 0
}

// UpgradeNode returns the range's node if it still exists and supports text ranges.
func (w WeakRange) UpgradeNode(treeState *State) (Node, bool) {
	node, ok := treeState.NodeByID(w.nodeID)
	if !ok || !node.SupportsTextRanges() {
		return Node{}, false
	}
	return node, true
}

// Upgrade returns the range in treeState if its node and endpoints still exist.
func (w WeakRange) Upgrade(treeState *State) (Range, bool) {
	node, ok := w.UpgradeNode(treeState)
	if !ok {
		return Range{}, false
	}
	start, ok := upgradeInnerPosition(treeState, w.start, w.nodeID)
	if !ok {
		return Range{}, false
	}
	end, ok := upgradeInnerPosition(treeState, w.end, w.nodeID)
	if !ok {
		return Range{}, false
	}
	return Range{node: node, start: start, end: end}, true
}

// TextPosition is the crate-root name of Position.
type TextPosition = Position

// TextRange is the crate-root name of Range.
type TextRange = Range

// TextRangePropertyValue is the crate-root name of RangePropertyValue.
type TextRangePropertyValue[T comparable] = RangePropertyValue[T]

// WeakTextRange is the crate-root name of WeakRange.
type WeakTextRange = WeakRange

func textNodeFilter(rootID NodeID, node *Node) FilterResult {
	if node.ID() == rootID || node.Role() == a11y.RoleTextRun {
		return FilterResultInclude
	}
	return FilterResultExcludeNode
}

func characterIndexAtPoint(node Node, point a11y.Point) int {
	rect, ok := node.Data().Bounds()
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	characterLengths := node.Data().CharacterLengths()
	positions, ok := node.Data().CharacterPositions()
	if !ok {
		return 0
	}
	widths, ok := node.Data().CharacterWidths()
	if !ok {
		return 0
	}
	direction, ok := node.TextDirection()
	if !ok {
		return 0
	}
	for i := min(len(positions), len(widths)) - 1; i >= 0; i-- {
		position, width := positions[i], widths[i]
		var relativePos float64
		switch direction {
		case a11y.TextDirectionLeftToRight:
			relativePos = point.X - rect.X0
		case a11y.TextDirectionRightToLeft:
			relativePos = rect.X1 - point.X
		case a11y.TextDirectionTopToBottom:
			relativePos = point.Y - rect.Y0
		case a11y.TextDirectionBottomToTop:
			relativePos = rect.Y1 - point.Y
		}
		if relativePos >= float64(position) && relativePos < float64(position+width) {
			return i
		}
	}
	return len(characterLengths)
}

// TextDirection returns the node's text direction, inherited from its ancestors.
func (n Node) TextDirection() (a11y.TextDirection, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).TextDirection)
}

// FontFamily returns the node's font family, inherited from its ancestors.
func (n Node) FontFamily() (string, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).FontFamily)
}

// Language returns the node's language, inherited from its ancestors.
func (n Node) Language() (string, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).Language)
}

// FontSize returns the node's font size, inherited from its ancestors.
func (n Node) FontSize() (float32, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).FontSize)
}

// FontWeight returns the node's font weight, inherited from its ancestors.
func (n Node) FontWeight() (float32, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).FontWeight)
}

// BackgroundColor returns the node's background color, inherited from its ancestors.
func (n Node) BackgroundColor() (a11y.Color, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).BackgroundColor)
}

// ForegroundColor returns the node's foreground color, inherited from its ancestors.
func (n Node) ForegroundColor() (a11y.Color, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).ForegroundColor)
}

// Overline returns the node's overline, inherited from its ancestors.
func (n Node) Overline() (a11y.TextDecoration, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).Overline)
}

// Strikethrough returns the node's strikethrough, inherited from its ancestors.
func (n Node) Strikethrough() (a11y.TextDecoration, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).Strikethrough)
}

// Underline returns the node's underline, inherited from its ancestors.
func (n Node) Underline() (a11y.TextDecoration, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).Underline)
}

// TextAlign returns the node's text alignment, inherited from its ancestors.
func (n Node) TextAlign() (a11y.TextAlign, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).TextAlign)
}

// VerticalOffset returns the node's vertical offset, inherited from its ancestors.
func (n Node) VerticalOffset() (a11y.VerticalOffset, bool) {
	return fetchInheritedProperty(n, (*a11y.Node).VerticalOffset)
}

// IsItalic reports whether the node or an ancestor is italic.
func (n Node) IsItalic() bool {
	return n.fetchInheritedFlag((*a11y.Node).IsItalic)
}

// TextDirection returns the text direction at the position.
func (p Position) TextDirection() (a11y.TextDirection, bool) { return p.inner.node.TextDirection() }

// FontFamily returns the font family at the position.
func (p Position) FontFamily() (string, bool) { return p.inner.node.FontFamily() }

// Language returns the language at the position.
func (p Position) Language() (string, bool) { return p.inner.node.Language() }

// FontSize returns the font size at the position.
func (p Position) FontSize() (float32, bool) { return p.inner.node.FontSize() }

// FontWeight returns the font weight at the position.
func (p Position) FontWeight() (float32, bool) { return p.inner.node.FontWeight() }

// BackgroundColor returns the background color at the position.
func (p Position) BackgroundColor() (a11y.Color, bool) { return p.inner.node.BackgroundColor() }

// ForegroundColor returns the foreground color at the position.
func (p Position) ForegroundColor() (a11y.Color, bool) { return p.inner.node.ForegroundColor() }

// Overline returns the overline at the position.
func (p Position) Overline() (a11y.TextDecoration, bool) { return p.inner.node.Overline() }

// Strikethrough returns the strikethrough at the position.
func (p Position) Strikethrough() (a11y.TextDecoration, bool) { return p.inner.node.Strikethrough() }

// Underline returns the underline at the position.
func (p Position) Underline() (a11y.TextDecoration, bool) { return p.inner.node.Underline() }

// TextAlign returns the text alignment at the position.
func (p Position) TextAlign() (a11y.TextAlign, bool) { return p.inner.node.TextAlign() }

// VerticalOffset returns the vertical offset at the position.
func (p Position) VerticalOffset() (a11y.VerticalOffset, bool) { return p.inner.node.VerticalOffset() }

// IsItalic reports whether the text at the position is italic.
func (p Position) IsItalic() bool { return p.inner.node.IsItalic() }

// TextDirection returns the text direction over the range.
func (r Range) TextDirection() RangePropertyValue[gpuiutil.Option[a11y.TextDirection]] {
	return fetchOptionalProperty(r, (*Node).TextDirection)
}

// FontFamily returns the font family over the range.
func (r Range) FontFamily() RangePropertyValue[gpuiutil.Option[string]] {
	return fetchOptionalProperty(r, (*Node).FontFamily)
}

// Language returns the language over the range.
func (r Range) Language() RangePropertyValue[gpuiutil.Option[string]] {
	return fetchOptionalProperty(r, (*Node).Language)
}

// FontSize returns the font size over the range.
func (r Range) FontSize() RangePropertyValue[gpuiutil.Option[float32]] {
	return fetchOptionalProperty(r, (*Node).FontSize)
}

// FontWeight returns the font weight over the range.
func (r Range) FontWeight() RangePropertyValue[gpuiutil.Option[float32]] {
	return fetchOptionalProperty(r, (*Node).FontWeight)
}

// BackgroundColor returns the background color over the range.
func (r Range) BackgroundColor() RangePropertyValue[gpuiutil.Option[a11y.Color]] {
	return fetchOptionalProperty(r, (*Node).BackgroundColor)
}

// ForegroundColor returns the foreground color over the range.
func (r Range) ForegroundColor() RangePropertyValue[gpuiutil.Option[a11y.Color]] {
	return fetchOptionalProperty(r, (*Node).ForegroundColor)
}

// Overline returns the overline over the range.
func (r Range) Overline() RangePropertyValue[gpuiutil.Option[a11y.TextDecoration]] {
	return fetchOptionalProperty(r, (*Node).Overline)
}

// Strikethrough returns the strikethrough over the range.
func (r Range) Strikethrough() RangePropertyValue[gpuiutil.Option[a11y.TextDecoration]] {
	return fetchOptionalProperty(r, (*Node).Strikethrough)
}

// Underline returns the underline over the range.
func (r Range) Underline() RangePropertyValue[gpuiutil.Option[a11y.TextDecoration]] {
	return fetchOptionalProperty(r, (*Node).Underline)
}

// TextAlign returns the text alignment over the range.
func (r Range) TextAlign() RangePropertyValue[gpuiutil.Option[a11y.TextAlign]] {
	return fetchOptionalProperty(r, (*Node).TextAlign)
}

// VerticalOffset returns the vertical offset over the range.
func (r Range) VerticalOffset() RangePropertyValue[gpuiutil.Option[a11y.VerticalOffset]] {
	return fetchOptionalProperty(r, (*Node).VerticalOffset)
}

// IsItalic returns whether the text over the range is italic.
func (r Range) IsItalic() RangePropertyValue[bool] {
	return fetchProperty(r, (*Node).IsItalic)
}

func sameOptional[T comparable](a T, aok bool, b T, bok bool) bool {
	return aok == bok && (!aok || a == b)
}

func (n Node) textAttributesDiffer(other Node) bool {
	return !sameOptional2(n.FontFamily, other.FontFamily) ||
		!sameOptional2(n.Language, other.Language) ||
		!sameOptional2(n.FontSize, other.FontSize) ||
		!sameOptional2(n.FontWeight, other.FontWeight) ||
		!sameOptional2(n.BackgroundColor, other.BackgroundColor) ||
		!sameOptional2(n.ForegroundColor, other.ForegroundColor) ||
		!sameOptional2(n.Overline, other.Overline) ||
		!sameOptional2(n.Strikethrough, other.Strikethrough) ||
		!sameOptional2(n.Underline, other.Underline) ||
		!sameOptional2(n.TextAlign, other.TextAlign) ||
		!sameOptional2(n.VerticalOffset, other.VerticalOffset)
}

func sameOptional2[T comparable](a, b func() (T, bool)) bool {
	av, aok := a()
	bv, bok := b()
	return sameOptional(av, aok, bv, bok)
}

func (n Node) textRuns() *FilteredChildren {
	id := n.ID()
	return n.FilteredChildren(func(node *Node) FilterResult { return textNodeFilter(id, node) })
}

func (n Node) followingTextRuns(rootNode Node) *FollowingFilteredSiblings {
	id := rootNode.ID()
	return n.FollowingFilteredSiblings(func(node *Node) FilterResult { return textNodeFilter(id, node) })
}

func (n Node) precedingTextRuns(rootNode Node) *PrecedingFilteredSiblings {
	id := rootNode.ID()
	return n.PrecedingFilteredSiblings(func(node *Node) FilterResult { return textNodeFilter(id, node) })
}

// SupportsTextRanges reports whether the node exposes text ranges.
func (n Node) SupportsTextRanges() bool {
	switch {
	case n.IsTextInput():
	case n.Role() == a11y.RoleLabel, n.Role() == a11y.RoleDocument, n.Role() == a11y.RoleTerminal:
	default:
		return false
	}
	_, ok := n.textRuns().Next()
	return ok
}

func (n Node) documentStartInner() innerPosition {
	node := unwrapNode(n.textRuns().Next())
	return innerPosition{node: node, characterIndex: 0}
}

// DocumentStart returns the start of the node's text.
func (n Node) DocumentStart() Position {
	return Position{rootNode: n, inner: n.documentStartInner()}
}

func (n Node) documentEndInner() innerPosition {
	node := unwrapNode(n.textRuns().NextBack())
	return innerPosition{node: node, characterIndex: len(node.Data().CharacterLengths())}
}

// DocumentEnd returns the end of the node's text.
func (n Node) DocumentEnd() Position {
	return Position{rootNode: n, inner: n.documentEndInner()}
}

// DocumentRange returns the range covering the node's text.
func (n Node) DocumentRange() Range {
	start := n.documentStartInner()
	end := n.documentEndInner()
	return newRange(n, start, end)
}

// HasTextSelection reports whether the node has a text selection.
func (n Node) HasTextSelection() bool {
	_, ok := n.Data().TextSelection()
	return ok
}

func (n Node) clampedUpgrade(weak a11y.TextPosition) innerPosition {
	pos, ok := clampedUpgradeInnerPosition(n.TreeState, weak, n.id)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return pos
}

// TextSelection returns the node's text selection as a range.
func (n Node) TextSelection() (Range, bool) {
	selection, ok := n.Data().TextSelection()
	if !ok {
		return Range{}, false
	}
	anchor := n.clampedUpgrade(selection.Anchor)
	focus := n.clampedUpgrade(selection.Focus)
	return newRange(n, anchor, focus), true
}

// TextSelectionAnchor returns the anchor of the node's text selection.
func (n Node) TextSelectionAnchor() (Position, bool) {
	selection, ok := n.Data().TextSelection()
	if !ok {
		return Position{}, false
	}
	return Position{rootNode: n, inner: n.clampedUpgrade(selection.Anchor)}, true
}

// TextSelectionFocus returns the focus of the node's text selection.
func (n Node) TextSelectionFocus() (Position, bool) {
	selection, ok := n.Data().TextSelection()
	if !ok {
		return Position{}, false
	}
	return Position{rootNode: n, inner: n.clampedUpgrade(selection.Focus)}, true
}

// TextPositionAtPoint returns the nearest text position to the given point in this node's coordinate space.
func (n Node) TextPositionAtPoint(point a11y.Point) Position {
	id := n.ID()
	if node, point, ok := n.hitTest(point, func(node *Node) FilterResult { return textNodeFilter(id, node) }); ok {
		if node.Role() == a11y.RoleTextRun {
			pos := innerPosition{node: node, characterIndex: characterIndexAtPoint(node, point)}
			return Position{rootNode: n, inner: pos}
		}
	}

	if node, ok := n.textRuns().Next(); ok {
		if rect, ok := node.boundingBoxInCoordinateSpace(n); ok {
			origin := rect.Origin()
			if point.X < origin.X || point.Y < origin.Y {
				return n.DocumentStart()
			}
		}
	}

	runs := n.textRuns()
	for {
		node, ok := runs.NextBack()
		if !ok {
			break
		}
		rect, ok := node.boundingBoxInCoordinateSpace(n)
		if !ok {
			continue
		}
		direction, ok := node.TextDirection()
		if !ok {
			continue
		}
		var isPastEnd bool
		switch direction {
		case a11y.TextDirectionLeftToRight:
			isPastEnd = point.Y >= rect.Y0 && point.Y < rect.Y1 && point.X >= rect.X1
		case a11y.TextDirectionRightToLeft:
			isPastEnd = point.Y >= rect.Y0 && point.Y < rect.Y1 && point.X < rect.X0
		case a11y.TextDirectionTopToBottom:
			isPastEnd = point.X >= rect.X0 && point.X < rect.X1 && point.Y >= rect.Y1
		case a11y.TextDirectionBottomToTop:
			isPastEnd = point.X >= rect.X0 && point.X < rect.X1 && point.Y < rect.Y0
		}
		if isPastEnd {
			return Position{rootNode: n, inner: innerPosition{node: node, characterIndex: len(node.Data().CharacterLengths())}}
		}
	}

	return n.DocumentEnd()
}

// LineRangeFromIndex returns the range of the line with the given index.
func (n Node) LineRangeFromIndex(lineIndex int) (Range, bool) {
	pos := n.DocumentStart()

	if lineIndex > 0 {
		if pos.IsDocumentEnd() || pos.ForwardToLineEnd().IsDocumentEnd() {
			return Range{}, false
		}
		for range lineIndex {
			if pos.IsDocumentEnd() {
				return Range{}, false
			}
			pos = pos.ForwardToLineStart()
		}
	}

	end := pos
	if !pos.IsDocumentEnd() {
		end = pos.ForwardToLineEnd()
	}
	return newRange(n, pos.inner, end.inner), true
}

func (n Node) textPositionFromGlobalIndex(index int, length func(string) int) (Position, bool) {
	total := 0
	runs := n.textRuns()
	for {
		node, ok := runs.Next()
		if !ok {
			break
		}
		nodeText := unwrapStr(node.Data().Value())
		newTotal := total + length(nodeText)
		if index >= total && index < newTotal {
			index := index - total
			utf8Length := 0
			unitLength := 0
			for characterIndex, utf8CharLength := range node.Data().CharacterLengths() {
				newUtf8Length := utf8Length + int(utf8CharLength)
				charStr := nodeText[utf8Length:newUtf8Length]
				newUnitLength := unitLength + length(charStr)
				if index >= unitLength && index < newUnitLength {
					return Position{rootNode: n, inner: innerPosition{node: node, characterIndex: characterIndex}}, true
				}
				utf8Length = newUtf8Length
				unitLength = newUnitLength
			}
			panic("index out of range")
		}
		total = newTotal
	}
	if index == total {
		return n.DocumentEnd(), true
	}
	return Position{}, false
}

// TextPositionFromGlobalUsvIndex returns the position at the given index in Unicode scalar values.
func (n Node) TextPositionFromGlobalUsvIndex(index int) (Position, bool) {
	return n.textPositionFromGlobalIndex(index, utf8.RuneCountInString)
}

// TextPositionFromGlobalUTF16Index returns the position at the given index in UTF-16 code units.
func (n Node) TextPositionFromGlobalUTF16Index(index int) (Position, bool) {
	return n.textPositionFromGlobalIndex(index, utf16Len)
}
