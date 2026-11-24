package gpui

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"

	"github.com/megascan/gpui-go/gpuiutil"
)

type dispatchNodeID int

type dispatchTree struct {
	nodeStack        []dispatchNodeID
	contextStack     []KeyContext
	viewStack        []EntityID
	nodes            []dispatchNode
	focusableNodeIDs map[FocusID]dispatchNodeID
	viewNodeIDs      map[EntityID]dispatchNodeID
	keymap           *Keymap
	actionRegistry   *actionRegistry
}

type dispatchNode struct {
	keyListeners              []keyListener
	actionListeners           []dispatchActionListener
	modifiersChangedListeners []modifiersChangedListener
	context                   Option[KeyContext]
	focusID                   Option[FocusID]
	viewID                    Option[EntityID]
	parent                    Option[dispatchNodeID]
}

type reusedSubtree struct {
	oldRange       Range
	newRange       Range
	focusContained bool
}

func (r *reusedSubtree) refreshNodeID(nodeID dispatchNodeID) dispatchNodeID {
	if gpuiutil.DebugAssertions && !(r.oldRange.Start <= int(nodeID) && int(nodeID) < r.oldRange.End) {
		panic(fmt.Sprintf("node %d was not part of the reused subtree %d..%d", nodeID, r.oldRange.Start, r.oldRange.End))
	}
	return dispatchNodeID((int(nodeID) - r.oldRange.Start) + r.newRange.Start)
}

func (r *reusedSubtree) containsFocus() bool {
	return r.focusContained
}

type replay struct {
	keystroke Keystroke
	bindings  []KeyBinding
}

type dispatchResult struct {
	pending           []Keystroke
	pendingHasBinding bool
	bindings          []KeyBinding
	toReplay          []replay
	contextStack      []KeyContext
}

type keyListener func(event any, phase DispatchPhase, window *Window, cx *App)

type modifiersChangedListener func(event *ModifiersChangedEvent, window *Window, cx *App)

type dispatchActionListener struct {
	actionType reflect.Type
	listener   func(action any, phase DispatchPhase, window *Window, cx *App)
}

func newDispatchTree(keymap *Keymap, actionRegistry *actionRegistry) *dispatchTree {
	return &dispatchTree{
		focusableNodeIDs: make(map[FocusID]dispatchNodeID),
		viewNodeIDs:      make(map[EntityID]dispatchNodeID),
		keymap:           keymap,
		actionRegistry:   actionRegistry,
	}
}

func (t *dispatchTree) clear() {
	t.nodeStack = t.nodeStack[:0]
	clear(t.contextStack)
	t.contextStack = t.contextStack[:0]
	t.viewStack = t.viewStack[:0]
	clear(t.nodes)
	t.nodes = t.nodes[:0]
	clear(t.focusableNodeIDs)
	clear(t.viewNodeIDs)
}

func (t *dispatchTree) len() int {
	return len(t.nodes)
}

func (t *dispatchTree) lastNode() Option[dispatchNodeID] {
	if len(t.nodeStack) == 0 {
		return None[dispatchNodeID]()
	}
	return Some(t.nodeStack[len(t.nodeStack)-1])
}

func (t *dispatchTree) pushNode() dispatchNodeID {
	parent := t.lastNode()
	nodeID := dispatchNodeID(len(t.nodes))

	t.nodes = append(t.nodes, dispatchNode{parent: parent})
	t.nodeStack = append(t.nodeStack, nodeID)
	return nodeID
}

func (t *dispatchTree) setActiveNode(nodeID dispatchNodeID) {
	nextNodeParent := t.nodes[nodeID].parent
	for t.lastNode() != nextNodeParent && len(t.nodeStack) != 0 {
		t.popNode()
	}

	if t.lastNode() == nextNodeParent {
		t.nodeStack = append(t.nodeStack, nodeID)
		activeNode := &t.nodes[nodeID]
		if viewID, ok := activeNode.viewID.Get(); ok {
			t.viewStack = append(t.viewStack, viewID)
		}
		if context, ok := activeNode.context.Get(); ok {
			t.contextStack = append(t.contextStack, context.Clone())
		}
	} else {
		if gpuiutil.DebugAssertions && len(t.nodeStack) != 0 {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: 0", len(t.nodeStack)))
		}

		currentNodeID := Some(nodeID)
		for {
			nodeID, ok := currentNodeID.Get()
			if !ok {
				break
			}
			node := &t.nodes[nodeID]
			if context, ok := node.context.Get(); ok {
				t.contextStack = append(t.contextStack, context.Clone())
			}
			if viewID, ok := node.viewID.Get(); ok {
				t.viewStack = append(t.viewStack, viewID)
			}
			t.nodeStack = append(t.nodeStack, nodeID)
			currentNodeID = node.parent
		}

		slices.Reverse(t.contextStack)
		slices.Reverse(t.viewStack)
		slices.Reverse(t.nodeStack)
	}
}

func (t *dispatchTree) setKeyContext(context KeyContext) {
	t.activeNode().context = Some(context.Clone())
	t.contextStack = append(t.contextStack, context)
}

func (t *dispatchTree) setFocusID(focusID FocusID) {
	nodeID := t.nodeStack[len(t.nodeStack)-1]
	t.nodes[nodeID].focusID = Some(focusID)
	t.focusableNodeIDs[focusID] = nodeID
}

func (t *dispatchTree) setViewID(viewID EntityID) {
	if len(t.viewStack) == 0 || t.viewStack[len(t.viewStack)-1] != viewID {
		nodeID := t.nodeStack[len(t.nodeStack)-1]
		t.nodes[nodeID].viewID = Some(viewID)
		t.viewNodeIDs[viewID] = nodeID
		t.viewStack = append(t.viewStack, viewID)
	}
}

func (t *dispatchTree) popNode() {
	node := t.activeNode()
	if node.context.IsSome() {
		t.contextStack[len(t.contextStack)-1] = KeyContext{}
		t.contextStack = t.contextStack[:len(t.contextStack)-1]
	}
	if node.viewID.IsSome() {
		t.viewStack = t.viewStack[:len(t.viewStack)-1]
	}
	t.nodeStack = t.nodeStack[:len(t.nodeStack)-1]
}

func (t *dispatchTree) moveNode(source *dispatchNode) {
	t.pushNode()
	if context, ok := source.context.Get(); ok {
		t.setKeyContext(context.Clone())
	}
	if focusID, ok := source.focusID.Get(); ok {
		t.setFocusID(focusID)
	}
	if viewID, ok := source.viewID.Get(); ok {
		t.setViewID(viewID)
	}

	target := t.activeNode()
	target.keyListeners, source.keyListeners = source.keyListeners, nil
	target.actionListeners, source.actionListeners = source.actionListeners, nil
	target.modifiersChangedListeners, source.modifiersChangedListeners = source.modifiersChangedListeners, nil
}

func (t *dispatchTree) reuseSubtree(oldRange Range, source *dispatchTree, focus Option[FocusID]) reusedSubtree {
	oldLen := oldRange.End - oldRange.Start
	newRange := Range{Start: len(t.nodes), End: len(t.nodes) + oldLen}

	containsFocus := false
	var sourceStack []dispatchNodeID
	for i := oldRange.Start; i < oldRange.End && i < len(source.nodes); i++ {
		sourceNode := &source.nodes[i]
		sourceNodeID := dispatchNodeID(i)
		for len(sourceStack) != 0 {
			sourceAncestor := sourceStack[len(sourceStack)-1]
			if sourceNode.parent == Some(sourceAncestor) {
				break
			}
			sourceStack = sourceStack[:len(sourceStack)-1]
			t.popNode()
		}

		sourceStack = append(sourceStack, sourceNodeID)
		if sourceNode.focusID.IsSome() && sourceNode.focusID == focus {
			containsFocus = true
		}
		t.moveNode(sourceNode)
	}

	for len(sourceStack) != 0 {
		sourceStack = sourceStack[:len(sourceStack)-1]
		t.popNode()
	}

	return reusedSubtree{oldRange: oldRange, newRange: newRange, focusContained: containsFocus}
}

func (t *dispatchTree) truncate(index int) {
	for i := index; i < len(t.nodes); i++ {
		node := &t.nodes[i]
		if focusID, ok := node.focusID.Get(); ok {
			delete(t.focusableNodeIDs, focusID)
		}
		if viewID, ok := node.viewID.Get(); ok {
			delete(t.viewNodeIDs, viewID)
		}
	}
	if index < len(t.nodes) {
		clear(t.nodes[index:])
		t.nodes = t.nodes[:index]
	}
}

func (t *dispatchTree) onKeyEvent(listener keyListener) {
	node := t.activeNode()
	node.keyListeners = append(node.keyListeners, listener)
}

func (t *dispatchTree) onModifiersChanged(listener modifiersChangedListener) {
	node := t.activeNode()
	node.modifiersChangedListeners = append(node.modifiersChangedListeners, listener)
}

func (t *dispatchTree) onAction(actionType reflect.Type, listener func(action any, phase DispatchPhase, window *Window, cx *App)) {
	node := t.activeNode()
	node.actionListeners = append(node.actionListeners, dispatchActionListener{actionType: actionType, listener: listener})
}

func (t *dispatchTree) focusContains(parent FocusID, child FocusID) bool {
	if parent == child {
		return true
	}

	if parentNodeID, ok := t.focusableNodeIDs[parent]; ok {
		currentNodeID := None[dispatchNodeID]()
		if childNodeID, ok := t.focusableNodeIDs[child]; ok {
			currentNodeID = Some(childNodeID)
		}
		for {
			nodeID, ok := currentNodeID.Get()
			if !ok {
				break
			}
			if nodeID == parentNodeID {
				return true
			}
			currentNodeID = t.nodes[nodeID].parent
		}
	}
	return false
}

func compareActionTypes(a, b reflect.Type) int {
	return strings.Compare(a.PkgPath()+"."+a.String(), b.PkgPath()+"."+b.String())
}

func (t *dispatchTree) availableActions(target dispatchNodeID) []Action {
	var actions []Action
	for _, nodeID := range t.dispatchPath(target) {
		node := &t.nodes[nodeID]
		for _, listener := range node.actionListeners {
			ix, found := slices.BinarySearchFunc(actions, listener.actionType, func(a Action, actionType reflect.Type) int {
				return compareActionTypes(reflect.TypeOf(a), actionType)
			})
			if !found {
				action, err := t.actionRegistry.buildActionType(listener.actionType)
				if err == nil {
					actions = slices.Insert(actions, ix, action)
				}
			}
		}
	}
	return actions
}

func (t *dispatchTree) isActionAvailable(action Action, target dispatchNodeID) bool {
	actionType := reflect.TypeOf(action)
	for _, nodeID := range t.dispatchPath(target) {
		node := &t.nodes[nodeID]
		for _, listener := range node.actionListeners {
			if listener.actionType == actionType {
				return true
			}
		}
	}
	return false
}

func (t *dispatchTree) bindingsForAction(action Action, contextStack []KeyContext) []KeyBinding {
	var result []KeyBinding
	for _, binding := range t.keymap.BindingsForAction(action) {
		if bindingMatchesPredicateAndNotShadowed(t.keymap, binding, contextStack) {
			result = append(result, binding.Clone())
		}
	}
	return result
}

func (t *dispatchTree) highestPrecedenceBindingForAction(action Action, contextStack []KeyContext) (KeyBinding, bool) {
	bindings := t.keymap.BindingsForAction(action)
	for i := len(bindings) - 1; i >= 0; i-- {
		if bindingMatchesPredicateAndNotShadowed(t.keymap, bindings[i], contextStack) {
			return bindings[i].Clone(), true
		}
	}
	return KeyBinding{}, false
}

func bindingMatchesPredicateAndNotShadowed(keymap *Keymap, binding *KeyBinding, contextStack []KeyContext) bool {
	bindings, _ := keymap.BindingsForInput(binding.keystrokes, contextStack)
	if len(bindings) != 0 {
		return bindings[0].action.PartialEq(binding.action)
	}
	return false
}

func (t *dispatchTree) bindingsForInput(input []Keystroke, dispatchPath []dispatchNodeID) ([]KeyBinding, bool, []KeyContext) {
	var contextStack []KeyContext
	for _, nodeID := range dispatchPath {
		if context, ok := t.node(nodeID).context.Get(); ok {
			contextStack = append(contextStack, context.Clone())
		}
	}

	bindings, partial := t.keymap.BindingsForInput(input, contextStack)
	return bindings, partial, contextStack
}

func (t *dispatchTree) possibleNextBindingsForInput(input []Keystroke, contextStack []KeyContext) []KeyBinding {
	return t.keymap.PossibleNextBindingsForInput(input, contextStack)
}

func (t *dispatchTree) dispatchKey(input []Keystroke, keystroke Keystroke, dispatchPath []dispatchNodeID) dispatchResult {
	input = append(slices.Clip(input), keystroke)
	bindings, pending, contextStack := t.bindingsForInput(input, dispatchPath)

	if pending {
		return dispatchResult{pending: input, pendingHasBinding: len(bindings) != 0, contextStack: contextStack}
	} else if len(bindings) != 0 {
		return dispatchResult{bindings: bindings, contextStack: contextStack}
	} else if len(input) == 1 {
		return dispatchResult{contextStack: contextStack}
	}
	input = input[:len(input)-1]

	suffix, toReplay := t.replayPrefix(input, dispatchPath)

	result := t.dispatchKey(suffix, keystroke, dispatchPath)
	toReplay = append(toReplay, result.toReplay...)
	result.toReplay = toReplay
	return result
}

func (t *dispatchTree) flushDispatch(input []Keystroke, dispatchPath []dispatchNodeID) []replay {
	suffix, toReplay := t.replayPrefix(input, dispatchPath)

	if len(suffix) != 0 {
		toReplay = append(toReplay, t.flushDispatch(suffix, dispatchPath)...)
	}

	return toReplay
}

func (t *dispatchTree) replayPrefix(input []Keystroke, dispatchPath []dispatchNodeID) ([]Keystroke, []replay) {
	var toReplay []replay
	for last := len(input) - 1; last >= 0; last-- {
		bindings, _, _ := t.bindingsForInput(input[:last+1], dispatchPath)
		if len(bindings) != 0 {
			toReplay = append(toReplay, replay{keystroke: input[last], bindings: bindings})
			input = slices.Clone(input[last+1:])
			break
		}
	}
	if len(toReplay) == 0 {
		toReplay = append(toReplay, replay{keystroke: input[0]})
		input = slices.Clone(input[1:])
	}
	return input, toReplay
}

func (t *dispatchTree) dispatchPath(target dispatchNodeID) []dispatchNodeID {
	var dispatchPath []dispatchNodeID
	currentNodeID := Some(target)
	for {
		nodeID, ok := currentNodeID.Get()
		if !ok {
			break
		}
		dispatchPath = append(dispatchPath, nodeID)
		if int(nodeID) < len(t.nodes) {
			currentNodeID = t.nodes[nodeID].parent
		} else {
			currentNodeID = None[dispatchNodeID]()
		}
	}
	slices.Reverse(dispatchPath)
	return dispatchPath
}

func (t *dispatchTree) focusPath(focusID FocusID) []FocusID {
	var focusPath []FocusID
	currentNodeID := None[dispatchNodeID]()
	if nodeID, ok := t.focusableNodeIDs[focusID]; ok {
		currentNodeID = Some(nodeID)
	}
	for {
		nodeID, ok := currentNodeID.Get()
		if !ok {
			break
		}
		node := t.node(nodeID)
		if focusID, ok := node.focusID.Get(); ok {
			focusPath = append(focusPath, focusID)
		}
		currentNodeID = node.parent
	}
	slices.Reverse(focusPath)
	return focusPath
}

func (t *dispatchTree) viewPathReversed(viewID EntityID) iter.Seq[EntityID] {
	return func(yield func(EntityID) bool) {
		nodeID, ok := t.viewNodeIDs[viewID]
		if !ok {
			return
		}
		node := t.node(nodeID)
		for {
			if viewID, ok := node.viewID.Get(); ok {
				if !yield(viewID) {
					return
				}
			}
			parent, ok := node.parent.Get()
			if !ok {
				return
			}
			node = t.node(parent)
		}
	}
}

func (t *dispatchTree) node(nodeID dispatchNodeID) *dispatchNode {
	return &t.nodes[nodeID]
}

func (t *dispatchTree) activeNode() *dispatchNode {
	activeNodeID, ok := t.activeNodeID()
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return &t.nodes[activeNodeID]
}

func (t *dispatchTree) focusableNodeID(target FocusID) (dispatchNodeID, bool) {
	nodeID, ok := t.focusableNodeIDs[target]
	return nodeID, ok
}

func (t *dispatchTree) rootNodeID() dispatchNodeID {
	if gpuiutil.DebugAssertions && len(t.nodes) == 0 {
		panic("assertion failed: !self.nodes.is_empty()")
	}
	return 0
}

func (t *dispatchTree) activeNodeID() (dispatchNodeID, bool) {
	return t.lastNode().Get()
}
