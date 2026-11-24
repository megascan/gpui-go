# Owner checks

Things only a person can confirm (a device, a browser, an interactive dialog, a screen reader). Folded from docs/port/owner-checks/<stage>.md by the exit stages; keep each item until the owner confirms it.

Folded: M1-linux-platform, M1-linux-screencap, M1-wayland-window, M3-a11y-atspi-bus, M4-apple-text, M4-macos-appkit, M4-macos-platform, M4-web-webgl, M4-web-wgpu-js, M4-windows-a11y-node.

## M1-linux-platform

- YOU: in an example app on Wayland and X11, call `cx.PromptForPaths` and `cx.PromptForNewPath`; check the xdg-desktop-portal file dialog opens, titled "Open File" / "Open Folder" / "Save File", and the chosen path comes back.

## M1-linux-screencap

- YOU: On a Wayland session with xdg-desktop-portal running, start a screen share from an example that uses screen capture (after M3-root-capture), pick a monitor in the portal dialog and confirm frames arrive; cancel the dialog once and confirm the error "User cancelled the interaction".
- YOU: On an X11 session, start a screen capture and confirm the captured frames show the screen with the cursor drawn in.

## M1-wayland-window

- YOU: on a KDE Plasma (< 6.7, kde-blur) and a compositor with ext-background-effect, open an example with `WindowBackgroundAppearanceBlurred` and check the blur follows the rounded client-decoration corners.
- YOU: on a layer-shell compositor (Hyprland / Sway), open a layer-shell window and an anchored popup with grab, check placement, dismissal on outside click and `xdg_popup.reposition` on resize.

## M3-a11y-atspi-bus

- YOU: with Orca or Accerciser running on a Linux desktop, start examples/a11y (after M3-exit wires the adapter) and confirm the window and its widgets appear in the tree, names update, and activating a button through the screen reader triggers it.

## M4-apple-text

- YOU: on a Mac, check that text in any example (e.g. hello_world) renders crisp with correct glyph shapes, emoji in colour, and ligatures in Lilex, after M4-macos-platform wires AppleTextSystem in.

## M4-macos-appkit

- YOU: on a Mac, run an example and check key handling for the layouts listed in events.rs parse_keystroke (Armenian s, Dvorak+QWERTY s, Ukrainian+QWERTY cmd-shift-s, Czech 7, Norwegian cmd-shift-7, Russian cmd-shift-7, German QWERTZ ;) gives the keystrokes in that table, and that cmd-[ on ABC-QWERTZ matches a cmd-ö binding.

## M4-macos-platform

- YOU: on macOS, run an example that sets menus and a dock menu; check menu titles, key equivalents, the Window and Services menus, item enabling (validateMenuItem) and that a menu action fires.
- YOU: on macOS, open the open and save panels (PromptForPaths with files, directories, multiple; PromptForNewPath with a suggested name) and check the returned paths, including the Sequoia double-extension fix on macOS 15.
- YOU: on macOS from a signed app bundle, post a system notification with actions, click the body and an action, and check the response callback; dismiss it by tag.
- YOU: on macOS 12.3+, grant Screen Recording, list ScreenCaptureSources (labels, main display, resolution) and start a stream; check frames arrive and Stop ends the capture.
- YOU: on macOS, check keychain credentials write/read/delete, system sleep/wake callbacks, display-change callbacks, OpenURL, RevealPath, OpenWithSystem, Restart and RequestWindowing between headless and windowed.
- YOU: on macOS, run `GPUI_TEST_DISPLAY=1 go test -run TestGpuiPlatform ./platform/macos/` on the main thread once window.rs is ported.

## M4-web-webgl

- YOU: open the web example in a browser with WebGPU disabled (Chrome `--disable-features=WebGPU`, or Firefox without `dom.webgpu.enabled`) or with the backend preference set to WebGl; check that quads, shadows, paths, underlines, text (mono sprites), images (poly sprites) draw right side up and match the WebGPU build, and that resizing the window redraws at the new size.
- YOU: in the same WebGL run, force a context loss (`WEBGL_lose_context` extension's `loseContext()` from the devtools console) and check the console logs "wgpu device lost" once.

## M4-web-wgpu-js

- YOU: in Chrome (WebGPU on) open a web example once M4-web-platform lands; check the console logs "Browser graphics initialized: requested=Auto, selected=BrowserWebGpu" and the canvas draws.

## M4-windows-a11y-node

- YOU: on Windows x64, once M4-windows-a11y wires the adapter, hover a gpui example with Inspect.exe or Accessibility Insights and check the hovered element is found (IRawElementProviderFragmentRoot::ElementProviderFromPoint through the D-WINFLOAT trampoline).
- YOU: on Windows x64, set a slider's value through Inspect.exe's RangeValue pattern (IRangeValueProvider::SetValue) and check the example receives the new value.
- YOU: on an ARM64 Windows device (CI has no arm64 runner), repeat both checks and also TextPattern.RangeFromPoint on a text input (UiaPoint is passed in D0/D1 there).

## M0-exit

- YOU: fix GitHub Actions billing / spending limit for megascan/gpui-go (see TECH_DEBT CI-BILLING), then confirm the ci workflow on the draft PR feat/port -> dev is green on ubuntu, macos-14 and windows.
