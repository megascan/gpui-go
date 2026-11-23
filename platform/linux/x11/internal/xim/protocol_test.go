//go:build linux

package xim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

var openReplyFixture = []byte{
	0x1f, 0x00, 0x59, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x0f, 0x00, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x53, 0x74, 0x79, 0x6c, 0x65, 0x00, 0x00, 0x00,
	0x44, 0x01, 0x00, 0x00, 0x01, 0x00, 0x03, 0x00, 0x0a, 0x00, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x53,
	0x74, 0x79, 0x6c, 0x65, 0x02, 0x00, 0x05, 0x00, 0x0c, 0x00, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x00, 0x00, 0x03, 0x00, 0x05, 0x00, 0x0b, 0x00, 0x66, 0x6f,
	0x63, 0x75, 0x73, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x00, 0x00, 0x00, 0x04, 0x00, 0x03, 0x00,
	0x0c, 0x00, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x00, 0x00,
	0x05, 0x00, 0xff, 0x7f, 0x11, 0x00, 0x70, 0x72, 0x65, 0x65, 0x64, 0x69, 0x74, 0x41, 0x74, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x00, 0x06, 0x00, 0xff, 0x7f, 0x10, 0x00, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x41, 0x74, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x65, 0x73, 0x00, 0x00,
	0x07, 0x00, 0x0d, 0x00, 0x07, 0x00, 0x66, 0x6f, 0x6e, 0x74, 0x53, 0x65, 0x74, 0x00, 0x00, 0x00,
	0x08, 0x00, 0x0b, 0x00, 0x04, 0x00, 0x61, 0x72, 0x65, 0x61, 0x00, 0x00, 0x09, 0x00, 0x0b, 0x00,
	0x0a, 0x00, 0x61, 0x72, 0x65, 0x61, 0x4e, 0x65, 0x65, 0x64, 0x65, 0x64, 0x0a, 0x00, 0x03, 0x00,
	0x08, 0x00, 0x63, 0x6f, 0x6c, 0x6f, 0x72, 0x4d, 0x61, 0x70, 0x00, 0x00, 0x0b, 0x00, 0x03, 0x00,
	0x0b, 0x00, 0x73, 0x74, 0x64, 0x43, 0x6f, 0x6c, 0x6f, 0x72, 0x4d, 0x61, 0x70, 0x00, 0x00, 0x00,
	0x0c, 0x00, 0x03, 0x00, 0x0a, 0x00, 0x66, 0x6f, 0x72, 0x65, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64,
	0x0d, 0x00, 0x03, 0x00, 0x0a, 0x00, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64,
	0x0e, 0x00, 0x03, 0x00, 0x10, 0x00, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64,
	0x50, 0x69, 0x78, 0x6d, 0x61, 0x70, 0x00, 0x00, 0x0f, 0x00, 0x0c, 0x00, 0x0c, 0x00, 0x73, 0x70,
	0x6f, 0x74, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x00, 0x00, 0x10, 0x00, 0x03, 0x00,
	0x09, 0x00, 0x6c, 0x69, 0x6e, 0x65, 0x53, 0x70, 0x61, 0x63, 0x65, 0x00, 0x11, 0x00, 0x00, 0x00,
	0x15, 0x00, 0x73, 0x65, 0x70, 0x61, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x6f, 0x66, 0x4e, 0x65, 0x73,
	0x74, 0x65, 0x64, 0x4c, 0x69, 0x73, 0x74, 0x00,
}

var errorFixture = []byte{
	20, 0, 7, 0, 2, 0, 1, 0, 3, 0, 2, 0, 16, 0, 0, 0,
	'i', 'n', 'v', 'a', 'l', 'i', 'd', ' ', 'i', 'm', ' ', 's', 't', 'y', 'l', 'e',
}

func TestProtocol_ConnectEncodesFixture(t *testing.T) {
	want := []byte{0x01, 0x00, 0x02, 0x00, 0x6c, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
	if got := Encode(Connect{Major: 1, Minor: 0}); !bytes.Equal(got, want) {
		t.Fatalf("Connect = % x, want % x", got, want)
	}
}

func TestProtocol_ConnectReplyEncodesFixture(t *testing.T) {
	want := []byte{0x02, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00}
	if got := Encode(ConnectReply{Major: 1, Minor: 0}); !bytes.Equal(got, want) {
		t.Fatalf("ConnectReply = % x, want % x", got, want)
	}
}

func TestProtocol_CreateICEncodesFixture(t *testing.T) {
	want := []byte{0x32, 0x00, 0x01, 0x00, 0x02, 0x00, 0x00, 0x00}
	if got := Encode(CreateIC{IMID: 2}); !bytes.Equal(got, want) {
		t.Fatalf("CreateIC = % x, want % x", got, want)
	}
}

func TestProtocol_SetEventMaskEncodesFixture(t *testing.T) {
	want := []byte{37, 0, 3, 0, 2, 0, 1, 0, 3, 0, 0, 0, 252, 255, 255, 255}
	msg := SetEventMask{IMID: 2, ICID: 1, ForwardMask: 3, SyncMask: 4294967292}
	if got := Encode(msg); !bytes.Equal(got, want) {
		t.Fatalf("SetEventMask = % x, want % x", got, want)
	}
	decoded, err := Decode(want)
	if err != nil || !reflect.DeepEqual(decoded, msg) {
		t.Fatalf("Decode = %#v, %v; want %#v", decoded, err, msg)
	}
}

func TestProtocol_ForwardEventHeaderIs44Bytes(t *testing.T) {
	got := Encode(ForwardEvent{})
	if len(got) != 44 || !bytes.HasPrefix(got, []byte{0x3c, 0x00, 0x0a, 0x00}) {
		t.Fatalf("ForwardEvent = % x", got)
	}
}

func TestProtocol_DecodeOpenReplyFixture(t *testing.T) {
	msg, err := Decode(openReplyFixture)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	open, ok := msg.(OpenReply)
	if !ok {
		t.Fatalf("Decode = %T, want OpenReply", msg)
	}
	if open.IMID != 1 || len(open.IMAttrs) != 1 || len(open.ICAttrs) != 17 {
		t.Fatalf("OpenReply ids = %d, %d, %d", open.IMID, len(open.IMAttrs), len(open.ICAttrs))
	}
	wantIM := Attr{ID: 0, Type: TypeStyle, Name: AttrQueryInputStyle}
	if open.IMAttrs[0] != wantIM {
		t.Fatalf("IM attr = %#v, want %#v", open.IMAttrs[0], wantIM)
	}
	wantSpot := Attr{ID: 15, Type: TypeXPoint, Name: AttrSpotLocation}
	if open.ICAttrs[14] != wantSpot {
		t.Fatalf("spot attr = %#v, want %#v", open.ICAttrs[14], wantSpot)
	}
	wantLast := Attr{ID: 17, Type: TypeSeparator, Name: AttrSeparatorofNestedList}
	if open.ICAttrs[16] != wantLast {
		t.Fatalf("last attr = %#v, want %#v", open.ICAttrs[16], wantLast)
	}
}

func TestProtocol_EncodeOpenReplyMatchesFixture(t *testing.T) {
	msg, err := Decode(openReplyFixture)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := Encode(msg); !bytes.Equal(got, openReplyFixture) {
		t.Fatalf("re-encoded OpenReply differs from fixture\n got % x\nwant % x", got, openReplyFixture)
	}
}

func TestProtocol_DecodeErrorFixture(t *testing.T) {
	msg, err := Decode(errorFixture)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := ErrorMessage{IMID: 2, ICID: 1, Flag: ErrorInputMethodIDValid | ErrorInputContextIDValid, Code: ErrBadStyle, Detail: "invalid im style"}
	if !reflect.DeepEqual(msg, want) {
		t.Fatalf("Decode = %#v, want %#v", msg, want)
	}
}

func TestProtocol_DecodeRejectsTruncatedFrame(t *testing.T) {
	if _, err := Decode(errorFixture[:10]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Decode short = %v, want ErrMalformed", err)
	}
	if _, err := Decode([]byte{0x01}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Decode header = %v, want ErrMalformed", err)
	}
}

func TestProtocol_DecodeRejectsUnknownOpcode(t *testing.T) {
	if _, err := Decode([]byte{0xf0, 0, 0, 0}); !errors.Is(err, ErrUnsupportedMessage) {
		t.Fatalf("Decode unknown = %v, want ErrUnsupportedMessage", err)
	}
}

func TestProtocol_RoundTripEveryMessage(t *testing.T) {
	xev := XEvent{ResponseType: 2, Detail: 38, Sequence: 9, Time: 123, Root: 1, Event: 2, Child: 3, RootX: -4, RootY: 5, EventX: 6, EventY: -7, State: 8, SameScreen: true}
	ids := map[AttributeName]uint16{AttrInputStyle: 1, AttrSpotLocation: 2}
	nested := NewAttributeBuilder(ids).Nested(AttrSpotLocation, func(b *AttributeBuilder) {
		b.Point(AttrSpotLocation, Point{X: 10, Y: -20})
	}).Build()
	cases := []Message{
		Connect{Major: 1, Minor: 0, AuthNames: []string{"MIT-MAGIC"}},
		ConnectReply{Major: 1, Minor: 0},
		Disconnect{},
		DisconnectReply{},
		Open{Locale: "en_US"},
		OpenReply{IMID: 1, IMAttrs: []Attr{{ID: 0, Type: TypeStyle, Name: AttrQueryInputStyle}}, ICAttrs: []Attr{{ID: 1, Type: TypeLong, Name: AttrInputStyle}}},
		Close{IMID: 3},
		CloseReply{IMID: 3},
		QueryExtension{IMID: 1, Extensions: []string{"XIM_EXT_MOVE"}},
		QueryExtensionReply{IMID: 1, Extensions: []Extension{{Major: 128, Minor: 1, Name: "XIM_EXT_MOVE"}}},
		EncodingNegotiation{IMID: 1, Encodings: []string{"COMPOUND_TEXT"}, EncodingInfos: []string{"info"}},
		EncodingNegotiationReply{IMID: 1, Category: 2, Index: -1},
		GetIMValues{IMID: 1, Names: []uint16{0, 3}},
		GetIMValuesReply{IMID: 1, Attrs: []Attribute{{ID: 0, Value: []byte{1, 2, 3}}}},
		CreateIC{IMID: 1, Attrs: nested},
		CreateICReply{IMID: 1, ICID: 4},
		DestroyIC{IMID: 1, ICID: 4},
		DestroyICReply{IMID: 1, ICID: 4},
		SetICValues{IMID: 1, ICID: 4, Attrs: nested},
		SetICValuesReply{IMID: 1, ICID: 4},
		SetEventMask{IMID: 1, ICID: 4, ForwardMask: 3, SyncMask: 4},
		SetICFocus{IMID: 1, ICID: 4},
		UnsetICFocus{IMID: 1, ICID: 4},
		ResetIC{IMID: 1, ICID: 4},
		ResetICReply{IMID: 1, ICID: 4, PreeditString: UTF8ToCompoundText("ab")},
		ForwardEvent{IMID: 1, ICID: 4, Flag: ForwardEventSynchronous, Serial: 9, Event: xev},
		Sync{IMID: 1, ICID: 4},
		Unhandled{Opcode: 34},
		SyncReply{IMID: 1, ICID: 4},
		Commit{IMID: 1, ICID: 4, Data: CommitChars{Text: UTF8ToCompoundText("x")}},
		Commit{IMID: 1, ICID: 4, Data: CommitKeysym{Keysym: 0x61}},
		Commit{IMID: 1, ICID: 4, Data: CommitChars{Text: UTF8ToCompoundText("x"), Sync: true}},
		Commit{IMID: 1, ICID: 4, Data: CommitKeysym{Keysym: 0x61, Sync: true}},
		Commit{IMID: 1, ICID: 4, Data: CommitBoth{Keysym: 0x61, Text: []byte("y")}},
		Commit{IMID: 1, ICID: 4, Data: CommitBoth{Keysym: 0x61, Text: []byte("y"), Sync: true}},
		PreeditStart{IMID: 1, ICID: 4},
		PreeditStartReply{IMID: 1, ICID: 4, ReturnValue: -1},
		PreeditDone{IMID: 1, ICID: 4},
		PreeditDraw{IMID: 1, ICID: 4, Caret: 2, ChgFirst: 0, ChgLength: 1, Status: PreeditNoFeedback, PreeditString: UTF8ToCompoundText("nihao"), Feedbacks: []Feedback{FeedbackUnderline, FeedbackReverse}},
		PreeditCaret{IMID: 1, ICID: 4, Position: 3, Direction: CaretForwardChar, Style: CaretPrimary},
		PreeditCaretReply{IMID: 1, ICID: 4, Position: 3},
		ErrorMessage{IMID: 1, ICID: 4, Flag: ErrorInputMethodIDValid, Code: ErrBadProtocol, Detail: "bad"},
	}
	for _, msg := range cases {
		wire := Encode(msg)
		if len(wire)%4 != 0 {
			t.Fatalf("%T encoded to %d bytes, not 4-aligned", msg, len(wire))
		}
		got, err := Decode(wire)
		if err != nil {
			t.Fatalf("%T Decode: %v", msg, err)
		}
		if !reflect.DeepEqual(got, msg) {
			t.Fatalf("%T round trip = %#v, want %#v", msg, got, msg)
		}
	}
}

func TestProtocol_AttributeBuilderSkipsUnknownNames(t *testing.T) {
	b := NewAttributeBuilder(map[AttributeName]uint16{AttrClientWindow: 5})
	attrs := b.Long(AttrClientWindow, 0x1234).Long(AttrFocusWindow, 9).Point(AttrSpotLocation, Point{}).Build()
	want := []Attribute{{ID: 5, Value: []byte{0x34, 0x12, 0, 0}}}
	if !reflect.DeepEqual(attrs, want) {
		t.Fatalf("Build = %#v, want %#v", attrs, want)
	}
}

func TestProtocol_DecodeConnectWithZeroHeaderLength(t *testing.T) {
	want := Connect{}
	got, err := Decode([]byte{0x01, 0x00, 0x00, 0x00, 0x6c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode = %#v, %v; want %#v", got, err, want)
	}
}

func TestProtocol_DecodeOpenAndQueryFixtures(t *testing.T) {
	open := []byte{30, 0, 2, 0, 5, 101, 110, 95, 85, 83, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if got, err := Decode(open); err != nil || !reflect.DeepEqual(got, Open{Locale: "en_US"}) {
		t.Fatalf("Decode open = %#v, %v", got, err)
	}
	query := []byte{40, 0, 5, 0, 0, 0, 13, 0, 12, 88, 73, 77, 95, 69, 88, 84, 95, 77, 79, 86, 69, 0, 0, 0}
	want := QueryExtension{IMID: 0, Extensions: []string{"XIM_EXT_MOVE"}}
	if got, err := Decode(query); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode query = %#v, %v", got, err)
	}
}

func TestProtocol_UpstreamReadOnlyOpcodesDecodeUnhandled(t *testing.T) {
	got, err := Decode([]byte{34, 0, 1, 0, 0, 0, 0, 0})
	if err != nil || !reflect.DeepEqual(got, Unhandled{Opcode: 34}) {
		t.Fatalf("Decode = %#v, %v", got, err)
	}
}
