package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const textInterfaceName = "org.a11y.atspi.Text"

type textInterface struct {
	node atspi.PlatformNode
}

func newTextInterface(node atspi.PlatformNode) *textInterface {
	return &textInterface{node: node}
}

func (i *textInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *textInterface) characterCount() (int32, *dbus.Error) {
	value, err := i.node.CharacterCount()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) caretOffset() (int32, *dbus.Error) {
	value, err := i.node.CaretOffset()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) getStringAtOffset(offset int32, granularity uint32) (string, int32, int32, *dbus.Error) {
	text, start, end, err := i.node.StringAtOffset(offset, atspi.Granularity(granularity))
	if err != nil {
		return "", 0, 0, i.mapError(err)
	}
	return text, start, end, nil
}

func (i *textInterface) getText(startOffset, endOffset int32) (string, *dbus.Error) {
	value, err := i.node.Text(startOffset, endOffset)
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) boolResult(value bool, err error) (bool, *dbus.Error) {
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) setCaretOffset(offset int32) (bool, *dbus.Error) {
	return i.boolResult(i.node.SetCaretOffset(offset))
}

func (i *textInterface) getAttributeValue(offset int32, attributeName string) (string, *dbus.Error) {
	value, err := i.node.TextAttributeValue(offset, attributeName)
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) getAttributes(offset int32) (map[string]string, int32, int32, *dbus.Error) {
	attributes, start, end, err := i.node.TextAttributes(offset)
	if err != nil {
		return nil, 0, 0, i.mapError(err)
	}
	return nonNilMap(attributes), start, end, nil
}

func (i *textInterface) getDefaultAttributes() (map[string]string, *dbus.Error) {
	value, err := i.node.DefaultTextAttributes()
	if err != nil {
		return nil, i.mapError(err)
	}
	return nonNilMap(value), nil
}

func (i *textInterface) getCharacterExtents(offset int32, coordType uint32) (int32, int32, int32, int32, *dbus.Error) {
	rect, err := i.node.CharacterExtents(offset, atspi.CoordType(coordType))
	if err != nil {
		return 0, 0, 0, 0, i.mapError(err)
	}
	return rect.X, rect.Y, rect.Width, rect.Height, nil
}

func (i *textInterface) getOffsetAtPoint(x, y int32, coordType uint32) (int32, *dbus.Error) {
	value, err := i.node.OffsetAtPoint(x, y, atspi.CoordType(coordType))
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) getNSelections() (int32, *dbus.Error) {
	value, err := i.node.NSelections()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *textInterface) getSelection(selectionNum int32) (int32, int32, *dbus.Error) {
	start, end, err := i.node.Selection(selectionNum)
	if err != nil {
		return 0, 0, i.mapError(err)
	}
	return start, end, nil
}

func (i *textInterface) addSelection(startOffset, endOffset int32) (bool, *dbus.Error) {
	return i.boolResult(i.node.AddSelection(startOffset, endOffset))
}

func (i *textInterface) removeSelection(selectionNum int32) (bool, *dbus.Error) {
	return i.boolResult(i.node.RemoveSelection(selectionNum))
}

func (i *textInterface) setSelection(selectionNum, startOffset, endOffset int32) (bool, *dbus.Error) {
	return i.boolResult(i.node.SetSelection(selectionNum, startOffset, endOffset))
}

func (i *textInterface) getRangeExtents(startOffset, endOffset int32, coordType uint32) (int32, int32, int32, int32, *dbus.Error) {
	rect, err := i.node.RangeExtents(startOffset, endOffset, atspi.CoordType(coordType))
	if err != nil {
		return 0, 0, 0, 0, i.mapError(err)
	}
	return rect.X, rect.Y, rect.Width, rect.Height, nil
}

func (i *textInterface) getAttributeRun(offset int32, includeDefaults bool) (map[string]string, int32, int32, *dbus.Error) {
	attributes, start, end, err := i.node.TextAttributeRun(offset, includeDefaults)
	if err != nil {
		return nil, 0, 0, i.mapError(err)
	}
	return nonNilMap(attributes), start, end, nil
}

func (i *textInterface) scrollSubstringTo(startOffset, endOffset int32, scrollType uint32) (bool, *dbus.Error) {
	return i.boolResult(i.node.ScrollSubstringTo(startOffset, endOffset, atspi.ScrollType(scrollType)))
}

func (i *textInterface) scrollSubstringToPoint(startOffset, endOffset int32, coordType uint32, x, y int32) (bool, *dbus.Error) {
	return i.boolResult(i.node.ScrollSubstringToPoint(startOffset, endOffset, atspi.CoordType(coordType), x, y))
}

func (i *textInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: textInterfaceName,
		methods: []dbusMethod{
			{name: "GetStringAtOffset", in: in("offset", "i", "granularity", "u"), out: out("s", "i", "i"), fn: i.getStringAtOffset},
			{name: "GetText", in: in("start_offset", "i", "end_offset", "i"), out: out("s"), fn: i.getText},
			{name: "SetCaretOffset", in: in("offset", "i"), out: out("b"), fn: i.setCaretOffset},
			{name: "GetAttributeValue", in: in("offset", "i", "attribute_name", "s"), out: out("s"), fn: i.getAttributeValue},
			{name: "GetAttributes", in: in("offset", "i"), out: out("a{ss}", "i", "i"), fn: i.getAttributes},
			{name: "GetDefaultAttributes", out: out("a{ss}"), fn: i.getDefaultAttributes},
			{name: "GetCharacterExtents", in: in("offset", "i", "coord_type", "u"), out: out("i", "i", "i", "i"), fn: i.getCharacterExtents},
			{name: "GetOffsetAtPoint", in: in("x", "i", "y", "i", "coord_type", "u"), out: out("i"), fn: i.getOffsetAtPoint},
			{name: "GetNSelections", out: out("i"), fn: i.getNSelections},
			{name: "GetSelection", in: in("selection_num", "i"), out: out("i", "i"), fn: i.getSelection},
			{name: "AddSelection", in: in("start_offset", "i", "end_offset", "i"), out: out("b"), fn: i.addSelection},
			{name: "RemoveSelection", in: in("selection_num", "i"), out: out("b"), fn: i.removeSelection},
			{name: "SetSelection", in: in("selection_num", "i", "start_offset", "i", "end_offset", "i"), out: out("b"), fn: i.setSelection},
			{name: "GetRangeExtents", in: in("start_offset", "i", "end_offset", "i", "coord_type", "u"), out: out("i", "i", "i", "i"), fn: i.getRangeExtents},
			{name: "GetAttributeRun", in: in("offset", "i", "include_defaults", "b"), out: out("a{ss}", "i", "i"), fn: i.getAttributeRun},
			{name: "ScrollSubstringTo", in: in("start_offset", "i", "end_offset", "i", "scroll_type", "u"), out: out("b"), fn: i.scrollSubstringTo},
			{name: "ScrollSubstringToPoint", in: in("start_offset", "i", "end_offset", "i", "coord_type", "u", "x", "i", "y", "i"), out: out("b"), fn: i.scrollSubstringToPoint},
		},
		properties: []dbusProperty{
			{name: "CharacterCount", signature: "i", get: getter(i.characterCount)},
			{name: "CaretOffset", signature: "i", get: getter(i.caretOffset)},
		},
	}
}
