//go:build linux || windows

package gpui

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

type fakeScapBackend struct {
	targets  []ScapTarget
	frames   int
	mu       sync.Mutex
	built    []*fakeScapCapturer
	firstErr error
}

func (b *fakeScapBackend) GetAllTargets() ([]ScapTarget, error) {
	return b.targets, nil
}

func (b *fakeScapBackend) BuildCapturer(options ScapCapturerOptions) (ScapCapturer, error) {
	if options.FPS != 60 || !options.ShowCursor || !options.ShowHighlight || options.OutputType != ScapFrameTypeYUVFrame || options.OutputResolution != ScapResolutionCaptured {
		return nil, errors.New("unexpected capturer options")
	}
	target := options.Target
	if target == nil {
		target = b.targets[0]
	}
	c := &fakeScapCapturer{target: target, frames: b.frames, firstErr: b.firstErr, closed: make(chan struct{})}
	b.mu.Lock()
	b.built = append(b.built, c)
	b.mu.Unlock()
	return c, nil
}

func (b *fakeScapBackend) capturer(t *testing.T) *fakeScapCapturer {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.built) != 1 {
		t.Fatalf("built %d capturers, want 1", len(b.built))
	}
	return b.built[0]
}

type fakeScapCapturer struct {
	target   ScapTarget
	frames   int
	firstErr error
	sent     atomic.Int32
	started  atomic.Bool
	stopped  atomic.Bool
	closed   chan struct{}
}

func (c *fakeScapCapturer) StartCapture() { c.started.Store(true) }
func (c *fakeScapCapturer) StopCapture()  { c.stopped.Store(true) }
func (c *fakeScapCapturer) Target() ScapTarget {
	return c.target
}
func (c *fakeScapCapturer) Close() { close(c.closed) }

func (c *fakeScapCapturer) GetNextFrame() (ScapFrame, error) {
	if c.firstErr != nil {
		return nil, c.firstErr
	}
	n := c.sent.Add(1)
	if int(n) > c.frames {
		return nil, errors.New("no more frames")
	}
	return ScapFrameBGRx{ScapPixels{Width: 800, Height: 600}}, nil
}

func (c *fakeScapCapturer) waitClosed(t *testing.T) {
	t.Helper()
	select {
	case <-c.closed:
	case <-time.After(5 * time.Second):
		t.Fatal("capturer was not closed")
	}
}

func withScapBackend(t *testing.T, backend ScapBackend) {
	t.Helper()
	scapBackendRegistry.mu.Lock()
	saved := scapBackendRegistry.backend
	scapBackendRegistry.backend = backend
	scapBackendRegistry.mu.Unlock()
	t.Cleanup(func() {
		scapBackendRegistry.mu.Lock()
		scapBackendRegistry.backend = saved
		scapBackendRegistry.mu.Unlock()
	})
}

func scapTestExecutor() ForegroundExecutor {
	dispatcher := NewTestDispatcher(0)
	dispatcher.AllowParking()
	return NewForegroundExecutor(dispatcher)
}

func blockOnScap[T any](executor ForegroundExecutor, rx *scheduler.OneshotReceiver[scheduler.Result[T]]) (T, error) {
	result := executor.BlockTest(scheduler.Future[scheduler.Result[scheduler.Result[T]]](rx))
	if result.Err != nil {
		var zero T
		return zero, result.Err
	}
	return result.Value.Value, result.Value.Err
}

func TestScapScreenCapture_NoBackend(t *testing.T) {
	withScapBackend(t, nil)
	executor := scapTestExecutor()
	if _, err := blockOnScap(executor, ScapScreenSources(executor)); err == nil || err.Error() != "Screen capturing is not supported" {
		t.Fatalf("ScapScreenSources error = %v", err)
	}
	if _, err := blockOnScap(executor, startScapDefaultTargetSource(executor)); err == nil || err.Error() != "Screen capturing is not supported" {
		t.Fatalf("startScapDefaultTargetSource error = %v", err)
	}
}

func TestScapScreenCapture_ScreenSourcesStream(t *testing.T) {
	display := &ScapDisplay{ID: 7, Title: "Main", Width: 1920, Height: 1080}
	backend := &fakeScapBackend{targets: []ScapTarget{&ScapWindow{ID: 3, Title: "win"}, display}, frames: 3}
	withScapBackend(t, backend)
	executor := scapTestExecutor()

	sources, err := blockOnScap(executor, ScapScreenSources(executor))
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("got %d sources, want 1", len(sources))
	}
	metadata, err := sources[0].Metadata()
	if err != nil {
		t.Fatal(err)
	}
	label, _ := metadata.Label.Get()
	if metadata.ID != 7 || label.String() != "Main" || metadata.IsMain.IsSome() || metadata.Resolution != NewSize(DevicePixels(1920), DevicePixels(1080)) {
		t.Fatalf("metadata = %+v", metadata)
	}

	var frames atomic.Int32
	stream, err := blockOnScap(executor, sources[0].Stream(executor, func(frame ScreenCaptureFrame) {
		frames.Add(1)
	}))
	if err != nil {
		t.Fatal(err)
	}
	streamMetadata, _ := stream.Metadata()
	if streamMetadata.ID != 7 || streamMetadata.Resolution != metadata.Resolution {
		t.Fatalf("stream metadata = %+v", streamMetadata)
	}
	capturer := backend.capturer(t)
	capturer.waitClosed(t)
	stream.Stop()
	if !capturer.started.Load() || !capturer.stopped.Load() || frames.Load() != 3 {
		t.Fatalf("started=%v stopped=%v frames=%d", capturer.started.Load(), capturer.stopped.Load(), frames.Load())
	}
}

func TestScapScreenCapture_DefaultTargetSource(t *testing.T) {
	display := &ScapDisplay{ID: 2, Title: "Portal", Width: 1280, Height: 720}
	backend := &fakeScapBackend{targets: []ScapTarget{display}, frames: 2}
	withScapBackend(t, backend)
	executor := scapTestExecutor()

	sources, err := blockOnScap(executor, startScapDefaultTargetSource(executor))
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("got %d sources, want 1", len(sources))
	}
	metadata, _ := sources[0].Metadata()
	if metadata.ID != 2 || metadata.Label.IsSome() || metadata.Resolution != NewSize(DevicePixels(800), DevicePixels(600)) {
		t.Fatalf("metadata = %+v", metadata)
	}

	var frames atomic.Int32
	stream, err := blockOnScap(executor, sources[0].Stream(executor, func(frame ScreenCaptureFrame) { frames.Add(1) }))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Stop()
	if _, err := blockOnScap(executor, sources[0].Stream(executor, func(ScreenCaptureFrame) {})); err == nil {
		t.Fatalf("second Stream error = %v", err)
	}
	capturer := backend.capturer(t)
	capturer.waitClosed(t)
	if frames.Load() != 1 {
		t.Fatalf("frames = %d, want 1 after the size probe", frames.Load())
	}
}

func TestScapScreenCapture_DefaultTargetReleased(t *testing.T) {
	backend := &fakeScapBackend{targets: []ScapTarget{&ScapDisplay{ID: 1}}, frames: 5}
	withScapBackend(t, backend)
	executor := scapTestExecutor()

	sources, err := blockOnScap(executor, startScapDefaultTargetSource(executor))
	if err != nil {
		t.Fatal(err)
	}
	sources[0].(interface{ Release() }).Release()
	backend.capturer(t).waitClosed(t)
}

func TestScapScreenCapture_DefaultTargetFirstFrameError(t *testing.T) {
	backend := &fakeScapBackend{targets: []ScapTarget{&ScapDisplay{ID: 1}}, firstErr: errors.New("boom")}
	withScapBackend(t, backend)
	executor := scapTestExecutor()

	_, err := blockOnScap(executor, startScapDefaultTargetSource(executor))
	if err == nil || err.Error() != "Failed to get first frame of screenshare to get the size.: boom" {
		t.Fatalf("error = %v", err)
	}
	backend.capturer(t).waitClosed(t)
}
