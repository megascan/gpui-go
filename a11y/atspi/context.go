package atspi

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
)

// ActionHandlerNoMut performs action requests without exclusive access to the handler.
type ActionHandlerNoMut interface {
	DoAction(request a11y.ActionRequest)
}

// ActionHandlerWrapper serializes calls to an a11y.ActionHandler behind a mutex.
type ActionHandlerWrapper struct {
	mu    sync.Mutex
	inner a11y.ActionHandler
}

// NewActionHandlerWrapper wraps inner.
func NewActionHandlerWrapper(inner a11y.ActionHandler) *ActionHandlerWrapper {
	return &ActionHandlerWrapper{inner: inner}
}

// DoAction forwards request to the wrapped handler while holding its lock.
func (w *ActionHandlerWrapper) DoAction(request a11y.ActionRequest) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inner.DoAction(request)
}

// Context is the per-adapter state shared with platform nodes.
type Context struct {
	appContext           *AppContext
	treeLock             sync.RWMutex
	tree                 *consumer.Tree
	actionHandler        ActionHandlerNoMut
	rootWindowBoundsLock sync.RWMutex
	rootWindowBounds     WindowBounds
	released             atomic.Bool
}

// String formats the context like upstream's Debug impl.
func (c *Context) String() string {
	return fmt.Sprintf("Context { app_context: %p, tree: %p, action_handler: \"ActionHandler\", root_window_bounds: %+v }", c.appContext, c.tree, c.readRootWindowBounds())
}

func newContext(appContext *AppContext, tree *consumer.Tree, actionHandler ActionHandlerNoMut, rootWindowBounds WindowBounds) *Context {
	return &Context{
		appContext:       appContext,
		tree:             tree,
		actionHandler:    actionHandler,
		rootWindowBounds: rootWindowBounds,
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

func (c *Context) readRootWindowBounds() WindowBounds {
	c.rootWindowBoundsLock.RLock()
	defer c.rootWindowBoundsLock.RUnlock()
	return c.rootWindowBounds
}

// DoAction forwards request to the adapter's action handler.
func (c *Context) DoAction(request a11y.ActionRequest) {
	c.actionHandler.DoAction(request)
}

func (c *Context) readAppContext() (*AppContext, func()) {
	c.appContext.lock.RLock()
	return c.appContext, c.appContext.lock.RUnlock
}

func (c *Context) writeAppContext() (*AppContext, func()) {
	c.appContext.lock.Lock()
	return c.appContext, c.appContext.lock.Unlock
}

type adapterEntry struct {
	id      int
	context *Context
}

// AppContext is the application-wide state shared by every adapter.
type AppContext struct {
	lock           sync.RWMutex
	name           gpuiutil.Option[string]
	toolkitName    gpuiutil.Option[string]
	toolkitVersion gpuiutil.Option[string]
	id             gpuiutil.Option[int32]
	adapters       []adapterEntry
}

// NewAppContext returns an application context with the given name.
func NewAppContext(name gpuiutil.Option[string]) *AppContext {
	return &AppContext{name: name}
}

// String formats the application context like upstream's Debug impl.
func (a *AppContext) String() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	ids := make([]int, len(a.adapters))
	for i, entry := range a.adapters {
		ids[i] = entry.id
	}
	return fmt.Sprintf("AppContext { name: %v, toolkit_name: %v, toolkit_version: %v, id: %v, adapters: %v }", a.name, a.toolkitName, a.toolkitVersion, a.id, ids)
}

func (a *AppContext) adapterIndex(id int) (int, bool) {
	lo, hi := 0, len(a.adapters)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		switch {
		case a.adapters[mid].id == id:
			return mid, true
		case a.adapters[mid].id < id:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return lo, false
}

func (a *AppContext) pushAdapter(id int, context *Context) {
	a.adapters = append(a.adapters, adapterEntry{id: id, context: context})
}

func (a *AppContext) removeAdapter(id int) {
	if index, ok := a.adapterIndex(id); ok {
		a.adapters = slices.Delete(a.adapters, index, index+1)
	}
}
