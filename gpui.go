package gpui

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/scheduler"
)

// AccessibleAction is an AccessKit action, re-exported as upstream does.
type AccessibleAction = a11y.Action

// Orientation is the AccessKit orientation property.
type Orientation = a11y.Orientation

// Role is the AccessKit role of an accessibility node.
type Role = a11y.Role

// Toggled is the AccessKit toggled state.
type Toggled = a11y.Toggled

// Task is a spawned future; cancel it with Cancel or let it finish with Detach (D-RAII).
type Task[T any] = scheduler.Task[T]

// FallibleTask is a task that resolves to an error instead of panicking when cancelled.
type FallibleTask[T any] = scheduler.FallibleTask[T]

// Priority is the scheduling priority of a task.
type Priority = scheduler.Priority

// DedicatedExecutor runs tasks on a dedicated goroutine.
type DedicatedExecutor = scheduler.DedicatedExecutor

// SchedulerLocalExecutor is the scheduler's local (foreground) executor.
type SchedulerLocalExecutor = scheduler.LocalExecutor

// AppContext allows the different contexts in GPUI to be used interchangeably for certain
// operations. It is sealed: its primitives are unexported, so only gpui's contexts and types
// that embed one of them implement it. The generic operations are the AppContext* functions
// and the same-named methods on each context type (D-CTX).
type AppContext interface {
	newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity
	reserveEntity(entityType reflect.Type) AnyEntity
	insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity
	updateEntity(handle AnyEntity, update func(state any, cx *App))
	asMut(handle AnyEntity) *App
	readEntity(handle AnyEntity, read func(state any, cx *App))
	updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error
	withWindow(entityID EntityID, f func(window *Window, cx *App)) bool
	readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error
	backgroundExecutor() BackgroundExecutor
	readGlobal(globalType reflect.Type, read func(global any, cx *App))
}

// AppContextNew creates a new entity in the app context.
func AppContextNew[T any](cx AppContext, buildEntity func(cx *Context[T]) *T) Entity[T] {
	handle := cx.newEntity(reflect.TypeFor[T](), func(slot AnyEntity, app *App) any {
		return buildEntity(newContext(app, entityFromAny[T](slot).Downgrade()))
	})
	return entityFromAny[T](handle)
}

// AppContextReserveEntity reserves a slot for an entity to be inserted later. The returned
// Reservation allows you to obtain the EntityID for the future entity.
func AppContextReserveEntity[T any](cx AppContext) Reservation[T] {
	return Reservation[T]{slot: slot[T]{entityFromAny[T](cx.reserveEntity(reflect.TypeFor[T]()))}}
}

// AppContextInsertEntity inserts a new entity in the app context based on a Reservation
// previously obtained from AppContextReserveEntity.
func AppContextInsertEntity[T any](cx AppContext, reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	s := reservation.slot
	handle := cx.insertEntity(s.IntoAny(), func(app *App) any {
		return buildEntity(newContext(app, s.Downgrade()))
	})
	return entityFromAny[T](handle)
}

// AppContextUpdateEntity updates an entity in the app context.
func AppContextUpdateEntity[T, R any](cx AppContext, handle Entity[T], update func(entity *T, cx *Context[T]) R) R {
	var result R
	cx.updateEntity(handle.IntoAny(), func(state any, app *App) {
		result = update(state.(*T), newContext(app, handle.Downgrade()))
	})
	return result
}

// AppContextAsMut leases an entity for mutable access; Release on the returned GpuiBorrow
// notifies the entity and ends the lease, as its Drop does upstream.
func AppContextAsMut[T any](cx AppContext, handle Entity[T]) GpuiBorrow[T] {
	app := cx.asMut(handle.IntoAny())
	return newGpuiBorrow(handle, app)
}

// AppContextReadEntity reads an entity from the app context.
func AppContextReadEntity[T, R any](cx AppContext, handle Entity[T], read func(entity *T, cx *App) R) R {
	var result R
	cx.readEntity(handle.IntoAny(), func(state any, app *App) {
		result = read(state.(*T), app)
	})
	return result
}

// AppContextUpdateWindow updates a window for the given handle.
func AppContextUpdateWindow[R any](cx AppContext, window AnyWindowHandle, update func(root AnyView, window *Window, cx *App) R) (R, error) {
	var result R
	err := cx.updateWindow(window, func(root AnyView, w *Window, app *App) {
		result = update(root, w, app)
	})
	return result, err
}

// AppContextWithWindow runs f against the entity's current window, the most recently rendered
// window that referenced the entity. It reports false if the entity has no current window or
// that window is unavailable.
func AppContextWithWindow[R any](cx AppContext, entityID EntityID, f func(window *Window, cx *App) R) (R, bool) {
	var result R
	ok := cx.withWindow(entityID, func(w *Window, app *App) {
		result = f(w, app)
	})
	return result, ok
}

// AppContextReadWindow reads a window off of the application context.
func AppContextReadWindow[V, R any](cx AppContext, window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	var result R
	var downcastErr error
	err := cx.readWindow(window.AnyWindowHandle, func(root AnyView, app *App) {
		view, ok := anyViewDowncast[V](root)
		if !ok {
			downcastErr = errors.New("root view's type has changed")
			return
		}
		result = read(view, app)
	})
	if err != nil {
		return result, err
	}
	return result, downcastErr
}

// AppContextBackgroundSpawn spawns a future on a background thread.
func AppContextBackgroundSpawn[R any](cx AppContext, future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.backgroundExecutor().Spawn(future)
}

// AppContextReadGlobal reads a global from this app context.
func AppContextReadGlobal[G, R any](cx AppContext, callback func(global *G, cx *App) R) R {
	var result R
	cx.readGlobal(reflect.TypeFor[G](), func(global any, app *App) {
		result = callback(global.(*G), app)
	})
	return result
}

// Reservation is returned by AppContextReserveEntity to later be passed to
// AppContextInsertEntity. It allows you to obtain the EntityID for an entity before it is
// created.
type Reservation[T any] struct {
	slot slot[T]
}

// EntityID returns the EntityID that will be associated with the entity once it is inserted.
func (r Reservation[T]) EntityID() EntityID {
	return r.slot.EntityID()
}

// VisualContext is implemented by the contexts in GPUI that require a window to be present.
// Upstream's associated Result<T> is error / (R, error) for every Go implementation.
type VisualContext interface {
	AppContext
	WindowHandle() AnyWindowHandle
	updateWindowContext(update func(window *Window, cx *App)) error
}

// VisualContextUpdateWindowEntity updates a view with the given callback.
func VisualContextUpdateWindowEntity[T, R any](cx VisualContext, entity Entity[T], update func(entity *T, window *Window, cx *Context[T]) R) (R, error) {
	var result R
	err := cx.updateWindowContext(func(w *Window, app *App) {
		result = AppContextUpdateEntity(app, entity, func(state *T, ecx *Context[T]) R {
			return update(state, w, ecx)
		})
	})
	return result, err
}

// VisualContextNewWindowEntity creates a new entity, with access to the Window.
func VisualContextNewWindowEntity[T any](cx VisualContext, buildEntity func(window *Window, cx *Context[T]) *T) (Entity[T], error) {
	var result Entity[T]
	err := cx.updateWindowContext(func(w *Window, app *App) {
		result = AppContextNew(app, func(ecx *Context[T]) *T {
			return buildEntity(w, ecx)
		})
	})
	return result, err
}

// VisualContextReplaceRootView replaces the root view of a window with a new view. V must
// implement Render[V]; Window.ReplaceRoot panics naming the type otherwise (D-COND).
func VisualContextReplaceRootView[V any](cx VisualContext, buildView func(window *Window, cx *Context[V]) *V) (Entity[V], error) {
	var result Entity[V]
	err := cx.updateWindowContext(func(w *Window, app *App) {
		result = w.ReplaceRoot(app, buildView)
	})
	return result, err
}

// VisualContextFocus focuses an entity in the window. *V must implement Focusable; it panics
// naming the type otherwise (D-COND).
func VisualContextFocus[V any](cx VisualContext, entity Entity[V]) error {
	return cx.updateWindowContext(func(w *Window, app *App) {
		focusable, ok := any(entity.Read(app)).(Focusable)
		if !ok {
			panic(fmt.Sprintf("gpui: %s does not implement Focusable", reflect.TypeFor[*V]()))
		}
		handle := focusable.FocusHandle(app)
		handle.Focus(w, app)
		handle.Release()
	})
}

// BorrowAppContext is implemented by the contexts that can lend out the App mutably: *App and
// the types that embed it (Context[T]). SetGlobal, UpdateGlobal and UpdateDefaultGlobal are
// its generic operations (D-CTX).
type BorrowAppContext interface {
	borrowApp() *App
}

func (cx *App) borrowApp() *App {
	return cx
}

// SetGlobal sets a global value on the context.
func SetGlobal[G any, C BorrowAppContext](cx C, global G) {
	cx.borrowApp().SetGlobal(global)
}

// UpdateGlobal updates the global state of the given type.
func UpdateGlobal[G any, C BorrowAppContext](cx C, f func(global *G, cx C)) {
	UpdateGlobalR(cx, func(global *G, cx C) struct{} {
		f(global, cx)
		return struct{}{}
	})
}

// UpdateGlobalR updates the global state of the given type and returns the callback's result.
func UpdateGlobalR[G any, C BorrowAppContext, R any](cx C, f func(global *G, cx C) R) R {
	global := cx.borrowApp().leaseGlobal[G]()
	result := f(global.get(), cx)
	cx.borrowApp().endGlobalLease(global)
	return result
}

// UpdateDefaultGlobal updates the global state of the given type, creating a default if it
// didn't exist before.
func UpdateDefaultGlobal[G any, C BorrowAppContext](cx C, f func(global *G, cx C)) {
	UpdateDefaultGlobalR(cx, func(global *G, cx C) struct{} {
		f(global, cx)
		return struct{}{}
	})
}

// UpdateDefaultGlobalR updates the global state of the given type, creating a default if it
// didn't exist before, and returns the callback's result.
func UpdateDefaultGlobalR[G any, C BorrowAppContext, R any](cx C, f func(global *G, cx C) R) R {
	cx.borrowApp().DefaultGlobal[G]()
	return UpdateGlobalR(cx, f)
}

// GPUSpecs is information about the GPU GPUI is running on.
type GPUSpecs struct {
	// IsSoftwareEmulated reports whether the GPU is really a fake (like llvmpipe) running on the CPU.
	IsSoftwareEmulated bool `json:"is_software_emulated"`
	// DeviceName is the name of the device, as reported by Vulkan.
	DeviceName string `json:"device_name"`
	// DriverName is the name of the driver, as reported by Vulkan.
	DriverName string `json:"driver_name"`
	// DriverInfo is further information about the driver, as reported by Vulkan.
	DriverInfo string `json:"driver_info"`
}
