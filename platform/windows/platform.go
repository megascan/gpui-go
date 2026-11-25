//go:build windows

package windows

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
	"weak"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/d3d"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/scheduler"
)

type trackedWindow struct {
	handle      SafeHwnd
	frameSignal *gpui.PlatformFrameSignal
}

func (w *trackedWindow) asRaw() HWND {
	return w.handle.AsRaw()
}

type trackedWindows struct {
	mu      sync.RWMutex
	windows []trackedWindow
	dropped atomic.Bool
}

// WindowsPlatform is the Win32 Platform. Close is its drop.
type WindowsPlatform struct {
	gpui.PlatformDefaults
	inner                     *windowsPlatformInner
	rawWindowHandles          *trackedWindows
	initialWindowing          gpui.Option[gpui.WindowingRequest]
	vsyncStop                 *atomic.Bool
	headless                  bool
	icon                      win32.HICON
	backgroundExecutor        gpui.BackgroundExecutor
	foregroundExecutor        gpui.ForegroundExecutor
	textSystem                gpui.PlatformTextSystem
	dropTargetHelper          *IDropTargetHelper
	invalidateDevices         *atomic.Bool
	handle                    HWND
	suspendResumeNotification gpui.Option[win32.HPOWERNOTIFY]
	powerSavingNotification   gpui.Option[win32.HPOWERNOTIFY]
	disableDirectComposition  bool
	hasPackageIdentity        bool
	appIdentity               gpui.Option[appIdentity]
	systemNotifications       *SystemNotificationState
	closed                    bool
}

var _ gpui.Platform = (*WindowsPlatform)(nil)

type windowsPlatformInner struct {
	state            WindowsPlatformState
	rawWindowHandles *trackedWindows
	validationNumber uintptr
	mainReceiver     *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	dispatcher       *WindowsDispatcher
}

// WindowsPlatformState is the platform state the platform window procedure mutates.
type WindowsPlatformState struct {
	callbacks           platformCallbacks
	menus               []gpui.OwnedMenu
	jumpList            JumpList
	currentCursor       win32.HCURSOR
	cursorVisible       *atomic.Bool
	drawCoordinator     *DrawCoordinator
	directxDevices      *DirectXDevices
	vsyncDevices        *DirectXDevices
	lowPowerModeEnabled bool
}

type platformCallbacks struct {
	openURLs               gpui.Option[func([]string)]
	quit                   gpui.Option[func() bool]
	reopen                 gpui.Option[func()]
	appMenuAction          gpui.Option[func(gpui.Action)]
	willOpenAppMenu        gpui.Option[func()]
	validateAppMenuCommand gpui.Option[func(gpui.Action) bool]
	keyboardLayoutChange   gpui.Option[func()]
	systemSleep            gpui.Option[func()]
	systemWake             gpui.Option[func()]
	lowPowerModeChange     gpui.Option[func()]
	displaysChanged        gpui.Option[func()]
}

func newWindowsPlatformState(directxDevices *DirectXDevices) WindowsPlatformState {
	currentCursor, _ := loadCursor(gpui.CursorStyleArrow)
	cursorVisible := &atomic.Bool{}
	cursorVisible.Store(true)
	return WindowsPlatformState{
		jumpList:        NewJumpList(),
		currentCursor:   currentCursor,
		cursorVisible:   cursorVisible,
		drawCoordinator: NewDrawCoordinator(),
		directxDevices:  directxDevices,
	}
}

const powerRequestSystemRequired int32 = 1

type powerRequest struct {
	handle win32.HANDLE
	reason []uint16
}

func powerRequestPreventIdleSleep(reason string) (*powerRequest, error) {
	reasonUTF16 := append(utf16.Encode([]rune(reason)), 0)
	context := win32.REASON_CONTEXT{
		Version:            win32.POWER_REQUEST_CONTEXT_VERSION,
		Flags:              win32.POWER_REQUEST_CONTEXT_SIMPLE_STRING,
		SimpleReasonString: &reasonUTF16[0],
	}
	handle, err := win32.PowerCreateRequest(&context)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a Windows power request: %w", err)
	}
	if err := win32.PowerSetRequest(handle, powerRequestSystemRequired); err != nil {
		if closeErr := xwin.CloseHandle(xwin.Handle(handle)); closeErr != nil {
			gpuiutil.LogErr(fmt.Errorf("Failed to close the Windows power request: %w", closeErr))
		}
		return nil, fmt.Errorf("Failed to set the Windows power request: %w", err)
	}
	return &powerRequest{handle: handle, reason: reasonUTF16}, nil
}

// Close is the power request's drop: it clears and closes the request.
func (r *powerRequest) Close() {
	if err := win32.PowerClearRequest(r.handle, powerRequestSystemRequired); err != nil {
		gpuiutil.LogErr(fmt.Errorf("Failed to clear the Windows power request: %w", err))
	}
	if err := xwin.CloseHandle(xwin.Handle(r.handle)); err != nil {
		gpuiutil.LogErr(fmt.Errorf("Failed to close the Windows power request: %w", err))
	}
	runtime.KeepAlive(r.reason)
}

// NewWindowsPlatform creates the platform on the calling (main) thread.
func NewWindowsPlatform(headless bool) (*WindowsPlatform, error) {
	if err := com.OleInitialize(); err != nil {
		return nil, fmt.Errorf("unable to initialize Windows OLE: %w", err)
	}
	var textSystem gpui.PlatformTextSystem
	if !headless {
		directWrite, err := newDirectWriteTextSystem()
		if err != nil {
			com.OleUninitialize()
			return nil, fmt.Errorf("Error creating DirectWriteTextSystem: %w", err)
		}
		textSystem = directWrite
	} else {
		textSystem = gpui.NewNoopTextSystem()
	}

	mainSender, mainReceiver := gpui.NewPriorityQueueReceiver[gpui.RunnableVariant]()
	var validationNumber uintptr
	if unsafe.Sizeof(uintptr(0)) == 8 {
		validationNumber = uintptr(rand.Uint64())
	} else {
		validationNumber = uintptr(rand.Uint32())
	}
	rawWindowHandles := &trackedWindows{}

	registerPlatformWindowClass()
	context := &platformWindowCreateContext{
		rawWindowHandles: rawWindowHandles,
		validationNumber: validationNumber,
		mainSender:       gpui.Some(mainSender),
		mainReceiver:     gpui.Some(mainReceiver),
	}
	platformCreateContextsMu.Lock()
	nextPlatformCreateContext++
	contextID := nextPlatformCreateContext
	platformCreateContexts[contextID] = context
	platformCreateContextsMu.Unlock()
	handle, createErr := win32.CreateWindowExW(
		win32.WS_EX_TOOLWINDOW,
		platformWindowClassName,
		nil,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		contextID,
	)
	platformCreateContextsMu.Lock()
	delete(platformCreateContexts, contextID)
	platformCreateContextsMu.Unlock()
	fail := func(err error) (*WindowsPlatform, error) {
		if handle != 0 {
			gpuiutil.LogErr(win32.DestroyWindow(handle))
		}
		rawWindowHandles.dropped.Store(true)
		if closer, ok := textSystem.(interface{ Close() }); ok {
			closer.Close()
		}
		com.OleUninitialize()
		return nil, err
	}
	if createErr != nil {
		return fail(createErr)
	}
	if !context.innerSet {
		return fail(errors.New("CreateWindowExW did not run correctly"))
	}
	if context.innerErr != nil {
		return fail(context.innerErr)
	}
	inner := context.inner
	var powerStatus win32.SYSTEM_POWER_STATUS
	if err := win32.GetSystemPowerStatus(&powerStatus); err != nil {
		slog.Warn(fmt.Sprintf("Failed to query Windows battery saver status: %v", err))
	} else {
		inner.state.lowPowerModeEnabled = powerStatus.SystemStatusFlag == 1
	}
	dispatcher := context.dispatcher
	if dispatcher == nil {
		return fail(errors.New("CreateWindowExW did not run correctly"))
	}

	disableValue, ok := os.LookupEnv(DISABLE_DIRECT_COMPOSITION)
	disableDirectComposition := ok && (disableValue == "true" || disableValue == "1")
	backgroundExecutor := gpui.NewBackgroundExecutor(dispatcher)
	foregroundExecutor := gpui.NewForegroundExecutor(dispatcher)

	var dropTargetHelper *IDropTargetHelper
	if !headless {
		helper, err := com.CreateInstance[IDropTargetHelper](&clsidDragDropHelper, win32.CLSCTX_INPROC_SERVER, &iidIDropTargetHelper)
		if err != nil {
			return fail(fmt.Errorf("Error creating drop target helper.: %w", err))
		}
		dropTargetHelper = helper
	}
	var icon win32.HICON
	if !headless {
		if loaded, err := loadIcon(); err == nil {
			icon = loaded
		}
	}

	p := &WindowsPlatform{
		inner:                    inner,
		handle:                   handle,
		rawWindowHandles:         rawWindowHandles,
		headless:                 headless,
		icon:                     icon,
		backgroundExecutor:       backgroundExecutor,
		foregroundExecutor:       foregroundExecutor,
		textSystem:               textSystem,
		disableDirectComposition: disableDirectComposition,
		hasPackageIdentity:       hasPackageIdentity(),
		dropTargetHelper:         dropTargetHelper,
		invalidateDevices:        &atomic.Bool{},
		systemNotifications:      NewSystemNotificationState(),
	}
	gpui.BindPlatformDefaults(&p.PlatformDefaults, p)
	return p, nil
}

func (p *WindowsPlatform) windowFromHwnd(hwnd HWND) (*WindowsWindowInner, bool) {
	return p.inner.windowFromHwnd(hwnd)
}

func (p *WindowsPlatform) postMessage(message uint32, wparam win32.WPARAM, lparam win32.LPARAM) {
	p.rawWindowHandles.mu.RLock()
	defer p.rawWindowHandles.mu.RUnlock()
	for i := range p.rawWindowHandles.windows {
		gpuiutil.LogErr(win32.PostMessageW(p.rawWindowHandles.windows[i].asRaw(), message, wparam, lparam))
	}
}

func (p *WindowsPlatform) generateCreationInfo() WindowCreationInfo {
	return WindowCreationInfo{
		icon:                     p.icon,
		executor:                 p.foregroundExecutor,
		currentCursor:            p.inner.state.currentCursor,
		cursorVisible:            p.inner.state.cursorVisible,
		dropTargetHelper:         p.dropTargetHelper,
		validationNumber:         p.inner.validationNumber,
		mainReceiver:             p.inner.mainReceiver,
		platformWindowHandle:     p.handle,
		disableDirectComposition: p.disableDirectComposition,
		directxDevices:           p.inner.state.directxDevices,
		invalidateDevices:        p.invalidateDevices,
		drawCoordinator:          p.inner.state.drawCoordinator,
	}
}

func dockMenuItems(menus []gpui.MenuItem) []DockMenuItem {
	var actions []DockMenuItem
	for _, menu := range menus {
		dockMenu, err := NewDockMenuItem(menu)
		if err != nil {
			gpuiutil.LogErr(err)
			continue
		}
		actions = append(actions, dockMenu)
	}
	return actions
}

func dockMenuEntries(items []DockMenuItem) []dockMenuEntry {
	entries := make([]dockMenuEntry, 0, len(items))
	for _, menu := range items {
		entries = append(entries, dockMenuEntry{name: menu.name, description: menu.description})
	}
	return entries
}

func cloneWorkspaces(entries [][]string) [][]string {
	cloned := make([][]string, len(entries))
	for i, entry := range entries {
		cloned[i] = slices.Clone(entry)
	}
	return cloned
}

func (p *WindowsPlatform) setDockMenus(menus []gpui.MenuItem) {
	actions := dockMenuItems(menus)
	state := &p.inner.state
	state.jumpList.dockMenus = actions
	dockMenus := dockMenuEntries(state.jumpList.dockMenus)
	recentWorkspaces := cloneWorkspaces(state.jumpList.recentWorkspaces)
	p.backgroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		_, err := updateJumpList(recentWorkspaces, dockMenus)
		gpuiutil.LogErr(err)
		return struct{}{}, nil
	}).Detach()
}

func (p *WindowsPlatform) updateJumpList(menus []gpui.MenuItem, entries [][]string) *gpui.Task[[][]string] {
	actions := dockMenuItems(menus)
	state := &p.inner.state
	state.jumpList.dockMenus = actions
	state.jumpList.recentWorkspaces = entries
	dockMenus := dockMenuEntries(state.jumpList.dockMenus)
	recentWorkspaces := cloneWorkspaces(state.jumpList.recentWorkspaces)
	return p.backgroundExecutor.Spawn(func(s *scheduler.Scope) ([][]string, error) {
		removed, err := updateJumpList(recentWorkspaces, dockMenus)
		if err != nil {
			gpuiutil.LogErr(err)
			return nil, nil
		}
		return removed, nil
	})
}

func (p *WindowsPlatform) findCurrentActiveWindow() (HWND, bool) {
	activeWindowHwnd := win32.GetActiveWindow()
	if activeWindowHwnd == 0 {
		return 0, false
	}
	p.rawWindowHandles.mu.RLock()
	defer p.rawWindowHandles.mu.RUnlock()
	for i := range p.rawWindowHandles.windows {
		if p.rawWindowHandles.windows[i].asRaw() == activeWindowHwnd {
			return activeWindowHwnd, true
		}
	}
	return 0, false
}

func (p *WindowsPlatform) isWindowed() bool {
	return p.inner.state.directxDevices != nil
}

func (p *WindowsPlatform) connectInitially() {
	request, ok := p.initialWindowing.Take().Get()
	if !ok {
		request = gpui.WindowingRequestWindowed{Environment: gpui.GraphicalEnvironmentDetect()}
	}
	windowed, ok := request.(gpui.WindowingRequestWindowed)
	if !ok {
		return
	}
	if err := checkCanShowWindows(windowed.Environment); err != nil {
		slog.Info(fmt.Sprintf("starting headless: %v", err))
		return
	}
	if err := p.attachGPU(); err != nil {
		slog.Error(fmt.Sprintf("failed to initialize DirectX, starting headless: %v", err))
	}
}

func (p *WindowsPlatform) attachGPU() error {
	devices, err := NewDirectXDevices()
	if err != nil {
		return fmt.Errorf("Creating DirectX devices: %w", err)
	}
	p.inner.state.directxDevices = devices
	p.beginVsyncThread()
	return nil
}

func (p *WindowsPlatform) detachGPU() {
	if stop := p.vsyncStop; stop != nil {
		p.vsyncStop = nil
		stop.Store(true)
	}
	p.inner.state.vsyncDevices = nil
	devices := p.inner.state.directxDevices
	p.inner.state.directxDevices = nil
	if devices != nil {
		devices.Release()
	}
}

func (p *WindowsPlatform) beginVsyncThread() {
	if p.inner.state.directxDevices == nil {
		return
	}
	directxDevice := p.inner.state.directxDevices.Clone()
	stop := &atomic.Bool{}
	p.vsyncStop = stop
	p.inner.state.vsyncDevices = directxDevice
	platformWindow := SafeHwndFrom(p.handle)
	validationNumber := p.inner.validationNumber
	allWindows := p.rawWindowHandles
	invalidateDevices := p.invalidateDevices

	go func() {
		defer directxDevice.Release()
		vsyncProvider := NewVSyncProvider()
		for {
			signalSource := vsyncProvider.WaitForVsync()
			signalAt, hasSignal := gpui.PlatformFrameSignalCapture(scheduler.InstantNow)
			if stop.Load() {
				break
			}
			if checkDeviceLost(directxDevice.Device) || invalidateDevices.Swap(false) {
				if err := handleGpuDeviceLost(directxDevice, platformWindow.AsRaw(), validationNumber, allWindows, stop); err != nil {
					slog.Error(fmt.Sprintf("Device lost: %v", err))
					invalidateDevices.Store(true)
					continue
				}
			}
			if allWindows.dropped.Load() {
				break
			}
			allWindows.mu.RLock()
			for i := range allWindows.windows {
				hwnd := &allWindows.windows[i]
				if hasSignal {
					if win32.IsWindowVisible(hwnd.asRaw()) && !win32.IsIconic(hwnd.asRaw()) {
						hwnd.frameSignal.Record(signalAt, signalSource)
					} else {
						hwnd.frameSignal.Take()
					}
				}
				if !win32.RedrawWindow(hwnd.asRaw(), nil, 0, win32.RDW_INVALIDATE) && win32.IsWindow(hwnd.asRaw()) {
					gpuiutil.LogErr(errors.New("RedrawWindow failed"))
				}
			}
			allWindows.mu.RUnlock()
		}
	}()
}

func encodeRestartArguments(arguments []string) string {
	var encoded strings.Builder
	for index, argument := range arguments {
		if index > 0 {
			encoded.WriteByte(' ')
		}
		encoded.WriteByte('"')
		backslashCount := 0
		for _, codeUnit := range argument {
			if codeUnit == '\\' {
				backslashCount++
			} else {
				if codeUnit == '"' {
					encoded.WriteString(strings.Repeat("\\", backslashCount*2+1))
				} else {
					encoded.WriteString(strings.Repeat("\\", backslashCount))
				}
				backslashCount = 0
				encoded.WriteRune(codeUnit)
			}
		}
		encoded.WriteString(strings.Repeat("\\", backslashCount*2))
		encoded.WriteByte('"')
	}
	return encoded.String()
}

// PlatformInfo reports Windows, native.
func (p *WindowsPlatform) PlatformInfo() gpui.PlatformInfo {
	return gpui.PlatformInfo{Os: gpui.OperatingSystemWindows, Target: gpui.PlatformTargetNative}
}

// BackgroundExecutor returns the thread-pool executor.
func (p *WindowsPlatform) BackgroundExecutor() gpui.BackgroundExecutor {
	return p.backgroundExecutor
}

// ForegroundExecutor returns the main-thread executor.
func (p *WindowsPlatform) ForegroundExecutor() gpui.ForegroundExecutor {
	return p.foregroundExecutor
}

// TextSystem returns DirectWrite, or a no-op text system when headless.
func (p *WindowsPlatform) TextSystem() gpui.PlatformTextSystem {
	return p.textSystem
}

// KeyboardLayout returns the active keyboard layout.
func (p *WindowsPlatform) KeyboardLayout() gpui.PlatformKeyboardLayout {
	layout, err := NewWindowsKeyboardLayout()
	if err != nil {
		gpuiutil.LogErr(err)
		return WindowsKeyboardLayoutUnknown()
	}
	return layout
}

// KeyboardMapper returns a mapper for the active keyboard layout.
func (p *WindowsPlatform) KeyboardMapper() gpui.PlatformKeyboardMapper {
	return NewWindowsKeyboardMapper()
}

// OnKeyboardLayoutChange registers the keyboard layout change callback.
func (p *WindowsPlatform) OnKeyboardLayoutChange(callback func()) {
	p.inner.state.callbacks.keyboardLayoutChange = gpui.Some(callback)
}

// OnThermalStateChange ignores the callback: Windows reports no thermal state.
func (p *WindowsPlatform) OnThermalStateChange(callback func()) {}

// ThermalState reports nominal.
func (p *WindowsPlatform) ThermalState() gpui.ThermalState {
	return gpui.ThermalStateNominal
}

// PreventIdleSleep holds a system-required power request until the guard is released.
func (p *WindowsPlatform) PreventIdleSleep(reason string) *gpui.Task[*gpui.ActivityGuard] {
	request, err := powerRequestPreventIdleSleep(reason)
	if err != nil {
		return scheduler.ReadyTaskErr[*gpui.ActivityGuard](err)
	}
	return scheduler.ReadyTask(gpui.NewActivityGuard(request.Close))
}

// Run connects, calls onFinishLaunching and runs the message loop until WM_QUIT.
func (p *WindowsPlatform) Run(onFinishLaunching func()) {
	if !p.headless {
		p.connectInitially()
	}
	onFinishLaunching()

	var msg win32.MSG
	for win32.GetMessageW(&msg, 0, 0, 0) != 0 {
		if !p.inner.translateAccelerator(&msg) {
			win32.TranslateMessage(&msg)
			win32.DispatchMessageW(&msg)
		}
	}

	withCallback(&p.inner.state.callbacks.quit, func(callback func() bool) {
		callback()
	})
}

// Quit posts WM_QUIT from the foreground executor.
func (p *WindowsPlatform) Quit() {
	p.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		win32.PostQuitMessage(0)
		return struct{}{}, nil
	}).Detach()
}

// SetInitialWindowing sets the windowing mode Run starts in.
func (p *WindowsPlatform) SetInitialWindowing(request gpui.WindowingRequest) {
	p.initialWindowing = gpui.Some(request)
}

// RequestWindowing switches between headless and windowed.
func (p *WindowsPlatform) RequestWindowing(request gpui.WindowingRequest) *gpui.Task[struct{}] {
	if p.headless {
		return scheduler.ReadyTaskErr[struct{}](errors.New("a platform created headless can't switch windowing modes"))
	}
	windowed := p.isWindowed()
	var err error
	switch request := request.(type) {
	case gpui.WindowingRequestHeadless:
		if !windowed {
			err = errors.New("already headless")
		} else {
			p.detachGPU()
		}
	case gpui.WindowingRequestWindowed:
		if windowed {
			err = errors.New("already windowed")
		} else if err = checkCanShowWindows(request.Environment); err == nil {
			err = p.attachGPU()
		}
	}
	if err != nil {
		return scheduler.ReadyTaskErr[struct{}](err)
	}
	return scheduler.ReadyTask(struct{}{})
}

// GraphicalEnvironment returns this process's session while windowed.
func (p *WindowsPlatform) GraphicalEnvironment() (gpui.GraphicalEnvironment, bool) {
	if !p.isWindowed() {
		return gpui.GraphicalEnvironment{}, false
	}
	return gpui.GraphicalEnvironmentDetect(), true
}

const restartScript = `
            $pidToWaitFor = $env:ZED_RESTART_PID
            $exePath = $env:ZED_RESTART_EXECUTABLE
            $argumentList = $env:ZED_RESTART_ARGUMENTS

            [Environment]::SetEnvironmentVariable("ZED_RESTART_PID", $null)
            [Environment]::SetEnvironmentVariable("ZED_RESTART_EXECUTABLE", $null)
            [Environment]::SetEnvironmentVariable("ZED_RESTART_ARGUMENTS", $null)

            while ($true) {
                $process = Get-Process -Id $pidToWaitFor -ErrorAction SilentlyContinue
                if (-not $process) {
                    if ([string]::IsNullOrEmpty($argumentList)) {
                        Start-Process -FilePath $exePath
                    } else {
                        Start-Process -FilePath $exePath -ArgumentList $argumentList
                    }
                    break
                }
                Start-Sleep -Seconds 0.1
            }
            `

// Restart waits for this process to exit in PowerShell and starts the app again.
func (p *WindowsPlatform) Restart(binaryPath gpui.Option[string], arguments []string) {
	pid := os.Getpid()
	appPath, ok := binaryPath.Get()
	if !ok {
		path, err := p.AppPath()
		if err != nil {
			gpuiutil.LogErr(err)
			return
		}
		appPath = path
	}
	powershell, ok := gpuiutil.GetPowershell()
	if !ok {
		slog.Error("failed to restart: PowerShell is unavailable")
		return
	}
	arguments = slices.Clone(arguments)
	p.foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		command := gpuiutil.NewStdCommand(powershell, "-command", restartScript)
		command.Env = append(os.Environ(),
			"ZED_RESTART_PID="+strconv.Itoa(pid),
			"ZED_RESTART_EXECUTABLE="+appPath,
			"ZED_RESTART_ARGUMENTS="+encodeRestartArguments(arguments),
		)
		if err := command.Start(); err != nil {
			slog.Error(fmt.Sprintf("failed to spawn restart script: %v", err))
			return struct{}{}, nil
		}
		win32.PostQuitMessage(0)
		return struct{}{}, nil
	}).Detach()
}

// Activate does nothing on Windows.
func (p *WindowsPlatform) Activate(ignoringOtherApps bool) {}

// Hide does nothing on Windows.
func (p *WindowsPlatform) Hide() {}

// HideOtherApps is not implemented on Windows.
func (p *WindowsPlatform) HideOtherApps() {
	panic("not implemented")
}

// UnhideOtherApps is not implemented on Windows.
func (p *WindowsPlatform) UnhideOtherApps() {
	panic("not implemented")
}

// Displays returns every monitor.
func (p *WindowsPlatform) Displays() []gpui.PlatformDisplay {
	return WindowsDisplayDisplays()
}

// PrimaryDisplay returns the primary monitor, or nil.
func (p *WindowsPlatform) PrimaryDisplay() gpui.PlatformDisplay {
	display, ok := WindowsDisplayPrimaryMonitor()
	if !ok {
		return nil
	}
	return display
}

// OnDisplaysChanged registers the display configuration change callback.
func (p *WindowsPlatform) OnDisplaysChanged(callback func()) {
	p.inner.state.callbacks.displaysChanged = gpui.Some(callback)
}

// IsScreenCaptureSupported reports true: capture goes through the registered scap back end.
func (p *WindowsPlatform) IsScreenCaptureSupported() bool {
	return true
}

// ScreenCaptureSources lists capturable screens through the registered scap back end.
func (p *WindowsPlatform) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	return scapScreenSources(p.foregroundExecutor)
}

func scapScreenSources(executor gpui.ForegroundExecutor) *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	tx, rx := scheduler.Oneshot[scheduler.Result[[]gpui.ScreenCaptureSource]]()
	gpuiutil.LogErr(tx.Send(scheduler.Result[[]gpui.ScreenCaptureSource]{Err: errors.New("screen capture sources are not available yet")}))
	return rx
}

// ActiveWindow returns the window that has activation, if gpui owns it.
func (p *WindowsPlatform) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	inner, ok := p.windowFromHwnd(win32.GetActiveWindow())
	if !ok {
		return gpui.AnyWindowHandle{}, false
	}
	return inner.handle, true
}

// OpenWindow creates a window; it fails while headless.
func (p *WindowsPlatform) OpenWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams) (gpui.PlatformWindow, error) {
	if !p.isWindowed() {
		return nil, errors.New("cannot open windows while headless")
	}
	window, err := NewWindowsWindow(handle, options, p.generateCreationInfo())
	if err != nil {
		return nil, err
	}
	hwnd := HWND(window.GetRawHandle())
	p.rawWindowHandles.mu.Lock()
	p.rawWindowHandles.windows = append(p.rawWindowHandles.windows, trackedWindow{
		handle:      SafeHwndFrom(hwnd),
		frameSignal: window.inner.state.frameSignal,
	})
	p.rawWindowHandles.mu.Unlock()
	return window, nil
}

// WindowAppearance returns the system light or dark appearance.
func (p *WindowsPlatform) WindowAppearance() gpui.WindowAppearance {
	appearance, err := systemAppearance()
	if err != nil {
		gpuiutil.LogErr(err)
		return gpui.WindowAppearanceLight
	}
	return appearance
}

// OpenURL opens url with its default handler.
func (p *WindowsPlatform) OpenURL(url string) {
	if url == "" {
		return
	}
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if err := openTarget(url); err != nil {
			gpuiutil.LogErr(fmt.Errorf("Opening url: %s: %w", url, err))
		}
		return struct{}{}, nil
	}).Detach()
}

// OnOpenURLs registers the open-URLs callback.
func (p *WindowsPlatform) OnOpenURLs(callback func(urls []string)) {
	p.inner.state.callbacks.openURLs = gpui.Some(callback)
}

func (p *WindowsPlatform) activeDialogOwner() *DialogOwner {
	hwnd, ok := p.findCurrentActiveWindow()
	if !ok {
		return nil
	}
	window, ok := p.windowFromHwnd(hwnd)
	if !ok {
		return nil
	}
	return window.dialogOwner
}

func forwardDialog[T any](executor gpui.ForegroundExecutor, dialog *DialogReceiver[T]) *scheduler.OneshotReceiver[scheduler.Result[T]] {
	doneTx, doneRx := scheduler.Oneshot[scheduler.Result[T]]()
	executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		defer doneTx.Close()
		var result scheduler.Result[T]
		won := scheduler.Select(s,
			scheduler.Case[scheduler.Result[scheduler.Result[T]]](dialog, func(r scheduler.Result[scheduler.Result[T]]) {
				if r.Err != nil {
					result = scheduler.Result[T]{Err: r.Err}
					return
				}
				result = r.Value
			}),
			scheduler.Case(scheduler.PollFn(func(*scheduler.Context) (struct{}, bool) {
				return struct{}{}, doneTx.IsCanceled()
			}), nil),
		)
		if won != 0 {
			dialog.Close()
			return struct{}{}, nil
		}
		_ = doneTx.Send(result)
		return struct{}{}, nil
	}).Detach()
	return doneRx
}

// PromptForPaths shows the open dialog.
func (p *WindowsPlatform) PromptForPaths(options gpui.PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[[]string]]] {
	owner := p.activeDialogOwner()
	dialog := showDialog(owner, p.foregroundExecutor, func(window HWND) (gpui.Option[[]string], error) {
		return fileOpenDialog(options, window)
	})
	return forwardDialog(p.foregroundExecutor, dialog)
}

// PromptForNewPath shows the save dialog.
func (p *WindowsPlatform) PromptForNewPath(directory string, suggestedName gpui.Option[string]) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[string]]] {
	owner := p.activeDialogOwner()
	dialog := showDialog(owner, p.foregroundExecutor, func(window HWND) (gpui.Option[string], error) {
		return fileSaveDialog(directory, suggestedName, window)
	})
	return forwardDialog(p.foregroundExecutor, dialog)
}

// CanSelectMixedFilesAndDirs reports false: FOS_PICKFOLDERS switches between files and folders.
func (p *WindowsPlatform) CanSelectMixedFilesAndDirs() bool {
	return false
}

// RevealPath selects path in Explorer.
func (p *WindowsPlatform) RevealPath(path string) {
	if path == "" {
		return
	}
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if err := openTargetInExplorer(path); err != nil {
			gpuiutil.LogErr(fmt.Errorf("Revealing path %s in explorer: %w", path, err))
		}
		return struct{}{}, nil
	}).Detach()
}

// OpenWithSystem opens path with its default handler.
func (p *WindowsPlatform) OpenWithSystem(path string) {
	if path == "" {
		return
	}
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		if err := openTarget(path); err != nil {
			gpuiutil.LogErr(fmt.Errorf("Opening %s with system: %w", path, err))
		}
		return struct{}{}, nil
	}).Detach()
}

// OnQuit registers the quit callback.
func (p *WindowsPlatform) OnQuit(callback func() bool) {
	p.inner.state.callbacks.quit = gpui.Some(callback)
}

// OnReopen registers the reopen callback.
func (p *WindowsPlatform) OnReopen(callback func()) {
	p.inner.state.callbacks.reopen = gpui.Some(callback)
}

// OnSystemSleep registers the suspend callback.
func (p *WindowsPlatform) OnSystemSleep(callback func()) {
	p.inner.state.callbacks.systemSleep = gpui.Some(callback)
}

// OnSystemWake registers the resume callback and subscribes to suspend and resume broadcasts.
func (p *WindowsPlatform) OnSystemWake(callback func()) {
	p.inner.state.callbacks.systemWake = gpui.Some(callback)
	if p.suspendResumeNotification.IsNone() {
		notification, err := win32.RegisterSuspendResumeNotification(win32.HANDLE(p.handle), win32.DEVICE_NOTIFY_WINDOW_HANDLE)
		if err != nil {
			gpuiutil.LogErr(err)
		} else {
			p.suspendResumeNotification = gpui.Some(notification)
		}
	}
}

// PowerMode reports low power while battery saver is on.
func (p *WindowsPlatform) PowerMode() gpui.PowerMode {
	if p.inner.state.lowPowerModeEnabled {
		return gpui.PowerModeLowPower
	}
	return gpui.PowerModeHighPower
}

// OnPowerModeChange registers the power mode callback and subscribes to battery saver changes.
func (p *WindowsPlatform) OnPowerModeChange(callback func()) {
	p.inner.state.callbacks.lowPowerModeChange = gpui.Some(callback)
	if p.powerSavingNotification.IsNone() {
		notification, err := win32.RegisterPowerSettingNotification(win32.HANDLE(p.handle), &guidPowerSavingStatus, win32.DEVICE_NOTIFY_WINDOW_HANDLE)
		if err != nil {
			gpuiutil.LogErr(err)
		} else {
			p.powerSavingNotification = gpui.Some(notification)
		}
	}
}

// SetAppIdentity sets the process AppUserModelID unless the package already provides one.
func (p *WindowsPlatform) SetAppIdentity(identifier string, name string) {
	if p.hasPackageIdentity {
		return
	}
	identifierUTF16, err := xwin.UTF16PtrFromString(identifier)
	if err == nil {
		err = win32.SetCurrentProcessExplicitAppUserModelID(identifierUTF16)
	}
	if err != nil {
		slog.Warn(fmt.Sprintf("failed to set the process AppUserModelID: %v", err))
	}
	p.appIdentity = gpui.Some(appIdentity{identifier: identifier, name: name})
}

// ShowSystemNotification shows a toast.
func (p *WindowsPlatform) ShowSystemNotification(notification gpui.SystemNotification) {
	gpuiutil.LogErr(p.systemNotifications.show(p.hasPackageIdentity, p.appIdentity, notification))
}

// DismissSystemNotification removes the toast with tag.
func (p *WindowsPlatform) DismissSystemNotification(tag string) *gpui.Task[struct{}] {
	p.systemNotifications.dismiss(tag)
	return scheduler.ReadyTask(struct{}{})
}

// OnSystemNotificationResponse registers the toast activation callback.
func (p *WindowsPlatform) OnSystemNotificationResponse(callback func(response gpui.SystemNotificationResponse)) {
	p.systemNotifications.onResponse(p.foregroundExecutor, callback)
}

// SetMenus stores the menus; Windows draws no native menu bar.
func (p *WindowsPlatform) SetMenus(menus []gpui.Menu, keymap *gpui.Keymap) {
	owned := make([]gpui.OwnedMenu, 0, len(menus))
	for _, menu := range menus {
		owned = append(owned, menu.Owned())
	}
	p.inner.state.menus = owned
}

// GetMenus returns the stored menus.
func (p *WindowsPlatform) GetMenus() ([]gpui.OwnedMenu, bool) {
	return slices.Clone(p.inner.state.menus), true
}

// SetDockMenu sets the jump list tasks.
func (p *WindowsPlatform) SetDockMenu(menu []gpui.MenuItem, keymap *gpui.Keymap) {
	p.setDockMenus(menu)
}

// OnAppMenuAction registers the menu action callback.
func (p *WindowsPlatform) OnAppMenuAction(callback func(action gpui.Action)) {
	p.inner.state.callbacks.appMenuAction = gpui.Some(callback)
}

// OnWillOpenAppMenu registers the menu-opening callback.
func (p *WindowsPlatform) OnWillOpenAppMenu(callback func()) {
	p.inner.state.callbacks.willOpenAppMenu = gpui.Some(callback)
}

// OnValidateAppMenuCommand registers the menu validation callback.
func (p *WindowsPlatform) OnValidateAppMenuCommand(callback func(action gpui.Action) bool) {
	p.inner.state.callbacks.validateAppMenuCommand = gpui.Some(callback)
}

// AppPath returns the running executable's path.
func (p *WindowsPlatform) AppPath() (string, error) {
	return os.Executable()
}

// PathForAuxiliaryExecutable is not implemented on Windows.
func (p *WindowsPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	return "", errors.New("not yet implemented")
}

// SetCursorStyle tells every window to use the cursor for style.
func (p *WindowsPlatform) SetCursorStyle(style gpui.CursorStyle) {
	hcursor, _ := loadCursor(style)
	if p.inner.state.currentCursor != hcursor {
		p.postMessage(WM_GPUI_CURSOR_STYLE_CHANGED, win32.WPARAM(p.inner.validationNumber), win32.LPARAM(hcursor))
		p.inner.state.currentCursor = hcursor
	}
}

// HideCursorUntilMouseMoves hides the cursor over the hovered window until the mouse moves.
func (p *WindowsPlatform) HideCursorUntilMouseMoves() {
	if !p.inner.state.cursorVisible.Swap(false) {
		return
	}
	p.rawWindowHandles.mu.RLock()
	defer p.rawWindowHandles.mu.RUnlock()
	for i := range p.rawWindowHandles.windows {
		window, ok := windowFromHwnd(p.rawWindowHandles.windows[i].asRaw())
		if !ok {
			continue
		}
		if window.state.hovered {
			win32.SetCursor(0)
			break
		}
	}
}

// IsCursorVisible reports whether the cursor is shown.
func (p *WindowsPlatform) IsCursorVisible() bool {
	return p.inner.state.cursorVisible.Load()
}

// ShouldAutoHideScrollbars reads the system's auto-hide scroll bars setting.
func (p *WindowsPlatform) ShouldAutoHideScrollbars() bool {
	value, err := shouldAutoHideScrollbars()
	if err != nil {
		gpuiutil.LogErr(err)
		return false
	}
	return value
}

// WriteToClipboard writes item to the clipboard.
func (p *WindowsPlatform) WriteToClipboard(item gpui.ClipboardItem) {
	writeToClipboard(item)
}

// ReadFromClipboard reads the clipboard.
func (p *WindowsPlatform) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	return readFromClipboard()
}

// WriteCredentials stores a generic credential for url in the Credential Manager.
func (p *WindowsPlatform) WriteCredentials(url string, username string, password []byte) *gpui.Task[struct{}] {
	if len(password) > win32.CRED_MAX_CREDENTIAL_BLOB_SIZE {
		return scheduler.ReadyTaskErr[struct{}](fmt.Errorf(
			"credential for %s is %d bytes, which exceeds the Windows Credential Manager limit of %d bytes",
			url, len(password), win32.CRED_MAX_CREDENTIAL_BLOB_SIZE,
		))
	}
	usernameUTF16, err := xwin.UTF16FromString(username)
	if err != nil {
		return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("credential username: %w", err))
	}
	targetName, err := xwin.UTF16FromString(windowsCredentialsTargetName(url))
	if err != nil {
		return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("credential target %s: %w", url, err))
	}
	password = slices.Clone(password)
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		var lastWritten xwin.Filetime
		xwin.GetSystemTimeAsFileTime(&lastWritten)
		credentials := win32.CREDENTIALW{
			LastWritten:        lastWritten,
			Type:               win32.CRED_TYPE_GENERIC,
			TargetName:         &targetName[0],
			CredentialBlobSize: uint32(len(password)),
			Persist:            win32.CRED_PERSIST_LOCAL_MACHINE,
			UserName:           &usernameUTF16[0],
		}
		if len(password) > 0 {
			credentials.CredentialBlob = &password[0]
		}
		err := win32.CredWriteW(&credentials, 0)
		runtime.KeepAlive(password)
		runtime.KeepAlive(targetName)
		runtime.KeepAlive(usernameUTF16)
		if err != nil {
			return struct{}{}, fmt.Errorf("Failed to write credentials to Windows Credential Manager: %w", err)
		}
		return struct{}{}, nil
	})
}

// ReadCredentials reads the credential for url; a missing one is None.
func (p *WindowsPlatform) ReadCredentials(url string) *gpui.Task[gpui.Option[gpui.Credentials]] {
	targetName, err := xwin.UTF16FromString(windowsCredentialsTargetName(url))
	if err != nil {
		return scheduler.ReadyTaskErr[gpui.Option[gpui.Credentials]](fmt.Errorf("credential target %s: %w", url, err))
	}
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (gpui.Option[gpui.Credentials], error) {
		credentials, err := win32.CredReadW(&targetName[0], win32.CRED_TYPE_GENERIC, 0)
		runtime.KeepAlive(targetName)
		if err != nil {
			if errors.Is(err, syscall.Errno(win32.ERROR_NOT_FOUND)) {
				return gpui.None[gpui.Credentials](), nil
			}
			return gpui.None[gpui.Credentials](), err
		}
		if credentials == nil {
			return gpui.None[gpui.Credentials](), nil
		}
		username, password, err := usernameAndPassword(credentials)
		win32.CredFree(unsafe.Pointer(credentials))
		if err != nil {
			return gpui.None[gpui.Credentials](), err
		}
		return gpui.Some(gpui.Credentials{Username: username, Password: password}), nil
	})
}

// DeleteCredentials deletes the credential for url.
func (p *WindowsPlatform) DeleteCredentials(url string) *gpui.Task[struct{}] {
	targetName, err := xwin.UTF16FromString(windowsCredentialsTargetName(url))
	if err != nil {
		return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("credential target %s: %w", url, err))
	}
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		err := win32.CredDeleteW(&targetName[0], win32.CRED_TYPE_GENERIC, 0)
		runtime.KeepAlive(targetName)
		return struct{}{}, err
	})
}

// RegisterURLScheme is not implemented on Windows.
func (p *WindowsPlatform) RegisterURLScheme(url string) *gpui.Task[struct{}] {
	return scheduler.ReadyTaskErr[struct{}](errors.New("register_url_scheme unimplemented"))
}

// PerformDockMenuAction runs the jump list task at index action from the message loop.
func (p *WindowsPlatform) PerformDockMenuAction(action int) {
	gpuiutil.LogErr(win32.PostMessageW(p.handle, WM_GPUI_DOCK_MENU_ACTION, win32.WPARAM(p.inner.validationNumber), win32.LPARAM(action)))
}

// UpdateJumpList replaces the jump list and returns the workspaces the user removed from it.
func (p *WindowsPlatform) UpdateJumpList(menus []gpui.MenuItem, entries [][]string) *gpui.Task[[][]string] {
	return p.updateJumpList(menus, entries)
}

// Close is the platform's drop: it stops vsync, unregisters power notifications, destroys the
// platform window and uninitializes OLE. It is idempotent.
func (p *WindowsPlatform) Close() {
	if p.closed {
		return
	}
	p.closed = true
	if stop := p.vsyncStop; stop != nil {
		p.vsyncStop = nil
		stop.Store(true)
	}
	p.inner.state.vsyncDevices = nil
	if notification, ok := p.suspendResumeNotification.Take().Get(); ok {
		gpuiutil.LogErr(win32.UnregisterSuspendResumeNotification(notification))
	}
	if notification, ok := p.powerSavingNotification.Take().Get(); ok {
		gpuiutil.LogErr(win32.UnregisterPowerSettingNotification(notification))
	}
	if err := win32.DestroyWindow(p.handle); err != nil {
		gpuiutil.LogErr(fmt.Errorf("Destroying platform window: %w", err))
	}
	p.rawWindowHandles.dropped.Store(true)
	com.OleUninitialize()
	if devices := p.inner.state.directxDevices; devices != nil {
		p.inner.state.directxDevices = nil
		devices.Release()
	}
	p.inner.mainReceiver.Close()
	if closer, ok := p.textSystem.(interface{ Close() }); ok {
		closer.Close()
	}
	if p.dropTargetHelper != nil {
		p.dropTargetHelper.Release()
		p.dropTargetHelper = nil
	}
}

func newWindowsPlatformInner(context *platformWindowCreateContext) (*windowsPlatformInner, error) {
	state := newWindowsPlatformState(context.directxDevices)
	context.directxDevices = nil
	if context.dispatcher == nil {
		return nil, errors.New("missing dispatcher")
	}
	mainReceiver, ok := context.mainReceiver.Take().Get()
	if !ok {
		return nil, errors.New("missing main receiver")
	}
	return &windowsPlatformInner{
		state:            state,
		rawWindowHandles: context.rawWindowHandles,
		dispatcher:       context.dispatcher,
		validationNumber: context.validationNumber,
		mainReceiver:     mainReceiver,
	}, nil
}

func (i *windowsPlatformInner) windowFromHwnd(hwnd HWND) (*WindowsWindowInner, bool) {
	if i.rawWindowHandles.dropped.Load() {
		return nil, false
	}
	i.rawWindowHandles.mu.RLock()
	found := false
	for j := range i.rawWindowHandles.windows {
		if i.rawWindowHandles.windows[j].asRaw() == hwnd {
			found = true
			break
		}
	}
	i.rawWindowHandles.mu.RUnlock()
	if !found {
		return nil, false
	}
	return windowFromHwnd(hwnd)
}

func (i *windowsPlatformInner) translateAccelerator(msg *win32.MSG) bool {
	if msg.Message != win32.WM_KEYDOWN && msg.Message != win32.WM_SYSKEYDOWN {
		return false
	}
	window, ok := i.windowFromHwnd(msg.Hwnd)
	if !ok {
		return false
	}
	result := window.handleKeydownMsg(msg.WParam, msg.LParam)
	return result.ok && result.value == 0
}

func withCallback[T any](slot *gpui.Option[T], f func(T)) {
	callback, ok := slot.Take().Get()
	if !ok {
		return
	}
	f(callback)
	*slot = gpui.Some(callback)
}

func (i *windowsPlatformInner) handleMsg(handle HWND, msg uint32, wparam win32.WPARAM, lparam win32.LPARAM) win32.LRESULT {
	var result gpui.Option[uintptr]
	switch msg {
	case WM_GPUI_CLOSE_ONE_WINDOW,
		WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD,
		WM_GPUI_DOCK_MENU_ACTION,
		WM_GPUI_KEYBOARD_LAYOUT_CHANGED,
		WM_GPUI_DISPLAYS_CHANGED,
		WM_GPUI_GPU_DEVICE_LOST:
		result = i.handleGpuiEvents(msg, wparam, lparam)
	case win32.WM_POWERBROADCAST:
		result = i.handlePowerBroadcast(wparam, lparam)
	case win32.WM_DISPLAYCHANGE:
		result = i.handleDisplayChange(handle)
	case win32.WM_QUERYENDSESSION:
		result = gpui.Some[uintptr](1)
	case win32.WM_ENDSESSION:
		if wparam != 0 {
			result = i.handleEndSession()
		}
	}
	if value, ok := result.Get(); ok {
		return win32.LRESULT(value)
	}
	return win32.DefWindowProcW(handle, msg, wparam, lparam)
}

func (i *windowsPlatformInner) handleGpuiEvents(message uint32, wparam win32.WPARAM, lparam win32.LPARAM) gpui.Option[uintptr] {
	if uintptr(wparam) != i.validationNumber {
		slog.Error(fmt.Sprintf("Wrong validation number while processing message: %d", message))
		return gpui.None[uintptr]()
	}
	switch message {
	case WM_GPUI_CLOSE_ONE_WINDOW:
		i.closeOneWindow(HWND(lparam))
		return gpui.Some[uintptr](0)
	case WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD:
		return i.runForegroundTask()
	case WM_GPUI_DOCK_MENU_ACTION:
		return i.handleDockActionEvent(int(lparam))
	case WM_GPUI_KEYBOARD_LAYOUT_CHANGED:
		return i.handleKeyboardLayoutChange()
	case WM_GPUI_DISPLAYS_CHANGED:
		return i.handleDisplaysChanged()
	case WM_GPUI_GPU_DEVICE_LOST:
		return i.handleDeviceLost(lparam)
	}
	panic("internal error: entered unreachable code")
}

func (i *windowsPlatformInner) handleDisplayChange(handle HWND) gpui.Option[uintptr] {
	gpuiutil.LogErr(win32.PostMessageW(handle, WM_GPUI_DISPLAYS_CHANGED, win32.WPARAM(i.validationNumber), 0))
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) handleDisplaysChanged() gpui.Option[uintptr] {
	withCallback(&i.state.callbacks.displaysChanged, func(callback func()) { callback() })
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) handleEndSession() gpui.Option[uintptr] {
	shutdownCompleted := false
	withCallback(&i.state.callbacks.quit, func(callback func() bool) {
		shutdownCompleted = callback()
	})
	if shutdownCompleted {
		os.Exit(0)
	}
	win32.PostQuitMessage(0)
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) closeOneWindow(targetWindow HWND) bool {
	if i.rawWindowHandles.dropped.Load() {
		slog.Error("Failed to upgrade raw window handles")
		return false
	}
	all := i.rawWindowHandles
	all.mu.Lock()
	defer all.mu.Unlock()
	index := slices.IndexFunc(all.windows, func(w trackedWindow) bool {
		return w.asRaw() == targetWindow
	})
	if index < 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	all.windows = slices.Delete(all.windows, index, index+1)
	return len(all.windows) == 0
}

const mainTaskTimeout = 10 * time.Millisecond

func (i *windowsPlatformInner) runForegroundTask() gpui.Option[uintptr] {
	start := time.Now()
tasks:
	for {
		for {
			if time.Since(start) >= mainTaskTimeout {
				slog.Debug("foreground task timeout reached")
				var msg win32.MSG
				processMessage := func(msg *win32.MSG) {
					if !i.translateAccelerator(msg) {
						win32.TranslateMessage(msg)
						win32.DispatchMessageW(msg)
					}
				}
				peekMsg := func(msg *win32.MSG, msgKind uint32) bool {
					return win32.PeekMessageW(msg, 0, 0, 0, win32.PM_REMOVE|msgKind)
				}
				if peekMsg(&msg, win32.PM_QS_PAINT) {
					processMessage(&msg)
				}
				for peekMsg(&msg, win32.PM_QS_INPUT) {
					processMessage(&msg)
				}
				if err := win32.PostMessageW(i.dispatcher.PlatformWindowHandle.AsRaw(), WM_GPUI_TASK_DISPATCHED_ON_MAIN_THREAD, win32.WPARAM(i.validationNumber), 0); err != nil {
					i.dispatcher.WakePosted.Store(false)
				}
				break tasks
			}
			runnable, ok, err := i.mainReceiver.TryPop()
			if err != nil || !ok {
				break
			}
			WindowsDispatcherExecuteRunnable(runnable)
		}

		i.dispatcher.WakePosted.Store(false)
		runnable, ok, err := i.mainReceiver.TryPop()
		if err != nil || !ok {
			break tasks
		}
		i.dispatcher.WakePosted.Store(true)
		WindowsDispatcherExecuteRunnable(runnable)
	}
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) handleDockActionEvent(actionIdx int) gpui.Option[uintptr] {
	var action gpui.Action
	if actionIdx >= 0 && actionIdx < len(i.state.jumpList.dockMenus) {
		action = i.state.jumpList.dockMenus[actionIdx].action.BoxedClone()
	}
	if action == nil {
		slog.Error(fmt.Sprintf("Dock menu for index %d not found", actionIdx))
		return gpui.Some[uintptr](1)
	}
	withCallback(&i.state.callbacks.appMenuAction, func(callback func(gpui.Action)) { callback(action) })
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) handleKeyboardLayoutChange() gpui.Option[uintptr] {
	withCallback(&i.state.callbacks.keyboardLayoutChange, func(callback func()) { callback() })
	return gpui.Some[uintptr](0)
}

func (i *windowsPlatformInner) handlePowerBroadcast(wparam win32.WPARAM, lparam win32.LPARAM) gpui.Option[uintptr] {
	switch uint32(wparam) {
	case win32.PBT_APMSUSPEND:
		withCallback(&i.state.callbacks.systemSleep, func(callback func()) { callback() })
	case win32.PBT_APMRESUMEAUTOMATIC:
		withCallback(&i.state.callbacks.systemWake, func(callback func()) { callback() })
	case win32.PBT_POWERSETTINGCHANGE:
		if enabled, ok := powerSavingStatus(lparam); ok {
			previous := i.state.lowPowerModeEnabled
			i.state.lowPowerModeEnabled = enabled
			if previous != enabled {
				withCallback(&i.state.callbacks.lowPowerModeChange, func(callback func()) { callback() })
			}
		}
	}
	return gpui.Some[uintptr](1)
}

func (i *windowsPlatformInner) handleDeviceLost(lparam win32.LPARAM) gpui.Option[uintptr] {
	directxDevices := lparamPtr[DirectXDevices](lparam)
	if directxDevices != i.state.vsyncDevices {
		return gpui.Some[uintptr](1)
	}
	if current := i.state.directxDevices; current != nil {
		i.state.directxDevices = directxDevices.Clone()
		current.Release()
	}
	return gpui.Some[uintptr](0)
}

var guidPowerSavingStatus = com.MustGUID("{E00958C0-C213-4ACE-AC77-FECCED2EEEA5}")

func powerSavingStatus(lparam win32.LPARAM) (bool, bool) {
	if lparam == 0 {
		return false, false
	}
	setting := lparamPtr[win32.POWERBROADCAST_SETTING](lparam)
	if setting.PowerSetting != guidPowerSavingStatus || setting.DataLength != 4 {
		return false, false
	}
	data := *(*uint32)(unsafe.Pointer(&setting.Data[0]))
	switch data {
	case 0:
		return false, true
	case 1:
		return true, true
	}
	return false, false
}

// WindowCreationInfo is what the platform hands a new window.
type WindowCreationInfo struct {
	icon                     win32.HICON
	executor                 gpui.ForegroundExecutor
	currentCursor            win32.HCURSOR
	cursorVisible            *atomic.Bool
	dropTargetHelper         *IDropTargetHelper
	validationNumber         uintptr
	mainReceiver             *gpui.PriorityQueueReceiver[gpui.RunnableVariant]
	platformWindowHandle     HWND
	disableDirectComposition bool
	directxDevices           *DirectXDevices
	invalidateDevices        *atomic.Bool
	drawCoordinator          *DrawCoordinator
}

type platformWindowCreateContext struct {
	inner            *windowsPlatformInner
	innerErr         error
	innerSet         bool
	rawWindowHandles *trackedWindows
	validationNumber uintptr
	mainSender       gpui.Option[*gpui.PriorityQueueSender[gpui.RunnableVariant]]
	mainReceiver     gpui.Option[*gpui.PriorityQueueReceiver[gpui.RunnableVariant]]
	directxDevices   *DirectXDevices
	dispatcher       *WindowsDispatcher
}

const appmodelErrorNoPackage = 15700

func hasPackageIdentity() bool {
	var packageFullNameLength uint32
	result := win32.GetCurrentPackageFullName(&packageFullNameLength, nil)
	switch result {
	case win32.ERROR_INSUFFICIENT_BUFFER:
		return true
	case appmodelErrorNoPackage:
		return false
	}
	slog.Warn(fmt.Sprintf("failed to determine whether the process has package identity: %d", result))
	return false
}

func openTarget(target string) error {
	targetUTF16, err := xwin.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	operation, _ := xwin.UTF16PtrFromString("open")
	ret := win32.ShellExecuteW(0, operation, targetUTF16, nil, nil, win32.SW_SHOWDEFAULT)
	if int(ret) <= 32 {
		return fmt.Errorf("Unable to open target: %w", xwin.GetLastError())
	}
	return nil
}

func openTargetInExplorer(target string) error {
	dir := filepath.Dir(target)
	if dir == target {
		return errors.New("No parent folder found")
	}
	desktop, err := shGetDesktopFolder()
	if err != nil {
		return err
	}
	defer desktop.Release()

	dirItem, err := desktop.parseDisplayName(dir)
	if err != nil {
		return err
	}
	defer freePidl(dirItem)
	fileItem, err := desktop.parseDisplayName(target)
	if err != nil {
		return err
	}
	defer freePidl(fileItem)

	highlight := []uintptr{fileItem}
	if err := shOpenFolderAndSelectItems(dirItem, highlight, 0); err != nil {
		if com.HRESULTFromError(err) == com.HRESULTFromWin32(win32.ERROR_FILE_NOT_FOUND) {
			if err := openTarget(dir); err != nil {
				return fmt.Errorf("Opening target parent folder: %w", err)
			}
			return nil
		}
		return fmt.Errorf("Can not open target path: %w", err)
	}
	return nil
}

func freePidl(pidl uintptr) {
	com.CoTaskMemFree(*(*unsafe.Pointer)(unsafe.Pointer(&pidl)))
}

func isDialogCancelled(err error) bool {
	return com.HRESULTFromError(err) == com.HRESULTFromWin32(win32.ERROR_CANCELLED)
}

func fileOpenDialog(options gpui.PathPromptOptions, window HWND) (gpui.Option[[]string], error) {
	folderDialog, err := com.CreateInstance[iFileOpenDialog](&clsidFileOpenDialog, win32.CLSCTX_ALL, &iidIFileOpenDialog)
	if err != nil {
		return gpui.None[[]string](), err
	}
	defer folderDialog.Release()

	dialogOptions := uint32(win32.FOS_FILEMUSTEXIST)
	if options.Multiple {
		dialogOptions |= win32.FOS_ALLOWMULTISELECT
	}
	if options.Directories {
		dialogOptions |= win32.FOS_PICKFOLDERS
	}

	if err := folderDialog.setOptions(dialogOptions); err != nil {
		return gpui.None[[]string](), err
	}
	if prompt, ok := options.Prompt.Get(); ok {
		if err := folderDialog.setOkButtonLabel(string(prompt)); err != nil {
			return gpui.None[[]string](), err
		}
	}
	if err := folderDialog.show(window); err != nil {
		if isDialogCancelled(err) {
			return gpui.None[[]string](), nil
		}
		return gpui.None[[]string](), err
	}

	results, err := folderDialog.getResults()
	if err != nil {
		return gpui.None[[]string](), err
	}
	defer results.Release()
	fileCount, err := results.getCount()
	if err != nil {
		return gpui.None[[]string](), err
	}
	if fileCount == 0 {
		return gpui.None[[]string](), nil
	}

	paths := make([]string, 0, fileCount)
	for i := range fileCount {
		item, err := results.getItemAt(i)
		if err != nil {
			return gpui.None[[]string](), err
		}
		path, err := item.getDisplayName(win32.SIGDN_FILESYSPATH)
		item.Release()
		if err != nil {
			return gpui.None[[]string](), err
		}
		paths = append(paths, path)
	}
	return gpui.Some(paths), nil
}

func fileSaveDialog(directory string, suggestedName gpui.Option[string], window HWND) (gpui.Option[string], error) {
	dialog, err := com.CreateInstance[iFileDialog](&clsidFileSaveDialog, win32.CLSCTX_ALL, &iidIFileSaveDialog)
	if err != nil {
		return gpui.None[string](), err
	}
	defer dialog.Release()
	if directory != "" {
		fullPath, err := canonicalize(directory)
		if err != nil {
			gpuiutil.LogErr(fmt.Errorf("failed to canonicalize directory: %w", err))
		} else {
			pathUTF16, err := xwin.UTF16PtrFromString(fullPath)
			if err != nil {
				return gpui.None[string](), err
			}
			pathItem, err := win32.SHCreateItemFromParsingName(pathUTF16, &iidIShellItem)
			if err != nil {
				return gpui.None[string](), err
			}
			if err := dialog.setFolder(pathItem); err != nil {
				gpuiutil.LogErr(fmt.Errorf("failed to set dialog folder: %w", err))
			}
			pathItem.Release()
		}
	}

	if name, ok := suggestedName.Get(); ok {
		if err := dialog.setFileName(name); err != nil {
			gpuiutil.LogErr(fmt.Errorf("failed to set file name: %w", err))
		}
	}

	if err := dialog.setFileTypes([]filterSpec{{name: "All files", spec: "*.*"}}); err != nil {
		return gpui.None[string](), err
	}
	if err := dialog.show(window); err != nil {
		if isDialogCancelled(err) {
			return gpui.None[string](), nil
		}
		return gpui.None[string](), err
	}
	shellItem, err := dialog.getResult()
	if err != nil {
		return gpui.None[string](), err
	}
	defer shellItem.Release()
	filePath, err := shellItem.getDisplayName(win32.SIGDN_FILESYSPATH)
	if err != nil {
		return gpui.None[string](), err
	}
	return gpui.Some(filePath), nil
}

func canonicalize(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func loadIcon() (win32.HICON, error) {
	module, err := win32.GetModuleHandleW(nil)
	if err != nil {
		return 0, fmt.Errorf("unable to get module handle: %w", err)
	}
	handle, err := win32.LoadImageW(win32.HINSTANCE(module), (*uint16)(win32.PtrFromUintptr(1)), win32.IMAGE_ICON, 0, 0, win32.LR_DEFAULTSIZE|win32.LR_SHARED)
	if err != nil {
		return 0, fmt.Errorf("unable to load icon file: %w", err)
	}
	return win32.HICON(handle), nil
}

var iidIUISettings5 = com.MustGUID("{5349D588-0CB5-5F05-BD34-706B3231F0BD}")

func shouldAutoHideScrollbars() (bool, error) {
	uiSettings, err := com.RoActivateInstance(uiSettingsClassName)
	if err != nil {
		return false, err
	}
	defer uiSettings.Release()
	settings5, err := com.Cast[com.IInspectable](&uiSettings.IUnknown, &iidIUISettings5)
	if err != nil {
		return false, err
	}
	defer settings5.Release()
	var value uint8
	if err := com.CallHR(settings5.Ptr(), com.IInspectableSlots, uintptr(unsafe.Pointer(&value))); err != nil {
		return false, err
	}
	return value != 0, nil
}

func checkDeviceLost(device *d3d.ID3D11Device) bool {
	if err := device.GetDeviceRemovedReason(); err != nil {
		slog.Error(fmt.Sprintf("DirectX device lost detected: %v", err))
		return true
	}
	return false
}

func checkCanShowWindows(environment gpui.GraphicalEnvironment) error {
	current := gpui.GraphicalEnvironmentDetect().SessionID
	requested, requestedOK := environment.SessionID.Get()
	currentID, currentOK := current.Get()
	if requestedOK && currentOK && requested != currentID {
		return fmt.Errorf("windows can only be shown in this process's session (%d), not session %d", currentID, requested)
	}
	if !isWindowStationVisible() {
		return errors.New("this process's window station is not interactive, so it can't show windows")
	}
	return nil
}

func isWindowStationVisible() bool {
	station, err := win32.GetProcessWindowStation()
	if err != nil {
		return false
	}
	var flags win32.USEROBJECTFLAGS
	err = win32.GetUserObjectInformationW(win32.HANDLE(station), win32.UOI_FLAGS, unsafe.Pointer(&flags), uint32(unsafe.Sizeof(flags)), nil)
	return err == nil && flags.DwFlags&win32.WSF_VISIBLE != 0
}

func handleGpuDeviceLost(directxDevices *DirectXDevices, platformWindow HWND, validationNumber uintptr, allWindows *trackedWindows, stop *atomic.Bool) error {
	time.Sleep(350 * time.Millisecond)

	recovered, err := tryToRecoverFromDeviceLost(func() (*DirectXDevices, error) {
		devices, err := NewDirectXDevices()
		if err != nil {
			return nil, fmt.Errorf("Failed to recreate new DirectX devices after device lost: %w", err)
		}
		return devices, nil
	})
	if err != nil {
		return err
	}
	directxDevices.Release()
	*directxDevices = *recovered
	slog.Info("DirectX devices successfully recreated.")

	lparam := win32.LPARAM(uintptr(unsafe.Pointer(directxDevices)))
	if win32.SendMessageW(platformWindow, WM_GPUI_GPU_DEVICE_LOST, win32.WPARAM(validationNumber), lparam) != 0 {
		runtime.KeepAlive(directxDevices)
		return nil
	}

	if !allWindows.dropped.Load() {
		allWindows.mu.RLock()
		windows := slices.Clone(allWindows.windows)
		allWindows.mu.RUnlock()
		for i := range windows {
			if stop.Load() {
				break
			}
			win32.SendMessageW(windows[i].asRaw(), WM_GPUI_GPU_DEVICE_LOST, win32.WPARAM(validationNumber), lparam)
		}
		time.Sleep(200 * time.Millisecond)
		for i := range windows {
			win32.SendMessageW(windows[i].asRaw(), WM_GPUI_FORCE_UPDATE_WINDOW, win32.WPARAM(validationNumber), 0)
		}
	}
	runtime.KeepAlive(directxDevices)
	return nil
}

var platformWindowClassName = wideZ("Zed::PlatformWindow")

var (
	registerPlatformWindowClassOnce sync.Once
	platformWindowProcedureCallback uintptr
	platformCreateContextsMu        sync.Mutex
	platformCreateContexts          = map[uintptr]*platformWindowCreateContext{}
	nextPlatformCreateContext       uintptr
	platformRegistryMu              sync.Mutex
	platformRegistry                = map[uintptr]weak.Pointer[windowsPlatformInner]{}
	nextPlatformRegistry            uintptr
)

func registerPlatformWindowClass() {
	registerPlatformWindowClassOnce.Do(func() {
		platformWindowProcedureCallback = xwin.NewCallback(platformWindowProcedure)
		wc := win32.WNDCLASSW{
			LpfnWndProc:   platformWindowProcedureCallback,
			LpszClassName: platformWindowClassName,
		}
		_, _ = win32.RegisterClassW(&wc)
	})
}

func platformWindowProcedure(hwnd HWND, msg uint32, wparam win32.WPARAM, lparam win32.LPARAM) uintptr {
	if msg == win32.WM_NCCREATE {
		params := lparamPtr[win32.CREATESTRUCTW](lparam)
		platformCreateContextsMu.Lock()
		creationContext := platformCreateContexts[params.LpCreateParams]
		platformCreateContextsMu.Unlock()
		if creationContext == nil {
			return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
		}

		mainSender, ok := creationContext.mainSender.Take().Get()
		if !ok {
			creationContext.innerErr = errors.New("missing main sender")
			creationContext.innerSet = true
			return 0
		}
		creationContext.dispatcher = NewWindowsDispatcher(mainSender, hwnd, creationContext.validationNumber)

		inner, err := newWindowsPlatformInner(creationContext)
		if err != nil {
			creationContext.innerErr = err
			creationContext.innerSet = true
			return 0
		}
		platformRegistryMu.Lock()
		nextPlatformRegistry++
		id := nextPlatformRegistry
		platformRegistry[id] = weak.Make(inner)
		platformRegistryMu.Unlock()
		setWindowLong(hwnd, win32.GWLP_USERDATA, int(id))
		creationContext.inner = inner
		creationContext.innerSet = true
		return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}

	id := uintptr(getWindowLong(hwnd, win32.GWLP_USERDATA))
	if id == 0 {
		return uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}
	platformRegistryMu.Lock()
	ptr := platformRegistry[id]
	platformRegistryMu.Unlock()
	var result uintptr
	if inner := ptr.Value(); inner != nil {
		result = uintptr(inner.handleMsg(hwnd, msg, wparam, lparam))
	} else {
		result = uintptr(win32.DefWindowProcW(hwnd, msg, wparam, lparam))
	}

	if msg == win32.WM_NCDESTROY {
		setWindowLong(hwnd, win32.GWLP_USERDATA, 0)
		platformRegistryMu.Lock()
		delete(platformRegistry, id)
		platformRegistryMu.Unlock()
	}

	return result
}

func usernameAndPassword(credential *win32.CREDENTIALW) (string, []byte, error) {
	username := ""
	if credential.UserName != nil {
		decoded, err := decodeUTF16Strict(utf16PtrSlice(credential.UserName))
		if err != nil {
			return "", nil, err
		}
		username = decoded
	}
	var password []byte
	if credential.CredentialBlob == nil {
		password = []byte{}
	} else {
		password = slices.Clone(unsafe.Slice(credential.CredentialBlob, credential.CredentialBlobSize))
	}
	return username, password, nil
}

func utf16PtrSlice(p *uint16) []uint16 {
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), n*2)) != 0 {
		n++
	}
	return unsafe.Slice(p, n)
}
