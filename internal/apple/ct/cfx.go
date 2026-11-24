//go:build darwin

package ct

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/cf"
)

// NumberType is a CFNumberType.
type NumberType int

const (
	// NumberSInt32Type is kCFNumberSInt32Type.
	NumberSInt32Type NumberType = 3
	// NumberSInt64Type is kCFNumberSInt64Type.
	NumberSInt64Type NumberType = 4
	// NumberFloat64Type is kCFNumberFloat64Type.
	NumberFloat64Type NumberType = 6
)

// Number is a CFNumberRef.
type Number cf.TypeRef

// MutableArray is a CFMutableArrayRef.
type MutableArray cf.TypeRef

// MutableAttributedString is a CFMutableAttributedStringRef.
type MutableAttributedString cf.TypeRef

var (
	cfNumberCreate                  func(alloc uintptr, t NumberType, value unsafe.Pointer) Number
	cfNumberGetValue                func(n Number, t NumberType, value unsafe.Pointer) bool
	cfArrayCreateMutable            func(alloc uintptr, capacity cf.Index, callbacks uintptr) MutableArray
	cfArrayAppendValue              func(a MutableArray, value cf.TypeRef)
	cfAttributedStringCreateMutable func(alloc uintptr, maxLength cf.Index) MutableAttributedString
	cfAttributedStringReplaceString func(s MutableAttributedString, r cf.Range, replacement cf.String)
	cfAttributedStringGetLength     func(s MutableAttributedString) cf.Index
	cfAttributedStringSetAttribute  func(s MutableAttributedString, r cf.Range, name cf.TypeRef, value cf.TypeRef)
	cfLocaleCopyPreferredLanguages  func() cf.Array
	cfPreferencesCopyAppValue       func(key cf.String, app cf.TypeRef) cf.TypeRef
	cfURLCopyFileSystemPath         func(url cf.TypeRef, style int) cf.String
	cfURLGetTypeID                  func() cf.TypeID
	typeArrayCallBacksAddr          uintptr
	preferencesCurrentApplication   cf.TypeRef
)

// NumberCreateInt32 returns a new CFNumber holding v.
func NumberCreateInt32(v int32) *cf.Owned[Number] {
	mustLoad()
	return cf.Own(cfNumberCreate(0, NumberSInt32Type, unsafe.Pointer(&v)))
}

// NumberCreateInt64 returns a new CFNumber holding v.
func NumberCreateInt64(v int64) *cf.Owned[Number] {
	mustLoad()
	return cf.Own(cfNumberCreate(0, NumberSInt64Type, unsafe.Pointer(&v)))
}

// NumberCreateFloat64 returns a new CFNumber holding v.
func NumberCreateFloat64(v float64) *cf.Owned[Number] {
	mustLoad()
	return cf.Own(cfNumberCreate(0, NumberFloat64Type, unsafe.Pointer(&v)))
}

// AsNumber downcasts ref to a CFNumber.
func AsNumber(ref cf.TypeRef) (Number, bool) {
	if ref == 0 || cf.GetTypeID(ref) != cf.NumberGetTypeID() {
		return 0, false
	}
	return Number(ref), true
}

// Int64 returns the number as an int64; ok is false when the conversion is lossy.
func (n Number) Int64() (int64, bool) {
	mustLoad()
	var v int64
	ok := cfNumberGetValue(n, NumberSInt64Type, unsafe.Pointer(&v))
	return v, ok
}

// Float64 returns the number as a float64; ok is false when the conversion is lossy.
func (n Number) Float64() (float64, bool) {
	mustLoad()
	var v float64
	ok := cfNumberGetValue(n, NumberFloat64Type, unsafe.Pointer(&v))
	return v, ok
}

// ArrayCreateMutable returns a new empty CFMutableArray with the CFType callbacks.
func ArrayCreateMutable() *cf.Owned[MutableArray] {
	mustLoad()
	return cf.Own(cfArrayCreateMutable(0, 0, typeArrayCallBacksAddr))
}

// Append appends value, which the array retains.
func (a MutableArray) Append(value cf.TypeRef) {
	mustLoad()
	cfArrayAppendValue(a, value)
}

// Array returns the array as an immutable CFArray.
func (a MutableArray) Array() cf.Array {
	return cf.Array(a)
}

// AttributedStringCreateMutable returns a new empty CFMutableAttributedString.
func AttributedStringCreateMutable() *cf.Owned[MutableAttributedString] {
	mustLoad()
	return cf.Own(cfAttributedStringCreateMutable(0, 0))
}

// ReplaceString replaces the characters in r with replacement.
func (s MutableAttributedString) ReplaceString(r cf.Range, replacement cf.String) {
	mustLoad()
	cfAttributedStringReplaceString(s, r, replacement)
}

// Len returns the string's length in UTF-16 code units.
func (s MutableAttributedString) Len() cf.Index {
	mustLoad()
	return cfAttributedStringGetLength(s)
}

// SetAttribute sets the attribute name to value over r.
func (s MutableAttributedString) SetAttribute(r cf.Range, name cf.String, value cf.TypeRef) {
	mustLoad()
	cfAttributedStringSetAttribute(s, r, cf.TypeRef(name), value)
}

// LocaleCopyPreferredLanguages returns the user's preferred languages.
func LocaleCopyPreferredLanguages() *cf.Owned[cf.Array] {
	mustLoad()
	return cf.Own(cfLocaleCopyPreferredLanguages())
}

// PreferencesCopyAppValue returns the current application's preference for key, or nil.
func PreferencesCopyAppValue(key cf.String) *cf.Owned[cf.TypeRef] {
	mustLoad()
	return cf.Own(cfPreferencesCopyAppValue(key, preferencesCurrentApplication))
}

// URLFileSystemPath returns the POSIX path of a CFURL, or false when ref is not a file URL.
func URLFileSystemPath(ref cf.TypeRef) (string, bool) {
	mustLoad()
	if ref == 0 || cf.GetTypeID(ref) != cfURLGetTypeID() {
		return "", false
	}
	s := cf.Own(cfURLCopyFileSystemPath(ref, 0))
	if s == nil {
		return "", false
	}
	defer s.Release()
	return s.Ref().GoString(), true
}
