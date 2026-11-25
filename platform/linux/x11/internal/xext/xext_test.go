//go:build linux

package xext

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/megascan/gpui-go/gpuiutil"
	"golang.org/x/sys/unix"
)

const (
	testXinputOpcode = 131
	testXkbOpcode    = 135
	testXkbEvent     = 85
	testSyncOpcode   = 134
	testDri3Opcode   = 149
)

type fakeServer struct {
	t    *testing.T
	conn *net.UnixConn
}

func TestMain(m *testing.M) {
	xgb.Logger = log.New(io.Discard, "", 0)
	xgb.NewEventFuncs[testXkbEvent] = xgb.NewExtEventFuncs[XkbExtensionName][0]
	os.Exit(m.Run())
}

func socketPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	toConn := func(fd int) *net.UnixConn {
		f := os.NewFile(uintptr(fd), "sock")
		c, err := net.FileConn(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		return c.(*net.UnixConn)
	}
	return toConn(fds[0]), toConn(fds[1])
}

func newTestConn(t *testing.T) (*Conn, *fakeServer) {
	t.Helper()
	cli, srv := socketPair(t)
	done := make(chan error, 1)
	go func() {
		hello := make([]byte, 12+20+16)
		if _, err := io.ReadFull(srv, hello); err != nil {
			done <- err
			return
		}
		setup := make([]byte, 24)
		setup[0] = 1
		xgb.Put16(setup[2:], 11)
		xgb.Put16(setup[6:], 4)
		xgb.Put32(setup[12:], 0x00400000)
		xgb.Put32(setup[16:], 0x001fffff)
		_, err := srv.Write(setup)
		done <- err
	}()
	w := NewConn(cli)
	xc, err := xgb.NewConnNetWithCookieHex(w, "00000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	w.X = xc
	xc.ExtLock.Lock()
	xc.Extensions[XinputExtensionName] = testXinputOpcode
	xc.Extensions[XkbExtensionName] = testXkbOpcode
	xc.Extensions[SyncExtensionName] = testSyncOpcode
	xc.Extensions[Dri3ExtensionName] = testDri3Opcode
	xc.ExtLock.Unlock()
	t.Cleanup(func() {
		xc.Close()
		srv.Close()
	})
	return w, &fakeServer{t: t, conn: srv}
}

func (s *fakeServer) expect(want []byte) {
	s.t.Helper()
	s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	got := make([]byte, len(want))
	if _, err := io.ReadFull(s.conn, got); err != nil {
		s.t.Fatalf("reading request: %v", err)
	}
	if !bytes.Equal(got, want) {
		s.t.Fatalf("request bytes\n got %v\nwant %v", got, want)
	}
}

func (s *fakeServer) send(b []byte) {
	s.t.Helper()
	if _, err := s.conn.Write(b); err != nil {
		s.t.Fatal(err)
	}
}

func reply(seq uint16, body []byte) []byte {
	if len(body) < 32 {
		body = append(body, make([]byte, 32-len(body))...)
	}
	body = append(body, make([]byte, xgb.Pad(len(body))-len(body))...)
	body[0] = 1
	xgb.Put16(body[2:], seq)
	xgb.Put32(body[4:], uint32((len(body)-32)/4))
	return body
}

func le16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }
func le32(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} }

func cat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

func motionFixture(seq uint16) []byte {
	return cat(
		[]byte{35, testXinputOpcode}, le16(seq), le32(18), le16(XinputMotion),
		le16(2), le32(1000), le32(0),
		le32(0x100), le32(0x200), le32(0x300),
		le32(10<<16), le32(20<<16), le32(3<<16|0x8000), le32(4<<16),
		le16(1), le16(1), le16(11), []byte{0, 0},
		le32(0),
		le32(1), le32(2), le32(4), le32(7),
		[]byte{0, 0, 1, 1},
		le32(0),
		le32(0b1100),
		le32(5), le32(0x80000000),
		le32(0xffffffff), le32(0),
	)
}

func TestConn_FramesGenericEvents(t *testing.T) {
	c, srv := newTestConn(t)
	cookie := XinputXIQueryVersion(c.X, 2, 4)
	srv.expect([]byte{testXinputOpcode, 47, 2, 0, 2, 0, 4, 0})
	key := make([]byte, 32)
	key[0] = xproto.KeyPress
	key[1] = 38
	xgb.Put16(key[2:], 1)
	srv.send(cat(motionFixture(1), reply(1, cat([]byte{0, 0, 0, 0, 0, 0, 0, 0}, le16(2), le16(4))), key))

	r, err := cookie.Reply()
	if err != nil {
		t.Fatal(err)
	}
	if r.MajorVersion != 2 || r.MinorVersion != 4 {
		t.Fatalf("version %d.%d", r.MajorVersion, r.MinorVersion)
	}
	ev, xerr := c.WaitForEvent()
	if xerr != nil {
		t.Fatal(xerr)
	}
	m, ok := ev.(XinputMotionEvent)
	if !ok {
		t.Fatalf("got %T", ev)
	}
	want := XinputMotionEvent{
		Extension: testXinputOpcode, Sequence: 1, EventType: XinputMotion, Deviceid: 2, Time: 1000,
		Root: 0x100, Event: 0x200, Child: 0x300,
		RootX: 10 << 16, RootY: 20 << 16, EventX: 3<<16 | 0x8000, EventY: 4 << 16,
		Sourceid: 11, Mods: XinputModifierInfo{1, 2, 4, 7}, Group: XinputGroupInfo{0, 0, 1, 1},
		ButtonMask: []uint32{0}, ValuatorMask: []uint32{0b1100},
		Axisvalues: []XinputFp3232{{5, 0x80000000}, {-1, 0}},
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("motion\n got %+v\nwant %+v", m, want)
	}
	if !bytes.Equal(m.Bytes(), motionFixture(1)) {
		t.Fatalf("motion re-encode\n got %v\nwant %v", m.Bytes(), motionFixture(1))
	}
	ev, xerr = c.WaitForEvent()
	if xerr != nil {
		t.Fatal(xerr)
	}
	if k, ok := ev.(xproto.KeyPressEvent); !ok || k.Detail != 38 {
		t.Fatalf("got %#v", ev)
	}
}

func TestXinput_SelectEventsRequest(t *testing.T) {
	c, srv := newTestConn(t)
	mask := XinputXIEventMaskMotion | XinputXIEventMaskButtonPress | XinputXIEventMaskButtonRelease | XinputXIEventMaskEnter | XinputXIEventMaskLeave |
		XinputXIEventMask(1<<XinputGesturePinchBegin) | XinputXIEventMask(1<<XinputGesturePinchUpdate) | XinputXIEventMask(1<<XinputGesturePinchEnd)
	XinputXISelectEvents(c.X, 0x01200001, []XinputEventMask{
		{Deviceid: XinputDeviceAllMaster, Mask: []XinputXIEventMask{mask}},
		{Deviceid: XinputDeviceAll, Mask: []XinputXIEventMask{XinputXIEventMaskHierarchy | XinputXIEventMaskDeviceChanged}},
	})
	srv.expect(cat(
		[]byte{testXinputOpcode, 46, 7, 0},
		le32(0x01200001), le16(2), []byte{0, 0},
		le16(1), le16(1), le32(uint32(mask)),
		le16(0), le16(1), le32(0x802),
	))
	if uint32(mask) != 0x380001f0 {
		t.Fatalf("mask %#x", uint32(mask))
	}
}

func TestXinput_QueryDeviceReply(t *testing.T) {
	c, srv := newTestConn(t)
	cookie := XinputXIQueryDevice(c.X, XinputDeviceAll)
	srv.expect([]byte{testXinputOpcode, 48, 2, 0, 0, 0, 0, 0})
	scroll := cat(le16(3), le16(6), le16(9), le16(3), le16(uint16(XinputScrollTypeVertical)), []byte{0, 0}, le32(0), le32(120), le32(0))
	valuator := cat(le16(2), le16(11), le16(9), le16(2), le32(77), le32(0), le32(0), le32(100), le32(0), le32(5), le32(1<<31), le32(1), []byte{1, 0, 0, 0})
	unknown := cat(le16(42), le16(2), le16(9), []byte{0xaa, 0xbb})
	info := cat(le16(9), le16(uint16(XinputDeviceTypeSlavePointer)), le16(2), le16(3), le16(5), []byte{1, 0}, []byte("mouse\x00\x00\x00"), scroll, valuator, unknown)
	srv.send(reply(1, cat([]byte{0, 0, 0, 0, 0, 0, 0, 0}, le16(1), make([]byte, 22), info)))
	r, err := cookie.Reply()
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Infos) != 1 {
		t.Fatalf("infos %d", len(r.Infos))
	}
	got := r.Infos[0]
	want := XinputXIDeviceInfo{
		Deviceid: 9, Type: XinputDeviceTypeSlavePointer, Attachment: 2, Enabled: true, Name: []byte("mouse"),
		Classes: []XinputDeviceClass{
			{Type: XinputDeviceClassTypeScroll, Sourceid: 9, Data: XinputDeviceClassDataScroll{Number: 3, ScrollType: XinputScrollTypeVertical, Increment: XinputFp3232{120, 0}}},
			{Type: XinputDeviceClassTypeValuator, Sourceid: 9, Data: XinputDeviceClassDataValuator{Number: 2, Label: 77, Max: XinputFp3232{100, 0}, Value: XinputFp3232{5, 1 << 31}, Resolution: 1, Mode: 1}},
			{Type: 42, Sourceid: 9, Data: XinputDeviceClassDataUnknown{Data: []byte{0xaa, 0xbb}}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("info\n got %+v\nwant %+v", got, want)
	}
	if s, ok := got.Classes[0].AsScroll(); !ok || s.Number != 3 {
		t.Fatal("AsScroll")
	}
	var w writer
	for _, dc := range got.Classes {
		writeDeviceClass(&w, dc)
	}
	if !bytes.Equal(w.buf, cat(scroll, valuator, unknown)) {
		t.Fatalf("class re-encode\n got %v\nwant %v", w.buf, cat(scroll, valuator, unknown))
	}
}

func TestXinput_EventRoundTrips(t *testing.T) {
	enter := cat(
		[]byte{35, testXinputOpcode}, le16(7), le32(11), le16(XinputLeave),
		le16(2), le32(55), le16(11), []byte{byte(XinputNotifyModeNormal), byte(XinputNotifyDetailNonlinear)},
		le32(1), le32(2), le32(3),
		le32(1<<16), le32(2<<16), le32(3<<16), le32(4<<16),
		[]byte{1, 0}, le16(1),
		le32(1), le32(0), le32(0), le32(1),
		[]byte{0, 0, 0, 0},
		le32(0x2),
	)
	hierarchy := cat(
		[]byte{35, testXinputOpcode}, le16(8), le32(3), le16(XinputHierarchy),
		le16(0), le32(66), le32(XinputHierarchyMaskSlaveAdded), le16(1), make([]byte, 10),
		le16(12), le16(2), []byte{byte(XinputDeviceTypeSlavePointer), 1, 0, 0}, le32(XinputHierarchyMaskSlaveAdded),
	)
	changed := cat(
		[]byte{35, testXinputOpcode}, le16(9), le32(6), le16(XinputDeviceChanged),
		le16(9), le32(77), le16(1), le16(9), []byte{XinputChangeReasonDeviceChange}, make([]byte, 11),
		le16(3), le16(6), le16(9), le16(4), le16(uint16(XinputScrollTypeHorizontal)), []byte{0, 0}, le32(XinputScrollFlagsPreferred), le32(15), le32(0),
	)
	pinch := cat(
		[]byte{35, testXinputOpcode}, le16(10), le32(17), le16(XinputGesturePinchUpdate),
		le16(2), le32(88), le32(2), le32(1), le32(2), le32(0),
		le32(1), le32(2), le32(3), le32(4), le32(5), le32(6), le32(7), le32(8), le32(0x18000), le32(0),
		le16(11), []byte{0, 0},
		le32(0), le32(0), le32(0), le32(0), []byte{0, 0, 0, 0}, le32(XinputGesturePinchEventFlagsCancelled),
	)
	c, _ := newTestConn(t)
	cases := []struct {
		raw  []byte
		want reflect.Type
	}{
		{enter, reflect.TypeFor[XinputLeaveEvent]()},
		{hierarchy, reflect.TypeFor[XinputHierarchyEvent]()},
		{changed, reflect.TypeFor[XinputDeviceChangedEvent]()},
		{pinch, reflect.TypeFor[XinputGesturePinchUpdateEvent]()},
	}
	for _, tc := range cases {
		ev := DecodeGenericEvent(c.X, tc.raw)
		if reflect.TypeOf(ev) != tc.want {
			t.Fatalf("got %T want %v", ev, tc.want)
		}
		if !bytes.Equal(ev.Bytes(), tc.raw) {
			t.Fatalf("%T re-encode\n got %v\nwant %v", ev, ev.Bytes(), tc.raw)
		}
	}
	l := DecodeGenericEvent(c.X, enter).(XinputLeaveEvent)
	if l.Mode != XinputNotifyModeNormal || l.Event != 2 || !l.SameScreen || l.Buttons[0] != 2 {
		t.Fatalf("leave %+v", l)
	}
	p := DecodeGenericEvent(c.X, pinch).(XinputGesturePinchUpdateEvent)
	if p.Scale != 0x18000 || p.Sourceid != 11 || p.Flags != 1 {
		t.Fatalf("pinch %+v", p)
	}
	h := DecodeGenericEvent(c.X, hierarchy).(XinputHierarchyEvent)
	if len(h.Infos) != 1 || h.Infos[0].Deviceid != 12 || !h.Infos[0].Enabled {
		t.Fatalf("hierarchy %+v", h)
	}
	other := cat([]byte{35, 200}, le16(1), le32(0), le16(1), make([]byte, 22))
	if _, ok := DecodeGenericEvent(c.X, other).(GenericEvent); !ok {
		t.Fatal("unknown extension should decode to GenericEvent")
	}
}

func TestXkb_UseExtensionAndSelectEvents(t *testing.T) {
	c, srv := newTestConn(t)
	cookie := XkbUseExtension(c.X, 1, 0)
	srv.expect([]byte{testXkbOpcode, 0, 2, 0, 1, 0, 0, 0})
	srv.send(reply(1, cat([]byte{0, 1, 0, 0, 0, 0, 0, 0}, le16(1), le16(0))))
	r, err := cookie.Reply()
	if err != nil {
		t.Fatal(err)
	}
	if !r.Supported || r.ServerMajor != 1 || r.ServerMinor != 0 {
		t.Fatalf("%+v", r)
	}

	events := XkbEventTypeStateNotify | XkbEventTypeMapNotify | XkbEventTypeNewKeyboardNotify
	parts := XkbMapPartKeyTypes | XkbMapPartKeySyms | XkbMapPartModifierMap | XkbMapPartExplicitComponents |
		XkbMapPartKeyActions | XkbMapPartKeyBehaviors | XkbMapPartVirtualMods | XkbMapPartVirtualModMap
	XkbSelectEvents(c.X, XkbIDUseCoreKbd, 0, events, parts, parts, XkbSelectEventsAux{})
	srv.expect([]byte{testXkbOpcode, 1, 4, 0, 0, 1, 7, 0, 0, 0, 7, 0, 0xff, 0, 0xff, 0})

	XkbSelectEvents(c.X, XkbIDUseCoreKbd, 0, 0, 0, 0, XkbSelectEventsAux{
		StateNotify:     gpuiutil.Some(XkbSelectEventsPair16{Affect: 0x3fff, Details: 0x1}),
		CompatMapNotify: gpuiutil.Some(XkbSelectEventsPair8{Affect: 3, Details: 1}),
	})
	srv.expect([]byte{testXkbOpcode, 1, 6, 0, 0, 1, 0x84, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0x3f, 1, 0, 3, 1, 0, 0})
}

func TestXkb_Events(t *testing.T) {
	c, srv := newTestConn(t)
	state := cat([]byte{testXkbEvent, XkbStateNotify}, le16(0), le32(123),
		[]byte{3, 0x5, 0x1, 0x0, 0x4, 1}, le16(1), le16(0), []byte{2, 0x4, 0, 0, 0x5, 0x5}, le16(0x100), le16(0x3), []byte{38, 2, 0, 0})
	mapn := cat([]byte{testXkbEvent, XkbMapNotify}, le16(0), le32(124), []byte{3, 0}, le16(0xff), []byte{8, 255, 0, 10, 8, 248, 8, 248, 8, 248, 8, 248, 8, 248, 8, 248}, le16(0xffff), []byte{0, 0})
	newkb := cat([]byte{testXkbEvent, XkbNewKeyboardNotify}, le16(0), le32(125), []byte{3, 3, 8, 255, 8, 255, testXkbOpcode, 9}, le16(1), make([]byte, 14))
	srv.send(cat(state, mapn, newkb))
	for _, want := range [][]byte{state, mapn, newkb} {
		ev, xerr := c.WaitForEvent()
		if xerr != nil {
			t.Fatal(xerr)
		}
		if !bytes.Equal(ev.Bytes(), want) {
			t.Fatalf("%T re-encode\n got %v\nwant %v", ev, ev.Bytes(), want)
		}
		if s, ok := ev.(XkbStateNotifyEvent); ok {
			if s.BaseMods != 1 || s.LatchedMods != 0 || s.LockedMods != 4 || s.Group != 1 || s.BaseGroup != 1 || s.LockedGroup != 2 || s.Keycode != 38 {
				t.Fatalf("state %+v", s)
			}
		}
	}
}

func TestSync_Requests(t *testing.T) {
	c, srv := newTestConn(t)
	cookie := SyncInitialize(c.X, 3, 1)
	srv.expect([]byte{testSyncOpcode, 0, 2, 0, 3, 1, 0, 0})
	srv.send(reply(1, []byte{0, 0, 0, 0, 0, 0, 0, 0, 3, 1}))
	r, err := cookie.Reply()
	if err != nil {
		t.Fatal(err)
	}
	if r.MajorVersion != 3 || r.MinorVersion != 1 {
		t.Fatalf("%+v", r)
	}
	SyncCreateCounter(c.X, 0x00400005, SyncInt64{Hi: 0, Lo: 0})
	srv.expect(cat([]byte{testSyncOpcode, 2, 4, 0}, le32(0x00400005), le32(0), le32(0)))
	SyncSetCounter(c.X, 0x00400005, SyncInt64{Hi: -1, Lo: 42})
	srv.expect(cat([]byte{testSyncOpcode, 3, 4, 0}, le32(0x00400005), le32(0xffffffff), le32(42)))
	SyncDestroyCounter(c.X, 0x00400005)
	srv.expect(cat([]byte{testSyncOpcode, 6, 2, 0}, le32(0x00400005)))
}

func TestDri3_OpenReceivesFd(t *testing.T) {
	c, srv := newTestConn(t)
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cookie := Dri3Open(c, 0x123, 0)
	srv.expect(cat([]byte{testDri3Opcode, 1, 3, 0}, le32(0x123), le32(0)))
	rep := reply(1, nil)
	rep[1] = 1
	if _, _, err := srv.conn.WriteMsgUnix(rep, unix.UnixRights(int(f.Fd())), nil); err != nil {
		t.Fatal(err)
	}
	r, err := cookie.Reply()
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(r.DeviceFd)
	var got, want unix.Stat_t
	if err := unix.Fstat(r.DeviceFd, &got); err != nil {
		t.Fatal(err)
	}
	if err := unix.Fstat(int(f.Fd()), &want); err != nil {
		t.Fatal(err)
	}
	if got.Rdev != want.Rdev || got.Ino != want.Ino {
		t.Fatalf("fd mismatch: %+v vs %+v", got, want)
	}
	flags, err := unix.FcntlInt(uintptr(r.DeviceFd), unix.F_GETFD, 0)
	if err != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatalf("fd not close-on-exec: %d %v", flags, err)
	}
}

func TestConn_ParseDisplayAndCookie(t *testing.T) {
	d, err := ParseDisplay(":1.2")
	if err != nil || d.Number != 1 || d.Screen != 2 || d.Display != "1" || d.Host != "" {
		t.Fatalf("%+v %v", d, err)
	}
	d, err = ParseDisplay("tcp/example:3")
	if err != nil || d.Protocol != "tcp" || d.Host != "example" || d.Number != 3 {
		t.Fatalf("%+v %v", d, err)
	}
	if _, err := ParseDisplay("nocolon"); err == nil {
		t.Fatal("expected error")
	}
	entry := func(family uint16, addr, disp, name string, data []byte) []byte {
		be := func(s []byte) []byte { return append([]byte{byte(len(s) >> 8), byte(len(s))}, s...) }
		return cat([]byte{byte(family >> 8), byte(family)}, be([]byte(addr)), be([]byte(disp)), be([]byte(name)), be(data))
	}
	cookie := bytes.Repeat([]byte{7}, 16)
	file := cat(
		entry(256, "host", "0", "MIT-MAGIC-COOKIE-1", bytes.Repeat([]byte{1}, 16)),
		entry(256, "other", "1", "MIT-MAGIC-COOKIE-1", bytes.Repeat([]byte{2}, 16)),
		entry(256, "host", "1", "MIT-MAGIC-COOKIE-1", cookie),
	)
	got, ok := FindCookie(file, "host", "1")
	if !ok || !bytes.Equal(got, cookie) {
		t.Fatalf("%v %v", got, ok)
	}
	if _, ok := FindCookie(file, "nobody", "5"); ok {
		t.Fatal("unexpected match")
	}
}
