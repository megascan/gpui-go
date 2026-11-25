package consumer

import (
	"io"
	"iter"
	"strings"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

// NodeID identifies a node across the root tree and its grafted subtrees.
type NodeID struct {
	treeIndex treeIndex
	localID   a11y.NodeID
}

func newNodeID(localID a11y.NodeID, ti treeIndex) NodeID {
	return NodeID{treeIndex: ti, localID: localID}
}

func (id NodeID) withSameTree(localID a11y.NodeID) NodeID {
	return NodeID{treeIndex: id.treeIndex, localID: localID}
}

func (id NodeID) toComponents() (a11y.NodeID, treeIndex) {
	return id.localID, id.treeIndex
}

// ToU128 returns the 128-bit form of the ID, (local_id << 64) | tree_index, as its high and low halves.
func (id NodeID) ToU128() (hi, lo uint64) {
	return uint64(id.localID), uint64(id.treeIndex)
}

type parentAndIndex struct {
	parent NodeID
	index  int
}

type nodeState struct {
	parentAndIndex gpuiutil.Option[parentAndIndex]
	data           a11y.Node
}

func (n *nodeState) clone() *nodeState {
	return &nodeState{parentAndIndex: n.parentAndIndex, data: n.data.Clone()}
}

// Node is a view of one node in a State.
type Node struct {
	TreeState *State
	id        NodeID
	state     *nodeState
}

// Data returns the node's data.
func (n Node) Data() *a11y.Node {
	return &n.state.data
}

// IsFocused reports whether the node has focus, taking active descendants into account.
func (n Node) IsFocused() bool {
	dominatedByActiveDescendant := func(nodeID NodeID) bool {
		node, ok := n.TreeState.NodeByID(nodeID)
		if !ok {
			return false
		}
		_, ok = node.ActiveDescendant()
		return ok
	}
	focusID, ok := n.TreeState.FocusID()
	if !ok {
		return false
	}
	if focusID == n.ID() {
		return !dominatedByActiveDescendant(focusID)
	}
	focused, ok := n.TreeState.NodeByID(focusID)
	if !ok {
		return false
	}
	ad, ok := focused.ActiveDescendant()
	return ok && ad.ID() == n.ID()
}

// IsFocusedInTree reports whether the node is the focus of its own tree.
func (n Node) IsFocusedInTree() bool {
	return n.TreeState.focus == n.ID()
}

// IsFocusable reports whether the node supports the focus action or is focused in its tree.
func (n Node) IsFocusable(parentFilter func(*Node) FilterResult) bool {
	return n.SupportsAction(a11y.ActionFocus, parentFilter) || n.IsFocusedInTree()
}

// IsRoot reports whether the node is the root of the tree state.
func (n Node) IsRoot() bool {
	return n.ID() == n.TreeState.RootID()
}

// IsGraft reports whether the node is a graft node.
func (n Node) IsGraft() bool {
	_, ok := n.state.data.TreeID()
	return ok
}

// ParentID returns the ID of the node's parent.
func (n Node) ParentID() (NodeID, bool) {
	pi, ok := n.state.parentAndIndex.Get()
	return pi.parent, ok
}

// Parent returns the node's parent.
func (n Node) Parent() (Node, bool) {
	id, ok := n.ParentID()
	if !ok {
		return Node{}, false
	}
	return n.TreeState.mustNode(id), true
}

// FilteredParent returns the nearest ancestor the filter includes.
func (n Node) FilteredParent(filter func(*Node) FilterResult) (Node, bool) {
	current, ok := n.Parent()
	if !ok {
		return Node{}, false
	}
	for {
		if filter(&current) == FilterResultInclude {
			return current, true
		}
		current, ok = current.Parent()
		if !ok {
			return Node{}, false
		}
	}
}

// ParentAndIndex returns the node's parent and its index among the parent's children.
func (n Node) ParentAndIndex() (Node, int, bool) {
	pi, ok := n.state.parentAndIndex.Get()
	if !ok {
		return Node{}, 0, false
	}
	return n.TreeState.mustNode(pi.parent), pi.index, true
}

func (n Node) graftChildID() (NodeID, bool) {
	treeID, ok := n.state.data.TreeID()
	if !ok {
		return NodeID{}, false
	}
	return n.TreeState.SubtreeRoot(treeID)
}

// ChildIDs returns an iterator over the IDs of the node's children.
func (n Node) ChildIDs() *ChildIDs {
	if n.IsGraft() {
		id, ok := n.graftChildID()
		return &ChildIDs{graft: true, graftID: id, hasGraftID: ok}
	}
	return &ChildIDs{parentID: n.id, children: n.state.data.Children()}
}

// Children returns an iterator over the node's children.
func (n Node) Children() *Nodes {
	return &Nodes{state: n.TreeState, ids: n.ChildIDs()}
}

// FilteredChildren returns an iterator over the node's children according to the filter.
func (n Node) FilteredChildren(filter func(*Node) FilterResult) *FilteredChildren {
	return newFilteredChildren(n, filter)
}

// FollowingSiblingIDs returns an iterator over the IDs of the node's following siblings.
func (n Node) FollowingSiblingIDs() *FollowingSiblings {
	return newFollowingSiblings(n)
}

// FollowingSiblings returns an iterator over the node's following siblings.
func (n Node) FollowingSiblings() *Nodes {
	return &Nodes{state: n.TreeState, ids: n.FollowingSiblingIDs()}
}

// FollowingFilteredSiblings returns an iterator over the node's following siblings according to the filter.
func (n Node) FollowingFilteredSiblings(filter func(*Node) FilterResult) *FollowingFilteredSiblings {
	return newFollowingFilteredSiblings(n, filter)
}

// PrecedingSiblingIDs returns an iterator over the IDs of the node's preceding siblings.
func (n Node) PrecedingSiblingIDs() *PrecedingSiblings {
	return newPrecedingSiblings(n)
}

// PrecedingSiblings returns an iterator over the node's preceding siblings.
func (n Node) PrecedingSiblings() *Nodes {
	return &Nodes{state: n.TreeState, ids: n.PrecedingSiblingIDs()}
}

// PrecedingFilteredSiblings returns an iterator over the node's preceding siblings according to the filter.
func (n Node) PrecedingFilteredSiblings(filter func(*Node) FilterResult) *PrecedingFilteredSiblings {
	return newPrecedingFilteredSiblings(n, filter)
}

// DeepestFirstChild returns the deepest node reached by following first children.
func (n Node) DeepestFirstChild() (Node, bool) {
	deepestChild, ok := n.Children().Next()
	if !ok {
		return Node{}, false
	}
	for {
		firstChild, ok := deepestChild.Children().Next()
		if !ok {
			return deepestChild, true
		}
		deepestChild = firstChild
	}
}

// DeepestFirstFilteredChild returns the deepest node reached by following first filtered children.
func (n Node) DeepestFirstFilteredChild(filter func(*Node) FilterResult) (Node, bool) {
	deepestChild, ok := n.firstFilteredChild(filter)
	if !ok {
		return Node{}, false
	}
	for {
		firstChild, ok := deepestChild.firstFilteredChild(filter)
		if !ok {
			return deepestChild, true
		}
		deepestChild = firstChild
	}
}

// DeepestLastChild returns the deepest node reached by following last children.
func (n Node) DeepestLastChild() (Node, bool) {
	deepestChild, ok := n.Children().NextBack()
	if !ok {
		return Node{}, false
	}
	for {
		lastChild, ok := deepestChild.Children().NextBack()
		if !ok {
			return deepestChild, true
		}
		deepestChild = lastChild
	}
}

// DeepestLastFilteredChild returns the deepest node reached by following last filtered children.
func (n Node) DeepestLastFilteredChild(filter func(*Node) FilterResult) (Node, bool) {
	deepestChild, ok := n.lastFilteredChild(filter)
	if !ok {
		return Node{}, false
	}
	for {
		lastChild, ok := deepestChild.lastFilteredChild(filter)
		if !ok {
			return deepestChild, true
		}
		deepestChild = lastChild
	}
}

// IsDescendantOf reports whether the node is ancestor or one of its descendants.
func (n Node) IsDescendantOf(ancestor Node) bool {
	current := n
	for {
		if current.ID() == ancestor.ID() {
			return true
		}
		parent, ok := current.Parent()
		if !ok {
			return false
		}
		current = parent
	}
}

// DirectTransform returns the transform defined directly on this node, or the identity
// transform, without taking into account transforms on ancestors.
func (n Node) DirectTransform() a11y.Affine {
	if value, ok := n.Data().Transform(); ok {
		return value
	}
	return a11y.AffineIdentity()
}

// Transform returns the combined affine transform of this node and its ancestors,
// up to and including the root of this node's tree.
func (n Node) Transform() a11y.Affine {
	acc := n.DirectTransform()
	current := n
	for {
		parent, ok := current.Parent()
		if !ok {
			return acc
		}
		acc = parent.DirectTransform().Mul(acc)
		current = parent
	}
}

func (n Node) relativeTransform(stopAt Node) a11y.Affine {
	acc := n.DirectTransform()
	current := n
	for {
		parent, ok := current.Parent()
		if !ok || parent.ID() == stopAt.ID() {
			return acc
		}
		acc = parent.DirectTransform().Mul(acc)
		current = parent
	}
}

// RawBounds returns the node's untransformed bounds.
func (n Node) RawBounds() (a11y.Rect, bool) {
	return n.Data().Bounds()
}

// HasBounds reports whether the node has bounds.
func (n Node) HasBounds() bool {
	_, ok := n.RawBounds()
	return ok
}

// BoundingBox returns the node's transformed bounding box relative to the tree's
// container (e.g. window).
func (n Node) BoundingBox() (a11y.Rect, bool) {
	rect, ok := n.RawBounds()
	if !ok {
		return a11y.Rect{}, false
	}
	return n.Transform().TransformRectBbox(rect), true
}

func (n Node) boundingBoxInCoordinateSpace(other Node) (a11y.Rect, bool) {
	rect, ok := n.RawBounds()
	if !ok {
		return a11y.Rect{}, false
	}
	return n.relativeTransform(other).TransformRectBbox(rect), true
}

type hitTestFrame struct {
	test  bool
	node  Node
	point a11y.Point
}

func (n Node) hitTest(point a11y.Point, filter func(*Node) FilterResult) (Node, a11y.Point, bool) {
	stack := make([]hitTestFrame, 0, n.Children().Len()+1)
	stack = append(stack, hitTestFrame{node: n, point: point})
	for len(stack) > 0 {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node, point := frame.node, frame.point
		if frame.test {
			if rect, ok := node.RawBounds(); ok && rect.Contains(point) {
				return node, point, true
			}
			continue
		}
		filterResult := filter(&node)
		if filterResult == FilterResultExcludeSubtree {
			continue
		}
		if filterResult == FilterResultInclude {
			stack = append(stack, hitTestFrame{test: true, node: node, point: point})
		}
		for child := range node.Children().All() {
			childPoint := child.DirectTransform().Inverse().MulPoint(point)
			stack = append(stack, hitTestFrame{node: child, point: childPoint})
		}
	}
	return Node{}, a11y.Point{}, false
}

// NodeAtPoint returns the deepest filtered node, either this node or a descendant,
// at the given point in this node's coordinate space.
func (n Node) NodeAtPoint(point a11y.Point, filter func(*Node) FilterResult) (Node, bool) {
	node, _, ok := n.hitTest(point, filter)
	return node, ok
}

// ID returns the node's ID.
func (n Node) ID() NodeID {
	return n.id
}

// Locate returns the node's tree-local ID and the ID of its tree.
func (n Node) Locate() (a11y.NodeID, a11y.TreeID) {
	localID, treeID, ok := n.TreeState.LocateNode(n.id)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return localID, treeID
}

// Role returns the node's role.
func (n Node) Role() a11y.Role {
	return n.Data().Role()
}

// RoleDescription returns the node's role description.
func (n Node) RoleDescription() (string, bool) {
	return n.Data().RoleDescription()
}

// HasRoleDescription reports whether the node has a role description.
func (n Node) HasRoleDescription() bool {
	_, ok := n.Data().RoleDescription()
	return ok
}

// IsLiveAtomic reports the node's live atomic flag.
func (n Node) IsLiveAtomic() bool {
	return n.Data().IsLiveAtomic()
}

// IsBusy reports the node's busy flag.
func (n Node) IsBusy() bool {
	return n.Data().IsBusy()
}

// ColumnIndexText returns the node's column index text.
func (n Node) ColumnIndexText() (string, bool) {
	return n.Data().ColumnIndexText()
}

// RowIndexText returns the node's row index text.
func (n Node) RowIndexText() (string, bool) {
	return n.Data().RowIndexText()
}

// BrailleLabel returns the node's braille label.
func (n Node) BrailleLabel() (string, bool) {
	return n.Data().BrailleLabel()
}

// HasBrailleLabel reports whether the node has a braille label.
func (n Node) HasBrailleLabel() bool {
	_, ok := n.Data().BrailleLabel()
	return ok
}

// BrailleRoleDescription returns the node's braille role description.
func (n Node) BrailleRoleDescription() (string, bool) {
	return n.Data().BrailleRoleDescription()
}

// HasBrailleRoleDescription reports whether the node has a braille role description.
func (n Node) HasBrailleRoleDescription() bool {
	_, ok := n.Data().BrailleRoleDescription()
	return ok
}

// AriaCurrent returns the node's aria-current value.
func (n Node) AriaCurrent() (a11y.AriaCurrent, bool) {
	return n.Data().AriaCurrent()
}

// HasPopup returns the node's popup kind.
func (n Node) HasPopup() (a11y.HasPopup, bool) {
	return n.Data().HasPopup()
}

// IsHidden reports whether the node or one of its ancestors is hidden.
func (n Node) IsHidden() bool {
	return n.fetchInheritedFlag((*a11y.Node).IsHidden)
}

// Level returns the node's level.
func (n Node) Level() (int, bool) {
	return n.Data().Level()
}

// IsDisabled reports the node's disabled flag.
func (n Node) IsDisabled() bool {
	return n.Data().IsDisabled()
}

// IsReadOnly reports whether the node is read-only, explicitly or by role.
func (n Node) IsReadOnly() bool {
	if n.Data().IsReadOnly() {
		return true
	}
	return n.ShouldHaveReadOnlyStateByDefault() || !n.IsReadOnlySupported()
}

// IsReadOnlyOrDisabled reports whether the node is read-only or disabled.
func (n Node) IsReadOnlyOrDisabled() bool {
	return n.IsReadOnly() || n.IsDisabled()
}

// Toggled returns the node's toggled state.
func (n Node) Toggled() (a11y.Toggled, bool) {
	return n.Data().Toggled()
}

// NumericValue returns the node's numeric value.
func (n Node) NumericValue() (float64, bool) {
	return n.Data().NumericValue()
}

// MinNumericValue returns the node's minimum numeric value.
func (n Node) MinNumericValue() (float64, bool) {
	return n.Data().MinNumericValue()
}

// MaxNumericValue returns the node's maximum numeric value.
func (n Node) MaxNumericValue() (float64, bool) {
	return n.Data().MaxNumericValue()
}

// NumericValueStep returns the node's numeric value step.
func (n Node) NumericValueStep() (float64, bool) {
	return n.Data().NumericValueStep()
}

// NumericValueJump returns the node's numeric value jump.
func (n Node) NumericValueJump() (float64, bool) {
	return n.Data().NumericValueJump()
}

// ClipsChildren reports the node's clips-children flag.
func (n Node) ClipsChildren() bool {
	return n.Data().ClipsChildren()
}

// ScrollX returns the node's horizontal scroll offset.
func (n Node) ScrollX() (float64, bool) {
	return n.Data().ScrollX()
}

// ScrollXMin returns the node's minimum horizontal scroll offset.
func (n Node) ScrollXMin() (float64, bool) {
	return n.Data().ScrollXMin()
}

// ScrollXMax returns the node's maximum horizontal scroll offset.
func (n Node) ScrollXMax() (float64, bool) {
	return n.Data().ScrollXMax()
}

// ScrollY returns the node's vertical scroll offset.
func (n Node) ScrollY() (float64, bool) {
	return n.Data().ScrollY()
}

// ScrollYMin returns the node's minimum vertical scroll offset.
func (n Node) ScrollYMin() (float64, bool) {
	return n.Data().ScrollYMin()
}

// ScrollYMax returns the node's maximum vertical scroll offset.
func (n Node) ScrollYMax() (float64, bool) {
	return n.Data().ScrollYMax()
}

func fetchInheritedProperty[T any](n Node, getter func(*a11y.Node) (T, bool)) (T, bool) {
	node := n
	for {
		if value, ok := getter(node.Data()); ok {
			return value, true
		}
		parent, ok := node.Parent()
		if !ok {
			var zero T
			return zero, false
		}
		node = parent
	}
}

func (n Node) fetchInheritedFlag(getter func(*a11y.Node) bool) bool {
	node := n
	for {
		if getter(node.Data()) {
			return true
		}
		parent, ok := node.Parent()
		if !ok {
			return false
		}
		node = parent
	}
}

// IsTextInput reports whether the node's role is a text input role.
func (n Node) IsTextInput() bool {
	switch n.Role() {
	case a11y.RoleTextInput, a11y.RoleMultilineTextInput, a11y.RoleSearchInput, a11y.RoleDateInput,
		a11y.RoleDateTimeInput, a11y.RoleWeekInput, a11y.RoleMonthInput, a11y.RoleTimeInput,
		a11y.RoleEmailInput, a11y.RoleNumberInput, a11y.RolePasswordInput, a11y.RolePhoneNumberInput,
		a11y.RoleUrlInput, a11y.RoleEditableComboBox, a11y.RoleSpinButton:
		return true
	}
	return false
}

// IsMultiline reports whether the node is a multiline text input.
func (n Node) IsMultiline() bool {
	return n.Role() == a11y.RoleMultilineTextInput
}

// Orientation returns the node's orientation, defaulting by role for list boxes and tab lists.
func (n Node) Orientation() (a11y.Orientation, bool) {
	if o, ok := n.Data().Orientation(); ok {
		return o, true
	}
	switch n.Role() {
	case a11y.RoleListBox:
		return a11y.OrientationVertical, true
	case a11y.RoleTabList:
		return a11y.OrientationHorizontal, true
	}
	return 0, false
}

// IsDialog reports whether the node is a dialog or alert dialog.
func (n Node) IsDialog() bool {
	switch n.Role() {
	case a11y.RoleAlertDialog, a11y.RoleDialog:
		return true
	}
	return false
}

// IsModal reports the node's modal flag.
func (n Node) IsModal() bool {
	return n.Data().IsModal()
}

// IsClickable reports whether the node supports the click action.
func (n Node) IsClickable(parentFilter func(*Node) FilterResult) bool {
	return n.SupportsAction(a11y.ActionClick, parentFilter)
}

// IsSelectable reports whether the node has a selected state and is not disabled.
func (n Node) IsSelectable() bool {
	_, ok := n.IsSelected()
	return ok && !n.IsDisabled()
}

// IsMultiselectable reports the node's multiselectable flag.
func (n Node) IsMultiselectable() bool {
	return n.Data().IsMultiselectable()
}

// SizeOfSetFromContainer returns the set size of the nearest filtered ancestor that has one.
func (n Node) SizeOfSetFromContainer(filter func(*Node) FilterResult) (int, bool) {
	parent, ok := n.FilteredParent(filter)
	for ok {
		if sizeOfSet, has := parent.SizeOfSet(); has {
			return sizeOfSet, true
		}
		parent, ok = parent.FilteredParent(filter)
	}
	return 0, false
}

// SizeOfSet returns the node's set size.
func (n Node) SizeOfSet() (int, bool) {
	return n.Data().SizeOfSet()
}

// PositionInSet returns the node's position in its set.
func (n Node) PositionInSet() (int, bool) {
	return n.Data().PositionInSet()
}

// SortDirection returns the node's sort direction.
func (n Node) SortDirection() (a11y.SortDirection, bool) {
	return n.Data().SortDirection()
}

// SupportsToggle reports whether the node has a toggled state.
func (n Node) SupportsToggle() bool {
	_, ok := n.Toggled()
	return ok
}

// SupportsExpandCollapse reports whether the node can be expanded or collapsed.
func (n Node) SupportsExpandCollapse() bool {
	if _, ok := n.HasPopup(); ok {
		return true
	}
	if _, ok := n.Data().IsExpanded(); ok {
		return true
	}
	switch n.Role() {
	case a11y.RoleComboBox, a11y.RoleEditableComboBox, a11y.RoleDisclosureTriangle, a11y.RoleTreeItem:
		return true
	}
	return false
}

// IsInvocable reports whether the node initiates an action when activated without keeping state.
func (n Node) IsInvocable(parentFilter func(*Node) FilterResult) bool {
	if !n.IsClickable(parentFilter) || n.IsTextInput() {
		return false
	}
	switch n.Role() {
	case a11y.RoleDocument, a11y.RoleTerminal:
		return false
	}
	_, selected := n.IsSelected()
	return !n.SupportsToggle() && !n.SupportsExpandCollapse() && !selected
}

// SupportsAction reports whether the node, or its filtered parent on behalf of children, supports the action.
func (n Node) SupportsAction(action a11y.Action, parentFilter func(*Node) FilterResult) bool {
	if n.Data().SupportsAction(action) {
		return true
	}
	if parent, ok := n.FilteredParent(parentFilter); ok {
		return parent.Data().ChildSupportsAction(action)
	}
	return false
}

// SupportsIncrement reports whether the node supports the increment action.
func (n Node) SupportsIncrement(parentFilter func(*Node) FilterResult) bool {
	return n.SupportsAction(a11y.ActionIncrement, parentFilter)
}

// SupportsDecrement reports whether the node supports the decrement action.
func (n Node) SupportsDecrement(parentFilter func(*Node) FilterResult) bool {
	return n.SupportsAction(a11y.ActionDecrement, parentFilter)
}

func descendantLabelFilter(node *Node) FilterResult {
	switch node.Role() {
	case a11y.RoleLabel, a11y.RoleImage:
		return FilterResultInclude
	case a11y.RoleGenericContainer:
		return FilterResultExcludeNode
	}
	return FilterResultExcludeSubtree
}

// LabelledBy returns an iterator over the nodes that label this node.
func (n Node) LabelledBy() *LabelledBy {
	explicit := n.state.data.LabelledBy()
	if len(explicit) == 0 {
		switch n.Role() {
		case a11y.RoleButton, a11y.RoleCheckBox, a11y.RoleDefaultButton, a11y.RoleLink,
			a11y.RoleMenuItem, a11y.RoleMenuItemCheckBox, a11y.RoleMenuItemRadio, a11y.RoleRadioButton:
			return &LabelledBy{fromDescendants: newFilteredChildren(n, descendantLabelFilter)}
		}
	}
	return &LabelledBy{ids: explicit, treeState: n.TreeState, nodeID: n.id}
}

// LabelComesFromValue reports whether the node's label is its value.
func (n Node) LabelComesFromValue() bool {
	return n.Role() == a11y.RoleLabel
}

// Label returns the node's label, computed from labelling nodes if it has none of its own.
func (n Node) Label() (string, bool) {
	var result strings.Builder
	ok, err := n.WriteLabel(&result)
	if err != nil {
		panic(err)
	}
	if !ok {
		return "", false
	}
	return result.String(), true
}

// HasLabel reports whether the node has a label of its own or from a labelling node.
func (n Node) HasLabel() bool {
	if _, ok := n.Data().Label(); ok {
		return true
	}
	for node := range n.LabelledBy().All() {
		if node.LabelComesFromValue() {
			if node.HasValue() {
				return true
			}
		} else if _, ok := node.Data().Label(); ok {
			return true
		}
	}
	return false
}

func (n Node) writeLabelDirect(writer io.StringWriter) (bool, error) {
	label, ok := n.Data().Label()
	if !ok {
		return false, nil
	}
	if _, err := writer.WriteString(label); err != nil {
		return false, err
	}
	return true, nil
}

// WriteLabel writes the node's label and reports whether there was one.
func (n Node) WriteLabel(writer io.StringWriter) (bool, error) {
	wrote, err := n.writeLabelDirect(writer)
	if err != nil {
		return false, err
	}
	if wrote {
		return true, nil
	}
	wroteOne := false
	for node := range n.LabelledBy().All() {
		w := &spacePrefixingWriter{inner: writer, needPrefix: wroteOne}
		var wroteThisTime bool
		if node.LabelComesFromValue() {
			wroteThisTime, err = node.WriteValue(w)
		} else {
			wroteThisTime, err = node.writeLabelDirect(w)
		}
		if err != nil {
			return false, err
		}
		wroteOne = wroteOne || wroteThisTime
	}
	return wroteOne, nil
}

// Description returns the node's description.
func (n Node) Description() (string, bool) {
	return n.Data().Description()
}

// HasDescription reports whether the node has a description.
func (n Node) HasDescription() bool {
	_, ok := n.Data().Description()
	return ok
}

// URL returns the node's URL.
func (n Node) URL() (string, bool) {
	return n.Data().URL()
}

// SupportsURL reports whether the node is a link-like node with a URL.
func (n Node) SupportsURL() bool {
	switch n.Role() {
	case a11y.RoleLink, a11y.RoleDocBackLink, a11y.RoleDocBiblioRef, a11y.RoleDocGlossRef, a11y.RoleDocNoteRef:
		_, ok := n.URL()
		return ok
	}
	return false
}

func (n Node) isEmptyTextInput() bool {
	textRuns := n.textRuns()
	firstTextRun, ok := textRuns.Next()
	if !ok {
		return true
	}
	value, ok := firstTextRun.Data().Value()
	if ok && value != "" {
		return false
	}
	_, more := textRuns.Next()
	return !more
}

// Placeholder returns the node's placeholder when it is an empty text input.
func (n Node) Placeholder() (string, bool) {
	placeholder, ok := n.Data().Placeholder()
	if !ok || !(n.IsTextInput() && n.isEmptyTextInput()) {
		return "", false
	}
	return placeholder, true
}

// Value returns the node's value, or its document text for single-line text nodes.
func (n Node) Value() (string, bool) {
	var result strings.Builder
	ok, err := n.WriteValue(&result)
	if err != nil {
		panic(err)
	}
	if !ok {
		return "", false
	}
	return result.String(), true
}

// WriteValue writes the node's value and reports whether there was one.
func (n Node) WriteValue(writer io.StringWriter) (bool, error) {
	if value, ok := n.Data().Value(); ok {
		if _, err := writer.WriteString(value); err != nil {
			return false, err
		}
		return true, nil
	}
	if n.SupportsTextRanges() && !n.IsMultiline() {
		if err := n.DocumentRange().WriteText(writer); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// HasValue reports whether the node has a value.
func (n Node) HasValue() bool {
	if _, ok := n.Data().Value(); ok {
		return true
	}
	return n.SupportsTextRanges() && !n.IsMultiline()
}

// IsReadOnlySupported reports whether the node's role can be read-only.
func (n Node) IsReadOnlySupported() bool {
	if n.IsTextInput() {
		return true
	}
	switch n.Role() {
	case a11y.RoleCheckBox, a11y.RoleColorWell, a11y.RoleComboBox, a11y.RoleGrid, a11y.RoleListBox,
		a11y.RoleMenuItemCheckBox, a11y.RoleMenuItemRadio, a11y.RoleMenuListPopup, a11y.RoleRadioButton,
		a11y.RoleRadioGroup, a11y.RoleSlider, a11y.RoleSwitch, a11y.RoleTreeGrid:
		return true
	}
	return false
}

// ShouldHaveReadOnlyStateByDefault reports whether the node's role is read-only by default.
func (n Node) ShouldHaveReadOnlyStateByDefault() bool {
	switch n.Role() {
	case a11y.RoleArticle, a11y.RoleDefinition, a11y.RoleDescriptionList, a11y.RoleDocument,
		a11y.RoleGraphicsDocument, a11y.RoleImage, a11y.RoleList, a11y.RoleListItem, a11y.RolePdfRoot,
		a11y.RoleProgressIndicator, a11y.RoleRootWebArea, a11y.RoleTerm, a11y.RoleTimer, a11y.RoleToolbar,
		a11y.RoleTooltip:
		return true
	}
	return false
}

// IsRequired reports the node's required flag.
func (n Node) IsRequired() bool {
	return n.Data().IsRequired()
}

// Live returns the node's live setting, inherited from ancestors, defaulting to off.
func (n Node) Live() a11y.Live {
	if live, ok := n.Data().Live(); ok {
		return live
	}
	parent, ok := n.Parent()
	if !ok {
		return a11y.LiveOff
	}
	return parent.Live()
}

// IsSelected returns the node's selected state.
func (n Node) IsSelected() (bool, bool) {
	return n.Data().IsSelected()
}

// IsTouchTransparent reports the node's touch-transparent flag.
func (n Node) IsTouchTransparent() bool {
	return n.Data().IsTouchTransparent()
}

// IsItemLike reports whether the node's role is an item role.
func (n Node) IsItemLike() bool {
	switch n.Role() {
	case a11y.RoleArticle, a11y.RoleComment, a11y.RoleListItem, a11y.RoleMenuItem, a11y.RoleMenuItemRadio,
		a11y.RoleTab, a11y.RoleMenuItemCheckBox, a11y.RoleTreeItem, a11y.RoleListBoxOption,
		a11y.RoleMenuListOption, a11y.RoleRadioButton, a11y.RoleTerm:
		return true
	}
	return false
}

// IsContainerWithSelectableChildren reports whether the node's role contains selectable items.
func (n Node) IsContainerWithSelectableChildren() bool {
	switch n.Role() {
	case a11y.RoleComboBox, a11y.RoleEditableComboBox, a11y.RoleGrid, a11y.RoleListBox, a11y.RoleListGrid,
		a11y.RoleMenu, a11y.RoleMenuBar, a11y.RoleMenuListPopup, a11y.RoleRadioGroup, a11y.RoleTabList,
		a11y.RoleToolbar, a11y.RoleTree, a11y.RoleTreeGrid:
		return true
	}
	return false
}

// Controls returns the nodes this node controls.
func (n Node) Controls() iter.Seq[Node] {
	state, id, controls := n.TreeState, n.id, n.state.data.Controls()
	return func(yield func(Node) bool) {
		for _, controlID := range controls {
			if !yield(state.mustNode(id.withSameTree(controlID))) {
				return
			}
		}
	}
}

// ActiveDescendant returns the node's active descendant.
func (n Node) ActiveDescendant() (Node, bool) {
	id, ok := n.state.data.ActiveDescendant()
	if !ok {
		return Node{}, false
	}
	return n.TreeState.NodeByID(n.id.withSameTree(id))
}

// RawTextSelection returns the node's text selection as stored.
func (n Node) RawTextSelection() (a11y.TextSelection, bool) {
	return n.Data().TextSelection()
}

// RawValue returns the node's value as stored.
func (n Node) RawValue() (string, bool) {
	return n.Data().Value()
}

// AuthorID returns the node's author ID.
func (n Node) AuthorID() (string, bool) {
	return n.Data().AuthorID()
}

// ClassName returns the node's class name.
func (n Node) ClassName() (string, bool) {
	return n.Data().ClassName()
}

// IndexPath returns the child indices leading from the root to this node.
func (n Node) IndexPath() []int {
	return n.RelativeIndexPath(n.TreeState.RootID())
}

// RelativeIndexPath returns the child indices leading from the given ancestor to this node.
func (n Node) RelativeIndexPath(ancestorID NodeID) []int {
	var result []int
	current := n
	for current.ID() != ancestorID {
		parent, index, ok := current.ParentAndIndex()
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		result = append(result, index)
		current = parent
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func (n Node) firstFilteredChild(filter func(*Node) FilterResult) (Node, bool) {
	stack := []*Nodes{n.Children()}
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		child, ok := it.Next()
		if !ok {
			stack = stack[:len(stack)-1]
			continue
		}
		switch filter(&child) {
		case FilterResultInclude:
			return child, true
		case FilterResultExcludeNode:
			stack = append(stack, child.Children())
		}
	}
	return Node{}, false
}

func (n Node) lastFilteredChild(filter func(*Node) FilterResult) (Node, bool) {
	stack := []*Nodes{n.Children()}
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		child, ok := it.NextBack()
		if !ok {
			stack = stack[:len(stack)-1]
			continue
		}
		switch filter(&child) {
		case FilterResultInclude:
			return child, true
		case FilterResultExcludeNode:
			stack = append(stack, child.Children())
		}
	}
	return Node{}, false
}

// SelectionContainer returns the nearest filtered ancestor that contains selectable children.
func (n Node) SelectionContainer(filter func(*Node) FilterResult) (Node, bool) {
	return n.FilteredParent(func(parent *Node) FilterResult {
		switch filterResult := filter(parent); {
		case filterResult == FilterResultInclude && parent.IsContainerWithSelectableChildren():
			return FilterResultInclude
		case filterResult == FilterResultInclude:
			return FilterResultExcludeNode
		default:
			return filterResult
		}
	})
}

// Items returns an iterator over the item-like filtered children of the node.
func (n Node) Items(filter func(*Node) FilterResult) *FilteredChildren {
	return n.FilteredChildren(func(child *Node) FilterResult {
		switch filterResult := filter(child); {
		case filterResult == FilterResultInclude && child.IsItemLike():
			return FilterResultInclude
		case filterResult == FilterResultInclude:
			return FilterResultExcludeNode
		default:
			return filterResult
		}
	})
}

type spacePrefixingWriter struct {
	inner      io.StringWriter
	needPrefix bool
}

func (w *spacePrefixingWriter) writePrefixIfNeeded() error {
	if w.needPrefix {
		if _, err := w.inner.WriteString(" "); err != nil {
			return err
		}
		w.needPrefix = false
	}
	return nil
}

func (w *spacePrefixingWriter) WriteString(s string) (int, error) {
	if err := w.writePrefixIfNeeded(); err != nil {
		return 0, err
	}
	return w.inner.WriteString(s)
}
