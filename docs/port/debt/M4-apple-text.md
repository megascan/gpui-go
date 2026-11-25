# M4-apple-text debt

- font-kit (zed-font-kit 0.14.1) has no Go equivalent: the slice gpui_apple uses (CoreText loader Font, MemSource, CoreTextSource.select_family_by_name, matching::find_best_match, weight/stretch mapping, OTC and data-fork unpacking) is ported as unexported helpers in platform/apple/font_kit.go, with font-kit's own core_text weight/stretch tests plus a find_best_match test.
- AddFonts takes [][]byte, so upstream's Cow::Borrowed path (CGDataProvider over a static slice, CGFont, CTFont) does not exist; every font goes through Handle::from_memory (CTFontManagerCreateFontDescriptorFromData), as Cow::Owned does upstream.
- CTRun glyphs, positions and string indices are always copied with CTRunGet{Glyphs,Positions,StringIndices}; the *Ptr fast paths are not bound (same values).
- The CoreFoundation types package cf lacks (CFNumber, CFMutableArray, CFMutableAttributedString, CFLocale, CFPreferences, CFURL path) are bound in internal/apple/ct/cfx.go; see handoff.
- internal/apple/cg binds only bitmap contexts, colour spaces and CGFont retain/release; CGPath is not bound because no ported text-system code uses it (font-kit's outline() is not used by gpui_apple).
- None of this has run on a Mac: locally only darwin/arm64 and darwin/amd64 vet + go test -c pass; M4-ci-macos is the first execution of every test here (including Lilex-based ligature test and glyph ids 68/69 for Helvetica a/b).
