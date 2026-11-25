//go:build linux

package wayland

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
)

type sec030Receiver struct{}

func (sec030Receiver) Receive(string, int) {}

func TestSecurity_SEC030_data_offer_mime_append_linear(t *testing.T) {
	allocs := testing.AllocsPerRun(5, func() {
		offer := NewDataOffer(sec030Receiver{})
		for range 20000 {
			offer.AddMimeType("text/plain")
		}
	})
	if allocs > 200 {
		t.Fatalf("20000 MIME appends made %.0f allocations, want amortized growth", allocs)
	}
}

func TestSecurity_FV3_nil_keymap_input_ignored(t *testing.T) {
	state := &WaylandClientState{}
	state.keyboardFocusedWindow = gpui.Some(WaylandWindowStatePtr{})
	p := WaylandClientStatePtr{cell: &clientCell{state: state}}
	listener := p.keyboardListener()
	listener.Modifiers(1, 1, 0, 0, 1)
	listener.Key(2, 0, 30, wlproto.WlKeyboardKeyStateReleased)
	if state.keymapState != nil || state.Modifiers != (gpui.Modifiers{}) {
		t.Fatalf("input before any keymap changed state: modifiers %+v", state.Modifiers)
	}
}
