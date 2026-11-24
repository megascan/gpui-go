package main

import (
	"path/filepath"
	"runtime"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// GifViewer shows an animated GIF.
type GifViewer struct {
	gifPath string
}

// NewGifViewer creates a GifViewer for the GIF at gifPath.
func NewGifViewer(gifPath string) *GifViewer {
	return &GifViewer{gifPath: gifPath}
}

// Render renders the GIF.
func (g *GifViewer) Render(_ *gpui.Window, _ *gpui.Context[GifViewer]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().Child(
		gpui.NewImg(gpui.ImageSourceFromPath(g.gifPath)).
			SizeFull().
			ObjectFit(gpui.ObjectFitContain).
			ID("gif"),
	)
}

func manifestDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		gifPath := filepath.Join(manifestDir(), "examples/gif_viewer/black-cat-typing.gif")

		options := gpui.DefaultWindowOptions()
		options.Focus = true
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[GifViewer] {
			return cx.New(func(_ *gpui.Context[GifViewer]) *GifViewer { return NewGifViewer(gifPath) })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
