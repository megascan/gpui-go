package gpui

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
	"weak"

	"github.com/megascan/gpui-go/scheduler"
)

func testWindow(app *AppCell, window AnyWindowHandle) *TestWindow {
	lock := app.BorrowMut()
	defer lock.Release()
	w, ok := lock.windows.get(window.id)
	if !ok {
		panic("window not found")
	}
	if w == nil {
		panic("window is being updated")
	}
	tw, ok := w.platformWindow.(*TestWindow)
	if !ok {
		panic("window is not a TestWindow")
	}
	return tw.Clone()
}

type fakeHTTPClient struct {
	statusCode int
}

func fakeHTTPClientWith404Response() HTTPClient {
	return fakeHTTPClient{statusCode: http.StatusNotFound}
}

func (c fakeHTTPClient) Send(req *http.Request) *Task[*http.Response] {
	return scheduler.ReadyTask(&http.Response{
		Status:     fmt.Sprintf("%d %s", c.statusCode, http.StatusText(c.statusCode)),
		StatusCode: c.statusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	})
}

func (fakeHTTPClient) UserAgent() (string, bool) {
	return "", false
}

func (fakeHTTPClient) Proxy() (*url.URL, bool) {
	return nil, false
}

func liveEntityHandleCount(refCounts *entityRefCounts) int {
	refCounts.mu.RLock()
	defer refCounts.mu.RUnlock()
	n := 0
	for _, data := range refCounts.leakDetector.entityHandles {
		n += len(data.handles)
	}
	return n
}

func runPendingCleanups() {
	done := make(chan struct{})
	sentinel := new([64]byte)
	runtime.AddCleanup(sentinel, func(ch chan struct{}) { close(ch) }, done)
	sentinel = nil
	runtime.GC()
	select {
	case <-done:
	case <-time.After(time.Second):
	}
	runtime.Gosched()
}

func releaseAppWindows(app *AppCell) {
	lock := app.BorrowMut()
	defer lock.Release()
	lock.dropWindows()
	releaseActiveDrag(lock.activeDrag.Take())
	clear(lock.windowHandles)
	lock.flushEffects()
}

func reportAppLeaks(app *AppCell) {
	if currentLeakDetectionMode() == leakDetectionOff {
		return
	}
	refCounts := app.app.entities.refCountsDropHandle()
	previous := -1
	stable := 0
	for range 32 {
		runPendingCleanups()
		lock := app.BorrowMut()
		lock.update(func(*App) {})
		lock.Release()
		count := liveEntityHandleCount(refCounts)
		if count == previous {
			stable++
			if stable >= 2 {
				break
			}
		} else {
			stable = 0
		}
		previous = count
	}
	app.app.entities.report()
}

// TestAppContext is provided to tests run with RunTest; it is an AppContext with additional
// methods that are useful in tests.
type TestAppContext struct {
	bgExecutor   BackgroundExecutor
	fgExecutor   ForegroundExecutor
	dispatcher   *TestDispatcher
	app          *AppCell
	testPlatform *TestPlatform
	textSystem   *TextSystem
	fnName       Option[string]
	onQuit       *[]func()
}

var _ AppContext = (*TestAppContext)(nil)

func (cx *TestAppContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.newEntity(entityType, build)
}

func (cx *TestAppContext) reserveEntity(entityType reflect.Type) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.reserveEntity(entityType)
}

func (cx *TestAppContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.insertEntity(slot, build)
}

func (cx *TestAppContext) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.updateEntity(handle, update)
}

func (cx *TestAppContext) asMut(handle AnyEntity) *App {
	panic("Cannot use as_mut with a test app context. Try calling update() first")
}

func (cx *TestAppContext) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readEntity(handle, read)
}

func (cx *TestAppContext) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateWindow(window, update)
}

func (cx *TestAppContext) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.withWindow(entityID, f)
}

func (cx *TestAppContext) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().readWindow(window, read)
}

func (cx *TestAppContext) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *TestAppContext) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readGlobal(globalType, read)
}

// New creates a new entity.
func (cx *TestAppContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later.
func (cx *TestAppContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot reserved with ReserveEntity.
func (cx *TestAppContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *TestAppContext) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on the background executor.
func (cx *TestAppContext) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.bgExecutor.Spawn(future)
}

// SetPlatformInfo sets the runtime identity used by this test's application.
func (cx *TestAppContext) SetPlatformInfo(platformInfo PlatformInfo) {
	cx.testPlatform.platformInfo = platformInfo
}

// TestAppContextBuild creates a new TestAppContext. RunTest usually does this for you.
func TestAppContextBuild(dispatcher *TestDispatcher, fnName Option[string]) *TestAppContext {
	arcDispatcher := dispatcher.Clone()
	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)
	platform := NewTestPlatform(backgroundExecutor, foregroundExecutor)
	return testAppContextBuildWithPlatform(dispatcher, fnName, backgroundExecutor, foregroundExecutor, platform)
}

// TestAppContextBuildWithTextSystem creates a test context backed by the provided platform
// text system.
func TestAppContextBuildWithTextSystem(dispatcher *TestDispatcher, fnName Option[string], platformTextSystem PlatformTextSystem) *TestAppContext {
	arcDispatcher := dispatcher.Clone()
	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)
	platform := TestPlatformWithTextSystem(backgroundExecutor, foregroundExecutor, platformTextSystem)
	return testAppContextBuildWithPlatform(dispatcher, fnName, backgroundExecutor, foregroundExecutor, platform)
}

func testAppContextBuildWithPlatform(dispatcher *TestDispatcher, fnName Option[string], backgroundExecutor BackgroundExecutor, foregroundExecutor ForegroundExecutor, platform *TestPlatform) *TestAppContext {
	assetSource := NoAssets{}
	httpClient := fakeHTTPClientWith404Response()
	textSystem := NewTextSystem(platform.TextSystem())

	app := newApp(platform, assetSource, httpClient)
	lock := app.BorrowMut()
	lock.mode = gpuiModeTest()
	lock.Release()

	return &TestAppContext{
		app:          app,
		bgExecutor:   backgroundExecutor,
		fgExecutor:   foregroundExecutor,
		dispatcher:   dispatcher,
		testPlatform: platform,
		textSystem:   textSystem,
		fnName:       fnName,
		onQuit:       new([]func()),
	}
}

// DisableAccessibility disables accessibility for subsequently created test windows.
func (cx *TestAppContext) DisableAccessibility() {
	cx.Update(func(cx *App) { cx.accessibilityForceDisabled = true })
}

// SkipDrawing skips all drawing operations for the duration of this test.
func (cx *TestAppContext) SkipDrawing() {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	lock.mode = gpuiMode{test: true, skipDrawing: true}
}

// TestAppContextSingle creates a single TestAppContext, for non-multi-client tests. Close ends
// it.
func TestAppContextSingle() *TestAppContext {
	dispatcher := NewTestDispatcher(0)
	return TestAppContextBuild(dispatcher, None[string]())
}

// TestFunctionName returns the name of the test function that created this TestAppContext.
func (cx *TestAppContext) TestFunctionName() (string, bool) {
	return cx.fnName.Get()
}

// DidPromptForNewPath reports whether there have been any new path prompts received by the
// platform.
func (cx *TestAppContext) DidPromptForNewPath() bool {
	return cx.testPlatform.DidPromptForNewPath()
}

// ActiveIdleSleepPreventions returns the number of active idle sleep prevention tokens.
func (cx *TestAppContext) ActiveIdleSleepPreventions() int {
	return cx.testPlatform.ActiveIdleSleepPreventions()
}

// SetIdleSleepPreventionDelay sets the delay for subsequent idle sleep prevention acquisitions.
func (cx *TestAppContext) SetIdleSleepPreventionDelay(delay time.Duration) {
	cx.testPlatform.SetIdleSleepPreventionDelay(delay)
}

// SetIdleSleepPreventionFails makes subsequent idle sleep prevention acquisitions fail or
// succeed.
func (cx *TestAppContext) SetIdleSleepPreventionFails(fails bool) {
	cx.testPlatform.SetIdleSleepPreventionFails(fails)
}

// NewApp returns a new TestAppContext re-using the same executors to interleave tasks.
func (cx *TestAppContext) NewApp() *TestAppContext {
	return TestAppContextBuild(cx.dispatcher.Clone(), cx.fnName)
}

// Quit runs the callbacks registered with OnQuit and shuts the application down.
func (cx *TestAppContext) Quit() {
	callbacks := *cx.onQuit
	*cx.onQuit = nil
	for _, f := range callbacks {
		f()
	}
	lock := cx.app.BorrowMut()
	defer lock.Release()
	lock.Shutdown()
}

// OnQuit registers cleanup to run when the test ends.
func (cx *TestAppContext) OnQuit(f func()) {
	*cx.onQuit = append(*cx.onQuit, f)
}

// Refresh schedules all windows to be redrawn on the next effect cycle.
func (cx *TestAppContext) Refresh() error {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.RefreshWindows()
	return nil
}

// Executor returns an executor for running tasks in the background.
func (cx *TestAppContext) Executor() BackgroundExecutor {
	return cx.bgExecutor
}

// ForegroundExecutor returns an executor for running tasks on the main thread.
func (cx *TestAppContext) ForegroundExecutor() ForegroundExecutor {
	return cx.fgExecutor
}

// Update gives f an *App for the duration of the call and flushes its effects.
func (cx *TestAppContext) Update(f func(cx *App)) {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	lock.update(f)
}

// UpdateR is Update for a callback that returns a value.
func (cx *TestAppContext) UpdateR[R any](f func(cx *App) R) R {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateR(f)
}

// Read gives f a read-only *App for the duration of the call.
func (cx *TestAppContext) Read(f func(cx *App)) {
	lock := cx.app.Borrow()
	defer lock.Release()
	f(lock.Get())
}

// ReadR is Read for a callback that returns a value.
func (cx *TestAppContext) ReadR[R any](f func(cx *App) R) R {
	lock := cx.app.Borrow()
	defer lock.Release()
	return f(lock.Get())
}

func testWindowOptions(bounds Bounds[Pixels]) WindowOptions {
	options := DefaultWindowOptions()
	options.WindowBounds = Some(WindowBoundsWindowed(bounds))
	return options
}

// AddWindow adds a new window backed by a TestWindow, which TestAppContext.testWindow
// retrieves. *V must implement Render[V].
func (cx *TestAppContext) AddWindow[V any](buildWindow func(window *Window, cx *Context[V]) *V) WindowHandle[V] {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	bounds := BoundsMaximized(None[DisplayID](), lock.App)
	handle, err := lock.OpenWindow(testWindowOptions(bounds), func(window *Window, cx *App) Entity[V] {
		return cx.New(func(cx *Context[V]) *V { return buildWindow(window, cx) })
	})
	unwrapTestResult(err)
	return handle
}

// OpenWindow opens a new window with a specific size. Unlike AddWindow, which uses maximized
// bounds, it controls the window dimensions, which matters for layout-sensitive tests.
func (cx *TestAppContext) OpenWindow[V any](windowSize Size[Pixels], buildWindow func(window *Window, cx *Context[V]) *V) WindowHandle[V] {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	handle, err := lock.OpenWindow(testWindowOptions(Bounds[Pixels]{Size: windowSize}), func(window *Window, cx *App) Entity[V] {
		return cx.New(func(cx *Context[V]) *V { return buildWindow(window, cx) })
	})
	unwrapTestResult(err)
	return handle
}

// AddEmptyWindow adds a new window with no content.
func (cx *TestAppContext) AddEmptyWindow() *VisualTestContext {
	lock := cx.app.BorrowMut()
	bounds := BoundsMaximized(None[DisplayID](), lock.App)
	window, err := lock.OpenWindow(testWindowOptions(bounds), func(_ *Window, cx *App) Entity[Empty] {
		return cx.New(func(*Context[Empty]) *Empty { return &Empty{} })
	})
	lock.Release()
	unwrapTestResult(err)
	vcx := VisualTestContextFromWindow(window.AnyWindowHandle, cx).IntoMut()
	vcx.RunUntilParked()
	return vcx
}

// AddWindowView adds a new window and returns its root view and a VisualTestContext which can
// be used as a Window and App for the rest of the test. The caller releases the view.
func (cx *TestAppContext) AddWindowView[V any](buildRootView func(window *Window, cx *Context[V]) *V) (Entity[V], *VisualTestContext) {
	lock := cx.app.BorrowMut()
	bounds := BoundsMaximized(None[DisplayID](), lock.App)
	window, err := lock.OpenWindow(testWindowOptions(bounds), func(window *Window, cx *App) Entity[V] {
		return cx.New(func(cx *Context[V]) *V { return buildRootView(window, cx) })
	})
	lock.Release()
	unwrapTestResult(err)
	view, err := window.Root(cx)
	unwrapTestResult(err)
	vcx := VisualTestContextFromWindow(window.AnyWindowHandle, cx).IntoMut()
	vcx.RunUntilParked()
	return view, vcx
}

// TextSystem returns the TextSystem.
func (cx *TestAppContext) TextSystem() *TextSystem {
	return cx.textSystem
}

// WriteToClipboard simulates writing to the platform clipboard.
func (cx *TestAppContext) WriteToClipboard(item ClipboardItem) {
	cx.testPlatform.WriteToClipboard(item)
}

// ReadFromClipboard simulates reading from the platform clipboard. It returns the most recent
// value from WriteToClipboard.
func (cx *TestAppContext) ReadFromClipboard() (ClipboardItem, bool) {
	return cx.testPlatform.ReadFromClipboard()
}

// SimulateNewPathSelection simulates choosing a file in the platform's "Save" dialog.
func (cx *TestAppContext) SimulateNewPathSelection(selectPath func(path string) Option[string]) {
	cx.testPlatform.SimulateNewPathSelection(selectPath)
}

// SimulatePathPromptResponse simulates responding to a PromptForPaths ("Open") dialog.
func (cx *TestAppContext) SimulatePathPromptResponse(selectPaths func(options *PathPromptOptions) Option[[]string]) {
	cx.testPlatform.SimulatePathPromptResponse(selectPaths)
}

// DidPromptForPaths reports whether there's a path selection dialog pending.
func (cx *TestAppContext) DidPromptForPaths() bool {
	return cx.testPlatform.DidPromptForPaths()
}

// SimulatePromptAnswer simulates clicking a button in a platform-level alert dialog.
func (cx *TestAppContext) SimulatePromptAnswer(button string) {
	cx.testPlatform.SimulatePromptAnswer(button)
}

// HasPendingPrompt reports whether there's an alert dialog open.
func (cx *TestAppContext) HasPendingPrompt() bool {
	return cx.testPlatform.HasPendingPrompt()
}

// PendingPrompt returns the message and detail of the open alert dialog.
func (cx *TestAppContext) PendingPrompt() (msg string, detail string, ok bool) {
	return cx.testPlatform.PendingPrompt()
}

// OpenedURL returns the URL most recently opened with App.OpenURL during this test.
func (cx *TestAppContext) OpenedURL() (string, bool) {
	return cx.testPlatform.openedURL.Get()
}

// AppIdentity returns the application identity configured during this test.
func (cx *TestAppContext) AppIdentity() (identifier SharedString, name SharedString, ok bool) {
	return cx.testPlatform.AppIdentity()
}

// ShownSystemNotifications returns all system notifications shown during this test, in order.
func (cx *TestAppContext) ShownSystemNotifications() []SystemNotification {
	return cx.testPlatform.ShownSystemNotifications()
}

// DeliveredSystemNotifications returns the system notifications currently delivered by the
// test platform.
func (cx *TestAppContext) DeliveredSystemNotifications() []SystemNotification {
	return cx.testPlatform.DeliveredSystemNotifications()
}

// DismissedSystemNotifications returns the tags of all system notifications dismissed during
// this test, in order.
func (cx *TestAppContext) DismissedSystemNotifications() []SharedString {
	return cx.testPlatform.DismissedSystemNotifications()
}

// SimulateSystemNotificationResponse simulates the user activating a system notification.
func (cx *TestAppContext) SimulateSystemNotificationResponse(response SystemNotificationResponse) {
	cx.testPlatform.SimulateSystemNotificationResponse(response)
}

// SimulateWindowResize simulates the user resizing the window to the new size.
func (cx *TestAppContext) SimulateWindowResize(windowHandle AnyWindowHandle, size Size[Pixels]) {
	cx.testWindow(windowHandle).SimulateResize(size)
}

// SimulateWindowVisibilityChange simulates a change in whether the platform is presenting the
// window.
func (cx *TestAppContext) SimulateWindowVisibilityChange(windowHandle AnyWindowHandle, visibility WindowVisibility) {
	cx.testWindow(windowHandle).SimulateVisibilityChange(visibility)
}

// SimulateWindowDecorationsChange simulates the platform changing how the window is decorated.
func (cx *TestAppContext) SimulateWindowDecorationsChange(windowHandle AnyWindowHandle, decorations Decorations) {
	cx.testWindow(windowHandle).SimulateDecorationsChange(decorations)
}

// SimulateDisplayAdded simulates connecting a display.
func (cx *TestAppContext) SimulateDisplayAdded(displayID DisplayID, refreshInterval Option[time.Duration]) {
	cx.testPlatform.SimulateDisplayAdded(displayID, refreshInterval)
}

// SimulateDisplayRemoved simulates disconnecting a display.
func (cx *TestAppContext) SimulateDisplayRemoved(displayID DisplayID) {
	cx.testPlatform.SimulateDisplayRemoved(displayID)
}

// SimulateDisplayRefreshIntervalChange simulates a connected display changing its refresh
// interval.
func (cx *TestAppContext) SimulateDisplayRefreshIntervalChange(displayID DisplayID, refreshInterval Option[time.Duration]) {
	cx.testPlatform.SimulateDisplayRefreshIntervalChange(displayID, refreshInterval)
}

// SimulateWindowVisualViewportChange simulates visible viewport changes without resizing the
// window's layout area.
func (cx *TestAppContext) SimulateWindowVisualViewportChange(windowHandle AnyWindowHandle, bounds Bounds[Pixels]) {
	cx.testWindow(windowHandle).SimulateVisualViewportChange(bounds)
}

// SimulateWindowScaleFactorChange simulates the window moving to a display with a different
// scale factor.
func (cx *TestAppContext) SimulateWindowScaleFactorChange(windowHandle AnyWindowHandle, scaleFactor float32) {
	cx.testWindow(windowHandle).SimulateScaleFactorChange(scaleFactor)
}

// ExpectRestart returns a receiver for the binary path and arguments of the next restart the
// application requests.
func (cx *TestAppContext) ExpectRestart() *scheduler.OneshotReceiver[TestRestartRequest] {
	tx, rx := scheduler.Oneshot[TestRestartRequest]()
	cx.testPlatform.expectRestart = tx
	return rx
}

// SetScreenCaptureSources causes the given sources to be returned if the application queries
// for screen capture sources.
func (cx *TestAppContext) SetScreenCaptureSources(sources []TestScreenCaptureSource) {
	cx.testPlatform.SetScreenCaptureSources(sources)
}

// Windows returns all windows open in the test.
func (cx *TestAppContext) Windows() []AnyWindowHandle {
	lock := cx.app.Borrow()
	defer lock.Release()
	return lock.Get().Windows()
}

// Spawn runs the given task on the main thread.
func (cx *TestAppContext) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R] {
	acx := cx.ToAsync()
	return cx.fgExecutor.Spawn(func(s *scheduler.Scope) (R, error) {
		acx.Scope = s
		return f(acx)
	})
}

// HasGlobal reports whether the global of type G is defined.
func (cx *TestAppContext) HasGlobal[G any]() bool {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().HasGlobal[G]()
}

// ReadGlobal runs read with a reference to the global of type G. It panics if HasGlobal would
// report false.
func (cx *TestAppContext) ReadGlobal[G, R any](read func(global *G, cx *App) R) R {
	app := cx.app.Borrow()
	defer app.Release()
	return read(app.Get().Global[G](), app.Get())
}

// TryReadGlobal runs read with a reference to the global of type G, if it is set.
func (cx *TestAppContext) TryReadGlobal[G, R any](read func(global *G, cx *App) R) (R, bool) {
	lock := cx.app.Borrow()
	defer lock.Release()
	global, ok := lock.Get().TryGlobal[G]()
	if !ok {
		var zero R
		return zero, false
	}
	return read(global, lock.Get()), true
}

// SetGlobal sets the global in this context.
func (cx *TestAppContext) SetGlobal[G any](global G) {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	lock.update(func(cx *App) { cx.SetGlobal(global) })
}

// UpdateGlobal updates the global in this context. It panics if HasGlobal would report false.
func (cx *TestAppContext) UpdateGlobal[G any](update func(global *G, cx *App)) {
	cx.UpdateGlobalR[G](func(global *G, cx *App) struct{} {
		update(global, cx)
		return struct{}{}
	})
}

// UpdateGlobalR is UpdateGlobal for a callback that returns a value.
func (cx *TestAppContext) UpdateGlobalR[G, R any](update func(global *G, cx *App) R) R {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateR(func(cx *App) R { return UpdateGlobalR(cx, update) })
}

// ToAsync returns an AsyncApp which can be used to run tasks that expect to be on a background
// thread on the current thread in tests.
func (cx *TestAppContext) ToAsync() *AsyncApp {
	return &AsyncApp{
		app:                weak.Make(cx.app),
		bgExecutor:         cx.bgExecutor,
		foregroundExecutor: cx.fgExecutor,
	}
}

// RunUntilParked waits until there are no more pending tasks.
func (cx *TestAppContext) RunUntilParked() {
	cx.dispatcher.RunUntilParked()
}

// DispatchAction simulates dispatching an action to the currently focused node in the window.
func (cx *TestAppContext) DispatchAction(window AnyWindowHandle, action Action) {
	unwrapTestResult(window.Update(cx, func(_ AnyView, window *Window, cx *App) {
		window.DispatchAction(action.BoxedClone(), cx)
	}))
	cx.bgExecutor.RunUntilParked()
}

func parseKeystrokes(sources []string, os OperatingSystem) []Keystroke {
	keystrokes := make([]Keystroke, 0, len(sources))
	for _, source := range sources {
		keystroke, err := KeystrokeParseWithOs(source, os)
		unwrapTestResult(err)
		keystrokes = append(keystrokes, keystroke)
	}
	return keystrokes
}

func splitEmpty(input string) []string {
	parts := []string{""}
	for len(input) > 0 {
		_, size := utf8.DecodeRuneInString(input)
		parts = append(parts, input[:size])
		input = input[size:]
	}
	return append(parts, "")
}

// SimulateKeystrokes types a space-separated list of keys, as in
// cx.SimulateKeystrokes(window, "cmd-shift-p b k s p enter"), then runs the background executor
// until it is parked.
func (cx *TestAppContext) SimulateKeystrokes(window AnyWindowHandle, keystrokes string) {
	os := cx.ReadR(func(cx *App) OperatingSystem { return cx.PlatformInfo().Os })
	for _, keystroke := range parseKeystrokes(strings.Split(keystrokes, " "), os) {
		cx.DispatchKeystroke(window, keystroke)
	}
	cx.bgExecutor.RunUntilParked()
}

// SimulateInput types a string of text into the window, then runs the background executor
// until it is parked.
func (cx *TestAppContext) SimulateInput(window AnyWindowHandle, input string) {
	os := cx.ReadR(func(cx *App) OperatingSystem { return cx.PlatformInfo().Os })
	for _, keystroke := range parseKeystrokes(splitEmpty(input), os) {
		cx.DispatchKeystroke(window, keystroke)
	}
	cx.bgExecutor.RunUntilParked()
}

// DispatchKeystroke dispatches a single keystroke (see also SimulateKeystrokes and
// SimulateInput).
func (cx *TestAppContext) DispatchKeystroke(window AnyWindowHandle, keystroke Keystroke) {
	_, err := AppContextUpdateWindow(cx, window, func(_ AnyView, window *Window, cx *App) bool {
		return window.DispatchKeystroke(keystroke, cx)
	})
	unwrapTestResult(err)
}

func (cx *TestAppContext) testWindow(window AnyWindowHandle) *TestWindow {
	return testWindow(cx.app, window)
}

// Notifications returns a stream of notifications whenever the entity is updated. The stream
// ends when the entity is released.
func (cx *TestAppContext) Notifications[T any](entity Entity[T]) *scheduler.ChanReceiver[struct{}] {
	tx, rx := scheduler.Chan[struct{}](0)
	cx.Update(func(cx *App) {
		cx.Observe(entity, func(Entity[T], *App) {
			_ = tx.TrySend(struct{}{})
		}).Detach()
		cx.ObserveRelease(entity, func(*T, *App) { tx.Close() }).Detach()
	})
	return rx
}

// Events returns a stream of the events of type E emitted by the given entity.
func (cx *TestAppContext) Events[T, E any](entity Entity[T]) *scheduler.ChanReceiver[E] {
	tx, rx := scheduler.Chan[E](0)
	entity.UpdateR(cx, func(_ *T, cx *Context[T]) Subscription {
		return cx.Subscribe(entity, func(_ *T, _ Entity[T], event *E, _ *Context[T]) {
			_ = tx.TrySend(*event)
		})
	}).Detach()
	return rx
}

// Condition runs until predicate holds for the entity, failing after three seconds of
// simulated time. Prefer RunUntilParked if you don't need to jump in at a specific time.
func (cx *TestAppContext) Condition[T any](s scheduler.Awaiter, entity Entity[T], predicate func(entity *T, cx *Context[T]) bool) {
	timer := cx.Executor().Timer(3 * time.Second)
	notifications := cx.Notifications(entity)
	defer notifications.Close()

	loop := scheduler.Async(func(s *scheduler.Scope) error {
		for {
			if entity.UpdateR(cx, predicate) {
				return nil
			}
			if _, ok := notifications.Recv(s); !ok {
				return errors.New("entity dropped")
			}
		}
	})
	var result error
	scheduler.Select(s,
		scheduler.Case(loop, func(err error) { result = err }),
		scheduler.Case[scheduler.Result[struct{}]](timer, func(scheduler.Result[struct{}]) {
			result = errors.New("condition timed out")
		}),
	)
	unwrapTestResult(result)
}

// SetName sets a name for this App.
func (cx *TestAppContext) SetName(name string) {
	cx.Update(func(cx *App) { cx.name = Some(name) })
}

func (cx *TestAppContext) release() {
	callbacks := *cx.onQuit
	*cx.onQuit = nil
	for _, f := range callbacks {
		f()
	}
	releaseAppWindows(cx.app)
}

// Close ends a context made with TestAppContextSingle or NewApp outside RunTest: it releases
// the app's windows, drops the scheduler's remaining tasks and reports leaked handles, as
// dropping the context does upstream.
func (cx *TestAppContext) Close() {
	callbacks := *cx.onQuit
	*cx.onQuit = nil
	for _, f := range callbacks {
		f()
	}
	releaseAppWindows(cx.app)
	cx.dispatcher.DrainTasks()
	reportAppLeaks(cx.app)
}

// NextEvent returns a future that resolves to the next event of type E emitted by the entity.
func (e Entity[T]) NextEvent[E any](cx *TestAppContext) scheduler.Future[E] {
	tx, rx := scheduler.Oneshot[E]()
	sent := false
	subscription := e.UpdateR(cx, func(_ *T, cx *Context[T]) Subscription {
		return cx.Subscribe(e, func(_ *T, _ Entity[T], event *E, _ *Context[T]) {
			if !sent {
				sent = true
				_ = tx.Send(*event)
			}
		})
	})
	return scheduler.Async(func(s *scheduler.Scope) E {
		event, err := rx.Recv(s)
		if err != nil {
			panic("no event emitted")
		}
		subscription.Unsubscribe()
		return event
	})
}

// NextNotification returns a future that resolves when the entity is next notified, after
// advancing the clock by advanceClockBy.
func (e Entity[T]) NextNotification(advanceClockBy time.Duration, cx *TestAppContext) scheduler.Future[struct{}] {
	tx, rx := scheduler.Chan[struct{}](1)
	lock := cx.app.BorrowMut()
	subscription := lock.Observe(e, func(Entity[T], *App) {
		_ = tx.TrySend(struct{}{})
	})
	lock.Release()

	cx.Executor().AdvanceClock(advanceClockBy)

	return scheduler.Async(func(s *scheduler.Scope) struct{} {
		if _, ok := rx.Recv(s); !ok {
			panic("entity dropped while test was waiting for its next notification")
		}
		subscription.Unsubscribe()
		return struct{}{}
	})
}

// Condition returns a future that resolves when predicate holds for the entity. It is checked
// whenever the entity is notified or emits an event of type Evt.
func (e Entity[T]) Condition[Evt any](cx *TestAppContext, predicate func(entity *T, cx *App) bool) scheduler.Future[struct{}] {
	tx, rx := scheduler.Chan[struct{}](1024)

	lock := cx.app.BorrowMut()
	observation := lock.Observe(e, func(Entity[T], *App) {
		_ = tx.TrySend(struct{}{})
	})
	subscription := lock.Subscribe(e, func(Entity[T], *Evt, *App) {
		_ = tx.TrySend(struct{}{})
	})
	app := lock.this.Value()
	lock.Release()
	if app == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	handle := e.Downgrade()

	return scheduler.Async(func(s *scheduler.Scope) struct{} {
		for {
			done := func() bool {
				borrow := app.Borrow()
				defer borrow.Release()
				entity, ok := handle.Upgrade()
				if !ok {
					panic("view dropped with pending condition")
				}
				defer entity.Release()
				return predicate(entity.Read(borrow.Get()), borrow.Get())
			}()
			if done {
				break
			}
			if _, ok := rx.Recv(s); !ok {
				panic("view dropped with pending condition")
			}
		}
		observation.Unsubscribe()
		subscription.Unsubscribe()
		return struct{}{}
	})
}

// VisualTestContext is the test equivalent of a Window and App. It runs window-specific test
// code and embeds the TestAppContext it was made from (upstream's Deref).
type VisualTestContext struct {
	*TestAppContext
	window AnyWindowHandle
}

var _ VisualContext = (*VisualTestContext)(nil)

// Update provides a Window and App for the duration of f.
func (cx *VisualTestContext) Update(f func(window *Window, cx *App)) {
	unwrapTestResult(cx.TestAppContext.updateWindow(cx.window, func(_ AnyView, window *Window, cx *App) {
		f(window, cx)
	}))
}

// UpdateR is Update for a callback that returns a value.
func (cx *VisualTestContext) UpdateR[R any](f func(window *Window, cx *App) R) R {
	result, err := AppContextUpdateWindow(cx.TestAppContext, cx.window, func(_ AnyView, window *Window, cx *App) R {
		return f(window, cx)
	})
	unwrapTestResult(err)
	return result
}

// VisualTestContextFromWindow creates a new VisualTestContext for the window.
func VisualTestContextFromWindow(window AnyWindowHandle, cx *TestAppContext) *VisualTestContext {
	return &VisualTestContext{TestAppContext: cx, window: window}
}

// RunUntilParked waits until there are no more pending tasks.
func (cx *VisualTestContext) RunUntilParked() {
	cx.bgExecutor.RunUntilParked()
}

// DispatchAction dispatches the action to the currently focused node.
func (cx *VisualTestContext) DispatchAction(action Action) {
	cx.TestAppContext.DispatchAction(cx.window, action)
}

// WindowTitle reads the title off the window, as set by Window.SetWindowTitle.
func (cx *VisualTestContext) WindowTitle() (string, bool) {
	tw := cx.testWindow(cx.window)
	state := tw.lock()
	defer tw.unlock()
	return state.title.Get()
}

// DocumentPath reads the document path off the window, as set by Window.SetDocumentPath.
func (cx *VisualTestContext) DocumentPath() (string, bool) {
	tw := cx.testWindow(cx.window)
	state := tw.lock()
	defer tw.unlock()
	return state.documentPath.Get()
}

// SimulateKeystrokes simulates a sequence of keystrokes, as in cx.SimulateKeystrokes("cmd-p
// escape"), and runs until parked.
func (cx *VisualTestContext) SimulateKeystrokes(keystrokes string) {
	cx.TestAppContext.SimulateKeystrokes(cx.window, keystrokes)
}

// SimulateInput simulates typing text and runs until parked.
func (cx *VisualTestContext) SimulateInput(input string) {
	cx.TestAppContext.SimulateInput(cx.window, input)
}

// SimulateMouseMove simulates a mouse move event to the given point.
func (cx *VisualTestContext) SimulateMouseMove(position Point[Pixels], button Option[MouseButton], modifiers Modifiers) {
	cx.SimulateEvent(MouseMoveEvent{
		Position:      position,
		Modifiers:     modifiers,
		PressedButton: button,
	})
}

// SimulateMouseDown simulates a mouse down event at the given point.
func (cx *VisualTestContext) SimulateMouseDown(position Point[Pixels], button MouseButton, modifiers Modifiers) {
	cx.SimulateEvent(MouseDownEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     button,
		ClickCount: 1,
		FirstMouse: false,
	})
}

// SimulateMouseUp simulates a mouse up event at the given point.
func (cx *VisualTestContext) SimulateMouseUp(position Point[Pixels], button MouseButton, modifiers Modifiers) {
	cx.SimulateEvent(MouseUpEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     button,
		ClickCount: 1,
	})
}

// SimulateClick simulates a primary mouse click at the given point.
func (cx *VisualTestContext) SimulateClick(position Point[Pixels], modifiers Modifiers) {
	cx.SimulateEvent(MouseDownEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     MouseButtonLeft,
		ClickCount: 1,
		FirstMouse: false,
	})
	cx.SimulateEvent(MouseUpEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     MouseButtonLeft,
		ClickCount: 1,
	})
}

// SimulateModifiersChange simulates a modifiers changed event.
func (cx *VisualTestContext) SimulateModifiersChange(modifiers Modifiers) {
	cx.SimulateEvent(ModifiersChangedEvent{
		Modifiers: modifiers,
		Capslock:  Capslock{On: false},
	})
}

// SimulateCapslockChange simulates a capslock changed event.
func (cx *VisualTestContext) SimulateCapslockChange(on bool) {
	cx.SimulateEvent(ModifiersChangedEvent{
		Modifiers: Modifiers{},
		Capslock:  Capslock{On: on},
	})
}

// SimulateResize simulates the user resizing the window to the new size.
func (cx *VisualTestContext) SimulateResize(size Size[Pixels]) {
	cx.SimulateWindowResize(cx.window, size)
}

// SimulateVisibilityChange simulates a change in whether the platform is presenting this
// window.
func (cx *VisualTestContext) SimulateVisibilityChange(visibility WindowVisibility) {
	cx.SimulateWindowVisibilityChange(cx.window, visibility)
}

// SimulateScaleFactorChange simulates the window moving to a display with a different scale
// factor.
func (cx *VisualTestContext) SimulateScaleFactorChange(scaleFactor float32) {
	cx.SimulateWindowScaleFactorChange(cx.window, scaleFactor)
}

// DebugBounds returns the bounds of the element with the given debug selector.
func (cx *VisualTestContext) DebugBounds(selector string) (Bounds[Pixels], bool) {
	type result struct {
		bounds Bounds[Pixels]
		ok     bool
	}
	r := cx.UpdateR(func(window *Window, _ *App) result {
		bounds, ok := window.renderedFrame.debugBounds[selector]
		return result{bounds, ok}
	})
	return r.bounds, r.ok
}

// Draw draws an element to the window, which is useful for simulating events or actions.
func (cx *VisualTestContext) Draw[RL, PP any](origin Point[Pixels], space SizeOf[AvailableSpace], f func(window *Window, cx *App) Element[RL, PP]) (RL, PP) {
	type states struct {
		requestLayout RL
		prepaint      PP
	}
	r := cx.UpdateR(func(window *Window, cx *App) states {
		window.invalidator.setPhase(drawPhasePrepaint)
		element := newDrawable(f(window, cx))
		element.layoutAsRoot(space, window, cx)
		window.WithAbsoluteElementOffset(origin, func(window *Window) struct{} {
			element.prepaint(window, cx)
			return struct{}{}
		})

		window.invalidator.setPhase(drawPhasePaint)
		requestLayoutState, prepaintState := element.paint(window, cx)

		window.invalidator.setPhase(drawPhaseNone)
		window.Refresh()

		return states{requestLayoutState, prepaintState}
	})
	return r.requestLayout, r.prepaint
}

// SimulateEvent simulates an event from the platform, such as a ScrollEvent. Call Draw first.
func (cx *VisualTestContext) SimulateEvent(event InputEvent) {
	cx.testWindow(cx.window).SimulateInput(event.ToPlatformInput())
	cx.bgExecutor.RunUntilParked()
}

// DeactivateWindow simulates the user blurring the window.
func (cx *VisualTestContext) DeactivateWindow() {
	if active, ok := cx.testPlatform.ActiveWindow(); ok && active == cx.window {
		cx.testPlatform.SetActiveWindow(nil)
	}
	cx.bgExecutor.RunUntilParked()
}

// SimulateClose simulates the user closing the window. It reports whether the window was
// closed.
func (cx *VisualTestContext) SimulateClose() bool {
	handler, err := AppContextUpdateWindow(cx.TestAppContext, cx.window, func(_ AnyView, window *Window, _ *App) func() bool {
		tw, ok := window.platformWindow.(*TestWindow)
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		state := tw.lock()
		defer tw.unlock()
		handler := state.shouldCloseHandler
		state.shouldCloseHandler = nil
		return handler
	})
	unwrapTestResult(err)
	if handler == nil {
		return false
	}
	shouldClose := handler()
	unwrapTestResult(cx.TestAppContext.updateWindow(cx.window, func(_ AnyView, window *Window, _ *App) {
		window.platformWindow.OnShouldClose(handler)
	}))
	return shouldClose
}

// IntoMut returns the context itself; it stays reachable for the rest of the test (D-RAII).
func (cx *VisualTestContext) IntoMut() *VisualTestContext {
	return cx
}

func (cx *VisualTestContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	var entity AnyEntity
	err := cx.window.Update(cx.TestAppContext, func(_ AnyView, _ *Window, app *App) {
		entity = app.newEntity(entityType, build)
	})
	if err != nil {
		panic("window was unexpectedly closed")
	}
	return entity
}

func (cx *VisualTestContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	var entity AnyEntity
	err := cx.window.Update(cx.TestAppContext, func(_ AnyView, _ *Window, app *App) {
		entity = app.insertEntity(slot, build)
	})
	if err != nil {
		panic("window was unexpectedly closed")
	}
	return entity
}

func (cx *VisualTestContext) updateWindowContext(update func(window *Window, cx *App)) error {
	err := cx.TestAppContext.updateWindow(cx.window, func(_ AnyView, window *Window, cx *App) {
		update(window, cx)
	})
	if err != nil {
		panic("window was unexpectedly closed")
	}
	return nil
}

// WindowHandle returns the window handle underlying this context.
func (cx *VisualTestContext) WindowHandle() AnyWindowHandle {
	return cx.window
}

// New creates a new entity in this context's window.
func (cx *VisualTestContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// InsertEntity inserts an entity into a reserved slot in this context's window.
func (cx *VisualTestContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// NewWindowEntity creates a new entity with access to the window.
func (cx *VisualTestContext) NewWindowEntity[T any](buildEntity func(window *Window, cx *Context[T]) *T) Entity[T] {
	entity, _ := VisualContextNewWindowEntity(cx, buildEntity)
	return entity
}

// UpdateWindowEntity updates a view in its current window. It panics if the entity has no
// current window.
func (cx *VisualTestContext) UpdateWindowEntity[V, R any](view Entity[V], update func(view *V, window *Window, cx *Context[V]) R) R {
	view = view.Clone()
	defer view.Release()
	lock := cx.app.BorrowMut()
	defer lock.Release()
	result, ok := lock.WithWindow(view.EntityID(), func(window *Window, app *App) R {
		return view.UpdateR(app, func(v *V, cx *Context[V]) R { return update(v, window, cx) })
	})
	if !ok {
		panic("entity has no current window; use `update` instead of `update_in`")
	}
	return result
}

// ReplaceRootView replaces the root view of the window. *V must implement Render[V].
func (cx *VisualTestContext) ReplaceRootView[V any](buildView func(window *Window, cx *Context[V]) *V) Entity[V] {
	entity, _ := VisualContextReplaceRootView(cx, buildView)
	return entity
}

// Focus focuses the view in the window. *V must implement Focusable.
func (cx *VisualTestContext) Focus[V any](view Entity[V]) {
	_ = VisualContextFocus(cx, view)
}

// BuildEntity creates the given view in this window. *V must implement Render[V].
func (h AnyWindowHandle) BuildEntity[V any](cx *TestAppContext, buildView func(window *Window, cx *Context[V]) *V) Entity[V] {
	var entity Entity[V]
	unwrapTestResult(h.Update(cx, func(_ AnyView, window *Window, cx *App) {
		entity = cx.New(func(cx *Context[V]) *V { return buildView(window, cx) })
	}))
	return entity
}

func unwrapTestResult(err error) {
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
}
