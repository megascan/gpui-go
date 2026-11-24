package main

import (
	"encoding/json/jsontext"
	"fmt"
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
}

// Example is the tab_stop example view.
type Example struct {
	focusHandle gpui.FocusHandle
	items       []gpui.FocusHandle
	message     gpui.SharedString
}

// NewExample creates the view and focuses it.
func NewExample(window *gpui.Window, cx *gpui.Context[Example]) *Example {
	items := []gpui.FocusHandle{
		cx.FocusHandle().WithTabIndex(1).WithTabStop(true),
		cx.FocusHandle().WithTabIndex(2).WithTabStop(true),
		cx.FocusHandle().WithTabIndex(3).WithTabStop(true),
		cx.FocusHandle(),
		cx.FocusHandle().WithTabIndex(2).WithTabStop(true),
	}

	focusHandle := cx.FocusHandle()
	window.Focus(focusHandle, cx.App)

	return &Example{
		focusHandle: focusHandle,
		items:       items,
		message:     "Press `Tab`, `Shift-Tab` to switch focus.",
	}
}

func (e *Example) onTab(_ *Tab, window *gpui.Window, cx *gpui.Context[Example]) {
	window.FocusNext(cx.App)
	e.message = "You have pressed `Tab`."
}

func (e *Example) onTabPrev(_ *TabPrev, window *gpui.Window, cx *gpui.Context[Example]) {
	window.FocusPrev(cx.App)
	e.message = "You have pressed `Shift-Tab`."
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

func tabStopStyle(this *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(styled(this).Border3()).BorderColor(gpui.Blue())
}

func tabStopStyleDiv(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
	return this.Border3().BorderColor(gpui.Blue())
}

func button(id string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		H10().
		Flex1().
		Flex().
		JustifyCenter().
		ItemsCenter().
		Border1().
		BorderColor(gpui.Black()).
		Bg(gpui.Black()).
		TextColor(gpui.White()).
		Focus(tabStopStyle).
		ShadowSm()
}

// Render renders the view.
func (e *Example) Render(window *gpui.Window, cx *gpui.Context[Example]) gpui.IntoElement {
	items := make([]gpui.IntoElement, 0, len(e.items))
	for ix, itemHandle := range e.items {
		items = append(items, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("item", uint64(ix))).
			TrackFocus(itemHandle).
			H10().
			WFull().
			Flex().
			JustifyCenter().
			ItemsCenter().
			Border1().
			BorderColor(gpui.Black()).
			When(itemHandle.TabStop && itemHandle.IsFocused(window), tabStopStyleDiv).
			Map(func(this *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
				if itemHandle.TabStop {
					return this.
						Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement {
							return styled(this).Bg(gpui.Black().Opacity(0.1))
						}).
						ChildText(fmt.Sprintf("tab_index: %d", itemHandle.TabIndex))
				}
				return this.Opacity(0.4).ChildText("tab_stop: false")
			}))
	}

	return gpui.NewDiv().
		ID("app").
		TrackFocus(e.focusHandle).
		OnAction(actionListener(cx.Listener((*Example).onTab))).
		OnAction(actionListener(cx.Listener((*Example).onTabPrev))).
		SizeFull().
		Flex().
		FlexCol().
		P4().
		Gap3().
		Bg(gpui.White()).
		TextColor(gpui.Black()).
		Child(e.message).
		Children(items...).
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				Gap3().
				ItemsCenter().
				Child(
					button("el1").
						TabIndex(4).
						ChildText("Button 1").
						OnClick(cx.Listener(func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							this.message = "You have clicked Button 1."
							cx.Notify()
						})),
				).
				Child(
					button("el2").
						TabIndex(5).
						ChildText("Button 2").
						OnClick(cx.Listener(func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							this.message = "You have clicked Button 2."
							cx.Notify()
						})),
				),
		).
		Child(
			gpui.NewDiv().
				ID("group-1").
				TabIndex(6).
				TabGroup().
				TabStop(false).
				Child(
					button("group-1-button-1").
						TabIndex(1).
						ChildText("Tab index [6, 1]"),
				).
				Child(
					button("group-1-button-2").
						TabIndex(2).
						ChildText("Tab index [6, 2]"),
				).
				Child(
					button("group-1-button-3").
						TabIndex(3).
						ChildText("Tab index [6, 3]"),
				),
		).
		Child(
			gpui.NewDiv().
				ID("group-2").
				TabIndex(7).
				TabGroup().
				TabStop(false).
				Child(
					button("group-2-button-1").
						TabIndex(1).
						ChildText("Tab index [7, 1]"),
				).
				Child(
					button("group-2-button-2").
						TabIndex(2).
						ChildText("Tab index [7, 2]"),
				).
				Child(
					button("group-2-button-3").
						TabIndex(3).
						ChildText("Tab index [7, 3]"),
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
