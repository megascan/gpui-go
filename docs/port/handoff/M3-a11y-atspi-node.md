# Handoff: M3-a11y-atspi-node -> M3-a11y-atspi-adapter

- a11y/atspi/stubs.go holds the context.rs, events.rs and adapter.rs shapes node.go uses. `Context`, `AppContext`, `newContext`, `adapterIndex`, `pushAdapter`, `readTree`, `readRootWindowBounds`, `readAppContext` and `DoAction` are real code (move them to context.go); `Adapter.emitObjectEvent` panics. Event / ObjectEvent / Property / CacheEvent / WindowEvent shapes are final candidates for events.go (CacheEvent is `{Removed bool; ID}`; change it if a sealed form reads better).
- `Context.released` must be set where upstream drops the adapter's `Arc<Context>` (D-ATSPIWEAK), otherwise PlatformNode never reports Defunct until GC.
- `readTree` returns the tree and an unlock func; node.go keeps upstream's lock scopes (`do_action_internal` unlocks before `DoAction`, the selection/text/scroll methods call it under the read lock).
- Blocked rows in ext-a11y-atspi.md carry guessed Go symbols; rename freely.
- `paritycheck -lint` currently fails on internal/linux/xcbshim/setup.go:5 (comment rule), not in a11y/atspi.
- `Context` is exported (upstream `pub(crate)`); unexport it when context.go lands if nothing outside the package needs it.
