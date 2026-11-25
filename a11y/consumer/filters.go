package consumer

import "github.com/megascan/gpui-go/a11y"

// FilterResult is the outcome of a node filter.
type FilterResult uint8

const (
	FilterResultInclude FilterResult = iota
	FilterResultExcludeNode
	FilterResultExcludeSubtree
)

func commonFilterBase(node *Node) (FilterResult, bool) {
	if node.IsFocused() {
		return FilterResultInclude, true
	}
	if node.IsHidden() {
		return FilterResultExcludeSubtree, true
	}
	if node.IsGraft() {
		return FilterResultExcludeNode, true
	}
	role := node.Role()
	if role == a11y.RoleGenericContainer || role == a11y.RoleTextRun {
		return FilterResultExcludeNode, true
	}
	return 0, false
}

func commonFilterWithoutParentChecks(node *Node) FilterResult {
	if result, ok := commonFilterBase(node); ok {
		return result
	}
	return FilterResultInclude
}

func isFirstSiblingInParentBbox(next func() (Node, bool), parentBbox a11y.Rect) bool {
	sibling, ok := next()
	if !ok {
		return false
	}
	bbox, ok := sibling.BoundingBox()
	return ok && !bbox.Intersect(parentBbox).IsEmpty()
}

// CommonFilter is the filter most platform adapters use to decide which nodes to expose.
func CommonFilter(node *Node) FilterResult {
	if result, ok := commonFilterBase(node); ok {
		return result
	}
	if parent, ok := node.Parent(); ok {
		if CommonFilter(&parent) == FilterResultExcludeSubtree {
			return FilterResultExcludeSubtree
		}
	}
	if parent, ok := node.FilteredParent(commonFilterWithoutParentChecks); ok && parent.ClipsChildren() {
		if bbox, ok := node.BoundingBox(); ok {
			if parentBbox, ok := parent.BoundingBox(); ok {
				if bbox.Intersect(parentBbox).IsEmpty() &&
					!(isFirstSiblingInParentBbox(node.FollowingFilteredSiblings(commonFilterWithoutParentChecks).Next, parentBbox) ||
						isFirstSiblingInParentBbox(node.PrecedingFilteredSiblings(commonFilterWithoutParentChecks).Next, parentBbox)) {
					return FilterResultExcludeSubtree
				}
			}
		}
	}
	return FilterResultInclude
}

// CommonFilterWithRootException is CommonFilter except that the root is always included.
func CommonFilterWithRootException(node *Node) FilterResult {
	if node.IsRoot() {
		return FilterResultInclude
	}
	return CommonFilter(node)
}
