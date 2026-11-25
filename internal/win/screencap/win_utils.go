//go:build windows

package screencap

import "github.com/megascan/gpui-go/internal/win/wgc"

func winIsSupported() bool {
	ok, err := wgc.GraphicsCaptureApiIsSupported()
	if err != nil {
		panic("Failed to check support: " + err.Error())
	}
	return ok
}
