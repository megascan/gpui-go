//go:build linux

package platform

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux"
	"github.com/megascan/gpui-go/render/wgpu"
)

// Linux returns a Linux app that may switch among allowedModes.
//
// It starts windowed in the process's own environment, or headless if that names no allowed
// display server. Set another initial mode with gpui.Application.WithWindowing, and switch later
// with gpui.App.RequestWindowing.
func Linux(allowedModes gpui.WindowingModes) *gpui.Application {
	return gpui.ApplicationWithPlatform(linux.LinuxPlatform(allowedModes))
}

func currentPlatform(headless bool) (Platform, error) {
	return linux.CurrentPlatform(headless), nil
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	renderer, err := wgpu.NewWgpuHeadlessRenderer()
	if err != nil {
		return nil, err
	}
	return renderer, nil
}
