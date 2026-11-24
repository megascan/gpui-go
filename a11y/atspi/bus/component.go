package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const componentInterfaceName = "org.a11y.atspi.Component"

type componentInterface struct {
	busName string
	node    atspi.PlatformNode
}

func newComponentInterface(busName string, node atspi.PlatformNode) *componentInterface {
	return &componentInterface{busName: busName, node: node}
}

func (i *componentInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *componentInterface) contains(x, y int32, coordType uint32) (bool, *dbus.Error) {
	value, err := i.node.Contains(x, y, atspi.CoordType(coordType))
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) getAccessibleAtPoint(x, y int32, coordType uint32) (ownedObjectAddress, *dbus.Error) {
	node, ok, err := i.node.AccessibleAtPoint(x, y, atspi.CoordType(coordType))
	if err != nil {
		return ownedObjectAddress{}, i.mapError(err)
	}
	return optionalObjectAddress(i.busName, objectIDNode(i.node.AdapterID(), node), ok), nil
}

func (i *componentInterface) getExtents(coordType uint32) (atspi.Rect, *dbus.Error) {
	value, err := i.node.Extents(atspi.CoordType(coordType))
	if err != nil {
		return atspi.Rect{}, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) getLayer() (atspi.Layer, *dbus.Error) {
	value, err := i.node.Layer()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) grabFocus() (bool, *dbus.Error) {
	value, err := i.node.GrabFocus()
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) scrollTo(scrollType uint32) (bool, *dbus.Error) {
	value, err := i.node.ScrollTo(atspi.ScrollType(scrollType))
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) scrollToPoint(coordType uint32, x, y int32) (bool, *dbus.Error) {
	value, err := i.node.ScrollToPoint(atspi.CoordType(coordType), x, y)
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *componentInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: componentInterfaceName,
		methods: []dbusMethod{
			{name: "Contains", in: in("x", "i", "y", "i", "coord_type", "u"), out: out("b"), fn: i.contains},
			{name: "GetAccessibleAtPoint", in: in("x", "i", "y", "i", "coord_type", "u"), out: out("(so)"), fn: i.getAccessibleAtPoint},
			{name: "GetExtents", in: in("coord_type", "u"), out: out("(iiii)"), fn: i.getExtents},
			{name: "GetLayer", out: out("u"), fn: i.getLayer},
			{name: "GrabFocus", out: out("b"), fn: i.grabFocus},
			{name: "ScrollTo", in: in("scroll_type", "u"), out: out("b"), fn: i.scrollTo},
			{name: "ScrollToPoint", in: in("coord_type", "u", "x", "i", "y", "i"), out: out("b"), fn: i.scrollToPoint},
		},
	}
}
