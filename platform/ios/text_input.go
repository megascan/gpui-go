//go:build ios

package ios

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"unicode/utf16"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type textInputSession struct{ _ byte }

type textPositionIvars struct {
	offset  int
	session *textInputSession
}

type textRangeIvars struct {
	textRange gpui.Range
	session   *textInputSession
}

type selectionRectIvars struct {
	rect          objc.CGRect
	containsStart bool
	containsEnd   bool
}

const (
	uiKeyboardTypeDefault objc.NSInteger = 0

	uiTextAutocorrectionTypeNo  objc.NSInteger = 1
	uiTextAutocorrectionTypeYes objc.NSInteger = 2

	uiTextAutocapitalizationTypeNone          objc.NSInteger = 0
	uiTextAutocapitalizationTypeWords         objc.NSInteger = 1
	uiTextAutocapitalizationTypeSentences     objc.NSInteger = 2
	uiTextAutocapitalizationTypeAllCharacters objc.NSInteger = 3

	uiTextSpellCheckingTypeNo  objc.NSInteger = 1
	uiTextSpellCheckingTypeYes objc.NSInteger = 2

	uiTextInlinePredictionTypeNo  objc.NSInteger = 1
	uiTextInlinePredictionTypeYes objc.NSInteger = 2

	uiReturnKeyTypeDefault objc.NSInteger = 0
	uiReturnKeyTypeGo      objc.NSInteger = 1
	uiReturnKeyTypeNext    objc.NSInteger = 4
	uiReturnKeyTypeSearch  objc.NSInteger = 6
	uiReturnKeyTypeSend    objc.NSInteger = 7
	uiReturnKeyTypeDone    objc.NSInteger = 9

	nsWritingDirectionNatural objc.NSInteger = -1

	uiTextLayoutDirectionRight objc.NSInteger = 2
	uiTextLayoutDirectionLeft  objc.NSInteger = 3
	uiTextLayoutDirectionUp    objc.NSInteger = 4
	uiTextLayoutDirectionDown  objc.NSInteger = 5

	nsOrderedAscending  objc.NSInteger = -1
	nsOrderedSame       objc.NSInteger = 0
	nsOrderedDescending objc.NSInteger = 1
)

type textInputIvars struct {
	window               windowReference
	inputField           gpui.Option[gpui.FocusID]
	session              *textInputSession
	configuration        gpui.TextInputConfiguration
	delegate             *uintptr
	markedStyle          *objc.Retained
	keyboardDesired      bool
	keyboardEligible     bool
	reconcilingResponder bool
	updatePending        bool
	configurationChanged bool
	selectionChanged     bool
	contentChanged       bool
	nativeEdit           bool
}

var (
	textInputObjectsMu sync.Mutex
	textInputViews     = map[objc.ID]*textInputIvars{}
	textPositions      = map[objc.ID]textPositionIvars{}
	textRanges         = map[objc.ID]textRangeIvars{}
	selectionRects     = map[objc.ID]selectionRectIvars{}
)

var (
	weakOnce             sync.Once
	objcStoreWeak        func(location *uintptr, object objc.ID) objc.ID
	objcLoadWeakRetained func(location *uintptr) objc.ID
	objcDestroyWeak      func(location *uintptr)
)

func loadWeakFunctions() {
	weakOnce.Do(func() {
		lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			panic(err)
		}
		purego.RegisterLibFunc(&objcStoreWeak, lib, "objc_storeWeak")
		purego.RegisterLibFunc(&objcLoadWeakRetained, lib, "objc_loadWeakRetained")
		purego.RegisterLibFunc(&objcDestroyWeak, lib, "objc_destroyWeak")
	})
}

func textInputIvarsFor(this objc.ID) *textInputIvars {
	textInputObjectsMu.Lock()
	defer textInputObjectsMu.Unlock()
	ivars, ok := textInputViews[this]
	if !ok {
		ivars = &textInputIvars{session: &textInputSession{}, delegate: new(uintptr)}
		textInputViews[this] = ivars
	}
	return ivars
}

const (
	textPositionClassName  = "GPUITextPosition"
	textRangeClassName     = "GPUITextRange"
	selectionRectClassName = "GPUITextSelectionRect"
	textInputViewClassName = "GPUITextInputView"
)

var (
	textPositionClassOnce  sync.Once
	textPositionClassVal   objc.Class
	textRangeClassOnce     sync.Once
	textRangeClassVal      objc.Class
	selectionRectClassOnce sync.Once
	selectionRectClassVal  objc.Class
	textInputViewClassOnce sync.Once
	textInputViewClassVal  objc.Class
)

func mustRegisterTextClass(name, superclass string, protocols []string, methods []objc.MethodDef) objc.Class {
	class, err := objc.RegisterClass(name, objc.GetClass(superclass), protocols, nil, methods)
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
	return class
}

func textPositionClass() objc.Class {
	textPositionClassOnce.Do(func() {
		textPositionClassVal = mustRegisterTextClass(textPositionClassName, "UITextPosition", []string{"NSObject"}, []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: func(this objc.ID, _ objc.SEL) {
				textInputObjectsMu.Lock()
				delete(textPositions, this)
				textInputObjectsMu.Unlock()
				objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
			}},
		})
	})
	return textPositionClassVal
}

func newTextPosition(offset int, session *textInputSession) objc.ID {
	this := objc.Send[objc.ID](objc.Alloc(textPositionClass()), objc.Sel("init"))
	textInputObjectsMu.Lock()
	textPositions[this] = textPositionIvars{offset: offset, session: session}
	textInputObjectsMu.Unlock()
	return objc.Autorelease(this)
}

func textRangeClass() objc.Class {
	textRangeClassOnce.Do(func() {
		textRangeClassVal = mustRegisterTextClass(textRangeClassName, "UITextRange", []string{"NSObject"}, []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: func(this objc.ID, _ objc.SEL) {
				textInputObjectsMu.Lock()
				delete(textRanges, this)
				textInputObjectsMu.Unlock()
				objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
			}},
			{Cmd: objc.Sel("start"), Fn: func(this objc.ID, _ objc.SEL) objc.ID {
				ivars := textRangeIvarsFor(this)
				return newTextPosition(ivars.textRange.Start, ivars.session)
			}},
			{Cmd: objc.Sel("end"), Fn: func(this objc.ID, _ objc.SEL) objc.ID {
				ivars := textRangeIvarsFor(this)
				return newTextPosition(ivars.textRange.End, ivars.session)
			}},
			{Cmd: objc.Sel("isEmpty"), Fn: func(this objc.ID, _ objc.SEL) bool {
				r := textRangeIvarsFor(this).textRange
				return r.Start >= r.End
			}},
		})
	})
	return textRangeClassVal
}

func textRangeIvarsFor(this objc.ID) textRangeIvars {
	textInputObjectsMu.Lock()
	defer textInputObjectsMu.Unlock()
	return textRanges[this]
}

func newTextRange(r gpui.Range, session *textInputSession) objc.ID {
	this := objc.Send[objc.ID](objc.Alloc(textRangeClass()), objc.Sel("init"))
	textInputObjectsMu.Lock()
	textRanges[this] = textRangeIvars{textRange: r, session: session}
	textInputObjectsMu.Unlock()
	return objc.Autorelease(this)
}

func selectionRectClass() objc.Class {
	selectionRectClassOnce.Do(func() {
		ivars := func(this objc.ID) selectionRectIvars {
			textInputObjectsMu.Lock()
			defer textInputObjectsMu.Unlock()
			return selectionRects[this]
		}
		selectionRectClassVal = mustRegisterTextClass(selectionRectClassName, "UITextSelectionRect", []string{"NSObject"}, []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: func(this objc.ID, _ objc.SEL) {
				textInputObjectsMu.Lock()
				delete(selectionRects, this)
				textInputObjectsMu.Unlock()
				objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
			}},
			{Cmd: objc.Sel("rect"), Fn: func(this objc.ID, _ objc.SEL) objc.CGRect { return ivars(this).rect }},
			{Cmd: objc.Sel("containsStart"), Fn: func(this objc.ID, _ objc.SEL) bool { return ivars(this).containsStart }},
			{Cmd: objc.Sel("containsEnd"), Fn: func(this objc.ID, _ objc.SEL) bool { return ivars(this).containsEnd }},
			{Cmd: objc.Sel("isVertical"), Fn: func(_ objc.ID, _ objc.SEL) bool { return false }},
			{Cmd: objc.Sel("writingDirection"), Fn: func(_ objc.ID, _ objc.SEL) objc.NSInteger { return nsWritingDirectionNatural }},
		})
	})
	return selectionRectClassVal
}

type textInputView struct {
	id    objc.ID
	ivars *textInputIvars
}

func textInputViewFor(this objc.ID) textInputView {
	return textInputView{id: this, ivars: textInputIvarsFor(this)}
}

func textInputViewClass() objc.Class {
	textInputViewClassOnce.Do(func() {
		v := textInputViewFor
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("dealloc"), Fn: textInputViewDealloc},
			{Cmd: objc.Sel("canBecomeFirstResponder"), Fn: func(this objc.ID, _ objc.SEL) bool { return v(this).canBecomeFirstResponder() }},
			{Cmd: objc.Sel("resignFirstResponder"), Fn: func(this objc.ID, _ objc.SEL) bool { return v(this).resignFirstResponder() }},
			{Cmd: objc.Sel("reconcileTextInput"), Fn: func(this objc.ID, _ objc.SEL) { v(this).reconcileTextInput() }},
			{Cmd: objc.Sel("hasText"), Fn: func(this objc.ID, _ objc.SEL) bool { return v(this).hasText() }},
			{Cmd: objc.Sel("insertText:"), Fn: func(this objc.ID, _ objc.SEL, text objc.ID) { v(this).insertText(text) }},
			{Cmd: objc.Sel("deleteBackward"), Fn: func(this objc.ID, _ objc.SEL) { v(this).deleteBackward() }},
			{Cmd: objc.Sel("keyboardType"), Fn: func(_ objc.ID, _ objc.SEL) objc.NSInteger { return uiKeyboardTypeDefault }},
			{Cmd: objc.Sel("autocorrectionType"), Fn: func(this objc.ID, _ objc.SEL) objc.NSInteger { return v(this).autocorrectionType() }},
			{Cmd: objc.Sel("autocapitalizationType"), Fn: func(this objc.ID, _ objc.SEL) objc.NSInteger { return v(this).autocapitalizationType() }},
			{Cmd: objc.Sel("spellCheckingType"), Fn: func(this objc.ID, _ objc.SEL) objc.NSInteger { return v(this).spellCheckingType() }},
			{Cmd: objc.Sel("inlinePredictionType"), Fn: func(this objc.ID, _ objc.SEL) objc.NSInteger { return v(this).inlinePredictionType() }},
			{Cmd: objc.Sel("returnKeyType"), Fn: func(this objc.ID, _ objc.SEL) objc.NSInteger { return v(this).returnKeyType() }},
			{Cmd: objc.Sel("textInRange:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID) objc.ID { return v(this).textInRange(r) }},
			{Cmd: objc.Sel("replaceRange:withText:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID, text objc.ID) { v(this).replaceRange(r, text) }},
			{Cmd: objc.Sel("selectedTextRange"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).selectedTextRange() }},
			{Cmd: objc.Sel("setSelectedTextRange:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID) { v(this).setSelectedTextRange(r) }},
			{Cmd: objc.Sel("markedTextRange"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).markedTextRange() }},
			{Cmd: objc.Sel("markedTextStyle"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).markedTextStyle() }},
			{Cmd: objc.Sel("setMarkedTextStyle:"), Fn: func(this objc.ID, _ objc.SEL, style objc.ID) { v(this).setMarkedTextStyle(style) }},
			{Cmd: objc.Sel("setMarkedText:selectedRange:"), Fn: func(this objc.ID, _ objc.SEL, text objc.ID, selection objc.NSRange) {
				v(this).setMarkedText(text, selection)
			}},
			{Cmd: objc.Sel("unmarkText"), Fn: func(this objc.ID, _ objc.SEL) { v(this).unmarkText() }},
			{Cmd: objc.Sel("beginningOfDocument"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).beginningOfDocument() }},
			{Cmd: objc.Sel("endOfDocument"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).endOfDocument() }},
			{Cmd: objc.Sel("textRangeFromPosition:toPosition:"), Fn: func(this objc.ID, _ objc.SEL, start, end objc.ID) objc.ID {
				return v(this).textRangeFromPosition(start, end)
			}},
			{Cmd: objc.Sel("positionFromPosition:offset:"), Fn: func(this objc.ID, _ objc.SEL, position objc.ID, offset objc.NSInteger) objc.ID {
				return v(this).positionFromPositionOffset(position, offset)
			}},
			{Cmd: objc.Sel("positionFromPosition:inDirection:offset:"), Fn: func(this objc.ID, _ objc.SEL, position objc.ID, direction objc.NSInteger, offset objc.NSInteger) objc.ID {
				return v(this).positionFromPositionInDirection(position, direction, offset)
			}},
			{Cmd: objc.Sel("comparePosition:toPosition:"), Fn: func(this objc.ID, _ objc.SEL, position, other objc.ID) objc.NSInteger {
				return v(this).comparePosition(position, other)
			}},
			{Cmd: objc.Sel("offsetFromPosition:toPosition:"), Fn: func(this objc.ID, _ objc.SEL, start, end objc.ID) objc.NSInteger {
				return v(this).offsetFromPosition(start, end)
			}},
			{Cmd: objc.Sel("inputDelegate"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).inputDelegate() }},
			{Cmd: objc.Sel("setInputDelegate:"), Fn: func(this objc.ID, _ objc.SEL, delegate objc.ID) { v(this).setInputDelegate(delegate) }},
			{Cmd: objc.Sel("tokenizer"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).tokenizer() }},
			{Cmd: objc.Sel("positionWithinRange:farthestInDirection:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID, direction objc.NSInteger) objc.ID {
				return v(this).positionWithinRangeFarthestInDirection(r, direction)
			}},
			{Cmd: objc.Sel("characterRangeByExtendingPosition:inDirection:"), Fn: func(this objc.ID, _ objc.SEL, position objc.ID, direction objc.NSInteger) objc.ID {
				return v(this).characterRangeByExtendingPosition(position, direction)
			}},
			{Cmd: objc.Sel("baseWritingDirectionForPosition:inDirection:"), Fn: func(this objc.ID, _ objc.SEL, position objc.ID, _ objc.NSInteger) objc.NSInteger {
				return v(this).baseWritingDirection(position)
			}},
			{Cmd: objc.Sel("setBaseWritingDirection:forRange:"), Fn: func(this objc.ID, _ objc.SEL, direction objc.NSInteger, r objc.ID) {
				v(this).setBaseWritingDirection(direction, r)
			}},
			{Cmd: objc.Sel("firstRectForRange:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID) objc.CGRect { return v(this).firstRectForRange(r) }},
			{Cmd: objc.Sel("caretRectForPosition:"), Fn: func(this objc.ID, _ objc.SEL, position objc.ID) objc.CGRect {
				return v(this).caretRectForPosition(position)
			}},
			{Cmd: objc.Sel("selectionRectsForRange:"), Fn: func(this objc.ID, _ objc.SEL, r objc.ID) objc.ID { return v(this).selectionRectsForRange(r) }},
			{Cmd: objc.Sel("closestPositionToPoint:"), Fn: func(this objc.ID, _ objc.SEL, point objc.CGPoint) objc.ID {
				return v(this).closestPositionToPoint(point)
			}},
			{Cmd: objc.Sel("closestPositionToPoint:withinRange:"), Fn: func(this objc.ID, _ objc.SEL, point objc.CGPoint, r objc.ID) objc.ID {
				return v(this).closestPositionToPointWithinRange(point, r)
			}},
			{Cmd: objc.Sel("characterRangeAtPoint:"), Fn: func(this objc.ID, _ objc.SEL, point objc.CGPoint) objc.ID {
				return v(this).characterRangeAtPoint(point)
			}},
			{Cmd: objc.Sel("textInputView"), Fn: func(this objc.ID, _ objc.SEL) objc.ID { return v(this).textInputView() }},
		}
		textInputViewClassVal = mustRegisterTextClass(textInputViewClassName, "UIView", []string{"NSObject", "UIKeyInput", "UITextInputTraits", "UITextInput"}, methods)
	})
	return textInputViewClassVal
}

func textInputViewDealloc(this objc.ID, _ objc.SEL) {
	textInputObjectsMu.Lock()
	ivars, ok := textInputViews[this]
	delete(textInputViews, this)
	textInputObjectsMu.Unlock()
	if ok {
		loadWeakFunctions()
		objcDestroyWeak(ivars.delegate)
		if ivars.markedStyle != nil {
			ivars.markedStyle.Release()
		}
	}
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

func newTextInputView(frame objc.CGRect) objc.ID {
	this := objc.Send[objc.ID](objc.Alloc(textInputViewClass()), objc.Sel("initWithFrame:"), frame)
	textInputIvarsFor(this)
	return this
}

func (v textInputView) canBecomeFirstResponder() bool {
	return v.ivars.keyboardDesired && v.ivars.keyboardEligible
}

func (v textInputView) isFirstResponder() bool {
	return objc.Send[bool](v.id, objc.Sel("isFirstResponder"))
}

func (v textInputView) resignFirstResponder() bool {
	wasFirstResponder := v.isFirstResponder()
	resigned := objc.SendSuper[bool](v.id, objc.Sel("resignFirstResponder"))
	if resigned && wasFirstResponder && !v.ivars.reconcilingResponder {
		v.ivars.keyboardDesired = false
		v.ivars.keyboardEligible = false
	}
	return resigned
}

func (v textInputView) reconcileTextInput() {
	v.ivars.updatePending = false
	eligible := false
	if v.ivars.keyboardDesired {
		result, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) bool {
			if !handler.QueryAcceptsFocusedTextInput() {
				return false
			}
			document, ok := documentRange(handler)
			if !ok {
				return false
			}
			selection, ok := handler.SelectedTextRange(false)
			return ok && containsRange(document, selection.Range)
		})
		eligible = ok && result
	}
	v.ivars.keyboardEligible = eligible
	if objc.Send[bool](v.id, objc.Sel("canBecomeFirstResponder")) {
		if !v.isFirstResponder() {
			objc.Send[bool](v.id, objc.Sel("becomeFirstResponder"))
		}
	} else if v.isFirstResponder() {
		previous := v.ivars.reconcilingResponder
		v.ivars.reconcilingResponder = true
		objc.Send[bool](v.id, objc.Sel("resignFirstResponder"))
		v.ivars.reconcilingResponder = previous
	}
	configurationChanged := v.ivars.configurationChanged
	v.ivars.configurationChanged = false
	if configurationChanged && v.isFirstResponder() {
		objc.SendVoid(v.id, objc.Sel("reloadInputViews"))
	}
	contentChanged := v.ivars.contentChanged
	v.ivars.contentChanged = false
	selectionChanged := v.ivars.selectionChanged
	v.ivars.selectionChanged = false
	if contentChanged || selectionChanged {
		delegate := v.loadDelegate()
		if delegate != objc.Nil {
			if contentChanged {
				objc.SendVoid(delegate, objc.Sel("textWillChange:"), v.id)
			}
			objc.SendVoid(delegate, objc.Sel("selectionWillChange:"), v.id)
			objc.SendVoid(delegate, objc.Sel("selectionDidChange:"), v.id)
			if contentChanged {
				objc.SendVoid(delegate, objc.Sel("textDidChange:"), v.id)
			}
			objc.Release(delegate)
		}
	}
}

func (v textInputView) hasText() bool {
	r, ok := v.documentRange()
	return ok && r.Start < r.End
}

func (v textInputView) insertText(text objc.ID) {
	s := objc.GoString(text)
	v.edit(func(handler *gpui.PlatformInputHandler) {
		handler.ReplaceTextInRange(gpui.None[gpui.Range](), s)
	})
}

func (v textInputView) deleteBackward() {
	v.edit(func(handler *gpui.PlatformInputHandler) {
		selection, ok := handler.SelectedTextRange(false)
		if !ok {
			return
		}
		r := selection.Range
		if r.Start >= r.End {
			start := 0
			if editable, ok := handler.TextInputEditableRange(); ok {
				start = editable.Start
			}
			if r.Start <= start {
				return
			}
			text, ok := readText(handler, gpui.Range{Start: start, End: r.Start})
			if !ok {
				return
			}
			nsText := objc.NSStringFromGo(text)
			defer objc.Release(nsText)
			length := int(objc.Send[objc.NSUInteger](nsText, objc.Sel("length")))
			character, ok := composedCharacterRange(nsText, max(length-1, 0))
			if !ok {
				return
			}
			r.Start = start + character.Start
		}
		if allowedRange(handler, r) {
			handler.ReplaceTextInRange(gpui.Some(r), "")
		}
	})
}

func (v textInputView) autocorrectionType() objc.NSInteger {
	if v.ivars.configuration.Autocorrect {
		return uiTextAutocorrectionTypeYes
	}
	return uiTextAutocorrectionTypeNo
}

func (v textInputView) autocapitalizationType() objc.NSInteger {
	switch v.ivars.configuration.Autocapitalize {
	case gpui.AutocapitalizeWords:
		return uiTextAutocapitalizationTypeWords
	case gpui.AutocapitalizeSentences:
		return uiTextAutocapitalizationTypeSentences
	case gpui.AutocapitalizeCharacters:
		return uiTextAutocapitalizationTypeAllCharacters
	default:
		return uiTextAutocapitalizationTypeNone
	}
}

func (v textInputView) spellCheckingType() objc.NSInteger {
	if v.ivars.configuration.Suggestions {
		return uiTextSpellCheckingTypeYes
	}
	return uiTextSpellCheckingTypeNo
}

func (v textInputView) inlinePredictionType() objc.NSInteger {
	if v.ivars.configuration.Suggestions {
		return uiTextInlinePredictionTypeYes
	}
	return uiTextInlinePredictionTypeNo
}

func (v textInputView) returnKeyType() objc.NSInteger {
	switch v.ivars.configuration.InputAction {
	case gpui.TextInputActionDone:
		return uiReturnKeyTypeDone
	case gpui.TextInputActionGo:
		return uiReturnKeyTypeGo
	case gpui.TextInputActionNext:
		return uiReturnKeyTypeNext
	case gpui.TextInputActionSearch:
		return uiReturnKeyTypeSearch
	case gpui.TextInputActionSend:
		return uiReturnKeyTypeSend
	default:
		return uiReturnKeyTypeDefault
	}
}

func (v textInputView) textInRange(textRange objc.ID) objc.ID {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return objc.Nil
	}
	type textResult struct {
		text string
		ok   bool
	}
	result, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) textResult {
		text, ok := readText(handler, r)
		return textResult{text, ok}
	})
	if !ok || !result.ok {
		return objc.Nil
	}
	return objc.Autorelease(objc.NSStringFromGo(result.text))
}

func (v textInputView) replaceRange(textRange objc.ID, text objc.ID) {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return
	}
	s := objc.GoString(text)
	v.edit(func(handler *gpui.PlatformInputHandler) {
		if allowedRange(handler, r) {
			handler.ReplaceTextInRange(gpui.Some(r), s)
		}
	})
}

func (v textInputView) selectedTextRange() objc.ID {
	selection, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) gpui.Option[gpui.UTF16Selection] {
		selection, ok := handler.SelectedTextRange(false)
		if !ok || !allowedRange(handler, selection.Range) {
			return gpui.None[gpui.UTF16Selection]()
		}
		return gpui.Some(selection)
	})
	if !ok {
		return objc.Nil
	}
	value, ok := selection.Get()
	if !ok {
		return objc.Nil
	}
	return v.newRange(value.Range)
}

func (v textInputView) setSelectedTextRange(textRange objc.ID) {
	if textRange == objc.Nil {
		return
	}
	r, ok := v.validatedRange(textRange)
	if !ok {
		return
	}
	v.edit(func(handler *gpui.PlatformInputHandler) {
		if allowedRange(handler, r) {
			handler.SetSelectedTextRange(r)
		}
	})
}

func (v textInputView) markedTextRange() objc.ID {
	marked, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) gpui.Option[gpui.Range] {
		r, ok := handler.MarkedTextRange()
		if !ok || !allowedRange(handler, r) {
			return gpui.None[gpui.Range]()
		}
		return gpui.Some(r)
	})
	if !ok {
		return objc.Nil
	}
	r, ok := marked.Get()
	if !ok {
		return objc.Nil
	}
	return v.newRange(r)
}

func (v textInputView) markedTextStyle() objc.ID {
	if v.ivars.markedStyle == nil {
		return objc.Nil
	}
	return objc.Autorelease(objc.Retain(v.ivars.markedStyle.ID()))
}

func (v textInputView) setMarkedTextStyle(style objc.ID) {
	previous := v.ivars.markedStyle
	v.ivars.markedStyle = nil
	if style != objc.Nil {
		v.ivars.markedStyle = objc.Own(objc.Send[objc.ID](style, objc.Sel("copy")))
	}
	if previous != nil {
		previous.Release()
	}
}

func (v textInputView) setMarkedText(text objc.ID, selection objc.NSRange) {
	s := ""
	if text != objc.Nil {
		s = objc.GoString(text)
	}
	selected := gpui.None[gpui.Range]()
	if selection.Location <= math.MaxInt && selection.Length <= math.MaxInt {
		if r, ok := checkedRange(int(selection.Location), int(selection.Length), utf16Len(s)); ok {
			selected = gpui.Some(r)
		}
	}
	v.edit(func(handler *gpui.PlatformInputHandler) {
		handler.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), s, selected)
	})
}

func (v textInputView) unmarkText() {
	v.edit((*gpui.PlatformInputHandler).UnmarkText)
}

func (v textInputView) beginningOfDocument() objc.ID {
	r, ok := v.documentRange()
	if !ok {
		return v.newPosition(0)
	}
	return v.newPosition(r.Start)
}

func (v textInputView) endOfDocument() objc.ID {
	r, ok := v.documentRange()
	if !ok {
		return v.newPosition(0)
	}
	return v.newPosition(r.End)
}

func (v textInputView) textRangeFromPosition(start, end objc.ID) objc.ID {
	startOffset, ok := v.validatedOffset(start)
	if !ok {
		return objc.Nil
	}
	endOffset, ok := v.validatedOffset(end)
	if !ok {
		return objc.Nil
	}
	return v.newRange(gpui.Range{Start: min(startOffset, endOffset), End: max(startOffset, endOffset)})
}

func (v textInputView) positionFromPositionOffset(position objc.ID, offset objc.NSInteger) objc.ID {
	positionOffset, ok := v.validatedOffset(position)
	if !ok {
		return objc.Nil
	}
	r, ok := v.documentRange()
	if !ok {
		return objc.Nil
	}
	result, ok := checkedOffset(positionOffset, offset, r.End)
	if !ok || result < r.Start {
		return objc.Nil
	}
	return v.newPosition(result)
}

func (v textInputView) positionFromPositionInDirection(position objc.ID, direction objc.NSInteger, offset objc.NSInteger) objc.ID {
	if _, ok := v.validatedOffset(position); !ok {
		return objc.Nil
	}
	if direction == uiTextLayoutDirectionLeft {
		if offset == math.MinInt {
			return objc.Nil
		}
		return v.positionFromPositionOffset(position, -offset)
	}
	if direction == uiTextLayoutDirectionRight {
		return v.positionFromPositionOffset(position, offset)
	}
	rectangle := v.caretRectForPosition(position)
	sign := 1.0
	if direction == uiTextLayoutDirectionUp {
		sign = -1.0
	}
	return v.closestPositionToPoint(objc.CGPoint{
		X: rectangle.Origin.X,
		Y: rectangle.Origin.Y + rectangle.Size.Height*(0.5+sign*float64(offset)),
	})
}

func (v textInputView) comparePosition(position, other objc.ID) objc.NSInteger {
	a, ok := v.validatedOffset(position)
	if !ok {
		return nsOrderedSame
	}
	b, ok := v.validatedOffset(other)
	if !ok {
		return nsOrderedSame
	}
	switch {
	case a < b:
		return nsOrderedAscending
	case a > b:
		return nsOrderedDescending
	default:
		return nsOrderedSame
	}
}

func (v textInputView) offsetFromPosition(start, end objc.ID) objc.NSInteger {
	startOffset, ok := v.validatedOffset(start)
	if !ok {
		return 0
	}
	endOffset, ok := v.validatedOffset(end)
	if !ok {
		return 0
	}
	return endOffset - startOffset
}

func (v textInputView) loadDelegate() objc.ID {
	loadWeakFunctions()
	return objcLoadWeakRetained(v.ivars.delegate)
}

func (v textInputView) inputDelegate() objc.ID {
	delegate := v.loadDelegate()
	if delegate == objc.Nil {
		return objc.Nil
	}
	return objc.Autorelease(delegate)
}

func (v textInputView) setInputDelegate(delegate objc.ID) {
	loadWeakFunctions()
	objcStoreWeak(v.ivars.delegate, delegate)
}

func (v textInputView) tokenizer() objc.ID {
	tokenizer := objc.Send[objc.ID](objc.Alloc(objc.GetClass("UITextInputStringTokenizer")), objc.Sel("initWithTextInput:"), v.id)
	return objc.Autorelease(tokenizer)
}

func (v textInputView) positionWithinRangeFarthestInDirection(textRange objc.ID, direction objc.NSInteger) objc.ID {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return objc.Nil
	}
	offset := r.End
	if direction == uiTextLayoutDirectionLeft || direction == uiTextLayoutDirectionUp {
		offset = r.Start
	}
	return v.newPosition(offset)
}

func (v textInputView) characterRangeByExtendingPosition(position objc.ID, direction objc.NSInteger) objc.ID {
	offset, ok := v.validatedOffset(position)
	if !ok {
		return objc.Nil
	}
	backwards := direction == uiTextLayoutDirectionLeft || direction == uiTextLayoutDirectionUp
	if backwards {
		if offset == 0 {
			return objc.Nil
		}
		offset--
	}
	return v.characterRange(offset)
}

func (v textInputView) baseWritingDirection(position objc.ID) objc.NSInteger {
	if _, ok := v.validatedOffset(position); !ok {
		return nsWritingDirectionNatural
	}
	return nsWritingDirectionNatural
}

func (v textInputView) setBaseWritingDirection(direction objc.NSInteger, textRange objc.ID) {
	if _, ok := v.validatedRange(textRange); !ok {
		return
	}
	if direction != nsWritingDirectionNatural {
		slog.Warn("GPUI input handlers do not support overriding paragraph writing direction")
	}
}

func (v textInputView) firstRectForRange(textRange objc.ID) objc.CGRect {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return objc.CGRect{}
	}
	rect, ok := v.bounds(r)
	if !ok {
		return objc.CGRect{}
	}
	return rect
}

func (v textInputView) caretRectForPosition(position objc.ID) objc.CGRect {
	offset, ok := v.validatedOffset(position)
	if !ok {
		return objc.CGRect{}
	}
	rect, ok := v.bounds(gpui.Range{Start: offset, End: offset})
	if !ok {
		return objc.CGRect{}
	}
	return nonemptyCaretRect(rect)
}

func (v textInputView) selectionRectsForRange(textRange objc.ID) objc.ID {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return objc.Autorelease(objc.NSArrayFromIDs(nil))
	}
	bounds, _ := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) []gpui.Bounds[gpui.Pixels] {
		if !allowedRange(handler, r) {
			return nil
		}
		return handler.SelectionBoundsForRange(r)
	})
	count := len(bounds)
	rectangles := make([]objc.ID, 0, count)
	for index, b := range bounds {
		this := objc.Send[objc.ID](objc.Alloc(selectionRectClass()), objc.Sel("init"))
		textInputObjectsMu.Lock()
		selectionRects[this] = selectionRectIvars{
			rect:          nativeRect(b),
			containsStart: index == 0,
			containsEnd:   index+1 == count,
		}
		textInputObjectsMu.Unlock()
		rectangles = append(rectangles, this)
	}
	array := objc.NSArrayFromIDs(rectangles)
	for _, rectangle := range rectangles {
		objc.Release(rectangle)
	}
	return objc.Autorelease(array)
}

func (v textInputView) closestPositionToPoint(point objc.CGPoint) objc.ID {
	offset, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) gpui.Option[int] {
		offset, ok := handler.CharacterIndexForPoint(gpui.NewPoint(gpui.Px(float32(point.X)), gpui.Px(float32(point.Y))))
		if !ok || !allowedRange(handler, gpui.Range{Start: offset, End: offset}) {
			return gpui.None[int]()
		}
		return gpui.Some(offset)
	})
	if !ok {
		return objc.Nil
	}
	value, ok := offset.Get()
	if !ok {
		return objc.Nil
	}
	return v.newPosition(value)
}

func (v textInputView) closestPositionToPointWithinRange(point objc.CGPoint, textRange objc.ID) objc.ID {
	r, ok := v.validatedRange(textRange)
	if !ok {
		return objc.Nil
	}
	position := v.closestPositionToPoint(point)
	if position == objc.Nil {
		return objc.Nil
	}
	offset, ok := v.validatedOffset(position)
	if !ok {
		return objc.Nil
	}
	return v.newPosition(min(max(offset, r.Start), r.End))
}

func (v textInputView) characterRangeAtPoint(point objc.CGPoint) objc.ID {
	position := v.closestPositionToPoint(point)
	if position == objc.Nil {
		return objc.Nil
	}
	offset, ok := v.validatedOffset(position)
	if !ok {
		return objc.Nil
	}
	return v.characterRange(offset)
}

func (v textInputView) textInputView() objc.ID {
	if superview := objc.Send[objc.ID](v.id, objc.Sel("superview")); superview != objc.Nil {
		return superview
	}
	return v.id
}

func textInputSetWindow(this objc.ID, window *IosWindowState) {
	textInputIvarsFor(this).window.set(window)
}

func (v textInputView) validatedOffset(position objc.ID) (int, bool) {
	textInputObjectsMu.Lock()
	ivars, ok := textPositions[position]
	textInputObjectsMu.Unlock()
	if !ok || ivars.session != v.ivars.session {
		return 0, false
	}
	return ivars.offset, true
}

func (v textInputView) validatedRange(textRange objc.ID) (gpui.Range, bool) {
	textInputObjectsMu.Lock()
	ivars, ok := textRanges[textRange]
	textInputObjectsMu.Unlock()
	if !ok || ivars.session != v.ivars.session {
		return gpui.Range{}, false
	}
	return ivars.textRange, true
}

func (v textInputView) newPosition(offset int) objc.ID {
	return newTextPosition(offset, v.ivars.session)
}

func (v textInputView) newRange(r gpui.Range) objc.ID {
	return newTextRange(r, v.ivars.session)
}

func (v textInputView) invalidateSession() {
	v.ivars.session = &textInputSession{}
}

func textInputWithHandlerR[R any](v textInputView, callback func(handler *gpui.PlatformInputHandler) R) (R, bool) {
	type handlerResult struct {
		value R
		ok    bool
	}
	result, ok := windowReferenceWithR(&v.ivars.window, func(window *IosWindowState) handlerResult {
		value, ok := iosWindowStateWithInputHandlerR(window, callback)
		return handlerResult{value, ok}
	})
	if !ok || !result.ok {
		var zero R
		return zero, false
	}
	return result.value, true
}

func (v textInputView) edit(callback func(handler *gpui.PlatformInputHandler)) {
	previous := v.ivars.nativeEdit
	v.ivars.nativeEdit = true
	textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) struct{} {
		selected, selectedOK := handler.SelectedTextRange(false)
		marked, markedOK := handler.MarkedTextRange()
		selectionAllowed := selectedOK && allowedRange(handler, selected.Range)
		markedAllowed := !markedOK || allowedRange(handler, marked)
		if handler.QueryAcceptsFocusedTextInput() && selectionAllowed && markedAllowed {
			callback(handler)
		}
		return struct{}{}
	})
	v.ivars.nativeEdit = previous
}

func (v textInputView) documentRange() (gpui.Range, bool) {
	type rangeResult struct {
		r  gpui.Range
		ok bool
	}
	result, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) rangeResult {
		r, ok := documentRange(handler)
		return rangeResult{r, ok}
	})
	return result.r, ok && result.ok
}

func (v textInputView) bounds(r gpui.Range) (objc.CGRect, bool) {
	bounds, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) gpui.Option[gpui.Bounds[gpui.Pixels]] {
		if !allowedRange(handler, r) {
			return gpui.None[gpui.Bounds[gpui.Pixels]]()
		}
		return gpui.OptionFrom(handler.BoundsForRange(r))
	})
	if !ok {
		return objc.CGRect{}, false
	}
	value, ok := bounds.Get()
	if !ok {
		return objc.CGRect{}, false
	}
	return nativeRect(value), true
}

func (v textInputView) characterRange(offset int) objc.ID {
	documentRange, text, ok := v.documentText()
	if !ok {
		return objc.Nil
	}
	defer objc.Release(text)
	if offset < documentRange.Start {
		return objc.Nil
	}
	character, ok := composedCharacterRange(text, offset-documentRange.Start)
	if !ok {
		return objc.Nil
	}
	return v.newRange(gpui.Range{Start: documentRange.Start + character.Start, End: documentRange.Start + character.End})
}

func (v textInputView) documentText() (gpui.Range, objc.ID, bool) {
	type documentResult struct {
		r    gpui.Range
		text string
		ok   bool
	}
	result, ok := textInputWithHandlerR(v, func(handler *gpui.PlatformInputHandler) documentResult {
		r, ok := documentRange(handler)
		if !ok {
			return documentResult{}
		}
		text, ok := readText(handler, r)
		if !ok {
			return documentResult{}
		}
		return documentResult{r, text, true}
	})
	if !ok || !result.ok {
		return gpui.Range{}, objc.Nil, false
	}
	return result.r, objc.NSStringFromGo(result.text), true
}

func (v textInputView) setKeyboardVisible(visible bool) {
	v.ivars.keyboardDesired = visible
	v.refreshKeyboard()
}

func textInputSetKeyboardVisible(this objc.ID, visible bool) {
	textInputViewFor(this).setKeyboardVisible(visible)
}

func (v textInputView) setInputField(field gpui.Option[gpui.FocusID]) {
	previous := v.ivars.inputField
	v.ivars.inputField = field
	if previous != field {
		v.invalidateSession()
		if field.IsSome() {
			v.ivars.contentChanged = true
			v.ivars.selectionChanged = true
			v.setKeyboardVisible(true)
		}
	}
}

func textInputSetInputField(this objc.ID, field gpui.Option[gpui.FocusID]) {
	textInputViewFor(this).setInputField(field)
}

func (v textInputView) refreshKeyboard() {
	pending := v.ivars.updatePending
	v.ivars.updatePending = true
	if pending {
		return
	}
	objc.SendVoid(v.id, objc.Sel("performSelector:withObject:afterDelay:"), objc.Sel("reconcileTextInput"), objc.Nil, float64(0.0))
}

func textInputRefreshKeyboard(this objc.ID) {
	textInputViewFor(this).refreshKeyboard()
}

func (v textInputView) setConfiguration(configuration gpui.TextInputConfiguration) {
	if v.ivars.configuration == configuration {
		return
	}
	v.ivars.configuration = configuration
	v.ivars.configurationChanged = true
	v.refreshKeyboard()
}

func textInputSetConfiguration(this objc.ID, configuration gpui.TextInputConfiguration) {
	textInputViewFor(this).setConfiguration(configuration)
}

func (v textInputView) stateChanged(change gpui.TextInputStateChange) {
	switch change {
	case gpui.TextInputStateChangeFocusGained:
		v.ivars.contentChanged = true
		v.ivars.selectionChanged = true
		v.setKeyboardVisible(true)
	case gpui.TextInputStateChangeFocusLost:
		v.invalidateSession()
		v.ivars.inputField = gpui.None[gpui.FocusID]()
		v.setKeyboardVisible(false)
	case gpui.TextInputStateChangeSelectionChanged, gpui.TextInputStateChangeContentChanged:
		if v.ivars.nativeEdit {
			return
		}
		v.ivars.selectionChanged = true
		if change == gpui.TextInputStateChangeContentChanged {
			v.ivars.contentChanged = true
		}
		v.refreshKeyboard()
	}
}

func textInputStateChanged(this objc.ID, change gpui.TextInputStateChange) {
	textInputViewFor(this).stateChanged(change)
}

func nativeRect(bounds gpui.Bounds[gpui.Pixels]) objc.CGRect {
	return objc.CGRect{
		Origin: objc.CGPoint{X: float64(bounds.Origin.X), Y: float64(bounds.Origin.Y)},
		Size:   objc.CGSize{Width: float64(bounds.Size.Width), Height: float64(bounds.Size.Height)},
	}
}

func nonemptyCaretRect(rect objc.CGRect) objc.CGRect {
	rect.Size.Width = max(rect.Size.Width, 1.)
	return rect
}

func containsRange(outer, inner gpui.Range) bool {
	return outer.Start <= inner.Start && inner.Start <= inner.End && inner.End <= outer.End
}

func documentRange(handler *gpui.PlatformInputHandler) (gpui.Range, bool) {
	r, ok := handler.TextInputEditableRange()
	if !ok {
		length, ok := handler.TextLengthUTF16()
		if !ok {
			return gpui.Range{}, false
		}
		r = gpui.Range{Start: 0, End: length}
	}
	if r.Start > r.End {
		return gpui.Range{}, false
	}
	return r, true
}

func allowedRange(handler *gpui.PlatformInputHandler, r gpui.Range) bool {
	editable, ok := documentRange(handler)
	return ok && containsRange(editable, r)
}

func readText(handler *gpui.PlatformInputHandler, r gpui.Range) (string, bool) {
	if !allowedRange(handler, r) {
		return "", false
	}
	adjusted := gpui.None[gpui.Range]()
	text, ok := handler.TextForRange(r, &adjusted)
	if !ok {
		return "", false
	}
	if a, ok := adjusted.Get(); ok && a != r {
		return "", false
	}
	if utf16Len(text) != r.End-r.Start {
		return "", false
	}
	return text, true
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

func checkedRange(location, length, documentLength int) (gpui.Range, bool) {
	if location < 0 || length < 0 || location > math.MaxInt-length {
		return gpui.Range{}, false
	}
	end := location + length
	if end > documentLength {
		return gpui.Range{}, false
	}
	return gpui.Range{Start: location, End: end}, true
}

func composedCharacterRange(text objc.ID, offset int) (gpui.Range, bool) {
	length := objc.Send[objc.NSUInteger](text, objc.Sel("length"))
	if offset < 0 || uint(offset) >= length {
		return gpui.Range{}, false
	}
	r := objc.Send[objc.NSRange](text, objc.Sel("rangeOfComposedCharacterSequenceAtIndex:"), objc.NSUInteger(offset))
	if r.Location > math.MaxInt || r.Length > math.MaxInt || length > math.MaxInt {
		return gpui.Range{}, false
	}
	return checkedRange(int(r.Location), int(r.Length), int(length))
}

func checkedOffset(position int, offset int, documentLength int) (int, bool) {
	if position < 0 {
		return 0, false
	}
	if offset > 0 && position > math.MaxInt-offset {
		return 0, false
	}
	result := position + offset
	if result < 0 || result > documentLength {
		return 0, false
	}
	return result, true
}
