//go:build !linux && !windows && !js

package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

const usage = "commands: ls | create [message] | open | close | quit"

func newApplication() *gpui.Application {
	return platform.Application()
}

func displayEnvironment(string) (gpui.GraphicalEnvironment, error) {
	return gpui.GraphicalEnvironmentDetect(), nil
}
