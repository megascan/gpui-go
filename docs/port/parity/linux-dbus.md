# Parity: linux-dbus

Scope: gpui_linux, D-Bus services (linux/power.rs, linux/system_notifications.rs, linux/xdg_desktop_portal.rs). Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_linux/src/linux/power.rs

Go target: `platform/linux/internal/dbus/power.go` (package `dbus`), tests in `power_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `maps_profiles_without_guessing_unknown_values` | test | 154 |  | `TestPower_MapsProfilesWithoutGuessingUnknownValues` | done |
| `monitors_profiles_and_service_restarts` | test | 171 |  | `TestPower_MonitorsProfilesAndServiceRestarts` | done |

## gpui_linux/src/linux/system_notifications.rs

Go target: `platform/linux/internal/dbus/system_notifications.go` (package `dbus`), tests in `system_notifications_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `caller_action_ids_do_not_collide_with_transport_action_ids` | test | 137 |  | `TestSystemNotifications_CallerActionIDsDoNotCollideWithTransportActionIDs` | done |

## gpui_linux/src/linux/xdg_desktop_portal.rs

Go target: `platform/linux/internal/dbus/xdg_desktop_portal.go` (package `dbus`), tests in `xdg_desktop_portal_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Event` | enum | 13 |  | `Event` | done |
| `XDPEventSource` | struct | 28 |  | `XDPEventSource` | done |
| `XDPEventSource::new` | assoc-fn | 34 |  | `NewXDPEventSource` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `text_scale_factor_is_clamped_to_a_usable_range` | test | 238 |  | `TestXdgDesktopPortal_TextScaleFactorIsClampedToAUsableRange` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-SEC-DBUSSENDER | Signals that complete a request or change state are accepted only from the unique name that owns the service (`OwnerPin`, resolved with `NameOwner` and refreshed once on a mismatch): notification `ActionInvoked` from the owner of `org.freedesktop.Notifications`, Secret Service prompt `Completed` from the owner of `org.freedesktop.secrets`, FileChooser and other portal `Request.Response` signals (`portalCall`, the ScreenCast portal) and Settings `SettingChanged` from the owner of `org.freedesktop.portal.Desktop`, logind `PrepareForSleep` from the owner of `org.freedesktop.login1`, power-profiles `PropertiesChanged` from the owner of that service and `NameOwnerChanged` only from the bus driver; the ScreenCast request token is 64 random bits and its Response wait is one overall deadline that unrelated signals do not shorten, so a same-bus peer cannot forge them (SEC-020, 021, 022, R2-9, R2-01, R2-02) | `gpui_linux/src/linux/system_notifications.rs`, `gpui_linux/src/linux/platform.rs` (keyring, file chooser via ashpd) | Signal match rules alone accept a signal from any peer on the session bus |
| D-SEC-SECRETPLAIN | When the Secret Service refuses the `dh-ietf1024-sha256-aes128-cbc-pkcs7` session, credential calls fail with `ErrPlainSessionRefused` instead of falling back to a `plain` session that would send the password unencrypted on the bus (SEC-023) | `gpui_linux/src/linux/platform.rs` (keyring, oo7 plain fallback) | A same-user bus monitor reads plain-session secrets |
