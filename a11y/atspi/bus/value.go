package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const valueInterfaceName = "org.a11y.atspi.Value"

type valueInterface struct {
	node atspi.PlatformNode
}

func newValueInterface(node atspi.PlatformNode) *valueInterface {
	return &valueInterface{node: node}
}

func (i *valueInterface) float(f func() (float64, error)) func() (any, *dbus.Error) {
	return func() (any, *dbus.Error) {
		value, err := f()
		if err != nil {
			return nil, mapErrorFromNode(i.node, err)
		}
		return value, nil
	}
}

func (i *valueInterface) minimumValue() (any, *dbus.Error) {
	return i.float(i.node.MinimumValue)()
}

func (i *valueInterface) maximumValue() (any, *dbus.Error) {
	return i.float(i.node.MaximumValue)()
}

func (i *valueInterface) minimumIncrement() (any, *dbus.Error) {
	return i.float(i.node.MinimumIncrement)()
}

func (i *valueInterface) currentValue() (any, *dbus.Error) {
	return i.float(i.node.CurrentValue)()
}

func (i *valueInterface) setCurrentValue(value float64) *dbus.Error {
	if err := i.node.SetCurrentValue(value); err != nil {
		return mapErrorFromNode(i.node, err)
	}
	return nil
}

func (i *valueInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: valueInterfaceName,
		properties: []dbusProperty{
			{name: "MinimumValue", signature: "d", get: i.minimumValue},
			{name: "MaximumValue", signature: "d", get: i.maximumValue},
			{name: "MinimumIncrement", signature: "d", get: i.minimumIncrement},
			{name: "CurrentValue", signature: "d", get: i.currentValue, set: func(value dbus.Variant) *dbus.Error {
				v, ok := value.Value().(float64)
				if !ok {
					return fdoInvalidArgs("Invalid value for property 'CurrentValue'")
				}
				return i.setCurrentValue(v)
			}},
		},
	}
}
