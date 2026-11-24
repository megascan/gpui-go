# Parity: macos

Scope: gpui_macos. Milestone: M4. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_macos/src/display.rs

Go target: `platform/macos/display.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacDisplay` | struct | 16 |  | `MacDisplay` | done |
| `MacDisplay::find_by_id` | assoc-fn | 22 |  | `MacDisplayFindByID` | done |
| `MacDisplay::primary` | assoc-fn | 28 |  | `MacDisplayPrimary` | done |
| `MacDisplay::all` | assoc-fn | 50 |  | `MacDisplayAll` | done |
| `CGDisplayCreateUUIDFromDisplayID` | extern-fn | 77 |  | `cgDisplayCreateUUIDFromDisplayID` | done |
| `PlatformDisplay for MacDisplay` | impl | 80 |  | `(*MacDisplay).UUID` | done |
| `MacDisplay::get_nsscreen` | method | 166 |  | `(*MacDisplay).getNSScreen` | done |

## gpui_macos/src/display_link.rs

Go target: `platform/macos/display_link.go` (package `macos`), tests in `display_link_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MetalFrames` | struct | 305 |  | `MetalFrames` | done |
| `MetalFrames::replace` | method | 312 |  | `(MetalFrames).Replace` | done |
| `MetalFrames::is_attached` | method | 316 |  | `(MetalFrames).IsAttached` | done |
| `MetalFrames::wake` | method | 320 |  | `(MetalFrames).Wake` | done |
| `MetalFrames::begin_frame` | method | 330 |  | `(MetalFrames).BeginFrame` | done |
| `MetalFrames::debug_state` | method | 334 |  | `(MetalFrames).DebugState` | done |
| `MetalFrames::end_frame` | method | 350 |  | `(MetalFrames).EndFrame` | done |
| `WindowFrameSource` | struct | 362 |  | `WindowFrameSource` | done |
| `WindowFrameSource::new` | assoc-fn | 369 |  | `NewWindowFrameSource` | done |
| `WindowFrameSource::start` | method | 391 |  | `(WindowFrameSource).Start` | done |
| `WindowFrameSource::debug_state` | method | 399 |  | `(WindowFrameSource).DebugState` | done |
| `WindowFrameSource::take_signal` | method | 407 |  | `(WindowFrameSource).TakeSignal` | done |
| `WindowFrameSource::stop` | method | 411 |  | `(WindowFrameSource).Stop` | done |
| `CVDisplayLink` | enum | 446 |  | `CVDisplayLink` | done |
| `CVTimeStampFlags` | type | 479 |  | `CVTimeStampFlags` | done |
| `kCVTimeStampVideoTimeValid` | const | 481 |  | `Kcvtimestampvideotimevalid` | done |
| `kCVTimeStampHostTimeValid` | const | 482 |  | `Kcvtimestamphosttimevalid` | done |
| `kCVTimeStampSMPTETimeValid` | const | 483 |  | `Kcvtimestampsmptetimevalid` | done |
| `kCVTimeStampVideoRefreshPeriodValid` | const | 484 |  | `Kcvtimestampvideorefreshperiodvalid` | done |
| `kCVTimeStampRateScalarValid` | const | 485 |  | `Kcvtimestampratescalarvalid` | done |
| `kCVTimeStampTopField` | const | 486 |  | `Kcvtimestamptopfield` | done |
| `kCVTimeStampBottomField` | const | 487 |  | `Kcvtimestampbottomfield` | done |
| `kCVTimeStampVideoHostTimeValid` | const | 488 |  | `Kcvtimestampvideohosttimevalid` | done |
| `kCVTimeStampIsInterlaced` | const | 490 |  | `Kcvtimestampisinterlaced` | done |
| `CVSMPTETimeType` | type | 507 |  | `CVSMPTETimeType` | done |
| `kCVSMPTETimeType24` | const | 509 |  | `Kcvsmptetimetype24` | done |
| `kCVSMPTETimeType25` | const | 510 |  | `Kcvsmptetimetype25` | done |
| `kCVSMPTETimeType30Drop` | const | 511 |  | `Kcvsmptetimetype30drop` | done |
| `kCVSMPTETimeType30` | const | 512 |  | `Kcvsmptetimetype30` | done |
| `kCVSMPTETimeType2997` | const | 513 |  | `Kcvsmptetimetype2997` | done |
| `kCVSMPTETimeType2997Drop` | const | 514 |  | `Kcvsmptetimetype2997drop` | done |
| `kCVSMPTETimeType60` | const | 515 |  | `Kcvsmptetimetype60` | done |
| `kCVSMPTETimeType5994` | const | 516 |  | `Kcvsmptetimetype5994` | done |
| `CVSMPTETimeFlags` | type | 518 |  | `CVSMPTETimeFlags` | done |
| `kCVSMPTETimeValid` | const | 520 |  | `Kcvsmptetimevalid` | done |
| `kCVSMPTETimeRunning` | const | 521 |  | `Kcvsmptetimerunning` | done |
| `CVDisplayLinkOutputCallback` | type | 523 |  | `CVDisplayLinkOutputCallback` | done |
| `DisplayLink::new` | assoc-fn | 561 |  | `NewDisplayLink` | done |
| `DisplayLinkRef::start` | method | 591 |  | `(DisplayLinkRef).Start` | done |
| `DisplayLinkRef::stop` | method | 600 |  | `(DisplayLinkRef).Stop` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `metal_frames_pause_only_when_gpui_neither_draws_nor_asks_again` | test | 617 |  | `TestDisplayLink_MetalFramesPauseOnlyWhenGpuiNeitherDrawsNorAsksAgain` | done |
| `a_renderer_draws_normally_after_its_display_link_is_invalidated` | test | 645 |  | `TestDisplayLink_ARendererDrawsNormallyAfterItsDisplayLinkIsInvalidated` | blocked:M4-apple-metal-renderer |
| `host_timestamp_preserves_request_age_at_different_clock_rates` | test | 665 |  | `TestDisplayLink_HostTimestampPreservesRequestAgeAtDifferentClockRates` | done |
| `invalid_host_timestamps_fall_back_to_receipt_time` | test | 680 |  | `TestDisplayLink_InvalidHostTimestampsFallBackToReceiptTime` | done |

## gpui_macos/src/events.rs

Go target: `platform/macos/events.go` (package `macos`), tests in `events_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ESCAPE_KEY` | const | 30 |  | `escapeKey` | done |
| `key_to_menu_equivalent` | fn | 46 |  | `keyToMenuEquivalent` | done |
| `read_modifiers` | fn | 101 |  | `readModifiers` | done |
| `platform_input_from_native` | fn | 118 |  | `platformInputFromNative` | done |
| `parse_keystroke` | fn | 348 |  | `parseKeystroke` | done |
| `always_use_command_layout` | fn | 504 |  | `alwaysUseCommandLayout` | done |
| `chars_for_modified_key` | fn | 517 |  | `charsForModifiedKey` | done |

## gpui_macos/src/gpui_macos.rs

Go target: `platform/macos/gpui_macos.go` (package `macos`), tests in `gpui_macos_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `metal_renderer::SurfaceBounds` | re-export | 20 |  | `SurfaceBounds` | blocked:M4-apple-metal-renderer |
| `metal_renderer::MetalHeadlessRenderer` | re-export | 23 |  | `MetalHeadlessRenderer` | blocked:M4-apple-metal-renderer |
| `MacPlatform` | re-export | 45 |  | `MacPlatform` | blocked:M4-macos-platform |
| `BoolExt` | trait | 47 |  | - | n/a:D-MACOS-OBJCBOOL |
| `NSStringExt::to_str` | method | 62 |  | `nsStringToStr` | done |
| `NSRange` | struct | 79 |  | `NSRange` | done |
| `NSRange::invalid` | assoc-fn | 85 |  | `nsRangeInvalid` | done |
| `NSRange::is_valid` | method | 92 |  | `(NSRange).isValid` | done |
| `NSRange::to_range` | method | 96 |  | `(NSRange).toRange` | done |
| `From<Range<usize>> for NSRange` | impl | 107 |  | `nsRangeFromRange` | done |
| `ns_string` | fn | 124 |  | `nsString` | done |

## gpui_macos/src/keyboard.rs

Go target: `platform/macos/keyboard.go` (package `macos`), tests in `keyboard_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacKeyboardLayout` | struct | 13 |  | `MacKeyboardLayout` | done |
| `MacKeyboardMapper` | struct | 18 |  | `MacKeyboardMapper` | done |
| `MacKeyboardLayout::id` | method | 23 |  | `(MacKeyboardLayout).ID` | done |
| `MacKeyboardLayout::name` | method | 27 |  | `(MacKeyboardLayout).Name` | done |
| `MacKeyboardMapper::map_key_equivalent` | method | 33 |  | `(MacKeyboardMapper).MapKeyEquivalent` | done |
| `MacKeyboardMapper::get_key_equivalents` | method | 48 |  | `(MacKeyboardMapper).GetKeyEquivalents` | done |
| `MacKeyboardLayout::new` | assoc-fn | 54 |  | `NewMacKeyboardLayout` | done |
| `MacKeyboardMapper::new` | assoc-fn | 86 |  | `NewMacKeyboardMapper` | done |
| `get_key_equivalents` | fn | 117 |  | `getKeyEquivalents` | done |

## gpui_macos/src/pasteboard.rs

Go target: `platform/macos/pasteboard.go` (package `macos`), tests in `pasteboard_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Pasteboard` | struct | 24 |  | `Pasteboard` | done |
| `Pasteboard::general` | assoc-fn | 31 |  | `PasteboardGeneral` | done |
| `Pasteboard::find` | assoc-fn | 37 |  | `PasteboardFind` | done |
| `Pasteboard::read` | method | 59 |  | `(Pasteboard).Read` | done |
| `Pasteboard::write` | method | 162 |  | `(Pasteboard).Write` | done |
| `UTType` | struct | 253 |  | `UTType` | done |
| `UTType::png` | assoc-fn | 256 |  | `UTTypePng` | done |
| `UTType::jpeg` | assoc-fn | 261 |  | `UTTypeJpeg` | done |
| `UTType::gif` | assoc-fn | 266 |  | `UTTypeGif` | done |
| `UTType::webp` | assoc-fn | 271 |  | `UTTypeWebp` | done |
| `UTType::bmp` | assoc-fn | 276 |  | `UTTypeBmp` | done |
| `UTType::svg` | assoc-fn | 281 |  | `UTTypeSvg` | done |
| `UTType::ico` | assoc-fn | 286 |  | `UTTypeIco` | done |
| `UTType::tiff` | assoc-fn | 291 |  | `UTTypeTiff` | done |
| `UTType::pnm` | assoc-fn | 296 |  | `UTTypePnm` | done |
| `From<ImageFormat> for UTType` | impl | 236 |  | `UTTypeFrom` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_string` | test | 314 |  | `TestPasteboard_String` | done |
| `test_custom_types_survive_creation_autorelease_pool` | test | 350 |  | `TestPasteboard_CustomTypesSurviveCreationAutoreleasePool` | done |
| `test_read_external_path` | test | 367 |  | `TestPasteboard_ReadExternalPath` | done |
| `test_read_external_paths_with_spaces` | test | 397 |  | `TestPasteboard_ReadExternalPathsWithSpaces` | done |
| `test_read_multiple_external_paths` | test | 416 |  | `TestPasteboard_ReadMultipleExternalPaths` | done |
| `test_read_image` | test | 448 |  | `TestPasteboard_ReadImage` | blocked:M0-app-contexts |

## gpui_macos/src/platform.rs

Go target: `platform/macos/platform.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacPlatform` | struct | 167 |  | `MacPlatform` | dev:D-MACOS-PLATFORMIVAR |
| `MacPlatform::new` | assoc-fn | 219 |  | `NewMacPlatform` | dev:D-FEATURE |
| `TISCopyCurrentKeyboardLayoutInputSource` | extern-fn | 1654 |  | `tisCopyCurrentKeyboardLayoutInputSource` | dev:D-MACOS-CARBON |
| `TISCopyCurrentKeyboardInputSource` | extern-fn | 1655 |  | `tisCopyCurrentKeyboardInputSource` | dev:D-MACOS-CARBON |
| `TISGetInputSourceProperty` | extern-fn | 1656 |  | `tisGetInputSourceProperty` | dev:D-MACOS-CARBON |
| `UCKeyTranslate` | extern-fn | 1661 |  | `ucKeyTranslate` | dev:D-MACOS-CARBON |
| `LMGetKbdType` | extern-fn | 1673 |  | `lmGetKbdType` | dev:D-MACOS-CARBON |
| `kTISPropertyUnicodeKeyLayoutData` | static | 1674 |  | `kTISPropertyUnicodeKeyLayoutData` | dev:D-MACOS-CARBON |
| `kTISPropertyInputSourceID` | static | 1675 |  | `kTISPropertyInputSourceID` | dev:D-MACOS-CARBON |
| `kTISPropertyLocalizedName` | static | 1676 |  | `kTISPropertyLocalizedName` | dev:D-MACOS-CARBON |
| `kTISPropertyInputSourceIsASCIICapable` | static | 1677 |  | `kTISPropertyInputSourceIsASCIICapable` | dev:D-MACOS-CARBON |
| `kTISPropertyInputSourceType` | static | 1678 |  | `kTISPropertyInputSourceType` | dev:D-MACOS-CARBON |
| `kTISTypeKeyboardInputMode` | static | 1679 |  | `kTISTypeKeyboardInputMode` | dev:D-MACOS-CARBON |

## gpui_macos/src/screen_capture.rs

Go target: `platform/macos/screen_capture.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `MacScreenCaptureSource` | struct | 25 |  | `MacScreenCaptureSource` | done |
| `MacScreenCaptureStream` | struct | 30 |  | `MacScreenCaptureStream` | dev:D-MACOS-SCKFRAME |

## gpui_macos/src/system_notifications.rs

Go target: `platform/macos/system_notifications.go` (package `macos`), tests in `system_notifications_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_notification_commands_do_not_wait_for_the_service` | gpui::test | 361 |  | `TestSystemNotifications_NotificationCommandsDoNotWaitForTheService` | done |

## gpui_macos/src/window.rs

Go target: `platform/macos/window.go` (package `macos`), tests in `window_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `UserTabbingPreference` | enum | 97 |  | `UserTabbingPreference` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `display_id_for_screen_handles_missing_metadata` | test | 4298 |  | `TestWindow_DisplayIDForScreenHandlesMissingMetadata` | blocked:M4-macos-window-objc |

## gpui_macos/src/window_appearance.rs

Go target: `platform/macos/window_appearance.go` (package `macos`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `window_appearance_from_native` | fn | 7 |  | `windowAppearanceFromNative` | done |
| `NSAppearanceNameAqua` | static | 34 |  | `windowAppearanceFromNative` | done |
| `NSAppearanceNameDarkAqua` | static | 35 |  | `windowAppearanceFromNative` | done |

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


## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-MACOS-OBJCBOOL | `BoolExt::to_objc` has no Go symbol: Go `bool` is passed to and returned from `objc.Send` directly and purego marshals it as Objective-C `BOOL` | `gpui_macos/src/gpui_macos.rs:47-55` | objc2's `Bool` newtype has no Go counterpart |
| D-MACOS-CARBON | The Carbon / HIToolbox externs of `platform.rs` (TIS*, UCKeyTranslate, LMGetKbdType, kTIS* statics) are lazily resolved purego bindings in `platform.go`, loaded by `loadCarbon`; `CFDataGetBytePtr` is bound there too | `gpui_macos/src/platform.rs:1652-1680` | `#[link(name = "Carbon")]` has no purego counterpart; resolved on first use instead of at link time |
| D-MACOS-PLATFORMIVAR | The `platform` ivar of `GPUIApplication` / `GPUIApplicationDelegate` is a package map from the object to its `*MacPlatform` (`setMacPlatformIvar` / `getMacPlatform`, same panic messages); the classes are registered lazily by `NewMacPlatform` instead of a `#[ctor]` | `gpui_macos/src/platform.rs:60-163,1324-1346` | Go pointers cannot be stored in Objective-C ivars |
| D-MACOS-SCKFRAME | `ScreenCaptureFrame.Frame` is the sample buffer's CVImageBufferRef borrowed for the duration of the frame callback, not a retained `CVImageBuffer` released on drop | `gpui_macos/src/screen_capture.rs:264-277` | Go has no drop to release a retained frame; callers that keep a frame retain it |
