//go:build darwin && !ios

package macos

import (
	"runtime"
	"sync"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const scrollToVisibleAction = "AXScrollToVisible"

func nsRole(node consumer.Node) objc.ID {
	role := node.Role()
	switch role {
	case a11y.RoleUnknown:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleTextRun:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleCell, a11y.RoleGridCell:
		return axConst("NSAccessibilityCellRole")
	case a11y.RoleLabel:
		return axConst("NSAccessibilityStaticTextRole")
	case a11y.RoleImage:
		return axConst("NSAccessibilityImageRole")
	case a11y.RoleLink:
		return axConst("NSAccessibilityLinkRole")
	case a11y.RoleRow:
		return axConst("NSAccessibilityRowRole")
	case a11y.RoleListItem:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleListMarker:
		return nsString("AXListMarker")
	case a11y.RoleTreeItem:
		return axConst("NSAccessibilityRowRole")
	case a11y.RoleListBoxOption:
		return axConst("NSAccessibilityStaticTextRole")
	case a11y.RoleMenuItem:
		return axConst("NSAccessibilityMenuItemRole")
	case a11y.RoleMenuListOption:
		return axConst("NSAccessibilityMenuItemRole")
	case a11y.RoleParagraph:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleGenericContainer:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleCheckBox:
		return axConst("NSAccessibilityCheckBoxRole")
	case a11y.RoleRadioButton:
		return axConst("NSAccessibilityRadioButtonRole")
	case a11y.RoleTextInput, a11y.RoleSearchInput, a11y.RoleEmailInput, a11y.RoleNumberInput, a11y.RolePasswordInput, a11y.RolePhoneNumberInput, a11y.RoleUrlInput:
		return axConst("NSAccessibilityTextFieldRole")
	case a11y.RoleButton:
		if _, ok := node.Toggled(); ok {
			return axConst("NSAccessibilityCheckBoxRole")
		}
		return axConst("NSAccessibilityButtonRole")
	case a11y.RoleDefaultButton:
		return axConst("NSAccessibilityButtonRole")
	case a11y.RolePane:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleRowHeader:
		return axConst("NSAccessibilityCellRole")
	case a11y.RoleColumnHeader:
		return axConst("NSAccessibilityCellRole")
	case a11y.RoleRowGroup:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleList:
		return axConst("NSAccessibilityListRole")
	case a11y.RoleTable:
		return axConst("NSAccessibilityTableRole")
	case a11y.RoleLayoutTableCell:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleLayoutTableRow:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleLayoutTable:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSwitch:
		return axConst("NSAccessibilityCheckBoxRole")
	case a11y.RoleMenu:
		return axConst("NSAccessibilityMenuRole")
	case a11y.RoleMultilineTextInput:
		return axConst("NSAccessibilityTextAreaRole")
	case a11y.RoleDateInput, a11y.RoleDateTimeInput, a11y.RoleWeekInput, a11y.RoleMonthInput:
		return nsString("AXDateField")
	case a11y.RoleTimeInput:
		return nsString("AXTimeField")
	case a11y.RoleAbbr:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleAlert:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleAlertDialog:
		return axConst("NSAccessibilityWindowRole")
	case a11y.RoleApplication:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleArticle:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleAudio:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleBanner:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleBlockquote:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleCanvas:
		return axConst("NSAccessibilityImageRole")
	case a11y.RoleCaption:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleCaret:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleCode:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleColorWell:
		return axConst("NSAccessibilityColorWellRole")
	case a11y.RoleComboBox:
		return axConst("NSAccessibilityPopUpButtonRole")
	case a11y.RoleEditableComboBox:
		return axConst("NSAccessibilityComboBoxRole")
	case a11y.RoleComplementary:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleComment:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleContentDeletion:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleContentInsertion:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleContentInfo:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDefinition:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDescriptionList:
		return axConst("NSAccessibilityListRole")
	case a11y.RoleDetails:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDialog:
		return axConst("NSAccessibilityWindowRole")
	case a11y.RoleDisclosureTriangle:
		return axConst("NSAccessibilityButtonRole")
	case a11y.RoleDocument:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleEmbeddedObject:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleEmphasis:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleFeed:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleFigureCaption:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleFigure:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleFooter:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleForm:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleGrid:
		return axConst("NSAccessibilityTableRole")
	case a11y.RoleGroup:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleHeader:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleHeading:
		return nsString("Heading")
	case a11y.RoleIframe:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleIframePresentational:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleImeCandidate:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleKeyboard:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleLegend:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleLineBreak:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleListBox:
		return axConst("NSAccessibilityListRole")
	case a11y.RoleLog:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleMain:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleMark:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleMarquee:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleMath:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleMenuBar:
		return axConst("NSAccessibilityMenuBarRole")
	case a11y.RoleMenuItemCheckBox:
		return axConst("NSAccessibilityMenuItemRole")
	case a11y.RoleMenuItemRadio:
		return axConst("NSAccessibilityMenuItemRole")
	case a11y.RoleMenuListPopup:
		return axConst("NSAccessibilityMenuRole")
	case a11y.RoleMeter:
		return axConst("NSAccessibilityLevelIndicatorRole")
	case a11y.RoleNavigation:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleNote:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RolePluginObject:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleProgressIndicator:
		return axConst("NSAccessibilityProgressIndicatorRole")
	case a11y.RoleRadioGroup:
		return axConst("NSAccessibilityRadioGroupRole")
	case a11y.RoleRegion:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleRootWebArea:
		return nsString("AXWebArea")
	case a11y.RoleRuby:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleRubyAnnotation:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleScrollBar:
		return axConst("NSAccessibilityScrollBarRole")
	case a11y.RoleScrollView:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleSearch:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSection:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSectionFooter:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSectionHeader:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSlider:
		return axConst("NSAccessibilitySliderRole")
	case a11y.RoleSpinButton:
		return axConst("NSAccessibilityIncrementorRole")
	case a11y.RoleSplitter:
		return axConst("NSAccessibilitySplitterRole")
	case a11y.RoleStatus:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleStrong:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSuggestion:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleSvgRoot:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTab:
		return axConst("NSAccessibilityRadioButtonRole")
	case a11y.RoleTabList:
		return axConst("NSAccessibilityTabGroupRole")
	case a11y.RoleTabPanel:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTerm:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTime:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTimer:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTitleBar:
		return axConst("NSAccessibilityStaticTextRole")
	case a11y.RoleToolbar:
		return axConst("NSAccessibilityToolbarRole")
	case a11y.RoleTooltip:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleTree:
		return axConst("NSAccessibilityOutlineRole")
	case a11y.RoleTreeGrid:
		return axConst("NSAccessibilityTableRole")
	case a11y.RoleVideo:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleWebView:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleWindow:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RolePdfActionableHighlight:
		return axConst("NSAccessibilityButtonRole")
	case a11y.RolePdfRoot:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleGraphicsDocument:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleGraphicsObject:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleGraphicsSymbol:
		return axConst("NSAccessibilityImageRole")
	case a11y.RoleDocAbstract:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocAcknowledgements:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocAfterword:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocAppendix:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocBackLink:
		return axConst("NSAccessibilityLinkRole")
	case a11y.RoleDocBiblioEntry:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocBibliography:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocBiblioRef:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocChapter:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocColophon:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocConclusion:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocCover:
		return axConst("NSAccessibilityImageRole")
	case a11y.RoleDocCredit:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocCredits:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocDedication:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocEndnote:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocEndnotes:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocEpigraph:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocEpilogue:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocErrata:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocExample:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocFootnote:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocForeword:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocGlossary:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocGlossRef:
		return axConst("NSAccessibilityLinkRole")
	case a11y.RoleDocIndex:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocIntroduction:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocNoteRef:
		return axConst("NSAccessibilityLinkRole")
	case a11y.RoleDocNotice:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPageBreak:
		return axConst("NSAccessibilitySplitterRole")
	case a11y.RoleDocPageFooter:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPageHeader:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPageList:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPart:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPreface:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPrologue:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocPullquote:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocQna:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocSubtitle:
		return nsString("AXHeading")
	case a11y.RoleDocTip:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleDocToc:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleListGrid:
		return axConst("NSAccessibilityUnknownRole")
	case a11y.RoleTerminal:
		return axConst("NSAccessibilityTextAreaRole")
	}
	return axConst("NSAccessibilityUnknownRole")
}

func nsSubRole(node consumer.Node) objc.ID {
	role := node.Role()
	if role == a11y.RoleButton {
		if _, ok := node.Toggled(); ok {
			return axConst("NSAccessibilityToggleSubrole")
		}
	}
	switch role {
	case a11y.RoleAlert:
		return nsString("AXApplicationAlert")
	case a11y.RoleAlertDialog:
		return axConst("NSAccessibilityDialogSubrole")
	case a11y.RoleArticle:
		return nsString("AXDocumentArticle")
	case a11y.RoleBanner:
		return nsString("AXLandmarkBanner")
	case a11y.RoleCode:
		return nsString("AXCodeStyleGroup")
	case a11y.RoleComplementary:
		return nsString("AXLandmarkComplementary")
	case a11y.RoleContentDeletion:
		return nsString("AXDeleteStyleGroup")
	case a11y.RoleContentInsertion:
		return nsString("AXInsertStyleGroup")
	case a11y.RoleContentInfo:
		return nsString("AXLandmarkContentInfo")
	case a11y.RoleDefinition:
		return nsString("AXDefinition")
	case a11y.RoleDialog:
		return axConst("NSAccessibilityDialogSubrole")
	case a11y.RoleDocument:
		return nsString("AXDocument")
	case a11y.RoleEmphasis:
		return nsString("AXEmphasisStyleGroup")
	case a11y.RoleFeed:
		return nsString("AXApplicationGroup")
	case a11y.RoleFooter:
		return nsString("AXLandmarkContentInfo")
	case a11y.RoleForm:
		return nsString("AXLandmarkForm")
	case a11y.RoleGraphicsDocument:
		return nsString("AXDocument")
	case a11y.RoleGroup:
		return nsString("AXApplicationGroup")
	case a11y.RoleHeader:
		return nsString("AXLandmarkBanner")
	case a11y.RoleLayoutTableCell:
		return axConst("NSAccessibilityGroupRole")
	case a11y.RoleLayoutTableRow:
		return axConst("NSAccessibilityTableRowSubrole")
	case a11y.RoleLog:
		return nsString("AXApplicationLog")
	case a11y.RoleMain:
		return nsString("AXLandmarkMain")
	case a11y.RoleMarquee:
		return nsString("AXApplicationMarquee")
	case a11y.RoleMath:
		return nsString("AXDocumentMath")
	case a11y.RoleMeter:
		return nsString("AXMeter")
	case a11y.RoleNavigation:
		return nsString("AXLandmarkNavigation")
	case a11y.RoleNote:
		return nsString("AXDocumentNote")
	case a11y.RolePasswordInput:
		return axConst("NSAccessibilitySecureTextFieldSubrole")
	case a11y.RoleRegion:
		return nsString("AXLandmarkRegion")
	case a11y.RoleSearch:
		return nsString("AXLandmarkSearch")
	case a11y.RoleSearchInput:
		return axConst("NSAccessibilitySearchFieldSubrole")
	case a11y.RoleSectionFooter:
		return nsString("AXSectionFooter")
	case a11y.RoleSectionHeader:
		return nsString("AXSectionHeader")
	case a11y.RoleStatus:
		return nsString("AXApplicationStatus")
	case a11y.RoleStrong:
		return nsString("AXStrongStyleGroup")
	case a11y.RoleSwitch:
		return axConst("NSAccessibilitySwitchSubrole")
	case a11y.RoleTab:
		return axConst("NSAccessibilityTabButtonSubrole")
	case a11y.RoleTabPanel:
		return nsString("AXTabPanel")
	case a11y.RoleTerm:
		return nsString("AXTerm")
	case a11y.RoleTime:
		return nsString("AXTimeGroup")
	case a11y.RoleTimer:
		return nsString("AXApplicationTimer")
	case a11y.RoleTreeItem:
		return axConst("NSAccessibilityOutlineRowSubrole")
	case a11y.RoleTooltip:
		return nsString("AXUserInterfaceTooltip")
	}
	return axConst("NSAccessibilityUnknownSubrole")
}

func canBeFocused(node *consumer.Node) bool {
	return filter(node) == consumer.FilterResultInclude && node.Role() != a11y.RoleWindow
}

type valueKind uint8

const (
	valueNone valueKind = iota
	valueBool
	valueNumber
	valueString
)

// Value is the value a node exposes through accessibilityValue.
type Value struct {
	kind   valueKind
	bool   bool
	number float64
	string string
}

func (v Value) isSome() bool {
	return v.kind != valueNone
}

type nodeWrapper struct {
	node *consumer.Node
}

func (w nodeWrapper) isRoot() bool {
	return w.node.IsRoot()
}

func (w nodeWrapper) title() (string, bool) {
	if w.isRoot() && w.node.Role() == a11y.RoleWindow {
		return "", false
	}
	return w.node.Label()
}

func (w nodeWrapper) description() (string, bool) {
	return w.node.Description()
}

func (w nodeWrapper) placeholder() (string, bool) {
	return w.node.Placeholder()
}

func (w nodeWrapper) value() Value {
	if toggled, ok := w.node.Toggled(); ok {
		return Value{kind: valueBool, bool: toggled != a11y.ToggledFalse}
	}
	if w.node.Role() == a11y.RoleTab {
		selected, _ := w.node.IsSelected()
		return Value{kind: valueBool, bool: selected}
	}
	if value, ok := w.node.Value(); ok {
		return Value{kind: valueString, string: value}
	}
	if value, ok := w.node.NumericValue(); ok {
		return Value{kind: valueNumber, number: value}
	}
	return Value{}
}

func (w nodeWrapper) supportsTextRanges() bool {
	return w.node.SupportsTextRanges()
}

func (w nodeWrapper) rawTextSelection() (a11y.TextSelection, bool) {
	return w.node.RawTextSelection()
}

func (w nodeWrapper) isContainerWithSelectableChildren() bool {
	return w.node.IsContainerWithSelectableChildren() && w.node.Role() != a11y.RoleTabList
}

func (w nodeWrapper) isItemLike() bool {
	return w.node.IsItemLike() && w.node.Role() != a11y.RoleTab
}

func downcastRef(obj objc.ID, class string) bool {
	return obj != objc.Nil && objc.Send[bool](obj, objc.Sel("isKindOfClass:"), objc.GetClass(class))
}

type platformNodeIvars struct {
	context weak.Pointer[Context]
	nodeID  consumer.NodeID
}

var (
	platformNodeIvarsMu sync.Mutex
	platformNodeIvarMap = map[objc.ID]*platformNodeIvars{}
)

func ivarsOf(this objc.ID) *platformNodeIvars {
	platformNodeIvarsMu.Lock()
	defer platformNodeIvarsMu.Unlock()
	return platformNodeIvarMap[this]
}

var (
	platformNodeClassOnce sync.Once
	platformNodeClassID   objc.Class
)

func platformNodeClass() objc.Class {
	platformNodeClassOnce.Do(registerPlatformNodeClass)
	return platformNodeClassID
}

func registerPlatformNodeClass() {
	mustLoadRuntime()
	class, err := objc.RegisterClass("AccessKitNode", objc.GetClass("NSAccessibilityElement"), nil, nil, []objc.MethodDef{
		{Cmd: objc.Sel("dealloc"), Fn: platformNodeDealloc},
		{Cmd: objc.Sel("accessibilityParent"), Fn: platformNodeParent},
		{Cmd: objc.Sel("accessibilityWindow"), Fn: platformNodeWindow},
		{Cmd: objc.Sel("accessibilityTopLevelUIElement"), Fn: platformNodeTopLevel},
		{Cmd: objc.Sel("accessibilityChildren"), Fn: platformNodeChildren},
		{Cmd: objc.Sel("accessibilityChildrenInNavigationOrder"), Fn: platformNodeChildrenInNavigationOrder},
		{Cmd: objc.Sel("accessibilitySelectedChildren"), Fn: platformNodeSelectedChildren},
		{Cmd: objc.Sel("accessibilityFrame"), Fn: platformNodeFrame},
		{Cmd: objc.Sel("accessibilityRole"), Fn: platformNodeRole},
		{Cmd: objc.Sel("accessibilitySubrole"), Fn: platformNodeSubRole},
		{Cmd: objc.Sel("accessibilityRoleDescription"), Fn: platformNodeRoleDescription},
		{Cmd: objc.Sel("accessibilityIdentifier"), Fn: platformNodeIdentifier},
		{Cmd: objc.Sel("accessibilityTitle"), Fn: platformNodeTitle},
		{Cmd: objc.Sel("accessibilityHelp"), Fn: platformNodeDescription},
		{Cmd: objc.Sel("accessibilityPlaceholderValue"), Fn: platformNodePlaceholder},
		{Cmd: objc.Sel("accessibilityValue"), Fn: platformNodeValue},
		{Cmd: objc.Sel("setAccessibilityValue:"), Fn: platformNodeSetValue},
		{Cmd: objc.Sel("accessibilityMinValue"), Fn: platformNodeMinValue},
		{Cmd: objc.Sel("accessibilityMaxValue"), Fn: platformNodeMaxValue},
		{Cmd: objc.Sel("accessibilityURL"), Fn: platformNodeURL},
		{Cmd: objc.Sel("accessibilityOrientation"), Fn: platformNodeOrientation},
		{Cmd: objc.Sel("isAccessibilityElement"), Fn: platformNodeIsAccessibilityElement},
		{Cmd: objc.Sel("isAccessibilityFocused"), Fn: platformNodeIsFocused},
		{Cmd: objc.Sel("isAccessibilityEnabled"), Fn: platformNodeIsEnabled},
		{Cmd: objc.Sel("setAccessibilityFocused:"), Fn: platformNodeSetFocused},
		{Cmd: objc.Sel("accessibilityPerformPress"), Fn: platformNodePress},
		{Cmd: objc.Sel("accessibilityPerformIncrement"), Fn: platformNodeIncrement},
		{Cmd: objc.Sel("accessibilityPerformDecrement"), Fn: platformNodeDecrement},
		{Cmd: objc.Sel("accessibilityNotifiesWhenDestroyed"), Fn: platformNodeNotifiesWhenDestroyed},
		{Cmd: objc.Sel("accessibilityNumberOfCharacters"), Fn: platformNodeNumberOfCharacters},
		{Cmd: objc.Sel("accessibilitySelectedText"), Fn: platformNodeSelectedText},
		{Cmd: objc.Sel("accessibilitySelectedTextRange"), Fn: platformNodeSelectedTextRange},
		{Cmd: objc.Sel("accessibilityInsertionPointLineNumber"), Fn: platformNodeInsertionPointLineNumber},
		{Cmd: objc.Sel("accessibilityRangeForLine:"), Fn: platformNodeRangeForLine},
		{Cmd: objc.Sel("accessibilityRangeForPosition:"), Fn: platformNodeRangeForPosition},
		{Cmd: objc.Sel("accessibilityStringForRange:"), Fn: platformNodeStringForRange},
		{Cmd: objc.Sel("accessibilityAttributedStringForRange:"), Fn: platformNodeAttributedStringForRange},
		{Cmd: objc.Sel("accessibilityFrameForRange:"), Fn: platformNodeFrameForRange},
		{Cmd: objc.Sel("accessibilityLineForIndex:"), Fn: platformNodeLineForIndex},
		{Cmd: objc.Sel("accessibilityRangeForIndex:"), Fn: platformNodeRangeForIndex},
		{Cmd: objc.Sel("accessibilityStyleRangeForIndex:"), Fn: platformNodeStyleRangeForIndex},
		{Cmd: objc.Sel("setAccessibilitySelectedTextRange:"), Fn: platformNodeSetSelectedTextRange},
		{Cmd: objc.Sel("isAccessibilityRequired"), Fn: platformNodeIsRequired},
		{Cmd: objc.Sel("isAccessibilitySelected"), Fn: platformNodeIsSelected},
		{Cmd: objc.Sel("setAccessibilitySelected:"), Fn: platformNodeSetSelected},
		{Cmd: objc.Sel("accessibilityAttributeValue:"), Fn: platformNodeAccessibilityAttributeValue},
		{Cmd: objc.Sel("accessibilityRows"), Fn: platformNodeRows},
		{Cmd: objc.Sel("accessibilitySelectedRows"), Fn: platformNodeSelectedRows},
		{Cmd: objc.Sel("accessibilityPerformPick"), Fn: platformNodePick},
		{Cmd: objc.Sel("accessibilityLinkedUIElements"), Fn: platformNodeLinkedUIElements},
		{Cmd: objc.Sel("accessibilityTabs"), Fn: platformNodeTabs},
		{Cmd: objc.Sel("isAccessibilityModal"), Fn: platformNodeIsModal},
		{Cmd: objc.Sel("accessibilityActionNames"), Fn: platformNodeActionNames},
		{Cmd: objc.Sel("accessibilityPerformAction:"), Fn: platformNodePerformAction},
		{Cmd: objc.Sel("isAccessibilitySelectorAllowed:"), Fn: platformNodeIsSelectorAllowed},
	})
	if err != nil {
		panic(err)
	}
	platformNodeClassID = class
}

func newPlatformNode(context weak.Pointer[Context], nodeID consumer.NodeID) objc.ID {
	this := objc.Send[objc.ID](objc.Alloc(platformNodeClass()), objc.Sel("init"))
	platformNodeIvarsMu.Lock()
	platformNodeIvarMap[this] = &platformNodeIvars{context: context, nodeID: nodeID}
	platformNodeIvarsMu.Unlock()
	return this
}

func platformNodeDealloc(this objc.ID, _ objc.SEL) {
	platformNodeIvarsMu.Lock()
	delete(platformNodeIvarMap, this)
	platformNodeIvarsMu.Unlock()
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

func resolveWithContext[T any](this objc.ID, f func(node *consumer.Node, tree *consumer.Tree, context *Context) T) (T, bool) {
	var zero T
	ivars := ivarsOf(this)
	if ivars == nil {
		return zero, false
	}
	context, ok := upgradeContext(ivars.context)
	if !ok {
		return zero, false
	}
	tree, unlock := context.readTree()
	defer unlock()
	node, ok := tree.State().NodeByID(ivars.nodeID)
	if !ok {
		return zero, false
	}
	return f(&node, tree, context), true
}

func resolve[T any](this objc.ID, f func(node *consumer.Node) T) (T, bool) {
	return resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, _ *Context) T {
		return f(node)
	})
}

func autoreleaseReturn(id objc.ID) objc.ID {
	return objc.Autorelease(objc.Retain(id))
}

func nsArrayAutoreleased(ids []objc.ID) objc.ID {
	return objc.Autorelease(objc.NSArrayFromIDs(ids))
}

func flatten(id objc.ID, _ bool) objc.ID {
	return id
}

func viewAccessibilityParent(context *Context) objc.ID {
	view := context.view.load()
	if view == objc.Nil {
		return objc.Nil
	}
	defer objc.Release(view)
	return autoreleaseReturn(objc.Send[objc.ID](view, objc.Sel("accessibilityParent")))
}

func platformNodeParent(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		if parent, ok := node.FilteredParent(filter); ok {
			return autoreleaseReturn(context.getOrCreatePlatformNode(parent.ID()))
		}
		return viewAccessibilityParent(context)
	}))
}

func platformNodeWindow(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(_ *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		return viewAccessibilityParent(context)
	}))
}

func platformNodeTopLevel(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(_ *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		return viewAccessibilityParent(context)
	}))
}

func platformNodeChildren(this objc.ID, _ objc.SEL) objc.ID {
	return childrenInternal(this)
}

func platformNodeChildrenInNavigationOrder(this objc.ID, _ objc.SEL) objc.ID {
	return childrenInternal(this)
}

func platformNodeSelectedChildren(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		wrapper := nodeWrapper{node}
		if !wrapper.isContainerWithSelectableChildren() {
			return objc.Nil
		}
		var platformNodes []objc.ID
		for item := range node.Items(filter).All() {
			if selected, ok := item.IsSelected(); ok && selected {
				platformNodes = append(platformNodes, context.getOrCreatePlatformNode(item.ID()))
			}
		}
		return nsArrayAutoreleased(platformNodes)
	}))
}

func platformNodeFrame(this objc.ID, _ objc.SEL) objc.CGRect {
	rect, ok := resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.CGRect {
		view := context.view.load()
		if view == objc.Nil {
			return objc.CGRect{}
		}
		defer objc.Release(view)
		if rect, ok := node.BoundingBox(); ok {
			return toNSRect(view, rect)
		}
		if node.IsRoot() {
			return objc.Send[objc.CGRect](view, objc.Sel("accessibilityFrame"))
		}
		return objc.CGRect{}
	})
	if !ok {
		return objc.CGRect{}
	}
	return rect
}

func platformNodeRole(this objc.ID, _ objc.SEL) objc.ID {
	role, ok := resolve(this, func(node *consumer.Node) objc.ID { return nsRole(*node) })
	if !ok {
		return axConst("NSAccessibilityUnknownRole")
	}
	return role
}

func platformNodeSubRole(this objc.ID, _ objc.SEL) objc.ID {
	subRole, ok := resolve(this, func(node *consumer.Node) objc.ID { return nsSubRole(*node) })
	if !ok {
		return axConst("NSAccessibilityUnknownSubrole")
	}
	return subRole
}

func platformNodeRoleDescription(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if roleDescription, ok := node.RoleDescription(); ok {
			return nsStringAutoreleased(roleDescription)
		}
		return objc.SendSuper[objc.ID](this, objc.Sel("accessibilityRoleDescription"))
	}))
}

func platformNodeIdentifier(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if authorID, ok := node.AuthorID(); ok {
			return nsStringAutoreleased(authorID)
		}
		return objc.Nil
	}))
}

func platformNodeTitle(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if title, ok := (nodeWrapper{node}).title(); ok {
			return nsStringAutoreleased(title)
		}
		return objc.Nil
	}))
}

func platformNodeDescription(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if description, ok := (nodeWrapper{node}).description(); ok {
			return nsStringAutoreleased(description)
		}
		return objc.Nil
	}))
}

func platformNodePlaceholder(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if placeholder, ok := (nodeWrapper{node}).placeholder(); ok {
			return nsStringAutoreleased(placeholder)
		}
		return objc.Nil
	}))
}

func nsNumberWithBool(value bool) objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithBool:"), value)
}

func nsNumberWithDouble(value float64) objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithDouble:"), value)
}

func nsNumberWithFloat(value float32) objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithFloat:"), value)
}

func nsNumberWithInteger(value objc.NSInteger) objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithInteger:"), value)
}

func platformNodeValue(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		value := nodeWrapper{node}.value()
		switch value.kind {
		case valueBool:
			return nsNumberWithBool(value.bool)
		case valueNumber:
			return nsNumberWithDouble(value.number)
		case valueString:
			return nsStringAutoreleased(value.string)
		}
		return objc.Nil
	}))
}

func doActionOn(node *consumer.Node, tree *consumer.Tree, context *Context, action a11y.Action, data a11y.ActionData) {
	if targetNode, targetTree, ok := tree.State().LocateNode(node.ID()); ok {
		context.doAction(a11y.ActionRequest{
			Action:     action,
			TargetTree: targetTree,
			TargetNode: targetNode,
			Data:       data,
		})
	}
}

func platformNodeSetValue(this objc.ID, _ objc.SEL, value objc.ID) {
	if downcastRef(value, "NSString") {
		resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
			doActionOn(node, tree, context, a11y.ActionSetValue, a11y.ActionDataValue(objc.GoString(value)))
			return struct{}{}
		})
	} else if downcastRef(value, "NSNumber") {
		resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
			number := objc.Send[float64](value, objc.Sel("doubleValue"))
			doActionOn(node, tree, context, a11y.ActionSetValue, a11y.ActionDataNumericValue(number))
			return struct{}{}
		})
	}
}

func platformNodeMinValue(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if value, ok := node.MinNumericValue(); ok {
			return nsNumberWithDouble(value)
		}
		return objc.Nil
	}))
}

func platformNodeMaxValue(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if value, ok := node.MaxNumericValue(); ok {
			return nsNumberWithDouble(value)
		}
		return objc.Nil
	}))
}

func platformNodeURL(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if !node.SupportsURL() {
			return objc.Nil
		}
		url, ok := node.URL()
		if !ok {
			return objc.Nil
		}
		nsString := objc.NSStringFromGo(url)
		defer objc.Release(nsString)
		return objc.SendClass[objc.ID](objc.GetClass("NSURL"), objc.Sel("URLWithString:"), nsString)
	}))
}

const (
	nsAccessibilityOrientationUnknown    objc.NSInteger = 0
	nsAccessibilityOrientationVertical   objc.NSInteger = 1
	nsAccessibilityOrientationHorizontal objc.NSInteger = 2
)

func platformNodeOrientation(this objc.ID, _ objc.SEL) objc.NSInteger {
	orientation, ok := resolve(this, func(node *consumer.Node) objc.NSInteger {
		orientation, ok := node.Orientation()
		switch {
		case ok && orientation == a11y.OrientationHorizontal:
			return nsAccessibilityOrientationHorizontal
		case ok && orientation == a11y.OrientationVertical:
			return nsAccessibilityOrientationVertical
		}
		return nsAccessibilityOrientationUnknown
	})
	if !ok {
		return nsAccessibilityOrientationUnknown
	}
	return orientation
}

func resolveBool(this objc.ID, f func(node *consumer.Node) bool) bool {
	result, ok := resolve(this, f)
	return ok && result
}

func platformNodeIsAccessibilityElement(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool { return filter(node) == consumer.FilterResultInclude })
}

func platformNodeIsFocused(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool { return node.IsFocused() && canBeFocused(node) })
}

func platformNodeIsEnabled(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool { return !node.IsDisabled() })
}

func platformNodeSetFocused(this objc.ID, _ objc.SEL, focused bool) {
	resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
		if focused {
			if node.IsFocusable(filter) {
				doActionOn(node, tree, context, a11y.ActionFocus, nil)
			}
		} else {
			root := tree.State().Root()
			if root.IsFocusable(filter) {
				doActionOn(&root, tree, context, a11y.ActionFocus, nil)
			}
		}
		return struct{}{}
	})
}

func resolveActionBool(this objc.ID, supported func(node *consumer.Node) bool, action a11y.Action) bool {
	result, ok := resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) bool {
		s := supported(node)
		if s {
			doActionOn(node, tree, context, action, nil)
		}
		return s
	})
	return ok && result
}

func platformNodePress(this objc.ID, _ objc.SEL) bool {
	return resolveActionBool(this, func(node *consumer.Node) bool { return node.IsClickable(filter) }, a11y.ActionClick)
}

func platformNodeIncrement(this objc.ID, _ objc.SEL) bool {
	return resolveActionBool(this, func(node *consumer.Node) bool { return node.SupportsIncrement(filter) }, a11y.ActionIncrement)
}

func platformNodeDecrement(this objc.ID, _ objc.SEL) bool {
	return resolveActionBool(this, func(node *consumer.Node) bool { return node.SupportsDecrement(filter) }, a11y.ActionDecrement)
}

func platformNodeNotifiesWhenDestroyed(_ objc.ID, _ objc.SEL) bool {
	return true
}

func platformNodeNumberOfCharacters(this objc.ID, _ objc.SEL) objc.NSInteger {
	n, _ := resolve(this, func(node *consumer.Node) objc.NSInteger {
		if node.SupportsTextRanges() {
			return objc.NSInteger(node.DocumentRange().End().ToGlobalUTF16Index())
		}
		return 0
	})
	return n
}

func platformNodeSelectedText(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if node.SupportsTextRanges() {
			if r, ok := node.TextSelection(); ok {
				return nsStringAutoreleased(r.Text())
			}
		}
		return objc.Nil
	}))
}

func platformNodeSelectedTextRange(this objc.ID, _ objc.SEL) objc.NSRange {
	r, _ := resolve(this, func(node *consumer.Node) objc.NSRange {
		if node.SupportsTextRanges() {
			if r, ok := node.TextSelection(); ok {
				return toNSRange(r)
			}
		}
		return objc.MakeNSRange(0, 0)
	})
	return r
}

func platformNodeInsertionPointLineNumber(this objc.ID, _ objc.SEL) objc.NSInteger {
	n, _ := resolve(this, func(node *consumer.Node) objc.NSInteger {
		if node.SupportsTextRanges() {
			if pos, ok := node.TextSelectionFocus(); ok {
				return objc.NSInteger(pos.ToLineIndex())
			}
		}
		return 0
	})
	return n
}

func platformNodeRangeForLine(this objc.ID, _ objc.SEL, lineIndex objc.NSInteger) objc.NSRange {
	r, _ := resolve(this, func(node *consumer.Node) objc.NSRange {
		if node.SupportsTextRanges() && lineIndex >= 0 {
			if r, ok := node.LineRangeFromIndex(int(lineIndex)); ok {
				return toNSRange(r)
			}
		}
		return objc.MakeNSRange(0, 0)
	})
	return r
}

func platformNodeRangeForPosition(this objc.ID, _ objc.SEL, point objc.CGPoint) objc.NSRange {
	r, _ := resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.NSRange {
		view := context.view.load()
		if view == objc.Nil {
			return objc.MakeNSRange(0, 0)
		}
		defer objc.Release(view)
		if node.SupportsTextRanges() {
			point := fromNSPoint(view, *node, point)
			pos := node.TextPositionAtPoint(point)
			return toNSRangeForCharacter(pos)
		}
		return objc.MakeNSRange(0, 0)
	})
	return r
}

func platformNodeStringForRange(this objc.ID, _ objc.SEL, nsRange objc.NSRange) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if node.SupportsTextRanges() {
			if r, ok := fromNSRange(*node, nsRange); ok {
				return nsStringAutoreleased(r.Text())
			}
		}
		return objc.Nil
	}))
}

func nsTextAlignment(align a11y.TextAlign) objc.NSInteger {
	usesIOSValues := runtime.GOARCH != "amd64"
	switch align {
	case a11y.TextAlignCenter:
		if usesIOSValues {
			return 1
		}
		return 2
	case a11y.TextAlignRight:
		if usesIOSValues {
			return 2
		}
		return 1
	case a11y.TextAlignJustify:
		return 3
	}
	return 0
}

func dictionarySet(dictionary, key, value objc.ID) {
	objc.SendVoid(dictionary, objc.Sel("setObject:forKey:"), value, key)
}

func dictionarySetOwned(dictionary, key, value objc.ID) {
	dictionarySet(dictionary, key, value)
	objc.Release(value)
}

func platformNodeAttributedStringForRange(this objc.ID, _ objc.SEL, nsRange objc.NSRange) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if !node.SupportsTextRanges() {
			return objc.Nil
		}
		r, ok := fromNSRange(*node, nsRange)
		if !ok {
			return objc.Nil
		}
		result := objc.AllocInit(objc.GetClass("NSMutableAttributedString"))
		objc.SendVoid(result, objc.Sel("beginEditing"))
		r.TraverseText(func(node *consumer.Node, text string) (struct{}, bool) {
			objc.AutoreleasePool(func() {
				nsText := objc.NSStringFromGo(text)
				defer objc.Release(nsText)
				attrs := objc.AllocInit(objc.GetClass("NSMutableDictionary"))
				defer objc.Release(attrs)
				if color, ok := node.BackgroundColor(); ok {
					dictionarySetOwned(attrs, axConst("NSAccessibilityBackgroundColorTextAttribute"), toColorAttribute(color))
				}
				if color, ok := node.ForegroundColor(); ok {
					dictionarySetOwned(attrs, axConst("NSAccessibilityForegroundColorTextAttribute"), toColorAttribute(color))
				}
				fontAttrs := objc.AllocInit(objc.GetClass("NSMutableDictionary"))
				defer objc.Release(fontAttrs)
				if family, ok := node.FontFamily(); ok {
					dictionarySet(fontAttrs, axConst("NSAccessibilityFontFamilyKey"), nsStringAutoreleased(family))
				}
				if size, ok := node.FontSize(); ok {
					dictionarySet(fontAttrs, axConst("NSAccessibilityFontSizeKey"), nsNumberWithFloat(size))
				}
				if weight, ok := node.FontWeight(); ok {
					if weight >= 700.0 {
						dictionarySet(fontAttrs, nsString("AXFontBold"), nsNumberWithBool(true))
					}
				}
				if node.IsItalic() {
					dictionarySet(fontAttrs, nsString("AXFontItalic"), nsNumberWithBool(true))
				}
				if objc.Send[objc.NSUInteger](fontAttrs, objc.Sel("count")) != 0 {
					dictionarySet(attrs, axConst("NSAccessibilityFontTextAttribute"), fontAttrs)
				}
				if deco, ok := node.Underline(); ok {
					dictionarySet(attrs, axConst("NSAccessibilityUnderlineTextAttribute"), nsNumberWithBool(true))
					dictionarySetOwned(attrs, axConst("NSAccessibilityUnderlineColorTextAttribute"), toColorAttribute(deco.Color))
				}
				if deco, ok := node.Strikethrough(); ok {
					dictionarySet(attrs, axConst("NSAccessibilityStrikethroughTextAttribute"), nsNumberWithBool(true))
					dictionarySetOwned(attrs, axConst("NSAccessibilityStrikethroughColorTextAttribute"), toColorAttribute(deco.Color))
				}
				if language, ok := node.Language(); ok {
					dictionarySet(attrs, axConst("NSAccessibilityLanguageTextAttribute"), nsStringAutoreleased(language))
				}
				if align, ok := node.TextAlign(); ok {
					dictionarySet(attrs, axConst("NSAccessibilityTextAlignmentAttribute"), nsNumberWithInteger(nsTextAlignment(align)))
				}
				part := objc.Send[objc.ID](objc.Alloc(objc.GetClass("NSAttributedString")), objc.Sel("initWithString:attributes:"), nsText, attrs)
				defer objc.Release(part)
				objc.SendVoid(result, objc.Sel("appendAttributedString:"), part)
			})
			return struct{}{}, false
		})
		objc.SendVoid(result, objc.Sel("endEditing"))
		return objc.Autorelease(result)
	}))
}

func platformNodeFrameForRange(this objc.ID, _ objc.SEL, nsRange objc.NSRange) objc.CGRect {
	rect, _ := resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.CGRect {
		view := context.view.load()
		if view == objc.Nil {
			return objc.CGRect{}
		}
		defer objc.Release(view)
		if node.SupportsTextRanges() {
			if r, ok := fromNSRange(*node, nsRange); ok {
				rects := r.BoundingBoxes()
				if len(rects) > 0 {
					rect := rects[0]
					for _, other := range rects[1:] {
						rect = rect.Union(other)
					}
					return toNSRect(view, rect)
				}
			}
		}
		return objc.CGRect{}
	})
	return rect
}

func platformNodeLineForIndex(this objc.ID, _ objc.SEL, index objc.NSInteger) objc.NSInteger {
	n, _ := resolve(this, func(node *consumer.Node) objc.NSInteger {
		if node.SupportsTextRanges() && index >= 0 {
			if pos, ok := node.TextPositionFromGlobalUTF16Index(int(index)); ok {
				return objc.NSInteger(pos.ToLineIndex())
			}
		}
		return 0
	})
	return n
}

func platformNodeRangeForIndex(this objc.ID, _ objc.SEL, index objc.NSInteger) objc.NSRange {
	r, _ := resolve(this, func(node *consumer.Node) objc.NSRange {
		if node.SupportsTextRanges() && index >= 0 {
			if pos, ok := node.TextPositionFromGlobalUTF16Index(int(index)); ok {
				return toNSRangeForCharacter(pos)
			}
		}
		return objc.MakeNSRange(0, 0)
	})
	return r
}

func platformNodeStyleRangeForIndex(this objc.ID, _ objc.SEL, index objc.NSInteger) objc.NSRange {
	r, _ := resolve(this, func(node *consumer.Node) objc.NSRange {
		if node.SupportsTextRanges() && index >= 0 {
			if pos, ok := node.TextPositionFromGlobalUTF16Index(int(index)); ok {
				start := pos
				if !pos.IsFormatStart() {
					start = pos.BackwardToFormatStart()
				}
				r := start.ToDegenerateRange()
				r.SetEnd(pos.ForwardToFormatEnd())
				return toNSRange(r)
			}
		}
		return objc.MakeNSRange(0, 0)
	})
	return r
}

func platformNodeSetSelectedTextRange(this objc.ID, _ objc.SEL, nsRange objc.NSRange) {
	resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
		if node.SupportsTextRanges() {
			if r, ok := fromNSRange(*node, nsRange); ok {
				doActionOn(node, tree, context, a11y.ActionSetTextSelection, a11y.ActionDataSetTextSelection(r.ToTextSelection()))
			}
		}
		return struct{}{}
	})
}

func platformNodeIsRequired(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool { return node.IsRequired() })
}

func platformNodeIsSelected(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool {
		selected, _ := node.IsSelected()
		return nodeWrapper{node}.isItemLike() && node.IsSelectable() && selected
	})
}

func platformNodeSetSelected(this objc.ID, _ objc.SEL, selected bool) {
	resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
		wrapper := nodeWrapper{node}
		if !node.IsClickable(filter) || !wrapper.isItemLike() || !node.IsSelectable() {
			return struct{}{}
		}
		if current, ok := node.IsSelected(); ok && current == selected {
			return struct{}{}
		}
		doActionOn(node, tree, context, a11y.ActionClick, nil)
		return struct{}{}
	})
}

func platformNodeAccessibilityAttributeValue(this objc.ID, _ objc.SEL, attr objc.ID) objc.ID {
	return flatten(resolve(this, func(node *consumer.Node) objc.ID {
		if nsStringEqual(attr, nsString("AXBrailleLabel")) && node.HasBrailleLabel() {
			label, _ := node.BrailleLabel()
			return nsStringAutoreleased(label)
		} else if nsStringEqual(attr, nsString("AXBrailleRoleDescription")) && node.HasBrailleRoleDescription() {
			description, _ := node.BrailleRoleDescription()
			return nsStringAutoreleased(description)
		}
		return objc.Nil
	}))
}

func platformNodeRows(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		if !(nodeWrapper{node}).isContainerWithSelectableChildren() {
			return objc.Nil
		}
		var platformNodes []objc.ID
		for child := range node.Items(filter).All() {
			platformNodes = append(platformNodes, context.getOrCreatePlatformNode(child.ID()))
		}
		return nsArrayAutoreleased(platformNodes)
	}))
}

func platformNodeSelectedRows(this objc.ID, _ objc.SEL) objc.ID {
	return platformNodeSelectedChildren(this, 0)
}

func platformNodePick(this objc.ID, _ objc.SEL) bool {
	return resolveActionBool(this, func(node *consumer.Node) bool {
		return node.IsClickable(filter) && nodeWrapper{node}.isItemLike() && node.IsSelectable()
	}, a11y.ActionClick)
}

func platformNodeLinkedUIElements(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		var platformNodes []objc.ID
		for controlled := range node.Controls() {
			if filter(&controlled) == consumer.FilterResultInclude {
				platformNodes = append(platformNodes, context.getOrCreatePlatformNode(controlled.ID()))
			}
		}
		if len(platformNodes) == 0 {
			return objc.Nil
		}
		return nsArrayAutoreleased(platformNodes)
	}))
}

func platformNodeTabs(this objc.ID, _ objc.SEL) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		if node.Role() != a11y.RoleTabList {
			return objc.Nil
		}
		var platformNodes []objc.ID
		for child := range node.FilteredChildren(filter).All() {
			if child.Role() == a11y.RoleTab {
				platformNodes = append(platformNodes, context.getOrCreatePlatformNode(child.ID()))
			}
		}
		return nsArrayAutoreleased(platformNodes)
	}))
}

func platformNodeIsModal(this objc.ID, _ objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool { return node.IsModal() })
}

func platformNodeActionNames(this objc.ID, _ objc.SEL) objc.ID {
	var result []objc.ID
	resolve(this, func(node *consumer.Node) struct{} {
		if node.SupportsAction(a11y.ActionScrollIntoView, filter) {
			result = append(result, nsString(scrollToVisibleAction))
		}
		return struct{}{}
	})
	return nsArrayAutoreleased(result)
}

func platformNodePerformAction(this objc.ID, _ objc.SEL, action objc.ID) {
	resolveWithContext(this, func(node *consumer.Node, tree *consumer.Tree, context *Context) struct{} {
		if nsStringEqual(action, nsString(scrollToVisibleAction)) {
			doActionOn(node, tree, context, a11y.ActionScrollIntoView, nil)
		}
		return struct{}{}
	})
}

func platformNodeIsSelectorAllowed(this objc.ID, _ objc.SEL, selector objc.SEL) bool {
	return resolveBool(this, func(node *consumer.Node) bool {
		is := func(names ...string) bool {
			for _, name := range names {
				if selector == objc.Sel(name) {
					return true
				}
			}
			return false
		}
		if is("setAccessibilityFocused:") {
			return node.IsFocusable(filter)
		}
		if is("accessibilityPerformPress") {
			return node.IsClickable(filter)
		}
		if is("accessibilityPerformIncrement") {
			return node.SupportsIncrement(filter)
		}
		if is("accessibilityPerformDecrement") {
			return node.SupportsDecrement(filter)
		}
		if is("accessibilityNumberOfCharacters",
			"accessibilitySelectedText",
			"accessibilitySelectedTextRange",
			"accessibilityInsertionPointLineNumber",
			"accessibilityRangeForLine:",
			"accessibilityRangeForPosition:",
			"accessibilityStringForRange:",
			"accessibilityAttributedStringForRange:",
			"accessibilityFrameForRange:",
			"accessibilityLineForIndex:",
			"accessibilityRangeForIndex:",
			"accessibilityStyleRangeForIndex:",
			"setAccessibilitySelectedTextRange:") {
			return node.SupportsTextRanges()
		}
		if is("setAccessibilityValue:") {
			return (node.SupportsTextRanges() && !node.IsReadOnly()) || node.SupportsAction(a11y.ActionSetValue, filter)
		}
		if is("isAccessibilitySelected") {
			return nodeWrapper{node}.isItemLike()
		}
		if is("accessibilityRows", "accessibilitySelectedRows") {
			return nodeWrapper{node}.isContainerWithSelectableChildren()
		}
		if is("setAccessibilitySelected:", "accessibilityPerformPick") {
			return node.IsClickable(filter) && nodeWrapper{node}.isItemLike() && node.IsSelectable()
		}
		if is("accessibilityTabs") {
			return node.Role() == a11y.RoleTabList
		}
		if is("isAccessibilityModal") {
			return node.IsDialog()
		}
		if is("accessibilityAttributeValue:") {
			return node.HasBrailleLabel() || node.HasBrailleRoleDescription()
		}
		if is("accessibilityURL") {
			return node.SupportsURL()
		}
		return is("accessibilityParent",
			"accessibilityChildren",
			"accessibilityChildrenInNavigationOrder",
			"accessibilitySelectedChildren",
			"accessibilityFrame",
			"accessibilityRole",
			"accessibilitySubrole",
			"isAccessibilityEnabled",
			"accessibilityWindow",
			"accessibilityTopLevelUIElement",
			"accessibilityLinkedUIElements",
			"accessibilityRoleDescription",
			"accessibilityIdentifier",
			"accessibilityTitle",
			"accessibilityHelp",
			"accessibilityPlaceholderValue",
			"accessibilityValue",
			"accessibilityMinValue",
			"accessibilityMaxValue",
			"isAccessibilityRequired",
			"accessibilityOrientation",
			"isAccessibilityElement",
			"isAccessibilityFocused",
			"accessibilityNotifiesWhenDestroyed",
			"isAccessibilitySelectorAllowed:",
			"accessibilityActionNames",
			"accessibilityPerformAction:")
	})
}

func childrenInternal(this objc.ID) objc.ID {
	return flatten(resolveWithContext(this, func(node *consumer.Node, _ *consumer.Tree, context *Context) objc.ID {
		var platformNodes []objc.ID
		for child := range node.FilteredChildren(filter).All() {
			platformNodes = append(platformNodes, context.getOrCreatePlatformNode(child.ID()))
		}
		return nsArrayAutoreleased(platformNodes)
	}))
}
