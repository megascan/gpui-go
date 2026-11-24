# Parity: core-profiler

Scope: gpui: profiler, debug overlay, hot reload, randomized element tree, bench context. Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/bench.rs

Go target: `bench.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `bench_group` | macro | 18 |  | `RunBench` (testing.B, see DESIGN §9 (bench_group!)) | dev:D-BENCH |
| `bench_main` | macro | 51 |  | `RunBench` (testing.B, see DESIGN §9 (bench_main!)) | dev:D-BENCH |

## gpui/src/debug_overlay.rs

Go target: `debug_overlay.go` (package `gpui`), tests in `debug_overlay_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DebugFrameOverlayMode` | enum | 13 |  | `DebugFrameOverlayMode` | todo |
| `DebugFrameOverlayMode::next` | method | 22 |  | `(DebugFrameOverlayMode).Next` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `all_rendered_characters_have_glyphs` | test | 325 |  | `TestDebugOverlay_AllRenderedCharactersHaveGlyphs` | todo |
| `percentile_lows_are_reported_as_times` | test | 360 |  | `TestDebugOverlay_PercentileLowsAreReportedAsTimes` | todo |
| `reset_clears_durations_but_keeps_frame_count` | test | 375 |  | `TestDebugOverlay_ResetClearsDurationsButKeepsFrameCount` | todo |
| `frame_count_accumulates_across_mode_changes` | test | 393 |  | `TestDebugOverlay_FrameCountAccumulatesAcrossModeChanges` | todo |
| `frame_count_is_right_aligned_and_saturates` | test | 410 |  | `TestDebugOverlay_FrameCountIsRightAlignedAndSaturates` | todo |
| `toggling_on_shows_previous_frame_immediately` | test | 422 |  | `TestDebugOverlay_TogglingOnShowsPreviousFrameImmediately` | todo |

## gpui/src/hot_reload.rs

Go target: `hot_reload.go` (package `gpui`), tests in `hot_reload_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `receives_only_matching_patches_over_the_websocket` | test | 136 |  | `TestHotReload_ReceivesOnlyMatchingPatchesOverTheWebsocket` | todo |
| `refresh_renders_cached_views_without_recreating_state` | gpui::test | 220 |  | `TestHotReload_RefreshRendersCachedViewsWithoutRecreatingState` | todo |

## gpui/src/profiler.rs

Go target: `profiler.go` (package `gpui`), tests in `profiler_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `get_all_timings` | fn | 36 | feat:profiler | `GetAllTimings` | done |
| `get_current_thread_timings` | fn | 42 | feat:profiler | `GetCurrentThreadTimings` | dev:D-PROFTHREAD |
| `take_all_stats` | fn | 48 | feat:profiler | `TakeAllStats` | done |
| `get_all_timings` | fn | 54 | feat:profiler | `GetAllTimings` | done |
| `get_current_thread_timings` | fn | 59 | feat:profiler | `GetCurrentThreadTimings` | dev:D-PROFTHREAD |
| `take_all_stats` | fn | 70 | feat:profiler | `TakeAllStats` | done |
| `YieldTime` | struct | 76 |  | `YieldTime` | done |
| `TaskTiming` | struct | 80 |  | `TaskTiming` | done |
| `ActiveTiming` | struct | 100 |  | `ActiveTiming` | done |
| `TaskTiming::placeholder` | assoc-fn | 108 |  | `TaskTimingPlaceholder` | done |
| `TaskTiming::poll_duration` | method | 119 |  | `(TaskTiming).PollDuration` | done |
| `ThreadTaskTimings` | struct | 131 |  | `ThreadTaskTimings` | done |
| `ThreadTaskTimings::collect` | assoc-fn | 141 |  | `ThreadTaskTimingsCollect` | done |
| `ThreadTaskStatistics` | struct | 182 |  | `ThreadTaskStatistics` | done |
| `ThreadTaskStatistics::collect_and_reset` | assoc-fn | 189 |  | `ThreadTaskStatisticsCollectAndReset` | done |
| `SerializedLocation` | struct | 230 |  | `SerializedLocation` | done |
| `SerializedTaskTiming` | struct | 251 |  | `SerializedTaskTiming` | done |
| `SerializedTaskTiming::convert` | assoc-fn | 266 |  | `SerializedTaskTimingConvert` | done |
| `SerializedTaskTiming::from` | assoc-fn | 284 |  | `SerializedTaskTimingFrom` | done |
| `SerializedThreadTaskTimings` | struct | 297 |  | `SerializedThreadTaskTimings` | done |
| `SerializedThreadTaskTimings::convert` | assoc-fn | 312 |  | `SerializedThreadTaskTimingsConvert` | done |
| `ThreadTimingsDelta` | struct | 329 |  | `ThreadTimingsDelta` | done |
| `ProfilingCollector` | struct | 341 |  | `ProfilingCollector` | done |
| `ProfilingCollector::new` | assoc-fn | 347 |  | `NewProfilingCollector` | done |
| `ProfilingCollector::startup_time` | method | 354 |  | `(ProfilingCollector).StartupTime` | done |
| `ProfilingCollector::collect_unseen` | method | 358 |  | `(ProfilingCollector).CollectUnseen` | done |
| `ProfilingCollector::reset` | method | 401 |  | `(ProfilingCollector).Reset` | done |
| `GuardedTaskTimings` | type | 416 |  | `GuardedTaskTimings` | done |
| `GlobalThreadTimings` | struct | 419 |  | `GlobalThreadTimings` | done |
| `TaskStatistics` | struct | 426 |  | `TaskStatistics` | done |
| `GLOBAL_THREAD_TIMINGS` | static | 515 |  | `globalThreadTimings` | dev:D-NAME |
| `ThreadTimings` | struct | 558 |  | `ThreadTimings` | done |
| `ThreadTimings::new` | assoc-fn | 568 |  | `NewThreadTimings` | done |
| `ThreadTimings::update_running_task` | method | 580 | feat:profiler | `(ThreadTimings).UpdateRunningTask` | done |
| `ThreadTimings::update_running_task` | method | 593 | feat:profiler | `(ThreadTimings).UpdateRunningTask` | done |
| `ThreadTimings::save_task_timing` | method | 596 | feat:profiler | `(ThreadTimings).SaveTaskTiming` | done |
| `ThreadTimings::save_task_timing` | method | 626 | feat:profiler | `(ThreadTimings).SaveTaskTiming` | done |
| `ThreadTimings::get_thread_task_timings` | method | 630 |  | `(ThreadTimings).GetThreadTaskTimings` | done |
| `update_running_task` | fn | 671 |  | `UpdateRunningTask` | dev:D-PROFTHREAD |
| `save_task_timing` | fn | 680 |  | `SaveTaskTiming` | dev:D-PROFTHREAD |
| `get_current_thread_task_timings` | fn | 694 |  | `GetCurrentThreadTaskTimings` | dev:D-PROFTHREAD |
| `set_trace_enabled` | fn | 709 |  | `SetTraceEnabled` | done |
| `trace_enabled` | fn | 766 |  | `TraceEnabled` | done |
| `FrameTiming` | struct | 789 | feat:profiler | `FrameTiming` | done |
| `DrawPhases` | struct | 814 | feat:profiler | `DrawPhases` | done |
| `ViewPhase` | enum | 827 | feat:profiler | `ViewPhase` | done |
| `ViewPhase::as_str` | method | 840 | feat:profiler | `(ViewPhase).AsStr` | done |
| `VIEW_TIMING_FLOOR` | const | 853 | feat:profiler | `ViewTimingFloor` | done |
| `ViewTiming` | struct | 859 | feat:profiler | `ViewTiming` | done |
| `FRAME_BUDGET_FLOOR` | const | 878 | feat:profiler | `FrameBudgetFloor` | done |
| `UNKNOWN_REFRESH_FRAME_BUDGET` | const | 884 | feat:profiler | `UnknownRefreshFrameBudget` | done |
| `FrameTiming::work_start` | method | 891 | feat:profiler | `(FrameTiming).WorkStart` | done |
| `FrameTiming::work` | method | 905 | feat:profiler | `(FrameTiming).Work` | done |
| `FrameTiming::is_late` | method | 911 | feat:profiler | `(FrameTiming).IsLate` | done |
| `FrameTiming::budget` | method | 918 | feat:profiler | `(FrameTiming).Budget` | done |
| `FrameTiming::missed_refreshes` | method | 927 | feat:profiler | `(FrameTiming).MissedRefreshes` | done |
| `FrameTiming::draw_duration` | method | 935 | feat:profiler | `(FrameTiming).DrawDuration` | done |
| `FrameTiming::dirty_to_draw_duration` | method | 941 | feat:profiler | `(FrameTiming).DirtyToDrawDuration` | done |
| `PresentTiming` | struct | 950 | feat:profiler | `PresentTiming` | done |
| `PresentTiming::present_duration` | method | 965 | feat:profiler | `(PresentTiming).PresentDuration` | done |
| `FrameEvent` | enum | 973 | feat:profiler | `FrameEvent` | done |
| `FrameDurationSnapshot` | struct | 984 | feat:profiler | `FrameDurationSnapshot` | done |
| `FRAME_DURATION_BUCKETS_MS` | const | 1001 | feat:profiler | `FrameDurationBucketsMs` | done |
| `FRAME_WORK_BUCKETS_PERCENT` | const | 1010 | feat:profiler | `FrameWorkBucketsPercent` | done |
| `histogram_bucket_counts` | fn | 1016 | feat:profiler | `HistogramBucketCounts` | done |
| `RefreshIntervalFrames` | struct | 1043 | feat:profiler | `RefreshIntervalFrames` | done |
| `RefreshIntervalFrames::since` | method | 1091 | feat:profiler | `(RefreshIntervalFrames).Since` | done |
| `InputLatencySnapshot` | struct | 1165 | feat:profiler | `InputLatencySnapshot` | done |
| `WindowProfiler` | struct | 1192 | feat:profiler | `WindowProfiler` | done |
| `WindowProfiler::new` | assoc-fn | 1222 | feat:profiler | `NewWindowProfiler` | done |
| `WindowProfiler::begin_input` | method | 1262 | feat:profiler | `(WindowProfiler).BeginInput` | done |
| `WindowProfiler::end_input` | method | 1271 | feat:profiler | `(WindowProfiler).EndInput` | done |
| `WindowProfiler::begin_action_handler` | method | 1312 | feat:profiler | `(WindowProfiler).BeginActionHandler` | done |
| `WindowProfiler::end_action_handler` | method | 1319 | feat:profiler | `(WindowProfiler).EndActionHandler` | done |
| `WindowProfiler::record_frame_signal` | method | 1341 | feat:profiler | `(WindowProfiler).RecordFrameSignal` | done |
| `WindowProfiler::set_refresh_interval` | method | 1347 | feat:profiler | `(WindowProfiler).SetRefreshInterval` | done |
| `WindowProfiler::begin_draw` | method | 1352 | feat:profiler | `(WindowProfiler).BeginDraw` | done |
| `WindowProfiler::record_draw_phases` | method | 1361 | feat:profiler | `(WindowProfiler).RecordDrawPhases` | done |
| `WindowProfiler::begin_view` | method | 1367 | feat:profiler | `(WindowProfiler).BeginView` | done |
| `WindowProfiler::end_view` | method | 1374 | feat:profiler | `(WindowProfiler).EndView` | done |
| `WindowProfiler::end_draw` | method | 1394 | feat:profiler | `(WindowProfiler).EndDraw` | done |
| `WindowProfiler::record_present` | method | 1430 | feat:profiler | `(WindowProfiler).RecordPresent` | done |
| `WindowProfiler::input_latency_snapshot` | method | 1446 | feat:profiler | `(WindowProfiler).InputLatencySnapshot` | done |
| `WindowProfiler::frame_duration_snapshot` | method | 1455 | feat:profiler | `(WindowProfiler).FrameDurationSnapshot` | done |
| `record_frame_event` | fn | 1599 | feat:profiler | `RecordFrameEvent` | done |
| `FrameTimingCollector` | struct | 1616 | feat:profiler | `FrameTimingCollector` | done |
| `FrameTimingCollector::new` | assoc-fn | 1630 | feat:profiler | `NewFrameTimingCollector` | done |
| `FrameTimingCollector::collect_unseen` | method | 1639 | feat:profiler | `(FrameTimingCollector).CollectUnseen` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `interruptions_drop_latency_samples_but_hiding_keeps_draw_work` | test | 1661 | feat:profiler | `TestProfiler_InterruptionsDropLatencySamplesButHidingKeepsDrawWork` | blocked:M3-exit |
| `frame_work_starts_at_the_later_of_signal_and_invalidation` | test | 1698 | feat:profiler | `TestProfiler_FrameWorkStartsAtTheLaterOfSignalAndInvalidation` | done |
| `frames_are_split_by_refresh_interval_and_classified` | test | 1731 | feat:profiler | `TestProfiler_FramesAreSplitByRefreshIntervalAndClassified` | done |
| `records_draw_events_only_while_tracing` | test | 1886 | feat:profiler | `TestProfiler_RecordsDrawEventsOnlyWhileTracing` | done |
| `records_present_events_for_newly_drawn_frames` | test | 1922 | feat:profiler | `TestProfiler_RecordsPresentEventsForNewlyDrawnFrames` | done |
| `disabling_tracing_clears_frame_events` | test | 1970 | feat:profiler | `TestProfiler_DisablingTracingClearsFrameEvents` | done |
| `records_intervals_only_between_animation_frames` | test | 2000 | feat:profiler | `TestProfiler_RecordsIntervalsOnlyBetweenAnimationFrames` | done |
| `missed_frames_stretch_the_recorded_interval` | test | 2020 | feat:profiler | `TestProfiler_MissedFramesStretchTheRecordedInterval` | done |
| `ignores_re_presents_of_unchanged_frames` | test | 2034 | feat:profiler | `TestProfiler_IgnoresRePresentsOfUnchangedFrames` | done |
| `skips_intervals_for_inactive_windows` | test | 2051 | feat:profiler | `TestProfiler_SkipsIntervalsForInactiveWindows` | done |
| `records_dirty_to_present_durations` | test | 2068 | feat:profiler | `TestProfiler_RecordsDirtyToPresentDurations` | done |
| `records_every_draw_duration` | test | 2085 | feat:profiler | `TestProfiler_RecordsEveryDrawDuration` | done |
| `records_input_latency_at_the_frame_presentation_timestamp` | test | 2098 | feat:profiler | `TestProfiler_RecordsInputLatencyAtTheFramePresentationTimestamp` | done |
| `excludes_input_that_arrives_during_a_draw` | test | 2123 | feat:profiler | `TestProfiler_ExcludesInputThatArrivesDuringADraw` | done |
| `overlapping_trace_scopes_keep_tracing_enabled` | test | 2139 | feat:profiler | `TestProfiler_OverlappingTraceScopesKeepTracingEnabled` | done |

## gpui/src/randomized_element_tree.rs

Go target: `randomized_element_tree.go` (package `gpui`), tests in `randomized_element_tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `RandomizedElementTreeAction` | struct | 20 |  | `RandomizedElementTreeAction` | done |
| `RandomizedElementTreeTopology` | enum | 24 |  | `RandomizedElementTreeTopology` | done |
| `RandomizedElementTreeTopology::ALL` | assoc-const | 37 |  | `RandomizedElementTreeTopologyAll` | done |
| `RandomizedElementTreeTopology::is_tall` | method | 43 |  | `(RandomizedElementTreeTopology).IsTall` | done |
| `RandomizedElementTreeConfig` | struct | 59 |  | `RandomizedElementTreeConfig` | done |
| `RandomizedElementTreeConfig::new` | assoc-fn | 73 |  | `NewRandomizedElementTreeConfig` | done |
| `RandomizedElementTreeConfig::with_cached_views` | method | 88 |  | `(RandomizedElementTreeConfig).WithCachedViews` | done |
| `RandomizedElementTreeConfig::with_animation_density` | method | 95 |  | `(RandomizedElementTreeConfig).WithAnimationDensity` | done |
| `RandomizedElementTreeConfig::with_hover_styles` | method | 102 |  | `(RandomizedElementTreeConfig).WithHoverStyles` | done |
| `RandomizedElementTreeConfig::with_paths` | method | 110 |  | `(RandomizedElementTreeConfig).WithPaths` | done |
| `RandomizedElementTreeConfig::with_topology` | method | 116 |  | `(RandomizedElementTreeConfig).WithTopology` | done |
| `RandomizedElementTreeConfig::with_entity_density` | method | 122 |  | `(RandomizedElementTreeConfig).WithEntityDensity` | done |
| `RandomizedElementTreeConfig::with_handler_density` | method | 128 |  | `(RandomizedElementTreeConfig).WithHandlerDensity` | done |
| `RandomizedElementTreeConfig::seed` | method | 134 |  | `(RandomizedElementTreeConfig).Seed` | done |
| `RandomizedElementTreeConfig::element_count` | method | 139 |  | `(RandomizedElementTreeConfig).ElementCount` | done |
| `RandomizedElementTreeConfig::topology` | method | 144 |  | `(RandomizedElementTreeConfig).Topology` | done |
| `RandomizedElementTreeConfig::entity_density` | method | 149 |  | `(RandomizedElementTreeConfig).EntityDensity` | done |
| `RandomizedElementTreeConfig::handler_density` | method | 154 |  | `(RandomizedElementTreeConfig).HandlerDensity` | done |
| `RandomizedElementTreeBounds` | struct | 165 |  | `RandomizedElementTreeBounds` | done |
| `RandomizedElementTreeBounds::new` | assoc-fn | 174 |  | `NewRandomizedElementTreeBounds` | done |
| `RandomizedElementTreeBounds::with_topologies` | method | 184 |  | `(RandomizedElementTreeBounds).WithTopologies` | done |
| `RandomizedElementTreeBounds::with_entity_density` | method | 196 |  | `(RandomizedElementTreeBounds).WithEntityDensity` | done |
| `RandomizedElementTreeBounds::with_handler_density` | method | 203 |  | `(RandomizedElementTreeBounds).WithHandlerDensity` | done |
| `RandomizedElementTreeBounds::sample` | method | 212 |  | `(RandomizedElementTreeBounds).Sample` | done |
| `RandomizedElementTreeWorkCounters` | struct | 235 |  | `RandomizedElementTreeWorkCounters` | done |
| `RandomizedElementTreeSnapshot` | struct | 311 |  | `RandomizedElementTreeSnapshot` | done |
| `RandomizedElementTreeSnapshot::descendant_count` | method | 321 |  | `(RandomizedElementTreeSnapshot).DescendantCount` | done |
| `RandomizedElementTreeSnapshot::max_depth` | method | 334 |  | `(RandomizedElementTreeSnapshot).MaxDepth` | done |
| `RandomizedElementTreeSnapshot::element_ids` | method | 349 |  | `(RandomizedElementTreeSnapshot).ElementIDs` | done |
| `RandomizedElementTreeSnapshot::entity_count` | method | 357 |  | `(RandomizedElementTreeSnapshot).EntityCount` | done |
| `RandomizedElementTreeSnapshot::handler_count` | method | 368 |  | `(RandomizedElementTreeSnapshot).HandlerCount` | done |
| `RandomizedElementTreeMutationKind` | enum | 392 |  | `RandomizedElementTreeMutationKind` | done |
| `RandomizedElementTreeMutationKind::ALL` | assoc-const | 434 |  | `RandomizedElementTreeMutationKindAll` | done |
| `RandomizedElementTreeMutation` | enum | 457 |  | `RandomizedElementTreeMutation` | done |
| `ChangeLocality` | enum | 540 |  | `ChangeLocality` | done |
| `RecoloredChildren` | struct | 552 |  | `RecoloredChildren` | done |
| `RandomizedElementTree` | struct | 560 |  | `RandomizedElementTree` | done |
| `RandomizedElementTree::new` | assoc-fn | 573 |  | `NewRandomizedElementTree` | done |
| `RandomizedElementTree::new_with_config` | assoc-fn | 578 |  | `NewRandomizedElementTreeWithConfig` | done |
| `RandomizedElementTree::from_snapshot` | assoc-fn | 631 |  | `RandomizedElementTreeFromSnapshot` | done |
| `RandomizedElementTree::from_snapshot_with_entities` | assoc-fn | 643 |  | `RandomizedElementTreeFromSnapshotWithEntities` | done |
| `RandomizedElementTree::snapshot` | method | 653 |  | `(RandomizedElementTree).Snapshot` | done |
| `RandomizedElementTree::mutate` | method | 673 |  | `(RandomizedElementTree).Mutate` | done |
| `RandomizedElementTree::apply_random_mutation` | method | 681 |  | `(RandomizedElementTree).ApplyRandomMutation` | done |
| `RandomizedElementTree::apply_mutation` | method | 692 |  | `(RandomizedElementTree).ApplyMutation` | done |
| `RandomizedElementTree::random_element_id` | method | 706 |  | `(RandomizedElementTree).RandomElementID` | done |
| `RandomizedElementTree::alternate_element_color` | method | 715 |  | `(RandomizedElementTree).AlternateElementColor` | done |
| `RandomizedElementTree::insert_child` | method | 742 |  | `(RandomizedElementTree).InsertChild` | done |
| `RandomizedElementTree::remove_child` | method | 750 |  | `(RandomizedElementTree).RemoveChild` | done |
| `RandomizedElementTree::recolor_children` | method | 903 |  | `(RandomizedElementTree).RecolorChildren` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `localized_recoloring_notifies_one_entity_until_its_subtree_is_exhausted` | gpui::test | 1799 |  | `TestRandomizedElementTree_LocalizedRecoloringNotifiesOneEntityUntilItsSubtreeIsExhausted` | dev:D-RNG |
| `bounds_sample_within_range_and_reproduce_from_the_seed` | test | 1827 |  | `TestRandomizedElementTree_BoundsSampleWithinRangeAndReproduceFromTheSeed` | done |
| `generated_topologies_include_wide_deep_narrow_and_empty_trees` | test | 1851 |  | `TestRandomizedElementTree_GeneratedTopologiesIncludeWideDeepNarrowAndEmptyTrees` | done |
| `randomized_element_ids_are_unique_and_stable` | test | 1870 |  | `TestRandomizedElementTree_RandomizedElementIDsAreUniqueAndStable` | done |
| `randomized_entities_and_work_counters_support_incremental_benchmarks` | gpui::test | 1915 |  | `TestRandomizedElementTree_RandomizedEntitiesAndWorkCountersSupportIncrementalBenchmarks` | blocked:M3-exit |
| `randomized_element_tree_matches_full_render` | gpui::property_test(config = ProptestConfig { cases: 4, ..Default::default() }) | 1965 |  | `TestRandomizedElementTree_RandomizedElementTreeMatchesFullRender` | blocked:M3-exit |

## gpui/src/app/bench_context.rs

Go target: `app_bench_context.go` (package `gpui`), tests in `app_bench_context_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `bench_platform` | fn | 47 |  | `BenchPlatform` | done |
| `ForegroundWorkSummary` | struct | 78 |  | `ForegroundWorkSummary` | done |
| `BenchReport` | struct | 103 |  | `BenchReport` | done |
| `BenchReport::with_fps` | assoc-fn | 118 |  | `BenchReportWithFps` | done |
| `BenchReport::with_frame_budget_nanos` | assoc-fn | 125 |  | `BenchReportWithFrameBudgetNanos` | done |
| `BenchReport::metrics` | method | 140 |  | `(BenchReport).Metrics` | done |
| `BenchReport::foreground_work` | method | 230 |  | `(BenchReport).ForegroundWork` | done |
| `BenchReport::print` | method | 252 |  | `(BenchReport).Print` | done |
| `BenchAppContext` | struct | 496 |  | `BenchAppContext` | done |
| `BenchAppContext::new` | assoc-fn | 511 |  | `NewBenchAppContext` | done |
| `BenchAppContext::new_with_platform_and_report` | assoc-fn | 525 |  | `NewBenchAppContextWithPlatformAndReport` | done |
| `BenchAppContext::benchmark_name` | method | 568 |  | `(BenchAppContext).BenchmarkName` | done |
| `BenchAppContext::background_executor` | method | 573 |  | `(BenchAppContext).BackgroundExecutor` | done |
| `BenchAppContext::foreground_executor` | method | 578 |  | `(BenchAppContext).ForegroundExecutor` | done |
| `BenchAppContext::update` | method | 583 |  | `(BenchAppContext).Update` | done |
| `BenchAppContext::read` | method | 589 |  | `(BenchAppContext).Read` | done |
| `BenchAppContext::run_until_idle` | method | 599 |  | `(BenchAppContext).RunUntilIdle` | done |
| `BenchAppContext::settle` | method | 629 |  | `(BenchAppContext).Settle` | done |
| `BenchAppContext::run_until` | method | 649 |  | `(BenchAppContext).RunUntil` | done |
| `BenchAppContext::bench_iter` | method | 671 |  | `(BenchAppContext).BenchIter` | done |
| `BenchAppContext::bench_task` | method | 700 |  | `(BenchAppContext).BenchTask` | done |
| `BenchAppContext::bench_batched_task` | method | 716 |  | `(BenchAppContext).BenchBatchedTask` | done |
| `BenchAppContext::bench_renderer` | method | 799 |  | `(BenchAppContext).BenchRenderer` | done |
| `BenchAppContext::bench_renderer_session` | method | 864 |  | `(BenchAppContext).BenchRendererSession` | done |
| `BenchAppContext::add_empty_window` | method | 1007 |  | `(BenchAppContext).AddEmptyWindow` | done |
| `BenchAppContext::teardown` | method | 1060 |  | `(BenchAppContext).Teardown` | done |
| `BenchWindowContext` | struct | 1182 |  | `BenchWindowContext` | done |
| `BenchWindowContext::app_context` | method | 1189 |  | `(BenchWindowContext).AppContext` | done |
| `BenchWindowContext::window_handle` | method | 1194 |  | `(BenchWindowContext).WindowHandle` | done |
| `BenchWindowContext::run_until_idle` | method | 1200 |  | `(BenchWindowContext).RunUntilIdle` | done |
| `BenchWindowContext::update` | method | 1205 |  | `(BenchWindowContext).Update` | done |
| `BenchWindowContext::simulate_resize` | method | 1215 |  | `(BenchWindowContext).SimulateResize` | done |
| `BenchMeasurement` | re-export | 13 |  | `BenchMeasurement` | dev:D-BENCHMETRICS |
| `MetricReport` | re-export | 13 |  | `MetricReport` | dev:D-BENCHMETRICS |
| `CountingAllocator` | re-export | 13 |  | `BenchMeasurementFromEnv` (heap allocation counters) | dev:D-BENCHMETRICS |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `secondary_metric_is_normalized_per_iteration_for_bench_iter` | test | 1428 | feat:test-support | `TestBenchContext_SecondaryMetricIsNormalizedPerIterationForBenchIter` | done |
| `secondary_metric_is_accumulated_across_per_iteration_batches` | test | 1443 | feat:test-support | `TestBenchContext_SecondaryMetricIsAccumulatedAcrossPerIterationBatches` | done |
| `foreground_work_reports_long_task_without_window_draw` | test | 1464 |  | `TestBenchContext_ForegroundWorkReportsLongTaskWithoutWindowDraw` | done |
| `foreground_work_excludes_setup_before_trace_scope_starts` | test | 1510 |  | `TestBenchContext_ForegroundWorkExcludesSetupBeforeTraceScopeStarts` | done |
| `bench_task_reports_long_task_without_window` | test | 1550 |  | `TestBenchContext_BenchTaskReportsLongTaskWithoutWindow` | done |
| `task_completion_supports_non_send_foreground_output` | test | 1588 |  | `TestBenchContext_TaskCompletionSupportsNonSendForegroundOutput` | done |

## gpui/src/profiler/actions.rs

Go target: `profiler_actions.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ActionStatistics` | struct | 11 |  | `ActionStatistics` | done |
| `ActionStatistics::take` | method | 67 |  | `(ActionStatistics).Take` | done |
| `ActionStatistics::is_empty` | method | 73 |  | `(ActionStatistics).IsEmpty` | done |
| `ActionStatistics::update_running_action` | method | 78 | feat:profiler | `(ActionStatistics).UpdateRunningAction` | done |
| `ActionStatistics::update_running_action` | method | 82 | feat:profiler | `(ActionStatistics).UpdateRunningAction` | done |
| `ActionStatistics::save_action_timing` | method | 85 | feat:profiler | `(ActionStatistics).SaveActionTiming` | done |
| `ActionStatistics::save_action_timing` | method | 131 | feat:profiler | `(ActionStatistics).SaveActionTiming` | done |
| `ActionStatistics::longest_runtimes` | method | 133 |  | `(ActionStatistics).LongestRuntimes` | done |
| `ActionTiming` | struct | 150 |  | `ActionTiming` | done |
| `ActionTiming::duration` | method | 166 |  | `(ActionTiming).Duration` | done |
| `ActionTiming::runtime` | method | 173 |  | `(ActionTiming).Runtime` | done |
| `take_action_stats` | fn | 205 | feat:profiler | `TakeActionStats` | done |
| `take_action_stats` | fn | 211 | feat:profiler | `TakeActionStats` | done |

## gpui/src/profiler/hang.rs

Go target: `profiler_hang.go` (package `gpui`), tests in `profiler_hang_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MEASUREMENT_VERSION` | const | 26 |  | `MeasurementVersion` | todo |
| `HangDetector` | struct | 39 |  | `HangDetector` | todo |
| `ACTIVE_USE_WINDOW` | const | 50 |  | `ActiveUseWindow` | todo |
| `HangIncident` | struct | 134 |  | `HangIncident` | todo |
| `HangTrigger` | enum | 158 |  | `HangTrigger` | todo |
| `HangTrigger::ALL` | assoc-const | 172 |  | `HangTriggerAll` | todo |
| `HangTrigger::as_str` | method | 175 |  | `(HangTrigger).AsStr` | todo |
| `HangDetector::new` | assoc-fn | 189 |  | `NewHangDetector` | todo |
| `HangDetector::take_active_time` | method | 206 |  | `(HangDetector).TakeActiveTime` | todo |
| `HangDetector::first_present_at` | method | 212 |  | `(HangDetector).FirstPresentAt` | todo |
| `HangDetector::poll` | method | 218 |  | `(HangDetector).Poll` | todo |
| `SerializedHangIncident` | struct | 249 |  | `SerializedHangIncident` | todo |
| `SerializedHangContributor` | enum | 313 |  | `SerializedHangContributor` | todo |
| `SerializedHangIncident::convert` | assoc-fn | 412 |  | `SerializedHangIncidentConvert` | todo |
| `HangIncident::active_window` | method | 578 |  | `(HangIncident).ActiveWindow` | todo |
| `HangIncident::detect` | assoc-fn | 599 |  | `HangIncidentDetect` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `completed_skips_preserve_real_hangs_and_serialize_their_boundary` | test | 694 |  | `TestHang_CompletedSkipsPreserveRealHangsAndSerializeTheirBoundary` | todo |
| `delayed_platform_signals_without_work_are_not_hangs` | test | 747 |  | `TestHang_DelayedPlatformSignalsWithoutWorkAreNotHangs` | todo |
| `a_hang_outliving_the_frame_deadline_keeps_its_frame_association` | test | 775 |  | `TestHang_AHangOutlivingTheFrameDeadlineKeepsItsFrameAssociation` | todo |
| `serialized_incident_reports_presented_seal_fields` | test | 821 |  | `TestHang_SerializedIncidentReportsPresentedSealFields` | todo |
| `hang_monitor_polls_on_its_interval_and_on_flush` | test | 910 |  | `TestHang_HangMonitorPollsOnItsIntervalAndOnFlush` | todo |
| `app_flushes_its_hang_monitor_on_shutdown` | gpui::test | 961 |  | `TestHang_AppFlushesItsHangMonitorOnShutdown` | todo |
| `starting_the_hang_monitor_twice_is_rejected` | gpui::test | 990 |  | `TestHang_StartingTheHangMonitorTwiceIsRejected` | todo |
| `serialized_incident_reports_idle_seal_fields` | test | 1014 |  | `TestHang_SerializedIncidentReportsIdleSealFields` | todo |
| `phase_is_startup_until_the_first_present` | test | 1040 |  | `TestHang_PhaseIsStartupUntilTheFirstPresent` | todo |
| `first_present_at_latches_on_the_first_observed_presentation` | test | 1074 |  | `TestHang_FirstPresentAtLatchesOnTheFirstObservedPresentation` | todo |
| `busy_fraction_excludes_small_polls_outside_the_active_window` | test | 1106 |  | `TestHang_BusyFractionExcludesSmallPollsOutsideTheActiveWindow` | todo |
| `an_interval_of_small_work_over_budget_is_an_incident` | test | 1140 |  | `TestHang_AnIntervalOfSmallWorkOverBudgetIsAnIncident` | todo |
| `a_journal_gap_suppresses_budget_inference_but_retains_observed_hangs` | test | 1190 |  | `TestHang_AJournalGapSuppressesBudgetInferenceButRetainsObservedHangs` | todo |
| `a_slow_frame_with_little_foreground_spend_is_not_an_incident` | test | 1229 |  | `TestHang_ASlowFrameWithLittleForegroundSpendIsNotAnIncident` | todo |
| `active_use_counts_time_within_a_minute_of_deliberate_input` | test | 1263 |  | `TestHang_ActiveUseCountsTimeWithinAMinuteOfDeliberateInput` | todo |
| `only_single_events_over_the_frame_budget_are_blocking_work` | test | 1313 |  | `TestHang_OnlySingleEventsOverTheFrameBudgetAreBlockingWork` | todo |
| `late_frames_are_budget_hangs` | test | 1339 |  | `TestHang_LateFramesAreBudgetHangs` | todo |
| `serialized_contributors_are_chronological_with_nesting_depths` | test | 1399 |  | `TestHang_SerializedContributorsAreChronologicalWithNestingDepths` | todo |
| `detects_randomly_placed_foreground_hangs` | gpui::test(iterations = 5) | 1600 |  | `TestHang_DetectsRandomlyPlacedForegroundHangs` | todo |
| `slow_views_are_journaled_with_their_own_time` | gpui::test | 1678 |  | `TestHang_SlowViewsAreJournaledWithTheirOwnTime` | todo |

## gpui/src/profiler/journal.rs

Go target: `profiler_journal.go` (package `gpui`), tests in `profiler_journal_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TASK_POLL_FLOOR` | const | 39 |  | `TaskPollFloor` | done |
| `FRAME_DEADLINE` | const | 47 |  | `FrameDeadline` | done |
| `ForegroundEvent` | enum | 78 |  | `ForegroundEvent` | done |
| `ForegroundEvent::start_time` | method | 99 |  | `(ForegroundEvent).StartTime` | done |
| `ForegroundEvent::end_time` | method | 112 |  | `(ForegroundEvent).EndTime` | done |
| `ForegroundEvent::duration` | method | 127 |  | `(ForegroundEvent).Duration` | done |
| `InputTiming` | struct | 134 |  | `InputTiming` | done |
| `PlatformSignal` | struct | 148 |  | `PlatformSignal` | done |
| `FrameSkipReason` | enum | 161 |  | `FrameSkipReason` | done |
| `FrameSkipped` | struct | 176 |  | `FrameSkipped` | done |
| `PollSummary` | struct | 187 |  | `PollSummary` | done |
| `SmallPollFlush` | struct | 203 |  | `SmallPollFlush` | done |
| `PresentedFrame` | struct | 214 |  | `PresentedFrame` | done |
| `PresentedFrame::dirty_to_present_duration` | method | 223 |  | `(PresentedFrame).DirtyToPresentDuration` | done |
| `IntervalBoundary` | enum | 232 |  | `IntervalBoundary` | done |
| `IntervalBoundary::end_time` | method | 254 |  | `(IntervalBoundary).EndTime` | done |
| `IntervalBoundary::dirty_at` | method | 263 |  | `(IntervalBoundary).DirtyAt` | done |
| `FrameStateChange` | enum | 273 |  | `FrameStateChange` | done |
| `ForegroundJournalEntry` | enum | 293 |  | `ForegroundJournalEntry` | done |
| `FrameSnapshot` | struct | 320 |  | `FrameSnapshot` | done |
| `FrameSnapshot::interval_end` | method | 343 |  | `(FrameSnapshot).IntervalEnd` | done |
| `FrameSnapshot::small_poll_summary` | method | 349 |  | `(FrameSnapshot).SmallPollSummary` | done |
| `FrameSnapshot::occupancy` | method | 362 |  | `(FrameSnapshot).Occupancy` | done |
| `FrameSnapshot::occupancy_within` | method | 372 |  | `(FrameSnapshot).OccupancyWithin` | done |
| `FrameSnapshot::busy_fraction` | method | 423 |  | `(FrameSnapshot).BusyFraction` | done |
| `ForegroundJournal` | struct | 1307 |  | `ForegroundJournal` | done |
| `ForegroundJournal::collector` | method | 1324 |  | `(ForegroundJournal).Collector` | done |
| `DrainedEntries` | struct | 1334 |  | `DrainedEntries` | done |
| `ForegroundJournalCollector` | struct | 1348 |  | `ForegroundJournalCollector` | done |
| `ForegroundJournalCollector::collect_unseen` | method | 1357 |  | `(ForegroundJournalCollector).CollectUnseen` | done |
| `IntervalSealer` | struct | 1395 |  | `IntervalSealer` | done |
| `IntervalSealer::new` | assoc-fn | 1406 |  | `NewIntervalSealer` | done |
| `IntervalSealer::note_lost` | method | 1418 |  | `(IntervalSealer).NoteLost` | done |
| `IntervalSealer::push_entries` | method | 1426 |  | `(IntervalSealer).PushEntries` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `platform_signals_are_journaled_without_sealing_or_occupying_time` | test | 1523 |  | `TestJournal_PlatformSignalsAreJournaledWithoutSealingOrOccupyingTime` | done |
| `completed_skips_seal_without_retiring_pending_demand` | test | 1567 |  | `TestJournal_CompletedSkipsSealWithoutRetiringPendingDemand` | done |
| `idle_refreshes_seal_nothing_and_keep_polls_folded` | test | 1626 |  | `TestJournal_IdleRefreshesSealNothingAndKeepPollsFolded` | done |
| `reentrant_skip_does_not_seal_the_enclosing_draw` | test | 1659 |  | `TestJournal_ReentrantSkipDoesNotSealTheEnclosingDraw` | done |
| `wake_rechecks_native_visibility` | gpui::test | 1707 |  | `TestJournal_WakeRechecksNativeVisibility` | done |
| `power_interruption_drops_the_poll_before_small_poll_folding` | test | 1728 |  | `TestJournal_PowerInterruptionDropsThePollBeforeSmallPollFolding` | done |
| `a_poll_spanning_an_unnotified_suspend_is_dropped` | test | 1765 |  | `TestJournal_APollSpanningAnUnnotifiedSuspendIsDropped` | done |
| `suspend_clock_ignores_jitter_and_notified_suspends` | test | 1800 |  | `TestJournal_SuspendClockIgnoresJitterAndNotifiedSuspends` | done |
| `visibility_and_power_changes_clear_pending_frames` | test | 1828 |  | `TestJournal_VisibilityAndPowerChangesClearPendingFrames` | done |
| `draw_waits_for_its_presentation_boundary` | test | 1860 |  | `TestJournal_DrawWaitsForItsPresentationBoundary` | done |
| `outermost_turn_seals_no_frame_work_when_idle` | test | 1923 |  | `TestJournal_OutermostTurnSealsNoFrameWorkWhenIdle` | done |
| `nested_turns_only_seal_after_the_outermost_turn` | test | 1964 |  | `TestJournal_NestedTurnsOnlySealAfterTheOutermostTurn` | done |
| `an_immediately_ready_runnable_prevents_an_idle_boundary_between_polls` | test | 2000 |  | `TestJournal_AnImmediatelyReadyRunnablePreventsAnIdleBoundaryBetweenPolls` | done |
| `a_pending_frame_prevents_idle_until_presentation` | test | 2035 |  | `TestJournal_APendingFramePreventsIdleUntilPresentation` | done |
| `a_present_without_a_draw_clears_the_window_pending_state` | test | 2067 |  | `TestJournal_APresentWithoutADrawClearsTheWindowPendingState` | done |
| `a_presentation_after_the_deadline_seals_the_whole_interval` | test | 2092 |  | `TestJournal_APresentationAfterTheDeadlineSealsTheWholeInterval` | done |
| `an_expired_frame_no_longer_blocks_idle_boundaries` | test | 2120 |  | `TestJournal_AnExpiredFrameNoLongerBlocksIdleBoundaries` | done |
| `closing_a_window_unblocks_idle_boundaries` | test | 2158 |  | `TestJournal_ClosingAWindowUnblocksIdleBoundaries` | done |
| `presenting_one_window_does_not_clear_another_pending_window` | test | 2183 |  | `TestJournal_PresentingOneWindowDoesNotClearAnotherPendingWindow` | done |
| `small_polls_are_flushed_immediately_before_a_retained_event` | test | 2221 |  | `TestJournal_SmallPollsAreFlushedImmediatelyBeforeARetainedEvent` | done |
| `small_polls_are_flushed_before_frame_state_changes` | test | 2268 |  | `TestJournal_SmallPollsAreFlushedBeforeFrameStateChanges` | done |
| `sparse_small_polls_are_discarded_when_the_foreground_returns_to_idle` | test | 2302 |  | `TestJournal_SparseSmallPollsAreDiscardedWhenTheForegroundReturnsToIdle` | done |
| `note_lost_is_reported_on_the_next_snapshot_only` | test | 2335 |  | `TestJournal_NoteLostIsReportedOnTheNextSnapshotOnly` | done |
| `losses_alone_seal_a_snapshot_at_the_next_boundary` | test | 2363 |  | `TestJournal_LossesAloneSealASnapshotAtTheNextBoundary` | done |
| `interval_event_cap_counts_overflowing_events` | test | 2379 |  | `TestJournal_IntervalEventCapCountsOverflowingEvents` | done |
| `collect_unseen_returns_each_entry_once_in_order` | test | 2403 |  | `TestJournal_CollectUnseenReturnsEachEntryOnceInOrder` | done |
| `concurrent_collection_preserves_the_complete_logical_sequence` | test | 2436 |  | `TestJournal_ConcurrentCollectionPreservesTheCompleteLogicalSequence` | done |
| `ring_matches_reference_model_under_wrap_collisions_and_independent_cursors` | proptest | 3117 |  | `TestJournal_RingMatchesReferenceModelUnderWrapCollisionsAndIndependentCursors` | dev:D-PROPTEST |
| `a_presentation_seals_exactly_once_regardless_of_delay` | proptest | 3219 |  | `TestJournal_APresentationSealsExactlyOnceRegardlessOfDelay` | dev:D-PROPTEST |
| `completion_order_sealer_matches_reference_under_arbitrary_batching` | proptest | 3245 |  | `TestJournal_CompletionOrderSealerMatchesReferenceUnderArbitraryBatching` | dev:D-PROPTEST |
| `occupancy_matches_interval_union_and_fold_apportionment` | proptest | 3283 |  | `TestJournal_OccupancyMatchesIntervalUnionAndFoldApportionment` | dev:D-PROPTEST |
| `writer_matches_state_model_across_turn_frame_and_runnable_transitions` | proptest | 3362 |  | `TestJournal_WriterMatchesStateModelAcrossTurnFrameAndRunnableTransitions` | dev:D-PROPTEST |

## gpui/src/profiler/hang/monitor.rs

Go target: `profiler_hang_monitor.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HangMonitorConfig` | struct | 12 |  | `HangMonitorConfig` | done |
| `HangMonitorError` | enum | 25 |  | `HangMonitorError` | done |
| `HangMonitorPollReason` | enum | 36 |  | `HangMonitorPollReason` | done |
| `HangMonitorPoll` | struct | 45 |  | `HangMonitorPoll` | done |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-PROFTHREAD | The thread-local `THREAD_TIMINGS` becomes a registry keyed by an explicit `ThreadID` (allocated with `NewThreadID`); `UpdateRunningTask`, `SaveTaskTiming`, `GetCurrentThreadTaskTimings` and `GetCurrentThreadTimings` take the dispatcher's `ThreadID` as first parameter, and `ReleaseThreadTimings` is the thread-exit drop | `gpui/src/profiler.rs:42`, `:59`, `:540-556`, `:671-697` | Go has no thread-local storage and goroutines have no stable identity; the dispatcher owns its thread identity |
| D-JOURNALTHREAD | The thread-local foreground journal and runnable counter become one process-wide journal behind a mutex; record calls from any goroutine reach it, and an unbalanced `end_turn` returns without the debug assertion (tests share the journal that earlier Apps installed, where upstream gives each test thread none) | `gpui/src/profiler/journal.rs:526-529`, `:727-731` | Go has no thread-local storage; the dispatchers must only journal the main thread's polls (handoff) |
| D-SUSPENDCLOCK | `system_suspended_time` is the package variable `systemSuspendedTime`, returning no clock until per-OS files set it (CLOCK_BOOTTIME - CLOCK_MONOTONIC on Linux, QueryInterruptTime - QueryUnbiasedInterruptTime on Windows, mach_continuous_time - mach_absolute_time on macOS) | `gpui/src/profiler/journal.rs:882-970` | One upstream file maps to one Go file and the root package cannot hold OS-specific code in profiler_journal.go; the per-OS bodies need files outside this stage (handoff) |
| D-BENCHMETRICS | The `bench_metrics` crate (not vendored in `_ref`) becomes `BenchMeasurement` / `MetricReport` / `Bencher` in app_bench_context.go: testing.B's wall time is the primary measurement, secondary counters are `BenchCounter` funcs; `BENCH_MEASUREMENT` accepts `default` (heap allocations and bytes from runtime/metrics instead of `CountingAllocator` and hardware counters) and `wall-time`; `bench_task` / `bench_batched_task` outputs and inputs are dropped by calling their `Release` method when they have one | `gpui/src/app/bench_context.rs:13`, `gpui/src/bench.rs:51` | Go cannot replace the global allocator or read perf counters without cgo; Criterion's `Bencher` is testing.B (D-BENCH) |
