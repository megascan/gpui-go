package gpui

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func buildWrapper() *LineWrapper {
	textSystem := NewTextSystem(NewNoopTextSystem())
	id := textSystem.ResolveFont(NewFont(".ZedMono"))
	return newLineWrapper(id, Px(16.), textSystem, defaultCharWidthCache())
}

func collectBoundaries(seq func(func(Boundary) bool)) []Boundary {
	var out []Boundary
	seq(func(b Boundary) bool {
		out = append(out, b)
		return true
	})
	return out
}

func boundaryIxs(bs []Boundary) []int {
	out := make([]int, len(bs))
	for i, b := range bs {
		out[i] = b.Ix
	}
	return out
}

func splitInclusiveChars(text string) []LineFragment {
	var out []LineFragment
	for ix, c := range text {
		out = append(out, LineFragmentText(text[ix:ix+utf8.RuneLen(c)]))
	}
	return out
}

func assertEmergencyWrapPreservesGraphemes(t *testing.T, wrapper *LineWrapper) {
	t.Helper()
	var failures []string
	for _, cluster := range []string{"é", "が", "👍🏽", "🇯🇵", "👩‍💻"} {
		clusterWidth := wrapper.charsWidth(cluster)
		first, _ := utf8.DecodeRuneInString(cluster)
		firstWidth := wrapper.widthForChar(first)
		for _, text := range []string{cluster, "x" + cluster + "y"} {
			graphemeBoundaries := append(graphemeIndices(text), len(text))
			for index := range lineWrapperBreakIndices(text) {
				if !slices.Contains(graphemeBoundaries, index) {
					t.Fatalf("UAX #14 boundary inside %q", text)
				}
			}
			fragments := []LineFragment{LineFragmentText(text)}
			fullWidth := wrapper.charsWidth(text)
			if got := collectBoundaries(wrapper.WrapLine(fragments, fullWidth+Px(1.), IndentAdjustmentNoIndent)); len(got) != 0 {
				t.Fatalf("unexpected boundaries %v for %q", got, text)
			}
			for _, wrapWidth := range []Pixels{
				firstWidth - Px(0.1),
				firstWidth,
				firstWidth + Px(0.1),
				clusterWidth - Px(0.1),
				clusterWidth,
				clusterWidth + Px(0.1),
			} {
				boundaries := boundaryIxs(collectBoundaries(wrapper.WrapLine(fragments, wrapWidth, IndentAdjustmentNoIndent)))
				splitBoundaries := boundaryIxs(collectBoundaries(wrapper.WrapLine(splitInclusiveChars(text), wrapWidth, IndentAdjustmentNoIndent)))
				if !slices.Equal(boundaries, splitBoundaries) {
					t.Fatalf("fragmentation changed %q: %v != %v", text, boundaries, splitBoundaries)
				}
				var invalid []int
				for _, index := range boundaries {
					if !slices.Contains(graphemeBoundaries, index) {
						invalid = append(invalid, index)
					}
				}
				if len(invalid) != 0 {
					failures = append(failures, fmt.Sprintf("text=%q, width=%v, cluster_width=%v, breaks=%v, inside_grapheme=%v", text, wrapWidth, clusterWidth, boundaries, invalid))
				}
			}
		}
	}
	if len(failures) != 0 {
		t.Fatal(strings.Join(failures, "\n"))
	}
}

func TestLineWrapper_EmergencyWrapPreservesGraphemes(t *testing.T) {
	assertEmergencyWrapPreservesGraphemes(t, buildWrapper())
}

func fillASCIIWidths(wrapper *LineWrapper, width Pixels) {
	for i := range wrapper.charWidths.ascii {
		wrapper.charWidths.ascii[i] = Some(width)
	}
}

func TestLineWrapper_EmergencyWrapExactBoundaries(t *testing.T) {
	wrapper := buildWrapper()
	fillASCIIWidths(wrapper, Px(1.))
	wrapper.charWidths.other['́'] = Px(1.)

	for _, tc := range []struct {
		text       string
		width      float32
		adjustment IndentAdjustment
		expected   []Boundary
	}{
		{"xéy", 1., IndentAdjustmentNoIndent, []Boundary{newBoundary(1, 0), newBoundary(4, 0)}},
		{"  xéyzéq", 5., IndentAdjustmentSameIndent, []Boundary{newBoundary(6, 2), newBoundary(8, 2)}},
		{"  xéyzéq", 5., IndentAdjustmentExtraColumns(1), []Boundary{newBoundary(6, 3), newBoundary(8, 3), newBoundary(11, 3)}},
	} {
		got := collectBoundaries(wrapper.WrapLine([]LineFragment{LineFragmentText(tc.text)}, Px(tc.width), tc.adjustment))
		if !slices.Equal(got, tc.expected) {
			t.Fatalf("%q, %v: got %v, want %v", tc.text, tc.adjustment, got, tc.expected)
		}
	}
}

func TestLineWrapper_EmergencyWrapElementsAndGraphemes(t *testing.T) {
	wrapper := buildWrapper()
	fillASCIIWidths(wrapper, Px(1.))
	wrapper.charWidths.other['́'] = Px(1.)

	for _, elementLength := range []int{1, 3, 32} {
		fragments := []LineFragment{
			LineFragmentText("xé"),
			LineFragmentElement(Px(4.), elementLength),
			LineFragmentText("éyz"),
		}
		got := collectBoundaries(wrapper.WrapLine(fragments, Px(2.), IndentAdjustmentNoIndent))
		want := []Boundary{
			newBoundary(1, 0),
			newBoundary(4, 0),
			newBoundary(4+elementLength, 0),
			newBoundary(7+elementLength, 0),
		}
		if !slices.Equal(got, want) {
			t.Fatalf("element_length=%d: got %v, want %v", elementLength, got, want)
		}
	}
}

func generateTestRuns(inputRunLen []int) []TextRun {
	runs := make([]TextRun, len(inputRunLen))
	for i, runLen := range inputRunLen {
		runs[i] = TextRun{
			Len: runLen,
			Font: Font{
				Family: "Dummy",
				Weight: DefaultFontWeight(),
				Style:  FontStyleNormal,
			},
		}
	}
	return runs
}

func assertWrap(t *testing.T, wrapper *LineWrapper, fragments []LineFragment, width Pixels, adjustment IndentAdjustment, want []Boundary) {
	t.Helper()
	got := collectBoundaries(wrapper.WrapLine(fragments, width, adjustment))
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLineWrapper_WrapLine(t *testing.T) {
	wrapper := buildWrapper()
	d := DefaultIndentAdjustment()
	b := newBoundary
	txt := LineFragmentText
	el := LineFragmentElement

	assertWrap(t, wrapper, []LineFragment{txt("aa bbb cccc ddddd eeee")}, Px(72.), d,
		[]Boundary{b(7, 0), b(12, 0), b(18, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("aaa aaaaaaaaaaaaaaaaaa")}, Px(72.0), d,
		[]Boundary{b(4, 0), b(11, 0), b(18, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("     aaaaaaa")}, Px(72.), d,
		[]Boundary{b(7, 5), b(9, 5), b(11, 5)})
	assertWrap(t, wrapper, []LineFragment{txt("                            ")}, Px(72.), d,
		[]Boundary{b(7, 0), b(14, 0), b(21, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("          aaaaaaaaaaaaaa")}, Px(72.), d,
		[]Boundary{b(7, 0), b(14, 3), b(18, 3), b(22, 3)})

	assertWrap(t, wrapper, []LineFragment{txt("aa bbb "), txt("cccc ddddd eeee")}, Px(72.), d,
		[]Boundary{b(7, 0), b(12, 0), b(18, 0)})

	assertWrap(t, wrapper, []LineFragment{
		txt("aa "),
		el(Px(20.), 1),
		txt(" bbb "),
		el(Px(30.), 1),
		txt(" cccc"),
	}, Px(72.), d, []Boundary{b(5, 0), b(9, 0), b(11, 0)})

	assertWrap(t, wrapper, []LineFragment{el(Px(50.), 1), txt(" aaaa bbbb cccc dddd")}, Px(72.), d,
		[]Boundary{b(2, 0), b(7, 0), b(12, 0), b(17, 0)})

	assertWrap(t, wrapper, []LineFragment{txt("short text "), el(Px(100.), 1), txt(" more text")}, Px(72.), d,
		[]Boundary{b(6, 0), b(11, 0), b(12, 0), b(18, 0)})

	assertWrap(t, wrapper, []LineFragment{txt("a b c‑d e")}, Px(72.0), d,
		[]Boundary{b(12, 0)})
}

func TestLineWrapper_WrapLineBreakOpportunities(t *testing.T) {
	wrapper := buildWrapper()
	for _, tc := range []struct {
		text, fittingPrefix string
		expectedIndex       int
	}{
		{"aaa required!", "aaa required", 4},
		{"aaa foo/bar", "aaa foo/", 8},
		{"aaa (hello)", "aaa (hello", 4},
		{"aaa αβγδε", "aaa αβγδ", 4},
		{"aaa a b", "aaa a ", 4},
		{"aaa 你好！", "aaa 你好", 7},
	} {
		wrapWidth := wrapper.charsWidth(tc.fittingPrefix)
		got := collectBoundaries(wrapper.WrapLine([]LineFragment{LineFragmentText(tc.text)}, wrapWidth, DefaultIndentAdjustment()))
		if want := []Boundary{newBoundary(tc.expectedIndex, 0)}; !slices.Equal(got, want) {
			t.Fatalf("%q: got %v, want %v", tc.text, got, want)
		}
	}
}

func TestLineWrapper_TruncateLineEnd(t *testing.T) {
	wrapper := buildWrapper()
	performTest := func(text, expected, ellipsis string) {
		t.Helper()
		dummyRuns := generateTestRuns([]int{len(text)})
		result, dummyRuns := wrapper.TruncateLine(SharedString(text), Px(220.), ellipsis, dummyRuns, TruncateFromEnd)
		if string(result) != expected {
			t.Fatalf("got %q, want %q", result, expected)
		}
		if dummyRuns[0].Len != len(result) {
			t.Fatalf("run len %d, want %d", dummyRuns[0].Len, len(result))
		}
	}
	performTest("aa bbb cccc ddddd eeee ffff gggg", "aa bbb cccc ddddd eeee", "")
	performTest("aa bbb cccc ddddd eeee ffff gggg", "aa bbb cccc ddddd eee…", "…")
	performTest("aa bbb cccc ddddd eeee ffff gggg", "aa bbb cccc dddd......", "......")
	performTest("aa bbb cccc 🦀🦀🦀🦀🦀 eeee ffff gggg", "aa bbb cccc 🦀🦀🦀🦀…", "…")
}

func TestLineWrapper_TruncateLineStart(t *testing.T) {
	wrapper := buildWrapper()
	performTest := func(text, expected, ellipsis string) {
		t.Helper()
		dummyRuns := generateTestRuns([]int{len(text)})
		result, dummyRuns := wrapper.TruncateLine(SharedString(text), Px(220.), ellipsis, dummyRuns, TruncateFromStart)
		if string(result) != expected {
			t.Fatalf("got %q, want %q", result, expected)
		}
		if dummyRuns[0].Len != len(result) {
			t.Fatalf("run len %d, want %d", dummyRuns[0].Len, len(result))
		}
	}
	performTest("aaaa bbbb cccc ddddd eeee fff gg", "cccc ddddd eeee fff gg", "")
	performTest("aaaa bbbb cccc ddddd eeee fff gg", "…ccc ddddd eeee fff gg", "…")
	performTest("aaaa bbbb cccc ddddd eeee fff gg", "......dddd eeee fff gg", "......")
	performTest("aaaa bbbb cccc 🦀🦀🦀🦀🦀 eeee fff gg", "…🦀🦀🦀🦀 eeee fff gg", "…")
}

func assertRunLens(t *testing.T, runs []TextRun, want []int) {
	t.Helper()
	for i := 0; i < len(runs) && i < len(want); i++ {
		if runs[i].Len != want[i] {
			t.Fatalf("run %d len %d, want %d", i, runs[i].Len, want[i])
		}
	}
}

func TestLineWrapper_TruncateMultipleRunsEnd(t *testing.T) {
	wrapper := buildWrapper()
	performTest := func(text, expected string, runLens, resultRunLen []int, lineWidth Pixels) {
		t.Helper()
		dummyRuns := generateTestRuns(runLens)
		result, dummyRuns := wrapper.TruncateLine(SharedString(text), lineWidth, "…", dummyRuns, TruncateFromEnd)
		if string(result) != expected {
			t.Fatalf("got %q, want %q", result, expected)
		}
		assertRunLens(t, dummyRuns, resultRunLen)
	}
	performTest("abcdefghijkl", "abcd…", []int{12}, []int{7}, Px(50.))
	performTest("abcdefghijkl", "abcdef…", []int{4, 4, 4}, []int{4, 5}, Px(70.))
	performTest("abcdefghijkl", "abcdefgh…", []int{4, 4, 4}, []int{4, 4, 3}, Px(90.))
}

func TestLineWrapper_TruncateMultipleRunsStart(t *testing.T) {
	wrapper := buildWrapper()
	performTest := func(text, expected string, runLens, resultRunLen []int, lineWidth Pixels) {
		t.Helper()
		dummyRuns := generateTestRuns(runLens)
		result, dummyRuns := wrapper.TruncateLine(SharedString(text), lineWidth, "…", dummyRuns, TruncateFromStart)
		if string(result) != expected {
			t.Fatalf("got %q, want %q", result, expected)
		}
		assertRunLens(t, dummyRuns, resultRunLen)
	}
	performTest("abcdefghijkl", "…ijkl", []int{12}, []int{7}, Px(50.))
	performTest("abcdefghijkl", "…ghijkl", []int{4, 4, 4}, []int{5, 4}, Px(70.))
	performTest("abcdefghijkl", "…efghijkl", []int{4, 4, 4}, []int{3, 4, 4}, Px(90.))
}

func TestLineWrapper_UpdateRunAfterTruncationEnd(t *testing.T) {
	performTest := func(result string, runLens, resultRunLens []int) {
		t.Helper()
		dummyRuns := generateTestRuns(runLens)
		updateRunsAfterTruncation(result, "…", &dummyRuns, TruncateFromEnd)
		assertRunLens(t, dummyRuns, resultRunLens)
	}
	performTest("abcd…", []int{12}, []int{7})
	performTest("abcdef…", []int{4, 4, 4}, []int{4, 5})
	performTest("abcdefgh…", []int{4, 4, 4}, []int{4, 4, 3})
}

func TestLineWrapper_WrapShapedLine(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("upstream gates this test on target_os = \"macos\"")
	}
	RunTest(t, func(tc *TestRun) {
		tc.Cx().Update(func(cx *App) {
			textSystem := NewWindowTextSystem(cx.TextSystem())

			normal := TextRun{Len: 0, Font: NewFont("Helvetica")}
			bold := TextRun{Len: 0, Font: NewFont("Helvetica").Bold()}
			withLen := func(run TextRun, n int) TextRun {
				run.Len = n
				return run
			}

			lines, err := textSystem.ShapeText(
				"aa bbb cccc ddddd eeee",
				Px(16.),
				[]TextRun{
					withLen(normal, 4),
					withLen(bold, 5),
					withLen(normal, 6),
					withLen(bold, 1),
					withLen(normal, 7),
				},
				Some(Px(72.)),
				None[int](),
			)
			if err != nil {
				t.Fatal(err)
			}

			want := []WrapBoundary{
				{RunIx: 0, GlyphIx: 7},
				{RunIx: 0, GlyphIx: 12},
				{RunIx: 0, GlyphIx: 18},
			}
			if got := lines[0].WrapBoundaries; !slices.Equal(got, want) {
				t.Fatalf("wrap boundaries = %v, want %v", got, want)
			}
		})
	})
}

func wrapCount(wrapper *LineWrapper, text SharedString, wrapWidth Pixels) int {
	return len(collectBoundaries(wrapper.WrapLine([]LineFragment{LineFragmentText(string(text))}, wrapWidth, DefaultIndentAdjustment())))
}

func TestLineWrapper_MultilineTruncationFitsWithinWrappedLines(t *testing.T) {
	wrapper := buildWrapper()
	text := "aa bbbbbb cccccc dddddd eeee ffff"
	wrapWidth := Px(72.)
	maxLines := 2

	runs := generateTestRuns([]int{len(text)})
	truncated, _ := wrapper.TruncateWrappedLine(SharedString(text), wrapWidth, maxLines, "…", runs, TruncateFromEnd)

	if n := wrapCount(wrapper, truncated, wrapWidth); n >= maxLines {
		t.Fatalf("Truncated text '%s' wraps into %d visual lines, expected at most %d", truncated, n+1, maxLines)
	}
	if !strings.HasSuffix(string(truncated), "…") {
		t.Fatalf("Truncated text '%s' should end with ellipsis", truncated)
	}
}

func TestLineWrapper_MultilineTruncationNoTruncationNeeded(t *testing.T) {
	wrapper := buildWrapper()
	text := "aa bbb cccccc"
	runs := generateTestRuns([]int{len(text)})
	result, _ := wrapper.TruncateWrappedLine(SharedString(text), Px(72.), 2, "…", runs, TruncateFromEnd)
	if string(result) != text {
		t.Fatalf("Text that fits should not be modified: %q", result)
	}
}

func TestLineWrapper_MultilineTruncationThreeLines(t *testing.T) {
	wrapper := buildWrapper()
	text := "aa bbb cccc ddddd eeee ffff gggg hhhh iiii jjjj"
	wrapWidth := Px(72.)
	maxLines := 3

	runs := generateTestRuns([]int{len(text)})
	truncated, _ := wrapper.TruncateWrappedLine(SharedString(text), wrapWidth, maxLines, "…", runs, TruncateFromEnd)

	if n := wrapCount(wrapper, truncated, wrapWidth); n >= maxLines {
		t.Fatalf("Truncated text '%s' wraps into %d visual lines, expected at most %d", truncated, n+1, maxLines)
	}
	if !strings.HasSuffix(string(truncated), "…") {
		t.Fatalf("Truncated text '%s' should end with ellipsis", truncated)
	}
}

func TestLineWrapper_MultilineTruncationPreservesBreaksAfterEmptyLines(t *testing.T) {
	wrapper := buildWrapper()
	for _, prefix := range []string{"", "\n", "\n\n", "hello\n\n"} {
		text := prefix + "aaa required! more words follow"
		runs := generateTestRuns([]int{len(text)})
		maxLines := strings.Count(prefix, "\n") + 2
		truncated, _ := wrapper.TruncateWrappedLine(SharedString(text), Px(72.), maxLines, "…", runs, TruncateFromEnd)
		if want := prefix + "aaa " + "required!"[:6] + "…"; string(truncated) != want {
			t.Fatalf("prefix: %q: got %q, want %q", prefix, truncated, want)
		}
	}
}

func TestLineWrapper_MultilineTruncationWithNewlines(t *testing.T) {
	wrapper := buildWrapper()
	text := "hello\nworld foo bar baz"
	wrapWidth := Px(72.)

	runs := generateTestRuns([]int{len(text)})
	truncated, _ := wrapper.TruncateWrappedLine(SharedString(text), wrapWidth, 2, "…", runs, TruncateFromEnd)

	parts := strings.SplitN(string(truncated), "\n", 2)
	if len(parts) != 2 {
		t.Fatalf("Newline should be preserved: '%s'", truncated)
	}
	if parts[0] != "hello" {
		t.Fatalf("first line %q, want hello", parts[0])
	}
	if secondLineWidth := wrapper.charsWidth(parts[1]); secondLineWidth > wrapWidth {
		t.Fatalf("Second line '%s' (%vpx) exceeds wrap_width (%vpx)", parts[1], secondLineWidth, wrapWidth)
	}
	if !strings.HasSuffix(string(truncated), "…") {
		t.Fatalf("Should end with ellipsis: '%s'", truncated)
	}
}

func TestLineWrapper_MultilineTruncationNewlineOnLastLine(t *testing.T) {
	wrapper := buildWrapper()
	text := "hello\nworld\nmore"
	runs := generateTestRuns([]int{len(text)})
	truncated, _ := wrapper.TruncateWrappedLine(SharedString(text), Px(72.), 2, "…", runs, TruncateFromEnd)

	parts := strings.SplitN(string(truncated), "\n", 2)
	if parts[0] != "hello" {
		t.Fatalf("first line %q, want hello", parts[0])
	}
	if !strings.HasSuffix(string(truncated), "…") {
		t.Fatalf("Should end with ellipsis since there's more content: '%s'", truncated)
	}
}

func TestLineWrapper_TruncateLineMiddle(t *testing.T) {
	wrapper := buildWrapper()

	shortText := "hello world"
	runs := generateTestRuns([]int{len(shortText)})
	result, resultRuns := wrapper.TruncateLine(SharedString(shortText), Px(10000.), "…", runs, TruncateFromMiddle)
	if string(result) != shortText || len(resultRuns) != 1 || resultRuns[0].Len != len(shortText) {
		t.Fatalf("short text: got %q %v", result, resultRuns)
	}

	longText := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	runs = generateTestRuns([]int{len(longText)})
	result, _ = wrapper.TruncateLine(SharedString(longText), Px(100.), "…", runs, TruncateFromMiddle)
	if !strings.ContainsRune(string(result), '…') {
		t.Fatalf("Middle-truncated result should contain '…', got: '%s'", result)
	}
	if utf8.RuneCountInString(string(result)) >= utf8.RuneCountInString(longText) {
		t.Fatal("Middle-truncated result should be shorter than original")
	}
	if result[0] != longText[0] {
		t.Fatal("Result should start with the same first character as original")
	}
	if result[len(result)-1] != longText[len(longText)-1] {
		t.Fatal("Result should end with the same last character as original")
	}

	text := "abcdef"
	runs = generateTestRuns([]int{len(text)})
	result, resultRuns = wrapper.TruncateLine(SharedString(text), Px(1.), "…", runs, TruncateFromMiddle)
	if string(result) != "…" || len(resultRuns) != 1 || resultRuns[0].Len != len("…") {
		t.Fatalf("degenerate: got %q %v", result, resultRuns)
	}

	multiRunText := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	runs = generateTestRuns([]int{20, 20, len(multiRunText) - 40})
	result, resultRuns = wrapper.TruncateLine(SharedString(multiRunText), Px(100.), "…", runs, TruncateFromMiddle)
	totalRunLen := 0
	for _, r := range resultRuns {
		totalRunLen += r.Len
	}
	if totalRunLen != len(result) {
		t.Fatalf("Sum of run lengths (%d) should equal result byte length (%d)", totalRunLen, len(result))
	}
}

func TestLineWrapper_MultilineTruncationTrailingNewline(t *testing.T) {
	wrapper := buildWrapper()
	text := "hello\nworld\n"
	runs := generateTestRuns([]int{len(text)})
	result, _ := wrapper.TruncateWrappedLine(SharedString(text), Px(72.), 2, "…", runs, TruncateFromEnd)
	if strings.HasSuffix(string(result), "…") {
		t.Fatalf("Trailing newline with no content should not add ellipsis: '%s'", result)
	}
}

func TestLineWrapper_MultilineTruncationNewlineFitsExactly(t *testing.T) {
	wrapper := buildWrapper()
	text := "hello\nworld"
	runs := generateTestRuns([]int{len(text)})
	result, _ := wrapper.TruncateWrappedLine(SharedString(text), Px(72.), 2, "…", runs, TruncateFromEnd)
	if string(result) != text {
		t.Fatalf("Text that fits exactly should not be modified: '%s'", result)
	}
}

func TestLineWrapper_ExtraColumnsOverflowGuard(t *testing.T) {
	wrapper := buildWrapper()
	spaceWidth := wrapper.widthForChar(' ')
	b := newBoundary
	txt := LineFragmentText

	text := "      ab cd ef gh"
	wrapWidth := spaceWidth * 10.0

	assertWrap(t, wrapper, []LineFragment{txt(text)}, wrapWidth, IndentAdjustmentExtraColumns(8),
		[]Boundary{b(9, 0)})
	assertWrap(t, wrapper, []LineFragment{txt(text)}, wrapWidth, IndentAdjustmentExtraColumns(2),
		[]Boundary{b(9, 8), b(11, 8), b(12, 8), b(14, 8), b(15, 8)})
	assertWrap(t, wrapper, []LineFragment{txt(" 🦀🦀🦀🦀🦀🦀🦀🦀")}, wrapWidth, IndentAdjustmentExtraColumns(8),
		[]Boundary{b(17, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("a abcdefghi")}, wrapWidth, IndentAdjustmentExtraColumns(1),
		[]Boundary{b(2, 1)})
	assertWrap(t, wrapper, []LineFragment{txt("a abcdefghi")}, wrapWidth, IndentAdjustmentExtraColumns(2),
		[]Boundary{b(2, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("a abcdefghij")}, wrapWidth, IndentAdjustmentExtraColumns(8),
		[]Boundary{b(2, 0)})
	assertWrap(t, wrapper, []LineFragment{
		txt("abcdefghijk "),
		LineFragmentElement(spaceWidth*8.0, 1),
		txt("z 12"),
	}, wrapWidth, IndentAdjustmentExtraColumns(8), []Boundary{b(10, 8), b(12, 0), b(15, 8)})
	assertWrap(t, wrapper, []LineFragment{txt("                    ")}, wrapWidth, IndentAdjustmentExtraColumns(8),
		[]Boundary{b(10, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("              ab cd ef gh")}, wrapWidth, IndentAdjustmentExtraColumns(2),
		[]Boundary{b(10, 0), b(20, 6), b(23, 6)})
	assertWrap(t, wrapper, []LineFragment{
		txt("aaaaaaaaaaaaaaaaaaa a"),
		LineFragmentElement(spaceWidth*10.0, 1),
	}, wrapWidth, IndentAdjustmentExtraColumns(2), []Boundary{b(10, 2), b(18, 2), b(21, 0)})
	assertWrap(t, wrapper, []LineFragment{txt("  aaaaaaaa")}, spaceWidth*4.0, IndentAdjustmentExtraColumns(^uint32(0)),
		[]Boundary{b(4, 0), b(8, 0)})
}
