//go:build linux

package dbus

import (
	"math"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

func TestXdgDesktopPortal_TextScaleFactorIsClampedToAUsableRange(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want float32
		ok   bool
	}{
		{1.0, 1.0, true},
		{1.25, 1.25, true},
		{2.0, 2.0, true},
		{0.1, 0.5, true},
		{10.0, 3.0, true},
		{0.0, 0, false},
		{-1.0, 0, false},
		{math.NaN(), 0, false},
		{math.Inf(1), 0, false},
	} {
		got, ok := textScaleFactor(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("textScaleFactor(%v) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

type fakeSettings struct{ values map[[2]string]godbus.Variant }

func (f fakeSettings) Read(namespace, key string) (godbus.Variant, *godbus.Error) {
	v, ok := f.values[[2]string{namespace, key}]
	if !ok {
		return godbus.Variant{}, godbus.NewError("org.freedesktop.portal.Error.NotFound", []any{key})
	}
	return godbus.MakeVariant(v), nil
}

func TestXdgDesktopPortal_ForwardsInitialAndChangedSettings(t *testing.T) {
	address := startTestBus(t)
	server := connectTestBus(t, address)
	fake := fakeSettings{values: map[[2]string]godbus.Variant{
		{appearanceNamespace, colorSchemeKey}:   godbus.MakeVariant(colorSchemePreferDark),
		{gnomeInterface, "cursor-size"}:         godbus.MakeVariant(int32(24)),
		{gnomeInterface, "text-scaling-factor"}: godbus.MakeVariant(10.0),
		{gnomeWMPreferences, "button-layout"}:   godbus.MakeVariant("close:"),
	}}
	if err := server.Export(fake, portalPath, portalSettings); err != nil {
		t.Fatal(err)
	}
	if _, err := server.RequestName(portalService, godbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	events := make(chan Event, 16)
	source := NewXDPEventSource(connectTestBus(t, address), func(e Event) { events <- e })
	defer source.Close()
	next := func() Event {
		t.Helper()
		select {
		case e := <-events:
			return e
		case <-time.After(10 * time.Second):
			t.Fatal("no portal event")
			return nil
		}
	}
	for _, want := range []Event{
		EventWindowAppearance{gpui.WindowAppearanceDark},
		EventCursorSize{24},
		EventButtonLayout{"close:"},
		EventTextScaleFactor{3},
	} {
		if got := next(); got != want {
			t.Fatalf("event = %#v, want %#v", got, want)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := server.Emit(portalPath, settingChangedSignal, gnomeInterface, "cursor-theme", godbus.MakeVariant("Adwaita")); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-events:
			if got != (EventCursorTheme{"Adwaita"}) {
				t.Fatalf("event = %#v", got)
			}
			if err := server.Emit(portalPath, settingChangedSignal, appearanceNamespace, colorSchemeKey, godbus.MakeVariant(colorSchemePreferLight)); err != nil {
				t.Fatal(err)
			}
			if got := next(); got != (EventWindowAppearance{gpui.WindowAppearanceLight}) {
				t.Fatalf("event = %#v", got)
			}
			return
		case <-time.After(100 * time.Millisecond):
			if time.Now().After(deadline) {
				t.Fatal("no change event")
			}
		}
	}
}
