//go:build darwin

// Package objc binds the Objective-C runtime and the Foundation types the Apple platform packages
// use, on purego without cgo.
package objc

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/purego"
	pobjc "github.com/ebitengine/purego/objc"
)

// ID is an Objective-C object pointer.
type ID = pobjc.ID

// SEL is a registered method selector.
type SEL = pobjc.SEL

// Class is an Objective-C class object.
type Class = pobjc.Class

// Protocol is an Objective-C protocol.
type Protocol = pobjc.Protocol

// IMP is a method implementation pointer.
type IMP = pobjc.IMP

// Ivar is an instance variable of a class.
type Ivar = pobjc.Ivar

// MethodDef pairs a selector with the Go function implementing it; the function takes (ID, SEL) first.
type MethodDef = pobjc.MethodDef

// FieldDef describes an instance variable added by RegisterClass.
type FieldDef = pobjc.FieldDef

// Nil is the nil object.
const Nil ID = 0

// FoundationPath is the install path of Foundation.framework.
const FoundationPath = "/System/Library/Frameworks/Foundation.framework/Foundation"

const libobjcPath = "/usr/lib/libobjc.A.dylib"

const maxRegisterStructSize = 16

var (
	loadOnce sync.Once
	loadErr  error

	libobjc    uintptr
	foundation uintptr

	msgSendAddr      uintptr
	msgSendStretAddr uintptr

	objc_retain              func(id ID) ID
	objc_release             func(id ID)
	objc_autorelease         func(id ID) ID
	objc_autoreleasePoolPush func() uintptr
	objc_autoreleasePoolPop  func(pool uintptr)
)

// Load opens libobjc and Foundation and resolves the runtime entry points. It is safe to call
// repeatedly and from any thread.
func Load() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen(libobjcPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("objc: open %s: %w", libobjcPath, err)
			return
		}
		libobjc = lib
		fw, err := purego.Dlopen(FoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("objc: open %s: %w", FoundationPath, err)
			return
		}
		foundation = fw
		if msgSendAddr, err = purego.Dlsym(lib, "objc_msgSend"); err != nil {
			loadErr = fmt.Errorf("objc: %w", err)
			return
		}
		if runtime.GOARCH == "amd64" {
			if msgSendStretAddr, err = purego.Dlsym(lib, "objc_msgSend_stret"); err != nil {
				loadErr = fmt.Errorf("objc: %w", err)
				return
			}
		}
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&objc_retain, "objc_retain"},
			{&objc_release, "objc_release"},
			{&objc_autorelease, "objc_autorelease"},
			{&objc_autoreleasePoolPush, "objc_autoreleasePoolPush"},
			{&objc_autoreleasePoolPop, "objc_autoreleasePoolPop"},
		} {
			addr, err := purego.Dlsym(lib, b.name)
			if err != nil {
				loadErr = fmt.Errorf("objc: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// Foundation returns the dlopen handle of Foundation.framework.
func Foundation() uintptr {
	mustLoad()
	return foundation
}

var (
	selectors sync.Map
	classes   sync.Map
)

// Sel returns the selector named name, registering it once and caching the result.
func Sel(name string) SEL {
	if s, ok := selectors.Load(name); ok {
		return s.(SEL)
	}
	mustLoad()
	s := pobjc.RegisterName(name)
	selectors.Store(name, s)
	return s
}

// GetClass returns the class named name, or 0 when it is not loaded. Found classes are cached.
func GetClass(name string) Class {
	if c, ok := classes.Load(name); ok {
		return c.(Class)
	}
	mustLoad()
	c := pobjc.GetClass(name)
	if c != 0 {
		classes.Store(name, c)
	}
	return c
}

// GetProtocol returns the protocol named name, or nil when it is not loaded.
func GetProtocol(name string) *Protocol {
	mustLoad()
	return pobjc.GetProtocol(name)
}

// Send sends sel to id and returns its result as T.
func Send[T any](id ID, sel SEL, args ...any) T {
	mustLoad()
	return pobjc.Send[T](id, sel, args...)
}

// SendVoid sends sel to id and discards any result.
func SendVoid(id ID, sel SEL, args ...any) {
	mustLoad()
	id.Send(sel, args...)
}

// SendClass sends sel to the class object c and returns its result as T.
func SendClass[T any](c Class, sel SEL, args ...any) T {
	return Send[T](ID(c), sel, args...)
}

// SendSuper sends sel to the superclass implementation of id's class.
func SendSuper[T any](id ID, sel SEL, args ...any) T {
	mustLoad()
	return pobjc.SendSuper[T](id, sel, args...)
}

// BindMsgSend binds fptr, a pointer to a func whose first two parameters are ID and SEL, to
// objc_msgSend, choosing objc_msgSend_stret on amd64 when the func returns a struct larger
// than 16 bytes.
func BindMsgSend(fptr any) {
	mustLoad()
	ty := reflect.TypeOf(fptr)
	if ty.Kind() != reflect.Pointer || ty.Elem().Kind() != reflect.Func {
		panic("objc: BindMsgSend needs a pointer to a func")
	}
	fn := ty.Elem()
	if fn.NumIn() < 2 || fn.In(0) != reflect.TypeFor[ID]() || fn.In(1) != reflect.TypeFor[SEL]() {
		panic("objc: BindMsgSend needs a func taking (ID, SEL, ...); got " + fn.String())
	}
	addr := msgSendAddr
	if runtime.GOARCH == "amd64" && fn.NumOut() == 1 && fn.Out(0).Kind() == reflect.Struct && fn.Out(0).Size() > maxRegisterStructSize {
		addr = msgSendStretAddr
	}
	purego.RegisterFunc(fptr, addr)
}

// Retain increments id's reference count and returns id.
func Retain(id ID) ID {
	mustLoad()
	if id == Nil {
		return Nil
	}
	return objc_retain(id)
}

// Release decrements id's reference count.
func Release(id ID) {
	mustLoad()
	if id != Nil {
		objc_release(id)
	}
}

// Autorelease adds id to the innermost autorelease pool and returns it.
func Autorelease(id ID) ID {
	mustLoad()
	if id == Nil {
		return Nil
	}
	return objc_autorelease(id)
}

// AutoreleasePool runs f inside a fresh autorelease pool, draining it when f returns or panics.
func AutoreleasePool(f func()) {
	mustLoad()
	pool := objc_autoreleasePoolPush()
	defer objc_autoreleasePoolPop(pool)
	f()
}

// Retained owns one strong reference to an object (objc2's Retained). Release is its drop.
type Retained struct {
	id       ID
	released atomic.Bool
}

// Own takes ownership of a +1 reference, such as the result of alloc/init, new or copy.
func Own(id ID) *Retained {
	return &Retained{id: id}
}

// RetainObject retains a +0 reference, such as an autoreleased return value, and owns the new one.
func RetainObject(id ID) *Retained {
	return &Retained{id: Retain(id)}
}

// ID returns the object; it panics after Release.
func (r *Retained) ID() ID {
	if r.released.Load() {
		panic("objc: Retained used after Release")
	}
	return r.id
}

// Release drops the owned reference. It is idempotent.
func (r *Retained) Release() {
	if r.released.Swap(true) {
		return
	}
	Release(r.id)
}

var (
	classMu         sync.Mutex
	registered      = map[string]Class{}
	registerFailure = map[string]error{}
)

// RegisterClass creates and registers the class name once per process, with the named protocols
// that exist on this system, ivars and methods. Later calls return the first result.
func RegisterClass(name string, superclass Class, protocols []string, ivars []FieldDef, methods []MethodDef) (Class, error) {
	mustLoad()
	classMu.Lock()
	defer classMu.Unlock()
	if c, ok := registered[name]; ok {
		return c, nil
	}
	if err, ok := registerFailure[name]; ok {
		return 0, err
	}
	var protos []*Protocol
	for _, p := range protocols {
		if proto := pobjc.GetProtocol(p); proto != nil {
			protos = append(protos, proto)
		}
	}
	c, err := pobjc.RegisterClass(name, superclass, protos, ivars, methods)
	if err != nil {
		err = fmt.Errorf("objc: register class %s: %w", name, err)
		registerFailure[name] = err
		return 0, err
	}
	registered[name] = c
	classes.Store(name, c)
	return c, nil
}

// RespondsToSelector reports whether id implements sel.
func RespondsToSelector(id ID, sel SEL) bool {
	return Send[bool](id, Sel("respondsToSelector:"), sel)
}

// InstancesRespondToSelector reports whether instances of c implement sel.
func InstancesRespondToSelector(c Class, sel SEL) bool {
	return SendClass[bool](c, Sel("instancesRespondToSelector:"), sel)
}

// Alloc sends alloc to c, returning a +1 uninitialised instance.
func Alloc(c Class) ID {
	return SendClass[ID](c, Sel("alloc"))
}

// AllocInit sends alloc then init to c, returning a +1 instance.
func AllocInit(c Class) ID {
	return Send[ID](Alloc(c), Sel("init"))
}
