//go:build darwin && !ios

package macos

import (
	"fmt"
	"strings"

	pobjc "github.com/ebitengine/purego/objc"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

func focusForwarder(this objc.ID, _ objc.SEL) objc.ID {
	view := objc.Send[objc.ID](this, objc.Sel("contentView"))
	if view == objc.Nil {
		return objc.Nil
	}
	return objc.Send[objc.ID](view, objc.Sel("accessibilityFocusedUIElement"))
}

// AddFocusForwarderToWindowClass adds to the named NSWindow subclass an
// accessibilityFocusedUIElement method that forwards to the window's content view, for windowing
// code that puts keyboard focus on the window instead of the content view. It cannot be undone.
func AddFocusForwarderToWindowClass(className string) {
	mustLoadRuntime()
	class := objc.GetClass(className)
	if class == 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	addMethod(class, "accessibilityFocusedUIElement", focusForwarder, 0, "@")
}

func addMethod(class objc.Class, sel string, fn any, argCount int, retEncoding string, argEncodings ...string) {
	selArgs := countArgs(sel)
	if selArgs != argCount {
		panic(fmt.Sprintf("assertion `left == right` failed: Selector %s accepts %d arguments, but function accepts %d\n  left: %d\n right: %d", sel, selArgs, argCount, selArgs, argCount))
	}
	types := methodTypeEncoding(retEncoding, argEncodings)
	if !class.AddMethod(objc.Sel(sel), pobjc.NewIMP(fn), types) {
		panic(fmt.Sprintf("Failed to add method %s", sel))
	}
}

func countArgs(sel string) int {
	return strings.Count(sel, ":")
}

func methodTypeEncoding(ret string, args []string) string {
	return ret + "@" + ":" + strings.Join(args, "")
}
