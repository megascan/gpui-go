//go:build windows

package windows

import (
	"weak"

	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/win/com"
)

// Adapter is the UI Automation adapter for one window; M4-windows-a11y ports it.
type Adapter struct{}

// QueuedEvents are UI Automation events to raise after the adapter's lock is released.
type QueuedEvents struct {
	events []queuedEvent
}

// SubclassingAdapter installs an Adapter by subclassing a window; M4-windows-a11y ports it.
type SubclassingAdapter struct{}

type platformRange struct {
	context weak.Pointer[Context]
	state   consumer.WeakRange
}

func newPlatformRange(context weak.Pointer[Context], r consumer.Range) *com.Object {
	panic("a11y/windows: unimplemented newPlatformRange")
}
