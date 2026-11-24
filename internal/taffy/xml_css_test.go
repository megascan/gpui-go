package taffy

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

type cssTokenKind uint8

const (
	cssTokenIdent cssTokenKind = iota
	cssTokenNumber
	cssTokenPercentage
	cssTokenDimension
	cssTokenFunction
	cssTokenParenBlock
	cssTokenSquareBracketBlock
	cssTokenComma
	cssTokenDelim
)

type cssToken struct {
	kind      cssTokenKind
	text      string
	value     float32
	unitValue float32
	intValue  int32
	hasInt    bool
	children  []cssToken
}

func (t cssToken) String() string {
	switch t.kind {
	case cssTokenIdent:
		return fmt.Sprintf("Ident(%q)", t.text)
	case cssTokenNumber:
		return fmt.Sprintf("Number(%v)", t.value)
	case cssTokenPercentage:
		return fmt.Sprintf("Percentage(%v)", t.unitValue)
	case cssTokenDimension:
		return fmt.Sprintf("Dimension(%v%s)", t.value, t.text)
	case cssTokenFunction:
		return fmt.Sprintf("Function(%q)", t.text)
	case cssTokenParenBlock:
		return "ParenthesisBlock"
	case cssTokenSquareBracketBlock:
		return "SquareBracketBlock"
	case cssTokenComma:
		return "Comma"
	default:
		return fmt.Sprintf("Delim(%q)", t.text)
	}
}

var errCSSEndOfInput = errors.New("EndOfInput")

func cssUnexpected(t cssToken) error {
	return fmt.Errorf("unexpected token: %v", t)
}

func cssIsNameStart(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func cssIsName(c rune) bool {
	return cssIsNameStart(c) || c == '-' || (c >= '0' && c <= '9')
}

func cssIsDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func cssIsWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func cssRuneAt(s string, i int) rune {
	if i >= len(s) {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
}

func cssStartsIdent(s string, i int) bool {
	c := cssRuneAt(s, i)
	if c == '-' {
		n := cssRuneAt(s, i+1)
		return n == '-' || cssIsNameStart(n)
	}
	return cssIsNameStart(c)
}

func cssStartsNumber(s string, i int) bool {
	if i >= len(s) {
		return false
	}
	switch s[i] {
	case '+', '-':
		i++
	}
	if i < len(s) && cssIsDigit(s[i]) {
		return true
	}
	return i+1 < len(s) && s[i] == '.' && cssIsDigit(s[i+1])
}

func cssConsumeName(s string, i int) (string, int) {
	start := i
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !cssIsName(r) {
			break
		}
		i += size
	}
	return s[start:i], i
}

func cssTokenize(s string, i int, closer byte) ([]cssToken, int, error) {
	var tokens []cssToken
	for i < len(s) {
		c := s[i]
		switch {
		case cssIsWhitespace(c):
			i++
		case closer != 0 && c == closer:
			return tokens, i + 1, nil
		case cssStartsNumber(s, i):
			start := i
			if s[i] == '+' || s[i] == '-' {
				i++
			}
			isInt := true
			for i < len(s) && cssIsDigit(s[i]) {
				i++
			}
			if i+1 < len(s) && s[i] == '.' && cssIsDigit(s[i+1]) {
				isInt = false
				i++
				for i < len(s) && cssIsDigit(s[i]) {
					i++
				}
			}
			if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
				j := i + 1
				if j < len(s) && (s[j] == '+' || s[j] == '-') {
					j++
				}
				if j < len(s) && cssIsDigit(s[j]) {
					isInt = false
					i = j
					for i < len(s) && cssIsDigit(s[i]) {
						i++
					}
				}
			}
			v, err := strconv.ParseFloat(s[start:i], 64)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				return nil, i, fmt.Errorf("invalid number %q: %w", s[start:i], err)
			}
			tok := cssToken{kind: cssTokenNumber, value: float32(v)}
			if isInt {
				tok.hasInt = true
				tok.intValue = int32(max(math.MinInt32, min(math.MaxInt32, v)))
			}
			switch {
			case i < len(s) && s[i] == '%':
				i++
				tok.kind = cssTokenPercentage
				tok.unitValue = float32(v / 100)
			case cssStartsIdent(s, i):
				tok.kind = cssTokenDimension
				tok.text, i = cssConsumeName(s, i)
			}
			tokens = append(tokens, tok)
		case cssStartsIdent(s, i):
			var name string
			name, i = cssConsumeName(s, i)
			if i < len(s) && s[i] == '(' {
				children, next, err := cssTokenize(s, i+1, ')')
				if err != nil {
					return nil, next, err
				}
				i = next
				tokens = append(tokens, cssToken{kind: cssTokenFunction, text: name, children: children})
			} else {
				tokens = append(tokens, cssToken{kind: cssTokenIdent, text: name})
			}
		case c == '(' || c == '[':
			kind, end := cssTokenParenBlock, byte(')')
			if c == '[' {
				kind, end = cssTokenSquareBracketBlock, ']'
			}
			children, next, err := cssTokenize(s, i+1, end)
			if err != nil {
				return nil, next, err
			}
			i = next
			tokens = append(tokens, cssToken{kind: kind, children: children})
		case c == ',':
			i++
			tokens = append(tokens, cssToken{kind: cssTokenComma})
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			i += size
			tokens = append(tokens, cssToken{kind: cssTokenDelim, text: string(r)})
		}
	}
	return tokens, i, nil
}

type cssParser struct {
	tokens []cssToken
	pos    int
}

func (p *cssParser) isExhausted() bool {
	return p.pos >= len(p.tokens)
}

func (p *cssParser) next() (cssToken, error) {
	if p.isExhausted() {
		return cssToken{}, errCSSEndOfInput
	}
	t := p.tokens[p.pos]
	p.pos++
	return t, nil
}

func (p *cssParser) expectExhausted() error {
	if !p.isExhausted() {
		return cssUnexpected(p.tokens[p.pos])
	}
	return nil
}

func (p *cssParser) expectIdent() (string, error) {
	t, err := p.next()
	if err != nil {
		return "", err
	}
	if t.kind != cssTokenIdent {
		return "", cssUnexpected(t)
	}
	return t.text, nil
}

func (p *cssParser) expectComma() error {
	t, err := p.next()
	if err != nil {
		return err
	}
	if t.kind != cssTokenComma {
		return cssUnexpected(t)
	}
	return nil
}

func (p *cssParser) expectFunctionMatching(name string) (cssToken, error) {
	t, err := p.next()
	if err != nil {
		return t, err
	}
	if t.kind != cssTokenFunction || !strings.EqualFold(t.text, name) {
		return t, cssUnexpected(t)
	}
	return t, nil
}

func (p *cssParser) expectSquareBracketBlock() (cssToken, error) {
	t, err := p.next()
	if err != nil {
		return t, err
	}
	if t.kind != cssTokenSquareBracketBlock {
		return t, cssUnexpected(t)
	}
	return t, nil
}

func cssTryParse[T any](p *cssParser, f func(*cssParser) (T, error)) (T, error) {
	saved := p.pos
	v, err := f(p)
	if err != nil {
		p.pos = saved
	}
	return v, err
}

func cssParseNestedBlock[T any](block cssToken, f func(*cssParser) (T, error)) (T, error) {
	return cssParseEntirely(&cssParser{tokens: block.children}, f)
}

func cssParseEntirely[T any](p *cssParser, f func(*cssParser) (T, error)) (T, error) {
	v, err := f(p)
	if err != nil {
		return v, err
	}
	if err := p.expectExhausted(); err != nil {
		var zero T
		return zero, err
	}
	return v, nil
}

func parseCSSStrEntirely[T any](input string, f func(*cssParser) (T, error)) (T, error) {
	tokens, _, err := cssTokenize(input, 0, 0)
	if err != nil {
		var zero T
		return zero, err
	}
	return cssParseEntirely(&cssParser{tokens: tokens}, f)
}

func cssKeywordEnum[T any](keywords map[string]T) func(*cssParser) (T, error) {
	return func(p *cssParser) (T, error) {
		ident, err := p.expectIdent()
		if err != nil {
			var zero T
			return zero, err
		}
		if v, ok := keywords[strings.ToLower(ident)]; ok {
			return v, nil
		}
		var zero T
		return zero, cssUnexpected(cssToken{kind: cssTokenIdent, text: ident})
	}
}

var (
	cssDisplay = cssKeywordEnum(map[string]Display{
		"none": DisplayNone, "flex": DisplayFlex, "grid": DisplayGrid, "block": DisplayBlock, "flow-root": DisplayFlowRoot,
	})
	cssPosition = cssKeywordEnum(map[string]Position{
		"static": PositionStatic, "relative": PositionRelative, "absolute": PositionAbsolute, "fixed": PositionFixed, "sticky": PositionSticky,
	})
	cssBoxSizing = cssKeywordEnum(map[string]BoxSizing{"border-box": BoxSizingBorderBox, "content-box": BoxSizingContentBox})
	cssOverflow  = cssKeywordEnum(map[string]Overflow{
		"visible": OverflowVisible, "hidden": OverflowHidden, "clip": OverflowClip, "scroll": OverflowScroll,
	})
	cssDirection     = cssKeywordEnum(map[string]Direction{"ltr": DirectionLtr, "rtl": DirectionRtl})
	cssFlexWrap      = cssKeywordEnum(map[string]FlexWrap{"nowrap": FlexWrapNoWrap, "wrap": FlexWrapWrap, "wrap-reverse": FlexWrapWrapReverse})
	cssFlexDirection = cssKeywordEnum(map[string]FlexDirection{
		"row": FlexDirectionRow, "column": FlexDirectionColumn, "row-reverse": FlexDirectionRowReverse, "column-reverse": FlexDirectionColumnReverse,
	})
	cssTextAlign = cssKeywordEnum(map[string]TextAlign{
		"auto": TextAlignAuto, "-webkit-left": TextAlignLegacyLeft, "-webkit-right": TextAlignLegacyRight, "-webkit-center": TextAlignLegacyCenter,
	})
	cssFloat = cssKeywordEnum(map[string]Float{"left": FloatLeft, "right": FloatRight, "none": FloatNone})
	cssClear = cssKeywordEnum(map[string]Clear{"left": ClearLeft, "right": ClearRight, "both": ClearBoth, "none": ClearNone})
)

func cssAlignItems(p *cssParser) (AlignItems, error) {
	first, err := p.expectIdent()
	if err != nil {
		return AlignItems{}, err
	}
	positions := map[string]AlignItemsKeyword{
		"start": AlignItemsKeywordStart, "end": AlignItemsKeywordEnd, "flex-start": AlignItemsKeywordFlexStart,
		"flex-end": AlignItemsKeywordFlexEnd, "self-start": AlignItemsKeywordSelfStart, "self-end": AlignItemsKeywordSelfEnd,
		"center": AlignItemsKeywordCenter,
	}
	switch strings.ToLower(first) {
	case "safe", "unsafe":
		pos, err := p.expectIdent()
		if err != nil {
			return AlignItems{}, err
		}
		keyword, ok := positions[strings.ToLower(pos)]
		if !ok {
			return AlignItems{}, cssUnexpected(cssToken{kind: cssTokenIdent, text: pos})
		}
		if strings.EqualFold(first, "safe") {
			return NewAlignItems(keyword, AlignmentSafetySafe), nil
		}
		return NewAlignItems(keyword, AlignmentSafetyUnsafe), nil
	}
	plain := map[string]AlignItems{
		"normal": AlignItemsNormal, "start": AlignItemsStart, "end": AlignItemsEnd, "flex-start": AlignItemsFlexStart,
		"flex-end": AlignItemsFlexEnd, "self-start": AlignItemsSelfStart, "self-end": AlignItemsSelfEnd,
		"center": AlignItemsCenter, "baseline": AlignItemsBaseline, "stretch": AlignItemsStretch,
	}
	if v, ok := plain[strings.ToLower(first)]; ok {
		return v, nil
	}
	return AlignItems{}, cssUnexpected(cssToken{kind: cssTokenIdent, text: first})
}

func cssAlignContent(p *cssParser) (AlignContent, error) {
	first, err := p.expectIdent()
	if err != nil {
		return AlignContent{}, err
	}
	positions := map[string]AlignContentKeyword{
		"start": AlignContentKeywordStart, "end": AlignContentKeywordEnd, "flex-start": AlignContentKeywordFlexStart,
		"flex-end": AlignContentKeywordFlexEnd, "center": AlignContentKeywordCenter,
	}
	switch strings.ToLower(first) {
	case "safe", "unsafe":
		pos, err := p.expectIdent()
		if err != nil {
			return AlignContent{}, err
		}
		keyword, ok := positions[strings.ToLower(pos)]
		if !ok {
			return AlignContent{}, cssUnexpected(cssToken{kind: cssTokenIdent, text: pos})
		}
		if strings.EqualFold(first, "safe") {
			return NewAlignContent(keyword, AlignmentSafetySafe), nil
		}
		return NewAlignContent(keyword, AlignmentSafetyUnsafe), nil
	}
	plain := map[string]AlignContent{
		"normal": AlignContentNormal, "start": AlignContentStart, "end": AlignContentEnd, "flex-start": AlignContentFlexStart,
		"flex-end": AlignContentFlexEnd, "center": AlignContentCenter, "stretch": AlignContentStretch,
		"space-between": AlignContentSpaceBetween, "space-evenly": AlignContentSpaceEvenly, "space-around": AlignContentSpaceAround,
	}
	if v, ok := plain[strings.ToLower(first)]; ok {
		return v, nil
	}
	return AlignContent{}, cssUnexpected(cssToken{kind: cssTokenIdent, text: first})
}

func cssContain(p *cssParser) (Contain, error) {
	const styleBit uint8 = 1 << 6
	flags := ContainNone
	var seen uint8
	for {
		ident, err := p.expectIdent()
		if err != nil {
			return 0, err
		}
		unexpected := cssUnexpected(cssToken{kind: cssTokenIdent, text: ident})
		var flag Contain
		var seenBit uint8
		switch strings.ToLower(ident) {
		case "none", "content":
			if seen != 0 || !p.isExhausted() {
				return 0, unexpected
			}
			if strings.EqualFold(ident, "content") {
				return ContainContent, nil
			}
			return ContainNone, nil
		case "layout":
			flag, seenBit = ContainLayout, uint8(ContainLayout)
		case "paint":
			flag, seenBit = ContainPaint, uint8(ContainPaint)
		case "style":
			flag, seenBit = ContainNone, styleBit
		default:
			return 0, unexpected
		}
		if seen&seenBit != 0 {
			return 0, unexpected
		}
		seen |= seenBit
		flags = flags.Union(flag)
		if p.isExhausted() {
			return flags, nil
		}
	}
}

func cssAvailableSpace(p *cssParser) (AvailableSpace, error) {
	t, err := p.next()
	if err != nil {
		return AvailableSpace{}, err
	}
	switch {
	case t.kind == cssTokenNumber && t.value >= 0.0, t.kind == cssTokenDimension && t.value >= 0.0:
		return AvailableSpaceDefinite(t.value), nil
	case t.kind == cssTokenIdent && t.text == "max-content":
		return AvailableSpaceMaxContent, nil
	case t.kind == cssTokenIdent && t.text == "min-content":
		return AvailableSpaceMinContent, nil
	}
	return AvailableSpace{}, cssUnexpected(t)
}

func cssLengthPercentage(p *cssParser) (LengthPercentage, error) {
	t, err := p.next()
	if err != nil {
		return LengthPercentage{}, err
	}
	switch {
	case t.kind == cssTokenPercentage:
		return LengthPercentagePercent(t.unitValue), nil
	case t.kind == cssTokenDimension && t.text == "px":
		return LengthPercentageLength(t.value), nil
	}
	return LengthPercentage{}, cssUnexpected(t)
}

func cssLengthPercentageAuto(p *cssParser) (LengthPercentageAuto, error) {
	t, err := p.next()
	if err != nil {
		return LengthPercentageAuto{}, err
	}
	switch {
	case t.kind == cssTokenPercentage:
		return LengthPercentageAutoPercent(t.unitValue), nil
	case t.kind == cssTokenDimension && t.text == "px":
		return LengthPercentageAutoLength(t.value), nil
	case t.kind == cssTokenIdent && t.text == "auto":
		return LengthPercentageAutoAuto(), nil
	}
	return LengthPercentageAuto{}, cssUnexpected(t)
}

func cssDimension(p *cssParser) (Dimension, error) {
	t, err := p.next()
	if err != nil {
		return Dimension{}, err
	}
	switch {
	case t.kind == cssTokenPercentage:
		return DimensionPercent(t.unitValue), nil
	case t.kind == cssTokenDimension && t.text == "px":
		return DimensionLength(t.value), nil
	case t.kind == cssTokenIdent:
		switch t.text {
		case "auto":
			return DimensionAuto(), nil
		case "min-content":
			return DimensionMinContent(), nil
		case "max-content":
			return DimensionMaxContent(), nil
		case "fit-content":
			return DimensionFitContent(), nil
		case "stretch":
			return DimensionStretch(), nil
		case "content":
			return DimensionContent(), nil
		}
	case t.kind == cssTokenFunction && t.text == "fit-content":
		return cssParseNestedBlock(t, func(p *cssParser) (Dimension, error) {
			t, err := p.next()
			if err != nil {
				return Dimension{}, err
			}
			switch {
			case t.kind == cssTokenPercentage:
				return DimensionFitContentPercent(t.unitValue), nil
			case t.kind == cssTokenDimension && t.text == "px":
				return DimensionFitContentPx(t.value), nil
			}
			return Dimension{}, cssUnexpected(t)
		})
	}
	return Dimension{}, cssUnexpected(t)
}

func cssGridAutoFlow(p *cssParser) (GridAutoFlow, error) {
	axis := ""
	dense := false
	for range 2 {
		ident, err := cssTryParse(p, (*cssParser).expectIdent)
		if err != nil {
			break
		}
		switch ident {
		case "row":
			axis = "row"
		case "column":
			axis = "column"
		case "dense":
			dense = true
		default:
			return 0, cssUnexpected(cssToken{kind: cssTokenIdent, text: ident})
		}
	}
	switch {
	case axis == "row" && !dense:
		return GridAutoFlowRow, nil
	case (axis == "row" || axis == "") && dense:
		return GridAutoFlowRowDense, nil
	case axis == "column" && !dense:
		return GridAutoFlowColumn, nil
	case axis == "column" && dense:
		return GridAutoFlowColumnDense, nil
	}
	t, err := p.next()
	if err != nil {
		return 0, err
	}
	return 0, cssUnexpected(t)
}

func saturatingI16(value int32) int16 {
	return int16(max(math.MinInt16, min(math.MaxInt16, value)))
}

func saturatingU16(value int32) uint16 {
	return uint16(max(0, min(math.MaxUint16, value)))
}

func cssGridPlacement(p *cssParser) (GridPlacement[defaultCheapStr], error) {
	span := false
	var number Option[int32]
	var ident Option[string]
	for !p.isExhausted() {
		t, err := p.next()
		if err != nil {
			return GridPlacement[defaultCheapStr]{}, err
		}
		switch {
		case t.kind == cssTokenIdent:
			switch t.text {
			case "auto":
				if span || number.IsSome() || ident.IsSome() {
					return GridPlacement[defaultCheapStr]{}, cssUnexpected(t)
				}
				if err := p.expectExhausted(); err != nil {
					return GridPlacement[defaultCheapStr]{}, err
				}
				return GridPlacementAuto[defaultCheapStr](), nil
			case "span":
				if span {
					return GridPlacement[defaultCheapStr]{}, cssUnexpected(t)
				}
				span = true
			default:
				if ident.IsSome() {
					return GridPlacement[defaultCheapStr]{}, cssUnexpected(t)
				}
				ident = some(t.text)
			}
		case t.kind == cssTokenNumber && t.hasInt && t.intValue != 0:
			if number.IsSome() {
				return GridPlacement[defaultCheapStr]{}, cssUnexpected(t)
			}
			number = some(t.intValue)
		default:
			return GridPlacement[defaultCheapStr]{}, cssUnexpected(t)
		}
	}
	n, hasNumber := number.Get()
	name, hasIdent := ident.Get()
	switch {
	case span && !hasNumber && !hasIdent:
		return GridPlacementSpan[defaultCheapStr](0), nil
	case span && hasNumber && !hasIdent:
		return GridPlacementSpan[defaultCheapStr](saturatingU16(n)), nil
	case span && !hasNumber && hasIdent:
		return GridPlacementNamedSpan(name, 0), nil
	case span:
		return GridPlacementNamedSpan(name, saturatingU16(n)), nil
	case hasNumber && !hasIdent:
		return GridPlacementLine[defaultCheapStr](GridLineFromI16(saturatingI16(n))), nil
	case hasNumber:
		return GridPlacementNamedLine(name, saturatingI16(n)), nil
	case hasIdent:
		return GridPlacementNamedLine(name, 0), nil
	}
	return GridPlacement[defaultCheapStr]{}, errCSSEndOfInput
}

func cssMaxTrackSizingFunction(p *cssParser) (MaxTrackSizingFunction, error) {
	t, err := p.next()
	if err != nil {
		return MaxTrackSizingFunction{}, err
	}
	switch {
	case t.kind == cssTokenPercentage:
		return MaxTrackSizingFunctionPercent(t.unitValue), nil
	case t.kind == cssTokenDimension && t.text == "px":
		return MaxTrackSizingFunctionLength(t.value), nil
	case t.kind == cssTokenDimension && t.text == "fr" && !math.Signbit(float64(t.value)):
		return MaxTrackSizingFunctionFr(t.value), nil
	case t.kind == cssTokenIdent:
		switch t.text {
		case "auto":
			return MaxTrackSizingFunctionAuto(), nil
		case "min-content":
			return MaxTrackSizingFunctionMinContent(), nil
		case "max-content":
			return MaxTrackSizingFunctionMaxContent(), nil
		}
	case t.kind == cssTokenFunction && t.text == "fit-content":
		return cssParseNestedBlock(t, func(p *cssParser) (MaxTrackSizingFunction, error) {
			t, err := p.next()
			if err != nil {
				return MaxTrackSizingFunction{}, err
			}
			switch {
			case t.kind == cssTokenPercentage:
				return MaxTrackSizingFunctionFitContentPercent(t.unitValue), nil
			case t.kind == cssTokenDimension && t.text == "px":
				return MaxTrackSizingFunctionFitContentPx(t.value), nil
			}
			return MaxTrackSizingFunction{}, cssUnexpected(t)
		})
	}
	return MaxTrackSizingFunction{}, cssUnexpected(t)
}

func cssMinTrackSizingFunction(p *cssParser) (MinTrackSizingFunction, error) {
	t, err := p.next()
	if err != nil {
		return MinTrackSizingFunction{}, err
	}
	switch {
	case t.kind == cssTokenPercentage:
		return MinTrackSizingFunctionPercent(t.unitValue), nil
	case t.kind == cssTokenDimension && t.text == "px":
		return MinTrackSizingFunctionLength(t.value), nil
	case t.kind == cssTokenIdent:
		switch t.text {
		case "auto":
			return MinTrackSizingFunctionAuto(), nil
		case "min-content":
			return MinTrackSizingFunctionMinContent(), nil
		case "max-content":
			return MinTrackSizingFunctionMaxContent(), nil
		}
	}
	return MinTrackSizingFunction{}, cssUnexpected(t)
}

func cssTrackSizingFunction(p *cssParser) (TrackSizingFunction, error) {
	value, err := cssTryParse(p, func(p *cssParser) (TrackSizingFunction, error) {
		fn, err := p.expectFunctionMatching("minmax")
		if err != nil {
			return TrackSizingFunction{}, err
		}
		return cssParseNestedBlock(fn, func(p *cssParser) (TrackSizingFunction, error) {
			minimum, err := cssMinTrackSizingFunction(p)
			if err != nil {
				return TrackSizingFunction{}, err
			}
			if err := p.expectComma(); err != nil {
				return TrackSizingFunction{}, err
			}
			maximum, err := cssMaxTrackSizingFunction(p)
			if err != nil {
				return TrackSizingFunction{}, err
			}
			return TrackSizingFunction{Min: minimum, Max: maximum}, nil
		})
	})
	if err == nil {
		return value, nil
	}
	maximum, err := cssMaxTrackSizingFunction(p)
	if err != nil {
		return TrackSizingFunction{}, err
	}
	return TrackSizingFunction{Min: MinTrackSizingFunctionFromMaxTrackSizingFunction(maximum), Max: maximum}, nil
}

func cssRepetitionCount(p *cssParser) (RepetitionCount, error) {
	t, err := p.next()
	if err != nil {
		return RepetitionCount{}, err
	}
	switch {
	case t.kind == cssTokenNumber && t.hasInt && t.intValue > 0:
		return RepetitionCountCount(saturatingU16(t.intValue)), nil
	case t.kind == cssTokenIdent && t.text == "auto-fit":
		return RepetitionCountAutoFit, nil
	case t.kind == cssTokenIdent && t.text == "auto-fill":
		return RepetitionCountAutoFill, nil
	}
	return RepetitionCount{}, cssUnexpected(t)
}

func cssGridTemplateComponent(p *cssParser) (GridTemplateComponent[defaultCheapStr], error) {
	value, err := cssTryParse(p, func(p *cssParser) (GridTemplateComponent[defaultCheapStr], error) {
		fn, err := p.expectFunctionMatching("repeat")
		if err != nil {
			return GridTemplateComponent[defaultCheapStr]{}, err
		}
		return cssParseNestedBlock(fn, func(p *cssParser) (GridTemplateComponent[defaultCheapStr], error) {
			count, err := cssRepetitionCount(p)
			if err != nil {
				return GridTemplateComponent[defaultCheapStr]{}, err
			}
			if err := p.expectComma(); err != nil {
				return GridTemplateComponent[defaultCheapStr]{}, err
			}
			tracks, err := cssGridTemplateTracks(p, cssTrackSizingFunction)
			if err != nil {
				return GridTemplateComponent[defaultCheapStr]{}, err
			}
			return GridTemplateComponentRepeat(GridTemplateRepetition[defaultCheapStr]{
				Count: count, Tracks: tracks.Tracks, LineNames: tracks.LineNames,
			}), nil
		})
	})
	if err == nil {
		return value, nil
	}
	trackSizingFunction, err := cssTrackSizingFunction(p)
	if err != nil {
		return GridTemplateComponent[defaultCheapStr]{}, err
	}
	return GridTemplateComponentSingle[defaultCheapStr](trackSizingFunction), nil
}

func cssTryParseLineNames(p *cssParser) ([]defaultCheapStr, error) {
	return cssTryParse(p, func(p *cssParser) ([]defaultCheapStr, error) {
		block, err := p.expectSquareBracketBlock()
		if err != nil {
			return nil, err
		}
		return cssParseNestedBlock(block, func(p *cssParser) ([]defaultCheapStr, error) {
			lineNames := []defaultCheapStr{}
			for !p.isExhausted() {
				ident, err := p.expectIdent()
				if err != nil {
					return nil, err
				}
				lineNames = append(lineNames, ident)
			}
			return lineNames, nil
		})
	})
}

func cssLineNamesOrDefault(p *cssParser) []defaultCheapStr {
	names, err := cssTryParseLineNames(p)
	if err != nil {
		return []defaultCheapStr{}
	}
	return names
}

func cssGridTemplateTracks[Track any](p *cssParser, track func(*cssParser) (Track, error)) (GridTemplateTracks[defaultCheapStr, Track], error) {
	tracks := GridTemplateTracksDefault[defaultCheapStr, Track]()
	tracks.LineNames = append(tracks.LineNames, cssLineNamesOrDefault(p))
	for !p.isExhausted() {
		t, err := track(p)
		if err != nil {
			return GridTemplateTracks[defaultCheapStr, Track]{}, err
		}
		tracks.Tracks = append(tracks.Tracks, t)
		tracks.LineNames = append(tracks.LineNames, cssLineNamesOrDefault(p))
	}
	if len(tracks.Tracks) == 0 {
		return GridTemplateTracks[defaultCheapStr, Track]{}, errCSSEndOfInput
	}
	return tracks, nil
}

func cssGridAutoTracks(p *cssParser) (GridAutoTracks, error) {
	var tracks GridAutoTracks
	for !p.isExhausted() {
		t, err := cssTrackSizingFunction(p)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	if len(tracks) == 0 {
		return nil, errCSSEndOfInput
	}
	return tracks, nil
}
