//go:build linux

package x11

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	"github.com/megascan/gpui-go/platform/linux/x11/internal/xim"
)

// XimCallbackEvent is a callback the XIM handler recorded for the client to act on:
// an XimCallbackEventXimXEvent, XimCallbackEventXimPreeditEvent or XimCallbackEventXimCommitEvent.
type XimCallbackEvent interface{ isXimCallbackEvent() }

// XimCallbackEventXimXEvent is a key event the input method forwarded back.
type XimCallbackEventXimXEvent struct{ Event xgb.Event }

// XimCallbackEventXimPreeditEvent is preedit text for a window.
type XimCallbackEventXimPreeditEvent struct {
	Window xproto.Window
	Text   string
}

// XimCallbackEventXimCommitEvent is committed text for a window.
type XimCallbackEventXimCommitEvent struct {
	Window xproto.Window
	Text   string
}

func (XimCallbackEventXimXEvent) isXimCallbackEvent()       {}
func (XimCallbackEventXimPreeditEvent) isXimCallbackEvent() {}
func (XimCallbackEventXimCommitEvent) isXimCallbackEvent()  {}

// XimHandler tracks the XIM input method and input context and records callbacks.
type XimHandler struct {
	xim.NopHandler
	ImID              uint16
	IcID              uint16
	Connected         bool
	Window            xproto.Window
	LastCallbackEvent XimCallbackEvent
}

// NewXimHandler returns a handler that is not yet connected.
func NewXimHandler() *XimHandler {
	return &XimHandler{}
}

// HandleConnect opens the "C" locale.
func (h *XimHandler) HandleConnect(client *xim.Client) error {
	return client.Open("C")
}

// HandleOpen records the input method and queries its input styles.
func (h *XimHandler) HandleOpen(client *xim.Client, inputMethodID uint16) error {
	h.ImID = inputMethodID
	return client.GetIMValues(inputMethodID, []xim.AttributeName{xim.AttrQueryInputStyle})
}

// HandleGetIMValues creates the input context.
func (h *XimHandler) HandleGetIMValues(client *xim.Client, inputMethodID uint16, _ map[xim.AttributeName][]byte) error {
	icAttributes := client.BuildICAttributes().
		Long(xim.AttrInputStyle, uint32(xim.InputStylePreeditCallbacks)).
		Long(xim.AttrClientWindow, uint32(h.Window)).
		Long(xim.AttrFocusWindow, uint32(h.Window)).
		Build()
	return client.CreateIC(inputMethodID, icAttributes)
}

// HandleCreateIC records the input context and marks the handler connected.
func (h *XimHandler) HandleCreateIC(_ *xim.Client, _ uint16, inputContextID uint16) error {
	h.Connected = true
	h.IcID = inputContextID
	return nil
}

// HandleCommit records committed text.
func (h *XimHandler) HandleCommit(_ *xim.Client, _, _ uint16, text string) error {
	h.LastCallbackEvent = XimCallbackEventXimCommitEvent{Window: h.Window, Text: text}
	return nil
}

// HandleForwardEvent records a forwarded key press or release.
func (h *XimHandler) HandleForwardEvent(_ *xim.Client, _, _ uint16, _ xim.ForwardEventFlag, xev xim.XEvent) error {
	switch xev.ResponseType {
	case xproto.KeyPress:
		h.LastCallbackEvent = XimCallbackEventXimXEvent{Event: keyPressFromXim(xev)}
	case xproto.KeyRelease:
		h.LastCallbackEvent = XimCallbackEventXimXEvent{Event: xproto.KeyReleaseEvent(keyPressFromXim(xev))}
	}
	return nil
}

// HandleClose disconnects from the server.
func (h *XimHandler) HandleClose(client *xim.Client, _ uint16) error {
	return client.Disconnect()
}

// HandlePreeditDraw records preedit text; the feedbacks cannot be represented.
func (h *XimHandler) HandlePreeditDraw(_ *xim.Client, _, _ uint16, _, _, _ int32, _ xim.PreeditDrawStatus, preeditString string, _ []xim.Feedback) error {
	h.LastCallbackEvent = XimCallbackEventXimPreeditEvent{Window: h.Window, Text: preeditString}
	return nil
}

func keyPressFromXim(xev xim.XEvent) xproto.KeyPressEvent {
	return xproto.KeyPressEvent{
		Sequence:   xev.Sequence,
		Detail:     xproto.Keycode(xev.Detail),
		Time:       xproto.Timestamp(xev.Time),
		Root:       xproto.Window(xev.Root),
		Event:      xproto.Window(xev.Event),
		Child:      xproto.Window(xev.Child),
		RootX:      xev.RootX,
		RootY:      xev.RootY,
		EventX:     xev.EventX,
		EventY:     xev.EventY,
		State:      xev.State,
		SameScreen: xev.SameScreen,
	}
}

func ximFromKeyEvent(responseType uint8, ev xproto.KeyPressEvent) xim.XEvent {
	return xim.XEvent{
		ResponseType: responseType,
		Detail:       uint8(ev.Detail),
		Sequence:     ev.Sequence,
		Time:         uint32(ev.Time),
		Root:         uint32(ev.Root),
		Event:        uint32(ev.Event),
		Child:        uint32(ev.Child),
		RootX:        ev.RootX,
		RootY:        ev.RootY,
		EventX:       ev.EventX,
		EventY:       ev.EventY,
		State:        ev.State,
		SameScreen:   ev.SameScreen,
	}
}
