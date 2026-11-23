# Parity: linux-dbus

Scope: gpui_linux, D-Bus services (linux/power.rs, linux/system_notifications.rs, linux/xdg_desktop_portal.rs). Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux/power.rs

Go target: `platform/linux/internal/dbus/power.go` (package `dbus`), tests in `power_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `maps_profiles_without_guessing_unknown_values` | test | 154 |  | `TestPower_MapsProfilesWithoutGuessingUnknownValues` | todo |
| `monitors_profiles_and_service_restarts` | test | 171 |  | `TestPower_MonitorsProfilesAndServiceRestarts` | todo |

## gpui_linux/src/linux/system_notifications.rs

Go target: `platform/linux/internal/dbus/system_notifications.go` (package `dbus`), tests in `system_notifications_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `caller_action_ids_do_not_collide_with_transport_action_ids` | test | 137 |  | `TestSystemNotifications_CallerActionIDsDoNotCollideWithTransportActionIDs` | todo |

## gpui_linux/src/linux/xdg_desktop_portal.rs

Go target: `platform/linux/internal/dbus/xdg_desktop_portal.go` (package `dbus`), tests in `xdg_desktop_portal_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Event` | enum | 13 |  | `Event` | todo |
| `XDPEventSource` | struct | 28 |  | `XDPEventSource` | todo |
| `XDPEventSource::new` | assoc-fn | 34 |  | `NewXDPEventSource` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text_scale_factor_is_clamped_to_a_usable_range` | test | 238 |  | `TestXdgDesktopPortal_TextScaleFactorIsClampedToAUsableRange` | todo |

