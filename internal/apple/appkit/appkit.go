//go:build darwin

// Package appkit binds the AppKit classes, enumerations and constants the macOS platform uses, as
// thin typed wrappers over internal/apple/objc on purego without cgo.
package appkit

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// AppKitPath is the install path of AppKit.framework.
const AppKitPath = "/System/Library/Frameworks/AppKit.framework/AppKit"

var (
	loadOnce sync.Once
	loadErr  error
	appkit   uintptr
)

// Load opens the Objective-C runtime and AppKit. It is safe to call repeatedly and from any
// thread.
func Load() error {
	loadOnce.Do(func() {
		if err := objc.Load(); err != nil {
			loadErr = err
			return
		}
		lib, err := purego.Dlopen(AppKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("appkit: open %s: %w", AppKitPath, err)
			return
		}
		appkit = lib
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// Handle returns the dlopen handle of AppKit.framework.
func Handle() uintptr {
	mustLoad()
	return appkit
}

// Constant returns the object stored in AppKit's data symbol name, panicking when it is missing.
func Constant(name string) objc.ID {
	id, err := objc.Constant(Handle(), name)
	if err != nil {
		panic(err)
	}
	return id
}

func class(name string) objc.Class {
	mustLoad()
	return objc.GetClass(name)
}

func send[T any](id objc.ID, sel string, args ...any) T {
	mustLoad()
	return objc.Send[T](id, objc.Sel(sel), args...)
}

func sendVoid(id objc.ID, sel string, args ...any) {
	mustLoad()
	objc.SendVoid(id, objc.Sel(sel), args...)
}

func sendClass[T any](name string, sel string, args ...any) T {
	return send[T](objc.ID(class(name)), sel, args...)
}

var nsBeep func()

// NSBeep plays the system beep.
func NSBeep() {
	mustLoad()
	if nsBeep == nil {
		purego.RegisterLibFunc(&nsBeep, appkit, "NSBeep")
	}
	nsBeep()
}
