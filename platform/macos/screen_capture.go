//go:build darwin && !ios

package macos

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/cg"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	screenCaptureKitPath = "/System/Library/Frameworks/ScreenCaptureKit.framework/ScreenCaptureKit"
	coreMediaPath        = "/System/Library/Frameworks/CoreMedia.framework/CoreMedia"

	scStreamOutputTypeScreen objc.NSInteger = 0
)

var (
	screenCaptureOnce sync.Once
	screenCaptureErr  error

	cgDisplayModeGetPixelWidth   func(mode uintptr) uintptr
	cgDisplayModeGetPixelHeight  func(mode uintptr) uintptr
	cmSampleBufferGetImageBuffer func(sampleBuffer objc.ID) uintptr

	msgSendAddStreamOutput    func(objc.ID, objc.SEL, objc.ID, objc.NSInteger, objc.ID, unsafe.Pointer) bool
	msgSendRemoveStreamOutput func(objc.ID, objc.SEL, objc.ID, objc.NSInteger, unsafe.Pointer) bool

	streamOutputClassOnce sync.Once
	streamOutputClass     objc.Class
	streamOutputErr       error

	streamOutputCallbacksMu sync.Mutex
	streamOutputCallbacks   = map[objc.ID]func(frame gpui.ScreenCaptureFrame){}
)

func loadScreenCapture() error {
	screenCaptureOnce.Do(func() {
		if err := loadDisplay(); err != nil {
			screenCaptureErr = err
			return
		}
		if _, err := purego.Dlopen(screenCaptureKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
			screenCaptureErr = fmt.Errorf("macos: open %s: %w", screenCaptureKitPath, err)
			return
		}
		cmLib, err := purego.Dlopen(coreMediaPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			screenCaptureErr = fmt.Errorf("macos: open %s: %w", coreMediaPath, err)
			return
		}
		cgLib, err := purego.Dlopen(cg.CoreGraphicsPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			screenCaptureErr = fmt.Errorf("macos: open %s: %w", cg.CoreGraphicsPath, err)
			return
		}
		purego.RegisterLibFunc(&cgDisplayModeGetPixelWidth, cgLib, "CGDisplayModeGetPixelWidth")
		purego.RegisterLibFunc(&cgDisplayModeGetPixelHeight, cgLib, "CGDisplayModeGetPixelHeight")
		purego.RegisterLibFunc(&cmSampleBufferGetImageBuffer, cmLib, "CMSampleBufferGetImageBuffer")
		objc.BindMsgSend(&msgSendAddStreamOutput)
		objc.BindMsgSend(&msgSendRemoveStreamOutput)
	})
	return screenCaptureErr
}

// MacScreenCaptureSource is a display that ScreenCaptureKit can capture.
type MacScreenCaptureSource struct {
	scDisplay *objc.Retained
	meta      gpui.Option[screenMeta]
}

// MacScreenCaptureStream is a running ScreenCaptureKit stream; Stop is upstream's drop.
type MacScreenCaptureStream struct {
	scStream       *objc.Retained
	scStreamOutput *objc.Retained
	meta           gpui.SourceMetadata
	stopped        bool
}

var (
	_ gpui.ScreenCaptureSource = (*MacScreenCaptureSource)(nil)
	_ gpui.ScreenCaptureStream = (*MacScreenCaptureStream)(nil)
)

func (s *MacScreenCaptureSource) metadata() gpui.SourceMetadata {
	displayID := objc.Send[uint32](s.scDisplay.ID(), objc.Sel("displayID"))
	displayMode := cgDisplayCopyDisplayMode(displayID)
	var width, height uintptr
	if displayMode != 0 {
		width = cgDisplayModeGetPixelWidth(displayMode)
		height = cgDisplayModeGetPixelHeight(displayMode)
		cgDisplayModeRelease(displayMode)
	}
	size := gpui.NewSize(gpui.DevicePixels(int32(width)), gpui.DevicePixels(int32(height)))
	label, isMain := gpui.None[gpui.SharedString](), gpui.None[bool]()
	if meta, ok := s.meta.Get(); ok {
		label, isMain = gpui.Some(meta.label), gpui.Some(meta.isMain)
	}
	return gpui.SourceMetadata{
		ID:         uint64(displayID),
		Label:      label,
		IsMain:     isMain,
		Resolution: size,
	}
}

// Metadata returns the display's id, label and pixel resolution.
func (s *MacScreenCaptureSource) Metadata() (gpui.SourceMetadata, error) {
	return s.metadata(), nil
}

// Stream starts capturing the display, calling frameCallback with each frame. The frame's
// CVImageBufferRef is borrowed for the duration of the callback.
func (s *MacScreenCaptureSource) Stream(foregroundExecutor gpui.ForegroundExecutor, frameCallback func(frame gpui.ScreenCaptureFrame)) *scheduler.OneshotReceiver[scheduler.Result[gpui.ScreenCaptureStream]] {
	sender, receiver := scheduler.Oneshot[scheduler.Result[gpui.ScreenCaptureStream]]()
	metadata := s.metadata()

	excludedWindows := objc.NSArrayFromIDs(nil)
	defer objc.Release(excludedWindows)
	filter := objc.Send[objc.ID](objc.Alloc(objc.GetClass("SCContentFilter")), objc.Sel("initWithDisplay:excludingWindows:"), s.scDisplay.ID(), excludedWindows)
	defer objc.Release(filter)
	configuration := objc.AllocInit(objc.GetClass("SCStreamConfiguration"))
	defer objc.Release(configuration)
	objc.SendVoid(configuration, objc.Sel("setScalesToFit:"), true)
	objc.SendVoid(configuration, objc.Sel("setPixelFormat:"), uint32(0x42475241))
	objc.SendVoid(configuration, objc.Sel("setWidth:"), uintptr(metadata.Resolution.Width))
	objc.SendVoid(configuration, objc.Sel("setHeight:"), uintptr(metadata.Resolution.Height))

	stream := objc.Own(objc.Send[objc.ID](objc.Alloc(objc.GetClass("SCStream")), objc.Sel("initWithFilter:configuration:delegate:"), filter, configuration, objc.Nil))
	output := newStreamOutput(frameCallback)

	var nsErr objc.ID
	if !msgSendAddStreamOutput(stream.ID(), objc.Sel("addStreamOutput:type:sampleHandlerQueue:error:"), output.ID(), scStreamOutputTypeScreen, objc.Nil, unsafe.Pointer(&nsErr)) {
		description := ""
		if err := objc.NSErrorFromID(nsErr); err != nil {
			description = err.LocalizedDescription
		}
		_ = sender.Send(scheduler.Result[gpui.ScreenCaptureStream]{Err: fmt.Errorf("failed to add stream output: %s", description)})
		releaseStreamOutput(output)
		stream.Release()
		return receiver
	}

	var taken atomic.Bool
	handler := objc.NewBlock(func(_ objc.Block, nsErr objc.ID) {
		if taken.Swap(true) {
			return
		}
		var result scheduler.Result[gpui.ScreenCaptureStream]
		if err := objc.NSErrorFromID(nsErr); err != nil {
			result.Err = fmt.Errorf("failed to start screen capture stream: %s", err.LocalizedDescription)
			releaseStreamOutput(output)
			stream.Release()
		} else {
			result.Value = &MacScreenCaptureStream{
				meta:           metadata,
				scStream:       stream,
				scStreamOutput: output,
			}
		}
		_ = sender.Send(result)
	})
	objc.SendVoid(stream.ID(), objc.Sel("startCaptureWithCompletionHandler:"), handler)
	handler.Release()

	return receiver
}

// Metadata returns the metadata of the captured display.
func (s *MacScreenCaptureStream) Metadata() (gpui.SourceMetadata, error) {
	return s.meta, nil
}

// Stop removes the stream output and stops the capture; it is idempotent.
func (s *MacScreenCaptureStream) Stop() {
	if s.stopped {
		return
	}
	s.stopped = true
	var nsErr objc.ID
	if !msgSendRemoveStreamOutput(s.scStream.ID(), objc.Sel("removeStreamOutput:type:error:"), s.scStreamOutput.ID(), scStreamOutputTypeScreen, unsafe.Pointer(&nsErr)) {
		if err := objc.NSErrorFromID(nsErr); err != nil {
			slog.Error("failed to remove screen capture stream output: " + err.LocalizedDescription)
		}
	}

	handler := objc.NewBlock(func(_ objc.Block, nsErr objc.ID) {
		if err := objc.NSErrorFromID(nsErr); err != nil {
			slog.Error("failed to stop screen capture stream: " + err.LocalizedDescription)
		}
	})
	objc.SendVoid(s.scStream.ID(), objc.Sel("stopCaptureWithCompletionHandler:"), handler)
	handler.Release()
	releaseStreamOutput(s.scStreamOutput)
	s.scStream.Release()
}

type screenMeta struct {
	label  gpui.SharedString
	isMain bool
}

func screenIDToHumanLabel(marker objc.MainThreadMarker) map[uint32]screenMeta {
	screens := appkit.NSScreenScreens()
	result := map[uint32]screenMeta{}
	screenNumberKey := objc.NSStringFromGo("NSScreenNumber")
	defer objc.Release(screenNumberKey)
	numberClass := objc.GetClass("NSNumber")
	for i, screen := range screens {
		description := screen.DeviceDescription()
		obj := objc.NSDictionaryObjectForKey(description, screenNumberKey)
		if obj == objc.Nil {
			continue
		}
		if !objc.Send[bool](obj, objc.Sel("isKindOfClass:"), numberClass) {
			continue
		}
		screenID := objc.Send[uint32](obj, objc.Sel("unsignedIntValue"))
		result[screenID] = screenMeta{
			label:  gpui.SharedString(screen.LocalizedName()),
			isMain: i == 0,
		}
	}
	return result
}

func getSources(marker objc.MainThreadMarker) *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[[]gpui.ScreenCaptureSource]]()
	if err := loadScreenCapture(); err != nil {
		_ = tx.Send(scheduler.Result[[]gpui.ScreenCaptureSource]{Err: err})
		return rx
	}
	screenIDToLabel := screenIDToHumanLabel(marker)

	var taken atomic.Bool
	handler := objc.NewBlock(func(_ objc.Block, content objc.ID, nsErr objc.ID) {
		if taken.Swap(true) {
			return
		}
		var result scheduler.Result[[]gpui.ScreenCaptureSource]
		if err := objc.NSErrorFromID(nsErr); err != nil {
			result.Err = fmt.Errorf("Screen share failed: %s", err.LocalizedDescription)
		} else if content != objc.Nil {
			sources := []gpui.ScreenCaptureSource{}
			for _, display := range objc.NSArrayToIDs(objc.Send[objc.ID](content, objc.Sel("displays"))) {
				id := objc.Send[uint32](display, objc.Sel("displayID"))
				meta := gpui.None[screenMeta]()
				if m, ok := screenIDToLabel[id]; ok {
					meta = gpui.Some(m)
				}
				sources = append(sources, &MacScreenCaptureSource{
					scDisplay: objc.RetainObject(display),
					meta:      meta,
				})
			}
			result.Value = sources
		} else {
			result.Err = errors.New("Screen share failed")
		}
		_ = tx.Send(result)
	})
	objc.SendVoid(
		objc.ID(objc.GetClass("SCShareableContent")),
		objc.Sel("getShareableContentExcludingDesktopWindows:onScreenWindowsOnly:completionHandler:"),
		true,
		true,
		handler,
	)
	handler.Release()
	return rx
}

func streamOutputClassRegistered() (objc.Class, error) {
	streamOutputClassOnce.Do(func() {
		streamOutputClass, streamOutputErr = objc.RegisterClass(
			"GPUIStreamOutput",
			objc.GetClass("NSObject"),
			[]string{"NSObject", "SCStreamOutput"},
			nil,
			[]objc.MethodDef{{
				Cmd: objc.Sel("stream:didOutputSampleBuffer:ofType:"),
				Fn:  streamDidOutputSampleBufferOfType,
			}},
		)
	})
	return streamOutputClass, streamOutputErr
}

func newStreamOutput(frameCallback func(frame gpui.ScreenCaptureFrame)) *objc.Retained {
	class, err := streamOutputClassRegistered()
	if err != nil {
		panic(err)
	}
	output := objc.AllocInit(class)
	streamOutputCallbacksMu.Lock()
	streamOutputCallbacks[output] = frameCallback
	streamOutputCallbacksMu.Unlock()
	return objc.Own(output)
}

func releaseStreamOutput(output *objc.Retained) {
	streamOutputCallbacksMu.Lock()
	delete(streamOutputCallbacks, output.ID())
	streamOutputCallbacksMu.Unlock()
	output.Release()
}

func streamDidOutputSampleBufferOfType(this objc.ID, _ objc.SEL, _ objc.ID, sampleBuffer objc.ID, bufferType objc.NSInteger) {
	if bufferType != scStreamOutputTypeScreen {
		return
	}
	imageBuffer := cmSampleBufferGetImageBuffer(sampleBuffer)
	if imageBuffer == 0 {
		return
	}
	streamOutputCallbacksMu.Lock()
	frameCallback := streamOutputCallbacks[this]
	streamOutputCallbacksMu.Unlock()
	if frameCallback != nil {
		frameCallback(gpui.ScreenCaptureFrame{Frame: gpui.PlatformScreenCaptureFrame(imageBuffer)})
	}
}
