package consumer

import (
	"testing"
	"time"

	"github.com/megascan/gpui-go/a11y"
)

func TestSecurity_SEC040_paragraph_backward_terminates(t *testing.T) {
	window := node(a11y.RoleWindow, 1)
	input := node(a11y.RoleMultilineTextInput, 2, 3, 4)
	input.SetBounds(a11y.Rect{X0: 0, Y0: 0, X1: 100, Y1: 40})
	input.SetTextDirection(a11y.TextDirectionLeftToRight)
	first := node(a11y.RoleTextRun)
	first.SetValue("ab\n")
	first.SetCharacterLengths(ones(3))
	first.SetCharacterPositions([]float32{0, 10, 20})
	first.SetCharacterWidths(widths(3, 10))
	empty := node(a11y.RoleTextRun)
	empty.SetValue("")
	empty.SetCharacterLengths([]uint8{})
	empty.SetCharacterPositions([]float32{})
	empty.SetCharacterWidths([]float32{})
	empty.SetNextOnLine(4)
	second := node(a11y.RoleTextRun)
	second.SetValue("hello")
	second.SetCharacterLengths(ones(5))
	second.SetCharacterPositions([]float32{0, 10, 20, 30, 40})
	second.SetCharacterWidths(widths(5, 10))
	second.SetPreviousOnLine(3)
	tree := NewTree(rootUpdate(1,
		ent(0, window), ent(1, input), ent(2, first), ent(3, empty), ent(4, second),
	), true)
	pos := must(byID(tree, 1).TextPositionFromGlobalUsvIndex(5))
	done := make(chan Position, 1)
	go func() { done <- pos.BackwardToParagraphStart() }()
	select {
	case got := <-done:
		t.Logf("returned paragraph start=%v", got.IsParagraphStart())
	case <-time.After(3 * time.Second):
		t.Fatal("BackwardToParagraphStart did not terminate on a line starting with an empty text run")
	}
}
