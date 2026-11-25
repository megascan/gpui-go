package atspi

import (
	"fmt"
	"strconv"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
)

func colorToString(color a11y.Color) string {
	return fmt.Sprintf("%d,%d,%d", color.Red, color.Green, color.Blue)
}

func formatF32(v float32) string {
	return strconv.FormatFloat(float64(v), 'f', -1, 32)
}

func familyName(node consumer.Node) (string, bool) {
	return node.FontFamily()
}

func size(node consumer.Node) (string, bool) {
	v, ok := node.FontSize()
	if !ok {
		return "", false
	}
	return formatF32(v), true
}

func weight(node consumer.Node) (string, bool) {
	v, ok := node.FontWeight()
	if !ok {
		return "", false
	}
	return formatF32(v), true
}

func style(node consumer.Node) (string, bool) {
	if node.IsItalic() {
		return "italic", true
	}
	return "", false
}

func strikethrough(node consumer.Node) (string, bool) {
	if _, ok := node.Strikethrough(); ok {
		return "true", true
	}
	return "", false
}

func underline(node consumer.Node) (string, bool) {
	deco, ok := node.Underline()
	if !ok {
		return "", false
	}
	if deco.Style == a11y.TextDecorationStyleDouble {
		return "double", true
	}
	return "single", true
}

func bgColor(node consumer.Node) (string, bool) {
	c, ok := node.BackgroundColor()
	if !ok {
		return "", false
	}
	return colorToString(c), true
}

func fgColor(node consumer.Node) (string, bool) {
	c, ok := node.ForegroundColor()
	if !ok {
		return "", false
	}
	return colorToString(c), true
}

func language(node consumer.Node) (string, bool) {
	return node.Language()
}

func justification(node consumer.Node) (string, bool) {
	align, ok := node.TextAlign()
	if !ok {
		return "", false
	}
	switch align {
	case a11y.TextAlignLeft:
		return "left", true
	case a11y.TextAlignCenter:
		return "center", true
	case a11y.TextAlignRight:
		return "right", true
	default:
		return "fill", true
	}
}

var attributeGetters = map[string]func(consumer.Node) (string, bool){
	"family-name":   familyName,
	"size":          size,
	"weight":        weight,
	"style":         style,
	"strikethrough": strikethrough,
	"underline":     underline,
	"bg-color":      bgColor,
	"fg-color":      fgColor,
	"language":      language,
	"justification": justification,
}
