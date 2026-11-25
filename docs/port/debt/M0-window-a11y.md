# M0-window-a11y debt

- window_a11y_debug.go writes the debug JSON with a hand-rolled ordered writer (serde_json `preserve_order` + `to_string_pretty`), including ryu-style float text (`800.0`, `1e16`); ryu's exact shortest-digit choice for f32 is approximated with strconv's shortest f32 formatting.
- NativeView's window list holds weak pointers; a released view is detected through `nativeViewState.dropped` because weak pointers only clear after GC.
- window_native_view_test.go's input handler holds a WeakEntity of the test view instead of upstream's strong Entity (a strong handle inside a per-frame input handler has no release point under strict leak detection).
