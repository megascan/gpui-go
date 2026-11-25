# M0-platform-data handoff

## Shared contract: Range

- `Range{Start, End int}` (platform.go, D-RANGE) is the root package's `std::ops::Range<usize>`. Reuse it for every `Range<usize>` (input.rs, text system, IME backends); `&mut Option<Range>` is `*Option[Range]`. M0-contracts / an exit stage: add it to docs/api-contracts.md.

## Stubs added to stubs.go (final signatures; the owning stage replaces them)

- M0-window-core: `FocusID uint64`; `Window` stub now has the field `focus Option[FocusID]` (upstream `window.focus`), read by `PlatformInputHandler.QueryAcceptsFocusedTextInput`. Keep the field name and type.
- M0-window-dispatch: `(*Window).HasPendingKeystrokes() bool`.
- M0-app-contexts: `AsyncWindowContext` with `Update(func(*Window, *App)) error` and `UpdateR[R](func(*Window, *App) R) (R, error)`; `PlatformInputHandler` holds an `AsyncWindowContext` by value. `Hash[T any](data T) uint64` (asset_cache.rs `hash`), used by `ImageFromBytes`.
- M0-app-core / M0-app-effects: `(*App).ToAsync() *AsyncApp`, `(*App).update(func(*App))`, `(*App).clearPendingKeystrokes()`, `(*App).IsActionAvailable(Action) bool`, `(*App).DispatchAction(Action)`, `(*AsyncApp).upgradeApp() (*AppCell, bool)` (upstream `cx.app.upgrade()`), used by `initAppMenus` in platform_app_menu.go. The app stage must call `initAppMenus(platform, cx)` where upstream calls `init_app_menus`.
- M0-input-types: `NoAction` (methods panic; the menu tests only construct it), `ExternalPaths []string` (the newtype over `SmallVec<[PathBuf; 2]>`; `ClipboardItem.Text` ranges over it), `AssetSource` interface, `RenderImage` with `AsBytes(int) ([]byte, bool)`, `Size(int) Size[DevicePixels]`, `Delay(int) time.Duration`, `FrameCount() int`, and `NewRenderImage([]ImageFrame) *RenderImage`. `RenderImage` holds `[]ImageFrame` (platform.go, D-IMAGEFRAME): `Buffer` is BGRA bytes in an `*image.RGBA`, `Delay` a `time.Duration`; `delay()`'s default `Delay::from_numer_denom_ms(100, 1)` is `100 * time.Millisecond`.
- M0-elements-media: `SvgRenderer` (value type) with `RenderSingleFrame([]byte, float32) (*RenderImage, error)` and `NewSvgRenderer(AssetSource) SvgRenderer`; the ported image tests call `NewSvgRenderer(nil)` for upstream's `SvgRenderer::new(Arc::new(()))`, so a nil asset source must behave as the unit source (no assets). `ImageSource` sealed interface with `ImageSourceImage{Image *Image}`, `AnyImageCache`, and the free functions `imageSourceUseData` / `imageSourceGetData` `(source, cache Option[AnyImageCache], window, cx) (*RenderImage, bool, error)` (ok false = upstream `None`), `imageSourceRemoveAsset(source, cx)`, `imageSourceIsAssetCached(source, cx) bool`, called by the `Image` methods.

## Blocked tests

- The three `image_tests` (core-platform.md, `blocked:M0-elements-media`) are written in platform_test.go and recover a `gpui: unimplemented ...` stub panic into `t.Skip`, so they start running by themselves once Hash, RenderImage and SvgRenderer are real. The stage or exit that finds them passing flips the rows to `done`; no test edit is needed.
- `TestPlatformImageDecode_*` in platform_test.go already exercise the decoders on the upstream fixtures (EXIF-rotated JPEG, animated WebP) without the stubs.

## Window icon

- `WindowOptions.Icon` / `WindowParams.Icon` are `Option[*image.NRGBA]` (straight alpha, as `image::RgbaImage`), not `*image.RGBA`.
