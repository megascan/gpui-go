//go:build js

package wgpu

import (
	"errors"
	"syscall/js"
	"testing"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func TestWgpuContextJS_WebBackendPreferenceBackends(t *testing.T) {
	cases := []struct {
		preference WebBackendPreference
		name       string
		backends   gpu.Backends
	}{
		{WebBackendPreferenceAuto, "Auto", gpu.BackendsBrowserWebGpu | gpu.BackendsGl},
		{WebBackendPreferenceWebGpu, "WebGpu", gpu.BackendsBrowserWebGpu},
		{WebBackendPreferenceWebGl, "WebGl", gpu.BackendsGl},
	}
	var zero WebBackendPreference
	if zero != WebBackendPreferenceAuto {
		t.Fatalf("default preference = %s, want Auto", zero)
	}
	for _, c := range cases {
		if got := c.preference.String(); got != c.name {
			t.Errorf("String() = %q, want %q", got, c.name)
		}
		if got := c.preference.backends(); got != c.backends {
			t.Errorf("%s backends = %v, want %v", c.name, got, c.backends)
		}
	}
}

func TestWgpuContextJS_WebGlBranchReturnsError(t *testing.T) {
	_, err := wgpuContextNewWebWithBackend(js.Undefined(), WebBackendPreferenceWebGl)
	if !errors.Is(err, errWebGlUnavailable) {
		t.Fatalf("err = %v, want errWebGlUnavailable", err)
	}
}
