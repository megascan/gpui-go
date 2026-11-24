package main

import (
	"encoding/json/jsontext"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// ImageContainer is a caption next to an image.
type ImageContainer struct {
	text gpui.SharedString
	src  gpui.ImageSource
}

// NewImageContainer creates an ImageContainer.
func NewImageContainer(text gpui.SharedString, src gpui.ImageSource) ImageContainer {
	return ImageContainer{text: text, src: src}
}

// IntoAnyElement renders the container as a component.
func (c ImageContainer) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(c)
}

// Render renders the container.
func (c ImageContainer) Render(_ *gpui.Window, _ *gpui.App) gpui.IntoElement {
	return gpui.NewDiv().Child(
		gpui.NewDiv().
			FlexRow().
			SizeFull().
			Gap4().
			Child(c.text).
			Child(gpui.NewImg(c.src).Size(gpui.Px(256.0))),
	)
}

// ImageShowcase shows images loaded from a file, a URL and an asset.
type ImageShowcase struct {
	localResource  string
	remoteResource gpui.SharedURI
	assetResource  gpui.SharedString
}

// Render renders the showcase.
func (s *ImageShowcase) Render(_ *gpui.Window, _ *gpui.Context[ImageShowcase]) gpui.IntoElement {
	return gpui.NewDiv().
		ID("main").
		Bg(gpui.White()).
		OverflowYScroll().
		P5().
		SizeFull().
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				JustifyCenter().
				ItemsCenter().
				Gap8().
				Child(gpui.NewImg(gpui.ImageSourceFromString(
					"https://github.com/zed-industries/zed/actions/workflows/ci.yml/badge.svg",
				))).
				Child(
					gpui.NewDiv().
						Flex().
						FlexRow().
						JustifyCenter().
						ItemsCenter().
						Gap8().
						Child(NewImageContainer(
							"Image loaded from a local file with EXIF orientation",
							gpui.ImageSourceFromPath(s.localResource),
						)).
						Child(NewImageContainer(
							"Image loaded from a remote resource",
							gpui.ImageSourceFromSharedURI(s.remoteResource),
						)).
						Child(NewImageContainer(
							"Image loaded from an asset",
							gpui.ImageSourceFromSharedString(s.assetResource),
						)),
				).
				Child(
					gpui.NewDiv().
						Flex().
						FlexRow().
						Gap8().
						Child(
							gpui.NewDiv().
								FlexCol().
								ChildText("Auto Width").
								Child(gpui.NewImg(gpui.ImageSourceFromString("https://picsum.photos/800/400")).H(gpui.Px(180.))),
						).
						Child(
							gpui.NewDiv().
								FlexCol().
								ChildText("Auto Height").
								Child(gpui.NewImg(gpui.ImageSourceFromString("https://picsum.photos/800/400")).W(gpui.Px(180.))),
						),
				).
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						JustifyCenter().
						ItemsCenter().
						WFull().
						Border1().
						BorderColor(gpui.Rgb(0xC0C0C0)).
						ChildText("image with max width 100%").
						Child(gpui.NewImg(gpui.ImageSourceFromString("https://picsum.photos/800/400")).MaxWFull()),
				),
		)
}

// Quit quits the example.
type Quit struct{}

// Name returns the action name.
func (Quit) Name() string { return "image::Quit" }

// BoxedClone clones the action.
func (Quit) BoxedClone() gpui.Action { return Quit{} }

// PartialEq reports whether other is a Quit.
func (Quit) PartialEq(other gpui.Action) bool {
	_, ok := other.(Quit)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "image::Quit",
		Type:  reflect.TypeFor[Quit](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Quit{}, nil },
	})
}

func manifestDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func runExample() {
	manifest := manifestDir()

	app := platform.Application()
	httpClient, err := support.NewHTTPClientWithUserAgent("gpui example", app.BackgroundExecutor())
	if err != nil {
		panic(fmt.Sprintf("failed to create HTTP client: %v", err))
	}
	app = app.WithHTTPClient(httpClient)
	app.WithAssets(Assets{base: filepath.Join(manifest, "examples")}).
		Run(func(cx *gpui.App) {
			if !support.LoadFonts(cx) {
				return
			}
			cx.Activate(true)
			cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
			cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("cmd-q", Quit{}, gpui.None[string]())})
			cx.SetMenus([]gpui.Menu{{
				Name:     "Image",
				Items:    []gpui.MenuItem{gpui.NewMenuItemAction("Quit", Quit{})},
				Disabled: false,
			}})

			windowOptions := gpui.WindowOptions{
				Titlebar: gpui.Some(gpui.TitlebarOptions{
					Title:              gpui.Some[gpui.SharedString]("Image Example"),
					AppearsTransparent: false,
				}),
				WindowBounds: gpui.Some(gpui.WindowBoundsWindowed(gpui.Bounds[gpui.Pixels]{
					Size:   gpui.NewSize(gpui.Px(1100.), gpui.Px(600.)),
					Origin: gpui.NewPoint(gpui.Px(200.), gpui.Px(200.)),
				})),
			}

			if _, err := cx.OpenWindow(windowOptions, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[ImageShowcase] {
				return cx.New(func(_ *gpui.Context[ImageShowcase]) *ImageShowcase {
					return &ImageShowcase{
						localResource:  filepath.Join(manifest, "examples/image_example/exif-orientation-rotate-180.jpg"),
						remoteResource: "https://picsum.photos/800/400",
						assetResource:  "image_example/color.svg",
					}
				})
			}); err != nil {
				panic(err)
			}
		})
}

func main() {
	runExample()
}
