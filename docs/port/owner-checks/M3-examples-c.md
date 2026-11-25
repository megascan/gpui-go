# Owner checks: M3-examples-c

- YOU: on macOS, `go run ./examples/native_view`: the WKWebView page shows next to the GPUI sidebar; cmd-shift-p opens the palette over the page edge, escape returns focus to the page field, cmd-k stays with the page, dragging the divider resizes the sidebar and the catch-up strip takes the page background color.
- YOU: on macOS, `go run ./examples/native_view_appkit_field`: the NSTextView sits in the top right corner, typing goes into it, the top line names the view with keyboard input, and the console prints "AppKit field took keyboard input: true" then "AppKit field kept keyboard input: true".
- YOU: on macOS, `go run ./examples/set_menus`: the menu bar shows set_menus with Services, a disabled item and submenu, a checked "List Mode", a Mode submenu whose check moves when toggled, and Quit (prints "Gracefully quitting the application...").
- YOU: on macOS and Windows, `go run ./examples/window`: every button opens its window kind (Popup, Floating, Dialog with nested dialog, custom titlebar, invisible, unmovable, unresizable, unminimizable), Hide Application restores after 3 s, Prompt and the Chinese Prompt show native dialogs and print the clicked button.
- YOU: on macOS, `go run ./examples/window_movable`: the two NOT movable windows cannot be dragged by the titlebar, the movable ones can.
- YOU: on Linux outside dbus-run-session (with a notification daemon), `go run ./examples/system_notifications`: "Show or replace" posts and replaces one notification with Open and Snooze, clicking it updates the status line, Dismiss removes it.
- YOU: on Linux, `go run ./examples/move_entity_between_windows`: clicking "Move me to a new window" opens a new window, closes the old one, and the tick lines print the new window id.
