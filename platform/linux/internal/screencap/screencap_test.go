//go:build linux

package screencap

import (
	"bufio"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	godbus "github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/pw"
)

const testBusConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=%DIR%</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
`

func startTestBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not found")
	}
	dir, err := os.MkdirTemp("", "gpuiscap")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(strings.ReplaceAll(testBusConfig, "%DIR%", dir)), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(daemon, "--config-file="+config, "--nofork", "--print-address=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(address)
}

func connectTestBus(t *testing.T, address string) *godbus.Conn {
	t.Helper()
	conn, err := godbus.Connect(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

type fakePortal struct {
	conn        *godbus.Conn
	cursorModes uint32
	selectCode  uint32
	mu          sync.Mutex
	calls       []string
	selectArgs  map[string]godbus.Variant
	nodeID      uint32
}

func (p *fakePortal) respond(path godbus.ObjectPath, code uint32, results map[string]godbus.Variant) {
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = p.conn.Emit(path, portalRequest+".Response", code, results)
	}()
}

func (p *fakePortal) record(name string) {
	p.mu.Lock()
	p.calls = append(p.calls, name)
	p.mu.Unlock()
}

func (p *fakePortal) CreateSession(options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
	p.record("CreateSession")
	token := options["handle_token"].Value().(string)
	path := godbus.ObjectPath("/org/freedesktop/portal/desktop/request/test/" + token + "_create")
	p.respond(path, 0, map[string]godbus.Variant{"session_handle": godbus.MakeVariant("/org/freedesktop/portal/desktop/session/test/" + token)})
	return path, nil
}

func (p *fakePortal) SelectSources(session godbus.ObjectPath, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
	p.record("SelectSources:" + string(session))
	p.mu.Lock()
	p.selectArgs = options
	p.mu.Unlock()
	path := godbus.ObjectPath("/org/freedesktop/portal/desktop/request/test/select")
	p.respond(path, p.selectCode, map[string]godbus.Variant{})
	return path, nil
}

func (p *fakePortal) Start(session godbus.ObjectPath, parent string, options map[string]godbus.Variant) (godbus.ObjectPath, *godbus.Error) {
	p.record("Start:" + parent)
	path := godbus.ObjectPath("/org/freedesktop/portal/desktop/request/test/start")
	streams := []struct {
		NodeID uint32
		Props  map[string]godbus.Variant
	}{{p.nodeID, map[string]godbus.Variant{"source_type": godbus.MakeVariant(uint32(1))}}}
	p.respond(path, 0, map[string]godbus.Variant{"streams": godbus.MakeVariant(streams)})
	return path, nil
}

type fakeProps struct{ p *fakePortal }

func (f fakeProps) Get(iface, name string) (godbus.Variant, *godbus.Error) {
	switch name {
	case "AvailableSourceTypes":
		return godbus.MakeVariant(uint32(3)), nil
	case "AvailableCursorModes":
		return godbus.MakeVariant(f.p.cursorModes), nil
	case "version":
		return godbus.MakeVariant(uint32(4)), nil
	}
	return godbus.Variant{}, godbus.MakeFailedError(errors.New("no such property"))
}

func startFakePortal(t *testing.T, p *fakePortal) *godbus.Conn {
	t.Helper()
	address := startTestBus(t)
	server := connectTestBus(t, address)
	p.conn = server
	if err := server.Export(p, portalDesktopPath, portalScreenCast); err != nil {
		t.Fatal(err)
	}
	if err := server.Export(fakeProps{p}, portalDesktopPath, "org.freedesktop.DBus.Properties"); err != nil {
		t.Fatal(err)
	}
	_ = server.Export(introspect.Introspectable(""), portalDesktopPath, "org.freedesktop.DBus.Introspectable")
	if reply, err := server.RequestName(portalDesktopService, godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("RequestName = %v, %v", reply, err)
	}
	client := connectTestBus(t, address)
	return client
}

func TestPortal_CreateStreamRunsTheScreenCastHandshake(t *testing.T) {
	p := &fakePortal{cursorModes: 3, nodeID: 42}
	client := startFakePortal(t, p)
	portal, err := NewScreenCastPortal(client).ShowCursor(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(portal.token, "scap_") {
		t.Fatalf("token = %q", portal.token)
	}
	stream, err := portal.CreateStream()
	if err != nil {
		t.Fatal(err)
	}
	if stream.PwNodeID() != 42 {
		t.Fatalf("node id = %d", stream.PwNodeID())
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	want := []string{"CreateSession", "SelectSources:/org/freedesktop/portal/desktop/session/test/" + portal.token, "Start:"}
	if strings.Join(p.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v", p.calls)
	}
	if p.selectArgs["cursor_mode"].Value() != uint32(2) || p.selectArgs["types"].Value() != uint32(3) || p.selectArgs["multiple"].Value() != false {
		t.Fatalf("SelectSources options = %v", p.selectArgs)
	}
}

func TestPortal_CancelledSelectionAndCursorModes(t *testing.T) {
	p := &fakePortal{cursorModes: 1, selectCode: 1}
	client := startFakePortal(t, p)
	if _, err := NewScreenCastPortal(client).ShowCursor(true); err == nil || err.Error() != "Unsupported cursor mode" {
		t.Fatalf("ShowCursor(true) err = %v", err)
	}
	portal, err := NewScreenCastPortal(client).ShowCursor(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := portal.CreateStream(); err == nil || err.Error() != "User cancelled the interaction" {
		t.Fatalf("CreateStream err = %v", err)
	}
}

func TestPortal_MatchResponse(t *testing.T) {
	if matchResponse(0) != nil {
		t.Fatal("0 is success")
	}
	if err := matchResponse(2); err == nil || err.Error() != "The user interaction was ended in some other way" {
		t.Fatalf("err = %v", err)
	}
	if err := matchResponse(3); err == nil || err.Error() != "Unknown portal response code 3" {
		t.Fatalf("err = %v", err)
	}
}

func TestQueue_DropsOldestWhenFull(t *testing.T) {
	q := newFrameQueue()
	for i := range frameQueueCap + 3 {
		q.send(gpui.ScapFrameBGRx{ScapPixels: gpui.ScapPixels{Width: int32(i)}}, nil)
	}
	q.close()
	res, err := q.recv()
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := res.frame.Size(); w != 3 {
		t.Fatalf("oldest kept width = %d", w)
	}
}

func TestWayland_CopyFrameDataHonoursChunk(t *testing.T) {
	src := make([]byte, 64)
	for i := range src {
		src[i] = byte(i)
	}
	chunk := pw.SpaChunk{Offset: 4, Size: 40, Stride: 12}
	data := pw.SpaData{MaxSize: uint32(len(src)), Data: unsafe.Pointer(&src[0]), Chunk: &chunk}
	out, err := copyFrameData(&data, 2, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{4, 5, 6, 7, 8, 9, 10, 11, 16, 17, 18, 19, 20, 21, 22, 23, 28, 29, 30, 31, 32, 33, 34, 35}
	if string(out) != string(want) {
		t.Fatalf("out = %v", out)
	}
	if _, err := copyFrameData(&data, 2, 4, 4); err == nil {
		t.Fatal("frame larger than chunk must fail")
	}
}

func TestX11_DrawCursorShortImageIsSkipped(t *testing.T) {
	if imageFitsBGRx(make([]byte, 31), 4, 2) || !imageFitsBGRx(make([]byte, 32), 4, 2) {
		t.Fatal("imageFitsBGRx")
	}
	if err := drawCursor(nil, make([]byte, 8), 0, 0, 4, 4, false, 0); err != nil {
		t.Fatal(err)
	}
}

func TestQueue_ClosedAfterProducerExits(t *testing.T) {
	q := newFrameQueue()
	q.send(gpui.ScapFrameBGRx{ScapPixels: gpui.ScapPixels{Width: 1}}, nil)
	q.close()
	c := &Capturer{engine: &Engine{linux: &LinuxCapturer{}}, rx: q}
	f, err := c.GetNextFrame()
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := f.Size(); w != 1 {
		t.Fatalf("width = %d", w)
	}
	if _, err := c.GetNextFrame(); err == nil || err.Error() != "receiving on a closed channel" {
		t.Fatalf("err = %v", err)
	}
}

func TestCapturer_ResolutionValue(t *testing.T) {
	if got := resolutionValue(gpui.ScapResolution1080p, 16.0/9.0); got != [2]uint32{1920, 1080} {
		t.Fatalf("got %v", got)
	}
	defer func() {
		if r := recover(); r != ".value should not be called when Resolution type is Captured" {
			t.Fatalf("recover = %v", r)
		}
	}()
	resolutionValue(gpui.ScapResolutionCaptured, 1)
}

func TestLinuxTargets_DecodeCompoundText(t *testing.T) {
	cases := []struct {
		in   []byte
		want string
	}{
		{[]byte("plain"), "plain"},
		{[]byte{'c', 0xe9}, "cé"},
		{append(append([]byte{0x1b, '%', 'G'}, "日本"...), 0x1b, '%', '@'), "日本"},
		{[]byte{0x1b, '$', '(', 'B', 0x30, 0x21}, "n/a"},
	}
	for _, c := range cases {
		got, err := decodeCompoundText(c.in)
		if err != nil || got != c.want {
			t.Errorf("decodeCompoundText(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	if _, err := decodeCompoundText([]byte{'a', 0, 'b'}); err == nil {
		t.Error("NUL byte must fail like CString::new")
	}
}

func TestWayland_ParamChangedAndProcessDeliverFrames(t *testing.T) {
	pod, err := hex.DecodeString("800000000f00000003000400040000000100000000000000040000000300000002000000000000000200000000000000040000000300000001000000000000000100020000000000040000000300000008000000000000000300020000000000080000000a00000080070000380400000400020000000000080000000b0000003c00000001000000")
	if err != nil {
		t.Fatal(err)
	}
	ud := &listenerUserData{tx: newFrameQueue()}
	paramChangedCallback(ud, pw.SpaParamEnumFormat, pod)
	if ud.format.Format != pw.VideoFormatUnknown {
		t.Fatal("non-Format param ids are ignored")
	}
	paramChangedCallback(ud, pw.SpaParamFormat, pod)
	if ud.format.Format != pw.VideoFormatBGRx || ud.format.Size.Width != 1920 {
		t.Fatalf("format = %+v", ud.format)
	}

	pixels := make([]byte, 64)
	pixels[0] = 7
	header := pw.SpaMetaHeaderData{Pts: 1234}
	meta := pw.SpaMeta{Type: pw.SpaMetaHeader, Size: uint32(unsafe.Sizeof(header)), Data: unsafe.Pointer(&header)}
	data := pw.SpaData{MaxSize: uint32(len(pixels)), Data: unsafe.Pointer(&pixels[0])}
	sb := pw.SpaBuffer{NMetas: 1, NDatas: 1, Metas: &meta, Datas: &data}
	frame, err := processCallbackImpl(&pw.Buffer{Buffer: &sb}, ud)
	if err != nil {
		t.Fatal(err)
	}
	bgrx, ok := frame.(gpui.ScapFrameBGRx)
	if !ok || bgrx.DisplayTime != 1234 || bgrx.Width != 1920 || bgrx.Height != 1080 || len(bgrx.Data) != 64 || bgrx.Data[0] != 7 {
		t.Fatalf("frame = %T %+v", frame, frame)
	}
	if _, err := processCallbackImpl(nil, ud); err == nil || err.Error() != "Wayland screen capture out of buffers." {
		t.Fatalf("err = %v", err)
	}
	ud.format.Format = pw.VideoFormatRGBA
	if _, err := processCallbackImpl(&pw.Buffer{Buffer: &sb}, ud); err == nil || err.Error() != "Unsupported frame format received" {
		t.Fatalf("err = %v", err)
	}
}

func TestX11_CapturesADisplayFrame(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" || os.Getenv("DISPLAY") == "" {
		t.Skip("needs an X server; set GPUI_TEST_DISPLAY=1 and DISPLAY")
	}
	targets, err := getX11Targets()
	if err != nil {
		t.Skipf("X11 targets unavailable: %v", err)
	}
	var display *gpui.ScapDisplay
	for _, target := range targets {
		if d, ok := target.(*gpui.ScapDisplay); ok {
			display = d
			break
		}
	}
	if display == nil {
		t.Skip("no RandR display")
	}
	q := newFrameQueue()
	c, err := NewX11Capturer(gpui.ScapCapturerOptions{FPS: 30, ShowCursor: true, Target: display}, q)
	if err != nil {
		t.Skipf("X11 capturer unavailable: %v", err)
	}
	d, ok := c.target.(*gpui.ScapDisplay)
	if !ok {
		t.Fatalf("target = %T", c.target)
	}
	c.StartCapture()
	res, err := q.recv()
	c.StopCapture()
	if err != nil || res.err != nil {
		t.Fatalf("frame: %v %v", err, res.err)
	}
	f := res.frame.(gpui.ScapFrameBGRx)
	conn, err := xgb.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	xc := &x11Conn{conn: conn, noShm: shm.Init(conn) != nil}
	defer xc.closeShm()
	shmData, err := getImage(xc, xproto.Window(display.RawHandle), display.XOffset, display.YOffset, display.Width, display.Height)
	if err != nil {
		t.Fatal(err)
	}
	if xc.noShm || xc.shm == nil {
		t.Fatal("MIT-SHM path was not used")
	}
	if want := int(display.Width) * int(display.Height) * 4; len(shmData) != want {
		t.Fatalf("shm image %d bytes, want %d", len(shmData), want)
	}
	if f.Width != int32(d.Width) || f.Height != int32(d.Height) || len(f.Data) < int(d.Width)*int(d.Height)*4 {
		t.Fatalf("frame %dx%d len %d for display %+v", f.Width, f.Height, len(f.Data), d)
	}
}
