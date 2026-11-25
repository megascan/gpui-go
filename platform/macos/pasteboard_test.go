//go:build darwin && !ios

package macos

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func pasteboardUnique(t *testing.T) *Pasteboard {
	t.Helper()
	var p *Pasteboard
	objc.AutoreleasePool(func() {
		name := objc.NSStringFromGo("zed-test-pasteboard-" + gpui.NewUUIDV4().String())
		defer objc.Release(name)
		p = newPasteboard(appkit.NSPasteboardWithName(name))
	})
	t.Cleanup(func() {
		p.ns().ReleaseGlobally()
		p.Release()
	})
	return p
}

func assertRead(t *testing.T, p *Pasteboard, want gpui.ClipboardItem, wantOK bool) {
	t.Helper()
	got, ok := p.Read()
	if ok != wantOK || (ok && !reflect.DeepEqual(got, want)) {
		t.Fatalf("read = %#v, %v; want %#v, %v", got, ok, want, wantOK)
	}
}

func TestPasteboard_String(t *testing.T) {
	pasteboard := pasteboardUnique(t)
	assertRead(t, pasteboard, gpui.ClipboardItem{}, false)

	item := gpui.NewClipboardItemString("1")
	pasteboard.Write(item)
	assertRead(t, pasteboard, item, true)

	item = gpui.ClipboardItem{Entries: []gpui.ClipboardEntry{gpui.ClipboardEntryString{
		ClipboardString: gpui.NewClipboardString("2").WithJSONMetadata([]int{3, 4}),
	}}}
	pasteboard.Write(item)
	assertRead(t, pasteboard, item, true)

	textFromOtherApp := "text from other app"
	objc.AutoreleasePool(func() {
		bytes := objc.NSDataFromBytes([]byte(textFromOtherApp))
		defer objc.Release(bytes)
		pasteboard.ns().SetDataForType(bytes, appkit.NSPasteboardTypeString())
	})
	assertRead(t, pasteboard, gpui.NewClipboardItemString(textFromOtherApp), true)

	for _, text := range []string{"", "héllo\x00世界🦀"} {
		item := gpui.NewClipboardItemString(text)
		pasteboard.Write(item)
		assertRead(t, pasteboard, item, true)
	}
}

func TestPasteboard_CustomTypesSurviveCreationAutoreleasePool(t *testing.T) {
	pasteboard := pasteboardUnique(t)

	if got := objc.GoString(pasteboard.textHashType.ID()); got != "zed-text-hash" {
		t.Fatalf("text hash type = %q", got)
	}
	if got := objc.GoString(pasteboard.metadataType.ID()); got != "zed-metadata" {
		t.Fatalf("metadata type = %q", got)
	}

	item := gpui.ClipboardItem{Entries: []gpui.ClipboardEntry{gpui.ClipboardEntryString{
		ClipboardString: gpui.NewClipboardString("retained pasteboard").WithJSONMetadata([]int{1, 2}),
	}}}
	objc.AutoreleasePool(func() { pasteboard.Write(item) })
	var got gpui.ClipboardItem
	var ok bool
	objc.AutoreleasePool(func() { got, ok = pasteboard.Read() })
	if !ok || !reflect.DeepEqual(got, item) {
		t.Fatalf("read = %#v, %v; want %#v", got, ok, item)
	}
}

func TestPasteboard_ReadExternalPath(t *testing.T) {
	pasteboard := pasteboardUnique(t)
	simulateExternalFileCopy(pasteboard, []string{"/test.txt"})

	item, ok := pasteboard.Read()
	if !ok {
		t.Fatal("should read clipboard item")
	}
	if len(item.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(item.Entries))
	}
	ep, ok := item.Entries[0].(gpui.ClipboardEntryExternalPaths)
	if !ok {
		t.Fatalf("expected ExternalPaths, got %#v", item.Entries[0])
	}
	if !slices.Equal(ep.Paths(), []string{"/test.txt"}) {
		t.Fatalf("paths = %v", ep.Paths())
	}
	s, ok := item.Entries[1].(gpui.ClipboardEntryString)
	if !ok {
		t.Fatalf("expected String, got %#v", item.Entries[1])
	}
	if s.Text != "/test.txt" {
		t.Fatalf("text = %q", s.Text)
	}
}

func TestPasteboard_ReadExternalPathsWithSpaces(t *testing.T) {
	pasteboard := pasteboardUnique(t)
	simulateExternalFileCopy(pasteboard, []string{"/some file with spaces.txt"})

	item, ok := pasteboard.Read()
	if !ok {
		t.Fatal("should read clipboard item")
	}
	ep, ok := item.Entries[0].(gpui.ClipboardEntryExternalPaths)
	if !ok {
		t.Fatalf("expected ExternalPaths, got %#v", item.Entries[0])
	}
	if !slices.Equal(ep.Paths(), []string{"/some file with spaces.txt"}) {
		t.Fatalf("paths = %v", ep.Paths())
	}
}

func TestPasteboard_ReadMultipleExternalPaths(t *testing.T) {
	pasteboard := pasteboardUnique(t)
	simulateExternalFileCopy(pasteboard, []string{"/file.txt", "/image.png"})

	item, ok := pasteboard.Read()
	if !ok {
		t.Fatal("should read clipboard item")
	}
	if len(item.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(item.Entries))
	}
	ep, ok := item.Entries[0].(gpui.ClipboardEntryExternalPaths)
	if !ok {
		t.Fatalf("expected ExternalPaths, got %#v", item.Entries[0])
	}
	if !slices.Equal(ep.Paths(), []string{"/file.txt", "/image.png"}) {
		t.Fatalf("paths = %v", ep.Paths())
	}
	s, ok := item.Entries[1].(gpui.ClipboardEntryString)
	if !ok {
		t.Fatalf("expected String, got %#v", item.Entries[1])
	}
	if s.Text != "/file.txt\n/image.png" {
		t.Fatalf("text = %q", s.Text)
	}
	if s.Metadata.IsSome() {
		t.Fatalf("metadata = %#v, want none", s.Metadata)
	}
}

func TestPasteboard_ReadImage(t *testing.T) {
	pasteboard := pasteboardUnique(t)

	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48,
		0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00,
		0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54, 0x78,
		0x9C, 0x62, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0xE5, 0x27, 0xDE, 0xFC, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	objc.AutoreleasePool(func() {
		nsPngType := appkit.NSPasteboardTypePNG()
		pasteboard.ns().DeclareTypes([]objc.ID{nsPngType}, objc.Nil)
		data := objc.NSDataFromBytes(pngBytes)
		defer objc.Release(data)
		pasteboard.ns().SetDataForType(data, nsPngType)
	})

	item, ok := pasteboard.Read()
	if !ok {
		t.Fatal("should read PNG image")
	}
	if len(item.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(item.Entries))
	}
	img, ok := item.Entries[0].(gpui.ClipboardEntryImage)
	if !ok {
		t.Fatalf("expected Image, got %#v", item.Entries[0])
	}
	if img.Format != gpui.ImageFormatPng {
		t.Fatalf("format = %v", img.Format)
	}
	if !slices.Equal(img.Bytes, pngBytes) {
		t.Fatalf("bytes = %x", img.Bytes)
	}
}

func simulateExternalFileCopy(pasteboard *Pasteboard, paths []string) {
	objc.AutoreleasePool(func() {
		nsPaths := make([]objc.ID, len(paths))
		for i, path := range paths {
			nsPaths[i] = objc.NSStringFromGo(path)
		}
		nsArray := objc.NSArrayFromIDs(nsPaths)
		for _, path := range nsPaths {
			objc.Release(path)
		}
		defer objc.Release(nsArray)

		filenamesType := appkit.Constant("NSFilenamesPboardType")
		pasteboard.ns().DeclareTypes([]objc.ID{filenamesType, appkit.NSPasteboardTypeString()}, objc.Nil)
		objc.SendVoid(pasteboard.inner.ID(), objc.Sel("setPropertyList:forType:"), nsArray, filenamesType)

		bytes := objc.NSDataFromBytes([]byte(strings.Join(paths, "\n")))
		defer objc.Release(bytes)
		pasteboard.ns().SetDataForType(bytes, appkit.NSPasteboardTypeString())
	})
}
