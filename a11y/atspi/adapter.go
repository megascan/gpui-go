package atspi

import (
	"fmt"
	"math"
	"slices"
	"sync/atomic"
	"unicode/utf8"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

type adapterChangeHandler struct {
	adapter           *Adapter
	addedNodes        map[consumer.NodeID]struct{}
	removedNodes      map[consumer.NodeID]struct{}
	checkedTextChange map[consumer.NodeID]struct{}
	selectionChanged  map[consumer.NodeID]struct{}
	selectionOrder    []consumer.NodeID
}

func newAdapterChangeHandler(adapter *Adapter) *adapterChangeHandler {
	return &adapterChangeHandler{
		adapter:           adapter,
		addedNodes:        map[consumer.NodeID]struct{}{},
		removedNodes:      map[consumer.NodeID]struct{}{},
		checkedTextChange: map[consumer.NodeID]struct{}{},
		selectionChanged:  map[consumer.NodeID]struct{}{},
	}
}

func insertID(set map[consumer.NodeID]struct{}, id consumer.NodeID) bool {
	if _, ok := set[id]; ok {
		return false
	}
	set[id] = struct{}{}
	return true
}

func (h *adapterChangeHandler) addNode(node *consumer.Node) {
	id := node.ID()
	if !insertID(h.addedNodes, id) {
		return
	}

	role := node.Role()
	isRoot := node.IsRoot()
	wrapper := nodeWrapper{node}
	interfaces := wrapper.interfaces()
	h.adapter.registerInterfaces(node.ID(), interfaces)
	h.adapter.emitCacheAdded(node.ID())
	if isRoot && role == a11y.RoleWindow {
		appContext, done := h.adapter.context.readAppContext()
		adapterIndex, ok := appContext.adapterIndex(h.adapter.id)
		done()
		if !ok {
			panic("called `Result::unwrap()` on an `Err` value")
		}
		h.adapter.windowCreated(adapterIndex, node.ID())
	}

	live := wrapper.live()
	if live != PolitenessNone {
		if name, ok := wrapper.name(); ok {
			h.adapter.emitObjectEvent(node.ID(), ObjectEventAnnouncement{Message: name, Politeness: live})
		}
	}
	if selected, ok := node.IsSelected(); ok && selected {
		h.enqueueSelectionChangedIfNeeded(node)
	}
}

func (h *adapterChangeHandler) addSubtree(node *consumer.Node) {
	h.addNode(node)
	for child := range node.FilteredChildren(filter).All() {
		h.addSubtree(&child)
	}
}

func (h *adapterChangeHandler) removeNode(node *consumer.Node) {
	id := node.ID()
	if !insertID(h.removedNodes, id) {
		return
	}

	role := node.Role()
	isRoot := node.IsRoot()
	wrapper := nodeWrapper{node}
	if isRoot && role == a11y.RoleWindow {
		h.adapter.windowDestroyed(node.ID())
	}
	h.adapter.emitObjectEvent(node.ID(), ObjectEventStateChanged{State: StateDefunct, Value: true})
	h.adapter.emitCacheRemoved(node.ID())
	h.adapter.unregisterInterfaces(node.ID(), wrapper.interfaces())
	if selected, ok := node.IsSelected(); ok && selected {
		h.enqueueSelectionChangedIfNeeded(node)
	}
}

func (h *adapterChangeHandler) removeSubtree(node *consumer.Node) {
	for child := range node.FilteredChildren(filter).All() {
		h.removeSubtree(&child)
	}
	h.removeNode(node)
}

func (h *adapterChangeHandler) emitTextChangeIfNeededParent(oldNode, newNode *consumer.Node) {
	if !newNode.SupportsTextRanges() || !oldNode.SupportsTextRanges() {
		return
	}
	id := newNode.ID()
	if !insertID(h.checkedTextChange, id) {
		return
	}
	oldText := oldNode.DocumentRange().Text()
	newText := newNode.DocumentRange().Text()

	prefixUSVCount := int32(0)
	prefixByteCount := 0
	for {
		oldChar, oldSize := utf8.DecodeRuneInString(oldText[prefixByteCount:])
		newChar, newSize := utf8.DecodeRuneInString(newText[prefixByteCount:])
		if oldSize > 0 && newSize > 0 && oldChar == newChar {
			prefixUSVCount++
			prefixByteCount += newSize
			continue
		}
		if oldSize == 0 && newSize == 0 {
			return
		}
		break
	}

	oldRest := oldText[prefixByteCount:]
	newRest := newText[prefixByteCount:]
	suffixByteCount := 0
	for {
		oldChar, oldSize := utf8.DecodeLastRuneInString(oldRest[:len(oldRest)-suffixByteCount])
		newChar, newSize := utf8.DecodeLastRuneInString(newRest[:len(newRest)-suffixByteCount])
		if oldSize == 0 || newSize == 0 || oldChar != newChar || oldSize != newSize {
			break
		}
		suffixByteCount += oldSize
	}

	oldContent := oldText[prefixByteCount : len(oldText)-suffixByteCount]
	if length := utf8.RuneCountInString(oldContent); length <= math.MaxInt32 && length > 0 {
		h.adapter.emitObjectEvent(id, ObjectEventTextRemoved{
			StartIndex: prefixUSVCount,
			Length:     int32(length),
			Content:    oldContent,
		})
	}

	newContent := newText[prefixByteCount : len(newText)-suffixByteCount]
	if length := utf8.RuneCountInString(newContent); length <= math.MaxInt32 && length > 0 {
		h.adapter.emitObjectEvent(id, ObjectEventTextInserted{
			StartIndex: prefixUSVCount,
			Length:     int32(length),
			Content:    newContent,
		})
	}
}

func (h *adapterChangeHandler) emitTextChangeIfNeeded(oldNode, newNode *consumer.Node) {
	if role := newNode.Role(); role == a11y.RoleTextRun || role == a11y.RoleGenericContainer {
		oldParent, oldOK := oldNode.FilteredParent(filter)
		newParent, newOK := newNode.FilteredParent(filter)
		if oldOK && newOK {
			h.emitTextChangeIfNeededParent(&oldParent, &newParent)
		}
	} else {
		h.emitTextChangeIfNeededParent(oldNode, newNode)
	}
}

func toI32Offset(n int) (int32, bool) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, false
	}
	return int32(n), true
}

func rawSelectionEqual(a, b *consumer.Node) bool {
	aSel, aOK := a.RawTextSelection()
	bSel, bOK := b.RawTextSelection()
	return aOK == bOK && (!aOK || aSel == bSel)
}

func (h *adapterChangeHandler) emitTextSelectionChange(oldNode, newNode *consumer.Node) {
	if !newNode.SupportsTextRanges() {
		return
	}
	if oldNode == nil {
		if selection, ok := newNode.TextSelection(); ok && !selection.IsDegenerate() {
			h.adapter.emitObjectEvent(newNode.ID(), ObjectEventTextSelectionChanged{})
		}
		if selectionFocus, ok := newNode.TextSelectionFocus(); ok {
			if offset, ok := toI32Offset(selectionFocus.ToGlobalUsvIndex()); ok {
				h.adapter.emitObjectEvent(newNode.ID(), ObjectEventCaretMoved{Offset: offset})
			}
		}
		return
	}
	if !oldNode.IsFocused() || rawSelectionEqual(newNode, oldNode) {
		return
	}

	if selection, ok := newNode.TextSelection(); ok {
		oldNonDegenerate := false
		if oldSelection, ok := oldNode.TextSelection(); ok {
			oldNonDegenerate = !oldSelection.IsDegenerate()
		}
		if !selection.IsDegenerate() || oldNonDegenerate {
			h.adapter.emitObjectEvent(newNode.ID(), ObjectEventTextSelectionChanged{})
		}
	}

	oldSel, oldOK := oldNode.RawTextSelection()
	newSel, newOK := newNode.RawTextSelection()
	if oldOK != newOK || (oldOK && oldSel.Focus != newSel.Focus) {
		if selectionFocus, ok := newNode.TextSelectionFocus(); ok {
			if offset, ok := toI32Offset(selectionFocus.ToGlobalUsvIndex()); ok {
				h.adapter.emitObjectEvent(newNode.ID(), ObjectEventCaretMoved{Offset: offset})
			}
		}
	}
}

func (h *adapterChangeHandler) enqueueSelectionChangedIfNeededParent(node consumer.Node) {
	if !node.IsContainerWithSelectableChildren() {
		return
	}
	id := node.ID()
	if !insertID(h.selectionChanged, id) {
		return
	}
	h.selectionOrder = append(h.selectionOrder, id)
}

func (h *adapterChangeHandler) enqueueSelectionChangedIfNeeded(node *consumer.Node) {
	if !node.IsItemLike() {
		return
	}
	if container, ok := node.SelectionContainer(filter); ok {
		h.enqueueSelectionChangedIfNeededParent(container)
	}
}

func (h *adapterChangeHandler) emitSelectionChanged() {
	for _, id := range h.selectionOrder {
		if _, removed := h.removedNodes[id]; removed {
			continue
		}
		h.adapter.emitObjectEvent(id, ObjectEventSelectionChanged{})
	}
}

func (h *adapterChangeHandler) NodeAdded(node *consumer.Node) {
	if filter(node) == consumer.FilterResultInclude {
		h.addNode(node)
	}
}

func (h *adapterChangeHandler) NodeUpdated(oldNode, newNode *consumer.Node) {
	h.emitTextChangeIfNeeded(oldNode, newNode)
	filterOld := filter(oldNode)
	filterNew := filter(newNode)
	if filterNew != filterOld {
		if filterNew == consumer.FilterResultInclude {
			if filterOld == consumer.FilterResultExcludeSubtree {
				h.addSubtree(newNode)
			} else {
				h.addNode(newNode)
			}
		} else if filterOld == consumer.FilterResultInclude {
			if filterNew == consumer.FilterResultExcludeSubtree {
				h.removeSubtree(oldNode)
			} else {
				h.removeNode(oldNode)
			}
		}
	} else if filterNew == consumer.FilterResultInclude {
		oldWrapper := nodeWrapper{oldNode}
		newWrapper := nodeWrapper{newNode}
		oldInterfaces := oldWrapper.interfaces()
		newInterfaces := newWrapper.interfaces()
		keptInterfaces := oldInterfaces.Intersection(newInterfaces)
		h.adapter.unregisterInterfaces(newWrapper.id(), oldInterfaces.SymmetricDifference(keptInterfaces))
		h.adapter.registerInterfaces(newNode.ID(), newInterfaces.SymmetricDifference(keptInterfaces))
		bounds := h.adapter.context.readRootWindowBounds()
		newWrapper.notifyChanges(&bounds, h.adapter, oldWrapper)
		h.emitTextSelectionChange(oldNode, newNode)
		newSelected, newOK := newNode.IsSelected()
		oldSelected, oldOK := oldNode.IsSelected()
		if newOK != oldOK || newSelected != oldSelected {
			h.enqueueSelectionChangedIfNeeded(newNode)
		}
	}
}

func (h *adapterChangeHandler) FocusMoved(oldNode, newNode *consumer.Node) {
	if oldNode == nil && newNode != nil {
		if window, ok := rootWindow(newNode.TreeState); ok {
			h.adapter.windowActivated(nodeWrapper{&window})
		}
	} else if oldNode != nil && newNode == nil {
		if window, ok := rootWindow(oldNode.TreeState); ok {
			h.adapter.windowDeactivated(nodeWrapper{&window})
		}
	}
	if newNode != nil {
		h.adapter.emitObjectEvent(newNode.ID(), ObjectEventStateChanged{State: StateFocused, Value: true})
		h.emitTextSelectionChange(nil, newNode)
	}
	if oldNode != nil {
		h.adapter.emitObjectEvent(oldNode.ID(), ObjectEventStateChanged{State: StateFocused, Value: false})
	}
}

func (h *adapterChangeHandler) NodeRemoved(node *consumer.Node) {
	if filter(node) == consumer.FilterResultInclude {
		h.removeNode(node)
	}
}

var nextAdapterIDCounter atomic.Uint64

// NextAdapterID returns a new adapter ID; callers must ensure only one adapter at a time has a given ID.
func NextAdapterID() int {
	return int(nextAdapterIDCounter.Add(1) - 1)
}

// Adapter translates one accesskit tree into AT-SPI objects and events.
type Adapter struct {
	id       int
	callback AdapterCallback
	context  *Context
	released bool
}

// String formats the adapter like upstream's Debug impl.
func (a *Adapter) String() string {
	return fmt.Sprintf("Adapter { id: %d, callback: \"AdapterCallback\", context: %v }", a.id, a.context)
}

// NewAdapter creates an adapter with the next adapter ID.
func NewAdapter(appContext *AppContext, callback AdapterCallback, initialState a11y.TreeUpdate, isWindowFocused bool, rootWindowBounds WindowBounds, actionHandler a11y.ActionHandler) *Adapter {
	id := NextAdapterID()
	return NewAdapterWithID(id, appContext, callback, initialState, isWindowFocused, rootWindowBounds, actionHandler)
}

// NewAdapterWithID creates an adapter with the given ID.
func NewAdapterWithID(id int, appContext *AppContext, callback AdapterCallback, initialState a11y.TreeUpdate, isWindowFocused bool, rootWindowBounds WindowBounds, actionHandler a11y.ActionHandler) *Adapter {
	return NewAdapterWithWrappedActionHandler(id, appContext, callback, initialState, isWindowFocused, rootWindowBounds, NewActionHandlerWrapper(actionHandler))
}

// NewAdapterWithWrappedActionHandler creates an adapter that shares actionHandler.
func NewAdapterWithWrappedActionHandler(id int, appContext *AppContext, callback AdapterCallback, initialState a11y.TreeUpdate, isWindowFocused bool, rootWindowBounds WindowBounds, actionHandler ActionHandlerNoMut) *Adapter {
	tree := consumer.NewTree(initialState, isWindowFocused)
	focusID, hasFocus := tree.State().FocusID()
	context := newContext(appContext, tree, actionHandler, rootWindowBounds)
	app, done := context.writeAppContext()
	app.pushAdapter(id, context)
	done()
	adapter := &Adapter{id: id, callback: callback, context: context}
	adapter.registerTree()
	if hasFocus {
		adapter.emitObjectEvent(focusID, ObjectEventStateChanged{State: StateFocused, Value: true})
	}
	return adapter
}

type objectToAdd struct {
	id         consumer.NodeID
	interfaces InterfaceSet
}

func addChildren(node consumer.Node, toAdd *[]objectToAdd) {
	for child := range node.FilteredChildren(filter).All() {
		childID := child.ID()
		wrapper := nodeWrapper{&child}
		interfaces := wrapper.interfaces()
		*toAdd = append(*toAdd, objectToAdd{childID, interfaces})
		addChildren(child, toAdd)
	}
}

func (a *Adapter) registerTree() {
	var objectsToAdd []objectToAdd

	adapterIndex, rootID := func() (int, consumer.NodeID) {
		tree, doneTree := a.context.readTree()
		defer doneTree()
		treeState := tree.State()
		appContext, doneApp := a.context.writeAppContext()
		defer doneApp()
		appContext.toolkitName = someString(treeState.ToolkitName(), true)
		appContext.toolkitVersion = someString(treeState.ToolkitVersion())
		adapterIndex, ok := appContext.adapterIndex(a.id)
		if !ok {
			panic("called `Result::unwrap()` on an `Err` value")
		}
		root := treeState.Root()
		rootID := root.ID()
		wrapper := nodeWrapper{&root}
		objectsToAdd = append(objectsToAdd, objectToAdd{rootID, wrapper.interfaces()})
		addChildren(root, &objectsToAdd)
		return adapterIndex, rootID
	}()

	for _, object := range objectsToAdd {
		a.registerInterfaces(object.id, object.interfaces)
		if object.id == rootID {
			a.windowCreated(adapterIndex, object.id)
		}
	}
}

// PlatformNode returns the AT-SPI object for id.
func (a *Adapter) PlatformNode(id consumer.NodeID) PlatformNode {
	return newPlatformNode(a.context, a.id, id)
}

// RootID returns the ID of the tree's root.
func (a *Adapter) RootID() consumer.NodeID {
	tree, done := a.context.readTree()
	defer done()
	return tree.State().RootID()
}

// PlatformRoot returns the application object.
func (a *Adapter) PlatformRoot() PlatformRoot {
	return NewPlatformRoot(a.context.appContext)
}

func (a *Adapter) registerInterfaces(id consumer.NodeID, newInterfaces InterfaceSet) {
	a.callback.RegisterInterfaces(a, id, newInterfaces)
}

func (a *Adapter) unregisterInterfaces(id consumer.NodeID, oldInterfaces InterfaceSet) {
	a.callback.UnregisterInterfaces(a, id, oldInterfaces)
}

func (a *Adapter) emitObjectEvent(target consumer.NodeID, event ObjectEvent) {
	a.callback.EmitEvent(a, EventObject{Target: NodeIDOrRootNode(target), Event: event})
}

func (a *Adapter) emitRootObjectEvent(event ObjectEvent) {
	a.callback.EmitEvent(a, EventObject{Target: NodeIDOrRootRoot(), Event: event})
}

func (a *Adapter) emitCacheAdded(target consumer.NodeID) {
	a.callback.EmitEvent(a, EventCache{Event: CacheEventAdded{ID: target}})
}

func (a *Adapter) emitCacheRemoved(target consumer.NodeID) {
	a.callback.EmitEvent(a, EventCache{Event: CacheEventRemoved{ID: target}})
}

// SetRootWindowBounds replaces the window bounds used for coordinate conversion.
func (a *Adapter) SetRootWindowBounds(newBounds WindowBounds) {
	a.context.rootWindowBoundsLock.Lock()
	defer a.context.rootWindowBoundsLock.Unlock()
	a.context.rootWindowBounds = newBounds
}

// Update applies update to the tree and emits the resulting events.
func (a *Adapter) Update(update a11y.TreeUpdate) {
	handler := newAdapterChangeHandler(a)
	tree, done := a.context.writeTree()
	tree.UpdateAndProcessChanges(update, handler)
	done()
	handler.emitSelectionChanged()
}

// UpdateWindowFocusState records whether the host window is focused and emits the resulting events.
func (a *Adapter) UpdateWindowFocusState(isFocused bool) {
	handler := newAdapterChangeHandler(a)
	tree, done := a.context.writeTree()
	defer done()
	tree.UpdateHostFocusStateAndProcessChanges(isFocused, handler)
}

func (a *Adapter) windowCreated(adapterIndex int, window consumer.NodeID) {
	a.emitRootObjectEvent(ObjectEventChildAdded{Index: adapterIndex, Child: window})
}

func (a *Adapter) windowActivated(window nodeWrapper) {
	name, _ := window.name()
	a.callback.EmitEvent(a, EventWindow{Target: window.id(), Name: name, Event: WindowEventActivated})
	a.emitObjectEvent(window.id(), ObjectEventStateChanged{State: StateActive, Value: true})
	a.emitRootObjectEvent(ObjectEventActiveDescendantChanged{ID: window.id()})
}

func (a *Adapter) windowDeactivated(window nodeWrapper) {
	name, _ := window.name()
	a.callback.EmitEvent(a, EventWindow{Target: window.id(), Name: name, Event: WindowEventDeactivated})
	a.emitObjectEvent(window.id(), ObjectEventStateChanged{State: StateActive, Value: false})
}

func (a *Adapter) windowDestroyed(window consumer.NodeID) {
	a.emitRootObjectEvent(ObjectEventChildRemoved{Child: window})
}

// ID returns the adapter's ID.
func (a *Adapter) ID() int {
	return a.id
}

// IsWindowFocused reports whether the host window is focused.
func (a *Adapter) IsWindowFocused() bool {
	tree, done := a.context.readTree()
	defer done()
	return tree.State().IsHostFocused()
}

// RootWindowBounds returns the current window bounds.
func (a *Adapter) RootWindowBounds() WindowBounds {
	return a.context.readRootWindowBounds()
}

// WrappedActionHandler returns the shared action handler.
func (a *Adapter) WrappedActionHandler() ActionHandlerNoMut {
	return a.context.actionHandler
}

func rootWindow(currentState *consumer.State) (consumer.Node, bool) {
	windowRoles := []a11y.Role{a11y.RoleAlertDialog, a11y.RoleDialog, a11y.RoleWindow}
	root := currentState.Root()
	if slices.Contains(windowRoles, root.Role()) {
		return root, true
	}
	return consumer.Node{}, false
}

// Release is upstream's Drop: it announces the window's removal and detaches the adapter from its AppContext. It is idempotent.
func (a *Adapter) Release() {
	if a.released {
		return
	}
	a.released = true
	rootID := a.RootID()
	a.windowDestroyed(rootID)
	appContext, done := a.context.writeAppContext()
	appContext.removeAdapter(a.id)
	done()
	a.context.released.Store(true)
}

func someString(value string, ok bool) gpuiutil.Option[string] {
	if ok {
		return gpuiutil.Some(value)
	}
	return gpuiutil.None[string]()
}
