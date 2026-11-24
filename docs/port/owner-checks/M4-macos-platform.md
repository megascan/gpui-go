# M4-macos-platform owner checks

- YOU: on macOS, run an example that sets menus and a dock menu; check menu titles, key equivalents, the Window and Services menus, item enabling (validateMenuItem) and that a menu action fires.
- YOU: on macOS, open the open and save panels (PromptForPaths with files, directories, multiple; PromptForNewPath with a suggested name) and check the returned paths, including the Sequoia double-extension fix on macOS 15.
- YOU: on macOS from a signed app bundle, post a system notification with actions, click the body and an action, and check the response callback; dismiss it by tag.
- YOU: on macOS 12.3+, grant Screen Recording, list ScreenCaptureSources (labels, main display, resolution) and start a stream; check frames arrive and Stop ends the capture.
- YOU: on macOS, check keychain credentials write/read/delete, system sleep/wake callbacks, display-change callbacks, OpenURL, RevealPath, OpenWithSystem, Restart and RequestWindowing between headless and windowed.
- YOU: on macOS, run `GPUI_TEST_DISPLAY=1 go test -run TestGpuiPlatform ./platform/macos/` on the main thread once window.rs is ported.
