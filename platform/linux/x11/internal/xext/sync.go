//go:build linux

package xext

import "github.com/jezek/xgb"

// SyncCounter is sync::Counter.
type SyncCounter uint32

// NewSyncCounterID allocates a counter id.
func NewSyncCounterID(c *xgb.Conn) (SyncCounter, error) {
	id, err := c.NewId()
	return SyncCounter(id), err
}

// SyncInt64 is sync::Int64; on the wire hi precedes lo.
type SyncInt64 struct {
	Hi int32
	Lo uint32
}

// SyncInitializeCookie is the cookie of SyncInitialize.
type SyncInitializeCookie struct{ *xgb.Cookie }

// SyncInitializeReply is sync::InitializeReply.
type SyncInitializeReply struct {
	Sequence     uint16
	Length       uint32
	MajorVersion uint8
	MinorVersion uint8
}

// SyncInitialize sends an Initialize request.
func SyncInitialize(c *xgb.Conn, desiredMajorVersion, desiredMinorVersion uint8) SyncInitializeCookie {
	return SyncInitializeCookie{send(c, SyncExtensionName, "Initialize", 0, []byte{desiredMajorVersion, desiredMinorVersion}, true, true)}
}

// Reply waits for and decodes the reply.
func (c SyncInitializeCookie) Reply() (*SyncInitializeReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	return SyncInitializeReplyNew(buf)
}

// SyncInitializeReplyNew decodes an Initialize reply.
func SyncInitializeReplyNew(buf []byte) (*SyncInitializeReply, error) {
	r := &reader{buf: buf}
	r.skip(2)
	v := &SyncInitializeReply{Sequence: r.u16(), Length: r.u32(), MajorVersion: r.u8(), MinorVersion: r.u8()}
	return v, r.err
}

func syncCounterBody(counter SyncCounter, value SyncInt64) []byte {
	var w writer
	w.u32(uint32(counter))
	w.u32(uint32(value.Hi))
	w.u32(value.Lo)
	return w.buf
}

// SyncCreateCounter sends an unchecked CreateCounter request.
func SyncCreateCounter(c *xgb.Conn, id SyncCounter, initialValue SyncInt64) xgb.Cookie {
	return *send(c, SyncExtensionName, "CreateCounter", 2, syncCounterBody(id, initialValue), false, false)
}

// SyncCreateCounterChecked sends a checked CreateCounter request.
func SyncCreateCounterChecked(c *xgb.Conn, id SyncCounter, initialValue SyncInt64) xgb.Cookie {
	return *send(c, SyncExtensionName, "CreateCounter", 2, syncCounterBody(id, initialValue), true, false)
}

// SyncSetCounter sends an unchecked SetCounter request.
func SyncSetCounter(c *xgb.Conn, counter SyncCounter, value SyncInt64) xgb.Cookie {
	return *send(c, SyncExtensionName, "SetCounter", 3, syncCounterBody(counter, value), false, false)
}

// SyncSetCounterChecked sends a checked SetCounter request.
func SyncSetCounterChecked(c *xgb.Conn, counter SyncCounter, value SyncInt64) xgb.Cookie {
	return *send(c, SyncExtensionName, "SetCounter", 3, syncCounterBody(counter, value), true, false)
}

// SyncDestroyCounter sends an unchecked DestroyCounter request.
func SyncDestroyCounter(c *xgb.Conn, counter SyncCounter) xgb.Cookie {
	var w writer
	w.u32(uint32(counter))
	return *send(c, SyncExtensionName, "DestroyCounter", 6, w.buf, false, false)
}
