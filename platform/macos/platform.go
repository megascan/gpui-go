//go:build darwin && !ios

package macos

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const carbonPath = "/System/Library/Frameworks/Carbon.framework/Carbon"

var (
	carbonOnce sync.Once
	carbonErr  error

	tisCopyCurrentKeyboardLayoutInputSource func() objc.ID
	tisCopyCurrentKeyboardInputSource       func() objc.ID
	tisGetInputSourceProperty               func(inputSource objc.ID, propertyKey cf.TypeRef) objc.ID
	ucKeyTranslate                          func(keyLayoutPtr unsafe.Pointer, virtualKeyCode uint16, keyAction uint16, modifierKeyState uint32, keyboardType uint32, keyTranslateOptions uint32, deadKeyState *uint32, maxStringLength uintptr, actualStringLength *uintptr, unicodeString *uint16) uint32
	lmGetKbdType                            func() uint16
	cfDataGetBytePtr                        func(data cf.TypeRef) unsafe.Pointer

	kTISPropertyUnicodeKeyLayoutData      cf.TypeRef
	kTISPropertyInputSourceID             cf.TypeRef
	kTISPropertyLocalizedName             cf.TypeRef
	kTISPropertyInputSourceIsASCIICapable cf.TypeRef
	kTISPropertyInputSourceType           cf.TypeRef
	kTISTypeKeyboardInputMode             cf.TypeRef
)

func loadCarbon() error {
	carbonOnce.Do(func() {
		if err := objc.Load(); err != nil {
			carbonErr = err
			return
		}
		lib, err := purego.Dlopen(carbonPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			carbonErr = fmt.Errorf("macos: open %s: %w", carbonPath, err)
			return
		}
		purego.RegisterLibFunc(&tisCopyCurrentKeyboardLayoutInputSource, lib, "TISCopyCurrentKeyboardLayoutInputSource")
		purego.RegisterLibFunc(&tisCopyCurrentKeyboardInputSource, lib, "TISCopyCurrentKeyboardInputSource")
		purego.RegisterLibFunc(&tisGetInputSourceProperty, lib, "TISGetInputSourceProperty")
		purego.RegisterLibFunc(&ucKeyTranslate, lib, "UCKeyTranslate")
		purego.RegisterLibFunc(&lmGetKbdType, lib, "LMGetKbdType")
		cfLib, err := purego.Dlopen(cf.CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			carbonErr = fmt.Errorf("macos: open %s: %w", cf.CoreFoundationPath, err)
			return
		}
		purego.RegisterLibFunc(&cfDataGetBytePtr, cfLib, "CFDataGetBytePtr")
		for _, c := range []struct {
			dst  *cf.TypeRef
			name string
		}{
			{&kTISPropertyUnicodeKeyLayoutData, "kTISPropertyUnicodeKeyLayoutData"},
			{&kTISPropertyInputSourceID, "kTISPropertyInputSourceID"},
			{&kTISPropertyLocalizedName, "kTISPropertyLocalizedName"},
			{&kTISPropertyInputSourceIsASCIICapable, "kTISPropertyInputSourceIsASCIICapable"},
			{&kTISPropertyInputSourceType, "kTISPropertyInputSourceType"},
			{&kTISTypeKeyboardInputMode, "kTISTypeKeyboardInputMode"},
		} {
			ref, err := cf.Constant(lib, c.name)
			if err != nil {
				carbonErr = fmt.Errorf("macos: %w", err)
				return
			}
			*c.dst = ref
		}
	})
	return carbonErr
}

func mustLoadCarbon() {
	if err := loadCarbon(); err != nil {
		panic(err)
	}
}
