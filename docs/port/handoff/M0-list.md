# M0-list handoff

- For M0-window-a11y (list.rs tests 2931-3883, elements_list_follow_test.go): upstream's shared test helpers at list.rs:3805-3883 (`draw_list`, `painted_item`, `item_readiness`, `list_offset`, `TestView`, `TestListView`) are already ported in elements_list_test.go as `listTestDrawList`, `listTestPaintedItem`, `listTestItemReadiness`, `listTestOffset`, `listTestView`, `listTestListView`, plus `listTestRun` (RunTest + AddEmptyWindow + release), `listTestDrawView`, `listTestNewView`, `listTestUpdateR`, `listTestFirst/Last/Find`, `listTestOpt`. Reuse them; do not redeclare. `state.0.borrow().is_provisional()` is `state.inner.isProvisional()`.
- `ListSizingBehavior` / `ListHorizontalSizingBehavior` moved from stubs.go to elements_list.go unchanged.
