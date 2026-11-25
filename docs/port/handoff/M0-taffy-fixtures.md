# M0-taffy-fixtures handoff

- Fixture count is 6504 (tag v0.15.0), not 6505 as STAGES.md says; per-category counts are in parity/ext-taffy.md `## tests/xml fixtures`.
- Runner: `TestXml/<category>/<file>` in internal/taffy/xml_test.go; enable categories by appending to `xmlFixturesEnabled` in fixtures_enabled_test.go. Disabled categories skip.
- A fixture that panics with a stub message (`taffy: unimplemented ...`) skips as blocked; any other panic or parse error fails. 4 `leaf` fixtures (leaf_overflow_scrollbars_overridden_by_available_space__*) lay out a flex container and skip until M0-taffy-flexbox replaces the stub; that stage flips the `tests/xml/leaf` row to done.
- 136 `flex` fixtures use `flex-wrap: balance` (cargo feature flexbox_balance, not enabled by gpui) and skip as n/a:D-TAFFY-FEATURE; `flex-line-count` is ignored as upstream does without the feature.
- The `parse` feature's FromCss impls are ported test-only in xml_css_test.go (D-TAFFY-XMLPARSE); every one of the 6504 fixtures' styles parses with it (checked once ad hoc).
- tests/common is tests_common_test.go (one upstream file, one Go file), shared by the hand_written stages.
- Generated rows resolve resolved-rows/columns as None (detailed_layout_info not enabled), same as upstream without the feature.
