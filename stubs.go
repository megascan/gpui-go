package gpui

import (
	"encoding/json/jsontext"
	"iter"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/jsonschema"
	"github.com/megascan/gpui-go/scheduler"
)

// App is the GPUI application context (M0-app-core ports it).
type App struct{}

// FindDisplay returns the display with the given id, or nil (M0-app-core ports it).
func (cx *App) FindDisplay(id DisplayID) PlatformDisplay {
	panic("gpui: unimplemented App.FindDisplay")
}

// PrimaryDisplay returns the primary display, or nil (M0-app-core ports it).
func (cx *App) PrimaryDisplay() PlatformDisplay {
	panic("gpui: unimplemented App.PrimaryDisplay")
}

// New builds an entity that is owned by the application (M0-app-contexts ports it).
func (cx *App) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	panic("gpui: unimplemented App.New")
}

// ReserveEntity reserves a slot for an entity to be inserted later (M0-app-contexts ports it).
func (cx *App) ReserveEntity[T any]() Reservation[T] {
	panic("gpui: unimplemented App.ReserveEntity")
}

// InsertEntity inserts an entity into a reserved slot (M0-app-contexts ports it).
func (cx *App) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	panic("gpui: unimplemented App.InsertEntity")
}

// AsMut leases an entity for mutable access (M0-app-contexts ports it).
func (cx *App) AsMut[T any](handle Entity[T]) GpuiBorrow[T] {
	panic("gpui: unimplemented App.AsMut")
}

// ReadWindow reads a window off of the application context (M0-app-contexts ports it).
func (cx *App) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	panic("gpui: unimplemented App.ReadWindow")
}

// WithWindow runs f against the entity's current window (M0-app-core ports it).
func (cx *App) WithWindow[R any](entityID EntityID, f func(window *Window, cx *App) R) (R, bool) {
	panic("gpui: unimplemented App.WithWindow")
}

// BackgroundSpawn spawns a future on a background thread (M0-app-contexts ports it).
func (cx *App) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	panic("gpui: unimplemented App.BackgroundSpawn")
}

// ReadGlobal reads a global from this app context (M0-app-contexts ports it).
func (cx *App) ReadGlobal[G, R any](callback func(global *G, cx *App) R) R {
	panic("gpui: unimplemented App.ReadGlobal")
}

// SetGlobal sets the value of the global of the given type (M0-app-effects ports it).
func (cx *App) SetGlobal[G any](global G) {
	panic("gpui: unimplemented App.SetGlobal")
}

// DefaultGlobal returns the global of the given type, inserting its zero value if missing (M0-app-effects ports it).
func (cx *App) DefaultGlobal[G any]() *G {
	panic("gpui: unimplemented App.DefaultGlobal")
}

// Observe arranges for f to be called whenever the entity notifies (M0-app-effects ports it).
func (cx *App) Observe[W any](entity Entity[W], onNotify func(entity Entity[W], cx *App)) Subscription {
	panic("gpui: unimplemented App.Observe")
}

// Spawn spawns the future returned by f on the foreground executor (M0-app-core ports it).
func (cx *App) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R] {
	panic("gpui: unimplemented App.Spawn")
}

// Defer schedules f to run at the end of the current effect cycle (M0-app-effects ports it).
func (cx *App) Defer(f func(cx *App)) {
	panic("gpui: unimplemented App.Defer")
}

// OpenWindow opens a new window whose root view is returned by build (M0-app-core ports it).
func (cx *App) OpenWindow[V any](options WindowOptions, build func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	panic("gpui: unimplemented App.OpenWindow")
}

func (cx *App) leaseGlobal[G any]() *globalLease[G] {
	panic("gpui: unimplemented App.leaseGlobal")
}

func (cx *App) endGlobalLease[G any](lease *globalLease[G]) {
	panic("gpui: unimplemented App.endGlobalLease")
}

func (cx *App) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	panic("gpui: unimplemented App.newEntity")
}

func (cx *App) reserveEntity(entityType reflect.Type) AnyEntity {
	panic("gpui: unimplemented App.reserveEntity")
}

func (cx *App) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	panic("gpui: unimplemented App.insertEntity")
}

func (cx *App) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	panic("gpui: unimplemented App.updateEntity")
}

func (cx *App) asMut(handle AnyEntity) *App {
	panic("gpui: unimplemented App.asMut")
}

func (cx *App) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	panic("gpui: unimplemented App.readEntity")
}

func (cx *App) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	panic("gpui: unimplemented App.updateWindow")
}

func (cx *App) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	panic("gpui: unimplemented App.withWindow")
}

func (cx *App) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	panic("gpui: unimplemented App.readWindow")
}

func (cx *App) backgroundExecutor() BackgroundExecutor {
	panic("gpui: unimplemented App.backgroundExecutor")
}

func (cx *App) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	panic("gpui: unimplemented App.readGlobal")
}

type globalLease[G any] struct {
	global any
}

func (l *globalLease[G]) get() *G {
	panic("gpui: unimplemented globalLease.get")
}

// AppCell wraps the App with upstream's borrow tracking (M0-app-core ports it).
type AppCell struct{}

// Borrow starts a shared borrow of the App; Release on the guard ends it (M0-app-core ports it).
func (c *AppCell) Borrow() AppRef {
	panic("gpui: unimplemented AppCell.Borrow")
}

// BorrowMut starts a mutable borrow of the App and panics on a conflicting borrow (M0-app-core ports it).
func (c *AppCell) BorrowMut() AppRefMut {
	panic("gpui: unimplemented AppCell.BorrowMut")
}

// TryBorrowMut starts a mutable borrow of the App or returns the borrow error (M0-app-core ports it).
func (c *AppCell) TryBorrowMut() (AppRefMut, error) {
	panic("gpui: unimplemented AppCell.TryBorrowMut")
}

// AppRef is a shared borrow of the App (M0-app-core ports it). It does not embed *App, so a
// shared borrow implements neither AppContext nor BorrowAppContext.
type AppRef struct {
	app *App
}

// Get returns the borrowed App for reads only, as upstream's Deref to &App; mutation needs
// BorrowMut (M0-app-core ports it).
func (r AppRef) Get() *App {
	panic("gpui: unimplemented AppRef.Get")
}

// Release ends the borrow, as the guard's Drop does upstream (M0-app-core ports it).
func (r AppRef) Release() {
	panic("gpui: unimplemented AppRef.Release")
}

// AppRefMut is a mutable borrow of the App (M0-app-core ports it).
type AppRefMut struct {
	*App
}

// Release ends the borrow, as the guard's Drop does upstream (M0-app-core ports it).
func (r AppRefMut) Release() {
	panic("gpui: unimplemented AppRefMut.Release")
}

// GpuiBorrow is a mutable lease of an entity obtained with AsMut (M0-app-effects ports it).
type GpuiBorrow[T any] struct {
	inner Entity[T]
	app   *App
}

func newGpuiBorrow[T any](inner Entity[T], app *App) GpuiBorrow[T] {
	panic("gpui: unimplemented newGpuiBorrow")
}

// Get returns the leased entity state (M0-app-effects ports it).
func (b GpuiBorrow[T]) Get() *T {
	panic("gpui: unimplemented GpuiBorrow.Get")
}

// Release notifies the entity, ends the lease and finishes the update, as its Drop does upstream (M0-app-effects ports it).
func (b GpuiBorrow[T]) Release() {
	panic("gpui: unimplemented GpuiBorrow.Release")
}

// DisplayID is an opaque identifier for a hardware display (M0-platform-types ports it).
type DisplayID uint64

// PlatformDisplay is a display attached to the platform (M0-platform-types ports the full method set).
type PlatformDisplay interface {
	ID() DisplayID
	Bounds() Bounds[Pixels]
	VisibleBounds() Bounds[Pixels]
}

// WindowOptions configures a new window (M0-platform-data ports it).
type WindowOptions struct{}

// EntityID is the identifier of an entity (M0-entities ports it).
type EntityID uint64

// AnyEntity is a type-erased strong handle to an entity (M0-entities ports it).
type AnyEntity struct {
	entityID EntityID
}

// EntityID returns the id of the entity (M0-entities ports it).
func (a AnyEntity) EntityID() EntityID {
	panic("gpui: unimplemented AnyEntity.EntityID")
}

// EntityType returns the type of the entity's state (M0-entities ports it).
func (a AnyEntity) EntityType() reflect.Type {
	panic("gpui: unimplemented AnyEntity.EntityType")
}

// Clone returns a new counted handle to the same entity (M0-entities ports it).
func (a AnyEntity) Clone() AnyEntity {
	panic("gpui: unimplemented AnyEntity.Clone")
}

// Release drops this handle (M0-entities ports it).
func (a AnyEntity) Release() {
	panic("gpui: unimplemented AnyEntity.Release")
}

// Downgrade returns a weak handle to the entity (M0-entities ports it).
func (a AnyEntity) Downgrade() AnyWeakEntity {
	panic("gpui: unimplemented AnyEntity.Downgrade")
}

// AnyWeakEntity is a type-erased weak handle to an entity (M0-entities ports it).
type AnyWeakEntity struct {
	entityID EntityID
}

// EntityID returns the id of the entity (M0-entities ports it).
func (w AnyWeakEntity) EntityID() EntityID {
	panic("gpui: unimplemented AnyWeakEntity.EntityID")
}

// Upgrade returns a strong handle if the entity is still alive (M0-entities ports it).
func (w AnyWeakEntity) Upgrade() (AnyEntity, bool) {
	panic("gpui: unimplemented AnyWeakEntity.Upgrade")
}

// Entity is a strong handle to an entity of type T (M0-entities ports it).
type Entity[T any] struct {
	AnyEntity
}

func entityFromAny[T any](handle AnyEntity) Entity[T] {
	panic("gpui: unimplemented entityFromAny")
}

// EntityID returns the id of the entity (M0-entities ports it).
func (e Entity[T]) EntityID() EntityID {
	panic("gpui: unimplemented Entity.EntityID")
}

// Clone returns a new counted handle to the same entity (M0-entities ports it).
func (e Entity[T]) Clone() Entity[T] {
	panic("gpui: unimplemented Entity.Clone")
}

// Release drops this handle (M0-entities ports it).
func (e Entity[T]) Release() {
	panic("gpui: unimplemented Entity.Release")
}

// Downgrade returns a weak handle to the entity (M0-entities ports it).
func (e Entity[T]) Downgrade() WeakEntity[T] {
	panic("gpui: unimplemented Entity.Downgrade")
}

// IntoAny returns this handle as an AnyEntity; both values alias one counted handle (M0-entities ports it).
func (e Entity[T]) IntoAny() AnyEntity {
	panic("gpui: unimplemented Entity.IntoAny")
}

// Read returns the entity's state (M0-entities ports it).
func (e Entity[T]) Read(cx *App) *T {
	panic("gpui: unimplemented Entity.Read")
}

// ReadWith reads the entity's state with a callback (M0-entities ports it).
func (e Entity[T]) ReadWith[R any](cx AppContext, f func(entity *T, cx *App) R) R {
	panic("gpui: unimplemented Entity.ReadWith")
}

// Update updates the entity's state (M0-entities ports it).
func (e Entity[T]) Update(cx AppContext, update func(entity *T, cx *Context[T])) {
	panic("gpui: unimplemented Entity.Update")
}

// UpdateR updates the entity's state and returns the callback's result (M0-entities ports it).
func (e Entity[T]) UpdateR[R any](cx AppContext, update func(entity *T, cx *Context[T]) R) R {
	panic("gpui: unimplemented Entity.UpdateR")
}

// UpdateIn updates the entity's state with access to the context's window (M0-entities ports it).
func (e Entity[T]) UpdateIn(cx VisualContext, update func(entity *T, window *Window, cx *Context[T])) error {
	panic("gpui: unimplemented Entity.UpdateIn")
}

// UpdateInR updates the entity's state with access to the window and returns the result (M0-entities ports it).
func (e Entity[T]) UpdateInR[R any](cx VisualContext, update func(entity *T, window *Window, cx *Context[T]) R) (R, error) {
	panic("gpui: unimplemented Entity.UpdateInR")
}

// IntoAnyElement renders the entity as a view; *T must implement Render[T] (M0-element ports it).
func (e Entity[T]) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Entity.IntoAnyElement")
}

// WeakEntity is a weak handle to an entity of type T (M0-entities ports it).
type WeakEntity[T any] struct {
	AnyWeakEntity
}

// Upgrade returns a strong handle if the entity is still alive (M0-entities ports it).
func (w WeakEntity[T]) Upgrade() (Entity[T], bool) {
	panic("gpui: unimplemented WeakEntity.Upgrade")
}

// Update updates the entity's state if it is still alive (M0-entities ports it).
func (w WeakEntity[T]) Update(cx AppContext, update func(entity *T, cx *Context[T])) error {
	panic("gpui: unimplemented WeakEntity.Update")
}

// UpdateR updates the entity's state if it is still alive and returns the result (M0-entities ports it).
func (w WeakEntity[T]) UpdateR[R any](cx AppContext, update func(entity *T, cx *Context[T]) R) (R, error) {
	panic("gpui: unimplemented WeakEntity.UpdateR")
}

type slot[T any] struct {
	Entity[T]
}

// Subscription keeps a callback registered until Unsubscribe or Detach (M0-entities ports it).
type Subscription struct {
	unsubscribe func()
}

// Unsubscribe removes the callback, as dropping the Subscription does upstream (M0-entities ports it).
func (s Subscription) Unsubscribe() {
	panic("gpui: unimplemented Subscription.Unsubscribe")
}

// Detach keeps the callback registered for the lifetime of its emitter (M0-entities ports it).
func (s Subscription) Detach() {
	panic("gpui: unimplemented Subscription.Detach")
}

// BackgroundExecutor spawns tasks on background threads (M0-executors ports it).
type BackgroundExecutor struct {
	inner scheduler.BackgroundExecutor
}

// Spawn spawns a future on a background thread (M0-executors ports it).
func (e BackgroundExecutor) Spawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	panic("gpui: unimplemented BackgroundExecutor.Spawn")
}

// ForegroundExecutor spawns tasks on the main thread (M0-executors ports it).
type ForegroundExecutor struct {
	inner scheduler.LocalExecutor
}

// Spawn spawns a future on the main thread (M0-executors ports it).
func (e ForegroundExecutor) Spawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	panic("gpui: unimplemented ForegroundExecutor.Spawn")
}

// Context is the context of an entity update: the App plus the entity being updated (M0-app-contexts ports it).
type Context[T any] struct {
	*App
	entityState WeakEntity[T]
}

func newContext[T any](app *App, entity WeakEntity[T]) *Context[T] {
	panic("gpui: unimplemented newContext")
}

// Notify tells observers of this entity that it changed (M0-app-contexts ports it).
func (cx *Context[T]) Notify() {
	panic("gpui: unimplemented Context.Notify")
}

// Observe arranges for f to be called whenever the observed entity notifies (M0-app-contexts ports it).
func (cx *Context[T]) Observe[W any](entity Entity[W], onNotify func(this *T, entity Entity[W], cx *Context[T])) Subscription {
	panic("gpui: unimplemented Context.Observe")
}

// Subscribe arranges for f to be called whenever the entity emits an event of type E (M0-app-contexts ports it).
func (cx *Context[T]) Subscribe[T2, E any](entity Entity[T2], onEvent func(this *T, entity Entity[T2], event *E, cx *Context[T])) Subscription {
	panic("gpui: unimplemented Context.Subscribe")
}

// Emit emits an event of type E to the entity's subscribers (M0-app-contexts ports it).
func (cx *Context[T]) Emit[E any](event E) {
	panic("gpui: unimplemented Context.Emit")
}

// Spawn spawns the future returned by f with a weak handle to this entity (M0-app-contexts ports it).
func (cx *Context[T]) Spawn[R any](f func(this WeakEntity[T], cx *AsyncApp) (R, error)) *Task[R] {
	panic("gpui: unimplemented Context.Spawn")
}

// Listener converts a listener on this entity into a window event listener (M0-app-contexts ports it).
func (cx *Context[T]) Listener[E any](listener func(this *T, event *E, window *Window, cx *Context[T])) func(event *E, window *Window, cx *App) {
	panic("gpui: unimplemented Context.Listener")
}

// AsyncApp is an App handle that can be held across await points (M0-app-contexts ports it).
type AsyncApp struct {
	*scheduler.Scope
}

// Update runs f with the App, parking the task until the main thread runs it (M0-app-contexts ports it).
func (cx *AsyncApp) Update(f func(cx *App)) {
	panic("gpui: unimplemented AsyncApp.Update")
}

// UpdateR runs f with the App and returns its result (M0-app-contexts ports it).
func (cx *AsyncApp) UpdateR[R any](f func(cx *App) R) R {
	panic("gpui: unimplemented AsyncApp.UpdateR")
}

// New creates a new entity (M0-app-contexts ports it).
func (cx *AsyncApp) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	panic("gpui: unimplemented AsyncApp.New")
}

// ReadGlobal reads a global (M0-app-contexts ports it).
func (cx *AsyncApp) ReadGlobal[G, R any](read func(global *G, cx *App) R) R {
	panic("gpui: unimplemented AsyncApp.ReadGlobal")
}

// BackgroundSpawn spawns a future on a background thread (M0-app-contexts ports it).
func (cx *AsyncApp) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	panic("gpui: unimplemented AsyncApp.BackgroundSpawn")
}

func (cx *AsyncApp) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	panic("gpui: unimplemented AsyncApp.newEntity")
}

func (cx *AsyncApp) reserveEntity(entityType reflect.Type) AnyEntity {
	panic("gpui: unimplemented AsyncApp.reserveEntity")
}

func (cx *AsyncApp) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	panic("gpui: unimplemented AsyncApp.insertEntity")
}

func (cx *AsyncApp) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	panic("gpui: unimplemented AsyncApp.updateEntity")
}

func (cx *AsyncApp) asMut(handle AnyEntity) *App {
	panic("gpui: unimplemented AsyncApp.asMut")
}

func (cx *AsyncApp) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	panic("gpui: unimplemented AsyncApp.readEntity")
}

func (cx *AsyncApp) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	panic("gpui: unimplemented AsyncApp.updateWindow")
}

func (cx *AsyncApp) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	panic("gpui: unimplemented AsyncApp.withWindow")
}

func (cx *AsyncApp) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	panic("gpui: unimplemented AsyncApp.readWindow")
}

func (cx *AsyncApp) backgroundExecutor() BackgroundExecutor {
	panic("gpui: unimplemented AsyncApp.backgroundExecutor")
}

func (cx *AsyncApp) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	panic("gpui: unimplemented AsyncApp.readGlobal")
}

// AnyView is a type-erased handle to an entity that implements Render (M0-element ports it).
type AnyView struct {
	entity AnyEntity
}

func anyViewDowncast[V any](view AnyView) (Entity[V], bool) {
	panic("gpui: unimplemented anyViewDowncast")
}

// View is an entity or component that can render itself (M0-element ports it).
type View interface {
	EntityID() (EntityID, bool)
	Render(window *Window, cx *App) IntoElement
}

// Render is implemented by entity states that can be drawn as views (M0-element ports it).
type Render[T any] interface {
	Render(window *Window, cx *Context[T]) IntoElement
}

// RenderOnce is implemented by components that render once into an element tree (M0-element ports it).
type RenderOnce interface {
	Render(window *Window, cx *App) IntoElement
}

// WindowID is the identifier of a window (M0-window-core ports it).
type WindowID uint64

// AnyWindowHandle is a type-erased handle to a window (M0-window-core ports it).
type AnyWindowHandle struct {
	id        WindowID
	stateType reflect.Type
}

// WindowHandle is a handle to a window whose root view has type V (M0-window-core ports it).
type WindowHandle[V any] struct {
	AnyWindowHandle
}

// Window is a GPUI window (M0-window-core ports it).
type Window struct{}

// RemSize returns the window's rem size (M0-window-paint ports it).
func (w *Window) RemSize() Pixels {
	panic("gpui: unimplemented Window.RemSize")
}

// Draw renders the window's root views into the next frame (M0-window-paint ports it).
func (w *Window) Draw(cx *App) ArenaClearNeeded {
	panic("gpui: unimplemented Window.Draw")
}

// RequestLayout adds a node to the layout tree for the current frame (M0-window-dispatch ports it).
func (w *Window) RequestLayout(style Style, children []LayoutID, cx *App) LayoutID {
	panic("gpui: unimplemented Window.RequestLayout")
}

// WithElementState gives f the element state of type S stored under id (M0-window-paint ports it).
func (w *Window) WithElementState[S, R any](id *GlobalElementID, f func(state Option[S], window *Window) (R, S)) R {
	panic("gpui: unimplemented Window.WithElementState")
}

// PaintQuad paints a quad into the scene for the next frame (M0-window-paint ports it).
func (w *Window) PaintQuad(quad PaintQuad) {
	panic("gpui: unimplemented Window.PaintQuad")
}

// InsertHitbox registers a hitbox for the current frame (M0-window-dispatch ports it).
func (w *Window) InsertHitbox(bounds Bounds[Pixels], behavior HitboxBehavior) Hitbox {
	panic("gpui: unimplemented Window.InsertHitbox")
}

// DispatchKeystroke dispatches a keystroke as if it had been typed (M0-window-dispatch ports it).
func (w *Window) DispatchKeystroke(keystroke Keystroke, cx *App) bool {
	panic("gpui: unimplemented Window.DispatchKeystroke")
}

// DispatchEvent dispatches a platform input event (M0-window-dispatch ports it).
func (w *Window) DispatchEvent(event PlatformInput, cx *App) DispatchEventResult {
	panic("gpui: unimplemented Window.DispatchEvent")
}

// Refresh schedules a redraw of the window (M0-window-paint ports it).
func (w *Window) Refresh() {
	panic("gpui: unimplemented Window.Refresh")
}

// OnNextFrame schedules f to run before the next frame (M0-window-paint ports it).
func (w *Window) OnNextFrame(f func(window *Window, cx *App)) {
	panic("gpui: unimplemented Window.OnNextFrame")
}

// RequestAnimationFrame schedules a redraw for the next frame (M0-window-paint ports it).
func (w *Window) RequestAnimationFrame() {
	panic("gpui: unimplemented Window.RequestAnimationFrame")
}

// ReplaceRoot replaces the window's root view; *V must implement Render[V] (M0-window-paint ports it).
func (w *Window) ReplaceRoot[V any](cx *App, buildView func(window *Window, cx *Context[V]) *V) Entity[V] {
	panic("gpui: unimplemented Window.ReplaceRoot")
}

// ArenaClearNeeded is returned by Window.Draw (M0-window-core ports it).
type ArenaClearNeeded struct{}

// DispatchPhase is the phase of event dispatch (M0-window-core ports it).
type DispatchPhase uint8

const (
	// DispatchPhaseBubble dispatches from the innermost element outwards.
	DispatchPhaseBubble DispatchPhase = iota
	// DispatchPhaseCapture dispatches from the window inwards.
	DispatchPhaseCapture
)

// FocusHandle is a ref-counted handle to a focusable element (M0-window-core ports it).
type FocusHandle struct {
	// TabIndex is the element's position in the tab order.
	TabIndex int
	// TabStop reports whether the element is reachable with tab navigation.
	TabStop bool
}

// Focus moves focus to the element (M0-window-core ports it).
func (h FocusHandle) Focus(window *Window, cx *App) {
	panic("gpui: unimplemented FocusHandle.Focus")
}

// Clone returns a new counted handle (M0-window-core ports it).
func (h FocusHandle) Clone() FocusHandle {
	panic("gpui: unimplemented FocusHandle.Clone")
}

// Release drops this handle (M0-window-core ports it).
func (h FocusHandle) Release() {
	panic("gpui: unimplemented FocusHandle.Release")
}

// Focusable is implemented by views that have a focus handle (M0-window-core ports it).
type Focusable interface {
	FocusHandle(cx *App) FocusHandle
}

// ContentMask restricts painting to the given bounds (M0-window-core ports it).
type ContentMask[T Number] struct {
	// Bounds is the area painting is clipped to.
	Bounds Bounds[T]
}

// ElementID is the identifier of an element among its siblings (M0-window-core ports it).
type ElementID struct {
	kind uint8
	name SharedString
	num  uint64
}

// ElementIDSource is the set of types an ElementID can be built from (M0-window-core ports it).
type ElementIDSource interface {
	~string | ~int | ~uint64 | ElementID
}

// GlobalElementID is the path of element ids from the window root (M0-element ports it).
type GlobalElementID struct {
	path []ElementID
}

// InspectorElementID identifies an element in the inspector (M0-element ports it).
type InspectorElementID struct{}

// SourceLocation is a source position recorded when an element is constructed (M0-element ports it).
type SourceLocation struct {
	// File is the source file name.
	File string
	// Line is the 1-based line number.
	Line uint32
	// Column is the 1-based column number.
	Column uint32
}

// Hitbox is a region that receives mouse events (M0-window-core ports it).
type Hitbox struct{}

// HitboxBehavior configures how a hitbox affects hit testing (M0-window-core ports it).
type HitboxBehavior uint8

// PaintQuad describes a quad to paint (M0-window-core ports it).
type PaintQuad struct{}

// DispatchEventResult reports what happened during event dispatch (M0-window-dispatch ports it).
type DispatchEventResult struct{}

// A11ySubtreeBuilder adds synthetic accessibility nodes below an element (M0-window-a11y ports it).
type A11ySubtreeBuilder struct {
	parentID a11y.NodeID
}

// Element is implemented by types that participate in laying out and painting the contents of a window (M0-element ports it).
type Element[RL, PP any] interface {
	ElementID() (ElementID, bool)
	SourceLocation() *SourceLocation
	RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, RL)
	Prepaint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, window *Window, cx *App) PP
	Paint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, prepaint *PP, window *Window, cx *App)
}

// ElementA11y is the optional accessibility hook of an Element (M0-element ports it).
type ElementA11y interface {
	A11yRole() (a11y.Role, bool)
	WriteA11yInfo(node *a11y.Node)
}

// ElementA11ySyntheticChildren is the optional hook that adds synthetic accessibility children (M0-element ports it).
type ElementA11ySyntheticChildren[PP any] interface {
	ElementA11ySyntheticChildren(prepaint *PP, builder *A11ySubtreeBuilder)
}

// IntoElement is implemented by any type that can be converted into an element (M0-element ports it).
type IntoElement interface {
	IntoAnyElement() AnyElement
}

// AnyElement is a type-erased element (M0-element ports it).
type AnyElement struct {
	element any
}

// NewAnyElement wraps a typed element (M0-element ports it).
func NewAnyElement[RL, PP any](element Element[RL, PP]) AnyElement {
	panic("gpui: unimplemented NewAnyElement")
}

// IntoAnyElement returns the element itself.
func (e AnyElement) IntoAnyElement() AnyElement {
	return e
}

// RequestLayout requests layout for the element (M0-element ports it).
func (e *AnyElement) RequestLayout(window *Window, cx *App) LayoutID {
	panic("gpui: unimplemented AnyElement.RequestLayout")
}

// LayoutAsRoot lays the element out as the root of a layout tree (M0-element ports it).
func (e *AnyElement) LayoutAsRoot(availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels] {
	panic("gpui: unimplemented AnyElement.LayoutAsRoot")
}

// PrepaintAt prepaints the element at the given origin (M0-element ports it).
func (e *AnyElement) PrepaintAt(origin Point[Pixels], window *Window, cx *App) {
	panic("gpui: unimplemented AnyElement.PrepaintAt")
}

// Paint paints the element (M0-element ports it).
func (e *AnyElement) Paint(window *Window, cx *App) {
	panic("gpui: unimplemented AnyElement.Paint")
}

// LayoutID identifies a node in the layout tree (M0-element ports it).
type LayoutID struct {
	node uint64
}

// AvailableSpace is the space available to an element during layout (M0-element ports it).
type AvailableSpace struct {
	kind     uint8
	definite Pixels
}

// Style is the computed style of an element (M0-style ports it).
type Style struct{}

// StyleRefinement is a partial Style (M0-style ports it).
type StyleRefinement struct{}

// Styler is implemented by elements that embed Styled (M0-style ports it).
type Styler interface {
	Style() *StyleRefinement
}

// Styled is the fluent styling mixin; bind it with BindStyled (M0-style ports it).
type Styled[S Styler] struct {
	self S
}

// BindStyled binds a Styled mixin to the element that embeds it.
func BindStyled[S Styler](m *Styled[S], self S) {
	m.self = self
}

// Flex sets the element to display flex (M0-style ports it).
func (s *Styled[S]) Flex() S {
	panic("gpui: unimplemented Styled.Flex")
}

// SizeFull sets the element's width and height to 100% (M0-style ports it).
func (s *Styled[S]) SizeFull() S {
	panic("gpui: unimplemented Styled.SizeFull")
}

// Interactivity holds the listeners and interactive style of an element (M0-div-api ports it).
type Interactivity struct{}

// InteractiveElement is implemented by elements that embed Interactive (M0-div-api ports it).
type InteractiveElement interface {
	Interactivity() *Interactivity
}

// Interactive is the InteractiveElement mixin; bind it with BindInteractive (M0-div-api ports it).
type Interactive[S InteractiveElement] struct {
	self S
}

// BindInteractive binds an Interactive mixin to the element that embeds it.
func BindInteractive[S InteractiveElement](m *Interactive[S], self S) {
	m.self = self
}

// OnClick registers a click listener (M0-div-api ports it).
func (i *Interactive[S]) OnClick(listener func(event *ClickEvent, window *Window, cx *App)) S {
	panic("gpui: unimplemented Interactive.OnClick")
}

// InteractiveID carries InteractiveElement's ID method, which cannot live on Interactive (D-IDMIXIN); bind it with BindInteractiveID (M0-div-api ports it).
type InteractiveID[S InteractiveElement] struct {
	self S
}

// BindInteractiveID binds an InteractiveID mixin to the element that embeds it.
func BindInteractiveID[S InteractiveElement](m *InteractiveID[S], self S) {
	m.self = self
}

// ID assigns the element an id, making it stateful (M0-div-api ports it).
func (i *InteractiveID[S]) ID[I ElementIDSource](id I) *Stateful[S] {
	panic("gpui: unimplemented InteractiveID.ID")
}

// StatefulInteractiveElement is implemented by stateful interactive elements (M0-div-api ports it).
type StatefulInteractiveElement interface {
	InteractiveElement
}

// StatefulInteractive is the StatefulInteractiveElement mixin; bind it with BindStatefulInteractive (M0-div-api ports it).
type StatefulInteractive[S StatefulInteractiveElement] struct {
	self S
}

// BindStatefulInteractive binds a StatefulInteractive mixin to the element that embeds it.
func BindStatefulInteractive[S StatefulInteractiveElement](m *StatefulInteractive[S], self S) {
	m.self = self
}

// ParentElement is implemented by elements that embed Parent (M0-element ports it).
type ParentElement interface {
	Extend(elements []AnyElement)
}

// Parent is the ParentElement mixin; bind it with BindParent (M0-element ports it).
type Parent[S ParentElement] struct {
	self S
}

// BindParent binds a Parent mixin to the element that embeds it.
func BindParent[S ParentElement](m *Parent[S], self S) {
	m.self = self
}

// Child adds a child element (M0-element ports it).
func (p *Parent[S]) Child(child IntoElement) S {
	panic("gpui: unimplemented Parent.Child")
}

// Children adds child elements (M0-element ports it).
func (p *Parent[S]) Children(children ...IntoElement) S {
	panic("gpui: unimplemented Parent.Children")
}

// ChildText adds a text child (M0-element ports it).
func (p *Parent[S]) ChildText(text string) S {
	panic("gpui: unimplemented Parent.ChildText")
}

// Fluent is the FluentBuilder mixin; bind it with BindFluent (M0-input-types ports it).
type Fluent[S any] struct {
	self S
}

// BindFluent binds a Fluent mixin to the element that embeds it.
func BindFluent[S any](m *Fluent[S], self S) {
	m.self = self
}

// When conditionally modifies self with the given closure (M0-input-types ports it).
func (f *Fluent[S]) When(condition bool, then func(this S) S) S {
	panic("gpui: unimplemented Fluent.When")
}

// Map applies a function to self (M0-input-types ports it).
func (f *Fluent[S]) Map[U any](fn func(this S) U) U {
	panic("gpui: unimplemented Fluent.Map")
}

// Div is the basic container element (M0-div-api ports it).
type Div struct {
	Styled[*Div]
	Interactive[*Div]
	InteractiveID[*Div]
	Parent[*Div]
	Fluent[*Div]
	interactivity Interactivity
	children      []AnyElement
}

// NewDiv constructs a Div (M0-div-api ports it).
func NewDiv() *Div {
	panic("gpui: unimplemented NewDiv")
}

// Style returns the element's style refinement (M0-div-api ports it).
func (d *Div) Style() *StyleRefinement {
	panic("gpui: unimplemented Div.Style")
}

// Interactivity returns the element's interactivity (M0-div-api ports it).
func (d *Div) Interactivity() *Interactivity {
	panic("gpui: unimplemented Div.Interactivity")
}

// Extend adds child elements (M0-div-api ports it).
func (d *Div) Extend(elements []AnyElement) {
	panic("gpui: unimplemented Div.Extend")
}

// IntoAnyElement converts the Div into an AnyElement (M0-div-paint ports it).
func (d *Div) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Div.IntoAnyElement")
}

// Stateful wraps an element that has an id; it is IntoElement only and delegates to its element (D-COND, M0-div-paint ports it).
type Stateful[E any] struct {
	Styled[*Stateful[E]]
	Interactive[*Stateful[E]]
	StatefulInteractive[*Stateful[E]]
	Parent[*Stateful[E]]
	Fluent[*Stateful[E]]
	element E
}

// Style returns the inner element's style refinement; E must implement Styler (M0-div-paint ports it).
func (s *Stateful[E]) Style() *StyleRefinement {
	panic("gpui: unimplemented Stateful.Style")
}

// Interactivity returns the inner element's interactivity; E must implement InteractiveElement (M0-div-paint ports it).
func (s *Stateful[E]) Interactivity() *Interactivity {
	panic("gpui: unimplemented Stateful.Interactivity")
}

// Extend adds children to the inner element; E must implement ParentElement (M0-div-paint ports it).
func (s *Stateful[E]) Extend(elements []AnyElement) {
	panic("gpui: unimplemented Stateful.Extend")
}

// IntoAnyElement returns the inner element's AnyElement (M0-div-paint ports it).
func (s *Stateful[E]) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented Stateful.IntoAnyElement")
}

// ClickEvent is a mouse click on an element (M0-input-types ports it).
type ClickEvent struct{}

// PlatformInput is an input event from the platform (M0-input-types ports it).
type PlatformInput interface {
	isPlatformInput()
}

// Modifiers is the state of the modifier keys (M0-input-types ports it).
type Modifiers struct {
	// Control is the control key.
	Control bool
	// Alt is the alt key, Option on macOS.
	Alt bool
	// Shift is the shift key.
	Shift bool
	// Platform is the command key on macOS, the windows key on Windows, the super key on Linux.
	Platform bool
	// Function is the function key.
	Function bool
}

// Keystroke is a key press with its modifiers (M0-input-types ports it).
type Keystroke struct {
	// Modifiers is the modifier state of the keystroke.
	Modifiers Modifiers
	// Key is the key that was pressed.
	Key string
	// KeyChar is the character the key would produce, if any.
	KeyChar Option[string]
}

// KeystrokeParse parses a keystroke from its source form (M0-input-types ports it).
func KeystrokeParse(source string) (Keystroke, error) {
	panic("gpui: unimplemented KeystrokeParse")
}

// Action is an action that can be bound to keystrokes and dispatched (M0-input-types ports it).
type Action interface {
	Name() string
	BoxedClone() Action
	PartialEq(other Action) bool
}

// ActionData is the registration record of an action type (M0-input-types ports it).
type ActionData struct {
	// Name is the action's qualified name.
	Name string
	// Type is the action's Go type.
	Type reflect.Type
	// Build builds the action from its JSON value.
	Build func(value jsontext.Value) (Action, error)
	// JSONSchema returns the JSON schema of the action's value.
	JSONSchema func() *jsonschema.Schema
	// DeprecatedAliases are other names the action can be built from.
	DeprecatedAliases []string
	// DeprecationMessage is shown when the action is used.
	DeprecationMessage string
	// Documentation is the action's documentation.
	Documentation string
}

// RegisterAction registers an action type; call it from init (M0-input-types ports it).
func RegisterAction(data ActionData) {
	panic("gpui: unimplemented RegisterAction")
}

// Scene is the list of primitives painted in one frame (M0-scene ports it).
type Scene struct{}

// PrimitiveBatch is a batch of primitives of one kind (M0-scene ports it).
type PrimitiveBatch interface {
	isBatch()
}

// Batches returns the scene's primitives in paint order (M0-scene ports it).
func (s *Scene) Batches() iter.Seq[PrimitiveBatch] {
	panic("gpui: unimplemented Scene.Batches")
}

// TextSystem is the application's text system (M0-text-system ports it).
type TextSystem struct{}

// WindowTextSystem is a window's view of the text system (M0-text-system ports it).
type WindowTextSystem struct{}

// Font is a font family with weight, style and features (M0-text-system ports it).
type Font struct{}

// FontRun is a run of text in one font (M0-text-system ports it).
type FontRun struct{}

// TextRun is a run of text with one style (M0-text-system ports it).
type TextRun struct{}

// FontMetrics are the metrics of a font (M0-text-system ports it).
type FontMetrics struct{}

// LineLayout is a shaped line of text (M0-text-system ports it).
type LineLayout struct{}

// ShapedRun is a run of shaped glyphs in one font (M0-text-system ports it).
type ShapedRun struct{}

// ShapedGlyph is one shaped glyph (M0-text-system ports it).
type ShapedGlyph struct{}

// WrappedLineLayout is a shaped line with wrap boundaries (M0-text-system ports it).
type WrappedLineLayout struct{}

// LineLayoutCache caches line layouts between frames (M0-text-system ports it).
type LineLayoutCache struct{}

// LineWrapper wraps lines of text (M0-line-wrapper ports it).
type LineWrapper struct{}

// Boundary is a wrap boundary in a line (M0-line-wrapper ports it).
type Boundary struct{}

// ShapedLine is a line ready to paint (M0-text-elements ports it).
type ShapedLine struct{}

// WrappedLine is a wrapped line ready to paint (M0-text-elements ports it).
type WrappedLine struct{}

// TestOption configures RunTest (M0-app-contexts ports it).
type TestOption struct {
	apply func(*testConfig)
}

type testConfig struct{}

// Seed runs the test with one seed (M0-app-contexts ports it).
func Seed(seed uint64) TestOption {
	panic("gpui: unimplemented Seed")
}

// Seeds runs the test once per seed (M0-app-contexts ports it).
func Seeds(seeds ...uint64) TestOption {
	panic("gpui: unimplemented Seeds")
}

// Iterations runs the test with seeds 0..n-1 (M0-app-contexts ports it).
func Iterations(n uint64) TestOption {
	panic("gpui: unimplemented Iterations")
}

// Retries reruns a failing test up to n times (M0-app-contexts ports it).
func Retries(n int) TestOption {
	panic("gpui: unimplemented Retries")
}

// OnFailure calls f when the test fails (M0-app-contexts ports it).
func OnFailure(f func()) TestOption {
	panic("gpui: unimplemented OnFailure")
}

// RunTest runs fn as a gpui test, once per seed (M0-app-contexts ports it).
func RunTest(t *testing.T, fn func(tc *TestRun), opts ...TestOption) {
	panic("gpui: unimplemented RunTest")
}

// TestRun is one seeded run of a gpui test (M0-app-contexts ports it).
type TestRun struct{}

// Cx returns the run's first TestAppContext (M0-app-contexts ports it).
func (tc *TestRun) Cx() *TestAppContext {
	panic("gpui: unimplemented TestRun.Cx")
}

// NewContext returns another TestAppContext on the same scheduler (M0-app-contexts ports it).
func (tc *TestRun) NewContext() *TestAppContext {
	panic("gpui: unimplemented TestRun.NewContext")
}

// Rand returns the run's seeded random number generator (M0-app-contexts ports it).
func (tc *TestRun) Rand() *rand.Rand {
	panic("gpui: unimplemented TestRun.Rand")
}

// BackgroundExecutor returns the run's background executor (M0-app-contexts ports it).
func (tc *TestRun) BackgroundExecutor() BackgroundExecutor {
	panic("gpui: unimplemented TestRun.BackgroundExecutor")
}

// TestAppContext is a test context with a TestPlatform (M0-test-contexts ports it).
type TestAppContext struct{}

// IntoAnyElement converts the string into a text element (M0-text-elements ports it).
func (s SharedString) IntoAnyElement() AnyElement {
	panic("gpui: unimplemented SharedString.IntoAnyElement")
}
