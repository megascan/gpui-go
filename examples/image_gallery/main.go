package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

const imagesInGallery = 30

// ImageGallery shows remote images through a manually and an automatically managed cache.
type ImageGallery struct {
	imageKey   string
	itemsCount int
	totalCount int
	imageCache gpui.Entity[gpui.RetainAllImageCache]
}

func (g *ImageGallery) onNextImage(_ *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[ImageGallery]) {
	g.imageCache.Update(cx, func(imageCache *gpui.RetainAllImageCache, cx *gpui.Context[gpui.RetainAllImageCache]) {
		imageCache.Clear(window, cx.App)
	})

	t := time.Now().UnixMilli()
	g.imageKey = fmt.Sprintf("%d", t)
	g.totalCount += g.itemsCount
	cx.Notify()
}

func (g *ImageGallery) galleryImages(imageURL gpui.SharedString) []gpui.IntoElement {
	images := make([]gpui.IntoElement, 0, g.itemsCount)
	for ix := range g.itemsCount {
		images = append(images, gpui.NewImg(gpui.ImageSourceFromString(fmt.Sprintf("%s-%d", imageURL, ix))).Size20())
	}
	return images
}

// Render renders the gallery.
func (g *ImageGallery) Render(_ *gpui.Window, cx *gpui.Context[ImageGallery]) gpui.IntoElement {
	imageURL := gpui.SharedString(fmt.Sprintf("https://picsum.photos/400/200?t=%s", g.imageKey))

	return gpui.NewDiv().
		Flex().
		FlexCol().
		TextColor(gpui.White()).
		ChildText("Manually managed image cache:").
		Child(
			gpui.NewDiv().
				ImageCache(gpui.AnyImageCacheFromEntity(g.imageCache)).
				ID("main").
				TextColor(gpui.Black()).
				Bg(gpui.Rgb(0xE9E9E9)).
				OverflowYScroll().
				P4().
				SizeFull().
				Flex().
				FlexCol().
				ItemsCenter().
				Gap2().
				Child(
					gpui.NewDiv().
						WFull().
						Flex().
						FlexRow().
						JustifyBetween().
						ChildText(fmt.Sprintf(
							"Example to show images and test memory usage (Rendered: %d images).",
							g.totalCount,
						)).
						Child(
							gpui.NewDiv().
								ID("btn").
								Py1().
								Px4().
								Bg(gpui.Black()).
								Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement { return styled(this).Opacity(0.8) }).
								TextColor(gpui.White()).
								TextCenter().
								W40().
								ChildText("Next Photos").
								OnClick(cx.Listener[gpui.ClickEvent]((*ImageGallery).onNextImage)),
						),
				).
				Child(
					gpui.NewDiv().
						ID("image-gallery").
						Flex().
						FlexRow().
						FlexWrap().
						GapX4().
						GapY2().
						JustifyAround().
						Children(g.galleryImages(imageURL)...),
				),
		).
		ChildText("Automatically managed image cache:").
		Child(gpui.NewImageCache(simpleLruCache("lru-cache", imagesInGallery)).Child(
			gpui.NewDiv().
				ID("main").
				Bg(gpui.Rgb(0xE9E9E9)).
				TextColor(gpui.Black()).
				OverflowYScroll().
				P4().
				SizeFull().
				Flex().
				FlexCol().
				ItemsCenter().
				Gap2().
				Child(
					gpui.NewDiv().
						ID("image-gallery").
						Flex().
						FlexRow().
						FlexWrap().
						GapX4().
						GapY2().
						JustifyAround().
						Children(g.galleryImages(imageURL)...),
				),
		))
}

func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func simpleLruCache(id string, maxItems int) *SimpleLruCacheProvider {
	return &SimpleLruCacheProvider{
		id:       gpui.NewElementID(id),
		maxItems: maxItems,
	}
}

// SimpleLruCacheProvider provides a SimpleLruCache kept in element state.
type SimpleLruCacheProvider struct {
	id       gpui.ElementID
	maxItems int
}

// Provide returns the cache kept in the element state of the provider's id.
func (p *SimpleLruCacheProvider) Provide(window *gpui.Window, cx *gpui.App) gpui.AnyImageCache {
	cache := window.WithGlobalID[gpui.Entity[SimpleLruCache]](p.id, func(globalID *gpui.GlobalElementID, window *gpui.Window) gpui.Entity[SimpleLruCache] {
		return window.WithElementState[gpui.Entity[SimpleLruCache], gpui.Entity[SimpleLruCache]](
			globalID,
			func(lruCache gpui.Option[gpui.Entity[SimpleLruCache]], _ *gpui.Window) (gpui.Entity[SimpleLruCache], gpui.Entity[SimpleLruCache]) {
				cache, ok := lruCache.Get()
				if !ok {
					cache = cx.New(func(cx *gpui.Context[SimpleLruCache]) *SimpleLruCache { return NewSimpleLruCache(p.maxItems, cx) })
				}
				if cache.Read(cx).maxItems != p.maxItems {
					cache.Release()
					cache = cx.New(func(cx *gpui.Context[SimpleLruCache]) *SimpleLruCache { return NewSimpleLruCache(p.maxItems, cx) })
				}
				return cache, cache
			},
		)
	})
	return gpui.AnyImageCacheFromEntity(cache)
}

// SimpleLruCache keeps the most recently used images and drops the oldest.
type SimpleLruCache struct {
	maxItems int
	usages   []gpui.Resource
	cache    map[gpui.Resource]*gpui.ImageCacheItem
}

// NewSimpleLruCache creates a cache holding at most maxItems images.
func NewSimpleLruCache(maxItems int, cx *gpui.Context[SimpleLruCache]) *SimpleLruCache {
	cx.OnRelease(func(simpleCache *SimpleLruCache, cx *gpui.App) {
		cache := simpleCache.cache
		simpleCache.cache = map[gpui.Resource]*gpui.ImageCacheItem{}
		for _, item := range cache {
			if image, ok, err := item.Get(); ok && err == nil {
				cx.DropImage(image, nil)
			}
			item.Release()
		}
	}).Detach()

	return &SimpleLruCache{
		maxItems: maxItems,
		usages:   make([]gpui.Resource, 0, maxItems),
		cache:    make(map[gpui.Resource]*gpui.ImageCacheItem, maxItems),
	}
}

// Load returns the image of resource, loading it and evicting the oldest image when full.
func (c *SimpleLruCache) Load(resource gpui.Resource, window *gpui.Window, cx *gpui.App) (*gpui.RenderImage, bool, error) {
	if len(c.usages) != len(c.cache) {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", len(c.usages), len(c.cache)))
	}
	if len(c.cache) > c.maxItems {
		panic("assertion failed: self.cache.len() <= self.max_items")
	}

	hash := resource

	if item, ok := c.cache[hash]; ok {
		currentIx := slices.Index(c.usages, hash)
		if currentIx < 0 {
			panic("cache and usages must stay in sync")
		}
		c.usages = slices.Delete(c.usages, currentIx, currentIx+1)
		c.usages = slices.Insert(c.usages, 0, hash)
		return item.UseImage(window)
	}

	if len(c.usages) == c.maxItems {
		oldest := c.usages[len(c.usages)-1]
		c.usages = c.usages[:len(c.usages)-1]
		image, ok := c.cache[oldest]
		if !ok {
			panic("cache and usages must be in sync")
		}
		delete(c.cache, oldest)
		if renderImage, loaded, err := image.Get(); loaded && err == nil {
			cx.DropImage(renderImage, window)
		}
		image.Release()
	}

	item := gpui.NewImageCacheItem(resource, cx)
	renderImage, loaded, err := item.UseImage(window)
	c.cache[hash] = item
	c.usages = slices.Insert(c.usages, 0, hash)
	return renderImage, loaded, err
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

func runExample() {
	app := platform.Application()
	httpClient, err := support.NewHTTPClientWithUserAgent("gpui example", app.BackgroundExecutor())
	if err != nil {
		panic(fmt.Sprintf("failed to create HTTP client: %v", err))
	}
	app = app.WithHTTPClient(httpClient)

	app.Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.Activate(true)
		cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })
		cx.BindKeys([]gpui.KeyBinding{gpui.NewKeyBinding("cmd-q", Quit{}, gpui.None[string]())})
		cx.SetMenus([]gpui.Menu{gpui.NewMenu("Image Gallery").WithItems(gpui.NewMenuItemAction("Quit", Quit{}))})

		windowOptions := gpui.DefaultWindowOptions()
		windowOptions.Titlebar = gpui.Some(gpui.TitlebarOptions{
			Title:              gpui.Some[gpui.SharedString]("Image Gallery"),
			AppearsTransparent: false,
		})
		windowOptions.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(1100.), gpui.Px(860.)),
			cx,
		)))

		if _, err := cx.OpenWindow(windowOptions, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[ImageGallery] {
			return cx.New(func(ctx *gpui.Context[ImageGallery]) *ImageGallery {
				return &ImageGallery{
					imageKey:   "",
					itemsCount: imagesInGallery,
					totalCount: 0,
					imageCache: gpui.NewRetainAllImageCache(ctx.App),
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
