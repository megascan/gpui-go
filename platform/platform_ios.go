//go:build ios

package platform

import "github.com/megascan/gpui-go"

func currentPlatform(headless bool) (Platform, error) {
	return nil, errPlatformNotPorted("iOS")
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
