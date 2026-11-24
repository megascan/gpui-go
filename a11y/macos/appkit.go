//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const libobjcPath = "/usr/lib/libobjc.A.dylib"

var (
	objectSetClass   func(obj objc.ID, class objc.Class) objc.Class
	classGetName     func(class objc.Class) string
	objcInitWeak     func(location *objc.ID, value objc.ID) objc.ID
	objcLoadWeakRet  func(location *objc.ID) objc.ID
	objcDestroyWeak  func(location *objc.ID)
	postNotification func(element objc.ID, notification objc.ID)
	postNotifUser    func(element objc.ID, notification objc.ID, userInfo objc.ID)
)

var loadRuntime = sync.OnceValue(func() error {
	if err := appkit.Load(); err != nil {
		return err
	}
	lib, err := purego.Dlopen(libobjcPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&objectSetClass, lib, "object_setClass")
	purego.RegisterLibFunc(&classGetName, lib, "class_getName")
	purego.RegisterLibFunc(&objcInitWeak, lib, "objc_initWeak")
	purego.RegisterLibFunc(&objcLoadWeakRet, lib, "objc_loadWeakRetained")
	purego.RegisterLibFunc(&objcDestroyWeak, lib, "objc_destroyWeak")
	purego.RegisterLibFunc(&postNotification, appkit.Handle(), "NSAccessibilityPostNotification")
	purego.RegisterLibFunc(&postNotifUser, appkit.Handle(), "NSAccessibilityPostNotificationWithUserInfo")
	return nil
})

func mustLoadRuntime() {
	if err := loadRuntime(); err != nil {
		panic(err)
	}
}

var (
	constMu    sync.Mutex
	constCache = map[string]objc.ID{}
	litCache   = map[string]objc.ID{}
)

func axConst(name string) objc.ID {
	mustLoadRuntime()
	constMu.Lock()
	defer constMu.Unlock()
	if id, ok := constCache[name]; ok {
		return id
	}
	id := appkit.Constant(name)
	constCache[name] = id
	return id
}

func nsString(literal string) objc.ID {
	constMu.Lock()
	defer constMu.Unlock()
	if id, ok := litCache[literal]; ok {
		return id
	}
	id := objc.NSStringFromGo(literal)
	litCache[literal] = id
	return id
}

func nsStringAutoreleased(s string) objc.ID {
	return objc.Autorelease(objc.NSStringFromGo(s))
}

func nsStringEqual(a, b objc.ID) bool {
	if a == objc.Nil || b == objc.Nil {
		return a == b
	}
	return objc.Send[bool](a, objc.Sel("isEqual:"), b)
}

func accessibilityPostNotification(element, notification objc.ID) {
	mustLoadRuntime()
	postNotification(element, notification)
}

func accessibilityPostNotificationWithUserInfo(element, notification, userInfo objc.ID) {
	mustLoadRuntime()
	postNotifUser(element, notification, userInfo)
}

type weakID struct {
	mu        sync.Mutex
	slot      *objc.ID
	destroyed bool
}

func newWeakID(id objc.ID) *weakID {
	mustLoadRuntime()
	w := &weakID{slot: new(objc.ID)}
	objcInitWeak(w.slot, id)
	return w
}

func (w *weakID) load() objc.ID {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.destroyed {
		return objc.Nil
	}
	return objcLoadWeakRet(w.slot)
}

func (w *weakID) destroy() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.destroyed {
		return
	}
	w.destroyed = true
	objcDestroyWeak(w.slot)
}
