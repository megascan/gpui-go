//go:build windows

package windows

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/win/com"
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

// Context is the per-adapter state shared with platform nodes.
type Context struct {
	hwnd              WindowHandle
	treeLock          sync.RWMutex
	tree              *consumer.Tree
	actionHandler     actionHandlerNoMut
	isPlaceholder     atomic.Bool
	platformNodesLock sync.Mutex
	platformNodes     map[consumer.NodeID]*com.Object
	released          atomic.Bool
}

// String formats the context like upstream's Debug impl.
func (c *Context) String() string {
	return fmt.Sprintf("Context { hwnd: WindowHandle(%#x), tree: %p, action_handler: \"ActionHandler\", is_placeholder: %v }", uintptr(c.hwnd.HWND), c.tree, c.isPlaceholder.Load())
}

func newContext(hwnd WindowHandle, tree *consumer.Tree, actionHandler actionHandlerNoMut, isPlaceholder bool) *Context {
	c := &Context{
		hwnd:          hwnd,
		tree:          tree,
		actionHandler: actionHandler,
		platformNodes: make(map[consumer.NodeID]*com.Object),
	}
	c.isPlaceholder.Store(isPlaceholder)
	return c
}

func (c *Context) readTree() (*consumer.Tree, func()) {
	c.treeLock.RLock()
	return c.tree, c.treeLock.RUnlock
}

func (c *Context) writeTree() (*consumer.Tree, func()) {
	c.treeLock.Lock()
	return c.tree, c.treeLock.Unlock
}

func (c *Context) clientTopLeft() a11y.Point {
	return clientTopLeft(c.hwnd)
}

func (c *Context) doAction(request a11y.ActionRequest) {
	c.actionHandler.DoAction(request)
}

func (c *Context) getOrCreatePlatformNode(id consumer.NodeID) *com.Object {
	c.platformNodesLock.Lock()
	defer c.platformNodesLock.Unlock()
	if result, ok := c.platformNodes[id]; ok {
		result.AddRef()
		return result
	}
	result := newPlatformNode(c, id)
	result.AddRef()
	c.platformNodes[id] = result
	return result
}

func (c *Context) removePlatformNode(id consumer.NodeID) {
	c.platformNodesLock.Lock()
	result, ok := c.platformNodes[id]
	delete(c.platformNodes, id)
	c.platformNodesLock.Unlock()
	if ok {
		result.Release()
	}
}

// Release is the drop of the last Arc<Context>: weak references stop upgrading and the
// platform nodes the context holds are released.
func (c *Context) Release() {
	if c.released.Swap(true) {
		return
	}
	c.platformNodesLock.Lock()
	nodes := c.platformNodes
	c.platformNodes = make(map[consumer.NodeID]*com.Object)
	c.platformNodesLock.Unlock()
	for _, node := range nodes {
		node.Release()
	}
}
