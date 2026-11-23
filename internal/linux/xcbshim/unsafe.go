//go:build linux

package xcbshim

import "unsafe"

func ptrAt(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

func load8(p uintptr) uint8 {
	return *(*uint8)(ptrAt(p))
}

func load16(p uintptr) uint16 {
	return *(*uint16)(ptrAt(p))
}

func load32(p uintptr) uint32 {
	return *(*uint32)(ptrAt(p))
}
