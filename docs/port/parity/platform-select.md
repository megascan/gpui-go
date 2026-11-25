# Parity: platform-select

Scope: gpui_platform. Milestone: M1. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_platform/benches/atlas_upload.rs

Go target: `platform/bench_atlas_upload_test.go` (package `platform_test`), tests in `bench_atlas_upload_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `atlas_upload` | criterion bench_group target | 34 | macos | `BenchmarkAtlasUpload_AtlasUpload` | dev:D-BENCHMETRICS |

## gpui_platform/benches/inspector_render.rs

Go target: `platform/bench_inspector_render_test.go` (package `platform_test`), tests in `bench_inspector_render_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `bare_grid` | gpui::bench(inputs = closed_inspector_inputs("nodes", [100, 1_000, 10_000]), input_name = "case", group = "closed_inspector_grid", fps = 120) | 67 |  | `BenchmarkInspectorRender_BareGrid` | done |
| `panel` | gpui::bench(inputs = closed_inspector_inputs("rows", [50, 500]), input_name = "case", group = "closed_inspector_panel", fps = 120) | 85 |  | `BenchmarkInspectorRender_Panel` | done |

## gpui_platform/benches/primitives.rs

Go target: `platform/bench_primitives_test.go` (package `platform_test`), tests in `bench_primitives_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `paint_rectangles` | gpui::bench(inputs = cases(), input_name = "case", group = "Primitives/paint", fps = 120) | 410 |  | `BenchmarkPrimitives_PaintRectangles` | done |
| `scene_ordering` | criterion bench_group target | 381 |  | `BenchmarkPrimitives_SceneOrdering` | dev:D-BENCHMETRICS |
| `scene_finish` | criterion bench_group target | 361 |  | `BenchmarkPrimitives_SceneFinish` | dev:D-BENCHMETRICS |

## gpui_platform/benches/randomized_element_tree.rs

Go target: `platform/bench_randomized_element_tree_test.go` (package `platform_test`), tests in `bench_randomized_element_tree_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/full refresh", fps = 120) | 393 |  | `BenchmarkRandomizedElementTree_FullRefresh` | done |
| `pathless_full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "PathlessTree/full refresh", fps = 120) | 409 |  | `BenchmarkRandomizedElementTree_PathlessFullRefresh` | done |
| `pathless_element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "PathlessTree/element style", fps = 120) | 418 |  | `BenchmarkRandomizedElementTree_PathlessElementStyle` | done |
| `unchanged` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/unchanged", fps = 120) | 427 |  | `BenchmarkRandomizedElementTree_Unchanged` | done |
| `element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/element style", fps = 120) | 452 |  | `BenchmarkRandomizedElementTree_ElementStyle` | done |
| `leaf_bounds` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/leaf bounds", fps = 120) | 460 |  | `BenchmarkRandomizedElementTree_LeafBounds` | done |
| `root_layout` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/root layout", fps = 120) | 469 |  | `BenchmarkRandomizedElementTree_RootLayout` | done |
| `insert_remove` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/insert-remove", fps = 120) | 479 |  | `BenchmarkRandomizedElementTree_InsertRemove` | done |
| `reorder` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/reorder", fps = 120) | 496 |  | `BenchmarkRandomizedElementTree_Reorder` | done |
| `hover` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/hover", fps = 120) | 562 |  | `BenchmarkRandomizedElementTree_Hover` | done |
| `mixed` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "RandomizedTree/mixed", fps = 120) | 774 |  | `BenchmarkRandomizedElementTree_Mixed` | done |
| `cached_full_refresh` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/full refresh", fps = 120) | 788 |  | `BenchmarkRandomizedElementTree_CachedFullRefresh` | done |
| `cached_element_style` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/element style", fps = 120) | 796 |  | `BenchmarkRandomizedElementTree_CachedElementStyle` | done |
| `cached_leaf_bounds` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/leaf bounds", fps = 120) | 802 |  | `BenchmarkRandomizedElementTree_CachedLeafBounds` | done |
| `cached_insert_remove` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/insert-remove", fps = 120) | 810 |  | `BenchmarkRandomizedElementTree_CachedInsertRemove` | done |
| `cached_hover` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/hover", fps = 120) | 825 |  | `BenchmarkRandomizedElementTree_CachedHover` | done |
| `cached_mixed` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/mixed", fps = 120) | 832 |  | `BenchmarkRandomizedElementTree_CachedMixed` | done |
| `cached_reorder` | gpui::bench(inputs = families(), input_name = "tree", iterations = 2, group = "CachedTree/reorder", fps = 120) | 838 |  | `BenchmarkRandomizedElementTree_CachedReorder` | done |
| `changing_share` | gpui::bench(inputs = changing_share_inputs(), input_name = "tree", iterations = 1, group = "RandomizedTree/changing share", fps = 120) | 907 |  | `BenchmarkRandomizedElementTree_ChangingShare` | done |
| `text_panel_updates` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/two updates", fps = 120) | 1016 |  | `BenchmarkRandomizedElementTree_TextPanelUpdates` | done |
| `installed_font_panel_updates` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/installed font two updates", fps = 120) | 1042 |  | `BenchmarkRandomizedElementTree_InstalledFontPanelUpdates` | done |
| `first_populated_text_panels` | gpui::bench(inputs = text_panel_counts(), input_name = "panels", group = "TextPanels/first populated installed font", fps = 120) | 1099 |  | `BenchmarkRandomizedElementTree_FirstPopulatedTextPanels` | done |
| `layout_churn` | gpui::bench(inputs = churn_workloads(), input_name = "workload", group = "LayoutChurn", fps = 120) | 1349 |  | `BenchmarkRandomizedElementTree_LayoutChurn` | done |
| `scroll` | gpui::bench(inputs = scroll_workloads(), input_name = "workload", group = "Scroll", fps = 360) | 1532 |  | `BenchmarkRandomizedElementTree_Scroll` | done |
| `family_summary` | criterion bench_group target | 345 |  | `BenchmarkRandomizedElementTree_FamilySummary` | dev:D-PLATFORM-FAMILYSUMMARY |

## gpui_platform/src/gpui_platform.rs

Go target: `platform/platform.go` (package `platform`), tests in `platform_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `background_executor` | fn | 12 |  | `BackgroundExecutor` | done |
| `application` | fn | 16 |  | `Application` | done |
| `headless` | fn | 20 |  | `Headless` | done |
| `linux` | fn | 33 | linux,freebsd | `Linux` | done |
| `application_with_web_backend` | fn | 38 | wasm | `ApplicationWithWebBackend` | dev:L-WEBTHREADS |
| `single_threaded_web` | fn | 48 | wasm | `SingleThreadedWeb` | done |
| `web_init` | fn | 56 | wasm | `WebInit` | done |
| `current_platform` | fn | 62 |  | `CurrentPlatform` | done |
| `bench_text_system` | fn | 92 | feat:bench-support | `BenchTextSystem` | dev:D-PLATFORM-BENCHTS |
| `current_headless_renderer` | fn | 102 | feat:bench-support,feat:test-support | `CurrentHeadlessRenderer` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_foreground_tasks_run_with_run_until_parked` | test ignore | 139 | macos | `TestGpuiPlatform_ForegroundTasksRunWithRunUntilParked` | done |
| `test_advance_clock_triggers_delayed_tasks` | test ignore | 168 | macos | `TestGpuiPlatform_AdvanceClockTriggersDelayedTasks` | done |
| `test_window_spawn_uses_test_dispatcher` | test ignore | 199 | macos | `TestGpuiPlatform_WindowSpawnUsesTestDispatcher` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-PLATFORM-BENCHTS | `BenchTextSystem` builds the text system once per process (`sync.OnceValue`) instead of once per thread | `gpui_platform/src/gpui_platform.rs:92-100` | Go has no thread-locals; gpui runs on the one locked main thread |
| D-PLATFORM-FAMILYSUMMARY | `family_summary`, the last Criterion target, is a benchmark declared last in its file, so `go test -bench` runs it after the tree benchmarks; it prints and clears the summary on its first call | `gpui_platform/benches/randomized_element_tree.rs:345` | Go has no bench_group target list; testing runs benchmarks in declaration order |
