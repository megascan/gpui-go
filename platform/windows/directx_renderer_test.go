//go:build windows

package windows

import (
	"testing"
	"unsafe"
)

func TestDirectxRenderer_ConstantBufferLayout(t *testing.T) {
	if got := unsafe.Sizeof(globalParams{}); got != 48 {
		t.Fatalf("globalParams size = %d, want 48", got)
	}
	if got := unsafe.Sizeof(batchParams{}); got != 16 {
		t.Fatalf("batchParams size = %d, want 16", got)
	}
	if got := nextPowerOfTwo(65); got != 128 {
		t.Fatalf("nextPowerOfTwo(65) = %d, want 128", got)
	}
}
