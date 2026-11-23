# Parity: ios

Scope: gpui_ios. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_ios/src/gpui_ios.rs

Go target: `platform/ios/gpui_ios_impl.go` (package `ios`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `StatusBarContentStyle` | enum | 15 |  | `StatusBarContentStyle` | todo |

## gpui_ios/src/ios.rs

Go target: `platform/ios/ios.go` (package `ios`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `current_platform` | fn | 56 |  | `CurrentPlatform` | todo |

## gpui_ios/src/ios/application.rs

Go target: `platform/ios/application.go` (package `ios`), tests in `application_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `browsing_activity_delivers_url_without_losing_query_or_fragment` | test | 417 |  | `TestApplication_BrowsingActivityDeliversURLWithoutLosingQueryOrFragment` | todo |
| `launch_callback_runs_once_and_can_register_callbacks` | test | 449 |  | `TestApplication_LaunchCallbackRunsOnceAndCanRegisterCallbacks` | todo |
| `lifecycle_callback_replacement_survives_dispatch` | test | 467 |  | `TestApplication_LifecycleCallbackReplacementSurvivesDispatch` | todo |
| `memory_warning_callback_is_restored_after_dispatch` | test | 493 |  | `TestApplication_MemoryWarningCallbackIsRestoredAfterDispatch` | todo |
| `lifecycle_removal_survives_reentrant_dispatch` | test | 511 |  | `TestApplication_LifecycleRemovalSurvivesReentrantDispatch` | todo |
| `application_callbacks_are_not_shared_between_platforms` | test | 530 |  | `TestApplication_ApplicationCallbacksAreNotSharedBetweenPlatforms` | todo |

## gpui_ios/src/ios/events.rs

Go target: `platform/ios/events.go` (package `ios`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `back_event` | fn | 23 |  | `BackEvent` | todo |
| `touch_location_in_view` | fn | 63 |  | `TouchLocationInView` | todo |
| `touch_phase` | fn | 68 |  | `TouchPhase` | todo |
| `touch_id` | fn | 77 |  | `TouchID` | todo |
| `scroll_event` | fn | 87 |  | `ScrollEvent` | todo |
| `touch_force` | fn | 122 |  | `TouchForce` | todo |

## gpui_ios/src/ios/platform.rs

Go target: `platform/ios/platform.go` (package `ios`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `IosPlatform` | struct | 47 |  | `IosPlatform` | todo |
| `IosPlatform::new` | assoc-fn | 68 |  | `NewIosPlatform` | todo |

## gpui_ios/src/ios/text_input.rs

Go target: `platform/ios/text_input.go` (package `ios`), tests in `text_input_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `insertion_geometry_has_a_visible_caret_width` | test | 722 |  | `TestTextInput_InsertionGeometryHasAVisibleCaretWidth` | todo |
| `utf16_ranges_and_offsets` | test | 1093 |  | `TestTextInput_UTF16RangesAndOffsets` | todo |

## gpui_ios/src/ios/window.rs

Go target: `platform/ios/window.go` (package `ios`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `set_status_bar_style` | fn | 122 |  | `SetStatusBarStyle` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `callback_is_unavailable_during_dispatch_and_restored_afterward` | test | 1056 |  | `TestWindow_CallbackIsUnavailableDuringDispatchAndRestoredAfterward` | todo |
| `callback_replacement_during_dispatch_is_preserved` | test | 1074 |  | `TestWindow_CallbackReplacementDuringDispatchIsPreserved` | todo |
| `withdrawing_an_in_flight_handler_does_not_restore_it` | test | 1085 |  | `TestWindow_WithdrawingAnInFlightHandlerDoesNotRestoreIt` | todo |
| `removing_a_replacement_does_not_resurrect_the_previous_handler` | test | 1096 |  | `TestWindow_RemovingAReplacementDoesNotResurrectThePreviousHandler` | todo |

