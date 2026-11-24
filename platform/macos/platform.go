//go:build darwin && !ios

package macos

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/appkit"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/gcd"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/platform/apple"
	"github.com/megascan/gpui-go/scheduler"
)

const macPlatformIvar = "platform"

var (
	buildClassesOnce sync.Once
	appClass         objc.Class
	appDelegateClass objc.Class

	macPlatformIvarsMu sync.Mutex
	macPlatformIvars   = map[objc.ID]*MacPlatform{}
)

func buildClasses() {
	buildClassesOnce.Do(func() {
		if err := appkit.Load(); err != nil {
			panic(err)
		}
		var err error
		appClass, err = objc.RegisterClass("GPUIApplication", objc.GetClass("NSApplication"), nil, nil, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to register GPUIApplication: %v", err))
		}
		notification := func(fn func(this objc.ID)) func(objc.ID, objc.SEL, objc.ID) {
			return func(this objc.ID, _ objc.SEL, _ objc.ID) { fn(this) }
		}
		handleMenuItemDef := func(sel string) objc.MethodDef {
			return objc.MethodDef{Cmd: objc.Sel(sel), Fn: handleMenuItem}
		}
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("applicationWillFinishLaunching:"), Fn: notification(willFinishLaunching)},
			{Cmd: objc.Sel("applicationDidFinishLaunching:"), Fn: notification(didFinishLaunching)},
			{Cmd: objc.Sel("applicationShouldHandleReopen:hasVisibleWindows:"), Fn: shouldHandleReopen},
			{Cmd: objc.Sel("applicationWillTerminate:"), Fn: notification(willTerminate)},
			handleMenuItemDef("handleGPUIMenuItem:"),
			handleMenuItemDef("cut:"),
			handleMenuItemDef("copy:"),
			handleMenuItemDef("paste:"),
			handleMenuItemDef("selectAll:"),
			handleMenuItemDef("undo:"),
			handleMenuItemDef("redo:"),
			{Cmd: objc.Sel("validateMenuItem:"), Fn: validateMenuItem},
			{Cmd: objc.Sel("menuWillOpen:"), Fn: notification(menuWillOpen)},
			{Cmd: objc.Sel("applicationDockMenu:"), Fn: handleDockMenu},
			{Cmd: objc.Sel("application:openURLs:"), Fn: openURLs},
			{Cmd: objc.Sel("onKeyboardLayoutChange:"), Fn: notification(onKeyboardLayoutChange)},
			{Cmd: objc.Sel("onScreenParametersChange:"), Fn: notification(onScreenParametersChange)},
			{Cmd: objc.Sel("onSystemSleep:"), Fn: notification(onSystemSleep)},
			{Cmd: objc.Sel("onSystemWake:"), Fn: notification(onSystemWake)},
		}
		appDelegateClass, err = objc.RegisterClass("GPUIApplicationDelegate", objc.GetClass("NSResponder"), nil, nil, methods)
		if err != nil {
			panic(fmt.Sprintf("failed to register GPUIApplicationDelegate: %v", err))
		}
	})
}

// MacPlatform is the macOS implementation of gpui.Platform.
type MacPlatform struct {
	gpui.PlatformDefaults
	mu     sync.Mutex
	state  macPlatformState
	marker objc.MainThreadMarker
}

var _ gpui.Platform = (*MacPlatform)(nil)

type macPlatformState struct {
	backgroundExecutor             gpui.BackgroundExecutor
	foregroundExecutor             gpui.ForegroundExecutor
	textSystem                     gpui.PlatformTextSystem
	rendererContext                rendererContext
	headless                       bool
	activationPolicy               gpui.ActivationPolicy
	applicationCreated             bool
	generalPasteboard              *Pasteboard
	findPasteboard                 *Pasteboard
	reopen                         func()
	onKeyboardLayoutChange         func()
	thermalObserver                *apple.ProcessInfoObserver
	powerModeObserver              *apple.ProcessInfoObserver
	onDisplaysChanged              func()
	onSystemSleep                  func()
	onSystemWake                   func()
	systemPowerObserversRegistered bool
	quit                           func() bool
	menuCommand                    func(action gpui.Action)
	validateMenuCommand            func(action gpui.Action) bool
	willOpenMenu                   func()
	menuActions                    []gpui.Action
	openURLs                       func(urls []string)
	finishLaunching                func()
	dockMenu                       *objc.Retained
	menus                          gpuiutil.Option[[]gpui.OwnedMenu]
	keyboardMapper                 *MacKeyboardMapper
	cursorVisible                  *atomic.Bool
	systemNotifications            *systemNotificationState
}

func activationPolicyFor(request gpui.WindowingRequest) gpui.ActivationPolicy {
	switch request.(type) {
	case gpui.WindowingRequestHeadless:
		return gpui.ActivationPolicyAccessory
	default:
		return gpui.ActivationPolicyRegular
	}
}

func nativeActivationPolicy(policy gpui.ActivationPolicy) appkit.NSApplicationActivationPolicy {
	switch policy {
	case gpui.ActivationPolicyAccessory:
		return appkit.NSApplicationActivationPolicyAccessory
	default:
		return appkit.NSApplicationActivationPolicyRegular
	}
}

func sharedApplication() objc.ID {
	return objc.SendClass[objc.ID](appClass, objc.Sel("sharedApplication"))
}

// NewMacPlatform creates the macOS platform; it must be called on the main thread.
func NewMacPlatform(headless bool) *MacPlatform {
	marker, ok := apple.MainThreadMarker()
	if !ok {
		panic("Mac platform not created on main thread")
	}
	buildClasses()
	dispatcher := apple.NewAppleDispatcher()
	textSystem := apple.NewAppleTextSystem()
	keyboardLayout := NewMacKeyboardLayout()
	keyboardMapper := NewMacKeyboardMapper(keyboardLayout.ID())
	cursorVisible := &atomic.Bool{}
	cursorVisible.Store(true)
	generalPasteboard, _ := PasteboardGeneral()
	findPasteboard, _ := PasteboardFind()
	p := &MacPlatform{
		state: macPlatformState{
			headless:            headless,
			activationPolicy:    gpui.ActivationPolicyRegular,
			textSystem:          textSystem,
			backgroundExecutor:  gpui.NewBackgroundExecutor(dispatcher),
			foregroundExecutor:  gpui.NewForegroundExecutor(dispatcher),
			rendererContext:     newRendererContext(),
			generalPasteboard:   generalPasteboard,
			findPasteboard:      findPasteboard,
			keyboardMapper:      keyboardMapper,
			cursorVisible:       cursorVisible,
			systemNotifications: newSystemNotificationState(),
		},
		marker: marker,
	}
	gpui.BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

func (p *MacPlatform) createMenuBar(menus []gpui.Menu, delegate objc.ID, actions *[]gpui.Action, keymap *gpui.Keymap) objc.ID {
	applicationMenu := appkit.NewNSMenu("")
	applicationMenu.SetDelegate(delegate)
	for _, menuConfig := range menus {
		menu := appkit.NewNSMenu("")
		menuTitle := objc.NSStringFromGo(string(menuConfig.Name))
		objc.SendVoid(menu.ID(), objc.Sel("setTitle:"), menuTitle)
		menu.SetDelegate(delegate)
		for _, itemConfig := range menuConfig.Items {
			item := p.createMenuItem(itemConfig, delegate, actions, keymap)
			menu.AddItem(appkit.NSMenuItem(item))
			objc.Release(item)
		}
		menuItem := appkit.NewNSMenuItem("", 0, "")
		objc.SendVoid(menuItem.ID(), objc.Sel("setTitle:"), menuTitle)
		menuItem.SetSubmenu(menu)
		applicationMenu.AddItem(menuItem)
		objc.Release(menuItem.ID())
		objc.Release(menuTitle)
		if menuConfig.Name == "Window" {
			objc.SendVoid(sharedApplication(), objc.Sel("setWindowsMenu:"), menu.ID())
		}
		objc.Release(menu.ID())
	}
	return applicationMenu.ID()
}

func (p *MacPlatform) createDockMenu(menuItems []gpui.MenuItem, delegate objc.ID, actions *[]gpui.Action, keymap *gpui.Keymap) objc.ID {
	dockMenu := appkit.NewNSMenu("")
	dockMenu.SetDelegate(delegate)
	for _, itemConfig := range menuItems {
		item := p.createMenuItem(itemConfig, delegate, actions, keymap)
		dockMenu.AddItem(appkit.NSMenuItem(item))
		objc.Release(item)
	}
	return dockMenu.ID()
}

var defaultMenuContext = sync.OnceValue(func() []gpui.KeyContext {
	workspaceContext := gpui.NewKeyContextWithDefaults(gpui.OperatingSystemMac)
	workspaceContext.Add("Workspace")
	paneContext := gpui.NewKeyContextWithDefaults(gpui.OperatingSystemMac)
	paneContext.Add("Pane")
	editorContext := gpui.NewKeyContextWithDefaults(gpui.OperatingSystemMac)
	editorContext.Add("Editor")
	paneContext.Extend(&editorContext)
	workspaceContext.Extend(&paneContext)
	return []gpui.KeyContext{workspaceContext}
})

func (p *MacPlatform) createMenuItem(item gpui.MenuItem, delegate objc.ID, actions *[]gpui.Action, keymap *gpui.Keymap) objc.ID {
	switch item := item.(type) {
	case gpui.MenuItemSeparator:
		return objc.Retain(appkit.NSMenuItemSeparator().ID())
	case gpui.MenuItemAction:
		var keystrokes []gpui.KeybindingKeystroke
		hasKeystrokes := false
		bindings := keymap.BindingsForAction(item.Action)
		for _, binding := range bindings {
			predicate := binding.Predicate()
			if predicate == nil || predicate.Eval(defaultMenuContext()) {
				keystrokes, hasKeystrokes = binding.Keystrokes(), true
				break
			}
		}
		if !hasKeystrokes && len(bindings) > 0 {
			keystrokes, hasKeystrokes = bindings[0].Keystrokes(), true
		}

		selector := objc.Sel("handleGPUIMenuItem:")
		if osAction, ok := item.OsAction.Get(); ok {
			switch osAction {
			case gpui.OsActionCut:
				selector = objc.Sel("cut:")
			case gpui.OsActionCopy:
				selector = objc.Sel("copy:")
			case gpui.OsActionPaste:
				selector = objc.Sel("paste:")
			case gpui.OsActionSelectAll:
				selector = objc.Sel("selectAll:")
			}
		}

		var menuItem appkit.NSMenuItem
		if hasKeystrokes && len(keystrokes) == 1 {
			keystroke := keystrokes[0]
			var mask appkit.NSEventModifierFlags
			modifiers := keystroke.Modifiers()
			for _, m := range []struct {
				modifier bool
				flag     appkit.NSEventModifierFlags
			}{
				{modifiers.Platform, appkit.NSEventModifierFlagCommand},
				{modifiers.Control, appkit.NSEventModifierFlagControl},
				{modifiers.Alt, appkit.NSEventModifierFlagOption},
				{modifiers.Shift, appkit.NSEventModifierFlagShift},
			} {
				if m.modifier {
					mask |= m.flag
				}
			}
			menuItem = appkit.NewNSMenuItem(string(item.Name), selector, keyToMenuEquivalent(keystroke.Key()))
			if macOSVersion().atLeast(12, 0, 0) {
				menuItem.SetAllowsAutomaticKeyEquivalentLocalization(false)
			}
			menuItem.SetKeyEquivalentModifierMask(mask)
		} else {
			menuItem = appkit.NewNSMenuItem(string(item.Name), selector, "")
		}

		if item.Checked {
			menuItem.SetState(appkit.NSControlStateValueOn)
		}
		menuItem.SetEnabled(!item.Disabled)

		menuItem.SetTag(len(*actions))
		*actions = append(*actions, item.Action.BoxedClone())
		return menuItem.ID()
	case gpui.MenuItemSubmenu:
		menuItem := appkit.NewNSMenuItem("", 0, "")
		submenu := appkit.NewNSMenu("")
		submenu.SetDelegate(delegate)
		for _, subitem := range item.Items {
			child := p.createMenuItem(subitem, delegate, actions, keymap)
			submenu.AddItem(appkit.NSMenuItem(child))
			objc.Release(child)
		}
		menuItem.SetSubmenu(submenu)
		objc.Release(submenu.ID())
		menuItem.SetEnabled(!item.Disabled)
		setNSTitle(menuItem.ID(), string(item.Name))
		return menuItem.ID()
	case gpui.MenuItemSystemMenu:
		menuItem := appkit.NewNSMenuItem("", 0, "")
		submenu := appkit.NewNSMenu("")
		submenu.SetDelegate(delegate)
		menuItem.SetSubmenu(submenu)
		objc.Release(submenu.ID())
		setNSTitle(menuItem.ID(), string(item.Name))
		switch item.MenuType {
		case gpui.SystemMenuTypeServices:
			objc.SendVoid(sharedApplication(), objc.Sel("setServicesMenu:"), menuItem.ID())
		}
		return menuItem.ID()
	default:
		panic(fmt.Sprintf("macos: unknown menu item %T", item))
	}
}

func setNSTitle(object objc.ID, title string) {
	s := objc.NSStringFromGo(title)
	objc.SendVoid(object, objc.Sel("setTitle:"), s)
	objc.Release(s)
}

type osVersion struct {
	major, minor, patch uint64
}

func (v osVersion) atLeast(major, minor, patch uint64) bool {
	if v.major != major {
		return v.major > major
	}
	if v.minor != minor {
		return v.minor > minor
	}
	return v.patch >= patch
}

type nsOperatingSystemVersion struct {
	MajorVersion objc.NSInteger
	MinorVersion objc.NSInteger
	PatchVersion objc.NSInteger
}

var (
	processInfoMsgOnce               sync.Once
	msgSendOperatingSystemVersion    func(objc.ID, objc.SEL) nsOperatingSystemVersion
	msgSendIsOperatingSystemAtLeast  func(objc.ID, objc.SEL, nsOperatingSystemVersion) bool
	coreFoundationRunLoopOnce        sync.Once
	cfRunLoopRunFn                   func()
	coreFoundationRunLoopErr         error
	errAppNotRunningInsideABundle    = errors.New("app is not running inside a bundle")
	errAuxiliaryExecutableNotFound   = errors.New("resource not found")
	errRegisterURLSchemeNeedsMacOS12 = errors.New("macOS 12.0 or later is required to register URL schemes")
)

func bindProcessInfoMsgSend() {
	processInfoMsgOnce.Do(func() {
		objc.BindMsgSend(&msgSendOperatingSystemVersion)
		objc.BindMsgSend(&msgSendIsOperatingSystemAtLeast)
	})
}

func processInfo() objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSProcessInfo"), objc.Sel("processInfo"))
}

func macOSVersion() osVersion {
	bindProcessInfoMsgSend()
	version := msgSendOperatingSystemVersion(processInfo(), objc.Sel("operatingSystemVersion"))
	return osVersion{
		major: uint64(version.MajorVersion),
		minor: uint64(version.MinorVersion),
		patch: uint64(version.PatchVersion),
	}
}

func cfRunLoopRun() {
	coreFoundationRunLoopOnce.Do(func() {
		lib, err := purego.Dlopen(cf.CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			coreFoundationRunLoopErr = fmt.Errorf("macos: open %s: %w", cf.CoreFoundationPath, err)
			return
		}
		purego.RegisterLibFunc(&cfRunLoopRunFn, lib, "CFRunLoopRun")
	})
	if coreFoundationRunLoopErr != nil {
		panic(coreFoundationRunLoopErr)
	}
	cfRunLoopRunFn()
}

func (p *MacPlatform) ensureSystemPowerObservers() {
	p.mu.Lock()
	registered := p.state.systemPowerObserversRegistered
	p.mu.Unlock()
	if registered {
		return
	}
	delegate := appkit.NSApplication(sharedApplication()).Delegate()
	if delegate != objc.Nil {
		registerSystemPowerObservers(delegate)
		p.mu.Lock()
		p.state.systemPowerObserversRegistered = true
		p.mu.Unlock()
	}
}

// PlatformInfo reports macOS running natively.
func (p *MacPlatform) PlatformInfo() gpui.PlatformInfo {
	return gpui.PlatformInfo{Os: gpui.OperatingSystemMac, Target: gpui.PlatformTargetNative}
}

// BackgroundExecutor returns the executor for background work.
func (p *MacPlatform) BackgroundExecutor() gpui.BackgroundExecutor {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.backgroundExecutor
}

// ForegroundExecutor returns the executor for main-thread work.
func (p *MacPlatform) ForegroundExecutor() gpui.ForegroundExecutor {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.foregroundExecutor
}

// TextSystem returns the Core Text text system.
func (p *MacPlatform) TextSystem() gpui.PlatformTextSystem {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.textSystem
}

// Run runs the application until it terminates, calling onFinishLaunching once it has launched.
func (p *MacPlatform) Run(onFinishLaunching func()) {
	p.mu.Lock()
	if p.state.headless {
		p.mu.Unlock()
		onFinishLaunching()
		cfRunLoopRun()
	} else {
		p.state.finishLaunching = onFinishLaunching
		p.mu.Unlock()
	}

	app := sharedApplication()
	p.mu.Lock()
	p.state.applicationCreated = true
	policy := p.state.activationPolicy
	p.mu.Unlock()
	if policy == gpui.ActivationPolicyAccessory {
		appkit.NSApplication(app).SetActivationPolicy(nativeActivationPolicy(policy))
	}
	appDelegate := objc.SendClass[objc.ID](appDelegateClass, objc.Sel("new"))
	appkit.NSApplication(app).SetDelegate(appDelegate)

	setMacPlatformIvar(app, p)
	setMacPlatformIvar(appDelegate, p)

	objc.AutoreleasePool(func() {
		appkit.NSApplication(app).Run()
	})

	setMacPlatformIvar(app, nil)
	setMacPlatformIvar(appkit.NSApplication(app).Delegate(), nil)
	p.mu.Lock()
	p.state.applicationCreated = false
	p.mu.Unlock()
}

// Quit terminates the application asynchronously, from the main queue.
func (p *MacPlatform) Quit() {
	gcd.MainQueue().Async(func() {
		app := appkit.NSApplicationSharedApplication()
		app.Terminate(objc.Nil)
	})
}

const restartScript = `
            while kill -0 $0 2> /dev/null; do
                sleep 0.1
            done
            app_path="$1"
            shift
            if (($# > 0)); then
                open "$app_path" --args "$@"
            else
                open "$app_path"
            fi
        `

// Restart waits for this process to exit and then reopens the app or binaryPath.
func (p *MacPlatform) Restart(binaryPath gpui.Option[string], arguments []string) {
	appPID := strconv.Itoa(os.Getpid())
	appPath, ok := binaryPath.Get()
	if !ok {
		if path, err := p.AppPath(); err == nil && filepath.Ext(path) == ".app" {
			appPath, ok = path, true
		}
	}
	if !ok {
		exe, err := os.Executable()
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		appPath = exe
	}

	args := append([]string{"-c", restartScript, appPID, appPath}, arguments...)
	command := gpuiutil.NewStdCommand("/bin/bash", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		slog.Error(fmt.Sprintf("failed to spawn restart script: %v", err))
		return
	}
	p.Quit()
}

// Activate brings the application to the front.
func (p *MacPlatform) Activate(ignoringOtherApps bool) {
	appkit.NSApplicationSharedApplication().ActivateIgnoringOtherApps(ignoringOtherApps)
}

// SetActivationPolicy records policy and applies it once the application exists.
func (p *MacPlatform) SetActivationPolicy(policy gpui.ActivationPolicy) {
	p.mu.Lock()
	p.state.activationPolicy = policy
	shouldApply := p.state.applicationCreated && !p.state.headless
	p.mu.Unlock()
	if shouldApply {
		appkit.NSApplication(sharedApplication()).SetActivationPolicy(nativeActivationPolicy(policy))
	}
}

// SetInitialWindowing sets the activation policy that stands for request.
func (p *MacPlatform) SetInitialWindowing(request gpui.WindowingRequest) {
	p.SetActivationPolicy(activationPolicyFor(request))
}

// RequestWindowing switches between the accessory (headless) and regular activation policies.
func (p *MacPlatform) RequestWindowing(request gpui.WindowingRequest) *gpui.Task[struct{}] {
	p.mu.Lock()
	if p.state.headless {
		p.mu.Unlock()
		return scheduler.ReadyTaskErr[struct{}](errors.New("a platform created headless has no application to switch windowing modes"))
	}
	policy := activationPolicyFor(request)
	if p.state.activationPolicy == policy {
		p.mu.Unlock()
		if _, headless := request.(gpui.WindowingRequestHeadless); headless {
			return scheduler.ReadyTaskErr[struct{}](errors.New("already headless"))
		}
		return scheduler.ReadyTaskErr[struct{}](errors.New("already windowed"))
	}
	p.mu.Unlock()
	p.SetActivationPolicy(policy)
	return scheduler.ReadyTask(struct{}{})
}

// GraphicalEnvironment returns the session's environment while the app is windowed.
func (p *MacPlatform) GraphicalEnvironment() (gpui.GraphicalEnvironment, bool) {
	p.mu.Lock()
	windowed := !p.state.headless && p.state.activationPolicy == gpui.ActivationPolicyRegular
	p.mu.Unlock()
	if windowed {
		return gpui.GraphicalEnvironmentDetect(), true
	}
	return gpui.GraphicalEnvironment{}, false
}

// Hide hides the application.
func (p *MacPlatform) Hide() {
	appkit.NSApplicationSharedApplication().Hide(objc.Nil)
}

// HideOtherApps hides every other application.
func (p *MacPlatform) HideOtherApps() {
	appkit.NSApplicationSharedApplication().HideOtherApplications(objc.Nil)
}

// UnhideOtherApps shows every other application.
func (p *MacPlatform) UnhideOtherApps() {
	appkit.NSApplicationSharedApplication().UnhideAllApplications(objc.Nil)
}

// PrimaryDisplay returns the display with the menu bar.
func (p *MacPlatform) PrimaryDisplay() gpui.PlatformDisplay {
	return MacDisplayPrimary()
}

// Displays returns every active display.
func (p *MacPlatform) Displays() []gpui.PlatformDisplay {
	var displays []gpui.PlatformDisplay
	for screen := range MacDisplayAll() {
		displays = append(displays, screen)
	}
	return displays
}

// OnDisplaysChanged registers the callback run after the display configuration changes.
func (p *MacPlatform) OnDisplaysChanged(callback func()) {
	p.mu.Lock()
	p.state.onDisplaysChanged = callback
	p.mu.Unlock()
}

// IsScreenCaptureSupported reports whether ScreenCaptureKit is available (macOS 12.3).
func (p *MacPlatform) IsScreenCaptureSupported() bool {
	bindProcessInfoMsgSend()
	minVersion := nsOperatingSystemVersion{MajorVersion: 12, MinorVersion: 3, PatchVersion: 0}
	return msgSendIsOperatingSystemAtLeast(processInfo(), objc.Sel("isOperatingSystemAtLeastVersion:"), minVersion)
}

// ScreenCaptureSources returns the displays ScreenCaptureKit can capture.
func (p *MacPlatform) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	return getSources(p.marker)
}

// ActiveWindow returns the key window's handle.
func (p *MacPlatform) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	return MacWindowActiveWindow()
}

// WindowStack returns the windows ordered front to back.
func (p *MacPlatform) WindowStack() ([]gpui.AnyWindowHandle, bool) {
	return MacWindowOrderedWindows(), true
}

// OpenWindow opens a window; native popups are rejected so gpui falls back to in-window popovers.
func (p *MacPlatform) OpenWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams) (gpui.PlatformWindow, error) {
	if _, ok := options.Kind.(gpui.WindowKindAnchoredPopup); ok {
		return nil, gpui.PopupNotSupportedError{}
	}
	p.mu.Lock()
	cursorVisible := p.state.cursorVisible
	foregroundExecutor := p.state.foregroundExecutor
	backgroundExecutor := p.state.backgroundExecutor
	rendererContext := p.state.rendererContext
	p.mu.Unlock()
	return MacWindowOpen(handle, options, cursorVisible, foregroundExecutor, backgroundExecutor, rendererContext, p.marker)
}

// WindowAppearance returns the application's effective appearance.
func (p *MacPlatform) WindowAppearance() gpui.WindowAppearance {
	return windowAppearanceFromNative(appkit.NSApplicationSharedApplication().EffectiveAppearance())
}

// SetWindowAppearance overrides the app's appearance; None follows the system setting again.
func (p *MacPlatform) SetWindowAppearance(appearance gpui.Option[gpui.WindowAppearance]) {
	var nsAppearance appkit.NSAppearance
	if appearance, ok := appearance.Get(); ok {
		var name objc.ID
		switch appearance {
		case gpui.WindowAppearanceLight:
			name = appkit.NSAppearanceNameAqua()
		case gpui.WindowAppearanceDark:
			name = appkit.NSAppearanceNameDarkAqua()
		case gpui.WindowAppearanceVibrantLight:
			name = appkit.NSAppearanceNameVibrantLight()
		case gpui.WindowAppearanceVibrantDark:
			name = appkit.NSAppearanceNameVibrantDark()
		}
		nsAppearance = appkit.NSAppearanceNamed(name)
	}
	objc.SendVoid(sharedApplication(), objc.Sel("setAppearance:"), nsAppearance.ID())
}

// OpenURL opens url with the default application.
func (p *MacPlatform) OpenURL(url string) {
	s := objc.NSStringFromGo(url)
	defer objc.Release(s)
	nsURL := objc.SendClass[objc.ID](objc.GetClass("NSURL"), objc.Sel("URLWithString:"), s)
	if nsURL == objc.Nil {
		slog.Error("Failed to create NSURL from string: " + url)
		return
	}
	_ = objc.Send[bool](sharedWorkspace(), objc.Sel("openURL:"), nsURL)
}

func sharedWorkspace() objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSWorkspace"), objc.Sel("sharedWorkspace"))
}

func mainBundle() objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("NSBundle"), objc.Sel("mainBundle"))
}

// RegisterURLScheme makes this bundled, installed app the default handler of scheme.
func (p *MacPlatform) RegisterURLScheme(scheme string) *gpui.Task[struct{}] {
	doneTx, doneRx := scheduler.Oneshot[error]()
	if !macOSVersion().atLeast(12, 0, 0) {
		return scheduler.ReadyTaskErr[struct{}](errRegisterURLSchemeNeedsMacOS12)
	}
	bundleID := objc.Send[objc.ID](mainBundle(), objc.Sel("bundleIdentifier"))
	if bundleID == objc.Nil {
		return scheduler.ReadyTaskErr[struct{}](errors.New("Can only register URL scheme in bundled apps"))
	}
	workspace := sharedWorkspace()
	app := objc.Send[objc.ID](workspace, objc.Sel("URLForApplicationWithBundleIdentifier:"), bundleID)
	if app == objc.Nil {
		return scheduler.ReadyTaskErr[struct{}](errors.New("Cannot register URL scheme until app is installed"))
	}
	nsScheme := objc.NSStringFromGo(scheme)
	defer objc.Release(nsScheme)

	var once sync.Once
	handler := objc.NewBlock(func(_ objc.Block, nsErr objc.ID) {
		var result error
		if err := objc.NSErrorFromID(nsErr); err != nil {
			result = fmt.Errorf("Failed to register: %s", err.LocalizedDescription)
		}
		once.Do(func() { _ = doneTx.Send(result) })
	})
	objc.SendVoid(workspace, objc.Sel("setDefaultApplicationAtURL:toOpenURLsWithScheme:completionHandler:"), app, nsScheme, handler)
	handler.Release()

	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		result, err := doneRx.Recv(s)
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, result
	})
}

// OnOpenURLs registers the callback for URLs the app is asked to open.
func (p *MacPlatform) OnOpenURLs(callback func(urls []string)) {
	p.mu.Lock()
	p.state.openURLs = callback
	p.mu.Unlock()
}

const nsModalResponseOK objc.NSInteger = 1

// PromptForPaths shows an open panel and sends the chosen paths, or None when cancelled.
func (p *MacPlatform) PromptForPaths(options gpui.PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[[]string]]] {
	doneTx, doneRx := scheduler.Oneshot[scheduler.Result[gpui.Option[[]string]]]()
	p.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		panel := objc.SendClass[objc.ID](objc.GetClass("NSOpenPanel"), objc.Sel("openPanel"))
		if panel == objc.Nil {
			_ = doneTx.Send(scheduler.Result[gpui.Option[[]string]]{Err: errors.New("macOS could not create the open dialog")})
			return struct{}{}, nil
		}
		objc.SendVoid(panel, objc.Sel("setCanChooseDirectories:"), options.Directories)
		objc.SendVoid(panel, objc.Sel("setCanChooseFiles:"), options.Files)
		objc.SendVoid(panel, objc.Sel("setAllowsMultipleSelection:"), options.Multiple)
		objc.SendVoid(panel, objc.Sel("setCanCreateDirectories:"), true)
		objc.SendVoid(panel, objc.Sel("setResolvesAliases:"), false)

		retainedPanel := objc.RetainObject(panel)
		var done atomic.Bool
		handler := objc.NewBlock(func(_ objc.Block, response objc.NSInteger) {
			if done.Swap(true) {
				return
			}
			defer retainedPanel.Release()
			result := gpui.None[[]string]()
			if response == nsModalResponseOK {
				var paths []string
				for _, url := range objc.NSArrayToIDs(objc.Send[objc.ID](retainedPanel.ID(), objc.Sel("URLs"))) {
					if !objc.Send[bool](url, objc.Sel("isFileURL")) {
						continue
					}
					if path, err := nsURLToPath(url); err == nil {
						paths = append(paths, path)
					}
				}
				result = gpui.Some(paths)
			}
			_ = doneTx.Send(scheduler.Result[gpui.Option[[]string]]{Value: result})
		})

		if prompt, ok := options.Prompt.Get(); ok {
			nsPrompt := objc.NSStringFromGo(string(prompt))
			objc.SendVoid(panel, objc.Sel("setPrompt:"), nsPrompt)
			objc.Release(nsPrompt)
		}

		objc.SendVoid(panel, objc.Sel("beginWithCompletionHandler:"), handler)
		handler.Release()
		return struct{}{}, nil
	}).Detach()
	return doneRx
}

// PromptForNewPath shows a save panel in directory and sends the chosen path, or None.
func (p *MacPlatform) PromptForNewPath(directory string, suggestedName gpui.Option[string]) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[string]]] {
	url := nsURLFromDirectoryPath(directory)
	doneTx, doneRx := scheduler.Oneshot[scheduler.Result[gpui.Option[string]]]()
	p.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer url.Release()
		panel := objc.SendClass[objc.ID](objc.GetClass("NSSavePanel"), objc.Sel("savePanel"))
		if panel == objc.Nil {
			_ = doneTx.Send(scheduler.Result[gpui.Option[string]]{Err: errors.New("macOS could not create the save dialog")})
			return struct{}{}, nil
		}
		objc.SendVoid(panel, objc.Sel("setDirectoryURL:"), url.ID())

		if name, ok := suggestedName.Get(); ok {
			nsName := objc.NSStringFromGo(name)
			objc.SendVoid(panel, objc.Sel("setNameFieldStringValue:"), nsName)
			objc.Release(nsName)
		}

		retainedPanel := objc.RetainObject(panel)
		var done atomic.Bool
		handler := objc.NewBlock(func(_ objc.Block, response objc.NSInteger) {
			if done.Swap(true) {
				return
			}
			defer retainedPanel.Release()
			result := gpui.None[string]()
			if response == nsModalResponseOK {
				nsURL := objc.Send[objc.ID](retainedPanel.ID(), objc.Sel("URL"))
				if nsURL != objc.Nil && objc.Send[bool](nsURL, objc.Sel("isFileURL")) {
					if path, err := nsURLToPath(nsURL); err == nil {
						result = gpui.Some(fixSequoiaExtension(path))
					}
				}
			}
			_ = doneTx.Send(scheduler.Result[gpui.Option[string]]{Value: result})
		})

		objc.SendVoid(panel, objc.Sel("beginWithCompletionHandler:"), handler)
		handler.Release()
		return struct{}{}, nil
	}).Detach()
	return doneRx
}

func fixSequoiaExtension(path string) string {
	filename := filepath.Base(path)
	if filename == "" || filename == "." || filename == "/" {
		return path
	}
	chunks := bytes.Split([]byte(filename), []byte("."))
	if len(chunks) == 3 && bytes.HasPrefix(chunks[1], chunks[2]) && macOSVersion().atLeast(15, 0, 0) {
		return strings.TrimSuffix(path, filepath.Ext(path))
	}
	return path
}

func nsURLFromDirectoryPath(directory string) *objc.Retained {
	s := objc.NSStringFromGo(directory)
	defer objc.Release(s)
	url := objc.SendClass[objc.ID](objc.GetClass("NSURL"), objc.Sel("fileURLWithPath:isDirectory:"), s, true)
	return objc.RetainObject(url)
}

// CanSelectMixedFilesAndDirs reports true: an open panel can choose both at once.
func (p *MacPlatform) CanSelectMixedFilesAndDirs() bool {
	return true
}

// RevealPath selects path in Finder.
func (p *MacPlatform) RevealPath(path string) {
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		objc.AutoreleasePool(func() {
			fullPath := path
			if !utf8.ValidString(fullPath) {
				fullPath = ""
			}
			nsFullPath := objc.NSStringFromGo(fullPath)
			rootFullPath := objc.NSStringFromGo("")
			_ = objc.Send[bool](sharedWorkspace(), objc.Sel("selectFile:inFileViewerRootedAtPath:"), nsFullPath, rootFullPath)
			objc.Release(nsFullPath)
			objc.Release(rootFullPath)
		})
		return struct{}{}, nil
	}).Detach()
}

// OpenWithSystem opens path with its default application through open(1).
func (p *MacPlatform) OpenWithSystem(path string) {
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if err := gpuiutil.NewStdCommand("open", "--", path).Run(); err != nil {
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				slog.Error(fmt.Errorf("invoking open command: %w", err).Error())
			}
		}
		return struct{}{}, nil
	}).Detach()
}

// OnQuit registers the callback run when the application terminates.
func (p *MacPlatform) OnQuit(callback func() bool) {
	p.mu.Lock()
	p.state.quit = callback
	p.mu.Unlock()
}

// OnReopen registers the callback run when the app is reopened without visible windows.
func (p *MacPlatform) OnReopen(callback func()) {
	p.mu.Lock()
	p.state.reopen = callback
	p.mu.Unlock()
}

// OnKeyboardLayoutChange registers the callback run after the keyboard layout changes.
func (p *MacPlatform) OnKeyboardLayoutChange(callback func()) {
	p.mu.Lock()
	p.state.onKeyboardLayoutChange = callback
	p.mu.Unlock()
}

// OnAppMenuAction registers the callback for menu item actions.
func (p *MacPlatform) OnAppMenuAction(callback func(action gpui.Action)) {
	p.mu.Lock()
	p.state.menuCommand = callback
	p.mu.Unlock()
}

// OnWillOpenAppMenu registers the callback run before a menu opens.
func (p *MacPlatform) OnWillOpenAppMenu(callback func()) {
	p.mu.Lock()
	p.state.willOpenMenu = callback
	p.mu.Unlock()
}

// OnValidateAppMenuCommand registers the callback deciding whether a menu item is enabled.
func (p *MacPlatform) OnValidateAppMenuCommand(callback func(action gpui.Action) bool) {
	p.mu.Lock()
	p.state.validateMenuCommand = callback
	p.mu.Unlock()
}

// OnThermalStateChange registers the callback run when the thermal state changes.
func (p *MacPlatform) OnThermalStateChange(callback func()) {
	executor := p.ForegroundExecutor()
	observer := apple.ObserveThermalState(&executor, callback)
	p.mu.Lock()
	previous := p.state.thermalObserver
	p.state.thermalObserver = observer
	p.mu.Unlock()
	if previous != nil {
		previous.Close()
	}
}

// OnSystemSleep registers the callback run when the system is about to sleep.
func (p *MacPlatform) OnSystemSleep(callback func()) {
	p.mu.Lock()
	p.state.onSystemSleep = callback
	p.mu.Unlock()
	p.ensureSystemPowerObservers()
}

// OnPowerModeChange registers the callback run when Low Power Mode changes.
func (p *MacPlatform) OnPowerModeChange(callback func()) {
	executor := p.ForegroundExecutor()
	observer := apple.ObservePowerMode(&executor, callback)
	p.mu.Lock()
	previous := p.state.powerModeObserver
	p.state.powerModeObserver = observer
	p.mu.Unlock()
	if previous != nil {
		previous.Close()
	}
}

// OnSystemWake registers the callback run when the system wakes.
func (p *MacPlatform) OnSystemWake(callback func()) {
	p.mu.Lock()
	p.state.onSystemWake = callback
	p.mu.Unlock()
	p.ensureSystemPowerObservers()
}

type macDebugState struct {
	AppIsActive              bool              `json:"app_is_active"`
	AppIsHidden              bool              `json:"app_is_hidden"`
	AppOcclusionStateVisible bool              `json:"app_occlusion_state_visible"`
	CurrentCursorStyle       *gpui.CursorStyle `json:"current_cursor_style"`
	KeyWindowNumber          *int              `json:"key_window_number"`
	LowPowerMode             bool              `json:"low_power_mode"`
	MainWindowNumber         *int              `json:"main_window_number"`
}

// DebugState reports application activity, the cursor and the key and main windows.
func (p *MacPlatform) DebugState() jsontext.Value {
	app := appkit.NSApplicationSharedApplication()
	keyWindow := objc.Send[objc.ID](app.ID(), objc.Sel("keyWindow"))
	mainWindow := objc.Send[objc.ID](app.ID(), objc.Sel("mainWindow"))
	windowNumber := func(window objc.ID) *int {
		if window == objc.Nil {
			return nil
		}
		n := objc.Send[int](window, objc.Sel("windowNumber"))
		return &n
	}
	currentCursor := objc.SendClass[objc.ID](objc.GetClass("NSCursor"), objc.Sel("currentCursor"))
	var currentCursorStyle *gpui.CursorStyle
	for style := range gpui.CursorStyleIter() {
		if nativeCursor(style) == currentCursor {
			currentCursorStyle = &style
			break
		}
	}
	state := macDebugState{
		AppIsActive:              app.IsActive(),
		AppIsHidden:              objc.Send[bool](app.ID(), objc.Sel("isHidden")),
		AppOcclusionStateVisible: app.OcclusionState()&appkit.NSApplicationOcclusionStateVisible != 0,
		CurrentCursorStyle:       currentCursorStyle,
		KeyWindowNumber:          windowNumber(keyWindow),
		LowPowerMode:             objc.Send[bool](processInfo(), objc.Sel("isLowPowerModeEnabled")),
		MainWindowNumber:         windowNumber(mainWindow),
	}
	value, err := json.Marshal(state)
	if err != nil {
		panic(fmt.Sprintf("macos: encode debug state: %v", err))
	}
	return jsontext.Value(value)
}

// ThermalState returns the system's thermal state.
func (p *MacPlatform) ThermalState() gpui.ThermalState {
	return apple.ThermalState()
}

// PreventIdleSleep begins a user-initiated activity that keeps the system awake.
func (p *MacPlatform) PreventIdleSleep(reason string) *gpui.Task[*gpui.ActivityGuard] {
	return scheduler.ReadyTask(apple.AppleActivityBegin(reason, objc.NSActivityUserInitiated))
}

// PowerMode returns the system's power mode.
func (p *MacPlatform) PowerMode() gpui.PowerMode {
	return apple.PowerMode()
}

// ShowSystemNotification posts notification to the notification center.
func (p *MacPlatform) ShowSystemNotification(notification gpui.SystemNotification) {
	p.mu.Lock()
	foregroundExecutor := p.state.foregroundExecutor
	backgroundExecutor := p.state.backgroundExecutor
	task := p.state.systemNotifications.send(foregroundExecutor, backgroundExecutor, notificationCommandShow{notification: notification})
	p.mu.Unlock()
	task.Detach()
}

// DismissSystemNotification removes the notifications posted with tag.
func (p *MacPlatform) DismissSystemNotification(tag string) *gpui.Task[struct{}] {
	p.mu.Lock()
	defer p.mu.Unlock()
	foregroundExecutor := p.state.foregroundExecutor
	backgroundExecutor := p.state.backgroundExecutor
	return p.state.systemNotifications.send(foregroundExecutor, backgroundExecutor, notificationCommandDismiss{tag: gpui.SharedString(tag)})
}

// OnSystemNotificationResponse registers the callback for activated notifications.
func (p *MacPlatform) OnSystemNotificationResponse(callback func(response gpui.SystemNotificationResponse)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	executor := p.state.foregroundExecutor
	p.state.systemNotifications.onResponse(executor, callback)
}

// KeyboardLayout returns the current keyboard layout.
func (p *MacPlatform) KeyboardLayout() gpui.PlatformKeyboardLayout {
	return NewMacKeyboardLayout()
}

// KeyboardMapper returns the mapper for the current keyboard layout.
func (p *MacPlatform) KeyboardMapper() gpui.PlatformKeyboardMapper {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.keyboardMapper
}

// AppPath returns the path of the app bundle.
func (p *MacPlatform) AppPath() (string, error) {
	bundle := mainBundle()
	if bundle == objc.Nil {
		return "", errAppNotRunningInsideABundle
	}
	return pathFromObjc(objc.Send[objc.ID](bundle, objc.Sel("bundlePath"))), nil
}

// SetMenus installs menus as the application's main menu.
func (p *MacPlatform) SetMenus(menus []gpui.Menu, keymap *gpui.Keymap) {
	app := sharedApplication()
	p.mu.Lock()
	delegate := appkit.NSApplication(app).Delegate()
	menu := p.createMenuBar(menus, delegate, &p.state.menuActions, keymap)
	p.mu.Unlock()
	objc.SendVoid(app, objc.Sel("setMainMenu:"), menu)
	objc.Release(menu)
	owned := make([]gpui.OwnedMenu, 0, len(menus))
	for _, menu := range menus {
		owned = append(owned, menu.Owned())
	}
	p.mu.Lock()
	p.state.menus = gpui.Some(owned)
	p.mu.Unlock()
}

// GetMenus returns the menus last passed to SetMenus.
func (p *MacPlatform) GetMenus() ([]gpui.OwnedMenu, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.menus.Get()
}

// SetDockMenu sets the menu shown for the app's Dock icon.
func (p *MacPlatform) SetDockMenu(menu []gpui.MenuItem, keymap *gpui.Keymap) {
	app := sharedApplication()
	p.mu.Lock()
	defer p.mu.Unlock()
	delegate := appkit.NSApplication(app).Delegate()
	dockMenu := objc.Own(p.createDockMenu(menu, delegate, &p.state.menuActions, keymap))
	if p.state.dockMenu != nil {
		p.state.dockMenu.Release()
	}
	p.state.dockMenu = dockMenu
}

// AddRecentDocument adds path to the recent documents list.
func (p *MacPlatform) AddRecentDocument(path string) {
	if !utf8.ValidString(path) {
		return
	}
	documentController := objc.SendClass[objc.ID](objc.GetClass("NSDocumentController"), objc.Sel("sharedDocumentController"))
	url := objc.NSURLFromFilePath(path)
	defer objc.Release(url)
	objc.SendVoid(documentController, objc.Sel("noteNewRecentDocumentURL:"), url)
}

// PathForAuxiliaryExecutable returns the path of the bundled auxiliary executable name.
func (p *MacPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	bundle := mainBundle()
	if bundle == objc.Nil {
		return "", errAppNotRunningInsideABundle
	}
	url := objc.Send[objc.ID](bundle, objc.Sel("URLForAuxiliaryExecutable:"), nsString(name))
	if url == objc.Nil {
		return "", errAuxiliaryExecutableNotFound
	}
	return nsURLToPath(url)
}

// SetCursorStyle sets the cursor of the active window to style.
func (p *MacPlatform) SetCursorStyle(style gpui.CursorStyle) {
	setActiveWindowCursorStyle(style)
}

// HideCursorUntilMouseMoves hides the cursor until the mouse moves.
func (p *MacPlatform) HideCursorUntilMouseMoves() {
	p.mu.Lock()
	cursorVisible := p.state.cursorVisible
	p.mu.Unlock()
	if !cursorVisible.Swap(false) {
		return
	}
	appkit.NSCursorSetHiddenUntilMouseMoves(true)
}

// IsCursorVisible reports whether the cursor is visible.
func (p *MacPlatform) IsCursorVisible() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.cursorVisible.Load()
}

const nsScrollerStyleOverlay objc.NSInteger = 1

// ShouldAutoHideScrollbars reports whether the system uses overlay scrollers.
func (p *MacPlatform) ShouldAutoHideScrollbars() bool {
	return objc.SendClass[objc.NSInteger](objc.GetClass("NSScroller"), objc.Sel("preferredScrollerStyle")) == nsScrollerStyleOverlay
}

// ReadFromClipboard reads the general pasteboard.
func (p *MacPlatform) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.generalPasteboard == nil {
		return gpui.ClipboardItem{}, false
	}
	return p.state.generalPasteboard.Read()
}

// WriteToClipboard writes item to the general pasteboard.
func (p *MacPlatform) WriteToClipboard(item gpui.ClipboardItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.generalPasteboard != nil {
		p.state.generalPasteboard.Write(item)
	}
}

// ReadFromFindPasteboard reads the find pasteboard.
func (p *MacPlatform) ReadFromFindPasteboard() (gpui.ClipboardItem, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.findPasteboard == nil {
		return gpui.ClipboardItem{}, false
	}
	return p.state.findPasteboard.Read()
}

// WriteToFindPasteboard writes item to the find pasteboard.
func (p *MacPlatform) WriteToFindPasteboard(item gpui.ClipboardItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.findPasteboard != nil {
		p.state.findPasteboard.Write(item)
	}
}

// WriteCredentials stores a password for url in the keychain.
func (p *MacPlatform) WriteCredentials(url string, username string, password []byte) *gpui.Task[struct{}] {
	password = append([]byte(nil), password...)
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		return struct{}{}, apple.WriteCredentials(url, username, password)
	})
}

// WriteCredentialsIfNotExists stores a password for url unless one exists, reporting whether it wrote.
func (p *MacPlatform) WriteCredentialsIfNotExists(url string, username string, password []byte) *gpui.Task[bool] {
	password = append([]byte(nil), password...)
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (bool, error) {
		return apple.WriteCredentialsIfNotExists(url, username, password)
	})
}

// ReadCredentials reads the credentials stored for url.
func (p *MacPlatform) ReadCredentials(url string) *gpui.Task[gpui.Option[gpui.Credentials]] {
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
func (p *MacPlatform) DeleteCredentials(url string) *gpui.Task[struct{}] {
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		return struct{}{}, apple.DeleteCredentials(url)
	})
}

func pathFromObjc(path objc.ID) string {
	return nsStringToStr(path)
}

func setMacPlatformIvar(object objc.ID, platform *MacPlatform) {
	macPlatformIvarsMu.Lock()
	defer macPlatformIvarsMu.Unlock()
	macPlatformIvars[object] = platform
}

func getMacPlatform(object objc.ID) *MacPlatform {
	macPlatformIvarsMu.Lock()
	platform, ok := macPlatformIvars[object]
	macPlatformIvarsMu.Unlock()
	if !ok {
		panic(macPlatformIvar + " ivar is missing")
	}
	if platform == nil {
		panic("assertion failed: !platform_ptr.is_null()")
	}
	return platform
}

func willFinishLaunching(_ objc.ID) {
	userDefaults := objc.SendClass[objc.ID](objc.GetClass("NSUserDefaults"), objc.Sel("standardUserDefaults"))
	name := nsString("NSAutoFillHeuristicControllerEnabled")
	existingValue := objc.Send[objc.ID](userDefaults, objc.Sel("objectForKey:"), name)
	if existingValue == objc.Nil {
		falseValue := objc.SendClass[objc.ID](objc.GetClass("NSNumber"), objc.Sel("numberWithBool:"), false)
		objc.SendVoid(userDefaults, objc.Sel("setObject:forKey:"), falseValue, name)
	}
}

func didFinishLaunching(this objc.ID) {
	app := sharedApplication()
	platform := getMacPlatform(this)
	platform.mu.Lock()
	policy := platform.state.activationPolicy
	platform.mu.Unlock()
	appkit.NSApplication(app).SetActivationPolicy(nativeActivationPolicy(policy))

	notificationCenter := objc.SendClass[objc.ID](objc.GetClass("NSNotificationCenter"), objc.Sel("defaultCenter"))
	name := nsString("NSTextInputContextKeyboardSelectionDidChangeNotification")
	objc.SendVoid(notificationCenter, objc.Sel("addObserver:selector:name:object:"), this, objc.Sel("onKeyboardLayoutChange:"), name, objc.Nil)
	objc.SendVoid(notificationCenter, objc.Sel("addObserver:selector:name:object:"), this, objc.Sel("onScreenParametersChange:"), nsString("NSApplicationDidChangeScreenParametersNotification"), objc.Nil)

	platform.mu.Lock()
	if (platform.state.onSystemSleep != nil || platform.state.onSystemWake != nil) && !platform.state.systemPowerObserversRegistered {
		registerSystemPowerObservers(this)
		platform.state.systemPowerObserversRegistered = true
	}
	callback := platform.state.finishLaunching
	platform.state.finishLaunching = nil
	platform.mu.Unlock()
	if callback != nil {
		callback()
	}
}

func registerSystemPowerObservers(observer objc.ID) {
	center := objc.Send[objc.ID](sharedWorkspace(), objc.Sel("notificationCenter"))
	objc.SendVoid(center, objc.Sel("addObserver:selector:name:object:"), observer, objc.Sel("onSystemSleep:"), appkit.Constant("NSWorkspaceWillSleepNotification"), objc.Nil)
	objc.SendVoid(center, objc.Sel("addObserver:selector:name:object:"), observer, objc.Sel("onSystemWake:"), appkit.Constant("NSWorkspaceDidWakeNotification"), objc.Nil)
}

func shouldHandleReopen(this objc.ID, _ objc.SEL, _ objc.ID, hasOpenWindows bool) bool {
	if !hasOpenWindows {
		platform := getMacPlatform(this)
		platform.mu.Lock()
		callback := platform.state.reopen
		platform.state.reopen = nil
		platform.mu.Unlock()
		if callback != nil {
			callback()
			platform.mu.Lock()
			if platform.state.reopen == nil {
				platform.state.reopen = callback
			}
			platform.mu.Unlock()
		}
	}
	return true
}

func willTerminate(this objc.ID) {
	platform := getMacPlatform(this)
	platform.mu.Lock()
	callback := platform.state.quit
	platform.state.quit = nil
	platform.mu.Unlock()
	if callback != nil {
		callback()
		platform.mu.Lock()
		if platform.state.quit == nil {
			platform.state.quit = callback
		}
		platform.mu.Unlock()
	}
}

func onKeyboardLayoutChange(this objc.ID) {
	platform := getMacPlatform(this)
	platform.mu.Lock()
	keyboardLayout := NewMacKeyboardLayout()
	platform.state.keyboardMapper = NewMacKeyboardMapper(keyboardLayout.ID())
	callback := platform.state.onKeyboardLayoutChange
	platform.state.onKeyboardLayoutChange = nil
	platform.mu.Unlock()
	if callback != nil {
		callback()
		platform.mu.Lock()
		if platform.state.onKeyboardLayoutChange == nil {
			platform.state.onKeyboardLayoutChange = callback
		}
		platform.mu.Unlock()
	}
}

func onScreenParametersChange(this objc.ID) {
	platform := getMacPlatform(this)
	gcd.MainQueue().Async(func() {
		platform.mu.Lock()
		callback := platform.state.onDisplaysChanged
		platform.state.onDisplaysChanged = nil
		platform.mu.Unlock()
		if callback != nil {
			callback()
			platform.mu.Lock()
			if platform.state.onDisplaysChanged == nil {
				platform.state.onDisplaysChanged = callback
			}
			platform.mu.Unlock()
		}
	})
}

func onSystemSleep(this objc.ID) {
	platform := getMacPlatform(this)
	gcd.MainQueue().Async(func() {
		platform.mu.Lock()
		callback := platform.state.onSystemSleep
		platform.state.onSystemSleep = nil
		platform.mu.Unlock()
		if callback != nil {
			callback()
			platform.mu.Lock()
			if platform.state.onSystemSleep == nil {
				platform.state.onSystemSleep = callback
			}
			platform.mu.Unlock()
		}
	})
}

func onSystemWake(this objc.ID) {
	platform := getMacPlatform(this)
	gcd.MainQueue().Async(func() {
		platform.mu.Lock()
		callback := platform.state.onSystemWake
		platform.state.onSystemWake = nil
		platform.mu.Unlock()
		if callback != nil {
			callback()
			platform.mu.Lock()
			if platform.state.onSystemWake == nil {
				platform.state.onSystemWake = callback
			}
			platform.mu.Unlock()
		}
	})
}

func openURLs(this objc.ID, _ objc.SEL, _ objc.ID, urls objc.ID) {
	var strs []string
	if urls != objc.Nil {
		for _, url := range objc.NSArrayToIDs(urls) {
			str := objc.Send[objc.ID](url, objc.Sel("absoluteString"))
			if str == objc.Nil {
				continue
			}
			s := nsStringToStr(str)
			if !utf8.ValidString(s) {
				slog.Error("error converting path to string: invalid utf-8 sequence")
				continue
			}
			strs = append(strs, s)
		}
	}
	platform := getMacPlatform(this)
	platform.mu.Lock()
	callback := platform.state.openURLs
	platform.state.openURLs = nil
	platform.mu.Unlock()
	if callback != nil {
		callback(strs)
		platform.mu.Lock()
		if platform.state.openURLs == nil {
			platform.state.openURLs = callback
		}
		platform.mu.Unlock()
	}
}

func handleMenuItem(this objc.ID, _ objc.SEL, item objc.ID) {
	platform := getMacPlatform(this)
	platform.mu.Lock()
	callback := platform.state.menuCommand
	if callback == nil {
		platform.mu.Unlock()
		return
	}
	platform.state.menuCommand = nil
	tag := 0
	if item != objc.Nil {
		tag = objc.Send[int](item, objc.Sel("tag"))
	}
	if tag >= 0 && tag < len(platform.state.menuActions) {
		action := platform.state.menuActions[tag].BoxedClone()
		platform.mu.Unlock()
		callback(action)
	} else {
		platform.mu.Unlock()
	}
	platform.mu.Lock()
	if platform.state.menuCommand == nil {
		platform.state.menuCommand = callback
	}
	platform.mu.Unlock()
}

func validateMenuItem(this objc.ID, _ objc.SEL, item objc.ID) bool {
	result := false
	platform := getMacPlatform(this)
	platform.mu.Lock()
	callback := platform.state.validateMenuCommand
	if callback == nil {
		platform.mu.Unlock()
		return result
	}
	platform.state.validateMenuCommand = nil
	tag := 0
	if item != objc.Nil {
		tag = objc.Send[int](item, objc.Sel("tag"))
	}
	if tag >= 0 && tag < len(platform.state.menuActions) {
		action := platform.state.menuActions[tag].BoxedClone()
		platform.mu.Unlock()
		result = callback(action)
	} else {
		platform.mu.Unlock()
	}
	platform.mu.Lock()
	if platform.state.validateMenuCommand == nil {
		platform.state.validateMenuCommand = callback
	}
	platform.mu.Unlock()
	return result
}

func menuWillOpen(this objc.ID) {
	platform := getMacPlatform(this)
	platform.mu.Lock()
	callback := platform.state.willOpenMenu
	platform.state.willOpenMenu = nil
	platform.mu.Unlock()
	if callback != nil {
		callback()
		platform.mu.Lock()
		if platform.state.willOpenMenu == nil {
			platform.state.willOpenMenu = callback
		}
		platform.mu.Unlock()
	}
}

func handleDockMenu(this objc.ID, _ objc.SEL, _ objc.ID) objc.ID {
	platform := getMacPlatform(this)
	platform.mu.Lock()
	defer platform.mu.Unlock()
	if platform.state.dockMenu == nil {
		return objc.Nil
	}
	return platform.state.dockMenu.ID()
}

func nsURLToPath(url objc.ID) (string, error) {
	path := objc.Send[*byte](url, objc.Sel("fileSystemRepresentation"))
	if path == nil {
		return "", fmt.Errorf("url is not a file path: %s", nsStringToStr(objc.Send[objc.ID](url, objc.Sel("absoluteString"))))
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(path), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(path, n)), nil
}

const carbonPath = "/System/Library/Frameworks/Carbon.framework/Carbon"

var (
	carbonOnce sync.Once
	carbonErr  error

	tisCopyCurrentKeyboardLayoutInputSource func() objc.ID
	tisCopyCurrentKeyboardInputSource       func() objc.ID
	tisGetInputSourceProperty               func(inputSource objc.ID, propertyKey cf.TypeRef) objc.ID
	ucKeyTranslate                          func(keyLayoutPtr unsafe.Pointer, virtualKeyCode uint16, keyAction uint16, modifierKeyState uint32, keyboardType uint32, keyTranslateOptions uint32, deadKeyState *uint32, maxStringLength uintptr, actualStringLength *uintptr, unicodeString *uint16) uint32
	lmGetKbdType                            func() uint16
	cfDataGetBytePtr                        func(data cf.TypeRef) unsafe.Pointer

	kTISPropertyUnicodeKeyLayoutData      cf.TypeRef
	kTISPropertyInputSourceID             cf.TypeRef
	kTISPropertyLocalizedName             cf.TypeRef
	kTISPropertyInputSourceIsASCIICapable cf.TypeRef
	kTISPropertyInputSourceType           cf.TypeRef
	kTISTypeKeyboardInputMode             cf.TypeRef
)

func loadCarbon() error {
	carbonOnce.Do(func() {
		if err := objc.Load(); err != nil {
			carbonErr = err
			return
		}
		lib, err := purego.Dlopen(carbonPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			carbonErr = fmt.Errorf("macos: open %s: %w", carbonPath, err)
			return
		}
		purego.RegisterLibFunc(&tisCopyCurrentKeyboardLayoutInputSource, lib, "TISCopyCurrentKeyboardLayoutInputSource")
		purego.RegisterLibFunc(&tisCopyCurrentKeyboardInputSource, lib, "TISCopyCurrentKeyboardInputSource")
		purego.RegisterLibFunc(&tisGetInputSourceProperty, lib, "TISGetInputSourceProperty")
		purego.RegisterLibFunc(&ucKeyTranslate, lib, "UCKeyTranslate")
		purego.RegisterLibFunc(&lmGetKbdType, lib, "LMGetKbdType")
		cfLib, err := purego.Dlopen(cf.CoreFoundationPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			carbonErr = fmt.Errorf("macos: open %s: %w", cf.CoreFoundationPath, err)
			return
		}
		purego.RegisterLibFunc(&cfDataGetBytePtr, cfLib, "CFDataGetBytePtr")
		for _, c := range []struct {
			dst  *cf.TypeRef
			name string
		}{
			{&kTISPropertyUnicodeKeyLayoutData, "kTISPropertyUnicodeKeyLayoutData"},
			{&kTISPropertyInputSourceID, "kTISPropertyInputSourceID"},
			{&kTISPropertyLocalizedName, "kTISPropertyLocalizedName"},
			{&kTISPropertyInputSourceIsASCIICapable, "kTISPropertyInputSourceIsASCIICapable"},
			{&kTISPropertyInputSourceType, "kTISPropertyInputSourceType"},
			{&kTISTypeKeyboardInputMode, "kTISTypeKeyboardInputMode"},
		} {
			ref, err := cf.Constant(lib, c.name)
			if err != nil {
				carbonErr = fmt.Errorf("macos: %w", err)
				return
			}
			*c.dst = ref
		}
	})
	return carbonErr
}

func mustLoadCarbon() {
	if err := loadCarbon(); err != nil {
		panic(err)
	}
}
