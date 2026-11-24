//go:build darwin && !ios

package macos

import "github.com/megascan/gpui-go/a11y/consumer"

func filter(node *consumer.Node) consumer.FilterResult {
	return consumer.CommonFilter(node)
}
