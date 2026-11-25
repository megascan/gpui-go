//go:build windows

package windows

import (
	"math"
	"runtime"
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/win32"
)

const simpleWindowTitle = "Simple test"

const (
	simpleWindowID a11y.NodeID = 0
	button1ID      a11y.NodeID = 1
	button2ID      a11y.NodeID = 2
)

func makeButton(label string) a11y.Node {
	node := a11y.NewNode(a11y.RoleButton)
	node.SetLabel(label)
	node.AddAction(a11y.ActionFocus)
	return node
}

func simpleInitialState() a11y.TreeUpdate {
	root := a11y.NewNode(a11y.RoleWindow)
	root.SetChildren([]a11y.NodeID{button1ID, button2ID})
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: simpleWindowID, Node: root},
			{ID: button1ID, Node: makeButton("Button 1")},
			{ID: button2ID, Node: makeButton("Button 2")},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(simpleWindowID)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  button1ID,
	}
}

type nullActionHandler struct{}

func (nullActionHandler) DoAction(a11y.ActionRequest) {}

type simpleActivationHandler struct {
	initial func() a11y.TreeUpdate
}

func (h simpleActivationHandler) RequestInitialTree() (a11y.TreeUpdate, bool) {
	return h.initial(), true
}

func simpleScope(t *testing.T, f func(*testScope) error) error {
	t.Helper()
	return scope(t, simpleWindowTitle, simpleActivationHandler{initial: simpleInitialState}, nullActionHandler{}, f)
}

func TestSimple_HasNativeUia(t *testing.T) {
	err := simpleScope(t, func(s *testScope) error {
		if !uiaHasServerSideProvider(s.window.HWND) {
			t.Fatal("assertion failed: has_native_uia")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isButtonNamed(element *com.IUnknown, expectedName string) bool {
	controlType, err := elementInt32(element, elementCurrentControlType)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	name, err := elementName(element)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	return controlType == uiaButtonControlTypeID && name == expectedName
}

func isButton1(element *com.IUnknown) bool {
	return isButtonNamed(element, "Button 1")
}

func isButton2(element *com.IUnknown) bool {
	return isButtonNamed(element, "Button 2")
}

func findChild(walker, start *com.IUnknown, startSlot, nextSlot int, match func(*com.IUnknown) bool) *com.IUnknown {
	child, err := walkerCall(walker, startSlot, start)
	for err == nil {
		if match(child) {
			return child
		}
		next, nextErr := walkerCall(walker, nextSlot, child)
		child.Release()
		child, err = next, nextErr
	}
	return nil
}

func TestSimple_Navigation(t *testing.T) {
	err := simpleScope(t, func(s *testScope) error {
		root, err := uiaElementFromHandleCall(s.uia, s.window.HWND)
		if err != nil {
			return err
		}
		defer root.Release()
		walker, err := uiaNoArgElement(s.uia, uiaControlViewWalker)
		if err != nil {
			return err
		}
		defer walker.Release()

		button1Forward := findChild(walker, root, walkerGetFirstChildElement, walkerGetNextSiblingElement, isButton1)
		if button1Forward == nil {
			t.Fatal("called `Option::unwrap()` on a `None` value")
		}
		defer button1Forward.Release()

		var button2Forward *com.IUnknown
		if child, err := walkerCall(walker, walkerGetNextSiblingElement, button1Forward); err == nil {
			if isButton2(child) {
				button2Forward = child
			} else {
				child.Release()
			}
		}
		if button2Forward == nil {
			t.Fatal("called `Option::unwrap()` on a `None` value")
		}
		defer button2Forward.Release()

		button2Backward := findChild(walker, root, walkerGetLastChildElement, walkerGetPreviousSiblingElement, isButton2)
		if button2Backward == nil {
			t.Fatal("called `Option::unwrap()` on a `None` value")
		}
		defer button2Backward.Release()

		var button1Backward *com.IUnknown
		if child, err := walkerCall(walker, walkerGetPreviousSiblingElement, button2Backward); err == nil {
			if isButton1(child) {
				button1Backward = child
			} else {
				child.Release()
			}
		}
		if button1Backward == nil {
			t.Fatal("called `Option::unwrap()` on a `None` value")
		}
		defer button1Backward.Release()

		equal, err := uiaCompareElementsCall(s.uia, button1Forward, button1Backward)
		if err != nil {
			return err
		}
		if !equal {
			t.Fatal("assertion failed: equal")
		}

		parent, err := walkerCall(walker, walkerGetParentElement, button1Forward)
		if err != nil {
			return err
		}
		defer parent.Release()
		equal, err = uiaCompareElementsCall(s.uia, parent, root)
		if err != nil {
			return err
		}
		if !equal {
			t.Fatal("assertion failed: equal")
		}

		desktopRoot, err := uiaNoArgElement(s.uia, uiaGetRootElement)
		if err != nil {
			return err
		}
		defer desktopRoot.Release()
		parent2, err := walkerCall(walker, walkerGetParentElement, root)
		if err != nil {
			return err
		}
		defer parent2.Release()
		equal, err = uiaCompareElementsCall(s.uia, parent2, desktopRoot)
		if err != nil {
			return err
		}
		if !equal {
			t.Fatal("assertion failed: equal")
		}

		if child, err := walkerCall(walker, walkerGetFirstChildElement, button1Forward); err != errEmpty {
			releaseAll(child)
			t.Fatalf("assertion `left == right` failed: left: Err(empty), right: %v", err)
		}
		if child, err := walkerCall(walker, walkerGetLastChildElement, button1Forward); err != errEmpty {
			releaseAll(child)
			t.Fatalf("assertion `left == right` failed: left: Err(empty), right: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSimple_Focus(t *testing.T) {
	err := simpleScope(t, func(s *testScope) error {
		handler, received := newFocusEventHandler()
		defer handler.Release()
		if err := com.CallHR(s.uia.Ptr(), uiaAddFocusChangedEventHandler, 0, uintptr(handler.Interface(0).Ptr())); err != nil {
			return err
		}

		s.showAndFocusWindow()
		focusFromEvent := received.wait(t, isButton1)
		defer focusFromEvent.Release()
		hasFocus, err := elementInt32(focusFromEvent, elementCurrentHasKeyboardFocus)
		if err != nil {
			return err
		}
		if hasFocus == 0 {
			t.Fatal("assertion failed: has_focus")
		}
		isFocusable, err := elementInt32(focusFromEvent, elementCurrentIsKeyboardFocusable)
		if err != nil {
			return err
		}
		if isFocusable == 0 {
			t.Fatal("assertion failed: is_focusable")
		}

		focusOnDemand, err := uiaNoArgElement(s.uia, uiaGetFocusedElement)
		if err != nil {
			return err
		}
		defer focusOnDemand.Release()
		equal, err := uiaCompareElementsCall(s.uia, focusFromEvent, focusOnDemand)
		if err != nil {
			return err
		}
		if !equal {
			t.Fatal("assertion failed: equal")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

const sliderID a11y.NodeID = 1

func sliderInitialState() a11y.TreeUpdate {
	root := a11y.NewNode(a11y.RoleWindow)
	root.SetChildren([]a11y.NodeID{sliderID})
	slider := a11y.NewNode(a11y.RoleSlider)
	slider.SetLabel("Slider")
	slider.SetBounds(a11y.Rect{X0: 0, Y0: 0, X1: 400, Y1: 200})
	slider.SetNumericValue(1)
	slider.SetMinNumericValue(0)
	slider.SetMaxNumericValue(10)
	slider.AddAction(a11y.ActionSetValue)
	slider.AddAction(a11y.ActionFocus)
	return a11y.TreeUpdate{
		Nodes: []a11y.NodeEntry{
			{ID: simpleWindowID, Node: root},
			{ID: sliderID, Node: slider},
		},
		Tree:   gpuiutil.Some(a11y.NewTree(simpleWindowID)),
		TreeID: a11y.TreeIDRoot(),
		Focus:  sliderID,
	}
}

type channelActionHandler struct {
	requests chan a11y.ActionRequest
}

func (h channelActionHandler) DoAction(request a11y.ActionRequest) {
	h.requests <- request
}

func TestSimple_ElementFromPointAndRangeValueSetValue(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("floating-point arguments through syscall.SyscallN are only passed in registers on amd64; arm64 is an owner check")
	}
	handler := channelActionHandler{requests: make(chan a11y.ActionRequest, 4)}
	err := scope(t, simpleWindowTitle, simpleActivationHandler{initial: sliderInitialState}, handler, func(s *testScope) error {
		s.showAndFocusWindow()
		topLeft := clientTopLeft(s.window)
		element, err := uiaElementFromPointCall(s.uia, win32.POINT{X: int32(topLeft.X) + 200, Y: int32(topLeft.Y) + 100})
		if err != nil {
			return err
		}
		defer element.Release()
		controlType, err := elementInt32(element, elementCurrentControlType)
		if err != nil {
			return err
		}
		if controlType != uiaSliderControlTypeID {
			t.Fatalf("ElementFromPoint returned control type %d, want slider", controlType)
		}

		var pattern *com.IUnknown
		if err := com.CallHR(element.Ptr(), elementGetCurrentPatternAs, uintptr(uiaRangeValuePatternID), uintptr(unsafe.Pointer(&iidIUIAutomationRangeValuePattern)), uintptr(unsafe.Pointer(&pattern))); err != nil {
			return err
		}
		if pattern == nil {
			t.Fatal("RangeValue pattern missing")
		}
		defer pattern.Release()
		if err := com.CallHR(pattern.Ptr(), rangeValueSetValue, uintptr(math.Float64bits(7.5))); err != nil {
			return err
		}
		for {
			select {
			case request := <-handler.requests:
				// Real UIA calls the provider's SetFocus on the slider before SetValue; Wine does not.
				if request.Action == a11y.ActionFocus && request.TargetNode == sliderID {
					continue
				}
				if request.Action != a11y.ActionSetValue || request.Data != a11y.ActionData(a11y.ActionDataNumericValue(7.5)) {
					t.Fatalf("got request %+v", request)
				}
				return nil
			default:
				t.Fatal("SetValue sent no action request")
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}
