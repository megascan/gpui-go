# M4-windows-a11y owner checks

- YOU: on Windows x64, run a gpui example with a text input under Narrator (or Accessibility Insights' Text pattern explorer) and check reading by character / word / line, caret position and selection (ITextProvider / ITextRangeProvider in a11y/windows/text.go).
- YOU: on Windows x64, open a gpui example, start Inspect.exe before and after the window appears, and check the tree, focus events and live-region announcements in Narrator (Adapter placeholder/active paths, QueuedEvents.Raise).
- YOU: on an ARM64 Windows device, repeat the above plus TestSimple_ElementFromPointAndRangeValueSetValue's checks by hand (it skips on arm64).
