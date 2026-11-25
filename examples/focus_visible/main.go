package main

import (
	"encoding/json/jsontext"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Tab moves focus to the next tab stop.
type Tab struct{}

// Name returns the action name.
func (Tab) Name() string { return "example::Tab" }

// BoxedClone clones the action.
func (Tab) BoxedClone() gpui.Action { return Tab{} }

// PartialEq reports whether other is a Tab.
func (Tab) PartialEq(other gpui.Action) bool {
	_, ok := other.(Tab)
	return ok
}

// TabPrev moves focus to the previous tab stop.
type TabPrev struct{}

// Name returns the action name.
func (TabPrev) Name() string { return "example::TabPrev" }

// BoxedClone clones the action.
func (TabPrev) BoxedClone() gpui.Action { return TabPrev{} }

// PartialEq reports whether other is a TabPrev.
func (TabPrev) PartialEq(other gpui.Action) bool {
	_, ok := other.(TabPrev)
	return ok
}

// Quit quits the example.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "example::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "example::Tab",
		Type:  reflect.TypeFor[Tab](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Tab{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "example::TabPrev",
		Type:  reflect.TypeFor[TabPrev](),
		Build: func(jsontext.Value) (gpui.Action, error) { return TabPrev{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "example::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}

type item struct {
	handle gpui.FocusHandle
	label  string
}

// Example is the focus_visible example view.
type Example struct {
	focusHandle gpui.FocusHandle
	items       []item
	message     gpui.SharedString
}

// NewExample creates the view and focuses it.
func NewExample(window *gpui.Window, cx *gpui.Context[Example]) *Example {
	items := []item{
		{
			cx.FocusHandle().WithTabIndex(1).WithTabStop(true),
			"Button with .focus() - always shows border when focused",
		},
		{
			cx.FocusHandle().WithTabIndex(2).WithTabStop(true),
			"Button with .focus_visible() - only shows border with keyboard",
		},
		{
			cx.FocusHandle().WithTabIndex(3).WithTabStop(true),
			"Button with both .focus() and .focus_visible()",
		},
	}

	focusHandle := cx.FocusHandle()
	window.Focus(focusHandle, cx.App)

	return &Example{
		focusHandle: focusHandle,
		items:       items,
		message:     "Try clicking vs tabbing! Click shows no border, Tab shows border.",
	}
}

func (e *Example) onTab(_ *Tab, window *gpui.Window, cx *gpui.Context[Example]) {
	window.FocusNext(cx.App)
	e.message = "Pressed Tab - focus-visible border should appear!"
}

func (e *Example) onTabPrev(_ *TabPrev, window *gpui.Window, cx *gpui.Context[Example]) {
	window.FocusPrev(cx.App)
	e.message = "Pressed Shift-Tab - focus-visible border should appear!"
}

func (e *Example) onQuit(_ *Quit, _ *gpui.Window, cx *gpui.Context[Example]) {
	cx.Quit()
}

func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func buttonBase(id string, label string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		H16().
		WFull().
		Flex().
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x2563eb)).
		TextColor(gpui.White()).
		RoundedMd().
		CursorPointer().
		Hover(func(style *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(style).Bg(gpui.Rgb(0x1d4ed8)) }).
		ChildText(label)
}

func focusStyle(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(styled(style).Border4()).BorderColor(gpui.Rgb(0xfbbf24))
}

func focusVisibleStyle(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(styled(style).Border4()).BorderColor(gpui.Rgb(0x10b981))
}

func heading(text string) *gpui.Div {
	return gpui.NewDiv().
		TextSm().
		FontWeight(gpui.FontWeightBold).
		TextColor(gpui.Rgb(0x374151)).
		ChildText(text)
}

// Render renders the view.
func (e *Example) Render(_ *gpui.Window, cx *gpui.Context[Example]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("app").
		TrackFocus(e.focusHandle).
		OnAction(actionListener(cx.Listener((*Example).onTab))).
		OnAction(actionListener(cx.Listener((*Example).onTabPrev))).
		OnAction(actionListener(cx.Listener((*Example).onQuit))).
		SizeFull().
		Flex().
		FlexCol().
		P8().
		Gap6().
		Bg(gpui.Rgb(0xf3f4f6)).
		Child(
			gpui.NewDiv().
				Text2xl().
				FontWeight(gpui.FontWeightBold).
				TextColor(gpui.Rgb(0x111827)).
				ChildText("CSS focus-visible Demo"),
		).
		Child(
			gpui.NewDiv().
				P4().
				RoundedMd().
				Bg(gpui.Rgb(0xdbeafe)).
				TextColor(gpui.Rgb(0x1e3a8a)).
				Child(e.message),
		).
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				Gap4().
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						Gap2().
						Child(heading("1. Regular .focus() - always visible:")).
						Child(
							buttonBase("button1", e.items[0].label).
								TrackFocus(e.items[0].handle).
								Focus(focusStyle).
								OnClick(cx.Listener(func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
									this.message = "Clicked button 1 - focus border is visible!"
									cx.Notify()
								})),
						),
				).
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						Gap2().
						Child(heading("2. New .focus_visible() - only keyboard:")).
						Child(
							buttonBase("button2", e.items[1].label).
								TrackFocus(e.items[1].handle).
								FocusVisible(focusVisibleStyle).
								OnClick(cx.Listener(func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
									this.message = "Clicked button 2 - no border! Try Tab instead."
									cx.Notify()
								})),
						),
				).
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						Gap2().
						Child(heading("3. Both .focus() (yellow) and .focus_visible() (green):")).
						Child(
							buttonBase("button3", e.items[2].label).
								TrackFocus(e.items[2].handle).
								Focus(focusStyle).
								FocusVisible(focusVisibleStyle).
								OnClick(cx.Listener(func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
									this.message = "Clicked button 3 - yellow border. Tab shows green!"
									cx.Notify()
								})),
						),
				),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("tab", Tab{}, gpui.None[string]()),
			gpui.NewKeyBinding("shift-tab", TabPrev{}, gpui.None[string]()),
			gpui.NewKeyBinding("cmd-q", Quit{}, gpui.None[string]()),
		})

		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(800.), gpui.Px(600.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[Example] {
			return cx.New(func(cx *gpui.Context[Example]) *Example { return NewExample(window, cx) })
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	runExample()
}
