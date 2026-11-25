//go:build windows

package gotext

import "github.com/go-text/typesetting/language"

var platformCommonFallback = []string{"Segoe UI", "Segoe UI Emoji", "Segoe UI Symbol", "Segoe UI Historic"}

var platformForbiddenFallback = []string{}

func hanUnification(locale string) []string {
	switch locale {
	case "ja":
		return []string{"Yu Gothic"}
	case "ko":
		return []string{"Malgun Gothic"}
	case "zh-HK":
		return []string{"MingLiU_HKSCS"}
	case "zh-TW":
		return []string{"Microsoft JhengHei UI"}
	default:
		return []string{"Microsoft YaHei UI"}
	}
}

func scriptFallback(script language.Script, locale string) []string {
	switch script {
	case language.Adlam:
		return []string{"Ebrima"}
	case language.Bengali:
		return []string{"Nirmala UI"}
	case language.Canadian_Aboriginal:
		return []string{"Gadugi"}
	case language.Chakma:
		return []string{"Nirmala UI"}
	case language.Cherokee:
		return []string{"Gadugi"}
	case language.Devanagari:
		return []string{"Nirmala UI"}
	case language.Ethiopic:
		return []string{"Ebrima"}
	case language.Gujarati:
		return []string{"Nirmala UI"}
	case language.Gurmukhi:
		return []string{"Nirmala UI"}
	case language.Han:
		return hanUnification(locale)
	case language.Hangul:
		return hanUnification("ko")
	case language.Hiragana:
		return hanUnification("ja")
	case language.Javanese:
		return []string{"Javanese Text"}
	case language.Kannada:
		return []string{"Nirmala UI"}
	case language.Katakana:
		return hanUnification("ja")
	case language.Khmer:
		return []string{"Leelawadee UI"}
	case language.Lao:
		return []string{"Leelawadee UI"}
	case language.Malayalam:
		return []string{"Nirmala UI"}
	case language.Mongolian:
		return []string{"Mongolian Baiti"}
	case language.Myanmar:
		return []string{"Myanmar Text"}
	case language.Oriya:
		return []string{"Nirmala UI"}
	case language.Sinhala:
		return []string{"Nirmala UI"}
	case language.Tamil:
		return []string{"Nirmala UI"}
	case language.Telugu:
		return []string{"Nirmala UI"}
	case language.Thaana:
		return []string{"MV Boli"}
	case language.Thai:
		return []string{"Leelawadee UI"}
	case language.Tibetan:
		return []string{"Microsoft Himalaya"}
	case language.Tifinagh:
		return []string{"Ebrima"}
	case language.Vai:
		return []string{"Ebrima"}
	case language.Yi:
		return []string{"Microsoft Yi Baiti"}
	}
	return nil
}
