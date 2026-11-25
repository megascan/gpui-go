package gpui

import (
	"testing"

	"github.com/megascan/gpui-go/internal/taffy"
)

func TestTaffy_BorderWidthsToTaffyUseStrokeSnapping(t *testing.T) {
	borderWidths := EdgesOf[AbsoluteLength]{
		Top:    Px(0.0).ToAbsoluteLength(),
		Right:  Px(0.4).ToAbsoluteLength(),
		Bottom: Px(0.5).ToAbsoluteLength(),
		Left:   Px(1.6).ToAbsoluteLength(),
	}
	taffyBorder := borderWidthsToTaffy(&borderWidths, Px(16.0), 1.0)

	if taffyBorder.Top != taffy.LengthPercentageLength(0.0) {
		t.Errorf("top = %v", taffyBorder.Top)
	}
	if taffyBorder.Right != taffy.LengthPercentageLength(1.0) {
		t.Errorf("right = %v", taffyBorder.Right)
	}
	if taffyBorder.Bottom != taffy.LengthPercentageLength(1.0) {
		t.Errorf("bottom = %v", taffyBorder.Bottom)
	}
	if taffyBorder.Left != taffy.LengthPercentageLength(2.0) {
		t.Errorf("left = %v", taffyBorder.Left)
	}
}
