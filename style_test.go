package gpui

import (
	"slices"
	"testing"
)

func TestStyle_BasicHighlightStyleCombination(t *testing.T) {
	styleA := HighlightStyle{}
	styleB := HighlightStyle{}
	styleA = styleA.Highlight(styleB)
	if styleA != (HighlightStyle{}) {
		t.Fatalf("Combining empty styles should not produce a non-empty style.")
	}

	styleB = HighlightStyle{
		Color:           Some(Red()),
		Strikethrough:   Some(StrikethroughStyle{Thickness: Px(2.), Color: Some(Blue())}),
		FadeOut:         Some[float32](0.),
		FontStyle:       Some(FontStyleItalic),
		FontWeight:      Some(FontWeight(300.)),
		BackgroundColor: Some(Yellow()),
		Underline:       Some(UnderlineStyle{Thickness: Px(2.), Color: Some(Red()), Wavy: true}),
	}
	expectedStyle := styleB

	styleA = styleA.Highlight(styleB)
	if styleA != expectedStyle {
		t.Fatalf("Blending an empty style with another style should return the other style\n got %+v\nwant %+v", styleA, expectedStyle)
	}

	styleB = styleB.Highlight(HighlightStyle{})
	if styleB != expectedStyle {
		t.Fatalf("Blending a style with an empty style should not change the style.\n got %+v\nwant %+v", styleB, expectedStyle)
	}

	styleC := expectedStyle

	styleD := HighlightStyle{
		Color:           Some(Blue().Alpha(0.7)),
		Strikethrough:   Some(StrikethroughStyle{Thickness: Px(4.), Color: Some(Red())}),
		FadeOut:         Some[float32](0.),
		FontStyle:       Some(FontStyleOblique),
		FontWeight:      Some(FontWeight(800.)),
		BackgroundColor: Some(Green()),
		Underline:       Some(UnderlineStyle{Thickness: Px(4.), Color: None[Hsla](), Wavy: false}),
	}

	expectedStyle = HighlightStyle{
		Color:           Some(Red().Blend(Blue().Alpha(0.7))),
		Strikethrough:   Some(StrikethroughStyle{Thickness: Px(4.), Color: Some(Red())}),
		FadeOut:         Some[float32](0.),
		FontStyle:       Some(FontStyleOblique),
		FontWeight:      Some(FontWeight(800.)),
		BackgroundColor: Some(Green()),
		Underline:       Some(UnderlineStyle{Thickness: Px(4.), Color: None[Hsla](), Wavy: false}),
	}

	styleC = styleC.Highlight(styleD)
	if styleC != expectedStyle {
		t.Fatalf("Blending styles should blend properties where possible and override all others\n got %+v\nwant %+v", styleC, expectedStyle)
	}
}

type rangedHighlight struct {
	r Range
	h HighlightStyle
}

func rangedHighlights(items ...rangedHighlight) func(func(Range, HighlightStyle) bool) {
	return func(yield func(Range, HighlightStyle) bool) {
		for _, item := range items {
			if !yield(item.r, item.h) {
				return
			}
		}
	}
}

func TestStyle_CombineHighlights(t *testing.T) {
	var got []rangedHighlight
	for r, h := range CombineHighlights(
		rangedHighlights(
			rangedHighlight{Range{0, 5}, Green().ToHighlightStyle()},
			rangedHighlight{Range{4, 10}, FontWeightBold.ToHighlightStyle()},
			rangedHighlight{Range{15, 20}, Yellow().ToHighlightStyle()},
		),
		rangedHighlights(
			rangedHighlight{Range{2, 6}, FontStyleItalic.ToHighlightStyle()},
			rangedHighlight{Range{1, 3}, Blue().ToHighlightStyle()},
			rangedHighlight{Range{21, 23}, Red().ToHighlightStyle()},
		),
	) {
		got = append(got, rangedHighlight{r, h})
	}
	want := []rangedHighlight{
		{Range{0, 1}, HighlightStyle{Color: Some(Green())}},
		{Range{1, 2}, HighlightStyle{Color: Some(Blue())}},
		{Range{2, 3}, HighlightStyle{Color: Some(Blue()), FontStyle: Some(FontStyleItalic)}},
		{Range{3, 4}, HighlightStyle{Color: Some(Green()), FontStyle: Some(FontStyleItalic)}},
		{Range{4, 5}, HighlightStyle{Color: Some(Green()), FontWeight: Some(FontWeightBold), FontStyle: Some(FontStyleItalic)}},
		{Range{5, 6}, HighlightStyle{FontWeight: Some(FontWeightBold), FontStyle: Some(FontStyleItalic)}},
		{Range{6, 10}, HighlightStyle{FontWeight: Some(FontWeightBold)}},
		{Range{15, 20}, HighlightStyle{Color: Some(Yellow())}},
		{Range{21, 23}, HighlightStyle{Color: Some(Red())}},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestStyle_TextStyleRefinement(t *testing.T) {
	style := DefaultStyle()
	style.Refine(newTestStyled().TextSize(Px(20.0)))
	style.Refine(newTestStyled().FontWeight(FontWeightSemibold))

	if got := style.TextStyle().FontSize; got != Some(Px(20.0).ToAbsoluteLength()) {
		t.Fatalf("font_size = %v", got)
	}
	if got := style.TextStyle().FontWeight; got != Some(FontWeightSemibold) {
		t.Fatalf("font_weight = %v", got)
	}
}

func newTestStyled() *Styled[*StyleRefinement] {
	var s Styled[*StyleRefinement]
	BindStyled(&s, &StyleRefinement{})
	return &s
}
