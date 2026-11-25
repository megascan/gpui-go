# Actions, keymaps and focus

Mouse input goes to the element under the pointer. Keyboard input goes to the focused element, as *actions*. This guide builds a keyboard-driven menu. The full code is in [snippets/actions](snippets/actions/), with a test in [menu_test.go](snippets/actions/menu_test.go).

For background on how dispatch works, read [docs/key-dispatch.md](../key-dispatch.md).

## Define actions

An action is a Go type that implements `gpui.Action` and is registered by name. The short way is the `actiongen` generator. Mark each struct with `//gpui:action` and add a `go:generate` line:

```go
package actions

//go:generate go run github.com/megascan/gpui-go/internal/cmd/actiongen

//gpui:action namespace=menu
type MoveUp struct{}

//gpui:action namespace=menu
type MoveDown struct{}

//gpui:action namespace=menu
type Confirm struct{}

//gpui:action namespace=menu
type Jump struct {
	Index int  `json:"index"`
	Wrap  bool `json:"wrap"`
}
```

Run `go generate ./...`. It writes `actions_gen.go` next to the file, with the `Action` methods and an `init` that registers each type. The names become `menu::MoveUp`, `menu::Jump` and so on. Fields are read from JSON when an action is built by name (see [Bind keys from data](#bind-keys-from-data)). Commit the generated file.

You can also write an action by hand. Implement the three methods and register the type from `init`:

```go
// Quit quits the application.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "app::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "app::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}
```

Two registrations with the same name panic when the app starts.

## Give the view a focus handle

Only the focused element and its ancestors receive actions. Create a `gpui.FocusHandle` when you create the view, release it in `Drop`, and implement `gpui.Focusable` so other code can focus the view:

```go
type Menu struct {
	items       []string
	selected    int
	confirmed   int
	focusHandle gpui.FocusHandle
}

// NewMenu creates a menu.
func NewMenu(cx *gpui.Context[Menu]) *Menu {
	return &Menu{
		items:       []string{"Open", "Save", "Close"},
		confirmed:   -1,
		focusHandle: cx.FocusHandle(),
	}
}

// Drop releases the menu's focus handle.
func (m *Menu) Drop(_ *gpui.App) {
	m.focusHandle.Release()
}

// FocusHandle returns a new handle to the menu's focus.
func (m *Menu) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return m.focusHandle.Clone()
}
```

A focus handle is counted like an entity handle: `Clone` makes another, `Release` drops one. `FocusHandle` returns a clone, so the caller releases it.

## Handle actions

Write one method per action, then attach them in `Render`:

```go
func (m *Menu) moveDown(_ *MoveDown, _ *gpui.Window, cx *gpui.Context[Menu]) {
	if m.selected < len(m.items)-1 {
		m.selected++
		cx.Notify()
	}
}
```

```go
func (m *Menu) Render(_ *gpui.Window, cx *gpui.Context[Menu]) gpui.IntoElement {
	list := gpui.NewDiv().
		KeyContext("Menu").
		TrackFocus(m.focusHandle).
		OnAction(actionListener(cx.Listener((*Menu).moveUp))).
		OnAction(actionListener(cx.Listener((*Menu).moveDown))).
		OnAction(actionListener(cx.Listener((*Menu).jump))).
		OnAction(actionListener(cx.Listener((*Menu).confirm))).
		Flex().
		FlexCol().
		SizeFull()
```

- `KeyContext("Menu")` names this part of the tree, so key bindings can target it.
- `TrackFocus` ties the element to the focus handle. The element keeps its own clone; your handle stays yours to release.
- `OnAction` registers a handler. The action type comes from the handler's parameter type.
- Actions bubble from the focused element up to the root. The first handler that matches stops the action. Call `cx.Propagate()` inside a handler to let ancestors see it too. `CaptureAction` registers a handler that runs on the way down, before any `OnAction`.

Element `OnAction` wants `func(action A, window, cx)` with the action by value, while `cx.Listener` produces `func(action *A, window, cx)`. This small adapter connects them, and the repo's examples use the same one:

```go
func actionListener[A any](listener func(action *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}
```

## Bind keys

Bind keystrokes to actions once, at startup. The third argument is the context the binding applies in:

```go
func BindMenuKeys(cx *gpui.App) {
	cx.BindKeys([]gpui.KeyBinding{
		gpui.NewKeyBinding("up", MoveUp{}, gpui.Some("Menu")),
		gpui.NewKeyBinding("down", MoveDown{}, gpui.Some("Menu")),
		gpui.NewKeyBinding("enter", Confirm{}, gpui.Some("Menu")),
		gpui.NewKeyBinding("home", Jump{Index: 0}, gpui.Some("Menu")),
		gpui.NewKeyBinding("ctrl-q", Quit{}, gpui.None[string]()),
	})
	cx.OnAction(func(_ Quit, cx *gpui.App) {
		cx.Quit()
	})
}
```

- Keystrokes are written like `ctrl-s`, `shift-tab`, `cmd-k`, `alt-enter`. Separate a sequence with spaces: `ctrl-k ctrl-s`.
- `gpui.None[string]()` makes a binding that applies everywhere.
- Context predicates support `&&`, `||`, `!`, `==`, `!=` and `>` (parent > child), for example `"Editor && mode == full"`.
- When several bindings match, the one whose context is deepest in the tree wins. At equal depth, the binding added last wins.
- `cx.OnAction` on the `App` registers a global handler. It runs when no element on the focus path handled the action.

## Focus the view

Nothing receives keys until something is focused. Focus the root view when you open the window:

```go
	_, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(window *gpui.Window, cx *gpui.App) gpui.Entity[Menu] {
		menu := cx.New(NewMenu)
		focusHandle := menu.FocusHandle(cx)
		focusHandle.Focus(window, cx)
		focusHandle.Release()
		return menu
	})
```

`entity.FocusHandle(cx)` works because `*Menu` implements `gpui.Focusable`. Inside the view itself, `cx.FocusSelf(window)` does the same.

Other focus tools:

- `window.Focus(handle, cx)` and `window.Blur(cx)` move focus.
- `handle.IsFocused(window)` and `handle.ContainsFocused(window, cx)` check it, for example to style the focused row.
- `cx.OnFocus`, `cx.OnBlur` and friends on `*gpui.Context[T]` subscribe to focus changes.
- `.Focus(style)` styles an element while it is focused. `.FocusVisible(style)` does the same only when focus came from the keyboard.

## Tab navigation

Give focusable elements a tab order with `FocusHandle.WithTabIndex(n).WithTabStop(true)` (or `TabIndex` and `TabStop` on the element), then bind Tab to an action that moves focus. From [examples/tab_stop](../../examples/tab_stop/main.go):

```go
func (e *Example) onTab(_ *Tab, window *gpui.Window, cx *gpui.Context[Example]) {
	window.FocusNext(cx.App)
	e.message = "You have pressed `Tab`."
}
```

```go
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("tab", Tab{}, gpui.None[string]()),
			gpui.NewKeyBinding("shift-tab", TabPrev{}, gpui.None[string]()),
		})
```

## Dispatch an action from code

- `window.DispatchAction(action, cx)` dispatches from the focused element, as if a key had been pressed.
- `handle.DispatchAction(action, window, cx)` dispatches from a specific focus handle.
- `cx.DispatchAction(action)` on the `App` dispatches in the active window.

Menu items, command palettes and buttons that mirror a shortcut usually call one of these.

## Bind keys from data

To load bindings from a file, build actions by name with `cx.BuildAction`. Actions with fields take a JSON value:

```go
		cx.Update(func(cx *gpui.App) {
			action, err = cx.BuildAction("menu::Jump", gpui.Some(jsontext.Value(`{"index": 2}`)))
		})
```

That returns a `menu.Jump{Index: 2}`. Pass the result to `gpui.NewKeyBinding` or dispatch it directly. gpui-go does not ship a keymap file format; [docs/key-dispatch.md](../key-dispatch.md#key-dispatch) shows the JSON layout Zed uses.

## Test it

You can test key handling without a real window:

```go
		menu, vcx := cx.AddWindowView(func(_ *gpui.Window, cx *gpui.Context[Menu]) *Menu {
			return NewMenu(cx)
		})
		defer menu.Release()
		vcx.Focus(menu)

		vcx.SimulateKeystrokes("down down up enter")
```

See [Testing](testing.md) for the rest.
