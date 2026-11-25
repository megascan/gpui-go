// Package platform ports gpui_platform: it picks the Platform for the current OS so apps don't
// need build constraints, and locks the main goroutine to the main OS thread.
package platform

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/megascan/gpui-go"
)

func init() {
	runtime.LockOSThread()
}

// Platform is the interface every platform back end implements.
type Platform = gpui.Platform

// BackgroundExecutor returns a background executor for the current platform.
func BackgroundExecutor() gpui.BackgroundExecutor {
	return CurrentPlatform(true).BackgroundExecutor()
}

// Application returns an app on the current platform.
func Application() *gpui.Application {
	return gpui.ApplicationWithPlatform(CurrentPlatform(false))
}

// Headless returns a headless app on the current platform.
func Headless() *gpui.Application {
	return gpui.ApplicationWithPlatform(CurrentPlatform(true))
}

// CurrentPlatform returns the default Platform for the current OS. It panics if the platform
// cannot be initialized.
func CurrentPlatform(headless bool) Platform {
	platform, err := currentPlatform(headless)
	if err != nil {
		panic(err)
	}
	return platform
}

func errPlatformNotPorted(name string) error {
	return fmt.Errorf("gpui: the %s platform is not ported yet", name)
}

var benchTextSystem = sync.OnceValue(func() gpui.PlatformTextSystem {
	return CurrentPlatform(true).TextSystem()
})

// BenchTextSystem returns the current platform's text system for benchmark contexts, built
// once per process and shared by every context.
//
// Building a whole platform for each benchmark context only to take its text system would leak
// the platform's goroutines, which slows every later benchmark in the process.
func BenchTextSystem() gpui.PlatformTextSystem {
	return benchTextSystem()
}

// CurrentHeadlessRenderer returns a new headless renderer for the current platform, or nil if
// it has none.
func CurrentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return currentHeadlessRenderer()
}
