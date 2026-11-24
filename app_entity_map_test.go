package gpui

import (
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type entityMapTestEntity struct {
	i int
}

func entityMapExpectPanic(t *testing.T, contains string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic containing %q", contains)
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, contains) {
			t.Fatalf("panic %q does not contain %q", msg, contains)
		}
	}()
	f()
}

func entityMapReport(t *testing.T, m *entityMap) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != nil {
			t.Fatalf("%v", r)
		}
	}()
	m.report()
}

func entityMapSet(ids ...EntityID) map[EntityID]struct{} {
	set := map[EntityID]struct{}{}
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func entityMapDroppedValues(dropped []droppedEntity) []int {
	values := make([]int, 0, len(dropped))
	for _, d := range dropped {
		values = append(values, d.entity.(*entityMapTestEntity).i)
	}
	return values
}

func entityMapWaitFor(cond func() bool) bool {
	for range 200 {
		runtime.GC()
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

func TestEntityMap_AccessedEntityScopes(t *testing.T) {
	entityMap := newEntityMap()
	slot := entityMap.reserve[entityMapTestEntity]()
	first := entityMap.insert(slot, &entityMapTestEntity{i: 1})
	slot = entityMap.reserve[entityMapTestEntity]()
	second := entityMap.insert(slot, &entityMapTestEntity{i: 2})
	entityMap.clearAccessed()

	entityMap.read(first)
	entityMap.pushAccessScope()
	entityMap.read(first)
	entityMap.pushAccessScope()
	entityMap.read(second)
	inner := entityMap.popAccessScope()
	outer := entityMap.popAccessScope()

	if !maps.Equal(inner, entityMapSet(second.EntityID())) {
		t.Fatalf("inner = %v", inner)
	}
	both := entityMapSet(first.EntityID(), second.EntityID())
	if !maps.Equal(outer, both) {
		t.Fatalf("outer = %v", outer)
	}
	if got := entityMap.takeAccessed(); !maps.Equal(got, both) {
		t.Fatalf("takeAccessed = %v", got)
	}

	second.Release()
	entityMap.takeDropped()
	entityMap.extendAccessed(slices.Collect(maps.Keys(both)))
	if got := entityMap.takeAccessed(); !maps.Equal(got, entityMapSet(first.EntityID())) {
		t.Fatalf("a released entity shouldn't be recorded again: %v", got)
	}

	first.Release()
	entityMapReport(t, entityMap)
}

func TestEntityMap_EntityMapSlotAssignmentBeforeCleanup(t *testing.T) {
	entityMap := newEntityMap()

	slot := entityMap.reserve[entityMapTestEntity]()
	entityMap.insert(slot, &entityMapTestEntity{i: 1}).Release()

	slot = entityMap.reserve[entityMapTestEntity]()
	entityMap.insert(slot, &entityMapTestEntity{i: 2}).Release()

	dropped := entityMap.takeDropped()
	if len(dropped) != 2 {
		t.Fatalf("len(dropped) = %d, want 2", len(dropped))
	}
	if got := entityMapDroppedValues(dropped); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("dropped = %v, want [1 2]", got)
	}
	entityMapReport(t, entityMap)
}

func TestEntityMap_EntityMapWeakUpgradeBeforeCleanup(t *testing.T) {
	entityMap := newEntityMap()

	slot := entityMap.reserve[entityMapTestEntity]()
	handle := entityMap.insert(slot, &entityMapTestEntity{i: 1})
	weak := handle.Downgrade()
	handle.Release()

	if strong, ok := weak.Upgrade(); ok {
		strong.Release()
		t.Fatal("upgraded a weak handle whose entity was dropped")
	}

	dropped := entityMap.takeDropped()
	if len(dropped) != 1 {
		t.Fatalf("len(dropped) = %d, want 1", len(dropped))
	}
	if got := entityMapDroppedValues(dropped); !slices.Equal(got, []int{1}) {
		t.Fatalf("dropped = %v, want [1]", got)
	}
	entityMapReport(t, entityMap)
}

func TestEntityMap_LeakDetectorSnapshotNoLeaks(t *testing.T) {
	entityMap := newEntityMap()

	slot := entityMap.reserve[entityMapTestEntity]()
	preExisting := entityMap.insert(slot, &entityMapTestEntity{i: 1})

	snapshot := entityMap.leakDetectorSnapshot()

	slot = entityMap.reserve[entityMapTestEntity]()
	temporary := entityMap.insert(slot, &entityMapTestEntity{i: 2})
	temporary.Release()

	entityMap.assertNoNewLeaks(&snapshot)

	preExisting.Release()
	entityMapReport(t, entityMap)
}

func TestEntityMap_LeakDetectorSnapshotDetectsNewLeak(t *testing.T) {
	entityMap := newEntityMap()

	slot := entityMap.reserve[entityMapTestEntity]()
	preExisting := entityMap.insert(slot, &entityMapTestEntity{i: 1})
	defer preExisting.Release()

	snapshot := entityMap.leakDetectorSnapshot()

	slot = entityMap.reserve[entityMapTestEntity]()
	leaked := entityMap.insert(slot, &entityMapTestEntity{i: 2})
	defer leaked.Release()

	entityMapExpectPanic(t, "New entity leaks detected since snapshot", func() {
		entityMap.assertNoNewLeaks(&snapshot)
	})
}

func TestEntityMap_EntityIDEncoding(t *testing.T) {
	entityMap := newEntityMap()
	first := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	id := first.EntityID()
	if id.AsU64() != 1<<32|1 || id.AsNonZeroU64() != id.AsU64() {
		t.Fatalf("first id = %d, want %d", id.AsU64(), uint64(1<<32|1))
	}
	if id.String() != "4294967297" || id.debugString() != "EntityId(1v1)" {
		t.Fatalf("id formats as %q / %q", id.String(), id.debugString())
	}
	first.Release()
	entityMap.takeDropped()

	reused := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 2})
	if got := reused.EntityID().debugString(); got != "EntityId(1v3)" {
		t.Fatalf("reused slot id = %s, want EntityId(1v3)", got)
	}
	if id == reused.EntityID() {
		t.Fatal("a reused slot must get a new id")
	}
	reused.Release()
	entityMap.takeDropped()
	entityMapReport(t, entityMap)
}

func TestEntityMap_ReleaseQueuesDroppedEntity(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 7})
	clone := handle.Clone()
	weak := handle.Downgrade()

	handle.Release()
	handle.Release()
	if got := entityMap.takeDropped(); len(got) != 0 {
		t.Fatalf("dropped %d entities while a clone was alive", len(got))
	}
	if !weak.IsUpgradable() {
		t.Fatal("entity should be upgradable while a clone is alive")
	}
	entityMapExpectPanic(t, "released entity handle", func() { handle.EntityID() })
	entityMapExpectPanic(t, "released entity handle", func() { handle.Clone() })

	clone.Release()
	if weak.IsUpgradable() {
		t.Fatal("entity should not be upgradable after its last release")
	}
	if got := entityMapDroppedValues(entityMap.takeDropped()); !slices.Equal(got, []int{7}) {
		t.Fatalf("dropped = %v, want [7]", got)
	}
	weak.AssertReleased()

	var zero Entity[entityMapTestEntity]
	zero.Release()
	entityMapExpectPanic(t, "zero entity handle", func() { zero.EntityID() })
	entityMapReport(t, entityMap)
}

func TestEntityMap_AssertReleasedListsLiveHandles(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	weak := handle.Downgrade()
	entityMapExpectPanic(t, "Handles for gpui.entityMapTestEntity leaked", func() { weak.AssertReleased() })
	handle.Release()
	entityMapExpectPanic(t, "entity is still alive, or was recently dropped", func() { weak.AssertReleased() })
	entityMap.takeDropped()
	weak.AssertReleased()
	entityMapReport(t, entityMap)
}

func TestEntityMap_ReportListsLeakedHandles(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	defer handle.Release()
	entityMapExpectPanic(t, "Exited with leaked handles:\nLeaked handle for entity gpui.entityMapTestEntity (EntityId(1v1))", func() {
		entityMap.report()
	})
}

func TestEntityMap_DoubleLeasePanics(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	defer handle.Release()

	l := entityMap.lease(handle)
	l.get().i = 2
	entityMapExpectPanic(t, "cannot read gpui.entityMapTestEntity while it is already being updated", func() {
		entityMap.read(handle)
	})
	entityMapExpectPanic(t, "cannot update gpui.entityMapTestEntity while it is already being updated", func() {
		entityMap.lease(handle)
	})
	entityMap.endLease(l)
	if got := entityMap.read(handle).i; got != 2 {
		t.Fatalf("i = %d after the lease, want 2", got)
	}
	entityMapExpectPanic(t, "called `Option::unwrap()` on a `None` value", func() { entityMap.endLease(l) })
}

func TestEntityMap_DowncastChecksType(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	anyHandle := handle.IntoAny()
	if _, ok := anyHandle.Downcast[int](); ok {
		t.Fatal("downcast to the wrong type succeeded")
	}
	typed, ok := anyHandle.Downcast[entityMapTestEntity]()
	if !ok || typed.EntityID() != handle.EntityID() {
		t.Fatal("downcast to the entity's type failed")
	}
	typed.Release()
	if invalid := NewWeakEntityInvalid[entityMapTestEntity](); invalid.IsUpgradable() {
		t.Fatal("an invalid weak entity is upgradable")
	} else if _, ok := invalid.Upgrade(); ok {
		t.Fatal("upgraded an invalid weak entity")
	}
	entityMapReport(t, entityMap)
}

//go:noinline
func entityMapDropCloneUnreleased(handle Entity[entityMapTestEntity]) {
	clone := handle.Clone()
	_ = clone.EntityID()
}

func TestEntityMap_CollectedHandleIsReleasedAndReported(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	takeStrictLeakReports()
	entityMapDropCloneUnreleased(handle)
	handle.Release()

	var dropped []droppedEntity
	if !entityMapWaitFor(func() bool {
		dropped = append(dropped, entityMap.takeDropped()...)
		return len(dropped) == 1
	}) {
		t.Fatal("the collected clone was never released")
	}
	if currentLeakDetectionMode() == leakDetectionStrict {
		reports := takeStrictLeakReports()
		if len(reports) != 1 || !strings.Contains(reports[0], "was collected without Release") {
			t.Fatalf("strict reports = %q", reports)
		}
	}
	entityMapReport(t, entityMap)
}

//go:noinline
func entityMapAbandonLease(entityMap *entityMap, handle Entity[entityMapTestEntity]) {
	l := entityMap.lease(handle)
	_ = l.get().i
}

func TestEntityMap_LeaseWithoutEndLeaseIsReported(t *testing.T) {
	if currentLeakDetectionMode() != leakDetectionStrict {
		t.Skip("needs GPUI_LEAK_DETECTION=strict")
	}
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	defer handle.Release()
	takeStrictLeakReports()
	entityMapAbandonLease(entityMap, handle)

	var reports []string
	if !entityMapWaitFor(func() bool {
		reports = append(reports, takeStrictLeakReports()...)
		return len(reports) > 0
	}) {
		t.Fatal("the abandoned lease was never reported")
	}
	if !strings.Contains(reports[0], "Leases must be ended with EntityMap::end_lease") {
		t.Fatalf("report = %q", reports[0])
	}
}

func TestEntityMap_ConcurrentCloneReleaseUpgrade(t *testing.T) {
	entityMap := newEntityMap()
	handle := entityMap.insert(entityMap.reserve[entityMapTestEntity](), &entityMapTestEntity{i: 1})
	weak := handle.Downgrade()

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				clone := handle.Clone()
				if strong, ok := weak.Upgrade(); ok {
					strong.Release()
				}
				clone.Release()
			}
		})
	}
	wg.Wait()

	if got := entityMap.takeDropped(); len(got) != 0 {
		t.Fatalf("dropped %d entities while the handle was alive", len(got))
	}
	handle.Release()
	if got := entityMapDroppedValues(entityMap.takeDropped()); !slices.Equal(got, []int{1}) {
		t.Fatalf("dropped = %v, want [1]", got)
	}
	entityMapReport(t, entityMap)
}

func TestEntityMap_LeakBacktraceFormat(t *testing.T) {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(1, pcs)
	out := formatLeakBacktrace(pcs[:n])
	if !strings.Contains(out, "TestEntityMap_LeakBacktraceFormat") || !strings.Contains(out, "at app_entity_map_test.go:") {
		t.Fatalf("backtrace misses the test frame:\n%s", out)
	}
	if strings.Contains(out, "testing.tRunner") {
		t.Fatalf("backtrace keeps frames from testing.tRunner on:\n%s", out)
	}
}
