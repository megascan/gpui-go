package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

func mapRootError(err error) *dbus.Error {
	return mapError(objectIDRoot(), err)
}

func optionalObjectAddress(busName string, id objectID, ok bool) ownedObjectAddress {
	if ok {
		return id.toAddress(busName)
	}
	return ownedObjectAddressNull()
}

func stateSetWire(states atspi.StateSet) []uint32 {
	bits := states.Bits()
	return []uint32{uint32(bits), uint32(bits >> 32)}
}

func interfaceSetWire(interfaces atspi.InterfaceSet) []string {
	names := []string{}
	for iface := range interfaces.Iter() {
		names = append(names, iface.Name())
	}
	return names
}

func childIndex(index int32) (int, *dbus.Error) {
	if index < 0 {
		return 0, fdoInvalidArgs("Index can't be negative.")
	}
	return int(index), nil
}
