//go:build linux

package xcursor

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"
)

func sec026Stream(width, height uint32) []byte {
	var b bytes.Buffer
	le := func(vs ...uint32) {
		for _, v := range vs {
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
	}
	le(0x72756358, 16, 1, 1, 0xfffd0002, 16, 28)
	le(36, 0xfffd0002, 16, 1, width, height, 0, 0, 0)
	return b.Bytes()
}

func TestSecurity_SEC026_xcursor_allocation_from_header(t *testing.T) {
	data := sec026Stream(0x2000, 0x2000)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := ParseXcursorStream(bytes.NewReader(data))
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatal("truncated image with a header larger than the file must fail")
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 16<<20 {
		t.Fatalf("parser allocated %d bytes for a %d-byte file", grown, len(data))
	}
}
