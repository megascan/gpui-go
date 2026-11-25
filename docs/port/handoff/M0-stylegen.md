# M0-stylegen handoff (to M0-style, M0-geometry-units)

`go run ./internal/cmd/stylegen [-out styled_gen.go] [-pkg gpui] [-macros a,b]` writes all 3099 methods (order = styled.rs:26-34 macro order, then upstream table order). The generated code needs these root symbols; the exact contract is type-checked in `internal/cmd/stylegen/testdata/contract/contract.go`:

- `Styled[S Styler]` with unexported field `self S`; `Styler.Style() *StyleRefinement`.
- StyleRefinement fields (Option = gpuiutil.Option, set with `gpuiutil.Some`): `Visibility`, `Position`, `MouseCursor`, `BorderColor`, `BoxShadow Option[[]BoxShadow]`, `Overflow.X/.Y`, `Inset/Margin.Top/Right/Bottom/Left` (Length), `Padding.*` (DefiniteLength), `BorderWidths.*` (AbsoluteLength), `Size/MinSize/MaxSize.Width/Height` (Length), `Gap.Width/Height` (DefiniteLength), `CornerRadii.TopLeft/TopRight/BottomRight/BottomLeft` (AbsoluteLength).
- `Px(float32) Pixels`, `Rems` conversion, `Relative(float32) DefiniteLength`, `Auto() Length`, `DefiniteLength.Neg()`; negated px/rems use the unary minus operator.
- Sealed into-interfaces: `IntoLength{ToLength() Length}`, `IntoDefiniteLength{ToDefiniteLength() DefiniteLength}`, `IntoAbsoluteLength{ToAbsoluteLength() AbsoluteLength}`, `IntoHsla{ToHsla() Hsla}`; implemented by Pixels, Rems (all three), DefiniteLength (Length, DefiniteLength), Length (Length), AbsoluteLength as upstream's From impls.
- `NewHsla`, `NewBoxShadow(x, y Pixels, color Hsla)`, `(BoxShadow).BlurRadius`, `(BoxShadow).SpreadRadius` (value receivers returning BoxShadow).
- Enum constants: `VisibilityVisible/Hidden`, `PositionRelative/Absolute`, `OverflowHidden`, `CursorStyle<Variant>` for the upstream variants.

Choices: `shadow_none` sets `Some([]BoxShadow(nil))` (Default::default() = zero value); the custom setter calls `length.ToX()` once per field (upstream's clone().into()); godoc is upstream's doc string with "Sets"/"Set" turned into "<Method> sets".
