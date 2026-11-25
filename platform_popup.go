package gpui

// PopupOptions are the options of a parent-anchored popup window such as a menu, dropdown,
// context menu or tooltip. The popup is placed relative to an anchor rectangle on its parent
// window and the platform resolves the final position, so it works both where the compositor
// owns placement (Wayland) and with absolute coordinates. Its size comes from
// WindowOptions.WindowBounds, whose origin is ignored; coordinates are logical pixels.
type PopupOptions struct {
	// Parent is the window the popup is anchored to.
	Parent AnyWindowHandle
	// AnchorRect is the rectangle the popup is positioned relative to, in the parent window's
	// coordinate space; a dropdown uses the bounds of the button that opened it.
	AnchorRect Bounds[Pixels]
	// Anchor is the point of AnchorRect the popup is anchored to.
	Anchor PopupAnchor
	// Gravity is the direction the popup extends away from the anchor point; a dropdown below
	// its button anchors to PopupAnchorBottomLeft with PopupGravityBottomRight.
	Gravity PopupGravity
	// ConstraintAdjustment is how the platform may adjust a popup that would be off-screen.
	ConstraintAdjustment PopupConstraintAdjustment
	// Offset is an additional offset applied after anchoring.
	Offset Point[Pixels]
	// Grab requests an explicit input grab, as menus and comboboxes take: the popup takes
	// keyboard focus and is dismissed by clicks outside it or a dismissing key. The grab must be
	// requested while the triggering input is active, so open grabbing popups from a mouse-down
	// handler. Dismissal only covers input aimed at other applications, and nested grabbing
	// popups must be closed in reverse order.
	Grab bool
}

// PopupAnchor is the point of the anchor rectangle a popup is anchored to.
type PopupAnchor uint8

const (
	// PopupAnchorCenter anchors to the center of the anchor rectangle.
	PopupAnchorCenter PopupAnchor = iota
	// PopupAnchorTop anchors to the center of the top edge.
	PopupAnchorTop
	// PopupAnchorBottom anchors to the center of the bottom edge.
	PopupAnchorBottom
	// PopupAnchorLeft anchors to the center of the left edge.
	PopupAnchorLeft
	// PopupAnchorRight anchors to the center of the right edge.
	PopupAnchorRight
	// PopupAnchorTopLeft anchors to the top-left corner.
	PopupAnchorTopLeft
	// PopupAnchorBottomLeft anchors to the bottom-left corner.
	PopupAnchorBottomLeft
	// PopupAnchorTopRight anchors to the top-right corner.
	PopupAnchorTopRight
	// PopupAnchorBottomRight anchors to the bottom-right corner.
	PopupAnchorBottomRight
)

// PopupGravity is the direction in which a popup extends away from its anchor point.
type PopupGravity uint8

const (
	// PopupGravityCenter centers the popup over the anchor point.
	PopupGravityCenter PopupGravity = iota
	// PopupGravityTop extends the popup upwards.
	PopupGravityTop
	// PopupGravityBottom extends the popup downwards.
	PopupGravityBottom
	// PopupGravityLeft extends the popup to the left.
	PopupGravityLeft
	// PopupGravityRight extends the popup to the right.
	PopupGravityRight
	// PopupGravityTopLeft extends the popup up and to the left.
	PopupGravityTopLeft
	// PopupGravityBottomLeft extends the popup down and to the left.
	PopupGravityBottomLeft
	// PopupGravityTopRight extends the popup up and to the right.
	PopupGravityTopRight
	// PopupGravityBottomRight extends the popup down and to the right.
	PopupGravityBottomRight
)

// PopupConstraintAdjustment is how a popup may be adjusted by the platform when the requested
// placement would put it off-screen. With no flags set the popup is placed exactly as requested
// and may be clipped.
type PopupConstraintAdjustment uint32

const (
	// PopupConstraintAdjustmentSlideX lets the popup slide horizontally to stay on-screen.
	PopupConstraintAdjustmentSlideX PopupConstraintAdjustment = 1
	// PopupConstraintAdjustmentSlideY lets the popup slide vertically to stay on-screen.
	PopupConstraintAdjustmentSlideY PopupConstraintAdjustment = 2
	// PopupConstraintAdjustmentFlipX lets anchor and gravity flip horizontally to stay on-screen.
	PopupConstraintAdjustmentFlipX PopupConstraintAdjustment = 4
	// PopupConstraintAdjustmentFlipY lets anchor and gravity flip vertically to stay on-screen.
	PopupConstraintAdjustmentFlipY PopupConstraintAdjustment = 8
	// PopupConstraintAdjustmentResizeX lets the popup shrink horizontally to stay on-screen.
	PopupConstraintAdjustmentResizeX PopupConstraintAdjustment = 16
	// PopupConstraintAdjustmentResizeY lets the popup shrink vertically to stay on-screen.
	PopupConstraintAdjustmentResizeY PopupConstraintAdjustment = 32
)

// Contains reports whether every flag of other is set.
func (a PopupConstraintAdjustment) Contains(other PopupConstraintAdjustment) bool {
	return a&other == other
}

// PopupNotSupportedError is returned when the current platform has no native popup
// implementation. Native popups differ from in-window popovers drawn as elements, so a caller
// that wants a popup everywhere treats this error as a cue to fall back to in-window rendering.
type PopupNotSupportedError struct{}

// Error returns upstream's message.
func (PopupNotSupportedError) Error() string {
	return "popups are not supported on this platform"
}
