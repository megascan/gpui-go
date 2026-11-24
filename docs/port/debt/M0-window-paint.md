# M0-window-paint debt

- Element state values that own handles (`UseKeyedState` stores an `Entity[S]`) are never released when `frame.clear` / `elementStateMap.clear` drops them (window.go); upstream's drop of the `ElementStateBox` releases them. Under strict leak detection the GC safety net reports them. Same for `AnyTooltip.View` in tooltip requests dropped by `frame.clear` or `Transact`. Whoever next edits window.go's frame clearing should release them.
- `IsWindowHovered` picks the hovered flag with `runtime.GOOS` (windows, linux, freebsd) as upstream's `cfg!`; js/wasm and darwin use `IsWindowActive`.
- `profiling::finish_frame!()` in `present` is not ported (no puffin).
- The four `feat:profiler` debug-overlay methods call M3-profiler-hang placeholders: `DebugFrameOverlayMode` returns 0 and `CycleDebugFrameOverlayMode` panics until then.
