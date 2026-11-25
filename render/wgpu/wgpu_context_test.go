//go:build linux || js

package wgpu

import (
	"fmt"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func observe(errors *DeviceErrorState, generation *uint64) gpuiutil.Option[string] {
	return gpuiutil.OptionFrom(errors.ObserveError(generation))
}

func assertObserved(t *testing.T, got gpuiutil.Option[string], want gpuiutil.Option[string]) {
	t.Helper()
	if got != want {
		t.Fatalf("observe_error = %v, want %v", got, want)
	}
}

func TestWgpuContext_DeviceErrorsAreObservedIndependently(t *testing.T) {
	errors := &DeviceErrorState{}
	var first, second uint64
	assertObserved(t, observe(errors, &first), gpuiutil.None[string]())
	errors.record("first")
	assertObserved(t, observe(errors, &first), gpuiutil.Some("first"))
	assertObserved(t, observe(errors, &first), gpuiutil.None[string]())
	assertObserved(t, observe(errors, &second), gpuiutil.Some("first"))
	assertObserved(t, observe(errors, &second), gpuiutil.None[string]())

	errors.record("second")
	assertObserved(t, observe(errors, &second), gpuiutil.Some("second"))
	assertObserved(t, observe(errors, &first), gpuiutil.Some("second"))
	assertObserved(t, observe(errors, &first), gpuiutil.None[string]())
	assertObserved(t, observe(errors, &second), gpuiutil.None[string]())
}

func TestWgpuContext_DeviceErrorsCoalesceAndNewObserversSeeLatest(t *testing.T) {
	errors := &DeviceErrorState{}
	errors.record("older")
	errors.record("latest")
	var generation uint64
	assertObserved(t, observe(errors, &generation), gpuiutil.Some("latest"))
	assertObserved(t, observe(errors, &generation), gpuiutil.None[string]())

	replacement := &DeviceErrorState{}
	replacement.record("replacement")
	generation = 0
	assertObserved(t, observe(replacement, &generation), gpuiutil.Some("replacement"))
	assertObserved(t, observe(replacement, &generation), gpuiutil.None[string]())
}

func TestWgpuContext_ParseDeviceID(t *testing.T) {
	ok := func(s string) uint32 {
		t.Helper()
		v, err := parsePCIID(s)
		if err != nil {
			t.Fatalf("parsePCIID(%q): %v", s, err)
		}
		return v
	}
	ok("0xABCD")
	ok("ABCD")
	ok("abcd")
	ok("1234")
	if _, err := parsePCIID("123"); err == nil {
		t.Fatal("parsePCIID(\"123\") should fail")
	}
	if ok(fmt.Sprintf("%x", 0x1234)) != ok(fmt.Sprintf("%X", 0x1234)) {
		t.Fatal("lower and upper hex differ")
	}
	if ok(fmt.Sprintf("%#x", 0x1234)) != ok(fmt.Sprintf("%#X", 0x1234)) {
		t.Fatal("prefixed lower and upper hex differ")
	}
}

type fakeAdapter struct {
	gpu.Adapter
	name string
	info gpu.AdapterInfo
}

func (a *fakeAdapter) GetInfo() gpu.AdapterInfo {
	info := a.info
	info.Name = a.name
	return info
}

func names(adapters []gpu.Adapter) []string {
	out := make([]string, len(adapters))
	for i, a := range adapters {
		out[i] = a.GetInfo().Name
	}
	return out
}

func TestWgpuContext_SortAdapters(t *testing.T) {
	newAdapters := func() []gpu.Adapter {
		return []gpu.Adapter{
			&fakeAdapter{name: "cpu", info: gpu.AdapterInfo{Vendor: 0x10005, Device: 0, DeviceType: gpu.DeviceTypeCpu, Backend: gpu.BackendVulkan}},
			&fakeAdapter{name: "gl-intel", info: gpu.AdapterInfo{Vendor: 0x8086, Device: 0, DeviceType: gpu.DeviceTypeOther, Backend: gpu.BackendGl}},
			&fakeAdapter{name: "intel", info: gpu.AdapterInfo{Vendor: 0x8086, Device: 0x46a6, DeviceType: gpu.DeviceTypeIntegratedGpu, Backend: gpu.BackendVulkan}},
			&fakeAdapter{name: "virtual", info: gpu.AdapterInfo{Vendor: 0x1af4, Device: 0x1050, DeviceType: gpu.DeviceTypeVirtualGpu, Backend: gpu.BackendVulkan}},
			&fakeAdapter{name: "nvidia-gl", info: gpu.AdapterInfo{Vendor: 0x10de, Device: 0x2520, DeviceType: gpu.DeviceTypeDiscreteGpu, Backend: gpu.BackendGl}},
			&fakeAdapter{name: "nvidia", info: gpu.AdapterInfo{Vendor: 0x10de, Device: 0x2520, DeviceType: gpu.DeviceTypeDiscreteGpu, Backend: gpu.BackendVulkan}},
		}
	}
	check := func(got, want []string) {
		t.Helper()
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	adapters := newAdapters()
	wgpuContextSortAdapters(adapters, gpuiutil.None[uint32](), gpuiutil.None[CompositorGPUHint]())
	check(names(adapters), []string{"nvidia", "nvidia-gl", "intel", "gl-intel", "virtual", "cpu"})

	adapters = newAdapters()
	wgpuContextSortAdapters(adapters, gpuiutil.None[uint32](), gpuiutil.Some(CompositorGPUHint{VendorID: 0x8086, DeviceID: 0x46a6}))
	check(names(adapters), []string{"intel", "nvidia", "nvidia-gl", "gl-intel", "virtual", "cpu"})

	adapters = newAdapters()
	wgpuContextSortAdapters(adapters, gpuiutil.Some[uint32](0x1050), gpuiutil.Some(CompositorGPUHint{VendorID: 0x8086, DeviceID: 0x46a6}))
	check(names(adapters), []string{"virtual", "intel", "nvidia", "nvidia-gl", "gl-intel", "cpu"})

	adapters = newAdapters()
	wgpuContextSortAdapters(adapters, gpuiutil.Some[uint32](0), gpuiutil.Some(CompositorGPUHint{VendorID: 0x8086, DeviceID: 0}))
	check(names(adapters), []string{"nvidia", "nvidia-gl", "intel", "gl-intel", "virtual", "cpu"})
}

type releaseCountingAdapter struct {
	fakeAdapter
	releases          int
	failConfigure     bool
	device            *fakeDevice
	queue             *fakeQueue
	surfaceFormats    []gpu.TextureFormat
	surfaceAlphaModes []gpu.CompositeAlphaMode
}

func (a *releaseCountingAdapter) Release() { a.releases++ }

func (a *releaseCountingAdapter) Features() gpu.Features { return 0 }

func (a *releaseCountingAdapter) Limits() gpu.Limits { return gpu.DownlevelDefaults() }

func (a *releaseCountingAdapter) GetTextureFormatFeatures(gpu.TextureFormat) gpu.TextureFormatFeatures {
	return gpu.TextureFormatFeatures{AllowedUsages: gpu.TextureUsagesTextureBinding | gpu.TextureUsagesCopyDst}
}

func (a *releaseCountingAdapter) GetDownlevelCapabilities() gpu.DownlevelCapabilities {
	return gpu.DownlevelCapabilities{Flags: gpu.DownlevelFlagsVertexStorage | gpu.DownlevelFlagsFragmentStorage}
}

func (a *releaseCountingAdapter) RequestDevice(*gpu.DeviceDescriptor) (gpu.Device, gpu.Queue, error) {
	a.device = &fakeDevice{failConfigure: a.failConfigure}
	a.queue = &fakeQueue{}
	return a.device, a.queue, nil
}

type fakeDevice struct {
	gpu.Device
	failConfigure bool
	scopes        []*gpu.Error
	releases      int
}

func (d *fakeDevice) Limits() gpu.Limits { return gpu.DownlevelDefaults() }

func (d *fakeDevice) PushErrorScope(gpu.ErrorFilter) { d.scopes = append(d.scopes, nil) }

func (d *fakeDevice) PopErrorScope() *gpu.Error {
	err := d.scopes[len(d.scopes)-1]
	d.scopes = d.scopes[:len(d.scopes)-1]
	return err
}

func (d *fakeDevice) Release() { d.releases++ }

type fakeQueue struct {
	gpu.Queue
	releases int
}

func (q *fakeQueue) Release() { q.releases++ }

type fakeSurface struct {
	gpu.Surface
}

func (s *fakeSurface) GetCapabilities(adapter gpu.Adapter) gpu.SurfaceCapabilities {
	a := adapter.(*releaseCountingAdapter)
	return gpu.SurfaceCapabilities{
		Formats:      a.surfaceFormats,
		PresentModes: []gpu.PresentMode{gpu.PresentModeFifo},
		AlphaModes:   a.surfaceAlphaModes,
		Usages:       gpu.TextureUsagesRenderAttachment,
	}
}

func (s *fakeSurface) Configure(device gpu.Device, _ *gpu.SurfaceConfiguration) {
	d := device.(*fakeDevice)
	if d.failConfigure && len(d.scopes) > 0 {
		d.scopes[len(d.scopes)-1] = &gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "configure failed"}
	}
}

type fakeInstance struct {
	gpu.Instance
	adapters []gpu.Adapter
}

func (i *fakeInstance) EnumerateAdapters(gpu.Backends) []gpu.Adapter { return i.adapters }

func newReleaseCountingAdapter(name string, formats bool, failConfigure bool) *releaseCountingAdapter {
	a := &releaseCountingAdapter{
		fakeAdapter:   fakeAdapter{name: name, info: gpu.AdapterInfo{Vendor: 0x10de, Device: 0x2520, DeviceType: gpu.DeviceTypeDiscreteGpu, Backend: gpu.BackendVulkan}},
		failConfigure: failConfigure,
	}
	if formats {
		a.surfaceFormats = []gpu.TextureFormat{gpu.TextureFormatBgra8Unorm}
		a.surfaceAlphaModes = []gpu.CompositeAlphaMode{gpu.CompositeAlphaModeOpaque}
	}
	return a
}

func TestWgpuContext_SelectAdapterReleasesRejectedAdaptersAndDevices(t *testing.T) {
	noFormats := newReleaseCountingAdapter("no-formats", false, false)
	badConfigure := newReleaseCountingAdapter("bad-configure", true, true)
	good := newReleaseCountingAdapter("good", true, false)
	unused := newReleaseCountingAdapter("unused", true, false)
	instance := &fakeInstance{adapters: []gpu.Adapter{noFormats, badConfigure, good, unused}}
	config := &WgpuSurfaceConfig{Size: gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}}

	selected, err := wgpuContextSelectAdapterAndDevice(instance, gpuiutil.None[uint32](), &fakeSurface{}, config, gpuiutil.None[CompositorGPUHint](), false)
	if err != nil {
		t.Fatalf("selection failed: %v", err)
	}
	if selected.adapter != gpu.Adapter(good) {
		t.Fatalf("selected %q, want good", selected.adapter.GetInfo().Name)
	}
	for _, a := range []*releaseCountingAdapter{noFormats, badConfigure, unused} {
		if a.releases != 1 {
			t.Fatalf("adapter %q released %d times, want 1", a.name, a.releases)
		}
	}
	if good.releases != 0 {
		t.Fatalf("selected adapter released %d times", good.releases)
	}
	if badConfigure.device.releases != 1 || badConfigure.queue.releases != 1 {
		t.Fatalf("rejected device released %d, queue %d, want 1 each", badConfigure.device.releases, badConfigure.queue.releases)
	}
	if good.device.releases != 0 || good.queue.releases != 0 {
		t.Fatal("selected device must not be released")
	}
}

func TestWgpuContext_SelectAdapterReleasesEveryAdapterOnFailure(t *testing.T) {
	a := newReleaseCountingAdapter("a", false, false)
	b := newReleaseCountingAdapter("b", true, true)
	instance := &fakeInstance{adapters: []gpu.Adapter{a, b}}
	config := &WgpuSurfaceConfig{Size: gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}}

	if _, err := wgpuContextSelectAdapterAndDevice(instance, gpuiutil.None[uint32](), &fakeSurface{}, config, gpuiutil.None[CompositorGPUHint](), false); err == nil {
		t.Fatal("selection should fail")
	}
	if a.releases != 1 || b.releases != 1 {
		t.Fatalf("adapters released %d and %d times, want 1 each", a.releases, b.releases)
	}
	if b.device.releases != 1 || b.queue.releases != 1 {
		t.Fatal("rejected device and queue should be released")
	}
}
