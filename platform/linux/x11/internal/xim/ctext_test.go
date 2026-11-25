//go:build linux

package xim

import (
	"bytes"
	"errors"
	"testing"
)

func TestCtext_KoreanFixture(t *testing.T) {
	const text = "가나다"
	want := []byte{27, 37, 71, 234, 176, 128, 235, 130, 152, 235, 139, 164, 27, 37, 64}
	if got := UTF8ToCompoundText(text); !bytes.Equal(got, want) {
		t.Fatalf("UTF8ToCompoundText = % v, want % v", got, want)
	}
	got, err := CompoundTextToUTF8(want)
	if err != nil || got != text {
		t.Fatalf("CompoundTextToUTF8 = %q, %v; want %q", got, err, text)
	}
}

func TestCtext_PlainTextPassesThrough(t *testing.T) {
	got, err := CompoundTextToUTF8([]byte("abc"))
	if err != nil || got != "abc" {
		t.Fatalf("CompoundTextToUTF8 = %q, %v", got, err)
	}
}

func TestCtext_EmptyInput(t *testing.T) {
	got, err := CompoundTextToUTF8(nil)
	if err != nil || got != "" {
		t.Fatalf("CompoundTextToUTF8(nil) = %q, %v", got, err)
	}
}

func TestCtext_JapaneseEscapeIsUnsupported(t *testing.T) {
	jis := []byte{27, 36, 40, 66, 69, 108, 53, 126}
	if _, err := CompoundTextToUTF8(jis); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Fatalf("CompoundTextToUTF8 ISO-2022-JP = %v, want ErrUnsupportedEncoding", err)
	}
}

func TestCtext_RejectsTruncatedEscape(t *testing.T) {
	if _, err := CompoundTextToUTF8([]byte{27, 37, 71, 'a'}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("CompoundTextToUTF8 short escape = %v, want ErrMalformed", err)
	}
}
