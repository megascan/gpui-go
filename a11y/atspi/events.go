package atspi

import "github.com/megascan/gpui-go/a11y/consumer"

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
type CacheEvent interface{ isCacheEvent() }

// CacheEventAdded reports a node added to the cache.
type CacheEventAdded struct{ ID consumer.NodeID }

// CacheEventRemoved reports a node removed from the cache.
type CacheEventRemoved struct{ ID consumer.NodeID }

func (CacheEventAdded) isCacheEvent()   {}
func (CacheEventRemoved) isCacheEvent() {}

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
