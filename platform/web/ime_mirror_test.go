//go:build js && wasm

package web

import (
	"testing"
	"unicode/utf16"

	"github.com/megascan/gpui-go"
)

type fakeTextArea struct {
	text       string
	start, end uint32
	writes     int
}

func (a *fakeTextArea) value() string { return a.text }

func (a *fakeTextArea) setValue(value string) {
	a.text = value
	n := uint32(utf16Len(value))
	a.start, a.end = n, n
	a.writes++
}

func (a *fakeTextArea) selectionStart() (uint32, bool) { return a.start, true }
func (a *fakeTextArea) selectionEnd() (uint32, bool)   { return a.end, true }

func (a *fakeTextArea) setSelectionRange(start, end uint32) {
	n := uint32(utf16Len(a.text))
	a.start, a.end = min(start, n), min(end, n)
	a.writes++
}

type fakeDocument struct {
	units     []uint16
	selection gpui.Range
	editable  gpui.Option[gpui.Range]
}

func (d *fakeDocument) SelectedTextRange(bool) (gpui.UTF16Selection, bool) {
	return gpui.UTF16Selection{Range: d.selection}, true
}

func (d *fakeDocument) TextForRange(r gpui.Range, adjusted *gpui.Option[gpui.Range]) (string, bool) {
	start, end := min(r.Start, len(d.units)), min(r.End, len(d.units))
	*adjusted = gpui.Some(gpui.Range{Start: start, End: end})
	return string(utf16.Decode(d.units[start:end])), true
}

func (d *fakeDocument) TextInputEditableRange() (gpui.Range, bool) {
	return d.editable.Get()
}

func newTestMirror() (*imeMirror, *fakeTextArea) {
	area := &fakeTextArea{}
	return &imeMirror{area: area}, area
}

func TestImeMirror_SyncMirrorsWindowAndSkipsConsistentWrites(t *testing.T) {
	mirror, area := newTestMirror()
	doc := &fakeDocument{units: utf16.Encode([]rune("hello world")), selection: gpui.Range{Start: 5, End: 5}}
	mirror.sync(false, doc)
	if area.text != "hello world" || mirror.selection != [2]uint32{5, 5} {
		t.Fatalf("after sync: %q %v", area.text, mirror.selection)
	}
	writes := area.writes
	mirror.sync(false, doc)
	if area.writes != writes {
		t.Fatalf("consistent sync wrote to the element")
	}
	doc.selection = gpui.Range{Start: 2, End: 4}
	mirror.sync(false, doc)
	if area.text != "hello world" || mirror.selection != [2]uint32{2, 4} || area.writes != writes+1 {
		t.Fatalf("selection move: %q %v writes %d", area.text, mirror.selection, area.writes-writes)
	}
}

func TestImeMirror_SyncClearsWithoutSelectionAndDefersToPendingImport(t *testing.T) {
	mirror, area := newTestMirror()
	doc := &fakeDocument{units: utf16.Encode([]rune("abc")), selection: gpui.Range{Start: 3, End: 3}}
	mirror.sync(false, doc)
	area.start = 1
	mirror.sync(false, &fakeDocument{units: utf16.Encode([]rune("xyz")), selection: gpui.Range{Start: 0, End: 0}})
	if area.text != "abc" {
		t.Fatal("sync clobbered a pending selection import")
	}
	mirror.rejectSelectionImport()
	mirror.sync(false, nil)
	if area.text != "" || mirror.text != "" || mirror.selection != [2]uint32{0, 0} {
		t.Fatalf("no handler: %q %q %v", area.text, mirror.text, mirror.selection)
	}
}

func TestImeMirror_SyncRespectsEditableRange(t *testing.T) {
	mirror, area := newTestMirror()
	doc := &fakeDocument{
		units:     utf16.Encode([]rune("prompt> typed")),
		selection: gpui.Range{Start: 13, End: 13},
		editable:  gpui.Some(gpui.Range{Start: 8, End: 13}),
	}
	mirror.sync(false, doc)
	if area.text != "typed" || mirror.selection != [2]uint32{5, 5} || mirror.windowHint != 8 {
		t.Fatalf("editable window: %q %v hint %d", area.text, mirror.selection, mirror.windowHint)
	}
}
