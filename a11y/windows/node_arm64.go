//go:build windows

package windows

import (
	"math"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	elementProviderFromPointTarget uintptr
	rangeValueSetValueTarget       uintptr
	textRangeFromPointTarget       uintptr
)

func elementProviderFromPointTramp()
func rangeValueSetValueTramp()
func textRangeFromPointTramp()
func elementProviderFromPointTrampAddr() uintptr
func rangeValueSetValueTrampAddr() uintptr
func textRangeFromPointTrampAddr() uintptr

func elementProviderFromPointSlot() uintptr {
	elementProviderFromPointTarget = windows.NewCallback(func(this *com.IUnknown, x, y uintptr, out **com.IUnknown) uintptr {
		return elementProviderFromPointCallback(this, math.Float64frombits(uint64(x)), math.Float64frombits(uint64(y)), out)
	})
	return elementProviderFromPointTrampAddr()
}

func rangeValueSetValueSlot() uintptr {
	rangeValueSetValueTarget = windows.NewCallback(func(this *com.IUnknown, value uintptr) uintptr {
		return rangeValueSetValueCallback(this, math.Float64frombits(uint64(value)))
	})
	return rangeValueSetValueTrampAddr()
}

func textRangeFromPointSlot() uintptr {
	textRangeFromPointTarget = windows.NewCallback(func(this *com.IUnknown, x, y uintptr, out **com.IUnknown) uintptr {
		return textRangeFromPointCallback(this, uiaPoint{X: math.Float64frombits(uint64(x)), Y: math.Float64frombits(uint64(y))}, out)
	})
	return textRangeFromPointTrampAddr()
}
