//go:build linux

package dbus

import (
	"context"
	"log/slog"
	"math"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

// Event is a desktop setting forwarded from the portal.
type Event interface{ isXDPEvent() }

// EventWindowAppearance reports the preferred color scheme.
type EventWindowAppearance struct{ Appearance gpui.WindowAppearance }

// EventCursorTheme reports the cursor theme name.
type EventCursorTheme struct{ Theme string }

// EventCursorSize reports the cursor size.
type EventCursorSize struct{ Size uint32 }

// EventButtonLayout reports the window manager button layout.
type EventButtonLayout struct{ Layout string }

// EventTextScaleFactor reports the clamped text scaling factor.
type EventTextScaleFactor struct{ Factor float32 }

func (EventWindowAppearance) isXDPEvent() {}
func (EventCursorTheme) isXDPEvent()      {}
func (EventCursorSize) isXDPEvent()       {}
func (EventButtonLayout) isXDPEvent()     {}
func (EventTextScaleFactor) isXDPEvent()  {}

const (
	portalService          = "org.freedesktop.portal.Desktop"
	portalPath             = godbus.ObjectPath("/org/freedesktop/portal/desktop")
	portalSettings         = "org.freedesktop.portal.Settings"
	appearanceNamespace    = "org.freedesktop.appearance"
	colorSchemeKey         = "color-scheme"
	gnomeInterface         = "org.gnome.desktop.interface"
	gnomeWMPreferences     = "org.gnome.desktop.wm.preferences"
	minTextScaleFactor     = float32(0.5)
	maxTextScaleFactor     = float32(3.)
	settingChangedSignal   = portalSettings + ".SettingChanged"
	colorSchemePreferDark  = uint32(1)
	colorSchemePreferLight = uint32(2)
)

// XDPEventSource forwards desktop settings from the portal. It owns the
// listener goroutine; Close cancels it and releases its D-Bus subscriptions.
type XDPEventSource struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// NewXDPEventSource starts listening on conn and calls send for every event,
// from the listener goroutine; the caller marshals events to the main thread.
func NewXDPEventSource(conn *godbus.Conn, send func(Event)) *XDPEventSource {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := listen(ctx, conn, send); err != nil {
			slog.Debug("stopped listening to desktop portal settings: " + err.Error())
		}
	}()
	return &XDPEventSource{cancel: cancel, done: done}
}

// Close stops the listener and waits for it to exit. It is idempotent.
func (s *XDPEventSource) Close() {
	s.cancel()
	<-s.done
}

type portalSetting struct {
	namespace string
	key       string
	convert   func(godbus.Variant) (Event, bool)
}

func stringSetting(f func(string) Event) func(godbus.Variant) (Event, bool) {
	return func(v godbus.Variant) (Event, bool) {
		s, ok := v.Value().(string)
		if !ok {
			return nil, false
		}
		return f(s), true
	}
}

var portalSettingsWatched = []portalSetting{
	{gnomeInterface, "cursor-theme", stringSetting(func(s string) Event { return EventCursorTheme{s} })},
	{gnomeInterface, "cursor-size", func(v godbus.Variant) (Event, bool) {
		n, ok := v.Value().(int32)
		if !ok {
			return nil, false
		}
		return EventCursorSize{uint32(n)}, true
	}},
	{gnomeWMPreferences, "button-layout", stringSetting(func(s string) Event { return EventButtonLayout{s} })},
	{gnomeInterface, "text-scaling-factor", func(v godbus.Variant) (Event, bool) {
		f, ok := v.Value().(float64)
		if !ok {
			return nil, false
		}
		scale, ok := textScaleFactor(f)
		if !ok {
			return nil, true
		}
		return EventTextScaleFactor{scale}, true
	}},
}

var colorSchemeSetting = portalSetting{appearanceNamespace, colorSchemeKey, func(v godbus.Variant) (Event, bool) {
	cs, ok := v.Value().(uint32)
	if !ok {
		return nil, false
	}
	return EventWindowAppearance{windowAppearanceFromColorScheme(cs)}, true
}}

func readSetting(ctx context.Context, conn *godbus.Conn, namespace, key string) (godbus.Variant, error) {
	var value godbus.Variant
	err := conn.Object(portalService, portalPath).CallWithContext(ctx, portalSettings+".Read", 0, namespace, key).Store(&value)
	if err != nil {
		return value, err
	}
	if inner, ok := value.Value().(godbus.Variant); ok {
		return inner, nil
	}
	return value, nil
}

func listen(ctx context.Context, conn *godbus.Conn, send func(Event)) error {
	signals := make(chan *godbus.Signal, 32)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	initial := append([]portalSetting{colorSchemeSetting}, portalSettingsWatched...)
	for _, setting := range initial {
		v, err := readSetting(ctx, conn, setting.namespace, setting.key)
		if err != nil {
			continue
		}
		if ev, ok := setting.convert(v); ok && ev != nil {
			send(ev)
		}
	}

	active := make(map[[2]string]portalSetting)
	var matches [][]godbus.MatchOption
	defer func() {
		for _, m := range matches {
			_ = conn.RemoveMatchSignal(m...)
		}
	}()
	watch := append(append([]portalSetting{}, portalSettingsWatched...), colorSchemeSetting)
	for _, setting := range watch {
		m := []godbus.MatchOption{
			godbus.WithMatchObjectPath(portalPath),
			godbus.WithMatchInterface(portalSettings),
			godbus.WithMatchMember("SettingChanged"),
			godbus.WithMatchArg(0, setting.namespace),
			godbus.WithMatchArg(1, setting.key),
		}
		if err := conn.AddMatchSignalContext(ctx, m...); err != nil {
			if setting.key == colorSchemeKey {
				return err
			}
			continue
		}
		matches = append(matches, m)
		active[[2]string{setting.namespace, setting.key}] = setting
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case sig, ok := <-signals:
			if !ok {
				return nil
			}
			if sig.Name != settingChangedSignal || sig.Path != portalPath || len(sig.Body) < 3 {
				continue
			}
			namespace, _ := sig.Body[0].(string)
			key, _ := sig.Body[1].(string)
			value, isVariant := sig.Body[2].(godbus.Variant)
			k := [2]string{namespace, key}
			setting, watched := active[k]
			if !watched || !isVariant {
				continue
			}
			ev, ok := setting.convert(value)
			if !ok {
				if key == colorSchemeKey {
					continue
				}
				delete(active, k)
				continue
			}
			if ev != nil {
				send(ev)
			}
		}
	}
}

func textScaleFactor(value float64) (float32, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, false
	}
	return min(max(float32(value), minTextScaleFactor), maxTextScaleFactor), true
}

func windowAppearanceFromColorScheme(cs uint32) gpui.WindowAppearance {
	switch cs {
	case colorSchemePreferDark:
		return gpui.WindowAppearanceDark
	case colorSchemePreferLight:
		return gpui.WindowAppearanceLight
	default:
		return gpui.WindowAppearanceLight
	}
}
