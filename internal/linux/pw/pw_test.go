//go:build linux

package pw

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"
	"time"
	"unsafe"
)

func TestLayout_MatchesPipeWireHeaders(t *testing.T) {
	cases := []struct {
		name      string
		got, want uintptr
	}{
		{"sizeof(spa_hook)", unsafe.Sizeof(spaHook{}), 48},
		{"sizeof(pw_stream_events)", unsafe.Sizeof(pwStreamEvents{}), 96},
		{"offsetof(pw_stream_events.state_changed)", unsafe.Offsetof(pwStreamEvents{}.stateChanged), 16},
		{"offsetof(pw_stream_events.param_changed)", unsafe.Offsetof(pwStreamEvents{}.paramChanged), 40},
		{"offsetof(pw_stream_events.process)", unsafe.Offsetof(pwStreamEvents{}.process), 64},
		{"sizeof(pw_buffer)", unsafe.Sizeof(Buffer{}), 40},
		{"sizeof(spa_buffer)", unsafe.Sizeof(SpaBuffer{}), 24},
		{"offsetof(spa_buffer.metas)", unsafe.Offsetof(SpaBuffer{}.Metas), 8},
		{"offsetof(spa_buffer.datas)", unsafe.Offsetof(SpaBuffer{}.Datas), 16},
		{"sizeof(spa_meta)", unsafe.Sizeof(SpaMeta{}), 16},
		{"offsetof(spa_meta.data)", unsafe.Offsetof(SpaMeta{}.Data), 8},
		{"sizeof(spa_data)", unsafe.Sizeof(SpaData{}), 40},
		{"offsetof(spa_data.maxsize)", unsafe.Offsetof(SpaData{}.MaxSize), 20},
		{"offsetof(spa_data.data)", unsafe.Offsetof(SpaData{}.Data), 24},
		{"offsetof(spa_data.chunk)", unsafe.Offsetof(SpaData{}.Chunk), 32},
		{"sizeof(spa_chunk)", unsafe.Sizeof(SpaChunk{}), 16},
		{"sizeof(spa_meta_header)", unsafe.Sizeof(SpaMetaHeaderData{}), 32},
		{"offsetof(spa_meta_header.pts)", unsafe.Offsetof(SpaMetaHeaderData{}.Pts), 8},
		{"sizeof(spa_dict)", unsafe.Sizeof(spaDict{}), 16},
		{"sizeof(spa_dict_item)", unsafe.Sizeof(spaDictItem{}), 16},
		{"offsetof(pw_loop.control)", unsafe.Offsetof(pwLoopHead{}.control), 16},
		{"offsetof(spa_interface.cb)", unsafe.Offsetof(spaInterface{}.cb), 16},
		{"offsetof(spa_loop_control_methods.enter)", unsafe.Offsetof(spaLoopControlMethods{}.enter), 24},
		{"offsetof(spa_loop_control_methods.leave)", unsafe.Offsetof(spaLoopControlMethods{}.leave), 32},
		{"offsetof(spa_loop_control_methods.iterate)", unsafe.Offsetof(spaLoopControlMethods{}.iterate), 40},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

const (
	goldenEnumFormat = "d80000000f000000030004000300000001000000000000000400000003000000020000000000000002000000000000000400000003000000010000000000000001000200000000002000000013000000030000000000000004000000030000000f0000000b0000000700000008000000030002000000000028000000130000000100000000000000080000000a000000800000008000000001000000010000000010000000100000040002000000000028000000130000000100000000000000080000000b0000001e000000010000000000000001000000e803000001000000"
	goldenMeta       = "380000000f0000000500040006000000010000000000000004000000030000000100000000000000020000000000000004000000040000002000000000000000"
	goldenFormat     = "800000000f00000003000400040000000100000000000000040000000300000002000000000000000200000000000000040000000300000001000000000000000100020000000000040000000300000008000000000000000300020000000000080000000a00000080070000380400000400020000000000080000000b0000003c00000001000000"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSpa_SerializeMatchesSpaPodBuilder(t *testing.T) {
	enum := Object{Type: SpaTypeObjectFormat, ID: SpaParamEnumFormat, Properties: []Property{
		{Key: SpaFormatMediaType, Value: ID(SpaMediaTypeVideo)},
		{Key: SpaFormatMediaSubtype, Value: ID(SpaMediaSubtypeRaw)},
		{Key: SpaFormatVideoFormat, Value: Choice{Type: SpaChoiceEnum, ChildType: SpaTypeID, Values: []Value{
			ID(VideoFormatRGB), ID(VideoFormatRGBA), ID(VideoFormatRGBx), ID(VideoFormatBGRx),
		}}},
		{Key: SpaFormatVideoSize, Value: Choice{Type: SpaChoiceRange, ChildType: SpaTypeRectangle, Values: []Value{
			RectangleValue{128, 128}, RectangleValue{1, 1}, RectangleValue{4096, 4096},
		}}},
		{Key: SpaFormatVideoFramerate, Value: Choice{Type: SpaChoiceRange, ChildType: SpaTypeFraction, Values: []Value{
			FractionValue{30, 1}, FractionValue{0, 1}, FractionValue{1000, 1},
		}}},
	}}
	got, err := Serialize(enum)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustHex(t, goldenEnumFormat); !bytes.Equal(got, want) {
		t.Fatalf("enum format pod\n got %x\nwant %x", got, want)
	}
	meta := Object{Type: SpaTypeObjectParamMeta, ID: SpaParamMeta, Properties: []Property{
		{Key: SpaParamMetaType, Value: ID(SpaMetaHeader)},
		{Key: SpaParamMetaSize, Value: Int(int32(unsafe.Sizeof(SpaMetaHeaderData{})))},
	}}
	got, err = Serialize(meta)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustHex(t, goldenMeta); !bytes.Equal(got, want) {
		t.Fatalf("meta pod\n got %x\nwant %x", got, want)
	}
}

func TestSpa_ParseFormatMatchesSpaFormatVideoRawBuild(t *testing.T) {
	pod := mustHex(t, goldenFormat)
	mt, mst, err := FormatParse(pod)
	if err != nil || mt != SpaMediaTypeVideo || mst != SpaMediaSubtypeRaw {
		t.Fatalf("FormatParse = %d, %d, %v", mt, mst, err)
	}
	var info VideoInfoRaw
	if err := info.Parse(pod); err != nil {
		t.Fatal(err)
	}
	if info.Format != VideoFormatBGRx || info.Size != (Rectangle{1920, 1080}) || info.Framerate != (Fraction{60, 1}) {
		t.Fatalf("info = %+v", info)
	}
	enum := mustHex(t, goldenEnumFormat)
	var choice VideoInfoRaw
	if err := choice.Parse(enum); err != nil {
		t.Fatal(err)
	}
	if choice.Format != VideoFormatUnknown || choice.Size != (Rectangle{}) {
		t.Fatalf("optional fields with non-None choices must be skipped: %+v", choice)
	}
	meta := mustHex(t, goldenMeta)
	if _, _, err := FormatParse(meta); err == nil {
		t.Fatal("FormatParse accepted a ParamMeta object")
	}
}

func TestPw_MainLoopWithoutDaemon(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("libpipewire-0.3 unavailable: %v", err)
	}
	props, err := NewProperties(Dict{KeyMediaType, "Video"}, Dict{KeyMediaRole, "Screen"})
	if err != nil {
		t.Fatal(err)
	}
	props.Free()
	ml, err := NewMainLoop(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ml.Destroy()
	if r := ml.Loop().Iterate(0); r < 0 {
		t.Fatalf("iterate = %d", r)
	}
}

func TestPw_StreamConnectsToDaemon(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" {
		t.Skip("needs a PipeWire daemon; set GPUI_TEST_DISPLAY=1")
	}
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	ml, err := NewMainLoop(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ml.Destroy()
	ctx, err := NewContext(ml.Loop())
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Destroy()
	core, err := ctx.Connect(nil)
	if err != nil {
		t.Skipf("no PipeWire daemon: %v", err)
	}
	defer core.Disconnect()
	props, err := NewProperties(Dict{KeyMediaType, "Video"}, Dict{KeyMediaCategory, "Capture"}, Dict{KeyMediaRole, "Screen"})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := NewStream(core, "gpui-test", props)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Destroy()
	var states []StreamState
	l, err := stream.AddListener(StreamEvents{StateChanged: func(_ *Stream, _, s StreamState, _ string) { states = append(states, s) }})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Remove()
	pod, err := Serialize(Object{Type: SpaTypeObjectParamMeta, ID: SpaParamMeta, Properties: []Property{
		{Key: SpaParamMetaType, Value: ID(SpaMetaHeader)},
		{Key: SpaParamMetaSize, Value: Int(32)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	id := IDAny
	if err := stream.Connect(DirectionInput, &id, StreamFlagMapBuffers, [][]byte{pod}); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		ml.Loop().Iterate(10 * time.Millisecond)
	}
	if len(states) == 0 {
		t.Fatal("no state_changed callback")
	}
}
