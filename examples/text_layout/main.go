package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the text_layout example view.
type HelloWorld struct{}

func highlights(yield func(gpui.Range, gpui.HighlightStyle) bool) {
	if !yield(gpui.Range{Start: 0, End: 1}, gpui.FontWeightExtraBold.ToHighlightStyle()) {
		return
	}
	yield(gpui.Range{Start: 2, End: 3}, gpui.FontStyleItalic.ToHighlightStyle())
}

// Render renders the view.
func (*HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		Bg(gpui.White()).
		Flex().
		FlexCol().
		Gap2().
		P4().
		SizeFull().
		Child(gpui.NewDiv().ChildText("Text left")).
		Child(gpui.NewDiv().TextCenter().ChildText("Text center")).
		Child(gpui.NewDiv().TextRight().ChildText("Text right")).
		Child(gpui.NewDiv().TextDecoration1().ChildText("Text left (underline)")).
		Child(
			gpui.NewDiv().
				TextCenter().
				TextDecoration1().
				ChildText("Text center (underline)"),
		).
		Child(
			gpui.NewDiv().
				TextRight().
				TextDecoration1().
				ChildText("Text right (underline)"),
		).
		Child(gpui.NewDiv().LineThrough().ChildText("Text left (line_through)")).
		Child(
			gpui.NewDiv().
				TextCenter().
				LineThrough().
				ChildText("Text center (line_through)"),
		).
		Child(
			gpui.NewDiv().
				TextRight().
				LineThrough().
				ChildText("Text right (line_through)"),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				JustifyBetween().
				Child(
					gpui.NewDiv().
						W(gpui.Px(400.)).
						Border1().
						BorderColor(gpui.Blue()).
						P1().
						WhitespaceNowrap().
						OverflowHidden().
						TextCenter().
						ChildText("A long non-wrapping text align center"),
				).
				Child(
					gpui.NewDiv().
						W32().
						Border1().
						BorderColor(gpui.Blue()).
						P1().
						WhitespaceNowrap().
						OverflowHidden().
						TextRight().
						ChildText("100%"),
				),
		).
		Child(gpui.NewDiv().Flex().Gap2().JustifyBetween().Child(
			gpui.NewStyledText("ABCD").WithHighlights(highlights),
		))
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(800.0), gpui.Px(600.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(_ *gpui.Context[HelloWorld]) *HelloWorld { return &HelloWorld{} })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
