// Command testing demonstrates GPUI's testing infrastructure.
//
// When run normally, this displays an interactive counter window. The tests in main_test.go
// demonstrate various GPUI testing patterns.
//
// Run the app: go run ./examples/testing
// Run tests:   go test ./examples/testing
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Increment increments the counter.
type Increment struct{}

// Name returns the action name.
func (Increment) Name() string { return "counter::Increment" }

// BoxedClone clones the action.
func (Increment) BoxedClone() gpui.Action { return Increment{} }

// PartialEq reports whether other is an Increment.
func (Increment) PartialEq(other gpui.Action) bool {
	_, ok := other.(Increment)
	return ok
}

// Decrement decrements the counter.
type Decrement struct{}

// Name returns the action name.
func (Decrement) Name() string { return "counter::Decrement" }

// BoxedClone clones the action.
func (Decrement) BoxedClone() gpui.Action { return Decrement{} }

// PartialEq reports whether other is a Decrement.
func (Decrement) PartialEq(other gpui.Action) bool {
	_, ok := other.(Decrement)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "counter::Increment",
		Type:  reflect.TypeFor[Increment](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Increment{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "counter::Decrement",
		Type:  reflect.TypeFor[Decrement](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Decrement{}, nil },
	})
}

// Counter is the testing example view.
type Counter struct {
	count        int32
	focusHandle  gpui.FocusHandle
	subscription gpui.Subscription
}

// CounterEvent is the event emitted by Counter.
type CounterEvent struct{}

// NewCounter creates a counter.
func NewCounter(cx *gpui.Context[Counter]) *Counter {
	subscription := cx.SubscribeSelf(func(this *Counter, _ *CounterEvent, _ *gpui.Context[Counter]) {
		this.count = 999
	})

	return &Counter{
		count:        0,
		focusHandle:  cx.FocusHandle(),
		subscription: subscription,
	}
}

// Drop releases the counter's handles.
func (c *Counter) Drop(_ *gpui.App) {
	c.subscription.Unsubscribe()
	c.focusHandle.Release()
}

func (c *Counter) increment(_ *Increment, _ *gpui.Window, cx *gpui.Context[Counter]) {
	c.count++
	cx.Notify()
}

func (c *Counter) decrement(_ *Decrement, _ *gpui.Window, cx *gpui.Context[Counter]) {
	c.count--
	cx.Notify()
}

func (c *Counter) load(cx *gpui.Context[Counter]) *gpui.Task[struct{}] {
	return cx.Spawn(func(this gpui.WeakEntity[Counter], cx *gpui.AsyncApp) (struct{}, error) {
		_ = this.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
			counter.count = 100
			cx.Notify()
		})
		return struct{}{}, nil
	})
}

func (c *Counter) reload(cx *gpui.Context[Counter]) {
	cx.Spawn(func(this gpui.WeakEntity[Counter], cx *gpui.AsyncApp) (struct{}, error) {
		_ = this.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
			counter.count += 50
			cx.Notify()
		})
		return struct{}{}, nil
	}).Detach()
}

// FocusHandle returns the counter's focus handle.
func (c *Counter) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return c.focusHandle.Clone()
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

func hoverBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x45475a))
}

func button(id string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Px4().
		Py2().
		Bg(gpui.Rgb(0x313244)).
		Hover(hoverBg).
		RoundedMd().
		CursorPointer().
		TextColor(gpui.Rgb(0xcdd6f4))
}

// Render renders the view.
func (c *Counter) Render(_ *gpui.Window, cx *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("counter").
		KeyContext("Counter").
		OnAction(actionListener(cx.Listener((*Counter).increment))).
		OnAction(actionListener(cx.Listener((*Counter).decrement))).
		TrackFocus(c.focusHandle).
		Flex().
		FlexCol().
		Gap4().
		Bg(gpui.Rgb(0x1e1e2e)).
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Child(
			gpui.NewDiv().
				Text3xl().
				TextColor(gpui.Rgb(0xcdd6f4)).
				ChildText(fmt.Sprintf("%d", c.count)),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(
					button("decrement").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[Counter]) {
							this.decrement(&Decrement{}, window, cx)
						})).
						ChildText("−"),
				).
				Child(
					button("increment").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[Counter]) {
							this.increment(&Increment{}, window, cx)
						})).
						ChildText("+"),
				),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(
					button("load").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.load(cx).Detach()
						})).
						ChildText("Load"),
				).
				Child(
					button("reload").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.reload(cx)
						})).
						ChildText("Reload"),
				),
		).
		Child(
			gpui.NewDiv().
				TextSm().
				TextColor(gpui.Rgb(0x6c7086)).
				ChildText("Press ↑/↓ or click buttons"),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("up", Increment{}, gpui.Some("Counter")),
			gpui.NewKeyBinding("down", Decrement{}, gpui.Some("Counter")),
		})

		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.), gpui.Px(200.)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
			counter := cx.New(NewCounter)
			focusHandle := counter.FocusHandle(cx)
			focusHandle.Focus(window, cx)
			focusHandle.Release()
			return counter
		}); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
	})
}

func main() {
	runExample()
}
