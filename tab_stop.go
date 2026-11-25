package gpui

import (
	"cmp"
	"slices"

	"github.com/megascan/gpui-go/gpuiutil"
)

type tabStopMap struct {
	currentPath      tabStopPath
	insertionHistory []TabStopOperation
	byID             map[FocusID]int
	nodes            []tabStopNode
	sorted           bool
}

// TabStopOperation is one recorded operation of a tab stop map: TabStopOperationInsert,
// TabStopOperationGroup or TabStopOperationGroupEnd.
type TabStopOperation interface {
	isTabStopOperation()
}

// TabStopOperationInsert inserts a focus handle.
type TabStopOperationInsert struct {
	// FocusHandle is the inserted handle.
	FocusHandle FocusHandle
}

// TabStopOperationGroup begins a group at a tab index.
type TabStopOperationGroup struct {
	// TabIndex is the group's tab index.
	TabIndex int
}

// TabStopOperationGroupEnd ends the current group.
type TabStopOperationGroupEnd struct{}

func (TabStopOperationInsert) isTabStopOperation()   {}
func (TabStopOperationGroup) isTabStopOperation()    {}
func (TabStopOperationGroupEnd) isTabStopOperation() {}

func tabStopOperationFocusHandle(op TabStopOperation) (*FocusHandle, bool) {
	if insert, ok := op.(TabStopOperationInsert); ok {
		return &insert.FocusHandle, true
	}
	return nil, false
}

type tabStopPath []int

type tabStopNode struct {
	path               tabStopPath
	nodeInsertionIndex int
	tabStop            bool
}

func compareTabStopNodes(a, b tabStopNode) int {
	return cmp.Or(slices.Compare(a.path, b.path), cmp.Compare(a.nodeInsertionIndex, b.nodeInsertionIndex))
}

func (m *tabStopMap) insert(focusHandle *FocusHandle) {
	m.insertionHistory = append(m.insertionHistory, TabStopOperationInsert{FocusHandle: focusHandle.Clone()})
	path := append(slices.Clone(m.currentPath), focusHandle.TabIndex)
	if m.byID == nil {
		m.byID = make(map[FocusID]int)
	}
	m.byID[focusHandle.id] = len(m.nodes)
	m.sorted = false
	m.nodes = append(m.nodes, tabStopNode{
		nodeInsertionIndex: len(m.insertionHistory) - 1,
		tabStop:            focusHandle.TabStop,
		path:               path,
	})
}

func (m *tabStopMap) beginGroup(tabIndex int) {
	m.insertionHistory = append(m.insertionHistory, TabStopOperationGroup{TabIndex: tabIndex})
	m.currentPath = append(m.currentPath, tabIndex)
}

func (m *tabStopMap) endGroup() {
	m.insertionHistory = append(m.insertionHistory, TabStopOperationGroupEnd{})
	if len(m.currentPath) != 0 {
		m.currentPath = m.currentPath[:len(m.currentPath)-1]
	}
}

func (m *tabStopMap) clear() {
	m.currentPath = m.currentPath[:0]
	for _, op := range m.insertionHistory {
		if handle, ok := tabStopOperationFocusHandle(op); ok {
			handle.Release()
		}
	}
	clear(m.insertionHistory)
	m.insertionHistory = m.insertionHistory[:0]
	clear(m.byID)
	clear(m.nodes)
	m.nodes = m.nodes[:0]
	m.sorted = false
}

func (m *tabStopMap) next(focusedID Option[FocusID]) (FocusHandle, bool) {
	m.ensureSorted()
	start := 0
	if id, ok := focusedID.Get(); ok {
		if position, ok := m.byID[id]; ok {
			start = position + 1
		}
	}
	for i := 0; i < len(m.nodes); i++ {
		node := &m.nodes[(start+i)%len(m.nodes)]
		if node.tabStop {
			return m.focusHandleForOrder(node)
		}
	}
	return FocusHandle{}, false
}

func (m *tabStopMap) prev(focusedID Option[FocusID]) (FocusHandle, bool) {
	m.ensureSorted()
	end := len(m.nodes)
	if id, ok := focusedID.Get(); ok {
		if position, ok := m.byID[id]; ok {
			end = position
		}
	}
	for i := end - 1; i >= 0; i-- {
		if m.nodes[i].tabStop {
			return m.focusHandleForOrder(&m.nodes[i])
		}
	}
	for i := len(m.nodes) - 1; i >= end; i-- {
		if m.nodes[i].tabStop {
			return m.focusHandleForOrder(&m.nodes[i])
		}
	}
	return FocusHandle{}, false
}

func (m *tabStopMap) ensureSorted() {
	if m.sorted {
		return
	}
	slices.SortFunc(m.nodes, compareTabStopNodes)
	clear(m.byID)
	if m.byID == nil {
		m.byID = make(map[FocusID]int)
	}
	for position, node := range m.nodes {
		if node.nodeInsertionIndex >= len(m.insertionHistory) {
			continue
		}
		focusHandle, ok := tabStopOperationFocusHandle(m.insertionHistory[node.nodeInsertionIndex])
		if !ok {
			continue
		}
		latest, ok := m.byID[focusHandle.id]
		if !ok {
			latest = position
		}
		if m.nodes[latest].nodeInsertionIndex < node.nodeInsertionIndex {
			latest = position
		}
		m.byID[focusHandle.id] = latest
	}
	m.sorted = true
}

func (m *tabStopMap) replay(nodes []TabStopOperation) {
	for _, node := range nodes {
		switch node := node.(type) {
		case TabStopOperationInsert:
			m.insert(&node.FocusHandle)
		case TabStopOperationGroup:
			m.beginGroup(node.TabIndex)
		case TabStopOperationGroupEnd:
			m.endGroup()
		}
	}
}

func (m *tabStopMap) paintIndex() int {
	return len(m.insertionHistory)
}

func (m *tabStopMap) tabStopCount() int {
	count := 0
	for _, index := range m.byID {
		if index < len(m.nodes) && m.nodes[index].tabStop {
			count++
		}
	}
	return count
}

func (m *tabStopMap) focusHandleForOrder(order *tabStopNode) (FocusHandle, bool) {
	handle, ok := tabStopOperationFocusHandle(m.insertionHistory[order.nodeInsertionIndex])
	if gpuiutil.DebugAssertions && !ok {
		panic("The order node did not correspond to an element, this is a GPUI bug")
	}
	if !ok {
		return FocusHandle{}, false
	}
	return handle.Clone(), true
}
