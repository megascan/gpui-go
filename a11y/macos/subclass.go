//go:build darwin && !ios

package macos

import (
	"fmt"
	"sync"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

type subclassEntry struct {
	prevClass objc.Class
	subclass  objc.Class
}

var (
	subclassesMu sync.Mutex
	subclasses   []subclassEntry
)

type associatedObjectState struct {
	adapter           *Adapter
	activationHandler a11y.ActivationHandler
}

type associatedObject struct {
	borrowed  bool
	state     associatedObjectState
	prevClass objc.Class
}

func (o *associatedObject) borrowMut() (*associatedObjectState, func()) {
	if o.borrowed {
		panic("already borrowed: BorrowMutError")
	}
	o.borrowed = true
	return &o.state, func() { o.borrowed = false }
}

var (
	associatedObjectsMu sync.Mutex
	associatedObjects   = map[objc.ID]*associatedObject{}
)

func getAssociatedObject(view objc.ID) *associatedObject {
	associatedObjectsMu.Lock()
	defer associatedObjectsMu.Unlock()
	return associatedObjects[view]
}

func setAssociatedObject(view objc.ID, object *associatedObject) {
	associatedObjectsMu.Lock()
	defer associatedObjectsMu.Unlock()
	if object == nil {
		delete(associatedObjects, view)
		return
	}
	associatedObjects[view] = object
}

func associatedObjectOf(view objc.ID) *associatedObject {
	associated := getAssociatedObject(view)
	if associated == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return associated
}

func subclassSuperclass(this objc.ID, _ objc.SEL) objc.Class {
	return associatedObjectOf(this).prevClass.SuperClass()
}

func subclassChildren(this objc.ID, _ objc.SEL) objc.ID {
	state, release := associatedObjectOf(this).borrowMut()
	defer release()
	return state.adapter.ViewChildren(state.activationHandler)
}

func subclassFocus(this objc.ID, _ objc.SEL) objc.ID {
	state, release := associatedObjectOf(this).borrowMut()
	defer release()
	return state.adapter.Focus(state.activationHandler)
}

func subclassHitTest(this objc.ID, _ objc.SEL, point objc.CGPoint) objc.ID {
	state, release := associatedObjectOf(this).borrowMut()
	defer release()
	return state.adapter.HitTest(point, state.activationHandler)
}

// SubclassingAdapter implements the NSView accessibility methods by dynamically subclassing a
// view, for when normal subclassing isn't an option. Release is its drop.
type SubclassingAdapter struct {
	view       *objc.Retained
	associated *associatedObject
	released   bool
}

// NewSubclassingAdapter creates an adapter that dynamically subclasses view. This must be done
// before the view is shown or focused for the first time. The action handler is always called on
// the main thread.
func NewSubclassingAdapter(view objc.ID, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) *SubclassingAdapter {
	if view == objc.Nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return newSubclassingAdapterInternal(objc.RetainObject(view), activationHandler, actionHandler)
}

func newSubclassingAdapterInternal(retainedView *objc.Retained, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) *SubclassingAdapter {
	mustLoadRuntime()
	view := retainedView.ID()
	if getAssociatedObject(view) != nil {
		panic(fmt.Sprintf("subclassing adapter already instantiated on view %#x", uintptr(view)))
	}
	adapter := NewAdapter(view, false, actionHandler)
	prevClass := view.Class()
	associated := &associatedObject{
		state:     associatedObjectState{adapter: adapter, activationHandler: activationHandler},
		prevClass: prevClass,
	}
	setAssociatedObject(view, associated)
	subclassesMu.Lock()
	var subclass objc.Class
	for _, entry := range subclasses {
		if entry.prevClass == prevClass {
			subclass = entry.subclass
			break
		}
	}
	if subclass == 0 {
		name := "AccessKitSubclassOf" + classGetName(prevClass)
		class, err := objc.RegisterClass(name, prevClass, nil, nil, []objc.MethodDef{
			{Cmd: objc.Sel("superclass"), Fn: subclassSuperclass},
			{Cmd: objc.Sel("accessibilityChildren"), Fn: subclassChildren},
			{Cmd: objc.Sel("accessibilityFocusedUIElement"), Fn: subclassFocus},
			{Cmd: objc.Sel("accessibilityHitTest:"), Fn: subclassHitTest},
		})
		if err != nil {
			subclassesMu.Unlock()
			panic(fmt.Sprintf("called `Option::unwrap()` on a `None` value: %v", err))
		}
		subclasses = append(subclasses, subclassEntry{prevClass: prevClass, subclass: class})
		subclass = class
	}
	subclassesMu.Unlock()
	objectSetClass(view, subclass)
	return &SubclassingAdapter{view: retainedView, associated: associated}
}

// SubclassingAdapterForWindow creates an adapter that dynamically subclasses the content view of
// window. It panics if the window has no content view.
func SubclassingAdapterForWindow(window objc.ID, activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler) *SubclassingAdapter {
	contentView := objc.Send[objc.ID](window, objc.Sel("contentView"))
	if contentView == objc.Nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return newSubclassingAdapterInternal(objc.RetainObject(contentView), activationHandler, actionHandler)
}

// UpdateIfActive calls updateFactory and applies the resulting update if and only if the tree has
// been initialized. When events are returned, the caller must call Raise on them.
func (s *SubclassingAdapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) (*QueuedEvents, bool) {
	state, release := s.associated.borrowMut()
	defer release()
	return state.adapter.UpdateIfActive(updateFactory)
}

// UpdateViewFocusState updates the tree state based on whether the window is focused. When events
// are returned, the caller must call Raise on them.
func (s *SubclassingAdapter) UpdateViewFocusState(isFocused bool) (*QueuedEvents, bool) {
	state, release := s.associated.borrowMut()
	defer release()
	return state.adapter.UpdateViewFocusState(isFocused)
}

// Release restores the view's original class, detaches the adapter and releases the view. It is
// idempotent.
func (s *SubclassingAdapter) Release() {
	if s.released {
		return
	}
	s.released = true
	prevClass := s.associated.prevClass
	view := s.view.ID()
	objectSetClass(view, prevClass)
	setAssociatedObject(view, nil)
	s.view.Release()
	s.associated.state.adapter.Release()
}
