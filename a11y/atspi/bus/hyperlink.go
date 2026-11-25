package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const hyperlinkInterfaceName = "org.a11y.atspi.Hyperlink"

type hyperlinkInterface struct {
	busName string
	node    atspi.PlatformNode
}

func newHyperlinkInterface(busName string, node atspi.PlatformNode) *hyperlinkInterface {
	return &hyperlinkInterface{busName: busName, node: node}
}

func (i *hyperlinkInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *hyperlinkInterface) nAnchors() (int32, *dbus.Error) {
	value, err := i.node.NAnchors()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *hyperlinkInterface) startIndex() (int32, *dbus.Error) {
	value, err := i.node.HyperlinkStartIndex()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *hyperlinkInterface) endIndex() (int32, *dbus.Error) {
	value, err := i.node.HyperlinkEndIndex()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *hyperlinkInterface) getObject(index int32) (ownedObjectAddress, *dbus.Error) {
	node, ok, err := i.node.HyperlinkObject(index)
	if err != nil {
		return ownedObjectAddress{}, i.mapError(err)
	}
	return optionalObjectAddress(i.busName, objectIDNode(i.node.AdapterID(), node), ok), nil
}

func (i *hyperlinkInterface) getURI(index int32) (string, *dbus.Error) {
	value, err := i.node.URI(index)
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *hyperlinkInterface) isValid() (bool, *dbus.Error) {
	value, err := i.node.HyperlinkIsValid()
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *hyperlinkInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: hyperlinkInterfaceName,
		methods: []dbusMethod{
			{name: "GetObject", in: in("index", "i"), out: out("(so)"), fn: i.getObject},
			{name: "GetURI", in: in("index", "i"), out: out("s"), fn: i.getURI},
			{name: "IsValid", out: out("b"), fn: i.isValid},
		},
		properties: []dbusProperty{
			{name: "NAnchors", signature: "i", get: getter(i.nAnchors)},
			{name: "StartIndex", signature: "i", get: getter(i.startIndex)},
			{name: "EndIndex", signature: "i", get: getter(i.endIndex)},
		},
	}
}
