# M0-test-platform handoff

- stubs.go: `DispatchEventResult` stub now carries its final exported fields `Propagate` and `DefaultPrevented` (upstream `pub propagate`, `pub default_prevented`, window.rs:2532), because `TestWindow.SimulateInput` reads `Propagate`. M0-window-dispatch keeps these names.
- `TestWindow` is a pointer type (`*TestWindow`); `Clone` returns another handle sharing the state (upstream `Rc<Mutex<TestWindowState>>`). `TestPlatform` holds `activeWindow *TestWindow`; window identity compares `state` pointers (`Rc::ptr_eq`).
- `TestPlatform` / `TestWindow` pub(crate) fields stay unexported for M0-test-contexts: `platformInfo`, `activeWindow`, `prompts`, `openedURL`, `systemNotifications`, `textSystem`, `expectRestart` (`*scheduler.OneshotSender[TestRestartRequest]`, D-TUPLE), `appLifecycleCallback`, `memoryWarningCallback`; window state `bounds`, `handle`, `title`, `edited`, `documentPath`, `shouldCloseHandler` (lock with `w.lock()` / `w.unlock()`).
- Constructors: `NewTestPlatform`, `TestPlatformWithTextSystem`, `TestPlatformWithPlatform(executor, fg, textSystem, headlessRendererFactory func() (PlatformHeadlessRenderer, error))` (nil factory = None, nil renderer = None).
- `bench-support` feature: `testWindowFrameWakerSchedulesFrame` (package var, false) makes `FrameWaker` also schedule a frame; M3-bench-context sets it.
- cfg-gated clipboard methods (primary on linux/freebsd, find pasteboard on darwin, `get_raw_handle` on windows) pick upstream's body by `runtime.GOOS` and fall back to the defaults elsewhere.
