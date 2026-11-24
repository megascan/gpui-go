# Parity: core-style

Scope: gpui: style, styled, taffy bridge, scene, paths, spring. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/src/path_builder.rs

Go target: `path_builder.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `PathStyle` | enum | 15 |  | `PathStyle` | done |
| `StrokeOptions` | struct | 24 |  | `StrokeOptions` | done |
| `FillOptions` | struct | 49 |  | `FillOptions` | done |
| `PathBuilder` | struct | 63 |  | `PathBuilder` | done |
| `PathBuilder::stroke` | assoc-fn | 126 |  | `PathBuilderStroke` | done |
| `PathBuilder::fill` | assoc-fn | 137 |  | `PathBuilderFill` | done |
| `PathBuilder::with_style` | method | 142 |  | `(PathBuilder).WithStyle` | done |
| `PathBuilder::dash_array` | method | 149 |  | `(PathBuilder).DashArray` | done |
| `PathBuilder::move_to` | method | 166 |  | `(PathBuilder).MoveTo` | done |
| `PathBuilder::line_to` | method | 172 |  | `(PathBuilder).LineTo` | done |
| `PathBuilder::curve_to` | method | 178 |  | `(PathBuilder).CurveTo` | done |
| `PathBuilder::cubic_bezier_to` | method | 185 |  | `(PathBuilder).CubicBezierTo` | done |
| `PathBuilder::arc_to` | method | 196 |  | `(PathBuilder).ArcTo` | done |
| `PathBuilder::relative_arc_to` | method | 213 |  | `(PathBuilder).RelativeArcTo` | done |
| `PathBuilder::add_polygon` | method | 230 |  | `(PathBuilder).AddPolygon` | done |
| `PathBuilder::close` | method | 240 |  | `(PathBuilder).Close` | done |
| `PathBuilder::transform` | method | 246 |  | `(PathBuilder).Transform` | done |
| `PathBuilder::translate` | method | 252 |  | `(PathBuilder).Translate` | done |
| `PathBuilder::scale` | method | 262 |  | `(PathBuilder).Scale` | done |
| `PathBuilder::rotate` | method | 274 |  | `(PathBuilder).Rotate` | done |
| `PathBuilder::build` | method | 284 |  | `(PathBuilder).Build` | done |
| `Path` | struct | 387 |  | `Path` | done |
| `pub use lyon_path::math::Transform` | type | 8 |  | `Transform` | done |
| `pub use lyon_path::FillRule` | type | 9 |  | `FillRule` | done |
| `FillRule::EvenOdd` (re-export) | variant | 9 |  | `FillRuleEvenOdd` | done |
| `FillRule::NonZero` (re-export) | variant | 9 |  | `FillRuleNonZero` | done |
| `pub use lyon_path::LineCap` | type | 9 |  | `LineCap` | done |
| `LineCap::Butt` (re-export) | variant | 9 |  | `LineCapButt` | done |
| `LineCap::Square` (re-export) | variant | 9 |  | `LineCapSquare` | done |
| `LineCap::Round` (re-export) | variant | 9 |  | `LineCapRound` | done |
| `pub use lyon_path::LineJoin` | type | 9 |  | `LineJoin` | done |
| `LineJoin::Miter` (re-export) | variant | 9 |  | `LineJoinMiter` | done |
| `LineJoin::MiterClip` (re-export) | variant | 9 |  | `LineJoinMiterClip` | done |
| `LineJoin::Round` (re-export) | variant | 9 |  | `LineJoinRound` | done |
| `LineJoin::Bevel` (re-export) | variant | 9 |  | `LineJoinBevel` | done |
| `PathStyle::Stroke` | variant | 17 |  | `PathStyleStroke` | done |
| `PathStyle::Fill` | variant | 19 |  | `PathStyleFill` | done |
| `impl Default for StrokeOptions` | impl | 37 |  | `DefaultStrokeOptions` | done |
| `impl Default for FillOptions` | impl | 55 |  | `DefaultFillOptions` | done |
| `impl Default for PathBuilder` | impl | 112 |  | `DefaultPathBuilder` | done |
| `impl From<lyon_path::Builder> for PathBuilder` | impl | 70 |  | `newPathBuilderFromBuilder` | done |
| `impl From<WithSvg<BuilderImpl>> for PathBuilder` | impl | 79 |  | `newPathBuilderFromSvg` | done |
| `impl From<lyon_path::math::Point> for Point<Pixels>` | impl | 88 |  | `pointFromLyon` | done |
| `impl From<Point<Pixels>> for lyon_path::math::Point` | impl | 94 |  | `lyonPoint` | done |
| `impl From<Point<Pixels>> for Vector` | impl | 100 |  | `lyonVector` | done |
| `impl From<Point<Pixels>> for Point2D<f32, Pixels>` | impl | 106 |  | `lyonPoint` | done |

## gpui/src/path_winding.rs

Go target: `path_winding.go` (package `gpui`), tests in `path_winding_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `rectangle_is_nine_tiles` | test | 715 |  | `TestPathWinding_RectangleIsNineTiles` | done |
| `huge_path_paints_only_visible_tiles` | test | 723 |  | `TestPathWinding_HugePathPaintsOnlyVisibleTiles` | done |
| `overflowing_paths_are_skipped` | test | 746 |  | `TestPathWinding_OverflowingPathsAreSkipped` | done |
| `non_finite_point_fails_path` | test | 759 |  | `TestPathWinding_NonFinitePointFailsPath` | done |
| `endpoints_balance` | test | 776 |  | `TestPathWinding_EndpointsBalance` | done |

## gpui/src/scene.rs

Go target: `scene.go` (package `gpui`), tests in `scene_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DrawOrder` | type | 18 |  | `DrawOrder` | done |
| `PaddedBool32` | struct | 26 |  | `PaddedBool32` | done |
| `PaddedBool32::as_bool` | method | 36 |  | `(PaddedBool32).AsBool` | done |
| `Scene` | struct | 53 |  | `Scene` | done |
| `Scene::clear` | method | 72 |  | `(Scene).Clear` | done |
| `Scene::len` | method | 131 |  | `(Scene).Len` | done |
| `Scene::push_layer` | method | 135 |  | `(Scene).PushLayer` | done |
| `Scene::pop_layer` | method | 147 |  | `(Scene).PopLayer` | done |
| `Scene::insert_primitive` | method | 220 |  | `(Scene).InsertPrimitive` | done |
| `Scene::replay` | method | 286 |  | `(Scene).Replay` | done |
| `Scene::finish` | method | 306 |  | `(Scene).Finish` | done |
| `Scene::batches` | method | 372 |  | `(Scene).Batches` | done |
| `quad_depth_scale` | fn | 437 |  | `QuadDepthScale` | done |
| `quad_depth` | fn | 443 |  | `QuadDepth` | done |
| `Primitive` | enum | 493 |  | `Primitive` | dev:D-PRIMITIVE |
| `Primitive::bounds` | method | 521 |  | `(Primitive).Bounds` | done |
| `Primitive::content_mask` | method | 549 |  | `(Primitive).ContentMask` | done |
| `PrimitiveBatch` | enum | 806 |  | `PrimitiveBatch` | done |
| `PrimitiveBatch::label` | method | 830 |  | `(PrimitiveBatch).Label` | done |
| `Quad` | struct | 866 |  | `Quad` | done |
| `Underline` | struct | 886 |  | `Underline` | done |
| `Shadow` | struct | 905 |  | `Shadow` | done |
| `BorderStyle` | enum | 927 |  | `BorderStyle` | dev:D-GPUENUM |
| `TransformationMatrix` | struct | 938 |  | `TransformationMatrix` | done |
| `TransformationMatrix::unit` | assoc-fn | 950 |  | `TransformationMatrixUnit` | done |
| `TransformationMatrix::translate` | method | 958 |  | `(TransformationMatrix).Translate` | done |
| `TransformationMatrix::rotate` | method | 966 |  | `(TransformationMatrix).Rotate` | done |
| `TransformationMatrix::scale` | method | 977 |  | `(TransformationMatrix).Scale` | done |
| `TransformationMatrix::compose` | method | 988 |  | `(TransformationMatrix).Compose` | done |
| `TransformationMatrix::apply` | method | 1045 |  | `(TransformationMatrix).Apply` | done |
| `MonochromeSprite` | struct | 1066 |  | `MonochromeSprite` | done |
| `SpriteAnimationId` | struct | 1082 |  | `SpriteAnimationID` | done |
| `SpriteAnimationId::NONE` | assoc-const | 1086 |  | `SpriteAnimationIDNone` | done |
| `SubpixelSprite` | struct | 1131 |  | `SubpixelSprite` | done |
| `PolychromeSprite` | struct | 1150 |  | `PolychromeSprite` | done |
| `PaintSurface` | struct | 1169 |  | `PaintSurface` | dev:D-SURFACEBUF |
| `PaintNativeView` | struct | 1189 |  | `PaintNativeView` | done |
| `PathCurve` | struct | 1214 |  | `PathCurve` | done |
| `PathPaint` | struct | 1226 |  | `PathPaint` | done |
| `PathTile` | struct | 1238 |  | `PathTile` | done |
| `PaintedPath` | struct | 1252 |  | `PaintedPath` | done |
| `PaintedPath::new` | assoc-fn | 1262 |  | `NewPaintedPath` | done |
| `PaintedPath::tile_count` | method | 1286 |  | `(PaintedPath).TileCount` | done |
| `PaintedPath::append_to` | method | 1290 |  | `(PaintedPath).AppendTo` | done |
| `impl From<bool> for PaddedBool32` | impl | 28 |  | `PaddedBool32FromBool` | done |
| `impl From<X> for Primitive` (every primitive) | impl | 877 |  | `IntoPrimitive` | dev:D-PRIMITIVE |
| `impl Default for TransformationMatrix` | impl | 1058 |  | `DefaultTransformationMatrix` | done |
| `impl Default for SpriteAnimationId` | impl | 1089 |  | `DefaultSpriteAnimationID` | done |
| `BorderStyle` serde (Serialize) | impl | 925 |  | `(BorderStyle).MarshalJSONTo` | done |
| `BorderStyle` serde (Deserialize) | impl | 925 |  | `(BorderStyle).UnmarshalJSONFrom` | done |
| `BorderStyle` schema (JsonSchema) | impl | 925 |  | `(BorderStyle).JSONSchema` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `a_native_view_in_a_layer_is_covered_by_what_the_layer_paints_after_it` | test | 1352 |  | `TestScene_ANativeViewInALayerIsCoveredByWhatTheLayerPaintsAfterIt` | done |
| `a_native_view_painted_twice_is_kept_where_it_was_painted_first` | test | 1375 |  | `TestScene_ANativeViewPaintedTwiceIsKeptWhereItWasPaintedFirst` | done |
| `layers_are_drawn_in_paint_order` | test | 1405 |  | `TestScene_LayersAreDrawnInPaintOrder` | done |
| `nested_layers_are_drawn_above_their_parents` | test | 1431 |  | `TestScene_NestedLayersAreDrawnAboveTheirParents` | done |

## gpui/src/spring.rs

Go target: `spring.go` (package `gpui`), tests in `spring_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SpringConfig` | struct | 13 |  | `SpringConfig` | done |
| `SpringConfig::new` | assoc-fn | 24 |  | `NewSpringConfig` | done |
| `SpringConfig::canonical` | method | 33 |  | `(SpringConfig).Canonical` | done |
| `SpringConfig::step` | method | 43 |  | `(SpringConfig).Step` | done |
| `SpringConfig::step_ramp` | method | 57 |  | `(SpringConfig).StepRamp` | done |
| `SpringConfig::propagator` | method | 87 |  | `(SpringConfig).Propagator` | done |
| `SpringConfig::is_settled` | method | 146 |  | `(SpringConfig).IsSettled` | done |
| `SpringConfig::settle_time` | method | 158 |  | `(SpringConfig).SettleTime` | done |
| `SpringState` | struct | 252 |  | `SpringState` | done |
| `SpringTarget` | trait | 264 |  | `SpringTarget` | done |
| `SpringTarget::target` | trait-method | 269 |  | `SpringTarget.Target` | done |
| `SpringTarget::resolve` | trait-method | 272 |  | `SpringTarget.Resolve` | done |
| `AnimationPhase` | struct | 328 |  | `AnimationPhase` | done |
| `AnimationPhase::clamp` | method | 335 |  | `(AnimationPhase).Clamp` | done |
| `AnimationPhase::interpolate` | method | 341 |  | `(AnimationPhase).Interpolate` | done |
| `AnimationPhase::interpolate_clamped` | method | 346 |  | `(AnimationPhase).InterpolateClamped` | done |
| `AnimationPhase::interpolate_between` | method | 351 |  | `(AnimationPhase).InterpolateBetween` | done |
| `AnimationPhase::interpolate_between_clamped` | method | 367 |  | `(AnimationPhase).InterpolateBetweenClamped` | done |
| `Interpolate` | trait | 408 |  | `Interpolate` | done |
| `Interpolate::interpolate` | trait-method | 410 |  | `Interpolate.Interpolate` | done |
| `SpringPlayback` | enum | 456 |  | `SpringPlayback` | done |
| `SpringAnimation` | struct | 472 |  | `SpringAnimation` | done |
| `SpringAnimation::new` | assoc-fn | 482 |  | `NewSpringAnimation` | done |
| `SpringAnimation::to` | method | 493 |  | `(SpringAnimation).To` | done |
| `SpringAnimation::with_epsilon` | method | 513 |  | `(SpringAnimation).WithEpsilon` | done |
| `SpringAnimation::playback` | method | 519 |  | `(SpringAnimation).Playback` | done |
| `SpringAnimation::from` | method | 527 |  | `(SpringAnimation).From` | done |
| `sampled_easing` | fn | 538 |  | `SampledEasing` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `spring_targets_resolve_typed_outputs` | test | 622 |  | `TestSpring_SpringTargetsResolveTypedOutputs` | done |
| `animation_phases_interpolate_over_arbitrary_ranges` | test | 633 |  | `TestSpring_AnimationPhasesInterpolateOverArbitraryRanges` | done |
| `hsla_interpolation_takes_the_shortest_hue_path` | test | 647 |  | `TestSpring_HslaInterpolationTakesTheShortestHuePath` | done |
| `propagators_compose_and_have_expected_determinant` | test | 669 |  | `TestSpring_PropagatorsComposeAndHaveExpectedDeterminant` | done |
| `step_preserves_semigroup_for_every_damping_regime` | test | 698 |  | `TestSpring_StepPreservesSemigroupForEveryDampingRegime` | done |
| `ramp_tracks_steady_state_lag` | test | 714 |  | `TestSpring_RampTracksSteadyStateLag` | done |
| `settling_requires_low_velocity` | test | 735 |  | `TestSpring_SettlingRequiresLowVelocity` | done |
| `settle_time_is_conservative_for_every_damping_regime` | test | 756 |  | `TestSpring_SettleTimeIsConservativeForEveryDampingRegime` | done |
| `settle_time_accounts_for_motion_outside_an_instantaneous_tolerance` | test | 778 |  | `TestSpring_SettleTimeAccountsForMotionOutsideAnInstantaneousTolerance` | done |
| `undamped_spring_never_settles` | test | 792 |  | `TestSpring_UndampedSpringNeverSettles` | done |
| `sampled_easing_has_exact_endpoints_and_can_overshoot` | test | 808 |  | `TestSpring_SampledEasingHasExactEndpointsAndCanOvershoot` | done |

## gpui/src/style.rs

Go target: `style.go` (package `gpui`), tests in `style_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DebugBelow` | struct | 23 | debug | `DebugBelow` | todo |
| `ObjectFit` | enum | 29 |  | `ObjectFit` | todo |
| `ObjectFit::get_bounds` | method | 44 |  | `(ObjectFit).GetBounds` | todo |
| `GridTemplateMinSize` | enum | 145 |  | `GridTemplateMinSize` | todo |
| `GridTemplate` | struct | 170 |  | `GridTemplate` | todo |
| `Style` | struct | 180 |  | `Style` | todo |
| `StyleRefinement::grid_location_mut` | method | 332 |  | `(StyleRefinement).GridLocationMut` | todo |
| `Visibility` | enum | 339 |  | `Visibility` | todo |
| `BoxShadow` | struct | 349 |  | `BoxShadow` | todo |
| `BoxShadow::new` | assoc-fn | 366 |  | `NewBoxShadow` | todo |
| `BoxShadow::blur_radius` | method | 377 |  | `(BoxShadow).BlurRadius` | todo |
| `BoxShadow::spread_radius` | method | 383 |  | `(BoxShadow).SpreadRadius` | todo |
| `BoxShadow::inset` | method | 389 |  | `(BoxShadow).Inset` | todo |
| `WhiteSpace` | enum | 397 |  | `WhiteSpace` | todo |
| `TextOverflow` | enum | 407 |  | `TextOverflow` | todo |
| `TextAlign` | enum | 423 |  | `TextAlign` | todo |
| `TextStyle` | struct | 438 |  | `TextStyle` | todo |
| `TextStyle::highlight` | method | 510 |  | `(TextStyle).Highlight` | todo |
| `TextStyle::font` | method | 543 |  | `(TextStyle).Font` | todo |
| `TextStyle::line_height_in_pixels` | method | 554 |  | `(TextStyle).LineHeightInPixels` | todo |
| `TextStyle::to_run` | method | 559 |  | `(TextStyle).ToRun` | todo |
| `HighlightStyle` | struct | 580 |  | `HighlightStyle` | todo |
| `Style::has_opaque_background` | method | 621 |  | `(Style).HasOpaqueBackground` | todo |
| `Style::text_style` | method | 628 |  | `(Style).TextStyle` | todo |
| `Style::overflow_mask` | method | 638 |  | `(Style).OverflowMask` | todo |
| `Style::paint` | method | 688 |  | `(Style).Paint` | todo |
| `UnderlineStyle` | struct | 828 |  | `UnderlineStyle` | todo |
| `StrikethroughStyle` | struct | 843 |  | `StrikethroughStyle` | todo |
| `Fill` | enum | 853 |  | `Fill` | todo |
| `Fill::color` | method | 862 |  | `(Fill).Color` | todo |
| `HighlightStyle::color` | assoc-fn | 915 |  | `HighlightStyleColor` | todo |
| `HighlightStyle::highlight` | method | 924 |  | `(HighlightStyle).Highlight` | todo |
| `combine_highlights` | fn | 990 |  | `CombineHighlights` | todo |
| `AlignItems` | enum | 1039 |  | `AlignItems` | todo |
| `JustifyItems` | type | 1066 |  | `JustifyItems` | todo |
| `AlignSelf` | type | 1073 |  | `AlignSelf` | todo |
| `JustifySelf` | type | 1080 |  | `JustifySelf` | todo |
| `AlignContent` | enum | 1089 |  | `AlignContent` | todo |
| `JustifyContent` | type | 1124 |  | `JustifyContent` | todo |
| `Display` | enum | 1131 |  | `Display` | todo |
| `FlexWrap` | enum | 1150 |  | `FlexWrap` | todo |
| `FlexDirection` | enum | 1173 |  | `FlexDirection` | todo |
| `Overflow` | enum | 1208 |  | `Overflow` | todo |
| `Position` | enum | 1236 |  | `Position` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_basic_highlight_style_combination` | perf | 1340 |  | `TestStyle_BasicHighlightStyleCombination` | todo |
| `test_combine_highlights` | perf | 1425 |  | `TestStyle_CombineHighlights` | todo |
| `test_text_style_refinement` | perf | 1514 |  | `TestStyle_TextStyleRefinement` | todo |

## gpui/src/styled.rs

Go target: `styled.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Styled` | trait | 22 |  | `Styled[S] mixin + Styler interface` | todo |
| `Styled::style` | trait-method | 24 |  | `(Styled[S]).Style` | todo |
| `Styled::block` | trait-method | 38 |  | `(Styled[S]).Block` | todo |
| `Styled::flex` | trait-method | 45 |  | `(Styled[S]).Flex` | todo |
| `Styled::grid` | trait-method | 52 |  | `(Styled[S]).Grid` | todo |
| `Styled::hidden` | trait-method | 59 |  | `(Styled[S]).Hidden` | todo |
| `Styled::scrollbar_width` | trait-method | 68 |  | `(Styled[S]).ScrollbarWidth` | todo |
| `Styled::whitespace_normal` | trait-method | 75 |  | `(Styled[S]).WhitespaceNormal` | todo |
| `Styled::whitespace_nowrap` | trait-method | 82 |  | `(Styled[S]).WhitespaceNowrap` | todo |
| `Styled::text_ellipsis` | trait-method | 89 |  | `(Styled[S]).TextEllipsis` | todo |
| `Styled::text_ellipsis_start` | trait-method | 97 |  | `(Styled[S]).TextEllipsisStart` | todo |
| `Styled::text_ellipsis_middle` | trait-method | 105 |  | `(Styled[S]).TextEllipsisMiddle` | todo |
| `Styled::text_overflow` | trait-method | 111 |  | `(Styled[S]).TextOverflow` | todo |
| `Styled::text_align` | trait-method | 117 |  | `(Styled[S]).TextAlign` | todo |
| `Styled::text_left` | trait-method | 123 |  | `(Styled[S]).TextLeft` | todo |
| `Styled::text_center` | trait-method | 128 |  | `(Styled[S]).TextCenter` | todo |
| `Styled::text_right` | trait-method | 133 |  | `(Styled[S]).TextRight` | todo |
| `Styled::truncate` | trait-method | 139 |  | `(Styled[S]).Truncate` | todo |
| `Styled::line_clamp` | trait-method | 145 |  | `(Styled[S]).LineClamp` | todo |
| `Styled::flex_col` | trait-method | 153 |  | `(Styled[S]).FlexCol` | todo |
| `Styled::flex_col_reverse` | trait-method | 160 |  | `(Styled[S]).FlexColReverse` | todo |
| `Styled::flex_row` | trait-method | 167 |  | `(Styled[S]).FlexRow` | todo |
| `Styled::flex_row_reverse` | trait-method | 174 |  | `(Styled[S]).FlexRowReverse` | todo |
| `Styled::flex_1` | trait-method | 181 |  | `(Styled[S]).Flex1` | todo |
| `Styled::flex_auto` | trait-method | 190 |  | `(Styled[S]).FlexAuto` | todo |
| `Styled::flex_initial` | trait-method | 199 |  | `(Styled[S]).FlexInitial` | todo |
| `Styled::flex_none` | trait-method | 208 |  | `(Styled[S]).FlexNone` | todo |
| `Styled::flex_basis` | trait-method | 217 |  | `(Styled[S]).FlexBasis` | todo |
| `Styled::flex_grow` | trait-method | 224 |  | `(Styled[S]).FlexGrow` | todo |
| `Styled::flex_grow_0` | trait-method | 231 |  | `(Styled[S]).FlexGrow0` | todo |
| `Styled::flex_grow_1` | trait-method | 238 |  | `(Styled[S]).FlexGrow1` | todo |
| `Styled::flex_shrink` | trait-method | 245 |  | `(Styled[S]).FlexShrink` | todo |
| `Styled::flex_shrink_0` | trait-method | 252 |  | `(Styled[S]).FlexShrink0` | todo |
| `Styled::flex_shrink_1` | trait-method | 259 |  | `(Styled[S]).FlexShrink1` | todo |
| `Styled::flex_wrap` | trait-method | 266 |  | `(Styled[S]).FlexWrap` | todo |
| `Styled::flex_wrap_reverse` | trait-method | 273 |  | `(Styled[S]).FlexWrapReverse` | todo |
| `Styled::flex_nowrap` | trait-method | 280 |  | `(Styled[S]).FlexNowrap` | todo |
| `Styled::items_start` | trait-method | 287 |  | `(Styled[S]).ItemsStart` | todo |
| `Styled::items_end` | trait-method | 294 |  | `(Styled[S]).ItemsEnd` | todo |
| `Styled::items_center` | trait-method | 301 |  | `(Styled[S]).ItemsCenter` | todo |
| `Styled::items_baseline` | trait-method | 308 |  | `(Styled[S]).ItemsBaseline` | todo |
| `Styled::items_stretch` | trait-method | 315 |  | `(Styled[S]).ItemsStretch` | todo |
| `Styled::self_start` | trait-method | 322 |  | `(Styled[S]).SelfStart` | todo |
| `Styled::self_end` | trait-method | 329 |  | `(Styled[S]).SelfEnd` | todo |
| `Styled::self_flex_start` | trait-method | 336 |  | `(Styled[S]).SelfFlexStart` | todo |
| `Styled::self_flex_end` | trait-method | 343 |  | `(Styled[S]).SelfFlexEnd` | todo |
| `Styled::self_center` | trait-method | 350 |  | `(Styled[S]).SelfCenter` | todo |
| `Styled::self_baseline` | trait-method | 357 |  | `(Styled[S]).SelfBaseline` | todo |
| `Styled::self_stretch` | trait-method | 364 |  | `(Styled[S]).SelfStretch` | todo |
| `Styled::justify_start` | trait-method | 371 |  | `(Styled[S]).JustifyStart` | todo |
| `Styled::justify_end` | trait-method | 378 |  | `(Styled[S]).JustifyEnd` | todo |
| `Styled::justify_center` | trait-method | 385 |  | `(Styled[S]).JustifyCenter` | todo |
| `Styled::justify_between` | trait-method | 393 |  | `(Styled[S]).JustifyBetween` | todo |
| `Styled::justify_around` | trait-method | 401 |  | `(Styled[S]).JustifyAround` | todo |
| `Styled::justify_evenly` | trait-method | 411 |  | `(Styled[S]).JustifyEvenly` | todo |
| `Styled::content_normal` | trait-method | 418 |  | `(Styled[S]).ContentNormal` | todo |
| `Styled::content_center` | trait-method | 425 |  | `(Styled[S]).ContentCenter` | todo |
| `Styled::content_start` | trait-method | 432 |  | `(Styled[S]).ContentStart` | todo |
| `Styled::content_end` | trait-method | 439 |  | `(Styled[S]).ContentEnd` | todo |
| `Styled::content_between` | trait-method | 447 |  | `(Styled[S]).ContentBetween` | todo |
| `Styled::content_around` | trait-method | 455 |  | `(Styled[S]).ContentAround` | todo |
| `Styled::content_evenly` | trait-method | 463 |  | `(Styled[S]).ContentEvenly` | todo |
| `Styled::content_stretch` | trait-method | 470 |  | `(Styled[S]).ContentStretch` | todo |
| `Styled::aspect_ratio` | trait-method | 477 |  | `(Styled[S]).AspectRatio` | todo |
| `Styled::aspect_square` | trait-method | 484 |  | `(Styled[S]).AspectSquare` | todo |
| `Styled::bg` | trait-method | 490 |  | `(Styled[S]).Bg` | todo |
| `Styled::border_dashed` | trait-method | 500 |  | `(Styled[S]).BorderDashed` | todo |
| `Styled::text_style` | trait-method | 506 |  | `(Styled[S]).TextStyle` | todo |
| `Styled::text_color` | trait-method | 514 |  | `(Styled[S]).TextColor` | todo |
| `Styled::font_weight` | trait-method | 522 |  | `(Styled[S]).FontWeight` | todo |
| `Styled::text_bg` | trait-method | 530 |  | `(Styled[S]).TextBg` | todo |
| `Styled::text_size` | trait-method | 538 |  | `(Styled[S]).TextSize` | todo |
| `Styled::text_xs` | trait-method | 545 |  | `(Styled[S]).TextXs` | todo |
| `Styled::text_sm` | trait-method | 552 |  | `(Styled[S]).TextSm` | todo |
| `Styled::text_base` | trait-method | 559 |  | `(Styled[S]).TextBase` | todo |
| `Styled::text_lg` | trait-method | 566 |  | `(Styled[S]).TextLg` | todo |
| `Styled::text_xl` | trait-method | 573 |  | `(Styled[S]).TextXl` | todo |
| `Styled::text_2xl` | trait-method | 580 |  | `(Styled[S]).Text2xl` | todo |
| `Styled::text_3xl` | trait-method | 587 |  | `(Styled[S]).Text3xl` | todo |
| `Styled::italic` | trait-method | 594 |  | `(Styled[S]).Italic` | todo |
| `Styled::not_italic` | trait-method | 601 |  | `(Styled[S]).NotItalic` | todo |
| `Styled::underline` | trait-method | 608 |  | `(Styled[S]).Underline` | todo |
| `Styled::line_through` | trait-method | 619 |  | `(Styled[S]).LineThrough` | todo |
| `Styled::text_decoration_none` | trait-method | 631 |  | `(Styled[S]).TextDecorationNone` | todo |
| `Styled::text_decoration_color` | trait-method | 637 |  | `(Styled[S]).TextDecorationColor` | todo |
| `Styled::text_decoration_solid` | trait-method | 646 |  | `(Styled[S]).TextDecorationSolid` | todo |
| `Styled::text_decoration_wavy` | trait-method | 655 |  | `(Styled[S]).TextDecorationWavy` | todo |
| `Styled::text_decoration_0` | trait-method | 664 |  | `(Styled[S]).TextDecoration0` | todo |
| `Styled::text_decoration_1` | trait-method | 673 |  | `(Styled[S]).TextDecoration1` | todo |
| `Styled::text_decoration_2` | trait-method | 682 |  | `(Styled[S]).TextDecoration2` | todo |
| `Styled::text_decoration_4` | trait-method | 691 |  | `(Styled[S]).TextDecoration4` | todo |
| `Styled::text_decoration_8` | trait-method | 700 |  | `(Styled[S]).TextDecoration8` | todo |
| `Styled::font_family` | trait-method | 708 |  | `(Styled[S]).FontFamily` | todo |
| `Styled::font_features` | trait-method | 714 |  | `(Styled[S]).FontFeatures` | todo |
| `Styled::font` | trait-method | 720 |  | `(Styled[S]).Font` | todo |
| `Styled::line_height` | trait-method | 740 |  | `(Styled[S]).LineHeight` | todo |
| `Styled::opacity` | trait-method | 746 |  | `(Styled[S]).Opacity` | todo |
| `Styled::grid_cols` | trait-method | 752 |  | `(Styled[S]).GridCols` | todo |
| `Styled::grid_cols_min_content` | trait-method | 762 |  | `(Styled[S]).GridColsMinContent` | todo |
| `Styled::grid_cols_max_content` | trait-method | 771 |  | `(Styled[S]).GridColsMaxContent` | todo |
| `Styled::grid_rows` | trait-method | 780 |  | `(Styled[S]).GridRows` | todo |
| `Styled::grid_rows_min_content` | trait-method | 790 |  | `(Styled[S]).GridRowsMinContent` | todo |
| `Styled::grid_rows_max_content` | trait-method | 799 |  | `(Styled[S]).GridRowsMaxContent` | todo |
| `Styled::col_start` | trait-method | 808 |  | `(Styled[S]).ColStart` | todo |
| `Styled::col_start_auto` | trait-method | 815 |  | `(Styled[S]).ColStartAuto` | todo |
| `Styled::col_end` | trait-method | 822 |  | `(Styled[S]).ColEnd` | todo |
| `Styled::col_end_auto` | trait-method | 829 |  | `(Styled[S]).ColEndAuto` | todo |
| `Styled::col_span` | trait-method | 836 |  | `(Styled[S]).ColSpan` | todo |
| `Styled::col_span_full` | trait-method | 843 |  | `(Styled[S]).ColSpanFull` | todo |
| `Styled::row_start` | trait-method | 850 |  | `(Styled[S]).RowStart` | todo |
| `Styled::row_start_auto` | trait-method | 857 |  | `(Styled[S]).RowStartAuto` | todo |
| `Styled::row_end` | trait-method | 864 |  | `(Styled[S]).RowEnd` | todo |
| `Styled::row_end_auto` | trait-method | 871 |  | `(Styled[S]).RowEndAuto` | todo |
| `Styled::row_span` | trait-method | 878 |  | `(Styled[S]).RowSpan` | todo |
| `Styled::row_span_full` | trait-method | 885 |  | `(Styled[S]).RowSpanFull` | todo |
| `Styled::debug` | trait-method | 893 | debug | `(Styled[S]).Debug` | todo |
| `Styled::debug_below` | trait-method | 900 | debug | `(Styled[S]).DebugBelow` | todo |
| `Styled::w` + `w_{s}` / `w_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:846 |  | `(Styled[S]).W`, `.W{S}`, `.WNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::h` + `h_{s}` / `h_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:852 |  | `(Styled[S]).H`, `.H{S}`, `.HNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::size` + `size_{s}` / `size_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:858 |  | `(Styled[S]).Size`, `.Size{S}`, `.SizeNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::min_size` + `min_size_{s}` / `min_size_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:864 |  | `(Styled[S]).MinSize`, `.MinSize{S}`, `.MinSizeNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::min_w` + `min_w_{s}` / `min_w_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:870 |  | `(Styled[S]).MinW`, `.MinW{S}`, `.MinWNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::min_h` + `min_h_{s}` / `min_h_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:878 |  | `(Styled[S]).MinH`, `.MinH{S}`, `.MinHNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::max_size` + `max_size_{s}` / `max_size_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:884 |  | `(Styled[S]).MaxSize`, `.MaxSize{S}`, `.MaxSizeNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::max_w` + `max_w_{s}` / `max_w_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:892 |  | `(Styled[S]).MaxW`, `.MaxW{S}`, `.MaxWNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::max_h` + `max_h_{s}` / `max_h_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (style_helpers!) | styles.rs:900 |  | `(Styled[S]).MaxH`, `.MaxH{S}`, `.MaxHNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::gap` + `gap_{s}` / `gap_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (style_helpers!) | styles.rs:906 |  | `(Styled[S]).Gap`, `.Gap{S}`, `.GapNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::gap_x` + `gap_x_{s}` / `gap_x_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (style_helpers!) | styles.rs:912 |  | `(Styled[S]).GapX`, `.GapX{S}`, `.GapXNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::gap_y` + `gap_y_{s}` / `gap_y_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (style_helpers!) | styles.rs:918 |  | `(Styled[S]).GapY`, `.GapY{S}`, `.GapYNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded` + `rounded_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1164 |  | `(Styled[S]).Rounded`, `.Rounded{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_t` + `rounded_t_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1174 |  | `(Styled[S]).RoundedT`, `.RoundedT{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_b` + `rounded_b_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1182 |  | `(Styled[S]).RoundedB`, `.RoundedB{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_r` + `rounded_r_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1190 |  | `(Styled[S]).RoundedR`, `.RoundedR{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_l` + `rounded_l_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1198 |  | `(Styled[S]).RoundedL`, `.RoundedL{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_tl` + `rounded_tl_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1206 |  | `(Styled[S]).RoundedTl`, `.RoundedTl{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_tr` + `rounded_tr_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1211 |  | `(Styled[S]).RoundedTr`, `.RoundedTr{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_bl` + `rounded_bl_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1216 |  | `(Styled[S]).RoundedBl`, `.RoundedBl{S}`; full list in styled-methods.tsv | todo |
| `Styled::rounded_br` + `rounded_br_{s}` family, 10 methods (corner_suffixes) | macro-family (style_helpers!) | styles.rs:1221 |  | `(Styled[S]).RoundedBr`, `.RoundedBr{S}`; full list in styled-methods.tsv | todo |
| `Styled::visible` | macro-method (visibility_style_methods!) | styles.rs:56 |  | `(Styled[S]).Visible` | todo |
| `Styled::invisible` | macro-method (visibility_style_methods!) | styles.rs:63 |  | `(Styled[S]).Invisible` | todo |
| `Styled::m` + `m_{s}` / `m_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:702 |  | `(Styled[S]).M`, `.M{S}`, `.MNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::mt` + `mt_{s}` / `mt_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:713 |  | `(Styled[S]).Mt`, `.Mt{S}`, `.MtNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::mb` + `mb_{s}` / `mb_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:719 |  | `(Styled[S]).Mb`, `.Mb{S}`, `.MbNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::my` + `my_{s}` / `my_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:725 |  | `(Styled[S]).My`, `.My{S}`, `.MyNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::mx` + `mx_{s}` / `mx_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:731 |  | `(Styled[S]).Mx`, `.Mx{S}`, `.MxNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::ml` + `ml_{s}` / `ml_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:737 |  | `(Styled[S]).Ml`, `.Ml{S}`, `.MlNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::mr` + `mr_{s}` / `mr_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (margin_style_methods!) | styles.rs:743 |  | `(Styled[S]).Mr`, `.Mr{S}`, `.MrNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::p` + `p_{s}` / `p_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:754 |  | `(Styled[S]).P`, `.P{S}`, `.PNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::pt` + `pt_{s}` / `pt_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:765 |  | `(Styled[S]).Pt`, `.Pt{S}`, `.PtNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::pb` + `pb_{s}` / `pb_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:771 |  | `(Styled[S]).Pb`, `.Pb{S}`, `.PbNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::px` + `px_{s}` / `px_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:777 |  | `(Styled[S]).Px`, `.Px{S}`, `.PxNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::py` + `py_{s}` / `py_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:783 |  | `(Styled[S]).Py`, `.Py{S}`, `.PyNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::pl` + `pl_{s}` / `pl_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:789 |  | `(Styled[S]).Pl`, `.Pl{S}`, `.PlNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::pr` + `pr_{s}` / `pr_neg_{s}` family, 91 methods (box_style_suffixes minus auto) | macro-family (padding_style_methods!) | styles.rs:795 |  | `(Styled[S]).Pr`, `.Pr{S}`, `.PrNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::relative` | macro-method (position_style_methods!) | styles.rs:111 |  | `(Styled[S]).Relative` | todo |
| `Styled::absolute` | macro-method (position_style_methods!) | styles.rs:118 |  | `(Styled[S]).Absolute` | todo |
| `Styled::inset` + `inset_{s}` / `inset_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (position_style_methods!) | styles.rs:806 |  | `(Styled[S]).Inset`, `.Inset{S}`, `.InsetNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::top` + `top_{s}` / `top_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (position_style_methods!) | styles.rs:817 |  | `(Styled[S]).Top`, `.Top{S}`, `.TopNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::bottom` + `bottom_{s}` / `bottom_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (position_style_methods!) | styles.rs:823 |  | `(Styled[S]).Bottom`, `.Bottom{S}`, `.BottomNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::left` + `left_{s}` / `left_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (position_style_methods!) | styles.rs:829 |  | `(Styled[S]).Left`, `.Left{S}`, `.LeftNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::right` + `right_{s}` / `right_neg_{s}` family, 92 methods (box_style_suffixes) | macro-family (position_style_methods!) | styles.rs:835 |  | `(Styled[S]).Right`, `.Right{S}`, `.RightNeg{S}`; full list in styled-methods.tsv | todo |
| `Styled::overflow_hidden` | macro-method (overflow_style_methods!) | styles.rs:135 |  | `(Styled[S]).OverflowHidden` | todo |
| `Styled::overflow_x_hidden` | macro-method (overflow_style_methods!) | styles.rs:143 |  | `(Styled[S]).OverflowXHidden` | todo |
| `Styled::overflow_y_hidden` | macro-method (overflow_style_methods!) | styles.rs:150 |  | `(Styled[S]).OverflowYHidden` | todo |
| `Styled::cursor` | macro-method (cursor_style_methods!) | styles.rs:164 |  | `(Styled[S]).Cursor` | todo |
| `Styled::cursor_default` | macro-method (cursor_style_methods!) | styles.rs:171 |  | `(Styled[S]).CursorDefault` | todo |
| `Styled::cursor_pointer` | macro-method (cursor_style_methods!) | styles.rs:178 |  | `(Styled[S]).CursorPointer` | todo |
| `Styled::cursor_text` | macro-method (cursor_style_methods!) | styles.rs:185 |  | `(Styled[S]).CursorText` | todo |
| `Styled::cursor_move` | macro-method (cursor_style_methods!) | styles.rs:192 |  | `(Styled[S]).CursorMove` | todo |
| `Styled::cursor_not_allowed` | macro-method (cursor_style_methods!) | styles.rs:199 |  | `(Styled[S]).CursorNotAllowed` | todo |
| `Styled::cursor_context_menu` | macro-method (cursor_style_methods!) | styles.rs:206 |  | `(Styled[S]).CursorContextMenu` | todo |
| `Styled::cursor_crosshair` | macro-method (cursor_style_methods!) | styles.rs:213 |  | `(Styled[S]).CursorCrosshair` | todo |
| `Styled::cursor_vertical_text` | macro-method (cursor_style_methods!) | styles.rs:220 |  | `(Styled[S]).CursorVerticalText` | todo |
| `Styled::cursor_alias` | macro-method (cursor_style_methods!) | styles.rs:227 |  | `(Styled[S]).CursorAlias` | todo |
| `Styled::cursor_copy` | macro-method (cursor_style_methods!) | styles.rs:234 |  | `(Styled[S]).CursorCopy` | todo |
| `Styled::cursor_no_drop` | macro-method (cursor_style_methods!) | styles.rs:241 |  | `(Styled[S]).CursorNoDrop` | todo |
| `Styled::cursor_grab` | macro-method (cursor_style_methods!) | styles.rs:248 |  | `(Styled[S]).CursorGrab` | todo |
| `Styled::cursor_grabbing` | macro-method (cursor_style_methods!) | styles.rs:255 |  | `(Styled[S]).CursorGrabbing` | todo |
| `Styled::cursor_ew_resize` | macro-method (cursor_style_methods!) | styles.rs:262 |  | `(Styled[S]).CursorEwResize` | todo |
| `Styled::cursor_ns_resize` | macro-method (cursor_style_methods!) | styles.rs:269 |  | `(Styled[S]).CursorNsResize` | todo |
| `Styled::cursor_nesw_resize` | macro-method (cursor_style_methods!) | styles.rs:276 |  | `(Styled[S]).CursorNeswResize` | todo |
| `Styled::cursor_nwse_resize` | macro-method (cursor_style_methods!) | styles.rs:283 |  | `(Styled[S]).CursorNwseResize` | todo |
| `Styled::cursor_col_resize` | macro-method (cursor_style_methods!) | styles.rs:290 |  | `(Styled[S]).CursorColResize` | todo |
| `Styled::cursor_row_resize` | macro-method (cursor_style_methods!) | styles.rs:297 |  | `(Styled[S]).CursorRowResize` | todo |
| `Styled::cursor_n_resize` | macro-method (cursor_style_methods!) | styles.rs:304 |  | `(Styled[S]).CursorNResize` | todo |
| `Styled::cursor_e_resize` | macro-method (cursor_style_methods!) | styles.rs:311 |  | `(Styled[S]).CursorEResize` | todo |
| `Styled::cursor_s_resize` | macro-method (cursor_style_methods!) | styles.rs:318 |  | `(Styled[S]).CursorSResize` | todo |
| `Styled::cursor_w_resize` | macro-method (cursor_style_methods!) | styles.rs:325 |  | `(Styled[S]).CursorWResize` | todo |
| `Styled::border_color` | macro-method (border_style_methods!) | styles.rs:369 |  | `(Styled[S]).BorderColor` | todo |
| `Styled::border` + `border_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1281 |  | `(Styled[S]).Border`, `.Border{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_t` + `border_t_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1291 |  | `(Styled[S]).BorderT`, `.BorderT{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_b` + `border_b_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1296 |  | `(Styled[S]).BorderB`, `.BorderB{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_r` + `border_r_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1301 |  | `(Styled[S]).BorderR`, `.BorderR{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_l` + `border_l_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1306 |  | `(Styled[S]).BorderL`, `.BorderL{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_x` + `border_x_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1311 |  | `(Styled[S]).BorderX`, `.BorderX{S}`; full list in styled-methods.tsv | todo |
| `Styled::border_y` + `border_y_{s}` family, 18 methods (border_suffixes) | macro-family (border_style_methods!) | styles.rs:1319 |  | `(Styled[S]).BorderY`, `.BorderY{S}`; full list in styled-methods.tsv | todo |
| `Styled::shadow` | macro-method (box_shadow_style_methods!) | styles.rs:390 |  | `(Styled[S]).Shadow` | todo |
| `Styled::shadow_none` | macro-method (box_shadow_style_methods!) | styles.rs:397 |  | `(Styled[S]).ShadowNone` | todo |
| `Styled::shadow_2xs` | macro-method (box_shadow_style_methods!) | styles.rs:404 |  | `(Styled[S]).Shadow2xs` | todo |
| `Styled::shadow_xs` | macro-method (box_shadow_style_methods!) | styles.rs:416 |  | `(Styled[S]).ShadowXs` | todo |
| `Styled::shadow_sm` | macro-method (box_shadow_style_methods!) | styles.rs:428 |  | `(Styled[S]).ShadowSm` | todo |
| `Styled::shadow_md` | macro-method (box_shadow_style_methods!) | styles.rs:441 |  | `(Styled[S]).ShadowMd` | todo |
| `Styled::shadow_lg` | macro-method (box_shadow_style_methods!) | styles.rs:454 |  | `(Styled[S]).ShadowLg` | todo |
| `Styled::shadow_xl` | macro-method (box_shadow_style_methods!) | styles.rs:467 |  | `(Styled[S]).ShadowXl` | todo |
| `Styled::shadow_2xl` | macro-method (box_shadow_style_methods!) | styles.rs:480 |  | `(Styled[S]).Shadow2xl` | todo |

## gpui/src/taffy.rs

Go target: `taffy.go` (package `gpui`), tests in `taffy_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TaffyLayoutEngine` | struct | 31 |  | `TaffyLayoutEngine` | todo |
| `TaffyLayoutEngine::new` | assoc-fn | 43 |  | `NewTaffyLayoutEngine` | todo |
| `TaffyLayoutEngine::clear` | method | 55 |  | `(TaffyLayoutEngine).Clear` | todo |
| `TaffyLayoutEngine::request_layout` | method | 62 |  | `(TaffyLayoutEngine).RequestLayout` | todo |
| `TaffyLayoutEngine::request_measured_layout` | method | 85 |  | `(TaffyLayoutEngine).RequestMeasuredLayout` | todo |
| `TaffyLayoutEngine::stretch_auto_size_to_fill` | method | 115 |  | `(TaffyLayoutEngine).StretchAutoSizeToFill` | todo |
| `TaffyLayoutEngine::compute_layout` | method | 187 |  | `(TaffyLayoutEngine).ComputeLayout` | todo |
| `TaffyLayoutEngine::layout_bounds` | method | 356 |  | `(TaffyLayoutEngine).LayoutBounds` | todo |
| `LayoutId` | struct | 396 |  | `LayoutID` | todo |
| `AvailableSpace` | enum | 689 |  | `AvailableSpace` | todo |
| `AvailableSpace::min_size` | assoc-fn | 713 |  | `AvailableSpaceMinSize` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `border_widths_to_taffy_use_stroke_snapping` | test | 761 |  | `TestTaffy_BorderWidthsToTaffyUseStrokeSnapping` | todo |

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-PRIMITIVE | `Primitive` is a sealed interface over wrapper variants (`PrimitiveQuad{Quad}`, `PrimitivePath{PaintedPath}`, ...); `Bounds` and `ContentMask` are methods of the wrappers. `impl Into<Primitive>` is the `IntoPrimitive` interface, implemented by every primitive struct and every wrapper, so `Scene.InsertPrimitive` takes either | `gpui/src/scene.rs:493-560`, `:220` | the primitive structs have exported `Bounds` / `ContentMask` fields, so they cannot also carry methods of those names |
| D-SURFACEBUF | `PaintSurface.ImageBuffer` (`CFRetained<CVPixelBuffer>`, macOS and iOS only) is a `uintptr` CVPixelBufferRef present on every GOOS, zero elsewhere; the Apple back end owns its retain and release | `gpui/src/scene.rs:1169-1175` | the root package imports no Apple bindings (DESIGN 4) and the struct does not vary by GOOS |
| D-SPRINGTARGET | `SpringTarget[O]` / `Interpolate[T]` are method interfaces implemented by `Pixels`, `Rems`, `AnimationPhase` (and `Rgba`, `Hsla` for Interpolate); `float32` and `bool` targets and `float32` interpolation go through the unexported dispatchers `springTargetOf` / `springResolveOf` / `interpolateValue`, which panic for other types. `AnimationPhase` is `type AnimationPhase float32`; `RangeInclusive<f32>` parameters are `start, end float32`; `SpringAnimation::new` returns `SpringAnimation[struct{}]` and `To` is a generic method; `Duration::MAX` is `time.Duration(math.MaxInt64)` (about 292 years) | `gpui/src/spring.rs:264-420`, `:472-535` | Go cannot add methods to `float32` / `bool` or constrain a receiver's type parameter in one method; `time.Duration` is an int64 of nanoseconds |
