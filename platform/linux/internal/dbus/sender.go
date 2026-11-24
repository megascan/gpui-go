//go:build linux

package dbus

import godbus "github.com/godbus/dbus/v5"

// NameOwner returns the unique connection name that currently owns the well-known name.
func NameOwner(conn *godbus.Conn, name string) (string, error) {
	var owner string
	err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, name).Store(&owner)
	return owner, err
}
