//go:build windows

package windows

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"unicode/utf16"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

// DialogOwner serializes the native dialogs of one window and cancels them when it closes.
type DialogOwner struct {
	hwnd          HWND
	closed        *atomic.Bool
	serialization dialogMutex
}

// NewDialogOwner returns the dialog owner of hwnd.
func NewDialogOwner(hwnd HWND) *DialogOwner {
	return &DialogOwner{
		hwnd:   hwnd,
		closed: new(atomic.Bool),
	}
}

func (o *DialogOwner) close() {
	o.closed.Store(true)
}

func (o *DialogOwner) whenIdle() scheduler.Future[struct{}] {
	lock := o.serialization.lock()
	return scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		guard, ok := lock.Poll(cx)
		if !ok {
			return struct{}{}, false
		}
		guard.Close()
		return struct{}{}, true
	})
}

type dialogMutex struct {
	mu      sync.Mutex
	locked  bool
	nextID  uint64
	waiters map[uint64]scheduler.Waker
}

type dialogGuard struct {
	m      *dialogMutex
	closed bool
}

func (m *dialogMutex) tryLock() (*dialogGuard, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.locked {
		return nil, false
	}
	m.locked = true
	return &dialogGuard{m: m}, true
}

type dialogLockFuture struct {
	m      *dialogMutex
	id     uint64
	queued bool
	done   bool
}

func (m *dialogMutex) lock() *dialogLockFuture {
	return &dialogLockFuture{m: m}
}

func (f *dialogLockFuture) Poll(cx *scheduler.Context) (*dialogGuard, bool) {
	m := f.m
	m.mu.Lock()
	if !m.locked {
		m.locked = true
		f.removeLocked()
		f.done = true
		m.mu.Unlock()
		return &dialogGuard{m: m}, true
	}
	if !f.queued {
		m.nextID++
		f.id = m.nextID
		f.queued = true
	}
	if m.waiters == nil {
		m.waiters = map[uint64]scheduler.Waker{}
	}
	if old, ok := m.waiters[f.id]; ok {
		old.Drop()
	}
	m.waiters[f.id] = cx.Waker().Clone()
	m.mu.Unlock()
	return nil, false
}

func (f *dialogLockFuture) removeLocked() {
	if !f.queued {
		return
	}
	if w, ok := f.m.waiters[f.id]; ok {
		w.Drop()
		delete(f.m.waiters, f.id)
	}
	f.queued = false
}

func (f *dialogLockFuture) Cancel() {
	f.m.mu.Lock()
	f.removeLocked()
	f.m.mu.Unlock()
}

func (g *dialogGuard) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	m := g.m
	m.mu.Lock()
	m.locked = false
	waiters := m.waiters
	m.waiters = nil
	m.mu.Unlock()
	for _, w := range waiters {
		w.Wake()
	}
}

type cancellation struct {
	requested   *atomic.Bool
	ownerClosed *atomic.Bool
}

func newCancellation(owner *DialogOwner) cancellation {
	c := cancellation{requested: new(atomic.Bool)}
	if owner != nil {
		c.ownerClosed = owner.closed
	}
	return c
}

func (c cancellation) isRequested() bool {
	return c.requested.Load() || (c.ownerClosed != nil && c.ownerClosed.Load())
}

type dialogResultState[T any] struct {
	mu          sync.Mutex
	canceled    bool
	cancelWaker scheduler.Waker
}

// DialogReceiver is the receiving half of showDialog's result. Close drops it, which cancels the
// dialog if it is still open.
type DialogReceiver[T any] struct {
	rx    *scheduler.OneshotReceiver[scheduler.Result[T]]
	state *dialogResultState[T]
}

// Poll resolves to the dialog's result, or to scheduler.ErrCanceled if the task was dropped.
func (r *DialogReceiver[T]) Poll(cx *scheduler.Context) (scheduler.Result[scheduler.Result[T]], bool) {
	return r.rx.Poll(cx)
}

// Recv awaits the dialog's result.
func (r *DialogReceiver[T]) Recv(cx scheduler.Awaiter) (T, error) {
	defer r.Close()
	res := scheduler.Await[scheduler.Result[scheduler.Result[T]]](cx, scheduler.Unowned[scheduler.Result[scheduler.Result[T]]](r))
	if res.Err != nil {
		var zero T
		return zero, res.Err
	}
	return res.Value.Value, res.Value.Err
}

// Close drops the receiver and requests cancellation of the dialog.
func (r *DialogReceiver[T]) Close() {
	r.rx.Close()
	s := r.state
	s.mu.Lock()
	if s.canceled {
		s.mu.Unlock()
		return
	}
	s.canceled = true
	w := s.cancelWaker
	s.cancelWaker = nil
	s.mu.Unlock()
	if w != nil {
		w.Wake()
	}
}

// Cancel is Close.
func (r *DialogReceiver[T]) Cancel() {
	r.Close()
}

func (s *dialogResultState[T]) cancellationFuture() scheduler.Future[struct{}] {
	return scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.canceled {
			return struct{}{}, true
		}
		if s.cancelWaker != nil {
			s.cancelWaker.Drop()
		}
		s.cancelWaker = cx.Waker().Clone()
		return struct{}{}, false
	})
}

func (s *dialogResultState[T]) isCanceled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.canceled
}

func showDialog[T any](owner *DialogOwner, executor gpui.ForegroundExecutor, dialog func(HWND) (T, error)) *DialogReceiver[T] {
	sender, receiver := scheduler.Oneshot[scheduler.Result[T]]()
	state := &dialogResultState[T]{}
	result := &DialogReceiver[T]{rx: receiver, state: state}
	c := newCancellation(owner)
	if c.isRequested() {
		_ = sender.Send(scheduler.Result[T]{Err: errors.New("dialog owner has closed")})
		return result
	}
	var hwnd gpui.Option[SafeHwnd]
	if owner != nil {
		hwnd = gpui.Some(SafeHwndFrom(owner.hwnd))
	}
	executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer sender.Close()
		var serialization *dialogGuard
		if owner != nil {
			won := scheduler.Select(s,
				scheduler.Case[*dialogGuard](owner.serialization.lock(), func(g *dialogGuard) { serialization = g }),
				scheduler.Case(state.cancellationFuture(), nil),
			)
			if won == 1 {
				return struct{}{}, nil
			}
		}
		if state.isCanceled() || c.isRequested() {
			serialization.Close()
			return struct{}{}, nil
		}
		resultSender, resultReceiver := scheduler.Oneshot[scheduler.Result[T]]()
		threadCancellation := c
		err := spawnDialogThread(func() {
			v, err := func() (T, error) {
				var zero T
				apartment, err := newApartment()
				if err != nil {
					return zero, err
				}
				defer apartment.Close()
				timer, err := newCancellationTimer(threadCancellation)
				if err != nil {
					return zero, err
				}
				defer timer.Close()
				if threadCancellation.isRequested() {
					return zero, errors.New("dialog cancelled")
				}
				raw := HWND(0)
				if h, ok := hwnd.Get(); ok {
					raw = h.AsRaw()
				}
				v, err := dialog(raw)
				if threadCancellation.isRequested() {
					return zero, errors.New("dialog cancelled")
				}
				return v, err
			}()
			_ = resultSender.Send(scheduler.Result[T]{Value: v, Err: err})
		})
		if err != nil {
			serialization.Close()
			_ = sender.Send(scheduler.Result[T]{Err: err})
			return struct{}{}, nil
		}
		var res scheduler.Result[scheduler.Result[T]]
		won := scheduler.Select(s,
			scheduler.Case(scheduler.Unowned[scheduler.Result[scheduler.Result[T]]](resultReceiver), func(r scheduler.Result[scheduler.Result[T]]) { res = r }),
			scheduler.Case(state.cancellationFuture(), nil),
		)
		if won == 1 {
			c.requested.Store(true)
			res = scheduler.Await[scheduler.Result[scheduler.Result[T]]](s, resultReceiver)
		} else {
			resultReceiver.Close()
		}
		serialization.Close()
		if res.Err != nil {
			_ = sender.Send(scheduler.Result[T]{Err: res.Err})
		} else {
			_ = sender.Send(res.Value)
		}
		return struct{}{}, nil
	}).Detach()
	return result
}

type apartment struct{}

func newApartment() (apartment, error) {
	if err := com.CoInitializeEx(com.COINIT_APARTMENTTHREADED); err != nil {
		return apartment{}, err
	}
	return apartment{}, nil
}

func (apartment) Close() {
	com.CoUninitialize()
}

var cancellationSlots sync.Map

func setThreadCancellation(c cancellation) {
	cancellationSlots.Store(xwin.GetCurrentThreadId(), c)
}

func takeThreadCancellation() {
	cancellationSlots.Delete(xwin.GetCurrentThreadId())
}

type cancellationTimer struct {
	id uintptr
}

func newCancellationTimer(c cancellation) (*cancellationTimer, error) {
	timer := win32.SetTimer(0, 0, 100, dialogCallbacks().cancelDialog)
	if timer == 0 {
		return nil, fmt.Errorf("starting native dialog cancellation timer: %w", xwin.GetLastError())
	}
	setThreadCancellation(c)
	return &cancellationTimer{id: timer}, nil
}

func (t *cancellationTimer) Close() {
	takeThreadCancellation()
	if err := win32.KillTimer(0, t.id); err != nil {
		gpuiutil.LogErr(err)
	}
}

func cancelDialog(_ HWND, _ uint32, _ uintptr, _ uint32) {
	if cancellationRequested() {
		logLastError(win32.EnumThreadWindows(xwin.GetCurrentThreadId(), dialogCallbacks().closeDialog, 0), "EnumThreadWindows")
	}
}

func cancellationRequested() bool {
	v, ok := cancellationSlots.Load(xwin.GetCurrentThreadId())
	if !ok {
		return false
	}
	return v.(cancellation).isRequested()
}

func taskDialogCallback(_ HWND, notification int32, button win32.WPARAM, _ win32.LPARAM, hasCancelButton int) com.HRESULT {
	if notification == win32.TDN_BUTTON_CLICKED &&
		button == win32.WPARAM(win32.IDCANCEL) &&
		hasCancelButton == 0 &&
		!cancellationRequested() {
		return com.S_FALSE
	}
	return com.S_OK
}

func closeDialog(hwnd HWND, _ win32.LPARAM) uintptr {
	var className [32]uint16
	length, _ := win32.GetClassNameW(hwnd, className[:])
	if int(length) <= len(className) && slices.Equal(className[:length], dialogClassName) {
		if err := win32.PostMessageW(hwnd, win32.WM_CLOSE, 0, 0); err != nil {
			gpuiutil.LogErr(err)
		}
	}
	return 1
}

var dialogClassName = utf16.Encode([]rune("#32770"))

type dialogCallbackSet struct {
	cancelDialog       uintptr
	closeDialog        uintptr
	taskDialogCallback uintptr
	threadStart        uintptr
}

var (
	dialogCallbacksOnce sync.Once
	dialogCallbackAddrs dialogCallbackSet
	dialogThreadFuncs   win32.Handles[func()]
)

func dialogCallbacks() dialogCallbackSet {
	dialogCallbacksOnce.Do(func() {
		dialogCallbackAddrs = dialogCallbackSet{
			cancelDialog: xwin.NewCallback(func(hwnd, msg, id, time uintptr) uintptr {
				cancelDialog(HWND(hwnd), uint32(msg), id, uint32(time))
				return 0
			}),
			closeDialog: xwin.NewCallback(func(hwnd, lparam uintptr) uintptr {
				return closeDialog(HWND(hwnd), win32.LPARAM(lparam))
			}),
			taskDialogCallback: xwin.NewCallback(func(hwnd, notification, button, lparam, refData uintptr) uintptr {
				return taskDialogCallback(HWND(hwnd), int32(notification), win32.WPARAM(button), win32.LPARAM(lparam), int(refData)).Uintptr()
			}),
			threadStart: xwin.NewCallback(func(parameter uintptr) uintptr {
				if f, ok := dialogThreadFuncs.Take(parameter); ok {
					f()
				}
				return 0
			}),
		}
	})
	return dialogCallbackAddrs
}

func taskDialogCallbackPtr() uintptr {
	return dialogCallbacks().taskDialogCallback
}

func spawnDialogThread(f func()) error {
	start := dialogCallbacks().threadStart
	key := dialogThreadFuncs.Insert(f)
	handle, _, err := win32.CreateThread(0, start, key, 0)
	if err != nil {
		dialogThreadFuncs.Take(key)
		return err
	}
	_ = xwin.CloseHandle(handle)
	return nil
}
