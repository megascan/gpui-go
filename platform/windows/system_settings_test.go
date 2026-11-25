//go:build windows

package windows

import (
	"testing"

	"github.com/megascan/gpui-go/internal/win/win32"
)

func TestSystemSettings_ReadsWheelSettings(t *testing.T) {
	settings := NewWindowsSystemSettings()
	settings.Update(win32.SPI_GETWHEELSCROLLLINES)
	settings.Update(0)
	_ = settings.MouseWheelSettings.WheelScrollLines
}
