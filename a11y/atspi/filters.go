package atspi

import "github.com/megascan/gpui-go/a11y/consumer"

func filter(node *consumer.Node) consumer.FilterResult {
	return consumer.CommonFilter(node)
}
