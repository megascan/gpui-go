//go:build linux

package xext

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/megascan/gpui-go/gpuiutil"
)

// XkbDeviceSpec is xkb::DeviceSpec.
type XkbDeviceSpec = uint16

const (
	XkbIDUseCoreKbd  XkbDeviceSpec = 256
	XkbIDUseCorePtr  XkbDeviceSpec = 512
	XkbIDDfltXIClass XkbDeviceSpec = 768
	XkbIDDfltXIId    XkbDeviceSpec = 1024
	XkbIDAllXIClass  XkbDeviceSpec = 1280
	XkbIDAllXIId     XkbDeviceSpec = 1536
	XkbIDXINone      XkbDeviceSpec = 65280
)

// XkbEventType is xkb::EventType.
type XkbEventType uint16

const (
	XkbEventTypeNewKeyboardNotify     XkbEventType = 1 << 0
	XkbEventTypeMapNotify             XkbEventType = 1 << 1
	XkbEventTypeStateNotify           XkbEventType = 1 << 2
	XkbEventTypeControlsNotify        XkbEventType = 1 << 3
	XkbEventTypeIndicatorStateNotify  XkbEventType = 1 << 4
	XkbEventTypeIndicatorMapNotify    XkbEventType = 1 << 5
	XkbEventTypeNamesNotify           XkbEventType = 1 << 6
	XkbEventTypeCompatMapNotify       XkbEventType = 1 << 7
	XkbEventTypeBellNotify            XkbEventType = 1 << 8
	XkbEventTypeActionMessage         XkbEventType = 1 << 9
	XkbEventTypeAccessXNotify         XkbEventType = 1 << 10
	XkbEventTypeExtensionDeviceNotify XkbEventType = 1 << 11
)

// XkbMapPart is xkb::MapPart.
type XkbMapPart uint16

const (
	XkbMapPartKeyTypes           XkbMapPart = 1 << 0
	XkbMapPartKeySyms            XkbMapPart = 1 << 1
	XkbMapPartModifierMap        XkbMapPart = 1 << 2
	XkbMapPartExplicitComponents XkbMapPart = 1 << 3
	XkbMapPartKeyActions         XkbMapPart = 1 << 4
	XkbMapPartKeyBehaviors       XkbMapPart = 1 << 5
	XkbMapPartVirtualMods        XkbMapPart = 1 << 6
	XkbMapPartVirtualModMap      XkbMapPart = 1 << 7
)

// XKB event numbers (the xkbType byte).
const (
	XkbNewKeyboardNotify = 0
	XkbMapNotify         = 1
	XkbStateNotify       = 2
)

// XkbUseExtensionCookie is the cookie of XkbUseExtension.
type XkbUseExtensionCookie struct{ *xgb.Cookie }

// XkbUseExtensionReply is xkb::UseExtensionReply.
type XkbUseExtensionReply struct {
	Supported   bool
	Sequence    uint16
	Length      uint32
	ServerMajor uint16
	ServerMinor uint16
}

// XkbUseExtension sends a UseExtension request.
func XkbUseExtension(c *xgb.Conn, wantedMajor, wantedMinor uint16) XkbUseExtensionCookie {
	return XkbUseExtensionCookie{send(c, XkbExtensionName, "UseExtension", 0, xkbUseExtensionBody(wantedMajor, wantedMinor), true, true)}
}

func xkbUseExtensionBody(wantedMajor, wantedMinor uint16) []byte {
	var w writer
	w.u16(wantedMajor)
	w.u16(wantedMinor)
	return w.buf
}

// Reply waits for and decodes the reply.
func (c XkbUseExtensionCookie) Reply() (*XkbUseExtensionReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	return XkbUseExtensionReplyNew(buf)
}

// XkbUseExtensionReplyNew decodes a UseExtension reply.
func XkbUseExtensionReplyNew(buf []byte) (*XkbUseExtensionReply, error) {
	r := &reader{buf: buf}
	r.skip(1)
	v := &XkbUseExtensionReply{Supported: r.u8() != 0, Sequence: r.u16(), Length: r.u32(), ServerMajor: r.u16(), ServerMinor: r.u16()}
	return v, r.err
}

// XkbSelectEventsPair16 is a 16-bit affect / details pair of SelectEventsAux.
type XkbSelectEventsPair16 struct{ Affect, Details uint16 }

// XkbSelectEventsPair32 is a 32-bit affect / details pair of SelectEventsAux.
type XkbSelectEventsPair32 struct{ Affect, Details uint32 }

// XkbSelectEventsPair8 is an 8-bit affect / details pair of SelectEventsAux.
type XkbSelectEventsPair8 struct{ Affect, Details uint8 }

// XkbSelectEventsAux is xkb::SelectEventsAux.
type XkbSelectEventsAux struct {
	NewKeyboardNotify     gpuiutil.Option[XkbSelectEventsPair16]
	StateNotify           gpuiutil.Option[XkbSelectEventsPair16]
	ControlsNotify        gpuiutil.Option[XkbSelectEventsPair32]
	IndicatorStateNotify  gpuiutil.Option[XkbSelectEventsPair32]
	IndicatorMapNotify    gpuiutil.Option[XkbSelectEventsPair32]
	NamesNotify           gpuiutil.Option[XkbSelectEventsPair16]
	CompatMapNotify       gpuiutil.Option[XkbSelectEventsPair8]
	BellNotify            gpuiutil.Option[XkbSelectEventsPair8]
	ActionMessage         gpuiutil.Option[XkbSelectEventsPair8]
	AccessXNotify         gpuiutil.Option[XkbSelectEventsPair16]
	ExtensionDeviceNotify gpuiutil.Option[XkbSelectEventsPair16]
}

func (a XkbSelectEventsAux) switchExpr() uint16 {
	var m uint16
	set := func(present bool, bit XkbEventType) {
		if present {
			m |= uint16(bit)
		}
	}
	set(a.NewKeyboardNotify.IsSome(), XkbEventTypeNewKeyboardNotify)
	set(a.StateNotify.IsSome(), XkbEventTypeStateNotify)
	set(a.ControlsNotify.IsSome(), XkbEventTypeControlsNotify)
	set(a.IndicatorStateNotify.IsSome(), XkbEventTypeIndicatorStateNotify)
	set(a.IndicatorMapNotify.IsSome(), XkbEventTypeIndicatorMapNotify)
	set(a.NamesNotify.IsSome(), XkbEventTypeNamesNotify)
	set(a.CompatMapNotify.IsSome(), XkbEventTypeCompatMapNotify)
	set(a.BellNotify.IsSome(), XkbEventTypeBellNotify)
	set(a.ActionMessage.IsSome(), XkbEventTypeActionMessage)
	set(a.AccessXNotify.IsSome(), XkbEventTypeAccessXNotify)
	set(a.ExtensionDeviceNotify.IsSome(), XkbEventTypeExtensionDeviceNotify)
	return m
}

func (a XkbSelectEventsAux) serialize(w *writer) {
	p16 := func(o gpuiutil.Option[XkbSelectEventsPair16]) {
		if v, ok := o.Get(); ok {
			w.u16(v.Affect)
			w.u16(v.Details)
		}
	}
	p32 := func(o gpuiutil.Option[XkbSelectEventsPair32]) {
		if v, ok := o.Get(); ok {
			w.u32(v.Affect)
			w.u32(v.Details)
		}
	}
	p8 := func(o gpuiutil.Option[XkbSelectEventsPair8]) {
		if v, ok := o.Get(); ok {
			w.u8(v.Affect)
			w.u8(v.Details)
		}
	}
	p16(a.NewKeyboardNotify)
	p16(a.StateNotify)
	p32(a.ControlsNotify)
	p32(a.IndicatorStateNotify)
	p32(a.IndicatorMapNotify)
	p16(a.NamesNotify)
	p8(a.CompatMapNotify)
	p8(a.BellNotify)
	p8(a.ActionMessage)
	p16(a.AccessXNotify)
	p16(a.ExtensionDeviceNotify)
}

// XkbSelectEvents sends an unchecked SelectEvents request. affectWhich is
// computed as x11rb does: the aux bitcases present | clear | selectAll.
func XkbSelectEvents(c *xgb.Conn, deviceSpec XkbDeviceSpec, clear, selectAll XkbEventType, affectMap, mapParts XkbMapPart, details XkbSelectEventsAux) xgb.Cookie {
	return *send(c, XkbExtensionName, "SelectEvents", 1, xkbSelectEventsBody(deviceSpec, clear, selectAll, affectMap, mapParts, details), false, false)
}

// XkbSelectEventsChecked sends a checked SelectEvents request.
func XkbSelectEventsChecked(c *xgb.Conn, deviceSpec XkbDeviceSpec, clear, selectAll XkbEventType, affectMap, mapParts XkbMapPart, details XkbSelectEventsAux) xgb.Cookie {
	return *send(c, XkbExtensionName, "SelectEvents", 1, xkbSelectEventsBody(deviceSpec, clear, selectAll, affectMap, mapParts, details), true, false)
}

func xkbSelectEventsBody(deviceSpec XkbDeviceSpec, clear, selectAll XkbEventType, affectMap, mapParts XkbMapPart, details XkbSelectEventsAux) []byte {
	affectWhich := details.switchExpr() | uint16(clear) | uint16(selectAll)
	var w writer
	w.u16(deviceSpec)
	w.u16(affectWhich)
	w.u16(uint16(clear))
	w.u16(uint16(selectAll))
	w.u16(uint16(affectMap))
	w.u16(uint16(mapParts))
	details.serialize(&w)
	return w.buf
}

// XkbNewKeyboardNotifyEvent is xkb::NewKeyboardNotifyEvent.
type XkbNewKeyboardNotifyEvent struct {
	ResponseType  byte
	XkbType       uint8
	Sequence      uint16
	Time          xproto.Timestamp
	DeviceID      uint8
	OldDeviceID   uint8
	MinKeyCode    xproto.Keycode
	MaxKeyCode    xproto.Keycode
	OldMinKeyCode xproto.Keycode
	OldMaxKeyCode xproto.Keycode
	RequestMajor  uint8
	RequestMinor  uint8
	Changed       uint16
}

// XkbMapNotifyEvent is xkb::MapNotifyEvent.
type XkbMapNotifyEvent struct {
	ResponseType     byte
	XkbType          uint8
	Sequence         uint16
	Time             xproto.Timestamp
	DeviceID         uint8
	PtrBtnActions    uint8
	Changed          XkbMapPart
	MinKeyCode       xproto.Keycode
	MaxKeyCode       xproto.Keycode
	FirstType        uint8
	NTypes           uint8
	FirstKeySym      xproto.Keycode
	NKeySyms         uint8
	FirstKeyAct      xproto.Keycode
	NKeyActs         uint8
	FirstKeyBehavior xproto.Keycode
	NKeyBehavior     uint8
	FirstKeyExplicit xproto.Keycode
	NKeyExplicit     uint8
	FirstModMapKey   xproto.Keycode
	NModMapKeys      uint8
	FirstVModMapKey  xproto.Keycode
	NVModMapKeys     uint8
	VirtualMods      uint16
}

// XkbStateNotifyEvent is xkb::StateNotifyEvent.
type XkbStateNotifyEvent struct {
	ResponseType      byte
	XkbType           uint8
	Sequence          uint16
	Time              xproto.Timestamp
	DeviceID          uint8
	Mods              uint8
	BaseMods          uint8
	LatchedMods       uint8
	LockedMods        uint8
	Group             uint8
	BaseGroup         int16
	LatchedGroup      int16
	LockedGroup       uint8
	CompatState       uint8
	GrabMods          uint8
	CompatGrabMods    uint8
	LookupMods        uint8
	CompatLoockupMods uint8
	PtrBtnState       uint16
	Changed           uint16
	Keycode           xproto.Keycode
	EventType         uint8
	RequestMajor      uint8
	RequestMinor      uint8
}

// XkbUnknownEvent is an XKB event this package does not decode.
type XkbUnknownEvent struct{ Data []byte }

func xkbEventNew(buf []byte) xgb.Event {
	if ev, ok := DecodeXkbEvent(buf); ok {
		return ev
	}
	return XkbUnknownEvent{Data: append([]byte(nil), buf...)}
}

// DecodeXkbEvent decodes a 32-byte XKB event by its xkbType byte.
func DecodeXkbEvent(buf []byte) (xgb.Event, bool) {
	if len(buf) < 32 {
		return nil, false
	}
	r := &reader{buf: buf}
	resp, typ, seq := r.u8(), r.u8(), r.u16()
	t := xproto.Timestamp(r.u32())
	switch typ {
	case XkbNewKeyboardNotify:
		v := XkbNewKeyboardNotifyEvent{ResponseType: resp, XkbType: typ, Sequence: seq, Time: t}
		v.DeviceID, v.OldDeviceID = r.u8(), r.u8()
		v.MinKeyCode, v.MaxKeyCode = xproto.Keycode(r.u8()), xproto.Keycode(r.u8())
		v.OldMinKeyCode, v.OldMaxKeyCode = xproto.Keycode(r.u8()), xproto.Keycode(r.u8())
		v.RequestMajor, v.RequestMinor = r.u8(), r.u8()
		v.Changed = r.u16()
		return v, r.err == nil
	case XkbMapNotify:
		v := XkbMapNotifyEvent{ResponseType: resp, XkbType: typ, Sequence: seq, Time: t}
		v.DeviceID, v.PtrBtnActions = r.u8(), r.u8()
		v.Changed = XkbMapPart(r.u16())
		v.MinKeyCode, v.MaxKeyCode = xproto.Keycode(r.u8()), xproto.Keycode(r.u8())
		v.FirstType, v.NTypes = r.u8(), r.u8()
		v.FirstKeySym, v.NKeySyms = xproto.Keycode(r.u8()), r.u8()
		v.FirstKeyAct, v.NKeyActs = xproto.Keycode(r.u8()), r.u8()
		v.FirstKeyBehavior, v.NKeyBehavior = xproto.Keycode(r.u8()), r.u8()
		v.FirstKeyExplicit, v.NKeyExplicit = xproto.Keycode(r.u8()), r.u8()
		v.FirstModMapKey, v.NModMapKeys = xproto.Keycode(r.u8()), r.u8()
		v.FirstVModMapKey, v.NVModMapKeys = xproto.Keycode(r.u8()), r.u8()
		v.VirtualMods = r.u16()
		return v, r.err == nil
	case XkbStateNotify:
		v := XkbStateNotifyEvent{ResponseType: resp, XkbType: typ, Sequence: seq, Time: t}
		v.DeviceID, v.Mods, v.BaseMods, v.LatchedMods, v.LockedMods, v.Group = r.u8(), r.u8(), r.u8(), r.u8(), r.u8(), r.u8()
		v.BaseGroup, v.LatchedGroup = int16(r.u16()), int16(r.u16())
		v.LockedGroup, v.CompatState, v.GrabMods, v.CompatGrabMods, v.LookupMods, v.CompatLoockupMods = r.u8(), r.u8(), r.u8(), r.u8(), r.u8(), r.u8()
		v.PtrBtnState, v.Changed = r.u16(), r.u16()
		v.Keycode = xproto.Keycode(r.u8())
		v.EventType, v.RequestMajor, v.RequestMinor = r.u8(), r.u8(), r.u8()
		return v, r.err == nil
	}
	return nil, false
}

func xkbHeader(w *writer, resp, typ byte, seq uint16, t xproto.Timestamp) {
	w.u8(resp)
	w.u8(typ)
	w.u16(seq)
	w.u32(uint32(t))
}

func pad32(w *writer) []byte {
	w.pad(32 - len(w.buf))
	return w.buf
}

// Bytes encodes the event.
func (v XkbNewKeyboardNotifyEvent) Bytes() []byte {
	var w writer
	xkbHeader(&w, v.ResponseType, v.XkbType, v.Sequence, v.Time)
	for _, b := range []byte{v.DeviceID, v.OldDeviceID, byte(v.MinKeyCode), byte(v.MaxKeyCode), byte(v.OldMinKeyCode), byte(v.OldMaxKeyCode), v.RequestMajor, v.RequestMinor} {
		w.u8(b)
	}
	w.u16(v.Changed)
	return pad32(&w)
}

// SequenceId returns the event's sequence number.
func (v XkbNewKeyboardNotifyEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XkbNewKeyboardNotifyEvent) String() string {
	type plain XkbNewKeyboardNotifyEvent
	return fmt.Sprintf("XkbNewKeyboardNotify %+v", plain(v))
}

// Bytes encodes the event.
func (v XkbMapNotifyEvent) Bytes() []byte {
	var w writer
	xkbHeader(&w, v.ResponseType, v.XkbType, v.Sequence, v.Time)
	w.u8(v.DeviceID)
	w.u8(v.PtrBtnActions)
	w.u16(uint16(v.Changed))
	for _, b := range []byte{byte(v.MinKeyCode), byte(v.MaxKeyCode), v.FirstType, v.NTypes, byte(v.FirstKeySym), v.NKeySyms, byte(v.FirstKeyAct), v.NKeyActs, byte(v.FirstKeyBehavior), v.NKeyBehavior, byte(v.FirstKeyExplicit), v.NKeyExplicit, byte(v.FirstModMapKey), v.NModMapKeys, byte(v.FirstVModMapKey), v.NVModMapKeys} {
		w.u8(b)
	}
	w.u16(v.VirtualMods)
	return pad32(&w)
}

// SequenceId returns the event's sequence number.
func (v XkbMapNotifyEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XkbMapNotifyEvent) String() string {
	type plain XkbMapNotifyEvent
	return fmt.Sprintf("XkbMapNotify %+v", plain(v))
}

// Bytes encodes the event.
func (v XkbStateNotifyEvent) Bytes() []byte {
	var w writer
	xkbHeader(&w, v.ResponseType, v.XkbType, v.Sequence, v.Time)
	for _, b := range []byte{v.DeviceID, v.Mods, v.BaseMods, v.LatchedMods, v.LockedMods, v.Group} {
		w.u8(b)
	}
	w.u16(uint16(v.BaseGroup))
	w.u16(uint16(v.LatchedGroup))
	for _, b := range []byte{v.LockedGroup, v.CompatState, v.GrabMods, v.CompatGrabMods, v.LookupMods, v.CompatLoockupMods} {
		w.u8(b)
	}
	w.u16(v.PtrBtnState)
	w.u16(v.Changed)
	w.u8(byte(v.Keycode))
	w.u8(v.EventType)
	w.u8(v.RequestMajor)
	w.u8(v.RequestMinor)
	return pad32(&w)
}

// SequenceId returns the event's sequence number.
func (v XkbStateNotifyEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XkbStateNotifyEvent) String() string {
	type plain XkbStateNotifyEvent
	return fmt.Sprintf("XkbStateNotify %+v", plain(v))
}

// Bytes returns the raw event.
func (v XkbUnknownEvent) Bytes() []byte { return v.Data }

// SequenceId returns the event's sequence number.
func (v XkbUnknownEvent) SequenceId() uint16 {
	if len(v.Data) < 4 {
		return 0
	}
	return xgb.Get16(v.Data[2:])
}

// String implements fmt.Stringer.
func (v XkbUnknownEvent) String() string {
	if len(v.Data) < 2 {
		return "XkbUnknownEvent {}"
	}
	return fmt.Sprintf("XkbUnknownEvent {XkbType: %d}", v.Data[1])
}
