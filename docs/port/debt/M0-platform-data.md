# M0-platform-data debt

- IMG-GIF-FRAMES: `image/gif` decodes all frames or fails; the image crate skips a bad frame and keeps the rest (`decode_animated_frames`). One corrupt GIF frame fails the whole image in Go (D-IMGDECODE).
- IMG-BMP-SUBSET: BMP files decode through x/image/bmp (1/4/8/24/32-bit uncompressed, BI_BITFIELDS only with default masks); the image crate also reads 16-bit, RLE4/RLE8 and arbitrary bitfields. ICO BMP entries use the in-repo DIB reader with the same subset (D-IMGDECODE).
- IMG-WEBP-LOSSY: lossy WebP pixels come from x/image's VP8 decoder and `color.YCbCrToRGB`, not image-webp's upsampling; values can differ slightly (like L-JPEG).
- IMG-ICO-PNG16: an ICO PNG entry is accepted only when `png.DecodeConfig` reports `color.NRGBAModel` (8-bit RGBA), matching the image crate's `PngNotRgba` check.
