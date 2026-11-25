# M0-window-dispatch handoff

window.rs 5700-7890 is in window_dispatch.go, window/prompts.rs in window_prompts.go, the smoke test in window_smoke_test.go.

API shape:
- `OnMouseEvent[E MouseEvent](func(*E, DispatchPhase, *Window, *App))`, `OnKeyEvent[E KeyEvent](...)`: listeners receive the pointer the dispatcher passes (`*MouseDownEvent`, `*KeyDownEvent`, ...).
- `Prompt(level, message, detail Option[string], answers []PromptButton, cx)`; a nil receiver from `PlatformWindow.Prompt` is upstream's None (falls back to the custom/fallback renderer). `fallback_prompt_renderer` is `NewFallbackPromptRenderer` (a `promptBuilder` renderer func). `PromptResponse` is `type PromptResponse int`. `PromptHandle.WithView[V]` checks `*V` implements `Render[V]` at run time and uses `Entity[V].FocusHandle`.
- `RenderablePromptHandle.view` is `promptViewHandle{anyView() AnyView; release()}`; the window releases the prompt view when a prompt is replaced or answered.
- `handler_for` is a method (`(Window).HandlerFor`); the scan listed it as an assoc fn.
- `PendingInput() (PendingInputStatus, bool)`, `PendingInputKeystrokes() ([]Keystroke, bool)`, `SetPendingInputTimeoutPaused[T](owner Entity[T], paused, cx) bool`, `WithInspectorState[T, R](id *InspectorElementID, cx, f) (R, bool)`, `WithImageCache[R]`, `ObserveGlobal[G](cx, f)`.
- `pendingInputIsNone()` (upstream cfg(test)) exists for M0-window-tests.

Stubs (stubs.go):
- M0-window-a11y: `windowA11y` gained placeholder fields `forceDisabled`, `focusIDs map[a11y.NodeID]FocusID`, `actionListeners map[a11y.NodeID][]a11yActionEntry` (`a11yActionEntry{action a11y.Action; listener a11yActionListener}`, `a11yActionListener func(a11y.ActionData, *Window, *App)`), set by `newWindowA11y`; `isEnabled()` is final (`!forceDisabled`); `debugTreeJSON() (string, bool)` panics. `Window.OnA11yAction` / `handleA11yAction` use these names.
- M0-div-api: `Interactive[S].TrackFocus(FocusHandle) S` stub added (FallbackPromptRenderer.Render uses it, plus NewDiv / ID / OnClick stubs). Rendering a fallback prompt panics until M0-div-api / M0-div-paint land.

For M0-exit (files without a builder owner):
- `Window.Spawn` / `SpawnWithPriority` (window_paint.go) build the AsyncWindowContext from `app.clone()`, which sets `Scope` to nil, so awaiting inside a window task panics. window_dispatch.go uses a private `spawnWindowTask` that passes the scoped AsyncApp; switch it back to `Window.Spawn` once that is fixed.
- The window root view is never released: `releaseAppWindows` (app_test_context.go) and app.go's window removal drop `window.root` without `Release`, so every test window leaks its root entity under strict leak detection. window_smoke_test.go takes and releases the root by hand at the end; remove that once the removal path releases the root.
- A detached prompt subscription never closes its oneshot sender when the prompt view is released unanswered (upstream drops the sender, cancelling the receiver).
