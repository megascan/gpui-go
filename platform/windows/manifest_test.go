//go:build windows && !gpui_no_manifest

package windows

import (
	"testing"

	"github.com/megascan/gpui-go/internal/win/win32"
)

func TestManifest_PerMonitorV2DpiAwareness(t *testing.T) {
	platform, err := NewWindowsPlatform(true)
	if err != nil {
		t.Fatal(err)
	}
	_ = platform
	if !win32.AreDpiAwarenessContextsEqual(win32.GetThreadDpiAwarenessContext(), win32.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) {
		t.Fatalf("thread DPI awareness context %#x is not per-monitor-v2", win32.GetThreadDpiAwarenessContext())
	}
}
