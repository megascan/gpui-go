package gpui

import (
	"fmt"
	"testing"
)

type deferredTestSubscriptionLeaf struct {
	value  int
	output *int
}

type deferredTestSubscriptionPanel struct {
	leaf          Entity[deferredTestSubscriptionLeaf]
	deferredChild bool
	renders       *int
}

func (p *deferredTestSubscriptionPanel) Render(_ *Window, _ *Context[deferredTestSubscriptionPanel]) IntoElement {
	*p.renders++
	leaf := p.leaf
	child := NewCanvas(
		func(Bounds[Pixels], *Window, *App) struct{} { return struct{}{} },
		func(_ Bounds[Pixels], _ struct{}, _ *Window, cx *App) {
			leaf := leaf.Read(cx)
			*leaf.output = leaf.value
		},
	).Size(Px(50.))
	if p.deferredChild {
		return NewDeferred(child).IntoAnyElement()
	}
	return child.IntoAnyElement()
}

func (p *deferredTestSubscriptionPanel) Drop(*App) {
	p.leaf.Release()
}

type deferredTestSubscriptionRoot struct {
	panel         Entity[deferredTestSubscriptionPanel]
	deferredPanel bool
	renders       *int
}

func (r *deferredTestSubscriptionRoot) Render(_ *Window, _ *Context[deferredTestSubscriptionRoot]) IntoElement {
	*r.renders++
	var style StyleRefinement
	divTestStyled(&style).Size(Px(100.))
	panel := r.panel.Clone().Cached(style)
	if r.deferredPanel {
		return NewDeferred(panel).IntoAnyElement()
	}
	return panel.IntoAnyElement()
}

func (r *deferredTestSubscriptionRoot) Drop(*App) {
	r.panel.Release()
}

func TestDeferred_CachedReuseKeepsEntitySubscriptions(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		for _, deferredChild := range []bool{true, false} {
			for _, deferredPanel := range []bool{false, true} {
				output := 0
				renders := 0
				rootRenders := 0
				leaf := cx.New(func(*Context[deferredTestSubscriptionLeaf]) *deferredTestSubscriptionLeaf {
					return &deferredTestSubscriptionLeaf{value: 1, output: &output}
				})
				panel := cx.New(func(*Context[deferredTestSubscriptionPanel]) *deferredTestSubscriptionPanel {
					return &deferredTestSubscriptionPanel{leaf: leaf.Clone(), deferredChild: deferredChild, renders: &renders}
				})
				window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(*Window, *Context[deferredTestSubscriptionRoot]) *deferredTestSubscriptionRoot {
					return &deferredTestSubscriptionRoot{panel: panel, deferredPanel: deferredPanel, renders: &rootRenders}
				})
				cx.RunUntilParked()
				if output != 1 {
					t.Fatalf("output = %d, want 1", output)
				}
				initialRenders := renders
				for range 2 {
					windowTestMust(window.Update(cx, func(_ *deferredTestSubscriptionRoot, _ *Window, cx *Context[deferredTestSubscriptionRoot]) {
						cx.Notify()
					}))
					cx.RunUntilParked()
					if renders != initialRenders {
						t.Fatalf("panel should reuse its cache: renders %d, want %d", renders, initialRenders)
					}
				}

				rendersBeforeNotify := rootRenders
				leaf.Update(cx, func(leaf *deferredTestSubscriptionLeaf, cx *Context[deferredTestSubscriptionLeaf]) {
					leaf.value = 2
					cx.Notify()
				})
				cx.RunUntilParked()
				context := fmt.Sprintf("deferred child %v, deferred panel %v", deferredChild, deferredPanel)
				if rootRenders <= rendersBeforeNotify {
					t.Fatalf("leaf notification should redraw: %s", context)
				}
				windowTestMust(window.Update(cx, func(_ *deferredTestSubscriptionRoot, window *Window, _ *Context[deferredTestSubscriptionRoot]) {
					window.Refresh()
				}))
				cx.RunUntilParked()
				if output != 2 {
					t.Fatalf("output = %d, want 2: %s", output, context)
				}
				leaf.Release()
			}
		}
	})
}

type deferredTestPanelView struct{}

func (*deferredTestPanelView) Render(_ *Window, _ *Context[deferredTestPanelView]) IntoElement {
	return NewDiv().KeyContext("Panel").SizeFull().Child(
		NewDeferred(
			NewAnchored().Position(NewPoint(Px(10.), Px(10.))).Child(
				NewDiv().KeyContext("Popover").W(Px(200.)).H(Px(200.)).Child(
					NewDeferred(
						NewAnchored().Position(NewPoint(Px(30.), Px(30.))).Child(
							NewDiv().
								KeyContext("NestedMenu").
								DebugSelector(func() string { return "NESTED_MENU" }).
								W(Px(50.)).
								H(Px(50.)),
						),
					).WithPriority(2),
				),
			),
		).WithPriority(1),
	)
}

type deferredTestRootView struct {
	panel Entity[deferredTestPanelView]
}

func (r *deferredTestRootView) Render(_ *Window, _ *Context[deferredTestRootView]) IntoElement {
	var style StyleRefinement
	divTestStyled(&style).SizeFull()
	return NewDiv().KeyContext("Root").SizeFull().Child(r.panel.Clone().Cached(style))
}

func (r *deferredTestRootView) Drop(*App) {
	r.panel.Release()
}

func TestDeferred_NestedDeferredDrawsWithReusedViews(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		defer windowTestReleaseAll(cx)
		window := cx.OpenWindow(NewSize(Px(800.), Px(600.)), func(_ *Window, cx *Context[deferredTestRootView]) *deferredTestRootView {
			panel := AppContextNew(cx, func(*Context[deferredTestPanelView]) *deferredTestPanelView { return &deferredTestPanelView{} })
			return &deferredTestRootView{panel: panel}
		})
		cx.RunUntilParked()

		var menuBounds Bounds[Pixels]
		var found bool
		windowTestMust(window.Update(cx, func(_ *deferredTestRootView, window *Window, _ *Context[deferredTestRootView]) {
			menuBounds, found = window.renderedFrame.debugBounds["NESTED_MENU"]
		}))
		if !found {
			t.Fatal("NESTED_MENU debug bounds not found")
		}
		if menuBounds.Size != NewSize(Px(50.), Px(50.)) {
			t.Fatalf("menu size = %v", menuBounds.Size)
		}

		windowTestMust(window.Update(cx, func(_ *deferredTestRootView, _ *Window, cx *Context[deferredTestRootView]) { cx.Notify() }))
		cx.RunUntilParked()

		windowTestMust(window.Update(cx, func(_ *deferredTestRootView, _ *Window, cx *Context[deferredTestRootView]) { cx.Notify() }))
		cx.RunUntilParked()

		windowTestMust(window.Update(cx, func(root *deferredTestRootView, _ *Window, cx *Context[deferredTestRootView]) {
			root.panel.Update(cx, func(_ *deferredTestPanelView, cx *Context[deferredTestPanelView]) { cx.Notify() })
		}))
		cx.RunUntilParked()

		windowTestMust(window.Update(cx, func(_ *deferredTestRootView, window *Window, _ *Context[deferredTestRootView]) {
			if got := len(window.renderedFrame.deferredDraws); got != 2 {
				t.Fatalf("deferred draws = %d, want 2", got)
			}
			if _, ok := window.renderedFrame.debugBounds["NESTED_MENU"]; !ok {
				t.Fatal("NESTED_MENU debug bounds not found")
			}
		}))
	})
}
