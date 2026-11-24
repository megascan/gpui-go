# M2-exit handoff

- To M3-exit (owed by M1-exit, not done here): a display-gated check that WaylandConnection.WindowIdentifier returns a `wayland:`-prefixed id for a mapped, keyboard-focused toplevel. It needs keyboard focus on a real compositor, which shoot cannot guarantee while another window is fullscreen on the workspace; no test was added.
- Resolved from M2-gotext-layout: TextSystem.resolveFontUncached now serialises cold lookups on fontIDsUpgradeMu and takes fontIDsByFontMu exclusively only to insert (upstream's upgradable_read), AddFonts takes both; the env gate in text/gotext/text_system_test.go is gone and the gotext row is done. TestCosmicTextSystem_ReportsGraphemesThatExhaustFontFallback is ported (TestAppContextBuildWithTextSystem + App.OnMissingGlyphs) and passes.
- paritycheck -lint still fails on files outside M2-exit: internal/linux/wl/display.go:285 (needs the lint allowlist, see M1-exit handoff), a11y/windows tests and platform/macos (M4-ci-windows / M4-ci-macos), platform/ios test and examples/ios_ios/export.go cgo (M4-ci-macos).
- A stray ELF binary `hello_world` sits in the worktree root (someone ran go build without -o); it is untracked, not created by M2-exit, and must not be committed.
- Resolution (2026-11-24, M3-exit): WindowIdentifier display check not added (TECH_DEBT M3-exit).
