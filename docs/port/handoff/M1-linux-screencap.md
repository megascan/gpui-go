# Handoff: M1-linux-screencap

- M3-root-capture: the Linux `gpui.ScapCapturer` (`screencap.Capturer`) also has `Close()`. Upstream drops the Capturer (closing the Wayland D-Bus connection) when `ScapStream` / the capture thread ends; call it through an `interface{ Close() }` assertion at that point.
- M3-root-capture: on Wayland, frames carry the PipeWire buffer's `maxsize` bytes (row padding included), and the formats are RGBx, RGB, xBGR or BGRx, not BGRA. On X11 they are BGRx from ZPixmap. The godoc of `gpui.ScapPixels.Data` says "rows without padding", which is not true for Wayland frames; the root owner may want to reword it.
- Exit stage: hoisting `platform/linux/x11/internal/xim/ctext.go` to a shared linux-internal package would let screencap reuse it (D-SCAP-CTEXT).
