//go:build linux

package xext

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// XinputDeviceID is xinput::DeviceId.
type XinputDeviceID = uint16

// XinputFp1616 is a 16.16 fixed point number.
type XinputFp1616 = int32

// XinputFp3232 is a 32.32 fixed point number.
type XinputFp3232 struct {
	Integral int32
	Frac     uint32
}

const (
	XinputDeviceAll       XinputDeviceID = 0
	XinputDeviceAllMaster XinputDeviceID = 1
)

// XinputDeviceType is xinput::DeviceType.
type XinputDeviceType uint16

const (
	XinputDeviceTypeMasterPointer  XinputDeviceType = 1
	XinputDeviceTypeMasterKeyboard XinputDeviceType = 2
	XinputDeviceTypeSlavePointer   XinputDeviceType = 3
	XinputDeviceTypeSlaveKeyboard  XinputDeviceType = 4
	XinputDeviceTypeFloatingSlave  XinputDeviceType = 5
)

// XinputDeviceClassType is xinput::DeviceClassType.
type XinputDeviceClassType uint16

const (
	XinputDeviceClassTypeKey      XinputDeviceClassType = 0
	XinputDeviceClassTypeButton   XinputDeviceClassType = 1
	XinputDeviceClassTypeValuator XinputDeviceClassType = 2
	XinputDeviceClassTypeScroll   XinputDeviceClassType = 3
	XinputDeviceClassTypeTouch    XinputDeviceClassType = 8
	XinputDeviceClassTypeGesture  XinputDeviceClassType = 9
)

// XinputScrollType is xinput::ScrollType.
type XinputScrollType uint16

const (
	XinputScrollTypeVertical   XinputScrollType = 1
	XinputScrollTypeHorizontal XinputScrollType = 2
)

const (
	XinputScrollFlagsNoEmulation uint32 = 1 << 0
	XinputScrollFlagsPreferred   uint32 = 1 << 1
)

// XinputNotifyMode is xinput::NotifyMode.
type XinputNotifyMode uint8

const (
	XinputNotifyModeNormal        XinputNotifyMode = 0
	XinputNotifyModeGrab          XinputNotifyMode = 1
	XinputNotifyModeUngrab        XinputNotifyMode = 2
	XinputNotifyModeWhileGrabbed  XinputNotifyMode = 3
	XinputNotifyModePassiveGrab   XinputNotifyMode = 4
	XinputNotifyModePassiveUngrab XinputNotifyMode = 5
)

// XinputNotifyDetail is xinput::NotifyDetail.
type XinputNotifyDetail uint8

const (
	XinputNotifyDetailAncestor         XinputNotifyDetail = 0
	XinputNotifyDetailVirtual          XinputNotifyDetail = 1
	XinputNotifyDetailInferior         XinputNotifyDetail = 2
	XinputNotifyDetailNonlinear        XinputNotifyDetail = 3
	XinputNotifyDetailNonlinearVirtual XinputNotifyDetail = 4
	XinputNotifyDetailPointer          XinputNotifyDetail = 5
	XinputNotifyDetailPointerRoot      XinputNotifyDetail = 6
	XinputNotifyDetailNone             XinputNotifyDetail = 7
)

// XinputXIEventMask is xinput::XIEventMask.
type XinputXIEventMask uint32

const (
	XinputXIEventMaskDeviceChanged    XinputXIEventMask = 1 << 1
	XinputXIEventMaskKeyPress         XinputXIEventMask = 1 << 2
	XinputXIEventMaskKeyRelease       XinputXIEventMask = 1 << 3
	XinputXIEventMaskButtonPress      XinputXIEventMask = 1 << 4
	XinputXIEventMaskButtonRelease    XinputXIEventMask = 1 << 5
	XinputXIEventMaskMotion           XinputXIEventMask = 1 << 6
	XinputXIEventMaskEnter            XinputXIEventMask = 1 << 7
	XinputXIEventMaskLeave            XinputXIEventMask = 1 << 8
	XinputXIEventMaskFocusIn          XinputXIEventMask = 1 << 9
	XinputXIEventMaskFocusOut         XinputXIEventMask = 1 << 10
	XinputXIEventMaskHierarchy        XinputXIEventMask = 1 << 11
	XinputXIEventMaskProperty         XinputXIEventMask = 1 << 12
	XinputXIEventMaskRawKeyPress      XinputXIEventMask = 1 << 13
	XinputXIEventMaskRawKeyRelease    XinputXIEventMask = 1 << 14
	XinputXIEventMaskRawButtonPress   XinputXIEventMask = 1 << 15
	XinputXIEventMaskRawButtonRelease XinputXIEventMask = 1 << 16
	XinputXIEventMaskRawMotion        XinputXIEventMask = 1 << 17
	XinputXIEventMaskTouchBegin       XinputXIEventMask = 1 << 18
	XinputXIEventMaskTouchUpdate      XinputXIEventMask = 1 << 19
	XinputXIEventMaskTouchEnd         XinputXIEventMask = 1 << 20
	XinputXIEventMaskTouchOwnership   XinputXIEventMask = 1 << 21
	XinputXIEventMaskRawTouchBegin    XinputXIEventMask = 1 << 22
	XinputXIEventMaskRawTouchUpdate   XinputXIEventMask = 1 << 23
	XinputXIEventMaskRawTouchEnd      XinputXIEventMask = 1 << 24
	XinputXIEventMaskBarrierHit       XinputXIEventMask = 1 << 25
	XinputXIEventMaskBarrierLeave     XinputXIEventMask = 1 << 26
)

const (
	XinputPointerEventFlagsPointerEmulated uint32 = 1 << 16
	XinputKeyEventFlagsKeyRepeat           uint32 = 1 << 16
	XinputGesturePinchEventFlagsCancelled  uint32 = 1 << 0
)

const (
	XinputHierarchyMaskMasterAdded    uint32 = 1 << 0
	XinputHierarchyMaskMasterRemoved  uint32 = 1 << 1
	XinputHierarchyMaskSlaveAdded     uint32 = 1 << 2
	XinputHierarchyMaskSlaveRemoved   uint32 = 1 << 3
	XinputHierarchyMaskSlaveAttached  uint32 = 1 << 4
	XinputHierarchyMaskSlaveDetached  uint32 = 1 << 5
	XinputHierarchyMaskDeviceEnabled  uint32 = 1 << 6
	XinputHierarchyMaskDeviceDisabled uint32 = 1 << 7
)

const (
	XinputChangeReasonSlaveSwitch  uint8 = 1
	XinputChangeReasonDeviceChange uint8 = 2
)

// XInput 2 event numbers.
const (
	XinputDeviceChanged      = 1
	XinputKeyPress           = 2
	XinputKeyRelease         = 3
	XinputButtonPress        = 4
	XinputButtonRelease      = 5
	XinputMotion             = 6
	XinputEnter              = 7
	XinputLeave              = 8
	XinputFocusIn            = 9
	XinputFocusOut           = 10
	XinputHierarchy          = 11
	XinputGesturePinchBegin  = 27
	XinputGesturePinchUpdate = 28
	XinputGesturePinchEnd    = 29
)

// XinputModifierInfo is xinput::ModifierInfo.
type XinputModifierInfo struct {
	Base, Latched, Locked, Effective uint32
}

// XinputGroupInfo is xinput::GroupInfo.
type XinputGroupInfo struct {
	Base, Latched, Locked, Effective uint8
}

func readFp3232(r *reader) XinputFp3232 {
	return XinputFp3232{Integral: int32(r.u32()), Frac: r.u32()}
}

func writeFp3232(w *writer, v XinputFp3232) {
	w.u32(uint32(v.Integral))
	w.u32(v.Frac)
}

func readModifierInfo(r *reader) XinputModifierInfo {
	return XinputModifierInfo{Base: r.u32(), Latched: r.u32(), Locked: r.u32(), Effective: r.u32()}
}

func writeModifierInfo(w *writer, v XinputModifierInfo) {
	w.u32(v.Base)
	w.u32(v.Latched)
	w.u32(v.Locked)
	w.u32(v.Effective)
}

func readGroupInfo(r *reader) XinputGroupInfo {
	return XinputGroupInfo{Base: r.u8(), Latched: r.u8(), Locked: r.u8(), Effective: r.u8()}
}

func writeGroupInfo(w *writer, v XinputGroupInfo) {
	w.u8(v.Base)
	w.u8(v.Latched)
	w.u8(v.Locked)
	w.u8(v.Effective)
}

// XinputDeviceClassData is the type-specific part of a device class.
type XinputDeviceClassData interface {
	xinputDeviceClassData()
}

// XinputDeviceClassDataKey is xinput::DeviceClassDataKey.
type XinputDeviceClassDataKey struct{ Keys []uint32 }

// XinputDeviceClassDataButton is xinput::DeviceClassDataButton.
type XinputDeviceClassDataButton struct {
	NumButtons uint16
	State      []uint32
	Labels     []xproto.Atom
}

// XinputDeviceClassDataValuator is xinput::DeviceClassDataValuator.
type XinputDeviceClassDataValuator struct {
	Number     uint16
	Label      xproto.Atom
	Min        XinputFp3232
	Max        XinputFp3232
	Value      XinputFp3232
	Resolution uint32
	Mode       uint8
}

// XinputDeviceClassDataScroll is xinput::DeviceClassDataScroll.
type XinputDeviceClassDataScroll struct {
	Number     uint16
	ScrollType XinputScrollType
	Flags      uint32
	Increment  XinputFp3232
}

// XinputDeviceClassDataTouch is xinput::DeviceClassDataTouch.
type XinputDeviceClassDataTouch struct {
	Mode       uint8
	NumTouches uint8
}

// XinputDeviceClassDataGesture is xinput::DeviceClassDataGesture.
type XinputDeviceClassDataGesture struct{ NumTouches uint8 }

// XinputDeviceClassDataUnknown holds the bytes of an unrecognised class.
type XinputDeviceClassDataUnknown struct{ Data []byte }

func (XinputDeviceClassDataKey) xinputDeviceClassData()      {}
func (XinputDeviceClassDataButton) xinputDeviceClassData()   {}
func (XinputDeviceClassDataValuator) xinputDeviceClassData() {}
func (XinputDeviceClassDataScroll) xinputDeviceClassData()   {}
func (XinputDeviceClassDataTouch) xinputDeviceClassData()    {}
func (XinputDeviceClassDataGesture) xinputDeviceClassData()  {}
func (XinputDeviceClassDataUnknown) xinputDeviceClassData()  {}

// XinputDeviceClass is xinput::DeviceClass.
type XinputDeviceClass struct {
	Type     XinputDeviceClassType
	Sourceid XinputDeviceID
	Data     XinputDeviceClassData
}

// AsScroll returns the scroll data when the class is a scroll class.
func (d XinputDeviceClass) AsScroll() (XinputDeviceClassDataScroll, bool) {
	s, ok := d.Data.(XinputDeviceClassDataScroll)
	return s, ok
}

func readDeviceClass(r *reader) XinputDeviceClass {
	typ := r.u16()
	length := int(r.u16()) * 4
	src := r.u16()
	if length < 6 || !r.need(length-6) {
		if r.err == nil {
			r.err = fmt.Errorf("xext: bad DeviceClass length %d", length)
		}
		return XinputDeviceClass{}
	}
	body := &reader{buf: r.buf[r.off : r.off+length-6]}
	r.off += length - 6
	dc := XinputDeviceClass{Type: XinputDeviceClassType(typ), Sourceid: src}
	switch dc.Type {
	case XinputDeviceClassTypeKey:
		n := int(body.u16())
		dc.Data = XinputDeviceClassDataKey{Keys: body.u32s(n)}
	case XinputDeviceClassTypeButton:
		n := body.u16()
		state := body.u32s((int(n) + 31) / 32)
		labels := make([]xproto.Atom, n)
		for i := range labels {
			labels[i] = xproto.Atom(body.u32())
		}
		dc.Data = XinputDeviceClassDataButton{NumButtons: n, State: state, Labels: labels}
	case XinputDeviceClassTypeValuator:
		v := XinputDeviceClassDataValuator{Number: body.u16(), Label: xproto.Atom(body.u32())}
		v.Min = readFp3232(body)
		v.Max = readFp3232(body)
		v.Value = readFp3232(body)
		v.Resolution = body.u32()
		v.Mode = body.u8()
		body.skip(3)
		dc.Data = v
	case XinputDeviceClassTypeScroll:
		s := XinputDeviceClassDataScroll{Number: body.u16(), ScrollType: XinputScrollType(body.u16())}
		body.skip(2)
		s.Flags = body.u32()
		s.Increment = readFp3232(body)
		dc.Data = s
	case XinputDeviceClassTypeTouch:
		dc.Data = XinputDeviceClassDataTouch{Mode: body.u8(), NumTouches: body.u8()}
	case XinputDeviceClassTypeGesture:
		dc.Data = XinputDeviceClassDataGesture{NumTouches: body.u8()}
	default:
		dc.Data = XinputDeviceClassDataUnknown{Data: append([]byte(nil), body.buf...)}
	}
	if body.err != nil && r.err == nil {
		r.err = body.err
	}
	return dc
}

func writeDeviceClass(w *writer, dc XinputDeviceClass) {
	var b writer
	switch d := dc.Data.(type) {
	case XinputDeviceClassDataKey:
		b.u16(uint16(len(d.Keys)))
		b.u32s(d.Keys)
	case XinputDeviceClassDataButton:
		b.u16(d.NumButtons)
		b.u32s(d.State)
		for _, l := range d.Labels {
			b.u32(uint32(l))
		}
	case XinputDeviceClassDataValuator:
		b.u16(d.Number)
		b.u32(uint32(d.Label))
		writeFp3232(&b, d.Min)
		writeFp3232(&b, d.Max)
		writeFp3232(&b, d.Value)
		b.u32(d.Resolution)
		b.u8(d.Mode)
		b.pad(3)
	case XinputDeviceClassDataScroll:
		b.u16(d.Number)
		b.u16(uint16(d.ScrollType))
		b.pad(2)
		b.u32(d.Flags)
		writeFp3232(&b, d.Increment)
	case XinputDeviceClassDataTouch:
		b.u8(d.Mode)
		b.u8(d.NumTouches)
	case XinputDeviceClassDataGesture:
		b.u8(d.NumTouches)
		b.pad(1)
	case XinputDeviceClassDataUnknown:
		b.raw(d.Data)
	}
	w.u16(uint16(dc.Type))
	w.u16(uint16((6 + len(b.buf) + 3) / 4))
	w.u16(dc.Sourceid)
	w.raw(b.buf)
	w.pad(xgb.Pad(6+len(b.buf)) - 6 - len(b.buf))
}

// XinputXIDeviceInfo is xinput::XIDeviceInfo.
type XinputXIDeviceInfo struct {
	Deviceid   XinputDeviceID
	Type       XinputDeviceType
	Attachment XinputDeviceID
	Enabled    bool
	Name       []byte
	Classes    []XinputDeviceClass
}

// XinputEventMask is xinput::EventMask.
type XinputEventMask struct {
	Deviceid XinputDeviceID
	Mask     []XinputXIEventMask
}

// XinputHierarchyInfo is xinput::HierarchyInfo.
type XinputHierarchyInfo struct {
	Deviceid   XinputDeviceID
	Attachment XinputDeviceID
	Type       XinputDeviceType
	Enabled    bool
	Flags      uint32
}

// XinputXIQueryVersionCookie is the cookie of XinputXIQueryVersion.
type XinputXIQueryVersionCookie struct{ *xgb.Cookie }

// XinputXIQueryVersionReply is xinput::XIQueryVersionReply.
type XinputXIQueryVersionReply struct {
	Sequence     uint16
	Length       uint32
	MajorVersion uint16
	MinorVersion uint16
}

// XinputXIQueryVersion sends an XIQueryVersion request.
func XinputXIQueryVersion(c *xgb.Conn, majorVersion, minorVersion uint16) XinputXIQueryVersionCookie {
	return XinputXIQueryVersionCookie{send(c, XinputExtensionName, "XIQueryVersion", 47, xinputXIQueryVersionBody(majorVersion, minorVersion), true, true)}
}

func xinputXIQueryVersionBody(majorVersion, minorVersion uint16) []byte {
	var w writer
	w.u16(majorVersion)
	w.u16(minorVersion)
	return w.buf
}

// Reply waits for and decodes the reply.
func (c XinputXIQueryVersionCookie) Reply() (*XinputXIQueryVersionReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	return XinputXIQueryVersionReplyNew(buf)
}

// XinputXIQueryVersionReplyNew decodes an XIQueryVersion reply.
func XinputXIQueryVersionReplyNew(buf []byte) (*XinputXIQueryVersionReply, error) {
	r := &reader{buf: buf}
	r.skip(2)
	v := &XinputXIQueryVersionReply{Sequence: r.u16(), Length: r.u32(), MajorVersion: r.u16(), MinorVersion: r.u16()}
	return v, r.err
}

// XinputXIQueryDeviceCookie is the cookie of XinputXIQueryDevice.
type XinputXIQueryDeviceCookie struct{ *xgb.Cookie }

// XinputXIQueryDeviceReply is xinput::XIQueryDeviceReply.
type XinputXIQueryDeviceReply struct {
	Sequence uint16
	Length   uint32
	Infos    []XinputXIDeviceInfo
}

// XinputXIQueryDevice sends an XIQueryDevice request.
func XinputXIQueryDevice(c *xgb.Conn, deviceid XinputDeviceID) XinputXIQueryDeviceCookie {
	return XinputXIQueryDeviceCookie{send(c, XinputExtensionName, "XIQueryDevice", 48, xinputXIQueryDeviceBody(deviceid), true, true)}
}

func xinputXIQueryDeviceBody(deviceid XinputDeviceID) []byte {
	var w writer
	w.u16(deviceid)
	w.pad(2)
	return w.buf
}

// Reply waits for and decodes the reply.
func (c XinputXIQueryDeviceCookie) Reply() (*XinputXIQueryDeviceReply, error) {
	buf, err := c.Cookie.Reply()
	if err != nil || buf == nil {
		return nil, err
	}
	return XinputXIQueryDeviceReplyNew(buf)
}

// XinputXIQueryDeviceReplyNew decodes an XIQueryDevice reply.
func XinputXIQueryDeviceReplyNew(buf []byte) (*XinputXIQueryDeviceReply, error) {
	r := &reader{buf: buf}
	r.skip(2)
	v := &XinputXIQueryDeviceReply{Sequence: r.u16(), Length: r.u32()}
	n := int(r.u16())
	r.skip(22)
	for i := 0; i < n && r.err == nil; i++ {
		info := XinputXIDeviceInfo{Deviceid: r.u16(), Type: XinputDeviceType(r.u16()), Attachment: r.u16()}
		numClasses := int(r.u16())
		nameLen := int(r.u16())
		info.Enabled = r.u8() != 0
		r.skip(1)
		info.Name = r.bytes(nameLen)
		r.skip(xgb.Pad(nameLen) - nameLen)
		for j := 0; j < numClasses && r.err == nil; j++ {
			info.Classes = append(info.Classes, readDeviceClass(r))
		}
		v.Infos = append(v.Infos, info)
	}
	return v, r.err
}

// XinputXISelectEvents sends an unchecked XISelectEvents request.
func XinputXISelectEvents(c *xgb.Conn, window xproto.Window, masks []XinputEventMask) xgb.Cookie {
	return *send(c, XinputExtensionName, "XISelectEvents", 46, xinputXISelectEventsBody(window, masks), false, false)
}

// XinputXISelectEventsChecked sends a checked XISelectEvents request.
func XinputXISelectEventsChecked(c *xgb.Conn, window xproto.Window, masks []XinputEventMask) xgb.Cookie {
	return *send(c, XinputExtensionName, "XISelectEvents", 46, xinputXISelectEventsBody(window, masks), true, false)
}

func xinputXISelectEventsBody(window xproto.Window, masks []XinputEventMask) []byte {
	var w writer
	w.u32(uint32(window))
	w.u16(uint16(len(masks)))
	w.pad(2)
	for _, m := range masks {
		w.u16(m.Deviceid)
		w.u16(uint16(len(m.Mask)))
		for _, x := range m.Mask {
			w.u32(uint32(x))
		}
	}
	return w.buf
}

func geHeader(w *writer, ext byte, seq uint16, evtype uint16) {
	w.u8(geGenericEvent)
	w.u8(ext)
	w.u16(seq)
	w.u32(0)
	w.u16(evtype)
}

func geFinish(w *writer) []byte {
	if len(w.buf) < 32 {
		w.pad(32 - len(w.buf))
	}
	w.align4()
	xgb.Put32(w.buf[4:], uint32((len(w.buf)-32)/4))
	return w.buf
}

// XinputButtonPressEvent is xinput::ButtonPressEvent; ButtonRelease and
// Motion share its layout.
type XinputButtonPressEvent struct {
	Extension    byte
	Sequence     uint16
	EventType    uint16
	Deviceid     XinputDeviceID
	Time         xproto.Timestamp
	Detail       uint32
	Root         xproto.Window
	Event        xproto.Window
	Child        xproto.Window
	RootX        XinputFp1616
	RootY        XinputFp1616
	EventX       XinputFp1616
	EventY       XinputFp1616
	Sourceid     XinputDeviceID
	Flags        uint32
	Mods         XinputModifierInfo
	Group        XinputGroupInfo
	ButtonMask   []uint32
	ValuatorMask []uint32
	Axisvalues   []XinputFp3232
}

// XinputButtonReleaseEvent is xinput::ButtonReleaseEvent.
type XinputButtonReleaseEvent XinputButtonPressEvent

// XinputMotionEvent is xinput::MotionEvent.
type XinputMotionEvent XinputButtonPressEvent

func decodeXinputButton(raw []byte) (XinputButtonPressEvent, error) {
	r := &reader{buf: raw}
	r.skip(1)
	v := XinputButtonPressEvent{Extension: r.u8(), Sequence: r.u16()}
	r.skip(4)
	v.EventType = r.u16()
	v.Deviceid = r.u16()
	v.Time = xproto.Timestamp(r.u32())
	v.Detail = r.u32()
	v.Root = xproto.Window(r.u32())
	v.Event = xproto.Window(r.u32())
	v.Child = xproto.Window(r.u32())
	v.RootX = int32(r.u32())
	v.RootY = int32(r.u32())
	v.EventX = int32(r.u32())
	v.EventY = int32(r.u32())
	buttonsLen := int(r.u16())
	valuatorsLen := int(r.u16())
	v.Sourceid = r.u16()
	r.skip(2)
	v.Flags = r.u32()
	v.Mods = readModifierInfo(r)
	v.Group = readGroupInfo(r)
	v.ButtonMask = r.u32s(buttonsLen)
	v.ValuatorMask = r.u32s(valuatorsLen)
	n := popcount(v.ValuatorMask)
	if r.err == nil && r.need(8*n) {
		v.Axisvalues = make([]XinputFp3232, n)
		for i := range v.Axisvalues {
			v.Axisvalues[i] = readFp3232(r)
		}
	}
	return v, r.err
}

func (v XinputButtonPressEvent) encode() []byte {
	var w writer
	geHeader(&w, v.Extension, v.Sequence, v.EventType)
	w.u16(v.Deviceid)
	w.u32(uint32(v.Time))
	w.u32(v.Detail)
	w.u32(uint32(v.Root))
	w.u32(uint32(v.Event))
	w.u32(uint32(v.Child))
	w.u32(uint32(v.RootX))
	w.u32(uint32(v.RootY))
	w.u32(uint32(v.EventX))
	w.u32(uint32(v.EventY))
	w.u16(uint16(len(v.ButtonMask)))
	w.u16(uint16(len(v.ValuatorMask)))
	w.u16(v.Sourceid)
	w.pad(2)
	w.u32(v.Flags)
	writeModifierInfo(&w, v.Mods)
	writeGroupInfo(&w, v.Group)
	w.u32s(v.ButtonMask)
	w.u32s(v.ValuatorMask)
	for _, a := range v.Axisvalues {
		writeFp3232(&w, a)
	}
	return geFinish(&w)
}

// Bytes encodes the event.
func (v XinputButtonPressEvent) Bytes() []byte { return v.encode() }

// SequenceId returns the event's sequence number.
func (v XinputButtonPressEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputButtonPressEvent) String() string {
	type plain XinputButtonPressEvent
	return fmt.Sprintf("XinputButtonPress %+v", plain(v))
}

// Bytes encodes the event.
func (v XinputButtonReleaseEvent) Bytes() []byte { return XinputButtonPressEvent(v).encode() }

// SequenceId returns the event's sequence number.
func (v XinputButtonReleaseEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputButtonReleaseEvent) String() string {
	type plain XinputButtonPressEvent
	return fmt.Sprintf("XinputButtonRelease %+v", plain(v))
}

// Bytes encodes the event.
func (v XinputMotionEvent) Bytes() []byte { return XinputButtonPressEvent(v).encode() }

// SequenceId returns the event's sequence number.
func (v XinputMotionEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputMotionEvent) String() string {
	type plain XinputButtonPressEvent
	return fmt.Sprintf("XinputMotion %+v", plain(v))
}

// XinputEnterEvent is xinput::EnterEvent; Leave shares its layout.
type XinputEnterEvent struct {
	Extension  byte
	Sequence   uint16
	EventType  uint16
	Deviceid   XinputDeviceID
	Time       xproto.Timestamp
	Sourceid   XinputDeviceID
	Mode       XinputNotifyMode
	Detail     XinputNotifyDetail
	Root       xproto.Window
	Event      xproto.Window
	Child      xproto.Window
	RootX      XinputFp1616
	RootY      XinputFp1616
	EventX     XinputFp1616
	EventY     XinputFp1616
	SameScreen bool
	Focus      bool
	Mods       XinputModifierInfo
	Group      XinputGroupInfo
	Buttons    []uint32
}

// XinputLeaveEvent is xinput::LeaveEvent.
type XinputLeaveEvent XinputEnterEvent

func decodeXinputEnter(raw []byte) (XinputEnterEvent, error) {
	r := &reader{buf: raw}
	r.skip(1)
	v := XinputEnterEvent{Extension: r.u8(), Sequence: r.u16()}
	r.skip(4)
	v.EventType = r.u16()
	v.Deviceid = r.u16()
	v.Time = xproto.Timestamp(r.u32())
	v.Sourceid = r.u16()
	v.Mode = XinputNotifyMode(r.u8())
	v.Detail = XinputNotifyDetail(r.u8())
	v.Root = xproto.Window(r.u32())
	v.Event = xproto.Window(r.u32())
	v.Child = xproto.Window(r.u32())
	v.RootX = int32(r.u32())
	v.RootY = int32(r.u32())
	v.EventX = int32(r.u32())
	v.EventY = int32(r.u32())
	v.SameScreen = r.u8() != 0
	v.Focus = r.u8() != 0
	n := int(r.u16())
	v.Mods = readModifierInfo(r)
	v.Group = readGroupInfo(r)
	v.Buttons = r.u32s(n)
	return v, r.err
}

func (v XinputEnterEvent) encode() []byte {
	var w writer
	geHeader(&w, v.Extension, v.Sequence, v.EventType)
	w.u16(v.Deviceid)
	w.u32(uint32(v.Time))
	w.u16(v.Sourceid)
	w.u8(uint8(v.Mode))
	w.u8(uint8(v.Detail))
	w.u32(uint32(v.Root))
	w.u32(uint32(v.Event))
	w.u32(uint32(v.Child))
	w.u32(uint32(v.RootX))
	w.u32(uint32(v.RootY))
	w.u32(uint32(v.EventX))
	w.u32(uint32(v.EventY))
	w.u8(boolByte(v.SameScreen))
	w.u8(boolByte(v.Focus))
	w.u16(uint16(len(v.Buttons)))
	writeModifierInfo(&w, v.Mods)
	writeGroupInfo(&w, v.Group)
	w.u32s(v.Buttons)
	return geFinish(&w)
}

// Bytes encodes the event.
func (v XinputEnterEvent) Bytes() []byte { return v.encode() }

// SequenceId returns the event's sequence number.
func (v XinputEnterEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputEnterEvent) String() string {
	type plain XinputEnterEvent
	return fmt.Sprintf("XinputEnter %+v", plain(v))
}

// Bytes encodes the event.
func (v XinputLeaveEvent) Bytes() []byte { return XinputEnterEvent(v).encode() }

// SequenceId returns the event's sequence number.
func (v XinputLeaveEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputLeaveEvent) String() string {
	type plain XinputEnterEvent
	return fmt.Sprintf("XinputLeave %+v", plain(v))
}

// XinputDeviceChangedEvent is xinput::DeviceChangedEvent.
type XinputDeviceChangedEvent struct {
	Extension byte
	Sequence  uint16
	EventType uint16
	Deviceid  XinputDeviceID
	Time      xproto.Timestamp
	Sourceid  XinputDeviceID
	Reason    uint8
	Classes   []XinputDeviceClass
}

func decodeXinputDeviceChanged(raw []byte) (XinputDeviceChangedEvent, error) {
	r := &reader{buf: raw}
	r.skip(1)
	v := XinputDeviceChangedEvent{Extension: r.u8(), Sequence: r.u16()}
	r.skip(4)
	v.EventType = r.u16()
	v.Deviceid = r.u16()
	v.Time = xproto.Timestamp(r.u32())
	n := int(r.u16())
	v.Sourceid = r.u16()
	v.Reason = r.u8()
	r.skip(11)
	for i := 0; i < n && r.err == nil; i++ {
		v.Classes = append(v.Classes, readDeviceClass(r))
	}
	return v, r.err
}

// Bytes encodes the event.
func (v XinputDeviceChangedEvent) Bytes() []byte {
	var w writer
	geHeader(&w, v.Extension, v.Sequence, v.EventType)
	w.u16(v.Deviceid)
	w.u32(uint32(v.Time))
	w.u16(uint16(len(v.Classes)))
	w.u16(v.Sourceid)
	w.u8(v.Reason)
	w.pad(11)
	for _, c := range v.Classes {
		writeDeviceClass(&w, c)
	}
	return geFinish(&w)
}

// SequenceId returns the event's sequence number.
func (v XinputDeviceChangedEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputDeviceChangedEvent) String() string {
	type plain XinputDeviceChangedEvent
	return fmt.Sprintf("XinputDeviceChanged %+v", plain(v))
}

// XinputHierarchyEvent is xinput::HierarchyEvent.
type XinputHierarchyEvent struct {
	Extension byte
	Sequence  uint16
	EventType uint16
	Deviceid  XinputDeviceID
	Time      xproto.Timestamp
	Flags     uint32
	Infos     []XinputHierarchyInfo
}

func decodeXinputHierarchy(raw []byte) (XinputHierarchyEvent, error) {
	r := &reader{buf: raw}
	r.skip(1)
	v := XinputHierarchyEvent{Extension: r.u8(), Sequence: r.u16()}
	r.skip(4)
	v.EventType = r.u16()
	v.Deviceid = r.u16()
	v.Time = xproto.Timestamp(r.u32())
	v.Flags = r.u32()
	n := int(r.u16())
	r.skip(10)
	for i := 0; i < n && r.err == nil; i++ {
		info := XinputHierarchyInfo{Deviceid: r.u16(), Attachment: r.u16(), Type: XinputDeviceType(r.u8()), Enabled: r.u8() != 0}
		r.skip(2)
		info.Flags = r.u32()
		v.Infos = append(v.Infos, info)
	}
	return v, r.err
}

// Bytes encodes the event.
func (v XinputHierarchyEvent) Bytes() []byte {
	var w writer
	geHeader(&w, v.Extension, v.Sequence, v.EventType)
	w.u16(v.Deviceid)
	w.u32(uint32(v.Time))
	w.u32(v.Flags)
	w.u16(uint16(len(v.Infos)))
	w.pad(10)
	for _, info := range v.Infos {
		w.u16(info.Deviceid)
		w.u16(info.Attachment)
		w.u8(uint8(info.Type))
		w.u8(boolByte(info.Enabled))
		w.pad(2)
		w.u32(info.Flags)
	}
	return geFinish(&w)
}

// SequenceId returns the event's sequence number.
func (v XinputHierarchyEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputHierarchyEvent) String() string {
	type plain XinputHierarchyEvent
	return fmt.Sprintf("XinputHierarchy %+v", plain(v))
}

// XinputGesturePinchBeginEvent is xinput::GesturePinchBeginEvent; Update and
// End share its layout.
type XinputGesturePinchBeginEvent struct {
	Extension     byte
	Sequence      uint16
	EventType     uint16
	Deviceid      XinputDeviceID
	Time          xproto.Timestamp
	Detail        uint32
	Root          xproto.Window
	Event         xproto.Window
	Child         xproto.Window
	RootX         XinputFp1616
	RootY         XinputFp1616
	EventX        XinputFp1616
	EventY        XinputFp1616
	DeltaX        XinputFp1616
	DeltaY        XinputFp1616
	DeltaUnaccelX XinputFp1616
	DeltaUnaccelY XinputFp1616
	Scale         XinputFp1616
	DeltaAngle    XinputFp1616
	Sourceid      XinputDeviceID
	Mods          XinputModifierInfo
	Group         XinputGroupInfo
	Flags         uint32
}

// XinputGesturePinchUpdateEvent is xinput::GesturePinchUpdateEvent.
type XinputGesturePinchUpdateEvent XinputGesturePinchBeginEvent

// XinputGesturePinchEndEvent is xinput::GesturePinchEndEvent.
type XinputGesturePinchEndEvent XinputGesturePinchBeginEvent

func decodeXinputGesturePinch(raw []byte) (XinputGesturePinchBeginEvent, error) {
	r := &reader{buf: raw}
	r.skip(1)
	v := XinputGesturePinchBeginEvent{Extension: r.u8(), Sequence: r.u16()}
	r.skip(4)
	v.EventType = r.u16()
	v.Deviceid = r.u16()
	v.Time = xproto.Timestamp(r.u32())
	v.Detail = r.u32()
	v.Root = xproto.Window(r.u32())
	v.Event = xproto.Window(r.u32())
	v.Child = xproto.Window(r.u32())
	for _, p := range []*int32{&v.RootX, &v.RootY, &v.EventX, &v.EventY, &v.DeltaX, &v.DeltaY, &v.DeltaUnaccelX, &v.DeltaUnaccelY, &v.Scale, &v.DeltaAngle} {
		*p = int32(r.u32())
	}
	v.Sourceid = r.u16()
	r.skip(2)
	v.Mods = readModifierInfo(r)
	v.Group = readGroupInfo(r)
	v.Flags = r.u32()
	return v, r.err
}

func (v XinputGesturePinchBeginEvent) encode() []byte {
	var w writer
	geHeader(&w, v.Extension, v.Sequence, v.EventType)
	w.u16(v.Deviceid)
	w.u32(uint32(v.Time))
	w.u32(v.Detail)
	w.u32(uint32(v.Root))
	w.u32(uint32(v.Event))
	w.u32(uint32(v.Child))
	for _, x := range []int32{v.RootX, v.RootY, v.EventX, v.EventY, v.DeltaX, v.DeltaY, v.DeltaUnaccelX, v.DeltaUnaccelY, v.Scale, v.DeltaAngle} {
		w.u32(uint32(x))
	}
	w.u16(v.Sourceid)
	w.pad(2)
	writeModifierInfo(&w, v.Mods)
	writeGroupInfo(&w, v.Group)
	w.u32(v.Flags)
	return geFinish(&w)
}

// Bytes encodes the event.
func (v XinputGesturePinchBeginEvent) Bytes() []byte { return v.encode() }

// SequenceId returns the event's sequence number.
func (v XinputGesturePinchBeginEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputGesturePinchBeginEvent) String() string {
	type plain XinputGesturePinchBeginEvent
	return fmt.Sprintf("XinputGesturePinchBegin %+v", plain(v))
}

// Bytes encodes the event.
func (v XinputGesturePinchUpdateEvent) Bytes() []byte {
	return XinputGesturePinchBeginEvent(v).encode()
}

// SequenceId returns the event's sequence number.
func (v XinputGesturePinchUpdateEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputGesturePinchUpdateEvent) String() string {
	type plain XinputGesturePinchBeginEvent
	return fmt.Sprintf("XinputGesturePinchUpdate %+v", plain(v))
}

// Bytes encodes the event.
func (v XinputGesturePinchEndEvent) Bytes() []byte { return XinputGesturePinchBeginEvent(v).encode() }

// SequenceId returns the event's sequence number.
func (v XinputGesturePinchEndEvent) SequenceId() uint16 { return v.Sequence }

// String implements fmt.Stringer.
func (v XinputGesturePinchEndEvent) String() string {
	type plain XinputGesturePinchBeginEvent
	return fmt.Sprintf("XinputGesturePinchEnd %+v", plain(v))
}

func decodeXinputEvent(evtype uint16, raw []byte) (xgb.Event, bool) {
	switch evtype {
	case XinputButtonPress, XinputButtonRelease, XinputMotion:
		v, err := decodeXinputButton(raw)
		if err != nil {
			return nil, false
		}
		switch evtype {
		case XinputButtonPress:
			return v, true
		case XinputButtonRelease:
			return XinputButtonReleaseEvent(v), true
		default:
			return XinputMotionEvent(v), true
		}
	case XinputEnter, XinputLeave:
		v, err := decodeXinputEnter(raw)
		if err != nil {
			return nil, false
		}
		if evtype == XinputEnter {
			return v, true
		}
		return XinputLeaveEvent(v), true
	case XinputDeviceChanged:
		v, err := decodeXinputDeviceChanged(raw)
		return v, err == nil
	case XinputHierarchy:
		v, err := decodeXinputHierarchy(raw)
		return v, err == nil
	case XinputGesturePinchBegin, XinputGesturePinchUpdate, XinputGesturePinchEnd:
		v, err := decodeXinputGesturePinch(raw)
		if err != nil {
			return nil, false
		}
		switch evtype {
		case XinputGesturePinchBegin:
			return v, true
		case XinputGesturePinchUpdate:
			return XinputGesturePinchUpdateEvent(v), true
		default:
			return XinputGesturePinchEndEvent(v), true
		}
	}
	return nil, false
}

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}
