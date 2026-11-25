//go:build windows

package screencap

import gpui "github.com/megascan/gpui-go"

// Backend is the Windows gpui.ScapBackend.
type Backend struct{}

// GetAllTargets is GetAllTargets.
func (Backend) GetAllTargets() ([]gpui.ScapTarget, error) { return GetAllTargets() }

// BuildCapturer is Build.
func (Backend) BuildCapturer(options gpui.ScapCapturerOptions) (gpui.ScapCapturer, error) {
	c, err := Build(options)
	if err != nil {
		return nil, err
	}
	return c, nil
}
