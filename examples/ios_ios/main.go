//go:build ios

package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
	"github.com/megascan/gpui-go/platform/ios"
)

// IosExample is the native iOS smoke application view.
type IosExample struct {
	tapCount int
}

// Render renders the view.
func (e *IosExample) Render(_ *gpui.Window, cx *gpui.Context[IosExample]) gpui.IntoElement {
	return gpui.NewDiv().
		SizeFull().
		Flex().
		FlexCol().
		ItemsCenter().
		JustifyCenter().
		Gap6().
		Bg(gpui.Rgb(0x101622)).
		TextColor(gpui.Rgb(0xf5f7ff)).
		Child(gpui.NewDiv().Text3xl().ChildText("GPUI on iOS")).
		Child(
			gpui.NewDiv().
				ID("tap-counter").
				Px6().
				Py3().
				RoundedLg().
				Bg(gpui.Rgb(0x246bfd)).
				TextXl().
				OnClick(cx.Listener(func(this *IosExample, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[IosExample]) {
					this.tapCount++
					cx.Notify()
				})).
				ChildText(fmt.Sprintf("Taps: %d", e.tapCount)),
		).
		Child(
			gpui.NewDiv().
				MaxW(gpui.Px(320.)).
				TextCenter().
				ChildText("Go app, native UIKit loop, GPUI rendering"),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		ios.SetStatusBarStyle(ios.StatusBarContentStyleLight)
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[IosExample] {
			return cx.New(func(_ *gpui.Context[IosExample]) *IosExample {
				return &IosExample{tapCount: 0}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
