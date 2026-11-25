//go:build linux

package dbus

import godbus "github.com/godbus/dbus/v5"

// NameOwner returns the unique connection name that currently owns the well-known name.
func NameOwner(conn *godbus.Conn, name string) (string, error) {
	var owner string
	err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, name).Store(&owner)
	return owner, err
}

// BusDriverName is the sender of signals emitted by the message bus itself.
const BusDriverName = "org.freedesktop.DBus"

// OwnerPin checks that a signal's sender is the current owner of a well-known name.
type OwnerPin struct {
	conn  *godbus.Conn
	name  string
	owner string
}

// NewOwnerPin returns a pin for name on conn, resolving the owner lazily.
func NewOwnerPin(conn *godbus.Conn, name string) *OwnerPin {
	return &OwnerPin{conn: conn, name: name}
}

// Matches reports whether sender is the owner of the pinned name, refreshing the owner once on a mismatch.
func (p *OwnerPin) Matches(sender string) bool {
	if sender == "" {
		return false
	}
	if p.owner != "" && sender == p.owner {
		return true
	}
	owner, err := NameOwner(p.conn, p.name)
	if err != nil {
		p.owner = ""
		return false
	}
	p.owner = owner
	return sender == owner
}
