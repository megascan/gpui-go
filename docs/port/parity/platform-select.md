# Parity: platform-select

Scope: gpui_platform. Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_platform/benches/inspector_render.rs

Go target: `platform/bench_inspector_render_test.go` (package `platform_test`), tests in `bench_inspector_render_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `bare_grid` | gpui::bench(inputs = closed_inspector_inputs("nodes", [100, 1_000, 10_000]), input_name = "case", group = "closed_inspector_grid", fps = 120) | 67 |  | `BenchmarkInspectorRender_BareGrid` | todo |
| `panel` | gpui::bench(inputs = closed_inspector_inputs("rows", [50, 500]), input_name = "case", group = "closed_inspector_panel", fps = 120) | 85 |  | `BenchmarkInspectorRender_Panel` | todo |

## gpui_platform/benches/primitives.rs

Go target: `platform/bench_primitives_test.go` (package `platform_test`), tests in `bench_primitives_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `paint_rectangles` | gpui::bench(inputs = cases(), input_name = "case", group = "Primitives/paint", fps = 120) | 410 |  | `BenchmarkPrimitives_PaintRectangles` | todo |

## gpui_platform/benches/randomized_element_tree.rs

Go target: `platform/bench_randomized_element_tree_test.go` (package `platform_test`), tests in `bench_randomized_element_tree_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/full refresh", fps = 120) | 393 |  | `BenchmarkRandomizedElementTree_FullRefresh` | todo |
| `pathless_full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "PathlessTree/full refresh", fps = 120) | 409 |  | `BenchmarkRandomizedElementTree_PathlessFullRefresh` | todo |
| `pathless_element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "PathlessTree/element style", fps = 120) | 418 |  | `BenchmarkRandomizedElementTree_PathlessElementStyle` | todo |
| `unchanged` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/unchanged", fps = 120) | 427 |  | `BenchmarkRandomizedElementTree_Unchanged` | todo |
| `element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/element style", fps = 120) | 452 |  | `BenchmarkRandomizedElementTree_ElementStyle` | todo |
| `leaf_bounds` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/leaf bounds", fps = 120) | 460 |  | `BenchmarkRandomizedElementTree_LeafBounds` | todo |
| `root_layout` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/root layout", fps = 120) | 469 |  | `BenchmarkRandomizedElementTree_RootLayout` | todo |
| `insert_remove` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/insert-remove", fps = 120) | 479 |  | `BenchmarkRandomizedElementTree_InsertRemove` | todo |
| `reorder` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/reorder", fps = 120) | 496 |  | `BenchmarkRandomizedElementTree_Reorder` | todo |
| `hover` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/hover", fps = 120) | 562 |  | `BenchmarkRandomizedElementTree_Hover` | todo |
| `mixed` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/mixed", fps = 120) | 774 |  | `BenchmarkRandomizedElementTree_Mixed` | todo |
| `cached_full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/full refresh", fps = 120) | 788 |  | `BenchmarkRandomizedElementTree_CachedFullRefresh` | todo |
| `cached_element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/element style", fps = 120) | 796 |  | `BenchmarkRandomizedElementTree_CachedElementStyle` | todo |
| `cached_leaf_bounds` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/leaf bounds", fps = 120) | 802 |  | `BenchmarkRandomizedElementTree_CachedLeafBounds` | todo |
| `cached_insert_remove` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/insert-remove", fps = 120) | 810 |  | `BenchmarkRandomizedElementTree_CachedInsertRemove` | todo |
| `cached_hover` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/hover", fps = 120) | 825 |  | `BenchmarkRandomizedElementTree_CachedHover` | todo |
| `cached_mixed` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/mixed", fps = 120) | 832 |  | `BenchmarkRandomizedElementTree_CachedMixed` | todo |
| `cached_reorder` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/reorder", fps = 120) | 838 |  | `BenchmarkRandomizedElementTree_CachedReorder` | todo |
| `changing_share` | gpui::bench(inputs = changing_share_inputs(), input_name = "tree", iterations = 1, group = "RandomizedTree/changing share", fps = 120) | 907 |  | `BenchmarkRandomizedElementTree_ChangingShare` | todo |
| `text_panel_updates` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/two updates", fps = 120) | 1016 |  | `BenchmarkRandomizedElementTree_TextPanelUpdates` | todo |
| `installed_font_panel_updates` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/installed font two updates", fps = 120) | 1042 |  | `BenchmarkRandomizedElementTree_InstalledFontPanelUpdates` | todo |
| `first_populated_text_panels` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/first populated installed font", fps = 120) | 1099 |  | `BenchmarkRandomizedElementTree_FirstPopulatedTextPanels` | todo |
| `layout_churn` | gpui::bench(inputs = churn_workloads(), input_name = "workload", group = "LayoutChurn", fps = 120) | 1349 |  | `BenchmarkRandomizedElementTree_LayoutChurn` | todo |
| `scroll` | gpui::bench(inputs = scroll_workloads(), input_name = "workload", group = "Scroll", fps = 360) | 1532 |  | `BenchmarkRandomizedElementTree_Scroll` | todo |

## gpui_platform/src/gpui_platform.rs

Go target: `platform/platform.go` (package `platform`), tests in `platform_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `background_executor` | fn | 12 |  | `BackgroundExecutor` | todo |
| `application` | fn | 16 |  | `Application` | todo |
| `headless` | fn | 20 |  | `Headless` | todo |
| `linux` | fn | 33 | linux,freebsd | `Linux` | todo |
| `application_with_web_backend` | fn | 38 | wasm | `ApplicationWithWebBackend` | todo |
| `single_threaded_web` | fn | 48 | wasm | `SingleThreadedWeb` | todo |
| `web_init` | fn | 56 | wasm | `WebInit` | todo |
| `current_platform` | fn | 62 |  | `CurrentPlatform` | todo |
| `bench_text_system` | fn | 92 | feat:bench-support | `BenchTextSystem` | todo |
| `current_headless_renderer` | fn | 102 | feat:bench-support,feat:test-support | `CurrentHeadlessRenderer` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_foreground_tasks_run_with_run_until_parked` | test ignore | 139 | macos | `TestGpuiPlatform_ForegroundTasksRunWithRunUntilParked` | todo |
| `test_advance_clock_triggers_delayed_tasks` | test ignore | 168 | macos | `TestGpuiPlatform_AdvanceClockTriggersDelayedTasks` | todo |
| `test_window_spawn_uses_test_dispatcher` | test ignore | 199 | macos | `TestGpuiPlatform_WindowSpawnUsesTestDispatcher` | todo |

