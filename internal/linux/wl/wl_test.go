//go:build linux

package wl

import (
	"errors"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestLayout_CTables(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout checks are written for 64-bit targets")
	}
	var m cMessage
	if got := unsafe.Sizeof(m); got != 24 {
		t.Fatalf("sizeof(wl_message) = %d, want 24", got)
	}
	if unsafe.Offsetof(m.signature) != 8 || unsafe.Offsetof(m.types) != 16 {
		t.Fatalf("wl_message offsets = %d, %d", unsafe.Offsetof(m.signature), unsafe.Offsetof(m.types))
	}
	var i cInterface
	if got := unsafe.Sizeof(i); got != 40 {
		t.Fatalf("sizeof(wl_interface) = %d, want 40", got)
	}
	offs := []uintptr{unsafe.Offsetof(i.name), unsafe.Offsetof(i.version), unsafe.Offsetof(i.methodCount), unsafe.Offsetof(i.methods), unsafe.Offsetof(i.eventCount), unsafe.Offsetof(i.events)}
	want := []uintptr{0, 8, 12, 16, 24, 32}
	for k := range offs {
		if offs[k] != want[k] {
			t.Fatalf("wl_interface field %d at %d, want %d", k, offs[k], want[k])
		}
	}
	var a cArray
	if got := unsafe.Sizeof(a); got != 24 || unsafe.Offsetof(a.data) != 16 {
		t.Fatalf("wl_array size %d data offset %d", got, unsafe.Offsetof(a.data))
	}
}

func TestLayout_MatchesLibwayland(t *testing.T) {
	if err := Load(); err != nil {
		t.Skipf("libwayland-client not available: %v", err)
	}
	callback := &Interface{Name: "wl_callback", Version: 1}
	callback.Events = []Message{{Name: "done", Signature: "u", Since: 1, Destructor: true}}
	ours := callback.C()
	theirs, err := LibrarySymbol("wl_callback_interface")
	if err != nil {
		t.Skip(err)
	}
	if got, want := DescribeC(ours), DescribeC(theirs); got != want {
		t.Fatalf("table mismatch:\nours   %s\ntheirs %s", got, want)
	}
}

func TestSignature_Parse(t *testing.T) {
	codes := parseSignature("2?osun")
	want := []argCode{{'o', true}, {'s', false}, {'u', false}, {'n', false}}
	if len(codes) != len(want) {
		t.Fatalf("got %v", codes)
	}
	for k := range want {
		if codes[k] != want[k] {
			t.Fatalf("code %d = %v, want %v", k, codes[k], want[k])
		}
	}
	m := Message{Signature: "?s", Since: 3}
	if m.CSignature() != "3?s" {
		t.Fatalf("CSignature = %q", m.CSignature())
	}
	m.Since = 1
	if m.CSignature() != "?s" {
		t.Fatalf("CSignature = %q", m.CSignature())
	}
}

func TestFixed_Conversions(t *testing.T) {
	cases := []struct {
		f    float64
		want Fixed
	}{{10.5, 2688}, {-1.25, -320}, {0, 0}, {1.0 / 512, 0}}
	for _, c := range cases {
		if got := FixedFromFloat64(c.f); got != c.want {
			t.Fatalf("FixedFromFloat64(%v) = %d, want %d", c.f, got, c.want)
		}
	}
	if Fixed(2688).Float64() != 10.5 || Fixed(-320).Float64() != -1.25 {
		t.Fatal("Float64 conversion")
	}
	if FixedFromInt(-3) != -768 || Fixed(-769).Int() != -3 {
		t.Fatal("int conversion")
	}
}

type testObject struct {
	Proxy
	seen []uint32
	fn   func(op uint32)
}

func (o *testObject) Dispatch(op uint32, a Args) {
	o.seen = append(o.seen, op)
	if o.fn != nil {
		o.fn(op)
	}
}

func registerTestObject(t *testing.T, ptr uintptr, iface *Interface, d *Display) *testObject {
	t.Helper()
	o := &testObject{}
	o.ptr = ptr
	o.iface = iface
	o.display = d
	o.version = 1
	o.alive = true
	registryMu.Lock()
	registry[ptr] = o
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, ptr)
		registryMu.Unlock()
	})
	return o
}

func testPipe(t *testing.T) (int, int) {
	t.Helper()
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Close(fds[0])
		if !fdClosed(fds[1]) {
			_ = syscall.Close(fds[1])
		}
	})
	return fds[0], fds[1]
}

func fdClosed(fd int) bool {
	_, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
	return e == syscall.EBADF
}

func TestDispatch_ViolationDiscardsAndKillsConnection(t *testing.T) {
	_, w := testPipe(t)
	iface := &Interface{Name: "test_iface", Version: 1, Events: []Message{{Name: "ev", Signature: "u"}}}
	d := &Display{}
	obj := registerTestObject(t, 0x1000, iface, d)
	sig := cString("uh")
	var pin runtime.Pinner
	defer pin.Unpin()
	pin.Pin(&sig[0])
	msg := cMessage{signature: &sig[0]}
	pin.Pin(&msg)
	args := []uint64{7, uint64(uint32(int32(w)))}
	pin.Pin(&args[0])
	ret := dispatcher(0, 0x1000, 5, uintptr(unsafe.Pointer(&msg)), uintptr(unsafe.Pointer(&args[0])))
	if ret != dispatchFailed {
		t.Fatalf("dispatcher returned %d", ret)
	}
	if !fdClosed(w) {
		t.Fatal("fd argument of a dropped event was not closed")
	}
	if !errors.Is(d.violation, ErrProtocolViolation) {
		t.Fatalf("violation = %v", d.violation)
	}
	if _, err := d.call(func() int32 { t.Fatal("libwayland called on a dead connection"); return 0 }); !errors.Is(err, ErrProtocolViolation) {
		t.Fatalf("call error = %v", err)
	}
	args2 := []uint64{1}
	pin.Pin(&args2[0])
	dispatcher(0, 0x1000, 0, 0, uintptr(unsafe.Pointer(&args2[0])))
	if len(obj.seen) != 0 {
		t.Fatalf("events dispatched after the violation: %v", obj.seen)
	}
}

func TestDispatch_UnknownTargetClosesFDs(t *testing.T) {
	_, w := testPipe(t)
	sig := cString("h")
	var pin runtime.Pinner
	defer pin.Unpin()
	pin.Pin(&sig[0])
	msg := cMessage{signature: &sig[0]}
	pin.Pin(&msg)
	args := []uint64{uint64(uint32(int32(w)))}
	pin.Pin(&args[0])
	if ret := dispatcher(0, 0x2000, 0, uintptr(unsafe.Pointer(&msg)), uintptr(unsafe.Pointer(&args[0]))); ret != 0 {
		t.Fatalf("dispatcher returned %d", ret)
	}
	if !fdClosed(w) {
		t.Fatal("fd argument of an unroutable event was not closed")
	}
}

func TestDispatch_ListenerPanicKeepsDispatching(t *testing.T) {
	iface := &Interface{Name: "test_iface", Version: 1, Events: []Message{{Name: "a", Signature: "u"}, {Name: "b", Signature: "u"}}}
	d := &Display{}
	obj := registerTestObject(t, 0x3000, iface, d)
	obj.fn = func(op uint32) {
		if op == 0 {
			panic("first")
		}
		panic("second")
	}
	args := []uint64{1}
	var pin runtime.Pinner
	defer pin.Unpin()
	pin.Pin(&args[0])
	dispatcher(0, 0x3000, 0, 0, uintptr(unsafe.Pointer(&args[0])))
	dispatcher(0, 0x3000, 1, 0, uintptr(unsafe.Pointer(&args[0])))
	if len(obj.seen) != 2 {
		t.Fatalf("seen = %v", obj.seen)
	}
	defer func() {
		if r := recover(); r != "first" {
			t.Fatalf("rethrown %v, want first", r)
		}
	}()
	rethrow()
}

func TestProxy_RequestNewerThanVersionPanics(t *testing.T) {
	iface := &Interface{Name: "test_iface", Version: 3, Methods: []Message{{Name: "late", Since: 3}}}
	p := &Proxy{iface: iface, version: 2, id: 9, alive: true}
	defer func() {
		r := recover()
		if s, _ := r.(string); !strings.Contains(s, "test_iface@9.late requires version 3") {
			t.Fatalf("panic = %v", r)
		}
	}()
	p.Marshal(0)
}

func TestProxy_StringWithNulPanics(t *testing.T) {
	iface := &Interface{Name: "test_iface", Version: 1, Methods: []Message{{Name: "set_title", Signature: "s", Since: 1}}}
	p := &Proxy{iface: iface, version: 1, id: 4, alive: true}
	defer func() {
		r := recover()
		if s, _ := r.(string); !strings.Contains(s, "nul byte") {
			t.Fatalf("panic = %v", r)
		}
	}()
	p.Marshal(0, String("a\x00b"))
}

func TestCString_RejectsNul(t *testing.T) {
	if _, err := cStringChecked("ab\x00c"); err == nil {
		t.Fatal("interior NUL accepted")
	}
	b, err := cStringChecked("abc")
	if err != nil || string(b) != "abc\x00" {
		t.Fatalf("cStringChecked = %q, %v", b, err)
	}
	if _, err := Connect("wayland\x00-0"); err == nil && Load() == nil {
		t.Fatal("Connect accepted a name with a NUL byte")
	}
}

func TestEventQueue_DestroyAfterDisconnectIsNoop(t *testing.T) {
	d := &Display{closed: true}
	q := &EventQueue{ptr: 0x4000, display: d}
	q.Destroy()
	if q.ptr != 0x4000 {
		t.Fatal("Destroy touched a queue of a disconnected display")
	}
}
