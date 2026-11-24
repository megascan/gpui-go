package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const selectionInterfaceName = "org.a11y.atspi.Selection"

type selectionInterface struct {
	busName string
	node    atspi.PlatformNode
}

func newSelectionInterface(busName string, node atspi.PlatformNode) *selectionInterface {
	return &selectionInterface{busName: busName, node: node}
}

func (i *selectionInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *selectionInterface) nSelectedChildren() (int32, *dbus.Error) {
	value, err := i.node.NSelectedChildren()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *selectionInterface) getSelectedChild(selectedChildIndex int32) (ownedObjectAddress, *dbus.Error) {
	index, derr := mapChildIndex(selectedChildIndex)
	if derr != nil {
		return ownedObjectAddress{}, derr
	}
	child, ok, err := i.node.SelectedChild(index)
	if err != nil {
		return ownedObjectAddress{}, i.mapError(err)
	}
	return optionalObjectAddress(i.busName, objectIDNode(i.node.AdapterID(), child), ok), nil
}

func (i *selectionInterface) withIndex(index int32, f func(int) (bool, error)) (bool, *dbus.Error) {
	n, derr := mapChildIndex(index)
	if derr != nil {
		return false, derr
	}
	value, err := f(n)
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *selectionInterface) selectChild(childIndex int32) (bool, *dbus.Error) {
	return i.withIndex(childIndex, i.node.SelectChild)
}

func (i *selectionInterface) deselectSelectedChild(selectedChildIndex int32) (bool, *dbus.Error) {
	return i.withIndex(selectedChildIndex, i.node.DeselectSelectedChild)
}

func (i *selectionInterface) isChildSelected(childIndex int32) (bool, *dbus.Error) {
	return i.withIndex(childIndex, i.node.IsChildSelected)
}

func (i *selectionInterface) selectAll() (bool, *dbus.Error) {
	value, err := i.node.SelectAll()
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *selectionInterface) clearSelection() (bool, *dbus.Error) {
	value, err := i.node.ClearSelection()
	if err != nil {
		return false, i.mapError(err)
	}
	return value, nil
}

func (i *selectionInterface) deselectChild(childIndex int32) (bool, *dbus.Error) {
	return i.withIndex(childIndex, i.node.DeselectChild)
}

func mapChildIndex(index int32) (int, *dbus.Error) {
	return childIndex(index)
}

func (i *selectionInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: selectionInterfaceName,
		methods: []dbusMethod{
			{name: "GetSelectedChild", in: in("selected_child_index", "i"), out: out("(so)"), fn: i.getSelectedChild},
			{name: "SelectChild", in: in("child_index", "i"), out: out("b"), fn: i.selectChild},
			{name: "DeselectSelectedChild", in: in("selected_child_index", "i"), out: out("b"), fn: i.deselectSelectedChild},
			{name: "IsChildSelected", in: in("child_index", "i"), out: out("b"), fn: i.isChildSelected},
			{name: "SelectAll", out: out("b"), fn: i.selectAll},
			{name: "ClearSelection", out: out("b"), fn: i.clearSelection},
			{name: "DeselectChild", in: in("child_index", "i"), out: out("b"), fn: i.deselectChild},
		},
		properties: []dbusProperty{
			{name: "NSelectedChildren", signature: "i", get: getter(i.nSelectedChildren)},
		},
	}
}
