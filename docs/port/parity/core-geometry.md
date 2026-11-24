# Parity: core-geometry

Scope: gpui: geometry, color, bounds_tree, util, arena, shared_uri. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/arena.rs

Go target: `arena.go` (package `gpui`), tests in `arena_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Arena` | struct | 81 |  | `Arena` | n/a:D-ARENA |
| `Arena::new` | assoc-fn | 97 |  | `NewArena` | n/a:D-ARENA |
| `Arena::capacity` | method | 109 |  | `(Arena).Capacity` | n/a:D-ARENA |
| `Arena::begin_scope` | method | 116 |  | `(Arena).BeginScope` | n/a:D-ARENA |
| `Arena::end_scope` | method | 126 |  | `(Arena).EndScope` | n/a:D-ARENA |
| `Arena::clear` | method | 139 |  | `(Arena).Clear` | n/a:D-ARENA |
| `Arena::alloc` | method | 161 |  | `(Arena).Alloc` | n/a:D-ARENA |
| `ArenaBox` | struct | 218 |  | `ArenaBox` | n/a:D-ARENA |
| `ArenaBox::map` | method | 225 |  | `(ArenaBox).Map` | n/a:D-ARENA |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_arena` | test | 266 |  | `TestArena_Arena` | n/a:D-ARENA |
| `test_arena_grow` | test | 301 |  | `TestArena_ArenaGrow` | n/a:D-ARENA |
| `test_arena_alignment` | test | 315 |  | `TestArena_ArenaAlignment` | n/a:D-ARENA |
| `test_arena_use_after_clear` | test | 335 |  | `TestArena_ArenaUseAfterClear` | n/a:D-ARENA |
| `test_clear_deferred_while_scope_active` | test | 344 |  | `TestArena_ClearDeferredWhileScopeActive` | n/a:D-ARENA |
| `test_clear_without_scope_is_immediate` | test | 385 |  | `TestArena_ClearWithoutScopeIsImmediate` | n/a:D-ARENA |
| `test_unbalanced_end_scope_panics` | test | 395 |  | `TestArena_UnbalancedEndScopePanics` | n/a:D-ARENA |

## gpui/src/bounds_tree.rs

Go target: `bounds_tree.go` (package `gpui`), tests in `bounds_tree_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_insert` | test | 398 |  | `TestBoundsTree_Insert` | done |
| `test_random_iterations` | test | 456 |  | `TestBoundsTree_RandomIterations` | dev:D-RNG |

## gpui/src/color.rs

Go target: `color.go` (package `gpui`), tests in `color_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `rgb` | fn | 14 |  | `Rgb` | done |
| `rgba` | fn | 20 |  | `NewRgba` | done |
| `swap_rgba_pa_to_bgra` | fn | 26 |  | `SwapRgbaPaToBgra` | done |
| `Rgba` | struct | 39 |  | `Rgba` | done |
| `Rgba::blend` | method | 58 |  | `(Rgba).Blend` | done |
| `Rgba::alpha` | method | 95 |  | `(Rgba).Alpha` | done |
| `Rgba::opacity` | method | 126 |  | `(Rgba).Opacity` | done |
| `Hsla` | struct | 334 |  | `Hsla` | done |
| `Hsla::opaque_strategy` | assoc-fn | 357 | feat:proptest | `HslaOpaqueStrategy` | dev:D-PROPTEST |
| `hsla` | fn | 424 |  | `NewHsla` | done |
| `black` | fn | 434 |  | `Black` | done |
| `transparent_black` | fn | 444 |  | `TransparentBlack` | done |
| `transparent_white` | fn | 454 |  | `TransparentWhite` | done |
| `opaque_grey` | fn | 464 |  | `OpaqueGrey` | done |
| `white` | fn | 474 |  | `White` | done |
| `red` | fn | 484 |  | `Red` | done |
| `blue` | fn | 494 |  | `Blue` | done |
| `green` | fn | 504 |  | `Green` | done |
| `yellow` | fn | 514 |  | `Yellow` | done |
| `Hsla::to_rgb` | method | 525 |  | `(Hsla).ToRgb` | done |
| `Hsla::red` | assoc-fn | 530 |  | `HslaRed` | done |
| `Hsla::green` | assoc-fn | 535 |  | `HslaGreen` | done |
| `Hsla::blue` | assoc-fn | 540 |  | `HslaBlue` | done |
| `Hsla::black` | assoc-fn | 545 |  | `HslaBlack` | done |
| `Hsla::white` | assoc-fn | 550 |  | `HslaWhite` | done |
| `Hsla::transparent_black` | assoc-fn | 555 |  | `HslaTransparentBlack` | done |
| `Hsla::is_transparent` | method | 560 |  | `(Hsla).IsTransparent` | done |
| `Hsla::is_opaque` | method | 565 |  | `(Hsla).IsOpaque` | done |
| `Hsla::blend` | method | 580 |  | `(Hsla).Blend` | done |
| `Hsla::grayscale` | method | 596 |  | `(Hsla).Grayscale` | done |
| `Hsla::fade_out` | method | 607 |  | `(Hsla).FadeOut` | done |
| `Hsla::opacity` | method | 637 |  | `(Hsla).Opacity` | done |
| `Hsla::alpha` | method | 667 |  | `(Hsla).Alpha` | done |
| `ColorSpace` | enum | 759 |  | `ColorSpace` | done |
| `Background` | struct | 779 |  | `Background` | done |
| `pattern_slash` | fn | 827 |  | `PatternSlash` | done |
| `checkerboard` | fn | 841 |  | `Checkerboard` | done |
| `solid_background` | fn | 851 |  | `SolidBackground` | done |
| `linear_gradient` | fn | 865 |  | `LinearGradient` | done |
| `LinearColorStop` | struct | 883 |  | `LinearColorStop` | done |
| `linear_color_stop` | fn | 893 |  | `NewLinearColorStop` | done |
| `LinearColorStop::opacity` | method | 902 |  | `(LinearColorStop).Opacity` | done |
| `Background::as_solid` | method | 912 |  | `(Background).AsSolid` | done |
| `Background::color_space` | method | 923 |  | `(Background).ColorSpace` | done |
| `Background::opacity` | method | 929 |  | `(Background).Opacity` | done |
| `Background::is_transparent` | method | 940 |  | `(Background).IsTransparent` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_deserialize_three_value_hex_to_rgba` | test | 977 |  | `TestColor_DeserializeThreeValueHexToRgba` | done |
| `test_deserialize_four_value_hex_to_rgba` | test | 984 |  | `TestColor_DeserializeFourValueHexToRgba` | done |
| `test_deserialize_six_value_hex_to_rgba` | test | 991 |  | `TestColor_DeserializeSixValueHexToRgba` | done |
| `test_deserialize_eight_value_hex_to_rgba` | test | 998 |  | `TestColor_DeserializeEightValueHexToRgba` | done |
| `test_deserialize_eight_value_hex_with_padding_to_rgba` | test | 1005 |  | `TestColor_DeserializeEightValueHexWithPaddingToRgba` | done |
| `test_deserialize_eight_value_hex_with_mixed_case_to_rgba` | test | 1012 |  | `TestColor_DeserializeEightValueHexWithMixedCaseToRgba` | done |
| `test_background_solid` | test | 1019 |  | `TestColor_BackgroundSolid` | done |
| `test_background_linear_gradient` | test | 1032 |  | `TestColor_BackgroundLinearGradient` | done |
| `test_rgba_alpha` | test | 1047 |  | `TestColor_RgbaAlpha` | done |
| `test_rgba_opacity` | test | 1060 |  | `TestColor_RgbaOpacity` | done |

## gpui/src/colors.rs

Go target: `colors.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Colors` | struct | 10 |  | `Colors` | done |
| `Colors::for_appearance` | assoc-fn | 37 |  | `ColorsForAppearance` | done |
| `Colors::dark` | assoc-fn | 45 |  | `ColorsDark` | done |
| `Colors::light` | assoc-fn | 59 |  | `ColorsLight` | done |
| `Colors::get_global` | assoc-fn | 73 |  | `ColorsGetGlobal` | done |
| `GlobalColors` | struct | 80 |  | `GlobalColors` | done |
| `DefaultColors` | trait | 93 |  | `DefaultColors` | done |
| `DefaultColors::default_colors` | trait-method | 95 |  | `DefaultColors.DefaultColors` | done |
| `DefaultAppearance` | enum | 108 |  | `DefaultAppearance` | done |

## gpui/src/geometry.rs

Go target: `geometry.go` (package `gpui`), tests in `geometry_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Axis` | enum | 26 |  | `Axis` | done |
| `Axis::invert` | method | 35 |  | `(Axis).Invert` | done |
| `Along` | trait | 44 |  | `Along` | done |
| `Along::along` | trait-method | 49 |  | `Along.Along` | done |
| `Along::apply_along` | trait-method | 52 |  | `Along.ApplyAlong` | done |
| `Point` | struct | 86 |  | `Point` | done |
| `point` | fn | 112 |  | `NewPoint` | done |
| `Point::new` | assoc-fn | 132 |  | `NewPoint` | done |
| `Point::map` | method | 155 |  | `(Point).Map` | done |
| `Point<Pixels>::scale` | method | 203 |  | `(Point).Scale` | done |
| `Point<Pixels>::magnitude` | method | 219 |  | `(Point).Magnitude` | done |
| `Point::relative_to` | method | 229 |  | `(Point).RelativeTo` | done |
| `Point::max` | method | 297 |  | `(Point).Max` | done |
| `Point::min` | method | 327 |  | `(Point).Min` | done |
| `Point::clamp` | method | 368 |  | `(Point).Clamp` | done |
| `Size` | struct | 397 |  | `Size` | done |
| `Size::new` | assoc-fn | 406 |  | `NewSize` | done |
| `size` | fn | 426 |  | `NewSize` | done |
| `Size::map` | method | 455 |  | `(Size).Map` | done |
| `Size::center` | method | 471 |  | `(Size).Center` | done |
| `Size<Pixels>::scale` | method | 498 |  | `(Size).Scale` | done |
| `Size::max` | method | 553 |  | `(Size).Max` | done |
| `Size::min` | method | 583 |  | `(Size).Min` | done |
| `Size<Length>::full` | assoc-fn | 678 |  | `SizeLengthFull` | done |
| `Size<Length>::auto` | assoc-fn | 696 |  | `SizeLengthAuto` | done |
| `Bounds` | struct | 724 |  | `Bounds` | done |
| `bounds` | fn | 732 |  | `NewBounds` | done |
| `Bounds<Pixels>::centered` | assoc-fn | 741 |  | `BoundsCentered` | done |
| `Bounds<Pixels>::maximized` | assoc-fn | 758 |  | `BoundsMaximized` | done |
| `Bounds::new` | assoc-fn | 786 |  | `NewBounds` | done |
| `Bounds::from_corners` | assoc-fn | 822 |  | `BoundsFromCorners` | done |
| `Bounds::from_anchor_and_size` | assoc-fn | 841 |  | `BoundsFromAnchorAndSize` | done |
| `Bounds::centered_at` | assoc-fn | 883 |  | `BoundsCenteredAt` | done |
| `Bounds::top_center` | method | 897 |  | `(Bounds).TopCenter` | done |
| `Bounds::bottom_center` | method | 905 |  | `(Bounds).BottomCenter` | done |
| `Bounds::left_center` | method | 913 |  | `(Bounds).LeftCenter` | done |
| `Bounds::right_center` | method | 921 |  | `(Bounds).RightCenter` | done |
| `Bounds::intersects` | method | 966 |  | `(Bounds).Intersects` | done |
| `Bounds::center` | method | 1002 |  | `(Bounds).Center` | done |
| `Bounds::half_perimeter` | method | 1032 |  | `(Bounds).HalfPerimeter` | done |
| `Bounds::dilate` | method | 1068 |  | `(Bounds).Dilate` | done |
| `Bounds::extend` | method | 1078 |  | `(Bounds).Extend` | done |
| `Bounds::inset` | method | 1103 |  | `(Bounds).Inset` | done |
| `Bounds::intersect` | method | 1144 |  | `(Bounds).Intersect` | done |
| `Bounds::union` | method | 1186 |  | `(Bounds).Union` | done |
| `Bounds::join` | method | 1213 |  | `(Bounds).Join` | done |
| `Bounds::space_within` | method | 1229 |  | `(Bounds).SpaceWithin` | done |
| `Bounds::top` | method | 1328 |  | `(Bounds).Top` | done |
| `Bounds::bottom` | method | 1337 |  | `(Bounds).Bottom` | done |
| `Bounds::left` | method | 1346 |  | `(Bounds).Left` | done |
| `Bounds::right` | method | 1355 |  | `(Bounds).Right` | done |
| `Bounds::top_right` | method | 1376 |  | `(Bounds).TopRight` | done |
| `Bounds::bottom_right` | method | 1400 |  | `(Bounds).BottomRight` | done |
| `Bounds::bottom_left` | method | 1424 |  | `(Bounds).BottomLeft` | done |
| `Bounds::corner` | method | 1453 |  | `(Bounds).Corner` | done |
| `Bounds::contains` | method | 1500 |  | `(Bounds).Contains` | done |
| `Bounds::is_contained_within` | method | 1541 |  | `(Bounds).IsContainedWithin` | done |
| `Bounds::map` | method | 1574 |  | `(Bounds).Map` | done |
| `Bounds::map_origin` | method | 1601 |  | `(Bounds).MapOrigin` | done |
| `Bounds::map_size` | method | 1625 |  | `(Bounds).MapSize` | done |
| `Bounds::localize` | method | 1638 |  | `(Bounds).Localize` | done |
| `Bounds::is_empty` | method | 1656 |  | `(Bounds).IsEmpty` | done |
| `Size<DevicePixels>::to_pixels` | method | 1675 |  | `(Size).ToPixels` | done |
| `Size<Pixels>::to_device_pixels` | method | 1685 |  | `(Size).ToDevicePixels` | done |
| `Bounds<Pixels>::scale` | method | 1730 |  | `(Bounds).Scale` | done |
| `Bounds<Pixels>::to_device_pixels` | method | 1738 |  | `(Bounds).ToDevicePixels` | done |
| `Bounds<DevicePixels>::to_pixels` | method | 1751 |  | `(Bounds).ToPixels` | done |
| `Edges` | struct | 1785 |  | `Edges` | done |
| `Edges::all` | assoc-fn | 1868 |  | `EdgesAll` | done |
| `Edges::map` | method | 1899 |  | `(Edges).Map` | done |
| `Edges::any` | method | 1938 |  | `(Edges).Any` | done |
| `Edges<Length>::auto` | assoc-fn | 1965 |  | `EdgesLengthAuto` | done |
| `Edges<Length>::zero` | assoc-fn | 1993 |  | `EdgesLengthZero` | done |
| `Edges<DefiniteLength>::zero` | assoc-fn | 2023 |  | `EdgesDefiniteLengthZero` | done |
| `Edges<DefiniteLength>::to_pixels` | method | 2068 |  | `EdgesDefiniteLengthToPixels` | done |
| `Edges<AbsoluteLength>::zero` | assoc-fn | 2098 |  | `EdgesAbsoluteLengthZero` | done |
| `Edges<AbsoluteLength>::to_pixels` | method | 2138 |  | `EdgesAbsoluteLengthToPixels` | done |
| `Edges<Pixels>::scale` | method | 2177 |  | `(Edges).Scale` | done |
| `Edges<Pixels>::max` | method | 2191 |  | `(Edges).Max` | done |
| `Anchor` | enum | 2247 |  | `Anchor` | done |
| `Anchor::opposite` | method | 2276 |  | `(Anchor).Opposite` | done |
| `Anchor::other_side_along` | method | 2299 |  | `(Anchor).OtherSideAlong` | done |
| `Anchor::is_center` | method | 2326 |  | `(Anchor).IsCenter` | done |
| `Anchor::is_bottom` | method | 2335 |  | `(Anchor).IsBottom` | done |
| `Corners` | struct | 2349 |  | `Corners` | done |
| `Corners::all` | assoc-fn | 2388 |  | `CornersAll` | done |
| `Corners::corner` | method | 2441 |  | `(Corners).Corner` | done |
| `Corners<AbsoluteLength>::to_pixels` | method | 2484 |  | `CornersAbsoluteLengthToPixels` | done |
| `Corners<Pixels>::scale` | method | 2524 |  | `(Corners).Scale` | done |
| `Corners<Pixels>::max` | method | 2539 |  | `(Corners).Max` | done |
| `Corners::clamp_radii_for_quad_size` | method | 2558 |  | `(Corners).ClampRadiiForQuadSize` | done |
| `Corners::map` | method | 2603 |  | `(Corners).Map` | done |
| `Radians` | struct | 2687 |  | `Radians` | done |
| `radians` | fn | 2690 |  | `Radians` | done |
| `Percentage` | struct | 2712 |  | `Percentage` | done |
| `percentage` | fn | 2715 |  | `NewPercentage` | dev:D-NAME |
| `Pixels` | struct | 2768 |  | `Pixels` | done |
| `Pixels::ZERO` | assoc-const | 2874 |  | `PixelsZero` | done |
| `Pixels::MAX` | assoc-const | 2876 |  | `PixelsMax` | done |
| `Pixels::MIN` | assoc-const | 2878 |  | `PixelsMin` | done |
| `Pixels::as_f32` | method | 2881 |  | `(Pixels).AsF32` | done |
| `Pixels::to_i32` | method | 2887 |  | `(Pixels).ToI32` | done |
| `Pixels::floor` | method | 2896 |  | `(Pixels).Floor` | done |
| `Pixels::round` | method | 2905 |  | `(Pixels).Round` | done |
| `Pixels::ceil` | method | 2914 |  | `(Pixels).Ceil` | done |
| `Pixels::scale` | method | 2927 |  | `(Pixels).Scale` | done |
| `Pixels::pow` | method | 2940 |  | `(Pixels).Pow` | done |
| `Pixels::abs` | method | 2949 |  | `(Pixels).Abs` | done |
| `Pixels::signum` | method | 2960 |  | `(Pixels).Signum` | done |
| `Pixels::to_f64` | method | 2969 |  | `(Pixels).ToF64` | done |
| `DevicePixels` | struct | 3081 |  | `DevicePixels` | done |
| `DevicePixels::to_bytes` | method | 3106 |  | `(DevicePixels).ToBytes` | done |
| `ScaledPixels` | struct | 3174 |  | `ScaledPixels` | done |
| `ScaledPixels::as_f32` | method | 3178 |  | `(ScaledPixels).AsF32` | done |
| `ScaledPixels::floor` | method | 3187 |  | `(ScaledPixels).Floor` | done |
| `ScaledPixels::round` | method | 3196 |  | `(ScaledPixels).Round` | done |
| `ScaledPixels::ceil` | method | 3205 |  | `(ScaledPixels).Ceil` | done |
| `Rems` | struct | 3349 |  | `Rems` | done |
| `Rems::ZERO` | assoc-const | 3353 |  | `RemsZero` | done |
| `Rems::to_pixels` | method | 3355 |  | `(Rems).ToPixels` | done |
| `Rems::from_pixels` | assoc-fn | 3359 |  | `RemsFromPixels` | done |
| `AbsoluteLength` | enum | 3409 |  | `AbsoluteLength` | done |
| `AbsoluteLength::is_zero` | method | 3418 |  | `(AbsoluteLength).IsZero` | done |
| `AbsoluteLength::to_pixels` | method | 3460 |  | `(AbsoluteLength).ToPixels` | done |
| `AbsoluteLength::to_rems` | method | 3476 |  | `(AbsoluteLength).ToRems` | done |
| `DefiniteLength` | enum | 3571 |  | `DefiniteLength` | done |
| `DefiniteLength::to_pixels` | method | 3607 |  | `(DefiniteLength).ToPixels` | done |
| `Length` | enum | 3722 |  | `Length` | done |
| `relative` | fn | 3816 |  | `Relative` | done |
| `phi` | fn | 3821 |  | `Phi` | done |
| `rems` | fn | 3834 |  | `Rems` | done |
| `px` | fn | 3847 |  | `Px` | done |
| `auto` | fn | 3862 |  | `Auto` | done |
| `GridLocation` | struct | 3904 |  | `GridLocation` | done |
| `GridPlacement` | enum | 3913 |  | `GridPlacement` | done |
| `Half` | trait | 3938 |  | `Half[S]` | done |
| `Half::half` | trait-method | 3944 |  | `Half.Half` | done |
| `IsZero` | trait | 3989 |  | `IsZero` | done |
| `IsZero::is_zero` | trait-method | 3995 |  | `IsZero.IsZero` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_pixels_to_i32` | test | 4087 |  | `TestGeometry_PixelsToI32` | done |
| `test_bounds_join` | test | 4102 |  | `TestGeometry_BoundsJoin` | done |
| `test_bounds_intersects` | test | 4133 |  | `TestGeometry_BoundsIntersects` | done |

## gpui/src/gpui.rs

Go target: `gpui.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Sealed` | trait | 169 |  | `-` | n/a:D-CTX |
| `AppContext` | trait | 174 |  | `AppContext` | done |
| `AppContext::new` | trait-method | 180 |  | `AppContextNew` | dev:D-CTX |
| `AppContext::reserve_entity` | trait-method | 184 |  | `AppContextReserveEntity` | dev:D-CTX |
| `AppContext::insert_entity` | trait-method | 189 |  | `AppContextInsertEntity` | dev:D-CTX |
| `AppContext::update_entity` | trait-method | 196 |  | `AppContextUpdateEntity` | dev:D-CTX |
| `AppContext::as_mut` | trait-method | 205 |  | `AppContextAsMut` | dev:D-CTX |
| `AppContext::read_entity` | trait-method | 210 |  | `AppContextReadEntity` | dev:D-CTX |
| `AppContext::update_window` | trait-method | 216 |  | `AppContextUpdateWindow` | dev:D-CTX |
| `AppContext::with_window` | trait-method | 224 |  | `AppContextWithWindow` | dev:D-CTX |
| `AppContext::read_window` | trait-method | 231 |  | `AppContextReadWindow` | dev:D-CTX |
| `AppContext::background_spawn` | trait-method | 240 |  | `AppContextBackgroundSpawn` | dev:D-CTX |
| `AppContext::read_global` | trait-method | 245 |  | `AppContextReadGlobal` | dev:D-CTX |
| `Reservation` | struct | 252 |  | `Reservation` | done |
| `Reservation::entity_id` | method | 256 |  | `(Reservation).EntityID` | done |
| `VisualContext` | trait | 263 |  | `VisualContext` | done |
| `VisualContext::window_handle` | trait-method | 268 |  | `VisualContext.WindowHandle` | done |
| `VisualContext::update_window_entity` | trait-method | 271 |  | `VisualContextUpdateWindowEntity` | dev:D-CTX |
| `VisualContext::new_window_entity` | trait-method | 278 |  | `VisualContextNewWindowEntity` | dev:D-CTX |
| `VisualContext::replace_root_view` | trait-method | 284 |  | `VisualContextReplaceRootView` | dev:D-CTX |
| `VisualContext::focus` | trait-method | 292 |  | `VisualContextFocus` | dev:D-CTX |
| `EventEmitter` | trait | 299 |  | `-` | n/a:D-EMIT |
| `BorrowAppContext` | trait | 303 |  | `BorrowAppContext` | done |
| `BorrowAppContext::set_global` | trait-method | 305 |  | `SetGlobal` | dev:D-CTX |
| `BorrowAppContext::update_global` | trait-method | 307 |  | `UpdateGlobal`, `UpdateGlobalR` | dev:D-CTX |
| `BorrowAppContext::update_default_global` | trait-method | 311 |  | `UpdateDefaultGlobal`, `UpdateDefaultGlobalR` | dev:D-CTX |
| `GpuSpecs` | struct | 346 |  | `GPUSpecs` | done |

## gpui/src/shared_uri.rs

Go target: `shared_uri.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SharedUri` | struct | 7 |  | `SharedURI` | done |

## gpui/src/util.rs

Go target: `util.go` (package `gpui`), tests in `util_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `FluentBuilder` | trait | 11 |  | `Fluent[S] mixin` | done |
| `FluentBuilder::map` | trait-method | 13 |  | `(Fluent[S]).Map` | done |
| `FluentBuilder::when` | trait-method | 21 |  | `(Fluent[S]).When` | done |
| `FluentBuilder::when_else` | trait-method | 29 |  | `(Fluent[S]).WhenElse` | done |
| `FluentBuilder::when_some` | trait-method | 42 |  | `(Fluent[S]).WhenSome` | done |
| `FluentBuilder::when_none` | trait-method | 55 |  | `(Fluent[S]).WhenNone` | done |
| `FutureExt` | trait | 64 |  | `FutureWithTimeout` (func) | dev:D-FUTUREEXT |
| `FutureExt::with_timeout` | trait-method | 67 |  | `FutureWithTimeout` (func) | dev:D-FUTUREEXT |
| `WithTimeout` | struct | 85 |  | `WithTimeout` | done |
| `Timeout` | struct | 95 |  | `Timeout` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_round_half_toward_zero` | test | 170 |  | `TestUtil_RoundHalfTowardZero` | done |
| `test_device_pixel_helpers` | test | 192 |  | `TestUtil_DevicePixelHelpers` | done |
| `test_round_half_toward_zero_f64` | test | 220 |  | `TestUtil_RoundHalfTowardZeroF64` | done |
| `test_with_timeout` | gpui::test | 229 |  | `TestUtil_WithTimeout` | done |


## Notes (M0-geometry)

- D-GEOM: numeric family `Point[T Number]`, `Size`, `Bounds`, `Edges` with operator methods (`Add`, `AddAssign`, `Sub`, `SubAssign`, `Neg`, `Mul`, `MulAssign`, `Div`, `Bounds.Add(Point)`, `Bounds.Sub(Point)`) matching upstream's derive_more / ops impls; scalar operands are `rhs T`. Non-numeric family `PointOf[T any]`, `SizeOf`, `EdgesOf` with `Map`, `Along`, `ApplyAlong`, `EdgesOfAll`, `Any`. `Along` is `Along[U, S]`.
- Cross-family maps: `(Point|Size|Edges).MapOf[U any]` return the Of type; `(PointOf|SizeOf|EdgesOf).MapNumber[U Number]` return the numeric type; `Map` stays within a family. `Bounds` has no Of variant.
- `Bounds::localize` returns `(Point[T], bool)` (D-OPT); `Bounds::centered` / `maximized` take `Option[DisplayID]`.

## Notes (M0-geometry-units)

- Corners[T Number] / CornersOf[T any] follow D-GEOM; `Corners<AbsoluteLength>::to_pixels` is the free function `CornersAbsoluteLengthToPixels`. Refinements `CornersRefinement[T]`, `CornersOfRefinement[T]` come from refinegen.
- AbsoluteLength, DefiniteLength, Length and GridPlacement are comparable tagged structs whose zero value is upstream's Default (px(0), Auto for GridPlacement). Variant constructors `AbsoluteLengthPixels`, `AbsoluteLengthRems`, `DefiniteLengthAbsolute`, `DefiniteLengthFraction`, `LengthDefinite`, `LengthAuto`, `GridPlacementLine`, `GridPlacementSpan`, `GridPlacementAuto`; accessors `AsPixels`, `AsRems`, `AsAbsolute`, `AsFraction`, `AsDefinite`, `IsAuto`, `AsLine`, `AsSpan`.
- `From` impls become methods: `ToLength`, `ToDefiniteLength`, `ToAbsoluteLength` (IntoLength / IntoDefiniteLength / IntoAbsoluteLength), `(Rgba).ToHsla`, `(Hsla).ToRgb`, `(Rgba|Hsla).ToBackground`, `(Rgba).ToU32`, `(ScaledPixels).ToDevicePixels` (ceil), `(DevicePixels).ToScaledPixels`, `(Pixels).ToU32/ToUsize`, `(ScaledPixels).ToU32` (Rust `as` saturation), `(Percentage).ToRadians`, `SizeDefiniteLengthFromPixels`, `SizeAbsoluteLengthFromPixels`. `TryFrom<&str>` impls are `ParsePixels`, `ParseRems`, `ParseAbsoluteLength`, `ParseDefiniteLength`, `ParseLength`, `ParseRgba`.
- `percentage()` is `NewPercentage` (keeps the debug assertion); `Percentage(x)` is the unchecked conversion.
- Hsla's PartialEq / Ord (total_cmp) are `(Hsla).Eq` / `(Hsla).Cmp`; Go `==` is IEEE. Background and LinearColorStop have `Eq` using it. refineable.Eq picks up these methods.
- BackgroundTag and ColorSpace are `uint32` (repr(C) enums are 4 bytes in the GPU struct). Background is 72 bytes, fields unexported as upstream's pub(crate).
- `Range<GridPlacement>` is `GridPlacementRange{Start, End}`.
- Edges / EdgesOf got their serde json tags (`top`, ...) which M0-geometry left out.

## Notes (M0-contracts)

- gpui.rs re-exports: `Task`, `FallibleTask`, `Priority`, `DedicatedExecutor`, `SchedulerLocalExecutor` (aliases of package scheduler, executor.rs's `pub use`), `AccessibleAction`, `Orientation`, `Role`, `Toggled` (aliases of package a11y). `anyhow::Result`, `ArcCow`, `pollster::block_on` and the macro re-exports have no Go alias (D-OPT, D-STRING, D-MACRO).
- AppContext primitives (unexported, in upstream's trait order plus `newEntity`, because `App::new` pushes `EntityCreated` and `insert_entity` does not): `newEntity`, `reserveEntity`, `insertEntity`, `updateEntity`, `asMut`, `readEntity`, `updateWindow`, `withWindow`, `readWindow`, `backgroundExecutor`, `readGlobal`. Return types follow upstream's trait at this commit: only `update_window` / `read_window` return `Result`, `with_window` returns `Option` (`(R, bool)`); `with_window` takes an `EntityID` as upstream (DESIGN 6.2 sketched a window handle).
- `background_spawn` takes `func(*scheduler.Scope) (R, error)` (the scheduler's task body) instead of DESIGN 6.2's `func() R`, so the body can await.
- VisualContext's associated `Result<T>` is `error` / `(R, error)` for every Go implementation (STAGES decision 11); its primitive is `updateWindowContext`.
- BorrowAppContext's `&mut Self` callback parameter is the context's own type through a type parameter `C BorrowAppContext`; D-VOID gives `UpdateGlobal` / `UpdateGlobalR` and `UpdateDefaultGlobal` / `UpdateDefaultGlobalR`.

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-CTX | Generic methods of the context traits (`AppContext`, `VisualContext`, `BorrowAppContext`) become free generic functions over a sealed interface of unexported, type-erased primitives (`AppContextNew`, `VisualContextFocus`, `SetGlobal`, ...); each concrete context also has the upstream-named generic methods as delegates. The `Sealed` marker has no Go symbol: the unexported primitives seal the interface | `gpui/src/gpui.rs:169-339` | A Go interface cannot declare generic methods (DESIGN 5.4, 6.2) |
| D-IDMIXIN | `InteractiveElement::id` lives on its own mixin `InteractiveID[S]` (bound with `BindInteractiveID`), which `Div` and the other interactive elements embed and `Stateful[E]` does not; `Stateful<E>::id` (re-wrapping) is absent | `gpui/src/elements/div.rs:803` | `Interactive[S].ID` returning `*Stateful[S]` while `Stateful[E]` embeds `Interactive[*Stateful[E]]` is an instantiation cycle the Go type checker rejects; re-wrapping a Stateful only re-sets the id on the shared Interactivity |
| D-FUTUREEXT | `FutureExt::with_timeout` (a blanket extension trait on every Future) is the free function `FutureWithTimeout(future, timeout, executor)` returning `*WithTimeout[T]`, a `scheduler.Future[scheduler.Result[T]]` whose error is `Timeout` | `gpui/src/util.rs:64-112` | Go cannot add methods to an interface's implementors |

## Notes (M0-input-types)

- util.rs: `Fluent[S]` has `Map`, `When`, `WhenElse`, `WhenSome[T]`, `WhenNone[T]`; `WhenNone` takes `Option[T]` by value. The crate-private rounding helpers (`roundHalfTowardZero`, `...F64`, `roundToDevicePixel`, `roundStrokeToDevicePixel`, `floorToDevicePixel`, `ceilToDevicePixel`) and `atomicIncrIfNotZero` (`*atomic.Uint64`) are unexported in util.go. `TestUtil_WithTimeout` was ported by M0-test-contexts; `now_or_never` is a single `Poll` with a no-op waker.
- colors.rs: `GlobalColors` embeds `*Colors` (Deref); `DefaultColors` is an interface implemented by `(*App).DefaultColors`; `impl Default for Colors` has no Go function (the name `DefaultColors` is the trait's), `ColorsLight` is the default; `From<WindowAppearance>` is `(WindowAppearance).ToDefaultAppearance`.
