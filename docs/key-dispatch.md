# Key dispatch

GPUI is built for keyboard-first interaction.

To make something available to the mouse, you render a button with a click handler.

To make something available to the keyboard, you bind an _action_ in a _key context_.

Actions work like the built-in events (`MouseDownEvent`, `KeyDownEvent`, ...), but you define them yourself. Mark a struct with the `//gpui:action` directive and run `go generate`; `actiongen` writes the `Action` methods and registers the type from `init`:

```go
package menu

//go:generate go run github.com/megascan/gpui-go/internal/cmd/actiongen

//gpui:action namespace=menu
type MoveUp struct{}

//gpui:action namespace=menu
type MoveDown struct{}
```

Most actions are empty structs like these. You can also write the three methods (`Name`, `BoxedClone`, `PartialEq`) by hand and call `gpui.RegisterAction` from `init`.

Actions can carry data. Exported fields are read from the action's JSON:

```go
//gpui:action namespace=menu
type Move struct {
	Direction Direction `json:"direction"`
	Select    bool      `json:"select"`
}
```

To handle an action, chain `OnAction` on an element. `cx.Listener` turns a method of your view into a listener:

```go
func (m *Menu) Render(window *gpui.Window, cx *gpui.Context[Menu]) gpui.IntoElement {
	return gpui.NewDiv().
		OnAction(func(action menu.MoveUp, window *gpui.Window, cx *gpui.App) {
			// ...
		}).
		OnAction(func(action menu.MoveDown, window *gpui.Window, cx *gpui.App) {
			// ...
		}).
		Children(m.items()...)
}
```

To bind keys to these actions, declare a _key context_ for part of the element tree with `KeyContext`:

```go
func (m *Menu) Render(window *gpui.Window, cx *gpui.Context[Menu]) gpui.IntoElement {
	return gpui.NewDiv().
		KeyContext("menu").
		OnAction(func(action menu.MoveUp, window *gpui.Window, cx *gpui.App) {
			// ...
		}).
		OnAction(func(action menu.MoveDown, window *gpui.Window, cx *gpui.App) {
			// ...
		}).
		Children(m.items()...)
}
```

Now bindings can target that context:

```go
cx.BindKeys([]gpui.KeyBinding{
	gpui.NewKeyBinding("up", menu.MoveUp{}, gpui.Some("menu")),
	gpui.NewKeyBinding("down", menu.MoveDown{}, gpui.Some("menu")),
})
```

Applications that load keymaps from JSON, as Zed does, name each action by its qualified name (`namespace::Type`) and build it with `cx.BuildAction(name, data)`:

```json
{
  "context": "menu",
  "bindings": {
    "up": "menu::MoveUp",
    "down": "menu::MoveDown"
  }
}
```

An action with fields is written as its name followed by its JSON value:

```json
{
  "context": "menu",
  "bindings": {
    "up": ["menu::Move", {"direction": "up", "select": false}],
    "down": ["menu::Move", {"direction": "down", "select": false}],
    "shift-up": ["menu::Move", {"direction": "up", "select": true}],
    "shift-down": ["menu::Move", {"direction": "down", "select": true}]
  }
}
```
