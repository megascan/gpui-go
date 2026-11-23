//go:build linux

package xim

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const nativeEndian uint8 = 0x6c

const (
	opConnect                  uint8 = 1
	opConnectReply             uint8 = 2
	opDisconnect               uint8 = 3
	opDisconnectReply          uint8 = 4
	opError                    uint8 = 20
	opOpen                     uint8 = 30
	opOpenReply                uint8 = 31
	opClose                    uint8 = 32
	opCloseReply               uint8 = 33
	opSetEventMask             uint8 = 37
	opEncodingNegotiation      uint8 = 38
	opEncodingNegotiationReply uint8 = 39
	opQueryExtension           uint8 = 40
	opQueryExtensionReply      uint8 = 41
	opGetIMValues              uint8 = 44
	opGetIMValuesReply         uint8 = 45
	opCreateIC                 uint8 = 50
	opCreateICReply            uint8 = 51
	opDestroyIC                uint8 = 52
	opDestroyICReply           uint8 = 53
	opSetICValues              uint8 = 54
	opSetICValuesReply         uint8 = 55
	opSetICFocus               uint8 = 58
	opUnsetICFocus             uint8 = 59
	opForwardEvent             uint8 = 60
	opSync                     uint8 = 61
	opSyncReply                uint8 = 62
	opCommit                   uint8 = 63
	opResetIC                  uint8 = 64
	opResetICReply             uint8 = 65
	opPreeditStart             uint8 = 73
	opPreeditStartReply        uint8 = 74
	opPreeditDraw              uint8 = 75
	opPreeditCaret             uint8 = 76
	opPreeditCaretReply        uint8 = 77
	opPreeditDone              uint8 = 78
)

// ErrMalformed reports a frame that is shorter than its header or list lengths claim.
var ErrMalformed = errors.New("xim: malformed message")

// ErrUnsupportedMessage reports a frame whose opcode is outside the client subset.
var ErrUnsupportedMessage = errors.New("xim: unsupported message")

// ErrorCode is the XIM_ERROR code a server reports.
type ErrorCode uint16

const (
	ErrBadAlloc           ErrorCode = 1
	ErrBadStyle           ErrorCode = 2
	ErrBadClientWindow    ErrorCode = 3
	ErrBadFocusWindow     ErrorCode = 4
	ErrBadArea            ErrorCode = 5
	ErrBadSpotLocation    ErrorCode = 6
	ErrBadColormap        ErrorCode = 7
	ErrBadAtom            ErrorCode = 8
	ErrBadPixel           ErrorCode = 9
	ErrBadPixmap          ErrorCode = 10
	ErrBadName            ErrorCode = 11
	ErrBadCursor          ErrorCode = 12
	ErrBadProtocol        ErrorCode = 13
	ErrBadForeground      ErrorCode = 14
	ErrBadBackground      ErrorCode = 15
	ErrLocaleNotSupported ErrorCode = 16
	ErrBadSomething       ErrorCode = 999
)

// ErrorFlag says which of the IM and IC ids in an Error message are valid.
type ErrorFlag uint16

const (
	ErrorInputMethodIDValid  ErrorFlag = 1
	ErrorInputContextIDValid ErrorFlag = 2
)

// ForwardEventFlag carries the synchronous and filtering bits of a ForwardEvent.
type ForwardEventFlag uint16

const (
	ForwardEventSynchronous         ForwardEventFlag = 1
	ForwardEventRequestFiltering    ForwardEventFlag = 2
	ForwardEventRequestLoopUpstring ForwardEventFlag = 4
)

// PreeditDrawStatus carries the no-string and no-feedback bits of a PreeditDraw.
type PreeditDrawStatus uint32

const (
	PreeditNoString   PreeditDrawStatus = 1
	PreeditNoFeedback PreeditDrawStatus = 2
)

// Feedback is the per-character rendering hint of a PreeditDraw.
type Feedback uint32

const (
	FeedbackReverse           Feedback = 1
	FeedbackUnderline         Feedback = 2
	FeedbackHighlight         Feedback = 4
	FeedbackPrimary           Feedback = 8
	FeedbackSecondary         Feedback = 16
	FeedbackTertiary          Feedback = 32
	FeedbackVisibleToForward  Feedback = 64
	FeedbackVisibleToBackward Feedback = 128
	FeedbackVisibleCenter     Feedback = 256
)

// InputStyle is the bit set an input context requests.
type InputStyle uint32

const (
	InputStylePreeditArea      InputStyle = 1
	InputStylePreeditCallbacks InputStyle = 2
	InputStylePreeditPosition  InputStyle = 4
	InputStylePreeditNothing   InputStyle = 8
	InputStylePreeditNone      InputStyle = 16
	InputStyleStatusArea       InputStyle = 256
	InputStyleStatusCallbacks  InputStyle = 512
	InputStyleStatusNothing    InputStyle = 1024
	InputStyleStatusNone       InputStyle = 2048
)

// CaretDirection is the direction of a PreeditCaret move.
type CaretDirection uint32

const (
	CaretForwardChar CaretDirection = iota
	CaretBackwardChar
	CaretForwardWord
	CaretBackwardWord
	CaretUp
	CaretDown
	CaretNextLine
	CaretPreviousLine
	CaretLineStart
	CaretLineEnd
	CaretAbsolutePosition
	CaretDontChange
)

// CaretStyle is the caret shape a PreeditCaret asks for.
type CaretStyle uint32

const (
	CaretInvisible CaretStyle = 0
	CaretPrimary   CaretStyle = 1
	CaretSecondary CaretStyle = 2
)

// Message is one XIM protocol frame; the concrete types below are the client subset.
type Message interface {
	opcode() uint8
	encodeBody(w *writer)
}

// CommitData is the payload of a Commit message.
type CommitData interface {
	isCommitData()
}

// CommitChars commits compound text.
type CommitChars struct {
	Text []byte
	Sync bool
}

// CommitKeysym commits a keysym.
type CommitKeysym struct {
	Keysym uint32
	Sync   bool
}

// CommitBoth commits a keysym and compound text.
type CommitBoth struct {
	Keysym uint32
	Text   []byte
	Sync   bool
}

func (CommitChars) isCommitData() {}

func (CommitKeysym) isCommitData() {}

func (CommitBoth) isCommitData() {}

// XEvent is the wire form of a core X KeyPress or KeyRelease carried by ForwardEvent.
type XEvent struct {
	ResponseType uint8
	Detail       uint8
	Sequence     uint16
	Time         uint32
	Root         uint32
	Event        uint32
	Child        uint32
	RootX        int16
	RootY        int16
	EventX       int16
	EventY       int16
	State        uint16
	SameScreen   bool
}

type Connect struct {
	Major, Minor uint16
	AuthNames    []string
}

type ConnectReply struct {
	Major, Minor uint16
}

type Disconnect struct{}

type DisconnectReply struct{}

type Open struct {
	Locale string
}

type OpenReply struct {
	IMID    uint16
	IMAttrs []Attr
	ICAttrs []Attr
}

type Close struct {
	IMID uint16
}

type CloseReply struct {
	IMID uint16
}

type QueryExtension struct {
	IMID       uint16
	Extensions []string
}

type QueryExtensionReply struct {
	IMID       uint16
	Extensions []Extension
}

type EncodingNegotiation struct {
	IMID          uint16
	Encodings     []string
	EncodingInfos []string
}

type EncodingNegotiationReply struct {
	IMID     uint16
	Category uint16
	Index    int16
}

type GetIMValues struct {
	IMID  uint16
	Names []uint16
}

type GetIMValuesReply struct {
	IMID  uint16
	Attrs []Attribute
}

type CreateIC struct {
	IMID  uint16
	Attrs []Attribute
}

type CreateICReply struct {
	IMID, ICID uint16
}

type DestroyIC struct {
	IMID, ICID uint16
}

type DestroyICReply struct {
	IMID, ICID uint16
}

type SetICValues struct {
	IMID, ICID uint16
	Attrs      []Attribute
}

type SetICValuesReply struct {
	IMID, ICID uint16
}

type SetEventMask struct {
	IMID, ICID  uint16
	ForwardMask uint32
	SyncMask    uint32
}

type SetICFocus struct {
	IMID, ICID uint16
}

type UnsetICFocus struct {
	IMID, ICID uint16
}

type ResetIC struct {
	IMID, ICID uint16
}

type ResetICReply struct {
	IMID, ICID    uint16
	PreeditString []byte
}

type ForwardEvent struct {
	IMID, ICID uint16
	Flag       ForwardEventFlag
	Serial     uint16
	Event      XEvent
}

type Sync struct {
	IMID, ICID uint16
}

type SyncReply struct {
	IMID, ICID uint16
}

type Commit struct {
	IMID, ICID uint16
	Data       CommitData
}

type PreeditStart struct {
	IMID, ICID uint16
}

type PreeditStartReply struct {
	IMID, ICID  uint16
	ReturnValue int32
}

type PreeditDone struct {
	IMID, ICID uint16
}

type PreeditDraw struct {
	IMID, ICID    uint16
	Caret         int32
	ChgFirst      int32
	ChgLength     int32
	Status        PreeditDrawStatus
	PreeditString []byte
	Feedbacks     []Feedback
}

type PreeditCaret struct {
	IMID, ICID uint16
	Position   int32
	Direction  CaretDirection
	Style      CaretStyle
}

type PreeditCaretReply struct {
	IMID, ICID uint16
	Position   int32
}

type ErrorMessage struct {
	IMID, ICID uint16
	Flag       ErrorFlag
	Code       ErrorCode
	Detail     string
}

func (Connect) opcode() uint8                  { return opConnect }
func (ConnectReply) opcode() uint8             { return opConnectReply }
func (Disconnect) opcode() uint8               { return opDisconnect }
func (DisconnectReply) opcode() uint8          { return opDisconnectReply }
func (Open) opcode() uint8                     { return opOpen }
func (OpenReply) opcode() uint8                { return opOpenReply }
func (Close) opcode() uint8                    { return opClose }
func (CloseReply) opcode() uint8               { return opCloseReply }
func (QueryExtension) opcode() uint8           { return opQueryExtension }
func (QueryExtensionReply) opcode() uint8      { return opQueryExtensionReply }
func (EncodingNegotiation) opcode() uint8      { return opEncodingNegotiation }
func (EncodingNegotiationReply) opcode() uint8 { return opEncodingNegotiationReply }
func (GetIMValues) opcode() uint8              { return opGetIMValues }
func (GetIMValuesReply) opcode() uint8         { return opGetIMValuesReply }
func (CreateIC) opcode() uint8                 { return opCreateIC }
func (CreateICReply) opcode() uint8            { return opCreateICReply }
func (DestroyIC) opcode() uint8                { return opDestroyIC }
func (DestroyICReply) opcode() uint8           { return opDestroyICReply }
func (SetICValues) opcode() uint8              { return opSetICValues }
func (SetICValuesReply) opcode() uint8         { return opSetICValuesReply }
func (SetEventMask) opcode() uint8             { return opSetEventMask }
func (SetICFocus) opcode() uint8               { return opSetICFocus }
func (UnsetICFocus) opcode() uint8             { return opUnsetICFocus }
func (ResetIC) opcode() uint8                  { return opResetIC }
func (ResetICReply) opcode() uint8             { return opResetICReply }
func (ForwardEvent) opcode() uint8             { return opForwardEvent }
func (Sync) opcode() uint8                     { return opSync }
func (SyncReply) opcode() uint8                { return opSyncReply }
func (Commit) opcode() uint8                   { return opCommit }
func (PreeditStart) opcode() uint8             { return opPreeditStart }
func (PreeditStartReply) opcode() uint8        { return opPreeditStartReply }
func (PreeditDone) opcode() uint8              { return opPreeditDone }
func (PreeditDraw) opcode() uint8              { return opPreeditDraw }
func (PreeditCaret) opcode() uint8             { return opPreeditCaret }
func (PreeditCaretReply) opcode() uint8        { return opPreeditCaretReply }
func (ErrorMessage) opcode() uint8             { return opError }

type writer struct {
	b []byte
}

func (w *writer) u8(v uint8)     { w.b = append(w.b, v) }
func (w *writer) u16(v uint16)   { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *writer) u32(v uint32)   { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *writer) i16(v int16)    { w.u16(uint16(v)) }
func (w *writer) i32(v int32)    { w.u32(uint32(v)) }
func (w *writer) bytes(v []byte) { w.b = append(w.b, v...) }

func (w *writer) pad4() {
	for len(w.b)%4 != 0 {
		w.b = append(w.b, 0)
	}
}

func (w *writer) str8(s string) {
	w.u8(uint8(len(s)))
	w.b = append(w.b, s...)
	w.pad4()
}

func (w *writer) str16(s string) {
	w.u16(uint16(len(s)))
	w.b = append(w.b, s...)
	w.pad4()
}

func (w *writer) attributes(attrs []Attribute) {
	for _, attr := range attrs {
		w.u16(attr.ID)
		w.u16(uint16(len(attr.Value)))
		w.bytes(attr.Value)
		w.pad4()
	}
}

func (w *writer) attrList(attrs []Attr) {
	for _, attr := range attrs {
		w.u16(attr.ID)
		w.u16(uint16(attr.Type))
		w.u16(uint16(len(attr.Name.String())))
		w.b = append(w.b, attr.Name.String()...)
		w.pad4()
	}
}

func attrListSize(attrs []Attr) int {
	size := 0
	for _, attr := range attrs {
		size += withPad4(len(attr.Name.String()) + 6)
	}
	return size
}

func attributesSize(attrs []Attribute) int {
	size := 0
	for _, attr := range attrs {
		size += withPad4(len(attr.Value) + 4)
	}
	return size
}

func withPad4(n int) int {
	return (n + 3) &^ 3
}

// Encode returns the wire form of m with its header length filled in.
func Encode(m Message) []byte {
	w := &writer{b: make([]byte, 4, 64)}
	w.b[0] = m.opcode()
	m.encodeBody(w)
	w.pad4()
	binary.LittleEndian.PutUint16(w.b[2:4], uint16((len(w.b)-4)/4))
	return w.b
}

func (m Connect) encodeBody(w *writer) {
	w.u8(nativeEndian)
	w.u8(0)
	w.u16(m.Major)
	w.u16(m.Minor)
	size := 0
	for _, name := range m.AuthNames {
		size += withPad4(len(name) + 2)
	}
	w.u16(uint16(size))
	for _, name := range m.AuthNames {
		w.u16(uint16(len(name)))
		w.b = append(w.b, name...)
		w.pad4()
	}
}

func (m ConnectReply) encodeBody(w *writer) {
	w.u16(m.Major)
	w.u16(m.Minor)
}

func (Disconnect) encodeBody(*writer) {}

func (DisconnectReply) encodeBody(*writer) {}

func (m Open) encodeBody(w *writer) {
	w.str8(m.Locale)
}

func (m OpenReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(uint16(attrListSize(m.IMAttrs)))
	w.attrList(m.IMAttrs)
	w.u16(uint16(attrListSize(m.ICAttrs)))
	w.u16(0)
	w.attrList(m.ICAttrs)
}

func (m Close) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(0)
}

func (m CloseReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(0)
}

func (m QueryExtension) encodeBody(w *writer) {
	w.u16(m.IMID)
	size := 0
	for _, name := range m.Extensions {
		size += len(name) + 1
	}
	w.u16(uint16(size))
	for _, name := range m.Extensions {
		w.u8(uint8(len(name)))
		w.b = append(w.b, name...)
	}
	w.pad4()
}

func (m QueryExtensionReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	size := 0
	for _, ext := range m.Extensions {
		size += withPad4(len(ext.Name) + 4)
	}
	w.u16(uint16(size))
	for _, ext := range m.Extensions {
		w.u8(ext.Major)
		w.u8(ext.Minor)
		w.u16(uint16(len(ext.Name)))
		w.b = append(w.b, ext.Name...)
		w.pad4()
	}
}

func (m EncodingNegotiation) encodeBody(w *writer) {
	w.u16(m.IMID)
	size := 0
	for _, enc := range m.Encodings {
		size += len(enc) + 1
	}
	w.u16(uint16(size))
	for _, enc := range m.Encodings {
		w.u8(uint8(len(enc)))
		w.b = append(w.b, enc...)
	}
	w.pad4()
	infos := 0
	for _, info := range m.EncodingInfos {
		infos += withPad4(len(info) + 2)
	}
	w.u16(uint16(infos))
	w.u16(0)
	for _, info := range m.EncodingInfos {
		w.str16(info)
	}
}

func (m EncodingNegotiationReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.Category)
	w.i16(m.Index)
	w.u16(0)
}

func (m GetIMValues) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(uint16(2 * len(m.Names)))
	for _, name := range m.Names {
		w.u16(name)
	}
	w.pad4()
}

func (m GetIMValuesReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(uint16(attributesSize(m.Attrs)))
	w.attributes(m.Attrs)
}

func (m CreateIC) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(uint16(attributesSize(m.Attrs)))
	w.attributes(m.Attrs)
}

func (m CreateICReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m DestroyIC) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m DestroyICReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m SetICValues) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.u16(uint16(attributesSize(m.Attrs)))
	w.u16(0)
	w.attributes(m.Attrs)
}

func (m SetICValuesReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m SetEventMask) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.u32(m.ForwardMask)
	w.u32(m.SyncMask)
}

func (m SetICFocus) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m UnsetICFocus) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m ResetIC) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m ResetICReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.u16(uint16(len(m.PreeditString)))
	w.bytes(m.PreeditString)
	w.pad4()
}

func (m ForwardEvent) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.u16(uint16(m.Flag))
	w.u16(m.Serial)
	m.Event.encode(w)
}

func (x XEvent) encode(w *writer) {
	w.u8(x.ResponseType)
	w.u8(x.Detail)
	w.u16(x.Sequence)
	w.u32(x.Time)
	w.u32(x.Root)
	w.u32(x.Event)
	w.u32(x.Child)
	w.i16(x.RootX)
	w.i16(x.RootY)
	w.i16(x.EventX)
	w.i16(x.EventY)
	w.u16(x.State)
	w.u8(boolByte(x.SameScreen))
	w.u8(0)
}

func boolByte(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

func (m Sync) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m SyncReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m Commit) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	switch d := m.Data.(type) {
	case CommitChars:
		w.u16(flagWithSync(2, d.Sync))
		w.u16(uint16(len(d.Text)))
		w.bytes(d.Text)
		w.pad4()
	case CommitKeysym:
		w.u16(flagWithSync(4, d.Sync))
		w.u16(0)
		w.u32(d.Keysym)
	case CommitBoth:
		w.u16(flagWithSync(6, d.Sync))
		w.u16(0)
		w.u32(d.Keysym)
		w.u16(uint16(len(d.Text)))
		w.bytes(d.Text)
		w.pad4()
	}
}

func flagWithSync(base uint16, sync bool) uint16 {
	if sync {
		return base + 1
	}
	return base
}

func (m PreeditStart) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m PreeditStartReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.i32(m.ReturnValue)
}

func (m PreeditDone) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
}

func (m PreeditDraw) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.i32(m.Caret)
	w.i32(m.ChgFirst)
	w.i32(m.ChgLength)
	w.u32(uint32(m.Status))
	w.u16(uint16(len(m.PreeditString)))
	w.bytes(m.PreeditString)
	w.pad4()
	w.u16(uint16(4 * len(m.Feedbacks)))
	w.u16(0)
	for _, fb := range m.Feedbacks {
		w.u32(uint32(fb))
	}
}

func (m PreeditCaret) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.i32(m.Position)
	w.u32(uint32(m.Direction))
	w.u32(uint32(m.Style))
}

func (m PreeditCaretReply) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.i32(m.Position)
}

func (m ErrorMessage) encodeBody(w *writer) {
	w.u16(m.IMID)
	w.u16(m.ICID)
	w.u16(uint16(m.Flag))
	w.u16(uint16(m.Code))
	w.u16(uint16(len(m.Detail)))
	w.u16(0)
	w.b = append(w.b, m.Detail...)
	w.pad4()
}

type errBox struct {
	err error
}

type reader struct {
	b   []byte
	pos int
	end int
	e   *errBox
}

func (r *reader) take(n int) []byte {
	if r.e.err != nil {
		return nil
	}
	if n < 0 || r.pos+n > r.end {
		r.e.err = ErrMalformed
		return nil
	}
	s := r.b[r.pos : r.pos+n]
	r.pos += n
	return s
}

func (r *reader) u8() uint8 {
	s := r.take(1)
	if s == nil {
		return 0
	}
	return s[0]
}

func (r *reader) u16() uint16 {
	s := r.take(2)
	if s == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(s)
}

func (r *reader) u32() uint32 {
	s := r.take(4)
	if s == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(s)
}

func (r *reader) i16() int16 { return int16(r.u16()) }

func (r *reader) i32() int32 { return int32(r.u32()) }

func (r *reader) remaining() int { return r.end - r.pos }

func (r *reader) pad4() {
	r.take((4 - r.pos%4) % 4)
}

func (r *reader) sub(n int) *reader {
	if n < 0 || r.pos+n > r.end {
		r.e.err = ErrMalformed
		return &reader{b: r.b, pos: r.end, end: r.end, e: r.e}
	}
	s := &reader{b: r.b, pos: r.pos, end: r.pos + n, e: r.e}
	r.pos += n
	return s
}

func (r *reader) list16() *reader {
	return r.sub(int(r.u16()))
}

func (r *reader) listWithPad() *reader {
	n := int(r.u16())
	r.u16()
	return r.sub(n)
}

func (r *reader) str8() string {
	n := r.u8()
	s := r.take(int(n))
	r.pad4()
	return string(s)
}

func (r *reader) str16() string {
	n := r.u16()
	s := r.take(int(n))
	r.pad4()
	return string(s)
}

func (r *reader) attributes(sub *reader) []Attribute {
	var out []Attribute
	for sub.remaining() > 0 && sub.e.err == nil {
		attr := Attribute{ID: sub.u16()}
		n := sub.u16()
		attr.Value = append([]byte(nil), sub.take(int(n))...)
		sub.pad4()
		out = append(out, attr)
	}
	return out
}

func (r *reader) attrList(sub *reader) ([]Attr, error) {
	var out []Attr
	for sub.remaining() > 0 && sub.e.err == nil {
		attr := Attr{ID: sub.u16()}
		ty, ok := attrTypeFromWire(sub.u16())
		if !ok {
			return nil, fmt.Errorf("xim: unknown attribute type: %w", ErrMalformed)
		}
		attr.Type = ty
		name, ok := attributeNameFromString(string(sub.take(int(sub.u16()))))
		if !ok {
			return nil, fmt.Errorf("xim: unknown attribute name: %w", ErrMalformed)
		}
		attr.Name = name
		sub.pad4()
		out = append(out, attr)
	}
	return out, nil
}

func (r *reader) extensions(sub *reader) []Extension {
	var out []Extension
	for sub.remaining() > 0 && sub.e.err == nil {
		ext := Extension{Major: sub.u8(), Minor: sub.u8()}
		ext.Name = string(sub.take(int(sub.u16())))
		sub.pad4()
		out = append(out, ext)
	}
	return out
}

// Decode parses one frame; list lengths bound their own items and trailing bytes are ignored.
func Decode(b []byte) (Message, error) {
	if len(b) < 4 {
		return nil, ErrMalformed
	}
	r := &reader{b: b, pos: 4, end: len(b), e: &errBox{}}
	msg, err := decodeBody(b[0], r)
	if err != nil {
		return nil, err
	}
	if r.e.err != nil {
		return nil, r.e.err
	}
	return msg, nil
}

func decodeBody(op uint8, r *reader) (Message, error) {
	switch op {
	case opConnect:
		if endian := r.u8(); endian != nativeEndian {
			return nil, fmt.Errorf("xim: big-endian peer: %w", ErrUnsupportedMessage)
		}
		r.u8()
		m := Connect{Major: r.u16(), Minor: r.u16()}
		sub := r.list16()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.AuthNames = append(m.AuthNames, sub.str16())
		}
		return m, nil
	case opConnectReply:
		return ConnectReply{Major: r.u16(), Minor: r.u16()}, nil
	case opDisconnect:
		return Disconnect{}, nil
	case opDisconnectReply:
		return DisconnectReply{}, nil
	case opOpen:
		return Open{Locale: r.str8()}, nil
	case opOpenReply:
		m := OpenReply{IMID: r.u16()}
		var err error
		if m.IMAttrs, err = r.attrList(r.list16()); err != nil {
			return nil, err
		}
		if m.ICAttrs, err = r.attrList(r.listWithPad()); err != nil {
			return nil, err
		}
		return m, nil
	case opClose:
		m := Close{IMID: r.u16()}
		r.u16()
		return m, nil
	case opCloseReply:
		m := CloseReply{IMID: r.u16()}
		r.u16()
		return m, nil
	case opQueryExtension:
		m := QueryExtension{IMID: r.u16()}
		sub := r.list16()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.Extensions = append(m.Extensions, string(sub.take(int(sub.u8()))))
		}
		r.pad4()
		return m, nil
	case opQueryExtensionReply:
		m := QueryExtensionReply{IMID: r.u16()}
		m.Extensions = r.extensions(r.list16())
		return m, nil
	case opEncodingNegotiation:
		m := EncodingNegotiation{IMID: r.u16()}
		sub := r.list16()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.Encodings = append(m.Encodings, string(sub.take(int(sub.u8()))))
		}
		r.pad4()
		sub = r.listWithPad()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.EncodingInfos = append(m.EncodingInfos, sub.str16())
		}
		return m, nil
	case opEncodingNegotiationReply:
		m := EncodingNegotiationReply{IMID: r.u16(), Category: r.u16(), Index: r.i16()}
		r.u16()
		return m, nil
	case opGetIMValues:
		m := GetIMValues{IMID: r.u16()}
		sub := r.list16()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.Names = append(m.Names, sub.u16())
		}
		r.pad4()
		return m, nil
	case opGetIMValuesReply:
		m := GetIMValuesReply{IMID: r.u16()}
		m.Attrs = r.attributes(r.list16())
		return m, nil
	case opCreateIC:
		m := CreateIC{IMID: r.u16()}
		m.Attrs = r.attributes(r.list16())
		return m, nil
	case opCreateICReply:
		return CreateICReply{IMID: r.u16(), ICID: r.u16()}, nil
	case opDestroyIC:
		return DestroyIC{IMID: r.u16(), ICID: r.u16()}, nil
	case opDestroyICReply:
		return DestroyICReply{IMID: r.u16(), ICID: r.u16()}, nil
	case opSetICValues:
		m := SetICValues{IMID: r.u16(), ICID: r.u16()}
		m.Attrs = r.attributes(r.listWithPad())
		return m, nil
	case opSetICValuesReply:
		return SetICValuesReply{IMID: r.u16(), ICID: r.u16()}, nil
	case opSetEventMask:
		return SetEventMask{IMID: r.u16(), ICID: r.u16(), ForwardMask: r.u32(), SyncMask: r.u32()}, nil
	case opSetICFocus:
		return SetICFocus{IMID: r.u16(), ICID: r.u16()}, nil
	case opUnsetICFocus:
		return UnsetICFocus{IMID: r.u16(), ICID: r.u16()}, nil
	case opResetIC:
		return ResetIC{IMID: r.u16(), ICID: r.u16()}, nil
	case opResetICReply:
		m := ResetICReply{IMID: r.u16(), ICID: r.u16()}
		m.PreeditString = append([]byte(nil), r.take(int(r.u16()))...)
		r.pad4()
		return m, nil
	case opForwardEvent:
		m := ForwardEvent{IMID: r.u16(), ICID: r.u16(), Flag: ForwardEventFlag(r.u16()), Serial: r.u16()}
		m.Event = decodeXEvent(r)
		return m, nil
	case opSync:
		return Sync{IMID: r.u16(), ICID: r.u16()}, nil
	case opSyncReply:
		return SyncReply{IMID: r.u16(), ICID: r.u16()}, nil
	case opCommit:
		m := Commit{IMID: r.u16(), ICID: r.u16()}
		flag := r.u16()
		switch flag {
		case 2, 3:
			m.Data = CommitChars{Text: append([]byte(nil), r.take(int(r.u16()))...), Sync: flag == 3}
			r.pad4()
		case 4, 5:
			r.u16()
			m.Data = CommitKeysym{Keysym: r.u32(), Sync: flag == 5}
		case 6, 7:
			r.u16()
			keysym := r.u32()
			text := append([]byte(nil), r.take(int(r.u16()))...)
			r.pad4()
			m.Data = CommitBoth{Keysym: keysym, Text: text, Sync: flag == 7}
		default:
			return nil, fmt.Errorf("xim: unknown commit flag %d: %w", flag, ErrMalformed)
		}
		return m, nil
	case opPreeditStart:
		return PreeditStart{IMID: r.u16(), ICID: r.u16()}, nil
	case opPreeditStartReply:
		return PreeditStartReply{IMID: r.u16(), ICID: r.u16(), ReturnValue: r.i32()}, nil
	case opPreeditDone:
		return PreeditDone{IMID: r.u16(), ICID: r.u16()}, nil
	case opPreeditDraw:
		m := PreeditDraw{IMID: r.u16(), ICID: r.u16(), Caret: r.i32(), ChgFirst: r.i32(), ChgLength: r.i32()}
		m.Status = PreeditDrawStatus(r.u32())
		m.PreeditString = append([]byte(nil), r.take(int(r.u16()))...)
		r.pad4()
		sub := r.listWithPad()
		for sub.remaining() > 0 && sub.e.err == nil {
			m.Feedbacks = append(m.Feedbacks, Feedback(sub.u32()))
		}
		return m, nil
	case opPreeditCaret:
		return PreeditCaret{IMID: r.u16(), ICID: r.u16(), Position: r.i32(), Direction: CaretDirection(r.u32()), Style: CaretStyle(r.u32())}, nil
	case opPreeditCaretReply:
		return PreeditCaretReply{IMID: r.u16(), ICID: r.u16(), Position: r.i32()}, nil
	case opError:
		m := ErrorMessage{IMID: r.u16(), ICID: r.u16(), Flag: ErrorFlag(r.u16()), Code: ErrorCode(r.u16())}
		n := r.u16()
		r.u16()
		m.Detail = string(r.take(int(n)))
		r.pad4()
		return m, nil
	}
	if upstreamReadOpcodes[op] {
		return Unhandled{Opcode: op}, nil
	}
	return nil, fmt.Errorf("xim: opcode %d: %w", op, ErrUnsupportedMessage)
}

var upstreamReadOpcodes = map[uint8]bool{
	10: true, 11: true, 12: true, 13: true, 14: true,
	34: true, 35: true, 36: true,
	42: true, 43: true,
	56: true, 57: true,
	70: true, 71: true, 72: true,
	79: true, 80: true, 81: true, 82: true,
}

// Unhandled is a frame the upstream reader parses but the client ignores.
type Unhandled struct {
	Opcode uint8
}

func (u Unhandled) opcode() uint8 { return u.Opcode }

func (Unhandled) encodeBody(*writer) {}

func decodeXEvent(r *reader) XEvent {
	x := XEvent{
		ResponseType: r.u8(),
		Detail:       r.u8(),
		Sequence:     r.u16(),
		Time:         r.u32(),
		Root:         r.u32(),
		Event:        r.u32(),
		Child:        r.u32(),
		RootX:        r.i16(),
		RootY:        r.i16(),
		EventX:       r.i16(),
		EventY:       r.i16(),
		State:        r.u16(),
		SameScreen:   r.u8() != 0,
	}
	r.u8()
	return x
}
