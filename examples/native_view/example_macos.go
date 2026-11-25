//go:build darwin && !ios

// Command native_view shows a live WKWebView composited into a GPUI window on macOS.
//
// Things to try:
//   - Press cmd-shift-p while typing in the page. The palette opens across the page's edge, takes
//     keyboard input, and escape returns the caret to the same page field.
//   - Click the palette's button where it overlaps the page, then scroll the exposed page.
//   - Click the page while the palette is open, which dismisses the palette.
//   - Click the GPUI text box, type, then click back into the page.
//   - Click an empty part of the page and press some keys. The page sees every key first, and the
//     sidebar lists the ones it left for GPUI. It keeps cmd-k for itself, while cmd-shift-p is
//     reserved, so it opens the palette whatever the page does.
//   - Drag the divider, then load the other page or toggle the page's colors, and drag it again.
//     The strip that shows for a frame where the page is catching up takes the page's own
//     background color.
package main

import (
	"encoding/json/jsontext"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform"
	"github.com/megascan/gpui-go/scheduler"
)

const webKitPath = "/System/Library/Frameworks/WebKit.framework/WebKit"

var loadWebKit = sync.OnceValue(func() error {
	if _, err := purego.Dlopen(webKitPath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		return fmt.Errorf("loading WebKit: %w", err)
	}
	return nil
})

// TogglePalette opens or closes the palette.
type TogglePalette struct{}

// Name returns the action name.
func (TogglePalette) Name() string { return "native_view_example::TogglePalette" }

// BoxedClone clones the action.
func (TogglePalette) BoxedClone() gpui.Action { return TogglePalette{} }

// PartialEq reports whether other is a TogglePalette.
func (TogglePalette) PartialEq(other gpui.Action) bool {
	_, ok := other.(TogglePalette)
	return ok
}

// DismissPalette closes the palette.
type DismissPalette struct{}

// Name returns the action name.
func (DismissPalette) Name() string { return "native_view_example::DismissPalette" }

// BoxedClone clones the action.
func (DismissPalette) BoxedClone() gpui.Action { return DismissPalette{} }

// PartialEq reports whether other is a DismissPalette.
func (DismissPalette) PartialEq(other gpui.Action) bool {
	_, ok := other.(DismissPalette)
	return ok
}

func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "native_view_example::TogglePalette",
		Type:  reflect.TypeFor[TogglePalette](),
		Build: func(jsontext.Value) (gpui.Action, error) { return TogglePalette{}, nil },
	})
	gpui.RegisterAction(gpui.ActionData{
		Name:  "native_view_example::DismissPalette",
		Type:  reflect.TypeFor[DismissPalette](),
		Build: func(jsontext.Value) (gpui.Action, error) { return DismissPalette{}, nil },
	})
}

const reportBackground = `(function () {
  let reported = null;
  let scheduled = false;
  function report() {
    scheduled = false;
    const transparent = "rgba(0, 0, 0, 0)";
    let color = getComputedStyle(document.body).backgroundColor;
    if (color === transparent) {
      color = getComputedStyle(document.documentElement).backgroundColor;
    }
    if (color === transparent) {
      color = "rgb(255, 255, 255)";
    }
    if (color !== reported) {
      reported = color;
      window.webkit.messageHandlers.background.postMessage(color);
    }
  }
  function schedule() {
    if (!scheduled) {
      scheduled = true;
      requestAnimationFrame(report);
    }
  }
  report();
  window.addEventListener("load", schedule);
  const observer = new MutationObserver(schedule);
  observer.observe(document.documentElement, { attributes: true });
  observer.observe(document.body, { attributes: true });
  observer.observe(document.head, { attributes: true, childList: true, subtree: true });
  matchMedia("(prefers-color-scheme: dark)").addEventListener("change", schedule);
  for (const element of [document.documentElement, document.body]) {
    element.addEventListener("transitionend", schedule);
  }
})();`

const otherPage = `<!doctype html>
<html>
  <body style="font: 15px -apple-system; margin: 24px; color: #eee; background: #204060">
    <h2>Another page</h2>
    <p><input placeholder="Type here" size="36" /></p>
  </body>
</html>`

const page = `<!doctype html>
<html>
  <body style="font: 15px -apple-system; margin: 24px; color: #222; background: #fdf6e3">
    <h2>A live WKWebView</h2>
    <p>
      <input placeholder="Type here, then press cmd-shift-p" size="36" autofocus />
    </p>
    <p><textarea rows="3" cols="38">A textarea with some text to select.</textarea></p>
    <p contenteditable style="border: 1px solid #999; padding: 6px">An editable paragraph.</p>
    <p><a href="https://example.com">A link, for the pointing-hand cursor</a></p>
    <p id="kept">This page keeps cmd-k for itself.</p>
    <button onclick="this.textContent = 'Clicked ' + (++window.clicks || (window.clicks = 1))">
      A page button
    </button>
    <button onclick="document.body.style.background = window.dark ? '#fdf6e3' : '#30343a'; document.body.style.color = window.dark ? '#222' : '#ddd'; window.dark = !window.dark">
      Toggle the page's colors
    </button>
    <script>
      document.addEventListener("keydown", (event) => {
        if (event.metaKey && event.key === "k") {
          event.preventDefault();
          window.keptCount = (window.keptCount || 0) + 1;
          document.getElementById("kept").textContent =
            "This page kept cmd-k " + window.keptCount + " times.";
        }
      });
      for (let line = 1; line <= 60; line++) {
        const paragraph = document.createElement("p");
        paragraph.textContent = "Scrollable line " + line;
        document.body.appendChild(paragraph);
      }
    </script>
  </body>
</html>`

// Example composites a WKWebView next to GPUI content.
type Example struct {
	webView            gpui.NativeView
	webViewObject      *objc.Retained
	showingOtherPage   bool
	backgroundUpdates  *gpui.Task[struct{}]
	webViewVisible     bool
	sidebarWidth       gpui.Pixels
	resizingSidebar    bool
	textBoxFocus       gpui.FocusHandle
	textBoxText        string
	paletteFocus       gpui.FocusHandle
	paletteText        string
	paletteClicks      int
	focusBeforePalette gpui.Option[gpui.FocusHandle]
	keysLeftByPage     []string
}

// NewExample creates the web view and the example's state.
func NewExample(window *gpui.Window, cx *gpui.Context[Example]) *Example {
	if err := loadWebKit(); err != nil {
		panic(err)
	}
	backgroundSender, backgrounds := scheduler.Chan[string](0)

	webView := objc.Own(objc.AllocInit(objc.GetClass("WKWebView")))

	configuration := objc.Send[objc.ID](webView.ID(), objc.Sel("configuration"))
	contentController := objc.Send[objc.ID](configuration, objc.Sel("userContentController"))
	const atDocumentEnd = 1
	script := objc.Own(objc.Send[objc.ID](
		objc.Alloc(objc.GetClass("WKUserScript")),
		objc.Sel("initWithSource:injectionTime:forMainFrameOnly:"),
		nsString(reportBackground),
		objc.NSInteger(atDocumentEnd),
		true,
	))
	objc.SendVoid(contentController, objc.Sel("addUserScript:"), script.ID())
	script.Release()
	reporter := backgroundReporter(backgroundSender)
	objc.SendVoid(
		contentController,
		objc.Sel("addScriptMessageHandler:name:"),
		reporter.ID(),
		nsString("background"),
	)
	reporter.Release()

	drawsBackground := objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithBool:"), false)
	objc.SendVoid(
		webView.ID(),
		objc.Sel("setValue:forKey:"),
		drawsBackground,
		nsString("drawsBackground"),
	)
	loadPage(webView.ID(), page)
	nativeView, err := gpui.NewNativeView(uintptr(webView.ID()), window, cx.App)
	if err != nil {
		panic(fmt.Sprintf("macOS windows composite native views: %v", err))
	}
	nativeView.ReserveAction[TogglePalette]()

	backgroundUpdates := cx.Spawn[struct{}](func(this gpui.WeakEntity[Example], cx *gpui.AsyncApp) (struct{}, error) {
		defer backgrounds.Close()
		for {
			color, ok := backgrounds.Recv(cx)
			if !ok {
				return struct{}{}, nil
			}
			parsed, ok := parseCSSColor(color)
			if !ok {
				continue
			}
			if err := this.Update(cx, func(this *Example, cx *gpui.Context[Example]) {
				this.webView.SetBackground(parsed, cx.App)
			}); err != nil {
				return struct{}{}, nil
			}
		}
	})

	return &Example{
		webView:            nativeView,
		webViewObject:      webView,
		showingOtherPage:   false,
		backgroundUpdates:  backgroundUpdates,
		webViewVisible:     true,
		sidebarWidth:       gpui.Px(240.),
		resizingSidebar:    false,
		textBoxFocus:       cx.FocusHandle(),
		textBoxText:        "",
		paletteFocus:       cx.FocusHandle(),
		paletteText:        "",
		paletteClicks:      0,
		focusBeforePalette: gpui.None[gpui.FocusHandle](),
		keysLeftByPage:     nil,
	}
}

func (e *Example) togglePalette(_ *TogglePalette, window *gpui.Window, cx *gpui.Context[Example]) {
	if e.focusBeforePalette.IsSome() {
		e.dismissPalette(&DismissPalette{}, window, cx)
		return
	}
	focused, ok := window.Focused(cx.App)
	if !ok {
		focused = e.textBoxFocus.Clone()
	}
	e.focusBeforePalette = gpui.Some(focused)
	window.Focus(e.paletteFocus, cx.App)
	cx.Notify()
}

func (e *Example) dismissPalette(_ *DismissPalette, window *gpui.Window, cx *gpui.Context[Example]) {
	previousFocus, ok := e.focusBeforePalette.Get()
	e.focusBeforePalette = gpui.None[gpui.FocusHandle]()
	if ok {
		window.Focus(previousFocus, cx.App)
		previousFocus.Release()
		cx.Notify()
	}
}

func (e *Example) renderPalette(cx *gpui.Context[Example]) gpui.IntoElement {
	return gpui.NewDeferred(
		gpui.NewAnchored().Position(gpui.NewPoint(gpui.Px(170.), gpui.Px(120.))).Child(
			gpui.NewDiv().
				ID("palette").
				TrackFocus(e.paletteFocus).
				OnMouseDownOut(cx.Listener[gpui.MouseDownEvent](func(this *Example, _ *gpui.MouseDownEvent, window *gpui.Window, cx *gpui.Context[Example]) {
					this.dismissPalette(&DismissPalette{}, window, cx)
				})).
				OnKeyDown(cx.Listener[gpui.KeyDownEvent](func(this *Example, event *gpui.KeyDownEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
					if text, ok := event.Keystroke.KeyChar.Get(); ok {
						this.paletteText += text
						cx.Notify()
					}
				})).
				Occlude().
				W(gpui.Px(320.)).
				P4().
				Flex().
				FlexCol().
				Gap3().
				RoundedLg().
				ShadowXl().
				Bg(gpui.Rgb(0x1f2933)).
				TextColor(gpui.White()).
				ChildText("A GPUI palette over the page").
				Child(
					gpui.NewDiv().
						P2().
						RoundedMd().
						Bg(gpui.Rgb(0x323f4b)).
						ChildText(fmt.Sprintf("Typed: %s", e.paletteText)),
				).
				Child(
					gpui.NewDiv().
						ID("palette-button").
						P2().
						RoundedMd().
						Bg(gpui.Rgb(0x3e7bfa)).
						CursorPointer().
						ChildText(fmt.Sprintf("Clicked %d times", e.paletteClicks)).
						OnClick(cx.Listener[gpui.ClickEvent](func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							this.paletteClicks++
							cx.Notify()
						})),
				),
		),
	).
		WithPriority(1)
}

func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}

// Render renders the example.
func (e *Example) Render(window *gpui.Window, cx *gpui.Context[Example]) gpui.IntoElement {
	textBoxFocused := e.textBoxFocus.IsFocused(window)
	textBoxBorder := gpui.Rgb(0x9aa5b1)
	if textBoxFocused {
		textBoxBorder = gpui.Rgb(0x3e7bfa)
	}
	toggleLabel := "Show the web view"
	if e.webViewVisible {
		toggleLabel = "Hide the web view"
	}

	return gpui.NewDiv().
		SizeFull().
		Flex().
		Bg(gpui.Rgb(0xf0f4f8)).
		TextColor(gpui.Rgb(0x1f2933)).
		OnAction(actionListener(cx.Listener((*Example).togglePalette))).
		OnAction(actionListener(cx.Listener((*Example).dismissPalette))).
		OnKeyDown(cx.Listener[gpui.KeyDownEvent](func(this *Example, event *gpui.KeyDownEvent, window *gpui.Window, cx *gpui.Context[Example]) {
			webViewFocus := this.webView.FocusHandle(cx.App)
			defer webViewFocus.Release()
			if webViewFocus.IsFocused(window) {
				this.keysLeftByPage = append(this.keysLeftByPage, event.Keystroke.Unparse(gpui.OperatingSystemMac))
				excess := max(len(this.keysLeftByPage)-8, 0)
				this.keysLeftByPage = this.keysLeftByPage[excess:]
				cx.Notify()
			}
		})).
		Child(
			gpui.NewDiv().
				FlexNone().
				W(e.sidebarWidth).
				P4().
				Flex().
				FlexCol().
				Gap3().
				ChildText("GPUI content").
				Child(
					gpui.NewDiv().
						ID("text-box").
						TrackFocus(e.textBoxFocus).
						OnKeyDown(cx.Listener[gpui.KeyDownEvent](func(this *Example, event *gpui.KeyDownEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							if text, ok := event.Keystroke.KeyChar.Get(); ok {
								this.textBoxText += text
								cx.Notify()
							}
						})).
						OnClick(cx.Listener[gpui.ClickEvent](func(this *Example, _ *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[Example]) {
							window.Focus(this.textBoxFocus, cx.App)
						})).
						MinH(gpui.Px(60.)).
						P2().
						RoundedMd().
						Border1().
						BorderColor(textBoxBorder).
						Bg(gpui.White()).
						ChildText(fmt.Sprintf("Click, then type: %s", e.textBoxText)),
				).
				Child(
					gpui.NewDiv().
						ID("toggle-web-view").
						P2().
						RoundedMd().
						Bg(gpui.Rgb(0xd9e2ec)).
						CursorPointer().
						ChildText(toggleLabel).
						OnClick(cx.Listener[gpui.ClickEvent](func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							this.webViewVisible = !this.webViewVisible
							cx.Notify()
						})),
				).
				Child(
					gpui.NewDiv().
						ID("load-other-page").
						P2().
						RoundedMd().
						Bg(gpui.Rgb(0xd9e2ec)).
						CursorPointer().
						ChildText("Load the other page").
						OnClick(cx.Listener[gpui.ClickEvent](func(this *Example, _ *gpui.ClickEvent, _ *gpui.Window, _ *gpui.Context[Example]) {
							this.showingOtherPage = !this.showingOtherPage
							next := page
							if this.showingOtherPage {
								next = otherPage
							}
							loadPage(this.webViewObject.ID(), next)
						})),
				).
				ChildText("cmd-shift-p toggles the palette, escape dismisses it.").
				ChildText(fmt.Sprintf(
					"Keys the page left for GPUI: %s",
					strings.Join(e.keysLeftByPage, " "),
				)),
		).
		Child(
			gpui.NewDiv().
				ID("divider").
				FlexNone().
				W(gpui.Px(6.)).
				HFull().
				Bg(gpui.Rgb(0x9aa5b1)).
				CursorColResize().
				OnMouseDown(
					gpui.MouseButtonLeft,
					cx.Listener[gpui.MouseDownEvent](func(this *Example, _ *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
						this.resizingSidebar = true
						cx.Notify()
					}),
				),
		).
		Child(
			gpui.NewDiv().
				Flex1().
				MinW0().
				HFull().
				When(e.webViewVisible, func(this *gpui.Div) *gpui.Div {
					return this.Child(e.webView)
				}),
		).
		When(e.focusBeforePalette.IsSome(), func(this *gpui.Div) *gpui.Div {
			return this.Child(e.renderPalette(cx))
		}).
		When(e.resizingSidebar, func(this *gpui.Div) *gpui.Div {
			return this.Child(
				gpui.NewDiv().
					Absolute().
					Inset0().
					Occlude().
					CursorColResize().
					OnMouseMove(cx.Listener[gpui.MouseMoveEvent](func(this *Example, event *gpui.MouseMoveEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
						this.sidebarWidth = min(max(event.Position.X, gpui.Px(120.)), gpui.Px(600.))
						cx.Notify()
					})).
					OnMouseUp(
						gpui.MouseButtonLeft,
						cx.Listener[gpui.MouseUpEvent](func(this *Example, _ *gpui.MouseUpEvent, _ *gpui.Window, cx *gpui.Context[Example]) {
							this.resizingSidebar = false
							cx.Notify()
						}),
					),
			)
		})
}

func nsString(text string) objc.ID {
	return objc.NSStringFromGo(text)
}

func loadPage(webView objc.ID, html string) {
	objc.Send[objc.ID](
		webView,
		objc.Sel("loadHTMLString:baseURL:"),
		nsString(html),
		objc.ID(0),
	)
}

var (
	reporterSendersMu sync.Mutex
	reporterSenders   = map[objc.ID]*scheduler.ChanSender[string]{}
)

func reporterReceived(this objc.ID, _ objc.SEL, _ objc.ID, message objc.ID) {
	reporterSendersMu.Lock()
	sender, ok := reporterSenders[this]
	reporterSendersMu.Unlock()
	if !ok {
		return
	}
	body := objc.Send[objc.ID](message, objc.Sel("body"))
	if body == 0 {
		return
	}
	color := objc.GoString(body)
	_ = sender.TrySend(color)
}

var reporterClass = sync.OnceValue(func() objc.Class {
	class, err := objc.RegisterClass(
		"NativeViewExampleBackgroundReporter",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{{
			Cmd: objc.Sel("userContentController:didReceiveScriptMessage:"),
			Fn:  reporterReceived,
		}},
	)
	if err != nil {
		panic(fmt.Sprintf("the class is only registered here: %v", err))
	}
	return class
})

func backgroundReporter(sender *scheduler.ChanSender[string]) *objc.Retained {
	reporter := objc.AllocInit(reporterClass())
	reporterSendersMu.Lock()
	reporterSenders[reporter] = sender
	reporterSendersMu.Unlock()
	return objc.Own(reporter)
}

func parseCSSColor(text string) (gpui.Rgba, bool) {
	_, rest, ok := strings.Cut(text, "(")
	if !ok {
		return gpui.Rgba{}, false
	}
	components, ok := strings.CutSuffix(rest, ")")
	if !ok {
		return gpui.Rgba{}, false
	}
	parts := strings.Split(components, ",")
	var rgb [3]float32
	for i := range rgb {
		if i >= len(parts) {
			return gpui.Rgba{}, false
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 32)
		if err != nil {
			return gpui.Rgba{}, false
		}
		rgb[i] = float32(value)
	}
	return gpui.Rgba{
		R: rgb[0] / 255.,
		G: rgb[1] / 255.,
		B: rgb[2] / 255.,
		A: 1.,
	}, true
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("cmd-shift-p", TogglePalette{}, gpui.None[string]()),
			gpui.NewKeyBinding("escape", DismissPalette{}, gpui.None[string]()),
		})
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(window *gpui.Window, cx *gpui.App) gpui.Entity[Example] {
			return cx.New(func(cx *gpui.Context[Example]) *Example { return NewExample(window, cx) })
		}); err != nil {
			panic(fmt.Sprintf("opening the example window: %v", err))
		}
		cx.Activate(true)
	})
}
