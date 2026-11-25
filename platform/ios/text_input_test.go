//go:build ios

package ios

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func TestTextInput_InsertionGeometryHasAVisibleCaretWidth(t *testing.T) {
	rect := nonemptyCaretRect(objc.CGRect{Origin: objc.CGPoint{X: 12., Y: 20.}, Size: objc.CGSize{Width: 0., Height: 30.}})
	if rect.Origin != (objc.CGPoint{X: 12., Y: 20.}) {
		t.Fatalf("origin = %v", rect.Origin)
	}
	if rect.Size != (objc.CGSize{Width: 1., Height: 30.}) {
		t.Fatalf("size = %v", rect.Size)
	}
}

func TestTextInput_UTF16RangesAndOffsets(t *testing.T) {
	r := func(start, end int) gpui.Range { return gpui.Range{Start: start, End: end} }
	if !containsRange(r(4, 8), r(4, 4)) || !containsRange(r(4, 8), r(8, 8)) {
		t.Fatal("contains_range rejected an edge range")
	}
	if containsRange(r(4, 8), r(3, 5)) || containsRange(r(4, 8), r(7, 9)) {
		t.Fatal("contains_range accepted an overlapping range")
	}
	if containsRange(r(4, 8), r(6, 5)) {
		t.Fatal("contains_range accepted a reversed range")
	}
	length := utf16Len("a😀é")
	if length != 5 {
		t.Fatalf("utf16 length = %d, want 5", length)
	}
	check := func(got gpui.Range, ok bool, want gpui.Range, wantOK bool) {
		t.Helper()
		if ok != wantOK || (ok && got != want) {
			t.Fatalf("got %v, %t; want %v, %t", got, ok, want, wantOK)
		}
	}
	got, ok := checkedRange(1, 2, length)
	check(got, ok, r(1, 3), true)
	got, ok = checkedRange(length, 0, length)
	check(got, ok, r(length, length), true)
	got, ok = checkedRange(math.MaxInt, 1, length)
	check(got, ok, gpui.Range{}, false)
	got, ok = checkedRange(4, 2, length)
	check(got, ok, gpui.Range{}, false)
	offset := func(got int, ok bool, want int, wantOK bool) {
		t.Helper()
		if ok != wantOK || (ok && got != want) {
			t.Fatalf("got %d, %t; want %d, %t", got, ok, want, wantOK)
		}
	}
	o, ok := checkedOffset(3, -2, length)
	offset(o, ok, 1, true)
	o, ok = checkedOffset(0, -1, length)
	offset(o, ok, 0, false)
	o, ok = checkedOffset(length, 1, length)
	offset(o, ok, 0, false)
	o, ok = checkedOffset(0, math.MinInt, length)
	offset(o, ok, 0, false)
}

var (
	testDelegateMu     sync.Mutex
	testDelegateEvents = map[objc.ID][]string{}
	testClassesOnce    sync.Once
	competingClass     objc.Class
	inputDelegateClass objc.Class
)

func registerTextInputTestClasses() {
	testClassesOnce.Do(func() {
		var err error
		competingClass, err = objc.RegisterClass("GPUICompetingResponderTestView", objc.GetClass("UIView"), []string{"NSObject"}, nil, []objc.MethodDef{
			{Cmd: objc.Sel("canBecomeFirstResponder"), Fn: func(_ objc.ID, _ objc.SEL) bool { return true }},
		})
		if err != nil {
			panic(err)
		}
		push := func(event string) func(objc.ID, objc.SEL, objc.ID) {
			return func(this objc.ID, _ objc.SEL, _ objc.ID) {
				testDelegateMu.Lock()
				testDelegateEvents[this] = append(testDelegateEvents[this], event)
				testDelegateMu.Unlock()
			}
		}
		inputDelegateClass, err = objc.RegisterClass("GPUITextInputTestDelegate", objc.GetClass("NSObject"), []string{"NSObject", "UITextInputDelegate"}, nil, []objc.MethodDef{
			{Cmd: objc.Sel("selectionWillChange:"), Fn: push("selection-will")},
			{Cmd: objc.Sel("selectionDidChange:"), Fn: push("selection-did")},
			{Cmd: objc.Sel("textWillChange:"), Fn: push("text-will")},
			{Cmd: objc.Sel("textDidChange:"), Fn: push("text-did")},
			{Cmd: objc.Sel("conversationContext:didChange:"), Fn: func(this objc.ID, _ objc.SEL, _ objc.ID, _ objc.ID) {
				testDelegateMu.Lock()
				testDelegateEvents[this] = append(testDelegateEvents[this], "conversation-did")
				testDelegateMu.Unlock()
			}},
		})
		if err != nil {
			panic(err)
		}
	})
}

func delegateEvents(delegate objc.ID) []string {
	testDelegateMu.Lock()
	defer testDelegateMu.Unlock()
	return slices.Clone(testDelegateEvents[delegate])
}

func clearDelegateEvents(delegate objc.ID) {
	testDelegateMu.Lock()
	defer testDelegateMu.Unlock()
	delete(testDelegateEvents, delegate)
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

var fullNotification = []string{"text-will", "selection-will", "selection-did", "text-did"}

func start(r objc.ID) objc.ID { return objc.Send[objc.ID](r, objc.Sel("start")) }
func end(r objc.ID) objc.ID   { return objc.Send[objc.ID](r, objc.Sel("end")) }

func nativeProtocolSmokeTest() {
	registerTextInputTestClasses()
	staleNativeObjectsTest()
	viewID := newTextInputView(objc.CGRect{})
	defer objc.Release(viewID)
	view := textInputViewFor(viewID)
	position := view.newPosition(3)
	o, ok := view.validatedOffset(position)
	assertf(ok && o == 3, "validated_offset(position) = %d, %t", o, ok)
	r := view.newRange(gpui.Range{Start: 1, End: 3})
	o, ok = view.validatedOffset(start(r))
	assertf(ok && o == 1, "range start = %d, %t", o, ok)
	o, ok = view.validatedOffset(end(r))
	assertf(ok && o == 3, "range end = %d, %t", o, ok)
	assertf(!objc.Send[bool](r, objc.Sel("isEmpty")), "1..3 is empty")
	assertf(objc.Send[bool](view.newRange(gpui.Range{Start: 3, End: 3}), objc.Sel("isEmpty")), "3..3 is not empty")

	text := objc.Autorelease(objc.NSStringFromGo("a😀é👨‍👩‍👧‍👦"))
	for _, c := range []struct {
		offset int
		want   gpui.Range
		ok     bool
	}{{1, gpui.Range{Start: 1, End: 3}, true}, {2, gpui.Range{Start: 1, End: 3}, true}, {4, gpui.Range{Start: 3, End: 5}, true}, {6, gpui.Range{Start: 5, End: 16}, true}, {16, gpui.Range{}, false}} {
		got, ok := composedCharacterRange(text, c.offset)
		assertf(ok == c.ok && (!ok || got == c.want), "composed_character_range(%d) = %v, %t", c.offset, got, ok)
	}

	textInputSetWindow(viewID, nil)
	canBecome := func() bool { return objc.Send[bool](viewID, objc.Sel("canBecomeFirstResponder")) }
	assertf(!canBecome(), "can become first responder without a window")
	view.setKeyboardVisible(true)
	view.reconcileTextInput()
	assertf(!canBecome(), "can become first responder without a handler")
	assertf(!objc.Send[bool](viewID, objc.Sel("hasText")), "has text without a handler")
	assertf(objc.Send[objc.ID](viewID, objc.Sel("selectedTextRange")) == objc.Nil, "selected range without a handler")
	assertf(objc.Send[objc.ID](viewID, objc.Sel("markedTextRange")) == objc.Nil, "marked range without a handler")
	o, ok = view.validatedOffset(objc.Send[objc.ID](viewID, objc.Sel("beginningOfDocument")))
	assertf(ok && o == 0, "beginning = %d, %t", o, ok)
	o, ok = view.validatedOffset(objc.Send[objc.ID](viewID, objc.Sel("endOfDocument")))
	assertf(ok && o == 0, "end = %d, %t", o, ok)
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("offsetFromPosition:toPosition:"), start(r), end(r)) == 2, "offset != 2")
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("comparePosition:toPosition:"), start(r), end(r)) == nsOrderedAscending, "compare != ascending")
	assertf(objc.Send[objc.ID](viewID, objc.Sel("textRangeFromPosition:toPosition:"), end(r), start(r)) != objc.Nil, "reversed text range is nil")
	objc.Send[objc.ID](viewID, objc.Sel("tokenizer"))
	delegate := objc.AllocInit(inputDelegateClass)
	objc.SendVoid(viewID, objc.Sel("setInputDelegate:"), delegate)
	view.stateChanged(gpui.TextInputStateChangeContentChanged)
	view.stateChanged(gpui.TextInputStateChangeSelectionChanged)
	assertf(len(delegateEvents(delegate)) == 0, "delegate notified before reconciliation")
	objc.AutoreleasePool(func() { view.reconcileTextInput() })
	assertf(slices.Equal(delegateEvents(delegate), fullNotification), "events = %v", delegateEvents(delegate))

	objc.AutoreleasePool(func() {
		firstField := gpui.Some(gpui.FocusID(1))
		secondField := gpui.Some(gpui.FocusID(2))
		clearDelegateEvents(delegate)
		view.setInputField(firstField)
		view.reconcileTextInput()
		assertf(slices.Equal(delegateEvents(delegate), fullNotification), "events = %v", delegateEvents(delegate))

		view.setKeyboardVisible(false)
		view.reconcileTextInput()
		assertf(view.ivars.inputField == firstField, "input field changed")
		clearDelegateEvents(delegate)
		view.setInputField(firstField)
		view.refreshKeyboard()
		view.reconcileTextInput()
		assertf(!view.ivars.keyboardDesired, "keyboard desired after hide")
		assertf(len(delegateEvents(delegate)) == 0, "events = %v", delegateEvents(delegate))

		view.setKeyboardVisible(true)
		assertf(view.ivars.inputField == firstField, "input field changed")
		assertf(view.ivars.keyboardDesired, "keyboard not desired")

		view.setInputField(secondField)
		assertf(view.ivars.keyboardDesired, "keyboard not desired")
		view.reconcileTextInput()
		assertf(slices.Equal(delegateEvents(delegate), fullNotification), "events = %v", delegateEvents(delegate))
		view.stateChanged(gpui.TextInputStateChangeFocusLost)
		assertf(view.ivars.inputField.IsNone(), "input field kept after focus loss")
		assertf(!view.ivars.keyboardDesired, "keyboard desired after focus loss")
	})

	objc.Release(delegate)
	clearDelegateEvents(delegate)
	objc.AutoreleasePool(func() {
		assertf(objc.Send[objc.ID](viewID, objc.Sel("inputDelegate")) == objc.Nil, "delegate outlived its release")
	})
	objc.SendVoid(viewID, objc.Sel("setMarkedText:selectedRange:"), text, objc.MakeNSRange(1, 2))
	objc.SendVoid(viewID, objc.Sel("unmarkText"))
	view.setConfiguration(gpui.TextInputConfiguration{
		Autocorrect:    true,
		Autocapitalize: gpui.AutocapitalizeSentences,
		Suggestions:    true,
		InputAction:    gpui.TextInputActionSearch,
	})
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("autocorrectionType")) == uiTextAutocorrectionTypeYes, "autocorrection")
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("autocapitalizationType")) == uiTextAutocapitalizationTypeSentences, "autocapitalization")
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("spellCheckingType")) == uiTextSpellCheckingTypeYes, "spell checking")
	assertf(objc.Send[objc.NSInteger](viewID, objc.Sel("returnKeyType")) == uiReturnKeyTypeSearch, "return key")
	view.setKeyboardVisible(false)
	view.reconcileTextInput()
	assertf(!canBecome(), "can become first responder after hide")

	window := objc.Send[objc.ID](objc.Alloc(objc.GetClass("UIWindow")), objc.Sel("initWithFrame:"), objc.MakeCGRect(0, 0, 320., 480.))
	defer objc.Release(window)
	controller := objc.AllocInit(objc.GetClass("UIViewController"))
	defer objc.Release(controller)
	objc.SendVoid(controller, objc.Sel("setView:"), viewID)
	objc.SendVoid(window, objc.Sel("setRootViewController:"), controller)
	objc.SendVoid(window, objc.Sel("makeKeyAndVisible"))
	become := func(id objc.ID) bool { return objc.Send[bool](id, objc.Sel("becomeFirstResponder")) }
	resign := func(id objc.ID) bool { return objc.Send[bool](id, objc.Sel("resignFirstResponder")) }
	isFirst := func(id objc.ID) bool { return objc.Send[bool](id, objc.Sel("isFirstResponder")) }
	view.setKeyboardVisible(true)
	view.ivars.keyboardEligible = true
	assertf(become(viewID), "become first responder failed")
	assertf(isFirst(viewID), "not first responder")
	assertf(resign(viewID), "resign failed")
	assertf(!isFirst(viewID), "still first responder")
	assertf(!view.ivars.keyboardDesired, "native resignation must cancel the outstanding keyboard request")

	view.refreshKeyboard()
	view.reconcileTextInput()
	assertf(!view.ivars.keyboardDesired, "keyboard desired after reconcile")
	assertf(!isFirst(viewID), "first responder after reconcile")

	view.setKeyboardVisible(true)
	view.ivars.keyboardEligible = true
	assertf(become(viewID), "become first responder failed")
	other := objc.Send[objc.ID](objc.Alloc(competingClass), objc.Sel("initWithFrame:"), objc.CGRect{})
	defer objc.Release(other)
	objc.SendVoid(viewID, objc.Sel("addSubview:"), other)
	assertf(become(other), "competing responder could not become first responder")
	assertf(!isFirst(viewID), "view still first responder")
	assertf(!view.ivars.keyboardDesired, "keyboard desired after transfer")
	view.refreshKeyboard()
	view.reconcileTextInput()
	assertf(isFirst(other), "reconcile reclaimed first responder")

	view.setKeyboardVisible(true)
	view.ivars.keyboardEligible = true
	assertf(become(viewID), "become first responder failed")
	view.reconcileTextInput()
	assertf(!isFirst(viewID), "first responder without an eligible handler")
	assertf(view.ivars.keyboardDesired, "pending keyboard request discarded")
	resign(viewID)
	assertf(view.ivars.keyboardDesired, "resigning an inactive view cancelled a queued request")
	objc.SendVoid(window, objc.Sel("setHidden:"), true)
}

func staleNativeObjectsTest() {
	viewID := newTextInputView(objc.CGRect{})
	defer objc.Release(viewID)
	view := textInputViewFor(viewID)
	first := gpui.Some(gpui.FocusID(1))
	second := gpui.Some(gpui.FocusID(2))
	view.setInputField(first)
	oldStart := view.beginningOfDocument()
	oldRange := view.newRange(gpui.Range{Start: 1, End: 3})
	got, ok := view.validatedRange(oldRange)
	assertf(ok && got == gpui.Range{Start: 1, End: 3}, "validated_range = %v, %t", got, ok)
	assertf(containsRange(gpui.Range{Start: 0, End: 4}, gpui.Range{Start: 1, End: 3}), "contains_range")

	view.setInputField(second)
	_, ok = view.validatedRange(oldRange)
	assertf(!ok, "A range must be rejected immediately on switching to equally-sized B")
	assertf(view.textRangeFromPosition(oldStart, view.endOfDocument()) == objc.Nil, "the native protocol must reject a stale position before reconciliation")
	freshStart := view.beginningOfDocument()
	freshRange := view.newRange(gpui.Range{Start: 1, End: 3})
	o, ok := view.validatedOffset(freshStart)
	assertf(ok && o == 0, "fresh start = %d, %t", o, ok)
	got, ok = view.validatedRange(freshRange)
	assertf(ok && got == gpui.Range{Start: 1, End: 3}, "fresh range = %v, %t", got, ok)
	_, ok = view.validatedOffset(start(oldRange))
	assertf(!ok, "stale range start validated")
	_, ok = view.validatedOffset(end(oldRange))
	assertf(!ok, "stale range end validated")
	assertf(view.comparePosition(start(oldRange), freshStart) == nsOrderedSame, "stale compare")
	assertf(view.offsetFromPosition(start(oldRange), freshStart) == 0, "stale offset")
	assertf(view.positionWithinRangeFarthestInDirection(oldRange, uiTextLayoutDirectionRight) == objc.Nil, "stale farthest")
	assertf(view.positionWithinRangeFarthestInDirection(freshRange, uiTextLayoutDirectionRight) != objc.Nil, "fresh farthest")
	assertf(view.positionFromPositionInDirection(oldStart, uiTextLayoutDirectionDown, 1) == objc.Nil, "stale direction")
	assertf(view.textRangeFromPosition(freshStart, view.endOfDocument()) != objc.Nil, "fresh text range")
	view.setInputField(first)
	_, ok = view.validatedRange(oldRange)
	assertf(!ok, "A→B→A must not revive A objects")
	_, ok = view.validatedOffset(oldStart)
	assertf(!ok, "A→B→A revived a position")
	returnedStart := view.beginningOfDocument()
	returnedRange := view.newRange(gpui.Range{Start: 1, End: 3})
	view.setInputField(first)
	view.stateChanged(gpui.TextInputStateChangeContentChanged)
	o, ok = view.validatedOffset(returnedStart)
	assertf(ok && o == 0, "returned start = %d, %t", o, ok)
	got, ok = view.validatedRange(returnedRange)
	assertf(ok && got == gpui.Range{Start: 1, End: 3}, "returned range = %v, %t", got, ok)
	view.stateChanged(gpui.TextInputStateChangeFocusLost)
	_, ok = view.validatedOffset(returnedStart)
	assertf(!ok, "position survived focus loss")
	_, ok = view.validatedRange(returnedRange)
	assertf(!ok, "range survived focus loss")
	foreignID := newTextInputView(objc.CGRect{})
	defer objc.Release(foreignID)
	foreign := textInputViewFor(foreignID)
	_, ok = foreign.validatedOffset(view.beginningOfDocument())
	assertf(!ok, "foreign position validated")
	_, ok = foreign.validatedRange(view.newRange(gpui.Range{Start: 1, End: 3}))
	assertf(!ok, "foreign range validated")
	assertf(foreign.textRangeFromPosition(view.beginningOfDocument(), foreign.endOfDocument()) == objc.Nil, "foreign text range")
}

func staleObjectsDoNotRouteToHandler(handlerQueries func() int) {
	viewID := newTextInputView(objc.CGRect{})
	defer objc.Release(viewID)
	view := textInputViewFor(viewID)
	view.setInputField(gpui.Some(gpui.FocusID(1)))
	oldRange := view.newRange(gpui.Range{Start: 1, End: 3})
	oldPosition := start(oldRange)
	view.setInputField(gpui.Some(gpui.FocusID(2)))
	queriesBefore := handlerQueries()
	replacement := objc.Autorelease(objc.NSStringFromGo("x"))
	objc.SendVoid(viewID, objc.Sel("replaceRange:withText:"), oldRange, replacement)
	objc.SendVoid(viewID, objc.Sel("setSelectedTextRange:"), oldRange)
	assertf(objc.Send[objc.ID](viewID, objc.Sel("textInRange:"), oldRange) == objc.Nil, "stale textInRange")
	assertf(view.firstRectForRange(oldRange) == objc.CGRect{}, "stale firstRect")
	assertf(view.caretRectForPosition(oldPosition) == objc.CGRect{}, "stale caretRect")
	assertf(len(objc.NSArrayToIDs(view.selectionRectsForRange(oldRange))) == 0, "stale selection rects")
	assertf(view.positionFromPositionOffset(oldPosition, 1) == objc.Nil, "stale position offset")
	assertf(view.characterRangeByExtendingPosition(oldPosition, uiTextLayoutDirectionRight) == objc.Nil, "stale extending range")
	assertf(view.closestPositionToPointWithinRange(objc.CGPoint{}, oldRange) == objc.Nil, "stale closest position")
	assertf(handlerQueries() == queriesBefore, "stale mutations and reads must be rejected before resolving B's handler")

	freshRange := view.newRange(gpui.Range{Start: 1, End: 3})
	objc.SendVoid(viewID, objc.Sel("replaceRange:withText:"), freshRange, replacement)
	assertf(handlerQueries() == queriesBefore+1, "fresh replace did not reach the handler boundary")
	objc.SendVoid(viewID, objc.Sel("setSelectedTextRange:"), freshRange)
	assertf(handlerQueries() == queriesBefore+2, "fresh selection did not reach the handler boundary")
}
