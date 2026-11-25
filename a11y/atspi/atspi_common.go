package atspi

import "iter"

// Role is an AT-SPI role, with atspi-common's u32 discriminants.
type Role uint32

// Role variants.
const (
	RoleInvalid Role = iota
	RoleAcceleratorLabel
	RoleAlert
	RoleAnimation
	RoleArrow
	RoleCalendar
	RoleCanvas
	RoleCheckBox
	RoleCheckMenuItem
	RoleColorChooser
	RoleColumnHeader
	RoleComboBox
	RoleDateEditor
	RoleDesktopIcon
	RoleDesktopFrame
	RoleDial
	RoleDialog
	RoleDirectoryPane
	RoleDrawingArea
	RoleFileChooser
	RoleFiller
	RoleFocusTraversable
	RoleFontChooser
	RoleFrame
	RoleGlassPane
	RoleHTMLContainer
	RoleIcon
	RoleImage
	RoleInternalFrame
	RoleLabel
	RoleLayeredPane
	RoleList
	RoleListItem
	RoleMenu
	RoleMenuBar
	RoleMenuItem
	RoleOptionPane
	RolePageTab
	RolePageTabList
	RolePanel
	RolePasswordText
	RolePopupMenu
	RoleProgressBar
	RoleButton
	RoleRadioButton
	RoleRadioMenuItem
	RoleRootPane
	RoleRowHeader
	RoleScrollBar
	RoleScrollPane
	RoleSeparator
	RoleSlider
	RoleSpinButton
	RoleSplitPane
	RoleStatusBar
	RoleTable
	RoleTableCell
	RoleTableColumnHeader
	RoleTableRowHeader
	RoleTearoffMenuItem
	RoleTerminal
	RoleText
	RoleToggleButton
	RoleToolBar
	RoleToolTip
	RoleTree
	RoleTreeTable
	RoleUnknown
	RoleViewport
	RoleWindow
	RoleExtended
	RoleHeader
	RoleFooter
	RoleParagraph
	RoleRuler
	RoleApplication
	RoleAutocomplete
	RoleEditbar
	RoleEmbedded
	RoleEntry
	RoleCHART
	RoleCaption
	RoleDocumentFrame
	RoleHeading
	RolePage
	RoleSection
	RoleRedundantObject
	RoleForm
	RoleLink
	RoleInputMethodWindow
	RoleTableRow
	RoleTreeItem
	RoleDocumentSpreadsheet
	RoleDocumentPresentation
	RoleDocumentText
	RoleDocumentWeb
	RoleDocumentEmail
	RoleComment
	RoleListBox
	RoleGrouping
	RoleImageMap
	RoleNotification
	RoleInfoBar
	RoleLevelBar
	RoleTitleBar
	RoleBlockQuote
	RoleAudio
	RoleVideo
	RoleDefinition
	RoleArticle
	RoleLandmark
	RoleLog
	RoleMarquee
	RoleMath
	RoleRating
	RoleTimer
	RoleStatic
	RoleMathFraction
	RoleMathRoot
	RoleSubscript
	RoleSuperscript
	RoleDescriptionList
	RoleDescriptionTerm
	RoleDescriptionValue
	RoleFootnote
	RoleContentDeletion
	RoleContentInsertion
	RoleMark
	RoleSuggestion
	RolePushButtonMenu
)

var roleNames = [...]string{
	"invalid",
	"accelerator label",
	"alert",
	"animation",
	"arrow",
	"calendar",
	"canvas",
	"check box",
	"check menu item",
	"color chooser",
	"column header",
	"combo box",
	"date editor",
	"desktop icon",
	"desktop frame",
	"dial",
	"dialog",
	"directory pane",
	"drawing area",
	"file chooser",
	"filler",
	"focus traversable",
	"font chooser",
	"frame",
	"glass pane",
	"html container",
	"icon",
	"image",
	"internal frame",
	"label",
	"layered pane",
	"list",
	"list item",
	"menu",
	"menu bar",
	"menu item",
	"option pane",
	"page tab",
	"page tab list",
	"panel",
	"password text",
	"popup menu",
	"progress bar",
	"button",
	"radio button",
	"radio menu item",
	"root pane",
	"row header",
	"scroll bar",
	"scroll pane",
	"separator",
	"slider",
	"spin button",
	"split pane",
	"status bar",
	"table",
	"table cell",
	"table column header",
	"table row header",
	"tearoff menu item",
	"terminal",
	"text",
	"toggle button",
	"tool bar",
	"tool tip",
	"tree",
	"tree table",
	"unknown",
	"viewport",
	"window",
	"extended",
	"header",
	"footer",
	"paragraph",
	"ruler",
	"application",
	"autocomplete",
	"editbar",
	"embedded",
	"entry",
	"chart",
	"caption",
	"document frame",
	"heading",
	"page",
	"section",
	"redundant object",
	"form",
	"link",
	"input method window",
	"table row",
	"tree item",
	"document spreadsheet",
	"document presentation",
	"document text",
	"document web",
	"document email",
	"comment",
	"list box",
	"grouping",
	"image map",
	"notification",
	"info bar",
	"level bar",
	"title bar",
	"block quote",
	"audio",
	"video",
	"definition",
	"article",
	"landmark",
	"log",
	"marquee",
	"math",
	"rating",
	"timer",
	"static",
	"math fraction",
	"math root",
	"subscript",
	"superscript",
	"description list",
	"description term",
	"description value",
	"footnote",
	"content deletion",
	"content insertion",
	"mark",
	"suggestion",
	"push button menu",
}

// RoleFromU32 converts an AT-SPI role number, reporting false for unknown values.
func RoleFromU32(value uint32) (Role, bool) {
	if int(value) >= len(roleNames) {
		return 0, false
	}
	return Role(value), true
}

// Name returns the role's AT-SPI name.
func (r Role) Name() string {
	return roleNames[r]
}

// String returns the role's AT-SPI name.
func (r Role) String() string {
	return r.Name()
}

// State is one AT-SPI state flag; each variant is a distinct bit.
type State uint64

// State variants.
const (
	StateInvalid State = 1 << iota
	StateActive
	StateArmed
	StateBusy
	StateChecked
	StateCollapsed
	StateDefunct
	StateEditable
	StateEnabled
	StateExpandable
	StateExpanded
	StateFocusable
	StateFocused
	StateHasTooltip
	StateHorizontal
	StateIconified
	StateModal
	StateMultiLine
	StateMultiselectable
	StateOpaque
	StatePressed
	StateResizable
	StateSelectable
	StateSelected
	StateSensitive
	StateShowing
	StateSingleLine
	StateStale
	StateTransient
	StateVertical
	StateVisible
	StateManagesDescendants
	StateIndeterminate
	StateRequired
	StateTruncated
	StateAnimated
	StateInvalidEntry
	StateSupportsAutocompletion
	StateSelectableText
	StateIsDefault
	StateVisited
	StateCheckable
	StateHasPopup
	StateReadOnly
)

var stateNames = [...]string{
	"invalid",
	"active",
	"armed",
	"busy",
	"checked",
	"collapsed",
	"defunct",
	"editable",
	"enabled",
	"expandable",
	"expanded",
	"focusable",
	"focused",
	"has-tooltip",
	"horizontal",
	"iconified",
	"modal",
	"multi-line",
	"multiselectable",
	"opaque",
	"pressed",
	"resizable",
	"selectable",
	"selected",
	"sensitive",
	"showing",
	"single-line",
	"stale",
	"transient",
	"vertical",
	"visible",
	"manages-descendants",
	"indeterminate",
	"required",
	"truncated",
	"animated",
	"invalid-entry",
	"supports-autocompletion",
	"selectable-text",
	"is-default",
	"visited",
	"checkable",
	"has-popup",
	"read-only",
}

// ToStaticStr returns the state's AT-SPI name.
func (s State) ToStaticStr() string {
	for i := range stateNames {
		if s == State(1)<<i {
			return stateNames[i]
		}
	}
	return stateNames[0]
}

// String returns the state's AT-SPI name.
func (s State) String() string {
	return s.ToStaticStr()
}

// StateFromStr parses an AT-SPI state name; unknown names are StateInvalid.
func StateFromStr(s string) State {
	for i, name := range stateNames {
		if name == s {
			return State(1) << i
		}
	}
	return StateInvalid
}

// StateSet is a set of State flags.
type StateSet struct {
	bits uint64
}

// NewStateSet returns a set holding the given states.
func NewStateSet(states ...State) StateSet {
	var set StateSet
	for _, s := range states {
		set.Insert(s)
	}
	return set
}

// StateSetEmpty returns the empty set.
func StateSetEmpty() StateSet {
	return StateSet{}
}

// StateSetFromBits returns the set with the given bits.
func StateSetFromBits(bits uint64) StateSet {
	return StateSet{bits: bits}
}

// Bits returns the set's bits.
func (s StateSet) Bits() uint64 {
	return s.bits
}

// Contains reports whether every flag in state is in the set.
func (s StateSet) Contains(state State) bool {
	return s.bits&uint64(state) == uint64(state)
}

// Insert adds the flags in state.
func (s *StateSet) Insert(state State) {
	s.bits |= uint64(state)
}

// Remove removes the flags in state.
func (s *StateSet) Remove(state State) {
	s.bits &^= uint64(state)
}

// IsEmpty reports whether the set has no flags.
func (s StateSet) IsEmpty() bool {
	return s.bits == 0
}

// Xor returns the symmetric difference of s and other.
func (s StateSet) Xor(other StateSet) StateSet {
	return StateSet{bits: s.bits ^ other.bits}
}

// Iter yields the set's states in ascending bit order.
func (s StateSet) Iter() iter.Seq[State] {
	return func(yield func(State) bool) {
		for i := range len(stateNames) {
			st := State(1) << i
			if s.bits&uint64(st) != 0 && !yield(st) {
				return
			}
		}
	}
}

// Interface is one AT-SPI D-Bus interface; each variant is a distinct bit.
type Interface uint32

// Interface variants.
const (
	InterfaceAccessible Interface = 1 << iota
	InterfaceAction
	InterfaceApplication
	InterfaceCache
	InterfaceCollection
	InterfaceComponent
	InterfaceDocument
	InterfaceDeviceEventController
	InterfaceDeviceEventListener
	InterfaceEditableText
	InterfaceHyperlink
	InterfaceHypertext
	InterfaceImage
	InterfaceRegistry
	InterfaceSelection
	InterfaceSocket
	InterfaceTable
	InterfaceTableCell
	InterfaceText
	InterfaceValue
)

var interfaceNames = [...]string{
	"org.a11y.atspi.Accessible",
	"org.a11y.atspi.Action",
	"org.a11y.atspi.Application",
	"org.a11y.atspi.Cache",
	"org.a11y.atspi.Collection",
	"org.a11y.atspi.Component",
	"org.a11y.atspi.Document",
	"org.a11y.atspi.DeviceEventController",
	"org.a11y.atspi.DeviceEventListener",
	"org.a11y.atspi.EditableText",
	"org.a11y.atspi.Hyperlink",
	"org.a11y.atspi.Hypertext",
	"org.a11y.atspi.Image",
	"org.a11y.atspi.Registry",
	"org.a11y.atspi.Selection",
	"org.a11y.atspi.Socket",
	"org.a11y.atspi.Table",
	"org.a11y.atspi.TableCell",
	"org.a11y.atspi.Text",
	"org.a11y.atspi.Value",
}

// Name returns the interface's D-Bus name.
func (i Interface) Name() string {
	for n := range interfaceNames {
		if i == Interface(1)<<n {
			return interfaceNames[n]
		}
	}
	return ""
}

// InterfaceSet is a set of Interface flags.
type InterfaceSet struct {
	bits uint32
}

// NewInterfaceSet returns a set holding the flags in value.
func NewInterfaceSet(value Interface) InterfaceSet {
	return InterfaceSet{bits: uint32(value)}
}

// InterfaceSetEmpty returns the empty set.
func InterfaceSetEmpty() InterfaceSet {
	return InterfaceSet{}
}

// InterfaceSetAll returns the set of every interface.
func InterfaceSetAll() InterfaceSet {
	return InterfaceSet{bits: 1<<len(interfaceNames) - 1}
}

// Bits returns the set's bits.
func (s InterfaceSet) Bits() uint32 {
	return s.bits
}

// Contains reports whether every flag in other is in the set.
func (s InterfaceSet) Contains(other Interface) bool {
	return s.bits&uint32(other) == uint32(other)
}

// Intersection returns the flags in both s and other (upstream's BitAnd).
func (s InterfaceSet) Intersection(other InterfaceSet) InterfaceSet {
	return InterfaceSet{bits: s.bits & other.bits}
}

// SymmetricDifference returns the flags in exactly one of s and other (upstream's BitXor).
func (s InterfaceSet) SymmetricDifference(other InterfaceSet) InterfaceSet {
	return InterfaceSet{bits: s.bits ^ other.bits}
}

// Insert adds the flags in other.
func (s *InterfaceSet) Insert(other Interface) {
	s.bits |= uint32(other)
}

// Iter yields the set's interfaces in ascending bit order.
func (s InterfaceSet) Iter() iter.Seq[Interface] {
	return func(yield func(Interface) bool) {
		for n := range len(interfaceNames) {
			i := Interface(1) << n
			if s.bits&uint32(i) != 0 && !yield(i) {
				return
			}
		}
	}
}

// CoordType is the coordinate system of a position.
type CoordType uint32

// CoordType variants.
const (
	CoordTypeScreen CoordType = iota
	CoordTypeWindow
	CoordTypeParent
)

// Granularity is a unit of text.
type Granularity uint32

// Granularity variants.
const (
	GranularityChar Granularity = iota
	GranularityWord
	GranularitySentence
	GranularityLine
	GranularityParagraph
)

// Layer is the layer a component is painted in.
type Layer uint32

// Layer variants.
const (
	LayerInvalid Layer = iota
	LayerBackground
	LayerCanvas
	LayerWidget
	LayerMdi
	LayerPopup
	LayerOverlay
	LayerWindow
)

// ScrollType is where scrolling should place an object.
type ScrollType uint32

// ScrollType variants.
const (
	ScrollTypeTopLeft ScrollType = iota
	ScrollTypeBottomRight
	ScrollTypeTopEdge
	ScrollTypeBottomEdge
	ScrollTypeLeftEdge
	ScrollTypeRightEdge
	ScrollTypeAnywhere
)

// Politeness is the urgency of an announcement.
type Politeness int32

// Politeness variants.
const (
	PolitenessNone      Politeness = 0
	PolitenessPolite    Politeness = 1
	PolitenessAssertive Politeness = 2
)

// RelationType is the kind of a relation between objects.
type RelationType uint32

// RelationType variants.
const (
	RelationTypeNull RelationType = iota
	RelationTypeLabelFor
	RelationTypeLabelledBy
	RelationTypeControllerFor
	RelationTypeControlledBy
	RelationTypeMemberOf
	RelationTypeTooltipFor
	RelationTypeNodeChildOf
	RelationTypeNodeParentOf
	RelationTypeExtended
	RelationTypeFlowsTo
	RelationTypeFlowsFrom
	RelationTypeSubwindowOf
	RelationTypeEmbeds
	RelationTypeEmbeddedBy
	RelationTypePopupFor
	RelationTypeParentWindowOf
	RelationTypeDescriptionFor
	RelationTypeDescribedBy
	RelationTypeDetails
	RelationTypeDetailsFor
	RelationTypeErrorMessage
	RelationTypeErrorFor
)
