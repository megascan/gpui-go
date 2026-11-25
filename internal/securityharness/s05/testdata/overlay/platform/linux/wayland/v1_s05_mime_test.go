//go:build securityharness && linux

package wayland

import (
	"strconv"
	"testing"
	"time"
)

type v1NopReceiver struct{}

func (v1NopReceiver) Receive(string, int) {}

func TestSecV1_S05_DataOfferMimeAppendQuadratic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		n     int
		limit time.Duration
	}{
		{"control_1k", 1000, time.Second},
		{"attack_200k", 200000, 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mimes := make([]string, tc.n)
			for i := range mimes {
				mimes[i] = "x/" + strconv.Itoa(i)
			}
			offer := NewDataOffer[v1NopReceiver](v1NopReceiver{})
			done := make(chan time.Duration, 1)
			go func() {
				start := time.Now()
				for _, m := range mimes {
					offer.AddMimeType(m)
				}
				done <- time.Since(start)
			}()
			select {
			case d := <-done:
				t.Logf("%d offers took %s", tc.n, d)
				if d > tc.limit {
					t.Fatalf("%d mime offers took %s (> %s)", tc.n, d, tc.limit)
				}
			case <-time.After(tc.limit):
				t.Fatalf("%d mime offers still being appended after %s (quadratic copy per AddMimeType)", tc.n, tc.limit)
			}
		})
	}
}
