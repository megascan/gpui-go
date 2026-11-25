//go:build linux

package wgpu

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu/webgpu"
	native "github.com/oliverbestmann/webgpu/wgpu"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func TestWgpuContext_FeatureProbe(t *testing.T) {
	instance := createInstance(gpu.InstanceDescriptor{Backends: gpu.BackendsAll})
	adapters := instance.EnumerateAdapters(gpu.BackendsAll)
	t.Logf("EnumerateAdapters: exported, %d adapter(s)", len(adapters))
	if len(adapters) == 0 {
		t.Skip("no GPU adapter")
	}
	child := exec.Command(os.Args[0], "-test.run=^TestWgpuContext_TransientAttachmentProbeChild$", "-test.v")
	child.Env = append(os.Environ(), "GPUI_WGPU_TRANSIENT_PROBE=1")
	output, err := child.CombinedOutput()
	switch {
	case err == nil && strings.Contains(string(output), "TRANSIENT_ATTACHMENT accepted"):
		t.Logf("TRANSIENT_ATTACHMENT: accepted by wgpu-native")
		if !webgpu.SupportsTransientAttachment {
			t.Logf("TRANSIENT_ATTACHMENT now works: drop the L-WGPUVER fallback (webgpu.SupportsTransientAttachment)")
		}
	default:
		t.Logf("TRANSIENT_ATTACHMENT: unusable (child %v): %s", err, lastLine(string(output)))
	}
	for _, adapter := range adapters {
		info := adapter.GetInfo()
		dual := adapter.Features().Contains(gpu.FeaturesDualSourceBlending)
		t.Logf("adapter %q driver=%q driver_info=%q vendor=0x%04x device=0x%04x type=%s backend=%s dual_source_blending=%v",
			info.Name, info.Driver, info.DriverInfo, info.Vendor, info.Device, info.DeviceType, info.Backend, dual)

		var features gpu.Features
		if dual {
			features = gpu.FeaturesDualSourceBlending
		}
		device, _, err := adapter.RequestDevice(&gpu.DeviceDescriptor{
			Label:            "probe",
			RequiredFeatures: features,
			RequiredLimits:   gpu.DownlevelDefaults().UsingResolution(adapter.Limits()).UsingAlignment(adapter.Limits()),
		})
		if err != nil {
			t.Logf("  RequestDevice failed: %v", err)
			continue
		}
		t.Logf("  device dual_source_blending=%v", device.Features().Contains(gpu.FeaturesDualSourceBlending))

		for _, variant := range []struct {
			name     string
			source   string
			subpixel bool
		}{
			{"storage", storageBufferShaders, false},
			{"webgl", webglShaders, false},
			{"subpixel_storage", subpixelStorageBufferShaders, true},
			{"subpixel_webgl", subpixelWebglShaders, true},
		} {
			if variant.subpixel && !dual {
				t.Logf("  shader %s: skipped, no dual-source blending", variant.name)
				continue
			}
			device.PushErrorScope(gpu.ErrorFilterValidation)
			module := device.CreateShaderModule(&gpu.ShaderModuleDescriptor{Label: variant.name, WGSL: variant.source})
			if scopeErr := device.PopErrorScope(); scopeErr != nil {
				t.Errorf("  shader %s failed: %v", variant.name, scopeErr)
			} else {
				t.Logf("  shader %s compiled", variant.name)
			}
			module.Release()
		}
		device.Release()
	}
}

func lastLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "panicked") || strings.Contains(line, "invalid") || strings.Contains(line, "rejected") {
			return strings.TrimSpace(line)
		}
	}
	return strings.TrimSpace(s)
}

func TestWgpuContext_TransientAttachmentProbeChild(t *testing.T) {
	if os.Getenv("GPUI_WGPU_TRANSIENT_PROBE") != "1" {
		t.Skip("run by TestWgpuContext_FeatureProbe in a child process")
	}
	instance := native.CreateInstance(&native.InstanceDescriptor{})
	adapter, err := instance.RequestAdapter(&native.RequestAdapterOptions{})
	if err != nil {
		t.Skipf("no adapter: %v", err)
	}
	device, err := adapter.RequestDevice(&native.DeviceDescriptor{Label: "probe"})
	if err != nil {
		t.Skipf("no device: %v", err)
	}
	_, err = device.TryCreateTexture(&native.TextureDescriptor{
		Label:         "probe_transient",
		Usage:         native.TextureUsageRenderAttachment | native.TextureUsageTransientAttachment,
		Dimension:     native.TextureDimension2D,
		Size:          native.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 1},
		Format:        native.TextureFormatBGRA8Unorm,
		MipLevelCount: 1,
		SampleCount:   1,
	})
	if err != nil {
		t.Fatalf("TRANSIENT_ATTACHMENT rejected: %v", err)
	}
	t.Log("TRANSIENT_ATTACHMENT accepted")
}
