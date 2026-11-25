package bus

import (
	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

const cacheInterfaceName = "org.a11y.atspi.Cache"

type cacheItem struct {
	Object    ownedObjectAddress
	App       ownedObjectAddress
	Parent    ownedObjectAddress
	Index     int32
	Children  int32
	Ifaces    []string
	ShortName string
	Role      atspi.Role
	Name      string
	States    []uint32
}

func objectRef(busName string, id objectID) ownedObjectAddress {
	return newOwnedObjectAddress(busName, id.path())
}

func cacheItemForNode(busName string, node atspi.PlatformNode) (cacheItem, error) {
	cacheNode, err := node.CacheNode()
	if err != nil {
		return cacheItem{}, err
	}
	return newCacheItem(busName, cacheNode), nil
}

func newCacheItem(busName string, node atspi.CacheNode) cacheItem {
	parent := objectIDRoot()
	if id, ok := node.Parent.Node(); ok {
		parent = objectIDNode(node.AdapterID, id)
	}
	return cacheItem{
		Object:    objectRef(busName, objectIDNode(node.AdapterID, node.ID)),
		App:       objectRef(busName, objectIDRoot()),
		Parent:    objectRef(busName, parent),
		Index:     node.IndexInParent,
		Children:  node.ChildCount,
		Ifaces:    interfaceSetWire(node.Interfaces),
		ShortName: node.Name,
		Role:      node.Role,
		Name:      node.Description,
		States:    stateSetWire(node.States),
	}
}

func applicationCacheItem(busName string, root atspi.PlatformRoot, desktop ownedObjectAddress) (cacheItem, error) {
	children, err := root.ChildCount()
	if err != nil {
		return cacheItem{}, err
	}
	shortName, err := root.Name()
	if err != nil {
		return cacheItem{}, err
	}
	name, err := root.Description()
	if err != nil {
		return cacheItem{}, err
	}
	return cacheItem{
		Object:    objectRef(busName, objectIDRoot()),
		App:       objectRef(busName, objectIDRoot()),
		Parent:    desktop,
		Index:     root.IndexInParent(),
		Children:  children,
		Ifaces:    interfaceSetWire(root.Interfaces()),
		ShortName: shortName,
		Role:      root.Role(),
		Name:      name,
		States:    stateSetWire(root.State()),
	}, nil
}

type cacheInterface struct {
	busName string
	root    atspi.PlatformRoot
	desktop *desktopCell
}

func newCacheInterface(busName string, root atspi.PlatformRoot, desktop *desktopCell) *cacheInterface {
	return &cacheInterface{busName: busName, root: root, desktop: desktop}
}

func (i *cacheInterface) items() ([]cacheItem, error) {
	descendants, err := i.root.MapDescendantCacheNodes[cacheItem](func(node atspi.CacheNode) cacheItem {
		return newCacheItem(i.busName, node)
	})
	if err != nil {
		return nil, err
	}
	desktop := i.desktop.getOrDefault()
	items := make([]cacheItem, 0, len(descendants)+1)
	app, err := applicationCacheItem(i.busName, i.root, desktop)
	if err != nil {
		return nil, err
	}
	items = append(items, app)
	items = append(items, descendants...)
	return items, nil
}

func (i *cacheInterface) getItems() ([]cacheItem, *dbus.Error) {
	items, err := i.items()
	if err != nil {
		return nil, mapRootError(err)
	}
	return items, nil
}

func (i *cacheInterface) dbusInterface() *dbusInterface {
	return &dbusInterface{
		name: cacheInterfaceName,
		methods: []dbusMethod{
			{name: "GetItems", out: out("a((so)(so)(so)iiassusau)"), fn: i.getItems},
		},
	}
}
