package gpui

import (
	"slices"
	"testing"
)

type testFocusHandles struct {
	next FocusID
}

func (f *testFocusHandles) handle(tabStop bool, tabIndex int) FocusHandle {
	f.next++
	return FocusHandle{id: f.next, TabStop: tabStop, TabIndex: tabIndex}
}

func nextID(m *tabStopMap, focused Option[FocusID]) Option[FocusID] {
	if handle, ok := m.next(focused); ok {
		return Some(handle.id)
	}
	return None[FocusID]()
}

func prevID(m *tabStopMap, focused Option[FocusID]) Option[FocusID] {
	if handle, ok := m.prev(focused); ok {
		return Some(handle.id)
	}
	return None[FocusID]()
}

func TestTabStop_TabHandles(t *testing.T) {
	var focusMap testFocusHandles
	var tabIndexMap tabStopMap

	focusHandles := []FocusHandle{
		focusMap.handle(true, 0),
		focusMap.handle(true, 1),
		focusMap.handle(true, 1),
		focusMap.handle(false, 0),
		focusMap.handle(false, 2),
		focusMap.handle(true, 0),
		focusMap.handle(true, 2),
	}

	for i := range focusHandles {
		tabIndexMap.insert(&focusHandles[i])
	}
	expected := []FocusID{focusHandles[0].id, focusHandles[5].id, focusHandles[1].id, focusHandles[2].id, focusHandles[6].id}

	prev := None[FocusID]()
	var found []FocusID
	for range expected {
		id := nextID(&tabIndexMap, prev).Unwrap()
		prev = Some(id)
		found = append(found, id)
	}
	if !slices.Equal(found, expected) {
		t.Fatalf("found = %v, want %v", found, expected)
	}

	check := func(got Option[FocusID], want FocusID) {
		t.Helper()
		if got != Some(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	check(nextID(&tabIndexMap, None[FocusID]()), expected[0])
	check(prevID(&tabIndexMap, None[FocusID]()), expected[len(expected)-1])

	for i := range expected {
		check(nextID(&tabIndexMap, Some(expected[i])), expected[(i+1)%len(expected)])
	}

	check(prevID(&tabIndexMap, None[FocusID]()), expected[4])
	for i := range expected {
		check(prevID(&tabIndexMap, Some(expected[i])), expected[(i+len(expected)-1)%len(expected)])
	}
}

func TestTabStop_TabNonStopFiltering(t *testing.T) {
	var focusMap testFocusHandles
	var tabIndexMap tabStopMap

	tabNonStop1 := focusMap.handle(false, 1)
	tabStop2 := focusMap.handle(true, 2)
	tabIndexMap.insert(&tabNonStop1)
	tabIndexMap.insert(&tabStop2)
	if got := nextID(&tabIndexMap, Some(tabNonStop1.id)); got != Some(tabStop2.id) {
		t.Fatalf("got %v", got)
	}

	tabStop0 := focusMap.handle(true, 0)
	tabNonStop0 := focusMap.handle(false, 0)
	tabIndexMap.insert(&tabStop0)
	tabIndexMap.insert(&tabNonStop0)
	if got := nextID(&tabIndexMap, Some(tabStop0.id)); got != Some(tabStop2.id) {
		t.Fatalf("got %v", got)
	}
}

type tabStopMapTest struct {
	t        *testing.T
	tabMap   tabStopMap
	focusMap testFocusHandles
	expected []tabStopExpectation
}

type tabStopExpectation struct {
	order int
	id    FocusID
}

func newTabStopMapTest(t *testing.T) *tabStopMapTest {
	return &tabStopMapTest{t: t}
}

func (m *tabStopMapTest) tabNonStop(index int) *tabStopMapTest {
	handle := m.focusMap.handle(false, index)
	m.tabMap.insert(&handle)
	return m
}

func (m *tabStopMapTest) tabStop(index int, expected int) *tabStopMapTest {
	handle := m.focusMap.handle(true, index)
	m.tabMap.insert(&handle)
	m.expected = append(m.expected, tabStopExpectation{order: expected, id: handle.id})
	slices.SortStableFunc(m.expected, func(a, b tabStopExpectation) int { return a.order - b.order })
	return m
}

func (m *tabStopMapTest) tabGroup(tabIndex int, children func(*tabStopMapTest) *tabStopMapTest) *tabStopMapTest {
	m.tabMap.beginGroup(tabIndex)
	m = children(m)
	m.tabMap.endGroup()
	return m
}

func (m *tabStopMapTest) traverseTabMap(traverse func(*tabStopMap, Option[FocusID]) Option[FocusID]) []FocusID {
	lastFocusID := None[FocusID]()
	var found []FocusID
	for range m.expected {
		id := traverse(&m.tabMap, lastFocusID).Unwrap()
		lastFocusID = Some(id)
		found = append(found, id)
	}
	return found
}

func (m *tabStopMapTest) assert() {
	t := m.t
	t.Helper()
	var expected []FocusID
	for _, e := range m.expected {
		expected = append(expected, e.id)
	}

	forwardFound := m.traverseTabMap(nextID)
	if !slices.Equal(forwardFound, expected) {
		t.Fatalf("forward = %v, want %v", forwardFound, expected)
	}

	if got := nextID(&m.tabMap, Some(forwardFound[len(forwardFound)-1])); got != Some(expected[0]) {
		t.Fatalf("overflow next = %v, want %v", got, expected[0])
	}

	reversedFound := m.traverseTabMap(prevID)
	slices.Reverse(expected)
	if !slices.Equal(reversedFound, expected) {
		t.Fatalf("reversed = %v, want %v", reversedFound, expected)
	}

	if got := prevID(&m.tabMap, Some(reversedFound[len(reversedFound)-1])); got != Some(expected[0]) {
		t.Fatalf("overflow prev = %v, want %v", got, expected[0])
	}
}

func TestTabStop_InsertAfterSorting(t *testing.T) {
	var focusMap testFocusHandles
	var tabIndexMap tabStopMap
	second := focusMap.handle(true, 1)
	first := focusMap.handle(true, 0)

	tabIndexMap.insert(&second)
	if got := nextID(&tabIndexMap, None[FocusID]()); got != Some(second.id) {
		t.Fatalf("got %v", got)
	}

	tabIndexMap.insert(&first)
	if got := nextID(&tabIndexMap, None[FocusID]()); got != Some(first.id) {
		t.Fatalf("got %v", got)
	}
	if got := nextID(&tabIndexMap, Some(first.id)); got != Some(second.id) {
		t.Fatalf("got %v", got)
	}
	if got := prevID(&tabIndexMap, Some(first.id)); got != Some(second.id) {
		t.Fatalf("got %v", got)
	}
}

func TestTabStop_WithDisabledTabStop(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(0, 0).
		tabNonStop(1).
		tabStop(2, 1).
		tabStop(3, 2).
		assert()
}

func TestTabStop_WithMultipleDisabledTabStops(t *testing.T) {
	newTabStopMapTest(t).
		tabNonStop(0).
		tabStop(1, 0).
		tabNonStop(3).
		tabStop(3, 1).
		tabNonStop(4).
		assert()
}

func TestTabStop_TabGroupFunctionality(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(0, 0).
		tabStop(0, 1).
		tabGroup(2, func(t *tabStopMapTest) *tabStopMapTest { return t.tabStop(0, 2).tabStop(1, 3) }).
		tabStop(3, 4).
		tabStop(4, 5).
		assert()
}

func TestTabStop_SiblingGroups(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(0, 0).
		tabStop(1, 1).
		tabGroup(2, func(t *tabStopMapTest) *tabStopMapTest { return t.tabStop(0, 2).tabStop(1, 3) }).
		tabStop(3, 4).
		tabStop(4, 5).
		tabGroup(6, func(t *tabStopMapTest) *tabStopMapTest { return t.tabStop(0, 6).tabStop(1, 7) }).
		tabStop(7, 8).
		tabStop(8, 9).
		assert()
}

func TestTabStop_NestedGroup(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(0, 0).
		tabStop(1, 1).
		tabGroup(2, func(t *tabStopMapTest) *tabStopMapTest {
			return t.tabGroup(0, func(t *tabStopMapTest) *tabStopMapTest { return t.tabStop(0, 2).tabStop(1, 3) }).
				tabStop(1, 4)
		}).
		tabStop(3, 5).
		tabStop(4, 6).
		assert()
}

func TestTabStop_SiblingNestedGroups(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(0, 0).
		tabStop(1, 1).
		tabGroup(2, func(b *tabStopMapTest) *tabStopMapTest {
			return b.
				tabStop(0, 2).
				tabStop(2, 5).
				tabGroup(1, func(b *tabStopMapTest) *tabStopMapTest { return b.tabStop(0, 3).tabStop(1, 4) }).
				tabGroup(3, func(b *tabStopMapTest) *tabStopMapTest { return b.tabStop(0, 6).tabStop(1, 7) })
		}).
		tabStop(3, 8).
		tabStop(4, 9).
		assert()
}

func TestTabStop_SiblingNestedGroupsOutOfOrder(t *testing.T) {
	newTabStopMapTest(t).
		tabStop(9, 9).
		tabStop(8, 8).
		tabGroup(7, func(b *tabStopMapTest) *tabStopMapTest {
			return b.
				tabStop(0, 2).
				tabStop(2, 5).
				tabGroup(3, func(b *tabStopMapTest) *tabStopMapTest { return b.tabStop(1, 7).tabStop(0, 6) }).
				tabGroup(1, func(b *tabStopMapTest) *tabStopMapTest { return b.tabStop(0, 3).tabStop(1, 4) })
		}).
		tabStop(3, 0).
		tabStop(4, 1).
		assert()
}
