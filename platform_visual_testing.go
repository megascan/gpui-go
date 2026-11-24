package gpui

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/megascan/gpui-go/scheduler"
)

// VisualTestPlatform combines a real platform's rendering with a controllable TestDispatcher, so
// visual tests render real UI for screenshots while task scheduling and time stay deterministic.
type VisualTestPlatform struct {
	PlatformDefaults
	dispatcher         *TestDispatcher
	backgroundExecutor BackgroundExecutor
	foregroundExecutor ForegroundExecutor
	platform           Platform
	clipboardMu        sync.Mutex
	clipboard          Option[ClipboardItem]
	findPasteboardMu   sync.Mutex
	findPasteboard     Option[ClipboardItem]
}

var _ Platform = (*VisualTestPlatform)(nil)

// NewVisualTestPlatform wraps platform with a TestDispatcher seeded with seed.
func NewVisualTestPlatform(platform Platform, seed uint64) *VisualTestPlatform {
	dispatcher := NewTestDispatcher(seed)
	arcDispatcher := dispatcher.Clone()

	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)

	p := &VisualTestPlatform{
		dispatcher:         dispatcher,
		backgroundExecutor: backgroundExecutor,
		foregroundExecutor: foregroundExecutor,
		platform:           platform,
	}
	BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

// Dispatcher returns the TestDispatcher that controls task scheduling and time.
func (p *VisualTestPlatform) Dispatcher() *TestDispatcher {
	return p.dispatcher
}

// PlatformInfo returns the wrapped platform's info.
func (p *VisualTestPlatform) PlatformInfo() PlatformInfo {
	return p.platform.PlatformInfo()
}

// BackgroundExecutor returns the test dispatcher's background executor.
func (p *VisualTestPlatform) BackgroundExecutor() BackgroundExecutor {
	return p.backgroundExecutor
}

// ForegroundExecutor returns the test dispatcher's foreground executor.
func (p *VisualTestPlatform) ForegroundExecutor() ForegroundExecutor {
	return p.foregroundExecutor
}

// TextSystem returns the wrapped platform's text system.
func (p *VisualTestPlatform) TextSystem() PlatformTextSystem {
	return p.platform.TextSystem()
}

// Run panics: visual tests never run the platform loop.
func (p *VisualTestPlatform) Run(onFinishLaunching func()) {
	panic("VisualTestPlatform::run should not be called in tests")
}

// Quit does nothing.
func (p *VisualTestPlatform) Quit() {}

// Restart does nothing.
func (p *VisualTestPlatform) Restart(binaryPath Option[string], arguments []string) {}

// Activate does nothing.
func (p *VisualTestPlatform) Activate(ignoringOtherApps bool) {}

// Hide does nothing.
func (p *VisualTestPlatform) Hide() {}

// HideOtherApps does nothing.
func (p *VisualTestPlatform) HideOtherApps() {}

// UnhideOtherApps does nothing.
func (p *VisualTestPlatform) UnhideOtherApps() {}

// Displays returns the wrapped platform's displays.
func (p *VisualTestPlatform) Displays() []PlatformDisplay {
	return p.platform.Displays()
}

// PrimaryDisplay returns the wrapped platform's primary display.
func (p *VisualTestPlatform) PrimaryDisplay() PlatformDisplay {
	return p.platform.PrimaryDisplay()
}

// OnDisplaysChanged registers callback with the wrapped platform.
func (p *VisualTestPlatform) OnDisplaysChanged(callback func()) {
	p.platform.OnDisplaysChanged(callback)
}

// ActiveWindow returns the wrapped platform's active window.
func (p *VisualTestPlatform) ActiveWindow() (AnyWindowHandle, bool) {
	return p.platform.ActiveWindow()
}

// WindowStack returns the wrapped platform's window stack.
func (p *VisualTestPlatform) WindowStack() ([]AnyWindowHandle, bool) {
	return p.platform.WindowStack()
}

// IsScreenCaptureSupported reports the wrapped platform's screen-capture support.
func (p *VisualTestPlatform) IsScreenCaptureSupported() bool {
	return p.platform.IsScreenCaptureSupported()
}

// ScreenCaptureSources returns the wrapped platform's screen capture sources.
func (p *VisualTestPlatform) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	return p.platform.ScreenCaptureSources()
}

// OpenWindow opens a real window on the wrapped platform.
func (p *VisualTestPlatform) OpenWindow(handle AnyWindowHandle, options WindowParams) (PlatformWindow, error) {
	return p.platform.OpenWindow(handle, options)
}

// WindowAppearance returns the wrapped platform's appearance.
func (p *VisualTestPlatform) WindowAppearance() WindowAppearance {
	return p.platform.WindowAppearance()
}

// OpenURL opens url through the wrapped platform.
func (p *VisualTestPlatform) OpenURL(url string) {
	p.platform.OpenURL(url)
}

// OnOpenURLs ignores the callback.
func (p *VisualTestPlatform) OnOpenURLs(callback func(urls []string)) {}

// RegisterURLScheme succeeds without registering anything.
func (p *VisualTestPlatform) RegisterURLScheme(url string) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// PromptForPaths resolves to no selection.
func (p *VisualTestPlatform) PromptForPaths(options PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[Option[[]string]]]()
	_ = tx.Send(scheduler.Result[Option[[]string]]{Value: None[[]string]()})
	return rx
}

// PromptForNewPath resolves to no selection.
func (p *VisualTestPlatform) PromptForNewPath(directory string, suggestedName Option[string]) *scheduler.OneshotReceiver[scheduler.Result[Option[string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[Option[string]]]()
	_ = tx.Send(scheduler.Result[Option[string]]{Value: None[string]()})
	return rx
}

// CanSelectMixedFilesAndDirs returns true.
func (p *VisualTestPlatform) CanSelectMixedFilesAndDirs() bool {
	return true
}

// RevealPath reveals path through the wrapped platform.
func (p *VisualTestPlatform) RevealPath(path string) {
	p.platform.RevealPath(path)
}

// OpenWithSystem opens path through the wrapped platform.
func (p *VisualTestPlatform) OpenWithSystem(path string) {
	p.platform.OpenWithSystem(path)
}

// OnQuit ignores the callback.
func (p *VisualTestPlatform) OnQuit(callback func() bool) {}

// OnReopen ignores the callback.
func (p *VisualTestPlatform) OnReopen(callback func()) {}

// OnSystemSleep ignores the callback.
func (p *VisualTestPlatform) OnSystemSleep(callback func()) {}

// OnSystemWake ignores the callback.
func (p *VisualTestPlatform) OnSystemWake(callback func()) {}

// SetMenus does nothing.
func (p *VisualTestPlatform) SetMenus(menus []Menu, keymap *Keymap) {}

// GetMenus returns no menus.
func (p *VisualTestPlatform) GetMenus() ([]OwnedMenu, bool) {
	return nil, false
}

// SetDockMenu does nothing.
func (p *VisualTestPlatform) SetDockMenu(menu []MenuItem, keymap *Keymap) {}

// OnAppMenuAction ignores the callback.
func (p *VisualTestPlatform) OnAppMenuAction(callback func(action Action)) {}

// OnWillOpenAppMenu ignores the callback.
func (p *VisualTestPlatform) OnWillOpenAppMenu(callback func()) {}

// OnValidateAppMenuCommand ignores the callback.
func (p *VisualTestPlatform) OnValidateAppMenuCommand(callback func(action Action) bool) {}

// AppPath returns the wrapped platform's app path.
func (p *VisualTestPlatform) AppPath() (string, error) {
	return p.platform.AppPath()
}

// PathForAuxiliaryExecutable returns the wrapped platform's path for name.
func (p *VisualTestPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	return p.platform.PathForAuxiliaryExecutable(name)
}

// SetCursorStyle sets the cursor style on the wrapped platform.
func (p *VisualTestPlatform) SetCursorStyle(style CursorStyle) {
	p.platform.SetCursorStyle(style)
}

// HideCursorUntilMouseMoves hides the cursor on the wrapped platform.
func (p *VisualTestPlatform) HideCursorUntilMouseMoves() {
	p.platform.HideCursorUntilMouseMoves()
}

// IsCursorVisible reports the wrapped platform's cursor visibility.
func (p *VisualTestPlatform) IsCursorVisible() bool {
	return p.platform.IsCursorVisible()
}

// ShouldAutoHideScrollbars reports the wrapped platform's scrollbar preference.
func (p *VisualTestPlatform) ShouldAutoHideScrollbars() bool {
	return p.platform.ShouldAutoHideScrollbars()
}

// ReadFromClipboard returns the item last written to the in-memory clipboard.
func (p *VisualTestPlatform) ReadFromClipboard() (ClipboardItem, bool) {
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	return p.clipboard.Get()
}

// WriteToClipboard stores item in the in-memory clipboard.
func (p *VisualTestPlatform) WriteToClipboard(item ClipboardItem) {
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	p.clipboard = Some(item)
}

// ReadFromFindPasteboard returns the item last written to the in-memory find pasteboard on macOS.
func (p *VisualTestPlatform) ReadFromFindPasteboard() (ClipboardItem, bool) {
	if runtime.GOOS != "darwin" {
		return p.PlatformDefaults.ReadFromFindPasteboard()
	}
	p.findPasteboardMu.Lock()
	defer p.findPasteboardMu.Unlock()
	return p.findPasteboard.Get()
}

// WriteToFindPasteboard stores item in the in-memory find pasteboard on macOS.
func (p *VisualTestPlatform) WriteToFindPasteboard(item ClipboardItem) {
	if runtime.GOOS != "darwin" {
		p.PlatformDefaults.WriteToFindPasteboard(item)
		return
	}
	p.findPasteboardMu.Lock()
	defer p.findPasteboardMu.Unlock()
	p.findPasteboard = Some(item)
}

// WriteCredentials succeeds without storing anything.
func (p *VisualTestPlatform) WriteCredentials(url string, username string, password []byte) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// ReadCredentials resolves to no credentials.
func (p *VisualTestPlatform) ReadCredentials(url string) *Task[Option[Credentials]] {
	return scheduler.ReadyTask(None[Credentials]())
}

// DeleteCredentials succeeds without deleting anything.
func (p *VisualTestPlatform) DeleteCredentials(url string) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// KeyboardLayout returns the wrapped platform's keyboard layout.
func (p *VisualTestPlatform) KeyboardLayout() PlatformKeyboardLayout {
	return p.platform.KeyboardLayout()
}

// KeyboardMapper returns the wrapped platform's keyboard mapper.
func (p *VisualTestPlatform) KeyboardMapper() PlatformKeyboardMapper {
	return p.platform.KeyboardMapper()
}

// OnKeyboardLayoutChange ignores the callback.
func (p *VisualTestPlatform) OnKeyboardLayoutChange(callback func()) {}

// ThermalState returns ThermalStateNominal.
func (p *VisualTestPlatform) ThermalState() ThermalState {
	return ThermalStateNominal
}

// OnThermalStateChange ignores the callback.
func (p *VisualTestPlatform) OnThermalStateChange(callback func()) {}

// PreventIdleSleep fails: visual tests do not prevent idle sleep.
func (p *VisualTestPlatform) PreventIdleSleep(reason string) *Task[*ActivityGuard] {
	return scheduler.ReadyTaskErr[*ActivityGuard](fmt.Errorf("Idle sleep prevention for %q is not supported in visual tests", reason))
}
