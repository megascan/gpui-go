package main

import (
	"testing"

	"github.com/megascan/gpui-go"
)

type harness struct {
	a gpui.Entity[Editor]
	b gpui.Entity[Editor]
}

func (h *harness) Drop(_ *gpui.App) {
	h.a.Release()
	h.b.Release()
}

func (h *harness) Render(_ *gpui.Window, _ *gpui.Context[harness]) gpui.IntoElement {
	return gpui.NewDiv().
		Child(InputEditor(h.a.Clone())).
		Child(InputEditor(h.b.Clone()))
}

func bindKeys(cx *gpui.TestAppContext) {
	cx.Update(func(cx *gpui.App) {
		none := gpui.None[string]()
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("backspace", Backspace{}, none),
			gpui.NewKeyBinding("delete", Delete{}, none),
			gpui.NewKeyBinding("left", Left{}, none),
			gpui.NewKeyBinding("right", Right{}, none),
			gpui.NewKeyBinding("home", Home{}, none),
			gpui.NewKeyBinding("end", End{}, none),
		})
	})
}

type fixture struct {
	editor gpui.Entity[Editor]
	aValue gpui.Entity[string]
	bValue gpui.Entity[string]
	cx     *gpui.VisualTestContext
	view   gpui.Entity[harness]
}

func (f fixture) release() {
	f.editor.Release()
	f.aValue.Release()
	f.bValue.Release()
	f.view.Release()
}

func setup(cx *gpui.TestAppContext) fixture {
	bindKeys(cx)

	view, vcx := cx.AddWindowView(func(window *gpui.Window, cx *gpui.Context[harness]) *harness {
		aValue := cx.New(func(*gpui.Context[string]) *string { s := ""; return &s })
		bValue := cx.New(func(*gpui.Context[string]) *string { s := ""; return &s })
		a := cx.New(func(cx *gpui.Context[Editor]) *Editor { return EditorOver(aValue, window, cx) })
		b := cx.New(func(cx *gpui.Context[Editor]) *Editor { return EditorOver(bValue, window, cx) })
		return &harness{a: a, b: b}
	})

	a := view.ReadWith(vcx, func(h *harness, _ *gpui.App) gpui.Entity[Editor] { return h.a.Clone() })
	b := view.ReadWith(vcx, func(h *harness, _ *gpui.App) gpui.Entity[Editor] { return h.b.Clone() })
	defer b.Release()
	aValue := a.ReadWith(vcx, func(e *Editor, _ *gpui.App) gpui.Entity[string] { return e.value.Clone() })
	bValue := b.ReadWith(vcx, func(e *Editor, _ *gpui.App) gpui.Entity[string] { return e.value.Clone() })

	vcx.Update(func(window *gpui.Window, cx *gpui.App) {
		focusHandle := a.Read(cx).focusHandle.Clone()
		defer focusHandle.Release()
		window.Focus(focusHandle, cx)
	})

	return fixture{editor: a, aValue: aValue, bValue: bValue, cx: vcx, view: view}
}

func readString(e gpui.Entity[string], cx gpui.AppContext) string {
	return e.ReadWith(cx, func(s *string, _ *gpui.App) string { return *s })
}

func readCursor(e gpui.Entity[Editor], cx gpui.AppContext) int {
	return e.ReadWith(cx, func(e *Editor, _ *gpui.App) int { return e.cursor })
}

func TestExample_TypingUpdatesTheSharedString(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		f := setup(tc.Cx())
		defer f.release()

		f.cx.SimulateInput("hello")

		if got := readString(f.aValue, f.cx); got != "hello" {
			t.Fatalf("value = %q, want %q", got, "hello")
		}
		if got := readCursor(f.editor, f.cx); got != 5 {
			t.Fatalf("cursor = %d, want 5", got)
		}
	})
}

func TestExample_SiblingInputsAreIsolated(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		f := setup(tc.Cx())
		defer f.release()

		f.cx.SimulateInput("x")

		if got := readString(f.aValue, f.cx); got != "x" {
			t.Fatalf("value = %q, want %q", got, "x")
		}
		if got := readString(f.bValue, f.cx); got != "" {
			t.Fatalf("typing in input A must not touch input B: %q", got)
		}
	})
}

func TestExample_ExternalWritesClampTheCursor(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		f := setup(tc.Cx())
		defer f.release()

		f.cx.SimulateInput("hello")
		if got := readCursor(f.editor, f.cx); got != 5 {
			t.Fatalf("cursor = %d, want 5", got)
		}

		f.cx.Update(func(_ *gpui.Window, cx *gpui.App) {
			f.aValue.Update(cx, func(value *string, cx *gpui.Context[string]) {
				*value = "日本"
				cx.Notify()
			})
		})

		if got := readString(f.aValue, f.cx); got != "日本" {
			t.Fatalf("value = %q, want %q", got, "日本")
		}
		if got := readCursor(f.editor, f.cx); got != 3 {
			t.Fatalf("cursor must clamp to a char boundary: cursor = %d, want 3", got)
		}
	})
}

func TestExample_ArrowsMoveTheCursor(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		f := setup(tc.Cx())
		defer f.release()

		f.cx.SimulateInput("abc")
		if got := readCursor(f.editor, f.cx); got != 3 {
			t.Fatalf("cursor = %d, want 3", got)
		}

		f.cx.SimulateKeystrokes("left left")
		if got := readCursor(f.editor, f.cx); got != 1 {
			t.Fatalf("cursor = %d, want 1", got)
		}
	})
}
