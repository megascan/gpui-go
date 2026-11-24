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
)

func elementProviderFromPointTramp()
func rangeValueSetValueTramp()
func elementProviderFromPointTrampAddr() uintptr
func rangeValueSetValueTrampAddr() uintptr

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
	return windows.NewCallback(func(this *com.IUnknown, point *uiaPoint, out **com.IUnknown) uintptr {
		return textRangeFromPointCallback(this, *point, out)
	})
}
