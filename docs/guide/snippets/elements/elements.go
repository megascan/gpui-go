// Package elements holds the snippets of docs/guide/elements-and-styling.md.
package elements

import (
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/megascan/gpui-go"
)

func card(title string, body string) *gpui.Div {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap2().
		P(gpui.Px(16)).
		W(gpui.Px(280)).
		RoundedLg().
		Border1().
		BorderColor(gpui.Rgb(0x45475a)).
		Bg(gpui.Rgb(0x1e1e2e)).
		ShadowLg().
		Child(gpui.NewDiv().TextLg().FontWeight(gpui.FontWeightBold).ChildText(title)).
		Child(gpui.NewDiv().TextSm().TextColor(gpui.Rgb(0xa6adc8)).ChildText(body))
}

func toolbar() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		FlexRow().
		ItemsCenter().
		JustifyBetween().
		WFull().
		H(gpui.Px(40)).
		Px(gpui.Px(12)).
		Child(gpui.NewDiv().ChildText("Left")).
		Child(gpui.NewDiv().ChildText("Right"))
}

func lengths() *gpui.Div {
	return gpui.NewDiv().
		W(gpui.Relative(0.5)).
		H(gpui.Rems(2)).
		MinW(gpui.Px(120)).
		Mx(gpui.Auto())
}

func colors() *gpui.Div {
	accent := gpui.NewHsla(0.6, 0.8, 0.6, 1)
	return gpui.NewDiv().
		Bg(gpui.Rgb(0x313244)).
		TextColor(gpui.White()).
		BorderColor(accent.Opacity(0.5)).
		Child(gpui.NewDiv().Size8().Bg(gpui.NewRgba(0xff000080))).
		Child(gpui.NewDiv().Size8().Bg(gpui.LinearGradient(
			90,
			gpui.NewLinearColorStop(gpui.Red(), 0),
			gpui.NewLinearColorStop(gpui.Blue(), 1),
		)))
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func hoverBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x45475a))
}

func activeBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x585b70))
}

func button(id string, label string, onClick func(event *gpui.ClickEvent, window *gpui.Window, cx *gpui.App)) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Px3().
		Py1().
		RoundedMd().
		Bg(gpui.Rgb(0x313244)).
		Hover(hoverBg).
		Active(activeBg).
		CursorPointer().
		OnClick(onClick).
		ChildText(label)
}

func row(selected bool, subtitle gpui.Option[string]) *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap2().
		When(selected, func(this *gpui.Div) *gpui.Div {
			return this.Bg(gpui.Rgb(0x45475a))
		}).
		WhenSome(subtitle, func(this *gpui.Div, text string) *gpui.Div {
			return this.Child(gpui.NewDiv().TextXs().ChildText(text))
		})
}

func statefulRow(ix int, selected bool) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		Px2().
		ID(ix).
		When(selected, func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
			return this.Bg(gpui.Rgb(0x45475a))
		})
}

// Badge is a small labelled pill.
type Badge struct {
	Label string
	Color gpui.Hsla
}

// IntoAnyElement renders the badge as a component.
func (b Badge) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(b)
}

// Render renders the badge.
func (b Badge) Render(_ *gpui.Window, _ *gpui.App) gpui.IntoElement {
	return gpui.NewDiv().
		Px2().
		RoundedFull().
		TextXs().
		Bg(b.Color.Opacity(0.2)).
		TextColor(b.Color).
		ChildText(b.Label)
}

func badges() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap1().
		Child(Badge{Label: "new", Color: gpui.Green()}).
		Child(Badge{Label: "beta", Color: gpui.Yellow()})
}

//go:embed assets
var assetFS embed.FS

// Assets serves embedded files to Svg.Path and image asset paths.
type Assets struct{}

// Load returns the bytes of the asset at path.
func (Assets) Load(path string) ([]byte, bool, error) {
	data, err := fs.ReadFile(assetFS, path)
	if err != nil {
		return nil, false, fmt.Errorf("load asset %s: %w", path, err)
	}
	return data, true, nil
}

// List returns the names of the assets under path.
func (Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := fs.ReadDir(assetFS, path)
	if err != nil {
		return nil, fmt.Errorf("list assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(entry.Name()))
	}
	return names, nil
}

func media() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		Gap2().
		Child(gpui.NewSvg().Path("assets/check.svg").Size6().TextColor(gpui.Green())).
		Child(gpui.NewImg(gpui.ImageSourceFromPath("/usr/share/pixmaps/example.png")).Size16()).
		Child(gpui.NewImg(gpui.ImageSourceFromString("https://example.com/avatar.png")).Size16().RoundedFull())
}

func spinner() gpui.IntoElement {
	return gpui.WithAnimation(
		gpui.NewSvg().Path("assets/check.svg").Size6().TextColor(gpui.White()),
		"spinner",
		gpui.NewAnimation(time.Second).Repeat(),
		func(svg *gpui.Svg, delta float32) *gpui.Svg {
			return svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))
		},
	)
}
