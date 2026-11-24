//go:build linux

package common

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/text/gotext"
)

func newTextSystem() gpui.PlatformTextSystem {
	return gotext.NewTextSystem("IBM Plex Sans")
}
