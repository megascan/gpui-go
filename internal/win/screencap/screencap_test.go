//go:build windows

package screencap

import (
	"errors"
	"os"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/wgc"
)

func TestWinEngine_GetAbsoluteValue(t *testing.T) {
	cases := []struct{ v, scale, want float64 }{
		{10, 1, 10},
		{10.7, 1, 10},
		{11, 1, 12},
		{11, 1.5, 16},
		{3.4, 1.25, 4},
	}
	for _, c := range cases {
		if got := getAbsoluteValue(c.v, c.scale); got != c.want {
			t.Errorf("getAbsoluteValue(%v, %v) = %v, want %v", c.v, c.scale, got, c.want)
		}
	}
}

func TestWinEngine_SaturatingCasts(t *testing.T) {
	if f64ToU32(-3) != 0 || f64ToU32(5.9) != 5 || f64ToU32(1e20) != 1<<32-1 {
		t.Fatal("f64ToU32")
	}
	if f64ToI32(-1e20) != -1<<31 || f64ToI32(7.2) != 7 {
		t.Fatal("f64ToI32")
	}
}

func TestQueue_CloseFailsSendAndRecv(t *testing.T) {
	q := newFrameQueue()
	if err := q.send(gpui.ScapFrameBGRA{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := q.recv(); err != nil {
		t.Fatal(err)
	}
	q.close()
	if err := q.send(gpui.ScapFrameBGRA{}, nil); !errors.Is(err, errSend) {
		t.Fatalf("send after close: %v", err)
	}
	if _, err := q.recv(); !errors.Is(err, errRecv) {
		t.Fatalf("recv after close: %v", err)
	}
}

func TestBackend_Interface(t *testing.T) {
	var _ gpui.ScapBackend = Backend{}
	var _ gpui.ScapCapturer = (*Capturer)(nil)
}

func TestCapturer_MainDisplay(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" {
		t.Skip("needs GPUI_TEST_DISPLAY=1")
	}
	if !IsSupported() {
		t.Skip("Windows.Graphics.Capture is unavailable")
	}
	device, deviceContext, err := wgc.CreateD3DDevice()
	if err != nil {
		t.Skipf("no D3D11 hardware device: %v", err)
	}
	device.Release()
	deviceContext.Release()
	monitor, err := wgc.MonitorPrimary()
	if err != nil {
		t.Skipf("no primary monitor: %v", err)
	}
	item, _, err := monitor.TryIntoCaptureItem()
	if err != nil {
		t.Skipf("monitor cannot be captured: %v", err)
	}
	item.Release()
	targets, err := GetAllTargets()
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) == 0 {
		t.Skip("no targets")
	}
	c, err := Build(gpui.ScapCapturerOptions{FPS: 30, OutputType: gpui.ScapFrameTypeBGRAFrame})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.StartCapture()
	got := make(chan gpui.ScapFrame, 1)
	go func() {
		f, err := c.GetNextFrame()
		if err == nil {
			got <- f
		}
	}()
	select {
	case f := <-got:
		bgra, ok := f.(gpui.ScapFrameBGRA)
		if !ok {
			t.Fatalf("frame type %T", f)
		}
		if int(bgra.Width)*int(bgra.Height)*4 != len(bgra.Data) {
			t.Fatalf("frame %dx%d has %d bytes", bgra.Width, bgra.Height, len(bgra.Data))
		}
	case <-time.After(10 * time.Second):
		t.Error("no frame within 10s")
	}
	c.StopCapture()
}

func TestFrameQueue_FinishDeliversErrorThenCloses(t *testing.T) {
	q := newFrameQueue()
	if err := q.send(gpui.ScapFrameBGRA{}, nil); err != nil {
		t.Fatal(err)
	}
	want := errors.New("thread ended")
	q.finish(want)
	q.finish(errors.New("second"))
	if r, err := q.recv(); err != nil || r.err != nil {
		t.Fatalf("first recv: %v %v", r.err, err)
	}
	if r, err := q.recv(); err != nil || !errors.Is(r.err, want) {
		t.Fatalf("second recv: %v %v", r.err, err)
	}
	if _, err := q.recv(); !errors.Is(err, errRecv) {
		t.Fatalf("recv after finish: %v", err)
	}
	if err := q.send(gpui.ScapFrameBGRA{}, nil); !errors.Is(err, errSend) {
		t.Fatalf("send after finish: %v", err)
	}
}
