//go:build linux

package xim

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

// ErrUnsupportedEncoding reports compound text in a non-UTF-8 designation (ISO-2022-JP, CN, KR).
var ErrUnsupportedEncoding = errors.New("xim: unsupported compound text encoding")

var (
	ctextUTF8Start = []byte{0x1B, 0x25, 0x47}
	ctextUTF8End   = []byte{0x1B, 0x25, 0x40}
)

// UTF8ToCompoundText wraps text in the UTF-8 escape of COMPOUND_TEXT.
func UTF8ToCompoundText(text string) []byte {
	out := make([]byte, 0, len(text)+6)
	out = append(out, ctextUTF8Start...)
	out = append(out, text...)
	return append(out, ctextUTF8End...)
}

// CompoundTextToUTF8 decodes COMPOUND_TEXT that is either unescaped or UTF-8 escaped.
func CompoundTextToUTF8(b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	if b[0] != 0x1B {
		if !utf8.Valid(b) {
			return "", fmt.Errorf("xim: compound text is not utf-8: %w", ErrUnsupportedEncoding)
		}
		return string(b), nil
	}
	if len(b) < 3 {
		return "", fmt.Errorf("xim: compound text escape: %w", ErrMalformed)
	}
	switch {
	case bytes.Equal(b[:3], ctextUTF8Start):
		body := b[3:]
		if len(body) < 3 {
			return "", fmt.Errorf("xim: compound text utf-8 escape: %w", ErrMalformed)
		}
		body = body[:len(body)-3]
		if !utf8.Valid(body) {
			return "", fmt.Errorf("xim: compound text is not utf-8: %w", ErrMalformed)
		}
		return string(body), nil
	case bytes.HasPrefix(b, []byte{0x1B, 0x24, 0x28}):
		return "", ErrUnsupportedEncoding
	}
	return "", fmt.Errorf("xim: compound text escape: %w", ErrMalformed)
}
