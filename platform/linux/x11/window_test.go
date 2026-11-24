//go:build linux

package x11

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
)

func TestWindow_EdgeConstraintsToTiling(t *testing.T) {
	ec := edgeConstraintsFromAtom(1<<0 | 1<<4 | 1<<3)
	got := ec.toTiling()
	want := gpui.Tiling{Top: true, Bottom: true}
	if got != want {
		t.Fatalf("tiling = %+v, want %+v", got, want)
	}
	if !ec.rightResizable || ec.rightTiled {
		t.Fatalf("right flags = %+v", ec)
	}
}

func TestWindow_ResizeEdgeToMoveresize(t *testing.T) {
	edges := []gpui.ResizeEdge{gpui.ResizeEdgeTopLeft, gpui.ResizeEdgeTop, gpui.ResizeEdgeTopRight, gpui.ResizeEdgeRight,
		gpui.ResizeEdgeBottomRight, gpui.ResizeEdgeBottom, gpui.ResizeEdgeBottomLeft, gpui.ResizeEdgeLeft}
	for i, edge := range edges {
		if got := resizeEdgeToMoveresize(edge); got != uint32(i) {
			t.Fatalf("edge %d = %d", edge, got)
		}
	}
}

func TestWindow_WmHintsUrgencyRoundTrip(t *testing.T) {
	value := u32Bytes(hintInput|hintWindowGroup, 1, 0, 0, 0, 0, 0, 0, 42)
	reply := &xproto.GetPropertyReply{Format: 32, Type: xproto.AtomWmHints, Value: value, ValueLen: 9}
	hints, ok, err := parseWmHints(reply)
	if err != nil || !ok || hints.urgent {
		t.Fatalf("parse = %+v %v %v", hints, ok, err)
	}
	hints.urgent = true
	out := hints.serialize()
	flags := binary.LittleEndian.Uint32(out)
	if flags != hintInput|hintWindowGroup|hintUrgency {
		t.Fatalf("flags = %#x", flags)
	}
	if binary.LittleEndian.Uint32(out[8*4:]) != 42 || binary.LittleEndian.Uint32(out[4:]) != 1 {
		t.Fatalf("values not preserved: %v", out)
	}
	if _, ok, err := parseWmHints(&xproto.GetPropertyReply{Type: xproto.AtomNone}); ok || err != nil {
		t.Fatalf("absent property: ok=%v err=%v", ok, err)
	}
	if _, _, err := parseWmHints(&xproto.GetPropertyReply{Type: xproto.AtomString, Format: 32}); err == nil {
		t.Fatal("wrong type accepted")
	}
}

func TestWindow_WmSizeHints(t *testing.T) {
	out := wmSizeHintsNormal(gpui.Some([2]int32{100, 50}), gpui.Some([2]int32{8192, 8192}))
	if len(out) != numWmSizeHintsElements*4 {
		t.Fatalf("len = %d", len(out))
	}
	word := func(i int) uint32 { return binary.LittleEndian.Uint32(out[i*4:]) }
	if word(0) != pMinSize|pMaxSize || word(5) != 100 || word(6) != 50 || word(7) != 8192 || word(8) != 8192 {
		t.Fatalf("hints = %v", out)
	}
}

func TestWindow_IconPropertyData(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 4})
	img.SetNRGBA(1, 0, color.NRGBA{R: 5, G: 6, B: 7, A: 8})
	got := iconPropertyData(img)
	want := []uint32{2, 1, 0x04010203, 0x08050607}
	if len(got) != len(want) {
		t.Fatalf("data = %x", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("data = %x, want %x", got, want)
		}
	}
}

func TestWindow_FindVisuals(t *testing.T) {
	setup := &xproto.SetupInfo{Roots: []xproto.ScreenInfo{{
		Root: 7, RootVisual: 33, DefaultColormap: 9, RootDepth: 24, BlackPixel: 0,
		AllowedDepths: []xproto.DepthInfo{
			{Depth: 24, Visuals: []xproto.VisualInfo{{VisualId: 33, RedMask: 0xFF0000, GreenMask: 0xFF00, BlueMask: 0xFF}}},
			{Depth: 32, Visuals: []xproto.VisualInfo{{VisualId: 90, RedMask: 0xFF0000, GreenMask: 0xFF00, BlueMask: 0xFF}}},
		},
	}}}
	set := findVisuals(setup, 0)
	opaque, ok := set.opaque.Get()
	if !ok || opaque.id != 33 {
		t.Fatalf("opaque = %+v", set.opaque)
	}
	transparent, ok := set.transparent.Get()
	if !ok || transparent.id != 90 || transparent.depth != 32 || transparent.colormap != 0 {
		t.Fatalf("transparent = %+v", set.transparent)
	}
	if set.inherit.colormap != 9 || set.root != 7 {
		t.Fatalf("inherit = %+v", set)
	}
}
