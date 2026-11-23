//go:build linux

package xext

import (
	"errors"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Dri3QueryVersionCookie is the cookie of Dri3QueryVersion.
type Dri3QueryVersionCookie struct{ *xgb.Cookie }

// Dri3QueryVersionReply is dri3::QueryVersionReply.
type Dri3QueryVersionReply struct {
	Sequence     uint16
	Length       uint32
	MajorVersion uint32
	MinorVersion uint32
}

// Dri3QueryVersion sends a QueryVersion request.
func Dri3QueryVersion(c *xgb.Conn, majorVersion, minorVersion uint32) Dri3QueryVersionCookie {
	var w writer
	w.u32(majorVersion)
	w.u32(minorVersion)
	return Dri3QueryVersionCookie{send(c, Dri3ExtensionName, "QueryVersion", 0, w.buf, true, true)}
}

// Reply waits for and decodes the reply.
func (c Dri3QueryVersionCookie) Reply() (*Dri3QueryVersionReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	return Dri3QueryVersionReplyNew(buf)
}

// Dri3QueryVersionReplyNew decodes a QueryVersion reply.
func Dri3QueryVersionReplyNew(buf []byte) (*Dri3QueryVersionReply, error) {
	r := &reader{buf: buf}
	r.skip(2)
	v := &Dri3QueryVersionReply{Sequence: r.u16(), Length: r.u32(), MajorVersion: r.u32(), MinorVersion: r.u32()}
	return v, r.err
}

// Dri3OpenCookie is the cookie of Dri3Open.
type Dri3OpenCookie struct {
	*xgb.Cookie
	conn *Conn
}

// Dri3OpenReply is dri3::OpenReply. The caller owns DeviceFd.
type Dri3OpenReply struct {
	Nfd      uint8
	Sequence uint16
	Length   uint32
	DeviceFd int
}

// Dri3Open sends an Open request. The device descriptor arrives with
// SCM_RIGHTS, so the request goes through the Conn that frames the socket.
func Dri3Open(c *Conn, drawable xproto.Drawable, provider uint32) Dri3OpenCookie {
	var w writer
	w.u32(uint32(drawable))
	w.u32(provider)
	return Dri3OpenCookie{Cookie: send(c.X, Dri3ExtensionName, "Open", 1, w.buf, true, true), conn: c}
}

// Reply waits for the reply and takes the passed descriptor.
func (c Dri3OpenCookie) Reply() (*Dri3OpenReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	v, err := Dri3OpenReplyNew(buf)
	if err != nil {
		return nil, err
	}
	if v.Nfd != 1 {
		return nil, errors.New("xext: DRI3 Open reply carries no file descriptor")
	}
	fd, ok := c.conn.TakeFd()
	if !ok {
		return nil, errors.New("xext: DRI3 Open reply file descriptor was not received")
	}
	v.DeviceFd = fd
	return v, nil
}

// Dri3OpenReplyNew decodes an Open reply without its descriptor.
func Dri3OpenReplyNew(buf []byte) (*Dri3OpenReply, error) {
	r := &reader{buf: buf}
	r.skip(1)
	v := &Dri3OpenReply{Nfd: r.u8(), Sequence: r.u16(), Length: r.u32(), DeviceFd: -1}
	return v, r.err
}
