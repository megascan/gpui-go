//go:build windows

package windows

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/megascan/gpui-go/internal/win/win32"
)

func readRawClipboardText() (string, bool) {
	clip, ok := openClipboardGuard()
	if !ok {
		return "", false
	}
	defer clip.Close()
	return getClipboardString(win32.CF_UNICODETEXT)
}

func TestClipboard_ConvertDibToBmp(t *testing.T) {
	if _, ok := convertDibToBmp(make([]byte, 39)); ok {
		t.Fatal("a DIB shorter than its header converted")
	}
	dib := make([]byte, 48)
	binary.LittleEndian.PutUint32(dib[0:4], 40)
	binary.LittleEndian.PutUint16(dib[14:16], 32)
	binary.LittleEndian.PutUint32(dib[16:20], 3)
	got, ok := convertDibToBmp(dib)
	if !ok {
		t.Fatal("DIB did not convert")
	}
	if !bytes.Equal(got[:2], []byte("BM")) ||
		binary.LittleEndian.Uint32(got[2:6]) != 14+48 ||
		binary.LittleEndian.Uint32(got[10:14]) != 14+40+12 ||
		!bytes.Equal(got[14:], dib) {
		t.Fatalf("unexpected BMP header % x", got[:14])
	}
}
