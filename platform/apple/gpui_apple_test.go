//go:build darwin

package apple

import (
	"testing"

	"github.com/megascan/gpui-go/internal/apple/gcd"
)

func TestGpuiApple_MainThreadMarkerRejectsBackgroundThreads(t *testing.T) {
	results := make(chan [2]bool, 1)
	if err := gcd.SpawnThread(func() {
		_, first := MainThreadMarker()
		_, second := MainThreadMarker()
		results <- [2]bool{first, second}
	}); err != nil {
		t.Fatalf("background thread panicked: %v", err)
	}
	got := <-results
	if got[0] || got[1] {
		t.Fatalf("main_thread_marker() on a background thread = %v, want None twice", got)
	}
}
