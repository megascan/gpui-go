//go:build darwin && !ios

package gotext

import "github.com/go-text/typesetting/language"

var platformCommonFallback = []string{".SF NS", "Menlo", "Apple Color Emoji", "Geneva", "Arial Unicode MS"}

var platformForbiddenFallback = []string{".LastResort"}

func hanUnification(locale string) []string {
	switch locale {
	case "ja":
		return []string{"Hiragino Sans"}
	case "ko":
		return []string{"Apple SD Gothic Neo"}
	case "zh-HK":
		return []string{"PingFang HK"}
	case "zh-TW":
		return []string{"PingFang TC"}
	default:
		return []string{"PingFang SC"}
	}
}

func scriptFallback(script language.Script, locale string) []string {
	switch script {
	case language.Adlam:
		return []string{"Noto Sans Adlam"}
	case language.Arabic:
		return []string{"Geeza Pro"}
	case language.Armenian:
		return []string{"Noto Sans Armenian"}
	case language.Bengali:
		return []string{"Bangla Sangam MN"}
	case language.Buhid:
		return []string{"Noto Sans Buhid"}
	case language.Canadian_Aboriginal:
		return []string{"Euphemia UCAS"}
	case language.Chakma:
		return []string{"Noto Sans Chakma"}
	case language.Devanagari:
		return []string{"Devanagari Sangam MN"}
	case language.Ethiopic:
		return []string{"Kefa"}
	case language.Gothic:
		return []string{"Noto Sans Gothic"}
	case language.Grantha:
		return []string{"Grantha Sangam MN"}
	case language.Gujarati:
		return []string{"Gujarati Sangam MN"}
	case language.Gurmukhi:
		return []string{"Gurmukhi Sangam MN"}
	case language.Han:
		return hanUnification(locale)
	case language.Hangul:
		return hanUnification("ko")
	case language.Hanunoo:
		return []string{"Noto Sans Hanunoo"}
	case language.Hebrew:
		return []string{"Arial"}
	case language.Hiragana:
		return hanUnification("ja")
	case language.Javanese:
		return []string{"Noto Sans Javanese"}
	case language.Kannada:
		return []string{"Noto Sans Kannada"}
	case language.Katakana:
		return hanUnification("ja")
	case language.Khmer:
		return []string{"Khmer Sangam MN"}
	case language.Lao:
		return []string{"Lao Sangam MN"}
	case language.Malayalam:
		return []string{"Malayalam Sangam MN"}
	case language.Mongolian:
		return []string{"Noto Sans Mongolian"}
	case language.Myanmar:
		return []string{"Noto Sans Myanmar"}
	case language.Oriya:
		return []string{"Noto Sans Oriya"}
	case language.Sinhala:
		return []string{"Sinhala Sangam MN"}
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
		return []string{"InaiMathi"}
	case language.Telugu:
		return []string{"Telugu Sangam MN"}
	case language.Thaana:
		return []string{"Noto Sans Thaana"}
	case language.Thai:
		return []string{"Ayuthaya"}
	case language.Tibetan:
		return []string{"Kailasa"}
	case language.Tifinagh:
		return []string{"Noto Sans Tifinagh"}
	case language.Vai:
		return []string{"Noto Sans Vai"}
	case language.Yi:
		return []string{"Noto Sans Yi", "PingFang SC"}
	}
	return nil
}
