//go:build windows

package wgc

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/com"
)

func pinterfaceIID(signature string) com.GUID {
	ns := com.MustGUID("{11f47ad5-7b73-42c0-abae-878b1e16adee}")
	var b [16]byte
	binary.BigEndian.PutUint32(b[0:], ns.Data1)
	binary.BigEndian.PutUint16(b[4:], ns.Data2)
	binary.BigEndian.PutUint16(b[6:], ns.Data3)
	copy(b[8:], ns.Data4[:])
	h := sha1.Sum(append(b[:], signature...))
	h[6] = h[6]&0x0f | 0x50
	h[8] = h[8]&0x3f | 0x80
	var g com.GUID
	g.Data1 = binary.BigEndian.Uint32(h[0:])
	g.Data2 = binary.BigEndian.Uint16(h[4:])
	g.Data3 = binary.BigEndian.Uint16(h[6:])
	copy(g.Data4[:], h[8:16])
	return g
}

func TestWinRT_PinterfaceIIDs(t *testing.T) {
	cases := []struct {
		sig  string
		want com.GUID
	}{
		{"pinterface({9fc2b0bb-e446-44e2-aa61-9cab8f636af2};b1)", com.MustGUID("{cdb5efb3-5788-509d-9be1-71ccb8a3362a}")},
		{"pinterface({9de1c534-6ae1-11e0-84e1-18a905bcc53f};rc(Windows.Graphics.Capture.Direct3D11CaptureFramePool;{24eb6d22-1975-422e-82e7-780dbd8ddf24});cinterface(IInspectable))", IID_TypedEventHandlerFramePoolInspect},
		{"pinterface({9de1c534-6ae1-11e0-84e1-18a905bcc53f};rc(Windows.Graphics.Capture.GraphicsCaptureItem;{79c3f95b-31f7-4ec2-a464-632ef5d30760});cinterface(IInspectable))", IID_TypedEventHandlerCaptureItemInspect},
	}
	for _, c := range cases {
		if got := pinterfaceIID(c.sig); got != c.want {
			t.Errorf("%s: got %v, want %v", c.sig, got, c.want)
		}
	}
}

func TestSys_Layouts(t *testing.T) {
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"DISPLAYCONFIG_PATH_INFO", unsafe.Sizeof(displayconfigPathInfo{}), 72},
		{"DISPLAYCONFIG_MODE_INFO", unsafe.Sizeof(displayconfigModeInfo{}), 64},
		{"DISPLAYCONFIG_DEVICE_INFO_HEADER", unsafe.Sizeof(displayconfigDeviceInfoHeader{}), 20},
		{"DISPLAYCONFIG_SOURCE_DEVICE_NAME", unsafe.Sizeof(displayconfigSourceDeviceName{}), 84},
		{"DISPLAYCONFIG_TARGET_DEVICE_NAME", unsafe.Sizeof(displayconfigTargetDeviceName{}), 420},
		{"DISPLAY_DEVICEW", unsafe.Sizeof(displayDeviceW{}), 840},
		{"DispatcherQueueOptions", unsafe.Sizeof(dispatcherQueueOptions{}), 12},
		{"SizeInt32", unsafe.Sizeof(SizeInt32{}), 8},
		{"TimeSpan", unsafe.Sizeof(TimeSpan{}), 8},
		{"DISPLAYCONFIG_PATH_INFO.targetInfo", unsafe.Offsetof(displayconfigPathInfo{}.TargetInfo), 20},
		{"DISPLAYCONFIG_TARGET_DEVICE_NAME.monitorFriendlyDeviceName", unsafe.Offsetof(displayconfigTargetDeviceName{}.MonitorFriendlyDeviceName), 36},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestSys_UTF16UntilNul(t *testing.T) {
	s, err := utf16UntilNul([]uint16{'a', 'b', 0, 'c'})
	if err != nil || s != "ab" {
		t.Fatalf("got %q, %v", s, err)
	}
	if _, err := utf16UntilNul([]uint16{0xd800, 'a'}); err == nil {
		t.Fatal("lone surrogate accepted")
	}
}

func TestFrame_AsNopaddingBuffer(t *testing.T) {
	raw := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 9, 10, 11, 12, 13, 14, 15, 16, 17, 9, 9}
	buf := make([]byte, 0, 4)
	fb := NewFrameBuffer(raw, &buf, 2, 2, 10, 20, ColorFormatBgra8)
	if !fb.HasPadding() {
		t.Fatal("expected padding")
	}
	got, err := fb.AsNopaddingBuffer()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 13, 14, 15, 16, 17}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	packed := NewFrameBuffer(want, &buf, 2, 2, 8, 16, ColorFormatRgba8)
	if got, _ := packed.AsNopaddingBuffer(); &got[0] != &want[0] {
		t.Fatal("unpadded buffer was copied")
	}
}

func TestGraphicsCaptureApi_IsSupported(t *testing.T) {
	if _, err := GraphicsCaptureApiIsSupported(); err != nil {
		t.Fatal(err)
	}
}

func TestMonitor_Enumerate(t *testing.T) {
	monitors, err := MonitorEnumerate()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range monitors {
		if _, err := m.DeviceName(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWindow_Enumerate(t *testing.T) {
	list, err := WindowEnumerate()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range list {
		if _, err := w.Title(); err != nil && !strings.Contains(err.Error(), "UTF-16") {
			t.Fatal(err)
		}
	}
}

type countingHandler struct {
	frames atomic.Int32
	sizes  chan [2]uint32
}

func (h *countingHandler) OnFrameArrived(frame *Frame, control InternalCaptureControl) error {
	fb, err := frame.Buffer()
	if err != nil {
		return err
	}
	if _, err := fb.AsNopaddingBuffer(); err != nil {
		return err
	}
	if h.frames.Add(1) == 1 {
		h.sizes <- [2]uint32{frame.Width(), frame.Height()}
	}
	control.Stop()
	return nil
}

func (h *countingHandler) OnClosed() error { return nil }

func TestCapture_PrimaryMonitor(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" {
		t.Skip("needs GPUI_TEST_DISPLAY=1")
	}
	if ok, err := GraphicsCaptureApiIsSupported(); err != nil || !ok {
		t.Skip("Windows.Graphics.Capture is unavailable")
	}
	monitor, err := MonitorPrimary()
	if err != nil {
		t.Skip("no monitor")
	}
	h := &countingHandler{sizes: make(chan [2]uint32, 1)}
	settings := NewSettings(monitor, CursorCaptureSettingsDefault, DrawBorderSettingsDefault, SecondaryWindowSettingsDefault, MinimumUpdateIntervalSettings{}, DirtyRegionSettingsDefault, ColorFormatBgra8, struct{}{})
	control, err := StartFreeThreaded(settings, func(Context[struct{}]) (GraphicsCaptureApiHandler, error) { return h, nil })
	if err != nil {
		t.Skipf("capture unavailable: %v", err)
	}
	select {
	case size := <-h.sizes:
		if size[0] == 0 || size[1] == 0 {
			t.Fatalf("empty frame %v", size)
		}
	case <-time.After(10 * time.Second):
		t.Error("no frame within 10s")
	}
	if err := control.Stop(); err != nil {
		t.Fatal(err)
	}
}
