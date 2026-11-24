//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSScreen wraps an NSScreen object.
type NSScreen objc.ID

// ID returns the underlying object.
func (s NSScreen) ID() objc.ID { return objc.ID(s) }

// NSScreenScreens returns every screen; the first holds the menu bar.
func NSScreenScreens() []NSScreen {
	ids := objc.NSArrayToIDs(sendClass[objc.ID]("NSScreen", "screens"))
	out := make([]NSScreen, len(ids))
	for i, id := range ids {
		out[i] = NSScreen(id)
	}
	return out
}

// NSScreenMainScreen returns the screen with the key window.
func NSScreenMainScreen() NSScreen { return NSScreen(sendClass[objc.ID]("NSScreen", "mainScreen")) }

// Frame returns the screen frame.
func (s NSScreen) Frame() objc.CGRect { return send[objc.CGRect](s.ID(), "frame") }

// VisibleFrame returns the frame excluding the menu bar and dock.
func (s NSScreen) VisibleFrame() objc.CGRect { return send[objc.CGRect](s.ID(), "visibleFrame") }

// BackingScaleFactor returns the backing scale factor.
func (s NSScreen) BackingScaleFactor() objc.CGFloat {
	return send[objc.CGFloat](s.ID(), "backingScaleFactor")
}

// DeviceDescription returns the screen's device description dictionary.
func (s NSScreen) DeviceDescription() objc.ID { return send[objc.ID](s.ID(), "deviceDescription") }

// LocalizedName returns the screen's localised name.
func (s NSScreen) LocalizedName() string {
	return objc.GoString(send[objc.ID](s.ID(), "localizedName"))
}

// MaximumFramesPerSecond returns the screen's maximum refresh rate.
func (s NSScreen) MaximumFramesPerSecond() int {
	return send[int](s.ID(), "maximumFramesPerSecond")
}
