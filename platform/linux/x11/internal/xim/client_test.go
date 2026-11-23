//go:build linux

package xim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

const (
	testRoot         uint32 = 1
	testClientWindow uint32 = 2
	testIMWindow     uint32 = 99
	testServerOwner  uint32 = 77
)

type propKey struct {
	window   uint32
	property uint32
}

type fakeX struct {
	atoms       map[string]uint32
	names       map[uint32]string
	props       map[propKey]Property
	owners      map[uint32]uint32
	conversions []SelectionNotify
	sent        []ClientMessage
}

func newFakeX() *fakeX {
	return &fakeX{
		atoms:  map[string]uint32{},
		names:  map[uint32]string{},
		props:  map[propKey]Property{},
		owners: map[uint32]uint32{},
	}
}

func (f *fakeX) InternAtom(name string) (uint32, error) {
	if atom, ok := f.atoms[name]; ok {
		return atom, nil
	}
	atom := uint32(100 + len(f.atoms))
	f.atoms[name] = atom
	f.names[atom] = name
	return atom, nil
}

func (f *fakeX) AtomName(atom uint32) (string, error) {
	name, ok := f.names[atom]
	if !ok {
		return "", errors.New("unknown atom")
	}
	return name, nil
}

func (f *fakeX) GetProperty(window, property, typ uint32, remove bool) (Property, error) {
	key := propKey{window, property}
	prop := f.props[key]
	if remove {
		delete(f.props, key)
	}
	return prop, nil
}

func (f *fakeX) AppendProperty(window, property, typ uint32, format uint8, data []byte) error {
	key := propKey{window, property}
	prop := f.props[key]
	prop.Type, prop.Format = typ, format
	prop.Value = append(prop.Value, data...)
	f.props[key] = prop
	return nil
}

func (f *fakeX) SelectionOwner(selection uint32) (uint32, error) {
	return f.owners[selection], nil
}

func (f *fakeX) ConvertSelection(requestor, selection, target, property uint32) error {
	f.conversions = append(f.conversions, SelectionNotify{Requestor: requestor, Selection: selection, Target: target, Property: property})
	return nil
}

func (f *fakeX) SendClientMessage(window, messageType uint32, format uint8, data [clientMessageLength]byte) error {
	f.sent = append(f.sent, ClientMessage{Window: window, Type: messageType, Format: format, Data: data})
	return nil
}

func (f *fakeX) Flush() error { return nil }

func (f *fakeX) lastSent(t *testing.T) (ClientMessage, Message) {
	t.Helper()
	if len(f.sent) == 0 {
		t.Fatal("no client message sent")
	}
	msg := f.sent[len(f.sent)-1]
	if msg.Format != 8 {
		return msg, nil
	}
	decoded, err := Decode(msg.Data[:])
	if err != nil {
		t.Fatalf("decode sent message: %v", err)
	}
	return msg, decoded
}

func (f *fakeX) serverSends(t *testing.T, c *Client, h Handler, msg Message) (bool, error) {
	t.Helper()
	wire := Encode(msg)
	if len(wire) < defaultTransportMax {
		var data [clientMessageLength]byte
		copy(data[:], wire)
		return c.FilterEvent(ClientMessage{Window: testClientWindow, Type: f.atoms["_XIM_PROTOCOL"], Format: 8, Data: data}, h)
	}
	atom, _ := f.InternAtom("_XIM_SERVER_DATA")
	f.props[propKey{testClientWindow, atom}] = Property{Type: atomSTRING, Format: 8, Value: wire}
	var data [clientMessageLength]byte
	binary.LittleEndian.PutUint32(data[0:4], uint32(len(wire)))
	binary.LittleEndian.PutUint32(data[4:8], atom)
	return c.FilterEvent(ClientMessage{Window: testClientWindow, Type: f.atoms["_XIM_PROTOCOL"], Format: 32, Data: data}, h)
}

type recordingHandler struct {
	NopHandler
	opened    []uint16
	created   [][2]uint16
	forwarded []XEvent
	commits   []string
	drafts    []string
	features  []Feedback
	errors    []ErrorCode
}

func (h *recordingHandler) HandleOpen(_ *Client, imID uint16) error {
	h.opened = append(h.opened, imID)
	return nil
}

func (h *recordingHandler) HandleCreateIC(_ *Client, imID, icID uint16) error {
	h.created = append(h.created, [2]uint16{imID, icID})
	return nil
}

func (h *recordingHandler) HandleForwardEvent(_ *Client, _, _ uint16, _ ForwardEventFlag, event XEvent) error {
	h.forwarded = append(h.forwarded, event)
	return nil
}

func (h *recordingHandler) HandleCommit(_ *Client, _, _ uint16, text string) error {
	h.commits = append(h.commits, text)
	return nil
}

func (h *recordingHandler) HandlePreeditDraw(_ *Client, _, _ uint16, _, _, _ int32, _ PreeditDrawStatus, text string, feedbacks []Feedback) error {
	h.drafts = append(h.drafts, text)
	h.features = append(h.features, feedbacks...)
	return nil
}

func newTestClient(t *testing.T, fx *fakeX, name string) *Client {
	t.Helper()
	root, _ := fx.InternAtom("XIM_SERVERS")
	server, _ := fx.InternAtom("@server=" + name)
	fx.props[propKey{testRoot, root}] = Property{Type: atomATOM, Format: 32, Value: binary.LittleEndian.AppendUint32(nil, server)}
	fx.owners[server] = testServerOwner
	c, err := Init(fx, testRoot, testClientWindow, name)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return c
}

func TestClient_InitRejectsMissingServer(t *testing.T) {
	fx := newFakeX()
	root, _ := fx.InternAtom("XIM_SERVERS")
	fx.props[propKey{testRoot, root}] = Property{Type: atomATOM, Format: 32}
	if _, err := Init(fx, testRoot, testClientWindow, "absent"); !errors.Is(err, ErrNoXimServer) {
		t.Fatalf("Init = %v, want ErrNoXimServer", err)
	}
}

func TestClient_InitWithoutNameNeedsXMODIFIERS(t *testing.T) {
	t.Setenv("XMODIFIERS", "")
	if _, err := Init(newFakeX(), testRoot, testClientWindow, ""); !errors.Is(err, ErrNoXimServer) {
		t.Fatalf("Init = %v, want ErrNoXimServer", err)
	}
}

func TestClient_HandshakeAndPropertyTransport(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	h := &recordingHandler{}
	transport, _ := fx.InternAtom("TRANSPORT")
	locales, _ := fx.InternAtom("LOCALES")
	xconnect, _ := fx.InternAtom("_XIM_XCONNECT")

	fx.props[propKey{testClientWindow, transport}] = Property{Value: []byte("@transport=X/")}
	if ok, err := c.FilterEvent(SelectionNotify{Requestor: testClientWindow, Property: transport}, h); !ok || err != nil {
		t.Fatalf("TRANSPORT notify = %v, %v", ok, err)
	}
	fx.props[propKey{testClientWindow, locales}] = Property{Value: []byte("en_US")}
	if ok, err := c.FilterEvent(SelectionNotify{Requestor: testClientWindow, Property: locales}, h); !ok || err != nil {
		t.Fatalf("LOCALES notify = %v, %v", ok, err)
	}
	xc, _ := fx.lastSent(t)
	if xc.Window != testServerOwner || xc.Type != xconnect || xc.Format != 32 {
		t.Fatalf("XCONNECT = %+v", xc)
	}

	var reply [clientMessageLength]byte
	binary.LittleEndian.PutUint32(reply[0:4], testIMWindow)
	binary.LittleEndian.PutUint32(reply[12:16], 20)
	if ok, err := c.FilterEvent(ClientMessage{Window: testClientWindow, Type: xconnect, Format: 32, Data: reply}, h); !ok || err != nil {
		t.Fatalf("XCONNECT reply = %v, %v", ok, err)
	}
	_, sentConnect := fx.lastSent(t)
	if !reflect.DeepEqual(sentConnect, Connect{Major: 1, Minor: 0}) {
		t.Fatalf("Connect sent = %#v", sentConnect)
	}

	open := OpenReply{
		IMID:    1,
		IMAttrs: []Attr{{ID: 0, Type: TypeStyle, Name: AttrQueryInputStyle}},
		ICAttrs: []Attr{{ID: 1, Type: TypeLong, Name: AttrInputStyle}, {ID: 2, Type: TypeWindow, Name: AttrClientWindow}},
	}
	if ok, err := fx.serverSends(t, c, h, open); !ok || err != nil {
		t.Fatalf("OpenReply = %v, %v", ok, err)
	}
	negotiation, decoded := fx.lastSent(t)
	if negotiation.Format != 32 {
		t.Fatalf("EncodingNegotiation sent inline, want property transport")
	}
	if decoded != nil {
		t.Fatalf("property transport decoded inline data %#v", decoded)
	}
	atom := binary.LittleEndian.Uint32(negotiation.Data[4:8])
	payload := fx.props[propKey{testIMWindow, atom}].Value
	want := EncodingNegotiation{IMID: 1, Encodings: []string{"COMPOUND_TEXT"}}
	if got, err := Decode(payload); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("EncodingNegotiation payload = %#v, %v", got, err)
	}

	if ok, err := fx.serverSends(t, c, h, EncodingNegotiationReply{IMID: 1}); !ok || err != nil {
		t.Fatalf("EncodingNegotiationReply = %v, %v", ok, err)
	}
	if !reflect.DeepEqual(h.opened, []uint16{1}) {
		t.Fatalf("opened = %v", h.opened)
	}

	if err := c.CreateIC(1, c.BuildICAttributes().Long(AttrInputStyle, uint32(InputStylePreeditCallbacks)).Long(AttrClientWindow, testClientWindow).Build()); err != nil {
		t.Fatalf("CreateIC: %v", err)
	}
	if ok, err := fx.serverSends(t, c, h, CreateICReply{IMID: 1, ICID: 7}); !ok || err != nil {
		t.Fatalf("CreateICReply = %v, %v", ok, err)
	}
	if !reflect.DeepEqual(h.created, [][2]uint16{{1, 7}}) {
		t.Fatalf("created = %v", h.created)
	}

	key := XEvent{ResponseType: 2, Detail: 38, Sequence: 5, Time: 10, State: 1}
	if ok, err := fx.serverSends(t, c, h, ForwardEvent{IMID: 1, ICID: 7, Flag: ForwardEventSynchronous, Serial: 5, Event: key}); !ok || err != nil {
		t.Fatalf("ForwardEvent = %v, %v", ok, err)
	}
	if len(h.forwarded) != 1 || h.forwarded[0] != key {
		t.Fatalf("forwarded = %#v", h.forwarded)
	}
	if _, sync := fx.lastSent(t); !reflect.DeepEqual(sync, SyncReply{IMID: 1, ICID: 7}) {
		t.Fatalf("synchronous forward answered with %#v", sync)
	}

	if ok, err := fx.serverSends(t, c, h, Commit{IMID: 1, ICID: 7, Data: CommitChars{Text: UTF8ToCompoundText("가")}}); !ok || err != nil {
		t.Fatalf("Commit = %v, %v", ok, err)
	}
	if !reflect.DeepEqual(h.commits, []string{"가"}) {
		t.Fatalf("commits = %q", h.commits)
	}
	if _, sync := fx.lastSent(t); !reflect.DeepEqual(sync, SyncReply{IMID: 1, ICID: 7}) {
		t.Fatalf("commit without sync flag answered with %#v", sync)
	}
	if ok, err := fx.serverSends(t, c, h, Commit{IMID: 1, ICID: 7, Data: CommitBoth{Text: []byte("b"), Sync: true}}); !ok || err != nil {
		t.Fatalf("Commit both = %v, %v", ok, err)
	}
	if _, sync := fx.lastSent(t); !reflect.DeepEqual(sync, SyncReply{IMID: 1, ICID: 7}) {
		t.Fatalf("synchronous commit-both answered with %#v", sync)
	}
	if !reflect.DeepEqual(h.commits, []string{"가", "b"}) {
		t.Fatalf("commit-both text = %q", h.commits)
	}
	before := len(fx.sent)
	if ok, err := fx.serverSends(t, c, h, Commit{IMID: 1, ICID: 7, Data: CommitKeysym{Keysym: 0x61, Sync: true}}); !ok || err != nil {
		t.Fatalf("Commit keysym = %v, %v", ok, err)
	}
	if _, sync := fx.lastSent(t); len(fx.sent) != before+1 || !reflect.DeepEqual(sync, SyncReply{IMID: 1, ICID: 7}) {
		t.Fatalf("synchronous keysym commit answered with %#v", sync)
	}
	before = len(fx.sent)
	if ok, err := fx.serverSends(t, c, h, Commit{IMID: 1, ICID: 7, Data: CommitChars{Text: UTF8ToCompoundText("c"), Sync: true}}); !ok || err != nil {
		t.Fatalf("Commit sync chars = %v, %v", ok, err)
	}
	if _, sync := fx.lastSent(t); len(fx.sent) != before+1 || !reflect.DeepEqual(sync, SyncReply{IMID: 1, ICID: 7}) {
		t.Fatalf("synchronous chars commit answered with %#v", sync)
	}

	draw := PreeditDraw{IMID: 1, ICID: 7, Caret: 2, ChgLength: 0, Status: 0, PreeditString: UTF8ToCompoundText("ni"), Feedbacks: []Feedback{FeedbackUnderline}}
	if ok, err := fx.serverSends(t, c, h, draw); !ok || err != nil {
		t.Fatalf("PreeditDraw = %v, %v", ok, err)
	}
	if !reflect.DeepEqual(h.drafts, []string{"ni"}) || !reflect.DeepEqual(h.features, []Feedback{FeedbackUnderline}) {
		t.Fatalf("drafts = %q, features = %v", h.drafts, h.features)
	}
}

func TestClient_InitIgnoresXMODIFIERSWithoutPrefix(t *testing.T) {
	t.Setenv("XMODIFIERS", "fcitx")
	if _, err := Init(newFakeX(), testRoot, testClientWindow, ""); !errors.Is(err, ErrNoXimServer) {
		t.Fatalf("Init = %v, want ErrNoXimServer", err)
	}
}

func TestClient_UpstreamReadOnlyFramesAreIgnored(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	h := &recordingHandler{}
	if ok, err := fx.serverSends(t, c, h, Unhandled{Opcode: 34}); !ok || err != nil {
		t.Fatalf("Unhandled = %v, %v", ok, err)
	}
}

func TestClient_ServerErrorIsReturned(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	h := &recordingHandler{}
	_, err := fx.serverSends(t, c, h, ErrorMessage{IMID: 1, ICID: 2, Flag: ErrorInputMethodIDValid, Code: ErrBadStyle, Detail: "no"})
	var serverErr *ServerError
	if !errors.As(err, &serverErr) || serverErr.Code != ErrBadStyle {
		t.Fatalf("error = %v, want ServerError with ErrBadStyle", err)
	}
}

func TestClient_WindowErrorForIMWindowReportsNoServer(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	c.imWindow = testIMWindow
	if _, err := c.FilterEvent(WindowError{BadValue: testIMWindow}, &recordingHandler{}); !errors.Is(err, ErrNoXimServer) {
		t.Fatalf("WindowError = %v, want ErrNoXimServer", err)
	}
}

func TestClient_UnsupportedTransportIsRejected(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	transport, _ := fx.InternAtom("TRANSPORT")
	fx.props[propKey{testClientWindow, transport}] = Property{Value: []byte("@transport=local/")}
	if _, err := c.FilterEvent(SelectionNotify{Requestor: testClientWindow, Property: transport}, &recordingHandler{}); !errors.Is(err, ErrUnsupportedTransport) {
		t.Fatalf("TRANSPORT = %v, want ErrUnsupportedTransport", err)
	}
}

func TestClient_ForeignEventsAreIgnored(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	if ok, err := c.FilterEvent(SelectionNotify{Requestor: 5}, &recordingHandler{}); ok || err != nil {
		t.Fatalf("foreign SelectionNotify = %v, %v", ok, err)
	}
	if ok, err := c.FilterEvent(ClientMessage{Window: 5}, &recordingHandler{}); ok || err != nil {
		t.Fatalf("foreign ClientMessage = %v, %v", ok, err)
	}
}

func TestClient_OversizedInlineFrameNeedsMultiCM(t *testing.T) {
	fx := newFakeX()
	c := newTestClient(t, fx, "test")
	c.transportMax = 64
	c.imWindow = testIMWindow
	if err := c.send(ErrorMessage{Detail: string(bytes.Repeat([]byte{'x'}, 8))}); !errors.Is(err, ErrMultiCM) {
		t.Fatalf("send = %v, want ErrMultiCM", err)
	}
}
