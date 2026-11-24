//go:build linux

package wayland

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/wl"
	"github.com/megascan/gpui-go/internal/linux/wlproto"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
)

var textMimeTypes = [3]string{"text/plain;charset=utf-8", "UTF8_STRING", "text/plain"}

const fileListMimeType = "text/uri-list"

var allowedTextMimeTypes = [2]string{"text/plain;charset=utf-8", "UTF8_STRING"}

// Clipboard holds the clipboard and primary selection contents of a connection.
type Clipboard struct {
	connection *wl.Display
	loopHandle *common.LoopHandle
	selfMime   string

	contents        gpui.Option[gpui.ClipboardItem]
	primaryContents gpui.Option[gpui.ClipboardItem]

	cachedRead          gpui.Option[gpui.ClipboardItem]
	currentOffer        gpui.Option[DataOffer[*wlproto.WlDataOffer]]
	cachedPrimaryRead   gpui.Option[gpui.ClipboardItem]
	currentPrimaryOffer gpui.Option[DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]]
}

// ReceiveData is an offer whose data can be received through a file descriptor.
type ReceiveData interface {
	Receive(mimeType string, fd int)
}

// DataOffer is a wl_data_offer or zwp_primary_selection_offer_v1 with the mime types it
// advertised.
type DataOffer[T ReceiveData] struct {
	Inner     T
	mimeTypes []string
}

// NewDataOffer wraps offer with no mime types yet.
func NewDataOffer[T ReceiveData](offer T) DataOffer[T] {
	return DataOffer[T]{Inner: offer}
}

// AddMimeType records a mime type the offer advertised.
func (o *DataOffer[T]) AddMimeType(mimeType string) {
	o.mimeTypes = append(slices.Clip(o.mimeTypes), mimeType)
}

func (o *DataOffer[T]) hasMimeType(mimeType string) bool {
	return slices.Contains(o.mimeTypes, mimeType)
}

func (o *DataOffer[T]) readBytes(connection *wl.Display, mimeType string) ([]byte, bool) {
	var fds [2]int
	if err := unix.Pipe2(fds[:], unix.O_CLOEXEC); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	o.Inner.Receive(mimeType, fds[1])
	fd := fds[0]
	unix.Close(fds[1])

	if err := connection.Flush(); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}

	bytes, err := common.ReadFdWithTimeout(fd, common.PipeReadTimeout)
	if err != nil {
		slog.Error(fmt.Sprintf("error reading clipboard pipe: %v", err))
		return nil, false
	}
	return bytes, true
}

func (o *DataOffer[T]) readText(connection *wl.Display) (gpui.ClipboardItem, bool) {
	index := slices.IndexFunc(o.mimeTypes, func(mimeType string) bool {
		return slices.Contains(allowedTextMimeTypes[:], mimeType)
	})
	if index < 0 {
		return gpui.ClipboardItem{}, false
	}
	bytes, ok := o.readBytes(connection, o.mimeTypes[index])
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	if !utf8.Valid(bytes) {
		slog.Error(fmt.Sprintf("Failed to convert clipboard content to UTF-8: %v", errors.New("invalid utf-8 sequence")))
		return gpui.ClipboardItem{}, false
	}
	result := strings.ReplaceAll(string(bytes), "\r\n", "\n")
	return gpui.NewClipboardItemString(result), true
}

func (o *DataOffer[T]) readImage(connection *wl.Display) (gpui.ClipboardItem, bool) {
	for format := range gpui.ImageFormatIter() {
		mimeType := format.MimeType()
		if !o.hasMimeType(mimeType) {
			continue
		}
		if bytes, ok := o.readBytes(connection, mimeType); ok {
			id := gpui.Hash(bytes)
			return gpui.ClipboardItem{Entries: []gpui.ClipboardEntry{gpui.ClipboardEntryImage{Image: gpui.Image{Format: format, Bytes: bytes, ID: id}}}}, true
		}
	}
	return gpui.ClipboardItem{}, false
}

func (o *DataOffer[T]) readTextOrImage(connection *wl.Display) (gpui.ClipboardItem, bool) {
	if item, ok := o.readText(connection); ok {
		return item, true
	}
	return o.readImage(connection)
}

// NewClipboard creates the clipboard state of a connection.
func NewClipboard(connection *wl.Display, loopHandle *common.LoopHandle) *Clipboard {
	return &Clipboard{
		connection: connection,
		loopHandle: loopHandle,
		selfMime:   fmt.Sprintf("pid/%d", os.Getpid()),
	}
}

// Set stores item as the clipboard contents.
func (c *Clipboard) Set(item gpui.ClipboardItem) {
	c.contents = gpui.Some(item)
}

// SetPrimary stores item as the primary selection contents.
func (c *Clipboard) SetPrimary(item gpui.ClipboardItem) {
	c.primaryContents = gpui.Some(item)
}

// SetOffer sets the current clipboard offer.
func (c *Clipboard) SetOffer(dataOffer gpui.Option[DataOffer[*wlproto.WlDataOffer]]) {
	c.cachedRead = gpui.None[gpui.ClipboardItem]()
	c.currentOffer = dataOffer
}

// SetPrimaryOffer sets the current primary selection offer.
func (c *Clipboard) SetPrimaryOffer(dataOffer gpui.Option[DataOffer[*wlproto.ZwpPrimarySelectionOfferV1]]) {
	c.cachedPrimaryRead = gpui.None[gpui.ClipboardItem]()
	c.currentPrimaryOffer = dataOffer
}

// SelfMime returns the mime type that marks this process's own offers.
func (c *Clipboard) SelfMime() string {
	return c.selfMime
}

// Send writes the clipboard text to fd, which it takes ownership of.
func (c *Clipboard) Send(mimeType string, fd int) {
	if contents, ok := c.contents.Get(); ok {
		if text, ok := contents.Text(); ok {
			c.SendBytes(fd, []byte(text))
			return
		}
	}
	unix.Close(fd)
}

// SendPrimary writes the primary selection text to fd, which it takes ownership of.
func (c *Clipboard) SendPrimary(mimeType string, fd int) {
	if contents, ok := c.primaryContents.Get(); ok {
		if text, ok := contents.Text(); ok {
			c.SendBytes(fd, []byte(text))
			return
		}
	}
	unix.Close(fd)
}

// Read reads the clipboard.
func (c *Clipboard) Read() (gpui.ClipboardItem, bool) {
	offer, ok := c.currentOffer.Get()
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	if cached, ok := c.cachedRead.Get(); ok {
		return cached, true
	}
	if offer.hasMimeType(c.selfMime) {
		return c.contents.Get()
	}
	item, ok := offer.readTextOrImage(c.connection)
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	c.cachedRead = gpui.Some(item)
	return item, true
}

// ReadPrimary reads the primary selection.
func (c *Clipboard) ReadPrimary() (gpui.ClipboardItem, bool) {
	offer, ok := c.currentPrimaryOffer.Get()
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	if cached, ok := c.cachedPrimaryRead.Get(); ok {
		return cached, true
	}
	if offer.hasMimeType(c.selfMime) {
		return c.primaryContents.Get()
	}
	item, ok := offer.readTextOrImage(c.connection)
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	c.cachedPrimaryRead = gpui.Some(item)
	return item, true
}

// SendBytes writes bytes to fd from the event loop as it becomes writable, then closes it.
func (c *Clipboard) SendBytes(fd int, bytes []byte) {
	written := 0
	source := common.NewGeneric(fd, common.Interest{Writable: true}, func(common.Readiness) (common.PostAction, error) {
		for {
			n, err := unix.Write(fd, bytes[written:])
			switch {
			case err == nil && written+n == len(bytes):
				written += n
				unix.Close(fd)
				return common.PostActionRemove, nil
			case err == nil:
				written += n
			case errors.Is(err, unix.EAGAIN):
				return common.PostActionContinue, nil
			default:
				unix.Close(fd)
				return common.PostActionRemove, nil
			}
		}
	})
	if _, err := c.loopHandle.InsertEventSource(source); err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
}
