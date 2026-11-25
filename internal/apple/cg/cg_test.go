//go:build darwin

package cg

import (
	"testing"
	"unsafe"
)

func TestCg_Layout(t *testing.T) {
	if s := unsafe.Sizeof(AffineTransform{}); s != 48 {
		t.Fatalf("sizeof(CGAffineTransform) = %d, want 48", s)
	}
	if s := unsafe.Sizeof(Rect{}); s != 32 {
		t.Fatalf("sizeof(CGRect) = %d, want 32", s)
	}
	if ImageAlphaOnly != 7 || ImageAlphaPremultipliedLast != 1 || TextFill != 0 {
		t.Fatal("CoreGraphics constants wrong")
	}
}

func TestCg_BitmapContext(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	cs := ColorSpaceCreateDeviceGray()
	defer cs.Release()
	buf := make([]byte, 4*4)
	cx := CreateBitmapContext(buf, 4, 4, 8, 4, cs, ImageAlphaOnly)
	if cx == nil {
		t.Fatal("CGBitmapContextCreate failed")
	}
	cx.SetGrayFillColor(0, 1)
	cx.FillRect(Rect{Size: Size{Width: 4, Height: 4}})
	cx.Release()
	cx.Release()
	for i, b := range buf {
		if b != 255 {
			t.Fatalf("pixel %d = %d, want 255", i, b)
		}
	}
	rgb := ColorSpaceCreateDeviceRGB()
	if rgb == 0 {
		t.Fatal("CGColorSpaceCreateDeviceRGB failed")
	}
	rgb.Release()
}
