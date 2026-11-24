package gpui

import (
	"image"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

// HeadlessAppContext is a cross-platform headless app context for tests that need real text
// shaping. It uses TestPlatform for deterministic scheduling and accepts a pluggable
// PlatformTextSystem, so tests get real glyph measurements; an optional renderer factory
// enables GPU rendering and CaptureScreenshot. Close ends it, as its Drop does upstream.
type HeadlessAppContext struct {
	app                *AppCell
	bgExecutor         BackgroundExecutor
	foregroundExecutor ForegroundExecutor
	dispatcher         *TestDispatcher
	textSystem         *TextSystem
	closed             bool
}

var _ AppContext = (*HeadlessAppContext)(nil)

// NewHeadlessAppContext creates a new headless app context with the given text system.
func NewHeadlessAppContext(platformTextSystem PlatformTextSystem) *HeadlessAppContext {
	return HeadlessAppContextWithPlatform(platformTextSystem, NoAssets{}, func() (PlatformHeadlessRenderer, error) { return nil, nil })
}

// HeadlessAppContextWithAssetSource creates a new headless app context with a custom text
// system and asset source.
func HeadlessAppContextWithAssetSource(platformTextSystem PlatformTextSystem, assetSource AssetSource) *HeadlessAppContext {
	return HeadlessAppContextWithPlatform(platformTextSystem, assetSource, func() (PlatformHeadlessRenderer, error) { return nil, nil })
}

// HeadlessAppContextWithPlatform creates a new headless app context with the given text
// system, asset source and renderer factory; a factory returning a nil renderer leaves
// screenshots unsupported.
func HeadlessAppContextWithPlatform(platformTextSystem PlatformTextSystem, assetSource AssetSource, rendererFactory func() (PlatformHeadlessRenderer, error)) *HeadlessAppContext {
	seed := seedFromEnv()

	dispatcher := NewTestDispatcher(seed)
	arcDispatcher := dispatcher.Clone()
	backgroundExecutor := NewBackgroundExecutor(arcDispatcher)
	foregroundExecutor := NewForegroundExecutor(arcDispatcher)

	platform := TestPlatformWithPlatform(backgroundExecutor, foregroundExecutor, platformTextSystem, rendererFactory)

	textSystem := NewTextSystem(platformTextSystem)
	httpClient := fakeHTTPClientWith404Response()
	app := newApp(platform, assetSource, httpClient)
	lock := app.BorrowMut()
	lock.mode = gpuiModeTest()
	lock.Release()

	return &HeadlessAppContext{
		app:                app,
		bgExecutor:         backgroundExecutor,
		foregroundExecutor: foregroundExecutor,
		dispatcher:         dispatcher,
		textSystem:         textSystem,
	}
}

func seedFromEnv() uint64 {
	if v, ok := os.LookupEnv("SEED"); ok {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

// OpenWindow opens a window for headless rendering.
func (cx *HeadlessAppContext) OpenWindow[V any](size Size[Pixels], buildRoot func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error) {
	bounds := Bounds[Pixels]{
		Origin: NewPoint(Px(0.0), Px(0.0)),
		Size:   size,
	}

	lock := cx.app.BorrowMut()
	defer lock.Release()
	options := DefaultWindowOptions()
	options.WindowBounds = Some(WindowBoundsWindowed(bounds))
	options.Focus = false
	options.Show = false
	return lock.OpenWindow(options, buildRoot)
}

// RunUntilParked runs all pending tasks until parked.
func (cx *HeadlessAppContext) RunUntilParked() {
	cx.dispatcher.RunUntilParked()
}

// SimulateWindowScaleFactorChange simulates the window moving to a display with a different
// scale factor.
func (cx *HeadlessAppContext) SimulateWindowScaleFactorChange(window AnyWindowHandle, scaleFactor float32) {
	testWindow(cx.app, window).SimulateScaleFactorChange(scaleFactor)
}

// AdvanceClock advances the simulated clock.
func (cx *HeadlessAppContext) AdvanceClock(duration time.Duration) {
	cx.dispatcher.AdvanceClock(duration)
}

// AllowParking enables parking mode, allowing blocking on real I/O such as async asset loading.
func (cx *HeadlessAppContext) AllowParking() {
	cx.dispatcher.AllowParking()
}

// ForbidParking disables parking mode, returning to deterministic test execution.
func (cx *HeadlessAppContext) ForbidParking() {
	cx.dispatcher.ForbidParking()
}

// Update updates app state.
func (cx *HeadlessAppContext) Update(f func(cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	f(app.App)
}

// UpdateR is Update for a callback that returns a value.
func (cx *HeadlessAppContext) UpdateR[R any](f func(cx *App) R) R {
	app := cx.app.BorrowMut()
	defer app.Release()
	return f(app.App)
}

// UpdateWindow updates a window.
func (cx *HeadlessAppContext) UpdateWindow(window AnyWindowHandle, f func(root AnyView, window *Window, cx *App)) error {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.updateWindow(window, f)
}

// CaptureScreenshot captures a screenshot from a window. It requires a context created with a
// renderer factory that returns a renderer.
func (cx *HeadlessAppContext) CaptureScreenshot(window AnyWindowHandle) (*image.RGBA, error) {
	app := cx.app.BorrowMut()
	defer app.Release()
	var img *image.RGBA
	var renderErr error
	if err := app.updateWindow(window, func(_ AnyView, window *Window, _ *App) {
		img, renderErr = window.RenderToImage()
	}); err != nil {
		return nil, err
	}
	return img, renderErr
}

// TextSystem returns the text system.
func (cx *HeadlessAppContext) TextSystem() *TextSystem {
	return cx.textSystem
}

// BackgroundExecutor returns the background executor.
func (cx *HeadlessAppContext) BackgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

// ForegroundExecutor returns the foreground executor.
func (cx *HeadlessAppContext) ForegroundExecutor() ForegroundExecutor {
	return cx.foregroundExecutor
}

// Close shuts the app down so windows are closed and entity handles are released, drops the
// remaining tasks and reports leaked handles. It is idempotent.
func (cx *HeadlessAppContext) Close() {
	if cx.closed {
		return
	}
	cx.closed = true
	lock := cx.app.BorrowMut()
	lock.Shutdown()
	lock.Release()
	cx.dispatcher.DrainTasks()
	reportAppLeaks(cx.app)
}

func (cx *HeadlessAppContext) newEntity(entityType reflect.Type, build func(slot AnyEntity, cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.newEntity(entityType, build)
}

func (cx *HeadlessAppContext) reserveEntity(entityType reflect.Type) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.reserveEntity(entityType)
}

func (cx *HeadlessAppContext) insertEntity(slot AnyEntity, build func(cx *App) any) AnyEntity {
	app := cx.app.BorrowMut()
	defer app.Release()
	return app.insertEntity(slot, build)
}

func (cx *HeadlessAppContext) updateEntity(handle AnyEntity, update func(state any, cx *App)) {
	app := cx.app.BorrowMut()
	defer app.Release()
	app.updateEntity(handle, update)
}

func (cx *HeadlessAppContext) asMut(handle AnyEntity) *App {
	panic("Cannot use as_mut with HeadlessAppContext. Call update() instead.")
}

func (cx *HeadlessAppContext) readEntity(handle AnyEntity, read func(state any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readEntity(handle, read)
}

func (cx *HeadlessAppContext) updateWindow(window AnyWindowHandle, update func(root AnyView, window *Window, cx *App)) error {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.updateWindow(window, update)
}

func (cx *HeadlessAppContext) withWindow(entityID EntityID, f func(window *Window, cx *App)) bool {
	lock := cx.app.BorrowMut()
	defer lock.Release()
	return lock.withWindow(entityID, f)
}

func (cx *HeadlessAppContext) readWindow(window AnyWindowHandle, read func(root AnyView, cx *App)) error {
	app := cx.app.Borrow()
	defer app.Release()
	return app.Get().readWindow(window, read)
}

func (cx *HeadlessAppContext) backgroundExecutor() BackgroundExecutor {
	return cx.bgExecutor
}

func (cx *HeadlessAppContext) readGlobal(globalType reflect.Type, read func(global any, cx *App)) {
	app := cx.app.Borrow()
	defer app.Release()
	app.Get().readGlobal(globalType, read)
}

// New creates a new entity.
func (cx *HeadlessAppContext) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextNew(cx, buildEntity)
}

// ReserveEntity reserves a slot for an entity to be inserted later.
func (cx *HeadlessAppContext) ReserveEntity[T any]() Reservation[T] {
	return AppContextReserveEntity[T](cx)
}

// InsertEntity inserts an entity into a slot reserved with ReserveEntity.
func (cx *HeadlessAppContext) InsertEntity[T any](reservation Reservation[T], buildEntity func(cx *Context[T]) *T) Entity[T] {
	return AppContextInsertEntity(cx, reservation, buildEntity)
}

// ReadWindow reads the root view of a window.
func (cx *HeadlessAppContext) ReadWindow[V, R any](window WindowHandle[V], read func(root Entity[V], cx *App) R) (R, error) {
	return AppContextReadWindow(cx, window, read)
}

// BackgroundSpawn spawns a future on the background executor.
func (cx *HeadlessAppContext) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R] {
	return cx.bgExecutor.Spawn(future)
}

// ReadGlobal reads the global of type G.
func (cx *HeadlessAppContext) ReadGlobal[G, R any](callback func(global *G, cx *App) R) R {
	return AppContextReadGlobal(cx, callback)
}
