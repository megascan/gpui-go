//go:build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log/slog"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const dragdropGetFilesCount uint32 = 0xFFFFFFFF

var (
	clipboardHashFormat     = sync.OnceValue(func() uint32 { return registerClipboardFormat("GPUI internal text hash") })
	clipboardMetadataFormat = sync.OnceValue(func() uint32 { return registerClipboardFormat("GPUI internal metadata") })
	clipboardSvgFormat      = sync.OnceValue(func() uint32 { return registerClipboardFormat("image/svg+xml") })
	clipboardGifFormat      = sync.OnceValue(func() uint32 { return registerClipboardFormat("GIF") })
	clipboardPngFormat      = sync.OnceValue(func() uint32 { return registerClipboardFormat("PNG") })
	clipboardJpgFormat      = sync.OnceValue(func() uint32 { return registerClipboardFormat("JFIF") })
)

var imageFormatsMap = sync.OnceValue(func() map[uint32]gpui.ImageFormat {
	return map[uint32]gpui.ImageFormat{
		clipboardPngFormat(): gpui.ImageFormatPng,
		clipboardGifFormat(): gpui.ImageFormatGif,
		clipboardJpgFormat(): gpui.ImageFormatJpeg,
		clipboardSvgFormat(): gpui.ImageFormatSvg,
	}
})

func registerClipboardFormat(format string) uint32 {
	name := xwin.StringToUTF16Ptr(format)
	ret, err := win32.RegisterClipboardFormatW(name)
	if ret == 0 {
		panic(fmt.Sprintf("Error when registering clipboard format: %v", err))
	}
	slog.Debug(fmt.Sprintf("Registered clipboard format %s as %d", format, ret))
	return ret
}

func getClipboardData(format uint32) (*lockedGlobal, bool) {
	handle, err := win32.GetClipboardData(format)
	if err != nil {
		return nil, false
	}
	return lockGlobal(win32.HGLOBAL(handle))
}

func writeToClipboard(item gpui.ClipboardItem) {
	clip, ok := openClipboardGuard()
	if !ok {
		return
	}
	defer clip.Close()

	err := func() error {
		if err := win32.EmptyClipboard(); err != nil {
			return err
		}
		for _, entry := range item.Entries {
			switch entry := entry.(type) {
			case gpui.ClipboardEntryString:
				if err := writeString(entry.ClipboardString); err != nil {
					return err
				}
			case gpui.ClipboardEntryImage:
				if err := writeImage(entry.Image); err != nil {
					return err
				}
			case gpui.ClipboardEntryExternalPaths:
			}
		}
		return nil
	}()

	if err != nil {
		slog.Error(fmt.Sprintf("Failed to write to clipboard: %v", err))
	}
}

func readFromClipboard() (gpui.ClipboardItem, bool) {
	clip, ok := openClipboardGuard()
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	defer clip.Close()

	var entries []gpui.ClipboardEntry
	haveText := false
	haveImage := false
	haveFiles := false

	count := win32.CountClipboardFormats()
	format := uint32(0)
	for range count {
		format = win32.EnumClipboardFormats(format)

		if !haveText && format == win32.CF_UNICODETEXT {
			if entry, ok := readString(); ok {
				entries = append(entries, entry)
				haveText = true
			}
		} else if !haveImage && isImageFormat(format) {
			if entry, ok := readImage(format); ok {
				entries = append(entries, entry)
				haveImage = true
			}
		} else if !haveFiles && format == win32.CF_HDROP {
			if entry, ok := readFiles(); ok {
				entries = append(entries, entry)
				haveFiles = true
			}
		}
	}

	if len(entries) == 0 {
		logUnsupportedClipboardFormats()
		return gpui.ClipboardItem{}, false
	}
	return gpui.ClipboardItem{Entries: entries}, true
}

func withFileNames(hdrop win32.HDROP, f func(string)) {
	fileCount := win32.DragQueryFileW(hdrop, dragdropGetFilesCount, nil)
	for fileIndex := range fileCount {
		filenameLength := int(win32.DragQueryFileW(hdrop, fileIndex, nil))
		buffer := make([]uint16, filenameLength+1)
		ret := win32.DragQueryFileW(hdrop, fileIndex, buffer)
		if ret == 0 {
			slog.Error("unable to read file name of dragged file")
			continue
		}
		fileName, err := decodeUTF16Strict(buffer[:filenameLength])
		if err != nil {
			slog.Error(fmt.Sprintf("dragged file name is not UTF-16: %v", err))
			continue
		}
		f(fileName)
	}
}

func decodeUTF16Strict(s []uint16) (string, error) {
	for i := 0; i < len(s); i++ {
		switch {
		case utf16.IsSurrogate(rune(s[i])) && s[i] < 0xDC00 && i+1 < len(s) && s[i+1] >= 0xDC00 && s[i+1] <= 0xDFFF:
			i++
		case utf16.IsSurrogate(rune(s[i])):
			return "", fmt.Errorf("invalid utf-16: lone surrogate found")
		}
	}
	return string(utf16.Decode(s)), nil
}

func setClipboardBytes(data []byte, format uint32) error {
	global, err := win32.GlobalAlloc(win32.GMEM_MOVEABLE, uintptr(len(data)))
	if err != nil {
		return err
	}
	ptr, _ := win32.GlobalLock(global)
	if ptr == nil {
		win32.GlobalFree(global)
		return errors.New("GlobalLock returned null")
	}
	copy(unsafe.Slice((*byte)(ptr), len(data)), data)
	win32.GlobalUnlock(global)
	if _, err := win32.SetClipboardData(format, win32.HANDLE(global)); err != nil {
		win32.GlobalFree(global)
		return err
	}
	return nil
}

func wideBytes(wide []uint16) []byte {
	if len(wide) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&wide[0])), len(wide)*2)
}

func getClipboardString(format uint32) (string, bool) {
	locked, ok := getClipboardData(format)
	if !ok {
		return "", false
	}
	defer locked.Close()
	b := locked.bytes()
	wordsLen := len(b) / 2
	if wordsLen == 0 {
		return "", true
	}
	slice := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), wordsLen)
	actualLen := wordsLen
	for i, c := range slice {
		if c == 0 {
			actualLen = i
			break
		}
	}
	return string(utf16.Decode(slice[:actualLen])), true
}

func isImageFormat(format uint32) bool {
	_, ok := imageFormatsMap()[format]
	return ok || format == win32.CF_DIB
}

func writeString(item gpui.ClipboardString) error {
	text := strings.ReplaceAll(strings.ReplaceAll(item.Text, "\x00", " "), "\r\n", "\n")
	wide := append(utf16.Encode([]rune(strings.ReplaceAll(text, "\n", "\r\n"))), 0)
	if err := setClipboardBytes(wideBytes(wide), win32.CF_UNICODETEXT); err != nil {
		return err
	}

	if metadata, ok := item.Metadata.Get(); ok {
		hashBytes := binary.NativeEndian.AppendUint64(nil, gpui.ClipboardStringTextHash(text))
		if err := setClipboardBytes(hashBytes, clipboardHashFormat()); err != nil {
			return err
		}

		wide := append(utf16.Encode([]rune(metadata)), 0)
		if err := setClipboardBytes(wideBytes(wide), clipboardMetadataFormat()); err != nil {
			return err
		}
	}
	return nil
}

func writeImage(item gpui.Image) error {
	var nativeFormat uint32
	hasNative := true
	switch item.Format {
	case gpui.ImageFormatSvg:
		nativeFormat = clipboardSvgFormat()
	case gpui.ImageFormatGif:
		nativeFormat = clipboardGifFormat()
	case gpui.ImageFormatPng:
		nativeFormat = clipboardPngFormat()
	case gpui.ImageFormatJpeg:
		nativeFormat = clipboardJpgFormat()
	default:
		hasNative = false
	}
	if hasNative {
		if err := setClipboardBytes(item.Bytes, nativeFormat); err != nil {
			return err
		}
	}

	if item.Format != gpui.ImageFormatSvg && !(hasNative && nativeFormat == clipboardPngFormat()) {
		if pngBytes, ok := convertToPng(item.Bytes, item.Format); ok {
			if err := setClipboardBytes(pngBytes, clipboardPngFormat()); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertToPng(data []byte, format gpui.ImageFormat) ([]byte, bool) {
	decode, ok := gpuiToImageFormat(format)
	if !ok {
		return nil, false
	}
	img, err := decode(bytes.NewReader(data))
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to decode image for PNG conversion: %v", err))
		return nil, false
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		slog.Warn(fmt.Sprintf("Failed to encode PNG: %v", err))
		return nil, false
	}
	return buf.Bytes(), true
}

func readString() (gpui.ClipboardEntry, bool) {
	text, ok := getClipboardString(win32.CF_UNICODETEXT)
	if !ok {
		return nil, false
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	metadata := readClipboardMetadata(text)
	return gpui.ClipboardEntryString{ClipboardString: gpui.ClipboardString{Text: text, Metadata: metadata}}, true
}

func readClipboardMetadata(text string) gpui.Option[string] {
	locked, ok := getClipboardData(clipboardHashFormat())
	if !ok {
		return gpui.None[string]()
	}
	b := locked.bytes()
	if len(b) < 8 {
		locked.Close()
		return gpui.None[string]()
	}
	hash := binary.NativeEndian.Uint64(b[:8])
	locked.Close()
	if hash != gpui.ClipboardStringTextHash(text) {
		return gpui.None[string]()
	}
	metadata, ok := getClipboardString(clipboardMetadataFormat())
	if !ok {
		return gpui.None[string]()
	}
	return gpui.Some(metadata)
}

func readImage(format uint32) (gpui.ClipboardEntry, bool) {
	locked, ok := getClipboardData(format)
	if !ok {
		return nil, false
	}
	defer locked.Close()
	var data []byte
	var imageFormat gpui.ImageFormat
	if format == win32.CF_DIB {
		bmpBytes, ok := convertDibToBmp(locked.bytes())
		if !ok {
			return nil, false
		}
		data, imageFormat = bmpBytes, gpui.ImageFormatBmp
	} else {
		f, ok := imageFormatsMap()[format]
		if !ok {
			return nil, false
		}
		data, imageFormat = bytes.Clone(locked.bytes()), f
	}
	id := gpui.Hash(data)
	return gpui.ClipboardEntryImage{Image: gpui.Image{Format: imageFormat, Bytes: data, ID: id}}, true
}

func readFiles() (gpui.ClipboardEntry, bool) {
	locked, ok := getClipboardData(win32.CF_HDROP)
	if !ok {
		return nil, false
	}
	defer locked.Close()
	hdrop := win32.HDROP(uintptr(locked.ptr))
	var filenames []string
	withFileNames(hdrop, func(name string) { filenames = append(filenames, name) })
	return gpui.ClipboardEntryExternalPaths{ExternalPaths: gpui.ExternalPaths(filenames)}, true
}

func convertDibToBmp(dib []byte) ([]byte, bool) {
	if len(dib) < 40 {
		return nil, false
	}

	headerSize := binary.LittleEndian.Uint32(dib[0:4])
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	compression := binary.LittleEndian.Uint32(dib[16:20])

	var colorTableSize uint32
	switch {
	case bitCount <= 8:
		colorsUsed := binary.LittleEndian.Uint32(dib[32:36])
		if colorsUsed == 0 {
			colorTableSize = (uint32(1) << bitCount) * 4
		} else {
			colorTableSize = colorsUsed * 4
		}
	case compression == 3:
		colorTableSize = 12
	}

	pixelOffset := 14 + headerSize + colorTableSize
	fileSize := 14 + uint32(len(dib))

	bmpBytes := make([]byte, 0, fileSize)
	bmpBytes = append(bmpBytes, 'B', 'M')
	bmpBytes = binary.LittleEndian.AppendUint32(bmpBytes, fileSize)
	bmpBytes = append(bmpBytes, 0, 0, 0, 0)
	bmpBytes = binary.LittleEndian.AppendUint32(bmpBytes, pixelOffset)
	bmpBytes = append(bmpBytes, dib...)
	return bmpBytes, true
}

func logUnsupportedClipboardFormats() {
	count := win32.CountClipboardFormats()
	format := uint32(0)
	for range count {
		format = win32.EnumClipboardFormats(format)
		var buffer [64]uint16
		win32.GetClipboardFormatNameW(format, buffer[:])
		formatName := string(utf16.Decode(buffer[:]))
		slog.Warn(fmt.Sprintf("Try to paste with unsupported clipboard format: %d, %s.", format, formatName))
	}
}

type imageDecoder func(r *bytes.Reader) (image.Image, error)

func gpuiToImageFormat(value gpui.ImageFormat) (imageDecoder, bool) {
	switch value {
	case gpui.ImageFormatPng:
		return func(r *bytes.Reader) (image.Image, error) { return png.Decode(r) }, true
	case gpui.ImageFormatJpeg:
		return func(r *bytes.Reader) (image.Image, error) { return jpeg.Decode(r) }, true
	case gpui.ImageFormatWebp:
		return func(r *bytes.Reader) (image.Image, error) { return webp.Decode(r) }, true
	case gpui.ImageFormatGif:
		return func(r *bytes.Reader) (image.Image, error) { return gif.Decode(r) }, true
	case gpui.ImageFormatBmp:
		return func(r *bytes.Reader) (image.Image, error) { return bmp.Decode(r) }, true
	case gpui.ImageFormatTiff:
		return func(r *bytes.Reader) (image.Image, error) { return tiff.Decode(r) }, true
	default:
		slog.Warn(fmt.Sprintf("No image crate equivalent for format: %v", value))
		return nil, false
	}
}

type clipboardGuard struct{}

func openClipboardGuard() (clipboardGuard, bool) {
	if err := win32.OpenClipboard(0); err != nil {
		slog.Error(fmt.Sprintf("Failed to open clipboard: %v", err))
		return clipboardGuard{}, false
	}
	return clipboardGuard{}, true
}

func (clipboardGuard) Close() {
	if err := win32.CloseClipboard(); err != nil {
		slog.Error(fmt.Sprintf("Failed to close clipboard: %v", err))
	}
}

type lockedGlobal struct {
	global win32.HGLOBAL
	ptr    unsafe.Pointer
	size   uintptr
}

func lockGlobal(global win32.HGLOBAL) (*lockedGlobal, bool) {
	size := win32.GlobalSize(global)
	ptr, _ := win32.GlobalLock(global)
	if ptr == nil {
		return nil, false
	}
	return &lockedGlobal{global: global, ptr: ptr, size: size}, true
}

func (l *lockedGlobal) bytes() []byte {
	if l.size == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(l.ptr), l.size)
}

func (l *lockedGlobal) Close() {
	win32.GlobalUnlock(l.global)
}
