//go:build js && wasm

package web

import (
	"github.com/megascan/gpui-go"
)

// CanvasFontFallback controls browser-font fallback when loaded fonts lack a glyph or its emoji
// presentation.
type CanvasFontFallback uint8

const (
	// CanvasFontFallbackDisabled uses only fonts loaded into GPUI.
	CanvasFontFallbackDisabled CanvasFontFallback = iota
	// CanvasFontFallbackEmoji uses Canvas for eligible emoji; it is the default.
	CanvasFontFallbackEmoji
	// CanvasFontFallbackEmojiAndCjk also allows approximate rendering of eligible horizontal CJK text.
	CanvasFontFallbackEmojiAndCjk
)

// DefaultCanvasFontFallback returns CanvasFontFallbackEmoji.
func DefaultCanvasFontFallback() CanvasFontFallback {
	return CanvasFontFallbackEmoji
}

func newWebTextSystem(fallbackFontFamily string, canvasFontFallback CanvasFontFallback) gpui.PlatformTextSystem {
	panic("gpui: unimplemented newWebTextSystem")
}
