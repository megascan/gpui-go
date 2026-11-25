//go:build linux

package x11

import (
	"errors"
	"testing"
)

func TestClipboard_ErrorKinds(t *testing.T) {
	if !errors.Is(&Error{Kind: ErrorContentNotAvailable}, ErrContentNotAvailable) {
		t.Fatal("ContentNotAvailable does not match")
	}
	if errors.Is(ErrConversionFailure, ErrContentNotAvailable) {
		t.Fatal("kinds must differ")
	}
	if got := errorUnknown("x").Error(); got != "Unknown error while interacting with the clipboard: x" {
		t.Fatalf("message = %q", got)
	}
	formats := parseFormats(u32Bytes(1, 2, 3))
	if len(formats) != 3 || formats[2] != 3 {
		t.Fatalf("formats = %v", formats)
	}
}
