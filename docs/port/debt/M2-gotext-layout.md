# M2-gotext-layout debt

- Bidi levels come from go-text `bidi.Paragraph`, whose `Runs` split on direction parity only, so nested levels of equal parity (for example 2 and 4 from explicit embeddings) collapse into one span before `reorder`. The paragraph level is read from a trailing space appended before segmentation (L1 resets it to the paragraph level). Fix: expose per-rune levels (port unicode-bidi's level resolution) if embeddings matter.
- cosmic-text's `Family::Monospace` fallback branch (monospace fallback buffer, per-script monospace ids) is not ported: gpui only sends `Family::Name`.
- `FontMatchKey.variable_weight_match` is always false and `get_font` ignores the requested weight (no `wght` variation applied): go-text keeps `fvar` unexported.
- cosmic-text's `shape-run-cache` is not ported; one reused harfbuzz buffer carries the shape-plan cache. BenchmarkLayoutLine_* runs at about 8.5 ms per 3.8k-char line on an i9-13900H, mostly go-text GSUB application (Lilex `calt`); upstream numbers not measured.
- The harfbuzz buffer language is cleared after `GuessSegmentProperties` to match harfrust's unset language; go-text would otherwise take it from the process locale.
- System fonts are found with `fontscan.DefaultFontDirectories` plus a recursive walk (ttf/ttc/otf/otc), not fontdb's fontconfig parser; face metadata is read eagerly, font bytes lazily on first use.
- `systemLocale` reads LC_ALL / LC_CTYPE / LANG like sys-locale on Unix; other platforms fall back to en-US.
- Glyph rasterisation (`GlyphRasterBounds`, `RasterizeGlyph`) returns an error until M2-gotext-raster.
- Test fonts (IBM Plex Sans Regular, Lilex Regular/Bold, OFL) are copies of zed@6501df1 `assets/fonts` in `text/gotext/testdata/fonts`.
- `GOOS=ios CGO_ENABLED=0 go vet ./text/gotext` fails with "requires external (cgo) linking", the same as for every package (e.g. ./gpuiutil); not checked for ios.
- Benchmarks use `testing.B.Loop` directly: `gpui.RunBench` does not exist yet (only mentioned in seeds.go).
