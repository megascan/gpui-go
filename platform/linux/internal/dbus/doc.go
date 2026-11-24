//go:build linux

// Package dbus ports the D-Bus services of gpui_linux: power profiles, system
// notifications, the XDG desktop portal settings and the Secret Service keyring.
// Every entry point takes the *dbus.Conn to use; callers own the connection.
package dbus
