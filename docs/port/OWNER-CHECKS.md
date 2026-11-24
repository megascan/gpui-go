# Owner checks

Things only a person can confirm (a device, a browser, an interactive dialog, a screen reader). Folded from docs/port/owner-checks/<stage>.md by the exit stages; keep each item until the owner confirms it.

Folded: M1-linux-platform, M1-linux-screencap, M1-wayland-window, M3-a11y-atspi-bus, M4-apple-text, M4-macos-appkit, M4-macos-platform, M4-web-webgl, M4-web-wgpu-js, M4-windows-a11y-node, M4-ios-app, M4-macos-a11y, M4-web-example, M4-windows-a11y, M4-windows-platform, M4-windows-screencap, M4-windows-shell, M1-exit, M3-examples-b, M3-examples-c, M3-exit, M4-ios-window, M4-ci-macos.

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

## M4-ios-app

- YOU: on an iOS simulator or device (after M4-ios-window), launch the iOS example; check the launch callback runs once the scene connects, background/foreground/active lifecycle callbacks fire, and the display link pauses in background.
- YOU: on iOS, open an https URL via `OpenURL` (Safari view controller appears, `Activate` dismisses it) and a custom-scheme URL; open the app from a universal link and a URL scheme and check `OnOpenURLs` gets the full URL with query and fragment.
- YOU: on iOS, check clipboard read/write, keychain credentials, `Displays`/`PrimaryDisplay` bounds and refresh interval, and that connecting an external display fires `OnDisplaysChanged`.

## M4-macos-a11y

- YOU: on macOS, run any GPUI app window (the hello_world example once it exists) with VoiceOver on (Cmd+F5); check that VoiceOver reads the window's elements, follows focus changes, and that VO+Space activates a button.
- YOU: on macOS, open Accessibility Inspector (Xcode > Open Developer Tool) on a GPUI window; check that roles, titles, values and frames are correct and that the frames line up with what is on screen on both Retina and non-Retina displays.

## M4-web-example

- YOU: serve examples/web_hello_web as the gallery root (index.html, gallery.js, wasm_exec.js copied from `$(go env GOROOT)/lib/wasm/`, gpui-web-examples.wasm built with `GOOS=js GOARCH=wasm go build -o <dir>/gpui-web-examples.wasm ./examples/web_hello_web`), open `/prime-sieve` in Chrome with WebGPU, check the window renders the prime sieve UI and "Count Primes" runs to completion with the history line.
- YOU: same page with `?backend=webgl` (WebGL2 fallback), check it renders and runs.
- YOU: open `/dynamic-fonts` with the network throttled or blocked, check the Devanagari sample falls back and the status line changes to "Font loaded" after the download, and "Font loading failed" when blocked.
- YOU: run `node test_canvas_fallback.mjs http://127.0.0.1:8080` (needs Chrome, not run here), check it passes.
- YOU: run `node test_http_client.mjs http://127.0.0.1:8080` (needs Chrome, not run here), check it passes.

## M4-windows-a11y

- YOU: on Windows x64, run a gpui example with a text input under Narrator (or Accessibility Insights' Text pattern explorer) and check reading by character / word / line, caret position and selection (ITextProvider / ITextRangeProvider in a11y/windows/text.go).
- YOU: on Windows x64, open a gpui example, start Inspect.exe before and after the window appears, and check the tree, focus events and live-region announcements in Narrator (Adapter placeholder/active paths, QueuedEvents.Raise).
- YOU: on an ARM64 Windows device, repeat the above plus TestSimple_ElementFromPointAndRangeValueSetValue's checks by hand (it skips on arm64).

## M4-windows-platform

- YOU: on Windows, run an example and check the open / save file dialogs, Reveal in Explorer, Open URL, jump list tasks and battery-saver power mode notifications behave like Zed.

## M4-windows-screencap

- YOU: on Windows 10 1903+ or 11, run a gpui example that shares the screen (or `GPUI_TEST_DISPLAY=1 go test -run 'TestCapture_PrimaryMonitor|TestCapturer_MainDisplay' ./internal/win/wgc/ ./internal/win/screencap/`) and check that frames arrive, the yellow capture border appears and disappears on stop, and the cursor shows only when requested.
- YOU: on a multi-monitor Windows machine, check that the display list shows each monitor's friendly name and that capturing a window target works.

## M4-windows-shell

- YOU: on Windows, run an example that posts a system notification (after App::set_app_identity): the toast appears silently with title, body and action buttons; clicking a button reports its action id, clicking the body reports none; posting the same tag replaces the old toast, dismiss hides it.
- YOU: on Windows, an app that sets a dock menu and recent workspaces shows them in the taskbar jump list ("Recent Folders" category and Tasks); removing a recent entry from the jump list is reported back on the next update.
- YOU: on a Windows precision touchpad, two-finger pan scrolls with Started/Moved/Ended phases and inertia, pinch zooms; a new gesture during inertia ends the old one.
- YOU: on Windows, open a file dialog / prompt, then close its owner window: the dialog closes and the window is destroyed only after it exits; a prompt without a Cancel button ignores Esc and the X button.
- YOU: on Windows, copy text with metadata, an image (PNG, JPEG, GIF, SVG) and files between gpui and another app (Notepad, Paint, Explorer): CRLF/LF round-trips, metadata survives a gpui-to-gpui copy, a pasted screenshot (CF_DIB) arrives as BMP.

## M1-exit

- YOU: on X11 / XWayland, open examples/hello_world (`go run ./examples/hello_world` with WAYLAND_DISPLAY unset) and compare with Zed upstream on the same machine: the area outside the 500px div is transparent here (Vulkan surface on a 32-bit ARGB visual offers no Opaque alpha mode, so the Inherit fallback is picked as upstream does); confirm upstream shows the same.

## M3-examples-b

- YOU: mouse_pressure on a macOS Force Touch trackpad: `go run ./examples/mouse_pressure`, press harder; "Pressure stage" goes Zero -> Normal -> Force and the amount changes.
- YOU: focus_visible / tab_stop interactively: Tab and Shift-Tab move focus (green focus-visible border only on keyboard, yellow focus border on click); tab_stop cycles 1,2,2,3,Button 1,Button 2,[6,1..3],[7,1..3].
- YOU: drag_drop: drag an item onto "Drop items here"; the preview follows the cursor and the target takes the item's color.
- YOU: painting: drag to draw, Shift for straight lines, Dashed/Scribble/Grid/Clear buttons.
- YOU: a11y with a screen reader (Orca on Linux, VoiceOver, Narrator): heading, spin button increment/decrement, switch, list of three tasks.

## M3-examples-c

- YOU: on macOS, `go run ./examples/native_view`: the WKWebView page shows next to the GPUI sidebar; cmd-shift-p opens the palette over the page edge, escape returns focus to the page field, cmd-k stays with the page, dragging the divider resizes the sidebar and the catch-up strip takes the page background color.
- YOU: on macOS, `go run ./examples/native_view_appkit_field`: the NSTextView sits in the top right corner, typing goes into it, the top line names the view with keyboard input, and the console prints "AppKit field took keyboard input: true" then "AppKit field kept keyboard input: true".
- YOU: on macOS, `go run ./examples/set_menus`: the menu bar shows set_menus with Services, a disabled item and submenu, a checked "List Mode", a Mode submenu whose check moves when toggled, and Quit (prints "Gracefully quitting the application...").
- YOU: on macOS and Windows, `go run ./examples/window`: every button opens its window kind (Popup, Floating, Dialog with nested dialog, custom titlebar, invisible, unmovable, unresizable, unminimizable), Hide Application restores after 3 s, Prompt and the Chinese Prompt show native dialogs and print the clicked button.
- YOU: on macOS, `go run ./examples/window_movable`: the two NOT movable windows cannot be dragged by the titlebar, the movable ones can.
- YOU: on Linux outside dbus-run-session (with a notification daemon), `go run ./examples/system_notifications`: "Show or replace" posts and replaces one notification with Open and Snooze, clicking it updates the status line, Dismiss removes it.
- YOU: on Linux, `go run ./examples/move_entity_between_windows`: clicking "Move me to a new window" opens a new window, closes the old one, and the tick lines print the new window id.

## M3-exit

- YOU: run `go run ./examples/a11y` on Wayland and on X11 (`WAYLAND_DISPLAY= go run ./examples/a11y`) with Orca enabled; check Orca reads the window's buttons and labels and that focus changes are announced.

## M4-ios-window

- YOU: on macOS with Xcode, run `examples/ios_ios/build-simulator.sh`, install the .app with `xcrun simctl install booted target/gpui-ios-example/GPUIIosExample.app` and launch it; check the dark screen with "GPUI on iOS", that tapping "Taps: N" increments, and that the status bar content is light.
- YOU: in the simulator app, rotate the device and check the layout resizes without stretching (drawable size and scale follow), safe-area insets are respected, and toggling light/dark appearance fires the appearance callback.
- YOU: on iOS with a focused text input, check the software keyboard appears, typing/deleting/emoji/marked (Japanese) text work, a downward swipe of at least 24pt dismisses the keyboard, and trackpad/mouse-wheel scrolling scrolls.
- YOU: on iOS, run the native text-input harness (`go test -c -tags ios ./platform/ios` built for the simulator with CGO_ENABLED=1, then run it in the simulator); it must print "UIKit text input native protocol smoke test passed".
- YOU: on an iOS 26 device or simulator, check `Insets().CornerRadii` reports the display's corner radii.

## M4-ci-macos

- YOU: on a Mac, `go run ./examples/hello_world`: the window opens through MacPlatform, the text renders crisp and the 500px div matches upstream.
- YOU: on a Mac, type in an example with a text input using the Japanese / Pinyin IME: marked text, candidate window placement and commit work.
- YOU: on a Mac, `go run ./examples/native_view` and `./examples/native_view_appkit_field`: native views are placed, sized and receive input as described under M3-examples-c.
- YOU: on a Mac with VoiceOver on, `go run ./examples/a11y`: VoiceOver reads the window, buttons and labels and announces focus changes.
- YOU: on a Mac, `GPUI_TEST_DISPLAY=1 go test ./platform/macos/ ./platform/apple/`: window, Metal headless render and shader tests pass on real hardware.
- YOU: GitHub Actions is blocked on the account (billing / spending limit; macos-14 also reported "configuration 'macos-14-arm64-us-xl' is not supported"). Fix billing, then re-run CI on feat/port so the macos-14 job runs.

## M4-ci-windows

- YOU: on Windows, `go run ./examples/hello_world`: the window opens through WindowsPlatform, text renders crisp through DirectWrite and the 500px div matches upstream.
- YOU: on Windows, type in an example with a text input using the Japanese / Pinyin IME: composition text, candidate window placement and commit work.
- YOU: on Windows, move a running example between monitors with different scale factors (or change display scaling in Settings): the window rescales without blur or wrong size.
- YOU: on Windows with Narrator on, `go run ./examples/a11y`: Narrator reads the window, buttons and labels and announces focus changes.
- YOU: on Windows, `GPUI_TEST_DISPLAY=1 go test ./platform/windows/ ./internal/win/...`: window, DirectX/HLSL and screen-capture tests pass on real hardware.
- YOU: GitHub Actions is blocked on the account (billing / spending limit). Fix billing, then re-run CI on feat/port so the windows-latest job runs; the Windows packages still need to be committed and pushed by the next exit/CI stage.
