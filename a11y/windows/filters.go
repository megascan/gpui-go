//go:build windows

package windows

import "github.com/megascan/gpui-go/a11y/consumer"

func filter(node *consumer.Node) consumer.FilterResult {
	return consumer.CommonFilter(node)
}

func filterWithRootException(node *consumer.Node) consumer.FilterResult {
	return consumer.CommonFilterWithRootException(node)
}
