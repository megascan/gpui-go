//go:build darwin

package objc

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// NSUTF8StringEncoding is Foundation's UTF-8 string encoding.
const NSUTF8StringEncoding NSUInteger = 4

// NSActivityOptions are the options of -[NSProcessInfo beginActivityWithOptions:reason:].
type NSActivityOptions uint64

const (
	// NSActivityIdleDisplaySleepDisabled keeps the display awake.
	NSActivityIdleDisplaySleepDisabled NSActivityOptions = 1 << 40
	// NSActivityIdleSystemSleepDisabled keeps the system awake.
	NSActivityIdleSystemSleepDisabled NSActivityOptions = 1 << 20
	// NSActivitySuddenTerminationDisabled disables sudden termination.
	NSActivitySuddenTerminationDisabled NSActivityOptions = 1 << 14
	// NSActivityAutomaticTerminationDisabled disables automatic termination.
	NSActivityAutomaticTerminationDisabled NSActivityOptions = 1 << 15
	// NSActivityAnimationTrackingEnabled tracks animation timing.
	NSActivityAnimationTrackingEnabled NSActivityOptions = 1 << 45
	// NSActivityTrackingEnabled tracks activity timing.
	NSActivityTrackingEnabled NSActivityOptions = 1 << 46
	// NSActivityUserInitiated is work the user asked for.
	NSActivityUserInitiated NSActivityOptions = 0x00FFFFFF | NSActivityIdleSystemSleepDisabled
	// NSActivityUserInitiatedAllowingIdleSystemSleep is user-initiated work that lets the system sleep.
	NSActivityUserInitiatedAllowingIdleSystemSleep NSActivityOptions = NSActivityUserInitiated &^ NSActivityIdleSystemSleepDisabled
	// NSActivityBackground is discretionary work.
	NSActivityBackground NSActivityOptions = 0x000000FF
	// NSActivityLatencyCritical is work that needs the lowest latency.
	NSActivityLatencyCritical NSActivityOptions = 0xFF00000000
	// NSActivityUserInteractive is latency-critical user-initiated work.
	NSActivityUserInteractive NSActivityOptions = NSActivityUserInitiated | NSActivityLatencyCritical
)

func ptrAt(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

// LoadPointer reads the pointer-sized value stored at address p.
func LoadPointer(p uintptr) uintptr {
	return *(*uintptr)(ptrAt(p))
}

// Constant returns the object stored in the exported data symbol name of the library handle lib,
// such as NSRunLoopCommonModes or a notification name.
func Constant(lib uintptr, name string) (ID, error) {
	addr, err := purego.Dlsym(lib, name)
	if err != nil {
		return Nil, fmt.Errorf("objc: %w", err)
	}
	return ID(LoadPointer(addr)), nil
}

// FoundationConstant returns the object stored in Foundation's data symbol name, panicking when it
// is missing.
func FoundationConstant(name string) ID {
	id, err := Constant(Foundation(), name)
	if err != nil {
		panic(err)
	}
	return id
}

// NSStringFromGo returns a +1 NSString holding s.
func NSStringFromGo(s string) ID {
	b := []byte(s)
	var p unsafe.Pointer
	if len(b) > 0 {
		p = unsafe.Pointer(&b[0])
	}
	str := Send[ID](Alloc(GetClass("NSString")), Sel("initWithBytes:length:encoding:"), p, NSUInteger(len(b)), NSUTF8StringEncoding)
	return str
}

// GoString returns the UTF-8 contents of an NSString, including any U+0000, or "" for nil.
func GoString(str ID) string {
	if str == Nil {
		return ""
	}
	var out string
	AutoreleasePool(func() {
		data := Send[ID](str, Sel("dataUsingEncoding:"), NSUTF8StringEncoding)
		if data == Nil {
			return
		}
		n := Send[NSUInteger](data, Sel("length"))
		if n == 0 {
			return
		}
		p := Send[unsafe.Pointer](data, Sel("bytes"))
		if p == nil {
			return
		}
		out = string(unsafe.Slice((*byte)(p), n))
	})
	return out
}

// NSArrayFromIDs returns a +1 NSArray of objects.
func NSArrayFromIDs(objects []ID) ID {
	var p unsafe.Pointer
	if len(objects) > 0 {
		p = unsafe.Pointer(&objects[0])
	}
	return Send[ID](Alloc(GetClass("NSArray")), Sel("initWithObjects:count:"), p, NSUInteger(len(objects)))
}

// NSArrayToIDs returns the elements of an NSArray, borrowed from it.
func NSArrayToIDs(array ID) []ID {
	if array == Nil {
		return nil
	}
	n := Send[NSUInteger](array, Sel("count"))
	out := make([]ID, 0, n)
	at := Sel("objectAtIndex:")
	for i := NSUInteger(0); i < n; i++ {
		out = append(out, Send[ID](array, at, i))
	}
	return out
}

// NSDictionaryFromIDs returns a +1 NSDictionary mapping keys[i] to values[i].
func NSDictionaryFromIDs(keys, values []ID) ID {
	if len(keys) != len(values) {
		panic("objc: NSDictionaryFromIDs needs as many keys as values")
	}
	var kp, vp unsafe.Pointer
	if len(keys) > 0 {
		kp = unsafe.Pointer(&keys[0])
		vp = unsafe.Pointer(&values[0])
	}
	return Send[ID](Alloc(GetClass("NSDictionary")), Sel("initWithObjects:forKeys:count:"), vp, kp, NSUInteger(len(keys)))
}

// NSDictionaryObjectForKey returns the value for key, borrowed from the dictionary, or nil.
func NSDictionaryObjectForKey(dictionary, key ID) ID {
	if dictionary == Nil {
		return Nil
	}
	return Send[ID](dictionary, Sel("objectForKey:"), key)
}

// NSDataFromBytes returns a +1 NSData holding a copy of b.
func NSDataFromBytes(b []byte) ID {
	var p unsafe.Pointer
	if len(b) > 0 {
		p = unsafe.Pointer(&b[0])
	}
	return Send[ID](Alloc(GetClass("NSData")), Sel("initWithBytes:length:"), p, NSUInteger(len(b)))
}

// NSDataToBytes returns a copy of an NSData's bytes.
func NSDataToBytes(data ID) []byte {
	if data == Nil {
		return nil
	}
	n := Send[NSUInteger](data, Sel("length"))
	out := make([]byte, n)
	if n > 0 {
		SendVoid(data, Sel("getBytes:length:"), unsafe.Pointer(&out[0]), n)
	}
	return out
}

// NSURLFromFilePath returns a +1 file NSURL for path.
func NSURLFromFilePath(path string) ID {
	s := NSStringFromGo(path)
	defer Release(s)
	return Send[ID](Alloc(GetClass("NSURL")), Sel("initFileURLWithPath:"), s)
}

// NSURLFromString returns a +1 NSURL parsed from s, or nil when s is not a valid URL.
func NSURLFromString(s string) ID {
	str := NSStringFromGo(s)
	defer Release(str)
	return Send[ID](Alloc(GetClass("NSURL")), Sel("initWithString:"), str)
}

// NSURLAbsoluteString returns the URL as a string.
func NSURLAbsoluteString(url ID) string {
	if url == Nil {
		return ""
	}
	return GoString(Send[ID](url, Sel("absoluteString")))
}

// NSURLPath returns the path component of the URL.
func NSURLPath(url ID) string {
	if url == Nil {
		return ""
	}
	return GoString(Send[ID](url, Sel("path")))
}

// NSError is a Go copy of an NSError.
type NSError struct {
	Domain               string
	Code                 NSInteger
	LocalizedDescription string
}

// Error returns the localized description with the domain and code.
func (e *NSError) Error() string {
	return fmt.Sprintf("%s (%s %d)", e.LocalizedDescription, e.Domain, e.Code)
}

// NSErrorFromID copies an NSError, returning nil for a nil object.
func NSErrorFromID(err ID) *NSError {
	if err == Nil {
		return nil
	}
	return &NSError{
		Domain:               GoString(Send[ID](err, Sel("domain"))),
		Code:                 Send[NSInteger](err, Sel("code")),
		LocalizedDescription: GoString(Send[ID](err, Sel("localizedDescription"))),
	}
}

// MainThreadMarker proves the holder runs on the main thread (objc2's MainThreadMarker).
type MainThreadMarker struct{}

// IsMainThread reports whether the calling OS thread is the main thread.
func IsMainThread() bool {
	return SendClass[bool](GetClass("NSThread"), Sel("isMainThread"))
}

// NewMainThreadMarker returns a marker when called on the main thread.
func NewMainThreadMarker() (MainThreadMarker, bool) {
	if IsMainThread() {
		return MainThreadMarker{}, true
	}
	return MainThreadMarker{}, false
}
