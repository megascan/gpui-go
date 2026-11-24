//go:build windows

package platform

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/screencap"
	"github.com/megascan/gpui-go/platform/windows"
)

func init() {
	gpui.RegisterScapBackend(screencap.Backend{})
}

func currentPlatform(headless bool) (Platform, error) {
	p, err := windows.NewWindowsPlatform(headless)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Windows platform: %w", err)
	}
	return p, nil
}

func currentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error) {
	return nil, nil
}
