//go:build linux

package dbus

import (
	"context"
	"sync"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

func TestPower_MapsProfilesWithoutGuessingUnknownValues(t *testing.T) {
	for _, c := range []struct {
		profile  string
		expected gpui.PowerMode
	}{
		{"power-saver", gpui.PowerModeLowPower},
		{"balanced", gpui.PowerModeBalanced},
		{"performance", gpui.PowerModeHighPower},
		{"custom", gpui.PowerModeUnknown},
		{"", gpui.PowerModeUnknown},
	} {
		if got := powerMode(c.profile); got != c.expected {
			t.Fatalf("powerMode(%q) = %v, want %v", c.profile, got, c.expected)
		}
	}
}

type lockedProfile struct {
	mu      sync.Mutex
	profile string
}

func (p *lockedProfile) set(v string) {
	p.mu.Lock()
	p.profile = v
	p.mu.Unlock()
}

func TestPower_MonitorsProfilesAndServiceRestarts(t *testing.T) {
	address := startTestBus(t)
	modern := &lockedProfile{profile: "balanced"}
	legacy := &lockedProfile{profile: "balanced"}
	server := connectTestBus(t, address)
	for i, profile := range []*lockedProfile{modern, legacy} {
		s := powerProfilesServices[i]
		props := fakeProperties{get: func(iface, name string) (godbus.Variant, bool) {
			if iface != s.service || name != "ActiveProfile" {
				return godbus.Variant{}, false
			}
			profile.mu.Lock()
			defer profile.mu.Unlock()
			return godbus.MakeVariant(profile.profile), true
		}}
		if err := server.Export(props, s.path, "org.freedesktop.DBus.Properties"); err != nil {
			t.Fatal(err)
		}
	}
	conn := connectTestBus(t, address)
	modes := make(chan gpui.PowerMode, 16)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- MonitorPowerMode(ctx, conn, func(mode gpui.PowerMode) error {
			modes <- mode
			return nil
		})
	}()
	defer func() {
		cancel()
		<-done
	}()
	deadline := time.After(15 * time.Second)
	expect := func(want gpui.PowerMode) {
		t.Helper()
		select {
		case got := <-modes:
			if got != want {
				t.Fatalf("mode = %v, want %v", got, want)
			}
		case <-deadline:
			t.Fatal("timed out monitoring mock power profiles")
		}
	}
	request := func(name string) {
		t.Helper()
		if reply, err := server.RequestName(name, godbus.NameFlagDoNotQueue); err != nil || reply != godbus.RequestNameReplyPrimaryOwner {
			t.Fatalf("RequestName(%s) = %v, %v", name, reply, err)
		}
	}
	release := func(name string) {
		t.Helper()
		if _, err := server.ReleaseName(name); err != nil {
			t.Fatal(err)
		}
	}
	expect(gpui.PowerModeUnknown)

	for i, active := range []*lockedProfile{modern, legacy} {
		s := powerProfilesServices[i]
		request(s.service)
		expect(gpui.PowerModeBalanced)
		for _, c := range []struct {
			profile  string
			expected gpui.PowerMode
		}{
			{"power-saver", gpui.PowerModeLowPower},
			{"performance", gpui.PowerModeHighPower},
			{"custom", gpui.PowerModeUnknown},
			{"balanced", gpui.PowerModeBalanced},
		} {
			active.set(c.profile)
			if err := server.Emit(s.path, "org.freedesktop.DBus.Properties.PropertiesChanged", s.service,
				map[string]godbus.Variant{"ActiveProfile": godbus.MakeVariant(c.profile)}, []string{}); err != nil {
				t.Fatal(err)
			}
			expect(c.expected)
		}
		release(s.service)
		expect(gpui.PowerModeUnknown)
		request(s.service)
		expect(gpui.PowerModeBalanced)
		release(s.service)
		expect(gpui.PowerModeUnknown)
	}
	legacy.set("power-saver")
	modern.set("performance")
	request(powerProfilesServices[1].service)
	expect(gpui.PowerModeLowPower)
	request(powerProfilesServices[0].service)
	expect(gpui.PowerModeHighPower)
	release(powerProfilesServices[0].service)
	expect(gpui.PowerModeLowPower)
	release(powerProfilesServices[1].service)
	expect(gpui.PowerModeUnknown)
}
