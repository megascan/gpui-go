//go:build darwin && !ios

// Command native_view_appkit_field shows that an AppKit control outside GPUI keeps keyboard input
// beside a native view.
//
// The window holds a WKWebView and, in its top right corner, an NSTextView that GPUI knows
// nothing about. Click that field and type: the text goes into it, and the line at the top says
// which of the three has keyboard input.
//
// A second after it opens, the example also gives the field keyboard input itself and prints
// whether the field still had it half a second later.
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform"
)

const webKitPath = "/System/Library/Frameworks/WebKit.framework/WebKit"

var loadWebKit = sync.OnceValue(func() error {
	if _, err := purego.Dlopen(webKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return fmt.Errorf("loading WebKit: %w", err)
	}
	return nil
})

const page = `<!doctype html>
<html>
  <body style="font: 15px -apple-system; margin: 24px">
    <p><input placeholder="A field in the page" size="36" autofocus /></p>
  </body>
</html>`

// KeyboardInput is the view that has keyboard input.
type KeyboardInput uint8

const (
	// KeyboardInputAppKitField is the AppKit text view.
	KeyboardInputAppKitField KeyboardInput = iota
	// KeyboardInputPage is the web page.
	KeyboardInputPage
	// KeyboardInputGpui is GPUI's own content.
	KeyboardInputGpui
)

// Example holds a web view and an AppKit field GPUI does not know about.
type Example struct {
	webView       gpui.NativeView
	webViewObject *objc.Retained
	appkitField   *objc.Retained
	keyboardInput KeyboardInput
	watch         *gpui.Task[struct{}]
}

// NewExample creates the views and the task that watches keyboard input.
func NewExample(window *gpui.Window, cx *gpui.Context[Example]) *Example {
	if err := loadWebKit(); err != nil {
		panic(err)
	}
	webViewObject := objc.Own(objc.AllocInit(objc.GetClass("WKWebView")))
	objc.Send[objc.ID](
		webViewObject.ID(),
		objc.Sel("loadHTMLString:baseURL:"),
		nsString(page),
		objc.ID(0),
	)
	frame := objc.MakeCGRect(0., 0., 220., 22.)
	appkitField := objc.Own(objc.Send[objc.ID](objc.Alloc(objc.GetClass("NSTextView")), objc.Sel("initWithFrame:"), frame))
	objc.SendVoid(appkitField.ID(), objc.Sel("setString:"), nsString("An AppKit field"))

	webView, err := gpui.NewNativeView(uintptr(webViewObject.ID()), window, cx.App)
	if err != nil {
		panic(fmt.Sprintf("this example only runs where native views are composited: %v", err))
	}

	nsWindow := objc.Send[objc.ID](webViewObject.ID(), objc.Sel("window"))
	contentView := objc.Send[objc.ID](nsWindow, objc.Sel("contentView"))
	bounds := objc.Send[objc.CGRect](contentView, objc.Sel("bounds"))
	isFlipped := objc.Send[bool](contentView, objc.Sel("isFlipped"))
	size := objc.MakeCGSize(220., 22.)
	top := bounds.Size.Height - size.Height - 8.
	if isFlipped {
		top = 8.
	}
	origin := objc.MakeCGPoint(bounds.Size.Width-size.Width-8., top)
	objc.SendVoid(appkitField.ID(), objc.Sel("setFrame:"), objc.CGRect{Origin: origin, Size: size})
	mask := objc.NSUInteger(1 | 8)
	if isFlipped {
		mask = 1 | 32
	}
	objc.SendVoid(appkitField.ID(), objc.Sel("setAutoresizingMask:"), mask)
	objc.SendVoid(contentView, objc.Sel("addSubview:"), appkitField.ID())

	webViewFocus := webView.FocusHandle(cx.App)
	window.Focus(webViewFocus, cx.App)
	webViewFocus.Release()

	watch := cx.Spawn[struct{}](func(this gpui.WeakEntity[Example], cx *gpui.AsyncApp) (struct{}, error) {
		elapsed := time.Duration(0)
		step := 250 * time.Millisecond
		for {
			if _, err := cx.BackgroundExecutor().Timer(step).Await(cx); err != nil {
				return struct{}{}, err
			}
			elapsed += step
			err := this.Update(cx, func(this *Example, cx *gpui.Context[Example]) {
				if elapsed == 1000*time.Millisecond {
					this.focusAppkitField()
				}
				keyboardInput := this.currentKeyboardInput()
				if elapsed == 1500*time.Millisecond {
					fmt.Printf("AppKit field kept keyboard input: %t\n", keyboardInput == KeyboardInputAppKitField)
				}
				if this.keyboardInput != keyboardInput {
					this.keyboardInput = keyboardInput
					cx.Notify()
				}
			})
			if err != nil {
				return struct{}{}, nil
			}
		}
	})

	return &Example{
		webView:       webView,
		webViewObject: webViewObject,
		appkitField:   appkitField,
		keyboardInput: KeyboardInputGpui,
		watch:         watch,
	}
}

func (e *Example) focusAppkitField() {
	nsWindow := objc.Send[objc.ID](e.appkitField.ID(), objc.Sel("window"))
	accepted := objc.Send[bool](nsWindow, objc.Sel("makeFirstResponder:"), e.appkitField.ID())
	fmt.Printf("AppKit field took keyboard input: %t\n", accepted)
}

func (e *Example) currentKeyboardInput() KeyboardInput {
	nsWindow := objc.Send[objc.ID](e.appkitField.ID(), objc.Sel("window"))
	responder := objc.Send[objc.ID](nsWindow, objc.Sel("firstResponder"))
	isView := responder != 0 && objc.Send[bool](responder, objc.Sel("isKindOfClass:"), objc.GetClass("NSView"))
	inside := func(view objc.ID) bool {
		return isView && objc.Send[bool](responder, objc.Sel("isDescendantOf:"), view)
	}
	switch {
	case inside(e.appkitField.ID()):
		return KeyboardInputAppKitField
	case inside(e.webViewObject.ID()):
		return KeyboardInputPage
	default:
		return KeyboardInputGpui
	}
}

// Render renders the example.
func (e *Example) Render(_ *gpui.Window, _ *gpui.Context[Example]) gpui.IntoElement {
	var keyboardInput string
	switch e.keyboardInput {
	case KeyboardInputAppKitField:
		keyboardInput = "the AppKit field"
	case KeyboardInputPage:
		keyboardInput = "the page"
	case KeyboardInputGpui:
		keyboardInput = "GPUI"
	}

	return gpui.NewDiv().
		SizeFull().
		Flex().
		FlexCol().
		Bg(gpui.Rgb(0x202428)).
		TextColor(gpui.Rgb(0xdddddd)).
		Child(
			gpui.NewDiv().
				P(gpui.Px(8.)).
				ChildText(fmt.Sprintf("Keyboard input is with %s.", keyboardInput)),
		).
		Child(gpui.NewDiv().Flex1().Child(e.webView))
}

func nsString(text string) objc.ID {
	return objc.NSStringFromGo(text)
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(window *gpui.Window, cx *gpui.App) gpui.Entity[Example] {
			return cx.New(func(cx *gpui.Context[Example]) *Example { return NewExample(window, cx) })
		}); err != nil {
			panic(fmt.Sprintf("opening the example window: %v", err))
		}
		cx.Activate(true)
	})
}
