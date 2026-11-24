//go:build linux

package wayland

import (
	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

const (
	zxdgExporterV1Version uint32 = 1
	zxdgExporterV2Version uint32 = 1
)

var (
	zxdgExporterV1Interface = &wl.Interface{Name: "zxdg_exporter_v1", Version: 1}
	zxdgExportedV1Interface = &wl.Interface{Name: "zxdg_exported_v1", Version: 1}
	zxdgExporterV2Interface = &wl.Interface{Name: "zxdg_exporter_v2", Version: 1}
	zxdgExportedV2Interface = &wl.Interface{Name: "zxdg_exported_v2", Version: 1}
	_                       = defineXdgForeign()
)

func defineXdgForeign() bool {
	zxdgExporterV1Interface.New = func() wl.Object { return &zxdgExporter{} }
	zxdgExporterV1Interface.Methods = []wl.Message{
		{Name: "destroy", Signature: "", Since: 1, Destructor: true},
		{Name: "export", Signature: "no", Since: 1, Types: []*wl.Interface{zxdgExportedV1Interface, wlproto.WlSurfaceInterface}},
	}
	zxdgExportedV1Interface.New = func() wl.Object { return &zxdgExported{} }
	zxdgExportedV1Interface.Methods = []wl.Message{{Name: "destroy", Signature: "", Since: 1, Destructor: true}}
	zxdgExportedV1Interface.Events = []wl.Message{{Name: "handle", Signature: "s", Since: 1}}
	zxdgExporterV2Interface.New = func() wl.Object { return &zxdgExporter{} }
	zxdgExporterV2Interface.Methods = []wl.Message{
		{Name: "destroy", Signature: "", Since: 1, Destructor: true},
		{Name: "export_toplevel", Signature: "no", Since: 1, Types: []*wl.Interface{zxdgExportedV2Interface, wlproto.WlSurfaceInterface}},
	}
	zxdgExportedV2Interface.New = func() wl.Object { return &zxdgExported{} }
	zxdgExportedV2Interface.Methods = []wl.Message{{Name: "destroy", Signature: "", Since: 1, Destructor: true}}
	zxdgExportedV2Interface.Events = []wl.Message{{Name: "handle", Signature: "s", Since: 1}}
	return true
}

type zxdgExporter struct {
	wl.Proxy
}

func (p *zxdgExporter) Destroy() { p.Marshal(0) }

func (p *zxdgExporter) export(surface *wlproto.WlSurface) *zxdgExported {
	child := zxdgExportedV2Interface
	if p.Interface() == zxdgExporterV1Interface {
		child = zxdgExportedV1Interface
	}
	obj, _ := p.MarshalConstructor(1, child, 0, wl.NewID(), wl.ObjectArg(surface)).(*zxdgExported)
	return obj
}

func (p *zxdgExporter) Dispatch(uint32, wl.Args) {}

type zxdgExported struct {
	wl.Proxy
	onHandle func(handle string)
}

func (p *zxdgExported) Destroy() { p.Marshal(0) }

func (p *zxdgExported) Dispatch(opcode uint32, a wl.Args) {
	if opcode == 0 && p.onHandle != nil {
		p.onHandle(a.String(0))
	}
}

type waylandWindowIdentifier struct {
	exported *zxdgExported
	handle   string
}

func waylandExportHandle(display *wl.Display, surface *wlproto.WlSurface) (*waylandWindowIdentifier, bool) {
	queue := display.CreateQueue()
	defer queue.Destroy()
	registry := wlproto.GetDisplay(display).GetRegistry()
	registry.SetQueue(queue)
	defer registry.Destroy()

	var exporter *zxdgExporter
	exporterIsV2 := false
	registry.SetListener(wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			switch iface {
			case "zxdg_exporter_v1":
				bound, _ := registry.Bind(name, zxdgExporterV1Interface, min(version, zxdgExporterV1Version)).(*zxdgExporter)
				if exporterIsV2 {
					bound.Proxy.Destroy()
				} else {
					if exporter != nil {
						exporter.Proxy.Destroy()
					}
					exporter = bound
				}
			case "zxdg_exporter_v2":
				bound, _ := registry.Bind(name, zxdgExporterV2Interface, min(version, zxdgExporterV2Version)).(*zxdgExporter)
				if exporter != nil {
					exporter.Proxy.Destroy()
				}
				exporter = bound
				exporterIsV2 = true
			}
		},
	})
	if _, err := display.RoundtripQueue(queue); err != nil || exporter == nil {
		return nil, false
	}
	exported := exporter.export(surface)
	identifier := &waylandWindowIdentifier{exported: exported}
	exported.onHandle = func(handle string) { identifier.handle = handle }
	if _, err := display.RoundtripQueue(queue); err != nil {
		exported.Destroy()
		exporter.Destroy()
		return nil, false
	}
	exporter.Destroy()
	exported.SetQueue(nil)
	return identifier, true
}

func (w *waylandWindowIdentifier) String() string { return "wayland:" + w.handle }

func (w *waylandWindowIdentifier) Destroy() { w.exported.Destroy() }
