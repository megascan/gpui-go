package taffy

import "fmt"

type frontBackVecBuilder[T any] struct {
	vec      []T
	capacity int
	frontLen int
	backLen  int
}

func frontBackVecBuilderWithCapacity[T any](capacity int) frontBackVecBuilder[T] {
	return frontBackVecBuilder[T]{vec: make([]T, capacity), capacity: capacity, frontLen: 0, backLen: 0}
}

func (b *frontBackVecBuilder[T]) pushFront(value T) {
	b.vec[b.frontLen] = value
	b.frontLen++
}

func (b *frontBackVecBuilder[T]) pushBack(value T) {
	b.vec[b.capacity-b.backLen-1] = value
	b.backLen++
}

func (b *frontBackVecBuilder[T]) intoVec() ([]T, int) {
	if b.frontLen+b.backLen != b.capacity {
		panic(fmt.Sprintf("assertion `left == right` failed: the number of items pushed to a FrontBackVecBuilder must equal its capacity\n  left: %d\n right: %d", b.frontLen+b.backLen, b.capacity))
	}
	vec := b.vec
	b.vec = nil
	return vec, b.frontLen
}
