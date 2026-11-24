# Parity: linux-x11

Scope: gpui_linux, X11 back end (linux/x11.rs, linux/x11/*). Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux/x11/client.rs

Go target: `platform/linux/x11/client.go` (package `x11`), tests in `client_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `EventHandlerError` | enum | 126 |  | `EventHandlerError` | done |
| `Xdnd` | struct | 155 |  | `Xdnd` | done |
| `X11ClientState` | struct | 182 |  | `X11ClientState` | done |
| `X11ClientStatePtr` | struct | 245 |  | `X11ClientStatePtr` | done |
| `X11ClientStatePtr::get_client` | method | 248 |  | `(X11ClientStatePtr).GetClient` | done |
| `X11ClientStatePtr::drop_window` | method | 252 |  | `(X11ClientStatePtr).DropWindow` | done |
| `X11ClientStatePtr::update_ime_position` | method | 277 |  | `(X11ClientStatePtr).UpdateIMEPosition` | done |
| `mode_refresh_rate` | fn | 2228 |  | `ModeRefreshRate` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `key_event_state_uses_event_modifiers_without_mutating_server_state` | test | 2944 |  | `TestClient_KeyEventStateUsesEventModifiersWithoutMutatingServerState` | done |
| `key_event_state_ignores_pointer_button_bits` | test | 2968 |  | `TestClient_KeyEventStateIgnoresPointerButtonBits` | done |
| `key_event_state_preserves_non_locked_layout_components` | test | 2997 |  | `TestClient_KeyEventStatePreservesNonLockedLayoutComponents` | done |
| `capslock_toggle_produces_uppercase` | test | 3023 |  | `TestClient_CapslockToggleProducesUppercase` | done |
| `neo2_level3_via_capslock_produces_ellipsis` | test | 3052 |  | `TestClient_Neo2Level3ViaCapslockProducesEllipsis` | done |
| `neo2_latched_mod5_preserved` | test | 3073 |  | `TestClient_Neo2LatchedMod5Preserved` | done |
| `german_layout_correct_key_resolution` | test | 3101 |  | `TestClient_GermanLayoutCorrectKeyResolution` | done |
| `space_works_with_cyrillic_layout_active` | test | 3119 |  | `TestClient_SpaceWorksWithCyrillicLayoutActive` | done |
| `macro_shift_bracket_produces_brace` | test | 3142 |  | `TestClient_MacroShiftBracketProducesBrace` | done |
| `sequential_key_events_do_not_corrupt_state` | test | 3164 |  | `TestClient_SequentialKeyEventsDoNotCorruptState` | done |
| `space_works_with_czech_layout_active` | test | 3203 |  | `TestClient_SpaceWorksWithCzechLayoutActive` | done |

## gpui_linux/src/linux/x11/clipboard.rs

Go target: `platform/linux/x11/clipboard.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ClipboardKind` | enum | 1182 |  | `ClipboardKind` | done |
| `Error` | enum | 1218 |  | `Error` | done |
| `Inner::handle_selection_request` | method | 671 |  | `clipboardInner.handleSelectionRequest` | dev:D-X11-CLIPBOARD |
| `Clipboard::set_image` | method | 1012 |  | `Clipboard.SetImage` | dev:D-X11-CLIPBOARD |

## gpui_linux/src/linux/x11/event.rs

Go target: `platform/linux/x11/event.go` (package `x11`), tests in `event_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_get_valuator_axis_index` | test | 130 |  | `TestEvent_GetValuatorAxisIndex` | done |

## gpui_linux/src/linux/x11/window.rs

Go target: `platform/linux/x11/window.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Callbacks` | struct | 245 |  | `Callbacks` | done |
| `X11WindowState` | struct | 259 |  | `X11WindowState` | done |
| `X11WindowState::new` | assoc-fn | 464 |  | `NewX11WindowState` | done |

## gpui_linux/src/linux/x11/xim_handler.rs

Go target: `platform/linux/x11/xim_handler.go` (package `x11`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `XimCallbackEvent` | enum | 6 |  | `XimCallbackEvent` | done |
| `XimHandler` | struct | 12 |  | `XimHandler` | done |
| `XimHandler::new` | assoc-fn | 21 |  | `NewXimHandler` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-X11-CLIPBOARD | Selection serving answers the advertised text/plain;charset=utf-8 / UTF-8 targets from UTF8_STRING data, uses the target as the property for ICCCM requests with property None, and sends data larger than the server's request limit (capped at 256 KiB) through INCR; reading caps INCR at 512 MiB and 30 s and reports INCR TARGETS replies as ATOM; SetImage offers the image under its own format's MIME atom instead of always image/png. A failed dialog setup also removes the dialog from its parent's children | `gpui_linux/src/linux/x11/clipboard.rs:671`, `:1012`, read_single; `x11/window.rs` X11WindowState::new | Upstream bugs found in review: refused pastes, BadLength connection loss, unbounded memory, mislabelled image bytes, parent left blocked |
