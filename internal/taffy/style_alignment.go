package taffy

// AlignItemsKeyword is the position-keyword half of AlignItems (and its aliases AlignSelf, JustifyItems, JustifySelf).
type AlignItemsKeyword uint8

const (
	AlignItemsKeywordNormal AlignItemsKeyword = iota
	AlignItemsKeywordStart
	AlignItemsKeywordEnd
	AlignItemsKeywordFlexStart
	AlignItemsKeywordFlexEnd
	AlignItemsKeywordSelfStart
	AlignItemsKeywordSelfEnd
	AlignItemsKeywordCenter
	AlignItemsKeywordBaseline
	AlignItemsKeywordStretch
)

// AlignItemsKeywordDefault is the Default value of AlignItemsKeyword.
const AlignItemsKeywordDefault = AlignItemsKeywordNormal

// AlignContentKeyword is the position-keyword half of AlignContent (and its alias JustifyContent).
type AlignContentKeyword uint8

const (
	AlignContentKeywordNormal AlignContentKeyword = iota
	AlignContentKeywordStart
	AlignContentKeywordEnd
	AlignContentKeywordFlexStart
	AlignContentKeywordFlexEnd
	AlignContentKeywordCenter
	AlignContentKeywordStretch
	AlignContentKeywordSpaceBetween
	AlignContentKeywordSpaceEvenly
	AlignContentKeywordSpaceAround
)

// AlignContentKeywordDefault is the Default value of AlignContentKeyword.
const AlignContentKeywordDefault = AlignContentKeywordNormal

func (k AlignContentKeyword) reversed() AlignContentKeyword {
	switch k {
	case AlignContentKeywordStart:
		return AlignContentKeywordEnd
	case AlignContentKeywordEnd:
		return AlignContentKeywordStart
	case AlignContentKeywordFlexStart:
		return AlignContentKeywordFlexEnd
	case AlignContentKeywordFlexEnd:
		return AlignContentKeywordFlexStart
	case AlignContentKeywordStretch:
		return AlignContentKeywordEnd
	}
	return k
}

// AlignmentSafety is the overflow-position modifier of an alignment value.
type AlignmentSafety uint8

const (
	AlignmentSafetyDefault AlignmentSafety = iota
	AlignmentSafetyUnsafe
	AlignmentSafetySafe
)

// AlignItems is how items are aligned along an axis: a position keyword plus a safety modifier.
type AlignItems struct {
	keyword AlignItemsKeyword
	safety  AlignmentSafety
}

// NewAlignItems builds an AlignItems from its keyword and safety fields.
func NewAlignItems(keyword AlignItemsKeyword, safety AlignmentSafety) AlignItems {
	return AlignItems{keyword: keyword, safety: safety}
}

var (
	AlignItemsNormal        = AlignItems{AlignItemsKeywordNormal, AlignmentSafetyDefault}
	AlignItemsStart         = AlignItems{AlignItemsKeywordStart, AlignmentSafetyDefault}
	AlignItemsEnd           = AlignItems{AlignItemsKeywordEnd, AlignmentSafetyDefault}
	AlignItemsFlexStart     = AlignItems{AlignItemsKeywordFlexStart, AlignmentSafetyDefault}
	AlignItemsFlexEnd       = AlignItems{AlignItemsKeywordFlexEnd, AlignmentSafetyDefault}
	AlignItemsSelfStart     = AlignItems{AlignItemsKeywordSelfStart, AlignmentSafetyDefault}
	AlignItemsSelfEnd       = AlignItems{AlignItemsKeywordSelfEnd, AlignmentSafetyDefault}
	AlignItemsCenter        = AlignItems{AlignItemsKeywordCenter, AlignmentSafetyDefault}
	AlignItemsBaseline      = AlignItems{AlignItemsKeywordBaseline, AlignmentSafetyDefault}
	AlignItemsStretch       = AlignItems{AlignItemsKeywordStretch, AlignmentSafetyDefault}
	AlignItemsSafeStart     = AlignItems{AlignItemsKeywordStart, AlignmentSafetySafe}
	AlignItemsSafeEnd       = AlignItems{AlignItemsKeywordEnd, AlignmentSafetySafe}
	AlignItemsSafeFlexStart = AlignItems{AlignItemsKeywordFlexStart, AlignmentSafetySafe}
	AlignItemsSafeFlexEnd   = AlignItems{AlignItemsKeywordFlexEnd, AlignmentSafetySafe}
	AlignItemsSafeCenter    = AlignItems{AlignItemsKeywordCenter, AlignmentSafetySafe}
	AlignItemsSafeSelfStart = AlignItems{AlignItemsKeywordSelfStart, AlignmentSafetySafe}
	AlignItemsSafeSelfEnd   = AlignItems{AlignItemsKeywordSelfEnd, AlignmentSafetySafe}
)

// AlignItemsDefault is the Default value of AlignItems.
var AlignItemsDefault = AlignItemsNormal

// IsSafe reports whether the safe overflow-position modifier is set.
func (a AlignItems) IsSafe() bool {
	return a.safety == AlignmentSafetySafe
}

// WithSafety returns a copy with the given safety modifier.
func (a AlignItems) WithSafety(safety AlignmentSafety) AlignItems {
	return AlignItems{keyword: a.keyword, safety: safety}
}

// Keyword returns the position keyword.
func (a AlignItems) Keyword() AlignItemsKeyword {
	return a.keyword
}

// Safety returns the overflow-position modifier.
func (a AlignItems) Safety() AlignmentSafety {
	return a.safety
}

// IsStretchOrNormal reports whether the keyword is Stretch or Normal.
func (a AlignItems) IsStretchOrNormal() bool {
	return a.keyword == AlignItemsKeywordStretch || a.keyword == AlignItemsKeywordNormal
}

// ResolveSelfRelative resolves SelfStart / SelfEnd to Start / End for an item of itemDirection in a container of containerDirection.
func (a AlignItems) ResolveSelfRelative(itemDirection, containerDirection Direction, axisIsInline bool) AlignItems {
	flip := axisIsInline && itemDirection != containerDirection
	keyword := a.keyword
	switch a.keyword {
	case AlignItemsKeywordSelfStart:
		if flip {
			keyword = AlignItemsKeywordEnd
		} else {
			keyword = AlignItemsKeywordStart
		}
	case AlignItemsKeywordSelfEnd:
		if flip {
			keyword = AlignItemsKeywordStart
		} else {
			keyword = AlignItemsKeywordEnd
		}
	}
	return AlignItems{keyword: keyword, safety: a.safety}
}

// JustifyItems is used to control how child nodes are aligned in the inline axis.
type JustifyItems = AlignItems

// AlignSelf controls alignment of an individual node in the cross/block axis.
type AlignSelf = AlignItems

// JustifySelf controls alignment of an individual node in the inline axis.
type JustifySelf = AlignItems

// AlignContent sets the distribution of space between and around content items: a position keyword plus a safety modifier.
type AlignContent struct {
	keyword AlignContentKeyword
	safety  AlignmentSafety
}

// NewAlignContent builds an AlignContent from its keyword and safety fields.
func NewAlignContent(keyword AlignContentKeyword, safety AlignmentSafety) AlignContent {
	return AlignContent{keyword: keyword, safety: safety}
}

var (
	AlignContentNormal        = AlignContent{AlignContentKeywordNormal, AlignmentSafetyDefault}
	AlignContentStart         = AlignContent{AlignContentKeywordStart, AlignmentSafetyDefault}
	AlignContentEnd           = AlignContent{AlignContentKeywordEnd, AlignmentSafetyDefault}
	AlignContentFlexStart     = AlignContent{AlignContentKeywordFlexStart, AlignmentSafetyDefault}
	AlignContentFlexEnd       = AlignContent{AlignContentKeywordFlexEnd, AlignmentSafetyDefault}
	AlignContentCenter        = AlignContent{AlignContentKeywordCenter, AlignmentSafetyDefault}
	AlignContentStretch       = AlignContent{AlignContentKeywordStretch, AlignmentSafetyDefault}
	AlignContentSpaceBetween  = AlignContent{AlignContentKeywordSpaceBetween, AlignmentSafetyDefault}
	AlignContentSpaceEvenly   = AlignContent{AlignContentKeywordSpaceEvenly, AlignmentSafetyDefault}
	AlignContentSpaceAround   = AlignContent{AlignContentKeywordSpaceAround, AlignmentSafetyDefault}
	AlignContentSafeStart     = AlignContent{AlignContentKeywordStart, AlignmentSafetySafe}
	AlignContentSafeEnd       = AlignContent{AlignContentKeywordEnd, AlignmentSafetySafe}
	AlignContentSafeFlexStart = AlignContent{AlignContentKeywordFlexStart, AlignmentSafetySafe}
	AlignContentSafeFlexEnd   = AlignContent{AlignContentKeywordFlexEnd, AlignmentSafetySafe}
	AlignContentSafeCenter    = AlignContent{AlignContentKeywordCenter, AlignmentSafetySafe}
)

// AlignContentDefault is the Default value of AlignContent.
var AlignContentDefault = AlignContentNormal

// IsSafe reports whether the safe overflow-position modifier is set.
func (a AlignContent) IsSafe() bool {
	return a.safety == AlignmentSafetySafe
}

// WithSafety returns a copy with the given safety modifier.
func (a AlignContent) WithSafety(safety AlignmentSafety) AlignContent {
	return AlignContent{keyword: a.keyword, safety: safety}
}

// Keyword returns the position keyword.
func (a AlignContent) Keyword() AlignContentKeyword {
	return a.keyword
}

// Safety returns the overflow-position modifier.
func (a AlignContent) Safety() AlignmentSafety {
	return a.safety
}

// JustifyContent sets the distribution of space between and around content items in the main/inline axis.
type JustifyContent = AlignContent
