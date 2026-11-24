//go:build linux

package wayland

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

func popupWaylandAnchor(anchor gpui.PopupAnchor) wlproto.XdgPositionerAnchor {
	switch anchor {
	case gpui.PopupAnchorTop:
		return wlproto.XdgPositionerAnchorTop
	case gpui.PopupAnchorBottom:
		return wlproto.XdgPositionerAnchorBottom
	case gpui.PopupAnchorLeft:
		return wlproto.XdgPositionerAnchorLeft
	case gpui.PopupAnchorRight:
		return wlproto.XdgPositionerAnchorRight
	case gpui.PopupAnchorTopLeft:
		return wlproto.XdgPositionerAnchorTopLeft
	case gpui.PopupAnchorBottomLeft:
		return wlproto.XdgPositionerAnchorBottomLeft
	case gpui.PopupAnchorTopRight:
		return wlproto.XdgPositionerAnchorTopRight
	case gpui.PopupAnchorBottomRight:
		return wlproto.XdgPositionerAnchorBottomRight
	default:
		return wlproto.XdgPositionerAnchorNone
	}
}

func waylandGravity(gravity gpui.PopupGravity) wlproto.XdgPositionerGravity {
	switch gravity {
	case gpui.PopupGravityTop:
		return wlproto.XdgPositionerGravityTop
	case gpui.PopupGravityBottom:
		return wlproto.XdgPositionerGravityBottom
	case gpui.PopupGravityLeft:
		return wlproto.XdgPositionerGravityLeft
	case gpui.PopupGravityRight:
		return wlproto.XdgPositionerGravityRight
	case gpui.PopupGravityTopLeft:
		return wlproto.XdgPositionerGravityTopLeft
	case gpui.PopupGravityBottomLeft:
		return wlproto.XdgPositionerGravityBottomLeft
	case gpui.PopupGravityTopRight:
		return wlproto.XdgPositionerGravityTopRight
	case gpui.PopupGravityBottomRight:
		return wlproto.XdgPositionerGravityBottomRight
	default:
		return wlproto.XdgPositionerGravityNone
	}
}

const constraintAdjustmentAll = wlproto.XdgPositionerConstraintAdjustmentSlideX | wlproto.XdgPositionerConstraintAdjustmentSlideY | wlproto.XdgPositionerConstraintAdjustmentFlipX | wlproto.XdgPositionerConstraintAdjustmentFlipY | wlproto.XdgPositionerConstraintAdjustmentResizeX | wlproto.XdgPositionerConstraintAdjustmentResizeY

func waylandConstraintAdjustment(adjustment gpui.PopupConstraintAdjustment) wlproto.XdgPositionerConstraintAdjustment {
	return wlproto.XdgPositionerConstraintAdjustment(adjustment) & constraintAdjustmentAll
}
