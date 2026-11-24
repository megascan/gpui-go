package gpui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xo/okage"
	okagepath "github.com/xo/okage/path"
	"github.com/xo/okage/svg"
	"golang.org/x/image/webp"
)

func emojiFontFamilies() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"Apple Color Emoji", ".AppleColorEmojiUI"}
	case "windows":
		return []string{"Segoe UI Emoji", "Segoe UI Symbol"}
	case "linux", "freebsd":
		return []string{"Noto Color Emoji", "Emoji One", "Twitter Color Emoji", "JoyPixels"}
	default:
		return nil
	}
}

var emojiPresentationRanges = [...][2]rune{
	{0x231A, 0x231B}, {0x23E9, 0x23EC}, {0x23F0, 0x23F0}, {0x23F3, 0x23F3}, {0x25FD, 0x25FE}, {0x2614, 0x2615},
	{0x2648, 0x2653}, {0x267F, 0x267F}, {0x2693, 0x2693}, {0x26A1, 0x26A1}, {0x26AA, 0x26AB}, {0x26BD, 0x26BE},
	{0x26C4, 0x26C5}, {0x26CE, 0x26CE}, {0x26D4, 0x26D4}, {0x26EA, 0x26EA}, {0x26F2, 0x26F3}, {0x26F5, 0x26F5},
	{0x26FA, 0x26FA}, {0x26FD, 0x26FD}, {0x2705, 0x2705}, {0x270A, 0x270B}, {0x2728, 0x2728}, {0x274C, 0x274C},
	{0x274E, 0x274E}, {0x2753, 0x2755}, {0x2757, 0x2757}, {0x2795, 0x2797}, {0x27B0, 0x27B0}, {0x27BF, 0x27BF},
	{0x2B1B, 0x2B1C}, {0x2B50, 0x2B50}, {0x2B55, 0x2B55}, {0x1F004, 0x1F004}, {0x1F0CF, 0x1F0CF}, {0x1F18E, 0x1F18E},
	{0x1F191, 0x1F19A}, {0x1F1E6, 0x1F1FF}, {0x1F201, 0x1F201}, {0x1F21A, 0x1F21A}, {0x1F22F, 0x1F22F}, {0x1F232, 0x1F236},
	{0x1F238, 0x1F23A}, {0x1F250, 0x1F251}, {0x1F300, 0x1F320}, {0x1F32D, 0x1F335}, {0x1F337, 0x1F37C}, {0x1F37E, 0x1F393},
	{0x1F3A0, 0x1F3CA}, {0x1F3CF, 0x1F3D3}, {0x1F3E0, 0x1F3F0}, {0x1F3F4, 0x1F3F4}, {0x1F3F8, 0x1F43E}, {0x1F440, 0x1F440},
	{0x1F442, 0x1F4FC}, {0x1F4FF, 0x1F53D}, {0x1F54B, 0x1F54E}, {0x1F550, 0x1F567}, {0x1F57A, 0x1F57A}, {0x1F595, 0x1F596},
	{0x1F5A4, 0x1F5A4}, {0x1F5FB, 0x1F64F}, {0x1F680, 0x1F6C5}, {0x1F6CC, 0x1F6CC}, {0x1F6D0, 0x1F6D2}, {0x1F6D5, 0x1F6D7},
	{0x1F6DC, 0x1F6DF}, {0x1F6EB, 0x1F6EC}, {0x1F6F4, 0x1F6FC}, {0x1F7E0, 0x1F7EB}, {0x1F7F0, 0x1F7F0}, {0x1F90C, 0x1F93A},
	{0x1F93C, 0x1F945}, {0x1F947, 0x1F9FF}, {0x1FA70, 0x1FA7C}, {0x1FA80, 0x1FA89}, {0x1FA8F, 0x1FAC6}, {0x1FACE, 0x1FADC},
	{0x1FADF, 0x1FAE9}, {0x1FAF0, 0x1FAF8},
}

func isEmojiPresentation(c rune) bool {
	i := sort.Search(len(emojiPresentationRanges), func(i int) bool { return emojiPresentationRanges[i][1] >= c })
	return i < len(emojiPresentationRanges) && emojiPresentationRanges[i][0] <= c
}

func fontHasChar(db *svg.FontDatabase, id svg.FontID, ch rune) bool {
	_, ok := db.NominalGlyph(id, ch)
	return ok
}

func normalFontQuery(families ...svg.FontFamily) svg.FontQuery {
	return svg.FontQuery{
		Families: families,
		Weight:   400,
		Stretch:  svg.FontStretchNormal,
		Style:    svg.FontStyleNormal,
	}
}

func selectEmojiFont(ch rune, fonts []svg.FontID, db *svg.FontDatabase, families []string) (svg.FontID, bool) {
	for _, familyName := range families {
		id, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: svg.FontFamilyNamed, Name: familyName}))
		if !ok {
			continue
		}
		if slices.Contains(fonts, id) || !fontHasChar(db, id, ch) {
			continue
		}
		return id, true
	}
	return 0, false
}

// SmoothSvgScaleFactor is the factor SVGs are rendered at, above their size, to get a
// higher-quality result.
const SmoothSvgScaleFactor float32 = 2.

// RenderSvgParams identifies a rasterized SVG in the sprite atlas.
type RenderSvgParams struct {
	// Path is the SVG's asset path.
	Path SharedString
	// Size is the rasterized size.
	Size Size[DevicePixels]
}

// SvgRenderer holds everything necessary to render SVGs. Copies share their state.
type SvgRenderer struct {
	assetSource AssetSource
	usvgOptions []svg.Option
}

// ParsedSvg is a parsed SVG document that can be rasterized at any scale.
//
// Produced by SvgRenderer.ParseSvg and rasterized by SvgRenderer.RenderParsed. Parsing resolves
// fonts and converts text to paths, so callers that need to rasterize the same SVG at multiple
// scales should retain this value to avoid re-paying the parse cost.
type ParsedSvg struct {
	tree *svg.Tree
}

// SvgTextFont is the font of an SVG `<text>` element, as the values of its presentation
// attributes.
type SvgTextFont struct {
	// Family is the `font-family` list.
	Family string
	// Size is the `font-size`, in pixels.
	Size float32
	// Weight is the `font-weight`.
	Weight string
	// Style is the `font-style`.
	Style string
}

type svgSizeKind uint8

const (
	svgSizeKindSize svgSizeKind = iota
	svgSizeKindExactSize
	svgSizeKindScaleFactor
)

// SvgSize is the size in which to rasterize the SVG. It is a comparable tagged struct.
type SvgSize struct {
	kind        svgSizeKind
	size        Size[DevicePixels]
	scaleFactor float32
}

// SvgSizeSize is a width in device pixels. The SVG retains its aspect ratio.
func SvgSizeSize(size Size[DevicePixels]) SvgSize {
	return SvgSize{kind: svgSizeKindSize, size: size}
}

// SvgSizeExactSize is an exact width and height in device pixels.
func SvgSizeExactSize(size Size[DevicePixels]) SvgSize {
	return SvgSize{kind: svgSizeKindExactSize, size: size}
}

// SvgSizeScaleFactor is a logical scaling factor to apply to the size provided by the SVG.
func SvgSizeScaleFactor(scaleFactor float32) SvgSize {
	return SvgSize{kind: svgSizeKindScaleFactor, scaleFactor: scaleFactor}
}

type enrichedFontDatabase struct {
	once        sync.Once
	initialized atomic.Bool
	fontdb      *fontDatabase
}

func (e *enrichedFontDatabase) getOrInit(init func() *fontDatabase) *fontDatabase {
	e.once.Do(func() {
		e.fontdb = init()
		e.initialized.Store(true)
	})
	return e.fontdb
}

func (e *enrichedFontDatabase) get() (*fontDatabase, bool) {
	if !e.initialized.Load() {
		return nil, false
	}
	return e.fontdb, true
}

var systemFontDB = sync.OnceValue(func() *fontFileDatabase {
	db := newFontFileDatabase()
	db.loadSystemFonts()
	return db
})

// NewSvgRenderer creates a new SVG renderer with the provided asset source.
func NewSvgRenderer(assetSource AssetSource) SvgRenderer {
	if assetSource == nil {
		assetSource = NoAssets{}
	}
	enrichedFontdb := &enrichedFontDatabase{}

	defaultFontResolver := svg.DefaultFontSelector(slog.Default())
	fontResolver := func(font *svg.Font, db *svg.FontDatabase) (svg.FontID, bool) {
		fontdb := enrichedFontdb.getOrInit(func() *fontDatabase {
			files := systemFontDB().clone()
			loadBundledFonts(assetSource, files.database)
			fixGenericFontFamilies(files.database)
			return &fontDatabase{
				database:            files.database,
				faceFiles:           files.faceFiles,
				ownedFaces:          map[svg.FontID]svg.FontID{},
				remainingOwnedBytes: maxOwnedFontBytes,
				fallbackFaces:       map[fallbackFaceKey]Option[svg.FontID]{},
			}
		})
		if db.IsEmpty() {
			fontdb.mu.Lock()
			*db = *fontdb.database.Clone()
			fontdb.mu.Unlock()
		}
		id, ok := defaultFontResolver(font, db)
		if !ok {
			id, ok = db.Query(normalFontQuery(svg.FontFamily{Kind: svg.FontFamilySansSerif}))
			if !ok {
				for face := range db.Faces() {
					id, ok = face.ID, true
					break
				}
			}
		}
		if !ok {
			return 0, false
		}
		fontdb.mu.Lock()
		defer fontdb.mu.Unlock()
		return fontdb.ownedFace(db, id), true
	}
	defaultFallbackSelection := svg.DefaultFallbackSelector(slog.Default())
	fallbackSelection := func(ch rune, fonts []svg.FontID, db *svg.FontDatabase) (svg.FontID, bool) {
		selectFn := func(fonts []svg.FontID) (svg.FontID, bool) {
			if isEmojiPresentation(ch) {
				if id, ok := selectEmojiFont(ch, fonts, db, emojiFontFamilies()); ok {
					return id, true
				}
			}
			return defaultFallbackSelection(ch, fonts, db)
		}
		fontdb, ok := enrichedFontdb.get()
		if !ok {
			return selectFn(fonts)
		}
		face, ok := cachedFallbackFace(fontdb, ch, fonts, selectFn)
		if !ok {
			return 0, false
		}
		fontdb.mu.Lock()
		ownedFace := fontdb.ownedFace(db, face)
		fontdb.mu.Unlock()
		if slices.Contains(fonts, ownedFace) {
			return face, true
		}
		return ownedFace, true
	}
	options := []svg.Option{
		svg.WithFontResolver(svg.FontResolver{
			SelectFont:     fontResolver,
			SelectFallback: fallbackSelection,
		}),
	}
	return SvgRenderer{
		assetSource: assetSource,
		usvgOptions: options,
	}
}

func (r SvgRenderer) parseOptions() []svg.Option {
	budget := newImageDecodeBudget()
	var resolveData svg.ImageHrefDataResolverFn
	options := append(slices.Clip(r.usvgOptions), svg.WithImageHrefResolver(svg.ImageHrefResolver{
		ResolveData: func(mime string, data []byte) (svg.ImageKind, bool) {
			return resolveData(mime, data)
		},
		ResolveString: func(string) (svg.ImageKind, bool) {
			return nil, false
		},
	}))
	var defaultData svg.ImageHrefDataResolverFn
	resolveData = func(mime string, data []byte) (svg.ImageKind, bool) {
		if defaultData == nil {
			defaultData = svg.DefaultDataResolver(options...)
		}
		if r.checkTextPaths(data) != nil {
			return nil, false
		}
		kind, ok := defaultData(mime, data)
		if !ok || reserveSvgRasterImage(budget, kind) != nil {
			return nil, false
		}
		return kind, true
	}
	return options
}

func reserveSvgRasterImage(budget *imageDecodeBudget, kind svg.ImageKind) error {
	var config image.Config
	var err error
	buffers := []uint64{4}
	switch data := kind.(type) {
	case svg.ImageKindPNG:
		config, err = png.DecodeConfig(bytes.NewReader(data))
		buffers = []uint64{colorModelBytesPerPixel(config.ColorModel)}
	case svg.ImageKindJPEG:
		config, err = jpeg.DecodeConfig(bytes.NewReader(data))
		buffers = jpegDecodeBuffers(data, config)
	case svg.ImageKindGIF:
		config, err = gif.DecodeConfig(bytes.NewReader(data))
		buffers = []uint64{1}
	case svg.ImageKindWEBP:
		config, err = webp.DecodeConfig(bytes.NewReader(data))
	default:
		return nil
	}
	if err != nil {
		return err
	}
	if config.Width < 0 || config.Height < 0 {
		return errImageLimitsExceeded
	}
	width, height := uint64(config.Width), uint64(config.Height)
	for _, bytesPerPixel := range append(buffers, 4, 4) {
		if err := budget.reserve(width, height, bytesPerPixel); err != nil {
			return err
		}
	}
	return nil
}

const (
	svgFilterMaxWork        float64 = 1 << 28
	svgFilterMaxRegionArea  float64 = 1 << 26
	svgNestedImageMaxPixels float64 = 1 << 27
)

var (
	errSvgFilterWorkExceeded  = errors.New("SVG filters exceed the rendering work limit")
	errSvgNestedImageExceeded = errors.New("SVG nested images exceed the rendering pixel limit")
)

type svgFilterWork struct {
	canvasLeft, canvasTop, canvasRight, canvasBottom float64
	canvasArea                                       float64
	total                                            float64
	nested                                           float64
	regionExceeded                                   bool
}

func newSvgFilterWork(width, height uint32) *svgFilterWork {
	w, h := float64(width), float64(height)
	return &svgFilterWork{canvasLeft: -2 * w, canvasTop: -2 * h, canvasRight: 3 * w, canvasBottom: 3 * h, canvasArea: w * h}
}

func (w *svgFilterWork) err() error {
	switch {
	case w.regionExceeded || w.total > svgFilterMaxWork:
		return errSvgFilterWorkExceeded
	case w.nested > svgNestedImageMaxPixels:
		return errSvgNestedImageExceeded
	}
	return nil
}

func checkSvgRenderWork(tree *svg.Tree, transform svg.Transform, width, height uint32) error {
	work := newSvgFilterWork(width, height)
	work.walk(tree.Root(), transform, false, work.canvasArea)
	return work.err()
}

func (w *svgFilterWork) region(filter *svg.Filter, transform svg.Transform, full bool) (float64, float64) {
	if full {
		return w.canvasRight - w.canvasLeft, w.canvasBottom - w.canvasTop
	}
	rect, ok := filter.Rect().Transform(transform)
	if !ok {
		return 0, 0
	}
	left := max(float64(rect.Left()), w.canvasLeft)
	top := max(float64(rect.Top()), w.canvasTop)
	right := min(float64(rect.Right()), w.canvasRight)
	bottom := min(float64(rect.Bottom()), w.canvasBottom)
	if right <= left || bottom <= top {
		return 0, 0
	}
	return math.Ceil(right - left), math.Ceil(bottom - top)
}

func svgFilterPrimitiveWork(kind svg.KindFilter, transform svg.Transform, width, height float64) float64 {
	sx, sy := transform.GetScale()
	switch k := kind.(type) {
	case *svg.GaussianBlurFilter, *svg.DropShadowFilter:
		return 5
	case *svg.TurbulenceFilter:
		return 1 + float64(k.NumOctaves())/2
	case *svg.MorphologyFilter:
		rx := min(math.Round(float64(k.RadiusX())*math.Abs(float64(sx))), width)
		ry := min(math.Round(float64(k.RadiusY())*math.Abs(float64(sy))), height)
		return 1 + (2*rx+1)*(2*ry+1)/10
	case *svg.ConvolveMatrixFilter:
		return 1 + float64(k.Matrix().Columns())*float64(k.Matrix().Rows())/16
	}
	return 1
}

func (w *svgFilterWork) walk(group *svg.Group, base svg.Transform, full bool, layer float64) {
	transform := base.PreConcat(group.AbsTransform())
	for _, filter := range group.Filters() {
		width, height := w.region(filter, transform, full)
		if width*height > svgFilterMaxRegionArea {
			w.regionExceeded = true
		}
		layer = max(layer, width*height)
		work := 1.0
		for _, primitive := range filter.Primitives() {
			work += svgFilterPrimitiveWork(primitive.Kind(), transform, width, height)
		}
		w.total += work * width * height
	}
	group.Subroots(func(g *svg.Group) {
		w.walk(g, transform, full, layer)
	})
	for _, child := range group.Children() {
		switch c := child.(type) {
		case *svg.Group:
			w.walk(c, base, full, layer)
		case *svg.Text:
			c.Subroots(func(g *svg.Group) {
				w.walk(g, base, full, layer)
			})
		case *svg.Image:
			if _, ok := c.Kind().(svg.ImageKindSVG); ok {
				w.nested += layer
			}
			c.Subroots(func(g *svg.Group) {
				w.walk(g, transform, true, layer)
			})
		default:
			c.Subroots(func(g *svg.Group) {
				w.walk(g, transform, true, layer)
			})
		}
	}
}

// ParseSvg parses SVG data into a ParsedSvg that can be rasterized at any scale.
func (r SvgRenderer) ParseSvg(bytes []byte) (*ParsedSvg, error) {
	tree, err := r.parseTree(bytes)
	if err != nil {
		return nil, err
	}
	return &ParsedSvg{tree: tree}, nil
}

// MeasureTextWidth returns the advance of text drawn by this renderer as a single line of SVG
// `<text>`.
func (r SvgRenderer) MeasureTextWidth(text string, font SvgTextFont) (float32, error) {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg"><text xml:space="preserve" text-decoration="underline" font-family="`)
	pushXMLEscaped(&b, font.Family)
	b.WriteString(`" font-weight="`)
	pushXMLEscaped(&b, font.Weight)
	b.WriteString(`" font-style="`)
	pushXMLEscaped(&b, font.Style)
	b.WriteString(`" font-size="` + strconv.FormatFloat(float64(font.Size), 'f', -1, 32) + `">`)
	pushXMLEscaped(&b, text)
	b.WriteString("</text></svg>")

	tree, err := svg.FromStr(b.String(), r.parseOptions()...)
	if err != nil {
		return 0, err
	}
	return underlineWidth(tree.Root()), nil
}

// RenderParsed rasterizes a previously parsed SVG into an image buffer.
func (r SvgRenderer) RenderParsed(parsed *ParsedSvg, size SvgSize) (*RenderImage, error) {
	imageScaleFactor := float32(1.0)
	if size.kind == svgSizeKindScaleFactor {
		size = SvgSizeScaleFactor(size.scaleFactor * SmoothSvgScaleFactor)
		imageScaleFactor = SmoothSvgScaleFactor
	}
	pixmap, err := rasterizeTree(parsed.tree, size)
	if err != nil {
		return nil, err
	}
	width, height := int(pixmap.Width()), int(pixmap.Height())
	buffer := pixmap.Take()

	for i := 0; i+4 <= len(buffer); i += 4 {
		SwapRgbaPaToBgra(buffer[i : i+4])
	}

	frame := &image.RGBA{Pix: buffer, Stride: 4 * width, Rect: image.Rect(0, 0, width, height)}
	img := NewRenderImage([]ImageFrame{newImageFrame(frame)})
	img.scaleFactor = imageScaleFactor
	return img, nil
}

// RenderSingleFrame renders the given bytes into an image buffer.
func (r SvgRenderer) RenderSingleFrame(bytes []byte, scaleFactor float32) (*RenderImage, error) {
	parsed, err := r.ParseSvg(bytes)
	if err != nil {
		return nil, err
	}
	return r.RenderParsed(parsed, SvgSizeScaleFactor(scaleFactor))
}

func (r SvgRenderer) renderAlphaMask(params RenderSvgParams, bytes []byte) (Size[DevicePixels], []byte, bool, error) {
	if params.Size.IsZero() {
		return Size[DevicePixels]{}, nil, false, errors.New("can't render at a zero size")
	}

	renderPixmap := func(bytes []byte) (Size[DevicePixels], []byte, bool, error) {
		pixmap, err := r.renderPixmap(bytes, SvgSizeSize(params.Size))
		if err != nil {
			return Size[DevicePixels]{}, nil, false, err
		}

		size := NewSize(DevicePixels(int32(pixmap.Width())), DevicePixels(int32(pixmap.Height())))
		data := pixmap.Data()
		alphaMask := make([]byte, 0, len(data)/4)
		for i := 3; i < len(data); i += 4 {
			alphaMask = append(alphaMask, data[i])
		}

		return size, alphaMask, true, nil
	}

	if bytes != nil {
		return renderPixmap(bytes)
	}
	loaded, ok, err := r.assetSource.Load(string(params.Path))
	if err != nil {
		return Size[DevicePixels]{}, nil, false, err
	}
	if ok {
		return renderPixmap(loaded)
	}
	return Size[DevicePixels]{}, nil, false, nil
}

func (r SvgRenderer) parseTree(data []byte) (*svg.Tree, error) {
	if err := r.checkTextPaths(data); err != nil {
		return nil, err
	}
	return svg.FromData(data, r.parseOptions()...)
}

func (r SvgRenderer) renderPixmap(bytes []byte, size SvgSize) (*okage.Pixmap, error) {
	tree, err := r.parseTree(bytes)
	if err != nil {
		return nil, err
	}
	return rasterizeTree(tree, size)
}

func f32AsU32(v float32) uint32 {
	switch {
	case math.IsNaN(float64(v)) || v <= 0:
		return 0
	case v >= math.MaxUint32:
		return math.MaxUint32
	default:
		return uint32(v)
	}
}

func rasterizeTree(tree *svg.Tree, size SvgSize) (*okage.Pixmap, error) {
	const maxSize float32 = 8192.0

	svgSize := tree.Size()
	var width, height float32
	switch size.kind {
	case svgSizeKindSize:
		scale := float32(int32(size.size.Width)) / svgSize.Width()
		width, height = svgSize.Width()*scale, svgSize.Height()*scale
	case svgSizeKindExactSize:
		width, height = float32(int32(size.size.Width)), float32(int32(size.size.Height))
	default:
		width, height = svgSize.Width()*size.scaleFactor, svgSize.Height()*size.scaleFactor
	}

	if width > maxSize {
		slog.Warn(fmt.Sprintf("Attempted to render pixmap where width (%v) > MAX_SIZE (%v)", width, maxSize))
	}
	if height > maxSize {
		slog.Warn(fmt.Sprintf("Attempted to render pixmap where height (%v) > MAX_SIZE (%v)", height, maxSize))
	}
	scale := min(min(maxSize/width, maxSize/height), 1.0)
	width *= scale
	height *= scale

	transform := okagepath.TransformFromScale(width/svgSize.Width(), height/svgSize.Height())
	if err := checkSvgRenderWork(tree, transform, f32AsU32(width), f32AsU32(height)); err != nil {
		return nil, err
	}

	pixmap, ok := okage.NewPixmap(f32AsU32(width), f32AsU32(height))
	if !ok {
		return nil, okage.ErrSVGInvalidSize
	}

	pixmapMut := pixmap.AsMut()
	svg.Render(tree, transform, &pixmapMut)

	return pixmap, nil
}

const maxOwnedFontBytes uint64 = 64 * 1024 * 1024

type fallbackFaceKey struct {
	base svg.FontID
	ch   rune
}

type fontDatabase struct {
	mu                  sync.Mutex
	database            *svg.FontDatabase
	faceFiles           map[svg.FontID]string
	ownedFaces          map[svg.FontID]svg.FontID
	remainingOwnedBytes uint64
	fallbackFaces       map[fallbackFaceKey]Option[svg.FontID]
}

func (f *fontDatabase) ownedFace(db *svg.FontDatabase, id svg.FontID) svg.FontID {
	if _, ok := db.Face(id); !ok {
		return id
	}
	path, ok := f.faceFiles[id]
	if !ok {
		return id
	}
	if _, ok := f.ownedFaces[id]; !ok {
		f.ownFontFile(path)
	}
	*db = *f.database.Clone()
	if owned, ok := f.ownedFaces[id]; ok {
		return owned
	}
	return id
}

func (f *fontDatabase) ownFontFile(path string) {
	type fileFace struct {
		id    svg.FontID
		index uint32
	}
	var fileFaces []fileFace
	for face := range f.database.Faces() {
		if facePath, ok := f.faceFiles[face.ID]; ok && facePath == path {
			fileFaces = append(fileFaces, fileFace{id: face.ID, index: face.Index})
		}
	}
	remainingOwnedBytes := f.remainingOwnedBytes
	var data []byte
	var err error
	info, err := os.Stat(path)
	if err == nil && uint64(info.Size()) <= remainingOwnedBytes {
		data, err = os.ReadFile(path)
	}
	var ownedIDs []svg.FontID
	switch {
	case err != nil:
		slog.Warn(fmt.Sprintf("Failed to read font %q: %v", path, err))
	case data != nil && uint64(len(data)) <= remainingOwnedBytes:
		size := uint64(len(data))
		ownedIDs = f.database.LoadFontData(data)
		if len(ownedIDs) != 0 {
			f.remainingOwnedBytes -= size
		}
	}
	for _, face := range fileFaces {
		ownedID := face.id
		for _, candidate := range ownedIDs {
			if info, ok := f.database.Face(candidate); ok && info.Index == face.index {
				ownedID = candidate
				break
			}
		}
		f.ownedFaces[face.id] = ownedID
	}
}

const maxCachedFallbackFaces = 16 * 1024

func cachedFallbackFace(fontdb *fontDatabase, ch rune, fonts []svg.FontID, selectFn func([]svg.FontID) (svg.FontID, bool)) (svg.FontID, bool) {
	if len(fonts) == 0 {
		return selectFn(fonts)
	}
	key := fallbackFaceKey{base: fonts[0], ch: ch}
	fontdb.mu.Lock()
	cachedFace, cached := fontdb.fallbackFaces[key]
	fontdb.mu.Unlock()
	var face Option[svg.FontID]
	if cached {
		face = cachedFace
	} else {
		if id, ok := selectFn(fonts[:1]); ok {
			face = Some(id)
		} else {
			face = None[svg.FontID]()
		}
		fontdb.mu.Lock()
		if len(fontdb.fallbackFaces) < maxCachedFallbackFaces {
			fontdb.fallbackFaces[key] = face
		}
		fontdb.mu.Unlock()
	}
	if id, ok := face.Get(); ok && slices.Contains(fonts, id) {
		return selectFn(fonts)
	}
	return face.Get()
}

func underlineWidth(group *svg.Group) float32 {
	var width float32
	for _, node := range group.Children() {
		switch node := node.(type) {
		case *svg.Text:
			var w float32
			for _, span := range node.Layouted() {
				if span.Underline != nil {
					w += span.Underline.Data().Bounds().Width()
				}
			}
			width += w
		case *svg.Group:
			width += underlineWidth(node)
		}
	}
	return width
}

func pushXMLEscaped(xml *strings.Builder, text string) {
	for _, character := range text {
		switch {
		case character == '&':
			xml.WriteString("&amp;")
		case character == '<':
			xml.WriteString("&lt;")
		case character == '>':
			xml.WriteString("&gt;")
		case character == '"':
			xml.WriteString("&quot;")
		case character == '\t' || character == '\n' || character == '\r':
			xml.WriteRune(character)
		case character <= 0x1f || character == 0xfffe || character == 0xffff:
		default:
			xml.WriteRune(character)
		}
	}
}

func loadBundledFonts(assetSource AssetSource, db *svg.FontDatabase) {
	fontPaths := []string{
		"fonts/ibm-plex-sans/IBMPlexSans-Regular.ttf",
		"fonts/lilex/Lilex-Regular.ttf",
	}
	for _, path := range fontPaths {
		data, ok, err := assetSource.Load(path)
		switch {
		case err != nil:
			slog.Warn(fmt.Sprintf("Failed to load bundled font %s: %v", path, err))
		case !ok:
			slog.Warn("Bundled font not found: " + path)
		default:
			db.LoadFontData(data)
		}
	}
}

func fixGenericFontFamilies(db *svg.FontDatabase) {
	familiesAndFallbacks := []struct {
		family   svg.FontFamilyKind
		fallback string
	}{
		{svg.FontFamilySansSerif, "IBM Plex Sans"},
		{svg.FontFamilySerif, "IBM Plex Sans"},
		{svg.FontFamilyMonospace, "Lilex"},
		{svg.FontFamilyCursive, "IBM Plex Sans"},
		{svg.FontFamilyFantasy, "IBM Plex Sans"},
	}

	for _, entry := range familiesAndFallbacks {
		if _, ok := db.Query(normalFontQuery(svg.FontFamily{Kind: entry.family})); ok {
			continue
		}
		switch entry.family {
		case svg.FontFamilySansSerif:
			db.SetSansSerifFamily(entry.fallback)
		case svg.FontFamilySerif:
			db.SetSerifFamily(entry.fallback)
		case svg.FontFamilyMonospace:
			db.SetMonospaceFamily(entry.fallback)
		case svg.FontFamilyCursive:
			db.SetCursiveFamily(entry.fallback)
		case svg.FontFamilyFantasy:
			db.SetFantasyFamily(entry.fallback)
		}
	}
}

type fontFileDatabase struct {
	database  *svg.FontDatabase
	faceFiles map[svg.FontID]string
}

func newFontFileDatabase() *fontFileDatabase {
	return &fontFileDatabase{database: svg.NewFontDatabase(), faceFiles: map[svg.FontID]string{}}
}

func (d *fontFileDatabase) clone() *fontFileDatabase {
	faceFiles := make(map[svg.FontID]string, len(d.faceFiles))
	for id, path := range d.faceFiles {
		faceFiles[id] = path
	}
	return &fontFileDatabase{database: d.database.Clone(), faceFiles: faceFiles}
}

var systemFontDirs = []string{
	"Windows/Fonts",
	"Library/Fonts",
	"System/Library/Fonts",
	"System/Library/AssetsV2/com_apple_MobileAsset_Font6",
	"Network/Library/Fonts",
	"usr/share/fonts",
	"usr/local/share/fonts",
}

var userFontDirs = []string{
	"AppData/Local/Microsoft/Windows/Fonts",
	"AppData/Roaming/Microsoft/Windows/Fonts",
	"Library/Fonts",
	".fonts",
	".local/share/fonts",
}

func (d *fontFileDatabase) loadSystemFonts() {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
		if v := filepath.VolumeName(os.Getenv("WINDIR")); v != "" {
			root = v + `\`
		}
	}
	for _, dir := range systemFontDirs {
		d.loadFontsDir(filepath.Join(root, filepath.FromSlash(dir)))
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, dir := range userFontDirs {
			d.loadFontsDir(filepath.Join(home, filepath.FromSlash(dir)))
		}
	}
}

func (d *fontFileDatabase) loadFontsDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		p := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			d.loadFontsDir(p)
			continue
		}
		switch strings.ToLower(path.Ext(entry.Name())) {
		case ".ttf", ".ttc", ".otf", ".otc":
			d.loadFontFile(p)
		}
	}
}

func (d *fontFileDatabase) loadFontFile(path string) {
	before := d.database.Len()
	d.database.LoadFontsFS(singleFileFS{dir: filepath.Dir(path), name: filepath.Base(path)}, ".")
	for id := before; id < d.database.Len(); id++ {
		d.faceFiles[svg.FontID(id)] = path
	}
}

type singleFileFS struct {
	dir  string
	name string
}

func (f singleFileFS) Open(name string) (fs.File, error) {
	if name != f.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return os.Open(filepath.Join(f.dir, f.name))
}

func (f singleFileFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name != "." {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	info, err := os.Stat(filepath.Join(f.dir, f.name))
	if err != nil {
		return nil, err
	}
	return []fs.DirEntry{fs.FileInfoToDirEntry(info)}, nil
}
