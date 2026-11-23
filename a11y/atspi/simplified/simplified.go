// Package simplified is a port of accesskit_atspi_common's simplified module:
// an API closer to the libatspi client API for bindings with simpler type systems.
package simplified

import (
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/gpuiutil"
)

// Accessible is either a platform node or the application root.
type Accessible struct {
	node   atspi.PlatformNode
	root   atspi.PlatformRoot
	isRoot bool
}

// AccessibleNode wraps a platform node.
func AccessibleNode(node atspi.PlatformNode) Accessible {
	return Accessible{node: node}
}

// AccessibleRoot wraps the application root.
func AccessibleRoot(root atspi.PlatformRoot) Accessible {
	return Accessible{root: root, isRoot: true}
}

// Node returns the wrapped platform node, or false for the application root.
func (a Accessible) Node() (atspi.PlatformNode, bool) {
	return a.node, !a.isRoot
}

// Root returns the wrapped application root, or false for a node.
func (a Accessible) Root() (atspi.PlatformRoot, bool) {
	return a.root, a.isRoot
}

// IsRoot reports whether a is the application root.
func (a Accessible) IsRoot() bool {
	return a.isRoot
}

// Equal reports whether a and other are the same object.
func (a Accessible) Equal(other Accessible) bool {
	if a.isRoot != other.isRoot {
		return false
	}
	if a.isRoot {
		return a.root.Equal(other.root)
	}
	return a.node.Equal(other.node)
}

// Role returns the AT-SPI role.
func (a Accessible) Role() (atspi.Role, error) {
	if a.isRoot {
		return a.root.Role(), nil
	}
	return a.node.Role()
}

// LocalizedRoleName returns the localized role name; "" for the application.
func (a Accessible) LocalizedRoleName() (string, error) {
	if a.isRoot {
		return "", nil
	}
	return a.node.LocalizedRoleName()
}

// Name returns the accessible name.
func (a Accessible) Name() (string, error) {
	if a.isRoot {
		return a.root.Name()
	}
	return a.node.Name()
}

// Description returns the accessible description.
func (a Accessible) Description() (string, error) {
	if a.isRoot {
		return a.root.Description()
	}
	return a.node.Description()
}

// State returns the state set.
func (a Accessible) State() atspi.StateSet {
	if a.isRoot {
		return a.root.State()
	}
	return a.node.State()
}

// Attributes returns the object attributes; empty for the application.
func (a Accessible) Attributes() (map[string]string, error) {
	if a.isRoot {
		return map[string]string{}, nil
	}
	return a.node.Attributes()
}

// Parent returns the parent object, or false for the application.
func (a Accessible) Parent() (Accessible, bool, error) {
	if a.isRoot {
		return Accessible{}, false, nil
	}
	parent, err := a.node.Parent()
	if err != nil {
		return Accessible{}, false, err
	}
	if id, ok := parent.Node(); ok {
		return AccessibleNode(a.node.Relative(id)), true, nil
	}
	root, err := a.node.Root()
	if err != nil {
		return Accessible{}, false, err
	}
	return AccessibleRoot(root), true, nil
}

// IndexInParent returns the index in the parent.
func (a Accessible) IndexInParent() (int32, error) {
	if a.isRoot {
		return a.root.IndexInParent(), nil
	}
	return a.node.IndexInParent()
}

// ChildCount returns the number of children.
func (a Accessible) ChildCount() (int32, error) {
	if a.isRoot {
		return a.root.ChildCount()
	}
	return a.node.ChildCount()
}

// AccessibleID returns the accessible ID; "" for the application.
func (a Accessible) AccessibleID() (string, error) {
	if a.isRoot {
		return "", nil
	}
	return a.node.AccessibleID()
}

// ChildAtIndex returns the child at index, or false if there is none.
func (a Accessible) ChildAtIndex(index int) (Accessible, bool, error) {
	if a.isRoot {
		child, ok, err := a.root.ChildAtIndex(index)
		if err != nil || !ok {
			return Accessible{}, false, err
		}
		return AccessibleNode(child), true, nil
	}
	id, ok, err := a.node.ChildAtIndex(index)
	if err != nil || !ok {
		return Accessible{}, false, err
	}
	return AccessibleNode(a.node.Relative(id)), true, nil
}

// MapChildren maps f over the children.
func (a Accessible) MapChildren[I any](f func(Accessible) I) ([]I, error) {
	if a.isRoot {
		return a.root.MapChildren(func(node atspi.PlatformNode) I { return f(AccessibleNode(node)) })
	}
	return a.node.MapChildren(func(id atspi.NodeID) I { return f(AccessibleNode(a.node.Relative(id))) })
}

// RelationSet returns the relations; empty for the application.
func (a Accessible) RelationSet() (map[atspi.RelationType][]Accessible, error) {
	if a.isRoot {
		return map[atspi.RelationType][]Accessible{}, nil
	}
	return a.node.RelationSet(func(id atspi.NodeID) Accessible { return AccessibleNode(a.node.Relative(id)) })
}

// Application returns the application object.
func (a Accessible) Application() (Accessible, error) {
	if a.isRoot {
		return AccessibleRoot(a.root), nil
	}
	root, err := a.node.Root()
	if err != nil {
		return Accessible{}, err
	}
	return AccessibleRoot(root), nil
}

// ToolkitName returns the toolkit name.
func (a Accessible) ToolkitName() (string, error) {
	if a.isRoot {
		return a.root.ToolkitName()
	}
	return a.node.ToolkitName()
}

// ToolkitVersion returns the toolkit version.
func (a Accessible) ToolkitVersion() (string, error) {
	if a.isRoot {
		return a.root.ToolkitVersion()
	}
	return a.node.ToolkitVersion()
}

// SupportsAction reports whether the object implements the Action interface.
func (a Accessible) SupportsAction() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsAction()
}

// NActions forwards to the node; the application object does not support it.
func (a Accessible) NActions() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.NActions()
}

// ActionName forwards to the node; the application object does not support it.
func (a Accessible) ActionName(index int32) (string, error) {
	if a.isRoot {
		return "", atspi.ErrorUnsupportedInterface
	}
	return a.node.ActionName(index)
}

// DoAction forwards to the node; the application object does not support it.
func (a Accessible) DoAction(index int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.DoAction(index)
}

// SupportsComponent reports whether the object implements the Component interface.
func (a Accessible) SupportsComponent() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsComponent()
}

// Contains forwards to the node; the application object does not support it.
func (a Accessible) Contains(x, y int32, coordType atspi.CoordType) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.Contains(x, y, coordType)
}

// AccessibleAtPoint returns the object at the point, or false if there is none.
func (a Accessible) AccessibleAtPoint(x, y int32, coordType atspi.CoordType) (Accessible, bool, error) {
	if a.isRoot {
		return Accessible{}, false, atspi.ErrorUnsupportedInterface
	}
	id, ok, err := a.node.AccessibleAtPoint(x, y, coordType)
	if err != nil || !ok {
		return Accessible{}, false, err
	}
	return AccessibleNode(a.node.Relative(id)), true, nil
}

// Extents forwards to the node; the application object does not support it.
func (a Accessible) Extents(coordType atspi.CoordType) (atspi.Rect, error) {
	if a.isRoot {
		return atspi.Rect{}, atspi.ErrorUnsupportedInterface
	}
	return a.node.Extents(coordType)
}

// Layer forwards to the node; the application object does not support it.
func (a Accessible) Layer() (atspi.Layer, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.Layer()
}

// GrabFocus forwards to the node; the application object does not support it.
func (a Accessible) GrabFocus() (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.GrabFocus()
}

// ScrollTo forwards to the node; the application object does not support it.
func (a Accessible) ScrollTo(scrollType atspi.ScrollType) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.ScrollTo(scrollType)
}

// ScrollToPoint forwards to the node; the application object does not support it.
func (a Accessible) ScrollToPoint(coordType atspi.CoordType, x, y int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.ScrollToPoint(coordType, x, y)
}

// SupportsHyperlink reports whether the object implements the Hyperlink interface.
func (a Accessible) SupportsHyperlink() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsHyperlink()
}

// NAnchors forwards to the node; the application object does not support it.
func (a Accessible) NAnchors() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.NAnchors()
}

// HyperlinkStartIndex forwards to the node; the application object does not support it.
func (a Accessible) HyperlinkStartIndex() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.HyperlinkStartIndex()
}

// HyperlinkEndIndex forwards to the node; the application object does not support it.
func (a Accessible) HyperlinkEndIndex() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.HyperlinkEndIndex()
}

// HyperlinkObject returns the anchor object at index, or false if there is none.
func (a Accessible) HyperlinkObject(index int32) (Accessible, bool, error) {
	if a.isRoot {
		return Accessible{}, false, atspi.ErrorUnsupportedInterface
	}
	id, ok, err := a.node.HyperlinkObject(index)
	if err != nil || !ok {
		return Accessible{}, false, err
	}
	return AccessibleNode(a.node.Relative(id)), true, nil
}

// URI forwards to the node; the application object does not support it.
func (a Accessible) URI(index int32) (string, error) {
	if a.isRoot {
		return "", atspi.ErrorUnsupportedInterface
	}
	return a.node.URI(index)
}

// HyperlinkIsValid forwards to the node; the application object does not support it.
func (a Accessible) HyperlinkIsValid() (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.HyperlinkIsValid()
}

// SupportsSelection reports whether the object implements the Selection interface.
func (a Accessible) SupportsSelection() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsSelection()
}

// NSelectedChildren forwards to the node; the application object does not support it.
func (a Accessible) NSelectedChildren() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.NSelectedChildren()
}

// SelectedChild returns the selected child at selectedChildIndex, or false if there is none.
func (a Accessible) SelectedChild(selectedChildIndex int) (Accessible, bool, error) {
	if a.isRoot {
		return Accessible{}, false, atspi.ErrorUnsupportedInterface
	}
	id, ok, err := a.node.SelectedChild(selectedChildIndex)
	if err != nil || !ok {
		return Accessible{}, false, err
	}
	return AccessibleNode(a.node.Relative(id)), true, nil
}

// SelectChild forwards to the node; the application object does not support it.
func (a Accessible) SelectChild(childIndex int) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.SelectChild(childIndex)
}

// DeselectSelectedChild forwards to the node; the application object does not support it.
func (a Accessible) DeselectSelectedChild(selectedChildIndex int) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.DeselectSelectedChild(selectedChildIndex)
}

// IsChildSelected forwards to the node; the application object does not support it.
func (a Accessible) IsChildSelected(childIndex int) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.IsChildSelected(childIndex)
}

// SelectAll forwards to the node; the application object does not support it.
func (a Accessible) SelectAll() (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.SelectAll()
}

// ClearSelection forwards to the node; the application object does not support it.
func (a Accessible) ClearSelection() (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.ClearSelection()
}

// DeselectChild forwards to the node; the application object does not support it.
func (a Accessible) DeselectChild(childIndex int) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.DeselectChild(childIndex)
}

// SupportsText reports whether the object implements the Text interface.
func (a Accessible) SupportsText() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsText()
}

// CharacterCount forwards to the node; the application object does not support it.
func (a Accessible) CharacterCount() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.CharacterCount()
}

// CaretOffset forwards to the node; the application object does not support it.
func (a Accessible) CaretOffset() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.CaretOffset()
}

// StringAtOffset forwards to the node; the application object does not support it.
func (a Accessible) StringAtOffset(offset int32, granularity atspi.Granularity) (string, int32, int32, error) {
	if a.isRoot {
		return "", 0, 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.StringAtOffset(offset, granularity)
}

// Text forwards to the node; the application object does not support it.
func (a Accessible) Text(startOffset, endOffset int32) (string, error) {
	if a.isRoot {
		return "", atspi.ErrorUnsupportedInterface
	}
	return a.node.Text(startOffset, endOffset)
}

// SetCaretOffset forwards to the node; the application object does not support it.
func (a Accessible) SetCaretOffset(offset int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.SetCaretOffset(offset)
}

// TextAttributeValue forwards to the node; the application object does not support it.
func (a Accessible) TextAttributeValue(offset int32, attributeName string) (string, error) {
	if a.isRoot {
		return "", atspi.ErrorUnsupportedInterface
	}
	return a.node.TextAttributeValue(offset, attributeName)
}

// TextAttributes forwards to the node; the application object does not support it.
func (a Accessible) TextAttributes(offset int32) (map[string]string, int32, int32, error) {
	if a.isRoot {
		return nil, 0, 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.TextAttributes(offset)
}

// DefaultTextAttributes forwards to the node; the application object does not support it.
func (a Accessible) DefaultTextAttributes() (map[string]string, error) {
	if a.isRoot {
		return nil, atspi.ErrorUnsupportedInterface
	}
	return a.node.DefaultTextAttributes()
}

// CharacterExtents forwards to the node; the application object does not support it.
func (a Accessible) CharacterExtents(offset int32, coordType atspi.CoordType) (atspi.Rect, error) {
	if a.isRoot {
		return atspi.Rect{}, atspi.ErrorUnsupportedInterface
	}
	return a.node.CharacterExtents(offset, coordType)
}

// OffsetAtPoint forwards to the node; the application object does not support it.
func (a Accessible) OffsetAtPoint(x, y int32, coordType atspi.CoordType) (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.OffsetAtPoint(x, y, coordType)
}

// NSelections forwards to the node; the application object does not support it.
func (a Accessible) NSelections() (int32, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.NSelections()
}

// Selection forwards to the node; the application object does not support it.
func (a Accessible) Selection(selectionNum int32) (int32, int32, error) {
	if a.isRoot {
		return 0, 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.Selection(selectionNum)
}

// AddSelection forwards to the node; the application object does not support it.
func (a Accessible) AddSelection(startOffset, endOffset int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.AddSelection(startOffset, endOffset)
}

// RemoveSelection forwards to the node; the application object does not support it.
func (a Accessible) RemoveSelection(selectionNum int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.RemoveSelection(selectionNum)
}

// SetSelection forwards to the node; the application object does not support it.
func (a Accessible) SetSelection(selectionNum, startOffset, endOffset int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.SetSelection(selectionNum, startOffset, endOffset)
}

// RangeExtents forwards to the node; the application object does not support it.
func (a Accessible) RangeExtents(startOffset, endOffset int32, coordType atspi.CoordType) (atspi.Rect, error) {
	if a.isRoot {
		return atspi.Rect{}, atspi.ErrorUnsupportedInterface
	}
	return a.node.RangeExtents(startOffset, endOffset, coordType)
}

// TextAttributeRun forwards to the node; the application object does not support it.
func (a Accessible) TextAttributeRun(offset int32, includeDefaults bool) (map[string]string, int32, int32, error) {
	if a.isRoot {
		return nil, 0, 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.TextAttributeRun(offset, includeDefaults)
}

// ScrollSubstringTo forwards to the node; the application object does not support it.
func (a Accessible) ScrollSubstringTo(startOffset, endOffset int32, scrollType atspi.ScrollType) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.ScrollSubstringTo(startOffset, endOffset, scrollType)
}

// ScrollSubstringToPoint forwards to the node; the application object does not support it.
func (a Accessible) ScrollSubstringToPoint(startOffset, endOffset int32, coordType atspi.CoordType, x, y int32) (bool, error) {
	if a.isRoot {
		return false, atspi.ErrorUnsupportedInterface
	}
	return a.node.ScrollSubstringToPoint(startOffset, endOffset, coordType, x, y)
}

// SupportsValue reports whether the object implements the Value interface.
func (a Accessible) SupportsValue() (bool, error) {
	if a.isRoot {
		return false, nil
	}
	return a.node.SupportsValue()
}

// MinimumValue forwards to the node; the application object does not support it.
func (a Accessible) MinimumValue() (float64, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.MinimumValue()
}

// MaximumValue forwards to the node; the application object does not support it.
func (a Accessible) MaximumValue() (float64, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.MaximumValue()
}

// MinimumIncrement forwards to the node; the application object does not support it.
func (a Accessible) MinimumIncrement() (float64, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.MinimumIncrement()
}

// CurrentValue forwards to the node; the application object does not support it.
func (a Accessible) CurrentValue() (float64, error) {
	if a.isRoot {
		return 0, atspi.ErrorUnsupportedInterface
	}
	return a.node.CurrentValue()
}

// SetCurrentValue forwards to the node; the application object does not support it.
func (a Accessible) SetCurrentValue(value float64) error {
	if a.isRoot {
		return atspi.ErrorUnsupportedInterface
	}
	return a.node.SetCurrentValue(value)
}

// Cache lists every object of an application for the AT-SPI Cache interface.
type Cache struct {
	root atspi.PlatformRoot
}

// NewCache returns the cache for root.
func NewCache(root atspi.PlatformRoot) *Cache {
	return &Cache{root: root}
}

// Items returns the application root followed by every descendant.
func (c *Cache) Items() ([]Accessible, error) {
	descendants, err := c.root.MapDescendants(func(node atspi.PlatformNode, _, _ int) Accessible {
		return AccessibleNode(node)
	})
	if err != nil {
		return nil, err
	}
	items := make([]Accessible, 0, len(descendants)+1)
	items = append(items, AccessibleRoot(c.root))
	items = append(items, descendants...)
	return items, nil
}

// EventData is the payload of an Event.
type EventData interface{ isEventData() }

// EventDataU32 is an unsigned payload.
type EventDataU32 uint32

// EventDataF64 is a floating-point payload.
type EventDataF64 float64

// EventDataString is a string payload.
type EventDataString string

// EventDataRect is a rectangle payload.
type EventDataRect atspi.Rect

// EventDataAccessible is an object payload.
type EventDataAccessible struct{ Accessible Accessible }

func (EventDataU32) isEventData()        {}
func (EventDataF64) isEventData()        {}
func (EventDataString) isEventData()     {}
func (EventDataRect) isEventData()       {}
func (EventDataAccessible) isEventData() {}

func eventDataEqual(a, b EventData) bool {
	switch av := a.(type) {
	case EventDataAccessible:
		bv, ok := b.(EventDataAccessible)
		return ok && av.Accessible.Equal(bv.Accessible)
	default:
		return a == b
	}
}

// Event is an AT-SPI event in libatspi's shape.
type Event struct {
	Kind    string
	Source  Accessible
	Detail1 int32
	Detail2 int32
	Data    gpuiutil.Option[EventData]
}

// Equal reports whether e and other are the same event.
func (e Event) Equal(other Event) bool {
	if e.Kind != other.Kind || !e.Source.Equal(other.Source) || e.Detail1 != other.Detail1 || e.Detail2 != other.Detail2 {
		return false
	}
	a, aOK := e.Data.Get()
	b, bOK := other.Data.Get()
	return aOK == bOK && (!aOK || eventDataEqual(a, b))
}

func target(adapter *atspi.Adapter, target atspi.NodeIDOrRoot) Accessible {
	if id, ok := target.Node(); ok {
		return AccessibleNode(adapter.PlatformNode(id))
	}
	return AccessibleRoot(adapter.PlatformRoot())
}

func data(value EventData) gpuiutil.Option[EventData] {
	return gpuiutil.Some(value)
}

// NewEvent converts an adapter event into libatspi's shape.
func NewEvent(adapter *atspi.Adapter, event atspi.Event) Event {
	switch ev := event.(type) {
	case atspi.EventObject:
		source := target(adapter, ev.Target)
		switch oe := ev.Event.(type) {
		case atspi.ObjectEventActiveDescendantChanged:
			child := AccessibleNode(adapter.PlatformNode(oe.ID))
			return Event{Kind: "object:active-descendant-changed", Source: source, Data: data(EventDataAccessible{child})}
		case atspi.ObjectEventAnnouncement:
			return Event{Kind: "object:announcement", Source: source, Detail1: int32(oe.Politeness), Data: data(EventDataString(oe.Message))}
		case atspi.ObjectEventBoundsChanged:
			return Event{Kind: "object:bounds-changed", Source: source, Data: data(EventDataRect(oe.Rect))}
		case atspi.ObjectEventCaretMoved:
			return Event{Kind: "object:text-caret-moved", Source: source, Detail1: oe.Offset}
		case atspi.ObjectEventChildAdded:
			child := AccessibleNode(adapter.PlatformNode(oe.Child))
			return Event{Kind: "object:children-changed:add", Source: source, Detail1: int32(oe.Index), Data: data(EventDataAccessible{child})}
		case atspi.ObjectEventChildRemoved:
			child := AccessibleNode(adapter.PlatformNode(oe.Child))
			return Event{Kind: "object:children-changed:remove", Source: source, Detail1: -1, Data: data(EventDataAccessible{child})}
		case atspi.ObjectEventPropertyChanged:
			var kind string
			var payload EventData
			switch p := oe.Property.(type) {
			case atspi.PropertyName:
				kind, payload = "object:property-change:accessible-name", EventDataString(p)
			case atspi.PropertyDescription:
				kind, payload = "object:property-change:accessible-description", EventDataString(p)
			case atspi.PropertyParent:
				kind, payload = "object:property-change:accessible-parent", EventDataAccessible{target(adapter, atspi.NodeIDOrRoot(p))}
			case atspi.PropertyRole:
				kind, payload = "object:property-change:accessible-role", EventDataU32(p)
			case atspi.PropertyValue:
				kind, payload = "object:property-change:accessible-value", EventDataF64(p)
			}
			return Event{Kind: kind, Source: source, Data: data(payload)}
		case atspi.ObjectEventSelectionChanged:
			return Event{Kind: "object:selection-changed", Source: source}
		case atspi.ObjectEventStateChanged:
			detail1 := int32(0)
			if oe.Value {
				detail1 = 1
			}
			return Event{Kind: "object:state-changed:" + oe.State.ToStaticStr(), Source: source, Detail1: detail1}
		case atspi.ObjectEventTextInserted:
			return Event{Kind: "object:text-changed:insert", Source: source, Detail1: oe.StartIndex, Detail2: oe.Length, Data: data(EventDataString(oe.Content))}
		case atspi.ObjectEventTextRemoved:
			return Event{Kind: "object:text-changed:delete", Source: source, Detail1: oe.StartIndex, Detail2: oe.Length, Data: data(EventDataString(oe.Content))}
		case atspi.ObjectEventTextSelectionChanged:
			return Event{Kind: "object:text-selection-changed", Source: source}
		}
	case atspi.EventWindow:
		kind := "window:activate"
		if ev.Event == atspi.WindowEventDeactivated {
			kind = "window:deactivate"
		}
		return Event{Kind: kind, Source: AccessibleNode(adapter.PlatformNode(ev.Target)), Data: data(EventDataString(ev.Name))}
	case atspi.EventCache:
		var kind string
		var id atspi.NodeID
		switch ce := ev.Event.(type) {
		case atspi.CacheEventAdded:
			kind, id = "cache:add", ce.ID
		case atspi.CacheEventRemoved:
			kind, id = "cache:remove", ce.ID
		}
		return Event{Kind: kind, Source: AccessibleNode(adapter.PlatformNode(id))}
	}
	panic("simplified: unknown event")
}
