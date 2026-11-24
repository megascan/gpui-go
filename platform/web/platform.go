//go:build js && wasm

package web

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// WebPlatform is the GPUI platform implementation for web browsers. It starts with an empty
// font database: applications must add fonts through App.TextSystem before opening a window.
type WebPlatform struct {
	gpui.PlatformDefaults
	browserWindow          js.Value
	platformInfo           gpui.PlatformInfo
	backgroundExecutor     gpui.BackgroundExecutor
	foregroundExecutor     gpui.ForegroundExecutor
	textSystem             gpui.PlatformTextSystem
	activeWindow           *gpui.Option[gpui.AnyWindowHandle]
	activeDisplay          gpui.PlatformDisplay
	callbacks              webPlatformCallbacks
	backendPreference      WebBackendPreference
	wgpuContext            **wgpu.WgpuContext
	preparedWindow         *gpui.Option[preparedWebWindow]
	windowLifecycle        *webWindowLifecycle
	cursorVisible          *bool
	lastCursorCSS          *string
	gestures               *webGestures
	cursorRestoreListeners []*EventListenerHandle
}

var _ gpui.Platform = (*WebPlatform)(nil)

type webGestures struct {
	tuning gpui.GestureTuning
}

func webGesturesFromUserAgent(userAgent string) *webGestures {
	var scrollPhysics gpui.ScrollPhysics
	if strings.Contains(userAgent, "Android") {
		scrollPhysics = gpui.ScrollPhysicsAndroid()
	} else {
		scrollPhysics = gpui.ScrollPhysicsIos()
	}
	tuning := gpui.DefaultGestureTuning()
	tuning.ScrollPhysics = scrollPhysics
	return &webGestures{tuning: tuning}
}

// Tuning returns the browser's gesture feel.
func (g *webGestures) Tuning() gpui.GestureTuning {
	return g.tuning
}

// NativeRecognizers returns no recognizers.
func (g *webGestures) NativeRecognizers() gpui.GestureKinds {
	return gpui.GestureKindsNone
}

type preparedWebWindow struct {
	canvas   js.Value
	graphics *wgpu.PreparedWebGraphics
}

type webWindowLifecycle uint8

const (
	webWindowLifecycleAvailable webWindowLifecycle = iota
	webWindowLifecycleOpen
	webWindowLifecycleClosed
	webWindowLifecycleUnavailable
)

// WebWindowErrorKind is the variant of a WebWindowError.
type WebWindowErrorKind uint8

const (
	// WebWindowErrorAlreadyOpen means a window is already open.
	WebWindowErrorAlreadyOpen WebWindowErrorKind = iota
	// WebWindowErrorReopeningUnsupported means the window was closed and cannot be reopened.
	WebWindowErrorReopeningUnsupported
	// WebWindowErrorUnsupportedWindowKind means the window kind cannot be a top-level window.
	WebWindowErrorUnsupportedWindowKind
	// WebWindowErrorGraphicsInitializationPending means graphics initialization has not completed
	// yet; retrying after it finishes (e.g. from the Platform.Run callback) can succeed.
	WebWindowErrorGraphicsInitializationPending
	// WebWindowErrorGraphicsUnavailable means graphics initialization or an earlier window
	// creation failed; retrying cannot succeed.
	WebWindowErrorGraphicsUnavailable
)

// WebWindowError is an error of WebPlatform.OpenWindow. It is comparable.
type WebWindowError struct {
	Kind WebWindowErrorKind
	// WindowKind names the rejected kind of an UnsupportedWindowKind error.
	WindowKind string
}

// Error formats the error as upstream's Display.
func (e WebWindowError) Error() string {
	switch e.Kind {
	case WebWindowErrorAlreadyOpen:
		return "GPUI web supports only one top-level window; a window is already open"
	case WebWindowErrorReopeningUnsupported:
		return "reopening the GPUI web top-level window after it closes is not supported"
	case WebWindowErrorUnsupportedWindowKind:
		return fmt.Sprintf("GPUI web does not support %s as a separate top-level window; render it inside the normal window instead", e.WindowKind)
	case WebWindowErrorGraphicsInitializationPending:
		return "browser graphics initialization has not completed yet; open windows from the callback passed to Platform::run"
	}
	return "browser graphics are unavailable because graphics initialization or an earlier window creation failed"
}

type webPlatformCallbacks struct {
	openURLs               gpui.Option[func([]string)]
	quit                   gpui.Option[func() bool]
	reopen                 gpui.Option[func()]
	appMenuAction          gpui.Option[func(gpui.Action)]
	willOpenAppMenu        gpui.Option[func()]
	validateAppMenuCommand gpui.Option[func(gpui.Action) bool]
	keyboardLayoutChange   gpui.Option[func()]
	thermalStateChange     gpui.Option[func()]
}

// NewWebPlatform returns the web platform with the automatic backend preference.
func NewWebPlatform(allowMultiThreading bool) *WebPlatform {
	return NewWebPlatformWithBackend(allowMultiThreading, WebBackendPreferenceAuto)
}

// NewWebPlatformWithBackend returns the web platform with backendPreference.
func NewWebPlatformWithBackend(allowMultiThreading bool, backendPreference WebBackendPreference) *WebPlatform {
	return NewWebPlatformWithBackendAndFontFallback(allowMultiThreading, backendPreference, DefaultCanvasFontFallback())
}

// NewWebPlatformWithBackendAndFontFallback configures browser fallback before any fonts or
// layouts are cached. Loaded fonts remain preferred, except when they cannot supply requested
// emoji presentation. The policy cannot be changed after construction.
func NewWebPlatformWithBackendAndFontFallback(allowMultiThreading bool, backendPreference WebBackendPreference, canvasFontFallback CanvasFontFallback) *WebPlatform {
	browserWindow := browserWindow()
	info := platformInfo(browserWindow)
	dispatcher := NewWebDispatcher(browserWindow, allowMultiThreading)
	backgroundExecutor := gpui.NewBackgroundExecutor(dispatcher)
	foregroundExecutor := gpui.NewForegroundExecutor(dispatcher)
	textSystem := newWebTextSystem("IBM Plex Sans", canvasFontFallback)
	activeDisplay := gpui.PlatformDisplay(NewWebDisplay(browserWindow))

	cursorVisible := new(bool)
	*cursorVisible = true
	lastCursorCSS := new(string)
	*lastCursorCSS = "default"
	cursorRestoreListeners := cursorRestoreListeners(browserWindow, cursorVisible, lastCursorCSS)
	userAgent, _ := jsString(jsGet(jsGet(browserWindow, "navigator"), "userAgent"))
	gestures := webGesturesFromUserAgent(userAgent)

	lifecycle := new(webWindowLifecycle)
	*lifecycle = webWindowLifecycleAvailable
	activeWindow := new(gpui.Option[gpui.AnyWindowHandle])
	prepared := new(gpui.Option[preparedWebWindow])

	p := &WebPlatform{
		browserWindow:          browserWindow,
		platformInfo:           info,
		backgroundExecutor:     backgroundExecutor,
		foregroundExecutor:     foregroundExecutor,
		textSystem:             textSystem,
		activeWindow:           activeWindow,
		activeDisplay:          activeDisplay,
		backendPreference:      backendPreference,
		wgpuContext:            new(*wgpu.WgpuContext),
		preparedWindow:         prepared,
		windowLifecycle:        lifecycle,
		cursorVisible:          cursorVisible,
		lastCursorCSS:          lastCursorCSS,
		gestures:               gestures,
		cursorRestoreListeners: cursorRestoreListeners,
	}
	gpui.BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

// FetchHTTPClient returns an HTTP client that runs browser Fetch operations on this platform's
// main thread.
func (p *WebPlatform) FetchHTTPClient() FetchHTTPClient {
	return NewFetchHTTPClient(p.foregroundExecutor.ToSend())
}

// FetchHTTPClientWithUserAgent returns a browser Fetch HTTP client with the given reported user agent.
func (p *WebPlatform) FetchHTTPClientWithUserAgent(userAgent string) (FetchHTTPClient, error) {
	return FetchHTTPClientWithUserAgent(p.foregroundExecutor.ToSend(), userAgent)
}

func isBrowserWebGPUSupported(s *scheduler.Scope) bool {
	gpu := jsGet(jsGet(js.Global(), "navigator"), "gpu")
	if jsIsNullish(gpu) {
		return false
	}
	promise, err := jsCall(gpu, "requestAdapter")
	if err != nil {
		return false
	}
	adapter, err := awaitPromise(s, promise)
	return err == nil && !jsIsNullish(adapter)
}

func removeCanvas(canvas js.Value) {
	_, _ = jsCall(canvas, "remove")
}

func initializeGraphics(s *scheduler.Scope, browserWindow js.Value, preference WebBackendPreference) (js.Value, *wgpu.PreparedWebGraphics, error) {
	switch preference {
	case WebBackendPreferenceAuto:
		webgpuCanvas, err := prepareCanvas(browserWindow)
		if err != nil {
			return js.Undefined(), nil, err
		}
		var webgpuResult *wgpu.PreparedWebGraphics
		var webgpuError error
		if isBrowserWebGPUSupported(s) {
			webgpuResult, webgpuError = wgpu.NewWgpuContextWeb(s, webgpuCanvas, WebBackendPreferenceWebGpu)
		} else {
			webgpuError = errors.New("browser WebGPU probe did not return a usable adapter")
		}
		if webgpuError == nil {
			return webgpuCanvas, webgpuResult, nil
		}
		removeCanvas(webgpuCanvas)
		slog.Warn(fmt.Sprintf("WebGPU initialization failed; falling back to WebGL2: %v", webgpuError))

		webglCanvas, err := prepareCanvas(browserWindow)
		if err != nil {
			return js.Undefined(), nil, fmt.Errorf("WebGPU initialization failed: %v. Failed to prepare a replacement canvas for WebGL2: %w", webgpuError, err)
		}
		graphics, webglError := wgpu.NewWgpuContextWeb(s, webglCanvas, WebBackendPreferenceWebGl)
		if webglError != nil {
			removeCanvas(webglCanvas)
			return js.Undefined(), nil, fmt.Errorf("No browser graphics backend could be initialized. Tried WebGPU, then WebGL2. WebGPU failure: %v. WebGL2 failure: %w", webgpuError, webglError)
		}
		return webglCanvas, graphics, nil
	default:
		backendName := "WebGL2"
		if preference == WebBackendPreferenceWebGpu {
			backendName = "WebGPU"
		}
		canvas, err := prepareCanvas(browserWindow)
		if err != nil {
			return js.Undefined(), nil, err
		}
		graphics, err := wgpu.NewWgpuContextWeb(s, canvas, preference)
		if err != nil {
			removeCanvas(canvas)
			return js.Undefined(), nil, fmt.Errorf("No browser graphics backend could be initialized. Only %s was tried because the application requested it explicitly. %s failure: %w", backendName, backendName, err)
		}
		return canvas, graphics, nil
	}
}

// PlatformInfo returns the browser's host operating system with the web target.
func (p *WebPlatform) PlatformInfo() gpui.PlatformInfo {
	return p.platformInfo
}

// BackgroundExecutor returns the background executor.
func (p *WebPlatform) BackgroundExecutor() gpui.BackgroundExecutor {
	return p.backgroundExecutor
}

// Gestures returns the browser's gesture tuning.
func (p *WebPlatform) Gestures() gpui.PlatformGestures {
	return p.gestures
}

// ForegroundExecutor returns the foreground executor.
func (p *WebPlatform) ForegroundExecutor() gpui.ForegroundExecutor {
	return p.foregroundExecutor
}

// TextSystem returns the web text system.
func (p *WebPlatform) TextSystem() gpui.PlatformTextSystem {
	return p.textSystem
}

// Run initializes browser graphics and calls onFinishLaunching once they are ready. It returns
// at once; the program's main must keep running (e.g. with select {}) for the page to stay live.
func (p *WebPlatform) Run(onFinishLaunching func()) {
	wgpuContext := p.wgpuContext
	preparedWindow := p.preparedWindow
	windowLifecycle := p.windowLifecycle
	browserWindow := p.browserWindow
	backendPreference := p.backendPreference
	p.foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		canvas, graphics, err := initializeGraphics(s, browserWindow, backendPreference)
		if err != nil {
			*windowLifecycle = webWindowLifecycleUnavailable
			slog.Error(fmt.Sprintf("Failed to initialize browser graphics: %v", err))
			showGraphicsUnavailableMessage(browserWindow, err)
			return struct{}{}, nil
		}
		slog.Info(fmt.Sprintf("Browser graphics initialized successfully with %s", graphics.Context.Backend()))
		*wgpuContext = graphics.Context
		graphics.Context = nil
		*preparedWindow = gpui.Some(preparedWebWindow{canvas: canvas, graphics: graphics})
		onFinishLaunching()
		return struct{}{}, nil
	}).Detach()
}

// Quit logs that quitting is unsupported.
func (p *WebPlatform) Quit() {
	slog.Warn("WebPlatform::quit called, but quitting is not supported in the browser .")
}

// Restart does nothing.
func (p *WebPlatform) Restart(binaryPath gpui.Option[string], arguments []string) {}

// Activate does nothing.
func (p *WebPlatform) Activate(ignoringOtherApps bool) {}

// Hide does nothing.
func (p *WebPlatform) Hide() {}

// HideOtherApps does nothing.
func (p *WebPlatform) HideOtherApps() {}

// UnhideOtherApps does nothing.
func (p *WebPlatform) UnhideOtherApps() {}

// Displays returns the browser display.
func (p *WebPlatform) Displays() []gpui.PlatformDisplay {
	return []gpui.PlatformDisplay{p.activeDisplay}
}

// PrimaryDisplay returns the browser display.
func (p *WebPlatform) PrimaryDisplay() gpui.PlatformDisplay {
	return p.activeDisplay
}

// OnDisplaysChanged does nothing: the browser is the only display, and it never changes.
func (p *WebPlatform) OnDisplaysChanged(callback func()) {}

// ActiveWindow returns the open window, if any.
func (p *WebPlatform) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	return p.activeWindow.Get()
}

// OpenWindow opens the single top-level window in the prepared canvas.
func (p *WebPlatform) OpenWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error) {
	switch params.Kind.(type) {
	case gpui.WindowKindNormal:
	case gpui.WindowKindAnchoredPopup:
		return nil, gpui.PopupNotSupportedError{}
	case gpui.WindowKindPopUp:
		return nil, WebWindowError{Kind: WebWindowErrorUnsupportedWindowKind, WindowKind: "popup windows"}
	case gpui.WindowKindFloating:
		return nil, WebWindowError{Kind: WebWindowErrorUnsupportedWindowKind, WindowKind: "floating windows"}
	case gpui.WindowKindDialog:
		return nil, WebWindowError{Kind: WebWindowErrorUnsupportedWindowKind, WindowKind: "dialog windows"}
	default:
		return nil, WebWindowError{Kind: WebWindowErrorUnsupportedWindowKind, WindowKind: "layer shell windows"}
	}

	switch *p.windowLifecycle {
	case webWindowLifecycleOpen:
		return nil, WebWindowError{Kind: WebWindowErrorAlreadyOpen}
	case webWindowLifecycleClosed:
		return nil, WebWindowError{Kind: WebWindowErrorReopeningUnsupported}
	case webWindowLifecycleUnavailable:
		return nil, WebWindowError{Kind: WebWindowErrorGraphicsUnavailable}
	}

	context := *p.wgpuContext
	if context == nil {
		return nil, WebWindowError{Kind: WebWindowErrorGraphicsInitializationPending}
	}
	prepared, ok := p.preparedWindow.Take().Get()
	if !ok {
		return nil, WebWindowError{Kind: WebWindowErrorGraphicsInitializationPending}
	}
	canvas := prepared.canvas

	window, err := newWebWindow(handle, params, context, canvas, prepared.graphics, p.browserWindow, p.windowLifecycle, p.activeWindow)
	if err != nil {
		prepared.graphics.Release()
		removeCanvas(canvas)
		*p.windowLifecycle = webWindowLifecycleUnavailable
		return nil, err
	}
	*p.windowLifecycle = webWindowLifecycleOpen
	*p.activeWindow = gpui.Some(handle)
	return window, nil
}

// WindowAppearance returns the page's preferred color scheme.
func (p *WebPlatform) WindowAppearance() gpui.WindowAppearance {
	mediaQuery, err := jsCall(p.browserWindow, "matchMedia", "(prefers-color-scheme: dark)")
	if err != nil || jsIsNullish(mediaQuery) {
		return gpui.WindowAppearanceLight
	}
	if matches, _ := jsBool(jsGet(mediaQuery, "matches")); matches {
		return gpui.WindowAppearanceDark
	}
	return gpui.WindowAppearanceLight
}

// OpenURL opens url with window.open.
func (p *WebPlatform) OpenURL(url string) {
	if _, err := jsCall(p.browserWindow, "open", url); err != nil {
		slog.Warn(fmt.Sprintf("Failed to open URL '%s': %s", url, jsErrorDebug(err)))
	}
}

// OnOpenURLs registers the open-URLs callback.
func (p *WebPlatform) OnOpenURLs(callback func(urls []string)) {
	p.callbacks.openURLs = gpui.Some(callback)
}

// RegisterURLScheme succeeds without doing anything.
func (p *WebPlatform) RegisterURLScheme(url string) *gpui.Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// PromptForPaths fails: the web has no file dialog.
func (p *WebPlatform) PromptForPaths(options gpui.PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[[]string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[gpui.Option[[]string]]]()
	_ = tx.Send(scheduler.Result[gpui.Option[[]string]]{Err: errors.New("prompt_for_paths is not supported on the web")})
	return rx
}

// PromptForNewPath fails: the web has no file dialog.
func (p *WebPlatform) PromptForNewPath(directory string, suggestedName gpui.Option[string]) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[string]]] {
	sender, receiver := scheduler.Oneshot[scheduler.Result[gpui.Option[string]]]()
	_ = sender.Send(scheduler.Result[gpui.Option[string]]{Err: errors.New("prompt_for_new_path is not supported on the web")})
	return receiver
}

// CanSelectMixedFilesAndDirs reports false.
func (p *WebPlatform) CanSelectMixedFilesAndDirs() bool {
	return false
}

// RevealPath does nothing.
func (p *WebPlatform) RevealPath(path string) {}

// OpenWithSystem does nothing.
func (p *WebPlatform) OpenWithSystem(path string) {}

// OnQuit registers the quit callback.
func (p *WebPlatform) OnQuit(callback func() bool) {
	p.callbacks.quit = gpui.Some(callback)
}

// OnReopen registers the reopen callback.
func (p *WebPlatform) OnReopen(callback func()) {
	p.callbacks.reopen = gpui.Some(callback)
}

// OnSystemSleep does nothing: browsers expose no system sleep signal.
func (p *WebPlatform) OnSystemSleep(callback func()) {}

// OnSystemWake does nothing: browsers expose no system wake signal.
func (p *WebPlatform) OnSystemWake(callback func()) {}

// SetMenus does nothing.
func (p *WebPlatform) SetMenus(menus []gpui.Menu, keymap *gpui.Keymap) {}

// SetDockMenu does nothing.
func (p *WebPlatform) SetDockMenu(menu []gpui.MenuItem, keymap *gpui.Keymap) {}

// OnAppMenuAction registers the app menu action callback.
func (p *WebPlatform) OnAppMenuAction(callback func(action gpui.Action)) {
	p.callbacks.appMenuAction = gpui.Some(callback)
}

// OnWillOpenAppMenu registers the will-open-app-menu callback.
func (p *WebPlatform) OnWillOpenAppMenu(callback func()) {
	p.callbacks.willOpenAppMenu = gpui.Some(callback)
}

// OnValidateAppMenuCommand registers the menu validation callback.
func (p *WebPlatform) OnValidateAppMenuCommand(callback func(action gpui.Action) bool) {
	p.callbacks.validateAppMenuCommand = gpui.Some(callback)
}

// ThermalState returns nominal.
func (p *WebPlatform) ThermalState() gpui.ThermalState {
	return gpui.ThermalStateNominal
}

// OnThermalStateChange registers the thermal state callback.
func (p *WebPlatform) OnThermalStateChange(callback func()) {
	p.callbacks.thermalStateChange = gpui.Some(callback)
}

// PreventIdleSleep fails: the browser cannot prevent idle sleep.
func (p *WebPlatform) PreventIdleSleep(reason string) *gpui.Task[*gpui.ActivityGuard] {
	return scheduler.ReadyTaskErr[*gpui.ActivityGuard](fmt.Errorf("Idle sleep prevention for %q is not supported in the browser", reason))
}

// CompositorName returns "Web".
func (p *WebPlatform) CompositorName() string {
	return "Web"
}

// AppPath fails on the web.
func (p *WebPlatform) AppPath() (string, error) {
	return "", errors.New("app_path is not available on the web")
}

// PathForAuxiliaryExecutable fails on the web.
func (p *WebPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	return "", errors.New("path_for_auxiliary_executable is not available on the web")
}

// SetCursorStyle sets the body's CSS cursor.
func (p *WebPlatform) SetCursorStyle(style gpui.CursorStyle) {
	var cssCursor string
	switch style {
	case gpui.CursorStyleArrow:
		cssCursor = "default"
	case gpui.CursorStyleIBeam:
		cssCursor = "text"
	case gpui.CursorStyleCrosshair:
		cssCursor = "crosshair"
	case gpui.CursorStyleClosedHand:
		cssCursor = "grabbing"
	case gpui.CursorStyleOpenHand:
		cssCursor = "grab"
	case gpui.CursorStylePointingHand:
		cssCursor = "pointer"
	case gpui.CursorStyleResizeLeft, gpui.CursorStyleResizeRight, gpui.CursorStyleResizeLeftRight:
		cssCursor = "ew-resize"
	case gpui.CursorStyleResizeUp, gpui.CursorStyleResizeDown, gpui.CursorStyleResizeUpDown:
		cssCursor = "ns-resize"
	case gpui.CursorStyleResizeUpLeftDownRight:
		cssCursor = "nesw-resize"
	case gpui.CursorStyleResizeUpRightDownLeft:
		cssCursor = "nwse-resize"
	case gpui.CursorStyleResizeColumn:
		cssCursor = "col-resize"
	case gpui.CursorStyleResizeRow:
		cssCursor = "row-resize"
	case gpui.CursorStyleIBeamCursorForVerticalLayout:
		cssCursor = "vertical-text"
	case gpui.CursorStyleOperationNotAllowed:
		cssCursor = "not-allowed"
	case gpui.CursorStyleDragLink:
		cssCursor = "alias"
	case gpui.CursorStyleDragCopy:
		cssCursor = "copy"
	case gpui.CursorStyleContextualMenu:
		cssCursor = "context-menu"
	}

	*p.lastCursorCSS = cssCursor
	if *p.cursorVisible {
		setBodyCursor(p.browserWindow, cssCursor)
	}
}

// HideCursorUntilMouseMoves hides the cursor until the next mouse movement.
func (p *WebPlatform) HideCursorUntilMouseMoves() {
	wasVisible := *p.cursorVisible
	*p.cursorVisible = false
	if !wasVisible {
		return
	}
	setBodyCursor(p.browserWindow, "none")
}

// IsCursorVisible reports whether the cursor is visible.
func (p *WebPlatform) IsCursorVisible() bool {
	return *p.cursorVisible
}

// ShouldAutoHideScrollbars reports true.
func (p *WebPlatform) ShouldAutoHideScrollbars() bool {
	return true
}

// ReadFromClipboard returns false: the browser clipboard is only readable asynchronously.
func (p *WebPlatform) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	return gpui.ClipboardItem{}, false
}

// ReadFromClipboardAsync reads text and images through navigator.clipboard.read; its task
// fails with a gpui.ClipboardReadError.
func (p *WebPlatform) ReadFromClipboardAsync() *gpui.Task[gpui.Option[gpui.ClipboardItem]] {
	navigator := jsGet(p.browserWindow, "navigator")
	clipboard := jsGet(navigator, "clipboard")
	if jsIsNullish(clipboard) {
		return scheduler.ReadyTaskErr[gpui.Option[gpui.ClipboardItem]](gpui.ClipboardReadErrorUnavailable())
	}
	read, err := jsCall(clipboard, "read")
	if err != nil {
		return scheduler.ReadyTaskErr[gpui.Option[gpui.ClipboardItem]](clipboardReadRejectionError(jsErrorValue(err)))
	}
	return p.foregroundExecutor.Spawn(func(s *scheduler.Scope) (gpui.Option[gpui.ClipboardItem], error) {
		items, err := awaitPromise(s, read)
		if err != nil {
			return gpui.None[gpui.ClipboardItem](), clipboardReadRejectionError(jsErrorValue(err))
		}
		var entries []gpui.ClipboardEntry
		sawUnsupportedType := false
		array := js.Global().Get("Array").Call("from", items)
		for i := range array.Length() {
			item := array.Index(i)
			types := js.Global().Get("Array").Call("from", jsGet(item, "types"))
			for j := range types.Length() {
				mimeType, ok := jsString(types.Index(j))
				if !ok {
					continue
				}
				if mimeType == "text/plain" {
					text, err := readClipboardItemText(s, item, mimeType)
					switch {
					case err != nil:
						logClipboardEntryError(mimeType, err)
					case text != "":
						entries = append(entries, gpui.ClipboardEntryString{ClipboardString: gpui.NewClipboardString(text)})
					}
				} else if format, ok := gpui.ImageFormatFromMimeType(mimeType); ok {
					bytes, err := readClipboardItemBytes(s, item, mimeType)
					if err != nil {
						logClipboardEntryError(mimeType, err)
					} else {
						entries = append(entries, gpui.ClipboardEntryImage{Image: gpui.ImageFromBytes(format, bytes)})
					}
				} else {
					sawUnsupportedType = true
				}
			}
		}
		switch {
		case len(entries) > 0:
			return gpui.Some(gpui.ClipboardItem{Entries: entries}), nil
		case sawUnsupportedType:
			return gpui.None[gpui.ClipboardItem](), gpui.ClipboardReadErrorUnsupportedContent()
		}
		return gpui.None[gpui.ClipboardItem](), nil
	})
}

// WriteToClipboard writes the item's text through navigator.clipboard.writeText.
func (p *WebPlatform) WriteToClipboard(item gpui.ClipboardItem) {
	text, ok := item.Text()
	window := js.Global().Get("window")
	if !ok || jsIsNullish(window) {
		return
	}
	_, _ = jsCall(jsGet(jsGet(window, "navigator"), "clipboard"), "writeText", text)
}

// WriteCredentials fails: credential storage is not available on the web.
func (p *WebPlatform) WriteCredentials(url string, username string, password []byte) *gpui.Task[struct{}] {
	return scheduler.ReadyTaskErr[struct{}](errors.New("credential storage is not available on the web"))
}

// ReadCredentials returns no credentials.
func (p *WebPlatform) ReadCredentials(url string) *gpui.Task[gpui.Option[gpui.Credentials]] {
	return scheduler.ReadyTask(gpui.None[gpui.Credentials]())
}

// DeleteCredentials fails: credential storage is not available on the web.
func (p *WebPlatform) DeleteCredentials(url string) *gpui.Task[struct{}] {
	return scheduler.ReadyTaskErr[struct{}](errors.New("credential storage is not available on the web"))
}

// KeyboardLayout returns the US layout.
func (p *WebPlatform) KeyboardLayout() gpui.PlatformKeyboardLayout {
	return WebKeyboardLayout{}
}

// KeyboardMapper returns the identity mapper.
func (p *WebPlatform) KeyboardMapper() gpui.PlatformKeyboardMapper {
	return gpui.DummyKeyboardMapper{}
}

// OnKeyboardLayoutChange registers the keyboard layout callback.
func (p *WebPlatform) OnKeyboardLayoutChange(callback func()) {
	p.callbacks.keyboardLayoutChange = gpui.Some(callback)
}

// Close removes the cursor-restore listeners; it is upstream's drop of the platform.
func (p *WebPlatform) Close() {
	for _, handle := range p.cursorRestoreListeners {
		handle.Close()
	}
	p.cursorRestoreListeners = nil
}

func jsErrorValue(err error) js.Value {
	var rejection jsRejection
	if errors.As(err, &rejection) {
		return rejection.value
	}
	var jsErr js.Error
	if errors.As(err, &jsErr) {
		return jsErr.Value
	}
	return js.ValueOf(err.Error())
}

func clipboardReadRejectionError(error js.Value) gpui.ClipboardReadError {
	name, ok := jsErrorName(error)
	if ok && (name == "NotAllowedError" || name == "SecurityError") {
		return gpui.ClipboardReadErrorDenied(jsErrorMessage(error))
	}
	return gpui.ClipboardReadErrorUnsupportedContent()
}

func readClipboardItemBlob(s *scheduler.Scope, item js.Value, mimeType string) (js.Value, error) {
	promise, err := jsCall(item, "getType", mimeType)
	if err != nil {
		return js.Undefined(), err
	}
	return awaitPromise(s, promise)
}

func readClipboardItemText(s *scheduler.Scope, item js.Value, mimeType string) (string, error) {
	blob, err := readClipboardItemBlob(s, item, mimeType)
	if err != nil {
		return "", err
	}
	promise, err := jsCall(blob, "text")
	if err != nil {
		return "", err
	}
	text, err := awaitPromise(s, promise)
	if err != nil {
		return "", err
	}
	str, ok := jsString(text)
	if !ok {
		return "", jsRejection{js.ValueOf("blob text is not a string")}
	}
	return str, nil
}

func readClipboardItemBytes(s *scheduler.Scope, item js.Value, mimeType string) ([]byte, error) {
	blob, err := readClipboardItemBlob(s, item, mimeType)
	if err != nil {
		return nil, err
	}
	return readBlobBytes(s, blob)
}

func readBlobBytes(s *scheduler.Scope, blob js.Value) ([]byte, error) {
	promise, err := jsCall(blob, "arrayBuffer")
	if err != nil {
		return nil, err
	}
	buffer, err := awaitPromise(s, promise)
	if err != nil {
		return nil, err
	}
	array := js.Global().Get("Uint8Array").New(buffer)
	bytes := make([]byte, array.Length())
	js.CopyBytesToGo(bytes, array)
	return bytes, nil
}

func jsErrorName(error js.Value) (string, bool) {
	return jsString(jsGet(error, "name"))
}

func jsErrorMessage(error js.Value) string {
	if message, ok := jsString(jsGet(error, "message")); ok {
		return message
	}
	if message, ok := jsString(error); ok {
		return message
	}
	return "unknown browser error"
}

func logClipboardEntryError(mimeType string, err error) {
	slog.Warn(fmt.Sprintf("failed to read clipboard entry with type %s: %s", mimeType, jsErrorMessage(jsErrorValue(err))))
}

func cursorRestoreListeners(browserWindow js.Value, cursorVisible *bool, lastCursorCSS *string) []*EventListenerHandle {
	var handles []*EventListenerHandle
	document := jsGet(browserWindow, "document")
	if jsIsNullish(document) {
		return handles
	}

	addListener := func(target js.Value, eventName string) {
		handles = append(handles, eventListenerHandleAdd(target, eventName, func(js.Value) {
			wasVisible := *cursorVisible
			*cursorVisible = true
			if !wasVisible {
				setBodyCursor(browserWindow, *lastCursorCSS)
			}
		}))
	}

	addListener(document, "mousemove")
	addListener(document, "mouseenter")
	addListener(browserWindow, "blur")
	addListener(document, "visibilitychange")

	return handles
}

func showGraphicsUnavailableMessage(browserWindow js.Value, err error) {
	document := jsGet(browserWindow, "document")
	if jsIsNullish(document) {
		return
	}
	body := jsGet(document, "body")
	if jsIsNullish(body) {
		return
	}
	message, callErr := jsCall(document, "createElement", "p")
	if callErr != nil {
		return
	}
	message.Set("textContent", fmt.Sprintf("Failed to initialize browser graphics: %v", err))
	_, _ = jsCall(body, "appendChild", message)
}

func setBodyCursor(browserWindow js.Value, cssCursor string) {
	body := jsGet(jsGet(browserWindow, "document"), "body")
	if jsIsNullish(body) {
		return
	}
	if _, err := jsCall(jsGet(body, "style"), "setProperty", "cursor", cssCursor); err != nil {
		slog.Warn(fmt.Sprintf("Failed to set cursor style: %s", jsErrorDebug(err)))
	}
}
