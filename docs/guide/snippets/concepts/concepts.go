// Package concepts holds the snippets of docs/guide/concepts.md and docs/guide/pitfalls.md.
package concepts

import (
	"fmt"

	"github.com/megascan/gpui-go"
)

// Document is a model entity.
type Document struct {
	title string
}

// Changed is emitted by Document.
type Changed struct{}

// Rename changes the title and tells observers.
func (d *Document) Rename(title string, cx *gpui.Context[Document]) {
	d.title = title
	cx.Notify()
	cx.Emit(Changed{})
}

// Editor is a view that shows a document.
type Editor struct {
	document     gpui.Entity[Document]
	renames      int
	observation  gpui.Subscription
	subscription gpui.Subscription
}

// NewEditor creates an editor. It clones the document handle it keeps.
func NewEditor(document gpui.Entity[Document], cx *gpui.Context[Editor]) *Editor {
	return &Editor{
		document: document.Clone(),
		observation: cx.Observe(document, func(_ *Editor, _ gpui.Entity[Document], cx *gpui.Context[Editor]) {
			cx.Notify()
		}),
		subscription: cx.Subscribe(document, func(this *Editor, _ gpui.Entity[Document], _ *Changed, _ *gpui.Context[Editor]) {
			this.renames++
		}),
	}
}

// Drop releases what the editor owns.
func (e *Editor) Drop(_ *gpui.App) {
	e.observation.Unsubscribe()
	e.subscription.Unsubscribe()
	e.document.Release()
}

// Render renders the view.
func (e *Editor) Render(_ *gpui.Window, cx *gpui.Context[Editor]) gpui.IntoElement {
	title := e.document.Read(cx.App).title
	return gpui.NewDiv().ChildText(fmt.Sprintf("%s (renamed %d times)", title, e.renames))
}

func handles(cx *gpui.App) {
	document := cx.New(func(_ *gpui.Context[Document]) *Document {
		return &Document{title: "untitled"}
	})
	defer document.Release()

	editor := cx.New(func(cx *gpui.Context[Editor]) *Editor {
		return NewEditor(document, cx)
	})
	defer editor.Release()

	document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {
		d.Rename("notes.txt", cx)
	})

	weak := document.Downgrade()
	if strong, ok := weak.Upgrade(); ok {
		fmt.Println(strong.Read(cx).title)
		strong.Release()
	}
}

// Workspace owns a child view and renders it.
type Workspace struct {
	editor gpui.Entity[Editor]
}

// Drop releases the child view.
func (w *Workspace) Drop(_ *gpui.App) {
	w.editor.Release()
}

// Render renders the view.
func (w *Workspace) Render(_ *gpui.Window, _ *gpui.Context[Workspace]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().Child(w.editor.Clone())
}

func reentrant(document gpui.Entity[Document], cx *gpui.App) {
	document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {
		d.title = "draft"
		cx.Defer(func(cx *gpui.App) {
			document.Update(cx, func(d *Document, cx *gpui.Context[Document]) {
				d.Rename("final", cx)
			})
		})
	})
}

// Theme is app-wide state stored as a global.
type Theme struct {
	Dark bool
}

func globals(cx *gpui.App) {
	cx.SetGlobal(Theme{Dark: true})
	theme := cx.Global[Theme]()
	fmt.Println(theme.Dark)
	gpui.UpdateGlobal(cx, func(theme *Theme, _ *gpui.App) {
		theme.Dark = false
	})
	if theme, ok := cx.TryGlobal[Theme](); ok {
		fmt.Println(theme.Dark)
	}
}
