//go:build windows

// Package windows is a port of accesskit_windows 0.34.0: UI Automation providers over an
// accesskit_consumer tree.
package windows

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

type uiaPropertyID = int32
type uiaPatternID = int32
type uiaEventID = int32
type uiaControlTypeID = int32

var (
	iidIRawElementProviderSimple       = com.MustGUID("{D6DD68D1-86FD-4332-8666-9ABEDEA2D24C}")
	iidIRawElementProviderFragment     = com.MustGUID("{F7063DA8-8359-439C-9297-BBC5299A7D87}")
	iidIRawElementProviderFragmentRoot = com.MustGUID("{620CE2A5-AB8F-40A9-86CB-DE3C75599B58}")
	iidIToggleProvider                 = com.MustGUID("{56D00BD0-C4F4-433C-A836-1A52A57E0892}")
	iidIInvokeProvider                 = com.MustGUID("{54FCB24B-E18E-47A2-B4D3-ECCBE77599A2}")
	iidIValueProvider                  = com.MustGUID("{C7935180-6FB3-4201-B174-7DF73ADBF64A}")
	iidIRangeValueProvider             = com.MustGUID("{36DC7AEF-33E6-4691-AFE1-2BE7274B3D33}")
	iidIScrollItemProvider             = com.MustGUID("{2360C714-4BF1-4B26-BA65-9B21316127EB}")
	iidISelectionItemProvider          = com.MustGUID("{2ACAD808-B2D4-452D-A407-91FF1AD167B2}")
	iidISelectionProvider              = com.MustGUID("{FB8B03AF-3BDF-48D4-BD36-1A65793BE168}")
	iidITextProvider                   = com.MustGUID("{3589C92C-63F3-4367-99BB-ADA653B77CF2}")
	iidIExpandCollapseProvider         = com.MustGUID("{D847D3A5-CAB0-4A98-8C32-ECB45C59AD24}")
	iidIWindowProvider                 = com.MustGUID("{987DF77B-DB06-4D77-8F8A-86A9C3BB90B9}")
	iidITextRangeProvider              = com.MustGUID("{5347AD7B-C355-46F8-AFF5-909033582F63}")
)

const (
	uiaAppendRuntimeID = 3

	providerOptionsServerSideProvider = 2

	navigateDirectionParent          = 0
	navigateDirectionNextSibling     = 1
	navigateDirectionPreviousSibling = 2
	navigateDirectionFirstChild      = 3
	navigateDirectionLastChild       = 4

	toggleStateOff           = 0
	toggleStateOn            = 1
	toggleStateIndeterminate = 2

	expandCollapseStateCollapsed = 0
	expandCollapseStateExpanded  = 1
	expandCollapseStateLeafNode  = 3

	liveSettingOff       = 0
	liveSettingPolite    = 1
	liveSettingAssertive = 2

	orientationTypeNone       = 0
	orientationTypeHorizontal = 1
	orientationTypeVertical   = 2

	supportedTextSelectionNone   = 0
	supportedTextSelectionSingle = 1

	windowInteractionStateReadyForUserInteraction = 2

	textDecorationLineStyleSingle = 1
	textDecorationLineStyleDouble = 3
	textDecorationLineStyleDot    = 4
	textDecorationLineStyleDash   = 5
	textDecorationLineStyleWavy   = 8

	horizontalTextAlignmentLeft      = 0
	horizontalTextAlignmentCentered  = 1
	horizontalTextAlignmentRight     = 2
	horizontalTextAlignmentJustified = 3

	uiaEElementNotEnabled   = com.HRESULT(-0x7FFBFE00)
	uiaEElementNotAvailable = com.HRESULT(-0x7FFBFDFF)
	uiaENotSupported        = com.HRESULT(-0x7FFBFDFC)
	uiaEInvalidOperation    = com.HRESULT(-0x7FECEAF7)

	localeAllowNeutralNames = 0x08000000

	wmGetText       = 0x000D
	wmGetTextLength = 0x000E
)

const (
	uiaInvokePatternID         uiaPatternID = 10000
	uiaSelectionPatternID      uiaPatternID = 10001
	uiaValuePatternID          uiaPatternID = 10002
	uiaRangeValuePatternID     uiaPatternID = 10003
	uiaExpandCollapsePatternID uiaPatternID = 10005
	uiaWindowPatternID         uiaPatternID = 10009
	uiaSelectionItemPatternID  uiaPatternID = 10010
	uiaTextPatternID           uiaPatternID = 10014
	uiaTogglePatternID         uiaPatternID = 10015
	uiaScrollItemPatternID     uiaPatternID = 10017
)

const (
	uiaToolTipOpenedEventID                            uiaEventID = 20000
	uiaAutomationFocusChangedEventID                   uiaEventID = 20005
	uiaSelectionItemElementAddedToSelectionEventID     uiaEventID = 20010
	uiaSelectionItemElementRemovedFromSelectionEventID uiaEventID = 20011
	uiaSelectionItemElementSelectedEventID             uiaEventID = 20012
	uiaSelectionInvalidatedEventID                     uiaEventID = 20013
	uiaTextTextSelectionChangedEventID                 uiaEventID = 20014
	uiaTextTextChangedEventID                          uiaEventID = 20015
	uiaWindowWindowOpenedEventID                       uiaEventID = 20016
	uiaWindowWindowClosedEventID                       uiaEventID = 20017
	uiaLiveRegionChangedEventID                        uiaEventID = 20024
)

type uiaTextAttributeID = int32

const (
	uiaBackgroundColorAttributeID         uiaTextAttributeID = 40001
	uiaCultureAttributeID                 uiaTextAttributeID = 40004
	uiaFontNameAttributeID                uiaTextAttributeID = 40005
	uiaFontSizeAttributeID                uiaTextAttributeID = 40006
	uiaFontWeightAttributeID              uiaTextAttributeID = 40007
	uiaForegroundColorAttributeID         uiaTextAttributeID = 40008
	uiaHorizontalTextAlignmentAttributeID uiaTextAttributeID = 40009
	uiaIsItalicAttributeID                uiaTextAttributeID = 40014
	uiaIsReadOnlyAttributeID              uiaTextAttributeID = 40015
	uiaIsSubscriptAttributeID             uiaTextAttributeID = 40016
	uiaIsSuperscriptAttributeID           uiaTextAttributeID = 40017
	uiaOverlineColorAttributeID           uiaTextAttributeID = 40023
	uiaOverlineStyleAttributeID           uiaTextAttributeID = 40024
	uiaStrikethroughColorAttributeID      uiaTextAttributeID = 40025
	uiaStrikethroughStyleAttributeID      uiaTextAttributeID = 40026
	uiaUnderlineColorAttributeID          uiaTextAttributeID = 40029
	uiaUnderlineStyleAttributeID          uiaTextAttributeID = 40030
	uiaCaretPositionAttributeID           uiaTextAttributeID = 40038
)

const (
	caretPositionUnknown         = 0
	caretPositionEndOfLine       = 1
	caretPositionBeginningOfLine = 2

	textPatternRangeEndpointStart = 0
	textPatternRangeEndpointEnd   = 1

	textUnitCharacter = 0
	textUnitFormat    = 1
	textUnitWord      = 2
	textUnitLine      = 3
	textUnitParagraph = 4
	textUnitPage      = 5
	textUnitDocument  = 6

	uiaRootObjectID = -25
	objidClient     = -4

	automationIdentifierTypeProperty = 0

	gwlpWndProc     = -4
	wmSetFocus      = 0x0007
	wmKillFocus     = 0x0008
	wmGetObject     = 0x003D
	wmNCDestroy     = 0x0082
	wmEnterMenuLoop = 0x0211
	wmExitMenuLoop  = 0x0212
	wmEnterSizeMove = 0x0231
	wmExitSizeMove  = 0x0232
)

var controlTypePropertyGUID = com.MustGUID("{CA774FEA-28AC-4BC2-94CA-ACEC6D6C10A3}")

const uiaSelectionItemIsSelectedPropertyID uiaPropertyID = 30079

const (
	uiaControlTypePropertyID                       uiaPropertyID = 30003
	uiaLocalizedControlTypePropertyID              uiaPropertyID = 30004
	uiaNamePropertyID                              uiaPropertyID = 30005
	uiaHasKeyboardFocusPropertyID                  uiaPropertyID = 30008
	uiaIsKeyboardFocusablePropertyID               uiaPropertyID = 30009
	uiaIsEnabledPropertyID                         uiaPropertyID = 30010
	uiaAutomationIDPropertyID                      uiaPropertyID = 30011
	uiaClassNamePropertyID                         uiaPropertyID = 30012
	uiaHelpTextPropertyID                          uiaPropertyID = 30013
	uiaCulturePropertyID                           uiaPropertyID = 30015
	uiaIsControlElementPropertyID                  uiaPropertyID = 30016
	uiaIsContentElementPropertyID                  uiaPropertyID = 30017
	uiaIsPasswordPropertyID                        uiaPropertyID = 30019
	uiaNativeWindowHandlePropertyID                uiaPropertyID = 30020
	uiaOrientationPropertyID                       uiaPropertyID = 30023
	uiaFrameworkIDPropertyID                       uiaPropertyID = 30024
	uiaIsRequiredForFormPropertyID                 uiaPropertyID = 30025
	uiaValueValuePropertyID                        uiaPropertyID = 30045
	uiaValueIsReadOnlyPropertyID                   uiaPropertyID = 30046
	uiaRangeValueValuePropertyID                   uiaPropertyID = 30047
	uiaRangeValueIsReadOnlyPropertyID              uiaPropertyID = 30048
	uiaRangeValueMinimumPropertyID                 uiaPropertyID = 30049
	uiaRangeValueMaximumPropertyID                 uiaPropertyID = 30050
	uiaRangeValueLargeChangePropertyID             uiaPropertyID = 30051
	uiaRangeValueSmallChangePropertyID             uiaPropertyID = 30052
	uiaSelectionCanSelectMultiplePropertyID        uiaPropertyID = 30060
	uiaSelectionIsSelectionRequiredPropertyID      uiaPropertyID = 30061
	uiaExpandCollapseExpandCollapseStatePropertyID uiaPropertyID = 30070
	uiaWindowIsModalPropertyID                     uiaPropertyID = 30077
	uiaToggleToggleStatePropertyID                 uiaPropertyID = 30086
	uiaAriaRolePropertyID                          uiaPropertyID = 30101
	uiaAriaPropertiesPropertyID                    uiaPropertyID = 30102
	uiaControllerForPropertyID                     uiaPropertyID = 30104
	uiaProviderDescriptionPropertyID               uiaPropertyID = 30107
	uiaLiveSettingPropertyID                       uiaPropertyID = 30135
	uiaPositionInSetPropertyID                     uiaPropertyID = 30152
	uiaSizeOfSetPropertyID                         uiaPropertyID = 30153
	uiaLevelPropertyID                             uiaPropertyID = 30154
	uiaFullDescriptionPropertyID                   uiaPropertyID = 30159
	uiaIsDialogPropertyID                          uiaPropertyID = 30174
)

const (
	uiaButtonControlTypeID      uiaControlTypeID = 50000
	uiaCheckBoxControlTypeID    uiaControlTypeID = 50002
	uiaComboBoxControlTypeID    uiaControlTypeID = 50003
	uiaEditControlTypeID        uiaControlTypeID = 50004
	uiaHyperlinkControlTypeID   uiaControlTypeID = 50005
	uiaImageControlTypeID       uiaControlTypeID = 50006
	uiaListItemControlTypeID    uiaControlTypeID = 50007
	uiaListControlTypeID        uiaControlTypeID = 50008
	uiaMenuControlTypeID        uiaControlTypeID = 50009
	uiaMenuBarControlTypeID     uiaControlTypeID = 50010
	uiaMenuItemControlTypeID    uiaControlTypeID = 50011
	uiaProgressBarControlTypeID uiaControlTypeID = 50012
	uiaRadioButtonControlTypeID uiaControlTypeID = 50013
	uiaScrollBarControlTypeID   uiaControlTypeID = 50014
	uiaSliderControlTypeID      uiaControlTypeID = 50015
	uiaSpinnerControlTypeID     uiaControlTypeID = 50016
	uiaStatusBarControlTypeID   uiaControlTypeID = 50017
	uiaTabControlTypeID         uiaControlTypeID = 50018
	uiaTabItemControlTypeID     uiaControlTypeID = 50019
	uiaTextControlTypeID        uiaControlTypeID = 50020
	uiaToolBarControlTypeID     uiaControlTypeID = 50021
	uiaToolTipControlTypeID     uiaControlTypeID = 50022
	uiaTreeControlTypeID        uiaControlTypeID = 50023
	uiaTreeItemControlTypeID    uiaControlTypeID = 50024
	uiaCustomControlTypeID      uiaControlTypeID = 50025
	uiaGroupControlTypeID       uiaControlTypeID = 50026
	uiaDataGridControlTypeID    uiaControlTypeID = 50028
	uiaDataItemControlTypeID    uiaControlTypeID = 50029
	uiaDocumentControlTypeID    uiaControlTypeID = 50030
	uiaWindowControlTypeID      uiaControlTypeID = 50032
	uiaPaneControlTypeID        uiaControlTypeID = 50033
	uiaTableControlTypeID       uiaControlTypeID = 50036
	uiaSeparatorControlTypeID   uiaControlTypeID = 50038
)

type uiaRect struct {
	Left   float64
	Top    float64
	Width  float64
	Height float64
}

type uiaPoint struct {
	X float64
	Y float64
}

var (
	uiautomationcore = windows.NewLazySystemDLL("uiautomationcore.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	oleaut32         = windows.NewLazySystemDLL("oleaut32.dll")

	procUiaHostProviderFromHwnd           = uiautomationcore.NewProc("UiaHostProviderFromHwnd")
	procUiaGetReservedMixedAttributeValue = uiautomationcore.NewProc("UiaGetReservedMixedAttributeValue")
	procUiaGetReservedNotSupportedValue   = uiautomationcore.NewProc("UiaGetReservedNotSupportedValue")
	procUiaLookupID                       = uiautomationcore.NewProc("UiaLookupId")
	procUiaReturnRawElementProvider       = uiautomationcore.NewProc("UiaReturnRawElementProvider")
	procUiaRaiseAutomationEvent           = uiautomationcore.NewProc("UiaRaiseAutomationEvent")
	procUiaRaiseAutomationPropertyChanged = uiautomationcore.NewProc("UiaRaiseAutomationPropertyChangedEvent")
	procLocaleNameToLCID                  = kernel32.NewProc("LocaleNameToLCID")
	user32                                = windows.NewLazySystemDLL("user32.dll")
	procCallWindowProcW                   = user32.NewProc("CallWindowProcW")
	procSetWindowLongPtrW                 = user32.NewProc("SetWindowLongPtrW")
	procSysAllocStringLen                 = oleaut32.NewProc("SysAllocStringLen")
)

func uiaHostProviderFromHwnd(hwnd windows.HWND) (*com.IUnknown, error) {
	var out *com.IUnknown
	r, _, _ := procUiaHostProviderFromHwnd.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func uiaGetReservedMixedAttributeValue() (*com.IUnknown, error) {
	var out *com.IUnknown
	r, _, _ := procUiaGetReservedMixedAttributeValue.Call(uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func localeNameToLCID(name string, flags uint32) uint32 {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procLocaleNameToLCID.Call(uintptr(unsafe.Pointer(p)), uintptr(flags))
	return uint32(r)
}

func bstrFromWide(wide []uint16) com.BSTR {
	var first *uint16
	if len(wide) > 0 {
		first = &wide[0]
	}
	r, _, _ := procSysAllocStringLen.Call(uintptr(unsafe.Pointer(first)), uintptr(len(wide)))
	if r == 0 {
		panic("BSTR allocation failed")
	}
	return com.BSTR((*uint16)(com.PtrFromUintptr(r)))
}

func uiaGetReservedNotSupportedValue() (*com.IUnknown, error) {
	var out *com.IUnknown
	r, _, _ := procUiaGetReservedNotSupportedValue.Call(uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func uiaLookupID(kind int32, guid *com.GUID) int32 {
	r, _, _ := procUiaLookupID.Call(uintptr(kind), uintptr(unsafe.Pointer(guid)))
	return int32(r)
}

func uiaReturnRawElementProvider(hwnd windows.HWND, wparam WPARAM, lparam LPARAM, el *com.IUnknown) LRESULT {
	r, _, _ := procUiaReturnRawElementProvider.Call(uintptr(hwnd), uintptr(wparam), uintptr(lparam), uintptr(unsafe.Pointer(el)))
	return LRESULT(r)
}

func uiaRaiseAutomationEvent(element *com.IUnknown, eventID uiaEventID) error {
	r, _, _ := procUiaRaiseAutomationEvent.Call(uintptr(unsafe.Pointer(element)), uintptr(eventID))
	return com.HRESULTFromUintptr(r).Err()
}

func uiaRaiseAutomationPropertyChangedEvent(element *com.IUnknown, propertyID uiaPropertyID, oldValue, newValue *com.VARIANT) error {
	r, _, _ := procUiaRaiseAutomationPropertyChanged.Call(uintptr(unsafe.Pointer(element)), uintptr(propertyID), uintptr(unsafe.Pointer(oldValue)), uintptr(unsafe.Pointer(newValue)))
	return com.HRESULTFromUintptr(r).Err()
}

func callWindowProcW(prev uintptr, hwnd windows.HWND, message uint32, wparam WPARAM, lparam LPARAM) LRESULT {
	r, _, _ := procCallWindowProcW.Call(prev, uintptr(hwnd), uintptr(message), uintptr(wparam), uintptr(lparam))
	return LRESULT(r)
}

func setWindowLongPtrW(hwnd windows.HWND, index int32, value uintptr) (uintptr, error) {
	r, _, err := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index), value)
	if r == 0 {
		return 0, err
	}
	return r, nil
}
