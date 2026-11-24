package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld shows how to use deferred to create floating layers.
type HelloWorld struct {
	open          bool
	secondaryOpen bool
}

func button(id string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Bg(gpui.Black()).
		TextColor(gpui.White()).
		Px3().
		Py1()
}

func popover() *gpui.Div {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		ItemsCenter().
		JustifyCenter().
		ShadowLg().
		P3().
		RoundedMd().
		Bg(gpui.White()).
		TextColor(gpui.Black()).
		Border1().
		TextSm().
		BorderColor(gpui.Black().Opacity(0.1))
}

func line(color gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().W(gpui.Px(480.)).H2().Bg(color.Opacity(0.25))
}

func (h *HelloWorld) renderSecondaryPopover(_ *gpui.Window, cx *gpui.Context[HelloWorld]) gpui.IntoElement {
	return button("secondary-btn").
		Mt2().
		ChildText("Child Popover").
		OnClick(cx.Listener(func(this *HelloWorld, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
			this.secondaryOpen = true
			cx.Notify()
		})).
		When(h.secondaryOpen, func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
			return this.Child(
				gpui.NewDeferred(
					gpui.NewAnchored().
						Anchor(gpui.AnchorTopLeft).
						SnapToWindowWithMargin(gpui.EdgesAll(gpui.Px(8.))).
						Child(
							popover().
								ChildText("This is second level Popover with nested deferred!").
								Bg(gpui.White()).
								BorderColor(gpui.Blue()).
								OnMouseDownOut(cx.Listener(func(this *HelloWorld, _ *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
									this.secondaryOpen = false
									cx.Notify()
								})),
						),
				).
					Priority(2),
			)
		})
}

// Render renders the view.
func (h *HelloWorld) Render(window *gpui.Window, cx *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		SizeFull().
		Bg(gpui.White()).
		TextColor(gpui.Black()).
		JustifyCenter().
		ItemsCenter().
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				Gap4().
				Child(
					button("popover0").ChildText("Opened Popover").Child(
						gpui.NewDeferred(
							gpui.NewAnchored().
								Anchor(gpui.AnchorTopLeft).
								SnapToWindowWithMargin(gpui.EdgesAll(gpui.Px(8.))).
								Child(popover().W96().Gap3().ChildText(
									"This is a default opened Popover, "+
										"we can use deferred to render it "+
										"in a floating layer.",
								)),
						).
							Priority(0),
					),
				).
				Child(
					button("popover1").
						ChildText("Open Popover").
						OnClick(cx.Listener(func(this *HelloWorld, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
							this.open = true
							cx.Notify()
						})).
						When(h.open, func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
							return this.Child(
								gpui.NewDeferred(
									gpui.NewAnchored().
										Anchor(gpui.AnchorTopLeft).
										SnapToWindowWithMargin(gpui.EdgesAll(gpui.Px(8.))).
										Child(
											popover().
												W96().
												Gap3().
												ChildText(
													"This is first level Popover, "+
														"we can use deferred to render it "+
														"in a floating layer.\n"+
														"Click outside to close.",
												).
												When(!h.secondaryOpen, func(this *gpui.Div) *gpui.Div {
													return this.OnMouseDownOut(cx.Listener(func(this *HelloWorld, _ *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
														this.open = false
														cx.Notify()
													}))
												}).
												Child(h.renderSecondaryPopover(window, cx)),
										),
								).
									Priority(1),
							)
						}),
				),
		).
		ChildText(
			"Here is an example text rendered, "+
				"to ensure the Popover will float above this contents.",
		).
		Children(
			line(gpui.Red()),
			line(gpui.Yellow()),
			line(gpui.Blue()),
			line(gpui.Green()),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(*gpui.Context[HelloWorld]) *HelloWorld {
				return &HelloWorld{open: false, secondaryOpen: false}
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
