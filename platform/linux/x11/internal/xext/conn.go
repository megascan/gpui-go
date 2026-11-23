//go:build linux

package xext

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"golang.org/x/sys/unix"
)

const geGenericEvent = 35

// Conn is a framing wrapper around the X server socket. It hands xgb only the
// first 32 bytes of each X Generic Event and keeps the full event, and it
// collects file descriptors passed with SCM_RIGHTS, neither of which xgb's
// reader supports.
type Conn struct {
	net.Conn
	uc *net.UnixConn

	X *xgb.Conn

	readMu    sync.Mutex
	raw       []byte
	out       []byte
	setupDone bool

	mu      sync.Mutex
	generic [][]byte
	fds     []int
}

// NewConn wraps an established connection to the X server. The caller must
// pass the wrapper to xgb (see Dial) before any byte is read from it.
func NewConn(c net.Conn) *Conn {
	w := &Conn{Conn: c}
	if uc, ok := c.(*net.UnixConn); ok {
		w.uc = uc
	}
	return w
}

// Read implements net.Conn and returns the reframed byte stream.
func (c *Conn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	for len(c.out) == 0 {
		if err := c.nextFrame(); err != nil {
			return 0, err
		}
	}
	n := copy(p, c.out)
	c.out = c.out[n:]
	return n, nil
}

// Close closes the socket and every received descriptor not yet taken.
func (c *Conn) Close() error {
	c.mu.Lock()
	for _, fd := range c.fds {
		unix.Close(fd)
	}
	c.fds = nil
	c.generic = nil
	c.mu.Unlock()
	return c.Conn.Close()
}

func (c *Conn) need(n int) error {
	for len(c.raw) < n {
		buf := make([]byte, 4096)
		var (
			k   int
			err error
		)
		if c.uc != nil {
			oob := make([]byte, unix.CmsgSpace(4*32))
			var oobn int
			k, oobn, _, _, err = c.uc.ReadMsgUnix(buf, oob)
			if oobn > 0 {
				c.takeRights(oob[:oobn])
			}
		} else {
			k, err = c.Conn.Read(buf)
		}
		if k > 0 {
			c.raw = append(c.raw, buf[:k]...)
		}
		if err != nil {
			if len(c.raw) >= n {
				return nil
			}
			if errors.Is(err, io.EOF) && len(c.raw) > 0 {
				return io.ErrUnexpectedEOF
			}
			return err
		}
	}
	return nil
}

func (c *Conn) takeRights(oob []byte) {
	msgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return
	}
	for i := range msgs {
		fds, err := unix.ParseUnixRights(&msgs[i])
		if err != nil {
			continue
		}
		for _, fd := range fds {
			unix.CloseOnExec(fd)
		}
		c.mu.Lock()
		c.fds = append(c.fds, fds...)
		c.mu.Unlock()
	}
}

func (c *Conn) nextFrame() error {
	if !c.setupDone {
		if err := c.need(8); err != nil {
			return err
		}
		total := 8 + int(binary.LittleEndian.Uint16(c.raw[6:]))*4
		if err := c.need(total); err != nil {
			return err
		}
		c.setupDone = true
		c.emit(total)
		return nil
	}
	if err := c.need(32); err != nil {
		return err
	}
	total := 32
	kind := c.raw[0] & 127
	if c.raw[0] == 1 || (c.raw[0] != 0 && kind == geGenericEvent) {
		total += int(binary.LittleEndian.Uint32(c.raw[4:])) * 4
	}
	if err := c.need(total); err != nil {
		return err
	}
	if c.raw[0] != 0 && c.raw[0] != 1 && kind == geGenericEvent {
		full := make([]byte, total)
		copy(full, c.raw[:total])
		c.mu.Lock()
		c.generic = append(c.generic, full)
		c.mu.Unlock()
		head := make([]byte, 32)
		copy(head, c.raw[:32])
		binary.LittleEndian.PutUint32(head[4:], 0)
		c.raw = c.raw[total:]
		c.out = head
		return nil
	}
	c.emit(total)
	return nil
}

func (c *Conn) emit(total int) {
	frame := make([]byte, total)
	copy(frame, c.raw[:total])
	c.raw = c.raw[total:]
	c.out = frame
}

// TakeFd removes and returns the oldest descriptor received from the server.
func (c *Conn) TakeFd() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.fds) == 0 {
		return -1, false
	}
	fd := c.fds[0]
	c.fds = c.fds[1:]
	return fd, true
}

// NextGenericEvent removes and returns the full bytes of the oldest X Generic
// Event. Each xproto.GeGenericEvent delivered by xgb corresponds to one call.
func (c *Conn) NextGenericEvent() ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.generic) == 0 {
		return nil, false
	}
	ev := c.generic[0]
	c.generic = c.generic[1:]
	return ev, true
}

// WaitForEvent is xgb.Conn.WaitForEvent with X Generic Events replaced by
// their decoded form.
func (c *Conn) WaitForEvent() (xgb.Event, xgb.Error) {
	ev, err := c.X.WaitForEvent()
	return c.resolve(ev), err
}

// PollForEvent is xgb.Conn.PollForEvent with X Generic Events replaced by
// their decoded form.
func (c *Conn) PollForEvent() (xgb.Event, xgb.Error) {
	ev, err := c.X.PollForEvent()
	return c.resolve(ev), err
}

func (c *Conn) resolve(ev xgb.Event) xgb.Event {
	if _, ok := ev.(xproto.GeGenericEvent); !ok {
		return ev
	}
	raw, ok := c.NextGenericEvent()
	if !ok {
		return ev
	}
	return DecodeGenericEvent(c.X, raw)
}

// Dial connects to the X server named by display (DISPLAY when empty) through
// a Conn, authenticating with MIT-MAGIC-COOKIE-1 from the Xauthority file.
func Dial(display string) (*Conn, error) {
	if display == "" {
		display = os.Getenv("DISPLAY")
	}
	d, err := ParseDisplay(display)
	if err != nil {
		return nil, err
	}
	var nc net.Conn
	switch {
	case d.Socket != "":
		nc, err = net.Dial("unix", d.Socket+":"+d.Display)
	case d.Host != "" && d.Host != "unix":
		proto := d.Protocol
		if proto == "" {
			proto = "tcp"
		}
		nc, err = net.Dial(proto, d.Host+":"+strconv.Itoa(6000+d.Number))
	default:
		nc, err = net.Dial("unix", "/tmp/.X11-unix/X"+d.Display)
	}
	if err != nil {
		return nil, fmt.Errorf("xext: cannot connect to %s: %w", display, err)
	}
	w := NewConn(nc)
	cookie, ok := ReadCookie(d.Host, d.Display)
	var xc *xgb.Conn
	if ok {
		xc, err = xgb.NewConnNetWithCookieHex(w, hex.EncodeToString(cookie))
	} else {
		xc, err = xgb.NewConnNet(w)
	}
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("xext: %w", err)
	}
	xc.DisplayNumber = d.Number
	xc.DefaultScreen = d.Screen
	w.X = xc
	return w, nil
}

// DisplayName is a parsed X display string.
type DisplayName struct {
	Protocol string
	Host     string
	Socket   string
	Display  string
	Number   int
	Screen   int
}

// ParseDisplay parses an X display string the way xgb does.
func ParseDisplay(display string) (DisplayName, error) {
	var d DisplayName
	orig := display
	if display == "" {
		return d, errors.New("xext: empty display string")
	}
	colon := strings.LastIndex(display, ":")
	if colon < 0 {
		return d, errors.New("xext: bad display string: " + orig)
	}
	if display[0] == '/' {
		d.Socket = display[:colon]
	} else if slash := strings.LastIndex(display, "/"); slash >= 0 {
		d.Protocol = display[:slash]
		d.Host = display[slash+1 : colon]
	} else {
		d.Host = display[:colon]
	}
	rest := display[colon+1:]
	if rest == "" {
		return d, errors.New("xext: bad display string: " + orig)
	}
	scr := ""
	if dot := strings.LastIndex(rest, "."); dot >= 0 {
		d.Display, scr = rest[:dot], rest[dot+1:]
	} else {
		d.Display = rest
	}
	n, err := strconv.Atoi(d.Display)
	if err != nil || n < 0 {
		return d, errors.New("xext: bad display string: " + orig)
	}
	d.Number = n
	if scr != "" {
		s, err := strconv.Atoi(scr)
		if err != nil {
			return d, errors.New("xext: bad display string: " + orig)
		}
		d.Screen = s
	}
	return d, nil
}

// ReadCookie returns the MIT-MAGIC-COOKIE-1 data for host and display number
// from $XAUTHORITY or ~/.Xauthority.
func ReadCookie(host, display string) ([]byte, bool) {
	fname := os.Getenv("XAUTHORITY")
	if fname == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return nil, false
		}
		fname = home + "/.Xauthority"
	}
	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, false
	}
	if host == "" || host == "localhost" || host == "unix" {
		host, err = os.Hostname()
		if err != nil {
			return nil, false
		}
	}
	return FindCookie(data, host, display)
}

// FindCookie searches Xauthority file contents for a MIT-MAGIC-COOKIE-1 entry
// matching host (FamilyLocal) or any host (FamilyWild) and display.
func FindCookie(data []byte, host, display string) ([]byte, bool) {
	const (
		familyLocal = 256
		familyWild  = 65535
	)
	field := func() ([]byte, bool) {
		if len(data) < 2 {
			return nil, false
		}
		n := int(binary.BigEndian.Uint16(data))
		if len(data) < 2+n {
			return nil, false
		}
		v := data[2 : 2+n]
		data = data[2+n:]
		return v, true
	}
	for len(data) >= 2 {
		family := binary.BigEndian.Uint16(data)
		data = data[2:]
		addr, ok1 := field()
		disp, ok2 := field()
		name, ok3 := field()
		cookie, ok4 := field()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, false
		}
		addrOK := family == familyWild || (family == familyLocal && string(addr) == host)
		dispOK := len(disp) == 0 || string(disp) == display
		if addrOK && dispOK && string(name) == "MIT-MAGIC-COOKIE-1" && len(cookie) == 16 {
			return cookie, true
		}
	}
	return nil, false
}
