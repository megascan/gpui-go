//go:build darwin

package apple

import (
	"cmp"
	"encoding/binary"
	"errors"
	"math"
	"slices"

	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/cg"
	"github.com/megascan/gpui-go/internal/apple/ct"
)

var errFontKitNotFound = errors.New("no font found")
var errFontKitParse = errors.New("failed to parse font")
var errFontKitNoSuchFontInCollection = errors.New("no such font in collection")

var fontWeightMapping = [9]float32{-0.7, -0.5, -0.23, 0.0, 0.2, 0.3, 0.4, 0.6, 0.8}

var stretchMapping = [9]float32{0.5, 0.625, 0.75, 0.875, 1.0, 1.125, 1.25, 1.5, 2.0}

const stretchNormal float32 = 1.0

type fontKitStyle uint8

const (
	fontKitStyleNormal fontKitStyle = iota
	fontKitStyleItalic
	fontKitStyleOblique
)

type fontKitProperties struct {
	style   fontKitStyle
	weight  float32
	stretch float32
}

type fontKitMetrics struct {
	unitsPerEm         uint32
	ascent             float32
	descent            float32
	lineGap            float32
	underlinePosition  float32
	underlineThickness float32
	capHeight          float32
	xHeight            float32
	boundingBox        rectF
}

type rectF struct {
	minX, minY, maxX, maxY float32
}

func newRectF(x, y, w, h float32) rectF {
	return rectF{minX: x, minY: y, maxX: x + w, maxY: y + h}
}

func (r rectF) scale(s float32) rectF {
	return rectF{minX: r.minX * s, minY: r.minY * s, maxX: r.maxX * s, maxY: r.maxY * s}
}

func (r rectF) width() float32  { return r.maxX - r.minX }
func (r rectF) height() float32 { return r.maxY - r.minY }

type rectI struct {
	minX, minY, maxX, maxY int32
}

func (r rectF) roundOutToI32() rectI {
	return rectI{
		minX: int32(float32(math.Floor(float64(r.minX)))),
		minY: int32(float32(math.Floor(float64(r.minY)))),
		maxX: int32(float32(math.Ceil(float64(r.maxX)))),
		maxY: int32(float32(math.Ceil(float64(r.maxY)))),
	}
}

type fontKitFont struct {
	native *cf.Owned[ct.Font]
}

func fontKitFromNativeFont(f ct.Font) *fontKitFont {
	return &fontKitFont{native: cf.RetainRef(f)}
}

func fontKitFromCoreGraphicsFont(g cg.Font) *fontKitFont {
	return &fontKitFont{native: ct.FontCreateWithGraphicsFont(g, 16.0)}
}

func fontKitFromBytes(data []byte, fontIndex uint32) (*fontKitFont, error) {
	if !fontIsSingleOTF(data) && !fontIsCollection(data) {
		unpacked := slices.Clone(data)
		if err := unpackDataForkFont(unpacked); err != nil {
			return nil, err
		}
		data = unpacked
	} else if fontIsCollection(data) {
		unpacked := slices.Clone(data)
		if err := unpackOTCFont(unpacked, fontIndex); err != nil {
			return nil, err
		}
		data = unpacked
	}
	cfData := cf.DataCreate(data)
	if cfData == nil {
		return nil, errFontKitParse
	}
	defer cfData.Release()
	desc := ct.FontManagerCreateFontDescriptorFromData(cfData.Ref())
	if desc == nil {
		return nil, errFontKitParse
	}
	defer desc.Release()
	return &fontKitFont{native: ct.FontCreateWithFontDescriptor(desc.Ref(), 16.0)}, nil
}

func (f *fontKitFont) nativeFont() ct.Font {
	return f.native.Ref()
}

func (f *fontKitFont) release() {
	f.native.Release()
}

func (f *fontKitFont) postscriptName() (string, bool) {
	return f.nativeFont().PostScriptName(), true
}

func (f *fontKitFont) fullName() string {
	return f.nativeFont().DisplayName()
}

func (f *fontKitFont) familyName() string {
	return f.nativeFont().FamilyName()
}

type fontTraits struct {
	dict *cf.Owned[cf.Dictionary]
}

func (f *fontKitFont) allTraits() fontTraits {
	return fontTraits{dict: f.nativeFont().CopyTraits()}
}

func (t fontTraits) release() {
	t.dict.Release()
}

func (t fontTraits) number(key cf.String) (ct.Number, bool) {
	if t.dict == nil {
		return 0, false
	}
	v, ok := t.dict.Ref().Get(cf.TypeRef(key))
	if !ok {
		return 0, false
	}
	return ct.AsNumber(v)
}

func (t fontTraits) extractFloat(key cf.String) float64 {
	n, ok := t.number(key)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	v, ok := n.Float64()
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return v
}

func (t fontTraits) normalizedWeight() float64 { return t.extractFloat(ct.FontWeightTrait) }
func (t fontTraits) normalizedWidth() float64  { return t.extractFloat(ct.FontWidthTrait) }
func (t fontTraits) normalizedSlant() float64  { return t.extractFloat(ct.FontSlantTrait) }

func (f *fontKitFont) properties() fontKitProperties {
	symbolicTraits := f.nativeFont().SymbolicTraits()
	allTraits := f.allTraits()
	defer allTraits.release()
	var style fontKitStyle
	switch {
	case symbolicTraits&ct.FontItalicTrait != 0:
		style = fontKitStyleItalic
	case allTraits.normalizedSlant() > 0.0:
		style = fontKitStyleOblique
	default:
		style = fontKitStyleNormal
	}
	return fontKitProperties{
		style:   style,
		weight:  coreTextToCSSFontWeight(float32(allTraits.normalizedWeight())),
		stretch: coreTextWidthToCSSStretchiness(float32(allTraits.normalizedWidth())),
	}
}

func (f *fontKitFont) glyphForChar(character rune) (uint32, bool) {
	var src [2]uint16
	if character >= 0x10000 {
		c := character - 0x10000
		src[0] = uint16(0xD800 + (c >> 10))
		src[1] = uint16(0xDC00 + (c & 0x3FF))
	} else {
		src[0] = uint16(character)
	}
	var dest [2]cg.Glyph
	f.nativeFont().GlyphsForCharacters(src[:], dest[:])
	id := uint32(dest[0])
	if id != 0 {
		return id, true
	}
	return 0, false
}

func (f *fontKitFont) unitsPerPoint() float64 {
	return float64(f.nativeFont().UnitsPerEm()) / f.nativeFont().PtSize()
}

func (f *fontKitFont) typographicBounds(glyphID uint32) (rectF, error) {
	rect := f.nativeFont().BoundingRectForGlyph(ct.FontDefaultOrientation, cg.Glyph(glyphID))
	r := newRectF(float32(rect.Origin.X), float32(rect.Origin.Y), float32(rect.Size.Width), float32(rect.Size.Height))
	return r.scale(float32(f.unitsPerPoint())), nil
}

func (f *fontKitFont) advance(glyphID uint32) (float32, float32, error) {
	a := f.nativeFont().AdvanceForGlyph(ct.FontDefaultOrientation, cg.Glyph(glyphID))
	upp := float32(f.unitsPerPoint())
	return float32(a.Width) * upp, float32(a.Height) * upp, nil
}

func (f *fontKitFont) metrics() fontKitMetrics {
	font := f.nativeFont()
	unitsPerEm := font.UnitsPerEm()
	unitsPerPoint := float64(unitsPerEm) / font.PtSize()
	bb := font.BoundingBox()
	boundingBox := newRectF(float32(bb.Origin.X), float32(bb.Origin.Y), float32(bb.Size.Width), float32(bb.Size.Height)).scale(float32(unitsPerPoint))
	return fontKitMetrics{
		unitsPerEm:         unitsPerEm,
		ascent:             float32(font.Ascent() * unitsPerPoint),
		descent:            float32(-font.Descent() * unitsPerPoint),
		lineGap:            float32(font.Leading() * unitsPerPoint),
		underlinePosition:  float32(font.UnderlinePosition() * unitsPerPoint),
		underlineThickness: float32(font.UnderlineThickness() * unitsPerPoint),
		capHeight:          float32(font.CapHeight() * unitsPerPoint),
		xHeight:            float32(font.XHeight() * unitsPerPoint),
		boundingBox:        boundingBox,
	}
}

func (f *fontKitFont) rasterBounds(glyphID uint32, pointSize float32, scale float32) (rectI, error) {
	typographicBounds, err := f.typographicBounds(glyphID)
	if err != nil {
		return rectI{}, err
	}
	b := typographicBounds.scale(pointSize / float32(f.metrics().unitsPerEm))
	flipped := newRectF(b.minX, -b.minY-b.height(), b.width(), b.height())
	return flipped.scale(scale).roundOutToI32(), nil
}

func coreTextToCSSFontWeight(coreTextWeight float32) float32 {
	index := piecewiseLinearFindIndex(coreTextWeight, fontWeightMapping[:])
	return index*100.0 + 100.0
}

func coreTextWidthToCSSStretchiness(coreTextWidth float32) float32 {
	return piecewiseLinearLookup((coreTextWidth+1.0)*4.0, stretchMapping[:])
}

func piecewiseLinearLookup(index float32, mapping []float32) float32 {
	lower := mapping[int(math.Floor(float64(index)))]
	upper := mapping[int(math.Ceil(float64(index)))]
	t := index - float32(math.Trunc(float64(index)))
	return lower + (upper-lower)*t
}

func piecewiseLinearFindIndex(query float32, mapping []float32) float32 {
	upperIndex, found := slices.BinarySearchFunc(mapping, query, func(v, q float32) int {
		if v < q {
			return -1
		}
		if v > q {
			return 1
		}
		if v == q {
			return 0
		}
		return -1
	})
	if found {
		return float32(upperIndex)
	}
	if upperIndex == 0 || upperIndex >= len(mapping) {
		return float32(upperIndex)
	}
	lowerIndex := upperIndex - 1
	upperValue, lowerValue := mapping[upperIndex], mapping[lowerIndex]
	t := (query - lowerValue) / (upperValue - lowerValue)
	return float32(lowerIndex) + t
}

var ttcTag = [4]byte{'t', 't', 'c', 'f'}
var ottoTag = [4]byte{'O', 'T', 'T', 'O'}

const (
	sfntHex uint32 = 0x73666e74
	ottoHex uint32 = 0x4f54544f
	trueHex uint32 = 0x74727565
	typ1Hex uint32 = 0x74797031
)

func fontIsCollection(header []byte) bool {
	return len(header) >= 4 && [4]byte(header[:4]) == ttcTag
}

func fontIsSingleOTF(header []byte) bool {
	return len(header) >= 4 && (binary.BigEndian.Uint32(header) == 0x00010000 || [4]byte(header[:4]) == ottoTag)
}

func readU32(data []byte, at int) (uint32, error) {
	if at < 0 || at+4 > len(data) {
		return 0, errFontKitParse
	}
	return binary.BigEndian.Uint32(data[at:]), nil
}

func readU16(data []byte, at int) (uint16, error) {
	if at < 0 || at+2 > len(data) {
		return 0, errFontKitParse
	}
	return binary.BigEndian.Uint16(data[at:]), nil
}

func readU24(data []byte, at int) (uint32, error) {
	if at < 0 || at+3 > len(data) {
		return 0, errFontKitParse
	}
	return uint32(data[at])<<16 | uint32(data[at+1])<<8 | uint32(data[at+2]), nil
}

func readNumberOfFontsFromOTCHeader(header []byte) (uint32, error) {
	if !fontIsCollection(header) {
		return 0, errors.New("unknown font format")
	}
	return readU32(header, 8)
}

func unpackOTCFont(data []byte, fontIndex uint32) error {
	n, err := readNumberOfFontsFromOTCHeader(data)
	if err != nil {
		return err
	}
	if fontIndex >= n {
		return errFontKitNoSuchFontInCollection
	}
	offsetTablePos, err := readU32(data, 12+4*int(fontIndex))
	if err != nil {
		return err
	}
	numTables, err := readU16(data, int(offsetTablePos)+4)
	if err != nil {
		return err
	}
	size := 12 + int(numTables)*16
	if int(offsetTablePos)+size > len(data) {
		return errFontKitParse
	}
	for offset := range size {
		data[offset] = data[int(offsetTablePos)+offset]
	}
	return nil
}

func unpackDataForkFont(data []byte) error {
	dataOffset32, err := readU32(data, 0)
	if err != nil {
		return err
	}
	dataOffset := int(dataOffset32)
	mapOffset32, err := readU32(data, 4)
	if err != nil {
		return err
	}
	mapOffset := int(mapOffset32)
	numTypes16, err := readU16(data, mapOffset+28)
	if err != nil {
		return err
	}
	numTypes := int(numTypes16) + 1
	fontDataOffset, fontDataLen := 0, 0
	typeList16, err := readU16(data, mapOffset+24)
	if err != nil {
		return err
	}
	typeListOffset := int(typeList16) + mapOffset
	for i := range numTypes {
		resType, err := readU32(data, mapOffset+30+i*8)
		if err != nil {
			return err
		}
		if resType == sfntHex {
			refListOffset, err := readU16(data, mapOffset+30+i*8+6)
			if err != nil {
				return err
			}
			resDataOffset, err := readU24(data, typeListOffset+int(refListOffset)+5)
			if err != nil {
				return err
			}
			l, err := readU32(data, dataOffset+int(resDataOffset))
			if err != nil {
				return err
			}
			fontDataLen = int(l)
			fontDataOffset = dataOffset + int(resDataOffset) + 4
			sfntVersion, err := readU32(data, fontDataOffset)
			if err != nil {
				return err
			}
			if sfntVersion == 0x00010000 || sfntVersion == ottoHex || sfntVersion == trueHex || sfntVersion == typ1Hex {
				break
			}
		}
	}
	if fontDataLen == 0 {
		return errFontKitParse
	}
	if fontDataOffset+fontDataLen > len(data) {
		return errFontKitParse
	}
	copy(data[:fontDataLen], data[fontDataOffset:fontDataOffset+fontDataLen])
	return nil
}

type fontHandle struct {
	bytes  []byte
	index  uint32
	native ct.Font
}

func (h fontHandle) load() (*fontKitFont, error) {
	if h.native != 0 {
		return fontKitFromNativeFont(h.native), nil
	}
	return fontKitFromBytes(h.bytes, h.index)
}

type familyHandle struct {
	fonts []fontHandle
	owned []*cf.Owned[ct.Font]
}

func (f familyHandle) release() {
	for _, o := range f.owned {
		o.Release()
	}
}

type memFamilyEntry struct {
	familyName     string
	postscriptName string
	font           fontHandle
}

type memSource struct {
	families []memFamilyEntry
}

func (m *memSource) addFonts(handles []fontHandle) error {
	for _, h := range handles {
		font, err := h.load()
		if err != nil {
			return err
		}
		if name, ok := font.postscriptName(); ok {
			m.families = append(m.families, memFamilyEntry{familyName: font.familyName(), postscriptName: name, font: h})
		}
		font.release()
	}
	slices.SortStableFunc(m.families, func(a, b memFamilyEntry) int { return cmp.Compare(a.familyName, b.familyName) })
	return nil
}

func (m *memSource) allFamilies() []string {
	var families []string
	for _, family := range m.families {
		if len(families) > 0 && families[len(families)-1] == family.familyName {
			continue
		}
		families = append(families, family.familyName)
	}
	return families
}

func (m *memSource) selectFamilyByName(familyName string) (familyHandle, error) {
	first, found := slices.BinarySearchFunc(m.families, familyName, func(e memFamilyEntry, name string) int {
		return cmp.Compare(e.familyName, name)
	})
	if !found {
		return familyHandle{}, errFontKitNotFound
	}
	for first > 0 && m.families[first-1].familyName == familyName {
		first--
	}
	last := first
	for last+1 < len(m.families) && m.families[last+1].familyName == familyName {
		last++
	}
	var fh familyHandle
	for _, e := range m.families[first : last+1] {
		fh.fonts = append(fh.fonts, e.font)
	}
	return fh, nil
}

type systemSource struct{}

func (systemSource) selectFamilyByName(familyName string) (familyHandle, error) {
	key := cf.StringCreate("NSFontFamilyAttribute")
	defer key.Release()
	name := cf.StringCreate(familyName)
	if name == nil {
		return familyHandle{}, errFontKitNotFound
	}
	defer name.Release()
	attrs := cf.DictionaryCreate([]cf.TypeRef{key.TypeRef()}, []cf.TypeRef{name.TypeRef()})
	defer attrs.Release()
	desc := ct.FontDescriptorCreateWithAttributes(attrs.Ref())
	defer desc.Release()
	descs := cf.ArrayCreate([]cf.TypeRef{desc.TypeRef()})
	defer descs.Release()
	collection := ct.FontCollectionCreateWithFontDescriptors(descs.Ref())
	if collection == nil {
		return familyHandle{}, errFontKitNotFound
	}
	defer collection.Release()
	return createHandlesFromCoreTextCollection(collection.Ref())
}

func createHandlesFromCoreTextCollection(collection ct.FontCollection) (familyHandle, error) {
	var fh familyHandle
	if descriptors := collection.CreateMatchingFontDescriptors(); descriptors != nil {
		arr := descriptors.Ref()
		for i := range arr.Count() {
			native := ct.FontCreateWithFontDescriptor(ct.FontDescriptor(arr.At(i)), 16.0)
			if native == nil {
				continue
			}
			fh.owned = append(fh.owned, native)
			fh.fonts = append(fh.fonts, fontHandle{native: native.Ref()})
		}
		descriptors.Release()
	}
	if len(fh.fonts) == 0 {
		return familyHandle{}, errFontKitNotFound
	}
	return fh, nil
}

func findBestMatch(candidates []fontKitProperties, query fontKitProperties) (int, error) {
	matchingSet := make([]int, len(candidates))
	for i := range candidates {
		matchingSet[i] = i
	}
	if len(matchingSet) == 0 {
		return 0, errFontKitNotFound
	}
	minBy := func(filter func(int) bool, key func(int) float32) (int, bool) {
		best, bestKey, ok := 0, float32(0), false
		for _, index := range matchingSet {
			if !filter(index) {
				continue
			}
			k := key(index)
			if !ok || floatOrdLess(k, bestKey) {
				best, bestKey, ok = index, k, true
			}
		}
		return best, ok
	}
	all := func(int) bool { return true }
	anyWhere := func(pred func(int) bool) bool {
		return slices.ContainsFunc(matchingSet, pred)
	}
	retain := func(pred func(int) bool) {
		matchingSet = slices.DeleteFunc(matchingSet, func(i int) bool { return !pred(i) })
	}

	var matchingStretch float32
	switch {
	case anyWhere(func(i int) bool { return candidates[i].stretch == query.stretch }):
		matchingStretch = query.stretch
	case query.stretch <= stretchNormal:
		if i, ok := minBy(func(i int) bool { return candidates[i].stretch < query.stretch }, func(i int) float32 { return query.stretch - candidates[i].stretch }); ok {
			matchingStretch = candidates[i].stretch
		} else {
			i, _ := minBy(all, func(i int) float32 { return candidates[i].stretch - query.stretch })
			matchingStretch = candidates[i].stretch
		}
	default:
		if i, ok := minBy(func(i int) bool { return candidates[i].stretch > query.stretch }, func(i int) float32 { return candidates[i].stretch - query.stretch }); ok {
			matchingStretch = candidates[i].stretch
		} else {
			i, _ := minBy(all, func(i int) float32 { return query.stretch - candidates[i].stretch })
			matchingStretch = candidates[i].stretch
		}
	}
	retain(func(i int) bool { return candidates[i].stretch == matchingStretch })

	var stylePreference [3]fontKitStyle
	switch query.style {
	case fontKitStyleItalic:
		stylePreference = [3]fontKitStyle{fontKitStyleItalic, fontKitStyleOblique, fontKitStyleNormal}
	case fontKitStyleOblique:
		stylePreference = [3]fontKitStyle{fontKitStyleOblique, fontKitStyleItalic, fontKitStyleNormal}
	default:
		stylePreference = [3]fontKitStyle{fontKitStyleNormal, fontKitStyleOblique, fontKitStyleItalic}
	}
	matchingStyle, found := fontKitStyle(0), false
	for _, s := range stylePreference {
		if anyWhere(func(i int) bool { return candidates[i].style == s }) {
			matchingStyle, found = s, true
			break
		}
	}
	if !found {
		panic("called `Option::unwrap()` on a `None` value")
	}
	retain(func(i int) bool { return candidates[i].style == matchingStyle })

	var matchingWeight float32
	switch {
	case anyWhere(func(i int) bool { return candidates[i].weight == query.weight }):
		matchingWeight = query.weight
	case query.weight >= 400.0 && query.weight < 450.0 && anyWhere(func(i int) bool { return candidates[i].weight == 500.0 }):
		matchingWeight = 500.0
	case query.weight >= 450.0 && query.weight <= 500.0 && anyWhere(func(i int) bool { return candidates[i].weight == 400.0 }):
		matchingWeight = 400.0
	case query.weight <= 500.0:
		if i, ok := minBy(func(i int) bool { return candidates[i].weight <= query.weight }, func(i int) float32 { return query.weight - candidates[i].weight }); ok {
			matchingWeight = candidates[i].weight
		} else {
			i, _ := minBy(all, func(i int) float32 { return candidates[i].weight - query.weight })
			matchingWeight = candidates[i].weight
		}
	default:
		if i, ok := minBy(func(i int) bool { return candidates[i].weight >= query.weight }, func(i int) float32 { return candidates[i].weight - query.weight }); ok {
			matchingWeight = candidates[i].weight
		} else {
			i, _ := minBy(all, func(i int) float32 { return query.weight - candidates[i].weight })
			matchingWeight = candidates[i].weight
		}
	}
	retain(func(i int) bool { return candidates[i].weight == matchingWeight })

	if len(matchingSet) == 0 {
		return 0, errFontKitNotFound
	}
	return matchingSet[0], nil
}

func floatOrdLess(a, b float32) bool {
	an, bn := math.IsNaN(float64(a)), math.IsNaN(float64(b))
	switch {
	case an && bn:
		return false
	case an:
		return false
	case bn:
		return true
	}
	return a < b
}
