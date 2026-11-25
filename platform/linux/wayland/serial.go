//go:build linux

package wayland

import "github.com/megascan/gpui-go/internal/linux/wlproto"

// SerialKind is the kind of input event a serial came from.
type SerialKind uint8

const (
	SerialKindDataDevice SerialKind = iota
	SerialKindInputMethod
	SerialKindMouseEnter
	SerialKindMousePress
	SerialKindKeyPress
)

// Serial is a Wayland event serial.
type Serial uint32

// AsRaw returns the serial's protocol value.
func (s Serial) AsRaw() uint32 { return uint32(s) }

// SelectionSerial is a serial that may be used to take ownership of a selection.
type SelectionSerial struct{ serial Serial }

// AsRaw returns the serial's protocol value.
func (s SelectionSerial) AsRaw() uint32 { return s.serial.AsRaw() }

// ActivationContext is the user interaction an xdg_activation_v1 token request must reference.
type ActivationContext struct {
	Serial            Serial
	RequestingSurface *wlproto.WlSurface
}

type press struct {
	serial         Serial
	focusedSurface *wlproto.WlSurface
}

// SerialTracker tracks the latest serial of each SerialKind.
type SerialTracker struct {
	serials     map[SerialKind]Serial
	latestPress *press
}

// NewSerialTracker returns an empty tracker.
func NewSerialTracker() *SerialTracker {
	return &SerialTracker{serials: map[SerialKind]Serial{}}
}

// Update records value as the latest serial of kind.
func (t *SerialTracker) Update(kind SerialKind, value uint32) {
	t.UpdateWithFocusedSurface(kind, value, nil)
}

// UpdateWithFocusedSurface is Update, but for key and pointer presses also records the surface
// that had focus, which activation token requests must name alongside the serial.
func (t *SerialTracker) UpdateWithFocusedSurface(kind SerialKind, value uint32, focusedSurface *wlproto.WlSurface) {
	serial := Serial(value)
	if kind == SerialKindKeyPress || kind == SerialKindMousePress {
		t.latestPress = &press{serial: serial, focusedSurface: focusedSurface}
	}
	t.serials[kind] = serial
}

// Get returns the latest tracked serial of kind, or 0 if the kind has not been tracked.
func (t *SerialTracker) Get(kind SerialKind) Serial {
	return t.serials[kind]
}

// SelectionSerial returns the serial of the latest key or pointer press.
func (t *SerialTracker) SelectionSerial() (SelectionSerial, bool) {
	if t.latestPress == nil {
		return SelectionSerial{}, false
	}
	return SelectionSerial{serial: t.latestPress.serial}, true
}

// ActivationContext returns the latest press if a still-alive surface had focus when it arrived.
func (t *SerialTracker) ActivationContext() (ActivationContext, bool) {
	p := t.latestPress
	if p == nil || p.focusedSurface == nil || !p.focusedSurface.IsAlive() {
		return ActivationContext{}, false
	}
	return ActivationContext{Serial: p.serial, RequestingSurface: p.focusedSurface}, true
}
