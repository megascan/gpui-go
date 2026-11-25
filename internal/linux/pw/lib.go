//go:build linux

// Package pw binds the subset of libpipewire-0.3 that zed-scap's Wayland engine uses, through
// purego, plus a Go SPA pod serializer and parser for the format parameters it exchanges.
package pw

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error
)

var (
	pw_init                  func(argc unsafe.Pointer, argv unsafe.Pointer)
	pw_main_loop_new         func(props unsafe.Pointer) unsafe.Pointer
	pw_main_loop_get_loop    func(loop unsafe.Pointer) unsafe.Pointer
	pw_main_loop_destroy     func(loop unsafe.Pointer)
	pw_context_new           func(loop unsafe.Pointer, props unsafe.Pointer, userDataSize uintptr) unsafe.Pointer
	pw_context_destroy       func(ctx unsafe.Pointer)
	pw_context_connect       func(ctx unsafe.Pointer, props unsafe.Pointer, userDataSize uintptr) unsafe.Pointer
	pw_context_connect_fd    func(ctx unsafe.Pointer, fd int32, props unsafe.Pointer, userDataSize uintptr) unsafe.Pointer
	pw_core_disconnect       func(core unsafe.Pointer) int32
	pw_properties_new_dict   func(dict unsafe.Pointer) unsafe.Pointer
	pw_properties_free       func(props unsafe.Pointer)
	pw_stream_new            func(core unsafe.Pointer, name string, props unsafe.Pointer) unsafe.Pointer
	pw_stream_destroy        func(stream unsafe.Pointer)
	pw_stream_add_listener   func(stream unsafe.Pointer, hook unsafe.Pointer, events unsafe.Pointer, data uintptr)
	pw_stream_connect        func(stream unsafe.Pointer, direction uint32, targetID uint32, flags uint32, params unsafe.Pointer, nParams uint32) int32
	pw_stream_dequeue_buffer func(stream unsafe.Pointer) unsafe.Pointer
	pw_stream_queue_buffer   func(stream unsafe.Pointer, buffer unsafe.Pointer) int32

	pw_loop_enter   func(loop unsafe.Pointer)
	pw_loop_leave   func(loop unsafe.Pointer)
	pw_loop_iterate func(loop unsafe.Pointer, timeout int32) int32

	calloc func(n, size uintptr) unsafe.Pointer
	free   func(p unsafe.Pointer)
)

type binding struct {
	fn   any
	name string
}

// Load opens libpipewire-0.3.so.0 and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen("libpipewire-0.3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("pw: open libpipewire-0.3.so.0: %w", err)
			return
		}
		if loadErr = register(lib, []binding{
			{&pw_init, "pw_init"},
			{&pw_main_loop_new, "pw_main_loop_new"},
			{&pw_main_loop_get_loop, "pw_main_loop_get_loop"},
			{&pw_main_loop_destroy, "pw_main_loop_destroy"},
			{&pw_context_new, "pw_context_new"},
			{&pw_context_destroy, "pw_context_destroy"},
			{&pw_context_connect, "pw_context_connect"},
			{&pw_context_connect_fd, "pw_context_connect_fd"},
			{&pw_core_disconnect, "pw_core_disconnect"},
			{&pw_properties_new_dict, "pw_properties_new_dict"},
			{&pw_properties_free, "pw_properties_free"},
			{&pw_stream_new, "pw_stream_new"},
			{&pw_stream_destroy, "pw_stream_destroy"},
			{&pw_stream_add_listener, "pw_stream_add_listener"},
			{&pw_stream_connect, "pw_stream_connect"},
			{&pw_stream_dequeue_buffer, "pw_stream_dequeue_buffer"},
			{&pw_stream_queue_buffer, "pw_stream_queue_buffer"},
		}); loadErr != nil {
			return
		}
		if sym, err := purego.Dlsym(lib, "pw_loop_iterate"); err == nil {
			purego.RegisterFunc(&pw_loop_iterate, sym)
			if sym, err := purego.Dlsym(lib, "pw_loop_enter"); err == nil {
				purego.RegisterFunc(&pw_loop_enter, sym)
			}
			if sym, err := purego.Dlsym(lib, "pw_loop_leave"); err == nil {
				purego.RegisterFunc(&pw_loop_leave, sym)
			}
		}
		libc, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("pw: open libc.so.6: %w", err)
			return
		}
		if loadErr = register(libc, []binding{{&calloc, "calloc"}, {&free, "free"}}); loadErr != nil {
			return
		}
		callbacks.state = purego.NewCallback(stateChangedTrampoline)
		callbacks.param = purego.NewCallback(paramChangedTrampoline)
		callbacks.process = purego.NewCallback(processTrampoline)
	})
	return loadErr
}

func register(lib uintptr, bs []binding) error {
	for _, b := range bs {
		sym, err := purego.Dlsym(lib, b.name)
		if err != nil {
			return fmt.Errorf("pw: resolve %s: %w", b.name, err)
		}
		purego.RegisterFunc(b.fn, sym)
	}
	return nil
}

func ptrAt(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

type spaCallbacks struct {
	funcs unsafe.Pointer
	data  unsafe.Pointer
}

type spaInterface struct {
	typ     unsafe.Pointer
	version uint32
	cb      spaCallbacks
}

type pwLoopHead struct {
	system  unsafe.Pointer
	loop    unsafe.Pointer
	control *spaInterface
}

type spaLoopControlMethods struct {
	version uint32
	getFD   uintptr
	addHook uintptr
	enter   uintptr
	leave   uintptr
	iterate uintptr
}

func loopControlCall(loop unsafe.Pointer, pick func(*spaLoopControlMethods) uintptr, args ...uintptr) int32 {
	ctl := (*pwLoopHead)(loop).control
	methods := (*spaLoopControlMethods)(ctl.cb.funcs)
	fn := pick(methods)
	if fn == 0 {
		return 0
	}
	r, _, _ := purego.SyscallN(fn, append([]uintptr{uintptr(ctl.cb.data)}, args...)...)
	return int32(uint32(r))
}
