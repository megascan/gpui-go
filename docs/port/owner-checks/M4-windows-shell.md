# M4-windows-shell owner checks

- YOU: on Windows, run an example that posts a system notification (after App::set_app_identity): the toast appears silently with title, body and action buttons; clicking a button reports its action id, clicking the body reports none; posting the same tag replaces the old toast, dismiss hides it.
- YOU: on Windows, an app that sets a dock menu and recent workspaces shows them in the taskbar jump list ("Recent Folders" category and Tasks); removing a recent entry from the jump list is reported back on the next update.
- YOU: on a Windows precision touchpad, two-finger pan scrolls with Started/Moved/Ended phases and inertia, pinch zooms; a new gesture during inertia ends the old one.
- YOU: on Windows, open a file dialog / prompt, then close its owner window: the dialog closes and the window is destroyed only after it exits; a prompt without a Cancel button ignores Esc and the X button.
- YOU: on Windows, copy text with metadata, an image (PNG, JPEG, GIF, SVG) and files between gpui and another app (Notepad, Paint, Explorer): CRLF/LF round-trips, metadata survives a gpui-to-gpui copy, a pasted screenshot (CF_DIB) arrives as BMP.
