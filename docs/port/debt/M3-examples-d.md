# Debt: M3-examples-d

- examples/input and examples/view_example release an unfocused ElementInputHandler themselves after `Window.HandleInput` (root leak, see handoff/M3-examples-d.md). Remove the workaround once HandleInput releases it.
- Custom elements (input `TextElement`, view_example `EditorText`) release their owned `Entity` at the end of `Paint`, the element's last use; there is no exported hook to release on frame-arena drop. An element laid out but never painted would leak its handle.
- view_example `standardActions` and TextArea's Enter listener capture a `WeakEntity[Editor]` instead of a strong clone (listener closures have no drop hook). The editor is kept alive by its owner, so behaviour matches upstream.
