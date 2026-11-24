//go:build linux

package screencap

import (
	"errors"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xfixes"
	"github.com/jezek/xgb/xproto"
	"golang.org/x/sys/unix"

	gpui "github.com/megascan/gpui-go"
)

const (
	xfixesMajorVersion = 5
	xfixesMinorVersion = 0
)

// X11Capturer captures a display or window of the X server.
type X11Capturer struct {
	capturerDone  chan struct{}
	capturerState *atomic.Uint32
	panicked      *atomic.Bool
	target        gpui.ScapTarget
}

type x11Conn struct {
	conn  *xgb.Conn
	root  xproto.Window
	shm   *x11Shm
	noShm bool
}

type x11Shm struct {
	seg  shm.Seg
	id   int
	data []byte
}

func (c *x11Conn) closeShm() {
	if c.shm == nil {
		return
	}
	shm.Detach(c.conn, c.shm.seg)
	_ = unix.SysvShmDetach(c.shm.data)
	c.shm = nil
}

func (c *x11Conn) shmSegment(size int) (*x11Shm, bool) {
	if c.noShm {
		return nil, false
	}
	if c.shm != nil && len(c.shm.data) >= size {
		return c.shm, true
	}
	c.closeShm()
	id, err := unix.SysvShmGet(unix.IPC_PRIVATE, size, unix.IPC_CREAT|0o600)
	if err != nil {
		return nil, false
	}
	data, err := unix.SysvShmAttach(id, 0, 0)
	if err != nil {
		_, _ = unix.SysvShmCtl(id, unix.IPC_RMID, nil)
		return nil, false
	}
	seg, err := shm.NewSegId(c.conn)
	if err == nil {
		err = shm.AttachChecked(c.conn, seg, uint32(id), false).Check()
	}
	_, _ = unix.SysvShmCtl(id, unix.IPC_RMID, nil)
	if err != nil {
		_ = unix.SysvShmDetach(data)
		c.noShm = true
		return nil, false
	}
	c.shm = &x11Shm{seg: seg, id: id, data: data}
	return c.shm, true
}

func imageFitsBGRx(img []byte, width, height int16) bool {
	return width > 0 && height > 0 && len(img) >= int(width)*int(height)*4
}

func drawCursor(c *x11Conn, img []byte, winX, winY, winWidth, winHeight int16, isWin bool, win xproto.Window) error {
	if !imageFitsBGRx(img, winWidth, winHeight) {
		return nil
	}
	cursorImage, err := xfixes.GetCursorImage(c.conn).Reply()
	if err != nil {
		return err
	}

	wx := int32(winX)
	wy := int32(winY)
	ww := int32(winWidth)
	wh := int32(winHeight)

	cursorX := int32(cursorImage.X) - int32(cursorImage.Xhot)
	cursorY := int32(cursorImage.Y) - int32(cursorImage.Yhot)
	if isWin {
		tr, err := xproto.TranslateCoordinates(c.conn, c.root, win, cursorImage.X, cursorImage.Y).Reply()
		if err != nil || !tr.SameScreen {
			return nil
		}
		cursorX = int32(tr.DstX) - int32(cursorImage.Xhot)
		cursorY = int32(tr.DstY) - int32(cursorImage.Yhot)
	}

	if cursorX >= ww+wx || cursorY >= wh+wy || cursorX < wx || cursorY < wy {
		return nil
	}

	x := max(cursorX, wx)
	y := max(cursorY, wy)

	w := uint32(min(cursorX+int32(cursorImage.Width), wx+ww) - x)
	h := uint32(min(cursorY+int32(cursorImage.Height), wy+wh) - y)

	cOff := uint32(x - cursorX)
	iOff := x - wx

	const stride uint32 = 4
	cursorIdx := uint32((y - cursorY) * int32(cursorImage.Width))
	imageIdx := uint32((y - wy) * ww * int32(stride))

	for range h {
		cursorIdx += cOff
		imageIdx += uint32(iOff) * stride
		for range w {
			cursorPix := cursorImage.CursorImage[cursorIdx]
			r := uint8(cursorPix & 0xFF)
			g := uint8((cursorPix >> 8) & 0xFF)
			b := uint8((cursorPix >> 16) & 0xFF)
			a := (cursorPix >> 24) & 0xFF

			i := int(imageIdx)
			if a == 0xFF {
				img[i] = r
				img[i+1] = g
				img[i+2] = b
			} else if a > 0 {
				a = 255 - a
				img[i] = r + uint8((uint32(img[i])*a+255/2)/255)
				img[i+1] = g + uint8((uint32(img[i+1])*a+255/2)/255)
				img[i+2] = b + uint8((uint32(img[i+2])*a+255/2)/255)
			}

			cursorIdx++
			imageIdx += stride
		}
		cursorIdx += uint32(cursorImage.Width) - w - cOff
		imageIdx += uint32(ww-int32(w)-iOff) * stride
	}
	return nil
}

func getImage(c *x11Conn, window xproto.Window, x, y int16, width, height uint16) ([]byte, error) {
	size := int(width) * int(height) * 4
	if size > 0 {
		if seg, ok := c.shmSegment(size); ok {
			reply, err := shm.GetImage(c.conn, xproto.Drawable(window), x, y, width, height, math.MaxUint32, xproto.ImageFormatZPixmap, seg.seg, 0).Reply()
			if err == nil && int(reply.Size) <= len(seg.data) {
				return append([]byte(nil), seg.data[:reply.Size]...), nil
			}
			c.closeShm()
			c.noShm = true
		}
	}
	img, err := xproto.GetImage(c.conn, xproto.ImageFormatZPixmap, xproto.Drawable(window), x, y, width, height, math.MaxUint32).Reply()
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), img.Data...), nil
}

func grab(c *x11Conn, target gpui.ScapTarget, showCursor bool) (gpui.ScapFrame, error) {
	var (
		x, y          int16
		width, height uint16
		window        xproto.Window
		isWin         bool
	)
	switch t := target.(type) {
	case *gpui.ScapWindow:
		geom, err := xproto.GetGeometry(c.conn, xproto.Drawable(t.RawHandle)).Reply()
		if err != nil {
			return nil, err
		}
		x, y, width, height, window, isWin = 0, 0, geom.Width, geom.Height, xproto.Window(t.RawHandle), true
	case *gpui.ScapDisplay:
		x, y, width, height, window, isWin = t.XOffset, t.YOffset, t.Width, t.Height, xproto.Window(t.RawHandle), false
	default:
		return nil, errors.New("screencap: unknown target")
	}

	imgData, err := getImage(c, window, x, y, width, height)
	if err != nil {
		return nil, err
	}

	if showCursor {
		if err := drawCursor(c, imgData, x, y, int16(width), int16(height), isWin, window); err != nil {
			return nil, err
		}
	}

	now := time.Now().UnixNano()
	if now < 0 {
		return nil, errors.New("Unix epoch is in the past")
	}
	return gpui.ScapFrameBGRx{ScapPixels: gpui.ScapPixels{
		DisplayTime: uint64(now),
		Width:       int32(width),
		Height:      int32(height),
		Data:        imgData,
	}}, nil
}

func queryXfixesVersion(conn *xgb.Conn) error {
	_, err := xfixes.QueryVersion(conn, xfixesMajorVersion, xfixesMinorVersion).Reply()
	return err
}

// NewX11Capturer connects to the X server and starts the (paused) capture goroutine.
func NewX11Capturer(options gpui.ScapCapturerOptions, tx *frameQueue) (*X11Capturer, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, linCapErrorFrom(err)
	}
	fail := func(err error) (*X11Capturer, error) {
		conn.Close()
		return nil, err
	}
	if err := randr.Init(conn); err != nil {
		return fail(linCapErrorFrom(err))
	}
	if err := xfixes.Init(conn); err != nil {
		return fail(linCapErrorFrom(err))
	}
	if err := queryXfixesVersion(conn); err != nil {
		return fail(linCapErrorFrom(err))
	}
	roots := xproto.Setup(conn).Roots
	if conn.DefaultScreen >= len(roots) {
		return fail(NewLinCapError("Failed to get setup root"))
	}
	screen := &roots[conn.DefaultScreen]

	target := options.Target
	if target == nil {
		d, err := getDefaultXDisplay(conn, screen)
		if err != nil {
			return fail(linCapErrorFrom(err))
		}
		target = d
	}

	c := &x11Conn{conn: conn, root: screen.Root, noShm: shm.Init(conn) != nil}

	framerate := float32(options.FPS)
	showCursor := options.ShowCursor
	capturerState := &atomic.Uint32{}
	done := make(chan struct{})
	panicked := &atomic.Bool{}

	go func() {
		defer close(done)
		defer tx.close()
		defer conn.Close()
		defer c.closeShm()
		for capturerState.Load() == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		secs := 1.0 / framerate
		if math.IsInf(float64(secs), 0) || math.IsNaN(float64(secs)) || float64(secs)*float64(time.Second) > math.MaxInt64 {
			panicked.Store(true)
			return
		}
		frameTime := time.Duration(float64(secs) * float64(time.Second))
		for capturerState.Load() == 1 {
			start := time.Now()
			frame, err := grab(c, target, showCursor)
			tx.send(frame, err)
			if err != nil {
				return
			}
			if elapsed := time.Since(start); elapsed < frameTime {
				time.Sleep(frameTime - elapsed)
			}
		}
	}()

	return &X11Capturer{capturerState: capturerState, capturerDone: done, panicked: panicked, target: target}, nil
}

// StartCapture starts grabbing frames.
func (c *X11Capturer) StartCapture() { c.capturerState.Store(1) }

// StopCapture stops grabbing and waits for the capture goroutine.
func (c *X11Capturer) StopCapture() {
	c.capturerState.Store(2)
	if c.capturerDone != nil {
		<-c.capturerDone
		c.capturerDone = nil
		if c.panicked.Load() {
			slog.Error("Failed to join X11 screen capture thread: Any { .. }")
		}
	}
}

// Target returns the captured target.
func (c *X11Capturer) Target() (gpui.ScapTarget, bool) { return c.target, true }

func (c *X11Capturer) close() {
	if c.capturerDone == nil {
		return
	}
	c.capturerState.Store(2)
	<-c.capturerDone
	c.capturerDone = nil
}
