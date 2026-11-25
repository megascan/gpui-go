# M4-windows-screencap handoffs

- M4-ci-windows: register the back end in platform/platform_windows.go with `gpui.RegisterScapBackend(screencap.Backend{})` (import `github.com/megascan/gpui-go/internal/win/screencap`), as platform/linux does for Linux.
- M4-ci-windows: run `go test ./internal/win/wgc/... ./internal/win/screencap/...` on the runner (amd64; arm64 if a runner exists). Pure tests: IIDs, struct layouts, UTF-16, FrameBuffer packing, queue, casts. `TestCapture_PrimaryMonitor` and `TestCapturer_MainDisplay` need `GPUI_TEST_DISPLAY=1` and an interactive desktop; they skip when Windows.Graphics.Capture is unavailable.
- M3-root-capture / M4-ci-windows: the Windows `gpui.ScapCapturer` (`screencap.Capturer`) has `Close()`; call it through an `interface{ Close() }` assertion where upstream drops the Capturer, as for Linux. Close makes the capture thread stop at its next frame.
