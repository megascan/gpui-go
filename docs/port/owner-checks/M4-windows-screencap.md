# M4-windows-screencap owner checks

- YOU: on Windows 10 1903+ or 11, run a gpui example that shares the screen (or `GPUI_TEST_DISPLAY=1 go test -run 'TestCapture_PrimaryMonitor|TestCapturer_MainDisplay' ./internal/win/wgc/ ./internal/win/screencap/`) and check that frames arrive, the yellow capture border appears and disappears on stop, and the cursor shows only when requested.
- YOU: on a multi-monitor Windows machine, check that the display list shows each monitor's friendly name and that capturing a window target works.
