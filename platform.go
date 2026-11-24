package gpui

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"image"
	"iter"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// WindowVisibility reports whether the platform is presenting a window's frames.
type WindowVisibility uint8

const (
	// WindowVisibilityVisible means at least part of the window is being presented.
	WindowVisibilityVisible WindowVisibility = iota
	// WindowVisibilityHidden means no part of the window is being presented.
	WindowVisibilityHidden
)

// IsVisible reports whether frames drawn for the window will be shown.
func (v WindowVisibility) IsVisible() bool {
	return v == WindowVisibilityVisible
}

// DisplayEvent is a change to the connected displays.
type DisplayEvent interface {
	isDisplayEvent()
}

// DisplayEventAdded reports that a display was connected.
type DisplayEventAdded struct {
	// ID is the connected display.
	ID DisplayID
}

// DisplayEventRemoved reports that a display was disconnected.
type DisplayEventRemoved struct {
	// ID is the disconnected display.
	ID DisplayID
}

// DisplayEventChanged reports that a connected display's properties changed.
type DisplayEventChanged struct {
	// ID is the display that changed.
	ID DisplayID
	// Changes is which of its properties changed.
	Changes DisplayChanges
}

func (DisplayEventAdded) isDisplayEvent()   {}
func (DisplayEventRemoved) isDisplayEvent() {}
func (DisplayEventChanged) isDisplayEvent() {}

// DisplayChanges is the set of display properties a DisplayEventChanged reports.
type DisplayChanges uint8

const (
	// DisplayChangesBounds means PlatformDisplay.Bounds changed.
	DisplayChangesBounds DisplayChanges = 1 << 0
	// DisplayChangesRefreshInterval means PlatformDisplay.RefreshInterval changed.
	DisplayChangesRefreshInterval DisplayChanges = 1 << 1
)

// Contains reports whether every flag of other is set in c.
func (c DisplayChanges) Contains(other DisplayChanges) bool {
	return c&other == other
}

// RefreshIntervalFromHz converts a refresh rate in hertz to the time between refreshes,
// rejecting the zero, negative and non-finite rates platforms use to mean "unknown".
func RefreshIntervalFromHz(hertz float64) (time.Duration, bool) {
	if !(!math.IsInf(hertz, 0) && !math.IsNaN(hertz) && hertz > 0.0) {
		return 0, false
	}
	nanos := math.Round(1.0 / hertz * 1e9)
	if nanos >= math.MaxInt64 {
		return time.Duration(math.MaxInt64), true
	}
	return time.Duration(nanos), true
}

// ActivationPolicy controls whether the application participates in the system's foreground UI.
type ActivationPolicy uint8

const (
	// ActivationPolicyRegular participates in foreground application UI, such as the Dock and menu bar on macOS.
	ActivationPolicyRegular ActivationPolicy = iota
	// ActivationPolicyAccessory runs without foreground application UI while retaining the ability to open windows.
	ActivationPolicyAccessory
)

// ActivityGuard keeps an operating system activity, such as an idle sleep inhibitor, alive until released.
type ActivityGuard struct {
	release gpuiutil.Deferred
}

// NewActivityGuard returns a guard that runs release when it is released.
func NewActivityGuard(release func()) *ActivityGuard {
	return &ActivityGuard{release: gpuiutil.Defer(release)}
}

// ActivityGuardNoop returns a guard for platforms without a corresponding activity.
func ActivityGuardNoop() *ActivityGuard {
	return NewActivityGuard(func() {})
}

// Release ends the activity; it is upstream's drop and runs at most once.
func (g *ActivityGuard) Release() {
	g.release.Run()
}

// WindowingRequest is a display mode for App.RequestWindowing to switch to.
type WindowingRequest interface {
	isWindowingRequest()
}

// WindowingRequestHeadless requests no display server: windows lay out and handle input but draw nothing.
type WindowingRequestHeadless struct{}

// WindowingRequestWindowed requests a connection to the display server that the environment names.
type WindowingRequestWindowed struct {
	// Environment is the graphical session to connect to.
	Environment GraphicalEnvironment
}

func (WindowingRequestHeadless) isWindowingRequest() {}
func (WindowingRequestWindowed) isWindowingRequest() {}

// Credentials are the username and password returned by Platform.ReadCredentials (D-TUPLE).
type Credentials struct {
	// Username is the stored user name.
	Username string
	// Password is the stored secret.
	Password []byte
}

// Platform is the interface every platform back end implements; embed PlatformDefaults for
// upstream's provided methods.
type Platform interface {
	PlatformInfo() PlatformInfo
	BackgroundExecutor() BackgroundExecutor
	ForegroundExecutor() ForegroundExecutor
	TextSystem() PlatformTextSystem
	Run(onFinishLaunching func())
	Quit()
	SetInitialWindowing(request WindowingRequest)
	RequestWindowing(request WindowingRequest) *Task[struct{}]
	Restart(binaryPath Option[string], arguments []string)
	Activate(ignoringOtherApps bool)
	SetActivationPolicy(policy ActivationPolicy)
	Hide()
	HideOtherApps()
	UnhideOtherApps()
	Displays() []PlatformDisplay
	PrimaryDisplay() PlatformDisplay
	OnDisplaysChanged(callback func())
	ActiveWindow() (AnyWindowHandle, bool)
	WindowStack() ([]AnyWindowHandle, bool)
	IsScreenCaptureSupported() bool
	ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]]
	OpenWindow(handle AnyWindowHandle, options WindowParams) (PlatformWindow, error)
	WindowAppearance() WindowAppearance
	SetWindowAppearance(appearance Option[WindowAppearance])
	ButtonLayout() (WindowButtonLayout, bool)
	TextScaleFactor() float32
	OnTextScaleFactorChange(callback func())
	OpenURL(url string)
	OnOpenURLs(callback func(urls []string))
	RegisterURLScheme(url string) *Task[struct{}]
	PromptForPaths(options PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]]
	PromptForNewPath(directory string, suggestedName Option[string]) *scheduler.OneshotReceiver[scheduler.Result[Option[string]]]
	CanSelectMixedFilesAndDirs() bool
	RevealPath(path string)
	OpenWithSystem(path string)
	OnQuit(callback func() bool)
	OnReopen(callback func())
	OnSystemSleep(callback func())
	OnSystemWake(callback func())
	OnAppLifecycle(callback func(phase AppLifecyclePhase))
	OnMemoryWarning(callback func())
	Gestures() PlatformGestures
	SetMenus(menus []Menu, keymap *Keymap)
	GetMenus() ([]OwnedMenu, bool)
	SetDockMenu(menu []MenuItem, keymap *Keymap)
	PerformDockMenuAction(action int)
	AddRecentDocument(path string)
	UpdateJumpList(menus []MenuItem, entries [][]string) *Task[[][]string]
	OnAppMenuAction(callback func(action Action))
	OnWillOpenAppMenu(callback func())
	OnValidateAppMenuCommand(callback func(action Action) bool)
	DebugState() jsontext.Value
	ThermalState() ThermalState
	OnThermalStateChange(callback func())
	PreventIdleSleep(reason string) *Task[*ActivityGuard]
	PowerMode() PowerMode
	OnPowerModeChange(callback func())
	SetAppIdentity(identifier string, name string)
	ShowSystemNotification(notification SystemNotification)
	DismissSystemNotification(tag string) *Task[struct{}]
	OnSystemNotificationResponse(callback func(response SystemNotificationResponse))
	CompositorName() string
	GraphicalEnvironment() (GraphicalEnvironment, bool)
	AppPath() (string, error)
	PathForAuxiliaryExecutable(name string) (string, error)
	SetCursorStyle(style CursorStyle)
	HideCursorUntilMouseMoves()
	IsCursorVisible() bool
	ShouldAutoHideScrollbars() bool
	ReadFromClipboard() (ClipboardItem, bool)
	WriteToClipboard(item ClipboardItem)
	ReadFromClipboardAsync() *Task[Option[ClipboardItem]]
	ReadFromPrimary() (ClipboardItem, bool)
	WriteToPrimary(item ClipboardItem)
	ReadFromFindPasteboard() (ClipboardItem, bool)
	WriteToFindPasteboard(item ClipboardItem)
	WriteCredentials(url string, username string, password []byte) *Task[struct{}]
	WriteCredentialsIfNotExists(url string, username string, password []byte) *Task[bool]
	ReadCredentials(url string) *Task[Option[Credentials]]
	DeleteCredentials(url string) *Task[struct{}]
	KeyboardLayout() PlatformKeyboardLayout
	KeyboardMapper() PlatformKeyboardMapper
	OnKeyboardLayoutChange(callback func())
	Close()
}

const unboundDefaultsMessage = "gpui: %s used before %s bound it to its implementor"

// PlatformDefaults carries the default bodies of Platform's provided methods (DESIGN 5.4).
// Embed it in a Platform implementation and bind it with BindPlatformDefaults in the constructor.
type PlatformDefaults struct {
	self Platform
}

// BindPlatformDefaults makes the provided methods that call other Platform methods dispatch to self.
func BindPlatformDefaults(d *PlatformDefaults, self Platform) {
	d.self = self
}

func (d *PlatformDefaults) bound() Platform {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "PlatformDefaults", "BindPlatformDefaults"))
	}
	return d.self
}

// SetInitialWindowing sets the windowing mode the platform starts in; the default ignores it.
func (d *PlatformDefaults) SetInitialWindowing(request WindowingRequest) {}

// RequestWindowing switches a capable platform between headless and windowed modes.
func (d *PlatformDefaults) RequestWindowing(request WindowingRequest) *Task[struct{}] {
	return scheduler.ReadyTaskErr[struct{}](errors.New("this platform cannot switch between headless and windowed modes"))
}

// SetActivationPolicy sets the activation policy; it has no effect outside macOS.
func (d *PlatformDefaults) SetActivationPolicy(policy ActivationPolicy) {}

// WindowStack returns the platform's window stack, if it reports one.
func (d *PlatformDefaults) WindowStack() ([]AnyWindowHandle, bool) {
	return nil, false
}

// IsScreenCaptureSupported reports whether screen capture is available.
func (d *PlatformDefaults) IsScreenCaptureSupported() bool {
	return false
}

// ScreenCaptureSources returns a receiver for the screens that can be captured.
func (d *PlatformDefaults) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	sourcesTx, sourcesRx := scheduler.Oneshot[scheduler.Result[[]ScreenCaptureSource]]()
	_ = sourcesTx.Send(scheduler.Result[[]ScreenCaptureSource]{Err: errors.New("gpui was compiled without the screen-capture feature")})
	return sourcesRx
}

// SetWindowAppearance overrides the appearance of the app's windows; the default ignores it.
func (d *PlatformDefaults) SetWindowAppearance(appearance Option[WindowAppearance]) {}

// ButtonLayout returns the window button layout configuration when supported.
func (d *PlatformDefaults) ButtonLayout() (WindowButtonLayout, bool) {
	return WindowButtonLayout{}, false
}

// TextScaleFactor returns the factor by which text is scaled on top of the window's scale factor.
func (d *PlatformDefaults) TextScaleFactor() float32 {
	return 1.0
}

// OnTextScaleFactorChange registers a callback for text scale factor changes; the default ignores it.
func (d *PlatformDefaults) OnTextScaleFactorChange(callback func()) {}

// OnAppLifecycle registers a callback for application lifecycle changes; desktop platforms never invoke it.
func (d *PlatformDefaults) OnAppLifecycle(callback func(phase AppLifecyclePhase)) {}

// OnMemoryWarning registers a callback for memory pressure; desktop platforms never invoke it.
func (d *PlatformDefaults) OnMemoryWarning(callback func()) {}

// Gestures returns the platform's gesture recognition services, or nil.
func (d *PlatformDefaults) Gestures() PlatformGestures {
	return nil
}

// GetMenus returns the application menus, if the platform keeps them.
func (d *PlatformDefaults) GetMenus() ([]OwnedMenu, bool) {
	return nil, false
}

// PerformDockMenuAction performs a dock menu action; the default ignores it.
func (d *PlatformDefaults) PerformDockMenuAction(action int) {}

// AddRecentDocument adds a path to the recent documents list; the default ignores it.
func (d *PlatformDefaults) AddRecentDocument(path string) {}

// UpdateJumpList updates the jump list and returns the entries the user removed.
func (d *PlatformDefaults) UpdateJumpList(menus []MenuItem, entries [][]string) *Task[[][]string] {
	return scheduler.ReadyTask([][]string{})
}

// DebugState returns platform-specific diagnostic state.
func (d *PlatformDefaults) DebugState() jsontext.Value {
	return jsontext.Value("null")
}

// PowerMode returns the operating system's power mode.
func (d *PlatformDefaults) PowerMode() PowerMode {
	return PowerModeUnknown
}

// OnPowerModeChange registers a callback for power mode changes; the default ignores it.
func (d *PlatformDefaults) OnPowerModeChange(callback func()) {}

// SetAppIdentity sets the application's process-wide identity and user-visible name.
func (d *PlatformDefaults) SetAppIdentity(identifier string, name string) {}

// ShowSystemNotification posts a notification to the operating system's notification center.
func (d *PlatformDefaults) ShowSystemNotification(notification SystemNotification) {}

// DismissSystemNotification removes the delivered or pending notification with this tag.
func (d *PlatformDefaults) DismissSystemNotification(tag string) *Task[struct{}] {
	return scheduler.ReadyTask(struct{}{})
}

// OnSystemNotificationResponse registers the callback for notification activations.
func (d *PlatformDefaults) OnSystemNotificationResponse(callback func(response SystemNotificationResponse)) {
}

// CompositorName returns the name of the compositor the platform is connected to.
func (d *PlatformDefaults) CompositorName() string {
	return ""
}

// GraphicalEnvironment returns the environment of the display server the platform is connected to.
func (d *PlatformDefaults) GraphicalEnvironment() (GraphicalEnvironment, bool) {
	return GraphicalEnvironment{}, false
}

// ReadFromClipboardAsync reads the clipboard, resolving once its contents are available.
func (d *PlatformDefaults) ReadFromClipboardAsync() *Task[Option[ClipboardItem]] {
	return scheduler.ReadyTask(gpuiutil.OptionFrom(d.bound().ReadFromClipboard()))
}

// ReadFromPrimary reads the primary selection; only Linux platforms provide it.
func (d *PlatformDefaults) ReadFromPrimary() (ClipboardItem, bool) {
	return ClipboardItem{}, false
}

// WriteToPrimary writes the primary selection; only Linux platforms provide it.
func (d *PlatformDefaults) WriteToPrimary(item ClipboardItem) {}

// ReadFromFindPasteboard reads the find pasteboard; only macOS provides it.
func (d *PlatformDefaults) ReadFromFindPasteboard() (ClipboardItem, bool) {
	return ClipboardItem{}, false
}

// WriteToFindPasteboard writes the find pasteboard; only macOS provides it.
func (d *PlatformDefaults) WriteToFindPasteboard(item ClipboardItem) {}

// WriteCredentialsIfNotExists atomically writes credentials unless an item with this URL and username exists.
func (d *PlatformDefaults) WriteCredentialsIfNotExists(url string, username string, password []byte) *Task[bool] {
	return scheduler.ReadyTaskErr[bool](errors.New("this platform does not support atomic credential writes"))
}

// Close releases the platform; it is upstream's drop of the platform value.
func (d *PlatformDefaults) Close() {}

// PlatformDisplay is a handle to a platform's display, e.g. a monitor or laptop screen; embed
// PlatformDisplayDefaults for its provided methods.
type PlatformDisplay interface {
	ID() DisplayID
	UUID() (UUID, error)
	Bounds() Bounds[Pixels]
	RefreshInterval() (time.Duration, bool)
	VisibleBounds() Bounds[Pixels]
	DefaultBounds() Bounds[Pixels]
}

// PlatformDisplayDefaults carries the default bodies of PlatformDisplay's provided methods.
// Bind it with BindPlatformDisplayDefaults in the implementation's constructor.
type PlatformDisplayDefaults struct {
	self PlatformDisplay
}

// BindPlatformDisplayDefaults makes the provided methods dispatch to self.
func BindPlatformDisplayDefaults(d *PlatformDisplayDefaults, self PlatformDisplay) {
	d.self = self
}

func (d *PlatformDisplayDefaults) bound() PlatformDisplay {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "PlatformDisplayDefaults", "BindPlatformDisplayDefaults"))
	}
	return d.self
}

// VisibleBounds returns the display's usable area; the default is the full display bounds.
func (d *PlatformDisplayDefaults) VisibleBounds() Bounds[Pixels] {
	return d.bound().Bounds()
}

// DefaultBounds returns the default bounds for a window placed on this display.
func (d *PlatformDisplayDefaults) DefaultBounds() Bounds[Pixels] {
	bounds := d.bound().Bounds()
	center := bounds.Center()
	clippedWindowSize := DefaultWindowSize.Min(bounds.Size)

	offset := clippedWindowSize.Div(2.0)
	origin := NewPoint(center.X-offset.Width, center.Y-offset.Height)
	return NewBounds(origin, clippedWindowSize)
}

// UUID is a universally unique identifier (the uuid crate's Uuid).
type UUID [16]byte

// NewUUIDV4 returns a random version 4 UUID.
func NewUUIDV4() UUID {
	var u UUID
	_, _ = rand.Read(u[:])
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	return u
}

// String formats the UUID in hyphenated lower-case form.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])
	return string(buf[:])
}

// SystemNotification is a notification posted to the operating system's notification center.
type SystemNotification struct {
	// Tag is the stable identity of the notification.
	Tag SharedString
	// Title is the notification's headline.
	Title SharedString
	// Body is additional text displayed below the title.
	Body SharedString
	// Actions are the buttons offered on the notification.
	Actions []SystemNotificationAction
}

// SystemNotificationAction is a button offered on a SystemNotification.
type SystemNotificationAction struct {
	// ID identifies the action in SystemNotificationResponse.ActionID.
	ID SharedString
	// Label is the button's user-visible label.
	Label SharedString
}

// SystemNotificationResponse is the user's activation of a SystemNotification.
type SystemNotificationResponse struct {
	// Tag is the SystemNotification.Tag of the activated notification.
	Tag SharedString
	// ActionID is the pressed button's id, or None when the body was activated.
	ActionID Option[SharedString]
}

// PowerMode is the operating system's preference for energy use versus performance.
type PowerMode uint8

const (
	// PowerModeUnknown means the platform cannot determine the mode.
	PowerModeUnknown PowerMode = iota
	// PowerModeLowPower means the system prioritizes reducing energy use.
	PowerModeLowPower
	// PowerModeBalanced means the system balances energy use and performance.
	PowerModeBalanced
	// PowerModeHighPower means the system prioritizes performance over energy use.
	PowerModeHighPower
)

// ThermalState is the thermal state of the system.
type ThermalState uint8

const (
	// ThermalStateNominal means the system has no thermal constraints.
	ThermalStateNominal ThermalState = iota
	// ThermalStateFair means the system is slightly constrained.
	ThermalStateFair
	// ThermalStateSerious means the system is moderately constrained.
	ThermalStateSerious
	// ThermalStateCritical means the system is critically constrained.
	ThermalStateCritical
)

// SourceMetadata is the metadata of a ScreenCaptureSource.
type SourceMetadata struct {
	// ID is the opaque identifier of this screen.
	ID uint64
	// Label is the human-readable label of this source.
	Label Option[SharedString]
	// IsMain reports whether this source is the main display.
	IsMain Option[bool]
	// Resolution is the video resolution of this source.
	Resolution Size[DevicePixels]
}

// ScreenCaptureSource is a source of on-screen video content that can be captured.
type ScreenCaptureSource interface {
	Metadata() (SourceMetadata, error)
	Stream(foregroundExecutor ForegroundExecutor, frameCallback func(frame ScreenCaptureFrame)) *scheduler.OneshotReceiver[scheduler.Result[ScreenCaptureStream]]
}

// ScreenCaptureStream is a video stream captured from a screen; Stop is upstream's drop.
type ScreenCaptureStream interface {
	Metadata() (SourceMetadata, error)
	Stop()
}

// ScreenCaptureFrame is a frame of video captured from a screen.
type ScreenCaptureFrame struct {
	// Frame is the platform's frame value.
	Frame PlatformScreenCaptureFrame
}

// DisplayID is an opaque identifier for a hardware display.
type DisplayID uint64

// NewDisplayID creates a DisplayID from a raw platform display identifier.
func NewDisplayID(id uint64) DisplayID {
	return DisplayID(id)
}

// String formats the id as upstream's Debug does.
func (id DisplayID) String() string {
	return fmt.Sprintf("DisplayId(%d)", uint64(id))
}

// ResizeEdge is the part of the window to resize.
type ResizeEdge uint8

const (
	// ResizeEdgeTop is the top edge.
	ResizeEdgeTop ResizeEdge = iota
	// ResizeEdgeTopRight is the top right corner.
	ResizeEdgeTopRight
	// ResizeEdgeRight is the right edge.
	ResizeEdgeRight
	// ResizeEdgeBottomRight is the bottom right corner.
	ResizeEdgeBottomRight
	// ResizeEdgeBottom is the bottom edge.
	ResizeEdgeBottom
	// ResizeEdgeBottomLeft is the bottom left corner.
	ResizeEdgeBottomLeft
	// ResizeEdgeLeft is the left edge.
	ResizeEdgeLeft
	// ResizeEdgeTopLeft is the top left corner.
	ResizeEdgeTopLeft
)

// WindowDecorations describes the requested decorations of a window.
type WindowDecorations uint8

const (
	// WindowDecorationsServer requests server side decorations.
	WindowDecorationsServer WindowDecorations = iota
	// WindowDecorationsClient requests client side decorations.
	WindowDecorationsClient
)

// Decorations describes how a window is currently configured; the zero value is Server.
type Decorations struct {
	client bool
	tiling Tiling
}

// DecorationsServer returns the server side decorations configuration.
func DecorationsServer() Decorations {
	return Decorations{}
}

// DecorationsClient returns the client side decorations configuration with the given tiling.
func DecorationsClient(tiling Tiling) Decorations {
	return Decorations{client: true, tiling: tiling}
}

// Client returns the tiling state when the window uses client side decorations.
func (d Decorations) Client() (Tiling, bool) {
	return d.tiling, d.client
}

// IsServer reports whether the window uses server side decorations.
func (d Decorations) IsServer() bool {
	return !d.client
}

// WindowControls is the set of window controls a platform supports.
type WindowControls struct {
	// Fullscreen reports whether the platform supports fullscreen.
	Fullscreen bool
	// Maximize reports whether the platform supports maximize.
	Maximize bool
	// Minimize reports whether the platform supports minimize.
	Minimize bool
	// WindowMenu reports whether the platform supports a window menu.
	WindowMenu bool
}

// DefaultWindowControls returns upstream's default: every control is supported.
func DefaultWindowControls() WindowControls {
	return WindowControls{
		Fullscreen: true,
		Maximize:   true,
		Minimize:   true,
		WindowMenu: true,
	}
}

// WindowButton is a window control button used in WindowButtonLayout.
type WindowButton uint8

const (
	// WindowButtonMinimize is the minimize button.
	WindowButtonMinimize WindowButton = iota
	// WindowButtonMaximize is the maximize button.
	WindowButtonMaximize
	// WindowButtonClose is the close button.
	WindowButtonClose
)

// ID returns a stable element id for rendering this button.
func (b WindowButton) ID() string {
	switch b {
	case WindowButtonMinimize:
		return "minimize"
	case WindowButtonMaximize:
		return "maximize"
	case WindowButtonClose:
		return "close"
	}
	panic(fmt.Sprintf("gpui: invalid WindowButton %d", uint8(b)))
}

// MaxButtonsPerSide is the maximum number of WindowButtons per side in the titlebar.
const MaxButtonsPerSide = 3

// WindowButtonLayout describes which WindowButtons appear on each side of the titlebar.
type WindowButtonLayout struct {
	// Left is the buttons on the left side of the titlebar.
	Left [MaxButtonsPerSide]Option[WindowButton]
	// Right is the buttons on the right side of the titlebar.
	Right [MaxButtonsPerSide]Option[WindowButton]
}

func (b WindowButton) index() int {
	switch b {
	case WindowButtonMinimize:
		return 0
	case WindowButtonMaximize:
		return 1
	case WindowButtonClose:
		return 2
	}
	panic(fmt.Sprintf("gpui: invalid WindowButton %d", uint8(b)))
}

// WindowButtonLayoutLinuxDefault returns Zed's built-in fallback button layout for Linux titlebars.
func WindowButtonLayoutLinuxDefault() WindowButtonLayout {
	return WindowButtonLayout{
		Right: [MaxButtonsPerSide]Option[WindowButton]{
			Some(WindowButtonMinimize),
			Some(WindowButtonMaximize),
			Some(WindowButtonClose),
		},
	}
}

// WindowButtonLayoutParse parses a GNOME-style button-layout string (e.g. "close,minimize:maximize").
func WindowButtonLayoutParse(layoutString string) (WindowButtonLayout, error) {
	parseSide := func(s string, seenButtons *[MaxButtonsPerSide]bool, unrecognized *[]string) [MaxButtonsPerSide]Option[WindowButton] {
		var result [MaxButtonsPerSide]Option[WindowButton]
		i := 0
		for name := range strings.SplitSeq(s, ",") {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			var button Option[WindowButton]
			switch trimmed {
			case "minimize":
				button = Some(WindowButtonMinimize)
			case "maximize":
				button = Some(WindowButtonMaximize)
			case "close":
				button = Some(WindowButtonClose)
			default:
				*unrecognized = append(*unrecognized, trimmed)
			}
			if button, ok := button.Get(); ok {
				if seenButtons[button.index()] {
					continue
				}
				if i < len(result) {
					result[i] = Some(button)
					seenButtons[button.index()] = true
					i++
				}
			}
		}
		return result
	}

	leftStr, rightStr, found := strings.Cut(layoutString, ":")
	if !found {
		leftStr, rightStr = "", layoutString
	}
	var unrecognized []string
	var seenButtons [MaxButtonsPerSide]bool
	layout := WindowButtonLayout{
		Left:  parseSide(leftStr, &seenButtons, &unrecognized),
		Right: parseSide(rightStr, &seenButtons, &unrecognized),
	}

	allNone := func(side [MaxButtonsPerSide]Option[WindowButton]) bool {
		for _, button := range side {
			if button.IsSome() {
				return false
			}
		}
		return true
	}
	if len(unrecognized) != 0 && allNone(layout.Left) && allNone(layout.Right) {
		return WindowButtonLayout{}, fmt.Errorf("button layout string %q contains no valid buttons (unrecognized: %s)", layoutString, strings.Join(unrecognized, ", "))
	}

	return layout, nil
}

func (l WindowButtonLayout) format() string {
	formatSide := func(buttons [MaxButtonsPerSide]Option[WindowButton]) string {
		var names []string
		for _, button := range buttons {
			if button, ok := button.Get(); ok {
				switch button {
				case WindowButtonMinimize:
					names = append(names, "minimize")
				case WindowButtonMaximize:
					names = append(names, "maximize")
				case WindowButtonClose:
					names = append(names, "close")
				}
			}
		}
		return strings.Join(names, ",")
	}

	return fmt.Sprintf("%s:%s", formatSide(l.Left), formatSide(l.Right))
}

// Tiling describes which sides of the window are currently tiled.
type Tiling struct {
	// Top reports whether the top edge is tiled.
	Top bool
	// Left reports whether the left edge is tiled.
	Left bool
	// Right reports whether the right edge is tiled.
	Right bool
	// Bottom reports whether the bottom edge is tiled.
	Bottom bool
}

// TilingTiled returns a Tiling with all sides tiled.
func TilingTiled() Tiling {
	return Tiling{Top: true, Left: true, Right: true, Bottom: true}
}

// IsTiled reports whether any edge is tiled.
func (t Tiling) IsTiled() bool {
	return t.Top || t.Left || t.Right || t.Bottom
}

// A11yCallbacks are the callbacks of the accessibility adapter.
type A11yCallbacks struct {
	// Activation is called when the adapter is activated (a screen reader connects).
	Activation func() (a11y.TreeUpdate, bool)
	// Action is called when an action is requested by the screen reader.
	Action func(request a11y.ActionRequest)
	// Deactivation is called when the adapter is deactivated (the screen reader disconnects).
	Deactivation func()
}

// FrameRequestSource is the source of a platform frame request's timestamp.
type FrameRequestSource uint8

const (
	// FrameRequestSourceNativeCallback is an OS callback or compositor-paced frame request.
	FrameRequestSourceNativeCallback FrameRequestSource = iota
	// FrameRequestSourceLocalSchedule is a refresh timer, retry, queued wakeup or fallback sleep paced by the app.
	FrameRequestSourceLocalSchedule
)

// RequestFrameOptions are the options of a platform frame request.
type RequestFrameOptions struct {
	// RequirePresentation reports whether a presentation is required.
	RequirePresentation bool
	// ForceRender forces a refresh of all rendering state.
	ForceRender bool
	// SignalAt is when the platform first requested this frame, before main-thread dispatch.
	SignalAt Option[scheduler.Instant]
	// SignalSource distinguishes native callbacks from local scheduling requests.
	SignalSource FrameRequestSource
}

const platformFrameSignalEmpty = math.MaxUint64

// PlatformFrameSignal preserves the first platform frame request time across coalesced notifications.
// Create it with NewPlatformFrameSignal; it must not be copied.
type PlatformFrameSignal struct {
	origin      scheduler.Instant
	firstSignal atomic.Uint64
}

// NewPlatformFrameSignal creates an empty signal accumulator.
func NewPlatformFrameSignal() *PlatformFrameSignal {
	s := &PlatformFrameSignal{}
	if profilerFeatureEnabled {
		s.origin = scheduler.InstantNow()
		s.firstSignal.Store(platformFrameSignalEmpty)
	}
	return s
}

// PlatformFrameSignalCapture captures a platform request timestamp only when profiling is enabled;
// capture is not invoked otherwise.
func PlatformFrameSignalCapture(capture func() scheduler.Instant) (scheduler.Instant, bool) {
	if profilerFeatureEnabled {
		return capture(), true
	}
	return scheduler.Instant{}, false
}

// Record records a platform frame request, retaining the first undrained timestamp.
func (s *PlatformFrameSignal) Record(at scheduler.Instant, source FrameRequestSource) {
	if !profilerFeatureEnabled {
		return
	}
	nanoseconds := min(uint64(at.SaturatingDurationSince(s.origin)), (uint64(math.MaxUint64)>>1)-1)
	var sourceBit uint64
	switch source {
	case FrameRequestSourceNativeCallback:
		sourceBit = 0
	case FrameRequestSourceLocalSchedule:
		sourceBit = 1
	}
	value := (nanoseconds << 1) | sourceBit
	for {
		current := s.firstSignal.Load()
		if current <= value || s.firstSignal.CompareAndSwap(current, value) {
			return
		}
	}
}

// Take drains the first platform frame request time and source, leaving the accumulator empty.
func (s *PlatformFrameSignal) Take() (scheduler.Instant, FrameRequestSource, bool) {
	if !profilerFeatureEnabled {
		return scheduler.Instant{}, FrameRequestSourceNativeCallback, false
	}
	signal := s.firstSignal.Swap(platformFrameSignalEmpty)
	if signal == platformFrameSignalEmpty {
		return scheduler.Instant{}, FrameRequestSourceNativeCallback, false
	}
	source := FrameRequestSourceLocalSchedule
	if signal&1 == 0 {
		source = FrameRequestSourceNativeCallback
	}
	return s.origin.Add(time.Duration(signal >> 1)), source, true
}

// AppLifecyclePhase is the application's lifecycle phase, as owned and reported by a mobile OS.
type AppLifecyclePhase uint8

const (
	// AppLifecyclePhaseActive is foreground and receiving input.
	AppLifecyclePhaseActive AppLifecyclePhase = iota
	// AppLifecyclePhaseInactive is foreground (visible) but not receiving input.
	AppLifecyclePhaseInactive
	// AppLifecyclePhaseBackground is not visible; the process may be killed without further notice.
	AppLifecyclePhaseBackground
	// AppLifecyclePhaseForeground is becoming visible again, before input is restored.
	AppLifecyclePhaseForeground
	// AppLifecyclePhaseDisconnected means the native scene has disconnected.
	AppLifecyclePhaseDisconnected
)

// WindowInsets are the regions of a window that are obscured or reserved by the system.
type WindowInsets struct {
	// SafeArea is the region covered by system UI or hardware.
	SafeArea Edges[Pixels]
	// IME is the region covered by the keyboard, when present.
	IME Edges[Pixels]
	// CornerRadii are the radii of the window's visible corners where the display rounds them.
	CornerRadii Corners[Pixels]
}

// Effective returns the combined inset content should avoid.
func (i WindowInsets) Effective() Edges[Pixels] {
	return Edges[Pixels]{
		Top:    max(i.SafeArea.Top, i.IME.Top),
		Right:  max(i.SafeArea.Right, i.IME.Right),
		Bottom: max(i.SafeArea.Bottom, i.IME.Bottom),
		Left:   max(i.SafeArea.Left, i.IME.Left),
	}
}

// TextInputStateChange is a change in the state of the focused text input.
type TextInputStateChange uint8

const (
	// TextInputStateChangeFocusGained means the window gained an active text input.
	TextInputStateChangeFocusGained TextInputStateChange = iota
	// TextInputStateChangeFocusLost means the window no longer has an active text input.
	TextInputStateChangeFocusLost
	// TextInputStateChangeSelectionChanged means the selection or caret moved.
	TextInputStateChangeSelectionChanged
	// TextInputStateChangeContentChanged means the content changed outside of platform-initiated edits.
	TextInputStateChangeContentChanged
)

// RawWindowHandleKind is the windowing system a RawWindowHandle belongs to.
type RawWindowHandleKind uint8

const (
	// RawWindowHandleNone is an empty handle.
	RawWindowHandleNone RawWindowHandleKind = iota
	// RawWindowHandleWayland is a wl_display and wl_surface pair.
	RawWindowHandleWayland
	// RawWindowHandleXcb is an xcb connection, window and visual.
	RawWindowHandleXcb
	// RawWindowHandleAppKit is an NSView.
	RawWindowHandleAppKit
	// RawWindowHandleUIKit is a UIView.
	RawWindowHandleUIKit
	// RawWindowHandleWin32 is an HWND and its HINSTANCE.
	RawWindowHandleWin32
	// RawWindowHandleWeb is a canvas element named by its id.
	RawWindowHandleWeb
)

// RawWindowHandle is the window and display handle pair a renderer creates its surface from
// (raw_window_handle's HasWindowHandle + HasDisplayHandle). Only the fields of Kind are set.
type RawWindowHandle struct {
	// Kind selects the windowing system.
	Kind RawWindowHandleKind
	// WaylandDisplay is the wl_display pointer.
	WaylandDisplay uintptr
	// WaylandSurface is the wl_surface pointer.
	WaylandSurface uintptr
	// XcbConnection is the xcb_connection_t pointer.
	XcbConnection uintptr
	// XcbWindow is the X11 window id.
	XcbWindow uint32
	// XcbVisual is the X11 visual id.
	XcbVisual uint32
	// AppKitView is the NSView pointer.
	AppKitView uintptr
	// UIKitView is the UIView pointer.
	UIKitView uintptr
	// Win32HWND is the window handle.
	Win32HWND uintptr
	// Win32HInstance is the module instance handle.
	Win32HInstance uintptr
	// WebCanvasID is the id attribute of the canvas element.
	WebCanvasID string
}

// NativeViewAttachment is a native view attached with PlatformWindow.AttachNativeView; Close
// removes it from the window (upstream drops the returned box).
type NativeViewAttachment interface {
	Close()
}

// PlatformWindow is the interface every platform window implements; embed PlatformWindowDefaults
// for upstream's provided methods. Methods upstream gates by cfg exist on every GOOS.
type PlatformWindow interface {
	RawWindowHandle() RawWindowHandle
	Bounds() Bounds[Pixels]
	IsMaximized() bool
	WindowBounds() WindowBounds
	ContentSize() Size[Pixels]
	VisualViewportBounds() Bounds[Pixels]
	OnVisualViewportChanged(callback func())
	PrepareFrame() bool
	Resize(size Size[Pixels])
	ScaleFactor() float32
	Appearance() WindowAppearance
	Display() PlatformDisplay
	MousePosition() Point[Pixels]
	Modifiers() Modifiers
	Capslock() Capslock
	SetInputHandler(inputHandler *PlatformInputHandler)
	TakeInputHandler() *PlatformInputHandler
	SetTextInputConfiguration(configuration TextInputConfiguration)
	Prompt(level PromptLevel, msg string, detail Option[string], answers []PromptButton) *scheduler.OneshotReceiver[int]
	Activate()
	RequestAttention()
	IsActive() bool
	Visibility() WindowVisibility
	IsHovered() bool
	BackgroundAppearance() WindowBackgroundAppearance
	SetTitle(title string)
	SetBackgroundAppearance(backgroundAppearance WindowBackgroundAppearance)
	Minimize()
	Zoom()
	ToggleFullscreen()
	IsFullscreen() bool
	DebugState() jsontext.Value
	FrameWaker() func()
	OnRequestFrame(callback func(options RequestFrameOptions))
	OnInput(callback func(input PlatformInput) DispatchEventResult)
	OnActiveStatusChange(callback func(active bool))
	OnVisibilityChange(callback func(visibility WindowVisibility))
	OnHoverStatusChange(callback func(hovered bool))
	OnResize(callback func(size Size[Pixels], scaleFactor float32))
	OnMoved(callback func())
	OnDisplayChanged(callback func())
	OnShouldClose(callback func() bool)
	OnHitTestWindowControl(callback func() (WindowControlArea, bool))
	OnClose(callback func())
	AttachNativeView(id NativeViewID, view uintptr) NativeViewAttachment
	SetFocusedNativeView(id Option[NativeViewID])
	OnHitTestNativeView(callback func(position Point[Pixels]) (NativeViewID, bool))
	OnNativeViewFocused(callback func(id NativeViewID))
	OnKeyDownBeforeNativeView(callback func(event KeyDownEvent) bool)
	OnAppearanceChanged(callback func())
	OnButtonLayoutChanged(callback func())
	Draw(scene *Scene)
	ScheduleFrame()
	SpriteAtlas() PlatformAtlas
	IsSubpixelRenderingSupported() bool
	GetTitle() string
	TabbedWindows() ([]SystemWindowTab, bool)
	TabBarVisible() bool
	SetEdited(edited bool)
	SetDocumentPath(path Option[string])
	ToggleSimpleFullscreen()
	IsSimpleFullscreen() bool
	SetTrafficLightPosition(position Point[Pixels])
	SetNativeTitlebarVisibility(visible bool)
	ShowCharacterPalette()
	TitlebarDoubleClick(isResizable bool, isMinimizable bool)
	OnMoveTabToNewWindow(callback func())
	OnMergeAllWindows(callback func())
	OnSelectPreviousTab(callback func())
	OnSelectNextTab(callback func())
	OnToggleTabBar(callback func())
	MergeAllWindows()
	MoveTabToNewWindow()
	ToggleWindowTabOverview()
	SetTabbingIdentifier(identifier Option[string])
	NativeWindowState() ([]byte, bool)
	RestoreNativeWindowState(state []byte)
	GetRawHandle() uintptr
	InnerWindowBounds() WindowBounds
	RequestDecorations(decorations WindowDecorations)
	ShowWindowMenu(position Point[Pixels])
	StartWindowMove()
	CanStartExternalDrag() bool
	StartExternalDrag(payload ExternalDragPayload) bool
	StartWindowResize(edge ResizeEdge)
	SetExclusiveZone(zone Pixels)
	SetExclusiveEdge(edge LayerShellAnchor)
	SetInputRegion(region Option[[]Bounds[Pixels]])
	WindowDecorations() Decorations
	SetAppID(appID string)
	MapWindow() error
	WindowControls() WindowControls
	SetClientInset(inset Pixels)
	SetClientCornerRadii(radii Corners[Pixels])
	GPUSpecs() (GPUSpecs, bool)
	UpdateIMEPosition(bounds Bounds[Pixels])
	Insets() WindowInsets
	OnInsetsChanged(callback func(insets WindowInsets))
	SetBackHandler(callback func())
	SetBackEnabled(enabled bool)
	ShowSoftKeyboard()
	HideSoftKeyboard()
	TextInputStateChanged(change TextInputStateChange)
	PlaySystemBell()
	A11yInit(callbacks A11yCallbacks)
	A11yTreeUpdate(treeUpdate a11y.TreeUpdate)
	A11yUpdateWindowBounds()
	AsTest() *TestWindow
	RenderToImage(scene *Scene) (*image.RGBA, error)
	Close()
}

// PlatformWindowDefaults carries the default bodies of PlatformWindow's provided methods (DESIGN 5.4).
// Embed it in a PlatformWindow implementation and bind it with BindPlatformWindowDefaults.
type PlatformWindowDefaults struct {
	self PlatformWindow
}

// BindPlatformWindowDefaults makes the provided methods that call other PlatformWindow methods dispatch to self.
func BindPlatformWindowDefaults(d *PlatformWindowDefaults, self PlatformWindow) {
	d.self = self
}

func (d *PlatformWindowDefaults) bound() PlatformWindow {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "PlatformWindowDefaults", "BindPlatformWindowDefaults"))
	}
	return d.self
}

// VisualViewportBounds returns the visible viewport relative to the content origin.
func (d *PlatformWindowDefaults) VisualViewportBounds() Bounds[Pixels] {
	return NewBounds(Point[Pixels]{}, d.bound().ContentSize())
}

// OnVisualViewportChanged registers a callback for visible geometry changes; the default ignores it.
func (d *PlatformWindowDefaults) OnVisualViewportChanged(callback func()) {}

// PrepareFrame samples platform geometry before a draw and reports whether view caches must be invalidated.
func (d *PlatformWindowDefaults) PrepareFrame() bool {
	return false
}

// SetTextInputConfiguration applies the focused text region's configuration; the default ignores it.
func (d *PlatformWindowDefaults) SetTextInputConfiguration(configuration TextInputConfiguration) {}

// RequestAttention asks the operating system to draw attention to this window.
func (d *PlatformWindowDefaults) RequestAttention() {}

// DebugState returns platform-specific diagnostic state.
func (d *PlatformWindowDefaults) DebugState() jsontext.Value {
	return jsontext.Value("null")
}

// FrameWaker returns a function that wakes the frame loop, or nil.
func (d *PlatformWindowDefaults) FrameWaker() func() {
	return nil
}

// AttachNativeView adds a platform view beneath the window's content; the default returns nil.
func (d *PlatformWindowDefaults) AttachNativeView(id NativeViewID, view uintptr) NativeViewAttachment {
	return nil
}

// SetFocusedNativeView gives keyboard input to the native view; the default ignores it.
func (d *PlatformWindowDefaults) SetFocusedNativeView(id Option[NativeViewID]) {}

// OnHitTestNativeView registers the native-view hit test callback; the default ignores it.
func (d *PlatformWindowDefaults) OnHitTestNativeView(callback func(position Point[Pixels]) (NativeViewID, bool)) {
}

// OnNativeViewFocused registers the native-view focus callback; the default ignores it.
func (d *PlatformWindowDefaults) OnNativeViewFocused(callback func(id NativeViewID)) {}

// OnKeyDownBeforeNativeView registers the key callback offered before the native view; the default ignores it.
func (d *PlatformWindowDefaults) OnKeyDownBeforeNativeView(callback func(event KeyDownEvent) bool) {}

// OnButtonLayoutChanged registers a callback for button layout changes; the default ignores it.
func (d *PlatformWindowDefaults) OnButtonLayoutChanged(callback func()) {}

// ScheduleFrame asks the platform for a frame; the default does nothing.
func (d *PlatformWindowDefaults) ScheduleFrame() {}

// GetTitle returns the window title (macOS).
func (d *PlatformWindowDefaults) GetTitle() string {
	return ""
}

// TabbedWindows returns the window's tab group (macOS).
func (d *PlatformWindowDefaults) TabbedWindows() ([]SystemWindowTab, bool) {
	return nil, false
}

// TabBarVisible reports whether the tab bar is visible (macOS).
func (d *PlatformWindowDefaults) TabBarVisible() bool {
	return false
}

// SetEdited marks the window as edited (macOS).
func (d *PlatformWindowDefaults) SetEdited(edited bool) {}

// SetDocumentPath sets the window's represented document (macOS).
func (d *PlatformWindowDefaults) SetDocumentPath(path Option[string]) {}

// ToggleSimpleFullscreen toggles simple fullscreen (macOS).
func (d *PlatformWindowDefaults) ToggleSimpleFullscreen() {}

// IsSimpleFullscreen reports whether the window is in simple fullscreen (macOS).
func (d *PlatformWindowDefaults) IsSimpleFullscreen() bool {
	return false
}

// SetTrafficLightPosition moves the traffic light buttons (macOS).
func (d *PlatformWindowDefaults) SetTrafficLightPosition(position Point[Pixels]) {}

// SetNativeTitlebarVisibility shows or hides the native titlebar (macOS).
func (d *PlatformWindowDefaults) SetNativeTitlebarVisibility(visible bool) {}

// ShowCharacterPalette shows the character palette.
func (d *PlatformWindowDefaults) ShowCharacterPalette() {}

// TitlebarDoubleClick performs the platform's titlebar double-click action.
func (d *PlatformWindowDefaults) TitlebarDoubleClick(isResizable bool, isMinimizable bool) {}

// OnMoveTabToNewWindow registers a tab callback (macOS).
func (d *PlatformWindowDefaults) OnMoveTabToNewWindow(callback func()) {}

// OnMergeAllWindows registers a tab callback (macOS).
func (d *PlatformWindowDefaults) OnMergeAllWindows(callback func()) {}

// OnSelectPreviousTab registers a tab callback (macOS).
func (d *PlatformWindowDefaults) OnSelectPreviousTab(callback func()) {}

// OnSelectNextTab registers a tab callback (macOS).
func (d *PlatformWindowDefaults) OnSelectNextTab(callback func()) {}

// OnToggleTabBar registers a tab callback (macOS).
func (d *PlatformWindowDefaults) OnToggleTabBar(callback func()) {}

// MergeAllWindows merges all windows into one tab group (macOS).
func (d *PlatformWindowDefaults) MergeAllWindows() {}

// MoveTabToNewWindow moves the current tab to a new window (macOS).
func (d *PlatformWindowDefaults) MoveTabToNewWindow() {}

// ToggleWindowTabOverview toggles the tab overview (macOS).
func (d *PlatformWindowDefaults) ToggleWindowTabOverview() {}

// SetTabbingIdentifier sets the window's tabbing identifier (macOS).
func (d *PlatformWindowDefaults) SetTabbingIdentifier(identifier Option[string]) {}

// NativeWindowState returns the platform's serialized window state.
func (d *PlatformWindowDefaults) NativeWindowState() ([]byte, bool) {
	return nil, false
}

// RestoreNativeWindowState restores serialized window state.
func (d *PlatformWindowDefaults) RestoreNativeWindowState(state []byte) {}

// GetRawHandle returns the HWND (Windows); other platforms return 0.
func (d *PlatformWindowDefaults) GetRawHandle() uintptr {
	return 0
}

// InnerWindowBounds returns the window bounds without decorations (Linux).
func (d *PlatformWindowDefaults) InnerWindowBounds() WindowBounds {
	return d.bound().WindowBounds()
}

// RequestDecorations requests server or client side decorations (Linux).
func (d *PlatformWindowDefaults) RequestDecorations(decorations WindowDecorations) {}

// ShowWindowMenu shows the window menu (Linux).
func (d *PlatformWindowDefaults) ShowWindowMenu(position Point[Pixels]) {}

// StartWindowMove starts an interactive move (Linux).
func (d *PlatformWindowDefaults) StartWindowMove() {}

// CanStartExternalDrag reports whether a native drag session can be started.
func (d *PlatformWindowDefaults) CanStartExternalDrag() bool {
	return false
}

// StartExternalDrag starts a native drag session with the payload.
func (d *PlatformWindowDefaults) StartExternalDrag(payload ExternalDragPayload) bool {
	return false
}

// StartWindowResize starts an interactive resize from the edge (Linux).
func (d *PlatformWindowDefaults) StartWindowResize(edge ResizeEdge) {}

// SetExclusiveZone sets the layer-shell exclusive zone (Linux).
func (d *PlatformWindowDefaults) SetExclusiveZone(zone Pixels) {}

// SetExclusiveEdge sets the layer-shell exclusive edge (Linux, Wayland).
func (d *PlatformWindowDefaults) SetExclusiveEdge(edge LayerShellAnchor) {}

// SetInputRegion sets the input region; None means the whole window and Some of an empty slice means none of it.
func (d *PlatformWindowDefaults) SetInputRegion(region Option[[]Bounds[Pixels]]) {}

// WindowDecorations returns the current decorations configuration (Linux).
func (d *PlatformWindowDefaults) WindowDecorations() Decorations {
	return DecorationsServer()
}

// SetAppID sets the application id of the window (Linux).
func (d *PlatformWindowDefaults) SetAppID(appID string) {}

// MapWindow maps the window (Linux).
func (d *PlatformWindowDefaults) MapWindow() error {
	return nil
}

// WindowControls returns the window controls the platform supports (Linux).
func (d *PlatformWindowDefaults) WindowControls() WindowControls {
	return DefaultWindowControls()
}

// SetClientInset sets the client-side decoration inset (Linux).
func (d *PlatformWindowDefaults) SetClientInset(inset Pixels) {}

// SetClientCornerRadii sets the client-side decoration corner radii (Linux).
func (d *PlatformWindowDefaults) SetClientCornerRadii(radii Corners[Pixels]) {}

// Insets returns the regions of this window obscured or reserved by the system.
func (d *PlatformWindowDefaults) Insets() WindowInsets {
	return WindowInsets{}
}

// OnInsetsChanged registers a callback for inset changes; the default ignores it.
func (d *PlatformWindowDefaults) OnInsetsChanged(callback func(insets WindowInsets)) {}

// SetBackHandler sets the handler for the system back action; the default ignores it.
func (d *PlatformWindowDefaults) SetBackHandler(callback func()) {}

// SetBackEnabled declares whether the application would handle the system back action.
func (d *PlatformWindowDefaults) SetBackEnabled(enabled bool) {}

// ShowSoftKeyboard requests that the soft keyboard be shown.
func (d *PlatformWindowDefaults) ShowSoftKeyboard() {}

// HideSoftKeyboard requests that the soft keyboard be hidden.
func (d *PlatformWindowDefaults) HideSoftKeyboard() {}

// TextInputStateChanged informs the operating system that the text input state has changed.
func (d *PlatformWindowDefaults) TextInputStateChanged(change TextInputStateChange) {}

// PlaySystemBell plays the system bell.
func (d *PlatformWindowDefaults) PlaySystemBell() {}

// A11yInit initializes the accessibility adapter with callbacks.
func (d *PlatformWindowDefaults) A11yInit(callbacks A11yCallbacks) {}

// A11yTreeUpdate provides a TreeUpdate to the accessibility adapter.
func (d *PlatformWindowDefaults) A11yTreeUpdate(treeUpdate a11y.TreeUpdate) {}

// A11yUpdateWindowBounds informs the adapter of updated window bounds.
func (d *PlatformWindowDefaults) A11yUpdateWindowBounds() {}

// AsTest returns the window as a TestWindow, or nil.
func (d *PlatformWindowDefaults) AsTest() *TestWindow {
	return nil
}

// RenderToImage renders the scene to an RGBA image without presenting it.
func (d *PlatformWindowDefaults) RenderToImage(scene *Scene) (*image.RGBA, error) {
	return nil, errors.New("render_to_image not implemented for this platform")
}

// Close releases the platform window; it is upstream's drop of the window value.
func (d *PlatformWindowDefaults) Close() {}

// PlatformHeadlessRenderer is a renderer for headless windows that produces real rendered output.
type PlatformHeadlessRenderer interface {
	RenderSceneToImage(scene *Scene, size Size[DevicePixels]) (*image.RGBA, error)
	RenderScene(scene *Scene, size Size[DevicePixels]) error
	SpriteAtlas() PlatformAtlas
}

// RunnableVariant is a runnable with metadata.
type RunnableVariant = *scheduler.Runnable

// TimerResolutionGuard restores the timer resolution when run.
type TimerResolutionGuard = gpuiutil.Deferred

// TasksIncluded selects which tasks a dispatcher reports.
type TasksIncluded uint8

const (
	// TasksIncludedOnlyCompleted reports completed tasks only.
	TasksIncludedOnlyCompleted TasksIncluded = iota
	// TasksIncludedCompletedAndRunning reports completed and running tasks.
	TasksIncludedCompletedAndRunning
)

// PlatformDispatcher schedules runnables on the platform's threads; embed PlatformDispatcherDefaults
// for upstream's provided methods.
type PlatformDispatcher interface {
	IsMainThread() bool
	Dispatch(runnable RunnableVariant, priority Priority)
	DispatchOnMainThread(runnable RunnableVariant, priority Priority)
	DispatchAfter(duration time.Duration, runnable RunnableVariant)
	DispatchOnMainThreadWhenIdle(runnable RunnableVariant, timeout Option[time.Duration])
	IdleTimeRemaining() (time.Duration, bool)
	SpawnRealtime(f func())
	Now() scheduler.Instant
	IncreaseTimerResolution() TimerResolutionGuard
	PreventAppNap(reason string) *ActivityGuard
	AsTest() *TestDispatcher
	AsThreaded() *ThreadedDispatcher
}

// PlatformDispatcherDefaults carries the default bodies of PlatformDispatcher's provided methods.
// Bind it with BindPlatformDispatcherDefaults in the implementation's constructor.
type PlatformDispatcherDefaults struct {
	self PlatformDispatcher
}

// BindPlatformDispatcherDefaults makes the provided methods dispatch to self.
func BindPlatformDispatcherDefaults(d *PlatformDispatcherDefaults, self PlatformDispatcher) {
	d.self = self
}

func (d *PlatformDispatcherDefaults) bound() PlatformDispatcher {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "PlatformDispatcherDefaults", "BindPlatformDispatcherDefaults"))
	}
	return d.self
}

// DispatchOnMainThreadWhenIdle runs the runnable on the main thread when it is idle; the default
// dispatches it at low priority.
func (d *PlatformDispatcherDefaults) DispatchOnMainThreadWhenIdle(runnable RunnableVariant, timeout Option[time.Duration]) {
	d.bound().DispatchOnMainThread(runnable, scheduler.PriorityLow)
}

// IdleTimeRemaining returns how long the main thread is expected to stay idle.
func (d *PlatformDispatcherDefaults) IdleTimeRemaining() (time.Duration, bool) {
	return 0, false
}

// Now returns the dispatcher's current instant.
func (d *PlatformDispatcherDefaults) Now() scheduler.Instant {
	return scheduler.InstantNow()
}

// IncreaseTimerResolution raises the timer resolution until the returned guard runs.
func (d *PlatformDispatcherDefaults) IncreaseTimerResolution() TimerResolutionGuard {
	return gpuiutil.Defer(func() {})
}

// PreventAppNap keeps the app from being napped until the guard is released.
func (d *PlatformDispatcherDefaults) PreventAppNap(reason string) *ActivityGuard {
	return ActivityGuardNoop()
}

// AsTest returns the dispatcher as a TestDispatcher, or nil.
func (d *PlatformDispatcherDefaults) AsTest() *TestDispatcher {
	return nil
}

// AsThreaded returns the dispatcher as a ThreadedDispatcher, or nil.
func (d *PlatformDispatcherDefaults) AsThreaded() *ThreadedDispatcher {
	return nil
}

// PlatformTextSystem is the platform's font and shaping back end; embed PlatformTextSystemDefaults
// for upstream's provided methods.
type PlatformTextSystem interface {
	AddFonts(fonts [][]byte) error
	SetMissingGlyphSink(sink MissingGlyphSink)
	AllFontNames() []string
	FontID(descriptor Font) (FontID, error)
	PrewarmFonts(fontIDs []FontID)
	FontMetrics(fontID FontID) FontMetrics
	TypographicBounds(fontID FontID, glyphID GlyphID) (Bounds[float32], error)
	Advance(fontID FontID, glyphID GlyphID) (Size[float32], error)
	GlyphForChar(fontID FontID, ch rune) (GlyphID, bool)
	GlyphRasterBounds(params RenderGlyphParams) (Bounds[DevicePixels], error)
	RasterizeGlyph(params RenderGlyphParams, rasterBounds Bounds[DevicePixels]) (Size[DevicePixels], []byte, error)
	LayoutLine(text string, fontSize Pixels, runs []FontRun) LineLayout
	RecommendedRenderingMode(fontID FontID, fontSize Pixels) TextRenderingMode
	GlyphDilationForColor(color Hsla) uint8
}

// PlatformTextSystemDefaults carries the default bodies of PlatformTextSystem's provided methods.
type PlatformTextSystemDefaults struct{}

// SetMissingGlyphSink installs a nonblocking sink for unresolved grapheme clusters; the default ignores it.
func (PlatformTextSystemDefaults) SetMissingGlyphSink(sink MissingGlyphSink) {}

// PrewarmFonts prewarms system font caches; the default does nothing.
func (PlatformTextSystemDefaults) PrewarmFonts(fontIDs []FontID) {}

// GlyphDilationForColor returns the dilation level for a glyph painted in the color.
func (PlatformTextSystemDefaults) GlyphDilationForColor(color Hsla) uint8 {
	return 0
}

// NoopTextSystem is a text system that shapes every character with fixed metrics.
type NoopTextSystem struct {
	PlatformTextSystemDefaults
}

// NewNoopTextSystem returns a NoopTextSystem.
func NewNoopTextSystem() *NoopTextSystem {
	return &NoopTextSystem{}
}

// AddFonts accepts and ignores the fonts.
func (*NoopTextSystem) AddFonts(fonts [][]byte) error {
	return nil
}

// AllFontNames returns no font names.
func (*NoopTextSystem) AllFontNames() []string {
	return []string{}
}

// FontID returns font 1 for every descriptor.
func (*NoopTextSystem) FontID(descriptor Font) (FontID, error) {
	return FontID(1), nil
}

// FontMetrics returns fixed metrics.
func (*NoopTextSystem) FontMetrics(fontID FontID) FontMetrics {
	return FontMetrics{
		UnitsPerEm:         1000,
		Ascent:             1025.0,
		Descent:            -275.0,
		LineGap:            0.0,
		UnderlinePosition:  -95.0,
		UnderlineThickness: 60.0,
		CapHeight:          698.0,
		XHeight:            516.0,
		BoundingBox: Bounds[float32]{
			Origin: Point[float32]{X: -260.0, Y: -245.0},
			Size:   Size[float32]{Width: 1501.0, Height: 1364.0},
		},
	}
}

// TypographicBounds returns fixed bounds.
func (*NoopTextSystem) TypographicBounds(fontID FontID, glyphID GlyphID) (Bounds[float32], error) {
	return Bounds[float32]{
		Origin: Point[float32]{X: 54.0, Y: 0.0},
		Size:   NewSize[float32](392.0, 528.0),
	}, nil
}

// Advance returns 600 units per glyph id.
func (*NoopTextSystem) Advance(fontID FontID, glyphID GlyphID) (Size[float32], error) {
	return NewSize(600.0*float32(glyphID), 0.0), nil
}

// GlyphForChar returns the character's UTF-16 length as its glyph id.
func (*NoopTextSystem) GlyphForChar(fontID FontID, ch rune) (GlyphID, bool) {
	return GlyphID(utf16.RuneLen(ch)), true
}

// GlyphRasterBounds returns empty bounds.
func (*NoopTextSystem) GlyphRasterBounds(params RenderGlyphParams) (Bounds[DevicePixels], error) {
	return Bounds[DevicePixels]{}, nil
}

// RasterizeGlyph returns the raster size with no pixels.
func (*NoopTextSystem) RasterizeGlyph(params RenderGlyphParams, rasterBounds Bounds[DevicePixels]) (Size[DevicePixels], []byte, error) {
	return rasterBounds.Size, []byte{}, nil
}

// LayoutLine lays out the text with one em per glyph and two per emoji.
func (s *NoopTextSystem) LayoutLine(text string, fontSize Pixels, runs []FontRun) LineLayout {
	position := Px(0.)
	metrics := s.FontMetrics(FontID(0))
	mGlyph, _ := s.GlyphForChar(FontID(0), 'm')
	mAdvance, err := s.Advance(FontID(0), mGlyph)
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %+v", err))
	}
	emWidth := fontSize * Pixels(mAdvance.Width) / Pixels(float32(metrics.UnitsPerEm))
	var glyphs []ShapedGlyph
	for ix, c := range text {
		if glyph, ok := s.GlyphForChar(FontID(0), c); ok {
			glyphs = append(glyphs, ShapedGlyph{
				ID:       glyph,
				Position: NewPoint(position, Px(0.)),
				Index:    ix,
				IsEmoji:  glyph == 2,
			})
			if glyph == 2 {
				position += emWidth * 2.0
			} else {
				position += emWidth
			}
		} else {
			position += emWidth
		}
	}
	var shapedRuns []ShapedRun
	if len(glyphs) != 0 {
		shapedRuns = append(shapedRuns, ShapedRun{
			FontID: FontID(0),
			Glyphs: glyphs,
		})
	} else {
		position = Px(0.)
	}

	return LineLayout{
		FontSize: fontSize,
		Width:    position,
		Ascent:   fontSize * Pixels(metrics.Ascent/float32(metrics.UnitsPerEm)),
		Descent:  fontSize * Pixels(metrics.Descent/float32(metrics.UnitsPerEm)),
		Runs:     shapedRuns,
		Len:      len(text),
	}
}

// RecommendedRenderingMode returns grayscale rendering.
func (*NoopTextSystem) RecommendedRenderingMode(fontID FontID, fontSize Pixels) TextRenderingMode {
	return TextRenderingModeGrayscale
}

var gammaIncorrectTargetRatios = [13][4]float32{
	{0.0000 / 4.0, 0.0000 / 4.0, 0.0000 / 4.0, 0.0000 / 4.0},
	{0.0166 / 4.0, -0.0807 / 4.0, 0.2227 / 4.0, -0.0751 / 4.0},
	{0.0350 / 4.0, -0.1760 / 4.0, 0.4325 / 4.0, -0.1370 / 4.0},
	{0.0543 / 4.0, -0.2821 / 4.0, 0.6302 / 4.0, -0.1876 / 4.0},
	{0.0739 / 4.0, -0.3963 / 4.0, 0.8167 / 4.0, -0.2287 / 4.0},
	{0.0933 / 4.0, -0.5161 / 4.0, 0.9926 / 4.0, -0.2616 / 4.0},
	{0.1121 / 4.0, -0.6395 / 4.0, 1.1588 / 4.0, -0.2877 / 4.0},
	{0.1300 / 4.0, -0.7649 / 4.0, 1.3159 / 4.0, -0.3080 / 4.0},
	{0.1469 / 4.0, -0.8911 / 4.0, 1.4644 / 4.0, -0.3234 / 4.0},
	{0.1627 / 4.0, -1.0170 / 4.0, 1.6051 / 4.0, -0.3347 / 4.0},
	{0.1773 / 4.0, -1.1420 / 4.0, 1.7385 / 4.0, -0.3426 / 4.0},
	{0.1908 / 4.0, -1.2652 / 4.0, 1.8650 / 4.0, -0.3476 / 4.0},
	{0.2031 / 4.0, -1.3864 / 4.0, 1.9851 / 4.0, -0.3501 / 4.0},
}

var gammaNorm13, gammaNorm24 = gammaNorms(0x10000, 0x100)

func gammaNorms(norm13, norm24 float64) (float32, float32) {
	return float32(norm13 / (255.0 * 255.0) * 4.0), float32(norm24 / (255.0) * 4.0)
}

// GetGammaCorrectionRatios computes gamma correction ratios for subpixel text rendering.
func GetGammaCorrectionRatios(gamma float32) [4]float32 {
	rounded := math.Round(float64(gamma * 10.0))
	if !(rounded >= 10) {
		rounded = 10
	}
	if rounded > 22 {
		rounded = 22
	}
	index := int(rounded) - 10
	ratios := gammaIncorrectTargetRatios[index]

	return [4]float32{
		ratios[0] * gammaNorm13,
		ratios[1] * gammaNorm24,
		ratios[2] * gammaNorm13,
		ratios[3] * gammaNorm24,
	}
}

// AtlasKeyKind is the variant of an AtlasKey.
type AtlasKeyKind uint8

const (
	// AtlasKeyGlyph is a rasterized glyph.
	AtlasKeyGlyph AtlasKeyKind = iota
	// AtlasKeySvg is a rasterized SVG.
	AtlasKeySvg
	// AtlasKeyImage is a decoded image frame.
	AtlasKeyImage
)

// AtlasKey identifies a tile in a sprite atlas. It is comparable and used as a map key; build it
// with AtlasKeyFromGlyph, AtlasKeyFromSvg or AtlasKeyFromImage.
type AtlasKey struct {
	kind  AtlasKeyKind
	glyph RenderGlyphParams
	svg   RenderSvgParams
	image RenderImageParams
}

// AtlasKeyFromGlyph returns the key of a rasterized glyph.
func AtlasKeyFromGlyph(params RenderGlyphParams) AtlasKey {
	return AtlasKey{kind: AtlasKeyGlyph, glyph: params}
}

// AtlasKeyFromSvg returns the key of a rasterized SVG.
func AtlasKeyFromSvg(params RenderSvgParams) AtlasKey {
	return AtlasKey{kind: AtlasKeySvg, svg: params}
}

// AtlasKeyFromImage returns the key of a decoded image frame.
func AtlasKeyFromImage(params RenderImageParams) AtlasKey {
	return AtlasKey{kind: AtlasKeyImage, image: params}
}

// Kind returns the key's variant.
func (k AtlasKey) Kind() AtlasKeyKind {
	return k.kind
}

// Glyph returns the glyph parameters when the key is a glyph.
func (k AtlasKey) Glyph() (RenderGlyphParams, bool) {
	return k.glyph, k.kind == AtlasKeyGlyph
}

// Svg returns the SVG parameters when the key is an SVG.
func (k AtlasKey) Svg() (RenderSvgParams, bool) {
	return k.svg, k.kind == AtlasKeySvg
}

// Image returns the image parameters when the key is an image.
func (k AtlasKey) Image() (RenderImageParams, bool) {
	return k.image, k.kind == AtlasKeyImage
}

// TextureKind returns the texture kind for this atlas key.
func (k AtlasKey) TextureKind() AtlasTextureKind {
	switch k.kind {
	case AtlasKeyGlyph:
		if k.glyph.IsEmoji {
			return AtlasTextureKindPolychrome
		} else if k.glyph.SubpixelRendering {
			return AtlasTextureKindSubpixel
		}
		return AtlasTextureKindMonochrome
	case AtlasKeySvg:
		return AtlasTextureKindMonochrome
	case AtlasKeyImage:
		return AtlasTextureKindPolychrome
	}
	panic(fmt.Sprintf("gpui: invalid AtlasKey kind %d", uint8(k.kind)))
}

// AtlasBuildFunc produces a tile's size and bytes; ok false means nothing to insert.
type AtlasBuildFunc func() (size Size[DevicePixels], bytes []byte, ok bool, err error)

// PlatformAtlas is a sprite atlas; embed PlatformAtlasDefaults for upstream's provided methods.
type PlatformAtlas interface {
	GetOrInsertWith(key AtlasKey, build AtlasBuildFunc) (AtlasTile, bool, error)
	Remove(key AtlasKey)
	Contains(key AtlasKey) bool
}

// PlatformAtlasDefaults carries the default bodies of PlatformAtlas's provided methods.
type PlatformAtlasDefaults struct{}

// Contains reports whether the atlas holds a tile for the key; the default reports false.
func (PlatformAtlasDefaults) Contains(key AtlasKey) bool {
	return false
}

// AtlasBackend allocates and frees tiles for an AtlasState.
type AtlasBackend interface {
	Insert(kind AtlasTextureKind, size Size[DevicePixels], bytes []byte) (AtlasTile, error)
	Remove(tile AtlasTile)
}

// AtlasState maps atlas keys to tiles allocated by its backend; B is usually a pointer type.
type AtlasState[B AtlasBackend] struct {
	tilesByKey map[AtlasKey]AtlasTile
	// Backend allocates the tiles.
	Backend B
}

// NewAtlasState returns an empty AtlasState over backend.
func NewAtlasState[B AtlasBackend](backend B) *AtlasState[B] {
	return &AtlasState[B]{
		tilesByKey: map[AtlasKey]AtlasTile{},
		Backend:    backend,
	}
}

// Contains reports whether the state holds a tile for the key.
func (s *AtlasState[B]) Contains(key AtlasKey) bool {
	_, ok := s.tilesByKey[key]
	return ok
}

// Clear forgets every tile and resets the backend.
func (s *AtlasState[B]) Clear(resetBackend func(backend B)) {
	clear(s.tilesByKey)
	resetBackend(s.Backend)
}

// GetOrInsertWith returns the key's tile, building and inserting it on a miss; a failed or
// empty build is not cached.
func (s *AtlasState[B]) GetOrInsertWith(key AtlasKey, build AtlasBuildFunc) (AtlasTile, bool, error) {
	if tile, ok := s.tilesByKey[key]; ok {
		return tile, true, nil
	}
	size, bytes, ok, err := build()
	if err != nil {
		return AtlasTile{}, false, err
	}
	if !ok {
		return AtlasTile{}, false, nil
	}
	tile, err := s.Backend.Insert(key.TextureKind(), size, bytes)
	if err != nil {
		return AtlasTile{}, false, err
	}
	s.tilesByKey[key] = tile
	return tile, true, nil
}

// Remove removes the key's tile and frees it in the backend.
func (s *AtlasState[B]) Remove(key AtlasKey) {
	if tile, ok := s.tilesByKey[key]; ok {
		delete(s.tilesByKey, key)
		s.Backend.Remove(tile)
	}
}

// HeadlessAtlas is a sprite atlas for windows without a GPU: it hands out uniquely identified
// tiles without uploading any pixels. The zero value is ready to use.
type HeadlessAtlas struct {
	mu    sync.Mutex
	state *AtlasState[*HeadlessAtlasBackend]
}

func (a *HeadlessAtlas) lock() *AtlasState[*HeadlessAtlasBackend] {
	a.mu.Lock()
	if a.state == nil {
		a.state = NewAtlasState(&HeadlessAtlasBackend{})
	}
	return a.state
}

// HeadlessAtlasBackend is the AtlasBackend of HeadlessAtlas.
type HeadlessAtlasBackend struct {
	nextID uint32
}

// Insert allocates a new texture and tile id for the tile.
func (b *HeadlessAtlasBackend) Insert(kind AtlasTextureKind, size Size[DevicePixels], bytes []byte) (AtlasTile, error) {
	b.nextID++
	textureID := b.nextID
	b.nextID++
	tileID := b.nextID
	return AtlasTile{
		TextureID: AtlasTextureID{
			Index: textureID,
			Kind:  kind,
		},
		TileID:  TileID(tileID),
		Padding: 0,
		Bounds: Bounds[DevicePixels]{
			Origin: Point[DevicePixels]{},
			Size:   size,
		},
	}, nil
}

// Remove frees nothing.
func (b *HeadlessAtlasBackend) Remove(tile AtlasTile) {}

// GetOrInsertWith returns the key's tile, building it on a miss.
func (a *HeadlessAtlas) GetOrInsertWith(key AtlasKey, build AtlasBuildFunc) (AtlasTile, bool, error) {
	state := a.lock()
	defer a.mu.Unlock()
	return state.GetOrInsertWith(key, build)
}

// Remove removes the key's tile.
func (a *HeadlessAtlas) Remove(key AtlasKey) {
	state := a.lock()
	defer a.mu.Unlock()
	state.Remove(key)
}

// Contains reports whether the atlas holds a tile for the key.
func (a *HeadlessAtlas) Contains(key AtlasKey) bool {
	state := a.lock()
	defer a.mu.Unlock()
	return state.Contains(key)
}

// AtlasTextureList is a list of atlas textures with a free list of empty slots.
type AtlasTextureList[T any] struct {
	// Textures holds the textures; None marks a freed slot.
	Textures []Option[T]
	// FreeList holds the indices of freed slots.
	FreeList []int
}

// Index returns the slot at index.
func (l *AtlasTextureList[T]) Index(index int) Option[T] {
	return l.Textures[index]
}

// Drain empties the list and returns its slots.
func (l *AtlasTextureList[T]) Drain() []Option[T] {
	l.FreeList = l.FreeList[:0]
	textures := l.Textures
	l.Textures = nil
	return textures
}

// IterMut yields a pointer to every present texture.
func (l *AtlasTextureList[T]) IterMut() iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for i := range l.Textures {
			if texture := l.Textures[i].AsMut(); texture != nil {
				if !yield(texture) {
					return
				}
			}
		}
	}
}

// AtlasTile is a tile in an atlas texture; its layout is shared with the GPU.
type AtlasTile struct {
	// TextureID is the texture this tile belongs to.
	TextureID AtlasTextureID
	// TileID is the unique id of this tile within its texture.
	TileID TileID
	// Padding is the padding around the tile content in pixels.
	Padding uint32
	// Bounds is the bounds of this tile within the texture.
	Bounds Bounds[DevicePixels]
}

// AtlasTextureID identifies an atlas texture; its layout is shared with the GPU.
type AtlasTextureID struct {
	// Index is the index of this texture in the atlas.
	Index uint32
	// Kind is the kind of content stored in this texture.
	Kind AtlasTextureKind
}

// AtlasTextureKind is the kind of content stored in an atlas texture; it is 4 bytes wide
// because it is part of GPU data (repr(C)).
type AtlasTextureKind uint32

const (
	// AtlasTextureKindMonochrome holds single-channel coverage.
	AtlasTextureKindMonochrome AtlasTextureKind = 0
	// AtlasTextureKindPolychrome holds colour pixels.
	AtlasTextureKindPolychrome AtlasTextureKind = 1
	// AtlasTextureKindSubpixel holds per-channel coverage.
	AtlasTextureKindSubpixel AtlasTextureKind = 2
)

// TileID is the id of a tile within its texture.
type TileID uint32

// ScapTarget is a screen-capture target of a ScapBackend (scap::Target): a *ScapDisplay or a *ScapWindow.
type ScapTarget interface {
	isScapTarget()
}

// ScapDisplay is a capturable display (scap::Display).
type ScapDisplay struct {
	// ID is the back end's display id.
	ID uint32
	// Title is the display's name.
	Title string
	// RawHandle is the HMONITOR on Windows and the X11 root window on Linux.
	RawHandle uintptr
	// Width is the display width in device pixels.
	Width uint16
	// Height is the display height in device pixels.
	Height uint16
	// XOffset is the display's x offset on the X11 screen (Linux).
	XOffset int16
	// YOffset is the display's y offset on the X11 screen (Linux).
	YOffset int16
}

// ScapWindow is a capturable window (scap::Window).
type ScapWindow struct {
	// ID is the back end's window id.
	ID uint32
	// Title is the window title.
	Title string
	// RawHandle is the HWND on Windows and the X11 window on Linux.
	RawHandle uintptr
}

func (*ScapDisplay) isScapTarget() {}
func (*ScapWindow) isScapTarget()  {}

// ScapFrameType is the requested output frame type (scap::frame::FrameType).
type ScapFrameType uint8

const (
	// ScapFrameTypeYUVFrame requests NV12 frames.
	ScapFrameTypeYUVFrame ScapFrameType = iota
	// ScapFrameTypeBGR0 requests BGR0 frames.
	ScapFrameTypeBGR0
	// ScapFrameTypeRGB requests RGB frames.
	ScapFrameTypeRGB
	// ScapFrameTypeBGRAFrame requests BGRA frames.
	ScapFrameTypeBGRAFrame
)

// ScapResolution is the requested output resolution (scap::capturer::Resolution).
type ScapResolution uint8

const (
	// ScapResolution480p is 640 pixels wide.
	ScapResolution480p ScapResolution = iota
	// ScapResolution720p is 1280 pixels wide.
	ScapResolution720p
	// ScapResolution1080p is 1920 pixels wide.
	ScapResolution1080p
	// ScapResolution1440p is 2560 pixels wide.
	ScapResolution1440p
	// ScapResolution2160p is 3840 pixels wide.
	ScapResolution2160p
	// ScapResolution4320p is 7680 pixels wide.
	ScapResolution4320p
	// ScapResolutionCaptured keeps the captured resolution.
	ScapResolutionCaptured
)

// ScapArea is a crop area in the target's coordinates (scap::capturer::Area).
type ScapArea struct {
	// X is the left edge.
	X float64
	// Y is the top edge.
	Y float64
	// Width is the area width.
	Width float64
	// Height is the area height.
	Height float64
}

// ScapCapturerOptions are the options passed to ScapBackend.BuildCapturer (scap::capturer::Options).
type ScapCapturerOptions struct {
	// FPS is the target frame rate.
	FPS uint32
	// ShowCursor draws the cursor into frames.
	ShowCursor bool
	// ShowHighlight highlights the captured target.
	ShowHighlight bool
	// Target is the target to capture; nil selects the default target (on Wayland, the portal prompt).
	Target ScapTarget
	// CropArea crops the frames.
	CropArea Option[ScapArea]
	// OutputType is the requested frame type; back ends may deliver another.
	OutputType ScapFrameType
	// OutputResolution is the requested output resolution.
	OutputResolution ScapResolution
	// ExcludedTargets are left out of the capture (macOS only upstream).
	ExcludedTargets []ScapTarget
}

// ScapFrame is a captured frame (scap::frame::Frame): one of the ScapFrame* types.
type ScapFrame interface {
	// Size returns the frame's width and height in pixels.
	Size() (width int32, height int32)
}

// ScapPixels is the payload of the packed-pixel frame variants.
type ScapPixels struct {
	// DisplayTime is the presentation timestamp reported by the back end.
	DisplayTime uint64
	// Width is the frame width in pixels.
	Width int32
	// Height is the frame height in pixels.
	Height int32
	// Data is the packed pixel data, rows without padding.
	Data []byte
}

// ScapFrameYUV is an NV12 frame (Frame::YUVFrame).
type ScapFrameYUV struct {
	// DisplayTime is the presentation timestamp reported by the back end.
	DisplayTime uint64
	// Width is the frame width in pixels.
	Width int32
	// Height is the frame height in pixels.
	Height int32
	// LuminanceBytes is the Y plane.
	LuminanceBytes []byte
	// LuminanceStride is the Y plane stride in bytes.
	LuminanceStride int32
	// ChrominanceBytes is the interleaved UV plane.
	ChrominanceBytes []byte
	// ChrominanceStride is the UV plane stride in bytes.
	ChrominanceStride int32
}

// ScapFrameRGB is an RGB frame (Frame::RGB).
type ScapFrameRGB struct{ ScapPixels }

// ScapFrameRGBx is an RGBx frame (Frame::RGBx).
type ScapFrameRGBx struct{ ScapPixels }

// ScapFrameXBGR is an XBGR frame (Frame::XBGR).
type ScapFrameXBGR struct{ ScapPixels }

// ScapFrameBGRx is a BGRx frame (Frame::BGRx).
type ScapFrameBGRx struct{ ScapPixels }

// ScapFrameBGR0 is a BGR0 frame (Frame::BGR0).
type ScapFrameBGR0 struct{ ScapPixels }

// ScapFrameBGRA is a BGRA frame (Frame::BGRA).
type ScapFrameBGRA struct{ ScapPixels }

// Size returns the frame's width and height in pixels.
func (f ScapFrameYUV) Size() (int32, int32) { return f.Width, f.Height }

// Size returns the frame's width and height in pixels.
func (p ScapPixels) Size() (int32, int32) { return p.Width, p.Height }

// ScapCapturer is a running or startable capture (scap::capturer::Capturer).
type ScapCapturer interface {
	StartCapture()
	StopCapture()
	GetNextFrame() (ScapFrame, error)
	Target() ScapTarget
}

// ScapBackend is the screen-capture engine the scap glue runs on (scap's free functions and
// Capturer::build). Linux and Windows back ends register one with RegisterScapBackend.
type ScapBackend interface {
	GetAllTargets() ([]ScapTarget, error)
	BuildCapturer(options ScapCapturerOptions) (ScapCapturer, error)
}

var scapBackendRegistry struct {
	mu      sync.Mutex
	backend ScapBackend
}

// RegisterScapBackend installs the process's screen-capture back end; call it from init.
func RegisterScapBackend(backend ScapBackend) {
	if backend == nil {
		panic("gpui: RegisterScapBackend called with a nil backend")
	}
	scapBackendRegistry.mu.Lock()
	defer scapBackendRegistry.mu.Unlock()
	if scapBackendRegistry.backend != nil {
		panic("gpui: RegisterScapBackend called twice")
	}
	scapBackendRegistry.backend = backend
}

func registeredScapBackend() (ScapBackend, bool) {
	scapBackendRegistry.mu.Lock()
	defer scapBackendRegistry.mu.Unlock()
	return scapBackendRegistry.backend, scapBackendRegistry.backend != nil
}
