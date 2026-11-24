# M0-line-wrapper handoff

- `TestTextSystemLineWrapperPoolTests_PooledLineWrappersDoNotRetainTextSystem` (text_system_test.go, owned by M0-text-system / exit stages) still skips with `blocked:M0-line-wrapper`. LineWrapper.WrapLine and LineFragmentText now exist; the test can be ported (weak check via `weak.Make` + runtime.GC) and its core-text.md row set to done.
- IndentAdjustment (data enum) is a comparable struct `{Kind, ExtraColumns}` with `IndentAdjustmentNoIndent`, `IndentAdjustmentSameIndent`, `IndentAdjustmentExtraColumns(n)`, `DefaultIndentAdjustment()`. LineFragment is a struct built by `LineFragmentText` / `LineFragmentElement`. WrapLine returns a single-use iter.Seq[Boundary].
- Tests build the wrapper on `NewTextSystem(NewNoopTextSystem())`, which is what upstream's TestAppContext::build uses (TestPlatform::new -> NoopTextSystem), so they run now instead of waiting for TestAppContext.
