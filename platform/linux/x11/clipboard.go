//go:build linux

package x11

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
)

var (
	clipboardMu     sync.Mutex
	globalClipboard *globalClipboardState
)

type clipboardAtoms struct {
	CLIPBOARD         xproto.Atom
	PRIMARY           xproto.Atom
	SECONDARY         xproto.Atom
	CLIPBOARD_MANAGER xproto.Atom
	SAVE_TARGETS      xproto.Atom
	TARGETS           xproto.Atom
	ATOM              xproto.Atom
	INCR              xproto.Atom
	UTF8_STRING       xproto.Atom
	UTF8_MIME_0       xproto.Atom
	UTF8_MIME_1       xproto.Atom
	STRING            xproto.Atom
	TEXT              xproto.Atom
	TEXT_MIME_UNKNOWN xproto.Atom
	PNG__MIME         xproto.Atom
	JPEG_MIME         xproto.Atom
	WEBP_MIME         xproto.Atom
	GIF__MIME         xproto.Atom
	SVG__MIME         xproto.Atom
	BMP__MIME         xproto.Atom
	TIFF_MIME         xproto.Atom
	ICO__MIME         xproto.Atom
	PNM__MIME         xproto.Atom
	ARBOARD_CLIPBOARD xproto.Atom
}

func newClipboardAtoms(c *xgb.Conn) (clipboardAtoms, error) {
	var a clipboardAtoms
	names := []struct {
		dst  *xproto.Atom
		name string
	}{
		{&a.CLIPBOARD, "CLIPBOARD"}, {&a.PRIMARY, "PRIMARY"}, {&a.SECONDARY, "SECONDARY"},
		{&a.CLIPBOARD_MANAGER, "CLIPBOARD_MANAGER"}, {&a.SAVE_TARGETS, "SAVE_TARGETS"}, {&a.TARGETS, "TARGETS"},
		{&a.ATOM, "ATOM"}, {&a.INCR, "INCR"}, {&a.UTF8_STRING, "UTF8_STRING"},
		{&a.UTF8_MIME_0, "text/plain;charset=utf-8"}, {&a.UTF8_MIME_1, "text/plain;charset=UTF-8"},
		{&a.STRING, "STRING"}, {&a.TEXT, "TEXT"}, {&a.TEXT_MIME_UNKNOWN, "text/plain"},
		{&a.PNG__MIME, gpui.ImageFormatPng.MimeType()}, {&a.JPEG_MIME, gpui.ImageFormatJpeg.MimeType()},
		{&a.WEBP_MIME, gpui.ImageFormatWebp.MimeType()}, {&a.GIF__MIME, gpui.ImageFormatGif.MimeType()},
		{&a.SVG__MIME, gpui.ImageFormatSvg.MimeType()}, {&a.BMP__MIME, gpui.ImageFormatBmp.MimeType()},
		{&a.TIFF_MIME, gpui.ImageFormatTiff.MimeType()}, {&a.ICO__MIME, gpui.ImageFormatIco.MimeType()},
		{&a.PNM__MIME, gpui.ImageFormatPnm.MimeType()}, {&a.ARBOARD_CLIPBOARD, "ARBOARD_CLIPBOARD"},
	}
	cookies := make([]xproto.InternAtomCookie, len(names))
	for i, n := range names {
		cookies[i] = xproto.InternAtom(c, false, uint16(len(n.name)), n.name)
	}
	for i, cookie := range cookies {
		reply, err := cookie.Reply()
		if err != nil {
			return clipboardAtoms{}, err
		}
		*names[i].dst = reply.Atom
	}
	return a, nil
}

var (
	atomNameCacheMu sync.Mutex
	atomNameCache   = map[xproto.Atom]string{}
)

const (
	longTimeoutDur  = 4000 * time.Millisecond
	shortTimeoutDur = 10 * time.Millisecond
	maxIncrReadDur  = 30 * time.Second
	maxIncrReadLen  = 512 << 20
)

type incrTransfer struct {
	requestor xproto.Window
	property  xproto.Atom
	target    xproto.Atom
	data      []byte
	offset    int
}

type incrKey struct {
	requestor xproto.Window
	property  xproto.Atom
}

type managerHandoverState uint8

const (
	managerHandoverStateIdle managerHandoverState = iota
	managerHandoverStateInProgress
	managerHandoverStateFinished
)

type globalClipboardState struct {
	inner      *clipboardInner
	serverDone chan struct{}
	handles    int
}

type xContext struct {
	conn  *xgb.Conn
	winID xproto.Window
}

type clipboardInner struct {
	display string
	server  xContext
	atoms   clipboardAtoms

	clipboard selection
	primary   selection
	secondary selection

	handoverMu    sync.Mutex
	handoverState managerHandoverState
	handoverDone  chan struct{}

	serveStopped atomic.Bool

	maxPropertyBytes int
	incrTransfers    map[incrKey]*incrTransfer
}

func newXContext(display string) (xContext, error) {
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		return xContext{}, errorUnknown("X11 server connection timed out because it was unreachable")
	}
	setup := xproto.Setup(conn)
	screenNum := conn.DefaultScreen
	if screenNum < 0 || screenNum >= len(setup.Roots) {
		conn.Close()
		return xContext{}, errorUnknown("no screen found")
	}
	screen := setup.Roots[screenNum]
	winID, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return xContext{}, intoUnknown(err)
	}
	eventMask := uint32(xproto.EventMaskPropertyChange | xproto.EventMaskStructureNotify)
	xproto.CreateWindow(conn, 0, winID, screen.Root, 0, 0, 1, 1, 0, xproto.WindowClassCopyFromParent, 0,
		xproto.CwEventMask, []uint32{eventMask})
	return xContext{conn: conn, winID: winID}, nil
}

type selection struct {
	dataMu  sync.RWMutex
	data    []clipboardData
	hasData bool

	mu      sync.Mutex
	changed chan struct{}
}

func (s *selection) notifyAll() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changed != nil {
		close(s.changed)
	}
	s.changed = make(chan struct{})
	return s.changed
}

type clipboardData struct {
	bytes  []byte
	format xproto.Atom
}

type readSelNotifyResult uint8

const (
	readSelNotifyResultGotData readSelNotifyResult = iota
	readSelNotifyResultIncrStarted
	readSelNotifyResultEventNotRecognized
)

func newClipboardInner(display string) (*clipboardInner, error) {
	server, err := newXContext(display)
	if err != nil {
		return nil, err
	}
	atoms, err := newClipboardAtoms(server.conn)
	if err != nil {
		server.conn.Close()
		return nil, intoUnknown(err)
	}
	maxPropertyBytes := int(xproto.Setup(server.conn).MaximumRequestLength)*4 - 24
	if maxPropertyBytes > 256*1024 {
		maxPropertyBytes = 256 * 1024
	}
	if maxPropertyBytes < 1024 {
		maxPropertyBytes = 1024
	}
	return &clipboardInner{display: display, server: server, atoms: atoms, maxPropertyBytes: maxPropertyBytes,
		incrTransfers: map[incrKey]*incrTransfer{}}, nil
}

func (in *clipboardInner) write(data []clipboardData, kind ClipboardKind, wait WaitConfig) error {
	if in.serveStopped.Load() {
		return errorUnknown("The clipboard handler thread seems to have stopped. Logging messages may reveal the cause. (See the `log` crate.)")
	}
	serverWin := in.server.winID
	xproto.SetSelectionOwner(in.server.conn, serverWin, in.atomOf(kind), xproto.TimeCurrentTime)

	sel := in.selectionOf(kind)
	sel.dataMu.Lock()
	sel.data = data
	sel.hasData = true
	sel.dataMu.Unlock()

	waitOn := sel.notifyAll()

	switch wait.kind {
	case waitKindForever:
		<-waitOn
	case waitKindUntil:
		timer := time.NewTimer(time.Until(wait.deadline))
		select {
		case <-waitOn:
		case <-timer.C:
		}
		timer.Stop()
	}
	return nil
}

func (in *clipboardInner) read(formats []xproto.Atom, kind ClipboardKind) (clipboardData, error) {
	isOwner, err := in.isOwner(kind)
	if err != nil {
		return clipboardData{}, err
	}
	if isOwner {
		sel := in.selectionOf(kind)
		sel.dataMu.RLock()
		defer sel.dataMu.RUnlock()
		if sel.hasData {
			for _, data := range sel.data {
				for _, format := range formats {
					if format == data.format {
						return clipboardData{bytes: slices.Clone(data.bytes), format: data.format}, nil
					}
				}
			}
		}
		return clipboardData{}, ErrContentNotAvailable
	}
	reader, err := newXContext(in.display)
	if err != nil {
		return clipboardData{}, err
	}
	done := make(chan struct{})
	defer func() {
		close(done)
		reader.conn.Close()
	}()
	events := pumpEvents(reader.conn, done)

	highestPrecedenceFormat := gpui.None[xproto.Atom]()
	if data, err := in.readSingle(reader, events, kind, in.atoms.TARGETS); err != nil {
		slog.Debug(fmt.Sprintf("Clipboard TARGETS query failed with %v", err))
	} else if data.format == in.atoms.ATOM {
		available := parseFormats(data.bytes)
		for _, format := range formats {
			if slices.Contains(available, format) {
				highestPrecedenceFormat = gpui.Some(format)
				break
			}
		}
	} else {
		slog.Debug(fmt.Sprintf("Unexpected clipboard TARGETS format %s", in.atomName(data.format)))
	}

	if format, ok := highestPrecedenceFormat.Get(); ok {
		data, err := in.readSingle(reader, events, kind, format)
		if err != nil {
			return clipboardData{}, err
		}
		if !slices.Contains(formats, data.format) {
			slog.Debug(fmt.Sprintf("Conversion to %s responded with %s which is not supported",
				in.atomName(format), in.atomName(data.format)))
			return clipboardData{}, ErrConversionFailure
		}
		return data, nil
	}

	slog.Debug("Falling back on attempting to convert clipboard to each format.")
	for _, format := range formats {
		data, err := in.readSingle(reader, events, kind, format)
		if err == nil {
			if slices.Contains(formats, data.format) {
				return data, nil
			}
			slog.Debug(fmt.Sprintf("Conversion to %s responded with %s which is not supported",
				in.atomName(format), in.atomName(data.format)))
			continue
		}
		if errors.Is(err, ErrContentNotAvailable) {
			continue
		}
		slog.Debug(fmt.Sprintf("Conversion to %s failed: %v", in.atomName(format), err))
		return clipboardData{}, err
	}
	slog.Debug("All conversions to supported formats failed.")
	return clipboardData{}, ErrContentNotAvailable
}

func pumpEvents(conn *xgb.Conn, done <-chan struct{}) <-chan xgb.Event {
	events := make(chan xgb.Event, 64)
	go func() {
		defer close(events)
		for {
			ev, err := conn.WaitForEvent()
			if ev == nil && err == nil {
				return
			}
			if ev != nil {
				select {
				case events <- ev:
				case <-done:
					return
				}
			}
		}
	}()
	return events
}

func parseFormats(bytes []byte) []xproto.Atom {
	formats := make([]xproto.Atom, 0, len(bytes)/4)
	for i := 0; i+4 <= len(bytes); i += 4 {
		formats = append(formats, xproto.Atom(binary.LittleEndian.Uint32(bytes[i:])))
	}
	return formats
}

func (in *clipboardInner) readSingle(reader xContext, events <-chan xgb.Event, kind ClipboardKind, targetFormat xproto.Atom) (clipboardData, error) {
	xproto.DeleteProperty(reader.conn, reader.winID, in.atoms.ARBOARD_CLIPBOARD)
	xproto.ConvertSelection(reader.conn, reader.winID, in.atomOf(kind), targetFormat, in.atoms.ARBOARD_CLIPBOARD, xproto.TimeCurrentTime)
	if _, err := xproto.GetInputFocus(reader.conn).Reply(); err != nil {
		return clipboardData{}, intoUnknown(err)
	}
	slog.Debug("Finished `convert_selection`")

	var incrData []byte
	usingIncr := false
	timeoutEnd := time.Now().Add(longTimeoutDur)
	var incrDeadline time.Time

	for time.Now().Before(timeoutEnd) {
		timer := time.NewTimer(time.Until(timeoutEnd))
		var event xgb.Event
		var open bool
		select {
		case event, open = <-events:
		case <-timer.C:
			timer.Stop()
			continue
		}
		timer.Stop()
		if !open {
			return clipboardData{}, intoUnknown(errConnectionClosed)
		}
		switch ev := event.(type) {
		case xproto.SelectionNotifyEvent:
			slog.Debug("Read SelectionNotify")
			result, data, err := in.handleReadSelectionNotify(reader, targetFormat, &usingIncr, &incrData, ev)
			if err != nil {
				return clipboardData{}, err
			}
			switch result {
			case readSelNotifyResultGotData:
				return data, nil
			case readSelNotifyResultIncrStarted:
				timeoutEnd = timeoutEnd.Add(shortTimeoutDur)
				incrDeadline = time.Now().Add(maxIncrReadDur)
			}
		case xproto.PropertyNotifyEvent:
			done, err := in.handleReadPropertyNotify(reader, targetFormat, usingIncr, &incrData, &timeoutEnd, ev)
			if err != nil {
				return clipboardData{}, err
			}
			if done {
				format := targetFormat
				if targetFormat == in.atoms.TARGETS {
					format = in.atoms.ATOM
				}
				return clipboardData{bytes: incrData, format: format}, nil
			}
			if !incrDeadline.IsZero() && timeoutEnd.After(incrDeadline) {
				timeoutEnd = incrDeadline
			}
		default:
			slog.Debug(fmt.Sprintf("An unexpected event arrived while reading the clipboard: %v", event))
		}
	}
	slog.Info("Time-out hit while reading the clipboard.")
	return clipboardData{}, ErrContentNotAvailable
}

func (in *clipboardInner) atomOf(kind ClipboardKind) xproto.Atom {
	switch kind {
	case ClipboardKindPrimary:
		return in.atoms.PRIMARY
	case ClipboardKindSecondary:
		return in.atoms.SECONDARY
	default:
		return in.atoms.CLIPBOARD
	}
}

func (in *clipboardInner) selectionOf(kind ClipboardKind) *selection {
	switch kind {
	case ClipboardKindPrimary:
		return &in.primary
	case ClipboardKindSecondary:
		return &in.secondary
	default:
		return &in.clipboard
	}
}

func (in *clipboardInner) kindOf(atom xproto.Atom) (ClipboardKind, bool) {
	switch atom {
	case in.atoms.CLIPBOARD:
		return ClipboardKindClipboard, true
	case in.atoms.PRIMARY:
		return ClipboardKindPrimary, true
	case in.atoms.SECONDARY:
		return ClipboardKindSecondary, true
	}
	return 0, false
}

func (in *clipboardInner) isOwner(kind ClipboardKind) (bool, error) {
	reply, err := xproto.GetSelectionOwner(in.server.conn, in.atomOf(kind)).Reply()
	if err != nil {
		return false, intoUnknown(err)
	}
	if reply == nil {
		return false, intoUnknown(errConnectionClosed)
	}
	return reply.Owner == in.server.winID, nil
}

func (in *clipboardInner) queryAtomName(atom xproto.Atom) (string, error) {
	reply, err := xproto.GetAtomName(in.server.conn, atom).Reply()
	if err != nil {
		return "", intoUnknown(err)
	}
	if reply == nil {
		return "", intoUnknown(errConnectionClosed)
	}
	if !utf8.ValidString(reply.Name) {
		return "", intoUnknown(errors.New("invalid utf-8 sequence"))
	}
	return reply.Name, nil
}

func (in *clipboardInner) atomName(atom xproto.Atom) string {
	atomNameCacheMu.Lock()
	defer atomNameCacheMu.Unlock()
	if name, ok := atomNameCache[atom]; ok {
		return name
	}
	name, err := in.queryAtomName(atom)
	if err != nil {
		name = "FAILED-TO-GET-THE-ATOM-NAME"
	}
	atomNameCache[atom] = name
	return name
}

func (in *clipboardInner) handleReadSelectionNotify(reader xContext, targetFormat xproto.Atom, usingIncr *bool, incrData *[]byte, event xproto.SelectionNotifyEvent) (readSelNotifyResult, clipboardData, error) {
	if event.Property == xproto.AtomNone || event.Target != targetFormat {
		return 0, clipboardData{}, ErrContentNotAvailable
	}
	if _, ok := in.kindOf(event.Selection); !ok {
		slog.Info("Received a SelectionNotify for a selection other than CLIPBOARD, PRIMARY or SECONDARY. This is unexpected.")
		return readSelNotifyResultEventNotRecognized, clipboardData{}, nil
	}
	if *usingIncr {
		slog.Warn("Received a SelectionNotify while already expecting INCR segments.")
		return readSelNotifyResultEventNotRecognized, clipboardData{}, nil
	}
	propertyType := xproto.Atom(xproto.GetPropertyTypeAny)
	reply, err := xproto.GetProperty(reader.conn, true, event.Requestor, event.Property, propertyType, 0, math.MaxUint32/4).Reply()
	if err != nil {
		return 0, clipboardData{}, intoUnknown(err)
	}
	if reply == nil {
		return 0, clipboardData{}, intoUnknown(errConnectionClosed)
	}
	if reply.Type == in.atoms.INCR {
		reply, err = xproto.GetProperty(reader.conn, true, event.Requestor, event.Property, in.atoms.INCR, 0, math.MaxUint32/4).Reply()
		if err != nil {
			return 0, clipboardData{}, intoUnknown(err)
		}
		if reply == nil {
			return 0, clipboardData{}, intoUnknown(errConnectionClosed)
		}
		slog.Debug("Receiving INCR segments")
		*usingIncr = true
		if reply.ValueLen == 4 {
			minDataLen := uint32(0)
			if reply.Format == 32 && len(reply.Value) >= 4 {
				minDataLen = binary.LittleEndian.Uint32(reply.Value)
			}
			*incrData = slices.Grow(*incrData, int(minDataLen))
		}
		return readSelNotifyResultIncrStarted, clipboardData{}, nil
	}
	return readSelNotifyResultGotData, clipboardData{bytes: reply.Value, format: reply.Type}, nil
}

func (in *clipboardInner) handleReadPropertyNotify(reader xContext, targetFormat xproto.Atom, usingIncr bool, incrData *[]byte, timeoutEnd *time.Time, event xproto.PropertyNotifyEvent) (bool, error) {
	if event.Atom != in.atoms.ARBOARD_CLIPBOARD || event.State != xproto.PropertyNewValue {
		return false, nil
	}
	if !usingIncr {
		return false, nil
	}
	typ := targetFormat
	if targetFormat == in.atoms.TARGETS {
		typ = in.atoms.ATOM
	}
	reply, err := xproto.GetProperty(reader.conn, true, event.Window, event.Atom, typ, 0, math.MaxUint32/4).Reply()
	if err != nil {
		return false, intoUnknown(err)
	}
	if reply == nil {
		return false, intoUnknown(errConnectionClosed)
	}
	if reply.ValueLen == 0 {
		return true, nil
	}
	if len(*incrData)+len(reply.Value) > maxIncrReadLen {
		return false, errorUnknown("INCR clipboard transfer exceeded the maximum size")
	}
	*incrData = append(*incrData, reply.Value...)
	*timeoutEnd = time.Now().Add(shortTimeoutDur)
	return false, nil
}

func (in *clipboardInner) handleSelectionRequest(event xproto.SelectionRequestEvent) error {
	kind, ok := in.kindOf(event.Selection)
	if !ok {
		slog.Warn("Received a selection request to a selection other than the CLIPBOARD, PRIMARY or SECONDARY. This is unexpected.")
		return nil
	}
	conn := in.server.conn
	if event.Property == xproto.AtomNone {
		event.Property = event.Target
	}
	var success bool
	if event.Target == in.atoms.TARGETS {
		slog.Debug(fmt.Sprintf("Handling TARGETS, dst property is %s", in.atomName(event.Property)))
		targets := make([]uint32, 0, 10)
		targets = append(targets, uint32(in.atoms.TARGETS), uint32(in.atoms.SAVE_TARGETS))
		sel := in.selectionOf(kind)
		sel.dataMu.RLock()
		if sel.hasData {
			for _, data := range sel.data {
				targets = append(targets, uint32(data.format))
				if data.format == in.atoms.UTF8_STRING {
					targets = append(targets, uint32(in.atoms.UTF8_MIME_0), uint32(in.atoms.UTF8_MIME_1))
				}
			}
		}
		sel.dataMu.RUnlock()
		xproto.ChangeProperty(conn, xproto.PropModeReplace, event.Requestor, event.Property, in.atoms.ATOM, 32,
			uint32(len(targets)), u32Bytes(targets...))
		success = true
	} else {
		slog.Debug("Handling request for (probably) the clipboard contents.")
		sel := in.selectionOf(kind)
		sel.dataMu.RLock()
		if sel.hasData {
			for _, data := range sel.data {
				if in.servesTarget(data.format, event.Target) {
					in.sendSelectionData(event.Requestor, event.Property, event.Target, data.bytes)
					success = true
					break
				}
			}
		}
		sel.dataMu.RUnlock()
	}
	property := event.Property
	if !success {
		property = xproto.AtomNone
	}
	notify := xproto.SelectionNotifyEvent{
		Sequence:  event.Sequence,
		Time:      event.Time,
		Requestor: event.Requestor,
		Selection: event.Selection,
		Target:    event.Target,
		Property:  property,
	}
	xproto.SendEvent(conn, false, event.Requestor, xproto.EventMaskNoEvent, string(notify.Bytes()))
	return nil
}

func (in *clipboardInner) servesTarget(format, target xproto.Atom) bool {
	if format == target {
		return true
	}
	return format == in.atoms.UTF8_STRING && (target == in.atoms.UTF8_MIME_0 || target == in.atoms.UTF8_MIME_1)
}

func (in *clipboardInner) sendSelectionData(requestor xproto.Window, property, target xproto.Atom, bytes []byte) {
	conn := in.server.conn
	if len(bytes) <= in.maxPropertyBytes {
		xproto.ChangeProperty(conn, xproto.PropModeReplace, requestor, property, target, 8, uint32(len(bytes)), bytes)
		return
	}
	key := incrKey{requestor: requestor, property: property}
	in.incrTransfers[key] = &incrTransfer{requestor: requestor, property: property, target: target, data: slices.Clone(bytes)}
	xproto.ChangeWindowAttributes(conn, requestor, xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange})
	xproto.ChangeProperty(conn, xproto.PropModeReplace, requestor, property, in.atoms.INCR, 32, 1, u32Bytes(uint32(len(bytes))))
}

func (in *clipboardInner) handleIncrPropertyNotify(event xproto.PropertyNotifyEvent) {
	if event.State != xproto.PropertyDelete {
		return
	}
	key := incrKey{requestor: event.Window, property: event.Atom}
	transfer, ok := in.incrTransfers[key]
	if !ok {
		return
	}
	conn := in.server.conn
	end := min(transfer.offset+in.maxPropertyBytes, len(transfer.data))
	chunk := transfer.data[transfer.offset:end]
	xproto.ChangeProperty(conn, xproto.PropModeReplace, transfer.requestor, transfer.property, transfer.target, 8,
		uint32(len(chunk)), chunk)
	transfer.offset = end
	if len(chunk) == 0 {
		delete(in.incrTransfers, key)
		for k := range in.incrTransfers {
			if k.requestor == transfer.requestor {
				return
			}
		}
		xproto.ChangeWindowAttributes(conn, transfer.requestor, xproto.CwEventMask, []uint32{xproto.EventMaskNoEvent})
	}
}

func (in *clipboardInner) askClipboardManagerToRequestOurData() error {
	if in.server.winID == 0 {
		slog.Error("The server's window id was 0. This is unexpected")
		return nil
	}
	isOwner, err := in.isOwner(ClipboardKindClipboard)
	if err != nil {
		return err
	}
	if !isOwner {
		return nil
	}
	sel := in.selectionOf(ClipboardKindClipboard)
	sel.dataMu.RLock()
	hasData := sel.hasData
	sel.dataMu.RUnlock()
	if !hasData {
		return nil
	}

	in.handoverMu.Lock()
	slog.Debug("Sending the data to the clipboard manager")
	xproto.ConvertSelection(in.server.conn, in.server.winID, in.atoms.CLIPBOARD_MANAGER, in.atoms.SAVE_TARGETS,
		in.atoms.ARBOARD_CLIPBOARD, xproto.TimeCurrentTime)
	in.handoverState = managerHandoverStateInProgress
	done := make(chan struct{})
	in.handoverDone = done
	in.handoverMu.Unlock()

	const maxHandoverDuration = 100 * time.Millisecond
	timer := time.NewTimer(maxHandoverDuration)
	timedOut := false
	select {
	case <-done:
	case <-timer.C:
		timedOut = true
	}
	timer.Stop()

	in.handoverMu.Lock()
	state := in.handoverState
	in.handoverMu.Unlock()
	if state == managerHandoverStateFinished {
		return nil
	}
	if timedOut {
		slog.Warn("Could not hand the clipboard contents over to the clipboard manager. The request timed out.")
		return nil
	}
	return errorUnknown("The handover was not finished and the condvar didn't time out, yet the condvar wait ended. This should be unreachable.")
}

func (in *clipboardInner) handoverFinished() {
	slog.Debug("Finishing clipboard manager handover.")
	in.handoverState = managerHandoverStateFinished
	if in.handoverDone != nil {
		close(in.handoverDone)
		in.handoverDone = nil
	}
}

func serveRequests(context *clipboardInner) error {
	slog.Debug("Started serve requests thread.")
	defer context.serveStopped.Store(true)

	written := false
	notified := false

	for {
		event, xerr := context.server.conn.WaitForEvent()
		if event == nil && xerr == nil {
			return intoUnknown(errConnectionClosed)
		}
		if xerr != nil {
			continue
		}
		switch ev := event.(type) {
		case xproto.DestroyNotifyEvent:
			slog.Debug("Clipboard server window is being destroyed x_x")
			return nil
		case xproto.SelectionClearEvent:
			slog.Debug("Somebody else owns the clipboard now")
			if kind, ok := context.kindOf(ev.Selection); ok {
				sel := context.selectionOf(kind)
				sel.dataMu.Lock()
				sel.data = nil
				sel.hasData = false
				sel.notifyAll()
				sel.dataMu.Unlock()
			}
		case xproto.SelectionRequestEvent:
			slog.Debug(fmt.Sprintf("SelectionRequest - selection is: %s, target is %s",
				context.atomName(ev.Selection), context.atomName(ev.Target)))
			if err := context.handleSelectionRequest(ev); err != nil {
				return err
			}
			context.handoverMu.Lock()
			if context.handoverState == managerHandoverStateInProgress && ev.Target != context.atoms.TARGETS {
				slog.Debug("The contents were written to the clipboard manager.")
				written = true
				if notified {
					context.handoverFinished()
				}
			}
			context.handoverMu.Unlock()
		case xproto.PropertyNotifyEvent:
			context.handleIncrPropertyNotify(ev)
		case xproto.SelectionNotifyEvent:
			if ev.Selection != context.atoms.CLIPBOARD_MANAGER {
				slog.Error("Received a `SelectionNotify` from a selection other than the CLIPBOARD_MANAGER. This is unexpected in this thread.")
				continue
			}
			context.handoverMu.Lock()
			if context.handoverState == managerHandoverStateInProgress {
				slog.Debug("The clipboard manager indicated that it's done requesting the contents from us.")
				notified = true
				if written {
					context.handoverFinished()
				}
			}
			context.handoverMu.Unlock()
		}
	}
}

// Clipboard owns and reads the X selections through its own connection, serving requests
// for owned selections from a background goroutine.
type Clipboard struct {
	inner  *clipboardInner
	closed bool
}

// NewClipboard returns the process's clipboard, connecting it to display if it does not
// exist yet.
func NewClipboard(display string) (*Clipboard, error) {
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	if globalClipboard != nil {
		globalClipboard.handles++
		return &Clipboard{inner: globalClipboard.inner}, nil
	}
	ctx, err := newClipboardInner(display)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := serveRequests(ctx); err != nil {
			slog.Error(fmt.Sprintf("Worker thread errored with: %v", err))
		}
	}()
	globalClipboard = &globalClipboardState{inner: ctx, serverDone: done, handles: 1}
	return &Clipboard{inner: ctx}, nil
}

// SetText offers text as UTF8_STRING on the selection.
func (c *Clipboard) SetText(message string, kind ClipboardKind, wait WaitConfig) error {
	data := []clipboardData{{bytes: []byte(message), format: c.inner.atoms.UTF8_STRING}}
	return c.inner.write(data, kind, wait)
}

func (c *Clipboard) imageFormatAtom(format gpui.ImageFormat) xproto.Atom {
	a := &c.inner.atoms
	switch format {
	case gpui.ImageFormatJpeg:
		return a.JPEG_MIME
	case gpui.ImageFormatWebp:
		return a.WEBP_MIME
	case gpui.ImageFormatGif:
		return a.GIF__MIME
	case gpui.ImageFormatSvg:
		return a.SVG__MIME
	case gpui.ImageFormatBmp:
		return a.BMP__MIME
	case gpui.ImageFormatTiff:
		return a.TIFF_MIME
	case gpui.ImageFormatIco:
		return a.ICO__MIME
	case gpui.ImageFormatPnm:
		return a.PNM__MIME
	default:
		return a.PNG__MIME
	}
}

// SetImage offers image on the selection under the MIME type of its format.
func (c *Clipboard) SetImage(image gpui.Image, kind ClipboardKind, wait WaitConfig) error {
	data := []clipboardData{{bytes: image.Bytes, format: c.imageFormatAtom(image.Format)}}
	return c.inner.write(data, kind, wait)
}

// GetAny reads the selection, preferring image formats over text.
func (c *Clipboard) GetAny(kind ClipboardKind) (gpui.ClipboardItem, error) {
	type imageEntry struct {
		atom   xproto.Atom
		format gpui.ImageFormat
	}
	var imageEntries []imageEntry
	for format := range gpui.ImageFormatIter() {
		imageEntries = append(imageEntries, imageEntry{c.imageFormatAtom(format), format})
	}
	a := &c.inner.atoms
	textFormatAtoms := []xproto.Atom{a.UTF8_STRING, a.UTF8_MIME_0, a.UTF8_MIME_1, a.STRING, a.TEXT, a.TEXT_MIME_UNKNOWN}

	formatAtoms := make([]xproto.Atom, 0, len(imageEntries)+len(textFormatAtoms))
	for _, e := range imageEntries {
		formatAtoms = append(formatAtoms, e.atom)
	}
	formatAtoms = append(formatAtoms, textFormatAtoms...)

	result, err := c.inner.read(formatAtoms, kind)
	if err != nil {
		return gpui.ClipboardItem{}, err
	}
	slog.Debug(fmt.Sprintf("read clipboard as format %q", c.inner.atomName(result.format)))

	for _, e := range imageEntries {
		if result.format == e.atom {
			bytes := result.bytes
			id := gpui.Hash(bytes)
			return gpui.NewClipboardItemImage(&gpui.Image{ID: id, Format: e.format, Bytes: bytes}), nil
		}
	}

	var text string
	if result.format == a.STRING {
		runes := make([]rune, len(result.bytes))
		for i, b := range result.bytes {
			runes[i] = rune(b)
		}
		text = string(runes)
	} else {
		if !utf8.Valid(result.bytes) {
			return gpui.ClipboardItem{}, ErrConversionFailure
		}
		text = string(result.bytes)
	}
	return gpui.NewClipboardItemString(text), nil
}

// IsOwner reports whether this process owns the selection.
func (c *Clipboard) IsOwner(kind ClipboardKind) bool {
	owner, err := c.inner.isOwner(kind)
	return err == nil && owner
}

// Close is the clipboard's drop: when the last handle closes, the data is handed to the
// clipboard manager on its own goroutine and the clipboard stops serving.
func (c *Clipboard) Close() {
	if c == nil || c.closed {
		return
	}
	c.closed = true
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	global := globalClipboard
	if global == nil || global.inner != c.inner {
		return
	}
	global.handles--
	if global.handles > 0 {
		return
	}
	globalClipboard = nil
	handoversMu.Lock()
	handovers.Add(1)
	handoversMu.Unlock()
	go func() {
		defer handovers.Done()
		shutDown(global.inner, global)
	}()
}

var (
	handoversMu sync.Mutex
	handovers   sync.WaitGroup
)

// WaitForClipboardHandovers waits for closed clipboards to finish handing their data to the
// clipboard manager. It is called before the process exits.
func WaitForClipboardHandovers() {
	handoversMu.Lock()
	defer handoversMu.Unlock()
	handovers.Wait()
}

func shutDown(inner *clipboardInner, global *globalClipboardState) {
	if err := inner.askClipboardManagerToRequestOurData(); err != nil {
		slog.Error(fmt.Sprintf("Could not hand the clipboard data over to the clipboard manager: %v", err))
	}
	if err := xproto.DestroyWindowChecked(inner.server.conn, inner.server.winID).Check(); err != nil {
		slog.Error(fmt.Sprintf("Failed to destroy the clipboard window. Error: %v", err))
		inner.server.conn.Close()
		return
	}
	<-global.serverDone
	inner.server.conn.Close()
}

func intoUnknown(err error) error {
	return &Error{Kind: ErrorUnknown, Description: err.Error()}
}

func errorUnknown(message string) error {
	return &Error{Kind: ErrorUnknown, Description: message}
}

// ClipboardKind selects an X selection.
type ClipboardKind uint8

const (
	// ClipboardKindClipboard is CLIPBOARD, used for explicit cut, copy and paste.
	ClipboardKindClipboard ClipboardKind = iota
	// ClipboardKindPrimary is PRIMARY, the current mouse selection pasted with a middle click.
	ClipboardKindPrimary
	// ClipboardKindSecondary is SECONDARY, rarely used.
	ClipboardKindSecondary
)

type waitKind uint8

const (
	waitKindNone waitKind = iota
	waitKindUntil
	waitKindForever
)

// WaitConfig selects how long a write waits for the selection to change owner.
type WaitConfig struct {
	kind     waitKind
	deadline time.Time
}

// WaitConfigNone does not wait.
var WaitConfigNone = WaitConfig{}

// WaitConfigUntil waits until deadline.
func WaitConfigUntil(deadline time.Time) WaitConfig {
	return WaitConfig{kind: waitKindUntil, deadline: deadline}
}

// WaitConfigForever waits until the selection changes.
func WaitConfigForever() WaitConfig {
	return WaitConfig{kind: waitKindForever}
}

// ErrorKind selects an Error variant.
type ErrorKind uint8

const (
	// ErrorContentNotAvailable means the clipboard is empty or has no compatible format.
	ErrorContentNotAvailable ErrorKind = iota
	// ErrorClipboardOccupied means another party holds the native clipboard.
	ErrorClipboardOccupied
	// ErrorConversionFailure means the data could not be converted to the requested format.
	ErrorConversionFailure
	// ErrorUnknown is any other error; Description helps the developer.
	ErrorUnknown
)

// Error is a clipboard error.
type Error struct {
	Kind        ErrorKind
	Description string
}

var (
	// ErrContentNotAvailable is the ContentNotAvailable error.
	ErrContentNotAvailable = &Error{Kind: ErrorContentNotAvailable}
	// ErrClipboardOccupied is the ClipboardOccupied error.
	ErrClipboardOccupied = &Error{Kind: ErrorClipboardOccupied}
	// ErrConversionFailure is the ConversionFailure error.
	ErrConversionFailure = &Error{Kind: ErrorConversionFailure}
)

func (e *Error) Error() string {
	switch e.Kind {
	case ErrorContentNotAvailable:
		return "The clipboard contents were not available in the requested format or the clipboard is empty."
	case ErrorClipboardOccupied:
		return "The native clipboard is not accessible due to being held by an other party."
	case ErrorConversionFailure:
		return "The image or the text that was about the be transferred to/from the clipboard could not be converted to the appropriate format."
	default:
		return "Unknown error while interacting with the clipboard: " + e.Description
	}
}

// Is matches errors of the same kind, so errors.Is(err, ErrContentNotAvailable) works.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && (t.Kind != ErrorUnknown || t.Description == e.Description)
}
