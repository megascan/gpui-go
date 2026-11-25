//go:build darwin

package objc

import pobjc "github.com/ebitengine/purego/objc"

// Block is an Objective-C block object.
type Block = pobjc.Block

// NewBlock returns a +1 heap block calling fn, a Go func whose first parameter is the Block itself.
// Release it when the receiver has copied it.
func NewBlock(fn any) Block {
	mustLoad()
	return pobjc.NewBlock(fn)
}

// InvokeBlock calls block with args and returns its single result.
func InvokeBlock[T any](block Block, args ...any) (T, error) {
	return pobjc.InvokeBlock[T](block, args...)
}
