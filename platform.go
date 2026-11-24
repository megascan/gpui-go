package gpui

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"iter"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"

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

// Range is a half-open range of offsets, upstream's std::ops::Range<usize> (D-RANGE).
type Range struct {
	// Start is the first offset in the range.
	Start int
	// End is one past the last offset in the range.
	End int
}

// PlatformInputHandler connects a window's focused InputHandler to the platform.
type PlatformInputHandler struct {
	cx      AsyncWindowContext
	handler InputHandler
	focusID Option[FocusID]
}

func (h *PlatformInputHandler) release() {
	if releaser, ok := h.handler.(interface{ Release() }); ok {
		releaser.Release()
	}
}

// NewPlatformInputHandler wraps handler so the platform can call it through cx.
func NewPlatformInputHandler(cx AsyncWindowContext, handler InputHandler) *PlatformInputHandler {
	return &PlatformInputHandler{
		cx:      cx,
		handler: handler,
		focusID: None[FocusID](),
	}
}

func (h *PlatformInputHandler) withFocus(focusID FocusID) *PlatformInputHandler {
	h.focusID = Some(focusID)
	return h
}

// FocusID identifies the input field registered through Window.HandleInput. The identity
// survives adapter replacement during repaint, so backends can detect a field change without
// restarting the keyboard session.
func (h *PlatformInputHandler) FocusID() (FocusID, bool) {
	return h.focusID.Get()
}

// SelectedTextRange returns the handler's selection, if any.
func (h *PlatformInputHandler) SelectedTextRange(ignoreDisabledInput bool) (UTF16Selection, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[UTF16Selection] {
		return OptionFrom(h.handler.SelectedTextRange(ignoreDisabledInput, window, cx))
	})
	if err != nil {
		return UTF16Selection{}, false
	}
	return result.Get()
}

// MarkedTextRange returns the handler's marked (composing) range, if any.
func (h *PlatformInputHandler) MarkedTextRange() (Range, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[Range] {
		return OptionFrom(h.handler.MarkedTextRange(window, cx))
	})
	if err != nil {
		return Range{}, false
	}
	return result.Get()
}

// TextForRange returns the text of the given UTF-16 range.
func (h *PlatformInputHandler) TextForRange(rangeUTF16 Range, adjusted *Option[Range]) (string, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[string] {
		return OptionFrom(h.handler.TextForRange(rangeUTF16, adjusted, window, cx))
	})
	if err != nil {
		return "", false
	}
	return result.Get()
}

// ReplaceTextInRange replaces the text in the given UTF-16 range.
func (h *PlatformInputHandler) ReplaceTextInRange(replacementRange Option[Range], text string) {
	_ = h.cx.Update(func(window *Window, cx *App) {
		h.handler.ReplaceTextInRange(replacementRange, text, window, cx)
	})
}

// ReplaceAndMarkTextInRange replaces the text in the given UTF-16 range and marks it as composing.
func (h *PlatformInputHandler) ReplaceAndMarkTextInRange(rangeUTF16 Option[Range], newText string, newSelectedRange Option[Range]) {
	_ = h.cx.Update(func(window *Window, cx *App) {
		h.handler.ReplaceAndMarkTextInRange(rangeUTF16, newText, newSelectedRange, window, cx)
	})
}

// UnmarkText ends the IME composition.
func (h *PlatformInputHandler) UnmarkText() {
	_ = h.cx.Update(func(window *Window, cx *App) {
		h.handler.UnmarkText(window, cx)
	})
}

// Paste inserts a platform-initiated paste.
func (h *PlatformInputHandler) Paste(item ClipboardItem) {
	_ = h.cx.Update(func(window *Window, cx *App) {
		h.handler.Paste(item, window, cx)
	})
}

// BoundsForRange returns the window bounds of the given UTF-16 range.
func (h *PlatformInputHandler) BoundsForRange(rangeUTF16 Range) (Bounds[Pixels], bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[Bounds[Pixels]] {
		return OptionFrom(h.handler.BoundsForRange(rangeUTF16, window, cx))
	})
	if err != nil {
		return Bounds[Pixels]{}, false
	}
	return result.Get()
}

// SelectionBoundsForRange returns the selection rectangles of the given UTF-16 range.
func (h *PlatformInputHandler) SelectionBoundsForRange(rangeUTF16 Range) []Bounds[Pixels] {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) []Bounds[Pixels] {
		return h.handler.SelectionBoundsForRange(rangeUTF16, window, cx)
	})
	if err != nil {
		return nil
	}
	return result
}

// ApplePressAndHoldEnabled reports whether the handler wants macOS press-and-hold.
func (h *PlatformInputHandler) ApplePressAndHoldEnabled() bool {
	return h.handler.ApplePressAndHoldEnabled()
}

func (h *PlatformInputHandler) unmarkTextDuringUpdate(window *Window, cx *App) {
	h.handler.UnmarkText(window, cx)
}

// DispatchInput inserts input at the current selection.
func (h *PlatformInputHandler) DispatchInput(input string, window *Window, cx *App) {
	h.handler.ReplaceTextInRange(None[Range](), input, window, cx)
}

// PlatformInputHandlerComputeIMECandidateBounds returns where the IME candidate window goes: the
// start of the composing range on the caret's visual line, or the selection endpoint.
func PlatformInputHandlerComputeIMECandidateBounds(markedRange Option[Range], selection *UTF16Selection, boundsForRange func(rangeUTF16 Range) (Bounds[Pixels], bool)) (Bounds[Pixels], bool) {
	if markedRange, ok := markedRange.Get(); ok {
		lineStart := markedRange.Start
		caret := selection.Range.End
		if caretBounds, ok := boundsForRange(Range{Start: caret, End: caret}); ok {
			for i := caret - 1; i >= markedRange.Start; i-- {
				if b, ok := boundsForRange(Range{Start: i, End: i}); ok {
					if (b.Origin.Y - caretBounds.Origin.Y).Abs() > Px(0.1) {
						lineStart = i + 1
						break
					}
				}
			}
		}
		return boundsForRange(Range{Start: lineStart, End: lineStart})
	}
	offset := selection.Range.End
	if selection.Reversed {
		offset = selection.Range.Start
	}
	return boundsForRange(Range{Start: offset, End: offset})
}

// SelectedBounds returns the IME candidate bounds from inside an update.
func (h *PlatformInputHandler) SelectedBounds(window *Window, cx *App) (Bounds[Pixels], bool) {
	markedRange := OptionFrom(h.handler.MarkedTextRange(window, cx))
	selection, ok := h.handler.SelectedTextRange(true, window, cx)
	if !ok {
		return Bounds[Pixels]{}, false
	}
	return PlatformInputHandlerComputeIMECandidateBounds(markedRange, &selection, func(rangeUTF16 Range) (Bounds[Pixels], bool) {
		return h.handler.BoundsForRange(rangeUTF16, window, cx)
	})
}

// IMECandidateBounds returns the IME candidate bounds from outside an update.
func (h *PlatformInputHandler) IMECandidateBounds() (Bounds[Pixels], bool) {
	markedRange := OptionFrom(h.MarkedTextRange())
	selection, ok := h.SelectedTextRange(true)
	if !ok {
		return Bounds[Pixels]{}, false
	}
	return PlatformInputHandlerComputeIMECandidateBounds(markedRange, &selection, h.BoundsForRange)
}

// CharacterIndexForPoint returns the UTF-16 offset of the character at point.
func (h *PlatformInputHandler) CharacterIndexForPoint(point Point[Pixels]) (int, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[int] {
		return OptionFrom(h.handler.CharacterIndexForPoint(point, window, cx))
	})
	if err != nil {
		return 0, false
	}
	return result.Get()
}

// SetSelectedTextRange sets the handler's selection; see InputHandler.SetSelectedTextRange.
func (h *PlatformInputHandler) SetSelectedTextRange(rangeUTF16 Range) {
	_ = h.cx.Update(func(window *Window, cx *App) {
		h.handler.SetSelectedTextRange(rangeUTF16, window, cx)
	})
}

// ElementBounds returns the focused element's bounds; see InputHandler.ElementBounds.
func (h *PlatformInputHandler) ElementBounds() (Bounds[Pixels], bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[Bounds[Pixels]] {
		return OptionFrom(h.handler.ElementBounds(window, cx))
	})
	if err != nil {
		return Bounds[Pixels]{}, false
	}
	return result.Get()
}

// TextLengthUTF16 returns the document length; see InputHandler.TextLengthUTF16.
func (h *PlatformInputHandler) TextLengthUTF16() (int, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[int] {
		return OptionFrom(h.handler.TextLengthUTF16(window, cx))
	})
	if err != nil {
		return 0, false
	}
	return result.Get()
}

// AcceptsTextInput reports whether the handler accepts text input, from inside an update.
func (h *PlatformInputHandler) AcceptsTextInput(window *Window, cx *App) bool {
	return h.handler.AcceptsTextInput(window, cx)
}

// QueryAcceptsTextInput reports whether the handler accepts text input; true when the window
// cannot be updated.
func (h *PlatformInputHandler) QueryAcceptsTextInput() bool {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) bool {
		return h.handler.AcceptsTextInput(window, cx)
	})
	if err != nil {
		return true
	}
	return result
}

// QueryAcceptsFocusedTextInput checks eligibility against the current focus, including before
// the next frame is drawn. It fails closed when the window cannot be updated and requires
// registration through Window.HandleInput.
func (h *PlatformInputHandler) QueryAcceptsFocusedTextInput() bool {
	focusID := h.focusID
	result, err := h.cx.UpdateR(func(window *Window, cx *App) bool {
		return focusID.IsSome() && window.focus == focusID && h.handler.AcceptsTextInput(window, cx)
	})
	if err != nil {
		return false
	}
	return result
}

// QueryPrefersIMEForPrintableKeys reports whether printable keys go to the IME first. While a
// multi-stroke binding is pending it returns false regardless of the handler's preference,
// because the next printable key may complete a binding whose prefix already bypassed the IME.
func (h *PlatformInputHandler) QueryPrefersIMEForPrintableKeys() bool {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) bool {
		return !window.HasPendingKeystrokes() && h.handler.PrefersIMEForPrintableKeys(window, cx)
	})
	if err != nil {
		return false
	}
	return result
}

// TextInputConfiguration returns the handler's text-assistance preferences.
func (h *PlatformInputHandler) TextInputConfiguration(window *Window, cx *App) TextInputConfiguration {
	return h.handler.TextInputConfiguration(window, cx)
}

// TextInputEditableRange returns the range platform text input may edit; see
// InputHandler.TextInputEditableRange.
func (h *PlatformInputHandler) TextInputEditableRange() (Range, bool) {
	result, err := h.cx.UpdateR(func(window *Window, cx *App) Option[Range] {
		return OptionFrom(h.handler.TextInputEditableRange(window, cx))
	})
	if err != nil {
		return Range{}, false
	}
	return result.Get()
}

// UTF16Selection is a selection in a text buffer, in UTF-16 code units. Unlike a Range, its head
// may be before its tail.
type UTF16Selection struct {
	// Range is the selected range of the document in UTF-16 code units.
	Range Range
	// Reversed reports whether the head of the selection is at the start (true) or end (false) of
	// the range.
	Reversed bool
}

// InputHandler is GPUI's interface for native text input, IME composition and selection.
// Platform backends translate their native text protocols into these operations. Offsets and
// ranges are UTF-16 code units unless stated otherwise. Embed InputHandlerDefaults for the
// provided methods and bind it with BindInputHandlerDefaults.
type InputHandler interface {
	// SelectedTextRange returns the selected range, if any (NSTextInputClient selectedRange).
	SelectedTextRange(ignoreDisabledInput bool, window *Window, cx *App) (UTF16Selection, bool)
	// MarkedTextRange returns the marked range, if any (NSTextInputClient markedRange).
	MarkedTextRange(window *Window, cx *App) (Range, bool)
	// TextForRange returns the text of the range (attributedSubstring(forProposedRange:actualRange:)).
	TextForRange(rangeUTF16 Range, adjustedRange *Option[Range], window *Window, cx *App) (string, bool)
	// ReplaceTextInRange replaces the range with text (insertText(_:replacementRange:)).
	ReplaceTextInRange(replacementRange Option[Range], text string, window *Window, cx *App)
	// ReplaceAndMarkTextInRange replaces the range and marks the new text as composing
	// (setMarkedText(_:selectedRange:replacementRange:)).
	ReplaceAndMarkTextInRange(rangeUTF16 Option[Range], newText string, newSelectedRange Option[Range], window *Window, cx *App)
	// UnmarkText removes the IME composing state (unmarkText()).
	UnmarkText(window *Window, cx *App)
	// Paste inserts a platform-initiated paste at the current selection; the default inserts
	// only the plain-text portion of the item.
	Paste(item ClipboardItem, window *Window, cx *App)
	// BoundsForRange returns the window bounds of the range, used to place the IME candidate
	// window (firstRect(forCharacterRange:actualRange:)).
	BoundsForRange(rangeUTF16 Range, window *Window, cx *App) (Bounds[Pixels], bool)
	// SelectionBoundsForRange returns selection rectangles in document order, as line or
	// shaped-run fragments; empty means native selection geometry is unavailable.
	SelectionBoundsForRange(rangeUTF16 Range, window *Window, cx *App) []Bounds[Pixels]
	// CharacterIndexForPoint returns the UTF-16 offset of the character at point (characterIndexForPoint:).
	CharacterIndexForPoint(point Point[Pixels], window *Window, cx *App) (int, bool)
	// SetSelectedTextRange sets the selection when the system text machinery moves it on the
	// application's behalf.
	SetSelectedTextRange(rangeUTF16 Range, window *Window, cx *App)
	// ElementBounds returns the focused text element's window bounds, if known.
	ElementBounds(window *Window, cx *App) (Bounds[Pixels], bool)
	// TextLengthUTF16 returns the document length in UTF-16 code units, if known.
	TextLengthUTF16(window *Window, cx *App) (int, bool)
	// ApplePressAndHoldEnabled opts the input context out of raw key repeats.
	ApplePressAndHoldEnabled() bool
	// AcceptsTextInput reports whether the handler accepts text input.
	AcceptsTextInput(window *Window, cx *App) bool
	// TextInputEditableRange returns the contiguous range platform text input may read and edit
	// around the selection; ok false places no bound.
	TextInputEditableRange(window *Window, cx *App) (Range, bool)
	// PrefersIMEForPrintableKeys reports whether printable keys go to the IME before keybinding
	// matching when a non-ASCII input source is active.
	PrefersIMEForPrintableKeys(window *Window, cx *App) bool
	// TextInputConfiguration returns the handler's text-assistance preferences; GPUI re-queries
	// it every frame, so it must be cheap.
	TextInputConfiguration(window *Window, cx *App) TextInputConfiguration
}

// InputHandlerDefaults carries the default bodies of InputHandler's provided methods (D-DEFAULTS).
type InputHandlerDefaults struct {
	self InputHandler
}

// BindInputHandlerDefaults makes the provided methods that call other InputHandler methods dispatch to self.
func BindInputHandlerDefaults(d *InputHandlerDefaults, self InputHandler) {
	d.self = self
}

func (d *InputHandlerDefaults) bound() InputHandler {
	if d.self == nil {
		panic(fmt.Sprintf(unboundDefaultsMessage, "InputHandlerDefaults", "BindInputHandlerDefaults"))
	}
	return d.self
}

// Paste inserts the plain-text portion of item at the current selection.
func (d *InputHandlerDefaults) Paste(item ClipboardItem, window *Window, cx *App) {
	if text, ok := item.Text(); ok {
		d.bound().ReplaceTextInRange(None[Range](), text, window, cx)
	}
}

// SelectionBoundsForRange returns no rectangles.
func (d *InputHandlerDefaults) SelectionBoundsForRange(rangeUTF16 Range, window *Window, cx *App) []Bounds[Pixels] {
	return nil
}

// SetSelectedTextRange ignores the request.
func (d *InputHandlerDefaults) SetSelectedTextRange(rangeUTF16 Range, window *Window, cx *App) {}

// ElementBounds reports no bounds.
func (d *InputHandlerDefaults) ElementBounds(window *Window, cx *App) (Bounds[Pixels], bool) {
	return Bounds[Pixels]{}, false
}

// TextLengthUTF16 reports no length.
func (d *InputHandlerDefaults) TextLengthUTF16(window *Window, cx *App) (int, bool) {
	return 0, false
}

// ApplePressAndHoldEnabled returns true.
func (d *InputHandlerDefaults) ApplePressAndHoldEnabled() bool {
	return true
}

// AcceptsTextInput returns true.
func (d *InputHandlerDefaults) AcceptsTextInput(window *Window, cx *App) bool {
	return true
}

// TextInputEditableRange places no bound.
func (d *InputHandlerDefaults) TextInputEditableRange(window *Window, cx *App) (Range, bool) {
	return Range{}, false
}

// PrefersIMEForPrintableKeys returns false.
func (d *InputHandlerDefaults) PrefersIMEForPrintableKeys(window *Window, cx *App) bool {
	return false
}

// TextInputConfiguration returns the default configuration.
func (d *InputHandlerDefaults) TextInputConfiguration(window *Window, cx *App) TextInputConfiguration {
	return TextInputConfiguration{}
}

// TextInputConfiguration holds the platform text-assistance preferences of the focused text
// region. The zero value disables all assistance and requests no particular action key.
type TextInputConfiguration struct {
	// Autocorrect reports whether the platform may automatically correct entered text.
	Autocorrect bool
	// Autocapitalize is how software keyboards capitalize entered text.
	Autocapitalize Autocapitalize
	// Suggestions reports whether software keyboards may offer word suggestions and spellcheck.
	Suggestions bool
	// InputAction is the action advertised on a software keyboard's confirm key.
	InputAction TextInputAction
}

// Autocapitalize is the automatic capitalization applied by software keyboards.
type Autocapitalize uint8

const (
	// AutocapitalizeNone applies no automatic capitalization.
	AutocapitalizeNone Autocapitalize = iota
	// AutocapitalizeWords capitalizes the first letter of each word.
	AutocapitalizeWords
	// AutocapitalizeSentences capitalizes the first letter of each sentence.
	AutocapitalizeSentences
	// AutocapitalizeCharacters capitalizes every letter.
	AutocapitalizeCharacters
)

// TextInputAction is the action a software keyboard advertises on its confirm key: the HTML
// enterkeyhint value set. It affects only how the key is presented.
type TextInputAction uint8

const (
	// TextInputActionUnspecified lets the platform choose its default presentation.
	TextInputActionUnspecified TextInputAction = iota
	// TextInputActionEnter is inserting a line break.
	TextInputActionEnter
	// TextInputActionDone is committing the field's value.
	TextInputActionDone
	// TextInputActionGo is navigating to the typed target.
	TextInputActionGo
	// TextInputActionNext is moving to the next field.
	TextInputActionNext
	// TextInputActionPrevious is moving to the previous field.
	TextInputActionPrevious
	// TextInputActionSearch is executing a search.
	TextInputActionSearch
	// TextInputActionSend is sending a message.
	TextInputActionSend
)

// WindowOptions are the variables that can be configured when creating a new window; start from
// DefaultWindowOptions.
type WindowOptions struct {
	// WindowBounds is the state and bounds of the window in screen coordinates; None inherits them.
	WindowBounds Option[WindowBounds]
	// Titlebar is the titlebar configuration of the window.
	Titlebar Option[TitlebarOptions]
	// Focus reports whether the window is focused when created.
	Focus bool
	// Show reports whether the window is shown when created.
	Show bool
	// Kind is the kind of window to create.
	Kind WindowKind
	// IsMovable reports whether the user can move the window; programmatic moves still work.
	IsMovable bool
	// AppOwnsTitlebarDrag reports whether the application, not AppKit, owns dragging of its
	// custom titlebar (macOS only).
	AppOwnsTitlebarDrag bool
	// InactiveFrameInterval is the minimum frame interval while inactive without sustained
	// high-rate input; None disables the limit.
	InactiveFrameInterval Option[time.Duration]
	// LowPowerFrameInterval is the minimum frame interval in Low Power Mode; None disables the limit.
	LowPowerFrameInterval Option[time.Duration]
	// ThermalFrameInterval is the minimum frame interval under serious or critical thermal
	// pressure; None disables the limit.
	ThermalFrameInterval Option[time.Duration]
	// IsResizable reports whether the user can resize the window.
	IsResizable bool
	// IsMinimizable reports whether the user can minimize the window.
	IsMinimizable bool
	// DisplayID is the display to create the window on.
	DisplayID Option[DisplayID]
	// WindowBackground is the appearance of the window background.
	WindowBackground WindowBackgroundAppearance
	// AppID is the application identifier desktop environments group windows by.
	AppID Option[string]
	// WindowMinSize is the window's minimum size.
	WindowMinSize Option[Size[Pixels]]
	// WindowDecorations requests client or server-side decorations on X11 and Wayland.
	WindowDecorations Option[WindowDecorations]
	// Icon is the window icon (X11 only).
	Icon Option[*image.NRGBA]
	// TabbingIdentifier groups windows as native tabs on macOS.
	TabbingIdentifier Option[string]
	// UseDisplayP3 renders the window in the Display P3 color space on macOS.
	UseDisplayP3 bool
}

// WindowParams are the variables the platform opens a window with.
type WindowParams struct {
	// Bounds is the window's bounds in screen coordinates.
	Bounds Bounds[Pixels]
	// Titlebar is the titlebar configuration of the window.
	Titlebar Option[TitlebarOptions]
	// Kind is the kind of window to create.
	Kind WindowKind
	// IsMovable reports whether the user can move the window.
	IsMovable bool
	// AppOwnsTitlebarDrag reports whether the application owns dragging of its titlebar (macOS only).
	AppOwnsTitlebarDrag bool
	// IsResizable reports whether the user can resize the window.
	IsResizable bool
	// IsMinimizable reports whether the user can minimize the window.
	IsMinimizable bool
	// Focus reports whether the window is focused when created.
	Focus bool
	// Show reports whether the window is shown when created.
	Show bool
	// Icon is the window icon (X11 only).
	Icon Option[*image.NRGBA]
	// DisplayID is the display to create the window on.
	DisplayID Option[DisplayID]
	// AppID is the application identifier of the window.
	AppID Option[string]
	// WindowMinSize is the window's minimum size.
	WindowMinSize Option[Size[Pixels]]
	// TabbingIdentifier groups windows as native tabs (macOS only).
	TabbingIdentifier Option[string]
	// UseDisplayP3 renders the window in the Display P3 color space (macOS only).
	UseDisplayP3 bool
}

// WindowBoundsKind is the variant of a WindowBounds.
type WindowBoundsKind uint8

const (
	// WindowBoundsKindWindowed opens the window windowed with the given bounds.
	WindowBoundsKindWindowed WindowBoundsKind = iota
	// WindowBoundsKindMaximized opens the window maximized; the bounds are its restore size.
	WindowBoundsKindMaximized
	// WindowBoundsKindFullscreen opens the window fullscreen; the bounds are its restore size.
	WindowBoundsKindFullscreen
)

// WindowBounds is how a window should be opened. The zero value is upstream's default,
// Windowed with zero bounds.
type WindowBounds struct {
	kind   WindowBoundsKind
	bounds Bounds[Pixels]
}

// WindowBoundsWindowed opens the window windowed with the given bounds.
func WindowBoundsWindowed(bounds Bounds[Pixels]) WindowBounds {
	return WindowBounds{kind: WindowBoundsKindWindowed, bounds: bounds}
}

// WindowBoundsMaximized opens the window maximized; bounds is its restore size.
func WindowBoundsMaximized(bounds Bounds[Pixels]) WindowBounds {
	return WindowBounds{kind: WindowBoundsKindMaximized, bounds: bounds}
}

// WindowBoundsFullscreen opens the window fullscreen; bounds is its restore size.
func WindowBoundsFullscreen(bounds Bounds[Pixels]) WindowBounds {
	return WindowBounds{kind: WindowBoundsKindFullscreen, bounds: bounds}
}

// Kind returns the variant.
func (b WindowBounds) Kind() WindowBoundsKind {
	return b.kind
}

// GetBounds returns the inner bounds.
func (b WindowBounds) GetBounds() Bounds[Pixels] {
	return b.bounds
}

// WindowBoundsCentered creates window bounds centered on the given display, or the primary
// display if none is provided.
func WindowBoundsCentered(displayID Option[DisplayID], size Size[Pixels], cx *App) WindowBounds {
	return WindowBoundsWindowed(BoundsCentered(displayID, size, cx))
}

// DefaultWindowOptions returns upstream's default window options.
func DefaultWindowOptions() WindowOptions {
	return WindowOptions{
		WindowBounds:          None[WindowBounds](),
		Titlebar:              Some(TitlebarOptions{}),
		Focus:                 true,
		Show:                  true,
		Kind:                  WindowKindNormal{},
		IsMovable:             true,
		AppOwnsTitlebarDrag:   false,
		InactiveFrameInterval: Some(33_333 * time.Microsecond),
		LowPowerFrameInterval: Some(33_333 * time.Microsecond),
		ThermalFrameInterval:  Some(16_667 * time.Microsecond),
		IsResizable:           true,
		IsMinimizable:         true,
		DisplayID:             None[DisplayID](),
		WindowBackground:      WindowBackgroundAppearanceOpaque,
		Icon:                  None[*image.NRGBA](),
		AppID:                 None[string](),
		WindowMinSize:         None[Size[Pixels]](),
		WindowDecorations:     None[WindowDecorations](),
		TabbingIdentifier:     None[string](),
		UseDisplayP3:          false,
	}
}

// TitlebarOptions are the options of a window's titlebar.
type TitlebarOptions struct {
	// Title is the initial title of the window.
	Title Option[SharedString]
	// AppearsTransparent hides the system titlebar for a custom-drawn one (macOS and Windows
	// only; see WindowOptions.WindowDecorations on Linux).
	AppearsTransparent bool
	// TrafficLightPosition is the position of the macOS traffic light buttons.
	TrafficLightPosition Option[Point[Pixels]]
}

// WindowKind is the kind of window to create: WindowKindNormal, WindowKindPopUp,
// WindowKindAnchoredPopup, WindowKindFloating, WindowKindLayerShell or WindowKindDialog.
type WindowKind interface {
	isWindowKind()
}

// WindowKindNormal is a normal application window.
type WindowKindNormal struct{}

// WindowKindPopUp is a window above all other windows, for alerts or popups; use sparingly.
type WindowKindPopUp struct{}

// WindowKindAnchoredPopup is a parent-anchored, platform-native popup window for menus,
// comboboxes, context menus and tooltips. Its size comes from WindowOptions.WindowBounds, whose
// origin is ignored. Platforms without a native implementation reject it with
// PopupNotSupportedError.
type WindowKindAnchoredPopup struct {
	PopupOptions
}

// WindowKindFloating is a window that floats on top of its parent window.
type WindowKindFloating struct{}

// WindowKindLayerShell is a Wayland layer-shell window for docks, notifications or wallpapers
// (Linux Wayland only).
type WindowKindLayerShell struct {
	LayerShellOptions
}

// WindowKindDialog is a window on top of its parent that blocks interaction with it until closed.
type WindowKindDialog struct{}

func (WindowKindNormal) isWindowKind()        {}
func (WindowKindPopUp) isWindowKind()         {}
func (WindowKindAnchoredPopup) isWindowKind() {}
func (WindowKindFloating) isWindowKind()      {}
func (WindowKindLayerShell) isWindowKind()    {}
func (WindowKindDialog) isWindowKind()        {}

// WindowAppearance is the appearance of the window, as defined by the operating system (on
// macOS, the named NSAppearance values).
type WindowAppearance uint8

const (
	// WindowAppearanceLight is a light appearance (macOS aqua).
	WindowAppearanceLight WindowAppearance = iota
	// WindowAppearanceVibrantLight is a light appearance with vibrant colors (NSAppearanceNameVibrantLight).
	WindowAppearanceVibrantLight
	// WindowAppearanceDark is a dark appearance (macOS darkAqua).
	WindowAppearanceDark
	// WindowAppearanceVibrantDark is a dark appearance with vibrant colors (NSAppearanceNameVibrantDark).
	WindowAppearanceVibrantDark
)

// WindowBackgroundAppearance is the appearance of the window's background where there is no
// content or the content is transparent.
type WindowBackgroundAppearance uint8

const (
	// WindowBackgroundAppearanceOpaque tells the window manager that content behind the window
	// need not be drawn; themes should define a fully opaque background color.
	WindowBackgroundAppearanceOpaque WindowBackgroundAppearance = iota
	// WindowBackgroundAppearanceTransparent is plain alpha transparency.
	WindowBackgroundAppearanceTransparent
	// WindowBackgroundAppearanceBlurred is transparency with the content behind blurred; not
	// always supported.
	WindowBackgroundAppearanceBlurred
	// WindowBackgroundAppearanceMicaBackdrop is the Mica backdrop material (Windows 11).
	WindowBackgroundAppearanceMicaBackdrop
	// WindowBackgroundAppearanceMicaAltBackdrop is the Mica Alt backdrop material (Windows 11).
	WindowBackgroundAppearanceMicaAltBackdrop
)

// TextRenderingMode is the text rendering mode used to draw glyphs.
type TextRenderingMode uint8

const (
	// TextRenderingModePlatformDefault uses the platform's default mode.
	TextRenderingModePlatformDefault TextRenderingMode = iota
	// TextRenderingModeSubpixel uses subpixel (ClearType-style) rendering.
	TextRenderingModeSubpixel
	// TextRenderingModeGrayscale uses grayscale rendering.
	TextRenderingModeGrayscale
)

// PathPromptOptions are the options of a file dialog prompt.
type PathPromptOptions struct {
	// Files reports whether files may be selected.
	Files bool
	// Directories reports whether directories may be selected.
	Directories bool
	// Multiple reports whether several paths may be selected.
	Multiple bool
	// Prompt is the prompt shown to the user.
	Prompt Option[SharedString]
}

// PromptLevel is the styling of a prompt.
type PromptLevel uint8

const (
	// PromptLevelInfo notifies the user of something.
	PromptLevelInfo PromptLevel = iota
	// PromptLevelWarning warns the user of a potential problem.
	PromptLevelWarning
	// PromptLevelCritical reports a critical problem.
	PromptLevelCritical
)

// PromptButtonKind is the variant of a PromptButton.
type PromptButtonKind uint8

const (
	// PromptButtonKindOk is an Ok button.
	PromptButtonKindOk PromptButtonKind = iota
	// PromptButtonKindCancel is a Cancel button.
	PromptButtonKindCancel
	// PromptButtonKindOther is any other button.
	PromptButtonKindOther
)

// PromptButton is a button of a prompt. It is comparable.
type PromptButton struct {
	kind  PromptButtonKind
	label SharedString
}

// NewPromptButton creates a button with the given label.
func NewPromptButton(label string) PromptButton {
	return PromptButton{kind: PromptButtonKindOther, label: SharedString(label)}
}

// PromptButtonOk creates an Ok button.
func PromptButtonOk(label string) PromptButton {
	return PromptButton{kind: PromptButtonKindOk, label: SharedString(label)}
}

// PromptButtonCancel creates a Cancel button.
func PromptButtonCancel(label string) PromptButton {
	return PromptButton{kind: PromptButtonKindCancel, label: SharedString(label)}
}

// PromptButtonFromStr maps "ok" and "cancel" (any case) to the Ok and Cancel buttons and any
// other value to a button with that label.
func PromptButtonFromStr(value string) PromptButton {
	switch strings.ToLower(value) {
	case "ok":
		return PromptButtonOk("OK")
	case "cancel":
		return PromptButtonCancel("Cancel")
	default:
		return NewPromptButton(value)
	}
}

// Kind returns the variant.
func (b PromptButton) Kind() PromptButtonKind {
	return b.kind
}

// IsCancel reports whether this is a cancel button.
func (b PromptButton) IsCancel() bool {
	return b.kind == PromptButtonKindCancel
}

// Label returns the label of the button.
func (b PromptButton) Label() SharedString {
	return b.label
}

// CursorStyle is the style of the mouse cursor.
type CursorStyle uint8

const (
	// CursorStyleArrow is the default cursor.
	CursorStyleArrow CursorStyle = iota
	// CursorStyleIBeam is a text input cursor (CSS text).
	CursorStyleIBeam
	// CursorStyleCrosshair is a crosshair cursor (CSS crosshair).
	CursorStyleCrosshair
	// CursorStyleClosedHand is a closed hand cursor (CSS grabbing).
	CursorStyleClosedHand
	// CursorStyleOpenHand is an open hand cursor (CSS grab).
	CursorStyleOpenHand
	// CursorStylePointingHand is a pointing hand cursor (CSS pointer).
	CursorStylePointingHand
	// CursorStyleResizeLeft is a resize left cursor (CSS w-resize).
	CursorStyleResizeLeft
	// CursorStyleResizeRight is a resize right cursor (CSS e-resize).
	CursorStyleResizeRight
	// CursorStyleResizeLeftRight is a left and right resize cursor (CSS ew-resize).
	CursorStyleResizeLeftRight
	// CursorStyleResizeUp is a resize up cursor (CSS n-resize).
	CursorStyleResizeUp
	// CursorStyleResizeDown is a resize down cursor (CSS s-resize).
	CursorStyleResizeDown
	// CursorStyleResizeUpDown is an up and down resize cursor (CSS ns-resize).
	CursorStyleResizeUpDown
	// CursorStyleResizeUpLeftDownRight is an up-left and down-right resize cursor (CSS nesw-resize).
	CursorStyleResizeUpLeftDownRight
	// CursorStyleResizeUpRightDownLeft is an up-right and down-left resize cursor (CSS nwse-resize).
	CursorStyleResizeUpRightDownLeft
	// CursorStyleResizeColumn resizes a column horizontally (CSS col-resize).
	CursorStyleResizeColumn
	// CursorStyleResizeRow resizes a row vertically (CSS row-resize).
	CursorStyleResizeRow
	// CursorStyleIBeamCursorForVerticalLayout is a text input cursor for vertical layout (CSS vertical-text).
	CursorStyleIBeamCursorForVerticalLayout
	// CursorStyleOperationNotAllowed shows that the operation is not allowed (CSS not-allowed).
	CursorStyleOperationNotAllowed
	// CursorStyleDragLink shows that the operation will create a link (CSS alias).
	CursorStyleDragLink
	// CursorStyleDragCopy shows that the operation will copy (CSS copy).
	CursorStyleDragCopy
	// CursorStyleContextualMenu shows that the operation opens a context menu (CSS context-menu).
	CursorStyleContextualMenu
)

var cursorStyleNames = [...]string{
	"Arrow",
	"IBeam",
	"Crosshair",
	"ClosedHand",
	"OpenHand",
	"PointingHand",
	"ResizeLeft",
	"ResizeRight",
	"ResizeLeftRight",
	"ResizeUp",
	"ResizeDown",
	"ResizeUpDown",
	"ResizeUpLeftDownRight",
	"ResizeUpRightDownLeft",
	"ResizeColumn",
	"ResizeRow",
	"IBeamCursorForVerticalLayout",
	"OperationNotAllowed",
	"DragLink",
	"DragCopy",
	"ContextualMenu",
}

// CursorStyleIter iterates over every CursorStyle in declaration order (strum EnumIter).
func CursorStyleIter() iter.Seq[CursorStyle] {
	return func(yield func(CursorStyle) bool) {
		for i := range cursorStyleNames {
			if !yield(CursorStyle(i)) {
				return
			}
		}
	}
}

// String formats the style like its Rust Debug output.
func (c CursorStyle) String() string {
	if int(c) < len(cursorStyleNames) {
		return cursorStyleNames[c]
	}
	return fmt.Sprintf("CursorStyle(%d)", uint8(c))
}

// MarshalJSONTo encodes the style as its variant name.
func (c CursorStyle) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(c.String()))
}

// UnmarshalJSONFrom decodes the style from its variant name.
func (c *CursorStyle) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '"' {
		return fmt.Errorf("invalid type: %v, expected variant identifier", tok.Kind())
	}
	s := tok.String()
	for i, name := range cursorStyleNames {
		if name == s {
			*c = CursorStyle(i)
			return nil
		}
	}
	return fmt.Errorf("unknown variant `%s`, expected one of %s", s, cursorStyleExpected())
}

func cursorStyleExpected() string {
	quoted := make([]string, len(cursorStyleNames))
	for i, name := range cursorStyleNames {
		quoted[i] = "`" + name + "`"
	}
	return strings.Join(quoted, ", ")
}

// JSONSchemaEnum lists the variant names for the JSON schema.
func (CursorStyle) JSONSchemaEnum() []string {
	return slices.Clone(cursorStyleNames[:])
}

// ClipboardItem is an item that is copied to or read from the clipboard.
type ClipboardItem struct {
	// Entries are the entries of this clipboard item.
	Entries []ClipboardEntry
}

// ClipboardReadErrorKind is the variant of a ClipboardReadError.
type ClipboardReadErrorKind uint8

const (
	// ClipboardReadErrorKindUnavailable means the platform clipboard is not available here.
	ClipboardReadErrorKindUnavailable ClipboardReadErrorKind = iota
	// ClipboardReadErrorKindDenied means the platform refused access.
	ClipboardReadErrorKindDenied
	// ClipboardReadErrorKindUnsupportedContent means the contents could not be converted.
	ClipboardReadErrorKindUnsupportedContent
)

// ClipboardReadError is an error of Platform.ReadFromClipboardAsync. Its variants call for
// different user-facing guidance. It is comparable.
type ClipboardReadError struct {
	kind    ClipboardReadErrorKind
	message string
}

// ClipboardReadErrorUnavailable is returned when the platform clipboard is not available, e.g.
// the browser has no async clipboard API or the page is not a secure context.
func ClipboardReadErrorUnavailable() ClipboardReadError {
	return ClipboardReadError{kind: ClipboardReadErrorKindUnavailable}
}

// ClipboardReadErrorDenied is returned when the platform refused access, e.g. the user declined
// the browser's clipboard permission prompt.
func ClipboardReadErrorDenied(message string) ClipboardReadError {
	return ClipboardReadError{kind: ClipboardReadErrorKindDenied, message: message}
}

// ClipboardReadErrorUnsupportedContent is returned when the contents could not be converted
// into a ClipboardItem.
func ClipboardReadErrorUnsupportedContent() ClipboardReadError {
	return ClipboardReadError{kind: ClipboardReadErrorKindUnsupportedContent}
}

// Kind returns the variant.
func (e ClipboardReadError) Kind() ClipboardReadErrorKind {
	return e.kind
}

// Message returns the platform's message of a Denied error.
func (e ClipboardReadError) Message() string {
	return e.message
}

// Error formats the error as upstream's Display.
func (e ClipboardReadError) Error() string {
	switch e.kind {
	case ClipboardReadErrorKindDenied:
		return "clipboard access was denied: " + e.message
	case ClipboardReadErrorKindUnsupportedContent:
		return "the clipboard contents are unsupported"
	default:
		return "the clipboard is unavailable"
	}
}

// ClipboardEntry is one entry of a ClipboardItem: ClipboardEntryString, ClipboardEntryImage or
// ClipboardEntryExternalPaths.
type ClipboardEntry interface {
	isClipboardEntry()
}

// ClipboardEntryString is a string entry.
type ClipboardEntryString struct {
	ClipboardString
}

// ClipboardEntryImage is an image entry.
type ClipboardEntryImage struct {
	Image
}

// ClipboardEntryExternalPaths is a file entry.
type ClipboardEntryExternalPaths struct {
	ExternalPaths
}

func (ClipboardEntryString) isClipboardEntry()        {}
func (ClipboardEntryImage) isClipboardEntry()         {}
func (ClipboardEntryExternalPaths) isClipboardEntry() {}

// NewClipboardItemString creates a string item with no metadata.
func NewClipboardItemString(text string) ClipboardItem {
	return ClipboardItem{Entries: []ClipboardEntry{ClipboardEntryString{NewClipboardString(text)}}}
}

// NewClipboardItemStringWithMetadata creates a string item with the given metadata.
func NewClipboardItemStringWithMetadata(text string, metadata string) ClipboardItem {
	return ClipboardItem{Entries: []ClipboardEntry{ClipboardEntryString{ClipboardString{
		Text:     text,
		Metadata: Some(metadata),
	}}}}
}

// NewClipboardItemStringWithJSONMetadata creates a string item whose metadata is metadata
// serialized as JSON.
func NewClipboardItemStringWithJSONMetadata[T any](text string, metadata T) ClipboardItem {
	return ClipboardItem{Entries: []ClipboardEntry{ClipboardEntryString{
		NewClipboardString(text).WithJSONMetadata(metadata),
	}}}
}

// NewClipboardItemImage creates an image item with no metadata.
func NewClipboardItemImage(img *Image) ClipboardItem {
	return ClipboardItem{Entries: []ClipboardEntry{ClipboardEntryImage{img.clone()}}}
}

// Text concatenates the item's string entries, or, without any, the display forms of its
// external paths; ok is false when the result is empty.
func (c ClipboardItem) Text() (string, bool) {
	var answer strings.Builder
	for _, entry := range c.Entries {
		if s, ok := entry.(ClipboardEntryString); ok {
			answer.WriteString(s.Text)
		}
	}
	if answer.Len() == 0 {
		for _, entry := range c.Entries {
			if paths, ok := entry.(ClipboardEntryExternalPaths); ok {
				for _, path := range paths.ExternalPaths {
					answer.WriteString(path)
				}
			}
		}
	}
	if answer.Len() != 0 {
		return answer.String(), true
	}
	return "", false
}

// Metadata returns the metadata of an item that is exactly one string entry.
func (c ClipboardItem) Metadata() (string, bool) {
	if len(c.Entries) == 1 {
		if s, ok := c.Entries[0].(ClipboardEntryString); ok {
			return s.Metadata.Get()
		}
	}
	return "", false
}

// IntoEntries iterates over the item's entries.
func (c ClipboardItem) IntoEntries() iter.Seq[ClipboardEntry] {
	return slices.Values(c.Entries)
}

// ClipboardEntryFromClipboardString wraps a ClipboardString as an entry.
func ClipboardEntryFromClipboardString(value ClipboardString) ClipboardEntry {
	return ClipboardEntryString{value}
}

// ClipboardEntryFromString wraps text as a string entry with no metadata.
func ClipboardEntryFromString(value string) ClipboardEntry {
	return ClipboardEntryFromClipboardString(ClipboardStringFromString(value))
}

// ClipboardEntryFromImage wraps an image as an entry.
func ClipboardEntryFromImage(value Image) ClipboardEntry {
	return ClipboardEntryImage{value}
}

// ClipboardItemFromEntry creates an item holding one entry.
func ClipboardItemFromEntry(value ClipboardEntry) ClipboardItem {
	return ClipboardItem{Entries: []ClipboardEntry{value}}
}

// ClipboardItemFromString creates an item holding one string entry.
func ClipboardItemFromString(value string) ClipboardItem {
	return ClipboardItemFromEntry(ClipboardEntryFromString(value))
}

// ClipboardItemFromImage creates an item holding one image entry.
func ClipboardItemFromImage(value Image) ClipboardItem {
	return ClipboardItemFromEntry(ClipboardEntryFromImage(value))
}

// ImageFormat is one of the supported image formats, used for images on the clipboard. The
// variants are sorted from most to least likely to be pasted into an editor, which matters when
// iterating over them to match clipboard content.
type ImageFormat uint8

const (
	// ImageFormatPng is .png.
	ImageFormatPng ImageFormat = iota
	// ImageFormatJpeg is .jpeg or .jpg.
	ImageFormatJpeg
	// ImageFormatWebp is .webp.
	ImageFormatWebp
	// ImageFormatGif is .gif.
	ImageFormatGif
	// ImageFormatSvg is .svg.
	ImageFormatSvg
	// ImageFormatBmp is .bmp.
	ImageFormatBmp
	// ImageFormatTiff is .tif or .tiff.
	ImageFormatTiff
	// ImageFormatIco is .ico.
	ImageFormatIco
	// ImageFormatPnm is the Netpbm formats (.pbm, .ppm, .pgm).
	ImageFormatPnm
)

const imageFormatCount = int(ImageFormatPnm) + 1

// ImageFormatIter iterates over every ImageFormat in declaration order (strum EnumIter).
func ImageFormatIter() iter.Seq[ImageFormat] {
	return func(yield func(ImageFormat) bool) {
		for i := range imageFormatCount {
			if !yield(ImageFormat(i)) {
				return
			}
		}
	}
}

// String formats the format like its Rust Debug output.
func (f ImageFormat) String() string {
	switch f {
	case ImageFormatPng:
		return "Png"
	case ImageFormatJpeg:
		return "Jpeg"
	case ImageFormatWebp:
		return "Webp"
	case ImageFormatGif:
		return "Gif"
	case ImageFormatSvg:
		return "Svg"
	case ImageFormatBmp:
		return "Bmp"
	case ImageFormatTiff:
		return "Tiff"
	case ImageFormatIco:
		return "Ico"
	case ImageFormatPnm:
		return "Pnm"
	}
	return fmt.Sprintf("ImageFormat(%d)", uint8(f))
}

// MimeType returns the mime type of the format.
func (f ImageFormat) MimeType() string {
	switch f {
	case ImageFormatPng:
		return "image/png"
	case ImageFormatJpeg:
		return "image/jpeg"
	case ImageFormatWebp:
		return "image/webp"
	case ImageFormatGif:
		return "image/gif"
	case ImageFormatSvg:
		return "image/svg+xml"
	case ImageFormatBmp:
		return "image/bmp"
	case ImageFormatTiff:
		return "image/tiff"
	case ImageFormatIco:
		return "image/ico"
	case ImageFormatPnm:
		return "image/x-portable-anymap"
	}
	panic(fmt.Sprintf("gpui: invalid ImageFormat %d", uint8(f)))
}

// Extension returns the file extension of the format, without the leading dot.
func (f ImageFormat) Extension() string {
	switch f {
	case ImageFormatPng:
		return "png"
	case ImageFormatJpeg:
		return "jpg"
	case ImageFormatWebp:
		return "webp"
	case ImageFormatGif:
		return "gif"
	case ImageFormatSvg:
		return "svg"
	case ImageFormatBmp:
		return "bmp"
	case ImageFormatTiff:
		return "tiff"
	case ImageFormatIco:
		return "ico"
	case ImageFormatPnm:
		return "pnm"
	}
	panic(fmt.Sprintf("gpui: invalid ImageFormat %d", uint8(f)))
}

// ImageFormatFromMimeType returns the format of the given mime type, including known aliases.
func ImageFormatFromMimeType(mimeType string) (ImageFormat, bool) {
	for format := range ImageFormatIter() {
		if format.MimeType() == mimeType {
			return format, true
		}
	}
	return imageFormatFromMimeTypeAlias(mimeType)
}

func imageFormatFromMimeTypeAlias(mimeType string) (ImageFormat, bool) {
	switch mimeType {
	case "image/jpg":
		return ImageFormatJpeg, true
	case "image/tif":
		return ImageFormatTiff, true
	}
	return 0, false
}

// Image is an image with a format and its encoded bytes.
type Image struct {
	// Format is the format the bytes are encoded in.
	Format ImageFormat
	// Bytes is the encoded image.
	Bytes []byte
	// ID is the unique ID of the image.
	ID uint64
}

// ImageFrame is one decoded image frame (image::Frame): a BGRA8 pixel buffer, its offset and
// its delay from the previous frame.
type ImageFrame struct {
	// Buffer holds the pixels, four bytes per pixel in B, G, R, A order.
	Buffer *image.RGBA
	// Left is the frame's x offset.
	Left uint32
	// Top is the frame's y offset.
	Top uint32
	// Delay is the delay of this frame from the previous one.
	Delay time.Duration
}

func newImageFrame(buffer *image.RGBA) ImageFrame {
	return ImageFrame{Buffer: buffer}
}

func (img *Image) clone() Image {
	return Image{Format: img.Format, Bytes: slices.Clone(img.Bytes), ID: img.ID}
}

type imageDecoder interface {
	orientation() (imageOrientation, error)
	decode() (image.Image, error)
}

func decodeStaticImage(bytes []byte, format ImageFormat) ([]ImageFrame, error) {
	decoder, err := newImageDecoder(bytes, format)
	if err != nil {
		return nil, fmt.Errorf("creating image decoder: %w", err)
	}
	return decodeStaticImageFromDecoder(decoder)
}

func decodeStaticImageFromDecoder(decoder imageDecoder) ([]ImageFrame, error) {
	orientation, err := decoder.orientation()
	if err != nil {
		return nil, fmt.Errorf("reading decoder's orientation: %w", err)
	}
	decoded, err := decoder.decode()
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	data := applyImageOrientation(imageIntoRGBA8(decoded), orientation)
	swapRedBlue(data.Pix)
	return []ImageFrame{newImageFrame(data)}, nil
}

func decodeAnimatedFrames(frames iter.Seq2[ImageFrame, error], formatName string) ([]ImageFrame, error) {
	var decoded []ImageFrame
	for frame, err := range frames {
		if err != nil {
			slog.Debug(fmt.Sprintf("Skipping %s frame due to decode error: %v", formatName, err))
			continue
		}
		swapRedBlue(frame.Buffer.Pix)
		decoded = append(decoded, frame)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("%s could not be decoded: all frames failed", formatName)
	}
	return decoded, nil
}

func decodeGifImage(bytes []byte) ([]ImageFrame, error) {
	frames, err := newGifFrames(bytes)
	if err != nil {
		return nil, err
	}
	return decodeAnimatedFrames(frames, "GIF")
}

func decodeWebpImage(bytes []byte) ([]ImageFrame, error) {
	decoder, err := newWebpDecoder(bytes)
	if err != nil {
		return nil, err
	}
	if decoder.hasAnimation() {
		return decodeAnimatedFrames(decoder.frames(), "WebP")
	}
	return decodeStaticImageFromDecoder(decoder)
}

// ImageEmpty returns an empty image containing no data.
func ImageEmpty() Image {
	return ImageFromBytes(ImageFormatPng, nil)
}

// ImageFromBytes creates an image from a format and its encoded bytes.
func ImageFromBytes(format ImageFormat, bytes []byte) Image {
	return Image{
		ID:     Hash(bytes),
		Format: format,
		Bytes:  bytes,
	}
}

// UseRenderImage makes the image renderable through the use_asset API; nil while it loads or
// when it failed.
func (img *Image) UseRenderImage(window *Window, cx *App) *RenderImage {
	result, ok, err := imageSourceUseData(ImageSourceImage{Image: img}, None[AnyImageCache](), window, cx)
	if !ok || err != nil {
		return nil
	}
	return result
}

// GetRenderImage makes the image renderable through the get_asset API; nil while it loads or
// when it failed.
func (img *Image) GetRenderImage(window *Window, cx *App) *RenderImage {
	result, ok, err := imageSourceGetData(ImageSourceImage{Image: img}, None[AnyImageCache](), window, cx)
	if !ok || err != nil {
		return nil
	}
	return result
}

// RemoveAsset drops the image from the asset cache, if possible.
func (img *Image) RemoveAsset(cx *App) {
	imageSourceRemoveAsset(ImageSourceImage{Image: img}, cx)
}

// IsAssetCached reports whether the image is in the asset cache (loading or loaded), without
// fetching it.
func (img *Image) IsAssetCached(cx *App) bool {
	return imageSourceIsAssetCached(ImageSourceImage{Image: img}, cx)
}

// ToImageData decodes the image into BGRA frames.
func (img *Image) ToImageData(svgRenderer SvgRenderer) (*RenderImage, error) {
	var frames []ImageFrame
	var err error
	switch img.Format {
	case ImageFormatGif:
		frames, err = decodeGifImage(img.Bytes)
	case ImageFormatPng:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatPng)
	case ImageFormatJpeg:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatJpeg)
	case ImageFormatWebp:
		frames, err = decodeWebpImage(img.Bytes)
	case ImageFormatBmp:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatBmp)
	case ImageFormatTiff:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatTiff)
	case ImageFormatIco:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatIco)
	case ImageFormatSvg:
		return svgRenderer.RenderSingleFrame(img.Bytes, 1.0)
	case ImageFormatPnm:
		frames, err = decodeStaticImage(img.Bytes, ImageFormatPnm)
	default:
		return nil, fmt.Errorf("unsupported image format %v", img.Format)
	}
	if err != nil {
		return nil, err
	}
	return NewRenderImage(frames), nil
}

// ClipboardString is a string on the clipboard with optional metadata.
type ClipboardString struct {
	// Text is the text content.
	Text string
	// Metadata is optional metadata associated with the string.
	Metadata Option[string]
}

// NewClipboardString creates a clipboard string with the given text and no metadata.
func NewClipboardString(text string) ClipboardString {
	return ClipboardString{Text: text, Metadata: None[string]()}
}

// WithJSONMetadata returns s with its metadata replaced by metadata serialized as JSON; it
// panics when metadata cannot be serialized.
func (s ClipboardString) WithJSONMetadata[T any](metadata T) ClipboardString {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	s.Metadata = Some(string(encoded))
	return s
}

// IntoText returns the text of the clipboard string.
func (s ClipboardString) IntoText() string {
	return s.Text
}

// MetadataJSON decodes the metadata as JSON into T.
func (s ClipboardString) MetadataJSON[T any]() (T, bool) {
	var value T
	metadata, ok := s.Metadata.Get()
	if !ok {
		return value, false
	}
	if err := json.Unmarshal([]byte(metadata), &value); err != nil {
		var zero T
		return zero, false
	}
	return value, true
}

// ClipboardStringTextHash hashes text for clipboard change detection.
func ClipboardStringTextHash(text string) uint64 {
	hasher := newSeaHasher()
	hasher.write([]byte(text))
	hasher.write([]byte{0xff})
	return hasher.finish()
}

// ClipboardStringFromString creates a clipboard string with no metadata.
func ClipboardStringFromString(value string) ClipboardString {
	return ClipboardString{Text: value, Metadata: None[string]()}
}

type imageOrientation uint8

const (
	imageOrientationNoTransforms imageOrientation = iota
	imageOrientationRotate90
	imageOrientationRotate180
	imageOrientationRotate270
	imageOrientationFlipHorizontal
	imageOrientationFlipVertical
	imageOrientationRotate90FlipH
	imageOrientationRotate270FlipH
)

func imageOrientationFromExif(exifOrientation uint8) (imageOrientation, bool) {
	switch exifOrientation {
	case 1:
		return imageOrientationNoTransforms, true
	case 2:
		return imageOrientationFlipHorizontal, true
	case 3:
		return imageOrientationRotate180, true
	case 4:
		return imageOrientationFlipVertical, true
	case 5:
		return imageOrientationRotate90FlipH, true
	case 6:
		return imageOrientationRotate90, true
	case 7:
		return imageOrientationRotate270FlipH, true
	case 8:
		return imageOrientationRotate270, true
	}
	return 0, false
}

func tiffByteOrder(header []byte) (binary.ByteOrder, bool) {
	if len(header) < 4 {
		return nil, false
	}
	switch {
	case header[0] == 0x49 && header[1] == 0x49 && header[2] == 42 && header[3] == 0:
		return binary.LittleEndian, true
	case header[0] == 0x4d && header[1] == 0x4d && header[2] == 0 && header[3] == 42:
		return binary.BigEndian, true
	}
	return nil, false
}

func imageOrientationFromExifChunk(chunk []byte) (imageOrientation, bool) {
	order, ok := tiffByteOrder(chunk)
	if !ok || len(chunk) < 8 {
		return 0, false
	}
	position := uint64(order.Uint32(chunk[4:8]))
	if position+2 > uint64(len(chunk)) {
		return 0, false
	}
	entries := order.Uint16(chunk[position:])
	position += 2
	for range entries {
		if position+12 > uint64(len(chunk)) {
			return 0, false
		}
		entry := chunk[position : position+12]
		position += 12
		tag := order.Uint16(entry[0:2])
		format := order.Uint16(entry[2:4])
		count := order.Uint32(entry[4:8])
		value := order.Uint16(entry[8:10])
		if tag == 0x112 && format == 3 && count == 1 {
			return imageOrientationFromExif(uint8(min(value, 255)))
		}
	}
	return 0, false
}

func tiffOrientation(data []byte) imageOrientation {
	order, ok := tiffByteOrder(data)
	if !ok || len(data) < 8 {
		return imageOrientationNoTransforms
	}
	position := uint64(order.Uint32(data[4:8]))
	if position+2 > uint64(len(data)) {
		return imageOrientationNoTransforms
	}
	entries := order.Uint16(data[position:])
	position += 2
	for range entries {
		if position+12 > uint64(len(data)) {
			break
		}
		entry := data[position : position+12]
		position += 12
		if order.Uint16(entry[0:2]) != 274 || order.Uint32(entry[4:8]) != 1 {
			continue
		}
		var value uint32
		switch order.Uint16(entry[2:4]) {
		case 1:
			value = uint32(entry[8])
		case 3:
			value = uint32(order.Uint16(entry[8:10]))
		case 4:
			value = order.Uint32(entry[8:12])
			if value > math.MaxUint16 {
				return imageOrientationNoTransforms
			}
		default:
			return imageOrientationNoTransforms
		}
		if orientation, ok := imageOrientationFromExif(uint8(min(value, 255))); ok {
			return orientation
		}
		return imageOrientationNoTransforms
	}
	return imageOrientationNoTransforms
}

func orientationFromExifOption(exif []byte, ok bool) imageOrientation {
	if !ok {
		return imageOrientationNoTransforms
	}
	if orientation, ok := imageOrientationFromExifChunk(exif); ok {
		return orientation
	}
	return imageOrientationNoTransforms
}

func jpegExifMetadata(data []byte) ([]byte, bool) {
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		return nil, false
	}
	position := 2
	for position+4 <= len(data) {
		if data[position] != 0xff {
			return nil, false
		}
		marker := data[position+1]
		if marker == 0xff {
			position++
			continue
		}
		if marker == 0xd8 || marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			position += 2
			continue
		}
		if marker == 0xda || marker == 0xd9 {
			return nil, false
		}
		length := int(binary.BigEndian.Uint16(data[position+2:]))
		if length < 2 || position+2+length > len(data) {
			return nil, false
		}
		segment := data[position+4 : position+2+length]
		if marker == 0xe1 && len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
			return segment[6:], true
		}
		position += 2 + length
	}
	return nil, false
}

func pngExifMetadata(data []byte) ([]byte, bool) {
	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(signature) || string(data[:len(signature)]) != signature {
		return nil, false
	}
	position := len(signature)
	for position+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[position:]))
		kind := string(data[position+4 : position+8])
		start := position + 8
		if length < 0 || start+length > len(data) {
			return nil, false
		}
		switch kind {
		case "eXIf":
			return data[start : start+length], true
		case "IDAT", "IEND":
			return nil, false
		}
		position = start + length + 4
	}
	return nil, false
}

func applyImageOrientation(img *image.RGBA, orientation imageOrientation) *image.RGBA {
	switch orientation {
	case imageOrientationRotate90:
		return rotateImage90(img)
	case imageOrientationRotate180:
		flipImageHorizontal(img)
		flipImageVertical(img)
	case imageOrientationRotate270:
		return rotateImage270(img)
	case imageOrientationFlipHorizontal:
		flipImageHorizontal(img)
	case imageOrientationFlipVertical:
		flipImageVertical(img)
	case imageOrientationRotate90FlipH:
		rotated := rotateImage90(img)
		flipImageHorizontal(rotated)
		return rotated
	case imageOrientationRotate270FlipH:
		rotated := rotateImage270(img)
		flipImageHorizontal(rotated)
		return rotated
	}
	return img
}

func rotateImage90(img *image.RGBA) *image.RGBA {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, height, width))
	for y := range height {
		for x := range width {
			copy(out.Pix[out.PixOffset(height-1-y, x):][:4], img.Pix[img.PixOffset(x, y):][:4])
		}
	}
	return out
}

func rotateImage270(img *image.RGBA) *image.RGBA {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, height, width))
	for y := range height {
		for x := range width {
			copy(out.Pix[out.PixOffset(y, width-1-x):][:4], img.Pix[img.PixOffset(x, y):][:4])
		}
	}
	return out
}

func flipImageHorizontal(img *image.RGBA) {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	for y := range height {
		row := img.Pix[y*img.Stride:]
		for x := range width / 2 {
			left, right := row[x*4:x*4+4], row[(width-1-x)*4:(width-1-x)*4+4]
			for i := range 4 {
				left[i], right[i] = right[i], left[i]
			}
		}
	}
}

func flipImageVertical(img *image.RGBA) {
	height := img.Rect.Dy()
	rowBytes := img.Rect.Dx() * 4
	for y := range height / 2 {
		top := img.Pix[y*img.Stride : y*img.Stride+rowBytes]
		bottom := img.Pix[(height-1-y)*img.Stride : (height-1-y)*img.Stride+rowBytes]
		for i := range rowBytes {
			top[i], bottom[i] = bottom[i], top[i]
		}
	}
}

func swapRedBlue(pix []byte) {
	for i := 0; i+3 < len(pix); i += 4 {
		pix[i], pix[i+2] = pix[i+2], pix[i]
	}
}

func u16ToU8(c uint32) uint8 {
	return uint8((c + 128) / 257)
}

func imageIntoRGBA8(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	switch m := img.(type) {
	case *image.NRGBA:
		for y := range height {
			copy(out.Pix[y*out.Stride:][:width*4], m.Pix[m.PixOffset(bounds.Min.X, bounds.Min.Y+y):][:width*4])
		}
	case *image.Gray:
		for y := range height {
			for x := range width {
				v := m.Pix[m.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)]
				copy(out.Pix[out.PixOffset(x, y):][:4], []byte{v, v, v, 255})
			}
		}
	case *image.YCbCr:
		for y := range height {
			for x := range width {
				c := m.YCbCrAt(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
				copy(out.Pix[out.PixOffset(x, y):][:4], []byte{r, g, b, 255})
			}
		}
	case *image.NYCbCrA:
		for y := range height {
			for x := range width {
				c := m.NYCbCrAAt(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
				copy(out.Pix[out.PixOffset(x, y):][:4], []byte{r, g, b, c.A})
			}
		}
	case *image.Paletted:
		palette := make([][4]byte, len(m.Palette))
		for i, c := range m.Palette {
			n := color.NRGBAModel.Convert(c).(color.NRGBA)
			palette[i] = [4]byte{n.R, n.G, n.B, n.A}
		}
		for y := range height {
			for x := range width {
				index := int(m.Pix[m.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)])
				var p [4]byte
				if index < len(palette) {
					p = palette[index]
				} else {
					p = [4]byte{0, 0, 0, 255}
				}
				copy(out.Pix[out.PixOffset(x, y):][:4], p[:])
			}
		}
	default:
		for y := range height {
			for x := range width {
				c := color.NRGBA64Model.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA64)
				copy(out.Pix[out.PixOffset(x, y):][:4], []byte{
					u16ToU8(uint32(c.R)), u16ToU8(uint32(c.G)), u16ToU8(uint32(c.B)), u16ToU8(uint32(c.A)),
				})
			}
		}
	}
	return out
}

type stdImageDecoder struct {
	data       []byte
	orient     imageOrientation
	decodeFunc func(r io.Reader) (image.Image, error)
}

func (d *stdImageDecoder) orientation() (imageOrientation, error) {
	return d.orient, nil
}

func (d *stdImageDecoder) decode() (image.Image, error) {
	return d.decodeFunc(bytes.NewReader(d.data))
}

func newImageDecoder(data []byte, format ImageFormat) (imageDecoder, error) {
	switch format {
	case ImageFormatPng:
		if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return &stdImageDecoder{data: data, orient: orientationFromExifOption(pngExifMetadata(data)), decodeFunc: png.Decode}, nil
	case ImageFormatJpeg:
		if _, err := jpeg.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return &stdImageDecoder{data: data, orient: orientationFromExifOption(jpegExifMetadata(data)), decodeFunc: jpeg.Decode}, nil
	case ImageFormatBmp:
		if _, err := bmp.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return &stdImageDecoder{data: data, decodeFunc: bmp.Decode}, nil
	case ImageFormatTiff:
		if _, err := tiff.DecodeConfig(bytes.NewReader(data)); err != nil {
			return nil, err
		}
		return &stdImageDecoder{data: data, orient: tiffOrientation(data), decodeFunc: tiff.Decode}, nil
	case ImageFormatIco:
		return newIcoDecoder(data)
	case ImageFormatPnm:
		return newPnmDecoder(data)
	case ImageFormatWebp:
		return newWebpDecoder(data)
	}
	return nil, fmt.Errorf("the image format %v is not supported", format)
}

type gifDisposal uint8

const (
	gifDisposalAny gifDisposal = iota
	gifDisposalKeep
	gifDisposalBackground
	gifDisposalPrevious
)

func newGifFrames(data []byte) (iter.Seq2[ImageFrame, error], error) {
	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	width, height := decoded.Config.Width, decoded.Config.Height
	return func(yield func(ImageFrame, error) bool) {
		nonDisposed := image.NewRGBA(image.Rect(0, 0, width, height))
		for i, paletted := range decoded.Image {
			disposal := gifDisposalAny
			if i < len(decoded.Disposal) {
				switch decoded.Disposal[i] {
				case gif.DisposalNone:
					disposal = gifDisposalKeep
				case gif.DisposalBackground:
					disposal = gifDisposalBackground
				case gif.DisposalPrevious:
					disposal = gifDisposalPrevious
				}
			}
			frame := imageIntoRGBA8(paletted)
			left, top := paletted.Rect.Min.X, paletted.Rect.Min.Y
			out := image.NewRGBA(image.Rect(0, 0, width, height))
			for y := range height {
				for x := range width {
					previous := nonDisposed.Pix[nonDisposed.PixOffset(x, y):][:4]
					target := out.Pix[out.PixOffset(x, y):][:4]
					frameX, frameY := x-left, y-top
					if frameX < 0 || frameY < 0 || frameX >= frame.Rect.Dx() || frameY >= frame.Rect.Dy() {
						copy(target, previous)
						continue
					}
					copy(target, frame.Pix[frame.PixOffset(frameX, frameY):][:4])
					if target[3] == 0 {
						copy(target, previous)
					}
					switch disposal {
					case gifDisposalAny, gifDisposalKeep:
						copy(previous, target)
					case gifDisposalBackground:
						clear(previous)
					}
				}
			}
			var delay time.Duration
			if i < len(decoded.Delay) {
				delay = time.Duration(decoded.Delay[i]*10) * time.Millisecond
			}
			result := newImageFrame(out)
			result.Delay = delay
			if !yield(result, nil) {
				return
			}
		}
	}, nil
}

var (
	errWebpChunkMissing          = errors.New("an expected chunk was missing")
	errWebpFrameOutsideImage     = errors.New("frame is outside the image")
	errWebpInconsistentImageSize = errors.New("image sizes are inconsistent")
	errWebpInvalidChunkSize      = errors.New("invalid chunk size")
	errWebpLimitsExceeded        = errors.New("the image is too large to decode within the memory limit")
	errWebpRiffSignatureInvalid  = errors.New("invalid RIFF signature")
	errWebpSignatureInvalid      = errors.New("invalid WebP signature")
)

type webpChunk struct {
	fourCC string
	data   []byte
}

type webpDecoder struct {
	data      []byte
	width     uint32
	height    uint32
	hasAlpha  bool
	animated  bool
	exif      []byte
	hasExif   bool
	frameData []webpChunk
}

func read3Bytes(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}

func readWebpChunks(data []byte) ([]webpChunk, error) {
	var chunks []webpChunk
	for len(data) >= 8 {
		size := binary.LittleEndian.Uint32(data[4:8])
		if uint64(size) > uint64(len(data)-8) {
			return chunks, errWebpInvalidChunkSize
		}
		chunks = append(chunks, webpChunk{fourCC: string(data[:4]), data: data[8 : 8+size]})
		rounded := uint64(size) + uint64(size&1)
		if 8+rounded >= uint64(len(data)) {
			break
		}
		data = data[8+rounded:]
	}
	return chunks, nil
}

func newWebpDecoder(data []byte) (*webpDecoder, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" {
		return nil, errWebpRiffSignatureInvalid
	}
	if string(data[8:12]) != "WEBP" {
		return nil, errWebpSignatureInvalid
	}
	riffSize := binary.LittleEndian.Uint32(data[4:8])
	body := data[12:]
	if uint64(riffSize) >= 4 && uint64(riffSize)-4 < uint64(len(body)) {
		body = body[:riffSize-4]
	}
	chunks, err := readWebpChunks(body)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, errWebpChunkMissing
	}
	d := &webpDecoder{data: data}
	switch first := chunks[0]; first.fourCC {
	case "VP8 ":
		config, err := webp.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		d.width, d.height = uint32(config.Width), uint32(config.Height)
	case "VP8L":
		if len(first.data) < 5 {
			return nil, errWebpInvalidChunkSize
		}
		header := binary.LittleEndian.Uint32(first.data[1:5])
		d.width = header&0x3fff + 1
		d.height = (header>>14)&0x3fff + 1
		d.hasAlpha = (header>>28)&1 != 0
	case "VP8X":
		if len(first.data) < 10 {
			return nil, errWebpInvalidChunkSize
		}
		flags := first.data[0]
		d.animated = flags&0x02 != 0
		d.hasAlpha = flags&0x10 != 0
		exifFlag := flags&0x08 != 0
		d.width = read3Bytes(first.data[4:7]) + 1
		d.height = read3Bytes(first.data[7:10]) + 1
		var hasAnim, hasVP8, hasVP8L bool
		for _, chunk := range chunks[1:] {
			switch chunk.fourCC {
			case "ANIM":
				hasAnim = true
			case "ANMF":
				if len(chunk.data) < 24 {
					return nil, errWebpInvalidChunkSize
				}
				d.frameData = append(d.frameData, chunk)
			case "EXIF":
				if !d.hasExif {
					d.exif, d.hasExif = chunk.data, true
				}
			case "VP8 ":
				hasVP8 = true
			case "VP8L":
				hasVP8L = true
			}
		}
		if d.animated && (!hasAnim || len(d.frameData) == 0) ||
			exifFlag && !d.hasExif ||
			!d.animated && hasVP8 == hasVP8L {
			return nil, errWebpChunkMissing
		}
	default:
		return nil, fmt.Errorf("invalid chunk signature: %q", first.fourCC)
	}
	return d, nil
}

func (d *webpDecoder) hasAnimation() bool {
	return d.animated
}

func (d *webpDecoder) orientation() (imageOrientation, error) {
	return orientationFromExifOption(d.exif, d.hasExif), nil
}

func (d *webpDecoder) decode() (image.Image, error) {
	return webp.Decode(bytes.NewReader(d.data))
}

func webpChunkBytes(fourCC string, data []byte) []byte {
	out := make([]byte, 0, 8+len(data)+1)
	out = append(out, fourCC...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	out = append(out, data...)
	if len(data)&1 == 1 {
		out = append(out, 0)
	}
	return out
}

func webpFile(chunks ...[]byte) []byte {
	size := 4
	for _, chunk := range chunks {
		size += len(chunk)
	}
	out := make([]byte, 0, 8+size)
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(size))
	out = append(out, "WEBP"...)
	for _, chunk := range chunks {
		out = append(out, chunk...)
	}
	return out
}

func decodeWebpFramePixels(file []byte, width, height uint32, withAlpha bool) ([]byte, error) {
	decoded, err := webp.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, err
	}
	if uint32(decoded.Bounds().Dx()) != width || uint32(decoded.Bounds().Dy()) != height {
		return nil, errWebpInconsistentImageSize
	}
	rgba := imageIntoRGBA8(decoded)
	if withAlpha {
		return rgba.Pix, nil
	}
	rgb := make([]byte, 0, int(width*height)*3)
	for i := 0; i+3 < len(rgba.Pix); i += 4 {
		rgb = append(rgb, rgba.Pix[i:i+3]...)
	}
	return rgb, nil
}

func (d *webpDecoder) readFrame(anmf webpChunk) (frame []byte, frameX, frameY, frameWidth, frameHeight, duration uint32, hasAlpha, blend, dispose bool, err error) {
	header := anmf.data
	frameX = read3Bytes(header[0:3]) * 2
	frameY = read3Bytes(header[3:6]) * 2
	frameWidth = read3Bytes(header[6:9]) + 1
	frameHeight = read3Bytes(header[9:12]) + 1
	if frameWidth > 16384 || frameHeight > 16384 {
		err = errors.New("image too large")
		return
	}
	if frameX+frameWidth > d.width || frameY+frameHeight > d.height {
		err = errWebpFrameOutsideImage
		return
	}
	duration = read3Bytes(header[12:15])
	frameInfo := header[15]
	blend = frameInfo&0b00000010 == 0
	dispose = frameInfo&0b00000001 != 0
	subchunks, chunkErr := readWebpChunks(header[16:])
	if chunkErr != nil || len(subchunks) == 0 {
		err = errWebpInvalidChunkSize
		return
	}
	switch subchunks[0].fourCC {
	case "VP8 ":
		frame, err = decodeWebpFramePixels(webpFile(webpChunkBytes("VP8 ", subchunks[0].data)), frameWidth, frameHeight, false)
	case "VP8L":
		hasAlpha = true
		frame, err = decodeWebpFramePixels(webpFile(webpChunkBytes("VP8L", subchunks[0].data)), frameWidth, frameHeight, true)
	case "ALPH":
		if len(subchunks) < 2 || subchunks[1].fourCC != "VP8 " {
			err = errWebpInvalidChunkSize
			return
		}
		hasAlpha = true
		vp8x := make([]byte, 10)
		vp8x[0] = 0x10
		vp8x[4], vp8x[5], vp8x[6] = byte(frameWidth-1), byte((frameWidth-1)>>8), byte((frameWidth-1)>>16)
		vp8x[7], vp8x[8], vp8x[9] = byte(frameHeight-1), byte((frameHeight-1)>>8), byte((frameHeight-1)>>16)
		frame, err = decodeWebpFramePixels(webpFile(
			webpChunkBytes("VP8X", vp8x),
			webpChunkBytes("ALPH", subchunks[0].data),
			webpChunkBytes("VP8 ", subchunks[1].data),
		), frameWidth, frameHeight, true)
	default:
		err = fmt.Errorf("invalid chunk signature: %q", subchunks[0].fourCC)
	}
	return
}

func (d *webpDecoder) frames() iter.Seq2[ImageFrame, error] {
	return func(yield func(ImageFrame, error) bool) {
		if uint64(d.width)*uint64(d.height)*4 > webpMaxCanvasBytes {
			yield(ImageFrame{}, errWebpLimitsExceeded)
			return
		}
		canvas := make([]byte, int(d.width)*int(d.height)*4)
		var disposeNextFrame bool
		var previousWidth, previousHeight, previousX, previousY uint32
		next := 0
		for range d.frameData {
			anmf := d.frameData[next]
			frame, frameX, frameY, frameWidth, frameHeight, duration, frameHasAlpha, blend, dispose, err := d.readFrame(anmf)
			if err != nil {
				next++
				if !yield(ImageFrame{}, err) {
					return
				}
				continue
			}
			var clearColor *[4]byte
			if disposeNextFrame {
				clearColor = &[4]byte{}
			}
			webpCompositeFrame(canvas, d.width, d.height, clearColor, frame, frameX, frameY, frameWidth, frameHeight, frameHasAlpha, blend, previousWidth, previousHeight, previousX, previousY)
			previousWidth, previousHeight, previousX, previousY = frameWidth, frameHeight, frameX, frameY
			disposeNextFrame = dispose
			next++
			out := image.NewRGBA(image.Rect(0, 0, int(d.width), int(d.height)))
			copy(out.Pix, canvas)
			if !d.hasAlpha {
				for i := 3; i < len(out.Pix); i += 4 {
					out.Pix[i] = 255
				}
			}
			result := newImageFrame(out)
			result.Delay = time.Duration(duration) * time.Millisecond
			if !yield(result, nil) {
				return
			}
		}
	}
}

func webpCompositeFrame(canvas []byte, canvasWidth, canvasHeight uint32, clearColor *[4]byte, frame []byte, frameOffsetX, frameOffsetY, frameWidth, frameHeight uint32, frameHasAlpha, frameUseAlphaBlending bool, previousFrameWidth, previousFrameHeight, previousFrameOffsetX, previousFrameOffsetY uint32) {
	frameIsFullSize := frameOffsetX == 0 && frameOffsetY == 0 && frameWidth == canvasWidth && frameHeight == canvasHeight
	if frameIsFullSize && !frameUseAlphaBlending {
		if frameHasAlpha {
			copy(canvas, frame)
		} else {
			for i, o := 0, 0; i+3 <= len(frame) && o+4 <= len(canvas); i, o = i+3, o+4 {
				copy(canvas[o:o+3], frame[i:i+3])
				canvas[o+3] = 255
			}
		}
		return
	}
	if clearColor != nil {
		switch {
		case frameIsFullSize && frameHasAlpha:
			for i := 0; i+4 <= len(canvas); i += 4 {
				copy(canvas[i:i+4], clearColor[:])
			}
		case frameIsFullSize:
			for i := 0; i+4 <= len(canvas); i += 4 {
				copy(canvas[i:i+3], clearColor[:3])
				canvas[i+3] = 255
			}
		case frameHasAlpha:
			for y := range previousFrameHeight {
				for x := range previousFrameWidth {
					index := ((x + previousFrameOffsetX) + (y+previousFrameOffsetY)*canvasWidth) * 4
					copy(canvas[index:index+4], clearColor[:])
				}
			}
		default:
			for y := range previousFrameHeight {
				for x := range previousFrameWidth {
					index := ((x + previousFrameOffsetX) + (y+previousFrameOffsetY)*canvasWidth) * 4
					copy(canvas[index:index+3], clearColor[:3])
					canvas[index+3] = 255
				}
			}
		}
	}
	width := int(min(frameWidth, saturatingSubU32(canvasWidth, frameOffsetX)))
	height := int(min(frameHeight, saturatingSubU32(canvasHeight, frameOffsetY)))
	switch {
	case frameHasAlpha && frameUseAlphaBlending:
		for y := range height {
			for x := range width {
				frameIndex := (x + y*int(frameWidth)) * 4
				canvasIndex := ((x + int(frameOffsetX)) + (y+int(frameOffsetY))*int(canvasWidth)) * 4
				blended := webpDoAlphaBlending([4]byte(frame[frameIndex:frameIndex+4]), [4]byte(canvas[canvasIndex:canvasIndex+4]))
				copy(canvas[canvasIndex:canvasIndex+4], blended[:])
			}
		}
	case frameHasAlpha:
		for y := range height {
			frameIndex := y * int(frameWidth) * 4
			canvasIndex := (int(frameOffsetX) + (y+int(frameOffsetY))*int(canvasWidth)) * 4
			copy(canvas[canvasIndex:canvasIndex+width*4], frame[frameIndex:frameIndex+width*4])
		}
	default:
		for y := range height {
			index := y * int(frameWidth) * 3
			canvasIndex := (int(frameOffsetX) + (y+int(frameOffsetY))*int(canvasWidth)) * 4
			for x := range width {
				copy(canvas[canvasIndex+x*4:canvasIndex+x*4+3], frame[index+x*3:index+x*3+3])
				canvas[canvasIndex+x*4+3] = 255
			}
		}
	}
}

const webpMaxCanvasBytes = 512 << 20

func saturatingSubU32(a, b uint32) uint32 {
	if b > a {
		return 0
	}
	return a - b
}

func webpDivBy255(v uint32) uint32 {
	return (((v + 0x80) >> 8) + v + 0x80) >> 8
}

func webpBlendChannelNonpremult(src uint32, srcA uint8, dst uint32, dstA uint8, scale uint32, shift uint32) uint8 {
	srcChannel := uint8((src >> shift) & 0xff)
	dstChannel := uint8((dst >> shift) & 0xff)
	blendUnscaled := uint32(srcChannel)*uint32(srcA) + uint32(dstChannel)*uint32(dstA)
	return uint8((blendUnscaled * scale) >> 24)
}

func webpBlendPixelNonpremult(src, dst uint32) uint32 {
	srcA := uint8((src >> 24) & 0xff)
	if srcA == 0 {
		return dst
	}
	dstA := uint8((dst >> 24) & 0xff)
	dstFactorA := webpDivBy255(uint32(dstA) * (255 - uint32(srcA)))
	blendA := uint32(srcA) + dstFactorA
	scale := (uint32(1) << 24) / blendA
	blendR := webpBlendChannelNonpremult(src, srcA, dst, uint8(dstFactorA), scale, 0)
	blendG := webpBlendChannelNonpremult(src, srcA, dst, uint8(dstFactorA), scale, 8)
	blendB := webpBlendChannelNonpremult(src, srcA, dst, uint8(dstFactorA), scale, 16)
	return uint32(blendR) | uint32(blendG)<<8 | uint32(blendB)<<16 | blendA<<24
}

func webpDoAlphaBlending(buffer, canvas [4]byte) [4]byte {
	blended := webpBlendPixelNonpremult(binary.LittleEndian.Uint32(buffer[:]), binary.LittleEndian.Uint32(canvas[:]))
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], blended)
	return out
}

type icoDirEntry struct {
	width        uint8
	height       uint8
	colorCount   uint8
	reserved     uint8
	colorPlanes  uint16
	bitsPerPixel uint16
	imageLength  uint32
	imageOffset  uint32
}

func (e icoDirEntry) realWidth() uint16 {
	if e.width == 0 {
		return 256
	}
	return uint16(e.width)
}

func (e icoDirEntry) realHeight() uint16 {
	if e.height == 0 {
		return 256
	}
	return uint16(e.height)
}

func (e icoDirEntry) matchesDimensions(width, height uint32) bool {
	return uint32(e.realWidth()) == min(width, 256) && uint32(e.realHeight()) == min(height, 256)
}

type icoDecoder struct {
	data  []byte
	entry icoDirEntry
	isPNG bool
	dib   icoDIB
}

var errUnexpectedEOF = io.ErrUnexpectedEOF

func newIcoDecoder(data []byte) (*icoDecoder, error) {
	if len(data) < 6 {
		return nil, errUnexpectedEOF
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	entries := make([]icoDirEntry, 0, count)
	for i := range count {
		offset := 6 + i*16
		if offset+16 > len(data) {
			return nil, errUnexpectedEOF
		}
		raw := data[offset : offset+16]
		entry := icoDirEntry{
			width:        raw[0],
			height:       raw[1],
			colorCount:   raw[2],
			reserved:     raw[3],
			colorPlanes:  binary.LittleEndian.Uint16(raw[4:6]),
			bitsPerPixel: binary.LittleEndian.Uint16(raw[6:8]),
			imageLength:  binary.LittleEndian.Uint32(raw[8:12]),
			imageOffset:  binary.LittleEndian.Uint32(raw[12:16]),
		}
		if entry.colorPlanes > 256 {
			return nil, errors.New("ICO image entry has too many color planes or too large hotspot value")
		}
		if entry.bitsPerPixel > 256 {
			return nil, errors.New("ICO image entry has too many bits per pixel or too large hotspot value")
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, errors.New("ICO directory contains no image")
	}
	best := entries[len(entries)-1]
	bestBits, bestArea := best.bitsPerPixel, uint32(best.realWidth())*uint32(best.realHeight())
	for _, entry := range entries[:len(entries)-1] {
		bits, area := entry.bitsPerPixel, uint32(entry.realWidth())*uint32(entry.realHeight())
		if bits > bestBits || bits == bestBits && area > bestArea {
			best, bestBits, bestArea = entry, bits, area
		}
	}
	start := int(best.imageOffset)
	if start+8 > len(data) {
		return nil, errUnexpectedEOF
	}
	d := &icoDecoder{data: data, entry: best, isPNG: string(data[start:start+8]) == "\x89PNG\r\n\x1a\n"}
	if d.isPNG {
		if _, err := png.DecodeConfig(bytes.NewReader(data[start:])); err != nil {
			return nil, err
		}
		return d, nil
	}
	dib, err := parseIcoDIB(data[start:])
	if err != nil {
		return nil, err
	}
	d.dib = dib
	return d, nil
}

func (d *icoDecoder) orientation() (imageOrientation, error) {
	return imageOrientationNoTransforms, nil
}

func (d *icoDecoder) decode() (image.Image, error) {
	start := int(d.entry.imageOffset)
	if d.isPNG {
		if d.entry.imageLength < 8 {
			return nil, errors.New("Entry specified a length that is shorter than PNG header!")
		}
		config, err := png.DecodeConfig(bytes.NewReader(d.data[start:]))
		if err != nil {
			return nil, err
		}
		if !d.entry.matchesDimensions(uint32(config.Width), uint32(config.Height)) {
			return nil, fmt.Errorf("Entry(%d, %d) and PNG(%d, %d) dimensions do not match!", d.entry.realWidth(), d.entry.realHeight(), config.Width, config.Height)
		}
		if config.ColorModel != color.NRGBAModel {
			return nil, errors.New("The PNG is not in RGBA format!")
		}
		return png.Decode(bytes.NewReader(d.data[start:]))
	}
	width, height := d.dib.width, d.dib.height
	if !d.entry.matchesDimensions(width, height) {
		return nil, fmt.Errorf("Entry(%d, %d) and BMP(%d, %d) dimensions do not match!", d.entry.realWidth(), d.entry.realHeight(), width, height)
	}
	img, imageEnd, err := d.dib.decode(d.data[start:])
	if err != nil {
		return nil, err
	}
	imageEnd += uint64(start)
	dataEnd := uint64(d.entry.imageOffset) + uint64(d.entry.imageLength)
	maskRowBytes := (width + 31) / 32 * 4
	maskLength := uint64(maskRowBytes) * uint64(height)
	switch {
	case dataEnd >= imageEnd+maskLength:
		position := imageEnd
		for y := range height {
			var x uint32
			for range maskRowBytes {
				if position >= uint64(len(d.data)) {
					return nil, errUnexpectedEOF
				}
				maskByte := d.data[position]
				position++
				for bit := 7; bit >= 0; bit-- {
					if x >= width {
						break
					}
					if maskByte&(1<<bit) != 0 {
						img.Pix[((height-y-1)*width+x)*4+3] = 0
					}
					x++
				}
			}
		}
		return img, nil
	case dataEnd == imageEnd:
		return img, nil
	}
	return nil, errors.New("ICO image data size did not match expected size")
}

type icoDIB struct {
	headerSize uint32
	width      uint32
	height     uint32
	topDown    bool
	bitCount   uint16
	colorsUsed uint32
}

func parseIcoDIB(data []byte) (icoDIB, error) {
	if len(data) < 4 {
		return icoDIB{}, errUnexpectedEOF
	}
	headerSize := binary.LittleEndian.Uint32(data[0:4])
	if headerSize < 40 {
		return icoDIB{}, fmt.Errorf("unsupported BMP header size %d in ICO entry", headerSize)
	}
	if uint64(len(data)) < uint64(headerSize) {
		return icoDIB{}, errUnexpectedEOF
	}
	width := int32(binary.LittleEndian.Uint32(data[4:8]))
	height := int32(binary.LittleEndian.Uint32(data[8:12]))
	bitCount := binary.LittleEndian.Uint16(data[14:16])
	compression := binary.LittleEndian.Uint32(data[16:20])
	colorsUsed := binary.LittleEndian.Uint32(data[32:36])
	if width <= 0 || height == 0 {
		return icoDIB{}, errors.New("invalid BMP dimensions in ICO entry")
	}
	topDown := height < 0
	if topDown {
		height = -height
	}
	height /= 2
	if compression != 0 {
		return icoDIB{}, fmt.Errorf("unsupported BMP compression %d in ICO entry", compression)
	}
	switch bitCount {
	case 1, 4, 8, 24, 32:
	default:
		return icoDIB{}, fmt.Errorf("unsupported BMP bit count %d in ICO entry", bitCount)
	}
	if bitCount <= 8 && colorsUsed > 1<<bitCount {
		return icoDIB{}, fmt.Errorf("palette size %d exceeds maximum size for BMP with bit count of %d", colorsUsed, bitCount)
	}
	return icoDIB{
		headerSize: headerSize,
		width:      uint32(width),
		height:     uint32(height),
		topDown:    topDown,
		bitCount:   bitCount,
		colorsUsed: colorsUsed,
	}, nil
}

func (d icoDIB) decode(data []byte) (*image.NRGBA, uint64, error) {
	position := uint64(d.headerSize)
	var palette [256][3]byte
	if d.bitCount <= 8 {
		size := uint64(d.colorsUsed)
		if size == 0 {
			size = 1 << d.bitCount
		}
		if position+size*4 > uint64(len(data)) {
			return nil, 0, errUnexpectedEOF
		}
		for i := range min(size, 256) {
			entry := data[position+i*4:]
			palette[i] = [3]byte{entry[2], entry[1], entry[0]}
		}
		position += size * 4
	}
	width, height := int(d.width), int(d.height)
	rowBytes := (width*int(d.bitCount) + 31) / 32 * 4
	if position+uint64(rowBytes)*uint64(height) > uint64(len(data)) {
		return nil, 0, errUnexpectedEOF
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for row := range height {
		y := height - 1 - row
		if d.topDown {
			y = row
		}
		src := data[position+uint64(row*rowBytes):][:rowBytes]
		dst := out.Pix[y*out.Stride:]
		for x := range width {
			pixel := dst[x*4 : x*4+4]
			switch d.bitCount {
			case 1, 4, 8:
				bits := int(d.bitCount)
				bitOffset := x * bits
				index := (src[bitOffset/8] >> (8 - bits - bitOffset%8)) & byte(1<<bits-1)
				p := palette[index]
				pixel[0], pixel[1], pixel[2], pixel[3] = p[0], p[1], p[2], 255
			case 24:
				pixel[0], pixel[1], pixel[2], pixel[3] = src[x*3+2], src[x*3+1], src[x*3], 255
			case 32:
				pixel[0], pixel[1], pixel[2], pixel[3] = src[x*4+2], src[x*4+1], src[x*4], src[x*4+3]
			}
		}
	}
	position += uint64(rowBytes) * uint64(height)
	return out, position, nil
}

type pnmTupleType uint8

const (
	pnmTuplePbmBit pnmTupleType = iota
	pnmTupleBWBit
	pnmTupleBWAlphaBit
	pnmTupleGray
	pnmTupleGrayAlpha
	pnmTupleRGB
	pnmTupleRGBAlpha
)

type pnmDecoder struct {
	data       []byte
	pixels     []byte
	ascii      bool
	width      uint32
	height     uint32
	maxval     uint32
	tuple      pnmTupleType
	components int
	wide       bool
}

func isPnmSeparator(c byte) bool {
	switch c {
	case '\t', '\n', '\x0b', '\x0c', '\r', ' ':
		return true
	}
	return false
}

func (d *pnmDecoder) readNextU32() (uint32, error) {
	var value uint32
	foundDigit := false
	inComment := false
	for len(d.pixels) > 0 {
		c := d.pixels[0]
		d.pixels = d.pixels[1:]
		if inComment {
			if c == '\r' || c == '\n' {
				inComment = false
			}
			continue
		}
		if c == '#' {
			inComment = true
			continue
		}
		if isPnmSeparator(c) {
			if foundDigit {
				break
			}
			continue
		}
		if c >= 0x80 {
			return 0, fmt.Errorf("Non ASCII character %#x in header", c)
		}
		if c < '0' || c > '9' {
			return 0, errors.New("invalid digit in preamble")
		}
		next := uint64(value)*10 + uint64(c-'0')
		if next > math.MaxUint32 {
			return 0, errors.New("overflow in preamble")
		}
		value = uint32(next)
		foundDigit = true
	}
	if !foundDigit {
		return 0, errUnexpectedEOF
	}
	return value, nil
}

func (d *pnmDecoder) readArbitraryHeader() error {
	if len(d.pixels) == 0 {
		return errUnexpectedEOF
	}
	if d.pixels[0] != '\n' {
		return fmt.Errorf("Expected newline after P7 magic, got %#x", d.pixels[0])
	}
	d.pixels = d.pixels[1:]
	var width, height, depth, maxval Option[uint32]
	tupltype := None[string]()
	for {
		end := bytes.IndexByte(d.pixels, '\n')
		var line string
		if end < 0 {
			if len(d.pixels) == 0 {
				return errors.New("Unexpected end of PNM header")
			}
			line, d.pixels = string(d.pixels), nil
		} else {
			line, d.pixels = string(d.pixels[:end]), d.pixels[end+1:]
		}
		if line == "" || strings.HasPrefix(line, "#") {
			if line == "" && end < 0 {
				return errors.New("Unexpected end of PNM header")
			}
			continue
		}
		if !isASCII(line) {
			return errors.New("Only ASCII characters allowed in PAM header")
		}
		key, rest, _ := strings.Cut(line, " ")
		parse := func(field *Option[uint32], name string) error {
			if field.IsSome() {
				return fmt.Errorf("The header line %s was specified twice", name)
			}
			v, err := strconv.ParseUint(strings.TrimSpace(rest), 10, 32)
			if err != nil {
				return fmt.Errorf("Error parsing %q as %s: %w", rest, name, err)
			}
			*field = Some(uint32(v))
			return nil
		}
		var err error
		switch key {
		case "ENDHDR":
			w, okW := width.Get()
			h, okH := height.Get()
			dp, okD := depth.Get()
			m, okM := maxval.Get()
			if !okW || !okH || !okD || !okM {
				return errors.New("Missing header line")
			}
			d.width, d.height, d.maxval = w, h, m
			return d.arbitraryTuple(dp, tupltype)
		case "HEIGHT":
			err = parse(&height, "HEIGHT")
		case "WIDTH":
			err = parse(&width, "WIDTH")
		case "DEPTH":
			err = parse(&depth, "DEPTH")
		case "MAXVAL":
			err = parse(&maxval, "MAXVAL")
		case "TUPLTYPE":
			identifier := strings.TrimSpace(rest)
			if current, ok := tupltype.Get(); ok {
				tupltype = Some(current + " " + identifier)
			} else {
				tupltype = Some(identifier)
			}
		default:
			return fmt.Errorf("Unknown header line with identifier %q", key)
		}
		if err != nil {
			return err
		}
	}
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func (d *pnmDecoder) arbitraryTuple(depth uint32, tupltype Option[string]) error {
	if d.maxval == 0 {
		return errors.New("Image MAXVAL is zero")
	}
	wide := d.maxval > 0xff
	if d.maxval > 0xffff {
		return fmt.Errorf("Image MAXVAL exceeds %d: %d", 0xffff, d.maxval)
	}
	invalid := func(name string) error {
		return fmt.Errorf("Invalid depth (%d) or maxval (%d) for tuple type %s", depth, d.maxval, name)
	}
	name, named := tupltype.Get()
	if !named {
		switch depth {
		case 1:
			d.tuple, d.components = pnmTupleGray, 1
		case 2:
			d.tuple, d.components = pnmTupleGrayAlpha, 2
		case 3:
			d.tuple, d.components = pnmTupleRGB, 3
		case 4:
			d.tuple, d.components = pnmTupleRGBAlpha, 4
		default:
			return fmt.Errorf("Tuple type not recognized")
		}
		d.wide = wide
		return nil
	}
	switch name {
	case "BLACKANDWHITE":
		if d.maxval != 1 || depth != 1 {
			return invalid(name)
		}
		d.tuple, d.components = pnmTupleBWBit, 1
	case "BLACKANDWHITE_ALPHA":
		if d.maxval != 1 || depth != 2 {
			return invalid(name)
		}
		d.tuple, d.components = pnmTupleBWAlphaBit, 2
	case "GRAYSCALE":
		if depth != 1 {
			return invalid(name)
		}
		d.tuple, d.components, d.wide = pnmTupleGray, 1, wide
	case "GRAYSCALE_ALPHA":
		if depth != 2 {
			return invalid(name)
		}
		d.tuple, d.components, d.wide = pnmTupleGrayAlpha, 2, wide
	case "RGB":
		if depth != 3 {
			return invalid(name)
		}
		d.tuple, d.components, d.wide = pnmTupleRGB, 3, wide
	case "RGB_ALPHA":
		if depth != 4 {
			return invalid(name)
		}
		d.tuple, d.components, d.wide = pnmTupleRGBAlpha, 4, wide
	default:
		return fmt.Errorf("Tuple type not recognized: %q", name)
	}
	return nil
}

func newPnmDecoder(data []byte) (*pnmDecoder, error) {
	if len(data) < 2 || data[0] != 'P' {
		return nil, errors.New("Expected magic constant for PNM: P1..P7")
	}
	d := &pnmDecoder{data: data, pixels: data[2:]}
	var err error
	switch data[1] {
	case '1', '4':
		d.ascii = data[1] == '1'
		d.tuple, d.components, d.maxval = pnmTuplePbmBit, 1, 1
		if d.width, err = d.readNextU32(); err == nil {
			d.height, err = d.readNextU32()
		}
	case '2', '5', '3', '6':
		d.ascii = data[1] == '2' || data[1] == '3'
		d.tuple, d.components = pnmTupleGray, 1
		if data[1] == '3' || data[1] == '6' {
			d.tuple, d.components = pnmTupleRGB, 3
		}
		if d.width, err = d.readNextU32(); err == nil {
			if d.height, err = d.readNextU32(); err == nil {
				d.maxval, err = d.readNextU32()
			}
		}
		if err == nil {
			switch {
			case d.maxval == 0:
				err = errors.New("Image MAXVAL is zero")
			case d.maxval > 0xffff:
				err = fmt.Errorf("Image MAXVAL exceeds %d: %d", 0xffff, d.maxval)
			default:
				d.wide = d.maxval > 0xff
			}
		}
	case '7':
		err = d.readArbitraryHeader()
	default:
		return nil, errors.New("Expected magic constant for PNM: P1..P7")
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *pnmDecoder) orientation() (imageOrientation, error) {
	return imageOrientationNoTransforms, nil
}

func (d *pnmDecoder) readAsciiSample() (uint16, error) {
	for len(d.pixels) > 0 && isPnmSeparator(d.pixels[0]) {
		d.pixels = d.pixels[1:]
	}
	var value uint32
	hadAny := false
	for len(d.pixels) > 0 && !isPnmSeparator(d.pixels[0]) {
		c := d.pixels[0]
		d.pixels = d.pixels[1:]
		if c < '0' || c > '9' {
			return 0, errors.New("invalid digit in sample")
		}
		value = value*10 + uint32(c-'0')
		if value > math.MaxUint16 {
			return 0, errors.New("overflow in sample")
		}
		hadAny = true
	}
	if len(d.pixels) > 0 {
		d.pixels = d.pixels[1:]
	}
	if !hadAny {
		return 0, errors.New("Not enough data was provided to the Decoder to decode the image")
	}
	return uint16(value), nil
}

func (d *pnmDecoder) decode() (image.Image, error) {
	count := int(d.width) * int(d.height) * d.components
	samples := make([]uint16, count)
	switch {
	case d.tuple == pnmTuplePbmBit && d.ascii:
		for i := range samples {
			for {
				if len(d.pixels) == 0 {
					return nil, errors.New("Not enough data was provided to the Decoder to decode the image")
				}
				c := d.pixels[0]
				d.pixels = d.pixels[1:]
				if isPnmSeparator(c) {
					continue
				}
				switch c {
				case '0':
					samples[i] = 255
				case '1':
					samples[i] = 0
				default:
					return nil, fmt.Errorf("Unexpected character %#x within sample raster", c)
				}
				break
			}
		}
	case d.tuple == pnmTuplePbmBit:
		lineLength := (int(d.width) + 7) / 8
		if len(d.pixels) < lineLength*int(d.height) {
			return nil, errUnexpectedEOF
		}
		for y := range int(d.height) {
			for x := range int(d.width) {
				v := (d.pixels[y*lineLength+x/8] >> (7 - x%8)) & 1
				samples[y*int(d.width)+x] = uint16(1 - v)
			}
		}
	case d.ascii:
		for i := range samples {
			v, err := d.readAsciiSample()
			if err != nil {
				return nil, err
			}
			if !d.wide && v > 0xff {
				return nil, errors.New("overflow in sample")
			}
			samples[i] = v
		}
	case d.wide:
		if len(d.pixels) < count*2 {
			return nil, errUnexpectedEOF
		}
		for i := range samples {
			samples[i] = binary.BigEndian.Uint16(d.pixels[i*2:])
		}
	default:
		if len(d.pixels) < count {
			return nil, errUnexpectedEOF
		}
		for i := range samples {
			samples[i] = uint16(d.pixels[i])
		}
		if d.tuple == pnmTupleBWBit || d.tuple == pnmTupleBWAlphaBit {
			for _, v := range samples {
				if v > 1 {
					return nil, fmt.Errorf("Sample value %d outside of bounds", v)
				}
			}
		}
	}
	target := uint32(0xff)
	if d.wide {
		target = 0xffff
	}
	if d.maxval != target {
		factor := float32(target) / float32(d.maxval)
		for i, v := range samples {
			scaled := math.Round(float64(float32(v) * factor))
			samples[i] = uint16(min(max(scaled, 0), float64(target)))
		}
	}
	toU8 := func(v uint16) uint8 {
		if d.wide {
			return u16ToU8(uint32(v))
		}
		return uint8(v)
	}
	out := image.NewNRGBA(image.Rect(0, 0, int(d.width), int(d.height)))
	for i := range int(d.width) * int(d.height) {
		s := samples[i*d.components : (i+1)*d.components]
		pixel := out.Pix[i*4 : i*4+4]
		switch d.components {
		case 1:
			v := toU8(s[0])
			pixel[0], pixel[1], pixel[2], pixel[3] = v, v, v, 255
		case 2:
			v := toU8(s[0])
			pixel[0], pixel[1], pixel[2], pixel[3] = v, v, v, toU8(s[1])
		case 3:
			pixel[0], pixel[1], pixel[2], pixel[3] = toU8(s[0]), toU8(s[1]), toU8(s[2]), 255
		case 4:
			pixel[0], pixel[1], pixel[2], pixel[3] = toU8(s[0]), toU8(s[1]), toU8(s[2]), toU8(s[3])
		}
	}
	return out, nil
}

type seaHasher struct {
	state   [4]uint64
	written uint64
	tail    uint64
	ntail   int
}

func newSeaHasher() *seaHasher {
	return &seaHasher{state: [4]uint64{
		0x16f11fe89b0d677c,
		0xb480a793d8e6c86c,
		0x6fe2e5aaf078ebc9,
		0x14f994a4c5259381,
	}}
}

func seaDiffuse(x uint64) uint64 {
	x *= 0x6eed0e9da4d94a4f
	a := x >> 32
	b := x >> 60
	x ^= a >> b
	x *= 0x6eed0e9da4d94a4f
	return x
}

func seaReadInt(buf []byte) uint64 {
	var x uint64
	for i := len(buf) - 1; i >= 0; i-- {
		x = x<<8 | uint64(buf[i])
	}
	return x
}

func (h *seaHasher) push(x uint64) {
	a := seaDiffuse(h.state[0] ^ x)
	h.state[0] = h.state[1]
	h.state[1] = h.state[2]
	h.state[2] = h.state[3]
	h.state[3] = a
	h.written += 8
}

func (h *seaHasher) write(data []byte) {
	copied := min(8-h.ntail, len(data))
	var this [8]byte
	binary.LittleEndian.PutUint64(this[:], h.tail)
	copy(this[h.ntail:], data[:copied])
	if copied+h.ntail != 8 {
		h.ntail += copied
		h.tail = binary.LittleEndian.Uint64(this[:])
		return
	}
	h.push(binary.LittleEndian.Uint64(this[:]))
	h.ntail = 0
	h.tail = 0
	rest := data[copied:]
	for len(rest) >= 32 {
		h.state[0] = seaDiffuse(h.state[0] ^ binary.LittleEndian.Uint64(rest[0:]))
		h.state[1] = seaDiffuse(h.state[1] ^ binary.LittleEndian.Uint64(rest[8:]))
		h.state[2] = seaDiffuse(h.state[2] ^ binary.LittleEndian.Uint64(rest[16:]))
		h.state[3] = seaDiffuse(h.state[3] ^ binary.LittleEndian.Uint64(rest[24:]))
		rest = rest[32:]
		h.written += 32
	}
	excessive := len(rest)
	switch {
	case excessive == 0:
	case excessive <= 7:
		h.tail = seaReadInt(rest)
		h.ntail = excessive
	case excessive == 8:
		h.push(binary.LittleEndian.Uint64(rest))
	case excessive <= 15:
		h.push(binary.LittleEndian.Uint64(rest))
		h.tail = seaReadInt(rest[8:])
		h.ntail = excessive - 8
	case excessive <= 23:
		a := seaDiffuse(h.state[0] ^ binary.LittleEndian.Uint64(rest[0:]))
		b := seaDiffuse(h.state[1] ^ binary.LittleEndian.Uint64(rest[8:]))
		h.state[0] = h.state[2]
		h.state[1] = h.state[3]
		h.state[2] = a
		h.state[3] = b
		if excessive > 16 {
			h.tail = seaReadInt(rest[16:])
			h.ntail = excessive - 16
		}
		h.written += 16
	default:
		a := seaDiffuse(h.state[0] ^ binary.LittleEndian.Uint64(rest[0:]))
		b := seaDiffuse(h.state[1] ^ binary.LittleEndian.Uint64(rest[8:]))
		c := seaDiffuse(h.state[2] ^ binary.LittleEndian.Uint64(rest[16:]))
		h.state[0] = h.state[3]
		h.state[1] = a
		h.state[2] = b
		h.state[3] = c
		if excessive > 24 {
			h.tail = seaReadInt(rest[24:])
			h.ntail = excessive - 24
		}
		h.written += 24
	}
}

func (h *seaHasher) finish() uint64 {
	a := h.state[0]
	if h.ntail > 0 {
		var tail [8]byte
		binary.LittleEndian.PutUint64(tail[:], h.tail)
		a = seaDiffuse(h.state[0] ^ seaReadInt(tail[:h.ntail]))
	}
	return seaDiffuse(a ^ h.state[1] ^ h.state[2] ^ h.state[3] ^ (h.written + uint64(h.ntail)))
}
