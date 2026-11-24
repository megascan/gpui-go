package gpui

import "slices"

// Menu is a menu of the application, either a main menu or a submenu.
type Menu struct {
	// Name is the name of the menu.
	Name SharedString
	// Items are the items in the menu.
	Items []MenuItem
	// Disabled reports whether the menu is disabled.
	Disabled bool
}

// NewMenu creates a menu with the given name.
func NewMenu(name string) Menu {
	return Menu{
		Name:     SharedString(name),
		Items:    nil,
		Disabled: false,
	}
}

// WithItems sets the items of the menu (D-WITH).
func (m Menu) WithItems(items ...MenuItem) Menu {
	m.Items = slices.Clone(items)
	return m
}

// WithDisabled sets whether the menu is disabled (D-WITH).
func (m Menu) WithDisabled(disabled bool) Menu {
	m.Disabled = disabled
	return m
}

// Owned creates an OwnedMenu from this menu.
func (m Menu) Owned() OwnedMenu {
	items := make([]OwnedMenuItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, item.Owned())
	}
	return OwnedMenu{
		Name:     m.Name,
		Items:    items,
		Disabled: m.Disabled,
	}
}

// OsMenu is a menu recognized by the operating system, which may provide specialized items for it.
type OsMenu struct {
	// Name is the name of the menu.
	Name SharedString
	// MenuType is the type of menu.
	MenuType SystemMenuType
}

// Owned creates an OwnedOsMenu from this menu.
func (m OsMenu) Owned() OwnedOsMenu {
	return OwnedOsMenu{
		Name:     m.Name,
		MenuType: m.MenuType,
	}
}

// SystemMenuType is the type of a system menu.
type SystemMenuType uint8

const (
	// SystemMenuTypeServices is the Services menu in the Application menu on macOS.
	SystemMenuTypeServices SystemMenuType = iota
)

// MenuItem is an item of a menu: MenuItemSeparator, MenuItemSubmenu, MenuItemSystemMenu or
// MenuItemAction.
type MenuItem interface {
	// Owned creates an OwnedMenuItem from this item.
	Owned() OwnedMenuItem
	// WithChecked sets whether the item is checked; only MenuItemAction is affected (D-WITH).
	WithChecked(checked bool) MenuItem
	// IsChecked reports whether the item is checked; false for anything but MenuItemAction.
	IsChecked() bool
	// WithDisabled sets whether the item is disabled; only MenuItemAction and MenuItemSubmenu
	// are affected (D-WITH).
	WithDisabled(disabled bool) MenuItem
	// IsDisabled reports whether the item is disabled; false for anything but MenuItemAction
	// and MenuItemSubmenu.
	IsDisabled() bool
	isMenuItem()
}

// MenuItemSeparator is a separator between items.
type MenuItemSeparator struct{}

// MenuItemSubmenu is a submenu.
type MenuItemSubmenu struct {
	Menu
}

// MenuItemSystemMenu is a menu managed by the system, for example the Services menu on macOS.
type MenuItemSystemMenu struct {
	OsMenu
}

// MenuItemAction is an item that performs an action.
type MenuItemAction struct {
	// Name is the name of the item.
	Name SharedString
	// Action is the action performed when the item is selected.
	Action Action
	// OsAction is the OS action that corresponds to this action, if any.
	OsAction Option[OsAction]
	// Checked reports whether the item is checked.
	Checked bool
	// Disabled reports whether the item is disabled.
	Disabled bool
}

func (MenuItemSeparator) isMenuItem()  {}
func (MenuItemSubmenu) isMenuItem()    {}
func (MenuItemSystemMenu) isMenuItem() {}
func (MenuItemAction) isMenuItem()     {}

// NewMenuItemSeparator creates a separator item.
func NewMenuItemSeparator() MenuItem {
	return MenuItemSeparator{}
}

// NewMenuItemSubmenu creates a submenu item.
func NewMenuItemSubmenu(menu Menu) MenuItem {
	return MenuItemSubmenu{menu}
}

// MenuItemOsSubmenu creates a submenu that the OS populates.
func MenuItemOsSubmenu(name string, menuType SystemMenuType) MenuItem {
	return MenuItemSystemMenu{OsMenu{
		Name:     SharedString(name),
		MenuType: menuType,
	}}
}

// NewMenuItemAction creates an item that invokes an action.
func NewMenuItemAction(name string, action Action) MenuItem {
	return MenuItemAction{
		Name:     SharedString(name),
		Action:   action,
		OsAction: None[OsAction](),
		Checked:  false,
		Disabled: false,
	}
}

// MenuItemOsAction creates an item that invokes an action and has an OS action.
func MenuItemOsAction(name string, action Action, osAction OsAction) MenuItem {
	return MenuItemAction{
		Name:     SharedString(name),
		Action:   action,
		OsAction: Some(osAction),
		Checked:  false,
		Disabled: false,
	}
}

// Owned creates an OwnedMenuItem from this item.
func (MenuItemSeparator) Owned() OwnedMenuItem {
	return OwnedMenuItemSeparator{}
}

// Owned creates an OwnedMenuItem from this item.
func (m MenuItemSubmenu) Owned() OwnedMenuItem {
	return OwnedMenuItemSubmenu{m.Menu.Owned()}
}

// Owned creates an OwnedMenuItem from this item.
func (m MenuItemSystemMenu) Owned() OwnedMenuItem {
	return OwnedMenuItemSystemMenu{m.OsMenu.Owned()}
}

// Owned creates an OwnedMenuItem from this item.
func (m MenuItemAction) Owned() OwnedMenuItem {
	return OwnedMenuItemAction{
		Name:     string(m.Name),
		Action:   m.Action,
		OsAction: m.OsAction,
		Checked:  m.Checked,
		Disabled: m.Disabled,
	}
}

// WithChecked returns the item unchanged.
func (m MenuItemSeparator) WithChecked(checked bool) MenuItem { return m }

// WithChecked returns the item unchanged.
func (m MenuItemSubmenu) WithChecked(checked bool) MenuItem { return m }

// WithChecked returns the item unchanged.
func (m MenuItemSystemMenu) WithChecked(checked bool) MenuItem { return m }

// WithChecked sets whether the item is checked.
func (m MenuItemAction) WithChecked(checked bool) MenuItem {
	m.Checked = checked
	return m
}

// IsChecked returns false.
func (MenuItemSeparator) IsChecked() bool { return false }

// IsChecked returns false.
func (MenuItemSubmenu) IsChecked() bool { return false }

// IsChecked returns false.
func (MenuItemSystemMenu) IsChecked() bool { return false }

// IsChecked reports whether the item is checked.
func (m MenuItemAction) IsChecked() bool { return m.Checked }

// WithDisabled returns the item unchanged.
func (m MenuItemSeparator) WithDisabled(disabled bool) MenuItem { return m }

// WithDisabled sets whether the submenu is disabled.
func (m MenuItemSubmenu) WithDisabled(disabled bool) MenuItem {
	m.Menu.Disabled = disabled
	return m
}

// WithDisabled returns the item unchanged.
func (m MenuItemSystemMenu) WithDisabled(disabled bool) MenuItem { return m }

// WithDisabled sets whether the item is disabled.
func (m MenuItemAction) WithDisabled(disabled bool) MenuItem {
	m.Disabled = disabled
	return m
}

// IsDisabled returns false.
func (MenuItemSeparator) IsDisabled() bool { return false }

// IsDisabled reports whether the submenu is disabled.
func (m MenuItemSubmenu) IsDisabled() bool { return m.Menu.Disabled }

// IsDisabled returns false.
func (MenuItemSystemMenu) IsDisabled() bool { return false }

// IsDisabled reports whether the item is disabled.
func (m MenuItemAction) IsDisabled() bool { return m.Disabled }

// OwnedOsMenu is the owned form of an OsMenu.
type OwnedOsMenu struct {
	// Name is the name of the menu.
	Name SharedString
	// MenuType is the type of menu.
	MenuType SystemMenuType
}

// OwnedMenu is the owned form of a Menu.
type OwnedMenu struct {
	// Name is the name of the menu.
	Name SharedString
	// Items are the items in the menu.
	Items []OwnedMenuItem
	// Disabled reports whether the menu is disabled.
	Disabled bool
}

// Clone deep-copies the menu, cloning its actions.
func (m OwnedMenu) Clone() OwnedMenu {
	items := make([]OwnedMenuItem, len(m.Items))
	for i, item := range m.Items {
		items[i] = item.Clone()
	}
	return OwnedMenu{Name: m.Name, Items: items, Disabled: m.Disabled}
}

// OwnedMenuItem is the owned form of a MenuItem: OwnedMenuItemSeparator, OwnedMenuItemSubmenu,
// OwnedMenuItemSystemMenu or OwnedMenuItemAction.
type OwnedMenuItem interface {
	// Clone deep-copies the item, cloning its action with BoxedClone.
	Clone() OwnedMenuItem
	isOwnedMenuItem()
}

// OwnedMenuItemSeparator is a separator between items.
type OwnedMenuItemSeparator struct{}

// OwnedMenuItemSubmenu is a submenu.
type OwnedMenuItemSubmenu struct {
	OwnedMenu
}

// OwnedMenuItemSystemMenu is a menu managed by the system.
type OwnedMenuItemSystemMenu struct {
	OwnedOsMenu
}

// OwnedMenuItemAction is an item that performs an action.
type OwnedMenuItemAction struct {
	// Name is the name of the item.
	Name string
	// Action is the action performed when the item is selected.
	Action Action
	// OsAction is the OS action that corresponds to this action, if any.
	OsAction Option[OsAction]
	// Checked reports whether the item is checked.
	Checked bool
	// Disabled reports whether the item is disabled.
	Disabled bool
}

func (OwnedMenuItemSeparator) isOwnedMenuItem()  {}
func (OwnedMenuItemSubmenu) isOwnedMenuItem()    {}
func (OwnedMenuItemSystemMenu) isOwnedMenuItem() {}
func (OwnedMenuItemAction) isOwnedMenuItem()     {}

// Clone returns the separator.
func (m OwnedMenuItemSeparator) Clone() OwnedMenuItem { return m }

// Clone deep-copies the submenu.
func (m OwnedMenuItemSubmenu) Clone() OwnedMenuItem {
	return OwnedMenuItemSubmenu{m.OwnedMenu.Clone()}
}

// Clone copies the system menu.
func (m OwnedMenuItemSystemMenu) Clone() OwnedMenuItem { return m }

// Clone copies the item, cloning its action with BoxedClone.
func (m OwnedMenuItemAction) Clone() OwnedMenuItem {
	return OwnedMenuItemAction{
		Name:     m.Name,
		Action:   m.Action.BoxedClone(),
		OsAction: m.OsAction,
		Checked:  m.Checked,
		Disabled: m.Disabled,
	}
}

// OsAction is an action recognized by the operating system, which may provide specialized
// behavior for it.
type OsAction uint8

const (
	// OsActionCut is the cut action.
	OsActionCut OsAction = iota
	// OsActionCopy is the copy action.
	OsActionCopy
	// OsActionPaste is the paste action.
	OsActionPaste
	// OsActionSelectAll is the select-all action.
	OsActionSelectAll
	// OsActionUndo is the undo action.
	OsActionUndo
	// OsActionRedo is the redo action.
	OsActionRedo
)

func initAppMenus(platform Platform, cx *App) {
	{
		asyncCx := cx.ToAsync()
		platform.OnWillOpenAppMenu(func() {
			if app, ok := asyncCx.upgradeApp(); ok {
				borrow := app.BorrowMut()
				defer borrow.Release()
				borrow.update(func(cx *App) { cx.clearPendingKeystrokes() })
			}
		})
	}
	{
		asyncCx := cx.ToAsync()
		platform.OnValidateAppMenuCommand(func(action Action) bool {
			app, ok := asyncCx.upgradeApp()
			if !ok {
				return false
			}
			borrow := app.BorrowMut()
			defer borrow.Release()
			var available bool
			borrow.update(func(cx *App) { available = cx.IsActionAvailable(action) })
			return available
		})
	}
	{
		asyncCx := cx.ToAsync()
		platform.OnAppMenuAction(func(action Action) {
			if app, ok := asyncCx.upgradeApp(); ok {
				borrow := app.BorrowMut()
				defer borrow.Release()
				borrow.update(func(cx *App) { cx.DispatchAction(action) })
			}
		})
	}
}
