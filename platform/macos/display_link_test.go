//go:build darwin && !ios

package macos

import (
	"math"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

func TestDisplayLink_MetalFramesPauseOnlyWhenGpuiNeitherDrawsNorAsksAgain(t *testing.T) {
	frames := &MetalFrames{}

	frames.BeginFrame()
	if frames.EndFrame(true) {
		t.Fatal("a frame that drew keeps the link running")
	}

	frames.BeginFrame()
	frames.Wake()
	if frames.EndFrame(false) {
		t.Fatal("a wake during the update, like a throttled frame's retry, keeps it running")
	}

	frames.Wake()
	frames.BeginFrame()
	if !frames.EndFrame(false) {
		t.Fatal("a wake before the update was served by it")
	}

	frames.BeginFrame()
	if !frames.EndFrame(false) {
		t.Fatal("an idle update pauses the link")
	}
}

func TestDisplayLink_ARendererDrawsNormallyAfterItsDisplayLinkIsInvalidated(t *testing.T) {
	if !apple.MetalFrameSourceIsAvailable() {
		return
	}
	renderer, err := apple.NewMetalRenderer(apple.NewInstanceBufferPool(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Destroy()
	renderer.UpdateDrawableSize(gpui.NewSize(gpui.DevicePixels(64), gpui.DevicePixels(64)))
	layer, ok := renderer.Layer()
	if !ok {
		t.Fatal("window renderer has a layer")
	}
	link := objc.Send[objc.ID](objc.Alloc(objc.GetClass("CAMetalDisplayLink")), objc.Sel("initWithMetalLayer:"), layer.ID())
	defer objc.Release(link)
	renderer.SetDisplayLinkAttached(true)
	renderer.Draw(&gpui.Scene{})

	objc.SendVoid(link, objc.Sel("invalidate"))
	renderer.SetDisplayLinkAttached(false)
	renderer.Draw(&gpui.Scene{})
}

func TestDisplayLink_HostTimestampPreservesRequestAgeAtDifferentClockRates(t *testing.T) {
	receivedAt := scheduler.InstantNow()
	for _, c := range []struct {
		hostTime, currentHostTime uint64
		frequency                 float64
		age                       time.Duration
	}{
		{4_000_000, 10_000_000, 1e9, 6 * time.Millisecond},
		{1_760_000, 2_000_000, 24e6, 10 * time.Millisecond},
	} {
		ts := testTimestamp(c.hostTime)
		got := frameRequestInstant(&ts, c.currentHostTime, c.frequency, receivedAt)
		if want := receivedAt.Add(-c.age); !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDisplayLink_InvalidHostTimestampsFallBackToReceiptTime(t *testing.T) {
	receivedAt := scheduler.InstantNow()
	valid := testTimestamp(1_000)
	missingHostTime := valid
	missingHostTime.flags = 0
	unknownVersion := valid
	unknownVersion.version = 1
	for _, ts := range []*cvTimeStamp{nil, &missingHostTime, &unknownVersion} {
		if got := frameRequestInstant(ts, 2_000, 1e9, receivedAt); !got.Equal(receivedAt) {
			t.Fatalf("got %v, want %v", got, receivedAt)
		}
	}
	if got := frameRequestInstant(&valid, 999, 1e9, receivedAt); !got.Equal(receivedAt) {
		t.Fatalf("got %v, want %v", got, receivedAt)
	}
	for _, frequency := range []float64{0.0, -1.0, math.NaN(), math.Inf(1), 0x1p-1022} {
		if got := frameRequestInstant(&valid, 2_000, frequency, receivedAt); !got.Equal(receivedAt) {
			t.Fatalf("frequency %v: got %v, want %v", frequency, got, receivedAt)
		}
	}
}

func testTimestamp(hostTime uint64) cvTimeStamp {
	return cvTimeStamp{hostTime: hostTime, flags: Kcvtimestamphosttimevalid}
}
