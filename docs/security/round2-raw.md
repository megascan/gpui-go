# Round 2 findings, against the round-1 fixes

Source key:
- **B**: breaker. Its tests are under the harness `r2_*` files, and every one currently FAILs.
- **R**: reviewer, from reading the code only.

## platform.go (decoders)
- **B R2-1:** animated WebP frames get past the frame budget. `decodeWebpFramePixels` decodes before it checks the inner stream's real dimensions. Test: `TestR2_S01_01`/`01b`.
- **B R2-5:** the TIFF strip buffer is sized at `w*h*8`, so memory far exceeds the reservation. Test: `TestR2_S01_06`.
- **B R2-6:** the budget applies to each decode, not to the process, so concurrent decodes together run out of memory. Test: `TestR2_S01_05`.
- **B BR-2 (behaviour regression):** the budget refuses legitimate images: a 640x480 GIF with 900 frames, a 1080p GIF with 150 frames, and a 61MP progressive JPEG. Tests: `TestR2_S01_02`/`03`/`04`.
- **INFO:** the budget estimate is about 1.5x too low for CMYK JPEG and lossless WebP.

## svg_renderer.go + elements_svg.go
- **B R2-7:** nested SVG `<image>` elements multiply canvas pixmaps, giving OOM and timeouts. Tests: `TestR2_S02_03`/`04`.
- **B R2-8:** a textPath on a path with coordinate 1e38 hangs. Test: `TestR2_S02_08`.
- **B BR-1 (behaviour regression):** the filter limit is too tight. A drop-shadow icon at 1024px and a single blur on a 1080p image are both refused. Tests: `TestR2_S02_05`/`06`.
- **R R2-04:** `Svg.Data` builds its atlas key from unkeyed SipHash (`elements_svg.go:65`), and `ExternalPath("__binary_svg__N")` shares the same key space.
- **R R2-05:** `svgAsset.Load` calls `os.ReadFile` with no regular-file check and no cap (`elements_svg.go:464`).

## asset_cache.go
- **B R2-2:** a background panic turns into a nil success because the `task.Await` error is dropped (`asset_cache.go:40-45`). The UI then dereferences nil (`elements_img.go:347`). Tests: `TestR2_S15_01`/`02`.
- **B/R R2-3:** the LRU livelocks or thrashes once the visible set exceeds its budget. Evicting a pending entry cancels it and notifies the view, and the background load is not cancelled. Test: `TestR2_S15_05`.
- **R I-1:** `trim` stops early when the back element is the one being kept.

## Linux IPC / open
- **B/R R2-9/R2-01:**
  - The portal Response waiters have no sender check: screencap `portal.go:282/292` and common `portalCall` at `:388/430`.
  - The screencap token is a uint16.
  - Junk signals count toward a 100-iteration limit.
  - Test: `s07/r2_screencap.sh`.
- **R R2-02:** the SettingChanged (`xdg_desktop_portal.go:190`), PrepareForSleep (`common.go:335`) and power (`power.go:120`) listeners do not check the sender.
- **B R2-10:** OpenURL `mailto:` keeps `attach=`/`attachment=` parameters. Test: `s04/r2_run.sh`.
- **B R2-11:** OpenWithSystem bypasses the scheme allowlist (`file:`, `smb:`, `javascript:`, custom schemes). Test: `s04/r2_run.sh`.
- **B R2-4:** the keymap memfd can shrink between fstat and the read, which raises SIGBUS (`xkb.go:182-193`). Test: `s09b/r2_run.sh`.

## Misc
- **R R2-06:** `examples/web_hello_web/dynamic_fonts.go:105` calls `io.ReadAll` with no bound.
- **B R2-12:** `examples/image_gallery/main.go:205` uses `gpui.Hash` as a cache key.
- **B R2-13:** Scoped drops its child task errors. SpawnRealtime, SpawnDedicated and DedicatedExecutor are not wrapped by `recoverBackgroundPanic`.
- **INFO:** the Windows clipboard image decode has no budget (`platform/windows/clipboard.go:414`).
