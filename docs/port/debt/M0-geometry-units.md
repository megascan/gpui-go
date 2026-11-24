# Debt: M0-geometry-units

- `(Pixels).Pow` goes through float64 math.Pow; Rust powf on f32 may differ by 1 ulp.
- JSONSchema for AbsoluteLength, DefiniteLength, Length, Rgba and Hsla is `{"type":"string"}` without upstream's `pattern`: internal/jsonschema.Schema has no Pattern field (handoff written).
- Pixels/ScaledPixels `Sum` impls are not ported (Go loops add natively); `Hash` impls rely on Go `==` map keys (no canonical_float_bits, so +0/-0 hash equal as in upstream, NaN keys differ).
- `IsZero` methods on Point/Size/Bounds/units are also picked up by encoding/json/v2 `omitzero`; Size.IsZero is true when either side is zero, so a bare `omitzero` Size field would be dropped then. Refinements wrap fields in Option, so generated code is unaffected.
- Parse error texts follow Rust's ParseFloatError / ParseIntError wording; anyhow context chains collapse to the outermost message except ParseDefiniteLength's percent branch, which appends the cause with %w.
- Float32 maths in color.go wraps products in float32() before adds to keep the compiler from fusing them into FMA on arm64.
