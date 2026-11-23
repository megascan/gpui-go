package taffy

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func expectPanic(t *testing.T, contains string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatal("expected a panic")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, contains) {
			t.Fatalf("panic %q does not contain %q", msg, contains)
		}
	}()
	f()
}

func TestUtilFrontBackVecBuilder_FrontItemsKeepTheirOrderAndBackItemsAreReversed(t *testing.T) {
	builder := frontBackVecBuilderWithCapacity[int](6)
	for value := range 6 {
		if value%2 == 0 {
			builder.pushFront(value)
		} else {
			builder.pushBack(value)
		}
	}
	vec, frontLen := builder.intoVec()
	if !slices.Equal(vec, []int{0, 2, 4, 5, 3, 1}) || frontLen != 3 {
		t.Fatalf("got %v, %d", vec, frontLen)
	}
}

func TestUtilFrontBackVecBuilder_AllItemsPushedToOneEnd(t *testing.T) {
	builder := frontBackVecBuilderWithCapacity[int](3)
	for value := range 3 {
		builder.pushFront(value)
	}
	vec, frontLen := builder.intoVec()
	if !slices.Equal(vec, []int{0, 1, 2}) || frontLen != 3 {
		t.Fatalf("got %v, %d", vec, frontLen)
	}

	builder = frontBackVecBuilderWithCapacity[int](3)
	for value := range 3 {
		builder.pushBack(value)
	}
	vec, frontLen = builder.intoVec()
	if !slices.Equal(vec, []int{2, 1, 0}) || frontLen != 0 {
		t.Fatalf("got %v, %d", vec, frontLen)
	}
}

func TestUtilFrontBackVecBuilder_ZeroCapacity(t *testing.T) {
	builder := frontBackVecBuilderWithCapacity[uint8](0)
	vec, frontLen := builder.intoVec()
	if len(vec) != 0 || frontLen != 0 {
		t.Fatalf("got %v, %d", vec, frontLen)
	}
}

func TestUtilFrontBackVecBuilder_OwnedItemsAreMovedIntoTheVec(t *testing.T) {
	builder := frontBackVecBuilderWithCapacity[[]int](2)
	builder.pushBack([]int{1, 2})
	builder.pushFront([]int{3})
	vec, frontLen := builder.intoVec()
	if len(vec) != 2 || !slices.Equal(vec[0], []int{3}) || !slices.Equal(vec[1], []int{1, 2}) || frontLen != 1 {
		t.Fatalf("got %v, %d", vec, frontLen)
	}
}

func TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheFrontPanicsInIntoVec(t *testing.T) {
	expectPanic(t, "must equal its capacity", func() {
		builder := frontBackVecBuilderWithCapacity[int](1)
		builder.pushBack(1)
		builder.pushFront(2)
		builder.intoVec()
	})
}

func TestUtilFrontBackVecBuilder_PushingTooManyItemsToTheBackPanicsInIntoVec(t *testing.T) {
	expectPanic(t, "must equal its capacity", func() {
		builder := frontBackVecBuilderWithCapacity[int](1)
		builder.pushFront(1)
		builder.pushBack(2)
		builder.intoVec()
	})
}

func TestUtilFrontBackVecBuilder_PushFrontOutOfBoundsPanics(t *testing.T) {
	expectPanic(t, "", func() {
		builder := frontBackVecBuilderWithCapacity[int](0)
		builder.pushFront(1)
	})
}

func TestUtilFrontBackVecBuilder_PushBackOutOfBoundsPanics(t *testing.T) {
	expectPanic(t, "", func() {
		builder := frontBackVecBuilderWithCapacity[int](1)
		builder.pushBack(1)
		builder.pushBack(2)
	})
}

func TestUtilFrontBackVecBuilder_IntoVecBeforeFullPanics(t *testing.T) {
	expectPanic(t, "must equal its capacity", func() {
		builder := frontBackVecBuilderWithCapacity[int](2)
		builder.pushFront(1)
		builder.intoVec()
	})
}
