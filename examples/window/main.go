package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
	"github.com/megascan/gpui-go/scheduler"
)

// SubWindow is the content of the windows the demo opens.
type SubWindow struct {
	customTitlebar bool
	isDialog       bool
}

func button(text string, onClick func(window *gpui.Window, cx *gpui.App)) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(text).
		FlexNone().
		Px2().
		Bg(gpui.Rgb(0xf7f7f7)).
		Active(func(this *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(this).Opacity(0.85) }).
		Border1().
		BorderColor(gpui.Rgb(0xe0e0e0)).
		RoundedSm().
		CursorPointer().
		ChildText(text).
		OnClick(func(_ *gpui.ClickEvent, window *gpui.Window, cx *gpui.App) { onClick(window, cx) })
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func openSubWindow(cx *gpui.App, options gpui.WindowOptions, sub SubWindow) {
	if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[SubWindow] {
		return cx.New(func(_ *gpui.Context[SubWindow]) *SubWindow {
			s := sub
			return &s
		})
	}); err != nil {
		panic(err)
	}
}

// Render renders the sub window.
func (s *SubWindow) Render(_ *gpui.Window, cx *gpui.Context[SubWindow]) gpui.IntoElement {
	windowBounds := gpui.WindowBoundsWindowed(gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(250.0), gpui.Px(200.0)), cx.App))

	return gpui.NewDiv().
		Flex().
		FlexCol().
		Bg(gpui.Rgb(0xffffff)).
		SizeFull().
		Gap2().
		When(s.customTitlebar, func(cx *gpui.Div) *gpui.Div {
			return cx.Child(
				gpui.NewDiv().
					Flex().
					H(gpui.Px(32.)).
					Px4().
					Bg(gpui.Blue()).
					TextColor(gpui.White()).
					WFull().
					Child(
						gpui.NewDiv().
							Flex().
							ItemsCenter().
							JustifyCenter().
							SizeFull().
							ChildText("Custom Titlebar"),
					),
			)
		}).
		Child(
			gpui.NewDiv().
				P8().
				Flex().
				FlexCol().
				Gap2().
				ChildText("SubWindow").
				When(s.isDialog, func(div *gpui.Div) *gpui.Div {
					return div.Child(button("Open Nested Dialog", func(_ *gpui.Window, cx *gpui.App) {
						options := gpui.DefaultWindowOptions()
						options.WindowBounds = gpui.Some(windowBounds)
						options.Kind = gpui.WindowKindDialog{}
						openSubWindow(cx, options, SubWindow{customTitlebar: false, isDialog: true})
					}))
				}).
				Child(button("Close", func(window *gpui.Window, _ *gpui.App) {
					window.RemoveWindow()
				})),
		)
}

// WindowDemo is the root view of the example.
type WindowDemo struct{}

// Render renders the demo.
func (d *WindowDemo) Render(_ *gpui.Window, cx *gpui.Context[WindowDemo]) gpui.IntoElement {
	windowBounds := gpui.WindowBoundsWindowed(gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)), cx.App))
	withBounds := func(mutate func(options *gpui.WindowOptions)) gpui.WindowOptions {
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(windowBounds)
		mutate(&options)
		return options
	}
	plain := SubWindow{customTitlebar: false, isDialog: false}

	return gpui.NewDiv().
		P4().
		Flex().
		FlexWrap().
		Bg(gpui.Rgb(0xffffff)).
		SizeFull().
		JustifyCenter().
		ContentCenter().
		Gap2().
		Child(button("Normal", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(*gpui.WindowOptions) {}), plain)
		})).
		Child(button("Popup", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.Kind = gpui.WindowKindPopUp{} }), plain)
		})).
		Child(button("Floating", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.Kind = gpui.WindowKindFloating{} }), plain)
		})).
		Child(button("Dialog", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.Kind = gpui.WindowKindDialog{} }), SubWindow{customTitlebar: false, isDialog: true})
		})).
		Child(button("Custom Titlebar", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.Titlebar = gpui.None[gpui.TitlebarOptions]() }), SubWindow{customTitlebar: true, isDialog: false})
		})).
		Child(button("Invisible", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.Show = false }), plain)
		})).
		Child(button("Unmovable", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) {
				o.IsMovable = false
				o.Titlebar = gpui.None[gpui.TitlebarOptions]()
			}), plain)
		})).
		Child(button("Unresizable", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.IsResizable = false }), plain)
		})).
		Child(button("Unminimizable", func(_ *gpui.Window, cx *gpui.App) {
			openSubWindow(cx, withBounds(func(o *gpui.WindowOptions) { o.IsMinimizable = false }), plain)
		})).
		Child(button("Hide Application", func(window *gpui.Window, cx *gpui.App) {
			cx.Hide()

			window.Spawn[struct{}](cx, func(cx *gpui.AsyncWindowContext) (struct{}, error) {
				if _, err := cx.BackgroundExecutor().Timer(3 * time.Second).Await(cx); err != nil {
					return struct{}{}, err
				}
				return struct{}{}, cx.Update(func(_ *gpui.Window, cx *gpui.App) {
					cx.Activate(false)
				})
			}).Detach()
		})).
		Child(button("Resize", func(window *gpui.Window, _ *gpui.App) {
			contentSize := window.Bounds().Size
			window.Resize(gpui.NewSize(contentSize.Height, contentSize.Width))
		})).
		Child(button("Prompt", func(window *gpui.Window, cx *gpui.App) {
			answer := window.Prompt(
				gpui.PromptLevelInfo,
				"Are you sure?",
				gpui.None[string](),
				[]gpui.PromptButton{gpui.PromptButtonFromStr("OK"), gpui.PromptButtonFromStr("Cancel")},
				cx,
			)
			spawnAnswer(cx, answer)
		})).
		Child(button("Prompt (non-English)", func(window *gpui.Window, cx *gpui.App) {
			answer := window.Prompt(
				gpui.PromptLevelInfo,
				"Are you sure?",
				gpui.None[string](),
				[]gpui.PromptButton{gpui.PromptButtonOk("确定"), gpui.PromptButtonCancel("取消")},
				cx,
			)
			spawnAnswer(cx, answer)
		}))
}

func spawnAnswer(cx *gpui.App, answer *scheduler.OneshotReceiver[int]) {
	cx.Spawn[struct{}](func(cx *gpui.AsyncApp) (struct{}, error) {
		index, err := answer.Recv(cx)
		if err != nil {
			panic(err)
		}
		if index == 0 {
			fmt.Println("You have clicked Ok")
		} else {
			fmt.Println("You have clicked Cancel")
		}
		return struct{}{}, nil
	}).Detach()
}

// Quit quits the example.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "window::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "window::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(800.0), gpui.Px(600.0)), cx)

		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[WindowDemo] {
			return cx.New(func(cx *gpui.Context[WindowDemo]) *WindowDemo {
				cx.ObserveWindowBounds(window, func(_ *WindowDemo, window *gpui.Window, _ *gpui.Context[WindowDemo]) {
					fmt.Printf("Window bounds changed: %v\n", window.Bounds())
				}).Detach()

				return &WindowDemo{}
			})
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
		cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
		cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("cmd-q", Quit{}, gpui.None[string]())})
	})
}

func main() {
	runExample()
}
