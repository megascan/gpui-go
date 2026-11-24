package gpui

import (
	"errors"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

func TestTextSystemFontResolutionTests_CachesResolvedFontFallbacks(t *testing.T) {
	textSystem := NewTextSystem(NewNoopTextSystem())
	requested := NewFont("Missing Font").Bold().Italic()
	missing := cachedFont{kind: cachedFontMissing, err: errors.New("missing font")}
	func() {
		textSystem.fontIDsByFontMu.Lock()
		defer textSystem.fontIDsByFontMu.Unlock()
		textSystem.fontIDsByFont[requested] = missing
		for _, fallback := range textSystem.fallbackFontStack {
			textSystem.fontIDsByFont[fallback] = missing
		}
		textSystem.fontIDsByFont[NewFont("Arial")] = cachedFont{kind: cachedFontDirect, fontID: FontID(42)}
	}()

	if got := textSystem.ResolveFont(requested); got != FontID(42) {
		t.Fatalf("ResolveFont = %v, want 42", got)
	}
	if got, ok := textSystem.GetFontForID(FontID(42)); !ok || got != NewFont("Arial") {
		t.Fatalf("GetFontForID = %+v, %v, want Arial", got, ok)
	}

	textSystem.fallbackFontStack = textSystem.fallbackFontStack[:0]
	if got := textSystem.ResolveFont(requested); got != FontID(42) {
		t.Fatalf("ResolveFont after clearing fallbacks = %v, want 42", got)
	}
}

func TestTextSystemLineWrapperPoolTests_PooledLineWrappersDoNotRetainTextSystem(t *testing.T) {
	t.Skip("blocked:M0-line-wrapper: needs LineWrapper.WrapLine and LineFragmentText")
}

func nowOrNever[T any](f scheduler.Future[T]) (T, bool) {
	out, ok := f.Poll(scheduler.NewContext(scheduler.WakerFn(func() {})))
	if !ok {
		if c, isCanceler := f.(interface{ Cancel() }); isCanceler {
			c.Cancel()
		}
	}
	return out, ok
}

type recvResult struct {
	glyphs []MissingGlyph
	err    error
}

func recvNowOrNever(receiver *MissingGlyphReceiver) (recvResult, bool) {
	return nowOrNever(scheduler.Async(func(s *scheduler.Scope) recvResult {
		glyphs, err := receiver.Recv(s)
		return recvResult{glyphs: glyphs, err: err}
	}))
}

func TestTextSystemMissingGlyphTests_BoundsRetainedMissingGlyphs(t *testing.T) {
	reporter, receiver := missingGlyphChannel()
	defer receiver.Close()
	glyphs := make([]MissingGlyph, 0, maxReportedMissingGlyphs)
	for index := range maxReportedMissingGlyphs {
		glyphs = append(glyphs, NewMissingGlyph(SharedString(strconv.Itoa(index)), FallbackFontClassProportional))
	}
	reporter.Report(glyphs)
	if result, ok := recvNowOrNever(receiver); !ok || result.err != nil {
		t.Fatalf("first recv = %+v, %v", result.err, ok)
	}

	newest := NewMissingGlyph("newest", FallbackFontClassMonospace)
	reporter.Report([]MissingGlyph{newest})
	if result, ok := recvNowOrNever(receiver); !ok || result.err != nil {
		t.Fatalf("second recv = %+v, %v", result.err, ok)
	}

	state := &receiver.state
	if len(state.reported) != maxReportedMissingGlyphs {
		t.Fatalf("reported len = %d", len(state.reported))
	}
	if len(state.reportedOrder) != maxReportedMissingGlyphs {
		t.Fatalf("reportedOrder len = %d", len(state.reportedOrder))
	}
	if _, ok := state.reported[newest]; !ok {
		t.Fatalf("newest not retained")
	}
}

func TestTextSystemMissingGlyphTests_DroppingReceiverClosesAndClearsReports(t *testing.T) {
	reporter, receiver := missingGlyphChannel()
	reporter.Report([]MissingGlyph{testMissingGlyph("missing")})

	rx := receiver.receiver
	receiver.Close()

	if !reporter.sender.IsClosed() {
		t.Fatalf("sender is not closed")
	}
	if _, ok, _ := rx.TryRecv(); ok {
		t.Fatalf("sender is not empty")
	}
}

func missingGlyphChannel() (*missingGlyphReporter, *MissingGlyphReceiver) {
	sender, receiver := scheduler.Chan[queuedMissingGlyph](maxReportedMissingGlyphs)
	generation := &atomic.Uint64{}
	reporter := &missingGlyphReporter{generation: generation, sender: sender}
	rx := &MissingGlyphReceiver{state: newMissingGlyphState(), generation: generation, receiver: receiver}
	return reporter, rx
}

func testMissingGlyph(grapheme string) MissingGlyph {
	return NewMissingGlyph(SharedString(grapheme), FallbackFontClassProportional)
}
