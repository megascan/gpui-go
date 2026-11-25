//go:build darwin && !ios

package macos

import (
	"encoding/binary"
	"strings"
	"unicode/utf8"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

// Pasteboard is an NSPasteboard read and written as gpui.ClipboardItem values. Release is its drop.
type Pasteboard struct {
	inner        *objc.Retained
	textHashType *objc.Retained
	metadataType *objc.Retained
}

// PasteboardGeneral returns the general pasteboard.
func PasteboardGeneral() (*Pasteboard, bool) {
	pasteboard := appkit.NSPasteboardGeneral()
	if pasteboard == 0 {
		return nil, false
	}
	return newPasteboard(pasteboard), true
}

// PasteboardFind returns the find pasteboard.
func PasteboardFind() (*Pasteboard, bool) {
	pasteboard := appkit.NSPasteboardWithName(appkit.NSPasteboardNameFind())
	if pasteboard == 0 {
		return nil, false
	}
	return newPasteboard(pasteboard), true
}

func newPasteboard(inner appkit.NSPasteboard) *Pasteboard {
	return &Pasteboard{
		inner:        objc.RetainObject(inner.ID()),
		textHashType: objc.Own(objc.NSStringFromGo("zed-text-hash")),
		metadataType: objc.Own(objc.NSStringFromGo("zed-metadata")),
	}
}

// Release drops the pasteboard's references. It is idempotent.
func (p *Pasteboard) Release() {
	p.inner.Release()
	p.textHashType.Release()
	p.metadataType.Release()
}

func (p *Pasteboard) ns() appkit.NSPasteboard {
	return appkit.NSPasteboard(p.inner.ID())
}

func isKindOf(object objc.ID, className string) bool {
	return object != objc.Nil && objc.Send[bool](object, objc.Sel("isKindOfClass:"), objc.ID(objc.GetClass(className)))
}

// Read returns the pasteboard's contents: file paths first, then a plain string, then the
// supported image types.
func (p *Pasteboard) Read() (gpui.ClipboardItem, bool) {
	var item gpui.ClipboardItem
	var ok bool
	objc.AutoreleasePool(func() {
		item, ok = p.read()
	})
	return item, ok
}

func (p *Pasteboard) read() (gpui.ClipboardItem, bool) {
	filenames := objc.Send[objc.ID](p.inner.ID(), objc.Sel("propertyListForType:"), appkit.Constant("NSFilenamesPboardType"))
	if isKindOf(filenames, "NSArray") {
		var paths gpui.ExternalPaths
		for _, file := range objc.NSArrayToIDs(filenames) {
			if !isKindOf(file, "NSString") {
				continue
			}
			if objc.Send[*byte](file, objc.Sel("UTF8String")) == nil {
				continue
			}
			paths = append(paths, utf8Lossy([]byte(nsStringToStr(file))))
		}
		if len(paths) > 0 {
			entries := []gpui.ClipboardEntry{gpui.ClipboardEntryExternalPaths{ExternalPaths: paths}}
			if stringItem, ok := p.readStringFromPasteboard(); ok {
				entries = append(entries, stringItem)
			}
			return gpui.ClipboardItem{Entries: entries}, true
		}
	}

	if stringEntry, ok := p.readStringFromPasteboard(); ok {
		return gpui.ClipboardItem{Entries: []gpui.ClipboardEntry{stringEntry}}, true
	}

	for format := range gpui.ImageFormatIter() {
		if item, ok := p.readImage(format); ok {
			return item, true
		}
	}
	return gpui.ClipboardItem{}, false
}

func (p *Pasteboard) readImage(format gpui.ImageFormat) (gpui.ClipboardItem, bool) {
	utType := UTTypeFrom(format)
	defer utType.Release()

	types := objc.Send[objc.ID](p.inner.ID(), objc.Sel("types"))
	if types == objc.Nil {
		return gpui.ClipboardItem{}, false
	}
	if !objc.Send[bool](types, objc.Sel("containsObject:"), utType.inner()) {
		return gpui.ClipboardItem{}, false
	}
	bytes, ok := p.dataForType(utType.inner())
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	return gpui.ClipboardItem{Entries: []gpui.ClipboardEntry{
		gpui.ClipboardEntryImage{Image: gpui.ImageFromBytes(format, bytes)},
	}}, true
}

func (p *Pasteboard) readStringFromPasteboard() (gpui.ClipboardEntry, bool) {
	pasteboardTypes := objc.Send[objc.ID](p.inner.ID(), objc.Sel("types"))
	if pasteboardTypes == objc.Nil {
		return nil, false
	}
	stringType := objc.NSStringFromGo("public.utf8-plain-text")
	defer objc.Release(stringType)

	if !objc.Send[bool](pasteboardTypes, objc.Sel("containsObject:"), stringType) {
		return nil, false
	}
	textBytes, ok := p.dataForType(stringType)
	if !ok {
		return nil, false
	}
	text := utf8Lossy(textBytes)
	metadata := gpui.None[string]()
	if hashBytes, ok := p.dataForType(p.textHashType.ID()); ok && len(hashBytes) == 8 {
		hash := binary.BigEndian.Uint64(hashBytes)
		if metadataBytes, ok := p.dataForType(p.metadataType.ID()); ok {
			if hash == gpui.ClipboardStringTextHash(text) && utf8.Valid(metadataBytes) {
				metadata = gpui.Some(string(metadataBytes))
			}
		}
	}
	return gpui.ClipboardEntryString{ClipboardString: gpui.ClipboardString{Text: text, Metadata: metadata}}, true
}

func (p *Pasteboard) dataForType(kind objc.ID) ([]byte, bool) {
	data := p.ns().DataForType(kind)
	if data == objc.Nil {
		return nil, false
	}
	return objc.NSDataToBytes(data), true
}

// Write replaces the pasteboard's contents with item. An empty item clears it; with more than one
// entry only the string entries are written, concatenated.
func (p *Pasteboard) Write(item gpui.ClipboardItem) {
	objc.AutoreleasePool(func() {
		p.write(item)
	})
}

func (p *Pasteboard) write(item gpui.ClipboardItem) {
	switch {
	case len(item.Entries) == 0:
		p.ns().ClearContents()
		return
	case len(item.Entries) == 1:
		switch entry := item.Entries[0].(type) {
		case gpui.ClipboardEntryString:
			p.writePlaintext(entry.ClipboardString)
			return
		case gpui.ClipboardEntryImage:
			p.writeImage(entry.Image)
			return
		case gpui.ClipboardEntryExternalPaths:
			return
		}
	}
	combined := gpui.ClipboardString{Text: "", Metadata: gpui.None[string]()}
	var text strings.Builder
	for _, entry := range item.Entries {
		if s, ok := entry.(gpui.ClipboardEntryString); ok {
			text.WriteString(s.Text)
			if combined.Metadata.IsNone() {
				combined.Metadata = s.Metadata
			}
		}
	}
	combined.Text = text.String()
	p.writePlaintext(combined)
}

func (p *Pasteboard) writePlaintext(str gpui.ClipboardString) {
	p.ns().ClearContents()

	textBytes := objc.NSDataFromBytes([]byte(str.Text))
	defer objc.Release(textBytes)
	p.ns().SetDataForType(textBytes, appkit.NSPasteboardTypeString())

	if metadata, ok := str.Metadata.Get(); ok {
		hashBytes := binary.BigEndian.AppendUint64(nil, gpui.ClipboardStringTextHash(str.Text))
		hashData := objc.NSDataFromBytes(hashBytes)
		defer objc.Release(hashData)
		p.ns().SetDataForType(hashData, p.textHashType.ID())

		metadataBytes := objc.NSDataFromBytes([]byte(metadata))
		defer objc.Release(metadataBytes)
		p.ns().SetDataForType(metadataBytes, p.metadataType.ID())
	}
}

func (p *Pasteboard) writeImage(image gpui.Image) {
	p.ns().ClearContents()

	bytes := objc.NSDataFromBytes(image.Bytes)
	defer objc.Release(bytes)
	utType := UTTypeFrom(image.Format)
	defer utType.Release()
	p.ns().SetDataForType(bytes, utType.inner())
}

func utf8Lossy(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var out strings.Builder
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size <= 1 {
			n := invalidSequenceLen(b)
			out.WriteRune(utf8.RuneError)
			b = b[n:]
			continue
		}
		out.Write(b[:size])
		b = b[size:]
	}
	return out.String()
}

func invalidSequenceLen(b []byte) int {
	first := b[0]
	var need int
	lo, hi := byte(0x80), byte(0xBF)
	switch {
	case first >= 0xC2 && first <= 0xDF:
		need = 1
	case first == 0xE0:
		need, lo = 2, 0xA0
	case first >= 0xE1 && first <= 0xEC, first == 0xEE, first == 0xEF:
		need = 2
	case first == 0xED:
		need, hi = 2, 0x9F
	case first == 0xF0:
		need, lo = 3, 0x90
	case first >= 0xF1 && first <= 0xF3:
		need = 3
	case first == 0xF4:
		need, hi = 3, 0x8F
	default:
		return 1
	}
	n := 1
	for i := 0; i < need && n < len(b); i++ {
		c := b[n]
		if c < lo || c > hi {
			break
		}
		lo, hi = 0x80, 0xBF
		n++
	}
	return n
}

// UTType is a uniform type identifier string used as a pasteboard type. Release is its drop.
type UTType struct {
	retained *objc.Retained
}

// UTTypeFrom returns the uniform type of an image format.
func UTTypeFrom(value gpui.ImageFormat) UTType {
	switch value {
	case gpui.ImageFormatPng:
		return UTTypePng()
	case gpui.ImageFormatJpeg:
		return UTTypeJpeg()
	case gpui.ImageFormatTiff:
		return UTTypeTiff()
	case gpui.ImageFormatWebp:
		return UTTypeWebp()
	case gpui.ImageFormatGif:
		return UTTypeGif()
	case gpui.ImageFormatBmp:
		return UTTypeBmp()
	case gpui.ImageFormatSvg:
		return UTTypeSvg()
	case gpui.ImageFormatIco:
		return UTTypeIco()
	default:
		return UTTypePnm()
	}
}

func utTypeFromStr(s string) UTType {
	return UTType{retained: objc.Own(objc.NSStringFromGo(s))}
}

// UTTypePng is public.png, the built-in NSPasteboardTypePNG.
func UTTypePng() UTType {
	return UTType{retained: objc.RetainObject(appkit.NSPasteboardTypePNG())}
}

// UTTypeJpeg is public.jpeg.
func UTTypeJpeg() UTType { return utTypeFromStr("public.jpeg") }

// UTTypeGif is com.compuserve.gif.
func UTTypeGif() UTType { return utTypeFromStr("com.compuserve.gif") }

// UTTypeWebp is org.webmproject.webp.
func UTTypeWebp() UTType { return utTypeFromStr("org.webmproject.webp") }

// UTTypeBmp is com.microsoft.bmp.
func UTTypeBmp() UTType { return utTypeFromStr("com.microsoft.bmp") }

// UTTypeSvg is public.svg-image.
func UTTypeSvg() UTType { return utTypeFromStr("public.svg-image") }

// UTTypeIco is com.microsoft.ico.
func UTTypeIco() UTType { return utTypeFromStr("com.microsoft.ico") }

// UTTypeTiff is public.tiff, the built-in NSPasteboardTypeTIFF.
func UTTypeTiff() UTType {
	return UTType{retained: objc.RetainObject(appkit.NSPasteboardTypeTIFF())}
}

// UTTypePnm is public.pbm.
func UTTypePnm() UTType { return utTypeFromStr("public.pbm") }

// Release drops the type string. It is idempotent.
func (t UTType) Release() { t.retained.Release() }

func (t UTType) inner() objc.ID { return t.retained.ID() }
