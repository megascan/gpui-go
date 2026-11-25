// Package testingguide holds the snippets of docs/guide/testing.md.
package testingguide

import (
	"fmt"

	"github.com/megascan/gpui-go"
)

// Changed is emitted when the count changes.
type Changed struct {
	Count int
}

// Counter is a view under test.
type Counter struct {
	count int
}

// Add changes the count, notifies observers and emits Changed.
func (c *Counter) Add(delta int, cx *gpui.Context[Counter]) {
	c.count += delta
	cx.Notify()
	cx.Emit(Changed{Count: c.count})
}

// Render renders the view.
func (c *Counter) Render(_ *gpui.Window, cx *gpui.Context[Counter]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		Gap2().
		Child(gpui.NewDiv().ChildText(fmt.Sprintf("%d", c.count))).
		Child(
			gpui.NewDiv().
				ID("increment").
				DebugSelector(func() string { return "increment" }).
				Size8().
				OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Counter]) {
					this.Add(1, cx)
				})).
				ChildText("+"),
		)
}
