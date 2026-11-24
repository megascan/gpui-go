# M4-windows-events debt

- Window-procedure handlers return an internal `msgResult` (value, ok) for upstream's `Option<isize>`.
- lParam pointers (MINMAXINFO, NCCALCSIZE_PARAMS, RECT, the WM_SETTINGCHANGE string, DirectXDevices) are read through `lparamPtr`, which reinterprets the integer via its address so `go vet`'s unsafeptr check passes.
- `HOVER_DEFAULT` and `ATTR_INPUT` are missing from internal/win/win32; events.go declares them locally (`hoverDefault`, `attrInput`). M4-ci-windows may move them.
- A negative `ImmGetCompositionStringW` result (IMM_ERROR) is treated as "no cursor / no attributes" instead of upstream's wrap-to-usize (which would attempt a huge allocation).
- keyboard.rs reads the layout name with golang.org/x/sys/windows/registry in place of the windows_registry crate.
