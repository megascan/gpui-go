package atspi

import "github.com/megascan/gpui-go/a11y/consumer"

// AdapterCallback receives interface registrations and events from an Adapter.
type AdapterCallback interface {
	RegisterInterfaces(adapter *Adapter, id consumer.NodeID, interfaces InterfaceSet)
	UnregisterInterfaces(adapter *Adapter, id consumer.NodeID, interfaces InterfaceSet)
	EmitEvent(adapter *Adapter, event Event)
}
