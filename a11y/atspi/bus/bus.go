package bus

import (
	"errors"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/a11y/consumer"
)

type desktopCell struct {
	mu    sync.Mutex
	value ownedObjectAddress
	set   bool
}

func (c *desktopCell) get() (ownedObjectAddress, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value, c.set
}

func (c *desktopCell) getOrDefault() ownedObjectAddress {
	if value, ok := c.get(); ok {
		return value
	}
	return ownedObjectAddressNull()
}

func (c *desktopCell) trySet(value ownedObjectAddress) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.set {
		c.value = value
		c.set = true
	}
}

type socketProxy struct {
	object dbus.BusObject
}

func newSocketProxy(conn *dbus.Conn) socketProxy {
	return socketProxy{object: conn.Object("org.a11y.atspi.Registry", "/org/a11y/atspi/accessible/root")}
}

func (p socketProxy) embed(plug ownedObjectAddress) (ownedObjectAddress, error) {
	var desktop ownedObjectAddress
	err := p.object.Call("org.a11y.atspi.Socket.Embed", 0, plug).Store(&desktop)
	return desktop, err
}

// Bus is the connection to the AT-SPI bus and the objects exported on it.
type Bus struct {
	conn        *dbus.Conn
	server      *objectServer
	socketProxy socketProxy
	desktop     *desktopCell
}

func busAddress(sessionBus *dbus.Conn) (string, error) {
	if address := os.Getenv("AT_SPI_BUS_ADDRESS"); address != "" {
		return address, nil
	}
	var address string
	err := sessionBus.Object("org.a11y.Bus", "/org/a11y/bus").Call("org.a11y.Bus.GetAddress", 0).Store(&address)
	return address, err
}

func newBus(sessionBus *dbus.Conn) (*Bus, error) {
	address, err := busAddress(sessionBus)
	if err != nil {
		return nil, err
	}
	conn, err := dbus.Connect(address)
	if err != nil {
		return nil, err
	}
	bus := &Bus{
		conn:        conn,
		server:      newObjectServer(conn),
		socketProxy: newSocketProxy(conn),
		desktop:     &desktopCell{},
	}
	if err := bus.registerRootNode(); err != nil {
		bus.close()
		return nil, err
	}
	return bus, nil
}

func (b *Bus) close() {
	_ = b.conn.Close()
}

func (b *Bus) uniqueName() string {
	return b.conn.Names()[0]
}

func (b *Bus) registerRootNode() error {
	node := atspi.NewPlatformRoot(getOrInitAppContext())
	path := objectIDRoot().path()

	added, err := b.server.at(path, (&applicationInterface{root: node}).dbusInterface())
	if err != nil {
		return err
	}
	if added {
		desktop, err := b.socketProxy.embed(newOwnedObjectAddress(b.uniqueName(), objectIDRoot().path()))
		if err != nil {
			return err
		}
		b.desktop.trySet(desktop)

		if _, err := b.server.at(path, newRootAccessibleInterface(b.uniqueName(), node, b.desktop).dbusInterface()); err != nil {
			return err
		}

		if _, err := b.server.at(cachePath(), newCacheInterface(b.uniqueName(), node, b.desktop).dbusInterface()); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bus) registerInterfaces(node atspi.PlatformNode, newInterfaces atspi.InterfaceSet) error {
	path := objectIDFromPlatformNode(node).path()
	busName := b.uniqueName()
	if newInterfaces.Contains(atspi.InterfaceAccessible) {
		if _, err := b.registerInterface(path, newNodeAccessibleInterface(busName, node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceAction) {
		if _, err := b.registerInterface(path, newActionInterface(node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceComponent) {
		if _, err := b.registerInterface(path, newComponentInterface(busName, node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceHyperlink) {
		if _, err := b.registerInterface(path, newHyperlinkInterface(busName, node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceSelection) {
		if _, err := b.registerInterface(path, newSelectionInterface(busName, node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceText) {
		if _, err := b.registerInterface(path, newTextInterface(node).dbusInterface()); err != nil {
			return err
		}
	}
	if newInterfaces.Contains(atspi.InterfaceValue) {
		if _, err := b.registerInterface(path, newValueInterface(node).dbusInterface()); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bus) registerInterface(path dbus.ObjectPath, iface *dbusInterface) (bool, error) {
	result, err := b.server.at(path, iface)
	return mapOrIgnoringRecoverableError(result, err, false, func(result bool) bool { return result })
}

func (b *Bus) unregisterInterfaces(adapterID int, nodeID consumer.NodeID, oldInterfaces atspi.InterfaceSet) error {
	path := objectIDNode(adapterID, nodeID).path()
	for _, iface := range []struct {
		flag atspi.Interface
		name string
	}{
		{atspi.InterfaceAccessible, accessibleInterfaceName},
		{atspi.InterfaceAction, actionInterfaceName},
		{atspi.InterfaceComponent, componentInterfaceName},
		{atspi.InterfaceHyperlink, hyperlinkInterfaceName},
		{atspi.InterfaceSelection, selectionInterfaceName},
		{atspi.InterfaceText, textInterfaceName},
		{atspi.InterfaceValue, valueInterfaceName},
	} {
		if oldInterfaces.Contains(iface.flag) {
			if _, err := b.unregisterInterface(path, iface.name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Bus) unregisterInterface(path dbus.ObjectPath, name string) (bool, error) {
	result, err := b.server.remove(path, name)
	return mapOrIgnoringRecoverableError(result, err, false, func(result bool) bool { return result })
}

type eventBody struct {
	kind    string
	detail1 int32
	detail2 int32
	anyData any
}

func defaultEventBody() eventBody {
	return eventBody{anyData: uint32(0)}
}

func (b *Bus) targetObjectID(adapterID int, target atspi.NodeIDOrRoot) objectID {
	if node, ok := target.Node(); ok {
		return objectIDNode(adapterID, node)
	}
	return objectIDRoot()
}

func (b *Bus) emitObjectEvent(adapterID int, target atspi.NodeIDOrRoot, event atspi.ObjectEvent) error {
	targetID := b.targetObjectID(adapterID, target)
	const iface = "org.a11y.atspi.Event.Object"
	var signal string
	switch event.(type) {
	case atspi.ObjectEventActiveDescendantChanged:
		signal = "ActiveDescendantChanged"
	case atspi.ObjectEventAnnouncement:
		signal = "Announcement"
	case atspi.ObjectEventBoundsChanged:
		signal = "BoundsChanged"
	case atspi.ObjectEventCaretMoved:
		signal = "TextCaretMoved"
	case atspi.ObjectEventChildAdded, atspi.ObjectEventChildRemoved:
		signal = "ChildrenChanged"
	case atspi.ObjectEventPropertyChanged:
		signal = "PropertyChange"
	case atspi.ObjectEventSelectionChanged:
		signal = "SelectionChanged"
	case atspi.ObjectEventStateChanged:
		signal = "StateChanged"
	case atspi.ObjectEventTextInserted, atspi.ObjectEventTextRemoved:
		signal = "TextChanged"
	case atspi.ObjectEventTextSelectionChanged:
		signal = "TextSelectionChanged"
	}
	body := defaultEventBody()
	switch e := event.(type) {
	case atspi.ObjectEventActiveDescendantChanged:
		body.anyData = objectIDNode(adapterID, e.ID).toAddress(b.uniqueName())
	case atspi.ObjectEventAnnouncement:
		body.detail1 = int32(e.Politeness)
		body.anyData = e.Message
	case atspi.ObjectEventBoundsChanged:
		body.anyData = e.Rect
	case atspi.ObjectEventCaretMoved:
		body.detail1 = e.Offset
	case atspi.ObjectEventChildAdded:
		body.kind = "add"
		body.detail1 = int32(e.Index)
		body.anyData = objectIDNode(adapterID, e.Child).toAddress(b.uniqueName())
	case atspi.ObjectEventChildRemoved:
		body.kind = "remove"
		body.detail1 = -1
		body.anyData = objectIDNode(adapterID, e.Child).toAddress(b.uniqueName())
	case atspi.ObjectEventPropertyChanged:
		switch p := e.Property.(type) {
		case atspi.PropertyName:
			body.kind = "accessible-name"
			body.anyData = string(p)
		case atspi.PropertyDescription:
			body.kind = "accessible-description"
			body.anyData = string(p)
		case atspi.PropertyParent:
			body.kind = "accessible-parent"
			body.anyData = b.targetObjectID(adapterID, atspi.NodeIDOrRoot(p)).toAddress(b.uniqueName())
		case atspi.PropertyRole:
			body.kind = "accessible-role"
			body.anyData = uint32(p)
		case atspi.PropertyValue:
			body.kind = "accessible-value"
			body.anyData = float64(p)
		}
	case atspi.ObjectEventStateChanged:
		body.kind = e.State.ToStaticStr()
		if e.Value {
			body.detail1 = 1
		}
	case atspi.ObjectEventTextInserted:
		body.kind = "insert"
		body.detail1 = e.StartIndex
		body.detail2 = e.Length
		body.anyData = e.Content
	case atspi.ObjectEventTextRemoved:
		body.kind = "delete"
		body.detail1 = e.StartIndex
		body.detail2 = e.Length
		body.anyData = e.Content
	}
	return b.emitEvent(targetID, iface, signal, body)
}

func (b *Bus) emitWindowEvent(adapterID int, target consumer.NodeID, windowName string, event atspi.WindowEvent) error {
	targetID := objectIDNode(adapterID, target)
	signal := "Activate"
	if event == atspi.WindowEventDeactivated {
		signal = "Deactivate"
	}
	body := defaultEventBody()
	body.anyData = windowName
	return b.emitEvent(targetID, "org.a11y.atspi.Event.Window", signal, body)
}

func (b *Bus) emitCacheAdd(node atspi.PlatformNode) error {
	item, err := cacheItemForNode(b.uniqueName(), node)
	if err != nil {
		return nil
	}
	return b.emitCacheSignal("AddAccessible", item.Object, item.App, item.Parent, item.Index, item.Children, item.Ifaces, item.ShortName, item.Role, item.Name, item.States)
}

func (b *Bus) emitCacheRemove(adapterID int, nodeID consumer.NodeID) error {
	reference := objectRef(b.uniqueName(), objectIDNode(adapterID, nodeID))
	return b.emitCacheSignal("RemoveAccessible", reference.BusName, reference.Path)
}

func (b *Bus) emitCacheSignal(signalName string, body ...any) error {
	err := b.conn.Emit(cachePath(), cacheInterfaceName+"."+signalName, body...)
	return mapOrIgnoringRecoverableErrorUnit(err)
}

func (b *Bus) emitEvent(target objectID, iface, signalName string, body eventBody) error {
	err := b.conn.Emit(target.path(), iface+"."+signalName, body.kind, body.detail1, body.detail2, dbus.MakeVariant(body.anyData), map[string]dbus.Variant{})
	return mapOrIgnoringRecoverableErrorUnit(err)
}

func zbusErrorIsUnrecoverable(err error) bool {
	return errors.Is(err, errInterfaceExists) || errors.Is(err, errMissingParameter)
}

func mapOrIgnoringRecoverableError[T, U any](result T, err error, def U, f func(T) U) (U, error) {
	if err == nil {
		return f(result), nil
	}
	if zbusErrorIsUnrecoverable(err) {
		return def, err
	}
	return def, nil
}

func mapOrIgnoringRecoverableErrorUnit(err error) error {
	_, err = mapOrIgnoringRecoverableError(struct{}{}, err, struct{}{}, func(struct{}) struct{} { return struct{}{} })
	return err
}
