package gpui

import (
	"fmt"
	"runtime"
	"slices"
	"testing"
	"time"
	"weak"

	"github.com/megascan/gpui-go/a11y"
)

type divTestCanvas struct {
	prepaint func(bounds Bounds[Pixels], window *Window, cx *App)
	paint    func(bounds Bounds[Pixels], window *Window, cx *App)
}

func (c *divTestCanvas) ElementID() (ElementID, bool) { return ElementID{}, false }

func (c *divTestCanvas) SourceLocation() *SourceLocation { return nil }

func (c *divTestCanvas) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	return window.RequestLayout(DefaultStyle(), nil, cx), struct{}{}
}

func (c *divTestCanvas) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) struct{} {
	c.prepaint(bounds, window, cx)
	return struct{}{}
}

func (c *divTestCanvas) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	c.paint(bounds, window, cx)
}

func (c *divTestCanvas) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](c)
}

func divTestUpdateWindow(cx *TestAppContext, window AnyWindowHandle, f func(window *Window, cx *App)) {
	_, err := AppContextUpdateWindow(cx, window, func(_ AnyView, w *Window, cx *App) struct{} {
		f(w, cx)
		return struct{}{}
	})
	if err != nil {
		panic(err)
	}
}

func divTestClearTabStops(window *Window) {
	window.renderedFrame.tabStops.clear()
	window.nextFrame.tabStops.clear()
}

func divTestTakeRoot(window *Window) {
	if root, ok := window.root.Take().Get(); ok {
		root.Release()
	}
	window.RemoveWindow()
}

func divTestReleaseRoot(cx *TestAppContext, window AnyWindowHandle) {
	for _, step := range []func(*Window){divTestClearTabStops, divTestTakeRoot} {
		_, _ = AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, _ *App) struct{} {
			step(window)
			return struct{}{}
		})
	}
}

func divTestClose(cx *TestAppContext, window AnyWindowHandle) {
	divTestReleaseRoot(cx, window)
	cx.Close()
}

func divTestRemoveWindow(cx *TestAppContext, window AnyWindowHandle) {
	divTestUpdateWindow(cx, window, func(window *Window, _ *App) { divTestClearTabStops(window) })
	divTestUpdateWindow(cx, window, func(window *Window, _ *App) { divTestTakeRoot(window) })
}

func divTestDraw(cx *TestAppContext, window AnyWindowHandle) {
	divTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.Draw(cx).Clear(cx) })
}

func divTestStyled(style *StyleRefinement) *Styled[*StyleRefinement] {
	var styled Styled[*StyleRefinement]
	BindStyled(&styled, style)
	return &styled
}

type groupHoverTestView struct {
	renderCount         *int
	anonymousPaintCount *int
	statefulWidth       *Pixels
}

func (v *groupHoverTestView) Render(_ *Window, _ *Context[groupHoverTestView]) IntoElement {
	*v.renderCount++
	anonymousPaintCount := v.anonymousPaintCount
	statefulWidth := v.statefulWidth
	return NewDiv().SizeFull().Child(
		NewDiv().
			Ml(Px(20.)).
			Mt(Px(20.)).
			Size(Px(50.)).
			Relative().
			Group("hover-group").
			Child(
				NewDiv().
					Absolute().
					SizeFull().
					Invisible().
					GroupHover("hover-group", func(style *StyleRefinement) *StyleRefinement { return divTestStyled(style).Visible() }).
					Child(&divTestCanvas{
						prepaint: func(Bounds[Pixels], *Window, *App) {},
						paint:    func(Bounds[Pixels], *Window, *App) { *anonymousPaintCount++ },
					}),
			).
			Child(
				NewDiv().
					ID("stateful-group-hover-target").
					Absolute().
					Top0().
					Left0().
					Size(Px(10.)).
					GroupHover("hover-group", func(style *StyleRefinement) *StyleRefinement { return divTestStyled(style).Size(Px(20.)) }).
					Child(&divTestCanvas{
						prepaint: func(bounds Bounds[Pixels], _ *Window, _ *App) { *statefulWidth = bounds.Size.Width },
						paint:    func(Bounds[Pixels], *Window, *App) {},
					}),
			),
	)
}

func TestDiv_GroupHoverStylesUpdateOnlyOnTransitions(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		renderCount := 0
		anonymousPaintCount := 0
		statefulWidth := Px(0.)
		window := cx.AddWindow(func(_ *Window, _ *Context[groupHoverTestView]) *groupHoverTestView {
			return &groupHoverTestView{renderCount: &renderCount, anonymousPaintCount: &anonymousPaintCount, statefulWidth: &statefulWidth}
		}).AnyWindowHandle
		defer divTestReleaseRoot(cx, window)

		divTestDraw(cx, window)
		if anonymousPaintCount != 0 || statefulWidth != Px(10.) {
			t.Fatalf("paint count %d, width %v", anonymousPaintCount, statefulWidth)
		}

		moveMouse := func(position Point[Pixels]) {
			divTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.SimulateMouseMove(position, cx) })
		}
		check := func(step string, renders, paints int, width Pixels) {
			t.Helper()
			if renderCount != renders || anonymousPaintCount != paints || statefulWidth != width {
				t.Fatalf("%s: render %d (want %d), paint %d (want %d), width %v (want %v)", step, renderCount, renders, anonymousPaintCount, paints, statefulWidth, width)
			}
		}

		initialRenderCount := renderCount
		moveMouse(NewPoint(Px(25.), Px(25.)))
		check("enter", initialRenderCount+1, 1, Px(20.))

		moveMouse(NewPoint(Px(30.), Px(30.)))
		check("move inside", initialRenderCount+1, 1, Px(20.))

		moveMouse(NewPoint(Px(5.), Px(5.)))
		check("leave", initialRenderCount+2, 1, Px(10.))

		moveMouse(NewPoint(Px(10.), Px(10.)))
		check("move outside", initialRenderCount+2, 1, Px(10.))
	})
}

type hoverListenerLayoutTestView struct {
	targetLeft       Pixels
	hoverTransitions *[]bool
}

func (v *hoverListenerLayoutTestView) Render(_ *Window, _ *Context[hoverListenerLayoutTestView]) IntoElement {
	hoverTransitions := v.hoverTransitions
	return NewDiv().Relative().SizeFull().Child(
		NewDiv().
			ID("hover-target").
			Absolute().
			Left(v.targetLeft).
			Top0().
			Size(Px(20.)).
			OnClick(func(*ClickEvent, *Window, *App) {}).
			OnHover(func(isHovered bool, _ *Window, _ *App) {
				*hoverTransitions = append(*hoverTransitions, isHovered)
			}),
	)
}

func divTestExpectTransitions(t *testing.T, got []bool, want ...bool) {
	t.Helper()
	if !slices.Equal(got, want) && !(len(got) == 0 && len(want) == 0) {
		t.Fatalf("hover transitions = %v, want %v", got, want)
	}
}

func TestDiv_DefaultHoverListenerUpdatesWhenLayoutChangesUnderStationaryMouse(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var hoverTransitions []bool
		window := cx.AddWindow(func(_ *Window, _ *Context[hoverListenerLayoutTestView]) *hoverListenerLayoutTestView {
			return &hoverListenerLayoutTestView{targetLeft: Px(40.), hoverTransitions: &hoverTransitions}
		})
		defer divTestReleaseRoot(cx, window.AnyWindowHandle)
		anyWindow := window.AnyWindowHandle

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.SimulateMouseMove(NewPoint(Px(10.), Px(10.)), cx)
		})
		divTestExpectTransitions(t, hoverTransitions)

		_ = window.Update(cx, func(view *hoverListenerLayoutTestView, _ *Window, cx *Context[hoverListenerLayoutTestView]) {
			view.targetLeft = Px(0.)
			cx.Notify()
		})
		divTestDraw(cx, anyWindow)
		divTestExpectTransitions(t, hoverTransitions, true)

		_ = window.Update(cx, func(view *hoverListenerLayoutTestView, _ *Window, cx *Context[hoverListenerLayoutTestView]) {
			view.targetLeft = Px(40.)
			cx.Notify()
		})
		divTestDraw(cx, anyWindow)
		divTestExpectTransitions(t, hoverTransitions, true, false)
	})
}

func TestDiv_DefaultHoverListenerEndsAfterKeyPress(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var hoverTransitions []bool
		window := cx.AddWindow(func(_ *Window, _ *Context[hoverListenerLayoutTestView]) *hoverListenerLayoutTestView {
			return &hoverListenerLayoutTestView{targetLeft: Px(0.), hoverTransitions: &hoverTransitions}
		})
		defer divTestReleaseRoot(cx, window.AnyWindowHandle)
		anyWindow := window.AnyWindowHandle

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.SimulateMouseMove(NewPoint(Px(10.), Px(10.)), cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true)

		divTestKeyDown(cx, anyWindow, "a")
		divTestDraw(cx, anyWindow)
		divTestExpectTransitions(t, hoverTransitions, true, false)
	})
}

type hoverListenerModeLayoutTestView struct {
	targetLeft       Pixels
	hoverTransitions *[]bool
}

func (v *hoverListenerModeLayoutTestView) Render(_ *Window, _ *Context[hoverListenerModeLayoutTestView]) IntoElement {
	hoverTransitions := v.hoverTransitions
	return NewDiv().Relative().SizeFull().Child(
		NewDiv().
			ID("hover-target").
			Absolute().
			Left(v.targetLeft).
			Top0().
			Size(Px(20.)).
			HoverListenerMode(HoverListenerModeInputModalityIndependent).
			OnHover(func(isHovered bool, _ *Window, _ *App) {
				*hoverTransitions = append(*hoverTransitions, isHovered)
			}),
	)
}

func TestDiv_InputModalityIndependentHoverListenerUpdatesAfterKeyPress(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var hoverTransitions []bool
		window := cx.AddWindow(func(_ *Window, _ *Context[hoverListenerModeLayoutTestView]) *hoverListenerModeLayoutTestView {
			return &hoverListenerModeLayoutTestView{targetLeft: Px(40.), hoverTransitions: &hoverTransitions}
		})
		defer divTestReleaseRoot(cx, window.AnyWindowHandle)
		anyWindow := window.AnyWindowHandle
		pointerPosition := NewPoint(Px(10.), Px(10.))

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.SimulateMouseMove(pointerPosition, cx)
		})
		divTestExpectTransitions(t, hoverTransitions)

		divTestKeyDown(cx, anyWindow, "a")
		_ = window.Update(cx, func(view *hoverListenerModeLayoutTestView, _ *Window, cx *Context[hoverListenerModeLayoutTestView]) {
			view.targetLeft = Px(0.)
			cx.Notify()
		})
		divTestDraw(cx, anyWindow)
		divTestExpectTransitions(t, hoverTransitions, true)

		divTestKeyDown(cx, anyWindow, "b")
		divTestDraw(cx, anyWindow)
		divTestExpectTransitions(t, hoverTransitions, true)

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.SimulateMouseMove(NewPoint(Px(30.), Px(10.)), cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true, false)

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.SimulateMouseMove(pointerPosition, cx)
			window.DispatchEvent(MouseExitEvent{Position: pointerPosition}.ToPlatformInput(), cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true, false, true, false)
	})
}

func TestDiv_DefaultHoverListenerRemainsHoveredDuringStationaryMousePress(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		var hoverTransitions []bool
		window := cx.AddWindow(func(_ *Window, _ *Context[hoverListenerLayoutTestView]) *hoverListenerLayoutTestView {
			return &hoverListenerLayoutTestView{targetLeft: Px(0.), hoverTransitions: &hoverTransitions}
		})
		defer divTestReleaseRoot(cx, window.AnyWindowHandle)
		anyWindow := window.AnyWindowHandle
		mousePosition := NewPoint(Px(10.), Px(10.))

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			window.SimulateMouseMove(mousePosition, cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true)

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.DispatchEvent(MouseDownEvent{Position: mousePosition, Button: MouseButtonLeft, ClickCount: 1}.ToPlatformInput(), cx)
			window.Draw(cx).Clear(cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true)

		divTestUpdateWindow(cx, anyWindow, func(window *Window, cx *App) {
			window.DispatchEvent(MouseUpEvent{Position: mousePosition, Button: MouseButtonLeft, ClickCount: 1}.ToPlatformInput(), cx)
			window.Draw(cx).Clear(cx)
		})
		divTestExpectTransitions(t, hoverTransitions, true)
	})
}

type testTooltipView struct{}

func (v *testTooltipView) Render(_ *Window, _ *Context[testTooltipView]) IntoElement {
	return NewDiv().W(Px(20.)).H(Px(20.)).ChildText("tooltip")
}

type capturedActiveTooltip struct {
	value Option[weak.Pointer[activeTooltipCell]]
}

func (c *capturedActiveTooltip) upgrade(t *testing.T) *activeTooltipCell {
	t.Helper()
	cell := c.value.Unwrap().Value()
	if cell == nil {
		t.Fatalf("active tooltip was released")
	}
	return cell
}

type tooltipCaptureElement struct {
	child                 AnyElement
	capturedActiveTooltip *capturedActiveTooltip
}

func (e *tooltipCaptureElement) ElementID() (ElementID, bool) { return ElementID{}, false }

func (e *tooltipCaptureElement) SourceLocation() *SourceLocation { return nil }

func (e *tooltipCaptureElement) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	return e.child.RequestLayout(window, cx), struct{}{}
}

func (e *tooltipCaptureElement) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, window *Window, cx *App) struct{} {
	e.child.Prepaint(window, cx)
	return struct{}{}
}

func (e *tooltipCaptureElement) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *struct{}, _ *struct{}, window *Window, cx *App) {
	e.child.Paint(window, cx)
	window.WithGlobalID(ElementIDName("target"), func(globalID *GlobalElementID, window *Window) struct{} {
		return window.WithElementState(globalID, func(state Option[InteractiveElementState], _ *Window) (struct{}, InteractiveElementState) {
			s := state.Unwrap()
			if s.activeTooltip != nil {
				e.capturedActiveTooltip.value = Some(weak.Make(s.activeTooltip))
			} else {
				e.capturedActiveTooltip.value = None[weak.Pointer[activeTooltipCell]]()
			}
			return struct{}{}, s
		})
	})
}

func (e *tooltipCaptureElement) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](e)
}

type tooltipOwner struct {
	capturedActiveTooltip *capturedActiveTooltip
	showDelayOverride     Option[time.Duration]
}

func (o *tooltipOwner) Render(_ *Window, _ *Context[tooltipOwner]) IntoElement {
	return &tooltipCaptureElement{
		child: NewDiv().
			SizeFull().
			Child(
				NewDiv().
					ID("target").
					W(Px(50.)).
					H(Px(50.)).
					Tooltip(func(_ *Window, cx *App) AnyView {
						return AnyViewFromEntity(cx.New(func(*Context[testTooltipView]) *testTooltipView { return &testTooltipView{} }))
					}).
					WhenSome(o.showDelayOverride, func(this *Stateful[*Div], delay time.Duration) *Stateful[*Div] {
						return this.TooltipShowDelay(delay)
					}),
			).
			IntoAnyElement(),
		capturedActiveTooltip: o.capturedActiveTooltip,
	}
}

func divTestActiveTooltipKind(cell *activeTooltipCell) (activeTooltipKind, bool) {
	value, ok := cell.value.Get()
	return value.kind, ok
}

func TestDiv_ScrollHandleAlignsWideChildrenToLeftEdge(t *testing.T) {
	handle := NewScrollHandle()
	state := handle.state
	state.bounds = NewBounds(NewPoint(Px(0.), Px(0.)), Size[Pixels]{Width: Px(80.), Height: Px(20.)})
	state.childBounds = []Bounds[Pixels]{NewBounds(NewPoint(Px(25.), Px(0.)), Size[Pixels]{Width: Px(200.), Height: Px(20.)})}
	state.overflow.X = OverflowScroll
	state.activeItem = Some(scrollActiveItem{index: 0, strategy: scrollStrategyFirstVisible})

	handle.scrollToActiveItem()

	if handle.Offset().X != Px(-25.) {
		t.Fatalf("offset x = %v, want -25", handle.Offset().X)
	}
}

func TestDiv_ScrollHandleAlignsTallChildrenToTopEdge(t *testing.T) {
	handle := NewScrollHandle()
	state := handle.state
	state.bounds = NewBounds(NewPoint(Px(0.), Px(0.)), Size[Pixels]{Width: Px(20.), Height: Px(80.)})
	state.childBounds = []Bounds[Pixels]{NewBounds(NewPoint(Px(0.), Px(25.)), Size[Pixels]{Width: Px(20.), Height: Px(200.)})}
	state.overflow.Y = OverflowScroll
	state.activeItem = Some(scrollActiveItem{index: 0, strategy: scrollStrategyFirstVisible})

	handle.scrollToActiveItem()

	if handle.Offset().Y != Px(-25.) {
		t.Fatalf("offset y = %v, want -25", handle.Offset().Y)
	}
}

func TestDiv_LongPressShowsTooltipUntilTouchEnds(t *testing.T) {
	t.Skip("blocked:M0-text-elements: the visible tooltip renders text through SharedString.IntoAnyElement")
	testApp, anyWindow, captured := createTooltipOwnerTest(None[time.Duration]())
	defer divTestClose(testApp, anyWindow)
	activeTooltip := captured.upgrade(t)
	touchID := TouchID(1)
	touchPosition := NewPoint(Px(49.), Px(10.))

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.DispatchEvent(TouchEvent{ID: touchID, Phase: TouchPhaseStarted, Position: touchPosition}.ToPlatformInput(), cx)
	})
	testApp.dispatcher.AdvanceClock(DefaultGestureTuning().LongPressDuration + time.Millisecond)
	testApp.RunUntilParked()

	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipVisible {
		t.Fatalf("tooltip is not visible after the long press")
	}

	movedPosition := NewPoint(Px(55.), Px(10.))
	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.DispatchEvent(TouchEvent{ID: touchID, Phase: TouchPhaseMoved, Position: movedPosition}.ToPlatformInput(), cx)
		window.Draw(cx).Clear(cx)
	})
	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipVisible {
		t.Fatalf("tooltip is not visible after the touch moved")
	}

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.DispatchEvent(TouchEvent{ID: touchID, Phase: TouchPhaseEnded, Position: movedPosition}.ToPlatformInput(), cx)
	})

	if activeTooltip.value.IsSome() {
		t.Fatalf("tooltip is still active after the touch ended")
	}
}

func setupTooltipOwnerTest(showDelayOverride Option[time.Duration]) (*TestAppContext, AnyWindowHandle, *capturedActiveTooltip) {
	testApp, anyWindow, captured := createTooltipOwnerTest(showDelayOverride)

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.DispatchEvent(MouseMoveEvent{Position: NewPoint(Px(10.), Px(10.))}.ToPlatformInput(), cx)
	})

	divTestDraw(testApp, anyWindow)

	return testApp, anyWindow, captured
}

func createTooltipOwnerTest(showDelayOverride Option[time.Duration]) (*TestAppContext, AnyWindowHandle, *capturedActiveTooltip) {
	testApp := TestAppContextSingle()
	captured := &capturedActiveTooltip{}
	window := testApp.AddWindow(func(_ *Window, _ *Context[tooltipOwner]) *tooltipOwner {
		return &tooltipOwner{capturedActiveTooltip: captured, showDelayOverride: showDelayOverride}
	})
	anyWindow := window.AnyWindowHandle

	divTestDraw(testApp, anyWindow)

	return testApp, anyWindow, captured
}

func divTestCollectedEventually(pointer weak.Pointer[activeTooltipCell]) bool {
	for range 8 {
		runtime.GC()
		if pointer.Value() == nil {
			return true
		}
	}
	return false
}

func TestDiv_TooltipWaitingForShowIsReleasedWhenItsOwnerDisappears(t *testing.T) {
	testApp, anyWindow, captured := setupTooltipOwnerTest(None[time.Duration]())
	defer divTestClose(testApp, anyWindow)

	weakActiveTooltip := captured.value.Unwrap()
	activeTooltip := captured.upgrade(t)
	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipWaitingForShow {
		t.Fatalf("tooltip is not waiting for show")
	}

	divTestRemoveWindow(testApp, anyWindow)
	testApp.RunUntilParked()
	activeTooltip = nil
	captured.value = None[weak.Pointer[activeTooltipCell]]()

	if !divTestCollectedEventually(weakActiveTooltip) {
		t.Fatalf("active tooltip was not released")
	}
}

func TestDiv_TooltipRespectsCustomShowDelay(t *testing.T) {
	t.Skip("blocked:M0-text-elements: the visible tooltip renders text through SharedString.IntoAnyElement")
	extraDelay := time.Second
	showDelayOverride := defaultTooltipShowDelay + extraDelay
	testApp, anyWindow, captured := setupTooltipOwnerTest(Some(showDelayOverride))
	defer divTestClose(testApp, anyWindow)

	activeTooltip := captured.upgrade(t)

	testApp.dispatcher.AdvanceClock(defaultTooltipShowDelay)
	testApp.RunUntilParked()

	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipWaitingForShow {
		t.Fatalf("tooltip is not waiting for show after the default delay")
	}

	testApp.dispatcher.AdvanceClock(extraDelay)
	testApp.RunUntilParked()

	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipVisible {
		t.Fatalf("tooltip is not visible after the custom delay")
	}
}

func TestDiv_TooltipIsReleasedWhenItsOwnerDisappears(t *testing.T) {
	t.Skip("blocked:M0-text-elements: the visible tooltip renders text through SharedString.IntoAnyElement")
	testApp, anyWindow, captured := setupTooltipOwnerTest(None[time.Duration]())
	defer divTestClose(testApp, anyWindow)

	weakActiveTooltip := captured.value.Unwrap()
	activeTooltip := captured.upgrade(t)

	testApp.dispatcher.AdvanceClock(defaultTooltipShowDelay)
	testApp.RunUntilParked()

	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipVisible {
		t.Fatalf("tooltip is not visible")
	}

	divTestRemoveWindow(testApp, anyWindow)
	testApp.RunUntilParked()
	activeTooltip = nil
	captured.value = None[weak.Pointer[activeTooltipCell]]()

	if !divTestCollectedEventually(weakActiveTooltip) {
		t.Fatalf("active tooltip was not released")
	}
}

func TestDiv_TooltipHidesAfterMouseLeavesOrigin(t *testing.T) {
	t.Skip("blocked:M0-text-elements: the visible tooltip renders text through SharedString.IntoAnyElement")
	testApp, anyWindow, captured := setupTooltipOwnerTest(None[time.Duration]())
	defer divTestClose(testApp, anyWindow)

	activeTooltip := captured.upgrade(t)

	testApp.dispatcher.AdvanceClock(defaultTooltipShowDelay)
	testApp.RunUntilParked()

	if kind, ok := divTestActiveTooltipKind(activeTooltip); !ok || kind != activeTooltipVisible {
		t.Fatalf("tooltip is not visible")
	}

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.DispatchEvent(MouseMoveEvent{Position: NewPoint(Px(75.), Px(75.))}.ToPlatformInput(), cx)
	})

	if activeTooltip.value.IsSome() {
		t.Fatalf("tooltip is still active after the mouse left")
	}
}

type mouseDownOutOwner struct {
	mouseDownOutCount *int
}

func (o *mouseDownOutOwner) Render(_ *Window, _ *Context[mouseDownOutOwner]) IntoElement {
	mouseDownOutCount := o.mouseDownOutCount
	return NewDiv().
		SizeFull().
		Child(NewDiv().ID("target").W(Px(50.)).H(Px(50.)).OnMouseDownOut(func(*MouseDownEvent, *Window, *App) {
			*mouseDownOutCount++
		}))
}

func TestDiv_MouseDownOutIsSuppressedWhileWindowPromptIsActive(t *testing.T) {
	t.Skip("blocked:M0-text-elements: the fallback prompt renders text through SharedString.IntoAnyElement")
	testApp := TestAppContextSingle()
	defer testApp.Close()
	mouseDownOutCount := 0
	window := testApp.AddWindow(func(_ *Window, _ *Context[mouseDownOutOwner]) *mouseDownOutOwner {
		return &mouseDownOutOwner{mouseDownOutCount: &mouseDownOutCount}
	})
	defer divTestReleaseRoot(testApp, window.AnyWindowHandle)
	anyWindow := window.AnyWindowHandle

	dispatchMouseDownOutsideTarget := func() {
		divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
			window.DispatchEvent(MouseDownEvent{Position: NewPoint(Px(75.), Px(75.)), Button: MouseButtonLeft, ClickCount: 1}.ToPlatformInput(), cx)
		})
	}

	divTestDraw(testApp, anyWindow)

	dispatchMouseDownOutsideTarget()
	if mouseDownOutCount != 1 {
		t.Fatalf("mouse down outside the element should fire mouse-down-out listeners: %d", mouseDownOutCount)
	}

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		cx.SetPromptBuilder(NewFallbackPromptRenderer)
		_ = window.Prompt(PromptLevelWarning, "message", None[string](), []PromptButton{PromptButtonFromStr("Ok")}, cx)
		if !window.HasActivePrompt() {
			t.Fatalf("no active prompt")
		}
		window.Draw(cx).Clear(cx)
	})

	dispatchMouseDownOutsideTarget()
	if mouseDownOutCount != 1 {
		t.Fatalf("mouse down over an active prompt should not fire mouse-down-out listeners: %d", mouseDownOutCount)
	}
}

func TestDiv_TouchTapOutFiresOnceAfterRelease(t *testing.T) {
	testApp := TestAppContextSingle()
	defer testApp.Close()
	mouseDownOutCount := 0
	window := testApp.AddWindow(func(_ *Window, _ *Context[mouseDownOutOwner]) *mouseDownOutOwner {
		return &mouseDownOutOwner{mouseDownOutCount: &mouseDownOutCount}
	})
	defer divTestReleaseRoot(testApp, window.AnyWindowHandle)
	anyWindow := window.AnyWindowHandle

	divTestUpdateWindow(testApp, anyWindow, func(window *Window, cx *App) {
		window.Draw(cx).Clear(cx)
		started := TouchEvent{ID: TouchID(1), Phase: TouchPhaseStarted, Position: NewPoint(Px(75.), Px(75.))}
		window.DispatchEvent(started.ToPlatformInput(), cx)
		if mouseDownOutCount != 0 {
			t.Fatalf("mouse down out fired on touch start")
		}
		ended := started
		ended.Phase = TouchPhaseEnded
		window.DispatchEvent(ended.ToPlatformInput(), cx)
	})

	if mouseDownOutCount != 1 {
		t.Fatalf("a touch tap outside the element should use mouse-down-out listeners: %d", mouseDownOutCount)
	}
}

func TestDiv_TouchPanAndCancelDoNotFireMouseDownOut(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		mouseDownOutCount := 0
		window := cx.AddWindow(func(_ *Window, _ *Context[mouseDownOutOwner]) *mouseDownOutOwner {
			return &mouseDownOutOwner{mouseDownOutCount: &mouseDownOutCount}
		})
		defer divTestReleaseRoot(cx, window.AnyWindowHandle)
		type sample struct {
			phase    TouchPhase
			vertical float32
		}
		divTestUpdateWindow(cx, window.AnyWindowHandle, func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			for _, touch := range []struct {
				id      TouchID
				samples [3]sample
			}{
				{TouchID(1), [3]sample{{TouchPhaseStarted, 100.}, {TouchPhaseMoved, 75.}, {TouchPhaseEnded, 75.}}},
				{TouchID(2), [3]sample{{TouchPhaseStarted, 100.}, {TouchPhaseMoved, 100.}, {TouchPhaseCancelled, 100.}}},
			} {
				for _, s := range touch.samples {
					window.DispatchEvent(TouchEvent{ID: touch.id, Phase: s.phase, Position: NewPoint(Px(75.), Px(s.vertical))}.ToPlatformInput(), cx)
				}
			}
		})
		if mouseDownOutCount != 0 {
			t.Fatalf("mouse down out count = %d, want 0", mouseDownOutCount)
		}
	})
}

func TestDiv_AccessibilityIDBuilderWritesAuthorID(t *testing.T) {
	element := NewDiv().ID("buffer-font-size").AccessibilityID("settings.buffer-font-size")
	node := a11y.NewNode(a11y.RoleSpinButton)
	element.Interactivity().writeA11yInfo(&node)

	if got, ok := node.AuthorID(); !ok || got != "settings.buffer-font-size" {
		t.Fatalf("author id = %q, %v", got, ok)
	}
}

func TestDiv_WriteA11yInfoStringAndNumericProperties(t *testing.T) {
	var interactivity Interactivity
	interactivity.aria.authorID = Some(SharedString("settings.buffer-font-size"))
	interactivity.aria.label = Some(SharedString("Buffer Font Size"))
	interactivity.aria.value = Some(SharedString("15"))
	interactivity.aria.placeholder = Some(SharedString("Search"))
	interactivity.aria.numericValue = Some(15.0)
	interactivity.aria.minNumericValue = Some(6.0)
	interactivity.aria.maxNumericValue = Some(72.0)
	interactivity.aria.numericValueStep = Some(1.0)

	node := a11y.NewNode(a11y.RoleSpinButton)
	interactivity.writeA11yInfo(&node)

	checkString := func(name string, got string, ok bool, want string) {
		t.Helper()
		if !ok || got != want {
			t.Fatalf("%s = %q, %v; want %q", name, got, ok, want)
		}
	}
	checkNumber := func(name string, got float64, ok bool, want float64) {
		t.Helper()
		if !ok || got != want {
			t.Fatalf("%s = %v, %v; want %v", name, got, ok, want)
		}
	}
	got, ok := node.AuthorID()
	checkString("author id", got, ok, "settings.buffer-font-size")
	got, ok = node.Label()
	checkString("label", got, ok, "Buffer Font Size")
	got, ok = node.Value()
	checkString("value", got, ok, "15")
	got, ok = node.Placeholder()
	checkString("placeholder", got, ok, "Search")
	n, ok := node.NumericValue()
	checkNumber("numeric value", n, ok, 15.0)
	n, ok = node.MinNumericValue()
	checkNumber("min numeric value", n, ok, 6.0)
	n, ok = node.MaxNumericValue()
	checkNumber("max numeric value", n, ok, 72.0)
	n, ok = node.NumericValueStep()
	checkNumber("numeric value step", n, ok, 1.0)
}

type keyboardActivationTest struct {
	focusA FocusHandle
	focusB FocusHandle
	clicks *[]string
}

func (v *keyboardActivationTest) Drop(*App) {
	v.focusA.Release()
	v.focusB.Release()
}

func (v *keyboardActivationTest) Render(_ *Window, _ *Context[keyboardActivationTest]) IntoElement {
	clicks := v.clicks
	return NewDiv().
		SizeFull().
		Child(
			NewDiv().
				ID("a").
				W(Px(50.)).
				H(Px(50.)).
				TrackFocus(v.focusA).
				OnClick(func(*ClickEvent, *Window, *App) { *clicks = append(*clicks, "a") }),
		).
		Child(
			NewDiv().
				ID("b").
				W(Px(50.)).
				H(Px(50.)).
				TrackFocus(v.focusB).
				OnClick(func(*ClickEvent, *Window, *App) { *clicks = append(*clicks, "b") }),
		)
}

func setupKeyboardActivationTest() (*TestAppContext, AnyWindowHandle, *[]string, FocusHandle, FocusHandle) {
	cx := TestAppContextSingle()
	var focusA, focusB FocusHandle
	cx.Update(func(cx *App) {
		focusA, focusB = cx.FocusHandle(), cx.FocusHandle()
	})
	clicks := &[]string{}
	window := cx.AddWindow(func(_ *Window, _ *Context[keyboardActivationTest]) *keyboardActivationTest {
		return &keyboardActivationTest{focusA: focusA.Clone(), focusB: focusB.Clone(), clicks: clicks}
	})
	return cx, window.AnyWindowHandle, clicks, focusA, focusB
}

func teardownKeyboardActivationTest(cx *TestAppContext, window AnyWindowHandle, focusA, focusB FocusHandle) {
	divTestReleaseRoot(cx, window)
	cx.Update(func(*App) {
		focusA.Release()
		focusB.Release()
	})
	cx.Close()
}

func focusAndDraw(cx *TestAppContext, window AnyWindowHandle, handle FocusHandle) {
	divTestUpdateWindow(cx, window, func(window *Window, cx *App) { window.Focus(handle, cx) })
	cx.RunUntilParked()
	divTestDraw(cx, window)
}

func divTestKeystroke(cx *TestAppContext, key string) Keystroke {
	os := cx.ReadR(func(cx *App) OperatingSystem { return cx.PlatformInfo().Os })
	keystroke, err := KeystrokeParseWithOs(key, os)
	if err != nil {
		panic(err)
	}
	return keystroke
}

func divTestKeyDown(cx *TestAppContext, window AnyWindowHandle, key string) {
	keystroke := divTestKeystroke(cx, key)
	divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
		window.DispatchEvent(KeyDownEvent{Keystroke: keystroke}.ToPlatformInput(), cx)
	})
}

func divTestKeyUp(cx *TestAppContext, window AnyWindowHandle, key string) {
	keystroke := divTestKeystroke(cx, key)
	divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
		window.DispatchEvent(KeyUpEvent{Keystroke: keystroke}.ToPlatformInput(), cx)
	})
}

func divTestExpectClicks(t *testing.T, clicks []string, want ...string) {
	t.Helper()
	if !slices.Equal(clicks, want) && !(len(clicks) == 0 && len(want) == 0) {
		t.Fatalf("clicks: %v, want %v", clicks, want)
	}
}

func TestDiv_KeyboardActivationFiresClickOnSameElement(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "enter")
	divTestKeyUp(cx, window, "enter")

	divTestExpectClicks(t, *clicks, "a")
}

func TestDiv_KeyboardActivationDoesNotLeakAcrossFocusChange(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "enter")

	focusAndDraw(cx, window, focusB)
	divTestKeyUp(cx, window, "enter")

	divTestExpectClicks(t, *clicks)
}

func TestDiv_KeyboardActivationDoesNotLeakWhenFocusReturns(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "enter")

	focusAndDraw(cx, window, focusB)
	focusAndDraw(cx, window, focusA)
	divTestKeyUp(cx, window, "enter")

	divTestExpectClicks(t, *clicks)
}

func TestDiv_KeyboardActivationClearedByInterveningKeyRelease(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "escape")
	divTestKeyDown(cx, window, "space")
	divTestKeyUp(cx, window, "escape")
	divTestKeyUp(cx, window, "space")

	divTestExpectClicks(t, *clicks)
}

func TestDiv_KeyboardActivationDoesNotDistinguishSpaceAndEnter(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "space")
	divTestKeyUp(cx, window, "enter")

	divTestExpectClicks(t, *clicks, "a")
}

func TestDiv_KeyboardActivationClearedByInterveningKeydown(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "enter")
	divTestKeyDown(cx, window, "a")
	divTestKeyUp(cx, window, "enter")

	divTestExpectClicks(t, *clicks)
}

func TestDiv_KeyboardActivationIgnoresModifiedKeys(t *testing.T) {
	cx, window, clicks, focusA, focusB := setupKeyboardActivationTest()
	defer teardownKeyboardActivationTest(cx, window, focusA, focusB)

	focusAndDraw(cx, window, focusA)
	divTestKeyDown(cx, window, "cmd-enter")
	divTestKeyUp(cx, window, "cmd-enter")

	divTestExpectClicks(t, *clicks)
}

type tabGroupFocus struct {
	groupA FocusHandle
	itemA  FocusHandle
	groupB FocusHandle
	itemB  FocusHandle
}

func (v *tabGroupFocus) Drop(*App) {
	v.groupA.Release()
	v.itemA.Release()
	v.groupB.Release()
	v.itemB.Release()
}

func (v *tabGroupFocus) Render(_ *Window, _ *Context[tabGroupFocus]) IntoElement {
	group := func(container, item FocusHandle) *Div {
		return NewDiv().
			TrackFocus(container).
			TabGroup().
			Child(NewDiv().TrackFocus(item))
	}
	return NewDiv().
		Child(group(v.groupA, v.itemA)).
		Child(group(v.groupB, v.itemB))
}

func TestDiv_FocusNextFromTabGroupContainerEntersThatGroup(t *testing.T) {
	cx := TestAppContextSingle()
	defer cx.Close()
	var groupA, itemA, groupB, itemB FocusHandle
	cx.Update(func(cx *App) {
		groupA, itemA, groupB, itemB = cx.FocusHandle(), cx.FocusHandle().WithTabStop(true), cx.FocusHandle(), cx.FocusHandle().WithTabStop(true)
	})
	defer cx.Update(func(*App) {
		groupA.Release()
		itemA.Release()
		groupB.Release()
		itemB.Release()
	})
	window := cx.AddWindow(func(_ *Window, _ *Context[tabGroupFocus]) *tabGroupFocus {
		return &tabGroupFocus{groupA: groupA.Clone(), itemA: itemA.Clone(), groupB: groupB.Clone(), itemB: itemB.Clone()}
	}).AnyWindowHandle
	defer divTestReleaseRoot(cx, window)
	divTestDraw(cx, window)

	focused := None[FocusID]()
	divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
		window.Focus(groupB, cx)
		window.FocusNext(cx)
		if handle, ok := window.Focused(cx); ok {
			focused = Some(handle.id)
			handle.Release()
		}
	})

	if focused != Some(itemB.id) {
		t.Fatalf("focused = %v, want %v", focused, itemB.id)
	}
}

type paddedContainer struct {
	scrollHandle ScrollHandle
}

func (v *paddedContainer) Render(_ *Window, _ *Context[paddedContainer]) IntoElement {
	return NewDiv().SizeFull().Child(
		NewDiv().
			ID("container").
			H(Px(50.)).
			W(Px(100.)).
			Py(Px(4.25)).
			OverflowYScroll().
			TrackScroll(v.scrollHandle).
			Child(NewDiv().WFull().H(Px(42.))),
	)
}

func TestDiv_FractionalPaddingDoesNotMakeAFittingContainerScrollable(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		scrollHandle := NewScrollHandle()
		window := cx.AddWindow(func(_ *Window, _ *Context[paddedContainer]) *paddedContainer {
			return &paddedContainer{scrollHandle: scrollHandle}
		}).AnyWindowHandle
		defer divTestReleaseRoot(cx, window)
		divTestDraw(cx, window)

		if scrollHandle.MaxOffset().Y != Px(0.) {
			t.Fatalf("max offset y = %v, want 0", scrollHandle.MaxOffset().Y)
		}
	})
}

type scrollStrip struct {
	scrollHandle ScrollHandle
}

func (v *scrollStrip) Render(_ *Window, _ *Context[scrollStrip]) IntoElement {
	children := make([]IntoElement, 0, 5)
	for range 5 {
		children = append(children, NewDiv().FlexNone().W(Px(40.)).HFull())
	}
	return NewDiv().SizeFull().Child(
		NewDiv().
			ID("strip").
			Flex().
			H(Px(20.)).
			W(Px(100.)).
			OverflowXScroll().
			TrackScroll(v.scrollHandle).
			Children(children...),
	)
}

func TestDiv_ScrollsToAnItemRequestedBeforeTheFirstFrame(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		scrollHandle := NewScrollHandle()
		scrollHandle.ScrollToItem(4)
		window := cx.AddWindow(func(_ *Window, _ *Context[scrollStrip]) *scrollStrip {
			return &scrollStrip{scrollHandle: scrollHandle}
		}).AnyWindowHandle
		defer divTestReleaseRoot(cx, window)
		divTestDraw(cx, window)

		if scrollHandle.Offset().X != Px(-100.) {
			t.Fatalf("offset x = %v, want -100", scrollHandle.Offset().X)
		}
	})
}

type contentSizedGrid struct{}

func (contentSizedGrid) Render(_ *Window, _ *Context[contentSizedGrid]) IntoElement {
	widths := []Pixels{Px(100.), Px(200.), Px(50.)}
	children := make([]IntoElement, 0, len(widths))
	for index, width := range widths {
		children = append(children, NewDiv().
			DebugSelector(func() string { return fmt.Sprintf("cell-%d", index) }).
			W(width).
			H(Px(10.)))
	}
	return NewDiv().SizeFull().Child(
		NewDiv().
			WFull().
			Grid().
			GridColsMaxContent(uint16(len(widths))).
			Children(children...),
	)
}

func TestDiv_GridColsMaxContentSizesColumnsToTheirContent(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		window := cx.AddWindow(func(_ *Window, _ *Context[contentSizedGrid]) *contentSizedGrid { return &contentSizedGrid{} }).AnyWindowHandle
		defer divTestReleaseRoot(cx, window)
		divTestDraw(cx, window)

		bounds := func(selector string) Bounds[Pixels] {
			var result Bounds[Pixels]
			found := false
			divTestUpdateWindow(cx, window, func(window *Window, _ *App) {
				result, found = window.renderedFrame.debugBounds[selector]
			})
			if !found {
				panic(selector + " was not rendered")
			}
			return result
		}

		for selector, want := range map[string]Pixels{"cell-0": Px(0.), "cell-1": Px(100.), "cell-2": Px(300.)} {
			if got := bounds(selector).Origin.X; got != want {
				t.Fatalf("%s origin x = %v, want %v", selector, got, want)
			}
		}
	})
}
