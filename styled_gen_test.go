package gpui

import (
	"bufio"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var handWrittenStyledMethods = []string{
	"Block", "Flex", "Grid", "Hidden", "ScrollbarWidth", "WhitespaceNormal", "WhitespaceNowrap",
	"TextEllipsis", "TextEllipsisStart", "TextEllipsisMiddle", "TextOverflow", "TextAlign",
	"TextLeft", "TextCenter", "TextRight", "Truncate", "LineClamp", "FlexCol", "FlexColReverse",
	"FlexRow", "FlexRowReverse", "Flex1", "FlexAuto", "FlexInitial", "FlexNone", "FlexBasis",
	"FlexGrow", "FlexGrow0", "FlexGrow1", "FlexShrink", "FlexShrink0", "FlexShrink1", "FlexWrap",
	"FlexWrapReverse", "FlexNowrap", "ItemsStart", "ItemsEnd", "ItemsCenter", "ItemsBaseline",
	"ItemsStretch", "SelfStart", "SelfEnd", "SelfFlexStart", "SelfFlexEnd", "SelfCenter",
	"SelfBaseline", "SelfStretch", "JustifyStart", "JustifyEnd", "JustifyCenter", "JustifyBetween",
	"JustifyAround", "JustifyEvenly", "ContentNormal", "ContentCenter", "ContentStart", "ContentEnd",
	"ContentBetween", "ContentAround", "ContentEvenly", "ContentStretch", "AspectRatio",
	"AspectSquare", "Bg", "BorderDashed", "TextStyle", "TextColor", "FontWeight", "TextBg",
	"TextSize", "TextXs", "TextSm", "TextBase", "TextLg", "TextXl", "Text2xl", "Text3xl", "Italic",
	"NotItalic", "Underline", "LineThrough", "TextDecorationNone", "TextDecorationColor",
	"TextDecorationSolid", "TextDecorationWavy", "TextDecoration0", "TextDecoration1",
	"TextDecoration2", "TextDecoration4", "TextDecoration8", "FontFamily", "FontFeatures", "Font",
	"LineHeight", "Opacity", "GridCols", "GridColsMinContent", "GridColsMaxContent", "GridRows",
	"GridRowsMinContent", "GridRowsMaxContent", "ColStart", "ColStartAuto", "ColEnd", "ColEndAuto",
	"ColSpan", "ColSpanFull", "RowStart", "RowStartAuto", "RowEnd", "RowEndAuto", "RowSpan",
	"RowSpanFull", "Debug", "DebugBelow",
}

func TestStyledGen_MethodSet(t *testing.T) {
	f, err := os.Open("docs/port/parity/styled-methods.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var generated []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 2 {
			t.Fatalf("malformed row %q", line)
		}
		generated = append(generated, cols[1])
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(generated) != 3099 {
		t.Fatalf("styled-methods.tsv has %d rows, want 3099", len(generated))
	}
	if len(handWrittenStyledMethods) != 115 {
		t.Fatalf("%d hand-written methods, want 115 (styled.rs's 116 minus style, which the element declares)", len(handWrittenStyledMethods))
	}

	want := slices.Concat(generated, handWrittenStyledMethods)
	slices.Sort(want)
	if dup := slices.Compact(slices.Clone(want)); len(dup) != len(want) {
		t.Fatalf("generated and hand-written method names overlap")
	}

	typ := reflect.TypeFor[*Styled[*Div]]()
	var got []string
	for m := range typ.Methods() {
		got = append(got, m.Name)
	}
	slices.Sort(got)

	if !slices.Equal(got, want) {
		gotSet := map[string]bool{}
		for _, n := range got {
			gotSet[n] = true
		}
		wantSet := map[string]bool{}
		for _, n := range want {
			wantSet[n] = true
			if !gotSet[n] {
				t.Errorf("missing method %s", n)
			}
		}
		for _, n := range got {
			if !wantSet[n] {
				t.Errorf("unexpected method %s", n)
			}
		}
	}
}
