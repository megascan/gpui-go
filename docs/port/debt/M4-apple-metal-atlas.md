# Debt: M4-apple-metal-atlas

- metalhdr feeds cbindgen one concatenated file (the six build.rs sources with `#![...]` / `//!` lines stripped) instead of Builder::with_src per file; output is believed identical, not diffed against a real build.rs run on macOS.
- platform/apple is `darwin`-only, so `go generate ./...` on Linux skips the metalhdr directive; regenerate with `GOOS=darwin go generate ./platform/apple` (cbindgen 0.28.0 from PATH or ~/.cargo/bin). TestMetalhdr_GeneratedFileIsCurrent catches staleness on Linux.
- mtl wrappers are raw objc.ID value types with explicit mtl.Retain / mtl.Release, not objc.Retained handles; ownership (+1 for new*/Create/Copy, +0 autoreleased otherwise) is documented per method.
- internal/apple/mtl, metal_atlas and shader tests only run on the macOS CI runner; locally only cross vet / test -c.
- GOOS=ios CGO_ENABLED=0 go vet fails before analysis with "ios/arm64 requires external (cgo) linking" (toolchain, not this code); the STAGES ios Verify form hits the same wall.
