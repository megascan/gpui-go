//go:build linux || js

package wgpu

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

// WgpuContext is the GPU instance, adapter, device and queue a renderer draws with. Copies
// share the same device.
type WgpuContext struct {
	Instance           gpu.Instance
	Adapter            gpu.Adapter
	Device             gpu.Device
	Queue              gpu.Queue
	backend            WgpuBackend
	dualSourceBlending bool
	colorTextureFormat gpu.TextureFormat
	errors             *DeviceErrorState
}

// DeviceErrorState holds the errors reported by wgpu's device-wide callbacks.
//
// A device has exactly one lost callback and one uncaptured-error callback, so this state is
// installed once per device and shared by every renderer using it. Renderers keep their own
// handle so they can keep observing a loss after the context that produced it has been
// dropped for recovery.
type DeviceErrorState struct {
	lost atomic.Bool

	mu                  sync.Mutex
	lastErrorGeneration uint64
	lastError           gpuiutil.Option[string]
}

func installDeviceErrorState(device gpu.Device) *DeviceErrorState {
	errors := &DeviceErrorState{}
	device.SetDeviceLostCallback(func(reason gpu.DeviceLostReason, message string) {
		slog.Error(fmt.Sprintf("wgpu device lost: reason=%s, message=%s", reason, message))
		if reason != gpu.DeviceLostReasonDestroyed {
			errors.lost.Store(true)
		}
	})
	device.OnUncapturedError(func(err *gpu.Error) {
		errors.record(err.Error())
	})
	return errors
}

// DeviceLost reports whether the GPU device was lost (e.g., due to driver crash, suspend/resume).
func (s *DeviceErrorState) DeviceLost() bool {
	return s.lost.Load()
}

func (s *DeviceErrorState) record(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastErrorGeneration++
	s.lastError = gpuiutil.Some(err)
}

// ObserveError returns the latest error once per observer, without consuming shared state.
// Start each observer at zero, including after switching devices, so errors raised during
// construction are observed. Multiple errors between observations coalesce into the latest
// error; this is not an error queue.
func (s *DeviceErrorState) ObserveError(generation *uint64) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if *generation == s.lastErrorGeneration {
		return "", false
	}
	*generation = s.lastErrorGeneration
	return s.lastError.Get()
}

// CurrentGeneration is the generation an observer should start from to see only errors raised
// after this call, for a renderer joining a device other renderers already used.
func (s *DeviceErrorState) CurrentGeneration() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErrorGeneration
}

// WgpuBackendKind is the variant of a WgpuBackend.
type WgpuBackendKind uint8

const (
	WgpuBackendBrowserWebGpu WgpuBackendKind = iota
	WgpuBackendGl
	WgpuBackendNative
)

// WgpuBackend is the backend a context runs on; Native is set for WgpuBackendNative.
type WgpuBackend struct {
	Kind   WgpuBackendKind
	Native gpu.Backend
}

// WgpuBackendNativeOf returns the Native(backend) variant.
func WgpuBackendNativeOf(backend gpu.Backend) WgpuBackend {
	return WgpuBackend{Kind: WgpuBackendNative, Native: backend}
}

// String formats the backend as upstream's Debug does.
func (b WgpuBackend) String() string {
	switch b.Kind {
	case WgpuBackendBrowserWebGpu:
		return "BrowserWebGpu"
	case WgpuBackendGl:
		return "Gl"
	}
	return "Native(" + b.Native.String() + ")"
}

// CompositorGPUHint names the GPU the display server renders on.
type CompositorGPUHint struct {
	VendorID uint32
	DeviceID uint32
}

// NewWgpuContext selects an adapter and creates a device that can configure surface.
func NewWgpuContext(instance gpu.Instance, surface gpu.Surface, config *WgpuSurfaceConfig, compositorGPU gpuiutil.Option[CompositorGPUHint]) (*WgpuContext, error) {
	return newWgpuContextWithOptions(instance, surface, config, compositorGPU, false)
}

// NewWgpuContextRejectingSoftware is NewWgpuContext skipping software (CPU) adapters.
func NewWgpuContextRejectingSoftware(instance gpu.Instance, surface gpu.Surface, config *WgpuSurfaceConfig, compositorGPU gpuiutil.Option[CompositorGPUHint]) (*WgpuContext, error) {
	return newWgpuContextWithOptions(instance, surface, config, compositorGPU, true)
}

func newWgpuContextWithOptions(
	instance gpu.Instance,
	surface gpu.Surface,
	config *WgpuSurfaceConfig,
	compositorGPU gpuiutil.Option[CompositorGPUHint],
	rejectSoftware bool,
) (*WgpuContext, error) {
	deviceIDFilter := wgpuContextDeviceIDFilter()

	selected, err := wgpuContextSelectAdapterAndDevice(
		instance,
		deviceIDFilter,
		surface,
		config,
		compositorGPU,
		rejectSoftware,
	)
	if err != nil {
		return nil, err
	}

	return wgpuContextFromNativeAdapterAndDevice(
		instance,
		selected.adapter,
		selected.device,
		selected.queue,
		selected.dualSourceBlending,
		selected.colorTextureFormat,
	), nil
}

type headlessContext struct {
	context      *WgpuContext
	targetFormat gpu.TextureFormat
}

var headless struct {
	mu      sync.Mutex
	created gpuiutil.Option[headlessContext]
}

func wgpuContextNewHeadless() (*WgpuContext, gpu.TextureFormat, error) {
	headless.mu.Lock()
	defer headless.mu.Unlock()
	if cached, ok := headless.created.Get(); ok && !cached.context.DeviceLost() {
		context := *cached.context
		return &context, cached.targetFormat, nil
	}
	context, format, err := wgpuContextCreateHeadless()
	if err != nil {
		return nil, 0, err
	}
	cached := *context
	headless.created = gpuiutil.Some(headlessContext{context: &cached, targetFormat: format})
	return context, format, nil
}

type selectedDevice struct {
	adapter            gpu.Adapter
	device             gpu.Device
	queue              gpu.Queue
	dualSourceBlending bool
	colorTextureFormat gpu.TextureFormat
}

func wgpuContextCreateHeadless() (*WgpuContext, gpu.TextureFormat, error) {
	instance := WgpuContextInstance(gpuiutil.None[gpui.RawWindowHandle]())
	deviceIDFilter := wgpuContextDeviceIDFilter()
	selected, targetFormat, err := func() (selectedDevice, gpu.TextureFormat, error) {
		adapters := instance.EnumerateAdapters(gpu.BackendsAll)
		wgpuContextSortAdapters(adapters, deviceIDFilter, gpuiutil.None[CompositorGPUHint]())

		for i, adapter := range adapters {
			adapterInfo := adapter.GetInfo()
			targetFormat, ok := wgpuContextHeadlessTargetFormat(adapter)
			if !ok {
				slog.Warn(fmt.Sprintf("Adapter %q has no supported headless render target format", adapterInfo.Name))
				continue
			}

			device, queue, dualSourceBlending, colorTextureFormat, err := wgpuContextCreateDevice(adapter)
			if err != nil {
				slog.Warn(fmt.Sprintf("Failed to create a headless device for adapter %q: %v", adapterInfo.Name, err))
				continue
			}
			wgpuContextReleaseAdaptersExcept(adapters, i)
			return selectedDevice{adapter, device, queue, dualSourceBlending, colorTextureFormat}, targetFormat, nil
		}

		wgpuContextReleaseAdaptersExcept(adapters, -1)
		return selectedDevice{}, 0, errors.New("No usable headless GPU adapter found")
	}()
	if err != nil {
		instance.Release()
		return nil, 0, err
	}

	return wgpuContextFromNativeAdapterAndDevice(
		instance,
		selected.adapter,
		selected.device,
		selected.queue,
		selected.dualSourceBlending,
		selected.colorTextureFormat,
	), targetFormat, nil
}

func wgpuContextHeadlessTargetFormat(adapter gpu.Adapter) (gpu.TextureFormat, bool) {
	requiredUsages := gpu.TextureUsagesRenderAttachment | gpu.TextureUsagesCopySrc
	for _, format := range []gpu.TextureFormat{gpu.TextureFormatRgba8Unorm, gpu.TextureFormatBgra8Unorm} {
		if adapter.GetTextureFormatFeatures(format).AllowedUsages.Contains(requiredUsages) {
			return format, true
		}
	}
	return 0, false
}

func wgpuContextFromNativeAdapterAndDevice(
	instance gpu.Instance,
	adapter gpu.Adapter,
	device gpu.Device,
	queue gpu.Queue,
	dualSourceBlending bool,
	colorTextureFormat gpu.TextureFormat,
) *WgpuContext {
	errors := installDeviceErrorState(device)

	slog.Info(fmt.Sprintf("Selected GPU adapter: %q (%s)", adapter.GetInfo().Name, adapter.GetInfo().Backend))
	backend := WgpuBackendNativeOf(adapter.GetInfo().Backend)

	return &WgpuContext{
		Instance:           instance,
		Adapter:            adapter,
		Device:             device,
		Queue:              queue,
		backend:            backend,
		dualSourceBlending: dualSourceBlending,
		colorTextureFormat: colorTextureFormat,
		errors:             errors,
	}
}

func wgpuContextDeviceIDFilter() gpuiutil.Option[uint32] {
	value, ok := os.LookupEnv("ZED_DEVICE_ID")
	if !ok {
		return gpuiutil.None[uint32]()
	}
	id, err := parsePCIID(value)
	if err != nil {
		gpuiutil.LogErr(fmt.Errorf("Failed to parse device ID from `ZED_DEVICE_ID` environment variable: %w", err))
		return gpuiutil.None[uint32]()
	}
	return gpuiutil.Some(id)
}

// SupportsTransientAttachments reports whether intermediate render targets may use the
// TRANSIENT_ATTACHMENT texture usage.
func (c *WgpuContext) SupportsTransientAttachments() bool {
	if c.backend.Kind == WgpuBackendBrowserWebGpu {
		return browserSupportsTransientAttachments()
	}
	return nativeSupportsTransientAttachments
}

func wgpuContextCreateDevice(adapter gpu.Adapter) (gpu.Device, gpu.Queue, bool, gpu.TextureFormat, error) {
	dualSourceBlending := adapter.Features().Contains(gpu.FeaturesDualSourceBlending)

	var requiredFeatures gpu.Features
	if dualSourceBlending {
		requiredFeatures |= gpu.FeaturesDualSourceBlending
	} else {
		slog.Warn("Dual-source blending not available on this GPU. Subpixel text antialiasing will be disabled.")
	}

	colorAtlasTextureFormat, err := wgpuContextSelectColorTextureFormat(adapter)
	if err != nil {
		return nil, nil, false, 0, err
	}
	baseLimits := gpu.DownlevelDefaults()
	if wgpuContextRequiresTextureInstanceData(adapter) {
		baseLimits = gpu.DownlevelWebGL2Defaults()
	}
	requiredLimits := baseLimits.UsingResolution(adapter.Limits()).UsingAlignment(adapter.Limits())

	device, queue, err := adapter.RequestDevice(&gpu.DeviceDescriptor{
		Label:            "gpui_device",
		RequiredFeatures: requiredFeatures,
		RequiredLimits:   requiredLimits,
	})
	if err != nil {
		return nil, nil, false, 0, fmt.Errorf("Failed to create wgpu device: %w", err)
	}

	return device, queue, dualSourceBlending, colorAtlasTextureFormat, nil
}

// WgpuContextInstance creates the native instance (Vulkan and GL backends). wgpu-native takes
// the display connection from each surface, so display is unused.
func WgpuContextInstance(display gpuiutil.Option[gpui.RawWindowHandle]) gpu.Instance {
	return createInstance(gpu.InstanceDescriptor{Backends: gpu.BackendsVulkan | gpu.BackendsGl})
}

// CheckCompatibleWithSurface fails when the context's adapter cannot present to surface.
func (c *WgpuContext) CheckCompatibleWithSurface(surface gpu.Surface) error {
	caps := surface.GetCapabilities(c.Adapter)
	if len(caps.Formats) == 0 {
		info := c.Adapter.GetInfo()
		return fmt.Errorf(
			"Adapter %q (backend=%s, device=0x%04x) is not compatible with the display surface for this window.",
			info.Name,
			info.Backend,
			info.Device,
		)
	}
	return nil
}

func wgpuContextSelectAdapterAndDevice(
	instance gpu.Instance,
	deviceIDFilter gpuiutil.Option[uint32],
	surface gpu.Surface,
	config *WgpuSurfaceConfig,
	compositorGPU gpuiutil.Option[CompositorGPUHint],
	rejectSoftware bool,
) (selectedDevice, error) {
	adapters := instance.EnumerateAdapters(gpu.BackendsAll)

	if len(adapters) == 0 {
		return selectedDevice{}, errors.New("No GPU adapters found")
	}

	if deviceID, ok := deviceIDFilter.Get(); ok {
		slog.Info(fmt.Sprintf("ZED_DEVICE_ID filter: 0x%04x", deviceID))
	}

	wgpuContextSortAdapters(adapters, deviceIDFilter, compositorGPU)

	slog.Info(fmt.Sprintf("Found %d GPU adapter(s):", len(adapters)))
	for _, adapter := range adapters {
		info := adapter.GetInfo()
		slog.Info(fmt.Sprintf(
			"  - %s (vendor=0x%04x, device=0x%04x, backend=%s, type=%s)",
			info.Name,
			info.Vendor,
			info.Device,
			info.Backend,
			info.DeviceType,
		))
	}

	for i, adapter := range adapters {
		info := adapter.GetInfo()

		if rejectSoftware && info.DeviceType == gpu.DeviceTypeCpu {
			slog.Info(fmt.Sprintf("Skipping software renderer: %s (%s)", info.Name, info.Backend))
			continue
		}

		slog.Info(fmt.Sprintf("Testing adapter: %s (%s)...", info.Name, info.Backend))

		device, queue, dualSourceBlending, colorAtlasTextureFormat, err := wgpuContextTryAdapterWithSurface(adapter, surface, config)
		if err != nil {
			slog.Info(fmt.Sprintf("  Adapter %s (%s) failed: %v, trying next...", info.Name, info.Backend, err))
			continue
		}
		slog.Info(fmt.Sprintf("Selected GPU (passed configuration test): %s (%s)", info.Name, info.Backend))
		wgpuContextReleaseAdaptersExcept(adapters, i)
		return selectedDevice{adapter, device, queue, dualSourceBlending, colorAtlasTextureFormat}, nil
	}

	wgpuContextReleaseAdaptersExcept(adapters, -1)
	return selectedDevice{}, errors.New("No GPU adapter found that can configure the display surface")
}

func wgpuContextReleaseAdaptersExcept(adapters []gpu.Adapter, keep int) {
	for i, adapter := range adapters {
		if i != keep {
			adapter.Release()
		}
	}
}

func wgpuContextSortAdapters(adapters []gpu.Adapter, deviceIDFilter gpuiutil.Option[uint32], compositorGPU gpuiutil.Option[CompositorGPUHint]) {
	key := func(adapter gpu.Adapter) [4]uint8 {
		info := adapter.GetInfo()
		deviceKnown := info.Device != 0
		var userOverride uint8 = 1
		if id, ok := deviceIDFilter.Get(); ok && deviceKnown && info.Device == id {
			userOverride = 0
		}
		var compositorMatch uint8 = 1
		if hint, ok := compositorGPU.Get(); ok && deviceKnown && info.Vendor == hint.VendorID && info.Device == hint.DeviceID {
			compositorMatch = 0
		}
		var typePriority uint8
		switch info.DeviceType {
		case gpu.DeviceTypeDiscreteGpu:
			typePriority = 0
		case gpu.DeviceTypeIntegratedGpu:
			typePriority = 1
		case gpu.DeviceTypeOther:
			typePriority = 2
		case gpu.DeviceTypeVirtualGpu:
			typePriority = 3
		case gpu.DeviceTypeCpu:
			typePriority = 4
		}
		var backendPriority uint8 = 1
		switch info.Backend {
		case gpu.BackendVulkan, gpu.BackendMetal, gpu.BackendDx12:
			backendPriority = 0
		}
		return [4]uint8{userOverride, compositorMatch, typePriority, backendPriority}
	}
	keys := make(map[gpu.Adapter][4]uint8, len(adapters))
	for _, adapter := range adapters {
		keys[adapter] = key(adapter)
	}
	slices.SortStableFunc(adapters, func(a, b gpu.Adapter) int {
		ka, kb := keys[a], keys[b]
		return slices.Compare(ka[:], kb[:])
	})
}

func wgpuContextTryAdapterWithSurface(adapter gpu.Adapter, surface gpu.Surface, config *WgpuSurfaceConfig) (gpu.Device, gpu.Queue, bool, gpu.TextureFormat, error) {
	caps := surface.GetCapabilities(adapter)
	if len(caps.Formats) == 0 {
		return nil, nil, false, 0, errors.New("no compatible surface formats")
	}
	if len(caps.AlphaModes) == 0 {
		return nil, nil, false, 0, errors.New("no compatible alpha modes")
	}

	device, queue, dualSourceBlending, colorAtlasTextureFormat, err := wgpuContextCreateDevice(adapter)
	if err != nil {
		return nil, nil, false, 0, err
	}
	surfaceConfig, _, _, err := wgpuRendererSurfaceConfiguration(surface, adapter, device, config)
	if err != nil {
		queue.Release()
		device.Release()
		return nil, nil, false, 0, err
	}
	device.PushErrorScope(gpu.ErrorFilterValidation)

	surface.Configure(device, &surfaceConfig)

	if scopeErr := device.PopErrorScope(); scopeErr != nil {
		queue.Release()
		device.Release()
		return nil, nil, false, 0, fmt.Errorf("surface configuration failed: %s", scopeErr.Error())
	}

	return device, queue, dualSourceBlending, colorAtlasTextureFormat, nil
}

func wgpuContextSelectColorTextureFormat(adapter gpu.Adapter) (gpu.TextureFormat, error) {
	requiredUsages := gpu.TextureUsagesTextureBinding | gpu.TextureUsagesCopyDst
	bgraFeatures := adapter.GetTextureFormatFeatures(gpu.TextureFormatBgra8Unorm)
	rgbaFeatures := adapter.GetTextureFormatFeatures(gpu.TextureFormatRgba8Unorm)
	if webGlPrefersRgba && adapter.GetInfo().Backend == gpu.BackendGl && rgbaFeatures.AllowedUsages.Contains(requiredUsages) {
		return gpu.TextureFormatRgba8Unorm, nil
	}
	if bgraFeatures.AllowedUsages.Contains(requiredUsages) {
		return gpu.TextureFormatBgra8Unorm, nil
	}
	if rgbaFeatures.AllowedUsages.Contains(requiredUsages) {
		info := adapter.GetInfo()
		slog.Warn(fmt.Sprintf(
			"Adapter %s (%s) does not support Bgra8Unorm atlas textures with usages %s; falling back to Rgba8Unorm atlas textures.",
			info.Name,
			info.Backend,
			requiredUsages,
		))
		return gpu.TextureFormatRgba8Unorm, nil
	}

	info := adapter.GetInfo()
	return 0, fmt.Errorf(
		"Adapter %s (%s, device=0x%04x) does not support a usable color atlas texture format with usages %s. Bgra8Unorm allowed usages: %s; Rgba8Unorm allowed usages: %s.",
		info.Name,
		info.Backend,
		info.Device,
		requiredUsages,
		bgraFeatures.AllowedUsages,
		rgbaFeatures.AllowedUsages,
	)
}

// Backend returns the backend the context runs on.
func (c *WgpuContext) Backend() WgpuBackend {
	return c.backend
}

// UsesTextureInstanceData reports whether instance data goes through a texture because the
// adapter lacks storage buffers in vertex and fragment shaders.
func (c *WgpuContext) UsesTextureInstanceData() bool {
	return wgpuContextRequiresTextureInstanceData(c.Adapter)
}

func wgpuContextRequiresTextureInstanceData(adapter gpu.Adapter) bool {
	if adapter.GetInfo().Backend != gpu.BackendGl {
		return false
	}
	return !adapter.GetDownlevelCapabilities().Flags.Contains(gpu.DownlevelFlagsVertexStorage | gpu.DownlevelFlagsFragmentStorage)
}

// SupportsDualSourceBlending reports whether the device has dual-source blending.
func (c *WgpuContext) SupportsDualSourceBlending() bool {
	return c.dualSourceBlending
}

// ColorTextureFormat is the format of colour atlas textures.
func (c *WgpuContext) ColorTextureFormat() gpu.TextureFormat {
	return c.colorTextureFormat
}

// DeviceLost reports whether the GPU device was lost (e.g., due to driver crash,
// suspend/resume). When this returns true, the context should be recreated.
func (c *WgpuContext) DeviceLost() bool {
	return c.errors.DeviceLost()
}

// Errors returns the device-wide error state, shared with renderers on this device.
func (c *WgpuContext) Errors() *DeviceErrorState {
	return c.errors
}

func parsePCIID(id string) (uint32, error) {
	id = strings.TrimSpace(id)

	if strings.HasPrefix(id, "0x") || strings.HasPrefix(id, "0X") {
		id = id[2:]
	}
	isHexString := true
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			isHexString = false
			break
		}
	}
	is4Chars := len(id) == 4
	if !(is4Chars && isHexString) {
		return 0, errors.New("Expected a 4 digit PCI ID in hexadecimal format")
	}

	value, err := strconv.ParseUint(id, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parsing PCI ID as hex: %w", err)
	}
	return uint32(value), nil
}
