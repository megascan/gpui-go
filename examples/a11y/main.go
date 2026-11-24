// Command a11y is the accessibility (AccessKit) demo app.
//
// It uses GPUI's accessibility APIs to attach structured information to the element tree, so
// assistive technology can see and interact with the UI programmatically: a heading, a spin
// button counter with Increment and Decrement actions, a reset button, a switch and a list of
// three tasks, in a window titled "GPUI Accessibility Demo".
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Tab moves focus to the next tab stop.
type Tab struct{}

// Name returns the action name.
func (Tab) Name() string { return "a11y_example::Tab" }

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
func (TabPrev) Name() string { return "a11y_example::TabPrev" }

// BoxedClone clones the action.
func (TabPrev) BoxedClone() gpui.Action { return TabPrev{} }

// PartialEq reports whether other is a TabPrev.
func (TabPrev) PartialEq(other gpui.Action) bool {
	_, ok := other.(TabPrev)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "a11y_example::Tab",
		Type:  reflect.TypeFor[Tab](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Tab{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "a11y_example::TabPrev",
		Type:  reflect.TypeFor[TabPrev](),
		Build: func(jsontext.Value) (gpui.Action, error) { return TabPrev{}, nil },
	})
}

// A11yDemo is the a11y example view.
type A11yDemo struct {
	focusHandle gpui.FocusHandle
	count       int32
	enabled     bool
}

// NewA11yDemo creates the view and focuses it.
func NewA11yDemo(window *gpui.Window, cx *gpui.Context[A11yDemo]) *A11yDemo {
	focusHandle := cx.FocusHandle()
	window.Focus(focusHandle, cx.App)
	return &A11yDemo{
		focusHandle: focusHandle,
		count:       0,
		enabled:     false,
	}
}

func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

// Render renders the view.
func (d *A11yDemo) Render(_ *gpui.Window, cx *gpui.Context[A11yDemo]) gpui.IntoElement {
	toggled := a11y.ToggledFalse
	if d.enabled {
		toggled = a11y.ToggledTrue
	}

	labels := []string{"Write code", "Run tests", "Ship it"}
	tasks := make([]gpui.IntoElement, 0, len(labels))
	for i, label := range labels {
		tasks = append(tasks, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("task", uint64(i))).
			Role(a11y.RoleListItem).
			AriaLabel(gpui.SharedString(label)).
			AriaPositionInSet(i+1).
			AriaSizeOfSet(3).
			Py1().
			Px2().
			Child(gpui.TextMacro(gpui.SharedString(fmt.Sprintf("%d. %s", i+1, label)))))
	}

	return gpui.NewDiv().
		ID("root").
		Role(a11y.RoleApplication).
		AriaLabel("Accessibility Demo").
		TrackFocus(d.focusHandle).
		OnAction(actionListener(cx.Listener(func(_ *A11yDemo, _ *Tab, window *gpui.Window, cx *gpui.Context[A11yDemo]) {
			window.FocusNext(cx.App)
		}))).
		OnAction(actionListener(cx.Listener(func(_ *A11yDemo, _ *TabPrev, window *gpui.Window, cx *gpui.Context[A11yDemo]) {
			window.FocusPrev(cx.App)
		}))).
		SizeFull().
		Flex().
		FlexCol().
		Gap4().
		P4().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Child(
			gpui.NewDiv().
				ID("heading").
				Role(a11y.RoleHeading).
				AriaLevel(1).
				AriaLabel("Accessibility Demo").
				TextXl().
				FontWeight(gpui.FontWeightBold).
				Child(gpui.TextMacro("Accessibility Demo")),
		).
		Child(
			gpui.NewDiv().
				Flex().
				ItemsCenter().
				Gap3().
				Child(
					gpui.NewDiv().
						ID("counter").
						Focusable().
						TabStop(true).
						Role(a11y.RoleSpinButton).
						AriaLabel(gpui.SharedString(fmt.Sprintf("Counter: %d", d.count))).
						AriaNumericValue(float64(d.count)).
						AriaMinNumericValue(0.0).
						OnA11yAction(a11y.ActionIncrement, func() func(a11y.ActionData, *gpui.Window, *gpui.App) {
							this := cx.WeakEntity()
							return func(_ a11y.ActionData, _ *gpui.Window, cx *gpui.App) {
								_ = this.Update(cx, func(this *A11yDemo, cx *gpui.Context[A11yDemo]) {
									this.count++
									cx.Notify()
								})
							}
						}()).
						OnA11yAction(a11y.ActionDecrement, func() func(a11y.ActionData, *gpui.Window, *gpui.App) {
							this := cx.WeakEntity()
							return func(_ a11y.ActionData, _ *gpui.Window, cx *gpui.App) {
								_ = this.Update(cx, func(this *A11yDemo, cx *gpui.Context[A11yDemo]) {
									this.count = max(this.count-1, 0)
									cx.Notify()
								})
							}
						}()).
						OnClick(cx.Listener(func(this *A11yDemo, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[A11yDemo]) {
							this.count++
							cx.Notify()
						})).
						Px3().
						Py1().
						RoundedMd().
						Bg(gpui.Rgb(0x89b4fa)).
						TextColor(gpui.Rgb(0x1e1e2e)).
						CursorPointer().
						Child(gpui.TextMacro(gpui.SharedString(fmt.Sprintf("Count: %d", d.count)))),
				).
				Child(
					gpui.NewDiv().
						ID("reset").
						Focusable().
						TabStop(true).
						Role(a11y.RoleButton).
						AriaLabel("Reset counter").
						Px3().
						Py1().
						RoundedMd().
						Bg(gpui.Rgb(0x585b70)).
						CursorPointer().
						OnClick(cx.Listener(func(this *A11yDemo, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[A11yDemo]) {
							this.count = 0
							cx.Notify()
						})).
						Child(gpui.TextMacro("Reset")),
				),
		).
		Child(
			gpui.NewDiv().
				Flex().
				ItemsCenter().
				Gap2().
				Child(
					gpui.NewDiv().
						ID("toggle").
						Focusable().
						TabStop(true).
						Role(a11y.RoleSwitch).
						AriaLabel("Enable feature").
						AriaToggled(toggled).
						W(gpui.Px(44.)).
						H(gpui.Px(24.)).
						RoundedFull().
						CursorPointer().
						When(d.enabled, func(el *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return el.Bg(gpui.Rgb(0x89b4fa)) }).
						When(!d.enabled, func(el *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] { return el.Bg(gpui.Rgb(0x585b70)) }).
						Child(
							gpui.NewDiv().
								Size(gpui.Px(20.)).
								RoundedFull().
								Bg(gpui.White()).
								Mt(gpui.Px(2.)).
								When(d.enabled, func(el *gpui.Div) *gpui.Div { return el.Ml(gpui.Px(22.)) }).
								When(!d.enabled, func(el *gpui.Div) *gpui.Div { return el.Ml(gpui.Px(2.)) }),
						).
						OnClick(cx.Listener(func(this *A11yDemo, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[A11yDemo]) {
							this.enabled = !this.enabled
							cx.Notify()
						})),
				).
				Child(gpui.TextMacro("Enable feature")),
		).
		Child(
			gpui.NewDiv().
				ID("task-list").
				Role(a11y.RoleList).
				AriaLabel("Tasks").
				Flex().
				FlexCol().
				Gap1().
				Children(tasks...),
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

		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(400.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{
			Title:                gpui.Some[gpui.SharedString]("GPUI Accessibility Demo"),
			TrafficLightPosition: gpui.None[gpui.Point[gpui.Pixels]](),
		})
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[A11yDemo] {
			return cx.New(func(cx *gpui.Context[A11yDemo]) *A11yDemo { return NewA11yDemo(window, cx) })
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	runExample()
}
