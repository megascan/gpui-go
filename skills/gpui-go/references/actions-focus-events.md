# Actions, keymaps, focus and input events

## Contents
- Defining actions (hand-written)
- Generating actions (actiongen)
- Key bindings and contexts
- Handling actions on elements
- Focus handles
- Mouse and keyboard events
- Drag and drop
- Complete example: keyboard todo list
- Checklist when a binding does nothing

## Defining actions (hand-written)

An action is any type that implements `gpui.Action` (`Name() string`, `BoxedClone() gpui.Action`, `PartialEq(gpui.Action) bool`) and is registered from `init`:

```go
type Quit struct{}

func (Quit) Name() string { return "app::Quit" }

func (Quit) BoxedClone() gpui.Action { return Quit{} }

func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}
```

```go
func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "app::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}
```

- Imports: `encoding/json/jsontext` and `reflect`.
- Names use the form `namespace::Name`. A duplicate name panics when an App loads the registry.
- `ActionData` also has `JSONSchema`, `DeprecatedAliases`, `DeprecationMessage` and `Documentation`.

## Generating actions (actiongen)

```go
package acts

//go:generate go run github.com/megascan/gpui-go/internal/cmd/actiongen

//gpui:action namespace=todo
type AddItem struct{}

//gpui:action namespace=todo
type SelectItem struct {
	Index int `json:"index"`
}
```

`go generate ./...` writes `<file>_gen.go` with the three methods, a JSON `Build` and an `init` registration. Actions with JSON fields import the public `github.com/megascan/gpui-go/jsonschema`, so this works from outside the gpui-go module (verified with `go generate` + `go vet` in an external module). Other directive keys are `name=`, `no_json`, `no_register`, `deprecated_aliases=` and `deprecated=`.

## Key bindings and contexts

```go
func InstallKeymap(cx *gpui.App) {
	cx.BindKeys([]gpui.KeyBinding{
		gpui.NewKeyBinding("ctrl-q", Quit{}, gpui.None[string]()),
		gpui.NewKeyBinding("enter", Confirm{}, gpui.Some("Editor")),
	})
	cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
}
```

- A keystroke string looks like `"ctrl-shift-k"`, `"cmd-s"`, `"up"`, `"space"`, `"enter"` or `"escape"`. A space between keystrokes makes a sequence (`"g g"`).
- The third argument is a key-context predicate: `gpui.None[string]()` matches globally, `gpui.Some("Editor")` matches only inside an element with `KeyContext("Editor")`. Predicates can be expressions such as `"Editor && !readonly"`.
- `cx.OnAction` registers an app-level handler. It runs in the bubble phase after the element handlers on the focused path, unless one of them stops propagation. It suits quit, open-settings and similar global actions.
- Dispatch from code with `window.DispatchAction(action, cx)`, `focusHandle.DispatchAction(action, window, cx)` or `cx.DispatchAction(action)`. Inspect bindings with `window.BindingsForAction(action)` and `cx.KeyBindings()`.

## Handling actions on elements

Element `OnAction[A]` wants `func(action A, window *gpui.Window, cx *gpui.App)`, with the action by value. `cx.Listener` produces `func(*E, ...)`. The examples bridge the two with this adapter (copy it into your package):

```go
func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}
```

The handler is a method taking `*Action`: `func (e *Editor) confirm(_ *Confirm, _ *gpui.Window, cx *gpui.Context[Editor])`. Wire it up with `.OnAction(actionListener(cx.Listener((*Editor).confirm)))`.

- `CaptureAction[A]` runs in the capture phase, from the root down.
- Inside a handler, `cx.StopPropagation()` and `cx.Propagate()` control bubbling.

## Focus handles

```go
type Editor struct {
	text        string
	confirmed   int
	focused     bool
	focusHandle gpui.FocusHandle
	subs        []gpui.Subscription
}

func NewEditor(window *gpui.Window, cx *gpui.Context[Editor]) *Editor {
	focusHandle := cx.FocusHandle()
	return &Editor{
		focusHandle: focusHandle,
		subs: []gpui.Subscription{
			cx.OnFocus(focusHandle, window, func(this *Editor, _ *gpui.Window, cx *gpui.Context[Editor]) {
				this.focused = true
				cx.Notify()
			}),
			cx.OnBlur(focusHandle, window, func(this *Editor, _ *gpui.Window, cx *gpui.Context[Editor]) {
				this.focused = false
				cx.Notify()
			}),
		},
	}
}

func (e *Editor) Drop(_ *gpui.App) {
	for _, s := range e.subs {
		s.Unsubscribe()
	}
	e.focusHandle.Release()
}

func (e *Editor) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return e.focusHandle.Clone()
}
```

- Implementing `FocusHandle(cx *gpui.App) gpui.FocusHandle` makes the view `gpui.Focusable`. That enables `entity.FocusHandle(cx)`, `VisualTestContext.Focus(view)` and `cx.FocusView`. Return a `Clone()`, because the caller releases it.
- Focusing: `h.Focus(window, cx)`, `window.Focus(h, cx)`, `window.FocusNext(cx)`/`FocusPrev(cx)`. Query with `h.IsFocused(window)`, `h.ContainsFocused(window, cx)` and `window.Focused(cx)` (release it).
- Tab order: `cx.FocusHandle().WithTabIndex(1).WithTabStop(true)` on the handle, or `TabIndex(n)`/`TabStop(true)`/`TabGroup()` on the element.
- Focus the initial view in `OpenWindow`'s builder:

```go
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[TodoList] {
			list := cx.New(NewTodoList(items))
			focus := list.FocusHandle(cx)
			focus.Focus(window, cx)
			focus.Release()
			return list
		}); err != nil {
			panic(err)
		}
```

## Mouse and keyboard events

```go
func (e *Editor) Render(_ *gpui.Window, cx *gpui.Context[Editor]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("editor").
		KeyContext("Editor").
		TrackFocus(e.focusHandle).
		OnAction(actionListener(cx.Listener((*Editor).confirm))).
		OnKeyDown(cx.Listener(func(this *Editor, event *gpui.KeyDownEvent, _ *gpui.Window, cx *gpui.Context[Editor]) {
			if ch, ok := event.Keystroke.KeyChar.Get(); ok {
				this.text += ch
				cx.Notify()
			}
		})).
		OnMouseDown(gpui.MouseButtonLeft, cx.Listener(func(this *Editor, _ *gpui.MouseDownEvent, window *gpui.Window, cx *gpui.Context[Editor]) {
			this.focusHandle.Focus(window, cx.App)
		})).
		FocusVisible(func(s *gpui.StyleRefinement) *gpui.StyleRefinement {
			var st gpui.Styled[*gpui.StyleRefinement]
			gpui.BindStyled(&st, s)
			st.Border2()
			return st.BorderColor(gpui.Blue())
		}).
		P(gpui.Px(8)).
		ChildText(e.text)
}
```

- On any element: `OnMouseDown(button, f)`, `OnMouseUp`, `OnAnyMouseDown`, `OnMouseDownOut`, `OnMouseMove`, `OnMouseExit`, `OnScrollWheel`, `OnKeyDown`, `OnKeyUp`, `OnModifiersChanged`, `Occlude()`.
- After `ID`: `OnClick(func(*gpui.ClickEvent, *gpui.Window, *gpui.App))`, `OnAuxClick`, `OnHover(func(bool, ...))`.
- A `KeyDownEvent` carries `Keystroke` with `Key`, `KeyChar gpui.Option[string]` and `Modifiers`, plus `IsHeld`.
- Real text input with IME, selection and a caret needs `EntityInputHandler`. Port examples/input rather than collecting `OnKeyDown` characters.

## Drag and drop

```go
func (b *Board) Render(_ *gpui.Window, cx *gpui.Context[Board]) gpui.IntoElement {
	onDrop := cx.Listener(func(this *Board, card *Card, _ *gpui.Window, cx *gpui.Context[Board]) {
		this.dropped = append(this.dropped, card.Title)
		cx.Notify()
	})
	return gpui.NewDiv().
		Flex().
		Gap4().
		Child(
			gpui.NewDiv().
				ID("source").
				ChildText("drag me").
				OnDrag(Card{Title: "card"}, func(card Card, _ gpui.Point[gpui.Pixels], _ *gpui.Window, cx *gpui.App) gpui.Entity[Card] {
					return cx.New(func(_ *gpui.Context[Card]) *Card { return &card })
				}),
		).
		Child(
			gpui.NewDiv().
				Size32().
				DragOver(func(s *gpui.StyleRefinement, _ Card, _ *gpui.Window, _ *gpui.App) *gpui.StyleRefinement {
					var st gpui.Styled[*gpui.StyleRefinement]
					gpui.BindStyled(&st, s)
					return st.Bg(gpui.Green().Opacity(0.2))
				}).
				OnDrop(func(card Card, window *gpui.Window, cx *gpui.App) {
					onDrop(&card, window, cx)
				}),
		)
}
```

The drag preview is an entity whose `*T` implements `Render` (here, `*Card`). The value's type parameter must be identical in `OnDrag`, `DragOver` and `OnDrop`.

## Complete example: keyboard todo list

This compiles in an external module, and its tests pass under strict leak detection (the tests are in the async and testing reference). It shows actions, key context, focus, a `Drop`, a uniform_list fed from a render-time snapshot, a scroll handle that follows the selection, and per-row click handlers through a `WeakEntity`.

```go
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

type SelectNext struct{}

func (SelectNext) Name() string                 { return "todo::SelectNext" }
func (SelectNext) BoxedClone() gpui.Action      { return SelectNext{} }
func (SelectNext) PartialEq(o gpui.Action) bool { _, ok := o.(SelectNext); return ok }

type SelectPrev struct{}

func (SelectPrev) Name() string                 { return "todo::SelectPrev" }
func (SelectPrev) BoxedClone() gpui.Action      { return SelectPrev{} }
func (SelectPrev) PartialEq(o gpui.Action) bool { _, ok := o.(SelectPrev); return ok }

type ToggleDone struct{}

func (ToggleDone) Name() string                 { return "todo::ToggleDone" }
func (ToggleDone) BoxedClone() gpui.Action      { return ToggleDone{} }
func (ToggleDone) PartialEq(o gpui.Action) bool { _, ok := o.(ToggleDone); return ok }

func init() {
	gpui.RegisterAction(gpui.ActionData{Name: "todo::SelectNext", Type: reflect.TypeFor[SelectNext](),
		Build: func(jsontext.Value) (gpui.Action, error) { return SelectNext{}, nil }})
	gpui.RegisterAction(gpui.ActionData{Name: "todo::SelectPrev", Type: reflect.TypeFor[SelectPrev](),
		Build: func(jsontext.Value) (gpui.Action, error) { return SelectPrev{}, nil }})
	gpui.RegisterAction(gpui.ActionData{Name: "todo::ToggleDone", Type: reflect.TypeFor[ToggleDone](),
		Build: func(jsontext.Value) (gpui.Action, error) { return ToggleDone{}, nil }})
}

type Item struct {
	Title string
	Done  bool
}

type TodoList struct {
	items       []Item
	selected    int
	focusHandle gpui.FocusHandle
	scroll      gpui.UniformListScrollHandle
}

func NewTodoList(items []Item) func(cx *gpui.Context[TodoList]) *TodoList {
	return func(cx *gpui.Context[TodoList]) *TodoList {
		return &TodoList{
			items:       items,
			focusHandle: cx.FocusHandle(),
			scroll:      gpui.NewUniformListScrollHandle(),
		}
	}
}

func (t *TodoList) Drop(_ *gpui.App) {
	t.focusHandle.Release()
}

func (t *TodoList) FocusHandle(_ *gpui.App) gpui.FocusHandle {
	return t.focusHandle.Clone()
}

func (t *TodoList) selectNext(_ *SelectNext, _ *gpui.Window, cx *gpui.Context[TodoList]) {
	if t.selected < len(t.items)-1 {
		t.selected++
		t.scroll.ScrollToItem(t.selected, gpui.ScrollStrategyNearest)
		cx.Notify()
	}
}

func (t *TodoList) selectPrev(_ *SelectPrev, _ *gpui.Window, cx *gpui.Context[TodoList]) {
	if t.selected > 0 {
		t.selected--
		t.scroll.ScrollToItem(t.selected, gpui.ScrollStrategyNearest)
		cx.Notify()
	}
}

func (t *TodoList) toggleDone(_ *ToggleDone, _ *gpui.Window, cx *gpui.Context[TodoList]) {
	if t.selected < len(t.items) {
		t.items[t.selected].Done = !t.items[t.selected].Done
		cx.Notify()
	}
}

func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

func (t *TodoList) Render(_ *gpui.Window, cx *gpui.Context[TodoList]) gpui.IntoElement {
	items, selected, this := t.items, t.selected, cx.WeakEntity()
	return gpui.NewDiv().
		ID("todo-list").
		KeyContext("TodoList").
		TrackFocus(t.focusHandle).
		OnAction(actionListener(cx.Listener((*TodoList).selectNext))).
		OnAction(actionListener(cx.Listener((*TodoList).selectPrev))).
		OnAction(actionListener(cx.Listener((*TodoList).toggleDone))).
		Flex().
		FlexCol().
		SizeFull().
		Bg(gpui.Rgb(0x1e1e2e)).
		TextColor(gpui.Rgb(0xcdd6f4)).
		Child(
			gpui.NewUniformList(
				"items",
				len(t.items),
				func(visible gpui.Range, _ *gpui.Window, _ *gpui.App) []*gpui.Stateful[*gpui.Div] {
					rows := make([]*gpui.Stateful[*gpui.Div], 0, visible.End-visible.Start)
					for ix := visible.Start; ix < visible.End; ix++ {
						item := items[ix]
						rows = append(rows, gpui.NewDiv().
							DebugSelector(func() string { return fmt.Sprintf("row-%d", ix) }).
							ID(ix).
							Px2().
							Py1().
							CursorPointer().
							When(ix == selected, func(row *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
								return row.Bg(gpui.Rgb(0x313244))
							}).
							When(item.Done, func(row *gpui.Stateful[*gpui.Div]) *gpui.Stateful[*gpui.Div] {
								return row.LineThrough().TextColor(gpui.Rgb(0x6c7086))
							}).
							OnClick(func(_ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.App) {
								_ = this.Update(cx, func(list *TodoList, cx *gpui.Context[TodoList]) {
									list.selected = ix
									cx.Notify()
								})
							}).
							ChildText(item.Title),
						)
					}
					return rows
				},
			).
				TrackUniformListScroll(t.scroll).
				Flex1(),
		)
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("down", SelectNext{}, gpui.Some("TodoList")),
			gpui.NewKeyBinding("up", SelectPrev{}, gpui.Some("TodoList")),
			gpui.NewKeyBinding("space", ToggleDone{}, gpui.Some("TodoList")),
		})
		items := make([]Item, 0, 1000)
		for i := range 1000 {
			items = append(items, Item{Title: fmt.Sprintf("Task %d", i+1)})
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(360.), gpui.Px(480.)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[TodoList] {
			list := cx.New(NewTodoList(items))
			focus := list.FocusHandle(cx)
			focus.Focus(window, cx)
			focus.Release()
			return list
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
```

## Checklist when a binding does nothing

1. The action is registered (`RegisterAction` in `init`, or the generated file is committed), and its `Name()` matches.
2. The binding's context (`gpui.Some("X")`) equals an ancestor's `KeyContext("X")`.
3. That element has `TrackFocus(h)`, and `h` (or a descendant) is focused. In tests the window must be active for focus listeners, but dispatching works either way.
4. `OnAction` sits on the same element or an ancestor on the focused path. Handlers on siblings never see the action.
5. Nothing earlier calls `cx.StopPropagation()`, and no deeper context binds the same keystroke.
