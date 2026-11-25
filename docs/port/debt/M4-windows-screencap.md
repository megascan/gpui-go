# M4-windows-screencap debt

- internal/win/wgc passes `SizeInt32` and `TimeSpan` by value as one uintptr, which is right for amd64 and arm64 only; windows/386 would need them split across two stack words.
- The live capture paths (WinRT delegates, dispatcher queue, CreateThread capture thread) were only cross-compiled on Linux; they are first exercised on the Windows runner (M4-ci-windows) and by the owner check.
- `CreateDispatcherQueueController`'s by-value `DispatcherQueueOptions` is passed per GOARCH (pointer on amd64, two registers on arm64, three words on 386); arm64 is unverified until an arm64 Windows runner or device runs the capture test.
