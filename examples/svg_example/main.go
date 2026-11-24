package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Assets loads assets from a directory.
type Assets struct {
	base string
}

// Load reads the file at path under the base directory.
func (a Assets) Load(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(filepath.Join(a.base, path))
	if err != nil {
		return nil, false, fmt.Errorf("loading asset %s: %w", path, err)
	}
	return data, true, nil
}

// List lists the file names under path.
func (a Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := os.ReadDir(filepath.Join(a.base, path))
	if err != nil {
		return nil, fmt.Errorf("listing assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(entry.Name()))
	}
	return names, nil
}

// SvgExample renders the dragon three times.
type SvgExample struct{}

// Render renders the view.
func (*SvgExample) Render(_ *gpui.Window, _ *gpui.Context[SvgExample]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexRow().
		SizeFull().
		JustifyCenter().
		ItemsCenter().
		Gap8().
		Bg(gpui.Rgb(0xffffff)).
		Child(
			gpui.NewSvg().
				Path("svg_example/dragon.svg").
				Size8().
				TextColor(gpui.Rgb(0xff0000)),
		).
		Child(
			gpui.NewSvg().
				Path("svg_example/dragon.svg").
				Size8().
				TextColor(gpui.Rgb(0x00ff00)),
		).
		Child(
			gpui.NewSvg().
				Path("svg_example/dragon.svg").
				Size8().
				TextColor(gpui.Rgb(0x0000ff)),
		)
}

func examplesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(file))
}

func runExample() {
	platform.Application().
		WithAssets(Assets{base: examplesDir()}).
		Run(func(cx *gpui.App) {
			if !support.LoadFonts(cx) {
				return
			}
			bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.0), gpui.Px(300.0)), cx)
			if _, err := cx.OpenWindow(gpui.WindowOptions{
				WindowBounds: gpui.Some(gpui.WindowBoundsWindowed(bounds)),
			}, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[SvgExample] {
				return cx.New(func(_ *gpui.Context[SvgExample]) *SvgExample { return &SvgExample{} })
			}); err != nil {
				panic(err)
			}
			cx.Activate(true)
		})
}

func main() {
	runExample()
}
