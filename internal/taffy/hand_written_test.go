package taffy

import (
	"fmt"
	"testing"
)

func hwAssertEq[T comparable](t *testing.T, left, right T) {
	t.Helper()
	if left != right {
		t.Fatalf("assertion `left == right` failed\n  left: %+v\n right: %+v", left, right)
	}
}

func hwUnwrap[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	return v
}

func hwOK(err error) {
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
}

func hwLayout[C any](tree *TaffyTree[C], node NodeID) Layout {
	return *hwUnwrap(tree.Layout(node))
}

func hwLPA(v float32) LengthPercentageAuto { return LengthPercentageAutoLength(v) }

func hwDim(v float32) Dimension { return DimensionLength(v) }

func hwSizeDim(w, h Dimension) Size[Dimension] { return Size[Dimension]{Width: w, Height: h} }

func hwDefinite(w, h float32) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: AvailableSpaceDefinite(w), Height: AvailableSpaceDefinite(h)}
}

func hwShouldPanic(t *testing.T, expected string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("test did not panic as expected")
		}
		msg := fmt.Sprint(r)
		if err, ok := r.(error); ok {
			msg = err.Error()
		}
		if !containsStr(msg, expected) {
			t.Fatalf("panic did not contain expected string\n      panic message: %q\n expected substring: %q", msg, expected)
		}
	}()
	f()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hwAssertEqMsg[T comparable](t *testing.T, left, right T, msg string) {
	t.Helper()
	if left != right {
		t.Fatalf("assertion `left == right` failed: %s\n  left: %+v\n right: %+v", msg, left, right)
	}
}
