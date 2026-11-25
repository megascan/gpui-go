# M0-blocked-tests debt

- input_test.go releases the views captured by `NewElementInputHandler` by hand (`inputTestReleaseHandlerViews`) because ElementInputHandler has no Release (handoff item 1).
- input_test.go uses window_test.go's `windowTestCanvas` in place of `canvas(...)`, which M0-elements-basic ports later; switch to `Canvas` once it exists.
- `TestLineWrapper_WrapShapedLine` is gated with a runtime.GOOS skip instead of a darwin build constraint (the gate is one test in a shared file). It uses the NoopTextSystem as upstream does and also passes on Linux.
