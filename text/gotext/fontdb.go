package gotext

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/fontscan"
)

type faceID int

type faceStyle uint8

const (
	faceStyleNormal faceStyle = iota
	faceStyleItalic
	faceStyleOblique
)

const (
	weightNormal  uint16 = 400
	stretchNormal uint16 = 5
)

type faceSource struct {
	data []byte
	path string
}

type faceInfo struct {
	id             faceID
	source         *faceSource
	index          int
	families       []string
	postScriptName string
	style          faceStyle
	weight         uint16
	stretch        uint16
	monospaced     bool
}

type fontDatabase struct {
	faces []faceInfo
}

func (db *fontDatabase) face(id faceID) (*faceInfo, bool) {
	if id < 0 || int(id) >= len(db.faces) {
		return nil, false
	}
	return &db.faces[id], true
}

func (db *fontDatabase) loadFontData(data []byte) {
	db.loadSource(&faceSource{data: data}, bytes.NewReader(data))
}

func (db *fontDatabase) loadFontFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	db.loadSource(&faceSource{path: path}, file)
}

func (db *fontDatabase) loadSource(source *faceSource, resource ot.Resource) {
	loaders, err := ot.NewLoaders(resource)
	if err != nil {
		return
	}
	for index, loader := range loaders {
		info, ok := parseFaceInfo(loader)
		if !ok {
			continue
		}
		info.id = faceID(len(db.faces))
		info.source = source
		info.index = index
		db.faces = append(db.faces, info)
	}
}

func (db *fontDatabase) loadSystemFonts() {
	dirs, err := fontscan.DefaultFontDirectories(discardLogger{})
	if err != nil {
		return
	}
	home, _ := os.UserHomeDir()
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		if strings.HasPrefix(dir, "~") {
			if home == "" {
				continue
			}
			dir = filepath.Join(home, dir[1:])
		}
		_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".ttc", ".otf", ".otc":
			default:
				return nil
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			if _, ok := seen[resolved]; ok {
				return nil
			}
			seen[resolved] = struct{}{}
			db.loadFontFile(resolved)
			return nil
		})
	}
}

type discardLogger struct{}

func (discardLogger) Printf(string, ...any) {}

func (db *fontDatabase) faceBytes(id faceID) ([]byte, int, error) {
	info, ok := db.face(id)
	if !ok {
		return nil, 0, errors.New("font face not found")
	}
	if info.source.data == nil {
		data, err := os.ReadFile(info.source.path)
		if err != nil {
			return nil, 0, err
		}
		info.source.data = data
	}
	return info.source.data, info.index, nil
}

func (db *fontDatabase) loadFont(id faceID) (*font.Font, *ot.Loader, error) {
	data, index, err := db.faceBytes(id)
	if err != nil {
		return nil, nil, err
	}
	loaders, err := ot.NewLoaders(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	if index >= len(loaders) {
		return nil, nil, errors.New("font face index out of range")
	}
	parsed, err := font.NewFont(loaders[index])
	if err != nil {
		return nil, nil, err
	}
	return parsed, loaders[index], nil
}

func (db *fontDatabase) query(family string, weight, stretch uint16, style faceStyle) (faceID, bool) {
	var candidates []*faceInfo
	for i := range db.faces {
		if db.faces[i].hasFamily(family) {
			candidates = append(candidates, &db.faces[i])
		}
	}
	if len(candidates) == 0 {
		return 0, false
	}
	index, ok := fontdbFindBestMatch(candidates, weight, stretch, style)
	if !ok {
		return 0, false
	}
	return candidates[index].id, true
}

func (f *faceInfo) hasFamily(name string) bool {
	for _, family := range f.families {
		if family == name {
			return true
		}
	}
	return false
}

func fontdbFindBestMatch(candidates []*faceInfo, weight, stretch uint16, style faceStyle) (int, bool) {
	matching := make([]int, len(candidates))
	for i := range matching {
		matching[i] = i
	}
	minBy := func(filter func(int) bool, key func(int) int) (int, bool) {
		best, bestKey, found := 0, 0, false
		for _, index := range matching {
			if !filter(index) {
				continue
			}
			k := key(index)
			if !found || k < bestKey {
				best, bestKey, found = index, k, true
			}
		}
		return best, found
	}
	all := func(int) bool { return true }
	exact := false
	for _, index := range matching {
		if candidates[index].stretch == stretch {
			exact = true
		}
	}
	var matchingStretch uint16
	if exact {
		matchingStretch = stretch
	} else if stretch <= stretchNormal {
		index, ok := minBy(func(i int) bool { return candidates[i].stretch < stretch }, func(i int) int { return int(stretch) - int(candidates[i].stretch) })
		if !ok {
			index, ok = minBy(all, func(i int) int { return int(candidates[i].stretch) - int(stretch) })
			if !ok {
				return 0, false
			}
		}
		matchingStretch = candidates[index].stretch
	} else {
		index, ok := minBy(func(i int) bool { return candidates[i].stretch > stretch }, func(i int) int { return int(candidates[i].stretch) - int(stretch) })
		if !ok {
			index, ok = minBy(all, func(i int) int { return int(stretch) - int(candidates[i].stretch) })
			if !ok {
				return 0, false
			}
		}
		matchingStretch = candidates[index].stretch
	}
	retain := func(keep func(int) bool) {
		kept := matching[:0]
		for _, index := range matching {
			if keep(index) {
				kept = append(kept, index)
			}
		}
		matching = kept
	}
	retain(func(i int) bool { return candidates[i].stretch == matchingStretch })

	var preference [3]faceStyle
	switch style {
	case faceStyleItalic:
		preference = [3]faceStyle{faceStyleItalic, faceStyleOblique, faceStyleNormal}
	case faceStyleOblique:
		preference = [3]faceStyle{faceStyleOblique, faceStyleItalic, faceStyleNormal}
	default:
		preference = [3]faceStyle{faceStyleNormal, faceStyleOblique, faceStyleItalic}
	}
	matchingStyle, found := faceStyleNormal, false
	for _, want := range preference {
		for _, index := range matching {
			if candidates[index].style == want {
				matchingStyle, found = want, true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return 0, false
	}
	retain(func(i int) bool { return candidates[i].style == matchingStyle })

	has := func(w uint16) bool {
		for _, index := range matching {
			if candidates[index].weight == w {
				return true
			}
		}
		return false
	}
	var matchingWeight uint16
	switch {
	case has(weight):
		matchingWeight = weight
	case weight >= 400 && weight < 450 && has(500):
		matchingWeight = 500
	case weight >= 450 && weight <= 500 && has(400):
		matchingWeight = 400
	case weight <= 500:
		index, ok := minBy(func(i int) bool { return candidates[i].weight <= weight }, func(i int) int { return int(weight) - int(candidates[i].weight) })
		if !ok {
			index, ok = minBy(all, func(i int) int { return int(candidates[i].weight) - int(weight) })
			if !ok {
				return 0, false
			}
		}
		matchingWeight = candidates[index].weight
	default:
		index, ok := minBy(func(i int) bool { return candidates[i].weight >= weight }, func(i int) int { return int(candidates[i].weight) - int(weight) })
		if !ok {
			index, ok = minBy(all, func(i int) int { return int(weight) - int(candidates[i].weight) })
			if !ok {
				return 0, false
			}
		}
		matchingWeight = candidates[index].weight
	}
	retain(func(i int) bool { return candidates[i].weight == matchingWeight })
	if len(matching) == 0 {
		return 0, false
	}
	return matching[0], true
}

var (
	tagName = ot.MustNewTag("name")
	tagOS2  = ot.MustNewTag("OS/2")
	tagPost = ot.MustNewTag("post")
	tagHead = ot.MustNewTag("head")
	tagHhea = ot.MustNewTag("hhea")
	tagHmtx = ot.MustNewTag("hmtx")
	tagVhea = ot.MustNewTag("vhea")
	tagVmtx = ot.MustNewTag("vmtx")
	tagMaxp = ot.MustNewTag("maxp")
)

func parseFaceInfo(loader *ot.Loader) (faceInfo, bool) {
	nameData, err := loader.RawTable(tagName)
	if err != nil {
		return faceInfo{}, false
	}
	families, postScriptName, ok := parseNames(nameData)
	if !ok {
		return faceInfo{}, false
	}
	style, weight, stretch := parseOS2(loader)
	monospaced, italic := parsePost(loader)
	if style == faceStyleNormal && italic {
		style = faceStyleItalic
	}
	return faceInfo{
		families:       families,
		postScriptName: postScriptName,
		style:          style,
		weight:         weight,
		stretch:        stretch,
		monospaced:     monospaced,
	}, true
}

type nameRecord struct {
	platformID uint16
	encodingID uint16
	languageID uint16
	nameID     uint16
	value      []byte
}

const (
	nameIDFamily            = 1
	nameIDPostScriptName    = 6
	nameIDTypographicFamily = 16
)

func parseNameRecords(data []byte) []nameRecord {
	if len(data) < 6 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(data[2:]))
	storage := int(binary.BigEndian.Uint16(data[4:]))
	var records []nameRecord
	for i := range count {
		offset := 6 + i*12
		if offset+12 > len(data) {
			break
		}
		r := data[offset:]
		length := int(binary.BigEndian.Uint16(r[8:]))
		start := storage + int(binary.BigEndian.Uint16(r[10:]))
		if start+length > len(data) {
			continue
		}
		records = append(records, nameRecord{
			platformID: binary.BigEndian.Uint16(r[0:]),
			encodingID: binary.BigEndian.Uint16(r[2:]),
			languageID: binary.BigEndian.Uint16(r[4:]),
			nameID:     binary.BigEndian.Uint16(r[6:]),
			value:      data[start : start+length],
		})
	}
	return records
}

func (r nameRecord) isUnicode() bool {
	switch r.platformID {
	case 0:
		return true
	case 3:
		return r.encodingID == 0 || r.encodingID == 1 || r.encodingID == 10
	}
	return false
}

func (r nameRecord) isMacRoman() bool {
	return r.platformID == 1 && r.encodingID == 0
}

func (r nameRecord) isEnglishUnitedStates() bool {
	switch r.platformID {
	case 3:
		return r.languageID == 0x0409
	case 1:
		return r.languageID == 0
	}
	return false
}

func (r nameRecord) toUnicode() (string, bool) {
	if r.isUnicode() {
		if len(r.value)%2 != 0 {
			return "", false
		}
		units := make([]uint16, len(r.value)/2)
		for i := range units {
			units[i] = binary.BigEndian.Uint16(r.value[2*i:])
		}
		for i := 0; i < len(units); i++ {
			u := units[i]
			if utf16.IsSurrogate(rune(u)) {
				if u >= 0xDC00 || i+1 >= len(units) || units[i+1] < 0xDC00 || units[i+1] > 0xDFFF {
					return "", false
				}
				i++
			}
		}
		return string(utf16.Decode(units)), true
	}
	if r.isMacRoman() {
		var b strings.Builder
		for _, c := range r.value {
			if c < 0x80 {
				b.WriteByte(c)
			} else {
				b.WriteRune(macRoman[c-0x80])
			}
		}
		return b.String(), true
	}
	return "", false
}

type familyName struct {
	name     string
	englishU bool
}

func collectFamilies(nameID uint16, records []nameRecord) []familyName {
	var families []familyName
	for _, r := range records {
		if r.nameID == nameID && r.isUnicode() {
			if family, ok := r.toUnicode(); ok {
				families = append(families, familyName{family, r.isEnglishUnitedStates()})
			}
		}
	}
	hasEnglish := false
	for _, f := range families {
		if f.englishU {
			hasEnglish = true
		}
	}
	if !hasEnglish {
		for _, r := range records {
			if r.nameID == nameID && r.isMacRoman() {
				if family, ok := r.toUnicode(); ok {
					families = append(families, familyName{family, r.isEnglishUnitedStates()})
					break
				}
			}
		}
	}
	return families
}

func parseNames(data []byte) ([]string, string, bool) {
	records := parseNameRecords(data)
	families := collectFamilies(nameIDTypographicFamily, records)
	if len(families) == 0 {
		families = collectFamilies(nameIDFamily, records)
	}
	if len(families) > 1 {
		for i, f := range families {
			if f.englishU {
				if i != 0 {
					families[0], families[i] = families[i], families[0]
				}
				break
			}
		}
	}
	if len(families) == 0 {
		return nil, "", false
	}
	postScriptName, found := "", false
	for _, r := range records {
		if r.nameID == nameIDPostScriptName && (r.isUnicode() || r.isMacRoman()) {
			postScriptName, found = r.toUnicode()
			break
		}
	}
	if !found {
		return nil, "", false
	}
	names := make([]string, len(families))
	for i, f := range families {
		names[i] = f.name
	}
	return names, postScriptName, true
}

func parseOS2(loader *ot.Loader) (faceStyle, uint16, uint16) {
	data, err := loader.RawTable(tagOS2)
	if err != nil || len(data) < 78 {
		return faceStyleNormal, weightNormal, stretchNormal
	}
	version := binary.BigEndian.Uint16(data[0:])
	selection := binary.BigEndian.Uint16(data[62:])
	style := faceStyleNormal
	if selection&1 != 0 {
		style = faceStyleItalic
	} else if version >= 4 && selection&(1<<9) != 0 {
		style = faceStyleOblique
	}
	weight := binary.BigEndian.Uint16(data[4:])
	stretch := binary.BigEndian.Uint16(data[6:])
	if stretch < 1 || stretch > 9 {
		stretch = stretchNormal
	}
	return style, weight, stretch
}

func parsePost(loader *ot.Loader) (bool, bool) {
	data, err := loader.RawTable(tagPost)
	if err != nil {
		return false, false
	}
	zero := []byte{0, 0, 0, 0}
	monospaced := len(data) < 16 || !bytes.Equal(data[12:16], zero)
	italic := len(data) < 8 || !bytes.Equal(data[4:8], zero)
	return monospaced, italic
}

var macRoman = [128]rune{
	0x00C4, 0x00C5, 0x00C7, 0x00C9, 0x00D1, 0x00D6, 0x00DC, 0x00E1, 0x00E0, 0x00E2, 0x00E4, 0x00E3, 0x00E5, 0x00E7, 0x00E9, 0x00E8,
	0x00EA, 0x00EB, 0x00ED, 0x00EC, 0x00EE, 0x00EF, 0x00F1, 0x00F3, 0x00F2, 0x00F4, 0x00F6, 0x00F5, 0x00FA, 0x00F9, 0x00FB, 0x00FC,
	0x2020, 0x00B0, 0x00A2, 0x00A3, 0x00A7, 0x2022, 0x00B6, 0x00DF, 0x00AE, 0x00A9, 0x2122, 0x00B4, 0x00A8, 0x2260, 0x00C6, 0x00D8,
	0x221E, 0x00B1, 0x2264, 0x2265, 0x00A5, 0x00B5, 0x2202, 0x2211, 0x220F, 0x03C0, 0x222B, 0x00AA, 0x00BA, 0x03A9, 0x00E6, 0x00F8,
	0x00BF, 0x00A1, 0x00AC, 0x221A, 0x0192, 0x2248, 0x2206, 0x00AB, 0x00BB, 0x2026, 0x00A0, 0x00C0, 0x00C3, 0x00D5, 0x0152, 0x0153,
	0x2013, 0x2014, 0x201C, 0x201D, 0x2018, 0x2019, 0x00F7, 0x25CA, 0x00FF, 0x0178, 0x2044, 0x20AC, 0x2039, 0x203A, 0xFB01, 0xFB02,
	0x2021, 0x00B7, 0x201A, 0x201E, 0x2030, 0x00C2, 0x00CA, 0x00C1, 0x00CB, 0x00C8, 0x00CD, 0x00CE, 0x00CF, 0x00CC, 0x00D3, 0x00D4,
	0xF8FF, 0x00D2, 0x00DA, 0x00DB, 0x00D9, 0x0131, 0x02C6, 0x02DC, 0x00AF, 0x02D8, 0x02D9, 0x02DA, 0x00B8, 0x02DD, 0x02DB, 0x02C7,
}
