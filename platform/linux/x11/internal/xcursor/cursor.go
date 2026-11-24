//go:build linux

package xcursor

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/render"
	"github.com/jezek/xgb/xproto"

	"github.com/megascan/gpui-go/platform/linux/x11/internal/xrm"
)

type renderSupport uint8

const (
	renderSupportNone renderSupport = iota
	renderSupportStaticCursor
	renderSupportAnimatedCursor
)

// ErrInvalidValue is x11rb's ParseError::InvalidValue for an unreadable cursor file.
var ErrInvalidValue = errors.New("Invalid value")

// Handle loads cursors of the configured theme (x11rb cursor::Handle).
type Handle struct {
	root          xproto.Window
	pictureFormat render.Pictformat
	renderSupport renderSupport
	theme         string
	hasTheme      bool
	cursorSize    uint32
}

// NewHandle queries RENDER (when the extension was initialized on c) and builds a Handle
// for screen from the resource database (Handle::new(...).reply()).
func NewHandle(c *xgb.Conn, screen *xproto.ScreenInfo, db *xrm.Database) (*Handle, error) {
	var renderVersion [2]uint32
	pictureFormat := render.Pictformat(0)
	c.ExtLock.RLock()
	_, hasRender := c.Extensions["RENDER"]
	c.ExtLock.RUnlock()
	if hasRender {
		versionCookie := render.QueryVersion(c, 0, 8)
		formatsCookie := render.QueryPictFormats(c)
		version, err := versionCookie.Reply()
		if err != nil {
			return nil, err
		}
		formats, err := formatsCookie.Reply()
		if err != nil {
			return nil, err
		}
		renderVersion = [2]uint32{version.MajorVersion, version.MinorVersion}
		pictureFormat = findFormat(formats)
	}
	return fromReplies(screen, db, renderVersion, pictureFormat), nil
}

func fromReplies(screen *xproto.ScreenInfo, db *xrm.Database, renderVersion [2]uint32, pictureFormat render.Pictformat) *Handle {
	support := renderSupportNone
	if renderVersion[0] >= 1 || renderVersion[1] >= 8 {
		support = renderSupportAnimatedCursor
	} else if renderVersion[0] >= 1 || renderVersion[1] >= 5 {
		support = renderSupportStaticCursor
	}
	theme, hasTheme := os.LookupEnv("XCURSOR_THEME")
	if !hasTheme {
		theme, hasTheme = db.GetString("Xcursor.theme", "")
	}
	cursorSize, ok, err := db.GetUint32("Xcursor.size", "")
	if !ok || err != nil {
		cursorSize = 0
	}
	xftDpi, ok, err := db.GetUint32("Xft.dpi", "")
	if !ok || err != nil {
		xftDpi = 0
	}
	return &Handle{
		root:          screen.Root,
		pictureFormat: pictureFormat,
		renderSupport: support,
		theme:         theme,
		hasTheme:      hasTheme,
		cursorSize:    getCursorSize(cursorSize, xftDpi, screen),
	}
}

// LoadCursor loads the named cursor; it returns 0 (NONE) when the theme has no such cursor.
func (h *Handle) LoadCursor(c *xgb.Conn, name string) (xproto.Cursor, error) {
	return loadCursor(c, h, name)
}

type cursorFile struct {
	coreChar uint16
	isCore   bool
	file     *os.File
}

func openCursor(theme string, hasTheme bool, name string) (cursorFile, bool) {
	if hasTheme {
		if f, ok := findCursor(theme, name); ok {
			return f, true
		}
	}
	return findCursor("default", name)
}

func findCursor(theme, name string) (cursorFile, bool) {
	if theme == "core" {
		if id, ok := cursorShapeToID(name); ok {
			return cursorFile{coreChar: id, isCore: true}, true
		}
	}
	if path, ok := LoadCursorTheme(theme).LoadIcon(name); ok {
		if f, err := os.Open(path); err == nil {
			return cursorFile{file: f}, true
		}
	}
	return cursorFile{}, false
}

func createCoreCursor(c *xgb.Conn, cursor uint16) (xproto.Cursor, error) {
	result, err := xproto.NewCursorId(c)
	if err != nil {
		return 0, err
	}
	font, err := xproto.NewFontId(c)
	if err != nil {
		return 0, err
	}
	xproto.OpenFont(c, font, uint16(len("cursor")), "cursor")
	xproto.CreateGlyphCursor(c, result, font, font, cursor, cursor+1, 0, 0, 0, 0xffff, 0xffff, 0xffff)
	xproto.CloseFont(c, font)
	return result, nil
}

type renderStorage struct {
	pixmap xproto.Pixmap
	gc     xproto.Gcontext
	w, h   uint16
	set    bool
}

func toU16(input uint32) uint16 {
	if input > 0xffff {
		panic("xcursor-rs has a 16 bit limit on cursor size, but some number does not fit into u16?!")
	}
	return uint16(input)
}

func createRenderCursor(c *xgb.Conn, h *Handle, image *Image, storage *renderStorage) (render.Animcursorelt, error) {
	width, height := toU16(image.Width), toU16(image.Height)
	if !storage.set || storage.w != width || storage.h != height {
		var pixmap xproto.Pixmap
		var gc xproto.Gcontext
		if storage.set {
			xproto.FreeGC(c, storage.gc)
			xproto.FreePixmap(c, storage.pixmap)
			pixmap, gc = storage.pixmap, storage.gc
		} else {
			var err error
			if pixmap, err = xproto.NewPixmapId(c); err != nil {
				return render.Animcursorelt{}, err
			}
			if gc, err = xproto.NewGcontextId(c); err != nil {
				return render.Animcursorelt{}, err
			}
		}
		xproto.CreatePixmap(c, 32, pixmap, xproto.Drawable(h.root), width, height)
		xproto.CreateGC(c, gc, xproto.Drawable(pixmap), 0, nil)
		*storage = renderStorage{pixmap: pixmap, gc: gc, w: width, h: height, set: true}
	}
	xproto.PutImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(storage.pixmap), storage.gc, width, height, 0, 0, 0, 32, image.PixelsRGBA)
	picture, err := render.NewPictureId(c)
	if err != nil {
		return render.Animcursorelt{}, err
	}
	render.CreatePicture(c, picture, xproto.Drawable(storage.pixmap), h.pictureFormat, 0, nil)
	cursor, err := xproto.NewCursorId(c)
	if err != nil {
		render.FreePicture(c, picture)
		return render.Animcursorelt{}, err
	}
	render.CreateCursor(c, cursor, picture, toU16(image.Xhot), toU16(image.Yhot))
	render.FreePicture(c, picture)
	return render.Animcursorelt{Cursor: cursor, Delay: image.Delay}, nil
}

func loadCursor(c *xgb.Conn, h *Handle, name string) (xproto.Cursor, error) {
	file, ok := openCursor(h.theme, h.hasTheme, name)
	if !ok {
		return 0, nil
	}
	if file.isCore {
		return createCoreCursor(c, file.coreChar)
	}
	defer file.file.Close()
	if h.renderSupport == renderSupportNone {
		return 0, nil
	}
	images, err := parseCursor(readSeeker(file.file), h.cursorSize)
	if err != nil {
		return 0, ErrInvalidValue
	}
	if h.renderSupport == renderSupportStaticCursor {
		images = images[0:1]
	}
	var storage renderStorage
	cursors := make([]render.Animcursorelt, 0, len(images))
	for i := range images {
		elt, err := createRenderCursor(c, h, &images[i], &storage)
		if err != nil {
			return 0, err
		}
		cursors = append(cursors, elt)
	}
	if storage.set {
		xproto.FreeGC(c, storage.gc)
		xproto.FreePixmap(c, storage.pixmap)
	}
	if len(cursors) == 1 {
		return cursors[0].Cursor, nil
	}
	result, err := xproto.NewCursorId(c)
	if err != nil {
		return 0, err
	}
	render.CreateAnimCursor(c, result, cursors)
	for _, elem := range cursors {
		xproto.FreeCursor(c, elem.Cursor)
	}
	return result, nil
}

type bufferedFile struct {
	f *os.File
	r *bufio.Reader
}

func readSeeker(f *os.File) io.ReadSeeker { return &bufferedFile{f: f, r: bufio.NewReader(f)} }

func (b *bufferedFile) Read(p []byte) (int, error) { return b.r.Read(p) }

func (b *bufferedFile) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		offset -= int64(b.r.Buffered())
	}
	n, err := b.f.Seek(offset, whence)
	b.r.Reset(b.f)
	return n, err
}

func findFormat(reply *render.QueryPictFormatsReply) render.Pictformat {
	for _, format := range reply.Formats {
		d := format.Direct
		if format.Type == render.PictTypeDirect && format.Depth == 32 &&
			d.RedShift == 16 && d.RedMask == 0xff &&
			d.GreenShift == 8 && d.GreenMask == 0xff &&
			d.BlueShift == 0 && d.BlueMask == 0xff &&
			d.AlphaShift == 24 && d.AlphaMask == 0xff {
			return format.Id
		}
	}
	panic("The X11 server is missing the RENDER ARGB_32 standard format!")
}

func getCursorSize(rmCursorSize, rmXftDpi uint32, screen *xproto.ScreenInfo) uint32 {
	if s, ok := os.LookupEnv("XCURSOR_SIZE"); ok {
		if size, err := strconv.ParseUint(s, 10, 32); err == nil {
			return uint32(size)
		}
	}
	if rmCursorSize > 0 {
		return rmCursorSize
	}
	if rmXftDpi > 0 {
		return rmXftDpi * 16 / 72
	}
	return uint32(min(screen.HeightInPixels, screen.WidthInPixels) / 48)
}

var coreCursors = [...]struct {
	name string
	id   uint16
}{
	{"X_cursor", 0}, {"arrow", 1}, {"based_arrow_down", 2}, {"based_arrow_up", 3}, {"boat", 4},
	{"bogosity", 5}, {"bottom_left_corner", 6}, {"bottom_right_corner", 7}, {"bottom_side", 8},
	{"bottom_tee", 9}, {"box_spiral", 10}, {"center_ptr", 11}, {"circle", 12}, {"clock", 13},
	{"coffee_mug", 14}, {"cross", 15}, {"cross_reverse", 16}, {"crosshair", 17},
	{"diamond_cross", 18}, {"dot", 19}, {"dotbox", 20}, {"double_arrow", 21}, {"draft_large", 22},
	{"draft_small", 23}, {"draped_box", 24}, {"exchange", 25}, {"fleur", 26}, {"gobbler", 27},
	{"gumby", 28}, {"hand1", 29}, {"hand2", 30}, {"heart", 31}, {"icon", 32}, {"iron_cross", 33},
	{"left_ptr", 34}, {"left_side", 35}, {"left_tee", 36}, {"leftbutton", 37}, {"ll_angle", 38},
	{"lr_angle", 39}, {"man", 40}, {"middlebutton", 41}, {"mouse", 42}, {"pencil", 43},
	{"pirate", 44}, {"plus", 45}, {"question_arrow", 46}, {"right_ptr", 47}, {"right_side", 48},
	{"right_tee", 49}, {"rightbutton", 50}, {"rtl_logo", 51}, {"sailboat", 52},
	{"sb_down_arrow", 53}, {"sb_h_double_arrow", 54}, {"sb_left_arrow", 55},
	{"sb_right_arrow", 56}, {"sb_up_arrow", 57}, {"sb_v_double_arrow", 58}, {"shuttle", 59},
	{"sizing", 60}, {"spider", 61}, {"spraycan", 62}, {"star", 63}, {"target", 64},
	{"tcross", 65}, {"top_left_arrow", 66}, {"top_left_corner", 67}, {"top_right_corner", 68},
	{"top_side", 69}, {"top_tee", 70}, {"trek", 71}, {"ul_angle", 72}, {"umbrella", 73},
	{"ur_angle", 74}, {"watch", 75}, {"xterm", 76},
}

func cursorShapeToID(name string) (uint16, bool) {
	for _, c := range coreCursors {
		if c.name == name {
			return c.id, true
		}
	}
	return 0, false
}
