//go:build linux

package xim

import "encoding/binary"

// AttributeName is the XIM attribute a wire id stands for.
type AttributeName uint8

const (
	AttrArea AttributeName = iota
	AttrAreaNeeded
	AttrBackground
	AttrBackgroundPixmap
	AttrClientWindow
	AttrColorMap
	AttrCursor
	AttrDestroyCallback
	AttrFilterEvents
	AttrFocusWindow
	AttrFontSet
	AttrForeground
	AttrGeometryCallback
	AttrHotKey
	AttrHotKeyState
	AttrInputStyle
	AttrLineSpace
	AttrNestedList
	AttrPreeditAttributes
	AttrPreeditCaretCallback
	AttrPreeditDoneCallback
	AttrPreeditDrawCallback
	AttrPreeditStartCallback
	AttrPreeditState
	AttrPreeditStateNotifyCallback
	AttrQueryICValuesList
	AttrQueryIMValuesList
	AttrQueryInputStyle
	AttrR6PreeditCallback
	AttrResetState
	AttrResourceClass
	AttrResourceName
	AttrSeparatorofNestedList
	AttrSpotLocation
	AttrStatusAttributes
	AttrStatusDoneCallback
	AttrStatusDrawCallback
	AttrStatusStartCallback
	AttrStdColorMap
	AttrStringConversion
	AttrStringConversionCallback
	AttrVisiblePosition
)

var attributeNameStrings = [...]string{
	AttrArea:                       "area",
	AttrAreaNeeded:                 "areaNeeded",
	AttrBackground:                 "background",
	AttrBackgroundPixmap:           "backgroundPixmap",
	AttrClientWindow:               "clientWindow",
	AttrColorMap:                   "colorMap",
	AttrCursor:                     "cursor",
	AttrDestroyCallback:            "destroyCallback",
	AttrFilterEvents:               "filterEvents",
	AttrFocusWindow:                "focusWindow",
	AttrFontSet:                    "fontSet",
	AttrForeground:                 "foreground",
	AttrGeometryCallback:           "geometryCallback",
	AttrHotKey:                     "hotKey",
	AttrHotKeyState:                "hotKeyState",
	AttrInputStyle:                 "inputStyle",
	AttrLineSpace:                  "lineSpace",
	AttrNestedList:                 "XNVaNestedList",
	AttrPreeditAttributes:          "preeditAttributes",
	AttrPreeditCaretCallback:       "preeditCaretCallback",
	AttrPreeditDoneCallback:        "preeditDoneCallback",
	AttrPreeditDrawCallback:        "preeditDrawCallback",
	AttrPreeditStartCallback:       "preeditStartCallback",
	AttrPreeditState:               "preeditState",
	AttrPreeditStateNotifyCallback: "preeditStateNotifyCallback",
	AttrQueryICValuesList:          "queryICValuesList",
	AttrQueryIMValuesList:          "queryIMValuesList",
	AttrQueryInputStyle:            "queryInputStyle",
	AttrR6PreeditCallback:          "r6PreeditCallback",
	AttrResetState:                 "resetState",
	AttrResourceClass:              "resourceClass",
	AttrResourceName:               "resourceName",
	AttrSeparatorofNestedList:      "separatorofNestedList",
	AttrSpotLocation:               "spotLocation",
	AttrStatusAttributes:           "statusAttributes",
	AttrStatusDoneCallback:         "statusDoneCallback",
	AttrStatusDrawCallback:         "statusDrawCallback",
	AttrStatusStartCallback:        "statusStartCallback",
	AttrStdColorMap:                "stdColorMap",
	AttrStringConversion:           "stringConversion",
	AttrStringConversionCallback:   "stringConversionCallback",
	AttrVisiblePosition:            "visiblePosition",
}

// String returns the attribute name as it appears on the wire.
func (n AttributeName) String() string {
	if int(n) >= len(attributeNameStrings) {
		return ""
	}
	return attributeNameStrings[n]
}

func attributeNameFromString(s string) (AttributeName, bool) {
	for i, name := range attributeNameStrings {
		if name == s {
			return AttributeName(i), true
		}
	}
	return 0, false
}

// AttrType is the value type a server declares for an attribute.
type AttrType uint16

const (
	TypeSeparator        AttrType = 0
	TypeByte             AttrType = 1
	TypeWord             AttrType = 2
	TypeLong             AttrType = 3
	TypeChar             AttrType = 4
	TypeWindow           AttrType = 5
	TypeStyle            AttrType = 10
	TypeXRectangle       AttrType = 11
	TypeXPoint           AttrType = 12
	TypeXFontSet         AttrType = 13
	TypeHotKeyTriggers   AttrType = 15
	TypeStringConversion AttrType = 17
	TypePreeditState     AttrType = 18
	TypeResetState       AttrType = 19
	TypeNestedList       AttrType = 32767
)

func attrTypeFromWire(v uint16) (AttrType, bool) {
	switch AttrType(v) {
	case TypeSeparator, TypeByte, TypeWord, TypeLong, TypeChar, TypeWindow, TypeStyle,
		TypeXRectangle, TypeXPoint, TypeXFontSet, TypeHotKeyTriggers, TypeStringConversion,
		TypePreeditState, TypeResetState, TypeNestedList:
		return AttrType(v), true
	}
	return 0, false
}

// Attr declares one attribute of an IM or IC: its wire id, value type and name.
type Attr struct {
	ID   uint16
	Type AttrType
	Name AttributeName
}

// Attribute is one id-tagged value sent in a Set or Get request.
type Attribute struct {
	ID    uint16
	Value []byte
}

// Point is the XPoint value of AttrSpotLocation.
type Point struct {
	X int16
	Y int16
}

// Rectangle is the XRectangle value of the area attributes.
type Rectangle struct {
	X      int16
	Y      int16
	Width  uint16
	Height uint16
}

// Extension names one protocol extension a server offers.
type Extension struct {
	Major uint8
	Minor uint8
	Name  string
}

// AttributeBuilder collects the attributes of one Set or Create request.
type AttributeBuilder struct {
	ids map[AttributeName]uint16
	out []Attribute
}

// NewAttributeBuilder returns a builder that keeps only the names present in ids.
func NewAttributeBuilder(ids map[AttributeName]uint16) *AttributeBuilder {
	return &AttributeBuilder{ids: ids}
}

// Long pushes a 32-bit value such as an InputStyle or a window id.
func (b *AttributeBuilder) Long(name AttributeName, v uint32) *AttributeBuilder {
	if id, ok := b.ids[name]; ok {
		b.out = append(b.out, Attribute{ID: id, Value: binary.LittleEndian.AppendUint32(nil, v)})
	}
	return b
}

// Point pushes an XPoint value.
func (b *AttributeBuilder) Point(name AttributeName, p Point) *AttributeBuilder {
	if id, ok := b.ids[name]; ok {
		value := binary.LittleEndian.AppendUint16(nil, uint16(p.X))
		value = binary.LittleEndian.AppendUint16(value, uint16(p.Y))
		b.out = append(b.out, Attribute{ID: id, Value: value})
	}
	return b
}

// Nested pushes a nested attribute list built by fill, skipped when name is unknown.
func (b *AttributeBuilder) Nested(name AttributeName, fill func(*AttributeBuilder)) *AttributeBuilder {
	id, ok := b.ids[name]
	if !ok {
		return b
	}
	inner := &AttributeBuilder{ids: b.ids}
	fill(inner)
	var value []byte
	for _, attr := range inner.out {
		value = appendAttribute(value, attr)
	}
	b.out = append(b.out, Attribute{ID: id, Value: value})
	return b
}

// Build returns the attributes collected so far.
func (b *AttributeBuilder) Build() []Attribute {
	return b.out
}

func appendAttribute(dst []byte, attr Attribute) []byte {
	dst = binary.LittleEndian.AppendUint16(dst, attr.ID)
	dst = binary.LittleEndian.AppendUint16(dst, uint16(len(attr.Value)))
	dst = append(dst, attr.Value...)
	for len(dst)%4 != 0 {
		dst = append(dst, 0)
	}
	return dst
}
