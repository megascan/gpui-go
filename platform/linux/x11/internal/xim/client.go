//go:build linux

package xim

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrNoXimServer reports that no input method server named by the locale or XMODIFIERS is running.
var ErrNoXimServer = errors.New("xim: can't connect xim server")

// ErrUnsupportedTransport reports a server whose transport is not "X/".
var ErrUnsupportedTransport = errors.New("xim: server transport is not supported")

// ErrInvalidReply reports a server reply that does not carry the expected property type or data.
var ErrInvalidReply = errors.New("xim: invalid reply from server")

// ErrMultiCM reports a frame between the transport limit and the 20-byte client message
// size, which needs the multi-client-message transport that zed-xim leaves unimplemented.
var ErrMultiCM = errors.New("xim: multi client message transport is not supported")

// ServerError is the XIM_ERROR a server sent.
type ServerError struct {
	Code   ErrorCode
	Detail string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("xim: server sent error code %d, detail: %s", e.Code, e.Detail)
}

const (
	anyPropertyType     uint32 = 0
	atomATOM            uint32 = 4
	atomSTRING          uint32 = 31
	currentTime         uint32 = 0
	defaultTransportMax        = 20
	clientMessageLength        = 20
	protocolMajor              = 1
	protocolMinor              = 0
	compoundTextName           = "COMPOUND_TEXT"
	transportPrefix            = "@transport=X/"
	serverPrefix               = "@server="
	imPrefix                   = "@im="
)

// Property is the value and type of an X property.
type Property struct {
	Type   uint32
	Format uint8
	Value  []byte
}

// Transport is the X11 surface the client needs: atoms, properties, selections and client messages.
type Transport interface {
	InternAtom(name string) (uint32, error)
	AtomName(atom uint32) (string, error)
	GetProperty(window, property, typ uint32, remove bool) (Property, error)
	AppendProperty(window, property, typ uint32, format uint8, data []byte) error
	SelectionOwner(selection uint32) (uint32, error)
	ConvertSelection(requestor, selection, target, property uint32) error
	SendClientMessage(window, messageType uint32, format uint8, data [clientMessageLength]byte) error
	Flush() error
}

// Event is one X event the client filters.
type Event interface {
	isEvent()
}

// SelectionNotify is the X SelectionNotify event.
type SelectionNotify struct {
	Requestor uint32
	Selection uint32
	Target    uint32
	Property  uint32
}

// ClientMessage is the X ClientMessage event; Data holds the 20 payload bytes.
type ClientMessage struct {
	Window uint32
	Type   uint32
	Format uint8
	Data   [clientMessageLength]byte
}

// WindowError is a BadWindow protocol error and carries the offending resource id.
type WindowError struct {
	BadValue uint32
}

func (SelectionNotify) isEvent() {}

func (ClientMessage) isEvent() {}

func (WindowError) isEvent() {}

type atomSet struct {
	ximServers  uint32
	locales     uint32
	transport   uint32
	ximXConnect uint32
	ximProtocol uint32
}

// Client is the XIM client connection; it is driven by FilterEvent and the request methods.
type Client struct {
	tr           Transport
	atoms        atomSet
	serverOwner  uint32
	serverAtom   uint32
	clientWindow uint32
	imWindow     uint32
	transportMax int
	imAttrs      map[AttributeName]uint16
	icAttrs      map[AttributeName]uint16
	sequence     uint16
}

// Init connects to the input method named imName, or XMODIFIERS' "@im=" value when imName is empty.
// root is the screen root window and clientWindow an input-only window the caller created.
func Init(tr Transport, root, clientWindow uint32, imName string) (*Client, error) {
	if imName == "" {
		if rest, ok := strings.CutPrefix(os.Getenv("XMODIFIERS"), imPrefix); ok {
			imName = rest
		}
	}
	if imName == "" {
		return nil, ErrNoXimServer
	}
	atoms, err := internAtoms(tr)
	if err != nil {
		return nil, err
	}
	servers, err := tr.GetProperty(root, atoms.ximServers, atomATOM, false)
	if err != nil {
		return nil, err
	}
	if servers.Type != atomATOM || servers.Format != 32 {
		return nil, ErrInvalidReply
	}
	for i := 0; i+4 <= len(servers.Value); i += 4 {
		server := binary.LittleEndian.Uint32(servers.Value[i:])
		name, err := tr.AtomName(server)
		if err != nil {
			return nil, err
		}
		serverName, ok := strings.CutPrefix(name, serverPrefix)
		if !ok || serverName != imName {
			continue
		}
		owner, err := tr.SelectionOwner(server)
		if err != nil {
			return nil, err
		}
		if err := tr.ConvertSelection(clientWindow, server, atoms.transport, atoms.transport); err != nil {
			return nil, err
		}
		if err := tr.Flush(); err != nil {
			return nil, err
		}
		return &Client{
			tr:           tr,
			atoms:        atoms,
			serverOwner:  owner,
			serverAtom:   server,
			clientWindow: clientWindow,
			transportMax: defaultTransportMax,
			imAttrs:      map[AttributeName]uint16{},
			icAttrs:      map[AttributeName]uint16{},
		}, nil
	}
	return nil, ErrNoXimServer
}

func internAtoms(tr Transport) (atomSet, error) {
	var atoms atomSet
	names := []struct {
		name string
		dst  *uint32
	}{
		{"XIM_SERVERS", &atoms.ximServers},
		{"LOCALES", &atoms.locales},
		{"TRANSPORT", &atoms.transport},
		{"_XIM_XCONNECT", &atoms.ximXConnect},
		{"_XIM_PROTOCOL", &atoms.ximProtocol},
	}
	for _, n := range names {
		atom, err := tr.InternAtom(n.name)
		if err != nil {
			return atomSet{}, err
		}
		*n.dst = atom
	}
	return atoms, nil
}

// FilterEvent handles the event if it belongs to this connection and reports whether it did.
func (c *Client) FilterEvent(ev Event, h Handler) (bool, error) {
	switch e := ev.(type) {
	case SelectionNotify:
		if e.Requestor != c.clientWindow {
			return false, nil
		}
		switch e.Property {
		case c.atoms.locales:
			if _, err := c.tr.GetProperty(c.clientWindow, c.atoms.locales, c.atoms.locales, true); err != nil {
				return false, err
			}
			return true, c.xconnect()
		case c.atoms.transport:
			prop, err := c.tr.GetProperty(c.clientWindow, c.atoms.transport, c.atoms.transport, true)
			if err != nil {
				return false, err
			}
			if !bytes.HasPrefix(prop.Value, []byte(transportPrefix)) {
				return false, ErrUnsupportedTransport
			}
			if err := c.tr.ConvertSelection(c.clientWindow, c.serverAtom, c.atoms.locales, c.atoms.locales); err != nil {
				return false, err
			}
			return true, c.tr.Flush()
		}
		return false, nil
	case ClientMessage:
		if e.Window != c.clientWindow {
			return false, nil
		}
		switch e.Type {
		case c.atoms.ximXConnect:
			c.imWindow = binary.LittleEndian.Uint32(e.Data[0:4])
			c.transportMax = int(binary.LittleEndian.Uint32(e.Data[12:16]))
			return true, c.send(Connect{Major: protocolMajor, Minor: protocolMinor})
		case c.atoms.ximProtocol:
			return true, c.handleProtocol(e, h)
		}
		return false, nil
	case WindowError:
		if e.BadValue == c.imWindow {
			return false, ErrNoXimServer
		}
		return false, nil
	}
	return false, nil
}

func (c *Client) handleProtocol(e ClientMessage, h Handler) error {
	if e.Format == 32 {
		atom := binary.LittleEndian.Uint32(e.Data[4:8])
		prop, err := c.tr.GetProperty(e.Window, atom, anyPropertyType, true)
		if err != nil {
			return err
		}
		if len(prop.Value) == 0 {
			return ErrInvalidReply
		}
		msg, err := Decode(prop.Value)
		if err != nil {
			return err
		}
		return c.handleRequest(h, msg)
	}
	if e.Format == 8 {
		msg, err := Decode(e.Data[:])
		if err != nil {
			return err
		}
		return c.handleRequest(h, msg)
	}
	return nil
}

func (c *Client) handleRequest(h Handler, msg Message) error {
	switch m := msg.(type) {
	case ConnectReply:
		return h.HandleConnect(c)
	case OpenReply:
		c.setAttrs(m.IMAttrs, m.ICAttrs)
		return c.send(EncodingNegotiation{IMID: m.IMID, Encodings: []string{compoundTextName}})
	case EncodingNegotiationReply:
		return h.HandleOpen(c, m.IMID)
	case QueryExtensionReply:
		return h.HandleQueryExtension(c, m.Extensions)
	case GetIMValuesReply:
		values := make(map[AttributeName][]byte, len(m.Attrs))
		for _, attr := range m.Attrs {
			for name, id := range c.imAttrs {
				if id == attr.ID {
					values[name] = attr.Value
				}
			}
		}
		return h.HandleGetIMValues(c, m.IMID, values)
	case SetICValuesReply:
		return h.HandleSetICValues(c, m.IMID, m.ICID)
	case CreateICReply:
		return h.HandleCreateIC(c, m.IMID, m.ICID)
	case DestroyICReply:
		return h.HandleDestroyIC(c, m.IMID, m.ICID)
	case SetEventMask:
		return h.HandleSetEventMask(c, m.IMID, m.ICID, m.ForwardMask, m.SyncMask)
	case CloseReply:
		return h.HandleClose(c, m.IMID)
	case DisconnectReply:
		h.HandleDisconnect()
		return nil
	case ResetICReply:
		text, err := CompoundTextToUTF8(m.PreeditString)
		if err != nil {
			return err
		}
		return h.HandleResetIC(c, m.IMID, m.ICID, text)
	case ErrorMessage:
		return &ServerError{Code: m.Code, Detail: m.Detail}
	case ForwardEvent:
		if err := h.HandleForwardEvent(c, m.IMID, m.ICID, m.Flag, m.Event); err != nil {
			return err
		}
		if m.Flag&ForwardEventSynchronous != 0 {
			return c.send(SyncReply{IMID: m.IMID, ICID: m.ICID})
		}
		return nil
	case Commit:
		return c.handleCommit(h, m)
	case Sync:
		return c.send(SyncReply{IMID: m.IMID, ICID: m.ICID})
	case PreeditStart:
		return h.HandlePreeditStart(c, m.IMID, m.ICID)
	case PreeditDone:
		return h.HandlePreeditDone(c, m.IMID, m.ICID)
	case PreeditDraw:
		text, err := CompoundTextToUTF8(m.PreeditString)
		if err != nil {
			return err
		}
		return h.HandlePreeditDraw(c, m.IMID, m.ICID, m.Caret, m.ChgFirst, m.ChgLength, m.Status, text, m.Feedbacks)
	case PreeditCaret:
		position := m.Position
		if err := h.HandlePreeditCaret(c, m.IMID, m.ICID, &position, m.Direction, m.Style); err != nil {
			return err
		}
		return c.send(PreeditCaretReply{IMID: m.IMID, ICID: m.ICID, Position: position})
	}
	return nil
}

func (c *Client) handleCommit(h Handler, m Commit) error {
	var raw []byte
	var sync, hasText bool
	switch d := m.Data.(type) {
	case CommitChars:
		raw, sync, hasText = d.Text, d.Sync, true
	case CommitBoth:
		raw, sync, hasText = d.Text, d.Sync, true
	case CommitKeysym:
		sync = d.Sync
	}
	if hasText {
		text, err := CompoundTextToUTF8(raw)
		if err != nil {
			return err
		}
		if err := h.HandleCommit(c, m.IMID, m.ICID, text); err != nil {
			return err
		}
	}
	if sync {
		return c.send(SyncReply{IMID: m.IMID, ICID: m.ICID})
	}
	return nil
}

func (c *Client) setAttrs(imAttrs, icAttrs []Attr) {
	for _, attr := range imAttrs {
		c.imAttrs[attr.Name] = attr.ID
	}
	for _, attr := range icAttrs {
		c.icAttrs[attr.Name] = attr.ID
	}
}

// BuildIMAttributes returns a builder over the IM attribute ids the server declared.
func (c *Client) BuildIMAttributes() *AttributeBuilder {
	return NewAttributeBuilder(c.imAttrs)
}

// BuildICAttributes returns a builder over the IC attribute ids the server declared.
func (c *Client) BuildICAttributes() *AttributeBuilder {
	return NewAttributeBuilder(c.icAttrs)
}

func (c *Client) xconnect() error {
	var data [clientMessageLength]byte
	binary.LittleEndian.PutUint32(data[0:4], c.clientWindow)
	if err := c.tr.SendClientMessage(c.serverOwner, c.atoms.ximXConnect, 32, data); err != nil {
		return err
	}
	return c.tr.Flush()
}

func (c *Client) send(msg Message) error {
	buf := Encode(msg)
	if len(buf) < c.transportMax {
		if len(buf) > clientMessageLength {
			return ErrMultiCM
		}
		var data [clientMessageLength]byte
		copy(data[:], buf)
		if err := c.tr.SendClientMessage(c.imWindow, c.atoms.ximProtocol, 8, data); err != nil {
			return err
		}
	} else {
		prop, err := c.tr.InternAtom(fmt.Sprintf("_XIM_DATA_%d", c.sequence))
		if err != nil {
			return err
		}
		c.sequence++
		if err := c.tr.AppendProperty(c.imWindow, prop, atomSTRING, 8, buf); err != nil {
			return err
		}
		var data [clientMessageLength]byte
		binary.LittleEndian.PutUint32(data[0:4], uint32(len(buf)))
		binary.LittleEndian.PutUint32(data[4:8], prop)
		if err := c.tr.SendClientMessage(c.imWindow, c.atoms.ximProtocol, 32, data); err != nil {
			return err
		}
	}
	return c.tr.Flush()
}

// Disconnect asks the server to end the connection.
func (c *Client) Disconnect() error {
	return c.send(Disconnect{})
}

// Open opens an input method for locale.
func (c *Client) Open(locale string) error {
	return c.send(Open{Locale: locale})
}

// Close closes the input method imID.
func (c *Client) Close(imID uint16) error {
	return c.send(Close{IMID: imID})
}

// QueryExtension asks imID which of the named extensions it supports.
func (c *Client) QueryExtension(imID uint16, names []string) error {
	return c.send(QueryExtension{IMID: imID, Extensions: names})
}

// GetIMValues requests the named IM attributes; names the server did not declare are skipped.
func (c *Client) GetIMValues(imID uint16, names []AttributeName) error {
	ids := make([]uint16, 0, len(names))
	for _, name := range names {
		if id, ok := c.imAttrs[name]; ok {
			ids = append(ids, id)
		}
	}
	return c.send(GetIMValues{IMID: imID, Names: ids})
}

// SetICValues updates attributes of the input context icID.
func (c *Client) SetICValues(imID, icID uint16, attrs []Attribute) error {
	return c.send(SetICValues{IMID: imID, ICID: icID, Attrs: attrs})
}

// CreateIC creates an input context on imID with attrs.
func (c *Client) CreateIC(imID uint16, attrs []Attribute) error {
	return c.send(CreateIC{IMID: imID, Attrs: attrs})
}

// DestroyIC destroys the input context icID.
func (c *Client) DestroyIC(imID, icID uint16) error {
	return c.send(DestroyIC{IMID: imID, ICID: icID})
}

// ForwardEvent sends a key event to the server; its serial is the event's sequence number.
func (c *Client) ForwardEvent(imID, icID uint16, flag ForwardEventFlag, event XEvent) error {
	return c.send(ForwardEvent{IMID: imID, ICID: icID, Flag: flag, Serial: event.Sequence, Event: event})
}

// SetICFocus gives the input context icID focus.
func (c *Client) SetICFocus(imID, icID uint16) error {
	return c.send(SetICFocus{IMID: imID, ICID: icID})
}

// UnsetICFocus removes focus from the input context icID.
func (c *Client) UnsetICFocus(imID, icID uint16) error {
	return c.send(UnsetICFocus{IMID: imID, ICID: icID})
}

// ResetIC resets the preedit state of the input context icID.
func (c *Client) ResetIC(imID, icID uint16) error {
	return c.send(ResetIC{IMID: imID, ICID: icID})
}
