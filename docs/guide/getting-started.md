# Getting started: a counter app

In this tutorial you build a window with a number and two buttons. Clicking a button changes the number. Along the way you meet the four things every gpui-go app uses: the application, a window, a view entity and its `Render` method.

The finished program is [examples/counter/main.go](../../examples/counter/main.go). Run it any time with:

```sh
go run ./examples/counter
```

Before you start, check the [install steps](../../README.md#install) for your OS. On Linux you need cgo and a C compiler.

## 1. Open a window with a view

Create a new module that depends on gpui-go (see [Install](../../README.md#install)) and write this `main.go`:

```go
package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// Counter is the root view.
type Counter struct {
	count int
}

// Render renders the view.
func (c *Counter) Render(_ *gpui.Window, _ *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Text3xl().
		ChildText(fmt.Sprintf("%d", c.count))
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(320), gpui.Px(240)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		_, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
			return cx.New(func(_ *gpui.Context[Counter]) *Counter {
				return &Counter{}
			})
		})
		if err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
```

Run it with `go run .`. A window opens with a `0` in the middle.

Here is what each part does:

- `platform.Application()` picks the backend for your OS. Importing `platform` also locks the main goroutine to the main OS thread, which every desktop UI toolkit needs. Call `Run` from `main`.
- `Run` takes a callback that receives the `*gpui.App`. The `App` owns all state in your program. On desktop, `Run` blocks until the app quits.
- `gpui.DefaultWindowOptions()` returns the default options: the window is shown, focused, resizable and has a title bar. Start from it and change the fields you need. A bare `gpui.WindowOptions{}` literal leaves all of those off.
- `cx.OpenWindow` calls your builder once and makes the returned entity the window's root view.
- `cx.New` creates an *entity*: state that the `App` owns. You get back an `Entity[Counter]`, a handle to that state. The window takes over this handle, so you do not release it here.
- `Counter` is a *view* because `*Counter` has a `Render` method. GPUI calls `Render` whenever the window needs a new frame of this view. It returns a tree of elements built from `gpui.NewDiv()` and the style methods.

The style methods follow Tailwind names: `Flex`, `SizeFull`, `JustifyCenter`, `ItemsCenter`, `Text3xl`. Lengths take a unit, so you write `gpui.Px(320)`, not `320`.

## 2. Change state and notify

State only changes inside an update. Add a method that changes the count and tells GPUI the view needs to render again:

```go
func (c *Counter) add(delta int, cx *gpui.Context[Counter]) {
	c.count += delta
	cx.Notify()
}
```

`cx` is a `*gpui.Context[Counter]`, the context for this one entity. `Notify` marks the view dirty, so its window redraws and anyone observing the entity is told it changed. If you change state and forget `Notify`, the screen keeps showing the old value.

## 3. Add buttons

A clickable element needs an id. Write a helper that builds a styled button:

```go
func button(id string, label string) *gpui.Stateful[*gpui.Div] {
	return gpui.NewDiv().
		ID(id).
		Size10().
		Flex().
		ItemsCenter().
		JustifyCenter().
		RoundedMd().
		Bg(gpui.Rgb(0x313244)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		CursorPointer().
		ChildText(label)
}
```

`ID` turns the `*gpui.Div` into a `*gpui.Stateful[*gpui.Div]`. Only stateful elements have `OnClick`, `Active`, `Tooltip` and scrolling, because GPUI needs the id to remember the element between frames.

## 4. Handle clicks

Replace `Render` with a version that shows the count and the two buttons:

```go
func (c *Counter) Render(_ *gpui.Window, cx *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap4().
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Child(
			gpui.NewDiv().
				Text3xl().
				ChildText(fmt.Sprintf("%d", c.count)),
		).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(
					button("decrement", "-").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.add(-1, cx)
						})),
				).
				Child(
					button("increment", "+").
						OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
							this.add(1, cx)
						})),
				),
		)
}
```

Element callbacks run later, after `Render` has returned, so they cannot use the `c` and `cx` from `Render`. `cx.Listener` solves that. It wraps your function so that, when the click arrives, GPUI updates the entity and passes you its current state as `this` plus a fresh context. The listener holds the entity weakly, so it does not keep the view alive.

Run the program again. Clicking `+` and `-` now changes the number.

## 5. Give the window a title

The final program also sets a title. Add this line after `options.WindowBounds`:

```go
		options.Titlebar = gpui.Some(gpui.TitlebarOptions{Title: gpui.Some(gpui.SharedString("Counter"))})
```

`gpui.Some` builds a `gpui.Option`. gpui-go uses `Option[T]` wherever the Rust API has an `Option<T>` field.

## What you learned

- An app starts with `platform.Application().Run`.
- `cx.New` creates an entity. An entity whose pointer type has `Render` is a view.
- `Render` returns elements. GPUI calls it again after `cx.Notify()`.
- `ID` makes an element stateful, which gives it `OnClick`.
- `cx.Listener` turns a method on your view into an element callback.

## Next steps

- [Concepts](concepts.md) explains entities, contexts and the render cycle in depth.
- [Elements and styling](elements-and-styling.md) covers layout, colors, images and components.
- [Actions, keymaps and focus](actions-keymaps-focus.md) adds keyboard control.
- [Testing](testing.md) shows how to test this kind of view without opening a window.
