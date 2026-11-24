package gpui

import (
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// TestApp is a test application context with a clean API. Unlike TestAppContext, it flushes
// effects and runs until parked after each update and provides simpler window management.
// Close ends it, as dropping it does upstream.
type TestApp struct {
	app                *AppCell
	platform           *TestPlatform
	bgExecutor         BackgroundExecutor
	foregroundExecutor ForegroundExecutor
	dispatcher         *TestDispatcher
	textSystem         *TextSystem
}

// NewTestApp creates a new test application.
func NewTestApp() *TestApp {
	return TestAppWithSeed(0)
}

// TestAppWithSeed creates a new test application with a specific random seed.
func TestAppWithSeed(seed uint64) *TestApp {
	return buildTestApp(seed, nil, NoAssets{})
}

// TestAppWithTextSystem creates a new test application with a custom text system for real
// font shaping.
func TestAppWithTextSystem(textSystem PlatformTextSystem) *TestApp {
	return buildTestApp(0, textSystem, NoAssets{})
}

// TestAppWithTextSystemAndAssets creates a new test application with a custom text system and
// asset source.
func TestAppWithTextSystemAndAssets(textSystem PlatformTextSystem, assetSource AssetSource) *TestApp {
	return buildTestApp(0, textSystem, assetSource)
}

func buildTestApp(seed uint64, platformTextSystem PlatformTextSystem, assetSource AssetSource) *TestApp {
	dispatcher := NewTestDispatcher(seed)
	arcDispatcher := dispatcher.Clone()
	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)
	var platform *TestPlatform
	if platformTextSystem != nil {
		platform = TestPlatformWithTextSystem(backgroundExecutor, foregroundExecutor, platformTextSystem)
	} else {
		platform = NewTestPlatform(backgroundExecutor, foregroundExecutor)
	}
	httpClient := fakeHTTPClientWith404Response()
	if platformTextSystem == nil {
		platformTextSystem = platform.textSystem
	}
	textSystem := NewTextSystem(platformTextSystem)

	app := newApp(platform, assetSource, httpClient)
	lock := app.BorrowMut()
	lock.mode = gpuiModeTest()
	lock.Release()

	return &TestApp{
		app:                app,
		platform:           platform,
		bgExecutor:         backgroundExecutor,
		foregroundExecutor: foregroundExecutor,
		dispatcher:         dispatcher,
		textSystem:         textSystem,
	}
}

// Update runs f with mutable access to the App and runs until parked afterwards.
func (a *TestApp) Update(f func(cx *App)) {
	updateTestApp(a, func(cx *App) struct{} {
		f(cx)
		return struct{}{}
	})
}

// UpdateR is Update for a callback that returns a value.
func (a *TestApp) UpdateR[R any](f func(cx *App) R) R {
	return updateTestApp(a, f)
}

func updateTestApp[R any](a *TestApp, f func(cx *App) R) R {
	result := func() R {
		app := a.app.BorrowMut()
		defer app.Release()
		return app.updateR(f)
	}()
	a.RunUntilParked()
	return result
}

// Read runs f with read-only access to the App.
func (a *TestApp) Read(f func(cx *App)) {
	app := a.app.Borrow()
	defer app.Release()
	f(app.Get())
}

// ReadR is Read for a callback that returns a value.
func (a *TestApp) ReadR[R any](f func(cx *App) R) R {
	app := a.app.Borrow()
	defer app.Release()
	return f(app.Get())
}

// NewEntity creates a new entity in the app.
func (a *TestApp) NewEntity[T any](build func(cx *Context[T]) *T) Entity[T] {
	return a.UpdateR(func(cx *App) Entity[T] { return cx.New(build) })
}

// UpdateEntity updates an entity.
func (a *TestApp) UpdateEntity[T any](entity Entity[T], f func(entity *T, cx *Context[T])) {
	a.Update(func(cx *App) { entity.Update(cx, f) })
}

// UpdateEntityR is UpdateEntity for a callback that returns a value.
func (a *TestApp) UpdateEntityR[T, R any](entity Entity[T], f func(entity *T, cx *Context[T]) R) R {
	return a.UpdateR(func(cx *App) R { return entity.UpdateR(cx, f) })
}

// ReadEntity reads an entity.
func (a *TestApp) ReadEntity[T any](entity Entity[T], f func(entity *T, cx *App)) {
	a.Read(func(cx *App) { f(entity.Read(cx), cx) })
}

// ReadEntityR is ReadEntity for a callback that returns a value.
func (a *TestApp) ReadEntityR[T, R any](entity Entity[T], f func(entity *T, cx *App) R) R {
	return a.ReadR(func(cx *App) R { return f(entity.Read(cx), cx) })
}

// OpenWindow opens a test window with the given root view, using maximized bounds. *V must
// implement Render[V].
func (a *TestApp) OpenWindow[V any](buildView func(window *Window, cx *Context[V]) *V) *TestAppWindow[V] {
	bounds := a.ReadR(func(cx *App) Bounds[Pixels] { return BoundsMaximized(None[DisplayID](), cx) })
	handle := a.UpdateR(func(cx *App) WindowHandle[V] {
		options := DefaultWindowOptions()
		options.WindowBounds = Some(WindowBoundsWindowed(bounds))
		handle, err := cx.OpenWindow(options, func(window *Window, cx *App) Entity[V] {
			return cx.New(func(cx *Context[V]) *V { return buildView(window, cx) })
		})
		unwrapTestResult(err)
		return handle
	})

	return &TestAppWindow[V]{
		handle:     handle,
		app:        a.app,
		platform:   a.platform,
		bgExecutor: a.bgExecutor,
	}
}

// OpenWindowWithOptions opens a test window with specific options. *V must implement
// Render[V].
func (a *TestApp) OpenWindowWithOptions[V any](options WindowOptions, buildView func(window *Window, cx *Context[V]) *V) *TestAppWindow[V] {
	handle := a.UpdateR(func(cx *App) WindowHandle[V] {
		handle, err := cx.OpenWindow(options, func(window *Window, cx *App) Entity[V] {
			return cx.New(func(cx *Context[V]) *V { return buildView(window, cx) })
		})
		unwrapTestResult(err)
		return handle
	})

	return &TestAppWindow[V]{
		handle:     handle,
		app:        a.app,
		platform:   a.platform,
		bgExecutor: a.bgExecutor,
	}
}

// RunUntilParked runs pending tasks until there's nothing left to do.
func (a *TestApp) RunUntilParked() {
	a.bgExecutor.RunUntilParked()
}

// AdvanceClock advances the simulated clock by the given duration.
func (a *TestApp) AdvanceClock(duration time.Duration) {
	a.bgExecutor.AdvanceClock(duration)
}

// Spawn spawns a future on the foreground executor.
func (a *TestApp) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R] {
	acx := a.ToAsync()
	return a.foregroundExecutor.Spawn(func(s *scheduler.Scope) (R, error) {
		acx.Scope = s
		return f(acx)
	})
}

// BackgroundSpawn spawns a future on the background executor.
func (a *TestApp) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return a.bgExecutor.Spawn(future)
}

// ToAsync returns an async handle to the app.
func (a *TestApp) ToAsync() *AsyncApp {
	return &AsyncApp{
		app:                a.app.app.this,
		bgExecutor:         a.bgExecutor,
		foregroundExecutor: a.foregroundExecutor,
	}
}

// BackgroundExecutor returns the background executor.
func (a *TestApp) BackgroundExecutor() BackgroundExecutor {
	return a.bgExecutor
}

// ForegroundExecutor returns the foreground executor.
func (a *TestApp) ForegroundExecutor() ForegroundExecutor {
	return a.foregroundExecutor
}

// TextSystem returns the text system.
func (a *TestApp) TextSystem() *TextSystem {
	return a.textSystem
}

// HasGlobal reports whether a global of type G exists.
func (a *TestApp) HasGlobal[G any]() bool {
	return a.ReadR(func(cx *App) bool { return cx.HasGlobal[G]() })
}

// SetGlobal sets a global value.
func (a *TestApp) SetGlobal[G any](global G) {
	a.Update(func(cx *App) { cx.SetGlobal(global) })
}

// ReadGlobal reads a global value.
func (a *TestApp) ReadGlobal[G, R any](f func(global *G, cx *App) R) R {
	return a.ReadR(func(cx *App) R { return f(cx.Global[G](), cx) })
}

// UpdateGlobal updates a global value.
func (a *TestApp) UpdateGlobal[G any](f func(global *G, cx *App)) {
	a.Update(func(cx *App) { UpdateGlobal(cx, f) })
}

// UpdateGlobalR is UpdateGlobal for a callback that returns a value.
func (a *TestApp) UpdateGlobalR[G, R any](f func(global *G, cx *App) R) R {
	return a.UpdateR(func(cx *App) R { return UpdateGlobalR(cx, f) })
}

// WriteToClipboard writes to the simulated clipboard.
func (a *TestApp) WriteToClipboard(item ClipboardItem) {
	a.platform.WriteToClipboard(item)
}

// ReadFromClipboard reads from the simulated clipboard.
func (a *TestApp) ReadFromClipboard() (ClipboardItem, bool) {
	return a.platform.ReadFromClipboard()
}

// OpenedURL returns the URL most recently opened with App.OpenURL.
func (a *TestApp) OpenedURL() (string, bool) {
	return a.platform.openedURL.Get()
}

// DidPromptForNewPath reports whether a file path prompt is pending.
func (a *TestApp) DidPromptForNewPath() bool {
	return a.platform.DidPromptForNewPath()
}

// SimulateNewPathSelection simulates answering a path selection dialog.
func (a *TestApp) SimulateNewPathSelection(selectPath func(path string) Option[string]) {
	a.platform.SimulateNewPathSelection(selectPath)
}

// HasPendingPrompt reports whether a prompt dialog is pending.
func (a *TestApp) HasPendingPrompt() bool {
	return a.platform.HasPendingPrompt()
}

// SimulatePromptAnswer simulates answering a prompt dialog.
func (a *TestApp) SimulatePromptAnswer(button string) {
	a.platform.SimulatePromptAnswer(button)
}

// Windows returns all open windows.
func (a *TestApp) Windows() []AnyWindowHandle {
	return a.ReadR(func(cx *App) []AnyWindowHandle { return cx.Windows() })
}

// Close ends the application: it releases its windows, drops the scheduler's remaining tasks
// and reports leaked handles, as dropping a TestApp does upstream.
func (a *TestApp) Close() {
	releaseAppWindows(a.app)
	a.dispatcher.DrainTasks()
	reportAppLeaks(a.app)
}

// TestAppWindow is a test window with inspection and simulation capabilities. Copies share the
// window (upstream's Clone).
type TestAppWindow[V any] struct {
	handle     WindowHandle[V]
	app        *AppCell
	platform   *TestPlatform
	bgExecutor BackgroundExecutor
}

// Handle returns the window handle.
func (w *TestAppWindow[V]) Handle() WindowHandle[V] {
	return w.handle
}

// Root returns the root view entity. The caller releases it.
func (w *TestAppWindow[V]) Root() Entity[V] {
	app := w.app.BorrowMut()
	defer app.Release()
	var root Entity[V]
	err := app.updateWindow(w.handle.AnyWindowHandle, func(rootView AnyView, _ *Window, _ *App) {
		view, ok := anyViewDowncast[V](rootView)
		if !ok {
			panic("root view type mismatch")
		}
		root = view
	})
	if err != nil {
		panic("window not found")
	}
	return root
}

// Update updates the root view and runs until parked afterwards.
func (w *TestAppWindow[V]) Update(f func(view *V, window *Window, cx *Context[V])) {
	updateTestAppWindow(w, func(view *V, window *Window, cx *Context[V]) struct{} {
		f(view, window, cx)
		return struct{}{}
	})
}

// UpdateR is Update for a callback that returns a value.
func (w *TestAppWindow[V]) UpdateR[R any](f func(view *V, window *Window, cx *Context[V]) R) R {
	return updateTestAppWindow(w, f)
}

func updateTestAppWindow[V, R any](w *TestAppWindow[V], f func(view *V, window *Window, cx *Context[V]) R) R {
	result := func() R {
		app := w.app.BorrowMut()
		defer app.Release()
		var result R
		err := app.updateWindow(w.handle.AnyWindowHandle, func(rootView AnyView, window *Window, cx *App) {
			view, ok := anyViewDowncast[V](rootView)
			if !ok {
				panic("root view type mismatch")
			}
			result = view.UpdateR(cx, func(view *V, cx *Context[V]) R { return f(view, window, cx) })
		})
		if err != nil {
			panic("window not found")
		}
		return result
	}()
	w.bgExecutor.RunUntilParked()
	return result
}

// Read reads the root view.
func (w *TestAppWindow[V]) Read(f func(view *V, cx *App)) {
	readTestAppWindow(w, func(view *V, cx *App) struct{} {
		f(view, cx)
		return struct{}{}
	})
}

// ReadR is Read for a callback that returns a value.
func (w *TestAppWindow[V]) ReadR[R any](f func(view *V, cx *App) R) R {
	return readTestAppWindow(w, f)
}

func readTestAppWindow[V, R any](w *TestAppWindow[V], f func(view *V, cx *App) R) R {
	app := w.app.Borrow()
	defer app.Release()
	window, ok := app.Get().windows.get(w.handle.id)
	if !ok || window == nil {
		panic("window or root view not found")
	}
	root, ok := window.root.Get()
	if !ok {
		panic("window or root view not found")
	}
	view, ok := anyViewDowncast[V](root)
	if !ok {
		panic("window or root view not found")
	}
	return f(view.Read(app.Get()), app.Get())
}

// Title returns the window title.
func (w *TestAppWindow[V]) Title() (string, bool) {
	app := w.app.Borrow()
	defer app.Release()
	title, err := AppContextReadWindow(app.Get(), w.handle, func(Entity[V], *App) Option[string] {
		return None[string]()
	})
	unwrapTestResult(err)
	return title.Get()
}

// SimulateKeystroke simulates a keystroke.
func (w *TestAppWindow[V]) SimulateKeystroke(keystroke string) {
	lock := w.app.Borrow()
	os := lock.Get().PlatformInfo().Os
	lock.Release()
	parsed, err := KeystrokeParseWithOs(keystroke, os)
	unwrapTestResult(err)
	func() {
		app := w.app.BorrowMut()
		defer app.Release()
		unwrapTestResult(app.updateWindow(w.handle.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
			window.DispatchKeystroke(parsed, cx)
		}))
	}()
	w.bgExecutor.RunUntilParked()
}

// SimulateKeystrokes simulates multiple space-separated keystrokes.
func (w *TestAppWindow[V]) SimulateKeystrokes(keystrokes string) {
	for _, keystroke := range splitSpace(keystrokes) {
		w.SimulateKeystroke(keystroke)
	}
}

func splitSpace(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// SimulateInput simulates typing text.
func (w *TestAppWindow[V]) SimulateInput(input string) {
	for _, char := range input {
		w.SimulateKeystroke(string(char))
	}
}

// SimulateMouseMove simulates a mouse move.
func (w *TestAppWindow[V]) SimulateMouseMove(position Point[Pixels]) {
	w.SimulateEvent(MouseMoveEvent{
		Position:      position,
		Modifiers:     Modifiers{},
		PressedButton: None[MouseButton](),
	})
}

// SimulateMouseDown simulates a mouse down event.
func (w *TestAppWindow[V]) SimulateMouseDown(position Point[Pixels], button MouseButton) {
	w.SimulateEvent(MouseDownEvent{
		Position:   position,
		Button:     button,
		Modifiers:  Modifiers{},
		ClickCount: 1,
		FirstMouse: false,
	})
}

// SimulateMouseUp simulates a mouse up event.
func (w *TestAppWindow[V]) SimulateMouseUp(position Point[Pixels], button MouseButton) {
	w.SimulateEvent(MouseUpEvent{
		Position:   position,
		Button:     button,
		Modifiers:  Modifiers{},
		ClickCount: 1,
	})
}

// SimulateClick simulates a click at the given position.
func (w *TestAppWindow[V]) SimulateClick(position Point[Pixels], button MouseButton) {
	w.SimulateMouseDown(position, button)
	w.SimulateMouseUp(position, button)
}

// SimulateScroll simulates a precise pointer scroll, as from a trackpad.
func (w *TestAppWindow[V]) SimulateScroll(position Point[Pixels], delta Point[Pixels]) {
	w.SimulateEvent(ScrollEvent{
		Position:   position,
		Delta:      ScrollDeltaPixels(delta),
		Modifiers:  Modifiers{},
		TouchPhase: TouchPhaseMoved,
		Source:     ScrollSourcePointer,
	})
}

// SimulateEvent simulates an input event.
func (w *TestAppWindow[V]) SimulateEvent(event InputEvent) {
	platformInput := event.ToPlatformInput()
	func() {
		app := w.app.BorrowMut()
		defer app.Release()
		unwrapTestResult(app.updateWindow(w.handle.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
			window.DispatchEvent(platformInput, cx)
		}))
	}()
	w.bgExecutor.RunUntilParked()
}

func (w *TestAppWindow[V]) testWindow() (*TestWindow, bool) {
	app := w.app.BorrowMut()
	defer app.Release()
	window, ok := app.windows.get(w.handle.id)
	if !ok || window == nil {
		return nil, false
	}
	tw, ok := window.platformWindow.(*TestWindow)
	if !ok {
		return nil, false
	}
	return tw.Clone(), true
}

// SimulateResize simulates resizing the window.
func (w *TestAppWindow[V]) SimulateResize(size Size[Pixels]) {
	app := w.app.BorrowMut()
	if window, ok := app.windows.get(w.handle.id); ok && window != nil {
		if tw, ok := window.platformWindow.(*TestWindow); ok {
			tw.SimulateResize(size)
		}
	}
	app.Release()
	w.bgExecutor.RunUntilParked()
}

// SimulateScaleFactorChange simulates the window moving to a display with a different scale
// factor.
func (w *TestAppWindow[V]) SimulateScaleFactorChange(scaleFactor float32) {
	if tw, ok := w.testWindow(); ok {
		tw.SimulateScaleFactorChange(scaleFactor)
	}
	w.bgExecutor.RunUntilParked()
}

// Draw forces a redraw of the window.
func (w *TestAppWindow[V]) Draw() {
	app := w.app.BorrowMut()
	defer app.Release()
	unwrapTestResult(app.updateWindow(w.handle.AnyWindowHandle, func(_ AnyView, window *Window, cx *App) {
		window.Draw(cx).Clear(cx)
	}))
}
