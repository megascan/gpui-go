# Go symbol collisions produced by the naming rules

Each row is a Go symbol that two or more upstream items map to inside one Go package. Most are `#[cfg]` alternatives of the same item (one Go symbol, per-OS files) or overloads split by `cfg`; the rest must be disambiguated by the owning stage and recorded as `dev:D-NAME` in the area file.

| Area | Package | Go symbol | Upstream locations |
|---|---|---|---|
| foundation | gpuiutil | `NewStdCommand` | gpui_util/src/lib.rs:22, gpui_util/src/lib.rs:31 |
| sumtree | sumtree | `NewSumTree` | sum_tree/src/property_test.rs:25, sum_tree/src/sum_tree.rs:226 |
| core-geometry | gpui | `NewPoint` | gpui/src/geometry.rs:112, gpui/src/geometry.rs:132 |
| core-geometry | gpui | `NewSize` | gpui/src/geometry.rs:406, gpui/src/geometry.rs:426 |
| core-geometry | gpui | `NewBounds` | gpui/src/geometry.rs:732, gpui/src/geometry.rs:786 |
| core-app | gpui | `(App).LeakDetectorSnapshot` | gpui/src/app.rs:1258, gpui/src/app.rs:1280 |
| core-app | gpui | `(App).AssertNoNewLeaks` | gpui/src/app.rs:1274, gpui/src/app.rs:1287 |
| core-platform | gpui | `GraphicalEnvironment` | gpui/src/platform.rs:227, gpui/src/platform.rs:254, gpui/src/platform.rs:290 |
| core-platform | gpui | `GraphicalEnvironmentDetect` | gpui/src/platform.rs:263, gpui/src/platform.rs:295, gpui/src/platform.rs:323 |
| core-platform | gpui | `(GraphicalEnvironment).ApplyTo` | gpui/src/platform.rs:279, gpui/src/platform.rs:300, gpui/src/platform.rs:336 |
| core-platform | gpui | `(GraphicalEnvironment).DesktopEnvironment` | gpui/src/platform.rs:282, gpui/src/platform.rs:303, gpui/src/platform.rs:353 |
| core-elements | gpui | `NewInteractivity` | gpui/src/elements/div.rs:96, gpui/src/elements/div.rs:105 |
| core-elements | gpui | `(Interactive[S]).DebugSelector` | gpui/src/elements/div.rs:909, gpui/src/elements/div.rs:922 |
| core-profiler | gpui | `GetAllTimings` | gpui/src/profiler.rs:36, gpui/src/profiler.rs:54 |
| core-profiler | gpui | `GetCurrentThreadTimings` | gpui/src/profiler.rs:42, gpui/src/profiler.rs:59 |
| core-profiler | gpui | `TakeAllStats` | gpui/src/profiler.rs:48, gpui/src/profiler.rs:70 |
| core-profiler | gpui | `GlobalThreadTimings` | gpui/src/profiler.rs:419, gpui/src/profiler.rs:515 |
| core-profiler | gpui | `(ThreadTimings).UpdateRunningTask` | gpui/src/profiler.rs:580, gpui/src/profiler.rs:593 |
| core-profiler | gpui | `(ThreadTimings).SaveTaskTiming` | gpui/src/profiler.rs:596, gpui/src/profiler.rs:626 |
| core-profiler | gpui | `(ActionStatistics).UpdateRunningAction` | gpui/src/profiler/actions.rs:78, gpui/src/profiler/actions.rs:82 |
| core-profiler | gpui | `(ActionStatistics).SaveActionTiming` | gpui/src/profiler/actions.rs:85, gpui/src/profiler/actions.rs:131 |
| core-profiler | gpui | `TakeActionStats` | gpui/src/profiler/actions.rs:205, gpui/src/profiler/actions.rs:211 |
