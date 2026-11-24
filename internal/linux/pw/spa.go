//go:build linux

package pw

import (
	"encoding/binary"
	"errors"
	"syscall"
	"unsafe"
)

// SPA pod types (enum spa_type).
const (
	SpaTypeNone      uint32 = 1
	SpaTypeID        uint32 = 3
	SpaTypeInt       uint32 = 4
	SpaTypeLong      uint32 = 5
	SpaTypeRectangle uint32 = 10
	SpaTypeFraction  uint32 = 11
	SpaTypeObject    uint32 = 15
	SpaTypeChoice    uint32 = 19
)

// SPA object types and parameter ids.
const (
	SpaTypeObjectFormat    uint32 = 0x40003
	SpaTypeObjectParamMeta uint32 = 0x40005

	SpaParamEnumFormat uint32 = 3
	SpaParamFormat     uint32 = 4
	SpaParamMeta       uint32 = 6

	SpaParamMetaType uint32 = 1
	SpaParamMetaSize uint32 = 2

	SpaMetaHeader uint32 = 1
)

// SPA choice types (enum spa_choice_type).
const (
	SpaChoiceNone  uint32 = 0
	SpaChoiceRange uint32 = 1
	SpaChoiceStep  uint32 = 2
	SpaChoiceEnum  uint32 = 3
	SpaChoiceFlags uint32 = 4
)

// Format property keys (enum spa_format).
const (
	SpaFormatMediaType               uint32 = 1
	SpaFormatMediaSubtype            uint32 = 2
	SpaFormatVideoFormat             uint32 = 0x20001
	SpaFormatVideoModifier           uint32 = 0x20002
	SpaFormatVideoSize               uint32 = 0x20003
	SpaFormatVideoFramerate          uint32 = 0x20004
	SpaFormatVideoMaxFramerate       uint32 = 0x20005
	SpaFormatVideoViews              uint32 = 0x20006
	SpaFormatVideoInterlaceMode      uint32 = 0x20007
	SpaFormatVideoPixelAspectRatio   uint32 = 0x20008
	SpaFormatVideoMultiviewMode      uint32 = 0x20009
	SpaFormatVideoMultiviewFlags     uint32 = 0x2000a
	SpaFormatVideoChromaSite         uint32 = 0x2000b
	SpaFormatVideoColorRange         uint32 = 0x2000c
	SpaFormatVideoColorMatrix        uint32 = 0x2000d
	SpaFormatVideoTransferFunction   uint32 = 0x2000e
	SpaFormatVideoColorPrimaries     uint32 = 0x2000f
	SpaMediaTypeVideo                uint32 = 2
	SpaMediaSubtypeRaw               uint32 = 1
	spaPodPropFlagDontFixate         uint32 = 1 << 4
	spaVideoFlagModifier             uint32 = 1 << 2
	spaVideoFlagModifierFixationReqd uint32 = 1 << 4
)

// VideoFormat is enum spa_video_format.
type VideoFormat uint32

// Video formats used by the scap engine.
const (
	VideoFormatUnknown VideoFormat = 0
	VideoFormatRGBx    VideoFormat = 7
	VideoFormatBGRx    VideoFormat = 8
	VideoFormatXBGR    VideoFormat = 10
	VideoFormatRGBA    VideoFormat = 11
	VideoFormatRGB     VideoFormat = 15
)

// Rectangle is struct spa_rectangle.
type Rectangle struct {
	Width  uint32
	Height uint32
}

// Fraction is struct spa_fraction.
type Fraction struct {
	Num   uint32
	Denom uint32
}

// SpaMetaHeaderData is struct spa_meta_header.
type SpaMetaHeaderData struct {
	Flags     uint32
	Offset    uint32
	Pts       int64
	DtsOffset int64
	Seq       uint64
}

// SpaMeta is struct spa_meta.
type SpaMeta struct {
	Type uint32
	Size uint32
	Data unsafe.Pointer
}

// SpaChunk is struct spa_chunk.
type SpaChunk struct {
	Offset uint32
	Size   uint32
	Stride int32
	Flags  int32
}

// SpaData is struct spa_data.
type SpaData struct {
	Type      uint32
	Flags     uint32
	FD        int64
	MapOffset uint32
	MaxSize   uint32
	Data      unsafe.Pointer
	Chunk     *SpaChunk
}

// SpaBuffer is struct spa_buffer.
type SpaBuffer struct {
	NMetas uint32
	NDatas uint32
	Metas  *SpaMeta
	Datas  *SpaData
}

// Buffer is struct pw_buffer.
type Buffer struct {
	Buffer    *SpaBuffer
	UserData  unsafe.Pointer
	Size      uint64
	Requested uint64
	Time      uint64
}

type spaDictItem struct {
	key   *byte
	value *byte
}

type spaDict struct {
	flags  uint32
	nItems uint32
	items  *spaDictItem
}

type spaHook struct {
	next, prev unsafe.Pointer
	funcs      unsafe.Pointer
	data       unsafe.Pointer
	removed    unsafe.Pointer
	priv       unsafe.Pointer
}

type pwStreamEvents struct {
	version      uint32
	destroy      uintptr
	stateChanged uintptr
	controlInfo  uintptr
	ioChanged    uintptr
	paramChanged uintptr
	addBuffer    uintptr
	removeBuffer uintptr
	process      uintptr
	drained      uintptr
	command      uintptr
	triggerDone  uintptr
}

const pwVersionStreamEvents = 2

// Value is a serializable SPA pod value (libspa::pod::Value): one of ID, Int, RectangleValue,
// FractionValue, Choice or Object.
type Value interface{ isValue() }

// ID is an Id pod.
type ID uint32

// Int is an Int pod.
type Int int32

// RectangleValue is a Rectangle pod.
type RectangleValue Rectangle

// FractionValue is a Fraction pod.
type FractionValue Fraction

// Choice is a Choice pod; Values holds the default first, then the alternatives or bounds.
type Choice struct {
	Type      uint32
	Flags     uint32
	ChildType uint32
	Values    []Value
}

// Property is one property of an Object pod.
type Property struct {
	Key   uint32
	Flags uint32
	Value Value
}

// Object is an Object pod.
type Object struct {
	Type       uint32
	ID         uint32
	Properties []Property
}

func (ID) isValue()             {}
func (Int) isValue()            {}
func (RectangleValue) isValue() {}
func (FractionValue) isValue()  {}
func (Choice) isValue()         {}
func (Object) isValue()         {}

// Serialize encodes v as an SPA pod (PodSerializer::serialize).
func Serialize(v Value) ([]byte, error) {
	var out []byte
	if err := appendPod(&out, v); err != nil {
		return nil, err
	}
	return out, nil
}

var errUnsupportedValue = errors.New("pw: unsupported pod value")

func le32(b []byte, v uint32) []byte { return binary.LittleEndian.AppendUint32(b, v) }

func pad8(b []byte) []byte {
	for len(b)%8 != 0 {
		b = append(b, 0)
	}
	return b
}

func valueBody(v Value) (uint32, []byte, error) {
	switch v := v.(type) {
	case ID:
		return SpaTypeID, le32(nil, uint32(v)), nil
	case Int:
		return SpaTypeInt, le32(nil, uint32(v)), nil
	case RectangleValue:
		return SpaTypeRectangle, le32(le32(nil, v.Width), v.Height), nil
	case FractionValue:
		return SpaTypeFraction, le32(le32(nil, v.Num), v.Denom), nil
	case Choice:
		body := le32(le32(nil, v.Type), v.Flags)
		var child []byte
		childSize := uint32(0)
		for i, item := range v.Values {
			typ, b, err := valueBody(item)
			if err != nil {
				return 0, nil, err
			}
			if typ != v.ChildType {
				return 0, nil, errUnsupportedValue
			}
			if i == 0 {
				childSize = uint32(len(b))
			}
			child = append(child, b...)
		}
		body = le32(le32(body, childSize), v.ChildType)
		return SpaTypeChoice, append(body, child...), nil
	case Object:
		body := le32(le32(nil, v.Type), v.ID)
		for _, p := range v.Properties {
			body = le32(le32(body, p.Key), p.Flags)
			if err := appendPod(&body, p.Value); err != nil {
				return 0, nil, err
			}
		}
		return SpaTypeObject, body, nil
	}
	return 0, nil, errUnsupportedValue
}

func appendPod(out *[]byte, v Value) error {
	typ, body, err := valueBody(v)
	if err != nil {
		return err
	}
	b := le32(le32(*out, uint32(len(body))), typ)
	*out = pad8(append(b, body...))
	return nil
}

// PodProp is a parsed object property: the key, flags and the raw value pod.
type PodProp struct {
	Key   uint32
	Flags uint32
	Type  uint32
	Body  []byte
}

// PodObject is a parsed Object pod.
type PodObject struct {
	Type  uint32
	ID    uint32
	Props []PodProp
}

var errProto = syscall.EPROTO

// ParseObject parses pod as an Object pod.
func ParseObject(pod []byte) (PodObject, error) {
	if len(pod) < 8 {
		return PodObject{}, syscall.EINVAL
	}
	size := binary.LittleEndian.Uint32(pod)
	typ := binary.LittleEndian.Uint32(pod[4:])
	if typ != SpaTypeObject || size < 8 || uint64(size)+8 > uint64(len(pod)) {
		return PodObject{}, errProto
	}
	body := pod[8 : 8+size]
	obj := PodObject{Type: binary.LittleEndian.Uint32(body), ID: binary.LittleEndian.Uint32(body[4:])}
	rest := body[8:]
	for len(rest) >= 16 {
		key := binary.LittleEndian.Uint32(rest)
		flags := binary.LittleEndian.Uint32(rest[4:])
		vsize := binary.LittleEndian.Uint32(rest[8:])
		vtype := binary.LittleEndian.Uint32(rest[12:])
		if uint64(vsize)+16 > uint64(len(rest)) {
			break
		}
		obj.Props = append(obj.Props, PodProp{Key: key, Flags: flags, Type: vtype, Body: rest[16 : 16+vsize]})
		n := (16 + int(vsize) + 7) &^ 7
		if n > len(rest) {
			break
		}
		rest = rest[n:]
	}
	return obj, nil
}

func (o PodObject) find(key uint32) (PodProp, bool) {
	for _, p := range o.Props {
		if p.Key == key {
			return p, true
		}
	}
	return PodProp{}, false
}

func collectable(p PodProp) (uint32, []byte) {
	if p.Type == SpaTypeChoice && len(p.Body) >= 16 && binary.LittleEndian.Uint32(p.Body) == SpaChoiceNone {
		childSize := binary.LittleEndian.Uint32(p.Body[8:])
		childType := binary.LittleEndian.Uint32(p.Body[12:])
		if uint64(childSize)+16 <= uint64(len(p.Body)) {
			return childType, p.Body[16 : 16+childSize]
		}
	}
	return p.Type, p.Body
}

type field struct {
	key      uint32
	typ      uint32
	optional bool
	set      func([]byte)
}

func parseObject(pod []byte, objType uint32, fields []field) (PodObject, error) {
	obj, err := ParseObject(pod)
	if err != nil {
		return obj, err
	}
	if obj.Type != objType {
		return obj, errProto
	}
	for _, f := range fields {
		p, ok := obj.find(f.key)
		if !ok {
			if f.optional {
				continue
			}
			return obj, syscall.ESRCH
		}
		typ, body := collectable(p)
		need := 4
		if typ == SpaTypeRectangle || typ == SpaTypeFraction || typ == SpaTypeLong {
			need = 8
		}
		if typ != f.typ || len(body) < need {
			if f.optional {
				continue
			}
			return obj, errProto
		}
		f.set(body)
	}
	return obj, nil
}

func u32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }

// FormatParse reads the media type and subtype of a Format pod (spa_format_parse).
func FormatParse(pod []byte) (mediaType, mediaSubtype uint32, err error) {
	_, err = parseObject(pod, SpaTypeObjectFormat, []field{
		{SpaFormatMediaType, SpaTypeID, false, func(b []byte) { mediaType = u32(b) }},
		{SpaFormatMediaSubtype, SpaTypeID, false, func(b []byte) { mediaSubtype = u32(b) }},
	})
	return mediaType, mediaSubtype, err
}

// VideoInfoRaw is struct spa_video_info_raw.
type VideoInfoRaw struct {
	Format           VideoFormat
	Flags            uint32
	Modifier         uint64
	Size             Rectangle
	Framerate        Fraction
	MaxFramerate     Fraction
	Views            uint32
	InterlaceMode    uint32
	PixelAspectRatio Fraction
	MultiviewMode    uint32
	MultiviewFlags   uint32
	ChromaSite       uint32
	ColorRange       uint32
	ColorMatrix      uint32
	TransferFunction uint32
	ColorPrimaries   uint32
}

// Parse fills info from a Format pod (spa_format_video_raw_parse).
func (info *VideoInfoRaw) Parse(pod []byte) error {
	info.Flags = 0
	if obj, err := ParseObject(pod); err == nil {
		if p, ok := obj.find(SpaFormatVideoModifier); ok {
			info.Flags |= spaVideoFlagModifier
			if p.Flags&spaPodPropFlagDontFixate == spaPodPropFlagDontFixate {
				info.Flags |= spaVideoFlagModifierFixationReqd
			}
		}
	}
	idInto := func(key uint32, dst *uint32) field {
		return field{key, SpaTypeID, true, func(b []byte) { *dst = u32(b) }}
	}
	fracInto := func(key uint32, dst *Fraction) field {
		return field{key, SpaTypeFraction, true, func(b []byte) { *dst = Fraction{u32(b), u32(b[4:])} }}
	}
	_, err := parseObject(pod, SpaTypeObjectFormat, []field{
		{SpaFormatVideoFormat, SpaTypeID, true, func(b []byte) { info.Format = VideoFormat(u32(b)) }},
		{SpaFormatVideoModifier, SpaTypeLong, true, func(b []byte) { info.Modifier = binary.LittleEndian.Uint64(b) }},
		{SpaFormatVideoSize, SpaTypeRectangle, true, func(b []byte) { info.Size = Rectangle{u32(b), u32(b[4:])} }},
		fracInto(SpaFormatVideoFramerate, &info.Framerate),
		fracInto(SpaFormatVideoMaxFramerate, &info.MaxFramerate),
		{SpaFormatVideoViews, SpaTypeInt, true, func(b []byte) { info.Views = u32(b) }},
		idInto(SpaFormatVideoInterlaceMode, &info.InterlaceMode),
		fracInto(SpaFormatVideoPixelAspectRatio, &info.PixelAspectRatio),
		idInto(SpaFormatVideoMultiviewMode, &info.MultiviewMode),
		idInto(SpaFormatVideoMultiviewFlags, &info.MultiviewFlags),
		idInto(SpaFormatVideoChromaSite, &info.ChromaSite),
		idInto(SpaFormatVideoColorRange, &info.ColorRange),
		idInto(SpaFormatVideoColorMatrix, &info.ColorMatrix),
		idInto(SpaFormatVideoTransferFunction, &info.TransferFunction),
		idInto(SpaFormatVideoColorPrimaries, &info.ColorPrimaries),
	})
	return err
}

// PodFromC copies the spa_pod at p, header included.
func PodFromC(p unsafe.Pointer) []byte {
	if p == nil {
		return nil
	}
	size := *(*uint32)(p)
	return append([]byte(nil), unsafe.Slice((*byte)(p), int(size)+8)...)
}
