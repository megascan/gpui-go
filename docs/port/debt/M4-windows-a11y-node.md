# M4-windows-a11y-node debt

- a11y/windows builds only for windows/amd64 and windows/arm64: the float-argument trampolines (D-WINFLOAT, node_{amd64,arm64}.{go,s}) have no 386 version.
- The trampolines' register moves follow the x64 and ARM64 Windows ABIs (doubles in XMM1/XMM2 resp. D0/D1, `UiaPoint` as an HFA on arm64) and are exercised only by real UIA clients on CI/hardware; TestNode_PlatformNodeVtablesBuild only checks the vtables build.
- `IValueProvider::SetValue` decodes with windows.UTF16PtrToString, which replaces unpaired surrogates instead of panicking like upstream's `to_string().unwrap()`.
- Rows for text.rs, adapter.rs, subclass.rs, lib.rs and tests/ are `blocked:M4-windows-a11y`; stubs.go holds their placeholder types and a panicking newPlatformRange.
