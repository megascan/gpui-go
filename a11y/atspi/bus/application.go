package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const applicationInterfaceName = "org.a11y.atspi.Application"

type applicationInterface struct {
	root atspi.PlatformRoot
}

func (i *applicationInterface) toolkitName() (string, *dbus.Error) {
	value, err := i.root.ToolkitName()
	if err != nil {
		return "", mapRootError(err)
	}
	return value, nil
}

func (i *applicationInterface) version() (string, *dbus.Error) {
	value, err := i.root.ToolkitVersion()
	if err != nil {
		return "", mapRootError(err)
	}
	return value, nil
}

func (i *applicationInterface) atspiVersion() string {
	return "2.1"
}

func (i *applicationInterface) id() (int32, *dbus.Error) {
	value, err := i.root.ID()
	if err != nil {
		return 0, mapRootError(err)
	}
	return value, nil
}

func (i *applicationInterface) setID(id int32) *dbus.Error {
	if err := i.root.SetID(id); err != nil {
		return mapRootError(err)
	}
	return nil
}

func (i *applicationInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: applicationInterfaceName,
		properties: []dbusProperty{
			{name: "ToolkitName", signature: "s", get: getter(i.toolkitName)},
			{name: "Version", signature: "s", get: getter(i.version)},
			{name: "AtspiVersion", signature: "s", get: func() (any, *dbus.Error) { return i.atspiVersion(), nil }},
			{name: "Id", signature: "i", get: getter(i.id), set: func(value dbus.Variant) *dbus.Error {
				id, ok := value.Value().(int32)
				if !ok {
					return fdoInvalidArgs("Invalid value for property 'Id'")
				}
				return i.setID(id)
			}},
		},
	}
}
