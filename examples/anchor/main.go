package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// AnchorDemo shows anchored popovers for each Anchor.
type AnchorDemo struct {
	hoveredButton gpui.Option[int]
}

// ButtonDemo is one button of the demo grid.
type ButtonDemo struct {
	label  gpui.SharedString
	corner gpui.Option[gpui.Anchor]
}

func resolvedPosition(corner gpui.Anchor, buttonSize gpui.Size[gpui.Pixels]) gpui.Point[gpui.Pixels] {
	offset := gpui.Point[gpui.Pixels]{
		X: gpui.Px(0.),
		Y: -buttonSize.Height,
	}

	var p gpui.Point[gpui.Pixels]
	switch corner.OtherSideAlong(gpui.AxisVertical) {
	case gpui.AnchorTopLeft:
		p = gpui.NewPoint(gpui.Px(0.0), gpui.Px(0.0))
	case gpui.AnchorTopCenter:
		p = gpui.NewPoint(buttonSize.Width.Half(), gpui.Px(0.0))
	case gpui.AnchorTopRight:
		p = gpui.NewPoint(buttonSize.Width, gpui.Px(0.0))
	case gpui.AnchorLeftCenter:
		p = gpui.NewPoint(buttonSize.Width, buttonSize.Height.Half())
	case gpui.AnchorRightCenter:
		p = gpui.NewPoint(gpui.Px(0.), buttonSize.Height.Half())
	case gpui.AnchorBottomLeft:
		p = gpui.NewPoint(gpui.Px(0.0), buttonSize.Height)
	case gpui.AnchorBottomCenter:
		p = gpui.NewPoint(buttonSize.Width/2.0, buttonSize.Height)
	case gpui.AnchorBottomRight:
		p = gpui.NewPoint(buttonSize.Width, buttonSize.Height)
	}
	return offset.Add(p)
}

func buttons() []ButtonDemo {
	return []ButtonDemo{
		{label: "TopLeft", corner: gpui.Some(gpui.AnchorTopLeft)},
		{label: "TopCenter", corner: gpui.Some(gpui.AnchorTopCenter)},
		{label: "TopRight", corner: gpui.Some(gpui.AnchorTopRight)},
		{label: "LeftCenter", corner: gpui.Some(gpui.AnchorLeftCenter)},
		{label: "Center", corner: gpui.None[gpui.Anchor]()},
		{label: "RightCenter", corner: gpui.Some(gpui.AnchorRightCenter)},
		{label: "BottomLeft", corner: gpui.Some(gpui.AnchorBottomLeft)},
		{label: "BottomCenter", corner: gpui.Some(gpui.AnchorBottomCenter)},
		{label: "BottomRight", corner: gpui.Some(gpui.AnchorBottomRight)},
	}
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func isSomeOf(o gpui.Option[int], index int) bool {
	v, ok := o.Get()
	return ok && v == index
}

// Render renders the view.
func (d *AnchorDemo) Render(_ *gpui.Window, cx *gpui.Context[AnchorDemo]) gpui.IntoElement {
	buttons := buttons()
	buttonSize := gpui.NewSize(gpui.Px(120.0), gpui.Px(65.0))

	children := make([]gpui.IntoElement, 0, len(buttons))
	for index, button := range buttons {
		isHovered := isSomeOf(d.hoveredButton, index)
		isHoverable := button.corner.IsSome()
		hovered := gpui.None[int]()
		if isHovered {
			hovered = d.hoveredButton
		}
		children = append(children, gpui.NewDiv().
			Relative().
			Child(
				gpui.NewDiv().
					ID(gpui.ElementIDNamedInteger("button", uint64(index))).
					W(buttonSize.Width).
					H(buttonSize.Height).
					Flex().
					ItemsCenter().
					JustifyCenter().
					Bg(gpui.White()).
					When(isHoverable, func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
						listener := cx.Listener(func(this *AnchorDemo, hovered *bool, _ *gpui.Window, cx *gpui.Context[AnchorDemo]) {
							if *hovered {
								this.hoveredButton = gpui.Some(index)
							} else if isSomeOf(this.hoveredButton, index) {
								this.hoveredButton = gpui.None[int]()
							}
							cx.Notify()
						})
						return this.Border1().
							RoundedLg().
							BorderColor(gpui.Black()).
							Hover(func(style *gpui.StyleRefinement) *gpui.StyleRefinement {
								return styled(styled(style).Bg(gpui.Black())).TextColor(gpui.White())
							}).
							OnHover(func(hovered bool, window *gpui.Window, cx *gpui.App) {
								listener(&hovered, window, cx)
							}).
							Child(button.label)
					}),
			).
			WhenSome(hovered, func(this *gpui.Div, index int) *gpui.Div {
				button := buttons[index]
				corner, ok := button.corner.Get()
				if !ok {
					return this
				}

				position := resolvedPosition(corner, buttonSize)
				return this.Child(gpui.NewDeferred(
					gpui.NewAnchored().
						Anchor(corner).
						Position(position).
						PositionMode(gpui.AnchoredPositionModeLocal).
						SnapToWindow().
						Child(
							gpui.NewDiv().
								Py0p5().
								Px2().
								Bg(gpui.Black().Opacity(0.75)).
								TextColor(gpui.Rgb(0xffffff)).
								RoundedSm().
								ShadowSm().
								MinW(gpui.Px(100.0)).
								TextSm().
								Child(button.label),
						),
				))
			}))
	}

	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		ItemsCenter().
		JustifyCenter().
		Bg(gpui.White()).
		Gap4().
		P10().
		ChildText("Popover with Anchor").
		Child(
			gpui.NewDiv().
				Size128().
				Grid().
				GridCols(3).
				Gap6().
				Relative().
				Children(children...),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(750.), gpui.Px(600.)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[AnchorDemo] {
			return cx.New(func(*gpui.Context[AnchorDemo]) *AnchorDemo {
				return &AnchorDemo{hoveredButton: gpui.None[int]()}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
