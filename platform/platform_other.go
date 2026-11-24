//go:build !linux && !darwin && !windows && !js

package platform

import "github.com/megascan/gpui-go"

func currentPlatform(headless bool) (Platform, error) {
	return nil, errPlatformNotPorted("current")
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
