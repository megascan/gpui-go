# Parity: linux-platform

Scope: gpui_linux, Linux entry points (gpui_linux.rs, linux.rs, linux/display_connection.rs). Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux.rs

Go target: `platform/linux/linux.go` (package `linux`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `current_platform` | fn | 40 |  | `CurrentPlatform` | done |
| `linux_platform` | fn | 52 |  | `LinuxPlatform` | done |
| `linux_platform` returns `Rc<dyn Platform>` of `WindowingModes::all()` | const | 44 |  | `windowingModesAll` | done |

## gpui_linux/src/linux/display_connection.rs

Go target: `platform/linux/display_connection.go` (package `linux`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `select_backend` | fn | 23 |  | `selectBackend` | done |
| `Backend` | enum | 42 |  | `backend` | done |
| `DisplayConnection` | enum | 66 |  | `displayConnection` | dev:D-LINUX-CLIENT |
| `dispatch!` | macro | 76 |  | `displayConnection` (nil `connected` is headless) | dev:D-LINUX-CLIENT |
| `DisplayConnection::is_headless` | method | 99 |  | `(*displayConnection).isHeadless` | done |
| `DisplayConnection::compositor_name` | method | 104 |  | `(*displayConnection).compositorName` | done |
| `DisplayConnection::has_windows` | method | 109 |  | `(*displayConnection).hasWindows` | done |
| `DisplayConnection::graphical_environment` | method | 114 |  | `(*displayConnection).graphicalEnvironment` | done |
| `DisplayConnection::keyboard_layout` | method | 118 |  | `(*displayConnection).keyboardLayout` | done |
| `DisplayConnection::displays` | method | 126 |  | `(*displayConnection).displays` | done |
| `DisplayConnection::primary_display` | method | 130 |  | `(*displayConnection).primaryDisplay` | done |
| `DisplayConnection::open_window` | method | 138 |  | `(*displayConnection).openWindow` | done |
| `DisplayConnection::active_window` | method | 150 |  | `(*displayConnection).activeWindow` | done |
| `DisplayConnection::window_stack` | method | 154 |  | `(*displayConnection).windowStack` | done |
| `DisplayConnection::set_cursor_style` | method | 158 |  | `(*displayConnection).setCursorStyle` | done |
| `DisplayConnection::hide_cursor_until_mouse_moves` | method | 162 |  | `(*displayConnection).hideCursorUntilMouseMoves` | done |
| `DisplayConnection::is_cursor_visible` | method | 166 |  | `(*displayConnection).isCursorVisible` | done |
| `DisplayConnection::open_uri` | method | 170 |  | `(*displayConnection).openURI` | done |
| `DisplayConnection::reveal_path` | method | 174 |  | `(*displayConnection).revealPath` | done |
| `DisplayConnection::write_to_primary` | method | 178 |  | `(*displayConnection).writeToPrimary` | done |
| `DisplayConnection::write_to_clipboard` | method | 182 |  | `(*displayConnection).writeToClipboard` | done |
| `DisplayConnection::read_from_primary` | method | 186 |  | `(*displayConnection).readFromPrimary` | done |
| `DisplayConnection::read_from_clipboard` | method | 190 |  | `(*displayConnection).readFromClipboard` | done |
| `DisplayConnection::window_identifier` | method | 194 |  | `(*displayConnection).windowIdentifier` | done |
| `DisplayConnection::is_screen_capture_supported` | method | 206 | feat:screen-capture | `(*displayConnection).isScreenCaptureSupported` | done |
| `DisplayConnection::screen_capture_sources` | method | 210 | feat:screen-capture | `(*displayConnection).screenCaptureSources` | done |
| drop of a `DisplayConnection` | impl | 66 |  | `(*displayConnection).close` | done |
