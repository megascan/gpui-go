//go:build linux

package xcbshim

import (
	"encoding/binary"
	"os"
	"runtime"
	"testing"
	"unsafe"
)

func fakeSetup(t *testing.T) ([]byte, uintptr) {
	t.Helper()
	var b []byte
	put8 := func(off int, v uint8) { b[off] = v }
	put16 := func(off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
	put32 := func(off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }

	b = make([]byte, 512)
	put16(24, 3)
	put8(28, 2)
	put8(29, 1)
	off := 40 + 3
	off = (off + 3) &^ 3
	off += 8

	screen0 := off
	put32(screen0+32, 0x21)
	put8(screen0+38, 24)
	put8(screen0+39, 2)
	d := screen0 + 40
	put8(d, 24)
	put16(d+2, 1)
	put32(d+8, 0x21)
	d += 8 + 24
	put8(d, 32)
	put16(d+2, 1)
	put32(d+8, 0x30)
	d += 8 + 24

	screen1 := d
	put32(screen1+32, 0x55)
	put8(screen1+38, 24)
	put8(screen1+39, 1)
	d = screen1 + 40
	put8(d, 24)
	put16(d+2, 1)
	put32(d+8, 0x56)

	return b, uintptr(unsafe.Pointer(&b[0]))
}

func TestXcbshim_ScreenWalkFakeSetup(t *testing.T) {
	b, setup := fakeSetup(t)
	defer runtime.KeepAlive(b)

	if id, ok := rootVisualOf(setup, 0); !ok || id != 0x21 {
		t.Fatalf("root visual screen 0 = %#x, %v; want 0x21", id, ok)
	}
	if id, ok := rootVisualOf(setup, 1); !ok || id != 0x55 {
		t.Fatalf("root visual screen 1 = %#x, %v; want 0x55", id, ok)
	}
	if _, ok := rootVisualOf(setup, 2); ok {
		t.Fatal("root visual for missing screen 2 reported found")
	}
	if id, ok := visualForDepthOf(setup, 0, 32); !ok || id != 0x30 {
		t.Fatalf("depth 32 visual screen 0 = %#x, %v; want 0x30", id, ok)
	}
	if id, ok := visualForDepthOf(setup, 1, 24); !ok || id != 0x56 {
		t.Fatalf("depth 24 visual screen 1 = %#x, %v; want 0x56", id, ok)
	}
	if _, ok := visualForDepthOf(setup, 1, 32); ok {
		t.Fatal("depth 32 visual on screen 1 reported found")
	}
	if _, ok := rootVisualOf(0, 0); ok {
		t.Fatal("null setup reported a root visual")
	}
}

func TestXcbshim_ConnectRejectsMissingDisplay(t *testing.T) {
	if err := Load(); err != nil {
		t.Skipf("libxcb unavailable: %v", err)
	}
	conn, err := Connect(":65530")
	if err == nil {
		conn.Close()
		t.Fatal("connect to a display with no server succeeded")
	}
}

func TestXcbshim_LiveConnection(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" || os.Getenv("DISPLAY") == "" {
		t.Skip("needs DISPLAY and GPUI_TEST_DISPLAY=1")
	}
	conn, err := Connect("")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(conn.Close)

	if conn.Raw() == 0 {
		t.Fatal("Raw returned a null connection")
	}
	if n := conn.ScreenCount(); conn.DefaultScreen() < 0 || conn.DefaultScreen() >= n {
		t.Fatalf("default screen %d outside %d screens", conn.DefaultScreen(), n)
	}
	if id, ok := conn.RootVisual(conn.DefaultScreen()); !ok || id == 0 {
		t.Fatalf("root visual = %#x, %v", id, ok)
	}
	if err := conn.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	conn.Close()
	conn.Close()
}
