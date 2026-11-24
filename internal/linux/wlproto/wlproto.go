//go:build linux

// Package wlproto holds the typed Wayland protocol proxies wlgen generates from the
// vendored XML in xml/.
package wlproto

import "github.com/megascan/gpui-go/internal/linux/wl"

//go:generate go run github.com/megascan/gpui-go/internal/cmd/wlgen -o . xml

// GetDisplay returns the wl_display proxy of d.
func GetDisplay(d *wl.Display) *WlDisplay {
	return d.Object(WlDisplayInterface).(*WlDisplay)
}

// Bind binds global name of the registry as T at version.
func Bind[T any, PT interface {
	*T
	wl.Object
}](r *WlRegistry, name uint32, iface *wl.Interface, version uint32) PT {
	obj, _ := r.Bind(name, iface, version).(PT)
	return obj
}
