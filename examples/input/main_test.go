package main

import (
	"strings"
	"testing"

	"github.com/megascan/gpui-go"
)

func input(content string, cx *gpui.Context[TextInput]) *TextInput {
	return newInput(gpui.SharedString(content), "", cx)
}

func assertEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("assertion `left == right` failed\n  left: %v\n right: %v", got, want)
	}
}

func TestInput_NativeSelectionAndLengthUseUTF16(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		window := cx.AddWindow(func(_ *gpui.Window, cx *gpui.Context[TextInput]) *TextInput { return input("e🙂漢", cx) })
		if err := window.Update(cx, func(input *TextInput, window *gpui.Window, cx *gpui.Context[TextInput]) {
			n, ok := input.TextLengthUTF16(window, cx)
			assertEq(t, ok, true)
			assertEq(t, n, 4)
			input.SetSelectedTextRange(gpui.Range{Start: 1, End: 3}, window, cx)
			assertEq(t, input.selectedRange, gpui.Range{Start: 1, End: 5})
			selection, ok := input.SelectedTextRange(false, window, cx)
			if !ok {
				panic("called `Option::unwrap()` on a `None` value")
			}
			assertEq(t, selection.Range, gpui.Range{Start: 1, End: 3})
			input.SetSelectedTextRange(gpui.Range{Start: 3, End: 99}, window, cx)
			assertEq(t, input.selectedRange, gpui.Range{Start: 5, End: 8})
		}); err != nil {
			panic(err)
		}
	})
}

func TestInput_MarkedSelectionIsRelativeToInsertedText(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		window := cx.AddWindow(func(_ *gpui.Window, cx *gpui.Context[TextInput]) *TextInput { return input("A🙂Z", cx) })
		if err := window.Update(cx, func(input *TextInput, window *gpui.Window, cx *gpui.Context[TextInput]) {
			input.ReplaceAndMarkTextInRange(gpui.Some(gpui.Range{Start: 1, End: 3}), "é🙂", gpui.Some(gpui.Range{Start: 1, End: 3}), window, cx)
			assertEq(t, input.content, "Aé🙂Z")
			assertEq(t, input.markedRange, gpui.Some(gpui.Range{Start: 1, End: 7}))
			assertEq(t, input.selectedRange, gpui.Range{Start: 3, End: 7})

			input.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), "漢", gpui.Some(gpui.Range{Start: 1, End: 1}), window, cx)
			assertEq(t, input.content, "A漢Z")
			assertEq(t, input.markedRange, gpui.Some(gpui.Range{Start: 1, End: 4}))
			assertEq(t, input.selectedRange, gpui.Range{Start: 4, End: 4})

			input.ReplaceAndMarkTextInRange(gpui.None[gpui.Range](), "", gpui.Some(gpui.Range{Start: 0, End: 0}), window, cx)
			assertEq(t, input.content, "AZ")
			assertEq(t, input.markedRange, gpui.None[gpui.Range]())
			assertEq(t, input.selectedRange, gpui.Range{Start: 1, End: 1})
		}); err != nil {
			panic(err)
		}
	})
}

func TestInput_LongSingleLineSelectionUsesOneGeometryFragment(t *testing.T) {
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		view, cx := tc.Cx().AddWindowView(func(_ *gpui.Window, cx *gpui.Context[TextInput]) *TextInput { return input("", cx) })
		defer view.Release()
		view.Update(cx, func(input *TextInput, cx *gpui.Context[TextInput]) {
			input.content = gpui.SharedString(strings.Repeat("a", 10_000))
			cx.Notify()
		})
		cx.RunUntilParked()
		if err := view.UpdateIn(cx, func(input *TextInput, window *gpui.Window, cx *gpui.Context[TextInput]) {
			bounds, ok := input.lastBounds.Get()
			if !ok {
				panic("input was painted")
			}
			rectangles := input.SelectionBoundsForRange(gpui.Range{Start: 0, End: 10_000}, bounds, window, cx)
			assertEq(t, len(rectangles), 1)
			if !(rectangles[0].Size.Width > gpui.Px(0.)) {
				t.Fatalf("width = %v, want > 0", rectangles[0].Size.Width)
			}
		}); err != nil {
			panic(err)
		}
	})
}
