package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// WindowShadow draws client-side decorations with a resizable shadow.
type WindowShadow struct{}

// Render renders the window.
func (w *WindowShadow) Render(window *gpui.Window, _ *gpui.Context[WindowShadow]) gpui.IntoElement {
	decorations := window.WindowDecorations()
	rounding := gpui.Px(10.0)
	shadowSize := gpui.Px(10.0)
	borderSize := gpui.Px(1.0)
	grey := gpui.Rgb(0x808080)
	window.SetClientInset(shadowSize)

	return gpui.NewDiv().
		ID("window-backdrop").
		Bg(gpui.TransparentBlack()).
		Map[*gpui.Stateful[*gpui.Div]](func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
		tiling, ok := decorations.Client()
		if !ok {
			return div
		}
		return div.
			Bg(gpui.TransparentBlack()).
			Child(
				gpui.NewCanvas(
					func(_ gpui.Bounds[gpui.Pixels], window *gpui.Window, _ *gpui.App) gpui.Hitbox {
						return window.InsertHitbox(
							gpui.NewBounds(
								gpui.NewPoint(gpui.Px(0.0), gpui.Px(0.0)),
								window.WindowBounds().GetBounds().Size,
							),
							gpui.HitboxBehaviorNormal,
						)
					},
					func(_ gpui.Bounds[gpui.Pixels], hitbox gpui.Hitbox, window *gpui.Window, _ *gpui.App) {
						mouse := window.MousePosition()
						size := window.WindowBounds().GetBounds().Size
						edge, ok := resizeEdge(mouse, shadowSize, size)
						if !ok {
							return
						}
						var style gpui.CursorStyle
						switch edge {
						case gpui.ResizeEdgeTop, gpui.ResizeEdgeBottom:
							style = gpui.CursorStyleResizeUpDown
						case gpui.ResizeEdgeLeft, gpui.ResizeEdgeRight:
							style = gpui.CursorStyleResizeLeftRight
						case gpui.ResizeEdgeTopLeft, gpui.ResizeEdgeBottomRight:
							style = gpui.CursorStyleResizeUpLeftDownRight
						case gpui.ResizeEdgeTopRight, gpui.ResizeEdgeBottomLeft:
							style = gpui.CursorStyleResizeUpRightDownLeft
						}
						window.SetCursorStyle(style, hitbox)
					},
				).
					SizeFull().
					Absolute(),
			).
			When(!(tiling.Top || tiling.Right), func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
				return div.RoundedTr(rounding)
			}).
			When(!(tiling.Top || tiling.Left), func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return div.RoundedTl(rounding) }).
			When(!tiling.Top, func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return div.Pt(shadowSize) }).
			When(!tiling.Bottom, func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return div.Pb(shadowSize) }).
			When(!tiling.Left, func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return div.Pl(shadowSize) }).
			When(!tiling.Right, func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return div.Pr(shadowSize) }).
			OnMouseMove(func(_ *gpui.MouseMoveEvent, window *gpui.Window, _ *gpui.App) { window.Refresh() }).
			OnMouseDown(gpui.MouseButtonLeft, func(e *gpui.MouseDownEvent, window *gpui.Window, _ *gpui.App) {
				size := window.WindowBounds().GetBounds().Size
				pos := e.Position

				if edge, ok := resizeEdge(pos, shadowSize, size); ok {
					window.StartWindowResize(edge)
				} else {
					window.StartWindowMove()
				}
			})
	}).
		SizeFull().
		Child(
			gpui.NewDiv().
				Cursor(gpui.CursorStyleArrow).
				Map[*gpui.Div](func(div *gpui.Div) *gpui.Div {
				tiling, ok := decorations.Client()
				if !ok {
					return div
				}
				return div.
					BorderColor(grey).
					When(!(tiling.Top || tiling.Right), func(div *gpui.Div) *gpui.Div {
						return div.RoundedTr(rounding)
					}).
					When(!(tiling.Top || tiling.Left), func(div *gpui.Div) *gpui.Div { return div.RoundedTl(rounding) }).
					When(!tiling.Top, func(div *gpui.Div) *gpui.Div { return div.BorderT(borderSize) }).
					When(!tiling.Bottom, func(div *gpui.Div) *gpui.Div { return div.BorderB(borderSize) }).
					When(!tiling.Left, func(div *gpui.Div) *gpui.Div { return div.BorderL(borderSize) }).
					When(!tiling.Right, func(div *gpui.Div) *gpui.Div { return div.BorderR(borderSize) }).
					When(!tiling.IsTiled(), func(div *gpui.Div) *gpui.Div {
						return div.Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(
								gpui.Px(0.),
								gpui.Px(0.),
								gpui.Hsla{H: 0., S: 0., L: 0., A: 0.4},
							).
								BlurRadius(shadowSize / 2.),
						})
					})
			}).
				OnMouseMove(func(_ *gpui.MouseMoveEvent, _ *gpui.Window, cx *gpui.App) {
					cx.StopPropagation()
				}).
				Bg(gpui.Rgb(0xCCCCFF)).
				SizeFull().
				Flex().
				FlexCol().
				JustifyAround().
				Child(
					gpui.NewDiv().WFull().Flex().FlexRow().JustifyAround().Child(
						gpui.NewDiv().
							Flex().
							Bg(gpui.White()).
							Size(gpui.Px(300.0)).
							JustifyCenter().
							ItemsCenter().
							ShadowLg().
							Border1().
							BorderColor(gpui.Rgb(0x0000ff)).
							TextXl().
							TextColor(gpui.Rgb(0xffffff)).
							Child(
								gpui.NewDiv().
									ID("hello").
									W(gpui.Px(200.0)).
									H(gpui.Px(100.0)).
									Bg(gpui.Green()).
									Shadow([]gpui.BoxShadow{
										gpui.NewBoxShadow(
											gpui.Px(0.),
											gpui.Px(0.),
											gpui.Hsla{H: 0., S: 0., L: 0., A: 1.0},
										).
											BlurRadius(gpui.Px(20.0)),
									}).
									Map[*gpui.Stateful[*gpui.Div]](func(div *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
									if decorations.IsServer() {
										return div
									}
									return div.
										OnMouseDown(
											gpui.MouseButtonLeft,
											func(_ *gpui.MouseDownEvent, window *gpui.Window, _ *gpui.App) {
												window.StartWindowMove()
											},
										).
										OnClick(func(e *gpui.ClickEvent, window *gpui.Window, _ *gpui.App) {
											if e.IsRightClick() {
												window.ShowWindowMenu(e.Position())
											}
										}).
										TextColor(gpui.Black()).
										ChildText("this is the custom titlebar")
								}),
							),
					),
				),
		)
}

func resizeEdge(pos gpui.Point[gpui.Pixels], shadowSize gpui.Pixels, size gpui.Size[gpui.Pixels]) (gpui.ResizeEdge, bool) {
	var edge gpui.ResizeEdge
	switch {
	case pos.Y < shadowSize && pos.X < shadowSize:
		edge = gpui.ResizeEdgeTopLeft
	case pos.Y < shadowSize && pos.X > size.Width-shadowSize:
		edge = gpui.ResizeEdgeTopRight
	case pos.Y < shadowSize:
		edge = gpui.ResizeEdgeTop
	case pos.Y > size.Height-shadowSize && pos.X < shadowSize:
		edge = gpui.ResizeEdgeBottomLeft
	case pos.Y > size.Height-shadowSize && pos.X > size.Width-shadowSize:
		edge = gpui.ResizeEdgeBottomRight
	case pos.Y > size.Height-shadowSize:
		edge = gpui.ResizeEdgeBottom
	case pos.X < shadowSize:
		edge = gpui.ResizeEdgeLeft
	case pos.X > size.Width-shadowSize:
		edge = gpui.ResizeEdgeRight
	default:
		return 0, false
	}
	return edge, true
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(600.0), gpui.Px(600.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		options.WindowBackground = gpui.WindowBackgroundAppearanceOpaque
		options.WindowDecorations = gpui.Some(gpui.WindowDecorationsClient)
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[WindowShadow] {
			return cx.New(func(cx *gpui.Context[WindowShadow]) *WindowShadow {
				cx.ObserveWindowAppearance(window, func(_ *WindowShadow, window *gpui.Window, _ *gpui.Context[WindowShadow]) {
					window.Refresh()
				}).Detach()
				return &WindowShadow{}
			})
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
