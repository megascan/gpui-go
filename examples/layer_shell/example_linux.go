package main

import (
	"fmt"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// LayerShellExample is a clock drawn on a Wayland layer-shell surface.
type LayerShellExample struct{}

// NewLayerShellExample creates the example and starts its redraw timer.
func NewLayerShellExample(cx *gpui.Context[LayerShellExample]) *LayerShellExample {
	cx.Spawn[struct{}](func(this gpui.WeakEntity[LayerShellExample], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			_ = this.Update(cx, func(_ *LayerShellExample, cx *gpui.Context[LayerShellExample]) { cx.Notify() })
			if _, err := cx.BackgroundExecutor().Timer(500 * time.Millisecond).Await(cx); err != nil {
				return struct{}{}, err
			}
		}
	}).Detach()
	return &LayerShellExample{}
}

// Render renders the clock.
func (l *LayerShellExample) Render(_ *gpui.Window, _ *gpui.Context[LayerShellExample]) gpui.IntoElement {
	now := uint64(time.Now().Unix())
	hours := (now / 3600) % 24
	minutes := (now / 60) % 60
	seconds := now % 60

	return gpui.NewDiv().
		SizeFull().
		Flex().
		ItemsCenter().
		JustifyCenter().
		TextSize(gpui.Rems(4.5)).
		FontWeight(gpui.FontWeightExtraBold).
		TextColor(gpui.White()).
		Bg(gpui.NewRgba(0x0000044)).
		RoundedXl().
		ChildText(fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))
}

func exampleMain() {
	platform.Application().Run(func(cx *gpui.App) {
		options := gpui.DefaultWindowOptions()
		options.Titlebar = gpui.None[gpui.TitlebarOptions]()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.Bounds[gpui.Pixels]{
			Origin: gpui.NewPoint(gpui.Px(0.), gpui.Px(0.)),
			Size:   gpui.NewSize(gpui.Px(500.), gpui.Px(200.)),
		}))
		options.AppID = gpui.Some("gpui-layer-shell-example")
		options.WindowBackground = gpui.WindowBackgroundAppearanceTransparent
		layer := gpui.DefaultLayerShellOptions()
		layer.Namespace = "gpui"
		layer.Anchor = gpui.LayerShellAnchorLeft | gpui.LayerShellAnchorRight | gpui.LayerShellAnchorBottom
		layer.Margin = gpui.Some([4]gpui.Pixels{gpui.Px(0.), gpui.Px(0.), gpui.Px(40.), gpui.Px(0.)})
		layer.KeyboardInteractivity = gpui.KeyboardInteractivityNone
		options.Kind = gpui.WindowKindLayerShell{LayerShellOptions: layer}
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[LayerShellExample] {
			return cx.New(NewLayerShellExample)
		}); err != nil {
			panic(err)
		}
	})
}
