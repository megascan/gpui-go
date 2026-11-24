# Handoff: M0-input-types

New stubs in stubs.go (final signatures; the named stage replaces them):

| Stub | Stage |
|---|---|
| `(*App).PlatformInfo() PlatformInfo` | M0-app-effects (app.rs:2155) |
| `(*App).Global[G any]() *G` | M0-app-effects (app.rs:2676); global.go's `Global[G](cx)` calls it |
| `(*Window).Appearance() WindowAppearance` | M0-window-paint (window.rs:3315) |
| `Empty` and `(Empty).IntoAnyElement` | M0-element (element.rs:795); `(*ExternalPaths).Render` returns `Empty{}` |
| `(BackgroundExecutor).Timer(time.Duration) *Task[struct{}]` | M0-executors (executor.rs:200); util.go's `FutureWithTimeout` calls it |
| `LongPressEvent`, `TouchDragEvent`, `BackGestureEvent` (empty structs) | M0-gestures-spring: replace only the type declarations; their `PlatformInput` methods (`isPlatformInput`, `KindName`, `TouchEvent`) are in interactive.go, and gestures.go adds `ToPlatformInput` / `sealedInputEvent` (+ `isMouseEvent`/`isGestureEvent` as upstream) |

Contract notes for api-contracts.md (exit stage): `ClickEvent` is a tagged struct (`Kind ClickEventKind`, fields `Mouse`, `Keyboard`, `Touch`, constructors `ClickEventMouse` / `ClickEventKeyboard` / `ClickEventTouch`); `PlatformInput` gained the methods `KindName() string` and `TouchEvent() (TouchEvent, bool)`; `Fluent[S]` gained `WhenElse`, `WhenSome`, `WhenNone`; the action registry is the unexported `actionRegistry` (`newActionRegistry()`), for M0-app-core to hold on App.

Window dispatch (M0-window-dispatch): `platformInputMouseEvent` / `platformInputKeyboardEvent` return a pointer to a copy of the event (`*MouseDownEvent`, ...; `*FileDropEvent` for the file-drop variants) or nil.
