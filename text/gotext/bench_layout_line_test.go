package gotext

import (
	"strings"
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func codeText() string {
	return strings.ReplaceAll(strings.Repeat(strings.Join([]string{
		"    fn compute_run_spans(\n",
		"        text: &str,\n",
		"        run_offset: usize,\n",
		"        run_len: usize,\n",
		"        primary: FontId,\n",
		"        fallback_chain: &[(FontId, SharedString)],\n",
		"        covers: &impl Fn(FontId, char) -> bool,\n",
		"    ) -> SmallVec<[RunSpan; 4]> {\n",
		"        let mut spans = SmallVec::new();\n",
		"        let run_end = run_offset + run_len;\n",
		"        if run_end <= run_offset { return spans; }\n",
		"        let run_text = &text[run_offset..run_end];\n",
		"        let mut span_start = run_offset;\n",
		"        let mut span_slot: Option<usize> = None;\n",
		"        for (ch_idx, ch) in run_text.char_indices() {\n",
		"            let abs = run_offset + ch_idx;\n",
		"            let next = pick_covering_slot(ch, span_slot, primary, fallback_chain, covers);\n",
		"            if next == span_slot { continue; }\n",
		"            if abs > span_start {\n",
		"                spans.push(RunSpan { start: span_start, end: abs, slot: span_slot });\n",
		"            }\n",
		"            span_start = abs;\n",
		"            span_slot = next;\n",
		"        }\n",
		"        spans\n",
		"    }\n",
	}, ""), 8), "\n", " ")
}

type layoutLineBench struct {
	system             *CosmicTextSystem
	text               string
	textMixedDirection string
	runsNoFallback     []gpui.FontRun
	runsWithFallback   []gpui.FontRun
	runsMixedDirection []gpui.FontRun
}

func newLayoutLineBench(b *testing.B) layoutLineBench {
	b.Helper()
	system := NewTextSystemWithoutSystemFonts("Lilex")
	if err := system.AddFonts([][]byte{lilex, ibmPlex}); err != nil {
		b.Fatal(err)
	}
	fontIDNoFallback, err := system.FontID(gpui.NewFont("Lilex"))
	if err != nil {
		b.Fatal(err)
	}
	withFallback := gpui.NewFont("Lilex")
	withFallback.Fallbacks = gpui.Some(gpui.FontFallbacksFromFonts([]string{"IBM Plex Sans"}))
	fontIDWithFallback, err := system.FontID(withFallback)
	if err != nil {
		b.Fatal(err)
	}
	text := codeText()
	textMixedDirection := text + "\u001cאב"
	if strings.Contains(text, "\n") {
		b.Fatal("fast-path corpus must contain no separator")
	}
	return layoutLineBench{
		system:             system,
		text:               text,
		textMixedDirection: textMixedDirection,
		runsNoFallback:     []gpui.FontRun{{Len: len(text), FontID: fontIDNoFallback}},
		runsWithFallback:   []gpui.FontRun{{Len: len(text), FontID: fontIDWithFallback}},
		runsMixedDirection: []gpui.FontRun{{Len: len(textMixedDirection), FontID: fontIDNoFallback}},
	}
}

func BenchmarkLayoutLine_NoFallback(b *testing.B) {
	bench := newLayoutLineBench(b)
	for b.Loop() {
		bench.system.LayoutLine(bench.text, gpui.Px(14.0), bench.runsNoFallback)
	}
}

func BenchmarkLayoutLine_WithFallbackAscii(b *testing.B) {
	bench := newLayoutLineBench(b)
	for b.Loop() {
		bench.system.LayoutLine(bench.text, gpui.Px(14.0), bench.runsWithFallback)
	}
}

func BenchmarkLayoutLine_MixedDirectionParagraphs(b *testing.B) {
	bench := newLayoutLineBench(b)
	for b.Loop() {
		bench.system.LayoutLine(bench.textMixedDirection, gpui.Px(14.0), bench.runsMixedDirection)
	}
}
