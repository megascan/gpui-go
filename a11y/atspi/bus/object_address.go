package bus

import "github.com/godbus/dbus/v5"

const nullPath = "/org/a11y/atspi/null"

type ownedObjectAddress struct {
	BusName string
	Path    dbus.ObjectPath
}

func newOwnedObjectAddress(busName string, path dbus.ObjectPath) ownedObjectAddress {
	return ownedObjectAddress{BusName: busName, Path: path}
}

func ownedObjectAddressNull() ownedObjectAddress {
	return ownedObjectAddress{BusName: "", Path: nullPath}
}
