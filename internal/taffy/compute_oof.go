package taffy

// ComputeOofLayout lays out the out-of-flow candidates in output for which nodeID is the containing block,
// records them as its hoisted children and leaves the unclaimed rest in output. It is a no-op when
// output.OofPositioningArea is None.
func ComputeOofLayout[S CheapCloneStr](tree LayoutContainingBlock[S], nodeID NodeID, output *LayoutOutput) {
	area, ok := output.OofPositioningArea.Get()
	if !ok {
		return
	}

	style := tree.GetCoreContainerStyle(nodeID)
	direction := style.Direction()
	claims := style.IsContainingBlock()

	candidates := output.OofCandidates.Take()
	result := ComputeOofLayoutForArea(tree, nodeID, candidates.AsSlice(), area, direction, claims)
	tree.ClearHoistedChildren(nodeID)
	tree.AddHoistedChildren(nodeID, result.Hoisted)
	output.OofCandidates = result.Unclaimed
	output.ScrollableOverflowRect = output.ScrollableOverflowRect.Union(result.ScrollableOverflowRect)
}
