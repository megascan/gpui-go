package atspi

import (
	"strconv"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
)

// WindowBounds holds a window's outer and inner (client area) rectangles in screen coordinates.
type WindowBounds struct {
	Outer a11y.Rect
	Inner a11y.Rect
}

// NewWindowBounds returns WindowBounds from its outer and inner rectangles.
func NewWindowBounds(outer, inner a11y.Rect) WindowBounds {
	return WindowBounds{Outer: outer, Inner: inner}
}

func (b *WindowBounds) accesskitPointToAtspiPoint(point a11y.Point, parent *consumer.Node, coordType CoordType) a11y.Point {
	origin := b.origin(parent, coordType)
	return a11y.NewPoint(origin.X+point.X, origin.Y+point.Y)
}

func (b *WindowBounds) atspiPointToAccesskitPoint(point a11y.Point, parent *consumer.Node, coordType CoordType) a11y.Point {
	origin := b.origin(parent, coordType)
	return a11y.NewPoint(point.X-origin.X, point.Y-origin.Y)
}

func (b *WindowBounds) origin(parent *consumer.Node, coordType CoordType) a11y.Point {
	switch coordType {
	case CoordTypeScreen:
		return b.Inner.Origin()
	case CoordTypeWindow:
		return a11y.PointZero()
	default:
		if parent != nil {
			bounds, _ := parent.BoundingBox()
			parentOrigin := bounds.Origin()
			return a11y.NewPoint(-parentOrigin.X, -parentOrigin.Y)
		}
		return b.Inner.Origin()
	}
}

func optionalNode(node consumer.Node, ok bool) *consumer.Node {
	if !ok {
		return nil
	}
	return &node
}

func textPositionFromOffset(node *consumer.Node, offset int32) (consumer.Position, bool) {
	if offset < 0 {
		return consumer.Position{}, false
	}
	return node.TextPositionFromGlobalUsvIndex(int(offset))
}

func textRangeFromOffset(node *consumer.Node, offset int32, granularity Granularity) (consumer.Range, error) {
	startOffset, ok := textPositionFromOffset(node, offset)
	if !ok {
		return consumer.Range{}, ErrorIndexOutOfRange
	}
	var start consumer.Position
	switch granularity {
	case GranularityChar:
		start = startOffset
	case GranularityLine:
		if startOffset.IsLineStart() {
			start = startOffset
		} else {
			start = startOffset.BackwardToLineStart()
		}
	case GranularityParagraph:
		if startOffset.IsParagraphStart() {
			start = startOffset
		} else {
			start = startOffset.BackwardToParagraphStart()
		}
	case GranularitySentence:
		return consumer.Range{}, ErrorUnsupportedTextGranularity
	default:
		if startOffset.IsWordStart() {
			start = startOffset
		} else {
			start = startOffset.BackwardToWordStart()
		}
	}
	var end consumer.Position
	switch granularity {
	case GranularityChar:
		if startOffset.IsDocumentEnd() {
			end = startOffset
		} else {
			end = start.ForwardToCharacterEnd()
		}
	case GranularityLine:
		end = start.ForwardToLineEnd()
	case GranularityParagraph:
		end = start.ForwardToParagraphEnd()
	case GranularitySentence:
		return consumer.Range{}, ErrorUnsupportedTextGranularity
	default:
		end = start.ForwardToWordEnd()
	}
	r := start.ToDegenerateRange()
	r.SetEnd(end)
	return r, nil
}

func textRangeFromOffsets(node *consumer.Node, startOffset, endOffset int32) (consumer.Range, bool) {
	start, ok := textPositionFromOffset(node, startOffset)
	if !ok {
		return consumer.Range{}, false
	}
	var end consumer.Position
	if endOffset == -1 {
		end = node.DocumentRange().End()
	} else {
		end, ok = textPositionFromOffset(node, endOffset)
		if !ok {
			return consumer.Range{}, false
		}
	}
	r := start.ToDegenerateRange()
	r.SetEnd(end)
	return r, true
}

func textRangeBoundsFromOffsets(node *consumer.Node, startOffset, endOffset int32) (a11y.Rect, bool) {
	r, ok := textRangeFromOffsets(node, startOffset, endOffset)
	if !ok {
		return a11y.Rect{}, false
	}
	boxes := r.BoundingBoxes()
	if len(boxes) == 0 {
		return a11y.Rect{}, false
	}
	result := boxes[0]
	for _, rect := range boxes[1:] {
		result = result.Union(rect)
	}
	return result, true
}

func atspiScrollTypeToScrollHint(scrollType ScrollType) (a11y.ScrollHint, bool) {
	switch scrollType {
	case ScrollTypeTopLeft:
		return a11y.ScrollHintTopLeft, true
	case ScrollTypeBottomRight:
		return a11y.ScrollHintBottomRight, true
	case ScrollTypeTopEdge:
		return a11y.ScrollHintTopEdge, true
	case ScrollTypeBottomEdge:
		return a11y.ScrollHintBottomEdge, true
	case ScrollTypeLeftEdge:
		return a11y.ScrollHintLeftEdge, true
	case ScrollTypeRightEdge:
		return a11y.ScrollHintRightEdge, true
	default:
		return 0, false
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
