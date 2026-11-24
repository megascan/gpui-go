package gpui

import (
	"fmt"

	"github.com/megascan/gpui-go/scheduler"
)

// NewImageCache creates an image cache element: all its child img elements will use the cache
// specified by this element. Note that this could be as simple as passing the AnyImageCache of
// an entity.
func NewImageCache(imageCacheProvider ImageCacheProvider) *ImageCacheElement {
	e := &ImageCacheElement{imageCacheProvider: imageCacheProvider}
	BindStyled(&e.Styled, e)
	BindParent(&e.Parent, e)
	BindFluent(&e.Fluent, e)
	return e
}

// AnyImageCache is a dynamically typed image cache, which can be used to store any image cache.
// It refers to its entity without keeping it alive (D-WEAKIMAGECACHE).
type AnyImageCache struct {
	imageCache AnyWeakEntity
	loadFn     func(resource Resource, window *Window, cx *App) (*RenderImage, bool, error)
}

// AnyImageCacheFromEntity returns the type-erased image cache of an entity whose state
// implements ImageCache.
func AnyImageCacheFromEntity[I any, PI interface {
	*I
	ImageCache
}](imageCache Entity[I]) AnyImageCache {
	weak := imageCache.Downgrade()
	return AnyImageCache{
		imageCache: weak.AnyWeakEntity,
		loadFn: func(resource Resource, window *Window, cx *App) (*RenderImage, bool, error) {
			var image *RenderImage
			var loaded bool
			var loadErr error
			if err := weak.Update(cx, func(imageCache *I, cx *Context[I]) {
				image, loaded, loadErr = PI(imageCache).Load(resource, window, cx.App)
			}); err != nil {
				panic(fmt.Sprintf("called `Option::unwrap()` on a `None` value: %v", err))
			}
			return image, loaded, loadErr
		},
	}
}

// Load loads an image given a resource. It returns the result of loading the image if it has
// finished loading, or false if it is still loading.
func (c AnyImageCache) Load(resource Resource, window *Window, cx *App) (*RenderImage, bool, error) {
	return c.loadFn(resource, window, cx)
}

// Provide returns c itself, so an AnyImageCache is its own provider (upstream's impl of
// ImageCacheProvider for Entity).
func (c AnyImageCache) Provide(_ *Window, _ *App) AnyImageCache {
	return c
}

// ImageCacheElement is an image cache element.
type ImageCacheElement struct {
	Styled[*ImageCacheElement]
	Parent[*ImageCacheElement]
	Fluent[*ImageCacheElement]
	imageCacheProvider ImageCacheProvider
	style              StyleRefinement
	children           []AnyElement
}

// Extend adds children to the element.
func (e *ImageCacheElement) Extend(elements []AnyElement) {
	e.children = append(e.children, elements...)
}

// Style returns the element's style refinement.
func (e *ImageCacheElement) Style() *StyleRefinement {
	return &e.style
}

// ElementID reports that the element has no id.
func (e *ImageCacheElement) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (e *ImageCacheElement) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests the children's layouts with the element's image cache current.
func (e *ImageCacheElement) RequestLayout(_ *GlobalElementID, _ *InspectorElementID, window *Window, cx *App) (LayoutID, []LayoutID) {
	imageCache := e.imageCacheProvider.Provide(window, cx)
	var childLayoutIDs []LayoutID
	layoutID := window.WithImageCache(Some(imageCache), func(window *Window) LayoutID {
		childLayoutIDs = make([]LayoutID, 0, len(e.children))
		for i := range e.children {
			childLayoutIDs = append(childLayoutIDs, e.children[i].RequestLayout(window, cx))
		}
		style := DefaultStyle()
		style.Refine(&e.style)
		return window.RequestLayout(style, childLayoutIDs, cx)
	})
	return layoutID, childLayoutIDs
}

// Prepaint prepaints the children.
func (e *ImageCacheElement) Prepaint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *[]LayoutID, window *Window, cx *App) struct{} {
	for i := range e.children {
		e.children[i].Prepaint(window, cx)
	}
	return struct{}{}
}

// Paint paints the children with the element's image cache current.
func (e *ImageCacheElement) Paint(_ *GlobalElementID, _ *InspectorElementID, _ Bounds[Pixels], _ *[]LayoutID, _ *struct{}, window *Window, cx *App) {
	imageCache := e.imageCacheProvider.Provide(window, cx)
	window.WithImageCache(Some(imageCache), func(window *Window) struct{} {
		for i := range e.children {
			e.children[i].Paint(window, cx)
		}
		return struct{}{}
	})
}

// IntoAnyElement converts the element into an AnyElement.
func (e *ImageCacheElement) IntoAnyElement() AnyElement {
	return NewAnyElement[[]LayoutID, struct{}](e)
}

// ImageCacheItem owns an image load and its cached result.
//
// Releasing the entry cancels a pending load. Image caches must remove loaded images from windows
// before releasing the entry.
type ImageCacheItem struct {
	load *cachedLoad[scheduler.Result[*RenderImage]]
}

// String formats the item like its Rust Debug output.
func (i *ImageCacheItem) String() string {
	image, ok, err := i.Get()
	result := "None"
	switch {
	case ok && err != nil:
		result = fmt.Sprintf("Some(Err(%v))", err)
	case ok:
		result = fmt.Sprintf("Some(Ok(%v))", image)
	}
	return fmt.Sprintf("ImageCacheItem { result: %s }", result)
}

// NewImageCacheItem starts loading an image into a new cache entry.
func NewImageCacheItem(source Resource, cx *App) *ImageCacheItem {
	var loader ImgResourceLoader
	future := loader.Load(source, cx)
	return &ImageCacheItem{load: newCachedLoad(future, cx)}
}

// Get returns the cached result without subscribing to completion notifications.
func (i *ImageCacheItem) Get() (*RenderImage, bool, error) {
	result, ok := i.load.get()
	return result.Value, ok, result.Err
}

// UseImage returns the cached result or subscribes the current view to completion.
func (i *ImageCacheItem) UseImage(window *Window) (*RenderImage, bool, error) {
	result, ok := i.load.useBy(window.CurrentView())
	return result.Value, ok, result.Err
}

// Release drops the entry, canceling a pending load.
func (i *ImageCacheItem) Release() {
	i.load.release()
}

// ImageCache is an object that can handle the caching and unloading of images. Implementations
// should ensure that images are removed from all windows when they are no longer needed.
type ImageCache interface {
	// Load loads an image given a resource. It returns the result of loading the image if it has
	// finished loading, or false if it is still loading.
	Load(resource Resource, window *Window, cx *App) (*RenderImage, bool, error)
}

// ImageCacheProvider is an object that can create an ImageCache during the render phase. See
// ImageCache for more information.
type ImageCacheProvider interface {
	// Provide is called during the request_layout phase to create an ImageCache.
	Provide(window *Window, cx *App) AnyImageCache
}

// RetainAllImageCache is an implementation of ImageCache that retains every image it loads.
type RetainAllImageCache struct {
	items map[Resource]*ImageCacheItem
}

// String formats the cache like its Rust Debug output.
func (c *RetainAllImageCache) String() string {
	return fmt.Sprintf("HashMapImageCache { num_images: %d }", len(c.items))
}

// NewRetainAllImageCache creates a new image cache.
func NewRetainAllImageCache(cx *App) Entity[RetainAllImageCache] {
	e := cx.New(func(*Context[RetainAllImageCache]) *RetainAllImageCache {
		return &RetainAllImageCache{items: map[Resource]*ImageCacheItem{}}
	})
	cx.ObserveRelease(e, func(imageCache *RetainAllImageCache, cx *App) {
		items := imageCache.items
		imageCache.items = map[Resource]*ImageCacheItem{}
		for _, item := range items {
			if image, ok, err := item.Get(); ok && err == nil {
				cx.DropImage(image, nil)
			}
			item.Release()
		}
	}).Detach()
	return e
}

// Load loads an image from the given source. It returns false if the image is loading.
func (c *RetainAllImageCache) Load(source Resource, window *Window, cx *App) (*RenderImage, bool, error) {
	item, ok := c.items[source]
	if !ok {
		item = NewImageCacheItem(source, cx)
		c.items[source] = item
	}
	return item.UseImage(window)
}

// Clear clears the image cache.
func (c *RetainAllImageCache) Clear(window *Window, cx *App) {
	items := c.items
	c.items = map[Resource]*ImageCacheItem{}
	for _, item := range items {
		if image, ok, err := item.Get(); ok && err == nil {
			cx.DropImage(image, window)
		}
		item.Release()
	}
}

// Remove removes the image of the given source from the cache.
func (c *RetainAllImageCache) Remove(source Resource, window *Window, cx *App) {
	item, ok := c.items[source]
	if !ok {
		return
	}
	delete(c.items, source)
	if image, ok, err := item.Get(); ok && err == nil {
		cx.DropImage(image, window)
	}
	item.Release()
}

// Len returns the number of images in the cache.
func (c *RetainAllImageCache) Len() int {
	return len(c.items)
}

// IsEmpty returns true if the cache is empty.
func (c *RetainAllImageCache) IsEmpty() bool {
	return len(c.items) == 0
}

// RetainAll constructs a retain-all image cache that uses the element state associated with the
// given ID.
func RetainAll[I ElementIDSource](id I) *RetainAllImageCacheProvider {
	return &RetainAllImageCacheProvider{id: NewElementID(id)}
}

// RetainAllImageCacheProvider is a provider struct for creating a retain-all image cache inline.
type RetainAllImageCacheProvider struct {
	id ElementID
}

// Provide returns the retain-all cache kept in the element state of the provider's id, creating
// it on first use.
func (p *RetainAllImageCacheProvider) Provide(window *Window, cx *App) AnyImageCache {
	cache := window.WithGlobalID(p.id, func(globalID *GlobalElementID, window *Window) Entity[RetainAllImageCache] {
		return window.WithElementState[Entity[RetainAllImageCache], Entity[RetainAllImageCache]](globalID, func(cache Option[Entity[RetainAllImageCache]], _ *Window) (Entity[RetainAllImageCache], Entity[RetainAllImageCache]) {
			c, ok := cache.Get()
			if !ok {
				c = NewRetainAllImageCache(cx)
			}
			return c, c
		})
	})
	return AnyImageCacheFromEntity(cache)
}
