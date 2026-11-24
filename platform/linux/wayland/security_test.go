//go:build linux

package wayland

import "testing"

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
