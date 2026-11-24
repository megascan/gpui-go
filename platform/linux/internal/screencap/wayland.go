//go:build linux

package screencap

import (
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
	"unsafe"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/pw"
)

var (
	errWaylandOutOfBuffers     = errors.New("Wayland screen capture out of buffers.")
	errUnsupportedFrameFormat  = errors.New("Unsupported frame format received")
	errWaylandStreamDropped    = errors.New("Wayland screen capture bug: stream unexpectedly dropped.")
	errMetaSizeOverflow        = errors.New("Not enough space in screen capture 'metas_values' param.")
	errValuesSizeOverflow      = errors.New("Not enough space in screen capture 'values' param.")
	errWaylandShortBuffer      = errors.New("Wayland screen capture buffer is smaller than the frame.")
	spaMetaHeaderSize          = int32(unsafe.Sizeof(pw.SpaMetaHeaderData{}))
	waylandIterateTimeout      = 100 * time.Millisecond
	waylandStartPollInterval   = 10 * time.Millisecond
	pipewireStreamName         = "scap"
	pipewireStreamMediaType    = "Video"
	pipewireStreamCategory     = "Capture"
	pipewireStreamRole         = "Screen"
	pipewireConnectStreamFlags = pw.StreamFlagAutoconnect | pw.StreamFlagMapBuffers
)

type waylandState struct {
	capturerState             atomic.Uint32
	streamStateChangedToError atomic.Bool
}

type listenerUserData struct {
	tx     *frameQueue
	format pw.VideoInfoRaw
}

func paramChangedCallback(userData *listenerUserData, id uint32, param []byte) {
	if param == nil {
		return
	}
	if id != pw.SpaParamFormat {
		return
	}
	mediaType, mediaSubtype, err := pw.FormatParse(param)
	if err != nil {
		return
	}
	if mediaType != pw.SpaMediaTypeVideo || mediaSubtype != pw.SpaMediaSubtypeRaw {
		return
	}
	if err := userData.format.Parse(param); err != nil {
		panic(fmt.Sprintf("Failed to parse format parameter: %v", err))
	}
}

func stateChangedCallback(state *waylandState, _ pw.StreamState, newState pw.StreamState, errMsg string) {
	if newState == pw.StreamStateError {
		slog.Debug(fmt.Sprintf("pipewire: State changed to error(%s)", errMsg))
		state.streamStateChangedToError.Store(true)
	}
}

func getTimestamp(buffer *pw.SpaBuffer) int64 {
	if buffer.NMetas == 0 {
		return 0
	}
	metas := unsafe.Slice(buffer.Metas, buffer.NMetas)
	for i := range metas {
		if metas[i].Type == pw.SpaMetaHeader {
			return (*pw.SpaMetaHeaderData)(metas[i].Data).Pts
		}
	}
	return 0
}

func processCallback(stream *pw.Stream, userData *listenerUserData) {
	buffer := stream.DequeueRawBuffer()
	frame, err := processCallbackImpl(buffer, userData)
	if frame != nil || err != nil {
		userData.tx.send(frame, err)
	}
	if buffer != nil {
		stream.QueueRawBuffer(buffer)
	}
}

func processCallbackImpl(pwBuffer *pw.Buffer, userData *listenerUserData) (gpui.ScapFrame, error) {
	if pwBuffer == nil {
		return nil, errWaylandOutOfBuffers
	}
	buffer := pwBuffer.Buffer
	if buffer == nil {
		slog.Error("Buffer pointer unexpectedly null in Wayland screen capture.")
		return nil, nil
	}

	timestamp := getTimestamp(buffer)

	if buffer.NDatas < 1 {
		return nil, nil
	}
	frameSize := userData.format.Size
	data := buffer.Datas
	if data.Data == nil {
		return nil, nil
	}
	var bpp uint32
	switch userData.format.Format {
	case pw.VideoFormatRGB:
		bpp = 3
	case pw.VideoFormatRGBx, pw.VideoFormatXBGR, pw.VideoFormatBGRx:
		bpp = 4
	default:
		return nil, errUnsupportedFrameFormat
	}
	frameData, err := copyFrameData(data, frameSize.Width, frameSize.Height, bpp)
	if err != nil {
		return nil, err
	}

	pixels := gpui.ScapPixels{
		DisplayTime: uint64(timestamp),
		Width:       int32(frameSize.Width),
		Height:      int32(frameSize.Height),
		Data:        frameData,
	}
	switch userData.format.Format {
	case pw.VideoFormatRGBx:
		return gpui.ScapFrameRGBx{ScapPixels: pixels}, nil
	case pw.VideoFormatRGB:
		return gpui.ScapFrameRGB{ScapPixels: pixels}, nil
	case pw.VideoFormatXBGR:
		return gpui.ScapFrameXBGR{ScapPixels: pixels}, nil
	}
	return gpui.ScapFrameBGRx{ScapPixels: pixels}, nil
}

func copyFrameData(data *pw.SpaData, width, height, bpp uint32) ([]byte, error) {
	src := unsafe.Slice((*byte)(data.Data), data.MaxSize)
	if data.Chunk == nil {
		return append([]byte(nil), src...), nil
	}
	chunk := data.Chunk
	rowBytes := uint64(width) * uint64(bpp)
	stride := uint64(rowBytes)
	if chunk.Stride > 0 {
		stride = uint64(chunk.Stride)
	}
	offset := uint64(chunk.Offset) % max(uint64(data.MaxSize), 1)
	end := min(offset+uint64(chunk.Size), uint64(data.MaxSize))
	if chunk.Size == 0 {
		end = uint64(data.MaxSize)
	}
	if height == 0 || rowBytes == 0 {
		return []byte{}, nil
	}
	if stride < rowBytes || offset+uint64(height-1)*stride+rowBytes > end {
		return nil, errWaylandShortBuffer
	}
	out := make([]byte, rowBytes*uint64(height))
	for row := range uint64(height) {
		s := offset + row*stride
		copy(out[row*rowBytes:(row+1)*rowBytes], src[s:s+rowBytes])
	}
	return out, nil
}

type pipewireSession struct {
	mainloop *pw.MainLoop
	context  *pw.Context
	core     *pw.Core
	stream   *pw.Stream
	listener *pw.StreamListener
}

func (s *pipewireSession) destroy() {
	if s.listener != nil {
		s.listener.Remove()
	}
	if s.stream != nil {
		s.stream.Destroy()
	}
	if s.core != nil {
		s.core.Disconnect()
	}
	if s.context != nil {
		s.context.Destroy()
	}
	if s.mainloop != nil {
		s.mainloop.Destroy()
	}
}

func enumFormatParam(fps uint32) pw.Object {
	return pw.Object{Type: pw.SpaTypeObjectFormat, ID: pw.SpaParamEnumFormat, Properties: []pw.Property{
		{Key: pw.SpaFormatMediaType, Value: pw.ID(pw.SpaMediaTypeVideo)},
		{Key: pw.SpaFormatMediaSubtype, Value: pw.ID(pw.SpaMediaSubtypeRaw)},
		{Key: pw.SpaFormatVideoFormat, Value: pw.Choice{Type: pw.SpaChoiceEnum, ChildType: pw.SpaTypeID, Values: []pw.Value{
			pw.ID(pw.VideoFormatRGB), pw.ID(pw.VideoFormatRGBx), pw.ID(pw.VideoFormatBGRx),
		}}},
		{Key: pw.SpaFormatVideoSize, Value: pw.Choice{Type: pw.SpaChoiceRange, ChildType: pw.SpaTypeRectangle, Values: []pw.Value{
			pw.RectangleValue{Width: 128, Height: 128},
			pw.RectangleValue{Width: 1, Height: 1},
			pw.RectangleValue{Width: 4096, Height: 4096},
		}}},
		{Key: pw.SpaFormatVideoFramerate, Value: pw.Choice{Type: pw.SpaChoiceRange, ChildType: pw.SpaTypeFraction, Values: []pw.Value{
			pw.FractionValue{Num: fps, Denom: 1},
			pw.FractionValue{Num: 0, Denom: 1},
			pw.FractionValue{Num: 1000, Denom: 1},
		}}},
	}}
}

func metaParam() pw.Object {
	return pw.Object{Type: pw.SpaTypeObjectParamMeta, ID: pw.SpaParamMeta, Properties: []pw.Property{
		{Key: pw.SpaParamMetaType, Value: pw.ID(pw.SpaMetaHeader)},
		{Key: pw.SpaParamMetaSize, Value: pw.Int(spaMetaHeaderSize)},
	}}
}

func startPipewireCapturer(state *waylandState, options gpui.ScapCapturerOptions, tx *frameQueue, streamID uint32) (*pipewireSession, error) {
	if err := pw.Init(); err != nil {
		return nil, err
	}
	s := &pipewireSession{}
	fail := func(err error) (*pipewireSession, error) {
		s.destroy()
		return nil, err
	}
	var err error
	if s.mainloop, err = pw.NewMainLoop(nil); err != nil {
		return fail(err)
	}
	if s.context, err = pw.NewContext(s.mainloop.Loop()); err != nil {
		return fail(err)
	}
	if s.core, err = s.context.Connect(nil); err != nil {
		return fail(err)
	}

	userData := &listenerUserData{tx: tx}

	props, err := pw.NewProperties(
		pw.Dict{Key: pw.KeyMediaType, Value: pipewireStreamMediaType},
		pw.Dict{Key: pw.KeyMediaCategory, Value: pipewireStreamCategory},
		pw.Dict{Key: pw.KeyMediaRole, Value: pipewireStreamRole},
	)
	if err != nil {
		return fail(err)
	}
	if s.stream, err = pw.NewStream(s.core, pipewireStreamName, props); err != nil {
		return fail(err)
	}

	if s.listener, err = s.stream.AddListener(pw.StreamEvents{
		StateChanged: func(_ *pw.Stream, old, newState pw.StreamState, errMsg string) {
			stateChangedCallback(state, old, newState, errMsg)
		},
		ParamChanged: func(_ *pw.Stream, id uint32, param []byte) { paramChangedCallback(userData, id, param) },
		Process:      func(stream *pw.Stream) { processCallback(stream, userData) },
	}); err != nil {
		return fail(err)
	}

	values, err := pw.Serialize(enumFormatParam(options.FPS))
	if err != nil {
		return fail(err)
	}
	metasValues, err := pw.Serialize(metaParam())
	if err != nil {
		return fail(err)
	}
	if len(values) < 8 {
		return fail(errValuesSizeOverflow)
	}
	if len(metasValues) < 8 {
		return fail(errMetaSizeOverflow)
	}

	if err := s.stream.Connect(pw.DirectionInput, &streamID, pipewireConnectStreamFlags, [][]byte{values, metasValues}); err != nil {
		return fail(err)
	}
	return s, nil
}

func pipewireCapturer(state *waylandState, options gpui.ScapCapturerOptions, tx *frameQueue, readySender chan<- error, streamID uint32) {
	defer tx.close()
	session, err := startPipewireCapturer(state, options, tx, streamID)
	readySender <- err
	if err != nil {
		return
	}
	defer session.destroy()

	for state.capturerState.Load() == 0 {
		time.Sleep(waylandStartPollInterval)
	}

	pwLoop := session.mainloop.Loop()
	for state.capturerState.Load() == 1 && !state.streamStateChangedToError.Load() {
		pwLoop.Iterate(waylandIterateTimeout)
	}
}

// WaylandCapturer captures the stream the user picks in the ScreenCast portal dialog.
type WaylandCapturer struct {
	capturerDone chan struct{}
	connection   *godbus.Conn
	state        *waylandState
}

// NewWaylandCapturer opens a private session-bus connection, runs the portal handshake and
// starts the (paused) PipeWire capture goroutine.
func NewWaylandCapturer(options gpui.ScapCapturerOptions, tx *frameQueue) (*WaylandCapturer, error) {
	connection, err := godbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("Failed to create dbus connection: %w", err)
	}
	return newWaylandCapturerOn(connection, options, tx)
}

func newWaylandCapturerOn(connection *godbus.Conn, options gpui.ScapCapturerOptions, tx *frameQueue) (*WaylandCapturer, error) {
	portal, err := NewScreenCastPortal(connection).ShowCursor(options.ShowCursor)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("Unsupported screen capture cursor display mode: %w", err)
	}
	stream, err := portal.CreateStream()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("Failed to get screen capture stream: %w", err)
	}
	streamID := stream.PwNodeID()

	state := &waylandState{}
	readySender := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(readySender)
		pipewireCapturer(state, options, tx, readySender, streamID)
	}()

	err, ok := <-readySender
	if !ok {
		_ = connection.Close()
		return nil, errWaylandStreamDropped
	}
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	return &WaylandCapturer{capturerDone: done, connection: connection, state: state}, nil
}

// StartCapture starts iterating the PipeWire loop.
func (c *WaylandCapturer) StartCapture() { c.state.capturerState.Store(1) }

// StopCapture stops the loop, waits for the capture goroutine and resets the capturer's state.
func (c *WaylandCapturer) StopCapture() {
	c.state.capturerState.Store(2)
	if c.capturerDone != nil {
		<-c.capturerDone
		c.capturerDone = nil
	}
	c.state.capturerState.Store(0)
	c.state.streamStateChangedToError.Store(false)
}

// Target is always unknown on Wayland: the portal picks it.
func (c *WaylandCapturer) Target() (gpui.ScapTarget, bool) { return nil, false }

func (c *WaylandCapturer) close() {
	if c.capturerDone != nil {
		c.state.capturerState.Store(2)
		<-c.capturerDone
		c.capturerDone = nil
	}
	if c.connection != nil {
		_ = c.connection.Close()
		c.connection = nil
	}
}
