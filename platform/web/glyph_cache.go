//go:build js && wasm

package web

import (
	"errors"
	"math"
	"strings"

	"github.com/megascan/gpui-go"
)

type canvasGlyph struct {
	fontID gpui.FontID
	text   string
	color  bool
}

type glyphCacheKey struct {
	fontID gpui.FontID
	color  bool
}

type glyphCache struct {
	glyphs    []canvasGlyph
	idsByFont map[glyphCacheKey]map[string]gpui.GlyphID
}

func (c *glyphCache) idFor(fontID gpui.FontID, text string, color bool) (gpui.GlyphID, bool) {
	ids, ok := c.idsByFont[glyphCacheKey{fontID, color}]
	if !ok {
		return 0, false
	}
	id, ok := ids[text]
	return id, ok
}

func (c *glyphCache) getOrInsert(fontID gpui.FontID, text string, color bool) (gpui.GlyphID, error) {
	if glyphID, ok := c.idFor(fontID, text, color); ok {
		return glyphID, nil
	}
	if uint64(len(c.glyphs)) > math.MaxUint32 {
		return 0, errors.New("out of range integral type conversion attempted")
	}
	glyphID := gpui.GlyphID(len(c.glyphs))
	owned := strings.Clone(text)
	c.glyphs = append(c.glyphs, canvasGlyph{fontID: fontID, text: owned, color: color})
	if c.idsByFont == nil {
		c.idsByFont = make(map[glyphCacheKey]map[string]gpui.GlyphID)
	}
	key := glyphCacheKey{fontID, color}
	ids, ok := c.idsByFont[key]
	if !ok {
		ids = make(map[string]gpui.GlyphID)
		c.idsByFont[key] = ids
	}
	ids[owned] = glyphID
	return glyphID, nil
}

func (c *glyphCache) glyph(fontID gpui.FontID, glyphID gpui.GlyphID) (*canvasGlyph, bool) {
	if uint64(glyphID) >= uint64(len(c.glyphs)) {
		return nil, false
	}
	glyph := &c.glyphs[glyphID]
	if glyph.fontID != fontID {
		return nil, false
	}
	return glyph, true
}
