# Debt: M3-root-capture

- `scapDefaultTargetCaptureSource.Release()` stands in for upstream dropping the source (ends a capture whose stream was never requested). `ScreenCaptureSource` has no drop method in Go, so callers must reach it through an `interface{ Release() }` assertion; a forgotten source keeps the capture goroutine and capturer alive. Consider adding `Release` to the `ScreenCaptureSource` interface (contract change, exit stage).
- `StyledReflectionMethods` embeds styled.go and styled_gen.go and parses them once at run time to recover godoc (Go has no run-time doc strings); adds the source text to binaries that link the root package.
