//go:build darwin && !ios

package platform

import "github.com/megascan/gpui-go"

func currentPlatform(headless bool) (Platform, error) {
	return nil, errPlatformNotPorted("macOS")
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, errPlatformNotPorted("macOS")
}
