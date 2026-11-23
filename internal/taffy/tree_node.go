package taffy

import "fmt"

// NodeID is the id of a single node in a tree of nodes, a wrapper around a uint64.
type NodeID struct {
	inner uint64
}

// NewNodeID creates a NodeID from a uint64 value.
func NewNodeID(val uint64) NodeID {
	return NodeID{inner: val}
}

// NodeIDFromU64 converts a uint64 into a NodeID.
func NodeIDFromU64(raw uint64) NodeID {
	return NodeID{inner: raw}
}

// U64FromNodeID converts a NodeID into its uint64 value.
func U64FromNodeID(id NodeID) uint64 {
	return id.inner
}

// NodeIDFromUsize converts an int (upstream usize) into a NodeID.
func NodeIDFromUsize(raw int) NodeID {
	return NodeID{inner: uint64(raw)}
}

// UsizeFromNodeID converts a NodeID into an int (upstream usize).
func UsizeFromNodeID(id NodeID) int {
	return int(id.inner)
}

// String formats the id like upstream's Debug impl: `NodeId(<value>)`.
func (id NodeID) String() string {
	return fmt.Sprintf("NodeId(%d)", id.inner)
}
