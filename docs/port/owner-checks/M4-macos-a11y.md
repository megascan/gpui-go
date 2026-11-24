# Owner checks: M4-macos-a11y

- YOU: on macOS, run any GPUI app window (the hello_world example once it exists) with VoiceOver on (Cmd+F5); check that VoiceOver reads the window's elements, follows focus changes, and that VO+Space activates a button.
- YOU: on macOS, open Accessibility Inspector (Xcode > Open Developer Tool) on a GPUI window; check that roles, titles, values and frames are correct and that the frames line up with what is on screen on both Retina and non-Retina displays.
