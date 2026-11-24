//go:build darwin && !ios

// Package macos is the macOS platform implementation for GPUI.
//
// macOS screens have a y axis that goes up from the bottom of the screen and an origin at the
// bottom left of the main display.
package macos

import (
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func nsStringToStr(str objc.ID) string {
	if str == 0 {
		return ""
	}
	cstr := objc.Send[*byte](str, objc.Sel("UTF8String"))
	if cstr == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(cstr), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(cstr, n))
}

// NSRange is a location and length in a text storage, as AppKit's NSTextInputClient uses it.
type NSRange struct {
	Location objc.NSUInteger
	Length   objc.NSUInteger
}

func nsRangeInvalid() NSRange {
	return NSRange{Location: objc.NSUInteger(objc.NSNotFound), Length: 0}
}

func (r NSRange) isValid() bool {
	return r.Location != objc.NSUInteger(objc.NSNotFound)
}

func (r NSRange) toRange() (gpui.Range, bool) {
	if r.isValid() {
		start := int(r.Location)
		end := start + int(r.Length)
		return gpui.Range{Start: start, End: end}, true
	}
	return gpui.Range{}, false
}

func nsRangeFromRange(r gpui.Range) NSRange {
	return NSRange{Location: objc.NSUInteger(r.Start), Length: objc.NSUInteger(max(r.End-r.Start, 0))}
}

func (r NSRange) toObjc() objc.NSRange {
	return objc.MakeNSRange(r.Location, r.Length)
}

func nsString(s string) objc.ID {
	return objc.Autorelease(objc.NSStringFromGo(s))
}
