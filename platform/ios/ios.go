//go:build ios

package ios

import (
	"sync"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
)

const coreVideoPath = "/System/Library/Frameworks/CoreVideo.framework/CoreVideo"

var (
	coreVideoOnce          sync.Once
	cvPixelBufferGetWidth  func(buffer uintptr) uintptr
	cvPixelBufferGetHeight func(buffer uintptr) uintptr
)

func init() {
	gpui.RegisterSurfaceSourceSize(surfaceSourceSize)
}

func surfaceSourceSize(surface uintptr) gpui.Size[gpui.DevicePixels] {
	coreVideoOnce.Do(func() {
		lib, err := purego.Dlopen(coreVideoPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			panic(err)
		}
		purego.RegisterLibFunc(&cvPixelBufferGetWidth, lib, "CVPixelBufferGetWidth")
		purego.RegisterLibFunc(&cvPixelBufferGetHeight, lib, "CVPixelBufferGetHeight")
	})
	return gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(int32(cvPixelBufferGetWidth(surface))),
		Height: gpui.DevicePixels(int32(cvPixelBufferGetHeight(surface))),
	}
}

// CurrentPlatform returns the native platform implementation for iOS.
func CurrentPlatform(headless bool) gpui.Platform {
	return NewIosPlatform()
}
