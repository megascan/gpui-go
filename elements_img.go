package gpui

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// LoadingDelay is the delay before showing the loading state.
const LoadingDelay = 200 * time.Millisecond

// ImgResourceLoader is the resource loader that the img element uses.
//
// Note: that this is only for Resources, like URLs or file paths. Custom loaders, or external
// images will not use this asset loader.
type ImgResourceLoader = AssetLogger[ImageAssetLoader, Resource, *RenderImage]

// ImageSource is a source of image content: ImageSourceResource, ImageSourceRender,
// ImageSourceImage or ImageSourceCustom.
type ImageSource interface {
	// RemoveAsset removes this image source from the asset system.
	RemoveAsset(cx *App)
	// IsAssetCached checks whether this image source is present in the asset system (loading or
	// loaded), without fetching it.
	IsAssetCached(cx *App) bool
	isImageSource()
}

// ImageSourceResource is image content that will be loaded from some resource location.
type ImageSourceResource struct {
	// Resource is the resource location.
	Resource Resource
}

// ImageSourceRender is cached image data.
type ImageSourceRender struct {
	// Image is the decoded image.
	Image *RenderImage
}

// ImageSourceImage is cached image data.
type ImageSourceImage struct {
	// Image is the encoded image.
	Image *Image
}

// ImageSourceCustom is a custom loading function to use. Load returns false while the image is
// loading.
type ImageSourceCustom struct {
	// Load loads the image.
	Load func(window *Window, cx *App) (*RenderImage, bool, error)
}

func (ImageSourceResource) isImageSource() {}
func (ImageSourceRender) isImageSource()   {}
func (ImageSourceImage) isImageSource()    {}
func (ImageSourceCustom) isImageSource()   {}

// RemoveAsset removes this image source from the asset system.
func (s ImageSourceResource) RemoveAsset(cx *App) { imageSourceRemoveAsset(s, cx) }

// RemoveAsset removes this image source from the asset system.
func (s ImageSourceRender) RemoveAsset(cx *App) { imageSourceRemoveAsset(s, cx) }

// RemoveAsset removes this image source from the asset system.
func (s ImageSourceImage) RemoveAsset(cx *App) { imageSourceRemoveAsset(s, cx) }

// RemoveAsset removes this image source from the asset system.
func (s ImageSourceCustom) RemoveAsset(cx *App) { imageSourceRemoveAsset(s, cx) }

// IsAssetCached checks whether this image source is present in the asset system.
func (s ImageSourceResource) IsAssetCached(cx *App) bool { return imageSourceIsAssetCached(s, cx) }

// IsAssetCached checks whether this image source is present in the asset system.
func (s ImageSourceRender) IsAssetCached(cx *App) bool { return imageSourceIsAssetCached(s, cx) }

// IsAssetCached checks whether this image source is present in the asset system.
func (s ImageSourceImage) IsAssetCached(cx *App) bool { return imageSourceIsAssetCached(s, cx) }

// IsAssetCached checks whether this image source is present in the asset system.
func (s ImageSourceCustom) IsAssetCached(cx *App) bool { return imageSourceIsAssetCached(s, cx) }

func isURI(uri string) bool {
	colon := strings.IndexByte(uri, ':')
	if colon <= 0 {
		return false
	}
	scheme := uri[:colon]
	for i := 0; i < len(scheme); i++ {
		c := scheme[i]
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if i == 0 && !isAlpha {
			return false
		}
		if !isAlpha && !(c >= '0' && c <= '9') && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	switch strings.ToLower(scheme) {
	case "http", "https", "ws", "wss", "ftp":
		parsed, err := url.Parse(uri)
		return err == nil && parsed.Host != ""
	}
	return true
}

// ImageSourceFromSharedURI is a resource at the given URI.
func ImageSourceFromSharedURI(uri SharedURI) ImageSource {
	return ImageSourceResource{Resource: ResourceURI(uri)}
}

// ImageSourceFromString is a URI resource when s parses as a URL, an embedded resource
// otherwise.
func ImageSourceFromString(s string) ImageSource {
	if isURI(s) {
		return ImageSourceResource{Resource: ResourceURI(SharedURI(s))}
	}
	return ImageSourceResource{Resource: ResourceEmbedded(SharedString(s))}
}

// ImageSourceFromSharedString is ImageSourceFromString of s.
func ImageSourceFromSharedString(s SharedString) ImageSource {
	return ImageSourceFromString(string(s))
}

// ImageSourceFromPath is a resource at the given file system path.
func ImageSourceFromPath(path string) ImageSource {
	return ImageSourceResource{Resource: ResourcePath(path)}
}

// ImageSourceFromRenderImage is cached image data.
func ImageSourceFromRenderImage(image *RenderImage) ImageSource {
	return ImageSourceRender{Image: image}
}

// ImageSourceFromImage is cached image data.
func ImageSourceFromImage(image *Image) ImageSource {
	return ImageSourceImage{Image: image}
}

// ImageSourceFromFunc is a custom loading function.
func ImageSourceFromFunc(load func(window *Window, cx *App) (*RenderImage, bool, error)) ImageSource {
	return ImageSourceCustom{Load: load}
}

// ImageStyle is the style of an image element.
type ImageStyle struct {
	grayscale bool
	objectFit ObjectFit
	loading   func() AnyElement
	fallback  func() AnyElement
}

// DefaultImageStyle returns the default image style.
func DefaultImageStyle() ImageStyle {
	return ImageStyle{
		grayscale: false,
		objectFit: ObjectFitContain,
		loading:   nil,
		fallback:  nil,
	}
}

// StyledImageElement is implemented by elements that carry an ImageStyle.
type StyledImageElement interface {
	// ImageStyle returns the element's image style.
	ImageStyle() *ImageStyle
}

// StyledImage is the StyledImage mixin; bind it with BindStyledImage.
type StyledImage[S StyledImageElement] struct {
	self S
}

// BindStyledImage binds a StyledImage mixin to the element that embeds it.
func BindStyledImage[S StyledImageElement](m *StyledImage[S], self S) {
	m.self = self
}

// Grayscale sets the image to be displayed in grayscale.
func (m StyledImage[S]) Grayscale(grayscale bool) S {
	m.self.ImageStyle().grayscale = grayscale
	return m.self
}

// ObjectFit sets the object fit for the image.
func (m StyledImage[S]) ObjectFit(objectFit ObjectFit) S {
	m.self.ImageStyle().objectFit = objectFit
	return m.self
}

// WithFallback sets a fallback function that will be invoked to render an error view should the
// image fail to load.
func (m StyledImage[S]) WithFallback(fallback func() AnyElement) S {
	m.self.ImageStyle().fallback = fallback
	return m.self
}

// WithLoading sets a fallback function that will be invoked to render a view while the image is
// still being loaded.
func (m StyledImage[S]) WithLoading(loading func() AnyElement) S {
	m.self.ImageStyle().loading = loading
	return m.self
}

// Img is an image element.
type Img struct {
	Styled[*Img]
	Interactive[*Img]
	InteractiveID[*Img]
	StatefulInteractive[*Img]
	Fluent[*Img]
	StyledImage[*Img]
	interactivity Interactivity
	source        ImageSource
	style         ImageStyle
	imageCache    Option[AnyImageCache]
}

// NewImg creates a new image element.
func NewImg(source ImageSource) *Img {
	i := &Img{
		source:     source,
		style:      DefaultImageStyle(),
		imageCache: None[AnyImageCache](),
	}
	i.interactivity.sourceLocation = CallerSourceLocation(1)
	BindStyled(&i.Styled, i)
	BindInteractive(&i.Interactive, i)
	BindInteractiveID(&i.InteractiveID, i)
	BindStatefulInteractive(&i.StatefulInteractive, i)
	BindFluent(&i.Fluent, i)
	BindStyledImage(&i.StyledImage, i)
	return i
}

// ImgExtensions returns a list of all format extensions currently supported by the img element.
func ImgExtensions() []string {
	return []string{
		"avif", "jpg", "jpeg", "png", "gif", "webp", "tif", "tiff", "tga", "dds", "bmp", "ico",
		"hdr", "exr", "pbm", "pam", "ppm", "pgm", "ff", "farbfeld", "qoi", "svg",
	}
}

// ImageCache sets the image cache for the current node.
//
// If the image cache is not explicitly provided, the element determines it by:
//
//  1. Checking if any ancestor node of the current node contains an ImageCacheElement. If such a
//     node exists, the image cache specified by that ancestor will be used.
//  2. If no ancestor node contains an ImageCacheElement, the global image cache will be used as a
//     fallback.
func (i *Img) ImageCache(imageCache AnyImageCache) *Img {
	i.imageCache = Some(imageCache)
	return i
}

// ImageStyle returns the element's image style.
func (i *Img) ImageStyle() *ImageStyle {
	return &i.style
}

type imgStartedLoading struct {
	at   scheduler.Instant
	task *Task[struct{}]
}

type imgState struct {
	frameIndex     int
	lastFrameTime  Option[scheduler.Instant]
	startedLoading Option[imgStartedLoading]
}

func (s *imgState) clearStartedLoading() {
	if started, ok := s.startedLoading.Take().Get(); ok {
		started.task.Cancel()
	}
}

// ImgLayoutState is the image layout state between frames.
type ImgLayoutState struct {
	frameIndex  int
	replacement Option[AnyElement]
}

// ElementID returns the element's id, if it has one.
func (i *Img) ElementID() (ElementID, bool) {
	return i.interactivity.ElementID.Get()
}

// SourceLocation returns where the element was created.
func (i *Img) SourceLocation() *SourceLocation {
	return i.interactivity.SourceLocation()
}

func (i *Img) currentImageCache(window *Window) Option[AnyImageCache] {
	if i.imageCache.IsSome() {
		return i.imageCache
	}
	if n := len(window.imageCacheStack); n != 0 {
		return Some(window.imageCacheStack[n-1])
	}
	return None[AnyImageCache]()
}

// RequestLayout loads the image and requests a layout sized by it, or by its loading or
// fallback replacement.
func (i *Img) RequestLayout(globalID *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, ImgLayoutState) {
	layoutState := ImgLayoutState{
		frameIndex:  0,
		replacement: None[AnyElement](),
	}

	layoutID := window.WithOptionalElementState[imgState, LayoutID](globalID, func(stateSlot Option[Option[imgState]], window *Window) (LayoutID, Option[imgState]) {
		var state *imgState
		if inner, ok := stateSlot.Get(); ok {
			s := inner.UnwrapOr(imgState{
				frameIndex:     0,
				lastFrameTime:  None[scheduler.Instant](),
				startedLoading: None[imgStartedLoading](),
			})
			state = &s
		}

		frameIndex := 0
		if state != nil {
			frameIndex = state.frameIndex
		}

		layoutID := i.interactivity.RequestLayout(globalID, inspectorID, window, cx, func(style Style, window *Window, cx *App) LayoutID {
			var replacementID []LayoutID

			data, loaded, err := imageSourceUseData(i.source, i.currentImageCache(window), window, cx)
			switch {
			case loaded && err == nil:
				frameCount := data.FrameCount()
				maxFrameIndex := max(frameCount-1, 0)

				if state != nil {
					state.frameIndex = min(state.frameIndex, maxFrameIndex)
					if frameCount > 1 && !cx.ReduceMotion() {
						if window.IsWindowActive() {
							currentTime := cx.BackgroundExecutor().Now()
							if lastFrameTime, ok := state.lastFrameTime.Get(); ok {
								elapsed := currentTime.Sub(lastFrameTime)
								frameDuration := data.Delay(state.frameIndex)

								if elapsed >= frameDuration {
									state.frameIndex = (state.frameIndex + 1) % frameCount
									state.lastFrameTime = Some(currentTime.Add(-(elapsed - frameDuration)))
								}
							} else {
								state.lastFrameTime = Some(currentTime)
							}
						} else {
							state.lastFrameTime = None[scheduler.Instant]()
						}
					} else {
						state.lastFrameTime = None[scheduler.Instant]()
					}
					state.clearStartedLoading()
					frameIndex = state.frameIndex
				}

				imageSize := data.renderSize(frameIndex)

				if style.AspectRatio.IsNone() {
					style.AspectRatio = Some(float32(imageSize.Width) / float32(imageSize.Height))
				}

				if style.Size.Width.IsAuto() {
					style.Size.Width = imageSize.Width.ToLength()
					if definite, ok := style.Size.Height.AsDefinite(); ok {
						if absLength, ok := definite.AsAbsolute(); ok {
							heightPx := absLength.ToPixels(window.RemSize())
							style.Size.Width = Px(float32(imageSize.Width) * float32(heightPx) / float32(imageSize.Height)).ToLength()
						}
					}
				}

				if style.Size.Height.IsAuto() {
					style.Size.Height = imageSize.Height.ToLength()
					if definite, ok := style.Size.Width.AsDefinite(); ok {
						if absLength, ok := definite.AsAbsolute(); ok {
							widthPx := absLength.ToPixels(window.RemSize())
							style.Size.Height = Px(float32(imageSize.Height) * float32(widthPx) / float32(imageSize.Width)).ToLength()
						}
					}
				}

				if globalID != nil && data.FrameCount() > 1 && window.IsWindowActive() && !cx.ReduceMotion() {
					window.RequestAnimationFrame()
				}
			case loaded:
				if i.style.fallback != nil {
					element := i.style.fallback()
					replacementID = []LayoutID{element.RequestLayout(window, cx)}
					layoutState.replacement = Some(element)
				}
				if state != nil {
					state.clearStartedLoading()
				}
			default:
				if state != nil {
					if startedLoading, ok := state.startedLoading.Get(); ok {
						if cx.BackgroundExecutor().Now().Sub(startedLoading.at) > LoadingDelay && i.style.loading != nil {
							element := i.style.loading()
							replacementID = []LayoutID{element.RequestLayout(window, cx)}
							layoutState.replacement = Some(element)
						}
					} else {
						currentView := window.CurrentView()
						task := window.Spawn(cx, func(cx *AsyncWindowContext) (struct{}, error) {
							_, _ = cx.BackgroundExecutor().Timer(LoadingDelay).Await(cx)
							_ = cx.Update(func(_ *Window, cx *App) {
								cx.Notify(currentView)
							})
							return struct{}{}, nil
						})
						state.startedLoading = Some(imgStartedLoading{at: cx.BackgroundExecutor().Now(), task: task})
					}
				}
			}

			return window.RequestLayout(style, replacementID, cx)
		})

		layoutState.frameIndex = frameIndex

		result := None[imgState]()
		if state != nil {
			result = Some(*state)
		}
		return layoutID, result
	})
	return layoutID, layoutState
}

// Prepaint prepaints the element's interactivity and its replacement, if any.
func (i *Img) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *ImgLayoutState, window *Window, cx *App) Option[Hitbox] {
	return i.interactivity.Prepaint(globalID, inspectorID, bounds, bounds.Size, window, cx, func(_ *Style, _ Point[Pixels], hitbox Option[Hitbox], window *Window, cx *App) Option[Hitbox] {
		if replacement, ok := requestLayout.replacement.Get(); ok {
			replacement.Prepaint(window, cx)
		}
		return hitbox
	})
}

// Paint paints the image fitted into bounds, or its replacement.
func (i *Img) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], layoutState *ImgLayoutState, hitbox *Option[Hitbox], window *Window, cx *App) {
	source := i.source
	var hitboxRef *Hitbox
	if hitbox.IsSome() {
		hitboxRef = hitbox.AsMut()
	}
	i.interactivity.Paint(globalID, inspectorID, bounds, hitboxRef, window, cx, func(style *Style, window *Window, cx *App) {
		data, loaded, err := imageSourceUseData(source, i.currentImageCache(window), window, cx)
		if loaded && err == nil {
			if data.FrameCount() == 0 {
				return
			}
			newBounds := i.style.objectFit.GetBounds(bounds, data.Size(layoutState.frameIndex))
			cornerRadii := CornersAbsoluteLengthToPixels(style.CornerRadii, window.RemSize())
			if err := window.PaintImage(bounds, newBounds, cornerRadii, data, layoutState.frameIndex, i.style.grayscale); err != nil {
				gpuiutil.LogErr(err)
			}
		} else if replacement, ok := layoutState.replacement.Get(); ok {
			replacement.Paint(window, cx)
		}
	})
}

// Style returns the element's base style.
func (i *Img) Style() *StyleRefinement {
	return &i.interactivity.BaseStyle
}

// Interactivity returns the element's interactivity.
func (i *Img) Interactivity() *Interactivity {
	return &i.interactivity
}

// IntoAnyElement converts the image into an AnyElement.
func (i *Img) IntoAnyElement() AnyElement {
	return NewAnyElement[ImgLayoutState, Option[Hitbox]](i)
}

type imageDecoderSource struct {
	image *Image
}

// Hash feeds the image's ID to h, as Image's Hash impl does.
func (s imageDecoderSource) Hash(h *FxHasher) {
	h.WriteU64(s.image.ID)
}

type imageDecoderAsset struct{}

// Load decodes the image.
func (imageDecoderAsset) Load(source imageDecoderSource, cx *App) func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
	renderer := cx.SvgRenderer()
	return func(*scheduler.Scope) scheduler.Result[*RenderImage] {
		image, err := source.image.ToImageData(renderer)
		if err != nil {
			return scheduler.Result[*RenderImage]{Err: &ImageCacheError{Kind: ImageCacheErrorKindOther, Err: err}}
		}
		return scheduler.Result[*RenderImage]{Value: image}
	}
}

type imageDecoderLogger = AssetLogger[imageDecoderAsset, imageDecoderSource, *RenderImage]

func imageSourceUseData(source ImageSource, cache Option[AnyImageCache], window *Window, cx *App) (*RenderImage, bool, error) {
	switch source := source.(type) {
	case ImageSourceResource:
		if cache, ok := cache.Get(); ok {
			return cache.Load(source.Resource, window, cx)
		}
		result, ok := window.UseAsset[ImgResourceLoader](source.Resource, cx)
		return result.Value, ok, result.Err
	case ImageSourceCustom:
		return source.Load(window, cx)
	case ImageSourceRender:
		return source.Image, true, nil
	case ImageSourceImage:
		result, ok := window.UseAsset[imageDecoderLogger](imageDecoderSource{image: source.Image}, cx)
		return result.Value, ok, result.Err
	}
	panic(fmt.Sprintf("gpui: unknown image source %T", source))
}

func imageSourceGetData(source ImageSource, cache Option[AnyImageCache], window *Window, cx *App) (*RenderImage, bool, error) {
	switch source := source.(type) {
	case ImageSourceResource:
		if cache, ok := cache.Get(); ok {
			return cache.Load(source.Resource, window, cx)
		}
		result, ok := window.GetAsset[ImgResourceLoader](source.Resource, cx)
		return result.Value, ok, result.Err
	case ImageSourceCustom:
		return source.Load(window, cx)
	case ImageSourceRender:
		return source.Image, true, nil
	case ImageSourceImage:
		result, ok := window.GetAsset[imageDecoderLogger](imageDecoderSource{image: source.Image}, cx)
		return result.Value, ok, result.Err
	}
	panic(fmt.Sprintf("gpui: unknown image source %T", source))
}

func imageSourceRemoveAsset(source ImageSource, cx *App) {
	switch source := source.(type) {
	case ImageSourceResource:
		cx.RemoveAsset[ImgResourceLoader](source.Resource)
	case ImageSourceImage:
		cx.RemoveAsset[imageDecoderLogger](imageDecoderSource{image: source.Image})
	}
}

func imageSourceIsAssetCached(source ImageSource, cx *App) bool {
	switch source := source.(type) {
	case ImageSourceResource:
		return cx.HasAsset[ImgResourceLoader](source.Resource)
	case ImageSourceImage:
		return cx.HasAsset[imageDecoderLogger](imageDecoderSource{image: source.Image})
	}
	return false
}

// ImageAssetLoader is an image loader for the GPUI asset system.
type ImageAssetLoader struct{}

// Load loads and decodes the image at source.
func (ImageAssetLoader) Load(source Resource, cx *App) func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
	client := cx.HTTPClient()
	svgRenderer := cx.SvgRenderer()
	assetSource := cx.AssetSource()
	fail := func(err *ImageCacheError) scheduler.Result[*RenderImage] {
		return scheduler.Result[*RenderImage]{Err: err}
	}
	return func(s *scheduler.Scope) scheduler.Result[*RenderImage] {
		var data []byte
		switch source.Kind {
		case ResourceKindPath:
			read, err := os.ReadFile(source.Path)
			if err != nil {
				return fail(&ImageCacheError{Kind: ImageCacheErrorKindIo, Err: err})
			}
			data = read
		case ResourceKindURI:
			uri := source.URI
			request, err := http.NewRequest(http.MethodGet, string(uri), nil)
			if err != nil {
				return fail(&ImageCacheError{Kind: ImageCacheErrorKindOther, Err: fmt.Errorf("loading image asset from %q: %w", string(uri), err)})
			}
			response, err := client.Send(request).Await(s)
			if err != nil {
				return fail(&ImageCacheError{Kind: ImageCacheErrorKindOther, Err: fmt.Errorf("loading image asset from %q: %w", string(uri), err)})
			}
			body, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				return fail(&ImageCacheError{Kind: ImageCacheErrorKindIo, Err: err})
			}
			if response.StatusCode < 200 || response.StatusCode > 299 {
				text := strings.ToValidUTF8(string(body), string(utf8.RuneError))
				firstLine, _, _ := strings.Cut(text, "\n")
				firstLine = strings.TrimSuffix(firstLine, "\r")
				firstLine = strings.TrimRightFunc(firstLine, isRustWhitespace)
				return fail(&ImageCacheError{
					Kind:   ImageCacheErrorKindBadStatus,
					URI:    uri,
					Status: response.StatusCode,
					Body:   firstLine,
				})
			}
			data = body
		case ResourceKindEmbedded:
			loaded, ok, err := assetSource.Load(string(source.Embedded))
			if err != nil || !ok {
				return fail(&ImageCacheError{
					Kind:  ImageCacheErrorKindAsset,
					Asset: SharedString(fmt.Sprintf("Embedded resource not found: %s", source.Embedded)),
				})
			}
			data = bytes.Clone(loaded)
		}

		if format, ok := guessImageFormat(data); ok {
			var frames []ImageFrame
			var err error
			switch format {
			case imageCrateFormatGif:
				frames, err = decodeGifImage(data)
			case imageCrateFormatWebP:
				frames, err = decodeWebpImage(data)
			default:
				frames, err = decodeStaticImageCrateFormat(data, format)
			}
			if err != nil {
				return fail(&ImageCacheError{Kind: ImageCacheErrorKindOther, Err: err})
			}

			return scheduler.Result[*RenderImage]{Value: NewRenderImage(frames)}
		}
		image, err := svgRenderer.RenderSingleFrame(data, 1.0)
		if err != nil {
			return fail(&ImageCacheError{Kind: ImageCacheErrorKindUsvg, Err: err})
		}
		return scheduler.Result[*RenderImage]{Value: image}
	}
}

func isRustWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0x200E, 0x200F, 0x2028, 0x2029:
		return true
	}
	return r == 0xA0 || r == 0x1680 || (r >= 0x2000 && r <= 0x200A) || r == 0x202F || r == 0x205F || r == 0x3000
}

type imageCrateFormat uint8

const (
	imageCrateFormatPng imageCrateFormat = iota
	imageCrateFormatJpeg
	imageCrateFormatGif
	imageCrateFormatWebP
	imageCrateFormatPnm
	imageCrateFormatTiff
	imageCrateFormatTga
	imageCrateFormatDds
	imageCrateFormatBmp
	imageCrateFormatIco
	imageCrateFormatHdr
	imageCrateFormatOpenExr
	imageCrateFormatFarbfeld
	imageCrateFormatAvif
	imageCrateFormatQoi
)

var imageCrateFormatNames = [...]string{"Png", "Jpeg", "Gif", "WebP", "Pnm", "Tiff", "Tga", "Dds", "Bmp", "Ico", "Hdr", "OpenExr", "Farbfeld", "Avif", "Qoi"}

func (f imageCrateFormat) String() string {
	return variantName(imageCrateFormatNames[:], f, "ImageFormat")
}

var imageMagicBytes = [...]struct {
	signature []byte
	mask      []byte
	format    imageCrateFormat
}{
	{[]byte("\x89PNG\r\n\x1a\n"), nil, imageCrateFormatPng},
	{[]byte{0xff, 0xd8, 0xff}, nil, imageCrateFormatJpeg},
	{[]byte("GIF89a"), nil, imageCrateFormatGif},
	{[]byte("GIF87a"), nil, imageCrateFormatGif},
	{[]byte("RIFF\x00\x00\x00\x00WEBP"), []byte("\xFF\xFF\xFF\xFF\x00\x00\x00\x00"), imageCrateFormatWebP},
	{[]byte("MM\x00*"), nil, imageCrateFormatTiff},
	{[]byte("II*\x00"), nil, imageCrateFormatTiff},
	{[]byte("DDS "), nil, imageCrateFormatDds},
	{[]byte("BM"), nil, imageCrateFormatBmp},
	{[]byte{0, 0, 1, 0}, nil, imageCrateFormatIco},
	{[]byte("#?RADIANCE"), nil, imageCrateFormatHdr},
	{[]byte("\x00\x00\x00\x00ftypavif"), []byte("\xFF\xFF\x00\x00"), imageCrateFormatAvif},
	{[]byte{0x76, 0x2f, 0x31, 0x01}, nil, imageCrateFormatOpenExr},
	{[]byte("qoif"), nil, imageCrateFormatQoi},
	{[]byte("P1"), nil, imageCrateFormatPnm},
	{[]byte("P2"), nil, imageCrateFormatPnm},
	{[]byte("P3"), nil, imageCrateFormatPnm},
	{[]byte("P4"), nil, imageCrateFormatPnm},
	{[]byte("P5"), nil, imageCrateFormatPnm},
	{[]byte("P6"), nil, imageCrateFormatPnm},
	{[]byte("P7"), nil, imageCrateFormatPnm},
	{[]byte("farbfeld"), nil, imageCrateFormatFarbfeld},
}

func guessImageFormat(buffer []byte) (imageCrateFormat, bool) {
	for _, magic := range imageMagicBytes {
		if len(magic.mask) == 0 {
			if bytes.HasPrefix(buffer, magic.signature) {
				return magic.format, true
			}
			continue
		}
		if len(buffer) < len(magic.signature) {
			continue
		}
		matches := true
		for i, sig := range magic.signature {
			mask := byte(0xFF)
			if i < len(magic.mask) {
				mask = magic.mask[i]
			}
			if buffer[i]&mask != sig {
				matches = false
				break
			}
		}
		if matches {
			return magic.format, true
		}
	}
	return 0, false
}

func decodeStaticImageCrateFormat(data []byte, format imageCrateFormat) ([]ImageFrame, error) {
	var gpuiFormat ImageFormat
	switch format {
	case imageCrateFormatPng:
		gpuiFormat = ImageFormatPng
	case imageCrateFormatJpeg:
		gpuiFormat = ImageFormatJpeg
	case imageCrateFormatTiff:
		gpuiFormat = ImageFormatTiff
	case imageCrateFormatBmp:
		gpuiFormat = ImageFormatBmp
	case imageCrateFormatIco:
		gpuiFormat = ImageFormatIco
	case imageCrateFormatPnm:
		gpuiFormat = ImageFormatPnm
	default:
		return nil, fmt.Errorf("creating image decoder: %w", fmt.Errorf("The image format %v is not supported", format))
	}
	return decodeStaticImage(data, gpuiFormat)
}

// ImageCacheErrorKind tells which variant an ImageCacheError is.
type ImageCacheErrorKind uint8

const (
	// ImageCacheErrorKindOther is some other kind of error.
	ImageCacheErrorKindOther ImageCacheErrorKind = iota
	// ImageCacheErrorKindIo is an error that occurred while reading the image from disk.
	ImageCacheErrorKindIo
	// ImageCacheErrorKindBadStatus is an unexpected HTTP status while fetching the image.
	ImageCacheErrorKindBadStatus
	// ImageCacheErrorKindAsset is an error that occurred while processing an asset.
	ImageCacheErrorKindAsset
	// ImageCacheErrorKindImage is an error that occurred while processing an image.
	ImageCacheErrorKindImage
	// ImageCacheErrorKindUsvg is an error that occurred while processing an SVG.
	ImageCacheErrorKindUsvg
)

// ImageCacheError is an error that can occur when interacting with the image cache.
type ImageCacheError struct {
	// Kind is the variant.
	Kind ImageCacheErrorKind
	// Err is the cause of the Other, Io, Image and Usvg variants.
	Err error
	// URI is the URI of the image, for BadStatus.
	URI SharedURI
	// Status is the HTTP status code, for BadStatus.
	Status int
	// Body is the first line of the HTTP response body, for BadStatus.
	Body string
	// Asset is the message of the Asset variant.
	Asset SharedString
}

// Error formats the error as its Display impl does.
func (e *ImageCacheError) Error() string {
	switch e.Kind {
	case ImageCacheErrorKindIo:
		return fmt.Sprintf("IO error: %v", e.Err)
	case ImageCacheErrorKindBadStatus:
		return fmt.Sprintf("unexpected http status for %s: %d %s, body: %s", e.URI, e.Status, http.StatusText(e.Status), e.Body)
	case ImageCacheErrorKindAsset:
		return fmt.Sprintf("asset error: %s", e.Asset)
	case ImageCacheErrorKindImage:
		return fmt.Sprintf("image error: %v", e.Err)
	case ImageCacheErrorKindUsvg:
		return fmt.Sprintf("svg error: %v", e.Err)
	default:
		return fmt.Sprintf("error: %v", e.Err)
	}
}

// Unwrap returns the cause of the error, if any.
func (e *ImageCacheError) Unwrap() error {
	return e.Err
}
