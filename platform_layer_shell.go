package gpui

// Layer is the layer a layer_shell surface is rendered on.
type Layer uint8

const (
	// LayerBackground is the background layer, typically used for wallpapers.
	LayerBackground Layer = iota
	// LayerBottom is the bottom layer.
	LayerBottom
	// LayerTop is the top layer, typically used for fullscreen windows.
	LayerTop
	// LayerOverlay is the overlay layer, used for surfaces that should always be on top.
	LayerOverlay
)

// DefaultLayer returns upstream's default layer, LayerOverlay.
func DefaultLayer() Layer {
	return LayerOverlay
}

// LayerShellAnchor is the set of screen edges a layer_shell surface is anchored to (layer_shell::Anchor, D-SUBMOD).
type LayerShellAnchor uint32

const (
	// LayerShellAnchorTop anchors to the top edge of the screen.
	LayerShellAnchorTop LayerShellAnchor = 1
	// LayerShellAnchorBottom anchors to the bottom edge of the screen.
	LayerShellAnchorBottom LayerShellAnchor = 2
	// LayerShellAnchorLeft anchors to the left edge of the screen.
	LayerShellAnchorLeft LayerShellAnchor = 4
	// LayerShellAnchorRight anchors to the right edge of the screen.
	LayerShellAnchorRight LayerShellAnchor = 8
)

// Contains reports whether every flag of other is set in a.
func (a LayerShellAnchor) Contains(other LayerShellAnchor) bool {
	return a&other == other
}

// KeyboardInteractivity is the keyboard interactivity mode of a layer_shell surface.
type KeyboardInteractivity uint8

const (
	// KeyboardInteractivityNone delivers no keyboard input to the surface.
	KeyboardInteractivityNone KeyboardInteractivity = iota
	// KeyboardInteractivityExclusive gives the surface exclusive keyboard focus while it is above the shell layer.
	KeyboardInteractivityExclusive
	// KeyboardInteractivityOnDemand lets the surface be focused like a normal window.
	KeyboardInteractivityOnDemand
)

// DefaultKeyboardInteractivity returns upstream's default, KeyboardInteractivityOnDemand.
func DefaultKeyboardInteractivity() KeyboardInteractivity {
	return KeyboardInteractivityOnDemand
}

// LayerShellOptions are the options for creating a layer_shell window.
type LayerShellOptions struct {
	// Namespace is the surface namespace, used by compositors to apply rules.
	Namespace string
	// Layer is the layer the surface is rendered on.
	Layer Layer
	// Anchor is the anchor point of the surface.
	Anchor LayerShellAnchor
	// ExclusiveZone requests that the compositor avoids occluding an area with other surfaces.
	ExclusiveZone Option[Pixels]
	// ExclusiveEdge is the anchor point of the exclusive zone; derived from Anchor when None.
	ExclusiveEdge Option[LayerShellAnchor]
	// Margin is the margins between the surface and its anchor points, in CSS order: top, right, bottom, left.
	Margin Option[[4]Pixels]
	// KeyboardInteractivity is how keyboard events are delivered to the surface.
	KeyboardInteractivity KeyboardInteractivity
}

// DefaultLayerShellOptions returns upstream's default options.
func DefaultLayerShellOptions() LayerShellOptions {
	return LayerShellOptions{
		Layer:                 DefaultLayer(),
		KeyboardInteractivity: DefaultKeyboardInteractivity(),
	}
}

// LayerShellNotSupportedError reports that the compositor doesn't support the layer_shell protocol.
type LayerShellNotSupportedError struct{}

// Error returns upstream's error message.
func (LayerShellNotSupportedError) Error() string {
	return "Compositor doesn't support zwlr_layer_shell_v1"
}
