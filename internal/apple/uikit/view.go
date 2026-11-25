//go:build darwin

package uikit

import "github.com/megascan/gpui-go/internal/apple/objc"

// UIWindow wraps a UIWindow object.
type UIWindow objc.ID

// ID returns the underlying object.
func (w UIWindow) ID() objc.ID { return objc.ID(w) }

// IsKeyWindow reports whether the window is the key window.
func (w UIWindow) IsKeyWindow() bool { return send[bool](w.ID(), "isKeyWindow") }

// RootViewController returns the window's root view controller, or nil.
func (w UIWindow) RootViewController() UIViewController {
	return UIViewController(send[objc.ID](w.ID(), "rootViewController"))
}

// UIViewController wraps a UIViewController object.
type UIViewController objc.ID

// ID returns the underlying object.
func (c UIViewController) ID() objc.ID { return objc.ID(c) }

// PresentedViewController returns the controller this one presents, or nil.
func (c UIViewController) PresentedViewController() UIViewController {
	return UIViewController(send[objc.ID](c.ID(), "presentedViewController"))
}

// PresentViewControllerAnimatedCompletion presents controller modally.
func (c UIViewController) PresentViewControllerAnimatedCompletion(controller objc.ID, animated bool, completion objc.ID) {
	sendVoid(c.ID(), "presentViewController:animated:completion:", controller, animated, completion)
}

// DismissViewControllerAnimatedCompletion dismisses the controller.
func (c UIViewController) DismissViewControllerAnimatedCompletion(animated bool, completion objc.ID) {
	sendVoid(c.ID(), "dismissViewControllerAnimated:completion:", animated, completion)
}

// UserInterfaceStyle returns the controller's trait collection interface style.
func (c UIViewController) UserInterfaceStyle() UIUserInterfaceStyle {
	traits := send[objc.ID](c.ID(), "traitCollection")
	return send[UIUserInterfaceStyle](traits, "userInterfaceStyle")
}

// UIView wraps a UIView object.
type UIView objc.ID

// ID returns the underlying object.
func (v UIView) ID() objc.ID { return objc.ID(v) }

// Bounds returns the view's bounds in points.
func (v UIView) Bounds() objc.CGRect { return send[objc.CGRect](v.ID(), "bounds") }

// UIScreen wraps a UIScreen object.
type UIScreen objc.ID

// ID returns the underlying object.
func (s UIScreen) ID() objc.ID { return objc.ID(s) }

// UIScreenMainScreen returns the device's main screen.
func UIScreenMainScreen() UIScreen { return UIScreen(sendClass[objc.ID]("UIScreen", "mainScreen")) }

// UIScreenScreens returns every connected screen.
func UIScreenScreens() []UIScreen {
	ids := objc.NSArrayToIDs(sendClass[objc.ID]("UIScreen", "screens"))
	out := make([]UIScreen, len(ids))
	for i, id := range ids {
		out[i] = UIScreen(id)
	}
	return out
}

// Bounds returns the screen bounds in points.
func (s UIScreen) Bounds() objc.CGRect { return send[objc.CGRect](s.ID(), "bounds") }

// NativeScale returns the physical scale factor.
func (s UIScreen) NativeScale() objc.CGFloat { return send[objc.CGFloat](s.ID(), "nativeScale") }

// Scale returns the logical scale factor.
func (s UIScreen) Scale() objc.CGFloat { return send[objc.CGFloat](s.ID(), "scale") }

// MaximumFramesPerSecond returns the screen's maximum refresh rate.
func (s UIScreen) MaximumFramesPerSecond() objc.NSInteger {
	return send[objc.NSInteger](s.ID(), "maximumFramesPerSecond")
}

// UIPasteboard wraps a UIPasteboard object.
type UIPasteboard objc.ID

// ID returns the underlying object.
func (p UIPasteboard) ID() objc.ID { return objc.ID(p) }

// UIPasteboardGeneralPasteboard returns the system pasteboard.
func UIPasteboardGeneralPasteboard() UIPasteboard {
	return UIPasteboard(sendClass[objc.ID]("UIPasteboard", "generalPasteboard"))
}

// String returns the pasteboard's string, if any.
func (p UIPasteboard) String() (string, bool) {
	s := send[objc.ID](p.ID(), "string")
	if s == objc.Nil {
		return "", false
	}
	return objc.GoString(s), true
}

// SetString replaces the pasteboard contents with s.
func (p UIPasteboard) SetString(s string) {
	str := objc.NSStringFromGo(s)
	defer objc.Release(str)
	sendVoid(p.ID(), "setString:", str)
}
