package atspi

import (
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

// Adapter translates one accesskit tree into AT-SPI objects and events.
type Adapter struct{}

func (a *Adapter) emitObjectEvent(target consumer.NodeID, event ObjectEvent) {
	panic("atspi: unimplemented Adapter.emit_object_event")
}

// Event is an AT-SPI event produced by an Adapter.
type Event interface{ isEvent() }

// EventObject is an event on an object.
type EventObject struct {
	Target NodeIDOrRoot
	Event  ObjectEvent
}

// EventWindow is an event on a window.
type EventWindow struct {
	Target consumer.NodeID
	Name   string
	Event  WindowEvent
}

// EventCache is a change to the AT-SPI cache.
type EventCache struct {
	Event CacheEvent
}

func (EventObject) isEvent() {}
func (EventWindow) isEvent() {}
func (EventCache) isEvent()  {}

// CacheEvent is a node added to or removed from the cache.
type CacheEvent struct {
	Removed bool
	ID      consumer.NodeID
}

// Property is a changed object property.
type Property interface{ isProperty() }

// PropertyName is a new name.
type PropertyName string

// PropertyDescription is a new description.
type PropertyDescription string

// PropertyParent is a new parent.
type PropertyParent NodeIDOrRoot

// PropertyRole is a new role.
type PropertyRole Role

// PropertyValue is a new numeric value.
type PropertyValue float64

func (PropertyName) isProperty()        {}
func (PropertyDescription) isProperty() {}
func (PropertyParent) isProperty()      {}
func (PropertyRole) isProperty()        {}
func (PropertyValue) isProperty()       {}

// ObjectEvent is an event on one object.
type ObjectEvent interface{ isObjectEvent() }

// ObjectEventActiveDescendantChanged reports a new active descendant.
type ObjectEventActiveDescendantChanged struct{ ID consumer.NodeID }

// ObjectEventAnnouncement is a live-region announcement.
type ObjectEventAnnouncement struct {
	Message    string
	Politeness Politeness
}

// ObjectEventBoundsChanged reports new extents.
type ObjectEventBoundsChanged struct{ Rect Rect }

// ObjectEventCaretMoved reports a new caret offset.
type ObjectEventCaretMoved struct{ Offset int32 }

// ObjectEventChildAdded reports a child added at an index.
type ObjectEventChildAdded struct {
	Index int
	Child consumer.NodeID
}

// ObjectEventChildRemoved reports a removed child.
type ObjectEventChildRemoved struct{ Child consumer.NodeID }

// ObjectEventPropertyChanged reports a changed property.
type ObjectEventPropertyChanged struct{ Property Property }

// ObjectEventSelectionChanged reports a changed selection.
type ObjectEventSelectionChanged struct{}

// ObjectEventStateChanged reports a state flag turned on or off.
type ObjectEventStateChanged struct {
	State State
	Value bool
}

// ObjectEventTextInserted reports inserted text.
type ObjectEventTextInserted struct {
	StartIndex int32
	Length     int32
	Content    string
}

// ObjectEventTextRemoved reports removed text.
type ObjectEventTextRemoved struct {
	StartIndex int32
	Length     int32
	Content    string
}

// ObjectEventTextSelectionChanged reports a changed text selection.
type ObjectEventTextSelectionChanged struct{}

func (ObjectEventActiveDescendantChanged) isObjectEvent() {}
func (ObjectEventAnnouncement) isObjectEvent()            {}
func (ObjectEventBoundsChanged) isObjectEvent()           {}
func (ObjectEventCaretMoved) isObjectEvent()              {}
func (ObjectEventChildAdded) isObjectEvent()              {}
func (ObjectEventChildRemoved) isObjectEvent()            {}
func (ObjectEventPropertyChanged) isObjectEvent()         {}
func (ObjectEventSelectionChanged) isObjectEvent()        {}
func (ObjectEventStateChanged) isObjectEvent()            {}
func (ObjectEventTextInserted) isObjectEvent()            {}
func (ObjectEventTextRemoved) isObjectEvent()             {}
func (ObjectEventTextSelectionChanged) isObjectEvent()    {}

// WindowEvent is a window activation change.
type WindowEvent uint8

// WindowEvent variants.
const (
	WindowEventActivated WindowEvent = iota
	WindowEventDeactivated
)
