// Command move_entity_between_windows registers callbacks on an entity via the In API family
// and then re-hosts the entity in a new window via a click. Callbacks dispatched after the move
// target the entity's current window rather than the window it was in at registration time.
package main

import (
	"fmt"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// MoveToNewWindow asks the entity to move to a new window.
type MoveToNewWindow struct{}

// HelloWorld counts the ticks and moves it observes.
type HelloWorld struct {
	text          gpui.SharedString
	tickCount     uint32
	moveCount     uint32
	tasks         []*gpui.Task[struct{}]
	subscriptions []gpui.Subscription
}

// NewHelloWorld creates the view, its tick task and its move subscription.
func NewHelloWorld(window *gpui.Window, cx *gpui.Context[HelloWorld]) *HelloWorld {
	selfEntity := cx.Entity()
	defer selfEntity.Release()
	task := cx.SpawnIn[struct{}](window, func(this gpui.WeakEntity[HelloWorld], cx *gpui.AsyncWindowContext) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(time.Second).Await(cx); err != nil {
				return struct{}{}, err
			}
			err := this.UpdateIn(cx, func(this *HelloWorld, window *gpui.Window, _ *gpui.Context[HelloWorld]) {
				this.tickCount++
				fmt.Printf(
					"tick #%d fired in entity's current window %d\n",
					this.tickCount,
					window.WindowHandle().WindowID().AsU64(),
				)
			})
			if err != nil {
				fmt.Printf("tick task giving up: %v\n", err)
				return struct{}{}, nil
			}
		}
	})

	subscription := cx.SubscribeIn[HelloWorld, MoveToNewWindow](
		selfEntity,
		window,
		func(this *HelloWorld, _ gpui.Entity[HelloWorld], _ *MoveToNewWindow, window *gpui.Window, cx *gpui.Context[HelloWorld]) {
			enteredWindowID := window.WindowHandle().WindowID().AsU64()
			fmt.Printf("MoveToNewWindow handler fired in entity's current window %d\n", enteredWindowID)
			this.moveCount++
			cx.Notify()

			entity := cx.Entity()
			oldWindow := window.WindowHandle()
			cx.Defer(func(cx *gpui.App) {
				bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.0), gpui.Px(500.0)), cx)
				options := gpui.DefaultWindowOptions()
				options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
				if _, err := cx.OpenWindow(options, func(*gpui.Window, *gpui.App) gpui.Entity[HelloWorld] {
					return entity
				}); err != nil {
					panic(fmt.Sprintf("failed to open new window: %v", err))
				}
				_ = oldWindow.Update(cx, func(_ gpui.AnyView, window *gpui.Window, _ *gpui.App) {
					window.RemoveWindow()
				})
			})
		},
	)

	return &HelloWorld{
		text:          "World",
		tickCount:     0,
		moveCount:     0,
		tasks:         []*gpui.Task[struct{}]{task},
		subscriptions: []gpui.Subscription{subscription},
	}
}

// Render renders the view.
func (h *HelloWorld) Render(window *gpui.Window, cx *gpui.Context[HelloWorld]) gpui.IntoElement {
	windowID := window.WindowHandle().WindowID().AsU64()

	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText(fmt.Sprintf("Hello, %s!", h.text)).
		ChildText(fmt.Sprintf("Rendering in window: %d", windowID)).
		ChildText(fmt.Sprintf("Ticks observed by entity: %d", h.tickCount)).
		ChildText(fmt.Sprintf("Moves observed by entity: %d", h.moveCount)).
		Child(
			gpui.NewDiv().
				Px4().
				Py2().
				Bg(gpui.Rgb(0x4040ff)).
				RoundedMd().
				ChildText("Move me to a new window").
				OnMouseDown(
					gpui.MouseButtonLeft,
					cx.Listener[gpui.MouseDownEvent](func(_ *HelloWorld, _ *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
						cx.Emit[MoveToNewWindow](MoveToNewWindow{})
					}),
				),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.0), gpui.Px(500.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(cx *gpui.Context[HelloWorld]) *HelloWorld { return NewHelloWorld(window, cx) })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
