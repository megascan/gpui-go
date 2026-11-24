package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const actionInterfaceName = "org.a11y.atspi.Action"

type actionInterface struct {
	node atspi.PlatformNode
}

func newActionInterface(node atspi.PlatformNode) *actionInterface {
	return &actionInterface{node: node}
}

func (i *actionInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *actionInterface) nActions() (int32, *dbus.Error) {
	value, err := i.node.NActions()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *actionInterface) getDescription(int32) (string, *dbus.Error) {
	return "", nil
}

func (i *actionInterface) getName(index int32) (string, *dbus.Error) {
	value, err := i.node.ActionName(index)
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *actionInterface) getLocalizedName(index int32) (string, *dbus.Error) {
	value, err := i.node.ActionName(index)
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *actionInterface) getKeyBinding(int32) (string, *dbus.Error) {
	return "", nil
}

func (i *actionInterface) getActions() ([]atspi.Action, *dbus.Error) {
	value, err := i.node.Actions()
	if err != nil {
		return nil, i.mapError(err)
	}
	return nonNil(value), nil
}

func (i *actionInterface) doAction(index int32) (bool, *dbus.Error) {
	value, err := i.node.DoAction(index)
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *actionInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: actionInterfaceName,
		methods: []dbusMethod{
			{name: "GetDescription", in: in("index", "i"), out: out("s"), fn: i.getDescription},
			{name: "GetName", in: in("index", "i"), out: out("s"), fn: i.getName},
			{name: "GetLocalizedName", in: in("index", "i"), out: out("s"), fn: i.getLocalizedName},
			{name: "GetKeyBinding", in: in("index", "i"), out: out("s"), fn: i.getKeyBinding},
			{name: "GetActions", out: out("a(sss)"), fn: i.getActions},
			{name: "DoAction", in: in("index", "i"), out: out("b"), fn: i.doAction},
		},
		properties: []dbusProperty{
			{name: "NActions", signature: "i", get: getter(i.nActions)},
		},
	}
}
