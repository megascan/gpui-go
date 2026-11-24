# M4-web-platform handoff

- M4-web-input: replace the events.rs / ime_mirror.rs stubs in platform/web/stubs.go (move EventListenerHandle, WebEventListeners, clickState, touchIDs into events.go; add `add_non_passive`). Window inner fields events.rs uses already exist on webWindowInner (isComposing, touchTapCandidate, suppressFocusStatusEvents, gestureStartVisualViewportHeight, pressedButton, clickState, touchIDs). registerVisibilityChange / registerAppearanceChange / registerFullscreenChange are in window.go; withCallback / withInputHandler are generic funcs in window.go.
- M4-web-text: replace newWebTextSystem and move CanvasFontFallback (Go zero value is Disabled; the default is DefaultCanvasFontFallback() = Emoji) into canvas_fallback.go.
- M4-web-example: main must block after app.Run (select {}).
