//go:build linux

// Command wlspike opens an xdg_toplevel through internal/linux/wl, creates a wgpu surface
// from the raw wl_display and wl_surface pointers, clears it for a number of frames and
// exits. It is the M1-wayland-ffi spike.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/oliverbestmann/webgpu/wgpu"
	"golang.org/x/sys/unix"
)

func main() {
	frames := flag.Int("frames", 90, "number of frames to present before exiting")
	width := flag.Int("width", 640, "initial width when the compositor leaves it to the client")
	height := flag.Int("height", 480, "initial height when the compositor leaves it to the client")
	deadline := flag.Duration("deadline", 45*time.Second, "give up after this long")
	flag.Parse()
	if err := run(*frames, *width, *height, *deadline); err != nil {
		fmt.Fprintln(os.Stderr, "wlspike:", err)
		os.Exit(1)
	}
}

type window struct {
	display    *wl.Display
	registry   *wlproto.WlRegistry
	compositor *wlproto.WlCompositor
	wmBase     *wlproto.XdgWmBase
	decoration *wlproto.ZxdgDecorationManagerV1
	surface    *wlproto.WlSurface
	xdgSurface *wlproto.XdgSurface
	toplevel   *wlproto.XdgToplevel
	topDecor   *wlproto.ZxdgToplevelDecorationV1

	width, height int
	configured    bool
	resized       bool
	closed        bool
}

func asPointer(u uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&u))
}

func run(frames, width, height int, deadline time.Duration) error {
	d, err := wl.Connect("")
	if err != nil {
		return err
	}
	w := &window{display: d, width: width, height: height}
	defer w.close()
	if err := w.open(); err != nil {
		return err
	}

	instance := wgpu.CreateInstance(nil)
	defer instance.Release()
	surface := instance.CreateSurface(&wgpu.SurfaceDescriptor{
		Label: "wlspike",
		WaylandSurface: &wgpu.SurfaceSourceWaylandSurface{
			Display: asPointer(d.Ptr()),
			Surface: asPointer(w.surface.Ptr()),
		},
	})
	defer surface.Release()
	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{CompatibleSurface: surface})
	if err != nil {
		return fmt.Errorf("request adapter: %w", err)
	}
	defer adapter.Release()
	device, err := adapter.RequestDevice(nil)
	if err != nil {
		return fmt.Errorf("request device: %w", err)
	}
	defer device.Release()
	queue := device.GetQueue()
	defer queue.Release()

	caps := surface.GetCapabilities(adapter)
	if len(caps.Formats) == 0 || len(caps.AlphaModes) == 0 {
		return errors.New("surface reports no formats")
	}
	config := &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      caps.Formats[0],
		Width:       uint32(w.width),
		Height:      uint32(w.height),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
	}
	surface.Configure(device, config)

	stop := time.Now().Add(deadline)
	presented := 0
	for presented < frames && !w.closed {
		if time.Now().After(stop) {
			return fmt.Errorf("deadline passed after %d of %d frames", presented, frames)
		}
		if err := w.pump(); err != nil {
			return err
		}
		if w.resized {
			w.resized = false
			config.Width, config.Height = uint32(w.width), uint32(w.height)
			surface.Configure(device, config)
		}
		ok, err := clear(device, queue, surface, presented, frames)
		if err != nil {
			return err
		}
		if ok {
			presented++
		} else {
			time.Sleep(16 * time.Millisecond)
		}
	}
	fmt.Printf("wlspike: presented %d frames at %dx%d (%s)\n", presented, w.width, w.height, adapter.GetInfo().Description)
	return nil
}

func clear(device *wgpu.Device, queue *wgpu.Queue, surface *wgpu.Surface, frame, frames int) (bool, error) {
	st, err := surface.TryGetCurrentTexture()
	if err != nil {
		return false, err
	}
	texture, ok := st.Get()
	if !ok {
		return false, nil
	}
	defer texture.Release()
	view, err := texture.TryCreateView(nil)
	if err != nil {
		return false, err
	}
	defer view.Release()
	encoder, err := device.TryCreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "wlspike"})
	if err != nil {
		return false, err
	}
	defer encoder.Release()
	t := float64(frame) / float64(max(frames-1, 1))
	pass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View:       view,
			LoadOp:     wgpu.LoadOpClear,
			StoreOp:    wgpu.StoreOpStore,
			ClearValue: wgpu.Color{R: 0.1 + 0.6*t, G: 0.2, B: 0.7 - 0.5*t, A: 1},
		}},
	})
	pass.End()
	pass.Release()
	cmd, err := encoder.TryFinish(nil)
	if err != nil {
		return false, err
	}
	defer cmd.Release()
	queue.Submit(cmd)
	surface.Present()
	return true, nil
}

func (w *window) open() error {
	d := w.display
	w.registry = wlproto.GetDisplay(d).GetRegistry()
	type global struct {
		name    uint32
		version uint32
	}
	globals := map[string]global{}
	w.registry.SetListener(wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			globals[iface] = global{name, version}
		},
	})
	if _, err := d.Roundtrip(); err != nil {
		return fmt.Errorf("registry roundtrip: %w", err)
	}
	comp, ok := globals["wl_compositor"]
	if !ok {
		return errors.New("compositor lacks wl_compositor")
	}
	wm, ok := globals["xdg_wm_base"]
	if !ok {
		return errors.New("compositor lacks xdg_wm_base")
	}
	w.compositor = wlproto.Bind[wlproto.WlCompositor](w.registry, comp.name, wlproto.WlCompositorInterface, min(comp.version, 5))
	w.wmBase = wlproto.Bind[wlproto.XdgWmBase](w.registry, wm.name, wlproto.XdgWmBaseInterface, min(wm.version, 5))
	w.wmBase.SetListener(wlproto.XdgWmBaseListener{Ping: func(serial uint32) { w.wmBase.Pong(serial) }})
	if dec, ok := globals["zxdg_decoration_manager_v1"]; ok {
		w.decoration = wlproto.Bind[wlproto.ZxdgDecorationManagerV1](w.registry, dec.name, wlproto.ZxdgDecorationManagerV1Interface, 1)
	}

	w.surface = w.compositor.CreateSurface()
	w.xdgSurface = w.wmBase.GetXdgSurface(w.surface)
	w.xdgSurface.SetListener(wlproto.XdgSurfaceListener{Configure: func(serial uint32) {
		w.xdgSurface.AckConfigure(serial)
		w.configured = true
	}})
	w.toplevel = w.xdgSurface.GetToplevel()
	w.toplevel.SetListener(wlproto.XdgToplevelListener{
		Configure: func(width, height int32, states []byte) {
			if width > 0 && height > 0 && (int(width) != w.width || int(height) != w.height) {
				w.width, w.height = int(width), int(height)
				w.resized = true
			}
		},
		Close: func() { w.closed = true },
	})
	w.toplevel.SetTitle("wlspike")
	w.toplevel.SetAppId("dev.gpui-go.wlspike")
	if w.decoration != nil {
		w.topDecor = w.decoration.GetToplevelDecoration(w.toplevel)
		w.topDecor.SetMode(wlproto.ZxdgToplevelDecorationV1ModeServerSide)
	}
	w.surface.Commit()
	for !w.configured {
		if _, err := d.Dispatch(); err != nil {
			return fmt.Errorf("waiting for configure: %w", err)
		}
	}
	w.resized = false
	return nil
}

func (w *window) pump() error {
	d := w.display
	for !d.PrepareRead() {
		if _, err := d.DispatchPending(); err != nil {
			return err
		}
	}
	if err := d.Flush(); err != nil && !errors.Is(err, unix.EAGAIN) {
		d.CancelRead()
		return err
	}
	fds := []unix.PollFd{{Fd: int32(d.FD()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	if err != nil && !errors.Is(err, unix.EINTR) {
		d.CancelRead()
		return fmt.Errorf("poll: %w", err)
	}
	if n > 0 {
		if err := d.ReadEvents(); err != nil {
			return err
		}
	} else {
		d.CancelRead()
	}
	_, err = d.DispatchPending()
	return err
}

func (w *window) close() {
	if w.topDecor != nil {
		w.topDecor.Destroy()
	}
	if w.toplevel != nil {
		w.toplevel.Destroy()
	}
	if w.xdgSurface != nil {
		w.xdgSurface.Destroy()
	}
	if w.surface != nil {
		w.surface.Destroy()
	}
	if w.decoration != nil {
		w.decoration.Destroy()
	}
	if w.wmBase != nil {
		w.wmBase.Destroy()
	}
	if w.compositor != nil {
		w.compositor.Destroy()
	}
	if w.registry != nil {
		w.registry.Destroy()
	}
	_ = w.display.Flush()
	w.display.Disconnect()
}
