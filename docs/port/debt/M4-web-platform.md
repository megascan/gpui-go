# M4-web-platform debt

- platform/web/stubs.go holds panicking stubs for events.rs / ime_mirror.rs / text_system.rs / canvas_fallback.rs symbols (registerEventListeners, showVirtualKeyboard, setVirtualKeyboardEnabled, focusImeMirror, isMacPlatform, platformInfo, imeMirror methods, newWebTextSystem). NewWebPlatform panics until M4-web-input and M4-web-text replace them. EventListenerHandle, WebEventListeners, clickState, touchIDs and CanvasFontFallback are real and should move to events.go / canvas_fallback.go.
- FetchHTTPClient: no RedirectPolicy (WEB-REDIRECT); request bodies are read with a plain io.Copy inside the send task (fine for in-memory bodies, blocks the page for a streaming body); response Body.Read must run on a goroutine, never in a task or browser callback (Go wasm deadlocks).
- WebPlatform.Run returns immediately like upstream; a Go wasm main must then block (select {}) or the module exits.
- InitLogging always uses Info (no debug_assertions in Go); console_error_panic_hook has no Go counterpart (Go wasm panics already print to the console). Log target is the caller's Go package path.
- WebDispatcher.IsMainThread is always true (L-WEBTHREADS).
- Untested in a browser: verify only compiles for js/wasm. M4-web-example's screenshots are the first runtime check.
