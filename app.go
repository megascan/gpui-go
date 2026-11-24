package gpui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"time"
	"weak"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/jsonschema"
	"github.com/megascan/gpui-go/scheduler"
)

// ShutdownTimeout is the duration for which native applications wait for futures returned from
// on_app_quit handlers before fully quitting.
const ShutdownTimeout = 200 * time.Millisecond

var trackThreadBorrows = func() bool {
	_, ok := os.LookupEnv("TRACK_THREAD_BORROWS")
	return ok
}()

// AppCell wraps the App with upstream's borrow tracking, to help debug double borrows.
type AppCell struct {
	app          App
	borrowFlag   int
	borrowOrigin Option[appBorrowOrigin]
}

type appBorrowOrigin struct {
	location gpuiutil.Location
	kind     string
}

type appBorrowError struct {
	attempted gpuiutil.Location
	kind      string
	origin    Option[appBorrowOrigin]
}

func formatAppBorrowLocation(location gpuiutil.Location) string {
	return fmt.Sprintf("%s:%d", location.File, location.Line)
}

func (e *appBorrowError) Error() string {
	message := fmt.Sprintf("RefCell already borrowed: attempted %s App borrow at %s; ", e.kind, formatAppBorrowLocation(e.attempted))
	if origin, ok := e.origin.Get(); ok {
		return message + fmt.Sprintf("existing %s borrow group began at %s", origin.kind, formatAppBorrowLocation(origin.location))
	}
	return message + "origin unavailable"
}

type appBorrowGuard struct {
	cell     *AppCell
	released bool
}

// Borrow starts a shared borrow of the App; Release on the guard ends it. It panics when the
// App is mutably borrowed.
func (c *AppCell) Borrow() AppRef {
	if trackThreadBorrows {
		fmt.Fprintf(os.Stderr, "borrowed ThreadId(%d)\n", goroutineID())
	}
	startsBorrowGroup := c.borrowFlag == 0
	attempted := gpuiutil.CallerLocation(1)
	if c.borrowFlag < 0 {
		panic((&appBorrowError{attempted: attempted, kind: "shared", origin: c.borrowOrigin}).Error())
	}
	c.borrowFlag++
	if startsBorrowGroup {
		c.borrowOrigin = Some(appBorrowOrigin{location: attempted, kind: "shared"})
	}
	return AppRef{app: &c.app, guard: &appBorrowGuard{cell: c}}
}

// BorrowMut starts a mutable borrow of the App and panics on a conflicting borrow.
func (c *AppCell) BorrowMut() AppRefMut {
	borrow, err := c.tryBorrowMut(gpuiutil.CallerLocation(1))
	if err != nil {
		panic(err.Error())
	}
	return borrow
}

// TryBorrowMut starts a mutable borrow of the App or returns the borrow error.
func (c *AppCell) TryBorrowMut() (AppRefMut, error) {
	return c.tryBorrowMut(gpuiutil.CallerLocation(1))
}

func (c *AppCell) tryBorrowMut(attempted gpuiutil.Location) (AppRefMut, error) {
	if trackThreadBorrows {
		fmt.Fprintf(os.Stderr, "borrowed ThreadId(%d)\n", goroutineID())
	}
	if c.borrowFlag != 0 {
		return AppRefMut{}, &appBorrowError{attempted: attempted, kind: "mutable", origin: c.borrowOrigin}
	}
	c.borrowFlag = -1
	c.borrowOrigin = Some(appBorrowOrigin{location: attempted, kind: "mutable"})
	return AppRefMut{App: &c.app, guard: &appBorrowGuard{cell: c}}, nil
}

// AppRef is a shared borrow of the App. It does not embed *App, so a shared borrow implements
// neither AppContext nor BorrowAppContext.
type AppRef struct {
	app   *App
	guard *appBorrowGuard
}

// Get returns the borrowed App for reads only, as upstream's Deref to &App; mutation needs
// BorrowMut.
func (r AppRef) Get() *App {
	if r.guard == nil || r.guard.released {
		panic("gpui: AppRef used after Release")
	}
	return r.app
}

// Release ends the borrow, as the guard's Drop does upstream.
func (r AppRef) Release() {
	if r.guard == nil || r.guard.released {
		return
	}
	r.guard.released = true
	if trackThreadBorrows {
		fmt.Fprintf(os.Stderr, "dropped borrow from ThreadId(%d)\n", goroutineID())
	}
	r.guard.cell.borrowFlag--
}

// AppRefMut is a mutable borrow of the App.
type AppRefMut struct {
	*App
	guard *appBorrowGuard
}

// Release ends the borrow, as the guard's Drop does upstream.
func (r AppRefMut) Release() {
	if r.guard == nil || r.guard.released {
		return
	}
	r.guard.released = true
	if trackThreadBorrows {
		fmt.Fprintf(os.Stderr, "dropped ThreadId(%d)\n", goroutineID())
	}
	r.guard.cell.borrowFlag = 0
}

// Application is a reference to a GPUI application, typically constructed in the main function
// of your app.
type Application struct {
	app *AppCell
}

var retainedApplications []*Application

// ApplicationHandle is a strong handle to an Application started with RunEmbedded. Releasing
// it releases the app, so an embedder must hold it for as long as the app should run.
type ApplicationHandle struct {
	app *AppCell
}

// Update invokes f with the app context. It must not be called re-entrantly from code that is
// already inside an update.
func (h *ApplicationHandle) Update(f func(cx *App)) {
	cx := h.app.BorrowMut()
	defer cx.Release()
	f(cx.App)
}

// UpdateR invokes f with the app context and returns its result.
func (h *ApplicationHandle) UpdateR[R any](f func(cx *App) R) R {
	cx := h.app.BorrowMut()
	defer cx.Release()
	return f(cx.App)
}

// ToAsync returns an AsyncApp for use across await points. It holds the app weakly.
func (h *ApplicationHandle) ToAsync() *AsyncApp {
	return h.UpdateR(func(cx *App) *AsyncApp { return cx.ToAsync() })
}

// ApplicationWithPlatform builds an app with a caller-provided platform implementation.
func ApplicationWithPlatform(platform Platform) *Application {
	return &Application{app: newApp(platform, NoAssets{}, NullHTTPClient{})}
}

// NewApplicationInaccessible builds an app with accessibility integration forcibly disabled.
func NewApplicationInaccessible(platform Platform) *Application {
	this := ApplicationWithPlatform(platform)
	borrow := this.app.BorrowMut()
	borrow.accessibilityForceDisabled = true
	borrow.Release()
	return this
}

// WithAssets assigns the source of assets for the application.
func (a *Application) WithAssets(assetSource AssetSource) *Application {
	contextLock := a.app.BorrowMut()
	contextLock.assetSource = assetSource
	contextLock.svgRenderer = NewSvgRenderer(assetSource)
	contextLock.Release()
	return a
}

// WithRestartArguments configures arguments to pass when restarting the application.
func (a *Application) WithRestartArguments(arguments []string) *Application {
	borrow := a.app.BorrowMut()
	borrow.restartArguments = arguments
	borrow.Release()
	return a
}

// WithHTTPClient sets the HTTP client for the application.
func (a *Application) WithHTTPClient(httpClient HTTPClient) *Application {
	contextLock := a.app.BorrowMut()
	contextLock.httpClient = httpClient
	contextLock.Release()
	return a
}

// WithQuitMode configures when the application should automatically quit.
func (a *Application) WithQuitMode(mode QuitMode) *Application {
	borrow := a.app.BorrowMut()
	borrow.quitMode = mode
	borrow.Release()
	return a
}

// WithWindowing sets the windowing mode the app starts in. See App.RequestWindowing.
func (a *Application) WithWindowing(request WindowingRequest) *Application {
	borrow := a.app.Borrow()
	borrow.Get().platform.SetInitialWindowing(request)
	borrow.Release()
	return a
}

// Run starts the application. onFinishLaunching is called once the app is fully launched.
func (a *Application) Run(onFinishLaunching func(cx *App)) {
	this := a.app
	borrow := a.app.Borrow()
	platform := borrow.Get().platform
	borrow.Release()
	platform.Run(func() {
		cx := this.BorrowMut()
		defer cx.Release()
		cx.displaysChanged()
		onFinishLaunching(cx.App)
	})
	if runtime.GOOS == "js" {
		retainedApplications = append(retainedApplications, a)
	}
	runtime.KeepAlive(a)
}

// RunEmbedded starts the application for an embedder that drives the run loop itself and
// returns the handle that keeps the app alive.
func (a *Application) RunEmbedded(onFinishLaunching func(cx *App)) *ApplicationHandle {
	this := a.app
	borrow := a.app.Borrow()
	platform := borrow.Get().platform
	borrow.Release()
	platform.Run(func() {
		cx := this.BorrowMut()
		defer cx.Release()
		cx.displaysChanged()
		onFinishLaunching(cx.App)
	})
	return &ApplicationHandle{app: a.app}
}

// OnOpenURLs registers a handler invoked when the platform asks the application to open URLs.
func (a *Application) OnOpenURLs(callback func(urls []string)) *Application {
	borrow := a.app.Borrow()
	borrow.Get().platform.OnOpenURLs(callback)
	borrow.Release()
	return a
}

// OnReopen invokes a handler when an already-running application is launched.
func (a *Application) OnReopen(callback func(cx *App)) *Application {
	this := weak.Make(a.app)
	borrow := a.app.BorrowMut()
	borrow.platform.OnReopen(func() {
		if app := this.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			callback(cx.App)
		}
	})
	borrow.Release()
	return a
}

// BackgroundExecutor returns the app's background executor.
func (a *Application) BackgroundExecutor() BackgroundExecutor {
	borrow := a.app.Borrow()
	defer borrow.Release()
	return borrow.Get().bgExecutor
}

// ForegroundExecutor returns the app's foreground executor.
func (a *Application) ForegroundExecutor() ForegroundExecutor {
	borrow := a.app.Borrow()
	defer borrow.Release()
	return borrow.Get().foregroundExecutor
}

// TextSystem returns the app's text system.
func (a *Application) TextSystem() *TextSystem {
	borrow := a.app.Borrow()
	defer borrow.Release()
	return borrow.Get().textSystem
}

// PathForAuxiliaryExecutable returns the path of the executable with the given name in the
// application bundle.
func (a *Application) PathForAuxiliaryExecutable(name string) (string, error) {
	borrow := a.app.Borrow()
	defer borrow.Release()
	return borrow.Get().PathForAuxiliaryExecutable(name)
}

type handler func(cx *App) bool

type appLifecycleHandler func(phase AppLifecyclePhase, cx *App) bool

type displayHandler func(event DisplayEvent, cx *App) bool

type displayState struct {
	bounds          Bounds[Pixels]
	refreshInterval Option[time.Duration]
}

func (s displayState) changesFrom(previous displayState) DisplayChanges {
	var changes DisplayChanges
	if s.bounds != previous.bounds {
		changes |= DisplayChangesBounds
	}
	if s.refreshInterval != previous.refreshInterval {
		changes |= DisplayChangesRefreshInterval
	}
	return changes
}

func readDisplays(platform Platform) map[DisplayID]displayState {
	displays := map[DisplayID]displayState{}
	for _, display := range platform.Displays() {
		state := displayState{
			bounds:          display.Bounds(),
			refreshInterval: OptionFrom(display.RefreshInterval()),
		}
		displays[display.ID()] = state
	}
	return displays
}

func displayEvents(previous, current map[DisplayID]displayState) []DisplayEvent {
	var events []DisplayEvent
	for _, id := range slices.Sorted(maps.Keys(previous)) {
		if _, ok := current[id]; !ok {
			events = append(events, DisplayEventRemoved{ID: id})
		}
	}
	for _, id := range slices.Sorted(maps.Keys(current)) {
		state := current[id]
		previousState, ok := previous[id]
		if !ok {
			events = append(events, DisplayEventAdded{ID: id})
			continue
		}
		if changes := state.changesFrom(previousState); changes != 0 {
			events = append(events, DisplayEventChanged{ID: id, Changes: changes})
		}
	}
	return events
}

type listener func(event any, cx *App) bool

type missingGlyphCallback func(missingGlyphs []MissingGlyph, cx *App)

type keystrokeObserver func(event *KeystrokeEvent, window *Window, cx *App) bool

type quitHandler func(cx *App) scheduler.Future[struct{}]

type windowClosedHandler func(cx *App, id WindowID)

type releaseListener func(entity any, cx *App)

type newEntityListener func(entity AnyEntity, window *Window, cx *App)

type eventListener struct {
	eventType reflect.Type
	listener  listener
}

type missingGlyphRegistration struct {
	_ byte
}

type missingGlyphCallbackEntry struct {
	registration *missingGlyphRegistration
	callback     Option[missingGlyphCallback]
}

type missingGlyphCallbackSlot struct {
	entry Option[missingGlyphCallbackEntry]
}

func (s *missingGlyphCallbackSlot) replace(callback missingGlyphCallback) *missingGlyphRegistration {
	registration := &missingGlyphRegistration{}
	s.entry.Replace(missingGlyphCallbackEntry{
		registration: registration,
		callback:     Some(callback),
	})
	return registration
}

func (s *missingGlyphCallbackSlot) invoke(missingGlyphs []MissingGlyph, cx *App) {
	entry := s.entry.AsMut()
	if entry == nil {
		return
	}
	callback, ok := entry.callback.Take().Get()
	if !ok {
		return
	}
	registration := entry.registration
	callback(missingGlyphs, cx)

	current := s.entry.AsMut()
	isCurrent := current != nil && current.registration == registration
	if isCurrent {
		current.callback = Some(callback)
	}
}

func (s *missingGlyphCallbackSlot) remove(registration *missingGlyphRegistration) bool {
	entry := s.entry.AsMut()
	isCurrent := entry != nil && entry.registration == registration
	if isCurrent {
		s.entry.Take()
	}
	return isCurrent
}

// QuitMode defines when the application should automatically quit.
type QuitMode uint8

const (
	// QuitModeDefault uses QuitModeExplicit on macOS and QuitModeLastWindowClosed on other platforms.
	QuitModeDefault QuitMode = iota
	// QuitModeLastWindowClosed quits automatically when the last window is closed.
	QuitModeLastWindowClosed
	// QuitModeExplicit quits only when requested via App.Quit.
	QuitModeExplicit
)

// CursorHideMode controls when GPUI hides the mouse cursor in response to keyboard input. Its
// default is CursorHideModeOnTypingAndAction.
type CursorHideMode uint8

const (
	// CursorHideModeNever never hides the cursor automatically.
	CursorHideModeNever CursorHideMode = iota
	// CursorHideModeOnTyping hides on character-producing key presses.
	CursorHideModeOnTyping
	// CursorHideModeOnTypingAndAction hides on character-producing key presses and when a key
	// binding resolves to an action that consumes the keystroke.
	CursorHideModeOnTypingAndAction
)

// SystemWindowTab is a window tab of the system tab bar.
type SystemWindowTab struct {
	ID           WindowID
	Title        SharedString
	Handle       AnyWindowHandle
	LastActiveAt scheduler.Instant
}

// NewSystemWindowTab creates a new instance of the window tab.
func NewSystemWindowTab(title SharedString, handle AnyWindowHandle) SystemWindowTab {
	return SystemWindowTab{
		ID:           handle.id,
		Title:        title,
		Handle:       handle,
		LastActiveAt: scheduler.InstantNow(),
	}
}

func (t SystemWindowTab) equal(other SystemWindowTab) bool {
	return t.ID == other.ID && t.Title == other.Title && t.Handle == other.Handle && t.LastActiveAt.Equal(other.LastActiveAt)
}

// SystemWindowTabController manages window tabs; it is a global.
type SystemWindowTabController struct {
	visible   Option[bool]
	tabGroups map[int][]SystemWindowTab
}

// NewSystemWindowTabController creates a new instance of the window tab controller.
func NewSystemWindowTabController() SystemWindowTabController {
	return SystemWindowTabController{
		visible:   None[bool](),
		tabGroups: map[int][]SystemWindowTab{},
	}
}

// SystemWindowTabControllerInit initializes the global window tab controller.
func SystemWindowTabControllerInit(cx *App) {
	cx.SetGlobal(NewSystemWindowTabController())
}

// TabGroups returns all tab groups.
func (c *SystemWindowTabController) TabGroups() map[int][]SystemWindowTab {
	return c.tabGroups
}

func (c *SystemWindowTabController) sortedGroupIDs() []int {
	return slices.Sorted(maps.Keys(c.tabGroups))
}

func mostRecentlyActiveTab(tabs []SystemWindowTab) (AnyWindowHandle, bool) {
	if len(tabs) == 0 {
		return AnyWindowHandle{}, false
	}
	best := 0
	for i := 1; i < len(tabs); i++ {
		if !tabs[i].LastActiveAt.Before(tabs[best].LastActiveAt) {
			best = i
		}
	}
	return tabs[best].Handle, true
}

func (c *SystemWindowTabController) currentGroup(id WindowID, groupIDs []int) (int, bool) {
	for _, group := range groupIDs {
		if slices.ContainsFunc(c.tabGroups[group], func(tab SystemWindowTab) bool { return tab.ID == id }) {
			return group, true
		}
	}
	return 0, false
}

// SystemWindowTabControllerGetNextTabGroupWindow returns the window handle of the next tab group.
func SystemWindowTabControllerGetNextTabGroupWindow(cx *App, id WindowID) (AnyWindowHandle, bool) {
	controller := Global[SystemWindowTabController](cx)
	groupIDs := controller.sortedGroupIDs()
	currentGroup, ok := controller.currentGroup(id, groupIDs)
	if !ok {
		return AnyWindowHandle{}, false
	}
	idx := slices.Index(groupIDs, currentGroup)
	if idx < 0 {
		return AnyWindowHandle{}, false
	}
	nextIdx := (idx + 1) % len(groupIDs)
	tabs, ok := controller.tabGroups[groupIDs[nextIdx]]
	if !ok {
		return AnyWindowHandle{}, false
	}
	return mostRecentlyActiveTab(tabs)
}

// SystemWindowTabControllerGetPrevTabGroupWindow returns the window handle of the previous tab group.
func SystemWindowTabControllerGetPrevTabGroupWindow(cx *App, id WindowID) (AnyWindowHandle, bool) {
	controller := Global[SystemWindowTabController](cx)
	groupIDs := controller.sortedGroupIDs()
	currentGroup, ok := controller.currentGroup(id, groupIDs)
	if !ok {
		return AnyWindowHandle{}, false
	}
	idx := slices.Index(groupIDs, currentGroup)
	if idx < 0 {
		return AnyWindowHandle{}, false
	}
	prevIdx := idx - 1
	if idx == 0 {
		prevIdx = len(groupIDs) - 1
	}
	tabs, ok := controller.tabGroups[groupIDs[prevIdx]]
	if !ok {
		return AnyWindowHandle{}, false
	}
	return mostRecentlyActiveTab(tabs)
}

// Tabs returns all tabs in the same window as id.
func (c *SystemWindowTabController) Tabs(id WindowID) ([]SystemWindowTab, bool) {
	for _, group := range c.sortedGroupIDs() {
		tabs := c.tabGroups[group]
		if slices.ContainsFunc(tabs, func(tab SystemWindowTab) bool { return tab.ID == id }) {
			return tabs, true
		}
	}
	return nil, false
}

// SystemWindowTabControllerInitVisible initializes the visibility of the system window tab controller.
func SystemWindowTabControllerInitVisible(cx *App, visible bool) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	if controller.visible.IsNone() {
		controller.visible = Some(visible)
	}
}

// IsVisible returns the visibility of the system window tab controller.
func (c *SystemWindowTabController) IsVisible() bool {
	return c.visible.UnwrapOr(false)
}

// SystemWindowTabControllerSetVisible sets the visibility of the system window tab controller.
func SystemWindowTabControllerSetVisible(cx *App, visible bool) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	controller.visible = Some(visible)
}

// SystemWindowTabControllerUpdateLastActive updates the last active time of a window.
func SystemWindowTabControllerUpdateLastActive(cx *App, id WindowID) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	for _, windows := range controller.tabGroups {
		for i := range windows {
			if windows[i].ID == id {
				windows[i].LastActiveAt = scheduler.InstantNow()
			}
		}
	}
}

// SystemWindowTabControllerUpdateTabPosition updates the position of a tab within its group.
func SystemWindowTabControllerUpdateTabPosition(cx *App, id WindowID, ix int) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	for _, group := range controller.sortedGroupIDs() {
		windows := controller.tabGroups[group]
		currentPos := slices.IndexFunc(windows, func(tab SystemWindowTab) bool { return tab.ID == id })
		if currentPos < 0 {
			continue
		}
		if ix < len(windows) && currentPos != ix {
			windowTab := windows[currentPos]
			windows = slices.Delete(windows, currentPos, currentPos+1)
			windows = slices.Insert(windows, ix, windowTab)
			controller.tabGroups[group] = windows
		}
		break
	}
}

// SystemWindowTabControllerUpdateTabTitle updates the title of a tab.
func SystemWindowTabControllerUpdateTabTitle(cx *App, id WindowID, title SharedString) {
	controller := Global[SystemWindowTabController](cx)
	found := false
	sameTitle := false
	for _, group := range controller.sortedGroupIDs() {
		for _, tab := range controller.tabGroups[group] {
			if tab.ID == id {
				found = true
				sameTitle = tab.Title == title
				break
			}
		}
		if found {
			break
		}
	}
	if !found || sameTitle {
		return
	}

	mutController := cx.GlobalMut[SystemWindowTabController]()
	for _, group := range mutController.sortedGroupIDs() {
		windows := mutController.tabGroups[group]
		for i := range windows {
			if windows[i].ID == id {
				windows[i].Title = title
				return
			}
		}
	}
}

// SystemWindowTabControllerAddTab inserts a tab into a tab group.
func SystemWindowTabControllerAddTab(cx *App, id WindowID, tabs []SystemWindowTab) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	tabIndex := slices.IndexFunc(tabs, func(tab SystemWindowTab) bool { return tab.ID == id })
	if tabIndex < 0 {
		return
	}
	tab := tabs[tabIndex]

	var expectedTabIDs []WindowID
	for _, t := range tabs {
		if t.ID != id {
			expectedTabIDs = append(expectedTabIDs, t.ID)
		}
	}
	slices.Sort(expectedTabIDs)

	tabGroupID := None[int]()
	for _, groupID := range controller.sortedGroupIDs() {
		var tabIDs []WindowID
		for _, t := range controller.tabGroups[groupID] {
			tabIDs = append(tabIDs, t.ID)
		}
		slices.Sort(tabIDs)
		if slices.Equal(tabIDs, expectedTabIDs) {
			tabGroupID = Some(groupID)
			break
		}
	}

	if groupID, ok := tabGroupID.Get(); ok {
		if groupTabs, ok := controller.tabGroups[groupID]; ok {
			controller.tabGroups[groupID] = append(groupTabs, tab)
		}
	} else {
		newGroupID := len(controller.tabGroups)
		controller.tabGroups[newGroupID] = tabs
	}
}

// SystemWindowTabControllerRemoveTab removes a tab from its tab group.
func SystemWindowTabControllerRemoveTab(cx *App, id WindowID) (SystemWindowTab, bool) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	removedTab := None[SystemWindowTab]()

	for _, group := range controller.sortedGroupIDs() {
		tabs := controller.tabGroups[group]
		if pos := slices.IndexFunc(tabs, func(tab SystemWindowTab) bool { return tab.ID == id }); pos >= 0 {
			removedTab = Some(tabs[pos])
			tabs = slices.Delete(tabs, pos, pos+1)
			controller.tabGroups[group] = tabs
		}
		if len(tabs) == 0 {
			delete(controller.tabGroups, group)
		}
	}

	return removedTab.Get()
}

// SystemWindowTabControllerMoveTabToNewWindow moves a tab to a new tab group.
func SystemWindowTabControllerMoveTabToNewWindow(cx *App, id WindowID) {
	removedTab, ok := SystemWindowTabControllerRemoveTab(cx, id)
	controller := cx.GlobalMut[SystemWindowTabController]()

	if ok {
		newGroupID := 0
		if len(controller.tabGroups) > 0 {
			newGroupID = slices.Max(slices.Collect(maps.Keys(controller.tabGroups))) + 1
		}
		controller.tabGroups[newGroupID] = []SystemWindowTab{removedTab}
	}
}

// SystemWindowTabControllerMergeAllWindows merges all tab groups into a single group.
func SystemWindowTabControllerMergeAllWindows(cx *App, id WindowID) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	initialTabs, ok := controller.Tabs(id)
	if !ok {
		return
	}

	initialTabsLen := len(initialTabs)
	allTabs := slices.Clone(initialTabs)

	groupIDs := controller.sortedGroupIDs()
	drained := make([][]SystemWindowTab, 0, len(groupIDs))
	for _, group := range groupIDs {
		drained = append(drained, controller.tabGroups[group])
	}
	clear(controller.tabGroups)
	for _, tabs := range drained {
		tabs = slices.DeleteFunc(slices.Clone(tabs), func(tab SystemWindowTab) bool {
			return slices.ContainsFunc(allTabs[:initialTabsLen], tab.equal)
		})
		allTabs = append(allTabs, tabs...)
	}

	controller.tabGroups[0] = allTabs
}

// SystemWindowTabControllerSelectNextTab selects the next tab in the tab group in the trailing direction.
func SystemWindowTabControllerSelectNextTab(cx *App, id WindowID) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	tabs, ok := controller.Tabs(id)
	if !ok {
		return
	}

	currentIndex := slices.IndexFunc(tabs, func(tab SystemWindowTab) bool { return tab.ID == id })
	if currentIndex < 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	nextIndex := (currentIndex + 1) % len(tabs)

	_ = tabs[nextIndex].Handle.Update(cx, func(_ AnyView, window *Window, _ *App) {
		window.ActivateWindow()
	})
}

// SystemWindowTabControllerSelectPreviousTab selects the previous tab in the tab group in the leading direction.
func SystemWindowTabControllerSelectPreviousTab(cx *App, id WindowID) {
	controller := cx.GlobalMut[SystemWindowTabController]()
	tabs, ok := controller.Tabs(id)
	if !ok {
		return
	}

	currentIndex := slices.IndexFunc(tabs, func(tab SystemWindowTab) bool { return tab.ID == id })
	if currentIndex < 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	previousIndex := currentIndex - 1
	if currentIndex == 0 {
		previousIndex = len(tabs) - 1
	}

	_ = tabs[previousIndex].Handle.Update(cx, func(_ AnyView, window *Window, _ *App) {
		window.ActivateWindow()
	})
}

type gpuiMode struct {
	test        bool
	skipDrawing bool
}

var gpuiModeProduction = gpuiMode{}

func gpuiModeTest() gpuiMode {
	return gpuiMode{test: true, skipDrawing: false}
}

func (m gpuiMode) shouldSkipDrawing() bool {
	if m.test {
		return m.skipDrawing
	}
	return false
}

type platformOwnedDrag struct {
	sourceWindow WindowID
	state        platformOwnedDragState
}

type platformOwnedDragState interface {
	isPlatformOwnedDragState()
}

type platformOwnedDragStateSuspended struct {
	drag AnyDrag
}

type platformOwnedDragStateRestoredInSourceWindow struct{}

func (platformOwnedDragStateSuspended) isPlatformOwnedDragState()              {}
func (platformOwnedDragStateRestoredInSourceWindow) isPlatformOwnedDragState() {}

type windowSlot struct {
	version  uint32
	nextFree uint32
	window   *Window
}

type windowSlotMap struct {
	slots    []windowSlot
	freeHead uint32
	numElems uint32
}

func newWindowSlotMap() windowSlotMap {
	return windowSlotMap{slots: []windowSlot{{}}, freeHead: 1}
}

func windowIDFromKey(idx, version uint32) WindowID {
	return WindowID(uint64(version|1)<<32 | uint64(idx))
}

func (m *windowSlotMap) insert(window *Window) WindowID {
	if m.numElems+1 == math.MaxUint32 {
		panic("SlotMap number of elements overflow")
	}
	if int(m.freeHead) < len(m.slots) {
		idx := m.freeHead
		slot := &m.slots[idx]
		occupiedVersion := slot.version | 1
		m.freeHead = slot.nextFree
		slot.window = window
		slot.version = occupiedVersion
		m.numElems++
		return windowIDFromKey(idx, occupiedVersion)
	}
	idx := uint32(len(m.slots))
	m.slots = append(m.slots, windowSlot{version: 1, window: window})
	m.freeHead = idx + 1
	m.numElems++
	return windowIDFromKey(idx, 1)
}

func (m *windowSlotMap) containsKey(id WindowID) bool {
	idx := uint32(id)
	return int(idx) < len(m.slots) && m.slots[idx].version == uint32(id>>32)|1
}

func (m *windowSlotMap) get(id WindowID) (*Window, bool) {
	if !m.containsKey(id) {
		return nil, false
	}
	return m.slots[uint32(id)].window, true
}

func (m *windowSlotMap) set(id WindowID, window *Window) bool {
	if !m.containsKey(id) {
		return false
	}
	m.slots[uint32(id)].window = window
	return true
}

func (m *windowSlotMap) remove(id WindowID) (*Window, bool) {
	if !m.containsKey(id) {
		return nil, false
	}
	idx := uint32(id)
	slot := &m.slots[idx]
	window := slot.window
	slot.window = nil
	slot.nextFree = m.freeHead
	m.freeHead = idx
	slot.version++
	m.numElems--
	return window, true
}

func (m *windowSlotMap) len() int {
	return int(m.numElems)
}

func (m *windowSlotMap) isEmpty() bool {
	return m.numElems == 0
}

func (m *windowSlotMap) all() iter.Seq2[WindowID, *Window] {
	return func(yield func(WindowID, *Window) bool) {
		for idx := 1; idx < len(m.slots); idx++ {
			slot := m.slots[idx]
			if slot.version%2 == 0 {
				continue
			}
			if !yield(windowIDFromKey(uint32(idx), slot.version), slot.window) {
				return
			}
		}
	}
}

func (m *windowSlotMap) keys() iter.Seq[WindowID] {
	return func(yield func(WindowID) bool) {
		for id := range m.all() {
			if !yield(id) {
				return
			}
		}
	}
}

func (m *windowSlotMap) clear() {
	for idx := 1; idx < len(m.slots); idx++ {
		if m.slots[idx].version%2 == 1 {
			m.remove(windowIDFromKey(uint32(idx), m.slots[idx].version))
		}
	}
}

type loadingAssetKey struct {
	assetType reflect.Type
	hash      uint64
}

// App contains the state of the full application and is passed as a reference to a variety of
// callbacks. Context[T] embeds it. You need an App to access the state of an Entity.
type App struct {
	this       weak.Pointer[AppCell]
	platform   Platform
	textSystem *TextSystem

	actions                    *actionRegistry
	activeDrag                 Option[AnyDrag]
	platformOwnedDrag          Option[platformOwnedDrag]
	bgExecutor                 BackgroundExecutor
	foregroundExecutor         ForegroundExecutor
	foregroundJournal          ForegroundJournal
	hangMonitor                *hangMonitor
	entities                   *entityMap
	newEntityObservers         subscriberSet[reflect.Type, newEntityListener]
	windows                    windowSlotMap
	windowHandles              map[WindowID]AnyWindowHandle
	focusHandles               *focusMap
	keymap                     *Keymap
	keyboardLayout             PlatformKeyboardLayout
	keyboardMapper             PlatformKeyboardMapper
	globalActionListeners      map[reflect.Type][]func(action any, phase DispatchPhase, cx *App)
	pendingEffects             []effect
	observers                  subscriberSet[EntityID, handler]
	eventListeners             subscriberSet[EntityID, eventListener]
	keystrokeObservers         singleSubscriberSet[keystrokeObserver]
	keystrokeInterceptors      singleSubscriberSet[keystrokeObserver]
	keyboardLayoutObservers    singleSubscriberSet[handler]
	textScaleFactorObservers   singleSubscriberSet[handler]
	missingGlyphCallback       *missingGlyphCallbackSlot
	thermalStateObservers      singleSubscriberSet[handler]
	displayObservers           singleSubscriberSet[displayHandler]
	displays                   map[DisplayID]displayState
	systemSleepObservers       singleSubscriberSet[handler]
	systemWakeObservers        singleSubscriberSet[handler]
	appLifecycleObservers      singleSubscriberSet[appLifecycleHandler]
	memoryWarningObservers     singleSubscriberSet[handler]
	releaseListeners           subscriberSet[EntityID, releaseListener]
	globalObservers            subscriberSet[reflect.Type, handler]
	quitObservers              singleSubscriberSet[quitHandler]
	restartObservers           singleSubscriberSet[handler]
	windowClosedObservers      singleSubscriberSet[windowClosedHandler]
	globalsByType              map[reflect.Type]any
	loadingAssets              map[loadingAssetKey]any
	assetSource                AssetSource
	svgRenderer                SvgRenderer
	httpClient                 HTTPClient
	pendingNotifications       collections.HashSet[EntityID]
	pendingGlobalNotifications collections.TypeIDHashSet
	restartPath                Option[string]
	restartArguments           []string
	layoutIDBuffer             []LayoutID
	propagateEvent             bool
	promptBuilder              Option[promptBuilder]
	windowInvalidatorsByEntity map[EntityID]map[WindowID]windowInvalidator
	trackedEntities            map[WindowID]collections.HashSet[EntityID]
	currentWindowByEntity      map[EntityID]WindowID
	inspectorRenderer          Option[InspectorRenderer]
	inspectorElementRegistry   inspectorElementRegistry
	name                       Option[string]
	textRenderingMode          *TextRenderingMode
	windowUpdateStack          []windowUpdateEntry
	mode                       gpuiMode
	cursorHideMode             CursorHideMode
	reduceMotion               bool
	syncedAnimationEpoch       scheduler.Instant
	powerMode                  PowerMode
	accessibilityForceDisabled bool
	pendingUpdates             int
	quitMode                   QuitMode
	quitting                   bool
	refCounts                  *entityRefCounts
}

type windowUpdateEntry struct {
	windowID  WindowID
	bounds    WindowBounds
	displayID Option[DisplayID]
}

func newApp(platform Platform, assetSource AssetSource, httpClient HTTPClient) *AppCell {
	backgroundExecutor := platform.BackgroundExecutor()
	foregroundExecutor := platform.ForegroundExecutor()
	if !backgroundExecutor.IsMainThread() {
		panic("must construct App on main thread")
	}
	var foregroundJournal ForegroundJournal
	if profilerFeatureEnabled {
		foregroundJournal = installForegroundJournal()
	}
	syncedAnimationEpoch := backgroundExecutor.Now()

	textSystem := NewTextSystem(platform.TextSystem())
	entities := newEntityMap()
	keyboardLayout := platform.KeyboardLayout()
	keyboardMapper := platform.KeyboardMapper()
	powerMode := platform.PowerMode()

	refCounts := entities.refCountsDropHandle()

	textRenderingMode := TextRenderingModePlatformDefault
	app := &AppCell{borrowOrigin: None[appBorrowOrigin]()}
	app.app = App{
		this:                       weak.Make(app),
		platform:                   platform,
		textSystem:                 textSystem,
		textRenderingMode:          &textRenderingMode,
		mode:                       gpuiModeProduction,
		actions:                    newActionRegistry(),
		pendingUpdates:             0,
		activeDrag:                 None[AnyDrag](),
		platformOwnedDrag:          None[platformOwnedDrag](),
		bgExecutor:                 backgroundExecutor,
		foregroundExecutor:         foregroundExecutor,
		foregroundJournal:          foregroundJournal,
		hangMonitor:                nil,
		svgRenderer:                NewSvgRenderer(assetSource),
		loadingAssets:              map[loadingAssetKey]any{},
		assetSource:                assetSource,
		httpClient:                 httpClient,
		globalsByType:              map[reflect.Type]any{},
		entities:                   entities,
		newEntityObservers:         newSubscriberSet[reflect.Type, newEntityListener](),
		windows:                    newWindowSlotMap(),
		windowUpdateStack:          nil,
		windowHandles:              map[WindowID]AnyWindowHandle{},
		focusHandles:               defaultFocusMap(),
		keymap:                     DefaultKeymap(),
		keyboardLayout:             keyboardLayout,
		keyboardMapper:             keyboardMapper,
		globalActionListeners:      map[reflect.Type][]func(action any, phase DispatchPhase, cx *App){},
		pendingEffects:             nil,
		pendingNotifications:       collections.HashSet[EntityID]{},
		pendingGlobalNotifications: collections.TypeIDHashSet{},
		observers:                  newSubscriberSet[EntityID, handler](),
		trackedEntities:            map[WindowID]collections.HashSet[EntityID]{},
		windowInvalidatorsByEntity: map[EntityID]map[WindowID]windowInvalidator{},
		currentWindowByEntity:      map[EntityID]WindowID{},
		eventListeners:             newSubscriberSet[EntityID, eventListener](),
		releaseListeners:           newSubscriberSet[EntityID, releaseListener](),
		keystrokeObservers:         newSingleSubscriberSet[keystrokeObserver](),
		keystrokeInterceptors:      newSingleSubscriberSet[keystrokeObserver](),
		keyboardLayoutObservers:    newSingleSubscriberSet[handler](),
		textScaleFactorObservers:   newSingleSubscriberSet[handler](),
		missingGlyphCallback:       &missingGlyphCallbackSlot{entry: None[missingGlyphCallbackEntry]()},
		thermalStateObservers:      newSingleSubscriberSet[handler](),
		displayObservers:           newSingleSubscriberSet[displayHandler](),
		displays:                   readDisplays(platform),
		systemSleepObservers:       newSingleSubscriberSet[handler](),
		systemWakeObservers:        newSingleSubscriberSet[handler](),
		appLifecycleObservers:      newSingleSubscriberSet[appLifecycleHandler](),
		memoryWarningObservers:     newSingleSubscriberSet[handler](),
		globalObservers:            newSubscriberSet[reflect.Type, handler](),
		quitObservers:              newSingleSubscriberSet[quitHandler](),
		restartObservers:           newSingleSubscriberSet[handler](),
		restartPath:                None[string](),
		restartArguments:           nil,
		windowClosedObservers:      newSingleSubscriberSet[windowClosedHandler](),
		layoutIDBuffer:             nil,
		propagateEvent:             true,
		promptBuilder:              Some[promptBuilder](promptBuilderDefault{}),
		inspectorRenderer:          None[InspectorRenderer](),
		inspectorElementRegistry:   defaultInspectorElementRegistry(),
		quitMode:                   QuitModeDefault,
		quitting:                   false,
		cursorHideMode:             CursorHideModeOnTypingAndAction,
		reduceMotion:               false,
		syncedAnimationEpoch:       syncedAnimationEpoch,
		powerMode:                  powerMode,
		accessibilityForceDisabled: false,
		name:                       None[string](),
		refCounts:                  refCounts,
	}

	func() {
		borrow := app.Borrow()
		defer borrow.Release()
		initAppMenus(platform, borrow.Get())
	}()
	func() {
		borrow := app.BorrowMut()
		defer borrow.Release()
		SystemWindowTabControllerInit(borrow.App)
	}()
	if profilerFeatureEnabled {
		func() {
			borrow := app.Borrow()
			defer borrow.Release()
			observePower(borrow.Get())
		}()
	}

	weakApp := weak.Make(app)

	platform.OnKeyboardLayoutChange(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.keyboardLayout = cx.platform.KeyboardLayout()
			cx.keyboardMapper = cx.platform.KeyboardMapper()
			cx.keyboardLayoutObservers.retain(func(callback *handler) bool { return (*callback)(cx.App) })
		}
	})

	platform.OnTextScaleFactorChange(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.textScaleFactorObservers.retain(func(callback *handler) bool { return (*callback)(cx.App) })
		}
	})

	platform.OnThermalStateChange(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.thermalStateObservers.retain(func(callback *handler) bool { return (*callback)(cx.App) })
		}
	})

	platform.OnDisplaysChanged(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.displaysChanged()
		}
	})

	platform.OnSystemSleep(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.systemSleepObservers.retain(func(callback *handler) bool { return (*callback)(cx.App) })
		}
	})

	platform.OnSystemWake(func() {
		if app := weakApp.Value(); app != nil {
			cx := app.BorrowMut()
			defer cx.Release()
			cx.systemWakeObservers.retain(func(callback *handler) bool { return (*callback)(cx.App) })
		}
	})

	platform.OnAppLifecycle(func(phase AppLifecyclePhase) {
		if app := weakApp.Value(); app != nil {
			borrow := app.BorrowMut()
			defer borrow.Release()
			borrow.update(func(cx *App) {
				cx.appLifecycleObservers.retain(func(callback *appLifecycleHandler) bool { return (*callback)(phase, cx) })
			})
		}
	})

	platform.OnMemoryWarning(func() {
		if app := weakApp.Value(); app != nil {
			borrow := app.BorrowMut()
			defer borrow.Release()
			borrow.update(func(cx *App) {
				cx.memoryWarningObservers.retain(func(callback *handler) bool { return (*callback)(cx) })
			})
		}
	})

	platform.OnPowerModeChange(func() {
		if app := weakApp.Value(); app != nil {
			borrow := app.Borrow()
			powerMode := borrow.Get().platform.PowerMode()
			borrow.Release()
			borrowMut := app.BorrowMut()
			borrowMut.powerMode = powerMode
			borrowMut.Release()
		}
	})

	platform.OnQuit(func() bool {
		cx := weakApp.Value()
		if cx == nil {
			return true
		}
		borrow, err := cx.TryBorrowMut()
		if err != nil {
			return false
		}
		defer borrow.Release()
		borrow.Shutdown()
		return true
	})

	return app
}

// RefCountsDropHandle returns a handle that keeps the entity ref-count table alive.
func (cx *App) RefCountsDropHandle() any {
	return cx.entities.refCountsDropHandle()
}

// LeakDetectorSnapshot captures a snapshot of all entities that currently have alive handles.
func (cx *App) LeakDetectorSnapshot() LeakDetectorSnapshot {
	return cx.entities.leakDetectorSnapshot()
}

// AssertNoNewLeaks asserts that no entities created after snapshot still have alive handles.
func (cx *App) AssertNoNewLeaks(snapshot *LeakDetectorSnapshot) {
	cx.entities.assertNoNewLeaks(snapshot)
}

func hangMonitorFlushFuture(flushed <-chan struct{}) scheduler.Future[struct{}] {
	var mu sync.Mutex
	var waker scheduler.Waker
	started := false
	return scheduler.PollFn(func(cx *scheduler.Context) (struct{}, bool) {
		select {
		case <-flushed:
			return struct{}{}, true
		default:
		}
		mu.Lock()
		waker = cx.Waker().Clone()
		start := !started
		started = true
		mu.Unlock()
		if start {
			go func() {
				<-flushed
				mu.Lock()
				w := waker
				mu.Unlock()
				w.Wake()
			}()
		}
		return struct{}{}, false
	})
}

// Shutdown quits the application gracefully. Native applications give on_app_quit handlers
// ShutdownTimeout to complete; WebAssembly runs them asynchronously.
func (cx *App) Shutdown() {
	var hangMonitorFlush <-chan struct{}
	hasHangMonitorFlush := false
	if profilerFeatureEnabled && runtime.GOOS != "js" && cx.hangMonitor != nil {
		hangMonitorFlush, hasHangMonitorFlush = cx.hangMonitor.requestFlush()
	}

	var futures []scheduler.Future[struct{}]

	for observer := range cx.quitObservers.remove() {
		futures = append(futures, observer(cx))
	}

	if hasHangMonitorFlush {
		futures = append(futures, hangMonitorFlushFuture(hangMonitorFlush))
	}

	cx.dropWindows()
	clear(cx.windowHandles)
	cx.flushEffects()
	cx.quitting = true

	joined := scheduler.Async(func(s *scheduler.Scope) struct{} {
		scheduler.Join(s, futures...)
		return struct{}{}
	})
	if runtime.GOOS != "js" {
		if _, ok := cx.foregroundExecutor.BlockWithTimeout(ShutdownTimeout, joined); !ok {
			slog.Error("timed out waiting on app_will_quit")
		}
	} else {
		cx.foregroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
			return scheduler.Await(s, joined), nil
		}).Detach()
	}

	cx.quitting = false
}

func (cx *App) dropWindows() {
	for _, window := range cx.windows.all() {
		if window != nil {
			window.drop()
		}
	}
	cx.windows.clear()
}

// KeyboardLayout returns the current keyboard layout.
func (cx *App) KeyboardLayout() PlatformKeyboardLayout {
	return cx.keyboardLayout
}

// KeyboardMapper returns the current keyboard mapper.
func (cx *App) KeyboardMapper() PlatformKeyboardMapper {
	return cx.keyboardMapper
}

// OnKeyboardLayoutChange invokes a handler when the current keyboard layout changes.
func (cx *App) OnKeyboardLayoutChange(callback func(cx *App)) Subscription {
	return cx.keyboardLayoutObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

// OnTextScaleFactorChange invokes a handler when the text scale factor changes.
func (cx *App) OnTextScaleFactorChange(callback func(cx *App)) Subscription {
	return cx.textScaleFactorObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

// Quit gracefully quits the application via the platform's standard routine.
func (cx *App) Quit() {
	cx.platform.Quit()
}

// RequestWindowing switches the platform between headless and windowed modes. The returned task
// fails if any window is open or the platform cannot apply the switch.
func (cx *App) RequestWindowing(request WindowingRequest) *Task[struct{}] {
	if !cx.windows.isEmpty() {
		return scheduler.ReadyTaskErr[struct{}](errors.New("cannot switch windowing modes while windows are open"))
	}
	return cx.platform.RequestWindowing(request)
}

// GraphicalEnvironment returns the environment of the display server the platform is connected
// to, or false while headless.
func (cx *App) GraphicalEnvironment() (GraphicalEnvironment, bool) {
	return cx.platform.GraphicalEnvironment()
}

// CursorHideMode returns the current policy for hiding the cursor in response to keyboard input.
func (cx *App) CursorHideMode() CursorHideMode {
	return cx.cursorHideMode
}

// SetCursorHideMode sets the policy controlling when GPUI hides the cursor in response to
// keyboard input.
func (cx *App) SetCursorHideMode(mode CursorHideMode) {
	cx.cursorHideMode = mode
}

// IsCursorVisible returns whether the cursor is currently visible according to the platform.
func (cx *App) IsCursorVisible() bool {
	return cx.platform.IsCursorVisible()
}

// ReduceMotion returns whether non-essential animations should be rendered in a static state.
func (cx *App) ReduceMotion() bool {
	return cx.reduceMotion
}

// SetReduceMotion sets whether non-essential animations should be rendered in a static state.
func (cx *App) SetReduceMotion(reduceMotion bool) {
	if cx.reduceMotion != reduceMotion {
		cx.reduceMotion = reduceMotion
		cx.RefreshWindows()
	}
}

// RefreshWindows schedules all windows in the application to be redrawn.
func (cx *App) RefreshWindows() {
	cx.pendingEffects = append(cx.pendingEffects, effectRefreshWindows{})
}

func (cx *App) update(update func(cx *App)) {
	cx.startUpdate()
	update(cx)
	cx.finishUpdate()
}

func (cx *App) updateR[R any](update func(cx *App) R) R {
	cx.startUpdate()
	result := update(cx)
	cx.finishUpdate()
	return result
}

func (cx *App) startUpdate() {
	cx.pendingUpdates++
}

func (cx *App) finishUpdate() {
	if cx.pendingUpdates == 1 {
		cx.flushEffects()
	}
	cx.pendingUpdates--
}

// Observe arranges for onNotify to be invoked when the given entity calls Notify on its context.
func (cx *App) Observe[W any](entity Entity[W], onNotify func(entity Entity[W], cx *App)) Subscription {
	return observeInternal(cx, entity, func(e Entity[W], cx *App) bool {
		onNotify(e, cx)
		return true
	})
}

func (cx *App) detectAccessedEntities[R any](callback func(cx *App) R) (R, collections.HashSet[EntityID]) {
	cx.entities.pushAccessScope()
	result := callback(cx)
	return result, cx.entities.popAccessScope()
}

func (cx *App) recordEntitiesAccessed(windowHandle AnyWindowHandle, invalidator windowInvalidator, entities collections.HashSet[EntityID]) {
	trackedEntities := cx.trackedEntities[windowHandle.id]
	if trackedEntities == nil {
		trackedEntities = collections.HashSet[EntityID]{}
	}
	cx.trackedEntities[windowHandle.id] = collections.HashSet[EntityID]{}
	for entity := range trackedEntities {
		if windows, ok := cx.windowInvalidatorsByEntity[entity]; ok {
			delete(windows, windowHandle.id)
		}
	}
	for entity := range entities {
		windows, ok := cx.windowInvalidatorsByEntity[entity]
		if !ok {
			windows = map[WindowID]windowInvalidator{}
			cx.windowInvalidatorsByEntity[entity] = windows
		}
		windows[windowHandle.id] = invalidator
		cx.currentWindowByEntity[entity] = windowHandle.id
	}
	clear(trackedEntities)
	for entity := range entities {
		trackedEntities[entity] = struct{}{}
	}
	cx.trackedEntities[windowHandle.id] = trackedEntities
}

func (cx *App) newObserver(key EntityID, value handler) Subscription {
	subscription, activate := cx.observers.insertPending(key, value)
	cx.Defer(func(*App) { activate() })
	return subscription
}

func observeInternal[W any](cx *App, entity Entity[W], onNotify func(entity Entity[W], cx *App) bool) Subscription {
	entityID := entity.EntityID()
	handle := entity.Downgrade()
	return cx.newObserver(entityID, func(cx *App) bool {
		entity, ok := handle.Upgrade()
		if !ok {
			return false
		}
		defer entity.Release()
		return onNotify(entity, cx)
	})
}

// Subscribe arranges for onEvent to be invoked whenever the given entity emits an event of type
// E. The callback is given a handle to the emitting entity and the emitted event.
func (cx *App) Subscribe[T, E any](entity Entity[T], onEvent func(entity Entity[T], event *E, cx *App)) Subscription {
	return subscribeInternal(cx, entity, func(entity Entity[T], event *E, cx *App) bool {
		onEvent(entity, event, cx)
		return true
	})
}

func (cx *App) newSubscription(key EntityID, value eventListener) Subscription {
	subscription, activate := cx.eventListeners.insertPending(key, value)
	cx.Defer(func(*App) { activate() })
	return subscription
}

func subscribeInternal[T, E any](cx *App, entity Entity[T], onEvent func(entity Entity[T], event *E, cx *App) bool) Subscription {
	entityID := entity.EntityID()
	handle := entity.Downgrade()
	return cx.newSubscription(entityID, eventListener{
		eventType: reflect.TypeFor[E](),
		listener: func(event any, cx *App) bool {
			typed, ok := event.(*E)
			if !ok {
				panic("invalid event type")
			}
			entity, ok := handle.Upgrade()
			if !ok {
				return false
			}
			defer entity.Release()
			return onEvent(entity, typed, cx)
		},
	})
}

// Windows returns handles to all open windows in the application.
func (cx *App) Windows() []AnyWindowHandle {
	var handles []AnyWindowHandle
	for windowID := range cx.windows.keys() {
		if handle, ok := cx.windowHandles[windowID]; ok {
			handles = append(handles, handle)
		}
	}
	return handles
}

// WindowStack returns the window handles ordered by their appearance on screen, front to back,
// or false if the platform does not implement it.
func (cx *App) WindowStack() ([]AnyWindowHandle, bool) {
	return cx.platform.WindowStack()
}

// ActiveWindow returns the window that is currently focused at the platform level.
func (cx *App) ActiveWindow() (AnyWindowHandle, bool) {
	return cx.platform.ActiveWindow()
}

// ActiveDisplay returns the display containing the window that is currently focused at the
// platform level, or nil.
func (cx *App) ActiveDisplay() PlatformDisplay {
	displayID, _, ok := cx.activeWindowPlacement()
	if !ok {
		return nil
	}
	id, ok := displayID.Get()
	if !ok {
		return nil
	}
	return cx.FindDisplay(id)
}

func (cx *App) activeWindowPlacement() (Option[DisplayID], WindowBounds, bool) {
	activeWindow, ok := cx.ActiveWindow()
	if !ok {
		return None[DisplayID](), WindowBounds{}, false
	}
	windowID := activeWindow.WindowID()

	for i := len(cx.windowUpdateStack) - 1; i >= 0; i-- {
		state := cx.windowUpdateStack[i]
		if state.windowID == windowID {
			return state.displayID, state.bounds, true
		}
	}

	window, ok := cx.windows.get(windowID)
	if !ok || window == nil {
		return None[DisplayID](), WindowBounds{}, false
	}
	return window.displayID, window.lastWindowBounds, true
}

func (cx *App) syncWindowUpdateEntry(window *Window) {
	for i := len(cx.windowUpdateStack) - 1; i >= 0; i-- {
		state := &cx.windowUpdateStack[i]
		if state.windowID == window.handle.id {
			state.bounds = window.lastWindowBounds
			state.displayID = window.displayID
			return
		}
	}
}

// OpenWindow opens a new window with the given options and the root view returned by
// buildRootView; *V must implement Render[V].
func (cx *App) OpenWindow[V any](options WindowOptions, buildRootView func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	var result WindowHandle[V]
	var resultErr error
	cx.update(func(cx *App) {
		id := cx.windows.insert(nil)
		handle := NewWindowHandle[V](id)
		window, err := newWindow(handle.AnyWindowHandle, options, cx)
		if err != nil {
			cx.windows.remove(id)
			resultErr = err
			return
		}
		cx.windowUpdateStack = append(cx.windowUpdateStack, windowUpdateEntry{
			windowID:  id,
			bounds:    window.lastWindowBounds,
			displayID: window.displayID,
		})
		rootView := buildRootView(window, cx)
		cx.windowUpdateStack = cx.windowUpdateStack[:len(cx.windowUpdateStack)-1]
		window.root.Replace(anyViewFromEntity(rootView))
		window.Defer(cx, func(window *Window, cx *App) { window.appearanceChanged(cx) })

		clearNeeded := window.Draw(cx)
		clearNeeded.Clear(cx)

		cx.windowHandles[id] = window.handle
		if !cx.windows.set(id, window) {
			panic("called `Option::unwrap()` on a `None` value")
		}
		result = handle
	})
	return result, resultErr
}

// Activate instructs the platform to bring the application to the foreground.
func (cx *App) Activate(ignoringOtherApps bool) {
	cx.platform.Activate(ignoringOtherApps)
}

// Hide hides the application at the platform level.
func (cx *App) Hide() {
	cx.platform.Hide()
}

// HideOtherApps hides other applications at the platform level.
func (cx *App) HideOtherApps() {
	cx.platform.HideOtherApps()
}

// UnhideOtherApps unhides other applications at the platform level.
func (cx *App) UnhideOtherApps() {
	cx.platform.UnhideOtherApps()
}

// Displays returns the list of currently active displays.
func (cx *App) Displays() []PlatformDisplay {
	return cx.platform.Displays()
}

// PrimaryDisplay returns the primary display that will be used for new windows, or nil.
func (cx *App) PrimaryDisplay() PlatformDisplay {
	return cx.platform.PrimaryDisplay()
}

// IsScreenCaptureSupported returns whether ScreenCaptureSources may work.
func (cx *App) IsScreenCaptureSupported() bool {
	return cx.platform.IsScreenCaptureSupported()
}

// ScreenCaptureSources returns a list of available screen capture sources.
func (cx *App) ScreenCaptureSources() *scheduler.OneshotReceiver[scheduler.Result[[]ScreenCaptureSource]] {
	return cx.platform.ScreenCaptureSources()
}

// FindDisplay returns the display with the given id, or nil.
func (cx *App) FindDisplay(id DisplayID) PlatformDisplay {
	for _, display := range cx.Displays() {
		if display.ID() == id {
			return display
		}
	}
	return nil
}

type debugStateDisplay struct {
	ID                     uint64          `json:"id"`
	UUID                   Option[string]  `json:"uuid"`
	Bounds                 Bounds[Pixels]  `json:"bounds"`
	VisibleBounds          Bounds[Pixels]  `json:"visible_bounds"`
	RefreshIntervalSeconds Option[float64] `json:"refresh_interval_seconds"`
}

type debugStateUpdatingWindow struct {
	ID       uint64 `json:"id"`
	Updating bool   `json:"updating"`
}

type debugStatePowerMode struct {
	Cached   string `json:"cached"`
	Platform string `json:"platform"`
}

type debugStateApp struct {
	Compositor       string              `json:"compositor"`
	ThermalState     string              `json:"thermal_state"`
	PowerMode        debugStatePowerMode `json:"power_mode"`
	WindowAppearance string              `json:"window_appearance"`
	CursorVisible    bool                `json:"cursor_visible"`
	ActiveWindowID   Option[uint64]      `json:"active_window_id"`
	WindowStack      Option[[]uint64]    `json:"window_stack"`
	PrimaryDisplayID Option[uint64]      `json:"primary_display_id"`
	Displays         []debugStateDisplay `json:"displays"`
	Windows          []jsontext.Value    `json:"windows"`
	Platform         jsontext.Value      `json:"platform"`
}

func thermalStateDebugName(state ThermalState) string {
	switch state {
	case ThermalStateNominal:
		return "Nominal"
	case ThermalStateFair:
		return "Fair"
	case ThermalStateSerious:
		return "Serious"
	case ThermalStateCritical:
		return "Critical"
	}
	return fmt.Sprintf("ThermalState(%d)", state)
}

func powerModeDebugName(mode PowerMode) string {
	switch mode {
	case PowerModeUnknown:
		return "Unknown"
	case PowerModeLowPower:
		return "LowPower"
	case PowerModeBalanced:
		return "Balanced"
	case PowerModeHighPower:
		return "HighPower"
	}
	return fmt.Sprintf("PowerMode(%d)", mode)
}

func windowAppearanceDebugName(appearance WindowAppearance) string {
	switch appearance {
	case WindowAppearanceLight:
		return "Light"
	case WindowAppearanceVibrantLight:
		return "VibrantLight"
	case WindowAppearanceDark:
		return "Dark"
	case WindowAppearanceVibrantDark:
		return "VibrantDark"
	}
	return fmt.Sprintf("WindowAppearance(%d)", appearance)
}

func marshalDebugState(value any) jsontext.Value {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("gpui: debug state: %v", err))
	}
	return jsontext.Value(data)
}

// DebugState returns a diagnostic snapshot of app, display and window state. Its layout is not
// stable; do not parse it for program logic.
func (cx *App) DebugState() jsontext.Value {
	displays := []debugStateDisplay{}
	for _, display := range cx.platform.Displays() {
		uuid := None[string]()
		if id, err := display.UUID(); err == nil {
			uuid = Some(id.String())
		}
		refreshIntervalSeconds := None[float64]()
		if interval, ok := display.RefreshInterval(); ok {
			refreshIntervalSeconds = Some(interval.Seconds())
		}
		displays = append(displays, debugStateDisplay{
			ID:                     uint64(display.ID()),
			UUID:                   uuid,
			Bounds:                 display.Bounds(),
			VisibleBounds:          display.VisibleBounds(),
			RefreshIntervalSeconds: refreshIntervalSeconds,
		})
	}
	windows := []jsontext.Value{}
	for windowID, window := range cx.windows.all() {
		if window != nil {
			windows = append(windows, window.debugState())
		} else {
			windows = append(windows, marshalDebugState(debugStateUpdatingWindow{ID: windowID.AsU64(), Updating: true}))
		}
	}
	activeWindowID := None[uint64]()
	if handle, ok := cx.platform.ActiveWindow(); ok {
		activeWindowID = Some(handle.WindowID().AsU64())
	}
	windowStack := None[[]uint64]()
	if stack, ok := cx.platform.WindowStack(); ok {
		ids := []uint64{}
		for _, handle := range stack {
			ids = append(ids, handle.WindowID().AsU64())
		}
		windowStack = Some(ids)
	}
	primaryDisplayID := None[uint64]()
	if display := cx.platform.PrimaryDisplay(); display != nil {
		primaryDisplayID = Some(uint64(display.ID()))
	}
	return marshalDebugState(debugStateApp{
		Compositor:   cx.platform.CompositorName(),
		ThermalState: thermalStateDebugName(cx.platform.ThermalState()),
		PowerMode: debugStatePowerMode{
			Cached:   powerModeDebugName(cx.powerMode),
			Platform: powerModeDebugName(cx.platform.PowerMode()),
		},
		WindowAppearance: windowAppearanceDebugName(cx.platform.WindowAppearance()),
		CursorVisible:    cx.platform.IsCursorVisible(),
		ActiveWindowID:   activeWindowID,
		WindowStack:      windowStack,
		PrimaryDisplayID: primaryDisplayID,
		Displays:         displays,
		Windows:          windows,
		Platform:         cx.platform.DebugState(),
	})
}

// ThermalState returns the current thermal state of the system.
func (cx *App) ThermalState() ThermalState {
	return cx.platform.ThermalState()
}

// PreventIdleSleep prevents idle sleep while the returned guard is held.
func (cx *App) PreventIdleSleep(reason string) *Task[*ActivityGuard] {
	return cx.platform.PreventIdleSleep(reason)
}

// OnThermalStateChange invokes a handler when the thermal state changes.
func (cx *App) OnThermalStateChange(callback func(cx *App)) Subscription {
	return cx.thermalStateObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

// OnSystemSleep invokes a handler when the system is about to sleep. Handlers should record
// state or cancel work rather than start it.
func (cx *App) OnSystemSleep(callback func(cx *App)) Subscription {
	return cx.systemSleepObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

func (cx *App) displaysChanged() {
	current := readDisplays(cx.platform)
	events := displayEvents(cx.displays, current)
	cx.displays = current
	for _, event := range events {
		changed, ok := event.(DisplayEventChanged)
		if !ok {
			continue
		}
		id := changed.ID
		for _, handle := range cx.Windows() {
			err := cx.updateWindow(handle, func(_ AnyView, window *Window, cx *App) {
				if displayID, ok := window.displayID.Get(); ok && displayID == id {
					window.notifyDisplayObservers(cx)
				}
			})
			if err != nil {
				gpuiutil.LogErr(err)
			}
		}
	}
	for _, event := range events {
		cx.displayObservers.retain(func(callback *displayHandler) bool { return (*callback)(event, cx) })
	}
}

func (cx *App) knowsDisplay(id DisplayID) bool {
	_, ok := cx.displays[id]
	return ok
}

func (cx *App) displayRefreshInterval(id DisplayID) (time.Duration, bool) {
	state, ok := cx.displays[id]
	if !ok {
		return 0, false
	}
	return state.refreshInterval.Get()
}

// ObserveDisplays invokes a handler when a display is connected, disconnected, or its
// properties change.
func (cx *App) ObserveDisplays(callback func(event DisplayEvent, cx *App)) Subscription {
	return cx.displayObservers.insert(func(event DisplayEvent, cx *App) bool {
		callback(event, cx)
		return true
	})
}

// OnSystemWake invokes a handler when the system wakes from sleep.
func (cx *App) OnSystemWake(callback func(cx *App)) Subscription {
	return cx.systemWakeObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

// OnAppLifecycle observes mobile application lifecycle changes. Desktop platforms do not emit
// these events. Unsubscribing removes the observer.
func (cx *App) OnAppLifecycle(callback func(phase AppLifecyclePhase, cx *App)) Subscription {
	return cx.appLifecycleObservers.insert(func(phase AppLifecyclePhase, cx *App) bool {
		callback(phase, cx)
		return true
	})
}

// OnMemoryWarning observes mobile memory-pressure warnings so applications can release
// expendable caches.
func (cx *App) OnMemoryWarning(callback func(cx *App)) Subscription {
	return cx.memoryWarningObservers.insert(func(cx *App) bool {
		callback(cx)
		return true
	})
}

// WindowAppearance returns the appearance of the application's windows.
func (cx *App) WindowAppearance() WindowAppearance {
	return cx.platform.WindowAppearance()
}

// SetWindowAppearance overrides the light or dark appearance applied to the app's windows; None
// follows the system again. It is a no-op on platforms other than macOS.
func (cx *App) SetWindowAppearance(appearance Option[WindowAppearance]) {
	cx.platform.SetWindowAppearance(appearance)
}

// ButtonLayout returns the window button layout configuration when supported.
func (cx *App) ButtonLayout() (WindowButtonLayout, bool) {
	return cx.platform.ButtonLayout()
}

// TextScaleFactor returns the factor by which text should be scaled, on top of the window's
// scale factor.
func (cx *App) TextScaleFactor() float32 {
	return cx.platform.TextScaleFactor()
}

// ReadFromClipboard reads data from the platform clipboard.
func (cx *App) ReadFromClipboard() (ClipboardItem, bool) {
	return cx.platform.ReadFromClipboard()
}

// ReadFromClipboardAsync reads data from the platform clipboard, resolving once the contents
// are available.
func (cx *App) ReadFromClipboardAsync() *Task[Option[ClipboardItem]] {
	return cx.platform.ReadFromClipboardAsync()
}

// SetTextRenderingMode sets the text rendering mode for the application.
func (cx *App) SetTextRenderingMode(mode TextRenderingMode) {
	*cx.textRenderingMode = mode
}

// TextRenderingMode returns the current text rendering mode for the application.
func (cx *App) TextRenderingMode() TextRenderingMode {
	return *cx.textRenderingMode
}

// PowerMode returns the operating system's power mode, or PowerModeUnknown when the platform
// cannot determine it.
func (cx *App) PowerMode() PowerMode {
	return cx.powerMode
}

// ToAsync creates an AsyncApp, which holds the app weakly and can be held across await points.
func (cx *App) ToAsync() *AsyncApp {
	return &AsyncApp{
		app:                cx.this,
		bgExecutor:         cx.bgExecutor,
		foregroundExecutor: cx.foregroundExecutor,
	}
}

// ForegroundExecutor returns the main-thread executor. It panics once on_app_quit handlers run.
func (cx *App) ForegroundExecutor() ForegroundExecutor {
	if cx.quitting {
		panic("Can't spawn on main thread after on_app_quit")
	}
	return cx.foregroundExecutor
}

func (cx *App) actionRegistry() *actionRegistry {
	return cx.actions
}

// WriteToClipboard writes data to the platform clipboard.
func (cx *App) WriteToClipboard(item ClipboardItem) {
	cx.platform.WriteToClipboard(item)
}

// ReadFromPrimary reads data from the primary selection buffer. Only available on Linux.
func (cx *App) ReadFromPrimary() (ClipboardItem, bool) {
	return cx.platform.ReadFromPrimary()
}

// WriteToPrimary writes data to the primary selection buffer. Only available on Linux.
func (cx *App) WriteToPrimary(item ClipboardItem) {
	cx.platform.WriteToPrimary(item)
}

// ReadFromFindPasteboard reads data from macOS's "Find" pasteboard, used to share the current
// search string between apps.
func (cx *App) ReadFromFindPasteboard() (ClipboardItem, bool) {
	return cx.platform.ReadFromFindPasteboard()
}

// WriteToFindPasteboard writes data to macOS's "Find" pasteboard, used to share the current
// search string between apps.
func (cx *App) WriteToFindPasteboard(item ClipboardItem) {
	cx.platform.WriteToFindPasteboard(item)
}

// WriteCredentials writes credentials to the platform keychain.
func (cx *App) WriteCredentials(url string, username string, password []byte) *Task[struct{}] {
	return cx.platform.WriteCredentials(url, username, password)
}

// WriteCredentialsIfNotExists atomically writes credentials unless an item with this URL and
// username exists. The task yields true if written, or false if the existing item was left
// unchanged; it fails if Keychain access fails or the platform lacks support.
func (cx *App) WriteCredentialsIfNotExists(url string, username string, password []byte) *Task[bool] {
	return cx.platform.WriteCredentialsIfNotExists(url, username, password)
}

// ReadCredentials reads credentials from the platform keychain.
func (cx *App) ReadCredentials(url string) *Task[Option[Credentials]] {
	return cx.platform.ReadCredentials(url)
}

// DeleteCredentials deletes credentials from the platform keychain.
func (cx *App) DeleteCredentials(url string) *Task[struct{}] {
	return cx.platform.DeleteCredentials(url)
}

// OpenURL directs the platform's default browser to open the given URL.
func (cx *App) OpenURL(url string) {
	cx.platform.OpenURL(url)
}

// RegisterURLScheme registers the given URL scheme (e.g. `zed` for `zed://` urls) to be opened
// by the current app.
func (cx *App) RegisterURLScheme(scheme string) *Task[struct{}] {
	return cx.platform.RegisterURLScheme(scheme)
}

// SetAppIdentity sets the application's process-wide identity and user-visible name. Call it
// once, early in startup, before opening windows or posting notifications.
func (cx *App) SetAppIdentity(identifier string, name string) {
	cx.platform.SetAppIdentity(identifier, name)
}

// ShowSystemNotification posts a notification to the operating system's notification center.
// A notification whose tag matches an earlier one replaces it where the platform supports it.
func (cx *App) ShowSystemNotification(notification SystemNotification) {
	cx.platform.ShowSystemNotification(notification)
}

// DismissSystemNotification removes the delivered or pending notification with this tag,
// best-effort. Dropping the task does not cancel dismissal.
func (cx *App) DismissSystemNotification(tag string) *Task[struct{}] {
	return cx.platform.DismissSystemNotification(tag)
}

// OnSystemNotificationResponse registers the handler invoked when the user activates a system
// notification. Subsequent registrations replace the handler.
func (cx *App) OnSystemNotificationResponse(callback func(response SystemNotificationResponse, cx *App)) {
	this := cx.this
	cx.platform.OnSystemNotificationResponse(func(response SystemNotificationResponse) {
		if app := this.Value(); app != nil {
			borrow := app.BorrowMut()
			defer borrow.Release()
			callback(response, borrow.App)
		}
	})
}

// AppPath returns the full pathname of the current app bundle, or an error if the app is not
// being run from a bundle.
func (cx *App) AppPath() (string, error) {
	return cx.platform.AppPath()
}

// CompositorName returns the name of the compositor in use on Linux, and an empty string on
// other platforms.
func (cx *App) CompositorName() string {
	return cx.platform.CompositorName()
}

// PlatformInfo returns the host OS and application target reported by the backend.
func (cx *App) PlatformInfo() PlatformInfo {
	return cx.platform.PlatformInfo()
}

// PathForAuxiliaryExecutable returns the path of the executable with the specified name in the
// application bundle.
func (cx *App) PathForAuxiliaryExecutable(name string) (string, error) {
	return cx.platform.PathForAuxiliaryExecutable(name)
}

// PromptForPaths displays a platform modal for selecting paths. The selection, or None when
// cancelled, is relayed through the returned receiver.
func (cx *App) PromptForPaths(options PathPromptOptions) *scheduler.OneshotReceiver[scheduler.Result[Option[[]string]]] {
	return cx.platform.PromptForPaths(options)
}

// PromptForNewPath displays a platform modal for selecting a new path where a file can be
// saved, starting in directory.
func (cx *App) PromptForNewPath(directory string, suggestedName Option[string]) *scheduler.OneshotReceiver[scheduler.Result[Option[string]]] {
	return cx.platform.PromptForNewPath(directory, suggestedName)
}

// RevealPath reveals the specified path at the platform level, such as in Finder on macOS.
func (cx *App) RevealPath(path string) {
	cx.platform.RevealPath(path)
}

// OpenWithSystem opens the specified path with the system's default application.
func (cx *App) OpenWithSystem(path string) {
	cx.platform.OpenWithSystem(path)
}

// ShouldAutoHideScrollbars reports whether the user has configured scrollbars to auto-hide at
// the platform level.
func (cx *App) ShouldAutoHideScrollbars() bool {
	return cx.platform.ShouldAutoHideScrollbars()
}

// Restart restarts the application.
func (cx *App) Restart() {
	cx.restartObservers.retain(func(observer *handler) bool { return (*observer)(cx) })
	arguments := cx.restartArguments
	cx.restartArguments = nil
	cx.platform.Restart(cx.restartPath.Take(), arguments)
}

// SetRestartPath sets the path to use when restarting the application.
func (cx *App) SetRestartPath(path string) {
	cx.restartPath = Some(path)
}

// HTTPClient returns the HTTP client for the application.
func (cx *App) HTTPClient() HTTPClient {
	return cx.httpClient
}

// SetHTTPClient sets the HTTP client for the application.
func (cx *App) SetHTTPClient(newClient HTTPClient) {
	cx.httpClient = newClient
}

// SetQuitMode configures when the application should automatically quit. By default,
// QuitModeDefault is used.
func (cx *App) SetQuitMode(mode QuitMode) {
	cx.quitMode = mode
}

// SetActivationPolicy sets whether the application participates in the system's foreground UI.
// It only has an effect on macOS.
func (cx *App) SetActivationPolicy(policy ActivationPolicy) {
	cx.platform.SetActivationPolicy(policy)
}

// SvgRenderer returns the SVG renderer used by the application.
func (cx *App) SvgRenderer() SvgRenderer {
	return cx.svgRenderer
}

func (cx *App) pushEffect(e effect) {
	switch e := e.(type) {
	case effectNotify:
		if _, ok := cx.pendingNotifications[e.emitter]; ok {
			return
		}
		cx.pendingNotifications[e.emitter] = struct{}{}
	case effectNotifyGlobalObservers:
		if _, ok := cx.pendingGlobalNotifications[e.globalType]; ok {
			return
		}
		cx.pendingGlobalNotifications[e.globalType] = struct{}{}
	}
	cx.pendingEffects = append(cx.pendingEffects, e)
}

func (cx *App) popEffect() (effect, bool) {
	if len(cx.pendingEffects) == 0 {
		return nil, false
	}
	e := cx.pendingEffects[0]
	cx.pendingEffects[0] = nil
	cx.pendingEffects = cx.pendingEffects[1:]
	return e, true
}

func (cx *App) flushEffects() {
	for {
		cx.releaseDroppedEntities()
		cx.releaseDroppedFocusHandles()
		if e, ok := cx.popEffect(); ok {
			switch e := e.(type) {
			case effectNotify:
				cx.applyNotifyEffect(e.emitter)
			case effectEmit:
				cx.applyEmitEffect(e.emitter, e.eventType, e.event)
			case effectRefreshWindows:
				cx.applyRefreshEffect()
			case effectNotifyGlobalObservers:
				cx.applyNotifyGlobalObserversEffect(e.globalType)
			case effectDefer:
				cx.applyDeferEffect(e.callback)
			case effectEntityCreated:
				cx.applyEntityCreatedEffect(e.entity, e.tid, e.window)
			}
		} else {
			if cx.mode.test {
				var dirty []AnyWindowHandle
				for _, window := range cx.windows.all() {
					if window != nil && window.invalidator.isDirty() {
						dirty = append(dirty, window.handle)
					}
				}
				for _, window := range dirty {
					_, err := cx.updateWindowID(window.id, func(_ AnyView, window *Window, cx *App) struct{} {
						window.Draw(cx).Clear(cx)
						return struct{}{}
					})
					if err != nil {
						panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
					}
				}
			}

			if len(cx.pendingEffects) == 0 {
				for _, window := range cx.windows.all() {
					if window == nil {
						continue
					}
					if window.invalidator.isDirty() || *window.needsPresent || len(*window.nextFrameCallbacks) != 0 {
						window.platformWindow.ScheduleFrame()
					}
				}
				break
			}
		}
	}
}

type entityDropper interface {
	Drop(cx *App)
}

func (cx *App) releaseDroppedEntities() {
	for {
		dropped := cx.entities.takeDropped()
		if len(dropped) == 0 {
			break
		}

		for _, d := range dropped {
			cx.observers.remove(d.entityID)
			cx.eventListeners.remove(d.entityID)
			delete(cx.windowInvalidatorsByEntity, d.entityID)
			delete(cx.currentWindowByEntity, d.entityID)
			for releaseCallback := range cx.releaseListeners.remove(d.entityID) {
				releaseCallback(d.entity, cx)
			}
			if dropper, ok := d.entity.(entityDropper); ok {
				dropper.Drop(cx)
			}
		}
	}
}

func (cx *App) releaseDroppedFocusHandles() {
	focusHandles := cx.focusHandles
	focusHandles.mu.Lock()
	if len(focusHandles.droppedHandles) == 0 {
		focusHandles.mu.Unlock()
		return
	}

	var windows []AnyWindowHandle
	for _, window := range cx.windows.all() {
		if window == nil {
			continue
		}
		focus, ok := window.focus.Get()
		if ok && slices.Contains(focusHandles.droppedHandles, focus) {
			windows = append(windows, window.handle)
		}
	}
	for _, handleID := range focusHandles.droppedHandles {
		focusHandles.removeHandle(handleID)
	}
	focusHandles.droppedHandles = focusHandles.droppedHandles[:0]
	focusHandles.mu.Unlock()

	for _, windowHandle := range windows {
		_ = windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) { window.Blur(cx) })
	}
}

func (cx *App) applyNotifyEffect(emitter EntityID) {
	delete(cx.pendingNotifications, emitter)
	cx.observers.retain(emitter, func(h *handler) bool { return (*h)(cx) })
}

func (cx *App) applyEmitEffect(emitter EntityID, eventType reflect.Type, event any) {
	cx.eventListeners.retain(emitter, func(l *eventListener) bool {
		if l.eventType == eventType {
			return l.listener(event, cx)
		}
		return true
	})
}

func (cx *App) applyRefreshEffect() {
	for _, window := range cx.windows.all() {
		if window != nil {
			window.refreshing = true
			window.invalidator.setDirty(true)
		}
	}
}

func (cx *App) applyNotifyGlobalObserversEffect(typeID reflect.Type) {
	delete(cx.pendingGlobalNotifications, typeID)
	cx.globalObservers.retain(typeID, func(observer *handler) bool { return (*observer)(cx) })
}

func (cx *App) applyDeferEffect(callback func(cx *App)) {
	callback(cx)
}

func (cx *App) applyEntityCreatedEffect(entity AnyEntity, tid reflect.Type, window Option[WindowID]) {
	defer entity.Release()
	if id, ok := window.Get(); ok {
		cx.currentWindowByEntity[entity.EntityID()] = id
	}

	cx.newEntityObservers.retain(tid, func(observer *newEntityListener) bool {
		if id, ok := window.Get(); ok {
			entity := entity.Clone()
			_, err := cx.updateWindowID(id, func(_ AnyView, window *Window, cx *App) struct{} {
				(*observer)(entity, window, cx)
				return struct{}{}
			})
			if err != nil {
				entity.Release()
				panic(fmt.Sprintf("All windows should be off the stack when flushing effects: %v", err))
			}
		} else {
			(*observer)(entity.Clone(), nil, cx)
		}
		return true
	})
}

// WithWindow runs f against the entity's current window: the most recently rendered window
// that referenced the entity, or its creation window if it has yet to be rendered. It reports
// false if the entity has no current window, that window has been closed, or it is already on
// the update stack.
func (cx *App) WithWindow[R any](entityID EntityID, f func(window *Window, cx *App) R) (R, bool) {
	windowID, ok := cx.currentWindowByEntity[entityID]
	if !ok {
		var zero R
		return zero, false
	}
	result, err := cx.updateWindowID(windowID, func(_ AnyView, window *Window, cx *App) R { return f(window, cx) })
	return result, err == nil
}

func (cx *App) ensureWindow(entityID EntityID, window WindowID) {
	if _, ok := cx.currentWindowByEntity[entityID]; !ok {
		cx.currentWindowByEntity[entityID] = window
	}
}

func (cx *App) updateWindowID[T any](id WindowID, update func(root AnyView, window *Window, cx *App) T) (T, error) {
	var result T
	found := false
	cx.updateWindowErased(id, func(root AnyView, window *Window, cx *App, ok bool) {
		if ok {
			result = update(root, window, cx)
			found = true
		} else {
			var zero T
			result = zero
			found = false
		}
	})
	if !found {
		return result, errors.New("window not found")
	}
	return result, nil
}

// BackgroundExecutor returns the executor used to spawn futures on background threads.
func (cx *App) BackgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

// IsTest reports whether this app runs on the deterministic test scheduler.
func (cx *App) IsTest() bool {
	return cx.bgExecutor.IsTest()
}

// ForegroundJournal returns the foreground work journal for this app's foreground thread.
// Apps constructed on the same thread share the stream.
func (cx *App) ForegroundJournal() ForegroundJournal {
	return cx.foregroundJournal
}

// StartHangMonitor starts detecting foreground hangs on a dedicated goroutine, passing each
// poll's incidents to onPoll. It fails if the monitor was already started or can't be spawned.
func (cx *App) StartHangMonitor(config HangMonitorConfig, onPoll func(poll HangMonitorPoll)) error {
	if cx.hangMonitor != nil {
		if gpuiutil.DebugAssertions {
			panic("the hang monitor was started twice")
		}
		return HangMonitorErrorAlreadyStarted{}
	}
	detector := NewHangDetector(cx.ForegroundJournal(), config.Threshold, config.FrameBudget)
	monitor, err := spawnHangMonitor(detector, config.Interval, onPoll)
	if err != nil {
		return &HangMonitorErrorSpawn{Err: err}
	}
	cx.hangMonitor = monitor
	return nil
}

// Spawn spawns the future returned by f on the main thread. f is invoked with an AsyncApp,
// which allows the application state to be accessed across await points.
func (cx *App) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R] {
	acx := cx.prepareSpawn()
	return cx.foregroundExecutor.Spawn(func(s *scheduler.Scope) (R, error) {
		acx.Scope = s
		return f(acx)
	})
}

// SpawnWithPriority spawns the future returned by f on the main thread with the given
// priority.
func (cx *App) SpawnWithPriority[R any](priority Priority, f func(cx *AsyncApp) (R, error)) *Task[R] {
	if cx.quitting {
		gpuiutil.DebugPanic("Can't spawn on main thread after on_app_quit")
	}
	acx := cx.ToAsync()
	return cx.foregroundExecutor.SpawnWithPriority(priority, func(s *scheduler.Scope) (R, error) {
		acx.Scope = s
		return f(acx)
	})
}

// Defer schedules f to be run at the end of the current effect cycle, allowing entities that
// are currently on the stack to be returned to the app.
func (cx *App) Defer(f func(cx *App)) {
	cx.pushEffect(effectDefer{callback: f})
}

// AssetSource returns the application's asset source, provided when constructing the App.
func (cx *App) AssetSource() AssetSource {
	return cx.assetSource
}

// TextSystem returns the text system.
func (cx *App) TextSystem() *TextSystem {
	return cx.textSystem
}

// OnMissingGlyphs invokes callback with grapheme clusters that exhausted font fallback.
// Registering a callback replaces the previous one and enables missing-glyph detection; the
// callback runs on the foreground executor. Unsubscribing disables detection until another
// callback is registered.
func (cx *App) OnMissingGlyphs(callback func(missingGlyphs []MissingGlyph, cx *App)) Subscription {
	registration := cx.missingGlyphCallback.replace(callback)

	if receiver := cx.textSystem.takeMissingGlyphReceiver(); receiver != nil {
		callback := cx.missingGlyphCallback
		cx.Spawn(func(acx *AsyncApp) (struct{}, error) {
			defer receiver.Close()
			for {
				missingGlyphs, err := receiver.Recv(acx)
				if err != nil {
					break
				}
				acx.Update(func(cx *App) { callback.invoke(missingGlyphs, cx) })
			}
			return struct{}{}, nil
		}).Detach()
	}
	cx.textSystem.enableMissingGlyphReporting()

	slot := cx.missingGlyphCallback
	textSystem := cx.textSystem
	return NewSubscription(func() {
		if slot.remove(registration) {
			textSystem.disableMissingGlyphReporting()
		}
	})
}

// HasGlobal reports whether a global of the given type has been assigned.
func (cx *App) HasGlobal[G any]() bool {
	_, ok := cx.globalsByType[reflect.TypeFor[G]()]
	return ok
}

// Global returns the global of the given type. It panics if a global for that type has not
// been assigned.
func (cx *App) Global[G any]() *G {
	global, ok := cx.globalsByType[reflect.TypeFor[G]()]
	if !ok {
		panic(fmt.Sprintf("no state of type %s exists", reflect.TypeFor[G]()))
	}
	return global.(*G)
}

// TryGlobal returns the global of the given type if a value has been assigned.
func (cx *App) TryGlobal[G any]() (*G, bool) {
	global, ok := cx.globalsByType[reflect.TypeFor[G]()]
	if !ok {
		return nil, false
	}
	return global.(*G), true
}

// GlobalMut returns the global of the given type for mutation and notifies its observers. It
// panics if a global for that type has not been assigned.
func (cx *App) GlobalMut[G any]() *G {
	globalType := reflect.TypeFor[G]()
	cx.pushEffect(effectNotifyGlobalObservers{globalType: globalType})
	global, ok := cx.globalsByType[globalType]
	if !ok {
		panic(fmt.Sprintf("no state of type %s exists", globalType))
	}
	return global.(*G)
}

// DefaultGlobal returns the global of the given type for mutation, assigning its zero value if
// a global of this type has not yet been assigned.
func (cx *App) DefaultGlobal[G any]() *G {
	globalType := reflect.TypeFor[G]()
	cx.pushEffect(effectNotifyGlobalObservers{globalType: globalType})
	global, ok := cx.globalsByType[globalType]
	if !ok {
		global = new(G)
		cx.globalsByType[globalType] = global
	}
	return global.(*G)
}

// SetGlobal sets the value of the global of the given type.
func (cx *App) SetGlobal[G any](global G) {
	globalType := reflect.TypeFor[G]()
	cx.pushEffect(effectNotifyGlobalObservers{globalType: globalType})
	cx.globalsByType[globalType] = &global
}

// ClearGlobals clears all stored globals. It does not notify global observers.
func (cx *App) ClearGlobals() {
	clear(cx.globalsByType)
}

// RemoveGlobal removes the global of the given type from the app context and returns it.
func (cx *App) RemoveGlobal[G any]() G {
	globalType := reflect.TypeFor[G]()
	cx.pushEffect(effectNotifyGlobalObservers{globalType: globalType})
	global, ok := cx.globalsByType[globalType]
	if !ok {
		panic(fmt.Sprintf("no global added for %s", globalType))
	}
	delete(cx.globalsByType, globalType)
	return *global.(*G)
}

// ObserveGlobal registers a callback to be invoked when a global of the given type is updated.
func (cx *App) ObserveGlobal[G any](f func(cx *App)) Subscription {
	subscription, activate := cx.globalObservers.insertPending(reflect.TypeFor[G](), func(cx *App) bool {
		f(cx)
		return true
	})
	cx.Defer(func(*App) { activate() })
	return subscription
}

func (cx *App) leaseGlobal[G any]() *globalLease[G] {
	globalType := reflect.TypeFor[G]()
	global, ok := cx.globalsByType[globalType]
	if !ok {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: no global registered of type %s", globalType))
	}
	delete(cx.globalsByType, globalType)
	return &globalLease[G]{global: global}
}

func (cx *App) endGlobalLease[G any](lease *globalLease[G]) {
	globalType := reflect.TypeFor[G]()
	cx.pushEffect(effectNotifyGlobalObservers{globalType: globalType})
	cx.globalsByType[globalType] = lease.global
}

func (cx *App) newEntityObserver(key reflect.Type, value newEntityListener) Subscription {
	return cx.newEntityObservers.insert(key, value)
}

// ObserveNew arranges for onNew to be invoked whenever an entity of type T is created. window
// is nil when the entity was created outside a window.
func (cx *App) ObserveNew[T any](onNew func(state *T, window *Window, cx *Context[T])) Subscription {
	return cx.newEntityObserver(reflect.TypeFor[T](), func(anyEntity AnyEntity, window *Window, cx *App) {
		defer anyEntity.Release()
		entity, ok := anyEntity.Downcast[T]()
		if !ok {
			panic("called `Result::unwrap()` on an `Err` value")
		}
		entity.Update(cx, func(state *T, cx *Context[T]) {
			onNew(state, window, cx)
		})
	})
}

// ObserveRelease observes the release of an entity. The callback is invoked after the entity
// has no more strong references but before it has been dropped.
func (cx *App) ObserveRelease[T any](handle Entity[T], onRelease func(state *T, cx *App)) Subscription {
	return cx.releaseListeners.insert(handle.EntityID(), func(entity any, cx *App) {
		state, ok := entity.(*T)
		if !ok {
			panic("invalid entity type")
		}
		onRelease(state, cx)
	})
}

// ObserveReleaseIn observes the release of an entity, invoking onRelease with the given
// window. The callback is invoked after the entity has no more strong references but before it
// has been dropped.
func (cx *App) ObserveReleaseIn[T any](handle Entity[T], window *Window, onRelease func(state *T, window *Window, cx *App)) Subscription {
	windowHandle := window.handle
	return cx.ObserveRelease(handle, func(entity *T, cx *App) {
		_ = windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) { onRelease(entity, window, cx) })
	})
}

// ObserveKeystrokes registers a callback to be invoked after a keystroke is resolved in any
// window, including the action that handled it, if any.
func (cx *App) ObserveKeystrokes(f func(event *KeystrokeEvent, window *Window, cx *App)) Subscription {
	return cx.keystrokeObservers.insert(func(event *KeystrokeEvent, window *Window, cx *App) bool {
		f(event, window, cx)
		return true
	})
}

// InterceptKeystrokes registers a callback to be invoked when a keystroke is received by the
// application in any window, before all other action and event mechanisms have resolved.
func (cx *App) InterceptKeystrokes(f func(event *KeystrokeEvent, window *Window, cx *App)) Subscription {
	return cx.keystrokeInterceptors.insert(func(event *KeystrokeEvent, window *Window, cx *App) bool {
		f(event, window, cx)
		return true
	})
}

// BindKeys registers key bindings.
func (cx *App) BindKeys(bindings []KeyBinding) {
	cx.keymap.AddBindings(bindings)
	cx.pendingEffects = append(cx.pendingEffects, effectRefreshWindows{})
}

// ClearKeyBindings clears all key bindings in the app.
func (cx *App) ClearKeyBindings() {
	cx.keymap.Clear()
	cx.pendingEffects = append(cx.pendingEffects, effectRefreshWindows{})
}

// KeyBindings returns the app's keymap.
func (cx *App) KeyBindings() *Keymap {
	return cx.keymap
}

// OnAction registers a global handler for actions of type A invoked via the keyboard. These
// handlers run at the end of the bubble phase for actions.
func (cx *App) OnAction[A Action](listener func(action A, cx *App)) *App {
	actionType := reflect.TypeFor[A]()
	cx.globalActionListeners[actionType] = append(cx.globalActionListeners[actionType], func(action any, phase DispatchPhase, cx *App) {
		if phase == DispatchPhaseBubble {
			listener(action.(A), cx)
		}
	})
	return cx
}

// StopPropagation stops dispatching the current event to handlers with a lower z-index (mouse)
// or higher in the tree (keyboard).
func (cx *App) StopPropagation() {
	cx.propagateEvent = false
}

// Propagate lets the current action continue to handlers higher in the element tree.
func (cx *App) Propagate() {
	cx.propagateEvent = true
}

// BuildAction builds an action from some arbitrary data, typically a keymap entry.
func (cx *App) BuildAction(name string, data Option[jsontext.Value]) (Action, error) {
	return cx.actions.buildAction(name, data)
}

// AllActionNames returns the names of all registered actions.
func (cx *App) AllActionNames() []string {
	return cx.actions.allActionNames()
}

// AllBindingsForInput returns key bindings that match the given input, without checking
// context, in the order they were added.
func (cx *App) AllBindingsForInput(input []Keystroke) []KeyBinding {
	return cx.keymap.AllBindingsForInput(input)
}

// ActionSchema is a registered action's name and its JSON schema; Schema is nil for actions
// without one.
type ActionSchema struct {
	Name   string
	Schema *jsonschema.Schema
}

// ActionSchemas returns all non-internal registered actions with their schemas.
func (cx *App) ActionSchemas() []ActionSchema {
	schemas := cx.actions.actionSchemas()
	out := make([]ActionSchema, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, ActionSchema{Name: s.name, Schema: s.schema})
	}
	return out
}

// ActionSchemaByName returns the schema of the named action. It reports false if the action
// is not found; the schema is nil if the action has none.
func (cx *App) ActionSchemaByName(name string) (*jsonschema.Schema, bool) {
	return cx.actions.actionSchemaByName(name)
}

// DeprecatedActionsToPreferredActions returns a map from a deprecated action name to the
// canonical name.
func (cx *App) DeprecatedActionsToPreferredActions() map[string]string {
	return cx.actions.deprecatedAliasesMap()
}

// ActionDeprecationMessages returns a map from an action name to its deprecation message.
func (cx *App) ActionDeprecationMessages() map[string]string {
	return cx.actions.deprecationMessagesMap()
}

// ActionDocumentation returns a map from an action name to its documentation.
func (cx *App) ActionDocumentation() map[string]string {
	return cx.actions.documentationMap()
}

// OnAppQuit registers a callback to be invoked when the application is about to quit. It is
// not possible to cancel the quit event at this point.
func (cx *App) OnAppQuit(onQuit func(cx *App) scheduler.Future[struct{}]) Subscription {
	return cx.quitObservers.insert(func(cx *App) scheduler.Future[struct{}] {
		return onQuit(cx)
	})
}

// OnAppRestart registers a callback to be invoked when the application is about to restart,
// before any OnAppQuit callbacks.
func (cx *App) OnAppRestart(onRestart func(cx *App)) Subscription {
	return cx.restartObservers.insert(func(cx *App) bool {
		onRestart(cx)
		return true
	})
}

// OnWindowClosed registers a callback to be invoked when a window is closed. The window is no
// longer accessible at the point this callback is invoked.
func (cx *App) OnWindowClosed(onClosed func(cx *App, id WindowID)) Subscription {
	return cx.windowClosedObservers.insert(onClosed)
}

func (cx *App) clearPendingKeystrokes() {
	for _, window := range cx.Windows() {
		_ = window.Update(cx, func(_ AnyView, window *Window, cx *App) {
			window.clearPendingKeystrokes(cx)
		})
	}
}

// IsActionAvailable reports whether the given action is bound in the current context, as
// defined by the app's current focus, the bindings in the element tree, and any global action
// listeners.
func (cx *App) IsActionAvailable(action Action) bool {
	actionAvailable := false
	if window, ok := cx.ActiveWindow(); ok {
		windowActionAvailable, err := AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, cx *App) bool {
			return window.IsActionAvailable(action, cx)
		})
		if err == nil {
			actionAvailable = windowActionAvailable
		}
	}

	if actionAvailable {
		return true
	}
	_, ok := cx.globalActionListeners[reflect.TypeOf(action)]
	return ok
}

// SetMenus sets the menu bar for this application, replacing any existing menu bar.
func (cx *App) SetMenus(menus []Menu) {
	cx.platform.SetMenus(menus, cx.keymap)
}

// GetMenus returns the menu bar for this application.
func (cx *App) GetMenus() ([]OwnedMenu, bool) {
	return cx.platform.GetMenus()
}

// SetDockMenu sets the right click menu for the app icon in the dock.
func (cx *App) SetDockMenu(menus []MenuItem) {
	cx.platform.SetDockMenu(menus, cx.keymap)
}

// PerformDockMenuAction performs the action associated with the given dock menu item, only
// used on Windows for now.
func (cx *App) PerformDockMenuAction(action int) {
	cx.platform.PerformDockMenuAction(action)
}

// AddRecentDocument adds the given path to the bottom of the list of recent paths for the
// application, moving it there if it is already in the list.
func (cx *App) AddRecentDocument(path string) {
	cx.platform.AddRecentDocument(path)
}

// UpdateJumpList updates the jump list with the updated list of recent paths for the
// application, only used on Windows for now. It also sets the dock menu on Windows.
func (cx *App) UpdateJumpList(menus []MenuItem, entries [][]string) *Task[[][]string] {
	return cx.platform.UpdateJumpList(menus, entries)
}

// DispatchAction dispatches an action to the currently active window or global action
// handler.
func (cx *App) DispatchAction(action Action) {
	if activeWindow, ok := cx.ActiveWindow(); ok {
		gpuiutil.LogErr(activeWindow.Update(cx, func(_ AnyView, window *Window, cx *App) {
			window.DispatchAction(action.BoxedClone(), cx)
		}))
	} else {
		cx.dispatchGlobalAction(action)
	}
}

func (cx *App) dispatchGlobalAction(action Action) {
	cx.propagateEvent = true
	actionType := reflect.TypeOf(action)

	if globalListeners, ok := cx.globalActionListeners[actionType]; ok {
		delete(cx.globalActionListeners, actionType)
		for _, listener := range globalListeners {
			listener(action, DispatchPhaseCapture, cx)
			if !cx.propagateEvent {
				break
			}
		}

		globalListeners = append(globalListeners, cx.globalActionListeners[actionType]...)
		delete(cx.globalActionListeners, actionType)
		cx.globalActionListeners[actionType] = globalListeners
	}

	if cx.propagateEvent {
		if globalListeners, ok := cx.globalActionListeners[actionType]; ok {
			delete(cx.globalActionListeners, actionType)
			for i := len(globalListeners) - 1; i >= 0; i-- {
				globalListeners[i](action, DispatchPhaseBubble, cx)
				if !cx.propagateEvent {
					break
				}
			}

			globalListeners = append(globalListeners, cx.globalActionListeners[actionType]...)
			delete(cx.globalActionListeners, actionType)
			cx.globalActionListeners[actionType] = globalListeners
		}
	}
}

// HasActiveDrag reports whether something is currently being dragged.
func (cx *App) HasActiveDrag() bool {
	return cx.activeDrag.IsSome()
}

// ActiveDragCursorStyle returns the cursor style of the currently active drag operation.
func (cx *App) ActiveDragCursorStyle() (CursorStyle, bool) {
	if drag := cx.activeDrag.AsMut(); drag != nil {
		return drag.CursorStyle.Get()
	}
	var zero CursorStyle
	return zero, false
}

// StopActiveDrag stops the active drag and clears any related effects.
func (cx *App) StopActiveDrag(window *Window) bool {
	if cx.activeDrag.IsSome() {
		releaseActiveDrag(cx.activeDrag.Take())
		if drag, ok := cx.platformOwnedDrag.Get(); ok && drag.sourceWindow == window.WindowHandle().WindowID() {
			if _, restored := drag.state.(platformOwnedDragStateRestoredInSourceWindow); restored {
				cx.platformOwnedDrag = None[platformOwnedDrag]()
			}
		}
		window.Refresh()
		return true
	}
	return false
}

func (cx *App) handActiveDragToPlatform(sourceWindow WindowID) bool {
	drag, ok := cx.activeDrag.Take().Get()
	if !ok {
		return false
	}
	cx.platformOwnedDrag = Some(platformOwnedDrag{
		sourceWindow: sourceWindow,
		state:        platformOwnedDragStateSuspended{drag: drag},
	})
	return true
}

func (cx *App) restorePlatformDrag(sourceWindow WindowID) bool {
	platformDrag := cx.platformOwnedDrag.AsMut()
	if platformDrag == nil || platformDrag.sourceWindow != sourceWindow {
		return false
	}
	state := platformDrag.state
	platformDrag.state = platformOwnedDragStateRestoredInSourceWindow{}
	suspended, ok := state.(platformOwnedDragStateSuspended)
	if !ok {
		return false
	}
	cx.activeDrag = Some(suspended.drag)
	return true
}

func (cx *App) handRestoredDragToPlatform(sourceWindow WindowID) bool {
	platformDrag := cx.platformOwnedDrag.AsMut()
	if platformDrag == nil || platformDrag.sourceWindow != sourceWindow {
		return false
	}
	if _, restored := platformDrag.state.(platformOwnedDragStateRestoredInSourceWindow); !restored {
		return false
	}
	drag, ok := cx.activeDrag.Take().Get()
	if !ok {
		return false
	}
	platformDrag.state = platformOwnedDragStateSuspended{drag: drag}
	return true
}

func (cx *App) endPlatformDrag(sourceWindow WindowID) bool {
	drag, ok := cx.platformOwnedDrag.Get()
	if !ok || drag.sourceWindow != sourceWindow {
		return false
	}
	cx.platformOwnedDrag = None[platformOwnedDrag]()
	if suspended, ok := drag.state.(platformOwnedDragStateSuspended); ok {
		suspended.drag.View.Release()
	}
	releaseActiveDrag(cx.activeDrag.Take())
	return true
}

// SetActiveDragCursorStyle sets the cursor style for the currently active drag operation.
func (cx *App) SetActiveDragCursorStyle(cursorStyle CursorStyle, window *Window) bool {
	if drag := cx.activeDrag.AsMut(); drag != nil {
		drag.CursorStyle = Some(cursorStyle)
		window.Refresh()
		return true
	}
	return false
}

// SetPromptBuilder sets the prompt renderer for GPUI, replacing the default or platform
// specific prompts with this custom implementation.
func (cx *App) SetPromptBuilder(renderer func(level PromptLevel, message string, detail Option[string], actions []PromptButton, handle PromptHandle, window *Window, cx *App) RenderablePromptHandle) {
	cx.promptBuilder = Some[promptBuilder](promptBuilderCustom{renderer: renderer})
}

// ResetPromptBuilder resets the prompt builder to the default implementation.
func (cx *App) ResetPromptBuilder() {
	cx.promptBuilder = Some[promptBuilder](promptBuilderDefault{})
}

// FocusHandle obtains a new FocusHandle, which allows you to track and manipulate the keyboard
// focus for elements rendered within a window.
func (cx *App) FocusHandle() FocusHandle {
	return newFocusHandle(cx.focusHandles)
}

// Notify tells GPUI that an entity has changed and observers of it should be notified.
func (cx *App) Notify(entityID EntityID) {
	windowInvalidators := cx.windowInvalidatorsByEntity[entityID]
	if windowInvalidators == nil {
		windowInvalidators = map[WindowID]windowInvalidator{}
	}
	cx.windowInvalidatorsByEntity[entityID] = map[WindowID]windowInvalidator{}

	windowIDs := slices.Sorted(maps.Keys(windowInvalidators))
	var liveInvalidators []windowInvalidator
	for _, windowID := range windowIDs {
		set, ok := cx.trackedEntities[windowID]
		if !ok {
			continue
		}
		if _, ok := set[entityID]; ok {
			liveInvalidators = append(liveInvalidators, windowInvalidators[windowID])
		}
	}

	if len(liveInvalidators) == 0 {
		if _, ok := cx.pendingNotifications[entityID]; !ok {
			cx.pendingNotifications[entityID] = struct{}{}
			cx.pendingEffects = append(cx.pendingEffects, effectNotify{emitter: entityID})
		}
	} else {
		for _, invalidator := range liveInvalidators {
			invalidator.invalidateView(entityID, cx)
		}
	}

	cx.windowInvalidatorsByEntity[entityID] = windowInvalidators
}

// GetName returns the name for this App.
func (cx *App) GetName() (string, bool) {
	return cx.name.Get()
}

// CanSelectMixedFilesAndDirs reports whether the platform file picker supports selecting a mix
// of files and directories.
func (cx *App) CanSelectMixedFilesAndDirs() bool {
	return cx.platform.CanSelectMixedFilesAndDirs()
}

// DropImage removes an image from the sprite atlas on all windows. Pass the window being
// updated as currentWindow, since it is not in the App's window map during the update; nil
// otherwise.
func (cx *App) DropImage(image *RenderImage, currentWindow *Window) {
	for _, window := range cx.windows.all() {
		if window != nil {
			_ = window.DropImage(image)
		}
	}

	if currentWindow != nil {
		_ = currentWindow.DropImage(image)
	}
}

// SetInspectorRenderer sets the renderer for the inspector.
func (cx *App) SetInspectorRenderer(f InspectorRenderer) {
	cx.inspectorRenderer = Some(f)
}

// RegisterInspectorElement registers a renderer specific to an inspector state of type T.
func (cx *App) RegisterInspectorElement[T any, R IntoElement](factory func(window *Window, cx *App) func(id InspectorElementID, state *T, window *Window, cx *App) R) {
	cx.inspectorElementRegistry.register(factory)
}

// InitColors initializes gpui's default colors for the application, accessible through
// DefaultColors.
func (cx *App) InitColors() {
	colors := ColorsLight()
	cx.SetGlobal(GlobalColors{Colors: &colors})
}

func (cx *App) updateWindowErased(windowID WindowID, update func(root AnyView, window *Window, cx *App, ok bool)) {
	cx.update(func(cx *App) {
		window, ok := cx.windows.get(windowID)
		if !ok || window == nil {
			update(AnyView{}, nil, nil, false)
			return
		}
		cx.windows.set(windowID, nil)

		rootView := window.root.Unwrap()

		cx.windowUpdateStack = append(cx.windowUpdateStack, windowUpdateEntry{
			windowID:  window.handle.id,
			bounds:    window.lastWindowBounds,
			displayID: window.displayID,
		})
		update(rootView, window, cx, true)
		if !updateWindowTrail(windowID, window, cx) {
			update(AnyView{}, nil, nil, false)
		}
	})
}

func updateWindowTrail(windowID WindowID, window *Window, cx *App) bool {
	cx.windowUpdateStack = cx.windowUpdateStack[:len(cx.windowUpdateStack)-1]

	if window.removed {
		window.drop()
		cx.endPlatformDrag(windowID)
		delete(cx.windowHandles, windowID)
		cx.windows.remove(windowID)
		if tracked, ok := cx.trackedEntities[windowID]; ok {
			delete(cx.trackedEntities, windowID)
			for entityID := range tracked {
				if windows, ok := cx.windowInvalidatorsByEntity[entityID]; ok {
					delete(windows, windowID)
				}
				if current, ok := cx.currentWindowByEntity[entityID]; ok && current == windowID {
					delete(cx.currentWindowByEntity, entityID)
				}
			}
		}

		cx.windowClosedObservers.retain(func(callback *windowClosedHandler) bool {
			(*callback)(cx, windowID)
			return true
		})

		var quitOnEmpty bool
		switch cx.quitMode {
		case QuitModeExplicit:
			quitOnEmpty = false
		case QuitModeLastWindowClosed:
			quitOnEmpty = true
		default:
			quitOnEmpty = runtime.GOOS != "darwin"
		}

		if quitOnEmpty && cx.windows.isEmpty() {
			cx.Quit()
		}
	} else if !cx.windows.set(windowID, window) {
		return false
	}
	return true
}

func (cx *App) updateEntityErased(handle AnyEntity, entityType string, update func(state any, cx *App)) {
	cx.update(func(cx *App) {
		lease := cx.entities.leaseErased(handle, entityType)
		update(lease.entity, cx)
		cx.entities.endLeaseErased(handle.EntityID(), lease)
	})
}

func (cx *App) prepareSpawn() *AsyncApp {
	if cx.quitting {
		gpuiutil.DebugPanic("Can't spawn on main thread after on_app_quit")
	}
	return cx.ToAsync()
}

func (cx *App) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	return cx.updateR(func(cx *App) AnyEntity {
		slot := cx.entities.reserveErased(entityType)
		handle := slot.Clone()
		entity := build(slot, cx)

		window := None[WindowID]()
		if n := len(cx.windowUpdateStack); n > 0 {
			window = Some(cx.windowUpdateStack[n-1].windowID)
		}
		cx.pushEffect(effectEntityCreated{entity: handle, tid: entityType, window: window})

		return cx.entities.insertErased(slot, entity)
	})
}

func (cx *App) reserveEntity(entityType reflect.Type) AnyEntity {
	return cx.entities.reserveErased(entityType)
}

func (cx *App) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	return cx.updateR(func(cx *App) AnyEntity {
		entity := build(cx)
		return cx.entities.insertErased(slot, entity)
	})
}

func (cx *App) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	cx.updateEntityErased(handle, handle.EntityType().String(), update)
}

func (cx *App) asMut(handle AnyEntity) *App {
	return cx
}

func (cx *App) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	entity := cx.entities.readErased(handle)
	read(entity, cx)
}

func (cx *App) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	_, err := cx.updateWindowID(window.id, func(root AnyView, window *Window, cx *App) struct{} {
		update(root, window, cx)
		return struct{}{}
	})
	return err
}

func (cx *App) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	_, ok := cx.WithWindow(entityID, func(window *Window, cx *App) struct{} {
		f(window, cx)
		return struct{}{}
	})
	return ok
}

func (cx *App) readWindow(handle AnyWindowHandle, read func(root AnyView, cx *App)) error {
	window, ok := cx.windows.get(handle.id)
	if !ok {
		return errors.New("window not found")
	}
	if window == nil {
		panic("attempted to read a window that is already on the stack")
	}

	rootView := window.root.Unwrap()
	read(rootView, cx)
	return nil
}

func (cx *App) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *App) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	global, ok := cx.globalsByType[globalType]
	if !ok {
		panic(fmt.Sprintf("no state of type %s exists", globalType))
	}
	read(global, cx)
}

// New builds an entity that is owned by the application. buildEntity is invoked with a
// Context and returns the entity's state.
func (cx *App) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later with InsertEntity.
func (cx *App) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot obtained from ReserveEntity.
func (cx *App) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// AsMut leases an entity for mutable access; Release on the returned GpuiBorrow notifies the
// entity and ends the lease.
func (cx *App) AsMut[T any](handle Entity[T]) GpuiBorrow[T] {
	return AppContextAsMut(cx, handle)
}

// ReadWindow reads the root view of a window off of the application context.
func (cx *App) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on a background thread.
func (cx *App) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return AppContextBackgroundSpawn(cx, future)
}

// ReadGlobal reads the global of type G. It panics if no such global has been assigned.
func (cx *App) ReadGlobal[G, R any](callback func(global *G, cx *App) R) R {
	return AppContextReadGlobal(cx, callback)
}

type effect interface {
	isEffect()
	String() string
}

type effectNotify struct {
	emitter EntityID
}

type effectEmit struct {
	emitter   EntityID
	eventType reflect.Type
	event     any
}

type effectRefreshWindows struct{}

type effectNotifyGlobalObservers struct {
	globalType reflect.Type
}

type effectDefer struct {
	callback func(cx *App)
}

type effectEntityCreated struct {
	entity AnyEntity
	tid    reflect.Type
	window Option[WindowID]
}

func (effectNotify) isEffect()                {}
func (effectEmit) isEffect()                  {}
func (effectRefreshWindows) isEffect()        {}
func (effectNotifyGlobalObservers) isEffect() {}
func (effectDefer) isEffect()                 {}
func (effectEntityCreated) isEffect()         {}

func (e effectNotify) String() string {
	return fmt.Sprintf("Notify(%s)", e.emitter)
}

func (e effectEmit) String() string {
	return fmt.Sprintf("Emit(%s)", e.emitter.debugString())
}

func (effectRefreshWindows) String() string {
	return "RefreshWindows"
}

func (e effectNotifyGlobalObservers) String() string {
	return fmt.Sprintf("NotifyGlobalObservers(%v)", e.globalType)
}

func (effectDefer) String() string {
	return "Defer(..)"
}

func (e effectEntityCreated) String() string {
	return fmt.Sprintf("EntityCreated(AnyEntity { entity_id: %s, entity_type: %v })", e.entity.EntityID().debugString(), e.entity.EntityType())
}

type globalLease[G any] struct {
	global any
}

func (l *globalLease[G]) get() *G {
	return l.global.(*G)
}

// AnyDrag contains state associated with an active drag operation, started by dragging an
// element within the window or by dragging into the app from the underlying platform.
type AnyDrag struct {
	// View is the view used to render this drag.
	View AnyView
	// Value is the value of the dragged item, to be dropped.
	Value any
	// CursorOffset is used to render the dragged item in the same place on the original element
	// that the drag was initiated.
	CursorOffset Point[Pixels]
	// CursorStyle is the cursor style to use while dragging.
	CursorStyle Option[CursorStyle]
	// ExternalPayloadSource resolves the payload to offer the platform if the drag leaves the
	// window. It is invoked at most once per drag gesture, at promotion time.
	ExternalPayloadSource Option[ExternalDragPayloadSource]
}

// ExternalDragPayloadSource lazily resolves the payload handed to the platform when an internal
// drag is promoted to a native drag session.
type ExternalDragPayloadSource func(window *Window, cx *App) (ExternalDragPayload, bool)

// AnyTooltip contains state associated with a tooltip. You'll only need it if you're
// implementing tooltip behavior on a custom element.
type AnyTooltip struct {
	// View is the view used to display the tooltip.
	View AnyView
	// MousePosition is the absolute position of the mouse when the tooltip was deployed.
	MousePosition Point[Pixels]
	// CheckVisibleAndUpdate checks, given the bounds of the tooltip, whether it should still be
	// visible and updates its state accordingly.
	CheckVisibleAndUpdate func(bounds Bounds[Pixels], window *Window, cx *App) bool
}

// InputPreference is whether a keystroke should prefer character input or key bindings.
type InputPreference uint8

const (
	// InputPreferenceCharacterInput prefers typing text over triggering key bindings.
	InputPreferenceCharacterInput InputPreference = iota
	// InputPreferenceKeyBindings dispatches key bindings normally, if any match.
	InputPreferenceKeyBindings
)

// String returns the variant name.
func (p InputPreference) String() string {
	switch p {
	case InputPreferenceCharacterInput:
		return "CharacterInput"
	case InputPreferenceKeyBindings:
		return "KeyBindings"
	}
	return fmt.Sprintf("InputPreference(%d)", uint8(p))
}

// KeystrokeEvent is a keystroke event, and potentially the associated action.
type KeystrokeEvent struct {
	// Keystroke is the keystroke that occurred.
	Keystroke Keystroke
	// InputPreference is whether this keystroke should prefer character input or key bindings.
	InputPreference InputPreference
	// Action is the action that was resolved for the keystroke, if any.
	Action Option[Action]
	// ContextStack is the context stack at the time.
	ContextStack []KeyContext
}

// HTTPClient is the HTTP client used by the application (D-HTTP: upstream's http_client
// crate over net/http).
type HTTPClient interface {
	// Send sends the request; the task yields the response.
	Send(req *http.Request) *Task[*http.Response]
	// UserAgent returns the user agent the client sends, if any.
	UserAgent() (string, bool)
	// Proxy returns the proxy the client uses, if any.
	Proxy() (*url.URL, bool)
}

// NullHTTPClient is an HTTPClient that fails every request.
type NullHTTPClient struct{}

// Send fails with "No HttpClient available".
func (NullHTTPClient) Send(req *http.Request) *Task[*http.Response] {
	return scheduler.ReadyTaskErr[*http.Response](errors.New("No HttpClient available"))
}

// UserAgent returns false.
func (NullHTTPClient) UserAgent() (string, bool) {
	return "", false
}

// Proxy returns false.
func (NullHTTPClient) Proxy() (*url.URL, bool) {
	return nil, false
}

// GpuiBorrow is a mutable reference to an entity owned by GPUI, obtained with AsMut. Release
// notifies the entity, ends the lease and finishes the update, as its Drop does upstream.
type GpuiBorrow[T any] struct {
	state *gpuiBorrowState[T]
}

type gpuiBorrowState[T any] struct {
	inner Option[lease[T]]
	app   *App
}

func newGpuiBorrow[T any](inner Entity[T], app *App) GpuiBorrow[T] {
	app.startUpdate()
	l := app.entities.lease(inner)
	return GpuiBorrow[T]{state: &gpuiBorrowState[T]{inner: Some(l), app: app}}
}

// Get returns the leased entity state. It panics after Release.
func (b GpuiBorrow[T]) Get() *T {
	if b.state == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return b.state.inner.Unwrap().get()
}

// Release notifies the entity's observers, ends the lease and finishes the update. It is
// idempotent.
func (b GpuiBorrow[T]) Release() {
	if b.state == nil {
		return
	}
	l, ok := b.state.inner.Take().Get()
	if !ok {
		return
	}
	b.state.app.Notify(l.id)
	b.state.app.entities.endLease(l)
	b.state.app.finishUpdate()
}
