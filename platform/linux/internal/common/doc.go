//go:build linux

// Package common ports the state and helpers that gpui_linux's Wayland, X11 and
// headless connections share: LinuxCommon, the dispatcher and its event loop,
// the headless connection and window, the keyboard layout and the xkb,
// cursor, pipe and portal helpers of platform.rs.
package common
