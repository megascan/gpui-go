package taffy

import "github.com/megascan/gpui-go/gpuiutil"

func resolveSelfAlignmentSafety(alignment AlignItems, overflows bool) AlignItemsKeyword {
	if alignment.safety == AlignmentSafetySafe && overflows {
		return AlignItemsKeywordStart
	}
	return alignment.keyword
}

func applyAlignmentFallback(freeSpace float32, numItems int, alignmentMode AlignContent) AlignContentKeyword {
	if gpuiutil.DebugAssertions && alignmentMode.keyword == AlignContentKeywordNormal {
		panic("normal must be resolved before computing content alignment")
	}
	keyword := alignmentMode.keyword
	if keyword == AlignContentKeywordNormal {
		keyword = AlignContentKeywordStretch
	}
	isSafe := alignmentMode.safety == AlignmentSafetySafe

	if numItems <= 1 || freeSpace <= 0.0 {
		switch keyword {
		case AlignContentKeywordStretch, AlignContentKeywordSpaceBetween:
			keyword, isSafe = AlignContentKeywordFlexStart, true
		case AlignContentKeywordSpaceAround, AlignContentKeywordSpaceEvenly:
			keyword, isSafe = AlignContentKeywordCenter, true
		}
	}

	if freeSpace <= 0.0 && isSafe {
		keyword = AlignContentKeywordStart
	}

	return keyword
}

func computeAlignmentOffset(
	freeSpace float32,
	numItems int,
	gap float32,
	alignmentMode AlignContentKeyword,
	layoutIsFlexReversed bool,
	isFirst bool,
) float32 {
	if isFirst {
		switch alignmentMode {
		case AlignContentKeywordStart:
			return 0.0
		case AlignContentKeywordFlexStart:
			if layoutIsFlexReversed {
				return freeSpace
			}
			return 0.0
		case AlignContentKeywordEnd:
			return freeSpace
		case AlignContentKeywordFlexEnd:
			if layoutIsFlexReversed {
				return 0.0
			}
			return freeSpace
		case AlignContentKeywordCenter:
			return freeSpace / 2.0
		case AlignContentKeywordNormal, AlignContentKeywordStretch:
			return 0.0
		case AlignContentKeywordSpaceBetween:
			return 0.0
		case AlignContentKeywordSpaceAround:
			if freeSpace >= 0.0 {
				return (freeSpace / float32(numItems)) / 2.0
			}
			return freeSpace / 2.0
		case AlignContentKeywordSpaceEvenly:
			if freeSpace >= 0.0 {
				return freeSpace / float32(numItems+1)
			}
			return freeSpace / 2.0
		}
		panic("internal error: entered unreachable code: invalid AlignContentKeyword")
	}
	freeSpace = f32Max(freeSpace, 0.0)
	var offset float32
	switch alignmentMode {
	case AlignContentKeywordStart,
		AlignContentKeywordFlexStart,
		AlignContentKeywordEnd,
		AlignContentKeywordFlexEnd,
		AlignContentKeywordCenter,
		AlignContentKeywordNormal,
		AlignContentKeywordStretch:
		offset = 0.0
	case AlignContentKeywordSpaceBetween:
		offset = freeSpace / float32(numItems-1)
	case AlignContentKeywordSpaceAround:
		offset = freeSpace / float32(numItems)
	case AlignContentKeywordSpaceEvenly:
		offset = freeSpace / float32(numItems+1)
	}
	return gap + offset
}
