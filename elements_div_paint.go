package gpui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

// ElementID returns the div's element id, if it has one.
func (d *Div) ElementID() (ElementID, bool) {
	return d.interactivity.ElementID.Get()
}

// SourceLocation returns where the div was constructed, or nil.
func (d *Div) SourceLocation() *SourceLocation {
	return d.interactivity.SourceLocation()
}

// A11yRole returns the div's accessible role; GenericContainer is never reported.
func (d *Div) A11yRole() (a11y.Role, bool) {
	role, ok := d.interactivity.overrideRole.Get()
	if !ok || role == a11y.RoleGenericContainer {
		return 0, false
	}
	return role, true
}

// WriteA11yInfo writes the div's accessibility properties to node.
func (d *Div) WriteA11yInfo(node *a11y.Node) {
	d.interactivity.writeA11yInfo(node)
}

// ElementA11ySyntheticChildren adds the div's synthetic accessibility children with builder.
func (d *Div) ElementA11ySyntheticChildren(prepaint *Option[Hitbox], builder *A11ySubtreeBuilder) {
	if f := d.interactivity.a11ySyntheticChildren; f != nil {
		d.interactivity.a11ySyntheticChildren = nil
		f(builder)
	}
}

func (d *Div) provideImageCache(window *Window, cx *App) Option[AnyImageCache] {
	if d.imageCache == nil {
		return None[AnyImageCache]()
	}
	return Some(d.imageCache.Provide(window, cx))
}

// RequestLayout requests the div's layout and the layouts of its children.
func (d *Div) RequestLayout(globalID *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, DivFrameState) {
	var childLayoutIDs []LayoutID
	imageCache := d.provideImageCache(window, cx)

	layoutID := window.WithImageCache(imageCache, func(window *Window) LayoutID {
		return d.interactivity.RequestLayout(globalID, inspectorID, window, cx, func(style Style, window *Window, cx *App) LayoutID {
			return window.WithTextStyle(optionalTextStyle(&style), func(window *Window) LayoutID {
				childLayoutIDs = make([]LayoutID, 0, len(d.children))
				for i := range d.children {
					childLayoutIDs = append(childLayoutIDs, d.children[i].RequestLayout(window, cx))
				}
				return window.RequestLayout(style, childLayoutIDs, cx)
			})
		})
	})

	return layoutID, DivFrameState{childLayoutIDs: childLayoutIDs}
}

// Prepaint commits the bounds of the div and prepaints its children.
func (d *Div) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *DivFrameState, window *Window, cx *App) Option[Hitbox] {
	imageCache := d.provideImageCache(window, cx)

	hasPrepaintListener := d.prepaintListener != nil
	capacity := 0
	if hasPrepaintListener {
		capacity = len(requestLayout.childLayoutIDs)
	}
	childrenBounds := make([]Bounds[Pixels], 0, capacity)

	childMin := NewPoint(PixelsMax, PixelsMax)
	childMax := Point[Pixels]{}
	if anchor, ok := d.interactivity.scrollAnchor.Get(); ok {
		*anchor.lastOrigin = bounds.Origin.Sub(window.ElementOffset())
	}
	var contentSize Size[Pixels]
	if len(requestLayout.childLayoutIDs) == 0 {
		contentSize = bounds.Size
	} else if scrollHandle, ok := d.interactivity.trackedScrollHandle.Get(); ok {
		state := scrollHandle.state
		state.childBounds = make([]Bounds[Pixels], 0, len(requestLayout.childLayoutIDs))
		for _, childLayoutID := range requestLayout.childLayoutIDs {
			childBounds := window.LayoutBounds(childLayoutID)
			childMin = childMin.Min(childBounds.Origin)
			childMax = childMax.Max(childBounds.BottomRight())
			state.childBounds = append(state.childBounds, childBounds)
		}
		contentSize = childMax.Sub(childMin).ToSize()
	} else {
		for _, childLayoutID := range requestLayout.childLayoutIDs {
			childBounds := window.LayoutBounds(childLayoutID)
			childMin = childMin.Min(childBounds.Origin)
			childMax = childMax.Max(childBounds.BottomRight())

			if hasPrepaintListener {
				childrenBounds = append(childrenBounds, childBounds)
			}
		}
		contentSize = childMax.Sub(childMin).ToSize()
	}

	return d.interactivity.Prepaint(globalID, inspectorID, bounds, contentSize, window, cx, func(style *Style, scrollOffset Point[Pixels], hitbox Option[Hitbox], window *Window, cx *App) Option[Hitbox] {
		if style.Display == DisplayNone {
			return hitbox
		}

		window.WithImageCache(imageCache, func(window *Window) struct{} {
			window.WithElementOffset(scrollOffset, func(window *Window) struct{} {
				if d.prepaintOrderFunc != nil {
					order := d.prepaintOrderFunc(window, cx)
					for _, idx := range order {
						if idx >= 0 && idx < len(d.children) {
							d.children[idx].Prepaint(window, cx)
						}
					}
				} else {
					for i := range d.children {
						d.children[i].Prepaint(window, cx)
					}
				}
				return struct{}{}
			})

			if d.prepaintListener != nil {
				d.prepaintListener(childrenBounds, window, cx)
			}
			return struct{}{}
		})

		return hitbox
	})
}

// Paint paints the div and its children.
func (d *Div) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *DivFrameState, hitbox *Option[Hitbox], window *Window, cx *App) {
	imageCache := d.provideImageCache(window, cx)

	var hitboxRef *Hitbox
	if hitbox.IsSome() {
		hitboxRef = hitbox.AsMut()
	}
	window.WithImageCache(imageCache, func(window *Window) struct{} {
		d.interactivity.Paint(globalID, inspectorID, bounds, hitboxRef, window, cx, func(style *Style, window *Window, cx *App) {
			if style.Display == DisplayNone {
				return
			}

			for i := range d.children {
				d.children[i].Paint(window, cx)
			}
		})
		return struct{}{}
	})
}

// IntoAnyElement converts the Div into an AnyElement.
func (d *Div) IntoAnyElement() AnyElement {
	return NewAnyElement[DivFrameState, Option[Hitbox]](d)
}

func optionalTextStyle(style *Style) Option[TextStyleRefinement] {
	if text := style.TextStyle(); text != nil {
		return Some(*text)
	}
	return None[TextStyleRefinement]()
}

func optionalOverflowMask(style *Style, bounds Bounds[Pixels], remSize Pixels) Option[ContentMask[Pixels]] {
	if mask, ok := style.OverflowMask(bounds, remSize); ok {
		return Some(mask)
	}
	return None[ContentMask[Pixels]]()
}

func interactiveElementStateOrDefault(state Option[Option[InteractiveElementState]]) Option[InteractiveElementState] {
	inner, ok := state.Get()
	if !ok {
		return None[InteractiveElementState]()
	}
	return Some(inner.UnwrapOrDefault())
}

func interactiveElementStateRef(state *Option[InteractiveElementState]) *InteractiveElementState {
	if state.IsSome() {
		return state.AsMut()
	}
	return nil
}

// RequestLayout lays this element out according to this interactivity state's configured styles.
func (i *Interactivity) RequestLayout(globalID *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App, f func(style Style, window *Window, cx *App) LayoutID) LayoutID {
	if gpuiutil.DebugAssertions {
		window.WithInspectorState(inspectorID, cx, func(inspectorState *Option[DivInspectorState], _ *Window) struct{} {
			if state, ok := inspectorState.Get(); ok {
				i.BaseStyle = *state.BaseStyle
			} else {
				baseStyle := i.BaseStyle
				*inspectorState = Some(DivInspectorState{BaseStyle: &baseStyle})
			}
			return struct{}{}
		})
	}

	return window.WithOptionalElementState(globalID, func(state Option[Option[InteractiveElementState]], window *Window) (LayoutID, Option[InteractiveElementState]) {
		elementState := interactiveElementStateOrDefault(state)

		if es, ok := elementState.Get(); ok && cx.HasActiveDrag() {
			if es.pendingMouseDown != nil {
				*es.pendingMouseDown = None[MouseDownEvent]()
			}
			if es.clickedState != nil {
				*es.clickedState = ElementClickedState{}
			}
		}

		if i.focusable && i.trackedFocusHandle.IsNone() && elementState.IsSome() {
			es := elementState.AsMut()
			if es.focusHandle.IsNone() {
				es.focusHandle = Some(cx.FocusHandle())
			}
			handle := es.focusHandle.Unwrap().Clone().WithTabStop(i.tabStop)

			if index, ok := i.tabIndex.Get(); ok {
				handle = handle.WithTabIndex(index)
			}

			i.trackedFocusHandle = Some(handle)
		}

		if scrollHandle, ok := i.trackedScrollHandle.Get(); ok {
			i.scrollOffset = scrollHandle.state.offset
			i.ongoingScroll = scrollHandle.state.ongoingScroll
		} else if (i.BaseStyle.Overflow.X == Some(OverflowScroll) || i.BaseStyle.Overflow.Y == Some(OverflowScroll)) && elementState.IsSome() {
			es := elementState.AsMut()
			if es.scrollOffset == nil {
				es.scrollOffset = &Point[Pixels]{}
			}
			i.scrollOffset = es.scrollOffset
			if es.ongoingScroll == nil {
				es.ongoingScroll = &OngoingScroll{}
			}
			i.ongoingScroll = es.ongoingScroll
		}

		if handle, ok := i.trackedFocusHandle.Get(); ok {
			window.deferArenaDrop(handle.Release)
		}

		style := i.computeStyleInternal(nil, interactiveElementStateRef(&elementState), window, cx)
		layoutID := f(style, window, cx)
		return layoutID, elementState
	})
}

// Prepaint commits the bounds of this element according to this interactivity state's configured
// styles.
func (i *Interactivity) Prepaint[R any](globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], contentSize Size[Pixels], window *Window, cx *App, f func(style *Style, scrollOffset Point[Pixels], hitbox Option[Hitbox], window *Window, cx *App) R) R {
	i.contentSize = contentSize

	if gpuiutil.DebugAssertions {
		window.WithInspectorState(inspectorID, cx, func(inspectorState *Option[DivInspectorState], _ *Window) struct{} {
			if inspectorState.IsSome() {
				state := inspectorState.AsMut()
				state.Bounds = bounds
				state.ContentSize = contentSize
			}
			return struct{}{}
		})
	}

	if focusHandle, ok := i.trackedFocusHandle.Get(); ok {
		window.SetFocusHandle(focusHandle, cx)

		if window.a11y.isActive() {
			if globalID != nil {
				nodeID := globalID.accesskitNodeID()
				window.a11y.setFocusable(nodeID, focusHandle.id)
				if focusHandle.IsFocused(window) {
					window.a11y.setFocus(nodeID)
				}
			} else if focusHandle.IsFocused(window) {
				window.a11y.noteFocusWithoutNode(focusHandle.id, "it has no element id")
			}
		}
	}

	if i.reportActiveDescendantFocus && window.a11y.isActive() {
		if globalID != nil {
			window.a11y.setActiveDescendant(globalID.accesskitNodeID())
		}
	}
	return window.WithOptionalElementState(globalID, func(state Option[Option[InteractiveElementState]], window *Window) (R, Option[InteractiveElementState]) {
		elementState := interactiveElementStateOrDefault(state)
		style := i.computeStyleInternal(nil, interactiveElementStateRef(&elementState), window, cx)

		if elementState.IsSome() {
			es := elementState.AsMut()
			if es.clickedState != nil {
				i.Active = Some(es.clickedState.Element)
			}
			if i.hoverStyle != nil || i.groupHoverStyle != nil {
				if es.hoverState == nil {
					es.hoverState = &ElementHoverState{}
				}
			}
			if es.activeTooltip != nil {
				if i.tooltipBuilder != nil {
					i.tooltipID = setTooltipOnWindow(es.activeTooltip, window)
				} else {
					if es.longPressTooltipActive != nil {
						*es.longPressTooltipActive = false
					}
					es.activeTooltip = nil
				}
			}
		}

		return window.WithTextStyle(optionalTextStyle(&style), func(window *Window) R {
			return window.WithContentMask(optionalOverflowMask(&style, bounds, window.RemSize()), func(window *Window) R {
				hitbox := None[Hitbox]()
				if i.shouldInsertHitbox(&style, window, cx) {
					hitbox = Some(window.InsertHitbox(bounds, i.hitboxBehavior))
				}

				scrollOffset := i.clampScrollPosition(bounds, &style, window, cx)
				return window.withElementOpacity(style.Opacity, func(window *Window) R {
					return f(&style, scrollOffset, hitbox, window, cx)
				})
			})
		}), elementState
	})
}

func (i *Interactivity) shouldInsertHitbox(style *Style, window *Window, cx *App) bool {
	return i.hitboxBehavior != HitboxBehaviorNormal ||
		i.windowControl.IsSome() ||
		style.MouseCursor.IsSome() ||
		i.group.IsSome() ||
		i.scrollOffset != nil ||
		i.trackedFocusHandle.IsSome() ||
		i.hoverStyle != nil ||
		i.groupHoverStyle != nil ||
		i.hoverListener != nil ||
		len(i.mouseUpListeners) != 0 ||
		len(i.mousePressureListeners) != 0 ||
		len(i.mouseDownListeners) != 0 ||
		len(i.mouseMoveListeners) != 0 ||
		len(i.mouseExitListeners) != 0 ||
		len(i.fileDropExitListeners) != 0 ||
		len(i.clickListeners) != 0 ||
		len(i.auxClickListeners) != 0 ||
		len(i.scrollListeners) != 0 ||
		i.hasPinchListeners() ||
		len(i.backGestureListeners) != 0 ||
		i.dragListener != nil ||
		len(i.dropListeners) != 0 ||
		len(i.dragOverStyles) != 0 ||
		i.tooltipBuilder != nil ||
		window.IsInspectorPicking(cx)
}

func roundToTwoDecimals(pixels Pixels) Pixels {
	const roundingFactor float32 = 100.0
	return Pixels(float32(Pixels(float32(pixels)*roundingFactor).Round()) / roundingFactor)
}

func clampPixels(value, min, max Pixels) Pixels {
	if min > max {
		panic(fmt.Sprintf("min > max, or either was NaN. min = %v, max = %v", min, max))
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (i *Interactivity) clampScrollPosition(bounds Bounds[Pixels], style *Style, window *Window, _ *App) Point[Pixels] {
	if i.scrollOffset == nil {
		return Point[Pixels]{}
	}
	scrollOffset := i.scrollOffset
	scrollToBottom := false
	if scrollHandle, ok := i.trackedScrollHandle.Get(); ok {
		state := scrollHandle.state
		state.overflow = style.Overflow
		state.bounds = bounds
		scrollToBottom = state.scrollToBottom
		state.scrollToBottom = false
		scrollHandle.scrollToActiveItem()
	}

	remSize := window.RemSize()
	padding := EdgesDefiniteLengthToPixels(style.Padding, SizeAbsoluteLengthFromPixels(bounds.Size), remSize).Map(func(edge Pixels) Pixels {
		return window.PixelSnap(edge)
	})
	paddingSize := Size[Pixels]{Width: padding.Left + padding.Right, Height: padding.Top + padding.Bottom}
	paddedContentSize := i.contentSize.Add(paddingSize)
	scrollMax := paddedContentSize.Sub(bounds.Size).ToPoint().Map(roundToTwoDecimals).Max(Point[Pixels]{})

	scrollOffset.X = clampPixels(scrollOffset.X, -scrollMax.X, Px(0.))
	if scrollToBottom {
		scrollOffset.Y = -scrollMax.Y
	} else {
		scrollOffset.Y = clampPixels(scrollOffset.Y, -scrollMax.Y, Px(0.))
	}

	if scrollHandle, ok := i.trackedScrollHandle.Get(); ok {
		scrollHandle.state.maxOffset = scrollMax
		scrollHandle.state.bounds = bounds
	}

	return *scrollOffset
}

// Paint paints this element according to this interactivity state's configured styles and binds
// the element's mouse and keyboard events; f receives the final computed style.
func (i *Interactivity) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], hitbox *Hitbox, window *Window, cx *App, f func(style *Style, window *Window, cx *App)) {
	if hitbox != nil {
		i.Hovered = Some(hitbox.IsHovered(window))
	} else {
		i.Hovered = None[bool]()
	}
	window.WithOptionalElementState(globalID, func(state Option[Option[InteractiveElementState]], window *Window) (struct{}, Option[InteractiveElementState]) {
		elementState := interactiveElementStateOrDefault(state)

		style := i.computeStyleInternal(hitbox, interactiveElementStateRef(&elementState), window, cx)

		if debugSelector, ok := i.debugSelector.Get(); ok {
			window.nextFrame.recordDebugBounds(debugSelector, bounds)
		}

		i.paintHoverGroupHandler(window, cx)

		if style.Visibility == VisibilityHidden {
			return struct{}{}, elementState
		}

		tabGroup := None[int]()
		if i.tabGroup {
			tabGroup = i.tabIndex
		}

		window.withElementOpacity(style.Opacity, func(window *Window) struct{} {
			style.Paint(bounds, window, cx, func(window *Window, cx *App) {
				window.WithTextStyle(optionalTextStyle(&style), func(window *Window) struct{} {
					return window.WithContentMask(optionalOverflowMask(&style, bounds, window.RemSize()), func(window *Window) struct{} {
						return window.WithTabGroup(tabGroup, func(window *Window) struct{} {
							if focusHandle, ok := i.trackedFocusHandle.Get(); ok {
								window.nextFrame.tabStops.insert(&focusHandle)
							}
							if hitbox != nil {
								if gpuiutil.DebugAssertions {
									i.paintDebugInfo(globalID, hitbox, &style, window, cx)
								}

								if drag, ok := cx.activeDrag.Get(); ok {
									if mouseCursor, ok := drag.CursorStyle.Get(); ok {
										window.SetWindowCursorStyle(mouseCursor)
									}
								} else if mouseCursor, ok := style.MouseCursor.Get(); ok {
									window.SetCursorStyle(mouseCursor, *hitbox)
								}

								if group, ok := i.group.Get(); ok {
									groupHitboxesPush(group, hitbox.ID, cx)
								}

								if area, ok := i.windowControl.Get(); ok {
									window.InsertWindowControlHitbox(area, *hitbox)
								}

								i.paintMouseListeners(hitbox, interactiveElementStateRef(&elementState), window, cx)
								i.paintScrollListener(hitbox, &style, window, cx)
							}

							i.paintKeyboardListeners(window, cx)

							if window.a11y.isActive() {
								if globalID != nil {
									if len(i.a11yActionListeners) != 0 {
										nodeID := globalID.accesskitNodeID()
										listeners := i.a11yActionListeners
										i.a11yActionListeners = nil
										for _, entry := range listeners {
											window.OnA11yAction(nodeID, entry.action, entry.listener)
										}
									}
								}
							}

							f(&style, window, cx)

							if hitbox != nil {
								if gpuiutil.DebugAssertions {
									window.InsertInspectorHitbox(hitbox.ID, inspectorID, cx)
								}

								if group, ok := i.group.Get(); ok {
									groupHitboxesPop(group, cx)
								}
							}
							return struct{}{}
						})
					})
				})
			})
			return struct{}{}
		})

		return struct{}{}, elementState
	})
}

const debugInfoFontSize Pixels = 10.

func (i *Interactivity) paintDebugInfo(globalID *GlobalElementID, hitbox *Hitbox, style *Style, window *Window, cx *App) {
	if globalID == nil || !(style.Debug || style.DebugBelow || cx.HasGlobal[DebugBelow]()) || !hitbox.IsHovered(window) {
		return
	}
	elementID := fmt.Sprintf("%#v", globalID.ids)
	strLen := len(elementID)

	renderDebugText := func(window *Window) struct{} {
		lines, err := window.TextSystem().ShapeText(SharedString(elementID), debugInfoFontSize, []TextRun{window.TextStyle().ToRun(strLen)}, None[Pixels](), None[int]())
		if err != nil || len(lines) == 0 {
			return struct{}{}
		}
		text := lines[len(lines)-1]
		_ = text.Paint(hitbox.Origin, debugInfoFontSize, TextAlignLeft, None[Bounds[Pixels]](), window, cx)

		textBounds := Bounds[Pixels]{Origin: hitbox.Origin, Size: text.Size(debugInfoFontSize)}
		sourceLocation := i.sourceLocation
		if sourceLocation != nil && textBounds.Contains(window.MousePosition()) && window.Modifiers().Secondary(cx) {
			secondaryHeld := window.Modifiers().Secondary(cx)
			window.OnKeyEvent(func(e *ModifiersChangedEvent, _ DispatchPhase, window *Window, cx *App) {
				if e.Modifiers.Secondary(cx) != secondaryHeld && textBounds.Contains(window.MousePosition()) {
					window.Refresh()
				}
			})

			wasHovered := hitbox.IsHovered(window)
			currentView := window.CurrentView()
			moveHitbox := *hitbox
			window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
				if phase == DispatchPhaseCapture {
					hovered := moveHitbox.IsHovered(window)
					if hovered != wasHovered {
						cx.Notify(currentView)
					}
				}
			})

			downHitbox := *hitbox
			window.OnMouseEvent(func(e *MouseDownEvent, phase DispatchPhase, window *Window, cx *App) {
				if textBounds.Contains(e.Position) && phase.Capture() && downHitbox.IsHovered(window) {
					cx.StopPropagation()
					dir, err := os.Getwd()
					if err != nil {
						return
					}

					fmt.Fprintf(os.Stderr, "This element was created at:\n%s:%d:%d\n", filepath.Join(dir, sourceLocation.File), sourceLocation.Line, sourceLocation.Column)
				}
			})
			window.PaintQuad(Outline(Bounds[Pixels]{
				Origin: hitbox.Origin.Add(NewPoint(Px(0.), debugInfoFontSize-Px(2.))),
				Size:   Size[Pixels]{Width: textBounds.Size.Width, Height: Px(1.)},
			}, Red(), BorderStyle(0)))
		}
		return struct{}{}
	}

	window.WithTextStyle(Some(TextStyleRefinement{
		Color:           Some(Red()),
		LineHeight:      Some(debugInfoFontSize.ToDefiniteLength()),
		BackgroundColor: Some(White()),
	}), renderDebugText)
}

func (i *Interactivity) paintMouseListeners(hitbox *Hitbox, elementState *InteractiveElementState, window *Window, cx *App) {
	isFocused := false
	if handle, ok := i.trackedFocusHandle.Get(); ok {
		isFocused = handle.IsFocused(window)
	}

	if focusHandle, ok := i.trackedFocusHandle.Get(); ok {
		weakFocusHandle := focusHandle.Downgrade()
		hitbox := *hitbox
		window.OnMouseEvent(func(_ *MouseDownEvent, phase DispatchPhase, window *Window, cx *App) {
			if phase == DispatchPhaseBubble && hitbox.IsHovered(window) && !window.DefaultPrevented() {
				if focusHandle, ok := weakFocusHandle.Upgrade(); ok {
					window.Focus(focusHandle, cx)
					focusHandle.Release()
				}
				window.PreventDefault()
			}
		})
	}

	for _, listener := range i.mouseDownListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *MouseDownEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.mouseDownListeners = nil

	for _, listener := range i.mouseUpListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *MouseUpEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.mouseUpListeners = nil

	for _, listener := range i.mousePressureListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *MousePressureEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.mousePressureListeners = nil

	for _, listener := range i.mouseMoveListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.mouseMoveListeners = nil

	for _, listener := range i.mouseExitListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *MouseExitEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.mouseExitListeners = nil

	for _, listener := range i.fileDropExitListeners {
		hitbox := *hitbox
		window.invalidator.debugAssertPaint()
		window.nextFrame.mouseListeners = append(window.nextFrame.mouseListeners, func(event any, phase DispatchPhase, window *Window, cx *App) {
			if event, ok := event.(*FileDropEvent); ok {
				listener(*event, phase, &hitbox, window, cx)
			}
		})
	}
	i.fileDropExitListeners = nil

	for _, listener := range i.scrollListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *ScrollEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.scrollListeners = nil

	for _, listener := range i.pinchListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *PinchEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.pinchListeners = nil

	for _, listener := range i.backGestureListeners {
		hitbox := *hitbox
		window.OnMouseEvent(func(event *BackGestureEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, &hitbox, window, cx)
		})
	}
	i.backGestureListeners = nil

	if i.hoverStyle != nil || i.BaseStyle.MouseCursor.IsSome() || cx.activeDrag.IsSome() && len(i.dragOverStyles) != 0 {
		hitbox := *hitbox
		var hoverState *ElementHoverState
		if i.hoverStyle != nil && elementState != nil {
			hoverState = elementState.hoverState
		}
		currentView := window.CurrentView()

		window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
			hovered := hitbox.IsHovered(window)
			wasHovered := hoverState != nil && hoverState.Element
			if phase == DispatchPhaseCapture && hovered != wasHovered {
				if hoverState != nil {
					hoverState.Element = hovered
					cx.Notify(currentView)
				}
			}
		})
	}

	if groupHover := i.groupHoverStyle; groupHover != nil {
		if groupHitboxID, ok := GroupHitboxesGet(groupHover.Group, cx); ok {
			var hoverState *ElementHoverState
			if elementState != nil {
				hoverState = elementState.hoverState
			}
			currentView := window.CurrentView()

			window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
				groupHovered := groupHitboxID.IsHovered(window)
				wasGroupHovered := hoverState != nil && hoverState.Group
				if phase == DispatchPhaseCapture && groupHovered != wasGroupHovered {
					if hoverState != nil {
						hoverState.Group = groupHovered
						cx.Notify(currentView)
					}
				}
			})
		}
	}

	dragCursorStyle := i.BaseStyle.MouseCursor

	dragListener := i.dragListener
	i.dragListener = nil
	dropListeners := i.dropListeners
	i.dropListeners = nil
	clickListeners := i.clickListeners
	i.clickListeners = nil
	auxClickListeners := i.auxClickListeners
	i.auxClickListeners = nil
	canDropPredicate := i.canDropPredicate
	i.canDropPredicate = nil

	if len(dropListeners) != 0 {
		hitbox := *hitbox
		window.OnMouseEvent(func(_ *MouseUpEvent, phase DispatchPhase, window *Window, cx *App) {
			drag, ok := cx.activeDrag.Get()
			if ok && phase == DispatchPhaseBubble && hitbox.IsHovered(window) {
				dragStateType := reflect.TypeOf(drag.Value)
				for _, entry := range dropListeners {
					if entry.valueType == dragStateType {
						drag := cx.activeDrag.Take().Expect("checked for type drag state type above")

						canDrop := true
						if canDropPredicate != nil {
							canDrop = canDropPredicate(drag.Value, window, cx)
						}

						if canDrop {
							entry.listener(drag.Value, window, cx)
							window.Refresh()
							cx.StopPropagation()
						}
						drag.View.Release()
					}
				}
			}
		})
	}

	if elementState == nil {
		return
	}
	if len(clickListeners) != 0 || len(auxClickListeners) != 0 || dragListener != nil {
		if elementState.pendingMouseDown == nil {
			elementState.pendingMouseDown = &Option[MouseDownEvent]{}
		}
		pendingMouseDown := elementState.pendingMouseDown

		if elementState.pendingKeyboardDown == nil {
			elementState.pendingKeyboardDown = &Option[uint64]{}
		}
		pendingKeyboardDown := elementState.pendingKeyboardDown

		if elementState.clickedState == nil {
			elementState.clickedState = &ElementClickedState{}
		}
		clickedState := elementState.clickedState

		{
			hitbox := *hitbox
			hasAuxClickListeners := len(auxClickListeners) != 0
			window.OnMouseEvent(func(event *MouseDownEvent, phase DispatchPhase, window *Window, _ *App) {
				if phase == DispatchPhaseBubble && (event.Button == MouseButtonLeft || hasAuxClickListeners) && hitbox.IsHovered(window) {
					*pendingMouseDown = Some(*event)
					window.Refresh()
				}
			})
		}

		{
			hitbox := *hitbox
			window.OnMouseEvent(func(event *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
				if phase == DispatchPhaseCapture {
					return
				}

				mouseDown, ok := pendingMouseDown.Get()
				if !ok || cx.HasActiveDrag() || event.Position.Sub(mouseDown.Position).Magnitude() <= dragThreshold || dragListener == nil {
					return
				}
				listener := dragListener
				dragListener = nil
				if mouseDown.Button != MouseButtonLeft {
					return
				}
				*clickedState = ElementClickedState{}
				cursorOffset := event.Position.Sub(hitbox.Origin)
				drag := listener.render(listener.value, cursorOffset, window, cx)
				externalPayloadSource := None[ExternalDragPayloadSource]()
				if externalPayload := listener.externalPayload; externalPayload != nil {
					value := listener.value
					externalPayloadSource = Some(ExternalDragPayloadSource(func(window *Window, cx *App) (ExternalDragPayload, bool) {
						return externalPayload(value, window, cx)
					}))
				}
				cx.activeDrag = Some(AnyDrag{
					View:                  drag,
					Value:                 listener.value,
					CursorOffset:          cursorOffset,
					CursorStyle:           dragCursorStyle,
					ExternalPayloadSource: externalPayloadSource,
				})
				*pendingMouseDown = None[MouseDownEvent]()
				window.Refresh()
				cx.StopPropagation()
			})
		}

		if isFocused {
			window.OnKeyEvent(func(event *KeyDownEvent, phase DispatchPhase, window *Window, _ *App) {
				if phase.Bubble() && !window.DefaultPrevented() {
					stroke := &event.Keystroke
					isActivationKey := (stroke.Key == "enter" || stroke.Key == "space") && !stroke.Modifiers.Modified()
					if isActivationKey {
						*pendingKeyboardDown = Some(window.focusGeneration)
					} else {
						*pendingKeyboardDown = None[uint64]()
					}
				}
			})

			{
				hitbox := *hitbox
				window.OnKeyEvent(func(event *KeyUpEvent, phase DispatchPhase, window *Window, cx *App) {
					if phase.Bubble() && !window.DefaultPrevented() {
						stroke := &event.Keystroke
						keyboardButton := None[KeyboardButton]()
						switch stroke.Key {
						case "enter":
							keyboardButton = Some(KeyboardButtonEnter)
						case "space":
							keyboardButton = Some(KeyboardButtonSpace)
						}

						if button, ok := keyboardButton.Get(); ok && !stroke.Modifiers.Modified() {
							pending := pendingKeyboardDown.Take()
							if pending != Some(window.focusGeneration) {
								return
							}

							clickEvent := ClickEventKeyboard(KeyboardClickEvent{
								Button: button,
								Bounds: hitbox.Bounds,
							})

							for _, listener := range clickListeners {
								listener(&clickEvent, window, cx)
							}
						} else {
							*pendingKeyboardDown = None[uint64]()
						}
					}
				})
			}
		}

		{
			capturedMouseDown := None[MouseDownEvent]()
			hitbox := *hitbox
			window.OnMouseEvent(func(event *MouseUpEvent, phase DispatchPhase, window *Window, cx *App) {
				switch phase {
				case DispatchPhaseCapture:
					if pendingMouseDown.IsSome() && hitbox.IsHovered(window) {
						capturedMouseDown = pendingMouseDown.Take()
						window.Refresh()
					} else if pendingMouseDown.IsSome() {
						pendingMouseDown.Take()
						window.Refresh()
					}
				case DispatchPhaseBubble:
					if mouseDown, ok := capturedMouseDown.Take().Get(); ok {
						btn := mouseDown.Button

						mouseClick := ClickEventMouse(MouseClickEvent{
							Down: mouseDown,
							Up:   *event,
						})

						if btn == MouseButtonLeft {
							for _, listener := range clickListeners {
								listener(&mouseClick, window, cx)
							}
						} else {
							for _, listener := range auxClickListeners {
								listener(&mouseClick, window, cx)
							}
						}
					}
				}
			})
		}
	}

	if hoverListener := i.hoverListener; hoverListener != nil {
		i.hoverListener = nil
		if elementState.hoverListenerState == nil {
			elementState.hoverListenerState = new(bool)
		}
		wasHovered := elementState.hoverListenerState
		if elementState.pendingMouseDown == nil {
			elementState.pendingMouseDown = &Option[MouseDownEvent]{}
		}
		hasMouseDown := elementState.pendingMouseDown
		hoverListenerState := wasHovered
		updateHover := func(isHovered bool, window *Window, cx *App) {
			if isHovered != *hoverListenerState {
				*hoverListenerState = isHovered
				hoverListener(isHovered, window, cx)
			}
		}
		hoverListenerMode := i.hoverListenerMode

		if hasMouseDown.IsNone() {
			isHovered := !cx.HasActiveDrag() && hoverListenerMode.isHovered(hitbox, window)
			if isHovered != *wasHovered {
				window.Defer(cx, func(window *Window, cx *App) {
					updateHover(isHovered, window, cx)
				})
			}
		}

		{
			hitbox := *hitbox
			window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
				if phase == DispatchPhaseBubble {
					isHovered := hasMouseDown.IsNone() && !cx.HasActiveDrag() && hoverListenerMode.isHovered(&hitbox, window)
					updateHover(isHovered, window, cx)
				}
			})
		}

		window.OnMouseEvent(func(_ *MouseExitEvent, phase DispatchPhase, window *Window, cx *App) {
			if phase == DispatchPhaseBubble {
				updateHover(false, window, cx)
			}
		})
	}

	if tooltipBuilder := i.tooltipBuilder; tooltipBuilder != nil {
		i.tooltipBuilder = nil
		if elementState.activeTooltip == nil {
			elementState.activeTooltip = &activeTooltipCell{}
		}
		activeTooltip := elementState.activeTooltip
		if elementState.pendingMouseDown == nil {
			elementState.pendingMouseDown = &Option[MouseDownEvent]{}
		}
		pendingMouseDown := elementState.pendingMouseDown
		if elementState.longPressTooltipActive == nil {
			elementState.longPressTooltipActive = new(bool)
		}
		longPressTooltipActive := elementState.longPressTooltipActive

		tooltipIsHoverable := tooltipBuilder.hoverable
		buildTooltip := func(window *Window, cx *App) (AnyView, bool, bool) {
			return tooltipBuilder.build(window, cx), tooltipIsHoverable, true
		}
		sourceBounds := hitbox.Bounds
		checkIsHoveredDuringPrepaint := func(window *Window) bool {
			return !window.LastInputWasKeyboard() && pendingMouseDown.IsNone() && sourceBounds.Contains(window.MousePosition())
		}
		checkHitbox := *hitbox
		checkIsHovered := func(window *Window) bool {
			return pendingMouseDown.IsNone() && checkHitbox.IsHovered(window)
		}
		registerTooltipMouseHandlers(activeTooltip, i.tooltipID, buildTooltip, checkIsHovered, checkIsHoveredDuringPrepaint, longPressTooltipActive, i.tooltipShowDelay, window)
	}

	if elementState.clickedState == nil {
		elementState.clickedState = &ElementClickedState{}
	}
	activeState := elementState.clickedState

	window.OnMouseEvent(func(_ *MouseUpEvent, phase DispatchPhase, window *Window, _ *App) {
		if phase == DispatchPhaseCapture && activeState.isClicked() {
			*activeState = ElementClickedState{}
			window.Refresh()
		}
	})

	{
		activeGroupHitbox := None[HitboxID]()
		if groupActive := i.groupActiveStyle; groupActive != nil {
			if id, ok := GroupHitboxesGet(groupActive.Group, cx); ok {
				activeGroupHitbox = Some(id)
			}
		}
		hitbox := *hitbox
		window.OnMouseEvent(func(_ *MouseDownEvent, phase DispatchPhase, window *Window, _ *App) {
			if phase == DispatchPhaseBubble && !window.DefaultPrevented() {
				groupHitboxID, ok := activeGroupHitbox.Get()
				groupHovered := ok && groupHitboxID.IsHovered(window)
				elementHovered := hitbox.IsHovered(window)
				if groupHovered || elementHovered {
					*activeState = ElementClickedState{
						Group:   groupHovered,
						Element: elementHovered,
					}
					window.Refresh()
				}
			}
		})
	}
}

func (i *Interactivity) paintKeyboardListeners(window *Window, _ *App) {
	keyDownListeners := i.keyDownListeners
	i.keyDownListeners = nil
	keyUpListeners := i.keyUpListeners
	i.keyUpListeners = nil
	modifiersChangedListeners := i.modifiersChangedListeners
	i.modifiersChangedListeners = nil
	actionListeners := i.actionListeners
	i.actionListeners = nil
	if context, ok := i.keyContext.Get(); ok {
		window.SetKeyContext(context)
	}

	for _, listener := range keyDownListeners {
		window.OnKeyEvent(func(event *KeyDownEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, window, cx)
		})
	}

	for _, listener := range keyUpListeners {
		window.OnKeyEvent(func(event *KeyUpEvent, phase DispatchPhase, window *Window, cx *App) {
			listener(event, phase, window, cx)
		})
	}

	for _, listener := range modifiersChangedListeners {
		window.OnModifiersChanged(func(event *ModifiersChangedEvent, window *Window, cx *App) {
			listener(event, window, cx)
		})
	}

	for _, entry := range actionListeners {
		window.OnAction(entry.actionType, entry.listener)
	}
}

func (i *Interactivity) paintHoverGroupHandler(window *Window, cx *App) {
	groupHover := i.groupHoverStyle
	if groupHover == nil {
		return
	}
	groupHitbox, ok := GroupHitboxesGet(groupHover.Group, cx)
	if !ok {
		return
	}
	wasHovered := groupHitbox.IsHovered(window)
	currentView := window.CurrentView()
	window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
		hovered := groupHitbox.IsHovered(window)
		if phase == DispatchPhaseCapture && hovered != wasHovered {
			cx.Notify(currentView)
		}
	})
}

func (i *Interactivity) paintScrollListener(hitbox *Hitbox, style *Style, window *Window, _ *App) {
	scrollOffset := i.scrollOffset
	if scrollOffset == nil {
		return
	}
	ongoingScroll := i.ongoingScroll
	overflow := style.Overflow
	allowConcurrentScroll := style.AllowConcurrentScroll
	restrictScrollToAxis := style.RestrictScrollToAxis
	lineHeight := window.LineHeight()
	h := *hitbox
	currentView := window.CurrentView()
	window.OnMouseEvent(func(event *ScrollEvent, phase DispatchPhase, window *Window, cx *App) {
		if phase == DispatchPhaseBubble && h.ShouldHandleScroll(window) {
			oldScrollOffset := *scrollOffset
			delta := event.Delta.PixelDelta(lineHeight)

			if restrictScrollToAxis && event.Delta.Precise() && ongoingScroll != nil {
				ongoingScroll.Filter(&delta, event.TouchPhase)
			}

			deltaX := PixelsZero
			if overflow.X == OverflowScroll {
				if !delta.X.IsZero() {
					deltaX = delta.X
				} else if !restrictScrollToAxis && overflow.Y != OverflowScroll {
					deltaX = delta.Y
				}
			}
			deltaY := PixelsZero
			if overflow.Y == OverflowScroll {
				if !delta.Y.IsZero() {
					deltaY = delta.Y
				} else if !restrictScrollToAxis && overflow.X != OverflowScroll {
					deltaY = delta.X
				}
			}
			if !allowConcurrentScroll && !deltaX.IsZero() && !deltaY.IsZero() {
				if deltaX.Abs() > deltaY.Abs() {
					deltaY = PixelsZero
				} else {
					deltaX = PixelsZero
				}
			}
			scrollOffset.Y += deltaY
			scrollOffset.X += deltaX
			if *scrollOffset != oldScrollOffset {
				cx.Notify(currentView)
			}
		}
	})
}

// ComputeStyle computes the visual style for this element, based on the current bounds and the
// element's state.
func (i *Interactivity) ComputeStyle(globalID *GlobalElementID, hitbox *Hitbox, window *Window, cx *App) Style {
	return window.WithOptionalElementState(globalID, func(state Option[Option[InteractiveElementState]], window *Window) (Style, Option[InteractiveElementState]) {
		elementState := interactiveElementStateOrDefault(state)
		style := i.computeStyleInternal(hitbox, interactiveElementStateRef(&elementState), window, cx)
		return style, elementState
	})
}

func (i *Interactivity) computeStyleInternal(hitbox *Hitbox, elementState *InteractiveElementState, window *Window, cx *App) Style {
	style := DefaultStyle()
	style.Refine(&i.BaseStyle)

	if focusHandle, ok := i.trackedFocusHandle.Get(); ok {
		if i.inFocusStyle != nil && focusHandle.WithinFocused(window, cx) {
			style.Refine(i.inFocusStyle)
		}

		if i.focusStyle != nil && focusHandle.IsFocused(window) {
			style.Refine(i.focusStyle)
		}

		if i.focusVisibleStyle != nil && focusHandle.IsFocused(window) && window.LastInputWasKeyboard() {
			style.Refine(i.focusVisibleStyle)
		}
	}

	if !cx.HasActiveDrag() {
		if groupHover := i.groupHoverStyle; groupHover != nil {
			isGroupHovered := false
			if groupHitboxID, ok := GroupHitboxesGet(groupHover.Group, cx); ok {
				isGroupHovered = !window.LastInputWasTouch() && groupHitboxID.IsHovered(window)
			} else if elementState != nil {
				isGroupHovered = !window.LastInputWasTouch() && elementState.hoverState != nil && elementState.hoverState.Group
			}

			if isGroupHovered {
				style.Refine(groupHover.Style)
			}
		}

		if i.hoverStyle != nil {
			isHovered := false
			if hitbox != nil {
				isHovered = !window.LastInputWasTouch() && hitbox.IsHovered(window)
			} else if elementState != nil {
				isHovered = !window.LastInputWasTouch() && elementState.hoverState != nil && elementState.hoverState.Element
			}

			if isHovered {
				style.Refine(i.hoverStyle)
			}
		}
	}

	if hitbox != nil {
		if drag, ok := cx.activeDrag.Take().Get(); ok {
			canDrop := true
			if i.canDropPredicate != nil {
				canDrop = i.canDropPredicate(drag.Value, window, cx)
			}

			if canDrop {
				dragType := reflect.TypeOf(drag.Value)
				for _, groupDragStyle := range i.groupDragOverStyles {
					if groupHitboxID, ok := GroupHitboxesGet(groupDragStyle.style.Group, cx); ok && groupDragStyle.valueType == dragType && groupHitboxID.IsHovered(window) {
						style.Refine(groupDragStyle.style.Style)
					}
				}

				for _, dragOver := range i.dragOverStyles {
					if dragOver.valueType == dragType && hitbox.IsHovered(window) {
						style.Refine(dragOver.style(drag.Value, window, cx))
					}
				}
			}

			style.MouseCursor = drag.CursorStyle
			cx.activeDrag = Some(drag)
		}
	}

	if elementState != nil {
		if elementState.clickedState == nil {
			elementState.clickedState = &ElementClickedState{}
		}
		clickedState := *elementState.clickedState
		if clickedState.Group && i.groupActiveStyle != nil {
			style.Refine(i.groupActiveStyle.Style)
		}

		if i.activeStyle != nil && clickedState.Element {
			style.Refine(i.activeStyle)
		}
	}

	return style
}

func (i *Interactivity) writeA11yInfo(node *a11y.Node) {
	if id, ok := i.aria.authorID.Get(); ok {
		node.SetAuthorID(string(id))
	}
	if label, ok := i.aria.label.Get(); ok {
		node.SetLabel(string(label))
	}
	if description, ok := i.aria.description.Get(); ok {
		node.SetDescription(string(description))
	}
	if keyshortcuts, ok := i.aria.keyshortcuts.Get(); ok {
		node.SetKeyboardShortcut(string(keyshortcuts))
	}
	if selected, ok := i.aria.selected.Get(); ok {
		node.SetSelected(selected)
	}
	if expanded, ok := i.aria.expanded.Get(); ok {
		node.SetExpanded(expanded)
	}
	if toggled, ok := i.aria.toggled.Get(); ok {
		node.SetToggled(toggled)
	}
	if value, ok := i.aria.numericValue.Get(); ok {
		node.SetNumericValue(value)
	}
	if value, ok := i.aria.minNumericValue.Get(); ok {
		node.SetMinNumericValue(value)
	}
	if value, ok := i.aria.maxNumericValue.Get(); ok {
		node.SetMaxNumericValue(value)
	}
	if step, ok := i.aria.numericValueStep.Get(); ok {
		node.SetNumericValueStep(step)
	}
	if value, ok := i.aria.value.Get(); ok {
		node.SetValue(string(value))
	}
	if placeholder, ok := i.aria.placeholder.Get(); ok {
		node.SetPlaceholder(string(placeholder))
	}
	if orientation, ok := i.aria.orientation.Get(); ok {
		node.SetOrientation(orientation)
	}
	if level, ok := i.aria.level.Get(); ok {
		node.SetLevel(level)
	}
	if position, ok := i.aria.positionInSet.Get(); ok {
		node.SetPositionInSet(position)
	}
	if size, ok := i.aria.sizeOfSet.Get(); ok {
		node.SetSizeOfSet(size)
	}
	if index, ok := i.aria.rowIndex.Get(); ok {
		node.SetRowIndex(index)
	}
	if index, ok := i.aria.columnIndex.Get(); ok {
		node.SetColumnIndex(index)
	}
	if count, ok := i.aria.rowCount.Get(); ok {
		node.SetRowCount(count)
	}
	if count, ok := i.aria.columnCount.Get(); ok {
		node.SetColumnCount(count)
	}
	if len(i.clickListeners) != 0 {
		node.AddAction(a11y.ActionClick)
	}
	if i.trackedFocusHandle.IsSome() || i.focusable {
		node.AddAction(a11y.ActionFocus)
	}
	for _, entry := range i.a11yActionListeners {
		node.AddAction(entry.action)
	}
}

// InteractiveElementState is the per-frame state of an interactive element, used for tracking
// stateful interactions like clicks and scroll offsets.
type InteractiveElementState struct {
	focusHandle            Option[FocusHandle]
	clickedState           *ElementClickedState
	hoverState             *ElementHoverState
	hoverListenerState     *bool
	pendingMouseDown       *Option[MouseDownEvent]
	pendingKeyboardDown    *Option[uint64]
	scrollOffset           *Point[Pixels]
	ongoingScroll          *OngoingScroll
	activeTooltip          *activeTooltipCell
	longPressTooltipActive *bool
}

// ElementClickedState is whether the element or a group that contains it is clicked by the mouse.
type ElementClickedState struct {
	// Group is true if this element's group has been clicked.
	Group bool
	// Element is true if this element has been clicked.
	Element bool
}

func (s ElementClickedState) isClicked() bool {
	return s.Group || s.Element
}

// ElementHoverState is whether the element or a group that contains it is hovered.
type ElementHoverState struct {
	// Group is true if this element's group is hovered.
	Group bool
	// Element is true if this element is hovered.
	Element bool
}

func (s *InteractiveElementState) dropElementState() {
	if s.activeTooltip != nil {
		s.activeTooltip.set(None[activeTooltipState]())
	}
	if focusHandle, ok := s.focusHandle.Get(); ok {
		focusHandle.Release()
		s.focusHandle = None[FocusHandle]()
	}
}

type activeTooltipKind uint8

const (
	activeTooltipWaitingForShow activeTooltipKind = iota
	activeTooltipVisible
	activeTooltipWaitingForHide
)

type activeTooltipState struct {
	kind        activeTooltipKind
	task        *Task[struct{}]
	tooltip     AnyTooltip
	isHoverable bool
}

func (t activeTooltipState) drop() {
	if t.task != nil {
		t.task.Cancel()
	}
	if t.kind != activeTooltipWaitingForShow {
		t.tooltip.View.Release()
	}
}

func cloneAnyTooltip(tooltip AnyTooltip) AnyTooltip {
	tooltip.View = tooltip.View.Clone()
	return tooltip
}

type activeTooltipCell struct {
	value Option[activeTooltipState]
}

func (c *activeTooltipCell) take() Option[activeTooltipState] {
	return c.value.Take()
}

func (c *activeTooltipCell) replace(value activeTooltipState) {
	if old, ok := c.value.Replace(value).Get(); ok {
		old.drop()
	}
}

func (c *activeTooltipCell) set(value Option[activeTooltipState]) {
	if old, ok := c.value.Get(); ok {
		old.drop()
	}
	c.value = value
}

func clearActiveTooltip(activeTooltip *activeTooltipCell, window *Window) {
	old, ok := activeTooltip.take().Get()
	if !ok {
		return
	}
	old.drop()
	switch old.kind {
	case activeTooltipWaitingForShow:
	case activeTooltipVisible, activeTooltipWaitingForHide:
		window.Refresh()
	}
}

func clearActiveTooltipIfNotHoverable(activeTooltip *activeTooltipCell, window *Window) {
	shouldClear := false
	if current, ok := activeTooltip.value.Get(); ok && current.kind == activeTooltipVisible {
		shouldClear = !current.isHoverable
	}
	if shouldClear {
		activeTooltip.set(None[activeTooltipState]())
		window.Refresh()
	}
}

func setTooltipOnWindow(activeTooltip *activeTooltipCell, window *Window) Option[TooltipID] {
	current, ok := activeTooltip.value.Get()
	if !ok || current.kind == activeTooltipWaitingForShow {
		return None[TooltipID]()
	}
	return Some(window.SetTooltip(cloneAnyTooltip(current.tooltip)))
}

type tooltipBuildFunc func(window *Window, cx *App) (view AnyView, hoverable bool, ok bool)

func registerTooltipMouseHandlers(activeTooltip *activeTooltipCell, tooltipID Option[TooltipID], buildTooltip tooltipBuildFunc, checkIsHovered func(window *Window) bool, checkIsHoveredDuringPrepaint func(window *Window) bool, longPressTooltipActive *bool, showDelay Option[time.Duration], window *Window) {
	currentView := window.CurrentView()
	delay := showDelay.UnwrapOr(defaultTooltipShowDelay)

	window.OnMouseEvent(func(_ *MouseMoveEvent, phase DispatchPhase, window *Window, cx *App) {
		handleTooltipMouseMove(activeTooltip, buildTooltip, checkIsHovered, checkIsHoveredDuringPrepaint, tooltipID, currentView, phase, delay, window, cx)
	})

	window.OnMouseEvent(func(_ *MouseDownEvent, _ DispatchPhase, window *Window, _ *App) {
		if id, ok := tooltipID.Get(); !ok || !id.IsHovered(window) {
			clearActiveTooltipIfNotHoverable(activeTooltip, window)
		}
	})

	window.OnMouseEvent(func(event *LongPressEvent, phase DispatchPhase, window *Window, cx *App) {
		if !phase.Bubble() {
			return
		}

		switch event.Phase {
		case TouchPhaseStarted:
			if !window.DefaultPrevented() && checkIsHovered(window) {
				if showTooltip(activeTooltip, buildTooltip, checkIsHoveredDuringPrepaint, longPressTooltipActive, window, cx) {
					*longPressTooltipActive = true
					window.PreventDefault()
					cx.StopPropagation()
				}
			}
		case TouchPhaseEnded, TouchPhaseCancelled:
			wasActive := *longPressTooltipActive
			*longPressTooltipActive = false
			if wasActive {
				clearActiveTooltip(activeTooltip, window)
				cx.StopPropagation()
			}
		}
	})

	window.OnMouseEvent(func(_ *ScrollEvent, _ DispatchPhase, window *Window, _ *App) {
		if id, ok := tooltipID.Get(); !ok || !id.IsHovered(window) {
			clearActiveTooltipIfNotHoverable(activeTooltip, window)
		}
	})
}

type tooltipMouseMoveAction uint8

const (
	tooltipMouseMoveNone tooltipMouseMoveAction = iota
	tooltipMouseMoveCancelShow
	tooltipMouseMoveScheduleShow
	tooltipMouseMoveCheckVisible
)

func handleTooltipMouseMove(activeTooltip *activeTooltipCell, buildTooltip tooltipBuildFunc, checkIsHovered func(window *Window) bool, checkIsHoveredDuringPrepaint func(window *Window) bool, tooltipID Option[TooltipID], currentView EntityID, phase DispatchPhase, showDelay time.Duration, window *Window, cx *App) {
	tooltipHovered := func() bool {
		id, ok := tooltipID.Get()
		return ok && id.IsHovered(window)
	}
	action := tooltipMouseMoveNone
	current, ok := activeTooltip.value.Get()
	switch {
	case !ok:
		if checkIsHovered(window) && phase.Bubble() {
			action = tooltipMouseMoveScheduleShow
		}
	case current.kind == activeTooltipWaitingForShow:
		if !checkIsHovered(window) {
			action = tooltipMouseMoveCancelShow
		}
	case current.kind == activeTooltipVisible:
		if phase.Capture() && !checkIsHovered(window) && (!current.isHoverable || !tooltipHovered()) {
			action = tooltipMouseMoveCheckVisible
		}
	case current.kind == activeTooltipWaitingForHide:
		if phase.Capture() && (checkIsHovered(window) || tooltipHovered()) {
			action = tooltipMouseMoveCheckVisible
		}
	}

	switch action {
	case tooltipMouseMoveNone:
	case tooltipMouseMoveCancelShow:
		activeTooltip.set(None[activeTooltipState]())
	case tooltipMouseMoveScheduleShow:
		weakActiveTooltip := weak.Make(activeTooltip)
		delayedShowTask := windowSpawn(window, cx, func(cx *AsyncWindowContext) {
			_, _ = cx.BackgroundExecutor().Timer(showDelay).Await(cx)
			activeTooltip := weakActiveTooltip.Value()
			if activeTooltip == nil {
				return
			}
			_ = cx.Update(func(window *Window, cx *App) {
				showTooltip(activeTooltip, buildTooltip, checkIsHoveredDuringPrepaint, nil, window, cx)
			})
		})
		activeTooltip.replace(activeTooltipState{kind: activeTooltipWaitingForShow, task: delayedShowTask})
	case tooltipMouseMoveCheckVisible:
		cx.Notify(currentView)
	}
}

func windowSpawn(window *Window, cx *App, f func(cx *AsyncWindowContext)) *Task[struct{}] {
	return window.Spawn(cx, func(cx *AsyncWindowContext) (struct{}, error) {
		f(cx)
		return struct{}{}, nil
	})
}

func showTooltip(activeTooltip *activeTooltipCell, buildTooltip tooltipBuildFunc, checkIsHoveredDuringPrepaint func(window *Window) bool, longPressTooltipActive *bool, window *Window, cx *App) bool {
	newTooltip := None[activeTooltipState]()
	if view, tooltipIsHoverable, ok := buildTooltip(window, cx); ok {
		weakActiveTooltip := weak.Make(activeTooltip)
		newTooltip = Some(activeTooltipState{
			kind: activeTooltipVisible,
			tooltip: AnyTooltip{
				View:          view,
				MousePosition: window.MousePosition(),
				CheckVisibleAndUpdate: func(tooltipBounds Bounds[Pixels], window *Window, cx *App) bool {
					if longPressTooltipActive != nil && *longPressTooltipActive {
						return true
					}
					activeTooltip := weakActiveTooltip.Value()
					if activeTooltip == nil {
						return false
					}
					return handleTooltipCheckVisibleAndUpdate(activeTooltip, tooltipIsHoverable, checkIsHoveredDuringPrepaint, tooltipBounds, window, cx)
				},
			},
			isHoverable: tooltipIsHoverable,
		})
	}
	didShow := newTooltip.IsSome()
	activeTooltip.set(newTooltip)
	window.Refresh()
	return didShow
}

type tooltipCheckVisibleAction uint8

const (
	tooltipCheckVisibleNone tooltipCheckVisibleAction = iota
	tooltipCheckVisibleHide
	tooltipCheckVisibleScheduleHide
	tooltipCheckVisibleCancelHide
)

func handleTooltipCheckVisibleAndUpdate(activeTooltip *activeTooltipCell, tooltipIsHoverable bool, checkIsHovered func(window *Window) bool, tooltipBounds Bounds[Pixels], window *Window, cx *App) bool {
	isHovered := checkIsHovered(window) || (tooltipIsHoverable && tooltipBounds.Contains(window.MousePosition()))
	action := tooltipCheckVisibleNone
	var tooltip AnyTooltip
	if current, ok := activeTooltip.value.Get(); ok {
		switch current.kind {
		case activeTooltipVisible:
			if !isHovered {
				if tooltipIsHoverable {
					action = tooltipCheckVisibleScheduleHide
					tooltip = cloneAnyTooltip(current.tooltip)
				} else {
					action = tooltipCheckVisibleHide
				}
			}
		case activeTooltipWaitingForHide:
			if isHovered {
				action = tooltipCheckVisibleCancelHide
				tooltip = cloneAnyTooltip(current.tooltip)
			}
		}
	}

	switch action {
	case tooltipCheckVisibleNone:
	case tooltipCheckVisibleHide:
		clearActiveTooltip(activeTooltip, window)
	case tooltipCheckVisibleScheduleHide:
		weakActiveTooltip := weak.Make(activeTooltip)
		delayedHideTask := windowSpawn(window, cx, func(cx *AsyncWindowContext) {
			_, _ = cx.BackgroundExecutor().Timer(hoverableTooltipHideDelay).Await(cx)
			activeTooltip := weakActiveTooltip.Value()
			if activeTooltip == nil {
				return
			}
			if taken, ok := activeTooltip.take().Get(); ok {
				taken.drop()
				_ = cx.Update(func(window *Window, _ *App) { window.Refresh() })
			}
		})
		activeTooltip.replace(activeTooltipState{kind: activeTooltipWaitingForHide, tooltip: tooltip, task: delayedHideTask})
	case tooltipCheckVisibleCancelHide:
		activeTooltip.replace(activeTooltipState{kind: activeTooltipVisible, tooltip: tooltip, isHoverable: true})
	}

	return activeTooltip.value.IsSome()
}

type groupHitboxes struct {
	stacks map[SharedString][]HitboxID
}

// GroupHitboxesGet returns the innermost hitbox of the named group being painted.
func GroupHitboxesGet(name SharedString, cx *App) (HitboxID, bool) {
	g := cx.DefaultGlobal[groupHitboxes]()
	stack := g.stacks[name]
	if len(stack) == 0 {
		return 0, false
	}
	return stack[len(stack)-1], true
}

func groupHitboxesPush(name SharedString, hitboxID HitboxID, cx *App) {
	g := cx.DefaultGlobal[groupHitboxes]()
	if g.stacks == nil {
		g.stacks = make(map[SharedString][]HitboxID)
	}
	g.stacks[name] = append(g.stacks[name], hitboxID)
}

func groupHitboxesPop(name SharedString, cx *App) {
	g := cx.DefaultGlobal[groupHitboxes]()
	stack, ok := g.stacks[name]
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	if len(stack) > 0 {
		g.stacks[name] = stack[:len(stack)-1]
	}
}

// Style returns the inner element's style refinement; E must implement Styler.
func (s *Stateful[E]) Style() *StyleRefinement {
	return any(s.element).(Styler).Style()
}

// Interactivity returns the inner element's interactivity; E must implement InteractiveElement.
func (s *Stateful[E]) Interactivity() *Interactivity {
	return any(s.element).(InteractiveElement).Interactivity()
}

// Extend adds children to the inner element; E must implement ParentElement.
func (s *Stateful[E]) Extend(elements []AnyElement) {
	any(s.element).(ParentElement).Extend(elements)
}

// IntoAnyElement returns the inner element's AnyElement; E must implement IntoElement (D-COND).
func (s *Stateful[E]) IntoAnyElement() AnyElement {
	return any(s.element).(IntoElement).IntoAnyElement()
}

// Stateful wraps an element that has an id; it is IntoElement only and delegates to its element (D-COND).
type Stateful[E any] struct {
	Styled[*Stateful[E]]
	Interactive[*Stateful[E]]
	StatefulInteractive[*Stateful[E]]
	Parent[*Stateful[E]]
	Fluent[*Stateful[E]]
	element E
}

// ScrollAnchor is an element that can be scrolled to in its parent element. Unlike
// ScrollHandle.ScrollToItem, an anchored element does not have to be an immediate child of the
// parent.
type ScrollAnchor struct {
	handle     ScrollHandle
	lastOrigin *Point[Pixels]
}

// ScrollAnchorForHandle creates a ScrollAnchor associated with handle.
func ScrollAnchorForHandle(handle ScrollHandle) ScrollAnchor {
	return ScrollAnchor{handle: handle, lastOrigin: &Point[Pixels]{}}
}

// ScrollTo requests a scroll to this item on the next frame.
func (a ScrollAnchor) ScrollTo(window *Window, _ *App) {
	this := a
	window.OnNextFrame(func(_ *Window, _ *App) {
		viewportBounds := this.handle.Bounds()
		selfBounds := *this.lastOrigin
		this.handle.SetOffset(viewportBounds.Origin.Sub(selfBounds))
	})
}

type scrollHandleState struct {
	offset         *Point[Pixels]
	ongoingScroll  *OngoingScroll
	bounds         Bounds[Pixels]
	maxOffset      Point[Pixels]
	childBounds    []Bounds[Pixels]
	scrollToBottom bool
	overflow       PointOf[Overflow]
	activeItem     Option[scrollActiveItem]
}

type scrollActiveItem struct {
	index    int
	strategy scrollStrategy
}

type scrollStrategy uint8

const (
	scrollStrategyFirstVisible scrollStrategy = iota
	scrollStrategyTop
)

// ScrollHandle is a handle to the scrollable aspects of an element, used for reading scroll state
// such as the current offset and for changing it, such as scrolling to a child. Copies share the
// same state; the zero value is not usable, create one with NewScrollHandle.
type ScrollHandle struct {
	state *scrollHandleState
}

// NewScrollHandle constructs a new scroll handle.
func NewScrollHandle() ScrollHandle {
	return ScrollHandle{state: &scrollHandleState{offset: &Point[Pixels]{}, ongoingScroll: &OngoingScroll{}}}
}

// Offset returns the current scroll offset.
func (h ScrollHandle) Offset() Point[Pixels] {
	return *h.state.offset
}

// MaxOffset returns the maximum scroll offset.
func (h ScrollHandle) MaxOffset() Point[Pixels] {
	return h.state.maxOffset
}

func (h ScrollHandle) itemAt(position Pixels) int {
	state := h.state
	lo, hi := 0, len(state.childBounds)
	for lo < hi {
		mid := lo + (hi-lo)/2
		bounds := state.childBounds[mid]
		switch {
		case position < bounds.Top():
			hi = mid
		case position > bounds.Bottom():
			lo = mid + 1
		default:
			return mid
		}
	}
	return min(lo, max(len(state.childBounds)-1, 0))
}

// TopItem returns the top child that's scrolled into view.
func (h ScrollHandle) TopItem() int {
	state := h.state
	top := state.bounds.Top() - state.offset.Y
	return h.itemAt(top)
}

// BottomItem returns the bottom child that's scrolled into view.
func (h ScrollHandle) BottomItem() int {
	state := h.state
	bottom := state.bounds.Bottom() - state.offset.Y
	return h.itemAt(bottom)
}

// Bounds returns the bounds into which this child is painted.
func (h ScrollHandle) Bounds() Bounds[Pixels] {
	return h.state.bounds
}

// BoundsForItem returns the bounds of child ix.
func (h ScrollHandle) BoundsForItem(ix int) (Bounds[Pixels], bool) {
	if ix < 0 || ix >= len(h.state.childBounds) {
		return Bounds[Pixels]{}, false
	}
	return h.state.childBounds[ix], true
}

// ScrollToItem makes child ix the active item to scroll to during prepaint.
func (h ScrollHandle) ScrollToItem(ix int) {
	h.state.activeItem = Some(scrollActiveItem{index: ix, strategy: scrollStrategyFirstVisible})
}

// ScrollToTopOfItem makes child ix the active item to scroll to during prepaint, scrolling the
// minimal amount to make it the first visible element.
func (h ScrollHandle) ScrollToTopOfItem(ix int) {
	h.state.activeItem = Some(scrollActiveItem{index: ix, strategy: scrollStrategyTop})
}

func (h ScrollHandle) scrollToActiveItem() {
	state := h.state

	activeItem, ok := state.activeItem.Get()
	if !ok {
		return
	}

	if activeItem.index < 0 || activeItem.index >= len(state.childBounds) {
		state.activeItem = Some(activeItem)
		return
	}
	bounds := state.childBounds[activeItem.index]
	scrollOffset := state.offset

	switch activeItem.strategy {
	case scrollStrategyFirstVisible:
		if state.overflow.Y == OverflowScroll {
			childHeight := bounds.Size.Height
			viewportHeight := state.bounds.Size.Height
			if childHeight > viewportHeight {
				scrollOffset.Y = state.bounds.Top() - bounds.Top()
			} else if bounds.Top()+scrollOffset.Y < state.bounds.Top() {
				scrollOffset.Y = state.bounds.Top() - bounds.Top()
			} else if bounds.Bottom()+scrollOffset.Y > state.bounds.Bottom() {
				scrollOffset.Y = state.bounds.Bottom() - bounds.Bottom()
			}
		}
	case scrollStrategyTop:
		scrollOffset.Y = state.bounds.Top() - bounds.Top()
	}

	if state.overflow.X == OverflowScroll {
		childWidth := bounds.Size.Width
		viewportWidth := state.bounds.Size.Width
		if childWidth > viewportWidth {
			scrollOffset.X = state.bounds.Left() - bounds.Left()
		} else if bounds.Left()+scrollOffset.X < state.bounds.Left() {
			scrollOffset.X = state.bounds.Left() - bounds.Left()
		} else if bounds.Right()+scrollOffset.X > state.bounds.Right() {
			scrollOffset.X = state.bounds.Right() - bounds.Right()
		}
	}
	state.activeItem = None[scrollActiveItem]()
}

// ScrollToBottom scrolls to the bottom.
func (h ScrollHandle) ScrollToBottom() {
	h.state.scrollToBottom = true
}

// SetOffset sets the offset explicitly. The offset is the distance from the top left of the
// parent container to the top left of the first child; it becomes more negative as you scroll
// further down.
func (h ScrollHandle) SetOffset(position Point[Pixels]) {
	*h.state.offset = position
}

// LogicalScrollTop returns the logical scroll top, as a child index and a pixel offset.
func (h ScrollHandle) LogicalScrollTop() (int, Pixels) {
	ix := h.TopItem()
	state := h.state

	if ix < len(state.childBounds) {
		childBounds := state.childBounds[ix]
		return ix, childBounds.Top() + state.offset.Y - state.bounds.Top()
	}
	return ix, Px(0.)
}

// LogicalScrollBottom returns the logical scroll bottom, as a child index and a pixel offset.
func (h ScrollHandle) LogicalScrollBottom() (int, Pixels) {
	ix := h.BottomItem()
	state := h.state

	if ix < len(state.childBounds) {
		childBounds := state.childBounds[ix]
		return ix, childBounds.Bottom() + state.offset.Y - state.bounds.Bottom()
	}
	return ix, Px(0.)
}

// ChildrenCount returns the count of children for the scrollable item.
func (h ScrollHandle) ChildrenCount() int {
	return len(h.state.childBounds)
}
