package taffy

import "testing"

func TestStyleBlock_Defaults(t *testing.T) {
	var c BlockContainerStyleDefaults
	if c.TextAlign() != TextAlignAuto || c.AlignContent() != AlignContentNormal || c.JustifyItems() != AlignItemsNormal {
		t.Error("BlockContainerStyleDefaults")
	}
	var i BlockItemStyleDefaults
	if i.IsTable() || i.AlignSelf().IsSome() || i.JustifySelf().IsSome() || i.AlignContent() != AlignContentNormal || i.Float() != FloatNone || i.Clear() != ClearNone {
		t.Error("BlockItemStyleDefaults")
	}
	if TextAlignDefault != TextAlignAuto {
		t.Error("TextAlignDefault")
	}
}
