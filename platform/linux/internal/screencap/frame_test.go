//go:build linux

package screencap

import (
	"bytes"
	"testing"
)

func TestFrame_RemoveAlphaChannel(t *testing.T) {
	if got := RemoveAlphaChannel([]byte{1, 2, 3, 0}); !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("got %v", got)
	}
	if got := RemoveAlphaChannel([]byte{1, 2, 3, 4, 5, 6, 7, 8}); !bytes.Equal(got, []byte{1, 2, 3, 5, 6, 7}) {
		t.Fatalf("got %v", got)
	}
}

func TestFrame_ConvertBGRAToRGB(t *testing.T) {
	if got := ConvertBGRAToRGB([]byte{1, 2, 3, 0}); !bytes.Equal(got, []byte{3, 2, 1}) {
		t.Fatalf("got %v", got)
	}
	if got := ConvertBGRAToRGB([]byte{1, 2, 3, 4, 5, 6, 7, 8}); !bytes.Equal(got, []byte{3, 2, 1, 7, 6, 5}) {
		t.Fatalf("got %v", got)
	}
}

func TestFrame_GetCroppedData(t *testing.T) {
	rgba := func(n byte) []byte { return []byte{n, n, n, n} }
	var data []byte
	for i := byte(1); i <= 9; i++ {
		data = append(data, rgba(i)...)
	}
	var expected []byte
	for _, i := range []byte{1, 2, 4, 5, 7, 8} {
		expected = append(expected, rgba(i)...)
	}
	if got := GetCroppedData(data, 3, 3, 2); !bytes.Equal(got, expected) {
		t.Fatalf("got %v, want %v", got, expected)
	}
}
