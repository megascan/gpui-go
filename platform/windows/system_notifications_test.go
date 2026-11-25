//go:build windows

package windows

import "testing"

func TestSystemNotifications_TagHashMatchesDefaultHasher(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want uint64
	}{
		{"", 0x30406ea523c53def},
		{"tag", 0x3344c74822e37db7},
		{"a-much-longer-notification-tag-value", 0x455d1819a108c6a0},
	} {
		if got := hashString(tc.tag); got != tc.want {
			t.Errorf("hashString(%q) = %016x, want %016x", tc.tag, got, tc.want)
		}
	}
}
