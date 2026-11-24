//go:build windows

package windows

import (
	"sync"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/dwrite"
	"github.com/megascan/gpui-go/internal/win/win32"
)

// Window messages gpui posts to its own windows (events.rs).
const (
	WM_GPUI_CURSOR_STYLE_CHANGED           uint32 = win32.WM_USER + 1
	WM_GPUI_CLOSE_ONE_WINDOW               uint32 = win32.WM_USER + 2
	WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD uint32 = win32.WM_USER + 3
	WM_GPUI_DOCK_MENU_ACTION               uint32 = win32.WM_USER + 4
	WM_GPUI_FORCE_UPDATE_WINDOW            uint32 = win32.WM_USER + 5
	WM_GPUI_KEYBOARD_LAYOUT_CHANGED        uint32 = win32.WM_USER + 6
	WM_GPUI_GPU_DEVICE_LOST                uint32 = win32.WM_USER + 7
	WM_GPUI_DISPLAYS_CHANGED               uint32 = win32.WM_USER + 10
)

type directXFontInfo struct {
	gammaRatios               [4]float32
	grayscaleEnhancedContrast float32
	subpixelEnhancedContrast  float32
	isBgr                     bool
}

var cachedFontInfo = sync.OnceValue(func() *directXFontInfo {
	factory, err := dwrite.DWriteCreateFactory(dwrite.DWRITE_FACTORY_TYPE_SHARED)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer factory.Release()
	base, err := factory.CreateRenderingParams()
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer base.Release()
	renderParams, err := base.CastRenderingParams1()
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer renderParams.Release()
	return &directXFontInfo{
		gammaRatios:               gpui.GetGammaCorrectionRatios(renderParams.GetGamma()),
		grayscaleEnhancedContrast: renderParams.GetGrayscaleEnhancedContrast(),
		subpixelEnhancedContrast:  renderParams.GetEnhancedContrast(),
		isBgr:                     renderParams.GetPixelGeometry() == dwrite.DWRITE_PIXEL_GEOMETRY_BGR,
	}
})

func directXRendererGetFontInfo() *directXFontInfo {
	return cachedFontInfo()
}
