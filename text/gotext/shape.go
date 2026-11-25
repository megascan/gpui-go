package gotext

import (
	"math"
	"os"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/bidi"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/segmenter"
)

const fontMatchesCacheSizeLimit = 256

type fontFeature struct {
	tag   [4]byte
	value uint32
}

type attrs struct {
	family   string
	stretch  uint16
	style    faceStyle
	weight   uint16
	features []fontFeature
	metadata int
}

func defaultAttrs() attrs {
	return attrs{family: "Arial", stretch: stretchNormal, style: faceStyleNormal, weight: weightNormal}
}

func (a *attrs) compatible(other *attrs) bool {
	return a.family == other.family && a.stretch == other.stretch && a.style == other.style && a.weight == other.weight
}

type matchAttrs struct {
	family  string
	stretch uint16
	style   faceStyle
	weight  uint16
}

func (a *attrs) matchKey() matchAttrs {
	return matchAttrs{family: a.family, stretch: a.stretch, style: a.style, weight: a.weight}
}

type attrsSpan struct {
	start, end int
	attrs      attrs
}

type attrsList struct {
	defaults attrs
	spans    []attrsSpan
}

func (l *attrsList) addSpan(start, end int, a attrs) {
	if start >= end {
		return
	}
	kept := l.spans[:0]
	var tail []attrsSpan
	for _, s := range l.spans {
		if s.end <= start || s.start >= end {
			kept = append(kept, s)
			continue
		}
		if s.start < start {
			kept = append(kept, attrsSpan{s.start, start, s.attrs})
		}
		if s.end > end {
			tail = append(tail, attrsSpan{end, s.end, s.attrs})
		}
	}
	kept = append(kept, attrsSpan{start, end, a})
	kept = append(kept, tail...)
	slices.SortFunc(kept, func(x, y attrsSpan) int { return x.start - y.start })
	l.spans = kept
}

func (l *attrsList) getSpan(index int) *attrs {
	for i := range l.spans {
		if l.spans[i].start <= index && index < l.spans[i].end {
			return &l.spans[i].attrs
		}
	}
	return &l.defaults
}

type fontMatchKey struct {
	notEmoji           bool
	fontWeightDiff     uint16
	fontStretchDiff    uint16
	fontStyleDiff      uint8
	fontWeight         uint16
	fontStretch        uint16
	id                 faceID
	variableWeightMatc bool
}

func absDiff(a, b uint16) uint16 {
	if a > b {
		return a - b
	}
	return b - a
}

func newFontMatchKey(a matchAttrs, face *faceInfo) fontMatchKey {
	var styleDiff uint8
	switch {
	case a.style == face.style:
		styleDiff = 0
	case a.style != faceStyleNormal && face.style != faceStyleNormal:
		styleDiff = 1
	default:
		styleDiff = 2
	}
	return fontMatchKey{
		notEmoji:        !strings.Contains(face.postScriptName, "Emoji"),
		fontWeightDiff:  absDiff(a.weight, face.weight),
		fontStretchDiff: absDiff(a.stretch, face.stretch),
		fontStyleDiff:   styleDiff,
		fontWeight:      face.weight,
		fontStretch:     face.stretch,
		id:              face.id,
	}
}

func boolCmp(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
}

func (k fontMatchKey) cmp(o fontMatchKey) int {
	if c := boolCmp(k.notEmoji, o.notEmoji); c != 0 {
		return c
	}
	for _, pair := range [][2]int{
		{int(k.fontWeightDiff), int(o.fontWeightDiff)},
		{int(k.fontStretchDiff), int(o.fontStretchDiff)},
		{int(k.fontStyleDiff), int(o.fontStyleDiff)},
		{int(k.fontWeight), int(o.fontWeight)},
		{int(k.fontStretch), int(o.fontStretch)},
		{int(k.id), int(o.id)},
	} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	return boolCmp(k.variableWeightMatc, o.variableWeightMatc)
}

type fontSystem struct {
	db               fontDatabase
	locale           string
	fontCache        map[faceID]*cosmicFont
	fontMatchesCache map[matchAttrs][]fontMatchKey
	buffer           *harfbuzz.Buffer
}

func newFontSystem(locale string) *fontSystem {
	return &fontSystem{
		locale:           locale,
		fontCache:        map[faceID]*cosmicFont{},
		fontMatchesCache: map[matchAttrs][]fontMatchKey{},
		buffer:           harfbuzz.NewBuffer(),
	}
}

func systemLocale() string {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := os.Getenv(name)
		if value == "" {
			continue
		}
		if i := strings.IndexAny(value, ".@"); i >= 0 {
			value = value[:i]
		}
		if value == "" || value == "C" || value == "POSIX" {
			continue
		}
		return strings.ReplaceAll(value, "_", "-")
	}
	return "en-US"
}

func (fs *fontSystem) dbMut() *fontDatabase {
	clear(fs.fontMatchesCache)
	return &fs.db
}

func (fs *fontSystem) getFont(id faceID) (*cosmicFont, bool) {
	if f, ok := fs.fontCache[id]; ok {
		return f, f != nil
	}
	f, ok := newCosmicFont(&fs.db, id)
	if !ok {
		fs.fontCache[id] = nil
		return nil, false
	}
	fs.fontCache[id] = f
	return f, true
}

func (fs *fontSystem) getFontMatches(a *attrs) []fontMatchKey {
	if len(fs.fontMatchesCache) >= fontMatchesCacheSizeLimit {
		clear(fs.fontMatchesCache)
	}
	key := a.matchKey()
	if keys, ok := fs.fontMatchesCache[key]; ok {
		return keys
	}
	keys := make([]fontMatchKey, 0, len(fs.db.faces))
	for i := range fs.db.faces {
		keys = append(keys, newFontMatchKey(key, &fs.db.faces[i]))
	}
	slices.SortStableFunc(keys, fontMatchKey.cmp)
	if id, ok := fs.db.query(a.family, a.weight, a.stretch, a.style); ok {
		if i := slices.IndexFunc(keys, func(k fontMatchKey) bool { return k.id == id }); i >= 0 {
			k := keys[i]
			keys = slices.Delete(keys, i, i+1)
			keys = slices.Insert(keys, 0, k)
		} else if face, ok := fs.db.face(id); ok {
			keys = slices.Insert(keys, 0, newFontMatchKey(key, face))
		}
	}
	fs.fontMatchesCache[key] = keys
	return keys
}

type fontFallbackIter struct {
	fs            *fontSystem
	keys          []fontMatchKey
	defaultFamily string
	defaultI      int
	scripts       []language.Script
	scriptI       [2]int
	commonI       int
	otherI        int
	end           bool
}

func newFontFallbackIter(fs *fontSystem, keys []fontMatchKey, defaultFamily string, scripts []language.Script) *fontFallbackIter {
	return &fontFallbackIter{fs: fs, keys: keys, defaultFamily: defaultFamily, scripts: scripts}
}

func (it *fontFallbackIter) faceContainsFamily(id faceID, family string) bool {
	face, ok := it.fs.db.face(id)
	return ok && face.hasFamily(family)
}

func (it *fontFallbackIter) next() (*cosmicFont, bool) {
	if it.defaultI < 1 {
		it.defaultI++
		for _, k := range it.keys {
			if (k.fontWeightDiff == 0 || k.variableWeightMatc) && it.faceContainsFamily(k.id, it.defaultFamily) {
				if f, ok := it.fs.getFont(k.id); ok {
					return f, true
				}
				break
			}
		}
	}
	fromFamilies := func(family string) (*cosmicFont, bool) {
		for _, k := range it.keys {
			if (k.fontWeightDiff == 0 || k.variableWeightMatc) && it.faceContainsFamily(k.id, family) {
				if f, ok := it.fs.getFont(k.id); ok {
					return f, true
				}
			}
		}
		return nil, false
	}
	for it.scriptI[0] < len(it.scripts) {
		families := scriptFallback(it.scripts[it.scriptI[0]], it.fs.locale)
		for it.scriptI[1] < len(families) {
			family := families[it.scriptI[1]]
			it.scriptI[1]++
			if f, ok := fromFamilies(family); ok {
				return f, true
			}
		}
		it.scriptI[0]++
		it.scriptI[1] = 0
	}
	common := commonFallback()
	for it.commonI < len(common) {
		family := common[it.commonI]
		it.commonI++
		if f, ok := fromFamilies(family); ok {
			return f, true
		}
	}
	forbidden := forbiddenFallback()
	for it.otherI < len(it.keys) {
		id := it.keys[it.otherI].id
		it.otherI++
		allowed := true
		for _, family := range forbidden {
			if it.faceContainsFamily(id, family) {
				allowed = false
			}
		}
		if allowed {
			if f, ok := it.fs.getFont(id); ok {
				return f, true
			}
		}
	}
	it.end = true
	return nil, false
}

type shapeGlyph struct {
	start, end int
	xAdvance   float32
	yAdvance   float32
	xOffset    float32
	yOffset    float32
	ascent     float32
	descent    float32
	fontID     faceID
	glyphID    uint16
	metadata   int
}

func (g *shapeGlyph) width(fontSize float32) float32 {
	return float32(fontSize * g.xAdvance)
}

func (fs *fontSystem) markUnloadable(id faceID) {
	fs.fontCache[id] = nil
	fs.buffer = harfbuzz.NewBuffer()
}

func shapeBuffer(buffer *harfbuzz.Buffer, font *harfbuzz.Font, features []harfbuzz.Feature) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	buffer.Shape(font, features)
	return true
}

func shapeFallback(buffer *harfbuzz.Buffer, glyphs []shapeGlyph, f *cosmicFont, line string, list *attrsList, startRun, endRun int, spanRTL bool) ([]shapeGlyph, []int, bool) {
	run := line[startRun:endRun]
	fontScale := f.metrics.unitsPerEm
	ascent := float32(f.metrics.ascent / fontScale)
	descent := float32(-f.metrics.descent / fontScale)

	text := run
	if strings.Contains(text, "\t") {
		text = strings.ReplaceAll(text, "\t", " ")
	}
	runes := make([]rune, 0, len(text))
	offsets := make([]int, 0, len(text))
	for i, r := range text {
		runes = append(runes, r)
		offsets = append(offsets, i)
	}
	buffer.Clear()
	buffer.ClusterLevel = harfbuzz.MonotoneGraphemes
	if spanRTL {
		buffer.Props.Direction = harfbuzz.RightToLeft
	} else {
		buffer.Props.Direction = harfbuzz.LeftToRight
	}
	buffer.AddRunes(runes, 0, len(runes))
	buffer.GuessSegmentProperties()
	buffer.Props.Language = ""

	spanAttrs := list.getSpan(startRun)
	features := make([]harfbuzz.Feature, 0, len(spanAttrs.features))
	for _, feature := range spanAttrs.features {
		features = append(features, harfbuzz.Feature{
			Tag:   harfbuzzTag(feature.tag),
			Value: feature.value,
			Start: harfbuzz.FeatureGlobalStart,
			End:   harfbuzz.FeatureGlobalEnd,
		})
	}
	if !shapeBuffer(buffer, f.shaper, features) {
		return glyphs, nil, false
	}

	var missing []int
	glyphStart := len(glyphs)
	for i, info := range buffer.Info {
		pos := buffer.Pos[i]
		startGlyph := startRun + offsets[info.Cluster]
		if info.Glyph == 0 {
			missing = append(missing, startGlyph)
		}
		a := list.getSpan(startGlyph)
		glyphs = append(glyphs, shapeGlyph{
			start:    startGlyph,
			end:      endRun,
			xAdvance: float32(float32(pos.XAdvance) / fontScale),
			yAdvance: float32(float32(pos.YAdvance) / fontScale),
			xOffset:  float32(float32(pos.XOffset) / fontScale),
			yOffset:  float32(float32(pos.YOffset) / fontScale),
			ascent:   ascent,
			descent:  descent,
			fontID:   f.id,
			glyphID:  uint16(info.Glyph),
			metadata: a.metadata,
		})
	}
	if spanRTL {
		for i := glyphStart + 1; i < len(glyphs); i++ {
			nextStart, nextEnd := glyphs[i-1].start, glyphs[i-1].end
			if glyphs[i].start == nextStart {
				glyphs[i].end = nextEnd
			} else {
				glyphs[i].end = nextStart
			}
		}
	} else {
		for i := len(glyphs) - 1; i > glyphStart; i-- {
			nextStart, nextEnd := glyphs[i].start, glyphs[i].end
			if glyphs[i-1].start == nextStart {
				glyphs[i-1].end = nextEnd
			} else {
				glyphs[i-1].end = nextStart
			}
		}
	}
	return glyphs, missing, true
}

func harfbuzzTag(tag [4]byte) ot.Tag {
	return ot.Tag(uint32(tag[0])<<24 | uint32(tag[1])<<16 | uint32(tag[2])<<8 | uint32(tag[3]))
}

func shapeRun(glyphs []shapeGlyph, fs *fontSystem, line string, list *attrsList, startRun, endRun int, spanRTL bool) []shapeGlyph {
	var scripts []language.Script
	for _, c := range line[startRun:endRun] {
		switch script := language.LookupScript(c); script {
		case language.Common, language.Inherited, language.Latin, language.Unknown:
		default:
			if !slices.Contains(scripts, script) {
				scripts = append(scripts, script)
			}
		}
	}
	a := list.getSpan(startRun)
	keys := fs.getFontMatches(a)
	iter := newFontFallbackIter(fs, keys, a.family, scripts)
	f, ok := iter.next()
	if !ok {
		panic("no default font found")
	}
	glyphStart := len(glyphs)
	var missing []int
	for {
		var shaped bool
		glyphs, missing, shaped = shapeFallback(fs.buffer, glyphs, f, line, list, startRun, endRun, spanRTL)
		if shaped {
			break
		}
		fs.markUnloadable(f.id)
		if f, ok = iter.next(); !ok {
			return glyphs
		}
	}
	for len(missing) > 0 {
		f, ok := iter.next()
		if !ok {
			break
		}
		var fbGlyphs []shapeGlyph
		fbGlyphs, fbMissing, shaped := shapeFallback(fs.buffer, fbGlyphs, f, line, list, startRun, endRun, spanRTL)
		if !shaped {
			fs.markUnloadable(f.id)
			continue
		}
		fbI := 0
		for fbI < len(fbGlyphs) {
			start, end := fbGlyphs[fbI].start, fbGlyphs[fbI].end
			if !slices.Contains(missing, start) || slices.Contains(fbMissing, start) {
				fbI++
				continue
			}
			missingI := 0
			for missingI < len(missing) {
				if missing[missingI] >= start && missing[missingI] < end {
					missing = slices.Delete(missing, missingI, missingI+1)
				} else {
					missingI++
				}
			}
			i := glyphStart
			for i < len(glyphs) {
				if glyphs[i].start >= start && glyphs[i].end <= end {
					break
				}
				i++
			}
			for i < len(glyphs) {
				if glyphs[i].start >= start && glyphs[i].end <= end {
					glyphs = slices.Delete(glyphs, i, i+1)
				} else {
					break
				}
			}
			for fbI < len(fbGlyphs) {
				if fbGlyphs[fbI].start >= start && fbGlyphs[fbI].end <= end {
					fbGlyph := fbGlyphs[fbI]
					fbGlyphs = slices.Delete(fbGlyphs, fbI, fbI+1)
					glyphs = slices.Insert(glyphs, i, fbGlyph)
					i++
				} else {
					break
				}
			}
		}
	}
	return glyphs
}

type shapeWord struct {
	blank  bool
	glyphs []shapeGlyph
}

func (w *shapeWord) width(fontSize float32) float32 {
	var width float32
	for i := range w.glyphs {
		width += w.glyphs[i].width(fontSize)
	}
	return width
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func graphemeOffsets(s string) []int {
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		return nil
	}
	var offsets []int
	it := seg.GraphemeIterator()
	for it.Next() {
		offsets = append(offsets, it.Grapheme().OffsetInBytes)
	}
	return offsets
}

func newShapeWord(fs *fontSystem, line string, list *attrsList, wordStart, wordEnd int, level bidi.Level, blank bool) shapeWord {
	word := line[wordStart:wordEnd]
	spanRTL := level%2 == 1
	var glyphs []shapeGlyph
	simpleASCII := isASCII(word) && !strings.ContainsFunc(word, func(c rune) bool { return c < 0x80 && unicode.IsControl(c) && c != '\t' })
	fast := simpleASCII && word != ""
	if fast {
		attrsStart := list.getSpan(wordStart)
		for i := range list.spans {
			other := &list.spans[i]
			if !(wordEnd <= other.start || other.end <= wordStart || attrsStart.compatible(&other.attrs)) {
				fast = false
				break
			}
		}
	}
	if fast {
		glyphs = shapeRun(glyphs, fs, line, list, wordStart, wordEnd, spanRTL)
	} else {
		startRun := wordStart
		current := &list.defaults
		for _, egc := range graphemeOffsets(word) {
			startEgc := wordStart + egc
			attrsEgc := list.getSpan(startEgc)
			if !current.compatible(attrsEgc) {
				glyphs = shapeRun(glyphs, fs, line, list, startRun, startEgc, spanRTL)
				startRun = startEgc
				current = attrsEgc
			}
		}
		if startRun < wordEnd {
			glyphs = shapeRun(glyphs, fs, line, list, startRun, wordEnd, spanRTL)
		}
	}
	return shapeWord{blank: blank, glyphs: glyphs}
}

type shapeSpan struct {
	level bidi.Level
	words []shapeWord
}

func lineBreaks(s string) []int {
	var seg segmenter.Segmenter
	if err := seg.InitWithString(s); err != nil {
		return nil
	}
	var breaks []int
	it := seg.LineIterator()
	for it.Next() {
		l := it.Line()
		breaks = append(breaks, l.OffsetInBytes+l.LengthInBytes)
	}
	return breaks
}

func isASCIIPunctuation(c rune) bool {
	return (c >= 0x21 && c <= 0x2F) || (c >= 0x3A && c <= 0x40) || (c >= 0x5B && c <= 0x60) || (c >= 0x7B && c <= 0x7E)
}

func newShapeSpan(fs *fontSystem, line string, list *attrsList, spanStart, spanEnd int, lineRTL bool, level bidi.Level) shapeSpan {
	span := line[spanStart:spanEnd]
	var words []shapeWord
	startWord := 0
	for _, endLB := range lineBreaks(span) {
		if endLB > 0 && endLB < len(span) {
			c1, _ := utf8.DecodeLastRuneInString(span[:endLB])
			c2, _ := utf8.DecodeRuneInString(span[endLB:])
			if isASCIIPunctuation(c1) && isASCIIPunctuation(c2) {
				probe := string([]rune{c1, c2})
				a := list.getSpan(spanStart + endLB)
				keys := fs.getFontMatches(a)
				iter := newFontFallbackIter(fs, keys, a.family, nil)
				if f, ok := iter.next(); ok {
					probeGlyphs, _, shaped := shapeFallback(fs.buffer, nil, f, probe, list, 0, len(probe), false)
					if !shaped {
						fs.markUnloadable(f.id)
					}
					if len(probeGlyphs) < 2 {
						continue
					}
					if len(probeGlyphs) == 2 && (f.charmap(c1) != probeGlyphs[0].glyphID || f.charmap(c2) != probeGlyphs[1].glyphID) {
						continue
					}
				}
			}
		}
		startLB := endLB
		segment := span[startWord:endLB]
		for i := len(segment); i > 0; {
			c, size := utf8.DecodeLastRuneInString(segment[:i])
			i -= size
			if unicode.IsSpace(c) {
				startLB = startWord + i
			} else {
				break
			}
		}
		if startWord < startLB {
			words = append(words, newShapeWord(fs, line, list, spanStart+startWord, spanStart+startLB, level, false))
		}
		if startLB < endLB {
			for i, c := range span[startLB:endLB] {
				start := spanStart + startLB + i
				words = append(words, newShapeWord(fs, line, list, start, start+utf8.RuneLen(c), level, true))
			}
		}
		startWord = endLB
	}
	if lineRTL {
		for i := range words {
			slices.Reverse(words[i].glyphs)
		}
	}
	if lineRTL != (level%2 == 1) {
		slices.Reverse(words)
	}
	return shapeSpan{level: level, words: words}
}

type shapeLine struct {
	rtl   bool
	spans []shapeSpan
}

func bidiLevels(text string) ([]bidi.Level, bidi.Level) {
	n := utf8.RuneCountInString(text)
	levels := make([]bidi.Level, n)
	if n == 0 {
		return levels, 0
	}
	if strings.ContainsFunc(text, isParagraphSeparator) {
		return levels, 0
	}
	var paragraph bidi.Paragraph
	runs := paragraph.SegmentString(text+" ", bidi.Neutral)
	var all []bidi.Level
	for i := range runs.NumRuns() {
		run := runs.Run(i)
		for range run.End - run.Start {
			all = append(all, run.Level)
		}
	}
	if len(all) != n+1 {
		return levels, 0
	}
	copy(levels, all[:n])
	return levels, all[n]
}

func newShapeLine(fs *fontSystem, line string, list *attrsList, tabWidth uint16) shapeLine {
	runeLevels, paragraphLevel := bidiLevels(line)
	rtl := paragraphLevel%2 == 1
	var spans []shapeSpan
	if line != "" {
		byteLevels := make([]bidi.Level, 0, len(runeLevels))
		starts := make([]int, 0, len(runeLevels))
		ri := 0
		for i := range line {
			starts = append(starts, i)
			byteLevels = append(byteLevels, runeLevels[ri])
			ri++
		}
		start := 0
		runLevel := byteLevels[0]
		for k := 1; k < len(starts); k++ {
			if byteLevels[k] != runLevel {
				spans = append(spans, newShapeSpan(fs, line, list, start, starts[k], rtl, runLevel))
				start = starts[k]
				runLevel = byteLevels[k]
			}
		}
		spans = append(spans, newShapeSpan(fs, line, list, start, len(line), rtl, runLevel))
	}
	var x float32
	for s := range spans {
		for w := range spans[s].words {
			glyphs := spans[s].words[w].glyphs
			for g := range glyphs {
				glyph := &glyphs[g]
				if glyph.start < glyph.end && glyph.end <= len(line) && line[glyph.start:glyph.end] == "\t" {
					tabXAdvance := float32(float32(tabWidth) * glyph.xAdvance)
					tabStop := float32((float32(math.Floor(float64(x/tabXAdvance))) + 1) * tabXAdvance)
					glyph.xAdvance = tabStop - x
				}
				x += glyph.xAdvance
			}
		}
	}
	return shapeLine{rtl: rtl, spans: spans}
}

type vlRange struct {
	span  int
	level bidi.Level
}

func reorder(ranges []vlRange) [][2]int {
	count := len(ranges)
	if count == 0 {
		return nil
	}
	elements := make([][2]int, count)
	for i := range elements {
		elements[i] = [2]int{i, i + 1}
	}
	minLevel, maxLevel := ranges[0].level, ranges[0].level
	for _, r := range ranges[1:] {
		minLevel = min(minLevel, r.level)
		maxLevel = max(maxLevel, r.level)
	}
	if minLevel%2 == 0 {
		minLevel++
	}
	for maxLevel >= minLevel {
		seqStart := 0
		for seqStart < count {
			if ranges[elements[seqStart][0]].level < maxLevel {
				seqStart++
				continue
			}
			seqEnd := seqStart + 1
			for seqEnd < count {
				if ranges[elements[seqEnd][0]].level < maxLevel {
					break
				}
				seqEnd++
			}
			slices.Reverse(elements[seqStart:seqEnd])
			seqStart = seqEnd
		}
		maxLevel--
	}
	return elements
}

type layoutGlyph struct {
	start, end int
	fontID     faceID
	glyphID    uint16
	x, y, w    float32
	xOffset    float32
	yOffset    float32
	metadata   int
}

type layoutLine struct {
	w          float32
	maxAscent  float32
	maxDescent float32
	glyphs     []layoutGlyph
}

func (l *shapeLine) layout(fontSize float32) layoutLine {
	var ranges []vlRange
	var w float32
	for spanIndex := range l.spans {
		span := &l.spans[spanIndex]
		congruent := l.rtl == (span.level%2 == 1)
		var wordRangeWidth float32
		if congruent {
			for i := range span.words {
				wordRangeWidth += span.words[i].width(fontSize)
			}
		} else {
			for i := len(span.words) - 1; i >= 0; i-- {
				wordRangeWidth += span.words[i].width(fontSize)
			}
		}
		if len(span.words) == 0 {
			continue
		}
		ranges = append(ranges, vlRange{span: spanIndex, level: span.level})
		w += wordRangeWidth
	}
	if len(ranges) == 0 {
		return layoutLine{}
	}
	lineWidth := max(float32(0), w)
	var x float32
	if l.rtl {
		x = lineWidth
	}
	var y, maxAscent, maxDescent float32
	var glyphs []layoutGlyph
	order := reorder(ranges)
	if l.rtl {
		slices.Reverse(order)
	}
	for _, element := range order {
		for _, r := range ranges[element[0]:element[1]] {
			for _, word := range l.spans[r.span].words {
				for _, glyph := range word.glyphs {
					xAdvance := float32(fontSize * glyph.xAdvance)
					if l.rtl {
						x -= xAdvance
					}
					yAdvance := float32(fontSize * glyph.yAdvance)
					glyphs = append(glyphs, layoutGlyph{
						start:    glyph.start,
						end:      glyph.end,
						fontID:   glyph.fontID,
						glyphID:  glyph.glyphID,
						x:        x,
						y:        y,
						w:        xAdvance,
						xOffset:  glyph.xOffset,
						yOffset:  glyph.yOffset,
						metadata: glyph.metadata,
					})
					if !l.rtl {
						x += xAdvance
					}
					y += yAdvance
					maxAscent = max(maxAscent, float32(fontSize*glyph.ascent))
					maxDescent = max(maxDescent, float32(fontSize*glyph.descent))
				}
			}
		}
	}
	return layoutLine{w: w, maxAscent: maxAscent, maxDescent: maxDescent, glyphs: glyphs}
}

func commonFallback() []string {
	return platformCommonFallback
}

func forbiddenFallback() []string {
	return platformForbiddenFallback
}
