package gpui

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"weak"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
)

// EntityID is a unique identifier for an entity across the application.
type EntityID uint64

// AsNonZeroU64 converts this entity id to a non-zero uint64.
func (id EntityID) AsNonZeroU64() uint64 {
	if id == 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return uint64(id)
}

// AsU64 converts this entity id to a uint64.
func (id EntityID) AsU64() uint64 {
	return uint64(id)
}

// String formats the id as its uint64 value, upstream's Display.
func (id EntityID) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

func (id EntityID) idx() uint32 {
	return uint32(id)
}

func (id EntityID) version() uint32 {
	return uint32(id>>32) | 1
}

func (id EntityID) debugString() string {
	if id.idx() == math.MaxUint32 {
		return "EntityId(null)"
	}
	return fmt.Sprintf("EntityId(%dv%d)", id.idx(), id.version())
}

func newEntityIDFromKey(idx, version uint32) EntityID {
	return EntityID(uint64(version|1)<<32 | uint64(idx))
}

type entitySlot struct {
	version  uint32
	nextFree uint32
	value    *atomic.Uint64
}

type entitySlotMap struct {
	slots    []entitySlot
	freeHead uint32
}

func newEntitySlotMap() entitySlotMap {
	return entitySlotMap{slots: []entitySlot{{}}, freeHead: 1}
}

func (m *entitySlotMap) insert(value *atomic.Uint64) EntityID {
	if int(m.freeHead) < len(m.slots) {
		idx := m.freeHead
		slot := &m.slots[idx]
		occupiedVersion := slot.version | 1
		m.freeHead = slot.nextFree
		slot.value = value
		slot.version = occupiedVersion
		return newEntityIDFromKey(idx, occupiedVersion)
	}
	if len(m.slots) >= math.MaxUint32 {
		panic("SlotMap is full")
	}
	idx := uint32(len(m.slots))
	m.slots = append(m.slots, entitySlot{version: 1, value: value})
	m.freeHead = idx + 1
	return newEntityIDFromKey(idx, 1)
}

func (m *entitySlotMap) containsKey(id EntityID) bool {
	idx := id.idx()
	return int(idx) < len(m.slots) && m.slots[idx].version == id.version()
}

func (m *entitySlotMap) get(id EntityID) (*atomic.Uint64, bool) {
	if !m.containsKey(id) {
		return nil, false
	}
	return m.slots[id.idx()].value, true
}

func (m *entitySlotMap) remove(id EntityID) (*atomic.Uint64, bool) {
	if !m.containsKey(id) {
		return nil, false
	}
	idx := id.idx()
	slot := &m.slots[idx]
	value := slot.value
	slot.value = nil
	slot.nextFree = m.freeHead
	m.freeHead = idx
	slot.version++
	return value, true
}

type entityMap struct {
	entities         map[EntityID]any
	accessedEntities accessedEntities
	refCounts        *entityRefCounts
	weakRefCounts    weak.Pointer[entityRefCounts]
}

type accessedEntities struct {
	frame  collections.HashSet[EntityID]
	scopes []collections.HashSet[EntityID]
}

func (a *accessedEntities) innermost() collections.HashSet[EntityID] {
	if len(a.scopes) > 0 {
		return a.scopes[len(a.scopes)-1]
	}
	return a.frame
}

type entityRefCounts struct {
	mu               sync.RWMutex
	counts           entitySlotMap
	droppedEntityIDs []EntityID
	leakDetector     leakDetector
}

type leaseInner struct {
	entity  any
	cleanup runtime.Cleanup
}

type droppedEntity struct {
	entityID EntityID
	entity   any
}

func newEntityMap() *entityMap {
	refCounts := &entityRefCounts{
		counts: newEntitySlotMap(),
		leakDetector: leakDetector{
			nextHandleID:  0,
			entityHandles: map[EntityID]*entityLeakData{},
		},
	}
	return &entityMap{
		entities: map[EntityID]any{},
		accessedEntities: accessedEntities{
			frame: collections.HashSet[EntityID]{},
		},
		refCounts:     refCounts,
		weakRefCounts: weak.Make(refCounts),
	}
}

func (m *entityMap) refCountsDropHandle() *entityRefCounts {
	return m.refCounts
}

func (m *entityMap) leakDetectorSnapshot() LeakDetectorSnapshot {
	m.refCounts.mu.RLock()
	defer m.refCounts.mu.RUnlock()
	return m.refCounts.leakDetector.snapshot()
}

func (m *entityMap) assertNoNewLeaks(snapshot *LeakDetectorSnapshot) {
	m.refCounts.mu.RLock()
	msg := m.refCounts.leakDetector.noNewLeaksMessage(snapshot)
	m.refCounts.mu.RUnlock()
	if msg != "" {
		panic(msg)
	}
}

func (m *entityMap) report() {
	m.refCounts.mu.Lock()
	msg := m.refCounts.leakDetector.reportMessage()
	m.refCounts.mu.Unlock()
	if msg != "" {
		panic(msg)
	}
}

func (m *entityMap) reserve[T any]() slot[T] {
	return slot[T]{entityFromAny[T](m.reserveErased(reflect.TypeFor[T]()))}
}

func (m *entityMap) reserveErased(entityType reflect.Type) AnyEntity {
	count := new(atomic.Uint64)
	count.Store(1)
	m.refCounts.mu.Lock()
	id := m.refCounts.counts.insert(count)
	m.refCounts.mu.Unlock()
	return newAnyEntity(id, entityType, m.weakRefCounts, entityType.String())
}

func (m *entityMap) insert[T any](s slot[T], entity *T) Entity[T] {
	return entityFromAny[T](m.insertErased(s.AnyEntity, entity))
}

func (m *entityMap) insertErased(s AnyEntity, entity any) AnyEntity {
	id := s.EntityID()
	m.accessedEntities.innermost()[id] = struct{}{}
	m.entities[id] = entity
	return s
}

func (m *entityMap) lease[T any](pointer Entity[T]) lease[T] {
	return lease[T]{
		id:    pointer.EntityID(),
		inner: m.leaseErased(pointer.AnyEntity, reflect.TypeFor[T]().String()),
	}
}

func (m *entityMap) endLease[T any](l lease[T]) {
	m.endLeaseErased(l.id, l.inner)
}

func (m *entityMap) read[T any](entity Entity[T]) *T {
	m.assertValidContext(entity.AnyEntity)
	state, ok := m.readInner(entity.EntityID())
	if ok {
		if typed, ok := state.(*T); ok {
			return typed
		}
	}
	doubleLeasePanic("read", reflect.TypeFor[T]().String())
	return nil
}

func (m *entityMap) readErased(pointer AnyEntity) any {
	m.assertValidContext(pointer)
	state, ok := m.readInner(pointer.EntityID())
	if !ok {
		doubleLeasePanic("read", pointer.EntityType().String())
	}
	return state
}

func (m *entityMap) leaseErased(pointer AnyEntity, entityType string) *leaseInner {
	m.assertValidContext(pointer)
	entity, ok := m.leaseInner(pointer.EntityID())
	if !ok {
		doubleLeasePanic("update", entityType)
	}
	inner := &leaseInner{entity: entity}
	if currentLeakDetectionMode() == leakDetectionStrict {
		inner.cleanup = runtime.AddCleanup(inner, func(entityType string) {
			recordStrictLeak("Leases must be ended with EntityMap::end_lease (lease of " + entityType + " was collected)")
		}, entityType)
	}
	return inner
}

func (m *entityMap) endLeaseErased(entityID EntityID, lease *leaseInner) {
	entity := lease.entity
	if entity == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	lease.entity = nil
	lease.cleanup.Stop()
	m.endLeaseInner(entityID, entity)
}

func (m *entityMap) assertValidContext(entity AnyEntity) {
	if gpuiutil.DebugAssertions && entity.handle().entityMap != m.weakRefCounts {
		panic("used a entity with the wrong context")
	}
}

func (m *entityMap) extendAccessed(entities []EntityID) {
	m.refCounts.mu.RLock()
	defer m.refCounts.mu.RUnlock()
	innermost := m.accessedEntities.innermost()
	for _, id := range entities {
		if m.refCounts.counts.containsKey(id) {
			innermost[id] = struct{}{}
		}
	}
}

func (m *entityMap) clearAccessed() {
	clear(m.accessedEntities.scopes)
	m.accessedEntities.scopes = m.accessedEntities.scopes[:0]
	clear(m.accessedEntities.frame)
}

func (m *entityMap) pushAccessScope() {
	m.accessedEntities.scopes = append(m.accessedEntities.scopes, collections.HashSet[EntityID]{})
}

func (m *entityMap) popAccessScope() collections.HashSet[EntityID] {
	scopes := m.accessedEntities.scopes
	if len(scopes) == 0 {
		if gpuiutil.DebugAssertions {
			panic("popped an access scope that wasn't pushed")
		}
		return collections.HashSet[EntityID]{}
	}
	scope := scopes[len(scopes)-1]
	scopes[len(scopes)-1] = nil
	m.accessedEntities.scopes = scopes[:len(scopes)-1]
	innermost := m.accessedEntities.innermost()
	for id := range scope {
		innermost[id] = struct{}{}
	}
	return scope
}

func (m *entityMap) takeAccessed() collections.HashSet[EntityID] {
	if gpuiutil.DebugAssertions && len(m.accessedEntities.scopes) != 0 {
		panic("access scope left open")
	}
	frame := m.accessedEntities.frame
	m.accessedEntities.frame = collections.HashSet[EntityID]{}
	return frame
}

func (m *entityMap) restoreAccessed(entities collections.HashSet[EntityID]) {
	m.accessedEntities.frame = entities
}

func (m *entityMap) takeDropped() []droppedEntity {
	m.refCounts.mu.Lock()
	droppedEntityIDs := m.refCounts.droppedEntityIDs
	m.refCounts.droppedEntityIDs = nil
	var dropped []droppedEntity
	for _, entityID := range droppedEntityIDs {
		count, ok := m.refCounts.counts.remove(entityID)
		if !ok {
			m.refCounts.mu.Unlock()
			panic("called `Option::unwrap()` on a `None` value")
		}
		if gpuiutil.DebugAssertions && count.Load() != 0 {
			m.refCounts.mu.Unlock()
			panic(fmt.Sprintf("assertion `left == right` failed: dropped an entity that was referenced\n  left: %d\n right: 0", count.Load()))
		}
		delete(m.accessedEntities.frame, entityID)
		for _, scope := range m.accessedEntities.scopes {
			delete(scope, entityID)
		}
		entity, ok := m.entities[entityID]
		if !ok {
			continue
		}
		delete(m.entities, entityID)
		dropped = append(dropped, droppedEntity{entityID: entityID, entity: entity})
	}
	m.refCounts.mu.Unlock()
	return dropped
}

func (m *entityMap) readInner(entityID EntityID) (any, bool) {
	m.accessedEntities.innermost()[entityID] = struct{}{}
	entity, ok := m.entities[entityID]
	return entity, ok
}

func (m *entityMap) leaseInner(entityID EntityID) (any, bool) {
	m.accessedEntities.innermost()[entityID] = struct{}{}
	entity, ok := m.entities[entityID]
	if ok {
		delete(m.entities, entityID)
	}
	return entity, ok
}

func (m *entityMap) endLeaseInner(entityID EntityID, entity any) {
	m.entities[entityID] = entity
}

func doubleLeasePanic(operation, entityType string) {
	panic(fmt.Sprintf("cannot %s %s while it is already being updated", operation, entityType))
}

type lease[T any] struct {
	id    EntityID
	inner *leaseInner
}

func (l lease[T]) get() *T {
	return l.inner.entity.(*T)
}

type slot[T any] struct {
	Entity[T]
}

type entityHandle struct {
	entityID   EntityID
	entityType reflect.Type
	entityMap  weak.Pointer[entityRefCounts]
	handleID   handleID
	released   atomic.Bool
	cleanup    runtime.Cleanup
}

type entityHandleCleanup struct {
	entityID  EntityID
	entityMap weak.Pointer[entityRefCounts]
	handleID  handleID
	typeName  string
}

// AnyEntity is a dynamically typed strong handle to an entity, which can be downcast into an
// Entity[T]. Copying the value aliases one counted handle; Clone makes a new one and Release
// drops it. Two handles refer to the same entity when their EntityID values are equal.
type AnyEntity struct {
	h *entityHandle
}

func newAnyEntity(id EntityID, entityType reflect.Type, entityMap weak.Pointer[entityRefCounts], typeName string) AnyEntity {
	var hid handleID
	if currentLeakDetectionMode() != leakDetectionOff {
		if refCounts := entityMap.Value(); refCounts != nil {
			refCounts.mu.Lock()
			hid = refCounts.leakDetector.handleCreated(id, typeName)
			refCounts.mu.Unlock()
		}
	}
	h := &entityHandle{
		entityID:   id,
		entityType: entityType,
		entityMap:  entityMap,
		handleID:   hid,
	}
	h.cleanup = runtime.AddCleanup(h, collectEntityHandle, entityHandleCleanup{
		entityID:  id,
		entityMap: entityMap,
		handleID:  hid,
		typeName:  entityType.String(),
	})
	return AnyEntity{h: h}
}

func collectEntityHandle(c entityHandleCleanup) {
	if currentLeakDetectionMode() == leakDetectionStrict {
		var site string
		if refCounts := c.entityMap.Value(); refCounts != nil {
			refCounts.mu.RLock()
			site = refCounts.leakDetector.allocationSite(c.entityID, c.handleID)
			refCounts.mu.RUnlock()
		}
		recordStrictLeak(fmt.Sprintf("Handle for entity %s (%s) was collected without Release: %s", c.typeName, c.entityID.debugString(), site))
	}
	dropEntityHandle(c.entityID, c.entityMap, c.handleID)
}

func dropEntityHandle(entityID EntityID, entityMap weak.Pointer[entityRefCounts], hid handleID) {
	refCounts := entityMap.Value()
	if refCounts != nil {
		refCounts.mu.RLock()
		count, ok := refCounts.counts.get(entityID)
		if !ok {
			refCounts.mu.RUnlock()
			panic("detected over-release of a handle.")
		}
		prevCount := count.Add(math.MaxUint64) + 1
		refCounts.mu.RUnlock()
		if prevCount == 0 {
			panic("assertion `left != right` failed: Detected over-release of a entity.\n  left: 0\n right: 0")
		}
		if prevCount == 1 {
			refCounts.mu.Lock()
			refCounts.droppedEntityIDs = append(refCounts.droppedEntityIDs, entityID)
			refCounts.mu.Unlock()
		}
	}
	if currentLeakDetectionMode() != leakDetectionOff && refCounts != nil {
		refCounts.mu.Lock()
		refCounts.leakDetector.handleReleased(entityID, hid)
		refCounts.mu.Unlock()
	}
}

func (a AnyEntity) handle() *entityHandle {
	if a.h == nil {
		panic("gpui: use of a zero entity handle")
	}
	if a.h.released.Load() {
		panic("gpui: use of a released entity handle")
	}
	return a.h
}

// EntityID returns the id associated with this entity.
func (a AnyEntity) EntityID() EntityID {
	return a.handle().entityID
}

// EntityType returns the type of the entity's state, upstream's TypeId.
func (a AnyEntity) EntityType() reflect.Type {
	return a.handle().entityType
}

// Downgrade converts this entity handle into a weak variant, which does not prevent it from
// being released.
func (a AnyEntity) Downgrade() AnyWeakEntity {
	h := a.handle()
	return AnyWeakEntity{
		entityID:        h.entityID,
		entityType:      h.entityType,
		entityRefCounts: h.entityMap,
	}
}

// Downcast converts this handle into a strongly typed handle of type T, aliasing the same
// counted handle. It reports false, leaving the handle with the caller, if the entity is not a T.
func (a AnyEntity) Downcast[T any]() (Entity[T], bool) {
	if reflect.TypeFor[T]() == a.handle().entityType {
		return Entity[T]{AnyEntity: a}, true
	}
	return Entity[T]{}, false
}

// Clone returns a new counted handle to the same entity.
func (a AnyEntity) Clone() AnyEntity {
	h := a.handle()
	if refCounts := h.entityMap.Value(); refCounts != nil {
		refCounts.mu.RLock()
		count, ok := refCounts.counts.get(h.entityID)
		if !ok {
			refCounts.mu.RUnlock()
			panic("detected over-release of a entity")
		}
		prevCount := count.Add(1) - 1
		refCounts.mu.RUnlock()
		if prevCount == 0 {
			panic("assertion `left != right` failed: Detected over-release of a entity.\n  left: 0\n right: 0")
		}
	}
	return newAnyEntity(h.entityID, h.entityType, h.entityMap, "")
}

// Release drops this handle: the count decrements and the last release queues the entity for
// release at the next effect flush. It is idempotent and a no-op on a zero handle.
func (a AnyEntity) Release() {
	h := a.h
	if h == nil || !h.released.CompareAndSwap(false, true) {
		return
	}
	h.cleanup.Stop()
	dropEntityHandle(h.entityID, h.entityMap, h.handleID)
}

// Entity is a strong, well-typed handle to a struct which is managed by GPUI.
type Entity[T any] struct {
	AnyEntity
}

func newEntityHandle[T any](id EntityID, entityMap weak.Pointer[entityRefCounts]) Entity[T] {
	entityType := reflect.TypeFor[T]()
	return Entity[T]{AnyEntity: newAnyEntity(id, entityType, entityMap, entityType.String())}
}

func entityFromAny[T any](handle AnyEntity) Entity[T] {
	if handle.h != nil && handle.h.entityType != reflect.TypeFor[T]() {
		panic(fmt.Sprintf("gpui: entity of type %s used as %s", handle.h.entityType, reflect.TypeFor[T]()))
	}
	return Entity[T]{AnyEntity: handle}
}

// EntityID returns the entity id associated with this entity.
func (e Entity[T]) EntityID() EntityID {
	return e.AnyEntity.EntityID()
}

// Clone returns a new counted handle to the same entity.
func (e Entity[T]) Clone() Entity[T] {
	return Entity[T]{AnyEntity: e.AnyEntity.Clone()}
}

// Release drops this handle; see AnyEntity.Release.
func (e Entity[T]) Release() {
	e.AnyEntity.Release()
}

// Downgrade returns a non-retaining weak handle to this entity.
func (e Entity[T]) Downgrade() WeakEntity[T] {
	return WeakEntity[T]{AnyWeakEntity: e.AnyEntity.Downgrade()}
}

// IntoAny returns this handle as an AnyEntity; both values alias one counted handle.
func (e Entity[T]) IntoAny() AnyEntity {
	return e.AnyEntity
}

// Read returns a reference to this entity's state from the context.
func (e Entity[T]) Read(cx *App) *T {
	return cx.entities.read(e)
}

// ReadWith reads the entity referenced by this handle with the given function.
func (e Entity[T]) ReadWith[R any](cx AppContext, f func(entity *T, cx *App) R) R {
	return AppContextReadEntity(cx, e, f)
}

// Update updates the entity referenced by this handle with the given function.
func (e Entity[T]) Update(cx AppContext, update func(entity *T, cx *Context[T])) {
	AppContextUpdateEntity(cx, e, func(entity *T, cx *Context[T]) struct{} {
		update(entity, cx)
		return struct{}{}
	})
}

// UpdateR updates the entity referenced by this handle and returns the callback's result.
func (e Entity[T]) UpdateR[R any](cx AppContext, update func(entity *T, cx *Context[T]) R) R {
	return AppContextUpdateEntity(cx, e, update)
}

// AsMut leases the entity for mutable access; Release on the GpuiBorrow ends it.
func (e Entity[T]) AsMut(cx AppContext) GpuiBorrow[T] {
	return AppContextAsMut(cx, e)
}

// Write replaces the entity's state with value and notifies its observers.
func (e Entity[T]) Write(cx AppContext, value T) {
	e.Update(cx, func(entity *T, cx *Context[T]) {
		*entity = value
		cx.Notify()
	})
}

// UpdateIn updates the entity within a visual context that has a window. It returns an error
// if the window has been closed.
func (e Entity[T]) UpdateIn(cx VisualContext, update func(entity *T, window *Window, cx *Context[T])) error {
	_, err := VisualContextUpdateWindowEntity(cx, e, func(entity *T, window *Window, cx *Context[T]) struct{} {
		update(entity, window, cx)
		return struct{}{}
	})
	return err
}

// UpdateInR updates the entity within a visual context that has a window and returns the
// callback's result.
func (e Entity[T]) UpdateInR[R any](cx VisualContext, update func(entity *T, window *Window, cx *Context[T]) R) (R, error) {
	return VisualContextUpdateWindowEntity(cx, e, update)
}

// AnyWeakEntity is a type-erased weak reference to an entity. It is a plain value: copies
// need no release.
type AnyWeakEntity struct {
	entityID        EntityID
	entityType      reflect.Type
	entityRefCounts weak.Pointer[entityRefCounts]
}

// EntityID returns the entity id associated with this weak reference.
func (w AnyWeakEntity) EntityID() EntityID {
	return w.entityID
}

// IsUpgradable reports whether this weak handle can be upgraded, or if the entity has already
// been dropped.
func (w AnyWeakEntity) IsUpgradable() bool {
	refCounts := w.entityRefCounts.Value()
	if refCounts == nil {
		return false
	}
	refCounts.mu.RLock()
	defer refCounts.mu.RUnlock()
	count, ok := refCounts.counts.get(w.entityID)
	return ok && count.Load() > 0
}

// Upgrade returns a strong reference to the entity, or false if it has been released.
func (w AnyWeakEntity) Upgrade() (AnyEntity, bool) {
	refCounts := w.entityRefCounts.Value()
	if refCounts == nil {
		return AnyEntity{}, false
	}
	refCounts.mu.RLock()
	count, ok := refCounts.counts.get(w.entityID)
	if !ok {
		refCounts.mu.RUnlock()
		return AnyEntity{}, false
	}
	if atomicIncrIfNotZero(count) == 0 {
		refCounts.mu.RUnlock()
		return AnyEntity{}, false
	}
	refCounts.mu.RUnlock()
	return newAnyEntity(w.entityID, w.entityType, w.entityRefCounts, ""), true
}

// AssertReleased panics if any strong handle to the entity is still alive, or if the entity
// was released but its resources are retained until the end of the effect cycle.
func (w AnyWeakEntity) AssertReleased() {
	refCounts := w.entityRefCounts.Value()
	if currentLeakDetectionMode() != leakDetectionOff && refCounts != nil {
		refCounts.mu.Lock()
		msg := refCounts.leakDetector.assertReleasedMessage(w.entityID)
		refCounts.mu.Unlock()
		if msg != "" {
			panic(msg)
		}
	}
	if refCounts == nil {
		return
	}
	refCounts.mu.RLock()
	_, alive := refCounts.counts.get(w.entityID)
	refCounts.mu.RUnlock()
	if alive {
		panic("entity is still alive, or was recently dropped and its resources are retained " +
			"until the end of the effect cycle. Build with GPUI_LEAK_DETECTION=1 to list " +
			"its live handles.")
	}
}

var invalidEntityIDCounter atomic.Uint64

// NewAnyWeakEntityInvalid creates a weak entity that can never be upgraded.
func NewAnyWeakEntityInvalid() AnyWeakEntity {
	entityID := uint64(math.MaxUint64) - (invalidEntityIDCounter.Add(1) - 1)
	return AnyWeakEntity{
		entityID:   EntityID(entityID),
		entityType: reflect.TypeFor[struct{}](),
	}
}

// WeakEntity is a weak reference to an entity of type T.
type WeakEntity[T any] struct {
	AnyWeakEntity
}

// Upgrade returns a strong reference to the entity, or false if it has been released.
func (w WeakEntity[T]) Upgrade() (Entity[T], bool) {
	handle, ok := w.AnyWeakEntity.Upgrade()
	if !ok {
		return Entity[T]{}, false
	}
	return Entity[T]{AnyEntity: handle}, true
}

// Update updates the entity if it still exists. It returns an error if the entity has been
// released.
func (w WeakEntity[T]) Update(cx AppContext, update func(entity *T, cx *Context[T])) error {
	_, err := w.UpdateR(cx, func(entity *T, cx *Context[T]) struct{} {
		update(entity, cx)
		return struct{}{}
	})
	return err
}

// UpdateR updates the entity if it still exists and returns the callback's result. It returns
// an error if the entity has been released.
func (w WeakEntity[T]) UpdateR[R any](cx AppContext, update func(entity *T, cx *Context[T]) R) (R, error) {
	entity, ok := w.Upgrade()
	if !ok {
		var zero R
		return zero, errEntityReleased
	}
	defer entity.Release()
	return AppContextUpdateEntity(cx, entity, update), nil
}

// UpdateIn updates the entity within its current window if the entity still exists. It returns
// an error if the entity has been released or has no current window.
func (w WeakEntity[T]) UpdateIn(cx AppContext, update func(entity *T, window *Window, cx *Context[T])) error {
	_, err := w.UpdateInR(cx, func(entity *T, window *Window, cx *Context[T]) struct{} {
		update(entity, window, cx)
		return struct{}{}
	})
	return err
}

// UpdateInR updates the entity within its current window if the entity still exists and
// returns the callback's result.
func (w WeakEntity[T]) UpdateInR[R any](cx AppContext, update func(entity *T, window *Window, cx *Context[T]) R) (R, error) {
	entity, ok := w.Upgrade()
	if !ok {
		var zero R
		return zero, errEntityReleased
	}
	defer entity.Release()
	result, ok := AppContextWithWindow(cx, entity.EntityID(), func(window *Window, app *App) R {
		return AppContextUpdateEntity(app, entity, func(state *T, cx *Context[T]) R {
			return update(state, window, cx)
		})
	})
	if !ok {
		return result, errEntityHasNoCurrentWindow
	}
	return result, nil
}

// ReadWith reads the entity if it still exists. It returns an error if the entity has been
// released.
func (w WeakEntity[T]) ReadWith[R any](cx AppContext, read func(entity *T, cx *App) R) (R, error) {
	entity, ok := w.Upgrade()
	if !ok {
		var zero R
		return zero, errEntityReleased
	}
	defer entity.Release()
	return AppContextReadEntity(cx, entity, read), nil
}

// NewWeakEntityInvalid creates a weak entity that can never be upgraded.
func NewWeakEntityInvalid[T any]() WeakEntity[T] {
	return WeakEntity[T]{AnyWeakEntity: NewAnyWeakEntityInvalid()}
}

var (
	errEntityReleased           = errors.New("entity released")
	errEntityHasNoCurrentWindow = errors.New("entity has no current window")
)

type leakDetectionMode uint8

const (
	leakDetectionOff leakDetectionMode = iota
	leakDetectionOn
	leakDetectionStrict
)

var currentLeakDetectionMode = sync.OnceValue(func() leakDetectionMode {
	value, set := os.LookupEnv("GPUI_LEAK_DETECTION")
	switch {
	case value == "strict":
		return leakDetectionStrict
	case testing.Testing():
		if !set {
			return leakDetectionStrict
		}
		return leakDetectionOn
	case value != "" && value != "0":
		return leakDetectionOn
	}
	return leakDetectionOff
})

var leakBacktrace = sync.OnceValue(func() bool {
	return os.Getenv("LEAK_BACKTRACE") != ""
})

var strictLeaks struct {
	mu      sync.Mutex
	reports []string
}

func recordStrictLeak(report string) {
	strictLeaks.mu.Lock()
	strictLeaks.reports = append(strictLeaks.reports, report)
	strictLeaks.mu.Unlock()
}

func takeStrictLeakReports() []string {
	strictLeaks.mu.Lock()
	reports := strictLeaks.reports
	strictLeaks.reports = nil
	strictLeaks.mu.Unlock()
	return reports
}

type handleID struct {
	id uint64
}

type leakBacktraceFrames struct {
	pcs []uintptr
}

type entityLeakData struct {
	handles  map[handleID]*leakBacktraceFrames
	typeName string
}

type leakDetector struct {
	nextHandleID  uint64
	entityHandles map[EntityID]*entityLeakData
}

// LeakDetectorSnapshot is the set of entities that had live handles at a point in time.
type LeakDetectorSnapshot struct {
	entityIDs collections.HashSet[EntityID]
}

func captureLeakBacktrace(skip int) *leakBacktraceFrames {
	if !leakBacktrace() {
		return nil
	}
	pcs := make([]uintptr, 64)
	n := runtime.Callers(skip, pcs)
	return &leakBacktraceFrames{pcs: pcs[:n]}
}

func (d *leakDetector) handleCreated(entityID EntityID, typeName string) handleID {
	id := gpuiutil.PostInc(&d.nextHandleID)
	hid := handleID{id: id}
	data, ok := d.entityHandles[entityID]
	if !ok {
		if typeName == "" {
			typeName = "<unknown>"
		}
		data = &entityLeakData{handles: map[handleID]*leakBacktraceFrames{}, typeName: typeName}
		d.entityHandles[entityID] = data
	}
	data.handles[hid] = captureLeakBacktrace(2)
	return hid
}

func (d *leakDetector) handleReleased(entityID EntityID, hid handleID) {
	data, ok := d.entityHandles[entityID]
	if !ok {
		return
	}
	delete(data.handles, hid)
	if len(data.handles) == 0 {
		delete(d.entityHandles, entityID)
	}
}

func (d *leakDetector) allocationSite(entityID EntityID, hid handleID) string {
	if data, ok := d.entityHandles[entityID]; ok {
		if bt := data.handles[hid]; bt != nil {
			return "\n" + formatLeakBacktrace(bt.pcs)
		}
	}
	return "(export LEAK_BACKTRACE to find allocation site)"
}

func (d *leakDetector) assertReleased(entityID EntityID) {
	if msg := d.assertReleasedMessage(entityID); msg != "" {
		panic(msg)
	}
}

func (d *leakDetector) assertReleasedMessage(entityID EntityID) string {
	data, ok := d.entityHandles[entityID]
	if !ok {
		return ""
	}
	delete(d.entityHandles, entityID)
	var out strings.Builder
	for _, bt := range data.handles {
		if bt != nil {
			fmt.Fprintf(&out, "Leaked handle:\n%s\n", formatLeakBacktrace(bt.pcs))
		} else {
			out.WriteString("Leaked handle: (export LEAK_BACKTRACE to find allocation site)\n")
		}
	}
	return fmt.Sprintf("Handles for %s leaked:\n%s", data.typeName, out.String())
}

func (d *leakDetector) snapshot() LeakDetectorSnapshot {
	ids := make(collections.HashSet[EntityID], len(d.entityHandles))
	for id := range d.entityHandles {
		ids[id] = struct{}{}
	}
	return LeakDetectorSnapshot{entityIDs: ids}
}

func (d *leakDetector) assertNoNewLeaks(snapshot *LeakDetectorSnapshot) {
	if msg := d.noNewLeaksMessage(snapshot); msg != "" {
		panic(msg)
	}
}

func (d *leakDetector) noNewLeaksMessage(snapshot *LeakDetectorSnapshot) string {
	var out strings.Builder
	for entityID, data := range d.entityHandles {
		if _, ok := snapshot.entityIDs[entityID]; ok {
			continue
		}
		writeLeakedHandles(&out, entityID, data)
	}
	if out.Len() == 0 {
		return ""
	}
	return "New entity leaks detected since snapshot:\n" + out.String()
}

func writeLeakedHandles(out *strings.Builder, entityID EntityID, data *entityLeakData) {
	for _, bt := range data.handles {
		if bt != nil {
			fmt.Fprintf(out, "Leaked handle for entity %s (%s):\n%s\n", data.typeName, entityID.debugString(), formatLeakBacktrace(bt.pcs))
		} else {
			fmt.Fprintf(out, "Leaked handle for entity %s (%s): (export LEAK_BACKTRACE to find allocation site)\n", data.typeName, entityID.debugString())
		}
	}
}

func (d *leakDetector) report() {
	if msg := d.reportMessage(); msg != "" {
		panic(msg)
	}
}

func (d *leakDetector) reportMessage() string {
	strict := takeStrictLeakReports()
	if len(d.entityHandles) == 0 && len(strict) == 0 {
		return ""
	}
	var out strings.Builder
	for entityID, data := range d.entityHandles {
		writeLeakedHandles(&out, entityID, data)
	}
	clear(d.entityHandles)
	for _, report := range strict {
		out.WriteString(report)
		out.WriteString("\n")
	}
	return "Exited with leaked handles:\n" + out.String()
}

const leakDetectorHandleCreated = "github.com/megascan/gpui-go.(*leakDetector).handleCreated"

func formatLeakBacktrace(pcs []uintptr) string {
	cwd, cwdErr := os.Getwd()
	frames := runtime.CallersFrames(pcs)
	var out strings.Builder
	strip := false
	for {
		frame, more := frames.Next()
		switch frame.Function {
		case "testing.tRunner":
			strip = true
		case leakDetectorHandleCreated:
			strip = false
		case "main.main":
			strip = true
			writeLeakFrame(&out, frame, cwd, cwdErr)
		default:
			if !strip {
				writeLeakFrame(&out, frame, cwd, cwdErr)
			}
		}
		if !more {
			return out.String()
		}
	}
}

func writeLeakFrame(out *strings.Builder, frame runtime.Frame, cwd string, cwdErr error) {
	file := frame.File
	if cwdErr == nil {
		if rel, err := filepath.Rel(cwd, file); err == nil && !strings.HasPrefix(rel, "..") {
			file = rel
		}
	}
	fmt.Fprintf(out, "  %s\n      at %s:%d\n", frame.Function, file, frame.Line)
}
