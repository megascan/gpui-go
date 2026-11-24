# gpui-go API fact sheet (verified)

Single source of truth for the user guide and the agent skill. Every signature is copied from the code at the cited `file:line` (repo root, branch `feat/docs`, 2026-11-24). Module `github.com/megascan/gpui-go`; root package `gpui`, platform picker `github.com/megascan/gpui-go/platform`.

Staleness warning: `docs/port/DESIGN.md` 6.x and `docs/api-contracts.md` signatures are abbreviated and partly stale (TECH_DEBT.md `M0-EXIT-OPEN`). Examples: DESIGN says `AppContextNew` returns `(Entity[T], error)`, code returns `Entity[T]` (gpui.go:58); DESIGN puts `ID`/`OnClick` on `Interactive`, code puts `ID` on `InteractiveID` and `OnClick` on `StatefulInteractive`. `docs/contexts.md` says async-context entity calls return `error`; `AsyncApp.Update` returns nothing and panics if the app is gone (app_async_context.go:188). Trust this file / the code, not those.

## 1. Install, build, first window

Toolchain and module
- `go.mod` go line `1.27.3` (TECH_DEBT `GO-DIRECTIVE`); generic methods on concrete types need Go 1.27 (RESEARCH-deps.md "Go language").
- No git tags exist (`git tag` empty) and the port lives on branch `feat/port` (draft PR to `dev`). There is no published version to `go get @vX`. Verified consumption path: a `replace` to a local checkout. Probe module (built OK on Linux):
```
module example.com/probe

go 1.27.3

require github.com/megascan/gpui-go v0.0.0

replace github.com/megascan/gpui-go => /path/to/gpui-go
```
  plus the repo's `go.sum` copied in; `go build` of a copy of examples/hello_world/main.go succeeded.
- Pinned deps (AGENT.md "Stack"): purego v0.11.3, oliverbestmann/webgpu v1.38.1 (wgpu-native v29), go-text/typesetting v0.3.5, xo/okage v0.20.1, jezek/xgb v1.3.1, jezek/xgbutil, godbus/dbus/v5 v5.2.4, x/image v0.48.0, x/sys v0.50.0, gen2brain/avif v0.6.0.

Per OS
- Linux: cgo REQUIRED. `render/wgpu` (`//go:build linux || js`, render/wgpu/gpui_wgpu.go:1) links static wgpu-native through `github.com/oliverbestmann/webgpu/wgpu` (cgo; libs pulled per OS via `libs-linux` module, wgpu/libs_linux.go). Verified: `CGO_ENABLED=0 go build` of hello_world fails with `imports github.com/oliverbestmann/webgpu/wgpu/internal/needscgo: build constraints exclude all Go files`. Needs a C toolchain.
  - Runtime shared libs loaded with purego (dlopen), from code grep: `libwayland-client.so.0`, `libwayland-cursor.so.0`, `libxkbcommon.so.0`, `libxkbcommon-x11.so.0`, `libxcb.so.1` (X11 surface only, DESIGN 8.1), `libpipewire-0.3.so.0` (screen capture only), `libc.so.6`; plus a Vulkan (or GLES) driver for wgpu.
  - The binding's own README suggests for its examples (not a gpui requirement list): `apt install -y libegl1-mesa-dev mesa-vulkan-drivers libasound2-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libxkbcommon-dev libwayland-dev`.
- macOS: purego/objc + Metal; no cgo in gpui (AGENT.md "Build": cgo only in render/wgpu and internal/cmd/wlspike). Shaders compiled at start with `newLibraryWithSource:` (DESIGN 8.3).
- Windows: x/sys/windows + hand COM, D3D11/DirectComposition/DirectWrite; HLSL compiled at start with `D3DCompile` (d3dcompiler_47) (DESIGN 8.4, D-WINSHADERRT). No cgo.
- Web: `GOOS=js GOARCH=wasm`; serve `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/` (OWNER-CHECKS.md:74).
- iOS: `GOOS=ios`, cgo, Xcode wrapper `examples/ios_ios/build-simulator.sh`; status in section 12.

Build tags (only these two exist)
- `gpui_release`: `gpuiutil.DebugAssertions = false` (gpuiutil/debug_assertions_release.go); default build has debug assertions on (gpuiutil/debug_assertions.go). E.g. `Hover` set twice panics only with assertions on (elements_div.go:519-521).
- `gpui_profiler`: profiler on (profiler_on.go:1, `profilerFeatureEnabled = true`); off by default (profiler_off.go). Replaces cargo feature `profiler` (D-FEATURE).

Env vars read by the code
- `ZED_HEADLESS` (any value) forces the headless Linux platform (platform/linux/linux.go:21, platform_cfg_freedesktop.go:14).
- `WAYLAND_DISPLAY`, `DISPLAY`, `XDG_RUNTIME_DIR`, `XDG_CURRENT_DESKTOP` (platform_cfg_freedesktop.go:42-50); `WAYLAND_SOCKET` (platform/linux/wayland/client.go:107); `XCURSOR_THEME`/`XCURSOR_SIZE`/`XCURSOR_PATH`; `GPUI_X11_SCALE_FACTOR` (platform/linux/x11/client.go:2260).
- `ZED_DEVICE_ID` (render/wgpu/wgpu_context.go:286) GPU pick; `GPUI_DISABLE_DIRECT_COMPOSITION` (platform/windows/platform.go:248).
- Tests: `GPUI_LEAK_DETECTION`, `LEAK_BACKTRACE`, `SEED`, `ITERATIONS`, `PROPTEST_CASES`, `GPUI_TEST_DISPLAY=1` (display tests only), `TRACK_THREAD_BORROWS` (D-BORROWLOC).

Minimal window: examples/hello_world/main.go, verbatim
```go
package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the hello_world example view.
type HelloWorld struct {
	text gpui.SharedString
}

func swatch(bg, border gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().
		Size8().
		Bg(bg).
		Border1().
		BorderDashed().
		RoundedMd().
		BorderColor(border)
}

// Render renders the view.
func (h *HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		Border1().
		BorderColor(gpui.Rgb(0x0000ff)).
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText(fmt.Sprintf("Hello, %s!", h.text)).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(swatch(gpui.Red(), gpui.White())).
				Child(swatch(gpui.Green(), gpui.White())).
				Child(swatch(gpui.Blue(), gpui.White())).
				Child(swatch(gpui.Yellow(), gpui.White())).
				Child(swatch(gpui.Black(), gpui.White()).RoundedMd()).
				Child(swatch(gpui.White(), gpui.Black())),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		if _, err := cx.OpenWindow(gpui.WindowOptions{
			WindowBounds: gpui.Some(gpui.WindowBoundsWindowed(bounds)),
		}, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(_ *gpui.Context[HelloWorld]) *HelloWorld {
				return &HelloWorld{text: "World"}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
```
WARNING (see 14.1): the `gpui.WindowOptions{...}` literal leaves `Show`/`Focus`/`IsResizable`/... false. Docs should teach the `DefaultWindowOptions()` form from examples/testing/main.go:232-241:
```go
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(300.), gpui.Px(200.)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(window *gpui.Window, cx *gpui.App) gpui.Entity[Counter] {
```
Fonts: on the web there are no system fonts; examples call `support.LoadFonts(cx)` which does `cx.TextSystem().AddFonts(fonts)` when `runtime.GOOS == "js"` (examples/internal/support). `examples/internal/support` is an internal package: not importable outside the repo.

## 2. Application, App, windows, bounds

Entry points (platform/platform.go)
- `func Application() *gpui.Application` :26, `func Headless() *gpui.Application` :31, `func CurrentPlatform(headless bool) Platform` :37 (panics on init failure), `func BackgroundExecutor() gpui.BackgroundExecutor` :21, `func CurrentHeadlessRenderer() (gpui.PlatformHeadlessRenderer, error)` :64.
- Package `platform` `init` calls `runtime.LockOSThread()` (platform/platform.go:13-15): importing it pins the main goroutine to the main OS thread. Linux only: `func Linux(allowedModes gpui.WindowingModes) *gpui.Application` (platform/platform_linux.go:16). Web only: `ApplicationWithWebBackend`, `SingleThreadedWeb`, `WebInit` (platform/platform_js.go:14,19,25).

`*gpui.Application` (app.go:164)
- `func ApplicationWithPlatform(platform Platform) *Application` :197
- `func (a *Application) Run(onFinishLaunching func(cx *App))` :252 (blocks on desktop; on js it returns and the app is retained, so web `main` ends with `select {}`, examples/web_hello_web/main.go:368)
- `func (a *Application) RunEmbedded(onFinishLaunching func(cx *App)) *ApplicationHandle` :271
- Builders returning `*Application`: `WithAssets(assetSource AssetSource)` :211, `WithRestartArguments([]string)` :220, `WithHTTPClient(HTTPClient)` :228, `WithQuitMode(QuitMode)` :236, `WithWindowing(WindowingRequest)` :244, `OnOpenURLs(func(urls []string))` :286, `OnReopen(func(cx *App))` :294.
- `type AssetSource interface { Load(path string) (data []byte, ok bool, err error); List(path string) ([]SharedString, error) }` (assets.go:10-15); file-backed implementation in examples/svg_example/main.go:19-39.

`*gpui.App` (app.go:997) common methods
- `func (cx *App) OpenWindow[V any](options WindowOptions, buildRootView func(window *Window, cx *App) Entity[V]) (WindowHandle[V], error)` :1655. The returned entity is moved into the window root (`window.root.Replace(anyViewFromEntity(rootView))`, :1674): do not release it.
- `Activate(ignoringOtherApps bool)` :1690, `Quit()` :1413, `Windows() []AnyWindowHandle` :1586, `ActiveWindow() (AnyWindowHandle, bool)` :1603, `Displays()` :1710, `PrimaryDisplay()` :1715, `Defer(f func(cx *App))` :2525, `OnWindowClosed(onClosed func(cx *App, id WindowID)) Subscription` :2844, `SetMenus([]Menu)` :2878, `OpenURL(string)` :2105, `WriteToClipboard(ClipboardItem)` :2056, `ReadFromClipboard() (ClipboardItem, bool)` :2008, `TextSystem() *TextSystem` :2535, `RefreshWindows()` :1462, `SetQuitMode(QuitMode)` :2222, `RequestWindowing(request WindowingRequest) *Task[struct{}]` :1419.
- Globals: `HasGlobal[G]() bool` :2572, `Global[G]() *G` :2579 (panics if unset), `TryGlobal[G]() (*G, bool)` :2588, `GlobalMut[G]() *G` :2598, `DefaultGlobal[G]() *G` :2610, `SetGlobal[G](global G)` :2622, `RemoveGlobal[G]() G` :2634, `ObserveGlobal[G](f func(cx *App)) Subscription` :2646; free funcs `Global[G any](cx *App) *G` (global.go:7), `SetGlobal[G any, C BorrowAppContext](cx C, global G)`, `UpdateGlobal[G any, C BorrowAppContext](cx C, f func(global *G, cx C))`, `UpdateGlobalR` (gpui.go:236-249). Any type can be a global (D-GLOBAL).

`WindowOptions` (platform.go:2651): fields `WindowBounds Option[WindowBounds]`, `Titlebar Option[TitlebarOptions]`, `Focus`, `Show`, `Kind WindowKind`, `IsMovable`, `AppOwnsTitlebarDrag`, `InactiveFrameInterval`/`LowPowerFrameInterval`/`ThermalFrameInterval Option[time.Duration]`, `IsResizable`, `IsMinimizable`, `DisplayID Option[DisplayID]`, `WindowBackground WindowBackgroundAppearance`, `AppID Option[string]`, `WindowMinSize Option[Size[Pixels]]`, `WindowDecorations Option[WindowDecorations]`, `Icon Option[*image.NRGBA]` (X11 only), `TabbingIdentifier Option[string]`, `UseDisplayP3 bool`.
- `func DefaultWindowOptions() WindowOptions` (platform.go:2782): `Titlebar: Some(TitlebarOptions{})`, `Focus: true`, `Show: true`, `Kind: WindowKindNormal{}`, `IsMovable: true`, `IsResizable: true`, `IsMinimizable: true`, frame intervals 33.333ms/33.333ms/16.667ms, `WindowBackground: WindowBackgroundAppearanceOpaque`.
- `TitlebarOptions{Title Option[SharedString]}` (platform.go:2808).
- `WindowBoundsWindowed(bounds Bounds[Pixels]) WindowBounds` :2751, `WindowBoundsMaximized` :2756, `WindowBoundsFullscreen` :2761, `WindowBoundsCentered(displayID Option[DisplayID], size Size[Pixels], cx *App) WindowBounds` :2777.
- `DefaultWindowSize = NewSize(Px(1536.), Px(1095.))` (window.go:24).

Bounds/geometry helpers (geometry.go)
- `NewPoint[T Number](x, y T) Point[T]` :94, `NewSize[T Number](width, height T) Size[T]` :301, `NewBounds[T Number](origin Point[T], size Size[T]) Bounds[T]` :500, `BoundsCentered(displayID Option[DisplayID], size Size[Pixels], cx *App) Bounds[Pixels]` :505, `BoundsMaximized(displayID Option[DisplayID], cx *App) Bounds[Pixels]` :521, `BoundsFromCorners` :536, `BoundsCenteredAt` :565, `EdgesAll[T Number](value T) Edges[T]` :833, `CornersAll` :1085.

`WindowHandle[V]` (window.go)
- `Root(cx AppContext) (Entity[V], error)` :2366 (new count; caller releases), `Update(cx AppContext, update func(view *V, window *Window, cx *Context[V])) error` :2385, `UpdateR[R]` :2394, `Read(cx *App) (*V, error)` :2413, `ReadWith[R]` :2431, `Entity(cx AppContext) (Entity[V], error)` :2437, `IsActive(cx *App) (active bool, ok bool)` :2443. `AnyWindowHandle`: `WindowID()` :2459, `Downcast[T]() (WindowHandle[T], bool)` :2470, `Update(cx AppContext, update func(root AnyView, window *Window, cx *App)) error` :2479.

`*Window` (not a context; passed beside `cx`): `Refresh()` window_paint.go:271, `RemoveWindow()` :279, `Bounds()` :679, `ViewportSize()` :741, `ScaleFactor()` :893, `RemSize()`/`SetRemSize` :904/:912, `SetWindowTitle(title string)` :842, `IsWindowActive()` :776, `MousePosition()` :1121, `Modifiers()` :1151, `OnNextFrame(callback func(window *Window, cx *App))` :564, `RequestAnimationFrame()` :572, `Defer(cx *App, f func(window *Window, cx *App))` :487, `Prompt(level PromptLevel, message string, detail Option[string], answers []PromptButton, cx *App) *scheduler.OneshotReceiver[int]` (window_dispatch.go:1220), `UseState[S any](cx *App, init func(window *Window, cx *Context[S]) *S) Entity[S]` (window_paint.go:1926), `UseKeyedState[S any](key ElementID, cx *App, init ...) Entity[S]` :1911 (both: "the caller releases the returned handle").

## 3. Entities, contexts, subscriptions, leaks

Handles (app_entity_map.go)
- `type Entity[T any] struct` :567. Methods: `EntityID()` :584, `Clone() Entity[T]` :589, `Release()` :594, `Downgrade() WeakEntity[T]` :599, `IntoAny() AnyEntity` :604, `Read(cx *App) *T` :609, `ReadWith[R any](cx AppContext, f func(entity *T, cx *App) R) R` :614, `Update(cx AppContext, update func(entity *T, cx *Context[T]))` :619, `UpdateR[R any](cx AppContext, update func(entity *T, cx *Context[T]) R) R` :627, `AsMut(cx AppContext) GpuiBorrow[T]` :632, `Write(cx AppContext, value T)` :637, `UpdateIn(cx VisualContext, update func(entity *T, window *Window, cx *Context[T])) error` :646, `UpdateInR[R]` :656; `FocusHandle(cx *App) FocusHandle` (window.go:737, panics unless `*T` implements `Focusable`, D-COND); `IntoAnyElement()` (view.go:130), `Cached(style StyleRefinement) *ViewElement` (view.go:139).
- `type WeakEntity[T any] struct` :743: `Upgrade() (Entity[T], bool)` :748, `Update(cx AppContext, ...) error` :758, `UpdateR` :768, `UpdateIn(cx AppContext, ...) error` :780, `UpdateInR` :790, `ReadWith[R](cx AppContext, read func(entity *T, cx *App) R) (R, error)` :810. Errors: `errEntityReleased = errors.New("entity released")`, `"entity has no current window"` :826-827.
- `AnyEntity`: `Downcast[T]() (Entity[T], bool)` :529, `Clone` :537, `Release` :557. `AnyWeakEntity.AssertReleased()` :708.
- Semantics (api-contracts.md 4; DESIGN 5.2): copying a handle value aliases one counted handle; `Clone()` makes a new count; `Release()` is idempotent per handle; use after release panics; zero `Entity` value allowed, `Release` on it is a no-op. `IntoAny()` returns the same counted handle (release one of the two). GC cleanup is only a safety net for entity and focus handles. Nested update/read of the same entity panics `"cannot %s %s while it is already being updated"` (app_entity_map.go:393).
- Entity state that owns handles implements `Drop(cx *App)`; the app calls it on release (interface `entityDropper`, app.go:2316-2318). Real example (examples/view_example/example_editor.go:70-78):
```go
// Drop releases the editor's handles, task and subscriptions.
func (e *Editor) Drop(_ *gpui.App) {
	e.blinkTask.Cancel()
	for _, s := range e.subscriptions {
		s.Unsubscribe()
	}
	e.focusHandle.Release()
	e.value.Release()
}
```

Creating (each context has delegates; plain value on `*App`/`*Context[T]`/test contexts):
- `func (cx *App) New[T any](buildEntity func(cx *Context[T]) *T) Entity[T]` (app.go:3312), `ReserveEntity[T]() Reservation[T]` :3317, `InsertEntity[T]` :3322. Same shape on `*AsyncApp` (app_async_context.go:140) and `*TestAppContext` (app_test_context.go:201).
- Free funcs over sealed `AppContext` (D-CTX, gpui.go): `AppContextNew[T any](cx AppContext, buildEntity func(cx *Context[T]) *T) Entity[T]` :58, `AppContextUpdateEntity[T, R any](cx AppContext, handle Entity[T], update func(entity *T, cx *Context[T]) R) R` :82, `AppContextReadEntity` :98, `AppContextUpdateWindow[R any](cx AppContext, window AnyWindowHandle, update func(root AnyView, window *Window, cx *App) R) (R, error)` :107, `AppContextWithWindow[R any](cx AppContext, entityID EntityID, f func(window *Window, cx *App) R) (R, bool)` :118, `AppContextReadWindow` :127, `AppContextBackgroundSpawn` :145, `AppContextReadGlobal` :150, `VisualContextUpdateWindowEntity` :179, `VisualContextNewWindowEntity[T any](cx VisualContext, buildEntity func(window *Window, cx *Context[T]) *T) (Entity[T], error)` :190, `VisualContextReplaceRootView` :202, `VisualContextFocus[V any](cx VisualContext, entity Entity[V]) error` :212.

`*Context[T]` (app_context.go:13) embeds `*App` (pass `cx.App` where `*App` is wanted).
- `EntityID()` :23, `Entity() Entity[T]` :28 ("The caller releases it."), `WeakEntity() WeakEntity[T]` :37, `Notify()` :138, `Emit[E any](event E)` :462 (not checked against an emitter type, D-EMIT).
- `Observe[W any](entity Entity[W], onNotify func(this *T, entity Entity[W], cx *Context[T])) Subscription` :42, `ObserveSelf` :50, `Subscribe[T2, E any](entity Entity[T2], onEvent func(this *T, entity Entity[T2], event *E, cx *Context[T])) Subscription` :59, `SubscribeSelf[E any](onEvent func(this *T, event *E, cx *Context[T])) Subscription` :67, `OnRelease` :76, `ObserveRelease` :87, `ObserveGlobal[G]` :103, `ObserveIn` :218, `SubscribeIn[Emitter, E any](emitter Entity[Emitter], window *Window, onEvent func(this *T, emitter Entity[Emitter], event *E, window *Window, cx *Context[T])) Subscription` :231, `ObserveWindowBounds` :267, `ObserveWindowActivation` :273, `ObserveKeystrokes` :305.
- `Listener[E any](listener func(this *T, event *E, window *Window, cx *Context[T])) func(event *E, window *Window, cx *App)` :152, `Processor[E, R any](f func(this *T, event E, window *Window, cx *Context[T]) R) func(event E, window *Window, cx *App) R` :165.
- `OnNextFrame(window *Window, f ...)` :195, `DeferIn` :205, `FocusView[W any](view Entity[W], window *Window)` :184, `FocusSelf(window *Window)` :446, `OnFocus`/`OnFocusIn`/`OnBlur`/`OnFocusLost`/`OnFocusOut` :347-391.
- `*App` forms: `Observe[W any](entity Entity[W], onNotify func(entity Entity[W], cx *App)) Subscription` (app.go:1491), `Subscribe[T, E any](entity Entity[T], onEvent func(entity Entity[T], event *E, cx *App)) Subscription` :1552, `ObserveNew[T]` :2677, `ObserveRelease[T]` :2692.
- Handles passed into Observe/Subscribe callbacks are borrowed for the call; `Clone()` to keep (api-contracts.md 4).

Observe/Subscribe/Emit example, examples/ownership_post/main.go:26-47 verbatim (`type Change struct{ increment int }` at :17-19):
```go
		counter := cx.New(func(_ *gpui.Context[Counter]) *Counter { return &Counter{count: 0} })
		subscriber := cx.New(func(cx *gpui.Context[Counter]) *Counter {
			cx.Subscribe(counter, func(subscriber *Counter, _ gpui.Entity[Counter], event *Change, _ *gpui.Context[Counter]) {
				subscriber.count += event.increment * 2
			}).Detach()

			return &Counter{
				count: counter.Read(cx.App).count * 2,
			}
		})

		counter.Update(cx, func(counter *Counter, cx *gpui.Context[Counter]) {
			counter.count += 2
			cx.Notify()
			cx.Emit(Change{increment: 2})
		})

		if got := subscriber.Read(cx).count; got != 4 {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", got, 4))
		}
		subscriber.Release()
		counter.Release()
```

`Subscription` (subscription.go:269): `Unsubscribe()` :307 (idempotent; cancels), `Detach()` :302 (keeps the callback until the subscribed entities are released), `NewSubscription(func()) Subscription` :274, `SubscriptionJoin` :314. Zero value is empty. No GC-driven unsubscribe (D-RAII).

Leak detection (app_entity_map.go:838-851, verbatim logic):
```go
var currentLeakDetectionMode = sync.OnceValue(func() leakDetectionMode {
	value, set := os.LookupEnv("GPUI_LEAK_DETECTION")
	switch {
	case value == "strict":
		return leakDetectionStrict
	case testing.Testing():
		if !set {
			return leakDetectionStrict
		}
		return leakDetectionOn
	case value != "" && value != "0":
		return leakDetectionOn
	}
	return leakDetectionOff
})
```
- Any test binary defaults to strict; `GPUI_LEAK_DETECTION=0` inside a test binary gives "on", not off. Outside tests it is off unless set to non-empty, non-`0`.
- Strict reports exist only for: entity handles "was collected without Release" (app_entity_map.go:464), subscriptions "Subscription was collected without Unsubscribe or Detach" (subscription.go:278), focus handles (window.go:514). Tasks are NOT reported (no `AddCleanup` on tasks; DESIGN 5.2:172 claims otherwise).
- `LEAK_BACKTRACE` (any non-empty) captures allocation sites (app_entity_map.go:855). Snapshot API: `(cx *App) LeakDetectorSnapshot() LeakDetectorSnapshot` app.go:1294, `AssertNoNewLeaks(snapshot *LeakDetectorSnapshot)` :1299.
- Strict reports are process-global: tests that check them must not run in parallel with leaking tests (api-contracts.md 4.1).

## 4. Views, Render, IntoElement, components

- `type Render[T any] interface { Render(window *Window, cx *Context[T]) IntoElement }` (element.go:56), implemented by `*T`.
- `type RenderOnce interface { Render(window *Window, cx *App) IntoElement }` (element.go:67), implemented by component values.
- `type IntoElement interface { IntoAnyElement() AnyElement }` (element.go:50). Implemented by `AnyElement`, `SharedString` (elements_text.go:139), `*Div`, `*Stateful[E]`, every element, `Entity[T]` (view.go:130; panics unless `*T` implements `Render[T]`, D-COND), `AnyView`.
- `type View interface` (view.go:220); `NewViewElement(view View) *ViewElement` (view.go:263); `AnyViewFromEntity[V any](value Entity[V]) AnyView` (view.go:25, "taking over the entity's handle").
- `Entity[T].IntoAnyElement()` "moving the handle into it" (view.go:128): `Child(entity)` consumes your handle; pass `entity.Clone()` to keep yours (examples/input/main.go:872 `Child(e.textInput.Clone())`).
- Component = `#[derive(IntoElement)]` replacement: `func NewComponent[C RenderOnce](component C) AnyElement` (view.go:245). Pattern (examples/text/main.go:84-90):
```go
// IntoAnyElement renders the specimen as a component.
func (s Specimen) IntoAnyElement() gpui.AnyElement {
	return gpui.NewComponent(s)
}

// Render renders the specimen.
func (s Specimen) Render(window *gpui.Window, cx *gpui.App) gpui.IntoElement {
```
- `type Empty struct{}` (element.go:556) is the empty element and also `Render[Empty]` (`func (Empty) Render(window *Window, cx *Context[Empty]) IntoElement`, element.go:62). `#[derive(Render)]` = hand-written empty `Render` (DESIGN 9).
- Custom elements implement `Element[RL, PP]` (`ElementID() (ElementID, bool)`, `SourceLocation() *SourceLocation`, `RequestLayout`, `Prepaint`, `Paint`) and wrap with `NewAnyElement[RL, PP any](element Element[RL, PP]) AnyElement` (element.go:500). `AnyElement` does not implement `Element` (D-INTOELEMENT).

## 5. Elements

Div and mixins (api-contracts.md 3)
- `func NewDiv() *Div` (elements_div.go:1124). `Div` embeds `Styled[*Div]`, `Interactive[*Div]`, `InteractiveID[*Div]`, `Parent[*Div]`, `Fluent[*Div]`.
- `func (i *InteractiveID[S]) ID[I ElementIDSource](id I) *Stateful[S]` (elements_div.go:452). After `ID`, the chain type is `*Stateful[*Div]` and gains `StatefulInteractive` (`OnClick`, `Active`, `Tooltip`, `OnDrag`, `OverflowScroll`, ...). `Stateful[E]` (elements_div_paint.go:1700) implements `IntoElement` only and has no `ID` (D-IDMIXIN).
- `ElementIDSource` (window.go:2533): `~string | ~int | ~int32 | ~uint | ~uint32 | ~uint64 | ElementID | *FocusHandle | *SourceLocation | [16]byte | [20]byte`. Helpers `ElementIDName(name SharedString) ElementID` :2573, `ElementIDNamedInteger(name SharedString, integer uint64) ElementID` :2590.
- Parent (element.go): `Child(child IntoElement) S` :90, `Children(children ...IntoElement) S` :96, `ChildText(text string) S` :106 (plain Go strings; `SharedString` goes through `Child`, D-STRING).
- Fluent (util.go): `When(condition bool, then func(this S) S) S` :28, `WhenElse` :38, `WhenSome[T any](option Option[T], then func(this S, value T) S) S` :48, `WhenNone` :58, `Map[U any](fn func(this S) U) U` :23.

Text
- `type SharedString string` (shared_string.go:10); `NewSharedString`, `NewSharedStringStatic` :13-18. `TextMacro(text SharedString) Text` (elements_text.go:58), `NewText(id ElementID, text SharedString) Text` :22, `NewStyledText(text SharedString) *StyledText` :163, `NewInteractiveText(id ElementID, text *StyledText) *InteractiveText` :752.

Img / Svg
- `NewImg(source ImageSource) *Img` (elements_img.go:230); sources `ImageSourceFromString` :123 (URL, or asset path), `ImageSourceFromPath(path string)` :136 (filesystem), `ImageSourceFromSharedURI` :117, `ImageSourceFromRenderImage` :141, `ImageSourceFromImage` :146, `ImageSourceFromFunc(load func(window *Window, cx *App) (*RenderImage, bool, error))` :151. Image style methods (`StyledImage[*Img]`) must be called before `ID` (D-STATEFULIMG).
- `NewSvg() *Svg` (elements_svg.go:30): `Path(path SharedString)` :46 (through the app's `AssetSource`), `ExternalPath` :52, `Data(data []byte)` :59, `WithTransformation(Transformation)` :74. Color via `TextColor`. SVG text/images not rendered (L-SVGTEXT).

Lists
- `NewUniformList[R IntoElement, I ElementIDSource](id I, itemCount int, f func(visibleRange Range, window *Window, cx *App) []R) *UniformList` (elements_uniform_list.go:15); `TrackUniformListScroll(handle UniformListScrollHandle)` :549 (D-UNIFORMTRACK), `NewUniformListScrollHandle()` :141. Example (examples/uniform_list/main.go:16-37):
```go
	return gpui.NewDiv().SizeFull().Bg(gpui.Rgb(0xffffff)).Child(
		gpui.NewUniformList(
			"entries",
			50,
			cx.Processor(func(_ *UniformListExample, visibleRange gpui.Range, _ *gpui.Window, _ *gpui.Context[UniformListExample]) []*gpui.Stateful[*gpui.Div] {
				var items []*gpui.Stateful[*gpui.Div]
				for ix := visibleRange.Start; ix < visibleRange.End; ix++ {
					item := ix + 1
					items = append(items,
						gpui.NewDiv().
							ID(ix).
							Px2().
							CursorPointer().
							OnClick(func(*gpui.ClickEvent, *gpui.Window, *gpui.App) {
								fmt.Printf("clicked Item %d\n", item)
							}).
							ChildText(fmt.Sprintf("Item %d", item)),
					)
				}
				return items
			}),
		).HFull(),
	)
```
- `NewListState(itemCount int, alignment ListAlignment, overdraw Pixels) ListState` (elements_list.go:447), `NewList(state ListState, renderItem func(ix int, window *Window, cx *App) AnyElement) *List` :16. `ListState` is a value sharing state across copies (D-LISTCELL); methods `MeasureAll()` :470, `Reset(int)` :489, `Splice(oldRange Range, count int)` :616, `ScrollToEnd()` :711, `ScrollToRevealItem(ix int)` :784, `SetScrollHandler` :666. Example: examples/list_example/main.go:24, :98-112 (`gpui.NewListState(itemCount, gpui.ListAlignmentBottom, gpui.Px(500.)).MeasureAll()`; render func returns `....IntoAnyElement()`).
- `Range{Start, End int}` replaces `Range<usize>` (D-RANGE).

Canvas, Anchored, Deferred
- `NewCanvas[T any](prepaint func(bounds Bounds[Pixels], window *Window, cx *App) T, paint func(bounds Bounds[Pixels], state T, window *Window, cx *App)) *Canvas[T]` (elements_canvas.go:5). Example examples/gradient/main.go:145-147.
- `NewAnchored() *Anchored` (elements_anchored.go:38): `Anchor(Anchor)` :51, `Position(Point[Pixels])` :58, `Attach` :71, `Offset` :78, `PositionMode(AnchoredPositionMode)` :86, `SnapToWindow()` :93, `SnapToWindowWithMargin` :99.
- `NewDeferred(child IntoElement) *Deferred` (elements_deferred.go:4), `WithPriority(int)` :20 / `Priority(int)` :59. Popover pattern examples/anchor/main.go:126-131:
```go
				return this.Child(gpui.NewDeferred(
					gpui.NewAnchored().
						Anchor(corner).
						Position(position).
						PositionMode(gpui.AnchoredPositionModeLocal).
						SnapToWindow().
```

Animation (D-ANIMEXT: free functions, not methods)
- `NewAnimation(duration time.Duration) Animation` (elements_animation.go:34); `Repeat()` :46, `RepeatSynced()` :53, `WithEasing(func(float32) float32)` :62, `WithMaxFps` :70, `WithSteps` :78.
- `WithAnimation[E IntoElement, I ElementIDSource](element E, id I, animation Animation, animator func(element E, delta float32) E) *AnimationElement[E]` :92; `WithAnimations` :100; `WithSpring[E IntoElement, I ElementIDSource, T, O any](...)` :109. Easings `Linear` :506, `Quadratic` :511, `EaseInOut` :517, `EaseOutQuint()` :527, `Bounce(easing)` :547, `PulsatingBetween(min, max float32)` :558. Example examples/animation/main.go:230-243:
```go
							gpui.WithAnimation(
								gpui.NewSvg().
									Size20().
									OverflowHidden().
									Path(arrowCircleSvg).
									TextColor(gpui.Black()),
								"image_circle",
								gpui.NewAnimation(2*time.Second).
									Repeat().
									WithEasing(gpui.Bounce(gpui.EaseInOut)),
								func(svg *gpui.Svg, delta float32) *gpui.Svg {
									return svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))
								},
							),
```
- Others: `NewContainerQuery` (elements_container_query.go:10), `NewImageCache` (elements_image_cache.go:12), `RetainAll[I](id I)` :270, `NewSurface` (elements_surface.go:30, draws only on darwin/ios, D-SURFACE).

## 6. Styling

- `type Styled[S Styler] struct` (styled.go:13); every method returns `S`, so `NewDiv().Flex().SizeFull()` stays `*Div` and after `ID` stays `*Stateful[*Div]`. 115 exported hand-written (styled.go) + 3099 generated (styled_gen.go, stylegen) methods.
- Layout: `Flex()` styled.go:29, `Block()` :23, `Grid()` :35, `Hidden()` :41, `FlexCol()` :130, `FlexRow()` :142, `Flex1()` :154, `FlexGrow(float32)` :192, `FlexShrink0()` :216, `FlexWrap()` :228, `ItemsCenter()` :258, `ItemsStart()` :246, `JustifyCenter()` :330, `JustifyBetween()` :337, `GridCols(uint16)` :647, `AspectRatio(float32)` :410.
- Size (styled_gen.go): `Size(length IntoLength)` :1660, `W(length IntoLength)` :8, `H(length IntoLength)` :834, `MinW` :3496, `MaxW` :6066, `SizeFull()` :2300, `WFull()` :584, `HFull()` :1410, scale shortcuts `Size8()` :1910 (Tailwind scale: `W128`, `H32`, ...).
- Spacing: `P(length IntoDefiniteLength)` :17378, `Px` :20102, `Py` :21010, `Pt` :18468, `Pl` :21918, `M(length IntoLength)` :11136, `Mx` :14808, `Gap(length IntoDefiniteLength)` :7718, `Gap2()` :7808 (and `Gap3`, `Gap4`, ...). Note: the padding method `Px` (styled) and the unit constructor `gpui.Px` share a name in different scopes.
- Position: `Relative()` :23553, `Absolute()` :23560, `Top(IntoLength)` :24668, `Left` :26320, `Inset0()` :23578, `OverflowHidden()` :27973; scroll via `StatefulInteractive.OverflowScroll/OverflowYScroll` (elements_div.go:929-943).
- Color/fill: `Bg(fill IntoFill)` styled.go:422; `TextColor(color IntoHsla)` :442; `Opacity(float32)` :641. `IntoFill` implemented by `FillColor`, `Hsla`, `Rgba`, `Background` (style.go:812-842); `IntoHsla` by `Rgba`, `Hsla` (color.go:96, :356).
- Border/radius/shadow: `Border1()` styled_gen.go:28191, `Border(length IntoAbsoluteLength)` :28167, `BorderColor(borderColor IntoHsla)` :28161, `Rounded(length IntoAbsoluteLength)` :10260, `RoundedMd()` :10308, `RoundedLg()` :10320, `RoundedFull()` :10368, `Shadow(shadows []BoxShadow)` :29378, `ShadowLg()` :29430 (all styled_gen.go); `BorderDashed()` styled.go:428.
- Text: `TextSize(size IntoAbsoluteLength)` styled.go:466, `TextXs/TextSm/TextBase/TextLg/TextXl/Text2xl/Text3xl` :472-508, `FontWeight(FontWeight)` :450, `FontFamily(string)` :612, `LineHeight(IntoDefiniteLength)` :635, `Italic()` :514, `Underline()` :526, `Truncate()` :116, `TextEllipsis()` :67, `LineClamp(int)` :123, `TextCenter()` :106, `WhitespaceNowrap()` :61.
- Cursor: `CursorPointer()` styled_gen.go:28008, `CursorMove()` :28022. Debug: `Debug()` styled.go:769.
- Units (geometry.go): `type Pixels float32` :1305, `func Px(pixels float32) Pixels` :1914, `type Rems float32` :1517 (construct with conversion `gpui.Rems(1.5)`, DESIGN 5.1), `func Relative(fraction float32) DefiniteLength` :1904, `func Auto() Length` :1919, `Phi()` :1909, `NewPercentage(value float32) Percentage` :1287, `RemsFromPixels` :1528. `IntoLength` (:1576) implemented by `Pixels`, `Rems`, `AbsoluteLength`, `DefiniteLength`, `Length` (geometry.go:1398-1854); `IntoDefiniteLength` by `Pixels`, `Rems`, `AbsoluteLength`, `DefiniteLength` (NOT `Length`/`Auto()`); `IntoAbsoluteLength` by `Pixels`, `Rems`, `AbsoluteLength`. Pixels arithmetic uses native operators; `Pixels.Half()` (examples/drag_drop/main.go:38).
- Colors (color.go): `func Rgb(hex uint32) Rgba` :18, `NewRgba(hex uint32) Rgba` :28 ("converts a 0xRRGGBBAA hex value"), `ParseRgba(string) (Rgba, error)` :163, `NewHsla(h, s, l, a float32) Hsla` :275, `Black() White() Red() Blue() Green() Yellow()` :280-320 (return `Hsla`), `TransparentBlack()` :285, `OpaqueGrey(lightness, opacity float32)` :295, `(c Hsla) Opacity(factor float32) Hsla` :427, `LinearGradient(angle float32, from, to LinearColorStop) Background` :647, `NewLinearColorStop(color IntoHsla, percentage float32)` :662, `PatternSlash` :620, `Checkerboard` :633, `SolidBackground` :642.
- `Option[T]` (gpuiutil/option.go:10, alias option.go:6): `Some(v)`, `None[T]()`, `OptionFrom(v, ok)` (option.go:9-19), `(o Option[T]) Get() (T, bool)` (gpuiutil/option.go:34).
- Styling a bare refinement (hover/active/focus callbacks receive `*StyleRefinement`, which implements `Styler`, style.go:232): bind a mixin by hand. From examples/testing/main.go:133-141:
```go
func styled(style *gpui.StyleRefinement) *gpui.Styled[*gpui.StyleRefinement] {
	var s gpui.Styled[*gpui.StyleRefinement]
	gpui.BindStyled(&s, style)
	return &s
}

func hoverBg(style *gpui.StyleRefinement) *gpui.StyleRefinement {
	return styled(style).Bg(gpui.Rgb(0x45475a))
}
```
  `BindStyled[S Styler](m *Styled[S], self S)` (styled.go:18). No library helper exists (grep: only `BindStyled`, `BindStyledImage`).

## 7. Interactivity

Interactive[S] (any element; elements_div.go): `OnMouseDown(button MouseButton, listener func(event *MouseDownEvent, window *Window, cx *App)) S` :533, `OnMouseUp` :559, `OnAnyMouseDown` :553, `OnMouseDownOut` :583, `OnMouseMove(listener func(event *MouseMoveEvent, ...))` :595, `OnMouseExit` :601, `OnScrollWheel` :627, `OnKeyDown(listener func(event *KeyDownEvent, ...))` :668, `OnKeyUp` :680, `OnModifiersChanged` :692, `Hover(f func(style *StyleRefinement) *StyleRefinement) S` :517, `GroupHover(groupName SharedString, f ...)` :527, `Group(SharedString)` :446, `Occlude()` :732, `OnDrop[T any](listener func(value T, window *Window, cx *App)) S` :720, `DragOver[T any](f func(style *StyleRefinement, value T, window *Window, cx *App) *StyleRefinement) S` :698, `CanDrop` :726, `OnDragMove[T]` :613.

StatefulInteractive[S] (after `ID`): `OnClick(listener func(event *ClickEvent, window *Window, cx *App)) S` :979, `OnAuxClick` :985, `Active(f func(style *StyleRefinement) *StyleRefinement) S` :967, `GroupActive` :973, `OnHover(listener func(hovered bool, window *Window, cx *App)) S` :1003, `Tooltip(buildTooltip func(window *Window, cx *App) AnyView) S` :1015, `HoverableTooltip` :1021, `TooltipShowDelay(time.Duration)` :1027, `OnDrag[T, W any](value T, constructor func(value T, offset Point[Pixels], window *Window, cx *App) Entity[W]) S` :991, `Focusable()` :923, `TrackScroll(ScrollHandle)` :955, a11y `Role`, `AriaLabel`, ... :768-916.

Listeners bound to view state: `cx.Listener(...)` (app_context.go:152). examples/testing/main.go:181-185:
```go
						button("decrement").
							OnClick(cx.Listener(func(this *Counter, _ *gpui.ClickEvent, window *gpui.Window, cx *gpui.Context[Counter]) {
								this.decrement(&Decrement{}, window, cx)
							})).
```

Tooltip (elements_div_test.go:390-396; package-internal test, so no `gpui.` qualifiers):
```go
				NewDiv().
					ID("target").
					W(Px(50.)).
					H(Px(50.)).
					Tooltip(func(_ *Window, cx *App) AnyView {
						return AnyViewFromEntity(cx.New(func(*Context[testTooltipView]) *testTooltipView { return &testTooltipView{} }))
					}).
```
(`Window.SetTooltip` takes ownership of the view, AGENT.md "State after M0-exit".)

Drag and drop: examples/drag_drop/main.go. Source (:73-94, the drag preview is an entity whose `*T` implements `Render`):
```go
		children = append(children, gpui.NewDiv().
			ID(gpui.ElementIDNamedInteger("item", uint64(ix))).
			Size32().
			Flex().
			JustifyCenter().
			ItemsCenter().
			Border2().
			BorderColor(color).
			TextColor(color).
			CursorMove().
			Hover(func(this *gpui.StyleRefinement) *gpui.StyleRefinement {
				var s gpui.Styled[*gpui.StyleRefinement]
				gpui.BindStyled(&s, this)
				return s.Bg(color.Opacity(0.2))
			}).
			ChildText(fmt.Sprintf("Item (%d)", ix)).
			OnDrag(dragInfo, func(info DragInfo, position gpui.Point[gpui.Pixels], _ *gpui.Window, cx *gpui.App) gpui.Entity[DragInfo] {
				return cx.New(func(_ *gpui.Context[DragInfo]) *DragInfo {
					moved := info.Position(position)
					return &moved
				})
			}))
```
Target (:101-103, :136-141):
```go
	onDrop := cx.Listener(func(this *DragDrop, info *DragInfo, _ *gpui.Window, _ *gpui.Context[DragDrop]) {
		this.dropOn = gpui.Some(*info)
	})
```
```go
				WhenSome(d.dropOn, func(this *gpui.Stateful[*gpui.Div], info DragInfo) *gpui.Stateful[*gpui.Div] {
					return this.Bg(info.color.Opacity(0.5))
				}).
				OnDrop(func(info DragInfo, window *gpui.Window, cx *gpui.App) {
					onDrop(&info, window, cx)
				}).
```
App drag state: `HasActiveDrag()` app.go:2958, `StopActiveDrag(window)` :2972.

## 8. Actions and keymap

- `type Action interface { Name() string; BoxedClone() Action; PartialEq(other Action) bool }` (action.go:19). `type ActionData struct { Name string; Type reflect.Type; Build func(jsontext.Value) (Action, error); JSONSchema func() *jsonschema.Schema; DeprecatedAliases []string; DeprecationMessage string; Documentation string }` (action.go:29-43). `func RegisterAction(data ActionData)` :53, call from `init`; duplicate names panic when an App loads the registry. Built-ins `NoAction`, `Unbind` (:257, :278).
- Hand-written (examples/testing/main.go:20-61):
```go
// Increment increments the counter.
type Increment struct{}

// Name returns the action name.
func (Increment) Name() string { return "counter::Increment" }

// BoxedClone clones the action.
func (Increment) BoxedClone() gpui.Action { return Increment{} }

// PartialEq reports whether other is an Increment.
func (Increment) PartialEq(other gpui.Action) bool {
	_, ok := other.(Increment)
	return ok
}
```
```go
func init() {
	gpui.RegisterAction(gpui.ActionData{
		Name:  "counter::Increment",
		Type:  reflect.TypeFor[Increment](),
		Build: func(jsontext.Value) (gpui.Action, error) { return Increment{}, nil },
	})
```
- actiongen: `//go:generate go run github.com/megascan/gpui-go/internal/cmd/actiongen` + `//gpui:action namespace=menu` on a struct (docs/key-dispatch.md:11-21). Keys: `namespace`, `name`, `no_json`, `no_register`, `deprecated_aliases`, `deprecated` (internal/cmd/actiongen/main.go:214). Writes `<file>_gen.go`. Verified from an external module (replace probe): `go run .../internal/cmd/actiongen` works, field-less actions and `no_json` actions compile, but an action with JSON fields generates `import "github.com/megascan/gpui-go/internal/jsonschema"` and fails: `use of internal package github.com/megascan/gpui-go/internal/jsonschema not allowed`. So the docs/key-dispatch.md:27-33 `Move` struct example only compiles inside this module. Outside: use `no_json`, or hand-write with `JSONSchema: nil` (the type `*jsonschema.Schema` is unnameable outside but nil is fine).
- Bindings: `func NewKeyBinding(keystrokes string, action Action, context Option[string]) KeyBinding` (keymap_binding.go:31); `func (cx *App) BindKeys(bindings []KeyBinding)` (app.go:2731); `ClearKeyBindings()` :2737; `KeyBindings() *Keymap` :2743; `KeystrokeParse(source string) (Keystroke, error)` (platform_keystroke.go:79); `KeyContextParse(source string) (KeyContext, error)` (keymap_context.go:84); `KeyBindingContextPredicateParse` :570. Example (examples/testing/main.go:227-230):
```go
		cx.BindKeys([]gpui.KeyBinding{
			gpui.NewKeyBinding("up", Increment{}, gpui.Some("Counter")),
			gpui.NewKeyBinding("down", Decrement{}, gpui.Some("Counter")),
		})
```
- Key context on an element: `func (m *Interactive[S]) KeyContext[C ~string | KeyContext](keyContext C) S` (elements_div.go:501; a parse error is logged, not returned).
- Handlers: element `func (m *Interactive[S]) OnAction[A Action](listener func(action A, window *Window, cx *App)) S` (elements_div.go:656), `CaptureAction[A]` :650, `OnBoxedAction` :662. App-level `func (cx *App) OnAction[A Action](listener func(action A, cx *App)) *App` (app.go:2749), e.g. `cx.OnAction(func(_ Quit, cx *gpui.App) { cx.Quit() })` (examples/image_example/main.go:192). `Context[T].OnAction(actionType reflect.Type, window *Window, listener func(this *T, action any, phase DispatchPhase, window *Window, cx *Context[T]))` (app_context.go:437).
- Element `OnAction` wants `func(A, ...)` by value while `cx.Listener` yields `func(*E, ...)`; six examples define this adapter (examples/testing/main.go:127-131, also a11y, focus_visible, input, tab_stop, native_view):
```go
func actionListener[A any](listener func(event *A, window *gpui.Window, cx *gpui.App)) func(action A, window *gpui.Window, cx *gpui.App) {
	return func(action A, window *gpui.Window, cx *gpui.App) {
		listener(&action, window, cx)
	}
}
```
  used as `OnAction(actionListener(cx.Listener((*Counter).increment)))` (examples/testing/main.go:160) with `func (c *Counter) increment(_ *Increment, _ *gpui.Window, cx *gpui.Context[Counter])`.
- Dispatch: `func (cx *App) DispatchAction(action Action)` (app.go:2912), `func (w *Window) DispatchAction(action Action, cx *App)` (window_paint.go:443), `func (h FocusHandle) DispatchAction(action Action, window *Window, cx *App)` (window.go:632). Building from JSON: `func (cx *App) BuildAction(name string, data Option[jsontext.Value]) (Action, error)` (app.go:2771). Availability: `IsActionAvailable` (app.go:2859, window_paint.go:1103). Binding lookup: `Window.BindingsForAction`, `HighestPrecedenceBindingForAction` (window_dispatch.go:1292-1325). Full guide: docs/key-dispatch.md.
- Propagation from an action/event handler: `cx.StopPropagation()` (app.go:2761), `cx.Propagate()` :2766.

## 9. Focus

- `type FocusHandle struct` (window.go:496), exported fields `TabIndex int`, `TabStop bool`. New handle: `func (cx *App) FocusHandle() FocusHandle` (app.go:3065) (also reachable as `cx.FocusHandle()` on `*Context[T]` by embedding). Counted: `Clone()` window.go:640, `Release()` :669 (idempotent). Strict leak mode reports a collected-unreleased focus handle (window.go:514).
- Methods: `Focus(window *Window, cx *App)` :598, `IsFocused(window *Window) bool` :603, `ContainsFocused` :609, `WithinFocused` :615, `Contains` :620, `IsRendered` :626, `DispatchAction` :632, `Downgrade() WeakFocusHandle` :592, builders `WithTabIndex(index int) FocusHandle` :564, `WithTabStop(tabStop bool) FocusHandle` :578 (mutate the shared entry and return the same handle; D-FIELDMETHOD).
- Element side: `TrackFocus(handle FocusHandle) S` (elements_div.go:468) stores `handle.Clone()` (borrows; keep and release your own). `TabStop(bool)` :476, `TabIndex(int)` :482, `TabGroup()` :491, style hooks `Focus(f)` :750, `InFocus(f)` :756, `FocusVisible(f)` :762 (applied when focused and `window.LastInputWasKeyboard()`, elements_div_paint.go:1170; `LastInputWasKeyboard` window_paint.go:1156).
- Window: `Focus(handle FocusHandle, cx *App)` (window_paint.go:312), `Blur(cx)` :327, `FocusNext(cx)` :349, `FocusPrev(cx)` :361, `Focused(cx *App) (FocusHandle, bool)` :284.
- `type Focusable interface { FocusHandle(cx *App) FocusHandle }` (window.go:730; "the caller releases it"); implementations return `c.focusHandle.Clone()` (examples/testing/main.go:123-125).
- Example (examples/focus_visible/main.go:88-101):
```go
	items := []item{
		{
			cx.FocusHandle().WithTabIndex(1).WithTabStop(true),
			"Button with .focus() - always shows border when focused",
		},
```
```go
	focusHandle := cx.FocusHandle()
	window.Focus(focusHandle, cx.App)
```
  and in OpenWindow (examples/testing/main.go:236-240):
```go
			counter := cx.New(NewCounter)
			focusHandle := counter.FocusHandle(cx)
			focusHandle.Focus(window, cx)
			focusHandle.Release()
			return counter
```

## 10. Async

- `type Task[T any] = scheduler.Task[T]` (gpui.go:25). Methods: `func (t *Task[T]) Await(cx Awaiter) (T, error)` scheduler/executor.go:465, `Wait(cx Awaiter) error` :471, `Detach()` :364 (run to completion), `Cancel()` :380 ("running work stops at its next park and never resumes"; both are no-ops after await/cancel/detach), `IsReady() bool` :347, `ReadyTask[T any](v T) *Task[T]` :326, `ReadyTaskErr[T any](err error) *Task[T]` :331. `Task<T>` and `Task<Result<T>>` are both `*Task[T]` (D-ASYNC). No GC-driven cancel: an unstored, undetached task keeps running.
- `type Awaiter interface { scope() *Scope }` (scheduler/future.go:97): implemented by `*scheduler.Scope` and by types embedding it (`AsyncApp` embeds `*scheduler.Scope`, app_async_context.go:19-20).
- Foreground: `func (cx *App) Spawn[R any](f func(cx *AsyncApp) (R, error)) *Task[R]` (app.go:2502), `SpawnWithPriority` :2512; `func (cx *Context[T]) Spawn[R any](f func(this WeakEntity[T], cx *AsyncApp) (R, error)) *Task[R]` (app_context.go:145, "must be held or detached"); `SpawnIn[R any](window *Window, f func(this WeakEntity[T], cx *AsyncWindowContext) (R, error)) *Task[R]` :402.
- Background: `func (cx *App) BackgroundSpawn[R any](future func(s *scheduler.Scope) (R, error)) *Task[R]` (app.go:3338); `BackgroundExecutor.Spawn` (executor.go:84), `SpawnWithPriority` :90; `Scoped(cx scheduler.Awaiter, f func(scope *Scope))` :104.
- Timers: `func (e BackgroundExecutor) Timer(duration time.Duration) *Task[struct{}]` (executor.go:140). From a task: `cx.BackgroundExecutor().Timer(500 * time.Millisecond).Await(cx)`.
- `*AsyncApp` (app_async_context.go): `Update(f func(cx *App))` :188 (no error; panics if the app is released, which "does not happen for foreground tasks spawned with Spawn" :15-16), `UpdateR[R]` :195, `New[T]` :140, `Spawn` :222, `OpenWindow` :211 (error `errAppQuitting` while quitting), `BackgroundExecutor()` :172, `ReadGlobal`/`UpdateGlobal` :239/:271. `*AsyncWindowContext` (:296): `Update(update func(window *Window, cx *App)) error` :311, `UpdateR` :316, `Spawn` :363, `Prompt` :373. Get one outside a task: `func (cx *App) ToAsync() *AsyncApp` (app.go:2035).
- Entity access from a task goes through the weak handle: `this.Update(cx, ...) error` (fails once the entity is released). Cursor-blink loop, examples/view_example/example_editor.go:111-127:
```go
func spawnBlinkTask(cx *gpui.Context[Editor]) *gpui.Task[struct{}] {
	return cx.Spawn(func(this gpui.WeakEntity[Editor], cx *gpui.AsyncApp) (struct{}, error) {
		for {
			if _, err := cx.BackgroundExecutor().Timer(500 * time.Millisecond).Await(cx); err != nil {
				return struct{}{}, fmt.Errorf("blink timer: %w", err)
			}
			result := this.Update(cx, func(editor *Editor, cx *gpui.Context[Editor]) {
				editor.cursorVisible = !editor.cursorVisible
				cx.Notify()
			})
			if result != nil {
				break
			}
		}
		return struct{}{}, nil
	})
}
```
  Replacing a stored task (:95-98): `e.blinkTask.Cancel(); e.blinkTask = task`, with `blinkTask: scheduler.ReadyTask(struct{}{})` as the initial placeholder (:63).
- Detached fire-and-forget (examples/testing/main.go:112-120): `cx.Spawn(...).Detach()`. Background send (examples/testing/main_test.go:229-232): `cx.BackgroundSpawn(func(_ *scheduler.Scope) (struct{}, error) { ...; return struct{}{}, nil }).Detach()`.
- Rule (DESIGN 5.6): a foreground task body may block only through gpui/scheduler primitives (`Await`, `Timer`, `scheduler.Oneshot`/`Chan` receive, `Yield`); a raw Go channel receive or `time.Sleep` blocks the UI thread. Combinators: `scheduler.Select(cx Awaiter, cases ...SelectCase) int` (scheduler/future.go:312), `scheduler.Join[T any](cx Awaiter, futures ...Future[T]) []T` :339, `Join2` :370, `scheduler.Oneshot[T]()` (examples/testing/main_test.go:115). Recover in a task body must re-panic `scheduler.IsCancelUnwind(r)` values (scheduler/future.go:118).
- Log-and-detach helpers: `TaskDetachAndLogErr[T any](task *Task[T], cx *App)` (executor.go:40), `TaskDetachAndLogErrWithBacktrace` :50 (D-TASKEXT). `FutureWithTimeout` (D-FUTUREEXT).

## 11. Testing

- `func RunTest(t *testing.T, fn func(tc *TestRun), opts ...TestOption)` (testing.go:169) = `#[gpui::test]`. Options: `Seed(uint64)` :36, `Seeds(...uint64)` :41, `Iterations(uint64)` :46, `Retries(int)` :52, `OnFailure(func())` :57, `Cases(uint32)` :62 (PropertyTest). `PropertyTest(t, fn func(tc *TestRun, rng *rand.Rand), opts...)` :265.
- `*TestRun` (testing.go:68): `Cx() *TestAppContext` :96, `NewContext() *TestAppContext` :105 (extra apps on the same scheduler), `Rand() *rand.Rand` :117 (PCG seeded with the run seed, D-RNG), `Seed()` :86, `BackgroundExecutor()` :125, `BlockTest(body func(s *scheduler.Scope))` :131 (async test body).
- Teardown (testing.go:139-153): per context RunUntilParked, ForbidParking + Quit, RunUntilParked, release; then `DrainTasks`, then leak report per app. Windows opened in a test need no manual close.
- Seeds (seeds.go:15-18): with `$SEED` set, runs `$SEED..$SEED+iterations` and ignores explicit seeds; else `0..iterations` then explicit seeds (the default single iteration is dropped when explicit seeds are given); `$ITERATIONS` overrides iterations. Multi-run failures print `failing seed: N` and the `SEED` hint (testing.go:216-219).
- `*TestAppContext` (app_test_context.go:125): `New[T]` :201, `Update(f func(cx *App))` :358, `UpdateR[R]` :365, `Read`/`ReadR` :372/:379, `AddWindow[V any](buildWindow func(window *Window, cx *Context[V]) *V) WindowHandle[V]` :393, `OpenWindow[V any](windowSize Size[Pixels], ...) WindowHandle[V]` :406, `AddEmptyWindow() *VisualTestContext` :417, `AddWindowView[V any](buildRootView ...) (Entity[V], *VisualTestContext)` :432, `RunUntilParked()` :659, `Executor() BackgroundExecutor` :348, `DispatchAction(window AnyWindowHandle, action Action)` :664, `SimulateKeystrokes(window AnyWindowHandle, keystrokes string)` :694, `SimulateInput` :704, `DispatchKeystroke` :714, `Notifications[T](entity) *scheduler.ChanReceiver[struct{}]` :727, `Events[T, E](entity) *scheduler.ChanReceiver[E]` :739, `Condition[T](s scheduler.Awaiter, entity, predicate)` :751, `SimulatePromptAnswer(button string)` :479, `SimulateWindowResize` :526, `WriteToClipboard`/`ReadFromClipboard` :453/:459, `ToAsync()` :650, `Spawn` :591. No `AdvanceClock` on `TestAppContext`: use `cx.Executor().AdvanceClock(d)` (`BackgroundExecutor.AdvanceClock` executor.go:165; also `Tick()` :170, `RunUntilParked()` :175, `AllowParking()` :180, `ForbidParking()` :194).
- `*VisualTestContext` (app_test_context.go:893), from `VisualTestContextFromWindow(window AnyWindowHandle, cx *TestAppContext) *VisualTestContext` :917 or `AddWindowView`: `Update(f func(window *Window, cx *App))` :901, `SimulateKeystrokes(keystrokes string)` :949, `SimulateInput` :954, `SimulateMouseMove(position Point[Pixels], button Option[MouseButton], modifiers Modifiers)` :959, `SimulateMouseDown` :968, `SimulateMouseUp` :979, `SimulateClick(position Point[Pixels], modifiers Modifiers)` :989, `SimulateModifiersChange` :1006, `SimulateResize` :1022, `SimulateEvent(event InputEvent)` :1078, `DispatchAction(action Action)` :927, `DebugBounds(selector string) (Bounds[Pixels], bool)` :1039 (pairs with element `DebugSelector`), `Draw[RL, PP]` :1052, `Focus[V](view Entity[V])` :1197, `NewWindowEntity` :1169, `RunUntilParked()` :922, `SimulateClose() bool` :1093.
- Different type: `*VisualTestAppContext` (app_visual_test_context.go:17) runs on a real platform (`NewVisualTestAppContext(platform Platform)` :31) with `OpenOffscreenWindow`, `CaptureScreenshot(window) (*image.RGBA, error)` :321, `Close()` :340. Also `*TestApp` (app_test_app.go:12, `NewTestApp()` :22, `OpenWindow` returning `*TestAppWindow[V]` with `SimulateClick(position, button)` etc. :408-546, `Close()` :293) and `*HeadlessAppContext` (app_headless_app_context.go:17, `CaptureScreenshot` :142, `Close()` :172).
- Entity test futures: `(e Entity[T]) NextEvent[E any](cx *TestAppContext) scheduler.Future[E]` (app_test_context.go:805), `NextNotification(advanceClockBy time.Duration, cx *TestAppContext)` :828, `Condition` :849.
- Rules (AGENT.md): tests never open real windows or touch the session bus, clipboard etc.; display tests only with `GPUI_TEST_DISPLAY=1`. Profiler-only tests: `go test -tags gpui_profiler .`.
- Real test, examples/testing/main_test.go:14-38:
```go
func TestTesting_BasicTesting(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		counter := cx.New(NewCounter)
		defer counter.Release()

		counter.Update(cx, func(counter *Counter, _ *gpui.Context[Counter]) {
			counter.count = 42
		})

		updated := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if updated != 42 {
			t.Fatalf("count = %d, want 42", updated)
		}

		counter.Update(cx, func(_ *Counter, cx *gpui.Context[Counter]) {
			cx.Emit(CounterEvent{})
		})

		countAfterUpdate := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if countAfterUpdate != 999 {
			t.Fatalf("Side effects should run after update completes: count = %d", countAfterUpdate)
		}
	})
}
```
  Window + action (:52-74):
```go
func TestTesting_CounterInWindow(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		window := openCounterWindow(tc.Cx())

		cx := gpui.VisualTestContextFromWindow(window.AnyWindowHandle, tc.Cx())
		counter, err := window.Root(cx)
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		defer counter.Release()

		focusHandle := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) gpui.FocusHandle { return counter.focusHandle.Clone() })
		defer focusHandle.Release()
		cx.Update(func(window *gpui.Window, cx *gpui.App) {
			focusHandle.DispatchAction(Increment{}, window, cx)
		})

		countAfter := counter.ReadWith(cx, func(counter *Counter, _ *gpui.App) int32 { return counter.count })
		if countAfter != 1 {
			t.Fatalf("Action dispatched via focus handle should increment: count = %d", countAfter)
		}
	})
}
```
  Async (:76-107) uses `tc.BlockTest(func(s *scheduler.Scope) {...})`, `task.Await(s)`, `cx.RunUntilParked()`; seeded loop uses `gpui.Iterations(10)` (:160) and `tc.Rand()`.

## 12. Platforms and status

- Linux backend selection (platform/linux/display_connection.go:13-24): Wayland if allowed and `WAYLAND_DISPLAY` non-empty, else X11 if allowed and `DISPLAY` non-empty, else headless; `ZED_HEADLESS` set forces headless (platform/linux/linux.go:21). Force a backend: unset the other variable, or `platform.Linux(gpui.WindowingModesX11 | gpui.WindowingModesHeadless)` (constants `WindowingModesWayland` 1<<0, `WindowingModesX11` 1<<1, `WindowingModesHeadless` 1<<2, platform_cfg_freedesktop.go:103-111). Switch at runtime with `App.RequestWindowing` (app.go:1419). Wayland via libwayland-client + purego, X11 via xgb (DESIGN 8.1).
- Renderer: Linux and web use `render/wgpu` (wgpu-native v29 via cgo on Linux, browser WebGPU on js with a WebGL2 fallback, DESIGN 8.2/8.5). macOS: Metal via purego/objc (`apple.NewMetalHeadlessRenderer` for headless, platform/platform_darwin.go). Windows: D3D11 + DirectComposition + DirectWrite (platform/platform_windows.go registers the WGC screen-capture backend); no headless renderer on Windows (`currentHeadlessRenderer` returns nil).
- Web: `GOOS=js GOARCH=wasm`; build `GOOS=js GOARCH=wasm go build -o <dir>/gpui-web-examples.wasm ./examples/web_hello_web` (OWNER-CHECKS.md:74). Single-threaded only (L-WEBTHREADS).
- iOS: `platform/ios` ported, PARITY examples row `examples/ios_ios` still `todo` (PARITY.md section 6); simulator/device checks are owner checks (OWNER-CHECKS.md:61-65, :129-133).
- Other GOOS: `errPlatformNotPorted` (platform/platform_other.go).
- Verified on: Linux only. Every Linux example has a checked screenshot under docs/port/screenshots (AGENT.md "State after M3-exit"; native_view*, ownership_post, web, iOS excepted).
- UNVERIFIED (TECH_DEBT `PENDING-REAL-OS-TESTS`, 2026-11-24): M4-ci-macos, M4-ci-windows, M4-exit not run because GitHub Actions is blocked by billing (`CI-BILLING`). macOS/iOS/Windows are verified only by cross `go vet` + `go test -c`; Windows tests also run under Wine (`WINE-TEST`) with known Wine-only failures (DirectWrite font matching E_NOTIMPL, UIA stubs, WinRT activation, a cross-built path test).
- Known limits (PARITY.md section 5): L-GLYPH (Linux glyph pixels differ from swash), L-JPEG (JPEG decode pixel values may differ), L-SVGTEXT (text and embedded images in SVG not rendered), L-WGPUVER (wgpu-native v29 vs upstream wgpu 30; fallbacks when `EnumerateAdapters`, `DUAL_SOURCE_BLENDING`, `TRANSIENT_ATTACHMENT` are missing), L-WEBTHREADS (no multi-threaded web dispatcher).
- Other open issues (TECH_DEBT M3-exit): window_positioning on Hyprland intermittently aborts in `wgpuSurfaceConfigure` (negative size wrapped to u32); some element states (list/uniform-list scroll handles, image cache items, `UseKeyedState` entities) still rely on the GC safety net (M0-EXIT-OPEN).

## 13. Go deviations from Rust GPUI (D-*)

Core (DESIGN 5.1-5.7, 9; PARITY.md section 5):

| Id | Rust form | Go form |
|---|---|---|
| D-NAME | `div()`, `img()`, `point()` next to types; `Element::id` | `NewDiv()`, `NewImg`, `NewPoint`; `ElementID()`; `W1_2` keeps digit underscores |
| D-RAII | `Drop` on `Entity`, `FocusHandle`, `Subscription`, `Task` | explicit `Release`, `Unsubscribe`/`Detach`, `Cancel`/`Detach`; GC cleanup only for entity/focus handles |
| D-OPT | `Option<T>`, `Result<T>` | `(T, bool)`, nil-able, `Option[T]` fields, `(T, error)`; panics keep messages |
| D-VOID | closure generic over return `R` | void form + `R` suffix (`Update`/`UpdateR`) |
| D-EMIT | `EventEmitter<E>` bound | not checked at compile time |
| D-COND | conditional impls (`impl<V: Render> IntoElement for Entity<V>`) | method always present, runtime panic naming the type |
| D-INTO | `impl Into<X>` params | concrete type, sealed `IntoX` (`IntoLength`, `IntoHsla`, `IntoFill`), or union-constrained generic (`ElementIDSource`) |
| D-GEOM | `Point<T>` for numbers and non-numbers | numeric `Point[T Number]` with methods + `PointOf[T any]`; single-instantiation impls are free funcs (`EdgesLengthZero`) |
| D-STRING | `SharedString` (SmolStr), `ArcCow` | `type SharedString string`; `ArcCow` not ported; plain strings via `ChildText` |
| D-ASYNC | futures, `.await`, `Task<T>`/`Task<Result<T>>` | iter.Pull coroutines; `task.Await(cx)`; `*Task[T]` returning `(T, error)`; app access via `AsyncApp.Update` |
| D-RNG | `StdRng` seeded | `math/rand/v2` PCG: same determinism, different interleavings per seed |
| D-ARENA | element arena | GC + per-frame slice pools |
| D-TOKIO | `gpui_tokio` | not ported; use `BackgroundExecutor.Spawn` |
| D-HOTRELOAD | subsecond hot reload | not ported |
| D-PROPTEST | proptest | `gpui.PropertyTest`, seeded, no shrinking |
| D-BENCH | criterion | `testing.B` via `gpui.RunBench` |
| D-SCHEMA | schemars | `internal/jsonschema` reflector |
| D-FEATURE | cargo features | always compiled, runtime switches, or tag `gpui_profiler` |
| D-HTTP | `http_client` crate | `gpui.HTTPClient` over `net/http` |
| D-MACRO | proc macros | generators (`actiongen`, `stylegen`, `refinegen`), embedding, reflection, helpers |

Registered in area files (docs/port/parity/*.md, "Deviations registered here"), user-facing core areas:

| Id | Area | Rust form -> Go form |
|---|---|---|
| D-ACTIONDATA | core-app | `Action`'s `Self: Sized` fns + `inventory` -> fields of `ActionData` + `RegisterAction` in `init`; `as_any` -> type assertion |
| D-GLOBAL | core-app | `ReadGlobal::global`, `UpdateGlobal` blanket traits -> `Global[G](cx)`, `UpdateGlobal`/`UpdateGlobalR`/`SetGlobal`; `Global` marker dropped |
| D-TASKEXT | core-app | `TaskExt` on `Task<Result>` -> `TaskDetachAndLogErr(task, cx)`, `TaskDetachAndLogErrWithBacktrace` |
| D-BORROWLOC | core-app | `#[track_caller]` borrow diagnostics -> `file:line` via `runtime.Caller`; `TRACK_THREAD_BORROWS` read once |
| D-ANIMEXT | core-elements | `AnimationExt` methods -> free `WithAnimation`, `WithAnimations`, `WithSpring` |
| D-UNIFORMTRACK | core-elements | `UniformList::track_scroll` -> `TrackUniformListScroll` |
| D-UNIFORMHANDLE | core-elements | `UniformListScrollHandle.0` -> field `State`; `Default` -> `NewUniformListScrollHandle()` |
| D-UNIFORMDECOR | core-elements | `impl UniformListDecoration for Entity<T>` -> `UniformListDecorationFromEntity[T, *T](entity)` (borrows) |
| D-SURFACE | core-elements | `SurfaceSource` CVPixelBuffer -> `uintptr`; `NewSurface` on every OS, paints only on darwin/ios |
| D-LISTCELL | core-elements | `ListState(Rc<RefCell>)` -> shared pointer without borrow tracking; `SpliceFocusable` takes over handles |
| D-WEAKIMAGECACHE | core-elements | `AnyImageCache` strong entity -> `AnyWeakEntity` upgraded on `Load`; caller keeps the entity alive |
| D-IMAGECACHEENTITY | core-elements | `From<Entity<I>> for AnyImageCache` -> `AnyImageCacheFromEntity[I, *I](entity)` |
| D-STATEFULIMG | core-elements | `Stateful<Img>` derefs to `Img` -> `StyledImage` methods only before `ID` |
| D-CTX | core-geometry | generic methods of `AppContext`/`VisualContext`/`BorrowAppContext` -> free funcs over sealed interface (`AppContextNew`, `VisualContextFocus`, `SetGlobal`) + per-context delegates |
| D-IDMIXIN | core-geometry | `InteractiveElement::id` -> separate `InteractiveID[S]` mixin; `Stateful` has no `ID` |
| D-FUTUREEXT | core-geometry | `FutureExt::with_timeout` -> `FutureWithTimeout(future, timeout, executor)` |
| D-FLATENUM | core-input | `MouseButton::Navigate(dir)` -> `MouseButtonNavigateBack`/`Forward`; `TouchPhase` zero = `Moved` |
| D-TYPEORDER | core-input | `available_actions` order -> by `PkgPath()+"."+String()` of the type |
| D-GESTUREDEFAULTS | core-input | `PlatformGestures` defaults -> embeddable `NullPlatformGestures`; `DefaultGestureTuning()`, `DefaultLongPressEvent()` |
| D-SUBMOD | core-platform | `layer_shell::Anchor` -> `LayerShellAnchor` (submodule prefix on collision) |
| D-TUPLE | core-platform | tuples -> named structs (`Credentials`), arrays (`Option[[4]Pixels]`), multiple results |
| D-DEFAULTS | core-platform | provided trait methods -> embeddable `*Defaults` structs bound with `Bind<Name>`; cfg-gated methods on every GOOS |
| D-GPUENUM | core-platform | `repr(C)` GPU enums -> `uint32` |
| D-PLATSEAM | core-platform | `HasWindowHandle` etc. -> `RawWindowHandle`; `Close` on Platform/PlatformWindow; `ScapBackend` seam; `UUID` |
| D-RANGE | core-platform | `Range<usize>` -> `Range{Start, End int}`; `&mut Option<Range>` -> `*Option[Range]` |
| D-WITH | core-platform | builder named like a field -> `With` prefix (`Menu.WithItems`, `MenuItem.WithChecked`) |
| D-GETTER | core-platform | getter equal to a public field -> read the field (`Image.ID`, `ClipboardItem.Entries`) |
| D-IMAGEFRAME | core-platform | image crate `Frame` -> `ImageFrame` with `*image.RGBA` (BGRA, not premultiplied), `Delay time.Duration` |
| D-IMGDECODE | core-platform | image crate decoders -> Go stdlib + x/image + in-repo ICO/PNM/EXIF/WebP-anim; one bad GIF frame fails the image |
| D-PROFTHREAD | core-profiler | thread-local `THREAD_TIMINGS` -> registry keyed by explicit `ThreadID` |
| D-JOURNALTHREAD | core-profiler | thread-local journal -> one process-wide journal behind a mutex |
| D-SUSPENDCLOCK | core-profiler | `system_suspended_time` -> package var set per OS |
| D-BENCHMETRICS | core-profiler | `bench_metrics` crate -> `BenchMeasurement`/`MetricReport`/`Bencher`; `BENCH_MEASUREMENT` = `default` or `wall-time` |
| D-PRIMITIVE | core-style | `Primitive` enum -> sealed interface over `PrimitiveQuad{Quad}` etc.; `IntoPrimitive` |
| D-SURFACEBUF | core-style | `PaintSurface.ImageBuffer` -> `uintptr` on every GOOS |
| D-SPRINGTARGET | core-style | spring/interpolate traits -> `SpringTarget[O]`/`Interpolate[T]` interfaces; `float32`/`bool` via internal dispatch; `SpringAnimation.To` generic method |
| D-FIELDMETHOD | core-text | field and method with one name -> method gets suffix (`FontMetrics.AscentPixels`); `FocusHandle.WithTabIndex` |
| D-SCHEMAKW | core-text | schema keywords `patternProperties`/`multipleOf`/`default` -> absent |
| D-PREPAINTFOCUS | core-window | `AnyElement::prepaint*` returns `Option<FocusHandle>` -> returns nothing |
| D-INTOELEMENT | core-window | `IntoElement::into_element` + associated `Element` -> only `IntoAnyElement`; `AnyElement` is not an `Element` |
| D-VIEWADAPT | core-window | `Entity`/`AnyView`/`RenderOnce` as `View` -> unexported adapters; `into_any_element` moves the handle |
| D-INSPECTORORDER | core-window | inspector states from `TypeIdHashMap` -> first-stored order |
| D-SRCLOC | core-window | `Location::caller()` column -> `SourceLocation.Column` = 0; `CallerSourceLocation(skip)` |
| D-SPRANGE | core-window | `&[Range<ScaledPixels>]` -> `[]ScaledPixelsRange` |
| D-A11YHASH | core-window | synthetic a11y node ids via SipHash -> FxHasher (different numbers, same uniqueness) |
| D-LOG | foundation | `log` crate -> `log/slog`; `debug_assertions` -> `gpuiutil.DebugAssertions` (`-tags gpui_release`) |
| D-SUMVALUE | sumtree | `add_summary(&mut self)` -> value method returning the result; defaults -> free funcs |
| D-SUMSEEK | sumtree | blanket seek impls -> `OrdDimension`, `DimensionsTarget`, `MapSeekKey` |
| D-SUMORD | sumtree | `TreeMap<K: Ord>` -> `K cmp.Ordered` |

Platform, renderer and vendored-crate deviations (full text from the area files; mostly invisible to app code):

| Id | Area | Deviation |
|---|---|---|
| D-APPLE-METALHDR | apple | The build script becomes `internal/cmd/metalhdr` (go:generate in platform/apple/shaders.go, output committed as shaders_scene_gen.go) plus the runtime_shaders path (`stitchedShaderSource` compiled with newLibraryWithSource at start); the prebuilt metallib path is not ported. cbindgen gets the six sources concatenated into one file with inner attributes stripped, where build.rs passes them through Builder::with_src |
| D-EXB-SVGINTO | examples-b | painting builds the Rust logo with `lyon.PathSvgBuilder()` and replays its events into `gpui.PathBuilderFill()` instead of converting the lyon builder |
| D-EXC-NVHANDLE | examples-c | native_view and native_view_appkit_field pass the view's own `NativeView` handle as the child instead of `Clone()` |
| D-EXC-IVAR | examples-c | the background reporter's sender lives in a Go map keyed by the reporter object instead of an objc ivar |
| D-EUCLID | ext-etagere | euclid `Point2D<i32>`, `Size2D<i32>` and `Box2D<i32>` become the `Point`, `Size` and `Rectangle` structs with the same methods used here; the euclid re-export is not ported |
| D-ETAGERE-ATLAS | ext-etagere | `AtlasAllocator` (`src/allocator.rs`) is not ported; gpui uses only `BucketedAtlasAllocator` |
| D-ETAGERE-FFI | ext-etagere | the C ABI module `ffi` is not ported; Go callers use the package directly |
| D-ETAGERE-SERDE | ext-etagere | the `serialization` feature (serde derives) is not ported; `AllocId` serializes through `Serialize` and `AllocIDDeserialize` |
| D-LYON-SUBSET | ext-lyon | only what `gpui/src/path_builder.rs:3-9` and its callees use is ported; attribute builders, flattening, bounding boxes, intersections, reversal, polygon iterators and the unported modules are not |
| D-LYON-F32 | ext-lyon | lyon_geom's `Scalar` generics are instantiated at f32 only (lyon_path uses nothing else); `cast` is absent and the two tests written for inferred f64 (`test_from_svg_arc`, the first half of `test_cubic_to_quadratics`) run at f32 with upstream's tolerances |
| D-LYON-TRIG | ext-lyon | f32 `sin`, `cos`, `tan` and `powf` are computed in float64 and rounded, where Rust calls libm's `sinf` etc.; results can differ by one ulp on rare inputs (the golden test is bit-exact) |
| D-LYON-DEBUG | ext-lyon | `debug_assert!` checks follow two rules: the DebugValidator's begin / end / edge / build checks are always on with upstream's messages, while `nan_check` and the NaN / tolerance debug_asserts are off, as in a release build, because gpui feeds non-finite points through the builder and rejects them later (`path_builder.rs:418-484`) |
| D-LYON-EUCLID | ext-lyon | the euclid types lyon uses become plain f32 structs `Point`, `Vector`, `Angle`, `Transform`, `Rotation`, `Range` in `geom/euclid.go`, with operators as methods (`Add`, `Sub`, `SubVector`, `Mul`, `Div`) and only the methods the subset needs; lyon_path re-exports them as aliases |
| D-LYON-EVENT | ext-lyon | `Event<Ep, Cp>` is a comparable tagged struct (`Kind` plus the variant fields `At`, `From`, `Ctrl`, `Ctrl1`, `Ctrl2`, `To`, `Last`, `First`, `Close`; unused fields are zero); its `from()` / `to()` methods are `FromEndpoint` / `ToEndpoint` because the fields take the names |
| D-LYON-SVGRET | ext-lyon | `SvgPathBuilder`'s `move_to`, `line_to`, `quadratic_bezier_to` and `cubic_bezier_to` return `EndpointID` in Go, because `WithSvg`'s inherent methods of the same names do and Go allows one method per name |
| D-LYON-TRAITDEF | ext-lyon | trait default methods become free functions over the interface (`PathBuilderPathEvent`, `PathBuilderAddPolygon`); `Build` bounds are checked by a type assertion in `NoAttributes.Build` / `WithSvg.Build` instead of a where clause |
| D-LYON-NOATTR | ext-lyon | `NO_ATTRIBUTES` is a nil `Attributes` slice |
| D-SCAP-ROOTTYPES | ext-scap-linux | scap's `Target`, `Display`, `Window`, `Options`, `Area`, `Resolution`, `FrameType` and the frame structs are the root's `gpui.Scap*` types (one set shared by the seam and every back end); `Area` is flattened to x, y, width, height |
| D-SCAP-MODS | ext-scap-linux | Rust module declarations have no Go counterpart; the modules are files of one package |
| D-SCAP-NONLINUX | ext-scap-linux | macOS and Windows items are not part of the Linux back end (ported by M4 stages) |
| D-SCAP-CTEXT | ext-scap-linux | COMPOUND_TEXT window titles are decoded in Go (ASCII, Latin-1 and the UTF-8 escape) instead of Xlib's `XmbTextPropertyToTextList`; other ISO 2022 charsets give "n/a" |
| D-SCAP-XSHM | ext-scap-linux | frames are read through MIT-SHM `GetImage` when the server supports it, falling back to core `GetImage` (upstream always uses core `GetImage`); the bytes are the same |
| D-SCAP-PWLIFETIME | ext-scap-linux | the stream, listener, core and context live until the capture loop ends; upstream drops them when `start_pipewire_capturer` returns, which destroys the stream before any frame arrives |
| D-SCAP-NULLDATA | ext-scap-linux | a buffer whose first data plane is not mapped (`data == NULL`) yields no frame; upstream builds a slice from the null pointer (undefined behaviour). A NULL `pw_buffer` from dequeue is not handed back to `pw_stream_queue_buffer`, which would dereference it now that the stream stays alive (D-SCAP-PWLIFETIME) |
| D-SCAP-DBUSCODEGEN | ext-scap-linux | dbus-codegen's `AppendAll` for the Response signal is only used to emit it; the port only reads it |
| D-SCAP-UNUSEDFRAMES | ext-scap-linux | `RGB8Frame` is not a `Frame` variant and is never produced |
| D-SCAP-BIN | ext-scap-linux | the zed-scap demo binary is not ported |
| D-SCAP-QUEUE | ext-scap-linux | the frame channel holds at most 8 items and drops the oldest when full; upstream's `mpsc::channel` is unbounded and grows by a frame per tick while the consumer stalls |
| D-SCAP-PACKED | ext-scap-linux | Wayland frames honour `chunk.offset`, `chunk.size` and `chunk.stride` and are repacked to `width*bpp` rows (matching `gpui.ScapPixels` and the X11 path); a chunk smaller than the frame gives an error. Upstream copies `maxsize` raw bytes. RGBA is no longer offered in EnumFormat (upstream offers it and then rejects every RGBA frame) |
| D-SCAP-STATE | ext-scap-linux | the Wayland capturer state and stream-error flag are per capturer, not process statics, so capturers do not start, stop or fail each other |
| D-SCAP-CLOSE | ext-scap-linux | `Capturer.Close` stops and joins the capture goroutine (X11 and Wayland) before releasing the connection, so a capturer closed without `stop_capture` leaks nothing; upstream leaks the thread in that case |
| D-SCAP-ERRSTOP | ext-scap-linux | an X11 grab error is sent once and ends the capture (the queue closes) instead of being re-sent every frame; a cursor overlay is skipped when the image is smaller than width*height*4 (non-32bpp visuals) instead of panicking |
| D-SCAP-PORTAL | ext-scap-linux | the portal Response match is added before each request call (filtered by the returned handle), avoiding the signal race; `Start` waits 1200x100ms like `SelectSources` since the picker can appear there; an unknown response code is an error, not `unreachable!()` |
| D-SCAPW-NONWIN | ext-scap-windows | macOS and Linux items are not part of the Windows back end (ported by M1-linux-screencap and the macOS stages) |
| D-SCAPW-QUEUE | ext-scap-windows | the frame channel holds at most 8 items and drops the oldest when full (as D-SCAP-QUEUE); it is closed when the capture thread ends (see D-SCAPW-HARDEN); `Capturer.Close` (the drop of `Capturer`, which drops the receiver) closes it, after which the frame handler's send fails, the handler returns the error and the capture thread halts, as upstream's `tx.send` error does |
| D-SCAPW-WCFORK | ext-scap-windows | zed pins a windows-capture git fork (rev f0d6c1b) whose `Settings::new` takes item, cursor, border, color format and flags; the port follows windows-capture 1.5.0 and passes the three extra settings (secondary windows, minimum update interval, dirty regions) as `Default`, which makes no session calls, so behaviour is the same |
| D-SCAPW-THREAD | ext-scap-windows | the capture thread is a native thread from `CreateThread` running a Go callback (only package platform may lock goroutines to threads); join waits on a channel and the thread handle; there is no `JoinHandle` to hand out, and a panic on the thread ends the process instead of surfacing as `FailedToJoinThread` |
| D-SCAPW-ENCODER | ext-scap-windows | `encoder.rs`, `ImageFormat` and the `save_as_image` methods (with their `UnsupportedFormat`, `ImageEncoderError` and `IoError` errors) are not ported; scap never saves images |
| D-SCAPW-UNMAP | ext-scap-windows | staging textures are unmapped and released when the frame handler returns; upstream releases the texture when `buffer`/`buffer_crop` returns and never unmaps it |
| D-SCAPW-ERRORS | ext-scap-windows | the thiserror enums become sentinel errors for unit variants and wrapper types (`WrappedError`, `D3D11Error`, `MonitorError`, `WindowError`, `WindowNotFoundError`, `FrameError`) for data variants, with upstream's messages |
| D-SCAPW-HARDEN | ext-scap-windows | recoverable failures never panic on the native capture thread: `buffer_crop`/`buffer` failures return a handler error; the crop box is clamped to the frame's size each frame and the reported width/height come from the mapped buffer; the non-crop branch packs rows with `AsNopaddingBuffer`; `Frame.BufferCrop` rejects a box past the texture with `ErrFrameInvalidSize`; start, main-display and target errors are returned (`CreateCapturer`, `GetCropArea`) or delivered to the frame queue; when the capture thread ends (stop, item closed, handler error) its error is queued and the queue closed, so `GetNextFrame` returns instead of blocking |
| D-SCAPW-GENERICS | ext-scap-windows | `GraphicsCaptureApiHandler`'s associated `Flags`/`Error` types and `Self::new` become a Go interface plus a `newHandler func(Context[Flags])` argument; `CaptureControl::callback` locks the handler for a callback instead of returning `Arc<Mutex<T>>`; the item type parameter of `Settings` is the `TryIntoCaptureItemWithType` interface, so scap's `Settings::{Window, Display}` enum is one `wgc.Settings`; `CaptureItemTypes` is a struct with one non-nil field; `MinimumUpdateIntervalSettings::Custom(Duration)` is `{IsCustom, Custom}`; `Send`/`Sync` impls have no Go counterpart |
| D-TAFFY-FEATURE | ext-taffy | Code behind cargo features gpui does not enable is not ported: `calc`, `serde`, `parse`, `detailed_layout_info`, `flexbox_balance`, `debug`, `profile`, `strict_provenance` and the `not(feature = "std")` branches. A test that needs one of them (also without its own cfg, as `calc_round_trips_pointer`) is n/a. |
| D-TAFFY-COMPACT | ext-taffy | `CompactLength` is a `{tag uint8, bits uint32}` struct instead of a tagged pointer: no pointer packing, the `compat` transmutes are `math.Float32bits` / `Float32frombits`, `tag_ptr` / `untag_ptr` do not exist. Equality stays bitwise. |
| D-TAFFY-MAYBEMATH | ext-taffy | The `MaybeMath` trait is not a Go interface: its impls for `Option<f32>` and `f32` are free functions, the two `AvailableSpace` impls are methods with an `F32` suffix for the `f32` right-hand side, and the blanket `Size<T>` impl is one function per instantiation (naming in the notes). |
| D-TAFFY-SYS | ext-taffy | The `sys` type aliases for `String`, `Map`, `Vec`, `Box`, `ChildrenVec` and `GridTrackVec` are not declared; ported code uses `string`, Go maps and slices directly |
| D-TAFFY-TRAITS | ext-taffy | Style traits are Go interfaces. Default method bodies live in embeddable `<Trait>Defaults` structs (`CoreStyleDefaults`, `FlexboxItemStyleDefaults`, ...); `CoreStyle::is_containing_block`, whose default calls `position()`, is the free function `CoreStyleDefaultIsContainingBlock(s)` that implementors call. The associated type `CustomIdent` is the type parameter `S` of the interfaces that use it (`OofItemStyle[S]`); `CheapCloneStr` is the constraint `~string`. `Style`'s fields share names with the trait methods, which Go forbids on one type, so `impl ... for Style` is implemented by `StyleRef[S]` (`(*Style[S]).Ref()`, upstream's `&Style`); `PhantomData` field `dummy` and the `impl for &T` forwarding impls do not exist. |
| D-TAFFY-ALIGNFIELDS | ext-taffy | `AlignItems` / `AlignContent` keep their `keyword` / `safety` fields unexported because the `keyword()` method would collide with an exported field; read them with `Keyword()` / `Safety()`, build with `NewAlignItems` / `NewAlignContent` or the `AlignItems<Name>` vars. Upstream's associated consts are package vars (Go has no struct consts). |
| D-TAFFY-SIZES | ext-taffy | `style_sizes` checks only sizes that Go lays out like Rust; `Option<AlignItems>` (Go 3 bytes, no niche), the string-dependent grid types and `Style` sizes differ and are not asserted. |
| D-TAFFY-GRIDLISTS | ext-taffy | The associated iterator types of `GridContainerStyle` and `GenericRepetition` are slices: template track lists are `[]GenericGridTemplateComponent[S, GenericRepetition[S]]` (the repetition is the interface, so `R` is not a trait parameter), auto track lists `[]TrackSizingFunction`, line names `TemplateLineNames[S]` (alias of `[][]S`), areas `[]GridTemplateArea[S]`. `StyleRef.GridTemplateRows` / `GridTemplateColumns` allocate the component-ref slice per call where upstream maps lazily. |
| D-TAFFY-SMALLVEC | ext-taffy | `SmallVec<[T; N]>` is a Go slice: there is no inline capacity, so `spilled()` is checked as `len <= N` (`alternating_spans_merge_occupancy_and_skip_the_occupied_prefix`). |
| D-TAFFY-HELPERS | ext-taffy | `style_helpers`: `line` is `NewLine` (Go cannot have a function and the type `Line` with one name). Go cannot add methods to `float32` or `gpuiutil.Option`, so the `f32` / `Option` impls are runtime cases inside the Point / Line / Size / Rect helpers (`PointZero[float32]`, `SizeLength[Option[float32]]`) and are not usable with the top-level generic helpers; `Option<T>` is covered for `T = f32` only. `repeat`'s `TryInto<RepetitionCount>` input is the union of `uint16`, `string` and `RepetitionCount`. |
| D-TAFFY-TREETRAITS | ext-taffy | Tree traits are Go interfaces with the conventions of D-TAFFY-TRAITS: associated style types are the style interfaces (`CoreStyle`, `OofItemStyle[S]`, ...), `CustomIdent` is the type parameter `S` of `LayoutContainingBlock` / `LayoutGridContainer` (`LayoutPartialTree` is not generic because `CoreStyle` is not), `ChildIter` is `iter.Seq[NodeID]`, `*const ()` is `unsafe.Pointer`, `BlockContext<'_>` is `*BlockContext`, and default bodies live in `<Trait>Defaults` structs or, when they call another trait method, in `<Trait>Default<Method>` functions. The marker trait `TraverseTree` has the marker method `TraverseTree()`. `get_detailed_layout_info` returns the value, not a reference. `print_tree` is `PrintTreeStdout`, because the trait `PrintTree` takes the name. `LayoutPartialTreeExt` methods are unexported free functions taking the tree. |
| D-TAFFY-CLONE | ext-taffy | `OofCandidates` holds its lazily allocated list behind a pointer (nil is upstream's `None`, a non-nil pointer to an empty slice is `Some(empty)`), so Go value copies of `OofCandidates`, `LayoutOutput` and `Cache` share it; ported code calls `Clone()` where upstream clones (`Cache::get` does) and `Equal` where upstream compares. The debug-build `MODE_CHANGE_EVICTIONS` thread-local is one process-wide atomic counter, counted while `gpuiutil.DebugAssertions` is set. |
| D-TAFFY-RESOLVE | ext-taffy | `MaybeResolve` / `ResolveOrZero` are generic interfaces implemented by `LengthPercentage`, `LengthPercentageAuto` and `Dimension` for an `Option<f32>` context; the blanket `f32`-context impl is `MaybeResolveF32`, and the `Size` / `Rect` impls are free generic functions (`SizeMaybeResolve`, `SizeMaybeResolveF32`, `SizeResolveOrZero`, `RectResolveOrZero`, `RectResolveOrZeroOptionF32`, `RectResolveOrZeroF32`). The `calc` closure is `func(unsafe.Pointer, float32) float32`. |
| D-TAFFY-RESULT | ext-taffy | `TaffyResult<T>` has no Go type: TaffyTree methods return `(T, error)` (or `error` for `()`), the error being a `TaffyError` value comparable with `==`; references (`&Style`, `&Layout`, `&[NodeId]`) are pointers / slices into the tree. `get_node_context(_mut)` return a nil-able pointer, `get_disjoint_node_context_mut` takes variadic ids and returns `([]*C, bool)`. |
| D-TAFFY-CTXDROP | ext-taffy | Go has no destructors: removing or clearing nodes drops the tree's reference to their contexts, which `remove_drops_the_nodes_context` / `clear_drops_every_nodes_context` check as an empty `node_context_data` instead of a drop counter. |
| D-TAFFY-RANGE | ext-taffy | `remove_children_range`'s `RangeBounds<usize>` is the half-open pair `start, end` (`1..=2` is `1, 3`); invalid ranges panic with core's slice-range messages. |
| D-TAFFY-XMLPARSE | ext-taffy | The `parse` feature is not ported (D-TAFFY-FEATURE), but `tests/xml.rs` reads every style through `FromStr`; the `FromCss` impls it reaches (keyword enums, alignment, lengths, `AvailableSpace`, `Contain`, grid placement / tracks / repetitions / auto-flow) are ported test-only on a minimal cssparser-like tokenizer in `xml_css_test.go`. Numbers parse as f64 then convert to f32, as cssparser does. |
| D-TAFFY-TESTNAMES | ext-taffy | Test helper constructors of `tests/common` are unexported (`testNodeContextZero`, ...) because `go vet` rejects `Test*` functions that are not tests. |
| D-GOTEXT-CTOR | gotext | `CosmicTextSystem::new` / `new_without_system_fonts` are `NewTextSystem` / `NewTextSystemWithoutSystemFonts` (the type keeps upstream's name); cosmic-text, fontdb, swash metrics and skrifa line metrics are re-expressed on go-text/typesetting in `fontdb.go`, `font.go`, `shape.go` and `fallback_*.go` |
| D-ZENO-FILL | gotext | Only zeno's fill path is ported (no stroker, segment iterator, SVG parser or AdaptiveStorage); `Style` is reduced to `Fill`, and the unscratched render path uses `HeapStorage` |
| D-SWASH-PNG | gotext | CBDT / sbix PNG strikes are decoded with Go's image/png into straight RGBA instead of swash's own inflate + PNG decoder |
| D-LINUX-CALLOOP | linux-common | calloop is replaced by `event_loop.go`: the goroutine that calls `Dispatch` / `Run` polls the registered fds plus an eventfd wakeup with `poll(2)`, fires due timers, then runs idle callbacks, with calloop's `PostAction`, `LoopSignal` (stop flag + wakeup ping), ping (eventfd), channel, generic and timer sources and the before-sleep / before-handle-events hooks; callbacks are bound when a source is inserted (`InsertSource`). The dispatcher's timer thread runs its own `EventLoop` as upstream's does |
| D-LINUX-RTPRIO | linux-common | `SpawnRealtime` runs f on a plain goroutine without SCHED_FIFO priority 65 |
| D-LINUX-BUS | linux-common | LinuxCommon opens the session and system buses lazily through the replaceable `ConnectSessionBus` / `ConnectSystemBus` fields; `SystemNotifications()` creates the notification state on first use; `Close` ends the dispatcher's goroutines and closes the buses |
| D-LINUX-CLIENT | linux-common | The Wayland and X11 connections satisfy the `LinuxClient` interface, which LinuxPlatform's DisplayConnection holds, instead of an enum matched by the `dispatch!` macro |
| D-LINUX-OPEN | linux-common | open::commands is ported without its WSL PowerShell launcher; launchers, order and option-safe paths match open-5.4.5 |
| D-LINUX-GPUHINT | linux-common | `CompositorGPUHintFromDevT` returns `(vendorID, deviceID, ok)` instead of `gpui_wgpu::CompositorGpuHint`; callers build `wgpu.CompositorGPUHint` |
| D-LINUX-BOUNDS | linux-common | `ReadFdWithTimeout` fails once a pipe delivers more than `MaxPipeReadSize` (256 MiB); a portal request waits at most `PortalResponseTimeout` (5 min) for its Response, then closes the Request and fails |
| D-LINUX-NOCELL | linux-common | LinuxPlatform's `RefCell` fields and `with_common` become plain fields used on the main goroutine; replacing a connection swaps the field before closing the old one, as upstream does outside the borrow |
| D-LINUX-FILECHOOSER | linux-common | The FileChooser portal is called with godbus (`fileChooserRequest`) instead of ashpd: a missing portal is detected from the `version` property or the D-Bus error name and reported as `FILE_PICKER_PORTAL_MISSING`; the blocking call runs on the background executor while the foreground task awaits it |
| D-WLFOREIGN | linux-wayland | `WaylandConnection.WindowIdentifier` exports the focused surface through xdg-foreign (zxdg_exporter_v2, else v1) synchronously on the main goroutine with a private event queue, as ashpd does on a helper thread, and keeps one export per surface until the window drops or the connection closes; later calls for the same surface return the same `wayland:<handle>`. Upstream creates a new export per call and destroys it when the returned identifier drops |
| D-X11-CLIPBOARD | linux-x11 | Selection serving answers the advertised text/plain;charset=utf-8 / UTF-8 targets from UTF8_STRING data, uses the target as the property for ICCCM requests with property None, and sends data larger than the server's request limit (capped at 256 KiB) through INCR; reading caps INCR at 512 MiB and 30 s, pre-sizes the INCR buffer from the owner's length hint only up to 1 MiB (`maxIncrPrealloc`) and requires a 32-bit hint (SEC-029) and reports INCR TARGETS replies as ATOM; SetImage offers the image under its own format's MIME atom instead of always image/png. A failed dialog setup also removes the dialog from its parent's children |
| D-MACOS-OBJCBOOL | macos | `BoolExt::to_objc` has no Go symbol: Go `bool` is passed to and returned from `objc.Send` directly and purego marshals it as Objective-C `BOOL` |
| D-MACOS-CARBON | macos | The Carbon / HIToolbox externs of `platform.rs` (TIS*, UCKeyTranslate, LMGetKbdType, kTIS* statics) are lazily resolved purego bindings in `platform.go`, loaded by `loadCarbon`; `CFDataGetBytePtr` is bound there too |
| D-MACOS-PLATFORMIVAR | macos | The `platform` ivar of `GPUIApplication` / `GPUIApplicationDelegate` is a package map from the object to its `*MacPlatform` (`setMacPlatformIvar` / `getMacPlatform`, same panic messages); the classes are registered lazily by `NewMacPlatform` instead of a `#[ctor]` |
| D-MACOS-SCKFRAME | macos | `ScreenCaptureFrame.Frame` is the sample buffer's CVImageBufferRef borrowed for the duration of the frame callback, not a retained `CVImageBuffer` released on drop |
| D-MACOS-WINDOWIVAR | macos | The `windowState` ivar of `GPUIWindow` / `GPUIPanel` / `GPUIView` is the package map behind `storeWindowState` / `loadWindowState` / `clearWindowState`; `get_window_state` panics with the ivar's `expect` message when the object has no entry, and the classes are registered lazily on first use (`sync.OnceValue`) instead of a `#[ctor]` |
| D-PLATFORM-BENCHTS | platform-select | `BenchTextSystem` builds the text system once per process (`sync.OnceValue`) instead of once per thread |
| D-PLATFORM-FAMILYSUMMARY | platform-select | `family_summary`, the last Criterion target, is a benchmark declared last in its file, so `go test -bench` runs it after the tree benchmarks; it prints and clears the summary on its first call |
| D-WGPUCAPS | wgpu | wgpu-native's C API has no adapter texture-format-features or downlevel-capabilities query; the webgpu facade answers `GetTextureFormatFeatures` from the usages WebGPU guarantees for the formats gpui uses (Bgra8Unorm, Rgba8Unorm, R8Unorm, Rgba32Uint, Depth32Float) and `GetDownlevelCapabilities` on GL from `MaxStorageBuffersPerShaderStage > 0` (VERTEX_STORAGE and FRAGMENT_STORAGE together). Affects `select_color_texture_format`, `headless_target_format` and `requires_texture_instance_data` |
| D-WGPUCONFIGURE | wgpu | wgpu-native v29 handles every `wgpuSurfaceConfigure` error with `handle_error_fatal` (process abort) instead of the device error sink, and the binding has no uncaptured-error callback. The webgpu facade's `Surface.Configure` therefore runs wgpu-core's `configure_surface` / `validate_surface_configuration` checks in Go first (view formats differing only in sRGB-ness, TooLarge against the device's `max_texture_dimension_2d`, present mode with Undefined as Fifo, format, alpha mode with Auto falling back to Opaque / Inherit, ZeroArea, in that order and with wgpu-core's messages) and reports a failure into the device's error scope without calling wgpu-native, so `try_adapter_with_surface` falls through to the next adapter as upstream. Not pre-checked, still fatal: the usage check (the binding drops `SurfaceCapabilities::usages`), the SURFACE_VIEW_FORMATS downlevel flag, colour space, and hal-level configure / GPU-wait failures |
| D-WGPUATLASBOUNDS | wgpu | `insert` rejects a tile whose bytes are fewer than width x height x bytes-per-pixel (1 for monochrome, 4 otherwise) with an error before allocating, instead of queueing a short upload that fails later in `write_texture`; `allocate` returns None for a tile larger than `max_texture_dimension_2d` without creating a texture, and drops a freshly pushed texture when its first allocation fails, where upstream keeps an empty clamped texture in storage forever |
| D-WEBGLPENDING | wgpu | Resolved by M4-web-webgl: the WebGL2 branch (preference WebGl, or Auto without `navigator.gpu`) now creates a render/wgpu/internal/gpu/webgl2 instance and canvas surface and follows the same adapter / device path as WebGPU. That facade maps the API as wgpu-hal's GLES backend (wgpu rev cddb2f5) with three differences: Bgra8Unorm data is swizzled to RGBA on upload and readback because WebGL2 has no BGRA external format; render-only textures are textures, not renderbuffers; bind groups are resolved against the current pipeline layout at draw time (wgpu-core's lazy binding) because the facade's `SetBindGroup` has no layout argument |
| D-WINVSYNCDIV | windows | `retrieveDuration` returns `(time.Duration, error)`: a zero tick rate is an error (so `NewVSyncProvider` falls back to `defaultVsyncInterval`), and a tick rate under 1 MHz (the `rateRefresh` fallback passes `uiNumerator`, e.g. 60) is converted exactly as `counts*1e6/ticksPerSecond` microseconds; at 1 MHz and above upstream's formula is kept |
| D-WINDISPFAIL | windows | When `TrySubmitThreadpoolCallback` or `CreateThreadpoolTimer` fails, the runnable is taken back out of the handle table and run on a goroutine (`time.AfterFunc` for the timer); a failed wake `PostMessageW` resets `WakePosted` to false |
| D-WINLASTERR | windows | `WindowsDisplayPrimaryMonitor` reads the last error captured by the `MonitorFromPoint` call itself (`win32.MonitorFromPointLastError`) |
| D-WINSTRNUL | windows | `com.NewHSTRING` / `NewBSTR` encode with `utf16.Encode` and `HSTRING.String` / `BSTRToString` decode the full counted length with `utf16.Decode`, so embedded NULs round-trip |
| D-WINSHADERRT | windows | build.rs's fxc step is replaced by compiling the embedded, byte-identical shaders.hlsl with `D3DCompile` (d3dcompiler_47) on first use (`NewRawShaderBytes`, platform/windows/directx_shaders.go, the `shader_resources` module of directx_renderer.rs), with the build.rs entry points and vs_4_1/ps_4_1 targets; flags are `D3DCOMPILE_OPTIMIZATION_LEVEL3` (fxc `/O3`), or upstream's debug flags under `gpuiutil.DebugAssertions`; the include is served by a Go `ID3DInclude` instead of fxc's file-system include; no SDK lookup |
| D-WINDXIFACE | windows | `DirectXRenderer` keeps upstream's methods (`DrawWithAppearance`, `Resize`, `HandleDeviceLost`, `MarkDrawable`, `RenderToImage`) and adds the `render.Renderer` set: `Draw(scene)` clears with the background chosen by `UpdateTransparency`, `GPUSpecs` logs `gpu_specs`' error, `DeviceLost` is false and `Recover` a no-op because the platform detects loss and calls `HandleDeviceLost`, `SupportsDualSourceBlending` is true, `SetSubpixelLayout` a no-op (the order comes from DirectWrite) |
| D-WINDEVLOST | windows | A failed device-lost recovery in the vsync goroutine logs the error and retries on the next vsync instead of panicking; the platform window's `WM_GPUI_GPU_DEVICE_LOST` handler ignores (returns 1) devices that are not the current vsync goroutine's, and the goroutine then skips the per-window messages, so a stale goroutine left by a headless/windowed toggle never replaces the live devices |
| D-WININITDX | windows | `connectInitially` logs a DirectX init failure and stays headless instead of panicking |
| D-WINNEWCLEAN | windows | `NewWindowsPlatform` destroys the platform window, closes the text system and calls `OleUninitialize` on every error return after they were created, and checks `CreateWindowExW`'s error before using the create context |
| D-WINREVEAL | windows | `openTargetInExplorer` frees both PIDLs with `CoTaskMemFree` and compares against `HRESULT_FROM_WIN32(ERROR_FILE_NOT_FOUND)`, so the open-parent-folder fallback runs |
| D-WINCREDNUL | windows | Credential username and target names containing NUL are rejected with an error (`UTF16FromString`) instead of being truncated at the Win32 boundary |
| D-WINSCAPSTUB | windows | `ScreenCaptureSources` delivers an error through the oneshot until the root scap glue (M3-root-capture) lands, instead of panicking |
| D-SEC-IMGBUDGET | core-platform | Raster decoders (static decoders, GIF and animated WebP frames, PNM/PAM) reserve every buffer against a per-image budget before allocating: one buffer may not exceed 512 MiB (`imageDecodeMaxAllocBytes` = `webpMaxCanvasBytes`), all buffers of one decode (every GIF/WebP animation frame plus the RGBA conversion) may not exceed 1 GiB, and `w*h*bpp` is computed with overflow checks; an image over budget fails with `errImageLimitsExceeded` instead of allocating or panicking. (SEC-001, 002, 003) |
| D-SEC-SVGHREF | core-text | SVG `<image href>` never resolves a string href (local path or URL): `SvgRenderer.parseOptions` installs a resolver whose `ResolveString` refuses every target, so no file is read; only `data:` hrefs decode (SEC-004) |
| D-SEC-SVGBUDGET | core-text | Embedded `data:` raster images in an SVG reserve their decoded size against the raster decode budget (D-SEC-IMGBUDGET) before okage decodes them, and over-budget images are dropped; filters are costed before rasterising (filter region clipped to a 5x canvas, weighted per primitive: blur/drop shadow 5, turbulence by octaves, morphology by radius, convolve by kernel size) and a render whose total exceeds 2^24 work units fails with `errSvgFilterWorkExceeded` (SEC-005, 006) |
| D-SEC-FONTINTERN | core-text | `FontFeatures` and `FontFallbacks` intern their content through `unique.Handle[string]`, so interned keys are released once no value refers to them instead of living in a never-evicted `sync.Map`; a JSON feature value above `math.MaxUint32` is rejected with `strconv.ErrRange` instead of truncated (SEC-035) |
| D-SEC-ASSETHASH | core-app | Asset cache keys hash the source with a per-process random `maphash` seed (`keyedHash`, the FxHash word stream fed into a seeded `maphash.Hash`) instead of the unkeyed 64-bit FxHash, so two sources cannot be forged to share a cache slot; `Image.ID` (`ImageFromBytes`, `imageID`) is a seeded `maphash` of format and bytes, and the clipboard readers on X11, Wayland, Windows and macOS build images through `ImageFromBytes`; `Hash` keeps upstream's FxHash for other callers (SEC-031) |
| D-SEC-ASSETCACHE | core-app | The app's asset cache is an LRU capped at 4096 entries and 1 GiB of loaded bytes (`maxAssetCacheEntries`, `maxAssetCacheBytes`; render images count their frame bytes, `Image`s and byte slices their length); inserting past either cap evicts least-recently-used entries, including failed loads, and notifies the views that observed them so they refetch (SEC-036) |
| D-SEC-IMGPATH | core-elements | `ImageAssetLoader.Load` of a path resource reads only regular files: it stats the path and refuses anything but a regular file, opens it, checks the opened file is the same regular file, and caps the read at 512 MiB (`imageAssetMaxBytes`), so `/dev/zero`, FIFOs and devices fail with an error instead of exhausting memory or blocking (SEC-007) |
| D-SEC-HTTPBODY | core-elements | `ImageAssetLoader.Load` of a URI sends the request with a 60 s context deadline (`imageRequestTimeout`), caps the response body at 512 MiB, and builds the non-2xx error from at most 4 KiB of body, cut to its first line, 1 KiB, with control and bidi-control characters removed (SEC-012, 014, 015); the `HTTPClient` contract is unchanged |
| D-SEC-KEYCTX | core-input | `KeyContextParse` returns an `unexpected character` error for a character that starts no identifier instead of recursing forever, and context predicates nest at most 128 levels (`maxPredicateDepth`, `errPredicateTooDeep`) instead of overflowing the stack (SEC-033, 034) |
| D-SEC-FONTPANIC | gotext | Panics from the go-text font parser and shaper on malformed font data are recovered: a resource that panics while loading or parsing face info is skipped, `charmap` returns glyph 0, a face whose shaping panics is marked unloadable and the next fallback face is tried, and glyph rendering returns an error (SEC-008) |
| D-SEC-GLYPHSIZE | gotext | Bitmap glyph sizes are computed in checked 64-bit math with each side at most 16384 px and an image at most 256 MiB (`maxGlyphImageDim`, `maxGlyphImageBytes`); PNG strikes check the IHDR size against the target before decoding and the decoded size after; the strike count is bounded by the table length; a glyph data length that differs from `w*h` is rejected instead of panicking; index format 4 lookups advance with `l = i + 1` (SEC-009, 010, 011, 068) |
| D-SEC-TASKPANIC | scheduler | A panic inside a background task body no longer kills the process: `BackgroundExecutor` wraps the body in `recoverBackgroundPanic`, which prints the panic and stack to stderr and resolves the task with `*PanicError{Value, Stack}`; cancellation unwinds are re-panicked and foreground task panics still propagate (SEC-037) |
| D-SEC-OPENURL | linux-common | `OpenURIInternal` opens only `http`, `https` and `mailto` URLs and logs an error for any other scheme; `RevealPathInternal` opens its target with `O_NONBLOCK` and reveals the parent directory for anything that is neither a directory nor a regular file, so a FIFO never blocks a background worker; `OpenWithSystem` prefixes a path starting with `-` with `./` before passing it to `xdg-open` (SEC-016, 017, 018) |
| D-SEC-READFD | linux-common | `ReadFdWithTimeout` also fails once the whole read takes longer than `PipeReadDeadlineFactor` (5) times the per-chunk timeout, so a clipboard owner trickling bytes cannot hold the reader indefinitely (SEC-027) |
| D-SEC-KEYMAPFD | linux-wayland | The `wl_keyboard.keymap` handler fstat()s the fd and refuses a keymap whose file is smaller than the declared size (mapping past EOF raises SIGBUS); a keymap libxkbcommon rejects is logged and ignored instead of panicking (SEC-024) |
| D-SEC-XRMINCLUDE | linux-x11 | `#include` in the X resource database (RESOURCE_MANAGER) opens files with `O_NONBLOCK`, reads only regular files of at most 1 MiB, skips non-UTF-8 paths, and processes at most 64 included files per database, so `/dev/zero`, FIFOs and include fan-out are refused (SEC-025) |
| D-SEC-XCURSOR | linux-x11 | The Xcursor parser rejects an image whose `4*w*h` pixels exceed the bytes left in the file before allocating, and caps the decoded pixels of one cursor file at 256 MiB (`maxDecodedBytes`) across all TOC entries (SEC-026) |
| D-SEC-DBUSSENDER | linux-dbus | Signals that complete a request are accepted only from the unique name that owns the service: notification `ActionInvoked` from the owner of `org.freedesktop.Notifications`, Secret Service prompt `Completed` from the owner of `org.freedesktop.secrets`, and FileChooser `Response` from the owner of `org.freedesktop.portal.Desktop` (resolved with `NameOwner`), so a same-bus peer cannot forge them (SEC-020, 021, 022) |
| D-SEC-SECRETPLAIN | linux-dbus | When the Secret Service refuses the `dh-ietf1024-sha256-aes128-cbc-pkcs7` session, credential calls fail with `ErrPlainSessionRefused` instead of falling back to a `plain` session that would send the password unencrypted on the bus (SEC-023) |
| D-SEC-A11YPARA | ext-a11y-consumer | `Position.BackwardToParagraphStart` stops when a backward line step makes no progress (an empty text run linked by `next_on_line`), instead of looping forever while holding the tree read lock (SEC-040). |

Not a D-id but a port-wide shape rule (AGENT.md "API shape"): generic methods on concrete types, free generic funcs over sealed `AppContext`, `R` suffix for value-returning closures, `(T, error)`, `(T, bool)` / `Option[T]`, `iter.Seq`, enums `type X uint8` with `XVariant` constants, `TypeId` -> `reflect.Type`, elements `Element[RL, PP]`, self-typed mixins, no `any`.

## 14. Go-port pitfalls (with evidence)

1. `WindowOptions{...}` literal is not `..Default::default()`. Zero values: `Show=false`, `Focus=false`, `IsMovable=false`, `IsResizable=false`, `IsMinimizable=false`, `Titlebar=None`, `Kind=nil`. macOS (`platform/macos/window.go:836-840`) and Windows (`platform/windows/window.go:483`) only show the window when `Show` is true; Linux ignores it, so hello_world (`examples/hello_world/main.go:57-59`, also image_example, text, svg_example, web_hello_web) works on Linux only. Upstream hello_world uses `..Default::default()` (`_ref/gpui/examples/hello_world.rs:103-104`). A nil `Kind` falls to the normal-window `default:` branch on macOS (`platform/macos/window.go:652-659`). Use `gpui.DefaultWindowOptions()` and set fields. This is from reading the code: no macOS or Windows run has checked it (PENDING-REAL-OS-TESTS).
2. Forgetting `Release`. Every `cx.New`, `Clone()`, `cx.Entity()` (app_context.go:27), `WindowHandle.Root` (window.go:2364), `Focusable.FocusHandle` (window.go:731), `UseState`/`UseKeyedState` (window_paint.go:1909) and `cx.FocusHandle()` result is a counted handle you must release. Use `defer h.Release()` or release it in the entity's `Drop(cx *gpui.App)`. In tests, strict mode reports the miss after a GC. Outside tests nothing reports it, and the GC cleanup eventually releases the handle.
3. Moving a handle without meaning to. `Child(entity)`, `AnyViewFromEntity(e)` and `OpenWindow`'s builder return value all take over the handle (view.go:128, view.go:23, app.go:1674). Write `Child(e.Clone())` if you still need `e`. `TrackFocus(h)` clones internally (elements_div.go:471), so keep and release your own `h`.
4. Tasks never stop by themselves. A `*Task` you neither store nor `Detach` keeps running, and strict leak mode does not report it (there is no task cleanup hook). Store tasks and `Cancel` them in `Drop`, or `Detach` them. `Cancel` on a nil `*Task` panics because it dereferences `t.kind` (scheduler/executor.go:380-381), so start fields with `scheduler.ReadyTask(struct{}{})` (examples/view_example/example_editor.go:63). Cancel the old task before replacing it (:95-98).
5. Subscriptions are the same. `Observe`/`Subscribe` return a `Subscription`, and an unused one keeps firing until its entities die. Call `.Detach()` to keep it on purpose, or store it and `Unsubscribe()` it in `Drop` (examples/testing/main.go:87-90). Strict mode reports one that is collected without either (subscription.go:278).
6. The `Styled` return type follows the receiver. After `.ID(...)` the chain is `*gpui.Stateful[*gpui.Div]`, so `When`/`WhenSome` closures must be typed `func(this *gpui.Stateful[*gpui.Div], ...) *gpui.Stateful[*gpui.Div]` (examples/drag_drop/main.go:136). Before `ID` they take `*gpui.Div` (examples/anchor/main.go:118). `OnClick`, `Active`, `Tooltip`, `OnDrag` and `OverflowScroll` exist only after `ID` (they are `StatefulInteractive` methods). A helper that returns `*gpui.Div` cannot be followed by `OnClick` until `.ID()` is called.
7. Hover/Active/Focus/FocusVisible callbacks get a bare `*StyleRefinement` with no fluent methods. Bind a `Styled[*StyleRefinement]` yourself (examples/testing/main.go:133-141, examples/drag_drop/main.go:83-87). `Hover` set twice panics while debug assertions are on (elements_div.go:519-521).
8. Element `OnAction` takes `func(action A, window, cx *App)` by value, but `cx.Listener` returns `func(*E, window, cx *App)`. The examples copy the same `actionListener` adapter six times because the library has none (examples/testing/main.go:127-131).
9. Generic methods cannot live in interfaces (Go 1.27, RESEARCH-deps.md). So `AppContext` is sealed and only has type-erased primitives (gpui.go). Generic operations are concrete-type methods (`cx.New`, `entity.Update`) or free functions (`AppContextNew(cx, ...)`, `UpdateGlobal[G](cx, ...)`). Code that is generic over contexts must take `gpui.AppContext` and call the free functions. Type parameters are sometimes needed explicitly, for example `cx.SpawnIn[struct{}](window, ...)` (examples/move_entity_between_windows/main.go:31) and `cx.SubscribeIn[HelloWorld, MoveToNewWindow](...)` (:50).
10. Main thread. Import `github.com/megascan/gpui-go/platform`, whose `init` locks the main goroutine to the OS thread (platform/platform.go:13-15), and call `Application().Run` from `main`. App state is main-thread only (the subscriber sets have no lock, api-contracts.md 4.1). Background goroutines reach it only through a foreground task's `AsyncApp.Update` (DESIGN 5.6).
11. Blocking inside a foreground task with `time.Sleep`, a raw `<-ch` or a mutex wait freezes the UI. Await gpui primitives instead: `Timer(...).Await(cx)`, `scheduler.Oneshot`, `scheduler.Select` (DESIGN 5.6). A deferred `recover()` in a task body must re-panic `scheduler.IsCancelUnwind(r)` (scheduler/future.go:118).
12. Re-entrancy. Updating or reading an entity from inside its own `Update` panics with "cannot update X while it is already being updated" (app_entity_map.go:393). Use `cx.Defer`/`DeferIn`, or the `this *T` you were already given.
13. Async error shapes differ by context. `AsyncApp.Update` returns nothing and panics once the app is gone (app_async_context.go:188). `WeakEntity.Update` returns `error` ("entity released", app_entity_map.go:826). `AsyncWindowContext.Update` returns `error` (app_async_context.go:311). In a loop, check `WeakEntity.Update`'s error to know when to stop (example_editor.go:117-123).
14. `Px` has two meanings. `gpui.Px(500.0)` is the `Pixels` constructor; `.Px(len)`/`.Px2()` on an element is horizontal padding. Untyped numbers do not satisfy `IntoLength`, so write `Size(gpui.Px(500))`, not `Size(500)`. `P`/`Px`/`Gap` take `IntoDefiniteLength`, so `gpui.Auto()` is rejected there.
15. actiongen outside this module: actions with JSON fields generate an import of `internal/jsonschema` that does not compile (section 8, verified by probe). Use `no_json` or a hand-written registration. `examples/internal/support` cannot be imported from outside the module either.
16. Leak-mode surprise: `GPUI_LEAK_DETECTION=0` does not turn detection off inside `go test`; it switches from strict to "on" (app_entity_map.go:843-847). Strict reports are process-global, so don't use `t.Parallel()` with leak-asserting tests (api-contracts.md 4.1).
17. `TestAppContext` has no `AdvanceClock`; use `cx.Executor().AdvanceClock(d)`. `VisualTestContext` (a window inside a `TestAppContext`) and `VisualTestAppContext` (a real-platform offscreen app) are different types.
18. On the web, `Run` returns, so `main` must block (`select {}`, examples/web_hello_web/main.go:368). There are no system fonts: call `cx.TextSystem().AddFonts(...)` before rendering text (examples/web_hello_web/main.go:334).
19. `Svg.Path` and `ImageSourceFromString` with an asset path resolve through the app's `AssetSource`, which is set by `Application.WithAssets` (examples/svg_example/main.go:79-81). Without an asset source nothing renders. `ImageSourceFromPath` reads the filesystem directly.
20. Docs drift: DESIGN 6.x and api-contracts.md disagree with the code in places (`AppContextNew` returns no error; `AppContextWithWindow` takes `EntityID` and returns `(R, bool)` at gpui.go:118). Copy signatures from code.

## Examples (examples/, one line each)

- a11y: accessibility tree demo with roles, labels and Tab/TabPrev actions; needs an a11y adapter to observe.
- active_state_bug: regression case for `Active` styling on a clicked element.
- anchor: anchored popovers for each `Anchor` corner, using `NewDeferred` + `NewAnchored`.
- animation: `WithAnimation` on SVGs and divs, with easings and repeat.
- data_table: large virtualized table built from `NewUniformList`, a scroll handle and a `NewCanvas` scrollbar.
- drag_drop: `OnDrag` with an entity preview, plus `OnDrop` on a target.
- focus_visible: `Focus` vs `FocusVisible` styles, tab indices, Tab/Shift-Tab actions.
- gif_viewer: animated GIF through `NewImg(ImageSourceFromPath)`.
- gradient: linear gradients across color spaces; canvas path painting.
- grid_layout: CSS-grid layout (`Grid`, `GridCols`, spans).
- hello_world: minimal window with styled divs (section 1).
- image_example: images from assets, files and URLs; `NewComponent` image containers.
- image_gallery: remote images through a manual and an automatic image cache.
- image_loading: images that load fast, load slowly, fail or are missing; `ImageSourceFromFunc`, timers.
- input: full single-line text input with IME (`EntityInputHandler`), selection, clipboard and actions; has tests.
- internal/support: shared helpers (bundled fonts, `LoadFonts`), not a program.
- ios_ios: iOS smoke app (GOOS=ios, Xcode wrapper); PARITY row still todo.
- layer_shell: Wayland layer-shell clock surface (Linux Wayland only).
- list_example: bottom-aligned variable-height `List` with a hand-drawn scrollbar.
- mouse_pressure: shows force-touch pressure events.
- move_entity_between_windows: one entity moved between windows; `SpawnIn` tick task and `SubscribeIn`.
- native_view: embeds AppKit native views (macOS only; other OSes print and exit).
- native_view_appkit_field: AppKit text field with keyboard input (macOS only).
- on_window_close_quit: quits the app when the last window closes.
- opacity: element opacity over images and SVGs.
- ownership_post: entities, Observe/Subscribe/Emit from docs/ownership-and-data-flow.md.
- painting: freehand and path painting on a canvas, `PathBuilderFill`.
- paths_bench: path-rendering stress test.
- pattern: `PatternSlash` pattern backgrounds.
- popover: floating layers with `NewDeferred`.
- scrollable: scrollable div with `OverflowScroll`.
- set_menus: application menus (visible on macOS).
- shadow: box shadow variants.
- svg_animation: animated SVG transformations (`WithTimedTransformation` vs animation).
- svg_example: the same SVG drawn three times in different colors via an `AssetSource`.
- system_notifications: OS notifications and their responses.
- tab_stop: tab order across `TabIndex`/`TabStop` elements.
- testing: counter app plus tests showing RunTest, VisualTestContext, BlockTest, Iterations and seeds.
- text: font specimens, text styles and components (`NewComponent`).
- text_layout: text alignment and wrapping layout.
- text_wrapper: line wrapping and truncation.
- toggle_headless: switching between windowed and headless at runtime (`RequestWindowing`).
- tree: deep child hierarchy stress test.
- uniform_list: 50-row `NewUniformList` with click handlers.
- view_example: editor/text-area views composed from entities, with blink tasks, `Drop` and tests.
- web_hello_web: wasm gallery (prime sieve, dynamic fonts, queue/HTTP tests); GOOS=js.
- window: opening, closing and configuring windows (kinds, titles, timers).
- window_movable: movability settings.
- window_positioning: requested vs actual window bounds on each display.
- window_shadow: client-side decorations with a custom shadow and resize edges.
