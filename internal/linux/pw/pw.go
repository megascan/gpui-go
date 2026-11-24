//go:build linux

package pw

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Property keys (pipewire::keys).
const (
	KeyMediaType     = "media.type"
	KeyMediaCategory = "media.category"
	KeyMediaRole     = "media.role"
)

// IDAny is PW_ID_ANY.
const IDAny uint32 = 0xffffffff

// Direction is enum spa_direction.
type Direction uint32

// Stream directions.
const (
	DirectionInput  Direction = 0
	DirectionOutput Direction = 1
)

// StreamFlags is enum pw_stream_flags.
type StreamFlags uint32

// Stream flags used by the scap engine.
const (
	StreamFlagAutoconnect StreamFlags = 1 << 0
	StreamFlagMapBuffers  StreamFlags = 1 << 2
)

// StreamState is enum pw_stream_state.
type StreamState int32

// Stream states.
const (
	StreamStateError       StreamState = -1
	StreamStateUnconnected StreamState = 0
	StreamStateConnecting  StreamState = 1
	StreamStatePaused      StreamState = 2
	StreamStateStreaming   StreamState = 3
)

// ErrCreationFailed is pipewire::Error::CreationFailed.
var ErrCreationFailed = errors.New("Creation failed")

func spaResult(r int32) error {
	if r < 0 {
		return fmt.Errorf("pw: %w", syscall.Errno(-r))
	}
	return nil
}

var initOnce sync.Once

// Init is pw_init(NULL, NULL); it loads the library first.
func Init() error {
	if err := Load(); err != nil {
		return err
	}
	initOnce.Do(func() { pw_init(nil, nil) })
	return nil
}

// Properties is an owned struct pw_properties.
type Properties struct{ ptr unsafe.Pointer }

// Dict is one key/value pair of NewProperties.
type Dict struct{ Key, Value string }

// NewProperties builds pw_properties from items (pipewire::properties!).
func NewProperties(items ...Dict) (*Properties, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	var pin runtime.Pinner
	defer pin.Unpin()
	cItems := make([]spaDictItem, len(items))
	for i, it := range items {
		k := append([]byte(it.Key), 0)
		v := append([]byte(it.Value), 0)
		pin.Pin(&k[0])
		pin.Pin(&v[0])
		cItems[i] = spaDictItem{key: &k[0], value: &v[0]}
	}
	d := &spaDict{nItems: uint32(len(cItems))}
	if len(cItems) > 0 {
		pin.Pin(&cItems[0])
		d.items = &cItems[0]
	}
	ptr := pw_properties_new_dict(unsafe.Pointer(d))
	runtime.KeepAlive(cItems)
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &Properties{ptr: ptr}, nil
}

// Free frees properties that were not handed to PipeWire.
func (p *Properties) Free() {
	if p != nil && p.ptr != nil {
		pw_properties_free(p.ptr)
		p.ptr = nil
	}
}

func (p *Properties) take() unsafe.Pointer {
	if p == nil {
		return nil
	}
	ptr := p.ptr
	p.ptr = nil
	return ptr
}

// MainLoop is struct pw_main_loop.
type MainLoop struct{ ptr unsafe.Pointer }

// NewMainLoop is pw_main_loop_new(props).
func NewMainLoop(props *Properties) (*MainLoop, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	var pp unsafe.Pointer
	if props != nil {
		pp = props.ptr
	}
	ptr := pw_main_loop_new(pp)
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &MainLoop{ptr: ptr}, nil
}

// Loop returns the main loop's pw_loop.
func (m *MainLoop) Loop() Loop { return Loop{ptr: pw_main_loop_get_loop(m.ptr)} }

// Destroy is pw_main_loop_destroy.
func (m *MainLoop) Destroy() {
	if m.ptr != nil {
		pw_main_loop_destroy(m.ptr)
		m.ptr = nil
	}
}

// Loop is a borrowed struct pw_loop.
type Loop struct{ ptr unsafe.Pointer }

// Enter is pw_loop_enter.
func (l Loop) Enter() {
	if pw_loop_enter != nil {
		pw_loop_enter(l.ptr)
		return
	}
	loopControlCall(l.ptr, func(m *spaLoopControlMethods) uintptr { return m.enter })
}

// Leave is pw_loop_leave.
func (l Loop) Leave() {
	if pw_loop_leave != nil {
		pw_loop_leave(l.ptr)
		return
	}
	loopControlCall(l.ptr, func(m *spaLoopControlMethods) uintptr { return m.leave })
}

// IterateUnguarded is pw_loop_iterate without entering the loop.
func (l Loop) IterateUnguarded(timeout time.Duration) int32 {
	ms := int32(timeout.Milliseconds())
	if pw_loop_iterate != nil {
		return pw_loop_iterate(l.ptr, ms)
	}
	return loopControlCall(l.ptr, func(m *spaLoopControlMethods) uintptr { return m.iterate }, uintptr(uint32(ms)))
}

// Iterate enters the loop, runs one iteration waiting at most timeout and leaves it
// (LoopRef::iterate).
func (l Loop) Iterate(timeout time.Duration) int32 {
	l.Enter()
	defer l.Leave()
	return l.IterateUnguarded(timeout)
}

// Context is struct pw_context.
type Context struct{ ptr unsafe.Pointer }

// NewContext is pw_context_new(loop, NULL, 0).
func NewContext(loop Loop) (*Context, error) {
	ptr := pw_context_new(loop.ptr, nil, 0)
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &Context{ptr: ptr}, nil
}

// Destroy is pw_context_destroy.
func (c *Context) Destroy() {
	if c.ptr != nil {
		pw_context_destroy(c.ptr)
		c.ptr = nil
	}
}

// Connect is pw_context_connect; props may be nil.
func (c *Context) Connect(props *Properties) (*Core, error) {
	ptr := pw_context_connect(c.ptr, props.take(), 0)
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &Core{ptr: ptr}, nil
}

// ConnectFD is pw_context_connect_fd; PipeWire takes ownership of fd.
func (c *Context) ConnectFD(fd int, props *Properties) (*Core, error) {
	ptr := pw_context_connect_fd(c.ptr, int32(fd), props.take(), 0)
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &Core{ptr: ptr}, nil
}

// Core is struct pw_core.
type Core struct{ ptr unsafe.Pointer }

// Disconnect is pw_core_disconnect.
func (c *Core) Disconnect() {
	if c.ptr != nil {
		pw_core_disconnect(c.ptr)
		c.ptr = nil
	}
}

// Stream is struct pw_stream.
type Stream struct{ ptr unsafe.Pointer }

// NewStream is pw_stream_new; the stream takes ownership of props.
func NewStream(core *Core, name string, props *Properties) (*Stream, error) {
	ptr := pw_stream_new(core.ptr, name, props.take())
	if ptr == nil {
		return nil, ErrCreationFailed
	}
	return &Stream{ptr: ptr}, nil
}

// Destroy is pw_stream_destroy.
func (s *Stream) Destroy() {
	if s.ptr != nil {
		pw_stream_destroy(s.ptr)
		s.ptr = nil
	}
}

// Connect is pw_stream_connect; id nil targets PW_ID_ANY. Each param is a serialized pod.
func (s *Stream) Connect(direction Direction, id *uint32, flags StreamFlags, params [][]byte) error {
	target := IDAny
	if id != nil {
		target = *id
	}
	var pin runtime.Pinner
	defer pin.Unpin()
	ptrs := make([]unsafe.Pointer, len(params))
	for i, p := range params {
		if len(p) == 0 {
			return syscall.EINVAL
		}
		pin.Pin(&p[0])
		ptrs[i] = unsafe.Pointer(&p[0])
	}
	var arr unsafe.Pointer
	if len(ptrs) > 0 {
		pin.Pin(&ptrs[0])
		arr = unsafe.Pointer(&ptrs[0])
	}
	r := pw_stream_connect(s.ptr, uint32(direction), target, uint32(flags), arr, uint32(len(ptrs)))
	runtime.KeepAlive(params)
	runtime.KeepAlive(ptrs)
	return spaResult(r)
}

// DequeueRawBuffer is pw_stream_dequeue_buffer; it returns nil when no buffer is ready.
func (s *Stream) DequeueRawBuffer() *Buffer {
	return (*Buffer)(pw_stream_dequeue_buffer(s.ptr))
}

// QueueRawBuffer is pw_stream_queue_buffer.
func (s *Stream) QueueRawBuffer(b *Buffer) {
	pw_stream_queue_buffer(s.ptr, unsafe.Pointer(b))
}

// StreamEvents are the stream callbacks the scap engine registers.
type StreamEvents struct {
	StateChanged func(stream *Stream, old, new StreamState, err string)
	ParamChanged func(stream *Stream, id uint32, param []byte)
	Process      func(stream *Stream)
}

// StreamListener is a registered set of StreamEvents (pipewire::stream::StreamListener).
type StreamListener struct {
	id     uintptr
	stream *Stream
	events StreamEvents
	hook   *spaHook
	table  *pwStreamEvents
}

var callbacks struct {
	state, param, process uintptr
}

var listeners struct {
	mu   sync.Mutex
	next uintptr
	m    map[uintptr]*StreamListener
}

// AddListener is pw_stream_add_listener with the given callbacks.
func (s *Stream) AddListener(events StreamEvents) (*StreamListener, error) {
	hook := (*spaHook)(calloc(1, unsafe.Sizeof(spaHook{})))
	table := (*pwStreamEvents)(calloc(1, unsafe.Sizeof(pwStreamEvents{})))
	if hook == nil || table == nil {
		free(unsafe.Pointer(hook))
		free(unsafe.Pointer(table))
		return nil, ErrCreationFailed
	}
	table.version = pwVersionStreamEvents
	if events.StateChanged != nil {
		table.stateChanged = callbacks.state
	}
	if events.ParamChanged != nil {
		table.paramChanged = callbacks.param
	}
	if events.Process != nil {
		table.process = callbacks.process
	}
	listeners.mu.Lock()
	if listeners.m == nil {
		listeners.m = map[uintptr]*StreamListener{}
	}
	listeners.next++
	l := &StreamListener{id: listeners.next, stream: s, events: events, hook: hook, table: table}
	listeners.m[l.id] = l
	listeners.mu.Unlock()
	pw_stream_add_listener(s.ptr, unsafe.Pointer(hook), unsafe.Pointer(table), l.id)
	return l, nil
}

// Remove unregisters the listener (spa_hook_remove); call it before destroying the stream.
func (l *StreamListener) Remove() {
	if l.hook == nil {
		return
	}
	listeners.mu.Lock()
	delete(listeners.m, l.id)
	listeners.mu.Unlock()
	h := l.hook
	if h.next != nil && h.prev != nil {
		(*spaHook)(h.prev).next = h.next
		(*spaHook)(h.next).prev = h.prev
	}
	free(unsafe.Pointer(h))
	free(unsafe.Pointer(l.table))
	l.hook, l.table = nil, nil
}

func lookupListener(data uintptr) *StreamListener {
	listeners.mu.Lock()
	defer listeners.mu.Unlock()
	return listeners.m[data]
}

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(ptrAt(p + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptrAt(p)), n))
}

func stateChangedTrampoline(data, old, state, errStr uintptr) {
	if l := lookupListener(data); l != nil && l.events.StateChanged != nil {
		l.events.StateChanged(l.stream, StreamState(int32(uint32(old))), StreamState(int32(uint32(state))), goString(errStr))
	}
}

func paramChangedTrampoline(data, id, param uintptr) {
	if l := lookupListener(data); l != nil && l.events.ParamChanged != nil {
		var pod []byte
		if param != 0 {
			pod = PodFromC(ptrAt(param))
		}
		l.events.ParamChanged(l.stream, uint32(id), pod)
	}
}

func processTrampoline(data uintptr) {
	if l := lookupListener(data); l != nil && l.events.Process != nil {
		l.events.Process(l.stream)
	}
}
