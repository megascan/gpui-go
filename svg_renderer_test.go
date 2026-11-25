package gpui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xo/okage"
	"github.com/xo/okage/svg"
)

//go:embed text/gotext/testdata/fonts/ibm-plex-sans_IBMPlexSans-Regular.ttf
var svgRendererTestIBMPlexRegular []byte

//go:embed text/gotext/testdata/fonts/lilex_Lilex-Regular.ttf
var svgRendererTestLilexRegular []byte

func svgRendererTestMust[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func TestSvgRenderer_RendersParsedSvgAtRequestedSize(t *testing.T) {
	renderer := NewSvgRenderer(NoAssets{})
	parsed := svgRendererTestMust(renderer.ParseSvg([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24pt" height="12pt"></svg>`)))
	requestedSize := NewSize(DevicePixels(24), DevicePixels(12))
	image := svgRendererTestMust(renderer.RenderParsed(parsed, SvgSizeExactSize(requestedSize)))

	if got := image.Size(0); got != requestedSize {
		t.Fatalf("size = %v, want %v", got, requestedSize)
	}
}

func TestSvgRenderer_PreservesAspectRatioForWidthConstrainedSize(t *testing.T) {
	renderer := NewSvgRenderer(NoAssets{})
	parsed := svgRendererTestMust(renderer.ParseSvg([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24pt" height="12pt"></svg>`)))
	image := svgRendererTestMust(renderer.RenderParsed(parsed, SvgSizeSize(NewSize(DevicePixels(24), DevicePixels(24)))))

	if got, want := image.Size(0), NewSize(DevicePixels(24), DevicePixels(12)); got != want {
		t.Fatalf("size = %v, want %v", got, want)
	}
}

func TestSvgRenderer_MeasuredTextWidthMatchesRasterizedAdvance(t *testing.T) {
	renderer := NewSvgRenderer(svgRendererTestBundledFonts{})
	font := SvgTextFont{
		Family: "IBM Plex Sans",
		Size:   40.,
		Weight: "normal",
		Style:  "normal",
	}
	frameWidth := svgRendererTestRasterizedTextWidth(t, renderer, "MM")
	if !(frameWidth > 0.) {
		t.Fatalf("frame width = %v", frameWidth)
	}

	for _, text := range []string{"MMMM", " ", "   ", " ", "​", "M​"} {
		measured := svgRendererTestMust(renderer.MeasureTextWidth(text, font))
		rasterized := svgRendererTestRasterizedTextWidth(t, renderer, "M"+text+"M") - frameWidth
		if svgTestAbs(measured-rasterized) > 1. {
			t.Fatalf("%q measured %vpx, rasterized %vpx", text, measured, rasterized)
		}
	}
	if a, b := svgRendererTestMust(renderer.MeasureTextWidth("M​", font)), svgRendererTestMust(renderer.MeasureTextWidth("M", font)); a != b {
		t.Fatalf("width of M+ZWSP = %v, width of M = %v", a, b)
	}
	if w := svgRendererTestMust(renderer.MeasureTextWidth("<&\"\u0001>", font)); !(w > 0.) {
		t.Fatalf("escaped width = %v", w)
	}
}

func TestSvgRenderer_OwnsFontBytesWithinBudget(t *testing.T) {
	dir := t.TempDir()
	fontPath := func(name string) string {
		return filepath.Join(dir, fmt.Sprintf("gpui-svg-renderer-%d-%s.ttf", os.Getpid(), name))
	}
	plexPath, lilexPath := fontPath("plex"), fontPath("lilex")
	if err := os.WriteFile(plexPath, svgRendererTestIBMPlexRegular, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lilexPath, svgRendererTestLilexRegular, 0o644); err != nil {
		t.Fatal(err)
	}
	files := newFontFileDatabase()
	files.loadFontFile(plexPath)
	files.loadFontFile(lilexPath)
	fonts := &fontDatabase{
		database:            files.database,
		faceFiles:           files.faceFiles,
		ownedFaces:          map[svg.FontID]svg.FontID{},
		remainingOwnedBytes: uint64(len(svgRendererTestIBMPlexRegular)),
		fallbackFaces:       map[fallbackFaceKey]Option[svg.FontID]{},
	}
	db := fonts.database.Clone()
	var faceIDs []svg.FontID
	for face := range db.Faces() {
		faceIDs = append(faceIDs, face.ID)
	}
	plexID, lilexID := faceIDs[0], faceIDs[1]

	ownedPlexID := fonts.ownedFace(db, plexID)
	ownedLilexID := fonts.ownedFace(db, lilexID)
	if err := os.WriteFile(plexPath, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	dataLen := func(id svg.FontID) Option[int] {
		data, _, ok := db.FaceData(id)
		if !ok {
			return None[int]()
		}
		return Some(len(data))
	}
	dataLens := [2]Option[int]{dataLen(plexID), dataLen(ownedPlexID)}

	if want := [2]Option[int]{Some(7), Some(len(svgRendererTestIBMPlexRegular))}; dataLens != want {
		t.Fatalf("data lengths = %v, want %v", dataLens, want)
	}
	if ownedLilexID != lilexID {
		t.Fatalf("owned lilex = %v, want %v", ownedLilexID, lilexID)
	}
	if fonts.remainingOwnedBytes != 0 {
		t.Fatalf("remaining owned bytes = %d", fonts.remainingOwnedBytes)
	}
	if got := fonts.ownedFace(db, plexID); got != ownedPlexID {
		t.Fatalf("owned plex = %v, want %v", got, ownedPlexID)
	}
}

func TestSvgRenderer_FallbackFacesAreRememberedUntilExcluded(t *testing.T) {
	database := svgRendererTestDBWithBundledFonts()
	var faceIDs []svg.FontID
	for face := range database.Faces() {
		faceIDs = append(faceIDs, face.ID)
	}
	base, fallback := faceIDs[0], faceIDs[1]
	fontdb := &fontDatabase{
		database:            database,
		faceFiles:           map[svg.FontID]string{},
		ownedFaces:          map[svg.FontID]svg.FontID{},
		remainingOwnedBytes: 0,
		fallbackFaces:       map[fallbackFaceKey]Option[svg.FontID]{},
	}
	selections := 0
	fallbackFace := func(ch rune, fonts []svg.FontID) Option[svg.FontID] {
		id, ok := cachedFallbackFace(fontdb, ch, fonts, func(excluded []svg.FontID) (svg.FontID, bool) {
			selections++
			if slices.Contains(excluded, fallback) {
				return 0, false
			}
			return fallback, true
		})
		if !ok {
			return None[svg.FontID]()
		}
		return Some(id)
	}

	check := func(got, want Option[svg.FontID]) {
		t.Helper()
		if got != want {
			t.Fatalf("face = %v, want %v", got, want)
		}
	}
	checkSelections := func(want int) {
		t.Helper()
		if selections != want {
			t.Fatalf("selections = %d, want %d", selections, want)
		}
	}
	check(fallbackFace('x', []svg.FontID{base}), Some(fallback))
	check(fallbackFace('x', []svg.FontID{base}), Some(fallback))
	checkSelections(1)
	check(fallbackFace('x', []svg.FontID{base, fallback}), None[svg.FontID]())
	checkSelections(2)
	check(fallbackFace('y', []svg.FontID{base, fallback}), None[svg.FontID]())
	checkSelections(4)
	check(fallbackFace('y', []svg.FontID{base}), Some(fallback))
	checkSelections(4)
}

func svgRendererTestDBWithBundledFonts() *svg.FontDatabase {
	db := svg.NewFontDatabase()
	db.LoadFontData(slices.Clone(svgRendererTestIBMPlexRegular))
	db.LoadFontData(slices.Clone(svgRendererTestLilexRegular))
	return db
}

func TestSvgRenderer_TextWithSplitGlyphClustersInMixedFontsDoesNotPanic(t *testing.T) {
	db := svg.NewFontDatabase()
	db.LoadFontData(slices.Clone(svgRendererTestIBMPlexRegular))
	db.LoadFontData(slices.Clone(svgRendererTestLilexRegular))

	zalgo := "é̂̃̄̆̇̈̊"
	data := fmt.Sprintf(`<svg viewBox="0 0 200 200" xmlns="http://www.w3.org/2000/svg"><text font-family="Lilex" font-size="32">%s<tspan font-family="IBM Plex Sans">%s</tspan></text></svg>`, zalgo, zalgo)

	if _, err := svg.FromData([]byte(data), svg.WithFontDatabase(db)); err != nil {
		t.Fatalf("SVG with mixed-font text should parse: %v", err)
	}
}

func TestSvgRenderer_IsEmojiPresentation(t *testing.T) {
	cases := []struct {
		s        string
		expected bool
	}{
		{"a", false},
		{"Z", false},
		{"1", false},
		{"#", false},
		{"*", false},
		{"漢", false},
		{"中", false},
		{"カ", false},
		{"©", false},
		{"♥", false},
		{"😀", true},
		{"✅", true},
		{"🇺🇸", true},
		{"©️", false},
		{"♥️", false},
		{"1️⃣", false},
	}
	for _, c := range cases {
		first := []rune(c.s)[0]
		if got := isEmojiPresentation(first); got != c.expected {
			t.Fatalf("for char %q: got %v, want %v", first, got, c.expected)
		}
	}
}

func TestSvgRenderer_FixGenericFontFamiliesSetsAllFamilies(t *testing.T) {
	db := svgRendererTestDBWithBundledFonts()
	fixGenericFontFamilies(db)

	families := []svg.FontFamilyKind{
		svg.FontFamilySansSerif,
		svg.FontFamilySerif,
		svg.FontFamilyMonospace,
		svg.FontFamilyCursive,
		svg.FontFamilyFantasy,
	}

	for _, family := range families {
		if _, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: family})); !ok {
			t.Fatalf("Expected generic family %v to resolve after fix_generic_font_families", family)
		}
	}
}

func TestSvgRenderer_SelectEmojiFontSkipsFamilyWithoutGlyph(t *testing.T) {
	db := svgRendererTestDBWithBundledFonts()

	ibmPlexSans, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: svg.FontFamilyNamed, Name: "IBM Plex Sans"}))
	if !ok {
		t.Fatal("IBM Plex Sans should resolve")
	}
	lilex, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: svg.FontFamilyNamed, Name: "Lilex"}))
	if !ok {
		t.Fatal("Lilex should resolve")
	}
	selected, ok := selectEmojiFont('│', nil, db, []string{"IBM Plex Sans", "Lilex"})
	if !ok {
		t.Fatal("a font should be selected")
	}

	if selected != lilex {
		t.Fatalf("selected = %v, want %v", selected, lilex)
	}
	if fontHasChar(db, ibmPlexSans, '│') {
		t.Fatal("IBM Plex Sans should not have │")
	}
	if !fontHasChar(db, selected, '│') {
		t.Fatal("the selected font should have │")
	}
}

func TestSvgRenderer_FixGenericFontFamiliesMonospaceResolvesToLilex(t *testing.T) {
	db := svgRendererTestDBWithBundledFonts()
	fixGenericFontFamilies(db)

	id, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: svg.FontFamilyMonospace}))
	if !ok {
		t.Fatal("Monospace should resolve")
	}
	face, ok := db.Face(id)
	if !ok {
		t.Fatal("Face should exist")
	}
	if !slices.ContainsFunc(face.Families, func(name svg.FontFamilyName) bool { return strings.Contains(name.Name, "Lilex") }) {
		t.Fatalf("Monospace should map to Lilex, got %v", face.Families)
	}
}

type svgRendererTestBundledFonts struct{}

func (svgRendererTestBundledFonts) Load(path string) ([]byte, bool, error) {
	switch path {
	case "fonts/ibm-plex-sans/IBMPlexSans-Regular.ttf":
		return svgRendererTestIBMPlexRegular, true, nil
	case "fonts/lilex/Lilex-Regular.ttf":
		return svgRendererTestLilexRegular, true, nil
	}
	return nil, false, nil
}

func (svgRendererTestBundledFonts) List(string) ([]SharedString, error) {
	return nil, nil
}

func svgRendererTestRasterizedTextWidth(t *testing.T, renderer SvgRenderer, text string) float32 {
	t.Helper()
	data := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="400" height="60"><text xml:space="preserve" x="10" y="45" font-family="IBM Plex Sans" font-size="40">%s</text></svg>`, text)
	pixmap := svgRendererTestMust(renderer.renderPixmap([]byte(data), SvgSizeScaleFactor(1.)))
	return svgRendererTestInkedWidth(pixmap)
}

func svgRendererTestInkedWidth(pixmap *okage.Pixmap) float32 {
	width := int(pixmap.Width())
	data := pixmap.Data()
	pixels := len(data) / 4
	first, last := -1, -1
	for column := range width {
		for i := column; i < pixels; i += width {
			if data[i*4+3] > 0 {
				if first < 0 {
					first = column
				}
				last = column
				break
			}
		}
	}
	if first < 0 {
		return 0.
	}
	return float32(last - first + 1)
}
