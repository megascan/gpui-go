//go:build linux

package wayland

import (
	"encoding/hex"
	"testing"
)

func TestDisplay_RefreshIntervalFromMillihertz(t *testing.T) {
	cases := []struct {
		millihertz int32
		micros     int64
		ok         bool
	}{
		{60_000, 16_666, true},
		{59_940, 16683, true},
		{143_998, 6944, true},
		{0, 0, false},
		{-1, 0, false},
	}
	for _, c := range cases {
		interval, ok := refreshIntervalFromMillihertz(c.millihertz).Get()
		if ok != c.ok || (ok && interval.Microseconds() != c.micros) {
			t.Fatalf("refreshIntervalFromMillihertz(%d) = (%v, %v), want (%dus, %v)", c.millihertz, interval, ok, c.micros, c.ok)
		}
	}
}

func TestDisplayUUIDV5(t *testing.T) {
	u := uuidNewV5(uuidNamespaceDNS, []byte("python.org"))
	if got := hex.EncodeToString(u[:]); got != "886313e13b8a53729b900c9aee199e5d" {
		t.Fatalf("uuid v5 = %s", got)
	}
}
