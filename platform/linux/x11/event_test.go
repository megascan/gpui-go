//go:build linux

package x11

import "testing"

func TestEvent_GetValuatorAxisIndex(t *testing.T) {
	tests := []struct {
		mask   []uint32
		number uint16
		want   int
		ok     bool
	}{
		{[]uint32{0b11}, 0, 0, true},
		{[]uint32{0b11}, 1, 1, true},
		{[]uint32{0b11}, 2, 0, false},

		{[]uint32{0b100}, 0, 0, false},
		{[]uint32{0b100}, 1, 0, false},
		{[]uint32{0b100}, 2, 0, true},
		{[]uint32{0b100}, 3, 0, false},

		{[]uint32{0b1010, 0}, 0, 0, false},
		{[]uint32{0b1010, 0}, 1, 0, true},
		{[]uint32{0b1010, 0}, 2, 0, false},
		{[]uint32{0b1010, 0}, 3, 1, true},

		{[]uint32{0b1010, 0b1}, 0, 0, false},
		{[]uint32{0b1010, 0b1}, 1, 0, true},
		{[]uint32{0b1010, 0b1}, 2, 0, false},
		{[]uint32{0b1010, 0b1}, 3, 1, true},
		{[]uint32{0b1010, 0b1}, 32, 2, true},
		{[]uint32{0b1010, 0b1}, 33, 0, false},

		{[]uint32{0b1010, 0b101}, 34, 3, true},
	}
	for _, tc := range tests {
		got, ok := getValuatorAxisIndex(tc.mask, tc.number)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("getValuatorAxisIndex(%b, %d) = %d, %v; want %d, %v", tc.mask, tc.number, got, ok, tc.want, tc.ok)
		}
	}
}
