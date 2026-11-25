//go:build linux

package screencap

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/platform/linux/internal/dbus"
)

const (
	portalDesktopService = "org.freedesktop.portal.Desktop"
	portalDesktopPath    = godbus.ObjectPath("/org/freedesktop/portal/desktop")
	portalScreenCast     = "org.freedesktop.portal.ScreenCast"
	portalRequest        = "org.freedesktop.portal.Request"
	portalCallTimeout    = 4 * time.Second
)

// PropMap is a D-Bus a{sv} dictionary.
type PropMap = map[string]godbus.Variant

type screenCastProxy struct {
	obj godbus.BusObject
}

func (p screenCastProxy) call(method string, out any, args ...any) error {
	ctx, cancel := context.WithTimeout(context.Background(), portalCallTimeout)
	defer cancel()
	call := p.obj.CallWithContext(ctx, portalScreenCast+"."+method, 0, args...)
	if call.Err != nil {
		return call.Err
	}
	if out == nil {
		return nil
	}
	return call.Store(out)
}

func (p screenCastProxy) property(name string) (uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), portalCallTimeout)
	defer cancel()
	var v godbus.Variant
	if err := p.obj.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, portalScreenCast, name).Store(&v); err != nil {
		return 0, err
	}
	u, ok := v.Value().(uint32)
	if !ok {
		return 0, fmt.Errorf("org.freedesktop.DBus.Error.InvalidArgs: property %s has type %s, want u", name, v.Signature())
	}
	return u, nil
}

// CreateSession is org.freedesktop.portal.ScreenCast.CreateSession.
func (p screenCastProxy) CreateSession(options PropMap) (godbus.ObjectPath, error) {
	var out godbus.ObjectPath
	return out, p.call("CreateSession", &out, options)
}

// SelectSources is org.freedesktop.portal.ScreenCast.SelectSources.
func (p screenCastProxy) SelectSources(sessionHandle godbus.ObjectPath, options PropMap) (godbus.ObjectPath, error) {
	var out godbus.ObjectPath
	return out, p.call("SelectSources", &out, sessionHandle, options)
}

// Start is org.freedesktop.portal.ScreenCast.Start.
func (p screenCastProxy) Start(sessionHandle godbus.ObjectPath, parentWindow string, options PropMap) (godbus.ObjectPath, error) {
	var out godbus.ObjectPath
	return out, p.call("Start", &out, sessionHandle, parentWindow, options)
}

// OpenPipeWireRemote is org.freedesktop.portal.ScreenCast.OpenPipeWireRemote; it returns the fd.
func (p screenCastProxy) OpenPipeWireRemote(sessionHandle godbus.ObjectPath, options PropMap) (int, error) {
	var out godbus.UnixFD
	if err := p.call("OpenPipeWireRemote", &out, sessionHandle, options); err != nil {
		return -1, err
	}
	return int(out), nil
}

// AvailableSourceTypes is the AvailableSourceTypes property.
func (p screenCastProxy) AvailableSourceTypes() (uint32, error) {
	return p.property("AvailableSourceTypes")
}

// AvailableCursorModes is the AvailableCursorModes property.
func (p screenCastProxy) AvailableCursorModes() (uint32, error) {
	return p.property("AvailableCursorModes")
}

// Version is the version property.
func (p screenCastProxy) Version() (uint32, error) { return p.property("version") }

func requestClose(conn *godbus.Conn, path godbus.ObjectPath) error {
	ctx, cancel := context.WithTimeout(context.Background(), portalCallTimeout)
	defer cancel()
	return conn.Object(portalDesktopService, path).CallWithContext(ctx, portalRequest+".Close", 0).Err
}

// RequestResponse is the org.freedesktop.portal.Request.Response signal.
type RequestResponse struct {
	Response uint32
	Results  PropMap
}

func readRequestResponse(sig *godbus.Signal) (RequestResponse, bool) {
	if len(sig.Body) < 2 {
		return RequestResponse{}, false
	}
	code, ok := sig.Body[0].(uint32)
	if !ok {
		return RequestResponse{}, false
	}
	results, ok := sig.Body[1].(map[string]godbus.Variant)
	if !ok {
		return RequestResponse{}, false
	}
	return RequestResponse{Response: code, Results: results}, true
}

// StreamVardict holds a portal stream's properties.
type StreamVardict struct {
	ID         *string
	Position   *[2]int32
	Size       *[2]int32
	SourceType *uint32
	MappingID  *string
}

// Stream is one stream returned by Start: the PipeWire node id and its properties.
type Stream struct {
	nodeID uint32
	props  StreamVardict
}

// PwNodeID returns the PipeWire node id.
func (s Stream) PwNodeID() uint32 { return s.nodeID }

func asIter(v any) ([]any, bool) {
	switch v := v.(type) {
	case godbus.Variant:
		return []any{v.Value()}, true
	case []any:
		return v, true
	case [][]any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = e
		}
		return out, true
	}
	return nil, false
}

func asU64(v any) (uint64, bool) {
	switch v := v.(type) {
	case uint8:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	case int16:
		return uint64(v), v >= 0
	case int32:
		return uint64(v), v >= 0
	case int64:
		return uint64(v), v >= 0
	}
	return 0, false
}

func first(v any) (any, bool) {
	items, ok := asIter(v)
	if !ok || len(items) == 0 {
		return nil, false
	}
	return items[0], true
}

// StreamFromDBus reads the first stream of Start's "streams" result.
func StreamFromDBus(stream godbus.Variant) (Stream, bool) {
	arr, ok := first(stream)
	if !ok {
		return Stream{}, false
	}
	st, ok := first(arr)
	if !ok {
		return Stream{}, false
	}
	nodeID, ok := first(st)
	if !ok {
		return Stream{}, false
	}
	id, ok := asU64(nodeID)
	if !ok {
		return Stream{}, false
	}
	return Stream{nodeID: uint32(id)}, true
}

func matchResponse(code uint32) error {
	switch code {
	case 0:
		return nil
	case 1:
		return NewLinCapError("User cancelled the interaction")
	case 2:
		return NewLinCapError("The user interaction was ended in some other way")
	}
	return NewLinCapError(fmt.Sprintf("Unknown portal response code %d", code))
}

// ScreenCastPortal drives a ScreenCast portal session on conn.
type ScreenCastPortal struct {
	conn       *godbus.Conn
	proxy      screenCastProxy
	token      string
	cursorMode uint32
}

// NewScreenCastPortal returns a portal client for conn with a fresh request token.
func NewScreenCastPortal(conn *godbus.Conn) *ScreenCastPortal {
	return &ScreenCastPortal{
		conn:       conn,
		proxy:      screenCastProxy{obj: conn.Object(portalDesktopService, portalDesktopPath)},
		token:      newPortalToken(),
		cursorMode: 1,
	}
}

func (p *ScreenCastPortal) createSessionArgs() PropMap {
	return PropMap{
		"handle_token":         godbus.MakeVariant(p.token),
		"session_handle_token": godbus.MakeVariant(p.token),
	}
}

func (p *ScreenCastPortal) selectSourcesArgs() (PropMap, error) {
	types, err := p.proxy.AvailableSourceTypes()
	if err != nil {
		return nil, err
	}
	return PropMap{
		"handle_token": godbus.MakeVariant(p.token),
		"types":        godbus.MakeVariant(types),
		"multiple":     godbus.MakeVariant(false),
		"cursor_mode":  godbus.MakeVariant(p.cursorMode),
	}, nil
}

func newPortalToken() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("scap_%016x", binary.LittleEndian.Uint64(b[:]))
}

type reqWaiter struct {
	conn *godbus.Conn
	ch   chan *godbus.Signal
	opts []godbus.MatchOption
	pin  *dbus.OwnerPin
}

func subscribeReqResponse(conn *godbus.Conn) (*reqWaiter, error) {
	opts := []godbus.MatchOption{
		godbus.WithMatchSender(portalDesktopService),
		godbus.WithMatchInterface(portalRequest),
	}
	ch := make(chan *godbus.Signal, 64)
	conn.Signal(ch)
	if err := conn.AddMatchSignal(opts...); err != nil {
		conn.RemoveSignal(ch)
		return nil, err
	}
	return &reqWaiter{conn: conn, ch: ch, opts: opts, pin: dbus.NewOwnerPin(conn, portalDesktopService)}, nil
}

func (w *reqWaiter) close() {
	_ = w.conn.RemoveMatchSignal(w.opts...)
	w.conn.RemoveSignal(w.ch)
}

func (w *reqWaiter) wait(path godbus.ObjectPath, iterations int, timeout time.Duration) *RequestResponse {
	timer := time.NewTimer(time.Duration(iterations) * timeout)
	defer timer.Stop()
	for {
		select {
		case sig, ok := <-w.ch:
			if !ok {
				return nil
			}
			if sig.Path != path || !strings.HasPrefix(sig.Name, portalRequest+".") || !w.pin.Matches(sig.Sender) {
				continue
			}
			if res, ok := readRequestResponse(sig); ok {
				return &res
			}
		case <-timer.C:
			return nil
		}
	}
}

func callWithResponse(conn *godbus.Conn, iterations int, call func() (godbus.ObjectPath, error)) (*RequestResponse, error) {
	w, err := subscribeReqResponse(conn)
	if err != nil {
		return nil, linCapErrorFrom(err)
	}
	defer w.close()
	requestHandle, err := call()
	if err != nil {
		return nil, linCapErrorFrom(err)
	}
	return w.wait(requestHandle, iterations, portalResponseTimeout), nil
}

const (
	portalResponseTimeout       = 100 * time.Millisecond
	portalShortIterations       = 100
	portalInteractiveIterations = 1200
)

func (p *ScreenCastPortal) createSession() (godbus.ObjectPath, error) {
	res, err := callWithResponse(p.conn, portalShortIterations, func() (godbus.ObjectPath, error) {
		return p.proxy.CreateSession(p.createSessionArgs())
	})
	if err != nil {
		return "", err
	}
	if res != nil {
		if err := matchResponse(res.Response); err != nil {
			return "", err
		}
		h, ok := res.Results["session_handle"]
		if !ok {
			return "", NewLinCapError("Did not get session handle")
		}
		switch v := h.Value().(type) {
		case string:
			return godbus.ObjectPath(v), nil
		case godbus.ObjectPath:
			return v, nil
		case godbus.Signature:
			return godbus.ObjectPath(v.String()), nil
		}
		return "", NewLinCapError("Invalid session_handle received")
	}
	return "", NewLinCapError("Did not get response")
}

func (p *ScreenCastPortal) selectSources(sessionHandle godbus.ObjectPath) error {
	args, err := p.selectSourcesArgs()
	if err != nil {
		return linCapErrorFrom(err)
	}
	res, err := callWithResponse(p.conn, portalInteractiveIterations, func() (godbus.ObjectPath, error) {
		return p.proxy.SelectSources(sessionHandle, args)
	})
	if err != nil {
		return err
	}
	if res != nil {
		return matchResponse(res.Response)
	}
	return NewLinCapError("Did not get response")
}

func (p *ScreenCastPortal) start(sessionHandle godbus.ObjectPath) (Stream, error) {
	res, err := callWithResponse(p.conn, portalInteractiveIterations, func() (godbus.ObjectPath, error) {
		return p.proxy.Start(sessionHandle, "", PropMap{})
	})
	if err != nil {
		return Stream{}, err
	}
	if res != nil {
		if err := matchResponse(res.Response); err != nil {
			return Stream{}, err
		}
		s, ok := res.Results["streams"]
		if !ok {
			return Stream{}, NewLinCapError("Did not get any streams")
		}
		st, ok := StreamFromDBus(s)
		if !ok {
			return Stream{}, NewLinCapError("Failed to extract stream properties")
		}
		return st, nil
	}
	return Stream{}, NewLinCapError("Did not get response")
}

// CreateStream runs CreateSession, SelectSources and Start and returns the selected stream.
func (p *ScreenCastPortal) CreateStream() (Stream, error) {
	sessionHandle, err := p.createSession()
	if err != nil {
		return Stream{}, err
	}
	if err := p.selectSources(sessionHandle); err != nil {
		return Stream{}, err
	}
	return p.start(sessionHandle)
}

// ShowCursor picks the embedded (mode) or hidden (!mode) cursor mode if the portal offers it.
func (p *ScreenCastPortal) ShowCursor(mode bool) (*ScreenCastPortal, error) {
	availableModes, err := p.proxy.AvailableCursorModes()
	if err != nil {
		return nil, linCapErrorFrom(err)
	}
	if mode && availableModes&2 == 2 {
		p.cursorMode = 2
		return p, nil
	}
	if !mode && availableModes&1 == 1 {
		p.cursorMode = 1
		return p, nil
	}
	return nil, NewLinCapError("Unsupported cursor mode")
}
