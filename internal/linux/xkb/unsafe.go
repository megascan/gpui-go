//go:build linux

package xkb

import "unsafe"

func unsafeAdd(p uintptr, n int) unsafe.Pointer {
	return unsafe.Add(*(*unsafe.Pointer)(unsafe.Pointer(&p)), n)
}

func unsafeBytes(p uintptr, n int) []byte {
	return unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), n)
}

func unsafeKeysyms(p uintptr, n int) []Keysym {
	if p == 0 || n <= 0 {
		return nil
	}
	return unsafe.Slice((*Keysym)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), n)
}

func uintptrOf(b *byte) uintptr { return uintptr(unsafe.Pointer(b)) }
