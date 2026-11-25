//go:build windows

package win32

var (
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
)

// CreateSolidBrush creates a brush of a COLORREF (0x00BBGGRR).
func CreateSolidBrush(color uint32) HBRUSH {
	r, _, _ := procCreateSolidBrush.Call(uintptr(color))
	return HBRUSH(r)
}

// DeleteObject deletes a GDI object.
func DeleteObject(object HGDIOBJ) bool {
	r, _, _ := procDeleteObject.Call(uintptr(object))
	return r != 0
}

// GetStockObject returns a stock pen, brush, font or palette.
func GetStockObject(index int32) HGDIOBJ {
	r, _, _ := procGetStockObject.Call(uintptr(index))
	return HGDIOBJ(r)
}
