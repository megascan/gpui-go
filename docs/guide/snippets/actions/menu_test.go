package actions

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/megascan/gpui-go"
)

func TestMenu_KeyboardNavigation(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		cx.Update(BindMenuKeys)

		menu, vcx := cx.AddWindowView(func(_ *gpui.Window, cx *gpui.Context[Menu]) *Menu {
			return NewMenu(cx)
		})
		defer menu.Release()
		vcx.Focus(menu)

		vcx.SimulateKeystrokes("down down up enter")

		got := menu.ReadWith(vcx, func(m *Menu, _ *gpui.App) int { return m.confirmed })
		if got != 1 {
			t.Fatalf("confirmed = %d, want 1", got)
		}
	})
}

func TestMenu_BuildActionFromJSON(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		menu, vcx := cx.AddWindowView(func(_ *gpui.Window, cx *gpui.Context[Menu]) *Menu {
			return NewMenu(cx)
		})
		defer menu.Release()
		vcx.Focus(menu)

		var action gpui.Action
		var err error
		cx.Update(func(cx *gpui.App) {
			action, err = cx.BuildAction("menu::Jump", gpui.Some(jsontext.Value(`{"index": 2}`)))
		})
		if err != nil {
			t.Fatal(err)
		}
		vcx.DispatchAction(action)

		got := menu.ReadWith(vcx, func(m *Menu, _ *gpui.App) int { return m.selected })
		if got != 2 {
			t.Fatalf("selected = %d, want 2", got)
		}
	})
}
