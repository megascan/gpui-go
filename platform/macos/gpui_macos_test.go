//go:build darwin && !ios

package macos

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func TestGpuiMacos_NSRange(t *testing.T) {
	if _, ok := nsRangeInvalid().toRange(); ok {
		t.Fatal("invalid range converted")
	}
	r := nsRangeFromRange(gpui.Range{Start: 3, End: 7})
	if got, ok := r.toRange(); !ok || got != (gpui.Range{Start: 3, End: 7}) {
		t.Fatalf("toRange = %v %v", got, ok)
	}
}
