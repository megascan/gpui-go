//go:build ios

package platform

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/ios"
)

func currentPlatform(headless bool) (Platform, error) {
	return ios.CurrentPlatform(headless), nil
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
