//go:build linux

package common

import "github.com/megascan/gpui-go"

func newTextSystem() gpui.PlatformTextSystem {
	return gpui.NewNoopTextSystem()
}
