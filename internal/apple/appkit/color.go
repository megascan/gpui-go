//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSColor wraps an NSColor object.
type NSColor objc.ID

// ID returns the underlying object.
func (c NSColor) ID() objc.ID { return objc.ID(c) }

// NSColorClear returns the clear colour.
func NSColorClear() NSColor { return NSColor(sendClass[objc.ID]("NSColor", "clearColor")) }

// NSColorWithSRGB returns an sRGB colour.
func NSColorWithSRGB(r, g, b, a objc.CGFloat) NSColor {
	return NSColor(sendClass[objc.ID]("NSColor", "colorWithSRGBRed:green:blue:alpha:", r, g, b, a))
}

// CGColor returns the colour as a CGColorRef.
func (c NSColor) CGColor() uintptr { return send[uintptr](c.ID(), "CGColor") }
