# M4-macos-appkit debt

- events.go, keyboard.go and internal/apple/appkit were only cross-compiled (darwin arm64/amd64 vet + test -c); no test has run on a Mac. First macOS CI run must exercise TestEvents_*, TestKeyboard_*, TestGpuiMacos_* and TestAppkit_*.
- appkit wrapper selectors and return types (bool for BOOL, float32 for `pressure`, uint16 for `keyCode`) are unverified at runtime until macOS CI.
