//go:build darwin && !ios

package macos

import (
	"fmt"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type actionHandlerNoMut interface {
	DoAction(request a11y.ActionRequest)
}

type actionHandlerWrapper struct {
	mu    sync.Mutex
	inner a11y.ActionHandler
}

func newActionHandlerWrapper(inner a11y.ActionHandler) *actionHandlerWrapper {
	return &actionHandlerWrapper{inner: inner}
}

func (w *actionHandlerWrapper) DoAction(request a11y.ActionRequest) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inner.DoAction(request)
}

// Context is the state shared between an adapter and its platform nodes.
type Context struct {
	view              *weakID
	treeLock          sync.RWMutex
	tree              *consumer.Tree
	actionHandler     actionHandlerNoMut
	platformNodesLock sync.Mutex
	platformNodes     map[consumer.NodeID]objc.ID
	released          atomic.Bool
}

// String formats the context like upstream's Debug impl.
func (c *Context) String() string {
	return fmt.Sprintf("Context { view: %p, tree: %p, action_handler: \"ActionHandler\", platform_nodes: %d }", c.view, c.tree, len(c.platformNodes))
}

func newContext(view *weakID, tree *consumer.Tree, actionHandler actionHandlerNoMut) *Context {
	return &Context{
		view:          view,
		tree:          tree,
		actionHandler: actionHandler,
		platformNodes: make(map[consumer.NodeID]objc.ID),
	}
}

func (c *Context) readTree() (*consumer.Tree, func()) {
	c.treeLock.RLock()
	return c.tree, c.treeLock.RUnlock
}

func (c *Context) writeTree() (*consumer.Tree, func()) {
	c.treeLock.Lock()
	return c.tree, c.treeLock.Unlock
}

func (c *Context) weak() weak.Pointer[Context] {
	return weak.Make(c)
}

func (c *Context) getOrCreatePlatformNode(id consumer.NodeID) objc.ID {
	c.platformNodesLock.Lock()
	defer c.platformNodesLock.Unlock()
	if result, ok := c.platformNodes[id]; ok {
		return result
	}
	result := newPlatformNode(c.weak(), id)
	c.platformNodes[id] = result
	return result
}

func (c *Context) removePlatformNode(id consumer.NodeID) (objc.ID, bool) {
	c.platformNodesLock.Lock()
	defer c.platformNodesLock.Unlock()
	result, ok := c.platformNodes[id]
	delete(c.platformNodes, id)
	return result, ok
}

func (c *Context) doAction(request a11y.ActionRequest) {
	c.actionHandler.DoAction(request)
}

// Release is the drop of the last Rc<Context>: every platform node is announced as destroyed
// and released, and weak references stop resolving.
func (c *Context) Release() {
	if c.released.Swap(true) {
		return
	}
	c.platformNodesLock.Lock()
	nodes := c.platformNodes
	c.platformNodes = make(map[consumer.NodeID]objc.ID)
	c.platformNodesLock.Unlock()
	destroyed := axConst("NSAccessibilityUIElementDestroyedNotification")
	for _, platformNode := range nodes {
		accessibilityPostNotification(platformNode, destroyed)
	}
	for _, platformNode := range nodes {
		objc.Release(platformNode)
	}
}

func upgradeContext(p weak.Pointer[Context]) (*Context, bool) {
	c := p.Value()
	if c == nil || c.released.Load() {
		return nil, false
	}
	return c, true
}
