//go:build linux

package dbus

import (
	"context"
	"errors"
	"log/slog"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

type powerProfilesService struct {
	service string
	path    godbus.ObjectPath
}

var powerProfilesServices = [2]powerProfilesService{
	{"org.freedesktop.UPower.PowerProfiles", "/org/freedesktop/UPower/PowerProfiles"},
	{"net.hadess.PowerProfiles", "/net/hadess/PowerProfiles"},
}

func powerMode(profile string) gpui.PowerMode {
	switch profile {
	case "power-saver":
		return gpui.PowerModeLowPower
	case "balanced":
		return gpui.PowerModeBalanced
	case "performance":
		return gpui.PowerModeHighPower
	default:
		return gpui.PowerModeUnknown
	}
}

// MonitorPowerMode reports the active power profile through onChange until ctx
// is cancelled, onChange returns an error or the connection closes. It blocks;
// run it on a goroutine and marshal onChange to the main thread.
func MonitorPowerMode(ctx context.Context, conn *godbus.Conn, onChange func(gpui.PowerMode) error) error {
	signals := make(chan *godbus.Signal, 64)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	var matches [][]godbus.MatchOption
	for _, s := range powerProfilesServices {
		matches = append(matches,
			[]godbus.MatchOption{
				godbus.WithMatchSender(s.service),
				godbus.WithMatchObjectPath(s.path),
				godbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
				godbus.WithMatchMember("PropertiesChanged"),
			},
			[]godbus.MatchOption{
				godbus.WithMatchSender("org.freedesktop.DBus"),
				godbus.WithMatchInterface("org.freedesktop.DBus"),
				godbus.WithMatchMember("NameOwnerChanged"),
				godbus.WithMatchArg(0, s.service),
			},
		)
	}
	for _, m := range matches {
		if err := conn.AddMatchSignalContext(ctx, m...); err != nil {
			return err
		}
	}
	defer func() {
		for _, m := range matches {
			_ = conn.RemoveMatchSignal(m...)
		}
	}()

	previous := gpui.None[gpui.PowerMode]()
	for {
		mode := gpui.PowerModeUnknown
		for _, s := range powerProfilesServices {
			var profile string
			err := conn.Object(s.service, s.path).CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, s.service, "ActiveProfile").Store(&profile)
			if err == nil {
				mode = powerMode(profile)
				break
			}
			slog.Debug("could not query power profile from " + s.service + ": " + err.Error())
		}
		if p, ok := previous.Get(); !ok || p != mode {
			if err := onChange(mode); err != nil {
				return err
			}
			previous = gpui.Some(mode)
		}
		if !waitPowerChange(ctx, signals) {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := onChange(gpui.PowerModeUnknown); err != nil {
				return err
			}
			return errors.New("power profile notification streams closed")
		}
	}
}

func waitPowerChange(ctx context.Context, signals <-chan *godbus.Signal) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case sig, ok := <-signals:
			if !ok {
				return false
			}
			if isPowerSignal(sig) {
				return true
			}
		}
	}
}

func isPowerSignal(sig *godbus.Signal) bool {
	switch sig.Name {
	case "org.freedesktop.DBus.NameOwnerChanged":
		if len(sig.Body) > 0 {
			if name, ok := sig.Body[0].(string); ok {
				for _, s := range powerProfilesServices {
					if name == s.service {
						return true
					}
				}
			}
		}
	case "org.freedesktop.DBus.Properties.PropertiesChanged":
		for _, s := range powerProfilesServices {
			if sig.Path == s.path {
				return true
			}
		}
	}
	return false
}
