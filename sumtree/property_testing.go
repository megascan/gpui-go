package sumtree

import "math/rand/v2"

// PropertySumTree returns a generator of trees with between minSize and
// maxSize-1 items drawn from values, upstream's proptest `sum_tree` strategy
// without shrinking.
func PropertySumTree[T Item[S, struct{}], S Summary[S, struct{}]](values func(*rand.Rand) T, minSize, maxSize int) func(*rand.Rand) SumTree[T, S, struct{}] {
	return func(rng *rand.Rand) SumTree[T, S, struct{}] {
		size := minSize
		if maxSize > minSize {
			size += rng.IntN(maxSize - minSize)
		}
		items := make([]T, size)
		for i := range items {
			items[i] = values(rng)
		}
		return SumTreeFromIter[T](items, struct{}{})
	}
}
