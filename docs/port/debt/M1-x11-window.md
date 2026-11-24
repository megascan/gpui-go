# M1-x11-window debt

- platform/linux/x11/window.go: X11WindowState has no RefCell borrow guard; upstream's re-entrant `borrow_mut` panics are not reproduced (callbacks are still taken out and restored around calls, as upstream).
- platform/linux/x11/window.go: `xcbFlush` is a no-op; xgb writes each request as it is issued. The round trip before surface creation is an explicit GetInputFocus reply.
- platform/linux/x11/clipboard.go: selection condvars are re-expressed as close-and-replace channels; the reader connection pumps events on a goroutine instead of polling with a 1 ms sleep (same timeouts).
- platform/linux/x11/a11y.go: `newA11yAdapter` returns nil until M3-exit wires a11y/atspi.
- No live-display test for X11Window or Clipboard (a window test would open a window, a clipboard test would touch the user's selection).
- window.go: the round trip runs after CreateWindow and the property/XInput setup, before the surface; mapping stays in MapWindow as upstream (decision 15 says "created and mapped", but mapping early would break Show=false and the WM-hint ordering).
