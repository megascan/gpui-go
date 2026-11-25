//go:build darwin

// Package mtl binds the Metal and QuartzCore (CAMetalLayer) API the Apple renderer uses, as thin
// typed wrappers over internal/apple/objc on purego without cgo.
package mtl

import (
	"fmt"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// MetalPath is the install path of Metal.framework.
const MetalPath = "/System/Library/Frameworks/Metal.framework/Metal"

// QuartzCorePath is the install path of QuartzCore.framework.
const QuartzCorePath = "/System/Library/Frameworks/QuartzCore.framework/QuartzCore"

var (
	loadOnce sync.Once
	loadErr  error

	mtlCreateSystemDefaultDevice func() objc.ID
	mtlCopyAllDevices            func() objc.ID

	msgSendIDUintUint   func(id objc.ID, sel objc.SEL, a objc.ID, b, c uint)
	msgSendPtrUintUint  func(id objc.ID, sel objc.SEL, a unsafe.Pointer, b, c uint)
	msgSendIDUint       func(id objc.ID, sel objc.SEL, a objc.ID, b uint)
	msgSendUint4        func(id objc.ID, sel objc.SEL, a, b, c, d uint)
	msgSendUint5        func(id objc.ID, sel objc.SEL, a, b, c, d, e uint)
	msgSendViewport     func(id objc.ID, sel objc.SEL, v Viewport)
	msgSendClearColor   func(id objc.ID, sel objc.SEL, c ClearColor)
	msgSendCGSize       func(id objc.ID, sel objc.SEL, s objc.CGSize)
	msgSendNSRange      func(id objc.ID, sel objc.SEL, r objc.NSRange)
	msgSendReplaceRgn   func(id objc.ID, sel objc.SEL, r Region, level uint, bytes unsafe.Pointer, bytesPerRow uint)
	msgSendGetBytes     func(id objc.ID, sel objc.SEL, bytes unsafe.Pointer, bytesPerRow uint, r Region, level uint)
	msgSendIDPtrRetID   func(id objc.ID, sel objc.SEL, a objc.ID, err unsafe.Pointer) objc.ID
	msgSendIDIDPtrRetID func(id objc.ID, sel objc.SEL, a, b objc.ID, err unsafe.Pointer) objc.ID
)

// Load opens the Objective-C runtime, Metal and QuartzCore and binds the entry points. It is safe
// to call repeatedly and from any thread.
func Load() error {
	loadOnce.Do(func() {
		if err := objc.Load(); err != nil {
			loadErr = err
			return
		}
		metal, err := purego.Dlopen(MetalPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("mtl: open %s: %w", MetalPath, err)
			return
		}
		if _, err := purego.Dlopen(QuartzCorePath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
			loadErr = fmt.Errorf("mtl: open %s: %w", QuartzCorePath, err)
			return
		}
		purego.RegisterLibFunc(&mtlCreateSystemDefaultDevice, metal, "MTLCreateSystemDefaultDevice")
		if _, err := purego.Dlsym(metal, "MTLCopyAllDevices"); err == nil {
			purego.RegisterLibFunc(&mtlCopyAllDevices, metal, "MTLCopyAllDevices")
		}
		for _, f := range []any{
			&msgSendIDUintUint, &msgSendPtrUintUint, &msgSendIDUint, &msgSendUint4, &msgSendUint5,
			&msgSendViewport, &msgSendClearColor, &msgSendCGSize, &msgSendNSRange,
			&msgSendReplaceRgn, &msgSendGetBytes, &msgSendIDPtrRetID, &msgSendIDIDPtrRetID,
		} {
			objc.BindMsgSend(f)
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

func send[T any](id objc.ID, sel string, args ...any) T {
	mustLoad()
	return objc.Send[T](id, objc.Sel(sel), args...)
}

func sendVoid(id objc.ID, sel string, args ...any) {
	mustLoad()
	objc.SendVoid(id, objc.Sel(sel), args...)
}

func newObject(class string) objc.ID {
	mustLoad()
	c := objc.GetClass(class)
	if c == 0 {
		panic("mtl: class " + class + " is not loaded")
	}
	return objc.SendClass[objc.ID](c, objc.Sel("new"))
}

func setLabel(id objc.ID, label string) {
	if label == "" {
		sendVoid(id, "setLabel:", objc.Nil)
		return
	}
	sendVoid(id, "setLabel:", objc.NSStringFromGo(label))
}

func nsError(err objc.ID) error {
	if e := objc.NSErrorFromID(err); e != nil {
		return e
	}
	return nil
}

// Retain increments the object's reference count.
func Retain[T ~uintptr](o T) T { return T(objc.Retain(objc.ID(o))) }

// Release decrements the object's reference count; a nil object is ignored.
func Release[T ~uintptr](o T) { objc.Release(objc.ID(o)) }

// Origin is MTLOrigin.
type Origin struct {
	_ structs.HostLayout
	X uint
	Y uint
	Z uint
}

// Size is MTLSize.
type Size struct {
	_      structs.HostLayout
	Width  uint
	Height uint
	Depth  uint
}

// Region is MTLRegion.
type Region struct {
	_      structs.HostLayout
	Origin Origin
	Size   Size
}

// ClearColor is MTLClearColor.
type ClearColor struct {
	_     structs.HostLayout
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

// Viewport is MTLViewport.
type Viewport struct {
	_       structs.HostLayout
	OriginX float64
	OriginY float64
	Width   float64
	Height  float64
	ZNear   float64
	ZFar    float64
}

// PixelFormat is MTLPixelFormat.
type PixelFormat uint

const (
	// PixelFormatInvalid is MTLPixelFormatInvalid.
	PixelFormatInvalid PixelFormat = 0
	// PixelFormatA8Unorm is MTLPixelFormatA8Unorm.
	PixelFormatA8Unorm PixelFormat = 1
	// PixelFormatR8Unorm is MTLPixelFormatR8Unorm.
	PixelFormatR8Unorm PixelFormat = 10
	// PixelFormatRG8Unorm is MTLPixelFormatRG8Unorm.
	PixelFormatRG8Unorm PixelFormat = 30
	// PixelFormatRGBA8Unorm is MTLPixelFormatRGBA8Unorm.
	PixelFormatRGBA8Unorm PixelFormat = 70
	// PixelFormatBGRA8Unorm is MTLPixelFormatBGRA8Unorm.
	PixelFormatBGRA8Unorm PixelFormat = 80
	// PixelFormatBGRA8UnormSRGB is MTLPixelFormatBGRA8Unorm_sRGB.
	PixelFormatBGRA8UnormSRGB PixelFormat = 81
	// PixelFormatRGBA16Float is MTLPixelFormatRGBA16Float.
	PixelFormatRGBA16Float PixelFormat = 115
	// PixelFormatDepth32Float is MTLPixelFormatDepth32Float.
	PixelFormatDepth32Float PixelFormat = 252
)

// TextureUsage is MTLTextureUsage.
type TextureUsage uint

const (
	// TextureUsageUnknown is MTLTextureUsageUnknown.
	TextureUsageUnknown TextureUsage = 0
	// TextureUsageShaderRead is MTLTextureUsageShaderRead.
	TextureUsageShaderRead TextureUsage = 1
	// TextureUsageShaderWrite is MTLTextureUsageShaderWrite.
	TextureUsageShaderWrite TextureUsage = 2
	// TextureUsageRenderTarget is MTLTextureUsageRenderTarget.
	TextureUsageRenderTarget TextureUsage = 4
)

// TextureType is MTLTextureType.
type TextureType uint

const (
	// TextureType2D is MTLTextureType2D.
	TextureType2D TextureType = 2
	// TextureType2DMultisample is MTLTextureType2DMultisample.
	TextureType2DMultisample TextureType = 4
)

// StorageMode is MTLStorageMode.
type StorageMode uint

const (
	// StorageModeShared is MTLStorageModeShared.
	StorageModeShared StorageMode = 0
	// StorageModeManaged is MTLStorageModeManaged.
	StorageModeManaged StorageMode = 1
	// StorageModePrivate is MTLStorageModePrivate.
	StorageModePrivate StorageMode = 2
	// StorageModeMemoryless is MTLStorageModeMemoryless.
	StorageModeMemoryless StorageMode = 3
)

// ResourceOptions is MTLResourceOptions.
type ResourceOptions uint

const (
	// ResourceCPUCacheModeDefaultCache is MTLResourceCPUCacheModeDefaultCache.
	ResourceCPUCacheModeDefaultCache ResourceOptions = 0
	// ResourceCPUCacheModeWriteCombined is MTLResourceCPUCacheModeWriteCombined.
	ResourceCPUCacheModeWriteCombined ResourceOptions = 1
	// ResourceStorageModeShared is MTLResourceStorageModeShared.
	ResourceStorageModeShared ResourceOptions = ResourceOptions(StorageModeShared) << 4
	// ResourceStorageModeManaged is MTLResourceStorageModeManaged.
	ResourceStorageModeManaged ResourceOptions = ResourceOptions(StorageModeManaged) << 4
	// ResourceStorageModePrivate is MTLResourceStorageModePrivate.
	ResourceStorageModePrivate ResourceOptions = ResourceOptions(StorageModePrivate) << 4
)

// LoadAction is MTLLoadAction.
type LoadAction uint

const (
	// LoadActionDontCare is MTLLoadActionDontCare.
	LoadActionDontCare LoadAction = 0
	// LoadActionLoad is MTLLoadActionLoad.
	LoadActionLoad LoadAction = 1
	// LoadActionClear is MTLLoadActionClear.
	LoadActionClear LoadAction = 2
)

// StoreAction is MTLStoreAction.
type StoreAction uint

const (
	// StoreActionDontCare is MTLStoreActionDontCare.
	StoreActionDontCare StoreAction = 0
	// StoreActionStore is MTLStoreActionStore.
	StoreActionStore StoreAction = 1
	// StoreActionMultisampleResolve is MTLStoreActionMultisampleResolve.
	StoreActionMultisampleResolve StoreAction = 2
)

// PrimitiveType is MTLPrimitiveType.
type PrimitiveType uint

const (
	// PrimitiveTypePoint is MTLPrimitiveTypePoint.
	PrimitiveTypePoint PrimitiveType = 0
	// PrimitiveTypeLine is MTLPrimitiveTypeLine.
	PrimitiveTypeLine PrimitiveType = 1
	// PrimitiveTypeLineStrip is MTLPrimitiveTypeLineStrip.
	PrimitiveTypeLineStrip PrimitiveType = 2
	// PrimitiveTypeTriangle is MTLPrimitiveTypeTriangle.
	PrimitiveTypeTriangle PrimitiveType = 3
	// PrimitiveTypeTriangleStrip is MTLPrimitiveTypeTriangleStrip.
	PrimitiveTypeTriangleStrip PrimitiveType = 4
)

// BlendFactor is MTLBlendFactor.
type BlendFactor uint

const (
	// BlendFactorZero is MTLBlendFactorZero.
	BlendFactorZero BlendFactor = 0
	// BlendFactorOne is MTLBlendFactorOne.
	BlendFactorOne BlendFactor = 1
	// BlendFactorSourceColor is MTLBlendFactorSourceColor.
	BlendFactorSourceColor BlendFactor = 2
	// BlendFactorOneMinusSourceColor is MTLBlendFactorOneMinusSourceColor.
	BlendFactorOneMinusSourceColor BlendFactor = 3
	// BlendFactorSourceAlpha is MTLBlendFactorSourceAlpha.
	BlendFactorSourceAlpha BlendFactor = 4
	// BlendFactorOneMinusSourceAlpha is MTLBlendFactorOneMinusSourceAlpha.
	BlendFactorOneMinusSourceAlpha BlendFactor = 5
	// BlendFactorDestinationColor is MTLBlendFactorDestinationColor.
	BlendFactorDestinationColor BlendFactor = 6
	// BlendFactorOneMinusDestinationColor is MTLBlendFactorOneMinusDestinationColor.
	BlendFactorOneMinusDestinationColor BlendFactor = 7
	// BlendFactorDestinationAlpha is MTLBlendFactorDestinationAlpha.
	BlendFactorDestinationAlpha BlendFactor = 8
	// BlendFactorOneMinusDestinationAlpha is MTLBlendFactorOneMinusDestinationAlpha.
	BlendFactorOneMinusDestinationAlpha BlendFactor = 9
)

// BlendOperation is MTLBlendOperation.
type BlendOperation uint

const (
	// BlendOperationAdd is MTLBlendOperationAdd.
	BlendOperationAdd BlendOperation = 0
	// BlendOperationSubtract is MTLBlendOperationSubtract.
	BlendOperationSubtract BlendOperation = 1
	// BlendOperationReverseSubtract is MTLBlendOperationReverseSubtract.
	BlendOperationReverseSubtract BlendOperation = 2
	// BlendOperationMin is MTLBlendOperationMin.
	BlendOperationMin BlendOperation = 3
	// BlendOperationMax is MTLBlendOperationMax.
	BlendOperationMax BlendOperation = 4
)

// CompareFunction is MTLCompareFunction.
type CompareFunction uint

const (
	// CompareFunctionNever is MTLCompareFunctionNever.
	CompareFunctionNever CompareFunction = 0
	// CompareFunctionLess is MTLCompareFunctionLess.
	CompareFunctionLess CompareFunction = 1
	// CompareFunctionEqual is MTLCompareFunctionEqual.
	CompareFunctionEqual CompareFunction = 2
	// CompareFunctionLessEqual is MTLCompareFunctionLessEqual.
	CompareFunctionLessEqual CompareFunction = 3
	// CompareFunctionGreater is MTLCompareFunctionGreater.
	CompareFunctionGreater CompareFunction = 4
	// CompareFunctionNotEqual is MTLCompareFunctionNotEqual.
	CompareFunctionNotEqual CompareFunction = 5
	// CompareFunctionGreaterEqual is MTLCompareFunctionGreaterEqual.
	CompareFunctionGreaterEqual CompareFunction = 6
	// CompareFunctionAlways is MTLCompareFunctionAlways.
	CompareFunctionAlways CompareFunction = 7
)

// SamplerMinMagFilter is MTLSamplerMinMagFilter.
type SamplerMinMagFilter uint

const (
	// SamplerMinMagFilterNearest is MTLSamplerMinMagFilterNearest.
	SamplerMinMagFilterNearest SamplerMinMagFilter = 0
	// SamplerMinMagFilterLinear is MTLSamplerMinMagFilterLinear.
	SamplerMinMagFilterLinear SamplerMinMagFilter = 1
)

// SamplerAddressMode is MTLSamplerAddressMode.
type SamplerAddressMode uint

const (
	// SamplerAddressModeClampToEdge is MTLSamplerAddressModeClampToEdge.
	SamplerAddressModeClampToEdge SamplerAddressMode = 0
	// SamplerAddressModeRepeat is MTLSamplerAddressModeRepeat.
	SamplerAddressModeRepeat SamplerAddressMode = 2
)

// GPUFamily is MTLGPUFamily.
type GPUFamily int

const (
	// GPUFamilyApple1 is MTLGPUFamilyApple1.
	GPUFamilyApple1 GPUFamily = 1001
	// GPUFamilyApple7 is MTLGPUFamilyApple7.
	GPUFamilyApple7 GPUFamily = 1007
	// GPUFamilyMac2 is MTLGPUFamilyMac2.
	GPUFamilyMac2 GPUFamily = 2002
	// GPUFamilyCommon1 is MTLGPUFamilyCommon1.
	GPUFamilyCommon1 GPUFamily = 3001
)

// CommandBufferStatus is MTLCommandBufferStatus.
type CommandBufferStatus uint

const (
	// CommandBufferStatusNotEnqueued is MTLCommandBufferStatusNotEnqueued.
	CommandBufferStatusNotEnqueued CommandBufferStatus = 0
	// CommandBufferStatusEnqueued is MTLCommandBufferStatusEnqueued.
	CommandBufferStatusEnqueued CommandBufferStatus = 1
	// CommandBufferStatusCommitted is MTLCommandBufferStatusCommitted.
	CommandBufferStatusCommitted CommandBufferStatus = 2
	// CommandBufferStatusScheduled is MTLCommandBufferStatusScheduled.
	CommandBufferStatusScheduled CommandBufferStatus = 3
	// CommandBufferStatusCompleted is MTLCommandBufferStatusCompleted.
	CommandBufferStatusCompleted CommandBufferStatus = 4
	// CommandBufferStatusError is MTLCommandBufferStatusError.
	CommandBufferStatusError CommandBufferStatus = 5
)
