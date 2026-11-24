//go:build js && wasm

package platform

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu"
)

// WebBackendPreference selects the browser graphics back end.
type WebBackendPreference = wgpu.WebBackendPreference

// ApplicationWithWebBackend returns a web app that uses backendPreference.
func ApplicationWithWebBackend(backendPreference WebBackendPreference) *gpui.Application {
	panic(errPlatformNotPorted("web"))
}

// SingleThreadedWeb returns a single-threaded web app, unlike Application.
func SingleThreadedWeb() *gpui.Application {
	panic(errPlatformNotPorted("web"))
}

// WebInit initializes panic hooks and logging for the web platform. Call it before running the
// application.
func WebInit() {
	panic(errPlatformNotPorted("web"))
}

func currentPlatform(headless bool) (Platform, error) {
	return nil, errPlatformNotPorted("web")
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
