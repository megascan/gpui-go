package main

import (
	"fmt"
	"slices"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// TextContext holds the base type settings of the specimens.
type TextContext struct {
	FontSize   float32
	LineHeight float32
	TypeScale  float32
}

// DefaultTextContext returns the default TextContext.
func DefaultTextContext() TextContext {
	return TextContext{FontSize: 16.0, LineHeight: 1.3, TypeScale: 1.33}
}

// TextContextGetGlobal returns the global TextContext.
func TextContextGetGlobal(cx *gpui.App) *TextContext {
	return gpui.Global[GlobalTextContext](cx).TextContext
}

// GlobalTextContext is the global wrapper of the shared TextContext.
type GlobalTextContext struct {
	*TextContext
}

func textContext(cx *gpui.App) *TextContext {
	return gpui.Global[GlobalTextContext](cx).TextContext
}

// SpecimenTheme is the foreground and background of a specimen.
type SpecimenTheme struct {
	Bg gpui.Hsla
	Fg gpui.Hsla
}

// DefaultSpecimenTheme returns black on white.
func DefaultSpecimenTheme() SpecimenTheme {
	return SpecimenTheme{Bg: gpui.White(), Fg: gpui.Black()}
}

// Invert swaps the colors.
func (t SpecimenTheme) Invert() SpecimenTheme {
	return SpecimenTheme{Bg: t.Fg, Fg: t.Bg}
}

// Specimen renders a pangram at a scale of the base type size.
type Specimen struct {
	id        gpui.ElementID
	scale     float32
	textStyle gpui.Option[gpui.TextStyle]
	str       gpui.SharedString
	invert    bool
}

// NewSpecimen creates a specimen with the given id.
func NewSpecimen(id int) Specimen {
	return Specimen{
		id:    gpui.ElementIDName(gpui.SharedString(fmt.Sprintf("specimen-%d", id))),
		scale: 1.0,
		str:   "The quick brown fox jumps over the lazy dog",
	}
}

// Invert toggles the inverted theme.
func (s Specimen) Invert() Specimen {
	s.invert = !s.invert
	return s
}

// Scale sets the scale.
func (s Specimen) Scale(scale float32) Specimen {
	s.scale = scale
	return s
}

// IntoAnyElement renders the specimen as a component.
func (s Specimen) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(s)
}

// Render renders the specimen.
func (s Specimen) Render(window *gpui.Window, cx *gpui.App) gpui.IntoElement {
	remSize := window.RemSize()
	scale := s.scale
	globalStyle := textContext(cx)

	fontSize := globalStyle.FontSize
	lineHeight := globalStyle.LineHeight

	if styleOverride, ok := s.textStyle.Get(); ok {
		fontSize = float32(styleOverride.FontSize.ToPixels(remSize))
		if absolute, ok := styleOverride.LineHeight.AsAbsolute(); ok {
			if rems, ok := absolute.AsRems(); ok {
				lineHeight = float32(rems.ToPixels(remSize))
			} else if pixels, ok := absolute.AsPixels(); ok {
				lineHeight = float32(pixels)
			}
		} else if fraction, ok := styleOverride.LineHeight.AsFraction(); ok {
			lineHeight = fraction
		}
	}

	theme := DefaultSpecimenTheme()
	if s.invert {
		theme = theme.Invert()
	}

	return gpui.NewDiv().
		ID(s.id).
		Bg(theme.Bg).
		TextColor(theme.Fg).
		TextSize(gpui.Px(fontSize * scale)).
		LineHeight(gpui.Relative(lineHeight)).
		P(gpui.Px(10.0)).
		Child(s.str)
}

// CharacterGrid renders a grid of characters across scripts.
type CharacterGrid struct {
	scale     float32
	invert    bool
	textStyle gpui.Option[gpui.TextStyle]
}

// NewCharacterGrid creates a character grid.
func NewCharacterGrid() CharacterGrid {
	return CharacterGrid{scale: 1.0}
}

// Scale sets the scale.
func (g CharacterGrid) Scale(scale float32) CharacterGrid {
	g.scale = scale
	return g
}

// IntoAnyElement renders the grid as a component.
func (g CharacterGrid) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(g)
}

// Render renders the grid.
func (g CharacterGrid) Render(_ *gpui.Window, _ *gpui.App) gpui.IntoElement {
	theme := DefaultSpecimenTheme()
	if g.invert {
		theme = theme.Invert()
	}

	characters := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "A", "B", "C", "D", "E", "F", "G",
		"H", "I", "J", "K", "L", "M", "N", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y",
		"Z", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "p", "q",
		"r", "s", "t", "u", "v", "w", "x", "y", "z", "ẞ", "ſ", "ß", "ð", "Þ", "þ", "α", "β",
		"Γ", "γ", "Δ", "δ", "η", "θ", "ι", "κ", "Λ", "λ", "μ", "ν", "ξ", "π", "τ", "υ", "φ",
		"χ", "ψ", "∂", "а", "в", "Ж", "ж", "З", "з", "К", "к", "л", "м", "Н", "н", "Р", "р",
		"У", "у", "ф", "ч", "ь", "ы", "Э", "э", "Я", "я", "ij", "öẋ", ".,", "⣝⣑", "~", "*",
		"_", "^", "`", "'", "(", "{", "«", "#", "&", "@", "$", "¢", "%", "|", "?", "¶", "µ",
		"❮", "<=", "!=", "==", "--", "++", "=>", "->", "🏀", "🎊", "😍", "❤️", "👍", "👎",
	}

	columns := 20
	rows := (len(characters) + columns - 1) / columns

	gridRows := make([]gpui.IntoElement, 0, rows)
	for rowIdx := range rows {
		startIdx := rowIdx * columns
		endIdx := min(startIdx+columns, len(characters))

		cells := make([]gpui.IntoElement, 0, endIdx-startIdx)
		for i := startIdx; i < endIdx; i++ {
			cells = append(cells, gpui.NewDiv().
				TextCenter().
				Size(gpui.Px(62.)).
				Bg(theme.Bg).
				TextColor(theme.Fg).
				TextSize(gpui.Px(24.0)).
				LineHeight(gpui.Relative(1.0)).
				ChildText(characters[i]))
		}
		gridRows = append(gridRows, gpui.NewDiv().
			WFull().
			Flex().
			FlexRow().
			Children(cells...).
			When(endIdx-startIdx < columns, func(d *gpui.Div) *gpui.Div {
				fill := make([]gpui.IntoElement, columns-(endIdx-startIdx))
				for i := range fill {
					fill[i] = gpui.NewDiv().Flex1()
				}
				return d.Children(fill...)
			}))
	}

	return gpui.NewDiv().P4().Gap2().Flex().FlexCol().Children(gridRows...)
}

// TextExample is the typography specimen view.
type TextExample struct {
	nextID     int
	fontFamily gpui.SharedString
}

func (t *TextExample) NextID() int {
	t.nextID++
	return t.nextID
}

func refine(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var styled gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&styled, style)
	return &styled
}

func button(text string, cx *gpui.Context[TextExample], onClick func(this *TextExample, cx *gpui.Context[TextExample])) gpui.IntoElement {
	return gpui.NewDiv().
		ID(text).
		FlexNone().
		ChildText(text).
		Bg(gpui.Black()).
		TextColor(gpui.White()).
		Active(func(this *gpui.StyleRefinement) *gpui.StyleRefinement { return refine(this).Opacity(0.8) }).
		Px3().
		Py1().
		OnClick(cx.Listener(func(this *TextExample, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[TextExample]) {
			onClick(this, cx)
		}))
}

var fontFamilies = [5]string{
	".ZedMono",
	".SystemUIFont",
	"Menlo",
	"Monaco",
	"Courier New",
}

// Render renders the specimen page.
func (t *TextExample) Render(_ *gpui.Window, cx *gpui.Context[TextExample]) gpui.IntoElement {
	tcx := textContext(cx.App)
	colors := *cx.DefaultColors()

	fontSize := tcx.FontSize
	lineHeight := tcx.LineHeight
	typeScale := tcx.TypeScale

	stepDown2 := 1.0 / (typeScale * typeScale)
	stepDown1 := 1.0 / typeScale
	base := float32(1.0)
	stepUp1 := base * typeScale
	stepUp2 := stepUp1 * typeScale
	stepUp3 := stepUp2 * typeScale
	stepUp4 := stepUp3 * typeScale
	stepUp5 := stepUp4 * typeScale
	stepUp6 := stepUp5 * typeScale

	return gpui.NewDiv().
		FontFamily(string(t.fontFamily)).
		SizeFull().
		Child(
			gpui.NewDiv().
				Bg(gpui.White()).
				BorderB1().
				BorderColor(gpui.Black()).
				P3().
				Flex().
				Child(button(string(t.fontFamily), cx, func(this *TextExample, cx *gpui.Context[TextExample]) {
					newFamily := fontFamilies[0]
					if idx := slices.Index(fontFamilies[:], string(this.fontFamily)); idx >= 0 {
						newFamily = fontFamilies[(idx+1)%len(fontFamilies)]
					}
					this.fontFamily = gpui.SharedString(newFamily)
					cx.Notify()
				})),
		).
		Child(
			gpui.NewDiv().
				ID("text-example").
				OverflowYScroll().
				OverflowXHidden().
				Bg(gpui.Rgb(0xffffff)).
				SizeFull().
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						P3().
						TextSize(gpui.Px(fontSize)).
						LineHeight(gpui.Relative(lineHeight)).
						Children(
							gpui.NewDiv().ChildText("CJK: 中文「日本語」한글"),
							gpui.NewDiv().ChildText("Kana: が / が; Hangul: 각 / 각"),
							gpui.NewDiv().ChildText("Emoji: 😀 ❤️ ❤︎ 👍🏽 🇯🇵 1️⃣ 👩‍💻 👨‍👩‍👧‍👦"),
						),
				).
				Child(gpui.NewDiv().Child(NewCharacterGrid().Scale(base))).
				Child(
					gpui.NewDiv().
						Child(NewSpecimen(t.NextID()).Scale(stepDown2)).
						Child(NewSpecimen(t.NextID()).Scale(stepDown2).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepDown1)).
						Child(NewSpecimen(t.NextID()).Scale(stepDown1).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(base)).
						Child(NewSpecimen(t.NextID()).Scale(base).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp1)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp1).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp2)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp2).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp3)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp3).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp4)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp4).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp5)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp5).Invert()).
						Child(NewSpecimen(t.NextID()).Scale(stepUp6)).
						Child(NewSpecimen(t.NextID()).Scale(stepUp6).Invert()),
				),
		).
		Child(gpui.NewDiv().W(gpui.Px(240.)).HFull().Bg(colors.Container))
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.SetMenus([]gpui.Menu{{
			Name:     "GPUI Typography",
			Disabled: false,
			Items:    nil,
		}})

		fonts := [][]byte{support.Font("Lilex-Regular.ttf")}
		_ = cx.TextSystem().AddFonts(fonts)

		cx.InitColors()
		defaultContext := DefaultTextContext()
		cx.SetGlobal(GlobalTextContext{TextContext: &defaultContext})

		options := gpui.DefaultWindowOptions()
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{
			Title: gpui.Some[gpui.SharedString]("GPUI Typography"),
		})
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.NewBounds(
			gpui.NewPoint(gpui.Px(0.0), gpui.Px(0.0)),
			gpui.NewSize(gpui.Px(920.), gpui.Px(720.)),
		)))
		window, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[TextExample] {
			return cx.New(func(_ *gpui.Context[TextExample]) *TextExample {
				return &TextExample{nextID: 0, fontFamily: ".ZedMono"}
			})
		})
		if err != nil {
			panic(err)
		}

		if err := window.Update(cx, func(_ *TextExample, _ *gpui.Window, cx *gpui.Context[TextExample]) {
			cx.Activate(true)
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
