# M2-gotext-raster debt

- No TrueType/CFF hinting: swash renders with `.hint(true)` (skrifa hinting instance); go-text has no hinter, so outlines are drawn unhinted (L-GLYPH). Glyph pixels and occasionally raster bounds differ by up to a pixel from upstream.
- `Source::Bitmap(StrikeWith::ExactSize)` (non-emoji bitmap strikes) and the colour sources are skipped in `swashRender.renderInto`; M2-gotext-color owns them (text/gotext/scale.go).
- Outline extraction is go-text's `Face.GlyphDataOutline` (implicit TrueType on-curve points become explicit quad endpoints) instead of skrifa's pen; coverage matches only where both emit the same curves (L-GLYPH).
