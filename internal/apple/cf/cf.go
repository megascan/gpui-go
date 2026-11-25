//go:build darwin

// Package cf binds the CoreFoundation types the Apple platform packages use, with explicit Release
// for owned references (D-RAII), on purego without cgo.
package cf

import (
	"fmt"
	"structs"
	"sync"
	"sync/atomic"
	"unicode/utf8"
	"unsafe"

	"github.com/ebitengine/purego"
)

// CoreFoundationPath is the install path of CoreFoundation.framework.
const CoreFoundationPath = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"

// TypeRef is an untyped CoreFoundation object (CFTypeRef).
type TypeRef uintptr

// TypeID identifies a CoreFoundation type (CFTypeID).
type TypeID uint

// Index is CoreFoundation's signed size type (CFIndex).
type Index = int

// StringEncoding is a CFStringEncoding.
type StringEncoding uint32

// StringEncodingUTF8 is kCFStringEncodingUTF8.
const StringEncodingUTF8 StringEncoding = 0x08000100

// Range is a CFRange.
type Range struct {
	_        structs.HostLayout
	Location Index
	Length   Index
}

var (
	loadOnce sync.Once
	loadErr  error
	lib      uintptr

	cfRetain                         func(ref TypeRef) TypeRef
	cfRelease                        func(ref TypeRef)
	cfGetTypeID                      func(ref TypeRef) TypeID
	cfGetRetainCount                 func(ref TypeRef) Index
	cfEqual                          func(a, b TypeRef) bool
	cfStringGetTypeID                func() TypeID
	cfDataGetTypeID                  func() TypeID
	cfDictionaryGetTypeID            func() TypeID
	cfBooleanGetTypeID               func() TypeID
	cfArrayGetTypeID                 func() TypeID
	cfNumberGetTypeID                func() TypeID
	cfStringCreateWithBytes          func(alloc uintptr, bytes unsafe.Pointer, n Index, encoding StringEncoding, external bool) TypeRef
	cfStringGetLength                func(s TypeRef) Index
	cfStringGetBytes                 func(s TypeRef, r Range, encoding StringEncoding, lossByte uint8, external bool, buf unsafe.Pointer, maxLen Index, used *Index) Index
	cfDataCreate                     func(alloc uintptr, bytes unsafe.Pointer, n Index) TypeRef
	cfDataGetLength                  func(d TypeRef) Index
	cfDataGetBytes                   func(d TypeRef, r Range, buf unsafe.Pointer)
	cfDictionaryCreate               func(alloc uintptr, keys, values unsafe.Pointer, n Index, keyCallbacks, valueCallbacks uintptr) TypeRef
	cfDictionaryGetValueIfPresent    func(d TypeRef, key TypeRef, value *TypeRef) bool
	cfDictionaryGetCount             func(d TypeRef) Index
	cfArrayCreate                    func(alloc uintptr, values unsafe.Pointer, n Index, callbacks uintptr) TypeRef
	cfArrayGetCount                  func(a TypeRef) Index
	cfArrayGetValueAtIndex           func(a TypeRef, i Index) TypeRef
	cfBooleanGetValue                func(b TypeRef) bool
	typeDictionaryKeyCallBacksAddr   uintptr
	typeDictionaryValueCallBacksAddr uintptr
	typeArrayCallBacksAddr           uintptr
	booleanTrue                      TypeRef
	booleanFalse                     TypeRef
)

func ptrAt(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

// LoadPointer reads the pointer-sized value stored at address p.
func LoadPointer(p uintptr) uintptr {
	return *(*uintptr)(ptrAt(p))
}

// Load opens CoreFoundation and resolves its symbols. It is safe to call repeatedly.
func Load() error {
	loadOnce.Do(func() {
		h, err := purego.Dlopen(CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("cf: open %s: %w", CoreFoundationPath, err)
			return
		}
		lib = h
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&cfRetain, "CFRetain"},
			{&cfRelease, "CFRelease"},
			{&cfGetTypeID, "CFGetTypeID"},
			{&cfGetRetainCount, "CFGetRetainCount"},
			{&cfEqual, "CFEqual"},
			{&cfStringGetTypeID, "CFStringGetTypeID"},
			{&cfDataGetTypeID, "CFDataGetTypeID"},
			{&cfDictionaryGetTypeID, "CFDictionaryGetTypeID"},
			{&cfBooleanGetTypeID, "CFBooleanGetTypeID"},
			{&cfArrayGetTypeID, "CFArrayGetTypeID"},
			{&cfNumberGetTypeID, "CFNumberGetTypeID"},
			{&cfStringCreateWithBytes, "CFStringCreateWithBytes"},
			{&cfStringGetLength, "CFStringGetLength"},
			{&cfStringGetBytes, "CFStringGetBytes"},
			{&cfDataCreate, "CFDataCreate"},
			{&cfDataGetLength, "CFDataGetLength"},
			{&cfDataGetBytes, "CFDataGetBytes"},
			{&cfDictionaryCreate, "CFDictionaryCreate"},
			{&cfDictionaryGetValueIfPresent, "CFDictionaryGetValueIfPresent"},
			{&cfDictionaryGetCount, "CFDictionaryGetCount"},
			{&cfArrayCreate, "CFArrayCreate"},
			{&cfArrayGetCount, "CFArrayGetCount"},
			{&cfArrayGetValueAtIndex, "CFArrayGetValueAtIndex"},
			{&cfBooleanGetValue, "CFBooleanGetValue"},
		} {
			addr, err := purego.Dlsym(h, b.name)
			if err != nil {
				loadErr = fmt.Errorf("cf: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
		for _, s := range []struct {
			dst  *uintptr
			name string
		}{
			{&typeDictionaryKeyCallBacksAddr, "kCFTypeDictionaryKeyCallBacks"},
			{&typeDictionaryValueCallBacksAddr, "kCFTypeDictionaryValueCallBacks"},
			{&typeArrayCallBacksAddr, "kCFTypeArrayCallBacks"},
		} {
			addr, err := purego.Dlsym(h, s.name)
			if err != nil {
				loadErr = fmt.Errorf("cf: %w", err)
				return
			}
			*s.dst = addr
		}
		for _, s := range []struct {
			dst  *TypeRef
			name string
		}{
			{&booleanTrue, "kCFBooleanTrue"},
			{&booleanFalse, "kCFBooleanFalse"},
		} {
			ref, err := Constant(h, s.name)
			if err != nil {
				loadErr = err
				return
			}
			*s.dst = ref
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// Constant returns the CoreFoundation object stored in the exported data symbol name of the
// library handle h, such as kSecClass.
func Constant(h uintptr, name string) (TypeRef, error) {
	addr, err := purego.Dlsym(h, name)
	if err != nil {
		return 0, fmt.Errorf("cf: %w", err)
	}
	return TypeRef(LoadPointer(addr)), nil
}

// Retain increments ref's retain count and returns it.
func Retain(ref TypeRef) TypeRef {
	mustLoad()
	if ref == 0 {
		return 0
	}
	return cfRetain(ref)
}

// Release decrements ref's retain count. A zero ref is ignored.
func Release(ref TypeRef) {
	mustLoad()
	if ref != 0 {
		cfRelease(ref)
	}
}

// GetTypeID returns ref's type.
func GetTypeID(ref TypeRef) TypeID {
	mustLoad()
	return cfGetTypeID(ref)
}

// GetRetainCount returns ref's retain count.
func GetRetainCount(ref TypeRef) Index {
	mustLoad()
	return cfGetRetainCount(ref)
}

// Equal reports whether a and b are equal by CFEqual.
func Equal(a, b TypeRef) bool {
	mustLoad()
	return cfEqual(a, b)
}

// Ref is the constraint satisfied by every typed CoreFoundation reference.
type Ref interface {
	~uintptr
}

// Owned owns one reference to a CoreFoundation object (CFRetained). Release is its drop.
type Owned[T Ref] struct {
	ref      T
	released atomic.Bool
}

// Own takes ownership of a reference obtained under the Create rule; it returns nil for a zero ref.
func Own[T Ref](ref T) *Owned[T] {
	if ref == 0 {
		return nil
	}
	return &Owned[T]{ref: ref}
}

// RetainRef retains a reference obtained under the Get rule and owns the new one.
func RetainRef[T Ref](ref T) *Owned[T] {
	if ref == 0 {
		return nil
	}
	return &Owned[T]{ref: T(Retain(TypeRef(ref)))}
}

// Ref returns the reference; it panics after Release.
func (o *Owned[T]) Ref() T {
	if o.released.Load() {
		panic("cf: Owned used after Release")
	}
	return o.ref
}

// TypeRef returns the reference as an untyped CFTypeRef; it panics after Release.
func (o *Owned[T]) TypeRef() TypeRef {
	return TypeRef(o.Ref())
}

// Release drops the owned reference. It is idempotent.
func (o *Owned[T]) Release() {
	if o == nil || o.released.Swap(true) {
		return
	}
	Release(TypeRef(o.ref))
}

// String is a CFStringRef.
type String TypeRef

// Data is a CFDataRef.
type Data TypeRef

// Dictionary is a CFDictionaryRef.
type Dictionary TypeRef

// Array is a CFArrayRef.
type Array TypeRef

// Boolean is a CFBooleanRef.
type Boolean TypeRef

// StringGetTypeID returns CFString's type.
func StringGetTypeID() TypeID {
	mustLoad()
	return cfStringGetTypeID()
}

// DataGetTypeID returns CFData's type.
func DataGetTypeID() TypeID {
	mustLoad()
	return cfDataGetTypeID()
}

// DictionaryGetTypeID returns CFDictionary's type.
func DictionaryGetTypeID() TypeID {
	mustLoad()
	return cfDictionaryGetTypeID()
}

// ArrayGetTypeID returns CFArray's type.
func ArrayGetTypeID() TypeID {
	mustLoad()
	return cfArrayGetTypeID()
}

// BooleanGetTypeID returns CFBoolean's type.
func BooleanGetTypeID() TypeID {
	mustLoad()
	return cfBooleanGetTypeID()
}

// NumberGetTypeID returns CFNumber's type.
func NumberGetTypeID() TypeID {
	mustLoad()
	return cfNumberGetTypeID()
}

// AsString downcasts ref to a CFString.
func AsString(ref TypeRef) (String, bool) {
	if ref == 0 || GetTypeID(ref) != StringGetTypeID() {
		return 0, false
	}
	return String(ref), true
}

// AsData downcasts ref to a CFData.
func AsData(ref TypeRef) (Data, bool) {
	if ref == 0 || GetTypeID(ref) != DataGetTypeID() {
		return 0, false
	}
	return Data(ref), true
}

// AsDictionary downcasts ref to a CFDictionary.
func AsDictionary(ref TypeRef) (Dictionary, bool) {
	if ref == 0 || GetTypeID(ref) != DictionaryGetTypeID() {
		return 0, false
	}
	return Dictionary(ref), true
}

// AsArray downcasts ref to a CFArray.
func AsArray(ref TypeRef) (Array, bool) {
	if ref == 0 || GetTypeID(ref) != ArrayGetTypeID() {
		return 0, false
	}
	return Array(ref), true
}

// StringCreate returns a new CFString holding s, or nil when s is not valid UTF-8 or
// CFStringCreateWithBytes fails.
func StringCreate(s string) *Owned[String] {
	mustLoad()
	if !utf8.ValidString(s) {
		return nil
	}
	b := []byte(s)
	var p unsafe.Pointer
	if len(b) > 0 {
		p = unsafe.Pointer(&b[0])
	}
	return Own(String(cfStringCreateWithBytes(0, p, len(b), StringEncodingUTF8, false)))
}

// GoString returns the string's UTF-8 contents.
func (s String) GoString() string {
	mustLoad()
	if s == 0 {
		return ""
	}
	n := cfStringGetLength(TypeRef(s))
	if n == 0 {
		return ""
	}
	r := Range{Location: 0, Length: n}
	var used Index
	cfStringGetBytes(TypeRef(s), r, StringEncodingUTF8, 0, false, nil, 0, &used)
	if used == 0 {
		return ""
	}
	buf := make([]byte, used)
	cfStringGetBytes(TypeRef(s), r, StringEncodingUTF8, 0, false, unsafe.Pointer(&buf[0]), used, &used)
	return string(buf[:used])
}

// DataCreate returns a new CFData holding a copy of b, or nil when CFDataCreate fails.
func DataCreate(b []byte) *Owned[Data] {
	mustLoad()
	var p unsafe.Pointer
	if len(b) > 0 {
		p = unsafe.Pointer(&b[0])
	}
	return Own(Data(cfDataCreate(0, p, len(b))))
}

// Bytes returns a copy of the data's bytes.
func (d Data) Bytes() []byte {
	mustLoad()
	if d == 0 {
		return nil
	}
	n := cfDataGetLength(TypeRef(d))
	out := make([]byte, n)
	if n > 0 {
		cfDataGetBytes(TypeRef(d), Range{Location: 0, Length: n}, unsafe.Pointer(&out[0]))
	}
	return out
}

// DictionaryCreate returns a new CFDictionary mapping keys[i] to values[i] with the CFType
// callbacks, so the dictionary retains its keys and values. It panics on a NULL key or value and
// returns nil when CFDictionaryCreate fails.
func DictionaryCreate(keys, values []TypeRef) *Owned[Dictionary] {
	mustLoad()
	if len(keys) != len(values) {
		panic("cf: DictionaryCreate needs as many keys as values")
	}
	for i := range keys {
		if keys[i] == 0 || values[i] == 0 {
			panic("cf: DictionaryCreate given a NULL key or value")
		}
	}
	var kp, vp unsafe.Pointer
	if len(keys) > 0 {
		kp = unsafe.Pointer(&keys[0])
		vp = unsafe.Pointer(&values[0])
	}
	return Own(Dictionary(cfDictionaryCreate(0, kp, vp, len(keys), typeDictionaryKeyCallBacksAddr, typeDictionaryValueCallBacksAddr)))
}

// Get returns the value for key, borrowed from the dictionary.
func (d Dictionary) Get(key TypeRef) (TypeRef, bool) {
	mustLoad()
	if d == 0 {
		return 0, false
	}
	var v TypeRef
	if !cfDictionaryGetValueIfPresent(TypeRef(d), key, &v) {
		return 0, false
	}
	return v, true
}

// Count returns the number of entries.
func (d Dictionary) Count() Index {
	mustLoad()
	return cfDictionaryGetCount(TypeRef(d))
}

// ArrayCreate returns a new CFArray of values with the CFType callbacks.
func ArrayCreate(values []TypeRef) *Owned[Array] {
	mustLoad()
	var p unsafe.Pointer
	if len(values) > 0 {
		p = unsafe.Pointer(&values[0])
	}
	return Own(Array(cfArrayCreate(0, p, len(values), typeArrayCallBacksAddr)))
}

// Count returns the number of elements.
func (a Array) Count() Index {
	mustLoad()
	return cfArrayGetCount(TypeRef(a))
}

// At returns element i, borrowed from the array.
func (a Array) At(i Index) TypeRef {
	mustLoad()
	return cfArrayGetValueAtIndex(TypeRef(a), i)
}

// BooleanTrue returns kCFBooleanTrue.
func BooleanTrue() Boolean {
	mustLoad()
	return Boolean(booleanTrue)
}

// BooleanFalse returns kCFBooleanFalse.
func BooleanFalse() Boolean {
	mustLoad()
	return Boolean(booleanFalse)
}

// Value returns the boolean's value.
func (b Boolean) Value() bool {
	mustLoad()
	return cfBooleanGetValue(TypeRef(b))
}
