package atspi

import (
	"math"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
)

type nodeWrapper struct {
	node *consumer.Node
}

func toI32(n int, err Error) (int32, error) {
	if n < 0 || n > math.MaxInt32 {
		return 0, err
	}
	return int32(n), nil
}

func rustMinF64(a, b float64) float64 {
	switch {
	case math.IsNaN(a):
		return b
	case math.IsNaN(b):
		return a
	}
	return math.Min(a, b)
}

func filteredParentOrRoot(node *consumer.Node) NodeIDOrRoot {
	if parent, ok := node.FilteredParent(filter); ok {
		return NodeIDOrRootNode(parent.ID())
	}
	return NodeIDOrRootRoot()
}

func (w nodeWrapper) name() (string, bool) {
	if w.node.LabelComesFromValue() {
		return w.node.Value()
	}
	return w.node.Label()
}

func (w nodeWrapper) description() (string, bool) {
	return w.node.Description()
}

func (w nodeWrapper) parentID() (consumer.NodeID, bool) {
	return w.node.ParentID()
}

func (w nodeWrapper) id() consumer.NodeID {
	return w.node.ID()
}

func (w nodeWrapper) cacheNode(adapterID int, indexInParent, childCount int32, windowFocused bool) CacheNode {
	name, _ := w.name()
	description, _ := w.description()
	return CacheNode{
		AdapterID:     adapterID,
		ID:            w.id(),
		Parent:        filteredParentOrRoot(w.node),
		IndexInParent: indexInParent,
		ChildCount:    childCount,
		Interfaces:    w.interfaces(),
		Name:          name,
		Description:   description,
		Role:          w.role(),
		States:        w.state(windowFocused),
	}
}

func (w nodeWrapper) filteredChildIDs() []consumer.NodeID {
	var ids []consumer.NodeID
	for child := range w.node.FilteredChildren(filter).All() {
		ids = append(ids, child.ID())
	}
	return ids
}

func (w nodeWrapper) role() Role {
	if w.node.HasRoleDescription() {
		return RoleExtended
	}
	switch w.node.Role() {
	case a11y.RoleAlert:
		return RoleNotification
	case a11y.RoleAlertDialog:
		return RoleAlert
	case a11y.RoleComment, a11y.RoleSuggestion:
		return RoleSection
	case a11y.RoleApplication:
		return RoleEmbedded
	case a11y.RoleArticle:
		return RoleArticle
	case a11y.RoleAudio:
		return RoleAudio
	case a11y.RoleBanner, a11y.RoleHeader:
		return RoleLandmark
	case a11y.RoleBlockquote:
		return RoleBlockQuote
	case a11y.RoleCaret:
		return RoleUnknown
	case a11y.RoleButton:
		if _, ok := w.node.Toggled(); ok {
			return RoleToggleButton
		}
		return RoleButton
	case a11y.RoleDefaultButton:
		return RoleButton
	case a11y.RoleCanvas:
		return RoleCanvas
	case a11y.RoleCaption:
		return RoleCaption
	case a11y.RoleCell, a11y.RoleGridCell:
		return RoleTableCell
	case a11y.RoleCheckBox:
		return RoleCheckBox
	case a11y.RoleSwitch:
		return RoleToggleButton
	case a11y.RoleColorWell:
		return RoleButton
	case a11y.RoleColumnHeader:
		return RoleColumnHeader
	case a11y.RoleComboBox, a11y.RoleEditableComboBox:
		return RoleComboBox
	case a11y.RoleComplementary:
		return RoleLandmark
	case a11y.RoleContentDeletion:
		return RoleContentDeletion
	case a11y.RoleContentInsertion:
		return RoleContentInsertion
	case a11y.RoleContentInfo, a11y.RoleFooter:
		return RoleLandmark
	case a11y.RoleDefinition:
		return RoleDescriptionValue
	case a11y.RoleDescriptionList:
		return RoleDescriptionList
	case a11y.RoleDetails:
		return RolePanel
	case a11y.RoleDialog:
		return RoleDialog
	case a11y.RoleDisclosureTriangle:
		return RoleToggleButton
	case a11y.RoleDocCover:
		return RoleImage
	case a11y.RoleDocBackLink, a11y.RoleDocBiblioRef, a11y.RoleDocGlossRef, a11y.RoleDocNoteRef:
		return RoleLink
	case a11y.RoleDocBiblioEntry, a11y.RoleDocEndnote:
		return RoleListItem
	case a11y.RoleDocNotice, a11y.RoleDocTip:
		return RoleComment
	case a11y.RoleDocFootnote:
		return RoleFootnote
	case a11y.RoleDocPageBreak:
		return RoleSeparator
	case a11y.RoleDocPageFooter:
		return RoleFooter
	case a11y.RoleDocPageHeader:
		return RoleHeader
	case a11y.RoleDocAcknowledgements, a11y.RoleDocAfterword, a11y.RoleDocAppendix, a11y.RoleDocBibliography,
		a11y.RoleDocChapter, a11y.RoleDocConclusion, a11y.RoleDocCredits, a11y.RoleDocEndnotes,
		a11y.RoleDocEpilogue, a11y.RoleDocErrata, a11y.RoleDocForeword, a11y.RoleDocGlossary,
		a11y.RoleDocIndex, a11y.RoleDocIntroduction, a11y.RoleDocPageList, a11y.RoleDocPart,
		a11y.RoleDocPreface, a11y.RoleDocPrologue, a11y.RoleDocToc:
		return RoleLandmark
	case a11y.RoleDocAbstract, a11y.RoleDocColophon, a11y.RoleDocCredit, a11y.RoleDocDedication,
		a11y.RoleDocEpigraph, a11y.RoleDocExample, a11y.RoleDocPullquote, a11y.RoleDocQna:
		return RoleSection
	case a11y.RoleDocSubtitle:
		return RoleHeading
	case a11y.RoleDocument:
		return RoleDocumentFrame
	case a11y.RoleEmbeddedObject:
		return RoleEmbedded
	case a11y.RoleForm:
		return RoleForm
	case a11y.RoleFigure, a11y.RoleFeed:
		return RolePanel
	case a11y.RoleGenericContainer, a11y.RoleRuby:
		return RoleSection
	case a11y.RoleGraphicsDocument:
		return RoleDocumentFrame
	case a11y.RoleGraphicsObject:
		return RolePanel
	case a11y.RoleGraphicsSymbol:
		return RoleImage
	case a11y.RoleGrid:
		return RoleTable
	case a11y.RoleGroup:
		return RolePanel
	case a11y.RoleHeading:
		return RoleHeading
	case a11y.RoleIframe, a11y.RoleIframePresentational:
		return RoleInternalFrame
	case a11y.RoleImage:
		return RoleImage
	case a11y.RoleTextRun:
		return RoleStatic
	case a11y.RoleLegend:
		return RoleLabel
	case a11y.RoleLayoutTable:
		return RoleSection
	case a11y.RoleLayoutTableCell:
		return RoleSection
	case a11y.RoleLayoutTableRow:
		return RoleSection
	case a11y.RoleLineBreak:
		return RoleStatic
	case a11y.RoleLink:
		return RoleLink
	case a11y.RoleList:
		return RoleList
	case a11y.RoleListBox:
		return RoleListBox
	case a11y.RoleListBoxOption:
		return RoleListItem
	case a11y.RoleListGrid:
		return RoleTable
	case a11y.RoleListItem:
		return RoleListItem
	case a11y.RoleListMarker:
		return RoleStatic
	case a11y.RoleLog:
		return RoleLog
	case a11y.RoleMain:
		return RoleLandmark
	case a11y.RoleMark:
		return RoleStatic
	case a11y.RoleMath:
		return RoleMath
	case a11y.RoleMarquee:
		return RoleMarquee
	case a11y.RoleMenu, a11y.RoleMenuListPopup:
		return RoleMenu
	case a11y.RoleMenuBar:
		return RoleMenuBar
	case a11y.RoleMenuItem, a11y.RoleMenuListOption:
		return RoleMenuItem
	case a11y.RoleMenuItemCheckBox:
		return RoleCheckMenuItem
	case a11y.RoleMenuItemRadio:
		return RoleRadioMenuItem
	case a11y.RoleMeter:
		return RoleLevelBar
	case a11y.RoleNavigation:
		return RoleLandmark
	case a11y.RoleNote:
		return RoleComment
	case a11y.RolePane, a11y.RoleScrollView:
		return RolePanel
	case a11y.RoleParagraph:
		return RoleParagraph
	case a11y.RolePdfActionableHighlight:
		return RoleButton
	case a11y.RolePdfRoot:
		return RoleDocumentFrame
	case a11y.RolePluginObject:
		return RoleEmbedded
	case a11y.RoleProgressIndicator:
		return RoleProgressBar
	case a11y.RoleRadioButton:
		return RoleRadioButton
	case a11y.RoleRadioGroup:
		return RolePanel
	case a11y.RoleRegion:
		return RoleLandmark
	case a11y.RoleRootWebArea:
		return RoleDocumentWeb
	case a11y.RoleRow:
		return RoleTableRow
	case a11y.RoleRowGroup:
		return RolePanel
	case a11y.RoleRowHeader:
		return RoleRowHeader
	case a11y.RoleRubyAnnotation:
		return RoleStatic
	case a11y.RoleSection:
		return RoleSection
	case a11y.RoleSectionFooter:
		return RoleFooter
	case a11y.RoleSectionHeader:
		return RoleHeader
	case a11y.RoleScrollBar:
		return RoleScrollBar
	case a11y.RoleSearch:
		return RoleLandmark
	case a11y.RoleSlider:
		return RoleSlider
	case a11y.RoleSpinButton:
		return RoleSpinButton
	case a11y.RoleSplitter:
		return RoleSeparator
	case a11y.RoleLabel:
		return RoleLabel
	case a11y.RoleStatus:
		return RoleStatusBar
	case a11y.RoleSvgRoot:
		return RoleDocumentFrame
	case a11y.RoleTab:
		return RolePageTab
	case a11y.RoleTable:
		return RoleTable
	case a11y.RoleTabList:
		return RolePageTabList
	case a11y.RoleTabPanel:
		return RoleScrollPane
	case a11y.RoleTerm:
		return RoleDescriptionTerm
	case a11y.RoleTitleBar:
		return RoleTitleBar
	case a11y.RoleTextInput, a11y.RoleMultilineTextInput, a11y.RoleSearchInput, a11y.RoleEmailInput,
		a11y.RoleNumberInput, a11y.RolePhoneNumberInput, a11y.RoleUrlInput:
		return RoleEntry
	case a11y.RoleDateInput, a11y.RoleDateTimeInput, a11y.RoleWeekInput, a11y.RoleMonthInput, a11y.RoleTimeInput:
		return RoleDateEditor
	case a11y.RolePasswordInput:
		return RolePasswordText
	case a11y.RoleAbbr, a11y.RoleCode, a11y.RoleEmphasis, a11y.RoleStrong, a11y.RoleTime:
		return RoleStatic
	case a11y.RoleTimer:
		return RoleTimer
	case a11y.RoleToolbar:
		return RoleToolBar
	case a11y.RoleTooltip:
		return RoleToolTip
	case a11y.RoleTree:
		return RoleTree
	case a11y.RoleTreeItem:
		return RoleTreeItem
	case a11y.RoleTreeGrid:
		return RoleTreeTable
	case a11y.RoleVideo:
		return RoleVideo
	case a11y.RoleWindow:
		return RoleFrame
	case a11y.RoleWebView:
		return RolePanel
	case a11y.RoleFigureCaption:
		return RoleCaption
	case a11y.RoleUnknown:
		return RoleUnknown
	case a11y.RoleImeCandidate, a11y.RoleKeyboard:
		return RoleRedundantObject
	case a11y.RoleTerminal:
		return RoleTerminal
	}
	panic("atspi: unmapped accesskit role " + w.node.Role().String())
}

func (w nodeWrapper) isFocused() bool {
	return w.node.IsFocused()
}

func (w nodeWrapper) state(isWindowFocused bool) StateSet {
	state := w.node
	atspiRole := w.role()
	atspiState := StateSetEmpty()
	if isWindowFocused {
		_, hasParent := state.ParentID()
		active := !hasParent && state.Role() == a11y.RoleWindow
		if !active && state.IsDialog() {
			if dialog, ok := state.TreeState.ActiveDialog(); ok && dialog.ID() == state.ID() {
				active = true
			}
		}
		if active {
			atspiState.Insert(StateActive)
		}
	}
	if state.IsTextInput() && !state.IsReadOnly() {
		atspiState.Insert(StateEditable)
	}
	if state.IsFocusable(filter) {
		atspiState.Insert(StateFocusable)
	}
	if filter(w.node) == consumer.FilterResultInclude {
		atspiState.Insert(StateVisible | StateShowing)
	}
	if state.IsRequired() {
		atspiState.Insert(StateRequired)
	}
	if state.IsMultiselectable() {
		atspiState.Insert(StateMultiselectable)
	}
	if orientation, ok := state.Orientation(); ok {
		if orientation == a11y.OrientationHorizontal {
			atspiState.Insert(StateHorizontal)
		} else {
			atspiState.Insert(StateVertical)
		}
	}
	toggled, hasToggled := state.Toggled()
	if atspiRole != RoleToggleButton && hasToggled {
		atspiState.Insert(StateCheckable)
	}
	if state.IsModal() {
		atspiState.Insert(StateModal)
	}
	if selected, ok := state.IsSelected(); ok {
		if !state.IsDisabled() {
			atspiState.Insert(StateSelectable)
		}
		if selected {
			atspiState.Insert(StateSelected)
		}
	}
	if state.IsTextInput() {
		atspiState.Insert(StateSelectableText)
		if state.IsMultiline() {
			atspiState.Insert(StateMultiLine)
		} else {
			atspiState.Insert(StateSingleLine)
		}
	}
	if state.Role() == a11y.RoleProgressIndicator {
		if _, ok := state.NumericValue(); !ok {
			atspiState.Insert(StateIndeterminate)
		}
	}
	if hasToggled {
		switch {
		case toggled == a11y.ToggledMixed:
			atspiState.Insert(StateIndeterminate)
		case toggled == a11y.ToggledTrue && atspiRole == RoleToggleButton:
			atspiState.Insert(StatePressed)
		case toggled == a11y.ToggledTrue:
			atspiState.Insert(StateChecked)
		}
	}
	if state.IsReadOnlySupported() && state.IsReadOnlyOrDisabled() {
		atspiState.Insert(StateReadOnly)
	} else {
		atspiState.Insert(StateEnabled | StateSensitive)
	}
	if w.isFocused() {
		atspiState.Insert(StateFocused)
	}
	return atspiState
}

func (w nodeWrapper) placeholder() (string, bool) {
	return w.node.Placeholder()
}

func (w nodeWrapper) positionInSet() (string, bool) {
	p, ok := w.node.PositionInSet()
	if !ok {
		return "", false
	}
	return itoa(p + 1), true
}

func (w nodeWrapper) sizeOfSet() (string, bool) {
	s, ok := w.node.SizeOfSetFromContainer(filter)
	if !ok {
		return "", false
	}
	return itoa(s), true
}

func (w nodeWrapper) brailleLabel() (string, bool) {
	return w.node.BrailleLabel()
}

func (w nodeWrapper) brailleRoleDescription() (string, bool) {
	return w.node.BrailleRoleDescription()
}

func (w nodeWrapper) attributes() map[string]string {
	attributes := map[string]string{}
	if placeholder, ok := w.placeholder(); ok {
		attributes["placeholder-text"] = placeholder
	}
	if positionInSet, ok := w.positionInSet(); ok {
		attributes["posinset"] = positionInSet
	}
	if sizeOfSet, ok := w.sizeOfSet(); ok {
		attributes["setsize"] = sizeOfSet
	}
	if label, ok := w.brailleLabel(); ok {
		attributes["braillelabel"] = label
	}
	if roleDescription, ok := w.brailleRoleDescription(); ok {
		attributes["brailleroledescription"] = roleDescription
	}
	return attributes
}

func (w nodeWrapper) isRoot() bool {
	return w.node.IsRoot()
}

func (w nodeWrapper) supportsAction() bool {
	return w.node.IsClickable(filter)
}

func (w nodeWrapper) supportsComponent() bool {
	_, ok := w.node.RawBounds()
	return ok || w.isRoot()
}

func (w nodeWrapper) supportsHyperlink() bool {
	return w.node.SupportsURL()
}

func (w nodeWrapper) supportsSelection() bool {
	return w.node.IsContainerWithSelectableChildren()
}

func (w nodeWrapper) supportsText() bool {
	return w.node.SupportsTextRanges()
}

func (w nodeWrapper) supportsValue() bool {
	_, ok := w.currentValue()
	return ok
}

func (w nodeWrapper) interfaces() InterfaceSet {
	interfaces := NewInterfaceSet(InterfaceAccessible)
	if w.supportsAction() {
		interfaces.Insert(InterfaceAction)
	}
	if w.supportsComponent() {
		interfaces.Insert(InterfaceComponent)
	}
	if w.supportsHyperlink() {
		interfaces.Insert(InterfaceHyperlink)
	}
	if w.supportsSelection() {
		interfaces.Insert(InterfaceSelection)
	}
	if w.supportsText() {
		interfaces.Insert(InterfaceText)
	}
	if w.supportsValue() {
		interfaces.Insert(InterfaceValue)
	}
	return interfaces
}

func (w nodeWrapper) live() Politeness {
	switch w.node.Live() {
	case a11y.LivePolite:
		return PolitenessPolite
	case a11y.LiveAssertive:
		return PolitenessAssertive
	default:
		return PolitenessNone
	}
}

func (w nodeWrapper) nActions() int32 {
	if w.node.IsClickable(filter) {
		return 1
	}
	return 0
}

func (w nodeWrapper) getActionName(index int32) string {
	if index != 0 {
		return ""
	}
	if w.node.IsClickable(filter) {
		return "click"
	}
	return ""
}

type boundsAndTransform struct {
	bounds    a11y.Rect
	hasBounds bool
	transform a11y.Affine
}

func (w nodeWrapper) rawBoundsAndTransform() boundsAndTransform {
	bounds, ok := w.node.RawBounds()
	if !ok {
		bounds = a11y.Rect{}
	}
	return boundsAndTransform{bounds: bounds, hasBounds: ok, transform: w.node.DirectTransform()}
}

func (w nodeWrapper) extents(windowBounds *WindowBounds, coordType CoordType) (a11y.Rect, bool) {
	bounds, ok := w.node.BoundingBox()
	if w.isRoot() {
		wb := windowBounds.Inner.WithOrigin(a11y.PointZero())
		if !wb.IsEmpty() {
			if ok {
				bounds.X0 = rustMinF64(bounds.X0, wb.X1)
				bounds.Y0 = rustMinF64(bounds.Y0, wb.Y1)
				bounds.X1 = rustMinF64(bounds.X1, wb.X1)
				bounds.Y1 = rustMinF64(bounds.Y1, wb.Y1)
			} else {
				bounds, ok = wb, true
			}
		}
	}
	if !ok {
		return a11y.Rect{}, false
	}
	newOrigin := windowBounds.accesskitPointToAtspiPoint(bounds.Origin(), optionalNode(w.node.FilteredParent(filter)), coordType)
	return bounds.WithOrigin(newOrigin), true
}

func (w nodeWrapper) currentValue() (float64, bool) {
	return w.node.NumericValue()
}

func (w nodeWrapper) notifyChanges(windowBounds *WindowBounds, adapter *Adapter, old nodeWrapper) {
	w.notifyStateChanges(adapter, old)
	w.notifyPropertyChanges(adapter, old)
	w.notifyBoundsChanges(windowBounds, adapter, old)
	w.notifyChildrenChanges(adapter, old)
}

func (w nodeWrapper) notifyStateChanges(adapter *Adapter, old nodeWrapper) {
	oldState := old.state(true)
	newState := w.state(true)
	changedStates := oldState.Xor(newState)
	for state := range changedStates.Iter() {
		if state == StateFocused {
			continue
		}
		adapter.emitObjectEvent(w.id(), ObjectEventStateChanged{State: state, Value: newState.Contains(state)})
	}
}

func (w nodeWrapper) notifyPropertyChanges(adapter *Adapter, old nodeWrapper) {
	name, hasName := w.name()
	oldName, oldHasName := old.name()
	if name != oldName || hasName != oldHasName {
		adapter.emitObjectEvent(w.id(), ObjectEventPropertyChanged{Property: PropertyName(name)})
		if live := w.live(); live != PolitenessNone {
			adapter.emitObjectEvent(w.id(), ObjectEventAnnouncement{Message: name, Politeness: live})
		}
	}
	description, hasDescription := w.description()
	oldDescription, oldHasDescription := old.description()
	if description != oldDescription || hasDescription != oldHasDescription {
		adapter.emitObjectEvent(w.id(), ObjectEventPropertyChanged{Property: PropertyDescription(description)})
	}
	parentID, hasParent := w.parentID()
	oldParentID, oldHasParent := old.parentID()
	if parentID != oldParentID || hasParent != oldHasParent {
		adapter.emitObjectEvent(w.id(), ObjectEventPropertyChanged{Property: PropertyParent(filteredParentOrRoot(w.node))})
	}
	if role := w.role(); role != old.role() {
		adapter.emitObjectEvent(w.id(), ObjectEventPropertyChanged{Property: PropertyRole(role)})
	}
	if value, ok := w.currentValue(); ok {
		if oldValue, oldOK := old.currentValue(); !oldOK || value != oldValue {
			adapter.emitObjectEvent(w.id(), ObjectEventPropertyChanged{Property: PropertyValue(value)})
		}
	}
}

func (w nodeWrapper) notifyBoundsChanges(windowBounds *WindowBounds, adapter *Adapter, old nodeWrapper) {
	if w.rawBoundsAndTransform() != old.rawBoundsAndTransform() {
		if extents, ok := w.extents(windowBounds, CoordTypeWindow); ok {
			adapter.emitObjectEvent(w.id(), ObjectEventBoundsChanged{Rect: RectFromAccessKit(extents)})
		}
	}
}

func containsID(ids []consumer.NodeID, id consumer.NodeID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func (w nodeWrapper) notifyChildrenChanges(adapter *Adapter, old nodeWrapper) {
	oldFilteredChildren := old.filteredChildIDs()
	newFilteredChildren := w.filteredChildIDs()
	for index, child := range newFilteredChildren {
		if !containsID(oldFilteredChildren, child) {
			adapter.emitObjectEvent(w.id(), ObjectEventChildAdded{Index: index, Child: child})
		}
	}
	for _, child := range oldFilteredChildren {
		if !containsID(newFilteredChildren, child) {
			adapter.emitObjectEvent(w.id(), ObjectEventChildRemoved{Child: child})
		}
	}
}

// PlatformNode is the AT-SPI object for one node of an adapter's tree.
type PlatformNode struct {
	context   weak.Pointer[Context]
	adapterID int
	id        consumer.NodeID
}

func newPlatformNode(context *Context, adapterID int, id consumer.NodeID) PlatformNode {
	return PlatformNode{context: weak.Make(context), adapterID: adapterID, id: id}
}

func platformNodeFromAdapterRoot(entry adapterEntry) PlatformNode {
	tree, done := entry.context.readTree()
	rootID := tree.State().RootID()
	done()
	return newPlatformNode(entry.context, entry.id, rootID)
}

func (p PlatformNode) upgradeContext() (*Context, error) {
	if context := p.context.Value(); context != nil && !context.released.Load() {
		return context, nil
	}
	return nil, ErrorDefunct
}

func withTreeState[T any](p PlatformNode, f func(*consumer.State) (T, error)) (T, error) {
	context, err := p.upgradeContext()
	if err != nil {
		var zero T
		return zero, err
	}
	tree, done := context.readTree()
	defer done()
	return f(tree.State())
}

func withTreeAndContext[T any](p PlatformNode, f func(*consumer.Tree, *Context) (T, error)) (T, error) {
	context, err := p.upgradeContext()
	if err != nil {
		var zero T
		return zero, err
	}
	tree, done := context.readTree()
	defer done()
	return f(tree, context)
}

func resolveWithContext[T any](p PlatformNode, f func(*consumer.Node, *consumer.Tree, *Context) (T, error)) (T, error) {
	return withTreeAndContext(p, func(tree *consumer.Tree, context *Context) (T, error) {
		if node, ok := tree.State().NodeByID(p.id); ok {
			return f(&node, tree, context)
		}
		var zero T
		return zero, ErrorDefunct
	})
}

func resolveForSelectionWithContext[T any](p PlatformNode, f func(*consumer.Node, *consumer.Tree, *Context) (T, error)) (T, error) {
	return resolveWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (T, error) {
		if (nodeWrapper{node}).supportsSelection() {
			return f(node, tree, context)
		}
		var zero T
		return zero, ErrorUnsupportedInterface
	})
}

func resolveForTextWithContext[T any](p PlatformNode, f func(*consumer.Node, *consumer.Tree, *Context) (T, error)) (T, error) {
	return resolveWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (T, error) {
		if (nodeWrapper{node}).supportsText() {
			return f(node, tree, context)
		}
		var zero T
		return zero, ErrorUnsupportedInterface
	})
}

func resolve[T any](p PlatformNode, f func(*consumer.Node) (T, error)) (T, error) {
	return resolveWithContext(p, func(node *consumer.Node, _ *consumer.Tree, _ *Context) (T, error) {
		return f(node)
	})
}

func resolveForSelection[T any](p PlatformNode, f func(*consumer.Node) (T, error)) (T, error) {
	return resolve(p, func(node *consumer.Node) (T, error) {
		if (nodeWrapper{node}).supportsSelection() {
			return f(node)
		}
		var zero T
		return zero, ErrorUnsupportedInterface
	})
}

func resolveForText[T any](p PlatformNode, f func(*consumer.Node) (T, error)) (T, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, _ *consumer.Tree, _ *Context) (T, error) {
		return f(node)
	})
}

func (p PlatformNode) doActionInternal(target consumer.NodeID, f func(*consumer.State, *Context, a11y.NodeID, a11y.TreeID) a11y.ActionRequest) error {
	context, err := p.upgradeContext()
	if err != nil {
		return err
	}
	tree, done := context.readTree()
	targetNode, targetTree, ok := tree.State().LocateNode(target)
	if !ok {
		done()
		return ErrorDefunct
	}
	request := f(tree.State(), context, targetNode, targetTree)
	done()
	context.DoAction(request)
	return nil
}

func doLocatedAction(tree *consumer.Tree, context *Context, target consumer.NodeID, action a11y.Action, data a11y.ActionData) error {
	targetNode, targetTree, ok := tree.State().LocateNode(target)
	if !ok {
		return ErrorDefunct
	}
	context.DoAction(a11y.ActionRequest{Action: action, TargetTree: targetTree, TargetNode: targetNode, Data: data})
	return nil
}

func simpleRequest(action a11y.Action, data a11y.ActionData) func(*consumer.State, *Context, a11y.NodeID, a11y.TreeID) a11y.ActionRequest {
	return func(_ *consumer.State, _ *Context, targetNode a11y.NodeID, targetTree a11y.TreeID) a11y.ActionRequest {
		return a11y.ActionRequest{Action: action, TargetTree: targetTree, TargetNode: targetNode, Data: data}
	}
}

// Name returns the accessible name.
func (p PlatformNode) Name() (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		name, _ := nodeWrapper{node}.name()
		return name, nil
	})
}

// Description returns the accessible description.
func (p PlatformNode) Description() (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		description, _ := nodeWrapper{node}.description()
		return description, nil
	})
}

// Relative returns the platform node for id in the same adapter.
func (p PlatformNode) Relative(id consumer.NodeID) PlatformNode {
	return PlatformNode{context: p.context, adapterID: p.adapterID, id: id}
}

// Root returns the application root object.
func (p PlatformNode) Root() (PlatformRoot, error) {
	context, err := p.upgradeContext()
	if err != nil {
		return PlatformRoot{}, err
	}
	return NewPlatformRoot(context.appContext), nil
}

// ToolkitName returns the tree's toolkit name.
func (p PlatformNode) ToolkitName() (string, error) {
	return withTreeState(p, func(state *consumer.State) (string, error) {
		return state.ToolkitName(), nil
	})
}

// ToolkitVersion returns the tree's toolkit version.
func (p PlatformNode) ToolkitVersion() (string, error) {
	return withTreeState(p, func(state *consumer.State) (string, error) {
		version, _ := state.ToolkitVersion()
		return version, nil
	})
}

// Parent returns the filtered parent, or the application root.
func (p PlatformNode) Parent() (NodeIDOrRoot, error) {
	return resolve(p, func(node *consumer.Node) (NodeIDOrRoot, error) {
		return filteredParentOrRoot(node), nil
	})
}

func filteredChildCount(node *consumer.Node) int {
	n := 0
	for range node.FilteredChildren(filter).All() {
		n++
	}
	return n
}

func precedingFilteredSiblingCount(node *consumer.Node) int {
	n := 0
	for range node.PrecedingFilteredSiblings(filter).All() {
		n++
	}
	return n
}

// ChildCount returns the number of filtered children.
func (p PlatformNode) ChildCount() (int32, error) {
	return resolve(p, func(node *consumer.Node) (int32, error) {
		return toI32(filteredChildCount(node), ErrorTooManyChildren)
	})
}

// AdapterID returns the ID of the adapter that owns the node.
func (p PlatformNode) AdapterID() int {
	return p.adapterID
}

// ID returns the node's ID.
func (p PlatformNode) ID() consumer.NodeID {
	return p.id
}

// AccessibleID returns the author ID, or "".
func (p PlatformNode) AccessibleID() (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		authorID, _ := node.AuthorID()
		return authorID, nil
	})
}

// ChildAtIndex returns the ID of the filtered child at index.
func (p PlatformNode) ChildAtIndex(index int) (consumer.NodeID, bool, error) {
	type result struct {
		id consumer.NodeID
		ok bool
	}
	r, err := resolve(p, func(node *consumer.Node) (result, error) {
		i := 0
		for child := range node.FilteredChildren(filter).All() {
			if i == index {
				return result{child.ID(), true}, nil
			}
			i++
		}
		return result{}, nil
	})
	return r.id, r.ok, err
}

// MapChildren maps f over the filtered children's IDs.
func (p PlatformNode) MapChildren[I any](f func(consumer.NodeID) I) ([]I, error) {
	return resolve(p, func(node *consumer.Node) ([]I, error) {
		var children []I
		for child := range node.FilteredChildren(filter).All() {
			children = append(children, f(child.ID()))
		}
		return children, nil
	})
}

func indexInParentOf(node *consumer.Node, context *Context, adapterID int) (int32, error) {
	if _, ok := node.FilteredParent(filter); ok {
		return toI32(precedingFilteredSiblingCount(node), ErrorIndexOutOfRange)
	}
	appContext, done := context.readAppContext()
	index, ok := appContext.adapterIndex(adapterID)
	done()
	if !ok {
		return 0, ErrorDefunct
	}
	return toI32(index, ErrorIndexOutOfRange)
}

// IndexInParent returns the index among the filtered siblings, or among the application's adapters for a root.
func (p PlatformNode) IndexInParent() (int32, error) {
	return resolveWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (int32, error) {
		return indexInParentOf(node, context, p.adapterID)
	})
}

// CacheNode returns the node's AT-SPI cache entry.
func (p PlatformNode) CacheNode() (CacheNode, error) {
	return resolveWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (CacheNode, error) {
		indexInParent, err := indexInParentOf(node, context, p.adapterID)
		if err != nil {
			return CacheNode{}, err
		}
		childCount, err := toI32(filteredChildCount(node), ErrorTooManyChildren)
		if err != nil {
			return CacheNode{}, err
		}
		_, focused := tree.State().FocusID()
		return nodeWrapper{node}.cacheNode(p.adapterID, indexInParent, childCount, focused), nil
	})
}

// RelationSet returns the node's relations, mapping each related node through f.
func (p PlatformNode) RelationSet[T any](f func(consumer.NodeID) T) (map[RelationType][]T, error) {
	return resolve(p, func(node *consumer.Node) (map[RelationType][]T, error) {
		relations := map[RelationType][]T{}
		var controls []T
		for controlled := range node.Controls() {
			if filter(&controlled) == consumer.FilterResultInclude {
				controls = append(controls, f(controlled.ID()))
			}
		}
		if len(controls) > 0 {
			relations[RelationTypeControllerFor] = controls
		}
		return relations, nil
	})
}

// Role returns the AT-SPI role.
func (p PlatformNode) Role() (Role, error) {
	return resolve(p, func(node *consumer.Node) (Role, error) {
		return nodeWrapper{node}.role(), nil
	})
}

// LocalizedRoleName returns the role description, or "".
func (p PlatformNode) LocalizedRoleName() (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		description, _ := node.RoleDescription()
		return description, nil
	})
}

// State returns the node's states, or only StateDefunct if the node is gone.
func (p PlatformNode) State() StateSet {
	s, err := resolveWithContext(p, func(node *consumer.Node, tree *consumer.Tree, _ *Context) (StateSet, error) {
		_, focused := tree.State().FocusID()
		return nodeWrapper{node}.state(focused), nil
	})
	if err != nil {
		return NewStateSet(StateDefunct)
	}
	return s
}

// Attributes returns the object attributes.
func (p PlatformNode) Attributes() (map[string]string, error) {
	return resolve(p, func(node *consumer.Node) (map[string]string, error) {
		return nodeWrapper{node}.attributes(), nil
	})
}

func resolveBool(p PlatformNode, f func(nodeWrapper) bool) (bool, error) {
	return resolve(p, func(node *consumer.Node) (bool, error) {
		return f(nodeWrapper{node}), nil
	})
}

// SupportsAction reports whether the node exposes the Action interface.
func (p PlatformNode) SupportsAction() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsAction)
}

// SupportsComponent reports whether the node exposes the Component interface.
func (p PlatformNode) SupportsComponent() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsComponent)
}

// SupportsHyperlink reports whether the node exposes the Hyperlink interface.
func (p PlatformNode) SupportsHyperlink() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsHyperlink)
}

// SupportsSelection reports whether the node exposes the Selection interface.
func (p PlatformNode) SupportsSelection() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsSelection)
}

// SupportsText reports whether the node exposes the Text interface.
func (p PlatformNode) SupportsText() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsText)
}

// SupportsValue reports whether the node exposes the Value interface.
func (p PlatformNode) SupportsValue() (bool, error) {
	return resolveBool(p, nodeWrapper.supportsValue)
}

// Interfaces returns the interfaces the node exposes.
func (p PlatformNode) Interfaces() (InterfaceSet, error) {
	return resolve(p, func(node *consumer.Node) (InterfaceSet, error) {
		return nodeWrapper{node}.interfaces(), nil
	})
}

// NActions returns the number of actions.
func (p PlatformNode) NActions() (int32, error) {
	return resolve(p, func(node *consumer.Node) (int32, error) {
		return nodeWrapper{node}.nActions(), nil
	})
}

// ActionName returns the name of the action at index.
func (p PlatformNode) ActionName(index int32) (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		return nodeWrapper{node}.getActionName(index), nil
	})
}

// Actions returns every action.
func (p PlatformNode) Actions() ([]Action, error) {
	return resolve(p, func(node *consumer.Node) ([]Action, error) {
		wrapper := nodeWrapper{node}
		nActions := int(wrapper.nActions())
		actions := make([]Action, 0, nActions)
		for i := range nActions {
			actions = append(actions, Action{LocalizedName: wrapper.getActionName(int32(i))})
		}
		return actions, nil
	})
}

// DoAction performs the action at index.
func (p PlatformNode) DoAction(index int32) (bool, error) {
	if index != 0 {
		return false, nil
	}
	if err := p.doActionInternal(p.id, simpleRequest(a11y.ActionClick, nil)); err != nil {
		return false, err
	}
	return true, nil
}

// Contains reports whether the point is inside the node's extents.
func (p PlatformNode) Contains(x, y int32, coordType CoordType) (bool, error) {
	return resolveWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (bool, error) {
		windowBounds := context.readRootWindowBounds()
		if extents, ok := (nodeWrapper{node}).extents(&windowBounds, coordType); ok {
			return extents.Contains(a11y.NewPoint(float64(x), float64(y))), nil
		}
		return false, nil
	})
}

// AccessibleAtPoint returns the deepest filtered node at the point.
func (p PlatformNode) AccessibleAtPoint(x, y int32, coordType CoordType) (consumer.NodeID, bool, error) {
	type result struct {
		id consumer.NodeID
		ok bool
	}
	r, err := resolveWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (result, error) {
		windowBounds := context.readRootWindowBounds()
		point := windowBounds.atspiPointToAccesskitPoint(a11y.NewPoint(float64(x), float64(y)), node, coordType)
		point = node.Transform().Inverse().MulPoint(point)
		if hit, ok := node.NodeAtPoint(point, filter); ok {
			return result{hit.ID(), true}, nil
		}
		return result{}, nil
	})
	return r.id, r.ok, err
}

// Extents returns the node's extents, or RectInvalid.
func (p PlatformNode) Extents(coordType CoordType) (Rect, error) {
	return resolveWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (Rect, error) {
		windowBounds := context.readRootWindowBounds()
		if extents, ok := (nodeWrapper{node}).extents(&windowBounds, coordType); ok {
			return RectFromAccessKit(extents), nil
		}
		return RectInvalid(), nil
	})
}

// Layer returns the component layer.
func (p PlatformNode) Layer() (Layer, error) {
	return resolve(p, func(node *consumer.Node) (Layer, error) {
		if (nodeWrapper{node}).role() == RoleWindow {
			return LayerWindow, nil
		}
		return LayerWidget, nil
	})
}

// GrabFocus requests focus.
func (p PlatformNode) GrabFocus() (bool, error) {
	if err := p.doActionInternal(p.id, simpleRequest(a11y.ActionFocus, nil)); err != nil {
		return false, err
	}
	return true, nil
}

func scrollHintData(scrollType ScrollType) a11y.ActionData {
	if hint, ok := atspiScrollTypeToScrollHint(scrollType); ok {
		return a11y.ActionDataScrollHint(hint)
	}
	return nil
}

// ScrollTo scrolls the node into view.
func (p PlatformNode) ScrollTo(scrollType ScrollType) (bool, error) {
	if err := p.doActionInternal(p.id, simpleRequest(a11y.ActionScrollIntoView, scrollHintData(scrollType))); err != nil {
		return false, err
	}
	return true, nil
}

// ScrollToPoint scrolls the node to the point.
func (p PlatformNode) ScrollToPoint(coordType CoordType, x, y int32) (bool, error) {
	_, err := resolveWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (struct{}, error) {
		windowBounds := context.readRootWindowBounds()
		point := windowBounds.atspiPointToAccesskitPoint(a11y.NewPoint(float64(x), float64(y)), optionalNode(node.FilteredParent(filter)), coordType)
		return struct{}{}, doLocatedAction(tree, context, p.id, a11y.ActionScrollToPoint, a11y.ActionDataScrollToPoint(point))
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// NAnchors returns the number of hyperlink anchors.
func (p PlatformNode) NAnchors() (int32, error) {
	return resolve(p, func(node *consumer.Node) (int32, error) {
		if _, ok := node.URL(); ok {
			return 1, nil
		}
		return 0, nil
	})
}

// HyperlinkStartIndex returns the hyperlink's start offset; rich text is unsupported, so -1.
func (p PlatformNode) HyperlinkStartIndex() (int32, error) {
	return resolve(p, func(*consumer.Node) (int32, error) { return -1, nil })
}

// HyperlinkEndIndex returns the hyperlink's end offset; rich text is unsupported, so -1.
func (p PlatformNode) HyperlinkEndIndex() (int32, error) {
	return resolve(p, func(*consumer.Node) (int32, error) { return -1, nil })
}

// HyperlinkObject returns the node itself for index 0.
func (p PlatformNode) HyperlinkObject(index int32) (consumer.NodeID, bool, error) {
	_, err := resolve(p, func(*consumer.Node) (struct{}, error) { return struct{}{}, nil })
	if err != nil || index != 0 {
		return consumer.NodeID{}, false, err
	}
	return p.id, true, nil
}

// URI returns the URL for index 0, or "".
func (p PlatformNode) URI(index int32) (string, error) {
	return resolve(p, func(node *consumer.Node) (string, error) {
		if index == 0 {
			url, _ := node.URL()
			return url, nil
		}
		return "", nil
	})
}

// HyperlinkIsValid reports whether the node has a URL.
func (p PlatformNode) HyperlinkIsValid() (bool, error) {
	return resolve(p, func(node *consumer.Node) (bool, error) {
		_, ok := node.URL()
		return ok, nil
	})
}

func isSelectedTrue(node consumer.Node) bool {
	selected, ok := node.IsSelected()
	return ok && selected
}

func nthSelectedItem(node *consumer.Node, index int) (consumer.Node, bool) {
	i := 0
	for item := range node.Items(filter).All() {
		if !isSelectedTrue(item) {
			continue
		}
		if i == index {
			return item, true
		}
		i++
	}
	return consumer.Node{}, false
}

func nthFilteredChild(node *consumer.Node, index int) (consumer.Node, bool) {
	i := 0
	for child := range node.FilteredChildren(filter).All() {
		if i == index {
			return child, true
		}
		i++
	}
	return consumer.Node{}, false
}

// NSelectedChildren returns the number of selected items.
func (p PlatformNode) NSelectedChildren() (int32, error) {
	return resolveForSelection(p, func(node *consumer.Node) (int32, error) {
		n := 0
		for item := range node.Items(filter).All() {
			if isSelectedTrue(item) {
				n++
			}
		}
		return toI32(n, ErrorTooManyChildren)
	})
}

// SelectedChild returns the ID of the selected item at selectedChildIndex.
func (p PlatformNode) SelectedChild(selectedChildIndex int) (consumer.NodeID, bool, error) {
	type result struct {
		id consumer.NodeID
		ok bool
	}
	r, err := resolveForSelection(p, func(node *consumer.Node) (result, error) {
		if item, ok := nthSelectedItem(node, selectedChildIndex); ok {
			return result{item.ID(), true}, nil
		}
		return result{}, nil
	})
	return r.id, r.ok, err
}

// SelectChild selects the filtered child at childIndex by clicking it.
func (p PlatformNode) SelectChild(childIndex int) (bool, error) {
	return resolveForSelectionWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		child, ok := nthFilteredChild(node, childIndex)
		if !ok {
			return false, ErrorDefunct
		}
		if isSelectedTrue(child) {
			return true, nil
		}
		if child.IsSelectable() && child.IsClickable(filter) {
			if err := doLocatedAction(tree, context, child.ID(), a11y.ActionClick, nil); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
}

// DeselectSelectedChild deselects the selected item at selectedChildIndex by clicking it.
func (p PlatformNode) DeselectSelectedChild(selectedChildIndex int) (bool, error) {
	return resolveForSelectionWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		child, ok := nthSelectedItem(node, selectedChildIndex)
		if !ok {
			return false, ErrorDefunct
		}
		if child.IsClickable(filter) {
			if err := doLocatedAction(tree, context, child.ID(), a11y.ActionClick, nil); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
}

// IsChildSelected reports whether the filtered child at childIndex is a selected item.
func (p PlatformNode) IsChildSelected(childIndex int) (bool, error) {
	return resolveForSelection(p, func(node *consumer.Node) (bool, error) {
		child, ok := nthFilteredChild(node, childIndex)
		if !ok {
			return false, ErrorDefunct
		}
		return child.IsItemLike() && isSelectedTrue(child), nil
	})
}

// SelectAll is unsupported and returns false.
func (p PlatformNode) SelectAll() (bool, error) {
	return false, nil
}

// ClearSelection is unsupported and returns false.
func (p PlatformNode) ClearSelection() (bool, error) {
	return false, nil
}

// DeselectChild deselects the filtered child at childIndex by clicking it.
func (p PlatformNode) DeselectChild(childIndex int) (bool, error) {
	return resolveForSelectionWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		child, ok := nthFilteredChild(node, childIndex)
		if !ok {
			return false, ErrorDefunct
		}
		if selected, has := child.IsSelected(); has && !selected {
			return true, nil
		}
		if child.IsSelectable() && child.IsClickable(filter) {
			if err := doLocatedAction(tree, context, child.ID(), a11y.ActionClick, nil); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
}

func usvOffset(pos consumer.Position) (int32, error) {
	return toI32(pos.ToGlobalUsvIndex(), ErrorTooManyCharacters)
}

// CharacterCount returns the number of characters.
func (p PlatformNode) CharacterCount() (int32, error) {
	return resolveForText(p, func(node *consumer.Node) (int32, error) {
		return usvOffset(node.DocumentRange().End())
	})
}

// CaretOffset returns the caret offset, or -1.
func (p PlatformNode) CaretOffset() (int32, error) {
	return resolveForText(p, func(node *consumer.Node) (int32, error) {
		focus, ok := node.TextSelectionFocus()
		if !ok {
			return -1, nil
		}
		return usvOffset(focus)
	})
}

// StringAtOffset returns the text unit at offset and its start and end offsets.
func (p PlatformNode) StringAtOffset(offset int32, granularity Granularity) (string, int32, int32, error) {
	type result struct {
		text       string
		start, end int32
	}
	r, err := resolveForText(p, func(node *consumer.Node) (result, error) {
		rng, err := textRangeFromOffset(node, offset, granularity)
		if err != nil {
			return result{}, err
		}
		text := rng.Text()
		start, err := usvOffset(rng.Start())
		if err != nil {
			return result{}, err
		}
		end, err := usvOffset(rng.End())
		if err != nil {
			return result{}, err
		}
		return result{text, start, end}, nil
	})
	return r.text, r.start, r.end, err
}

// Text returns the text between the offsets; an end offset of -1 means the end of the text.
func (p PlatformNode) Text(startOffset, endOffset int32) (string, error) {
	return resolveForText(p, func(node *consumer.Node) (string, error) {
		rng, ok := textRangeFromOffsets(node, startOffset, endOffset)
		if !ok {
			return "", ErrorIndexOutOfRange
		}
		return rng.Text(), nil
	})
}

// SetCaretOffset moves the caret.
func (p PlatformNode) SetCaretOffset(offset int32) (bool, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		pos, ok := textPositionFromOffset(node, offset)
		if !ok {
			return false, ErrorIndexOutOfRange
		}
		data := a11y.ActionDataSetTextSelection(pos.ToDegenerateRange().ToTextSelection())
		if err := doLocatedAction(tree, context, node.ID(), a11y.ActionSetTextSelection, data); err != nil {
			return false, err
		}
		return true, nil
	})
}

// TextAttributeValue returns one text attribute at offset, or "".
func (p PlatformNode) TextAttributeValue(offset int32, attributeName string) (string, error) {
	return resolveForText(p, func(node *consumer.Node) (string, error) {
		pos, ok := textPositionFromOffset(node, offset)
		if !ok {
			return "", ErrorIndexOutOfRange
		}
		if getter, ok := attributeGetters[attributeName]; ok {
			value, _ := getter(pos.InnerNode())
			return value, nil
		}
		return "", nil
	})
}

// TextAttributes returns the non-default attributes of the run at offset and its bounds.
func (p PlatformNode) TextAttributes(offset int32) (map[string]string, int32, int32, error) {
	return p.TextAttributeRun(offset, false)
}

// DefaultTextAttributes returns the attributes of the text node itself.
func (p PlatformNode) DefaultTextAttributes() (map[string]string, error) {
	return resolveForText(p, func(node *consumer.Node) (map[string]string, error) {
		result := make(map[string]string, len(attributeGetters))
		for name, getter := range attributeGetters {
			if value, ok := getter(*node); ok {
				result[name] = value
			}
		}
		return result, nil
	})
}

// CharacterExtents returns the extents of the character at offset, or RectInvalid.
func (p PlatformNode) CharacterExtents(offset int32, coordType CoordType) (Rect, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (Rect, error) {
		rng, err := textRangeFromOffset(node, offset, GranularityChar)
		if err != nil {
			return Rect{}, err
		}
		boxes := rng.BoundingBoxes()
		if len(boxes) == 0 {
			return RectInvalid(), nil
		}
		bounds := boxes[0]
		windowBounds := context.readRootWindowBounds()
		newOrigin := windowBounds.accesskitPointToAtspiPoint(bounds.Origin(), node, coordType)
		return RectFromAccessKit(bounds.WithOrigin(newOrigin)), nil
	})
}

// OffsetAtPoint returns the character offset at the point.
func (p PlatformNode) OffsetAtPoint(x, y int32, coordType CoordType) (int32, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (int32, error) {
		windowBounds := context.readRootWindowBounds()
		point := windowBounds.atspiPointToAccesskitPoint(a11y.NewPoint(float64(x), float64(y)), node, coordType)
		point = node.Transform().Inverse().MulPoint(point)
		return usvOffset(node.TextPositionAtPoint(point))
	})
}

func nonDegenerateSelection(node *consumer.Node) (consumer.Range, bool) {
	rng, ok := node.TextSelection()
	if !ok || rng.IsDegenerate() {
		return consumer.Range{}, false
	}
	return rng, true
}

// NSelections returns 1 if there is a non-empty text selection, else 0.
func (p PlatformNode) NSelections() (int32, error) {
	return resolveForText(p, func(node *consumer.Node) (int32, error) {
		if _, ok := nonDegenerateSelection(node); ok {
			return 1, nil
		}
		return 0, nil
	})
}

// Selection returns the offsets of selection selectionNum, or (-1, -1).
func (p PlatformNode) Selection(selectionNum int32) (int32, int32, error) {
	if selectionNum != 0 {
		return -1, -1, nil
	}
	type result struct{ start, end int32 }
	r, err := resolveForText(p, func(node *consumer.Node) (result, error) {
		rng, ok := nonDegenerateSelection(node)
		if !ok {
			return result{-1, -1}, nil
		}
		start, err := usvOffset(rng.Start())
		if err != nil {
			return result{}, err
		}
		end, err := usvOffset(rng.End())
		if err != nil {
			return result{}, err
		}
		return result{start, end}, nil
	})
	return r.start, r.end, err
}

// AddSelection sets the only supported selection.
func (p PlatformNode) AddSelection(startOffset, endOffset int32) (bool, error) {
	return p.SetSelection(0, startOffset, endOffset)
}

// RemoveSelection collapses the selection to the caret, or to the start of the text.
func (p PlatformNode) RemoveSelection(selectionNum int32) (bool, error) {
	if selectionNum != 0 {
		return false, nil
	}
	return resolveForTextWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		selectionEnd, ok := node.TextSelectionFocus()
		if !ok {
			selectionEnd = node.DocumentRange().Start()
		}
		data := a11y.ActionDataSetTextSelection(selectionEnd.ToDegenerateRange().ToTextSelection())
		if err := doLocatedAction(tree, context, node.ID(), a11y.ActionSetTextSelection, data); err != nil {
			return false, err
		}
		return true, nil
	})
}

// SetSelection sets selection selectionNum; only 0 is supported.
func (p PlatformNode) SetSelection(selectionNum, startOffset, endOffset int32) (bool, error) {
	if selectionNum != 0 {
		return false, nil
	}
	return resolveForTextWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		rng, ok := textRangeFromOffsets(node, startOffset, endOffset)
		if !ok {
			return false, ErrorIndexOutOfRange
		}
		if err := doLocatedAction(tree, context, node.ID(), a11y.ActionSetTextSelection, a11y.ActionDataSetTextSelection(rng.ToTextSelection())); err != nil {
			return false, err
		}
		return true, nil
	})
}

// RangeExtents returns the bounds of the text between the offsets, or RectInvalid.
func (p PlatformNode) RangeExtents(startOffset, endOffset int32, coordType CoordType) (Rect, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, _ *consumer.Tree, context *Context) (Rect, error) {
		rect, ok := textRangeBoundsFromOffsets(node, startOffset, endOffset)
		if !ok {
			return RectInvalid(), nil
		}
		windowBounds := context.readRootWindowBounds()
		newOrigin := windowBounds.accesskitPointToAtspiPoint(rect.Origin(), node, coordType)
		return RectFromAccessKit(rect.WithOrigin(newOrigin)), nil
	})
}

// TextAttributeRun returns the attributes of the run at offset and its start and end offsets.
func (p PlatformNode) TextAttributeRun(offset int32, includeDefaults bool) (map[string]string, int32, int32, error) {
	type result struct {
		attrs      map[string]string
		start, end int32
	}
	r, err := resolveForText(p, func(node *consumer.Node) (result, error) {
		pos, ok := textPositionFromOffset(node, offset)
		if !ok {
			return result{}, ErrorIndexOutOfRange
		}
		attrs := make(map[string]string, len(attributeGetters))
		for name, getter := range attributeGetters {
			value, ok := getter(pos.InnerNode())
			if !ok {
				continue
			}
			if !includeDefaults {
				if def, ok := getter(*node); ok && value == def {
					continue
				}
			}
			attrs[name] = value
		}
		start := pos
		if !pos.IsFormatStart() {
			start = pos.BackwardToFormatStart()
		}
		end := pos.ForwardToFormatEnd()
		startOffset, err := usvOffset(start)
		if err != nil {
			return result{}, err
		}
		endOffset, err := usvOffset(end)
		if err != nil {
			return result{}, err
		}
		return result{attrs, startOffset, endOffset}, nil
	})
	return r.attrs, r.start, r.end, err
}

// ScrollSubstringTo scrolls the text between the offsets into view.
func (p PlatformNode) ScrollSubstringTo(startOffset, endOffset int32, scrollType ScrollType) (bool, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		rng, ok := textRangeFromOffsets(node, startOffset, endOffset)
		if !ok {
			return false, nil
		}
		position := rng.Start()
		if scrollType == ScrollTypeBottomRight || scrollType == ScrollTypeBottomEdge || scrollType == ScrollTypeRightEdge {
			position = rng.End()
		}
		if err := doLocatedAction(tree, context, position.InnerNode().ID(), a11y.ActionScrollIntoView, scrollHintData(scrollType)); err != nil {
			return false, err
		}
		return true, nil
	})
}

// ScrollSubstringToPoint scrolls the text between the offsets to the point.
func (p PlatformNode) ScrollSubstringToPoint(startOffset, endOffset int32, coordType CoordType, x, y int32) (bool, error) {
	return resolveForTextWithContext(p, func(node *consumer.Node, tree *consumer.Tree, context *Context) (bool, error) {
		windowBounds := context.readRootWindowBounds()
		targetPoint := windowBounds.atspiPointToAccesskitPoint(a11y.NewPoint(float64(x), float64(y)), node, coordType)
		if rect, ok := textRangeBoundsFromOffsets(node, startOffset, endOffset); ok {
			point := a11y.NewPoint(targetPoint.X-rect.X0, targetPoint.Y-rect.Y0)
			if err := doLocatedAction(tree, context, node.ID(), a11y.ActionScrollToPoint, a11y.ActionDataScrollToPoint(point)); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	})
}

// MinimumValue returns the minimum value, or the most negative float64.
func (p PlatformNode) MinimumValue() (float64, error) {
	return resolve(p, func(node *consumer.Node) (float64, error) {
		if v, ok := node.MinNumericValue(); ok {
			return v, nil
		}
		return -math.MaxFloat64, nil
	})
}

// MaximumValue returns the maximum value, or the largest float64.
func (p PlatformNode) MaximumValue() (float64, error) {
	return resolve(p, func(node *consumer.Node) (float64, error) {
		if v, ok := node.MaxNumericValue(); ok {
			return v, nil
		}
		return math.MaxFloat64, nil
	})
}

// MinimumIncrement returns the value step, or 0.
func (p PlatformNode) MinimumIncrement() (float64, error) {
	return resolve(p, func(node *consumer.Node) (float64, error) {
		v, _ := node.NumericValueStep()
		return v, nil
	})
}

// CurrentValue returns the numeric value, or 0.
func (p PlatformNode) CurrentValue() (float64, error) {
	return resolve(p, func(node *consumer.Node) (float64, error) {
		v, _ := nodeWrapper{node}.currentValue()
		return v, nil
	})
}

// SetCurrentValue requests a new numeric value.
func (p PlatformNode) SetCurrentValue(value float64) error {
	return p.doActionInternal(p.id, simpleRequest(a11y.ActionSetValue, a11y.ActionDataNumericValue(value)))
}

// Equal reports whether p and other are the same node of the same adapter.
func (p PlatformNode) Equal(other PlatformNode) bool {
	return p.adapterID == other.adapterID && p.id == other.id
}

// PlatformNodeKey is the hashable identity of a PlatformNode.
type PlatformNodeKey struct {
	AdapterID int
	ID        consumer.NodeID
}

// Key returns the node's identity for use as a map key.
func (p PlatformNode) Key() PlatformNodeKey {
	return PlatformNodeKey{AdapterID: p.adapterID, ID: p.id}
}

// PlatformRoot is the AT-SPI application object.
type PlatformRoot struct {
	appContext weak.Pointer[AppContext]
}

// NewPlatformRoot returns the root object for appContext.
func NewPlatformRoot(appContext *AppContext) PlatformRoot {
	return PlatformRoot{appContext: weak.Make(appContext)}
}

func resolveAppContext[T any](r PlatformRoot, f func(*AppContext) (T, error)) (T, error) {
	appContext := r.appContext.Value()
	if appContext == nil {
		var zero T
		return zero, ErrorDefunct
	}
	appContext.lock.RLock()
	defer appContext.lock.RUnlock()
	return f(appContext)
}

// Name returns the application name.
func (r PlatformRoot) Name() (string, error) {
	return resolveAppContext(r, func(c *AppContext) (string, error) {
		return c.name.UnwrapOrDefault(), nil
	})
}

// Description returns "".
func (r PlatformRoot) Description() (string, error) {
	return "", nil
}

// ChildCount returns the number of adapters.
func (r PlatformRoot) ChildCount() (int32, error) {
	return resolveAppContext(r, func(c *AppContext) (int32, error) {
		return toI32(len(c.adapters), ErrorTooManyChildren)
	})
}

// Interfaces returns Accessible and Application.
func (r PlatformRoot) Interfaces() InterfaceSet {
	return NewInterfaceSet(InterfaceAccessible | InterfaceApplication)
}

// Role returns RoleApplication.
func (r PlatformRoot) Role() Role {
	return RoleApplication
}

// IndexInParent returns -1.
func (r PlatformRoot) IndexInParent() int32 {
	return -1
}

// State returns the empty set.
func (r PlatformRoot) State() StateSet {
	return StateSetEmpty()
}

// ChildAtIndex returns the root node of the adapter at index.
func (r PlatformRoot) ChildAtIndex(index int) (PlatformNode, bool, error) {
	type result struct {
		node PlatformNode
		ok   bool
	}
	res, err := resolveAppContext(r, func(c *AppContext) (result, error) {
		if index < 0 || index >= len(c.adapters) {
			return result{}, nil
		}
		return result{platformNodeFromAdapterRoot(c.adapters[index]), true}, nil
	})
	return res.node, res.ok, err
}

func adapterRootID(entry adapterEntry) consumer.NodeID {
	tree, done := entry.context.readTree()
	defer done()
	return tree.State().RootID()
}

// AdapterRoot is an adapter ID and the ID of its tree's root.
type AdapterRoot struct {
	AdapterID int
	ID        consumer.NodeID
}

// ChildIDAtIndex returns the adapter ID and root node ID at index.
func (r PlatformRoot) ChildIDAtIndex(index int) (AdapterRoot, bool, error) {
	type result struct {
		root AdapterRoot
		ok   bool
	}
	res, err := resolveAppContext(r, func(c *AppContext) (result, error) {
		if index < 0 || index >= len(c.adapters) {
			return result{}, nil
		}
		entry := c.adapters[index]
		return result{AdapterRoot{entry.id, adapterRootID(entry)}, true}, nil
	})
	return res.root, res.ok, err
}

// MapChildren maps f over each adapter's root node.
func (r PlatformRoot) MapChildren[I any](f func(PlatformNode) I) ([]I, error) {
	return resolveAppContext(r, func(c *AppContext) ([]I, error) {
		children := make([]I, 0, len(c.adapters))
		for _, entry := range c.adapters {
			children = append(children, f(platformNodeFromAdapterRoot(entry)))
		}
		return children, nil
	})
}

// MapChildIDs maps f over each adapter's ID and root node ID.
func (r PlatformRoot) MapChildIDs[I any](f func(AdapterRoot) I) ([]I, error) {
	return resolveAppContext(r, func(c *AppContext) ([]I, error) {
		children := make([]I, 0, len(c.adapters))
		for _, entry := range c.adapters {
			children = append(children, f(AdapterRoot{entry.id, adapterRootID(entry)}))
		}
		return children, nil
	})
}

// ToolkitName returns the toolkit name.
func (r PlatformRoot) ToolkitName() (string, error) {
	return resolveAppContext(r, func(c *AppContext) (string, error) {
		return c.toolkitName.UnwrapOrDefault(), nil
	})
}

// ToolkitVersion returns the toolkit version.
func (r PlatformRoot) ToolkitVersion() (string, error) {
	return resolveAppContext(r, func(c *AppContext) (string, error) {
		return c.toolkitVersion.UnwrapOrDefault(), nil
	})
}

// ID returns the application ID assigned by the registry, or -1.
func (r PlatformRoot) ID() (int32, error) {
	return resolveAppContext(r, func(c *AppContext) (int32, error) {
		return c.id.UnwrapOr(-1), nil
	})
}

// SetID stores the application ID assigned by the registry.
func (r *PlatformRoot) SetID(id int32) error {
	appContext := r.appContext.Value()
	if appContext == nil {
		return ErrorDefunct
	}
	appContext.lock.Lock()
	defer appContext.lock.Unlock()
	appContext.id.Replace(id)
	return nil
}

func collectDescendants[I any](node consumer.Node, indexInParent, adapterID int, context *Context, f func(PlatformNode, int, int) I, results *[]I) {
	var children []consumer.Node
	for child := range node.FilteredChildren(filter).All() {
		children = append(children, child)
	}
	platformNode := newPlatformNode(context, adapterID, node.ID())
	*results = append(*results, f(platformNode, indexInParent, len(children)))
	for childIndex, child := range children {
		collectDescendants(child, childIndex, adapterID, context, f, results)
	}
}

// MapDescendants maps f over every filtered node of every adapter, depth first, with its index in parent and child count.
func (r PlatformRoot) MapDescendants[I any](f func(PlatformNode, int, int) I) ([]I, error) {
	return resolveAppContext(r, func(c *AppContext) ([]I, error) {
		var results []I
		for adapterSlot, entry := range c.adapters {
			tree, done := entry.context.readTree()
			collectDescendants(tree.State().Root(), adapterSlot, entry.id, entry.context, f, &results)
			done()
		}
		return results, nil
	})
}

func collectDescendantCacheNodes[I any](node consumer.Node, indexInParent int32, adapterID int, windowFocused bool, f func(CacheNode) I, results *[]I) error {
	var children []consumer.Node
	for child := range node.FilteredChildren(filter).All() {
		children = append(children, child)
	}
	childCount, err := toI32(len(children), ErrorTooManyChildren)
	if err != nil {
		return err
	}
	*results = append(*results, f(nodeWrapper{&node}.cacheNode(adapterID, indexInParent, childCount, windowFocused)))
	for childIndex, child := range children {
		index, err := toI32(childIndex, ErrorIndexOutOfRange)
		if err != nil {
			return err
		}
		if err := collectDescendantCacheNodes(child, index, adapterID, windowFocused, f, results); err != nil {
			return err
		}
	}
	return nil
}

// MapDescendantCacheNodes maps f over the cache entry of every filtered node of every adapter, depth first.
func (r PlatformRoot) MapDescendantCacheNodes[I any](f func(CacheNode) I) ([]I, error) {
	return resolveAppContext(r, func(c *AppContext) ([]I, error) {
		var results []I
		for adapterSlot, entry := range c.adapters {
			tree, done := entry.context.readTree()
			state := tree.State()
			indexInParent, err := toI32(adapterSlot, ErrorIndexOutOfRange)
			if err != nil {
				done()
				return nil, err
			}
			_, windowFocused := state.FocusID()
			err = collectDescendantCacheNodes(state.Root(), indexInParent, entry.id, windowFocused, f, &results)
			done()
			if err != nil {
				return nil, err
			}
		}
		return results, nil
	})
}

// Equal reports whether r and other refer to the same application context.
func (r PlatformRoot) Equal(other PlatformRoot) bool {
	return r.appContext == other.appContext
}

// NodeIDOrRoot is either a node ID or the application root.
type NodeIDOrRoot struct {
	node   consumer.NodeID
	isNode bool
}

// NodeIDOrRootNode returns the variant holding id.
func NodeIDOrRootNode(id consumer.NodeID) NodeIDOrRoot {
	return NodeIDOrRoot{node: id, isNode: true}
}

// NodeIDOrRootRoot returns the root variant.
func NodeIDOrRootRoot() NodeIDOrRoot {
	return NodeIDOrRoot{}
}

// Node returns the node ID, or false for the root variant.
func (v NodeIDOrRoot) Node() (consumer.NodeID, bool) {
	return v.node, v.isNode
}

// IsRoot reports whether v is the root variant.
func (v NodeIDOrRoot) IsRoot() bool {
	return !v.isNode
}

// CacheNode is one entry of the AT-SPI cache.
type CacheNode struct {
	AdapterID     int
	ID            consumer.NodeID
	Parent        NodeIDOrRoot
	IndexInParent int32
	ChildCount    int32
	Interfaces    InterfaceSet
	Name          string
	Description   string
	Role          Role
	States        StateSet
}
