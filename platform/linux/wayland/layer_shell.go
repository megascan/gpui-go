//go:build linux

package wayland

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

func waylandLayer(layer gpui.Layer) wlproto.ZwlrLayerShellV1Layer {
	switch layer {
	case gpui.LayerBackground:
		return wlproto.ZwlrLayerShellV1LayerBackground
	case gpui.LayerBottom:
		return wlproto.ZwlrLayerShellV1LayerBottom
	case gpui.LayerTop:
		return wlproto.ZwlrLayerShellV1LayerTop
	default:
		return wlproto.ZwlrLayerShellV1LayerOverlay
	}
}

const layerShellAnchorAll = wlproto.ZwlrLayerSurfaceV1AnchorTop | wlproto.ZwlrLayerSurfaceV1AnchorBottom | wlproto.ZwlrLayerSurfaceV1AnchorLeft | wlproto.ZwlrLayerSurfaceV1AnchorRight

func layerShellWaylandAnchor(anchor gpui.LayerShellAnchor) wlproto.ZwlrLayerSurfaceV1Anchor {
	return wlproto.ZwlrLayerSurfaceV1Anchor(anchor) & layerShellAnchorAll
}

func waylandKeyboardInteractivity(value gpui.KeyboardInteractivity) wlproto.ZwlrLayerSurfaceV1KeyboardInteractivity {
	switch value {
	case gpui.KeyboardInteractivityNone:
		return wlproto.ZwlrLayerSurfaceV1KeyboardInteractivityNone
	case gpui.KeyboardInteractivityExclusive:
		return wlproto.ZwlrLayerSurfaceV1KeyboardInteractivityExclusive
	default:
		return wlproto.ZwlrLayerSurfaceV1KeyboardInteractivityOnDemand
	}
}
