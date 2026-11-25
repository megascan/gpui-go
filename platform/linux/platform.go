//go:build linux

package linux

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform/linux/internal/common"
	"github.com/megascan/gpui-go/platform/linux/internal/dbus"
	"github.com/megascan/gpui-go/platform/linux/wayland"
	"github.com/megascan/gpui-go/platform/linux/x11"
	"github.com/megascan/gpui-go/scheduler"
	"golang.org/x/sys/unix"
)

type linuxPlatform struct {
	gpui.PlatformDefaults
	common             *common.LinuxCommon
	eventLoop          *common.EventLoop
	ownedLoop          *common.EventLoop
	loopHandle         *common.LoopHandle
	connection         *displayConnection
	allowedModes       gpui.WindowingModes
	initialWindowing   gpui.Option[gpui.WindowingRequest]
	startupEnvironment *startupEnvironment
	pendingMode        gpui.Option[gpui.WindowingRequest]
	transitionWaiter   *scheduler.OneshotSender[error]
	powerModeCancel    context.CancelFunc
	closed             bool
}

type startupEnvironment struct {
	activationToken gpui.Option[string]
	waylandSocket   gpui.Option[int]
}

func (s *startupEnvironment) close() {
	if fd, ok := s.waylandSocket.Get(); ok {
		_ = unix.Close(fd)
		s.waylandSocket = gpui.None[int]()
	}
}

func newLinuxPlatform(allowedModes gpui.WindowingModes) *linuxPlatform {
	startup := &startupEnvironment{
		activationToken: wayland.TakeStartupActivationTokenFromEnvironment(),
		waylandSocket:   wayland.TakeWaylandSocketFromEnvironment(),
	}
	eventLoop, err := common.NewEventLoop()
	if err != nil {
		panic(fmt.Sprintf("failed to create Linux event loop: %v", err))
	}
	linuxCommon, mainReceiver, powerReceiver := common.NewLinuxCommon(eventLoop.GetSignal())
	loopHandle := eventLoop.Handle()
	if err := common.RegisterSources(linuxCommon, loopHandle, mainReceiver, powerReceiver); err != nil {
		panic(fmt.Sprintf("failed to register Linux event sources: %v", err))
	}
	p := &linuxPlatform{
		common:             linuxCommon,
		eventLoop:          eventLoop,
		ownedLoop:          eventLoop,
		loopHandle:         loopHandle,
		connection:         displayConnectionHeadless(common.NewHeadlessConnection()),
		allowedModes:       allowedModes,
		startupEnvironment: startup,
	}
	gpui.BindPlatformDefaults(&p.PlatformDefaults, p)
	return p
}

func windowingModesDebug(modes gpui.WindowingModes) string {
	names := []struct {
		flag gpui.WindowingModes
		name string
	}{
		{gpui.WindowingModesWayland, "WAYLAND"},
		{gpui.WindowingModesX11, "X11"},
		{gpui.WindowingModesHeadless, "HEADLESS"},
	}
	if modes == 0 {
		return "WindowingModes(0x0)"
	}
	var parts []string
	remaining := modes
	for _, entry := range names {
		if modes.Contains(entry.flag) {
			parts = append(parts, entry.name)
			remaining &^= entry.flag
		}
	}
	if remaining != 0 {
		parts = append(parts, "0x"+strconv.FormatUint(uint64(remaining), 16))
	}
	return "WindowingModes(" + strings.Join(parts, " | ") + ")"
}

func (p *linuxPlatform) connectInitially() {
	request, ok := p.initialWindowing.Get()
	p.initialWindowing = gpui.None[gpui.WindowingRequest]()
	if !ok {
		request = gpui.WindowingRequestWindowed{Environment: gpui.GraphicalEnvironmentDetect()}
	}
	startup := p.startupEnvironment
	p.startupEnvironment = nil
	if startup == nil {
		startup = &startupEnvironment{}
	}
	defer startup.close()
	var selected backend
	var environment gpui.GraphicalEnvironment
	found := false
	if windowed, isWindowed := request.(gpui.WindowingRequestWindowed); isWindowed {
		environment = windowed.Environment
		selected, found = selectBackend(p.allowedModes, environment)
	}
	if found {
		connection, err := p.connect(selected, environment, startup)
		if err != nil {
			panic(err.Error())
		}
		p.replaceConnection(connection)
		return
	}
	if !p.allowedModes.Contains(gpui.WindowingModesHeadless) {
		panic(fmt.Sprintf("%s does not allow starting headless, and no allowed display server was given", windowingModesDebug(p.allowedModes)))
	}
}

func (p *linuxPlatform) connect(selected backend, environment gpui.GraphicalEnvironment, startup *startupEnvironment) (*displayConnection, error) {
	switch selected {
	case backendWayland:
		activationToken := environment.ActivationToken
		if !activationToken.IsSome() {
			activationToken = startup.activationToken
		}
		socket := startup.waylandSocket
		startup.waylandSocket = gpui.None[int]()
		connection, err := wayland.Attach(p.loopHandle, p.common, environment, socket, activationToken)
		if err != nil {
			return nil, err
		}
		return displayConnectionConnected(connection), nil
	default:
		connection, err := x11.X11ConnectionAttach(p.loopHandle, p.common, environment)
		if err != nil {
			return nil, err
		}
		return displayConnectionConnected(connection), nil
	}
}

func (p *linuxPlatform) replaceConnection(connection *displayConnection) {
	previous := p.connection
	p.connection = connection
	previous.close()
	callbacks := &p.common.Callbacks
	if callback := callbacks.KeyboardLayoutChange; callback != nil {
		callbacks.KeyboardLayoutChange = nil
		callback()
		if p.common.Callbacks.KeyboardLayoutChange == nil {
			p.common.Callbacks.KeyboardLayoutChange = callback
		}
	}
	if callback := callbacks.DisplaysChanged; callback != nil {
		callbacks.DisplaysChanged = nil
		callback()
		if p.common.Callbacks.DisplaysChanged == nil {
			p.common.Callbacks.DisplaysChanged = callback
		}
	}
}

func (p *linuxPlatform) applyPendingMode() {
	mode, ok := p.pendingMode.Get()
	if !ok {
		return
	}
	var result error
	if p.connection.hasWindows() {
		result = errors.New("a window was opened while switching display modes")
	} else {
		switch mode := mode.(type) {
		case gpui.WindowingRequestHeadless:
			p.replaceConnection(displayConnectionHeadless(common.NewHeadlessConnection()))
		case gpui.WindowingRequestWindowed:
			selected, found := selectBackend(p.allowedModes, mode.Environment)
			if !found {
				result = errors.New("the environment names no allowed Wayland or X11 display server")
			} else if connection, err := p.connect(selected, mode.Environment, &startupEnvironment{}); err != nil {
				result = err
			} else {
				p.replaceConnection(connection)
			}
		}
	}
	p.pendingMode = gpui.None[gpui.WindowingRequest]()
	if waiter := p.transitionWaiter; waiter != nil {
		p.transitionWaiter = nil
		_ = waiter.Send(result)
	}
}

func (p *linuxPlatform) requestMode(mode gpui.WindowingRequest) *gpui.Task[struct{}] {
	if p.pendingMode.IsSome() {
		return scheduler.ReadyTaskErr[struct{}](errors.New("a display mode transition is already in progress"))
	}
	isHeadless := p.connection.isHeadless()
	switch mode := mode.(type) {
	case gpui.WindowingRequestHeadless:
		if !p.allowedModes.Contains(gpui.WindowingModesHeadless) {
			return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("%s does not allow headless mode", windowingModesDebug(p.allowedModes)))
		}
		if isHeadless {
			return scheduler.ReadyTaskErr[struct{}](errors.New("already headless"))
		}
	case gpui.WindowingRequestWindowed:
		if !isHeadless {
			return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("already windowed (%s); switch to headless mode first", p.connection.compositorName()))
		}
		if _, found := selectBackend(p.allowedModes, mode.Environment); !found {
			return scheduler.ReadyTaskErr[struct{}](fmt.Errorf("the environment names no display server that %s allows", windowingModesDebug(p.allowedModes)))
		}
	}
	sender, receiver := scheduler.Oneshot[error]()
	if previous := p.transitionWaiter; previous != nil {
		previous.Close()
	}
	p.transitionWaiter = sender
	p.pendingMode = gpui.Some(mode)
	p.common.Signal.Wakeup()
	return p.common.ForegroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		result, err := receiver.Recv(s)
		if err != nil {
			return struct{}{}, errors.New("display mode transition was canceled")
		}
		return struct{}{}, result
	})
}

// Close is the platform's drop: it closes the display connection, then waits for X11
// clipboard handovers and stops the shared executors.
func (p *linuxPlatform) Close() {
	if p.closed {
		return
	}
	p.closed = true
	p.replaceConnection(displayConnectionHeadless(common.NewHeadlessConnection()))
	x11.WaitForClipboardHandovers()
	if p.powerModeCancel != nil {
		p.powerModeCancel()
		p.powerModeCancel = nil
	}
	if p.startupEnvironment != nil {
		p.startupEnvironment.close()
		p.startupEnvironment = nil
	}
	if waiter := p.transitionWaiter; waiter != nil {
		p.transitionWaiter = nil
		waiter.Close()
	}
	p.common.Close()
	p.ownedLoop.Close()
}

// PlatformInfo reports Linux or FreeBSD as a native target.
func (p *linuxPlatform) PlatformInfo() gpui.PlatformInfo {
	os := gpui.OperatingSystemLinux
	if runtime.GOOS == "freebsd" {
		os = gpui.OperatingSystemFreeBsd
	}
	return gpui.PlatformInfo{Os: os, Target: gpui.PlatformTargetNative}
}

// BackgroundExecutor returns the executor for background work.
func (p *linuxPlatform) BackgroundExecutor() gpui.BackgroundExecutor {
	return p.common.BackgroundExecutor
}

// ForegroundExecutor returns the executor for the main thread.
func (p *linuxPlatform) ForegroundExecutor() gpui.ForegroundExecutor {
	return p.common.ForegroundExecutor
}

// TextSystem returns the platform text system.
func (p *linuxPlatform) TextSystem() gpui.PlatformTextSystem {
	return p.common.TextSystem
}

// KeyboardLayout returns the current connection's keyboard layout.
func (p *linuxPlatform) KeyboardLayout() gpui.PlatformKeyboardLayout {
	return p.connection.keyboardLayout()
}

// KeyboardMapper returns the identity keyboard mapper.
func (p *linuxPlatform) KeyboardMapper() gpui.PlatformKeyboardMapper {
	return gpui.DummyKeyboardMapper{}
}

// OnKeyboardLayoutChange registers the keyboard layout change callback.
func (p *linuxPlatform) OnKeyboardLayoutChange(callback func()) {
	p.common.Callbacks.KeyboardLayoutChange = callback
}

// OnThermalStateChange ignores callback: Linux reports no thermal state.
func (p *linuxPlatform) OnThermalStateChange(callback func()) {}

// ThermalState is always nominal on Linux.
func (p *linuxPlatform) ThermalState() gpui.ThermalState {
	return gpui.ThermalStateNominal
}

// PowerMode returns the last power mode reported by power-profiles-daemon.
func (p *linuxPlatform) PowerMode() gpui.PowerMode {
	return p.common.PowerMode
}

// OnPowerModeChange registers callback and starts monitoring the system power mode once.
func (p *linuxPlatform) OnPowerModeChange(callback func()) {
	linuxCommon := p.common
	linuxCommon.Callbacks.PowerModeChange = callback
	if linuxCommon.PowerModeListener == nil {
		sender := linuxCommon.PowerSender.Clone()
		ctx, cancel := context.WithCancel(context.Background())
		p.powerModeCancel = cancel
		linuxCommon.PowerModeListener = linuxCommon.BackgroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
			defer sender.Close()
			conn, err := linuxCommon.SystemBus()
			if err == nil {
				err = dbus.MonitorPowerMode(ctx, conn, func(mode gpui.PowerMode) error {
					return sender.Send(common.SystemPowerEventModeChangedTo(mode))
				})
			}
			if err != nil && ctx.Err() == nil {
				slog.Debug(fmt.Sprintf("failed to monitor system power mode: %v", err))
			}
			return struct{}{}, nil
		})
	}
}

// PreventIdleSleep asks the Inhibit portal to keep the system awake until the guard is released.
func (p *linuxPlatform) PreventIdleSleep(reason string) *gpui.Task[*gpui.ActivityGuard] {
	executor := p.BackgroundExecutor()
	guardTx, guardRx := scheduler.Oneshot[scheduler.Result[*gpui.ActivityGuard]]()
	bus := p.common.SessionBus
	executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		guard, err := common.InhibitIdleSleep(reason, executor, bus)
		if sendErr := guardTx.Send(scheduler.Result[*gpui.ActivityGuard]{Value: guard, Err: err}); sendErr != nil && guard != nil {
			guard.Release()
		}
		return struct{}{}, nil
	}).Detach()
	return executor.Spawn(func(s *scheduler.Scope) (*gpui.ActivityGuard, error) {
		return common.AwaitIdleSleepPrevention(s, guardRx, executor)
	})
}

// Run connects in the initial mode, calls onFinishLaunching and runs the event loop until Quit.
func (p *linuxPlatform) Run(onFinishLaunching func()) {
	p.connectInitially()
	onFinishLaunching()

	eventLoop := p.eventLoop
	if eventLoop == nil {
		panic("App is already running")
	}
	p.eventLoop = nil
	if err := eventLoop.Run(gpui.None[time.Duration](), p.applyPendingMode); err != nil {
		slog.Error(err.Error())
	}

	quit := p.common.Callbacks.Quit
	p.common.Callbacks.Quit = nil
	if quit != nil {
		quit()
	}
	x11.WaitForClipboardHandovers()
}

// Quit stops the event loop.
func (p *linuxPlatform) Quit() {
	p.common.Signal.Stop()
}

// SetInitialWindowing sets the mode Run connects in.
func (p *linuxPlatform) SetInitialWindowing(request gpui.WindowingRequest) {
	p.initialWindowing = gpui.Some(request)
}

// RequestWindowing switches between headless and windowed modes between event-loop iterations.
func (p *linuxPlatform) RequestWindowing(request gpui.WindowingRequest) *gpui.Task[struct{}] {
	return p.requestMode(request)
}

// CompositorName names the display server the platform is connected to, or "headless".
func (p *linuxPlatform) CompositorName() string {
	return p.connection.compositorName()
}

// GraphicalEnvironment returns the connection's environment, or false while headless.
func (p *linuxPlatform) GraphicalEnvironment() (gpui.GraphicalEnvironment, bool) {
	return p.connection.graphicalEnvironment()
}

const restartScript = `
            while kill -0 "$0" 2>/dev/null; do
                sleep 0.1
            done

            app_path="$1"
            shift
            "$app_path" "$@"
            `

// Restart starts a script that waits for this process to exit and relaunches the app, then quits.
func (p *linuxPlatform) Restart(binaryPath gpui.Option[string], arguments []string) {
	appPID := strconv.Itoa(os.Getpid())
	appPath, ok := binaryPath.Get()
	if !ok {
		path, err := p.AppPath()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to get app path: %v", err))
			return
		}
		appPath = path
	}

	slog.Info(fmt.Sprintf("Restarting process, using app path: %q", appPath))

	args := append([]string{"bash", "-c", restartScript, appPID, appPath}, arguments...)
	restartProcess := exec.Command("/usr/bin/env", args...)
	restartProcess.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	if err := restartProcess.Start(); err != nil {
		slog.Error(fmt.Sprintf("failed to spawn restart script: %v", err))
		return
	}
	_ = restartProcess.Process.Release()
	p.Quit()
}

// Activate is not implemented on Linux.
func (p *linuxPlatform) Activate(ignoringOtherApps bool) {
	slog.Info("activate is not implemented on Linux, ignoring the call")
}

// Hide is not implemented on Linux.
func (p *linuxPlatform) Hide() {
	slog.Info("hide is not implemented on Linux, ignoring the call")
}

// HideOtherApps is not implemented on Linux.
func (p *linuxPlatform) HideOtherApps() {
	slog.Info("hide_other_apps is not implemented on Linux, ignoring the call")
}

// UnhideOtherApps is not implemented on Linux.
func (p *linuxPlatform) UnhideOtherApps() {
	slog.Info("unhide_other_apps is not implemented on Linux, ignoring the call")
}

// PrimaryDisplay returns the connection's primary display, or nil.
func (p *linuxPlatform) PrimaryDisplay() gpui.PlatformDisplay {
	return p.connection.primaryDisplay()
}

// Displays returns the connection's displays.
func (p *linuxPlatform) Displays() []gpui.PlatformDisplay {
	return p.connection.displays()
}

// OnDisplaysChanged registers the displays-changed callback.
func (p *linuxPlatform) OnDisplaysChanged(callback func()) {
	p.common.Callbacks.DisplaysChanged = callback
}

// IsScreenCaptureSupported reports whether the connection can capture screens.
func (p *linuxPlatform) IsScreenCaptureSupported() bool {
	return p.connection.isScreenCaptureSupported()
}

// ScreenCaptureSources returns the connection's capturable screens.
func (p *linuxPlatform) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]] {
	return p.connection.screenCaptureSources()
}

// ActiveWindow returns the focused window, if any.
func (p *linuxPlatform) ActiveWindow() (gpui.AnyWindowHandle, bool) {
	return p.connection.activeWindow()
}

// WindowStack returns the windows in stacking order, if the connection reports it.
func (p *linuxPlatform) WindowStack() ([]gpui.AnyWindowHandle, bool) {
	return p.connection.windowStack()
}

// OpenWindow opens a window on the current connection.
func (p *linuxPlatform) OpenWindow(handle gpui.AnyWindowHandle, options gpui.WindowParams) (gpui.PlatformWindow, error) {
	return p.connection.openWindow(handle, options)
}

// OpenURL opens url with the desktop's handler.
func (p *linuxPlatform) OpenURL(url string) {
	p.connection.openURI(url)
}

// OnOpenURLs registers the open-URLs callback.
func (p *linuxPlatform) OnOpenURLs(callback func(urls []string)) {
	p.common.Callbacks.OpenURLs = callback
}

// PromptForPaths asks the FileChooser portal for files or directories to open.
func (p *linuxPlatform) PromptForPaths(options gpui.PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[[]string]]] {
	doneTx, doneRx := scheduler.Oneshot[scheduler.Result[gpui.Option[[]string]]]()
	identifier := p.connection.windowIdentifier()
	background := p.BackgroundExecutor()
	bus := p.common.SessionBus
	p.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		title := "Open File"
		if options.Directories {
			title = "Open Folder"
		}
		parentWindow, _ := identifier.Await(s)
		portalOptions := map[string]godbus.Variant{
			"modal":     godbus.MakeVariant(true),
			"multiple":  godbus.MakeVariant(options.Multiple),
			"directory": godbus.MakeVariant(options.Directories),
		}
		if prompt, ok := options.Prompt.Get(); ok {
			portalOptions["accept_label"] = godbus.MakeVariant(prompt.AsStr())
		}
		request := background.Spawn(func(s *scheduler.Scope) (gpui.Option[[]string], error) {
			uris, err := fileChooserRequest(bus, "OpenFile", parentWindow.UnwrapOrDefault(), title, portalOptions)
			if err != nil {
				return gpui.None[[]string](), err
			}
			if !uris.IsSome() {
				return gpui.None[[]string](), nil
			}
			paths := []string{}
			for _, uri := range uris.UnwrapOrDefault() {
				if path, ok := fileURIToPath(uri); ok {
					paths = append(paths, path)
				}
			}
			return gpui.Some(paths), nil
		})
		paths, err := request.Await(s)
		_ = doneTx.Send(scheduler.Result[gpui.Option[[]string]]{Value: paths, Err: err})
		return struct{}{}, nil
	}).Detach()
	return doneRx
}

// PromptForNewPath asks the FileChooser portal where to save a file.
func (p *linuxPlatform) PromptForNewPath(directory string, suggestedName gpui.Option[string]) *scheduler.OneshotReceiver[scheduler.Result[gpui.Option[string]]] {
	doneTx, doneRx := scheduler.Oneshot[scheduler.Result[gpui.Option[string]]]()
	identifier := p.connection.windowIdentifier()
	background := p.BackgroundExecutor()
	bus := p.common.SessionBus
	p.ForegroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		parentWindow, _ := identifier.Await(s)
		if strings.ContainsRune(directory, 0) {
			panic("pathbuf should not be nul terminated")
		}
		portalOptions := map[string]godbus.Variant{
			"modal":          godbus.MakeVariant(true),
			"current_folder": godbus.MakeVariant(append([]byte(directory), 0)),
		}
		if name, ok := suggestedName.Get(); ok {
			portalOptions["current_name"] = godbus.MakeVariant(name)
		}
		request := background.Spawn(func(s *scheduler.Scope) (gpui.Option[string], error) {
			uris, err := fileChooserRequest(bus, "SaveFile", parentWindow.UnwrapOrDefault(), "Save File", portalOptions)
			if err != nil || !uris.IsSome() {
				return gpui.None[string](), err
			}
			list := uris.UnwrapOrDefault()
			if len(list) == 0 {
				return gpui.None[string](), nil
			}
			path, ok := fileURIToPath(list[0])
			return gpui.OptionFrom(path, ok), nil
		})
		path, err := request.Await(s)
		_ = doneTx.Send(scheduler.Result[gpui.Option[string]]{Value: path, Err: err})
		return struct{}{}, nil
	}).Detach()
	return doneRx
}

const (
	portalService            = "org.freedesktop.portal.Desktop"
	portalPath               = godbus.ObjectPath("/org/freedesktop/portal/desktop")
	portalRequestInterface   = "org.freedesktop.portal.Request"
	fileChooserInterface     = "org.freedesktop.portal.FileChooser"
	dbusErrorServiceUnknown  = "org.freedesktop.DBus.Error.ServiceUnknown"
	dbusErrorUnknownMethod   = "org.freedesktop.DBus.Error.UnknownMethod"
	dbusErrorUnknownIface    = "org.freedesktop.DBus.Error.UnknownInterface"
	dbusErrorUnknownProperty = "org.freedesktop.DBus.Error.UnknownProperty"
	dbusErrorInvalidArgs     = "org.freedesktop.DBus.Error.InvalidArgs"
)

func isPortalNotFound(err error) bool {
	var dbusErr godbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case dbusErrorServiceUnknown, dbusErrorUnknownMethod, dbusErrorUnknownIface, dbusErrorUnknownProperty, dbusErrorInvalidArgs:
			return true
		}
	}
	var dbusErrPtr *godbus.Error
	if errors.As(err, &dbusErrPtr) && dbusErrPtr != nil {
		switch dbusErrPtr.Name {
		case dbusErrorServiceUnknown, dbusErrorUnknownMethod, dbusErrorUnknownIface, dbusErrorUnknownProperty, dbusErrorInvalidArgs:
			return true
		}
	}
	return false
}

func fileChooserRequest(bus func() (*godbus.Conn, error), method, parentWindow, title string, options map[string]godbus.Variant) (gpui.Option[[]string], error) {
	conn, err := bus()
	if err != nil {
		return gpui.None[[]string](), err
	}
	if _, err := conn.Object(portalService, portalPath).GetProperty(fileChooserInterface + ".version"); err != nil {
		if isPortalNotFound(err) {
			return gpui.None[[]string](), errors.New(common.FilePickerPortalMissing)
		}
		return gpui.None[[]string](), err
	}
	var tokenBytes [8]byte
	_, _ = rand.Read(tokenBytes[:])
	token := "gpui" + hex.EncodeToString(tokenBytes[:])
	options["handle_token"] = godbus.MakeVariant(token)
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	path := godbus.ObjectPath(fmt.Sprintf("%s/request/%s/%s", portalPath, sender, token))

	match := []godbus.MatchOption{
		godbus.WithMatchInterface(portalRequestInterface),
		godbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		return gpui.None[[]string](), err
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()
	signals := make(chan *godbus.Signal, 16)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	var handle godbus.ObjectPath
	if err := conn.Object(portalService, portalPath).Call(fileChooserInterface+"."+method, 0, parentWindow, title, options).Store(&handle); err != nil {
		if isPortalNotFound(err) {
			return gpui.None[[]string](), errors.New(common.FilePickerPortalMissing)
		}
		return gpui.None[[]string](), err
	}
	if handle != "" {
		path = handle
	}
	owner, err := dbus.NameOwner(conn, portalService)
	if err != nil {
		return gpui.None[[]string](), err
	}
	for signal := range signals {
		if signal.Sender != owner || signal.Path != path || signal.Name != portalRequestInterface+".Response" || len(signal.Body) < 2 {
			continue
		}
		if response, _ := signal.Body[0].(uint32); response != 0 {
			return gpui.None[[]string](), nil
		}
		results, _ := signal.Body[1].(map[string]godbus.Variant)
		var uris []string
		if value, ok := results["uris"]; ok {
			_ = value.Store(&uris)
		}
		return gpui.Some(uris), nil
	}
	return gpui.None[[]string](), errors.New("D-Bus connection closed")
}

func fileURIToPath(uri string) (string, bool) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return "", false
	}
	if host := parsed.Host; host != "" && host != "localhost" {
		return "", false
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return "", false
	}
	return parsed.Path, true
}

// CanSelectMixedFilesAndDirs is false: the FileChooser portal picks either files or directories.
func (p *linuxPlatform) CanSelectMixedFilesAndDirs() bool {
	return false
}

// RevealPath shows path in the file manager.
func (p *linuxPlatform) RevealPath(path string) {
	p.connection.revealPath(path)
}

// OpenWithSystem opens path with xdg-open on a background goroutine.
func (p *linuxPlatform) OpenWithSystem(path string) {
	target, err := openWithSystemTarget(path)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	environment, hasEnvironment := p.connection.graphicalEnvironment()
	p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		command := exec.Command("xdg-open", target)
		if hasEnvironment {
			environment.ApplyTo(command)
		}
		if err := command.Run(); err != nil {
			slog.Error(fmt.Errorf("invoking xdg-open: %w", err).Error())
		}
		return struct{}{}, nil
	}).Detach()
}

func openWithSystemTarget(path string) (string, error) {
	if path == "" {
		return "", errors.New("OpenWithSystem: empty path")
	}
	if !filepath.IsAbs(path) {
		if scheme, _, found := strings.Cut(path, ":"); found && !strings.Contains(scheme, "/") {
			return "", fmt.Errorf("OpenWithSystem: refusing URL-shaped target %q", path)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("OpenWithSystem: %w", err)
		}
		path = abs
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("OpenWithSystem: %w", err)
	}
	return path, nil
}

// OnQuit registers the quit callback.
func (p *linuxPlatform) OnQuit(callback func() bool) {
	p.common.Callbacks.Quit = callback
}

// OnReopen registers the reopen callback.
func (p *linuxPlatform) OnReopen(callback func()) {
	p.common.Callbacks.Reopen = callback
}

// OnSystemSleep registers the sleep callback and starts the power listener.
func (p *linuxPlatform) OnSystemSleep(callback func()) {
	p.common.Callbacks.SystemSleep = callback
	p.common.StartPowerListener()
}

// OnSystemWake registers the wake callback and starts the power listener.
func (p *linuxPlatform) OnSystemWake(callback func()) {
	p.common.Callbacks.SystemWake = callback
	p.common.StartPowerListener()
}

// SetAppIdentity records the app name shown in notifications.
func (p *linuxPlatform) SetAppIdentity(identifier string, name string) {
	p.common.AppName = gpui.Some(name)
}

// ShowSystemNotification shows notification through the notification daemon.
func (p *linuxPlatform) ShowSystemNotification(notification gpui.SystemNotification) {
	notifications, err := p.common.SystemNotifications()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to show system notification: %v", err))
		return
	}
	notifications.Show(p.common.AppName, notification)
}

// DismissSystemNotification dismisses the notification tagged tag.
func (p *linuxPlatform) DismissSystemNotification(tag string) *gpui.Task[struct{}] {
	if notifications, err := p.common.SystemNotifications(); err == nil {
		notifications.Dismiss(tag)
	}
	return scheduler.ReadyTask(struct{}{})
}

// OnSystemNotificationResponse registers the notification response callback.
func (p *linuxPlatform) OnSystemNotificationResponse(callback func(response gpui.SystemNotificationResponse)) {
	notifications, err := p.common.SystemNotifications()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to listen for system notification responses: %v", err))
		return
	}
	executor := p.common.ForegroundExecutor.ToSend()
	notifications.OnResponse(func(f func()) {
		executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
			f()
			return struct{}{}, nil
		}).Detach()
	}, callback)
}

// OnAppMenuAction registers the app menu action callback.
func (p *linuxPlatform) OnAppMenuAction(callback func(action gpui.Action)) {
	p.common.Callbacks.AppMenuAction = callback
}

// OnWillOpenAppMenu registers the will-open-app-menu callback.
func (p *linuxPlatform) OnWillOpenAppMenu(callback func()) {
	p.common.Callbacks.WillOpenAppMenu = callback
}

// OnValidateAppMenuCommand registers the menu command validation callback.
func (p *linuxPlatform) OnValidateAppMenuCommand(callback func(action gpui.Action) bool) {
	p.common.Callbacks.ValidateAppMenuCommand = callback
}

// AppPath returns the path of the running executable.
func (p *linuxPlatform) AppPath() (string, error) {
	return os.Executable()
}

// SetMenus stores the app menus.
func (p *linuxPlatform) SetMenus(menus []gpui.Menu, keymap *gpui.Keymap) {
	owned := make([]gpui.OwnedMenu, 0, len(menus))
	for _, menu := range menus {
		owned = append(owned, menu.Owned())
	}
	p.common.Menus = owned
}

// GetMenus returns the stored app menus.
func (p *linuxPlatform) GetMenus() ([]gpui.OwnedMenu, bool) {
	return append([]gpui.OwnedMenu(nil), p.common.Menus...), true
}

// SetDockMenu does nothing on Linux.
func (p *linuxPlatform) SetDockMenu(menu []gpui.MenuItem, keymap *gpui.Keymap) {}

// PathForAuxiliaryExecutable is not implemented on Linux.
func (p *linuxPlatform) PathForAuxiliaryExecutable(name string) (string, error) {
	return "", errors.New("Platform<LinuxPlatform>::path_for_auxiliary_executable is not implemented yet")
}

// SetCursorStyle sets the cursor on the current connection.
func (p *linuxPlatform) SetCursorStyle(style gpui.CursorStyle) {
	p.connection.setCursorStyle(style)
}

// HideCursorUntilMouseMoves hides the cursor until the mouse moves.
func (p *linuxPlatform) HideCursorUntilMouseMoves() {
	p.connection.hideCursorUntilMouseMoves()
}

// IsCursorVisible reports whether the cursor is visible.
func (p *linuxPlatform) IsCursorVisible() bool {
	return p.connection.isCursorVisible()
}

// ShouldAutoHideScrollbars returns the desktop's scrollbar setting.
func (p *linuxPlatform) ShouldAutoHideScrollbars() bool {
	return p.common.AutoHideScrollbars
}

// WriteCredentials stores credentials in the Secret Service.
func (p *linuxPlatform) WriteCredentials(url string, username string, password []byte) *gpui.Task[struct{}] {
	password = append([]byte(nil), password...)
	bus := p.common.SessionBus
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		conn, err := bus()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, dbus.WriteCredentials(conn, url, username, password)
	})
}

// ReadCredentials reads credentials from the Secret Service.
func (p *linuxPlatform) ReadCredentials(url string) *gpui.Task[gpui.Option[gpui.Credentials]] {
	bus := p.common.SessionBus
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (gpui.Option[gpui.Credentials], error) {
		conn, err := bus()
		if err != nil {
			return gpui.None[gpui.Credentials](), err
		}
		username, password, ok, err := dbus.ReadCredentials(conn, url)
		if err != nil || !ok {
			return gpui.None[gpui.Credentials](), err
		}
		return gpui.Some(gpui.Credentials{Username: username, Password: password}), nil
	})
}

// DeleteCredentials deletes credentials from the Secret Service.
func (p *linuxPlatform) DeleteCredentials(url string) *gpui.Task[struct{}] {
	bus := p.common.SessionBus
	return p.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (struct{}, error) {
		conn, err := bus()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, dbus.DeleteCredentials(conn, url)
	})
}

// WindowAppearance returns the desktop's appearance.
func (p *linuxPlatform) WindowAppearance() gpui.WindowAppearance {
	return p.common.Appearance
}

// ButtonLayout returns the desktop's window button layout.
func (p *linuxPlatform) ButtonLayout() (gpui.WindowButtonLayout, bool) {
	return p.common.ButtonLayout, true
}

// TextScaleFactor returns the desktop's text scale factor.
func (p *linuxPlatform) TextScaleFactor() float32 {
	return p.common.TextScaleFactor
}

// OnTextScaleFactorChange registers the text scale factor callback.
func (p *linuxPlatform) OnTextScaleFactorChange(callback func()) {
	p.common.Callbacks.TextScaleFactorChange = callback
}

// RegisterURLScheme is not implemented on Linux.
func (p *linuxPlatform) RegisterURLScheme(scheme string) *gpui.Task[struct{}] {
	return scheduler.ReadyTaskErr[struct{}](errors.New("register_url_scheme unimplemented"))
}

// WriteToPrimary writes item to the primary selection.
func (p *linuxPlatform) WriteToPrimary(item gpui.ClipboardItem) {
	p.connection.writeToPrimary(item)
}

// WriteToClipboard writes item to the clipboard.
func (p *linuxPlatform) WriteToClipboard(item gpui.ClipboardItem) {
	p.connection.writeToClipboard(item)
}

// ReadFromPrimary reads the primary selection.
func (p *linuxPlatform) ReadFromPrimary() (gpui.ClipboardItem, bool) {
	return p.connection.readFromPrimary()
}

// ReadFromClipboard reads the clipboard.
func (p *linuxPlatform) ReadFromClipboard() (gpui.ClipboardItem, bool) {
	return p.connection.readFromClipboard()
}

// AddRecentDocument does nothing on Linux.
func (p *linuxPlatform) AddRecentDocument(path string) {}

var _ gpui.Platform = (*linuxPlatform)(nil)
