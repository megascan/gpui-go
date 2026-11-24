# Debt: M3-examples-c

- examples/native_view, examples/native_view_appkit_field: the background reporter keeps its sender in a Go map keyed by the reporter object instead of an objc ivar; entries are never removed (upstream leaks the boxed sender the same way).
- examples/gif_viewer, image_gallery, image_loading: upstream's `env_logger::init()` has no Go counterpart; the default slog handler prints the logs.
- examples/window_positioning, layer_shell, toggle_headless: see docs/port/handoff/M3-examples-c.md (Wayland negative-size abort, shoot cannot capture layer surfaces or headless starts).
