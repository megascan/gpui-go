//go:build js && wasm

package web

import (
	"slices"
	"testing"
)

func TestCanvasFallback_CanvasFontFallbackPolicy(t *testing.T) {
	if DefaultCanvasFontFallback() != CanvasFontFallbackEmoji {
		t.Fatalf("default = %v", DefaultCanvasFontFallback())
	}
	for _, tc := range []struct {
		grapheme string
		allowed  bool
	}{
		{"\U0001f600", true},
		{"❤️", true},
		{"1️⃣", true},
		{"\U0001f468‍\U0001f469‍\U0001f467‍\U0001f466", true},
		{"\U0001f578", true},
		{"\U0001f578︎", true},
		{"\U0001f578️", true},
		{"©", true},
		{"❤︎", true},
		{"中", false},
		{"が", false},
		{"각", false},
	} {
		fallback, ok := ClassifyCanvasFallback(tc.grapheme)
		if !ok {
			t.Fatalf("eligible grapheme %q", tc.grapheme)
		}
		if CanvasFontFallbackDisabled.allows(fallback.IsEmoji) {
			t.Fatalf("disabled allows %q", tc.grapheme)
		}
		if CanvasFontFallbackEmoji.allows(fallback.IsEmoji) != tc.allowed {
			t.Fatalf("%q", tc.grapheme)
		}
		if !CanvasFontFallbackEmojiAndCjk.allows(fallback.IsEmoji) {
			t.Fatalf("emoji and cjk rejects %q", tc.grapheme)
		}
	}
}

func assertClassify(t *testing.T, text string, want CanvasFallback, wantOK bool) {
	t.Helper()
	got, ok := ClassifyCanvasFallback(text)
	if ok != wantOK || (ok && got != want) {
		t.Fatalf("%q: got (%+v, %v), want (%+v, %v)", text, got, ok, want, wantOK)
	}
}

func TestCanvasFallback_AsciiIsIneligibleButKeycapsArePreserved(t *testing.T) {
	for b := 0; b <= 0x7f; b++ {
		assertClassify(t, string(rune(b)), CanvasFallback{}, false)
	}
	for _, text := range []string{"", "Hello", "0123456789#*", "\r\n"} {
		assertClassify(t, text, CanvasFallback{}, false)
	}
	for _, base := range "0123456789#*" {
		for _, text := range []string{string(base) + "⃣", string(base) + "️⃣"} {
			assertClassify(t, text, CanvasFallback{IsEmoji: true, EmojiPresentation: true}, true)
		}
	}
}

func TestCanvasFallback_OrdinaryCjkClusters(t *testing.T) {
	for _, text := range []string{
		"漢", "\U00020000", "﨑", "漢︀", "葛\U000e0100", "葛\U000e01ef",
		"あ", "ガ", "が", "パ", "가", "각", "가",
		"각", "각", "ㄱ", "、", "。", "「", "」",
		"（", "！", "ー", "々",
	} {
		assertClassify(t, text, CanvasFallback{IsEmoji: false, EmojiPresentation: false}, true)
	}
}

func TestCanvasFallback_EmojiPresentationIsPreserved(t *testing.T) {
	for _, tc := range []struct {
		text              string
		emojiPresentation bool
	}{
		{"\U0001f600", true},
		{"\U0001f578", false},
		{"\U0001f578︎", false},
		{"\U0001f578️", true},
		{"©", false},
		{"©︎", false},
		{"©️", true},
		{"❤", false},
		{"❤︎", false},
		{"❤️", true},
		{"\U0001f600︎", false},
		{"\U0001f44d\U0001f3fd", true},
		{"☝\U0001f3fd", true},
		{"\U0001f44d️\U0001f3fd", true},
		{"\U0001f1ef\U0001f1f5", true},
		{"1⃣", true},
		{"#️⃣", true},
		{"*️⃣", true},
		{"\U0001f469\U0001f3fd‍\U0001f4bb", true},
		{"\U0001f9d1‍⚕️", true},
		{"\U0001f468‍\U0001f469‍\U0001f467‍\U0001f466", true},
		{"\U0001f3f3️‍\U0001f308", true},
		{"\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f", true},
		{"\U0001f3f4\U000e0067\U000e0062\U000e0073\U000e0063\U000e0074\U000e007f", true},
		{"\U0001f3f4\U000e0067\U000e0062\U000e0077\U000e006c\U000e0073\U000e007f", true},
	} {
		assertClassify(t, tc.text, CanvasFallback{IsEmoji: true, EmojiPresentation: tc.emojiPresentation}, true)
	}
}

func TestCanvasFallback_UnsupportedScriptsAndCjkFormsStayOnCosmic(t *testing.T) {
	for _, text := range []string{
		"a", "é", "α", "Ж", "ش", "ש", "क", "क्ष",
		"ก", "ក", "ᠠ", "ཀ", "\x00", "\n", "\r\n", "‮", "‍", "﷐",
		"\U0010ffff", "\U0002a6e0", "", "゙", "️", "\U000e0100", "漢́",
		"漢︃", "漢️", "漢\U000e0100\U000e0101", "あ゙", "か゚",
		"が゙", "ᄀ", "ᅡ", "ᆨ", "ᄓᅡ", "ᄀᅶ",
		"가ᇃ", "각ᆨ", "ᄀ가", "ㅤ", "ㅥ", "⺀",
		"⼀", "㇀", "㆐", "ㄅ", "㐀‍", "\U0001b000", "ｶ", "！́",
	} {
		assertClassify(t, text, CanvasFallback{}, false)
	}
}

func TestCanvasFallback_MalformedOrOutOfPolicyEmojiStayOnCosmic(t *testing.T) {
	for _, text := range []string{
		"0", "#", "*", "1️", "#︎", "1︎⃣", "a⃣", "\U0001f3fd", "\U0001f1ef",
		"\U0001f600\U0001f3fd", "\U0001f44d\U0001f3fd\U0001f3fd", "\U0001f44d︎\U0001f3fd",
		"\U0001f44d\U0001f3fd️", "❤️️", "\U0001f600́", "\U0001f9b0",
		"\U0001f600‍\U0001f600", "\U0001f469‍", "\U0001f469‍\U0001f4bb‍\U0001f680",
		"\U0001f469︎‍\U0001f4bb", "\U0001f3f4\U000e0067", "\U0001f3f4\U000e0061\U000e0062\U000e007f",
		"\U0001f600\U000e0067\U000e007f",
	} {
		assertClassify(t, text, CanvasFallback{}, false)
	}
}

func TestCanvasFallback_AcceptsOnlyOneWholeExtendedGrapheme(t *testing.T) {
	for _, text := range []string{
		"", "漢字", "あい", "가나", "\U0001f600\U0001f600",
		"\U0001f1ef\U0001f1f5\U0001f1fa", "\U0001f1ef\U0001f1f5\U0001f1fa\U0001f1f8", " 漢", "漢\n",
	} {
		assertClassify(t, text, CanvasFallback{}, false)
	}
	text := "aが\U0001f469\U0001f3fd‍\U0001f4bb漢\U000e0100"
	var eligible []string
	for _, g := range graphemeIndices(text) {
		if _, ok := ClassifyCanvasFallback(g.text); ok {
			eligible = append(eligible, text[g.start:g.start+len(g.text)])
		}
	}
	want := []string{"が", "\U0001f469\U0001f3fd‍\U0001f4bb", "漢\U000e0100"}
	if !slices.Equal(eligible, want) {
		t.Fatalf("eligible = %q, want %q", eligible, want)
	}
}
