package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the text_wrapper example view.
type HelloWorld struct{}

const text = "The longest word 你好世界这段是中文，こんにちはこの段落は日本語です in any of the major " +
	"English language dictionaries is pneumonoultramicroscopicsilicovolcanoconiosis, a word that " +
	"refers to a lung disease contracted from the inhalation of very fine silica particles, " +
	"a url https://github.com/zed-industries/zed/pull/35724?query=foo&bar=2, " +
	"specifically from a volcano; medically, it is the same as silicosis."

// Render renders the view.
func (*HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("page").
		SizeFull().
		Flex().
		FlexCol().
		P2().
		Gap2().
		Bg(gpui.White()).
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				FlexShrink0().
				Gap2().
				Child(
					gpui.NewDiv().
						Flex().
						Border1().
						BorderColor(gpui.Red()).
						TextEllipsis().
						ChildText("longer text in flex 1"),
				).
				Child(
					gpui.NewDiv().
						Flex().
						Border1().
						BorderColor(gpui.Red()).
						TextEllipsis().
						ChildText("short flex"),
				).
				Child(
					gpui.NewDiv().
						OverflowHidden().
						Border1().
						BorderColor(gpui.Red()).
						TextEllipsis().
						WFull().
						ChildText("A short text in normal div"),
				),
		).
		Child(
			gpui.NewDiv().
				FlexShrink0().
				TextXl().
				Truncate().
				Border1().
				BorderColor(gpui.Blue()).
				ChildText("ELLIPSIS: " + text),
		).
		Child(
			gpui.NewDiv().
				FlexShrink0().
				TextXl().
				OverflowHidden().
				TextEllipsis().
				LineClamp(2).
				Border1().
				BorderColor(gpui.Blue()).
				ChildText("ELLIPSIS 2 lines: " + text),
		).
		Child(
			gpui.NewDiv().
				FlexShrink0().
				TextXl().
				OverflowHidden().
				TextOverflow(gpui.TextOverflowTruncate("")).
				Border1().
				BorderColor(gpui.Green()).
				ChildText("TRUNCATE: " + text),
		).
		Child(
			gpui.NewDiv().
				FlexShrink0().
				TextXl().
				OverflowHidden().
				TextOverflow(gpui.TextOverflowTruncate("")).
				LineClamp(3).
				Border1().
				BorderColor(gpui.Green()).
				ChildText("TRUNCATE 3 lines: " + text),
		).
		Child(
			gpui.NewDiv().
				FlexShrink0().
				TextXl().
				WhitespaceNowrap().
				OverflowHidden().
				Border1().
				BorderColor(gpui.Black()).
				ChildText("NOWRAP: " + text),
		).
		Child(gpui.NewDiv().TextXl().WFull().ChildText(text))
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
