# M0-elements-media debt

- `ImageAssetLoader` recognises all 22 magic-byte signatures of image 0.25.10's `guess_format`, but decodes only formats that `newImageDecoder` (platform.go) supports: PNG, JPEG, GIF, WebP, BMP, TIFF, ICO and PNM. AVIF, DDS, HDR, OpenEXR, QOI and farbfeld fail with "The image format X is not supported". `ImgExtensions` still lists them, as upstream does. gen2brain/avif is a pinned dependency, but nothing wires it in yet.
- `isURI` approximates `url::Url::from_str`. It accepts a WHATWG scheme (`[A-Za-z][A-Za-z0-9+.-]*:`), and for http, https, ws, wss and ftp it also needs `net/url` to parse a host. Inputs where the url crate's full host and path validation disagree with this can end up on the other branch (URI vs embedded).
- `RetainAllImageCacheProvider` stores the cache `Entity` in window element state. The element-state map (window.go `elementStateMap.clear`) never releases dropped states, so under strict leak detection a `RetainAll` cache outlives its element (handoff).
- `AnyImageCache` is weak (D-WEAKIMAGECACHE). An `Img`/`ImageCacheElement` whose cache entity was released panics on load, where upstream kept the entity alive.
- `Svg.Data` hashes with SipHash-1-3, keys 0, over a length prefix plus the bytes. This is std's `DefaultHasher` today. Rust does not promise that this algorithm stays fixed.
- `TimedTransformation.WithMaxFps` truncates `1/max_fps` seconds to nanoseconds. Rust's `Duration::from_secs_f64` may round the last nanosecond.
- The SVG system font database walks the same folders as okage's `LoadOSFonts` one file at a time, so it can record each face's file (needed by `owned_face`). If okage changes its folder list, this copy has to change too.
