package bus

import (
	"fmt"
	"math/big"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/a11y/consumer"
)

const (
	accessiblePathPrefix = "/org/a11y/atspi/accessible/"
	rootPath             = "/org/a11y/atspi/accessible/root"
	cachePathStr         = "/org/a11y/atspi/cache"
)

func cachePath() dbus.ObjectPath {
	return cachePathStr
}

type objectID struct {
	isNode  bool
	adapter int
	node    consumer.NodeID
}

func objectIDRoot() objectID {
	return objectID{}
}

func objectIDNode(adapter int, node consumer.NodeID) objectID {
	return objectID{isNode: true, adapter: adapter, node: node}
}

func objectIDFromPlatformNode(node atspi.PlatformNode) objectID {
	return objectIDNode(node.AdapterID(), node.ID())
}

func nodeIDU128String(id consumer.NodeID) string {
	hi, lo := id.ToU128()
	value := new(big.Int).SetUint64(hi)
	value.Lsh(value, 64)
	value.Or(value, new(big.Int).SetUint64(lo))
	return value.String()
}

func (o objectID) toAddress(busName string) ownedObjectAddress {
	return newOwnedObjectAddress(busName, o.path())
}

func (o objectID) path() dbus.ObjectPath {
	if !o.isNode {
		return rootPath
	}
	return dbus.ObjectPath(fmt.Sprintf("%s%d/%s", accessiblePathPrefix, o.adapter, nodeIDU128String(o.node)))
}

func (o objectID) serialize() string {
	if !o.isNode {
		return "root"
	}
	return nodeIDU128String(o.node)
}
