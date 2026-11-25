# Debt: M0-scene

- path_winding.go: upstream's `thread_local!` BINNER scratch is a `sync.Pool` of `*pathBinner`; same reuse, but a binner can move between goroutines and the pool may drop it under GC pressure.
- path_winding.go: `listed` is an `iter.Seq` closure per cell; upstream's iterator is allocation-free. Profile once the wgpu renderer draws real paths.
- scene.go: `Primitive` values are interfaces, so every `InsertPrimitive` boxes the primitive (one allocation); upstream stores the enum inline. Revisit with the window paint benches.
- path_builder.go: strokes use xo/okage/path, a port of tiny-skia-path 0.12.0, while upstream locks 0.11.4. Diffed stroker.rs, dash.rs, path.rs, path_builder.rs, path_geometry.rs and scalar.rs between the two: only lifetimes, derives and one unused helper differ, no behaviour.
- path_builder.go: `Transform`'s constructors (`TransformTranslation`, `TransformRotation`, ...) and `Angle` live in internal/lyon/geom, so code outside the module can build a `Transform` only as a composite literal (`gpui.Transform{M11: ...}`) or through `PathBuilder.Translate` / `Scale` / `Rotate`. Upstream users call euclid directly.
- scene.go: `TransformationMatrix.Rotate` uses `math.Cos` / `math.Sin` in float64, rounded to float32; Rust's f32 `cos` / `sin` may differ by one ulp.
