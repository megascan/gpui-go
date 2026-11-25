# Debt: M0-geometry

- Debug output (`GoString`) and Display (`String`) format numbers with Go `%v`: `5` where Rust Debug prints `5.0` for f32 fields. Pixels gets its own String in M0-geometry-units.
- `SharedString.GoString` approximates Rust `str::escape_debug`: non-printable runes become `\u{..}` via `unicode.IsPrint`; Rust also escapes grapheme extenders at string start, Go does not.
- Operator impls with a foreign right-hand type (`Point<T>: Mul<Rhs>`, `Size<T> * Rhs -> Size<Rhs>`, `Bounds<T> * Rhs -> Bounds<Rhs>`, `Div<S>`, `MulAssign<S>`) take `rhs T` (same unit as the container). Callers convert the scalar (`p.Mul(Pixels(f))`), which is exact for Pixels/ScaledPixels * f32 and DevicePixels * i32.
- Generic comparisons go through `geomPartialCmp`, which type-switches on Pixels (partial_cmp, then total_cmp for NaN) and ScaledPixels (total_cmp); every other T uses IEEE / integer ordering, as Rust's PartialOrd does. Rems has no PartialOrd upstream; Go orders it natively.
- `stubs.go` holds `App.FindDisplay`, `App.PrimaryDisplay`, `DisplayID` and a three-method `PlatformDisplay` (ID, Bounds, VisibleBounds) for BoundsCentered / BoundsMaximized; M0-app-core and M0-platform-types replace them. BoundsCentered / BoundsMaximized are not exercised by tests until then.
