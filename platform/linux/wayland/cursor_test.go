//go:build linux

package wayland

import (
	"os"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"

	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
)

func TestCursorDestroyDestroysSurface(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" || os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("needs WAYLAND_DISPLAY and GPUI_TEST_DISPLAY=1")
	}
	display, err := wl.Connect("")
	if err != nil {
		t.Skip(err)
	}
	defer display.Disconnect()
	registry := wlproto.GetDisplay(display).GetRegistry()
	globals := &globalList{registry: registry}
	registry.SetListener(wlproto.WlRegistryListener{
		Global: func(name uint32, iface string, version uint32) {
			globals.list = append(globals.list, globalEntry{name: name, iface: iface, version: version})
		},
	})
	if _, err := display.Roundtrip(); err != nil {
		t.Fatal(err)
	}
	compositor, err := bindGlobal[wlproto.WlCompositor](globals, wlproto.WlCompositorInterface, 3, 6)
	if err != nil {
		t.Fatal(err)
	}
	shm, err := bindGlobal[wlproto.WlShm](globals, wlproto.WlShmInterface, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	cursor := NewCursor(display, &Globals{Compositor: compositor, Shm: shm}, 24)
	if cursor.loadedTheme == nil {
		t.Fatal("no cursor theme loaded")
	}
	surface := cursor.surface
	cursor.Destroy()
	if surface.IsAlive() || cursor.loadedTheme != nil {
		t.Fatal("Destroy left the surface or theme alive")
	}
	if _, err := display.Roundtrip(); err != nil {
		t.Fatal(err)
	}
}

func TestClientWaylandSourceDispatchesThroughEventLoop(t *testing.T) {
	if os.Getenv("GPUI_TEST_DISPLAY") != "1" || os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("needs WAYLAND_DISPLAY and GPUI_TEST_DISPLAY=1")
	}
	display, err := wl.Connect("")
	if err != nil {
		t.Skip(err)
	}
	defer display.Disconnect()
	loop, err := common.NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer loop.Close()
	token, err := loop.Handle().InsertEventSource(newWaylandSource(display))
	if err != nil {
		t.Fatal(err)
	}
	defer loop.Handle().Remove(token)
	done := 0
	for range 3 {
		callback := wlproto.GetDisplay(display).Sync()
		callback.SetListener(wlproto.WlCallbackListener{Done: func(uint32) { done++ }})
	}
	for i := 0; i < 20 && done < 3; i++ {
		if err := loop.Dispatch(gpui.Some(100 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	if done != 3 {
		t.Fatalf("dispatched %d of 3 callbacks", done)
	}
	_ = detectCompositorGPU()
}
