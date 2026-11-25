//go:build darwin

// Package uikit binds the UIKit classes, enumerations and constants the iOS platform uses, as
// thin typed wrappers over internal/apple/objc on purego without cgo.
package uikit

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// UIKitPath is the install path of UIKit.framework.
const UIKitPath = "/System/Library/Frameworks/UIKit.framework/UIKit"

var (
	loadOnce sync.Once
	loadErr  error
	uikit    uintptr

	uiApplicationMain func(argc int32, argv **byte, principalClassName objc.ID, delegateClassName objc.ID) int32
)

// Load opens the Objective-C runtime and UIKit. It is safe to call repeatedly and from any
// thread.
func Load() error {
	loadOnce.Do(func() {
		if err := objc.Load(); err != nil {
			loadErr = err
			return
		}
		lib, err := purego.Dlopen(UIKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("uikit: open %s: %w", UIKitPath, err)
			return
		}
		uikit = lib
		purego.RegisterLibFunc(&uiApplicationMain, lib, "UIApplicationMain")
	})
	return loadErr
}

func mustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// Handle returns the dlopen handle of UIKit.framework.
func Handle() uintptr {
	mustLoad()
	return uikit
}

// Constant returns the object stored in UIKit's data symbol name, panicking when it is missing.
func Constant(name string) objc.ID {
	id, err := objc.Constant(Handle(), name)
	if err != nil {
		panic(err)
	}
	return id
}

func class(name string) objc.Class {
	mustLoad()
	return objc.GetClass(name)
}

func send[T any](id objc.ID, sel string, args ...any) T {
	mustLoad()
	return objc.Send[T](id, objc.Sel(sel), args...)
}

func sendVoid(id objc.ID, sel string, args ...any) {
	mustLoad()
	objc.SendVoid(id, objc.Sel(sel), args...)
}

func sendClass[T any](name string, sel string, args ...any) T {
	return send[T](objc.ID(class(name)), sel, args...)
}

// IsKindOfClass reports whether object is an instance of the class name or a subclass.
func IsKindOfClass(object objc.ID, name string) bool {
	if object == objc.Nil {
		return false
	}
	return send[bool](object, "isKindOfClass:", objc.ID(class(name)))
}

// UIApplicationMain runs UIKit's main loop with the process arguments, principalClassName
// (nil for UIApplication) and delegateClassName. It never returns.
func UIApplicationMain(principalClassName objc.ID, delegateClassName objc.ID) int32 {
	mustLoad()
	args := make([][]byte, len(os.Args))
	argv := make([]*byte, len(os.Args)+1)
	for i, arg := range os.Args {
		args[i] = append([]byte(arg), 0)
		argv[i] = unsafe.SliceData(args[i])
	}
	return uiApplicationMain(int32(len(os.Args)), unsafe.SliceData(argv), principalClassName, delegateClassName)
}

// UIScreenDidConnectNotification returns UIKit's UIScreenDidConnectNotification name.
func UIScreenDidConnectNotification() objc.ID { return Constant("UIScreenDidConnectNotification") }

// UIScreenDidDisconnectNotification returns UIKit's UIScreenDidDisconnectNotification name.
func UIScreenDidDisconnectNotification() objc.ID {
	return Constant("UIScreenDidDisconnectNotification")
}

// UIScreenModeDidChangeNotification returns UIKit's UIScreenModeDidChangeNotification name.
func UIScreenModeDidChangeNotification() objc.ID {
	return Constant("UIScreenModeDidChangeNotification")
}

// UITouchPhase is the phase of a UITouch.
type UITouchPhase objc.NSInteger

const (
	// UITouchPhaseBegan is a new touch.
	UITouchPhaseBegan UITouchPhase = 0
	// UITouchPhaseMoved is a touch that moved.
	UITouchPhaseMoved UITouchPhase = 1
	// UITouchPhaseStationary is a touch that did not move.
	UITouchPhaseStationary UITouchPhase = 2
	// UITouchPhaseEnded is a touch that lifted.
	UITouchPhaseEnded UITouchPhase = 3
	// UITouchPhaseCancelled is a touch the system cancelled.
	UITouchPhaseCancelled UITouchPhase = 4
	// UITouchPhaseRegionEntered is a pointer that entered the view.
	UITouchPhaseRegionEntered UITouchPhase = 5
	// UITouchPhaseRegionMoved is a pointer that moved inside the view.
	UITouchPhaseRegionMoved UITouchPhase = 6
	// UITouchPhaseRegionExited is a pointer that left the view.
	UITouchPhaseRegionExited UITouchPhase = 7
)

// UIGestureRecognizerState is the state of a UIGestureRecognizer.
type UIGestureRecognizerState objc.NSInteger

const (
	// UIGestureRecognizerStatePossible has not recognized its gesture yet.
	UIGestureRecognizerStatePossible UIGestureRecognizerState = 0
	// UIGestureRecognizerStateBegan has begun a continuous gesture.
	UIGestureRecognizerStateBegan UIGestureRecognizerState = 1
	// UIGestureRecognizerStateChanged has changed a continuous gesture.
	UIGestureRecognizerStateChanged UIGestureRecognizerState = 2
	// UIGestureRecognizerStateEnded has ended its gesture.
	UIGestureRecognizerStateEnded UIGestureRecognizerState = 3
	// UIGestureRecognizerStateCancelled has cancelled its gesture.
	UIGestureRecognizerStateCancelled UIGestureRecognizerState = 4
	// UIGestureRecognizerStateFailed failed to recognize its gesture.
	UIGestureRecognizerStateFailed UIGestureRecognizerState = 5
)

// UIRectEdge is a set of rectangle edges.
type UIRectEdge objc.NSUInteger

const (
	// UIRectEdgeNone is no edge.
	UIRectEdgeNone UIRectEdge = 0
	// UIRectEdgeTop is the top edge.
	UIRectEdgeTop UIRectEdge = 1 << 0
	// UIRectEdgeLeft is the left edge.
	UIRectEdgeLeft UIRectEdge = 1 << 1
	// UIRectEdgeBottom is the bottom edge.
	UIRectEdgeBottom UIRectEdge = 1 << 2
	// UIRectEdgeRight is the right edge.
	UIRectEdgeRight UIRectEdge = 1 << 3
	// UIRectEdgeAll is every edge.
	UIRectEdgeAll = UIRectEdgeTop | UIRectEdgeLeft | UIRectEdgeBottom | UIRectEdgeRight
)

// UIKeyModifierFlags is the set of modifier keys held during an event.
type UIKeyModifierFlags objc.NSInteger

const (
	// UIKeyModifierAlphaShift is caps lock.
	UIKeyModifierAlphaShift UIKeyModifierFlags = 1 << 16
	// UIKeyModifierShift is shift.
	UIKeyModifierShift UIKeyModifierFlags = 1 << 17
	// UIKeyModifierControl is control.
	UIKeyModifierControl UIKeyModifierFlags = 1 << 18
	// UIKeyModifierAlternate is option.
	UIKeyModifierAlternate UIKeyModifierFlags = 1 << 19
	// UIKeyModifierCommand is command.
	UIKeyModifierCommand UIKeyModifierFlags = 1 << 20
	// UIKeyModifierNumericPad is a numeric keypad key.
	UIKeyModifierNumericPad UIKeyModifierFlags = 1 << 21
)

// Contains reports whether every flag in other is set.
func (f UIKeyModifierFlags) Contains(other UIKeyModifierFlags) bool { return f&other == other }

// UIUserInterfaceStyle is light or dark appearance.
type UIUserInterfaceStyle objc.NSInteger

const (
	// UIUserInterfaceStyleUnspecified follows the parent.
	UIUserInterfaceStyleUnspecified UIUserInterfaceStyle = 0
	// UIUserInterfaceStyleLight is the light appearance.
	UIUserInterfaceStyleLight UIUserInterfaceStyle = 1
	// UIUserInterfaceStyleDark is the dark appearance.
	UIUserInterfaceStyleDark UIUserInterfaceStyle = 2
)
