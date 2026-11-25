# M2-exit debt

- examples/svg_example and examples/image_example find their asset directory with runtime.Caller (the source tree, upstream's CARGO_MANIFEST_DIR), so a copied binary only finds its assets next to the checkout. Same limitation as upstream's env!("CARGO_MANIFEST_DIR").
- examples/internal/support.HTTPClient is the stand-in for reqwest_client (D-HTTP): net/http on a goroutine, body read into memory before the task resolves, no proxy support. Replace if a shared gpui HTTP client package lands.
- text example on this machine: 😀, 😍, ❤️ and ❤︎ render monochrome from DejaVu Sans and the keycap from Noto Sans Symbols. Verified parity: cosmic-text 0.19.0 run offline on this machine with the same Lilex primary font picks the same faces.
- internal/cmd/shoot with -size floats the window where Hyprland puts it, sometimes partly off-monitor (y < 0 or past the bottom), so tall captures can be cut at the monitor edge. The recorded m2 screenshots use the tiled size (no -size).
- The image example's EXIF image lays out 256x512 instead of 256x256: taffy 0.15 block layout applies aspect_ratio over a definite height there. Checked against the Rust taffy 0.15.0 crate: identical result, so this is upstream behaviour, not a port bug.
