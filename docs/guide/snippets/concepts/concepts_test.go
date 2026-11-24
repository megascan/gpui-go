package concepts

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestConcepts_RunWithoutLeaks(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		cx.Update(handles)
		cx.Update(globals)

		document := cx.New(func(_ *gpui.Context[Document]) *Document { return &Document{} })
		defer document.Release()
		editor := cx.New(func(cx *gpui.Context[Editor]) *Editor { return NewEditor(document, cx) })
		defer editor.Release()

		cx.Update(func(cx *gpui.App) { reentrant(document, cx) })
		cx.RunUntilParked()

		renames := editor.ReadWith(cx, func(e *Editor, _ *gpui.App) int { return e.renames })
		title := document.ReadWith(cx, func(d *Document, _ *gpui.App) string { return d.title })
		if renames != 1 || title != "final" {
			t.Fatalf("renames = %d, title = %q", renames, title)
		}
	})
}

func TestConcepts_WorkspaceOwnsEditor(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		document := cx.New(func(_ *gpui.Context[Document]) *Document { return &Document{} })
		defer document.Release()

		workspace, vcx := cx.AddWindowView(func(_ *gpui.Window, cx *gpui.Context[Workspace]) *Workspace {
			return &Workspace{editor: cx.New(func(cx *gpui.Context[Editor]) *Editor { return NewEditor(document, cx) })}
		})
		defer workspace.Release()
		vcx.RunUntilParked()
	})
}
