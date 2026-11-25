# M0-elements-basic debt

- Animation element state (`animationState`) holds the pending step wake task; upstream cancels it when the state is dropped with the frame. The window's element-state map has no drop hook, so a discarded state's wake still fires and notifies the view once (a spurious re-render, no leak).
- `Surface` reads the CVPixelBuffer size through `RegisterSurfaceSourceSize`; nothing registers it yet (handoff to the Apple stages).
- `UniformListScrollHandle{}` (zero value) has no state and panics on use; upstream's `Default` builds a fresh state. Use `NewUniformListScrollHandle()`.
- `PulsatingBetween` and `Heartbeat` evaluate sin / exp in float64 and round to float32, which can differ from Rust's f32 libm by one ulp.
- `TestUniformList_*` use a local weak-entity processor (`uniformListTestProcessor`) instead of upstream's `cx.processor`, because `Context.Processor` leaks its strong handle (handoff).
- Animations use the unexported `windowSpawn` helper instead of `Window.Spawn`, whose cloned AsyncApp has no Scope and cannot Await (handoff).
