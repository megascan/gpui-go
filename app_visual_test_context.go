package gpui

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// VisualTestAppContext is a test context that uses real platform rendering instead of mocked
// rendering, for visual tests that capture actual screenshots, with deterministic task
// scheduling through a TestDispatcher. Windows it creates are positioned off-screen. Copies
// share the app (upstream's Clone); Close ends it.
type VisualTestAppContext struct {
	app                *AppCell
	bgExecutor         BackgroundExecutor
	foregroundExecutor ForegroundExecutor
	dispatcher         *TestDispatcher
	platform           Platform
	textSystem         *TextSystem
}

var _ AppContext = (*VisualTestAppContext)(nil)

// NewVisualTestAppContext creates a new VisualTestAppContext with real platform rendering but
// deterministic task scheduling. It uses a no-op asset source, so SVG icons won't render; use
// VisualTestAppContextWithAssetSource to provide real assets.
func NewVisualTestAppContext(platform Platform) *VisualTestAppContext {
	return VisualTestAppContextWithAssetSource(platform, NoAssets{})
}

// VisualTestAppContextWithAssetSource creates a new VisualTestAppContext with a custom asset
// source.
func VisualTestAppContextWithAssetSource(platform Platform, assetSource AssetSource) *VisualTestAppContext {
	seed := seedFromEnv()

	visualPlatform := NewVisualTestPlatform(platform, seed)

	dispatcher := visualPlatform.Dispatcher()
	backgroundExecutor := visualPlatform.BackgroundExecutor()
	foregroundExecutor := visualPlatform.ForegroundExecutor()

	textSystem := NewTextSystem(visualPlatform.TextSystem())

	httpClient := fakeHTTPClientWith404Response()

	app := newApp(visualPlatform, assetSource, httpClient)
	lock := app.BorrowMut()
	lock.mode = gpuiModeTest()
	lock.Release()

	return &VisualTestAppContext{
		app:                app,
		bgExecutor:         backgroundExecutor,
		foregroundExecutor: foregroundExecutor,
		dispatcher:         dispatcher,
		platform:           visualPlatform,
		textSystem:         textSystem,
	}
}

// OpenOffscreenWindow opens a window positioned at (-10000, -10000), off every display but
// still fully rendered by the compositor.
func (cx *VisualTestAppContext) OpenOffscreenWindow[V any](size Size[Pixels], buildRoot func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	bounds := Bounds[Pixels]{
		Origin: NewPoint(Px(-10000.0), Px(-10000.0)),
		Size:   size,
	}

	lock := cx.app.BorrowMut()
	defer lock.Release()
	options := DefaultWindowOptions()
	options.WindowBounds = Some(WindowBoundsWindowed(bounds))
	options.Focus = false
	options.Show = true
	return lock.OpenWindow(options, buildRoot)
}

// OpenOffscreenWindowDefault opens an off-screen window with the default size (1280x800).
func (cx *VisualTestAppContext) OpenOffscreenWindowDefault[V any](buildRoot func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	return cx.OpenOffscreenWindow(NewSize(Px(1280.0), Px(800.0)), buildRoot)
}

// IsScreenCaptureSupported reports whether screen capture is supported on this platform.
func (cx *VisualTestAppContext) IsScreenCaptureSupported() bool {
	return cx.platform.IsScreenCaptureSupported()
}

// TextSystem returns the text system used by this context.
func (cx *VisualTestAppContext) TextSystem() *TextSystem {
	return cx.textSystem
}

// Executor returns the background executor.
func (cx *VisualTestAppContext) Executor() BackgroundExecutor {
	return cx.bgExecutor
}

// ForegroundExecutor returns the foreground executor.
func (cx *VisualTestAppContext) ForegroundExecutor() ForegroundExecutor {
	return cx.foregroundExecutor
}

// RunUntilParked runs all pending foreground and background tasks until there's nothing left
// to do.
func (cx *VisualTestAppContext) RunUntilParked() {
	cx.dispatcher.RunUntilParked()
}

// AdvanceClock advances the simulated clock by the given duration and processes the tasks that
// become ready.
func (cx *VisualTestAppContext) AdvanceClock(duration time.Duration) {
	cx.dispatcher.AdvanceClock(duration)
}

// Update updates the app state.
func (cx *VisualTestAppContext) Update(f func(cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	f(app.App)
}

// UpdateR is Update for a callback that returns a value.
func (cx *VisualTestAppContext) UpdateR[R any](f func(cx *App) R) R {
	app := cx.app.BorrowMut()
	defer app.Release()
	return f(app.App)
}

// Read reads from the app state.
func (cx *VisualTestAppContext) Read(f func(cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	f(app.Get())
}

// ReadR is Read for a callback that returns a value.
func (cx *VisualTestAppContext) ReadR[R any](f func(cx *App) R) R {
	app := cx.app.Borrow()
	defer app.Release()
	return f(app.Get())
}

// UpdateWindow updates a window.
func (cx *VisualTestAppContext) UpdateWindow(window AnyWindowHandle, f func(root AnyView, window *Window, cx *App)) error {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateWindow(window, f)
}

// Spawn spawns a task on the foreground executor.
func (cx *VisualTestAppContext) Spawn[R any](f func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.foregroundExecutor.Spawn(f)
}

// HasGlobal reports whether a global of type G exists.
func (cx *VisualTestAppContext) HasGlobal[G any]() bool {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().HasGlobal[G]()
}

// ReadGlobal reads a global value.
func (cx *VisualTestAppContext) ReadGlobal[G, R any](f func(global *G, cx *App) R) R {
	app := cx.app.Borrow()
	defer app.Release()
	return f(app.Get().Global[G](), app.Get())
}

// SetGlobal sets a global value.
func (cx *VisualTestAppContext) SetGlobal[G any](global G) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.SetGlobal(global)
}

// UpdateGlobal updates a global value.
func (cx *VisualTestAppContext) UpdateGlobal[G any](f func(global *G, cx *App)) {
	cx.UpdateGlobalR[G](func(global *G, cx *App) struct{} {
		f(global, cx)
		return struct{}{}
	})
}

// UpdateGlobalR is UpdateGlobal for a callback that returns a value.
func (cx *VisualTestAppContext) UpdateGlobalR[G, R any](f func(global *G, cx *App) R) R {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateR(func(cx *App) R {
		global := cx.leaseGlobal[G]()
		result := f(global.get(), cx)
		cx.endGlobalLease(global)
		return result
	})
}

// SimulateKeystrokes simulates a whitespace-separated sequence of keystrokes on the window,
// such as "cmd-p escape".
func (cx *VisualTestAppContext) SimulateKeystrokes(window AnyWindowHandle, keystrokes string) {
	os := cx.ReadR(func(cx *App) OperatingSystem { return cx.PlatformInfo().Os })
	for _, keystrokeText := range strings.Fields(keystrokes) {
		keystroke, err := KeystrokeParseWithOs(keystrokeText, os)
		if err != nil {
			panic("Invalid keystroke: " + keystrokeText)
		}
		cx.DispatchKeystroke(window, keystroke)
	}
	cx.RunUntilParked()
}

// DispatchKeystroke dispatches a single keystroke to a window.
func (cx *VisualTestAppContext) DispatchKeystroke(window AnyWindowHandle, keystroke Keystroke) {
	_ = cx.UpdateWindow(window, func(_ AnyView, window *Window, cx *App) {
		window.DispatchKeystroke(keystroke, cx)
	})
}

// SimulateInput simulates typing text input on the window.
func (cx *VisualTestAppContext) SimulateInput(window AnyWindowHandle, input string) {
	for _, char := range input {
		key := string(char)
		keystroke := Keystroke{
			Modifiers: Modifiers{},
			Key:       key,
			KeyChar:   Some(key),
		}
		cx.DispatchKeystroke(window, keystroke)
	}
	cx.RunUntilParked()
}

// SimulateMouseMove simulates a mouse move event.
func (cx *VisualTestAppContext) SimulateMouseMove(window AnyWindowHandle, position Point[Pixels], button Option[MouseButton], modifiers Modifiers) {
	cx.SimulateEvent(window, MouseMoveEvent{
		Position:      position,
		Modifiers:     modifiers,
		PressedButton: button,
	})
}

// SimulateMouseDown simulates a mouse down event.
func (cx *VisualTestAppContext) SimulateMouseDown(window AnyWindowHandle, position Point[Pixels], button MouseButton, modifiers Modifiers) {
	cx.SimulateEvent(window, MouseDownEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     button,
		ClickCount: 1,
		FirstMouse: false,
	})
}

// SimulateMouseUp simulates a mouse up event.
func (cx *VisualTestAppContext) SimulateMouseUp(window AnyWindowHandle, position Point[Pixels], button MouseButton, modifiers Modifiers) {
	cx.SimulateEvent(window, MouseUpEvent{
		Position:   position,
		Modifiers:  modifiers,
		Button:     button,
		ClickCount: 1,
	})
}

// SimulateClick simulates a click, a mouse down followed by a mouse up.
func (cx *VisualTestAppContext) SimulateClick(window AnyWindowHandle, position Point[Pixels], modifiers Modifiers) {
	cx.SimulateMouseDown(window, position, MouseButtonLeft, modifiers)
	cx.SimulateMouseUp(window, position, MouseButtonLeft, modifiers)
}

// SimulateEvent simulates an input event on the window.
func (cx *VisualTestAppContext) SimulateEvent(window AnyWindowHandle, event InputEvent) {
	_ = cx.UpdateWindow(window, func(_ AnyView, window *Window, cx *App) {
		window.DispatchEvent(event.ToPlatformInput(), cx)
	})
	cx.RunUntilParked()
}

// DispatchAction dispatches an action to the window.
func (cx *VisualTestAppContext) DispatchAction(window AnyWindowHandle, action Action) {
	_ = cx.UpdateWindow(window, func(_ AnyView, window *Window, cx *App) {
		window.DispatchAction(action.BoxedClone(), cx)
	})
	cx.RunUntilParked()
}

// WriteToClipboard writes to the clipboard.
func (cx *VisualTestAppContext) WriteToClipboard(item ClipboardItem) {
	cx.platform.WriteToClipboard(item)
}

// ReadFromClipboard reads from the clipboard.
func (cx *VisualTestAppContext) ReadFromClipboard() (ClipboardItem, bool) {
	return cx.platform.ReadFromClipboard()
}

// WaitFor waits for predicate to hold for the entity, failing after timeout of real time.
func (cx *VisualTestAppContext) WaitFor[T any](s scheduler.Awaiter, entity Entity[T], predicate func(entity *T) bool, timeout time.Duration) error {
	start := time.Now()
	for {
		satisfied := func() bool {
			app := cx.app.Borrow()
			defer app.Release()
			return predicate(entity.Read(app.Get()))
		}()
		if satisfied {
			return nil
		}

		if time.Since(start) > timeout {
			return errors.New("Timed out waiting for condition")
		}

		cx.RunUntilParked()
		_, _ = cx.bgExecutor.Timer(10 * time.Millisecond).Await(s)
	}
}

// CaptureScreenshot captures a screenshot of the window by rendering its scene to a texture and
// reading the pixels back, which does not require the window to be visible.
func (cx *VisualTestAppContext) CaptureScreenshot(window AnyWindowHandle) (*image.RGBA, error) {
	var img *image.RGBA
	var renderErr error
	if err := cx.UpdateWindow(window, func(_ AnyView, window *Window, _ *App) {
		img, renderErr = window.RenderToImage()
	}); err != nil {
		return nil, err
	}
	return img, renderErr
}

// WaitForAnimations waits a couple of frames for animations to complete.
func (cx *VisualTestAppContext) WaitForAnimations(s scheduler.Awaiter) {
	_, _ = cx.bgExecutor.Timer(32 * time.Millisecond).Await(s)
	cx.RunUntilParked()
}

// Close releases the app's windows, drops the remaining tasks and reports leaked handles, as
// dropping the last copy of the context does upstream.
func (cx *VisualTestAppContext) Close() {
	releaseAppWindows(cx.app)
	cx.dispatcher.DrainTasks()
	reportAppLeaks(cx.app)
}

func (cx *VisualTestAppContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.newEntity(entityType, build)
}

func (cx *VisualTestAppContext) reserveEntity(entityType reflect.Type) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.reserveEntity(entityType)
}

func (cx *VisualTestAppContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.insertEntity(slot, build)
}

func (cx *VisualTestAppContext) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.updateEntity(handle, update)
}

func (cx *VisualTestAppContext) asMut(handle AnyEntity) *App {
	panic("Cannot use as_mut with a visual test app context. Try calling update() first")
}

func (cx *VisualTestAppContext) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readEntity(handle, read)
}

func (cx *VisualTestAppContext) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateWindow(window, update)
}

func (cx *VisualTestAppContext) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.withWindow(entityID, f)
}

func (cx *VisualTestAppContext) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().readWindow(window, read)
}

func (cx *VisualTestAppContext) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *VisualTestAppContext) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readGlobal(globalType, read)
}

// New creates a new entity.
func (cx *VisualTestAppContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later.
func (cx *VisualTestAppContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot reserved with ReserveEntity.
func (cx *VisualTestAppContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *VisualTestAppContext) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on the background executor.
func (cx *VisualTestAppContext) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.bgExecutor.Spawn(future)
}
