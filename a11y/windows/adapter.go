//go:build windows

package windows

import (
	"fmt"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
)

func focusEvent(context *Context, nodeID consumer.NodeID) queuedEvent {
	platformNode := context.getOrCreatePlatformNode(nodeID)
	element := platformNode.Interface(platformNodeSimple)
	return &queuedEventSimple{element: element, eventID: uiaAutomationFocusChangedEventID}
}

type adapterChangeHandler struct {
	context          *Context
	queue            []queuedEvent
	textChanged      map[consumer.NodeID]struct{}
	selectionChanged map[consumer.NodeID]*selectionChanges
}

func newAdapterChangeHandler(context *Context) *adapterChangeHandler {
	return &adapterChangeHandler{
		context:          context,
		textChanged:      map[consumer.NodeID]struct{}{},
		selectionChanged: map[consumer.NodeID]*selectionChanges{},
	}
}

func (h *adapterChangeHandler) insertTextChangeIfNeededParent(node consumer.Node) {
	if !node.SupportsTextRanges() {
		return
	}
	id := node.ID()
	if _, ok := h.textChanged[id]; ok {
		return
	}
	platformNode := h.context.getOrCreatePlatformNode(node.ID())
	element := platformNode.Interface(platformNodeSimple)
	h.queue = append([]queuedEvent{&queuedEventSimple{element: element, eventID: uiaTextTextChangedEventID}}, h.queue...)
	h.textChanged[id] = struct{}{}
}

func (h *adapterChangeHandler) insertTextChangeIfNeeded(node *consumer.Node) {
	if node.Role() != a11y.RoleTextRun {
		return
	}
	if parent, ok := node.FilteredParent(filter); ok {
		h.insertTextChangeIfNeededParent(parent)
	}
}

func (h *adapterChangeHandler) elementForPossiblyRemovedNode(treeState *consumer.TreeState, id consumer.NodeID) *com.IUnknown {
	if _, ok := treeState.NodeByID(id); ok {
		return h.context.getOrCreatePlatformNode(id).Interface(platformNodeSimple)
	}
	return newPlatformNode(h.context, id).Interface(platformNodeSimple)
}

func (h *adapterChangeHandler) handleSelectionStateChange(node *consumer.Node, isSelected bool) {
	key := node.ID()
	if container, ok := node.SelectionContainer(filter); ok {
		key = container.ID()
	}
	changes, ok := h.selectionChanged[key]
	if !ok {
		changes = &selectionChanges{
			addedItems:   map[consumer.NodeID]struct{}{},
			removedItems: map[consumer.NodeID]struct{}{},
		}
		h.selectionChanged[key] = changes
	}
	if isSelected {
		changes.addedItems[node.ID()] = struct{}{}
	} else {
		changes.removedItems[node.ID()] = struct{}{}
	}
}

const invalidateLimit = 20

func (h *adapterChangeHandler) pushIsSelectedChanged(element *com.IUnknown, oldValue, newValue bool) {
	h.queue = append(h.queue, &queuedEventPropertyChanged{
		element:    element,
		propertyID: uiaSelectionItemIsSelectedPropertyID,
		oldValue:   variantFromBool(oldValue).into(),
		newValue:   variantFromBool(newValue).into(),
	})
}

func (h *adapterChangeHandler) enqueueSelectionChanges(tree *consumer.Tree) {
	treeState := tree.State()
	for id, changes := range h.selectionChanged {
		node, ok := treeState.NodeByID(id)
		if !ok {
			continue
		}
		var container gpuiutil.Option[consumer.Node]
		var onlySelectedChild gpuiutil.Option[consumer.Node]
		if node.IsContainerWithSelectableChildren() {
			container = gpuiutil.Some(node)
			for child := range node.FilteredChildren(filter).All() {
				if selected, ok := child.IsSelected(); ok && selected {
					if !onlySelectedChild.IsSome() {
						onlySelectedChild = gpuiutil.Some(child)
					} else {
						onlySelectedChild = gpuiutil.None[consumer.Node]()
						break
					}
				}
			}
		}

		if child, ok := onlySelectedChild.Get(); ok {
			h.queue = append(h.queue, &queuedEventSimple{
				element: h.context.getOrCreatePlatformNode(child.ID()).Interface(platformNodeSimple),
				eventID: uiaSelectionItemElementSelectedEventID,
			})
			h.pushIsSelectedChanged(h.context.getOrCreatePlatformNode(child.ID()).Interface(platformNodeSimple), false, true)
			for childID := range changes.removedItems {
				element := h.elementForPossiblyRemovedNode(treeState, childID)
				h.pushIsSelectedChanged(element, true, false)
			}
			continue
		}
		c, hasContainer := container.Get()
		if hasContainer && len(changes.addedItems)+len(changes.removedItems) > invalidateLimit {
			platformNode := h.context.getOrCreatePlatformNode(c.ID())
			h.queue = append(h.queue, &queuedEventSimple{
				element: platformNode.Interface(platformNodeSimple),
				eventID: uiaSelectionInvalidatedEventID,
			})
			continue
		}
		containerIsMultiselectable := hasContainer && c.IsMultiselectable()
		for addedID := range changes.addedItems {
			eventID := uiaSelectionItemElementSelectedEventID
			if containerIsMultiselectable {
				eventID = uiaSelectionItemElementAddedToSelectionEventID
			}
			h.queue = append(h.queue, &queuedEventSimple{
				element: h.context.getOrCreatePlatformNode(addedID).Interface(platformNodeSimple),
				eventID: eventID,
			})
			h.pushIsSelectedChanged(h.context.getOrCreatePlatformNode(addedID).Interface(platformNodeSimple), false, true)
		}
		for removedID := range changes.removedItems {
			element := h.elementForPossiblyRemovedNode(treeState, removedID)
			element.AddRef()
			h.queue = append(h.queue, &queuedEventSimple{
				element: element,
				eventID: uiaSelectionItemElementRemovedFromSelectionEventID,
			})
			h.pushIsSelectedChanged(element, true, false)
		}
	}
}

type selectionChanges struct {
	addedItems   map[consumer.NodeID]struct{}
	removedItems map[consumer.NodeID]struct{}
}

func (h *adapterChangeHandler) pushSimple(id consumer.NodeID, eventID uiaEventID) {
	platformNode := h.context.getOrCreatePlatformNode(id)
	h.queue = append(h.queue, &queuedEventSimple{element: platformNode.Interface(platformNodeSimple), eventID: eventID})
}

// NodeAdded handles a node added by a tree update.
func (h *adapterChangeHandler) NodeAdded(node *consumer.Node) {
	h.insertTextChangeIfNeeded(node)
	if filter(node) != consumer.FilterResultInclude {
		return
	}
	buffer := acquireStringBuffer()
	defer buffer.release()
	wrapper := nodeWrapper{node: node, stringBuffer: &buffer.buf}
	if node.IsDialog() {
		h.pushSimple(node.ID(), uiaWindowWindowOpenedEventID)
	}
	if wrapper.name().ok && node.Live() != a11y.LiveOff {
		h.pushSimple(node.ID(), uiaLiveRegionChangedEventID)
	}
	if wrapper.isSelectionItemPatternSupported() && wrapper.isSelected() {
		h.handleSelectionStateChange(node, true)
	}
}

func sameRawValue(oldNode, newNode *consumer.Node) bool {
	oldValue, oldOK := oldNode.RawValue()
	newValue, newOK := newNode.RawValue()
	return oldOK == newOK && oldValue == newValue
}

// NodeUpdated handles a node changed by a tree update.
func (h *adapterChangeHandler) NodeUpdated(oldNode, newNode *consumer.Node) {
	if !sameRawValue(oldNode, newNode) {
		h.insertTextChangeIfNeeded(newNode)
	}
	oldNodeWasFilteredOut := filter(oldNode) != consumer.FilterResultInclude
	if filter(newNode) != consumer.FilterResultInclude {
		if !oldNodeWasFilteredOut {
			if oldNode.IsDialog() {
				h.pushSimple(oldNode.ID(), uiaWindowWindowClosedEventID)
			}
			oldBuffer := acquireStringBuffer()
			defer oldBuffer.release()
			oldWrapper := nodeWrapper{node: oldNode, stringBuffer: &oldBuffer.buf}
			if oldWrapper.isSelectionItemPatternSupported() && oldWrapper.isSelected() {
				h.handleSelectionStateChange(oldNode, false)
			}
		}
		return
	}
	platformNode := h.context.getOrCreatePlatformNode(newNode.ID())
	element := platformNode.Interface(platformNodeSimple)
	oldBuffer := acquireStringBuffer()
	defer oldBuffer.release()
	oldWrapper := nodeWrapper{node: oldNode, stringBuffer: &oldBuffer.buf}
	newBuffer := acquireStringBuffer()
	defer newBuffer.release()
	newWrapper := nodeWrapper{node: newNode, stringBuffer: &newBuffer.buf}
	newWrapper.enqueuePropertyChanges(&h.queue, h.context, element, &oldWrapper)
	newName := newWrapper.name()
	if newName.ok && newNode.Live() != a11y.LiveOff &&
		(newNode.Live() != oldNode.Live() || oldNodeWasFilteredOut || newName != oldWrapper.name()) {
		h.queue = append(h.queue, &queuedEventSimple{element: element, eventID: uiaLiveRegionChangedEventID})
	} else {
		element.Release()
	}
	if oldNodeWasFilteredOut && newNode.IsDialog() {
		h.pushSimple(newNode.ID(), uiaWindowWindowOpenedEventID)
	}
	if newWrapper.isSelectionItemPatternSupported() &&
		(newWrapper.isSelected() != oldWrapper.isSelected() || (oldNodeWasFilteredOut && newWrapper.isSelected())) {
		h.handleSelectionStateChange(newNode, newWrapper.isSelected())
	}
}

// FocusMoved handles a focus change made by a tree update.
func (h *adapterChangeHandler) FocusMoved(_, newNode *consumer.Node) {
	if newNode != nil {
		h.queue = append(h.queue, focusEvent(h.context, newNode.ID()))
	}
}

// NodeRemoved handles a node removed by a tree update.
func (h *adapterChangeHandler) NodeRemoved(node *consumer.Node) {
	h.insertTextChangeIfNeeded(node)
	h.context.removePlatformNode(node.ID())
	if filter(node) != consumer.FilterResultInclude {
		return
	}
	if node.IsDialog() {
		platformNode := newPlatformNode(h.context, node.ID())
		h.queue = append(h.queue, &queuedEventSimple{element: platformNode.Interface(platformNodeSimple), eventID: uiaWindowWindowClosedEventID})
	}
	buffer := acquireStringBuffer()
	defer buffer.release()
	wrapper := nodeWrapper{node: node, stringBuffer: &buffer.buf}
	if wrapper.isSelectionItemPatternSupported() {
		h.handleSelectionStateChange(node, false)
	}
}

const placeholderRootID a11y.NodeID = 0

type adapterStateKind uint8

const (
	adapterStateInactive adapterStateKind = iota
	adapterStatePlaceholder
	adapterStateActive
)

type adapterState struct {
	kind            adapterStateKind
	hwnd            WindowHandle
	isWindowFocused bool
	actionHandler   actionHandlerNoMut
	context         *Context
}

// String formats the state like upstream's Debug impl.
func (s *adapterState) String() string {
	switch s.kind {
	case adapterStateInactive:
		return fmt.Sprintf("Inactive { hwnd: WindowHandle(%#x), is_window_focused: %v, action_handler: \"ActionHandler\" }", uintptr(s.hwnd.HWND), s.isWindowFocused)
	case adapterStatePlaceholder:
		return fmt.Sprintf("Placeholder(%s)", s.context)
	default:
		return fmt.Sprintf("Active(%s)", s.context)
	}
}

// Adapter is the UI Automation adapter for one window.
type Adapter struct {
	state adapterState
}

// String formats the adapter like upstream's Debug impl.
func (a *Adapter) String() string {
	return fmt.Sprintf("Adapter { state: %s }", &a.state)
}

// NewAdapter creates a new Windows platform adapter.
//
// The action handler may or may not be called on the thread that owns the window.
// This must not be called while handling the WM_GETOBJECT message.
func NewAdapter(hwnd HWND, isWindowFocused bool, actionHandler a11y.ActionHandler) *Adapter {
	return newAdapterWithWrappedActionHandler(hwnd, isWindowFocused, newActionHandlerWrapper(actionHandler))
}

func newAdapterWithWrappedActionHandler(hwnd HWND, isWindowFocused bool, actionHandler actionHandlerNoMut) *Adapter {
	initUia()
	return &Adapter{state: adapterState{
		kind:            adapterStateInactive,
		hwnd:            WindowHandleFromHWND(hwnd),
		isWindowFocused: isWindowFocused,
		actionHandler:   actionHandler,
	}}
}

// UpdateIfActive calls updateFactory and applies the resulting update if and only if the
// tree has been initialized. If events are returned, the caller must call Raise on them.
func (a *Adapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) (*QueuedEvents, bool) {
	switch a.state.kind {
	case adapterStateInactive:
		return nil, false
	case adapterStatePlaceholder:
		context := a.state.context
		current, unlock := context.readTree()
		isWindowFocused := current.State().IsHostFocused()
		unlock()
		tree := consumer.NewTree(updateFactory(), isWindowFocused)
		context.treeLock.Lock()
		context.tree = tree
		context.treeLock.Unlock()
		context.isPlaceholder.Store(false)
		current, unlock = context.readTree()
		var result *QueuedEvents
		focus, ok := current.State().Focus()
		unlock()
		if ok {
			result = &QueuedEvents{events: []queuedEvent{focusEvent(context, focus.ID())}}
		}
		a.state = adapterState{kind: adapterStateActive, context: context}
		return result, ok
	default:
		context := a.state.context
		handler := newAdapterChangeHandler(context)
		tree, unlock := context.writeTree()
		defer unlock()
		tree.UpdateAndProcessChanges(updateFactory(), handler)
		handler.enqueueSelectionChanges(tree)
		return &QueuedEvents{events: handler.queue}, true
	}
}

// UpdateWindowFocusState updates the tree state based on whether the window is focused.
// If events are returned, the caller must call Raise on them.
func (a *Adapter) UpdateWindowFocusState(isFocused bool) (*QueuedEvents, bool) {
	switch a.state.kind {
	case adapterStateInactive:
		a.state.isWindowFocused = isFocused
		return nil, false
	default:
		context := a.state.context
		handler := newAdapterChangeHandler(context)
		tree, unlock := context.writeTree()
		defer unlock()
		tree.UpdateHostFocusStateAndProcessChanges(isFocused, handler)
		return &QueuedEvents{events: handler.queue}, true
	}
}

// HandleWmGetobject handles the WM_GETOBJECT window message, lazily initializing the tree
// with activationHandler. It returns false when the caller should pass the message to
// DefWindowProc; otherwise the caller must call the returned function, outside any lock it
// holds on the adapter, to produce the message's result.
func (a *Adapter) HandleWmGetobject(wparam WPARAM, lparam LPARAM, activationHandler a11y.ActivationHandler) (func() LRESULT, bool) {
	objid := normalizeObjid(lparam)
	if objid < 0 && objid != uiaRootObjectID && objid != objidClient {
		return nil, false
	}

	var hwnd WindowHandle
	var platformNode *com.Object
	switch a.state.kind {
	case adapterStateInactive:
		hwnd = a.state.hwnd
		if initialState, ok := activationHandler.RequestInitialTree(); ok {
			tree := consumer.NewTree(initialState, a.state.isWindowFocused)
			context := newContext(hwnd, tree, a.state.actionHandler, false)
			current, unlock := context.readTree()
			nodeID := current.State().RootID()
			unlock()
			platformNode = context.getOrCreatePlatformNode(nodeID)
			a.state = adapterState{kind: adapterStateActive, context: context}
		} else {
			root := a11y.NewNode(a11y.RoleWindow)
			placeholderUpdate := a11y.TreeUpdate{
				Nodes:  []a11y.NodeEntry{{ID: placeholderRootID, Node: root}},
				Tree:   gpuiutil.Some(a11y.NewTree(placeholderRootID)),
				TreeID: a11y.TreeIDRoot(),
				Focus:  placeholderRootID,
			}
			placeholderTree := consumer.NewTree(placeholderUpdate, a.state.isWindowFocused)
			context := newContext(hwnd, placeholderTree, a.state.actionHandler, true)
			platformNode = unspecifiedRootPlatformNode(context)
			a.state = adapterState{kind: adapterStatePlaceholder, context: context}
		}
	case adapterStatePlaceholder:
		hwnd = a.state.context.hwnd
		platformNode = unspecifiedRootPlatformNode(a.state.context)
	default:
		context := a.state.context
		current, unlock := context.readTree()
		nodeID := current.State().RootID()
		unlock()
		hwnd = context.hwnd
		platformNode = context.getOrCreatePlatformNode(nodeID)
	}
	result := wmGetObjectResult{hwnd: hwnd, wparam: wparam, lparam: lparam, el: platformNode.Interface(platformNodeSimple)}
	return result.lresult, true
}

// Release drops the adapter: its context stops resolving and releases its platform nodes.
func (a *Adapter) Release() {
	if a.state.context != nil {
		a.state.context.Release()
	}
}

func initUia() {
	uiaLookupID(automationIdentifierTypeProperty, &controlTypePropertyGUID)
}

func normalizeObjid(lparam LPARAM) int32 {
	return int32(uint32(uintptr(lparam) & 0xFFFFFFFF))
}

type wmGetObjectResult struct {
	hwnd   WindowHandle
	wparam WPARAM
	lparam LPARAM
	el     *com.IUnknown
}

func (r wmGetObjectResult) lresult() LRESULT {
	defer r.el.Release()
	return uiaReturnRawElementProvider(r.hwnd.HWND, r.wparam, r.lparam, r.el)
}

// QueuedEvents are the events generated by a tree update.
type QueuedEvents struct {
	events []queuedEvent
}

// Raise raises all queued events synchronously. The window may receive WM_GETOBJECT
// messages during this call, so no lock the WM_GETOBJECT handler needs may be held. It
// should be called on the thread that owns the window.
func (q *QueuedEvents) Raise() {
	events := q.events
	q.events = nil
	for i, event := range events {
		func() {
			defer event.release()
			var err error
			switch e := event.(type) {
			case *queuedEventSimple:
				err = uiaRaiseAutomationEvent(e.element, e.eventID)
			case *queuedEventPropertyChanged:
				err = uiaRaiseAutomationPropertyChangedEvent(e.element, e.propertyID, &e.oldValue, &e.newValue)
			}
			if err != nil {
				for _, rest := range events[i+1:] {
					rest.release()
				}
				panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
			}
		}()
	}
}
