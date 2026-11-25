//go:build windows

package com

import (
	"math"
	"unicode/utf16"
	"unsafe"
)

var (
	procSysAllocStringLen     = oleaut.NewProc("SysAllocStringLen")
	procSysFreeString         = oleaut.NewProc("SysFreeString")
	procSysStringLen          = oleaut.NewProc("SysStringLen")
	procVariantClear          = oleaut.NewProc("VariantClear")
	procVariantInit           = oleaut.NewProc("VariantInit")
	procSafeArrayCreateVector = oleaut.NewProc("SafeArrayCreateVector")
	procSafeArrayDestroy      = oleaut.NewProc("SafeArrayDestroy")
	procSafeArrayPutElement   = oleaut.NewProc("SafeArrayPutElement")
	procSafeArrayGetElement   = oleaut.NewProc("SafeArrayGetElement")
	procSafeArrayGetLBound    = oleaut.NewProc("SafeArrayGetLBound")
	procSafeArrayGetUBound    = oleaut.NewProc("SafeArrayGetUBound")
	procSafeArrayGetVartype   = oleaut.NewProc("SafeArrayGetVartype")
)

// BSTR is a length-prefixed UTF-16 string allocated by SysAllocString; nil is the empty string.
type BSTR *uint16

// NewBSTR allocates a BSTR holding s; the caller (or the callee it is handed to) frees it.
func NewBSTR(s string) (BSTR, error) {
	u := utf16.Encode([]rune(s))
	var first *uint16
	if len(u) > 0 {
		first = &u[0]
	}
	r, _, _ := procSysAllocStringLen.Call(uintptr(unsafe.Pointer(first)), uintptr(len(u)))
	if r == 0 {
		return nil, E_OUTOFMEMORY
	}
	return BSTR((*uint16)(PtrFromUintptr(r))), nil
}

// BSTRLen returns the length of b in UTF-16 units.
func BSTRLen(b BSTR) int {
	if b == nil {
		return 0
	}
	r, _, _ := procSysStringLen.Call(uintptr(unsafe.Pointer(b)))
	return int(uint32(r))
}

// BSTRToString converts b to a Go string.
func BSTRToString(b BSTR) string {
	n := BSTRLen(b)
	if n == 0 {
		return ""
	}
	return string(utf16.Decode(unsafe.Slice((*uint16)(b), n)))
}

// FreeBSTR frees b; nil is ignored.
func FreeBSTR(b BSTR) {
	if b == nil {
		return
	}
	procSysFreeString.Call(uintptr(unsafe.Pointer(b)))
}

// VARTYPE is a VARIANT's type tag.
type VARTYPE uint16

// VARIANT type tags.
const (
	VT_EMPTY    VARTYPE = 0
	VT_NULL     VARTYPE = 1
	VT_I2       VARTYPE = 2
	VT_I4       VARTYPE = 3
	VT_R4       VARTYPE = 4
	VT_R8       VARTYPE = 5
	VT_BSTR     VARTYPE = 8
	VT_DISPATCH VARTYPE = 9
	VT_ERROR    VARTYPE = 10
	VT_BOOL     VARTYPE = 11
	VT_VARIANT  VARTYPE = 12
	VT_UNKNOWN  VARTYPE = 13
	VT_I1       VARTYPE = 16
	VT_UI1      VARTYPE = 17
	VT_UI2      VARTYPE = 18
	VT_UI4      VARTYPE = 19
	VT_I8       VARTYPE = 20
	VT_UI8      VARTYPE = 21
	VT_INT      VARTYPE = 22
	VT_UINT     VARTYPE = 23
	VT_ARRAY    VARTYPE = 0x2000
	VT_BYREF    VARTYPE = 0x4000
)

// VARIANT_BOOL values.
const (
	VARIANT_TRUE  int16 = -1
	VARIANT_FALSE int16 = 0
)

// VARIANT is the OLE automation variant; Val is its 16-byte union.
type VARIANT struct {
	VT         VARTYPE
	wReserved1 uint16
	wReserved2 uint16
	wReserved3 uint16
	Val        [2]uintptr
}

// VariantInit sets v to VT_EMPTY.
func VariantInit(v *VARIANT) {
	procVariantInit.Call(uintptr(unsafe.Pointer(v)))
}

// VariantClear frees what v owns (BSTR, interface, array) and sets it to VT_EMPTY.
func VariantClear(v *VARIANT) error {
	r, _, _ := procVariantClear.Call(uintptr(unsafe.Pointer(v)))
	return HRESULTFromUintptr(r).Err()
}

func (v *VARIANT) union() unsafe.Pointer {
	return unsafe.Pointer(&v.Val[0])
}

// VariantEmpty returns a VT_EMPTY variant.
func VariantEmpty() VARIANT {
	return VARIANT{VT: VT_EMPTY}
}

// VariantI4 returns a VT_I4 variant.
func VariantI4(value int32) VARIANT {
	v := VARIANT{VT: VT_I4}
	*(*int32)(v.union()) = value
	return v
}

// VariantR8 returns a VT_R8 variant.
func VariantR8(value float64) VARIANT {
	v := VARIANT{VT: VT_R8}
	*(*uint64)(v.union()) = math.Float64bits(value)
	return v
}

// VariantBool returns a VT_BOOL variant.
func VariantBool(value bool) VARIANT {
	v := VARIANT{VT: VT_BOOL}
	if value {
		*(*int16)(v.union()) = VARIANT_TRUE
	}
	return v
}

// VariantBSTR returns a VT_BSTR variant owning a new BSTR of s.
func VariantBSTR(s string) (VARIANT, error) {
	b, err := NewBSTR(s)
	if err != nil {
		return VARIANT{}, err
	}
	v := VARIANT{VT: VT_BSTR}
	*(*BSTR)(v.union()) = b
	return v, nil
}

// VariantUnknown returns a VT_UNKNOWN variant owning a reference to u, which it AddRefs.
func VariantUnknown(u *IUnknown) VARIANT {
	v := VARIANT{VT: VT_UNKNOWN}
	if u != nil {
		u.AddRef()
	}
	*(**IUnknown)(v.union()) = u
	return v
}

// VariantArray returns a VT_ARRAY|elem variant owning array.
func VariantArray(elem VARTYPE, array *SAFEARRAY) VARIANT {
	v := VARIANT{VT: VT_ARRAY | elem}
	*(**SAFEARRAY)(v.union()) = array
	return v
}

// I4 returns the VT_I4 value.
func (v *VARIANT) I4() int32 {
	return *(*int32)(v.union())
}

// R8 returns the VT_R8 value.
func (v *VARIANT) R8() float64 {
	return math.Float64frombits(*(*uint64)(v.union()))
}

// Bool returns the VT_BOOL value.
func (v *VARIANT) Bool() bool {
	return *(*int16)(v.union()) != VARIANT_FALSE
}

// BSTR returns the VT_BSTR value, still owned by v.
func (v *VARIANT) BSTR() BSTR {
	return *(*BSTR)(v.union())
}

// Unknown returns the VT_UNKNOWN or VT_DISPATCH value, still owned by v.
func (v *VARIANT) Unknown() *IUnknown {
	return *(**IUnknown)(v.union())
}

// Array returns the VT_ARRAY value, still owned by v.
func (v *VARIANT) Array() *SAFEARRAY {
	return *(**SAFEARRAY)(v.union())
}

// SAFEARRAYBOUND is one dimension of a SAFEARRAY.
type SAFEARRAYBOUND struct {
	CElements uint32
	LLbound   int32
}

// SAFEARRAY is an OLE automation array.
type SAFEARRAY struct {
	CDims      uint16
	FFeatures  uint16
	CbElements uint32
	CLocks     uint32
	PvData     unsafe.Pointer
	Rgsabound  [1]SAFEARRAYBOUND
}

// SafeArrayCreateVector creates a one-dimensional array of n elements of type vt.
func SafeArrayCreateVector(vt VARTYPE, lbound int32, n uint32) (*SAFEARRAY, error) {
	r, _, _ := procSafeArrayCreateVector.Call(uintptr(vt), uintptr(lbound), uintptr(n))
	if r == 0 {
		return nil, E_OUTOFMEMORY
	}
	return (*SAFEARRAY)(PtrFromUintptr(r)), nil
}

// SafeArrayDestroy frees an array and what its elements own.
func SafeArrayDestroy(array *SAFEARRAY) error {
	r, _, _ := procSafeArrayDestroy.Call(uintptr(unsafe.Pointer(array)))
	return HRESULTFromUintptr(r).Err()
}

// SafeArrayPutElement copies the element at value (a BSTR, an interface pointer or a plain
// value, by the array's type) into a one-dimensional array at index.
func SafeArrayPutElement(array *SAFEARRAY, index int32, value unsafe.Pointer) error {
	r, _, _ := procSafeArrayPutElement.Call(uintptr(unsafe.Pointer(array)), uintptr(unsafe.Pointer(&index)), uintptr(value))
	return HRESULTFromUintptr(r).Err()
}

// SafeArrayGetElement copies the element at index of a one-dimensional array into out.
func SafeArrayGetElement(array *SAFEARRAY, index int32, out unsafe.Pointer) error {
	r, _, _ := procSafeArrayGetElement.Call(uintptr(unsafe.Pointer(array)), uintptr(unsafe.Pointer(&index)), uintptr(out))
	return HRESULTFromUintptr(r).Err()
}

// SafeArrayGetLBound returns the lower bound of dimension dim (1-based).
func SafeArrayGetLBound(array *SAFEARRAY, dim uint32) (int32, error) {
	var bound int32
	r, _, _ := procSafeArrayGetLBound.Call(uintptr(unsafe.Pointer(array)), uintptr(dim), uintptr(unsafe.Pointer(&bound)))
	return bound, HRESULTFromUintptr(r).Err()
}

// SafeArrayGetUBound returns the upper bound of dimension dim (1-based).
func SafeArrayGetUBound(array *SAFEARRAY, dim uint32) (int32, error) {
	var bound int32
	r, _, _ := procSafeArrayGetUBound.Call(uintptr(unsafe.Pointer(array)), uintptr(dim), uintptr(unsafe.Pointer(&bound)))
	return bound, HRESULTFromUintptr(r).Err()
}

// SafeArrayGetVartype returns the element type of an array.
func SafeArrayGetVartype(array *SAFEARRAY) (VARTYPE, error) {
	var vt VARTYPE
	r, _, _ := procSafeArrayGetVartype.Call(uintptr(unsafe.Pointer(array)), uintptr(unsafe.Pointer(&vt)))
	return vt, HRESULTFromUintptr(r).Err()
}

// SafeArrayFromInt32 creates a VT_I4 vector holding values.
func SafeArrayFromInt32(values []int32) (*SAFEARRAY, error) {
	array, err := SafeArrayCreateVector(VT_I4, 0, uint32(len(values)))
	if err != nil {
		return nil, err
	}
	for i := range values {
		if err := SafeArrayPutElement(array, int32(i), unsafe.Pointer(&values[i])); err != nil {
			SafeArrayDestroy(array)
			return nil, err
		}
	}
	return array, nil
}

// SafeArrayFromFloat64 creates a VT_R8 vector holding values.
func SafeArrayFromFloat64(values []float64) (*SAFEARRAY, error) {
	array, err := SafeArrayCreateVector(VT_R8, 0, uint32(len(values)))
	if err != nil {
		return nil, err
	}
	for i := range values {
		if err := SafeArrayPutElement(array, int32(i), unsafe.Pointer(&values[i])); err != nil {
			SafeArrayDestroy(array)
			return nil, err
		}
	}
	return array, nil
}

// SafeArrayFromUnknowns creates a VT_UNKNOWN vector holding values; the array AddRefs each.
func SafeArrayFromUnknowns(values []*IUnknown) (*SAFEARRAY, error) {
	array, err := SafeArrayCreateVector(VT_UNKNOWN, 0, uint32(len(values)))
	if err != nil {
		return nil, err
	}
	for i := range values {
		if err := SafeArrayPutElement(array, int32(i), unsafe.Pointer(values[i])); err != nil {
			SafeArrayDestroy(array)
			return nil, err
		}
	}
	return array, nil
}
