package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
	"github.com/megascan/gpui-go/scheduler"
)

// Assets loads assets from the file system by absolute or working-directory-relative path.
type Assets struct{}

// Load reads the file at path.
func (Assets) Load(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("loading asset %s: %w", path, err)
	}
	return data, true, nil
}

// List lists the paths of the entries under path.
func (Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listing assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(filepath.Join(path, entry.Name())))
	}
	return names, nil
}

func imagePath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "app-icon.png")
}

// LoadImageParameters are the delay and outcome of a LoadImageWithParameters load.
type LoadImageParameters struct {
	timeout time.Duration
	fail    bool
}

// LoadImageWithParameters loads the example image after a delay, or fails.
type LoadImageWithParameters struct{}

// Load waits for the parameters' timeout, then yields the image or an error.
func (LoadImageWithParameters) Load(parameters LoadImageParameters, cx *gpui.App) func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
	timer := cx.BackgroundExecutor().Timer(parameters.timeout)
	data := gpui.AssetLogger[gpui.ImageAssetLoader, gpui.Resource, *gpui.RenderImage]{}.Load(
		gpui.ResourcePath(imagePath()),
		cx,
	)
	return func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage] {
		if _, err := timer.Await(s); err != nil {
			return scheduler.Result[*gpui.RenderImage]{Err: err}
		}
		if parameters.fail {
			slog.Error("Intentionally failed to load image")
			return scheduler.Result[*gpui.RenderImage]{Err: &gpui.ImageCacheError{
				Kind: gpui.ImageCacheErrorKindOther,
				Err:  errors.New("Failed to load image"),
			}}
		}
		return data(s)
	}
}

// ImageLoadingExample shows images that load fast, load slowly, fail, and are missing.
type ImageLoadingExample struct{}

func loadingElement() gpui.IntoElement {
	return gpui.NewDiv().SizeFull().FlexNone().P0p5().RoundedXs().Child(
		gpui.WithAnimation(
			gpui.NewDiv().SizeFull(),
			"loading-bg",
			gpui.NewAnimation(3*time.Second).
				Repeat().
				WithEasing(gpui.PulsatingBetween(0.04, 0.24)),
			func(this *gpui.Div, delta float32) *gpui.Div { return this.Bg(gpui.Black().Opacity(delta)) },
		),
	)
}

func fallbackElement() gpui.IntoElement {
	fallbackColor := gpui.Black().Opacity(0.5)

	return gpui.NewDiv().SizeFull().FlexNone().P0p5().Child(
		gpui.NewDiv().
			SizeFull().
			Flex().
			ItemsCenter().
			JustifyCenter().
			RoundedXs().
			TextSm().
			TextColor(fallbackColor).
			Border1().
			BorderColor(fallbackColor).
			ChildText("?"),
	)
}

func fallback() gpui.AnyElement { return fallbackElement().IntoAnyElement() }

func loading() gpui.AnyElement { return loadingElement().IntoAnyElement() }

func parameterImage(id string, imageSource LoadImageParameters) gpui.IntoElement {
	image := gpui.NewImg(gpui.ImageSourceFromFunc(func(window *gpui.Window, cx *gpui.App) (*gpui.RenderImage, bool, error) {
		result, ok := window.UseAsset[LoadImageWithParameters](imageSource, cx)
		return result.Value, ok, result.Err
	})).
		WithFallback(fallback).
		WithLoading(loading).
		Size12().
		Border1()
	return image.
		BorderColor(gpui.Red()).
		ID(id).
		OnClick(func(_ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.App) {
			cx.RemoveAsset[LoadImageWithParameters](imageSource)
		})
}

// Render renders the example.
func (e *ImageLoadingExample) Render(_ *gpui.Window, _ *gpui.Context[ImageLoadingExample]) gpui.IntoElement {
	missing := "this/file/really/shouldn't/exist/or/won't/be/an/image/I/hope"

	return gpui.NewDiv().Flex().FlexCol().SizeFull().JustifyAround().Child(
		gpui.NewDiv().Flex().FlexRow().WFull().JustifyAround().Child(
			gpui.NewDiv().
				Flex().
				Bg(gpui.White()).
				Size(gpui.LengthDefinite(gpui.Px(300.0).ToDefiniteLength())).
				JustifyCenter().
				ItemsCenter().
				Child(parameterImage("image-1", LoadImageParameters{
					timeout: max(gpui.LoadingDelay-25*time.Millisecond, 0),
					fail:    false,
				})).
				Child(parameterImage("image-2", LoadImageParameters{
					timeout: 5 * time.Second,
					fail:    false,
				})).
				Child(parameterImage("image-3", LoadImageParameters{
					timeout: 5 * time.Second,
					fail:    true,
				})).
				Child(
					gpui.NewImg(gpui.ImageSourceFromPath(missing)).
						Border1().
						Size12().
						WithFallback(fallback).
						BorderColor(gpui.Red()).
						WithLoading(loading).
						ID("image-4").
						OnClick(func(_ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.App) {
							cx.RemoveAsset[gpui.ImgResourceLoader](gpui.ResourcePath(missing))
						}),
				),
		),
	)
}

func runExample() {
	platform.Application().WithAssets(Assets{}).Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(300.), gpui.Px(300.)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[ImageLoadingExample] {
			cx.Activate(false)
			return cx.New(func(_ *gpui.Context[ImageLoadingExample]) *ImageLoadingExample { return &ImageLoadingExample{} })
		}); err != nil {
			panic(err)
		}
	})
}

func main() {
	runExample()
}
