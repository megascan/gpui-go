# Handoff: M3-root-capture

- M3-exit: `gpui.ScapScreenSources(executor)` now exists. Replace the panicking body of `scapScreenSources` at the end of platform/linux/x11/client.go with `return gpui.ScapScreenSources(executor)` (client.rs:1688; resolves the M1-x11-window and M1-x11-client handoffs).
- M4-ci-windows: replace `scapScreenSources` in platform/windows/platform.go with `gpui.ScapScreenSources(p.foregroundExecutor)` (platform.rs:733), with the `RegisterScapBackend` call from the M4-windows-screencap handoff.
- Capturer `Close()` (M1-linux-screencap, M4-windows-screencap handoffs) is called by the root glue through an `interface{ Close() }` assertion wherever upstream drops the Capturer; nothing left for the back ends.
- M3-exit (optional): the godoc of `gpui.ScapPixels.Data` in platform.go says "rows without padding"; Wayland frames carry PipeWire row padding (M1-linux-screencap handoff). Reword it, e.g. "the packed pixel data; rows may be padded on Wayland".
- Resolution (2026-11-24, M3-exit): scapScreenSources wired; ScapPixels.Data godoc reworded.
