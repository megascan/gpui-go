package consumer

import (
	"io"

	"github.com/megascan/gpui-go/a11y"
)

func textNodeFilter(rootID NodeID, node *Node) FilterResult {
	if node.ID() == rootID || node.Role() == a11y.RoleTextRun {
		return FilterResultInclude
	}
	return FilterResultExcludeNode
}

func (n Node) textRuns() *FilteredChildren {
	id := n.ID()
	return n.FilteredChildren(func(node *Node) FilterResult { return textNodeFilter(id, node) })
}

// SupportsTextRanges reports whether the node exposes text ranges.
func (n Node) SupportsTextRanges() bool {
	switch {
	case n.IsTextInput():
	case n.Role() == a11y.RoleLabel, n.Role() == a11y.RoleDocument, n.Role() == a11y.RoleTerminal:
	default:
		return false
	}
	_, ok := n.textRuns().Next()
	return ok
}

func (n Node) writeDocumentText(io.StringWriter) error {
	panic("gpui: unimplemented consumer.Node.DocumentRange")
}
