//go:build windows

package wgc

import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/d3d"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// Capture control errors (capture::CaptureControlError).
var (
	ErrFailedToJoinThread        = errors.New("Failed to join thread")
	ErrThreadHandleIsTaken       = errors.New("Thread handle is taken out of the struct")
	ErrFailedToPostThreadMessage = errors.New("Failed to post thread message")
)

// Capture start errors (capture::GraphicsCaptureApiError).
var (
	ErrFailedToInitWinRT                          = errors.New("Failed to initialize WinRT")
	ErrFailedToCreateDispatcherQueueController    = errors.New("Failed to create dispatcher queue controller")
	ErrFailedToShutdownDispatcherQueue            = errors.New("Failed to shut down dispatcher queue")
	ErrFailedToSetDispatcherQueueCompletedHandler = errors.New("Failed to set dispatcher queue completed handler")
	ErrItemConvertFailed                          = errors.New("Failed to convert item to `GraphicsCaptureItem`")
)

func captureControlError(err error) error {
	if err == nil {
		return nil
	}
	return &WrappedError{Msg: "Windows capture error", Err: err}
}

// Context is what a handler is created with. Device and DeviceContext are borrowed for the
// capture's lifetime.
type Context[Flags any] struct {
	// Flags are the user flags from Settings.
	Flags Flags
	// Device is the capture's D3D11 device.
	Device *d3d.ID3D11Device
	// DeviceContext is the device's immediate context.
	DeviceContext *d3d.ID3D11DeviceContext
}

// GraphicsCaptureApiHandler receives captured frames.
type GraphicsCaptureApiHandler interface {
	// OnFrameArrived is called on the capture thread for every frame.
	OnFrameArrived(frame *Frame, captureControl InternalCaptureControl) error
	// OnClosed is called when the captured item closes.
	OnClosed() error
}

// CaptureControl controls a capture running on its own thread.
type CaptureControl struct {
	thread     *captureThread
	exited     <-chan struct{}
	exitErr    *error
	haltHandle *atomic.Bool
	callback   *lockedHandler
}

// Exited returns a channel that is closed when the capture thread has ended.
func (c *CaptureControl) Exited() <-chan struct{} { return c.exited }

// ExitErr returns the error the capture thread ended with; it is valid once Exited is closed.
func (c *CaptureControl) ExitErr() error {
	<-c.exited
	return *c.exitErr
}

// IsFinished reports whether the capture thread has ended.
func (c *CaptureControl) IsFinished() bool {
	return c.thread == nil || c.thread.isFinished()
}

// HaltHandle returns the flag that stops frame delivery.
func (c *CaptureControl) HaltHandle() *atomic.Bool { return c.haltHandle }

// Callback calls f with the handler locked.
func (c *CaptureControl) Callback(f func(GraphicsCaptureApiHandler)) {
	c.callback.mu.Lock()
	defer c.callback.mu.Unlock()
	f(c.callback.h)
}

// Wait blocks until the capture thread ends.
func (c *CaptureControl) Wait() error {
	if c.thread == nil {
		return ErrThreadHandleIsTaken
	}
	t := c.thread
	c.thread = nil
	return captureControlError(t.join())
}

// Stop halts the capture, posts WM_QUIT to its thread and waits for it.
func (c *CaptureControl) Stop() error {
	c.haltHandle.Store(true)
	if c.thread == nil {
		return ErrThreadHandleIsTaken
	}
	t := c.thread
	c.thread = nil
	for {
		err := postQuit(t.id)
		if err == nil {
			break
		}
		if t.isFinished() {
			break
		}
		if com.HRESULTFromError(err) != errorInvalidThreadIDHRESULT {
			return ErrFailedToPostThreadMessage
		}
	}
	return captureControlError(t.join())
}

type captureThread struct {
	handle  windows.Handle
	id      uint32
	done    chan error
	exited  chan struct{}
	exitErr error
	once    sync.Once
	result  error
}

func (t *captureThread) isFinished() bool {
	ev, _ := windows.WaitForSingleObject(t.handle, 0)
	return ev == windows.WAIT_OBJECT_0
}

func (t *captureThread) join() error {
	t.once.Do(func() {
		t.result = <-t.done
		_, _ = windows.WaitForSingleObject(t.handle, windows.INFINITE)
		_ = windows.CloseHandle(t.handle)
	})
	return t.result
}

var (
	initMTA           sync.Once
	threadFuncs       win32.Handles[func()]
	threadStartOnce   sync.Once
	threadStartThunk  uintptr
	asyncCompletedNop = func(*IAsyncAction, int32) error {
		win32.PostQuitMessage(0)
		return nil
	}
)

func threadStart() uintptr {
	threadStartOnce.Do(func() {
		threadStartThunk = windows.NewCallback(func(parameter uintptr) uintptr {
			if f, ok := threadFuncs.Take(parameter); ok {
				f()
			}
			return 0
		})
	})
	return threadStartThunk
}

func spawnThread(f func()) (windows.Handle, uint32, error) {
	key := threadFuncs.Insert(f)
	handle, id, err := win32.CreateThread(0, threadStart(), key, 0)
	if err != nil {
		threadFuncs.Take(key)
		return 0, 0, err
	}
	return handle, id, nil
}

func pumpMessages() {
	var message win32.MSG
	for win32.GetMessageW(&message, 0, 0, 0) != 0 {
		win32.TranslateMessage(&message)
		win32.DispatchMessageW(&message)
	}
}

func runCapture[Flags any](settings Settings[Flags], newHandler func(Context[Flags]) (GraphicsCaptureApiHandler, error), started func(halt *atomic.Bool, callback *lockedHandler)) error {
	initMTA.Do(func() {
		if _, err := com.CoIncrementMTAUsage(); err != nil {
			panic("Failed to increment MTA usage: " + err.Error())
		}
	})
	if hr := roInitialize(roInitMultithreaded); hr.Failed() {
		return ErrFailedToInitWinRT
	}
	options := dispatcherQueueOptions{
		DwSize:        uint32(unsafe.Sizeof(dispatcherQueueOptions{})),
		ThreadType:    dqtypeThreadCurrent,
		ApartmentType: dqtatComNone,
	}
	controller, err := createDispatcherQueueController(options)
	if err != nil {
		return ErrFailedToCreateDispatcherQueueController
	}
	defer controller.Release()

	threadID := windows.GetCurrentThreadId()

	device, deviceContext, err := CreateD3DDevice()
	if err != nil {
		return &WrappedError{Msg: "DirectX error", Err: err}
	}
	result := &resultSlot{}
	ctx := Context[Flags]{Flags: settings.flags, Device: device, DeviceContext: deviceContext}
	h, err := newHandler(ctx)
	if err != nil {
		device.Release()
		deviceContext.Release()
		return &WrappedError{Msg: "New handler error", Err: err}
	}
	callback := &lockedHandler{h: h}

	item, itemType, err := settings.item.TryIntoCaptureItem()
	if err != nil {
		device.Release()
		deviceContext.Release()
		return ErrItemConvertFailed
	}

	capture, err := NewGraphicsCaptureApi(device, deviceContext, item, itemType, callback,
		settings.cursorCaptureSettings,
		settings.drawBorderSettings,
		settings.secondaryWindowSettings,
		settings.minimumUpdateIntervalSettings,
		settings.dirtyRegionSettings,
		settings.colorFormat,
		threadID,
		result,
	)
	if err != nil {
		return &WrappedError{Msg: "Graphics capture error", Err: err}
	}
	if err := capture.StartCapture(); err != nil {
		capture.Release()
		return &WrappedError{Msg: "Graphics capture error", Err: err}
	}

	if started != nil {
		started(capture.HaltHandle(), callback)
	}

	pumpMessages()

	asyncAction, err := controller.ShutdownQueueAsync()
	if err != nil {
		capture.Release()
		return ErrFailedToShutdownDispatcherQueue
	}
	defer asyncAction.Release()
	if err := asyncAction.SetCompleted(asyncCompletedNop); err != nil {
		capture.Release()
		return ErrFailedToSetDispatcherQueueCompletedHandler
	}

	pumpMessages()

	capture.StopCapture()

	if e := result.take(); e != nil {
		return &WrappedError{Msg: "Frame handler error", Err: e}
	}
	return nil
}

// Start runs a capture on the calling thread until it is stopped; the calling goroutine must be
// locked to its OS thread.
func Start[Flags any](settings Settings[Flags], newHandler func(Context[Flags]) (GraphicsCaptureApiHandler, error)) error {
	return runCapture(settings, newHandler, nil)
}

type startInfo struct {
	halt     *atomic.Bool
	callback *lockedHandler
}

// StartFreeThreaded runs a capture on a new native thread and returns once it has started.
func StartFreeThreaded[Flags any](settings Settings[Flags], newHandler func(Context[Flags]) (GraphicsCaptureApiHandler, error)) (*CaptureControl, error) {
	startedCh := make(chan startInfo, 1)
	t := &captureThread{done: make(chan error, 1), exited: make(chan struct{})}
	handle, id, err := spawnThread(func() {
		err := runCapture(settings, newHandler, func(halt *atomic.Bool, callback *lockedHandler) {
			startedCh <- startInfo{halt: halt, callback: callback}
		})
		t.exitErr = captureControlError(err)
		close(t.exited)
		t.done <- err
		close(startedCh)
	})
	if err != nil {
		return nil, &WrappedError{Msg: "Windows API error", Err: err}
	}
	t.handle, t.id = handle, id
	info, ok := <-startedCh
	if !ok {
		return nil, t.join()
	}
	return &CaptureControl{thread: t, exited: t.exited, exitErr: &t.exitErr, haltHandle: info.halt, callback: info.callback}, nil
}
