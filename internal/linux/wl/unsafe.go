//go:build linux

package wl

import (
	"fmt"
	"strings"
	"unsafe"
)

func ptrAt(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

func load32(p uintptr) uint32 {
	return *(*uint32)(ptrAt(p))
}

func load64(p uintptr) uint64 {
	return *(*uint64)(ptrAt(p))
}

func loadPtr(p uintptr) uintptr {
	return *(*uintptr)(ptrAt(p))
}

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(ptrAt(p + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptrAt(p)), n))
}

func goBytes(p uintptr, n int) []byte {
	if p == 0 || n == 0 {
		return []byte{}
	}
	out := make([]byte, n)
	copy(out, unsafe.Slice((*byte)(ptrAt(p)), n))
	return out
}

func cString(s string) []byte {
	b, err := cStringChecked(s)
	if err != nil {
		panic("wl: " + err.Error())
	}
	return b
}

func cStringChecked(s string) ([]byte, error) {
	if i := strings.IndexByte(s, 0); i >= 0 {
		return nil, fmt.Errorf("nul byte found in provided data at position: %d", i)
	}
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b, nil
}
