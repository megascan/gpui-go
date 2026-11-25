//go:build linux

package linux

import (
	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux/internal/screencap"
)

func init() {
	gpui.RegisterScapBackend(screencap.Backend{})
}
