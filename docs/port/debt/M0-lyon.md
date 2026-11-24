# M0-lyon debt

- internal/lyon/geom: every float product that feeds a sum is wrapped in `float32(...)` so Go does not fuse it into an FMA (Go may fuse on arm64, ppc64, s390x, riscv64 and amd64 v3; Rust never does). New arithmetic in this package must follow the same rule, or arm64 output stops matching upstream. `TestParity_RustGoldenEvents` (internal/lyon/parity_test.go) catches drift; its golden text was produced by lyon_path 1.0.19 / lyon_geom 1.0.20 with rustc 1.99 on x86_64.
- f32 trig goes through float64 (D-LYON-TRIG); a one-ulp difference from libm `sinf` is possible on inputs the golden test does not cover.
- `Path` shares its point and verb slices with the builder that produced it (Rust moves the builder). Reusing a builder after `Build` would alias; `Transformed` copies before mutating.
