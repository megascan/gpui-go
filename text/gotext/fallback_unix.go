//go:build (unix && !darwin && !android) || ios

package gotext

import "github.com/go-text/typesetting/language"

var platformCommonFallback = []string{"Noto Sans", "DejaVu Sans", "FreeSans", "Noto Sans Mono", "DejaVu Sans Mono", "FreeMono", "Noto Sans Symbols", "Noto Sans Symbols2", "Noto Color Emoji"}

var platformForbiddenFallback = []string{}

func hanUnification(locale string) []string {
	switch locale {
	case "ja":
		return []string{"Noto Sans CJK JP"}
	case "ko":
		return []string{"Noto Sans CJK KR"}
	case "zh-HK":
		return []string{"Noto Sans CJK HK"}
	case "zh-TW":
		return []string{"Noto Sans CJK TC"}
	default:
		return []string{"Noto Sans CJK SC"}
	}
}

func scriptFallback(script language.Script, locale string) []string {
	switch script {
	case language.Adlam:
		return []string{"Noto Sans Adlam", "Noto Sans Adlam Unjoined"}
	case language.Arabic:
		return []string{"Noto Sans Arabic"}
	case language.Armenian:
		return []string{"Noto Sans Armenian"}
	case language.Bengali:
		return []string{"Noto Sans Bengali"}
	case language.Bopomofo:
		return hanUnification(locale)
	case language.Braille:
		return []string{"FreeMono"}
	case language.Buhid:
		return []string{"Noto Sans Buhid"}
	case language.Chakma:
		return []string{"Noto Sans Chakma"}
	case language.Cherokee:
		return []string{"Noto Sans Cherokee"}
	case language.Deseret:
		return []string{"Noto Sans Deseret"}
	case language.Devanagari:
		return []string{"Noto Sans Devanagari"}
	case language.Ethiopic:
		return []string{"Noto Sans Ethiopic"}
	case language.Georgian:
		return []string{"Noto Sans Georgian"}
	case language.Gothic:
		return []string{"Noto Sans Gothic"}
	case language.Grantha:
		return []string{"Noto Sans Grantha"}
	case language.Gujarati:
		return []string{"Noto Sans Gujarati"}
	case language.Gurmukhi:
		return []string{"Noto Sans Gurmukhi"}
	case language.Han:
		return hanUnification(locale)
	case language.Hangul:
		return hanUnification("ko")
	case language.Hanunoo:
		return []string{"Noto Sans Hanunoo"}
	case language.Hebrew:
		return []string{"Noto Sans Hebrew"}
	case language.Hiragana:
		return hanUnification("ja")
	case language.Javanese:
		return []string{"Noto Sans Javanese"}
	case language.Kannada:
		return []string{"Noto Sans Kannada"}
	case language.Katakana:
		return hanUnification("ja")
	case language.Khmer:
		return []string{"Noto Sans Khmer"}
	case language.Lao:
		return []string{"Noto Sans Lao"}
	case language.Malayalam:
		return []string{"Noto Sans Malayalam"}
	case language.Mongolian:
		return []string{"Noto Sans Mongolian"}
	case language.Myanmar:
		return []string{"Noto Sans Myanmar"}
	case language.Oriya:
		return []string{"Noto Sans Oriya"}
	case language.Runic:
		return []string{"Noto Sans Runic"}
	case language.Sinhala:
		return []string{"Noto Sans Sinhala"}
	case language.Syriac:
		return []string{"Noto Sans Syriac"}
	case language.Tagalog:
		return []string{"Noto Sans Tagalog"}
	case language.Tagbanwa:
		return []string{"Noto Sans Tagbanwa"}
	case language.Tai_Le:
		return []string{"Noto Sans Tai Le"}
	case language.Tai_Tham:
		return []string{"Noto Sans Tai Tham"}
	case language.Tai_Viet:
		return []string{"Noto Sans Tai Viet"}
	case language.Tamil:
		return []string{"Noto Sans Tamil"}
	case language.Telugu:
		return []string{"Noto Sans Telugu"}
	case language.Thaana:
		return []string{"Noto Sans Thaana"}
	case language.Thai:
		return []string{"Noto Sans Thai"}
	case language.Tibetan:
		return []string{"Noto Serif Tibetan"}
	case language.Tifinagh:
		return []string{"Noto Sans Tifinagh"}
	case language.Vai:
		return []string{"Noto Sans Vai"}
	case language.Yi:
		return []string{"Noto Sans Yi", "Noto Sans CJK SC"}
	}
	return nil
}
