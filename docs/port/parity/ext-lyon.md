# Parity: ext-lyon

Scope: the lyon_path 1.0.19 / lyon_geom 1.0.20 subset that `gpui/src/path_builder.rs:3-9` imports (Path, Builder, BuilderImpl, WithSvg, SvgPathBuilder, events, ArcFlags, Polygon, FillRule / LineCap / LineJoin, math Point / Vector / Transform / Angle, CubicBezierSegment, QuadraticBezierSegment, SVG arc conversion), plus the euclid 0.22.14 pieces they need. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/{lyon_path-1.0.19,lyon_geom-1.0.20,euclid-0.22.14}/`.

Notes on the port:

- Two packages mirror the two crates: `internal/lyon` (lyon_path) and `internal/lyon/geom` (lyon_geom plus the euclid subset in `euclid.go`). `internal/lyon` re-exports the math types as aliases.
- Everything is f32 (D-LYON-F32). Arithmetic keeps upstream's operation order, and every product that feeds a sum is wrapped in `float32(...)` so Go never contracts it into an FMA (Rust never does). `TestParity_RustGoldenEvents` pins the emitted events bit for bit against output of the Rust crates (arcs, relative arcs, smooth curves, transform, cubic-to-quadratic, monotonic split); it passes on amd64, amd64 with GOAMD64=v3 and arm64 (qemu).
- Items or tests outside the subset are `n/a:D-LYON-SUBSET`. Only items in the ported files are listed; the unported modules (`commands`, `iterator`, `path_buffer`, lyon_geom `line`, `triangle`, `segment`, `utils`, `cubic_bezier_intersections`) are out of scope as a whole.

## lyon_path/src/lib.rs

Go target: `internal/lyon/lib.go` (package `lyon`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use lyon_geom as geom` | use | 50 |  | - | dev:D-LYON-EUCLID |
| `pub mod builder` | mod | 56 |  | `PathBuilder` | done |
| `pub mod commands` | mod | 57 |  | - | n/a:D-LYON-SUBSET |
| `pub mod iterator` | mod | 59 |  | - | n/a:D-LYON-SUBSET |
| `pub mod path` | mod | 62 |  | `Path` | done |
| `pub mod path_buffer` | mod | 64 |  | - | n/a:D-LYON-SUBSET |
| `pub mod polygon` | mod | 65 |  | `Polygon` | done |
| `pub mod private` | mod | 68 |  | `DebugValidator` | done |
| `pub use crate::geom::ArcFlags` | use | 73 |  | `ArcFlags` | done |
| `pub use crate::polygon::{IdPolygon, Polygon}` (`IdPolygon`) | use | 79 |  | - | n/a:D-LYON-SUBSET |
| `math::Point` | type | 100 |  | `Point` | done |
| `math::Vector` | type | 103 |  | `Vector` | done |
| `math::Size` | type | 106 |  | - | n/a:D-LYON-SUBSET |
| `math::Box2D` | type | 109 |  | - | n/a:D-LYON-SUBSET |
| `math::Transform` | type | 112 |  | `Transform` | done |
| `math::Rotation` | type | 115 |  | `Rotation` | done |
| `math::Translation` | type | 118 |  | - | n/a:D-LYON-SUBSET |
| `math::Scale` | type | 121 |  | - | n/a:D-LYON-SUBSET |
| `math::Angle` | type | 124 |  | `Angle` | done |
| `math::vector` | fn | 128 |  | `NewVector` | done |
| `math::point` | fn | 134 |  | `NewPoint` | done |
| `math::size` | fn | 140 |  | - | n/a:D-LYON-SUBSET |
| `LineCap` | enum | 170 |  | `LineCap` | done |
| `LineJoin` | enum | 193 |  | `LineJoin` | done |
| `Side` | enum | 224 |  | `Side` | done |
| `Side::opposite` | method | 231 |  | `(Side).Opposite` | done |
| `Side::is_positive` | method | 239 |  | `(Side).IsPositive` | done |
| `Side::is_negative` | method | 244 |  | `(Side).IsNegative` | done |
| `Side::to_f32` | method | 249 |  | `(Side).ToF32` | done |
| `FillRule` | enum | 262 |  | `FillRule` | done |
| `FillRule::is_in` | method | 269 |  | `(FillRule).IsIn` | done |
| `FillRule::is_out` | method | 277 |  | `(FillRule).IsOut` | done |
| `Winding` | enum | 287 |  | `Winding` | done |
| `ControlPointId` | struct | 295 |  | `ControlPointID` | done |
| `ControlPointId::INVALID` | const | 298 |  | `ControlPointIDInvalid` | done |
| `ControlPointId::offset` | method | 299 |  | `(ControlPointID).Offset` | done |
| `ControlPointId::to_usize` | method | 302 |  | `(ControlPointID).ToUsize` | done |
| `ControlPointId::from_usize` | assoc-fn | 305 |  | `ControlPointIDFromUsize` | done |
| `EndpointId` | struct | 319 |  | `EndpointID` | done |
| `EndpointId::INVALID` | const | 321 |  | `EndpointIDInvalid` | done |
| `EndpointId::offset` | method | 322 |  | `(EndpointID).Offset` | done |
| `EndpointId::to_usize` | method | 325 |  | `(EndpointID).ToUsize` | done |
| `EndpointId::from_usize` | assoc-fn | 328 |  | `EndpointIDFromUsize` | done |
| `EventId` | struct | 342 |  | `EventID` | done |
| `EventId::INVALID` | const | 345 |  | `EventIDInvalid` | done |
| `EventId::to_usize` | method | 346 |  | `(EventID).ToUsize` | done |
| `Position` | trait | 353 |  | - | n/a:D-LYON-SUBSET |
| `PositionStore` | trait | 391 |  | - | n/a:D-LYON-SUBSET |
| `AttributeStore` | trait | 409 |  | - | n/a:D-LYON-SUBSET |
| `AttributeSlice` | struct | 432 |  | - | n/a:D-LYON-SUBSET |
| `AttributeIndex` | type | 459 |  | - | n/a:D-LYON-SUBSET |
| `Attributes` | type | 461 |  | `Attributes` | done |
| `NO_ATTRIBUTES` | const | 463 |  | - | dev:D-LYON-NOATTR |

## lyon_path/src/events.rs

Go target: `internal/lyon/events.go` (package `lyon`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Event` | enum | 8 |  | `Event` | dev:D-LYON-EVENT |
| `PathEvent` | type | 35 |  | `PathEvent` | done |
| `IdEvent` | type | 38 |  | `IDEvent` | done |
| `Event::is_edge` | method | 41 |  | `(Event).IsEdge` | done |
| `Event::from` | method | 51 |  | `(Event).FromEndpoint` | dev:D-LYON-EVENT |
| `Event::to` | method | 64 |  | `(Event).ToEndpoint` | dev:D-LYON-EVENT |
| `Event::with_points` | method | 77 |  | - | n/a:D-LYON-SUBSET |
| `PathEvent::transformed` | method | 114 |  | `PathEventTransformed` | done |

## lyon_path/src/polygon.rs

Go target: `internal/lyon/polygon.go` (package `lyon`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Polygon` | struct | 30 |  | `Polygon` | done |
| `Polygon::iter` | method | 37 |  | - | n/a:D-LYON-SUBSET |
| `Polygon::id_iter` | method | 47 |  | - | n/a:D-LYON-SUBSET |
| `Polygon::path_events` | method | 52 |  | - | n/a:D-LYON-SUBSET |
| `Polygon::event` | method | 65 |  | - | n/a:D-LYON-SUBSET |
| `IdPolygon` | struct | 95 |  | - | n/a:D-LYON-SUBSET |
| `IdPolygonIter` | struct | 134 |  | - | n/a:D-LYON-SUBSET |
| `PolygonIter` | struct | 172 |  | - | n/a:D-LYON-SUBSET |
| `PathEvents` | struct | 207 |  | - | n/a:D-LYON-SUBSET |
| `PolygonIdIter` | struct | 244 |  | - | n/a:D-LYON-SUBSET |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `event_ids` | test | 304 |  | - | n/a:D-LYON-SUBSET |
| `polygon_slice_id_ite` | test | 377 |  | - | n/a:D-LYON-SUBSET |

## lyon_path/src/private.rs

Go target: `internal/lyon/private.go` (package `lyon`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `DebugValidator` | struct | 13 |  | `DebugValidator` | dev:D-LYON-DEBUG |
| `DebugValidator::new` | assoc-fn | 20 |  | `NewDebugValidator` | done |
| `DebugValidator::begin` | method | 25 |  | `(DebugValidator).Begin` | done |
| `DebugValidator::end` | method | 34 |  | `(DebugValidator).End` | done |
| `DebugValidator::edge` | method | 43 |  | `(DebugValidator).Edge` | done |
| `DebugValidator::build` | method | 51 |  | `(DebugValidator).Build` | done |
| `flatten_quadratic_bezier` | fn | 57 |  | - | n/a:D-LYON-SUBSET |
| `flatten_cubic_bezier` | fn | 87 |  | - | n/a:D-LYON-SUBSET |

## lyon_path/src/path.rs

Go target: `internal/lyon/path.go` (package `lyon`), tests in `path_test.go` and `parity_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Verb` (pub(crate)) | enum | 29 |  | `verb` | done |
| `Path` | struct | 75 |  | `Path` | done |
| `PathSlice` | struct | 83 |  | - | n/a:D-LYON-SUBSET |
| `Builder` | type | 89 |  | `Builder` | done |
| `Path::builder` | assoc-fn | 93 |  | `NewBuilder` | dev:D-NAME |
| `Path::builder_with_attributes` | assoc-fn | 99 |  | - | n/a:D-LYON-SUBSET |
| `Path::svg_builder` | assoc-fn | 105 |  | `PathSvgBuilder` | done |
| `Path::new` | assoc-fn | 111 |  | `NewPath` | done |
| `Path::with_attributes` | assoc-fn | 120 |  | - | n/a:D-LYON-SUBSET |
| `Path::as_slice` | method | 130 |  | - | n/a:D-LYON-SUBSET |
| `Path::attributes` | method | 140 |  | - | n/a:D-LYON-SUBSET |
| `Path::iter` | method | 145 |  | `(Path).Iter` | done |
| `Path::id_iter` | method | 150 |  | - | n/a:D-LYON-SUBSET |
| `Path::iter_with_attributes` | method | 155 |  | - | n/a:D-LYON-SUBSET |
| `Path::transformed` | method | 161 |  | `(Path).Transformed` | done |
| `Path::reversed` | method | 168 |  | - | n/a:D-LYON-SUBSET |
| `Path::first_endpoint` | method | 174 |  | - | n/a:D-LYON-SUBSET |
| `Path::last_endpoint` | method | 180 |  | - | n/a:D-LYON-SUBSET |
| `impl FromIterator<PathEvent> for Path` | trait-impl | 219 |  | `PathFromEvents` | done |
| `BuilderImpl` | struct | 472 |  | `BuilderImpl` | done |
| `BuilderImpl::new` | assoc-fn | 480 |  | `NewBuilderImpl` | done |
| `BuilderImpl::with_capacity` | assoc-fn | 489 |  | `BuilderImplWithCapacity` | done |
| `BuilderImpl::with_svg` | method | 498 |  | `(BuilderImpl).WithSvg` | done |
| `BuilderImpl::extend_from_paths` | method | 504 |  | - | n/a:D-LYON-SUBSET |
| `impl PathBuilder for BuilderImpl` | trait-impl | 516 |  | `(BuilderImpl).Begin` | done |
| `impl Build for BuilderImpl` | trait-impl | 600 |  | `(BuilderImpl).Build` | done |
| `impl Default for BuilderImpl` | trait-impl | 613 |  | `NewBuilderImpl` | done |
| `BuilderWithAttributes` | struct | 624 |  | - | n/a:D-LYON-SUBSET |
| `nan_check` | fn | 810 |  | `nanCheck` | dev:D-LYON-DEBUG |
| `Iter` | struct | 817 |  | `pathIter` | done |
| `PointIter` | struct | 917 |  | `pointIter` | done |
| `IterWithAttributes` | struct | 967 |  | - | n/a:D-LYON-SUBSET |
| `IdIter` | struct | 1177 |  | `idIter` | done |
| `Reversed` | struct | 1309 |  | - | n/a:D-LYON-SUBSET |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_reverse_path_simple` | test | 1463 |  | - | n/a:D-LYON-SUBSET |
| `test_reverse_path_1` | test | 1529 |  | - | n/a:D-LYON-SUBSET |
| `test_reverse_path_no_close` | test | 1689 |  | - | n/a:D-LYON-SUBSET |
| `test_reverse_empty_path` | test | 1732 |  | - | n/a:D-LYON-SUBSET |
| `test_reverse_single_point` | test | 1738 |  | - | n/a:D-LYON-SUBSET |
| `test_path_builder_1` | test | 1763 |  | - | n/a:D-LYON-SUBSET |
| `test_path_builder_empty` | test | 1958 |  | - | n/a:D-LYON-SUBSET |
| `test_path_builder_empty_begin` | test | 1966 |  | - | n/a:D-LYON-SUBSET |
| `test_extend_from_paths` | test | 2024 |  | - | n/a:D-LYON-SUBSET |
| `flattened_custom_attributes` | test | 2106 |  | - | n/a:D-LYON-SUBSET |
| `first_last` | test | 2145 |  | - | n/a:D-LYON-SUBSET |
| `id_events` | test | 2174 |  | - | n/a:D-LYON-SUBSET |

## lyon_path/src/builder.rs

Go target: `internal/lyon/builder.go` (package `lyon`), tests in `builder_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `BorderRadii` | struct | 100 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes` | struct | 138 |  | `NoAttributes` | done |
| `NoAttributes::wrap` | assoc-fn | 144 |  | `NoAttributesWrap` | done |
| `NoAttributes::new` | assoc-fn | 149 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::with_capacity` | assoc-fn | 156 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::begin` | method | 170 |  | `(NoAttributes).Begin` | done |
| `NoAttributes::end` | method | 180 |  | `(NoAttributes).End` | done |
| `NoAttributes::close` | method | 188 |  | `(NoAttributes).Close` | done |
| `NoAttributes::line_to` | method | 196 |  | `(NoAttributes).LineTo` | done |
| `NoAttributes::quadratic_bezier_to` | method | 204 |  | `(NoAttributes).QuadraticBezierTo` | done |
| `NoAttributes::cubic_bezier_to` | method | 212 |  | `(NoAttributes).CubicBezierTo` | done |
| `NoAttributes::reserve` | method | 222 |  | `(NoAttributes).Reserve` | done |
| `NoAttributes::path_event` | method | 233 |  | `(NoAttributes).PathEvent` | done |
| `NoAttributes::add_polygon` | method | 242 |  | `(NoAttributes).AddPolygon` | done |
| `NoAttributes::add_point` | method | 251 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::add_line_segment` | method | 260 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::add_ellipse` | method | 269 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::add_circle` | method | 285 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::add_rectangle` | method | 298 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::add_rounded_rectangle` | method | 307 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::flattened` | method | 317 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::transformed` | method | 328 |  | - | n/a:D-LYON-SUBSET |
| `NoAttributes::with_svg` | method | 345 |  | `(NoAttributes).WithSvg` | done |
| `NoAttributes::build` | method | 354 |  | `(NoAttributes).Build` | dev:D-LYON-TRAITDEF |
| `NoAttributes::inner` | method | 362 |  | `(NoAttributes).Inner` | done |
| `NoAttributes::inner_mut` | method | 367 |  | `(NoAttributes).Inner` | done |
| `NoAttributes::into_inner` | method | 372 |  | `(NoAttributes).IntoInner` | done |
| `PathBuilder` | trait | 451 |  | `PathBuilder` | done |
| `PathBuilder::path_event` | trait-method | 512 |  | `PathBuilderPathEvent` | dev:D-LYON-TRAITDEF |
| `PathBuilder::event` | trait-method | 534 |  | - | n/a:D-LYON-SUBSET |
| `PathBuilder::add_polygon` | trait-method | 560 |  | `PathBuilderAddPolygon` | dev:D-LYON-TRAITDEF |
| `PathBuilder::add_point` and the shape helpers through `with_svg` | trait-method | 579 |  | - | n/a:D-LYON-SUBSET |
| `SvgPathBuilder` | trait | 730 |  | `SvgPathBuilder` | dev:D-LYON-SVGRET |
| `Build` | trait | 932 |  | `Build` | done |
| `Flattened` | struct | 941 |  | - | n/a:D-LYON-SUBSET |
| `Transformed` | struct | 1055 |  | - | n/a:D-LYON-SUBSET |
| `WithSvg` | struct | 1144 |  | `WithSvg` | done |
| `WithSvg::new` | assoc-fn | 1157 |  | `NewWithSvg` | done |
| `WithSvg::build` | method | 1171 |  | `(WithSvg).Build` | dev:D-LYON-TRAITDEF |
| `WithSvg::flattened` | method | 1179 |  | - | n/a:D-LYON-SUBSET |
| `WithSvg::transformed` | method | 1183 |  | - | n/a:D-LYON-SUBSET |
| `WithSvg::move_to` | method | 1193 |  | `(WithSvg).MoveTo` | done |
| `WithSvg::line_to` | method | 1207 |  | `(WithSvg).LineTo` | done |
| `WithSvg::close` | method | 1218 |  | `(WithSvg).Close` | done |
| `WithSvg::quadratic_bezier_to` | method | 1246 |  | `(WithSvg).QuadraticBezierTo` | done |
| `WithSvg::cubic_bezier_to` | method | 1259 |  | `(WithSvg).CubicBezierTo` | done |
| `WithSvg::arc` | method | 1272 |  | `(WithSvg).Arc` | done |
| `WithSvg::current_position` | method | 1343 |  | `(WithSvg).CurrentPosition` | done |
| `WithSvg::reserve` | method | 1347 |  | `(WithSvg).Reserve` | done |
| `WithSvg::set_transform` | method | 1376 |  | - | n/a:D-LYON-SUBSET |
| `impl SvgPathBuilder for WithSvg` (`relative_move_to`) | trait-impl | 1411 |  | `(WithSvg).RelativeMoveTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_line_to`) | trait-impl | 1416 |  | `(WithSvg).RelativeLineTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_quadratic_bezier_to`) | trait-impl | 1421 |  | `(WithSvg).RelativeQuadraticBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_cubic_bezier_to`) | trait-impl | 1427 |  | `(WithSvg).RelativeCubicBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`smooth_cubic_bezier_to`) | trait-impl | 1434 |  | `(WithSvg).SmoothCubicBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`smooth_relative_cubic_bezier_to`) | trait-impl | 1439 |  | `(WithSvg).SmoothRelativeCubicBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`smooth_quadratic_bezier_to`) | trait-impl | 1446 |  | `(WithSvg).SmoothQuadraticBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`smooth_relative_quadratic_bezier_to`) | trait-impl | 1451 |  | `(WithSvg).SmoothRelativeQuadraticBezierTo` | done |
| `impl SvgPathBuilder for WithSvg` (`horizontal_line_to`) | trait-impl | 1457 |  | `(WithSvg).HorizontalLineTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_horizontal_line_to`) | trait-impl | 1462 |  | `(WithSvg).RelativeHorizontalLineTo` | done |
| `impl SvgPathBuilder for WithSvg` (`vertical_line_to`) | trait-impl | 1467 |  | `(WithSvg).VerticalLineTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_vertical_line_to`) | trait-impl | 1472 |  | `(WithSvg).RelativeVerticalLineTo` | done |
| `impl SvgPathBuilder for WithSvg` (`arc_to`) | trait-impl | 1477 |  | `(WithSvg).ArcTo` | done |
| `impl SvgPathBuilder for WithSvg` (`relative_arc_to`) | trait-impl | 1497 |  | `(WithSvg).RelativeArcTo` | done |
| `SvgPathBuilder::add_polygon` (default, used by WithSvg) | trait-method | 909 |  | `(WithSvg).AddPolygon` | done |
| `add_circle` | fn | 1508 |  | - | n/a:D-LYON-SUBSET |
| `add_rounded_rectangle` | fn | 1551 |  | - | n/a:D-LYON-SUBSET |
| `nan_check` | fn | 1665 |  | `nanCheck` | dev:D-LYON-DEBUG |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `svg_builder_line_to_after_close` | test | 1671 |  | `TestBuilder_SvgBuilderLineToAfterClose` | done |
| `svg_builder_relative_curves` | test | 1721 |  | `TestBuilder_SvgBuilderRelativeCurves` | done |
| `svg_builder_arc_to_update_position` | test | 1765 |  | `TestBuilder_SvgBuilderArcToUpdatePosition` | done |
| `issue_650` | test | 1781 |  | `TestBuilder_Issue650` | done |
| `straight_line_arc` | test | 1793 |  | `TestBuilder_StraightLineArc` | done |
| `top_right_rounded_rect` | test | 1808 |  | - | n/a:D-LYON-SUBSET |

## lyon_geom/src/lib.rs

Go target: `internal/lyon/geom/lib.go` (package `geom`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub use arrayvec` | use | 81 |  | - | n/a:D-LYON-SUBSET |
| `pub use euclid` | use | 82 |  | - | dev:D-LYON-EUCLID |
| `pub use crate::arc::{Arc, ArcFlags, SvgArc}` (`Arc`) | use | 99 |  | `Arc` | done |
| `pub use crate::arc::{Arc, ArcFlags, SvgArc}` (`ArcFlags`) | use | 99 |  | `ArcFlags` | done |
| `pub use crate::arc::{Arc, ArcFlags, SvgArc}` (`SvgArc`) | use | 99 |  | `SvgArc` | done |
| `pub use crate::cubic_bezier::CubicBezierSegment` | use | 101 |  | `CubicBezierSegment` | done |
| `pub use crate::line::{Line, LineEquation, LineSegment}` | use | 103 |  | - | n/a:D-LYON-SUBSET |
| `pub use crate::quadratic_bezier::QuadraticBezierSegment` | use | 105 |  | `QuadraticBezierSegment` | done |
| `pub use crate::segment::Segment` | use | 107 |  | - | n/a:D-LYON-SUBSET |
| `pub use crate::triangle::Triangle` | use | 109 |  | - | n/a:D-LYON-SUBSET |
| `Scalar` (f32 constants) | trait | 121 |  | `Epsilon` | dev:D-LYON-F32 |
| `Scalar::epsilon_for` (f32) | trait-method | 183 |  | `EpsilonFor` | done |
| `Point` | use | 244 |  | `Point` | done |
| `Vector` | use | 247 |  | `Vector` | done |
| `Size` | use | 250 |  | - | n/a:D-LYON-SUBSET |
| `Box2D` | use | 253 |  | - | n/a:D-LYON-SUBSET |
| `Transform` | type | 256 |  | `Transform` | done |
| `Rotation` | type | 259 |  | `Rotation` | done |
| `Translation` | type | 262 |  | - | n/a:D-LYON-SUBSET |
| `Scale` | use | 265 |  | - | n/a:D-LYON-SUBSET |
| `Angle` | use | 268 |  | `Angle` | done |
| `vector` | fn | 272 |  | `NewVector` | done |
| `point` | fn | 278 |  | `NewPoint` | done |
| `size` | fn | 284 |  | - | n/a:D-LYON-SUBSET |
| `traits::Transformation` | trait | 293 |  | `Transformation` | done |

## lyon_geom/src/arc.rs

Go target: `internal/lyon/geom/arc.go` (package `geom`), tests in `arc_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Arc` | struct | 16 |  | `Arc` | done |
| `SvgArc` | struct | 27 |  | `SvgArc` | done |
| `Arc::cast` | method | 36 |  | - | n/a:D-LYON-F32 |
| `Arc::circle` | assoc-fn | 47 |  | `ArcCircle` | done |
| `Arc::from_svg_arc` | assoc-fn | 58 |  | `ArcFromSvgArc` | done |
| `Arc::to_svg_arc` | method | 149 |  | `(Arc).ToSvgArc` | done |
| `Arc::for_each_quadratic_bezier` | method | 167 |  | `(Arc).ForEachQuadraticBezier` | done |
| `Arc::for_each_quadratic_bezier_with_t` | method | 176 |  | `(Arc).ForEachQuadraticBezierWithT` | done |
| `Arc::for_each_cubic_bezier` | method | 185 |  | `(Arc).ForEachCubicBezier` | done |
| `Arc::sample` | method | 194 |  | `(Arc).Sample` | done |
| `Arc::x` | method | 200 |  | `(Arc).X` | done |
| `Arc::y` | method | 205 |  | `(Arc).Y` | done |
| `Arc::sample_tangent` | method | 211 |  | `(Arc).SampleTangent` | done |
| `Arc::get_angle` | method | 217 |  | `(Arc).GetAngle` | done |
| `Arc::end_angle` | method | 222 |  | `(Arc).EndAngle` | done |
| `Arc::from` | method | 227 |  | `(Arc).From` | done |
| `Arc::to` | method | 232 |  | `(Arc).To` | done |
| `Arc::split_range` | method | 239 |  | - | n/a:D-LYON-SUBSET |
| `Arc::split` | method | 253 |  | - | n/a:D-LYON-SUBSET |
| `Arc::before_split` | method | 274 |  | - | n/a:D-LYON-SUBSET |
| `Arc::after_split` | method | 286 |  | - | n/a:D-LYON-SUBSET |
| `Arc::flip` | method | 298 |  | `(Arc).Flip` | done |
| `Arc::for_each_flattened` | method | 310 |  | - | n/a:D-LYON-SUBSET |
| `Arc::for_each_flattened_with_t` | method | 339 |  | - | n/a:D-LYON-SUBSET |
| `Arc::flattening_step` | method | 372 |  | - | n/a:D-LYON-SUBSET |
| `Arc::flattened` | method | 392 |  | - | n/a:D-LYON-SUBSET |
| `Arc::fast_bounding_box` | method | 397 |  | - | n/a:D-LYON-SUBSET |
| `Arc::bounding_box` | method | 405 |  | - | n/a:D-LYON-SUBSET |
| `Arc::for_each_local_x_extremum_t` | method | 424 |  | - | n/a:D-LYON-SUBSET |
| `Arc::for_each_local_y_extremum_t` | method | 436 |  | - | n/a:D-LYON-SUBSET |
| `Arc::bounding_range_x` | method | 480 |  | - | n/a:D-LYON-SUBSET |
| `Arc::bounding_range_y` | method | 485 |  | - | n/a:D-LYON-SUBSET |
| `Arc::fast_bounding_range_x` | method | 490 |  | - | n/a:D-LYON-SUBSET |
| `Arc::fast_bounding_range_y` | method | 495 |  | - | n/a:D-LYON-SUBSET |
| `Arc::approximate_length` | method | 500 |  | - | n/a:D-LYON-SUBSET |
| `Arc::tangent_at_angle` | method | 510 |  | `(Arc).tangentAtAngle` | done |
| `impl From<SvgArc> for Arc` | trait-impl | 519 |  | `ArcFromSvgArc` | done |
| `SvgArc::to_arc` | method | 527 |  | `(SvgArc).ToArc` | done |
| `SvgArc::is_straight_line` | method | 534 |  | `(SvgArc).IsStraightLine` | done |
| `SvgArc::for_each_quadratic_bezier` | method | 541 |  | `(SvgArc).ForEachQuadraticBezier` | done |
| `SvgArc::for_each_quadratic_bezier_with_t` | method | 558 |  | `(SvgArc).ForEachQuadraticBezierWithT` | done |
| `SvgArc::for_each_cubic_bezier` | method | 578 |  | `(SvgArc).ForEachCubicBezier` | done |
| `SvgArc::for_each_flattened` | method | 599 |  | - | n/a:D-LYON-SUBSET |
| `SvgArc::for_each_flattened_with_t` | method | 617 |  | - | n/a:D-LYON-SUBSET |
| `ArcFlags` | struct | 647 |  | `ArcFlags` | done |
| `arc_to_quadratic_beziers_with_t` | fn | 668 |  | `arcToQuadraticBeziersWithT` | done |
| `arc_to_cubic_beziers` | fn | 703 |  | `arcToCubicBeziers` | done |
| `sample_ellipse` | fn | 741 |  | `sampleEllipse` | done |
| `impl Segment for Arc` | trait-impl | 748 |  | - | n/a:D-LYON-SUBSET |
| `Flattened` | struct | 800 |  | - | n/a:D-LYON-SUBSET |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_from_svg_arc` | test | 835 |  | `TestArc_FromSvgArc` | dev:D-LYON-F32 |
| `test_to_quadratics_and_cubics` | test | 941 |  | `TestArc_ToQuadraticsAndCubics` | done |
| `test_bounding_box` | test | 1020 |  | - | n/a:D-LYON-SUBSET |
| `test_bounding_box_negative_sweep` | test | 1144 |  | - | n/a:D-LYON-SUBSET |
| `negative_flattening_step` | test | 1208 |  | - | n/a:D-LYON-SUBSET |

## lyon_geom/src/quadratic_bezier.rs

Go target: `internal/lyon/geom/quadratic_bezier.go` (package `geom`), tests in `quadratic_bezier_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `QuadraticBezierSegment` | struct | 19 |  | `QuadraticBezierSegment` | done |
| `QuadraticBezierSegment::cast` | method | 26 |  | - | n/a:D-LYON-F32 |
| `QuadraticBezierSegment::sample` | method | 35 |  | `(QuadraticBezierSegment).Sample` | done |
| `QuadraticBezierSegment::x` | method | 44 |  | `(QuadraticBezierSegment).X` | done |
| `QuadraticBezierSegment::y` | method | 53 |  | `(QuadraticBezierSegment).Y` | done |
| `QuadraticBezierSegment::derivative_coefficients` | method | 62 |  | `(QuadraticBezierSegment).derivativeCoefficients` | done |
| `QuadraticBezierSegment::derivative` | method | 67 |  | `(QuadraticBezierSegment).Derivative` | done |
| `QuadraticBezierSegment::dx` | method | 73 |  | `(QuadraticBezierSegment).Dx` | done |
| `QuadraticBezierSegment::dy` | method | 79 |  | `(QuadraticBezierSegment).Dy` | done |
| `QuadraticBezierSegment::flip` | method | 85 |  | `(QuadraticBezierSegment).Flip` | done |
| `QuadraticBezierSegment::y_maximum_t` | method | 96 |  | `(QuadraticBezierSegment).YMaximumT` | done |
| `QuadraticBezierSegment::y_minimum_t` | method | 114 |  | `(QuadraticBezierSegment).YMinimumT` | done |
| `QuadraticBezierSegment::local_y_extremum_t` | method | 130 |  | `(QuadraticBezierSegment).LocalYExtremumT` | done |
| `QuadraticBezierSegment::x_maximum_t` | method | 146 |  | `(QuadraticBezierSegment).XMaximumT` | done |
| `QuadraticBezierSegment::x_minimum_t` | method | 164 |  | `(QuadraticBezierSegment).XMinimumT` | done |
| `QuadraticBezierSegment::local_x_extremum_t` | method | 180 |  | `(QuadraticBezierSegment).LocalXExtremumT` | done |
| `QuadraticBezierSegment::split_range` | method | 196 |  | `(QuadraticBezierSegment).SplitRange` | done |
| `QuadraticBezierSegment::split` | method | 208 |  | `(QuadraticBezierSegment).Split` | done |
| `QuadraticBezierSegment::before_split` | method | 226 |  | `(QuadraticBezierSegment).BeforeSplit` | done |
| `QuadraticBezierSegment::after_split` | method | 235 |  | `(QuadraticBezierSegment).AfterSplit` | done |
| `QuadraticBezierSegment::to_cubic` | method | 244 |  | `(QuadraticBezierSegment).ToCubic` | done |
| `QuadraticBezierSegment::is_a_point` | method | 263 |  | `(QuadraticBezierSegment).IsAPoint` | done |
| `QuadraticBezierSegment::for_each_monotonic_range` | method | 405 |  | `(QuadraticBezierSegment).ForEachMonotonicRange` | done |
| `QuadraticBezierSegment::for_each_monotonic` | method | 440 |  | `(QuadraticBezierSegment).ForEachMonotonic` | done |
| remaining `QuadraticBezierSegment` methods (linearity, flattening, bounding boxes, intersections, length, drag) | method | 271 |  | - | n/a:D-LYON-SUBSET |
| `FlatteningParameters` | struct | 906 |  | - | n/a:D-LYON-SUBSET |
| `QuadraticBezierPolynomial` | struct | 921 |  | - | n/a:D-LYON-SUBSET |
| `Flattened` | struct | 979 |  | - | n/a:D-LYON-SUBSET |
| `FlattenedT` | struct | 1038 |  | - | n/a:D-LYON-SUBSET |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `bounding_box_for_monotonic_quadratic_bezier_segment` | test | 1101 |  | - | n/a:D-LYON-SUBSET |
| `fast_bounding_box_for_quadratic_bezier_segment` | test | 1119 |  | - | n/a:D-LYON-SUBSET |
| `minimum_bounding_box_for_quadratic_bezier_segment` | test | 1137 |  | - | n/a:D-LYON-SUBSET |
| `y_maximum_t_for_simple_segment` | test | 1155 |  | `TestQuadraticBezier_YMaximumTForSimpleSegment` | done |
| `local_y_extremum_for_simple_segment` | test | 1170 |  | `TestQuadraticBezier_LocalYExtremumForSimpleSegment` | done |
| `y_minimum_t_for_simple_segment` | test | 1186 |  | `TestQuadraticBezier_YMinimumTForSimpleSegment` | done |
| `x_maximum_t_for_simple_segment` | test | 1201 |  | `TestQuadraticBezier_XMaximumTForSimpleSegment` | done |
| `local_x_extremum_for_simple_segment` | test | 1216 |  | `TestQuadraticBezier_LocalXExtremumForSimpleSegment` | done |
| `x_minimum_t_for_simple_segment` | test | 1232 |  | `TestQuadraticBezier_XMinimumTForSimpleSegment` | done |
| `length_straight_line` | test | 1247 |  | - | n/a:D-LYON-SUBSET |
| `derivatives` | test | 1270 |  | `TestQuadraticBezier_Derivatives` | done |
| `fat_line` | test | 1283 |  | - | n/a:D-LYON-SUBSET |
| `is_linear` | test | 1302 |  | - | n/a:D-LYON-SUBSET |
| `test_flattening` | test | 1322 |  | - | n/a:D-LYON-SUBSET |
| `test_flattening_empty_curve` | test | 1382 |  | - | n/a:D-LYON-SUBSET |
| `test_flattening_straight_line` | test | 1402 |  | - | n/a:D-LYON-SUBSET |
| `test_flattened_t_values_are_monotonic` | test | 1422 |  | - | n/a:D-LYON-SUBSET |
| `issue_678` | test | 1464 |  | - | n/a:D-LYON-SUBSET |
| `line_intersections_t` | test | 1490 |  | - | n/a:D-LYON-SUBSET |
| `line_intersections_linear_case` | test | 1537 |  | - | n/a:D-LYON-SUBSET |
| `line_intersections_degenerate_curve_on_line` | test | 1557 |  | - | n/a:D-LYON-SUBSET |
| `drag` | test | 1602 |  | - | n/a:D-LYON-SUBSET |
| `arc_length` | test | 1626 |  | - | n/a:D-LYON-SUBSET |

## lyon_geom/src/cubic_bezier.rs

Go target: `internal/lyon/geom/cubic_bezier.go` (package `geom`), tests in `cubic_bezier_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `CubicBezierSegment` | struct | 23 |  | `CubicBezierSegment` | done |
| `CubicBezierSegment::sample` | method | 32 |  | `(CubicBezierSegment).Sample` | done |
| `CubicBezierSegment::x` | method | 46 |  | `(CubicBezierSegment).X` | done |
| `CubicBezierSegment::y` | method | 60 |  | `(CubicBezierSegment).Y` | done |
| `CubicBezierSegment::derivative` | method | 130 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::dx` | method | 139 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::dy` | method | 145 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::split_range` | method | 153 |  | `(CubicBezierSegment).SplitRange` | done |
| `CubicBezierSegment::split` | method | 177 |  | `(CubicBezierSegment).Split` | done |
| `CubicBezierSegment::before_split` | method | 202 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::after_split` | method | 219 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::flip` | method | 330 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::to_quadratic` | method | 344 |  | `(CubicBezierSegment).ToQuadratic` | done |
| `CubicBezierSegment::to_quadratic_error` | method | 356 |  | `(CubicBezierSegment).ToQuadraticError` | done |
| `CubicBezierSegment::num_quadratics` | method | 380 |  | `(CubicBezierSegment).NumQuadratics` | done |
| `CubicBezierSegment::num_quadratics_impl` | method | 384 |  | `(CubicBezierSegment).numQuadraticsImpl` | done |
| `CubicBezierSegment::for_each_monotonic_range` | method | 421 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::for_each_monotonic` | method | 442 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierSegment::for_each_quadratic_bezier` | method | 526 |  | `(CubicBezierSegment).ForEachQuadraticBezier` | done |
| `CubicBezierSegment::for_each_quadratic_bezier_with_t` | method | 535 |  | `(CubicBezierSegment).ForEachQuadraticBezierWithT` | done |
| remaining `CubicBezierSegment` methods (flattening, extrema, bounding boxes, intersections, length) | method | 596 |  | - | n/a:D-LYON-SUBSET |
| `CubicBezierPolynomial` | struct | 1374 |  | - | n/a:D-LYON-SUBSET |
| `Flattened` | struct | 1435 |  | - | n/a:D-LYON-SUBSET |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_iterator_builder_1` | test | 1511 |  | - | n/a:D-LYON-SUBSET |
| `test_iterator_builder_2` | test | 1530 |  | - | n/a:D-LYON-SUBSET |
| `test_iterator_builder_3` | test | 1549 |  | - | n/a:D-LYON-SUBSET |
| `test_issue_19` | test | 1568 |  | - | n/a:D-LYON-SUBSET |
| `test_issue_194` | test | 1588 |  | - | n/a:D-LYON-SUBSET |
| `flatten_with_t` | test | 1605 |  | - | n/a:D-LYON-SUBSET |
| `test_flatten_end` | test | 1645 |  | - | n/a:D-LYON-SUBSET |
| `test_flatten_point` | test | 1662 |  | - | n/a:D-LYON-SUBSET |
| `issue_652` | test | 1679 |  | - | n/a:D-LYON-SUBSET |
| `fast_bounding_box_for_cubic_bezier_segment` | test | 1699 |  | - | n/a:D-LYON-SUBSET |
| `minimum_bounding_box_for_cubic_bezier_segment` | test | 1718 |  | - | n/a:D-LYON-SUBSET |
| `y_maximum_t_for_simple_cubic_segment` | test | 1742 |  | - | n/a:D-LYON-SUBSET |
| `y_minimum_t_for_simple_cubic_segment` | test | 1758 |  | - | n/a:D-LYON-SUBSET |
| `y_extrema_for_simple_cubic_segment` | test | 1774 |  | - | n/a:D-LYON-SUBSET |
| `x_extrema_for_simple_cubic_segment` | test | 1791 |  | - | n/a:D-LYON-SUBSET |
| `x_maximum_t_for_simple_cubic_segment` | test | 1808 |  | - | n/a:D-LYON-SUBSET |
| `x_minimum_t_for_simple_cubic_segment` | test | 1823 |  | - | n/a:D-LYON-SUBSET |
| `derivatives` | test | 1839 |  | - | n/a:D-LYON-SUBSET |
| `fat_line` | test | 1853 |  | - | n/a:D-LYON-SUBSET |
| `is_linear` | test | 1903 |  | - | n/a:D-LYON-SUBSET |
| `test_monotonic` | test | 1925 |  | - | n/a:D-LYON-SUBSET |
| `test_line_segment_intersections` | test | 1941 |  | - | n/a:D-LYON-SUBSET |
| `test_parameters_for_value` | test | 1991 |  | - | n/a:D-LYON-SUBSET |
| `test_cubic_intersection_deduping` | test | 2028 |  | - | n/a:D-LYON-SUBSET |
| `cubic_line_intersection_on_endpoint` | test | 2075 |  | - | n/a:D-LYON-SUBSET |
| `test_cubic_to_quadratics` | test | 2123 |  | `TestCubicBezier_CubicToQuadratics` | dev:D-LYON-F32 |

## euclid-0.22.14 (subset)

Go target: `internal/lyon/geom/euclid.go` (package `geom`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Point2D` (`src/point.rs`) | struct | 40 |  | `Point` | dev:D-LYON-EUCLID |
| `Point2D::lerp` | method | 430 |  | `(Point).Lerp` | done |
| `Vector2D` (`src/vector.rs`) | struct | 40 |  | `Vector` | dev:D-LYON-EUCLID |
| `Vector2D::angle_from_x_axis` | method | 560 |  | `(Vector).AngleFromXAxis` | done |
| `Vector2D::lerp` | method | 613 |  | `(Vector).Lerp` | done |
| `Angle` (`src/angle.rs`) | struct | 30 |  | `Angle` | dev:D-LYON-EUCLID |
| `Angle::degrees` | assoc-fn | 84 |  | `AngleDegrees` | done |
| `Angle::positive` | method | 107 |  | `(Angle).Positive` | done |
| `Angle::signed` | method | 117 |  | `(Angle).Signed` | done |
| `Transform2D` (`src/transform2d.rs`) | struct | 70 |  | `Transform` | dev:D-LYON-EUCLID |
| `Transform2D::identity` | assoc-fn | 380 |  | `TransformIdentity` | done |
| `Transform2D::then` | method | 404 |  | `(Transform).Then` | done |
| `Transform2D::translation` | assoc-fn | 432 |  | `TransformTranslation` | done |
| `Transform2D::then_translate` | method | 446 |  | `(Transform).ThenTranslate` | done |
| `Transform2D::pre_translate` | method | 456 |  | `(Transform).PreTranslate` | done |
| `Transform2D::scale` | assoc-fn | 475 |  | `TransformScale` | done |
| `Transform2D::then_scale` | method | 491 |  | `(Transform).ThenScale` | done |
| `Transform2D::pre_scale` | method | 502 |  | `(Transform).PreScale` | done |
| `Transform2D::transform_point` | method | 522 |  | `(Transform).TransformPoint` | done |
| `Transform2D::transform_vector` | method | 532 |  | `(Transform).TransformVector` | done |
| `Transform2D::determinant` | method | 579 |  | `(Transform).Determinant` | done |
| `Transform2D::inverse` | method | 591 |  | `(Transform).Inverse` | done |
| `Transform2D::rotation` | assoc-fn | 724 |  | `TransformRotation` | done |
| `Transform2D::then_rotate` | method | 738 |  | `(Transform).ThenRotate` | done |
| `Transform2D::pre_rotate` | method | 745 |  | `(Transform).PreRotate` | done |
| `Rotation2D::new` (`src/rotation.rs`) | assoc-fn | 99 |  | `NewRotation` | done |
| `Rotation2D::transform_point` | method | 235 |  | `(Rotation).TransformPoint` | done |
| `Rotation2D::transform_vector` | method | 244 |  | `(Rotation).TransformVector` | done |
| `Trig::fast_atan2` (`src/trig.rs`) | trait-method | 40 |  | `FastAtan2` | done |
| `Trig::sin` / `cos` / `tan` (f32) | trait-method | 24 |  | `sin` | dev:D-LYON-TRIG |
| `ApproxEq` for f32 (`src/approxeq.rs`) | trait-impl | 41 |  | `ApproxEqF32` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-LYON-SUBSET | only what `gpui/src/path_builder.rs:3-9` and its callees use is ported; attribute builders, flattening, bounding boxes, intersections, reversal, polygon iterators and the unported modules are not | lyon_path 1.0.19, lyon_geom 1.0.20 | `internal/lyon`, `internal/lyon/geom` |
| D-LYON-F32 | lyon_geom's `Scalar` generics are instantiated at f32 only (lyon_path uses nothing else); `cast` is absent and the two tests written for inferred f64 (`test_from_svg_arc`, the first half of `test_cubic_to_quadratics`) run at f32 with upstream's tolerances | `lyon_geom/src/lib.rs:121`, `arc.rs:835`, `cubic_bezier.rs:2123` | `internal/lyon/geom` |
| D-LYON-TRIG | f32 `sin`, `cos`, `tan` and `powf` are computed in float64 and rounded, where Rust calls libm's `sinf` etc.; results can differ by one ulp on rare inputs (the golden test is bit-exact) | euclid `src/trig.rs:24-35`, num_traits Float | `internal/lyon/geom/euclid.go` |
| D-LYON-DEBUG | `debug_assert!` checks follow two rules: the DebugValidator's begin / end / edge / build checks are always on with upstream's messages, while `nan_check` and the NaN / tolerance debug_asserts are off, as in a release build, because gpui feeds non-finite points through the builder and rejects them later (`path_builder.rs:418-484`) | `lyon_path/src/private.rs:13-55`, `path.rs:810`, `builder.rs:1665`, `lyon_geom/src/arc.rs:59-65` | `internal/lyon/private.go`, `internal/lyon/path.go` |
| D-LYON-EUCLID | the euclid types lyon uses become plain f32 structs `Point`, `Vector`, `Angle`, `Transform`, `Rotation`, `Range` in `geom/euclid.go`, with operators as methods (`Add`, `Sub`, `SubVector`, `Mul`, `Div`) and only the methods the subset needs; lyon_path re-exports them as aliases | euclid 0.22.14 | `internal/lyon/geom/euclid.go`, `internal/lyon/lib.go` |
| D-LYON-EVENT | `Event<Ep, Cp>` is a comparable tagged struct (`Kind` plus the variant fields `At`, `From`, `Ctrl`, `Ctrl1`, `Ctrl2`, `To`, `Last`, `First`, `Close`; unused fields are zero); its `from()` / `to()` methods are `FromEndpoint` / `ToEndpoint` because the fields take the names | `lyon_path/src/events.rs:8-75` | `internal/lyon/events.go` |
| D-LYON-SVGRET | `SvgPathBuilder`'s `move_to`, `line_to`, `quadratic_bezier_to` and `cubic_bezier_to` return `EndpointID` in Go, because `WithSvg`'s inherent methods of the same names do and Go allows one method per name | `lyon_path/src/builder.rs:738-777`, `:1193-1259` | `internal/lyon/builder.go` |
| D-LYON-TRAITDEF | trait default methods become free functions over the interface (`PathBuilderPathEvent`, `PathBuilderAddPolygon`); `Build` bounds are checked by a type assertion in `NoAttributes.Build` / `WithSvg.Build` instead of a where clause | `lyon_path/src/builder.rs:354`, `:512-575`, `:1171` | `internal/lyon/builder.go` |
| D-LYON-NOATTR | `NO_ATTRIBUTES` is a nil `Attributes` slice | `lyon_path/src/lib.rs:463` | `internal/lyon/lib.go` |
