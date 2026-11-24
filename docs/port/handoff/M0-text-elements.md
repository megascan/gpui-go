# M0-text-elements handoff

SharedString now implements IntoElement (elements_text.go), so these `blocked:M0-text-elements` items can run; their files are owned by other stages:

- elements_div_test.go: un-skip `TestDiv_LongPressShowsTooltipUntilTouchEnds`, `TestDiv_TooltipRespectsCustomShowDelay`, `TestDiv_TooltipIsReleasedWhenItsOwnerDisappears`, `TestDiv_TooltipHidesAfterMouseLeavesOrigin`, `TestDiv_MouseDownOutIsSuppressedWhileWindowPromptIsActive` and set rows core-elements.md:305-310 to `done` (M0-window-tests owns elements_div*.go).
- docs/port/parity/windows.md:35-37 (`TestDirectWrite_*ShapedLineCursor`, platform/windows/direct_write_test.go): ShapedLine.Cursor / TakeUntil exist now; port them in the Windows text stage or M4-ci-windows.

API notes: `TextMacro(s)` is upstream's `text!` (id hashed from caller file/line; column is 0, D-SRCLOC, so two calls on one line share an id). Highlight / font-override parameters are `iter.Seq2[Range, HighlightStyle]` / `iter.Seq2[Range, SharedString]` so `CombineHighlights` output plugs in directly. `TextLayout.IndexForPosition` returns `(index, ok)` with ok=false for upstream's `Err(index)`. `ShapedLine.PaintWithUnderlineHandler` takes an `UnderlineHandler`.
