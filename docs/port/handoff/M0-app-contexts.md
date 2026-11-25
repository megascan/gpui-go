# M0-app-contexts handoff

- docs/port/parity/macos.md `test_read_image` is `blocked:M0-app-contexts`; this stage is done, the macOS owner can port it (Hash and the async contexts exist).
- `paritycheck -lint` fails on files outside this stage: internal/linux/wl/display.go:285 (LockOSThread outside package platform), internal/linux/xcbshim/setup.go:5 (comment).
- M0-test-contexts: implement the stubs TestAppContextBuild, (TestAppContext).Update, RunUntilParked and drop (the per-context teardown RunTest calls; drop does the GC loop and leak report of DESIGN 6.11).
