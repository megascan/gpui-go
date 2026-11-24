//go:build js && wasm

package web

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/segmenter"
)

// CanvasFontFallback controls browser-font fallback when loaded fonts lack a glyph or its emoji
// presentation.
type CanvasFontFallback uint8

const (
	// CanvasFontFallbackDisabled uses only fonts loaded into GPUI.
	CanvasFontFallbackDisabled CanvasFontFallback = iota
	// CanvasFontFallbackEmoji uses Canvas for eligible emoji, including missing text-presentation
	// symbols; it is the default.
	CanvasFontFallbackEmoji
	// CanvasFontFallbackEmojiAndCjk also allows approximate independent rendering of eligible
	// horizontal CJK text.
	CanvasFontFallbackEmojiAndCjk
)

// DefaultCanvasFontFallback returns CanvasFontFallbackEmoji.
func DefaultCanvasFontFallback() CanvasFontFallback {
	return CanvasFontFallbackEmoji
}

func (f CanvasFontFallback) allows(isEmoji bool) bool {
	switch f {
	case CanvasFontFallbackDisabled:
		return false
	case CanvasFontFallbackEmoji:
		return isEmoji
	default:
		return true
	}
}

// CanvasFallback is a supported grapheme assumed to be independently renderable.
type CanvasFallback struct {
	// IsEmoji reports whether this is an emoji-capable grapheme, independent of its presentation.
	IsEmoji bool
	// EmojiPresentation requests emoji presentation instead of ordinary text presentation.
	EmojiPresentation bool
}

// ClassifyCanvasFallback returns a presentation only when grapheme is exactly one supported
// extended grapheme cluster. No normalization, splitting, or font-coverage decision occurs.
func ClassifyCanvasFallback(grapheme string) (CanvasFallback, bool) {
	if isASCII(grapheme) {
		return CanvasFallback{}, false
	}
	first, ok := firstGrapheme(grapheme)
	if !ok || first != len(grapheme) {
		return CanvasFallback{}, false
	}

	if isCjk(grapheme) {
		return CanvasFallback{IsEmoji: false, EmojiPresentation: false}, true
	}
	emojiPresentation, ok := classifyEmoji(grapheme)
	if !ok {
		return CanvasFallback{}, false
	}
	return CanvasFallback{IsEmoji: true, EmojiPresentation: emojiPresentation}, true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func firstGrapheme(s string) (int, bool) {
	for _, g := range graphemeIndices(s) {
		return len(g.text), true
	}
	return 0, false
}

type graphemeSpan struct {
	start int
	text  string
}

func graphemeIndices(s string) []graphemeSpan {
	if s == "" {
		return nil
	}
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		return nil
	}
	var spans []graphemeSpan
	it := seg.GraphemeIterator()
	for it.Next() {
		g := it.Grapheme()
		spans = append(spans, graphemeSpan{start: g.OffsetInBytes, text: s[g.OffsetInBytes : g.OffsetInBytes+g.LengthInBytes]})
	}
	return spans
}

func isCjk(grapheme string) bool {
	base, size := utf8.DecodeRuneInString(grapheme)
	if size == 0 {
		return false
	}
	suffix := grapheme[size:]

	if unicode.Is(unicode.Han, base) && unicode.Is(unicode.Lo, base) &&
		(base >= 0x3400 && base <= 0x9fff || base >= 0xf900 && base <= 0xfaff || base >= 0x20000 && base <= 0x323af) {
		if suffix == "" {
			return true
		}
		selector, n := utf8.DecodeRuneInString(suffix)
		return (selector >= 0xfe00 && selector <= 0xfe02 || selector >= 0xe0100 && selector <= 0xe01ef) && n == len(suffix)
	}

	if base >= 0x3041 && base <= 0x3096 || base >= 0x30a1 && base <= 0x30fa {
		if suffix == "" {
			return true
		}
		switch suffix {
		case "゙":
			return strings.ContainsRune("うかきくけこさしすせそたちつてとはひふへほウカキクケコサシスセソタチツテトハヒフヘホワヰヱヲ", base)
		case "゚":
			return strings.ContainsRune("はひふへほハヒフヘホ", base)
		default:
			return false
		}
	}

	if base >= 0xac00 && base <= 0xd7a3 {
		return suffix == "" || ((base-0xac00)%28 == 0 && isSingleModernTrailingJamo(suffix))
	}
	if base >= 0x1100 && base <= 0x1112 {
		vowel, n := utf8.DecodeRuneInString(suffix)
		if n == 0 || vowel < 0x1161 || vowel > 0x1175 {
			return false
		}
		rest := suffix[n:]
		return rest == "" || isSingleModernTrailingJamo(rest)
	}

	return suffix == "" &&
		(base >= 0x3131 && base <= 0x3163 ||
			strings.ContainsRune("、。〈〉《》「」『』【】〔〕（）［］｛｝，．！？：；・ー々〆", base))
}

func isSingleModernTrailingJamo(text string) bool {
	r, n := utf8.DecodeRuneInString(text)
	return n > 0 && r >= 0x11a8 && r <= 0x11c2 && n == len(text)
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1f1e6 && r <= 0x1f1ff
}

func classifyEmoji(grapheme string) (bool, bool) {
	base, size := utf8.DecodeRuneInString(grapheme)
	if size == 0 {
		return false, false
	}
	suffix := grapheme[size:]

	if base >= '0' && base <= '9' || base == '#' || base == '*' {
		switch suffix {
		case "⃣", "️⃣":
			return true, true
		default:
			return false, false
		}
	}
	if isRegionalIndicator(base) {
		next, n := utf8.DecodeRuneInString(suffix)
		if n > 0 && isRegionalIndicator(next) && n == len(suffix) {
			return true, true
		}
		return false, false
	}
	switch grapheme {
	case "\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f",
		"\U0001f3f4\U000e0067\U000e0062\U000e0073\U000e0063\U000e0074\U000e007f",
		"\U0001f3f4\U000e0067\U000e0062\U000e0077\U000e006c\U000e0073\U000e007f":
		return true, true
	}
	if strings.ContainsRune(grapheme, 0x200d) {
		if isSupportedZwjSequence(grapheme) {
			return true, true
		}
		return false, false
	}
	return classifyEmojiUnit(grapheme)
}

func classifyEmojiUnit(text string) (bool, bool) {
	base, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return false, false
	}
	rest := text[size:]
	status := emojiStatusOf(base)
	if !status.isEmojiChar() || status.isEmojiComponent() {
		return false, false
	}
	emojiPresentation := status == emojiStatusEmojiPresentation || status == emojiStatusEmojiPresentationAndModifierBase
	next, n := utf8.DecodeRuneInString(rest)
	hasNext := n > 0
	rest = rest[n:]
	explicitText := hasNext && next == 0xfe0e
	if hasNext && (next == 0xfe0e || next == 0xfe0f) {
		emojiPresentation = !explicitText
		next, n = utf8.DecodeRuneInString(rest)
		hasNext = n > 0
		rest = rest[n:]
	}
	if hasNext {
		if explicitText ||
			!(status == emojiStatusEmojiModifierBase || status == emojiStatusEmojiPresentationAndModifierBase) ||
			emojiStatusOf(next) != emojiStatusEmojiPresentationAndModifierAndEmojiComponent {
			return false, false
		}
		emojiPresentation = true
	}
	if rest != "" {
		return false, false
	}
	return emojiPresentation, true
}

func isSupportedZwjSequence(grapheme string) bool {
	switch grapheme {
	case "\U0001f468‍\U0001f469‍\U0001f467",
		"\U0001f468‍\U0001f469‍\U0001f467‍\U0001f466",
		"\U0001f469‍\U0001f469‍\U0001f467‍\U0001f466",
		"\U0001f468‍\U0001f468‍\U0001f467‍\U0001f466",
		"\U0001f3f3️‍\U0001f308",
		"\U0001f3f3️‍⚧️",
		"\U0001f3f4‍☠️",
		"❤️‍\U0001f525",
		"❤️‍\U0001fa79",
		"\U0001f441️‍\U0001f5e8️",
		"\U0001f43b‍❄️":
		return true
	}
	person, profession, ok := strings.Cut(grapheme, "‍")
	if !ok {
		return false
	}
	first, _ := utf8.DecodeRuneInString(person)
	if first != 0x1f468 && first != 0x1f469 && first != 0x1f9d1 {
		return false
	}
	if presentation, ok := classifyEmojiUnit(person); !ok || !presentation {
		return false
	}
	switch profession {
	case "⚕️", "⚖️", "✈️", "\U0001f33e", "\U0001f373", "\U0001f393",
		"\U0001f3a4", "\U0001f3a8", "\U0001f3eb", "\U0001f3ed", "\U0001f4bb", "\U0001f4bc",
		"\U0001f527", "\U0001f52c", "\U0001f680", "\U0001f692":
		return true
	}
	return false
}

type emojiStatus uint8

const (
	emojiStatusNonEmoji emojiStatus = iota
	emojiStatusNonEmojiButEmojiComponent
	emojiStatusEmojiPresentation
	emojiStatusEmojiModifierBase
	emojiStatusEmojiPresentationAndModifierBase
	emojiStatusEmojiOther
	emojiStatusEmojiPresentationAndEmojiComponent
	emojiStatusEmojiPresentationAndModifierAndEmojiComponent
	emojiStatusEmojiOtherAndEmojiComponent
)

func (s emojiStatus) isEmojiChar() bool {
	return s != emojiStatusNonEmoji && s != emojiStatusNonEmojiButEmojiComponent
}

func (s emojiStatus) isEmojiComponent() bool {
	return s == emojiStatusEmojiPresentationAndEmojiComponent ||
		s == emojiStatusEmojiPresentationAndModifierAndEmojiComponent ||
		s == emojiStatusEmojiOtherAndEmojiComponent
}

type emojiStatusRange struct {
	lo, hi rune
	status emojiStatus
}

func emojiStatusOf(r rune) emojiStatus {
	i := sort.Search(len(emojiStatusTable), func(i int) bool { return emojiStatusTable[i].hi >= r })
	if i < len(emojiStatusTable) && emojiStatusTable[i].lo <= r {
		return emojiStatusTable[i].status
	}
	return emojiStatusNonEmoji
}

var emojiStatusTable = [...]emojiStatusRange{
	{0x0, 0x22, emojiStatusNonEmoji},
	{0x23, 0x23, emojiStatusEmojiOtherAndEmojiComponent},
	{0x24, 0x29, emojiStatusNonEmoji},
	{0x2a, 0x2a, emojiStatusEmojiOtherAndEmojiComponent},
	{0x2b, 0x2f, emojiStatusNonEmoji},
	{0x30, 0x39, emojiStatusEmojiOtherAndEmojiComponent},
	{0x3a, 0xa8, emojiStatusNonEmoji},
	{0xa9, 0xa9, emojiStatusEmojiOther},
	{0xaa, 0xad, emojiStatusNonEmoji},
	{0xae, 0xae, emojiStatusEmojiOther},
	{0xaf, 0x200c, emojiStatusNonEmoji},
	{0x200d, 0x200d, emojiStatusNonEmojiButEmojiComponent},
	{0x200e, 0x203b, emojiStatusNonEmoji},
	{0x203c, 0x203c, emojiStatusEmojiOther},
	{0x203d, 0x2048, emojiStatusNonEmoji},
	{0x2049, 0x2049, emojiStatusEmojiOther},
	{0x204a, 0x20e2, emojiStatusNonEmoji},
	{0x20e3, 0x20e3, emojiStatusNonEmojiButEmojiComponent},
	{0x20e4, 0x2121, emojiStatusNonEmoji},
	{0x2122, 0x2122, emojiStatusEmojiOther},
	{0x2123, 0x2138, emojiStatusNonEmoji},
	{0x2139, 0x2139, emojiStatusEmojiOther},
	{0x213a, 0x2193, emojiStatusNonEmoji},
	{0x2194, 0x2199, emojiStatusEmojiOther},
	{0x219a, 0x21a8, emojiStatusNonEmoji},
	{0x21a9, 0x21aa, emojiStatusEmojiOther},
	{0x21ab, 0x2319, emojiStatusNonEmoji},
	{0x231a, 0x231b, emojiStatusEmojiPresentation},
	{0x231c, 0x2327, emojiStatusNonEmoji},
	{0x2328, 0x2328, emojiStatusEmojiOther},
	{0x2329, 0x23ce, emojiStatusNonEmoji},
	{0x23cf, 0x23cf, emojiStatusEmojiOther},
	{0x23d0, 0x23e8, emojiStatusNonEmoji},
	{0x23e9, 0x23ec, emojiStatusEmojiPresentation},
	{0x23ed, 0x23ee, emojiStatusEmojiOther},
	{0x23ef, 0x23ef, emojiStatusEmojiOther},
	{0x23f0, 0x23f0, emojiStatusEmojiPresentation},
	{0x23f1, 0x23f2, emojiStatusEmojiOther},
	{0x23f3, 0x23f3, emojiStatusEmojiPresentation},
	{0x23f4, 0x23f7, emojiStatusNonEmoji},
	{0x23f8, 0x23fa, emojiStatusEmojiOther},
	{0x23fb, 0x24c1, emojiStatusNonEmoji},
	{0x24c2, 0x24c2, emojiStatusEmojiOther},
	{0x24c3, 0x25a9, emojiStatusNonEmoji},
	{0x25aa, 0x25ab, emojiStatusEmojiOther},
	{0x25ac, 0x25b5, emojiStatusNonEmoji},
	{0x25b6, 0x25b6, emojiStatusEmojiOther},
	{0x25b7, 0x25bf, emojiStatusNonEmoji},
	{0x25c0, 0x25c0, emojiStatusEmojiOther},
	{0x25c1, 0x25fa, emojiStatusNonEmoji},
	{0x25fb, 0x25fc, emojiStatusEmojiOther},
	{0x25fd, 0x25fe, emojiStatusEmojiPresentation},
	{0x25ff, 0x25ff, emojiStatusNonEmoji},
	{0x2600, 0x2601, emojiStatusEmojiOther},
	{0x2602, 0x2603, emojiStatusEmojiOther},
	{0x2604, 0x2604, emojiStatusEmojiOther},
	{0x2605, 0x260d, emojiStatusNonEmoji},
	{0x260e, 0x260e, emojiStatusEmojiOther},
	{0x260f, 0x2610, emojiStatusNonEmoji},
	{0x2611, 0x2611, emojiStatusEmojiOther},
	{0x2612, 0x2613, emojiStatusNonEmoji},
	{0x2614, 0x2615, emojiStatusEmojiPresentation},
	{0x2616, 0x2617, emojiStatusNonEmoji},
	{0x2618, 0x2618, emojiStatusEmojiOther},
	{0x2619, 0x261c, emojiStatusNonEmoji},
	{0x261d, 0x261d, emojiStatusEmojiModifierBase},
	{0x261e, 0x261f, emojiStatusNonEmoji},
	{0x2620, 0x2620, emojiStatusEmojiOther},
	{0x2621, 0x2621, emojiStatusNonEmoji},
	{0x2622, 0x2623, emojiStatusEmojiOther},
	{0x2624, 0x2625, emojiStatusNonEmoji},
	{0x2626, 0x2626, emojiStatusEmojiOther},
	{0x2627, 0x2629, emojiStatusNonEmoji},
	{0x262a, 0x262a, emojiStatusEmojiOther},
	{0x262b, 0x262d, emojiStatusNonEmoji},
	{0x262e, 0x262e, emojiStatusEmojiOther},
	{0x262f, 0x262f, emojiStatusEmojiOther},
	{0x2630, 0x2637, emojiStatusNonEmoji},
	{0x2638, 0x2639, emojiStatusEmojiOther},
	{0x263a, 0x263a, emojiStatusEmojiOther},
	{0x263b, 0x263f, emojiStatusNonEmoji},
	{0x2640, 0x2640, emojiStatusEmojiOther},
	{0x2641, 0x2641, emojiStatusNonEmoji},
	{0x2642, 0x2642, emojiStatusEmojiOther},
	{0x2643, 0x2647, emojiStatusNonEmoji},
	{0x2648, 0x2653, emojiStatusEmojiPresentation},
	{0x2654, 0x265e, emojiStatusNonEmoji},
	{0x265f, 0x265f, emojiStatusEmojiOther},
	{0x2660, 0x2660, emojiStatusEmojiOther},
	{0x2661, 0x2662, emojiStatusNonEmoji},
	{0x2663, 0x2663, emojiStatusEmojiOther},
	{0x2664, 0x2664, emojiStatusNonEmoji},
	{0x2665, 0x2666, emojiStatusEmojiOther},
	{0x2667, 0x2667, emojiStatusNonEmoji},
	{0x2668, 0x2668, emojiStatusEmojiOther},
	{0x2669, 0x267a, emojiStatusNonEmoji},
	{0x267b, 0x267b, emojiStatusEmojiOther},
	{0x267c, 0x267d, emojiStatusNonEmoji},
	{0x267e, 0x267e, emojiStatusEmojiOther},
	{0x267f, 0x267f, emojiStatusEmojiPresentation},
	{0x2680, 0x2691, emojiStatusNonEmoji},
	{0x2692, 0x2692, emojiStatusEmojiOther},
	{0x2693, 0x2693, emojiStatusEmojiPresentation},
	{0x2694, 0x2694, emojiStatusEmojiOther},
	{0x2695, 0x2695, emojiStatusEmojiOther},
	{0x2696, 0x2697, emojiStatusEmojiOther},
	{0x2698, 0x2698, emojiStatusNonEmoji},
	{0x2699, 0x2699, emojiStatusEmojiOther},
	{0x269a, 0x269a, emojiStatusNonEmoji},
	{0x269b, 0x269c, emojiStatusEmojiOther},
	{0x269d, 0x269f, emojiStatusNonEmoji},
	{0x26a0, 0x26a0, emojiStatusEmojiOther},
	{0x26a1, 0x26a1, emojiStatusEmojiPresentation},
	{0x26a2, 0x26a6, emojiStatusNonEmoji},
	{0x26a7, 0x26a7, emojiStatusEmojiOther},
	{0x26a8, 0x26a9, emojiStatusNonEmoji},
	{0x26aa, 0x26ab, emojiStatusEmojiPresentation},
	{0x26ac, 0x26af, emojiStatusNonEmoji},
	{0x26b0, 0x26b1, emojiStatusEmojiOther},
	{0x26b2, 0x26bc, emojiStatusNonEmoji},
	{0x26bd, 0x26be, emojiStatusEmojiPresentation},
	{0x26bf, 0x26c3, emojiStatusNonEmoji},
	{0x26c4, 0x26c5, emojiStatusEmojiPresentation},
	{0x26c6, 0x26c7, emojiStatusNonEmoji},
	{0x26c8, 0x26c8, emojiStatusEmojiOther},
	{0x26c9, 0x26cd, emojiStatusNonEmoji},
	{0x26ce, 0x26ce, emojiStatusEmojiPresentation},
	{0x26cf, 0x26cf, emojiStatusEmojiOther},
	{0x26d0, 0x26d0, emojiStatusNonEmoji},
	{0x26d1, 0x26d1, emojiStatusEmojiOther},
	{0x26d2, 0x26d2, emojiStatusNonEmoji},
	{0x26d3, 0x26d3, emojiStatusEmojiOther},
	{0x26d4, 0x26d4, emojiStatusEmojiPresentation},
	{0x26d5, 0x26e8, emojiStatusNonEmoji},
	{0x26e9, 0x26e9, emojiStatusEmojiOther},
	{0x26ea, 0x26ea, emojiStatusEmojiPresentation},
	{0x26eb, 0x26ef, emojiStatusNonEmoji},
	{0x26f0, 0x26f1, emojiStatusEmojiOther},
	{0x26f2, 0x26f3, emojiStatusEmojiPresentation},
	{0x26f4, 0x26f4, emojiStatusEmojiOther},
	{0x26f5, 0x26f5, emojiStatusEmojiPresentation},
	{0x26f6, 0x26f6, emojiStatusNonEmoji},
	{0x26f7, 0x26f8, emojiStatusEmojiOther},
	{0x26f9, 0x26f9, emojiStatusEmojiModifierBase},
	{0x26fa, 0x26fa, emojiStatusEmojiPresentation},
	{0x26fb, 0x26fc, emojiStatusNonEmoji},
	{0x26fd, 0x26fd, emojiStatusEmojiPresentation},
	{0x26fe, 0x2701, emojiStatusNonEmoji},
	{0x2702, 0x2702, emojiStatusEmojiOther},
	{0x2703, 0x2704, emojiStatusNonEmoji},
	{0x2705, 0x2705, emojiStatusEmojiPresentation},
	{0x2706, 0x2707, emojiStatusNonEmoji},
	{0x2708, 0x2709, emojiStatusEmojiOther},
	{0x270a, 0x270b, emojiStatusEmojiPresentationAndModifierBase},
	{0x270c, 0x270c, emojiStatusEmojiModifierBase},
	{0x270d, 0x270d, emojiStatusEmojiModifierBase},
	{0x270e, 0x270e, emojiStatusNonEmoji},
	{0x270f, 0x270f, emojiStatusEmojiOther},
	{0x2710, 0x2711, emojiStatusNonEmoji},
	{0x2712, 0x2712, emojiStatusEmojiOther},
	{0x2713, 0x2713, emojiStatusNonEmoji},
	{0x2714, 0x2714, emojiStatusEmojiOther},
	{0x2715, 0x2715, emojiStatusNonEmoji},
	{0x2716, 0x2716, emojiStatusEmojiOther},
	{0x2717, 0x271c, emojiStatusNonEmoji},
	{0x271d, 0x271d, emojiStatusEmojiOther},
	{0x271e, 0x2720, emojiStatusNonEmoji},
	{0x2721, 0x2721, emojiStatusEmojiOther},
	{0x2722, 0x2727, emojiStatusNonEmoji},
	{0x2728, 0x2728, emojiStatusEmojiPresentation},
	{0x2729, 0x2732, emojiStatusNonEmoji},
	{0x2733, 0x2734, emojiStatusEmojiOther},
	{0x2735, 0x2743, emojiStatusNonEmoji},
	{0x2744, 0x2744, emojiStatusEmojiOther},
	{0x2745, 0x2746, emojiStatusNonEmoji},
	{0x2747, 0x2747, emojiStatusEmojiOther},
	{0x2748, 0x274b, emojiStatusNonEmoji},
	{0x274c, 0x274c, emojiStatusEmojiPresentation},
	{0x274d, 0x274d, emojiStatusNonEmoji},
	{0x274e, 0x274e, emojiStatusEmojiPresentation},
	{0x274f, 0x2752, emojiStatusNonEmoji},
	{0x2753, 0x2755, emojiStatusEmojiPresentation},
	{0x2756, 0x2756, emojiStatusNonEmoji},
	{0x2757, 0x2757, emojiStatusEmojiPresentation},
	{0x2758, 0x2762, emojiStatusNonEmoji},
	{0x2763, 0x2763, emojiStatusEmojiOther},
	{0x2764, 0x2764, emojiStatusEmojiOther},
	{0x2765, 0x2794, emojiStatusNonEmoji},
	{0x2795, 0x2797, emojiStatusEmojiPresentation},
	{0x2798, 0x27a0, emojiStatusNonEmoji},
	{0x27a1, 0x27a1, emojiStatusEmojiOther},
	{0x27a2, 0x27af, emojiStatusNonEmoji},
	{0x27b0, 0x27b0, emojiStatusEmojiPresentation},
	{0x27b1, 0x27be, emojiStatusNonEmoji},
	{0x27bf, 0x27bf, emojiStatusEmojiPresentation},
	{0x27c0, 0x2933, emojiStatusNonEmoji},
	{0x2934, 0x2935, emojiStatusEmojiOther},
	{0x2936, 0x2b04, emojiStatusNonEmoji},
	{0x2b05, 0x2b07, emojiStatusEmojiOther},
	{0x2b08, 0x2b1a, emojiStatusNonEmoji},
	{0x2b1b, 0x2b1c, emojiStatusEmojiPresentation},
	{0x2b1d, 0x2b4f, emojiStatusNonEmoji},
	{0x2b50, 0x2b50, emojiStatusEmojiPresentation},
	{0x2b51, 0x2b54, emojiStatusNonEmoji},
	{0x2b55, 0x2b55, emojiStatusEmojiPresentation},
	{0x2b56, 0x302f, emojiStatusNonEmoji},
	{0x3030, 0x3030, emojiStatusEmojiOther},
	{0x3031, 0x303c, emojiStatusNonEmoji},
	{0x303d, 0x303d, emojiStatusEmojiOther},
	{0x303e, 0x3296, emojiStatusNonEmoji},
	{0x3297, 0x3297, emojiStatusEmojiOther},
	{0x3298, 0x3298, emojiStatusNonEmoji},
	{0x3299, 0x3299, emojiStatusEmojiOther},
	{0x329a, 0xd7ff, emojiStatusNonEmoji},
	{0xe000, 0xfe0e, emojiStatusNonEmoji},
	{0xfe0f, 0xfe0f, emojiStatusNonEmojiButEmojiComponent},
	{0xfe10, 0x1f003, emojiStatusNonEmoji},
	{0x1f004, 0x1f004, emojiStatusEmojiPresentation},
	{0x1f005, 0x1f0ce, emojiStatusNonEmoji},
	{0x1f0cf, 0x1f0cf, emojiStatusEmojiPresentation},
	{0x1f0d0, 0x1f16f, emojiStatusNonEmoji},
	{0x1f170, 0x1f171, emojiStatusEmojiOther},
	{0x1f172, 0x1f17d, emojiStatusNonEmoji},
	{0x1f17e, 0x1f17f, emojiStatusEmojiOther},
	{0x1f180, 0x1f18d, emojiStatusNonEmoji},
	{0x1f18e, 0x1f18e, emojiStatusEmojiPresentation},
	{0x1f18f, 0x1f190, emojiStatusNonEmoji},
	{0x1f191, 0x1f19a, emojiStatusEmojiPresentation},
	{0x1f19b, 0x1f1e5, emojiStatusNonEmoji},
	{0x1f1e6, 0x1f1ff, emojiStatusEmojiPresentationAndEmojiComponent},
	{0x1f200, 0x1f200, emojiStatusNonEmoji},
	{0x1f201, 0x1f201, emojiStatusEmojiPresentation},
	{0x1f202, 0x1f202, emojiStatusEmojiOther},
	{0x1f203, 0x1f219, emojiStatusNonEmoji},
	{0x1f21a, 0x1f21a, emojiStatusEmojiPresentation},
	{0x1f21b, 0x1f22e, emojiStatusNonEmoji},
	{0x1f22f, 0x1f22f, emojiStatusEmojiPresentation},
	{0x1f230, 0x1f231, emojiStatusNonEmoji},
	{0x1f232, 0x1f236, emojiStatusEmojiPresentation},
	{0x1f237, 0x1f237, emojiStatusEmojiOther},
	{0x1f238, 0x1f23a, emojiStatusEmojiPresentation},
	{0x1f23b, 0x1f24f, emojiStatusNonEmoji},
	{0x1f250, 0x1f251, emojiStatusEmojiPresentation},
	{0x1f252, 0x1f2ff, emojiStatusNonEmoji},
	{0x1f300, 0x1f30c, emojiStatusEmojiPresentation},
	{0x1f30d, 0x1f30e, emojiStatusEmojiPresentation},
	{0x1f30f, 0x1f30f, emojiStatusEmojiPresentation},
	{0x1f310, 0x1f310, emojiStatusEmojiPresentation},
	{0x1f311, 0x1f311, emojiStatusEmojiPresentation},
	{0x1f312, 0x1f312, emojiStatusEmojiPresentation},
	{0x1f313, 0x1f315, emojiStatusEmojiPresentation},
	{0x1f316, 0x1f318, emojiStatusEmojiPresentation},
	{0x1f319, 0x1f319, emojiStatusEmojiPresentation},
	{0x1f31a, 0x1f31a, emojiStatusEmojiPresentation},
	{0x1f31b, 0x1f31b, emojiStatusEmojiPresentation},
	{0x1f31c, 0x1f31c, emojiStatusEmojiPresentation},
	{0x1f31d, 0x1f31e, emojiStatusEmojiPresentation},
	{0x1f31f, 0x1f320, emojiStatusEmojiPresentation},
	{0x1f321, 0x1f321, emojiStatusEmojiOther},
	{0x1f322, 0x1f323, emojiStatusNonEmoji},
	{0x1f324, 0x1f32c, emojiStatusEmojiOther},
	{0x1f32d, 0x1f32f, emojiStatusEmojiPresentation},
	{0x1f330, 0x1f331, emojiStatusEmojiPresentation},
	{0x1f332, 0x1f333, emojiStatusEmojiPresentation},
	{0x1f334, 0x1f335, emojiStatusEmojiPresentation},
	{0x1f336, 0x1f336, emojiStatusEmojiOther},
	{0x1f337, 0x1f34a, emojiStatusEmojiPresentation},
	{0x1f34b, 0x1f34b, emojiStatusEmojiPresentation},
	{0x1f34c, 0x1f34f, emojiStatusEmojiPresentation},
	{0x1f350, 0x1f350, emojiStatusEmojiPresentation},
	{0x1f351, 0x1f37b, emojiStatusEmojiPresentation},
	{0x1f37c, 0x1f37c, emojiStatusEmojiPresentation},
	{0x1f37d, 0x1f37d, emojiStatusEmojiOther},
	{0x1f37e, 0x1f37f, emojiStatusEmojiPresentation},
	{0x1f380, 0x1f384, emojiStatusEmojiPresentation},
	{0x1f385, 0x1f385, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f386, 0x1f393, emojiStatusEmojiPresentation},
	{0x1f394, 0x1f395, emojiStatusNonEmoji},
	{0x1f396, 0x1f397, emojiStatusEmojiOther},
	{0x1f398, 0x1f398, emojiStatusNonEmoji},
	{0x1f399, 0x1f39b, emojiStatusEmojiOther},
	{0x1f39c, 0x1f39d, emojiStatusNonEmoji},
	{0x1f39e, 0x1f39f, emojiStatusEmojiOther},
	{0x1f3a0, 0x1f3c1, emojiStatusEmojiPresentation},
	{0x1f3c2, 0x1f3c4, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f3c5, 0x1f3c5, emojiStatusEmojiPresentation},
	{0x1f3c6, 0x1f3c6, emojiStatusEmojiPresentation},
	{0x1f3c7, 0x1f3c7, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f3c8, 0x1f3c8, emojiStatusEmojiPresentation},
	{0x1f3c9, 0x1f3c9, emojiStatusEmojiPresentation},
	{0x1f3ca, 0x1f3ca, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f3cb, 0x1f3cc, emojiStatusEmojiModifierBase},
	{0x1f3cd, 0x1f3ce, emojiStatusEmojiOther},
	{0x1f3cf, 0x1f3d3, emojiStatusEmojiPresentation},
	{0x1f3d4, 0x1f3df, emojiStatusEmojiOther},
	{0x1f3e0, 0x1f3e3, emojiStatusEmojiPresentation},
	{0x1f3e4, 0x1f3e4, emojiStatusEmojiPresentation},
	{0x1f3e5, 0x1f3f0, emojiStatusEmojiPresentation},
	{0x1f3f1, 0x1f3f2, emojiStatusNonEmoji},
	{0x1f3f3, 0x1f3f3, emojiStatusEmojiOther},
	{0x1f3f4, 0x1f3f4, emojiStatusEmojiPresentation},
	{0x1f3f5, 0x1f3f5, emojiStatusEmojiOther},
	{0x1f3f6, 0x1f3f6, emojiStatusNonEmoji},
	{0x1f3f7, 0x1f3f7, emojiStatusEmojiOther},
	{0x1f3f8, 0x1f3fa, emojiStatusEmojiPresentation},
	{0x1f3fb, 0x1f3ff, emojiStatusEmojiPresentationAndModifierAndEmojiComponent},
	{0x1f400, 0x1f407, emojiStatusEmojiPresentation},
	{0x1f408, 0x1f408, emojiStatusEmojiPresentation},
	{0x1f409, 0x1f40b, emojiStatusEmojiPresentation},
	{0x1f40c, 0x1f40e, emojiStatusEmojiPresentation},
	{0x1f40f, 0x1f410, emojiStatusEmojiPresentation},
	{0x1f411, 0x1f412, emojiStatusEmojiPresentation},
	{0x1f413, 0x1f413, emojiStatusEmojiPresentation},
	{0x1f414, 0x1f414, emojiStatusEmojiPresentation},
	{0x1f415, 0x1f415, emojiStatusEmojiPresentation},
	{0x1f416, 0x1f416, emojiStatusEmojiPresentation},
	{0x1f417, 0x1f429, emojiStatusEmojiPresentation},
	{0x1f42a, 0x1f42a, emojiStatusEmojiPresentation},
	{0x1f42b, 0x1f43e, emojiStatusEmojiPresentation},
	{0x1f43f, 0x1f43f, emojiStatusEmojiOther},
	{0x1f440, 0x1f440, emojiStatusEmojiPresentation},
	{0x1f441, 0x1f441, emojiStatusEmojiOther},
	{0x1f442, 0x1f443, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f444, 0x1f445, emojiStatusEmojiPresentation},
	{0x1f446, 0x1f450, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f451, 0x1f464, emojiStatusEmojiPresentation},
	{0x1f465, 0x1f465, emojiStatusEmojiPresentation},
	{0x1f466, 0x1f46b, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f46c, 0x1f46d, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f46e, 0x1f478, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f479, 0x1f47b, emojiStatusEmojiPresentation},
	{0x1f47c, 0x1f47c, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f47d, 0x1f480, emojiStatusEmojiPresentation},
	{0x1f481, 0x1f483, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f484, 0x1f484, emojiStatusEmojiPresentation},
	{0x1f485, 0x1f487, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f488, 0x1f48e, emojiStatusEmojiPresentation},
	{0x1f48f, 0x1f48f, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f490, 0x1f490, emojiStatusEmojiPresentation},
	{0x1f491, 0x1f491, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f492, 0x1f4a9, emojiStatusEmojiPresentation},
	{0x1f4aa, 0x1f4aa, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f4ab, 0x1f4ac, emojiStatusEmojiPresentation},
	{0x1f4ad, 0x1f4ad, emojiStatusEmojiPresentation},
	{0x1f4ae, 0x1f4b5, emojiStatusEmojiPresentation},
	{0x1f4b6, 0x1f4b7, emojiStatusEmojiPresentation},
	{0x1f4b8, 0x1f4eb, emojiStatusEmojiPresentation},
	{0x1f4ec, 0x1f4ed, emojiStatusEmojiPresentation},
	{0x1f4ee, 0x1f4ee, emojiStatusEmojiPresentation},
	{0x1f4ef, 0x1f4ef, emojiStatusEmojiPresentation},
	{0x1f4f0, 0x1f4f4, emojiStatusEmojiPresentation},
	{0x1f4f5, 0x1f4f5, emojiStatusEmojiPresentation},
	{0x1f4f6, 0x1f4f7, emojiStatusEmojiPresentation},
	{0x1f4f8, 0x1f4f8, emojiStatusEmojiPresentation},
	{0x1f4f9, 0x1f4fc, emojiStatusEmojiPresentation},
	{0x1f4fd, 0x1f4fd, emojiStatusEmojiOther},
	{0x1f4fe, 0x1f4fe, emojiStatusNonEmoji},
	{0x1f4ff, 0x1f502, emojiStatusEmojiPresentation},
	{0x1f503, 0x1f503, emojiStatusEmojiPresentation},
	{0x1f504, 0x1f507, emojiStatusEmojiPresentation},
	{0x1f508, 0x1f508, emojiStatusEmojiPresentation},
	{0x1f509, 0x1f509, emojiStatusEmojiPresentation},
	{0x1f50a, 0x1f514, emojiStatusEmojiPresentation},
	{0x1f515, 0x1f515, emojiStatusEmojiPresentation},
	{0x1f516, 0x1f52b, emojiStatusEmojiPresentation},
	{0x1f52c, 0x1f52d, emojiStatusEmojiPresentation},
	{0x1f52e, 0x1f53d, emojiStatusEmojiPresentation},
	{0x1f53e, 0x1f548, emojiStatusNonEmoji},
	{0x1f549, 0x1f54a, emojiStatusEmojiOther},
	{0x1f54b, 0x1f54e, emojiStatusEmojiPresentation},
	{0x1f54f, 0x1f54f, emojiStatusNonEmoji},
	{0x1f550, 0x1f55b, emojiStatusEmojiPresentation},
	{0x1f55c, 0x1f567, emojiStatusEmojiPresentation},
	{0x1f568, 0x1f56e, emojiStatusNonEmoji},
	{0x1f56f, 0x1f570, emojiStatusEmojiOther},
	{0x1f571, 0x1f572, emojiStatusNonEmoji},
	{0x1f573, 0x1f573, emojiStatusEmojiOther},
	{0x1f574, 0x1f575, emojiStatusEmojiModifierBase},
	{0x1f576, 0x1f579, emojiStatusEmojiOther},
	{0x1f57a, 0x1f57a, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f57b, 0x1f586, emojiStatusNonEmoji},
	{0x1f587, 0x1f587, emojiStatusEmojiOther},
	{0x1f588, 0x1f589, emojiStatusNonEmoji},
	{0x1f58a, 0x1f58d, emojiStatusEmojiOther},
	{0x1f58e, 0x1f58f, emojiStatusNonEmoji},
	{0x1f590, 0x1f590, emojiStatusEmojiModifierBase},
	{0x1f591, 0x1f594, emojiStatusNonEmoji},
	{0x1f595, 0x1f596, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f597, 0x1f5a3, emojiStatusNonEmoji},
	{0x1f5a4, 0x1f5a4, emojiStatusEmojiPresentation},
	{0x1f5a5, 0x1f5a5, emojiStatusEmojiOther},
	{0x1f5a6, 0x1f5a7, emojiStatusNonEmoji},
	{0x1f5a8, 0x1f5a8, emojiStatusEmojiOther},
	{0x1f5a9, 0x1f5b0, emojiStatusNonEmoji},
	{0x1f5b1, 0x1f5b2, emojiStatusEmojiOther},
	{0x1f5b3, 0x1f5bb, emojiStatusNonEmoji},
	{0x1f5bc, 0x1f5bc, emojiStatusEmojiOther},
	{0x1f5bd, 0x1f5c1, emojiStatusNonEmoji},
	{0x1f5c2, 0x1f5c4, emojiStatusEmojiOther},
	{0x1f5c5, 0x1f5d0, emojiStatusNonEmoji},
	{0x1f5d1, 0x1f5d3, emojiStatusEmojiOther},
	{0x1f5d4, 0x1f5db, emojiStatusNonEmoji},
	{0x1f5dc, 0x1f5de, emojiStatusEmojiOther},
	{0x1f5df, 0x1f5e0, emojiStatusNonEmoji},
	{0x1f5e1, 0x1f5e1, emojiStatusEmojiOther},
	{0x1f5e2, 0x1f5e2, emojiStatusNonEmoji},
	{0x1f5e3, 0x1f5e3, emojiStatusEmojiOther},
	{0x1f5e4, 0x1f5e7, emojiStatusNonEmoji},
	{0x1f5e8, 0x1f5e8, emojiStatusEmojiOther},
	{0x1f5e9, 0x1f5ee, emojiStatusNonEmoji},
	{0x1f5ef, 0x1f5ef, emojiStatusEmojiOther},
	{0x1f5f0, 0x1f5f2, emojiStatusNonEmoji},
	{0x1f5f3, 0x1f5f3, emojiStatusEmojiOther},
	{0x1f5f4, 0x1f5f9, emojiStatusNonEmoji},
	{0x1f5fa, 0x1f5fa, emojiStatusEmojiOther},
	{0x1f5fb, 0x1f5ff, emojiStatusEmojiPresentation},
	{0x1f600, 0x1f600, emojiStatusEmojiPresentation},
	{0x1f601, 0x1f606, emojiStatusEmojiPresentation},
	{0x1f607, 0x1f608, emojiStatusEmojiPresentation},
	{0x1f609, 0x1f60d, emojiStatusEmojiPresentation},
	{0x1f60e, 0x1f60e, emojiStatusEmojiPresentation},
	{0x1f60f, 0x1f60f, emojiStatusEmojiPresentation},
	{0x1f610, 0x1f610, emojiStatusEmojiPresentation},
	{0x1f611, 0x1f611, emojiStatusEmojiPresentation},
	{0x1f612, 0x1f614, emojiStatusEmojiPresentation},
	{0x1f615, 0x1f615, emojiStatusEmojiPresentation},
	{0x1f616, 0x1f616, emojiStatusEmojiPresentation},
	{0x1f617, 0x1f617, emojiStatusEmojiPresentation},
	{0x1f618, 0x1f618, emojiStatusEmojiPresentation},
	{0x1f619, 0x1f619, emojiStatusEmojiPresentation},
	{0x1f61a, 0x1f61a, emojiStatusEmojiPresentation},
	{0x1f61b, 0x1f61b, emojiStatusEmojiPresentation},
	{0x1f61c, 0x1f61e, emojiStatusEmojiPresentation},
	{0x1f61f, 0x1f61f, emojiStatusEmojiPresentation},
	{0x1f620, 0x1f625, emojiStatusEmojiPresentation},
	{0x1f626, 0x1f627, emojiStatusEmojiPresentation},
	{0x1f628, 0x1f62b, emojiStatusEmojiPresentation},
	{0x1f62c, 0x1f62c, emojiStatusEmojiPresentation},
	{0x1f62d, 0x1f62d, emojiStatusEmojiPresentation},
	{0x1f62e, 0x1f62f, emojiStatusEmojiPresentation},
	{0x1f630, 0x1f633, emojiStatusEmojiPresentation},
	{0x1f634, 0x1f634, emojiStatusEmojiPresentation},
	{0x1f635, 0x1f635, emojiStatusEmojiPresentation},
	{0x1f636, 0x1f636, emojiStatusEmojiPresentation},
	{0x1f637, 0x1f640, emojiStatusEmojiPresentation},
	{0x1f641, 0x1f644, emojiStatusEmojiPresentation},
	{0x1f645, 0x1f647, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f648, 0x1f64a, emojiStatusEmojiPresentation},
	{0x1f64b, 0x1f64f, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f650, 0x1f67f, emojiStatusNonEmoji},
	{0x1f680, 0x1f680, emojiStatusEmojiPresentation},
	{0x1f681, 0x1f682, emojiStatusEmojiPresentation},
	{0x1f683, 0x1f685, emojiStatusEmojiPresentation},
	{0x1f686, 0x1f686, emojiStatusEmojiPresentation},
	{0x1f687, 0x1f687, emojiStatusEmojiPresentation},
	{0x1f688, 0x1f688, emojiStatusEmojiPresentation},
	{0x1f689, 0x1f689, emojiStatusEmojiPresentation},
	{0x1f68a, 0x1f68b, emojiStatusEmojiPresentation},
	{0x1f68c, 0x1f68c, emojiStatusEmojiPresentation},
	{0x1f68d, 0x1f68d, emojiStatusEmojiPresentation},
	{0x1f68e, 0x1f68e, emojiStatusEmojiPresentation},
	{0x1f68f, 0x1f68f, emojiStatusEmojiPresentation},
	{0x1f690, 0x1f690, emojiStatusEmojiPresentation},
	{0x1f691, 0x1f693, emojiStatusEmojiPresentation},
	{0x1f694, 0x1f694, emojiStatusEmojiPresentation},
	{0x1f695, 0x1f695, emojiStatusEmojiPresentation},
	{0x1f696, 0x1f696, emojiStatusEmojiPresentation},
	{0x1f697, 0x1f697, emojiStatusEmojiPresentation},
	{0x1f698, 0x1f698, emojiStatusEmojiPresentation},
	{0x1f699, 0x1f69a, emojiStatusEmojiPresentation},
	{0x1f69b, 0x1f6a1, emojiStatusEmojiPresentation},
	{0x1f6a2, 0x1f6a2, emojiStatusEmojiPresentation},
	{0x1f6a3, 0x1f6a3, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f6a4, 0x1f6a5, emojiStatusEmojiPresentation},
	{0x1f6a6, 0x1f6a6, emojiStatusEmojiPresentation},
	{0x1f6a7, 0x1f6ad, emojiStatusEmojiPresentation},
	{0x1f6ae, 0x1f6b1, emojiStatusEmojiPresentation},
	{0x1f6b2, 0x1f6b2, emojiStatusEmojiPresentation},
	{0x1f6b3, 0x1f6b3, emojiStatusEmojiPresentation},
	{0x1f6b4, 0x1f6b5, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f6b6, 0x1f6b6, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f6b7, 0x1f6b8, emojiStatusEmojiPresentation},
	{0x1f6b9, 0x1f6be, emojiStatusEmojiPresentation},
	{0x1f6bf, 0x1f6bf, emojiStatusEmojiPresentation},
	{0x1f6c0, 0x1f6c0, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f6c1, 0x1f6c5, emojiStatusEmojiPresentation},
	{0x1f6c6, 0x1f6ca, emojiStatusNonEmoji},
	{0x1f6cb, 0x1f6cb, emojiStatusEmojiOther},
	{0x1f6cc, 0x1f6cc, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f6cd, 0x1f6cf, emojiStatusEmojiOther},
	{0x1f6d0, 0x1f6d0, emojiStatusEmojiPresentation},
	{0x1f6d1, 0x1f6d2, emojiStatusEmojiPresentation},
	{0x1f6d3, 0x1f6d4, emojiStatusNonEmoji},
	{0x1f6d5, 0x1f6d5, emojiStatusEmojiPresentation},
	{0x1f6d6, 0x1f6d7, emojiStatusEmojiPresentation},
	{0x1f6d8, 0x1f6db, emojiStatusNonEmoji},
	{0x1f6dc, 0x1f6dc, emojiStatusEmojiPresentation},
	{0x1f6dd, 0x1f6df, emojiStatusEmojiPresentation},
	{0x1f6e0, 0x1f6e5, emojiStatusEmojiOther},
	{0x1f6e6, 0x1f6e8, emojiStatusNonEmoji},
	{0x1f6e9, 0x1f6e9, emojiStatusEmojiOther},
	{0x1f6ea, 0x1f6ea, emojiStatusNonEmoji},
	{0x1f6eb, 0x1f6ec, emojiStatusEmojiPresentation},
	{0x1f6ed, 0x1f6ef, emojiStatusNonEmoji},
	{0x1f6f0, 0x1f6f0, emojiStatusEmojiOther},
	{0x1f6f1, 0x1f6f2, emojiStatusNonEmoji},
	{0x1f6f3, 0x1f6f3, emojiStatusEmojiOther},
	{0x1f6f4, 0x1f6f6, emojiStatusEmojiPresentation},
	{0x1f6f7, 0x1f6f8, emojiStatusEmojiPresentation},
	{0x1f6f9, 0x1f6f9, emojiStatusEmojiPresentation},
	{0x1f6fa, 0x1f6fa, emojiStatusEmojiPresentation},
	{0x1f6fb, 0x1f6fc, emojiStatusEmojiPresentation},
	{0x1f6fd, 0x1f7df, emojiStatusNonEmoji},
	{0x1f7e0, 0x1f7eb, emojiStatusEmojiPresentation},
	{0x1f7ec, 0x1f7ef, emojiStatusNonEmoji},
	{0x1f7f0, 0x1f7f0, emojiStatusEmojiPresentation},
	{0x1f7f1, 0x1f90b, emojiStatusNonEmoji},
	{0x1f90c, 0x1f90c, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f90d, 0x1f90e, emojiStatusEmojiPresentation},
	{0x1f90f, 0x1f90f, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f910, 0x1f917, emojiStatusEmojiPresentation},
	{0x1f918, 0x1f918, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f919, 0x1f91e, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f91f, 0x1f91f, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f920, 0x1f925, emojiStatusEmojiPresentation},
	{0x1f926, 0x1f926, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f927, 0x1f927, emojiStatusEmojiPresentation},
	{0x1f928, 0x1f92f, emojiStatusEmojiPresentation},
	{0x1f930, 0x1f930, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f931, 0x1f932, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f933, 0x1f939, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f93a, 0x1f93a, emojiStatusEmojiPresentation},
	{0x1f93b, 0x1f93b, emojiStatusNonEmoji},
	{0x1f93c, 0x1f93e, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f93f, 0x1f93f, emojiStatusEmojiPresentation},
	{0x1f940, 0x1f945, emojiStatusEmojiPresentation},
	{0x1f946, 0x1f946, emojiStatusNonEmoji},
	{0x1f947, 0x1f94b, emojiStatusEmojiPresentation},
	{0x1f94c, 0x1f94c, emojiStatusEmojiPresentation},
	{0x1f94d, 0x1f94f, emojiStatusEmojiPresentation},
	{0x1f950, 0x1f95e, emojiStatusEmojiPresentation},
	{0x1f95f, 0x1f96b, emojiStatusEmojiPresentation},
	{0x1f96c, 0x1f970, emojiStatusEmojiPresentation},
	{0x1f971, 0x1f971, emojiStatusEmojiPresentation},
	{0x1f972, 0x1f972, emojiStatusEmojiPresentation},
	{0x1f973, 0x1f976, emojiStatusEmojiPresentation},
	{0x1f977, 0x1f977, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f978, 0x1f978, emojiStatusEmojiPresentation},
	{0x1f979, 0x1f979, emojiStatusEmojiPresentation},
	{0x1f97a, 0x1f97a, emojiStatusEmojiPresentation},
	{0x1f97b, 0x1f97b, emojiStatusEmojiPresentation},
	{0x1f97c, 0x1f97f, emojiStatusEmojiPresentation},
	{0x1f980, 0x1f984, emojiStatusEmojiPresentation},
	{0x1f985, 0x1f991, emojiStatusEmojiPresentation},
	{0x1f992, 0x1f997, emojiStatusEmojiPresentation},
	{0x1f998, 0x1f9a2, emojiStatusEmojiPresentation},
	{0x1f9a3, 0x1f9a4, emojiStatusEmojiPresentation},
	{0x1f9a5, 0x1f9aa, emojiStatusEmojiPresentation},
	{0x1f9ab, 0x1f9ad, emojiStatusEmojiPresentation},
	{0x1f9ae, 0x1f9af, emojiStatusEmojiPresentation},
	{0x1f9b0, 0x1f9b3, emojiStatusEmojiPresentationAndEmojiComponent},
	{0x1f9b4, 0x1f9b4, emojiStatusEmojiPresentation},
	{0x1f9b5, 0x1f9b6, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f9b7, 0x1f9b7, emojiStatusEmojiPresentation},
	{0x1f9b8, 0x1f9b9, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f9ba, 0x1f9ba, emojiStatusEmojiPresentation},
	{0x1f9bb, 0x1f9bb, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f9bc, 0x1f9bf, emojiStatusEmojiPresentation},
	{0x1f9c0, 0x1f9c0, emojiStatusEmojiPresentation},
	{0x1f9c1, 0x1f9c2, emojiStatusEmojiPresentation},
	{0x1f9c3, 0x1f9ca, emojiStatusEmojiPresentation},
	{0x1f9cb, 0x1f9cb, emojiStatusEmojiPresentation},
	{0x1f9cc, 0x1f9cc, emojiStatusEmojiPresentation},
	{0x1f9cd, 0x1f9cf, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f9d0, 0x1f9d0, emojiStatusEmojiPresentation},
	{0x1f9d1, 0x1f9dd, emojiStatusEmojiPresentationAndModifierBase},
	{0x1f9de, 0x1f9e6, emojiStatusEmojiPresentation},
	{0x1f9e7, 0x1f9ff, emojiStatusEmojiPresentation},
	{0x1fa00, 0x1fa6f, emojiStatusNonEmoji},
	{0x1fa70, 0x1fa73, emojiStatusEmojiPresentation},
	{0x1fa74, 0x1fa74, emojiStatusEmojiPresentation},
	{0x1fa75, 0x1fa77, emojiStatusEmojiPresentation},
	{0x1fa78, 0x1fa7a, emojiStatusEmojiPresentation},
	{0x1fa7b, 0x1fa7c, emojiStatusEmojiPresentation},
	{0x1fa7d, 0x1fa7f, emojiStatusNonEmoji},
	{0x1fa80, 0x1fa82, emojiStatusEmojiPresentation},
	{0x1fa83, 0x1fa86, emojiStatusEmojiPresentation},
	{0x1fa87, 0x1fa88, emojiStatusEmojiPresentation},
	{0x1fa89, 0x1fa89, emojiStatusEmojiPresentation},
	{0x1fa8a, 0x1fa8e, emojiStatusNonEmoji},
	{0x1fa8f, 0x1fa8f, emojiStatusEmojiPresentation},
	{0x1fa90, 0x1fa95, emojiStatusEmojiPresentation},
	{0x1fa96, 0x1faa8, emojiStatusEmojiPresentation},
	{0x1faa9, 0x1faac, emojiStatusEmojiPresentation},
	{0x1faad, 0x1faaf, emojiStatusEmojiPresentation},
	{0x1fab0, 0x1fab6, emojiStatusEmojiPresentation},
	{0x1fab7, 0x1faba, emojiStatusEmojiPresentation},
	{0x1fabb, 0x1fabd, emojiStatusEmojiPresentation},
	{0x1fabe, 0x1fabe, emojiStatusEmojiPresentation},
	{0x1fabf, 0x1fabf, emojiStatusEmojiPresentation},
	{0x1fac0, 0x1fac2, emojiStatusEmojiPresentation},
	{0x1fac3, 0x1fac5, emojiStatusEmojiPresentationAndModifierBase},
	{0x1fac6, 0x1fac6, emojiStatusEmojiPresentation},
	{0x1fac7, 0x1facd, emojiStatusNonEmoji},
	{0x1face, 0x1facf, emojiStatusEmojiPresentation},
	{0x1fad0, 0x1fad6, emojiStatusEmojiPresentation},
	{0x1fad7, 0x1fad9, emojiStatusEmojiPresentation},
	{0x1fada, 0x1fadb, emojiStatusEmojiPresentation},
	{0x1fadc, 0x1fadc, emojiStatusEmojiPresentation},
	{0x1fadd, 0x1fade, emojiStatusNonEmoji},
	{0x1fadf, 0x1fadf, emojiStatusEmojiPresentation},
	{0x1fae0, 0x1fae7, emojiStatusEmojiPresentation},
	{0x1fae8, 0x1fae8, emojiStatusEmojiPresentation},
	{0x1fae9, 0x1fae9, emojiStatusEmojiPresentation},
	{0x1faea, 0x1faef, emojiStatusNonEmoji},
	{0x1faf0, 0x1faf6, emojiStatusEmojiPresentationAndModifierBase},
	{0x1faf7, 0x1faf8, emojiStatusEmojiPresentationAndModifierBase},
	{0x1faf9, 0xe001f, emojiStatusNonEmoji},
	{0xe0020, 0xe007f, emojiStatusNonEmojiButEmojiComponent},
	{0xe0080, 0x10ffff, emojiStatusNonEmoji},
}
