# Parity: macos

Scope: gpui_macos. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_macos/src/display_link.rs

Go target: `platform/macos/display_link.go` (package `macos`), tests in `display_link_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalFrames` | struct | 305 |  | `MetalFrames` | todo |
| `MetalFrames::replace` | method | 312 |  | `(MetalFrames).Replace` | todo |
| `MetalFrames::is_attached` | method | 316 |  | `(MetalFrames).IsAttached` | todo |
| `MetalFrames::wake` | method | 320 |  | `(MetalFrames).Wake` | todo |
| `MetalFrames::begin_frame` | method | 330 |  | `(MetalFrames).BeginFrame` | todo |
| `MetalFrames::debug_state` | method | 334 |  | `(MetalFrames).DebugState` | todo |
| `MetalFrames::end_frame` | method | 350 |  | `(MetalFrames).EndFrame` | todo |
| `WindowFrameSource` | struct | 362 |  | `WindowFrameSource` | todo |
| `WindowFrameSource::new` | assoc-fn | 369 |  | `NewWindowFrameSource` | todo |
| `WindowFrameSource::start` | method | 391 |  | `(WindowFrameSource).Start` | todo |
| `WindowFrameSource::debug_state` | method | 399 |  | `(WindowFrameSource).DebugState` | todo |
| `WindowFrameSource::take_signal` | method | 407 |  | `(WindowFrameSource).TakeSignal` | todo |
| `WindowFrameSource::stop` | method | 411 |  | `(WindowFrameSource).Stop` | todo |
| `CVDisplayLink` | enum | 446 |  | `CVDisplayLink` | todo |
| `CVTimeStampFlags` | type | 479 |  | `CVTimeStampFlags` | todo |
| `kCVTimeStampVideoTimeValid` | const | 481 |  | `Kcvtimestampvideotimevalid` | todo |
| `kCVTimeStampHostTimeValid` | const | 482 |  | `Kcvtimestamphosttimevalid` | todo |
| `kCVTimeStampSMPTETimeValid` | const | 483 |  | `Kcvtimestampsmptetimevalid` | todo |
| `kCVTimeStampVideoRefreshPeriodValid` | const | 484 |  | `Kcvtimestampvideorefreshperiodvalid` | todo |
| `kCVTimeStampRateScalarValid` | const | 485 |  | `Kcvtimestampratescalarvalid` | todo |
| `kCVTimeStampTopField` | const | 486 |  | `Kcvtimestamptopfield` | todo |
| `kCVTimeStampBottomField` | const | 487 |  | `Kcvtimestampbottomfield` | todo |
| `kCVTimeStampVideoHostTimeValid` | const | 488 |  | `Kcvtimestampvideohosttimevalid` | todo |
| `kCVTimeStampIsInterlaced` | const | 490 |  | `Kcvtimestampisinterlaced` | todo |
| `CVSMPTETimeType` | type | 507 |  | `CVSMPTETimeType` | todo |
| `kCVSMPTETimeType24` | const | 509 |  | `Kcvsmptetimetype24` | todo |
| `kCVSMPTETimeType25` | const | 510 |  | `Kcvsmptetimetype25` | todo |
| `kCVSMPTETimeType30Drop` | const | 511 |  | `Kcvsmptetimetype30drop` | todo |
| `kCVSMPTETimeType30` | const | 512 |  | `Kcvsmptetimetype30` | todo |
| `kCVSMPTETimeType2997` | const | 513 |  | `Kcvsmptetimetype2997` | todo |
| `kCVSMPTETimeType2997Drop` | const | 514 |  | `Kcvsmptetimetype2997drop` | todo |
| `kCVSMPTETimeType60` | const | 515 |  | `Kcvsmptetimetype60` | todo |
| `kCVSMPTETimeType5994` | const | 516 |  | `Kcvsmptetimetype5994` | todo |
| `CVSMPTETimeFlags` | type | 518 |  | `CVSMPTETimeFlags` | todo |
| `kCVSMPTETimeValid` | const | 520 |  | `Kcvsmptetimevalid` | todo |
| `kCVSMPTETimeRunning` | const | 521 |  | `Kcvsmptetimerunning` | todo |
| `CVDisplayLinkOutputCallback` | type | 523 |  | `CVDisplayLinkOutputCallback` | todo |
| `DisplayLink::new` | assoc-fn | 561 |  | `NewDisplayLink` | todo |
| `DisplayLinkRef::start` | method | 591 |  | `(DisplayLinkRef).Start` | todo |
| `DisplayLinkRef::stop` | method | 600 |  | `(DisplayLinkRef).Stop` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `metal_frames_pause_only_when_gpui_neither_draws_nor_asks_again` | test | 617 |  | `TestDisplayLink_MetalFramesPauseOnlyWhenGpuiNeitherDrawsNorAsksAgain` | todo |
| `a_renderer_draws_normally_after_its_display_link_is_invalidated` | test | 645 |  | `TestDisplayLink_ARendererDrawsNormallyAfterItsDisplayLinkIsInvalidated` | todo |
| `host_timestamp_preserves_request_age_at_different_clock_rates` | test | 665 |  | `TestDisplayLink_HostTimestampPreservesRequestAgeAtDifferentClockRates` | todo |
| `invalid_host_timestamps_fall_back_to_receipt_time` | test | 680 |  | `TestDisplayLink_InvalidHostTimestampsFallBackToReceiptTime` | todo |

## gpui_macos/src/pasteboard.rs

Go target: `platform/macos/pasteboard.go` (package `macos`), tests in `pasteboard_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Pasteboard` | struct | 24 |  | `Pasteboard` | todo |
| `Pasteboard::general` | assoc-fn | 31 |  | `PasteboardGeneral` | todo |
| `Pasteboard::find` | assoc-fn | 37 |  | `PasteboardFind` | todo |
| `Pasteboard::read` | method | 59 |  | `(Pasteboard).Read` | todo |
| `Pasteboard::write` | method | 162 |  | `(Pasteboard).Write` | todo |
| `UTType` | struct | 253 |  | `UTType` | todo |
| `UTType::png` | assoc-fn | 256 |  | `UTTypePng` | todo |
| `UTType::jpeg` | assoc-fn | 261 |  | `UTTypeJpeg` | todo |
| `UTType::gif` | assoc-fn | 266 |  | `UTTypeGif` | todo |
| `UTType::webp` | assoc-fn | 271 |  | `UTTypeWebp` | todo |
| `UTType::bmp` | assoc-fn | 276 |  | `UTTypeBmp` | todo |
| `UTType::svg` | assoc-fn | 281 |  | `UTTypeSvg` | todo |
| `UTType::ico` | assoc-fn | 286 |  | `UTTypeIco` | todo |
| `UTType::tiff` | assoc-fn | 291 |  | `UTTypeTiff` | todo |
| `UTType::pnm` | assoc-fn | 296 |  | `UTTypePnm` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_string` | test | 314 |  | `TestPasteboard_String` | todo |
| `test_custom_types_survive_creation_autorelease_pool` | test | 350 |  | `TestPasteboard_CustomTypesSurviveCreationAutoreleasePool` | todo |
| `test_read_external_path` | test | 367 |  | `TestPasteboard_ReadExternalPath` | todo |
| `test_read_external_paths_with_spaces` | test | 397 |  | `TestPasteboard_ReadExternalPathsWithSpaces` | todo |
| `test_read_multiple_external_paths` | test | 416 |  | `TestPasteboard_ReadMultipleExternalPaths` | todo |
| `test_read_image` | test | 448 |  | `TestPasteboard_ReadImage` | todo |

## gpui_macos/src/platform.rs

Go target: `platform/macos/platform.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacPlatform` | struct | 167 |  | `MacPlatform` | todo |
| `MacPlatform::new` | assoc-fn | 219 |  | `NewMacPlatform` | todo |

## gpui_macos/src/screen_capture.rs

Go target: `platform/macos/screen_capture.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacScreenCaptureSource` | struct | 25 |  | `MacScreenCaptureSource` | todo |
| `MacScreenCaptureStream` | struct | 30 |  | `MacScreenCaptureStream` | todo |

## gpui_macos/src/system_notifications.rs

Go target: `platform/macos/system_notifications.go` (package `macos`), tests in `system_notifications_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_notification_commands_do_not_wait_for_the_service` | gpui::test | 361 |  | `TestSystemNotifications_NotificationCommandsDoNotWaitForTheService` | todo |

## gpui_macos/src/window.rs

Go target: `platform/macos/window.go` (package `macos`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `UserTabbingPreference` | enum | 97 |  | `UserTabbingPreference` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `display_id_for_screen_handles_missing_metadata` | test | 4298 |  | `TestWindow_DisplayIDForScreenHandlesMissingMetadata` | todo |

## gpui_macos/src/window/native_views.rs

Go target: `platform/macos/window_native_views.go` (package `macos`), tests in `window_native_views_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `placement_clips_a_view_to_its_content_mask` | test | 1441 |  | `TestNativeViews_PlacementClipsAViewToItsContentMask` | todo |
| `keyboard_input_follows_gpui_focus_except_from_views_gpui_did_not_attach` | test | 1476 |  | `TestNativeViews_KeyboardInputFollowsGpuiFocusExceptFromViewsGpuiDidNotAttach` | todo |
| `views_take_keyboard_input_only_when_gpui_focus_a_click_or_voice_over_allows` | test | 1508 |  | `TestNativeViews_ViewsTakeKeyboardInputOnlyWhenGpuiFocusAClickOrVoiceOverAllows` | todo |
| `a_view_that_declines_keyboard_input_leaves_it_with_the_gpui_view` | test | 1540 |  | `TestNativeViews_AViewThatDeclinesKeyboardInputLeavesItWithTheGpuiView` | todo |
| `gpui_is_owed_the_release_of_each_button_pressed_in_a_native_view` | test | 1565 |  | `TestNativeViews_GpuiIsOwedTheReleaseOfEachButtonPressedInANativeView` | todo |
| `stacking_puts_shown_views_between_the_gpui_view_and_everything_else` | test | 1578 |  | `TestNativeViews_StackingPutsShownViewsBetweenTheGpuiViewAndEverythingElse` | todo |

