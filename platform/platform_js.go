//go:build js && wasm

package platform

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/web"
)

// WebBackendPreference selects the browser graphics back end.
type WebBackendPreference = web.WebBackendPreference

// ApplicationWithWebBackend returns a web app that uses backendPreference.
func ApplicationWithWebBackend(backendPreference WebBackendPreference) *gpui.Application {
	return gpui.ApplicationWithPlatform(web.NewWebPlatformWithBackend(true, backendPreference))
}

// SingleThreadedWeb returns a single-threaded web app, unlike Application.
func SingleThreadedWeb() *gpui.Application {
	return gpui.ApplicationWithPlatform(web.NewWebPlatform(false))
}

// WebInit initializes logging for the web platform; Go already reports panics to the browser
// console. Call it before running the application.
func WebInit() {
	web.InitLogging()
}

func currentPlatform(headless bool) (Platform, error) {
	return web.NewWebPlatformWithBackend(true, web.WebBackendPreferenceAuto), nil
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
