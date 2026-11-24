package gpui

import "testing"

type anchoredTestView struct {
	position Point[Pixels]
}

func (v *anchoredTestView) Render(_ *Window, _ *Context[anchoredTestView]) IntoElement {
	return NewDiv().SizeFull().Child(
		NewDiv().
			ID("scroll-container").
			OverflowYScroll().
			SizeFull().
			Child(NewDiv().H(Px(2000.)).WFull()).
			Child(
				NewDeferred(
					NewAnchored().
						SnapToWindow().
						Position(v.position).
						Child(
							NewDiv().
								ID("menu").
								DebugSelector(func() string { return "MENU" }).
								W(Px(200.)).
								H(Px(300.)),
						),
				).WithPriority(1),
			),
	)
}

func anchoredTestMenuBounds[V any](t *testing.T, cx *TestAppContext, window WindowHandle[V]) Bounds[Pixels] {
	t.Helper()
	var bounds Bounds[Pixels]
	var found bool
	windowTestMust(window.Update(cx, func(_ *V, window *Window, _ *Context[V]) {
		bounds, found = window.renderedFrame.debugBounds["MENU"]
	}))
	if !found {
		t.Fatal("MENU debug bounds not found")
	}
	return bounds
}

func TestAnchored_AnchoredPositionWithoutScroll(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(*Window, *Context[anchoredTestView]) *anchoredTestView {
			return &anchoredTestView{position: NewPoint(Px(100.), Px(100.))}
		})

		cx.RunUntilParked()

		menuBounds := anchoredTestMenuBounds(t, cx, window)
		if menuBounds.Origin != NewPoint(Px(100.), Px(100.)) {
			t.Fatalf("origin = %v", menuBounds.Origin)
		}
		if menuBounds.Size != NewSize(Px(200.), Px(300.)) {
			t.Fatalf("size = %v", menuBounds.Size)
		}
	})
}

func TestAnchored_AnchoredPositionWhenScrolled(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(*Window, *Context[anchoredTestView]) *anchoredTestView {
			return &anchoredTestView{position: NewPoint(Px(100.), Px(100.))}
		})

		cx.RunUntilParked()

		windowTestMust(window.Update(cx, func(_ *anchoredTestView, window *Window, cx *Context[anchoredTestView]) {
			event := ScrollEvent{
				Position: NewPoint(Px(400.), Px(300.)),
				Delta:    ScrollDeltaPixels(NewPoint(Px(0.), Px(-1000.))),
			}
			window.DispatchEvent(event, cx.App)
		}))

		cx.RunUntilParked()

		menuBounds := anchoredTestMenuBounds(t, cx, window)
		if menuBounds.Origin != NewPoint(Px(100.), Px(100.)) {
			t.Fatalf("origin = %v", menuBounds.Origin)
		}
		if menuBounds.Size != NewSize(Px(200.), Px(300.)) {
			t.Fatalf("size = %v", menuBounds.Size)
		}
	})
}

func TestAnchored_AnchoredSnapsToWindow(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(*Window, *Context[anchoredTestView]) *anchoredTestView {
			return &anchoredTestView{position: NewPoint(Px(100.), Px(500.))}
		})

		cx.RunUntilParked()

		menuBounds := anchoredTestMenuBounds(t, cx, window)
		if menuBounds.Origin != NewPoint(Px(100.), Px(300.)) {
			t.Fatalf("origin = %v", menuBounds.Origin)
		}
		if menuBounds.Size != NewSize(Px(200.), Px(300.)) {
			t.Fatalf("size = %v", menuBounds.Size)
		}
	})
}

type anchoredTestAttachedView struct {
	target Bounds[Pixels]
	corner Anchor
	anchor Anchor
}

func (v *anchoredTestAttachedView) Render(_ *Window, _ *Context[anchoredTestAttachedView]) IntoElement {
	return NewDiv().SizeFull().Child(
		NewDeferred(
			NewAnchored().
				Attach(v.target, v.corner).
				Anchor(v.anchor).
				Offset(NewPoint(Px(-2.), Px(4.))).
				SnapToWindowWithMargin(EdgesAll(Px(8.))).
				Child(
					NewDiv().
						DebugSelector(func() string { return "MENU" }).
						W(Px(200.)).
						H(Px(300.)),
				),
		).WithPriority(1),
	)
}

func TestAnchored_AnchoredAttachMovesToTheSideWithRoom(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		trigger := func(x, y float32) Bounds[Pixels] {
			return NewBounds(NewPoint(Px(x), Px(y)), NewSize(Px(100.), Px(20.)))
		}
		type placement struct{ corner, anchor Anchor }
		submenu := placement{AnchorTopRight, AnchorTopLeft}
		dropdown := placement{AnchorBottomLeft, AnchorTopLeft}
		cases := []struct {
			target         Bounds[Pixels]
			placement      placement
			expectedOrigin Point[Pixels]
		}{
			{trigger(100., 100.), submenu, NewPoint(Px(198.), Px(104.))},
			{trigger(550., 100.), submenu, NewPoint(Px(352.), Px(104.))},
			{trigger(100., 400.), dropdown, NewPoint(Px(98.), Px(96.))},
			{trigger(498., 100.), submenu, NewPoint(Px(596.), Px(104.))},
			{NewBounds(NewPoint(Px(180.), Px(100.)), NewSize(Px(500.), Px(20.))), submenu, NewPoint(Px(8.), Px(104.))},
			{NewBounds(NewPoint(Px(50.), Px(100.)), NewSize(Px(650.), Px(20.))), submenu, NewPoint(Px(592.), Px(104.))},
			{trigger(700., 100.), dropdown, NewPoint(Px(592.), Px(124.))},
			{trigger(100., 400.), placement{AnchorBottomCenter, AnchorTopCenter}, NewPoint(Px(48.), Px(96.))},
		}
		for _, c := range cases {
			window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(*Window, *Context[anchoredTestAttachedView]) *anchoredTestAttachedView {
				return &anchoredTestAttachedView{target: c.target, corner: c.placement.corner, anchor: c.placement.anchor}
			})

			cx.RunUntilParked()

			menuBounds := anchoredTestMenuBounds(t, cx, window)
			if menuBounds.Origin != c.expectedOrigin {
				t.Fatalf("target: %v, corner: %v, anchor: %v: origin = %v, want %v", c.target, c.placement.corner, c.placement.anchor, menuBounds.Origin, c.expectedOrigin)
			}
		}
	})
}
