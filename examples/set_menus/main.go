package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// SetMenus is the example's window content.
type SetMenus struct{}

// Render renders the window content.
func (s *SetMenus) Render(_ *gpui.Window, _ *gpui.Context[SetMenus]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		Bg(gpui.White()).
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		TextXl().
		TextColor(gpui.Black()).
		ChildText("Set Menus Example")
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.SetGlobal(NewAppState())
		cx.Activate(true)
		cx.OnAction(quit)
		cx.OnAction(toggleCheck)
		setAppMenus(cx)
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[SetMenus] {
			return cx.New(func(_ *gpui.Context[SetMenus]) *SetMenus { return &SetMenus{} })
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}

// ViewMode is how the example's content would be laid out.
type ViewMode uint8

const (
	// ViewModeList is the list layout.
	ViewModeList ViewMode = iota
	// ViewModeGrid is the grid layout.
	ViewModeGrid
)

// Toggle switches between the list and grid modes.
func (v *ViewMode) Toggle() {
	switch *v {
	case ViewModeList:
		*v = ViewModeGrid
	case ViewModeGrid:
		*v = ViewModeList
	}
}

// SharedString returns the mode's label.
func (v ViewMode) SharedString() gpui.SharedString {
	switch v {
	case ViewModeList:
		return "List"
	default:
		return "Grid"
	}
}

// AppState is the example's global state.
type AppState struct {
	viewMode ViewMode
}

// NewAppState creates the initial state.
func NewAppState() AppState {
	return AppState{viewMode: ViewModeList}
}

func setAppMenus(cx *gpui.App) {
	appState := cx.Global[AppState]()
	cx.SetMenus([]gpui.Menu{gpui.NewMenu("set_menus").WithItems(
		gpui.MenuItemOsSubmenu("Services", gpui.SystemMenuTypeServices),
		gpui.NewMenuItemSeparator(),
		gpui.NewMenuItemAction("Disabled Item", gpui.NoAction{}).WithDisabled(true),
		gpui.NewMenuItemSubmenu(gpui.NewMenu("Disabled Submenu").WithDisabled(true)),
		gpui.NewMenuItemSeparator(),
		gpui.NewMenuItemAction("List Mode", ToggleCheck{}).WithChecked(appState.viewMode == ViewModeList),
		gpui.NewMenuItemSubmenu(
			gpui.NewMenu("Mode").WithItems(
				gpui.NewMenuItemAction(string(ViewModeList.SharedString()), ToggleCheck{}).
					WithChecked(appState.viewMode == ViewModeList),
				gpui.NewMenuItemAction(string(ViewModeGrid.SharedString()), ToggleCheck{}).
					WithChecked(appState.viewMode == ViewModeGrid),
			),
		),
		gpui.NewMenuItemSeparator(),
		gpui.NewMenuItemAction("Quit", Quit{}),
	)})
}

// Quit quits the application.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "set_menus::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

// ToggleCheck toggles the view mode.
type ToggleCheck struct{}

// Name returns the action name.
func (ToggleCheck) Name() string { return "set_menus::ToggleCheck" }

// BoxedClone clones the action.
func (ToggleCheck) BoxedClone() gpui.Action { return ToggleCheck{} }

// PartialEq reports whether other is a ToggleCheck.
func (ToggleCheck) PartialEq(other gpui.Action) bool {
	_, ok := other.(ToggleCheck)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "set_menus::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "set_menus::ToggleCheck",
		Type:  reflect.TypeFor[ToggleCheck](),
		Build: func(jsontext.Value) (gpui.Action, error) { return ToggleCheck{}, nil },
	})
}

func quit(_ Quit, cx *gpui.App) {
	fmt.Println("Gracefully quitting the application...")
	cx.Quit()
}

func toggleCheck(_ ToggleCheck, cx *gpui.App) {
	appState := cx.GlobalMut[AppState]()
	appState.viewMode.Toggle()
	setAppMenus(cx)
}
