//go:build darwin && !ios

package platform

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/platform/macos"
)

func currentPlatform(headless bool) (Platform, error) {
	return macos.NewMacPlatform(headless), nil
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	renderer, err := apple.NewMetalHeadlessRenderer()
	if err != nil {
		return nil, err
	}
	return renderer, nil
}
