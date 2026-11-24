package gpui

import (
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/megascan/gpui-go/scheduler"
)

// TestPlatform implements the Platform interface for use in tests.
type TestPlatform struct {
	PlatformDefaults
	platformInfo       PlatformInfo
	backgroundExecutor BackgroundExecutor
	foregroundExecutor ForegroundExecutor

	activeWindow             *TestWindow
	displays                 []*TestDisplay
	displaysChangedCallback  func()
	activeCursorMu           sync.Mutex
	activeCursor             CursorStyle
	clipboardMu              sync.Mutex
	currentClipboardItem     Option[ClipboardItem]
	currentPrimaryItem       Option[ClipboardItem]
	currentFindPasteboard    Option[ClipboardItem]
	prompts                  testPrompts
	screenCaptureSources     []TestScreenCaptureSource
	openedURL                Option[string]
	systemNotifications      testSystemNotifications
	textSystem               PlatformTextSystem
	expectRestart            *scheduler.OneshotSender[TestRestartRequest]
	idleSleepPreventionCount *atomic.Int64
	idleSleepPreventionDelay time.Duration
	idleSleepPreventionFails bool
	headlessRendererFactory  func() (PlatformHeadlessRenderer, error)
	weak                     weak.Pointer[TestPlatform]
	menus                    []OwnedMenu
	appLifecycleCallback     func(phase AppLifecyclePhase)
	memoryWarningCallback    func()
}

// TestRestartRequest is the binary path and arguments a test platform was asked to restart with.
type TestRestartRequest struct {
	// BinaryPath is the binary to restart, or None for the current one.
	BinaryPath Option[string]
	// Arguments are the arguments to restart with.
	Arguments []string
}

// TestScreenCaptureSource is a fake screen capture source, used for testing.
type TestScreenCaptureSource struct{}

// TestScreenCaptureStream is a fake screen capture stream, used for testing.
type TestScreenCaptureStream struct{}

// Metadata returns a 1x1 source with id 0.
func (TestScreenCaptureSource) Metadata() (SourceMetadata, error) {
	return SourceMetadata{
		ID:         0,
		IsMain:     None[bool](),
		Label:      None[SharedString](),
		Resolution: NewSize(DevicePixels(1), DevicePixels(1)),
	}, nil
}

// Stream returns a receiver that already holds a TestScreenCaptureStream.
func (TestScreenCaptureSource) Stream(foregroundExecutor ForegroundExecutor, frameCallback func(frame ScreenCaptureFrame)) *scheduler.OneshotReceiver[scheduler.Result[ScreenCaptureStream]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[ScreenCaptureStream]]()
	stream := TestScreenCaptureStream{}
	_ = tx.Send(scheduler.Result[ScreenCaptureStream]{Value: stream})
	return rx
}

// Metadata returns the metadata of TestScreenCaptureSource.
func (TestScreenCaptureStream) Metadata() (SourceMetadata, error) {
	return TestScreenCaptureSource{}.Metadata()
}

// Stop does nothing: the stream captures nothing.
func (TestScreenCaptureStream) Stop() {}

type testPrompt struct {
	msg     string
	detail  Option[string]
	answers []string
	tx      *scheduler.OneshotSender[int]
}

type testSystemNotifications struct {
	appIdentity      Option[[2]SharedString]
	shown            []SystemNotification
	delivered        []SystemNotification
	dismissed        []SharedString
	responseCallback func(response SystemNotificationResponse)
}

type testNewPathPrompt struct {
	path string
	tx   *scheduler.OneshotSender[scheduler.Result[Option[string]]]
}

type testPathsPrompt struct {
	options PathPromptOptions
	tx      *scheduler.OneshotSender[scheduler.Result[Option[[]string]]]
}

type testPrompts struct {
	multipleChoice []testPrompt
	newPath        []testNewPathPrompt
	paths          []testPathsPrompt
}

// CursorStyle returns the cursor style last set on the platform.
func (p *TestPlatform) CursorStyle() CursorStyle {
	p.activeCursorMu.Lock()
	defer p.activeCursorMu.Unlock()
	return p.activeCursor
}

// NewTestPlatform returns a test platform with a NoopTextSystem.
func NewTestPlatform(executor BackgroundExecutor, foregroundExecutor ForegroundExecutor) *TestPlatform {
	return TestPlatformWithPlatform(executor, foregroundExecutor, NewNoopTextSystem(), nil)
}

// TestPlatformWithTextSystem returns a test platform with the given text system.
func TestPlatformWithTextSystem(executor BackgroundExecutor, foregroundExecutor ForegroundExecutor, textSystem PlatformTextSystem) *TestPlatform {
	return TestPlatformWithPlatform(executor, foregroundExecutor, textSystem, nil)
}

// SimulateDisplayAdded connects a display and reports the change.
func (p *TestPlatform) SimulateDisplayAdded(displayID DisplayID, refreshInterval Option[time.Duration]) {
	p.ConnectDisplay(displayID, refreshInterval)
	p.reportDisplaysChanged()
}

// ConnectDisplay adds a display without telling App yet, as when the platform's
// notifications about a window arrive before those about displays.
func (p *TestPlatform) ConnectDisplay(displayID DisplayID, refreshInterval Option[time.Duration]) {
	display := TestDisplayWithID(displayID)
	display.refreshInterval = refreshInterval
	p.displays = append(p.displays, display)
}

// SimulateDisplayRemoved removes the display and reports the change.
func (p *TestPlatform) SimulateDisplayRemoved(displayID DisplayID) {
	p.displays = slices.DeleteFunc(p.displays, func(display *TestDisplay) bool {
		return display.ID() == displayID
	})
	p.reportDisplaysChanged()
}

// SimulateDisplayRefreshIntervalChange changes the display's refresh interval and reports the change.
func (p *TestPlatform) SimulateDisplayRefreshIntervalChange(displayID DisplayID, refreshInterval Option[time.Duration]) {
	for _, display := range p.displays {
		if display.ID() == displayID {
			display.refreshInterval = refreshInterval
		}
	}
	p.reportDisplaysChanged()
}

func (p *TestPlatform) reportDisplaysChanged() {
	callback := p.displaysChangedCallback
	p.displaysChangedCallback = nil
	if callback != nil {
		callback()
		if p.displaysChangedCallback == nil {
			p.displaysChangedCallback = callback
		}
	}
}

// TestPlatformWithPlatform returns a test platform; headlessRendererFactory may be nil, and a
// factory returning a nil renderer means none.
func TestPlatformWithPlatform(executor BackgroundExecutor, foregroundExecutor ForegroundExecutor, textSystem PlatformTextSystem, headlessRendererFactory func() (PlatformHeadlessRenderer, error)) *TestPlatform {
	p := &TestPlatform{
		platformInfo:             testPlatformInfo(),
		backgroundExecutor:       executor,
		foregroundExecutor:       foregroundExecutor,
		displays:                 []*TestDisplay{NewTestDisplay()},
		idleSleepPreventionCount: new(atomic.Int64),
		textSystem:               textSystem,
		headlessRendererFactory:  headlessRendererFactory,
	}
	p.weak = weak.Make(p)
	BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

func testPlatformInfo() PlatformInfo {
	info := PlatformInfo{Target: PlatformTargetNative}
	switch runtime.GOOS {
	case "darwin":
		info.Os = OperatingSystemMac
	case "windows":
		info.Os = OperatingSystemWindows
	case "linux":
		info.Os = OperatingSystemLinux
	case "freebsd":
		info.Os = OperatingSystemFreeBsd
	case "ios":
		info.Os = OperatingSystemIos
	case "android":
		info.Os = OperatingSystemAndroid
	case "js", "wasip1":
		info.Os = OperatingSystemUnknown
		info.Target = PlatformTargetWeb
	default:
		panic("unsupported test platform operating system")
	}
	return info
}

// SimulateNewPathSelection answers the oldest pending new-path prompt with selectPath's result.
func (p *TestPlatform) SimulateNewPathSelection(selectPath func(path string) Option[string]) {
	if len(p.prompts.newPath) == 0 {
		panic("no pending new path prompt")
	}
	prompt := p.prompts.newPath[0]
	p.prompts.newPath = p.prompts.newPath[1:]
	_ = prompt.tx.Send(scheduler.Result[Option[string]]{Value: selectPath(prompt.path)})
}

// SimulatePathPromptResponse answers the oldest pending paths prompt with selectPaths' result.
func (p *TestPlatform) SimulatePathPromptResponse(selectPaths func(options *PathPromptOptions) Option[[]string]) {
	if len(p.prompts.paths) == 0 {
		panic("no pending paths prompt")
	}
	prompt := p.prompts.paths[0]
	p.prompts.paths = p.prompts.paths[1:]
	selection := selectPaths(&prompt.options)
	if paths, ok := selection.Get(); ok && !prompt.options.Multiple && len(paths) > 1 {
		panic(fmt.Sprintf("selected %d paths for a prompt that does not allow multiple selection", len(paths)))
	}
	_ = prompt.tx.Send(scheduler.Result[Option[[]string]]{Value: selection})
}

// DidPromptForPaths reports whether a paths prompt is pending.
func (p *TestPlatform) DidPromptForPaths() bool {
	return len(p.prompts.paths) != 0
}

// SimulatePromptAnswer answers the oldest pending multiple-choice prompt with the button labelled response.
func (p *TestPlatform) SimulatePromptAnswer(response string) {
	if len(p.prompts.multipleChoice) == 0 {
		panic("no pending multiple choice prompt")
	}
	prompt := p.prompts.multipleChoice[0]
	p.prompts.multipleChoice = p.prompts.multipleChoice[1:]
	ix := slices.Index(prompt.answers, response)
	if ix < 0 {
		detail := "None"
		if d, ok := prompt.detail.Get(); ok {
			detail = fmt.Sprintf("Some(%q)", d)
		}
		panic(fmt.Sprintf("PROMPT: %s\n%s\n%q\nCannot respond with %s", prompt.msg, detail, prompt.answers, response))
	}
	_ = prompt.tx.Send(ix)
}

// HasPendingPrompt reports whether a multiple-choice prompt is pending.
func (p *TestPlatform) HasPendingPrompt() bool {
	return len(p.prompts.multipleChoice) != 0
}

// PendingPrompt returns the message and detail of the oldest pending multiple-choice prompt.
func (p *TestPlatform) PendingPrompt() (msg string, detail string, ok bool) {
	if len(p.prompts.multipleChoice) == 0 {
		return "", "", false
	}
	prompt := p.prompts.multipleChoice[0]
	return prompt.msg, prompt.detail.UnwrapOrDefault(), true
}

// SetScreenCaptureSources sets the sources ScreenCaptureSources reports.
func (p *TestPlatform) SetScreenCaptureSources(sources []TestScreenCaptureSource) {
	p.screenCaptureSources = sources
}

// prompt queues the prompt so a test can later inspect or answer it through
// PendingPrompt and SimulatePromptAnswer.
func (p *TestPlatform) prompt(msg string, detail Option[string], answers []PromptButton) *scheduler.OneshotReceiver[int] {
	tx, rx := scheduler.Oneshot[int]()
	labels := make([]string, 0, len(answers))
	for _, answer := range answers {
		labels = append(labels, string(answer.Label()))
	}
	p.prompts.multipleChoice = append(p.prompts.multipleChoice, testPrompt{
		msg:     msg,
		detail:  detail,
		answers: labels,
		tx:      tx,
	})
	return rx
}

// SetActiveWindow makes window the active window and reports the activation changes from a
// foreground task.
func (p *TestPlatform) SetActiveWindow(window *TestWindow) {
	executor := p.ForegroundExecutor()
	previousWindow := p.activeWindow
	p.activeWindow = nil
	if window != nil {
		p.activeWindow = window.Clone()
	}

	executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if previousWindow != nil {
			if window != nil && previousWindow.state == window.state {
				return struct{}{}, nil
			}
			previousWindow.simulateActiveStatusChange(false)
		}
		if window != nil {
			window.simulateActiveStatusChange(true)
		}
		return struct{}{}, nil
	}).Detach()
}

// DidPromptForNewPath reports whether a new-path prompt is pending.
func (p *TestPlatform) DidPromptForNewPath() bool {
	return len(p.prompts.newPath) != 0
}

// ActiveIdleSleepPreventions returns how many idle sleep preventions are held.
func (p *TestPlatform) ActiveIdleSleepPreventions() int {
	return int(p.idleSleepPreventionCount.Load())
}

// SetIdleSleepPreventionDelay makes PreventIdleSleep acquire after delay.
func (p *TestPlatform) SetIdleSleepPreventionDelay(delay time.Duration) {
	p.idleSleepPreventionDelay = delay
}

// SetIdleSleepPreventionFails makes PreventIdleSleep fail.
func (p *TestPlatform) SetIdleSleepPreventionFails(fails bool) {
	p.idleSleepPreventionFails = fails
}

// AppIdentity returns the identifier and name set with SetAppIdentity.
func (p *TestPlatform) AppIdentity() (identifier SharedString, name SharedString, ok bool) {
	identity, ok := p.systemNotifications.appIdentity.Get()
	return identity[0], identity[1], ok
}

// ShownSystemNotifications returns every notification shown, in order.
func (p *TestPlatform) ShownSystemNotifications() []SystemNotification {
	return slices.Clone(p.systemNotifications.shown)
}

// DeliveredSystemNotifications returns the notifications still delivered.
func (p *TestPlatform) DeliveredSystemNotifications() []SystemNotification {
	return slices.Clone(p.systemNotifications.delivered)
}

// DismissedSystemNotifications returns the tags of dismissed notifications, in order.
func (p *TestPlatform) DismissedSystemNotifications() []SharedString {
	return slices.Clone(p.systemNotifications.dismissed)
}

// SimulateSystemNotificationResponse delivers response to the registered response callback.
func (p *TestPlatform) SimulateSystemNotificationResponse(response SystemNotificationResponse) {
	callback := p.systemNotifications.responseCallback
	p.systemNotifications.responseCallback = nil
	if callback != nil {
		callback(response)
		if p.systemNotifications.responseCallback == nil {
			p.systemNotifications.responseCallback = callback
		}
	}
}

// PlatformInfo returns the host operating system and target.
func (p *TestPlatform) PlatformInfo() PlatformInfo {
	return p.platformInfo
}

// BackgroundExecutor returns the background executor.
func (p *TestPlatform) BackgroundExecutor() BackgroundExecutor {
	return p.backgroundExecutor
}

// ForegroundExecutor returns the foreground executor.
func (p *TestPlatform) ForegroundExecutor() ForegroundExecutor {
	return p.foregroundExecutor
}

// TextSystem returns the platform text system.
func (p *TestPlatform) TextSystem() PlatformTextSystem {
	return p.textSystem
}

// KeyboardLayout returns the test keyboard layout.
func (p *TestPlatform) KeyboardLayout() PlatformKeyboardLayout {
	return testKeyboardLayout{}
}

// KeyboardMapper returns a DummyKeyboardMapper.
func (p *TestPlatform) KeyboardMapper() PlatformKeyboardMapper {
	return DummyKeyboardMapper{}
}

// OnKeyboardLayoutChange ignores the callback.
func (p *TestPlatform) OnKeyboardLayoutChange(callback func()) {}

// OnThermalStateChange ignores the callback.
func (p *TestPlatform) OnThermalStateChange(callback func()) {}

// ThermalState returns ThermalStateNominal.
func (p *TestPlatform) ThermalState() ThermalState {
	return ThermalStateNominal
}

// PreventIdleSleep counts an idle sleep prevention until the returned guard is released.
func (p *TestPlatform) PreventIdleSleep(reason string) *Task[*ActivityGuard] {
	count := p.idleSleepPreventionCount
	fails := p.idleSleepPreventionFails
	acquire := func() (*ActivityGuard, error) {
		if fails {
			return nil, fmt.Errorf("Idle sleep prevention for %q is set to fail in this test", reason)
		}
		count.Add(1)
		return NewActivityGuard(func() {
			count.Add(-1)
		}), nil
	}

	delay := p.idleSleepPreventionDelay
	if delay == 0 {
		guard, err := acquire()
		if err != nil {
			return scheduler.ReadyTaskErr[*ActivityGuard](err)
		}
		return scheduler.ReadyTask(guard)
	}
	timer := p.backgroundExecutor.Timer(delay)
	return p.foregroundExecutor.Spawn(func(s *scheduler.Scope) (*ActivityGuard, error) {
		_, _ = timer.Await(s)
		return acquire()
	})
}

// Run is not implemented by the test platform.
func (p *TestPlatform) Run(onFinishLaunching func()) {
	panic("not implemented")
}

// Quit does nothing.
func (p *TestPlatform) Quit() {}

// Restart sends the request to the expected-restart receiver, if there is one.
func (p *TestPlatform) Restart(binaryPath Option[string], arguments []string) {
	tx := p.expectRestart
	p.expectRestart = nil
	if tx != nil {
		if err := tx.Send(TestRestartRequest{BinaryPath: binaryPath, Arguments: arguments}); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
	}
}

// Activate does nothing.
func (p *TestPlatform) Activate(ignoringOtherApps bool) {}

// Hide is not implemented by the test platform.
func (p *TestPlatform) Hide() {
	panic("not implemented")
}

// HideOtherApps is not implemented by the test platform.
func (p *TestPlatform) HideOtherApps() {
	panic("not implemented")
}

// UnhideOtherApps is not implemented by the test platform.
func (p *TestPlatform) UnhideOtherApps() {
	panic("not implemented")
}

// Displays returns copies of the connected displays.
func (p *TestPlatform) Displays() []PlatformDisplay {
	displays := make([]PlatformDisplay, 0, len(p.displays))
	for _, display := range p.displays {
		displays = append(displays, display.Clone())
	}
	return displays
}

// PrimaryDisplay returns a copy of the first display, or nil.
func (p *TestPlatform) PrimaryDisplay() PlatformDisplay {
	if len(p.displays) == 0 {
		return nil
	}
	return p.displays[0].Clone()
}

// OnDisplaysChanged registers the callback for display changes.
func (p *TestPlatform) OnDisplaysChanged(callback func()) {
	p.displaysChangedCallback = callback
}

// IsScreenCaptureSupported returns true.
func (p *TestPlatform) IsScreenCaptureSupported() bool {
	return true
}

// ScreenCaptureSources returns a receiver holding the sources set with SetScreenCaptureSources.
func (p *TestPlatform) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[[]ScreenCaptureSource]]()
	sources := make([]ScreenCaptureSource, 0, len(p.screenCaptureSources))
	for _, source := range p.screenCaptureSources {
		sources = append(sources, source)
	}
	_ = tx.Send(scheduler.Result[[]ScreenCaptureSource]{Value: sources})
	return rx
}

// ActiveWindow returns the handle of the active window.
func (p *TestPlatform) ActiveWindow() (AnyWindowHandle, bool) {
	if p.activeWindow == nil {
		return AnyWindowHandle{}, false
	}
	p.activeWindow.state.mu.Lock()
	defer p.activeWindow.state.mu.Unlock()
	return p.activeWindow.state.handle, true
}

// OpenWindow opens a TestWindow on the primary display.
func (p *TestPlatform) OpenWindow(handle AnyWindowHandle, params WindowParams) (PlatformWindow, error) {
	var renderer PlatformHeadlessRenderer
	if p.headlessRendererFactory != nil {
		r, err := p.headlessRendererFactory()
		if err != nil {
			return nil, err
		}
		renderer = r
	}
	display := p.PrimaryDisplay()
	if display == nil {
		display = NewTestDisplay()
	}
	window := NewTestWindow(handle, params, p.weak, display, renderer)
	return window, nil
}

// WindowAppearance returns WindowAppearanceLight.
func (p *TestPlatform) WindowAppearance() WindowAppearance {
	return WindowAppearanceLight
}

// OpenURL records the URL.
func (p *TestPlatform) OpenURL(url string) {
	p.openedURL = Some(url)
}

// OnOpenURLs is not implemented by the test platform.
func (p *TestPlatform) OnOpenURLs(callback func(urls []string)) {
	panic("not implemented")
}

// PromptForPaths queues the prompt so a test can later answer it through SimulatePathPromptResponse.
func (p *TestPlatform) PromptForPaths(options PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[Option[[]string]]]()
	p.prompts.paths = append(p.prompts.paths, testPathsPrompt{options: options, tx: tx})
	return rx
}

// PromptForNewPath queues the prompt so a test can later answer it through SimulateNewPathSelection.
func (p *TestPlatform) PromptForNewPath(directory string, suggestedName Option[string]) *scheduler.OneshotReceiver[scheduler.Result[Option[string]]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[Option[string]]]()
	p.prompts.newPath = append(p.prompts.newPath, testNewPathPrompt{path: directory, tx: tx})
	return rx
}

// CanSelectMixedFilesAndDirs returns true.
func (p *TestPlatform) CanSelectMixedFilesAndDirs() bool {
	return true
}

// RevealPath is not implemented by the test platform.
func (p *TestPlatform) RevealPath(path string) {
	panic("not implemented")
}

// OnQuit ignores the callback.
func (p *TestPlatform) OnQuit(callback func() bool) {}

// OnReopen is not implemented by the test platform.
func (p *TestPlatform) OnReopen(callback func()) {
	panic("not implemented")
}

// OnSystemSleep ignores the callback.
func (p *TestPlatform) OnSystemSleep(callback func()) {}

// OnSystemWake ignores the callback.
func (p *TestPlatform) OnSystemWake(callback func()) {}

// OnAppLifecycle registers the lifecycle callback.
func (p *TestPlatform) OnAppLifecycle(callback func(phase AppLifecyclePhase)) {
	p.appLifecycleCallback = callback
}

// OnMemoryWarning registers the memory warning callback.
func (p *TestPlatform) OnMemoryWarning(callback func()) {
	p.memoryWarningCallback = callback
}

// SetAppIdentity records the identity notifications are posted under.
func (p *TestPlatform) SetAppIdentity(identifier string, name string) {
	p.systemNotifications.appIdentity = Some([2]SharedString{SharedString(identifier), SharedString(name)})
}

// ShowSystemNotification records the notification once an app identity is set.
func (p *TestPlatform) ShowSystemNotification(notification SystemNotification) {
	n := &p.systemNotifications
	if !n.appIdentity.IsSome() {
		return
	}
	ix := slices.IndexFunc(n.delivered, func(delivered SystemNotification) bool {
		return delivered.Tag == notification.Tag
	})
	if ix >= 0 {
		n.delivered[ix] = notification
	} else {
		n.delivered = append(n.delivered, notification)
	}
	n.shown = append(n.shown, notification)
}

// DismissSystemNotification removes the delivered notification with this tag and records the dismissal.
func (p *TestPlatform) DismissSystemNotification(tag string) *Task[struct{}] {
	n := &p.systemNotifications
	n.delivered = slices.DeleteFunc(n.delivered, func(notification SystemNotification) bool {
		return string(notification.Tag) == tag
	})
	n.dismissed = append(n.dismissed, SharedString(tag))
	return scheduler.ReadyTask(struct{}{})
}

// OnSystemNotificationResponse registers the notification response callback.
func (p *TestPlatform) OnSystemNotificationResponse(callback func(response SystemNotificationResponse)) {
	p.systemNotifications.responseCallback = callback
}

// SetMenus records the menus.
func (p *TestPlatform) SetMenus(menus []Menu, keymap *Keymap) {
	owned := make([]OwnedMenu, 0, len(menus))
	for _, menu := range menus {
		owned = append(owned, menu.Owned())
	}
	p.menus = owned
}

// GetMenus returns the menus set with SetMenus.
func (p *TestPlatform) GetMenus() ([]OwnedMenu, bool) {
	return slices.Clone(p.menus), true
}

// SetDockMenu ignores the menu.
func (p *TestPlatform) SetDockMenu(menu []MenuItem, keymap *Keymap) {}

// AddRecentDocument ignores the path.
func (p *TestPlatform) AddRecentDocument(path string) {}

// OnAppMenuAction ignores the callback.
func (p *TestPlatform) OnAppMenuAction(callback func(action Action)) {}

// OnWillOpenAppMenu ignores the callback.
func (p *TestPlatform) OnWillOpenAppMenu(callback func()) {}

// OnValidateAppMenuCommand ignores the callback.
func (p *TestPlatform) OnValidateAppMenuCommand(callback func(action Action) bool) {}

// AppPath is not implemented by the test platform.
func (p *TestPlatform) AppPath() (string, error) {
	panic("not implemented")
}

// PathForAuxiliaryExecutable is not implemented by the test platform.
func (p *TestPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	panic("not implemented")
}

// SetCursorStyle records the cursor style.
func (p *TestPlatform) SetCursorStyle(style CursorStyle) {
	p.activeCursorMu.Lock()
	defer p.activeCursorMu.Unlock()
	p.activeCursor = style
}

// HideCursorUntilMouseMoves does nothing.
func (p *TestPlatform) HideCursorUntilMouseMoves() {}

// IsCursorVisible returns true.
func (p *TestPlatform) IsCursorVisible() bool {
	return true
}

// ShouldAutoHideScrollbars returns false.
func (p *TestPlatform) ShouldAutoHideScrollbars() bool {
	return false
}

// ReadFromClipboard returns the item last written to the clipboard.
func (p *TestPlatform) ReadFromClipboard() (ClipboardItem, bool) {
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	return p.currentClipboardItem.Get()
}

// WriteToClipboard stores the item.
func (p *TestPlatform) WriteToClipboard(item ClipboardItem) {
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	p.currentClipboardItem = Some(item)
}

func testPlatformHasPrimary() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "freebsd"
}

// ReadFromPrimary returns the item last written to the primary selection (Linux and FreeBSD).
func (p *TestPlatform) ReadFromPrimary() (ClipboardItem, bool) {
	if !testPlatformHasPrimary() {
		return p.PlatformDefaults.ReadFromPrimary()
	}
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	return p.currentPrimaryItem.Get()
}

// WriteToPrimary stores the item in the primary selection (Linux and FreeBSD).
func (p *TestPlatform) WriteToPrimary(item ClipboardItem) {
	if !testPlatformHasPrimary() {
		p.PlatformDefaults.WriteToPrimary(item)
		return
	}
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	p.currentPrimaryItem = Some(item)
}

// ReadFromFindPasteboard returns the item last written to the find pasteboard (macOS).
func (p *TestPlatform) ReadFromFindPasteboard() (ClipboardItem, bool) {
	if runtime.GOOS != "darwin" {
		return p.PlatformDefaults.ReadFromFindPasteboard()
	}
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	return p.currentFindPasteboard.Get()
}

// WriteToFindPasteboard stores the item in the find pasteboard (macOS).
func (p *TestPlatform) WriteToFindPasteboard(item ClipboardItem) {
	if runtime.GOOS != "darwin" {
		p.PlatformDefaults.WriteToFindPasteboard(item)
		return
	}
	p.clipboardMu.Lock()
	defer p.clipboardMu.Unlock()
	p.currentFindPasteboard = Some(item)
}

// WriteCredentials succeeds without storing anything.
func (p *TestPlatform) WriteCredentials(url string, username string, password []byte) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// ReadCredentials finds no credentials.
func (p *TestPlatform) ReadCredentials(url string) *Task[Option[Credentials]] {
	return scheduler.ReadyTask(None[Credentials]())
}

// DeleteCredentials succeeds without deleting anything.
func (p *TestPlatform) DeleteCredentials(url string) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// RegisterURLScheme is not implemented by the test platform.
func (p *TestPlatform) RegisterURLScheme(url string) *Task[struct{}] {
	panic("not implemented")
}

// OpenWithSystem is not implemented by the test platform.
func (p *TestPlatform) OpenWithSystem(path string) {
	panic("not implemented")
}

// NewTestScreenCaptureSource creates a fake screen capture source, for testing.
func NewTestScreenCaptureSource() TestScreenCaptureSource {
	return TestScreenCaptureSource{}
}

type testKeyboardLayout struct{}

func (testKeyboardLayout) ID() string {
	return "zed.keyboard.example"
}

func (testKeyboardLayout) Name() string {
	return "zed.keyboard.example"
}

var (
	_ Platform               = (*TestPlatform)(nil)
	_ ScreenCaptureSource    = TestScreenCaptureSource{}
	_ ScreenCaptureStream    = TestScreenCaptureStream{}
	_ PlatformKeyboardLayout = testKeyboardLayout{}
	_ PlatformWindow         = (*TestWindow)(nil)
	_ PlatformDisplay        = (*TestDisplay)(nil)
	_ NativeViewAttachment   = (*testNativeViewAttachment)(nil)
)
