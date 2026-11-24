package bus

import (
	"cmp"
	"slices"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/a11y/consumer"
)

const accessibleInterfaceName = "org.a11y.atspi.Accessible"

type relation struct {
	Type    atspi.RelationType
	Targets []ownedObjectAddress
}

type nodeAccessibleInterface struct {
	busName string
	node    atspi.PlatformNode
}

func newNodeAccessibleInterface(busName string, node atspi.PlatformNode) *nodeAccessibleInterface {
	return &nodeAccessibleInterface{busName: busName, node: node}
}

func (i *nodeAccessibleInterface) mapError(err error) *dbus.Error {
	return mapErrorFromNode(i.node, err)
}

func (i *nodeAccessibleInterface) name() (string, *dbus.Error) {
	value, err := i.node.Name()
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) description() (string, *dbus.Error) {
	value, err := i.node.Description()
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) parent() (ownedObjectAddress, *dbus.Error) {
	parent, err := i.node.Parent()
	if err != nil {
		return ownedObjectAddress{}, i.mapError(err)
	}
	if node, ok := parent.Node(); ok {
		return objectIDNode(i.node.AdapterID(), node).toAddress(i.busName), nil
	}
	return objectIDRoot().toAddress(i.busName), nil
}

func (i *nodeAccessibleInterface) childCount() (int32, *dbus.Error) {
	value, err := i.node.ChildCount()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) locale() string {
	return ""
}

func (i *nodeAccessibleInterface) accessibleID() (string, *dbus.Error) {
	value, err := i.node.AccessibleID()
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) getChildAtIndex(index int32) (ownedObjectAddress, *dbus.Error) {
	n, derr := childIndex(index)
	if derr != nil {
		return ownedObjectAddress{}, derr
	}
	child, ok, err := i.node.ChildAtIndex(n)
	if err != nil {
		return ownedObjectAddress{}, i.mapError(err)
	}
	return optionalObjectAddress(i.busName, objectIDNode(i.node.AdapterID(), child), ok), nil
}

func (i *nodeAccessibleInterface) getChildren() ([]ownedObjectAddress, *dbus.Error) {
	children, err := i.node.MapChildren[ownedObjectAddress](func(child consumer.NodeID) ownedObjectAddress {
		return objectIDNode(i.node.AdapterID(), child).toAddress(i.busName)
	})
	if err != nil {
		return nil, i.mapError(err)
	}
	return nonNil(children), nil
}

func (i *nodeAccessibleInterface) getIndexInParent() (int32, *dbus.Error) {
	value, err := i.node.IndexInParent()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) getRelationSet() ([]relation, *dbus.Error) {
	set, err := i.node.RelationSet[ownedObjectAddress](func(target consumer.NodeID) ownedObjectAddress {
		return objectIDNode(i.node.AdapterID(), target).toAddress(i.busName)
	})
	if err != nil {
		return nil, i.mapError(err)
	}
	relations := make([]relation, 0, len(set))
	for kind, targets := range set {
		relations = append(relations, relation{Type: kind, Targets: nonNil(targets)})
	}
	slices.SortFunc(relations, func(a, b relation) int { return cmp.Compare(a.Type, b.Type) })
	return relations, nil
}

func (i *nodeAccessibleInterface) getRole() (atspi.Role, *dbus.Error) {
	value, err := i.node.Role()
	if err != nil {
		return 0, i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) getLocalizedRoleName() (string, *dbus.Error) {
	value, err := i.node.LocalizedRoleName()
	if err != nil {
		return "", i.mapError(err)
	}
	return value, nil
}

func (i *nodeAccessibleInterface) getState() ([]uint32, *dbus.Error) {
	return stateSetWire(i.node.State()), nil
}

func (i *nodeAccessibleInterface) getAttributes() (map[string]string, *dbus.Error) {
	value, err := i.node.Attributes()
	if err != nil {
		return nil, i.mapError(err)
	}
	return nonNilMap(value), nil
}

func (i *nodeAccessibleInterface) getApplication() (ownedObjectAddress, *dbus.Error) {
	return objectIDRoot().toAddress(i.busName), nil
}

func (i *nodeAccessibleInterface) getInterfaces() ([]string, *dbus.Error) {
	value, err := i.node.Interfaces()
	if err != nil {
		return nil, i.mapError(err)
	}
	return interfaceSetWire(value), nil
}

func (i *nodeAccessibleInterface) dbusInterface() *dbusInterface {
	return accessibleDBusInterface(accessibleImpl{
		name:                 i.name,
		description:          i.description,
		parent:               i.parent,
		childCount:           i.childCount,
		locale:               i.locale,
		accessibleID:         i.accessibleID,
		getChildAtIndex:      i.getChildAtIndex,
		getChildren:          i.getChildren,
		getIndexInParent:     i.getIndexInParent,
		getRelationSet:       i.getRelationSet,
		getRole:              i.getRole,
		getLocalizedRoleName: i.getLocalizedRoleName,
		getState:             i.getState,
		getAttributes:        i.getAttributes,
		getApplication:       i.getApplication,
		getInterfaces:        i.getInterfaces,
	})
}

type rootAccessibleInterface struct {
	busName string
	root    atspi.PlatformRoot
	desktop *desktopCell
}

func newRootAccessibleInterface(busName string, root atspi.PlatformRoot, desktop *desktopCell) *rootAccessibleInterface {
	return &rootAccessibleInterface{busName: busName, root: root, desktop: desktop}
}

func (i *rootAccessibleInterface) name() (string, *dbus.Error) {
	value, err := i.root.Name()
	if err != nil {
		return "", mapRootError(err)
	}
	return value, nil
}

func (i *rootAccessibleInterface) description() (string, *dbus.Error) {
	value, err := i.root.Description()
	if err != nil {
		return "", mapRootError(err)
	}
	return value, nil
}

func (i *rootAccessibleInterface) parent() (ownedObjectAddress, *dbus.Error) {
	return i.desktop.getOrDefault(), nil
}

func (i *rootAccessibleInterface) childCount() (int32, *dbus.Error) {
	value, err := i.root.ChildCount()
	if err != nil {
		return 0, mapRootError(err)
	}
	return value, nil
}

func (i *rootAccessibleInterface) locale() string {
	return ""
}

func (i *rootAccessibleInterface) accessibleID() (string, *dbus.Error) {
	return "", nil
}

func (i *rootAccessibleInterface) getChildAtIndex(index int32) (ownedObjectAddress, *dbus.Error) {
	n, derr := childIndex(index)
	if derr != nil {
		return ownedObjectAddress{}, derr
	}
	child, ok, err := i.root.ChildIDAtIndex(n)
	if err != nil {
		return ownedObjectAddress{}, mapRootError(err)
	}
	return optionalObjectAddress(i.busName, objectIDNode(child.AdapterID, child.ID), ok), nil
}

func (i *rootAccessibleInterface) getChildren() ([]ownedObjectAddress, *dbus.Error) {
	children, err := i.root.MapChildIDs[ownedObjectAddress](func(child atspi.AdapterRoot) ownedObjectAddress {
		return objectIDNode(child.AdapterID, child.ID).toAddress(i.busName)
	})
	if err != nil {
		return nil, mapRootError(err)
	}
	return nonNil(children), nil
}

func (i *rootAccessibleInterface) getIndexInParent() (int32, *dbus.Error) {
	return i.root.IndexInParent(), nil
}

func (i *rootAccessibleInterface) getRelationSet() ([]relation, *dbus.Error) {
	return []relation{}, nil
}

func (i *rootAccessibleInterface) getRole() (atspi.Role, *dbus.Error) {
	return i.root.Role(), nil
}

func (i *rootAccessibleInterface) getState() ([]uint32, *dbus.Error) {
	return stateSetWire(i.root.State()), nil
}

func (i *rootAccessibleInterface) getApplication() (ownedObjectAddress, *dbus.Error) {
	return objectIDRoot().toAddress(i.busName), nil
}

func (i *rootAccessibleInterface) getInterfaces() ([]string, *dbus.Error) {
	return interfaceSetWire(i.root.Interfaces()), nil
}

func (i *rootAccessibleInterface) dbusInterface() *dbusInterface {
	return accessibleDBusInterface(accessibleImpl{
		name:             i.name,
		description:      i.description,
		parent:           i.parent,
		childCount:       i.childCount,
		locale:           i.locale,
		accessibleID:     i.accessibleID,
		getChildAtIndex:  i.getChildAtIndex,
		getChildren:      i.getChildren,
		getIndexInParent: i.getIndexInParent,
		getRelationSet:   i.getRelationSet,
		getRole:          i.getRole,
		getState:         i.getState,
		getApplication:   i.getApplication,
		getInterfaces:    i.getInterfaces,
	})
}

type accessibleImpl struct {
	name                 func() (string, *dbus.Error)
	description          func() (string, *dbus.Error)
	parent               func() (ownedObjectAddress, *dbus.Error)
	childCount           func() (int32, *dbus.Error)
	locale               func() string
	accessibleID         func() (string, *dbus.Error)
	getChildAtIndex      func(int32) (ownedObjectAddress, *dbus.Error)
	getChildren          func() ([]ownedObjectAddress, *dbus.Error)
	getIndexInParent     func() (int32, *dbus.Error)
	getRelationSet       func() ([]relation, *dbus.Error)
	getRole              func() (atspi.Role, *dbus.Error)
	getLocalizedRoleName func() (string, *dbus.Error)
	getState             func() ([]uint32, *dbus.Error)
	getAttributes        func() (map[string]string, *dbus.Error)
	getApplication       func() (ownedObjectAddress, *dbus.Error)
	getInterfaces        func() ([]string, *dbus.Error)
}

func accessibleDBusInterface(impl accessibleImpl) *dbusInterface {
	methods := []dbusMethod{
		{name: "GetChildAtIndex", in: in("index", "i"), out: out("(so)"), fn: impl.getChildAtIndex},
		{name: "GetChildren", out: out("a(so)"), fn: impl.getChildren},
		{name: "GetIndexInParent", out: out("i"), fn: impl.getIndexInParent},
		{name: "GetRelationSet", out: out("a(ua(so))"), fn: impl.getRelationSet},
		{name: "GetRole", out: out("u"), fn: impl.getRole},
	}
	if impl.getLocalizedRoleName != nil {
		methods = append(methods, dbusMethod{name: "GetLocalizedRoleName", out: out("s"), fn: impl.getLocalizedRoleName})
	}
	methods = append(methods, dbusMethod{name: "GetState", out: out("au"), fn: impl.getState})
	if impl.getAttributes != nil {
		methods = append(methods, dbusMethod{name: "GetAttributes", out: out("a{ss}"), fn: impl.getAttributes})
	}
	methods = append(methods,
		dbusMethod{name: "GetApplication", out: out("(so)"), fn: impl.getApplication},
		dbusMethod{name: "GetInterfaces", out: out("as"), fn: impl.getInterfaces},
	)
	return &dbusInterface{
		name:    accessibleInterfaceName,
		methods: methods,
		properties: []dbusProperty{
			{name: "Name", signature: "s", get: getter(impl.name)},
			{name: "Description", signature: "s", get: getter(impl.description)},
			{name: "Parent", signature: "(so)", get: getter(impl.parent)},
			{name: "ChildCount", signature: "i", get: getter(impl.childCount)},
			{name: "Locale", signature: "s", get: func() (any, *dbus.Error) { return impl.locale(), nil }},
			{name: "AccessibleId", signature: "s", get: getter(impl.accessibleID)},
		},
	}
}

func getter[T any](f func() (T, *dbus.Error)) func() (any, *dbus.Error) {
	return func() (any, *dbus.Error) {
		value, err := f()
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func nonNilMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	return values
}
