//go:build windows

package platform

import "github.com/megascan/gpui-go"

func currentPlatform(headless bool) (Platform, error) {
	return nil, errPlatformNotPorted("Windows")
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
