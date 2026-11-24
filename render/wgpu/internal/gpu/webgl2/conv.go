//go:build js && wasm

package webgl2

import (
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

type textureFormatDesc struct {
	internal      uint32
	external      uint32
	dataType      uint32
	bytesPerTexel uint32
	swizzleBgra   bool
	filterable    bool
	depth         bool
}

func describeTextureFormat(format gpu.TextureFormat) (textureFormatDesc, bool) {
	switch format {
	case gpu.TextureFormatR8Unorm:
		return textureFormatDesc{internal: glR8, external: glRed, dataType: glUnsignedByte, bytesPerTexel: 1, filterable: true}, true
	case gpu.TextureFormatRgba8Unorm:
		return textureFormatDesc{internal: glRGBA8, external: glRGBA, dataType: glUnsignedByte, bytesPerTexel: 4, filterable: true}, true
	case gpu.TextureFormatRgba8UnormSrgb:
		return textureFormatDesc{internal: glSRGB8Alpha8, external: glRGBA, dataType: glUnsignedByte, bytesPerTexel: 4, filterable: true}, true
	case gpu.TextureFormatBgra8Unorm:
		return textureFormatDesc{internal: glRGBA8, external: glRGBA, dataType: glUnsignedByte, bytesPerTexel: 4, swizzleBgra: true, filterable: true}, true
	case gpu.TextureFormatBgra8UnormSrgb:
		return textureFormatDesc{internal: glSRGB8Alpha8, external: glRGBA, dataType: glUnsignedByte, bytesPerTexel: 4, swizzleBgra: true, filterable: true}, true
	case gpu.TextureFormatRgba32Uint:
		return textureFormatDesc{internal: glRGBA32UI, external: glRGBAInteger, dataType: glUnsignedInt, bytesPerTexel: 16}, true
	case gpu.TextureFormatDepth32Float:
		return textureFormatDesc{internal: glDepthComponent32F, external: glDepthComponent, dataType: glFloat, bytesPerTexel: 4, depth: true}, true
	}
	return textureFormatDesc{}, false
}

func swizzleRB(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	for i := 0; i+3 < len(out); i += 4 {
		out[i], out[i+2] = out[i+2], out[i]
	}
	return out
}

func mapFilterModes(min, mag, mip gpu.FilterMode) (uint32, uint32) {
	magFilter := uint32(glNearest)
	if mag == gpu.FilterModeLinear {
		magFilter = glLinear
	}
	minLinear := min == gpu.FilterModeLinear
	mipLinear := mip == gpu.FilterModeLinear
	var minFilter uint32
	switch {
	case !minLinear && !mipLinear:
		minFilter = glNearestMipmapNearest
	case !minLinear && mipLinear:
		minFilter = glNearestMipmapLinear
	case minLinear && !mipLinear:
		minFilter = glLinearMipmapNearest
	default:
		minFilter = glLinearMipmapLinear
	}
	return minFilter, magFilter
}

func mapAddressMode(mode gpu.AddressMode) uint32 {
	switch mode {
	case gpu.AddressModeRepeat:
		return glRepeat
	case gpu.AddressModeMirrorRepeat:
		return glMirroredRepeat
	}
	return glClampToEdge
}

func mapCompareFunc(f gpu.CompareFunction) uint32 {
	switch f {
	case gpu.CompareFunctionNever:
		return glNever
	case gpu.CompareFunctionLess:
		return glLess
	case gpu.CompareFunctionLessEqual:
		return glLequal
	case gpu.CompareFunctionEqual:
		return glEqual
	case gpu.CompareFunctionGreaterEqual:
		return glGequal
	case gpu.CompareFunctionGreater:
		return glGreater
	case gpu.CompareFunctionNotEqual:
		return glNotequal
	}
	return glAlways
}

func mapPrimitiveTopology(t gpu.PrimitiveTopology) uint32 {
	switch t {
	case gpu.PrimitiveTopologyPointList:
		return glPoints
	case gpu.PrimitiveTopologyLineList:
		return glLines
	case gpu.PrimitiveTopologyLineStrip:
		return glLineStrip
	case gpu.PrimitiveTopologyTriangleStrip:
		return glTriangleStrip
	}
	return glTriangles
}

type primitiveState struct {
	frontFace uint32
	cullFace  uint32
}

func mapPrimitiveState(state gpu.PrimitiveState) primitiveState {
	frontFace := uint32(glCW)
	if state.FrontFace == gpu.FrontFaceCw {
		frontFace = glCCW
	}
	var cull uint32
	switch state.CullMode {
	case gpu.CullModeFront:
		cull = glFront
	case gpu.CullModeBack:
		cull = glBack
	}
	return primitiveState{frontFace: frontFace, cullFace: cull}
}

func mapBlendFactor(f gpu.BlendFactor) uint32 {
	switch f {
	case gpu.BlendFactorZero:
		return glZero
	case gpu.BlendFactorOne:
		return glOne
	case gpu.BlendFactorSrc:
		return glSrcColor
	case gpu.BlendFactorOneMinusSrc:
		return glOneMinusSrcColor
	case gpu.BlendFactorDst:
		return glDstColor
	case gpu.BlendFactorOneMinusDst:
		return glOneMinusDstColor
	case gpu.BlendFactorSrcAlpha:
		return glSrcAlpha
	case gpu.BlendFactorOneMinusSrcAlpha:
		return glOneMinusSrcAlpha
	case gpu.BlendFactorDstAlpha:
		return glDstAlpha
	case gpu.BlendFactorOneMinusDstAlpha:
		return glOneMinusDstAlpha
	}
	return glOne
}

func mapBlendOperation(op gpu.BlendOperation) uint32 {
	switch op {
	case gpu.BlendOperationSubtract:
		return glFuncSubtract
	case gpu.BlendOperationReverseSubtract:
		return glFuncReverseSubtract
	case gpu.BlendOperationMin:
		return glMin
	case gpu.BlendOperationMax:
		return glMax
	}
	return glFuncAdd
}

type blendDesc struct {
	srcColor, dstColor, srcAlpha, dstAlpha uint32
	opColor, opAlpha                       uint32
}

type colorTargetDesc struct {
	mask  gpu.ColorWrites
	blend *blendDesc
}

func (a colorTargetDesc) equal(b colorTargetDesc) bool {
	if a.mask != b.mask || (a.blend == nil) != (b.blend == nil) {
		return false
	}
	return a.blend == nil || *a.blend == *b.blend
}

func mapColorTarget(target gpu.ColorTargetState) colorTargetDesc {
	desc := colorTargetDesc{mask: target.WriteMask}
	if target.Blend != nil {
		desc.blend = &blendDesc{
			srcColor: mapBlendFactor(target.Blend.Color.SrcFactor),
			dstColor: mapBlendFactor(target.Blend.Color.DstFactor),
			srcAlpha: mapBlendFactor(target.Blend.Alpha.SrcFactor),
			dstAlpha: mapBlendFactor(target.Blend.Alpha.DstFactor),
			opColor:  mapBlendOperation(target.Blend.Color.Operation),
			opAlpha:  mapBlendOperation(target.Blend.Alpha.Operation),
		}
	}
	return desc
}

type depthState struct {
	function uint32
	mask     bool
}

func mapDepth(state *gpu.DepthStencilState) *depthState {
	if state == nil {
		return nil
	}
	return &depthState{function: mapCompareFunc(state.DepthCompare), mask: state.DepthWriteEnabled}
}
