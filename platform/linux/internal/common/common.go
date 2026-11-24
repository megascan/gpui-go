//go:build linux

package common

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	godbus "github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/platform/linux/internal/dbus"
	"github.com/megascan/gpui-go/scheduler"
	"golang.org/x/sys/unix"
)

// ScrollLines is the number of lines one wheel click scrolls.
const ScrollLines float32 = 3.0

// DoubleClickInterval is GTK's default double-click time.
const DoubleClickInterval = 400 * time.Millisecond

// DoubleClickDistance is GTK's default double-click distance.
const DoubleClickDistance gpui.Pixels = 5.0

// KeyringLabel labels the credentials gpui writes to the Secret Service.
const KeyringLabel = "zed-github-account"

// FilePickerPortalMissing is the error shown when no file chooser portal exists.
const FilePickerPortalMissing = "Couldn't open file picker due to missing xdg-desktop-portal implementation."

// PlatformHandlers are the app's platform callbacks; a nil field is an unset callback.
type PlatformHandlers struct {
	OpenURLs               func(urls []string)
	Quit                   func() bool
	Reopen                 func()
	AppMenuAction          func(action gpui.Action)
	WillOpenAppMenu        func()
	ValidateAppMenuCommand func(action gpui.Action) bool
	KeyboardLayoutChange   func()
	TextScaleFactorChange  func()
	SystemSleep            func()
	SystemWake             func()
	PowerModeChange        func()
	DisplaysChanged        func()
}

// SystemPowerEventKind selects a SystemPowerEvent variant.
type SystemPowerEventKind uint8

const (
	// SystemPowerEventSleep is sent before the system sleeps.
	SystemPowerEventSleep SystemPowerEventKind = iota
	// SystemPowerEventWake is sent after the system wakes.
	SystemPowerEventWake
	// SystemPowerEventModeChanged is sent when the power mode changes to Mode.
	SystemPowerEventModeChanged
)

// SystemPowerEvent is a system power change, forwarded from D-Bus to the client's event loop.
type SystemPowerEvent struct {
	Kind SystemPowerEventKind
	Mode gpui.PowerMode
}

// SystemPowerEventModeChangedTo returns the ModeChanged event for mode.
func SystemPowerEventModeChangedTo(mode gpui.PowerMode) SystemPowerEvent {
	return SystemPowerEvent{Kind: SystemPowerEventModeChanged, Mode: mode}
}

// LinuxClient is what a display-server connection (Wayland or X11) provides to LinuxPlatform,
// the methods display_connection.rs forwards to it. Close is its drop.
type LinuxClient interface {
	CompositorName() string
	HasWindows() bool
	GraphicalEnvironment() gpui.GraphicalEnvironment
	KeyboardLayout() gpui.PlatformKeyboardLayout
	Displays() []gpui.PlatformDisplay
	PrimaryDisplay() gpui.PlatformDisplay
	OpenWindow(handle gpui.AnyWindowHandle, params gpui.WindowParams) (gpui.PlatformWindow, error)
	ActiveWindow() (gpui.AnyWindowHandle, bool)
	WindowStack() ([]gpui.AnyWindowHandle, bool)
	SetCursorStyle(style gpui.CursorStyle)
	HideCursorUntilMouseMoves()
	IsCursorVisible() bool
	OpenURI(uri string)
	RevealPath(path string)
	WriteToPrimary(item gpui.ClipboardItem)
	WriteToClipboard(item gpui.ClipboardItem)
	ReadFromPrimary() (gpui.ClipboardItem, bool)
	ReadFromClipboard() (gpui.ClipboardItem, bool)
	WindowIdentifier() *gpui.Task[gpui.Option[string]]
	IsScreenCaptureSupported() bool
	ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]gpui.ScreenCaptureSource]]
	Close()
}

// LinuxCommon is the session-independent state every Linux connection shares: executors, text
// system, settings, callbacks and the power listener. It is used only on the main goroutine.
type LinuxCommon struct {
	BackgroundExecutor   gpui.BackgroundExecutor
	ForegroundExecutor   gpui.ForegroundExecutor
	TextSystem           gpui.PlatformTextSystem
	Appearance           gpui.WindowAppearance
	AutoHideScrollbars   bool
	ButtonLayout         gpui.WindowButtonLayout
	TextScaleFactor      float32
	Callbacks            PlatformHandlers
	Signal               LoopSignal
	Menus                []gpui.OwnedMenu
	AppName              gpui.Option[string]
	PowerSender          *ChannelSender[SystemPowerEvent]
	PowerMode            gpui.PowerMode
	PowerModeListener    *gpui.Task[struct{}]
	AfterRunnable        func()
	ConnectSessionBus    func() (*godbus.Conn, error)
	ConnectSystemBus     func() (*godbus.Conn, error)
	dispatcher           *LinuxDispatcher
	mainThread           gpui.ThreadID
	powerListenerStarted bool
	systemNotifications  *dbus.SystemNotificationState
	sessionBus           lazyBus
	systemBus            lazyBus
}

type lazyBus struct {
	once sync.Once
	mu   sync.Mutex
	conn *godbus.Conn
	err  error
}

func (b *lazyBus) get(connect func() (*godbus.Conn, error)) (*godbus.Conn, error) {
	b.once.Do(func() {
		if connect == nil {
			b.err = errors.New("no D-Bus connection available")
			return
		}
		conn, err := connect()
		b.mu.Lock()
		b.conn, b.err = conn, err
		b.mu.Unlock()
	})
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.conn, b.err
}

func (b *lazyBus) close() {
	b.once.Do(func() {
		b.mu.Lock()
		b.err = errors.New("D-Bus connection closed")
		b.mu.Unlock()
	})
	b.mu.Lock()
	conn := b.conn
	b.conn = nil
	b.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

// NewLinuxCommon creates the shared state with its dispatcher, returning the receivers that
// RegisterSources puts on the platform's loop.
func NewLinuxCommon(signal LoopSignal) (*LinuxCommon, *PriorityQueueCalloopReceiver[gpui.RunnableVariant], *Channel[SystemPowerEvent]) {
	mainSender, mainReceiver := NewPriorityQueueCalloopReceiver[gpui.RunnableVariant]()
	powerSender, powerReceiver, err := NewChannel[SystemPowerEvent]()
	if err != nil {
		panic(fmt.Sprintf("failed to create the power channel: %v", err))
	}
	textSystem := newTextSystem()
	dispatcher := NewLinuxDispatcher(mainSender)
	backgroundExecutor := gpui.NewBackgroundExecutor(dispatcher)
	common := &LinuxCommon{
		BackgroundExecutor: backgroundExecutor,
		ForegroundExecutor: gpui.NewForegroundExecutor(dispatcher),
		TextSystem:         textSystem,
		Appearance:         gpui.WindowAppearanceLight,
		ButtonLayout:       gpui.WindowButtonLayoutLinuxDefault(),
		TextScaleFactor:    1.0,
		Signal:             signal,
		PowerSender:        powerSender,
		PowerMode:          gpui.PowerModeUnknown,
		ConnectSessionBus:  func() (*godbus.Conn, error) { return godbus.ConnectSessionBus() },
		ConnectSystemBus:   func() (*godbus.Conn, error) { return godbus.ConnectSystemBus() },
		dispatcher:         dispatcher,
		mainThread:         gpui.NewMainThreadID(gpui.None[string]()),
	}
	return common, mainReceiver, powerReceiver
}

// Close stops the dispatcher's goroutines, the power listener and closes the D-Bus connections.
func (c *LinuxCommon) Close() {
	if c.PowerModeListener != nil {
		c.PowerModeListener.Cancel()
		c.PowerModeListener = nil
	}
	c.PowerSender.Close()
	c.dispatcher.Close()
	c.sessionBus.close()
	c.systemBus.close()
}

// SessionBus returns the session bus, connecting on first use.
func (c *LinuxCommon) SessionBus() (*godbus.Conn, error) {
	return c.sessionBus.get(c.ConnectSessionBus)
}

// SystemBus returns the system bus, connecting on first use.
func (c *LinuxCommon) SystemBus() (*godbus.Conn, error) {
	return c.systemBus.get(c.ConnectSystemBus)
}

// SystemNotifications returns the notification state over the session bus, created on first use.
func (c *LinuxCommon) SystemNotifications() (*dbus.SystemNotificationState, error) {
	if c.systemNotifications == nil {
		conn, err := c.SessionBus()
		if err != nil {
			return nil, err
		}
		c.systemNotifications = dbus.NewSystemNotificationState(conn)
	}
	return c.systemNotifications, nil
}

// StartPowerListener starts listening for logind sleep and wake signals once.
func (c *LinuxCommon) StartPowerListener() {
	if !c.powerListenerStarted {
		powerSender := c.PowerSender.Clone()
		go func() {
			defer powerSender.Close()
			conn, err := c.SystemBus()
			if err == nil {
				err = listenForSystemPowerEvents(conn, powerSender)
			}
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to listen for system sleep/wake events: %v", err))
			}
		}()
		c.powerListenerStarted = true
	}
}

// RegisterSources registers the foreground executor and power listener on a loop shared with
// display clients. Foreground runnables run as idle callbacks, after the loop has dispatched all
// pending events.
func RegisterSources(common *LinuxCommon, loopHandle *LoopHandle, mainReceiver *PriorityQueueCalloopReceiver[gpui.RunnableVariant], powerReceiver *Channel[SystemPowerEvent]) error {
	if _, err := InsertSource(loopHandle, mainReceiver, func(event ChannelEvent[gpui.RunnableVariant]) {
		if event.Closed {
			return
		}
		runnable := event.Msg
		loopHandle.InsertIdle(func() {
			runProfiled(common.mainThread, runnable)
			if afterRunnable := common.AfterRunnable; afterRunnable != nil {
				afterRunnable()
			}
		})
	}); err != nil {
		return fmt.Errorf("failed to register foreground executor: %w", err)
	}
	if _, err := InsertSource(loopHandle, powerReceiver, func(event ChannelEvent[SystemPowerEvent]) {
		if !event.Closed {
			handleSystemPowerEvent(common, event.Msg)
		}
	}); err != nil {
		return fmt.Errorf("failed to register power listener: %w", err)
	}
	return nil
}

func powerSlot(callbacks *PlatformHandlers, event SystemPowerEvent) *func() {
	switch event.Kind {
	case SystemPowerEventSleep:
		return &callbacks.SystemSleep
	case SystemPowerEventWake:
		return &callbacks.SystemWake
	default:
		return &callbacks.PowerModeChange
	}
}

func handleSystemPowerEvent(common *LinuxCommon, event SystemPowerEvent) {
	if event.Kind == SystemPowerEventModeChanged {
		if common.PowerMode == event.Mode {
			return
		}
		common.PowerMode = event.Mode
	}
	slot := powerSlot(&common.Callbacks, event)
	callback := *slot
	*slot = nil
	if callback != nil {
		callback()
		slot := powerSlot(&common.Callbacks, event)
		if *slot == nil {
			*slot = callback
		}
	}
}

const (
	login1Service   = "org.freedesktop.login1"
	login1Path      = godbus.ObjectPath("/org/freedesktop/login1")
	login1Interface = "org.freedesktop.login1.Manager"
)

func listenForSystemPowerEvents(conn *godbus.Conn, powerSender *ChannelSender[SystemPowerEvent]) error {
	if err := conn.AddMatchSignal(
		godbus.WithMatchSender(login1Service),
		godbus.WithMatchObjectPath(login1Path),
		godbus.WithMatchInterface(login1Interface),
		godbus.WithMatchMember("PrepareForSleep"),
	); err != nil {
		return err
	}
	signals := make(chan *godbus.Signal, 16)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	pin := dbus.NewOwnerPin(conn, login1Service)
	for message := range signals {
		if message.Path != login1Path || message.Name != login1Interface+".PrepareForSleep" || !pin.Matches(message.Sender) {
			continue
		}
		if len(message.Body) != 1 {
			return fmt.Errorf("PrepareForSleep: expected one argument, got %d", len(message.Body))
		}
		start, ok := message.Body[0].(bool)
		if !ok {
			return fmt.Errorf("PrepareForSleep: expected a boolean, got %T", message.Body[0])
		}
		event := SystemPowerEvent{Kind: SystemPowerEventWake}
		if start {
			event.Kind = SystemPowerEventSleep
		}
		_ = powerSender.Send(event)
	}
	return nil
}

const (
	portalService          = "org.freedesktop.portal.Desktop"
	portalPath             = godbus.ObjectPath("/org/freedesktop/portal/desktop")
	portalRequestInterface = "org.freedesktop.portal.Request"
)

type portalResponseError struct{ response uint32 }

func (e *portalResponseError) Error() string {
	if e.response == 1 {
		return "Portal request didn't succeed: Cancelled"
	}
	return "Portal request didn't succeed: Other"
}

func isPortalResponse(err error) bool {
	var responseErr *portalResponseError
	return errors.As(err, &responseErr)
}

func isPortalCancelled(err error) bool {
	var responseErr *portalResponseError
	return errors.As(err, &responseErr) && responseErr.response == 1
}

type portalRequest struct {
	conn *godbus.Conn
	path godbus.ObjectPath
}

func (r portalRequest) close() error {
	return r.conn.Object(portalService, r.path).Call(portalRequestInterface+".Close", 0).Err
}

func portalCall(conn *godbus.Conn, method string, options map[string]godbus.Variant, args ...any) (portalRequest, error) {
	var tokenBytes [8]byte
	_, _ = rand.Read(tokenBytes[:])
	token := "gpui" + hex.EncodeToString(tokenBytes[:])
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	path := godbus.ObjectPath(fmt.Sprintf("%s/request/%s/%s", portalPath, sender, token))
	options["handle_token"] = godbus.MakeVariant(token)

	match := []godbus.MatchOption{
		godbus.WithMatchInterface(portalRequestInterface),
		godbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		return portalRequest{}, err
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()
	signals := make(chan *godbus.Signal, 16)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	var handle godbus.ObjectPath
	if err := conn.Object(portalService, portalPath).Call(method, 0, append(args, options)...).Store(&handle); err != nil {
		return portalRequest{}, err
	}
	if handle != "" {
		path = handle
	}
	pin := dbus.NewOwnerPin(conn, portalService)
	timer := time.NewTimer(PortalResponseTimeout)
	defer timer.Stop()
	for {
		var signal *godbus.Signal
		var open bool
		select {
		case signal, open = <-signals:
		case <-timer.C:
			_ = portalRequest{conn: conn, path: path}.close()
			return portalRequest{}, fmt.Errorf("timed out waiting for portal response after %v", PortalResponseTimeout)
		}
		if !open {
			return portalRequest{}, errors.New("D-Bus connection closed")
		}
		if signal.Path != path || signal.Name != portalRequestInterface+".Response" || len(signal.Body) < 1 || !pin.Matches(signal.Sender) {
			continue
		}
		response, _ := signal.Body[0].(uint32)
		switch response {
		case 0:
			return portalRequest{conn: conn, path: path}, nil
		default:
			return portalRequest{}, &portalResponseError{response: response}
		}
	}
}

// PortalResponseTimeout bounds how long a portal request waits for its Response signal.
const PortalResponseTimeout = 5 * time.Minute

func openFileRequestSendURI(conn *godbus.Conn, uri *url.URL, activationToken gpui.Option[string]) error {
	options := map[string]godbus.Variant{}
	if token, ok := activationToken.Get(); ok {
		options["activation_token"] = godbus.MakeVariant(token)
	}
	_, err := portalCall(conn, "org.freedesktop.portal.OpenURI.OpenURI", options, "", uri.String())
	return err
}

func openDirectoryRequestSend(conn *godbus.Conn, dir *os.File, activationToken gpui.Option[string]) error {
	options := map[string]godbus.Variant{}
	if token, ok := activationToken.Get(); ok {
		options["activation_token"] = godbus.MakeVariant(token)
	}
	_, err := portalCall(conn, "org.freedesktop.portal.OpenURI.OpenDirectory", options, "", godbus.UnixFD(dir.Fd()))
	return err
}

func spawnDetached(executor gpui.BackgroundExecutor, body func()) {
	executor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		body()
		return struct{}{}, nil
	}).Detach()
}

var openURISchemes = []string{"http", "https", "mailto"}

// OpenURIInternal opens uri with the desktop's launchers, falling back to the OpenURI portal.
func OpenURIInternal(executor gpui.BackgroundExecutor, bus func() (*godbus.Conn, error), uri string, activationToken gpui.Option[string], environment gpui.GraphicalEnvironment) {
	parsed, err := url.Parse(uri)
	if err == nil && !parsed.IsAbs() {
		err = fmt.Errorf("relative URL without a base: %q", uri)
	}
	if err == nil && !slices.Contains(openURISchemes, parsed.Scheme) {
		err = fmt.Errorf("refusing to open URL with scheme %q: %q", parsed.Scheme, uri)
	}
	if err != nil {
		slog.Error(err.Error())
		return
	}
	if parsed.Scheme == "mailto" {
		parsed.RawQuery = filterMailtoQuery(parsed.RawQuery)
	}
	spawnDetached(executor, func() {
		opened := runOpenCommands(parsed.String(), activationToken, environment)
		if !opened {
			conn, err := bus()
			if err == nil {
				err = openFileRequestSendURI(conn, parsed, activationToken)
			}
			if err != nil && !isPortalCancelled(err) {
				slog.Error(fmt.Sprintf("Failed to open with dbus: %v", err))
			}
		}
	})
}

var mailtoAllowedFields = []string{"to", "cc", "bcc", "subject", "body"}

func filterMailtoQuery(raw string) string {
	var kept []string
	for field := range strings.SplitSeq(raw, "&") {
		key, _, _ := strings.Cut(field, "=")
		name, err := url.QueryUnescape(key)
		if err != nil {
			continue
		}
		if slices.Contains(mailtoAllowedFields, strings.ToLower(name)) {
			kept = append(kept, field)
		}
	}
	return strings.Join(kept, "&")
}

// RevealPathInternal opens the directory of path with the OpenURI portal, falling back to the
// desktop's launchers.
func RevealPathInternal(executor gpui.BackgroundExecutor, bus func() (*godbus.Conn, error), path string, activationToken gpui.Option[string], environment gpui.GraphicalEnvironment) {
	spawnDetached(executor, func() {
		dir, err := openRevealTarget(path)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		defer dir.Close()
		path = dir.Name()
		conn, err := bus()
		if err == nil {
			err = openDirectoryRequestSend(conn, dir, activationToken)
		}
		if err == nil || isPortalResponse(err) {
			return
		}
		slog.Error(fmt.Sprintf("Failed to open with dbus: %v", err))
		directory := path
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			directory = filepath.Dir(path)
			if directory == "." && !strings.HasPrefix(path, ".") {
				directory = ""
			}
		}
		runOpenCommands(directory, activationToken, environment)
	})
}

func openRevealTarget(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.IsDir() || info.Mode().IsRegular() {
		return file, nil
	}
	file.Close()
	return os.OpenFile(filepath.Dir(path), os.O_RDONLY|unix.O_NONBLOCK, 0)
}

type launcherSpec struct {
	program           string
	args              []string
	supportsSeparator bool
}

var desktopLaunchers = []launcherSpec{
	{program: "xdg-open"},
	{program: "gio", args: []string{"open"}},
	{program: "gnome-open"},
	{program: "kde-open6", supportsSeparator: true},
	{program: "kde-open5", supportsSeparator: true},
	{program: "kde-open", supportsSeparator: true},
}

func optionSafePath(path string) string {
	if strings.HasPrefix(path, "-") {
		return "./" + path
	}
	return path
}

func openCommands(target string) []*exec.Cmd {
	safePath := optionSafePath(target)
	commands := make([]*exec.Cmd, 0, len(desktopLaunchers))
	for _, spec := range desktopLaunchers {
		args := append([]string(nil), spec.args...)
		if spec.supportsSeparator {
			args = append(args, "--", target)
		} else {
			args = append(args, safePath)
		}
		commands = append(commands, exec.Command(spec.program, args...))
	}
	return commands
}

func runOpenCommands(target string, activationToken gpui.Option[string], environment gpui.GraphicalEnvironment) bool {
	for _, command := range openCommands(target) {
		environment.ApplyTo(command)
		if token, ok := activationToken.Get(); ok {
			command.Env = append(command.Env, "XDG_ACTIVATION_TOKEN="+token)
		}
		program := strconv.Quote(command.Args[0])
		if err := command.Start(); err != nil {
			slog.Error(fmt.Sprintf("Failed to open with %s: %v", program, err))
			continue
		}
		err := command.Wait()
		var exitErr *exec.ExitError
		switch {
		case err == nil:
			return true
		case errors.As(err, &exitErr):
			slog.Error(fmt.Sprintf("Command %s exited with status: %v", program, exitErr.ProcessState))
		default:
			slog.Error(fmt.Sprintf("Failed to get status from %s: %v", program, err))
		}
	}
	return false
}

// IsWithinClickDistance reports whether b is within the double-click distance of a.
func IsWithinClickDistance(a, b gpui.Point[gpui.Pixels]) bool {
	diff := a.Sub(b)
	return diff.X.Abs() <= DoubleClickDistance && diff.Y.Abs() <= DoubleClickDistance
}

// NewXkbContext creates an XKB context for keymaps supplied by Wayland or X11. Server keymaps
// are already resolved and need no local keyboard definitions.
func NewXkbContext() (*xkb.Context, error) {
	return validateXkbContext(xkb.NewContext(xkb.ContextNoDefaultIncludes))
}

func validateXkbContext(context *xkb.Context) (*xkb.Context, error) {
	if context.GetRawPtr() == 0 {
		return nil, errors.New("libxkbcommon failed to create an XKB context")
	}
	return context, nil
}

// GetXkbComposeState returns a compose state for LC_CTYPE's locale, or for "C".
func GetXkbComposeState(cx *xkb.Context) (*xkb.ComposeState, bool) {
	var locales []string
	if locale, ok := os.LookupEnv("LC_CTYPE"); ok {
		locales = append(locales, locale)
	}
	locales = append(locales, "C")
	for _, locale := range locales {
		if table, ok := xkb.NewComposeTableFromLocale(cx, locale, xkb.ComposeCompileNoFlags); ok {
			state := xkb.NewComposeState(table, xkb.ComposeStateNoFlags)
			table.Close()
			return state, true
		}
	}
	return nil, false
}

// PipeReadTimeout is how long ReadFdWithTimeout waits for each chunk of clipboard data.
const PipeReadTimeout = 4 * time.Second

// MaxPipeReadSize caps the bytes ReadFdWithTimeout accepts from one pipe.
const MaxPipeReadSize = 256 << 20

// PipeReadDeadlineFactor bounds a whole ReadFdWithTimeout call to this many per-chunk timeouts.
const PipeReadDeadlineFactor = 5

// ReadFdWithTimeout reads fd until end of file, failing when no data arrives for timeout, when the
// whole read takes longer than PipeReadDeadlineFactor times timeout, or when more than
// MaxPipeReadSize bytes arrive. It takes ownership of fd and closes it.
func ReadFdWithTimeout(fd int, timeout time.Duration) ([]byte, error) {
	defer unix.Close(fd)
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	total := timeout * PipeReadDeadlineFactor
	deadline := time.Now().Add(total)
	buffer := []byte{}
	var chunk [8192]byte
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("timed out reading pipe after %v", total)
		}
		pollFds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		ready, err := unix.Poll(pollFds, int(min(timeout, remaining).Milliseconds()))
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return nil, err
		}
		if ready == 0 {
			if remaining < timeout {
				return nil, fmt.Errorf("timed out reading pipe after %v", total)
			}
			return nil, fmt.Errorf("timed out waiting for data on pipe after %v", timeout)
		}
		n, err := unix.Read(fd, chunk[:])
		switch {
		case err == nil && n == 0:
			return buffer, nil
		case err == nil:
			if len(buffer)+n > MaxPipeReadSize {
				return nil, fmt.Errorf("pipe data exceeds %d bytes", MaxPipeReadSize)
			}
			buffer = append(buffer, chunk[:n]...)
		case errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR):
		default:
			return nil, err
		}
	}
}

// DefaultCursorIconName is the cursor theme name of the arrow cursor.
const DefaultCursorIconName = "left_ptr"

// CursorStyleToIconNames returns the cursor theme names to try for style, based on chromium's.
func CursorStyleToIconNames(style gpui.CursorStyle) []string {
	switch style {
	case gpui.CursorStyleArrow:
		return []string{DefaultCursorIconName}
	case gpui.CursorStyleIBeam:
		return []string{"text", "xterm"}
	case gpui.CursorStyleCrosshair:
		return []string{"crosshair", "cross"}
	case gpui.CursorStyleClosedHand:
		return []string{"closedhand", "grabbing", "hand2"}
	case gpui.CursorStyleOpenHand:
		return []string{"openhand", "grab", "hand1"}
	case gpui.CursorStylePointingHand:
		return []string{"pointer", "hand", "hand2"}
	case gpui.CursorStyleResizeLeft:
		return []string{"w-resize", "left_side"}
	case gpui.CursorStyleResizeRight:
		return []string{"e-resize", "right_side"}
	case gpui.CursorStyleResizeLeftRight:
		return []string{"ew-resize", "sb_h_double_arrow"}
	case gpui.CursorStyleResizeUp:
		return []string{"n-resize", "top_side"}
	case gpui.CursorStyleResizeDown:
		return []string{"s-resize", "bottom_side"}
	case gpui.CursorStyleResizeUpDown:
		return []string{"sb_v_double_arrow", "ns-resize"}
	case gpui.CursorStyleResizeUpLeftDownRight:
		return []string{"size_fdiag", "bd_double_arrow", "nwse-resize"}
	case gpui.CursorStyleResizeUpRightDownLeft:
		return []string{"size_bdiag", "nesw-resize", "fd_double_arrow"}
	case gpui.CursorStyleResizeColumn:
		return []string{"col-resize", "sb_h_double_arrow"}
	case gpui.CursorStyleResizeRow:
		return []string{"row-resize", "sb_v_double_arrow"}
	case gpui.CursorStyleIBeamCursorForVerticalLayout:
		return []string{"vertical-text"}
	case gpui.CursorStyleOperationNotAllowed:
		return []string{"not-allowed", "crossed_circle"}
	case gpui.CursorStyleDragLink:
		return []string{"alias"}
	case gpui.CursorStyleDragCopy:
		return []string{"copy"}
	case gpui.CursorStyleContextualMenu:
		return []string{"context-menu"}
	}
	return []string{DefaultCursorIconName}
}

// LogCursorIconWarning logs a cursor loading failure, mentioning XCURSOR_PATH when it is set.
func LogCursorIconWarning(message string) {
	if xcursorPath, ok := os.LookupEnv("XCURSOR_PATH"); ok {
		slog.Warn(fmt.Sprintf("%s\ncursor icon loading may be failing if XCURSOR_PATH environment variable is invalid. XCURSOR_PATH overrides the default icon search. Its current value is '%s'", message, xcursorPath))
	} else {
		slog.Warn(message)
	}
}

func guessASCII(keycode xkb.Keycode, shift bool) (rune, bool) {
	switch keycode {
	case 24:
		return 'q', true
	case 25:
		return 'w', true
	case 26:
		return 'e', true
	case 27:
		return 'r', true
	case 28:
		return 't', true
	case 29:
		return 'y', true
	case 30:
		return 'u', true
	case 31:
		return 'i', true
	case 32:
		return 'o', true
	case 33:
		return 'p', true
	case 34:
		return pick(shift, '{', '['), true
	case 35:
		return pick(shift, '}', ']'), true
	case 38:
		return 'a', true
	case 39:
		return 's', true
	case 40:
		return 'd', true
	case 41:
		return 'f', true
	case 42:
		return 'g', true
	case 43:
		return 'h', true
	case 44:
		return 'j', true
	case 45:
		return 'k', true
	case 46:
		return 'l', true
	case 47:
		return pick(shift, ':', ';'), true
	case 48:
		return pick(shift, '"', '\''), true
	case 49:
		return pick(shift, '~', '`'), true
	case 51:
		return pick(shift, '|', '\\'), true
	case 52:
		return 'z', true
	case 53:
		return 'x', true
	case 54:
		return 'c', true
	case 55:
		return 'v', true
	case 56:
		return 'b', true
	case 57:
		return 'n', true
	case 58:
		return 'm', true
	case 59:
		return pick(shift, '>', ','), true
	case 60:
		return pick(shift, '<', '.'), true
	case 61:
		return pick(shift, '?', '/'), true
	}
	return 0, false
}

func pick(shift bool, shifted, unshifted rune) rune {
	if shift {
		return shifted
	}
	return unshifted
}

// KeystrokeFromXkb builds the keystroke for keycode in state with modifiers.
func KeystrokeFromXkb(state *xkb.State, modifiers gpui.Modifiers, keycode xkb.Keycode) gpui.Keystroke {
	keyUTF32 := state.KeyGetUtf32(keycode)
	keyUTF8 := state.KeyGetUtf8(keycode)
	keySym := state.KeyGetOneSym(keycode)

	var key string
	switch keySym {
	case xkb.KEY_Return:
		key = "enter"
	case xkb.KEY_Prior:
		key = "pageup"
	case xkb.KEY_Next:
		key = "pagedown"
	case xkb.KEY_ISO_Left_Tab:
		key = "tab"
	case xkb.KEY_KP_Prior:
		key = "pageup"
	case xkb.KEY_KP_Next:
		key = "pagedown"
	case xkb.KEY_XF86Back:
		key = "back"
	case xkb.KEY_XF86Forward:
		key = "forward"
	case xkb.KEY_XF86Cut:
		key = "cut"
	case xkb.KEY_XF86Copy:
		key = "copy"
	case xkb.KEY_XF86Paste:
		key = "paste"
	case xkb.KEY_XF86New:
		key = "new"
	case xkb.KEY_XF86Open:
		key = "open"
	case xkb.KEY_XF86Save:
		key = "save"
	case xkb.KEY_comma:
		key = ","
	case xkb.KEY_period:
		key = "."
	case xkb.KEY_less:
		key = "<"
	case xkb.KEY_greater:
		key = ">"
	case xkb.KEY_slash:
		key = "/"
	case xkb.KEY_question:
		key = "?"
	case xkb.KEY_semicolon:
		key = ";"
	case xkb.KEY_colon:
		key = ":"
	case xkb.KEY_apostrophe:
		key = "'"
	case xkb.KEY_quotedbl:
		key = "\""
	case xkb.KEY_bracketleft:
		key = "["
	case xkb.KEY_braceleft:
		key = "{"
	case xkb.KEY_bracketright:
		key = "]"
	case xkb.KEY_braceright:
		key = "}"
	case xkb.KEY_backslash:
		key = "\\"
	case xkb.KEY_bar:
		key = "|"
	case xkb.KEY_grave:
		key = "`"
	case xkb.KEY_asciitilde:
		key = "~"
	case xkb.KEY_exclam:
		key = "!"
	case xkb.KEY_at:
		key = "@"
	case xkb.KEY_numbersign:
		key = "#"
	case xkb.KEY_dollar:
		key = "$"
	case xkb.KEY_percent:
		key = "%"
	case xkb.KEY_asciicircum:
		key = "^"
	case xkb.KEY_ampersand:
		key = "&"
	case xkb.KEY_asterisk:
		key = "*"
	case xkb.KEY_parenleft:
		key = "("
	case xkb.KEY_parenright:
		key = ")"
	case xkb.KEY_minus:
		key = "-"
	case xkb.KEY_underscore:
		key = "_"
	case xkb.KEY_equal:
		key = "="
	case xkb.KEY_plus:
		key = "+"
	case xkb.KEY_space:
		key = "space"
	case xkb.KEY_BackSpace:
		key = "backspace"
	case xkb.KEY_Tab:
		key = "tab"
	case xkb.KEY_Delete:
		key = "delete"
	case xkb.KEY_Escape:
		key = "escape"
	case xkb.KEY_Left:
		key = "left"
	case xkb.KEY_Right:
		key = "right"
	case xkb.KEY_Up:
		key = "up"
	case xkb.KEY_Down:
		key = "down"
	case xkb.KEY_Home:
		key = "home"
	case xkb.KEY_End:
		key = "end"
	case xkb.KEY_Insert:
		key = "insert"
	default:
		name := strings.ToLower(xkb.KeysymGetName(keySym))
		first, _ := utf8.DecodeRuneInString(keyUTF8)
		switch {
		case keySym.IsKeypadKey():
			key = strings.ReplaceAll(name, "kp_", "")
		case keyUTF8 != "" && len(keyUTF8) == 1 && first < utf8.RuneSelf:
			if first >= 0x21 && first <= 0x7e {
				key = strings.ToLower(keyUTF8)
			} else if keyUTF32 <= 0x1f && !(name != "" && name[0] >= '0' && name[0] <= '9') {
				key = strings.ToLower(string(rune(byte(keyUTF32) + 0x40)))
			} else {
				key = name
			}
		default:
			if keyEn, ok := guessASCII(keycode, modifiers.Shift); ok {
				key = string(keyEn)
			} else {
				key = name
			}
		}
	}

	if modifiers.Shift {
		if utf8.RuneCountInString(key) == 1 && strings.ToLower(key) == strings.ToUpper(key) {
			modifiers.Shift = false
		}
	}

	keyChar := gpui.None[string]()
	if keyUTF32 >= 32 && keyUTF32 != 127 && keyUTF8 != "" {
		keyChar = gpui.Some(keyUTF8)
	}

	return gpui.Keystroke{
		Modifiers: modifiers,
		Key:       key,
		KeyChar:   keyChar,
	}
}

// KeystrokeUnderlyingDeadKey returns which symbol the dead key keysym represents.
func KeystrokeUnderlyingDeadKey(keysym xkb.Keysym) (string, bool) {
	switch keysym {
	case xkb.KEY_dead_grave:
		return "`", true
	case xkb.KEY_dead_acute:
		return "´", true
	case xkb.KEY_dead_circumflex:
		return "^", true
	case xkb.KEY_dead_tilde:
		return "~", true
	case xkb.KEY_dead_macron:
		return "¯", true
	case xkb.KEY_dead_breve:
		return "˘", true
	case xkb.KEY_dead_abovedot:
		return "˙", true
	case xkb.KEY_dead_diaeresis:
		return "¨", true
	case xkb.KEY_dead_abovering:
		return "˚", true
	case xkb.KEY_dead_doubleacute:
		return "˝", true
	case xkb.KEY_dead_caron:
		return "ˇ", true
	case xkb.KEY_dead_cedilla:
		return "¸", true
	case xkb.KEY_dead_ogonek:
		return "˛", true
	case xkb.KEY_dead_iota:
		return "ͅ", true
	case xkb.KEY_dead_voiced_sound:
		return "゙", true
	case xkb.KEY_dead_semivoiced_sound:
		return "゚", true
	case xkb.KEY_dead_belowdot:
		return "̣̣", true
	case xkb.KEY_dead_hook:
		return "̡", true
	case xkb.KEY_dead_horn:
		return "̛", true
	case xkb.KEY_dead_stroke:
		return "̶̶", true
	case xkb.KEY_dead_abovecomma:
		return "̓̓", true
	case xkb.KEY_dead_abovereversedcomma:
		return "ʽ", true
	case xkb.KEY_dead_doublegrave:
		return "̏", true
	case xkb.KEY_dead_belowring:
		return "˳", true
	case xkb.KEY_dead_belowmacron:
		return "̱", true
	case xkb.KEY_dead_belowcircumflex:
		return "ꞈ", true
	case xkb.KEY_dead_belowtilde:
		return "̰", true
	case xkb.KEY_dead_belowbreve:
		return "̮", true
	case xkb.KEY_dead_belowdiaeresis:
		return "̤", true
	case xkb.KEY_dead_invertedbreve:
		return "̯", true
	case xkb.KEY_dead_belowcomma:
		return "̦", true
	case xkb.KEY_dead_small_schwa:
		return "ə", true
	case xkb.KEY_dead_capital_schwa:
		return "Ə", true
	case xkb.KEY_dead_currency, xkb.KEY_dead_lowline, xkb.KEY_dead_aboveverticalline, xkb.KEY_dead_belowverticalline, xkb.KEY_dead_longsolidusoverlay, xkb.KEY_dead_a, xkb.KEY_dead_A, xkb.KEY_dead_e, xkb.KEY_dead_E, xkb.KEY_dead_i, xkb.KEY_dead_I, xkb.KEY_dead_o, xkb.KEY_dead_O, xkb.KEY_dead_u, xkb.KEY_dead_U, xkb.KEY_dead_greek:
		return "", false
	}
	return "", false
}

// ModifiersFromXkb returns the effective modifiers of keymapState.
func ModifiersFromXkb(keymapState *xkb.State) gpui.Modifiers {
	shift := keymapState.ModNameIsActive(xkb.ModNameShift, xkb.StateModsEffective)
	alt := keymapState.ModNameIsActive(xkb.ModNameAlt, xkb.StateModsEffective)
	control := keymapState.ModNameIsActive(xkb.ModNameCtrl, xkb.StateModsEffective)
	platform := keymapState.ModNameIsActive(xkb.ModNameLogo, xkb.StateModsEffective)
	return gpui.Modifiers{
		Shift:    shift,
		Alt:      alt,
		Control:  control,
		Platform: platform,
		Function: false,
	}
}

// CapslockFromXkb returns the effective capslock state of keymapState.
func CapslockFromXkb(keymapState *xkb.State) gpui.Capslock {
	on := keymapState.ModNameIsActive(xkb.ModNameCaps, xkb.StateModsEffective)
	return gpui.Capslock{On: on}
}

// CompositorGPUHintFromDevT resolves a Linux dev_t to the PCI vendor and device ids of the GPU
// through sysfs, for render/wgpu's CompositorGPUHint.
func CompositorGPUHintFromDevT(dev uint64) (vendorID, deviceID uint32, ok bool) {
	devMajor := func(dev uint64) uint32 {
		return uint32((dev>>8)&0xfff) | uint32((dev>>32)&^uint64(0xfff))
	}
	devMinor := func(dev uint64) uint32 {
		return uint32(dev&0xff) | uint32((dev>>12)&^uint64(0xff))
	}
	readSysfsHexID := func(path string) (uint32, bool) {
		content, err := os.ReadFile(path)
		if err != nil {
			return 0, false
		}
		trimmed := strings.TrimPrefix(strings.TrimSpace(string(content)), "0x")
		v, err := strconv.ParseUint(trimmed, 16, 32)
		if err != nil {
			return 0, false
		}
		return uint32(v), true
	}

	major := devMajor(dev)
	minor := devMinor(dev)
	vendorPath := fmt.Sprintf("/sys/dev/char/%d:%d/device/vendor", major, minor)
	devicePath := fmt.Sprintf("/sys/dev/char/%d:%d/device/device", major, minor)
	vendorID, ok = readSysfsHexID(vendorPath)
	if !ok {
		return 0, 0, false
	}
	deviceID, ok = readSysfsHexID(devicePath)
	if !ok {
		return 0, 0, false
	}
	slog.Info(fmt.Sprintf("Compositor GPU hint: vendor=%#06x, device=%#06x (from dev %d:%d)", vendorID, deviceID, major, minor))
	return vendorID, deviceID, true
}

const (
	inhibitInterface = "org.freedesktop.portal.Inhibit"
	inhibitSuspend   = uint32(4)
)

// InhibitIdleSleep asks the Inhibit portal to keep the system from suspending for reason; it
// blocks until the portal answers. Releasing the guard closes the request on executor.
func InhibitIdleSleep(reason string, executor gpui.BackgroundExecutor, bus func() (*godbus.Conn, error)) (*gpui.ActivityGuard, error) {
	conn, err := bus()
	if err == nil {
		_, err = conn.Object(portalService, portalPath).GetProperty(inhibitInterface + ".version")
	}
	if err != nil {
		return nil, fmt.Errorf("Idle sleep prevention portal is unavailable: %w", err)
	}
	options := map[string]godbus.Variant{"reason": godbus.MakeVariant(reason)}
	request, err := portalCall(conn, inhibitInterface+".Inhibit", options, "", inhibitSuspend)
	if err != nil {
		if isPortalResponse(err) {
			return nil, fmt.Errorf("Idle sleep prevention request was rejected: %w", err)
		}
		return nil, fmt.Errorf("Failed to request idle sleep prevention: %w", err)
	}
	return releaseOnDrop(request, executor), nil
}

func releaseOnDrop(request portalRequest, executor gpui.BackgroundExecutor) *gpui.ActivityGuard {
	return gpui.NewActivityGuard(func() {
		spawnDetached(executor, func() {
			if err := request.close(); err != nil {
				slog.Error(fmt.Sprintf("Failed to release idle sleep prevention: %v", err))
			}
		})
	})
}

// AwaitIdleSleepPrevention waits up to ten seconds for the guard InhibitIdleSleep sends on
// guardRx. A guard sent after the timeout fails to send and stays with its sender to release.
func AwaitIdleSleepPrevention(s *scheduler.Scope, guardRx *scheduler.OneshotReceiver[scheduler.Result[*gpui.ActivityGuard]], executor gpui.BackgroundExecutor) (*gpui.ActivityGuard, error) {
	var guard *gpui.ActivityGuard
	var err error
	scheduler.Select(s,
		scheduler.Case(guardRx, func(r scheduler.Result[scheduler.Result[*gpui.ActivityGuard]]) {
			if r.Err != nil {
				err = errors.New("Idle sleep prevention request was abandoned")
				return
			}
			guard, err = r.Value.Value, r.Value.Err
		}),
		scheduler.Case(executor.Timer(10*time.Second), func(scheduler.Result[struct{}]) {
			err = errors.New("Idle sleep prevention acquisition timed out after 10 seconds")
		}),
	)
	return guard, err
}
