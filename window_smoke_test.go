package gpui

import (
	"testing"
)

type smokeElement struct {
	clicks *int
	keys   *int
	focus  FocusHandle
	bounds Bounds[Pixels]
}

func (e *smokeElement) ElementID() (ElementID, bool) { return ElementID{}, false }

func (e *smokeElement) SourceLocation() *SourceLocation { return nil }

func (e *smokeElement) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Size = SizeOf[Length]{Width: e.bounds.Size.Width.ToLength(), Height: e.bounds.Size.Height.ToLength()}
	return window.RequestLayout(style, nil, cx), struct{}{}
}

func (e *smokeElement) Prepaint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, window *Window, cx *App) Hitbox {
	window.SetFocusHandle(e.focus, cx)
	return window.InsertHitbox(bounds, HitboxBehaviorNormal)
}

func (e *smokeElement) Paint(_ *GlobalElementID, _ *InspectorElementID, bounds Bounds[Pixels], _ *struct{}, hitbox *Hitbox, window *Window, _ *App) {
	window.PaintQuad(NewFill(bounds, NewRgba(0xff0000ff).ToBackground()))
	clicks, keys := e.clicks, e.keys
	h := *hitbox
	window.OnKeyEvent(func(event *KeyDownEvent, phase DispatchPhase, _ *Window, _ *App) {
		if phase == DispatchPhaseBubble && event.Keystroke.Key == "a" {
			*keys++
		}
	})
	window.OnMouseEvent(func(event *MouseDownEvent, phase DispatchPhase, window *Window, _ *App) {
		if phase == DispatchPhaseBubble && h.IsHovered(window) {
			*clicks++
		}
	})
}

type smokeView struct {
	clicks int
	keys   int
	focus  FocusHandle
}

func (v *smokeView) Render(_ *Window, _ *Context[smokeView]) IntoElement {
	return NewAnyElement[struct{}, Hitbox](&smokeElement{
		clicks: &v.clicks,
		keys:   &v.keys,
		focus:  v.focus,
		bounds: Bounds[Pixels]{Size: Size[Pixels]{Width: Px(40), Height: Px(30)}},
	})
}

func TestWindowSmoke_ElementLayoutPaintAndClick(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		view, vcx := cx.AddWindowView(func(_ *Window, cx *Context[smokeView]) *smokeView { return &smokeView{focus: cx.FocusHandle()} })
		defer view.Release()

		vcx.Update(func(window *Window, cx *App) {
			window.Draw(cx).Clear(cx)
			quads := window.PaintedQuads()
			if len(quads) != 1 {
				t.Fatalf("expected 1 painted quad, got %d", len(quads))
			}
			want := Bounds[Pixels]{Size: Size[Pixels]{Width: Px(40), Height: Px(30)}}.Scale(window.ScaleFactor())
			if quads[0].Bounds != want {
				t.Fatalf("quad bounds = %v, want %v", quads[0].Bounds, want)
			}
		})

		vcx.SimulateClick(Point[Pixels]{X: Px(10), Y: Px(10)}, Modifiers{})
		vcx.SimulateClick(Point[Pixels]{X: Px(100), Y: Px(100)}, Modifiers{})

		clicks := view.ReadWith(cx, func(v *smokeView, _ *App) int { return v.clicks })
		if clicks != 1 {
			t.Fatalf("clicks = %d, want 1", clicks)
		}

		vcx.Update(func(window *Window, cx *App) {
			window.Focus(view.Read(cx).focus, cx)
		})
		vcx.SimulateKeystrokes("a")
		keys := view.ReadWith(cx, func(v *smokeView, _ *App) int { return v.keys })
		if keys != 1 {
			t.Fatalf("keys = %d, want 1", keys)
		}

		vcx.Update(func(window *Window, cx *App) {
			view.Read(cx).focus.Release()
		})
		vcx.RunUntilParked()
		vcx.Update(func(window *Window, cx *App) {
			if root, ok := window.root.Take().Get(); ok {
				root.Release()
			}
		})
	})
}
