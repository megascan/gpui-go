//go:build ios

package ios

import (
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"weak"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

// IosPlatform is the iOS implementation of gpui.Platform. Close is its drop.
type IosPlatform struct {
	gpui.PlatformDefaults
	state *iosPlatformState
}

var _ gpui.Platform = (*IosPlatform)(nil)

type iosPlatformState struct {
	application              *IosApplicationState
	backgroundExecutor       gpui.BackgroundExecutor
	foregroundExecutor       gpui.ForegroundExecutor
	textSystem               gpui.PlatformTextSystem
	thermalObserver          *apple.ProcessInfoObserver
	powerModeObserver        *apple.ProcessInfoObserver
	displayObservers         []*objc.Retained
	displayTask              *gpui.Task[struct{}]
	displaysChangedCallback  callbackSlot[func()]
	closed                   bool
	notificationCenterHandle *objc.Retained
}

// NewIosPlatform creates the iOS platform.
func NewIosPlatform() *IosPlatform {
	if err := uikit.Load(); err != nil {
		panic(err)
	}
	dispatcher := apple.NewAppleDispatcher()
	textSystem := apple.NewAppleTextSystem()
	state := &iosPlatformState{
		application:        &IosApplicationState{},
		backgroundExecutor: gpui.NewBackgroundExecutor(dispatcher),
		foregroundExecutor: gpui.NewForegroundExecutor(dispatcher),
		textSystem:         textSystem,
	}
	sender, receiver := scheduler.Chan[struct{}](0)
	block := objc.NewBlock(func(_ objc.Block, _ objc.ID) {
		if sender.TrySend(struct{}{}) != nil {
			slog.Debug("display observer stopped before notification delivery")
		}
	})
	center := objc.RetainObject(objc.SendClass[objc.ID](objc.GetClass("NSNotificationCenter"), objc.Sel("defaultCenter")))
	state.notificationCenterHandle = center
	for _, name := range []objc.ID{
		uikit.UIScreenDidConnectNotification(),
		uikit.UIScreenDidDisconnectNotification(),
		uikit.UIScreenModeDidChangeNotification(),
	} {
		observer := objc.Send[objc.ID](center.ID(), objc.Sel("addObserverForName:object:queue:usingBlock:"), name, objc.Nil, objc.Nil, block)
		state.displayObservers = append(state.displayObservers, objc.RetainObject(observer))
	}
	block.Release()
	weakState := weak.Make(state)
	state.displayTask = state.foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer receiver.Close()
		for {
			if _, ok := receiver.Recv(s); !ok {
				break
			}
			if state := weakState.Value(); state != nil {
				state.displaysChangedCallback.with(func(callback *func()) { (*callback)() })
			}
		}
		return struct{}{}, nil
	})
	p := &IosPlatform{state: state}
	gpui.BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

func (p *IosPlatform) rootViewController() (uikit.UIViewController, bool) {
	var window uikit.UIWindow
	if scene, ok := p.state.application.windowScene(); ok {
		windows := scene.Windows()
		for _, candidate := range windows {
			if candidate.IsKeyWindow() {
				window = candidate
				break
			}
		}
		if window == 0 && len(windows) > 0 {
			window = windows[0]
		}
	} else {
		mustMainThread()
		window = uikit.UIApplicationSharedApplication().KeyWindow()
	}
	if window == 0 {
		return 0, false
	}
	controller := window.RootViewController()
	return controller, controller != 0
}

func (p *IosPlatform) presentedViewController() (uikit.UIViewController, bool) {
	viewController, ok := p.rootViewController()
	if !ok {
		return 0, false
	}
	for {
		presented := viewController.PresentedViewController()
		if presented == 0 {
			return viewController, true
		}
		viewController = presented
	}
}

func (p *IosPlatform) dismissPresentedBrowser() {
	viewController, ok := p.rootViewController()
	if !ok {
		return
	}
	for {
		presented := viewController.PresentedViewController()
		if presented == 0 {
			return
		}
		if uikit.IsKindOfClass(presented.ID(), "SFSafariViewController") {
			presented.DismissViewControllerAnimatedCompletion(true, objc.Nil)
			return
		}
		viewController = presented
	}
}

func openURLWithSystem(url objc.ID) {
	mustMainThread()
	options := objc.SendClass[objc.ID](objc.GetClass("NSDictionary"), objc.Sel("dictionary"))
	uikit.UIApplicationSharedApplication().OpenURLOptionsCompletionHandler(url, options, objc.Nil)
}

type iosKeyboardLayout struct{}

func (iosKeyboardLayout) ID() string   { return "ios-default" }
func (iosKeyboardLayout) Name() string { return "iOS Default" }

// PlatformInfo reports iOS running natively.
func (p *IosPlatform) PlatformInfo() gpui.PlatformInfo {
	return gpui.PlatformInfo{Os: gpui.OperatingSystemIos, Target: gpui.PlatformTargetNative}
}

// OnDisplaysChanged registers the callback run after a screen connects, disconnects or changes
// mode.
func (p *IosPlatform) OnDisplaysChanged(callback func()) {
	p.state.displaysChangedCallback.set(callback)
}

// BackgroundExecutor returns the executor for background work.
func (p *IosPlatform) BackgroundExecutor() gpui.BackgroundExecutor {
	return p.state.backgroundExecutor
}

// ForegroundExecutor returns the executor for main-thread work.
func (p *IosPlatform) ForegroundExecutor() gpui.ForegroundExecutor {
	return p.state.foregroundExecutor
}

// TextSystem returns the Core Text text system.
func (p *IosPlatform) TextSystem() gpui.PlatformTextSystem {
	return p.state.textSystem
}

// Run enters UIApplicationMain; onFinishLaunching runs once a scene can host a window.
func (p *IosPlatform) Run(onFinishLaunching func()) {
	run(p.state.application, onFinishLaunching)
}

// Quit logs a warning: iOS apps cannot quit themselves.
func (p *IosPlatform) Quit() {
	slog.Warn("iOS apps cannot programmatically quit")
}

// Restart logs a warning: iOS apps cannot restart themselves.
func (p *IosPlatform) Restart(binaryPath gpui.Option[string], arguments []string) {
	slog.Warn("iOS apps cannot restart themselves")
}

// Activate dismisses a browser the app presented.
func (p *IosPlatform) Activate(ignoringOtherApps bool) {
	p.dismissPresentedBrowser()
}

// Hide does nothing on iOS.
func (p *IosPlatform) Hide() {}

// HideOtherApps does nothing on iOS.
func (p *IosPlatform) HideOtherApps() {}

// UnhideOtherApps does nothing on iOS.
func (p *IosPlatform) UnhideOtherApps() {}

// Displays returns every connected screen.
func (p *IosPlatform) Displays() []gpui.PlatformDisplay {
	all := IosDisplayAll()
	displays := make([]gpui.PlatformDisplay, len(all))
	for i, display := range all {
		displays[i] = display
	}
	return displays
}

// PrimaryDisplay returns the main screen.
func (p *IosPlatform) PrimaryDisplay() gpui.PlatformDisplay {
	return IosDisplayMain()
}

// ActiveWindow reports no window: the key window is not tracked.
func (p *IosPlatform) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	return gpui.AnyWindowHandle{}, false
}

// OpenWindow creates a UIKit window and registers it with the application.
func (p *IosPlatform) OpenWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams) (gpui.PlatformWindow, error) {
	window, err := newIosWindow(handle, options, p.state)
	if err != nil {
		return nil, err
	}
	window.register()
	return window, nil
}

// WindowAppearance returns the root view controller's interface style.
func (p *IosPlatform) WindowAppearance() gpui.WindowAppearance {
	controller, ok := p.rootViewController()
	if !ok {
		return gpui.WindowAppearanceLight
	}
	if controller.UserInterfaceStyle() == uikit.UIUserInterfaceStyleDark {
		return gpui.WindowAppearanceDark
	}
	return gpui.WindowAppearanceLight
}

// OpenURL presents http(s) URLs in a Safari view controller and hands others to the system.
func (p *IosPlatform) OpenURL(url string) {
	s := objc.NSStringFromGo(url)
	defer objc.Release(s)
	nativeURL := objc.SendClass[objc.ID](objc.GetClass("NSURL"), objc.Sel("URLWithString:"), s)
	if nativeURL == objc.Nil {
		slog.Error("GPUI iOS: Could not parse URL: " + url)
		return
	}
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		if viewController, ok := p.presentedViewController(); ok {
			if safariClass, ok := safariViewControllerClass(); ok {
				browser := objc.Send[objc.ID](objc.Alloc(safariClass), objc.Sel("initWithURL:"), nativeURL)
				viewController.PresentViewControllerAnimatedCompletion(browser, true, objc.Nil)
				objc.Release(browser)
				return
			}
		}
	}
	openURLWithSystem(nativeURL)
}

// OnOpenURLs registers the callback for URLs the app is asked to open.
func (p *IosPlatform) OnOpenURLs(callback func(urls []string)) {
	p.state.application.openURLs.set(callback)
}

// RegisterURLScheme succeeds immediately: iOS URL schemes are declared in Info.plist.
func (p *IosPlatform) RegisterURLScheme(url string) *gpui.Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// PromptForPaths fails: the file picker is not implemented on iOS.
func (p *IosPlatform) PromptForPaths(options gpui.PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[[]string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[gpui.Option[[]string]]]()
	if tx.Send(scheduler.Result[gpui.Option[[]string]]{Err: errors.New("File picker not yet implemented for iOS")}) != nil {
		slog.Debug("GPUI iOS: File picker receiver was dropped")
	}
	return rx
}

// PromptForNewPath fails: the save dialog is not implemented on iOS.
func (p *IosPlatform) PromptForNewPath(directory string, suggestedName gpui.Option[string]) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[gpui.Option[string]]]()
	if tx.Send(scheduler.Result[gpui.Option[string]]{Err: errors.New("Save dialog not yet implemented for iOS")}) != nil {
		slog.Debug("GPUI iOS: Save dialog receiver was dropped")
	}
	return rx
}

// CanSelectMixedFilesAndDirs reports false.
func (p *IosPlatform) CanSelectMixedFilesAndDirs() bool { return false }

// RevealPath does nothing: iOS has no file manager like Finder.
func (p *IosPlatform) RevealPath(path string) {}

// OpenWithSystem does nothing on iOS.
func (p *IosPlatform) OpenWithSystem(path string) {}

// OnQuit registers the callback run when UIKit terminates the app; it cannot veto termination.
func (p *IosPlatform) OnQuit(callback func() bool) {
	p.state.application.quit.set(func() { callback() })
}

// OnReopen does nothing: iOS reopens apps through the scene lifecycle.
func (p *IosPlatform) OnReopen(callback func()) {}

// OnSystemWake does nothing on iOS.
func (p *IosPlatform) OnSystemWake(callback func()) {}

// OnSystemSleep does nothing on iOS.
func (p *IosPlatform) OnSystemSleep(callback func()) {}

// PreventIdleSleep fails: idle sleep prevention is not implemented on iOS.
func (p *IosPlatform) PreventIdleSleep(reason string) *gpui.Task[*gpui.ActivityGuard] {
	return scheduler.ReadyTaskErr[*gpui.ActivityGuard](errors.New("Idle sleep prevention is not implemented for iOS"))
}

// OnAppLifecycle registers the callback run on scene lifecycle changes.
func (p *IosPlatform) OnAppLifecycle(callback func(phase gpui.AppLifecyclePhase)) {
	p.state.application.appLifecycle.set(callback)
}

// OnMemoryWarning registers the callback run when UIKit reports memory pressure.
func (p *IosPlatform) OnMemoryWarning(callback func()) {
	p.state.application.memoryWarning.set(callback)
}

// SetMenus does nothing: iOS has no menu bar.
func (p *IosPlatform) SetMenus(menus []gpui.Menu, keymap *gpui.Keymap) {}

// SetDockMenu does nothing: iOS has no dock menu.
func (p *IosPlatform) SetDockMenu(menu []gpui.MenuItem, keymap *gpui.Keymap) {}

// OnAppMenuAction does nothing on iOS.
func (p *IosPlatform) OnAppMenuAction(callback func(action gpui.Action)) {}

// OnWillOpenAppMenu does nothing on iOS.
func (p *IosPlatform) OnWillOpenAppMenu(callback func()) {}

// OnValidateAppMenuCommand does nothing on iOS.
func (p *IosPlatform) OnValidateAppMenuCommand(callback func(action gpui.Action) bool) {}

// AppPath returns the main bundle's path.
func (p *IosPlatform) AppPath() (string, error) {
	bundle := objc.SendClass[objc.ID](objc.GetClass("NSBundle"), objc.Sel("mainBundle"))
	return objc.GoString(objc.Send[objc.ID](bundle, objc.Sel("bundlePath"))), nil
}

// PathForAuxiliaryExecutable returns name inside the app bundle.
func (p *IosPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	appPath, err := p.AppPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(appPath, name), nil
}

// SetCursorStyle does nothing: iOS has no visible cursor.
func (p *IosPlatform) SetCursorStyle(style gpui.CursorStyle) {}

// HideCursorUntilMouseMoves does nothing on iOS.
func (p *IosPlatform) HideCursorUntilMouseMoves() {}

// IsCursorVisible reports false.
func (p *IosPlatform) IsCursorVisible() bool { return false }

// ShouldAutoHideScrollbars reports true: iOS always auto-hides scrollbars.
func (p *IosPlatform) ShouldAutoHideScrollbars() bool { return true }

// WriteToClipboard writes item's text to the general pasteboard.
func (p *IosPlatform) WriteToClipboard(item gpui.ClipboardItem) {
	pasteboard := uikit.UIPasteboardGeneralPasteboard()
	if text, ok := item.Text(); ok {
		pasteboard.SetString(text)
	}
}

// ReadFromClipboard reads the general pasteboard's string.
func (p *IosPlatform) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	text, ok := uikit.UIPasteboardGeneralPasteboard().String()
	if !ok {
		return gpui.ClipboardItem{}, false
	}
	return gpui.NewClipboardItemString(text), true
}

// WriteCredentials stores a password for url in the keychain.
func (p *IosPlatform) WriteCredentials(url string, username string, password []byte) *gpui.Task[struct{}] {
	password = append([]byte(nil), password...)
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		return struct{}{}, apple.WriteCredentials(url, username, password)
	})
}

// ReadCredentials reads the credentials stored for url.
func (p *IosPlatform) ReadCredentials(url string) *gpui.Task[gpui.Option[gpui.Credentials]] {
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (gpui.Option[gpui.Credentials], error) {
		username, password, ok, err := apple.ReadCredentials(url)
		if err != nil {
			return gpui.None[gpui.Credentials](), err
		}
		if !ok {
			return gpui.None[gpui.Credentials](), nil
		}
		return gpui.Some(gpui.Credentials{Username: username, Password: password}), nil
	})
}

// DeleteCredentials deletes the credentials stored for url.
func (p *IosPlatform) DeleteCredentials(url string) *gpui.Task[struct{}] {
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		return struct{}{}, apple.DeleteCredentials(url)
	})
}

// OnKeyboardLayoutChange does nothing on iOS.
func (p *IosPlatform) OnKeyboardLayoutChange(callback func()) {}

// ThermalState returns the system's thermal state.
func (p *IosPlatform) ThermalState() gpui.ThermalState {
	return apple.ThermalState()
}

// OnThermalStateChange registers the callback run when the thermal state changes.
func (p *IosPlatform) OnThermalStateChange(callback func()) {
	executor := p.state.foregroundExecutor
	observer := apple.ObserveThermalState(&executor, callback)
	previous := p.state.thermalObserver
	p.state.thermalObserver = observer
	if previous != nil {
		previous.Close()
	}
}

// PowerMode returns the system's power mode.
func (p *IosPlatform) PowerMode() gpui.PowerMode {
	return apple.PowerMode()
}

// OnPowerModeChange registers the callback run when Low Power Mode changes.
func (p *IosPlatform) OnPowerModeChange(callback func()) {
	executor := p.state.foregroundExecutor
	observer := apple.ObservePowerMode(&executor, callback)
	previous := p.state.powerModeObserver
	p.state.powerModeObserver = observer
	if previous != nil {
		previous.Close()
	}
}

// KeyboardLayout returns the fixed iOS keyboard layout.
func (p *IosPlatform) KeyboardLayout() gpui.PlatformKeyboardLayout {
	return iosKeyboardLayout{}
}

// KeyboardMapper returns a mapper that maps every keystroke to itself.
func (p *IosPlatform) KeyboardMapper() gpui.PlatformKeyboardMapper {
	return gpui.DummyKeyboardMapper{}
}

// Close removes the screen observers, cancels the display task, closes the process-info observers
// and stops the display link. It is the platform state's drop and is idempotent.
func (p *IosPlatform) Close() {
	state := p.state
	if state.closed {
		return
	}
	state.closed = true
	for _, observer := range state.displayObservers {
		objc.SendVoid(state.notificationCenterHandle.ID(), objc.Sel("removeObserver:"), observer.ID())
		observer.Release()
	}
	state.displayObservers = nil
	state.notificationCenterHandle.Release()
	state.displayTask.Cancel()
	if state.thermalObserver != nil {
		state.thermalObserver.Close()
	}
	if state.powerModeObserver != nil {
		state.powerModeObserver.Close()
	}
	state.application.Close()
}

const safariServicesPath = "/System/Library/Frameworks/SafariServices.framework/SafariServices"

var (
	safariOnce  sync.Once
	safariClass objc.Class
)

func safariViewControllerClass() (objc.Class, bool) {
	safariOnce.Do(func() {
		if _, err := purego.Dlopen(safariServicesPath, purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
			slog.Error("GPUI iOS: " + err.Error())
			return
		}
		safariClass = objc.GetClass("SFSafariViewController")
	})
	return safariClass, safariClass != 0
}
