package actions

import (
	"fmt"

	"github.com/megascan/gpui-go"
)

// Menu is a keyboard-driven list.
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

func actionListener[A any](listener func(action *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

func (m *Menu) moveUp(_ *MoveUp, _ *gpui.Window, cx *gpui.Context[Menu]) {
	if m.selected > 0 {
		m.selected--
		cx.Notify()
	}
}

func (m *Menu) moveDown(_ *MoveDown, _ *gpui.Window, cx *gpui.Context[Menu]) {
	if m.selected < len(m.items)-1 {
		m.selected++
		cx.Notify()
	}
}

func (m *Menu) jump(action *Jump, _ *gpui.Window, cx *gpui.Context[Menu]) {
	m.selected = max(0, min(action.Index, len(m.items)-1))
	cx.Notify()
}

func (m *Menu) confirm(_ *Confirm, _ *gpui.Window, cx *gpui.Context[Menu]) {
	m.confirmed = m.selected
	cx.Notify()
}

// Render renders the view.
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
	for ix, item := range m.items {
		list = list.Child(
			gpui.NewDiv().
				Px2().
				When(ix == m.selected, func(this *gpui.Div) *gpui.Div {
					return this.Bg(gpui.Rgb(0x45475a))
				}).
				ChildText(fmt.Sprintf("%d. %s", ix+1, item)),
		)
	}
	return list
}

// BindMenuKeys registers the key bindings and app-level handlers.
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

// OpenMenu opens a window with a focused menu.
func OpenMenu(cx *gpui.App) error {
	_, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(window *gpui.Window, cx *gpui.App) gpui.Entity[Menu] {
		menu := cx.New(NewMenu)
		focusHandle := menu.FocusHandle(cx)
		focusHandle.Focus(window, cx)
		focusHandle.Release()
		return menu
	})
	if err != nil {
		return fmt.Errorf("open menu window: %w", err)
	}
	return nil
}
