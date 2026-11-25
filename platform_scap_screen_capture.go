//go:build linux || windows

package gpui

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/scheduler"
)

var errScapNotSupported = errors.New("Screen capturing is not supported")

// ScapScreenSources populates the receiver with the screens that can be captured.
//
// startScapDefaultTargetSource should be used instead on Wayland, since ScapScreenSources
// won't return any results.
func ScapScreenSources(foregroundExecutor ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	sourcesTx, sourcesRx := scheduler.Oneshot[scheduler.Result[[]*scapCaptureSource]]()
	getScreenTargets(sourcesTx)
	return toDynScreenCaptureSources(sourcesRx, foregroundExecutor)
}

func startScapDefaultTargetSource(foregroundExecutor ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	sourcesTx, sourcesRx := scheduler.Oneshot[scheduler.Result[[]*scapDefaultTargetCaptureSource]]()
	startDefaultTargetScreenCapture(sourcesTx)
	return toDynScreenCaptureSources(sourcesRx, foregroundExecutor)
}

type scapCaptureSource struct {
	target ScapDisplay
	size   Size[DevicePixels]
}

func scapGetAllTargets() ([]ScapTarget, error) {
	backend, ok := registeredScapBackend()
	if !ok {
		return nil, errScapNotSupported
	}
	return backend.GetAllTargets()
}

func getScreenTargets(sourcesTx *scheduler.OneshotSender[scheduler.Result[[]*scapCaptureSource]]) {
	go func() {
		targets, err := scapGetAllTargets()
		if err != nil {
			_ = sourcesTx.Send(scheduler.Result[[]*scapCaptureSource]{Err: err})
			return
		}
		sources := make([]*scapCaptureSource, 0, len(targets))
		for _, target := range targets {
			display, ok := target.(*ScapDisplay)
			if !ok {
				continue
			}
			size := Size[DevicePixels]{
				Width:  DevicePixels(int32(display.Width)),
				Height: DevicePixels(int32(display.Height)),
			}
			sources = append(sources, &scapCaptureSource{target: *display, size: size})
		}
		_ = sourcesTx.Send(scheduler.Result[[]*scapCaptureSource]{Value: sources})
	}()
}

func (s *scapCaptureSource) Metadata() (SourceMetadata, error) {
	return SourceMetadata{
		Resolution: s.size,
		Label:      Some(NewSharedString(s.target.Title)),
		IsMain:     None[bool](),
		ID:         uint64(s.target.ID),
	}, nil
}

func (s *scapCaptureSource) Stream(foregroundExecutor ForegroundExecutor, frameCallback func(frame ScreenCaptureFrame)) *scheduler.OneshotReceiver[scheduler.Result[ScreenCaptureStream]] {
	streamTx, streamRx := scheduler.Oneshot[scheduler.Result[*scapStream]]()
	target := s.target

	go func() {
		displayTarget := target
		capturer, err := newScapCapturer(&displayTarget)
		if err != nil {
			_ = streamTx.Send(scheduler.Result[*scapStream]{Err: err})
			return
		}
		capturer.StartCapture()
		runCapture(capturer, target, frameCallback, streamTx)
	}()

	return toDynScreenCaptureStream(streamRx, foregroundExecutor)
}

type scapStreamCall struct {
	streamTx      *scheduler.OneshotSender[scheduler.Result[*scapStream]]
	frameCallback func(frame ScreenCaptureFrame)
}

type scapDefaultTargetCaptureSource struct {
	streamCallMu           sync.Mutex
	streamCallClosed       bool
	streamCallDisconnected bool
	streamCallTx           chan scapStreamCall
	target                 ScapDisplay
	size                   Size[DevicePixels]
}

func startDefaultTargetScreenCapture(sourcesTx *scheduler.OneshotSender[scheduler.Result[[]*scapDefaultTargetCaptureSource]]) {
	go func() {
		capturer, size, target, err := func() (ScapCapturer, Size[DevicePixels], ScapTarget, error) {
			capturer, err := newScapCapturer(nil)
			if err != nil {
				return nil, Size[DevicePixels]{}, nil, err
			}
			capturer.StartCapture()
			firstFrame, err := capturer.GetNextFrame()
			if err != nil {
				closeScapCapturer(capturer)
				return nil, Size[DevicePixels]{}, nil, fmt.Errorf("Failed to get first frame of screenshare to get the size.: %w", err)
			}
			size := frameSize(firstFrame)
			target := capturer.Target()
			if target == nil {
				closeScapCapturer(capturer)
				return nil, Size[DevicePixels]{}, nil, errors.New("Unable to determine the target display.")
			}
			return capturer, size, target, nil
		}()
		if err != nil {
			_ = sourcesTx.Send(scheduler.Result[[]*scapDefaultTargetCaptureSource]{Err: err})
			return
		}

		display, ok := target.(*ScapDisplay)
		if !ok {
			closeScapCapturer(capturer)
			_ = sourcesTx.Send(scheduler.Result[[]*scapDefaultTargetCaptureSource]{Err: errors.New("The screen capture source is not a display")})
			return
		}
		streamRx := make(chan scapStreamCall, 1)
		source := &scapDefaultTargetCaptureSource{
			streamCallTx: streamRx,
			size:         size,
			target:       *display,
		}
		defer source.disconnectStreamCalls()
		if err := sourcesTx.Send(scheduler.Result[[]*scapDefaultTargetCaptureSource]{Value: []*scapDefaultTargetCaptureSource{source}}); err != nil {
			closeScapCapturer(capturer)
			return
		}
		call, ok := <-streamRx
		if !ok {
			closeScapCapturer(capturer)
			return
		}
		runCapture(capturer, *display, call.frameCallback, call.streamTx)
	}()
}

func (s *scapDefaultTargetCaptureSource) Metadata() (SourceMetadata, error) {
	return SourceMetadata{
		Resolution: s.size,
		Label:      None[SharedString](),
		IsMain:     None[bool](),
		ID:         uint64(s.target.ID),
	}, nil
}

func (s *scapDefaultTargetCaptureSource) Stream(foregroundExecutor ForegroundExecutor, frameCallback func(frame ScreenCaptureFrame)) *scheduler.OneshotReceiver[scheduler.Result[ScreenCaptureStream]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[*scapStream]]()
	sent := false
	s.streamCallMu.Lock()
	if !s.streamCallClosed && !s.streamCallDisconnected {
		select {
		case s.streamCallTx <- scapStreamCall{streamTx: tx, frameCallback: frameCallback}:
			sent = true
		default:
		}
	}
	s.streamCallMu.Unlock()
	if !sent {
		_ = tx.Send(scheduler.Result[*scapStream]{Err: errors.New("Can't call ScapDefaultTargetCaptureSource::stream multiple times.")})
	}
	return toDynScreenCaptureStream(rx, foregroundExecutor)
}

// Release drops the source; a capture whose stream was never requested stops.
func (s *scapDefaultTargetCaptureSource) Release() {
	s.streamCallMu.Lock()
	defer s.streamCallMu.Unlock()
	if s.streamCallClosed {
		return
	}
	s.streamCallClosed = true
	close(s.streamCallTx)
}

func (s *scapDefaultTargetCaptureSource) disconnectStreamCalls() {
	s.streamCallMu.Lock()
	defer s.streamCallMu.Unlock()
	s.streamCallDisconnected = true
	for {
		select {
		case call, ok := <-s.streamCallTx:
			if !ok {
				return
			}
			call.streamTx.Close()
		default:
			return
		}
	}
}

func newScapCapturer(target ScapTarget) (ScapCapturer, error) {
	backend, ok := registeredScapBackend()
	if !ok {
		return nil, errScapNotSupported
	}
	return backend.BuildCapturer(ScapCapturerOptions{
		FPS:              60,
		ShowCursor:       true,
		ShowHighlight:    true,
		OutputType:       ScapFrameTypeYUVFrame,
		OutputResolution: ScapResolutionCaptured,
		CropArea:         None[ScapArea](),
		Target:           target,
		ExcludedTargets:  nil,
	})
}

func closeScapCapturer(capturer ScapCapturer) {
	if closer, ok := capturer.(interface{ Close() }); ok {
		closer.Close()
	}
}

func runCapture(capturer ScapCapturer, display ScapDisplay, frameCallback func(frame ScreenCaptureFrame), streamTx *scheduler.OneshotSender[scheduler.Result[*scapStream]]) {
	defer closeScapCapturer(capturer)
	cancelStream := new(atomic.Bool)
	size := Size[DevicePixels]{
		Width:  DevicePixels(int32(display.Width)),
		Height: DevicePixels(int32(display.Height)),
	}
	if err := streamTx.Send(scheduler.Result[*scapStream]{Value: &scapStream{
		cancelStream: cancelStream,
		display:      display,
		size:         size,
	}}); err != nil {
		return
	}
	for !cancelStream.Load() {
		frame, err := capturer.GetNextFrame()
		if err != nil {
			slog.Error(fmt.Sprintf("Halting screen capture due to error: %v", err))
			break
		}
		frameCallback(ScreenCaptureFrame{Frame: frame})
	}
	capturer.StopCapture()
}

type scapStream struct {
	cancelStream *atomic.Bool
	display      ScapDisplay
	size         Size[DevicePixels]
}

func (s *scapStream) Metadata() (SourceMetadata, error) {
	return SourceMetadata{
		Resolution: s.size,
		Label:      Some(NewSharedString(s.display.Title)),
		IsMain:     None[bool](),
		ID:         uint64(s.display.ID),
	}, nil
}

// Stop ends the capture; it is upstream's drop of ScapStream.
func (s *scapStream) Stop() {
	s.cancelStream.Store(true)
}

func frameSize(frame ScapFrame) Size[DevicePixels] {
	width, height := frame.Size()
	return NewSize(DevicePixels(width), DevicePixels(height))
}

func toDynScreenCaptureSources[T ScreenCaptureSource](sourcesRx *scheduler.OneshotReceiver[scheduler.Result[[]T]], foregroundExecutor ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	dynSourcesTx, dynSourcesRx := scheduler.Oneshot[scheduler.Result[[]ScreenCaptureSource]]()
	foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		result, err := sourcesRx.Recv(s)
		if err != nil {
			dynSourcesTx.Close()
			return struct{}{}, nil
		}
		if result.Err != nil {
			_ = dynSourcesTx.Send(scheduler.Result[[]ScreenCaptureSource]{Err: result.Err})
			return struct{}{}, nil
		}
		sources := make([]ScreenCaptureSource, 0, len(result.Value))
		for _, source := range result.Value {
			sources = append(sources, source)
		}
		if dynSourcesTx.Send(scheduler.Result[[]ScreenCaptureSource]{Value: sources}) != nil {
			for _, source := range sources {
				if releaser, ok := source.(interface{ Release() }); ok {
					releaser.Release()
				}
			}
		}
		return struct{}{}, nil
	}).Detach()
	return dynSourcesRx
}

func toDynScreenCaptureStream[T ScreenCaptureStream](sourcesRx *scheduler.OneshotReceiver[scheduler.Result[T]], foregroundExecutor ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[ScreenCaptureStream]] {
	dynSourcesTx, dynSourcesRx := scheduler.Oneshot[scheduler.Result[ScreenCaptureStream]]()
	foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		result, err := sourcesRx.Recv(s)
		if err != nil {
			dynSourcesTx.Close()
			return struct{}{}, nil
		}
		if result.Err != nil {
			_ = dynSourcesTx.Send(scheduler.Result[ScreenCaptureStream]{Err: result.Err})
			return struct{}{}, nil
		}
		if dynSourcesTx.Send(scheduler.Result[ScreenCaptureStream]{Value: result.Value}) != nil {
			result.Value.Stop()
		}
		return struct{}{}, nil
	}).Detach()
	return dynSourcesRx
}
