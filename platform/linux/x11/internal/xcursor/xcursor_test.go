//go:build linux

package xcursor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fakeImageWithSize(size uint32) Image {
	return Image{Size: size, Width: 42, Height: 42}
}

func TestParseCursor_FindBestSizeEmptyInput(t *testing.T) {
	if _, err := findBestSize(nil, 42); !errors.Is(err, errNoImages) {
		t.Fatalf("err = %v", err)
	}
}

func TestParseCursor_FindBestSizeOneInput(t *testing.T) {
	if got, err := findBestSize([]Image{fakeImageWithSize(42)}, 10); err != nil || got != 42 {
		t.Fatalf("got %d, %v", got, err)
	}
}

func TestParseCursor_FindBestSizeSelectsBetter(t *testing.T) {
	in := []Image{fakeImageWithSize(42), fakeImageWithSize(32), fakeImageWithSize(3), fakeImageWithSize(22)}
	if got, err := findBestSize(in, 10); err != nil || got != 3 {
		t.Fatalf("got %d, %v", got, err)
	}
}

func TestFindCursor_CursorShapeToID(t *testing.T) {
	if id, ok := cursorShapeToID("heart"); !ok || id != 31 {
		t.Fatalf("got %d, %v", id, ok)
	}
}

func le(vs ...uint32) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return b
}

func TestParser_ParseXcursor(t *testing.T) {
	var file []byte
	file = append(file, "Xcur"...)
	file = append(file, le(16, 0x10000, 1)...)
	file = append(file, le(0xfffd0002, 4, 28)...)
	file = append(file, le(36, 0xfffd0002, 4, 1, 2, 2, 1, 1, 7)...)
	pixels := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	file = append(file, pixels...)
	imgs, err := ParseXcursorStream(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	want := []Image{{
		Size: 4, Width: 2, Height: 2, Xhot: 1, Yhot: 1, Delay: 7,
		PixelsRGBA: pixels,
		PixelsARGB: []byte{4, 1, 2, 3, 8, 5, 6, 7, 12, 9, 10, 11, 16, 13, 14, 15},
	}}
	if !reflect.DeepEqual(imgs, want) {
		t.Fatalf("got %+v", imgs)
	}
}

func TestLib_ParseTheme(t *testing.T) {
	tests := map[string]string{
		"Inherits=breeze":                "breeze",
		"Inherits = breeze":              "breeze",
		"[Icon Theme]\nInherits=a,b":     "a",
		"Inherits=;,  core ;x":           "core",
		"Inherit=x\nInherits =\tAdwaita": "Adwaita",
	}
	for in, want := range tests {
		if got, ok := parseTheme(in); !ok || got != want {
			t.Errorf("%q: got %q, %v", in, got, ok)
		}
	}
	if _, ok := parseTheme("Inherits="); ok {
		t.Error("empty Inherits parsed")
	}
}

func TestLib_ThemeSearchPaths(t *testing.T) {
	got := themeSearchPaths(searchPathsEnvironment{home: "/home/u"})
	want := []string{"/home/u/.local/share/icons", "/home/u/.icons", "/usr/local/share/icons", "/usr/share/icons", "/usr/share/pixmaps", "/home/u/.cursors", "/usr/share/cursors/xorg-x11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	got = themeSearchPaths(searchPathsEnvironment{home: "/home/u", xcursorPath: "~/.icons::/a"})
	if !reflect.DeepEqual(got, []string{"/home/u/.icons", "/a"}) {
		t.Fatalf("got %v", got)
	}
}

func TestLib_LoadIconInherits(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "child"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "child", "index.theme"), []byte("Inherits=parent\n"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "parent", "cursors"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "parent", "cursors", "left_ptr"), []byte("x"), 0o644))
	theme := CursorTheme{theme: loadCursorThemeImpl("child", []string{dir}), searchPaths: []string{dir}}
	path, ok := theme.LoadIcon("left_ptr")
	if !ok || path != filepath.Join(dir, "parent", "cursors", "left_ptr") {
		t.Fatalf("got %q, %v", path, ok)
	}
	if _, ok := theme.LoadIcon("missing"); ok {
		t.Fatal("found a missing icon")
	}
}
