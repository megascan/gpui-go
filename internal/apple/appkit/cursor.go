//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSCursor wraps an NSCursor object.
type NSCursor objc.ID

// ID returns the underlying object.
func (c NSCursor) ID() objc.ID { return objc.ID(c) }

// NSCursorNamed returns the class cursor returned by selector name, such as arrowCursor.
func NSCursorNamed(name string) NSCursor {
	return NSCursor(sendClass[objc.ID]("NSCursor", name))
}

// NSCursorResponds reports whether NSCursor implements the class selector name.
func NSCursorResponds(name string) bool {
	return objc.RespondsToSelector(objc.ID(class("NSCursor")), objc.Sel(name))
}

// Set makes the cursor current.
func (c NSCursor) Set() { sendVoid(c.ID(), "set") }

// NSCursorHide hides the cursor.
func NSCursorHide() { sendVoid(objc.ID(class("NSCursor")), "hide") }

// NSCursorUnhide shows the cursor.
func NSCursorUnhide() { sendVoid(objc.ID(class("NSCursor")), "unhide") }

// NSCursorSetHiddenUntilMouseMoves hides the cursor until the mouse moves.
func NSCursorSetHiddenUntilMouseMoves(flag bool) {
	sendVoid(objc.ID(class("NSCursor")), "setHiddenUntilMouseMoves:", flag)
}
