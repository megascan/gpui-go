//go:build linux

package wlproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/linux/wl"
)

var coreInterfaces = []*wl.Interface{
	WlDisplayInterface, WlRegistryInterface, WlCallbackInterface, WlCompositorInterface,
	WlShmPoolInterface, WlShmInterface, WlBufferInterface, WlDataOfferInterface,
	WlDataSourceInterface, WlDataDeviceInterface, WlDataDeviceManagerInterface,
	WlShellInterface, WlShellSurfaceInterface, WlSurfaceInterface, WlSeatInterface,
	WlPointerInterface, WlKeyboardInterface, WlTouchInterface, WlOutputInterface,
	WlRegionInterface, WlSubcompositorInterface, WlSubsurfaceInterface,
	WlFixesInterface,
}

func TestTables_MatchLibwaylandCore(t *testing.T) {
	if err := wl.Load(); err != nil {
		t.Skipf("libwayland-client not available: %v", err)
	}
	for _, iface := range coreInterfaces {
		theirs, err := wl.LibrarySymbol(iface.Name + "_interface")
		if err != nil {
			t.Logf("%s not exported by this libwayland: %v", iface.Name, err)
			continue
		}
		ours := iface.C()
		om, oe := wl.MessageCounts(ours)
		tm, te := wl.MessageCounts(theirs)
		got := wl.DescribeCPrefix(ours, min(om, tm), min(oe, te))
		want := wl.DescribeCPrefix(theirs, min(om, tm), min(oe, te))
		gotHeader, gotBody, _ := strings.Cut(got, "\n")
		_, wantBody, _ := strings.Cut(want, "\n")
		if !strings.HasPrefix(gotHeader, iface.Name+" v") || !strings.HasPrefix(want, iface.Name+" v") {
			t.Errorf("%s headers: ours %q theirs %q", iface.Name, gotHeader, want[:strings.IndexByte(want+"\n", '\n')])
		}
		if gotBody != wantBody {
			t.Errorf("%s shared messages differ:\nours   %s\ntheirs %s", iface.Name, got, want)
		}
	}
}

func TestTables_ReferencesResolve(t *testing.T) {
	all := []*wl.Interface{
		XdgWmBaseInterface, XdgToplevelInterface, XdgPopupInterface, ZxdgToplevelDecorationV1Interface,
		XdgActivationV1Interface, XdgWmDialogV1Interface, XdgSystemBellV1Interface,
		WpFractionalScaleManagerV1Interface, WpViewporterInterface, WpCursorShapeManagerV1Interface,
		ZwpTextInputManagerV3Interface, ZwpPrimarySelectionDeviceManagerV1Interface,
		ZwpPointerGesturesV1Interface, ZwpLinuxDmabufV1Interface, ZwpLinuxDmabufFeedbackV1Interface,
		ZwlrLayerShellV1Interface, ZwlrLayerSurfaceV1Interface, OrgKdeKwinBlurManagerInterface,
		ExtBackgroundEffectManagerV1Interface, ZwpTabletToolV2Interface,
	}
	for _, iface := range all {
		if iface.New == nil {
			t.Fatalf("%s has no constructor", iface.Name)
		}
		for _, group := range [][]wl.Message{iface.Methods, iface.Events} {
			for _, m := range group {
				for _, ty := range m.Types {
					if ty != nil && ty.New == nil {
						t.Fatalf("%s.%s references %s without a constructor", iface.Name, m.Name, ty.Name)
					}
				}
			}
		}
		if s := wl.DescribeC(iface.C()); !strings.HasPrefix(s, iface.Name) {
			t.Fatalf("table for %s describes as %q", iface.Name, s)
		}
	}
	if got := wl.DescribeC(WpCursorShapeManagerV1Interface.C()); !strings.Contains(got, "zwp_tablet_tool_v2") {
		t.Fatalf("cursor-shape table does not reference the tablet tool:\n%s", got)
	}
}

type fakeServer struct {
	t  *testing.T
	fd int
}

type wire struct{ bytes.Buffer }

func (w *wire) u32(v uint32) *wire {
	_ = binary.Write(w, binary.NativeEndian, v)
	return w
}

func (w *wire) i32(v int32) *wire { return w.u32(uint32(v)) }

func (w *wire) str(s string) *wire {
	w.u32(uint32(len(s) + 1))
	w.WriteString(s)
	w.WriteByte(0)
	for w.Len()%4 != 0 {
		w.WriteByte(0)
	}
	return w
}

func (w *wire) arr(b []byte) *wire {
	w.u32(uint32(len(b)))
	w.Write(b)
	for w.Len()%4 != 0 {
		w.WriteByte(0)
	}
	return w
}

func (s *fakeServer) send(id uint32, opcode uint16, payload *wire, fds ...int) {
	s.t.Helper()
	var body []byte
	if payload != nil {
		body = payload.Bytes()
	}
	msg := make([]byte, 8+len(body))
	binary.NativeEndian.PutUint32(msg[0:], id)
	binary.NativeEndian.PutUint32(msg[4:], uint32(len(msg))<<16|uint32(opcode))
	copy(msg[8:], body)
	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	if err := syscall.Sendmsg(s.fd, msg, oob, nil, 0); err != nil {
		s.t.Fatalf("sendmsg: %v", err)
	}
}

func (s *fakeServer) readFull(n int) []byte {
	s.t.Helper()
	buf := make([]byte, n)
	for off := 0; off < n; {
		k, err := syscall.Read(s.fd, buf[off:])
		if err != nil || k == 0 {
			s.t.Fatalf("read: %d %v", k, err)
		}
		off += k
	}
	return buf
}

func (s *fakeServer) expect(id uint32, opcode uint16, payload *wire) {
	s.t.Helper()
	hdr := s.readFull(8)
	gotID := binary.NativeEndian.Uint32(hdr[0:])
	word := binary.NativeEndian.Uint32(hdr[4:])
	size := int(word >> 16)
	body := s.readFull(size - 8)
	var want []byte
	if payload != nil {
		want = payload.Bytes()
	}
	if gotID != id || uint16(word) != opcode || !bytes.Equal(body, want) {
		s.t.Fatalf("request = %d.%d %x, want %d.%d %x", gotID, uint16(word), body, id, opcode, want)
	}
}

func newPair(t *testing.T) (*wl.Display, *fakeServer) {
	t.Helper()
	if err := wl.Load(); err != nil {
		t.Skipf("libwayland-client not available: %v", err)
	}
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	d, err := wl.ConnectToFD(fds[0])
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{t: t, fd: fds[1]}
	t.Cleanup(func() {
		d.Disconnect()
		_ = syscall.Close(fds[1])
	})
	return d, s
}

func flush(t *testing.T, d *wl.Display) {
	t.Helper()
	if err := d.Flush(); err != nil {
		t.Fatal(err)
	}
}

func dispatch(t *testing.T, d *wl.Display) {
	t.Helper()
	if _, err := d.Dispatch(); err != nil {
		t.Fatal(err)
	}
}

func TestWire_RoundTrip(t *testing.T) {
	d, s := newPair(t)
	display := GetDisplay(d)
	reg := display.GetRegistry()
	type global struct {
		name    uint32
		iface   string
		version uint32
	}
	var globals []global
	reg.SetListener(WlRegistryListener{Global: func(name uint32, iface string, version uint32) {
		globals = append(globals, global{name, iface, version})
	}})
	flush(t, d)
	s.expect(1, 1, new(wire).u32(2))
	if reg.ID() != 2 || reg.Version() != 1 {
		t.Fatalf("registry %s version %d", reg, reg.Version())
	}

	s.send(2, 0, new(wire).u32(7).str("wl_seat").u32(9))
	s.send(2, 0, new(wire).u32(8).str("wl_compositor").u32(6))
	s.send(2, 0, new(wire).u32(9).str("wl_data_device_manager").u32(3))
	dispatch(t, d)
	if len(globals) != 3 || globals[0] != (global{7, "wl_seat", 9}) || globals[1].iface != "wl_compositor" {
		t.Fatalf("globals = %v", globals)
	}

	seat := Bind[WlSeat](reg, 7, WlSeatInterface, 5)
	flush(t, d)
	s.expect(2, 0, new(wire).u32(7).str("wl_seat").u32(5).u32(3))
	if seat == nil || seat.ID() != 3 || seat.Version() != 5 {
		t.Fatalf("seat = %v", seat)
	}
	var caps WlSeatCapability
	var seatName string
	seat.SetListener(WlSeatListener{
		Capabilities: func(c WlSeatCapability) { caps = c },
		Name:         func(n string) { seatName = n },
	})
	s.send(3, 0, new(wire).u32(uint32(WlSeatCapabilityPointer|WlSeatCapabilityKeyboard)))
	s.send(3, 1, new(wire).str("seat0"))
	dispatch(t, d)
	if !caps.Contains(WlSeatCapabilityKeyboard) || caps.Contains(WlSeatCapabilityTouch) || seatName != "seat0" {
		t.Fatalf("caps %d name %q", caps, seatName)
	}

	keyboard := seat.GetKeyboard()
	pointer := seat.GetPointer()
	compositor := Bind[WlCompositor](reg, 8, WlCompositorInterface, 6)
	surface := compositor.CreateSurface()
	flush(t, d)
	s.expect(3, 1, new(wire).u32(4))
	s.expect(3, 0, new(wire).u32(5))
	s.expect(2, 0, new(wire).u32(8).str("wl_compositor").u32(6).u32(6))
	s.expect(6, 0, new(wire).u32(7))
	if surface.Version() != 6 || surface.Ptr() == 0 {
		t.Fatalf("surface %s version %d", surface, surface.Version())
	}

	var keymap []byte
	var keymapFormat WlKeyboardKeymapFormat
	var enterSurface *WlSurface
	var enterKeys []byte
	keyboard.SetListener(WlKeyboardListener{
		Keymap: func(format WlKeyboardKeymapFormat, fd int, size uint32) {
			keymapFormat = format
			f := os.NewFile(uintptr(fd), "keymap")
			defer f.Close()
			keymap = make([]byte, size)
			if _, err := f.ReadAt(keymap, 0); err != nil {
				t.Errorf("read keymap: %v", err)
			}
		},
		Enter: func(serial uint32, s *WlSurface, keys []byte) {
			enterSurface = s
			enterKeys = keys
		},
	})
	file, err := os.CreateTemp(t.TempDir(), "keymap")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("xkb_keymap {}"); err != nil {
		t.Fatal(err)
	}
	s.send(4, 0, new(wire).u32(uint32(WlKeyboardKeymapFormatXkbV1)).u32(13), int(file.Fd()))
	_ = file.Close()
	s.send(4, 1, new(wire).u32(11).u32(7).arr([]byte{30, 0, 0, 0, 31, 0, 0, 0}))
	dispatch(t, d)
	if keymapFormat != WlKeyboardKeymapFormatXkbV1 || string(keymap) != "xkb_keymap {}" {
		t.Fatalf("keymap %d %q", keymapFormat, keymap)
	}
	if enterSurface != surface || !bytes.Equal(enterKeys, []byte{30, 0, 0, 0, 31, 0, 0, 0}) {
		t.Fatalf("enter %v %v", enterSurface, enterKeys)
	}

	var mx, my float64
	var motionTime uint32
	pointer.SetListener(WlPointerListener{Motion: func(time uint32, x, y float64) {
		motionTime, mx, my = time, x, y
	}})
	s.send(5, 2, new(wire).u32(1234).i32(2688).i32(-320))
	dispatch(t, d)
	if motionTime != 1234 || mx != 10.5 || my != -1.25 {
		t.Fatalf("motion %d %v %v", motionTime, mx, my)
	}

	ddm := Bind[WlDataDeviceManager](reg, 9, WlDataDeviceManagerInterface, 3)
	device := ddm.GetDataDevice(seat)
	flush(t, d)
	s.expect(2, 0, new(wire).u32(9).str("wl_data_device_manager").u32(3).u32(8))
	s.expect(8, 1, new(wire).u32(9).u32(3))
	var offer *WlDataOffer
	var mimes []string
	selectionCalled := false
	var selection *WlDataOffer
	device.SetListener(WlDataDeviceListener{
		DataOffer: func(o *WlDataOffer) {
			offer = o
			o.SetListener(WlDataOfferListener{Offer: func(m string) { mimes = append(mimes, m) }})
		},
		Selection: func(o *WlDataOffer) {
			selectionCalled = true
			selection = o
		},
	})
	s.send(9, 0, new(wire).u32(0xff000000))
	s.send(0xff000000, 0, new(wire).str("text/plain"))
	s.send(9, 5, new(wire).u32(0xff000000))
	dispatch(t, d)
	if offer == nil || offer.ID() != 0xff000000 || len(mimes) != 1 || mimes[0] != "text/plain" || selection != offer {
		t.Fatalf("offer %v mimes %v selection %v", offer, mimes, selection)
	}
	selectionCalled = false
	s.send(9, 5, new(wire).u32(0))
	dispatch(t, d)
	if !selectionCalled || selection != nil {
		t.Fatalf("null selection: called %v value %v", selectionCalled, selection)
	}
	offer.Accept(5, gpuiutil.None[string]())
	offer.Accept(6, gpuiutil.Some("text/plain"))
	flush(t, d)
	s.expect(0xff000000, 0, new(wire).u32(5).u32(0))
	s.expect(0xff000000, 0, new(wire).u32(6).str("text/plain"))

	cb := display.Sync()
	var doneData uint32
	cb.SetListener(WlCallbackListener{Done: func(v uint32) { doneData = v }})
	flush(t, d)
	s.expect(1, 0, new(wire).u32(10))
	s.send(10, 0, new(wire).u32(77))
	s.send(1, 1, new(wire).u32(10))
	dispatch(t, d)
	if doneData != 77 || cb.IsAlive() {
		t.Fatalf("callback done %d alive %v", doneData, cb.IsAlive())
	}

	surface.Destroy()
	surface.Commit()
	if surface.IsAlive() {
		t.Fatal("surface alive after destroy")
	}
	dead := compositor.CreateRegion()
	dead.Add(0, 0, 1, 1)
	flush(t, d)
	s.expect(7, 0, nil)
	s.expect(6, 1, new(wire).u32(10))
	s.expect(10, 1, new(wire).i32(0).i32(0).i32(1).i32(1))

	pointer.SetListener(WlPointerListener{Frame: func() { panic("listener panic") }})
	s.send(5, 5, nil)
	func() {
		defer func() {
			if r := recover(); r != "listener panic" {
				t.Fatalf("recovered %v", r)
			}
		}()
		_, _ = d.Dispatch()
		t.Fatal("dispatch did not re-panic")
	}()

	s.send(1, 0, new(wire).u32(4).u32(3).str("boom"))
	_, err = d.Dispatch()
	var perr *wl.ProtocolError
	if !errors.As(err, &perr) || perr.Code != 3 || perr.Interface != "wl_keyboard" || perr.ID != 4 {
		t.Fatalf("protocol error = %v", err)
	}
}

func TestWire_InertAfterDestroy(t *testing.T) {
	d, s := newPair(t)
	reg := GetDisplay(d).GetRegistry()
	flush(t, d)
	s.expect(1, 1, new(wire).u32(2))
	compositor := Bind[WlCompositor](reg, 1, WlCompositorInterface, 4)
	flush(t, d)
	s.expect(2, 0, new(wire).u32(1).str("wl_compositor").u32(4).u32(3))
	compositor.Destroy()
	surface := compositor.CreateSurface()
	if surface == nil || surface.IsAlive() || surface.Ptr() != 0 {
		t.Fatalf("constructor on a dead proxy = %v", surface)
	}
	surface.Commit()
	GetDisplay(d).Sync()
	flush(t, d)
	s.expect(1, 0, new(wire).u32(4))
}
