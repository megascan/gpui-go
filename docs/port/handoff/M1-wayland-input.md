# M1-wayland-input handoff

## For M1-wayland-window

- client_input.go holds the seat, keyboard, pointer, pinch, text-input, data device / offer / source and primary selection listeners. stubs.go gained stubs for symbols it calls that window.rs / clipboard.rs own: `IMEInput` (+ `IMEInputInsertText`, `IMEInputSetMarkedText`, `IMEInputUnmarkText`, `IMEInputDeleteText`), `WaylandWindowStatePtr.HandleIME/HandleInput/SetFocused/SetHovered/HandleFractionalScaleEvent/HandleToplevelDecorationEvent`, `NewDataOffer`, `(*DataOffer[T]).AddMimeType`, `Clipboard.SetOffer/SetPrimaryOffer/Send/SendPrimary/SendBytes` (fd arguments are owned: close them).
- Install `FractionalScaleListener(surfaceID)` on each wp_fractional_scale_v1 and `ToplevelDecorationListener(surfaceID)` on each zxdg_toplevel_decoration_v1 right after creating them; events arrive as `WpFractionalScaleV1EventPreferredScale` / `ZxdgToplevelDecorationV1EventConfigure`.
- `DataOffer` values are copied into the clipboard (Rust clones them); keep the copy independent of later `AddMimeType` calls (e.g. `slices.Clip` the mime type slice in AddMimeType or clone in SetOffer).
- Touch: upstream's Wayland client never binds wl_touch, so there is nothing to port; xdg_wm_dialog_v1 / xdg_dialog_v1 have no events, so no listeners exist.
