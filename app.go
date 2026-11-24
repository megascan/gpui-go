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
	"os"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"time"
	"weak"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
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

	cx.windows.clear()
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
