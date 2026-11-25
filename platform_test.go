package gpui

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func skipOnUnimplementedStub(t *testing.T) {
	if r := recover(); r != nil {
		if message, ok := r.(string); ok && strings.HasPrefix(message, "gpui: unimplemented ") {
			t.Skip("blocked on a later stage: " + message)
		}
		panic(r)
	}
}

var exifOrientationRotate180JPG = []byte{
	0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
	0x00, 0x01, 0x00, 0x00, 0xff, 0xe1, 0x00, 0x22, 0x45, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4d, 0x4d,
	0x00, 0x2a, 0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x01, 0x12, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01,
	0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43, 0x00, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0xff, 0xc0, 0x00,
	0x0b, 0x08, 0x00, 0x20, 0x00, 0x10, 0x01, 0x01, 0x11, 0x00, 0xff, 0xc4, 0x00, 0x1f, 0x00, 0x00,
	0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0xff, 0xc4, 0x00, 0xb5, 0x10,
	0x00, 0x02, 0x01, 0x03, 0x03, 0x02, 0x04, 0x03, 0x05, 0x05, 0x04, 0x04, 0x00, 0x00, 0x01, 0x7d,
	0x01, 0x02, 0x03, 0x00, 0x04, 0x11, 0x05, 0x12, 0x21, 0x31, 0x41, 0x06, 0x13, 0x51, 0x61, 0x07,
	0x22, 0x71, 0x14, 0x32, 0x81, 0x91, 0xa1, 0x08, 0x23, 0x42, 0xb1, 0xc1, 0x15, 0x52, 0xd1, 0xf0,
	0x24, 0x33, 0x62, 0x72, 0x82, 0x09, 0x0a, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49,
	0x4a, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69,
	0x6a, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89,
	0x8a, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
	0xa8, 0xa9, 0xaa, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xba, 0xc2, 0xc3, 0xc4, 0xc5,
	0xc6, 0xc7, 0xc8, 0xc9, 0xca, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xe1, 0xe2,
	0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8,
	0xf9, 0xfa, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3f, 0x00, 0xff, 0x00, 0x3f, 0xfa,
	0x28, 0xa2, 0xbf, 0xdf, 0xe2, 0x8a, 0x28, 0xaf, 0xff, 0xd9,
}

var animatedWebp = []byte{
	82, 73, 70, 70, 192, 0, 0, 0, 87, 69, 66, 80, 86, 80, 56, 88, 10, 0, 0, 0, 2, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 65, 78, 73, 77, 6, 0, 0, 0, 255, 255, 255, 255, 0, 0, 65, 78, 77, 70,
	72, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 2, 86, 80, 56, 32, 48, 0,
	0, 0, 208, 1, 0, 157, 1, 42, 1, 0, 1, 0, 2, 0, 52, 37, 160, 2, 116, 186, 1, 248, 0, 3,
	176, 0, 254, 240, 196, 11, 255, 32, 185, 97, 117, 200, 215, 255, 32, 63, 228, 7, 252,
	128, 255, 248, 242, 0, 0, 0, 65, 78, 77, 70, 68, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 100, 0, 0, 0, 86, 80, 56, 32, 44, 0, 0, 0, 148, 1, 0, 157, 1, 42, 1, 0, 1, 0, 0,
	0, 52, 37, 160, 2, 116, 186, 0, 3, 152, 0, 254, 249, 147, 111, 255, 144, 31, 255, 144,
	31, 255, 144, 31, 255, 32, 63, 226, 23, 123, 32, 48, 0,
}

func TestPlatformImageTests_ImageToImageDataAppliesExifOrientation(t *testing.T) {
	defer skipOnUnimplementedStub(t)
	img := ImageFromBytes(ImageFormatJpeg, bytes.Clone(exifOrientationRotate180JPG))

	renderImage, err := img.ToImageData(NewSvgRenderer(nil))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := renderImage.Size(0), NewSize(DevicePixels(16), DevicePixels(32)); got != want {
		t.Fatalf("size = %v, want %v", got, want)
	}

	data, ok := renderImage.AsBytes(0)
	if !ok {
		t.Fatal("frame 0 has no bytes")
	}
	if !bytes.Equal(data[:4], []byte{255, 255, 255, 255}) {
		t.Fatalf("first pixel = %v", data[:4])
	}
	if !bytes.Equal(data[(16*32-1)*4:], []byte{0, 0, 0, 255}) {
		t.Fatalf("last pixel = %v", data[(16*32-1)*4:])
	}
}

func TestPlatformImageTests_SvgImageToImageDataConvertsToBgra(t *testing.T) {
	defer skipOnUnimplementedStub(t)
	img := ImageFromBytes(ImageFormatSvg, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1">
<rect width="1" height="1" fill="#38BDF8"/>
</svg>`))

	renderImage, err := img.ToImageData(NewSvgRenderer(nil))
	if err != nil {
		t.Fatal(err)
	}
	data, ok := renderImage.AsBytes(0)
	if !ok {
		t.Fatal("frame 0 has no bytes")
	}

	for i := 0; i+4 <= len(data); i += 4 {
		if !bytes.Equal(data[i:i+4], []byte{0xF8, 0xBD, 0x38, 0xFF}) {
			t.Fatalf("pixel %d = %v", i/4, data[i:i+4])
		}
	}
}

func TestPlatformImageTests_AnimatedWebpImageToImageDataDecodesFrames(t *testing.T) {
	defer skipOnUnimplementedStub(t)
	img := ImageFromBytes(ImageFormatWebp, bytes.Clone(animatedWebp))
	renderImage, err := img.ToImageData(NewSvgRenderer(nil))
	if err != nil {
		t.Fatal(err)
	}

	if got := renderImage.FrameCount(); got != 2 {
		t.Fatalf("frame count = %d, want 2", got)
	}
	if got, want := renderImage.Size(0), NewSize(DevicePixels(1), DevicePixels(1)); got != want {
		t.Fatalf("size = %v, want %v", got, want)
	}
	if got := renderImage.Delay(0); got != 100*time.Millisecond {
		t.Fatalf("delay(0) = %v", got)
	}
	if got := renderImage.Delay(1); got != 100*time.Millisecond {
		t.Fatalf("delay(1) = %v", got)
	}
}

func TestPlatformImageDecode_ExifOrientationFrames(t *testing.T) {
	frames, err := decodeStaticImage(exifOrientationRotate180JPG, ImageFormatJpeg)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 {
		t.Fatalf("frames = %d", len(frames))
	}
	buffer := frames[0].Buffer
	if buffer.Rect.Dx() != 16 || buffer.Rect.Dy() != 32 {
		t.Fatalf("size = %v", buffer.Rect)
	}
	if !bytes.Equal(buffer.Pix[:4], []byte{255, 255, 255, 255}) {
		t.Fatalf("first pixel = %v", buffer.Pix[:4])
	}
	if !bytes.Equal(buffer.Pix[(16*32-1)*4:], []byte{0, 0, 0, 255}) {
		t.Fatalf("last pixel = %v", buffer.Pix[(16*32-1)*4:])
	}
}

func TestPlatformImageDecode_AnimatedWebpFrames(t *testing.T) {
	frames, err := decodeWebpImage(animatedWebp)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	for i, frame := range frames {
		if frame.Buffer.Rect.Dx() != 1 || frame.Buffer.Rect.Dy() != 1 {
			t.Fatalf("frame %d size = %v", i, frame.Buffer.Rect)
		}
		if frame.Delay != 100*time.Millisecond {
			t.Fatalf("frame %d delay = %v", i, frame.Delay)
		}
	}
	red := frames[0].Buffer.Pix
	if red[2] < 200 || red[0] > 60 || red[3] != 255 {
		t.Fatalf("frame 0 BGRA = %v, want red", red)
	}
	blue := frames[1].Buffer.Pix
	if blue[0] < 200 || blue[2] > 60 || blue[3] != 255 {
		t.Fatalf("frame 1 BGRA = %v, want blue", blue)
	}
}

func TestPlatformImageDecode_PngIsBgra(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 4})
	src.SetNRGBA(1, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, src); err != nil {
		t.Fatal(err)
	}
	frames, err := decodeStaticImage(encoded.Bytes(), ImageFormatPng)
	if err != nil {
		t.Fatal(err)
	}
	if got := frames[0].Buffer.Pix; !bytes.Equal(got, []byte{3, 2, 1, 4, 30, 20, 10, 255}) {
		t.Fatalf("pixels = %v", got)
	}
}

func TestPlatformImageDecode_Pnm(t *testing.T) {
	cases := []struct {
		name string
		data string
		want []byte
	}{
		{"ascii pbm", "P1\n# c\n2 1\n0 1\n", []byte{255, 255, 255, 255, 0, 0, 0, 255}},
		{"binary pgm scaled", "P5 2 1 15\n\x0f\x00", []byte{255, 255, 255, 255, 0, 0, 0, 255}},
		{"ascii ppm", "P3 1 1 255 1 2 3", []byte{3, 2, 1, 255}},
		{"pam rgba", "P7\nWIDTH 1\nHEIGHT 1\nDEPTH 4\nMAXVAL 255\nTUPLTYPE RGB_ALPHA\nENDHDR\n\x01\x02\x03\x04", []byte{3, 2, 1, 4}},
	}
	for _, c := range cases {
		frames, err := decodeStaticImage([]byte(c.data), ImageFormatPnm)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := frames[0].Buffer.Pix; !bytes.Equal(got, c.want) {
			t.Fatalf("%s: pixels = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPlatformImageDecode_IcoBmpWithMask(t *testing.T) {
	var dib bytes.Buffer
	header := make([]byte, 40)
	header[0] = 40
	header[4] = 2
	header[8] = 2
	header[12] = 1
	header[14] = 32
	dib.Write(header)
	dib.Write([]byte{1, 2, 3, 255, 4, 5, 6, 255})
	dib.Write([]byte{0x80, 0, 0, 0})
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := []byte{2, 1, 0, 0, 1, 0, 32, 0, byte(dib.Len()), 0, 0, 0, 22, 0, 0, 0}
	ico.Write(entry)
	ico.Write(dib.Bytes())
	frames, err := decodeStaticImage(ico.Bytes(), ImageFormatIco)
	if err != nil {
		t.Fatal(err)
	}
	if got := frames[0].Buffer.Pix; !bytes.Equal(got, []byte{1, 2, 3, 0, 4, 5, 6, 255}) {
		t.Fatalf("pixels = %v", got)
	}
}

func seaReferenceHash(buf []byte) uint64 {
	state := [4]uint64{0x16f11fe89b0d677c, 0xb480a793d8e6c86c, 0x6fe2e5aaf078ebc9, 0x14f994a4c5259381}
	for i := 0; i < len(buf); i += 8 {
		x := seaReadInt(buf[i:min(i+8, len(buf))])
		a := seaDiffuse(state[0] ^ x)
		state[0], state[1], state[2], state[3] = state[1], state[2], state[3], a
	}
	return seaDiffuse(state[0] ^ state[1] ^ state[2] ^ state[3] ^ uint64(len(buf)))
}

func TestPlatformSeaHasher_MatchesReference(t *testing.T) {
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i*31 + 7)
	}
	for n := range len(data) {
		for split := 0; split <= n; split += max(1, n/7) {
			h := newSeaHasher()
			h.write(data[:split])
			h.write(data[split:n])
			if got, want := h.finish(), seaReferenceHash(data[:n]); got != want {
				t.Fatalf("len %d split %d: got %#x want %#x", n, split, got, want)
			}
		}
	}
	hasher := newSeaHasher()
	hasher.write([]byte("hello"))
	hasher.write([]byte{0xff})
	if got := ClipboardStringTextHash("hello"); got != hasher.finish() {
		t.Fatalf("text hash = %#x", got)
	}
}

func TestPlatformCursorStyle_JSON(t *testing.T) {
	for style := range CursorStyleIter() {
		encoded, err := json.Marshal(style)
		if err != nil {
			t.Fatal(err)
		}
		var decoded CursorStyle
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded != style {
			t.Fatalf("%s round-tripped to %s", encoded, decoded)
		}
	}
	encoded, _ := json.Marshal(CursorStyleResizeUpLeftDownRight)
	if string(encoded) != `"ResizeUpLeftDownRight"` {
		t.Fatalf("encoded = %s", encoded)
	}
	var style CursorStyle
	if err := json.Unmarshal([]byte(`"Nope"`), &style); err == nil {
		t.Fatal("unknown variant decoded")
	}
}

func TestPlatformClipboard_Basics(t *testing.T) {
	item := ClipboardItem{Entries: []ClipboardEntry{
		ClipboardEntryFromString("a"),
		ClipboardEntryExternalPaths{ExternalPaths{"/x"}},
		ClipboardEntryFromString("b"),
	}}
	if text, ok := item.Text(); !ok || text != "ab" {
		t.Fatalf("text = %q, %v", text, ok)
	}
	paths := ClipboardItem{Entries: []ClipboardEntry{ClipboardEntryExternalPaths{ExternalPaths{"/x", "/y"}}}}
	if text, ok := paths.Text(); !ok || text != "/x/y" {
		t.Fatalf("paths text = %q, %v", text, ok)
	}
	if _, ok := (ClipboardItem{}).Text(); ok {
		t.Fatal("empty item has text")
	}
	withMetadata := NewClipboardItemStringWithJSONMetadata("t", map[string]int{"k": 1})
	metadata, ok := withMetadata.Metadata()
	if !ok || metadata != `{"k":1}` {
		t.Fatalf("metadata = %q, %v", metadata, ok)
	}
	decoded, ok := withMetadata.Entries[0].(ClipboardEntryString).MetadataJSON[map[string]int]()
	if !ok || decoded["k"] != 1 {
		t.Fatalf("decoded = %v, %v", decoded, ok)
	}
	if err := error(ClipboardReadErrorDenied("no")); err.Error() != "clipboard access was denied: no" {
		t.Fatalf("error = %q", err)
	}
	if !errors.Is(error(ClipboardReadErrorUnavailable()), ClipboardReadErrorUnavailable()) {
		t.Fatal("errors.Is failed")
	}
}

func TestPlatformPromptAndFormats(t *testing.T) {
	if b := PromptButtonFromStr("OK"); b != PromptButtonOk("OK") {
		t.Fatalf("ok = %v", b)
	}
	if b := PromptButtonFromStr("CANCEL"); !b.IsCancel() || b.Label() != "Cancel" {
		t.Fatalf("cancel = %v", b)
	}
	if b := PromptButtonFromStr("Save"); b != NewPromptButton("Save") {
		t.Fatalf("other = %v", b)
	}
	for _, c := range []struct {
		mime string
		want ImageFormat
	}{{"image/png", ImageFormatPng}, {"image/ico", ImageFormatIco}, {"image/jpg", ImageFormatJpeg}, {"image/tif", ImageFormatTiff}} {
		if got, ok := ImageFormatFromMimeType(c.mime); !ok || got != c.want {
			t.Fatalf("%s = %v, %v", c.mime, got, ok)
		}
	}
	if _, ok := ImageFormatFromMimeType("image/avif"); ok {
		t.Fatal("avif matched")
	}
	options := DefaultWindowOptions()
	if v, _ := options.InactiveFrameInterval.Get(); v != 33333*time.Microsecond {
		t.Fatalf("inactive = %v", v)
	}
	if v, _ := options.ThermalFrameInterval.Get(); v != 16667*time.Microsecond {
		t.Fatalf("thermal = %v", v)
	}
	if options.Kind != (WindowKindNormal{}) || !options.Titlebar.IsSome() {
		t.Fatalf("options = %+v", options)
	}
	if (PopupNotSupportedError{}).Error() != "popups are not supported on this platform" {
		t.Fatal("popup error message")
	}
}

func TestPlatformInputHandler_ComputeIMECandidateBounds(t *testing.T) {
	lines := map[int]Pixels{0: 0, 1: 0, 2: 0, 3: 20, 4: 20}
	boundsFor := func(r Range) (Bounds[Pixels], bool) {
		y, ok := lines[r.Start]
		return Bounds[Pixels]{Origin: Point[Pixels]{X: Pixels(r.Start), Y: y}}, ok
	}
	selection := UTF16Selection{Range: Range{Start: 4, End: 4}}
	got, ok := PlatformInputHandlerComputeIMECandidateBounds(Some(Range{Start: 1, End: 4}), &selection, boundsFor)
	if !ok || got.Origin.X != 3 {
		t.Fatalf("marked = %v, %v", got, ok)
	}
	reversed := UTF16Selection{Range: Range{Start: 1, End: 4}, Reversed: true}
	got, ok = PlatformInputHandlerComputeIMECandidateBounds(None[Range](), &reversed, boundsFor)
	if !ok || got.Origin.X != 1 {
		t.Fatalf("selection = %v, %v", got, ok)
	}
}

var atlasTestTileSize = Size[DevicePixels]{Width: 1, Height: 1}

type recordingAtlasBackend struct {
	insertCalls    uint32
	failNextInsert bool
	removedTiles   []AtlasTile
}

func (b *recordingAtlasBackend) Insert(kind AtlasTextureKind, size Size[DevicePixels], _ []byte) (AtlasTile, error) {
	b.insertCalls++
	if b.failNextInsert {
		b.failNextInsert = false
		return AtlasTile{}, errors.New("backend failed")
	}
	return AtlasTile{
		TextureID: AtlasTextureID{Index: 0, Kind: kind},
		TileID:    TileID(b.insertCalls),
		Padding:   0,
		Bounds:    Bounds[DevicePixels]{Size: size},
	}, nil
}

func (b *recordingAtlasBackend) Remove(tile AtlasTile) {
	b.removedTiles = append(b.removedTiles, tile)
}

func atlasTestImageKey(imageID int) AtlasKey {
	return AtlasKeyFromImage(RenderImageParams{ImageID: ImageID(imageID), FrameIndex: 0})
}

func atlasTestBuildTile() (Size[DevicePixels], []byte, bool, error) {
	return atlasTestTileSize, []byte{0, 0, 0, 255}, true, nil
}

func TestPlatformAtlasTests_OnlySuccessfulInsertsAreCached(t *testing.T) {
	state := NewAtlasState(&recordingAtlasBackend{})
	key := atlasTestImageKey(1)

	if _, ok, err := state.GetOrInsertWith(key, func() (Size[DevicePixels], []byte, bool, error) {
		return Size[DevicePixels]{}, nil, false, nil
	}); err != nil || ok {
		t.Fatalf("empty build = %v, %v", ok, err)
	}
	if _, _, err := state.GetOrInsertWith(key, func() (Size[DevicePixels], []byte, bool, error) {
		return Size[DevicePixels]{}, nil, false, errors.New("builder failed")
	}); err == nil {
		t.Fatal("builder error should propagate")
	}
	if state.Contains(key) {
		t.Fatal("failed build was cached")
	}
	if state.Backend.insertCalls != 0 {
		t.Fatalf("insert calls = %d", state.Backend.insertCalls)
	}

	state.Backend.failNextInsert = true
	if _, _, err := state.GetOrInsertWith(key, atlasTestBuildTile); err == nil {
		t.Fatal("backend error should propagate")
	}
	if state.Contains(key) {
		t.Fatal("failed insert was cached")
	}
	if state.Backend.insertCalls != 1 {
		t.Fatalf("insert calls = %d", state.Backend.insertCalls)
	}

	tile, ok, err := state.GetOrInsertWith(key, atlasTestBuildTile)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("builder should produce a tile")
	}
	if tile.TextureID.Kind != key.TextureKind() {
		t.Fatalf("kind = %v", tile.TextureID.Kind)
	}
	cached, ok, err := state.GetOrInsertWith(key, func() (Size[DevicePixels], []byte, bool, error) {
		return Size[DevicePixels]{}, nil, false, errors.New("cache hit must not call the builder")
	})
	if err != nil || !ok || cached != tile {
		t.Fatalf("cache hit = %v, %v, %v", cached, ok, err)
	}
	if !state.Contains(key) {
		t.Fatal("tile not cached")
	}
	if state.Backend.insertCalls != 2 {
		t.Fatalf("insert calls = %d", state.Backend.insertCalls)
	}
}

func TestPlatformAtlasTests_RemoveAndClearInvalidateKeys(t *testing.T) {
	state := NewAtlasState(&recordingAtlasBackend{})
	key := atlasTestImageKey(1)
	otherKey := atlasTestImageKey(2)
	tile, ok, err := state.GetOrInsertWith(key, atlasTestBuildTile)
	if err != nil || !ok {
		t.Fatalf("builder should produce a tile: %v", err)
	}
	if _, ok, err := state.GetOrInsertWith(otherKey, atlasTestBuildTile); err != nil || !ok {
		t.Fatalf("builder should produce another tile: %v", err)
	}

	state.Remove(key)
	state.Remove(key)
	if state.Contains(key) {
		t.Fatal("removed key still cached")
	}
	if !state.Contains(otherKey) {
		t.Fatal("other key was removed")
	}
	if len(state.Backend.removedTiles) != 1 || state.Backend.removedTiles[0] != tile {
		t.Fatalf("removed tiles = %v", state.Backend.removedTiles)
	}

	resetCalls := 0
	state.Clear(func(*recordingAtlasBackend) { resetCalls++ })
	if resetCalls != 1 {
		t.Fatalf("reset calls = %d", resetCalls)
	}
	if state.Contains(otherKey) {
		t.Fatal("clear kept a key")
	}
	if len(state.Backend.removedTiles) != 1 || state.Backend.removedTiles[0] != tile {
		t.Fatalf("removed tiles = %v", state.Backend.removedTiles)
	}
}

func TestPlatformFrameSignalTests_CoalescedSignalsRetainTheEarliestUntilDrained(t *testing.T) {
	if !profilerFeatureEnabled {
		t.Skip("needs the gpui_profiler build tag")
	}
	for _, sources := range [][2]FrameRequestSource{
		{FrameRequestSourceNativeCallback, FrameRequestSourceLocalSchedule},
		{FrameRequestSourceLocalSchedule, FrameRequestSourceNativeCallback},
	} {
		firstSource, laterSource := sources[0], sources[1]
		signal := NewPlatformFrameSignal()
		first, ok := PlatformFrameSignalCapture(scheduler.InstantNow)
		if !ok {
			t.Fatal("profiling captures requests")
		}
		if _, _, ok := signal.Take(); ok {
			t.Fatal("new signal is not empty")
		}
		signal.Record(first.Add(16*time.Millisecond), laterSource)
		signal.Record(first, firstSource)
		signal.Record(first.Add(32*time.Millisecond), laterSource)
		if at, source, ok := signal.Take(); !ok || !at.Equal(first) || source != firstSource {
			t.Fatalf("take = %v, %v, %v", at, source, ok)
		}
		if _, _, ok := signal.Take(); ok {
			t.Fatal("drained signal is not empty")
		}
		next := first.Add(48 * time.Millisecond)
		signal.Record(next, laterSource)
		if at, source, ok := signal.Take(); !ok || !at.Equal(next) || source != laterSource {
			t.Fatalf("take = %v, %v, %v", at, source, ok)
		}
	}
}

func TestPlatformFrameSignalTests_DisabledProfilingDoesNotCaptureOrStoreTimestamps(t *testing.T) {
	if profilerFeatureEnabled {
		t.Skip("runs without the gpui_profiler build tag")
	}
	if _, ok := PlatformFrameSignalCapture(func() scheduler.Instant {
		panic("profiler-disabled builds must not evaluate the capture closure")
	}); ok {
		t.Fatal("captured a timestamp")
	}
	signal := NewPlatformFrameSignal()
	signal.Record(scheduler.InstantNow(), FrameRequestSourceLocalSchedule)
	if _, _, ok := signal.Take(); ok {
		t.Fatal("stored a timestamp")
	}
}

func skipUnlessLinuxOrFreeBSD(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "freebsd" {
		t.Skip("upstream runs this test on linux and freebsd only")
	}
}

type buttonSide = [MaxButtonsPerSide]Option[WindowButton]

func buttons(values ...WindowButton) buttonSide {
	var side buttonSide
	for i, value := range values {
		side[i] = Some(value)
	}
	return side
}

func mustParseLayout(t *testing.T, layout string) WindowButtonLayout {
	t.Helper()
	parsed, err := WindowButtonLayoutParse(layout)
	if err != nil {
		t.Fatalf("parse %q: %v", layout, err)
	}
	return parsed
}

func assertSide(t *testing.T, name string, got, want buttonSide) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestPlatform_RefreshIntervalFromHz(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	if got, ok := RefreshIntervalFromHz(50.0); !ok || got != 20*time.Millisecond {
		t.Fatalf("50 Hz = %v, %v", got, ok)
	}
	if got, ok := RefreshIntervalFromHz(120.0); !ok || got != time.Second/120 {
		t.Fatalf("120 Hz = %v, %v", got, ok)
	}
	for _, unknown := range []float64{0.0, -60.0, math.NaN(), math.Inf(1)} {
		if got, ok := RefreshIntervalFromHz(unknown); ok {
			t.Fatalf("%v Hz = %v", unknown, got)
		}
	}
}

func TestPlatform_WindowButtonLayoutParseStandard(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close,minimize:maximize")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose, WindowButtonMinimize))
	assertSide(t, "right", layout.Right, buttons(WindowButtonMaximize))
}

func TestPlatform_WindowButtonLayoutParseRightOnly(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "minimize,maximize,close")
	assertSide(t, "left", layout.Left, buttons())
	assertSide(t, "right", layout.Right, buttons(WindowButtonMinimize, WindowButtonMaximize, WindowButtonClose))
}

func TestPlatform_WindowButtonLayoutParseLeftOnly(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close,minimize,maximize:")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose, WindowButtonMinimize, WindowButtonMaximize))
	assertSide(t, "right", layout.Right, buttons())
}

func TestPlatform_WindowButtonLayoutParseWithWhitespace(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, " close , minimize : maximize ")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose, WindowButtonMinimize))
	assertSide(t, "right", layout.Right, buttons(WindowButtonMaximize))
}

func TestPlatform_WindowButtonLayoutParseEmpty(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "")
	assertSide(t, "left", layout.Left, buttons())
	assertSide(t, "right", layout.Right, buttons())
}

func TestPlatform_WindowButtonLayoutParseIntentionallyEmpty(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, ":")
	assertSide(t, "left", layout.Left, buttons())
	assertSide(t, "right", layout.Right, buttons())
}

func TestPlatform_WindowButtonLayoutParseInvalidButtons(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close,invalid,minimize:maximize,foo")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose, WindowButtonMinimize))
	assertSide(t, "right", layout.Right, buttons(WindowButtonMaximize))
}

func TestPlatform_WindowButtonLayoutParseDeduplicatesSameSideButtons(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close,close,minimize")
	assertSide(t, "right", layout.Right, buttons(WindowButtonClose, WindowButtonMinimize))
	if got := layout.format(); got != ":close,minimize" {
		t.Fatalf("format = %q", got)
	}
}

func TestPlatform_WindowButtonLayoutParseDeduplicatesButtonsAcrossSides(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close:maximize,close,minimize")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose))
	assertSide(t, "right", layout.Right, buttons(WindowButtonMaximize, WindowButtonMinimize))

	var buttonIDs []string
	for _, side := range []buttonSide{layout.Left, layout.Right} {
		for _, button := range side {
			if value, ok := button.Get(); ok {
				buttonIDs = append(buttonIDs, value.ID())
			}
		}
	}
	unique := map[string]struct{}{}
	for _, id := range buttonIDs {
		unique[id] = struct{}{}
	}
	if len(unique) != len(buttonIDs) {
		t.Fatalf("duplicate button ids: %v", buttonIDs)
	}
	if got := layout.format(); got != "close:maximize,minimize" {
		t.Fatalf("format = %q", got)
	}
}

func TestPlatform_WindowButtonLayoutParseGnomeStyle(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close")
	assertSide(t, "left", layout.Left, buttons())
	assertSide(t, "right", layout.Right, buttons(WindowButtonClose))
}

func TestPlatform_WindowButtonLayoutParseElementaryStyle(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := mustParseLayout(t, "close:maximize")
	assertSide(t, "left", layout.Left, buttons(WindowButtonClose))
	assertSide(t, "right", layout.Right, buttons(WindowButtonMaximize))
}

func TestPlatform_WindowButtonLayoutRoundTrip(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	for _, c := range []string{
		"close:minimize,maximize",
		"minimize,maximize,close:",
		":close",
		"close:",
		"close:maximize",
		":",
	} {
		layout := mustParseLayout(t, c)
		if got := layout.format(); got != c {
			t.Fatalf("Round-trip failed for: %s (got %q)", c, got)
		}
	}
}

func TestPlatform_WindowButtonLayoutLinuxDefault(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	layout := WindowButtonLayoutLinuxDefault()
	assertSide(t, "left", layout.Left, buttons())
	assertSide(t, "right", layout.Right, buttons(WindowButtonMinimize, WindowButtonMaximize, WindowButtonClose))

	roundTripped := mustParseLayout(t, layout.format())
	if roundTripped != layout {
		t.Fatalf("round trip = %v, want %v", roundTripped, layout)
	}
}

func TestPlatform_WindowButtonLayoutParseAllInvalid(t *testing.T) {
	skipUnlessLinuxOrFreeBSD(t)
	if _, err := WindowButtonLayoutParse("asdfghjkl"); err == nil {
		t.Fatal("parse succeeded")
	}
}
