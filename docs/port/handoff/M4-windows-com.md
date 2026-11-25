# M4-windows-com handoff

## internal/win/win32 has no owner after this stage

- For: M4-windows-directwrite, M4-windows-d3d-atlas, M4-windows-directx-renderer, M4-windows-events, M4-windows-shell, M4-windows-window, M4-windows-platform, M4-windows-a11y-node, M4-windows-a11y (and M4-ci-windows, which has fix authority).
- internal/win/win32 declares the user32 / gdi32 / dwmapi / shcore / imm32 / ole32 / shell32 / kernel32 / advapi32 / comctl32 procedures gpui_windows calls (taken from every Win32 call in `_ref/gpui_windows/src`), the structures they use (sizes pinned in types_test.go) and the constants upstream names (consts.go). Virtual-key codes other than those upstream names are not declared.
- Later stages do not own internal/win/win32. A procedure, structure or constant that is missing there goes into the stage's own package (for platform/windows a package-local file, e.g. `win32_extra.go`), or is reported here for M4-ci-windows to move.
- COM interfaces: embed com.IUnknown (or com.IInspectable), call methods with com.Call / com.CallHR and the slot index (com.IUnknownSlots + n), cast with com.Cast[T]. Go-implemented interfaces (drop target, DirectWrite renderer and loaders, UI Automation providers) use com.NewVtable through a package-level com.LazyVtable and com.NewObject; com.ImplOf recovers the Go implementation inside each thunk.
