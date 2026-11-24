# M4-windows-platform handoffs

- M4-ci-windows: platform/platform_windows.go still returns errPlatformNotPorted; wire `currentPlatform` to `windows.NewWindowsPlatform(headless)` (import `github.com/megascan/gpui-go/platform/windows`) together with the scap registration from the M4-windows-screencap handoff. Run `GPUI_TEST_CLIPBOARD=1 go test ./platform/windows/...` on the runner (TestPlatform_Clipboard and the credential tests touch the real clipboard / Credential Manager; credentials use `zed-test-<pid>-<name>` session targets and delete them).
- M3-root-capture (or M4-ci-windows if it lands first): replace `scapScreenSources` in platform/windows/platform.go with the root scap glue.
- package-local stubs.go is deleted; `WindowCreationInfo` lives in platform.go. The platform passes its own drop target helper, main receiver and DirectX devices to `NewWindowsWindow` without cloning (the window AddRefs / clones them).
- M4-windows-directwrite handoff item resolved here: the three DirectWrite cursor tests now include upstream's `cursor()` / `take_until` assertions and their rows are `done`.
