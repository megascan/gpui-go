//go:build linux

// Package xext implements the X11 extension requests, replies and events that
// gpui_linux uses and github.com/jezek/xgb lacks (XInput 2.4, XKB, SYNC and
// DRI3), with x11rb-protocol's field layouts, plus a socket wrapper that
// frames X Generic Events and receives file descriptors.
package xext

import (
	"fmt"
	"math/bits"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	// XinputExtensionName is the X11 name of the XInput extension.
	XinputExtensionName = "XInputExtension"
	// XkbExtensionName is the X11 name of the XKB extension.
	XkbExtensionName = "XKEYBOARD"
	// SyncExtensionName is the X11 name of the SYNC extension.
	SyncExtensionName = "SYNC"
	// Dri3ExtensionName is the X11 name of the DRI3 extension.
	Dri3ExtensionName = "DRI3"
)

func init() {
	for _, name := range []string{XinputExtensionName, XkbExtensionName, SyncExtensionName, Dri3ExtensionName} {
		if xgb.NewExtEventFuncs[name] == nil {
			xgb.NewExtEventFuncs[name] = make(map[int]xgb.NewEventFun)
		}
		if xgb.NewExtErrorFuncs[name] == nil {
			xgb.NewExtErrorFuncs[name] = make(map[int]xgb.NewErrorFun)
		}
	}
	xgb.NewExtEventFuncs[XkbExtensionName][0] = xkbEventNew
	for i, n := range []string{"Device", "Event", "Mode", "DeviceBusy", "Class"} {
		xgb.NewExtErrorFuncs[XinputExtensionName][i] = extErrorNew("Xinput" + n)
	}
	xgb.NewExtErrorFuncs[XkbExtensionName][0] = extErrorNew("XkbKeyboard")
	xgb.NewExtErrorFuncs[SyncExtensionName][0] = extErrorNew("SyncCounter")
	xgb.NewExtErrorFuncs[SyncExtensionName][1] = extErrorNew("SyncAlarm")
}

// InitXinput must be called before using the XInput extension.
func InitXinput(c *xgb.Conn) error { return initExtension(c, XinputExtensionName) }

// InitXkb must be called before using the XKB extension.
func InitXkb(c *xgb.Conn) error { return initExtension(c, XkbExtensionName) }

// InitSync must be called before using the SYNC extension.
func InitSync(c *xgb.Conn) error { return initExtension(c, SyncExtensionName) }

// InitDri3 must be called before using the DRI3 extension.
func InitDri3(c *xgb.Conn) error { return initExtension(c, Dri3ExtensionName) }

func initExtension(c *xgb.Conn, name string) error {
	reply, err := xproto.QueryExtension(c, uint16(len(name)), name).Reply()
	switch {
	case err != nil:
		return err
	case !reply.Present:
		return xgb.Errorf("No extension named %s could be found on on the server.", name)
	}
	RegisterExtension(c, name, reply.MajorOpcode, reply.FirstEvent, reply.FirstError)
	return nil
}

// RegisterExtension records an extension's opcode and installs its event and
// error constructors, as the generated xgb Init functions do.
func RegisterExtension(c *xgb.Conn, name string, major, firstEvent, firstError byte) {
	c.ExtLock.Lock()
	c.Extensions[name] = major
	c.ExtLock.Unlock()
	for evNum, fun := range xgb.NewExtEventFuncs[name] {
		xgb.NewEventFuncs[int(firstEvent)+evNum] = fun
	}
	for errNum, fun := range xgb.NewExtErrorFuncs[name] {
		xgb.NewErrorFuncs[int(firstError)+errNum] = fun
	}
}

func extensionOpcode(c *xgb.Conn, ext, request string) byte {
	c.ExtLock.RLock()
	defer c.ExtLock.RUnlock()
	op, ok := c.Extensions[ext]
	if !ok {
		panic("Cannot issue request '" + request + "' using the uninitialized extension '" + ext + "'. xext.Init" + initName(ext) + "(xgb.Conn) must be called first.")
	}
	return op
}

func initName(ext string) string {
	switch ext {
	case XinputExtensionName:
		return "Xinput"
	case XkbExtensionName:
		return "Xkb"
	case SyncExtensionName:
		return "Sync"
	default:
		return "Dri3"
	}
}

func extensionFor(c *xgb.Conn, major byte) string {
	c.ExtLock.RLock()
	defer c.ExtLock.RUnlock()
	for name, op := range c.Extensions {
		if op == major {
			return name
		}
	}
	return ""
}

func send(c *xgb.Conn, ext, request string, minor byte, body []byte, checked, reply bool) *xgb.Cookie {
	buf := requestBytes(extensionOpcode(c, ext, request), minor, body)
	cookie := c.NewCookie(checked, reply)
	c.NewRequest(buf, cookie)
	return cookie
}

func requestBytes(major, minor byte, body []byte) []byte {
	size := xgb.Pad(4 + len(body))
	buf := make([]byte, size)
	buf[0] = major
	buf[1] = minor
	xgb.Put16(buf[2:], uint16(size/4))
	copy(buf[4:], body)
	return buf
}

// ExtError is an error from one of the extensions in this package.
type ExtError struct {
	Name     string
	Code     byte
	Sequence uint16
	BadValue uint32
	Minor    uint16
	Major    byte
}

func extErrorNew(name string) xgb.NewErrorFun {
	return func(buf []byte) xgb.Error {
		return ExtError{
			Name:     name,
			Code:     buf[1],
			Sequence: xgb.Get16(buf[2:]),
			BadValue: xgb.Get32(buf[4:]),
			Minor:    xgb.Get16(buf[8:]),
			Major:    buf[10],
		}
	}
}

// SequenceId returns the sequence id of the request that failed.
func (e ExtError) SequenceId() uint16 { return e.Sequence }

// BadId returns the bad value carried by the error.
func (e ExtError) BadId() uint32 { return e.BadValue }

// Error implements error.
func (e ExtError) Error() string {
	return fmt.Sprintf("%s {Sequence: %d, BadValue: %d, MinorOpcode: %d, MajorOpcode: %d}", e.Name, e.Sequence, e.BadValue, e.Minor, e.Major)
}

// GenericEvent is an X Generic Event no decoder in this package handles.
type GenericEvent struct {
	Extension byte
	Sequence  uint16
	EventType uint16
	Data      []byte
}

// DecodeGenericEvent decodes the full bytes of an X Generic Event, returning
// a GenericEvent when the extension or event type is not known.
func DecodeGenericEvent(c *xgb.Conn, raw []byte) xgb.Event {
	g := GenericEvent{Data: raw}
	if len(raw) < 10 {
		return g
	}
	g.Extension = raw[1]
	g.Sequence = xgb.Get16(raw[2:])
	g.EventType = xgb.Get16(raw[8:])
	if c != nil && extensionFor(c, g.Extension) == XinputExtensionName {
		if ev, ok := decodeXinputEvent(g.EventType, raw); ok {
			return ev
		}
	}
	return g
}

// Bytes returns the raw event.
func (g GenericEvent) Bytes() []byte { return g.Data }

// SequenceId returns the event's sequence number.
func (g GenericEvent) SequenceId() uint16 { return g.Sequence }

// String implements fmt.Stringer.
func (g GenericEvent) String() string {
	return fmt.Sprintf("GenericEvent {Extension: %d, EventType: %d, Length: %d}", g.Extension, g.EventType, len(g.Data))
}

type reader struct {
	buf []byte
	off int
	err error
}

func (r *reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if r.off+n > len(r.buf) {
		r.err = fmt.Errorf("xext: short buffer: need %d bytes at offset %d, have %d", n, r.off, len(r.buf))
		return false
	}
	return true
}

func (r *reader) u8() byte {
	if !r.need(1) {
		return 0
	}
	v := r.buf[r.off]
	r.off++
	return v
}

func (r *reader) u16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := xgb.Get16(r.buf[r.off:])
	r.off += 2
	return v
}

func (r *reader) u32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := xgb.Get32(r.buf[r.off:])
	r.off += 4
	return v
}

func (r *reader) skip(n int) {
	if r.need(n) {
		r.off += n
	}
}

func (r *reader) bytes(n int) []byte {
	if !r.need(n) {
		return nil
	}
	v := append([]byte(nil), r.buf[r.off:r.off+n]...)
	r.off += n
	return v
}

func (r *reader) u32s(n int) []uint32 {
	if n < 0 || !r.need(4*n) {
		return nil
	}
	v := make([]uint32, n)
	for i := range v {
		v[i] = r.u32()
	}
	return v
}

type writer struct{ buf []byte }

func (w *writer) u8(v byte)    { w.buf = append(w.buf, v) }
func (w *writer) u16(v uint16) { w.buf = append(w.buf, byte(v), byte(v>>8)) }
func (w *writer) u32(v uint32) {
	w.buf = append(w.buf, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}
func (w *writer) pad(n int)    { w.buf = append(w.buf, make([]byte, n)...) }
func (w *writer) raw(b []byte) { w.buf = append(w.buf, b...) }
func (w *writer) align4()      { w.pad(xgb.Pad(len(w.buf)) - len(w.buf)) }
func (w *writer) u32s(v []uint32) {
	for _, x := range v {
		w.u32(x)
	}
}

func popcount(mask []uint32) int {
	n := 0
	for _, m := range mask {
		n += bits.OnesCount32(m)
	}
	return n
}
