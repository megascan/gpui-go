//go:build !js

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/megascan/gpui-go"
)

// Todo is one todo item.
type Todo struct {
	message string
}

// Todos is the list of todos and the window that shows them, if any.
type Todos struct {
	items  []gpui.Entity[Todo]
	window gpui.Option[gpui.AnyWindowHandle]
}

// TodoWindow is the view of the todos window.
type TodoWindow struct {
	todos             gpui.Entity[Todos]
	todosSubscription gpui.Subscription
}

// Render renders the todos.
func (w *TodoWindow) Render(_ *gpui.Window, cx *gpui.Context[TodoWindow]) gpui.IntoElement {
	items := w.todos.Read(cx.App).items
	rows := make([]gpui.IntoElement, 0, len(items))
	for index, todo := range items {
		rows = append(rows, gpui.NewDiv().ChildText(describeTodo(index, todo, cx.App)))
	}

	return gpui.NewDiv().
		SizeFull().
		Bg(gpui.Rgb(0xffffff)).
		TextColor(gpui.Rgb(0x202020)).
		Flex().
		FlexCol().
		Gap2().
		P4().
		ChildText("Switchable display todos").
		ChildText("Use the terminal: create [message], ls, close, open, quit").
		When(len(rows) == 0, func(view *gpui.Div) *gpui.Div {
			return view.ChildText("No todos yet. Type `create` in the terminal.")
		}).
		Children(rows...)
}

var sampleTodos = [5]string{
	"milk / 牛奶 / 牛乳 / 우유 🥛",
	"apple / 苹果 / りんご / 사과 🍎",
	"cat / 猫 / ねこ / 고양이 🐱",
	"book / 书 / 本 / 책 📚",
	"rain / 雨 / あめ / 비 🌧️",
}

func entityIDDebug(id gpui.EntityID) string {
	if uint32(id) == ^uint32(0) {
		return "EntityId(null)"
	}
	return fmt.Sprintf("EntityId(%dv%d)", uint32(id), uint32(id>>32)|1)
}

func describeTodo(index int, todo gpui.Entity[Todo], cx *gpui.App) string {
	todoReference := todo.Read(cx)
	return fmt.Sprintf(
		"%d - %s - %s @%p",
		index,
		todoReference.message,
		entityIDDebug(todo.EntityID()),
		todoReference,
	)
}

func printTodos(todos gpui.Entity[Todos], cx *gpui.App) {
	if len(todos.Read(cx).items) == 0 {
		fmt.Println("no todos")
	}
	for index, todo := range todos.Read(cx).items {
		fmt.Println(describeTodo(index, todo, cx))
	}
}

func openWindow(todos gpui.Entity[Todos], cx *gpui.App) error {
	if todos.Read(cx).window.IsSome() {
		return nil
	}
	options := gpui.DefaultWindowOptions()
	options.Titlebar = gpui.Some(gpui.TitlebarOptions{
		Title: gpui.Some[gpui.SharedString]("Switchable display todos"),
	})
	handle, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[TodoWindow] {
		return cx.New(func(cx *gpui.Context[TodoWindow]) *TodoWindow {
			return &TodoWindow{
				todos: todos.Clone(),
				todosSubscription: cx.Observe[Todos](todos, func(_ *TodoWindow, _ gpui.Entity[Todos], cx *gpui.Context[TodoWindow]) {
					cx.Notify()
				}),
			}
		})
	})
	if err != nil {
		return err
	}
	cx.Activate(true)
	if err := handle.Update(cx, func(_ *TodoWindow, window *gpui.Window, _ *gpui.Context[TodoWindow]) {
		window.ActivateWindow()
	}); err != nil {
		return err
	}
	todos.Update(cx, func(todos *Todos, _ *gpui.Context[Todos]) {
		todos.window = gpui.Some(handle.AnyWindowHandle)
	})
	return nil
}

func handleCommand(command string, todos gpui.Entity[Todos], cx *gpui.App) (gpui.Option[gpui.WindowingRequest], error) {
	name, arguments, found := strings.Cut(command, " ")
	if !found {
		name, arguments = command, ""
	}
	switch name {
	case "ls":
		printTodos(todos, cx)
	case "create":
		count := len(todos.Read(cx).items)
		sample := sampleTodos[count%len(sampleTodos)]
		message := sample
		if strings.TrimSpace(arguments) != "" {
			message = fmt.Sprintf("%s · %s", arguments, sample)
		}
		todo := cx.New(func(_ *gpui.Context[Todo]) *Todo { return &Todo{message: message} })
		todos.Update(cx, func(todos *Todos, cx *gpui.Context[Todos]) {
			todos.items = append(todos.items, todo)
			fmt.Printf("todo added, total %d\n", len(todos.items))
			cx.Notify()
		})
	case "open":
		environment, err := displayEnvironment(arguments)
		if err != nil {
			return gpui.None[gpui.WindowingRequest](), err
		}
		return gpui.Some[gpui.WindowingRequest](gpui.WindowingRequestWindowed{Environment: environment}), nil
	case "close":
		var window gpui.Option[gpui.AnyWindowHandle]
		todos.Update(cx, func(todos *Todos, _ *gpui.Context[Todos]) {
			window = todos.window
			todos.window = gpui.None[gpui.AnyWindowHandle]()
		})
		if handle, ok := window.Get(); ok {
			if err := handle.Update(cx, func(_ gpui.AnyView, window *gpui.Window, _ *gpui.App) {
				window.RemoveWindow()
			}); err != nil {
				return gpui.None[gpui.WindowingRequest](), err
			}
		}
		return gpui.Some[gpui.WindowingRequest](gpui.WindowingRequestHeadless{}), nil
	case "quit":
		cx.Quit()
	case "":
	default:
		fmt.Println(usage)
	}
	return gpui.None[gpui.WindowingRequest](), nil
}

func runCommand(command string, todos gpui.Entity[Todos], cx *gpui.AsyncApp) error {
	type handled struct {
		request gpui.Option[gpui.WindowingRequest]
		err     error
	}
	result := cx.UpdateR[handled](func(cx *gpui.App) handled {
		request, err := handleCommand(command, todos, cx)
		return handled{request: request, err: err}
	})
	if result.err != nil {
		return result.err
	}
	request, ok := result.request.Get()
	if !ok {
		return nil
	}
	isWindowed := func() bool {
		return cx.UpdateR[bool](func(cx *gpui.App) bool {
			_, ok := cx.GraphicalEnvironment()
			return ok
		})
	}
	switch request.(type) {
	case gpui.WindowingRequestHeadless:
		if isWindowed() {
			task := cx.UpdateR[*gpui.Task[struct{}]](func(cx *gpui.App) *gpui.Task[struct{}] { return cx.RequestWindowing(request) })
			if _, err := task.Await(cx); err != nil {
				return err
			}
		}
	case gpui.WindowingRequestWindowed:
		if !isWindowed() {
			task := cx.UpdateR[*gpui.Task[struct{}]](func(cx *gpui.App) *gpui.Task[struct{}] { return cx.RequestWindowing(request) })
			if _, err := task.Await(cx); err != nil {
				return err
			}
		}
		if err := cx.UpdateR[error](func(cx *gpui.App) error { return openWindow(todos, cx) }); err != nil {
			return err
		}
	}
	mode := cx.UpdateR[string](func(cx *gpui.App) string {
		if _, ok := cx.GraphicalEnvironment(); !ok {
			return "headless"
		}
		switch compositor := cx.CompositorName(); compositor {
		case "":
			return "windowed"
		default:
			return compositor
		}
	})
	fmt.Printf("mode: %s\n", mode)
	return nil
}

func errorChain(err error) string {
	var b strings.Builder
	b.WriteString(err.Error())
	for inner := errors.Unwrap(err); inner != nil; inner = errors.Unwrap(inner) {
		if strings.HasSuffix(b.String(), inner.Error()) {
			break
		}
		b.WriteString(": ")
		b.WriteString(inner.Error())
	}
	return b.String()
}

func main() {
	application := newApplication()
	application.
		WithWindowing(gpui.WindowingRequestHeadless{}).
		WithQuitMode(gpui.QuitModeExplicit).
		Run(func(cx *gpui.App) {
			todos := cx.New(func(_ *gpui.Context[Todos]) *Todos {
				return &Todos{
					items:  nil,
					window: gpui.None[gpui.AnyWindowHandle](),
				}
			})
			windowClosedSubscription := func() gpui.Subscription {
				todos := todos.Clone()
				return cx.OnWindowClosed(func(cx *gpui.App, windowID gpui.WindowID) {
					todos.Update(cx, func(todos *Todos, _ *gpui.Context[Todos]) {
						if window, ok := todos.window.Get(); ok && window.WindowID() == windowID {
							todos.window = gpui.None[gpui.AnyWindowHandle]()
						}
					})
				})
			}()

			commands := make(chan string, 1024)
			go func() {
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					commands <- scanner.Text()
				}
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "stdin error: %v\n", err)
				}
			}()
			fmt.Println(usage)
			cx.Spawn[struct{}](func(cx *gpui.AsyncApp) (struct{}, error) {
				defer windowClosedSubscription.Unsubscribe()
				for {
					if _, err := cx.BackgroundExecutor().Timer(25 * time.Millisecond).Await(cx); err != nil {
						return struct{}{}, err
					}
				drain:
					for {
						select {
						case command := <-commands:
							if err := runCommand(command, todos, cx); err != nil {
								fmt.Printf("error: %s\n", errorChain(err))
							}
						default:
							break drain
						}
					}
				}
			}).Detach()
		})
}
