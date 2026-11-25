# M1-wayland-client debt

- `WaylandClientState` is a plain pointer behind `clientCell`; upstream's `RefCell` borrow panics on re-entrant mutable borrows are not reproduced. `WaylandClientStatePtr.GetClient` / `TryGetClient` model only the `Weak` upgrade (nil once `WaylandConnection.Close` ran).
- D-WLFOREIGN: xdg-foreign exports live until the window drops instead of until the identifier drops; xdg-foreign is hand-defined in platform/linux/wayland/xdg_foreign.go (no wlgen output). `waylandExportHandle` has no automated test: exporting needs a toplevel surface, i.e. a window.
- Map iteration order (`Displays`, window ticks) is Go-random where upstream iterates a HashMap in its own unspecified order.
- `Cursor` loads themes through libwayland-cursor; the no-name load reads XCURSOR_THEME / XCURSOR_SIZE itself as wayland-cursor's `CursorTheme::load` does. libwayland-cursor loads every cursor of the theme up front, wayland-cursor loads lazily.
- `Attach` panics until M1-wayland-input and M1-wayland-window replace the package stubs.go (seat / data-device listeners, `NewClipboard`).
