//go:build linux

// Package wayland is the Wayland back end of the Linux platform.
package wayland

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

func toShape(style gpui.CursorStyle) wlproto.WpCursorShapeDeviceV1Shape {
	switch style {
	case gpui.CursorStyleArrow:
		return wlproto.WpCursorShapeDeviceV1ShapeDefault
	case gpui.CursorStyleIBeam:
		return wlproto.WpCursorShapeDeviceV1ShapeText
	case gpui.CursorStyleCrosshair:
		return wlproto.WpCursorShapeDeviceV1ShapeCrosshair
	case gpui.CursorStyleClosedHand:
		return wlproto.WpCursorShapeDeviceV1ShapeGrabbing
	case gpui.CursorStyleOpenHand:
		return wlproto.WpCursorShapeDeviceV1ShapeGrab
	case gpui.CursorStylePointingHand:
		return wlproto.WpCursorShapeDeviceV1ShapePointer
	case gpui.CursorStyleResizeLeft:
		return wlproto.WpCursorShapeDeviceV1ShapeWResize
	case gpui.CursorStyleResizeRight:
		return wlproto.WpCursorShapeDeviceV1ShapeEResize
	case gpui.CursorStyleResizeLeftRight:
		return wlproto.WpCursorShapeDeviceV1ShapeEwResize
	case gpui.CursorStyleResizeUp:
		return wlproto.WpCursorShapeDeviceV1ShapeNResize
	case gpui.CursorStyleResizeDown:
		return wlproto.WpCursorShapeDeviceV1ShapeSResize
	case gpui.CursorStyleResizeUpDown:
		return wlproto.WpCursorShapeDeviceV1ShapeNsResize
	case gpui.CursorStyleResizeUpLeftDownRight:
		return wlproto.WpCursorShapeDeviceV1ShapeNwseResize
	case gpui.CursorStyleResizeUpRightDownLeft:
		return wlproto.WpCursorShapeDeviceV1ShapeNeswResize
	case gpui.CursorStyleResizeColumn:
		return wlproto.WpCursorShapeDeviceV1ShapeColResize
	case gpui.CursorStyleResizeRow:
		return wlproto.WpCursorShapeDeviceV1ShapeRowResize
	case gpui.CursorStyleIBeamCursorForVerticalLayout:
		return wlproto.WpCursorShapeDeviceV1ShapeVerticalText
	case gpui.CursorStyleOperationNotAllowed:
		return wlproto.WpCursorShapeDeviceV1ShapeNotAllowed
	case gpui.CursorStyleDragLink:
		return wlproto.WpCursorShapeDeviceV1ShapeAlias
	case gpui.CursorStyleDragCopy:
		return wlproto.WpCursorShapeDeviceV1ShapeCopy
	case gpui.CursorStyleContextualMenu:
		return wlproto.WpCursorShapeDeviceV1ShapeContextMenu
	}
	panic("gpui: invalid CursorStyle")
}
