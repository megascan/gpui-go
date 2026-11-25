//go:build darwin && !ios

package macos

import (
	"fmt"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const placeholderRootID a11y.NodeID = 0

type stateKind uint8

const (
	stateInactive stateKind = iota
	statePlaceholder
	stateActive
)

type state struct {
	kind               stateKind
	view               *weakID
	isViewFocused      bool
	actionHandler      actionHandlerNoMut
	placeholderContext *Context
	context            *Context
}

// String formats s like upstream's Debug impl.
func (s *state) String() string {
	switch s.kind {
	case stateInactive:
		return fmt.Sprintf("Inactive { view: %p, is_view_focused: %v }", s.view, s.isViewFocused)
	case statePlaceholder:
		return fmt.Sprintf("Placeholder { placeholder_context: %v, is_view_focused: %v }", s.placeholderContext, s.isViewFocused)
	}
	return fmt.Sprintf("Active { context: %v }", s.context)
}

type placeholderActionHandler struct{}

func (placeholderActionHandler) DoAction(a11y.ActionRequest) {}

// Adapter exposes an accessibility tree to NSAccessibility for one view.
type Adapter struct {
	state state
}

// String formats a like upstream's Debug impl.
func (a *Adapter) String() string {
	return fmt.Sprintf("Adapter { state: %v }", &a.state)
}

// NewAdapter creates a macOS adapter for view. It must be called on the main thread; the action
// handler is always called on the main thread.
func NewAdapter(view objc.ID, isViewFocused bool, actionHandler a11y.ActionHandler) *Adapter {
	if view == objc.Nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return &Adapter{state: state{
		kind:          stateInactive,
		view:          newWeakID(view),
		isViewFocused: isViewFocused,
		actionHandler: newActionHandlerWrapper(actionHandler),
	}}
}

// UpdateIfActive calls updateFactory and applies the resulting update if and only if the tree has
// been initialized. When events are returned, the caller must call Raise on them.
func (a *Adapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) (*QueuedEvents, bool) {
	switch a.state.kind {
	case stateInactive:
		return nil, false
	case statePlaceholder:
		placeholderContext := a.state.placeholderContext
		tree := consumer.NewTree(updateFactory(), a.state.isViewFocused)
		context := newContext(placeholderContext.view, tree, a.state.actionHandler)
		var result *QueuedEvents
		ok := false
		if node, focused := context.tree.State().Focus(); focused {
			result, ok = newQueuedEvents(context, []queuedEvent{focusEvent(node.ID())}), true
		}
		a.state = state{kind: stateActive, view: a.state.view, context: context}
		placeholderContext.Release()
		return result, ok
	}
	context := a.state.context
	eventGenerator := newEventGenerator(context)
	tree, unlock := context.writeTree()
	tree.UpdateAndProcessChanges(updateFactory(), eventGenerator)
	unlock()
	return eventGenerator.intoResult(), true
}

// UpdateViewFocusState updates the tree state based on whether the window is focused. When events
// are returned, the caller must call Raise on them.
func (a *Adapter) UpdateViewFocusState(isFocused bool) (*QueuedEvents, bool) {
	switch a.state.kind {
	case stateInactive, statePlaceholder:
		a.state.isViewFocused = isFocused
		return nil, false
	}
	context := a.state.context
	eventGenerator := newEventGenerator(context)
	tree, unlock := context.writeTree()
	tree.UpdateHostFocusStateAndProcessChanges(isFocused, eventGenerator)
	unlock()
	return eventGenerator.intoResult(), true
}

func (a *Adapter) getOrInitContext(activationHandler a11y.ActivationHandler) *Context {
	switch a.state.kind {
	case statePlaceholder:
		return a.state.placeholderContext
	case stateActive:
		return a.state.context
	}
	if initialState, ok := activationHandler.RequestInitialTree(); ok {
		tree := consumer.NewTree(initialState, a.state.isViewFocused)
		context := newContext(a.state.view, tree, a.state.actionHandler)
		a.state = state{kind: stateActive, view: a.state.view, context: context}
		return context
	}
	placeholderUpdate := a11y.TreeUpdate{
		Nodes:  []a11y.NodeEntry{{ID: placeholderRootID, Node: a11y.NewNode(a11y.RoleWindow)}},
		Tree:   gpuiutil.Some(a11y.NewTree(placeholderRootID)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  placeholderRootID,
	}
	placeholderTree := consumer.NewTree(placeholderUpdate, false)
	placeholderContext := newContext(a.state.view, placeholderTree, newActionHandlerWrapper(placeholderActionHandler{}))
	a.state = state{
		kind:               statePlaceholder,
		view:               a.state.view,
		placeholderContext: placeholderContext,
		isViewFocused:      a.state.isViewFocused,
		actionHandler:      a.state.actionHandler,
	}
	return placeholderContext
}

// ViewChildren returns the autoreleased NSArray of the view's accessibility children.
func (a *Adapter) ViewChildren(activationHandler a11y.ActivationHandler) objc.ID {
	context := a.getOrInitContext(activationHandler)
	tree, unlock := context.readTree()
	defer unlock()
	node := tree.State().Root()
	var platformNodes []objc.ID
	if filter(&node) == consumer.FilterResultInclude {
		platformNodes = []objc.ID{context.getOrCreatePlatformNode(node.ID())}
	} else {
		for child := range node.FilteredChildren(filter).All() {
			platformNodes = append(platformNodes, context.getOrCreatePlatformNode(child.ID()))
		}
	}
	return nsArrayAutoreleased(platformNodes)
}

// Focus returns the autoreleased platform node of the focused node, or nil.
func (a *Adapter) Focus(activationHandler a11y.ActivationHandler) objc.ID {
	context := a.getOrInitContext(activationHandler)
	tree, unlock := context.readTree()
	defer unlock()
	if node, ok := tree.State().Focus(); ok {
		if canBeFocused(&node) {
			return autoreleaseReturn(context.getOrCreatePlatformNode(node.ID()))
		}
	}
	return objc.Nil
}

func (a *Adapter) weakView() *weakID {
	return a.state.view
}

// HitTest returns the autoreleased platform node at point, in screen coordinates.
func (a *Adapter) HitTest(point objc.CGPoint, activationHandler a11y.ActivationHandler) objc.ID {
	view := a.weakView().load()
	if view == objc.Nil {
		return objc.Nil
	}
	defer objc.Release(view)
	context := a.getOrInitContext(activationHandler)
	tree, unlock := context.readTree()
	defer unlock()
	root := tree.State().Root()
	p := fromNSPoint(view, root, point)
	node, ok := root.NodeAtPoint(p, filter)
	if !ok {
		node = root
	}
	return autoreleaseReturn(context.getOrCreatePlatformNode(node.ID()))
}

// Release drops the adapter: the contexts it owns announce their platform nodes as destroyed and
// its weak reference to the view is cleared.
func (a *Adapter) Release() {
	switch a.state.kind {
	case statePlaceholder:
		a.state.placeholderContext.Release()
	case stateActive:
		a.state.context.Release()
	}
	a.state.view.destroy()
}
