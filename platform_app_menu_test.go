package gpui

import "testing"

func TestAppMenu_Menu(t *testing.T) {
	menu := NewMenu("App").
		WithItems(
			NewMenuItemAction("Action 1", NoAction{}),
			NewMenuItemSeparator(),
		).
		WithDisabled(true)

	if menu.Name != "App" {
		t.Fatalf("name = %q", menu.Name)
	}
	if len(menu.Items) != 2 {
		t.Fatalf("items = %d", len(menu.Items))
	}
	if !menu.Disabled {
		t.Fatal("menu not disabled")
	}
}

func TestAppMenu_MenuItemBuilder(t *testing.T) {
	item := NewMenuItemAction("Test Action", NoAction{})
	action, ok := item.(MenuItemAction)
	if !ok {
		t.Fatalf("item = %T", item)
	}
	if action.Name != "Test Action" {
		t.Fatalf("name = %q", action.Name)
	}
	if action.Checked || action.Disabled {
		t.Fatalf("item = %+v", action)
	}

	if !NewMenuItemAction("Test Action", NoAction{}).WithChecked(true).IsChecked() {
		t.Fatal("item not checked")
	}
	if !NewMenuItemAction("Test Action", NoAction{}).WithDisabled(true).IsDisabled() {
		t.Fatal("item not disabled")
	}

	submenu := NewMenuItemSubmenu(Menu{
		Name:     "Submenu",
		Items:    nil,
		Disabled: true,
	})
	menu, ok := submenu.(MenuItemSubmenu)
	if !ok {
		t.Fatalf("submenu = %T", submenu)
	}
	if menu.Name != "Submenu" {
		t.Fatalf("name = %q", menu.Name)
	}
	if submenu.IsChecked() {
		t.Fatal("submenu checked")
	}
	if !submenu.IsDisabled() {
		t.Fatal("submenu not disabled")
	}
}
