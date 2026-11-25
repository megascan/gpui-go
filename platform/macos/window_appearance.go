//go:build darwin && !ios

package macos

import (
	"fmt"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func windowAppearanceFromNative(appearance appkit.NSAppearance) gpui.WindowAppearance {
	if appearance == 0 {
		return gpui.WindowAppearanceLight
	}
	name := appearance.Name()
	if name == objc.Nil {
		return gpui.WindowAppearanceLight
	}
	switch name {
	case appkit.NSAppearanceNameVibrantLight():
		return gpui.WindowAppearanceVibrantLight
	case appkit.NSAppearanceNameVibrantDark():
		return gpui.WindowAppearanceVibrantDark
	case appkit.NSAppearanceNameAqua():
		return gpui.WindowAppearanceLight
	case appkit.NSAppearanceNameDarkAqua():
		return gpui.WindowAppearanceDark
	default:
		fmt.Printf("unknown appearance: %q\n", nsStringToStr(name))
		return gpui.WindowAppearanceLight
	}
}
