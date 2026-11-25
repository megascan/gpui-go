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

// HelloWorld is the opacity example view.
type HelloWorld struct {
	task      *gpui.Task[struct{}]
	opacity   float32
	animating bool
}

// NewHelloWorld creates the view.
func NewHelloWorld(_ *gpui.Window, _ *gpui.Context[HelloWorld]) *HelloWorld {
	return &HelloWorld{
		task:      nil,
		opacity:   0.5,
		animating: false,
	}
}

func (h *HelloWorld) startAnimation(_ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[HelloWorld]) {
	h.opacity = 0.0
	h.animating = true
	cx.Notify()
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

// Render renders the view.
func (h *HelloWorld) Render(window *gpui.Window, cx *gpui.Context[HelloWorld]) gpui.IntoElement {
	if h.animating {
		h.opacity += 0.005
		if h.opacity >= 1.0 {
			h.animating = false
			h.opacity = 1.0
		} else {
			window.RequestAnimationFrame()
		}
	}

	emojis := []string{"🎊", "✈️", "🎉", "🎈", "🎁", "🎂"}
	emojiDivs := make([]gpui.IntoElement, 0, len(emojis))
	for _, emoji := range emojis {
		emojiDivs = append(emojiDivs, gpui.NewDiv().
			ChildText(emoji).
			Hover(func(style *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(style).Opacity(0.5) }))
	}

	return gpui.NewDiv().
		Flex().
		FlexRow().
		SizeFull().
		Bg(gpui.Rgb(0xe0e0e0)).
		TextXl().
		Child(
			gpui.NewDiv().
				Flex().
				SizeFull().
				JustifyCenter().
				ItemsCenter().
				Border1().
				TextColor(gpui.Blue()).
				Child(gpui.NewDiv().ChildText("This is background text.")),
		).
		Child(
			gpui.NewDiv().
				ID("panel").
				OnClick(cx.Listener((*HelloWorld).startAnimation)).
				Absolute().
				Top8().
				Left8().
				Right8().
				Bottom8().
				Opacity(h.opacity).
				Flex().
				JustifyCenter().
				ItemsCenter().
				Bg(gpui.White()).
				Border3().
				BorderColor(gpui.Red()).
				TextColor(gpui.Yellow()).
				Child(
					gpui.NewDiv().
						Flex().
						FlexCol().
						Gap2().
						JustifyCenter().
						ItemsCenter().
						Size(gpui.Px(300.)).
						Bg(gpui.Blue()).
						Border3().
						BorderColor(gpui.Black()).
						Shadow([]gpui.BoxShadow{
							gpui.NewBoxShadow(gpui.Px(10.0), gpui.Px(10.0), gpui.NewHsla(0.0, 0.0, 0.0, 0.5)).
								BlurRadius(gpui.Px(1.0)).
								SpreadRadius(gpui.Px(5.0)),
						}).
						Child(gpui.NewImg(gpui.ImageSourceFromString("image/app-icon.png")).Size8()).
						ChildText("Opacity Panel (Click to test)").
						Child(
							gpui.NewDiv().
								ID("deep-level-text").
								Flex().
								JustifyCenter().
								ItemsCenter().
								P4().
								Bg(gpui.Black()).
								TextColor(gpui.White()).
								TextDecoration2().
								TextDecorationWavy().
								TextDecorationColor(gpui.Red()).
								ChildText(fmt.Sprintf("opacity: %.1f", h.opacity)),
						).
						Child(
							gpui.NewSvg().
								Path("image/arrow_circle.svg").
								TextColor(gpui.Black()).
								Text2xl().
								Size8(),
						).
						Child(gpui.NewDiv().Flex().Children(emojiDivs...)).
						Child(gpui.NewImg(gpui.ImageSourceFromString("image/black-cat-typing.gif")).Size12()),
				),
		)
}

func exampleDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func runExample() {
	platform.Application().
		WithAssets(Assets{base: exampleDir()}).
		Run(func(cx *gpui.App) {
			if !support.LoadFonts(cx) {
				return
			}
			bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.0), gpui.Px(500.0)), cx)
			options := gpui.DefaultWindowOptions()
			options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
			if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
				return cx.New(func(cx *gpui.Context[HelloWorld]) *HelloWorld { return NewHelloWorld(window, cx) })
			}); err != nil {
				panic(err)
			}
			cx.Activate(true)
		})
}

func main() {
	runExample()
}
