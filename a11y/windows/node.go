//go:build windows

package windows

import (
	"iter"
	"math"
	"strings"
	"sync"
	"weak"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/win/com"
)

const runtimeIDSize = 5

func runtimeIDFromNodeID(id consumer.NodeID) [runtimeIDSize]int32 {
	hi, lo := id.ToU128()
	return [runtimeIDSize]int32{
		uiaAppendRuntimeID,
		int32(uint32(hi >> 32)),
		int32(uint32(hi)),
		int32(uint32(lo >> 32)),
		int32(uint32(lo)),
	}
}

type propValue interface {
	toVariant(buffer *[]uint16) Variant
}

type optString struct {
	value string
	ok    bool
}

type wideValue string

type localeName struct {
	value string
	ok    bool
}

type i32Value int32

type optI32 struct {
	value int32
	ok    bool
}

type boolValue bool

type f64Value float64

func (v optString) toVariant(buffer *[]uint16) Variant {
	if !v.ok {
		return variantEmpty()
	}
	return variantFromString(v.value, buffer)
}

func (v wideValue) toVariant(buffer *[]uint16) Variant {
	return variantFromString(string(v), buffer)
}

func (v localeName) toVariant(*[]uint16) Variant {
	if !v.ok {
		return variantEmpty()
	}
	return variantFromLocaleName(v.value)
}

func (v i32Value) toVariant(*[]uint16) Variant {
	return variantFromI32(int32(v))
}

func (v optI32) toVariant(*[]uint16) Variant {
	if !v.ok {
		return variantEmpty()
	}
	return variantFromI32(v.value)
}

func (v boolValue) toVariant(*[]uint16) Variant {
	return variantFromBool(bool(v))
}

func (v f64Value) toVariant(*[]uint16) Variant {
	return variantFromF64(float64(v))
}

func unwrapWrite(ok bool, err error) bool {
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	return ok
}

func unwrapOption[T any](value T, ok bool) T {
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return value
}

type nodeWrapper struct {
	node         *consumer.Node
	stringBuffer *[]uint16
}

func (w *nodeWrapper) controlType() i32Value {
	switch w.node.Role() {
	case a11y.RoleUnknown:
		return i32Value(uiaCustomControlTypeID)
	case a11y.RoleTextRun:
		return i32Value(uiaCustomControlTypeID)
	case a11y.RoleCell, a11y.RoleGridCell:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleLabel:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleImage:
		return i32Value(uiaImageControlTypeID)
	case a11y.RoleLink:
		return i32Value(uiaHyperlinkControlTypeID)
	case a11y.RoleRow:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleListItem:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleListMarker:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleTreeItem:
		return i32Value(uiaTreeItemControlTypeID)
	case a11y.RoleListBoxOption:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleMenuItem:
		return i32Value(uiaMenuItemControlTypeID)
	case a11y.RoleMenuListOption:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleParagraph:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleGenericContainer:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleCheckBox:
		return i32Value(uiaCheckBoxControlTypeID)
	case a11y.RoleRadioButton:
		return i32Value(uiaRadioButtonControlTypeID)
	case a11y.RoleTextInput,
		a11y.RoleMultilineTextInput,
		a11y.RoleSearchInput,
		a11y.RoleDateInput,
		a11y.RoleDateTimeInput,
		a11y.RoleWeekInput,
		a11y.RoleMonthInput,
		a11y.RoleTimeInput,
		a11y.RoleEmailInput,
		a11y.RoleNumberInput,
		a11y.RolePasswordInput,
		a11y.RolePhoneNumberInput,
		a11y.RoleUrlInput:
		return i32Value(uiaEditControlTypeID)
	case a11y.RoleButton, a11y.RoleDefaultButton:
		return i32Value(uiaButtonControlTypeID)
	case a11y.RolePane:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleRowHeader:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleColumnHeader:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleRowGroup:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleList:
		return i32Value(uiaListControlTypeID)
	case a11y.RoleTable:
		return i32Value(uiaTableControlTypeID)
	case a11y.RoleLayoutTableCell:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleLayoutTableRow:
		return i32Value(uiaDataItemControlTypeID)
	case a11y.RoleLayoutTable:
		return i32Value(uiaTableControlTypeID)
	case a11y.RoleSwitch:
		return i32Value(uiaButtonControlTypeID)
	case a11y.RoleMenu:
		return i32Value(uiaMenuControlTypeID)
	case a11y.RoleAbbr:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleAlert:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleAlertDialog:
		return i32Value(uiaWindowControlTypeID)
	case a11y.RoleApplication:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleArticle:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleAudio:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleBanner:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleBlockquote:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleCanvas:
		return i32Value(uiaImageControlTypeID)
	case a11y.RoleCaption:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleCaret:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleCode:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleColorWell:
		return i32Value(uiaButtonControlTypeID)
	case a11y.RoleComboBox, a11y.RoleEditableComboBox:
		return i32Value(uiaComboBoxControlTypeID)
	case a11y.RoleComplementary:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleComment:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleContentDeletion:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleContentInsertion:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleContentInfo:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDefinition:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDescriptionList:
		return i32Value(uiaListControlTypeID)
	case a11y.RoleDetails:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDialog:
		return i32Value(uiaWindowControlTypeID)
	case a11y.RoleDisclosureTriangle:
		return i32Value(uiaButtonControlTypeID)
	case a11y.RoleDocument, a11y.RoleTerminal:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleEmbeddedObject:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleEmphasis:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleFeed:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleFigureCaption:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleFigure:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleFooter:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleForm:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleGrid:
		return i32Value(uiaDataGridControlTypeID)
	case a11y.RoleGroup:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleHeader:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleHeading:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleIframe:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleIframePresentational:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleImeCandidate:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleKeyboard:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleLegend:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleLineBreak:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleListBox:
		return i32Value(uiaListControlTypeID)
	case a11y.RoleLog:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleMain:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleMark:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleMarquee:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleMath:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleMenuBar:
		return i32Value(uiaMenuBarControlTypeID)
	case a11y.RoleMenuItemCheckBox:
		return i32Value(uiaCheckBoxControlTypeID)
	case a11y.RoleMenuItemRadio:
		return i32Value(uiaRadioButtonControlTypeID)
	case a11y.RoleMenuListPopup:
		return i32Value(uiaListControlTypeID)
	case a11y.RoleMeter:
		return i32Value(uiaProgressBarControlTypeID)
	case a11y.RoleNavigation:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleNote:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RolePluginObject:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleProgressIndicator:
		return i32Value(uiaProgressBarControlTypeID)
	case a11y.RoleRadioGroup:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleRegion:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleRootWebArea:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleRuby:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleRubyAnnotation:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleScrollBar:
		return i32Value(uiaScrollBarControlTypeID)
	case a11y.RoleScrollView:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleSearch:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleSection:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleSectionFooter:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleSectionHeader:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleSlider:
		return i32Value(uiaSliderControlTypeID)
	case a11y.RoleSpinButton:
		return i32Value(uiaSpinnerControlTypeID)
	case a11y.RoleSplitter:
		return i32Value(uiaSeparatorControlTypeID)
	case a11y.RoleStatus:
		return i32Value(uiaStatusBarControlTypeID)
	case a11y.RoleStrong:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleSuggestion:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleSvgRoot:
		return i32Value(uiaImageControlTypeID)
	case a11y.RoleTab:
		return i32Value(uiaTabItemControlTypeID)
	case a11y.RoleTabList:
		return i32Value(uiaTabControlTypeID)
	case a11y.RoleTabPanel:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleTerm:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleTime:
		return i32Value(uiaTextControlTypeID)
	case a11y.RoleTimer:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleTitleBar:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleToolbar:
		return i32Value(uiaToolBarControlTypeID)
	case a11y.RoleTooltip:
		return i32Value(uiaToolTipControlTypeID)
	case a11y.RoleTree:
		return i32Value(uiaTreeControlTypeID)
	case a11y.RoleTreeGrid:
		return i32Value(uiaDataGridControlTypeID)
	case a11y.RoleVideo:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleWebView:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleWindow:
		return i32Value(uiaWindowControlTypeID)
	case a11y.RolePdfActionableHighlight:
		return i32Value(uiaCustomControlTypeID)
	case a11y.RolePdfRoot:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleGraphicsDocument:
		return i32Value(uiaDocumentControlTypeID)
	case a11y.RoleGraphicsObject:
		return i32Value(uiaPaneControlTypeID)
	case a11y.RoleGraphicsSymbol:
		return i32Value(uiaImageControlTypeID)
	case a11y.RoleDocAbstract:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocAcknowledgements:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocAfterword:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocAppendix:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocBackLink:
		return i32Value(uiaHyperlinkControlTypeID)
	case a11y.RoleDocBiblioEntry:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleDocBibliography:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocBiblioRef:
		return i32Value(uiaHyperlinkControlTypeID)
	case a11y.RoleDocChapter:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocColophon:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocConclusion:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocCover:
		return i32Value(uiaImageControlTypeID)
	case a11y.RoleDocCredit:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocCredits:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocDedication:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocEndnote:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleDocEndnotes:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocEpigraph:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocEpilogue:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocErrata:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocExample:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocFootnote:
		return i32Value(uiaListItemControlTypeID)
	case a11y.RoleDocForeword:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocGlossary:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocGlossRef:
		return i32Value(uiaHyperlinkControlTypeID)
	case a11y.RoleDocIndex:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocIntroduction:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocNoteRef:
		return i32Value(uiaHyperlinkControlTypeID)
	case a11y.RoleDocNotice:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPageBreak:
		return i32Value(uiaSeparatorControlTypeID)
	case a11y.RoleDocPageFooter:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPageHeader:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPageList:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPart:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPreface:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPrologue:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocPullquote:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocQna:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocSubtitle:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocTip:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleDocToc:
		return i32Value(uiaGroupControlTypeID)
	case a11y.RoleListGrid:
		return i32Value(uiaDataGridControlTypeID)
	}
	panic("a11y/windows: unknown role")
}

func (w *nodeWrapper) localizedControlType() optString {
	s, ok := w.node.RoleDescription()
	return optString{value: s, ok: ok}
}

func (w *nodeWrapper) ariaRole() optString {
	var role string
	switch w.node.Role() {
	case a11y.RoleAlert:
		role = "alert"
	case a11y.RoleAlertDialog:
		role = "alertdialog"
	case a11y.RoleApplication:
		role = "application"
	case a11y.RoleArticle:
		role = "article"
	case a11y.RoleBanner, a11y.RoleHeader:
		role = "banner"
	case a11y.RoleButton, a11y.RoleDefaultButton:
		role = "button"
	case a11y.RoleBlockquote:
		role = "blockquote"
	case a11y.RoleCaption, a11y.RoleFigureCaption:
		role = "caption"
	case a11y.RoleCell:
		role = "cell"
	case a11y.RoleCheckBox:
		role = "checkbox"
	case a11y.RoleCode:
		role = "code"
	case a11y.RoleColumnHeader:
		role = "columnheader"
	case a11y.RoleComboBox, a11y.RoleEditableComboBox:
		role = "combobox"
	case a11y.RoleComment:
		role = "comment"
	case a11y.RoleComplementary:
		role = "complementary"
	case a11y.RoleContentInfo, a11y.RoleFooter:
		role = "contentinfo"
	case a11y.RoleDefinition:
		role = "definition"
	case a11y.RoleContentDeletion:
		role = "deletion"
	case a11y.RoleDialog:
		role = "dialog"
	case a11y.RoleDocument,
		a11y.RoleIframe,
		a11y.RoleWebView,
		a11y.RoleRootWebArea,
		a11y.RoleTerminal,
		a11y.RolePdfRoot:
		role = "document"
	case a11y.RoleEmphasis:
		role = "emphasis"
	case a11y.RoleFeed:
		role = "feed"
	case a11y.RoleFigure:
		role = "figure"
	case a11y.RoleForm:
		role = "form"
	case a11y.RoleGenericContainer:
		role = "generic"
	case a11y.RoleGraphicsDocument:
		role = "graphics-document"
	case a11y.RoleGraphicsObject:
		role = "graphics-object"
	case a11y.RoleGraphicsSymbol:
		role = "graphics-symbol"
	case a11y.RoleGrid, a11y.RoleListGrid:
		role = "grid"
	case a11y.RoleGridCell:
		role = "gridcell"
	case a11y.RoleGroup,
		a11y.RoleDetails,
		a11y.RoleIframePresentational,
		a11y.RoleTitleBar,
		a11y.RoleLayoutTable,
		a11y.RoleLayoutTableCell,
		a11y.RoleLayoutTableRow,
		a11y.RoleAudio,
		a11y.RoleVideo,
		a11y.RoleListMarker,
		a11y.RoleEmbeddedObject,
		a11y.RoleImeCandidate:
		role = "group"
	case a11y.RoleHeading:
		role = "heading"
	case a11y.RoleImage, a11y.RoleCanvas:
		role = "img"
	case a11y.RoleContentInsertion:
		role = "insertion"
	case a11y.RoleLink:
		role = "link"
	case a11y.RoleList, a11y.RoleDescriptionList, a11y.RoleMenuListPopup:
		role = "list"
	case a11y.RoleListBox:
		role = "listbox"
	case a11y.RoleListItem:
		role = "listitem"
	case a11y.RoleLog:
		role = "log"
	case a11y.RoleMain:
		role = "main"
	case a11y.RoleMark:
		role = "marker"
	case a11y.RoleMarquee:
		role = "marquee"
	case a11y.RoleMath:
		role = "math"
	case a11y.RoleMenu:
		role = "menu"
	case a11y.RoleMenuBar:
		role = "menubar"
	case a11y.RoleMenuItem:
		role = "menuitem"
	case a11y.RoleMenuItemCheckBox:
		role = "menuitemcheckbox"
	case a11y.RoleMenuItemRadio:
		role = "menuitemradio"
	case a11y.RoleMeter:
		role = "meter"
	case a11y.RoleNavigation:
		role = "navigation"
	case a11y.RoleNote:
		role = "note"
	case a11y.RoleListBoxOption, a11y.RoleMenuListOption:
		role = "option"
	case a11y.RoleParagraph:
		role = "paragraph"
	case a11y.RoleProgressIndicator:
		role = "progressbar"
	case a11y.RoleRadioButton:
		role = "radio"
	case a11y.RoleRadioGroup:
		role = "radiogroup"
	case a11y.RoleRegion,
		a11y.RolePane,
		a11y.RoleWindow,
		a11y.RoleKeyboard,
		a11y.RoleUnknown,
		a11y.RoleScrollView,
		a11y.RoleCaret:
		role = "region"
	case a11y.RoleRow:
		role = "row"
	case a11y.RoleRowGroup:
		role = "rowgroup"
	case a11y.RoleRowHeader:
		role = "rowheader"
	case a11y.RoleScrollBar:
		role = "scrollbar"
	case a11y.RoleSearch:
		role = "search"
	case a11y.RoleSearchInput:
		role = "searchbox"
	case a11y.RoleSectionFooter:
		role = "sectionfooter"
	case a11y.RoleSectionHeader:
		role = "sectionheader"
	case a11y.RoleSplitter:
		role = "separator"
	case a11y.RoleSlider:
		role = "slider"
	case a11y.RoleSpinButton:
		role = "spinbutton"
	case a11y.RoleStatus:
		role = "status"
	case a11y.RoleStrong:
		role = "strong"
	case a11y.RoleSuggestion:
		role = "suggestion"
	case a11y.RoleSwitch:
		role = "switch"
	case a11y.RoleTab:
		role = "tab"
	case a11y.RoleTable:
		role = "table"
	case a11y.RoleTabList:
		role = "tablist"
	case a11y.RoleTabPanel:
		role = "tabpanel"
	case a11y.RoleTerm:
		role = "term"
	case a11y.RoleTextInput,
		a11y.RoleMultilineTextInput,
		a11y.RoleDateInput,
		a11y.RoleDateTimeInput,
		a11y.RoleWeekInput,
		a11y.RoleMonthInput,
		a11y.RoleTimeInput,
		a11y.RoleEmailInput,
		a11y.RoleNumberInput,
		a11y.RolePasswordInput,
		a11y.RolePhoneNumberInput,
		a11y.RoleUrlInput,
		a11y.RoleColorWell:
		role = "textbox"
	case a11y.RoleTime:
		role = "time"
	case a11y.RoleTimer:
		role = "timer"
	case a11y.RoleToolbar:
		role = "toolbar"
	case a11y.RoleTooltip:
		role = "tooltip"
	case a11y.RoleTree:
		role = "tree"
	case a11y.RoleTreeGrid:
		role = "treegrid"
	case a11y.RoleTreeItem:
		role = "treeitem"
	default:
		return optString{}
	}
	return optString{value: role, ok: true}
}

func (w *nodeWrapper) name() optString {
	var result strings.Builder
	var ok bool
	if w.node.LabelComesFromValue() {
		ok = unwrapWrite(w.node.WriteValue(&result))
	} else {
		ok = unwrapWrite(w.node.WriteLabel(&result))
	}
	return optString{value: result.String(), ok: ok}
}

func (w *nodeWrapper) description() optString {
	s, ok := w.node.Description()
	return optString{value: s, ok: ok}
}

func (w *nodeWrapper) culture() localeName {
	s, ok := w.node.Language()
	return localeName{value: s, ok: ok}
}

func (w *nodeWrapper) placeholder() optString {
	s, ok := w.node.Placeholder()
	return optString{value: s, ok: ok}
}

func (w *nodeWrapper) isContentElement() boolValue {
	return filter(w.node) == consumer.FilterResultInclude
}

func (w *nodeWrapper) ariaProperties() optString {
	var result strings.Builder
	properties := newAriaProperties(&result)
	must := func(err error) {
		if err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
	}

	if w.node.IsLiveAtomic() {
		must(properties.writeBoolProperty("atomic", true))
	}

	if label, ok := w.node.BrailleLabel(); ok {
		must(properties.writeProperty("braillelabel", label))
	}

	if description, ok := w.node.BrailleRoleDescription(); ok {
		must(properties.writeProperty("brailleroledescription", description))
	}

	if w.node.IsBusy() {
		must(properties.writeBoolProperty("busy", true))
	}

	if colindextext, ok := w.node.ColumnIndexText(); ok {
		must(properties.writeProperty("colindextext", colindextext))
	}

	if current, ok := w.node.AriaCurrent(); ok {
		if current != a11y.AriaCurrentFalse {
			var value string
			switch current {
			case a11y.AriaCurrentTrue:
				value = "true"
			case a11y.AriaCurrentPage:
				value = "page"
			case a11y.AriaCurrentStep:
				value = "step"
			case a11y.AriaCurrentLocation:
				value = "location"
			case a11y.AriaCurrentDate:
				value = "date"
			case a11y.AriaCurrentTime:
				value = "time"
			default:
				panic("internal error: entered unreachable code")
			}
			must(properties.writeProperty("current", value))
		}
	}

	if hasPopup, ok := w.node.HasPopup(); ok {
		var value string
		switch hasPopup {
		case a11y.HasPopupMenu:
			value = "menu"
		case a11y.HasPopupListbox:
			value = "listbox"
		case a11y.HasPopupTree:
			value = "tree"
		case a11y.HasPopupGrid:
			value = "grid"
		case a11y.HasPopupDialog:
			value = "dialog"
		}
		must(properties.writeProperty("haspopup", value))
	}

	if level, ok := w.node.Level(); ok {
		must(properties.writeUsizeProperty("level", level))
	}

	if w.node.IsMultiline() {
		must(properties.writeBoolProperty("multiline", true))
	}

	if rowindextext, ok := w.node.RowIndexText(); ok {
		must(properties.writeProperty("rowindextext", rowindextext))
	}

	if sortDirection, ok := w.node.SortDirection(); ok {
		var value string
		switch sortDirection {
		case a11y.SortDirectionAscending:
			value = "ascending"
		case a11y.SortDirectionDescending:
			value = "descending"
		case a11y.SortDirectionOther:
			value = "other"
		}
		must(properties.writeProperty("sort", value))
	}

	if properties.hasProperties() {
		return optString{value: result.String(), ok: true}
	}
	return optString{}
}

func (w *nodeWrapper) isEnabled() boolValue {
	return boolValue(!w.node.IsDisabled())
}

func (w *nodeWrapper) isFocusable() boolValue {
	return boolValue(w.node.IsFocusable(filter))
}

func (w *nodeWrapper) isFocused() boolValue {
	return boolValue(w.node.IsFocused())
}

func (w *nodeWrapper) liveSetting() i32Value {
	switch w.node.Live() {
	case a11y.LivePolite:
		return liveSettingPolite
	case a11y.LiveAssertive:
		return liveSettingAssertive
	default:
		return liveSettingOff
	}
}

func (w *nodeWrapper) automationID() optString {
	s, ok := w.node.AuthorID()
	return optString{value: s, ok: ok}
}

func (w *nodeWrapper) className() optString {
	s, ok := w.node.ClassName()
	return optString{value: s, ok: ok}
}

func (w *nodeWrapper) orientation() i32Value {
	orientation, ok := w.node.Orientation()
	switch {
	case ok && orientation == a11y.OrientationHorizontal:
		return orientationTypeHorizontal
	case ok && orientation == a11y.OrientationVertical:
		return orientationTypeVertical
	default:
		return orientationTypeNone
	}
}

func (w *nodeWrapper) isTogglePatternSupported() bool {
	_, ok := w.node.Toggled()
	return ok && !w.isSelectionItemPatternSupported()
}

func (w *nodeWrapper) toggleState() i32Value {
	switch unwrapOption(w.node.Toggled()) {
	case a11y.ToggledTrue:
		return toggleStateOn
	case a11y.ToggledMixed:
		return toggleStateIndeterminate
	default:
		return toggleStateOff
	}
}

func (w *nodeWrapper) isInvokePatternSupported() bool {
	return w.node.IsInvocable(filter)
}

func (w *nodeWrapper) isValuePatternSupported() bool {
	if w.node.SupportsURL() {
		return true
	}
	return w.node.HasValue() && !w.node.LabelComesFromValue()
}

func (w *nodeWrapper) isRangeValuePatternSupported() bool {
	_, ok := w.node.NumericValue()
	return ok
}

func (w *nodeWrapper) value() wideValue {
	if w.node.SupportsURL() {
		if url, ok := w.node.URL(); ok {
			return wideValue(url)
		}
	}
	var result strings.Builder
	unwrapWrite(w.node.WriteValue(&result))
	return wideValue(result.String())
}

func (w *nodeWrapper) isReadOnly() boolValue {
	return boolValue(w.node.IsReadOnly())
}

func (w *nodeWrapper) numericValue() f64Value {
	return f64Value(unwrapOption(w.node.NumericValue()))
}

func (w *nodeWrapper) minNumericValue() f64Value {
	if v, ok := w.node.MinNumericValue(); ok {
		return f64Value(v)
	}
	return 0.0
}

func (w *nodeWrapper) maxNumericValue() f64Value {
	if v, ok := w.node.MaxNumericValue(); ok {
		return f64Value(v)
	}
	return 0.0
}

func (w *nodeWrapper) numericValueStep() f64Value {
	if v, ok := w.node.NumericValueStep(); ok {
		return f64Value(v)
	}
	return 0.0
}

func (w *nodeWrapper) numericValueJump() f64Value {
	if v, ok := w.node.NumericValueJump(); ok {
		return f64Value(v)
	}
	return w.numericValueStep()
}

func (w *nodeWrapper) isRequired() boolValue {
	return boolValue(w.node.IsRequired())
}

func (w *nodeWrapper) isScrollItemPatternSupported() bool {
	return w.node.SupportsAction(a11y.ActionScrollIntoView, filter)
}

func (w *nodeWrapper) isSelectionItemPatternSupported() bool {
	switch w.node.Role() {
	case a11y.RoleRadioButton, a11y.RoleMenuItemRadio:
		toggled, ok := w.node.Toggled()
		return ok && (toggled == a11y.ToggledTrue || toggled == a11y.ToggledFalse)
	case a11y.RoleListBoxOption,
		a11y.RoleListItem,
		a11y.RoleMenuListOption,
		a11y.RoleTab,
		a11y.RoleTreeItem:
		_, ok := w.node.IsSelected()
		return ok
	case a11y.RoleGridCell:
		return true
	default:
		return false
	}
}

func (w *nodeWrapper) isSelected() bool {
	switch w.node.Role() {
	case a11y.RoleRadioButton, a11y.RoleMenuItemRadio:
		toggled, ok := w.node.Toggled()
		return ok && toggled == a11y.ToggledTrue
	default:
		selected, ok := w.node.IsSelected()
		return ok && selected
	}
}

func (w *nodeWrapper) positionInSet() optI32 {
	p, ok := w.node.PositionInSet()
	if !ok || p < 0 || p > math.MaxInt32 {
		return optI32{}
	}
	return optI32{value: int32(p) + 1, ok: true}
}

func (w *nodeWrapper) sizeOfSet() optI32 {
	s, ok := w.node.SizeOfSetFromContainer(filter)
	if !ok || s < 0 || s > math.MaxInt32 {
		return optI32{}
	}
	return optI32{value: int32(s), ok: true}
}

func (w *nodeWrapper) level() optI32 {
	level, ok := w.node.Level()
	if !ok || level < 0 || level >= math.MaxInt32 {
		return optI32{}
	}
	return optI32{value: int32(level + 1), ok: true}
}

func (w *nodeWrapper) isSelectionPatternSupported() bool {
	return w.node.IsContainerWithSelectableChildren()
}

func (w *nodeWrapper) isMultiselectable() boolValue {
	return boolValue(w.node.IsMultiselectable())
}

func (w *nodeWrapper) isTextPatternSupported() bool {
	return w.node.SupportsTextRanges()
}

func (w *nodeWrapper) isExpandCollapsePatternSupported() bool {
	return w.node.SupportsExpandCollapse()
}

func (w *nodeWrapper) expandCollapseState() i32Value {
	expanded, ok := w.node.Data().IsExpanded()
	switch {
	case ok && expanded:
		return expandCollapseStateExpanded
	case ok:
		return expandCollapseStateCollapsed
	default:
		return expandCollapseStateLeafNode
	}
}

func (w *nodeWrapper) isPassword() boolValue {
	return w.node.Role() == a11y.RolePasswordInput
}

func (w *nodeWrapper) isDialog() boolValue {
	return boolValue(w.node.IsDialog())
}

func (w *nodeWrapper) isWindowPatternSupported() bool {
	return w.node.IsDialog()
}

func (w *nodeWrapper) isModal() boolValue {
	return boolValue(w.node.IsModal())
}

func (w *nodeWrapper) enqueuePropertyChanges(queue *[]queuedEvent, context *Context, element *com.IUnknown, old *nodeWrapper) {
	w.enqueueSimplePropertyChanges(queue, context, element, old)
	w.enqueuePatternPropertyChanges(queue, element, old)
	w.enqueuePropertyImpliedEvents(queue, element, old)
}

func (w *nodeWrapper) enqueuePropertyImpliedEvents(queue *[]queuedEvent, element *com.IUnknown, old *nodeWrapper) {
	if w.isTextPatternSupported() && old.isTextPatternSupported() {
		newSelection, newOK := w.node.RawTextSelection()
		oldSelection, oldOK := old.node.RawTextSelection()
		if newOK != oldOK || newSelection != oldSelection {
			element.AddRef()
			*queue = append(*queue, &queuedEventSimple{element: element, eventID: uiaTextTextSelectionChangedEventID})
		}
	}
}

func enqueuePropertyChange(queue *[]queuedEvent, element *com.IUnknown, propertyID uiaPropertyID, oldValue, newValue Variant) {
	element.AddRef()
	*queue = append(*queue, &queuedEventPropertyChanged{
		element:    element,
		propertyID: propertyID,
		oldValue:   oldValue.into(),
		newValue:   newValue.into(),
	})
}

type nodeProperty struct {
	id  uiaPropertyID
	get func(*nodeWrapper) propValue
}

var nodeProperties = [...]nodeProperty{
	{uiaControlTypePropertyID, func(w *nodeWrapper) propValue { return w.controlType() }},
	{uiaLocalizedControlTypePropertyID, func(w *nodeWrapper) propValue { return w.localizedControlType() }},
	{uiaAriaRolePropertyID, func(w *nodeWrapper) propValue { return w.ariaRole() }},
	{uiaNamePropertyID, func(w *nodeWrapper) propValue { return w.name() }},
	{uiaFullDescriptionPropertyID, func(w *nodeWrapper) propValue { return w.description() }},
	{uiaCulturePropertyID, func(w *nodeWrapper) propValue { return w.culture() }},
	{uiaHelpTextPropertyID, func(w *nodeWrapper) propValue { return w.placeholder() }},
	{uiaIsContentElementPropertyID, func(w *nodeWrapper) propValue { return w.isContentElement() }},
	{uiaIsControlElementPropertyID, func(w *nodeWrapper) propValue { return w.isContentElement() }},
	{uiaIsEnabledPropertyID, func(w *nodeWrapper) propValue { return w.isEnabled() }},
	{uiaIsKeyboardFocusablePropertyID, func(w *nodeWrapper) propValue { return w.isFocusable() }},
	{uiaHasKeyboardFocusPropertyID, func(w *nodeWrapper) propValue { return w.isFocused() }},
	{uiaLiveSettingPropertyID, func(w *nodeWrapper) propValue { return w.liveSetting() }},
	{uiaAutomationIDPropertyID, func(w *nodeWrapper) propValue { return w.automationID() }},
	{uiaClassNamePropertyID, func(w *nodeWrapper) propValue { return w.className() }},
	{uiaOrientationPropertyID, func(w *nodeWrapper) propValue { return w.orientation() }},
	{uiaIsRequiredForFormPropertyID, func(w *nodeWrapper) propValue { return w.isRequired() }},
	{uiaIsPasswordPropertyID, func(w *nodeWrapper) propValue { return w.isPassword() }},
	{uiaLevelPropertyID, func(w *nodeWrapper) propValue { return w.level() }},
	{uiaPositionInSetPropertyID, func(w *nodeWrapper) propValue { return w.positionInSet() }},
	{uiaSizeOfSetPropertyID, func(w *nodeWrapper) propValue { return w.sizeOfSet() }},
	{uiaAriaPropertiesPropertyID, func(w *nodeWrapper) propValue { return w.ariaProperties() }},
	{uiaIsDialogPropertyID, func(w *nodeWrapper) propValue { return w.isDialog() }},
}

func (w *nodeWrapper) getPropertyValue(propertyID uiaPropertyID) Variant {
	for _, p := range nodeProperties {
		if p.id == propertyID {
			return p.get(w).toVariant(w.stringBuffer)
		}
	}
	return variantEmpty()
}

func (w *nodeWrapper) enqueueSimplePropertyChanges(queue *[]queuedEvent, context *Context, element *com.IUnknown, old *nodeWrapper) {
	for _, p := range nodeProperties {
		oldValue := p.get(old)
		newValue := p.get(w)
		if oldValue != newValue {
			enqueuePropertyChange(queue, element, p.id, oldValue.toVariant(old.stringBuffer), newValue.toVariant(w.stringBuffer))
		}
	}

	oldNext, oldStop := pullIncluded(old.node)
	defer oldStop()
	newNext, newStop := pullIncluded(w.node)
	defer newStop()
	areEqual := true
	var controls []*com.IUnknown
	for {
		a, aok := oldNext()
		b, bok := newNext()
		if aok && bok {
			areEqual = areEqual && a.ID() == b.ID()
			controls = append(controls, context.getOrCreatePlatformNode(b.ID()).Interface(platformNodeSimple))
		} else if !aok && !bok {
			break
		} else {
			areEqual = false
		}
	}
	if !areEqual {
		enqueuePropertyChange(queue, element, uiaControllerForPropertyID, variantEmpty(), variantFromUnknowns(controls))
	} else {
		for _, c := range controls {
			c.Release()
		}
	}
}

func pullIncluded(node *consumer.Node) (func() (consumer.Node, bool), func()) {
	return iter.Pull(iter.Seq[consumer.Node](func(yield func(consumer.Node) bool) {
		for controlled := range node.Controls() {
			if filter(&controlled) == consumer.FilterResultInclude {
				if !yield(controlled) {
					return
				}
			}
		}
	}))
}

type patternProperty struct {
	id  uiaPropertyID
	get func(*nodeWrapper) propValue
}

type nodePattern struct {
	id          uiaPatternID
	iface       int
	isSupported func(*nodeWrapper) bool
	properties  []patternProperty
}

var nodePatterns = [...]nodePattern{
	{uiaTogglePatternID, platformNodeToggle, (*nodeWrapper).isTogglePatternSupported, []patternProperty{
		{uiaToggleToggleStatePropertyID, func(w *nodeWrapper) propValue { return w.toggleState() }},
	}},
	{uiaInvokePatternID, platformNodeInvoke, (*nodeWrapper).isInvokePatternSupported, nil},
	{uiaValuePatternID, platformNodeValue, (*nodeWrapper).isValuePatternSupported, []patternProperty{
		{uiaValueValuePropertyID, func(w *nodeWrapper) propValue { return w.value() }},
		{uiaValueIsReadOnlyPropertyID, func(w *nodeWrapper) propValue { return w.isReadOnly() }},
	}},
	{uiaRangeValuePatternID, platformNodeRangeValue, (*nodeWrapper).isRangeValuePatternSupported, []patternProperty{
		{uiaRangeValueValuePropertyID, func(w *nodeWrapper) propValue { return w.numericValue() }},
		{uiaRangeValueIsReadOnlyPropertyID, func(w *nodeWrapper) propValue { return w.isReadOnly() }},
		{uiaRangeValueMinimumPropertyID, func(w *nodeWrapper) propValue { return w.minNumericValue() }},
		{uiaRangeValueMaximumPropertyID, func(w *nodeWrapper) propValue { return w.maxNumericValue() }},
		{uiaRangeValueSmallChangePropertyID, func(w *nodeWrapper) propValue { return w.numericValueStep() }},
		{uiaRangeValueLargeChangePropertyID, func(w *nodeWrapper) propValue { return w.numericValueJump() }},
	}},
	{uiaScrollItemPatternID, platformNodeScrollItem, (*nodeWrapper).isScrollItemPatternSupported, nil},
	{uiaSelectionItemPatternID, platformNodeSelectionItem, (*nodeWrapper).isSelectionItemPatternSupported, nil},
	{uiaSelectionPatternID, platformNodeSelection, (*nodeWrapper).isSelectionPatternSupported, []patternProperty{
		{uiaSelectionCanSelectMultiplePropertyID, func(w *nodeWrapper) propValue { return w.isMultiselectable() }},
		{uiaSelectionIsSelectionRequiredPropertyID, func(w *nodeWrapper) propValue { return w.isRequired() }},
	}},
	{uiaTextPatternID, platformNodeText, (*nodeWrapper).isTextPatternSupported, nil},
	{uiaExpandCollapsePatternID, platformNodeExpandCollapse, (*nodeWrapper).isExpandCollapsePatternSupported, []patternProperty{
		{uiaExpandCollapseExpandCollapseStatePropertyID, func(w *nodeWrapper) propValue { return w.expandCollapseState() }},
	}},
	{uiaWindowPatternID, platformNodeWindow, (*nodeWrapper).isWindowPatternSupported, []patternProperty{
		{uiaWindowIsModalPropertyID, func(w *nodeWrapper) propValue { return w.isModal() }},
	}},
}

func (w *nodeWrapper) enqueuePatternPropertyChanges(queue *[]queuedEvent, element *com.IUnknown, old *nodeWrapper) {
	for _, pattern := range nodePatterns {
		if pattern.isSupported(w) && pattern.isSupported(old) {
			for _, p := range pattern.properties {
				oldValue := p.get(old)
				newValue := p.get(w)
				if oldValue != newValue {
					enqueuePropertyChange(queue, element, p.id, oldValue.toVariant(old.stringBuffer), newValue.toVariant(w.stringBuffer))
				}
			}
		}
	}
}

type platformNode struct {
	context   weak.Pointer[Context]
	nodeID    consumer.NodeID
	hasNodeID bool
	object    *com.Object
}

func newPlatformNode(context *Context, nodeID consumer.NodeID) *com.Object {
	return newPlatformNodeObject(&platformNode{context: weak.Make(context), nodeID: nodeID, hasNodeID: true})
}

func unspecifiedRootPlatformNode(context *Context) *com.Object {
	return newPlatformNodeObject(&platformNode{context: weak.Make(context)})
}

func newPlatformNodeObject(impl *platformNode) *com.Object {
	o := com.NewObject(impl, platformNodeVtables()...)
	impl.object = o
	return o
}

func (p *platformNode) toInterface(index int) *com.IUnknown {
	p.object.AddRef()
	return p.object.Interface(index)
}

func (p *platformNode) upgradeContext() (*Context, error) {
	return upgrade(p.context)
}

func withTreeStateAndContext[T any](p *platformNode, f func(*consumer.TreeState, *Context) (T, error)) (T, error) {
	return withTreeAndContext(p, func(tree *consumer.Tree, context *Context) (T, error) {
		return f(tree.State(), context)
	})
}

func withTreeAndContext[T any](p *platformNode, f func(*consumer.Tree, *Context) (T, error)) (T, error) {
	context, err := p.upgradeContext()
	if err != nil {
		var zero T
		return zero, err
	}
	tree, unlock := context.readTree()
	defer unlock()
	return f(tree, context)
}

func (p *platformNode) node(tree *consumer.Tree) (consumer.Node, error) {
	state := tree.State()
	if p.hasNodeID {
		if node, ok := state.NodeByID(p.nodeID); ok {
			return node, nil
		}
		return consumer.Node{}, elementNotAvailable()
	}
	return state.Root(), nil
}

func (p *platformNode) nodeWithLocation(tree *consumer.Tree) (consumer.Node, a11y.NodeID, a11y.TreeID, error) {
	node, err := p.node(tree)
	if err != nil {
		return consumer.Node{}, 0, a11y.TreeID{}, err
	}
	localID, treeID, ok := tree.State().LocateNode(node.ID())
	if !ok {
		return consumer.Node{}, 0, a11y.TreeID{}, elementNotAvailable()
	}
	return node, localID, treeID, nil
}

func resolveWithContext[T any](p *platformNode, f func(consumer.Node, *Context) (T, error)) (T, error) {
	return withTreeAndContext(p, func(tree *consumer.Tree, context *Context) (T, error) {
		node, err := p.node(tree)
		if err != nil {
			var zero T
			return zero, err
		}
		return f(node, context)
	})
}

func resolveWithTreeStateAndContext[T any](p *platformNode, f func(consumer.Node, *consumer.TreeState, *Context) (T, error)) (T, error) {
	return withTreeAndContext(p, func(tree *consumer.Tree, context *Context) (T, error) {
		node, err := p.node(tree)
		if err != nil {
			var zero T
			return zero, err
		}
		return f(node, tree.State(), context)
	})
}

func resolve[T any](p *platformNode, f func(consumer.Node) (T, error)) (T, error) {
	return resolveWithContext(p, func(node consumer.Node, _ *Context) (T, error) {
		return f(node)
	})
}

func resolveWithContextForTextPattern[T any](p *platformNode, f func(consumer.Node, *Context) (T, error)) (T, error) {
	return withTreeAndContext(p, func(tree *consumer.Tree, context *Context) (T, error) {
		node, err := p.node(tree)
		if err != nil {
			var zero T
			return zero, err
		}
		if node.SupportsTextRanges() {
			return f(node, context)
		}
		var zero T
		return zero, elementNotAvailable()
	})
}

func resolveForTextPattern[T any](p *platformNode, f func(consumer.Node) (T, error)) (T, error) {
	return resolveWithContextForTextPattern(p, func(node consumer.Node, _ *Context) (T, error) {
		return f(node)
	})
}

func (p *platformNode) doComplexAction(f func(consumer.Node, a11y.NodeID, a11y.TreeID) (*a11y.ActionRequest, error)) error {
	context, err := p.upgradeContext()
	if err != nil {
		return err
	}
	if context.isPlaceholder.Load() {
		return elementNotEnabled()
	}
	tree, unlock := context.readTree()
	node, targetNode, targetTree, err := p.nodeWithLocation(tree)
	if err != nil {
		unlock()
		return err
	}
	request, err := f(node, targetNode, targetTree)
	unlock()
	if err != nil {
		return err
	}
	if request != nil {
		context.doAction(*request)
	}
	return nil
}

func (p *platformNode) doAction(f func() (a11y.Action, a11y.ActionData)) error {
	return p.doComplexAction(func(node consumer.Node, targetNode a11y.NodeID, targetTree a11y.TreeID) (*a11y.ActionRequest, error) {
		if node.IsDisabled() {
			return nil, elementNotEnabled()
		}
		action, data := f()
		return &a11y.ActionRequest{Action: action, TargetTree: targetTree, TargetNode: targetNode, Data: data}, nil
	})
}

func (p *platformNode) click() error {
	return p.doAction(func() (a11y.Action, a11y.ActionData) { return a11y.ActionClick, nil })
}

func (p *platformNode) setExpanded(expanded bool) error {
	return p.doComplexAction(func(node consumer.Node, targetNode a11y.NodeID, targetTree a11y.TreeID) (*a11y.ActionRequest, error) {
		if node.IsDisabled() {
			return nil, elementNotEnabled()
		}
		current, ok := node.Data().IsExpanded()
		if !ok {
			return nil, invalidOperation()
		}
		if current == expanded {
			return nil, invalidOperation()
		}
		action := a11y.ActionCollapse
		if expanded {
			action = a11y.ActionExpand
		}
		return &a11y.ActionRequest{Action: action, TargetTree: targetTree, TargetNode: targetNode}, nil
	})
}

func (p *platformNode) setSelected(selected bool) error {
	return p.doComplexAction(func(node consumer.Node, targetNode a11y.NodeID, targetTree a11y.TreeID) (*a11y.ActionRequest, error) {
		if node.IsDisabled() {
			return nil, elementNotEnabled()
		}
		buffer := acquireStringBuffer()
		defer buffer.release()
		wrapper := nodeWrapper{node: &node, stringBuffer: &buffer.buf}
		if selected == wrapper.isSelected() {
			return nil, nil
		}
		return &a11y.ActionRequest{Action: a11y.ActionClick, TargetTree: targetTree, TargetNode: targetNode}, nil
	})
}

func (p *platformNode) isRoot(state *consumer.TreeState) bool {
	return p.hasNodeID && p.nodeID == state.RootID()
}

func (p *platformNode) providerOptions() (int32, error) {
	return providerOptionsServerSideProvider, nil
}

func (p *platformNode) getPatternProvider(patternID uiaPatternID) (*com.IUnknown, error) {
	return p.patternProvider(patternID)
}

func (p *platformNode) getPropertyValue(propertyID uiaPropertyID) (com.VARIANT, error) {
	return resolveWithTreeStateAndContext(p, func(node consumer.Node, state *consumer.TreeState, context *Context) (com.VARIANT, error) {
		buffer := acquireStringBuffer()
		defer buffer.release()
		wrapper := nodeWrapper{node: &node, stringBuffer: &buffer.buf}
		result := wrapper.getPropertyValue(propertyID)
		if result.isEmpty() {
			if node.IsRoot() {
				switch propertyID {
				case uiaNamePropertyID:
					title, ok := windowTitle(context.hwnd, &buffer.buf)
					result = variantFromOption(title, ok, variantFromBSTR)
				case uiaNativeWindowHandlePropertyID:
					result = variantFromI32(int32(context.hwnd.HWND))
				}
			}
			switch propertyID {
			case uiaFrameworkIDPropertyID:
				result = variantFromString(state.ToolkitName(), &buffer.buf)
			case uiaProviderDescriptionPropertyID:
				result = toolkitDescription(state, &buffer.buf).toVariant()
			case uiaControllerForPropertyID:
				var controlled []*com.IUnknown
				for c := range node.Controls() {
					if filter(&c) == consumer.FilterResultInclude {
						controlled = append(controlled, context.getOrCreatePlatformNode(c.ID()).Interface(platformNodeSimple))
					}
				}
				result = variantFromUnknowns(controlled)
			}
		}
		return result.into(), nil
	})
}

func (p *platformNode) hostRawElementProvider() (*com.IUnknown, error) {
	return withTreeStateAndContext(p, func(state *consumer.TreeState, context *Context) (*com.IUnknown, error) {
		if p.isRoot(state) {
			return uiaHostProviderFromHwnd(context.hwnd.HWND)
		}
		return nil, nil
	})
}

func (p *platformNode) navigate(direction int32) (*com.IUnknown, error) {
	return resolveWithContext(p, func(node consumer.Node, context *Context) (*com.IUnknown, error) {
		var result consumer.Node
		var ok bool
		switch direction {
		case navigateDirectionParent:
			result, ok = node.FilteredParent(filterWithRootException)
		case navigateDirectionNextSibling:
			result, ok = node.FollowingFilteredSiblings(filter).Next()
		case navigateDirectionPreviousSibling:
			result, ok = node.PrecedingFilteredSiblings(filter).Next()
		case navigateDirectionFirstChild:
			result, ok = node.FilteredChildren(filter).Next()
		case navigateDirectionLastChild:
			result, ok = node.FilteredChildren(filter).NextBack()
		}
		if !ok {
			return nil, nil
		}
		return context.getOrCreatePlatformNode(result.ID()).Interface(platformNodeFragment), nil
	})
}

func (p *platformNode) getRuntimeID() (*com.SAFEARRAY, error) {
	if !p.hasNodeID {
		return nil, notImplemented()
	}
	runtimeID := runtimeIDFromNodeID(p.nodeID)
	return safeArrayFromI32Slice(runtimeID[:]), nil
}

func (p *platformNode) boundingRectangle() (uiaRect, error) {
	return resolveWithContext(p, func(node consumer.Node, context *Context) (uiaRect, error) {
		rect, ok := node.BoundingBox()
		if !ok {
			return uiaRect{}, nil
		}
		clientTopLeft := context.clientTopLeft()
		return uiaRect{
			Left:   rect.X0 + clientTopLeft.X,
			Top:    rect.Y0 + clientTopLeft.Y,
			Width:  rect.Width(),
			Height: rect.Height(),
		}, nil
	})
}

func (p *platformNode) getEmbeddedFragmentRoots() (*com.SAFEARRAY, error) {
	return nil, nil
}

func (p *platformNode) setFocus() error {
	return p.doAction(func() (a11y.Action, a11y.ActionData) { return a11y.ActionFocus, nil })
}

func (p *platformNode) fragmentRoot() (*com.IUnknown, error) {
	return withTreeStateAndContext(p, func(state *consumer.TreeState, context *Context) (*com.IUnknown, error) {
		if p.isRoot(state) {
			return p.toInterface(platformNodeFragmentRoot), nil
		}
		rootID := state.RootID()
		return context.getOrCreatePlatformNode(rootID).Interface(platformNodeFragmentRoot), nil
	})
}

func (p *platformNode) elementProviderFromPoint(x, y float64) (*com.IUnknown, error) {
	return resolveWithContext(p, func(node consumer.Node, context *Context) (*com.IUnknown, error) {
		clientTopLeft := context.clientTopLeft()
		point := a11y.NewPoint(x-clientTopLeft.X, y-clientTopLeft.Y)
		point = node.Transform().Inverse().MulPoint(point)
		found, ok := node.NodeAtPoint(point, filter)
		if !ok {
			return nil, nil
		}
		return context.getOrCreatePlatformNode(found.ID()).Interface(platformNodeFragment), nil
	})
}

func (p *platformNode) getFocus() (*com.IUnknown, error) {
	return withTreeStateAndContext(p, func(state *consumer.TreeState, context *Context) (*com.IUnknown, error) {
		if node, ok := state.Focus(); ok {
			id := node.ID()
			selfID := state.RootID()
			if p.hasNodeID {
				selfID = p.nodeID
			}
			if id != selfID {
				return context.getOrCreatePlatformNode(id).Interface(platformNodeFragment), nil
			}
		}
		return nil, nil
	})
}

func (p *platformNode) patternProvider(patternID uiaPatternID) (*com.IUnknown, error) {
	return resolve(p, func(node consumer.Node) (*com.IUnknown, error) {
		buffer := acquireStringBuffer()
		defer buffer.release()
		wrapper := nodeWrapper{node: &node, stringBuffer: &buffer.buf}
		for _, pattern := range nodePatterns {
			if pattern.id == patternID && pattern.isSupported(&wrapper) {
				return p.toInterface(platformNodeSimple), nil
			}
		}
		return nil, nil
	})
}

func patternGetter[T any](p *platformNode, getter func(*nodeWrapper) T) (T, error) {
	return resolve(p, func(node consumer.Node) (T, error) {
		buffer := acquireStringBuffer()
		defer buffer.release()
		wrapper := nodeWrapper{node: &node, stringBuffer: &buffer.buf}
		return getter(&wrapper), nil
	})
}

func (p *platformNode) toggle() error {
	return p.click()
}

func (p *platformNode) invoke() error {
	return p.click()
}

func (p *platformNode) valueSetValue(value string) error {
	return p.doAction(func() (a11y.Action, a11y.ActionData) {
		return a11y.ActionSetValue, a11y.ActionDataValue(value)
	})
}

func (p *platformNode) rangeValueSetValue(value float64) error {
	return p.doAction(func() (a11y.Action, a11y.ActionData) {
		return a11y.ActionSetValue, a11y.ActionDataNumericValue(value)
	})
}

func (p *platformNode) scrollIntoView() error {
	return p.doComplexAction(func(_ consumer.Node, targetNode a11y.NodeID, targetTree a11y.TreeID) (*a11y.ActionRequest, error) {
		return &a11y.ActionRequest{Action: a11y.ActionScrollIntoView, TargetTree: targetTree, TargetNode: targetNode}, nil
	})
}

func (p *platformNode) selectionItemIsSelected() (bool, error) {
	return patternGetter(p, (*nodeWrapper).isSelected)
}

func (p *platformNode) selectionItemSelect() error {
	return p.setSelected(true)
}

func (p *platformNode) selectionItemAddToSelection() error {
	return p.setSelected(true)
}

func (p *platformNode) selectionItemRemoveFromSelection() error {
	return p.setSelected(false)
}

func (p *platformNode) selectionContainer() (*com.IUnknown, error) {
	return resolveWithContext(p, func(node consumer.Node, context *Context) (*com.IUnknown, error) {
		if container, ok := node.SelectionContainer(filter); ok {
			return context.getOrCreatePlatformNode(container.ID()).Interface(platformNodeSimple), nil
		}
		return nil, com.E_FAIL
	})
}

func (p *platformNode) selectionGetSelection() (*com.SAFEARRAY, error) {
	return resolveWithContext(p, func(node consumer.Node, context *Context) (*com.SAFEARRAY, error) {
		var selection []*com.IUnknown
		for item := range node.Items(filter).All() {
			if selected, ok := item.IsSelected(); ok && selected {
				selection = append(selection, context.getOrCreatePlatformNode(item.ID()).Interface(platformNodeSimple))
			}
		}
		defer func() {
			for _, item := range selection {
				item.Release()
			}
		}()
		return safeArrayFromComSlice(selection), nil
	})
}

func (p *platformNode) textGetSelection() (*com.SAFEARRAY, error) {
	return resolveForTextPattern(p, func(node consumer.Node) (*com.SAFEARRAY, error) {
		r, ok := node.TextSelection()
		if !ok {
			return nil, nil
		}
		platformRange := newPlatformRange(p.context, r)
		defer platformRange.Release()
		return safeArrayFromComSlice([]*com.IUnknown{platformRange.Interface(0)}), nil
	})
}

func (p *platformNode) textGetVisibleRanges() (*com.SAFEARRAY, error) {
	return nil, nil
}

func (p *platformNode) textRangeFromChild(*com.IUnknown) (*com.IUnknown, error) {
	return nil, notImplemented()
}

func (p *platformNode) textRangeFromPoint(point uiaPoint) (*com.IUnknown, error) {
	return resolveWithContextForTextPattern(p, func(node consumer.Node, context *Context) (*com.IUnknown, error) {
		clientTopLeft := context.clientTopLeft()
		pt := a11y.NewPoint(point.X-clientTopLeft.X, point.Y-clientTopLeft.Y)
		pt = node.Transform().Inverse().MulPoint(pt)
		pos := node.TextPositionAtPoint(pt)
		r := pos.ToDegenerateRange()
		return newPlatformRange(p.context, r).Interface(0), nil
	})
}

func (p *platformNode) textDocumentRange() (*com.IUnknown, error) {
	return resolveForTextPattern(p, func(node consumer.Node) (*com.IUnknown, error) {
		r := node.DocumentRange()
		return newPlatformRange(p.context, r).Interface(0), nil
	})
}

func (p *platformNode) textSupportedTextSelection() (int32, error) {
	return resolveForTextPattern(p, func(node consumer.Node) (int32, error) {
		if node.HasTextSelection() {
			return supportedTextSelectionSingle, nil
		}
		return supportedTextSelectionNone, nil
	})
}

func (p *platformNode) expand() error {
	return p.setExpanded(true)
}

func (p *platformNode) collapse() error {
	return p.setExpanded(false)
}

func (p *platformNode) windowSetVisualState(int32) error {
	return invalidOperation()
}

func (p *platformNode) windowClose() error {
	return notSupported()
}

func (p *platformNode) windowWaitForInputIdle(int32) (bool, error) {
	return false, notSupported()
}

func (p *platformNode) windowCanMaximize() (bool, error) {
	return false, notSupported()
}

func (p *platformNode) windowCanMinimize() (bool, error) {
	return false, notSupported()
}

func (p *platformNode) windowVisualState() (int32, error) {
	return 0, notSupported()
}

func (p *platformNode) windowInteractionState() (int32, error) {
	return windowInteractionStateReadyForUserInteraction, nil
}

func (p *platformNode) windowIsTopmost() (bool, error) {
	return false, notSupported()
}

const (
	platformNodeSimple = iota
	platformNodeFragment
	platformNodeFragmentRoot
	platformNodeToggle
	platformNodeInvoke
	platformNodeValue
	platformNodeRangeValue
	platformNodeScrollItem
	platformNodeSelectionItem
	platformNodeSelection
	platformNodeText
	platformNodeExpandCollapse
	platformNodeWindow
)

func hresult(err error) uintptr {
	return com.HRESULTFromError(err).Uintptr()
}

func boolOut(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func platformNodeOf(this *com.IUnknown) *platformNode {
	p, ok := com.ImplOf[*platformNode](this)
	if !ok {
		return nil
	}
	return p
}

func outInterface(out **com.IUnknown, f func(*platformNode) (*com.IUnknown, error), this *com.IUnknown) uintptr {
	if out == nil {
		return com.E_POINTER.Uintptr()
	}
	*out = nil
	p := platformNodeOf(this)
	if p == nil {
		return com.E_UNEXPECTED.Uintptr()
	}
	result, err := f(p)
	if err != nil {
		if result != nil {
			result.Release()
		}
		return hresult(err)
	}
	*out = result
	return com.S_OK.Uintptr()
}

func outValue[T any](out *T, f func(*platformNode) (T, error), this *com.IUnknown) uintptr {
	if out == nil {
		return com.E_POINTER.Uintptr()
	}
	var zero T
	*out = zero
	p := platformNodeOf(this)
	if p == nil {
		return com.E_UNEXPECTED.Uintptr()
	}
	result, err := f(p)
	if err != nil {
		return hresult(err)
	}
	*out = result
	return com.S_OK.Uintptr()
}

func outBool(out *int32, f func(*platformNode) (bool, error), this *com.IUnknown) uintptr {
	return outValue(out, func(p *platformNode) (int32, error) {
		v, err := f(p)
		return boolOut(v), err
	}, this)
}

func noOut(f func(*platformNode) error, this *com.IUnknown) uintptr {
	p := platformNodeOf(this)
	if p == nil {
		return com.E_UNEXPECTED.Uintptr()
	}
	return hresult(f(p))
}

func newPlatformNodeVtable(iid com.GUID, methods ...any) *com.Vtable {
	slots := make([]uintptr, len(methods))
	for i, m := range methods {
		switch m := m.(type) {
		case func() uintptr:
			slots[i] = m()
		default:
			slots[i] = windows.NewCallback(m)
		}
	}
	return com.NewVtable([]com.GUID{iid}, slots...)
}

var (
	platformNodeVtablesOnce  sync.Once
	platformNodeVtablesCache []*com.Vtable
)

func platformNodeVtables() []*com.Vtable {
	platformNodeVtablesOnce.Do(buildPlatformNodeVtables)
	return platformNodeVtablesCache
}

func buildPlatformNodeVtables() {
	platformNodeVtablesCache = []*com.Vtable{
		newPlatformNodeVtable(iidIRawElementProviderSimple,
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, (*platformNode).providerOptions, this)
			},
			func(this *com.IUnknown, patternID int32, out **com.IUnknown) uintptr {
				return outInterface(out, func(p *platformNode) (*com.IUnknown, error) { return p.getPatternProvider(patternID) }, this)
			},
			func(this *com.IUnknown, propertyID int32, out *com.VARIANT) uintptr {
				return outValue(out, func(p *platformNode) (com.VARIANT, error) { return p.getPropertyValue(propertyID) }, this)
			},
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, (*platformNode).hostRawElementProvider, this)
			},
		),
		newPlatformNodeVtable(iidIRawElementProviderFragment,
			func(this *com.IUnknown, direction int32, out **com.IUnknown) uintptr {
				return outInterface(out, func(p *platformNode) (*com.IUnknown, error) { return p.navigate(direction) }, this)
			},
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return outValue(out, (*platformNode).getRuntimeID, this)
			},
			func(this *com.IUnknown, out *uiaRect) uintptr {
				return outValue(out, (*platformNode).boundingRectangle, this)
			},
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return outValue(out, (*platformNode).getEmbeddedFragmentRoots, this)
			},
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).setFocus, this)
			},
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, (*platformNode).fragmentRoot, this)
			},
		),
		newPlatformNodeVtable(iidIRawElementProviderFragmentRoot,
			elementProviderFromPointSlot,
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, (*platformNode).getFocus, this)
			},
		),
		newPlatformNodeVtable(iidIToggleProvider,
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).toggle, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, func(p *platformNode) (int32, error) {
					v, err := patternGetter(p, (*nodeWrapper).toggleState)
					return int32(v), err
				}, this)
			},
		),
		newPlatformNodeVtable(iidIInvokeProvider,
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).invoke, this)
			},
		),
		newPlatformNodeVtable(iidIValueProvider,
			func(this *com.IUnknown, value *uint16) uintptr {
				return noOut(func(p *platformNode) error { return p.valueSetValue(windows.UTF16PtrToString(value)) }, this)
			},
			func(this *com.IUnknown, out *com.BSTR) uintptr {
				return outValue(out, func(p *platformNode) (com.BSTR, error) {
					return resolve(p, func(node consumer.Node) (com.BSTR, error) {
						buffer := acquireStringBuffer()
						defer buffer.release()
						wrapper := nodeWrapper{node: &node, stringBuffer: &buffer.buf}
						result := newWideString(&buffer.buf)
						result.WriteString(string(wrapper.value()))
						return result.toBSTR(), nil
					})
				}, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, boolGetter((*nodeWrapper).isReadOnly), this)
			},
		),
		newPlatformNodeVtable(iidIRangeValueProvider,
			rangeValueSetValueSlot,
			func(this *com.IUnknown, out *float64) uintptr {
				return outValue(out, f64Getter((*nodeWrapper).numericValue), this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, boolGetter((*nodeWrapper).isReadOnly), this)
			},
			func(this *com.IUnknown, out *float64) uintptr {
				return outValue(out, f64Getter((*nodeWrapper).maxNumericValue), this)
			},
			func(this *com.IUnknown, out *float64) uintptr {
				return outValue(out, f64Getter((*nodeWrapper).minNumericValue), this)
			},
			func(this *com.IUnknown, out *float64) uintptr {
				return outValue(out, f64Getter((*nodeWrapper).numericValueJump), this)
			},
			func(this *com.IUnknown, out *float64) uintptr {
				return outValue(out, f64Getter((*nodeWrapper).numericValueStep), this)
			},
		),
		newPlatformNodeVtable(iidIScrollItemProvider,
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).scrollIntoView, this)
			},
		),
		newPlatformNodeVtable(iidISelectionItemProvider,
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).selectionItemSelect, this)
			},
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).selectionItemAddToSelection, this)
			},
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).selectionItemRemoveFromSelection, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, (*platformNode).selectionItemIsSelected, this)
			},
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, (*platformNode).selectionContainer, this)
			},
		),
		newPlatformNodeVtable(iidISelectionProvider,
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return outValue(out, (*platformNode).selectionGetSelection, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, boolGetter((*nodeWrapper).isMultiselectable), this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, boolGetter((*nodeWrapper).isRequired), this)
			},
		),
		newPlatformNodeVtable(iidITextProvider,
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return outValue(out, (*platformNode).textGetSelection, this)
			},
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return outValue(out, (*platformNode).textGetVisibleRanges, this)
			},
			func(this *com.IUnknown, child *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, func(p *platformNode) (*com.IUnknown, error) { return p.textRangeFromChild(child) }, this)
			},
			textRangeFromPointSlot,
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return outInterface(out, (*platformNode).textDocumentRange, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, (*platformNode).textSupportedTextSelection, this)
			},
		),
		newPlatformNodeVtable(iidIExpandCollapseProvider,
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).expand, this)
			},
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).collapse, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, func(p *platformNode) (int32, error) {
					v, err := patternGetter(p, (*nodeWrapper).expandCollapseState)
					return int32(v), err
				}, this)
			},
		),
		newPlatformNodeVtable(iidIWindowProvider,
			func(this *com.IUnknown, state int32) uintptr {
				return noOut(func(p *platformNode) error { return p.windowSetVisualState(state) }, this)
			},
			func(this *com.IUnknown) uintptr {
				return noOut((*platformNode).windowClose, this)
			},
			func(this *com.IUnknown, milliseconds int32, out *int32) uintptr {
				return outBool(out, func(p *platformNode) (bool, error) { return p.windowWaitForInputIdle(milliseconds) }, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, (*platformNode).windowCanMaximize, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, (*platformNode).windowCanMinimize, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, boolGetter((*nodeWrapper).isModal), this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, (*platformNode).windowVisualState, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outValue(out, (*platformNode).windowInteractionState, this)
			},
			func(this *com.IUnknown, out *int32) uintptr {
				return outBool(out, (*platformNode).windowIsTopmost, this)
			},
		),
	}
}

func boolGetter(getter func(*nodeWrapper) boolValue) func(*platformNode) (bool, error) {
	return func(p *platformNode) (bool, error) {
		v, err := patternGetter(p, getter)
		return bool(v), err
	}
}

func f64Getter(getter func(*nodeWrapper) f64Value) func(*platformNode) (float64, error) {
	return func(p *platformNode) (float64, error) {
		v, err := patternGetter(p, getter)
		return float64(v), err
	}
}

func elementProviderFromPointCallback(this *com.IUnknown, x, y float64, out **com.IUnknown) uintptr {
	return outInterface(out, func(p *platformNode) (*com.IUnknown, error) { return p.elementProviderFromPoint(x, y) }, this)
}

func rangeValueSetValueCallback(this *com.IUnknown, value float64) uintptr {
	return noOut(func(p *platformNode) error { return p.rangeValueSetValue(value) }, this)
}

func textRangeFromPointCallback(this *com.IUnknown, point uiaPoint, out **com.IUnknown) uintptr {
	return outInterface(out, func(p *platformNode) (*com.IUnknown, error) { return p.textRangeFromPoint(point) }, this)
}
