# M1-wayland-client handoff

## For M1-wayland-input (client_input.go)

- stubs.go holds the second-half listeners as panicking stubs with final signatures; implement them in client_input.go and delete them from stubs.go: `seatListener(seat)`, `dataDeviceListener()`, `primarySelectionDeviceListener()`, `dataSourceListener(source, kind)`, `primarySelectionSourceListener(source)`. Go listeners are installed at proxy creation (client.go already calls these), so every other proxy you create (pointer, keyboard, text input, gestures, data offers, fractional scale, decorations, dialogs) needs its listener set right after the constructor.
- All `WaylandClientState` fields of client.rs are declared in client.go (unexported, upstream names). `Close` already releases pointer, cursor-shape device, data device, text input and closes keymap / compose state.
- The 10 client.rs tests (`TestClient_*`) are ported in client_test.go and `done`; `dragState`, `fileURIList`, `updateIMECursorRectangle`, `setIMECursorRectangleAfterDone` are in client.go.
- `ObjectID` is the `*wl.Proxy` of an object (`&surface.Proxy`), never its protocol id, which is reused.

## For M1-wayland-window (window.go, clipboard.go, popup.go, layer_shell.go)

- stubs.go declares `WaylandWindowState`, `WaylandWindowStatePtr` and the methods client.go calls on it (`Handle`, `Surface`, `Toplevel` (nil when not a toplevel), `PtrEq`, `IsBlocked`, `PrimaryOutputScale`, `GetIMEArea`, `ScheduledFrameFired`, `RetryTimerFired(time.Time)`, `FrameCallbackFired(gpui.Option[scheduler.Instant])`, `HandleSurfaceEvent`, `HandleOutputChanged`, `HandleXdgSurfaceEvent`, `HandleToplevelEvent`, `HandleLayersurfaceEvent`, `HandlePopupEvent`, `Close`, `SetAppearance`, `SetButtonLayout`), `WaylandWindow` (must keep a `ptr WaylandWindowStatePtr` field and implement gpui.PlatformWindow), `newWaylandWindow(...) (*WaylandWindow, ObjectID, error)`, and the clipboard.rs symbols (`Clipboard`, `NewClipboard`, `Set`, `SetPrimary`, `SelfMime`, `Read`, `ReadPrimary`, `DataOffer[T]`, `fileListMimeType`, `textMimeTypes`). Replace them and delete stubs.go when empty.
- Protocol events are sealed Go types in client.go (`WlSurfaceEvent*`, `XdgSurfaceEvent*`, `XdgToplevelEvent*`, `ZwlrLayerSurfaceV1Event*`, `XdgPopupEvent*`). Install the dispatchers on the objects you create: `SurfaceListener(surface)`, `XdgSurfaceListener(surfaceID)`, `ToplevelListener(surfaceID)`, `LayerSurfaceListener(surfaceID)`, `PopupListener(surfaceID)`, `FrameCallbackListener(surfaceID)` on each frame wl_callback.
- `PendingActivationWindow{Window: surfaceID}` + `WaylandConnection.requestActivationToken` is the activation path for `activate()`.
- `WaylandConnection.Close` disconnects the wl_display: windows (and their wgpu renderers) must be closed before it.

- Owed drop test (DESIGN 5.2): once stubs.go no longer panics in `Attach`, add a `GPUI_TEST_DISPLAY=1`-gated Attach -> Close test (binding a seat opens no window): the loop registrations are removed, `WaylandClientStatePtr.TryGetClient` fails, and the display is disconnected.

## For M1-wayland-window

- `WindowIdentifier` (xdg_foreign.go) is untested because exporting needs a toplevel; add a gated test once windows open (the shoot-verified examples), checking the result starts with `wayland:`.

## For M1-wayland-ffi / an exit stage

- D-WLFOREIGN: xdg-foreign-unstable-v1/-v2 are hand-defined in platform/linux/wayland/xdg_foreign.go. If wlproto gains them through wlgen, switch to the generated types. Releasing the export when the identifier drops needs `common.LinuxClient.WindowIdentifier` to return a releasable value.
